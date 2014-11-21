package com.google.shipshape.service;

import com.google.common.collect.ImmutableList;
import com.google.shipshape.proto.NotesProto.Note;
import com.google.shipshape.proto.ShipshapeContextProto.ShipshapeContext;

/**
 * API for analyzers written in Java that do not carry java specific state.
 */
public abstract class StatelessAnalyzer implements Analyzer<Void> {

  /**
   * Returns zero or more {@link Note}s for the referenced {@code shipshapeContext}.
   */
  public abstract ImmutableList<Note> analyze(ShipshapeContext shipshapeContext)
      throws AnalyzerException;

  @Override
  public ImmutableList<Note> analyze(ShipshapeContext shipshapeContext, Void state)
      throws AnalyzerException {
    return analyze(shipshapeContext);
  }
}
