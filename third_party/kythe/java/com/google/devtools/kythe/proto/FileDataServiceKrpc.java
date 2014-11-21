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
