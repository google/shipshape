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
