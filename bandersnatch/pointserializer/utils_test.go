package pointserializer

import (
	"bytes"
	"errors"
	"io"
	"math"
	"runtime/debug"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/bandersnatchErrors"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/errorsWithData"
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

	if !testutils.CheckPanic(consumeExpectRead, nil, []byte{0}) {
		t.Fatalf("consumeExpectRead on empty reader did not panic")
	}
	if !testutils.CheckPanic(consumeExpectRead, nil, nil) {
		t.Fatalf("consumeExpectRead on nil inputs did not panic")
	}
	if !testutils.CheckPanic(consumeExpectRead, &buf, nil) {
		t.Fatalf("consumeExpectRead on nil expectToRead did not panic")
	}

	designatedErr := errors.New("ExpectedError")
	var faultyBuf = testutils.NewFaultyBuffer(2, designatedErr)

	_, errInternal := faultyBuf.Write([]byte{6, 7})
	testutils.Assert(errInternal == nil)

	expectedRead := []byte{6, 5, 8}
	bytes_read, err = consumeExpectRead(faultyBuf, expectedRead)
	testutils.FatalUnless(t, bytes_read == 2, "consumeExpectRead did not read 2 bytes on faulty buffer")
	testutils.FatalUnless(t, errors.Is(err, designatedErr), "consumeExpectRead did not return expected error on faulty buffer read")
	errData = err.GetData()
	testutils.FatalUnless(t, errData.PartialRead == true, "consumeExpectRead did not report partial read")
	testutils.FatalUnless(t, bytes.Equal(errData.ActuallyRead, []byte{6, 7}), "consumeExpectRead did not report actually read data")
	testutils.FatalUnless(t, bytes.Equal(errData.ExpectedToRead, expectedRead), "consumeExpectRead did not report intended read")
	testutils.FatalUnless(t, errData.BytesRead == 2, "consumeExpectRead did not report bytes read in metadata correctly")
	testutils.FatalUnless(t, !testutils.CheckSliceAlias(expectedRead, errData.ExpectedToRead), "error metadata aliases input slice")

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

func TestWriteFull(t *testing.T) {
	var emptyWrite []byte = []byte("")
	var realWrite []byte = []byte("WXYZ")
	var realWriteCopy []byte = []byte("WXYZ")
	testutils.Assert(len(realWrite) == 4)

	// Note: writeFull's behaviour on nil pointers of concrete type satisfying io.Writer with emtpyWrite is unspecified.
	testutils.FatalUnless(t, testutils.CheckPanic(writeFull, nil, emptyWrite), "writeFull did not panic on nil io.Writer")

	var buf bytes.Buffer

	testutils.FatalUnless(t, testutils.CheckPanic(writeFull, &buf, nil), "writeFull did not panic on nil byteSlice")

	buf.Reset()
	bytesWritten, err := writeFull(&buf, emptyWrite)
	testutils.FatalUnless(t, bytesWritten == 0, "writeFull writes more than 0 bytes on empty byte")
	testutils.FatalUnless(t, err == nil, "writeFull failed on empty write with bytes.Buffer with error %v", err)

	testutils.FatalUnless(t, buf.Len() == 0, "writeFull write something on empty byte")

	buf.Reset()
	bytesWritten, err = writeFull(&buf, realWrite)
	testutils.FatalUnless(t, bytesWritten == 4, "writeFull wrote %v bytes instead of the expected 4", bytesWritten)
	testutils.FatalUnless(t, err == nil, "writeFull failed on write with bytes.Buffer with error %v", err)

	content := buf.Bytes()
	testutils.FatalUnless(t, bytes.Equal(content, realWriteCopy), "writeFull did not write to buffer as expected. Content after write %v", content)

	designatedErr := errors.New("ExpectedError")
	var faultyBuf2 = testutils.NewFaultyBuffer(2, designatedErr) // write will fail after 2 bytes

	bytesWritten, err = writeFull(faultyBuf2, emptyWrite)
	testutils.FatalUnless(t, bytesWritten == 0, "writeFull writes more than 0 bytes on empty byte on faultyBuf2")
	testutils.FatalUnless(t, err == nil, "writeFull failed on empty write with faultyBuf2 with error %v", err)

	faultyBuf2.Reset()

	bytesWritten, err = writeFull(faultyBuf2, realWrite)
	testutils.FatalUnless(t, bytesWritten == 2, "writeFull wrote %v bytes instead of the expected 2 on faultyBuf2", bytesWritten)
	testutils.FatalUnless(t, errors.Is(err, designatedErr), "writeFull did not fail on write with faultyBuf2 with the expected error. Error was %v", err)

	var errData bandersnatchErrors.WriteErrorData = err.GetData()
	testutils.FatalUnless(t, errData.PartialWrite == true, "writeFull did not report partialWrite")
	testutils.FatalUnless(t, errData.BytesWritten == 2, "writeFull did not report BytesWritten")
	intendedToWrite := errorsWithData.GetDataFromError[struct{ Data []byte }](err).Data

	testutils.FatalUnless(t, bytes.Equal(intendedToWrite, realWriteCopy), "writeFull did not write to buffer as expected. Content after write %v", content)
	testutils.FatalUnless(t, !testutils.CheckSliceAlias(intendedToWrite, realWrite), "writeFull's error Data aliases the given data")

}
