package errorsWithData

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

// this file contains functionality used for translating maps[string]any <-> struct{...}
// The translation works by treating a struct Foo struct{A int; B byte} as a map m of type
// map[string]any where m["A"] has type int, m["B"] has type byte.
// Some care needs to be taken with non-exported, embedded and shadowed fields:

// lookupStructMapConversion is a lookup table (only depending of T) that contains the
// relevant data for converting an instance of a struct T to a map[string]any.
type lookupStructMapConversion = []reflect.StructField

var enforcedDataTypeMapMutex sync.RWMutex
var enforcedDataTypeMap map[reflect.Type]lookupStructMapConversion = make(map[reflect.Type]lookupStructMapConversion)

// getStructMapConversionLookup obtains a lookup table for converting a struct data type (passed as reflect.Type)
// to a map[string]any. Repeated calls with the same argument give identical results (slice with same backing array)
//
// This essentially is just reflect.VisibleFields with some extra checks upfront and skipping embedded fields.
func getStructMapConversionLookup(tType reflect.Type) (ret lookupStructMapConversion) {
	// Note: We cache the result in a global map. This implies
	// that e.g. the order of entries in the returned struct is consistent.
	// (Note sure whether this is needed and this is probably true anyway, but it is not guaranteed by the spec of reflect.VisibleFields otherwise)

	// Check if we alread have the table in the cache.
	enforcedDataTypeMapMutex.RLock()
	ret, ok := enforcedDataTypeMap[tType]
	enforcedDataTypeMapMutex.RUnlock()
	if ok {
		return
	}
	// We get here if we did not find it in the cache.

	// Make some sanity checks: This function only makes sense for struct types.
	if tType == nil {
		panic(ErrorPrefix + "called getStructMapConversionLookup with nil argment")
	}
	if tType.Kind() != reflect.Struct {
		panic(ErrorPrefix + "using getStructMapConversionLookup with non-struct type")
	}

	// The intended result will be a subset of allVisibleFields
	allVisibleFields := reflect.VisibleFields(tType)
	ret = make(lookupStructMapConversion, 0, len(allVisibleFields))

	// ensure everything is exported and filter out embedded fields
outer_loop:
	for _, visibleField := range allVisibleFields {
		// .Anonymous denotes whether the field is embedded (a bit of a misnomer).
		// for an embedded field, reflect.VisibleFields returns both the name of the embedded type and its included field
		// We only want the latter, so we skip here.
		if visibleField.Anonymous {
			continue
		}
		if !visibleField.IsExported() {
			panic(ErrorPrefix + "using errorsWithData with struct type containing unexported fields")
		}

		for pos, existingField := range ret {
			// Shadowing:
			if existingField.Name == visibleField.Name {
				// shorter length of Index is the one that get precedence (according to Go's shadowing rules).
				// In case of ambiguity, we panic:
				if len(existingField.Index) == len(visibleField.Index) {
					panic(ErrorPrefix + "using errorsWithData with struct type that has an ambiguous promoted field")
				}
				// ensure that for existingField.Index and visibleField.Index, one is a prefix of the other.
				// Note that this is stronger that the usual rules, which just compare len of Index.
				if len(existingField.Index) < len(visibleField.Index) {
					for i := 0; i < len(existingField.Index); i++ {
						if existingField.Index[i] != visibleField.Index[i] {
							panic(ErrorPrefix + "using errorsWithData with struct type that has a promoted field through different embedded fields")
						}
					}
					continue outer_loop // don't add visible field, the existing one takes precedence
				} else { // len(existingField.Index) > len(visibleField.Index)
					for i := 0; i < len(visibleField.Index); i++ {
						if existingField.Index[i] != visibleField.Index[i] {
							panic(ErrorPrefix + "using errorsWithData with struct type that has a promoted field through different embedded fields")
						}
					}
					ret[pos] = visibleField // overwrite previous entry
					continue outer_loop     // to skip the append below
				}
			}
		}

		ret = append(ret, visibleField)
	}

	// Write ret into the cache. We RW-lock the cache for this.
	enforcedDataTypeMapMutex.Lock()
	defer enforcedDataTypeMapMutex.Unlock()
	// We need to check if some other goroutine already filled the cache in the meantime.
	_, ok = enforcedDataTypeMap[tType]
	if ok {
		ret = enforcedDataTypeMap[tType]
	} else {
		enforcedDataTypeMap[tType] = ret
	}
	return
}

// CheckParametersForStruct_exact[StructType](fieldNames) checks whether the name of the fields coincides with
// the slice of fieldNames. Note that we require equality, i.e. the list of fieldNames is exhaustive; all fields of StructType must be exported.
// This is intented to be used in init-routines or tests accompanying places in the code
// where we assume that a certain struct has exactly a given set of field names.
// The purpose is to create guards in the code. It panics on failure.
func CheckParametersForStruct_exact[StructType any](fieldNames []string) {
	allExpectedFields := getStructMapConversionLookup(utils.TypeOfType[StructType]())
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
// The purpose is to create guards in the code. It panics on failure.
func CheckParameterForStruct[StructType any](fieldName string) {
	allExpectedFields := getStructMapConversionLookup(utils.TypeOfType[StructType]())
	for _, expectedField := range allExpectedFields {
		if expectedField.Name == fieldName {
			return
		}
	}
	panic(fmt.Errorf(ErrorPrefix+"The given struct does not contain an exported field named %v", fieldName))
}

// CheckIsSubtype checks that both StructType1 and StructType2 only contain exported names and those of StructType1 are a subset of those of StructType2.
// Note that Struct embedding StructType1 in the definition of StructType2 may be preferred to this approach.
//
// CheckIsSubtype only cares about the names of the fields. It completely ignores the types.
// The purpose is to create guards in the code. It panics on failure.
func CheckIsSubtype[StructType1 any, StructType2 any]() {
	allExpectedFields1 := getStructMapConversionLookup(utils.TypeOfType[StructType1]())
	for _, expectedField1 := range allExpectedFields1 {
		CheckParameterForStruct[StructType2](expectedField1.Name)
	}
}

// canMakeStructFromParametersInError checks whether e actually contains data for all fields of a struct of type StructType.
// This is called after creating an error with T==StructType.
// e == nil is treated as error without any data.
func canMakeStructFromParametersInError[StructType any](e error) (err error) {
	structType := utils.TypeOfType[StructType]()
	allExpectedFields := getStructMapConversionLookup(structType)
	m := GetAllParametersFromError(e)
	for _, expectedField := range allExpectedFields {
		// Special case e==nil for better error message.
		// If e == nil, GetParameterFromError returns (nil, false) so any iteration of the for loop ends up here.
		// (We check this inside the loop, because expectedFields may be an empty list)
		if e == nil {
			err = fmt.Errorf(ErrorPrefix+"nil error does not contain any parameters, but a parameter named %v was requested", expectedField.Name)
			return
		}

		mapEntry, exists := m[expectedField.Name]
		if !exists {
			err = fmt.Errorf(ErrorPrefix+"error %v does not contain a parameter named %v, neccessary to export data a a struct of type %v", e, expectedField.Name, structType)
			return
		}
		// requires special casing due to what I consider a design error in reflection.
		// See https://github.com/golang/go/issues/51649 for an actual discussion to change it for Go1.19 or later.
		if mapEntry == nil {
			if utils.IsNilable(expectedField.Type) {
				continue
			} else {
				err = fmt.Errorf(ErrorPrefix+"error %v contains a parameter %v that is set to nil. This cannot be used to construct a struct of type %v",
					e, expectedField.Name, structType)
				return
			}
		}
		mapEntryType := reflect.TypeOf(mapEntry)
		// interface types as fields in StructType need special handling, because reflect.TypeOf(mapEntry) contains the dynamic type.
		if expectedField.Type.Kind() == reflect.Interface {
			if !mapEntryType.AssignableTo(expectedField.Type) {
				err = fmt.Errorf(ErrorPrefix+"error %v has parameter %v set to a value %v; cannot export that in as struct of type %v, because that that value is not assignable to the intended field of interface type", e, expectedField.Name, mapEntry, structType)
				return
			}
		} else { // field of non-interface type in T: We require the types to match exactly.
			if mapEntryType != expectedField.Type {
				err = fmt.Errorf(ErrorPrefix+" error %v has parameter %v of wrong type to construct struct of type %v. Value is %v of type %v, expected %v",
					e, expectedField.Name, structType, mapEntry, mapEntryType, expectedField.Type)
				return
			}
		}
	}
	return nil
}

// Note: returning an error rather than panicking is somewhat difficult here, because several of the called functions
// can panic, including some standard library functions. Catching all error cases here would be very difficult.

// makeStructFromMap constructs a struct of type T from a map m of type map[string]any by
// setting all visible fields (possibly from embedded anonymous structs) in T according to m.
// The map must contain an entry for every such field of T and T must not not contain non-exported fields.
// m is allowed to contain additional entries that are not required/used for T.
//
// Returns an error if m does not contain data for some struct fields or data of invalid type;
// On error, the value for the returned struct ret is the zero value of the struct.
// We ask that data has exactly matching type except for interface types in the struct or nil values (of type any) in the map.
// m == nil is treated like an empty map. Using an invalid T causes panic.
func makeStructFromMap[StructType any](m map[string]any) (ret StructType, err error) {
	reflectedStructType := utils.TypeOfType[StructType]() // could do reflect.TypeOf(ret), but this gives better errors in case someone wrongly sets StructType to an interface type.
	allStructFields := getStructMapConversionLookup(reflectedStructType)
	retValue := reflect.ValueOf(&ret).Elem() // need to pass pointer for settability
	for _, structField := range allStructFields {
		fieldInRetValue := retValue.FieldByIndex(structField.Index)
		valueFromMap, ok := m[structField.Name]
		if !ok {
			err = fmt.Errorf(ErrorPrefix+"trying to construct value of type %v containing field named %v from parameters, but there is no entry for this",
				reflectedStructType, structField.Name)
			var zero StructType
			ret = zero
			return
		}
		// This is stupid, but Go1.18 requires special-casing here.
		// There is even a discussion to change the behaviour of reflect.ValueOf(nil)
		// cf. https://github.com/golang/go/issues/51649
		if valueFromMap == nil {
			if utils.IsNilable(fieldInRetValue.Type()) {
				appropriateNil := reflect.Zero(fieldInRetValue.Type())
				fieldInRetValue.Set(appropriateNil)
			} else {
				err = fmt.Errorf(ErrorPrefix+"trying to construct value of type %v from parameters; parameter named %v is set to nil, which is not valid for the struct field",
					reflectedStructType, structField.Name)
				var zero StructType
				ret = zero
				return
			}
		} else { // valueFromMap non-nil
			// reflect.Value.Set only requires assignability, not equality of types.
			// We want perfect roundtripping without conversion, so we need
			// type equality for concrete types, but assignability for struct fields of interface type.
			if fieldInRetValue.Kind() == reflect.Interface {
				if !reflect.TypeOf(valueFromMap).AssignableTo(fieldInRetValue.Type()) {
					err = fmt.Errorf(ErrorPrefix+"trying to construct value of type %v from parameters; parameter named %v is not assignable type: expected %v, got %v",
						reflectedStructType, structField.Name, fieldInRetValue.Type(), reflect.TypeOf(valueFromMap))
					var zero StructType
					ret = zero
					return
				}
			} else { // non-interface type for the field
				if fieldInRetValue.Type() != reflect.TypeOf(valueFromMap) {
					err = fmt.Errorf(ErrorPrefix+" trying to construct value of type %v from parameters; parameter named %v is of wrong type: expected %v, got %v",
						reflectedStructType, structField.Name, fieldInRetValue.Type(), reflect.TypeOf(valueFromMap))
					var zero StructType
					ret = zero
					return
				}
			}
			fieldInRetValue.Set(reflect.ValueOf(valueFromMap))
		}
	}
	return
}

// NOTE: Adding entries to an existing map is more convenient for our use cases than returning a map.

// fillMapFromStruct converts a struct of type StructType into a map[string]any
// by adding an entry to the provided (existing) map m for each visible field of StructType (including from embedded structs).
// This modifies m, treating a nil map as an empty map.
//
// StructType must contain only exported fields. If m points to something inside s (or similar shenanigans), the behaviour is undefined.
// Preexisting entries of m that do not correspond to a field of the struct are left unchanged.
func fillMapFromStruct[StructType any](s *StructType, m *map[string]any) {
	if *m == nil {
		*m = make(map[string]any)
	}
	reflectedStructType := utils.TypeOfType[StructType]()
	allStructFields := getStructMapConversionLookup(reflectedStructType)
	structValue := reflect.ValueOf(s).Elem()
	for _, structField := range allStructFields {
		fieldInStruct := structValue.FieldByIndex(structField.Index).Interface()
		(*m)[structField.Name] = fieldInStruct
	}
}
