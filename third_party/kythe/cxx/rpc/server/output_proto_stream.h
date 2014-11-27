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

#ifndef KYTHE_RPC_CXX_SERVER_OUTPUT_PROTO_STREAM_H_
#define KYTHE_RPC_CXX_SERVER_OUTPUT_PROTO_STREAM_H_

/// \file
/// \brief Defines the class template krpc::OutputProtoStream.

#include <string>
#include <type_traits>

#include "third_party/kythe/cxx/rpc/server/output_sink.h"
#include "google/protobuf/message.h"

namespace krpc {

/// \brief A typed stream of protocol buffers.
template <typename TOutput>
class OutputProtoStream {
  static_assert(std::is_base_of<google::protobuf::Message, TOutput>::value,
                "TOutput must be a proto message type");

 public:
  /// \brief Creates a proto stream writing to a given \p sink.
  ///
  /// \p sink must be non-null and must remain valid so long as this stream
  /// is in use.
  explicit OutputProtoStream(OutputSink* sink) : sink_(sink) {}

  /// \brief Writes \p message to this stream's output sink.
  void Write(const TOutput& message) { sink_->WriteMessage(message); }

 private:
  // No copying.
  OutputProtoStream(const OutputProtoStream&) = delete;
  OutputProtoStream& operator=(const OutputProtoStream&) = delete;

  /// \brief The output sink to which to write.
  OutputSink* const sink_;
};
}  // namespace krpc

#endif  // KYTHE_RPC_CXX_SERVER_OUTPUT_PROTO_STREAM_H_
