package com.google.devtools.kythe.analyzers.base;

import com.google.devtools.kythe.proto.Storage.VName;

/** Emitter of facts. */
public interface FactEmitter {
  /**
   * Emits a single fact to some data sink. {@link edgeKind} and {@link target} must both be either
   * {@code null} (for a node entry) or non-{@code null} (for an edge entry).
   */
  public void emit(VName source, String edgeKind, VName target, String factName, byte[] factValue);
}
