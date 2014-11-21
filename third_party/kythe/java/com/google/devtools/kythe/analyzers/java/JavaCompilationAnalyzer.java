package com.google.devtools.kythe.analyzers.java;

import com.google.devtools.kythe.analyzers.base.AbstractCompilationAnalyzer;
import com.google.devtools.kythe.analyzers.base.FactEmitter;
import com.google.devtools.kythe.platform.java.JavacAnalysisDriver;
import com.google.devtools.kythe.platform.rpc.HttpServerFrontend;
import com.google.devtools.kythe.platform.rpc.Server;
import com.google.devtools.kythe.platform.shared.AnalysisException;
import com.google.devtools.kythe.platform.shared.FileDataProvider;
import com.google.devtools.kythe.proto.Analysis.CompilationUnit;

import com.beust.jcommander.JCommander;
import com.beust.jcommander.Parameter;

import java.io.IOException;

/** Binary to run a K-RPC {@link Server} for Kythe's Java {@link AbstractCompilationAnalyzer}. */
public class JavaCompilationAnalyzer {

  @Parameter(names = "--port", description = "port for RPC server; 0 indicates don't launch one")
  private int port = 0;

  @Parameter(names = "--file_data",
      description = "Address of a default FileDataService or filesystem root for required files")
  private String fileDataService = System.getProperty("user.dir");

  public void run() throws IOException {
    final Server server = new JavaIndexer(fileDataService)
        .addToServer(new Server());
    new HttpServerFrontend(server, port).run();
  }

  public static void main(String[] args) throws AnalysisException, IOException {
    JavaCompilationAnalyzer indexerServer = new JavaCompilationAnalyzer();
    new JCommander(indexerServer, args);
    indexerServer.run();
  }


  /** CompilationAnalyzer implementation for Java. */
  private static class JavaIndexer extends AbstractCompilationAnalyzer {
    private final JavacAnalysisDriver driver = new JavacAnalysisDriver();

    public JavaIndexer(String fileDataAddress) {
      super(fileDataAddress);
    }

    @Override
    protected void analyze(CompilationUnit compilationUnit, FileDataProvider fileDataProvider,
        FactEmitter emitter) throws AnalysisException {
      driver.analyze(new KytheJavacAnalyzer(emitter), compilationUnit, fileDataProvider, false);
    }
  }
}
