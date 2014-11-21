// Checks that forward declarations from headers are not ucompleted in TUs.
#include "enum_decl_ty_header_completes.h"
//- HeaderAnchor defines HEnumEFwd
//- HeaderAnchor childof vname(_,_,_,
//-     "third_party/kythe/cxx/indexer/cxx/testdata/enum_decl_ty_header_completes.h",_)
//- @E defines EnumEFwd
//- EnumEFwd named EnumEName
//- EnumEName.node/kind name
//- EnumEFwd.complete complete
enum class E : short;
//- @E defines EnumE
//- @E ucompletes EnumEFwd
//- @E completes HEnumEFwd
enum class E : short { };
//- EnumE named EnumEName
//- EnumE is ShortType
//- EnumE.complete definition
//- HEnumEFwd.complete complete
//- HEnumEFwd is ShortType
//- HEnumEFwd named EnumEName