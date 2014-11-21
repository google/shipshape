/// \file IndexerFrontendAction.h
/// \brief Defines a tool that passes notifications to a `GraphObserver`.

// This file uses the Clang style conventions.

#ifndef KYTHE_CXX_INDEXER_CXX_INDEXER_FRONTEND_ACTION_H_
#define KYTHE_CXX_INDEXER_CXX_INDEXER_FRONTEND_ACTION_H_

#include <limits.h>
#include <stdio.h>
#include <stdlib.h>

#include <memory>
#include <set>
#include <string>
#include <utility>

#include "clang/Frontend/CompilerInstance.h"
#include "clang/Frontend/FrontendAction.h"
#include "clang/Lex/Preprocessor.h"
#include "clang/Tooling/Tooling.h"

#include "llvm/ADT/StringRef.h"
#include "llvm/ADT/Twine.h"

#include "GraphObserver.h"
#include "IndexerASTHooks.h"
#include "IndexerPPCallbacks.h"

namespace kythe {

/// \brief Runs a given tool on a piece of code with a given assumed filename.
/// \returns true on success, false on failure.
bool RunToolOnCode(std::unique_ptr<clang::FrontendAction> tool_action,
                   llvm::Twine code, const std::string &filename);

// A FrontendAction that extracts information about a translation unit both
// from its AST (using an ASTConsumer) and from preprocessing (with a
// PPCallbacks implementation).
//
// TODO(jdennett): Test/implement/document the rest of this.
//
// TODO(jdennett): Consider moving/renaming this to kythe::ExtractIndexAction.
class IndexerFrontendAction : public clang::ASTFrontendAction {
public:
  explicit IndexerFrontendAction(GraphObserver *GO) : Observer(GO) {
    assert(GO != nullptr);
  }

  /// \brief Barrel through even if we don't understand part of a program?
  /// \param I The behavior to use when an unimplemented entity is encountered.
  void setIgnoreUnimplemented(BehaviorOnUnimplemented B) {
    IgnoreUnimplemented = B;
  }

private:
  std::unique_ptr<clang::ASTConsumer>
  CreateASTConsumer(clang::CompilerInstance &CI,
                    llvm::StringRef Filename) override {
    if (Observer) {
      Observer->setSourceManager(&CI.getSourceManager());
      Observer->setLangOptions(&CI.getLangOpts());
      Observer->setPreprocessor(&CI.getPreprocessor());
    }
    return llvm::make_unique<IndexerASTConsumer>(Observer, IgnoreUnimplemented);
  }

  bool BeginSourceFileAction(clang::CompilerInstance &CI,
                             llvm::StringRef Filename) override {
    if (Observer) {
      auto *Callbacks = new IndexerPPCallbacks(CI.getPreprocessor(), *Observer);
      CI.getPreprocessor().addPPCallbacks(Callbacks); // Clang takes ownership
    }
    return true;
  }

  bool usesPreprocessorOnly() const override { return false; }

  /// The `GraphObserver` used for reporting information.
  GraphObserver *Observer;
  /// Whether to die on missing cases or to continue onward.
  BehaviorOnUnimplemented IgnoreUnimplemented = BehaviorOnUnimplemented::Abort;
};

} // namespace kythe

#endif // KYTHE_CXX_INDEXER_CXX_INDEXER_FRONTEND_ACTION_H_
