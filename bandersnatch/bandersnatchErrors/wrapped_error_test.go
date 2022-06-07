package bandersnatchErrors

import (
	"errors"
	"io"
	"testing"
)

var _ error = NewWrappedError(io.EOF, "some string")

func TestWrappedError(t *testing.T) {
	err1 := NewWrappedError(io.EOF, "test1")
	err2 := NewWrappedError(err1, "test2")
	if !errors.Is(err1, io.EOF) {
		t.Fatal("WrappedError does not wrap intended error")
	}
	if !errors.Is(err2, io.EOF) {
		t.Fatal("WrappedError does not wrap nestedly")
	}
	if !errors.Is(err2, err1) {
		t.Fatal("WrappedError does not wrap already wrapped error")
	}
	if errors.Is(err1, err2) {
		t.Fatal("WrappedError wraps the wrong way around")
	}
	if errors.Is(err1, io.ErrUnexpectedEOF) {
		t.Fatal("WrappedError wraps unintended error")
	}
	if errors.Is(err2, io.ErrUnexpectedEOF) {
		t.Fatal("WrappedError wraps unintended error")
	}
	if err1.Error() != "test1" {
		t.Fatal("WrappedError does not report intended string")
	}
	if err2.Error() != "test2" {
		t.Fatal("WrappedError does not report indented string")
	}
}

/*
var _ error = NewErrorWithData(io.EOF, "some string", int32(5))


func TestErrorWithData(t *testing.T) {
	err := NewErrorWithData(io.EOF, "t", int64(6))
	if err.Data != int64(6) {
		t.Fatal("ErrorWithData does not contain intended data")
	}
	err1 := NewErrorWithData(io.EOF, "dce1", int64(5))
	err2 := NewErrorWithData(err1, "dce2", int8(6))
	if !errors.Is(err1, io.EOF) {
		t.Fatal("ErrorWithData does not wrap intended error")
	}
	if !errors.Is(err2, io.EOF) {
		t.Fatal("ErrorWithData does not wrap nestedly")
	}
	if !errors.Is(err2, err1) {
		t.Fatal("ErrorWithData does not wrap already wrapped error")
	}
	if errors.Is(err1, err2) {
		t.Fatal("ErrorWithData wraps the wrong way around")
	}
	if errors.Is(err1, io.ErrUnexpectedEOF) {
		t.Fatal("ErrorWithData wraps unintended error")
	}
	if errors.Is(err2, io.ErrUnexpectedEOF) {
		t.Fatal("ErrorWithData wraps unintended error")
	}
	if err1.Error() != "dce1" {
		t.Fatal("ErrorWithData does not report intended string")
	}
	if err2.Error() != "dce2" {
		t.Fatal("ErrorWithData does not report indented string")
	}
	unwrap := err2.Unwrap().(*ErrorWithData[int64])
	if unwrap.Data != int64(5) {
		t.Fatal("Could not retrieve wrapped contained data")
	}
}

*/
