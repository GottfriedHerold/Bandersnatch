package errorsWithData

import (
	"fmt"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
)

func TestNewErrorWithData_struct(t *testing.T) {
	type invalid struct{ _ int } // unexported
	type DataXY struct{ X, Y int }
	type DataXZ struct{ X, Z int }
	type DataX struct{ X uint } // note different type
	type empty struct{}

	didPanic := testutils.CheckPanic(func() { NewErrorWithData_struct(nil, "foo", &invalid{}) }) // defining a local function because CheckPanic does not handle variadics well
	testutils.FatalUnless(t, didPanic, "")

	error1, err := NewErrorWithData_struct(nil, "Error1", &DataX{X: 5})
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError(t, error1, "Error1", &DataX{X: 5}, ParamMap{"X": uint(5)}, nil)

	didPanic = testutils.CheckPanic(func() { NewErrorWithData_struct(nil, "", &empty{}) })
	testutils.FatalUnless(t, didPanic, "%v", err)
	didPanic = testutils.CheckPanic(func() { NewErrorWithData_struct(nil, "", &empty{}, ReturnError) })
	testutils.FatalUnless(t, didPanic, "%v", err)
	error2, err := NewErrorWithData_struct(nil, "", &empty{}, AllowEmptyString)
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError(t, error2, "", &empty{}, ParamMap{}, nil)

	error3, err := NewErrorWithData_struct(nil, "XOwn:%{X},X:${X}", &DataX{X: 1})
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError(t, error3, "XOwn:1,X:1", &DataX{X: 1}, ParamMap{"X": uint(1)}, nil)

	error4, err := NewErrorWithData_struct(error3, "", &DataXY{X: 2, Y: 3})
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError(t, error4, "XOwn:1,X:2", &DataXY{X: 2, Y: 3}, ParamMap{"X": 2, "Y": 3}, []error{error3})

	error5, err := NewErrorWithData_struct(error3, "", &DataXY{X: 2, Y: 3}, PreferPreviousData) // fails, because X is taken from error3 and has the wrong type.
	testutils.FatalUnless(t, err != nil, "")
	testError(t, error5, "XOwn:1,X:0", &DataXY{X: 0, Y: 3}, ParamMap{"X": 0, "Y": 3}, []error{error3})

	didPanic, err2 := testutils.CheckPanic2(func() { NewErrorWithData_struct(error3, "", &DataXY{X: 2, Y: 3}, PreferPreviousData, PanicOnAllErrors) })
	testutils.FatalUnless(t, didPanic, "")
	testutils.FatalUnless(t, err.Error() == err2.(error).Error(), "%v\n!=\n%v", err, err2) // check that panic value is the same as the returned error

	error6, err := NewErrorWithData_struct(nil, "X:%{X}Z:%{Z}", &DataXY{X: 1, Y: 2}, ErrorUnlessValidBase)
	testutils.FatalUnless(t, err != nil, "")
	testError(t, error6, "ignore", &DataXY{X: 1, Y: 2}, ParamMap{"X": 1, "Y": 2}, nil)

	error7, err := NewErrorWithData_struct(nil, "X:%{X}Z:${Z}", &DataXY{X: 1, Y: 2}, ErrorUnlessValidBase)
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError(t, error7, "ignore", &DataXY{X: 1, Y: 2}, ParamMap{"X": 1, "Y": 2}, nil)
	testutils.FatalUnless(t, error7.ValidateError_Final() != nil, "")

	error8, err := NewErrorWithData_struct(error7, "", &DataXZ{X: 3, Z: 4})
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError(t, error8, "X:1Z:4", &DataXZ{X: 3, Z: 4}, ParamMap{"X": 3, "Y": 2, "Z": 4}, []error{error7})

	error9, err := NewErrorWithData_struct(error7, "$w${U}", &DataXZ{X: 3, Z: 4}, EnsureDataIsNotReplaced, ErrorUnlessValidFinal) // 2 types of error
	testutils.FatalUnless(t, err != nil, "")
	// fmt.Println(err) -- looks good
	testError(t, error9, "ignore", &DataXZ{X: 3, Z: 4}, ParamMap{"X": 3, "Y": 2, "Z": 4}, []error{error7}) // Note: X gets replaced.
}

func TestNewErrorWithData_params(t *testing.T) {
	type invalid struct{ _ int } // unexported
	type DataXY struct{ X, Y int }
	type DataXZ struct{ X, Z int }
	type DataX struct{ X uint } // note different type
	type empty struct{}

	didPanic := testutils.CheckPanic(func() { NewErrorWithData_params[invalid](nil, "foo") })
	testutils.FatalUnless(t, didPanic, "")
	didPanic, panicVal := testutils.CheckPanic2(func() { NewErrorWithData_params[empty](nil, "foo") })
	testutils.FatalUnless(t, didPanic == false, "%v", panicVal)
	didPanic = testutils.CheckPanic(func() { NewErrorWithData_params[empty](nil, "foo", "bar") })
	testutils.FatalUnless(t, didPanic == true, "")
	didPanic, panicVal = testutils.CheckPanic2(func() { NewErrorWithData_params[empty](nil, "foo", nil) })
	testutils.FatalUnless(t, didPanic == true, "")
	didPanic, panicVal = testutils.CheckPanic2(func() { NewErrorWithData_params[empty](nil, "foo", EnsureDataIsPresent) }) // invalid flag for this function
	testutils.FatalUnless(t, didPanic == true, "")
	didPanic, panicVal = testutils.CheckPanic2(func() { NewErrorWithData_params[empty](nil, "") }) // empty error without AllowEmptyString
	testutils.FatalUnless(t, didPanic == true, "")

	error1, err := NewErrorWithData_params[DataX](nil, "Error1", "X", uint(5))
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError(t, error1, "Error1", &DataX{X: 5}, ParamMap{"X": uint(5)}, nil)

	error2, err := NewErrorWithData_params[empty](nil, "", AllowEmptyString)
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError(t, error2, "", &empty{}, ParamMap{}, nil)

	error3, err := NewErrorWithData_params[DataX](nil, "XOwn:%{X},X:${X}", "X", uint(1))
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError(t, error3, "XOwn:1,X:1", &DataX{X: 1}, ParamMap{"X": uint(1)}, nil)

	error4, err := NewErrorWithData_params[DataXY](error3, "", "X", 2, "Y", 3)
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError(t, error4, "XOwn:1,X:2", &DataXY{X: 2, Y: 3}, ParamMap{"X": 2, "Y": 3}, []error{error3})

	error5, err := NewErrorWithData_params[DataXY](error3, "", "X", 2, PreferPreviousData, "Y", 3) // fails, because X is taken from error3 and has the wrong type.
	testutils.FatalUnless(t, err != nil, "")
	testError(t, error5, "XOwn:1,X:0", &DataXY{X: 0, Y: 3}, ParamMap{"X": 0, "Y": 3}, []error{error3})

	error6, err := NewErrorWithData_params[DataXY](nil, "X:%{X}Z:%{Z}", "X", 1, "Y", 2, ErrorUnlessValidBase)
	testutils.FatalUnless(t, err != nil, "")
	testError(t, error6, "ignore", &DataXY{X: 1, Y: 2}, ParamMap{"X": 1, "Y": 2}, nil)

	error7, err := NewErrorWithData_params[DataXY](nil, "X:%{X}Z:${Z}", "X", 1, "Y", 2, ErrorUnlessValidBase)
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError(t, error7, "ignore", &DataXY{X: 1, Y: 2}, ParamMap{"X": 1, "Y": 2}, nil)
	testutils.FatalUnless(t, error7.ValidateError_Final() != nil, "")

	error8, err := NewErrorWithData_params[DataXZ](error7, "", "X", 3, "Z", 4)
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError(t, error8, "X:1Z:4", &DataXZ{X: 3, Z: 4}, ParamMap{"X": 3, "Y": 2, "Z": 4}, []error{error7})

	error9, err := NewErrorWithData_params[DataXZ](error7, "$w${U}", "X", 3, "Z", 4, EnsureDataIsNotReplaced, ErrorUnlessValidFinal) // 2 types of error
	testutils.FatalUnless(t, err != nil, "")
	// fmt.Println(err) -- looks good
	testError(t, error9, "ignore", &DataXZ{X: 3, Z: 4}, ParamMap{"X": 3, "Y": 2, "Z": 4}, []error{error7}) // Note: X gets replaced.

	error10, err := NewErrorWithData_params[DataXZ](nil, "S", "X", 1, "Y", 5) // missing data: Z
	testutils.FatalUnless(t, err != nil, "")
	testError(t, error10, "S", &DataXZ{X: 1, Z: 0}, ParamMap{"X": 1, "Y": 5, "Z": 0}, nil)

	error11, err := NewErrorWithData_params[DataXZ](nil, "S", "X", 1, "Y", 5, MissingDataAsZero) // missing data: Z, but ignore this
	testutils.FatalUnless(t, err == nil, "")
	testError(t, error11, "S", &DataXZ{X: 1, Z: 0}, ParamMap{"X": 1, "Y": 5, "Z": 0}, nil)

	// fmt.Println(panicVal)
	_ = panicVal
}

func TestNewErrorWithDataMap(t *testing.T) {
	type invalid struct{ _ int } // unexported
	type DataXY struct{ X, Y int }
	type DataXZ struct{ X, Z int }
	type DataX struct{ X uint } // note different type
	type empty struct{}

	didPanic := testutils.CheckPanic(func() { NewErrorWithData_map[invalid](nil, "foo", ParamMap{}) })
	testutils.FatalUnless(t, didPanic, "")
	didPanic, panicVal := testutils.CheckPanic2(func() { NewErrorWithData_map[empty](nil, "foo", nil) })
	testutils.FatalUnless(t, didPanic == false, "%v", panicVal)
	didPanic, panicVal = testutils.CheckPanic2(func() { NewErrorWithData_map[empty](nil, "", nil) }) // empty error without AllowEmptyString
	testutils.FatalUnless(t, didPanic == true, "")

	didPanic, panicVal = testutils.CheckPanic2(func() { NewErrorWithData_map[empty](nil, "foo", nil, nil) })
	testutils.FatalUnless(t, didPanic == true, "")
	didPanic, panicVal = testutils.CheckPanic2(func() { NewErrorWithData_map[empty](nil, "foo", nil, nil...) })
	testutils.FatalUnless(t, didPanic == false, "")

	error1, err := NewErrorWithData_map[DataX](nil, "Error1", ParamMap{"X": uint(5)})
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError(t, error1, "Error1", &DataX{X: 5}, ParamMap{"X": uint(5)}, nil)

	error2, err := NewErrorWithData_map[empty](nil, "", nil, AllowEmptyString)
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError(t, error2, "", &empty{}, ParamMap{}, nil)

	error3, err := NewErrorWithData_map[DataX](nil, "XOwn:%{X},X:${X}", ParamMap{"X": uint(1)})
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError(t, error3, "XOwn:1,X:1", &DataX{X: 1}, ParamMap{"X": uint(1)}, nil)

	error4, err := NewErrorWithData_map[DataXY](error3, "", ParamMap{"X": 2, "Y": 3})
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError(t, error4, "XOwn:1,X:2", &DataXY{X: 2, Y: 3}, ParamMap{"X": 2, "Y": 3}, []error{error3})

	error5, err := NewErrorWithData_map[DataXY](error3, "", ParamMap{"X": 2, "Y": 3}, PreferPreviousData) // fails, because X is taken from error3 and has the wrong type.
	testutils.FatalUnless(t, err != nil, "")
	testError(t, error5, "XOwn:1,X:0", &DataXY{X: 0, Y: 3}, ParamMap{"X": 0, "Y": 3}, []error{error3})

	error6, err := NewErrorWithData_map[DataXY](nil, "X:%{X}Z:%{Z}", ParamMap{"X": 1, "Y": 2}, ErrorUnlessValidBase)
	testutils.FatalUnless(t, err != nil, "")
	testError(t, error6, "ignore", &DataXY{X: 1, Y: 2}, ParamMap{"X": 1, "Y": 2}, nil)

	error7, err := NewErrorWithData_map[DataXY](nil, "X:%{X}Z:${Z}", ParamMap{"X": 1, "Y": 2}, ErrorUnlessValidBase)
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError(t, error7, "ignore", &DataXY{X: 1, Y: 2}, ParamMap{"X": 1, "Y": 2}, nil)
	testutils.FatalUnless(t, error7.ValidateError_Final() != nil, "")

	error8, err := NewErrorWithData_map[DataXZ](error7, "", ParamMap{"X": 3, "Z": 4})
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError(t, error8, "X:1Z:4", &DataXZ{X: 3, Z: 4}, ParamMap{"X": 3, "Y": 2, "Z": 4}, []error{error7})

	error9, err := NewErrorWithData_map[DataXZ](error7, "$w${U}", ParamMap{"X": 3, "Z": 4}, EnsureDataIsNotReplaced, ErrorUnlessValidFinal) // 2 types of error
	testutils.FatalUnless(t, err != nil, "")
	// fmt.Println(err) -- looks good
	testError(t, error9, "ignore", &DataXZ{X: 3, Z: 4}, ParamMap{"X": 3, "Y": 2, "Z": 4}, []error{error7}) // Note: X gets replaced.

	didPanic, panicVal = testutils.CheckPanic2(func() {
		NewErrorWithData_map[DataXZ](error7, "$w${U}", ParamMap{"X": 3, "Z": 4}, EnsureDataIsNotReplaced, ErrorUnlessValidFinal, PanicOnAllErrors) // same as above
	})
	testutils.FatalUnless(t, didPanic == true, "")
	testutils.FatalUnless(t, panicVal.(error).Error() == err.Error(), "")

	error10, err := NewErrorWithData_map[DataXZ](nil, "S", ParamMap{"X": 1, "Y": 5}) // missing data: Z
	testutils.FatalUnless(t, err != nil, "")
	testError(t, error10, "S", &DataXZ{X: 1, Z: 0}, ParamMap{"X": 1, "Y": 5, "Z": 0}, nil)

	error11, err := NewErrorWithData_map[DataXZ](nil, "S", ParamMap{"X": 1, "Y": 5}, MissingDataAsZero) // missing data: Z, but ignore this
	testutils.FatalUnless(t, err == nil, "")
	testError(t, error11, "S", &DataXZ{X: 1, Z: 0}, ParamMap{"X": 1, "Y": 5, "Z": 0}, nil)

	// fmt.Println(panicVal)
	_ = panicVal
}

func TestDeleteParameterFromError_any(t *testing.T) {
	type invalid struct{ _ int } // unexported
	type DataXY struct{ X, Y int }
	type DataXZ struct{ X, Z int }
	type DataX struct{ X uint } // note different type
	type empty struct{}

	error1, err := DeleteParameterFromError_any(nil, "")
	testutils.FatalUnless(t, error1 == nil, "%v", error1)
	testutils.FatalUnless(t, err == nil, "%v", err)

	didPanic, panicVal := testutils.CheckPanic2(func() { DeleteParameterFromError_any(nil, "", AllowEmptyString) })
	testutils.FatalUnless(t, didPanic == true, "")
	didPanic, panicVal = testutils.CheckPanic2(func() { DeleteParameterFromError_any(nil, "foo", 5) })
	testutils.FatalUnless(t, didPanic == true, "")
	didPanic, panicVal = testutils.CheckPanic2(func() { DeleteParameterFromError_any(nil, "foo", "param", nil) })
	testutils.FatalUnless(t, didPanic == true, "")
	didPanic, panicVal = testutils.CheckPanic2(func() { DeleteParameterFromError_any(nil, "foo", "param", MissingDataAsZero) }) // invalid flag for this function
	testutils.FatalUnless(t, didPanic == true, "")

	error2, err := DeleteParameterFromError_any(nil, "err", nil...)
	testError_any(t, error2, "err", ParamMap{}, nil)

	errBase, _ := NewErrorWithData_struct(nil, "${X} %{X}", &DataX{X: 5}, ErrorUnlessValidFinal, PanicOnAllErrors)
	testError(t, errBase, "5 5", &DataX{X: 5}, ParamMap{"X": uint(5)}, nil)

	error3, err := DeleteParameterFromError_any(errBase, "", "X")
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError_any(t, error3, "$v!<missing value> 5", ParamMap{}, []error{errBase})

	error4, err := DeleteParameterFromError_any(errBase, "", "X", "Y", ErrorUnlessValidFinal) // same as error3, but notice there is an error
	testutils.FatalUnless(t, err != nil, "")
	testError_any(t, error4, "$v!<missing value> 5", ParamMap{}, []error{errBase})
	fmt.Println(err)

	error5, err := DeleteParameterFromError_any(errBase, "", "X", "Y", ErrorUnlessValidBase) // same as error3,error4, but we expect no err
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError_any(t, error5, "$v!<missing value> 5", ParamMap{}, []error{errBase})

	errBase2, err := NewErrorWithData_struct(errBase, "${X} ${Y}", &DataXY{X: 1, Y: 2})
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError(t, errBase2, "1 2", &DataXY{X: 1, Y: 2}, ParamMap{"X": 1, "Y": 2}, []error{errBase})

	error6, err := DeleteParameterFromError_any(errBase2, "${Y}", "X", ErrorUnlessValidFinal, PanicOnAllErrors)
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError_any(t, error6, "2", ParamMap{"Y": 2}, []error{errBase2})

	_ = panicVal
}
