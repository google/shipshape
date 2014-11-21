package com.google.devtools.kythe.platform.shared;

import com.google.common.collect.ImmutableMap;
import com.google.common.util.concurrent.Futures;
import com.google.devtools.kythe.proto.Analysis.FileData;

import java.util.Map;
import java.util.concurrent.Future;

/**
 * {@link FileDataProvider} that looks up file data from a given {@link List} of {@link FileData}.
 */
public class FileDataCache implements FileDataProvider {
  private final Map<String, byte[]> fileContents;

  public FileDataCache(Iterable<FileData> fileData) {
    ImmutableMap.Builder<String, byte[]> builder = ImmutableMap.builder();
    for (FileData data : fileData) {
      builder.put(data.getInfo().getDigest(), data.getContent().toByteArray());
    }
    fileContents = builder.build();
  }

  @Override
  public Future<byte[]> startLookup(String path, String digest) {
    byte[] content = fileContents.get(digest);
    return content != null
        ? Futures.immediateFuture(content)
        : Futures.immediateFailedFuture(new RuntimeException(
              "Cache does not contain file for digest: " + digest));
  }
}
