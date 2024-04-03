package errorsWithData

import (
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

func TestCheckParamsForStruct(t *testing.T) {
	type EmptyStruct struct{}
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
	type t1 = T1
	type NestedT2_anon struct {
		t1
		Name4 byte
	}

	var EmptyList []string = []string{}
	var T1List []string = []string{"Name1", "Name3", "Name2"} // intentionally different order than in T1
	var NestedT1List []string = []string{"Name1", "Name3", "Name4", "Name2"}
	var NestedT2_anonList []string = []string{"Name1", "Name2", "Name3", "Name4"}
	CheckParametersForStruct_all[EmptyStruct](EmptyList)
	CheckParametersForStruct_all[T1](T1List)
	CheckParametersForStruct_all[NestedT1](NestedT1List)
	CheckParametersForStruct_all[NestedT2_anon](NestedT2_anonList)

	if !testutils.CheckPanic(CheckParametersForStruct_all[T1], NestedT1List) {
		t.Fatalf("T1")
	}
	if !testutils.CheckPanic(CheckParametersForStruct_all[NestedT1], T1List) {
		t.Fatalf("T2")
	}
}

func testcaseCheckParameterForStruct[T any](t *testing.T, fieldName string, expectedPanic bool) {
	didPanic, panicValue := testutils.CheckPanic2(CheckParameterForStruct[T], fieldName)
	typeName := utils.NameOfType[T]()
	if expectedPanic == false {
		testutils.FatalUnless(t, didPanic == expectedPanic, "CheckParameterForStruct did not behave as expected for %v and %v and panicked with %v", typeName, fieldName, panicValue)
	} else {
		testutils.FatalUnless(t, didPanic == expectedPanic, "CheckParameterForStruct did not behave as expected for %v and %v and did not panic", typeName, fieldName)
	}
}

func TestCheckParameterForStruct(t *testing.T) {
	type EmptyStruct struct{}
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
	type t1 = T1
	type NestedT2_anon struct {
		t1
		Name4 byte
	}
	type invalid struct {
		unexported int
		Exported   int
	}

	testcaseCheckParameterForStruct[EmptyStruct](t, "", true)
	testcaseCheckParameterForStruct[EmptyStruct](t, "foo", true)
	testcaseCheckParameterForStruct[EmptyStruct](t, "Foo", true)
	testcaseCheckParameterForStruct[T1](t, "Name1", false)
	testcaseCheckParameterForStruct[T1](t, "Name2", false)
	testcaseCheckParameterForStruct[T1](t, "Name3", false)
	testcaseCheckParameterForStruct[T1](t, "Name4", true)
	testcaseCheckParameterForStruct[T1](t, "name1", true)
	testcaseCheckParameterForStruct[T1](t, "", true)
	testcaseCheckParameterForStruct[NestedT1](t, "Name1", false)
	testcaseCheckParameterForStruct[NestedT1](t, "Name2", false)
	testcaseCheckParameterForStruct[NestedT1](t, "Name3", false)
	testcaseCheckParameterForStruct[NestedT1](t, "Name4", false)
	testcaseCheckParameterForStruct[NestedT1](t, "Name5", true)
	testcaseCheckParameterForStruct[NestedT1](t, "T1", true)
	testcaseCheckParameterForStruct[NestedT1](t, "", true)
	testcaseCheckParameterForStruct[NestedT1](t, "unexported", true)
	testcaseCheckParameterForStruct[t1](t, "Name1", false)
	testcaseCheckParameterForStruct[NestedT2_anon](t, "Name4", false)
	testcaseCheckParameterForStruct[NestedT2_anon](t, "Name1", false)
	testcaseCheckParameterForStruct[NestedT2_anon](t, "Name2", false)
	testcaseCheckParameterForStruct[NestedT2_anon](t, "Name3", false)
	testcaseCheckParameterForStruct[NestedT2_anon](t, "t1", true)
	testcaseCheckParameterForStruct[NestedT2_anon](t, "T1", true)

	testcaseCheckParameterForStruct[invalid](t, "unexported", true)
	testcaseCheckParameterForStruct[invalid](t, "Exported", true) // also fails!
}

func testcaseCheckIsSubtype[T1, T2 any](t *testing.T, expectedPanic bool) {
	didPanic, panicValue := testutils.CheckPanic2(CheckIsSubtype[T1, T2])
	typeName1 := utils.NameOfType[T1]()
	typeName2 := utils.NameOfType[T2]()
	if expectedPanic == false {
		testutils.FatalUnless(t, didPanic == expectedPanic, "CheckIsSubtype did not behave as expected for %v and %v and panicked with %v", typeName1, typeName2, panicValue)
	} else {
		testutils.FatalUnless(t, didPanic == expectedPanic, "CheckIsSubtype did not behave as expected for %v and %v and did not panic", typeName1, typeName2)
	}
}

func TestCheckIsSubtype(t *testing.T) {
	type EmptyStruct struct{}
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
	type t1 = T1
	type NestedT2_anon struct {
		t1
		Name4 byte
	}
	type invalid struct {
		unexported int
		Exported   int
	}

	// Note: the true/false - arg denotes whether we expect a panic, i.e. the negation(!) of whether type1 is a type2
	testcaseCheckIsSubtype[EmptyStruct, EmptyStruct](t, false)
	testcaseCheckIsSubtype[EmptyStruct, T1](t, false)
	testcaseCheckIsSubtype[EmptyStruct, NestedT1](t, false)
	testcaseCheckIsSubtype[EmptyStruct, t1](t, false)
	testcaseCheckIsSubtype[EmptyStruct, NestedT2_anon](t, false)
	testcaseCheckIsSubtype[EmptyStruct, invalid](t, true) // !

	testcaseCheckIsSubtype[struct{ X int }, struct{ X uint }](t, false)
	testcaseCheckIsSubtype[invalid, invalid](t, true)
	testcaseCheckIsSubtype[T1, NestedT1](t, false)
	testcaseCheckIsSubtype[NestedT1, T1](t, true)
	testcaseCheckIsSubtype[NestedT2_anon, NestedT1](t, false)
	testcaseCheckIsSubtype[NestedT1, NestedT2_anon](t, false)
	testcaseCheckIsSubtype[struct{ Name4 string }, NestedT2_anon](t, false)
}
