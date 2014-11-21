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
