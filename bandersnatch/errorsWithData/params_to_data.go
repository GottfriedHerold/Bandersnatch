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

// StructSuitableForErrorsWithData is used to check whether a given StructType can be used as a generic parameter for various functions/methods/types in this package.
// If StructType is suitable, this function returns nil; if unsuitable, returns an error describing a reason why StructType is unsuitable.
//
// Using an StructType that does not pass this generic function with any function/method/type of this package other than StructSuitableForErrorsWithData may generate a panic.
//
// The precise restrictions we place on StructType and check with this function are as follows:
//
//   - StructType must be a struct
//   - All non-embedded field names must be exported.
//   - Embedded pointer-to-non-structs must be exported
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

// getStructMapConversionLookup obtains a lookup data structure for converting a struct data type (passed as reflect.Type)
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
	// Also, it means we don't really have to care about efficiency.

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
	// our rules would give no clear winner and reject T as invalid.
	// Note that our rules differ from Go here, which only looks at depth, making T.S1.X the winner -- Go's way of doing it makes embedding depth an observable property, i.e. the fact that/how a field was defined via embedding leaks the abstraction.
	// T would get the X field via incomparable paths. We would reject T as invalid.
	// If we sort allFields, we would process an possible T.X first. In particular, we encountere colliding T.S1.X and T.S2.S3.X, we know there will be no
	// saving T.X encountered later during the iteration. This allows to detect that T is invalid right away, simplifying things.
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

// ensureCanMakeStructFromParameters checks whether *m actually contains suitable data for all fields of a struct of type StructType.
// Possibly, the function modifies *m to ensure this is the case.
// It returns nil on success, otherwise an error describing the reason of failure.
//
// Note that this function is used both to verify and modify *m. Using a single function for this simplifies the code.
//
// On type mismatch between type expected for StructType and what's in *m, we always return an error.
// If c_SetZeros.ModifyData() is true, then we modify those entries with type-mismatch or missing data.
// If data is merely missing, the behaviour depends on c_ImplicitZero:
//   - Iff c_ImplicitZero.IsMissingDataError() == true, this function reports an error if data is missing.
//   - Iff c_SetZeros.ModifyData() == true, this function fills the map with zero of appropriate type.

// (note these the latter two questions are actually orthogonal).
// The function does not abort on first error.
// If there is an error for multiple keys/required fields of StructType, the returned err message
// contains information about all of them.
//
// The point of having ModifyData() set to true is that this actually guarantees that
// the resulting *m can be used to construct an instance of StructType. This is used to by calls from ErrorWithData[T]-creating functions to
// make sure that even if an error occurs, the created instance of ErrorWithData[T] satisfies appropriate invariants.
//
// *m == nil must not be used. The function will panic in this case (use an empty map instead).
//
// This function panics if called with an invalid (see [StructSuitableForErrorsWithData]) StructType
func ensureCanMakeStructFromParameters[StructType any](m *ParamMap, c_ImplicitZero config_ImplicitZero, c_SetZeros config_SetZeros) (err error) {
	if *m == nil {
		panic(ErrorPrefix + "called ensureMakeStructFromParameters with pointer to nil map")
	}

	// we do not abort on first error, but actually collect and report all of them in the returned err.
	// TODO: Use Go >=1.21 (IIRC) multiple-error wrapping to report all errors.
	var errors []error = make([]error, 0)

	structType := utils.TypeOfType[StructType]()
	allExpectedFields, errBadStructType := getStructMapConversionLookup(structType)
	if errBadStructType != nil {
		panic(errBadStructType) // -> return to change to non-panicking behaviour if desired
	}
	for _, expectedField := range allExpectedFields {

		mapEntry, exists := (*m)[expectedField.Name]
		if !exists {
			if c_SetZeros.ModifyData() {
				(*m)[expectedField.Name] = reflect.Zero(expectedField.Type).Interface()
			}
			// no else!
			if c_ImplicitZero.IsMissingDataError() {
				errors = append(errors, fmt.Errorf("lacking a parameter named %v", expectedField.Name))
			}
			continue // no need to check the type
		}
		// requires special casing due to what I consider a design error in reflection.
		// See https://github.com/golang/go/issues/51649 for an actual discussion to change it for Go1.19 or later.
		if mapEntry == nil {
			if utils.IsNilable(expectedField.Type) {
				continue // no further check neccessary.
			} else {
				errors = append(errors, fmt.Errorf("containing a parameter %v that is set to the nil interface. This cannot be used for the corresponding field of non-nilable type %v",
					expectedField.Name, utils.GetReflectName(expectedField.Type)))
				continue
			}
		}
		mapEntryType := reflect.TypeOf(mapEntry) // Note: mapEntry cannot be nil here, because that was handled already
		// interface types as fields in StructType need special handling, because reflect.TypeOf(mapEntry) contains the dynamic type.
		if expectedField.Type.Kind() == reflect.Interface {
			if !mapEntryType.AssignableTo(expectedField.Type) {
				errors = append(errors, fmt.Errorf("parameter %v is set to the value %v; this value is not assignable to the intended field (which is of interface type) of the struct",
					expectedField.Name, mapEntry))
				if c_SetZeros.ModifyData() {
					(*m)[expectedField.Name] = reflect.Zero(expectedField.Type).Interface()
				}
				// typeError = true
				continue
			}
		} else { // field of non-interface type in T: We require the types to match exactly.
			if mapEntryType != expectedField.Type {
				errors = append(errors, fmt.Errorf("parameter %v is of wrong type.\nValue is %v of type %v, but the expected type is %v",
					expectedField.Name, mapEntry, utils.GetReflectName(mapEntryType), utils.GetReflectName(expectedField.Type)))
				if c_SetZeros.ModifyData() {
					(*m)[expectedField.Name] = reflect.Zero(expectedField.Type).Interface()
				}
				// typeError = true
				continue
			}
		}
	}
	if len(errors) != 0 {
		typeName := utils.GetReflectName(structType)
		if len(errors) == 1 {
			err = fmt.Errorf(ErrorPrefix+"not possible to construct a struct of type %v from the given parameters for the following reason: %w", typeName, errors[0])
		} else {
			err = fmt.Errorf(ErrorPrefix+"not possible to construct a struct of type %v from the given parameters for the following %v reasons: %v", typeName, len(errors), errors)
		}
	}
	return
}

// Note: Always returning an error rather than panicking is somewhat finicky, because several of the called functions
// can panic, including some standard library functions. Catching all error cases here is very difficult.
// Check this very carefully.

// makeStructFromMap constructs a struct of type StructType from a map m of type map[string]any by
// setting all visible fields (possibly from embedded anonymous structs) in StructType according to m.
//
// m is allowed to contain additional entries that are not required/used for StructType.
//
// If there is no entry in m for some field of StructType, the behaviour depends on structFieldConfig.
// if c_ImplicitZero.IsMissingDataError() == true, this causes an error to be returned.
// Independently from that, the corresponding field in ret is zero-initialized.
//
// If the data type in m is inappropriate for some field of ret, this is also an error and the corresponding field is zero-initialized.
//
// We do *NOT* abort on first error. The returned err contains diagnostic information about each field of StructType where an error occurred.
// In case of error, the retuned ret will have all non-failing entries set according to m.
//
// We ask that data (if present) has exactly matching type except for interface types in the struct or nil interface values in the map.
// m == nil is treated like an empty map.
// An invalid StructType in the sense of [StructSuitableForErrorsWithData] causes a panic.
func makeStructFromMap[StructType any](m map[string]any, c_ImplicitZero config_ImplicitZero) (ret StructType, err error) {

	// ret starts off zero-initialized and gets modified (via reflection) within this function.

	// reflect.TypeOf(ret) would not fail via the intended way in case someone wrongly sets StructType to an interface type.
	// getStructMapConversionLookup panics either way, but the error message would be confusing.
	reflectedStructType := utils.TypeOfType[StructType]()
	allStructFields, errInvalidStructType := getStructMapConversionLookup(reflectedStructType)
	if errInvalidStructType != nil {
		panic(errInvalidStructType)
	}

	// we do not abort on first error, but actually collect and report all of them in the returned err.
	// TODO: Use Go >=1.21 (IIRC) multiple-error wrapping to report all errors.
	var errors []error = make([]error, 0)

	retValue := reflect.ValueOf(&ret).Elem() // need to pass pointer (and deref via .Elem() ) for settability
	for _, structField := range allStructFields {
		fieldInRetValue := retValue.FieldByIndex(structField.Index)
		valueFromMap, ok := m[structField.Name]
		if !ok {
			// missing value:
			// We need to zero-initialize the appropriate field. This was already done automatically by the compiler when we declared ret, so we don't need to do anything.

			if c_ImplicitZero.IsMissingDataError() {
				errors = append(errors, fmt.Errorf(ErrorPrefix+"for the field named %v, there is no entry in the parameter map",
					structField.Name))
			}
			continue // done with this value (the code below would not work, in fact)
		}

		// This is annoying, but Go1.18 requires special-casing here.
		// There is even a discussion to change the behaviour of reflect.ValueOf(nil)
		// cf. https://github.com/golang/go/issues/51649
		if valueFromMap == nil { // nil interface in map
			if utils.IsNilable(fieldInRetValue.Type()) {
				appropriateNil := reflect.Zero(fieldInRetValue.Type())
				fieldInRetValue.Set(appropriateNil)
			} else {
				errors = append(errors, fmt.Errorf("parameter named %v is set to any(nil), but the struct field cannot be nil",
					structField.Name))

			}
			continue
		} else { // valueFromMap non-nil (i.e. value in map is not nil interface)
			// reflect.Value.Set only requires assignability, not equality of types.
			// We want perfect roundtripping without conversion here, so we need
			// type equality for concrete types, but assignability for struct fields of interface type.
			if fieldInRetValue.Kind() == reflect.Interface {
				if !reflect.TypeOf(valueFromMap).AssignableTo(fieldInRetValue.Type()) {
					errors = append(errors, fmt.Errorf("parameter named %v with value %v of (dynamic) type %T does not satisfy the required interface type %v",
						structField.Name, valueFromMap, valueFromMap, fieldInRetValue.Type()))
					continue // actual value in struct stays zero-initialized
				}
			} else { // non-interface type for the struct field
				if fieldInRetValue.Type() != reflect.TypeOf(valueFromMap) {
					errors = append(errors, fmt.Errorf("parameter named %v is of wrong type: expected type %v, got value %v of type %T",
						structField.Name, fieldInRetValue.Type(), valueFromMap, valueFromMap))
					continue // actual value in struct stays zero-initialized
				}
			}
			// no problem detected. Actually set the value
			fieldInRetValue.Set(reflect.ValueOf(valueFromMap))
		}
	}
	if len(errors) != 0 {
		typeName := utils.GetReflectName(utils.TypeOfType[StructType]())
		if len(errors) == 1 {
			err = fmt.Errorf(ErrorPrefix+"error constructing a struct of type %v from the given parameter map for the following reason: %w", typeName, errors[0])
		} else {
			err = fmt.Errorf(ErrorPrefix+"error constructing a struct of type %v from the given parameter map for the following list of %v reasons: %v", typeName, len(errors), errors)
		}
	}
	return
}
