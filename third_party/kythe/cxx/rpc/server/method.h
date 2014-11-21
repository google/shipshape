// A method represents a unit of work that can be called given an input,
// serialized as a string, and that will send its outputs to the given output
// sink.
#ifndef KYTHE_RPC_CXX_SERVER_METHOD_H_
#define KYTHE_RPC_CXX_SERVER_METHOD_H_

#include <memory>
#include <string>
#include <type_traits>

#include "glog/logging.h"
#include "google/protobuf/message.h"
#include "third_party/kythe/cxx/rpc/server/output_json_stream.h"
#include "third_party/kythe/cxx/rpc/server/output_proto_stream.h"
#include "third_party/kythe/cxx/rpc/server/output_sink.h"
#include "third_party/kythe/cxx/rpc/server/status.h"

namespace krpc {

// The format descriptor describes the format accepted either as input or output
// by a method. See the krpc specification (//kythe/rpc/krpc-spec.txt) for a
// description of the /ServiceInfo service.
struct FormatDescriptor {
  // What format is to be used, valid values are the empty string to signify
  // that there's no input (or output), "json" for the case when the
  // input/output is a json serialized object and "proto" when the input/output
  // is a protocol buffer.
  std::string format;

  // Optional label for the format used, if format is "proto" then this member
  // contains the fully qualified name of the proto message to use.
  std::string label;
};

// Defines the basic interface for a Krpc method, a unit of work that can be
// invoked via a serialized input and that will write its outputs to a given
// output sink.
class Method {
 public:
  virtual ~Method() {}

  // Performs the operation represented by this method, passing in the |input|
  // string containing the serialized input (can be a serialized proto or a JSON
  // string) and the OutputSink that will receive all of the output from the
  // method.
  virtual Status Call(const std::string& input, OutputSink* output) const = 0;

  // Returns the format descriptor for the input accepted by this method if any.
  virtual FormatDescriptor InputFormat() const = 0;

  // Returns the format descriptor for the output accepted by this method if
  // any.
  virtual FormatDescriptor OutputFormat() const = 0;

 protected:
  Method() = default;
};

// Implements the Method interface for the case when the input is a proto and
// the output is a stream of protos.
template <typename TFunc, typename TInput, typename TOutput>
class MethodProtoStream : public Method {
  static_assert(std::is_base_of<google::protobuf::Message, TInput>::value,
                "TInput must be a proto message type");
  static_assert(std::is_base_of<google::protobuf::Message, TOutput>::value,
                "TOutput must be a proto message type");

 public:
  // No copying
  MethodProtoStream(const MethodProtoStream&) = delete;
  MethodProtoStream& operator=(const MethodProtoStream&) = delete;

  // Initializes the method with the given functor.
  explicit MethodProtoStream(TFunc func) : func_(func) {}

 private:
  // Calls the wrapped functor first by deserializing the proto in |input| and
  // calling the functor with it.
  Status Call(const std::string& input, OutputSink* output) const override {
    TInput message_input;
    if (!message_input.ParseFromString(input)) {
      return Status::Error("Input message cannot be parsed");
    }
    OutputProtoStream<TOutput> method_output(output);
    return func_(message_input, &method_output);
  }

  FormatDescriptor InputFormat() const override {
    return {"proto", TInput::descriptor()->full_name()};
  }

  FormatDescriptor OutputFormat() const override {
    return {"proto", TOutput::descriptor()->full_name()};
  }

  const TFunc func_;
};

// Implements the Method interface for functions where there's no input and the
// output is JSON.
template <typename TFunc>
class MethodNoInputJSONStream : public Method {
 public:
  MethodNoInputJSONStream(const MethodNoInputJSONStream&) = delete;
  MethodNoInputJSONStream& operator=(const MethodNoInputJSONStream&) = delete;

  explicit MethodNoInputJSONStream(TFunc func) : func_(func) {}

 private:
  // Calls the functor droping the input.
  Status Call(const std::string& input, OutputSink* output) const override {
    LOG_IF(WARNING, !input.empty()) << "Input dropped";
    OutputJSONStream method_output(output);
    return func_(&method_output);
  }

  FormatDescriptor InputFormat() const override { return {}; }

  FormatDescriptor OutputFormat() const override { return {"json"}; }

  const TFunc func_;
};

namespace details {

// Creates a Method based on the given |TInput| and |TOutput| types assuming
// that |TFunc| is a pointer to member function (const or not) for
// |TService|. Assumes that |TInput| and |TOutput| are proto message types.
template <typename TInput, typename TOutput, typename TService, typename TFunc>
std::unique_ptr<Method> MakeProtoInputOutputMethod(TService* service,
                                                   const TFunc& func) {
  static_assert(std::is_base_of<google::protobuf::Message, TInput>::value,
                "The method being wrapped should accept a proto message");
  static_assert(std::is_base_of<google::protobuf::Message, TOutput>::value,
                "The method being wrapped should emit a proto message output");

  auto func_call = [service, func](const TInput& input,
                                   OutputProtoStream<TOutput>* output)
                       -> Status { return (service->*func)(input, output); };

  return std::unique_ptr<Method>(
      new MethodProtoStream<decltype(func_call), TInput, TOutput>(func_call));
}

// Creates a Method instance assuming that |TFunc| is a pointer to member
// function type on |TService|. This template is only to be used when the method
// is a JSON output only method.
template <typename TService, typename TFunc>
std::unique_ptr<Method> MakeJSONOutputOnlyMethod(TService* service,
                                                 const TFunc& func) {
  auto func_call = [service, func](OutputJSONStream* output)
                       -> Status { return (service->*func)(output); };

  return std::unique_ptr<Method>(
      new MethodNoInputJSONStream<decltype(func_call)>(func_call));
}

}  // namespace details

// Creates an initializes a Method instance that wraps a member function that
// accepts a proto and produces a stream of protos as its output. All of the
// output protos are of the same type.
template <typename TService, typename TInput, typename TOutput>
std::unique_ptr<Method> MakeMethod(
    TService* service,
    Status (TService::*func)(const TInput&, OutputProtoStream<TOutput>*)) {
  return details::MakeProtoInputOutputMethod<TInput, TOutput>(service, func);
}

// Overload that allows the creation of a Method instance based on a const
// member function.
template <typename TService, typename TInput, typename TOutput>
std::unique_ptr<Method> MakeMethod(
    TService* service,
    Status (TService::*func)(const TInput&, OutputProtoStream<TOutput>*)
    const) {
  return details::MakeProtoInputOutputMethod<TInput, TOutput>(service, func);
}

// Creates and initializes a Method instance that wraps a member function that
// accepts no input and produces a JSON stream as its output.
template <typename TService>
std::unique_ptr<Method> MakeMethod(
    TService* service, Status (TService::*func)(OutputJSONStream*)) {
  return details::MakeJSONOutputOnlyMethod(service, func);
}

// Creates and initializes a Method instance that wraps a const member function
// that accepts no input and produces a JSON stream as its output.
template <typename TService>
std::unique_ptr<Method> MakeMethod(TService* service,
                                   Status (TService::*func)(OutputJSONStream*)
                                   const) {
  return details::MakeJSONOutputOnlyMethod(service, func);
}

// TODO(ivann): Other variants of the MakeMethod template, accept input and
// returns JSON output and accepts no input and returns proto stream. These are
// not implemented yet because we don't have a service that needs them yet.
}

#endif  // KYTHE_RPC_CXX_SERVER_METHOD_H_
