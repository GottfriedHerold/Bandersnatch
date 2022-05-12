package pointserializer

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/bandersnatchErrors"
	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
)

// keys in the global serializerParams act case-insensitve, which is implemented via normalization to lowercase. So the entries in the map must be lowercase.
func TestParamsLowercase(t *testing.T) {
	for key := range serializerParams {
		if key != strings.ToLower(key) {
			t.Fatalf("serializerParams has non-lowercased key %v", key)
		}
	}
}

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
	// in particular, this checks that err.Data is non-nil
	if len(err.Data) != 0 {
		t.Fatalf("consumderExpectRead on empty read did not report empty read values on error. Got %v instead", err.Data)
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
	// bytes.Equal would treat nil as empty slice.
	if err.Data == nil {
		t.Fatalf("consumeExpectRead on too short reader reported nil err.Data")
	}
	if !bytes.Equal(err.Data, data[0:3]) {
		t.Fatalf("consumeExpectRead on too short reader reported from data in err.Data. Got %v instead", err.Data)
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
	if err.Data == nil {
		t.Fatalf("consumeExpectRead on mismatched data returned nil in err.Data")
	}
	if !bytes.Equal(err.Data, data[0:5]) {
		t.Fatalf("consumeExpectRead on mismatched data did not return expected err.Data. Got %v instead", err.Data)
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

type dummyGetterOnly struct{}

func (*dummyGetterOnly) GetEndianness() binary.ByteOrder { return binary.LittleEndian }

type dummySetterOnly struct{}

func (*dummySetterOnly) SetEndianness(b binary.ByteOrder) {}

type dummyGetterAndSetter struct {
	dummyGetterOnly
	dummySetterOnly
}

func TestHasParameters(t *testing.T) {
	var nilEndianness *fieldElementEndianness = nil
	if !testutils.CheckPanic(hasParameter[fieldElementEndianness], nilEndianness, "invalidParameter") {
		t.Fatalf("hasParameter did not panic on unrecognized parameter")
	}
	if hasParameter(nilEndianness, "SubgroupOnly") {
		t.Fatalf("hasParameter returned true when it should not")
	}
	if !hasParameter(nilEndianness, "endianness") {
		t.Fatalf("hasParameter returned false when it should not")
	}
	var getterOnly *dummyGetterOnly = nil
	var setterOnly *dummySetterOnly = nil
	var setterAndGetter *dummyGetterAndSetter = nil
	if hasParameter(getterOnly, "endianness") {
		t.Fatalf("hasParameter returned true for struct with getter only")
	}
	if hasParameter(setterOnly, "endianness") {
		t.Fatalf("hasParameter returned true for struct with setter only")
	}
	if !hasParameter(setterAndGetter, "endianness") {
		t.Fatalf("hasParamter returned false for struct with both getter and setter")
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
