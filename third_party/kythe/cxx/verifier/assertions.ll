%{
// Started from the calc++ example code as part of the Bison-3.0 distribution.
#include "assertions.h"
#include "assertions.tab.hh"

// The location of the current token (as row and column).
static yy::location loc;
// The offset of the current token (as byte offset).
static size_t loc_ofs;
%}
%option noyywrap nounput batch debug noinput
id    [a-zA-Z/][a-zA-Z_0-9/]*
int   [0-9]+
blank [ \t]

%{
  // Code run each time a pattern is matched.
  #define YY_USER_ACTION  loc_ofs += yyleng; loc.columns(yyleng);
%}

/* The lexer has the following states:
 *   INITIAL: We aren't sure whether this line is relevant to parsing
 *            rules; check after every character whether to switch states.
 *   IGNORED: This line is definitely not one that contains input for our
 *            parser. While still updating the file location, wait until
 *            an endline.
 *    NORMAL: This line contains input that must be passed on to the parser. */
%s IGNORED NORMAL
%%

%{
  // Code run each time yylex is called.
  loc.step();
%}

<INITIAL>{
\n       loc.lines(yyleng); loc.step(); context.ResetLexCheck();
.        {
            int lex_check = context.NextLexCheck(yytext);
            if (lex_check < 0) {
              BEGIN(IGNORED);
            } else if (lex_check > 0) {
              BEGIN(NORMAL);
            }
         }
}  /* INITIAL state */

<IGNORED>{
\n       {
            // Resolve locations after the first endline.
            if (!context.ResolveLocations(loc, loc_ofs)) {
              context.Error(loc, "could not resolve all locations");
            }
            loc.lines(yyleng);
            loc.step();
            BEGIN(INITIAL);
         }
[^\n]*   context.AppendToLine(yytext);
}  /* IGNORED state */

<NORMAL>{
{blank}+ loc.step ();
\n      loc.lines(yyleng); loc.step(); context.ResetLexCheck(); BEGIN(INITIAL);
"("      return yy::AssertionParserImpl::make_LPAREN(loc);
")"      return yy::AssertionParserImpl::make_RPAREN(loc);
","      return yy::AssertionParserImpl::make_COMMA(loc);
"_"      return yy::AssertionParserImpl::make_DONTCARE(loc);
"'"      return yy::AssertionParserImpl::make_APOSTROPHE(loc);
"@"      return yy::AssertionParserImpl::make_AT(loc);
"."      return yy::AssertionParserImpl::make_DOT(loc);
"?"      return yy::AssertionParserImpl::make_WHAT(loc);
"="      return yy::AssertionParserImpl::make_EQUALS(loc);
{int}    return yy::AssertionParserImpl::make_NUMBER(yytext, loc);
{id}     return yy::AssertionParserImpl::make_IDENTIFIER(yytext, loc);
\"(\\.|[^\\"])*\" {
                   std::string out;
                   if (!context.Unescape(yytext, &out)) {
                     context.Error(loc, "invalid literal string");
                   }
                   return yy::AssertionParserImpl::make_STRING(out, loc);
                 }
.        context.Error (loc, "invalid character");
}  /* NORMAL state */

<<EOF>>  {
            context.save_eof(loc, loc_ofs);
            return yy::AssertionParserImpl::make_END(loc);
         }
%%

namespace kythe {
namespace verifier {

static YY_BUFFER_STATE stringBufferState = nullptr;
static std::string *kNoFile = new std::string("no-file");

void AssertionParser::ScanBeginString(const std::string &data,
                                      bool trace_scanning) {
  BEGIN(INITIAL);
  loc = yy::location(&file_, 0, 1u);
  loc_ofs = 0;
  yy_flex_debug = trace_scanning;
  assert(stringBufferState == nullptr);
  stringBufferState = yy_scan_bytes(data.c_str(), data.size());
}

void AssertionParser::ScanBeginFile(bool trace_scanning) {
  BEGIN(INITIAL);
  loc = yy::location(&file_, 0, 1u);
  loc_ofs = 0;
  yy_flex_debug = trace_scanning;
  if (file_.empty() || file_ == "-") {
    yyin = stdin;
  } else if (!(yyin = fopen(file_.c_str(), "r"))) {
    Error("cannot open " + file_ + ": " + strerror(errno));
    exit(EXIT_FAILURE);
  }
}

void AssertionParser::ScanEnd(const yy::location &eof_loc,
                              size_t eof_loc_ofs) {
  // Imagine that all files end with an endline.
  if (!ResolveLocations(eof_loc, eof_loc_ofs + 1)) {
    Error(eof_loc, "could not resolve all locations at end of file");
  }
  if (stringBufferState) {
    yy_delete_buffer(stringBufferState);
    stringBufferState = nullptr;
  } else {
    fclose(yyin);
  }
}

}
}
