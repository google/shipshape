// Checks that we index function templates with mutliple forward declarations.
//- @f defines Decl1
template <typename T> void f();
//- @f defines Decl2
template <typename T> void f();
//- @f defines Defn
//- @f ucompletes Decl1
//- @f ucompletes Decl2
template <typename T> void f() { }
