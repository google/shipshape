package com.google.shipshape.analyzers.testdata;

public class Test {
  // warning: 'private' modifier out of order with the JLS suggestions.
  final private static String s = "whatever";

  // warning: Member name 'foo_bar' must match pattern '^[a-z][a-z0-9][a-zA-Z0-9]*$'.
  int foo_bar;
}
