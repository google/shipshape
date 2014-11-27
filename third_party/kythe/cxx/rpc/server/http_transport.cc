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

#include "third_party/kythe/cxx/rpc/server/http_transport.h"

#include <algorithm>
#include <cstdint>
#include <cstdlib>
#include <string>
#include <vector>

#include "CivetServer.h"

#include "glog/logging.h"

#include "third_party/kythe/cxx/rpc/server/memory_output.h"

using std::vector;

namespace krpc {

namespace {

// Parses a url of the form "/Service/Method".
bool SplitServiceMethod(const std::string& url, std::string* service_name,
                        std::string* method_name) {
  if (url.empty()) {
    LOG(ERROR) << "Empty service/method URL";
    return false;
  }
  auto search_begin = begin(url);
  if (*search_begin == '/') {
    ++search_begin;  // Avoid the / at the beginning
  }

  auto split = std::find(search_begin, end(url), '/');
  if (split == end(url) ||
      std::find(std::next(split), end(url), '/') != end(url)) {
    LOG(ERROR) << "Invalid service/method URL \"" << url << "\"";
    return false;
  }

  *service_name = std::string(search_begin, split);
  *method_name = std::string(std::next(split), end(url));
  return true;
}

void SendError(::mg_connection* conn, int error, const std::string& message) {
  ::mg_printf(conn, "HTTP/1.1 %d Error (%s)\r\n\r\n%s", error, message.c_str(),
              message.c_str());
}

bool WriteResponse(::mg_connection* conn, const vector<uint8_t>& contents) {
  LOG(INFO) << "Sending data of size: " << contents.size();
  ::mg_printf(conn,
              "HTTP/1.1 200 OK (KRPC) \r\n"
              "Content-Type: application/octet-stream\r\n"
              "Content-Length: %zu\r\n"
              "\r\n",
              contents.size());
  size_t written = ::mg_write(conn, reinterpret_cast<const void*>(contents.data()),
                           contents.size());
  return written == contents.size();
}

class RpcHandler : public CivetHandler {
 public:
  // Initializes the RpcHandler with the |end_point| that contains the services
  // exposed. |end_point| must not be null.
  explicit RpcHandler(const EndPoint* end_point)
      : end_point_(CHECK_NOTNULL(end_point)) {}

  bool handlePost(CivetServer* server, struct ::mg_connection* conn) override {
    auto* info = ::mg_get_request_info(conn);
    std::string service_name, method_name;
    if (!SplitServiceMethod(info->uri, &service_name, &method_name)) {
      SendError(conn, 400, "Invalid uri");
      return true;
    }
    LOG(INFO) << "Service: " << service_name << " Method: " << method_name;

    auto* method = end_point_->ResolveMethod(service_name, method_name);
    if (method == nullptr) {
      SendError(conn, 404, "Unknown method");
      return true;
    }

    const auto& input_descriptor = method->InputFormat();

    auto* content_length_header = ::mg_get_header(conn, "Content-Length");
    long content_length = 0;
    if (content_length_header != nullptr) {
      content_length = std::strtol(content_length_header, nullptr, 10);
      if (errno == ERANGE) {
        LOG(ERROR) << "Bad content length: " << content_length;
        SendError(conn, 400, "Bad request");
        return true;
      }
      LOG(INFO) << "Receiving data size " << content_length;
    }

    if (content_length == 0 && !input_descriptor.format.empty()) {
      LOG(ERROR) << "No data passed as input and method " << info->uri
                 << " requires it";
      SendError(conn, 400, "No input data");
      return true;
    }

    std::string content(content_length, 0);

    if (::mg_read(conn, &content[0], content_length) != content_length) {
      SendError(conn, 500, "Error processing request");
      return true;
    }

    InMemoryOutputSink sink;
    auto status = method->Call(content, &sink);
    if (!status.ok()) {
      SendError(conn, 500, status.error_detail().empty()
                               ? "Error processing method"
                               : status.error_detail());
      return true;
    }

    bool result = WriteResponse(conn, sink.data());
    LOG_IF(WARNING, !result) << "Failed to send all of the data";
    return true;
  }

 private:
  const EndPoint* const end_point_;
};

}  // namespace

bool HttpTransport::StartServing(int port) const {
  auto port_string = std::to_string(port);
  const char* options[] = {"listening_ports", port_string.c_str(), nullptr};

  {
    CivetServer server(options);
    RpcHandler handler(&this->end_point());

    server.addHandler("/", &handler);

    while (true) {
      sleep(10);
    }
  }

  return false;  // Unreachable.
}
}  // namespace krpc
