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
