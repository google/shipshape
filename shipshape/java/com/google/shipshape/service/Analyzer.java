package com.google.shipshape.service;

import com.google.common.collect.ImmutableList;
import com.google.shipshape.proto.NotesProto.Note;
import com.google.shipshape.proto.ShipshapeContextProto.ShipshapeContext;

/**
 * Simplified API for analyzers written in Java.
 * @param <T> State provided to the analyzer (optional).
 */
public interface Analyzer<T> {
  /**
   * Returns zero or more {@link Note} referenced by {@code shipshapeContext}.
   */
  ImmutableList<Note> analyze(ShipshapeContext shipshapeContext, T state)
      throws AnalyzerException;

  /**
   * Returns the category used in every note from this analyzer.
   */
  String getCategory();
}
