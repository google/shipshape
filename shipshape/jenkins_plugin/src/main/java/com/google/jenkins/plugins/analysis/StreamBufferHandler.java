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

import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.io.InputStream;

/**
 * Utility class for moving data from a stream to a buffer.
 */
public class StreamBufferHandler extends Thread {
  private InputStream stream;
  private ByteArrayOutputStream buffer;
  private ShipshapeStreamLogger logger;
  
  /**
   * Creates a handler moving data from a stream to a buffer.
   * @param logger The logger to use.
   * @param buffer Buffer to put input data in.
   * @param stream The stream to read from.
   */
  public StreamBufferHandler(ShipshapeStreamLogger logger, ByteArrayOutputStream buffer,
      InputStream stream) {
    this.logger = logger;
    this.stream = stream;
    this.buffer = buffer;
  }
  
  @Override
  public void run() {
    try {
      int next;
      while ((next = stream.read()) != -1) {
        buffer.write(next);
      }
    } catch (IOException e) {
      logger.log(String.format("Problem reading from stream, error: %s, data read so far: %s",
            e.getMessage(), buffer.toString()));
    }
  }
}