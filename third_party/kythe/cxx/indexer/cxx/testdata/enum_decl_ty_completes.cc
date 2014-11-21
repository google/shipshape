// Checks that enumerations complete forward declarations (with types).
//- @E defines EEnumFwdT
//- EEnumFwdT.complete complete
//- EEnumFwdT named EEnumName
//- EEnumFwdT is ShortType
enum class E : short;
//- @E defines EEnum
//- EEnum.complete definition
//- @E ucompletes EEnumFwdT
//- EEnum is ShortType
//- EEnum named EEnumName
enum class E : short { };
