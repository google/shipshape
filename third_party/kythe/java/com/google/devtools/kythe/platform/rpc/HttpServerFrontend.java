package com.google.devtools.kythe.platform.rpc;

import com.google.common.base.Preconditions;
import com.google.common.collect.ForwardingMap;
import com.google.gson.Gson;
import com.google.gson.JsonElement;

import com.sun.net.httpserver.Headers;
import com.sun.net.httpserver.HttpExchange;
import com.sun.net.httpserver.HttpHandler;
import com.sun.net.httpserver.HttpServer;

import java.io.IOException;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.io.OutputStreamWriter;
import java.io.PrintStream;
import java.io.PrintWriter;
import java.io.Reader;
import java.net.InetSocketAddress;
import java.nio.charset.Charset;
import java.nio.charset.StandardCharsets;
import java.util.HashMap;
import java.util.Iterator;
import java.util.LinkedList;
import java.util.List;
import java.util.Map;
import java.util.concurrent.Executors;

/**
 * {@link Server.Frontend} based on HTTP. A method call on a service is
 * represented by a POST with a JSON {@link Request} payload. The HTTP response
 * of the method call is a stream of JSON-encoded {@link Response}s.
 */
public class HttpServerFrontend implements Server.Frontend {
  public static final Charset DEFAULT_CHARSET = StandardCharsets.UTF_8;
  public static final int DEFAULT_PORT = 8080;

  private final Server server;
  private final int port;

  /**
   * Construct an {@link HttpServerFrontend} listening on the default port.
   *
   * @see HttpServerFrontend#DEFAULT_PORT
   */
  public HttpServerFrontend(Server server) {
    this(server, DEFAULT_PORT);
  }

  public HttpServerFrontend(Server server, int port) {
    Preconditions.checkArgument(server != null, "Server invalid");
    Preconditions.checkArgument(port > 0, "Port invalid");
    this.server = server;
    this.port = port;
  }

  @Override
  public Server getServer() {
    return server;
  }

  @Override
  public void run() throws IOException {
    InetSocketAddress addr = new InetSocketAddress(port);
    HttpServer server = HttpServer.create(addr, 0);

    server.createContext("/", new Handler());
    server.setExecutor(Executors.newCachedThreadPool());
    server.start();
  }

  private static Charset getContentCharset(String contentEncoding) {
    try {
      return Charset.forName(contentEncoding);
    } catch (Throwable t) {
      return DEFAULT_CHARSET;
    }
  }

  private class Handler implements HttpHandler {
    @Override
    public void handle(HttpExchange exchange) throws IOException {
      PrintWriter responseStream = null;
      Protocol.ResponseWriter wr = null;
      try {
        if (!exchange.getRequestMethod().equals("POST")) {
          exchange.sendResponseHeaders(405, -1);
          return;
        }

        HttpContext ctx = new HttpContext(server.getGson(), exchange.getRequestHeaders());
        Charset contentCharset = getContentCharset(ctx.get("Content-Encoding"));
        Protocol.Request request;
        try (
            InputStream body = exchange.getRequestBody();
            Reader r = new InputStreamReader(exchange.getRequestBody(), contentCharset)) {
          request = server.getGson().fromJson(r, Protocol.Request.class);
        }

        Headers headers = exchange.getResponseHeaders();
        headers.add("Content-Type", "application/json; charset=" + DEFAULT_CHARSET);

        responseStream = new PrintWriter(
            new OutputStreamWriter(exchange.getResponseBody(), DEFAULT_CHARSET), true);
        wr = new Protocol.ResponseWriter(server.getGson(), request, responseStream);
        exchange.sendResponseHeaders(200, 0);

        Method.ServerImpl method = server.getMethod(request.method);
        if (method == null) {
          wr.writeError(Protocol.Error.Code.METHOD_NOT_FOUND,
              "Request for unknown service method: " + request.method);
          return;
        }

        boolean streaming = method.getInfo().stream;
        if (!streaming) {
          // Downgrade protocol
          request.version = Protocol.Version.TWO;
        }

        // Call service method
        Iterator<JsonElement> results = server.handleRequest(ctx, method, request.params);
        switch (request.version) {
          case TWO:
            List<JsonElement> resultsList = new LinkedList<>();
            while (results.hasNext()) {
              resultsList.add(results.next());
            }

            if (streaming) {
              // If the client requested V2, but the method is streaming, we
              // must collect the results into a JSON array.
              wr.writeResult(server.getGson().toJsonTree(resultsList));
            } else if (resultsList.size() == 1) {
              // If the client requested V2 and the method is non-streaming, we
              // just return the result.
              wr.writeResult(resultsList.get(0));
            } else {
              throw new IllegalStateException(
                  "non-streaming method did not return single result: " + resultsList);
            }
            break;
          case TWO_STREAMING:
            while (results.hasNext()) {
              wr.writeResult(results.next());
            }
            wr.writeSuccess();
            break;
          default:
            throw new IllegalStateException("unknown protocol version: " + request.version);
        }
      } catch (Protocol.Error e) {
        if (wr == null) {
          internalServerError(exchange, e);
        } else {
          wr.writeError(e);
        }
      } catch (IOException ioe) {
        throw ioe;
      } catch (Throwable t) {
        if (wr == null) {
          internalServerError(exchange, t);
        } else {
          t.printStackTrace();
          wr.writeError(t);
        }
      } finally {
        if (responseStream != null) {
          responseStream.close();
        }
        exchange.close();
      }
    }
  }

  private static void internalServerError(HttpExchange exchange, Throwable t) throws IOException {
    System.err.println("Internal server error: ");
    t.printStackTrace();
    exchange.sendResponseHeaders(500, 0);
    t.printStackTrace(new PrintStream(exchange.getResponseBody()));
  }

  private static class HttpContext extends ForwardingMap<String, String> implements Context {
    private final Map<String, String> headers = new HashMap<>();
    private final Gson gson;

    public HttpContext(Gson gson, Headers headers) {
      this.gson = gson;
      for (String key : headers.keySet()) {
        this.headers.put(key, headers.getFirst(key));
      }
    }

    @Override
    public Gson getGson() {
      return gson;
    }

    @Override
    protected Map<String, String> delegate() {
      return headers;
    }
  }
}
