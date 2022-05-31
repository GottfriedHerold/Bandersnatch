package bandersnatchErrors

import (
	"io"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

func TestGetStructMapConversionLookup(t *testing.T) {
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
	type structWithUnexported struct {
		unexported int
		Exported1  int
	}
	type nestedUnexported struct {
		structWithUnexported
		Exported2 int
	}
	REmpty := getStructMapConversionLookup(utils.TypeOfType[EmptyStruct]())
	R1 := getStructMapConversionLookup(utils.TypeOfType[T1]())
	R1again := getStructMapConversionLookup(utils.TypeOfType[T1]())
	RNested1 := getStructMapConversionLookup(utils.TypeOfType[NestedT1]())
	if len(REmpty) != 0 {
		t.Fatalf("getStructMapConversionLookup on empty struct gives non-empty list")
	}
	if !testutils.CheckSliceAlias(R1, R1again) {
		t.Fatalf("getStructMapConversionLookup does not return identical data when called twice")
	}
	if len(R1) != 3 {
		t.Fatalf("E1")
	}
	if len(RNested1) != 4 {
		t.Fatalf("E2")
	}
	if !testutils.CheckPanic(getStructMapConversionLookup, utils.TypeOfType[structWithUnexported]()) {
		t.Fatalf("E3")
	}
	if !testutils.CheckPanic(getStructMapConversionLookup, utils.TypeOfType[nestedUnexported]()) {
		t.Fatalf("E4")
	}
	if !testutils.CheckPanic(getStructMapConversionLookup, utils.TypeOfType[error]()) {
		t.Fatalf("E5")
	}
	if !testutils.CheckPanic(getStructMapConversionLookup, utils.TypeOfType[int]()) {
		t.Fatalf("E6")
	}
	if !testutils.CheckPanic(getStructMapConversionLookup, nil) {
		t.Fatalf("E7")
	}
}

func TestMapStructConversion(t *testing.T) {
	var m map[string]any = make(map[string]any)
	type Empty struct{}
	_, err := makeStructFromMap[Empty](nil)
	if err != nil {
		t.Fatalf("Could not create empty struct from nil")
	}
	_, err = makeStructFromMap[Empty](m)
	if err != nil {
		t.Fatalf("Could not create empty struct from empty map")
	}

	type T0 struct {
		Name1 int
	}

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

	m["Name1"] = 5
	m["Name2"] = "foo"
	m["Name3"] = nil
	m["Name4"] = byte(10)

	somet0, err := makeStructFromMap[T0](m)
	if err != nil {
		t.Fatalf("E1")
	}
	somet1, err := makeStructFromMap[T1](m)
	if err != nil {
		t.Fatalf("E2")
	}
	_, err = makeStructFromMap[NestedT1](m)
	if err == nil { // We expect an error because of type mismatch uint vs int
		t.Fatalf("E3")
	}
	_ = somet0
	_ = somet1
	if somet1.Name1 != 5 || somet1.Name2 != "foo" || somet1.Name3 != nil {
		t.Fatalf("E4")
	}
	delete(m, "Name3")
	_, err = makeStructFromMap[T1](m)
	if err == nil {
		t.Fatalf("E5")
	}
	m["Name3"] = io.EOF
	somet1, err = makeStructFromMap[T1](m)
	if err != nil {
		t.Fatalf("E6: %v", err)
	}
	if somet1.Name3 != io.EOF {
		t.Fatalf("E7")
	}
	m["Name3"] = "some string, which does not satisfy the error interface"
	_, err = makeStructFromMap[T1](m)
	if err == nil {
		t.Fatalf("E8")
	}
	m["Name3"] = io.EOF
	m["Name1"] = uint(6)
	nested, err := makeStructFromMap[NestedT1](m)
	if err != nil {
		t.Fatalf("E9 %v", err)
	}
	if nested.Name1 != uint(6) {
		t.Fatalf("E10")
	}
	if nested.T1.Name1 != 0 {
		t.Fatalf("E11")
	}
	m["Name1"] = nil
	_, err = makeStructFromMap[NestedT1](m)
	if err == nil {
		t.Fatalf("E12")
	}
}

func TestFillMapFromStruct(t *testing.T) {
	var m map[string]any
	var empty struct{}
	fillMapFromStruct(&empty, &m) // note: implied type parameter is unnamed
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
	fillMapFromStruct(&t1, &m)
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
	fillMapFromStruct(&tEmbed, &m2)
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
