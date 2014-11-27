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
