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

type fe_serialization_fun[FEPtr FieldElementInterface_common] func(x FEPtr, output io.Writer, byteOrder common.FieldElementEndianness) (int, bandersnatchErrors.SerializationError)
type fe_deserialization_fun[FEPtr FieldElementInterface_common] func(x FEPtr, input io.Reader, byteOrder common.FieldElementEndianness) (int, bandersnatchErrors.DeserializationError)

type hasSerializer interface {
	Serialize(io.Writer, FieldElementEndianness) (int, bandersnatchErrors.SerializationError)
}
type hasDeserializer interface {
	Deserialize(io.Reader, FieldElementEndianness) (int, bandersnatchErrors.DeserializationError)
}

func bind23[Arg1 any, Arg2 any, Arg3 any](f func(arg1 Arg1, arg2 Arg2, arg3 Arg3), arg2 Arg2, arg3 Arg3) func(arg1 Arg1) {
	return func(arg1 Arg1) { f(arg1, arg2, arg3) }
}

func bind234[Arg1 any, Arg2 any, Arg3 any, Arg4 any](f func(arg1 Arg1, arg2 Arg2, arg3 Arg3, arg4 Arg4), arg2 Arg2, arg3 Arg3, arg4 Arg4) func(arg1 Arg1) {
	return func(arg1 Arg1) { f(arg1, arg2, arg3, arg4) }
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

		t.Run("Serialization Roundtrip "+endianness.String(), bind234(testFESerialization_Roundtrip[FEType, FEPtr, FieldElementInterface_common], endianness, SerializeFieldElement, DeserializeFieldElement))
		t.Run("Deserializing non-reduced numbers "+endianness.String(), bind23(testFESerialization_NonNormalizedDeserialization[FEType, FEPtr, FieldElementInterface_common], endianness, DeserializeFieldElement))
		t.Run("Deserializing and EOF "+endianness.String(), bind23(testFESerialization_EOFDeserialization[FEType, FEPtr, FieldElementInterface_common], endianness, DeserializeFieldElement))
		t.Run("Serialization with IO errors "+endianness.String(), bind23(testFESerialization_IOError[FEType, FEPtr, FieldElementInterface_common], endianness, SerializeFieldElement))
		t.Run("Deserialization with IO errors "+endianness.String(), bind23(testFEDeserialization_IOError[FEType, FEPtr, FieldElementInterface_common], endianness, DeserializeFieldElement))
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
