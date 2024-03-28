package errorsWithData

import (
	"errors"
	"fmt"
	"maps"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

func TestMergeMapsPreferOld(t *testing.T) {
	var m1 ParamMap = ParamMap{}
	var m2 ParamMap = ParamMap{"Foo": 5}

	var config errorCreationConfig
	parseFlagArgs(&config, PreferPreviousData)
	var configPreferOld config_OldData = config.config_OldData

	ret := mergeMaps(&m2, ParamMap{}, configPreferOld)
	testutils.FatalUnless(t, utils.CompareMaps(m2, ParamMap{"Foo": 5}), "")
	testutils.FatalUnless(t, ret == nil, "")
	ret = mergeMaps(&m2, nil, configPreferOld)
	testutils.FatalUnless(t, utils.CompareMaps(m2, ParamMap{"Foo": 5}), "")
	testutils.FatalUnless(t, ret == nil, "")
	ret = mergeMaps(&m1, ParamMap{"Foo": 6}, configPreferOld)
	testutils.FatalUnless(t, utils.CompareMaps(m1, ParamMap{"Foo": 6}), "")
	testutils.FatalUnless(t, ret == nil, "")
	ret = mergeMaps(&m1, ParamMap{"Foo": nil, "Bar": nil}, configPreferOld)
	testutils.FatalUnless(t, utils.CompareMaps(m1, ParamMap{"Foo": 6, "Bar": nil}), "")
	testutils.FatalUnless(t, ret == nil, "")
}

func TestMergeMapsPreferNew(t *testing.T) {
	var m1 ParamMap = ParamMap{}
	var m2 ParamMap = ParamMap{"Foo": 5}

	var config errorCreationConfig
	parseFlagArgs(&config, ReplacePreviousData)
	var configPreferOld config_OldData = config.config_OldData

	ret := mergeMaps(&m2, ParamMap{}, configPreferOld)
	testutils.FatalUnless(t, utils.CompareMaps(m2, ParamMap{"Foo": 5}), "")
	testutils.FatalUnless(t, ret == nil, "")
	ret = mergeMaps(&m2, nil, configPreferOld)
	testutils.FatalUnless(t, utils.CompareMaps(m2, ParamMap{"Foo": 5}), "")
	testutils.FatalUnless(t, ret == nil, "")
	ret = mergeMaps(&m1, ParamMap{"Foo": 6}, configPreferOld)
	testutils.FatalUnless(t, utils.CompareMaps(m1, ParamMap{"Foo": 6}), "")
	testutils.FatalUnless(t, ret == nil, "")
	ret = mergeMaps(&m1, ParamMap{"Foo": nil, "Bar": nil}, configPreferOld)
	testutils.FatalUnless(t, utils.CompareMaps(m1, ParamMap{"Foo": nil, "Bar": nil}), "")
	testutils.FatalUnless(t, ret == nil, "")
}

func TestMergeMaps_EqualityCheck(t *testing.T) {
	var configGeneral errorCreationConfig
	parseFlagArgs(&configGeneral, ReplacePreviousData, EnsureDataIsNotReplaced) // Note: Flag order matters here.
	config := configGeneral.config_OldData
	type incomp struct{ utils.MakeIncomparable } // incomparable type

	// comparison that returns true (and does not panic) if x and y have the same non-comparable type
	dummyEqFn := func(x, y any) bool {
		result, didPanic := compare_catch_panic(x, y)
		return result || didPanic
	}
	// compare maps with this, using maps.EqualFunc from the standard library
	mapsEq := func(x, y ParamMap) bool {
		return maps.EqualFunc(x, y, dummyEqFn)
	}

	// We work with m all the time, resetting it to startMap after each actual modification.
	startMap := ParamMap{"Nil": nil, "TypedNil": (*int)(nil), "Foo": int(5), "Inc": incomp{}}
	m := maps.Clone(startMap)

	ret := mergeMaps(&m, ParamMap{}, config)
	testutils.FatalUnless(t, mapsEq(m, startMap), "")
	testutils.FatalUnless(t, ret == nil, "")
	ret = mergeMaps(&m, nil, config)
	testutils.FatalUnless(t, mapsEq(m, startMap), "")
	testutils.FatalUnless(t, ret == nil, "")

	ret = mergeMaps(&m, ParamMap{"Nil": (*uint)(nil), "Bar": -5, "Foo": uint(5)}, config)
	testutils.FatalUnless(t, mapsEq(m, ParamMap{"Nil": (*uint)(nil), "TypedNil": (*int)(nil), "Bar": -5, "Foo": uint(5), "Inc": incomp{}}), "Unexpected value of m: %v", m)
	//fmt.Println(ret)
	testutils.FatalUnless(t, len(ret) == 1, "unexpected errors: %v", ret) // expect 1 error (from type mismatch with Foo)

	m = maps.Clone(startMap)
	ret = mergeMaps(&m, m, config)
	testutils.FatalUnless(t, mapsEq(m, startMap), "")
	// fmt.Println(ret)
	testutils.FatalUnless(t, len(ret) == 1, "") // 1 error from the incomparable value (with panic being caught)

	m = maps.Clone(startMap)
	ret = mergeMaps(&m, ParamMap{"Foo": 5}, config)
	testutils.FatalUnless(t, mapsEq(m, startMap), "")
	testutils.FatalUnless(t, ret == nil, "")

	// repeat the above with config.PreferNew() set to false
	parseFlagArgs(&configGeneral, PreferPreviousData, EnsureDataIsNotReplaced)
	config = configGeneral.config_OldData
	m = maps.Clone(startMap)

	ret = mergeMaps(&m, ParamMap{}, config)
	testutils.FatalUnless(t, mapsEq(m, startMap), "")
	testutils.FatalUnless(t, ret == nil, "")
	ret = mergeMaps(&m, nil, config)
	testutils.FatalUnless(t, mapsEq(m, startMap), "")
	testutils.FatalUnless(t, ret == nil, "")

	ret = mergeMaps(&m, ParamMap{"Nil": (*uint)(nil), "Bar": -5, "Foo": uint(5)}, config)
	testutils.FatalUnless(t, mapsEq(m, ParamMap{"Nil": nil, "TypedNil": (*int)(nil), "Bar": -5, "Foo": int(5), "Inc": incomp{}}), "Unexpected value of m: %v", m)
	//fmt.Println(ret)
	testutils.FatalUnless(t, len(ret) == 1, "unexpected errors: %v", ret) // expect 1 error (from type mismatch with Foo)

	m = maps.Clone(startMap)
	ret = mergeMaps(&m, m, config)
	testutils.FatalUnless(t, mapsEq(m, startMap), "")
	// fmt.Println(ret)
	testutils.FatalUnless(t, len(ret) == 1, "") // 1 error from the incomparable value (with panic being caught)

	m = maps.Clone(startMap)
	ret = mergeMaps(&m, ParamMap{"Foo": 5}, config)
	testutils.FatalUnless(t, mapsEq(m, startMap), "")
	testutils.FatalUnless(t, ret == nil, "")

	errPanic := fmt.Errorf("Some Error")
	var expectedPanicValue any = errPanic
	var numCalls int
	panickingCompare := func(x, y any) bool { numCalls++; panic(expectedPanicValue) }
	parseFlagArgs(&configGeneral, EnsureDataIsNotReplaced_fun(panickingCompare))
	config = configGeneral.config_OldData
	m = maps.Clone(startMap)
	ret = mergeMaps(&m, ParamMap{"New": 10, "Foo": 7, "Nil": nil}, config)
	testutils.FatalUnless(t, mapsEq(m, ParamMap{"Nil": nil, "TypedNil": (*int)(nil), "Foo": 5, "Inc": incomp{}, "New": 10}), "")
	// We expect 2 calls to the comparison function, from "Foo" and "Nil"
	testutils.FatalUnless(t, len(ret) == 2, "%v", ret)
	testutils.FatalUnless(t, numCalls == 2, "")
	// returned errors should wrap the panic value if it is an error
	testutils.FatalUnless(t, errors.Is(ret[0], errPanic), "")
	testutils.FatalUnless(t, errors.Is(ret[1], errPanic), "")

	expectedPanicValue = 0 // repeat with non-error type for panicValue
	numCalls = 0
	m = maps.Clone(startMap)
	ret = mergeMaps(&m, ParamMap{"New": 10, "Foo": 7, "Nil": nil}, config)
	testutils.FatalUnless(t, mapsEq(m, ParamMap{"Nil": nil, "TypedNil": (*int)(nil), "Foo": 5, "Inc": incomp{}, "New": 10}), "")
	// We expect 2 calls to the comparison function, from "Foo" and "Nil"
	testutils.FatalUnless(t, len(ret) == 2, "%v", ret)
	testutils.FatalUnless(t, numCalls == 2, "")

	//repeat with ReplacePreviousData
	parseFlagArgs(&configGeneral, ReplacePreviousData, EnsureDataIsNotReplaced_fun(panickingCompare))
	config = configGeneral.config_OldData
	m = maps.Clone(startMap)
	numCalls = 0
	expectedPanicValue = errPanic
	ret = mergeMaps(&m, ParamMap{"New": 10, "Foo": 7, "Nil": nil}, config)
	testutils.FatalUnless(t, mapsEq(m, ParamMap{"Nil": nil, "TypedNil": (*int)(nil), "Foo": 7, "Inc": incomp{}, "New": 10}), "")
	// We expect 2 calls to the comparison function, from "Foo" and "Nil"
	testutils.FatalUnless(t, len(ret) == 2, "%v", ret)
	testutils.FatalUnless(t, numCalls == 2, "")
	// returned errors should wrap the panic value if it is an error
	testutils.FatalUnless(t, errors.Is(ret[0], errPanic), "")
	testutils.FatalUnless(t, errors.Is(ret[1], errPanic), "")

	numCalls = 0
	expectedPanicValue = 0
	m = maps.Clone(startMap)
	ret = mergeMaps(&m, ParamMap{"New": 10, "Foo": 7, "Nil": nil}, config)
	testutils.FatalUnless(t, mapsEq(m, ParamMap{"Nil": nil, "TypedNil": (*int)(nil), "Foo": 7, "Inc": incomp{}, "New": 10}), "")
	// We expect 2 calls to the comparison function, from "Foo" and "Nil"
	testutils.FatalUnless(t, len(ret) == 2, "%v", ret)
	testutils.FatalUnless(t, numCalls == 2, "")

	//not catching panics:
	parseFlagArgs(&configGeneral, LetComparisonFunctionPanic)
	config = configGeneral.config_OldData
	m = maps.Clone(startMap)
	numCalls = 0
	expectedPanicValue = errPanic
	didPanic, panicValue := testutils.CheckPanic2(mergeMaps, &m, ParamMap{"New": 10, "Foo": 7, "Nil": nil}, config)
	testutils.FatalUnless(t, didPanic == true, "")
	testutils.FatalUnless(t, panicValue == errPanic, "")
	testutils.FatalUnless(t, numCalls == 1, "")

	configGeneral = errorCreationConfig{}
	parseFlagArgs(&configGeneral, LetComparisonFunctionPanic, EnsureDataIsNotReplaced)
	config = configGeneral.config_OldData
	m = maps.Clone(startMap)
	var nilslice []int
	ret = mergeMaps(&m, ParamMap{"Nil": nilslice, "Foo": -1}, config)
	testutils.FatalUnless(t, mapsEq(m, ParamMap{"Nil": []int{}, "Foo": -1, "TypedNil": (*int)(nil), "Inc": incomp{}}), "") // Note: two of these comparisons are for incomparable types.
	testutils.FatalUnless(t, len(ret) == 1, "%v", ret)
}

// mostly copy&pasted from the above test. The functions are very similar, after all.
func TestFillMapFromStruct_EqualityTest(t *testing.T) {
	var configGeneral errorCreationConfig
	parseFlagArgs(&configGeneral, ReplacePreviousData, EnsureDataIsNotReplaced) // Note: Flag order matters here.
	config := configGeneral.config_OldData
	type incomp struct{ utils.MakeIncomparable } // incomparable type

	// comparison that returns true (and does not panic) if x and y have the same non-comparable type
	dummyEqFn := func(x, y any) bool {
		result, didPanic := compare_catch_panic(x, y)
		return result || didPanic
	}
	// compare maps with this, using maps.EqualFunc from the standard library
	mapsEq := func(x, y ParamMap) bool {
		return maps.EqualFunc(x, y, dummyEqFn)
	}

	// We work with m all the time, resetting it to startMap after each actual modification.
	startMap := ParamMap{"Nil": nil, "TypedNil": (*int)(nil), "Foo": int(5), "Inc": incomp{}}
	m := maps.Clone(startMap)

	type emptyStruct struct{}
	ret := fillMapFromStruct(&m, &emptyStruct{}, config) // config: ReplacePrevious, EnsureDataNotReplaced
	testutils.FatalUnless(t, mapsEq(m, startMap), "")
	testutils.FatalUnless(t, ret == nil, "")

	m = nil
	ret = fillMapFromStruct(&m, &emptyStruct{}, config)   // config: ReplacePrevious, EnsureDataNotReplaced
	testutils.FatalUnless(t, m != nil && len(m) == 0, "") //empty map, not nil map
	testutils.FatalUnless(t, ret == nil, "")

	type invalidType struct{ unexported int }
	var didPanic bool = testutils.CheckPanic(fillMapFromStruct[invalidType], &m, &invalidType{}, config)
	testutils.FatalUnless(t, didPanic == true, "")

	type S1 struct {
		Nil *uint
		Bar int
		Foo uint
	}
	m = maps.Clone(startMap)
	ret = fillMapFromStruct(&m, &S1{Nil: nil, Bar: -5, Foo: 5}, config) // config: ReplacePrevious, EnsureDataNotReplaced
	testutils.FatalUnless(t, mapsEq(m, ParamMap{"Nil": (*uint)(nil), "TypedNil": (*int)(nil), "Bar": -5, "Foo": uint(5), "Inc": incomp{}}), "Unexpected value of m: %v", m)
	//fmt.Println(ret)
	testutils.FatalUnless(t, len(ret) == 1, "unexpected errors: %v", ret) // expect 1 error (from type mismatch with Foo)

	type SStart struct {
		Nil      any
		TypedNil *int
		Foo      int
		Inc      incomp
	}
	var startMap_struct SStart = SStart{Nil: nil, TypedNil: nil, Foo: 5, Inc: incomp{}}
	m = maps.Clone(startMap)
	ret = fillMapFromStruct(&m, &startMap_struct, config) // config: ReplacePrevious, EnsureDataNotReplaced
	testutils.FatalUnless(t, mapsEq(m, startMap), "")
	// fmt.Println(ret)
	testutils.FatalUnless(t, len(ret) == 1, "") // 1 error from the incomparable value (with panic being caught)

	m = maps.Clone(startMap)
	ret = fillMapFromStruct(&m, &struct{ Foo int }{Foo: 5}, config) // config: ReplacePrevious, EnsureDataNotReplaced
	testutils.FatalUnless(t, mapsEq(m, startMap), "")
	testutils.FatalUnless(t, ret == nil, "")

	// repeat the above with config.PreferNew() set to false
	parseFlagArgs(&configGeneral, PreferPreviousData, EnsureDataIsNotReplaced)
	config = configGeneral.config_OldData
	m = maps.Clone(startMap)

	ret = fillMapFromStruct(&m, &struct{}{}, config) // config: PreferPreviousData, EnsureDataNotReplaced
	testutils.FatalUnless(t, mapsEq(m, startMap), "")
	testutils.FatalUnless(t, ret == nil, "")

	ret = fillMapFromStruct(&m, &S1{Nil: nil, Bar: -5, Foo: 5}, config) // config: PreferPreviousData, EnsureDataNotReplaced
	testutils.FatalUnless(t, mapsEq(m, ParamMap{"Nil": nil, "TypedNil": (*int)(nil), "Bar": -5, "Foo": int(5), "Inc": incomp{}}), "Unexpected value of m: %v", m)
	testutils.FatalUnless(t, len(ret) == 1, "unexpected errors: %v", ret) // expect 1 error (from type mismatch with Foo)

	m = maps.Clone(startMap)
	ret = fillMapFromStruct(&m, &startMap_struct, config) // config: PreferPreviousData, EnsureDataNotReplaced
	testutils.FatalUnless(t, mapsEq(m, startMap), "")
	testutils.FatalUnless(t, len(ret) == 1, "") // 1 error from the incomparable value (with panic being caught)

	m = maps.Clone(startMap)
	ret = fillMapFromStruct(&m, &struct{ Foo int }{Foo: 5}, config) // config: PreferPreviousData, EnsureDataNotReplaced
	testutils.FatalUnless(t, mapsEq(m, startMap), "")
	testutils.FatalUnless(t, ret == nil, "")

	errPanic := fmt.Errorf("Some Error")
	var expectedPanicValue any = errPanic
	var numCalls int
	panickingCompare := func(x, y any) bool { numCalls++; panic(expectedPanicValue) }

	parseFlagArgs(&configGeneral, EnsureDataIsNotReplaced_fun(panickingCompare))
	config = configGeneral.config_OldData
	m = maps.Clone(startMap)
	type SNew struct {
		New int
		Foo int
		Nil any
	}
	ret = fillMapFromStruct(&m, &SNew{New: 10, Foo: 7, Nil: nil}, config) // config: PreverPrevious, Panicking compare (caught)
	testutils.FatalUnless(t, mapsEq(m, ParamMap{"Nil": nil, "TypedNil": (*int)(nil), "Foo": 5, "Inc": incomp{}, "New": 10}), "")
	// We expect 2 calls to the comparison function, from "Foo" and "Nil"
	testutils.FatalUnless(t, len(ret) == 2, "%v", ret)
	testutils.FatalUnless(t, numCalls == 2, "")
	// returned errors should wrap the panic value if it is an error
	testutils.FatalUnless(t, errors.Is(ret[0], errPanic), "")
	testutils.FatalUnless(t, errors.Is(ret[1], errPanic), "")

	expectedPanicValue = 0 // repeat with non-error type for panicValue
	numCalls = 0
	m = maps.Clone(startMap)
	ret = fillMapFromStruct(&m, &SNew{New: 10, Foo: 7, Nil: nil}, config) // config: PreverPrevious, Panicking compare (caught)
	testutils.FatalUnless(t, mapsEq(m, ParamMap{"Nil": nil, "TypedNil": (*int)(nil), "Foo": 5, "Inc": incomp{}, "New": 10}), "")
	// We expect 2 calls to the comparison function, from "Foo" and "Nil"
	testutils.FatalUnless(t, len(ret) == 2, "%v", ret)
	testutils.FatalUnless(t, numCalls == 2, "")

	//repeat with ReplacePreviousData
	parseFlagArgs(&configGeneral, ReplacePreviousData, EnsureDataIsNotReplaced_fun(panickingCompare))
	config = configGeneral.config_OldData
	m = maps.Clone(startMap)
	numCalls = 0
	expectedPanicValue = errPanic
	ret = fillMapFromStruct(&m, &SNew{New: 10, Foo: 7, Nil: nil}, config) // config: PreferNew, Panicking compare (caught)
	testutils.FatalUnless(t, mapsEq(m, ParamMap{"Nil": nil, "TypedNil": (*int)(nil), "Foo": 7, "Inc": incomp{}, "New": 10}), "")
	// We expect 2 calls to the comparison function, from "Foo" and "Nil"
	testutils.FatalUnless(t, len(ret) == 2, "%v", ret)
	testutils.FatalUnless(t, numCalls == 2, "")
	// returned errors should wrap the panic value if it is an error
	testutils.FatalUnless(t, errors.Is(ret[0], errPanic), "")
	testutils.FatalUnless(t, errors.Is(ret[1], errPanic), "")

	numCalls = 0
	expectedPanicValue = 0
	m = maps.Clone(startMap)
	ret = fillMapFromStruct(&m, &SNew{New: 10, Foo: 7, Nil: nil}, config) // config: PreferNew, Panicking compare (caught)
	testutils.FatalUnless(t, mapsEq(m, ParamMap{"Nil": nil, "TypedNil": (*int)(nil), "Foo": 7, "Inc": incomp{}, "New": 10}), "")
	// We expect 2 calls to the comparison function, from "Foo" and "Nil"
	testutils.FatalUnless(t, len(ret) == 2, "%v", ret)
	testutils.FatalUnless(t, numCalls == 2, "")

	//not catching panics:
	parseFlagArgs(&configGeneral, LetComparisonFunctionPanic)
	config = configGeneral.config_OldData
	m = maps.Clone(startMap)
	numCalls = 0
	expectedPanicValue = errPanic
	didPanic, panicValue := testutils.CheckPanic2(fillMapFromStruct[SNew], &m, &SNew{New: 10, Foo: 7, Nil: nil}, config) // config: PreferNew, Panicking compare (not caught)
	testutils.FatalUnless(t, didPanic == true, "")
	testutils.FatalUnless(t, panicValue == errPanic, "")
	testutils.FatalUnless(t, numCalls == 1, "")

	configGeneral = errorCreationConfig{}
	parseFlagArgs(&configGeneral, LetComparisonFunctionPanic, EnsureDataIsNotReplaced)
	config = configGeneral.config_OldData
	m = maps.Clone(startMap)

	ret = fillMapFromStruct(&m, &struct {
		Nil []int
		Foo int
	}{Nil: nil, Foo: -1}, config) // config: PreferNew, Default compare (panics not caught)
	testutils.FatalUnless(t, mapsEq(m, ParamMap{"Nil": []int{}, "Foo": -1, "TypedNil": (*int)(nil), "Inc": incomp{}}), "") // Note: two of these comparisons are for incomparable types.
	testutils.FatalUnless(t, len(ret) == 1, "%v", ret)
}

func TestFillMapFromStruct_PreferOld(t *testing.T) {
	var m1 ParamMap = ParamMap{}
	var m2 ParamMap = ParamMap{"Foo": 5}

	var config errorCreationConfig
	parseFlagArgs(&config, PreferPreviousData)
	var configPreferOld config_OldData = config.config_OldData

	ret := fillMapFromStruct(&m2, &struct{}{}, configPreferOld)
	testutils.FatalUnless(t, utils.CompareMaps(m2, ParamMap{"Foo": 5}), "")
	testutils.FatalUnless(t, ret == nil, "")

	ret = fillMapFromStruct(&m1, &struct{ Foo int }{Foo: 6}, configPreferOld)
	testutils.FatalUnless(t, utils.CompareMaps(m1, ParamMap{"Foo": 6}), "")
	testutils.FatalUnless(t, ret == nil, "")

	ret = fillMapFromStruct(&m1, &struct {
		Foo any
		Bar any
	}{}, configPreferOld)
	testutils.FatalUnless(t, utils.CompareMaps(m1, ParamMap{"Foo": 6, "Bar": nil}), "")
	testutils.FatalUnless(t, ret == nil, "")
}

func TestFillMapFromStruct_PreferNew(t *testing.T) {
	var m1 ParamMap = ParamMap{}
	var m2 ParamMap = ParamMap{"Foo": 5}

	var config errorCreationConfig
	parseFlagArgs(&config, ReplacePreviousData)
	var configPreferOld config_OldData = config.config_OldData

	ret := fillMapFromStruct(&m2, &struct{}{}, configPreferOld)
	testutils.FatalUnless(t, utils.CompareMaps(m2, ParamMap{"Foo": 5}), "")
	testutils.FatalUnless(t, ret == nil, "")

	ret = fillMapFromStruct(&m1, &struct{ Foo int }{Foo: 6}, configPreferOld)
	testutils.FatalUnless(t, utils.CompareMaps(m1, ParamMap{"Foo": 6}), "")
	testutils.FatalUnless(t, ret == nil, "")

	ret = fillMapFromStruct(&m1, &struct{ Foo, Bar any }{}, configPreferOld)
	testutils.FatalUnless(t, utils.CompareMaps(m1, ParamMap{"Foo": nil, "Bar": nil}), "")
	testutils.FatalUnless(t, ret == nil, "")
}

// Test that fillMapFromStruct honors our struct embedding rules and picks up the correct value.
func TestFillMapFromStruct_CorrectVal(t *testing.T) {

	var m ParamMap
	type intContainer struct{ X int }
	type wrappedIntContainer struct{ intContainer }
	type yContainer struct{ Y int }
	type wrappedYContainer struct{ yContainer }
	type zContainer struct{ Z int }

	type S struct {
		wrappedIntContainer
		intContainer
		X string
		wrappedYContainer
		zContainer
		U uint
	}

	var s S = S{
		X: "X",
		// intcontainer and wrappedIntContainer do not matter
		wrappedYContainer: wrappedYContainer{yContainer{10}},
		zContainer:        zContainer{20},
		U:                 uint(30),
	}

	ret := fillMapFromStruct(&m, &s, config_OldData{})
	testutils.FatalUnless(t, ret == nil, "unexpected error %v", ret)
	testutils.FatalUnless(t, utils.CompareMaps(m, ParamMap{
		"X": "X",
		"Y": int(10),
		"Z": int(20),
		"U": uint(30),
	}), "Unexpecter map value %v", m)

}
