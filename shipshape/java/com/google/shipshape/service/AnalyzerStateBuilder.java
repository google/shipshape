package com.google.shipshape.service;

import com.google.shipshape.proto.ShipshapeContextProto.ShipshapeContext;

public interface AnalyzerStateBuilder<T> {
  T build(ShipshapeContext context) throws AnalyzerException;
}
