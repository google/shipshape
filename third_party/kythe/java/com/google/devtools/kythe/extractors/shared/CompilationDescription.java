package com.google.devtools.kythe.extractors.shared;

import com.google.devtools.kythe.proto.Analysis.CompilationUnit;
import com.google.devtools.kythe.proto.Analysis.FileData;

/**
 * Contains all data to completely describes a compilation.
 * Includes compilation metadata and all required input files.
 */
public class CompilationDescription {
  private final CompilationUnit compilationUnit;
  private final Iterable<FileData> fileContents;

  public CompilationDescription(CompilationUnit compilationUnit, Iterable<FileData> fileContents) {
    this.compilationUnit = compilationUnit;
    this.fileContents = fileContents;
  }

  public CompilationUnit getCompilationUnit() {
    return compilationUnit;
  }

  public Iterable<FileData> getFileContents() {
    return this.fileContents;
  }
}
