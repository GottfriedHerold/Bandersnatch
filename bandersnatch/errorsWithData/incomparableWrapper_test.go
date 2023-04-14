package errorsWithData

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
