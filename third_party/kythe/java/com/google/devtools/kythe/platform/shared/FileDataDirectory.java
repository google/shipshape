package com.google.devtools.kythe.platform.shared;

import com.google.common.io.Files;
import com.google.common.util.concurrent.Futures;
import com.google.devtools.kythe.common.PathUtil;

import java.io.File;
import java.util.concurrent.Future;

/**
 * {@link FileDataProvider} that looks up file data using the local filesystem. Each lookup only
 * uses the path (not the digest) and prefixes the path with a given root directory.
 */
public class FileDataDirectory implements FileDataProvider {
  private final String rootDirectory;

  public FileDataDirectory(String rootDirectory) {
    this.rootDirectory = rootDirectory;
  }

  @Override
  public Future<byte[]> startLookup(String path, String digest) {
    try {
      return Futures.immediateFuture(
          Files.asByteSource(new File(PathUtil.join(rootDirectory, path))).read());
    } catch (Throwable t) {
      return Futures.immediateFailedFuture(t);
    }
  }
}