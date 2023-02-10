package errorsWithData

import (
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
)

// Test % and $ - escaping
func TestEscaper(t *testing.T) {
	var emptyMap map[string]any = make(map[string]any)
	for _, s := range []string{"", "abc", "$", "%", "$$", "ab%c%%$$%$d%#%$$asd"} {
		escaped := escaper.Replace(s)
		retrieve1, err1 := formatError_new(escaped, emptyMap, emptyMap, nil, full_eval)
		retrieve2, err2 := formatError_new(escaped, emptyMap, emptyMap, nil, partial_eval)
		testutils.FatalUnless(t, err1 == nil, "failed to retrieve string after escaping (full eval): %s was escaped as %s. Eval resulted in error %v", s, escaped, err1)
		testutils.FatalUnless(t, err2 == nil, "failed to retrieve string after escaping (partial eval): %s was escaped as %s. Eval resulted in error %v", s, escaped, err2)
		testutils.FatalUnless(t, retrieve1 == s, "failed to retrieve string after escaping (full eval): %s escaped as %s and retrieved as %s", s, escaped, retrieve1)
		testutils.FatalUnless(t, retrieve1 == s, "failed to retrieve string after escaping (partial eval): %s escaped as %s and retrieved as %s", s, escaped, retrieve2)
	}
}

func expectGood(t *testing.T, formatString string, paramters_own map[string]any, parameters_passed map[string]any, baseError error, expectedString string) {
	res1, err1 := formatError_new(formatString, paramters_own, parameters_passed, baseError, full_eval)
	testutils.FatalUnless(t, err1 == nil, "Error formatting: formatting %s gave result %s and error %v", formatString, res1, err1)
	testutils.FatalUnless(t, res1 == expectedString, "Error formatting: Formatting %s gave result %s, expected %s", formatString, res1, err1)
	resPartial, errPartial := formatError_new(formatString, paramters_own, parameters_passed, baseError, partial_eval)
	testutils.FatalUnless(t, errPartial == nil, "Error formatting: partially formatting %s gave result %s and error %v", formatString, resPartial, errPartial)
	res2, err2 := formatError_new(resPartial, paramters_own, parameters_passed, baseError, full_eval)
	testutils.FatalUnless(t, err2 == nil, "Error formatting: partially-then full formatting %s gave result %s and error %v.\n Intermediate result %s", formatString, res2, err2, resPartial)
	testutils.FatalUnless(t, res2 == expectedString, "Error formatting: partial-then full formatting %s gave result %s, expected %s.\n Intermediate result was %s", formatString, res2, expectedString, resPartial)
	res3, err3 := formatError_new(formatString, paramters_own, parameters_passed, baseError, parse_check)
	testutils.FatalUnless(t, res3 == "", "parse check gave non-empty result %s", res3)
	testutils.FatalUnless(t, err3 == nil, "Parse check failed for %s with error %v", formatString, err3)
	res4, err4 := formatError_new(formatString, paramters_own, parameters_passed, baseError, param_check)
	testutils.FatalUnless(t, res4 == "", "param check gave non-empty result %s", res4)
	testutils.FatalUnless(t, err4 == nil, "Param check failed for %s with error %v", formatString, err4)
}

func TestFormatter(t *testing.T) {
	var emptyMap map[string]any = make(map[string]any)
	var oneElementMap map[string]any = map[string]any{"Foo": "bar"}
	var numMap map[string]any = map[string]any{"X": 16}
	expectGood(t, "", emptyMap, emptyMap, nil, "")
	expectGood(t, "a%%b", emptyMap, emptyMap, nil, "a%b")
	expectGood(t, "a%$b", emptyMap, emptyMap, nil, "a$b")

	expectGood(t, "abc%!M>0{d%%ef}ghj", emptyMap, emptyMap, nil, "abcghj")
	expectGood(t, "abc%!M>0{d%%ef}ghj", oneElementMap, emptyMap, nil, "abcd%efghj")
	expectGood(t, "abc%!M>0{d%%ef}ghj", emptyMap, oneElementMap, nil, "abcghj")

	expectGood(t, "abc$!M>0{d%%ef}ghj", emptyMap, emptyMap, nil, "abcghj")
	expectGood(t, "abc$!M>0{d%%ef}ghj", oneElementMap, emptyMap, nil, "abcghj")
	expectGood(t, "abc$!M>0{d%%ef}ghj", emptyMap, oneElementMap, nil, "abcd%efghj")

	expectGood(t, "Some %{Foo} test", oneElementMap, emptyMap, nil, "Some bar test")
	expectGood(t, "Some ${Foo} test", emptyMap, oneElementMap, nil, "Some bar test")

	expectGood(t, "%{X}", numMap, emptyMap, nil, "16")
	expectGood(t, "%v{X}", numMap, emptyMap, nil, "16")
	expectGood(t, "%b{X}", numMap, emptyMap, nil, "10000") // binary

}
