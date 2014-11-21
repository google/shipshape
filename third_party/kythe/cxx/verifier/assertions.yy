// Started from the calc++ example code as part of the Bison-3.0 distribution.
%skeleton "lalr1.cc"
%defines
%define parser_class_name {AssertionParserImpl}
%define api.token.constructor
%define api.value.type variant
%code requires
{
#include <string>
#include "assertion_ast.h"
#define yylex kythe::verifier::AssertionParser::lex
namespace kythe {
namespace verifier {
class AssertionParser;
}
}
}
%param { ::kythe::verifier::AssertionParser &context }
%locations
%initial-action
{
  @$.begin.filename = @$.end.filename = &context.file();
};
%define parse.trace
%define parse.error verbose
%code
{
#include "assertions.h"
#define newAst new (context.arena_) kythe::verifier::
}
%define api.token.prefix {TOK_}
%token
  END 0 "end of file"
  LPAREN  "("
  RPAREN  ")"
  COMMA   ","
  DONTCARE "_"
  APOSTROPHE "'"
  AT "@"
  DOT "."
  WHAT "?"
  EQUALS "="
;
%token <std::string> IDENTIFIER "identifier"
%token <std::string> STRING "string"
%token <std::string> NUMBER "number"
%type <kythe::verifier::AstNode*> exp
%type <kythe::verifier::AstNode*> atom
%type <kythe::verifier::AstNode*> goal
%type <kythe::verifier::AstNode*> exp_tuple_star
%type <std::string> string_or_identifier
%type <int> location_spec
%type <size_t> exp_tuple_plus
%printer { yyoutput << $$; } <*>;
%%
%start unit;
unit: goals  { };

goals:
  %empty {}
| goals goal { context.AppendGoal($2); }

string_or_identifier:
  "identifier" { $$ = $1; }
| "string" { $$ = $1; }

goal:
  exp string_or_identifier exp {
    $$ = context.CreateSimpleEdgeFact(@1 + @3, $1, $2, $3, nullptr);
  }
| exp "." string_or_identifier exp {
    $$ = context.CreateSimpleNodeFact(@1 + @4, $1, $3, $4);
  }
| exp string_or_identifier "." atom exp {
    $$ = context.CreateSimpleEdgeFact(@1 + @5, $1, $2, $5, $4);
  }

exp:
  atom exp_tuple_star { $$ = newAst App($1, $2); };
| atom { $$ = $1; };
| atom "=" exp {
    context.AppendGoal(context.CreateEqualityConstraint(@2, $1, $3));
    $$ = $1;
  };

atom:
    "identifier"      { $$ = context.CreateAtom(@1, $1); }
  | "string"          { $$ = context.CreateIdentifier(@1, $1); }
  | "_"               { $$ = context.CreateDontCare(@1); }
  | "number"          { $$ = context.CreateIdentifier(@1, $1); };
  | "@" location_spec { $$ = context.CreateAnchorSpec(@1); };
  | "identifier" "?"  { $$ = context.CreateInspect(@2, $1,
                                                   context.CreateAtom(@1, $1));
                      }
  | "_" "?"           { $$ = context.CreateInspect(@2, "_",
                                                   context.CreateDontCare(@1));
                      }

exp_tuple_plus:
    exp_tuple_plus "," exp { context.PushNode($3); $$ = $1 + 1; }
  | exp { context.PushNode($1); $$ = 1; }

exp_tuple_star:
    "(" ")" { $$ = newAst Tuple(@1, 0, nullptr); }
  | "(" exp_tuple_plus ")" {
    $$ = newAst Tuple(@1, $2, context.PopNodes($2));
  }

location_spec:
    string_or_identifier { context.PushLocationSpec($1); $$ = 0; }

%%
void yy::AssertionParserImpl::error(const location_type &l,
                                    const std::string &m) {
  context.Error(l, m);
}
