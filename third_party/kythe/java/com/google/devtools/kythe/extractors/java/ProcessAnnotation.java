// Copyright 2011 Google Inc. All Rights Reserved.

package com.google.devtools.kythe.extractors.java;

import com.google.devtools.kythe.common.FormattingLogger;

import com.sun.tools.javac.code.Symbol.TypeSymbol;

import java.io.IOException;
import java.util.Set;

import javax.annotation.processing.AbstractProcessor;
import javax.annotation.processing.RoundEnvironment;
import javax.annotation.processing.SupportedAnnotationTypes;
import javax.lang.model.SourceVersion;
import javax.lang.model.element.TypeElement;
import javax.tools.JavaFileObject.Kind;
import javax.tools.StandardLocation;

/**
 * This class is used to visit all the annotation used in Java files and record the usage of these
 * annotation as JavaFileObject. This processing will eliminate some of the platform errors caused
 * by javac not being able to find the class for some of the annotations in the analysis phase. The
 * recorded files will later be put in the bigtable for analysis phase.
 *
 * @author amshali@google.com (Amin Shali)
 */
@SupportedAnnotationTypes(value = {"*"})
public class ProcessAnnotation extends AbstractProcessor {

  private static final FormattingLogger logger =
      FormattingLogger.getLogger(ProcessAnnotation.class);

  UsageAsInputReportingFileManager fileManager;
  public ProcessAnnotation(UsageAsInputReportingFileManager fileManager) {
    this.fileManager = fileManager;
  }

  @Override
  public boolean process(Set<? extends TypeElement> annotations, RoundEnvironment roundEnv) {
    for (TypeElement e : annotations) {
      TypeSymbol s = (TypeSymbol) e;
      try {
        UsageAsInputReportingJavaFileObject jfo = (UsageAsInputReportingJavaFileObject)
          fileManager.getJavaFileForInput(StandardLocation.CLASS_OUTPUT, s.flatName().toString(),
            Kind.CLASS);
        if (jfo == null) {
          jfo = (UsageAsInputReportingJavaFileObject) fileManager.getJavaFileForInput(
            StandardLocation.CLASS_PATH, s.flatName().toString(), Kind.CLASS);
        }
        if (jfo == null) {
          jfo = (UsageAsInputReportingJavaFileObject) fileManager.getJavaFileForInput(
            StandardLocation.SOURCE_PATH, s.flatName().toString(), Kind.CLASS);
        }
        if (jfo != null) {
          jfo.markUsed();
        }
      } catch (IOException ex) {
        // We only log any IO exception here and do not cancel the whole processing because of an
        // exception in this stage.
        logger.severefmt("Error in annotation processing: %s", ex.getMessage());
      }
    }
    // We must return false so normal processors run after us.
    return false;
  }

  @Override
  public SourceVersion getSupportedSourceVersion() {
    return SourceVersion.latest();
  }
}
