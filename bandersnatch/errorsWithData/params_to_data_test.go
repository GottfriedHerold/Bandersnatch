package errorsWithData

import (
	"io"
	"reflect"
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

func TestGetStructMapConversionLookup(t *testing.T) {

	// Go does not support generic closures, so we take typ as reflect.Type rather than a generic parameter
	// (Alternatively, we could define it globally, but then we couldn't capture t)
	checkType := func(typ reflect.Type, expected []reflect.StructField) {

		typeName := utils.GetReflectName(typ)
		lookup := getStructMapConversionLookup(typ)
		if len(lookup) != len(expected) {
			t.Fatalf("Wrong number of returned arguments for %v: got %v, expected %v", typeName, len(lookup), len(expected))
		}

		for _, expectedField := range expected {
			name := expectedField.Name // we assume each name in expected is unique
			// find match in lookup
			var i int = -1
			for j, f := range lookup {
				if f.Name == name {
					i = j
					break
				}
			}
			testutils.FatalUnless(t, i != -1, "No field named %v returned by getStructMapConversionLookup for %v", name, typeName)
			relevantField := lookup[i]
			// check that relevantField and expectedField match in the relevant categories (We don't check all entries of reflect.StructField)
			testutils.FatalUnless(t, relevantField.Name == expectedField.Name, "Cannot happen")
			testutils.FatalUnless(t, relevantField.IsExported() == true, "getStructMapConversionLookup returned non-exported field for %v", typeName)
			testutils.FatalUnless(t, relevantField.Anonymous == expectedField.Anonymous, "getStructMapConversionLookup returned wrong value for Anonymous for %v.%v", typeName, name)
			testutils.FatalUnless(t, relevantField.Type == expectedField.Type, "getStructMapConversionLookup returned wrong value for type for %v. Expected: %v, Got %v", typeName, expectedField.Type, relevantField.Type)

			byName, ok := typ.FieldByName(name)
			testutils.FatalUnless(t, ok, "Should not be possible")
			IndexViaName := byName.Index
			testutils.FatalUnless(t, utils.CompareSlices(IndexViaName, relevantField.Index), "Index does not match usual Go lookup rules")
			testutils.FatalUnless(t, utils.CompareSlices(IndexViaName, expectedField.Index), "getStructMapConversionLookup returned unexpected Index for %v", typeName)
		}
	}

	intType := utils.TypeOfType[int]()
	stringType := utils.TypeOfType[string]()
	errorType := utils.TypeOfType[error]()
	uintType := utils.TypeOfType[uint]()
	byteType := utils.TypeOfType[byte]()

	// Note: go-staticcheck may wrongly complain about unused types here. This was reported and confirmed as a bug of staticcheck.

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
	type Intbased int
	type containsInt struct {
		Intbased
	}
	type containsIntPtr struct {
		*Intbased
	}
	type WrappedT1 struct {
		T1
	}
	type Dups struct {
		T1
		WrappedT1
	}

	R1 := getStructMapConversionLookup(utils.TypeOfType[T1]())
	R1again := getStructMapConversionLookup(utils.TypeOfType[T1]())

	if !testutils.CheckSliceAlias(R1, R1again) {
		t.Fatalf("getStructMapConversionLookup does not return identical data when called twice with same argument")
	}

	IntbasedType := utils.TypeOfType[Intbased]()

	checkType(utils.TypeOfType[EmptyStruct](), []reflect.StructField{})

	checkType(utils.TypeOfType[T1](), []reflect.StructField{
		{Name: "Name1", Type: intType, Index: []int{0}},
		{Name: "Name2", Type: stringType, Index: []int{1}},
		{Name: "Name3", Type: errorType, Index: []int{2}},
	})

	checkType(utils.TypeOfType[NestedT1](), []reflect.StructField{
		{Name: "Name1", Type: uintType, Index: []int{1}},
		{Name: "Name2", Type: stringType, Index: []int{0, 1}},
		{Name: "Name3", Type: errorType, Index: []int{0, 2}},
		{Name: "Name4", Type: byteType, Index: []int{2}},
	})

	checkType(utils.TypeOfType[containsInt](), []reflect.StructField{
		{Name: "Intbased", Type: IntbasedType, Index: []int{0}, Anonymous: true},
	})

	checkType(utils.TypeOfType[containsIntPtr](), []reflect.StructField{
		{Name: "Intbased", Type: reflect.PointerTo(IntbasedType), Index: []int{0}, Anonymous: true},
	})

	if !testutils.CheckPanic(getStructMapConversionLookup, utils.TypeOfType[structWithUnexported]()) {
		t.Fatalf("No panic, but we expected to")
	}
	if !testutils.CheckPanic(getStructMapConversionLookup, utils.TypeOfType[nestedUnexported]()) {
		t.Fatalf("No panic, but we expected to")
	}
	if !testutils.CheckPanic(getStructMapConversionLookup, utils.TypeOfType[error]()) {
		t.Fatalf("No panic, but we expected to")
	}
	if !testutils.CheckPanic(getStructMapConversionLookup, utils.TypeOfType[int]()) {
		t.Fatalf("No panic, but we expected to")
	}
	if !testutils.CheckPanic(getStructMapConversionLookup, utils.TypeOfType[Intbased]()) {
		t.Fatalf("No panic, but we expected to")
	}
	if !testutils.CheckPanic(getStructMapConversionLookup, nil) {
		t.Fatalf("No panic, but we expected to")
	}

	if !testutils.CheckPanic(getStructMapConversionLookup, utils.TypeOfType[Dups]()) {
		t.Fatalf("No panic, but we expected to (for case where promoted fields come through different paths)")
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

func TestCanMakeStructFromParams(t *testing.T) {
	var (
		mEmpty        ParamMap = make(ParamMap)
		mSomeArgs     ParamMap = ParamMap{"Arg1": 5, "Arg2": nil}
		mNilInterface ParamMap = ParamMap{"Arg": nil}
	)
	type EmbeddedInt = int
	var issue error = canMakeStructFromParameters[struct{}](mEmpty)
	testutils.FatalUnless(t, issue == nil, "%v", issue)
	issue = canMakeStructFromParameters[struct{}](mSomeArgs)
	testutils.FatalUnless(t, issue == nil, "%v", issue)
	issue = canMakeStructFromParameters[struct{ Arg *int }](mNilInterface)
	testutils.FatalUnless(t, issue == nil, "%v", issue)
	issue = canMakeStructFromParameters[struct{ EmbeddedInt }](ParamMap{"EmbeddedInt": 5})
	testutils.FatalUnless(t, issue == nil, "%v", issue)
	issue = canMakeStructFromParameters[struct{ EmbeddedInt }](ParamMap{"EmbeddedInt": uint(5)})
	testutils.FatalUnless(t, issue != nil, "") // Wrong type
	issue = canMakeStructFromParameters[struct{ *EmbeddedInt }](ParamMap{"EmbeddedInt": new(int)})
	testutils.FatalUnless(t, issue == nil, "%v", issue)

	type T1 struct {
		X int
		Y error
	}
	type T2 struct {
		T1
		X uint
	}

	issue = canMakeStructFromParameters[T2](ParamMap{"X": uint(0), "Y": io.EOF})
	testutils.FatalUnless(t, issue == nil, "%v", issue)
	issue = canMakeStructFromParameters[T2](ParamMap{"X": uint(0), "Y": nil, "Z": "foo"})
	testutils.FatalUnless(t, issue == nil, "%v", issue)
	issue = canMakeStructFromParameters[T2](ParamMap{"X": int(0), "Y": io.EOF})
	testutils.FatalUnless(t, issue != nil, "") // wrong type
	issue = canMakeStructFromParameters[T2](ParamMap{"X": uint(0), "Z": io.EOF})
	testutils.FatalUnless(t, issue != nil, "") // missing Y
	issue = canMakeStructFromParameters[T2](ParamMap{"Y": io.EOF, "Z": 0})
	testutils.FatalUnless(t, issue != nil, "") // missing X
	issue = canMakeStructFromParameters[T2](ParamMap{"X": uint(0), "Y": "foo", "Z": 0})
	testutils.FatalUnless(t, issue != nil, "") // Y is no error

	testutils.FatalUnless(t, canMakeStructFromParameters[struct{ Arg int }](mNilInterface) != nil, "")
}
