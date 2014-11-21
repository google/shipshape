package com.google.devtools.kythe.util;

/** Structure representing some arbitrary offset span. */
public class Span {
  private final int start, end;

  public Span(int startOffset, int endOffset) {
    this.start = startOffset;
    this.end = endOffset;
  }

  public int getStart() {
    return start;
  }

  public int getEnd() {
    return end;
  }

  @Override
  public String toString() {
    return String.format("Span{%d, %d}", start, end);
  }
}
