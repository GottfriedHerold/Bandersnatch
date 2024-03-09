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
var unset struct{}

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
	StructIsEqual_Val struct{ val int } // dummy types
	StructIsEqual_Ptr struct{ val int }
)

// same types as above, but incomparable
type (
	IncomparableType     struct{ utils.MakeIncomparable }
	IncomparableType_Val struct {
		utils.MakeIncomparable
		StructIsEqual_Val
	}
	IncomparableType_Ptr struct {
		utils.MakeIncomparable
		StructIsEqual_Ptr
	}
)

func (x StructIsEqual_Val) IsEqual(y StructIsEqual_Val) bool {
	last_comparison_target = y
	last_comparison_id = 1
	return x.val == y.val
}
func (x *StructIsEqual_Ptr) IsEqual(y *StructIsEqual_Ptr) bool {
	last_comparison_target = y
	last_comparison_id = 2
	return x.val == y.val
}

func withPanicResults(f EqualityComparisonFunction) func(any, any) (result bool, didPanic bool, panicValue any) {
	return func(x, y any) (result bool, didPanic bool, panicValue any) {
		didPanic, panicValue = testutils.CheckPanic2(func() { result = f(x, y) })
		return
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
	_, didPanic, _ := comparison_handleNilExt(IncomparableType{}, IncomparableType{})
	testutils.FatalUnless(t, didPanic == true, "")
}

func TestDummy(t *testing.T) {
	VType := reflect.TypeOf(StructIsEqual_Val{})
	VTypePtr := reflect.PointerTo(VType)
	PtrType := reflect.TypeOf(StructIsEqual_Ptr{})
	PtrTypePtr := reflect.PointerTo(PtrType)

	_, found := VType.MethodByName("IsEqual")
	fmt.Println(found)
	_, found = VTypePtr.MethodByName("IsEqual")
	fmt.Println(found)
	_, found = PtrType.MethodByName("IsEqual")
	fmt.Println(found)
	_, found = PtrTypePtr.MethodByName("IsEqual")
	fmt.Println(found)
}

var Comp_IsEqual2 = CustomComparisonMethod("IsEqual") // functionally equivalent to Comparison_IsEqual

func TestComparisonIsEqual(t *testing.T) {
	last_comparsion_mutex.Lock()
	defer last_comparsion_mutex.Unlock()
	clearComparsionState()

	checkPair := func(x any, y any, expectedResult bool, target ...any) {
		testutils.FatalUnless(t, Comparison_IsEqual(x, y) == expectedResult, "")
		assumeComparisonState(t, target...)
		testutils.FatalUnless(t, Comparison_IsEqual(y, x) == expectedResult, "")
		assumeComparisonState(t, target...)
		testutils.FatalUnless(t, Comp_IsEqual2(x, y) == expectedResult, "")
		assumeComparisonState(t, target...)
		testutils.FatalUnless(t, Comp_IsEqual2(y, x) == expectedResult, "")
		assumeComparisonState(t, target...)

	}
	checkPair(4, 5, false, unset, unset)
	checkPair(4, 4, true, unset, unset)
	checkPair(nil, (*int)(nil), true, unset, unset)
	checkPair(nil, new(int), false, unset, unset)
	checkPair(nil, nil, true, unset, unset)
	checkPair(new(int), new(int), false, unset, unset)

}
