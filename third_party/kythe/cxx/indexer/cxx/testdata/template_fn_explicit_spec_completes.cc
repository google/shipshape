// Checks that we get completion edges for explicit function template specs.
template <typename T> T id(T x);
//- @id defines Decl1
template <> int id(int x);
//- @id defines Defn
//- @id ucompletes Decl2
//- @id ucompletes Decl1
template <> int id(int x) { return x; }
//- @id defines Decl2
template <> int id(int x);
