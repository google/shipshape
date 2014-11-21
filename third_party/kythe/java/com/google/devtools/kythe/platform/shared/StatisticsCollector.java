// Copyright 2011 Google Inc. All Rights Reserved.

package com.google.devtools.kythe.platform.shared;

/**
 * Allows different analysis drivers to plug-in different statistics collectors
 * to the {@link com.google.devtools.kythe.platform.java.JavacAnalyzer JavacAnalyzer}.
 *
 * @author jvg@google.com (Jeffrey van Gogh)
 */
public interface StatisticsCollector {
  /**
   * Increments the named counter by one.
   */
  void incrementCounter(String name);

  /**
   * Increments the named counter by specified amount.
   */
  void incrementCounter(String name, int amount);
}
