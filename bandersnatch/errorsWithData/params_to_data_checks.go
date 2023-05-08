package errorsWithData

import (
	"fmt"

	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

// This file contains check functions that take a type and purported field names
// and panics if the field names and the types don't match
//
// The intented usage of these functions is to create tests and
// add this in places in the code where you have a dependency on
// a struct having precisely a given set of field names.
// The purpose is to have failures after refactoring if you forget to change things.

// CheckParametersForStruct_all[StructType](fieldNames) checks whether the name of the fields coincides with
// the slice of fieldNames. Note that we require equality (up to order), i.e. the list of fieldNames is exhaustive;
//
// All fields of StructType must be exported and shadowed names must only appear once.
// This is intented to be used in init-routines or tests accompanying places in the code
// where we assume that a certain struct has exactly a given set of field names.
// The purpose is to create guards in the code / tests. It panics on failure.
func CheckParametersForStruct_all[StructType any](fieldNames []string) {

	// quadratic, but I don't care.
	for i := 0; i < len(fieldNames); i++ {
		for j := i + 1; j < len(fieldNames); j++ {
			if fieldNames[i] == fieldNames[j] {
				panic(fmt.Errorf(ErrorPrefix+"In call to CheckParametersForStruct, the given list of field names contains a duplicate: %v", fieldNames[i]))
			}
		}
	}
	allExpectedFields, err := getStructMapConversionLookup(utils.TypeOfType[StructType]())
	if err != nil {
		panic(err)
	}
	for _, expectedField := range allExpectedFields {
		expectedFieldName := expectedField.Name
		found := false
		for _, givenFieldName := range fieldNames {
			if expectedFieldName == givenFieldName {
				found = true
				break
			}
		}
		if !found {
			panic(fmt.Errorf(ErrorPrefix+"Field named %v required is not contained among the given list", expectedFieldName))
		}
	}
	// We intentionally make that check *after* the above checks.
	if len(allExpectedFields) != len(fieldNames) {
		panic(fmt.Errorf(ErrorPrefix + "list of given field names contains more field names than required"))
	}
}

// CheckParameterForStruct[StructType](fieldNames) checks whether the name of the (exported) fields contains the given
// fieldName. This is intented to be used in init-routines or tests accompanying places in the code
// where we assume that a certain struct contains a field of a given name.
// The purpose is to create guards in the code / tests. It panics on failure.
func CheckParameterForStruct[StructType any](fieldName string) {
	// No need to check that fieldName is a valid exported name. The function will fail anyway if this is not satisfied.
	allExpectedFields, err := getStructMapConversionLookup(utils.TypeOfType[StructType]())
	if err != nil {
		panic(err)
	}
	for _, expectedField := range allExpectedFields {
		if expectedField.Name == fieldName {
			return
		}
	}
	panic(fmt.Errorf(ErrorPrefix+"The given struct does not contain an exported field named %v", fieldName))
}

// CheckIsSubtype checks that both StructType1 and StructType2 are valid for errorsWithData and the exported fields of StructType1 are a subset of those of StructType2.
// Note that Struct embedding StructType1 in the definition of StructType2 may be preferred to this approach.
//
// CheckIsSubtype only cares about the names of the fields. It completely ignores the types.
// The purpose is to create guards in the code / tests. It panics on failure.
func CheckIsSubtype[StructType1 any, StructType2 any]() {
	allExpectedFields1, err := getStructMapConversionLookup(utils.TypeOfType[StructType1]())
	if err != nil {
		panic(err)
	}
	for _, expectedField1 := range allExpectedFields1 {
		CheckParameterForStruct[StructType2](expectedField1.Name)
	}
}
