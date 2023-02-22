package errorsWithData

import (
	"errors"
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

type dummy_interpolatableError struct {
	error
}

func (d *dummy_interpolatableError) Error_interpolate(paramMap) string {
	return d.error.Error()
}

var _ ErrorInterpolater = &dummy_interpolatableError{}

func TestVerifySyntax(t *testing.T) {
	var baseError error = errors.New("some error")
	var baseInterpolatableError ErrorInterpolater = &dummy_interpolatableError{error: baseError}

	var p_direct paramMap = map[string]any{"Direct": 1}
	var p_passed paramMap = map[string]any{"Passed": 1}
	var emptyMap paramMap = make(paramMap)

	// checks whether VerifyParameters_passed reports an error
	testVerifyParametersPassed := func(s string, params_direct paramMap, params_passed paramMap, _baseError error, expectedGood bool) {
		tokens := tokenizeFormatString(s)
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
	testVerifyParametersDirect := func(s string, params_direct paramMap, _baseError error, expectedGood bool) {
		tokens := tokenizeFormatString(s)
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
		if ParamDirectCheck != nil {
			testVerifyParametersPassed(s, params_direct, emptyMap, _baseError, false)
			testVerifyParametersPassed(s, params_direct, p_direct, _baseError, false)
			testVerifyParametersPassed(s, params_direct, p_passed, _baseError, false)
		}

	}

	testSyntaxCheck := func(s string, expectedGood bool) {
		tokens := tokenizeFormatString(s)
		parsed, errParsing := make_ast(tokens)
		if errParsing != nil {
			t.Fatalf("Unexpected parsing error when processing string %s, %v", s, errParsing)
		}
		syntaxCheck := parsed.VerifySyntax()
		if expectedGood {
			if syntaxCheck != nil {
				t.Fatalf("Unexpected error processing %s returned by VerifySyntax: %v", s, syntaxCheck)
			}
		} else {
			if syntaxCheck == nil {
				t.Fatalf("VerifySyntax unexpectedly reported no error on %s", s)
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
	testSyntaxCheck(`abc$fmt{map}`, true)
	testSyntaxCheck(`abc%fmt{params}`, true)
	testSyntaxCheck(`abc%v{Foo.Bar}`, false)

	testSyntaxCheck("%!m>0{Foo}", true)
	testSyntaxCheck("$!m>0{Foo}", true)

	testVerifyParametersDirect("a%wb", emptyMap, nil, false)
	testVerifyParametersDirect("a%wb", emptyMap, baseError, true)
	testVerifyParametersDirect("a%wb", emptyMap, baseInterpolatableError, true)

	testVerifyParametersDirect("a$wb", emptyMap, nil, false)
	testVerifyParametersDirect("a$wb", emptyMap, baseError, false)
	testVerifyParametersDirect("a$wb", emptyMap, baseInterpolatableError, true)

	testSyntaxCheck("%!m>0{%{NonExistent}}", true)
	testVerifyParametersDirect("%!m>0{%{NonExistent}}", emptyMap, nil, true)
	testVerifyParametersDirect("%!m>0{%{NonExistent}}", p_direct, nil, false)

	testSyntaxCheck("%!m=0{%{NonExistent}}", true)
	testVerifyParametersDirect("%!m=0{%{NonExistent}}", emptyMap, nil, false)
	testVerifyParametersDirect("%!m=0{%{NonExistent}}", p_direct, nil, true)

	testSyntaxCheck("$!m>0{%{NonExistent}}", true)
	testVerifyParametersDirect("$!m>0{%{NonExistent}}", emptyMap, nil, true)
	testVerifyParametersDirect("$!m>0{%{NonExistent}}", p_direct, nil, true)

	testSyntaxCheck("$!m=0{%{NonExistent}}", true)
	testVerifyParametersDirect("$!m=0{%{NonExistent}}", emptyMap, nil, true)
	testVerifyParametersDirect("$!m=0{%{NonExistent}}", p_direct, nil, true)

	testSyntaxCheck("%!m>0{%{Direct}}", true)
	testVerifyParametersDirect("%!m>0{%{Direct}}", emptyMap, nil, true)
	testVerifyParametersDirect("%!m>0{%{Direct}}", p_direct, nil, true)

	testSyntaxCheck("%!m=0{%{Direct}}", true)
	testVerifyParametersDirect("%!m=0{%{Direct}}", emptyMap, nil, false)
	testVerifyParametersDirect("%!m=0{%{Direct}}", p_direct, nil, true)

	testSyntaxCheck("$!m>0{%{Direct}}", true)
	testVerifyParametersDirect("$!m>0{%{Direct}}", emptyMap, nil, true)
	testVerifyParametersDirect("$!m>0{%{Direct}}", p_direct, nil, true)

	testSyntaxCheck("$!m=0{%{Direct}}", true)
	testVerifyParametersDirect("$!m=0{%{Direct}}", emptyMap, nil, true)
	testVerifyParametersDirect("$!m=0{%{Direct}}", p_direct, nil, true)

}
