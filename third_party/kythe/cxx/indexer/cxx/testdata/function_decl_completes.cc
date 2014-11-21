// Checks that function defns complete function decls.
#include "void_f.h"
//- @f defines FSDecl
void f();
//- @f defines FSDefn
//- @f completes FHDecl
//- @f ucompletes FSDecl
void f() { }
//- FHAnchor defines FHDecl
//- FHAnchor childof vname(_,_,_,"third_party/kythe/cxx/indexer/cxx/testdata/void_f.h",_)