package pointserializer

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/bandersnatchErrors"
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
	var nilSimpleHeaderDeserializer *simpleHeaderDeserializer = nil
	var nilSimpleHeaderSerializer *simpleHeaderSerializer = nil

	allHeaderParams := nilSimpleHeaderDeserializer.RecognizedParameters()
	allHeaderParams2 := nilSimpleHeaderSerializer.RecognizedParameters()
	// Would not be a problem, but not what we expect.
	if !utils.CompareSlices(allHeaderParams, allHeaderParams2) {
		t.Fatalf("serializer parameter names and deserializer unexpectedly differ")
	}
	for _, arg := range allHeaderParams {
		argNormalized := normalizeParameter(arg)
		_, ok := serializerParams[argNormalized]
		if !ok {
			t.Fatalf("serializer parameter named %v not recognized by global parameter lookup table", arg)
		}
		if !nilSimpleHeaderDeserializer.HasParameter(arg) {
			t.Fatalf("deserializer parameter %v not recognized by HasParameter", arg)
		}
		if !nilSimpleHeaderSerializer.HasParameter(arg) {
			t.Fatalf("serializer parameter %v not recognized by HasParameter", arg)
		}
	}
	if nilSimpleHeaderDeserializer.HasParameter("InvalidParamSAFASF") {
		t.Fatalf("derserializer recognizes invalid parameter")
	}
	if nilSimpleHeaderSerializer.HasParameter("InvalidParamGAGAG") {
		t.Fatalf("serializer recognizes invalid parameter")
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
		shd = makeCopyWithParameters(&shd, paramName, m[paramName])
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

// Serializer that writes a literal "GlobalSliceHeader" etc. as slice header
var testSimpleHeaderSerializer simpleHeaderSerializer

func init() {
	testSimpleHeaderSerializer.sliceSizeEndianness = binary.LittleEndian
	for _, paramName := range headerSerializerParams {
		testSimpleHeaderSerializer = makeCopyWithParameters(&testSimpleHeaderSerializer, paramName, []byte(paramName))
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

// Note: these are methods
type ser_fun = func(io.Writer) (int, bandersnatchErrors.SerializationError)
type deser_fun = func(io.Reader) (int, bandersnatchErrors.DeserializationError)

// Note: these are unbound methods (i.e. functions)
var hs_setter_funs = []func(*simpleHeaderDeserializer, []byte){(*simpleHeaderDeserializer).SetPerPointHeader, (*simpleHeaderDeserializer).SetPerPointFooter, (*simpleHeaderDeserializer).SetSinglePointHeader, (*simpleHeaderDeserializer).SetSinglePointFooter, (*simpleHeaderDeserializer).SetGlobalSliceFooter, (*simpleHeaderDeserializer).SetGlobalSliceHeader}
var hs_getter_funs = []func(*simpleHeaderDeserializer) []byte{(*simpleHeaderDeserializer).GetPerPointHeader, (*simpleHeaderDeserializer).GetPerPointFooter, (*simpleHeaderDeserializer).GetSinglePointHeader, (*simpleHeaderDeserializer).GetSinglePointFooter, (*simpleHeaderDeserializer).GetGlobalSliceFooter, (*simpleHeaderDeserializer).GetGlobalSliceHeader}

func get_ser_funs(serializer *simpleHeaderSerializer) []ser_fun {
	return []ser_fun{serializer.serializePerPointHeader, serializer.serializePerPointFooter, serializer.serializeSinglePointHeader, serializer.serializeSinglePointFooter, serializer.serializeGlobalSliceFooter}
}

func get_deser_funs(serializer *simpleHeaderDeserializer) []deser_fun {
	return []deser_fun{serializer.deserializePerPointHeader, serializer.deserializePerPointFooter, serializer.deserializeSinglePointHeader, serializer.deserializeSinglePointFooter, serializer.deserializeGlobalSliceFooter}
}

func TestHeaderDeserializationIOErrors(t *testing.T) {
	MaxHeaderLength := 0
	for _, getter := range hs_getter_funs {
		newLen := len(getter(&testSimpleHeaderSerializer.simpleHeaderDeserializer))
		// fmt.Printf("%s\n", getter(&testSimpleHeaderSerializer.simpleHeaderDeserializer))
		if newLen > MaxHeaderLength {
			MaxHeaderLength = newLen
		}
	}

	// Note: MaxHeaderLength == 17 here

	MaxHeaderLength += simpleHeaderSliceLengthOverhead // == 21

	for i := 0; i < MaxHeaderLength; i++ {
		designatedErr := errors.New("designated IO error")
		faultyBuf := testutils.NewFaultyBuffer(i, designatedErr) // will fail after reading / writing i bytes
		serializers := get_ser_funs(&testSimpleHeaderSerializer)
		deserializers := get_deser_funs(&testSimpleHeaderSerializer.simpleHeaderDeserializer)
		for j := 0; j < 6; j++ { // The last iteration j == 5 (GlobalSliceHeader) is special
			L := len(hs_getter_funs[j](&testSimpleHeaderSerializer.simpleHeaderDeserializer))
			faultyBuf.Reset()
			if ((i >= L) && (j < 5)) || ((i >= L+simpleHeaderSliceLengthOverhead) && (j == 5)) {
				var bytesWritten int
				var writeErr bandersnatchErrors.SerializationError

				if j < 5 {
					bytesWritten, writeErr = serializers[j](faultyBuf)
				} else {
					bytesWritten, writeErr = testSimpleHeaderSerializer.serializeGlobalSliceHeader(faultyBuf, 200)
				}

				if writeErr != nil {
					t.Fatalf("Unexpected write error %v", j)
				}
				if bytesWritten != L {
					// fmt.Println(bytesWritten)
					// fmt.Println(L)
					// fmt.Println(testSimpleHeaderSerializer.simpleHeaderDeserializer)
					t.Fatalf("Unexpected number of bytes written @%v", j)
				}
				var bytesRead int
				var readErr bandersnatchErrors.DeserializationError
				var sizeRead int32

				if j < 5 {
					bytesRead, readErr = deserializers[j](faultyBuf)
				} else {
					bytesRead, sizeRead, readErr = testSimpleHeaderSerializer.deserializeGlobalSliceHeader(faultyBuf)
					if sizeRead != 200 {
						t.Fatalf("Did not read back slice length")
					}
				}

				if readErr != nil {
					// fmt.Printf("%s", readErr.GetData().ActuallyRead)
					t.Fatalf("Unexpected read error @ parameter %v, error was %v", j, readErr)
				}
				if bytesRead != bytesWritten {
					t.Fatalf("Unexpected number of bytes read @%v", j)
				}
			} else { // we expect to get errors
				var bytesWritten int
				var writeErr bandersnatchErrors.SerializationError

				if j < 5 {
					bytesWritten, writeErr = serializers[j](faultyBuf)
				} else {
					bytesWritten, writeErr = testSimpleHeaderSerializer.serializeGlobalSliceHeader(faultyBuf, 200)
				}

				if writeErr == nil {
					t.Fatalf("Expected write error, but got nil @%v, fault threshold %v", j, i)
				}
				if !errors.Is(writeErr, designatedErr) {
					t.Fatalf("Did not get designated error on write @%v", j)
				}
				if bytesWritten != i {
					t.Fatalf("Did not read as far as it could @%v", j)
				}

				var bytesRead int
				var readErr bandersnatchErrors.DeserializationError

				if j < 5 {
					bytesRead, readErr = deserializers[j](faultyBuf)
				} else {
					bytesRead, _, readErr = testSimpleHeaderSerializer.deserializeGlobalSliceHeader(faultyBuf)

				}

				if readErr == nil {
					t.Fatalf("Expected read error, but got nil @%v", j)
				}
				if !errors.Is(readErr, designatedErr) {
					t.Fatalf("Did not get designated error on read @%v", j)
				}
				if bytesRead != bytesWritten {
					t.Fatalf("Did not read as much as written @%v", j)
				}
			}
		}

	}

}
