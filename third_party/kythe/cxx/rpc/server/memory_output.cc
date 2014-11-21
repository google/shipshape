#include "third_party/kythe/cxx/rpc/server/memory_output.h"

#include "google/protobuf/io/coded_stream.h"

using google::protobuf::Message;
using google::protobuf::io::CodedOutputStream;
using std::uint32_t;
using std::uint8_t;
using std::vector;

namespace krpc {

void InMemoryOutputSink::WriteMessage(const Message& message) {
  std::string serialized;
  message.SerializeToString(&serialized);

  const size_t offset = data_.size();
  data_.resize(offset + CodedOutputStream::VarintSize32(serialized.size()) +
               serialized.size());

  uint8_t* next = data_.data() + offset;
  next = CodedOutputStream::WriteVarint32ToArray(serialized.size(), next);
  next = CodedOutputStream::WriteRawToArray(serialized.data(),
                                            serialized.size(), next);
}

void InMemoryOutputSink::WriteJSONString(const std::string& json) {
  data_.insert(end(data_), begin(json), end(json));
  data_.push_back('\n');
}
}  // namespace krpc
