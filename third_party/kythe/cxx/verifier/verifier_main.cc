#include <stdio.h>
#include <unistd.h>
#include <string>

#include "google/protobuf/io/zero_copy_stream.h"
#include "google/protobuf/io/zero_copy_stream_impl.h"
#include "google/protobuf/io/coded_stream.h"
#include "leveldb/db.h"
#include "third_party/glog/include/glog/logging.h"

#include "third_party/kythe/proto/storage.pb.h"

#include "assertion_ast.h"
#include "verifier.h"

static const size_t npos = -1;

// Decodes the next Kythe key field starting at data[offset], returning
// npos on failure or the offset of the delimiter.
size_t NextField(const char *data, size_t offset, size_t max_offset,
                 const char delimiter = '\n') {
  while (offset < max_offset && data[offset] != delimiter) {
    ++offset;
  }
  if (offset > max_offset) {
    return npos;
  }
  return offset;
}

// Decodes the vname starting at data[offset], returning npos on failure
// or the offset of the byte following the vname.
size_t DecodeVName(const char *data, size_t offset, size_t max_offset,
                   kythe::proto::VName *vname) {
  size_t signature_end = NextField(data, offset, max_offset, 0);
  if (signature_end == npos) return npos;
  size_t corpus_end = NextField(data, signature_end + 1, max_offset, 0);
  if (corpus_end == npos) return npos;
  size_t root_end = NextField(data, corpus_end + 1, max_offset, 0);
  if (root_end == npos) return npos;
  size_t path_end = NextField(data, root_end + 1, max_offset, 0);
  if (path_end == npos) return npos;
  size_t language_end = NextField(data, path_end + 1, max_offset, '\n');
  if (language_end == npos) return npos;
  vname->set_signature(data + offset, signature_end - offset);
  vname->set_corpus(data + signature_end + 1, corpus_end - signature_end - 1);
  vname->set_root(data + corpus_end + 1, root_end - corpus_end - 1);
  vname->set_path(data + root_end + 1, path_end - root_end - 1);
  vname->set_language(data + path_end + 1, language_end - path_end - 1);
  return language_end;
}

DEFINE_string(leveldb, "", "Path to leveldb storage");
DEFINE_bool(show_protos, false, "Show protocol buffers read from standard in");
DEFINE_bool(show_goals, false, "Show goals after parsing");
DEFINE_bool(ignore_dups, false, "Ignore duplicate facts during verification");
DEFINE_bool(graphviz, false, "Only dump facts as a GraphViz-compatible graph");
DEFINE_bool(json, false, "Only dump facts as JSON");

int main(int argc, char **argv) {
  GOOGLE_PROTOBUF_VERIFY_VERSION;
  ::google::SetVersionString("0.1");
  ::google::SetUsageMessage(R"(Verification tool for Kythe databases.
Reads Kythe facts from standard input or from LevelDB and checks them against
one or more rule files. See the DESIGN file for more details on invocation and
rule syntax.

Example:
  ${INDEXER_BIN} -i $1 | ${VERIFIER_BIN} --show_protos --show_goals $1
  cat foo.entries | ${VERIFIER_BIN} goals1.cc goals2.cc
)");
  ::google::ParseCommandLineFlags(&argc, &argv, true);
  ::google::InitGoogleLogging(argv[0]);

  kythe::verifier::Verifier v;

  if (FLAGS_ignore_dups) {
    v.IgnoreDuplicateFacts();
  }

  if (!FLAGS_graphviz && !FLAGS_json) {
    std::vector<std::string> rule_files(argv + 1, argv + argc);
    if (rule_files.empty()) {
      fprintf(stderr, "No rule files specified\n");
      return 1;
    }

    for (const auto &rule_file : rule_files) {
      if (!v.LoadInlineRuleFile(rule_file)) {
        fprintf(stderr, "Failed loading %s.\n", rule_file.c_str());
        return 1;
      }
    }
  }

  std::string dbname = "database";
  size_t facts = 0;

  if (!FLAGS_leveldb.empty()) {
    leveldb::DB *db;
    leveldb::Options options;
    options.create_if_missing = false;
    leveldb::Status status =
        leveldb::DB::Open(options, FLAGS_leveldb.c_str(), &db);
    if (!status.ok()) {
      fprintf(stderr, "LevelDB error: %s\n", status.ToString().c_str());
      return 1;
    }
    leveldb::Iterator *it = db->NewIterator(leveldb::ReadOptions());
    for (it->SeekToFirst(); it->Valid(); ++facts, it->Next()) {
      kythe::proto::Entry entry;
      size_t max_offset = it->key().size();
      const char *data = it->key().data();
      size_t source_end =
          DecodeVName(data, 0, max_offset, entry.mutable_source());
      if (source_end == npos) {
        fprintf(stderr, "Error decoding source VName at fact %zu\n", facts);
        continue;
      }
      size_t edge_kind_end = NextField(data, source_end + 1, max_offset);
      if (edge_kind_end == npos) {
        fprintf(stderr, "Error decoding edge kind at fact %zu\n", facts);
        continue;
      }
      size_t fact_name_end = NextField(data, edge_kind_end + 1, max_offset);
      if (fact_name_end == npos) {
        fprintf(stderr, "Error decoding fact name at fact %zu\n", facts);
        continue;
      }
      entry.set_edge_kind(data + source_end + 1,
                          edge_kind_end - source_end - 1);
      entry.set_fact_name(data + edge_kind_end + 1,
                          fact_name_end - edge_kind_end - 1);
      if (fact_name_end + 1 != max_offset) {
        size_t target_end = DecodeVName(data, fact_name_end + 1, max_offset,
                                        entry.mutable_target());
        if (target_end == npos || target_end != max_offset) {
          fprintf(stderr, "Error decoding target VName at fact %zu\n", facts);
          exit(1);
          continue;
        }
      }
      entry.set_fact_value(it->value().data(), it->value().size());
      if (FLAGS_show_protos) {
        entry.PrintDebugString();
      }
      v.AssertSingleFact(&dbname, facts, entry);
    }
    status = it->status();
    if (!status.ok()) {
      fprintf(stderr, "LevelDB error: %s\n", status.ToString().c_str());
      return 1;
    }
    delete it;
    delete db;
  } else {
    kythe::proto::Entry entry;
    google::protobuf::uint32 byte_size;
    google::protobuf::io::FileInputStream raw_input(STDIN_FILENO);
    google::protobuf::io::CodedInputStream coded_input(&raw_input);
    while (coded_input.ReadVarint32(&byte_size)) {
      auto limit = coded_input.PushLimit(byte_size);
      if (!entry.ParseFromCodedStream(&coded_input)) {
        fprintf(stderr, "Error reading around fact %zu\n", facts);
        return 1;
      }
      coded_input.PopLimit(limit);
      if (FLAGS_show_protos) {
        entry.PrintDebugString();
      }
      v.AssertSingleFact(&dbname, facts, entry);
      ++facts;
    }
  }

  if (FLAGS_show_goals) {
    v.ShowGoals();
  }

  if (FLAGS_graphviz) {
    v.DumpAsDot();
  }

  if (FLAGS_json) {
    v.DumpAsJson();
  }

  if (!v.VerifyAllGoals()) {
    fprintf(stderr,
            "Could not verify all goals. The furthest we reached was:\n  ");
    v.DumpErrorGoal(v.highest_goal_reached());
    return 1;
  }

  return 0;
}
