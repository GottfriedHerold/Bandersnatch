package errorsWithData

import (
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
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
