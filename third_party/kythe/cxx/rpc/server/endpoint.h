#ifndef KYTHE_RPC_CXX_ENDPOINT_H_
#define KYTHE_RPC_CXX_ENDPOINT_H_

/// \file
/// \brief Defines the class krpc::EndPoint.

#include <map>
#include <memory>
#include <string>

#include "third_party/kythe/cxx/rpc/server/method.h"
#include "third_party/kythe/cxx/rpc/server/service.h"

namespace krpc {

/// \brief Defines an endpoint as a container for services.
///
/// An endpoint is a set of services exposed from an application.  Typically
/// only one EndPoint instance per application will exist, containing all of
/// the services offered by that application.
///
/// A typical use of this class would be:
/// \code
///   EndPoint end_point;
///   end_point.RegisterService(unique_ptr<Service1>(new Service1));
///   ...
///   end_point.RegisterService(unique_ptr<ServiceN>(new Service2));
/// \endcode
///
/// Typically all of the services will be registered on startup and then
/// the endpoint will be frozen at that point. All of the query methods
/// are thread safe at that point. The EndPoint instance owns the lifetime
/// of all of the services registered with it.
class EndPoint {
 public:
  /// \brief Creates an EndPoint with only have the ServiceInfo service
  /// registered.
  EndPoint();

  /// \brief Registers a new service with this EndPoint.
  ///
  /// Registration will fail if a service with the same name
  /// is already registered.
  ///
  /// \return true if registration was successful, false otherwise.
  bool RegisterService(std::unique_ptr<Service> service);

  /// \brief Resolves \p name to the service it represents.
  ///
  /// Ownership of the Service is not transferred.
  ///
  /// \return The service with the given \p name if such a service is
  /// registered with this EndPoint, and null otherwise.
  const Service* ResolveService(const std::string& name) const;

  /// \brief Resolves \p service_name and \p method_name to a method.
  ///
  /// This wraps a call to ResolveService followed by a call to
  /// Service::ResolveMethod (the latter being guarded by a check that
  /// the former succeeded).
  ///
  /// Ownership of the Method is not transferred.
  ///
  /// \return The matching service if one exists, and null if either
  /// if either \p service_name or \p method_name could not be resolved.
  const Method* ResolveMethod(const std::string& service_name,
                              const std::string& method_name) const;

  /// \brief Invokes the method denoted by \p service_name and \p method_name.
  ///
  /// \return The method's return value if the method was found, and an error
  /// if it was not.
  Status InvokeMethod(const std::string& service_name,
                      const std::string& method_name, const std::string& input,
                      OutputSink* output) const;

  /// \brief Retrieves the services registered with this endpoint.
  /// \return The services registered with this endpoint, keyed by their names.
  const std::map<std::string, std::unique_ptr<Service>>& services() const {
    return services_;
  }

 private:
  EndPoint(const EndPoint&) = delete;
  EndPoint& operator=(const EndPoint&) = delete;

  /// \brief The services registered with this endpoint keyed by their names.
  std::map<std::string, std::unique_ptr<Service>> services_;
};
}

#endif  // KYTHE_RPC_CXX_ENDPOINT_H_
