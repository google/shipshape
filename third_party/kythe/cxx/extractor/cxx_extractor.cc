#include "cxx_extractor.h"

#include <unordered_map>

#include <fcntl.h>
#include <sys/stat.h>

#include "clang/Frontend/CompilerInstance.h"
#include "clang/Frontend/FrontendAction.h"
#include "clang/Lex/HeaderSearch.h"
#include "clang/Lex/PPCallbacks.h"
#include "clang/Lex/Preprocessor.h"
#include "clang/Tooling/Tooling.h"
#include "gflags/gflags.h"
#include "glog/logging.h"
#include "third_party/kythe/proto/analysis.pb.h"
#include "llvm/Support/Path.h"
#include "sha2.h"

namespace kythe {
namespace {

/// \brief Hooks the Clang preprocessor to detect required include files.
class ExtractorPPCallbacks : public clang::PPCallbacks {
 public:
  ExtractorPPCallbacks(
      clang::SourceManager* source_manager, clang::Preprocessor* preprocessor,
      std::string* main_source_file,
      std::unordered_map<std::string, std::string>* source_files)
      : source_manager_(source_manager),
        preprocessor_(preprocessor),
        main_source_file_(main_source_file),
        source_files_(source_files) {}

  void FileChanged(clang::SourceLocation /*Loc*/, FileChangeReason Reason,
                   clang::SrcMgr::CharacteristicKind /*FileType*/,
                   clang::FileID /*PrevFID*/) override {
    if (Reason == EnterFile) {
      if (last_inclusion_directive_path_.empty()) {
        current_files_.push(GetMainFile()->getName());
      } else {
        current_files_.push(last_inclusion_directive_path_);
      }
    } else if (Reason == ExitFile) {
      current_files_.pop();
    }
  }

  /// \brief Records the content of `file` (with spelled path `path`)
  /// if it has not already been recorded.
  void AddFile(const clang::FileEntry* file, const std::string& path) {
    auto contents = source_files_->insert(std::make_pair(path, std::string()));
    if (contents.second) {
      const llvm::MemoryBuffer* buffer =
          source_manager_->getMemoryBufferForFile(file);
      contents.first->second.assign(buffer->getBufferStart(),
                                    buffer->getBufferEnd());
      LOG(INFO) << "added content for " << path << "\n";
    }
  }

  void InclusionDirective(
      clang::SourceLocation HashLoc, const clang::Token& IncludeTok,
      llvm::StringRef FileName, bool IsAngled, clang::CharSourceRange Range,
      const clang::FileEntry* File, llvm::StringRef SearchPath,
      llvm::StringRef RelativePath, const clang::Module* Imported) override {
    if (File == nullptr) {
      LOG(WARNING) << "Found null file: " << FileName.str();
      LOG(WARNING) << "Search path was " << SearchPath.str();
      LOG(WARNING) << "Relative path was " << RelativePath.str();
      LOG(WARNING) << "Imported was set to " << Imported;
      const auto* options =
          &preprocessor_->getHeaderSearchInfo().getHeaderSearchOpts();
      LOG(WARNING) << "Resource directory is " << options->ResourceDir;
      for (const auto& entry : options->UserEntries) {
        LOG(WARNING) << "User entry: " << entry.Path;
      }
      for (const auto& prefix : options->SystemHeaderPrefixes) {
        LOG(WARNING) << "System entry: " << prefix.Prefix;
      }
      LOG(WARNING) << "Sysroot set to " << options->Sysroot;
      return;
    }
    CHECK(!current_files_.top().empty());
    const auto* search_path_entry =
        source_manager_->getFileManager().getDirectory(SearchPath);
    const auto* current_file_parent_entry =
        source_manager_->getFileManager()
            .getFile(current_files_.top().c_str())
            ->getDir();
    // If the include file was found relatively to the current file's parent
    // directory or a search path, we need to normalize it. This is necessary
    // because llvm internalizes the path by which an inode was first accessed,
    // and always returns that path afterwards. If we do not normalize this
    // we will get an error when we replay the compilation, as the virtual
    // file system is not aware of inodes.
    if (search_path_entry == current_file_parent_entry) {
      auto parent =
          llvm::sys::path::parent_path(current_files_.top().c_str()).str();

      // If the file is a top level file ("file.cc"), we normalize to a path
      // relative to "./".
      if (parent.empty() || parent == "/") {
        parent = ".";
      }

      // Otherwise we take the literal path as we stored it for the current
      // file, and append the relative path.
      last_inclusion_directive_path_ = parent + "/" + RelativePath.str();
    } else if (!SearchPath.empty()) {
      last_inclusion_directive_path_ =
          SearchPath.str() + "/" + RelativePath.str();
    } else {
      CHECK(llvm::sys::path::is_absolute(FileName)) << FileName.str();
      last_inclusion_directive_path_ = FileName.str();
    }
    AddFile(File, last_inclusion_directive_path_);
  }

  void EndOfMainFile() override {
    AddFile(GetMainFile(), GetMainFile()->getName());
  }

 private:
  /// \brief Returns the main file for this compile action.
  const clang::FileEntry* GetMainFile() {
    return source_manager_->getFileEntryForID(source_manager_->getMainFileID());
  }

  /// The `SourceManager` used for the compilation.
  clang::SourceManager* source_manager_;
  /// The `Preprocessor` we're attached to.
  clang::Preprocessor* preprocessor_;
  /// The path of the file that was last referenced by an inclusion directive,
  /// normalized for includes that are relative to a different source file.
  std::string last_inclusion_directive_path_;
  /// The stack of files we've entered. top() gives the current file.
  std::stack<std::string> current_files_;
  /// The main source file for the compilation (assuming only one).
  std::string* const main_source_file_;
  /// Contents of the files we've used, indexed by normalized path.
  std::unordered_map<std::string, std::string>* const source_files_;
};

class ExtractorAction : public clang::PreprocessorFrontendAction {
 public:
  explicit ExtractorAction(ExtractorCallback callback) : callback_(callback) {}

  void ExecuteAction() override {
    const auto inputs = getCompilerInstance().getFrontendOpts().Inputs;
    CHECK_EQ(1, inputs.size()) << "Expected to see only one TU; instead saw "
                               << inputs.size() << ".";
    main_source_file_ = inputs[0].getFile();
    auto* preprocessor = &getCompilerInstance().getPreprocessor();
    auto* callbacks = new ExtractorPPCallbacks(
        &getCompilerInstance().getSourceManager(), preprocessor,
        &main_source_file_, &source_files_);
    preprocessor->addPPCallbacks(callbacks);  // Clang takes ownership.
    preprocessor->EnterMainSourceFile();
    clang::Token token;
    do {
      preprocessor->Lex(token);
    } while (token.isNot(clang::tok::eof));
  }

  void EndSourceFileAction() override {
    callback_(main_source_file_, source_files_,
              getCompilerInstance().getDiagnostics().hasErrorOccurred());
  }

 private:
  ExtractorCallback callback_;
  /// The main source file for the compilation (assuming only one).
  std::string main_source_file_;
  /// Contents of the files we've used, indexed by normalized path.
  std::unordered_map<std::string, std::string> source_files_;
};

}  // anonymous namespace

void KindexWriterSink::OpenIndex(const std::string& path) {
  using namespace google::protobuf::io;
  CHECK(open_path_.empty() && fd_ < 0)
      << "Reopening a KindexWriterSink (old fd:" << fd_
      << " old path: " << open_path_ << ")";
  fd_ = open(path.c_str(), O_WRONLY | O_CREAT | O_TRUNC, S_IREAD | S_IWRITE);
  CHECK_GE(fd_, 0) << "Couldn't open output file " << path;
  open_path_ = path;
  file_stream_.reset(new FileOutputStream(fd_));
  GzipOutputStream::Options options;
  // Accept the default compression level and compression strategy.
  options.format = GzipOutputStream::GZIP;
  gzip_stream_.reset(new GzipOutputStream(file_stream_.get(), options));
  coded_stream_.reset(new CodedOutputStream(gzip_stream_.get()));
}

KindexWriterSink::~KindexWriterSink() {
  CHECK(!coded_stream_->HadError()) << "Errors encountered writing to "
                                    << open_path_;
  coded_stream_.reset(nullptr);
  gzip_stream_.reset(nullptr);
  file_stream_.reset(nullptr);
  close(fd_);
}

void KindexWriterSink::WriteHeader(
    const kythe::proto::CompilationUnit& header) {
  coded_stream_->WriteVarint32(header.ByteSize());
  CHECK(header.SerializeToCodedStream(coded_stream_.get()))
      << "Couldn't write header to " << open_path_;
}

void KindexWriterSink::WriteFileContent(const kythe::proto::FileData& content) {
  coded_stream_->WriteVarint32(content.ByteSize());
  CHECK(content.SerializeToCodedStream(coded_stream_.get()))
      << "Couldn't write content to " << open_path_;
}

// We need "the lowercase ascii hex SHA-256 digest of the file contents."
static constexpr char kHexDigits[] = "0123456789abcdef";

/// \brief Returns the lowercase-string-hex-encoded sha256 digest of the first
/// `length` bytes of `bytes`.
static std::string Sha256(const void* bytes, size_t length) {
  unsigned char sha_buf[SHA256_DIGEST_SIZE];
  sha256(reinterpret_cast<const unsigned char*>(bytes), length, sha_buf);
  std::string sha_text(SHA256_DIGEST_SIZE * 2, '\0');
  for (unsigned i = 0; i < SHA256_DIGEST_SIZE; ++i) {
    sha_text[i * 2] = kHexDigits[(sha_buf[i] >> 4) & 0xF];
    sha_text[i * 2 + 1] = kHexDigits[sha_buf[i] & 0xF];
  }
  return sha_text;
}

bool IndexWriter::SetVNameConfiguration(const std::string& json) {
  std::string error_text;
  if (!vname_generator_.LoadJsonString(json, &error_text)) {
    LOG(ERROR) << "Could not parse vname generator configuration: "
               << error_text;
    return false;
  }
  return true;
}

kythe::proto::VName IndexWriter::VNameForPath(const std::string& path) {
  kythe::proto::VName out = vname_generator_.LookupVName(path);
  out.set_language("c++");
  if (!out.has_corpus()) {
    out.set_corpus(corpus_);
  }
  return out;
}

std::string IndexWriter::MakeCleanAbsolutePath(const std::string& in_path) {
  using namespace llvm::sys::path;
  std::string abs_path = clang::tooling::getAbsolutePath(in_path);
  std::string root_part =
      (root_name(abs_path) + root_directory(abs_path)).str();
  llvm::SmallString<1024> out_path = llvm::StringRef(root_part);
  std::vector<llvm::StringRef> path_components;
  int skip_count = 0;
  for (reverse_iterator node = rbegin(abs_path), node_end = rend(abs_path);
       node != node_end; ++node) {
    if (*node == "..") {
      ++skip_count;
    } else if (*node != ".") {
      if (skip_count > 0) {
        --skip_count;
      } else {
        path_components.push_back(*node);
      }
    }
  }
  for (auto node = path_components.crbegin(),
            node_end = path_components.crend();
       node != node_end; ++node) {
    append(out_path, *node);
  }
  return out_path.str();
}

std::string IndexWriter::RelativizePath(const std::string& to_relativize,
                                        const std::string& relativize_against) {
  std::string to_relativize_abs = MakeCleanAbsolutePath(to_relativize);
  std::string relativize_against_abs =
      MakeCleanAbsolutePath(relativize_against);
  llvm::StringRef to_relativize_parent =
      llvm::sys::path::parent_path(to_relativize_abs);
  std::string ret =
      to_relativize_parent.startswith(relativize_against_abs)
          ? to_relativize_abs.substr(relativize_against_abs.size() +
                                     llvm::sys::path::get_separator().size())
          : to_relativize_abs;
  return ret;
}

void IndexWriter::FillFileInput(
    const std::string& clang_path, const std::string& sha256,
    kythe::proto::CompilationUnit_FileInput* file_input) {
  file_input->mutable_v_name()->CopyFrom(
      VNameForPath(RelativizePath(clang_path, root_directory_)));
  // This path is distinct from the VName path. It is used by analysis tools
  // to configure Clang's virtual filesystem.
  auto* file_info = file_input->mutable_info();
  file_info->set_path(clang_path);
  file_info->set_digest(sha256);
}

void IndexWriter::WriteIndex(
    std::unique_ptr<IndexWriterSink> sink, const std::string& main_source_file,
    const std::unordered_map<std::string, std::string>& source_files,
    bool had_errors) {
  kythe::proto::CompilationUnit unit;
  std::string identifying_blob;
  identifying_blob.append(corpus_);
  for (const auto& arg : args_) {
    identifying_blob.append(arg);
    unit.add_argument(arg);
  }
  identifying_blob.append(main_source_file);
  std::string identifying_blob_digest =
      Sha256(identifying_blob.c_str(), identifying_blob.size());
  auto* unit_vname = unit.mutable_v_name();

  kythe::proto::VName main_vname = VNameForPath(main_source_file);
  unit_vname->CopyFrom(main_vname);
  unit_vname->set_signature("cu#" + identifying_blob_digest);
  unit_vname->clear_path();

  for (const auto& file : source_files) {
    FillFileInput(file.first, Sha256(file.second.c_str(), file.second.size()),
                  unit.add_required_input());
  }
  unit.set_has_compile_errors(had_errors);
  unit.add_source_file(main_source_file);
  unit.set_working_directory(root_directory_);
  std::string output_path = output_directory_;
  output_path.append("/" + identifying_blob_digest + ".kindex");
  sink->OpenIndex(output_path);
  sink->WriteHeader(unit);
  unsigned info_index = 0;
  for (const auto& file : source_files) {
    kythe::proto::FileData file_content;
    file_content.set_content(file.second);
    file_content.mutable_info()->CopyFrom(
        unit.required_input(info_index++).info());
    sink->WriteFileContent(file_content);
  }
}

std::unique_ptr<clang::FrontendAction> NewExtractor(
    ExtractorCallback callback) {
  return std::unique_ptr<clang::FrontendAction>(new ExtractorAction(callback));
}

}  // namespace kythe
