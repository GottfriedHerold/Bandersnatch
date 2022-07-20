package pointserializer

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

var _ headerDeserializer = &simpleHeaderDeserializer{}
var _ headerSerializer = &simpleHeaderSerializer{}
var _ headerDeserializer = &simpleHeaderSerializer{}

var _ utils.Clonable[*simpleHeaderDeserializer] = &simpleHeaderDeserializer{}
var _ utils.Clonable[*simpleHeaderSerializer] = &simpleHeaderSerializer{}

func TestFixNilEntries(t *testing.T) {
	var shd simpleHeaderDeserializer
	shd.sliceSizeEndianness = binary.LittleEndian
	shd.fixNilEntries()
	for _, arg := range [][]byte{shd.headerSlice, shd.footerSlice, shd.headerPerCurvePoint, shd.footerPerCurvePoint, shd.headerSingleCurvePoint, shd.footerSingleCurvePoint} {
		if arg == nil {
			t.Fatalf("fixNilEntries kept nil")
		}
	}
	shd.Validate()
}

func TestRecognizeParameterNames(t *testing.T) {
	for _, arg := range headerSerializerParams {
		arg = strings.ToLower(arg)
		_, ok := serializerParams[arg]
		if !ok {
			t.Fatalf("serializer parameter named %v not recognized by global parameter lookup table", arg)
		}
	}
}

// getParamDirectlyForSimpleHeaderDeserializer returns the []byte stored in a simpleHeaderDeserializer bypassing the getter.
func getParamDirectlyForSimpleHeaderDeserializer(shd *simpleHeaderDeserializer, paramName string) []byte {
	switch paramName {
	case "GlobalSliceHeader":
		return shd.headerSlice
	case "GlobalSliceFooter":
		return shd.footerSlice
	case "SinglePointHeader":
		return shd.headerSingleCurvePoint
	case "SinglePointFooter":
		return shd.footerSingleCurvePoint
	case "PerPointHeader":
		return shd.headerPerCurvePoint
	case "PerPointFooter":
		return shd.footerPerCurvePoint
	default:
		panic("unrecognized paramName")
	}
}

func TestSettersAndGetters(t *testing.T) {
	var shd simpleHeaderDeserializer
	shd.sliceSizeEndianness = binary.LittleEndian
	shd.Validate()

	var m map[string][]byte = make(map[string][]byte)

	for _, paramName := range headerSerializerParams {
		m[paramName] = []byte(paramName)
		shd = makeCopyWithParams(&shd, paramName, m[paramName])
	}
	for _, paramName := range headerSerializerParams {
		arg := getSerializerParam(&shd, paramName).([]byte)
		if arg == nil {
			t.Fatalf("Getter returned nil")
		}
		if testutils.CheckSliceAlias(arg, m[paramName]) {
			t.Fatalf("Getter returned alias to value that was set")
		}
		if !bytes.Equal(arg, m[paramName]) {
			t.Fatalf("Getter did not return value that was set")
		}
		arg2 := getParamDirectlyForSimpleHeaderDeserializer(&shd, paramName)
		if testutils.CheckSliceAlias(arg, arg2) {
			t.Fatalf("Getter returns value that aliases stored value")
		}
		if !bytes.Equal(arg, arg2) {
			t.Fatalf("Getter does not return stored value")
		}

	}
}

var testSimpleHeaderSerializer simpleHeaderSerializer

func init() {
	testSimpleHeaderSerializer.sliceSizeEndianness = binary.LittleEndian
	for _, paramName := range headerSerializerParams {
		testSimpleHeaderSerializer = makeCopyWithParams(&testSimpleHeaderSerializer, paramName, []byte(paramName))
	}
	testSimpleHeaderSerializer.Validate()
}

func TestSerializeHeaders(t *testing.T) {
	var buf bytes.Buffer
	const size = 121
	var err error
	w1, err := testSimpleHeaderSerializer.serializeGlobalSliceHeader(&buf, size)
	if err != nil {
		t.Fatalf("Error when writing Global header: %v", err)
	}
	w2, err := testSimpleHeaderSerializer.serializePerPointHeader(&buf)
	if err != nil {
		t.Fatalf("Error when writing per point header: %v", err)
	}
	w3, err := testSimpleHeaderSerializer.serializePerPointFooter(&buf)
	if err != nil {
		t.Fatalf("Error when writing per point footer: %v", err)
	}
	w4, err := testSimpleHeaderSerializer.serializeGlobalSliceFooter(&buf)
	if err != nil {
		t.Fatalf("Error when writing global footer: %v", err)
	}
	w5, err := testSimpleHeaderSerializer.serializeSinglePointHeader(&buf)
	if err != nil {
		t.Fatalf("Error when writing sp header: %v", err)
	}
	w6, err := testSimpleHeaderSerializer.serializeSinglePointFooter(&buf)
	if err != nil {
		t.Fatalf("Error when writing sp footer: %v", err)
	}

	// read back:

	r1, sr, err := testSimpleHeaderSerializer.deserializeGlobalSliceHeader(&buf)
	if err != nil {
		t.Fatalf("Error when reading Global header: %v", err)
	}
	if r1 != w1 {
		t.Fatalf("Bytes Read != Bytes Written for Global Slice Header")
	}
	if sr != size {
		t.Fatalf("Could not read back slice size")
	}

	r2, err := testSimpleHeaderSerializer.deserializePerPointHeader(&buf)
	if err != nil {
		t.Fatalf("Error when reading per point header: %v", err)
	}
	if r2 != w2 {
		t.Fatalf("BytesRead mismatches BytesWritten (2)")
	}
	r3, err := testSimpleHeaderSerializer.deserializePerPointFooter(&buf)
	if err != nil {
		t.Fatalf("Error when reading per point footer: %v", err)
	}
	if r3 != w3 {
		t.Fatalf("BytesRead mismatches BytesWritten (3)")
	}
	r4, err := testSimpleHeaderSerializer.deserializeGlobalSliceFooter(&buf)
	if err != nil {
		t.Fatalf("Error when reading global footer: %v", err)
	}
	if r4 != w4 {
		t.Fatalf("BytesRead mismatches BytesWritten (4)")
	}

	r5, err := testSimpleHeaderSerializer.deserializeSinglePointHeader(&buf)
	if err != nil {
		t.Fatalf("Error when writing sp header: %v", err)
	}
	if r5 != w5 {
		t.Fatalf("BytesRead mismatches BytesWritten (5)")
	}

	r6, err := testSimpleHeaderSerializer.deserializeSinglePointFooter(&buf)
	if err != nil {
		t.Fatalf("Error when writing sp footer: %v", err)
	}
	if r6 != w6 {
		t.Fatalf("BytesRead mismatches BytesWritten (6)")
	}

}
