package errorsWithData

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

var (
	_ ErrorWithData_any                = &errorWithParameters_common{}
	_ ErrorWithData[struct{ Foo int }] = &errorWithParameters_T[struct{ Foo int }]{}
)

func TestForgetStructType(t *testing.T) {

	var e_T ErrorWithData[struct{ foo int }] = &errorWithParameters_T[struct{ foo int }]{}
	_, ok := forgetStructType(e_T).(*errorWithParameters_common)
	testutils.FatalUnless(t, ok, "")

	// testing other branch of forgetStructType would require a mock implementation of ErrorWithData -- not worth it.
}

// helper function for TestPanicOnNilValues.
// not defined as a lambda, because it's generic.

func testPanicOnConcreteNil[E ErrorWithData_any](t *testing.T) {
	EType := utils.TypeOfType[E]()
	if EType.Kind() == reflect.Interface {
		panic("called testPanicOnConcreteNil with interface type")
	}
	nilable := utils.IsNilable(EType)
	if !nilable {
		return
	}
	var zeroOfE E
	zero := ErrorWithData_any(zeroOfE)
	zeroOfEVal := reflect.ValueOf(zeroOfE)
	if !zeroOfEVal.IsNil() {
		panic("cannot happen")
	}
	testutils.FatalUnless(t, testutils.CheckPanic(zero.Error_interpolate, ParamMap{}), "No panic when calling Error_interpolate on typed nil value")
	testutils.FatalUnless(t, testutils.CheckPanic(zero.GetParameter, "foo"), "No panic when calling GetParameter on typed nil value")
	testutils.FatalUnless(t, testutils.CheckPanic(zero.HasParameter, "foo"), "No panic when calling HasParameter on typed nil value")
	testutils.FatalUnless(t, testutils.CheckPanic(zero.GetData_map), "No panic when calling GetData_map on typed nil value")
	testutils.FatalUnless(t, testutils.CheckPanic(zero.ValidateSyntax), "No panic when calling ValidateSyntax on typed nil value")
	testutils.FatalUnless(t, testutils.CheckPanic(zero.ValidateError_Final), "No panic when calling ValidateError_Final on typed nil value")
	testutils.FatalUnless(t, testutils.CheckPanic(zero.ValidateError_Base), "No panic when calling ValidateError_Base on typed nil value")
	testutils.FatalUnless(t, testutils.CheckPanic(zero.ValidateError_Params, ParamMap{}), "No panic when calling ValidateError_Params on typed nil value")
}

func TestPanicOnNilValues(t *testing.T) {
	testPanicOnConcreteNil[*errorWithParameters_common](t)
	testPanicOnConcreteNil[*errorWithParameters_T[struct{}]](t)
	testPanicOnConcreteNil[*errorWithParameters_T[struct{ _ int }]](t)
}

func TestErrorInterpolateForConcreteImplementation(t *testing.T) {
	paramMapEmpty := ParamMap{}
	paramMap1 := ParamMap{"Foo": 5}

	baseErr := fmt.Errorf("BASE")
	base2 := &errorWithParameters_common{
		wrapped_error:             nil,
		parsedInterpolationString: make_ast_successfully("Foo%{Foo}${Foo}"),
		params:                    ParamMap{"Foo": 4},
	}

	base3 := &errorWithParameters_common{
		wrapped_error:             base2,
		parsedInterpolationString: make_ast_successfully("$w${Bar}"),
		params:                    ParamMap{"Foo": 3},
	}

	testutils.FatalUnless(t, base2.Error() == "Foo44", "")
	testutils.FatalUnless(t, base2.Error_interpolate(paramMap1) == "Foo45", "")
	testutils.FatalUnless(t, base2.ValidateSyntax() == nil, "Unexpected error: %v", base2.ValidateSyntax())
	testutils.FatalUnless(t, base2.ValidateError_Base() == nil, "Unexpected error: %v", base2.ValidateError_Base())
	testutils.FatalUnless(t, base2.ValidateError_Final() == nil, "Unexpected error: %v", base2.ValidateError_Final())
	testutils.FatalUnless(t, base2.ValidateError_Params(paramMapEmpty) != nil, "Unexpectedly got no error")
	testutils.FatalUnless(t, base2.ValidateError_Params(paramMap1) == nil, "Unexpected error: %v", base2.ValidateError_Params(paramMap1))

	testutils.FatalUnless(t, base3.ValidateSyntax() == nil, "")
	testutils.FatalUnless(t, base3.ValidateError_Base() == nil, "")
	testutils.FatalUnless(t, base3.ValidateError_Final() != nil, "")
	testutils.FatalUnless(t, base3.ValidateError_Params(paramMapEmpty) != nil, "")
	testutils.FatalUnless(t, base3.ValidateError_Params(paramMap1) != nil, "")
	testutils.FatalUnless(t, base3.ValidateError_Params(ParamMap{"Foo": 1, "Bar": 2}) == nil, "")
	testutils.FatalUnless(t, base3.Error_interpolate(ParamMap{"Foo": 1, "Bar": 2}) == "Foo412", "")

	err1 := makeErrorWithParametersCommon_any(baseErr, "%w")
	testutils.FatalUnless(t, len(err1.params) == 0, "")
	testutils.FatalUnless(t, err1.params != nil, "nil map rather than empty map")
	testutils.FatalUnless(t, err1.Error() == "BASE", "")
	testutils.FatalUnless(t, err1.Error_interpolate(paramMapEmpty) == "BASE", "")
	testutils.FatalUnless(t, err1.Error_interpolate(paramMap1) == "BASE", "")
	testutils.FatalUnless(t, err1.ValidateError_Base() == nil, "")
	testutils.FatalUnless(t, err1.ValidateSyntax() == nil, "")
	testutils.FatalUnless(t, err1.ValidateError_Final() == nil, "")
	testutils.FatalUnless(t, err1.ValidateError_Params(paramMapEmpty) == nil, "")
	testutils.FatalUnless(t, err1.ValidateError_Params(paramMap1) == nil, "")

	// err2 := makeErrorWithParametersCommon(base2, "$w")
}

func TestUnwrapForErrorWithParameters(t *testing.T) {
	baseErr := fmt.Errorf("BASE")
	err1 := &errorWithParameters_common{
		wrapped_error:             baseErr,
		parsedInterpolationString: make_ast_successfully("Foo%{Foo}${Foo}"),
		params:                    ParamMap{"Foo": 4},
	}

	err2 := &errorWithParameters_common{
		wrapped_error:             err1,
		parsedInterpolationString: make_ast_successfully("$w${Bar}"),
		params:                    ParamMap{"Foo": 3},
	}

	err3 := &errorWithParameters_T[struct{ Foo int }]{
		errorWithParameters_common: errorWithParameters_common{
			wrapped_error:             baseErr,
			parsedInterpolationString: make_ast_successfully("%v{Foo}"),
			params:                    ParamMap{"Foo": 1},
		},
	}

	testutils.FatalUnless(t, errors.Is(err1, err1), "")
	testutils.FatalUnless(t, errors.Is(err1, baseErr), "")
	testutils.FatalUnless(t, !errors.Is(err1, err2), "")
	testutils.FatalUnless(t, !errors.Is(err1, err3), "")

	testutils.FatalUnless(t, errors.Is(err2, err2), "")
	testutils.FatalUnless(t, errors.Is(err2, err1), "")
	testutils.FatalUnless(t, errors.Is(err2, baseErr), "")
	testutils.FatalUnless(t, !errors.Is(err2, err3), "")

	testutils.FatalUnless(t, errors.Is(err3, baseErr), "")

}

func TestGetData_structForErrorWithParameters(t *testing.T) {
	baseErr := fmt.Errorf("BASE")
	err1 := &errorWithParameters_common{
		wrapped_error:             baseErr,
		parsedInterpolationString: make_ast_successfully("Foo%{Foo}${Foo}"),
		params:                    ParamMap{"Foo": 4, "Foo2": -1},
	}

	type T struct {
		Foo int
		Bar int
	}

	errT1 := &errorWithParameters_T[T]{
		errorWithParameters_common: errorWithParameters_common{
			wrapped_error:             err1,
			parsedInterpolationString: make_ast_successfully("%v{Foo}"),
			params:                    ParamMap{"Bar": 1, "Baz": 10},
		},
	}

	errT2 := &errorWithParameters_T[T]{
		errorWithParameters_common: errorWithParameters_common{
			wrapped_error:             err1,
			parsedInterpolationString: make_ast_successfully("%v{Foo}"),
			params:                    ParamMap{"Foo": 5, "Bar": 6, "Baz": 11},
		},
	}

	testutils.FatalUnless(t, testutils.CheckPanic(errT1.GetData_struct) == true, "")
	testutils.FatalUnless(t, errT2.GetData_struct() == T{Foo: 5, Bar: 6}, "")

	M1 := errT1.GetData_map()
	M2 := errT2.GetData_map()
	M22 := errT2.GetData_map()

	testutils.FatalUnless(t, len(M1) == 2, "")
	testutils.FatalUnless(t, len(M2) == 3, "")

	testutils.FatalUnless(t, M1["Bar"] == 1 && M1["Baz"] == 10, "")
	testutils.FatalUnless(t, M2["Foo"] == 5 && M2["Bar"] == 6 && M2["Baz"] == 11, "")
	testutils.FatalUnless(t, M22["Foo"] == 5 && M22["Bar"] == 6 && M22["Baz"] == 11, "")

	M22["Foo"] = 3
	testutils.FatalUnless(t, M2["Foo"] == 5, "Aliasing issue with output of GetData_map")

	M23 := errT2.GetData_map()

	testutils.FatalUnless(t, M23["Foo"] == 5, "Aliasing issue with output of GetData_map")

}

// NOTE: Test for the unexported newErrorWithData_* and deleteParameterFromError_* methods
// are covered by test for the calling exported functions.
