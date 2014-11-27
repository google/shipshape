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

import java.util.HashMap;
import java.util.LinkedList;
import java.util.List;

/** Collection of RPC methods under a single service name. */
public interface Service {
  /** Returns the {@link Method} of the given name; {@code null} otherwise. */
  public Method getMethod(String methodName);

  /** {@link Server} information about a {@link Service}. */
  public static class Info {
    public final String name;
    public final List<Method.Info> methods;

    public Info(String name, List<Method.Info> methods) {
      this.name = name;
      this.methods = methods;
    }
  }

  /** Interface for a {@link Service} to be registered with an RPC {@link Server}. */
  public interface ServerImpl extends Service {
    public String getName();
    public Info getInfo();
    @Override
    public Method.ServerImpl getMethod(String methodName);
  }

  /** Static {@link Service} mapping string names to their corresponding {@link Method}s. */
  public static class Map implements ServerImpl {
    private final String name;
    private final java.util.Map<String, Method.ServerImpl> methods = new HashMap<>();

    /** Create a new {@link Service} with no registered {@link Method}s. */
    public Map(String name) {
      Preconditions.checkArgument(!Strings.isNullOrEmpty(name), "service names must be non-empty");
      this.name = name;
    }

    /**
     * Add the given {@link Method} to the {@link Service}. An exception is
     * thrown if the method's name is already registered.
     */
    public final Map addMethod(Method.ServerImpl method) {
      Preconditions.checkArgument(!methods.containsKey(method.getName()),
          getClass() + " already contains method: " + method.getName());
      methods.put(method.getName(), method);
      return this;
    }

    @Override
    public final Method.ServerImpl getMethod(String methodName) {
      return methods.get(methodName);
    }

    @Override
    public final String getName() {
      return name;
    }

    @Override
    public final Service.Info getInfo() {
      List<Method.Info> methods = new LinkedList<>();
      for (Method.ServerImpl method : this.methods.values()) {
        methods.add(method.getInfo());
      }
      return new Service.Info(getName(), methods);
    }
  }
}
