package fieldElements

import (
	"bytes"
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

// Serialize(output, byteOrder) serializes the receiver to output. byteOrder should be BigEndian or LittleEndian and refers to the ordering of bytes in the output.
//
// The return values are the actual number of bytes written and a potential error (such as io errors).
// If no error happened, err == nil. In that case we are guaranteed that bytes_written == 32.
//
// There are special-case methods [Serialize_Buffer] and [Serialize_Bytes] for the same functionality for writing to a [bytes.Buffer] and []byte.
// (These are orders of magnitude faster because of the way interfaces in Go work)
func (z *Uint256) Serialize(output io.Writer, byteOrder FieldElementEndianness) (bytesWritten int, err bandersnatchErrors.SerializationError) {

	var errPlain error

	var buf [32]byte // = make([]byte, 32)
	byteOrder.PutUint256_array(&buf, (*[4]uint64)(z))
	bytesWritten, errPlain = output.Write(buf[:]) // Note: because output is an interface, this causes esacpe analysis to fail, so buf is heap-allocated.
	if errPlain != nil {
		err = errorsWithData.AddDataToError_struct(errPlain, &bandersnatchErrors.WriteErrorData{PartialWrite: bytesWritten != 0 && bytesWritten != 32, BytesWritten: bytesWritten})
	}
	return
}

// Serialize_Buffer performs the same functionality as Serialize, but with output of concrete type [*bytes.Buffer].
//
// Due to known issues with Go's function API, this is an order of magnitude more efficient than the general version.
// On failure, this method panics (because [bytes.Buffer] does), so the return value is always (32, nil)
func (z *Uint256) Serialize_Buffer(output *bytes.Buffer, byteOrder FieldElementEndianness) (bytesWritten int, err bandersnatchErrors.SerializationError) {
	// var errPlain error

	var buf [32]byte // = make([]byte, 32)
	byteOrder.PutUint256_array(&buf, (*[4]uint64)(z))
	bytesWritten, _ = output.Write(buf[:]) // bytes.Buffer's Write method is guaranteed to never return an error. It panics instead (if out-of-memory, e.g.)
	/*
		if errPlain != nil {
			err = errorsWithData.IncludeDataInError(errPlain, &bandersnatchErrors.WriteErrorData{PartialWrite: bytesWritten != 0 && bytesWritten != 32, BytesWritten: bytesWritten})
		}
	*/
	return
}

// Serialize(output, byteOrder) serializes the receiver to output, which must hold enough space for at least 32 bytes.
// byteOrder should be [BigEndian] or [LittleEndian] and refers to the ordering of bytes in the output.
//
// The return values are the actual number of bytes written (always 32) and a potential error (always nil).
func (z *Uint256) Serialize_Bytes(output []byte, byteOrder FieldElementEndianness) (bytesWritten int, err bandersnatchErrors.SerializationError) {

	// var errPlain error

	// var buf [32]byte // = make([]byte, 32)
	// byteOrder.PutUint256(buf[:], *z)
	byteOrder.PutUint256_ptr(output, (*[4]uint64)(z))
	bytesWritten = 32
	err = nil
	// bytesWritten, errPlain = output.Write(buf[:])
	// if errPlain != nil {
	//		err = errorsWithData.IncludeDataInError(errPlain, &bandersnatchErrors.WriteErrorData{PartialWrite: bytesWritten != 0 && bytesWritten != 32, BytesWritten: bytesWritten})
	//}
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
// Note that the fields of BitHeader are non-exported (to ensure invariants). Use [common.MakeBitHeader] to generate a [common.BitHeader].
//
// output is an [io.Writer]. Use e.g. the standard library [bytes.Buffer] type to wrap an existing byte-slice.
//
// byteOrder has type [FieldElementEndianness] and wraps either [binary.BigEndian] or [binary.LittleEndian] from the standard library.
// We provide a BigEndian, LittleEndian, DefaultEndian constant for this.
// The endiannness choice only affects the order in which the bytes are written to output, NOT the inclusion of a prefix, which always happens inside the most signifant byte.
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
		err = errorsWithData.NewErrorWithData_struct(ErrPrefixDoesNotFit, "", &bandersnatchErrors.WriteErrorData{PartialWrite: false, BytesWritten: 0})
		return
	}

	zCopy := *z

	// put prefix into msb of low_endian_words
	zCopy[3] |= (uint64(prefix_bits) << (64 - prefix_length))

	return zCopy.Serialize(output, byteOrder)
}

// Note: Putting the common code to construct zCopy from z into a separate functions turned out to be slower.

// SerializeWithPrefix_Buffer is a specialiazition of [SerializeWithPrefix] for the case where the output is a [*bytes.Buffer].
//
// Due to the way interfaces in Go work, this method is an order of magnitude faster.
func (z *Uint256) SerializeWithPrefix_Buffer(output *bytes.Buffer, prefix BitHeader, byteOrder FieldElementEndianness) (bytesWritten int, err bandersnatchErrors.SerializationError) {

	prefix_length := prefix.PrefixLen()
	prefix_bits := prefix.PrefixBits()
	if bits.LeadingZeros64(z[3]) < int(prefix_length) {
		err = errorsWithData.NewErrorWithData_struct(ErrPrefixDoesNotFit, "", &bandersnatchErrors.WriteErrorData{PartialWrite: false, BytesWritten: 0})
		return
	}

	zCopy := *z

	// put prefix into msb of low_endian_words
	zCopy[3] |= (uint64(prefix_bits) << (64 - prefix_length))

	return zCopy.Serialize_Buffer(output, byteOrder)
}

// SerializeWithPrefix_Bytes is a specialization of [SerializeWithPrefix] for the case where the output is a []byte.
//
// The output slice must have at least a size of 32 bytes, otherwise we panic.
// This method cannot return an io error, so the only potential error is ErrPrefixDoesNotFit.
// Due to the way interfaces in Go work, this method is an order of magnitude faster than the general version.
func (z *Uint256) SerializeWithPrefix_Bytes(output []byte, prefix BitHeader, byteOrder FieldElementEndianness) (bytesWritten int, err bandersnatchErrors.SerializationError) {

	prefix_length := prefix.PrefixLen()
	prefix_bits := prefix.PrefixBits()
	if bits.LeadingZeros64(z[3]) < int(prefix_length) {
		err = errorsWithData.NewErrorWithData_struct(ErrPrefixDoesNotFit, "", &bandersnatchErrors.WriteErrorData{PartialWrite: false, BytesWritten: 0})
		return
	}

	zCopy := *z

	// put prefix into msb of low_endian_words
	zCopy[3] |= (uint64(prefix_bits) << (64 - prefix_length))

	return zCopy.Serialize_Bytes(output, byteOrder)
}

// Deserialize(input, byteOrder) deserializes from input, reading 32 bytes from it and interpreting it as an Uint256 according to byteOrder.
// The result is stored in the receiver. byteOrder should be either BigEndian or LittleEndian and relates to the order of bytes in input.
//
// If any error occurs, z is not modified.
func (z *Uint256) Deserialize(input io.Reader, byteOrder FieldElementEndianness) (bytesRead int, err bandersnatchErrors.DeserializationError) {
	// We read all input into buf first, because we don't want to touch z on intermediate IO errors.
	var errPlain error
	var buf [32]byte // := make([]byte, 32)
	bytesRead, errPlain = io.ReadFull(input, buf[:])
	if errPlain != nil {
		err = errorsWithData.AddDataToError_struct(errPlain, &bandersnatchErrors.ReadErrorData{
			PartialRead:  bytesRead != 0 && bytesRead != 32,
			BytesRead:    bytesRead,
			ActuallyRead: buf[0:bytesRead],
		})
		return
	}

	// Write to z
	byteOrder.Uint256_array(&buf, (*[4]uint64)(z))

	return
}

func (z *Uint256) Deserialize_Buffer(input *bytes.Buffer, byteOrder FieldElementEndianness) (bytesRead int, err bandersnatchErrors.DeserializationError) {

	// Optimization: Instead of copying the input into buf, we check for the correct size (this is the only possible error condition) and then read directly from the underlying wrapped []byte.
	if input.Len() < 32 {
		// If the input does not have sufficent size, we actually read it all (draining the buffer) and report an error (This is mostly to be consistent with the general method)
		// For simplicity, we just call the general function, as we don't care about speed on error.
		return z.Deserialize(input, byteOrder)
	}

	// Write to z
	byteOrder.Uint256_indirect(input.Bytes(), (*[4]uint64)(z))
	return
}

func (z *Uint256) Deserialize_Bytes(input []byte, byteOrder FieldElementEndianness) (bytesRead int, err bandersnatchErrors.DeserializationError) {

	if len(input) < 32 {
		panic("Not handled yet") // TODO
	}

	// Write to z
	byteOrder.Uint256_indirect(input, (*[4]uint64)(z))
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
		err = errorsWithData.NewErrorWithData_struct(ErrPrefixLengthInvalid, "", &bandersnatchErrors.ReadErrorData{
			PartialRead:  false,
			BytesRead:    0,
			ActuallyRead: nil, // We do not even try to read
		})
		// Should we panic(err) ???
		return
	}

	// We read all input into buf first, because we don't want to touch z on intermediate IO errors.
	var errPlain error
	var buf [32]byte // := make([]byte, 32)
	bytesRead, errPlain = io.ReadFull(input, buf[:])
	if errPlain != nil {
		err = errorsWithData.AddDataToError_struct(errPlain, &bandersnatchErrors.ReadErrorData{
			PartialRead:  bytesRead != 0 && bytesRead != 32,
			BytesRead:    bytesRead,
			ActuallyRead: buf[0:bytesRead],
		})
		return
	}

	// Write to z
	byteOrder.Uint256_array(&buf, (*[4]uint64)(z))

	// read out the top prefixLength many bits.
	prefix = common.PrefixBits(z[3] >> (64 - prefixLength))

	// clear those bits from z
	var bitmask_remaining uint64 = 0xFFFFFFFF_FFFFFFFF >> prefixLength
	z[3] &= bitmask_remaining

	return
}

func (z *Uint256) DeserializeAndGetPrefix_Buffer(input *bytes.Buffer, prefixLength uint8, byteOrder FieldElementEndianness) (bytesRead int, prefix common.PrefixBits, err bandersnatchErrors.DeserializationError) {
	if prefixLength > common.MaxLengthPrefixBits { // prefixLength > 8
		err = errorsWithData.NewErrorWithData_struct(ErrPrefixLengthInvalid, "", &bandersnatchErrors.ReadErrorData{
			PartialRead:  false,
			BytesRead:    0,
			ActuallyRead: nil, // We do not even try to read
		})
		// Should we panic(err) ???
		return
	}

	// Optimization: Instead of copying the data into buf, we check for the correct size and then read directly from the underlying wrapped []byte via input.Bytes()

	// check that the input has sufficient size
	if inputSize := input.Len(); inputSize < 32 {
		// If the input does not have sufficent size, we actually read it all (draining the buffer) and report an error (This is mostly to be consistent with the general method)
		// For simplicity, we just call the general function, as we don't care about speed on error.
		return z.DeserializeAndGetPrefix(input, prefixLength, byteOrder)
	}

	// Write to z
	byteOrder.Uint256_indirect(input.Bytes(), (*[4]uint64)(z))

	// read out the top prefixLength many bits.
	prefix = common.PrefixBits(z[3] >> (64 - prefixLength))

	// clear those bits from z
	var bitmask_remaining uint64 = 0xFFFFFFFF_FFFFFFFF >> prefixLength
	z[3] &= bitmask_remaining

	return
}

func (z *Uint256) DeserializeAndGetPrefix_Bytes(input []byte, prefixLength uint8, byteOrder FieldElementEndianness) (bytesRead int, prefix common.PrefixBits, err bandersnatchErrors.DeserializationError) {
	if prefixLength > common.MaxLengthPrefixBits { // prefixLength > 8
		err = errorsWithData.NewErrorWithData_struct(ErrPrefixLengthInvalid, "", &bandersnatchErrors.ReadErrorData{
			PartialRead:  false,
			BytesRead:    0,
			ActuallyRead: nil, // We do not even try to read
		})
		// Should we panic(err) ???
		return
	}

	if len(input) < 32 {
		panic("Not handled yet") // TODO
	}

	// Write to z
	byteOrder.Uint256_indirect(input, (*[4]uint64)(z))
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
		err = errorsWithData.AddDataToError_struct(errPlain, &bandersnatchErrors.ReadErrorData{
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
