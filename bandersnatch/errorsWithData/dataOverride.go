package errorsWithData

import (
	"fmt"
	"reflect"

	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

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
func mergeMaps(target *ParamMap, source ParamMap, config config_OldData) (err error) {
	if !config.PerformEqualityCheck() {
		if config.PreferOld() {
			mergeMaps_preferOld(target, source)
		} else {
			mergeMaps_preferNew(target, source)
		}
		return nil
	} else {
		return mergeMaps_EqualityCheck(target, source, config)
	}
}

func mergeMaps_preferOld(target *ParamMap, source ParamMap) {
	for key, value := range source {
		if _, alreadyPresent := (*target)[key]; !alreadyPresent {
			(*target)[key] = value
		}
	}
}

func mergeMaps_preferNew(target *ParamMap, source ParamMap) {
	for key, value := range source {
		(*target)[key] = value
	}
}

func comparison_very_naive(x, y any) (equal bool, reason error) {
	return x == y, nil
}

// TODO: This is a dummy implementation. It has bad error reporting and the default comparison function does not work well.
func mergeMaps_EqualityCheck(target *ParamMap, source ParamMap, config config_OldData) (err error) {
	if !config.PerformEqualityCheck() {
		panic("Cannot happen")
	}
	checkFun := config.GetCheckFun()
	for key, newValue := range source {
		if oldValue, alreadyPresent := (*target)[key]; alreadyPresent {

			// may override anyway, depending on config
			if config.PreferNew() {
				(*target)[key] = newValue
			}
			// only report first error
			if err != nil {
				continue
			}

			comparisonEqual, reason := checkFun(oldValue, newValue)
			if !comparisonEqual {
				// TODO
				if reason == nil {
					reason = fmt.Errorf(ErrorPrefix+"%v != %v", oldValue, newValue)
				}
				err = reason
			}

		} else { // no old value, just use the new one
			(*target)[key] = newValue
		}
	}
	return
}

func mergeMaps_errorIfPresent(target *ParamMap, source ParamMap) (err error) {
	for key, value := range source {
		if _, alreadyPresent := (*target)[key]; alreadyPresent {
			if err == nil { // report first error
				err = fmt.Errorf(ErrorPrefix+"overwriting data for error when flag was set to consider this an error.\nKey value: %v\nOld value: %v\nNew value: %v", key, (*target)[key], value)
				// NO RETURN HERE. We continue, implicitly prefering new values.
			}
			(*target)[key] = value
		}
	}
	return
}

func mergeMaps_errorOnCollisionNaive(target *ParamMap, source ParamMap) (err error) {
	for key, value := range source {
		if oldVal, alreadyPresent := (*target)[key]; alreadyPresent {
			if err != nil { // only report first error.
				continue
			}
			OldType := reflect.TypeOf(oldVal)
			if !OldType.Comparable() {
				// treat oldVal as different from value, but with special error message
				err = fmt.Errorf(ErrorPrefix+"overwriting data for error when flag was set to err if oldValue != newValue. The old value for parameter named %v is incomparable", key)
			} else if oldVal != value {
				err = fmt.Errorf(ErrorPrefix+"trying to overwrite data under key %v for error when flag was set to err if oldValue != newValue.\noldValue: %v\nnewValue: %v", key, oldVal, value)
			}
		}
		(*target)[key] = value

	}
	return
}

var boolType reflect.Type = utils.TypeOfType[bool]()

func mergeMaps_errorOnCollisionomparator(target *ParamMap, source ParamMap) (err error) {
	for key, value := range source {
		if oldVal, alreadyPresent := (*target)[key]; alreadyPresent {
			if err != nil {
				// only report first error. We don't overwrite in any case, so just do nothing
				continue
			}
			oldValReflectedPtr := reflect.ValueOf(&oldVal)
			oldValReflected := oldValReflectedPtr.Elem()

			// NOTE: methodValue and methodType correspond to a function with the receiver set to the oldValue, so no explicit receiver argument.
			// It's really a reflectValue of a function (or bound method), not of a method.
			var methodValue reflect.Value
			var methodType reflect.Type

			methodValue = oldValReflectedPtr.MethodByName("IsEqual")
			if !methodValue.IsValid() {
				// try value receiver
				methodValue = oldValReflected.MethodByName("IsEqual")
				if !methodValue.IsValid() {
					goto direct_comparison
				}
			}

			methodType = methodValue.Type()

			if methodType.Kind() != reflect.Func {
				panic("cannot happen")
			}

			if methodType.NumIn() != 1 {
				panic(0)

			}
			if methodType.NumOut() != 1 {
				panic(0)
			}
			if methodType.Out(0) != boolType {
				panic(0)
			}
			{ // limit the scope of declared variables, else the above goto would not work
				newValueType := reflect.TypeOf(value)
				funcArgType := methodType.In(1)
				var resultBool bool
				if newValueType.AssignableTo(funcArgType) {
					callResult := methodValue.Call([]reflect.Value{reflect.ValueOf(value)})
					resultBool = callResult[0].Interface().(bool)
				} else if reflect.PointerTo(newValueType).AssignableTo(funcArgType) {
					callResult := methodValue.Call([]reflect.Value{reflect.ValueOf(&value)})
					resultBool = callResult[0].Interface().(bool)
				} else {
					// There is some IsEqual Method, but it's not callable
				}
				if resultBool { // [&]oldValue.IsEqual([&]Value) was callable for some choices of &'s and returned true
					// This is the good case. There is no error. Just continue with the for loop.
					continue
				} else {
					// IsEqual was callable, but returned false:
					err = fmt.Errorf(ErrorPrefix+"overwriting data under key %v for error when flag was set to err if !oldValue.IsEqual(newValue).\noldValue: %v\nnewValue: %v", key, oldVal, value)
					continue
				}
			}

		direct_comparison:
			// If we get here, no suitable IsEqual method was found.
			// methodType is unset, methodValue is invalid.
			// oldValReflected is valid
			if !oldValReflected.Type().Comparable() {
				err = fmt.Errorf(ErrorPrefix+"overwriting data for error when flag was set to err unless oldValue.IsEqual(newValue) || oldValue == newValue. There was no suitable IsEqual method and the old value for parameter named %v is incomparable", key)
				continue
			} else if oldVal != value {
				err = fmt.Errorf(ErrorPrefix+"overwriting data for error under key %v when flag was set to err unless oldValue.IsEqual(newValue) || oldValue == newValue. There was no suitable IsEqual method and the values differ.\noldValue: %v\nnewValue: %v", key, oldVal, value)
				continue
			} else {
				// values are just equal via the old-fashioned ==.
				continue
			}
		}
		(*target)[key] = value
	}
	return
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
func fillMapFromStruct[StructType any](s *StructType, m *map[string]any, config config_OldData) (err error) {
	if *m == nil {
		*m = make(map[string]any)
	}
	reflectedStructType := utils.TypeOfType[StructType]()
	allStructFields, errLookup := getStructMapConversionLookup(reflectedStructType)
	if errLookup != nil {
		panic(errLookup)
	}
	structValue := reflect.ValueOf(s).Elem()
	if !config.PerformEqualityCheck() {
		// simple case. Just prefer old / new value depending on config
		if config.PreferOld() {
			for _, structField := range allStructFields {
				_, alreadyPresent := (*m)[structField.Name]
				if !alreadyPresent {
					fieldInStruct := structValue.FieldByIndex(structField.Index).Interface()
					(*m)[structField.Name] = fieldInStruct
				}
			}
		} else { // config.preferOld not set. We always use the new value
			for _, structField := range allStructFields {
				fieldInStruct := structValue.FieldByIndex(structField.Index).Interface()
				(*m)[structField.Name] = fieldInStruct
			}
		}
		return nil // no possible error
	} else {
		// config.PerformEqualityCheck() returned true
		checkFun := config.GetCheckFun()
		for _, structField := range allStructFields {
			oldValue, alreadyPresent := (*m)[structField.Name]
			newValue := structValue.FieldByIndex(structField.Index).Interface()
			if !alreadyPresent {
				(*m)[structField.Name] = newValue
				continue
			}
			if config.PreferNew() {
				(*m)[structField.Name] = newValue
			}
			if err != nil { // only report first error
				continue
			}
			isEqual, reason := checkFun(oldValue, newValue)
			if !isEqual {
				if reason == nil {
					reason = fmt.Errorf(ErrorPrefix+"%v != %v", oldValue, newValue) // TODO: Better error message
				}
				err = reason
			}
		}
	}
	return
}
