package errorsWithData

import (
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
)

var _ errorWithParameters_commonInterface = &errorWithParameters_common{}

func TestErrorWithParameters(t *testing.T) {
	var nilError error = nil
	err1 := fmt.Errorf("error1")
	err2 := fmt.Errorf("error wrapping %w", err1)
	errEOF := io.EOF
	wrappedEOF := fmt.Errorf("error wrapping EOF %w", errEOF)

	for _, err := range []error{nilError, err1, err2, errEOF, wrappedEOF} {
		if HasParameter(err, "x") || HasParameter(err, "") {
			t.Fatalf("HasParameters returns true for plain error")
		}
		val, present := GetParameterFromError(err, "x")
		if val != nil {
			t.Fatalf("GetDataFromError gives non-nil for plain error")
		}
		if present {
			t.Fatalf("GetDataFromError gives true for plain error")
		}
	}

	nilModified := IncludeParametersInErrorUnconstrained(nilError, "foo", true)
	nilModified2 := IncludeParametersInError[struct{ Bar bool }](nilError, "Foo", true) // mismatch, but nil stays nil
	nilModified3 := IncludeParametersInErrorUnconstrained(nilError)
	if nilModified != nil {
		t.Fatalf("E1-1")
	}
	if nilModified2 != nil {
		t.Fatalf("E2-2")
	}
	if nilModified3 != nil {
		t.Fatalf("E1-3")
	}

	nilWithFoo := NewErrorWithParametersUnconstrained(nilError, "message", "foo", true)
	if nilWithFoo == nil {
		t.Fatalf("E2-1")
	}

	nilWithFoo2 := NewErrorWithParameters[struct{ Foo bool }](nilError, "message", "Foo", true)
	if nilWithFoo2 == nil {
		t.Fatalf("E2-2")
	}

	// need to create anonymous function, because CheckPanic has issues with the pair (variadic functions, nil arguments).
	if !testutils.CheckPanic(func() { NewErrorWithParametersUnconstrained(nilError, "", "foo", "true") }) {
		t.Fatalf("E2-3")
	}

	if !testutils.CheckPanic(func() { NewErrorWithParametersUnconstrained(nilError, "message", "foo") }) {
		t.Fatalf("E2-4")
	}

	if !testutils.CheckPanic(func() { NewErrorWithParametersUnconstrained(err1, "message", true, "foo") }) {
		t.Fatalf("E2-5")
	}

	if !testutils.CheckPanic(func() { NewErrorWithParameters[struct{ Foo1 bool }](err1, "", "Foo2", false) }) {
		t.Fatalf("E3-6")
	}

	stillNil := NewErrorWithParametersUnconstrained(nilError, "")
	if stillNil != nil {
		t.Fatalf("E3")
	}

	wrappedEOFWithData1 := IncludeParametersInErrorUnconstrained(wrappedEOF, "Data1", 5, "Data2", 6)
	wrappedEOFWithData12 := fmt.Errorf("Wrapping %w", IncludeParametersInErrorUnconstrained(wrappedEOFWithData1, "Data2", "arg2"))

	m := GetAllParametersFromError(wrappedEOFWithData12)
	if m["Data1"] != 5 || m["Data2"] != "arg2" {
		t.Fatalf("E4-1")
	}
	if !errors.Is(wrappedEOFWithData12, io.EOF) {
		t.Fatalf("E4-2")
	}
	if len(m) != 2 {
		t.Fatalf("E4-3")
	}

	val, ok := GetParameterFromError(wrappedEOFWithData12, "Data1")
	if !ok || val != 5 {
		t.Fatalf("E4-4")
	}
	val, ok = GetParameterFromError(wrappedEOFWithData12, "Data2")
	if !ok || val != "arg2" {
		t.Fatalf("E4-5")
	}

	type S struct {
		Data1 int
		Data2 string
	}

	s := GetDataFromError[S](wrappedEOFWithData12)
	if s.Data1 != 5 || s.Data2 != "arg2" {
		t.Fatalf("E4-6")
	}

	sPart := GetDataFromError[struct{ Data1 int }](wrappedEOFWithData12)
	if sPart.Data1 != 5 {
		t.Fatalf("E4-7")
	}

	if !testutils.CheckPanic(func() { GetDataFromError[struct{ Foo bool }](wrappedEOFWithData12) }) {
		t.Fatalf("E4-8")
	}

	errorWithData2Deleted := DeleteParameterFromError(wrappedEOFWithData12, "Data2")
	m = GetAllParametersFromError(errorWithData2Deleted)
	if len(m) != 1 {
		t.Fatalf("E5-1")
	}
	val, ok = GetParameterFromError(errorWithData2Deleted, "Data2")
	if ok || val != nil {
		t.Fatalf("E5-2")
	}

	if HasParameter(errorWithData2Deleted, "Data2") {
		t.Fatalf("E5-3")
	}

	wrappedEOFWithData34 := NewErrorWithParametersAsData(errEOF, "msg", &struct {
		Data3 bool
		Data4 int
	}{true, 5})

	type struct35 struct {
		Data3 error
		Data5 string
	}

	wrapper := fmt.Errorf("%w", wrappedEOFWithData34)
	wrappedEOFWithData345 := IncludeDataInError(wrapper, &struct35{nil, "foo"})

	var data35 struct35 = wrappedEOFWithData345.GetData()
	if data35.Data3 != nil || data35.Data5 != "foo" {
		t.Fatalf("E6")
	}

	m = GetAllParametersFromError(wrappedEOFWithData345)
	if len(m) != 3 {
		t.Fatalf("E7-1")
	}
	if m["Data3"] != nil || m["Data4"] != 5 || m["Data5"] != "foo" {
		t.Fatalf("E7-2")
	}

	type struct34 struct {
		Data3 error
		Data4 int
	}

	if HasData[struct34](wrappedEOFWithData34) { // the error has some Data3 and Data4, but Data3 has wrong type
		t.Fatalf("E8-1")
	}

	if !HasData[struct34](wrappedEOFWithData345) {
		t.Fatalf("E8-2")
	}

	if !HasData[struct34](fmt.Errorf("as%w", wrappedEOFWithData345)) {
		t.Fatalf("E8-3")
	}

	/*
		old2, present2 := IncludeParametersInError(&wrappedEOF, "data1", 5)
		if old2 != nil || present2 {
			t.Fatalf("AddDataToError on plain error returned previous value")
		}
		if !errors.Is(wrappedEOF, errEOF) || !errors.Is(wrappedEOF, io.EOF) {
			t.Fatalf("AddDataToError does not preserve error wrapping")
		}
		if _, ok := wrappedEOF.(ErrorWithParameters); !ok {
			t.Fatalf("AddDataToError did not turn error into BandersnatchError")
		}
		err5 := fmt.Errorf("Wrapping error4 %w", wrappedEOF)
		if !HasData(wrappedEOF, "data1") {
			t.Fatalf("HasData did not respect AddDataToError")
		}
		if !HasData(err5, "data1") {
			t.Fatalf("HasData did not respect AddDataToError (wrapped)")
		}
		got, present := GetParameterFromError(wrappedEOF, "data1")
		if got != 5 || !present {
			t.Fatalf("GetDataFromError did not return added Data")
		}
		got, present = GetParameterFromError(err5, "data1")
		if got != 5 || !present {
			t.Fatalf("GetDataFromError did not return added Data (wrapped)")
		}
		got, present = IncludeParametersInError(&err5, "data1", 6)
		if got != 5 || !present {
			t.Fatalf("AddDataToError did not return previous values (from chained error)")
		}
		got, present = GetParameterFromError(err5, "data1")
		if got != 6 || !present {
			t.Fatalf("GetDataFromError did not respect override")
		}
		got, present = IncludeParametersInError(&err5, "data2", nil)
		if got != nil || present {
			t.Fatalf("AddDataToError did return unexpected values")
		}
		got, present = IncludeParametersInError(&err5, "data1", nil)
		if got != 6 || !present {
			t.Fatalf("AddDataToError returned unexpected result")
		}
		got, present = GetParameterFromError(err5, "data1")
		if got != nil || !present {
			t.Fatalf("AddDataToError returned non (nil, true) after setting value to nil")
		}

		// This behaviour is not really guaranteed by the documentation,
		// but what the current implementation supposedly does.
		// It relies on AddDataToError not wrapping if the error is already
		// a BandersnatchError; this test is intended to verify this works.
		err5.(ErrorWithParameters).DeleteData("data1")
		got, present = GetParameterFromError(err5, "data1")
		if got != 5 || !present {
			t.Fatalf("Deleting data did not restore previous value")
		}

	*/

}

/*

			func TestErrorWithParams(t *testing.T) {
				err1 := fmt.Errorf("error1")
				err2 := NewErrorWithParameters(err1, "")
				if err2.Error() != "error1" {
					t.Fatalf("Error message not kept by NewErrorWithParams")
				}
				err3 := NewErrorWithParameters(err2, "error2")
				if err3.Error() != "error2" {
					t.Fatalf("Error message not overridden by NewErrorWithParams")
				}
				if !errors.Is(err3, err2) {
					t.Fatalf("Errors not wrapping as expected")
				}
				if !errors.Is(err3, err1) {
					t.Fatalf("Errors not wrapping recursively as expected")
				}
				if !errors.Is(err2, err1) {
					t.Fatalf("Errors not wrapping as expected (2)")
				}
				if errors.Is(err3, io.EOF) {
					t.Fatalf("Errors wrapping unrelated error")
				}
				if errors.Is(err1, err3) || errors.Is(err1, err2) || errors.Is(err2, err3) {
					t.Fatalf("Errors wrapping the wrong way around")
				}
				errD1 := NewErrorWithParameters(err1, "", "Data1", 5, "DatA2", true)
				if !errD1.HasData("Data1") {
					t.Fatalf("Data1 not recorded")
				}
				if !errD1.HasData("dAtA1") {
					t.Fatalf("Data1 not case-insensitive")
				}
				if !errD1.HasData("data2") {
					t.Fatalf("data2 not recorded")
				}
				if errD1.HasData("") {
					t.Fatalf("errD1 has data for empty string")
				}
				if errD1.HasData("data") {
					t.Fatalf("errD1 has data for unrelated string")
				}
				D1, ok1 := errD1.GetData("data1")
				D2, ok2 := errD1.GetData("data2")
				D3, ok3 := errD1.GetData("data3")
				if !ok1 || !ok2 || ok3 {
					t.Fatalf("GetData not as expected")
				}
				if D1.(int) != 5 {
					t.Fatalf("Could not recover D1")
				}
				if D2.(bool) != true {
					t.Fatalf("Could not recover D2")
				}
				if D3 != nil {
					t.Fatalf("Got non-nil for invalid entry")
				}
				old1, ok1 := errD1.AddData("data1", "X")
				old2, ok2 := errD1.AddData("data3", "Y")
				if !ok1 || ok2 {
					t.Fatalf("Got wrong alreadyPresent for AddData")
				}
				if old1 != 5 {
					t.Fatalf("Got wrong oldValue for data1 from AddData")
				}
				if old2 != nil {
					t.Fatalf("Got wrong oldValue for data3 from AddData")
				}
				D1, ok1 = errD1.GetData("data1")
				D3, ok3 = errD1.GetData("data3")
				if !ok1 || !ok3 {
					t.Fatalf("Data not present after AddData")
				}
				if D1 != "X" || D3 != "Y" {
					t.Fatalf("Data not recovered after AddData")
				}
				old1, ok1 = errD1.DeleteData("data1")
				old4, ok4 := errD1.DeleteData("data4")
				if !ok1 {
					t.Fatalf("Got wrong alreadyPresent from DeleteData for present value")
				}
				if ok4 {
					t.Fatalf("Got wrong alreadyPresent from DeleteData for non-present value")
				}
				if old4 != nil {
					t.Fatalf("Got wrong value from DeleteData for non-present value")
				}
				if old1 != "X" {
					t.Fatalf("Got wrong value from DeleteData for present value")
				}
				D1, ok1 = errD1.GetData("data1")
				if ok1 || D1 != nil {
					t.Fatalf("Value not deleted by DeleteData")
				}
				panic1 := testutils.CheckPanic(NewErrorWithParameters, err1, "", "blah")
				if !panic1 {
					t.Fatalf("NewErrosWithParams did not panic with wrong number of arguments")
				}
				panic2 := testutils.CheckPanic(NewErrorWithParameters, err1, "", 5, 5)
				if !panic2 {
					t.Fatalf("NewErrorsWithParams did not panic with malformed arguments")
				}
				var e *errorWithParams = nil
				var _ ErrorWithParameters = e
				// Checks that this does not panic
				if e.HasData("") {
					t.Fatalf("errorWithParams.HasData panics on nil-receiver")
				}
				if !testutils.CheckPanic(func() { e.GetData("") }) {
					t.Fatalf("errorWithParams.GetData does not panic on nil receivers")
				}
				if !testutils.CheckPanic(func() { e.AddData("param1", 0) }) {
					t.Fatalf("errorWithParams.AddData does not panic on nil receivers")
				}
				if !testutils.CheckPanic(func() { e.DeleteData("param1") }) {
					t.Fatalf("errorWithParams.DeleteData does not panic on nil receivers")
				}
				if NewErrorWithParameters(nil, "") != nil {
					t.Fatalf("NewErrorWithParams(nil,\"\") did not return nil")
				}
				if !testutils.CheckPanic(func() { NewErrorWithParameters(nil, "", "arg", 5) }) {
					t.Fatalf("NewErrorWithParams(nil,\"\", args) did not panic")
				}
				err4 := NewErrorWithParameters(nil, "error message")
				err5 := NewErrorWithParameters(nil, "error message2", "data1", 5)
				if err4.Error() != "error message" {
					t.Fatalf("NewErrorWithParams(nil,...) does not create correct error message")
				}
				OLD := SetShowEmbeddedDataOnError(false)
				if err5.Error() != "error message2" {
					t.Fatalf("NewErroWithParams(nil,...) with data does not create correct error message. Got %v", err5.Error())
				}
				SetShowEmbeddedDataOnError(OLD)
				if errors.Unwrap(err4) != nil {
					t.Fatalf("NewErrorWithParams(nil) creates non-nil-wrapping Error")
				}
				if errors.Unwrap(err5) != nil {
					t.Fatalf("NewErrorWithParams(nil, message, extra data) does not wrap non-nil")

}
*/
