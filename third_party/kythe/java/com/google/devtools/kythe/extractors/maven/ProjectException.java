package com.google.devtools.kythe.extractors.maven;

import java.lang.Throwable;

/**
 * Exception class thrown from the {@link Project} class.
 **/
public class ProjectException extends Throwable {
  public ProjectException(String message, Throwable cause) { super(message, cause); }
}