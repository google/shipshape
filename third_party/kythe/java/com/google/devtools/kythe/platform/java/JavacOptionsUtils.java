package com.google.devtools.kythe.platform.java;

import com.google.common.base.Joiner;

import java.util.ArrayList;
import java.util.List;

import javax.annotation.Nullable;

/**
 * A utility class for dealing with javac command-line options.
 */
public class JavacOptionsUtils {

  /**
   * Extract the encoding flag from the list of javac options. If the flag is specified more than
   * once, returns the last copy, which matches javac's behavior.  If the flag is not specified,
   * returns null.
   */
  public static @Nullable String getEncodingOption(List<String> options) {
    int i = options.lastIndexOf("-encoding");
    return (i >= 0) ? options.get(i + 1) : null;
  }

  /** If there is no encoding set, make sure to set the default encoding.*/
  public static List<String> ensureEncodingSet(List<String> options, String defaultEncoding) {
    if (getEncodingOption(options) == null) {
      options.add("-encoding");
      options.add(defaultEncoding);
    }
    return options;
  }

  /** Remove the existing warning options, and do all instead.*/
  public static List<String> useAllWarnings(List<String> options) {
    List<String> result = new ArrayList<>();
    for (String option : options) {
      if (!option.startsWith("-Xlint")) {
        result.add(option);
      }
    }
    result.add("-Xlint:all");
    return result;
  }

  /** Append the classpath to the list of options.*/
  public static List<String> appendClasspathToOptions(List<String> options, List<String> paths) {
    StringBuilder classPath = new StringBuilder();
    Joiner.on(':').appendTo(classPath, paths);

    if (classPath.length() > 0) {
      options.add("-cp");
      options.add(classPath.toString());
    }
    return options;
  }
}
