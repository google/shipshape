package com.google.devtools.kythe.platform.shared;

import java.util.concurrent.Future;

/**
 * Arbitrary provider of file data that could be backed by local/networked filesystems, cloud
 * storage, SQL database, etc.
 */
public interface FileDataProvider {
  /**
   * Returns a {@link Future<byte[]>} for the contents of the file described by the given path and
   * digest. At least one of path or digest must be specified for each file lookup.
   */
  public Future<byte[]> startLookup(String path, String digest);
}