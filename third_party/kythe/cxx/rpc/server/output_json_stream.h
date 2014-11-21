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
