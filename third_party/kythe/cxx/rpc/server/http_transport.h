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

// The HttpTransport class allows KRPC calls to be made through an HTTP
// channel.
// A typical use of this class is:
//   void ServeData(int port) {
//     HttpTransport http_endpoint;
//      http_endpoint.end_point().RegisterService(
//        unique_ptr<PingService>(new PingService));
//      http_endpoint.StartServing(port);
//   }

#ifndef KYTHE_RPC_CXX_SERVER_HTTP_TRANSPORT_H_
#define KYTHE_RPC_CXX_SERVER_HTTP_TRANSPORT_H_

#include "third_party/kythe/cxx/rpc/server/endpoint.h"

namespace krpc {

// The HttpTransport class implements the HTTP bindings for the KRPC system and
// exposes the contained EndPoint. To expose a service then all an application
// has to do is register it with the HttpTransport's EndPoint instance. More
// than one HttpTransport instance can be used in a single process as long as
// they are serving in different ports. Because an HttpTransport owns a thread
// pool internally multiple requests might be served in parallel therefore the
// implementation of the services registered with it must be thread safe.
class HttpTransport {
 public:
  // No copy
  HttpTransport(const HttpTransport&) = delete;
  HttpTransport& operator=(const HttpTransport&) = delete;

  // A default-initialized HttpTransport has a default-initialized EndPoint,
  // only the /ServiceInfo service is registered.
  HttpTransport() = default;

  // Starts the HTTP server on the given |port| and will not return until the
  // service stops.
  // TODO: Have a way to stop the server, currently this method will
  // never return.
  bool StartServing(int port) const;

  // Give access to the end point to register services.
  EndPoint& end_point() { return end_point_; }
  const EndPoint& end_point() const { return end_point_; }

 private:
  // The end point exposed by this instance.
  EndPoint end_point_;
};
}  // namespace krpc

#endif  // KYTHE_RPC_CXX_SERVER_HTTP_TRANSPORT_H_
