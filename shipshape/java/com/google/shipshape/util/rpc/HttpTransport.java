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

package com.google.shipshape.util.rpc;

import static com.google.shipshape.util.rpc.HttpServerFrontend.DEFAULT_CHARSET;

import com.google.common.base.Preconditions;
import com.google.common.base.Strings;
import com.google.gson.Gson;

import java.io.IOException;
import java.io.InputStreamReader;
import java.io.OutputStream;
import java.io.OutputStreamWriter;
import java.io.Reader;
import java.net.HttpURLConnection;
import java.net.MalformedURLException;
import java.net.URL;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

/** K-RPC over HTTP {@link Connection.Transport}. */
public class HttpTransport implements Connection.Transport {
  private static final Pattern ADDRESS_PATTERN = Pattern.compile("\\s*(.*?):(\\d+)\\s*");

  private final URL url;

  private HttpTransport(URL url) {
    Preconditions.checkNotNull(url, "url must be non-null");
    this.url = url;
  }

  public static Connection newHttpConnection(URL url) {
    return new Connection(new HttpTransport(url));
  }

  /** Returns a {@link Connection} to the given host over HTTP. */
  public static Connection newHttpConnection(String host, int port) {
    Preconditions.checkArgument(!Strings.isNullOrEmpty(host), "host must be non-empty");
    Preconditions.checkArgument(port > 0, "port must be positive");
    try {
      return newHttpConnection(new URL("http", host, port, "/"));
    } catch (MalformedURLException e) {
      throw new IllegalArgumentException(e);
    }
  }

  /** Returns a {@link Connection} from a "host:port" formatted {@link String}. */
  public static Connection parse(String address) {
    Matcher m = ADDRESS_PATTERN.matcher(address);
    Preconditions.checkArgument(m.matches(),
        "serverAddress does not match host:port format");
    return newHttpConnection(m.group(1), Integer.parseInt(m.group(2)));
  }

  @Override
  public Reader sendRequest(Gson gson, Protocol.Request request) throws IOException {
    String json = gson.toJson(request);

    HttpURLConnection connection = (HttpURLConnection) url.openConnection();
    connection.setRequestMethod("POST");
    connection.setRequestProperty("Content-Encoding",
        "application/json; charset=" + DEFAULT_CHARSET);
    connection.setRequestProperty("Content-Length", "" + json.length());
    connection.setDoOutput(true);
    connection.setDoInput(true);
    connection.setInstanceFollowRedirects(false);
    connection.setUseCaches(false);

    try (OutputStream stream = connection.getOutputStream();
        OutputStreamWriter writer = new OutputStreamWriter(stream, DEFAULT_CHARSET)) {
      writer.write(json);
    }

    return new InputStreamReader(connection.getInputStream(), DEFAULT_CHARSET);
  }
}
