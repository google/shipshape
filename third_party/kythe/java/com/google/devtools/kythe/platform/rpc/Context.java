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
