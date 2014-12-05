/*
 * Copyright 2014 Google Inc. All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package com.google.shipshape.analyzers;

import com.google.common.base.Preconditions;
import com.google.common.collect.ImmutableList;
import com.google.devtools.kythe.platform.java.JavaCompilationDetails;
import com.google.devtools.source.v1.SourceContextProto.SourceContext;
import com.google.errorprone.DescriptionListener;
import com.google.errorprone.ErrorProneScanner;
import com.google.errorprone.Scanner;
import com.google.errorprone.VisitorState;
import com.google.errorprone.fixes.Fix;
import com.google.errorprone.matchers.Description;
import com.google.shipshape.proto.NotesProto;
import com.google.shipshape.proto.NotesProto.Location;
import com.google.shipshape.proto.NotesProto.Note;
import com.google.shipshape.proto.TextRangeProto.TextRange;
import com.google.shipshape.proto.ShipshapeContextProto.Stage;
import com.google.shipshape.proto.ShipshapeContextProto.ShipshapeContext;
import com.google.shipshape.service.AnalyzerException;
import com.google.shipshape.service.Analyzer;
import com.google.shipshape.service.ShipshapeLogger;

import com.sun.source.tree.CompilationUnitTree;
import com.sun.source.util.JavacTask;
import com.sun.tools.javac.api.JavacTaskImpl;
import com.sun.tools.javac.tree.JCTree;
import com.sun.tools.javac.tree.JCTree.JCCompilationUnit;
import com.sun.tools.javac.util.Context;
import com.sun.tools.javac.util.Position;

import java.io.IOException;
import java.io.UnsupportedEncodingException;
import java.nio.charset.Charset;
import java.util.List;

import javax.annotation.Nullable;

/**
 * Uses the ErrorProne Scanner to create notes for the given CompilationUnit.
 */
public class ErrorProneAnalyzer extends JavacAnalyzer {
  private static final ShipshapeLogger logger = ShipshapeLogger.getLogger(ErrorProneAnalyzer.class);

  public static final String CATEGORY = "ErrorProne";

  @Override
  public String getCategory() {
    return CATEGORY;
  }

  @Override
  public ImmutableList<Note> analyzeFile(final ShipshapeContext shipshapeContext,
      final JavaCompilationDetails compilationDetails,
      final CompilationUnitTree file, String path) throws AnalyzerException {

    final ImmutableList.Builder<Note> notes = new ImmutableList.Builder<>();

    // Don't run error-prone if the compilation failed in any way.
    if (compilationDetails.hasCompileErrors() || compilationDetails.inBadCompilationState()) {
      return ImmutableList.of();
    }

    // TODO(ciera): Create an EnabledPredicate that uses a configuration file
    // and push that change upstream to errorprone.
    Scanner scanner = new ErrorProneScanner(ErrorProneScanner.EnabledPredicate.DEFAULT_CHECKS);

    JavacTask javacTask = compilationDetails.getJavac();
    Context context = ((JavacTaskImpl) javacTask).getContext();
    JCCompilationUnit compilationUnit = (JCCompilationUnit) file;

    CharSequence source = null;
    try {
      source = file.getSourceFile().getCharContent(false);
    } catch (IOException e) {
      logger.severe(e, "Unable to read source file " + path, shipshapeContext, CATEGORY);
    }
    NoteAdapter adapter = new NoteAdapter(notes, path, compilationUnit,
        shipshapeContext, source, compilationDetails.getEncoding());
    scanner.scan(file, new VisitorState(context, adapter));
    return notes.build();
  }

  /** A class to conver ErrorProne descriptions to Shipshape Notes.*/
  private static class NoteAdapter implements DescriptionListener {
    private final ImmutableList.Builder<Note> notes;
    private final String path;
    private final JCCompilationUnit compilationUnit;
    private final ShipshapeContext shipshapeContext;
    @Nullable private final CharSequence source;
    private final String encoding;

    public NoteAdapter(ImmutableList.Builder<Note> notes,
        String path,
        JCCompilationUnit compilationUnit,
        ShipshapeContext shipshapeContext,
        @Nullable CharSequence source,
        String encoding) {
      this.notes = notes;
      this.path = path;
      this.compilationUnit = compilationUnit;
      this.shipshapeContext = shipshapeContext;
      this.source = source;
      this.encoding = Preconditions.checkNotNull(encoding);
    }

    @Override
    public void onDescribed(Description description) {
      SourceContext sourceContext = shipshapeContext.getSourceContext();

      // Create a TextRange to mark where the problem occurred.
      JCTree treeNode = (JCTree) description.node;
      Position.LineMap lineMap = compilationUnit.getLineMap();
      TextRange textRange = TextRange.newBuilder()
          .setStartLine(lineMap.getLineNumber(treeNode.getStartPosition()))
          .setStartColumn(lineMap.getColumnNumber(treeNode.getStartPosition()))
          .setEndLine(lineMap.getLineNumber(treeNode.getEndPosition(compilationUnit.endPositions)))
          .setEndColumn(
              lineMap.getColumnNumber(treeNode.getEndPosition(compilationUnit.endPositions)))
          .build();

      // Create a Note for this problem.
      Note.Builder noteBuilder = Note.newBuilder()
          .setDescription(description.getMessageWithoutCheckName())
          .setCategory(CATEGORY)
          .setSubcategory(description.checkName)
          .setLocation(Location.newBuilder()
              .setPath(path)
              .setRange(textRange)
              .setSourceContext(sourceContext));

      // If the source is available, create fix protos for the problem.
      if (source != null) {
        try {
          EncodingOffsetConverter encodingOffsetConverter =
              new EncodingOffsetConverter(source, Charset.forName(encoding));
          FixAndReplacementConverter translator = new FixAndReplacementConverter(
              path, sourceContext, encodingOffsetConverter, compilationUnit);
          for (int i = 0; i < description.fixes.size(); i++) {
            Fix errorProneFix = description.fixes.get(i);
            NotesProto.Fix fix = translator.fromErrorProneFix(errorProneFix,
                String.format("Fix #%d for error-prone %s warning", i + 1, description.checkName));
            noteBuilder.addFix(fix);
          }
        } catch (UnsupportedEncodingException e) {
          logger.info("Cannot convert to byte index", shipshapeContext, CATEGORY);
        }
      }
      notes.add(noteBuilder.build());
    }
  }
}
