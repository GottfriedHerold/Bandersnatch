package errorsWithData

import (
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
)

func TestTokenizer(t *testing.T) {
	tokens0 := tokenizeFormatString("")
	testutils.FatalUnless(t, len(tokens0) == 0, "")

	test_token_case := func(s string, expected string) {
		tokenized := tokenizeFormatString(s)
		tokenized_as_string := tokenized.String()
		testutils.FatalUnless(t, tokenized_as_string == expected, "tokenizer did not get expected result for input string %s.\nGot: %s. Expected:%s\n", s, tokenized_as_string, expected)
	}
	test_token_case("", "[]")
	test_token_case("abc\ndef", "[\"abc\ndef\"]")
	test_token_case("%%%%%", `["%%" %]`)
	test_token_case("%", `[%]`)
	test_token_case("$", `[$]`)
	test_token_case("{", `[{]`)
	test_token_case("}", `[}]`)
	test_token_case("{{", `["{"]`)
	test_token_case("%%", `["%"]`)
	test_token_case("$$", `["$"]`)
	test_token_case("%}", `["}"]`)
	test_token_case("%w", `[%w]`)
	test_token_case("$w", `[$w]`)
	test_token_case("%%w%ww%", `["%w" %w "w" %]`)
	test_token_case("%!w", `[%! "w"]`)
	test_token_case("$!w", `[$! "w"]`)
	test_token_case("%{Foo%%}", `[% "v" { "Foo%" }]`)
	test_token_case("${Foo%%}", `[$ "v" { "Foo%" }]`)
}
