package com.google.devtools.kythe.extractors.shared;

import com.google.devtools.kythe.proto.Analysis.CompilationUnit.FileInput;

import java.util.Comparator;

/**
 * A {@code Comparator} for {@code FileInput}.
 */
public class CompilationFileInputComparator implements Comparator<FileInput> {
  private static final CompilationFileInputComparator COMPARATOR =
      new CompilationFileInputComparator();

  @Override
  public int compare(FileInput left, FileInput right) {
    return left.getInfo().getPath().compareTo(right.getInfo().getPath());
  }

  private CompilationFileInputComparator() {}

  public static CompilationFileInputComparator getComparator() {
    return COMPARATOR;
  }
}
