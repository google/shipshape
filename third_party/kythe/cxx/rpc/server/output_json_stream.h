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

#ifndef KYTHE_RPC_CXX_SERVER_OUTPUT_JSON_STREAM_H_
#define KYTHE_RPC_CXX_SERVER_OUTPUT_JSON_STREAM_H_

#include <string>

#include "third_party/kythe/cxx/rpc/server/output_sink.h"

namespace krpc {

// Represents an output stream of JSON encoded objects.
class OutputJSONStream {
 public:
  // No copying
  OutputJSONStream(const OutputJSONStream&) = delete;
  OutputJSONStream& operator=(const OutputJSONStream&) = delete;

  // Initializes the instance with the sink to use, |sink| must not be null.
  explicit OutputJSONStream(OutputSink* sink) : sink_(sink) {}

  // Writes the JSON encoded object to the output sink.
  void Write(const std::string& json) { sink_->WriteJSONString(json); }

 private:
  // The output sink in which to write.
  OutputSink* sink_;
};
}  // namespace krpc

#endif  // KYTHE_RPC_CXX_SERVER_OUTPUT_JSON_STREAM_H_
