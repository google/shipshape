// Copyright 2011 Google Inc. All Rights Reserved.

package com.google.devtools.kythe.platform.shared;

/**
 * Exception to signal catastrophic failure in sharded analysis.
 *
 * @author jvg@google.com (Jeffrey van Gogh)
 */
public class AnalysisException extends Exception {
  public AnalysisException(String message) {
    super(message);
  }

  public AnalysisException(String message, Throwable innerException) {
    super(message, innerException);
  }

  public AnalysisException(Throwable innerException) {
    super(innerException);
  }
}
