package errorsWithData

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
)

// validateTokenList checks for the validity conditions on [tokenList] (as described in its doc) and bails out with t.Fatal on error.
func validateTokenList(t *testing.T, tokenized tokenList) {
	var previousTokenWasString bool
	for i, token := range tokenized {
		testutils.FatalUnless(t, token != tokenInvalid, "tokenized string contains invalid special token")
		if i == 0 {
			special, ok := token.(specialToken)
			testutils.FatalUnless(t, ok && special == tokenStart, "tokenized string did not start with tokenStart")
			continue
		}
		if i == len(tokenized)-1 {
			special, ok := token.(specialToken)
			testutils.FatalUnless(t, ok && special == tokenEnd, "tokenized string did not end with tokenEnd")
			continue
		}
		s, tokenIsString := token.(stringToken)
		if tokenIsString {
			testutils.FatalUnless(t, !previousTokenWasString, "tokenized string contain consecutive string tokens")
			testutils.FatalUnless(t, len(s) > 0, "tokenized string contains empty string")
			testutils.FatalUnless(t, utf8.ValidString(string(s)), "Invalid UTF-8 in token string")
		} else {
			special, ok := token.(specialToken)
			testutils.FatalUnless(t, ok, "tokenized string was neither a special token or a string token")
			testutils.FatalUnless(t, special != tokenStart, "tokenized string contained tokenStart at non-starting position %v", i)
			testutils.FatalUnless(t, special != tokenEnd, "tokenized string contained tokenEnd at non-ending position %v", i)
		}
		previousTokenWasString = tokenIsString
	}
}

func TestTokenizer(t *testing.T) {

	test_token_case := func(s string, expected string) {
		tokenized := tokenizeInterpolationString(s)
		validateTokenList(t, tokenized)
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
	test_token_case(`%%%`, `[ "%" % ]`)
	test_token_case("a\xc0\xffb", "[ \"a\uFFFDb\" ]") // unicode replacement character
	// test_token_case("$%%x{%{")

}

// Check that String() outputs a string that, if tokenized, recovers the token.

func TestSpecialTokenToString(t *testing.T) {
	for _, token := range allSpecialTokens {
		_ = token.String()
	}
	for _, token := range allStringExpressibleSpecialTokens {
		s := token.String()
		tokenized := tokenizeInterpolationString(s)
		validateTokenList(t, tokenized)
		testutils.FatalUnless(t, len(tokenized) == 3, "")
		testutils.FatalUnless(t, tokenized[1] == token, "")
		// tokenized[0] == tokenStart and tokenized[2] == tokenEnd are checked by validateTokenList
	}
}

/**
 * Tests for individual ast types.
 * All these are internal tests for unexported types and methods.
 */

// Create a list containing each type that is intended to satisfy the interface.
// Note that this also ensures (by failing to compile otherwise) that the intended types actually satisfy the interfaces
// These are used to consolidate tests

func new_asts_with_children() [3]ast_with_children {
	return [3]ast_with_children{new_ast_root(), new_ast_condDollar(), new_ast_condPercent()}
}

func new_asts_fmt() [2]ast_fmt {
	return [2]ast_fmt{new_ast_fmtDollar(), new_ast_fmtPercent()}
}

func new_asts_cond() [2]ast_cond {
	return [2]ast_cond{new_ast_condDollar(), new_ast_condPercent()}
}

// TestASTWithChildren tests that all ast types satisfying [ast_with_children] ([ast_root], [ast_condDollar], [ast_condPercent])
// satisfy the contracts of that interface (beyond ast_I)
func TestASTWithChildren(t *testing.T) {
	L := new_asts_with_children()
	for _, ast := range L {
		ast.set_child_list(nil)
		var astCopy ast_with_children = ast
		newList := new_ast_list()
		ast.set_child_list(newList)
		testutils.FatalUnless(t, ast.get_children() == newList, "Did not get back list")
		testutils.FatalUnless(t, astCopy.get_children() == newList, "Assignment not shallow")
		ast.simplify()
		testutils.FatalUnless(t, ast.get_children() == newList, "Simplify changed empty list")
		var ast_c1 ast_I = new_ast_list() // not important that it's a list, but we want sth. with state.
		var ast_c2 ast_I = new_ast_list() // not important that it's a list, but we want sth. with state.
		newList.append_ast(ast_c1)
		newList.append_ast(ast_c2)
		testutils.FatalUnless(t, ast.get_children() == newList, "Modifying child list breaks link")
		ast.simplify()
		testutils.FatalUnless(t, ast.get_children() == newList, "Simplify changed 2-element list")
		newList.remove_last()
		testutils.FatalUnless(t, ast.get_children() == newList, "Modifying child list breaks link (2)")
		ast.simplify()
		testutils.FatalUnless(t, ast.get_children() == ast_c1, "simplfy did not work")
		testutils.FatalUnless(t, astCopy.get_children() == ast_c1, "Assignment not shallow (2)")
	}
}

// TestASTFmt tests that all ast types satisfying [ast_fmt], i.e. [ast_fmtPercent] and [ast_fmtDollar]
// satisfy the contracts of that interface (beyond ast_I)
func TestASTFmt(t *testing.T) {
	L := new_asts_fmt()
	for _, ast := range L {
		testutils.FatalUnless(t, ast.get_formatString() == "" || ast.get_formatString() == "v", "initial fmt string non-empty or \"v\"") // currently, it is "", but "v" might be a valid and meaningful implementation.
		testutils.FatalUnless(t, ast.get_variableName() == "", "initial variable name non-empty")
		var astCopy ast_fmt = ast
		ast.set_variableName("var")
		ast.set_formatString("fmt")
		testutils.FatalUnless(t, ast.get_variableName() == "var", "setter/getter failure for variable name")
		testutils.FatalUnless(t, ast.get_formatString() == "fmt", "setter/getter failure for format string")
		testutils.FatalUnless(t, astCopy.get_variableName() == "var", "setter/getter not shallow for variable name")
		testutils.FatalUnless(t, astCopy.get_formatString() == "fmt", "setter/getter not shallow for format string")
		tok := ast.token()
		switch ast.(type) {
		case ast_fmtDollar:
			testutils.FatalUnless(t, tok == `$`, "")
		case ast_fmtPercent:
			testutils.FatalUnless(t, tok == `%`, "")
		default:
			t.Fatalf("unexpected ast_fmt type %T", ast)
		}
	}
}

// TestASTCond test that all ast types satisfying [ast_cond], i.e. [ast_condPercent] and [ast_condDollar]
// satisfy the contracts of that interface (beyond ast_I)
func TestASTCond(t *testing.T) {
	L := new_asts_cond()
	for _, ast := range L {
		testutils.FatalUnless(t, ast.is_valid(), "fresh cond is invalid")
		testutils.FatalUnless(t, ast.get_condition() == "", "fresh condition ast has non-empty condition %v", ast.get_condition())
		var astCopy ast_cond = ast
		ast.set_condition("cond")
		testutils.FatalUnless(t, ast.get_condition() == "cond", "setter/getter failure for cond")
		testutils.FatalUnless(t, astCopy.get_condition() == "cond", "shallowness failure for cond")
		tok := ast.token()
		switch ast.(type) {
		case ast_condDollar:
			testutils.FatalUnless(t, tok == `$!`, "")
		case ast_condPercent:
			testutils.FatalUnless(t, tok == `%!`, "")
		default:
			t.Fatalf("Unexpected ast_cond type %T", ast)
		}
		ast.make_invalid(1)
		testutils.FatalUnless(t, !ast.is_valid(), "")
		testutils.FatalUnless(t, !astCopy.is_valid(), "")
		ast.make_invalid(3)
		testutils.FatalUnless(t, !ast.is_valid(), "")
		testutils.FatalUnless(t, !astCopy.is_valid(), "")
	}
}

// Parser check for all valid(!) parse cases.
// Note that this does not call HandleSyntaxConditions, which would detect errors due to invalid conditions.

func TestParserValidCases(t *testing.T) {
	test_parse_case := func(s string, expected string) {
		tokenized := tokenizeInterpolationString(s)
		parse_result, err := make_ast(tokenized)
		ast_as_string := parse_result.String()
		if err != nil {
			t.Fatalf("Parsing error when processing input %v, tokenized as %v\n Built-up ast was %v\n Error was %v\n", s, tokenized, ast_as_string, err)
		}

		testutils.FatalUnless(t, ast_as_string == expected, "parser did not get expected result for input string \"%s\".\nGot: %s. Expected:%s\n", s, ast_as_string, expected)
	}
	test_parse_case(``, `AST([])`)
	test_parse_case(`ABC`, `AST("ABC")`)
	test_parse_case(`a\\b`, `AST("a\b")`)
	test_parse_case(`%w`, `AST(%w)`)
	test_parse_case(`$w`, `AST($w)`)
	test_parse_case(`%fmt{Var}`, `AST(%fmt{Var})`)
	test_parse_case(`$fmt{Var}`, `AST($fmt{Var})`)
	test_parse_case(`%!cond{}`, `AST(%!cond{[]})`)
	test_parse_case(`$!cond{}`, `AST($!cond{[]})`)
	test_parse_case(`ABC%wDEF`, `AST(["ABC",%w,"DEF"])`)
	test_parse_case(`%{\$Foo}`, `AST(%v{$Foo})`)
	test_parse_case(`%w%w`, `AST([%w,%w])`)

	test_parse_case(`a$!C{DEF}`, `AST(["a",$!C{"DEF"}])`)
	test_parse_case(`a%!C1{%!C2{a$w}}`, `AST(["a",%!C1{%!C2{["a",$w]}}])`)
	test_parse_case(`$%%{!x}`, `AST($%{!x})`)
}

func TestMisparses(t *testing.T) {
	const showall_err = true    // set to true to display error messages
	const showall_inband = true // set to true to display in-band error message

	test_misparse_case := func(s string, printError bool, printInBand bool) {
		if printError || printInBand || showall_err || showall_inband {
			fmt.Printf("Output requested for input %v.\n", s)
		}
		tokenized := tokenizeInterpolationString(s)
		parse_result, err := make_ast(tokenized)
		ast_as_string := parse_result.String()
		if printError || showall_err {
			fmt.Printf("\terror given as \"%v\"\n", parse_result.parseError)
		}
		if printInBand || showall_inband {
			var builder strings.Builder
			parse_result.Interpolate(make(ParamMap), nil, nil, &builder)
			fmt.Printf("\tIn-band error would be \"%v\"\n", builder.String())

		}
		testutils.FatalUnless(t, err != nil, "Got nil error when misparse was expected.\nInput string was %v\nast is %v", s, ast_as_string)
		testutils.FatalUnless(t, parse_result.parseError != nil, "Got no error when misparse was expected.\nInput string was %v\nast is %v", s, ast_as_string)
	}

	// check correct tail handling

	test_misparse_case("%!m>0{DONT DISPLAY} $w $!cond{%w $%} TAIL %w $!m>0{%w}", true, true)

	// unexpected tokens in list mode
	test_misparse_case("{", false, false)
	test_misparse_case("}", false, false)
	// trailing stray }
	test_misparse_case("x%v{f}y}", false, false)

	// wrong tokens where format verb or { was expected after % or $
	// (Note: %% is an escape sequence for %, so certain tokens cannot follow after %)
	test_misparse_case("%", false, false)         // missing { after %
	test_misparse_case("%$", false, false)        // $ after %
	test_misparse_case("%}", false, false)        // } instead of fmt
	test_misparse_case("%$w", false, false)       // $w instead of fmt
	test_misparse_case("%$!cond{}", false, false) // condition instead

	test_misparse_case("$", false, false)
	test_misparse_case("$%", false, false)
	test_misparse_case("$$", false, false)
	test_misparse_case("$}", false, false)
	test_misparse_case("$%w", false, false)
	test_misparse_case("$$w", false, false)
	test_misparse_case("$%!cond{}", false, false)
	test_misparse_case("$$!cond{}", false, false)

	// wrong token after format verb
	test_misparse_case("%x%", false, false)
	test_misparse_case("%x$", false, false)
	test_misparse_case("%x%!", false, false)
	test_misparse_case("%x$!", false, false)
	test_misparse_case("%x}", false, false)
	test_misparse_case("%x%w", false, false)
	test_misparse_case("%x$w", false, false)

	test_misparse_case("$x%", false, false)
	test_misparse_case("$x$", false, false)
	test_misparse_case("$x%!", false, false)
	test_misparse_case("$x$!", false, false)
	test_misparse_case("$x}", false, false)
	test_misparse_case("$x%w", false, false)
	test_misparse_case("$x$w", false, false)

	// wrong token when variable name was expected
	test_misparse_case("%x{%w}", false, false)  // %w - expressing in variable name
	test_misparse_case("a%x{$w}", false, false) // $w- expression in variable name
	test_misparse_case("%v{{}}", false, false)  // stray { in variable name
	test_misparse_case("%v{%}", false, false)   // stray % in variable name
	test_misparse_case("%v{$}", false, false)   // stray $ in variable name
	test_misparse_case("%v{%!}", false, false)  // stray %! in variable name
	test_misparse_case("%v{$!}", false, false)  // stray $! in variable name
	test_misparse_case("%{", false, false)      // ends when variable name was expected
	test_misparse_case("%{{", false, false)     // { instead of variable name
	test_misparse_case("%{}", false, false)     // empty variable name after %

	test_misparse_case("$x{%w}", false, false)  // %w - expressing in variable name
	test_misparse_case("a$x{$w}", false, false) // $w- expression in variable name
	test_misparse_case("$v{{}}", false, false)  // stray { in variable name
	test_misparse_case("$v{%}", false, false)   // stray % in variable name
	test_misparse_case("$v{$}", false, false)   // stray $ in variable name
	test_misparse_case("$v{%!}", false, false)  // stray %! in variable name
	test_misparse_case("$v{$!}", false, false)  // stray $! in variable name
	test_misparse_case("${", false, false)      // ends when variable name was expected
	test_misparse_case("${{", false, false)     // { instead of variable name
	test_misparse_case("${}", false, false)     // empty variable name after %

	// wrong token after variable name
	test_misparse_case("%{x", false, false)
	test_misparse_case("%{x{}}", false, false)
	test_misparse_case("%{x%w}", false, false)
	test_misparse_case("%{x$w}", false, false)
	test_misparse_case("%{x%}", false, false)
	test_misparse_case("%{x$}", false, false)
	test_misparse_case("%{x%!cond{}}", false, false)
	test_misparse_case("%{x$!cond{}}", false, false)
	test_misparse_case("%{x{}}", false, false)

	test_misparse_case("${x", false, false)
	test_misparse_case("${x{}}", false, false)
	test_misparse_case("${x%w}", false, false)
	test_misparse_case("${x$w}", false, false)
	test_misparse_case("${x%}", false, false)
	test_misparse_case("${x$}", false, false)
	test_misparse_case("${x%!cond{}}", false, false)
	test_misparse_case("${x$!cond{}}", false, false)
	test_misparse_case("${x{}}", false, false)

	// wrong token when condition string was expected
	test_misparse_case("a %w %!", false, false)
	test_misparse_case("a %w %!{}", false, false)
	test_misparse_case("a %w %!%w{}", false, false)
	test_misparse_case("a %w %!$w{}", false, false)
	test_misparse_case("a %w %!%!cond{}{}", false, false)
	test_misparse_case("a %w %!$!cond{}{}", false, false)
	test_misparse_case("a %w %!%{var}{}", false, false)
	test_misparse_case("a %w %!${var}{}", false, false)
	test_misparse_case("a %w %!}", false, false)

	test_misparse_case("a %w $!", false, false)
	test_misparse_case("a %w $!{}", false, false)
	test_misparse_case("a %w $!%w{}", false, false)
	test_misparse_case("a %w $!$w{}", false, false)
	test_misparse_case("a %w $!%!cond{}{}", false, false)
	test_misparse_case("a %w $!$!cond{}{}", false, false)
	test_misparse_case("a %w $!%{var}{}", false, false)
	test_misparse_case("a %w $!${var}{}", false, false)
	test_misparse_case("a %w $!}", false, false)

	// wrong token after condition string
	test_misparse_case("b %w %!cond%fmt{}{}", false, false)
	test_misparse_case("b %w %!cond$fmt{}{}", false, false)
	test_misparse_case("b %w %!cond", false, false)
	test_misparse_case("b %w %!cond}", false, false)
	test_misparse_case("b %w %!cond%w{}", false, false)
	test_misparse_case("b %w %!cond$w{}", false, false)
	test_misparse_case("b %w %!cond%!cond{}{}", false, false)
	test_misparse_case("b %w %!cond$!cond{}{}", false, false)

	test_misparse_case("b %w $!cond%fmt{}{}", false, false)
	test_misparse_case("b %w $!cond$fmt{}{}", false, false)
	test_misparse_case("b %w $!cond", false, false)
	test_misparse_case("b %w $!cond}", false, false)
	test_misparse_case("b %w $!cond%w{}", false, false)
	test_misparse_case("b %w $!cond$w{}", false, false)
	test_misparse_case("b %w $!cond%!cond{}{}", false, false)
	test_misparse_case("b %w $!cond$!cond{}{}", false, false)

	// check some case of invalid subtrees
	test_misparse_case("c %w %cond{", false, false)       // unterminated
	test_misparse_case("c %w %cond{string", false, false) // unterminated
	test_misparse_case("c %w %cond{{}}", false, false)    // stray { in list mode
	test_misparse_case("c %w %cond{}}", false, false)

	test_misparse_case("c %w $cond{", false, false)       // unterminated
	test_misparse_case("c %w $cond{string", false, false) // unterminated
	test_misparse_case("c %w $cond{{}}", false, false)    // stray { in list mode
	test_misparse_case("c %w $cond{}}", false, false)

}

func TestValidVariableName(t *testing.T) {
	for _, special := range validMapSelectors {
		testutils.FatalUnless(t, len(special) > 0, "validMapSelectors contains empty string")
		testutils.FatalUnless(t, ValidInterpolationName(special) == false, "special variable name %v considered a valid variable name (causing ambiguity)", special)
		testutils.FatalUnless(t, special[0] == specialVariableNameIndicator, "special variable name %v does not start with %v", special, string(specialVariableNameIndicator))
	}
}

// dummy_interpolatableError is a struct satisfying ErrorInterpolater.
//
// It simply embeds an error and a function/closure f. If f is non-nil, f is used for Error_interpolate and called on the ParamMap. This is only used in testing.
type dummy_interpolatableError struct {
	DummyValidator
	f func(ParamMap) string
	error
}

func (d *dummy_interpolatableError) Error_interpolate(p ParamMap) string {
	if d.f == nil {
		return d.error.Error()
	} else {
		return d.f(p)
	}
}

var _ ErrorInterpolater = &dummy_interpolatableError{}

func TestVerifySyntax(t *testing.T) {
	var baseError error = errors.New("some error")
	var baseInterpolatableError ErrorInterpolater = &dummy_interpolatableError{error: baseError, f: nil}

	var p_direct ParamMap = map[string]any{"Direct": 1}
	var p_passed ParamMap = map[string]any{"Passed": 1}
	var emptyMap ParamMap = make(ParamMap)

	// checks whether VerifyParameters_passed reports an error
	testVerifyParametersPassed := func(s string, params_direct ParamMap, params_passed ParamMap, _baseError error, expectedGood bool) {
		tokens := tokenizeInterpolationString(s)
		parsed, errParsing := make_ast(tokens)
		if errParsing != nil {
			t.Fatalf("Unexpected parsing error when processing string %s, %v", s, errParsing)
		}
		CheckPassed := parsed.VerifyParameters_passed(params_direct, params_passed, _baseError)
		if expectedGood {
			if CheckPassed != nil {
				t.Fatalf("Unexpected error processing %s returned by VerifyParameters_passed: %v", s, CheckPassed)
			}
		} else {
			if CheckPassed == nil {
				t.Fatalf("VerifyParameters_passed unexpectedly reported no error on %s", s)
			}
		}
	}

	// checks whether VerifyParameters_direct reports an error
	testVerifyParametersDirect := func(s string, params_direct ParamMap, _baseError error, expectedGood bool) {
		tokens := tokenizeInterpolationString(s)
		parsed, errParsing := make_ast(tokens)
		if errParsing != nil {
			t.Fatalf("Unexpected parsing error when processing string %s, %v", s, errParsing)
		}
		ParamDirectCheck := parsed.VerifyParameters_direct(params_direct, _baseError)
		if expectedGood {
			if ParamDirectCheck != nil {
				t.Fatalf("Unexpected error processing %s returned by VerifyParameters_direct.\n Error was: %v\n params_direct was %v\nbaseError was %v", s, ParamDirectCheck, params_direct, _baseError)
			}
		} else {
			if ParamDirectCheck == nil {
				t.Fatalf("VerifyParameters_direct unexpectedly reported no error on %s", s)
			}
		}
		// ensure that VerifyParameters_passed ALSO fails.
		/*
			if ParamDirectCheck != nil {
				testVerifyParametersPassed(s, params_direct, emptyMap, _baseError, false)
				testVerifyParametersPassed(s, params_direct, p_direct, _baseError, false)
				testVerifyParametersPassed(s, params_direct, p_passed, _baseError, false)
			}
		*/

	}

	testSyntaxCheck := func(s string, expectedGood bool) {
		tokens := tokenizeInterpolationString(s)
		parsed, errParsing := make_ast(tokens)
		if errParsing != nil {
			t.Fatalf("Unexpected parsing error when processing string %s, %v", s, errParsing)
		}
		syntaxCheck := parsed.HandleSyntaxConditions()
		if expectedGood {
			if syntaxCheck != nil {
				t.Fatalf("Unexpected error processing %s returned by HandleSyntaxConditions: %v", s, syntaxCheck)
			}
		} else {
			if syntaxCheck == nil {
				t.Fatalf("HandleSyntaxConditions unexpectedly reported no error on %s", s)
			}
		}
		// If Syntax check fails, make sure that Parameters_direct also fails (which in turns check Parameters_passed as well)
		if syntaxCheck != nil {
			testVerifyParametersDirect(s, emptyMap, nil, false)
			testVerifyParametersDirect(s, p_direct, nil, false)
			testVerifyParametersDirect(s, emptyMap, baseError, false)
			testVerifyParametersDirect(s, p_direct, baseError, false)
			testVerifyParametersDirect(s, emptyMap, baseInterpolatableError, false)
			testVerifyParametersDirect(s, p_direct, baseInterpolatableError, false)
		}

	}

	testVerifyParametersPassed("", emptyMap, emptyMap, nil, true)
	testVerifyParametersDirect("", emptyMap, nil, true)
	testSyntaxCheck("", true)

	testSyntaxCheck("abc", true)
	testSyntaxCheck("abc%w$w%!m=0{Foo}", true)
	testSyntaxCheck("abc%w$w%!n=0{Foo}", false)

	testSyntaxCheck(`abc%\%{V}`, false)
	testSyntaxCheck(`abc$\%{V}`, false)
	testSyntaxCheck(`abc${V}`, true)
	testSyntaxCheck(`abc%{V}`, true)
	testSyntaxCheck(`abc$fmt{V}`, true)
	testSyntaxCheck(`abc%fmt{V}`, true)
	testSyntaxCheck(`abc$fmt{v}`, false)
	testSyntaxCheck(`abc%fmt{v}`, false)
	testSyntaxCheck(`abc$fmt{!map}`, true)
	testSyntaxCheck(`abc%fmt{!params}`, true)
	testSyntaxCheck(`abc%v{Foo.Bar}`, false)

	testSyntaxCheck("%!m>0{Foo}", true)
	testSyntaxCheck("$!m>0{Foo}", true)

	testVerifyParametersDirect("a%wb", emptyMap, nil, false)
	testVerifyParametersDirect("a%wb", emptyMap, baseError, true)
	testVerifyParametersDirect("a%wb", emptyMap, baseInterpolatableError, true)
	testVerifyParametersPassed("abc%wdef", emptyMap, emptyMap, nil, false)
	testVerifyParametersPassed("abc%wdef", emptyMap, emptyMap, baseError, true)
	testVerifyParametersPassed("abc%wdef", emptyMap, emptyMap, baseInterpolatableError, true)

	testVerifyParametersDirect("a$wb", emptyMap, nil, false)
	testVerifyParametersDirect("a$wb", emptyMap, baseError, false)
	testVerifyParametersDirect("a$wb", emptyMap, baseInterpolatableError, true)
	testVerifyParametersPassed("abc$wdef", emptyMap, emptyMap, nil, false)
	testVerifyParametersPassed("abc$wdef", emptyMap, emptyMap, baseError, false)
	testVerifyParametersPassed("abc$wdef", emptyMap, emptyMap, baseInterpolatableError, true)

	testVerifyParametersDirect("%{NonExistent}", emptyMap, nil, false)
	testVerifyParametersDirect("${NonExistent}", emptyMap, nil, true)
	testVerifyParametersPassed("${NonExistent}", emptyMap, emptyMap, nil, false)

	testVerifyParametersPassed("%{Direct}", p_direct, p_passed, nil, true)
	testVerifyParametersPassed("%{Passed}", p_direct, p_passed, nil, false)
	testVerifyParametersPassed("${Direct}", p_direct, p_passed, nil, false)
	testVerifyParametersPassed("${Passed}", p_direct, p_passed, nil, true)

	testVerifyParametersDirect("%{Direct}", p_direct, nil, true)
	testVerifyParametersDirect("%{Passed}", p_direct, nil, false)
	testVerifyParametersDirect("${Direct}", p_direct, nil, true)
	testVerifyParametersDirect("${Passed}", p_direct, nil, true)

	testSyntaxCheck("%!m>0{%{NonExistent}}1", true)
	testVerifyParametersDirect("%!m>0{%{NonExistent}}2", emptyMap, nil, true)
	testVerifyParametersDirect("%!m>0{%{NonExistent}}3", p_direct, nil, false)

	testSyntaxCheck("%!m=0{%{NonExistent}}4", true)
	testVerifyParametersDirect("%!m=0{%{NonExistent}}5", emptyMap, nil, false)
	testVerifyParametersDirect("%!m=0{%{NonExistent}}6", p_direct, nil, true)

	testSyntaxCheck("$!m>0{%{NonExistent}}7", true)
	testVerifyParametersDirect("$!m>0{%{NonExistent}}8", emptyMap, nil, false)
	testVerifyParametersDirect("$!m>0{%{NonExistent}}9", p_direct, nil, false)

	testSyntaxCheck("$!m=0{%{NonExistent}}10", true)
	testVerifyParametersDirect("$!m=0{%{NonExistent}}11", emptyMap, nil, false)
	testVerifyParametersDirect("$!m=0{%{NonExistent}}12", p_direct, nil, false)

	testSyntaxCheck("%!m>0{%{Direct}}13", true)
	testVerifyParametersDirect("%!m>0{%{Direct}}14", emptyMap, nil, true)
	testVerifyParametersDirect("%!m>0{%{Direct}}15", p_direct, nil, true)

	testSyntaxCheck("%!m=0{%{Direct}}16", true)
	testVerifyParametersDirect("%!m=0{%{Direct}}17", emptyMap, nil, false)
	testVerifyParametersDirect("%!m=0{%{Direct}}18", p_direct, nil, true)

	testSyntaxCheck("$!m>0{%{Direct}}19", true)
	testVerifyParametersDirect("$!m>0{%{Direct}}20", emptyMap, nil, false)
	testVerifyParametersDirect("$!m>0{%{Direct}}21", p_direct, nil, true)

	testSyntaxCheck("$!m=0{%{Direct}}22", true)
	testVerifyParametersDirect("$!m=0{%{Direct}}23", emptyMap, nil, false)
	testVerifyParametersDirect("$!m=0{%{Direct}}24", p_direct, nil, true)

	testVerifyParametersPassed("$!m=0{%{NonExistent}}25", p_direct, p_passed, nil, true)
	testVerifyParametersPassed("$!m=0{%{NonExistent}}26", p_direct, emptyMap, nil, false)
	testVerifyParametersPassed("$!m>0{%{NonExistent}}27", p_direct, p_passed, nil, false)
	testVerifyParametersPassed("$!m>0{%{NonExistent}}28", p_direct, emptyMap, nil, true)
}

func TestInterpolation(t *testing.T) {
	var p_direct ParamMap = ParamMap{"ValHundreds": 128, "StringABC": "abc"}
	var p_passed ParamMap = ParamMap{"ValHundreds": 256, "StringDEF": "def"}
	var emptyMap ParamMap = ParamMap{}

	errPlain := errors.New("BASE")
	errBase := &dummy_interpolatableError{error: errPlain, f: func(p ParamMap) string {
		r := p["StringDEF"]
		if r == "def" {
			return "OK"
		} else {
			return "NOTOK"
		}
	}}

	// errBase is an error whose error message depends on parameters in the following way:
	testutils.Assert(errBase.Error() == "BASE")
	testutils.Assert(errBase.Error_interpolate(emptyMap) == "NOTOK")
	testutils.Assert(errBase.Error_interpolate(p_direct) == "NOTOK")
	testutils.Assert(errBase.Error_interpolate(p_passed) == "OK")

	// We now check input -> output pairs for fixed parameter map p_direct, p_passed and errBase as base error.
	// This function performs the check for a given input/output pair.
	testInterpolation := func(inputString string, expectedOutput string) {
		tokens := tokenizeInterpolationString(inputString)
		tree, err := make_ast(tokens)
		if err != nil {
			t.Fatalf("Unexpected error parsing input from %s\nReported error was:\n%v", inputString, err)
		}
		if err = tree.HandleSyntaxConditions(); err != nil {
			t.Fatalf("Unexpected error during syntax check for input %s\n%v", inputString, err)
		}
		if err = tree.VerifyParameters_direct(p_direct, errBase); err != nil {
			t.Fatalf("Unexpected error during direct param check for input %s\n%v", inputString, err)
		}
		if err = tree.VerifyParameters_passed(p_direct, p_passed, errBase); err != nil {
			t.Fatalf("Unexpected error during passed param check for input %s\n%v", inputString, err)
		}
		var builder strings.Builder
		tree.Interpolate(p_direct, p_passed, errBase, &builder)
		interpolatedString := builder.String()
		if interpolatedString != expectedOutput {
			t.Fatalf("Unexpected output from string interpolation. Expected\n%s\nGot\n%s", expectedOutput, interpolatedString)
		}
	}

	testInterpolation("", "")
	s := "Some string with a linebreak\nin between "
	testInterpolation(s, s)
	testInterpolation(`abc\\def`, `abc\def`) // escaped \\ (this is a raw string, the escaping is for our interpolation language)
	testInterpolation(`abc\%def`, `abc%def`)
	testInterpolation(`abc\$def`, `abc$def`)
	testInterpolation(`abc\{def`, `abc{def`)
	testInterpolation(`abc\}def`, `abc}def`)
	testInterpolation("%{ValHundreds}", "128")
	testInterpolation("${ValHundreds}", "256")
	testInterpolation("0b%b{ValHundreds}", "0b10000000")  // binary output
	testInterpolation("0b$b{ValHundreds}", "0b100000000") // binary output
	testInterpolation("Refer to base: %w", "Refer to base: BASE")
	testInterpolation("Derived: $w", "Derived: OK")
	testInterpolation("%!m=0{Foo}", "")
	testInterpolation("%!m>0{Bar}", "Bar")
	testInterpolation("$!m=0{%{ValHundreds}}", "")
	testInterpolation("$!m>0{%{ValHundreds}}", "128")
}

func PrintInterpolationWrong(t *testing.T, s string, parseError bool) {
	var p_direct ParamMap = ParamMap{"ValHundreds": 128, "StringABC": "abc"}
	var p_passed ParamMap = ParamMap{"ValHundreds": 256, "StringDEF": "def"}
	errPlain := errors.New("BASE")
	errBase := &dummy_interpolatableError{error: errPlain, f: func(p ParamMap) string {
		r := p["StringDEF"]
		if r == "def" {
			return "OK"
		} else {
			return "NOTOK"
		}
	}}

	tokens := tokenizeInterpolationString(s)
	tree, err := make_ast(tokens)
	testutils.FatalUnless(t, parseError == (err != nil), "\nInput string:%s\n, tokenized as:%v\nParsed as:%v\nWas parse error expected: %t\nGot: %v\n", s, tokens, tree, parseError, err)
	if parseError {
		testutils.FatalUnless(t, err == tree.HandleSyntaxConditions(), "")
		testutils.FatalUnless(t, err == tree.VerifyParameters_direct(p_direct, errBase), "")
		testutils.FatalUnless(t, err == tree.VerifyParameters_passed(p_direct, p_passed, errBase), "")
	}
	var b strings.Builder
	tree.Interpolate(p_direct, p_passed, errBase, &b)
	output := b.String()
	fmt.Println(output)
}

/*
func TestPrintSomeOutput(t *testing.T) {

	PrintInterpolationWrong(t, "Invalid", false)
	PrintInterpolationWrong(t, "%%%", true)
	PrintInterpolationWrong(t, "Fine1 %${{}{}} FineEnd", true)
	PrintInterpolationWrong(t, "Fine2 %${{{}{}} FineEnd", true)
	PrintInterpolationWrong(t, "Fine3 $$!} FineEnd", true)
	PrintInterpolationWrong(t, "Fine4 #fmt{Foo} FineEnd", true)

	PrintInterpolationWrong(t, "Fine5 %!", true)
	PrintInterpolationWrong(t, "Fine6 %!{", true)
	PrintInterpolationWrong(t, "Fine7 %!m=0", true)
	PrintInterpolationWrong(t, "Fine8 %!m=0{", true)
	PrintInterpolationWrong(t, "Fine9 %!m=0{Bar", true)

	PrintInterpolationWrong(t, "Fine10 $!", true)
	PrintInterpolationWrong(t, "Fine11 $!{", true)
	PrintInterpolationWrong(t, "Fine12 $!m=0", true)
	PrintInterpolationWrong(t, "Fine13 $!m=0{", true)
	PrintInterpolationWrong(t, "Fine14 $!m=0{Bar", true)

	PrintInterpolationWrong(t, "Fine15 %", true)
	PrintInterpolationWrong(t, "Fine16 %fmt", true)
	PrintInterpolationWrong(t, "Fine17 %fmt{", true)
	PrintInterpolationWrong(t, "Fine18 %{", true)
	PrintInterpolationWrong(t, "Fine19 %fmt{Var", true)
	PrintInterpolationWrong(t, "Fine20 %{Var", true)

	PrintInterpolationWrong(t, "Fine21 $", true)
	PrintInterpolationWrong(t, "Fine22 $fmt", true)
	PrintInterpolationWrong(t, "Fine23 $fmt{", true)
	PrintInterpolationWrong(t, "Fine24 ${", true)
	PrintInterpolationWrong(t, "Fine25 $fmt{Var", true)
	PrintInterpolationWrong(t, "Fine26 ${Var", true)

	PrintInterpolationWrong(t, "Fine27 %!m=0{Foo}}", true)

}
*/
