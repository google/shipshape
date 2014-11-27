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

import com.google.common.collect.ForwardingMap;
import com.google.gson.Gson;
import com.google.gson.GsonBuilder;

import java.util.HashMap;
import java.util.Map;

/** Virtual collection of request-specific key/value metadata for an RPC call. */
public interface Context extends Map<String, String> {

  /** Returns a {@link Gson} capable of en/decoding {@link Protocol} messages. */
  public Gson getGson();

  /** {@link HashMap} backed {@link Context}. */
  public class Hash extends ForwardingMap<String, String> implements Context {
    private final Map<String, String> map = new HashMap<>();
    private final Gson gson;

    public Hash() {
      this(new GsonBuilder());
    }

    public Hash(GsonBuilder gson) {
      this(Protocol.constructGson(gson));
    }

    Hash(Gson gson) {
      this.gson = gson;
    }

    @Override
    public Gson getGson() {
      return gson;
    }

    @Override
    public Map<String, String> delegate() {
      return map;
    }
  }
}
