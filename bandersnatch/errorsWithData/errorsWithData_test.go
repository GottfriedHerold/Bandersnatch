package errorsWithData

import (
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
)

var _ errorWithParameters_commonInterfaceNew = &errorWithParameters_common{}

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

	wrappedEOFWithData34 := NewErrorWithParametersFromData(errEOF, "msg", &struct {
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

}
