package bandersnatchErrors

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

// lookupStructMapConversion is a lookup table (only depending of T) that contains the
// relevant data for converting an instance of a struct T to a map[string]any.
type lookupStructMapConversion = []reflect.StructField

var enforcedDataTypeMapMutex sync.RWMutex
var enforcedDataTypeMap map[reflect.Type]lookupStructMapConversion = make(map[reflect.Type]lookupStructMapConversion)

// getStructMapConversionLookup obtains a lookup table for converting a struct data type (passed as reflect.Type)
// to a map[string]any. This essentially is just reflect.VisibleFields with some extra checks upfront and skipping embedded fields.
func getStructMapConversionLookup(tType reflect.Type) (ret lookupStructMapConversion) {
	// Note: Our rw locking strategy can lead to multiple goroutines get different lookupStructMapConversion
	// (i.e. with different underlying arrays), but equivalent content.
	enforcedDataTypeMapMutex.RLock()
	ret, ok := enforcedDataTypeMap[tType]
	enforcedDataTypeMapMutex.RUnlock()
	if ok {
		return
	}
	if tType == nil {
		panic("bandersnatch / error handling: Called getStructMapConversionLookup with nil argment")
	}
	if tType.Kind() != reflect.Struct {
		panic("bandersnatch / error handling: Using getStructMapConversionLookup with non-struct type")
	}
	allVisibleFields := reflect.VisibleFields(tType)
	ret = make(lookupStructMapConversion, 0, len(allVisibleFields))
	for _, visibleField := range allVisibleFields {
		if !visibleField.IsExported() {
			panic("bandersnatch / error handling: Using errorWithEnsuredParameters with struct type containing unexported fields")
		}
		// .Anonymous denotes whether the field is embedded.
		// for an embedded field, reflect.VisibleFields returns both the name of the embedded type and its included field
		// We only want the latter, so we skip here.
		if visibleField.Anonymous {
			continue
		}
		ret = append(ret, visibleField)
	}
	enforcedDataTypeMapMutex.Lock()
	enforcedDataTypeMap[tType] = ret
	enforcedDataTypeMapMutex.Unlock()
	return
}

// validateErrorContainsData checks whether e actually contains data for all fields of a struct of type StructType.
// This is called after creating an error with T==StructType.
// e == nil is treated as error without any data.
func validateErrorContainsData[T any, StructType any](e *errorWithEnsuredParameters[T]) (err error) {
	structType := utils.TypeOfType[StructType]()
	allExpectedFields := getStructMapConversionLookup(structType)
	for _, expectedField := range allExpectedFields {
		mapEntry, exists := GetParameterFromError(e, expectedField.Name)
		if !exists {
			// Special case e==nil for better error message.
			// If e == nil, GetParameterFromError returns nil, false so any iteration of the for loop ends up here.
			if e == nil {
				err = fmt.Errorf("bandersnatch / error handling: nil error does not contain any parameters, but a parameter named %v was requested", expectedField.Name)
				return
			}
			err = fmt.Errorf("bandersnatch / error handling: error %v does not contain a parameters named %v, neccessary to export data a a struct of type %v", e, expectedField.Name, structType)
			return
		}
		// requires special casing due to what I consider a design error in reflection.
		// See https://github.com/golang/go/issues/51649 for an actual discussion to change it
		// for Go1.19 or later.
		if mapEntry == nil {
			if utils.IsNilable(expectedField.Type) {
				continue
			} else {
				err = fmt.Errorf("bandersnatch / error handling: error %v contains a parameter %v that is set to nil. This cannot be used to construct a struct of type %v",
					e, expectedField.Name, structType)
				return
			}
		}
		mapEntryType := reflect.TypeOf(mapEntry)
		// interface types as fields in StructType need special handling, because reflect.TypeOf(mapEntry) contains the dynamic type.
		if expectedField.Type.Kind() == reflect.Interface {
			if !mapEntryType.AssignableTo(expectedField.Type) {
				err = fmt.Errorf("bandersnatch / error handling: error %v has parameter %v set to a value %v; cannot export that in as struct of type %v, because that that value is not assignable to the intended field of interface type", e, expectedField.Name, mapEntry, structType)
				return
			}
		} else { // field of non-interface type in T
			if mapEntryType != expectedField.Type {
				err = fmt.Errorf("bandersnatch / error handling: error %v has parameter %v of wrong type to construct struct of type %v. Value is %v of type %v, expected %v",
					e, expectedField.Name, structType, mapEntry, mapEntryType, expectedField.Type)
				return
			}
		}
	}
	return nil
}

// Note: returning an error rather than panicking is somewhat difficult here, because several of the called functions
// can panic, including some standard library functions. Catching all error cases here is annoying.

// makeStructFromMap constructs a struct of type T from a map m of type map[string]any by
// setting all visible fields (possibly from embedded anonymous structs) in T according to m.
// The map must contain an entry for every such field of T and T must not not contain non-exported fields.
// m is allowed to contain entries that are not required for T.
// panics on failure.
func makeStructFromMap[StructType any](m map[string]any) (ret StructType, err error) {
	reflectedStructType := utils.TypeOfType[StructType]() // could do reflect.TypeOf(ret), but this gives better errors in case someone wrongly sets T to an interface type.
	allStructFields := getStructMapConversionLookup(reflectedStructType)
	retValue := reflect.ValueOf(&ret).Elem() // need to pass pointer for settability
	for _, structField := range allStructFields {
		fieldInRetValue := retValue.FieldByIndex(structField.Index)
		valueFromMap, ok := m[structField.Name]
		if !ok {
			err = fmt.Errorf("bandersnatch / error handling: trying to construct value of type %v containing field named %v from parameters, but there is no entry for this",
				reflectedStructType, structField.Name)
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
				err = fmt.Errorf("bandersnatch / error handling: trying to construct value of type %v from parameters; parameter named %v is set to nil, which is not valid for the struct field",
					reflectedStructType, structField.Name)
				return
			}
		} else { // valueFromMap non-nil
			// reflect.Value.Set only requires assignability, not equality of types.
			// We want perfect roundtripping without conversion, so we need
			// type equality for concrete types, but assignability for struct fields of interface type.
			if fieldInRetValue.Kind() == reflect.Interface {
				if !reflect.TypeOf(valueFromMap).AssignableTo(fieldInRetValue.Type()) {
					err = fmt.Errorf("bandersnatch / error handling: trying to construct value of type %v from parameters; parameter named %v is not assignable type: expected %v, got %v",
						reflectedStructType, structField.Name, fieldInRetValue.Type(), reflect.TypeOf(valueFromMap))
					return
				}
			} else { // non-interface type for the field
				if fieldInRetValue.Type() != reflect.TypeOf(valueFromMap) {
					err = fmt.Errorf("bandersnatch / error handling: trying to construct value of type %v from parameters; parameter named %v is of wrong type: expected %v, got %v",
						reflectedStructType, structField.Name, fieldInRetValue.Type(), reflect.TypeOf(valueFromMap))
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
// This modifies m, treating a nil map as an empty map. StructType must contain only exported fields.
// preexisting entries of m that do not correspond to a field of the struct are left unchanged.
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
