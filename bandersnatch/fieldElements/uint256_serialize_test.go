package fieldElements

import (
	"bytes"
	"errors"
	"io"
	"math/rand"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
)

func TestUint256_SerializationRoundtrip(t *testing.T) {
	prepareTestFieldElements(t)
	const iterations = 1000
	var xs []Uint256 = CachedUint256.GetElements(SeedAndRange{allowedRange: twoTo256_Int, seed: 10001}, iterations)

	// Uint256 -> buffer -> Uint256
	for _, endianness := range []FieldElementEndianness{LittleEndian, BigEndian, DefaultEndian} {
		var buf bytes.Buffer
		for _, x := range xs {
			bytesWritten, err := x.Serialize(&buf, endianness)
			testutils.FatalUnless(t, err == nil, "")
			testutils.FatalUnless(t, bytesWritten == 32, "")
		}
		for _, x := range xs {
			var y Uint256
			bytesRead, err := y.Deserialize(&buf, endianness)
			testutils.FatalUnless(t, err == nil, "")
			testutils.FatalUnless(t, bytesRead == 32, "")
			testutils.FatalUnless(t, x == y, "")
		}
	}

	// buffer -> Uint256 -> buffer
	for _, endianness := range []FieldElementEndianness{LittleEndian, BigEndian, DefaultEndian} {

		var data []byte = make([]byte, 32*iterations)
		var dataCopy []byte = make([]byte, 32*iterations)
		var rng *rand.Rand = rand.New(rand.NewSource(10002))
		written, errRng := rng.Read(data)
		testutils.FatalUnless(t, written == 32*iterations, "internal error")
		testutils.FatalUnless(t, errRng == nil, "")

		written = copy(dataCopy, data)
		testutils.FatalUnless(t, written == 32*iterations, "internal error")
		var buf *bytes.Buffer = bytes.NewBuffer(data)
		var buf2 *bytes.Buffer = new(bytes.Buffer)

		// copy buf -> buf2 via deserialize and serialize until EOF.
		for {
			var x Uint256
			bytesRead, errRead := x.Deserialize(buf, endianness)
			if errors.Is(errRead, io.EOF) {
				testutils.FatalUnless(t, bytesRead == 0, "")
				errData := errRead.GetData()
				testutils.FatalUnless(t, errData.PartialRead == false, "")
				testutils.FatalUnless(t, errData.BytesRead == 0, "")
				testutils.FatalUnless(t, len(errData.ActuallyRead) == 0, "") // might be nil or zero-length slice -- either is OK.
				break
			}
			testutils.FatalUnless(t, errRead == nil, "unexpected deserialization error %v", errRead)
			testutils.FatalUnless(t, bytesRead == 32, "unexpected number %v of bytes read", bytesRead)

			bytesWritten, errWrite := x.Serialize(buf2, endianness)
			testutils.FatalUnless(t, errWrite == nil, "")
			testutils.FatalUnless(t, bytesWritten == 32, "")
		}
		data2 := buf2.Bytes()
		testutils.FatalUnless(t, bytes.Equal(data2, dataCopy), "")
	}
}

func TestUint256Serialize(t *testing.T) {

	// known answer test

	// InitUint256FromString works via big.Int, so "natural" way of writing i.e. BigEndian.
	// This means that x's msbyte is 0x01 and lsbyte is 0x20 == 32
	x := InitUint256FromString("0x0102030405060708090a0b0c0d0e0f10_1112131415161718191a1b1c1d1e1f20")
	var buf bytes.Buffer
	bytesWritten, errWrite := x.Serialize(&buf, LittleEndian)
	testutils.FatalUnless(t, bytesWritten == 32 && errWrite == nil, "")
	xBytes := buf.Bytes()
	testutils.FatalUnless(t, len(xBytes) == 32, "")
	for i := 0; i < 32; i++ {
		testutils.FatalUnless(t, xBytes[i]+byte(i) == 32, "")
	}
	buf.Reset()
	bytesWritten, errWrite = x.Serialize(&buf, BigEndian)
	testutils.FatalUnless(t, bytesWritten == 32 && errWrite == nil, "")
	xBytes = buf.Bytes()
	testutils.FatalUnless(t, len(xBytes) == 32, "")
	for i := 0; i < 32; i++ {
		testutils.FatalUnless(t, xBytes[i] == byte(i+1), "")
	}

	// check behaviour under errors:
	designatedErr := errors.New("fresh error")

	for _, endianness := range []FieldElementEndianness{LittleEndian, BigEndian} {
		// get correct result with given endianness
		buf.Reset()
		x.Serialize(&buf, endianness)
		correctResult := buf.Bytes()

		for i := 0; i < 32; i++ {

			// write to faulty buf that gives IO error after i bytes.
			faultyBuf := testutils.NewFaultyBuffer(i, designatedErr)
			bytesWritten, err := x.Serialize(faultyBuf, endianness)

			// check correct error handling:
			testutils.FatalUnless(t, bytesWritten == i, "")
			testutils.FatalUnless(t, errors.Is(err, designatedErr), "")
			errData := err.GetData()
			testutils.FatalUnless(t, errData.PartialWrite == (i != 0), "")
			testutils.FatalUnless(t, errData.BytesWritten == i, "")

			// determine the i bytes that were actually written to faultyBuf
			var readBuf []byte = make([]byte, i)
			actuallyWritten, err2 := faultyBuf.Read(readBuf)
			testutils.FatalUnless(t, err2 == nil && actuallyWritten == i, "internal error") // problem with FaultyBuffer

			expected := correctResult[0:i]
			testutils.FatalUnless(t, bytes.Equal(readBuf, expected), "Expected to have written: 0x%X, actually wrote 0x%X", expected, readBuf)
		}
	}
}

func TestUint256Deserialize(t *testing.T) {

	// no known answer test. the KAT for Serialize & rountrip takes care of that.

	// check behaviour under errors:
	designatedErr := errors.New("fresh error")
	x := InitUint256FromString("0x0102030405060708090a0b0c0d0e0f10_1112131415161718191a1b1c1d1e1f20")

	for _, endianness := range []FieldElementEndianness{LittleEndian, BigEndian} {
		// get correct result with given endianness
		var buf bytes.Buffer
		x.Serialize(&buf, endianness)
		correctResult := buf.Bytes()

		for i := 0; i < 32; i++ {

			// faulty buf that gives IO error after i bytes.
			faultyBuf := testutils.NewFaultyBuffer(i, designatedErr)
			faultyBuf.SetContent(correctResult) // correctResult may be longer than i.

			bytesRead, err := x.Deserialize(faultyBuf, endianness)

			// check correct error handling:
			testutils.FatalUnless(t, bytesRead == i, "")
			testutils.FatalUnless(t, errors.Is(err, designatedErr), "")
			errData := err.GetData()
			testutils.FatalUnless(t, errData.PartialRead == (i != 0), "")
			testutils.FatalUnless(t, errData.BytesRead == i, "")
			actuallyRead := errData.ActuallyRead
			testutils.FatalUnless(t, bytes.Equal(actuallyRead, correctResult[0:i]), "Expected to have read: 0x%X, actually read 0x%X", correctResult[0:i], actuallyRead)
		}
	}
}
