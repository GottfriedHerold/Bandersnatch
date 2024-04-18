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
