package errorsWithData

import (
	"fmt"
	"reflect"

	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

// This particular API (modifying *target) just happens to be convenient for our purpose.

// mergeMaps modifies *target, setting it to the union of *target and source.
// source == nil is treated as an empty map.
//
// The behaviour when *target == nil is unspecified. Use an empty map for *target.
//
// The handling of duplicate map keys that appear in both maps depends on config:
//   - Either old values take precendence or new values take precendence.
//   - We might ensure that old value and new value coincide. This comparison may be performed by a custom comparison function.
//     NOTE: In the latter case, we still honor the old value vs. new value choice.
//     If old and new values do not coincide, we report errors. Note that we do not abort on first error, but rather continue and we report all errors.
//
// Note that the returned errors for this internal function do not have ErrorPrefix.
func mergeMaps(target *ParamMap, source ParamMap, config config_OldData) (errors []error) {
	// just dispatch to one of the mergeMaps_<foo> functions below.
	if !config.PerformEqualityCheck() {
		if config.PreferOld() {
			mergeMaps_preferOld(target, source)
		} else {
			mergeMaps_preferNew(target, source)
		}
		return nil // the only cases that can fail are with PerformEqualityCheck.
	} else {
		return mergeMaps_EqualityCheck(target, source, config)
	}
}

// mergeMaps_preferOld is the implementation of [mergeMaps] for the case EqualityCheck == false, PreferOld == true
func mergeMaps_preferOld(target *ParamMap, source ParamMap) {
	// set *target to the union on *target and source, prefering *target[key] for keys present in both.
	for key, value := range source {
		if _, alreadyPresent := (*target)[key]; !alreadyPresent {
			(*target)[key] = value
		}
	}
}

// mergeMaps_preferNew is the implementation of [mergeMaps] for the case EqualityCheck == false, PreferOld == false
func mergeMaps_preferNew(target *ParamMap, source ParamMap) {
	// set *target to the union on *target and source, prefering source[key] for keys present in both.
	for key, value := range source {
		(*target)[key] = value
	}
}

// mergeMaps_EqualityCheck is the implementation of [mergeMaps] for the case EqualityCheck == true.
//
// See the documentation of [mergeMaps] for its semantics.
func mergeMaps_EqualityCheck(target *ParamMap, source ParamMap, config config_OldData) (errors []error) {
	// This function is only called from [mergeMaps] if PerformEqualityCheck is true.
	// For simplicity, we just forward config as-is, rather than stripping off the PerformEqualityCheck bool.
	if !config.PerformEqualityCheck() {
		panic("Cannot happen")
	}

	var checkFun EqualityComparisonFunction = config.GetCheckFun()
	checkFunWithPanicRecovery := withPanicResults(checkFun)
	for key, newValue := range source {
		if oldValue, alreadyPresent := (*target)[key]; alreadyPresent {

			// If PreferNew is set, we always override the value, no matter what.
			// The old value is still saved in oldValue
			if config.PreferNew() {
				(*target)[key] = newValue
			}

			if config.CatchPanic() {
				// Call checkFun with panic recovery. Note that if we get a panic, then comparisonResult is guaranteed to be false.
				comparisonResult, didPanic, panicValue := checkFunWithPanicRecovery(oldValue, newValue)
				if comparisonResult == false {
					var newError error
					if !didPanic {
						// No ErrorPrefix here, no line break
						newError = fmt.Errorf("for key %v, there was already a value present that differs from the new one: old value: %v, new value: %v", key, oldValue, newValue)
					} else { // recovered panic in comparison function.
						newError = fmt.Errorf("for key %v, there was already a value present. When comparing the old and new values, a panic was encountered in the comparison function. Old value: %v, new value: %v, panic was: %v", key, oldValue, newValue, panicValue)
					}
					errors = append(errors, newError)
				}
			} else { // config.CatchPanic set to false
				comparisonResult := checkFun(oldValue, newValue)
				if comparisonResult == false {
					errors = append(errors, fmt.Errorf("for key %v, there was already a value present that differs from the new one: old value: %v, new value: %v", key, oldValue, newValue))
				}
			}

		} else { // alreadyPresent == false. So we have no entry under key in *target yet. We just use the new one.
			(*target)[key] = newValue
		}
	}
	return
}

// DEPRECATED FUNCTIONS:

/*
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

*/

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
// The meaning of config and error reporting is the same as [mergeMaps]
//
// Note that the returned errors for this internal function do not have ErrorPrefix. We return errors==nil rather than an empty list in case of success.
func fillMapFromStruct[StructType any](s *StructType, m *map[string]any, config config_OldData) (errors []error) {
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
		checkFunWithPanicRecovery := withPanicResults(checkFun)

		for _, structField := range allStructFields {
			var key string = structField.Name
			oldValue, alreadyPresent := (*m)[key]
			newValue := structValue.FieldByIndex(structField.Index).Interface()
			if !alreadyPresent {
				(*m)[key] = newValue
				continue
			}
			if config.PreferNew() {
				(*m)[key] = newValue // unconditionally write. Note there is no "continue" here.
			}

			if config.CatchPanic() {
				comparisonResult, didPanic, panicValue := checkFunWithPanicRecovery(oldValue, newValue)
				if comparisonResult == false {
					var err error
					if !didPanic {
						// No ErrorPrefix here, no line break
						err = fmt.Errorf("for key/field name %v, there was already a value present that differs from the new one: old value: %v, new value: %v", key, oldValue, newValue)
					} else { // recovered panic in comparison function.
						err = fmt.Errorf("for key/field name %v, there was already a value present. When comparing the old and new values, a panic was encountered in the comparison function. Old value: %v, new value: %v, panic was: %v", key, oldValue, newValue, panicValue)
					}
					errors = append(errors, err)
				}
			} else {
				// config.CatchPanic set to false
				comparisonResult := checkFun(oldValue, newValue)
				if comparisonResult == false {
					errors = append(errors, fmt.Errorf("for key/field name %v, there was already a value present that differs from the new one: old value: %v, new value: %v", key, oldValue, newValue))
				}
			}

		}
	}
	return
}
