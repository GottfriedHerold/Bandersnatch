package errorsWithData

import (
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
)

var _ ErrorWithData_any = &errorWithParameters_common{}

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
		val, present := GetParameter(err, "x")
		if val != nil {
			t.Fatalf("GetDataFromError gives non-nil for plain error")
		}
		if present {
			t.Fatalf("GetDataFromError gives true for plain error")
		}
	}

	nilModified := IncludeParametersInError(nilError, "foo", true)
	nilModified2 := IncludeGuaranteedParametersInError[struct{ Bar bool }](nilError, "Foo", true) // mismatch, but nil stays nil
	nilModified3 := IncludeParametersInError(nilError)
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

	stillNil := NewErrorWithData_any_params(nilError, "")
	if stillNil != nil {
		t.Fatalf("E3")
	}

	wrappedEOFWithData1 := IncludeParametersInError(wrappedEOF, "Data1", 5, "Data2", 6)
	wrappedEOFWithData12 := fmt.Errorf("Wrapping %w", IncludeParametersInError(wrappedEOFWithData1, "Data2", "arg2"))

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
	wrappedEOFWithData345 := IncludeDataInError(wrapper, &struct35{nil, "foo"})

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
	e3 := IncludeParametersInError(e1)
	if e3.Error() != "ABC" {
		t.Fatalf("Error message output not as expected for empty map")
	}
}
