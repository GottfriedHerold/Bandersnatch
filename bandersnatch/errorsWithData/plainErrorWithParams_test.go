package errorsWithData

import (
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
	testPanicOnConcreteNil[*errorWithParameters_T[struct{ Asdasfga int }]](t)
}
