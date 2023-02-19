package errorsWithData

import (
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
)

func TestTokenizer(t *testing.T) {

	test_token_case := func(s string, expected string) {
		tokenized := tokenizeFormatString(s)
		tokenized_as_string := tokenized.String()
		testutils.FatalUnless(t, tokenized_as_string == expected, "tokenizer did not get expected result for input string \"%s\".\nGot: %s. Expected:%s\n", s, tokenized_as_string, expected)
	}
	test_token_case("", "[ ]")
	test_token_case("abc\ndef", "[ \"abc\ndef\" ]")
	test_token_case(`\\\\\`, `[ "\\\" ]`)
	test_token_case("%", `[ % ]`)
	test_token_case("$", `[ $ ]`)
	test_token_case("{", `[ { ]`)
	test_token_case("}", `[ } ]`)
	test_token_case(`\{`, `[ "{" ]`)
	test_token_case(`\%`, `[ "%" ]`)
	test_token_case(`\$`, `[ "$" ]`)
	test_token_case(`\}`, `[ "}" ]`)
	test_token_case("%w", `[ %w ]`)
	test_token_case("$w", `[ $w ]`)
	test_token_case(`\%w%ww%`, `[ "%w" %w "w" % ]`)
	test_token_case("%!w", `[ %! "w" ]`)
	test_token_case("$!w", `[ $! "w" ]`)
	test_token_case(`%{Foo\%}`, `[ % { "Foo%" } ]`)
	test_token_case(`${Foo\%}`, `[ $ { "Foo%" } ]`)
	// test_token_case("$%%x{%{")
}

func TestParser(t *testing.T) {
	test_parse_case := func(s string, expected string) {
		tokenized := tokenizeFormatString(s)
		parse_result, err := make_ast(tokenized)
		ast_as_string := parse_result.String()
		if err != nil {
			t.Fatalf("Parsing error when processing input %v, tokenized as %v\n Built-up ast was %v\n Error was %v\n", s, tokenized, ast_as_string, err)
		}

		testutils.FatalUnless(t, ast_as_string == expected, "parser did not get expected result for input string \"%s\".\nGot: %s. Expected:%s\n", s, ast_as_string, expected)
	}
	test_parse_case(``, `AST([])`)
	test_parse_case(`ABC`, `AST("ABC")`)
	test_parse_case(`%w%w`, `AST([%w,%w])`)
	test_parse_case(`%w`, `AST(%w)`)
	test_parse_case(`ABC%wDEF`, `AST(["ABC",%w,"DEF"])`)
	test_parse_case(`%{\$Foo}`, `AST(%v{$Foo})`)
	test_parse_case(`a$!C{DEF}`, `AST(["a",$!C{"DEF"}])`)
	test_parse_case(`a%!C1{%!C2{a$w}}`, `AST(["a",%!C1{%!C2{["a",$w]}}])`)
}
