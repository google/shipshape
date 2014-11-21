package com.google.devtools.kythe.analyzers.base;

import com.google.common.base.Stopwatch;
import com.google.common.base.Strings;
import com.google.common.base.Throwables;
import com.google.devtools.kythe.common.FormattingLogger;
import com.google.devtools.kythe.platform.rpc.OutputChannel;
import com.google.devtools.kythe.platform.shared.AnalysisException;
import com.google.devtools.kythe.platform.shared.FileDataDirectory;
import com.google.devtools.kythe.platform.shared.FileDataProvider;
import com.google.devtools.kythe.platform.shared.RemoteFileData;
import com.google.devtools.kythe.proto.Analysis.AnalysisRequest;
import com.google.devtools.kythe.proto.Analysis.CompilationUnit;
import com.google.devtools.kythe.proto.CompilationAnalyzerKrpc;
import com.google.devtools.kythe.proto.Storage.Entry;
import com.google.devtools.kythe.proto.Storage.VName;
import com.google.protobuf.ByteString;

import java.io.File;

/**
 * Service method for a CompilationAnalyzer. Each analyzer receives an {@link AnalysisRequest}
 * protobuf as its argument and returns any number of {@link Entry} protobufs.
 *
 * A CompilationAnalyzer should implement the {@link #analyze(CompilationUnit, FileDataProvider,
 * FactEmitter)} method which provides the compilation to analyze, a provider for any file data not
 * embedded in the compilation, and an emitter for all of the analyzer's analysis outputs.
 */
public abstract class AbstractCompilationAnalyzer extends CompilationAnalyzerKrpc.AnalyzeMethod {
  private static final FormattingLogger logger =
      FormattingLogger.getLogger(AbstractCompilationAnalyzer.class);

  private final FileDataProvider defaultFileDataProvider;

  public AbstractCompilationAnalyzer(String defaultFileData) {
    defaultFileDataProvider = getFileDataProvider(defaultFileData);
  }

  @Override
  // TODO(schroederc): change stream to be an OutputChannel<AnalysisOutput> and nest Entries
  public final void analyze(AnalysisRequest request, OutputChannel<Entry> responses) {
    Stopwatch timer = Stopwatch.createUnstarted();
    CompilationUnit compilation = request.getCompilation();
    try {
      FileDataProvider fileDataProvider = !Strings.isNullOrEmpty(request.getFileDataService())
          ? getFileDataProvider(request.getFileDataService())
          : defaultFileDataProvider;
      if (fileDataProvider == null) {
        throw new AnalysisException("No file_data_service given for compilation: "
            + compilation.getVName());
      }

      logger.infofmt("Indexing {\n%s}", compilation.getVName());
      timer.start();
      analyze(compilation, fileDataProvider, new EntryChannel(responses));
    } catch (Throwable t) {
      logger.warningfmt(t, "exception while analyzing {\n%s}", compilation.getVName());
      Throwables.propagate(t);
    } finally {
      responses.onCompleted();
      if (timer.isRunning()) {
        timer.stop();
        logger.infofmt("Done with analysis in %s {\n%s}", timer, compilation.getVName());
      }
    }
  }

  /**
   * Analyze the given compilation, using the given {@link FileDataProvider} for retrieving any
   * necessary file data and the {@link FactEmitter} to emit all resulting facts.
   */
  protected abstract void analyze(CompilationUnit compilationUnit,
      FileDataProvider fileDataProvider, FactEmitter emitter) throws AnalysisException;

  private static FileDataProvider getFileDataProvider(String fileData) {
    if (Strings.isNullOrEmpty(fileData)) {
      return null;
    } else if (new File(fileData).isDirectory()) {
      return new FileDataDirectory(fileData);
    }
    return new RemoteFileData(fileData);
  }

  /** {@link FactEmitter} directly streaming to an {@link OutputChannel}. */
  private static class EntryChannel implements FactEmitter {
    private final OutputChannel<Entry> channel;

    public EntryChannel(OutputChannel<Entry> channel) {
      this.channel = channel;
    }

    @Override
    public void emit(VName source, String edgeKind, VName target,
        String factName, byte[] factValue) {
      Entry.Builder entry = Entry.newBuilder()
          .setSource(source)
          .setFactName(factName)
          .setFactValue(ByteString.copyFrom(factValue));
      if (edgeKind != null) {
        entry.setEdgeKind(edgeKind).setTarget(target);
      }
      channel.send(entry.build());
    }
  }
}
