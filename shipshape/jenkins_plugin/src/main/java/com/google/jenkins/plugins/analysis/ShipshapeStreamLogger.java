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

package com.google.jenkins.plugins.analysis;

import java.io.File;
import java.io.FileOutputStream;
import java.io.IOException;
import java.io.PrintStream;
import java.util.ArrayList;
import java.util.List;

/**
 * Utility logger class outputting to one or more streams.
 */
public class ShipshapeStreamLogger {
  private List<PrintStream> streamList;
  private List<PrintStream> closeableStreams;
  
  /**
   * Creates a logger handling outputting to multiple streams.
   * This will be created with no streams, call addStream to add as many as
   * are needed.
   */
  public ShipshapeStreamLogger() {
    streamList = new ArrayList<PrintStream>();
    closeableStreams = new ArrayList<PrintStream>();
  }
  
  /**
   * Adds an existing PrintStream to the logger.
   */
  public void addStream(PrintStream stream, boolean handleClose) {
    streamList.add(stream);
    if (handleClose) {
      closeableStreams.add(stream);
    }
  }
  
  /**
   * Adds a file stream to logger.
   * @param filePath The path to the file to create stream to.
   * @param append Whether to append to the file stream.
   * @throws IOException If creation of stream fails.
   */
  public void addStream(String filePath, boolean append) throws IOException {
    File file = new File(filePath);
    file.createNewFile();
    PrintStream fileStream = new PrintStream(new FileOutputStream(file), append);
    streamList.add(fileStream);
    closeableStreams.add(fileStream);
  }
  
  /**
   * Closes all closeable streams
   * @throws IOException if close failed for a stream.
   */
  public void closeStreams() throws IOException {
    for (PrintStream stream : closeableStreams) {
      stream.close();
    }
  }
  
  /**
   * Flushes all streams.
   * @throws IOException if flush failed for a stream.
   */
  public void flushStreams() throws IOException {
    for (PrintStream stream : streamList) {
      stream.flush();
    }
  }
  
  /**
   * Logs a message to available streams.
   * @param msg The message to log.
   */
  public void log(String msg) {
    for (PrintStream stream : streamList) {
      stream.println(String.format("[Shipshape] %s", msg));
    }
  }
  
  /**
   * Logs the message of an exception and then its stack trace.
   * @param e The exception to log.
   */
  public void logWithTrace(Exception e) {
    for (PrintStream stream : streamList) {
      stream.println(String.format("[Shipshape] %s", e.getMessage()));
      e.printStackTrace(stream);
    }
  }
}