package bandersnatchErrors

import (
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
)

var _ BandersnatchError = &errorWithParams{}

func TestBandersnatchError(t *testing.T) {
	var nilError error = nil
	err1 := fmt.Errorf("error1")
	err2 := fmt.Errorf("error wrapping %w", err1)
	err3 := io.EOF
	err4 := fmt.Errorf("error wrapping EOF %w", err3)
	for _, err := range []error{nilError, err1, err2, err3, err4} {
		if HasData(err, "x") || HasData(err, "") {
			t.Fatalf("HasData returns true for plain error")
		}
		val, present := GetDataFromError(err, "x")
		if val != nil {
			t.Fatalf("GetDataFromError gives non-nil for plain error")
		}
		if present {
			t.Fatalf("GetDataFromError gives true for plain error")
		}
	}
	old1, present1 := AddDataToError(&nilError, "foo", true)
	if old1 != nil || present1 {
		t.Fatalf("unexpected return from AddDataToError(nil,...)")
	}
	if nilError != nil {
		t.Fatalf("AddDataToError(&nilerror,...) modified nil error")
	}

	old2, present2 := AddDataToError(&err4, "data1", 5)
	if old2 != nil || present2 {
		t.Fatalf("AddDataToError on plain error returned previous value")
	}
	if !errors.Is(err4, err3) || !errors.Is(err4, io.EOF) {
		t.Fatalf("AddDataToError does not preserve error wrapping")
	}
	if _, ok := err4.(BandersnatchError); !ok {
		t.Fatalf("AddDataToError did not turn error into BandersnatchError")
	}
	err5 := fmt.Errorf("Wrapping error4 %w", err4)
	if !HasData(err4, "data1") {
		t.Fatalf("HasData did not respect AddDataToError")
	}
	if !HasData(err5, "data1") {
		t.Fatalf("HasData did not respect AddDataToError (wrapped)")
	}
	got, present := GetDataFromError(err4, "data1")
	if got != 5 || !present {
		t.Fatalf("GetDataFromError did not return added Data")
	}
	got, present = GetDataFromError(err5, "data1")
	if got != 5 || !present {
		t.Fatalf("GetDataFromError did not return added Data (wrapped)")
	}
	got, present = AddDataToError(&err5, "data1", 6)
	if got != 5 || !present {
		t.Fatalf("AddDataToError did not return previous values (from chained error)")
	}
	got, present = GetDataFromError(err5, "data1")
	if got != 6 || !present {
		t.Fatalf("GetDataFromError did not respect override")
	}
	got, present = AddDataToError(&err5, "data2", nil)
	if got != nil || present {
		t.Fatalf("AddDataToError did return unexpected values")
	}
	got, present = AddDataToError(&err5, "data1", nil)
	if got != 6 || !present {
		t.Fatalf("AddDataToError returned unexpected result")
	}
	got, present = GetDataFromError(err5, "data1")
	if got != nil || !present {
		t.Fatalf("AddDataToError returned non (nil, true) after setting value to nil")
	}

	// This behaviour is not really guaranteed by the documentation,
	// but what the current implementation supposedly does.
	// It relies on AddDataToError not wrapping if the error is already
	// a BandersnatchError; this test is intended to verify this works.
	err5.(BandersnatchError).DeleteData("data1")
	got, present = GetDataFromError(err5, "data1")
	if got != 5 || !present {
		t.Fatalf("Deleting data did not restore previous value")
	}

}

func TestErrorWithParams(t *testing.T) {
	err1 := fmt.Errorf("error1")
	err2 := NewErrorWithParams(err1, "")
	if err2.Error() != "error1" {
		t.Fatalf("Error message not kept by NewErrorWithParams")
	}
	err3 := NewErrorWithParams(err2, "error2")
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
	errD1 := NewErrorWithParams(err1, "", "Data1", 5, "DatA2", true)
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
	panic1 := testutils.CheckPanic(NewErrorWithParams, err1, "", "blah")
	if !panic1 {
		t.Fatalf("NewErrosWithParams did not panic with wrong number of arguments")
	}
	panic2 := testutils.CheckPanic(NewErrorWithParams, err1, "", 5, 5)
	if !panic2 {
		t.Fatalf("NewErrorsWithParams did not panic with malformed arguments")
	}
	var e *errorWithParams = nil
	var _ BandersnatchError = e
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
	if NewErrorWithParams(nil, "") != nil {
		t.Fatalf("NewErrorWithParams(nil,\"\") did not return nil")
	}
	if !testutils.CheckPanic(func() { NewErrorWithParams(nil, "", "arg", 5) }) {
		t.Fatalf("NewErrorWithParams(nil,\"\", args) did not panic")
	}
	err4 := NewErrorWithParams(nil, "error message")
	err5 := NewErrorWithParams(nil, "error message2", "data1", 5)
	if err4.Error() != "error message" {
		t.Fatalf("NewErrorWithParams(nil,...) does not create correct error message")
	}
	if err5.Error() != "error message2" {
		t.Fatalf("NewErroWithParams(nil,...) with data does not create correct error message")
	}
	if errors.Unwrap(err4) != nil {
		t.Fatalf("NewErrorWithParams(nil) creates non-nil-wrapping Error")
	}
	if errors.Unwrap(err5) != nil {
		t.Fatalf("NewErrorWithParams(nil, message, extra data) does not wrap non-nil")
	}
}
