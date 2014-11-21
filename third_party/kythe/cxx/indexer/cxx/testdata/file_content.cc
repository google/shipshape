// Checks that the indexer emits file nodes with content.
#include "a.h"
//- vname("", "", "", "third_party/kythe/cxx/indexer/cxx/testdata/file_content.cc",
//-   "c++").node/kind file
//- vname("", "", "", "third_party/kythe/cxx/indexer/cxx/testdata/a.h", "c++")
//-   .node/kind file
//- vname("", "", "", "third_party/kythe/cxx/indexer/cxx/testdata/a.h", "c++")
//-   .text "#ifndef A_H_\n#define A_H_\n#endif  // A_H_"