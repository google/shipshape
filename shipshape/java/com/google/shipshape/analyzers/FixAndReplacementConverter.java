package com.google.shipshape.analyzers;

import com.google.devtools.source.v1.SourceContextProto.SourceContext;
import com.google.shipshape.proto.TextRangeProto.TextRange;
import com.google.shipshape.proto.NotesProto;
import com.google.shipshape.proto.NotesProto.Location;
import com.google.shipshape.proto.NotesProto.Note;
import com.google.errorprone.fixes.Fix;
import com.google.errorprone.fixes.Replacement;

import com.sun.tools.javac.tree.JCTree.JCCompilationUnit;

import java.util.ArrayList;
import java.util.List;
import java.util.Set;

/**
 * Converts error-prone {@link Fix}es and {@link Replacement}s into
 * {@link com.google.shipshape.proto.NotesProto.Fix}es and
 * {@link com.google.shipshape.proto.NotesProto.Replacement}s.
 */
public class FixAndReplacementConverter {

  private final String path;
  private final SourceContext sourceContext;
  private final EncodingOffsetConverter encodingOffsetConverter;
  private final JCCompilationUnit compilationUnit;

  /**
   * @param path The path to the file in which to make the replacement
   * @param sourceContext The source context in which to interpret the path and positions
   */
  public FixAndReplacementConverter(String path, SourceContext sourceContext,
      EncodingOffsetConverter encodingOffsetConverter, JCCompilationUnit compilationUnit) {
    this.path = path;
    this.sourceContext = sourceContext;
    this.encodingOffsetConverter = encodingOffsetConverter;
    this.compilationUnit = compilationUnit;
  }

  /**
   * Translate an error-prone {@link Fix} into a {@link com.google.shipshape.proto.NotesProto.Fix}
   * proto.
   *
   * @param fix The error-prone fix
   * @param desc The description to include in the Fix proto
   */
  public NotesProto.Fix fromErrorProneFix(Fix fix, String desc) {
    NotesProto.Fix.Builder fixBuilder = NotesProto.Fix.newBuilder().setDescription(desc);
    Set<Replacement> replacements = fix.getReplacements(compilationUnit.endPositions);

    for (Replacement replacement : replacements) {
      fixBuilder.addReplacement(getReplacement(
          replacement.startPosition(), replacement.endPosition(), replacement.replaceWith()));
    }

    // TODO(ciera): Use Refactory to handle imports properly and to format the code nicely.

    return fixBuilder.build();
  }

  /**
   * @param startPosUTF16 The index of the first UTF-16 code unit in the range (inclusive) to
   *        replace
   * @param endPosUTF16 The index of the last UTF-16 code unit in the range (inclusive) to replace
   * @param newContent The text to substitute into the range
   */
  public NotesProto.Replacement getReplacement(int startPosUTF16, int endPosUTF16,
      String newContent) {
    int byteStartPos = encodingOffsetConverter.toByteIndex(startPosUTF16);
    int byteEndPos = encodingOffsetConverter.toByteIndex(endPosUTF16);
    NotesProto.Replacement.Builder builder = NotesProto.Replacement.newBuilder()
        .setPath(path)
        .setRange(NotesProto.FixRange.newBuilder()
            .setStart(NotesProto.FixRange.Position.newBuilder().setByte(byteStartPos))
            .setEnd(NotesProto.FixRange.Position.newBuilder().setByte(byteEndPos)))
        .setNewContent(newContent);
    return builder.build();
  }

  /**
   * Returns a list of all the replacement text from this fix, excluding imports.
   */
  public List<String> getReplacementText(Fix fix) {
    List<String> replStrings = new ArrayList<>();
    Set<Replacement> replacements = fix.getReplacements(compilationUnit.endPositions);
    for (Replacement replacement : replacements) {
      replStrings.add(replacement.replaceWith());
    }
    return replStrings;
  }
}
