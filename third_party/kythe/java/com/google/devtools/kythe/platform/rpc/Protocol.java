package com.google.devtools.kythe.platform.rpc;

import static java.util.Spliterators.AbstractSpliterator;

import com.google.common.base.Throwables;
import com.google.gson.Gson;
import com.google.gson.GsonBuilder;
import com.google.gson.JsonDeserializationContext;
import com.google.gson.JsonDeserializer;
import com.google.gson.JsonElement;
import com.google.gson.JsonObject;
import com.google.gson.JsonParseException;
import com.google.gson.JsonPrimitive;
import com.google.gson.JsonSerializationContext;
import com.google.gson.JsonSerializer;
import com.google.gson.JsonStreamParser;
import com.google.gson.protobuf.ProtoTypeAdapter;
import com.google.protobuf.ByteString;
import com.google.protobuf.GeneratedMessage;
import com.google.protobuf.ProtocolMessageEnum;

import java.io.IOException;
import java.io.PrintWriter;
import java.io.Reader;
import java.lang.reflect.Type;
import java.util.Base64;
import java.util.NoSuchElementException;
import java.util.Optional;
import java.util.Spliterator;
import java.util.function.Consumer;

/** Types and constants for the JSON K-RPC protocol. */
public class Protocol {
  private static final JsonObject EMPTY_PARAMETERS = new JsonObject();

  /** JSON-RPC request to a server. */
  public static class Request {
    public Version version;
    public JsonElement id;

    public String method;
    public JsonObject params = EMPTY_PARAMETERS;
  }

  /** JSON-RPC response to a client's {@link Request}. */
  public static class Response {
    public Version version;
    public JsonElement id;

    public JsonElement result;
    public Error error;
    public boolean success;
  }

  /** JSON-RPC {@link Response} error. */
  public static class Error extends IOException {
    public Code code;
    public String message;
    public JsonElement data;

    public static enum Code {
      PARSING(-32700),
      INVALID_REQUEST(-32600),
      METHOD_NOT_FOUND(-32601),
      INVALID_PARAMS(-32602),
      INTERNAL(-32603),
      APPLICATION(0);

      final int num;
      Code(int num) {
        this.num = num;
      }

      public static Code fromNum(int num) {
        switch (num) {
          case -32700: return PARSING;
          case -32600: return INVALID_REQUEST;
          case -32601: return METHOD_NOT_FOUND;
          case -32602: return INVALID_PARAMS;
          case -32603: return INTERNAL;
          default: return APPLICATION;
        }
      }
    }
  }

  /** Protocol version enum. */
  public static enum Version {
    TWO("2.0"),
    TWO_STREAMING("2.0 streaming");

    final String version;
    Version(String version) {
      this.version = version;
    }

    public static Version fromString(String str) {
      switch (str) {
        case "2.0": return TWO;
        case "2.0 streaming": return TWO_STREAMING;
        default:
          throw new IllegalArgumentException("unknown version: " + str);
      }
    }
  }

  /** Returns a {@link Gson} capable of en/decoding {@link Protocol} messages. */
  public static Gson constructGson(GsonBuilder builder) {
    if (builder == null) {
      builder = new GsonBuilder();
    }
    return builder
        .registerTypeAdapter(Error.class, new ErrorTypeAdapter())
        .registerTypeAdapter(Request.class, new RequestTypeAdapter())
        .registerTypeAdapter(Response.class, new ResponseTypeAdapter())
        .registerTypeAdapter(Method.Info.class, new MethodInfoSerializer())
        .registerTypeAdapter(Version.class, new VersionTypeAdapter())
        .registerTypeHierarchyAdapter(ProtocolMessageEnum.class, new ProtoEnumTypeAdapter())
        .registerTypeHierarchyAdapter(GeneratedMessage.class, new ProtoTypeAdapter())
        .registerTypeHierarchyAdapter(ByteString.class, new ByteStringTypeAdapter())
        .registerTypeAdapter(byte[].class, new ByteArrayTypeAdapter())
        .create();
  }

  /** Reader for {@link Response} results. */
  static class ResultReader<T> extends AbstractSpliterator<T> {
    private final Gson gson;
    private final JsonStreamParser responses;
    private final Type typeOfT;

    private Optional<Response> next = Optional.empty();

    public ResultReader(GsonBuilder gson, Reader reader, Type typeOfT) {
      this(constructGson(gson), reader, typeOfT);
    }

    ResultReader(Gson gson, Reader reader, Type typeOfT) {
      super(Long.MAX_VALUE, Spliterator.NONNULL|Spliterator.ORDERED);
      this.gson = gson;
      this.responses = new JsonStreamParser(reader);
      this.typeOfT = typeOfT;
    }

    /** Returns {@code true} if there is a result left to read. */
    public boolean hasResult() {
      if (next.isPresent()) {
        return true;
      } else if (!responses.hasNext()) {
        return false;
      }
      JsonElement el = responses.next();
      Response resp = gson.fromJson(el, Response.class);
      if (resp.success) {
        return false;
      }
      next = Optional.of(resp);
      return true;
    }

    /**
     * Returns the next {@link Response} result. If the {@link Response} contained an error, it is
     * thrown.
     */
    public T nextResult() throws Error {
      if (!hasResult()) {
        throw new NoSuchElementException();
      }

      Response resp = next.get();
      next = Optional.empty();

      if (resp.error != null) {
        throw (Error) resp.error.fillInStackTrace();
      }

      return gson.fromJson(resp.result, typeOfT);
    }

    @Override
    public boolean tryAdvance(Consumer<? super T> action) {
      if (!hasResult()) {
        return false;
      }
      try {
        action.accept(nextResult());
        return true;
      } catch (Error err) {
        throw Throwables.propagate(err);
      }
    }
  }

  /** Utility to write RPC responses to a stream for a particular {@link Request}. */
  static class ResponseWriter {
    private final Gson gson;
    private final Request req;
    private final PrintWriter writer;

    public ResponseWriter(GsonBuilder gson, Request req, PrintWriter writer) {
      this(constructGson(gson), req, writer);
    }

    ResponseWriter(Gson gson, Request req, PrintWriter writer) {
      this.gson = gson;
      this.req = req;
      this.writer = writer;
    }

    /** Writes the given element as a {@link Response} result. */
    public void writeResult(JsonElement el) {
      Response resp = new Response();
      resp.result = el;
      write(resp);
    }

    /** Writes a trailing {@link Response} marking the success of the stream. */
    public void writeSuccess() {
      Response resp = new Response();
      resp.success = true;
      write(resp);
    }

    /** Constructs new {@link Error} and writes it as a {@link Response}. */
    public void writeError(Error.Code code, String message, JsonElement data) {
      Error err = new Error();
      err.code = code;
      err.message = message;
      err.data = data;
      writeError(err);
    }

    /** Constructs new {@link Error} without data and writes it as a {@link Response}. */
    public void writeError(Error.Code code, String message) {
      writeError(code, message, null);
    }

    /** Writes the error as a {@link Response}. */
    public void writeError(Error err) {
      Response resp = new Response();
      resp.error = err;
      write(resp);
    }

    /** Constructs new {@link Error} and writes it as a {@link Response}. */
    public void writeError(Throwable t) {
      writeError(Error.Code.APPLICATION, t.toString());
    }

    private void write(Response resp) {
      resp.id = req.id;
      resp.version = req.version;
      writer.println(gson.toJson(resp));
    }
  }

  //// Gson type adapters for protocol types

  private static class ErrorTypeAdapter
      implements JsonSerializer<Error>, JsonDeserializer<Error> {
    @Override
    public JsonElement serialize(Error err, Type t, JsonSerializationContext ctx) {
      JsonObject obj = new JsonObject();
      obj.addProperty("code", err.code.num);
      obj.addProperty("message", err.message);
      obj.add("data", err.data);
      return obj;
    }

    @Override
    public Error deserialize(JsonElement json, Type typeOfT,
        JsonDeserializationContext ctx) throws JsonParseException {
      JsonObject obj = json.getAsJsonObject();
      Error err = new Error();
      err.setStackTrace(new StackTraceElement[0]);
      err.code = Error.Code.fromNum(obj.getAsJsonPrimitive("code").getAsInt());
      err.message = obj.getAsJsonPrimitive("message").getAsString();
      err.data = obj.get("data");
      return err;
    }
  }

  private static class VersionTypeAdapter
      implements JsonSerializer<Version>, JsonDeserializer<Version> {
    @Override
    public JsonElement serialize(Version v, Type t, JsonSerializationContext ctx) {
      return new JsonPrimitive(v.version);
    }

    @Override
    public Version deserialize(JsonElement json, Type t, JsonDeserializationContext ctx) {
      return Version.fromString(json.getAsJsonPrimitive().getAsString());
    }
  }

  private static class RequestTypeAdapter
      implements JsonSerializer<Request>, JsonDeserializer<Request> {
    @Override
    public JsonElement serialize(Request req, Type t, JsonSerializationContext ctx) {
      JsonObject obj = new JsonObject();
      obj.add("id", req.id);
      obj.add("jsonrpc", ctx.serialize(req.version));
      obj.addProperty("method", req.method);

      if (req.params != null) {
        obj.add("params", req.params);
      }
      return obj;
    }

    @Override
    public Request deserialize(JsonElement json, Type t, JsonDeserializationContext ctx) {
      JsonObject obj = json.getAsJsonObject();
      Request req = new Request();
      req.id = obj.get("id");
      req.version = ctx.deserialize(obj.get("jsonrpc"), Version.class);
      req.method = obj.getAsJsonPrimitive("method").getAsString();

      if (obj.has("params")) {
        req.params = obj.get("params").getAsJsonObject();
      }
      return req;
    }
  }

  private static class ResponseTypeAdapter
      implements JsonSerializer<Response>, JsonDeserializer<Response> {
    @Override
    public JsonElement serialize(Response resp, Type t, JsonSerializationContext ctx) {
      JsonObject obj = new JsonObject();
      obj.add("id", resp.id);
      obj.add("jsonrpc", ctx.serialize(resp.version));

      obj.add("result", resp.result);
      obj.add("error", ctx.serialize(resp.error));
      if (resp.success) {
        obj.addProperty("success", resp.success);
      }
      return obj;
    }

    @Override
    public Response deserialize(JsonElement json, Type t, JsonDeserializationContext ctx) {
      JsonObject obj = json.getAsJsonObject();
      Response resp = new Response();
      resp.id = obj.get("id");
      resp.version = ctx.deserialize(obj.get("jsonrpc"), Version.class);

      if (obj.has("result")) {
        resp.result = obj.get("result");
      }
      if (obj.has("error")) {
        resp.error = ctx.deserialize(obj.get("error"), Error.class);
      }
      if (obj.has("success")) {
        resp.success = obj.getAsJsonPrimitive("success").getAsBoolean();
      }
      return resp;
    }
  }

  private static class MethodInfoSerializer implements JsonSerializer<Method.Info> {
    @Override
    public JsonElement serialize(Method.Info info, Type t, JsonSerializationContext ctx) {
      JsonObject obj = new JsonObject();
      obj.addProperty("name", info.name);
      obj.add("params", ctx.serialize(info.params));
      if (info.stream) {
        obj.addProperty("stream", info.stream);
      }
      return obj;
    }
  }

  private static class ByteStringTypeAdapter
      implements JsonSerializer<ByteString>, JsonDeserializer<ByteString> {
    @Override
    public JsonElement serialize(ByteString str, Type t, JsonSerializationContext ctx) {
      return ctx.serialize(str.toByteArray());
    }

    @Override
    public ByteString deserialize(JsonElement json, Type typeOfT,
        JsonDeserializationContext context) throws JsonParseException {
      return ByteString.copyFrom((byte[]) context.deserialize(json, byte[].class));
    }
  }

  private static class ByteArrayTypeAdapter
      implements JsonSerializer<byte[]>, JsonDeserializer<byte[]> {
    private static final Base64.Encoder ENCODER = Base64.getEncoder();
    private static final Base64.Decoder DECODER = Base64.getDecoder();

    @Override
    public JsonElement serialize(byte[] arry, Type t, JsonSerializationContext ctx) {
      return new JsonPrimitive(ENCODER.encodeToString(arry));
    }

    @Override
    public byte[] deserialize(JsonElement json, Type typeOfT,
        JsonDeserializationContext context) throws JsonParseException {
      return DECODER.decode((String) context.deserialize(json, String.class));
    }
  }

  // Type adapter for bare protobuf enum values.
  private static class ProtoEnumTypeAdapter
      implements JsonSerializer<ProtocolMessageEnum>, JsonDeserializer<ProtocolMessageEnum> {
    @Override
    public JsonElement serialize(ProtocolMessageEnum e, Type t, JsonSerializationContext ctx) {
      return new JsonPrimitive(e.getNumber());
    }

    @Override
    public ProtocolMessageEnum deserialize(JsonElement json, Type t,
        JsonDeserializationContext ctx) throws JsonParseException {
      int num = json.getAsJsonPrimitive().getAsInt();
      Class<? extends ProtocolMessageEnum> enumClass = (Class<? extends ProtocolMessageEnum>) t;
      try {
        return (ProtocolMessageEnum) enumClass.getMethod("valueOf", int.class).invoke(null, num);
      } catch (Exception e) {
        throw new JsonParseException(e);
      }
    }
  }
}
