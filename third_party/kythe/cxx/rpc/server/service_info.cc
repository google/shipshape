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
