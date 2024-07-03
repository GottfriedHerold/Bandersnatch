package errorsWithData

import (
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
	didPanic = testutils.CheckPanic(func() { NewErrorWithData_struct(nil, "", &empty{}, ReturnMistake) })
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

	didPanic, err2 := testutils.CheckPanic2(func() {
		NewErrorWithData_struct(error3, "", &DataXY{X: 2, Y: 3}, PreferPreviousData, PanicOnAllMistakes)
	})
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

	error9, err := NewErrorWithData_struct(error7, "$w${U}", &DataXZ{X: 3, Z: 4}, MistakeIfDataIsReplaced, ErrorUnlessValidFinal) // 2 types of error
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

	error9, err := NewErrorWithData_params[DataXZ](error7, "$w${U}", "X", 3, "Z", 4, MistakeIfDataIsReplaced, ErrorUnlessValidFinal) // 2 types of error
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

func TestAddDataToError(t *testing.T) {
	type invalid struct{ _ int } // unexported
	type DataXY struct{ X, Y int }
	type DataXZ struct{ X, Z int }
	type DataX struct{ X uint } // note different type
	type empty struct{}

	errBase1, _ := NewErrorWithData_params[DataX](nil, "Err ${X} %{X}", "X", uint(5), PanicOnAllMistakes)
	for _, flag := range validFlags_AddErrorToData {
		e, mistake := AddDataToError(errBase1, flag)
		testutils.FatalUnless(t, e.Error() == "Err 5 5", "%v", e.Error())
		testutils.FatalUnless(t, mistake == nil, "%v", mistake)

		e, mistake = AddDataToError[DataX](nil, flag)
		testutils.FatalUnless(t, e == nil, "")
		testutils.FatalUnless(t, mistake == nil, "%v", mistake)
	}

	didPanic := testutils.CheckPanic(func() {
		var e ErrorWithData[invalid] = &errorWithParameters_T[invalid]{}
		AddDataToError(e)
	})
	testutils.FatalUnless(t, didPanic == true, "")

	didPanic = testutils.CheckPanic(func() {
		AddDataToError(errBase1, MissingDataAsZero) // invalid parameter
	})
	testutils.FatalUnless(t, didPanic == true, "")

	didPanic = testutils.CheckPanic(func() {
		AddDataToError(errBase1, "Arg") // missing value
	})
	testutils.FatalUnless(t, didPanic == true, "")

	didPanic = testutils.CheckPanic(func() {
		AddDataToError(errBase1, "Arg", 5, 5) // non-string
	})
	testutils.FatalUnless(t, didPanic == true, "")

	error1, err := AddDataToError(errBase1)
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError(t, error1, "Err 5 5", &DataX{X: 5}, ParamMap{"X": uint(5)}, []error{errBase1})

	error2, err := AddDataToError(errBase1, "X", uint(6), "Y", 7)
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError(t, error2, "Err 6 5", &DataX{X: 6}, ParamMap{"X": uint(6), "Y": 7}, []error{errBase1})

	error3, err := AddDataToError(errBase1, "X", 5, "Y", 8) // 5 is int, not uint
	testutils.FatalUnless(t, err != nil, "")                // expect error
	testError(t, error3, "ignore", &DataX{X: 0}, ParamMap{"X": uint(0), "Y": int(8)}, []error{errBase1})

	errBase2, _ := NewErrorWithData_struct(nil, "${X} ${Y}", &DataXZ{}, PanicOnAllMistakes)

	error4, err := AddDataToError(errBase2, "Y", 5)
	testutils.FatalUnless(t, err == nil, "")
	testError(t, error4, "0 5", &DataXZ{}, ParamMap{"X": 0, "Y": 5, "Z": 0}, []error{errBase2})

	error5, err := AddDataToError(errBase2, "Z", 5, ErrorUnlessValidFinal)
	testutils.FatalUnless(t, err != nil, "") // missing data
	testError(t, error5, "ignore", &DataXZ{Z: 5}, ParamMap{"X": 0, "Z": 5}, []error{errBase2})

	didPanic = testutils.CheckPanic(func() {
		AddDataToError(errBase2, "Z", 5, ErrorUnlessValidFinal, PanicOnAllMistakes)
	})
	testutils.FatalUnless(t, didPanic == true, "")

	error6, err := AddDataToError(errBase2, "Z", "Z", ErrorUnlessValidFinal) // wrong type (string) AND missing data
	testutils.FatalUnless(t, err != nil, "")
	testError(t, error6, "ignore", &DataXZ{Z: 0}, ParamMap{"X": 0, "Z": 0}, []error{errBase2})

}

func TestNewErrorWithData_any_params(t *testing.T) {
	type invalid struct{ _ int } // unexported
	type DataXY struct{ X, Y int }
	type DataXZ struct{ X, Z int }
	type DataX struct{ X uint } // note different type
	type empty struct{}

	didPanic := testutils.CheckPanic(func() { NewErrorWithData_any_params(nil, "foo", "bar") }) // invalid argument
	testutils.FatalUnless(t, didPanic == true, "")
	didPanic, _ = testutils.CheckPanic2(func() { NewErrorWithData_any_params(nil, "foo", nil) }) // invalid argument
	testutils.FatalUnless(t, didPanic == true, "")
	didPanic, _ = testutils.CheckPanic2(func() { NewErrorWithData_any_params(nil, "foo", EnsureDataIsPresent) }) // invalid flag for this function
	testutils.FatalUnless(t, didPanic == true, "")
	didPanic, _ = testutils.CheckPanic2(func() { NewErrorWithData_any_params(nil, "") }) // empty error without AllowEmptyString
	testutils.FatalUnless(t, didPanic == true, "")

	error1, err := NewErrorWithData_any_params(nil, "Error1", "X", 5)
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError_any(t, error1, "Error1", ParamMap{"X": 5}, nil)

	error2, err := NewErrorWithData_any_params(nil, "", AllowEmptyString)
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError_any(t, error2, "", ParamMap{}, nil)

	error3, err := NewErrorWithData_any_params(nil, "XOwn:%{X},X:${X}", "X", 1)
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError_any(t, error3, "XOwn:1,X:1", ParamMap{"X": 1}, nil)

	error4, err := NewErrorWithData_any_params(error3, "", "X", 2, "Y", 3)
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError_any(t, error4, "XOwn:1,X:2", ParamMap{"X": 2, "Y": 3}, []error{error3})

	error5, err := NewErrorWithData_any_params(nil, "X:%{X}Z:%{Z}", "X", 1, "Y", 2, ErrorUnlessValidBase) // validation fails
	testutils.FatalUnless(t, error5.ValidateError_Base() != nil, "")
	testutils.FatalUnless(t, err != nil, "")
	testError_any(t, error5, "ignore", ParamMap{"X": 1, "Y": 2}, nil)

	error6, err := NewErrorWithData_any_params(nil, "X:%{X}Z:${Z}", "X", 1, "Y", 2, ErrorUnlessValidBase) // validation succeeds (but not for Final)
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError_any(t, error6, "ignore", ParamMap{"X": 1, "Y": 2}, nil)
	testutils.FatalUnless(t, error6.ValidateError_Final() != nil, "")

	error7, err := NewErrorWithData_any_params(error6, "", "X", 3, "Z", 4)
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError_any(t, error7, "X:1Z:4", ParamMap{"X": 3, "Y": 2, "Z": 4}, []error{error7})

	error8, err := NewErrorWithData_any_params(error7, "$w${U}", "X", 10, "Z", 4, MistakeIfDataIsReplaced, ErrorUnlessValidFinal) // 2 types of error
	testutils.FatalUnless(t, err != nil, "")
	// fmt.Println(err) -- looks good
	testError_any(t, error8, "ignore", ParamMap{"X": 10, "Y": 2, "Z": 4}, []error{error7}) // Note: X gets replaced.

	error9, err := NewErrorWithData_any_params(error7, "$w${U}", "X", 10, "Z", 4, MistakeIfDataIsReplaced) // 1 types of error
	testutils.FatalUnless(t, err != nil, "")
	// fmt.Println(err) -- looks good
	testError_any(t, error9, "ignore", ParamMap{"X": 10, "Y": 2, "Z": 4}, []error{error7}) // Note: X gets replaced.

	didPanic, panicValue := testutils.CheckPanic2(func() {
		NewErrorWithData_any_params(error7, "$w${U}", "X", 10, "Z", 4, MistakeIfDataIsReplaced, PanicOnAllMistakes)
	}) // same as above})
	testutils.FatalUnless(t, didPanic == true, "")
	testutils.FatalUnless(t, panicValue.(error).Error() == err.Error(), "")

	// fmt.Println(panicVal)
	// _ = panicVal
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

	error9, err := NewErrorWithData_map[DataXZ](error7, "$w${U}", ParamMap{"X": 3, "Z": 4}, MistakeIfDataIsReplaced, ErrorUnlessValidFinal) // 2 types of error
	testutils.FatalUnless(t, err != nil, "")
	// fmt.Println(err) -- looks good
	testError(t, error9, "ignore", &DataXZ{X: 3, Z: 4}, ParamMap{"X": 3, "Y": 2, "Z": 4}, []error{error7}) // Note: X gets replaced.

	didPanic, panicVal = testutils.CheckPanic2(func() {
		NewErrorWithData_map[DataXZ](error7, "$w${U}", ParamMap{"X": 3, "Z": 4}, MistakeIfDataIsReplaced, ErrorUnlessValidFinal, PanicOnAllMistakes) // same as above
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

func TestNewErrorWithDataAnyMap(t *testing.T) {

	didPanic, _ := testutils.CheckPanic2(func() { NewErrorWithData_any_map(nil, "foo", nil, nil) })
	testutils.FatalUnless(t, didPanic == true, "")
	didPanic, _ = testutils.CheckPanic2(func() { NewErrorWithData_any_map(nil, "", nil) }) // empty error without AllowEmptyString
	testutils.FatalUnless(t, didPanic == true, "")
	var error0 ErrorWithData_any
	didPanic, panicVal := testutils.CheckPanic2(func() { error0, _ = NewErrorWithData_any_map(nil, "foo", nil, nil...) })
	testutils.FatalUnless(t, didPanic == false, "%v", panicVal)
	testError_any(t, error0, "foo", nil, nil)

	error1, err := NewErrorWithData_any_map(nil, "Error1", ParamMap{"X": 5})
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError_any(t, error1, "Error1", ParamMap{"X": 5}, nil)

	error2, err := NewErrorWithData_any_map(nil, "", nil, AllowEmptyString)
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError_any(t, error2, "", ParamMap{}, nil)

	error3, err := NewErrorWithData_any_map(nil, "XOwn:%{X},X:${X}", ParamMap{"X": 1})
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError_any(t, error3, "XOwn:1,X:1", ParamMap{"X": 1}, nil)

	error4, err := NewErrorWithData_any_map(error3, "", ParamMap{"X": 2, "Y": 3})
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError_any(t, error4, "XOwn:1,X:2", ParamMap{"X": 2, "Y": 3}, []error{error3})

	error5, err := NewErrorWithData_any_map(error3, "", ParamMap{"X": 2, "Y": 3}, PreferPreviousData)
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError_any(t, error5, "XOwn:1,X:1", ParamMap{"X": 1, "Y": 3}, []error{error3})

	error6, err := NewErrorWithData_any_map(error3, "", ParamMap{"X": 2, "Y": 3}, MistakeIfDataIsReplaced) // causes error, data is replaced nonetheless
	testutils.FatalUnless(t, err != nil, "")
	testError_any(t, error6, "XOwn:1,X:2", ParamMap{"X": 2, "Y": 3}, []error{error3})

	error7, err := NewErrorWithData_any_map(error3, "%{Z}", ParamMap{"X": 2, "Y": 3}, MistakeIfDataIsReplaced, ErrorUnlessValidFinal) // validation and creation error
	testutils.FatalUnless(t, err != nil, "")
	testError_any(t, error7, "ignore", ParamMap{"X": 2, "Y": 3}, []error{error3})

	error8, err := NewErrorWithData_any_map(nil, "%{", ParamMap{"X": nil}) // invalid syntax
	testutils.FatalUnless(t, err != nil, "")
	testError_any(t, error8, "ignore", ParamMap{"X": nil}, nil)

	didPanic, panicVal = testutils.CheckPanic2(func() { NewErrorWithData_any_map(nil, "%{", ParamMap{"X": nil}, PanicOnAllMistakes) })
	testutils.FatalUnless(t, didPanic == true, "")
	testutils.FatalUnless(t, panicVal.(error).Error() == err.Error(), "")

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
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError_any(t, error2, "err", ParamMap{}, nil)

	errBase, _ := NewErrorWithData_struct(nil, "${X} %{X}", &DataX{X: 5}, ErrorUnlessValidFinal, PanicOnAllMistakes)
	testError(t, errBase, "5 5", &DataX{X: 5}, ParamMap{"X": uint(5)}, nil)

	error3, err := DeleteParameterFromError_any(errBase, "", "X")
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError_any(t, error3, "$v!<missing value> 5", ParamMap{}, []error{errBase})

	error4, err := DeleteParameterFromError_any(errBase, "", "X", "Y", ErrorUnlessValidFinal) // same as error3, but notice there is an error
	testutils.FatalUnless(t, err != nil, "")
	testError_any(t, error4, "$v!<missing value> 5", ParamMap{}, []error{errBase})
	// fmt.Println(err)

	error5, err := DeleteParameterFromError_any(errBase, "", "X", "Y", ErrorUnlessValidBase) // same as error3,error4, but we expect no err
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError_any(t, error5, "$v!<missing value> 5", ParamMap{}, []error{errBase})

	errBase2, err := NewErrorWithData_struct(errBase, "${X} ${Y}", &DataXY{X: 1, Y: 2})
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError(t, errBase2, "1 2", &DataXY{X: 1, Y: 2}, ParamMap{"X": 1, "Y": 2}, []error{errBase})

	error6, err := DeleteParameterFromError_any(errBase2, "${Y}", "X", ErrorUnlessValidFinal, PanicOnAllMistakes)
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError_any(t, error6, "2", ParamMap{"Y": 2}, []error{errBase2})

	_ = panicVal
}

func TestDeleteParamterFromError(t *testing.T) {
	type invalid struct{ _ int } // unexported
	type DataXY struct{ X, Y int }
	type DataXZ struct{ X, Z int }
	type DataX struct{ X uint } // note different type
	type empty struct{}

	error1, err := DeleteParameterFromError[empty](nil, "")
	testutils.FatalUnless(t, error1 == nil, "%v", error1)
	testutils.FatalUnless(t, err == nil, "%v", err)

	didPanic, panicVal := testutils.CheckPanic2(func() { DeleteParameterFromError[empty](nil, "", AllowEmptyString) })
	testutils.FatalUnless(t, didPanic == true, "")
	didPanic, panicVal = testutils.CheckPanic2(func() { DeleteParameterFromError[empty](nil, "foo", 5) })
	testutils.FatalUnless(t, didPanic == true, "")
	didPanic, panicVal = testutils.CheckPanic2(func() { DeleteParameterFromError[empty](nil, "foo", "param", nil) })
	testutils.FatalUnless(t, didPanic == true, "")
	didPanic, panicVal = testutils.CheckPanic2(func() { DeleteParameterFromError[empty](nil, "foo", "param", PreferPreviousData) }) // invalid flag for this function
	testutils.FatalUnless(t, didPanic == true, "")
	didPanic, panicVal = testutils.CheckPanic2(func() { DeleteParameterFromError[invalid](nil, "foo") }) // invalid flag for this function
	testutils.FatalUnless(t, didPanic == true, "")

	error2, err := DeleteParameterFromError[empty](nil, "err", nil...)
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError(t, error2, "err", &empty{}, ParamMap{}, nil)

	error3, err := DeleteParameterFromError[DataX](nil, "err", MissingDataAsZero)
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError(t, error3, "err", &DataX{X: uint(0)}, ParamMap{"X": uint(0)}, nil)

	error4, err := DeleteParameterFromError[DataX](nil, "err")
	testutils.FatalUnless(t, err != nil, "")
	testError(t, error4, "err", &DataX{X: uint(0)}, ParamMap{"X": uint(0)}, nil)

	errBase, _ := NewErrorWithData_struct(nil, "${X} %{X}", &DataX{X: 5}, ErrorUnlessValidFinal, PanicOnAllMistakes)
	testError(t, errBase, "5 5", &DataX{X: 5}, ParamMap{"X": uint(5)}, nil)

	errBase2, err := NewErrorWithData_struct(errBase, "${X} ${Y}", &DataXY{X: 1, Y: 2})
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError(t, errBase2, "1 2", &DataXY{X: 1, Y: 2}, ParamMap{"X": 1, "Y": 2}, []error{errBase})

	error5, err := DeleteParameterFromError[empty](errBase, "", "X")
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError(t, error5, "$v!<missing value> 5", &empty{}, ParamMap{}, []error{errBase})

	error6, err := DeleteParameterFromError[DataXZ](errBase2, "", "X", "Z", MissingDataAsZero, ErrorUnlessValidFinal)
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError(t, error6, "0 2", &DataXZ{X: 0, Z: 0}, ParamMap{"X": 0, "Y": 2, "Z": 0}, []error{errBase2})

	error7, err := DeleteParameterFromError[DataXZ](errBase2, "$w", "Y", MissingDataIsMistake, ErrorUnlessValidFinal) // 2 types of error: Z missing for DataXZ, ${Y} missing
	testutils.FatalUnless(t, err != nil, "%v", err)
	testError(t, error7, "1 $v!<missing value>", &DataXZ{X: 1, Z: 0}, ParamMap{"X": 1, "Z": 0}, []error{errBase2})

	// same, but panic this time
	didPanic, panicVal = testutils.CheckPanic2(func() {
		DeleteParameterFromError[DataXZ](errBase2, "$w", "Y", MissingDataIsMistake, ErrorUnlessValidFinal, PanicOnAllMistakes)
	})
	testutils.FatalUnless(t, didPanic == true, "")
	testutils.FatalUnless(t, panicVal.(error).Error() == err.Error(), "%v %v", panicVal, err)

	// _ = panicVal
}

func TestAsErrorWithData(t *testing.T) {
	type invalid struct{ _ int } // unexported
	type DataXY struct{ X, Y int }
	type DataXZ struct{ X, Z int }
	type DataX struct{ X uint } // note different type
	type empty struct{}

	baseError1, _ := NewErrorWithData_params[DataXY](nil, "${X} ${Z}", "X", 1, "Y", 2, "Z", 3, PanicOnAllMistakes)
	baseError2, _ := NewErrorWithData_params[DataXY](nil, "${X} ${Z}", "X", 1, "Y", 2, ErrorUnlessValidBase, PanicOnAllMistakes)

	didPanic, panicVal := testutils.CheckPanic2(func() { AsErrorWithData[invalid](nil) })
	testutils.FatalUnless(t, didPanic == true, "")

	didPanic, panicVal = testutils.CheckPanic2(func() { AsErrorWithData[invalid](baseError1) })
	testutils.FatalUnless(t, didPanic == true, "")

	error1, err := AsErrorWithData[DataXZ](nil)
	testutils.FatalUnless(t, error1 == nil, "%v", error1)
	testutils.FatalUnless(t, err == nil, "%v", err)

	error2, err := AsErrorWithData[empty](baseError1)
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError(t, error2, "1 3", &empty{}, ParamMap{"X": 1, "Y": 2, "Z": 3}, []error{baseError1})

	error3, err := AsErrorWithData[DataXZ](error2)
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError(t, error3, "1 3", &DataXZ{X: 1, Z: 3}, ParamMap{"X": 1, "Y": 2, "Z": 3}, []error{baseError1, error2})

	error4, err := AsErrorWithData[empty](baseError2)
	testutils.FatalUnless(t, err == nil, "%v", err)
	testutils.FatalUnless(t, error4.ValidateError_Final() != nil, "")

	error5, err := AsErrorWithData[DataXZ](baseError2) // Adds Z == 0, but considered an error
	testutils.FatalUnless(t, err != nil, "")
	testError(t, error5, "1 0", &DataXZ{X: 1, Z: 0}, ParamMap{"X": 1, "Y": 2, "Z": 0}, []error{baseError2})

	error6, err := AsErrorWithData[DataXZ](baseError2, MissingDataAsZero) // Adds Z == 0, but NOT considered an error
	testutils.FatalUnless(t, err == nil, "%v", err)
	testError(t, error6, "1 0", &DataXZ{X: 1, Z: 0}, ParamMap{"X": 1, "Y": 2, "Z": 0}, []error{baseError2})

	error7, err := AsErrorWithData[DataX](baseError1, MissingDataAsZero) // Changes X to uint(0), because of int != uint type mismatch
	testutils.FatalUnless(t, err != nil, "%v", err)
	testError(t, error7, "0 3", &DataX{X: 0}, ParamMap{"X": uint(0), "Y": 2, "Z": 3}, []error{baseError1})

	didPanic, panicVal = testutils.CheckPanic2(func() { AsErrorWithData[DataX](baseError1, MissingDataAsZero, PanicOnAllMistakes) })
	testutils.FatalUnless(t, didPanic == true, "")
	testutils.FatalUnless(t, panicVal.(error).Error() == err.Error(), "")

	_ = panicVal

}
