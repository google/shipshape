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

package com.google.devtools.kythe.proto;

import com.google.devtools.kythe.platform.rpc.Connection;
import com.google.devtools.kythe.platform.rpc.OutputChannel;
import com.google.devtools.kythe.proto.Analysis.FileData;
import com.google.devtools.kythe.proto.Analysis.FileInfo;

import java.io.IOException;

// Pseudo-generated KRPC interfaces for a FileDataService
public class FileDataServiceKrpc {

  private static final String SERVICE_NAME = "FileData";
  private static final String GET_METHOD_NAME = "/" + SERVICE_NAME + "/Get";

  public static FileDataServiceStub newStub(Connection conn) {
    return new FileDataServiceStub(conn);
  }

  public static FileDataServiceBlockingStub newBlockingStub(Connection conn) {
    return new FileDataServiceBlockingStub(conn);
  }

  public static interface FileDataService {
    public void getFileData(FileInfo info, OutputChannel<FileData> channel);
  }

  public static interface FileDataServiceBlockingClient {
    public FileData getFileData(FileInfo info) throws IOException;
  }

  public static class FileDataServiceStub implements FileDataService {
    private final Connection conn;

    private FileDataServiceStub(Connection conn) {
      this.conn = conn;
    }

    @Override
    public void getFileData(FileInfo request, OutputChannel<FileData> channel) {
      conn.channel(GET_METHOD_NAME, request, channel, FileData.class);
    }
  }

  public static class FileDataServiceBlockingStub implements FileDataServiceBlockingClient {
    private final Connection conn;

    private FileDataServiceBlockingStub(Connection conn) {
      this.conn = conn;
    }

    @Override
    public FileData getFileData(FileInfo request) throws IOException {
      return conn.call(GET_METHOD_NAME, request, FileData.class);
    }
  }
}
