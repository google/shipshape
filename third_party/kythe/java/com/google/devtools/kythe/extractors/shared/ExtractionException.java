// Copyright 2011 Google Inc. All Rights Reserved.

package com.google.devtools.kythe.extractors.shared;

/**
 * Exception describing issues extracting compilation unit information.
 * Thrown for issues such as: Bigtable connectivity, IO issues, compiler
 * failures etc...
 *
 * @author jvg@google.com (Jeffrey van Gogh)
 */
public class ExtractionException extends Exception {

  private final boolean shouldRetry;
  public ExtractionException(String message, boolean shouldRetry) {
    super(message);
    this.shouldRetry = shouldRetry;
  }

  public ExtractionException(Throwable innerException, boolean shouldRetry) {
    super(innerException);
    this.shouldRetry = shouldRetry;
  }

  public ExtractionException(String message, Throwable innerException, boolean shouldRetry) {
    super(message, innerException);
    this.shouldRetry = shouldRetry;
  }

  public boolean shouldRetry() {
    return shouldRetry;
  }
}
