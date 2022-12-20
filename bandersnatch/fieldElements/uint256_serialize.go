package fieldElements

import (
	"io"
	"math/bits"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/bandersnatchErrors"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/common"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/errorsWithData"
	"github.com/GottfriedHerold/Bandersnatch/internal/errorTransform"
)

// This file is part of the fieldElements package. See the documentation of field_element.go for general remarks.

// This file contains the code used to serialize Uint256s. Serialization of field elements x works via conversion to this.
// The reason is that we serialize field elements in "plain", non-Montgomery format and do not want the serialization format be dependent
// on the field element type used.

// Deserialize(input, byteOrder) deserializes from input, reading 32 bytes from it and interpreting it as an Uint256 according to byteOrder.
// The result is stored in the receiver. byteOrder should be either BigEndian or LittleEndian and relates to the order of bytes in input.
//
// If any error occurs, z is not modified.
func (z *Uint256) Deserialize(input io.Reader, byteOrder FieldElementEndianness) (bytesRead int, err bandersnatchErrors.DeserializationError) {
	bytesRead, _, err = z.DeserializeAndGetPrefix(input, 0, byteOrder) // The ignored _ is guaranteed to be 0
	return
}

// Serialize(output, byteOrder) serializes the receiver to output. byteOrder should be BigEndian or LittleEndian and refers to the ordering of bytes in the output.
//
// The return values are the actual number of bytes written and a potential error (such as io errors).
// If no error happened, err == nil. In that case we are guaranteed that bytes_written == 32.
func (z *Uint256) Serialize(output io.Writer, byteOrder FieldElementEndianness) (bytesWritten int, err bandersnatchErrors.SerializationError) {
	bytesWritten, err = z.SerializeWithPrefix(output, BitHeader{}, byteOrder)
	return
}

// SerializeWithPrefix is used to serialize the given Uint256 with some extra prefix bits squeezed into the most significant byte.
// This function is needed for "compressed" serialization of curve points, where we often need to write an extra sign bit.
//
// Usage example: z.SerializeWithPrefix(output, common.MakeBitHeader(PrefixBits(0b01), 2), LittleEndian)
//
// Notably, it performs the following operation:
// Ensure the prefix.prefixLen many most significant bits of z are zero.
// If so, then temporarily replace those bits with prefix.prefixBits and write the resulting 256 bits=32 bytes to output in byte order determined by byteOrder.
//
// prefix is a [common.BitHeader], meaning it consists of prefixBits and prefixLen. Note that if e.g. prefixLen==3, then prefixBits has at most 3 bits;
// those 3 bits are in lsb position inside prefixBits (e.g. prefixBits = 0b101), even though they end up in higher-order bits during serialization.
// Note that the fields of BitHeader are non-exported (to ensure invariants). Use [common.MakeBitHeader] to generate a BitHeader.
//
// output is an [io.Writer]. Use e.g. the standard library [bytes.Buffer] type to wrap an existing byte-slice.
//
// byteOrder has type [FieldElementEndianness] and wraps either [binary.BigEndian] or [binary.LittleEndian] from the standard library.
// We provide a BigEndian, LittleEndian, DefaultEndian constant for this.
// Note that the endiannness choice only affects the order in which the bytes are written to output, NOT the replacement above, which always happens inside the most signifant byte.
//
// It returns the number of actually written bytes and an error (nil if ok).
// If the prefix.prefixLen bits of z are not all zero, we report an error wrapping [ErrPrefixDoesNotFit] and do not write anything to output.
// On other (io-related) errors, we might perform (partial) writes to output.
//
// Possible errors: io errors and ErrPrefixDoesNotFit (all possibly wrapped)
// The error data's BytesWritten always equals the directly returned bytesWritten
func (z *Uint256) SerializeWithPrefix(output io.Writer, prefix BitHeader, byteOrder FieldElementEndianness) (bytesWritten int, err bandersnatchErrors.SerializationError) {

	prefix_length := prefix.PrefixLen()
	prefix_bits := prefix.PrefixBits()
	if bits.LeadingZeros64(z[3]) < int(prefix_length) {
		err = errorsWithData.NewErrorWithParametersFromData(ErrPrefixDoesNotFit, "", &bandersnatchErrors.WriteErrorData{PartialWrite: false, BytesWritten: 0})
		return
	}

	zCopy := *z

	// put prefix into msb of low_endian_words
	zCopy[3] |= (uint64(prefix_bits) << (64 - prefix_length))

	var errPlain error

	var buf [32]byte // = make([]byte, 32)
	byteOrder.PutUint256(buf[:], zCopy)
	bytesWritten, errPlain = output.Write(buf[:])
	err = errorsWithData.IncludeDataInError(errPlain, &bandersnatchErrors.WriteErrorData{PartialWrite: bytesWritten != 0 && bytesWritten != 32, BytesWritten: bytesWritten})
	return
}

// DeserializeAndGetPrefix is an inverse to SerializeWithPrefix. It reads a 32*8 bit number from input in byte order determined by byteOrder;
// The prefixLength many most significant bits of the resulting number are returned in prefix, the remaining bits are interpreted and stored into the Uint256 z.
// (This means the prefixLength many most significant bits of z will be zero after a successful read)
//
// As with SerializeWithPrefix, the prefix bits are returned in the lower-order bits (i.e. shifted) inside the 8-bit prefix value, even though they originally belonged to the most significant bits inside the most significant byte of the input.
// prefixLength can be at most 8.
//
// On error, we return a non-nil error in err and do not modify z.
//
// possible errors: errors wrapping ErrPrefixLengthInvalid, io errors
// The error data's ActuallyRead and BytesRead are guaranteed to contain the raw bytes and their number that were read;
// ActuallyRead is nil if no read attempt was made due to invalid function arguments.
func (z *Uint256) DeserializeAndGetPrefix(input io.Reader, prefixLength uint8, byteOrder FieldElementEndianness) (bytesRead int, prefix common.PrefixBits, err bandersnatchErrors.DeserializationError) {
	if prefixLength > common.MaxLengthPrefixBits { // prefixLength > 8
		err = errorsWithData.NewErrorWithParametersFromData(ErrPrefixLengthInvalid, "", &bandersnatchErrors.ReadErrorData{
			PartialRead:  false,
			BytesRead:    0,
			ActuallyRead: nil, // We do not even try to read
		})
		// Should we panic(err) ???
		return
	}
	var errPlain error
	// We read all input into buf first, because we don't want to touch z on intermediate IO errors.
	var buf [32]byte // := make([]byte, 32)
	bytesRead, errPlain = io.ReadFull(input, buf[:])
	if errPlain != nil {
		err = errorsWithData.IncludeDataInError(errPlain, &bandersnatchErrors.ReadErrorData{
			PartialRead:  bytesRead != 0 && bytesRead != 32,
			BytesRead:    bytesRead,
			ActuallyRead: buf[0:bytesRead],
		})
		return
	}

	// Write to z
	*z = byteOrder.Uint256(buf[:])

	// read out the top prefixLength many bits.
	prefix = common.PrefixBits(z[3] >> (64 - prefixLength))

	// clear those bits from z
	var bitmask_remaining uint64 = 0xFFFFFFFF_FFFFFFFF >> prefixLength
	z[3] &= bitmask_remaining

	return
}

// DeserializeWithExpectedPrefix works like DeserializeAndGetPrefix, but instead of returning a prefix, it checks whether an expected prefix is present;
// it is intended to verify and consume expected "headers" of sub-byte size.
//
// If the prefix is not present, we return an error wrapping ErrPrefixMismatch.
// On any error, we do not write to z.
//
// NOTE: On error, err's BytesRead and ActuallyRead accurately reflect what and how much was read by this method.
// NOTE2: In the big endian case, we only read 1 byte (which contains the prefix) in case of a prefix-mismatch.
// For the little endian case, we always try to read 32 bytes.
// This behaviour might change in the future. Do not rely on it and check the returned bytesRead.
func (z *Uint256) DeserializeWithExpectedPrefix(input io.Reader, expectedPrefix BitHeader, byteOrder FieldElementEndianness) (bytesRead int, err bandersnatchErrors.DeserializationError) {
	// var fieldElementBuffer bsFieldElement_64
	var zTemp [4]uint64 // we do not write to z directly, because we need to check for errors first.
	var buf [32]byte    // for receiving the input of io.ReadFull

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
		var bytes_just_read int
		bytes_just_read, errPlain = io.ReadFull(input, buf[1:32])
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

	zTemp = byteOrder.Uint256(buf[:])

	// endianness and IO no longer play a role. We have everything in zTemp now.
	// Note that for BigEndian, we actually check the prefix twice.

	readPrefixBits := common.PrefixBits(zTemp[3] >> (64 - expectedPrefixLength))
	if readPrefixBits != expectedPrefixBits {
		if byteOrder.StartsWithMSB() {
			panic(ErrorPrefix + "Cannot happen") // We already checked the prefix above and should not have come this far.
		}
		errPlain = ErrPrefixMismatch
		return
	}

	// remove prefix from read data and copy to z.
	var bitmask_remaining uint64 = 0xFFFFFFFF_FFFFFFFF >> expectedPrefixLength
	zTemp[3] &= bitmask_remaining
	*z = zTemp

	return
}
