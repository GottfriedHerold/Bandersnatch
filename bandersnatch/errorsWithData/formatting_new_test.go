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
	test_token_case("%%%%%", `[ "%%" % ]`)
	test_token_case("%", `[ % ]`)
	test_token_case("$", `[ $ ]`)
	test_token_case("{", `[ { ]`)
	test_token_case("}", `[ } ]`)
	test_token_case("{{", `[ "{" ]`)
	test_token_case("%%", `[ "%" ]`)
	test_token_case("$$", `[ "$" ]`)
	test_token_case("%}", `[ "}" ]`)
	test_token_case("%w", `[ %w ]`)
	test_token_case("$w", `[ $w ]`)
	test_token_case("%%w%ww%", `[ "%w" %w "w" % ]`)
	test_token_case("%!w", `[ %! "w" ]`)
	test_token_case("$!w", `[ $! "w" ]`)
	test_token_case("%{Foo%%}", `[ % { "Foo%" } ]`)
	test_token_case("${Foo%%}", `[ $ { "Foo%" } ]`)
}

var (
	_ ast_I = new_ast_root()
	_ ast_I = new_ast_list()
	_ ast_I = new_ast_string("")
	_ ast_I = new_ast_fmtPercent()
	_ ast_I = new_ast_fmtDollar()
	_ ast_I = new_ast_parentPercent()
	_ ast_I = new_ast_parentDollar()
	_ ast_I = new_ast_condPercent()
	_ ast_I = new_ast_condDollar()
)
