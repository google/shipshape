// Copyright 2011 Google Inc. All Rights Reserved.

package com.google.devtools.kythe.platform.shared;

import java.io.Serializable;

/**
 * A statistics collector that ignores all statistics.
 *
 * @author jvg@google.com (Jeffrey van Gogh)
 */
public class NullStatisticsCollector implements StatisticsCollector,
    Serializable {

  private static NullStatisticsCollector instance = new NullStatisticsCollector();

  /**
   * Returns the single instance of the statistics collector that
   * ignores all statistics.
   */
  public static NullStatisticsCollector getInstance() {
    return instance;
  }

  private NullStatisticsCollector() {
  }

  @Override
  public void incrementCounter(String name) {
  }

  @Override
  public void incrementCounter(String name, int amount) {
  }
}
