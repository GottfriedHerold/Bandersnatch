package errorsWithData

import (
	"fmt"
	"reflect"

	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

// Functions and methods that modify errors with data take as input a parameter that controls how already-present data should be handled.
// The options are to prefer the old, prefer the new or panic on ambiguity.
// For type-safety, this choice is passed as a parameter of designated type PreviousDataTreatment.
// This file defined this type and its associated methods.

// PreviousDataTreatment is an encapsulated enum type passed to functions and methods that modify the data associated to errors.
//
// It controls how the library should treat setting values that are already present.
// We provide [PreferPreviousData], [ReplacePreviousData], [AssertDataIsNotReplaced] as potential values.
// The zero value of this type is not a valid PreviousDataTreatment. Using such a zero value will cause panics.
type PreviousDataTreatment struct {
	keep int
}

// internal int-based enum for [PreviousDataTreatment]. We use a struct wrapping an int in our exported API.
// This is because we want stronger typing for methods that already take "any" or generic-parameter dependent values.
const (
	treatPreviousData_Unset = iota // zero value. This is not a valid value.
	treatPreviousData_Override
	treatPreviousData_PreferOld
	treatPreviousData_PanicOnCollision
)

var (
	// PreferPreviousData means that when replacing associated data in errors, we keep the old value if some value is already present for a given key.
	PreferPreviousData = PreviousDataTreatment{keep: treatPreviousData_PreferOld}
	// ReplacePreviousData means that when replacing associated data in errors, we unconditionally override already-present values for a given key.
	ReplacePreviousData = PreviousDataTreatment{keep: treatPreviousData_Override}
	// AssertDataIsNotReplaced means that when replacing associated data in errors, we panic if a different value was already present for a given key.
	AssertDataIsNotReplaced = PreviousDataTreatment{keep: treatPreviousData_PanicOnCollision}
)

// Helps with diagnostics

// String is provided to make PreviousDataTreatment satisfy fmt.Stringer. It returns a string representing the meaning of the value.
func (s PreviousDataTreatment) String() string { // Note: Value receiver
	switch s.keep {
	case treatPreviousData_Unset:
		return "Unset value" // should we panic? I guess not, since this is just for diagnostics.
	case treatPreviousData_Override:
		return "Override old value"
	case treatPreviousData_PreferOld:
		return "Keep previous value"
	case treatPreviousData_PanicOnCollision:
		return "Panic on ambiguity"
	default:
		// cannot really happen unless users use unsafe, because we don't export the type.
		panic(fmt.Errorf(ErrorPrefix+"invalid value of PreviousDataTreatment : %v", s.keep))
	}
}

// TODO: Return error rather than panic on AssertDataIsNotReplaced?

// This particular API (modifying *target) just happens to be convenient for our purpose.

// mergeMaps modifies *target, setting it to the union of *target and source.
// source == nil is treated as an empty map.
//
// The behaviour when *target == nil is unspecified. Use an empty map for *target.
//
// The handling of duplicate map keys that appear in both maps depends on mode:
//   - mode == [PreferPreviousData]: values already in target take precendence
//   - mode == [ReplacePreviousData]: values in source take precedence
//   - mode == [AssertDataIsNotReplaced]: this function panics for duplicate keys, unless the values are comparable and equal.
func mergeMaps(target *ParamMap, source ParamMap, mode PreviousDataTreatment) {
	switch mode.keep {
	case treatPreviousData_Override:
		for key, value := range source {
			(*target)[key] = value
		}
	case treatPreviousData_PreferOld:
		for key, value := range source {
			if _, alreadyPresent := (*target)[key]; !alreadyPresent {
				(*target)[key] = value
			}
		}
	case treatPreviousData_PanicOnCollision:
		for key, value := range source {
			oldVal, alreadyPresent := (*target)[key]
			if alreadyPresent {
				// TODO: Handle incomparable types specially?
				if oldVal != value {
					panic(fmt.Errorf(ErrorPrefix+"trying to overwrite data for error when AssertDataIsNotReplaced was set.\nPrevious data: %v\nNew data:%v", oldVal, value))
				}
			} else {
				(*target)[key] = value
			}
		}
	default:
		panic(fmt.Errorf(ErrorPrefix+"called mergeMaps with invalid value %v for mode", mode))
	}
}

// NOTE: Adding entries to an existing map is more convenient for our use cases than returning a map.
// This duplicates some code from mergeMaps, but the alternative would be even more copying.

// fillMapFromStruct converts a struct of type StructType into a map[string]any.
// This function adds an entry to the provided (existing) map *m for each visible field of StructType (including from embedded structs).
// This modifies *m, converting a nil map to an empty map. This conversion happens even for empty StructType.
//
// StructType must be valid for use in this library (i.e. satisfy [StructSuitableForErrorsWithData]).
// This functions panics otherwise.
// If *m is a field inside *s (or similar shenanigans), the behaviour is undefined.
// Preexisting entries of *m that do not correspond to a field of the struct are left unchanged.
//
// Treatment of preexisting keys in *m that correspond to a field of the struct depends on mode:
//   - mode == [PreferPreviousData]: preexisting values take precendence
//   - mode == [ReplacePreviousData]: values from *s take precedence
//   - mode == [AssertDataIsNotReplaced]: panic if a key in *m corresponds to a field in struct, unless the values are (comparable and) equal.
func fillMapFromStruct[StructType any](s *StructType, m *map[string]any, mode PreviousDataTreatment) {
	if *m == nil {
		*m = make(map[string]any)
	}
	reflectedStructType := utils.TypeOfType[StructType]()
	allStructFields, err := getStructMapConversionLookup(reflectedStructType)
	if err != nil {
		panic(err)
	}
	structValue := reflect.ValueOf(s).Elem()
	switch mode.keep {
	case treatPreviousData_Override:
		for _, structField := range allStructFields {
			fieldInStruct := structValue.FieldByIndex(structField.Index).Interface()
			(*m)[structField.Name] = fieldInStruct
		}
	case treatPreviousData_PreferOld:
		for _, structField := range allStructFields {
			_, alreadyPresent := (*m)[structField.Name]
			if !alreadyPresent {
				fieldInStruct := structValue.FieldByIndex(structField.Index).Interface()
				(*m)[structField.Name] = fieldInStruct
			}
		}
	case treatPreviousData_PanicOnCollision:
		for _, structField := range allStructFields {
			oldValue, alreadyPresent := (*m)[structField.Name]
			if !alreadyPresent {
				fieldInStruct := structValue.FieldByIndex(structField.Index).Interface()
				(*m)[structField.Name] = fieldInStruct
			} else {
				fieldInStruct := structValue.FieldByIndex(structField.Index).Interface()
				// TODO: Handle incomparable types specially?
				if fieldInStruct != oldValue {
					panic(fmt.Errorf(ErrorPrefix+"trying to overwrite data for error when AssertDataIsNotReplaced was set.\nPrevious data: %v\nNew data:%v", oldValue, fieldInStruct))
				}
			}
		}
	default:
		panic(fmt.Errorf(ErrorPrefix+"called fillMapFromStruct with invalid value %v for mode", mode))
	}
}
