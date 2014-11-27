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

#include <memory>
#include <string>

#include "gflags/gflags.h"
#include "glog/logging.h"
#include "third_party/kythe/cxx/rpc/server/http_transport.h"
#include "third_party/kythe/cxx/rpc/server/output_proto_stream.h"
#include "third_party/kythe/cxx/rpc/server/service.h"
#include "third_party/kythe/cxx/rpc/server/status.h"
#include "third_party/kythe/cxx/rpc/server/test_data/example.pb.h"

DEFINE_int32(port, 8080, "the port where to listen for connections");

using krpc::HttpTransport;
using krpc::OutputProtoStream;
using krpc::ServiceBase;
using krpc::Status;
using kythe::example::PingRequest;
using kythe::example::PingResponse;
using std::unique_ptr;

class PingService : public ServiceBase<PingService> {
 public:
  PingService() : ServiceBase("PingService") {
    RegisterMethod("Ping", &PingService::Ping);
  }

 private:
  Status Ping(const PingRequest& request,
              OutputProtoStream<PingResponse>* output) const {
    LOG(INFO) << "Ping: " << request.ping();

    std::string pong = request.ping();
    if (request.has_prefix()) {
      LOG(INFO) << "Prefix: " << request.prefix();
      if (request.prefix() == "invalid") {
        LOG(INFO) << "Returning fake error";
        return Status::Error("Invalid prefix: " + request.prefix());
      }
      pong += request.prefix();
    }

    PingResponse response;
    response.set_pong(pong);

    output->Write(response);
    return Status::Ok();
  }
};

void ServeData(int port) {
  HttpTransport http_endpoint;
  http_endpoint.end_point().RegisterService(
      unique_ptr<PingService>(new PingService));

  LOG(INFO) << "Waiting for requests at " << port;
  http_endpoint.StartServing(port);
}

int main(int argc, char* argv[]) {
  google::ParseCommandLineFlags(&argc, &argv, true);
  google::InitGoogleLogging(argv[0]);

  ServeData(FLAGS_port);
}
