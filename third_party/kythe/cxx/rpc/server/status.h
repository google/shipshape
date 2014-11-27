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

// A status instance allows kprc methods to return both the status of the
// called, whether it succeeded or not, as well as an error message (in case of
// erro) to be sent back to the client.
// There are static helper functions provided to make the use of the class even
// simpler so a typicall use would be:
//   Status MyMethod(const MyRequest& request, OutputProtoStream<MyResponse>&
//   output) {
//     if (!Validate(request)) {
//       return Status::Error("Invalid request");
//     }
//     ...
//     return Status::Ok();
//   }
//
// When checking the status the typical usage would be:
//   auto status = end_point->InvokeMethod(...);
//   if (!status.ok()) {
//     SendResponse(status.error_detail());
//   }
#ifndef KYTHE_RPC_CXX_SERVER_STATUS_H_
#define KYTHE_RPC_CXX_SERVER_STATUS_H_

#include <string>
#include <utility>

namespace krpc {

// Contains both the results status of the call (success or failure) as well as
// an optional error detail string for the error.
class Status {
 public:
  // Use this constructor if no error message is to be specified.
  explicit Status(bool ok = true) : ok_(ok) {}

  // Use this constructor if an error message is specified, this constructor
  // implies that there was an error.
  explicit Status(std::string error_detail)
      : ok_(false), error_detail_(std::move(error_detail)) {}

  // Returns whether the operation succeeded or not.
  bool ok() const { return ok_; }

  const std::string& error_detail() const { return error_detail_; }

  // Creates a status in the successful state.
  static Status Ok() { return {}; }

  // Creates a status in the error state but no error is specified
  static Status Error() { return Status{false}; }

  // Creates a status in the error state with |message| as the error detail.
  static Status Error(std::string message) {
    return Status{std::move(message)};
  }

 private:
  bool ok_ = true;
  std::string error_detail_;
};
}  // namespace krpc

#endif  // KYTHE_RPC_CXX_SERVER_STATUS_H_
