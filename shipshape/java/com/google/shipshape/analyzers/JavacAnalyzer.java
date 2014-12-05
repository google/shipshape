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

import com.google.devtools.kythe.platform.java.JavaCompilationDetails;
import com.google.errorprone.Scanner;
import com.google.errorprone.VisitorState;
import com.google.common.collect.ImmutableList;
import com.google.common.collect.Iterables;
import com.google.shipshape.proto.NotesProto.Location;
import com.google.shipshape.proto.NotesProto.Note;
import com.google.shipshape.proto.ShipshapeContextProto.Stage;
import com.google.shipshape.proto.ShipshapeContextProto.ShipshapeContext;
import com.google.shipshape.service.ShipshapeLogger;
import com.google.shipshape.service.AnalyzerException;
import com.google.shipshape.service.Analyzer;

import com.sun.source.tree.CompilationUnitTree;

import java.net.URI;

import javax.annotation.Nullable;

/**
 * An abstract JavacAnalyzer. This implements the analyzer method and provides an
 * abstract method for analyzing a single file at a time.
 */
public abstract class JavacAnalyzer implements Analyzer<JavaCompilationDetails> {
  private static final ShipshapeLogger logger = ShipshapeLogger.getLogger(JavacAnalyzer.class);

  @Override
  public ImmutableList<Note> analyze(
      ShipshapeContext shipshapeContext, JavaCompilationDetails details)
      throws AnalyzerException {
    ImmutableList.Builder<Note> notes = new ImmutableList.Builder<>();

    if (details.getAnalysisCrash() != null) {
      throw new AnalyzerException(
          getCategory(), shipshapeContext, "Exception from javac", details.getAnalysisCrash());
    }

    for (CompilationUnitTree file : details.getAsts()) {
      URI uri = file.getSourceFile().toUri();
      String path = getPathRelativeToRoot(shipshapeContext, uri);
      logger.info("Investigating file " + path, shipshapeContext, getCategory());
      if (path == null) {
        String msg = String.format(
            "The provided path %s was invalid for root %s", path, shipshapeContext.getRepoRoot());
        throw new AnalyzerException(getCategory(), shipshapeContext, msg);
      }
      if (isRelevantJavaFile(shipshapeContext, path)) {
        logger.info("Analyzing file " + path, shipshapeContext, getCategory());
        notes.addAll(analyzeFile(shipshapeContext, details, file, path));
      }
    }
    return notes.build();
  }

  /**
    * Returns true if this is a java file, it is in the file list for the context,
    * and it is in the current compilation unit.
    */
  public boolean isRelevantJavaFile(ShipshapeContext context, String path) {
    for (String sourceFile : context.getCompilationDetails().getCompilationUnit().getSourceFileList()) {
      if (path.endsWith(sourceFile)) {
        return path.endsWith(".java") && Iterables.contains(context.getFilePathList(), path);
      }
    }
    return false;
  }

  /**
   * Convers a URI from Javac into a path relative to the repo root in the shipshape
   * context.
   */
  public String getPathRelativeToRoot(ShipshapeContext context, URI javacUri) {
    return javacUri.getRawPath().substring(1);
  }

  public abstract ImmutableList<Note> analyzeFile(final ShipshapeContext context,
      final JavaCompilationDetails details,
      final CompilationUnitTree file, String path) throws AnalyzerException;
}
