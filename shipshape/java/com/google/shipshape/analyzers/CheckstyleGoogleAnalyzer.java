/*
 * Copyright 2015 Google Inc. All rights reserved.
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
import com.google.common.collect.ImmutableMap;
import com.google.shipshape.proto.NotesProto.Note;
import com.google.shipshape.proto.ShipshapeContextProto.ShipshapeContext;
import com.google.shipshape.service.AnalyzerException;
import com.google.shipshape.service.StatelessAnalyzer;

/**
 * A Shipshape analyzer that wraps Checkstyle (http://checkstyle.sourceforge.net/) configured with
 * Google style.
 */
public class CheckstyleGoogleAnalyzer extends StatelessAnalyzer {
  private static final String CATEGORY = "CheckstyleGoogle";

  @Override
  public String getCategory() {
    return CATEGORY;
  }

  @Override
  public ImmutableList<Note> analyze(ShipshapeContext context) throws AnalyzerException {
    ImmutableMap<String, String> javaFiles = CheckstyleUtils.getJavaFiles(context);
    if (javaFiles.isEmpty()) {
      return ImmutableList.of();
    }
    return CheckstyleUtils.runCheckstyle(context, getCategory(), javaFiles, "/google_checks.xml");
  }
}
