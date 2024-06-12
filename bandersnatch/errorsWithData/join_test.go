package errorsWithData

import (
	"errors"
	"slices"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
)

func TestNonNilUnion(t *testing.T) {
	var res []error
	testCase := func(expectedResult []error) {
		if len(res) == 0 {
			testutils.FatalUnless(t, res == nil, "res is an empty slice, not nil slice")
		}
		testutils.FatalUnless(t, slices.Equal(res, expectedResult), "%v != %v", res, expectedResult)
	}
	err1 := errors.New("1")
	err2 := errors.New("2")
	err3 := errors.New("3")
	errSliceEmpty := []error{}
	errSliceNil := []error(nil)
	errSliceWithNil := []error{nil}
	errSlice1 := []error{err1}
	errSlice23 := []error{err2, err3}

	res = nonNilUnion(err1, err2)
	testCase([]error{err1, err2})
	res = nonNilUnion()
	testCase(nil)
	res = nonNilUnion(errSliceEmpty)
	testCase(nil)
	res = nonNilUnion(nil...)
	testCase(nil)
	res = nonNilUnion(errSliceNil)
	testCase(nil)
	res = nonNilUnion(err1, errSliceNil)
	testCase([]error{err1})
	res = nonNilUnion(errSlice1, errSlice23, err1)
	testCase([]error{err1, err2, err3, err1})
	res = nonNilUnion(errSliceWithNil)
	testCase(nil)
	res = nonNilUnion(nil, err1)
	testCase([]error{err1})

	didPanic := testutils.CheckPanic(func() { nonNilUnion(1) })
	testutils.FatalUnless(t, didPanic == true, "")
	didPanic = testutils.CheckPanic(func() { nonNilUnion(err1, [1]error{err1}) }) // array rather than slice
	testutils.FatalUnless(t, didPanic == true, "")
	didPanic = testutils.CheckPanic(func() { nonNilUnion([]int{}) })
	testutils.FatalUnless(t, didPanic == true, "")
	didPanic = testutils.CheckPanic(func() { nonNilUnion(new([]error)) }) // pointer-to-slice rather than slice
	testutils.FatalUnless(t, didPanic == true, "")
}

var _ ErrorWithData[struct{}] = &joinedErrors[struct{}]{}
var _ ErrorWithData_any = &joinedErrors_any{}

func TestExtractNonNilErrors(t *testing.T) {
	err1 := errors.New("A")
	err2 := errors.New("B")
	errArray1 := [2]error{err1, err2}
	errArray2 := [3]error{nil, err2, nil}
	errArrayEmpty := [0]error{}
	errSlice1 := []error{err1, err2}
	errSlice2 := []error{nil, err2, nil}
	errSliceEmpty := []error{}
	errSliceNil := []error(nil)
	type errorSlice []error

	testCase := func(x any, expected []error, expectSuccess bool) {
		var target []error = nil
		succ := extractNonNilErrors(&target, x)
		testutils.FatalUnless(t, expectSuccess == (succ == nil), "")
		if len(expected) == 0 {
			testutils.FatalUnless(t, target == nil, "%v not nil for %v", target, x)
		} else {
			testutils.FatalUnless(t, slices.Equal(target, expected), "For %v, got %v instead of %v", x, target, expected)
		}

		target = make([]error, 0)
		succ = extractNonNilErrors(&target, x)
		testutils.FatalUnless(t, expectSuccess == (succ == nil), "")
		testutils.FatalUnless(t, slices.Equal(target, expected), "For %v, got %v instead of %v", x, target, expected)

		errTemp := errors.New("X")
		expectTemp := []error{errTemp}
		expectTemp = append(expectTemp, expected...)

		target = []error{errTemp}
		succ = extractNonNilErrors(&target, x)
		testutils.FatalUnless(t, expectSuccess == (succ == nil), "")
		testutils.FatalUnless(t, slices.Equal(target, expectTemp), "For %v, got %v instead of %v", x, target, expectTemp)
	}

	testCase(err1, []error{err1}, true)
	testCase(err1, []error{err1}, true)
	testCase(errArray1, []error{err1, err2}, true)
	testCase(errArray2, []error{err2}, true)
	testCase(&errArray1, []error{err1, err2}, true)
	testCase(&errArray2, []error{err2}, true)
	testCase(errArrayEmpty, []error{}, true)
	testCase(&errArrayEmpty, []error{}, true)
	testCase(errSlice1, []error{err1, err2}, true)
	testCase(&errSlice1, []error{err1, err2}, true)
	testCase(errorSlice(errSlice1), []error{err1, err2}, true)
	errSlice1Masked := errorSlice(errSlice1)
	testCase(&errSlice1Masked, []error{err1, err2}, true)
	testCase(errSlice2, []error{err2}, true)
	testCase(&errSlice2, []error{err2}, true)
	testCase(errSliceEmpty, []error{}, true)
	testCase(&errSliceEmpty, []error{}, true)
	testCase(errSliceNil, []error{}, true)
	testCase(&errSliceNil, []error{}, true)
	testCase(new([]error), []error{}, true)
	testCase(new([1]error), []error{}, true)
	testCase([]int{}, nil, true)
	testCase(&[]int{}, nil, true)
	testCase([0]int{}, nil, true)
	testCase(&[0]int{}, nil, true)

	testCase(1, []error{}, false)
	testCase(new(int), []error{}, false)
	testCase([]any{err1}, []error{err1}, true)
	testCase(&[]any{err1}, []error{err1}, true)
	testCase([1]any{err1}, []error{err1}, true)
	testCase(&[1]any{err1}, []error{err1}, true)
	testCase([]any{err1, 1, err2}, []error{err1}, false)
	testCase(&[]any{err1, 1, err2}, []error{err1}, false)
	testCase([3]any{err1, 1, err2}, []error{err1}, false)
	testCase(&[3]any{err1, 1, err2}, []error{err1}, false)
	testCase((*[]error)(nil), []error{}, true)
	testCase((*[1]error)(nil), []error{}, true)

	didPanic := testutils.CheckPanic(func() { extractNonNilErrors(nil, nil) })
	testutils.FatalUnless(t, didPanic == true, "")
}

// Note: We don't really test the intricacies of parameter passing here very much. We consider this covered by the tests for extractNonNilErrors.

func TestJoinAny(t *testing.T) {
	err1 := errors.New("A")
	err2 := errors.New("B")
	ewd1, _ := NewErrorWithData_any_params(nil, "%v{X}", "X", 2, PanicOnAllMistakes, ErrorUnlessValidFinal)
	ewd2, e := NewErrorWithData_any_params(nil, "{", "X", 3) // syntax error
	testutils.FatalUnless(t, e != nil, "")
	ewd3, _ := NewErrorWithData_any_params(nil, "$v{X}", PanicOnAllMistakes, ErrorUnlessValidBase)
	testutils.FatalUnless(t, ewd3.ValidateError_Final() != nil, "")
	ewd4, _ := NewErrorWithData_any_params(nil, "%{Y}", "X", "X", "Y", "Y", "Z", "Z")

	res, err := Join_any(err1, err2)
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError_any(t, res, "A\nB", nil, []error{err1, err2})
	res, err = NewErrorWithData_any_params(res, "")
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError_any(t, res, "A\nB", nil, []error{err1, err2})

	res, err = Join_any([]error{err1, nil, err2}, nil)
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError_any(t, res, "A\nB", nil, []error{err1, err2})

	res, err = Join_any([3]error{err1, nil, err2}, nil)
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError_any(t, res, "A\nB", nil, []error{err1, err2})

	res, err = Join_any(ewd1, ewd4)
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError_any(t, res, "2\nY", ParamMap{"X": "X", "Y": "Y", "Z": "Z"}, []error{ewd1, ewd4})

	res, err = Join_any(ewd3, ewd1)
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError_any(t, res, "ignore", ParamMap{"X": 2}, []error{ewd1, ewd3})
	testutils.FatalUnless(t, res.Error() != "2\n2", "%v", res.Error())

	// NOTE: Wrapping once changes the message!
	wrappedRes, err := NewErrorWithData_any_params(res, "")
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError_any(t, wrappedRes, "2\n2", ParamMap{"X": 2}, []error{ewd1, ewd3})

	res, err = Join_any(ewd2)
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError_any(t, res, "ignore", ParamMap{"X": 3}, []error{ewd2})
	testutils.FatalUnless(t, res.ValidateSyntax() != nil, "")
	testutils.FatalUnless(t, res.ValidateError_Base() != nil, "")
	testutils.FatalUnless(t, res.ValidateError_Final() != nil, "")

	res, err = Join_any()
	testutils.FatalUnless(t, err == nil, "%v", err)
	testutils.FatalUnless(t, res == nil, "%v", res)

	res, err = Join_any(ewd1, ewd4, MistakeIfDataIsReplaced)
	testutils.FatalUnless(t, err != nil, "")
	testError_any(t, res, "2\nY", ParamMap{"X": "X", "Y": "Y", "Z": "Z"}, []error{ewd1, ewd4})

	didPanic, panicValue := testutils.CheckPanic2(func() { Join_any(ewd1, ewd4, MistakeIfDataIsReplaced, PanicOnAllMistakes) })
	testutils.FatalUnless(t, didPanic == true, "")
	testutils.FatalUnless(t, panicValue.(error).Error() == err.Error(), "")

	res, err = Join_any(ewd1, ewd4, PreferPreviousData, MistakeIfDataIsReplaced)
	testutils.FatalUnless(t, err != nil, "")
	testError_any(t, res, "2\nY", ParamMap{"X": 2, "Y": "Y", "Z": "Z"}, []error{ewd1, ewd4})

	didPanic = testutils.CheckPanic(func() { Join_any(ewd1, 2) })
	testutils.FatalUnless(t, didPanic == true, "")

	didPanic = testutils.CheckPanic(func() { Join_any(ErrorUnlessValidSyntax) }) // flag invalid for Join_any
	testutils.FatalUnless(t, didPanic == true, "")
}

func TestJoin(t *testing.T) {
	type invalid struct{ unexported int }
	type WithX struct{ X int }
	type empty struct{}

	didPanic := testutils.CheckPanic(func() { Join[invalid]() })
	testutils.FatalUnless(t, didPanic == true, "")

	err1 := errors.New("A")
	err2 := errors.New("B")
	ewd1, _ := NewErrorWithData_any_params(nil, "%v{X}", "X", 2, PanicOnAllMistakes, ErrorUnlessValidFinal)
	ewd2, e := NewErrorWithData_any_params(nil, "{", "X", 3) // syntax error
	testutils.FatalUnless(t, e != nil, "")
	ewd3, _ := NewErrorWithData_any_params(nil, "$v{X}", PanicOnAllMistakes, ErrorUnlessValidBase)
	testutils.FatalUnless(t, ewd3.ValidateError_Final() != nil, "")
	ewd4, _ := NewErrorWithData_any_params(nil, "%{Y}", "X", "X", "Y", "Y", "Z", "Z")

	var res ErrorWithData_any

	res, err := Join[WithX](err1, err2)
	testutils.FatalUnless(t, err != nil, "%v", err)
	testError(t, res.(ErrorWithData[WithX]), "A\nB", &WithX{}, ParamMap{"X": 0}, []error{err1, err2})
	res, err = NewErrorWithData_params[WithX](res, "") // add level of wrapping
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError(t, res.(ErrorWithData[WithX]), "A\nB", &WithX{}, ParamMap{"X": 0}, []error{err1, err2})

	res, err = Join[WithX]([]error{err1, nil, err2}, nil, MissingDataAsZero)
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError(t, res.(ErrorWithData[WithX]), "A\nB", &WithX{}, ParamMap{"X": 0}, []error{err1, err2})

	res, err = Join[WithX]([3]error{err1, nil, err2}, nil, MissingDataAsZero)
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError(t, res.(ErrorWithData[WithX]), "A\nB", &WithX{}, ParamMap{"X": 0}, []error{err1, err2})

	res, err = Join[WithX](ewd1, ewd4, MissingDataAsZero)
	testutils.FatalUnless(t, err != nil, "%v", err) // X has wrong type
	testError(t, res.(ErrorWithData[WithX]), "2\nY", &WithX{}, ParamMap{"X": 0, "Y": "Y", "Z": "Z"}, []error{ewd1, ewd4})

	res, err = Join[empty](ewd3, ewd1)
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError(t, res.(ErrorWithData[empty]), "ignore", &empty{}, ParamMap{"X": 2}, []error{ewd1, ewd3})
	testutils.FatalUnless(t, res.Error() != "2\n2", "%v", res.Error())

	// NOTE: Wrapping once changes the message!
	wrappedRes, err := NewErrorWithData_any_params(res, "")
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError_any(t, wrappedRes, "2\n2", ParamMap{"X": 2}, []error{ewd1, ewd3})

	res, err = Join[empty](ewd2)
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError_any(t, res, "ignore", ParamMap{"X": 3}, []error{ewd2})
	testutils.FatalUnless(t, res.ValidateSyntax() != nil, "")
	testutils.FatalUnless(t, res.ValidateError_Base() != nil, "")
	testutils.FatalUnless(t, res.ValidateError_Final() != nil, "")

	res, err = Join[empty]()
	testutils.FatalUnless(t, err == nil, "%v", err)
	testutils.FatalUnless(t, res == nil, "%v", res)

	res, err = Join[WithX](ewd1, ewd4, MistakeIfDataIsReplaced)
	testutils.FatalUnless(t, err != nil, "")
	testError_any(t, res, "2\nY", ParamMap{"X": 0, "Y": "Y", "Z": "Z"}, []error{ewd1, ewd4})

	didPanic, panicValue := testutils.CheckPanic2(func() { Join[WithX](ewd1, ewd4, MistakeIfDataIsReplaced, PanicOnAllMistakes) })
	testutils.FatalUnless(t, didPanic == true, "")
	testutils.FatalUnless(t, panicValue.(error).Error() == err.Error(), "")

	res, err = Join[empty](ewd1, ewd4, PreferPreviousData, MistakeIfDataIsReplaced)
	testutils.FatalUnless(t, err != nil, "")
	testError_any(t, res, "2\nY", ParamMap{"X": 2, "Y": "Y", "Z": "Z"}, []error{ewd1, ewd4})

	didPanic = testutils.CheckPanic(func() { Join[empty](ewd1, 2) })
	testutils.FatalUnless(t, didPanic == true, "")

	didPanic = testutils.CheckPanic(func() { Join[empty](AllowEmptyString) }) // flag invalid for Join_any
	testutils.FatalUnless(t, didPanic == true, "")
}

func TestJoinedErrorValidation(t *testing.T) {
	// test some subtleties of Join
	err1, err := NewErrorWithData_any_params(nil, "BAR$!m!=0{${X}}")
	testutils.FatalUnless(t, err == nil, "%v", err)
	testutils.FatalUnless(t, err1.Error() == "BAR", "%v", err1.Error())
	testutils.FatalUnless(t, err1.ValidateError_Final() == nil, "%v", err1.ValidateError_Final())
	testutils.FatalUnless(t, err1.ValidateError_Base() == nil, "%v", err1.ValidateError_Base())
	testutils.FatalUnless(t, err1.ValidateError_Params(nil) == nil, "%v", err1.ValidateError_Params(nil))

	err2, err := NewErrorWithData_any_params(nil, "FOO", "Y", 1)
	testutils.FatalUnless(t, err == nil, "%v", err)
	testutils.FatalUnless(t, err2.Error() == "FOO", "%v", err2.Error())
	testutils.FatalUnless(t, err2.ValidateError_Final() == nil, "%v", err2.ValidateError_Final())
	testutils.FatalUnless(t, err2.ValidateError_Base() == nil, "%v", err2.ValidateError_Base())
	testutils.FatalUnless(t, err2.ValidateError_Params(nil) == nil, "%v", err2.ValidateError_Params(nil))

	err2b, err := NewErrorWithData_any_params(nil, "FOO", "X", 1)
	testutils.FatalUnless(t, err == nil, "%v", err)
	testutils.FatalUnless(t, err2b.Error() == "FOO", "%v", err2b.Error())
	testutils.FatalUnless(t, err2b.ValidateError_Final() == nil, "%v", err2b.ValidateError_Final())
	testutils.FatalUnless(t, err2b.ValidateError_Base() == nil, "%v", err2b.ValidateError_Base())
	testutils.FatalUnless(t, err2b.ValidateError_Params(nil) == nil, "%v", err2b.ValidateError_Params(nil))

	err3 := errors.New("BAZ")

	errJoined, err := Join_any(err1, err2, err3)
	testutils.FatalUnless(t, err == nil, "%v", err)
	testutils.FatalUnless(t, errJoined.Error() == "BAR\nFOO\nBAZ", "%v", errJoined.Error())
	testutils.FatalUnless(t, errJoined.Error_interpolate(nil) != "BAR\nFOO\nBAZ", "%v", errJoined.Error_interpolate(nil)) // errJoined.Error_interpolate(nil) contains an error.
	testutils.FatalUnless(t, errJoined.ValidateSyntax() == nil, "%v", errJoined.ValidateSyntax())
	testutils.FatalUnless(t, errJoined.ValidateError_Final() == nil, "%v", errJoined.ValidateError_Final())
	testutils.FatalUnless(t, errJoined.ValidateError_Base() == nil, "%v", errJoined.ValidateError_Base())
	testutils.FatalUnless(t, errJoined.ValidateError_Params(nil) != nil, "") // !

	errJoined2, err := Join_any(err1, err2b, err3)
	testutils.FatalUnless(t, err == nil, "%v", err)
	testutils.FatalUnless(t, errJoined2.Error() == "BAR\nFOO\nBAZ", "%v", errJoined2.Error())
	testutils.FatalUnless(t, errJoined2.Error_interpolate(nil) == "BAR1\nFOO\nBAZ", "%v", errJoined2.Error_interpolate(nil))
	testutils.FatalUnless(t, errJoined2.ValidateSyntax() == nil, "%v", errJoined2.ValidateSyntax())
	testutils.FatalUnless(t, errJoined2.ValidateError_Final() == nil, "%v", errJoined2.ValidateError_Final())
	testutils.FatalUnless(t, errJoined2.ValidateError_Base() == nil, "%v", errJoined2.ValidateError_Base())
	testutils.FatalUnless(t, errJoined2.ValidateError_Params(nil) == nil, "")

}
