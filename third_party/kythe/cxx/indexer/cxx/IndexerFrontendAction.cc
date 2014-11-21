// This file uses the Clang style conventions.

#include "IndexerFrontendAction.h"

#include <memory>
#include <string>

#include "clang/Frontend/FrontendAction.h"
#include "clang/Tooling/Tooling.h"

#include "llvm/ADT/Twine.h"

namespace kythe {

bool RunToolOnCode(std::unique_ptr<clang::FrontendAction> tool_action,
                   llvm::Twine code, const std::string &filename) {
  if (tool_action == nullptr)
    return false;
  return clang::tooling::runToolOnCode(tool_action.release(), code, filename);
}

} // namespace kythe
