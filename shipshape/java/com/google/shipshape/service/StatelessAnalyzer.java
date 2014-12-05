/*
 * Copyright 2014 Google Inc. All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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
