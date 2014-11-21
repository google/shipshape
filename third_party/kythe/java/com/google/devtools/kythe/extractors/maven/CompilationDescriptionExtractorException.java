package com.google.devtools.kythe.extractors.maven;

/**
 * This exception class to be thrown from the {@link CompilationDescriptionExtractor} class.
 **/
public class CompilationDescriptionExtractorException extends Throwable {
  public CompilationDescriptionExtractorException(String message) { super(message); }

  public CompilationDescriptionExtractorException(String message, Throwable cause) {
    super(message, cause);
  }
}