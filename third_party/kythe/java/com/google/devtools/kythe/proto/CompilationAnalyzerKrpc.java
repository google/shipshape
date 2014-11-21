package com.google.devtools.kythe.proto;

import com.google.devtools.kythe.platform.rpc.Connection;
import com.google.devtools.kythe.platform.rpc.Context;
import com.google.devtools.kythe.platform.rpc.Method;
import com.google.devtools.kythe.platform.rpc.OutputChannel;
import com.google.devtools.kythe.platform.rpc.Server;
import com.google.devtools.kythe.platform.rpc.Service;
import com.google.devtools.kythe.proto.Analysis.AnalysisRequest;
import com.google.devtools.kythe.proto.Storage.Entry;
import com.google.gson.reflect.TypeToken;

import java.io.IOException;
import java.util.List;

// Pseudo-generated KRPC interfaces for a CompilationAnalyzer
public class CompilationAnalyzerKrpc {

  private static final String SERVICE_NAME = "CompilationAnalyzer";
  private static final String ANALYZE_METHOD_NAME = "/" + SERVICE_NAME + "/Analyze";

  public static CompilationAnalyzerStub newStub(Connection conn) {
    return new CompilationAnalyzerStub(conn);
  }

  public static CompilationAnalyzerBlockingStub newBlockingStub(Connection conn) {
    return new CompilationAnalyzerBlockingStub(conn);
  }

  // TODO(schroederc): response type should be AnalysisOutput
  public static interface CompilationAnalyzer {
    public void analyze(AnalysisRequest info, OutputChannel<Entry> channel);
  }

  public static interface CompilationAnalyzerBlockingClient {
    public List<Entry> analyze(AnalysisRequest info) throws IOException;
  }

  public static class CompilationAnalyzerStub implements CompilationAnalyzer {
    private final Connection conn;

    private CompilationAnalyzerStub(Connection conn) {
      this.conn = conn;
    }

    @Override
    public void analyze(AnalysisRequest request, OutputChannel<Entry> channel) {
      conn.channel(ANALYZE_METHOD_NAME, request, channel, Entry.class);
    }
  }

  public static class CompilationAnalyzerBlockingStub implements CompilationAnalyzerBlockingClient {
    private final Connection conn;

    private CompilationAnalyzerBlockingStub(Connection conn) {
      this.conn = conn;
    }

    @Override
    public List<Entry> analyze(AnalysisRequest request) throws IOException {
      return conn.call(ANALYZE_METHOD_NAME, request, new TypeToken<List<Entry>>(){}.getType());
    }
  }

  public abstract static class AnalyzeMethod
      implements Method.Streaming<AnalysisRequest, Entry>, CompilationAnalyzer {
    /**
     * Returns the given {@link Server} after adding to it the CompilationAnalyzer service with
     * {@code this} Analyze method (i.e. adding the /CompilationAnalyzer/Analyze service method).
     */
    public Server addToServer(Server server) {
      return server
          .addService(new Service.Map(CompilationAnalyzerKrpc.SERVICE_NAME)
              .addMethod(this));
    }

    @Override
    public String getName() {
      return "Analyze";
    }

    @Override
    public final void call(Context ctx, AnalysisRequest request, OutputChannel<Entry> stream) {
      analyze(request, stream);
    }
  }
}
