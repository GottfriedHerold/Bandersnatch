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

/*
func TestJoinAny(t *testing.T) {
	err1 := errors.New("A")
	err2 := errors.New("B")
	ewd1, _ := NewErrorWithData_any_params(nil, "%v{X}", "X", 2, PanicOnAllErrors, ErrorUnlessValidFinal)
	ewd2, e := NewErrorWithData_any_params(nil, "{", "X", 3) // syntax error
	testutils.FatalUnless(t, e != nil, "")
	ewd3, _ := NewErrorWithData_any_params(nil, "$v{X}", PanicOnAllErrors, ErrorUnlessValidBase)
	testutils.FatalUnless(t, ewd.ValidateError_Final() != nil, "")
	ewd4, _ := NewErrorWithData_any_params(nil, "%{Y}", "Y", "Y", "Z", "Z")

	res, err := Join_any(err1, err2)
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError_any(t, res, "A\nB", nil, []error{err1, err2})

	res, err = Join_any([]error{err1, nil, err2}, nil)
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError_any(t, res, "A\nB", nil, []error{err1, err2})

	res, err = Join_any([3]error{err1, nil, err2}, nil)
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError_any(t, res, "A\nB", nil, []error{err1, err2})

}
*/
