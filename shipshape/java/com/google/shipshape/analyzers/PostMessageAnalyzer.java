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

package com.google.shipshape.analyzers;

import com.google.common.collect.ImmutableList;
import com.google.shipshape.proto.NotesProto.Location;
import com.google.shipshape.proto.NotesProto.Note;
import com.google.shipshape.proto.ShipshapeContextProto.ShipshapeContext;
import com.google.shipshape.service.AnalyzerException;
import com.google.shipshape.service.StatelessAnalyzer;

/**
 * Analyzer that posts a custom message, specified as a flag. This can be used to, e.g., send global
 * notifications about the state of Shipshape.
 */
public class PostMessageAnalyzer extends StatelessAnalyzer {

  public static final String CATEGORY = "Shipshape";
  public static String POST_MESSAGE = "Hello World";

  @Override
  public String getCategory() {
    return CATEGORY;
  }

  @Override
  public ImmutableList<Note> analyze(ShipshapeContext shipshapeContext)
      throws AnalyzerException {
    Note note = Note.newBuilder()
        .setCategory(CATEGORY)
        .setDescription(POST_MESSAGE)
        .setLocation(Location.newBuilder()
            .setSourceContext(shipshapeContext.getSourceContext()))
        .build();
    return ImmutableList.of(note);
  }
}
