package com.google.devtools.kythe.platform.java;

import com.google.common.base.Strings;
import com.google.devtools.kythe.platform.shared.AnalysisException;
import com.google.devtools.kythe.platform.shared.NullStatisticsCollector;
import com.google.devtools.kythe.platform.shared.StatisticsCollector;
import com.google.devtools.kythe.proto.Analysis.CompilationUnit;

import com.sun.source.tree.CompilationUnitTree;

import java.io.Serializable;
import java.net.URI;

/**
 * Recreates a javac compiler instance from a CompilationUnit and allows
 * derived classes to analyze the instance and report results.
 */
public abstract class JavacAnalyzer implements Serializable {

  private final StatisticsCollector collector;

  public JavacAnalyzer() {
    this(NullStatisticsCollector.getInstance());
  }

  public JavacAnalyzer(StatisticsCollector collector) {
    this.collector = collector;
  }

  public StatisticsCollector getStatisticsCollector() {
    return collector;
  }

  /**
   * Overridden in derived classes to perform analysis on a java CompilationUnit.
   *
   * @param compilationDetails contains all information needed to perform java analysis.
   * @throws AnalysisException if analysis has a catastrophic failure.
   */
  public void analyzeCompilationUnit(JavaCompilationDetails compilationDetails)
      throws AnalysisException {
    for (CompilationUnitTree file : compilationDetails.getAsts()) {
      getStatisticsCollector().incrementCounter("kythe-analyzer-file-started");
      URI uri = file.getSourceFile().toUri();
      String fullPath = file.getSourceFile().toUri().getRawPath();
      if (!uri.getScheme().equals("file")) {
        fullPath = fullPath.substring(1);
      }
      String compilationUnitPath = fullPath;

      for (String sourceFile : compilationDetails.getCompilationUnit().getSourceFileList()) {
        if (fullPath.endsWith(sourceFile)) {
          compilationUnitPath = sourceFile;
          break;
        }
      }

      if (Strings.isNullOrEmpty(compilationUnitPath)) {
        continue;
      }

      analyzeFile(compilationDetails, file);
      getStatisticsCollector().incrementCounter("kythe-analyzer-file-finished");
    }
  }

  /**
   * Overridden in derived classes to perform analysis on a specific file of the CompilationUnit.
   *
   * @param compilationDetails contains all information needed to perform java analysis.
   * @throws AnalysisException if analysis has a catastrophic failure.
   */
  public void analyzeFile(JavaCompilationDetails compilationDetails, CompilationUnitTree file)
      throws AnalysisException {
  }
}
