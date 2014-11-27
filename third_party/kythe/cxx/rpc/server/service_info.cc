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

#include "third_party/kythe/cxx/rpc/server/service_info.h"

#include <memory>
#include <utility>

#include "third_party/kythe/cxx/rpc/server/endpoint.h"
#include "third_party/kythe/cxx/rpc/server/json_helper.h"
#include "third_party/kythe/cxx/rpc/server/output_json_stream.h"

using json::JSONObjectSerializer;
using json::SerializeArrayToJSON;
using std::pair;
using std::unique_ptr;

namespace krpc {

namespace {

void PrintFormatDescriptor(const FormatDescriptor& descriptor,
                           JSONObjectSerializer* serializer) {
  serializer->WriteProperty("format", descriptor.format);
  if (!descriptor.label.empty()) {
    serializer->WriteProperty("label", descriptor.label);
  }
}

void PrintMethodToJSON(const pair<const std::string, unique_ptr<Method>>& entry,
                       JSONObjectSerializer* serializer) {
  serializer->WriteProperty("name", entry.first);

  const auto& input_descriptor = entry.second->InputFormat();
  const auto& output_descriptor = entry.second->OutputFormat();

  if (!input_descriptor.format.empty()) {
    serializer->WriteObject("input", input_descriptor, PrintFormatDescriptor);
  }

  if (!output_descriptor.format.empty()) {
    serializer->WriteObject("output", output_descriptor, PrintFormatDescriptor);
  }
}

void PrintServiceToJSON(
    const pair<const std::string, unique_ptr<Service>>& entry,
    JSONObjectSerializer* serializer) {
  serializer->WriteProperty("name", entry.first);
  serializer->WriteArray("methods", entry.second->methods(), PrintMethodToJSON);
}

}  // namespace

Status ServiceInfo::List(OutputJSONStream* output) const {
  output->Write(
      SerializeArrayToJSON(end_point_->services(), PrintServiceToJSON));

  return Status::Ok();
}
}  // namespace krpc
