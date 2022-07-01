package fieldElements

import (
	"io"
	"math/bits"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/bandersnatchErrors"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/common"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/errorsWithData"
	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
)

// forwarding types and variables / constants so users don't need to include /common
type BitHeader = common.BitHeader
type FieldElementEndianness = common.FieldElementEndianness

var (
	LittleEndian  FieldElementEndianness = common.LittleEndian
	BigEndian     FieldElementEndianness = common.BigEndian
	DefaultEndian FieldElementEndianness = common.DefaultEndian
)

// SerializeWithPrefix is used to serialize the given number with some extra prefix bits squeezed into the most significant byte of the field element.
// This function is needed for "compressed" serialization of curve points, where we often need to write an extra sign bit.
//
// Notably, it performs the following operation:
// Reduce the field element modulo BaseFieldSize and interpret it as a 256-bit string (Note the most significant bit is always zero, because BaseFieldSize has only 255 bits).
// Ensure the prefix.PrefixLen many most significant bits of the field element are zero. If so, then temporarily replace those bits with prefix.PrefixBits and write the resulting 256 bits=32 bytes to output in byte order determined by byteOrder.
//
// prefix is a BitHeader, meaning it consists of PrefixBits and PrefixLen. Note that if e.g. PrefixLen==3, then PrefixBits has at most 3 bits;
// those 3 bits are is lsb position inside PrefixBits (e.g. PrefixBits = 0b101), even though they end up in higher-order bits during serialization.
//
// output is a io.Writer. Use e.g. the standard library type bytes.Buffer to wrap an existing byte-slice.
//
// byteOrder wraps either binary.BigEndian or binary.LittleEndian from the standard library. We provide a BigEndian, LittleEndian, DefaultEndian constant for this.
// Note that the endiannness choice only affects the order in which the bytes are written to output, NOT the replacement above, which always happens inside the most signifant byte.
//
// It returns the number of actually written bytes and an error (nil if ok).
// If the prefix.prefixLen bits of z are not all zero, we report an error wrapping ErrPrefixDoesNotFit and do not write anything to output.
// On other (io-related) errors, we might perform (partial) writes to output.
//
// Possible errors: io errors and ErrPrefixDoesNotFit (all possibly wrapped)
func (z *bsFieldElement_64) SerializeWithPrefix(output io.Writer, prefix BitHeader, byteOrder FieldElementEndianness) (bytes_written int, err errorsWithData.ErrorWithGuaranteedParameters[struct{ PartialWrite bool }]) {
	var low_endian_words [4]uint64 = z.undoMontgomery() // words in low endian order in the "obvious" representation.
	prefix_length := prefix.PrefixLen()
	prefix_bits := prefix.PrefixBits()
	if bits.LeadingZeros64(low_endian_words[3]) < int(prefix_length) {
		err = errorsWithData.NewErrorWithParametersFromData(ErrPrefixDoesNotFit, "", &struct{ PartialWrite bool }{false})
		return
	}

	// put prefix into msb of low_endian_words
	low_endian_words[3] |= (uint64(prefix_bits) << (64 - prefix_length))

	var errPlain error

	var buf []byte = make([]byte, 32)
	byteOrder.PutUint256(buf, low_endian_words)
	bytes_written, errPlain = output.Write(buf)
	err = errorsWithData.IncludeDataInError(errPlain, &struct{ PartialWrite bool }{bytes_written != 0 && bytes_written != 32})
	return
}

// DeserializeAndGetPrefix is an inverse to SerializeWithPrefix. It reads a 32*8 bit number from input in byte order determined by byteOrder;
// The prefixLength many most significant bits of the resulting number are returned in prefix, the remaining bits are interpreted and stored as a field element.
//
// As with SerializeWithPrefix, the prefix bits are returned in the lower-order bits (i.e. shifted), even though they belonged to the most significant bits inside the most significant byte of the input.
// prefixLength can be at most 8.
//
// On error, we return a non-nil error in err.
// If the integer to be stored (modulo BaseFieldSize) in z is not the smallest non-negative representative of the field element (this can only happen with prefix_length <= 1), we set
// we return an error wrapping ErrNonNormalizedDeserialization. This error is only returned if no other error occurred and in this case we write the number to z.
// On all other errors, z is untouched.
//
// possible errors: errors wrapping ErrPrefixLengthInvalid, ErrInvalidByteOrder, ErrNonNormalizedDeserialization, io errors
func (z *bsFieldElement_64) DeserializeAndGetPrefix(input io.Reader, prefixLength uint8, byteOrder FieldElementEndianness) (bytesRead int, prefix common.PrefixBits, err errorsWithData.ErrorWithGuaranteedParameters[struct{ PartialRead bool }]) {
	if prefixLength > common.MaxLengthPrefixBits {
		err = errorsWithData.NewErrorWithParametersFromData(ErrPrefixLengthInvalid, "", &struct{ PartialRead bool }{false})
		return
	}
	var errPlain error
	// We read all input into buf first, because we don't want to touch z on most errors.
	buf := make([]byte, 32)
	bytesRead, errPlain = io.ReadFull(input, buf)
	if errPlain != nil {
		err = errorsWithData.IncludeDataInError(errPlain, &struct{ PartialRead bool }{bytesRead != 0 && bytesRead != 32})
		return
	}

	// This writes to z in non-Montgomery form (including the prefix, which we will remove subsequently)
	z.words = byteOrder.Uint256(buf)

	// read out the top prefixLength many bits.
	prefix = common.PrefixBits(z.words[3] >> (64 - prefixLength))

	// clear those bits from z
	var bitmask_remaining uint64 = 0xFFFFFFFF_FFFFFFFF >> prefixLength
	z.words[3] &= bitmask_remaining

	if !z.isNormalized() {
		err = errorsWithData.NewErrorWithParametersFromData(ErrNonNormalizedDeserialization, "", &struct{ PartialRead bool }{false})

		// We do not immediately return, because we put z in Montgomery form before, such that the output is what we read modulo BaseFieldSize, even though we have an error.
	}
	z.restoreMontgomery()
	return
}

// DeserializeWithPrefix works like DeserializeAndGetPrefix, but instead of returning a prefix, it checks whether an expected prefix is present;
// it is intended to verify and consume expected "headers" of sub-byte size.
//
// If the prefix is not present, we return an error wrapping ErrPrefixMismatch and do not write to z.
// Similar to DeserializeAndGetPrefix, we return err=ErrNonNormalizedDeserialization iff the non-negative integer value that is to be written to z modulo BaseFieldSize is not the smallest representative and no other error occurred.
// In this case, we actually write to z. On all other errors, z is untouched.
//
// Note: In the big endian case, we only read 1 byte (which contains the prefix) in case of a prefix-mismatch.
// For the little endian case, we always try to read 32 bytes.
// This behaviour might change in the future. Do not rely on it and check the returned bytesRead.
func (z *bsFieldElement_64) DeserializeWithPrefix(input io.Reader, expectedPrefix BitHeader, byteOrder FieldElementEndianness) (bytesRead int, err errorsWithData.ErrorWithGuaranteedParameters[struct{ PartialRead bool }]) {

	// var fieldElementBuffer bsFieldElement_64
	var little_endian_words [4]uint64 // we do not write to z directly, because we need to check for errors first.
	var buf [32]byte                  // for receiving the input of io.ReadFull

	var errPlain error // io error; we need to add PartialRead flag
	// automatically fill err from errPlain at the end
	defer func() {
		err = errorsWithData.IncludeDataInError(errPlain, &struct{ PartialRead bool }{PartialRead: bytesRead != 0 && bytesRead != 32})
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
			bandersnatchErrors.UnexpectEOF(&errPlain) // Replace io.EOF -> io.ErrUnexpectedEOF
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

		// No return; we need undo Montgomery representation first.
	}
	z.restoreMontgomery()
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
func (z *bsFieldElement_64) Deserialize(input io.Reader, byteOrder FieldElementEndianness) (bytesRead int, err errorsWithData.ErrorWithGuaranteedParameters[struct{ PartialRead bool }]) {
	bytesRead, _, err = z.DeserializeAndGetPrefix(input, 0, byteOrder) // _ == 0
	return
}

// Serialize(output, byteOrder) serializes the received field element to output. It interprets the field element as a 32-byte number in 0<=.<BaseFieldSize (not in Montgomery Form) and tries to write
// 32 bytes to output. byteOrder should be BigEndian or LittleEndian and refers to the ordering of bytes (not words) in output.
// The return values are the actual number of bytes written and a potential error (such as io errors).
// If no error happened, err == nil, in which case we are guaranteed that bytes_written == 32.
func (z *bsFieldElement_64) Serialize(output io.Writer, byteOrder FieldElementEndianness) (bytesWritten int, err errorsWithData.ErrorWithGuaranteedParameters[struct{ PartialWrite bool }]) {
	bytesWritten, err = z.SerializeWithPrefix(output, BitHeader{}, byteOrder)
	return
}
