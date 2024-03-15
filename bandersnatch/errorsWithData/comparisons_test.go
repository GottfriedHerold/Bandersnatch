package errorsWithData

import (
	"fmt"
	"reflect"
	"sync"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

// This file contains tests for our comparison methods.
// These comparison methods look (via reflection logic) for the presence of certain functions / check whether types are comparable or not.
// Depending on that, we either use the special functions or use a plain ==.
// Note that checking the presence of certain functions cannot be done by interfaces here (because we don't know the argument type);
// Go's lack of interface contra- and covariance makes them useless for this purpose.
// Part of the relevant testing here is "what function actually got called".
// Since the comparison methods themselves have no way of returning "I got called with params...", we set some global variables and check those.
// To avoid race-conditions if someone tries to run these tests in parallel, we include a mutex.
// This mutex must be locked at the beginning of every test that accesses these global variables.

// as explained above, these are global variables that our (test-)comparison functions write to in order to provide a channel to indicate what was called with what.
// (Note: we could make these actual Go channels, but that wouldn't help much, as we still need the mutex.
var (
	last_comparison_target any = unset
	last_comparison_id     any = unset
)
var last_comparsion_mutex sync.Mutex // mutex protecting the above

// special value to indicate "no value". We cannot use nil for that, as any(nil) might appear naturally.
var unset = new(struct{ bool }) // we use a pointer to have a unique value. Note that struct{}{} == struct{}{} in Go.

// helper function for equality checks in tests

// compare_catch_panic returns in ret whether x==y and in didPanic whether the comparison panics.
// In the latter case, we return ret == true (i.e. as if x==y was the case -- this is appropriate for our use cases).
// Recall that the comparison panics iff x and y have the same non-comparable type.
func compare_catch_panic(x, y any) (ret bool, didPanic bool) {
	didPanic = false // zero-initialized anyway, but added for clarity
	defer func() {   // catch panic
		if recover() != nil {
			ret = true
			didPanic = true
		}
	}()
	ret = (x == y)
	return
}

// clearComparisonState unsets the last_comparison variables. This must only be called if last_comparison_mutex is held.
func clearComparsionState() {
	last_comparison_target = unset
	last_comparison_id = unset
}

// assumeComparisonState is a helper function that checks that last_comparison_target
// (and optionally last_comparison_id and the type of target) has the expected values.
// Usage: assumeComparisonState(t, target, id, type), where id and target are optional and type has type reflect.Type
// It also unsets those values after the comparison.
func assumeComparisonState(t *testing.T, target ...any) {
	testutils.FatalUnless(t, len(target) == 1 || len(target) == 2 || len(target) == 3, "assumeComparsionState called with wrong number of arguments")
	compare, _ := compare_catch_panic(last_comparison_target, target[0])
	testutils.FatalUnless(t, compare, "last_comparison_target does not have expected value: Got %v, expected %v", last_comparison_target, target[0])
	if len(target) > 1 {
		// compare, _ = compare_catch_panic(last_comparison_id, target[1])
		testutils.FatalUnless(t, last_comparison_id == target[1], "last_comparison_id does not have expected value: Got %v, expected %v", last_comparison_id, target[1])
	}
	if len(target) > 2 {
		targetType := target[2].(reflect.Type) // may panic. That would be a wrong usage of assumeComparisonState and panic is appropriate.
		testutils.FatalUnless(t, reflect.TypeOf(last_comparison_target) == targetType, "last_comparison_target has unexpected type: Got %T, expected %v", last_comparison_target, targetType)
	}
	clearComparsionState()
}

// types with an IsEqual method defined on value resp. pointer receivers and arguments
type (
	structIsEqual_Val struct{ val int } // dummy types
	structIsEqual_Ptr struct{ val int }
)

// same types as above, but incomparable
type (
	incomparableType     struct{ utils.MakeIncomparable }
	incomparableType_Val struct {
		utils.MakeIncomparable
		structIsEqual_Val
	}
	incomparableType_Ptr struct {
		utils.MakeIncomparable
		structIsEqual_Ptr
	}
)

func (x structIsEqual_Val) IsEqual(y structIsEqual_Val) bool {
	last_comparison_target = y
	last_comparison_id = 1
	return x.val == y.val
}
func (x *structIsEqual_Ptr) IsEqual(y *structIsEqual_Ptr) bool {
	last_comparison_target = *y
	last_comparison_id = 2
	return x.val == y.val
}

func (x incomparableType_Val) IsEqual(y incomparableType_Val) bool {
	last_comparison_target = y.structIsEqual_Val
	last_comparison_id = 3
	return x.val == y.val
}

func (x *incomparableType_Ptr) IsEqual(y *incomparableType_Ptr) bool {
	last_comparison_target = y.structIsEqual_Ptr
	last_comparison_id = 4
	return x.val == y.val
}

type i2_Val struct {
	utils.MakeIncomparable
	structIsEqual_Val
}

type i2_Ptr struct {
	utils.MakeIncomparable
	structIsEqual_Ptr
}

func (x i2_Val) IsEqual(y any) bool {
	last_comparison_target = y
	last_comparison_id = 5
	switch y := y.(type) {
	case i2_Val:
		return x.val == y.val
	case structIsEqual_Val:
		return x.val == y.val
	case incomparableType_Val:
		return x.val == y.val
	default:
		panic(fmt.Errorf("I2_Val.IsEqual called with %v of type %T", y, y))
	}
}

func (x *i2_Ptr) IsEqual(y any) bool {
	last_comparison_id = 6
	switch y := y.(type) {
	case *i2_Ptr:
		last_comparison_target = y.structIsEqual_Ptr
		return x.val == y.val
	case *structIsEqual_Ptr:
		last_comparison_target = *y
		return x.val == y.val
	case *incomparableType_Ptr:
		last_comparison_target = y.structIsEqual_Ptr
		return x.val == y.val
	default:
		panic(fmt.Errorf("I2_Val.IsEqual called with %v of type %T", y, y))
	}
}

func TestComparisonHandleNils(t *testing.T) {
	testutils.FatalUnless(t, comparison_handleNils(5, 4) == false, "")
	testutils.FatalUnless(t, comparison_handleNils(5, 5) == true, "")
	testutils.FatalUnless(t, comparison_handleNils(int(5), uint(5)) == false, "")
	testutils.FatalUnless(t, comparison_handleNils(nil, 4) == false, "")
	testutils.FatalUnless(t, comparison_handleNils(4, nil) == false, "")
	testutils.FatalUnless(t, comparison_handleNils(nil, (*int)(nil)) == true, "")
	testutils.FatalUnless(t, comparison_handleNils((*int)(nil), nil) == true, "")
	testutils.FatalUnless(t, comparison_handleNils(nil, nil) == true, "")
	comparison_handleNilExt := withPanicResults(comparison_handleNils)
	_, didPanic, _ := comparison_handleNilExt(incomparableType{}, incomparableType{})
	testutils.FatalUnless(t, didPanic == true, "")
}

var Comp_IsEqual2 = CustomComparisonMethod("IsEqual") // functionally equivalent to Comparison_IsEqual

var expectPanic = new(struct{ bool })
var noFlip = new(struct{ bool })

type namedT[T any] struct {
	val  T
	name string
}

func withName[T any](x T, name string) namedT[T] {
	return namedT[T]{val: x, name: name}
}

// makeCheckerForComparisonFunctions returns a function test_fun(x,y,results, target...) that is used to test all comparison functions fun among funsWithNames
// It should be called as makeCheckerForComparisonFunctions(t, withName[EqualityComparisonFunction](fun, "name"), ...)
// where "name" is the name of the function fun (this name is used in error messages). The generic parameter may be deducible and omitted, depending on how fun was defined.
//
// The returned test_fun(x,y, result, target...) checks for each comparison function that
// fun(x,y) == results
// if target gets parsed as follows:
//   - If expectPanic is among targets, we expect fun(x,y) to panic, otherwise we don't
//   - If noFlip is among targets, we don't check fun(y,x)
//   - If any remaining optional arguments are present, they passed to assumeComparisonState.
//   - expectPanic and/or noFlip must come before the arguments passed to assumeComparisonState
//
// Unless noFlip was set, all tests are repeated for fun(y,x), i.e. with order of arguments flipped, *except for the call to assumeComparisonState*
//
// The resulting test_fun must only be called if last_comparison_mutex is held.
func makeCheckerForComparisonFunctions(t *testing.T, funsWithNames ...namedT[EqualityComparisonFunction]) func(any, any, bool, ...any) {
	testutils.FatalUnless(t, unset != expectPanic, "")
	return func(x any, y any, expectedResult bool, target ...any) {
		var shouldPanic bool = false
		var doNotFlip bool = false
		for len(target) > 0 && (target[0] == expectPanic || target[0] == noFlip) {
			switch target[0] {
			case expectPanic:
				shouldPanic = true
			case noFlip:
				doNotFlip = true
			default:
				panic("Cannot happen")
			}
			target = target[1:]

		}

		for _, namedFunDirect := range funsWithNames {
			funDirect := namedFunDirect.val
			name := namedFunDirect.name

			funCatchPanic := withPanicResults(funDirect)
			var res, didPanic bool
			var panicVal any

			res, didPanic, panicVal = funCatchPanic(x, y)
			if len(target) > 0 {
				assumeComparisonState(t, target...)
			}
			testutils.FatalUnless(t, didPanic == shouldPanic,
				"Comparison function %v, called with %v and %v did not match expected panic behavior: Expected Panic: %v, GotPanic: %v, PanicValue: %v",
				name, x, y, shouldPanic, didPanic, panicVal)

			testutils.FatalUnless(t, res == expectedResult,
				"Comparison function %v, called with %v and %v did not produce the expected result: Got %v, expected %v. GotPanic: %v",
				name, x, y, res, expectedResult, didPanic)

			if doNotFlip {
				return
			}

			res, didPanic, panicVal = funCatchPanic(y, x)
			/*if len(target) > 0 {
				assumeComparisonState(t, target...)
			}
			*/
			testutils.FatalUnless(t, didPanic == shouldPanic,
				"Comparison function %v, called with %v and %v did not match expected panic behavior: Expected Panic: %v, GotPanic: %v, PanicValue: %v",
				name, y, x, shouldPanic, didPanic, panicVal)
			testutils.FatalUnless(t, res == expectedResult,
				"Comparison function %v, called with %v and %v did not produce the expected result: Got %v, expected %v. GotPanic: %v",
				name, y, x, res, expectedResult, didPanic)

		}
	}
}

func TestComparisonIsEqual(t *testing.T) {
	last_comparsion_mutex.Lock()
	defer last_comparsion_mutex.Unlock()
	clearComparsionState()

	checkPair := makeCheckerForComparisonFunctions(t, withName[EqualityComparisonFunction](Comparison_IsEqual, "Comparison_IsEqual"), withName(Comp_IsEqual2, "Comp_IsEqual2"))

	checkPair(4, 5, false)
	checkPair(4, 4, true)
	checkPair(nil, (*int)(nil), true)
	checkPair(nil, new(int), false)
	checkPair(nil, nil, true)
	checkPair(new(int), new(int), false)
	intPtr := new(int)
	checkPair(intPtr, intPtr, true)

	incompValue := incomparableType{}

	sVal := structIsEqual_Val{val: 4}
	sVal2 := structIsEqual_Val{val: 5}
	sVal3 := structIsEqual_Val{val: 4}
	pVal := structIsEqual_Ptr{val: 4}
	pVal2 := structIsEqual_Ptr{val: 5}
	pVal3 := structIsEqual_Ptr{val: 4}

	sValI := incomparableType_Val{structIsEqual_Val: structIsEqual_Val{val: 4}}
	sValI2 := incomparableType_Val{structIsEqual_Val: structIsEqual_Val{val: 5}}
	pValI := incomparableType_Ptr{structIsEqual_Ptr: structIsEqual_Ptr{val: 4}}
	pValI2 := incomparableType_Ptr{structIsEqual_Ptr: structIsEqual_Ptr{val: 5}}

	checkPair(incompValue, incompValue, false, expectPanic)
	checkPair(sVal, sVal, true, sVal, 1)
	checkPair(sVal, sVal2, false, sVal2, 1)
	checkPair(sVal, sVal3, true, sVal3, 1)
	checkPair(pVal, pVal, true, pVal, 2)
	checkPair(pVal, pVal2, false, pVal2, 2)
	checkPair(pVal, pVal3, true, pVal3, 2)

	checkPair(sValI, sValI, true, sVal, 3)
	checkPair(sValI, sValI2, false, sVal2, 3)
	checkPair(pValI, pValI, true, pVal, 4)
	checkPair(pValI, pValI2, false, pVal2, 4)

	iVal := i2_Val{structIsEqual_Val: structIsEqual_Val{val: 4}}
	iVal2 := i2_Val{structIsEqual_Val: structIsEqual_Val{val: 5}}
	iVal3 := i2_Val{structIsEqual_Val: structIsEqual_Val{val: 4}}

	ipVal := i2_Ptr{structIsEqual_Ptr: structIsEqual_Ptr{val: 4}}
	ipVal2 := i2_Ptr{structIsEqual_Ptr: structIsEqual_Ptr{val: 5}}
	ipVal3 := i2_Ptr{structIsEqual_Ptr: structIsEqual_Ptr{val: 4}}

	checkPair(iVal, iVal, true, iVal, 5)
	checkPair(iVal, iVal2, false, iVal2, 5)
	checkPair(iVal, iVal3, true, iVal3, 5)

	checkPair(ipVal, ipVal, true, pVal, 6)
	checkPair(ipVal, ipVal2, false, pVal2, 6)
	checkPair(ipVal, ipVal3, true, pVal3, 6)

	checkPair(iVal, sVal, true, noFlip, sVal, 5) // sVal.IsEqual(iVal) is not tested, as that would panic
	checkPair(sVal, iVal, false, noFlip, expectPanic)
	checkPair(iVal, pVal, false, noFlip, expectPanic)

	checkPair(ipVal, pVal, true, noFlip, pVal, 6)
	checkPair(ipVal, sVal, false, noFlip, expectPanic)

}
