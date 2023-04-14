package errorsWithData

/*

import (
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
)

var _ ErrorWithData_any = &errorWithParameters_common{}

type data struct{ Data int }
type invalidType struct{ data int }

var error1 error = fmt.Errorf("error1")
var wrappedError1 error = fmt.Errorf("error wrapping %w", error1)
var wrappedEOF = fmt.Errorf("error wrapping EOF %w", io.EOF)

var errorWithData_any_Empty = NewErrorWithData_any_params(nil, "ewd1")
var errorWithData_any_Data1 = NewErrorWithData_any_params(nil, "ewd2", "Data", 1)
var errorWithData_Empty = NewErrorWithData_params[struct{}](nil, "ewd3")
var errorWithData_Data2 = NewErrorWithData_struct[data](nil, "ewd4", &data{Data: 2})
var errorWithData_Data3 = NewErrorWithData_map[data](nil, "ewd5", ParamMap{"Data": 3, "Other": `foo`})

var errorWithData_emptyBase = NewErrorWithData_any_params(nil, "Data=${Data}")
var errorWithData_wrappedBase = AddDataToError_any_params(errorWithData_emptyBase, "Data", 3)
var errorWithData_deletedBase = DeleteParameterFromError(errorWithData_wrappedBase, "Data")

func TestHasParameter(t *testing.T) {
	for _, err := range []error{nil, error1, wrappedError1, wrappedEOF, errorWithData_any_Empty, errorWithData_any_Data1, errorWithData_Data2, errorWithData_Data3, errorWithData_emptyBase, errorWithData_wrappedBase, errorWithData_deletedBase} {
		testutils.FatalUnless(t, !HasParameter(err, "Invalid"), "HasParameter returns true for non-existent parameter")
		testutils.FatalUnless(t, !HasParameter(err, ""), "HasParameter returns true for empty-string parameter")
		value, present := GetParameter(err, "Invalid")
		testutils.FatalUnless(t, value == nil, "GetParameter returns non-nil for non existing parameter")
		testutils.FatalUnless(t, !present, "GetParameter wrongly returns parameter as existent")
	}
	for _, err := range []error{errorWithData_any_Data1, errorWithData_Data2, errorWithData_Data3, errorWithData_wrappedBase} {
		testutils.FatalUnless(t, HasParameter(err, "Data"), "HasParameter does not detect parameter")
	}
	testutils.FatalUnless(t, !HasParameter(errorWithData_emptyBase, "Data"), "")
	testutils.FatalUnless(t, !HasParameter(errorWithData_deletedBase, "Data"), "HasParameter detects deleted parameter")
}

func FullValidate(t *testing.T, err interface {
	ValidateSyntax() error
	ValidateError_Base() error
	ValidateError_Final() error
}) {
	errSyntax := err.ValidateSyntax()
	testutils.FatalUnless(t, errSyntax == nil, "Syntax error: %v", errSyntax)
	errBase := err.ValidateError_Base()
	testutils.FatalUnless(t, errBase == nil, "Base error: %v", errBase)
	errFinal := err.ValidateError_Final()
	testutils.FatalUnless(t, errFinal == nil, "Final error: %v", errBase)
}

func TestNewErrorWithData(t *testing.T) {
	var didPanic bool

	// invalid Struct type causes panic:
	didPanic = testutils.CheckPanic(NewErrorWithData_struct[invalidType], nil, "foo", &invalidType{})
	testutils.FatalUnless(t, didPanic, "")
	didPanic = testutils.CheckPanic(NewErrorWithData_params[invalidType], error1, "foo", "data", 5) // nil -> error1 due to restriction of CheckPanic
	testutils.FatalUnless(t, didPanic, "")
	didPanic = testutils.CheckPanic(NewErrorWithData_map[invalidType], nil, "foo", ParamMap{"data": 5})
	testutils.FatalUnless(t, didPanic, "")

	// failing to provide parameters for data struct causes panic.
	didPanic = testutils.CheckPanic(NewErrorWithData_params[data], error1, "foo", "Stuff", -1)
	testutils.FatalUnless(t, didPanic, "")
	didPanic = testutils.CheckPanic(NewErrorWithData_map[data], error1, "foo", ParamMap{"Stuff": -1})
	testutils.FatalUnless(t, didPanic, "")

	// nil base error and empty interpolation string is not allowed.
	// NOTE: No check for _param variants. This is due to current limitations of CheckPanic
	didPanic = testutils.CheckPanic(NewErrorWithData_struct[data], nil, "", &data{})
	testutils.FatalUnless(t, didPanic, "")
	didPanic = testutils.CheckPanic(NewErrorWithData_map[data], nil, "", ParamMap{"Data": 1})
	testutils.FatalUnless(t, didPanic, "")
	didPanic = testutils.CheckPanic(NewErrorWithData_any_map, nil, "", ParamMap{"Data": 1})
	testutils.FatalUnless(t, didPanic, "")

	err1 := NewErrorWithData_struct(nil, "D%v{Data}", &data{Data: 1})
	testutils.FatalUnless(t, err1.Error() == "D1", "")
	FullValidate(t, err1)
	err2 := NewErrorWithData_params[data](nil, "D%v{Data}", "Data", 2, "Other", 5)
	testutils.FatalUnless(t, err2.Error() == "D2", "")
	FullValidate(t, err2)
	err3 := NewErrorWithData_map[data](nil, "D%v{Data}", ParamMap{"Data": 3, "Other": "foo"})
	testutils.FatalUnless(t, err3.Error() == "D3", "")
	FullValidate(t, err3)

	errBase := NewErrorWithData_any_params(nil, "D${Data}")
	testutils.FatalUnless(t, errBase.ValidateSyntax() == nil, "")
	testutils.FatalUnless(t, errBase.ValidateError_Base() == nil, "")
	testutils.FatalUnless(t, errBase.ValidateError_Final() != nil, "")
	errFinal := NewErrorWithData_any_params(errBase, "", "Data", 4)
	FullValidate(t, errFinal)
	testutils.FatalUnless(t, errFinal.Error() == "D4", "")

}

// some old test; kept around because it still works.

func TestErrorWithParameters(t *testing.T) {
	var nilError error = nil
	err1 := fmt.Errorf("error1")
	err2 := fmt.Errorf("error wrapping %w", err1)
	errEOF := io.EOF
	wrappedEOF := fmt.Errorf("error wrapping EOF %w", errEOF)

	// check that HasParameter works for errors without any parameters
	for _, err := range []error{nilError, err1, err2, errEOF, wrappedEOF} {
		if HasParameter(err, "x") || HasParameter(err, "") {
			t.Fatalf("HasParameters returns true for plain error")
		}
		val, present := GetParameter(err, "x")
		if val != nil {
			t.Fatalf("GetDataFromError gives non-nil for plain error")
		}
		if present {
			t.Fatalf("GetDataFromError gives true for plain error")
		}
	}

	nilModified := AddDataToError_any_params(nilError, "foo", true)
	nilModified2 := AddDataToError_params[struct{ Bar bool }](nilError, "Foo", true) // mismatch, but nil stays nil
	nilModified3 := AddDataToError_any_params(nilError)
	if nilModified != nil {
		t.Fatalf("E1-1")
	}
	if nilModified2 != nil {
		t.Fatalf("E2-2")
	}
	if nilModified3 != nil {
		t.Fatalf("E1-3")
	}

	nilWithFoo := NewErrorWithData_any_params(nilError, "message", "foo", true)
	if nilWithFoo == nil {
		t.Fatalf("E2-1")
	}

	nilWithFoo2 := NewErrorWithData_params[struct{ Foo bool }](nilError, "message", "Foo", true)
	if nilWithFoo2 == nil {
		t.Fatalf("E2-2")
	}

	// need to create anonymous function, because CheckPanic has issues with the pair (variadic functions, nil arguments).
	if !testutils.CheckPanic(func() { NewErrorWithData_any_params(nilError, "", "foo", "true") }) {
		t.Fatalf("E2-3")
	}

	if !testutils.CheckPanic(func() { NewErrorWithData_any_params(nilError, "message", "foo") }) {
		t.Fatalf("NewErrorWithParameters did not panic when called with odd number of variadic arguments.")
	}

	if !testutils.CheckPanic(func() { NewErrorWithData_any_params(err1, "message", true, "foo") }) {
		t.Fatalf("E2-5")
	}

	if !testutils.CheckPanic(func() { NewErrorWithData_params[struct{ Foo1 bool }](err1, "", "Foo2", false) }) {
		t.Fatalf("E3-6")
	}

	wrappedEOFWithData1 := AddDataToError_any_params(wrappedEOF, "Data1", 5, "Data2", 6)
	wrappedEOFWithData12 := fmt.Errorf("Wrapping %w", AddDataToError_any_params(wrappedEOFWithData1, "Data2", "arg2"))

	m := GetData_map(wrappedEOFWithData12)
	if m["Data1"] != 5 || m["Data2"] != "arg2" {
		t.Fatalf("E4-1")
	}
	if !errors.Is(wrappedEOFWithData12, io.EOF) {
		t.Fatalf("E4-2")
	}
	if len(m) != 2 {
		t.Fatalf("E4-3")
	}

	val, ok := GetParameter(wrappedEOFWithData12, "Data1")
	if !ok || val != 5 {
		t.Fatalf("E4-4")
	}
	val, ok = GetParameter(wrappedEOFWithData12, "Data2")
	if !ok || val != "arg2" {
		t.Fatalf("E4-5")
	}

	type S struct {
		Data1 int
		Data2 string
	}

	s := GetData_struct[S](wrappedEOFWithData12)
	if s.Data1 != 5 || s.Data2 != "arg2" {
		t.Fatalf("E4-6")
	}

	sPart := GetData_struct[struct{ Data1 int }](wrappedEOFWithData12)
	if sPart.Data1 != 5 {
		t.Fatalf("E4-7")
	}

	if !testutils.CheckPanic(func() { GetData_struct[struct{ Foo bool }](wrappedEOFWithData12) }) {
		t.Fatalf("E4-8")
	}

	errorWithData2Deleted := DeleteParameterFromError(wrappedEOFWithData12, "Data2")
	m = GetData_map(errorWithData2Deleted)
	if len(m) != 1 {
		t.Fatalf("E5-1")
	}
	val, ok = GetParameter(errorWithData2Deleted, "Data2")
	if ok || val != nil {
		t.Fatalf("E5-2")
	}

	if HasParameter(errorWithData2Deleted, "Data2") {
		t.Fatalf("E5-3")
	}

	wrappedEOFWithData34 := NewErrorWithData_struct(errEOF, "msg", &struct {
		Data3 bool
		Data4 int
	}{true, 5})

	type struct35 struct {
		Data3 error
		Data5 string
	}

	wrapper := fmt.Errorf("%w", wrappedEOFWithData34)
	wrappedEOFWithData345 := AddDataToError_struct(wrapper, &struct35{nil, "foo"})

	var data35 struct35 = wrappedEOFWithData345.GetData_struct()
	if data35.Data3 != nil || data35.Data5 != "foo" {
		t.Fatalf("E6")
	}

	m = GetData_map(wrappedEOFWithData345)
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

	e1 := fmt.Errorf("ABC")
	e2 := NewErrorWithData_any_params(e1, "%w %{Param1}", "Param1", 5)
	if e2.Error() != "ABC 5" {
		t.Fatalf("Error message output not as expected")
	}
	e3 := AddDataToError_any_params(e1)
	if e3.Error() != "ABC" {
		t.Fatalf("Error message output not as expected for empty map. Output was %v", e3.Error())
	}
}

*/
