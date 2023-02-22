package errorsWithData

import (
	"errors"
	"strings"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
)

func TestTokenizer(t *testing.T) {

	test_token_case := func(s string, expected string) {
		tokenized := tokenizeInterpolationString(s)
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
	test_parse_case(`%w%w`, `AST([%w,%w])`)
	test_parse_case(`%w`, `AST(%w)`)
	test_parse_case(`ABC%wDEF`, `AST(["ABC",%w,"DEF"])`)
	test_parse_case(`%{\$Foo}`, `AST(%v{$Foo})`)
	test_parse_case(`a$!C{DEF}`, `AST(["a",$!C{"DEF"}])`)
	test_parse_case(`a%!C1{%!C2{a$w}}`, `AST(["a",%!C1{%!C2{["a",$w]}}])`)
}

type dummy_interpolatableError struct {
	f func(paramMap) string
	error
}

func (d *dummy_interpolatableError) Error_interpolate(p paramMap) string {
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

	var p_direct paramMap = map[string]any{"Direct": 1}
	var p_passed paramMap = map[string]any{"Passed": 1}
	var emptyMap paramMap = make(paramMap)

	// checks whether VerifyParameters_passed reports an error
	testVerifyParametersPassed := func(s string, params_direct paramMap, params_passed paramMap, _baseError error, expectedGood bool) {
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
	testVerifyParametersDirect := func(s string, params_direct paramMap, _baseError error, expectedGood bool) {
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
		if ParamDirectCheck != nil {
			testVerifyParametersPassed(s, params_direct, emptyMap, _baseError, false)
			testVerifyParametersPassed(s, params_direct, p_direct, _baseError, false)
			testVerifyParametersPassed(s, params_direct, p_passed, _baseError, false)
		}

	}

	testSyntaxCheck := func(s string, expectedGood bool) {
		tokens := tokenizeInterpolationString(s)
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
	testSyntaxCheck(`abc$fmt{map}`, true)
	testSyntaxCheck(`abc%fmt{params}`, true)
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

	testVerifyParametersPassed("$!m=0{%{NonExistent}}", p_direct, p_passed, nil, true)
	testVerifyParametersPassed("$!m=0{%{NonExistent}}", p_direct, emptyMap, nil, false)
	testVerifyParametersPassed("$!m>0{%{NonExistent}}", p_direct, p_passed, nil, false)
	testVerifyParametersPassed("$!m>0{%{NonExistent}}", p_direct, emptyMap, nil, true)
}

func TestInterpolation(t *testing.T) {
	var p_direct paramMap = paramMap{"ValHundreds": 128, "StringABC": "abc"}
	var p_passed paramMap = paramMap{"ValHundreds": 256, "StringDEF": "def"}
	var emptyMap paramMap = paramMap{}

	errPlain := errors.New("BASE")
	errBase := &dummy_interpolatableError{error: errPlain, f: func(p paramMap) string {
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
		if err = tree.VerifySyntax(); err != nil {
			t.Fatalf("Unexpected error during syntax check for input %s\n%v", inputString, err)
		}
		if err = tree.VerifyParameters_direct(p_direct, errBase); err != nil {
			t.Fatalf("Unexpected error during direct param check for input %s\n%v", inputString, err)
		}
		if err = tree.VerifyParameters_passed(p_direct, p_passed, errBase); err != nil {
			t.Fatalf("Unexpected error during passed param check for input %s\n%v", inputString, err)
		}
		var builder strings.Builder
		if err = tree.Interpolate(p_direct, p_passed, errBase, &builder); err != nil {
			t.Fatalf("Unexpected error during actual interpolation for input %s\n%v", inputString, err)
		}
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
