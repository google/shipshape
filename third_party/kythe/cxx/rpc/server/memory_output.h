// The InMemoryOutputSink class provides an implementation of the OutputSink
// interface that stores the output in memory.
#ifndef KYTHE_RPC_CXX_SERVER_MEMORYOUTPUT_H_
#define KYTHE_RPC_CXX_SERVER_MEMORYOUTPUT_H_

#include <cstdint>
#include <vector>

#include "third_party/kythe/cxx/rpc/server/output_sink.h"
#include "google/protobuf/message.h"

namespace krpc {

// In-memory implementation of the OutputSink interface that stores the protobuf
// messages and serialized JSON strings as a delimited stream as specified in
// the krpc spec. This class is thread compatible.
class InMemoryOutputSink : public OutputSink {
 public:
  void WriteMessage(const google::protobuf::Message& message) override;

  void WriteJSONString(const std::string& json) override;

  // Returns the data stored in the sink.
  const std::vector<std::uint8_t>& data() const { return data_; }

 private:
  // The delimited stream of data written to this sink.
  std::vector<std::uint8_t> data_;
};
}

#endif  // KYTHE_RPC_CXX_SERVER_MEMORYOUTPUT_H_
