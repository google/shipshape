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

package com.google.shipshape.service;

import com.google.shipshape.proto.ShipshapeContextProto.ShipshapeContext;

/**
 * Exception class for the analyzer plugins. Analyzers should throw this exception if the analysis
 * encounters an error.
 */
public class AnalyzerException extends Exception {

  private final String category;
  private final ShipshapeContext shipshapeContext;

  public AnalyzerException(String category, ShipshapeContext shipshapeContext, String message) {
    super(category + " failed: " + message);
    this.category = category;
    this.shipshapeContext = shipshapeContext;
  }

  public AnalyzerException(String category, ShipshapeContext shipshapeContext, String message,
      Throwable err) {
    super(category + " failed: " + message, err);
    this.category = category;
    this.shipshapeContext = shipshapeContext;
  }

  public AnalyzerException(ShipshapeContext shipshapeContext, String message) {
    this(null, shipshapeContext, message);
  }

  public AnalyzerException(ShipshapeContext shipshapeContext, String message, Throwable err) {
    this(null, shipshapeContext, message, err);
  }

  public String getCategory() {
    return category;
  }

  public ShipshapeContext getShipshapeContext() {
    return shipshapeContext;
  }
}
