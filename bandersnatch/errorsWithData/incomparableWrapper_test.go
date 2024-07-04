package errorsWithData

import (
	"errors"
	"maps"
	"reflect"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
)

// Check interface satisfaction
var _ BoxableError = &errorWithParameters_T[struct{}]{}
var _ BoxableError = &errorWithParameters_common{}
var _ BoxableError = &joinedErrors_any{}
var _ BoxableError = &joinedErrors[struct{}]{}

var _ unboxableError = incomparableError{}

// error type that can be boxed without satisfying error with data.
type plainBoxableError struct {
	error
}

func (e plainBoxableError) Is(target error) bool          { return e == UnboxError(target) }
func (e plainBoxableError) SupportsBoxingAsIncomparable() {}

// Check that the BoxErrorAsIncomparable actually returns incomparable errors.
// Also check that double-boxing works as intended.
// Also check that Unboxing works as intended
func TestBoxingAsIncomparable(t *testing.T) {
	didPanic := testutils.CheckPanic(func() { BoxErrorAsIncomparable(nil) })
	testutils.FatalUnless(t, didPanic == true, "")

	e_plain := plainBoxableError{error: errors.New("foo")}
	e_any, _ := NewErrorWithData_any_params(e_plain, "", PanicOnAllMistakes)
	e_T, _ := NewErrorWithData_params[struct{}](e_plain, "", PanicOnAllMistakes)

	boxed_plain := BoxErrorAsIncomparable(e_plain)
	boxed_any := BoxErrorAsIncomparable(e_any)
	boxed_T := BoxErrorAsIncomparable(e_T)

	var boxed_errors__all []incomparableError = []incomparableError{
		boxed_plain,
		boxed_any,
		boxed_T,
	}

	for _, e := range boxed_errors__all {
		testutils.FatalUnless(t, reflect.TypeOf(e).Comparable() == false, "")

		// Check that double-boxing is a no-op.
		// Note that we cannot compare double_box == e, because they are incomparable.
		// We need to reach into the boxed for that.
		double_box := BoxErrorAsIncomparable(e)
		testutils.FatalUnless(t, double_box.BoxableError == e.BoxableError, "")
		testutils.FatalUnless(t, e.BoxableError == UnboxError(e), "")
	}
	// Check unboxing on "plain" errors:
	testutils.FatalUnless(t, UnboxError(nil) == nil, "")
	testutils.FatalUnless(t, UnboxError(e_plain) == e_plain, "")
	testutils.FatalUnless(t, UnboxError(e_any) == e_any, "")
	testutils.FatalUnless(t, UnboxError(e_T) == e_T, "")
}

// returns true iff there is an incomparable error in e's error tree.
//
// This does not check e itself.
func incomparableErrorsInSubtree(e error) (result bool) {
	result = false // no-op, but added for clarity.
	closure := func(child error) {
		if child == nil {
			panic("Cannot happen")
		}
		if reflect.TypeOf(child).Comparable() == false {
			result = true
		}
	}
	callOnErrorSubtree(e, closure)
	return
}

/*
 * Unneeded: We only need the Subtree version.
 */

/*
// returns true iff there is an incomparable error in e's error tree (including e itself)
func incomparableErrorsInTree(e error) (result bool) {
	result = false // no-op, but added for clarity.
	closure := func(child error) {
		if child == nil {
			panic("Cannot happen")
		}
		if reflect.TypeOf(child).Comparable() == false {
			result = true
		}
	}
	callOnErrorTree(e, closure)
	return
}
*/

// Ensures that the free functions provided by this package work for boxed errors, i.e. that
// we can retrieve parameters.
func TestBoxingPreservesParams(t *testing.T) {
	type fooType struct{ Foo int }

	err, _ := NewErrorWithData_any_params(nil, "some error", "Foo", 5, PanicOnAllMistakes)
	boxedErr := BoxErrorAsIncomparable(err)

	testutils.FatalUnless(t, maps.Equal(GetData_map(boxedErr), ParamMap{"Foo": 5}), "")
	foo1, _ := GetData_struct[fooType](err, MissingDataIsMistake, PanicOnAllMistakes)
	foo2, _ := GetData_struct[fooType](boxedErr, MissingDataIsMistake, PanicOnAllMistakes)
	testutils.FatalUnless(t, foo1 == foo2, "")

	testutils.FatalUnless(t, HasData[fooType](err) == HasData[fooType](boxedErr), "")
	testutils.FatalUnless(t, HasParameter(err, "Foo") == HasParameter(boxedErr, "Foo"), "")
	testutils.FatalUnless(t, HasParameter(err, "nonexistant") == HasParameter(boxedErr, "nonexistant"), "")

	get1, ok1 := GetParameter(err, "Foo")
	get2, ok2 := GetParameter(boxedErr, "Foo")

	testutils.FatalUnless(t, get1 == get2, "")
	testutils.FatalUnless(t, ok1 == ok2, "")

	get1, ok1 = GetParameter(err, "nonexistant")
	get2, ok2 = GetParameter(boxedErr, "nonexistant")

	testutils.FatalUnless(t, get1 == get2, "")
	testutils.FatalUnless(t, ok1 == ok2, "")
}

// This test checks that errors created by our API will not contain incomparable errors in the error tree.
func TestMakeIncomparableErrorCreation(t *testing.T) {
	err1 := errors.New("Some error")
	err2, _ := NewErrorWithData_any_params(err1, "", "Foo", 5, "Bar", 10, PanicOnAllMistakes)
	errBoxed := BoxErrorAsIncomparable(err2)

	testfun := func(e error) {
		testutils.FatalUnless(t, HasParameter(e, "Foo") == true, "") // sanity check
		testutils.FatalUnless(t, incomparableErrorsInSubtree(e) == false, "%v contained incomparable error in subtree", e)
		testutils.FatalUnless(t, errors.Is(e, errBoxed) == true, "%v is not a boxed error", e)
		testutils.FatalUnless(t, errors.Is(errBoxed, e) == false, "errors.Is returns true the wrong way for %v", e)
	}

	type fooType struct{ Foo int }

	// Check all ways to create errors with our API:
	e_typed, _ := NewErrorWithData_struct(errBoxed, "", &fooType{Foo: 5}, PanicOnAllMistakes)
	testfun(e_typed)
	e_typed, _ = NewErrorWithData_params[fooType](errBoxed, "", PanicOnAllMistakes)
	testfun(e_typed)
	e_any, _ := NewErrorWithData_any_params(errBoxed, "", PanicOnAllMistakes)
	testfun(e_any)
	e_typed, _ = NewErrorWithData_map[fooType](errBoxed, "", ParamMap{}, PanicOnAllMistakes)
	testfun(e_typed)
	e_any, _ = NewErrorWithData_any_map(errBoxed, "", ParamMap{}, PanicOnAllMistakes)
	testfun(e_any)
	e_any, _ = DeleteParameterFromError_any(errBoxed, "", "Bar", PanicOnAllMistakes)
	testfun(e_any)
	e_typed, _ = DeleteParameterFromError[fooType](errBoxed, "", "Bar", PanicOnAllMistakes)
	testfun(e_typed)
	e_typed, _ = AsErrorWithData[fooType](errBoxed, PanicOnAllMistakes)
	testfun(e_typed)
	// Same tests for Join and Join_any.
	// Note that we need to check all ways of passing args to Join and Join_any
	e_any, _ = Join_any(errBoxed, PanicOnAllMistakes)
	testfun(e_any)
	e_any, _ = Join_any([]error{errBoxed}, PanicOnAllMistakes)
	testfun(e_any)
	e_any, _ = Join_any([1]error{errBoxed}, PanicOnAllMistakes)
	testfun(e_any)
	e_any, _ = Join_any(&[]error{errBoxed}, PanicOnAllMistakes)
	testfun(e_any)
	e_any, _ = Join_any(&[1]error{errBoxed}, PanicOnAllMistakes)
	testfun(e_any)

	e_typed, _ = Join[fooType](errBoxed, PanicOnAllMistakes)
	testfun(e_typed)
	e_typed, _ = Join[fooType]([]error{errBoxed}, PanicOnAllMistakes)
	testfun(e_typed)
	e_typed, _ = Join[fooType]([1]error{errBoxed}, PanicOnAllMistakes)
	testfun(e_typed)
	e_typed, _ = Join[fooType](&[]error{errBoxed}, PanicOnAllMistakes)
	testfun(e_typed)
	e_typed, _ = Join[fooType](&[1]error{errBoxed}, PanicOnAllMistakes)
	testfun(e_typed)

	e_derive_from_unboxed, _ := NewErrorWithData_params[fooType](err2, "")
	testfun(e_derive_from_unboxed)
}

func TestSanity(t *testing.T) {
	err, _ := NewErrorWithData_struct(errors.New("sth"), "Foo", &struct{}{}, PanicOnAllMistakes, ErrorUnlessValidFinal)
	// err, _ := NewErrorWithData_any_params(nil, "Foo")
	Err := BoxErrorAsIncomparable(err)

	errDerived, _ := NewErrorWithData_params[struct{}](err, "")
	testutils.FatalUnless(t, errors.Is(errDerived, err), "")
	testutils.FatalUnless(t, errors.Is(errDerived, Err), "")

	errDerived, _ = NewErrorWithData_params[struct{}](Err, "")
	testutils.FatalUnless(t, errors.Is(errDerived, err), "")
	testutils.FatalUnless(t, errors.Is(errDerived, Err), "")
}

/*

import (
	"errors"
	"fmt"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
)

// dummy type for testing incomparableError_plain

type eIncomparableMaker struct {
	error
}

type eIncomparableMaker_any struct {
	ErrorWithData_any
}

type eIncomparableMaker_struct[StructType any] struct {
	ErrorWithData[StructType]
}

func (eIncomparableMaker) CanMakeIncomparable()           {}
func (eIncomparableMaker_any) CanMakeIncomparable()       {}
func (eIncomparableMaker_struct[_]) CanMakeIncomparable() {}

func (e eIncomparableMaker) Is(target error) bool {
	return e == UnboxError(target)
}

func (e eIncomparableMaker_any) Is(target error) bool {
	return e == UnboxError(target)
}

func (e eIncomparableMaker_struct[_]) Is(target error) bool {
	return e == UnboxError(target)
}

// arbirary struct type
type t1 struct {
	X int
}

var _ IncomparableMaker = eIncomparableMaker{} // value-based
var _ IncomparableMaker = eIncomparableMaker_any{}
var _ IncomparableMaker = eIncomparableMaker_struct[t1]{}

var _ ComparableMaker = incomparableError_plain{}
var _ ComparableMaker = incomparableError_any{}
var _ ComparableMaker = incomparableError[t1]{}

func TestBoxingAndUnboxing(t *testing.T) {
	testutils.FatalUnless(t, UnboxError(nil) == nil, "")
	testutils.FatalUnless(t, UnboxError_any(nil) == nil, "")
	testutils.FatalUnless(t, UnboxError_struct[t1](nil) == nil, "")

	var ePlain eIncomparableMaker = eIncomparableMaker{error: fmt.Errorf("err")}
	var eAny eIncomparableMaker_any = eIncomparableMaker_any{NewErrorWithData_any_params(nil, "errAny", PreferPreviousData, "X", 1)}
	var eT eIncomparableMaker_struct[t1] = eIncomparableMaker_struct[t1]{NewErrorWithData_struct(nil, "errT", &t1{X: 1})}

	ePlainBoxed := MakeErrorIncomparable(ePlain)
	ePlainBoxed2 := MakeErrorIncomparable(ePlainBoxed)
	testutils.FatalUnless(t, ePlainBoxed.IncomparableMaker == ePlainBoxed2.IncomparableMaker, "")

	ePlainWrapped := fmt.Errorf("%w", ePlain)

	testutils.FatalUnless(t, ePlain.Is(ePlainBoxed), "E1")

	testutils.FatalUnless(t, errors.Is(ePlainWrapped, ePlainBoxed), "")
	eAnyBoxed := MakeErrorIncomparable_any(eAny)
	testutils.FatalUnless(t, errors.Is(eAnyBoxed, eAny), "")
	eTBoxed := MakeErrorIncomparable_struct[t1](eT)
	testutils.FatalUnless(t, errors.Is(eTBoxed, eT), "")

	testutils.FatalUnless(t, ePlainBoxed.AsComparable() == ePlain, "")
	testutils.FatalUnless(t, eAnyBoxed.AsComparable() == eAny, "")
	testutils.FatalUnless(t, eT == eTBoxed.AsComparable(), "")
}

*/
