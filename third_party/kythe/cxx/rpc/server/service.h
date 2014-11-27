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

// A service represents a set of functions that are exposed together and form a
// coherent unit of functionality.
//
// A service implementer will typically derive from ServiceBase to get all of
// the basic functionality in that class for free, the implementation of the
// name() and ResolveMethod() methods. A typical service implementation will
// pass the service name to the ServiceBase constructor and register all of the
// callable methods in its constructor like so:
//
// class SampleService : public ServiceBase<SampleService> {
//   public:
//     SampleService() : ServiceBase("SampleService") {
//       RegisterMethod("AwesomeMethod", &SampleService::AwesomeMethod);
//     }
//
//   private:
//     bool AwesomeMethod(const MyInput& input, OutputSink* output);
// };
//
// It is recommended that the methods remain private so they can only be
// accessed by using the Service interface, or even better, the EndPoint
// interface that aggregates all of the exposed services.
#ifndef KYTHE_RPC_CXX_SERVER_SERVICE_H_
#define KYTHE_RPC_CXX_SERVER_SERVICE_H_

#include <functional>
#include <string>
#include <utility>
#include <map>
#include <memory>
#include <type_traits>

#include "glog/logging.h"
#include "third_party/kythe/cxx/rpc/server/method.h"

namespace krpc {

// Defines the basic interface for a service class.
class Service {
 public:
  virtual ~Service() {}

  // Returns the name of the service.
  virtual const std::string& name() const = 0;

  virtual const std::map<std::string, std::unique_ptr<Method>>& methods()
      const = 0;

  // Resolves the |name| to a method objects.
  virtual const Method* ResolveMethod(const std::string& name) const = 0;

 protected:
  Service() = default;
};

// Implements the basic functionality of a service for |TDerived| including the
// registration of methods.
// The typical use calls for deriving from this class and calling RegisterMethod
// during the construction of the service class being exposed. If that pattern
// is followed then after construction the methods name() and ResolveMethod()
// are thread-safe as they don't modify the instance structures. The Service
// instance owns the lifetime of all of the Method objects created when
// registering them.
// Example of use:
//   class MyService : public krpc::ServiceBase<MyService> {
//     public:
//      MyService() {
//        RegisterMethod("MyMethod", &MyService::MyMethod);
//      }
//   ...
//   }
template <typename TDerived>
class ServiceBase : public Service {
 public:
  // No copying
  ServiceBase(const Service&) = delete;
  ServiceBase& operator=(const Service&) = delete;

  // Returns the name of the service.
  const std::string& name() const override { return name_; }

  // Returns the methods registered with this service keyed by their names.
  const std::map<std::string, std::unique_ptr<Method>>& methods()
      const override {
    return methods_;
  }

  // Resolves |name| to the method. Returns null if no method with that name is
  // found.
  const Method* ResolveMethod(const std::string& name) const override {
    auto it = methods_.find(name);
    if (it == end(methods_)) {
      LOG(ERROR) << "Can't find method \"" << name << "\"";
      return nullptr;
    }
    return it->second.get();
  }

 protected:
  explicit ServiceBase(const std::string& name) : name_(name) {}

  // Registers a member pointer to the method to run under |name|.
  template <typename TMethod>
  void RegisterMethod(const std::string& name, const TMethod& method) {
    static_assert(std::is_base_of<ServiceBase<TDerived>, TDerived>::value,
                  "TDerived must derive from ServiceBase itself");

    methods_[name] = MakeMethod(static_cast<TDerived*>(this), method);
  }

 private:
  // The methods registered with this service keyed by their names.
  std::map<std::string, std::unique_ptr<Method>> methods_;

  // This service's name.
  const std::string name_;
};
}

#endif  // KYTHE_RPC_CXX_SERVER_SERVICE_H_
