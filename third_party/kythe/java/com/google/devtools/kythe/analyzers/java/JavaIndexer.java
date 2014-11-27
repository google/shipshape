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

package com.google.devtools.kythe.analyzers.java;

import com.google.common.base.Throwables;
import com.google.devtools.kythe.analyzers.base.FactEmitter;
import com.google.devtools.kythe.extractors.shared.CompilationDescription;
import com.google.devtools.kythe.extractors.shared.IndexInfoUtils;
import com.google.devtools.kythe.platform.java.JavacAnalysisDriver;
import com.google.devtools.kythe.platform.shared.AnalysisException;
import com.google.devtools.kythe.platform.shared.FileDataCache;
import com.google.devtools.kythe.proto.Storage.Entry;
import com.google.devtools.kythe.proto.Storage.VName;
import com.google.protobuf.ByteString;

import java.io.IOException;
import java.io.OutputStream;
import java.io.OutputStreamWriter;
import java.util.Arrays;

/** Binary to run Kythe's Java index over a single .kindex file, emitting entries to STDOUT. */
public class JavaIndexer {
  public static void main(String[] args) throws AnalysisException, IOException {
    if (args.length != 1) {
      System.err.println("Java indexer must only receive 1 argument; got " + Arrays.toString(args));
      usage(1);
    } else if (args[0] == "--help" || args[0] == "-h") {
      usage(0);
    }

    CompilationDescription desc = IndexInfoUtils.readIndexInfoFromFile(args[0]);

    try (OutputStream stream = System.out;
        OutputStreamWriter writer = new OutputStreamWriter(stream)) {
      new JavacAnalysisDriver()
          .analyze(new KytheJavacAnalyzer(new StreamFactEmitter(writer)),
              desc.getCompilationUnit(),
              new FileDataCache(desc.getFileContents()),
              false);
    }
  }

  private static void usage(int exitCode) {
    System.err.println("usage: java_indexer kindex-file");
    System.exit(exitCode);
  }

  /** {@link FactEmitter} directly streaming to an {@link OutputValueStream}. */
  private static class StreamFactEmitter implements FactEmitter {
    private final OutputStreamWriter writer;

    public StreamFactEmitter(OutputStreamWriter writer) {
      this.writer = writer;
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

      try {
        entry.build().writeDelimitedTo(System.out);
      } catch (IOException ioe) {
        Throwables.propagate(ioe);
      }
    }
  }
}
