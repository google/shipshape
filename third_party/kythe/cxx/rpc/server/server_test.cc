#include "third_party/kythe/cxx/rpc/server/service.h"

#include <memory>
#include <string>
#include <utility>

#include "glog/logging.h"
#include "gtest/gtest.h"
#include "third_party/kythe/cxx/rpc/server/endpoint.h"
#include "third_party/kythe/cxx/rpc/server/memory_output.h"
#include "third_party/kythe/cxx/rpc/server/output_proto_stream.h"
#include "third_party/kythe/cxx/rpc/server/test_data/test.pb.h"

using krpc::EndPoint;
using krpc::OutputSink;
using krpc::Service;
using krpc::ServiceBase;
using kythe::test::MyMessage;
using kythe::test::MyResponse;
using std::unique_ptr;

namespace krpc {
namespace {

class MyService : public ServiceBase<MyService> {
 public:
  MyService() : ServiceBase("MyService") {
    RegisterMethod("MyMethod", &MyService::MyMethod);
  }

  int counter() const { return counter_; }
  int last_age() const { return last_age_; }
  std::string last_name() const { return last_name_; }

 private:
  Status MyMethod(const MyMessage& input,
                  OutputProtoStream<MyResponse>* output) {
    if (input.age() == 42) {
      // Simulate an error
      return Status::Error("Invalid age");
    }

    ++counter_;
    last_age_ = input.age();
    last_name_ = input.name();

    return Status::Ok();
  }

  int counter_ = 0;
  int last_age_ = 0;
  std::string last_name_;
};

class MyServiceMultipleOutput : public ServiceBase<MyServiceMultipleOutput> {
 public:
  MyServiceMultipleOutput() : ServiceBase("MyServiceMultipleOutput") {
    RegisterMethod("MyMethod", &MyServiceMultipleOutput::MyMethod);
  }

 private:
  Status MyMethod(const MyMessage& input,
                  OutputProtoStream<MyResponse>* output) const {
    for (int i = 0; i < input.age(); ++i) {
      MyResponse response;
      response.set_value("ok");
      output->Write(response);
    }

    return Status::Ok();
  }
};

class MyServiceJSON : public ServiceBase<MyServiceJSON> {
 public:
  MyServiceJSON() : ServiceBase("MyServiceJSON") {
    RegisterMethod("MyMethod", &MyServiceJSON::MyMethod);
  }

 private:
  Status MyMethod(OutputJSONStream* output) {
    output->Write("Hello, there");
    output->Write("Something, something");
    return Status::Ok();
  }
};

TEST(ServiceTests, MethodRegistrationAndLookupTests) {
  MyService s;

  ASSERT_EQ("MyService", s.name());
  ASSERT_EQ(1, s.methods().size());
  ASSERT_NE(nullptr, s.ResolveMethod("MyMethod"));
  ASSERT_EQ(nullptr, s.ResolveMethod("NotExisting"));
}

TEST(ServiceTests, MethodCallingTests) {
  MyService s;

  auto* method = s.ResolveMethod("MyMethod");
  ASSERT_NE(nullptr, method);

  MyMessage msg;
  msg.set_name("hello");
  msg.set_age(2);

  std::string serialized;
  msg.SerializeToString(&serialized);

  InMemoryOutputSink sink;

  ASSERT_TRUE(method->Call(serialized, &sink).ok());

  ASSERT_EQ("hello", s.last_name());
  ASSERT_EQ(2, s.last_age());
}

TEST(ServiceTests, MethodCallWithErrorTests) {
  MyService s;

  auto* method = s.ResolveMethod("MyMethod");
  ASSERT_NE(nullptr, method);

  MyMessage msg;
  msg.set_name("hello");
  msg.set_age(42);

  std::string serialized;
  msg.SerializeToString(&serialized);

  auto status = method->Call(serialized, nullptr);
  ASSERT_FALSE(status.ok());
  ASSERT_EQ("Invalid age", status.error_detail());
}

TEST(ServiceTests, MethodCallJSONOutputTests) {
  MyServiceJSON s;

  auto* method = s.ResolveMethod("MyMethod");

  InMemoryOutputSink sink;

  ASSERT_TRUE(method->Call("hello", &sink).ok());
}

TEST(EndPointTests, RegisteringServiceTests) {
  auto s = unique_ptr<MyService>(new MyService);
  EndPoint e;
  e.RegisterService(std::move(s));

  auto* service = e.ResolveService("MyService");
  ASSERT_NE(nullptr, service);

  auto* method = e.ResolveMethod("MyService", "MyMethod");
  ASSERT_NE(nullptr, method);
}

TEST(EndPointTests, InvokeMethodTests) {
  EndPoint e;
  ASSERT_TRUE(e.RegisterService(unique_ptr<MyService>(new MyService)));
  EXPECT_FALSE(e.RegisterService(unique_ptr<MyService>(new MyService)));

  auto* service = static_cast<const MyService*>(e.ResolveService("MyService"));

  MyMessage msg;
  msg.set_name("hello");
  msg.set_age(2);

  std::string serialized;
  msg.SerializeToString(&serialized);

  InMemoryOutputSink sink;

  ASSERT_TRUE(e.InvokeMethod("MyService", "MyMethod", serialized, &sink).ok());

  EXPECT_EQ("hello", service->last_name());
  EXPECT_EQ(2, service->last_age());
}

TEST(EndPointTests, ServiceInfoTests) {
  EndPoint e;
  auto* service_info = e.ResolveService("ServiceInfo");
  ASSERT_NE(nullptr, service_info);

  EXPECT_TRUE(e.RegisterService(unique_ptr<MyService>(new MyService)));

  InMemoryOutputSink sink;
  ASSERT_TRUE(e.InvokeMethod("ServiceInfo", "List", "", &sink).ok());

  const auto& data = sink.data();
  std::string json(begin(data), end(data));

  LOG(ERROR) << json;
}

}  // namespace
}  // namespace krpc

int main(int argc, char** argv) {
  google::InitGoogleLogging(argv[0]);
  ::testing::InitGoogleTest(&argc, argv);
  return RUN_ALL_TESTS();
}
