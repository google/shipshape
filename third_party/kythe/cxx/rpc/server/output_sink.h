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

#ifndef KYTHE_RPC_CXX_SERVER_OUTPUT_SINK_H_
#define KYTHE_RPC_CXX_SERVER_OUTPUT_SINK_H_

#include <string>

#include "google/protobuf/message.h"

namespace krpc {

// An output sink is the object that will receive all of the output of a KRPC
// method. This class represents arround what the KRPC spec calls a "delimited
// stream", see //kythe/rpc/krpc-spec.txt for more details on what a "delimited
// stream" is.
class OutputSink {
 public:
  virtual ~OutputSink() {}

  // Writes |message| to the output sink.
  virtual void WriteMessage(const google::protobuf::Message& message) = 0;

  // Writes |data| to the output sink. In this context |data| is a string that
  // contains an entity serialized as JSON.
  virtual void WriteJSONString(const std::string& data) = 0;
};
}

#endif  // KYTHE_RPC_CXX_SERVER_OUTPUT_SINK_H_
