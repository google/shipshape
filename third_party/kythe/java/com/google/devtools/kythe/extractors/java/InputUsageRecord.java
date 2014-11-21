package com.google.devtools.kythe.extractors.java;

import javax.tools.JavaFileObject;

/**
 * A compilation with multiple rounds of annotation processing will create new file objects
 * for each round. This class records which input paths were used at any point in the compilation.
 * One instance is created for each unique input path.
 */
public class InputUsageRecord {

  private final JavaFileObject fileObject;
  private boolean isUsed = false;

  public InputUsageRecord(JavaFileObject fileObject) {
    if (fileObject == null) {
      throw new IllegalStateException();
    }
    this.fileObject = fileObject;
  }

  /**
   * Record that the compiler used this file as input.
   */
  public void markUsed() {
    isUsed = true;
  }

  /**
   * @return true if the compiler used this file as input.
   */
  public boolean isUsed() {
    return isUsed;
  }

  /**
   * @return the first file object created for this file.
   */
  public JavaFileObject fileObject() {
    return fileObject;
  }
}
