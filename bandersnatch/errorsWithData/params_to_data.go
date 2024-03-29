package errorsWithData

import (
	"fmt"
	"reflect"
	"sort"
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

// structTypeToFieldsLookupEntry is an entry in a lookup table (depending on the type T) that contains the
// relevant data for converting an instance of a struct of type T to a map[string]any.
type structTypeToFieldsLookupEntry = []reflect.StructField

// structTypeToFieldsMap is a global mutex-protected lookup table StructType T -> relevant data for converting instances of T to map[string]any
var (
	structTypeToFieldsMap      map[reflect.Type]structTypeToFieldsLookupEntry = make(map[reflect.Type]structTypeToFieldsLookupEntry)
	structTypeToFieldsMapMutex sync.RWMutex
)

// MissingDataTreatment is a type used to pass to exported functions how the library should treat missing parameters
// when using the struct-based API.
//
// We provide [MissingDataAsZero] and [EnsureDataIsPresent] as possible values. The zero value of this type is invalid.
//
// Selecting [EnsureDataIsPresent] causes the package to report an error or panic if parameters are missing (This is explicitly specified at a per-function basis).
// [MissingDataAsZero] causes the package to zero-initialize values if parameters are missing.
type MissingDataTreatment struct {
	handleMissingData int
}

// potential values of MissingDataTreatment.handleMissingData

const (
	handleMissingData_Unset         = iota // The zero value of [MissingDataTreatment] corresponds to an unset value. Using this causes a panic rather than some default behaviour.
	handleMissingData_AddZeros             // zero-initialize missing values
	handleMissingData_AssertPresent        // panic if values are missing

	// handleMissingData_TreatAsZero
	// handleMissingData_Ignore

)

var (
	// MissingDataAsZero is passed to functions to indicate that missing data should be zero initialized
	MissingDataAsZero = MissingDataTreatment{handleMissingData: handleMissingData_AddZeros}
	// EnsureDataIsPresent is passed to functions to indicate that the function should report an error (possibly panic) if data is missing
	EnsureDataIsPresent = MissingDataTreatment{handleMissingData: handleMissingData_AssertPresent}
)

// String is provided to make MissingDataTreatment satisfy fmt.Stringer. This is used for debugging only.
func (m MissingDataTreatment) String() string { // Note: value receiver
	switch m.handleMissingData {
	case handleMissingData_Unset:
		return "Unset value for missing data treatment"
	case handleMissingData_AddZeros:
		return "Fill missing data with zeros"
	case handleMissingData_AssertPresent:
		return "Error if data is missing"
	default:
		// cannot happen without using unsafe methods.
		return fmt.Sprintf("Unexpected internal value %v for MissingDataTreatment", m.handleMissingData)
	}
}

// StructSuitableForErrorsWithData is used to check whether a given StructType can be used as a generic parameter for various functions/methods/types in this package.
// If StructType is suitable, this function returns nil; if unsuitable, returns an error describing a reason why StructType is unsuitable.
//
// Using an StructType that does not pass this generic function with any function/method/type of this package other than StructSuitableForErrorsWithData may generate a panic.
//
// The precise restrictions we place on StructType and check with this function are as follows:
//
//   - StructType must be a struct
//   - All non-embedded field names must be exported.
//   - Embedded types must not be pointer-to-struct. Embedded structs or embedded pointer-to-non-structs are allowed.
//   - Embedded structs lead to a promoted field hierarchy, which has a tree structure.
//     We are more strict than the usual Go rules and allow shadowing of fields
//     only iff every shadowed field is actually in a subtree of the struct that defines the shadowing field.
//
// An example of the last item above is the following: Consider types
//
//	type T struct{X int}
//	type WrappedT struct{T}
//	type S struct{T;WrappedT}
//
// we disallow S because S has a promoted field X via both S.T.X and S.WrappedT.T.X.
// While Go itself would allow S.X as a promoted form of S.T.X (S.T.X wins over S.WrappedT.T.X due to lower depth),
// we reject this construction, because the candidates get promoted via different pathways (WrappedT vs. T):
// S.T.X cannot shadow S.WrappedT.T.X because the latter is defined in S.WrappedT.T, which is not in a subtree of S.T.
// If, in this example, S itself additionally defined its own field X, then S would satisfy our restrictions.
// We do not expect such corner-cases to come up, really. Frankly speaking, the fact that the Go language allows S is questionable to start with.
func StructSuitableForErrorsWithData[StructType any]() (err error) {
	_, err = getStructMapConversionLookup(utils.TypeOfType[StructType]())
	return
}

// Note: getStructMapConversionLookup used to call reflect.VisibleFields rather than utils.AllFields.
// Unfortunately, reflect.VisibleFields does not handle embedded fields of non-struct type the way we like.
// (e.g. one issue is embedded struct pointers)

// getStructMapConversionLookup obtains a data structure for converting a struct data type (passed as reflect.Type)
// to a map[string]any. Repeated calls with the same argument give identical results (slice with same backing array)
// since we use a global lookup-table to cache results.
//
// This mostly is just utils.AllFields with some extra checks upfront, skipping embedded fields and handling shadowed fields.
//
// This functions returns an error if called with a structType that is deemed invalid for our purpose.
func getStructMapConversionLookup(structType reflect.Type) (ret structTypeToFieldsLookupEntry, err error) {
	// Note: We cache the result in a global map. This implies
	// that e.g. the order of entries in the returned struct is consistent.
	// (Not sure whether this is needed and this is actually true anyway due to the way utils.AllFields works)
	// Also, it means we don't have to care about efficiency.

	// Check if we alread have the table in the cache.
	structTypeToFieldsMapMutex.RLock()
	ret, ok := structTypeToFieldsMap[structType]
	structTypeToFieldsMapMutex.RUnlock()
	if ok {
		return
	}
	// We get here if we did not find the answer in the cache.

	// Make some sanity checks: This function only makes sense for struct types.
	if structType == nil {
		// The exported API always passes structType as a generic parameter.
		// We get a reflect.Type via utils.TypeOfType[StructType](), which never returns nil.
		// As a consequence, this is not triggerable from the exported API unless we screwed up internally.
		// This is why we panic on error.
		err = fmt.Errorf(ErrorPrefix + "internal error: called getStructMapConversionLookup with nil argment")
		panic(err)
	}
	if structType.Kind() != reflect.Struct {
		err = fmt.Errorf(ErrorPrefix+"%v is not a struct type", utils.GetReflectName(structType))
		return
	}

	// The intended result will be a subset of all fields
	allFields, embeddedStructPointer := utils.AllFields(structType)
	if embeddedStructPointer {
		err = fmt.Errorf(ErrorPrefix+"the struct type %v contains an embedded struct pointer. This is not supported", utils.GetReflectName(structType))
		return
	}

	// We sort allFields such that we process shorter index sequences first.
	// This is needed to correctly handle certain cases of shadowing promoted fields:
	// We only accept shadowing promoted fields iff the "winner" has the property that each
	// field that is shadowed by it is actually deeper in the promoted field hiearchy tree.
	// and on a path from the winner.
	//
	// Now, there can be sitations where for a struct T and field name X, T.X shadows both T.S1.X and T.S2.S3.X
	// In this case, T.X should be the winner.
	// However, if T.X was not present and we would only have T.S1.X and T.S2.S3.X,
	// our rules (different from Go, which only looks at depth) would give no clear winner.
	// T would get the X field via incomparable paths. We would reject T as invalid.
	// If we sort allFields, we know that whenever we encounter such a situation, there will be no
	// saving T.X encountered later during the iteration and we detect that T is invalid right away, simplifying things.
	sort.Slice(allFields, func(i int, j int) bool {
		return len(allFields[i].Index) < len(allFields[j].Index)
	})

	ret = make(structTypeToFieldsLookupEntry, 0, len(allFields))

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
			err = fmt.Errorf(ErrorPrefix+"struct type %v contains unexported fields. This is not supported", utils.GetReflectName(structType))
			return
		}

		// check for shadowing of existing fields. This has quadratic running time, but we don't care.
		for _, existingField := range ret {
			// Shadowing:
			if existingField.Name == newField.Name {
				// shorter length of Index is the one that get precedence according to Go's shadowing rules.
				// We are stricter

				// Note that existingField is iterated in order of length and only covers those already put into ret.

				// ensure that for existingField.Index and field.Index, one is a prefix of the other (except for the last entry).
				// Note that this is stronger that the usual rules, which just compare len of Index.
				if len(existingField.Index) < len(newField.Index) {
					for i := 0; i < len(existingField.Index)-1; i++ {
						if existingField.Index[i] != newField.Index[i] {
							err = fmt.Errorf(ErrorPrefix+"struct type %v has promoted fields names %v through different embedded fields. This is not supported", utils.GetReflectName(structType), existingField.Name)
							return
						}
					}
					continue outer_loop // don't add visible field, the existing one takes precedence
				}

				// In case of ambiguity, we trigger an error.
				// Note that due to sorting, existingField is known to be a field with shortest index length among the equally named,
				// so there cannot be other fields that would resolve the ambiguity.
				if len(existingField.Index) == len(newField.Index) {
					err = fmt.Errorf(ErrorPrefix+"struct type %v has an ambiguous promoted field named %v", utils.GetReflectName(structType), existingField.Name)
					return
				}

				// We would get here if len(existingField.Index) > len(field.Index)
				// This is unreachable due to sorting.
				panic(ErrorPrefix + "internal error: allFieldNames supposed to have been sorted")
			}
		}

		ret = append(ret, newField)
	}

	// Write ret into the cache. We RW-lock the cache for this.
	structTypeToFieldsMapMutex.Lock()
	defer structTypeToFieldsMapMutex.Unlock()
	// We need to check if some other goroutine already filled the cache in the meantime.
	_, ok = structTypeToFieldsMap[structType]
	if ok {
		ret = structTypeToFieldsMap[structType] // use the value already in the map, to give consistently identical (not just equal) results.
	} else {
		structTypeToFieldsMap[structType] = ret
	}
	return
}

// ensureCanMakeStructFromParameters checks whether m actually contains suitable data for all fields of a struct of type StructType.
// It returns nil on success, otherwise an error describing the reason of failure.
//
// On type mismatch, we always return an error.
// If data is merely missing, the behaviour depends on missingDataTreatment:
//   - For missingDataTreatment == [EnsureDataIsPresent], this function returns an error
//   - For missingDataTreatment == [MissingDataAsZero], this function adds entries to *m (thereby modifying *m)
//
// This is called after creating an error with T==StructType.
// *m == nil never not used. We prefer an empty map.
//
// This function panics if called with an invalid StructType (may be changed)
func ensureCanMakeStructFromParameters[StructType any](m *ParamMap, missingDataTreatment MissingDataTreatment) (err error) {
	structType := utils.TypeOfType[StructType]()
	typeName := utils.GetReflectName(structType)
	allExpectedFields, err := getStructMapConversionLookup(structType)
	if err != nil {
		panic(err) // -> return to change to non-panicking behaviour if desired
	}
	for _, expectedField := range allExpectedFields {

		mapEntry, exists := (*m)[expectedField.Name]
		if !exists {
			switch missingDataTreatment.handleMissingData {
			case handleMissingData_AssertPresent:
				err = fmt.Errorf(ErrorPrefix+"lacking a parameter named %v, neccessary to export data a a struct of type %v", expectedField.Name, typeName)
				return
			case handleMissingData_AddZeros:
				(*m)[expectedField.Name] = reflect.Zero(expectedField.Type).Interface()
				continue
			default:
				panic(ErrorPrefix + "invalid value for missingDataTreatment")
			}

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
		mapEntryType := reflect.TypeOf(mapEntry) // Note: mapEntry cannot be nil here, because that was handled already
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

// makeStructFromMap constructs a struct of type StructType from a map m of type map[string]any by
// setting all visible fields (possibly from embedded anonymous structs) in StructType according to m.
//
// m is allowed to contain additional entries that are not required/used for StructType.
//
// If there is no entry in m for some field of StructType, the behaviour depends on missingDataTreatment.
// if missingDataTreatment is [EnsureDataIsPresent], we return an error
// if missingDataTreatment is [AddZeroForMissingData], the field in StructType is zero-initialized.
//
// On error, the value for the returned struct ret is the zero value of the struct.
// We ask that data (if present) has exactly matching type except for interface types in the struct or nil interface values in the map.
// m == nil is treated like an empty map. An invalid StructType causes an error to be returned (not a panic).
func makeStructFromMap[StructType any](m map[string]any, missingDataTreatment MissingDataTreatment) (ret StructType, err error) {
	// ret starts off zero-initialized and gets modified (via reflection) within this function.
	reflectedStructType := utils.TypeOfType[StructType]() // reflect.TypeOf(ret) would not work in case someone wrongly sets StructType to an interface type.
	allStructFields, err := getStructMapConversionLookup(reflectedStructType)
	if err != nil {
		return
	}
	retValue := reflect.ValueOf(&ret).Elem() // need to pass pointer for settability
	for _, structField := range allStructFields {
		fieldInRetValue := retValue.FieldByIndex(structField.Index)
		valueFromMap, ok := m[structField.Name]
		if !ok {
			switch missingDataTreatment.handleMissingData {
			case handleMissingData_AssertPresent:
				err = fmt.Errorf(ErrorPrefix+"trying to construct value of type %v containing field named %v from parameters, but there is no entry for this",
					reflectedStructType, structField.Name)
				var zero StructType
				ret = zero
				return
			case MissingDataAsZero.handleMissingData:
				// leave the field zero-initialized
				continue
			default:
				panic(fmt.Errorf(ErrorPrefix+"Invalid value for missingDataTreament: %v", missingDataTreatment))
			}
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
