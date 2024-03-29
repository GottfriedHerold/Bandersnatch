package utils

import (
	"reflect"
	"testing"
)

func TestIncomparability(t *testing.T) {
	type dummyType struct {
		_ [4]uint64
	}

	type dummyType_incomparable struct {
		MakeIncomparable
		_ [4]uint64
	}

	type dummyType_incomparable2 struct {
		MakeIncomparable
		dummyType
	}

	var type_comparable reflect.Type = TypeOfType[dummyType]()
	var type_incomparable reflect.Type = TypeOfType[dummyType_incomparable]()
	var type_incomparable2 reflect.Type = TypeOfType[dummyType_incomparable2]()

	// type anyComparable[T comparable] interface{}

	// expanded below, we avoid testutils.FatalUnless due to dependency cycles
	/*
		testutils.FatalUnless(t, type_comparable.Comparable(), "dummy type incomparable")
		testutils.FatalUnless(t, !type_incomparable.Comparable(), "dummy type not incomparable")
		testutils.FatalUnless(t, !type_incomparable2.Comparable(), "dummy type2 not incomparable")
		testutils.FatalUnless(t, type_comparable.Size() == type_incomparable.Size(), "MakeIncomparable changes memory size")
		testutils.FatalUnless(t, type_comparable.Size() == type_incomparable2.Size(), "MakeIncomparable changes memory size (2)")
	*/

	if !type_comparable.Comparable() {
		t.Fatalf("dummy type incomparable")
	}
	if type_incomparable.Comparable() {
		t.Fatalf("dummy type not incomparable")
	}
	if type_incomparable2.Comparable() {
		t.Fatalf("dummy type2 not incomparable")
	}
	if type_comparable.Size() != type_incomparable.Size() {
		t.Fatalf("MakeIncomparable changes memory size")
	}
	if type_comparable.Size() != type_incomparable2.Size() {
		t.Fatalf("MakeIncomaprable changes memory size (2)")
	}
}
