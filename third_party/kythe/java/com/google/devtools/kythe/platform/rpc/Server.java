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
import com.google.gson.JsonElement;
import com.google.gson.JsonObject;

import java.io.IOException;
import java.util.HashMap;
import java.util.Iterator;
import java.util.LinkedList;
import java.util.List;
import java.util.Map;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

/** Core of the Kythe server platform. */
public class Server {

  private final Map<String, Service.ServerImpl> services = new HashMap<>();
  private final Gson gson;

  /** Constructs a new {@link Server} without any registered {@link Service}s. */
  public Server() {
    this(null);
  }

  /**
   * Constructs a new {@link Server} without any registered {@link Service}s that will use the given
   * {@link GsonBuilder} as the base {@link Gson} for en/decoding protocol messages.
   */
  public Server(GsonBuilder gson) {
    this.gson = Protocol.constructGson(gson);
    addService(new Service.Map("ServerInfo")
        .addMethod(Method.simple("List", this::getListings, Object.class, List.class)));
  }

  /** Returns the {@link Server}s associated {@link Gson} for en/decoding protocol messages. */
  public Gson getGson() {
    return gson;
  }

  /** Adds the given {@link Service} to the {@link Server}. */
  public Server addService(Service.ServerImpl service) {
    Preconditions.checkArgument(!services.containsKey(service.getName()),
        getClass() + " already contains service: " + service.getName());
    services.put(service.getName(), service);
    return this;
  }

  /**
   * Represents a frontend for readings server requests and streaming back the outputs.
   *
   * @see HttpServerFrontend
   */
  public interface Frontend {
    public void run() throws IOException;
    public Server getServer();
  }

  /**
   * The internal handler for a server request. {@link Frontend}s should expose this through their
   * respective interfaces. {@link Frontend}s should expose this through their respective
   * interfaces.
   */
  protected Iterator<JsonElement> handleRequest(Context ctx, Method method, JsonObject params)
      throws Exception {
    QueuedOutputChannel stream = new QueuedOutputChannel();
    method.baseCall(ctx, params, stream);
    return stream.getOutputs();
  }

  /**
   * Returns the method associated with the given {@link String} representation of a
   * {@link ServiceMethodUri}.
   */
  protected Method.ServerImpl getMethod(String uri) {
    return getMethod(new ServiceMethodUri(uri));
  }

  /** Returns the method associated with the given {@link ServiceMethodUri}. */
  protected Method.ServerImpl getMethod(ServiceMethodUri uri) {
    return getService(uri.getServiceName()).getMethod(uri.getMethodName());
  }

  /**
   * Internal implementation for the service listings method exposed by the
   * {@link ServerInfoService}.
   */
  protected List<Service.Info> getListings(Context ctx, Object params) {
    List<Service.Info> listings = new LinkedList<>();
    for (Service.ServerImpl service : services.values()) {
      listings.add(service.getInfo());
    }
    return listings;
  }

  private Service.ServerImpl getService(String serviceName) {
    Preconditions.checkArgument(!Strings.isNullOrEmpty(serviceName),
        "serviceName must be non-empty");
    Service.ServerImpl service = services.get(serviceName);
    Preconditions.checkArgument(service != null,
        String.format("Service '%s' could not be recognized.", serviceName));
    return service;
  }

  protected static class ServiceMethodUri {
    /**
     * Pattern matching a uri of the form /<ServiceName>/<MethodName> where <ServiceName> and
     * <MethodName> are non-empty strings and <MethodName> contains no '/'s.
     */
    private static final Pattern URI_PATTERN = Pattern.compile("/(.+)/([^/]+)");

    private final String serviceName;
    private final String methodName;

    public ServiceMethodUri(String uri) {
      Preconditions.checkArgument(!Strings.isNullOrEmpty(uri), "uri must be non-empty");
      Matcher m = URI_PATTERN.matcher(uri);
      Preconditions.checkArgument(m.matches(),
          String.format("'%s' is not a valid ServiceMethodUri", uri));
      serviceName = m.group(1);
      methodName = m.group(2);
    }

    public ServiceMethodUri(String serviceName, String methodName) {
      this.serviceName = serviceName;
      this.methodName = methodName;
    }

    public String getServiceName() {
      return serviceName;
    }

    public String getMethodName() {
      return methodName;
    }

    @Override
    public String toString() {
      return String.format("/%s/%s", serviceName, methodName);
    }
  }
}
