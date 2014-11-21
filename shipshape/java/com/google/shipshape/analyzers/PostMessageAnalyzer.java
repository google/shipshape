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
