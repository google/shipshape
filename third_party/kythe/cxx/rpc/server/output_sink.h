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
