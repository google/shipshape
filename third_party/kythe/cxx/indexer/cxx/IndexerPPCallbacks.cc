// This file uses the Clang style conventions.

#include "GraphObserver.h"
#include "IndexerPPCallbacks.h"

#include "clang/Basic/SourceManager.h"
#include "clang/Basic/FileManager.h"
#include "clang/Lex/PPCallbacks.h"
#include "clang/Lex/Preprocessor.h"

#include "llvm/Support/raw_ostream.h"
#include "llvm/Support/Debug.h"

namespace kythe {

IndexerPPCallbacks::IndexerPPCallbacks(const clang::Preprocessor &PP,
                                       GraphObserver &GO)
    : Preprocessor(PP), Observer(GO) {}

IndexerPPCallbacks::~IndexerPPCallbacks() {}

void IndexerPPCallbacks::FileChanged(clang::SourceLocation Loc,
                                     PPCallbacks::FileChangeReason Reason,
                                     clang::SrcMgr::CharacteristicKind FileType,
                                     clang::FileID PrevFID) {
  switch (Reason) {
  case clang::PPCallbacks::EnterFile:
    Observer.pushFile(Loc);
    break;
  case clang::PPCallbacks::ExitFile:
    Observer.popFile();
    break;
  case clang::PPCallbacks::SystemHeaderPragma:
    break;
  // RenameFile occurs when a #line directive is encountered, for example:
  // #line 10 "foo.cc"
  case clang::PPCallbacks::RenameFile:
    break;
  default:
    llvm::dbgs() << "Unknown FileChangeReason " << Reason << "\n";
  }
}

void IndexerPPCallbacks::EndOfMainFile() { Observer.popFile(); }
}
