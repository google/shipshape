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

package com.google.shipshape.service;

import com.google.devtools.kythe.extractors.shared.CompilationDescription;
import com.google.devtools.kythe.extractors.shared.IndexInfoUtils;
import com.google.devtools.kythe.platform.java.JavaCompilationDetails;
import com.google.devtools.kythe.platform.shared.FileDataCache;
import com.google.devtools.kythe.proto.Analysis.CompilationUnit;
import com.google.shipshape.proto.ShipshapeContextProto.ShipshapeContext;

import java.io.IOException;

/**
 * Creates a JavaCompilationDetails from a Java-based CompilationUnit.
 */
public class JavacStateBuilder implements AnalyzerStateBuilder<JavaCompilationDetails> {
  private boolean storeOutput;

  public JavacStateBuilder(boolean storeJavacOutput) {
    this.storeOutput = storeJavacOutput;
  }

  public JavaCompilationDetails build(ShipshapeContext context) throws AnalyzerException {
    if (!context.getCompilationDetails().hasCompilationDescriptionPath()) {
      return null;
    }

    // Read the files from the provided kindex file, not from the source tree.
    // This way, we have the source for generated files as well.
    String compilationPath = context.getCompilationDetails().getCompilationDescriptionPath();
    CompilationDescription desc;
    try {
      desc = IndexInfoUtils.readIndexInfoFromFile(compilationPath);
    } catch (IOException err) {
      throw new AnalyzerException(
          context, "Could not read compilation description from " + compilationPath, err);
    }

    FileDataCache cachedFiles = new FileDataCache(desc.getFileContents());
    CompilationUnit compilationUnit = context.getCompilationDetails().getCompilationUnit();
    return JavaCompilationDetails.createDetails(compilationUnit, cachedFiles, storeOutput);
  }
}
