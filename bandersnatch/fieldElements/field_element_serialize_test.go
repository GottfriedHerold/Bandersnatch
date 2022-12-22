package fieldElements

import (
	"bytes"
	"errors"
	"io"
	"math/bits"
	"math/rand"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/bandersnatchErrors"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/common"
	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
)

// This file is part of the fieldElements package. See the documentation of field_element.go for general remarks.

type fe_serialization_fun[FEPtr FieldElementInterface_common] func(x FEPtr, output io.Writer, byteOrder FieldElementEndianness) (int, bandersnatchErrors.SerializationError)
type fe_deserialization_fun[FEPtr FieldElementInterface_common] func(x FEPtr, input io.Reader, byteOrder FieldElementEndianness) (int, bandersnatchErrors.DeserializationError)
type fe_serializeWithPrefix_fun[FEPtr FieldElementInterface_common] func(x FEPtr, output io.Writer, prefix BitHeader, byteOrder FieldElementEndianness) (int, bandersnatchErrors.SerializationError)
type fe_deserializeAndGetPrefix_fun[FEPtr FieldElementInterface_common] func(x FEPtr, input io.Reader, prefixLength uint8, byteOrder FieldElementEndianness) (int, common.PrefixBits, bandersnatchErrors.DeserializationError)
type fe_deserializeWithExpectedPrefix_fun[FEPtr FieldElementInterface_common] func(x FEPtr, input io.Reader, expectedPrefix BitHeader, byteOrder FieldElementEndianness) (int, bandersnatchErrors.DeserializationError)

type hasSerializer interface {
	Serialize(io.Writer, FieldElementEndianness) (int, bandersnatchErrors.SerializationError)
}
type hasDeserializer interface {
	Deserialize(io.Reader, FieldElementEndianness) (int, bandersnatchErrors.DeserializationError)
}

type hasPrefixSerializerAndDeserializer interface {
	SerializeWithPrefix(io.Writer, BitHeader, FieldElementEndianness) (int, bandersnatchErrors.SerializationError)
	DeserializeAndGetPrefix(io.Reader, uint8, FieldElementEndianness) (int, common.PrefixBits, bandersnatchErrors.DeserializationError)
	DeserializeWithExpectedPrefix(io.Reader, BitHeader, FieldElementEndianness) (int, bandersnatchErrors.DeserializationError)
}

// internal sanity check if the above was typo-free.
var (
	_ hasSerializer                      = &bsFieldElement_MontgomeryNonUnique{}
	_ hasDeserializer                    = &bsFieldElement_MontgomeryNonUnique{}
	_ hasPrefixSerializerAndDeserializer = &bsFieldElement_MontgomeryNonUnique{}
)

func bind23[Arg1 any, Arg2 any, Arg3 any](f func(arg1 Arg1, arg2 Arg2, arg3 Arg3), arg2 Arg2, arg3 Arg3) func(arg1 Arg1) {
	return func(arg1 Arg1) { f(arg1, arg2, arg3) }
}

func bind234[Arg1 any, Arg2 any, Arg3 any, Arg4 any](f func(arg1 Arg1, arg2 Arg2, arg3 Arg3, arg4 Arg4), arg2 Arg2, arg3 Arg3, arg4 Arg4) func(arg1 Arg1) {
	return func(arg1 Arg1) { f(arg1, arg2, arg3, arg4) }
}

func bind2345[Arg1 any, Arg2 any, Arg3 any, Arg4 any, Arg5 any](f func(arg1 Arg1, arg2 Arg2, arg3 Arg3, arg4 Arg4, arg5 Arg5), arg2 Arg2, arg3 Arg3, arg4 Arg4, arg5 Arg5) func(arg1 Arg1) {
	return func(arg1 Arg1) { f(arg1, arg2, arg3, arg4, arg5) }
}

func TestFESerialization(t *testing.T) {
	t.Run("MontgomeryRepresentation", testFESerialization_All[bsFieldElement_MontgomeryNonUnique])
	t.Run("BigInt-Implementation", testFESerialization_All[bsFieldElement_BigInt])
}

func testFESerialization_All[FEType any, FEPtr interface {
	*FEType
	FieldElementInterface[FEPtr]
}](t *testing.T) {

	// If FEPtr has a method Serialize, we want to get FEPtr.Serialize as a function.
	// Unfortunately, Go does not let one type-assert on a type parameter (there are numerous feature requests for this),
	// only on a variable of that type.
	// This is quite annoying, because we use ReceiverType.MethodName to treat a method as a function,

	// we would want to declare hasSerializer etc. inside this function, but Go currently does not support
	// type declarations inside generic functions.
	var dummy FEPtr
	var funSer fe_serialization_fun[FEPtr]
	var funDeser fe_deserialization_fun[FEPtr]
	var funSerWithPrefix fe_serializeWithPrefix_fun[FEPtr]
	var funDeserAndGetPrefix fe_deserializeAndGetPrefix_fun[FEPtr]
	var funDeserWithExpectedPrefix fe_deserializeWithExpectedPrefix_fun[FEPtr]

	if _, ok := any(dummy).(hasSerializer); ok {
		funSer = func(x FEPtr, output io.Writer, e FieldElementEndianness) (int, bandersnatchErrors.SerializationError) {
			return any(x).(hasSerializer).Serialize(output, e)
		}
	}
	if _, ok := any(dummy).(hasDeserializer); ok {
		funDeser = func(x FEPtr, input io.Reader, e FieldElementEndianness) (int, bandersnatchErrors.DeserializationError) {
			return any(x).(hasDeserializer).Deserialize(input, e)
		}
	}
	if _, ok := any(dummy).(hasPrefixSerializerAndDeserializer); ok {
		funSerWithPrefix = func(x FEPtr, output io.Writer, prefix BitHeader, e FieldElementEndianness) (int, bandersnatchErrors.SerializationError) {
			return any(x).(hasPrefixSerializerAndDeserializer).SerializeWithPrefix(output, prefix, e)
		}
		funDeserAndGetPrefix = func(x FEPtr, input io.Reader, prefixLength uint8, e FieldElementEndianness) (int, common.PrefixBits, bandersnatchErrors.DeserializationError) {
			return any(x).(hasPrefixSerializerAndDeserializer).DeserializeAndGetPrefix(input, prefixLength, e)
		}
		funDeserWithExpectedPrefix = func(x FEPtr, input io.Reader, expectedPrefix BitHeader, e FieldElementEndianness) (int, bandersnatchErrors.DeserializationError) {
			return any(x).(hasPrefixSerializerAndDeserializer).DeserializeWithExpectedPrefix(input, expectedPrefix, e)
		}
	}
	_ = funSerWithPrefix
	_ = funDeserAndGetPrefix
	_ = funDeserWithExpectedPrefix

	for _, endianness := range []FieldElementEndianness{BigEndian, LittleEndian} {

		if funSer != nil && funDeser != nil {
			t.Run("Serialization Roundtrip (method) "+endianness.String(), bind234(testFESerialization_Roundtrip[FEType, FEPtr, FEPtr], endianness, funSer, funDeser))
		}
		if funDeser != nil {
			t.Run("Deserialization of non-reduced numbers(method) "+endianness.String(), bind23(testFESerialization_NonNormalizedDeserialization[FEType, FEPtr, FEPtr], endianness, funDeser))
			t.Run("Deserialization and EOF (method) "+endianness.String(), bind23(testFESerialization_EOFDeserialization[FEType, FEPtr, FEPtr], endianness, funDeser))
			t.Run("Deserialization with IO errors (method) "+endianness.String(), bind23(testFEDeserialization_IOError[FEType, FEPtr, FEPtr], endianness, funDeser))
		}
		if funSer != nil {
			t.Run("Serialization with IO errors (method) "+endianness.String(), bind23(testFESerialization_IOError[FEType, FEPtr, FEPtr], endianness, funSer))
		}
		if funSerWithPrefix != nil && funDeserAndGetPrefix != nil && funDeserWithExpectedPrefix != nil {
			t.Run("Serialization with Prefix roundtrip (method) "+endianness.String(), bind2345(testFESerialization_PrefixRoundtrip[FEType, FEPtr, FEPtr],
				endianness, funSerWithPrefix, funDeserAndGetPrefix, funDeserWithExpectedPrefix))
			t.Run("Serialization with Prefix error handling (method) "+endianness.String(), bind2345(testFESerialization_PrefixErrorHandling[FEType, FEPtr, FEPtr],
				endianness, funSerWithPrefix, funDeserAndGetPrefix, funDeserWithExpectedPrefix))

		}

		t.Run("Serialization Roundtrip "+endianness.String(), bind234(testFESerialization_Roundtrip[FEType, FEPtr, FieldElementInterface_common], endianness, SerializeFieldElement, DeserializeFieldElement))
		t.Run("Deserializing non-reduced numbers "+endianness.String(), bind23(testFESerialization_NonNormalizedDeserialization[FEType, FEPtr, FieldElementInterface_common], endianness, DeserializeFieldElement))
		t.Run("Deserializing and EOF "+endianness.String(), bind23(testFESerialization_EOFDeserialization[FEType, FEPtr, FieldElementInterface_common], endianness, DeserializeFieldElement))
		t.Run("Serialization with IO errors "+endianness.String(), bind23(testFESerialization_IOError[FEType, FEPtr, FieldElementInterface_common], endianness, SerializeFieldElement))
		t.Run("Deserialization with IO errors "+endianness.String(), bind23(testFEDeserialization_IOError[FEType, FEPtr, FieldElementInterface_common], endianness, DeserializeFieldElement))
		t.Run("Serialization with Prefix roundtrip "+endianness.String(), bind2345(testFESerialization_PrefixRoundtrip[FEType, FEPtr, FieldElementInterface_common],
			endianness, SerializeFieldElementWithPrefix, DeserializeFieldElementAndGetPrefix, DeserializeFieldElementWithExpectedPrefix))
		t.Run("Serialization with Prefix error handling "+endianness.String(), bind2345(testFESerialization_PrefixErrorHandling[FEType, FEPtr, FieldElementInterface_common],
			endianness, SerializeFieldElementWithPrefix, DeserializeFieldElementAndGetPrefix, DeserializeFieldElementWithExpectedPrefix))
	}

}

func testFESerialization_Roundtrip[FEType any, FEPtr interface {
	*FEType
	FieldElementInterface[FEPtr]
	// SerArg -- Go does not let one write this
}, SerArg FieldElementInterface_common](t *testing.T, endianness FieldElementEndianness, serFun fe_serialization_fun[SerArg], deserFun fe_deserialization_fun[SerArg]) {
	prepareTestFieldElements(t)
	const iterations = 1000
	var xs []FEType = GetPrecomputedFieldElements[FEType, FEPtr](10001, iterations)

	// good serialization:
	var buf bytes.Buffer
	for _, x := range xs {
		xSerArg := any(&x).(SerArg)
		bytesWritten, errWrite := serFun(xSerArg, &buf, endianness)
		testutils.FatalUnless(t, errWrite == nil, "Serialization failed with error %v", errWrite)
		testutils.FatalUnless(t, bytesWritten == 32, "unexpected number of bytes written during serialization: Wrote %v, expected 32", bytesWritten)
	}
	// read back
	for _, x := range xs {
		var y FEType
		yFEPtr := FEPtr(&y)
		yDeserArg := any(&y).(SerArg)
		bytesRead, errRead := deserFun(yDeserArg, &buf, endianness)
		testutils.FatalUnless(t, errRead == nil, "Deserialization failed with error %v", errRead)
		testutils.FatalUnless(t, bytesRead == 32, "Unexpected number of bytes read during deserialization: Read %v, expected 32", bytesRead)
		testutils.FatalUnless(t, yFEPtr.IsEqual(&x), "Roundtrip failure")
	}
}

// test deserializing data that corresponds to a non-reduced field element.
func testFESerialization_NonNormalizedDeserialization[FEType any, FEPtr interface {
	*FEType
	FieldElementInterface[FEPtr]
	// SerArg -- Go does not let one write this
}, SerArg FieldElementInterface_common](t *testing.T, endianness FieldElementEndianness, deserFun fe_deserialization_fun[SerArg]) {
	// Test deserialization from buffer that was not created by Serialize -- we get some expected errors here.

	var buf bytes.Buffer
	prepareTestFieldElements(t)
	const iterations = 1000

	var us []Uint256 = CachedUint256.GetElements(SeedAndRange{seed: 10002, allowedRange: twoTo256_Int}, iterations)
	us = append(us, baseFieldSize_uint256)
	us = append(us, zero_uint256)
	us = append(us, one_uint256)
	us = append(us, twiceBaseFieldSize_64)
	L := len(us)
	var isGood []bool = make([]bool, L) // whether us[i] is in [0, BaseFieldSize)
	for i := 0; i < L; i++ {
		isGood[i] = us[i].is_fully_reduced()
	}
	testutils.Assert(!isGood[L-1] && isGood[L-2] && isGood[L-3] && !isGood[L-4])
	for _, u := range us {
		_, err256 := u.Serialize(&buf, endianness)
		testutils.FatalUnless(t, err256 == nil, "Error in Uint256 Serialization, cannot test Field Element implementation")
	}
	testutils.Assert(len(buf.Bytes()) == 32*L)
	bytesCopy := make([]byte, 32*L)
	testutils.Assert(copy(bytesCopy, buf.Bytes()) == 32*L)
	for i := 0; i < L; i++ {
		var y FEType
		yFEPtr := FEPtr(&y)
		yDeserArg := any(&y).(SerArg)
		yFEPtr.SetInt64(12345) // arbitrary value
		bytesRead, errRead := deserFun(yDeserArg, &buf, endianness)
		if isGood[i] {
			testutils.FatalUnless(t, errRead == nil, "Deserialization failed with error %v", errRead)
			testutils.FatalUnless(t, bytesRead == 32, "Unexpected number of bytes read during deserialization: Read %v, expected 32", bytesRead)
			testutils.FatalUnless(t, IsEqualAsUint256(yFEPtr, &us[i]), "Roundtrip failure")
		} else {
			testutils.FatalUnless(t, errors.Is(errRead, ErrNonNormalizedDeserialization), "Deserializing Non-Normalized element returned error %v, which is not the expected one", errRead)
			testutils.FatalUnless(t, bytesRead == 32, "Unexpected number of bytes read during deserialization: Read %v, expected 32", bytesRead)
			var expected Uint256 = us[i]
			expected.Reduce()
			testutils.FatalUnless(t, IsEqualAsUint256(yFEPtr, &expected), "Roundtrip failure")
			errData := errRead.GetData()
			testutils.FatalUnless(t, errData.PartialRead == false, "")
			testutils.FatalUnless(t, errData.BytesRead == 32, "")
			testutils.FatalUnless(t, bytes.Equal(errData.ActuallyRead, bytesCopy[32*i:32*(i+1)]), "ErrorData.actually read was inaccurate") // failing this would actually be OK per spec, but we don't want that.
		}
	}
}

// check EOF behaviour
func testFESerialization_EOFDeserialization[FEType any, FEPtr interface {
	*FEType
	FieldElementInterface[FEPtr]
	// SerArg -- Go does not let one write this
}, SerArg FieldElementInterface_common](t *testing.T, endianness FieldElementEndianness, deserFun fe_deserialization_fun[SerArg]) {

	prepareTestFieldElements(t)
	var buf bytes.Buffer
	buf.Reset()
	// buf is at EOF now, check that this works as expected

	// reading from buf should give EOF
	var y FEType
	yFEPtr := FEPtr(&y)
	yDeserArg := any(&y).(SerArg)
	yFEPtr.SetInt64(12345) // arbitrary value
	yCopy := y
	bytesRead, errRead := deserFun(yDeserArg, &buf, endianness)
	testutils.FatalUnless(t, errors.Is(errRead, io.EOF), "Deserializing empty buffer returned unexpected non-EOF error %v", errRead)
	testutils.FatalUnless(t, bytesRead == 0, "Deserializing empty buffer returned unexpected number of bytes read %v", bytesRead)
	testutils.FatalUnless(t, yFEPtr.IsEqual(&yCopy), "Deserializing empty buffer wrote to receiver")
	errData := errRead.GetData()
	testutils.FatalUnless(t, errData.PartialRead == false, "")
	testutils.FatalUnless(t, errData.BytesRead == 0, "")
	testutils.FatalUnless(t, len(errData.ActuallyRead) == 0, "") // nil or zero-length slice

	// If buf contains 1-31 bytes, we expect an ErrUnexpectedEOF error.
	for i := 1; i < 32; i++ {
		buf.Reset()
		for j := 0; j < i; j++ {
			buf.WriteByte(byte(j))
		}
		testutils.Assert(buf.Len() == i) // 1 <= i <= 31. We expect an error
		var y FEType
		yFEPtr := FEPtr(&y)
		yDeserArg := any(&y).(SerArg)
		yFEPtr.SetInt64(12345) // arbitrary value
		yCopy := y
		bytesRead, errRead := deserFun(yDeserArg, &buf, endianness)
		testutils.FatalUnless(t, errors.Is(errRead, io.ErrUnexpectedEOF), "Deserializing too short buffer returned not ErrUnexpectedEOF, but %v", errRead)
		testutils.FatalUnless(t, bytesRead == i, "Deserializing too short buffer of lenght %v returned %v bytes read", i, bytesRead)
		testutils.FatalUnless(t, yFEPtr.IsEqual(&yCopy), "Deserializing too short buffer wrote to receiver")
		errData := errRead.GetData()
		testutils.FatalUnless(t, errData.PartialRead == true, "")
		testutils.FatalUnless(t, errData.BytesRead == i, "")
		testutils.FatalUnless(t, len(errData.ActuallyRead) == i, "")
		for j := 0; j < i; j++ {
			testutils.FatalUnless(t, errData.ActuallyRead[j] == byte(j), "")
		}
	}
}

// check IO error behaviour for serializer
func testFESerialization_IOError[FEType any, FEPtr interface {
	*FEType
	FieldElementInterface[FEPtr]
	// SerArg -- Go does not let one write this
}, SerArg FieldElementInterface_common](t *testing.T, endianness FieldElementEndianness, serFun fe_serialization_fun[SerArg]) {
	prepareTestFieldElements(t)

	// arbitrary field element
	var x FEType = InitFieldElementFromString[FEType, FEPtr]("0x0102030405060708090a0b0c0d0e0f10_1112131415161718191a1b1c1d1e1f20")
	// xPtr := FEPtr(&x)
	xSerArg := any(&x).(SerArg)

	designatedError := errors.New("designated error")

	// write to buffer using the current endianness. This is just to get buf.Bytes() holding the result of a good write we can compare against.
	expectedBytes := func() []byte {
		var buf bytes.Buffer
		written, err := serFun(xSerArg, &buf, endianness)
		testutils.FatalUnless(t, err == nil && written == 32, "Write failure, aborting this test (look at other tests' failure)")
		return buf.Bytes()
	}()

	// IO failure after writing i bytes
	for i := 0; i < 32; i++ {
		faultyBuf := testutils.NewFaultyBuffer(i, designatedError)
		bytesWritten, writeError := serFun(xSerArg, faultyBuf, endianness)
		testutils.FatalUnless(t, errors.Is(writeError, designatedError), "Did not get expected io error, got %v instead", writeError)
		testutils.FatalUnless(t, bytesWritten == i, "Did not write expected number %v of bytes, wrote %v instead ", i, bytesWritten)
		errData := writeError.GetData()
		testutils.FatalUnless(t, errData.PartialWrite == (i != 0), "")
		testutils.FatalUnless(t, errData.BytesWritten == i, "") // not really required, but we expect it to be true
		testutils.FatalUnless(t, bytes.Equal(faultyBuf.Bytes(), expectedBytes[:i]), "")
	}
}

// check IO error behaviour for deserializer
func testFEDeserialization_IOError[FEType any, FEPtr interface {
	*FEType
	FieldElementInterface[FEPtr]
	// SerArg -- Go does not let one write this
}, SerArg FieldElementInterface_common](t *testing.T, endianness FieldElementEndianness, deserFun fe_deserialization_fun[SerArg]) {
	prepareTestFieldElements(t)

	// arbitrary Uint256
	var xUint256 Uint256 = InitUint256FromString("0x0102030405060708090a0b0c0d0e0f10_1112131415161718191a1b1c1d1e1f20")
	designatedError := errors.New("designated error")

	// write to buffer using the current endianness. This is just to get buf.Bytes() holding some meaningful data to read.
	// We need some non-trivial data (that distinguished the byte ordering) to properly test the error data, hence the particular value of xUint256 above.
	bytesInBuffer := func() []byte {
		var buf bytes.Buffer
		written, err := xUint256.Serialize(&buf, endianness)
		testutils.FatalUnless(t, err == nil && written == 32, "Write failure, aborting this test (look at other tests' failure)")
		return buf.Bytes()
	}()
	bytesCopy := make([]byte, 32) // for copying bytes into

	// IO failure after reading i bytes
	for i := 0; i < 32; i++ {
		// initialize Fault
		faultyBuf := testutils.NewFaultyBuffer(i, designatedError)
		copy(bytesCopy, bytesInBuffer) // need to copy, because SetContent takes ownership of the argument.
		faultyBuf.SetContent(bytesCopy)
		var x FEType
		xSerArg := any(&x).(SerArg)
		xPtr := FEPtr(&x)
		xPtr.SetInt64(123)
		xCopy := x

		bytesRead, readError := deserFun(xSerArg, faultyBuf, endianness)
		testutils.FatalUnless(t, errors.Is(readError, designatedError), "Did not get expected io error, got %v instead", readError)
		testutils.FatalUnless(t, bytesRead == i, "Did not read expected number %v of bytes, read %v instead ", i, bytesRead)
		testutils.FatalUnless(t, xPtr.IsEqual(&xCopy), "Failing read changed receiver")

		errData := readError.GetData()
		testutils.FatalUnless(t, errData.PartialRead == (i != 0), "")
		testutils.FatalUnless(t, errData.BytesRead == i, "") // not really required, but we expect it to be true
		testutils.FatalUnless(t, bytes.Equal(errData.ActuallyRead, bytesInBuffer[:i]), "")
	}
}

// check Roundtrip behaviour for Serialization with prefix
func testFESerialization_PrefixRoundtrip[FEType any, FEPtr interface {
	*FEType
	FieldElementInterface[FEPtr]
	// SerArg -- Go does not let one write this
}, SerArg FieldElementInterface_common](t *testing.T, endianness FieldElementEndianness,
	serWithPrefix fe_serializeWithPrefix_fun[SerArg],
	deserAndGetPrefix fe_deserializeAndGetPrefix_fun[SerArg],
	deserWithExpectedPrefix fe_deserializeWithExpectedPrefix_fun[SerArg],
) {
	prepareTestFieldElements(t)
	const iterations = 1000
	var xs []FEType = GetPrecomputedFieldElements[FEType, FEPtr](10001, iterations)
	var y FEType
	yPtr := FEPtr(&y)
	ySer := any(&y).(SerArg)

	var prefixes []BitHeader = []BitHeader{
		common.MakeBitHeader(0, 0),
		common.MakeBitHeader(0b0, 1),
		common.MakeBitHeader(0b1, 1),
		common.MakeBitHeader(0b10, 2),
		common.MakeBitHeader(0b00, 2),
		common.MakeBitHeader(0b100, 3),
		common.MakeBitHeader(0b11111111, 8),
	}
	var buf bytes.Buffer
	for _, x := range xs {
		xPtr := FEPtr(&x)
		xSer := any(&x).(SerArg)
		buf.Reset()
		var xUint256 Uint256
		xPtr.ToUint256(&xUint256)
		xBitLen := xUint256.BitLen()
		for _, prefix := range prefixes {
			prefixFit := xBitLen+int(prefix.PrefixLen()) <= 256
			bufLen := buf.Len()
			bytesWritten, errWrite := serWithPrefix(xSer, &buf, prefix, endianness)
			testutils.FatalUnless(t, buf.Len() == bufLen+bytesWritten, "")
			if !prefixFit {
				testutils.FatalUnless(t, errors.Is(errWrite, ErrPrefixDoesNotFit), "")
				testutils.FatalUnless(t, bytesWritten == 0, "")
				errData := errWrite.GetData()
				testutils.FatalUnless(t, errData.BytesWritten == 0, "")
				testutils.FatalUnless(t, errData.PartialWrite == false, "")
				continue // skip roundtrip test
			}
			testutils.FatalUnless(t, errWrite == nil, "Unexpected error writing with prefix: %v.", errWrite)
			testutils.FatalUnless(t, bytesWritten == 32, "Unexpected number of bytes written: %v, expected 32", bytesWritten)

			// read back
			testutils.Assert(prefixFit)
			bytesRead, prefixBits, errRead := deserAndGetPrefix(ySer, &buf, prefix.PrefixLen(), endianness)

			testutils.FatalUnless(t, errRead == nil, "unexpected error reading field element and prefix: %v", errRead)
			testutils.FatalUnless(t, bytesRead == 32, "unexpected number of bytes read: got %v, expected 32", bytesRead)
			testutils.FatalUnless(t, prefixBits == prefix.PrefixBits(), "")
			testutils.FatalUnless(t, yPtr.IsEqual(xPtr), "")

			// write again, to read back with expected prefix
			bytesWritten, errWrite = serWithPrefix(xSer, &buf, prefix, endianness)
			testutils.Assert(bytesWritten == 32 && errWrite == nil)

			// read back in with deserWithExpectedPrefix
			yPtr.SetZero()
			bytesRead, errRead = deserWithExpectedPrefix(ySer, &buf, prefix, endianness)
			testutils.FatalUnless(t, errRead == nil, "")
			testutils.FatalUnless(t, bytesRead == 32, "")
			testutils.FatalUnless(t, yPtr.IsEqual(xPtr), "")

			// Now try deserWithExpectedPrefix with a modified prefix and make sure it fails
			for i := 0; i < int(prefix.PrefixLen()); i++ {

				buf.Reset() // we may not read as much as we write, so we may lose sync.

				// flip i'th bit of prefix
				prefixBits := prefix.PrefixBits()
				prefixBits ^= 1 << i
				modifiedPrefix := common.MakeBitHeader(prefixBits, prefix.PrefixLen())

				// write with unmodifed prefix
				bytesWritten, errWrite = serWithPrefix(xSer, &buf, prefix, endianness)
				testutils.Assert(bytesWritten == 32 && errWrite == nil)

				yPtr.SetUint64(12345) // arbitrary value to check that it stays unmodified
				yCopy := y
				bufCopy := make([]byte, 32)
				testutils.Assert(copy(bufCopy, buf.Bytes()) == 32) // make a copy of the buffer's content to check errData.ActuallyRead is correct

				// read with wrong prefix
				bytesRead, errRead = deserWithExpectedPrefix(ySer, &buf, modifiedPrefix, endianness)
				testutils.FatalUnless(t, errors.Is(errRead, ErrPrefixMismatch), "")
				errData := errRead.GetData()
				testutils.FatalUnless(t, bytesRead > 0, "") // must have read something
				testutils.FatalUnless(t, errData.PartialRead == (bytesRead < 32), "")
				testutils.FatalUnless(t, yPtr.IsEqual(&yCopy), "")
				testutils.FatalUnless(t, errData.BytesRead == bytesRead, "")                         // not really required by spec, but current implementation satisfies it.
				testutils.FatalUnless(t, bytes.Equal(errData.ActuallyRead, bufCopy[:bytesRead]), "") // not really required by spec, but current implementation satisfies it.
			}
			buf.Reset()
		}
	}
}

// check error handling for Serialization with prefix
func testFESerialization_PrefixErrorHandling[FEType any, FEPtr interface {
	*FEType
	FieldElementInterface[FEPtr]
	// SerArg -- Go does not let one write this
}, SerArg FieldElementInterface_common](t *testing.T, endianness FieldElementEndianness,
	serWithPrefix fe_serializeWithPrefix_fun[SerArg],
	deserAndGetPrefix fe_deserializeAndGetPrefix_fun[SerArg],
	deserWithExpectedPrefix fe_deserializeWithExpectedPrefix_fun[SerArg],
) {
	prepareTestFieldElements(t)

	const numX = 100
	var xs []FEType = GetPrecomputedFieldElements[FEType, FEPtr](10001, numX)
	xs = append(xs, InitFieldElementFromString[FEType, FEPtr]("0"))
	xs = append(xs, InitFieldElementFromString[FEType, FEPtr]("1"))
	xs = append(xs, InitFieldElementFromString[FEType, FEPtr]("-1"))
	var xSerializedBytes = make([]byte, 32)

	var prefixes []BitHeader = []BitHeader{
		common.MakeBitHeader(0, 0),
		common.MakeBitHeader(0b0, 1),
		common.MakeBitHeader(0b1, 1),
		common.MakeBitHeader(0b10, 2),
		common.MakeBitHeader(0b00, 2),
		common.MakeBitHeader(0b100, 3),
		common.MakeBitHeader(0b11111111, 8),
	}

	for _, x := range xs {
		for _, prefix := range prefixes {
			xPtr := FEPtr(&x)
			xSer := any(&x).(SerArg)
			var goodBuf bytes.Buffer

			// get "good" representation of x:
			_, errPrepare := serWithPrefix(xSer, &goodBuf, prefix, endianness)
			if errPrepare != nil {
				testutils.FatalUnless(t, errors.Is(errPrepare, ErrPrefixDoesNotFit), "Aborting further error handling test, as SerializeWithPrefix does not work. Fix other tests first.")
				continue // skip
			}
			testutils.Assert(copy(xSerializedBytes, goodBuf.Bytes()) == 32)

			// test EOF behaviour for deserialization:
			for i := 0; i < 32; i++ {
				// function to test a given buffer with a given expected error. We define a function to avoid writing it 4 times:
				// We test this with deserTest ~ DeserializeAndGetPrefix or deserTest ~DeserializeWithExpectedPrefix
				// and buf == truncated buffer and buf == io.Reader that gives an io.error after reading i bytes.
				testBuf := func(buf io.Reader, expectedError error, deserTest func(SerArg, io.Reader) (int, bandersnatchErrors.DeserializationError)) {
					var y FEType
					yPtr := FEPtr(&y)
					ySer := any(&y).(SerArg)
					yCopy := y
					bytesRead, errRead := deserTest(ySer, buf)
					testutils.FatalUnless(t, bytesRead == i, "")
					testutils.FatalUnless(t, errors.Is(errRead, expectedError), "")
					testutils.FatalUnless(t, yPtr.IsEqual(&yCopy), "")
					errData := errRead.GetData()
					testutils.FatalUnless(t, errData.PartialRead == (i != 0), "")
					testutils.FatalUnless(t, errData.BytesRead == i, "")
					testutils.FatalUnless(t, bytes.Equal(errData.ActuallyRead, xSerializedBytes[:i]), "")
				}

				// wrap functions to be tested to be usable by the above by creating a closure fixing some arguments.
				wrapDeserAndGetPrefix := func(x SerArg, reader io.Reader) (int, bandersnatchErrors.DeserializationError) {
					bytesRead, _, errRead := deserAndGetPrefix(x, reader, prefix.PrefixLen(), endianness)
					return bytesRead, errRead
				}
				wrapDeserWithExpectedPrefix := func(x SerArg, reader io.Reader) (int, bandersnatchErrors.DeserializationError) {
					return deserWithExpectedPrefix(x, reader, prefix, endianness)
				}

				xSerializedBytesCopy := make([]byte, i)
				testutils.Assert(copy(xSerializedBytesCopy, xSerializedBytes[:i]) == i)
				var buf *bytes.Buffer = bytes.NewBuffer(xSerializedBytesCopy) // takes ownership of xSerializedBytesCopy (hence the need to copy)
				// buf now contains the first i bytes of a good serialization

				// check that deserAndGetPrefix works as intended
				var expectedError error
				if i == 0 {
					expectedError = io.EOF
				} else {
					expectedError = io.ErrUnexpectedEOF
				}
				testBuf(buf, expectedError, wrapDeserAndGetPrefix)
				// same test with DeserializeWithExpectedPrefix:
				testutils.Assert(copy(xSerializedBytesCopy, xSerializedBytes[:i]) == i)
				buf = bytes.NewBuffer(xSerializedBytesCopy) // takes ownership of xSerializedBytesCopy (hence the need to copy)
				testBuf(buf, expectedError, wrapDeserWithExpectedPrefix)

				// test general IO error:
				expectedError = errors.New("designated error")
				faultyBuf := testutils.NewFaultyBuffer(i, expectedError)
				testutils.Assert(copy(xSerializedBytesCopy, xSerializedBytes[:i]) == i)
				faultyBuf.SetContent(xSerializedBytesCopy)
				testBuf(faultyBuf, expectedError, wrapDeserAndGetPrefix)
				testutils.Assert(copy(xSerializedBytesCopy, xSerializedBytes[:i]) == i)
				faultyBuf.SetContent(xSerializedBytesCopy)
				testBuf(faultyBuf, expectedError, wrapDeserWithExpectedPrefix)

				// try serializing to faulty buffer:
				faultyBuf.Reset()
				bytesWritten, errWrite := serWithPrefix(xSer, faultyBuf, prefix, endianness)
				testutils.FatalUnless(t, errors.Is(errWrite, expectedError), "")
				testutils.FatalUnless(t, bytesWritten == i, "")
				errData := errWrite.GetData()
				testutils.FatalUnless(t, errData.PartialWrite == (i != 0), "")
				testutils.FatalUnless(t, errData.BytesWritten == i, "") // not required by spec
				actuallyWritten := faultyBuf.Bytes()
				testutils.FatalUnless(t, len(actuallyWritten) == i, "")
				testutils.FatalUnless(t, bytes.Equal(actuallyWritten, xSerializedBytes[:i]), "")
			}

			// check behaviour of reading non-normalized input:
			var buf bytes.Buffer
			var xUint256 Uint256
			xPtr.ToUint256(&xUint256)
			xUint256.Add(&xUint256, &baseFieldSize_uint256) // cannot overflow
			_, err2 := xUint256.SerializeWithPrefix(&buf, prefix, endianness)
			if err2 != nil {
				continue
			}
			testutils.Assert(copy(xSerializedBytes, buf.Bytes()) == 32)
			var y FEType
			yPtr := FEPtr(&y)
			ySer := any(&y).(SerArg)
			bytesRead, prefixRead, errRead := deserAndGetPrefix(ySer, &buf, prefix.PrefixLen(), endianness)
			testutils.FatalUnless(t, bytesRead == 32, "")
			testutils.FatalUnless(t, errors.Is(errRead, ErrNonNormalizedDeserialization), "")
			testutils.FatalUnless(t, prefixRead == prefix.PrefixBits(), "")
			errData := errRead.GetData()
			testutils.FatalUnless(t, errData.PartialRead == false, "")
			testutils.FatalUnless(t, bytes.Equal(errData.ActuallyRead, xSerializedBytes), "")
			testutils.FatalUnless(t, errData.BytesRead == 32, "")
			testutils.FatalUnless(t, yPtr.IsEqual(xPtr), "")

			buf.Reset()
			xUint256.SerializeWithPrefix(&buf, prefix, endianness)
			yPtr.SetUint64(12135) // set arbitrary value to check that it gets set to the right one

			bytesRead, errRead = deserWithExpectedPrefix(ySer, &buf, prefix, endianness)
			testutils.FatalUnless(t, bytesRead == 32, "")
			testutils.FatalUnless(t, errors.Is(errRead, ErrNonNormalizedDeserialization), "")
			testutils.FatalUnless(t, prefixRead == prefix.PrefixBits(), "")
			errData = errRead.GetData()
			testutils.FatalUnless(t, errData.PartialRead == false, "")
			testutils.FatalUnless(t, bytes.Equal(errData.ActuallyRead, xSerializedBytes), "prefix: %v\n%v\n%v\n", prefix, errData.ActuallyRead, xSerializedBytes)
			testutils.FatalUnless(t, errData.BytesRead == 32, "")
			testutils.FatalUnless(t, yPtr.IsEqual(xPtr), "")
		}
	}
}

// old test, kept around (doesn't hurt)
func TestSerializeFieldElements(t *testing.T) {
	prepareTestFieldElements(t)
	const iterations = 100
	var drng *rand.Rand = rand.New(rand.NewSource(87))
	var err error // fe.Serialize and fe.Deserialize each give types extending error, but they are incompatible; we cannot use := for this reason.

	// Declared upfront as well for the above reason. Using := in a tuple assignment <foo>, err := ... would create a new err in the local scope of wrong type shadowing err of type error.
	var bytes_written int
	var bytes_read int

	for i := 0; i < iterations; i++ {
		var buf bytes.Buffer
		var fe bsFieldElement_MontgomeryNonUnique
		fe.SetRandomUnsafe(drng)
		// do little endian and big endian half the time
		var byteOrder FieldElementEndianness = LittleEndian
		if i%2 == 0 {
			byteOrder = BigEndian
		}

		bytes_written, err = fe.Serialize(&buf, byteOrder)
		if err != nil {
			t.Fatal("Serialization of field element failed with error ", err)
		}
		if bytes_written != BaseFieldByteLength {
			t.Fatal("Serialization of field element did not write exptected number of bytes")
		}
		var fe2 bsFieldElement_MontgomeryNonUnique
		bytes_read, err = fe2.Deserialize(&buf, byteOrder)
		if err != nil {
			t.Fatal("Deserialization of field element failed with error ", err)
		}
		if bytes_read != BaseFieldByteLength {
			t.Fatal("Deserialization of field element did not read expceted number of bytes")
		}
		if !fe.IsEqual(&fe2) {
			t.Fatal("Deserializing of field element did not reproduce what was serialized")

		}
	}
	for i := 0; i < iterations; i++ {
		var buf bytes.Buffer
		var fe, fe2 bsFieldElement_MontgomeryNonUnique
		fe.SetRandomUnsafe(drng)
		if fe.Sign() < 0 {
			fe.NegEq()
		}
		if fe.Sign() < 0 {
			t.Fatal("Sign does not work as expected")
		}
		if bits.LeadingZeros64(fe.words.ToNonMontgomery_fc()[3]) < 2 {
			t.Fatal("Positive sign field elements do not start with 00")
		}
		var random_prefix common.PrefixBits = (common.PrefixBits(i) / 2) % 4
		var byteOrder FieldElementEndianness = LittleEndian
		if i%2 == 0 {
			byteOrder = BigEndian
		}

		bytes_written, err = fe.SerializeWithPrefix(&buf, common.MakeBitHeader(random_prefix, 2), byteOrder)
		if err != nil || bytes_written != BaseFieldByteLength {
			t.Fatal("Serialization of field element failed with long prefix: ", err)
		}
		bytes_read, err = fe2.DeserializeWithExpectedPrefix(&buf, common.MakeBitHeader(random_prefix, 2), byteOrder)
		if err != nil || bytes_read != BaseFieldByteLength {
			t.Fatal("Deserialization of field element failed with long prefix: ", err)
		}
		if !fe.IsEqual(&fe2) {
			t.Fatal("Roundtripping field elements failed with long prefix")
		}
		buf.Reset() // not really needed
		bytes_written, err = fe.SerializeWithPrefix(&buf, common.MakeBitHeader(1, 1), byteOrder)
		if bytes_written != BaseFieldByteLength || err != nil {
			t.Fatal("Serialization of field elements failed on resetted buffer")
		}
		_, err = fe2.DeserializeWithExpectedPrefix(&buf, common.MakeBitHeader(0, 1), byteOrder)
		if !errors.Is(err, ErrPrefixMismatch) {
			t.Fatal("Prefix mismatch was not detected in deserialization of field elements")
		}
		buf.Reset()
		fe.Serialize(&buf, BigEndian)
		buf.Bytes()[0] |= 0x80
		bytes_read, err = fe2.Deserialize(&buf, BigEndian)
		if bytes_read != BaseFieldByteLength || !errors.Is(err, ErrNonNormalizedDeserialization) {
			t.Fatal("Non-normalized field element not recognized as such during deserialization")
		}
	}
}
