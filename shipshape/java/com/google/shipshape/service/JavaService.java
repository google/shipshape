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

package com.google.shipshape.service;

import com.google.devtools.kythe.common.FormattingLogger;
import com.google.shipshape.util.rpc.HttpServerFrontend;
import com.google.shipshape.util.rpc.Server;
import com.google.shipshape.analyzers.CheckstyleGoogleAnalyzer;
import com.google.shipshape.analyzers.PostMessageAnalyzer;
import com.google.shipshape.proto.ShipshapeContextProto.Stage;

import com.beust.jcommander.JCommander;
import com.beust.jcommander.Parameter;

import java.util.ArrayList;

/**
 * An analyzer service for java analyzers not needing compilation details.
 */
class JavaService {

  @Parameter(names = "--port", description = "port for RPC server")
  private int port = 10008;

  private static FormattingLogger logger = FormattingLogger.getLogger(JavaService.class);

  public static void main(String[] args) {
    try {
      JavaService service = new JavaService();
      new JCommander(service, args);

      ArrayList<Analyzer> analyzers = new ArrayList<>();
// TODO(ciera): uncomment when #110 is fixed
//      analyzers.add(new PostMessageAnalyzer());
      analyzers.add(new CheckstyleGoogleAnalyzer());

      final Server server = new Server();
      JavaDispatcher<Object> dispatcher = new JavaDispatcher<>(analyzers, Stage.PRE_BUILD, null);
      dispatcher.register(server);
      logger.infofmt("Starting java service at %d", service.port);
      new HttpServerFrontend(server, service.port).run();
    } catch (Throwable t) {
      logger.severefmt(t, "Error starting service");
      Runtime.getRuntime().halt(1);
    }
  }
}
