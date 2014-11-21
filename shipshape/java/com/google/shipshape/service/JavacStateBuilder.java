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
