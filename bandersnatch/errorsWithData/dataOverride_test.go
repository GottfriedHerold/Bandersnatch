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
	mergeMaps(&m2, nil, configPreferOld)
	testutils.FatalUnless(t, utils.CompareMaps(m2, ParamMap{"Foo": 5}), "")
	testutils.FatalUnless(t, ret == nil, "")
	mergeMaps(&m1, ParamMap{"Foo": 6}, configPreferOld)
	testutils.FatalUnless(t, utils.CompareMaps(m1, ParamMap{"Foo": 6}), "")
	testutils.FatalUnless(t, ret == nil, "")
	mergeMaps(&m1, ParamMap{"Foo": nil, "Bar": nil}, configPreferOld)
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
	var numCalls int
	panickingCompare := func(x, y any) bool { numCalls++; panic(errPanic) }
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

	//repeat with ReplacePreviousData
	parseFlagArgs(&configGeneral, ReplacePreviousData, EnsureDataIsNotReplaced_fun(panickingCompare))
	config = configGeneral.config_OldData
	m = maps.Clone(startMap)
	numCalls = 0
	ret = mergeMaps(&m, ParamMap{"New": 10, "Foo": 7, "Nil": nil}, config)
	testutils.FatalUnless(t, mapsEq(m, ParamMap{"Nil": nil, "TypedNil": (*int)(nil), "Foo": 7, "Inc": incomp{}, "New": 10}), "")
	// We expect 2 calls to the comparison function, from "Foo" and "Nil"
	testutils.FatalUnless(t, len(ret) == 2, "%v", ret)
	testutils.FatalUnless(t, numCalls == 2, "")
	// returned errors should wrap the panic value if it is an error
	testutils.FatalUnless(t, errors.Is(ret[0], errPanic), "")
	testutils.FatalUnless(t, errors.Is(ret[1], errPanic), "")

	//not catching panics:
	parseFlagArgs(&configGeneral, LetComparisonFunctionPanic)
	config = configGeneral.config_OldData
	m = maps.Clone(startMap)
	numCalls = 0
	didPanic, panicValue := testutils.CheckPanic2(mergeMaps, &m, ParamMap{"New": 10, "Foo": 7, "Nil": nil}, config)
	testutils.FatalUnless(t, didPanic == true, "")
	testutils.FatalUnless(t, panicValue == errPanic, "")
	testutils.FatalUnless(t, numCalls == 1, "")

}

/*

func TestFillMapFromStruct2(t *testing.T) {
	var m1 ParamMap = ParamMap{}
	var m2 ParamMap = ParamMap{"Foo": 5}
	type T1 struct{ Foo int }
	type T2 struct{ Bar uint }
	fillMapFromStruct(&T1{Foo: 4}, &m2, PreferPreviousData)
	testutils.FatalUnless(t, utils.CompareParamMaps(m2, ParamMap{"Foo": 5}), "")
	delete(m2, "Foo")
	fillMapFromStruct(&T1{Foo: 4}, &m2, PreferPreviousData)
	testutils.FatalUnless(t, utils.CompareParamMaps(m2, ParamMap{"Foo": 4}), "")
	fillMapFromStruct(&T1{Foo: 6}, &m2, ReplacePreviousData)
	testutils.FatalUnless(t, utils.CompareParamMaps(m2, ParamMap{"Foo": 6}), "")
	fillMapFromStruct(&T1{Foo: 6}, &m2, AssertDataIsNotReplaced)
	testutils.FatalUnless(t, utils.CompareParamMaps(m2, ParamMap{"Foo": 6}), "")

	fillMapFromStruct(&T2{Bar: 5}, &m1, AssertDataIsNotReplaced)
	fillMapFromStruct(&T2{Bar: 6}, &m1, ReplacePreviousData)
	fillMapFromStruct(&T2{Bar: uint(7)}, &m1, PreferPreviousData)
	testutils.FatalUnless(t, utils.CompareParamMaps(m1, ParamMap{"Bar": uint(6)}), "%v", m1)

	testutils.FatalUnless(t, testutils.CheckPanic(fillMapFromStruct[T2], &T2{Bar: 8}, &m1, AssertDataIsNotReplaced), "")

	var invalid flagPreviousDataTreatment
	testutils.FatalUnless(t, testutils.CheckPanic(fillMapFromStruct[T2], &T2{}, &m1, invalid), "No panic for invalid PreviousDataTreatment")

	var mNil ParamMap = nil
	fillMapFromStruct(&struct{}{}, &mNil, PreferPreviousData)
	testutils.FatalUnless(t, mNil != nil, "fillMapFromStruct does not work on nil maps")
	testutils.FatalUnless(t, len(mNil) == 0, "")
	mNil = nil
	fillMapFromStruct(&T1{Foo: 10}, &mNil, PreferPreviousData)
	testutils.FatalUnless(t, utils.CompareParamMaps(mNil, ParamMap{"Foo": 10}), "")

	type invalidType = struct{ *int }
	testutils.FatalUnless(t, testutils.CheckPanic(fillMapFromStruct[invalidType], &invalidType{}, &ParamMap{}, AssertDataIsNotReplaced), "No panic on invalid type")
}
*/

/*
func TestPrintPreviousDataTreatment(t *testing.T) {
	s1 := AssertDataIsNotReplaced.String()
	s2 := PreferPreviousData.String()
	s3 := ReplacePreviousData.String()
	s0 := flagPreviousDataTreatment{}.String()
	testutils.FatalUnless(t, s1 != "", "")
	testutils.FatalUnless(t, s2 != "", "")
	testutils.FatalUnless(t, s3 != "", "")
	testutils.FatalUnless(t, s0 != "", "")

	testutils.FatalUnless(t, s1 != s2, "")
	testutils.FatalUnless(t, s1 != s3, "")
	testutils.FatalUnless(t, s2 != s3, "")

	testutils.FatalUnless(t, s0 != s1, "")
	testutils.FatalUnless(t, s0 != s2, "")
	testutils.FatalUnless(t, s0 != s3, "")
}

*/

// OLD TEST

/*
func TestFillMapFromStruct(t *testing.T) {
	var m map[string]any
	var empty struct{}
	fillMapFromStruct(&empty, &m, AssertDataIsNotReplaced) // note: implied type parameter is unnamed
	if m == nil {
		t.Fatalf("E1")
	}
	if len(m) != 0 {
		t.Fatalf("E2")
	}
	m["x"] = 1
	type T1 struct {
		Name1 int
		Name2 string
		Name3 error // NOTE: interface type
	}
	type NestedT1 struct {
		T1
		Name1 uint // shadows T2.name1
		Name4 byte
	}
	var t1 T1 = T1{Name1: 1, Name2: "foo", Name3: io.EOF}
	fillMapFromStruct(&t1, &m, ReplacePreviousData)
	if m["x"] != 1 || m["Name1"] != int(1) || m["Name2"] != "foo" || m["Name3"] != io.EOF {
		t.Fatalf("E3")
	}
	t1copy, err := makeStructFromMap[T1](m)
	if err != nil {
		t.Fatalf("E4")
	}
	if t1copy != t1 {
		t.Fatalf("E5")
	}
	var m2 map[string]any
	t1other := T1{Name1: 2, Name2: "bar", Name3: nil}
	tEmbed := NestedT1{T1: t1other, Name1: 3, Name4: 4}
	fillMapFromStruct(&tEmbed, &m2, ReplacePreviousData)
	if m2["Name3"] != nil {
		t.Fatalf("E6")
	}
	if m2["Name1"] != uint(3) {
		t.Fatalf("E7")
	}
	_, ok := m2["T1"]
	if ok {
		t.Fatalf("E8")
	}
	t1EmbedRetrieved, _ := makeStructFromMap[NestedT1](m2)
	// Roundtrip will not work, because shadowed fields differ
	if t1EmbedRetrieved == tEmbed {
		t.Fatalf("E9")
	}
	// After zeroing shadowed field, it should behave like roundtrip
	tEmbed.T1.Name1 = 0
	if t1EmbedRetrieved != tEmbed {
		t.Fatalf("E10")
	}
}
*/
