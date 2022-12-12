package fieldElements

import (
	"bytes"
	"io"
	"math/bits"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/bandersnatchErrors"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/common"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/errorsWithData"
	"github.com/GottfriedHerold/Bandersnatch/internal/errorTransform"
	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
)

// This file is part of the fieldElements package. See the documentation of field_element.go for general remarks.

// This file contains the code used for serializing field elements.

// forwarding types and variables / constants so users don't need to include /common
type BitHeader = common.BitHeader
type FieldElementEndianness = common.FieldElementEndianness

var (
	LittleEndian  FieldElementEndianness = common.LittleEndian
	BigEndian     FieldElementEndianness = common.BigEndian
	DefaultEndian FieldElementEndianness = common.DefaultEndian
)

/***
	General free functions
***/

// handleNonNormalizedReads is an internal function used to handle the case when during reading a field element we read a uint256 that is not reduced.
//
// This function fully reduces z and returns an error wrapping ErrNonNormalizedDeserialization.
// The bytesRead and bitHeader parameters are only used to get the error metadata right:
// We use them to write the unreduced z back to
func handleNonNormalizedReads(z *Uint256, bytesRead int, bitHeader common.BitHeader, byteOrder FieldElementEndianness) (err bandersnatchErrors.DeserializationError) {

	var buf bytes.Buffer
	bufBytesWritten, err2 := z.SerializeWithPrefix(&buf, bitHeader, byteOrder)
	if err2 != nil || buf.Len() != 32 || bufBytesWritten != 32 {
		panic(ErrorPrefix + "cannot happen")
	}

	var errData bandersnatchErrors.ReadErrorData = bandersnatchErrors.ReadErrorData{
		PartialRead:  false,
		BytesRead:    bytesRead,
		ActuallyRead: buf.Bytes(), // Note: buf.Bytes() does not copy, but that's OK here, as we throw away the bytes.Buffer
	}
	err = errorsWithData.NewErrorWithParametersFromData(ErrNonNormalizedDeserialization, "", &errData)

	// fully reduce z
	z.Reduce_fa()
	return err
}

func SerializeFieldElementWithPrefix(x FieldElementInterface_common, output io.Writer, prefix BitHeader, byteOrder FieldElementEndianness) (bytesWritten int, err bandersnatchErrors.SerializationError) {
	// could do more efficiently by unrolling (thereby saving a copy).
	var x256 Uint256
	x.ToUint256(&x256)
	return x256.SerializeWithPrefix(output, prefix, byteOrder)
}

func DeserializeFieldElementAndGetPrefix(z FieldElementInterface_common, input io.Reader, prefixLength uint8, byteOrder FieldElementEndianness) (bytesRead int, prefix common.PrefixBits, err bandersnatchErrors.DeserializationError) {
	var zUint256 Uint256
	bytesRead, prefix, err = zUint256.DeserializeAndGetPrefix(input, prefixLength, byteOrder)
	if err != nil {
		return
	}
	if !zUint256.is_fully_reduced() {
		err = handleNonNormalizedReads(&zUint256, bytesRead, common.MakeBitHeader(prefix, prefixLength), byteOrder)
	}
	z.SetUint256(&zUint256)
	return
}

func DeserializeFieldElementWithExpectedPrefix(z FieldElementInterface_common, input io.Reader, expectedPrefix BitHeader, byteOrder FieldElementEndianness) (bytesRead int, err bandersnatchErrors.DeserializationError) {
	var zUint256 Uint256
	bytesRead, err = zUint256.DeserializeWithExpectedPrefix(input, expectedPrefix, byteOrder)
	if err != nil {
		return
	}
	if !zUint256.is_fully_reduced() {
		err = handleNonNormalizedReads(&zUint256, bytesRead, common.BitHeader{}, byteOrder)
	}
	z.SetUint256(&zUint256)
	return
}

// SerializeWithPrefix is used to serialize the given number with some extra prefix bits squeezed into the most significant byte of the field element.
// This function is needed for "compressed" serialization of curve points, where we often need to write an extra sign bit.
//
// Notably, it performs the following operation:
// Reduce the field element modulo BaseFieldSize and interpret it as a 256-bit string (The most significant bit is always zero, because BaseFieldSize has only 255 bits).
// Ensure the prefix.PrefixLen many most significant bits of the field element are zero. If so, then temporarily replace those bits with prefix.PrefixBits and write the resulting 256 bits=32 bytes to output in byte order determined by byteOrder.
//
// prefix is a BitHeader, meaning it consists of PrefixBits and PrefixLen. Note that if e.g. PrefixLen==3, then PrefixBits has at most 3 bits;
// those 3 bits are in lsb position inside PrefixBits (e.g. PrefixBits = 0b101), even though they end up in higher-order bits during serialization.
//
// output is a io.Writer. Use e.g. the standard library bytes.Buffer type to wrap an existing byte-slice.
//
// byteOrder wraps either binary.BigEndian or binary.LittleEndian from the standard library. We provide a BigEndian, LittleEndian, DefaultEndian constant for this.
// Note that the endiannness choice only affects the order in which the bytes are written to output, NOT the replacement above, which always happens inside the most signifant byte.
//
// It returns the number of actually written bytes and an error (nil if ok).
// If the prefix.prefixLen bits of z are not all zero, we report an error wrapping ErrPrefixDoesNotFit and do not write anything to output.
// On other (io-related) errors, we might perform (partial) writes to output.
//
// Possible errors: io errors and ErrPrefixDoesNotFit (all possibly wrapped)
// The error data's BytesWritten always equals the directly returned bytesWritten
func (z *bsFieldElement_MontgomeryNonUnique) SerializeWithPrefix(output io.Writer, prefix BitHeader, byteOrder FieldElementEndianness) (bytesWritten int, err bandersnatchErrors.SerializationError) {
	var zUint256 Uint256 // = z.words.ToNonMontgomery_fc() // words in low endian order in the "obvious" representation.
	zUint256.FromMontgomeryRepresentation_fc(&z.words)
	prefix_length := prefix.PrefixLen()
	prefix_bits := prefix.PrefixBits()
	if bits.LeadingZeros64(zUint256[3]) < int(prefix_length) {
		err = errorsWithData.NewErrorWithParametersFromData(ErrPrefixDoesNotFit, "", &bandersnatchErrors.WriteErrorData{PartialWrite: false, BytesWritten: 0})
		return
	}

	// put prefix into msb of low_endian_words
	zUint256[3] |= (uint64(prefix_bits) << (64 - prefix_length))

	var errPlain error

	var buf []byte = make([]byte, 32)
	byteOrder.PutUint256(buf, zUint256)
	bytesWritten, errPlain = output.Write(buf)
	err = errorsWithData.IncludeDataInError(errPlain, &bandersnatchErrors.WriteErrorData{PartialWrite: bytesWritten != 0 && bytesWritten != 32, BytesWritten: bytesWritten})
	return
}

// DeserializeAndGetPrefix is an inverse to SerializeWithPrefix. It reads a 32*8 bit number from input in byte order determined by byteOrder;
// The prefixLength many most significant bits of the resulting number are returned in prefix, the remaining bits are interpreted and stored as a field element.
//
// As with SerializeWithPrefix, the prefix bits are returned in the lower-order bits (i.e. shifted), even though they belonged to the most significant bits inside the most significant byte of the input.
// prefixLength can be at most 8.
//
// On error, we return a non-nil error in err.
// If the integer to be stored (modulo BaseFieldSize) in z is not the smallest non-negative representative of the field element (this can only happen with prefix_length <= 1),
// we return an error wrapping ErrNonNormalizedDeserialization. This error is only returned if no other error occurred and in this case we write the number to z.
// On all other errors, z is untouched.
//
// possible errors: errors wrapping ErrPrefixLengthInvalid, ErrInvalidByteOrder, ErrNonNormalizedDeserialization, io errors
// The error data's ActuallyRead and BytesRead are guaranteed to contain the raw bytes and their number that were read; ActuallyRead is nil if no read attempt was made due to invalid function arguments.
func (z *bsFieldElement_MontgomeryNonUnique) DeserializeAndGetPrefix(input io.Reader, prefixLength uint8, byteOrder FieldElementEndianness) (bytesRead int, prefix common.PrefixBits, err bandersnatchErrors.DeserializationError) {
	bytesRead, prefix, err = z.words.DeserializeAndGetPrefix(input, prefixLength, byteOrder)
	if err != nil {
		return
	}
	if !z.isNormalized() {

		// Try to reconstruct the raw bytes we just read
		var buf bytes.Buffer
		bufBytesWritten, err2 := z.words.SerializeWithPrefix(&buf, common.MakeBitHeader(prefix, prefixLength), byteOrder)
		if err2 != nil || buf.Len() != 32 || bufBytesWritten != 32 {
			panic(ErrorPrefix + "cannot happen")
		}

		var errData bandersnatchErrors.ReadErrorData = bandersnatchErrors.ReadErrorData{
			PartialRead:  false,
			BytesRead:    bytesRead,
			ActuallyRead: buf.Bytes(),
		}
		err = errorsWithData.NewErrorWithParametersFromData(ErrNonNormalizedDeserialization, "", &errData)

		// We do not immediately return, because we put z in Montgomery form before, such that the output is what we read modulo BaseFieldSize, even though we have an error.
		z.words.Reduce_ca()
	}
	z.words.ConvertToMontgomeryRepresentation_c(&z.words)
	return
}

// DeserializeWithExpectedPrefix works like DeserializeAndGetPrefix, but instead of returning a prefix, it checks whether an expected prefix is present;
// it is intended to verify and consume expected "headers" of sub-byte size.
//
// If the prefix is not present, we return an error wrapping ErrPrefixMismatch and do not write to z.
// Similar to DeserializeAndGetPrefix, we return err wrapping ErrNonNormalizedDeserialization iff the non-negative integer value that is to be written to z modulo BaseFieldSize is not the smallest representative and no other error occurred.
// In this case, we actually write to z. On all other errors, z is untouched.
//
// NOTE: On error, err's BytesRead and ActuallyRead accurately reflect what and how much was read by this method.
// NOTE2: In the big endian case, we only read 1 byte (which contains the prefix) in case of a prefix-mismatch.
// For the little endian case, we always try to read 32 bytes.
// This behaviour might change in the future. Do not rely on it and check the returned bytesRead.
func (z *bsFieldElement_MontgomeryNonUnique) DeserializeWithExpectedPrefix(input io.Reader, expectedPrefix BitHeader, byteOrder FieldElementEndianness) (bytesRead int, err bandersnatchErrors.DeserializationError) {

	// var fieldElementBuffer bsFieldElement_64
	var little_endian_words [4]uint64 // we do not write to z directly, because we need to check for errors first.
	var buf [32]byte                  // for receiving the input of io.ReadFull

	var errPlain error // errors returned to this function;
	// automatically fill err from errPlain at the end
	defer func() {
		err = errorsWithData.IncludeDataInError(errPlain, &bandersnatchErrors.ReadErrorData{
			PartialRead:  bytesRead != 0 && bytesRead != 32,
			BytesRead:    bytesRead,
			ActuallyRead: buf[0:bytesRead],
		})
	}()
	expectedPrefixLength := expectedPrefix.PrefixLen()
	expectedPrefixBits := expectedPrefix.PrefixBits()

	// The case distinction is done to abort reading after 1 byte if the prefix did not match.
	if byteOrder.StartsWithMSB() {
		bytesRead, errPlain = io.ReadFull(input, buf[0:1])
		if errPlain != nil { // ioError (most likely EOF)
			return
		}
		if buf[0]>>(8-expectedPrefixLength) != byte(expectedPrefixBits) {
			errPlain = ErrPrefixMismatch
			return
		}
		bytes_just_read, errPlain := io.ReadFull(input, buf[1:32])
		bytesRead += bytes_just_read
		if errPlain != nil {
			errorTransform.UnexpectEOF(&errPlain) // Replace io.EOF -> io.ErrUnexpectedEOF
			return
		}
	} else {
		bytesRead, errPlain = io.ReadFull(input, buf[0:32])
		if errPlain != nil {
			return
		}
	}

	little_endian_words = byteOrder.Uint256(buf[:])

	// endianness and IO no longer play a role. We have everything in little_endian_words now.
	// Note that for BigEndian, we actually check the prefix twice.

	readPrefixBits := common.PrefixBits(little_endian_words[3] >> (64 - expectedPrefixLength))
	if readPrefixBits != expectedPrefixBits {
		testutils.Assert(!byteOrder.StartsWithMSB()) // We already checked the prefix above and should not have come this far.
		errPlain = ErrPrefixMismatch
		return
	}

	// remove prefix from read data and copy to z.
	var bitmask_remaining uint64 = 0xFFFFFFFF_FFFFFFFF >> expectedPrefixLength
	little_endian_words[3] &= bitmask_remaining
	z.words = little_endian_words

	// Note: We need to call isNormalized before restoreMontgomery (because the latter would normalize).
	if !z.isNormalized() {
		errPlain = ErrNonNormalizedDeserialization
		z.words.Reduce_ca()
		// No return; we need undo Montgomery representation first.
	}
	z.words.ConvertToMontgomeryRepresentation_c(&z.words)
	return
}

// Deserialize(input, byteOrder) deserializes from input, reading 32 bytes from it and interpreting it as an integer.
// The result is stored in the receiver. byteOrder should be either BigEndian or LittleEndian and relates to the order of bytes in input.
// Note that the input byte stream, interpreted as number, must be in 0 <= . < BaseFieldSize.
// We return an error wrapping ErrNonNormalizedDeserialization iff this is violated and we have no other error.
// In this case, the result is still correct modulo BaseFieldSize.
// Other values for err are possible: in particular io errors from input.
//
// If any error other than ErrNonNormalizedDeserialization occurs, we keep z untouched.
func (z *bsFieldElement_MontgomeryNonUnique) Deserialize(input io.Reader, byteOrder FieldElementEndianness) (bytesRead int, err bandersnatchErrors.DeserializationError) {
	bytesRead, _, err = z.DeserializeAndGetPrefix(input, 0, byteOrder) // The ignored _ is guaranteed to be 0
	return
}

// Serialize(output, byteOrder) serializes the received field element to output. It interprets the field element as a 32-byte number in 0<=.<BaseFieldSize (not in Montgomery Form) and tries to write
// 32 bytes to output. byteOrder should be BigEndian or LittleEndian and refers to the ordering of bytes (not words) in output.
// The return values are the actual number of bytes written and a potential error (such as io errors).
// If no error happened, err == nil. In that case we are guaranteed that bytes_written == 32.
func (z *bsFieldElement_MontgomeryNonUnique) Serialize(output io.Writer, byteOrder FieldElementEndianness) (bytesWritten int, err bandersnatchErrors.SerializationError) {
	bytesWritten, err = z.SerializeWithPrefix(output, BitHeader{}, byteOrder)
	return
}
