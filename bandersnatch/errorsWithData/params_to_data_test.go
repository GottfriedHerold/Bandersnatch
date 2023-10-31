package errorsWithData

/*

import (
	"io"
	"reflect"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

func TestGetStructMapConversionLookup(t *testing.T) {

	// Go does not support generic closures, so we take typ as reflect.Type rather than a generic parameter
	// (Alternatively, we could define it globally, but then we couldn't capture t)
	checkType := func(typ reflect.Type, expected []reflect.StructField) {

		typeName := utils.GetReflectName(typ)
		lookup, lookupError := getStructMapConversionLookup(typ)
		if lookupError != nil {
			t.Fatalf("Unexpected error reported by getStructMapConversionLookup for %v: %v", typeName, lookupError)
		}
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

	R1, e1 := getStructMapConversionLookup(utils.TypeOfType[T1]())
	R1again, e2 := getStructMapConversionLookup(utils.TypeOfType[T1]())

	testutils.FatalUnless(t, e1 == nil, "Unexpected error %v", e1)
	testutils.FatalUnless(t, e2 == nil, "Unexpected error %v", e2)

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

	_, err := getStructMapConversionLookup(utils.TypeOfType[structWithUnexported]())
	testutils.FatalUnless(t, err != nil, "No error, but we expected an error")

	_, err = getStructMapConversionLookup(utils.TypeOfType[nestedUnexported]())
	testutils.FatalUnless(t, err != nil, "No error, but we expected an error")

	_, err = getStructMapConversionLookup(utils.TypeOfType[error]())
	testutils.FatalUnless(t, err != nil, "No error, but we expected an error")

	_, err = getStructMapConversionLookup(utils.TypeOfType[int]())
	testutils.FatalUnless(t, err != nil, "No error, but we expected an error")

	_, err = getStructMapConversionLookup(utils.TypeOfType[Intbased]())
	testutils.FatalUnless(t, err != nil, "No error, but we expected an error")

	_, err = getStructMapConversionLookup(utils.TypeOfType[Dups]())
	testutils.FatalUnless(t, err != nil, "No error, but we expected an error")

	if !testutils.CheckPanic(getStructMapConversionLookup, nil) {
		t.Fatalf("No panic, but we expected to")
	}

}

func TestMapToStructConversion(t *testing.T) {
	var m map[string]any = make(map[string]any)
	type Empty struct{}
	_, err := makeStructFromMap[Empty](nil, MissingDataIsError)
	if err != nil {
		t.Fatalf("Could not create empty struct from nil")
	}
	_, err = makeStructFromMap[Empty](m, MissingDataIsError)
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

	m["Name1"] = 5 // int, not uint
	m["Name2"] = "foo"
	m["Name3"] = nil
	m["Name4"] = byte(10)

	somet0, err := makeStructFromMap[T0](m, MissingDataIsError)
	testutils.FatalUnless(t, err == nil, "Unexpected error : %v", err)

	somet1, err := makeStructFromMap[T1](m, MissingDataIsError)
	testutils.FatalUnless(t, err == nil, "Unexpected error : %v", err)

	_, err = makeStructFromMap[NestedT1](m, MissingDataIsError)
	// We expect an error because of type mismatch uint vs int
	testutils.FatalUnless(t, err != nil, "No error, but we expected one")

	// silence unused variable errors.
	_ = somet0
	_ = somet1

	testutils.FatalUnless(t, somet1 == T1{5, "foo", nil}, "Unexpected value for somet1: %v", somet1)

	delete(m, "Name3")
	_, err = makeStructFromMap[T1](m, MissingDataIsError)
	testutils.FatalUnless(t, err != nil, "No error, but expected one (parameter missing)")

	somet1, err = makeStructFromMap[T1](m, MissingDataAsZero)
	testutils.FatalUnless(t, err == nil, "Unexpected error: %v", err)
	testutils.FatalUnless(t, somet1 == T1{5, "foo", nil}, "unexpected value for somet1: %v", somet1)

	// make sure that using MissingDataAsZero does not modify the input map
	_, present := m["Name3"]
	testutils.FatalUnless(t, !present, "makeStructFromMap modified input map")

	m["Name3"] = io.EOF
	somet1, err = makeStructFromMap[T1](m, MissingDataIsError)
	testutils.FatalUnless(t, err == nil, "Unexpected error: %v", err)
	testutils.FatalUnless(t, somet1.Name3 == io.EOF, "Unexpected value for somet1: %v", somet1)

	m["Name3"] = "some string, which does not satisfy the error interface"
	_, err = makeStructFromMap[T1](m, MissingDataAsZero)
	testutils.FatalUnless(t, err != nil, "No error, but we expected one")
	_, err = makeStructFromMap[T1](m, MissingDataIsError)
	testutils.FatalUnless(t, err != nil, "No error, but we expected one")

	m["Name3"] = io.EOF
	m["Name1"] = uint(6)
	nested, err := makeStructFromMap[NestedT1](m, MissingDataIsError)
	testutils.FatalUnless(t, err == nil, "Unexpected error: %v", err)

	testutils.FatalUnless(t, nested.Name1 == uint(6), "Unexpected value for nested %v", nested)
	// Note: While we expected nested.T1.Name to be zero-initialized (what else?), this is actually not enforced by our specification.
	// So we could in principle tolerate failure of this test.
	testutils.FatalUnless(t, nested.T1.Name1 == int(0), "Unexpected value for shadowed values. Nested == %v", nested)

	m["Name1"] = nil
	_, err = makeStructFromMap[NestedT1](m, MissingDataIsError)
	testutils.FatalUnless(t, err != nil, "No error, but we expected one")
	delete(m, "Name1")
	_, err = makeStructFromMap[NestedT1](m, MissingDataIsError)
	testutils.FatalUnless(t, err != nil, "No error, but we expected one")
	nested, err = makeStructFromMap[NestedT1](m, MissingDataAsZero)
	testutils.FatalUnless(t, err == nil, "Unexpected error: %v", err)
	testutils.FatalUnless(t, nested.Name1 == 0, "Unexpected value for nested.Name1. nested == %v", nested)
	testutils.FatalUnless(t, nested.Name3 == io.EOF && nested.Name4 == 10 && nested.Name2 == "foo", "Unexptected value for nested :%v", nested)

}
func TestEnsureCanMakeStructFromParams(t *testing.T) {
	var (
		mEmpty        ParamMap = make(ParamMap)
		mSomeArgs     ParamMap = ParamMap{"Arg1": 5, "Arg2": nil}
		mNilInterface ParamMap = ParamMap{"Arg": nil}
	)
	type EmbeddedInt = int
	var issue error = ensureCanMakeStructFromParameters[struct{}](&mEmpty, MissingDataIsError)
	testutils.FatalUnless(t, issue == nil, "%v", issue)
	testutils.FatalUnless(t, len(mEmpty) == 0, "map modified: %v", mEmpty)
	issue = ensureCanMakeStructFromParameters[struct{}](&mSomeArgs, MissingDataIsError)
	testutils.FatalUnless(t, issue == nil, "%v", issue)
	issue = ensureCanMakeStructFromParameters[struct{ Arg *int }](&mNilInterface, MissingDataIsError)
	testutils.FatalUnless(t, issue == nil, "%v", issue)
	issue = ensureCanMakeStructFromParameters[struct{ EmbeddedInt }](&ParamMap{"EmbeddedInt": 5}, MissingDataIsError)
	testutils.FatalUnless(t, issue == nil, "%v", issue)
	issue = ensureCanMakeStructFromParameters[struct{ EmbeddedInt }](&ParamMap{"EmbeddedInt": uint(5)}, MissingDataIsError)
	testutils.FatalUnless(t, issue != nil, "") // Wrong type)
	issue = ensureCanMakeStructFromParameters[struct{ *EmbeddedInt }](&ParamMap{"EmbeddedInt": new(int)}, MissingDataIsError)
	testutils.FatalUnless(t, issue == nil, "%v", issue)

	type T1 struct {
		X int
		Y error
	}
	type T2 struct {
		T1
		X uint
	}

	issue = ensureCanMakeStructFromParameters[T2](&ParamMap{"X": uint(0), "Y": io.EOF}, MissingDataIsError)
	testutils.FatalUnless(t, issue == nil, "%v", issue)
	issue = ensureCanMakeStructFromParameters[T2](&ParamMap{"X": uint(0), "Y": nil, "Z": "foo"}, MissingDataIsError)
	testutils.FatalUnless(t, issue == nil, "%v", issue)
	issue = ensureCanMakeStructFromParameters[T2](&ParamMap{"X": int(0), "Y": io.EOF}, MissingDataIsError)
	testutils.FatalUnless(t, issue != nil, "") // wrong type
	issue = ensureCanMakeStructFromParameters[T2](&ParamMap{"X": int(0), "Y": io.EOF}, MissingDataAsZero)
	testutils.FatalUnless(t, issue != nil, "") // wrong type
	issue = ensureCanMakeStructFromParameters[T2](&ParamMap{"X": uint(0), "Z": io.EOF}, MissingDataIsError)
	testutils.FatalUnless(t, issue != nil, "") // missing Y
	issue = ensureCanMakeStructFromParameters[T2](&ParamMap{"X": uint(0), "Z": io.EOF}, MissingDataAsZero)
	testutils.FatalUnless(t, issue == nil, "") // missing Y, but filled in with 0

	issue = ensureCanMakeStructFromParameters[T2](&ParamMap{"Y": io.EOF, "Z": 0}, MissingDataIsError)
	testutils.FatalUnless(t, issue != nil, "") // missing X
	issue = ensureCanMakeStructFromParameters[T2](&ParamMap{"Y": io.EOF, "Z": 0}, MissingDataAsZero)
	testutils.FatalUnless(t, issue == nil, "") // missing X, but filled in with 0
	issue = ensureCanMakeStructFromParameters[T2](&ParamMap{"X": uint(0), "Y": "foo", "Z": 0}, MissingDataIsError)
	testutils.FatalUnless(t, issue != nil, "") // Y is no error
	issue = ensureCanMakeStructFromParameters[T2](&ParamMap{"X": uint(0), "Y": "foo", "Z": 0}, MissingDataAsZero)
	testutils.FatalUnless(t, issue != nil, "") // Y is no error

	testutils.FatalUnless(t, ensureCanMakeStructFromParameters[struct{ Arg int }](&mNilInterface, MissingDataIsError) != nil, "")
	testutils.FatalUnless(t, ensureCanMakeStructFromParameters[struct{ Arg int }](&mNilInterface, MissingDataAsZero) != nil, "")
}

*/
