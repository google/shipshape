package com.google.devtools.kythe.analyzers.base;

/** Schema-defined Kythe edge kinds. */
public enum EdgeKind {
  DEFINES(true, "defines"),
  REF(true, "ref"),

  CHILDOF("childof"),
  EXTENDS("extends"),
  IMPLEMENTS("implements"),
  IS("is"),
  NAMED("named"),
  PARAM("param");

  private  static final String EDGE_PREFIX = "/kythe/edge/";

  private final boolean isAnchorEdge;
  private final String kind;
  EdgeKind(boolean isAnchorEdge, String kind) {
    this.isAnchorEdge = isAnchorEdge;
    this.kind = EDGE_PREFIX + kind;
  }

  EdgeKind(String kind) {
    this(false, kind);
  }

  /** Returns {@code true} if the edge is used for {@link NodeKind.ANCHOR}s. */
  public final boolean isAnchorEdge() {
    return isAnchorEdge;
  }

  /** Returns the edge kind's Kythe GraphStore value. */
  public final String getValue() {
    return kind;
  }

  @Override
  public String toString() {
    return kind;
  }
}
