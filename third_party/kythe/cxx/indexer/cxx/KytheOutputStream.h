#ifndef KYTHE_CXX_INDEXER_CXX_KYTHE_OUTPUT_STREAM_H_
#define KYTHE_CXX_INDEXER_CXX_KYTHE_OUTPUT_STREAM_H_

#include <memory>
#include <vector>

#include "google/protobuf/io/coded_stream.h"
#include "google/protobuf/io/zero_copy_stream_impl.h"

#include "gflags/gflags.h"
#include "third_party/kythe/proto/storage.pb.h"

DECLARE_bool(flush_after_each_entry);

namespace kythe {

// Interface for receiving Kythe `Entity` instances.
// TODO(jdennett): This interface should be recast in terms of emitting facts
//                 instead of directly constructing `Entity` protos.
//                 (Also see KytheGraphObserver.cc.)
class KytheOutputStream {
 public:
  virtual void Emit(const kythe::proto::Entry &entry) = 0;
};

// A `KytheOutputStream` that records `Entry` instances to a
// `FileOutputStream`.
class FileOutputStream : public KytheOutputStream {
 public:
  FileOutputStream(google::protobuf::io::FileOutputStream *stream)
      : stream_(stream) {}

  void Emit(const kythe::proto::Entry &entry) override {
    {
      google::protobuf::io::CodedOutputStream coded_stream(stream_);
      coded_stream.WriteVarint32(entry.ByteSize());
      entry.SerializeToCodedStream(&coded_stream);
    }
    if (FLAGS_flush_after_each_entry) {
      stream_->Flush();
    }
  }

 private:
  google::protobuf::io::FileOutputStream *stream_;
};

}  // namespace kythe

#endif  // KYTHE_CXX_INDEXER_CXX_KYTHE_OUTPUT_STREAM_H_
