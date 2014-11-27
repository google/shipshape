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

import com.google.common.base.Throwables;
import com.google.common.reflect.TypeToken;
import com.google.gson.JsonElement;
import com.google.gson.JsonObject;
import com.google.protobuf.Descriptors.Descriptor;
import com.google.protobuf.Descriptors.FieldDescriptor;
import com.google.protobuf.GeneratedMessage;

import java.lang.reflect.Field;
import java.lang.reflect.InvocationTargetException;
import java.lang.reflect.ParameterizedType;
import java.lang.reflect.Type;
import java.util.Arrays;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.function.Function;

/**
 * K-RPC method interface.
 *
 * @see Method.ServerImpl
 * @see Method.Simple
 * @see Method.Streaming
 */
public interface Method<I, O> {
  /** Sets the {@link Function} for a class to determine a {@link Method.Info}'s parameter names. */
  public static void setParametersDescriptor(Class<?> cls, Function<Class<?>, String[]> getParams) {
    ReflectionUtil.PARAMETER_DESCRIPTORS.put(cls, getParams);
  }

  /** Entry-point for an RPC call. This is usually called by a type-safe adapter. */
  public void baseCall(Context ctx, JsonObject params, OutputChannel<JsonElement> results)
      throws Exception;

  /** Returns a simple {@link Method} returning a single value. */
  public static <I, O> Simple<I, O> simple(final String name, final Simple<I, O> f,
      final Class<I> inputClass, final Class<O> outputClass) {
    final String[] params = ReflectionUtil.getFieldNames(inputClass);
    return new Simple<I, O>() {
      @Override
      public String getName() {
        return name;
      }
      @Override
      public String[] getParams() {
        return params;
      }
      @Override
      public Type getInputType() {
        return inputClass;
      }
      @Override
      public Type getOutputType() {
        return outputClass;
      }
      @Override
      public O call(Context ctx, I params) throws Exception {
        return f.call(ctx, params);
      }
    };
  }

  /** Returns a streaming {@link Method} returning 0+ results through an {@link OutputChannel}. */
  public static <I, O> Streaming<I, O> streaming(final String name, final Streaming<I, O> f,
      final Class<I> inputClass, final Class<O> outputClass) {
    final String[] params = ReflectionUtil.getFieldNames(inputClass);
    return new Streaming<I, O>() {
      @Override
      public String getName() {
        return name;
      }
      @Override
      public String[] getParams() {
        return params;
      }
      @Override
      public Type getInputType() {
        return inputClass;
      }
      @Override
      public Type getOutputType() {
        return outputClass;
      }
      @Override
      public void call(Context ctx, I params, OutputChannel<O> stream) throws Exception {
        f.call(ctx, params, stream);
      }
    };
  }

  /** Interface for {@link Method}s being registered with an RPC {@link Server}. */
  public interface ServerImpl<I, O> extends Method<I, O> {
    default public boolean isStreaming() {
      return true;
    }
    default public String getName() {
      return getClass().getSimpleName();
    }
    default public String[] getParams() {
      return ReflectionUtil.getFieldNames(getInputType());
    }
    default public Type getInputType() {
      return ReflectionUtil.getParameterTypes(getClass())[0];
    }
    default public Type getOutputType() {
      return ReflectionUtil.getParameterTypes(getClass())[1];
    }
    default public Info getInfo() {
      return new Info(getName(), getParams(), isStreaming());
    }
  }

  /** {@link Method} returns 0 or more results through an {@link OutputChannel}. */
  @FunctionalInterface
  public interface Streaming<I, O> extends ServerImpl<I, O> {
    public void call(Context ctx, I params, OutputChannel<O> results) throws Exception;

    default public void baseCall(Context ctx, JsonObject params,
        OutputChannel<JsonElement> results) throws Exception {
      call(ctx,
          ctx.getGson().fromJson(params, getInputType()),
          results.transformingStream(ctx.getGson()::toJsonTree));
    }
  }

  /** {@link Method} returning exactly 1 result. */
  @FunctionalInterface
  public interface Simple<I, O> extends ServerImpl<I, O> {
    public O call(Context ctx, I params) throws Exception;

    default public void baseCall(Context ctx, JsonObject params,
        OutputChannel<JsonElement> results) throws Exception {
      O result = call(ctx, ctx.getGson().fromJson(params, getInputType()));
      results.send(ctx.getGson().toJsonTree(result));
      results.onCompleted();
    }
    default public boolean isStreaming() {
      return false;
    }
  }

  /** RPC {@link Method} information. */
  public static class Info {
    // This interface must be kept in sync w/ Protocol.MethodInfoSerializer

    public final String name;
    public final String[] params;
    public final boolean stream;

    public Info(String name, String[] params, boolean stream) {
      this.name = name;
      this.params = params;
      this.stream = stream;
    }
  }
}

class ReflectionUtil {
  static final Map<Class<?>, Function<Class<?>, String[]>> PARAMETER_DESCRIPTORS =
      new HashMap<>();

  static {
    Method.setParametersDescriptor(GeneratedMessage.class, (cls) -> {
          try {
            Descriptor descriptor = (Descriptor) cls.getMethod("getDescriptor").invoke(null);
            List<FieldDescriptor> fields = descriptor.getFields();
            String[] names = new String[fields.size()];
            int i = 0;
            for (FieldDescriptor f : fields) {
              names[i] = f.getName();
              i++;
            }
            return names;
          } catch (NoSuchMethodException|IllegalAccessException|InvocationTargetException e) {
            throw Throwables.propagate(e);
          }
        });
  }

  static String[] getFieldNames(Type t) {
    Class cls = TypeToken.of(t).getRawType();

    // Check for class specific parameter descriptions
    for (Class superCls : PARAMETER_DESCRIPTORS.keySet()) {
      if (superCls.isAssignableFrom(cls)) {
        String[] names = PARAMETER_DESCRIPTORS.get(superCls).apply(cls);
        Arrays.sort(names);
        return names;
      }
    }

    // Fallback to generic reflection method
    Field[] fields = cls.getDeclaredFields();
    String[] names = new String[fields.length];
    for (int i = 0; i < fields.length; i++) {
      names[i] = fields[i].getName();
    }
    Arrays.sort(names);
    return names;
  }

  static Type[] getParameterTypes(Class cls) {
    TypeToken t = getMethodInterface(cls);
    Type[] types = ((ParameterizedType) t.getType()).getActualTypeArguments();
    if (types.length != 2) {
      throw new IllegalStateException("unknown method type signature: " + Arrays.toString(types));
    }
    return types;
  }

  static TypeToken getMethodInterface(Type t) {
    TypeToken token = TypeToken.of(t);
    Set<TypeToken> interfaces = token.getTypes().interfaces();
    for (TypeToken i : interfaces) {
      if (i.getRawType().equals(Method.ServerImpl.class)) {
        return i;
      }
    }
    throw new IllegalArgumentException(
        "Type " + t + " does not implement the Method.ServerImpl interface");
  }
}
