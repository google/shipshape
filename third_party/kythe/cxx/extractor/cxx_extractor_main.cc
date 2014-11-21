// cxx_extractor is meant to be a drop-in replacement for clang/gcc's frontend.
// It collects all of the resources that clang would use to compile a single
// source file (as determined by the command line arguments) and produces a
// .kindex file.
//
// We read environment variables KYTHE_CORPUS (to set the default corpus),
// KYTHE_ROOT_DIRECTORY (to set the default root directory and to configure
// Clang's header search), KYTHE_OUTPUT_DIRECTORY (to control where kindex
// files are deposited), and KYTHE_VNAMES (to control vname generation).
//
// If the first two arguments are --with_executable /foo/bar, the extractor
// will consider /foo/bar to be the executable it was called as for purposes
// of argument interpretation. These arguments are then stripped.

// If --resource-dir (a Clang argument) is *not* provided, the path to the
// extractor's actual executable (or at least the one it was called from) is
// used to infer the location of certain "builtin" header files.

#include "cxx_extractor.h"

#include "clang/Basic/Version.h"
#include "clang/Frontend/FrontendActions.h"
#include "clang/Tooling/Tooling.h"
#include "gflags/gflags.h"
#include "glog/logging.h"
#include "google/protobuf/stubs/common.h"
#include "third_party/kythe/proto/analysis.pb.h"
#include "third_party/kythe/cxx/common/CommandLineUtils.h"
#include "llvm/Support/Path.h"

int main(int argc, char* argv[]) {
  GOOGLE_PROTOBUF_VERIFY_VERSION;
  google::InitGoogleLogging(argv[0]);
  google::SetVersionString("0.1");
  std::vector<std::string> final_args(argv, argv + argc);
  std::string actual_executable = final_args.size() ? final_args[0] : "";
  if (final_args.size() >= 3 && final_args[1] == "--with_executable") {
    final_args.assign(final_args.begin() + 2, final_args.end());
  }
  final_args = kythe::common::GCCArgsToClangSyntaxOnlyArgs(final_args);
  kythe::IndexWriter index_writer;
  // Check to see if an alternate resource-dir was specified; otherwise,
  // invent one. We need this to find stddef.h and friends.
  if (std::find(final_args.begin(), final_args.end(), "-resource-dir") ==
      final_args.end()) {
    std::string llvm_root_directory = actual_executable;
    llvm_root_directory =
        llvm_root_directory.substr(0, llvm_root_directory.find_last_of('/'));
    llvm_root_directory.append(
        "/../../../../../third_party/llvm/lib/clang/" CLANG_VERSION_STRING);
    final_args.insert(final_args.begin() + 1, llvm_root_directory);
    final_args.insert(final_args.begin() + 1, "-resource-dir");
  }
  // Store the arguments post-filtering.
  index_writer.set_args(final_args);
  clang::FileSystemOptions file_system_options;
  if (auto* env_corpus = getenv("KYTHE_CORPUS")) {
    index_writer.set_corpus(env_corpus);
  }
  if (auto* vname_file = getenv("KYTHE_VNAMES")) {
    FILE* vname_handle = fopen(vname_file, "rb");
    CHECK(vname_handle != nullptr) << "Couldn't open input vnames file "
                                   << vname_file;
    CHECK_EQ(fseek(vname_handle, 0, SEEK_END), 0) << "Couldn't seek "
                                                  << vname_file;
    long vname_size = ftell(vname_handle);
    CHECK_GE(vname_size, 0) << "Bad size for " << vname_file;
    CHECK_EQ(fseek(vname_handle, 0, SEEK_SET), 0) << "Couldn't seek "
                                                  << vname_file;
    std::string vname_content;
    vname_content.resize(vname_size);
    CHECK_EQ(fread(&vname_content[0], vname_size, 1, vname_handle), 1)
        << "Couldn't read " << vname_file;
    CHECK_NE(fclose(vname_handle), EOF) << "Couldn't close " << vname_file;
    if (!index_writer.SetVNameConfiguration(vname_content)) {
      fprintf(stderr, "Couldn't configure vnames from %s\n", vname_file);
      exit(1);
    }
  }
  if (auto* env_root_directory = getenv("KYTHE_ROOT_DIRECTORY")) {
    index_writer.set_root_directory(env_root_directory);
    file_system_options.WorkingDir = env_root_directory;
  }
  if (auto* env_output_directory = getenv("KYTHE_OUTPUT_DIRECTORY")) {
    index_writer.set_output_directory(env_output_directory);
  }
  {
    llvm::IntrusiveRefCntPtr<clang::FileManager> file_manager(
        new clang::FileManager(file_system_options));
    auto extractor = kythe::NewExtractor([&index_writer](
        const std::string& main_source_file,
        const std::unordered_map<std::string, std::string>& source_files,
        bool had_errors) {
      index_writer.WriteIndex(std::unique_ptr<kythe::KindexWriterSink>(
                                  new kythe::KindexWriterSink()),
                              main_source_file, source_files, had_errors);
    });
    clang::tooling::ToolInvocation invocation(final_args, extractor.release(),
                                              file_manager.get());
    invocation.run();
  }
  google::protobuf::ShutdownProtobufLibrary();
  return 0;
}
