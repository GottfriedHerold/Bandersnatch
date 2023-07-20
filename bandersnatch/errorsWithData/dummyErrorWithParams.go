package errorsWithData

// currently unused

/*

import (
	"fmt"
	"reflect"

	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

// dummyErrorWithData_any is a wrapper around a plain error that implements the [ErrorWithData_any] interface.
// Note that a dummyErrorWithData_any may actually have parameters (namely iff the wrapped error has).
type dummyErrorWithData_any struct {
	DummyValidator
	error
}

// dummyErrorWithData[StructType] is a wrapper around a plain error that implements the [ErrorWithData_any][StructType] interface.
//
// such an error has parameters for at least every field of StructType. Missing parameters are simply zero-intialized.
type dummyErrorWithData[StructType any] struct {
	dummyErrorWithData_any
}

// Note that Unwap gets promoted to dummyErrorWithData[StructType]

// Unwrap is provided to satisfy the error chaining mechnanism of the [errors] standard library package.
func (e dummyErrorWithData_any) Unwrap() error {
	return e.error
}

// Error_interpolate is provided to satisfy [ErrorWithData_any].
func (e dummyErrorWithData_any) Error_interpolate(p ParamMap) string {
	if errParamAware, ok := e.error.(ErrorInterpolater); ok {
		return errParamAware.Error_interpolate(p)
	}
	return e.Error()
}

func (e dummyErrorWithData_any) HasParameter(parameterName string) bool {
	return HasParameter(e.error, parameterName)
}

func (e dummyErrorWithData[StructType]) HasParameter(parameterName string) bool {

	for _, structField := range getStructMapConversionLookup(utils.TypeOfType[StructType]()) {
		if parameterName == structField.Name {
			return true
		}
	}
	return HasParameter(e.error, parameterName)
}

func (e dummyErrorWithData_any) GetParameter(parameterName string) (value any, wasPresent bool) {
	return GetParameter(e.error, parameterName)
}

func (e dummyErrorWithData[StructType]) GetParameter(parameterName string) (value any, wasPresent bool) {
	value, wasPresent = GetParameter(e.error, parameterName)
	if wasPresent {
		return
	}

	for _, structField := range getStructMapConversionLookup(utils.TypeOfType[StructType]()) {
		if parameterName == structField.Name {
			var zero StructType
			value = zero
			wasPresent = true
			return
		}
	}
	return
}

func (e dummyErrorWithData_any) GetData_map() ParamMap {
	return GetData_map(e.error)
}

func (e dummyErrorWithData[StructType]) GetData_map() (ret ParamMap) {
	ret = GetData_map(e.error)
	for _, structField := range getStructMapConversionLookup(utils.TypeOfType[StructType]()) {
		_, ok := ret[structField.Name]
		if !ok {
			var zero StructType
			ret[structField.Name] = zero
		}
	}
	return
}

func (e dummyErrorWithData[StructType]) GetData_struct() (ret StructType) {
	// ret is zero - initialized
	params := e.GetData_map()
	if len(params) == 0 {
		return
	}

	// We expect to rarely hit this.
	// If there are already parameters present from the error, we actually use these
	// rather than use a zero value.
	// The code here is essentiall the same as makeStructFromMap,
	// except for error handling:
	//  - we panic on failure (type mismatches)
	//  - missing parameters are not considered a failure

	retReflected := reflect.ValueOf(&ret).Elem()
	for _, structField := range getStructMapConversionLookup(utils.TypeOfType[StructType]()) {
		fieldInRetValue := retReflected.FieldByIndex(structField.Index)
		valueFromMap, ok := params[structField.Name]
		if !ok {
			continue // keep the field at its zero value
		}
		// This is annoying, but Go1.18 requires special-casing here.
		// There is even a discussion to change the behaviour of reflect.ValueOf(nil)
		// cf. https://github.com/golang/go/issues/51649
		if valueFromMap == nil { // nil interface in map
			if utils.IsNilable(fieldInRetValue.Type()) {
				appropriateNil := reflect.Zero(fieldInRetValue.Type())
				fieldInRetValue.Set(appropriateNil)
			} else {
				panic(fmt.Errorf(ErrorPrefix+
					"trying to construct value of type %v from parameters; parameter named %v is set to nil, which is not valid for the struct field",
					utils.GetReflectName(retReflected.Type()), structField.Name))
			}
		} else { // valueFromMap non-nil (i.e. value in map is not nil interface)
			// reflect.Value.Set only requires assignability, not equality of types.
			// We want perfect roundtripping without conversion, so we need
			// type equality for concrete types, but assignability for struct fields of interface type.
			if fieldInRetValue.Kind() == reflect.Interface {
				if !reflect.TypeOf(valueFromMap).AssignableTo(fieldInRetValue.Type()) {
					panic(fmt.Errorf(ErrorPrefix+
						"trying to construct value of type %v from parameters; parameter named %v is not assignable type: expected %v, got %v",
						utils.GetReflectName(retReflected.Type()), structField.Name, fieldInRetValue.Type(), reflect.TypeOf(valueFromMap)))

				}
			} else { // non-interface type for the field
				if fieldInRetValue.Type() != reflect.TypeOf(valueFromMap) {
					panic(fmt.Errorf(ErrorPrefix+" trying to construct value of type %v from parameters; parameter named %v is of wrong type: expected %v, got %v ",
						utils.GetReflectName(retReflected.Type()), structField.Name, fieldInRetValue.Type(), reflect.TypeOf(valueFromMap)))

				}
			}
			fieldInRetValue.Set(reflect.ValueOf(valueFromMap))
		}
	}
	return
}

*/
