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

// This file uses the Clang style conventions.

#ifndef KYTHE_CXX_INDEXER_CXX_PP_CALLBACKS_H_
#define KYTHE_CXX_INDEXER_CXX_PP_CALLBACKS_H_

#include "clang/Basic/SourceManager.h"
#include "clang/Lex/PPCallbacks.h"

namespace clang {
class Preprocessor;
} // namespace clang

namespace kythe {

class GraphObserver;

/// \brief Listener for preprocessor events, handling file tracking and macro
/// use and definition.
class IndexerPPCallbacks : public clang::PPCallbacks {
public:
  IndexerPPCallbacks(const clang::Preprocessor &PP, GraphObserver &GO);
  ~IndexerPPCallbacks() override;

  void FileChanged(clang::SourceLocation Loc,
                   PPCallbacks::FileChangeReason Reason,
                   clang::SrcMgr::CharacteristicKind FileType,
                   clang::FileID PrevFID) override;

  void EndOfMainFile() override;

private:
  /// The `clang::Preprocessor` to which this `IndexerPPCallbacks` is listening.
  const clang::Preprocessor &Preprocessor;
  /// The `GraphObserver` we will use for reporting information.
  GraphObserver &Observer;
};

} // namespace kythe

#endif // KYTHE_CXX_INDEXER_CXX_PP_CALLBACKS_H_