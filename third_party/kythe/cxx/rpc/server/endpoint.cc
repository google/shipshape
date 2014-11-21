#include "third_party/kythe/cxx/rpc/server/endpoint.h"

#include "glog/logging.h"

#include "third_party/kythe/cxx/rpc/server/service_info.h"

using std::unique_ptr;

namespace krpc {

// Every EndPoint must have a ServiceInfo service that can serve the metadata
// about the services registered in that EndPoint.
EndPoint::EndPoint() {
  this->RegisterService(unique_ptr<Service>(new ServiceInfo(this)));
}

bool EndPoint::RegisterService(unique_ptr<Service> service) {
  if (services_.find(service->name()) != end(services_)) {
    LOG(ERROR) << "The service \"" << service->name()
               << "\" was already registered";
    return false;
  }

  services_[service->name()] = std::move(service);
  return true;
}

const Service* EndPoint::ResolveService(const std::string& name) const {
  auto it = services_.find(name);
  if (it == end(services_)) {
    LOG(ERROR) << "Can't find service \"" << name << "\"";
    return nullptr;
  }
  return it->second.get();
}

const Method* EndPoint::ResolveMethod(const std::string& service_name,
                                      const std::string& method_name) const {
  auto* service = this->ResolveService(service_name);
  if (service == nullptr) {
    return nullptr;
  }

  return service->ResolveMethod(method_name);
}

Status EndPoint::InvokeMethod(const std::string& service_name,
                              const std::string& method_name,
                              const std::string& input,
                              OutputSink* output) const {
  auto* method = this->ResolveMethod(service_name, method_name);
  if (method == nullptr) {
    return Status::Error("Unknown service or method");
  }

  return method->Call(input, output);
}
}  // namespace krpc
