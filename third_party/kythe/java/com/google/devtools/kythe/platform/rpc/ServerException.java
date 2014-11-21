package com.google.devtools.kythe.platform.rpc;

import java.io.IOException;

public class ServerException extends IOException {
  public ServerException() { super(); }
  public ServerException(String message) { super(message); }
  public ServerException(String message, Throwable cause) { super(message, cause); }
  public ServerException(Throwable cause) { super(cause); }
}
