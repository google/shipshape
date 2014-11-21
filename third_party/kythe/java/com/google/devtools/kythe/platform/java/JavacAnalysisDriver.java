package com.google.devtools.kythe.platform.java;


import com.google.common.collect.ImmutableList;
import com.google.devtools.kythe.common.FormattingLogger;
import com.google.devtools.kythe.platform.shared.AnalysisException;
import com.google.devtools.kythe.platform.shared.FileDataProvider;
import com.google.devtools.kythe.proto.Analysis.CompilationUnit;

import com.sun.tools.javac.api.JavacTool;

import java.util.List;

import javax.annotation.processing.Processor;
import javax.tools.JavaCompiler;

/**
 * Base implementation for the various analysis drivers (bigtable, streaming, blaze etc..).
 * Allows running {@link JavacAnalyzer} over compilations that are retrieved from various locations.
 */
public class JavacAnalysisDriver {
  private static final FormattingLogger logger =
      FormattingLogger.getLogger(JavacAnalysisDriver.class);
  private final List<Processor> processors;

  public JavacAnalysisDriver() {
    this(ImmutableList.<Processor>of());
  }

  public JavacAnalysisDriver(List<Processor> processors) {
    this.processors = processors;
  }

  /**
   * Provides a {@link JavaCompiler} that this code is compiled with instead of the default behavior
   * of loading the {@link JavaCompiler} available at runtime. This is needed to run the Java 6
   * compiler on the Java 7 runtime.
   */
  public static JavaCompiler getCompiler() {
    return JavacTool.create();
  }

  public void analyze(JavacAnalyzer analyzer, CompilationUnit compilationUnit,
      FileDataProvider fileDataProvider, boolean isLocalAnalysis) throws AnalysisException {
    JavaCompilationDetails details =
        JavaCompilationDetails.createDetails(compilationUnit, fileDataProvider, isLocalAnalysis, processors);
    if (details == null) {
      throw new AnalysisException(
          "Could not build javac compilation details; perhaps missing javac arguments?");
    }

    analyzer.analyzeCompilationUnit(details);
  }
}
