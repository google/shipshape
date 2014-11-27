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

package com.google.devtools.kythe.platform.rpc;

import com.google.common.base.Preconditions;
import com.google.common.base.Strings;
import com.google.gson.Gson;
import com.google.gson.GsonBuilder;
import com.google.gson.JsonPrimitive;

import java.io.IOException;
import java.io.Reader;
import java.lang.reflect.Type;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.atomic.AtomicInteger;
import java.util.stream.Stream;
import java.util.stream.StreamSupport;

/** Connection interface to an arbitrary {@link Server} over a {@link Transport}. */
public class Connection {
  private static final ExecutorService ASYNC_EXECUTOR = Executors.newCachedThreadPool();

  private final Gson gson;
  private final Transport transport;
  private final AtomicInteger id = new AtomicInteger();

  public Connection(Transport transport) {
    this(null, transport);
  }

  public Connection(GsonBuilder gson, Transport transport) {
    this.gson = Protocol.constructGson(gson);
    this.transport = transport;
  }

  /** Transport for RPC {@link Protocol} messages. */
  public static interface Transport {
    /**
     * Sends the given {@link Protocol.Request} over the transport, encoding it with {@code gson},
     * and returns the resulting {@link Protocol.Response} stream.
     */
    public Reader sendRequest(Gson gson, Protocol.Request request) throws IOException;
  }

  public static void shutdownAsyncExecutors() {
    ASYNC_EXECUTOR.shutdown();
  }

  /** Calls the given RPC method, expecting a single result {@link Protocol.Response}. */
  public <T> T call(String method, Object params, Type typeOfT) throws IOException {
    Protocol.ResultReader<T> results =
        getResults(Protocol.Version.TWO, method, params, typeOfT);
    T result = results.nextResult();
    if (results.hasResult()) {
      throw new IllegalStateException("Method " + method + " returned more than 1 result");
    }
    return result;
  }

  /** Calls the given RPC method, returning a {@link Protocol.ResultReader} for the responses. */
  public <T> Protocol.ResultReader<T> reader(String method, Object params, Type typeOfT)
      throws IOException {
    return getResults(Protocol.Version.TWO_STREAMING, method, params, typeOfT);
  }

  /**
   * Calls the given RPC method, sending each result asynchronously to the given
   * {@link OutputChannel}.
   */
  public <T> void channel(String method, Object params, OutputChannel<T> channel, Type typeOfT) {
    ASYNC_EXECUTOR.execute(() -> {
          try {
            Protocol.ResultReader<T> results = reader(method, params, typeOfT);
            while (results.hasResult()) {
              channel.onValue(results.nextResult());
            }
            channel.onCompleted();
          } catch (Throwable t) {
            channel.onError(t);
          }
        });
  }

  /** Calls the given RPC method and returns the results in a {@link Stream}. */
  public <T> Stream<T> stream(String method, Object params, Type typeOfT) throws IOException {
    return StreamSupport.stream(reader(method, params, typeOfT), false);
  }

  private <T> Protocol.ResultReader<T> getResults(Protocol.Version version, String method,
      Object params, Type typeOfT) throws IOException {
    return new Protocol.ResultReader(gson,
        transport.sendRequest(gson, createRequest(version, method, params)), typeOfT);
  }

  private Protocol.Request createRequest(Protocol.Version version, String method, Object params) {
    Preconditions.checkArgument(version != null, "version must be non-null");
    Preconditions.checkArgument(!Strings.isNullOrEmpty(method), "method must be non-empty");
    Protocol.Request request = new Protocol.Request();
    request.version = version;
    request.method = method;
    if (params != null) {
      request.params = gson.toJsonTree(params).getAsJsonObject();
    }
    request.id = new JsonPrimitive(id.incrementAndGet());
    return request;
  }
}
