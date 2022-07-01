package pointserializer

import (
	"bytes"
	"errors"
	"io"
	"math"
	"runtime/debug"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/bandersnatchErrors"
	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
)

func TestConsumeExpectRead(t *testing.T) {
	var buf bytes.Buffer
	var data []byte = []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	buf.Write(data)
	bytes_read, err := consumeExpectRead(&buf, data[0:4])
	if bytes_read != 4 {
		t.Fatalf("consumeExpectRead did not read expected amount of bytes: Expected %v, got %v", 4, bytes_read)
	}
	if err != nil {
		t.Fatalf("consumeExpectRead returned unexpected error: Error was %v", err)
	}
	if math.MaxInt > math.MaxInt32 {
		var toolargedata = make([]byte, math.MaxInt32+1)
		buf.Reset()
		buf.Write(data)
		if !testutils.CheckPanic(consumeExpectRead, &buf, toolargedata) {
			toolargedata = nil
			debug.FreeOSMemory()
			t.Fatalf("consumeExpectRead on too large data did not fail")
		}
		toolargedata = nil
		debug.FreeOSMemory()
	}
	buf.Reset()
	bytes_read, err = consumeExpectRead(&buf, data[0:4])
	if bytes_read != 0 {
		t.Fatalf("consumeExpectRead on empty reader reported %v > 0 bytes read", bytes_read)
	}
	if !errors.Is(err, io.EOF) {
		t.Fatalf("consumeExpectRead on empty read did not report EOF. Got %v instead", err)
	}
	errData := err.GetData()

	if errData.BytesRead != 0 || len(errData.ActuallyRead) != 0 || errData.PartialRead {
		t.Fatalf("consumderExpectRead on empty read did reported unexpected metadata %v", errData)
	}
	buf.Reset()
	buf.Write(data[0:3])
	bytes_read, err = consumeExpectRead(&buf, data[0:4])
	if bytes_read != 3 {
		t.Fatalf("consumeExpectRead on size-3 reader (asking for 4 bytes) returned 3!= %v bytes read", bytes_read)
	}
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("consumeExpectRead on too short reader did not report io.ErrUnexpectedEOF. Got %v instead", err)
	}

	errData = err.GetData()
	if !bytes.Equal(errData.ActuallyRead, data[0:3]) || !errData.PartialRead {
		t.Fatalf("consumeExpectRead on too short reader reported unexpected metadata. Got %v", errData)
	}

	data2 := []byte{1, 2, 3, 5, 6}
	buf.Reset()
	buf.Write(data)
	bytes_read, err = consumeExpectRead(&buf, data2)
	if bytes_read != 5 {
		t.Fatalf("consumdeExpectRead on mismatched data reported unexpected number %v of bytes_read", bytes_read)
	}
	if !errors.Is(err, bandersnatchErrors.ErrDidNotReadExpectedString) {
		t.Fatalf("consumeExpectRead on mismatched data did not report expected error. Got %v instead", err)
	}

	errData = err.GetData()
	if !bytes.Equal(errData.ActuallyRead, data[0:5]) || !bytes.Equal(errData.ExpectedToRead, data2) {
		t.Fatalf("consumeExpectRead on mismatched data did not return expected metadata. Got %v instead", errData)
	}

	bytes_read, err = consumeExpectRead(nil, []byte{})
	if err != nil {
		t.Fatalf("consumeExpectRead of empty slice failed for nil reader with error %v", err)
	}
	if bytes_read != 0 {
		t.Fatalf("consumeExpectRead of empty slice returned %v > 0 bytes_read", bytes_read)
	}

	// CheckPanic does not work with nil interfaces (due to peculiarities of reflection), so we need nil of a concrete non-interface type.
	if !testutils.CheckPanic(consumeExpectRead, (*bytes.Buffer)(nil), []byte{0}) {
		t.Fatalf("consumeExpectRead on empty reader did not panic")
	}
	if !testutils.CheckPanic(consumeExpectRead, (*bytes.Buffer)(nil), []byte(nil)) {
		t.Fatalf("consumeExpectRead on nil inputs did not panic")
	}
	if !testutils.CheckPanic(consumeExpectRead, &buf, []byte(nil)) {
		t.Fatalf("consumeExpectRead on nil expectToRead did not panic")
	}
}

func TestCopyByteSlice(t *testing.T) {
	var v []byte = nil
	w := copyByteSlice(v)
	if w == nil {
		t.Fatalf("copyByteSlice(nil) == nil")
	}
	if len(w) != 0 {
		t.Fatalf("copyByteSlice(nil) != size-0 slice")
	}
	v = []byte{1, 2, 3, 4}
	w = copyByteSlice(v)
	if w == nil {
		t.Fatalf("copyByteSlice returned nil")
	}
	if testutils.CheckSliceAlias(v, w) {
		t.Fatalf("copyByteSlice returns aliasing result")
	}
	if !bytes.Equal(v, w) {
		t.Fatalf("copyByteSlice did not return an equal copy")
	}
}
