// Checks that completion edges are properly recorded for specializations.

//- @C defines TemplateTS
template <typename T, typename S> class C;
//- @C defines TemplateT
template <typename T> class C<int, T>;
//- @C defines Template
template <> class C<int, float>;

//- @C ucompletes TemplateTS
template <typename T, typename S> class C { };

//- @C ucompletes TemplateT
template <typename T> class C<int, T> { };

//- @C ucompletes Template
template <> class C<int, float> { };
