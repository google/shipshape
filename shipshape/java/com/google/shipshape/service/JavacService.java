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

import com.google.common.base.Throwables;
import com.google.devtools.kythe.common.FormattingLogger;
import com.google.devtools.kythe.platform.rpc.HttpServerFrontend;
import com.google.devtools.kythe.platform.rpc.Server;
import com.google.devtools.kythe.platform.java.JavaCompilationDetails;
import com.google.shipshape.analyzers.ErrorProneAnalyzer;
import com.google.shipshape.proto.ShipshapeContextProto.Stage;

import com.beust.jcommander.JCommander;
import com.beust.jcommander.Parameter;

import java.util.ArrayList;

/**
 * An analyzer service for analyzers that use javac.
 * This service uses a DetailsBuilder that creates a JavaCompilationDetails
 * for each compilation unit.
 */
public class JavacService {

  @Parameter(names = "--port", description = "port for RPC server")
  private int port = 10006;

  @Parameter(names = "--javac_out",
      description = "whether to log the output from running javac. False will silence the compiler.")
  private boolean javacOut = false;

  private static FormattingLogger logger = FormattingLogger.getLogger(JavacService.class);

  public static void main(String[] args) {
    try {
      JavacService service = new JavacService();
      new JCommander(service, args);

      ArrayList<Analyzer> analyzers = new ArrayList<>();
      analyzers.add(new ErrorProneAnalyzer());

      final Server server = new Server();
      JavaDispatcher<JavaCompilationDetails> dispatcher =
          new JavaDispatcher<>(analyzers, Stage.POST_BUILD, new JavacStateBuilder(service.javacOut));
      dispatcher.register(server);
      logger.infofmt("Starting service at %d", service.port);
      new HttpServerFrontend(server, service.port).run();
    } catch (Throwable t) {
      logger.severefmt(t, "Error starting service");
      Runtime.getRuntime().halt(1);
    }
  }
}
