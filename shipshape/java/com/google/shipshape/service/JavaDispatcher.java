package com.google.shipshape.service;

import com.google.devtools.kythe.platform.rpc.Context;
import com.google.devtools.kythe.platform.rpc.Method;
import com.google.devtools.kythe.platform.rpc.Server;
import com.google.devtools.kythe.platform.rpc.Service;
import com.google.shipshape.proto.NotesProto.Note;
import com.google.shipshape.proto.ShipshapeContextProto.ShipshapeContext;
import com.google.shipshape.proto.ShipshapeContextProto.Stage;
import com.google.shipshape.proto.ShipshapeRpcProto.AnalysisFailure;
import com.google.shipshape.proto.ShipshapeRpcProto.AnalyzeRequest;
import com.google.shipshape.proto.ShipshapeRpcProto.AnalyzeResponse;
import com.google.shipshape.proto.ShipshapeRpcProto.GetCategoryRequest;
import com.google.shipshape.proto.ShipshapeRpcProto.GetCategoryResponse;
import com.google.shipshape.proto.ShipshapeRpcProto.GetStageRequest;
import com.google.shipshape.proto.ShipshapeRpcProto.GetStageResponse;

import java.util.List;

/**
 * Helper class that wraps a set of Java-implemented analyzers implementing
 * {@link Analyzer} in a service.
 * The analyzers must all be using the same stage and state.
 * @param <T> state to pass to the analyzers this dispatching analyzer dispatches to.
 */
public class JavaDispatcher<T> {

  private static final String SERVICE_NAME = "AnalyzerService";

  private final List<Analyzer> analyzers;
  private final Stage stage;
  private final AnalyzerStateBuilder<T> stateBuilder;
  private static final ShipshapeLogger logger = ShipshapeLogger.getLogger(JavaDispatcher.class);

  public JavaDispatcher(
      List<Analyzer> analyzers,
      Stage stage,
      AnalyzerStateBuilder<T> stateBuilder) {
    this.stage = stage;
    this.stateBuilder = stateBuilder;
    this.analyzers = analyzers;
  }

  public List<Analyzer> getAnalyzers() {
    return analyzers;
  }

  public AnalyzeResponse analyze(Context ctx, final AnalyzeRequest analyzeRequest) {
    ShipshapeContext shipshapeContext = analyzeRequest.getShipshapeContext();
    AnalyzeResponse.Builder response = AnalyzeResponse.newBuilder();

    logger.info("Received request", analyzeRequest.getShipshapeContext());
    T state = null;
    if (stateBuilder != null) {
      try {
        state = stateBuilder.build(shipshapeContext);
      } catch (AnalyzerException e) {
        response.addFailure(AnalysisFailure.newBuilder()
            .setCategory("InternalShipshapeFailure")
            .setFailureMessage(e.getMessage()));
        logger.severe(e, "Failed to build state", shipshapeContext);
        return response.build();
      }
    }

    if (stateBuilder != null && state == null) {
      // This was not the state we needed.
      // Perhaps it was the wrong language, or the shipshape context did not have
      // the right information for the analyzers we are dispatching to.
      return response.build();
    }

    for (Analyzer analyzer : analyzers) {
      try {
        if (!analyzeRequest.getCategoryList().contains(analyzer.getCategory())) {
          continue;
        }
        logger.info("Running analyzer", shipshapeContext, analyzer.getCategory());
        List<Note> notes = analyzer.analyze(shipshapeContext, state);
        response.addAllNote(notes);
      } catch (AnalyzerException e) {
       response.addFailure(AnalysisFailure.newBuilder()
           .setCategory(analyzer.getCategory())
           .setFailureMessage(e.getMessage()));
        logger.severe(e, "Analyzer failed", shipshapeContext, analyzer.getCategory());
      }
    }

    return response.build();
  }

  public GetStageResponse getStage(Context ctx, GetStageRequest getStageRequest) {
    return GetStageResponse.newBuilder().setStage(stage).build();
  }

  public GetCategoryResponse getCategory(Context ctx, GetCategoryRequest getCategoryRequest) {
    GetCategoryResponse.Builder getCategoryResponse =  GetCategoryResponse.newBuilder();
    for (Analyzer analyzer : analyzers) {
      getCategoryResponse.addCategory(analyzer.getCategory());
    }
    return getCategoryResponse.build();
  }

  public void register(Server server) {
    server.addService(new Service.Map(SERVICE_NAME)
        .addMethod(Method.simple("Analyze", this::analyze,
                AnalyzeRequest.class, AnalyzeResponse.class))
        .addMethod(Method.simple("GetCategory", this::getCategory,
                GetCategoryRequest.class, GetCategoryResponse.class))
        .addMethod(Method.simple("GetStage", this::getStage,
                GetStageRequest.class, GetStageResponse.class)));
  }
}
