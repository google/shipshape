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

#ifndef KYTHE_RPC_CXX_SERVICE_INFO_H_
#define KYTHE_RPC_CXX_SERVICE_INFO_H_

#include "third_party/kythe/cxx/rpc/server/output_json_stream.h"
#include "third_party/kythe/cxx/rpc/server/service.h"
#include "third_party/kythe/cxx/rpc/server/status.h"

namespace krpc {
class OutputSink;
class EndPoint;

// This class implements the /ServiceInfo/List service required by the KRPC spec
// which returns the list of services registered with the EndPoint encoded in
// JSON.
class ServiceInfo : public ServiceBase<ServiceInfo> {
 public:
  explicit ServiceInfo(const EndPoint* end_point)
      : ServiceBase("ServiceInfo"), end_point_(end_point) {
    RegisterMethod("List", &ServiceInfo::List);
  }

 private:
  // Implements the /ServiceInfo/List method, as defined in
  // //kythe/rpc/krpc-spec.txt, by encoding all of the services and their
  // methods as a JSON string and writing it to |output|.
  Status List(OutputJSONStream* output) const;

  const EndPoint* const end_point_;
};
}  // namespace krpc

#endif  // KYTHE_RPC_CXX_SERVICE_INFO_H_
