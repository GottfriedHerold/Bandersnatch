package bandersnatchErrors

import (
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/errorsWithData"
)

func TestUnexpectEOF(t *testing.T) {
	err1 := io.EOF
	errBlah := errors.New("blah")
	err2 := errBlah
	UnexpectEOF(&err1)
	if errors.Is(err1, io.EOF) {
		t.Fatalf("E1-1")
	}
	if !errors.Is(err1, io.ErrUnexpectedEOF) {
		t.Fatalf("E1-2")
	}
	UnexpectEOF(&err2)
	if err2 != errBlah {
		t.Fatalf("E2")
	}

	err3 := fmt.Errorf("wrapping %w", io.EOF)
	err3 = errorsWithData.IncludeParametersInError(err3, "Param1", true)
	err3 = errorsWithData.IncludeParametersInError(err3, "Param2", 5)
	UnexpectEOF(&err3)
	if errors.Is(err3, io.EOF) {
		t.Fatalf("E3")
	}
	if !errors.Is(err3, io.ErrUnexpectedEOF) {
		t.Fatalf("E4")
	}
	m := errorsWithData.GetAllParametersFromError(err3)
	if m["Param1"] != true || m["Param2"] != 5 {
		t.Fatalf("E5")
	}
}

func TestUnexpectEOF2(t *testing.T) {
	type dataType = struct{ X int } // type alias. Note the "="
	err1 := errorsWithData.NewErrorWithGuaranteedParameters[dataType](io.EOF, "", "X", 4)
	errBlah := errorsWithData.NewErrorWithGuaranteedParameters[struct{ X int }](nil, "blah", "X", 5)
	err2 := errBlah
	UnexpectEOF2(&err1)
	if errors.Is(err1, io.EOF) {
		t.Fatalf("E1-1")
	}
	if !errors.Is(err1, io.ErrUnexpectedEOF) {
		t.Fatalf("E1-2")
	}
	UnexpectEOF2(&err2)
	if err2 != errBlah {
		t.Fatalf("E2")
	}

	err3 := errorsWithData.NewErrorWithGuaranteedParameters[dataType](io.EOF, "wrapping %w", "X", 3)
	err3 = errorsWithData.IncludeGuaranteedParametersInError[dataType](err3, "Param1", true)
	err3 = errorsWithData.IncludeGuaranteedParametersInError[dataType](err3, "Param2", 5)
	UnexpectEOF2(&err3)
	if errors.Is(err3, io.EOF) {
		t.Fatalf("E3")
	}
	if !errors.Is(err3, io.ErrUnexpectedEOF) {
		t.Fatalf("E4")
	}
	m := errorsWithData.GetAllParametersFromError(err3)
	if m["Param1"] != true || m["Param2"] != 5 {
		t.Fatalf("E5")
	}
}
