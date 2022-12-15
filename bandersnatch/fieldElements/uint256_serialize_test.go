package fieldElements

import (
	"bytes"
	"errors"
	"io"
	"math/rand"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/common"
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

func TestUint256SerializePrefix(t *testing.T) {
	const num = 1000

	var xs []Uint256 = CachedUint256.GetElements(SeedAndRange{allowedRange: twoTo256_Int, seed: 10001}, num)
	var prefixes []BitHeader = []BitHeader{
		BitHeader{}, // == MakeBitHeader(0,0)
		common.MakeBitHeader(0b0, 1),
		common.MakeBitHeader(0b1, 1),
		common.MakeBitHeader(0b10, 2),
		common.MakeBitHeader(0b00, 2),
		common.MakeBitHeader(0b100, 3),
		common.MakeBitHeader(0b11111111, 8),
	}
	designatedError := errors.New("some_error")

	for _, endianness := range []FieldElementEndianness{BigEndian, LittleEndian, DefaultEndian} {
		for _, x := range xs {
			bitLen := x.BitLen()

			// writing to good buffer, roundtrip check for DeserializeAndGetPrefix
			{
				var buf bytes.Buffer
				for _, prefix := range prefixes {
					// try writing with prefix and check if it works.
					lenStored := len(buf.Bytes())
					bytesWritten, writeError := x.SerializeWithPrefix(&buf, prefix, endianness)
					testutils.FatalUnless(t, len(buf.Bytes()) == lenStored+bytesWritten, "bytesWritten wrong")
					prefixFit := bitLen+int(prefix.PrefixLen()) <= 256
					if !prefixFit { // we expect an error
						testutils.FatalUnless(t, writeError != nil, "Uint256.SerializeWithPrefix did not report error, even though prefix did not fit")
						testutils.FatalUnless(t, errors.Is(writeError, ErrPrefixDoesNotFit), "Uint256.SerializeWithPrefix did not return expected error: Got %v", writeError)
						testutils.FatalUnless(t, bytesWritten == 0, "")
						errData := writeError.GetData()
						testutils.FatalUnless(t, errData.PartialWrite == false, "")
						testutils.FatalUnless(t, errData.BytesWritten == 0, "")
						continue // no roundtrip tests
					}
					// we expect roundtrip
					testutils.FatalUnless(t, writeError == nil && bytesWritten == 32, "")
					var y Uint256
					bytesRead, prefixBits, readError := y.DeserializeAndGetPrefix(&buf, prefix.PrefixLen(), endianness)
					testutils.FatalUnless(t, readError == nil && bytesRead == 32, "")
					testutils.FatalUnless(t, x == y, "Roundtrip error for SerializeWithPrefix: field element")
					testutils.FatalUnless(t, prefixBits == prefix.PrefixBits(), "Roundtrip error for SerializeWithPrefix: prefix")
				}
			}

			// same as above, but writing to bad buffer where we get IO errors.
			{
				var faultyBuf testutils.FaultyBuffer = *testutils.NewFaultyBuffer(16, designatedError)
				for _, prefix := range prefixes {
					faultyBuf.Reset()
					prefixFit := bitLen+int(prefix.PrefixLen()) <= 256
					bytesWritten, writeError := x.SerializeWithPrefix(&faultyBuf, prefix, endianness)
					if !prefixFit {
						// same as above
						testutils.FatalUnless(t, writeError != nil, "Uint256.SerializeWithPrefix did not report error, even though prefix did not fit")
						testutils.FatalUnless(t, errors.Is(writeError, ErrPrefixDoesNotFit), "Uint256.SerializeWithPrefix did not return expected error: Got %v", writeError)
						testutils.FatalUnless(t, bytesWritten == 0, "")
						errData := writeError.GetData()
						testutils.FatalUnless(t, errData.PartialWrite == false, "")
						testutils.FatalUnless(t, errData.BytesWritten == 0, "")
						continue
					}
					// expect IO error:
					testutils.FatalUnless(t, writeError != nil, "Write to faulty buf did not cause error")
					testutils.FatalUnless(t, errors.Is(writeError, designatedError), "Write to faulty buf gave unexpected error %v", writeError)
					testutils.FatalUnless(t, bytesWritten == 16, "Write to faulty buf gave unexpted number of bytes Written %v", bytesWritten)

					errData := writeError.GetData()
					testutils.FatalUnless(t, errData.PartialWrite == true, "")
					testutils.FatalUnless(t, errData.BytesWritten == 16, "")
				}
			}

			// reading too large prefix ought to fail as specified
			{
				var buf1 *bytes.Buffer = &bytes.Buffer{}
				var buf2 *bytes.Buffer = bytes.NewBuffer(make([]byte, 32))
				for _, buf := range []*bytes.Buffer{buf1, buf2} {
					var y Uint256
					y.SetUint64(12321) // dummy value
					yCopy := y
					bytesRead, _, err := y.DeserializeAndGetPrefix(buf, 9, endianness)
					testutils.FatalUnless(t, bytesRead == 0, "")
					testutils.FatalUnless(t, errors.Is(err, ErrPrefixLengthInvalid), "")
					testutils.FatalUnless(t, yCopy == y, "y was written to on error")
					errData := err.GetData()
					testutils.FatalUnless(t, errData.PartialRead == false, "")
					testutils.FatalUnless(t, errData.ActuallyRead == nil, "")
					testutils.FatalUnless(t, errData.BytesRead == 0, "")
				}
			}

			// Reading from bad Buffer
			{
				for _, prefix := range prefixes {
					var faultyBuf *testutils.FaultyBuffer = testutils.NewFaultyBuffer(16, designatedError)
					var goodBuf bytes.Buffer
					_, errGood := x.SerializeWithPrefix(&goodBuf, prefix, endianness)
					if errGood != nil {
						continue // same as test above
					}
					faultyBuf.SetContent(goodBuf.Bytes())
					var y Uint256
					y.SetUint64(12321) // dummy value
					yCopy := y
					bytesRead, _, errRead := y.DeserializeAndGetPrefix(faultyBuf, prefix.PrefixLen(), endianness)
					testutils.FatalUnless(t, bytesRead == 16, "")
					testutils.FatalUnless(t, errors.Is(errRead, designatedError), "")
					testutils.FatalUnless(t, y == yCopy, "y was written to on error")
					errData := errRead.GetData()
					testutils.FatalUnless(t, errData.PartialRead == true, "")
					testutils.FatalUnless(t, errData.BytesRead == 16, "")
					testutils.FatalUnless(t, bytes.Equal(errData.ActuallyRead, goodBuf.Bytes()[0:16]), "")
				}
			}

			// writing to buffer, deserialize with ExpectedPrefix
			{
				var buf bytes.Buffer
				var badBuf1 = testutils.NewFaultyBuffer(1, designatedError)
				var badBuf2 = testutils.NewFaultyBuffer(31, designatedError)

				for _, prefix := range prefixes {
					// expectedPrefixes[0] == prefix, all others have 1 bit flipped
					var expectedPrefixes []common.BitHeader = make([]common.BitHeader, 0)
					expectedPrefixes = append(expectedPrefixes, prefix)
					for i := 0; i < int(prefix.PrefixLen()); i++ {
						prefixBits := prefix.PrefixBits()
						prefixBits ^= 1 << i
						expectedPrefixes = append(expectedPrefixes, common.MakeBitHeader(prefixBits, prefix.PrefixLen()))
					}

					for i, expectedPrefix := range expectedPrefixes {

						var prefixMatch bool = (i == 0)
						if prefixMatch {
							testutils.Assert(prefix == expectedPrefix)
						}

						// write to buf, badBuf1, badBuf2
						buf.Reset()
						_, writeError := x.SerializeWithPrefix(&buf, prefix, endianness)
						if writeError != nil {
							continue
						}
						badBuf1.SetContent(buf.Bytes())
						badBuf2.SetContent(buf.Bytes())

						// check roundtrip / error behaviour for each of buf, badBuf1, badBuf2 when reading back
						// via DeserializeWithExpectedPrefix
						var y Uint256
						y.SetUint64(12321) // dummy value
						yCopy := y

						{
							bytesRead, readError := y.DeserializeWithExpectedPrefix(&buf, expectedPrefix, endianness)

							if prefixMatch { // expectedPrefix matches the written prefix
								testutils.FatalUnless(t, bytesRead == 32 && readError == nil, "")

								testutils.FatalUnless(t, x == y, "DeserializeWithExpectedPrefix did not read back bytes: written 0x%x, read 0x%x, prefix was %v", x, y, prefix)
							} else {
								testutils.FatalUnless(t, errors.Is(readError, ErrPrefixMismatch), "")
								testutils.FatalUnless(t, y == yCopy, "y was written to on error")
								errData := readError.GetData()
								testutils.FatalUnless(t, errData.PartialRead == (bytesRead > 0 && bytesRead < 32), "")
								// NOTE: errData.ActuallyRead and errData.BytesRead have unspecified behaviour.
							}
						}

						y = yCopy
						bytesRead, readError := y.DeserializeWithExpectedPrefix(badBuf1, expectedPrefix, endianness)
						testutils.FatalUnless(t, readError != nil, "DeserializeWithExpectedPrefix did not return error when called on faultyBuffer, bytesRead == %v. PrefixMatch == %v, endianness == %v", bytesRead, prefixMatch, endianness) // !
						errData := readError.GetData()
						testutils.FatalUnless(t, errData.PartialRead == true, "")
						testutils.FatalUnless(t, bytesRead == 1, "")
						testutils.FatalUnless(t, y == yCopy, "")
						// No hard guarantees about the other entries in errData
						if prefixMatch {
							testutils.FatalUnless(t, errors.Is(readError, designatedError), "")
						} else {
							testutils.FatalUnless(t, errors.Is(readError, designatedError) || errors.Is(readError, ErrPrefixMismatch), "")
						}

						y = yCopy
						bytesRead, readError = y.DeserializeWithExpectedPrefix(badBuf2, expectedPrefix, endianness)
						testutils.FatalUnless(t, readError != nil, "")
						errData = readError.GetData()
						testutils.FatalUnless(t, errData.PartialRead == true, "")
						testutils.FatalUnless(t, y == yCopy, "")
						testutils.FatalUnless(t, bytesRead >= 1 && bytesRead <= 31, "")
						// No hard guarantees about the other entries in errData
						if prefixMatch {
							testutils.FatalUnless(t, errors.Is(readError, designatedError), "")
						} else {
							testutils.FatalUnless(t, errors.Is(readError, designatedError) || errors.Is(readError, ErrPrefixMismatch), "")
						}

					}
				}
			}
		}
	}
}

// test behaviour of deserialization routines upon EOF
func TestUint256DeserializeEOF(t *testing.T) {

	// arbitrary element with high bits unset in both msbyte and lsbyte
	var x Uint256 = InitUint256FromString("0x0102030405060708090a0b0c0d0e0f10_1112131415161718191a1b1c1d1e1f20")
	var xBytes [40]byte // last bytes uninitialized, only first 32 bytes matter
	for i := 0; i < 32; i++ {
		xBytes[i] = byte(i + 1)
	}
	testutils.Assert(x.BitLen() == 256-7)

	var startVal Uint256 = InitUint256FromString("0xf1245123562367347347346ac1412412_15125326347347346235213634763473") // arbitrary value (with all bytes set)
	testutils.Assert(startVal.BitLen() == 256)

	for _, endianness := range []FieldElementEndianness{BigEndian, LittleEndian, DefaultEndian} {
		for i := 0; i < len(xBytes); i++ {

			{
				xCopy1 := xBytes
				xCopy2 := xBytes
				xCopy3 := xBytes
				buf1 := bytes.NewBuffer(xCopy1[0:i])
				buf2 := bytes.NewBuffer(xCopy2[0:i])
				buf3 := bytes.NewBuffer(xCopy3[0:i])
				y1 := startVal
				y2 := startVal
				y3 := startVal

				bytesRead1, err1 := y1.Deserialize(buf1, endianness)
				bytesRead2, _, err2 := y2.DeserializeAndGetPrefix(buf2, 8, endianness)
				bytesRead3, err3 := y3.DeserializeWithExpectedPrefix(buf3, common.MakeBitHeader(common.PrefixBits(0b00), 2), endianness) // Note: expected BitHeader is good, no matter the endianness.

				if err1 != nil {
					testutils.FatalUnless(t, y1 == startVal, "Deserialize changed receiver on error. Error was %v", err1)
				}

				if err2 != nil {
					testutils.FatalUnless(t, y2 == startVal, "DeserializeAndGetPrefix changed receiver on error. Error was %v", err2)
				}

				if err3 != nil {
					testutils.FatalUnless(t, y3 == startVal, "DeserializeWithExpectedPrefix changed receiver on error. Error was %v", err3)
				}

				if i == 0 { // reading from empty io.Reader, we expect io.EOF
					testutils.FatalUnless(t, bytesRead1 == 0, "Unexpected number of bytes read: %v", bytesRead1)
					testutils.FatalUnless(t, bytesRead2 == 0, "Unexpected number of bytes read: %v", bytesRead2)
					testutils.FatalUnless(t, bytesRead3 == 0, "Unexpected number of bytes read: %v", bytesRead3)
					testutils.FatalUnless(t, errors.Is(err1, io.EOF), "Unexpected error: Expected EOF, got %v", err1)
					testutils.FatalUnless(t, errors.Is(err2, io.EOF), "Unexpected error: Expected EOF, got %v", err2)
					testutils.FatalUnless(t, errors.Is(err3, io.EOF), "Unexpected error: Expected EOF, got %v", err3)
					// y's unchanged by the above
					errData1 := err1.GetData()
					errData2 := err2.GetData()
					errData3 := err3.GetData()
					testutils.FatalUnless(t, errData1.PartialRead == false, "Unexpected PartialRead flag")
					testutils.FatalUnless(t, errData2.PartialRead == false, "Unexpected PartialRead flag")
					testutils.FatalUnless(t, errData3.PartialRead == false, "Unexpected PartialRead flag")
					testutils.FatalUnless(t, errData1.BytesRead == 0, "")
					testutils.FatalUnless(t, errData2.BytesRead == 0, "")
					testutils.FatalUnless(t, errData3.BytesRead == 0, "")
				} else if i < 32 { // EOF in the middle of reading
					testutils.FatalUnless(t, bytesRead1 == i, "Unexpected number of bytes read: Input had length %v, but read %v bytes", i, bytesRead1)
					testutils.FatalUnless(t, bytesRead2 == i, "Unexpected number of bytes read: Input had length %v, but read %v bytes", i, bytesRead2)
					testutils.FatalUnless(t, bytesRead3 == i, "Unexpected number of bytes read: Input had length %v, but read %v bytes", i, bytesRead3)
					testutils.FatalUnless(t, errors.Is(err1, io.ErrUnexpectedEOF), "Unexpected error: Expected ErrUnexpectedEOF, got %v", err1)
					testutils.FatalUnless(t, errors.Is(err2, io.ErrUnexpectedEOF), "Unexpected error: Expected ErrUnexpectedEOF, got %v", err2)
					testutils.FatalUnless(t, errors.Is(err3, io.ErrUnexpectedEOF), "Unexpected error: Expected ErrUnexpectedEOF, got %v", err3)
					errData1 := err1.GetData()
					errData2 := err2.GetData()
					errData3 := err3.GetData()
					testutils.FatalUnless(t, errData1.PartialRead == true, "Unexpected PartialRead flag")
					testutils.FatalUnless(t, errData2.PartialRead == true, "Unexpected PartialRead flag")
					testutils.FatalUnless(t, errData3.PartialRead == true, "Unexpected PartialRead flag")
				} else { // read succeeded
					testutils.FatalUnless(t, err1 == nil, "Unexpected error %v", err1)
					testutils.FatalUnless(t, err2 == nil, "Unexpected error %v", err2)
					testutils.FatalUnless(t, err3 == nil, "Unexpected error %v", err3)
					testutils.FatalUnless(t, bytesRead1 == 32, "Unexpected number of bytes read, was expecting 32, got %v", bytesRead1)
					testutils.FatalUnless(t, bytesRead2 == 32, "Unexpected number of bytes read, was expecting 32, got %v", bytesRead2)
					testutils.FatalUnless(t, bytesRead3 == 32, "Unexpected number of bytes read, was expecting 32, got %v", bytesRead3)
					testutils.FatalUnless(t, y1 != startVal, "Successful deserialization did not change receiver")
					testutils.FatalUnless(t, y2 != startVal, "Successful deserialization did not change receiver")
					testutils.FatalUnless(t, y3 != startVal, "Successful deserialization did not change receiver")
				}

			}

		}

	}

}
