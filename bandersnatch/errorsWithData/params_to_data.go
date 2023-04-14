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
// Some care needs to be taken with non-exported, embedded and shadowed fields and nil interfaces:
//
// embedded fields are ignored (or rather, "flattened" -- On the map level, all promoted fields are at the same level)
// non-exported fields are disallowed (except for the name of a type used in field embedding)
// shadowed fields are allowed (the map has only one field per name)
// nil interface entries in the maps get converted to a typed nil of the appropriate type in the struct (the conversion vice-versa is to a typed nil)

// lookupStructMapConversion is a lookup table (only depending of T) that contains the
// relevant data for converting an instance of a struct T to a map[string]any.
type lookupStructMapConversion = []reflect.StructField

var enforcedDataTypeMapMutex sync.RWMutex
var enforcedDataTypeMap map[reflect.Type]lookupStructMapConversion = make(map[reflect.Type]lookupStructMapConversion)

// Note: This used to call reflect.VisibleFields  rather than utils.AllFields.
// Unfortunately, reflect.VisibleFields does not handle embedded fields of non-struct type the way we like.
// (e.g. one issue is embedded struct pointers)

// getStructMapConversionLookup obtains a lookup table for converting a struct data type (passed as reflect.Type)
// to a map[string]any. Repeated calls with the same argument give identical results (slice with same backing array)
//
// This essentially is just utils.AllFields with some extra checks upfront, skipping embedded fields and handling shadowed fields.
//
// This functions panics if called with a structType that is deemed invalid for our purpose.
func getStructMapConversionLookup(structType reflect.Type) (ret lookupStructMapConversion) {
	// Note: We cache the result in a global map. This implies
	// that e.g. the order of entries in the returned struct is consistent.
	// (Not sure whether this is needed and this is actually true anyway due to the way AllFields works)
	// Also, it means we don't have to care about efficiency.

	// Check if we alread have the table in the cache.
	enforcedDataTypeMapMutex.RLock()
	ret, ok := enforcedDataTypeMap[structType]
	enforcedDataTypeMapMutex.RUnlock()
	if ok {
		return
	}
	// We get here if we did not find the answer in the cache.

	// Make some sanity checks: This function only makes sense for struct types.
	if structType == nil {
		panic(ErrorPrefix + "called getStructMapConversionLookup with nil argment")
	}
	if structType.Kind() != reflect.Struct {
		panic(ErrorPrefix + "using getStructMapConversionLookup with non-struct type")
	}

	// The intended result will be a subset of all fields
	allFields, embeddedStructPointer := utils.AllFields(structType)
	if embeddedStructPointer {
		panic(ErrorPrefix + "using getStructMapConversionLookup with a struct that contains an embedded struct pointer")
	}
	ret = make(lookupStructMapConversion, 0, len(allFields))

	// ensure everything is exported and filter out embedded fields
outer_loop:
	for _, newField := range allFields {
		// .Anonymous denotes whether the field is embedded (a bit of a misnomer).
		// for an embedded struct, utils.AllFields returns both the name of the embedded type and its included field
		// We only want the latter, so we skip here.
		if newField.Anonymous && newField.Type.Kind() == reflect.Struct {
			continue
		}
		if !newField.IsExported() {
			panic(ErrorPrefix + "using errorsWithData with struct type containing unexported fields")
		}

		for pos, existingField := range ret {
			// Shadowing:
			if existingField.Name == newField.Name {
				// shorter length of Index is the one that get precedence according to Go's shadowing rules.
				// In case of ambiguity, we panic:
				if len(existingField.Index) == len(newField.Index) {
					panic(ErrorPrefix + "using errorsWithData with struct type that has an ambiguous promoted field")
				}
				// ensure that for existingField.Index and field.Index, one is a prefix of the other (except for the last entry).
				// Note that this is stronger that the usual rules, which just compare len of Index.
				if len(existingField.Index) < len(newField.Index) {
					for i := 0; i < len(existingField.Index)-1; i++ {
						if existingField.Index[i] != newField.Index[i] {
							panic(ErrorPrefix + "using errorsWithData with struct type that has a promoted field through different embedded fields")
						}
					}
					continue outer_loop // don't add visible field, the existing one takes precedence
				} else { // len(existingField.Index) > len(field.Index)
					for i := 0; i < len(newField.Index)-1; i++ {
						if existingField.Index[i] != newField.Index[i] {
							panic(ErrorPrefix + "using errorsWithData with struct type that has a promoted field through different embedded fields")
						}
					}
					ret[pos] = newField // overwrite previous entry
					continue outer_loop // to skip the append below
				}
			}
		}

		ret = append(ret, newField)
	}

	// Write ret into the cache. We RW-lock the cache for this.
	enforcedDataTypeMapMutex.Lock()
	defer enforcedDataTypeMapMutex.Unlock()
	// We need to check if some other goroutine already filled the cache in the meantime.
	_, ok = enforcedDataTypeMap[structType]
	if ok {
		ret = enforcedDataTypeMap[structType]
	} else {
		enforcedDataTypeMap[structType] = ret
	}
	return
}

// CheckParametersForStruct_all[StructType](fieldNames) checks whether the name of the fields coincides with
// the slice of fieldNames. Note that we require equality, i.e. the list of fieldNames is exhaustive;
//
// All fields of StructType must be exported and shadowed names must only appear once.
// This is intented to be used in init-routines or tests accompanying places in the code
// where we assume that a certain struct has exactly a given set of field names.
// The purpose is to create guards in the code. It panics on failure.
func CheckParametersForStruct_all[StructType any](fieldNames []string) {

	// quadratic, but I don't care.
	for i := 0; i < len(fieldNames); i++ {
		for j := i + 1; j < len(fieldNames); j++ {
			if fieldNames[i] == fieldNames[j] {
				panic(fmt.Errorf(ErrorPrefix+"In call to CheckParametersForStruct, the given list of field names contains a duplicate: %v", fieldNames[i]))
			}
		}
	}
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
	// No need to check that fieldName is a valid exported name. The function will fail anyway if this is not satisfied.
	allExpectedFields := getStructMapConversionLookup(utils.TypeOfType[StructType]())
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
// The purpose is to create guards in the code. It panics on failure.
func CheckIsSubtype[StructType1 any, StructType2 any]() {
	allExpectedFields1 := getStructMapConversionLookup(utils.TypeOfType[StructType1]())
	for _, expectedField1 := range allExpectedFields1 {
		CheckParameterForStruct[StructType2](expectedField1.Name)
	}
}

// canMakeStructFromParameters checks whether m actually contains data for all fields of a struct of type StructType.
// It returns nil on success, otherwise an error describing the reason of failure.
//
// This is called after creating an error with T==StructType.
// m == nil is not used. We prefer an empty map.
func canMakeStructFromParameters[StructType any](m ParamMap) (err error) {
	structType := utils.TypeOfType[StructType]()
	typeName := utils.GetReflectName(structType)
	allExpectedFields := getStructMapConversionLookup(structType)
	for _, expectedField := range allExpectedFields {

		mapEntry, exists := m[expectedField.Name]
		if !exists {
			err = fmt.Errorf(ErrorPrefix+"lacking a parameter named %v, neccessary to export data a a struct of type %v", expectedField.Name, typeName)
			return
		}
		// requires special casing due to what I consider a design error in reflection.
		// See https://github.com/golang/go/issues/51649 for an actual discussion to change it for Go1.19 or later.
		if mapEntry == nil {
			if utils.IsNilable(expectedField.Type) {
				continue // no further check neccessary.
			} else {
				err = fmt.Errorf(ErrorPrefix+"containing a parameter %v that is set to nil (untyped/interface). This cannot be used for the corresponding field of non-nilable type %v in the struct %v",
					expectedField.Name, utils.GetReflectName(expectedField.Type), typeName)
				return
			}
		}
		mapEntryType := reflect.TypeOf(mapEntry)
		// interface types as fields in StructType need special handling, because reflect.TypeOf(mapEntry) contains the dynamic type.
		if expectedField.Type.Kind() == reflect.Interface {
			if !mapEntryType.AssignableTo(expectedField.Type) {
				err = fmt.Errorf(ErrorPrefix+"parameter %v is set to the value %v; this cannot be used to construct a struct of type %v, because the value is not assignable to the intended field (which is of interface type) of the struct",
					expectedField.Name, mapEntry, typeName)
				return
			}
		} else { // field of non-interface type in T: We require the types to match exactly.
			if mapEntryType != expectedField.Type {
				err = fmt.Errorf(ErrorPrefix+" parameter %v is of wrong type to construct struct of type %v.\nValue is %v of type %v, but the expected type is %v",
					expectedField.Name, typeName, mapEntry, utils.GetReflectName(mapEntryType), utils.GetReflectName(expectedField.Type))
				return
			}
		}
	}
	return nil
}

// Note: Always returning an error rather than panicking is somewhat finicky, because several of the called functions
// can panic, including some standard library functions. Catching all error cases here is very difficult.
// So be aware that this may possibly panic (even though returning an error seems to indicate otherwise)

// makeStructFromMap constructs a struct of type T from a map m of type map[string]any by
// setting all visible fields (possibly from embedded anonymous structs) in T according to m.
// The map must contain an entry for every such field of T and T must not contain non-exported fields.
// m is allowed to contain additional entries that are not required/used for T.
//
// Returns an error if m does not contain data for some struct fields or data of invalid type;
// On error, the value for the returned struct ret is the zero value of the struct.
// We ask that data has exactly matching type except for interface types in the struct or nil interface values in the map.
// m == nil is treated like an empty map. Using an invalid T causes panic (This might change in the future and give an error instead).
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
		// This is annoying, but Go1.18 requires special-casing here.
		// There is even a discussion to change the behaviour of reflect.ValueOf(nil)
		// cf. https://github.com/golang/go/issues/51649
		if valueFromMap == nil { // nil interface in map
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
		} else { // valueFromMap non-nil (i.e. value in map is not nil interface)
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
