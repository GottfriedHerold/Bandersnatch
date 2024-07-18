package fieldElements

import (
	"bytes"
	"io"
	"math/bits"
	"reflect"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/common"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/errorsWithData"
	"github.com/GottfriedHerold/Bandersnatch/internal/errorconsts"
)

// This file is part of the fieldElements package. See the documentation of field_element.go for general remarks.

// This file contains the code used to serialize Uint256s. Serialization of field elements x works (usually) via conversion to this.
// The reason is that we serialize field elements in "plain", non-Montgomery format and do not want the serialization format be dependent
// on the field element type used.

// Serialize(output, byteOrder) serializes the receiver to output. byteOrder should be [BigEndian] or [LittleEndian] and refers to the ordering of bytes in the output.
//
// The return values are the actual number of bytes written and a potential error (such as io errors).
// If no error happened, err == nil. In that case we are guaranteed that bytes_written == 32.
//
// There are special-cased methods [Serialize_Buffer] and [Serialize_Bytes] for essentially the same functionality for writing to a [bytes.Buffer] and []byte.
// (These are orders of magnitude faster because of the way interfaces in Go work and how they interact with escape analysis.)
func (z *Uint256) Serialize(output io.Writer, byteOrder FieldElementEndianness) (bytesWritten int, err common.SerializationError) {

	var errPlain error
	var buf [32]byte // = make([]byte, 32)
	byteOrder.PutUint256_array(&buf, (*[4]uint64)(z))
	bytesWritten, errPlain = output.Write(buf[:]) // Note: because output is an interface, this causes escape analysis to fail, so buf is heap-allocated.
	if errPlain != nil {
		// NOTE: These two calls to NewErrorWithData_struct could be consolidated for efficiency.
		// The downside is that we would need to select %w vs. $w manually by checking errPlain's type and
		// we could not use the NewIntermediateWriteErrorData convenience function.

		err, _ = errorsWithData.NewErrorWithData_struct(errPlain, "", errorconsts.NewIntermediateWriteErrorData(bytesWritten, 32))
		err, _ = errorsWithData.NewErrorWithData_map[common.WriteErrorData](err,
			ErrorPrefix+"call to Serialize with receiver ${ValueType} with value ${Value} and io.Writer of type ${WriterType} failed after writing %{BytesWritten} bytes with the following error:\n$w",
			errorsWithData.ParamMap{
				"Value":      *z,
				"ValueType":  "Uint256",
				"WriterType": reflect.TypeOf(output)},
		)
	}
	return
}

// Serialize_Buffer performs the same functionality as [Serialize], but with output of concrete type [*bytes.Buffer].
//
// Due to known issues with Go's escape analysis, this is an order of magnitude more efficient than the general [Serialize].
// On failure, this method panics (because that is what [bytes.Buffer] does), so the return values are guaranteed to be (32, nil).
//
// Note that the only way for this to fail really is running out of memory / the buffer exceeding a limit controlled by the Go runtime.
func (z *Uint256) Serialize_Buffer(output *bytes.Buffer, byteOrder FieldElementEndianness) (bytesWritten int, err common.SerializationError) {
	var buf [32]byte // = make([]byte, 32)
	byteOrder.PutUint256_array(&buf, (*[4]uint64)(z))

	// bytes.Buffer's Write method is guaranteed to never return an error. It panics instead (if out-of-memory, e.g.)
	// So we don't need to handle errors here.
	bytesWritten, _ = output.Write(buf[:])
	return // always (32, nil) according to bytes.Buffer.Write
}

// Serialize_Bytes performs the same functionality as [Serialize], but is special-cased for writing to a []byte slice.
// Given (output, byteOrder), it serializes the receiver to output, which must have len(output) >= 32; note that we check output's length, not capacity.
// byteOrder should be [BigEndian] or [LittleEndian] and refers to the ordering of bytes in the output.
//
// The return values are the actual number of bytes written (alywas 32 or 0) and a potential error.
// If output does not have sufficient length, returns errors wrapping [ErrEmptyByteSlice] (if len(output)==0) or [ErrTooSmallByteSlice] (if 0<len(output)<32).
// For consistency with [Serialize], these errors wrap [io.EOF] respectively [io.UnexpectedEOF].
//
// Note that on error, this function will never write anything to output and bytesWritte is always 0; this behaviour differs from [Serialize].
func (z *Uint256) Serialize_Bytes(output []byte, byteOrder FieldElementEndianness) (bytesWritten int, err common.SerializationError) {

	// handle error cases before writing anything: PutUint256_ptr panics on insufficent slice length (for consistency reasons with binary.ByteOrder's interface)
	// We want Serialize_Bytes to be consistent with the other Serialize methods, so we catch this.
	if len(output) < 32 {
		return handleTooSmallByteSlice_serialize(output, errorsWithData.ParamMap{
			"Value":        *z,
			"ValueType":    "Uint256",
			"NilSlice":     output == nil,
			"ErrorPrefix":  ErrorPrefix,
			"RequiredSize": 32,
		})
	}
	byteOrder.PutUint256_ptr(output, (*[4]uint64)(z))
	bytesWritten = 32
	return
}

// SerializeWithPrefix is used to serialize the given Uint256 with some extra prefix bits squeezed into the most significant byte.
// This function is used for "compressed" serialization of curve points, where we would need to write an extra sign bit.
//
// Usage example: z.SerializeWithPrefix(output, common.MakeBitHeader(PrefixBits(0b01), 2), LittleEndian)
//
// Notably, it performs the following operation:
// Ensure the prefix.prefixLen many most significant bits of z are zero.
// If so, then temporarily replace those bits with prefix.prefixBits and write the resulting 256 bits == 32 bytes to output in byte order determined by byteOrder.
//
// prefix is a [common.BitHeader], meaning it consists of prefixBits and prefixLen. BitHeader specifies that if e.g. prefixLen==3, then prefixBits has at most 3 bits;
// those 3 bits are in lsb position inside prefixBits (e.g. prefixBits = 0b101), even though they end up in higher-order bits during serialization.
// Note that the fields of BitHeader are non-exported (to ensure invariants). Use [common.MakeBitHeader] to generate a [common.BitHeader].
//
// output is an [io.Writer].
// To write to a [bytes.Buffer] or a []byte, you can use the special-cased [SerializeWithPrefix_Buffer] or [SerializeWithPrefix_Bytes] methods.
// (These are much faster than using plain SerializeWithPrefix with a [bytes.Buffer])
//
// byteOrder has type [FieldElementEndianness] and wraps either [binary.BigEndian] or [binary.LittleEndian] from the standard library.
// We provide a [BigEndian], [LittleEndian], [DefaultEndian] constant for this.
// The endiannness choice only affects the order in which the bytes are written to output, NOT the inclusion of a prefix.
// The prefix-inclusion always happens inside the most significant byte, which may then end up in the beginning or end of the output.
//
// SerializeWithPrefix returns the number of actually written bytes and an error (nil if ok).
// If the prefix.prefixLen bits of z are not all zero, we report an error wrapping [ErrPrefixDoesNotFit] and do not write anything to output.
// On other (io-related) errors, we might perform (partial) writes to output.
//
// Possible errors: I/O errors (depending on output's type) and ErrPrefixDoesNotFit, all wrapped.
// If an error occurs, err.GetData().BytesWritten always equals the bytesWritten value directly returned.
func (z *Uint256) SerializeWithPrefix(output io.Writer, prefix BitHeader, byteOrder FieldElementEndianness) (bytesWritten int, err common.SerializationError) {

	prefix_length := prefix.PrefixLen()
	prefix_bits := prefix.PrefixBits()
	// Note: The actual number of leading zeroes might be >64, which this does not pick up.
	// However, if that happens, we don't care.
	if leadingZeroes64 := bits.LeadingZeros64(z[3]); leadingZeroes64 < int(prefix_length) {
		err, _ = errorsWithData.NewErrorWithData_map[common.WriteErrorData](ErrPrefixDoesNotFit, "",
			errorsWithData.ParamMap{
				"Value":         *z,
				"PrefixLength":  prefix_length,
				"LeadingZeroes": leadingZeroes64,
				"ValueType":     "Uint256",
				"ErrorPrefix":   ErrorPrefix})
		return
	}

	zCopy := *z

	// put prefix into msb of zCopy
	zCopy[3] |= (uint64(prefix_bits) << (64 - prefix_length))

	bytesWritten, err = zCopy.Serialize(output, byteOrder)

	// The error message actually refers to the value to be serialized. We need to set it to *z rather than zCopy.
	if err != nil {
		err, _ = errorsWithData.NewErrorWithData_params[common.WriteErrorData](err, "", "Value", *z)
	}

	return
}

// SerializeWithPrefix_Buffer is a specialiazition of [SerializeWithPrefix] for the case where the output is a [*bytes.Buffer].
//
// Due to the way interfaces in Go work, this method is much faster.
//
// Error handling: Due to the fact that writes to [bytes.Buffer] never return an error
// (the only failure cases are running out of memory, in which case we get a panic in the Go runtime), the only potential error reported by this function is
// [ErrPrefixDoesNotFit].
func (z *Uint256) SerializeWithPrefix_Buffer(output *bytes.Buffer, prefix BitHeader, byteOrder FieldElementEndianness) (bytesWritten int, err common.SerializationError) {

	// almost literally the same code as the general version (except for using Serialize_Buffer and not needing to handle error from that.)

	prefix_length := prefix.PrefixLen()
	prefix_bits := prefix.PrefixBits()
	// Note: The actual number of leading zeroes might be >64, which this does not pick up.
	// However, if that happens, we don't care.
	if leadingZeroes64 := bits.LeadingZeros64(z[3]); leadingZeroes64 < int(prefix_length) {
		err, _ = errorsWithData.NewErrorWithData_map[common.WriteErrorData](ErrPrefixDoesNotFit, "",
			errorsWithData.ParamMap{
				"Value":         *z,
				"PrefixLength":  prefix_length,
				"LeadingZeroes": leadingZeroes64,
				"ValueType":     "Uint256",
				"ErrorPrefix":   ErrorPrefix})
		return
	}

	zCopy := *z

	// put prefix into msb of zCopy
	zCopy[3] |= (uint64(prefix_bits) << (64 - prefix_length))

	// NOTE: Serialize_Buffer cannot fail, so we don't need to handle he error.
	bytesWritten, _ = zCopy.Serialize_Buffer(output, byteOrder)

	return // alawys (32, nil) if we get here.
}

// SerializeWithPrefix_Bytes is a specialization of [SerializeWithPrefix] for the case where the output is a []byte.
//
// The output slice must have at least a size of 32 bytes.
// We follow the same conventions as [Serialize_Bytes]:
// for insufficient output length, we return an error wrapping either [ErrEmptyByteSlice] (if len(output)==0) or [ErrTooSmallByteSlice] (0<len(output)<32),
// which in turn wrap [io.EOF] resp. [io.ErrUnexpectedEOF]. We do not write anything in these cases.
//
// Due to the way interfaces in Go work, this method is an order of magnitude faster than [SerializeWithPrefix].
func (z *Uint256) SerializeWithPrefix_Bytes(output []byte, prefix BitHeader, byteOrder FieldElementEndianness) (bytesWritten int, err common.SerializationError) {

	prefix_length := prefix.PrefixLen()
	prefix_bits := prefix.PrefixBits()
	// Note: The actual number of leading zeroes might be >64, which this does not pick up.
	// However, if that happens, we don't care.
	if leadingZeroes64 := bits.LeadingZeros64(z[3]); leadingZeroes64 < int(prefix_length) {
		err, _ = errorsWithData.NewErrorWithData_params[common.WriteErrorData](ErrPrefixDoesNotFit, "",
			"Value", *z,
			"ValueType", "Uint256",
			"PrefixLength", prefix_length,
			"LeadingZeroes", leadingZeroes64)
		return
	}

	zCopy := *z

	// put prefix into msb of low_endian_words
	zCopy[3] |= (uint64(prefix_bits) << (64 - prefix_length))

	bytesWritten, err = zCopy.Serialize_Bytes(output, byteOrder)

	// The error message actually refers to the value to be serialized. We need to set it to *z rather than zCopy.
	if err != nil {
		err, _ = errorsWithData.NewErrorWithData_params[common.WriteErrorData](err, "", "Value", *z)
	}

	return
}

// Deserialize(input, byteOrder) deserializes from input, reading 32 bytes from it and interpreting it as an Uint256 according to byteOrder.
// The result is stored in the receiver. byteOrder should be either [BigEndian], [LittleEndian] or [DefaultEndian] and relates to the order of bytes in input.
//
// If any error occurs, *z is not modified.
// All errors returned from this function are I/O errors, so have the IoError flag set.
// For insufficient data in the reader, the returned error wraps [io.EOF] resp. [io.ErrUnexpectedEOF] if input was empty resp. too small.
//
// For input of type [bytes.Buffer] or to deserialize from a []byte, we have more efficienct, special-cased methods
// [Derserialize_Buffer] resp. [Deserialize_Bytes]
func (z *Uint256) Deserialize(input io.Reader, byteOrder FieldElementEndianness) (bytesRead int, err common.DeserializationError) {
	// We read all input into buf first, because we don't want to touch z on intermediate IO errors.
	var errPlain error
	var buf [32]byte
	bytesRead, errPlain = io.ReadFull(input, buf[:])
	if errPlain != nil {
		bufCopy := buf // to avoid allocating buf on the heap in the happy case.
		err, _ = errorsWithData.NewErrorWithData_params[common.ReadErrorData](errPlain, "",
			"PartialRead", bytesRead != 0 && bytesRead != 32,
			"BytesRead", bytesRead,
			"ActuallyRead", bufCopy[0:bytesRead],
			"IoError", true,
			"ValueType", "Uint256",
		)
		return
	}

	// Write from buf to z
	byteOrder.Uint256_array(&buf, (*[4]uint64)(z))

	return
}

// Deserialize_Buffer is a special-cased version of [Deserialize] for input of type [*bytes.Buffer].
//
// This version is equivalent to, but more efficient than the general version.
//
// The error behaviour is equivalent to [Deserialize]: if input is empty resp. too small, we return an
// error wrapping [io.EOF] resp. [io.ErrUnexpectedEOF] and empty the buffer. We do not change z in this case.
//
// Note that these are the only potential errors; other failure conditions (such as out-of-memory) cause [bytes.Buffer] to panic.
//
// Calling this with input==nil also causes a panic.
func (z *Uint256) Deserialize_Buffer(input *bytes.Buffer, byteOrder FieldElementEndianness) (bytesRead int, err common.DeserializationError) {

	// Check for errors. The only error case is if the buffer has insufficient size.
	// We avoid io.ReadFull here (to make sure we don't trigger any function pointer calls).
	//
	// NOTE: We actually "read" from input (or rather, call input.Reset() to drain it)
	// in order to match the behaviour of the general Deserialize function (including its error reporting).
	if input.Len() < 32 {
		return handleTooSMallBuffer(input, errorsWithData.ParamMap{
			"ValueType":   "Uint256",
			"ErrorPrefix": ErrorPrefix,
		})
	}

	// Otherwise, Write to z directly. This cannot fail.
	byteOrder.Uint256_indirect(input.Next(32), (*[4]uint64)(z))
	return 32, nil
}

// Deserialize_Bytes is a equivalent to [Deserialize], but reads from a byte slice.
//
// This is more efficient than wrapping the byte slice in a [bytes.Buffer] and using either [Deserialize] or [Deserialize_Buffer].
// Also, note that Deserialize_Bytes has no notion of "consuming" its inputs. If the input is larger than 32 bytes, we ignore the remaining bytes.
//
// If the input silce is too small or nil, we return an error wrapping [ErrTooSmalBytesSlice] or [ErrEmptyByteSlice] as appropriate.
// We do not modify *z on error.
func (z *Uint256) Deserialize_Bytes(input []byte, byteOrder FieldElementEndianness) (bytesRead int, err common.DeserializationError) {

	// handle the (only) error case first:
	if len(input) < 32 {
		return handleTooSmallByteSlice_deserialize(input, errorsWithData.ParamMap{
			"ValueType":    "Uint256",
			"RequiredSize": 32,
			"ErrorPrefix":  ErrorPrefix,
		})
	}

	// Write to z. This cannot fail.
	byteOrder.Uint256_indirect(input, (*[4]uint64)(z))
	bytesRead = 32
	return
}

// var uint256Type = utils.TypeOfType[Uint256]()

// DeserializeAndGetPrefix is an inverse to [SerializeWithPrefix]. It reads a 32*8 bit number from input in byte order determined by byteOrder;
// The prefixLength many most significant bits of the resulting number are returned in prefix, the remaining bits are interpreted and stored into the Uint256 z.
// (This implies the prefixLength many most significant bits of z will always be zero after a successful read)
//
// As with [SerializeWithPrefix], the prefix bits are returned in the lower-order bits (i.e. shifted) inside the 8-bit prefix value, even though they originally belonged to the most significant bits inside the most significant byte of the input.
// prefixLength can be at most 8.
//
// On error, we return a non-nil error in err and do not modify z. The returned prefix is meaningless on error.
//
// possible errors: errors wrapping [ErrPrefixLengthInvalid], I/O errors
// The error data's ActuallyRead and BytesRead are guaranteed to contain the raw bytes and their number that were read;
// ActuallyRead is nil if no read attempt was made due to invalid function arguments.
//
// We have special-cased methods [DeserializeAndGetPrefix_Buffer] and [DeserializeAndGetPrefix_Bytes] for inputs of type bytes.Buffer and []byte. These are more efficient.
func (z *Uint256) DeserializeAndGetPrefix(input io.Reader, prefixLength uint8, byteOrder FieldElementEndianness) (bytesRead int, prefix common.PrefixBits, err common.DeserializationError) {
	if prefixLength > common.MaxLengthPrefixBits { // prefixLength > 8
		err, _ = errorsWithData.NewErrorWithData_params[common.ReadErrorData](errPrefixLengthInvalid_Deserialize, "",
			"ValueType", "Uint256 and prefix",
			"PrefixLength", prefixLength,
		)
		// Should we panic(err) ???
		return
	}

	// We read all input into buf first, because we don't want to touch z on intermediate I/O errors.
	var errPlain error
	var buf [32]byte // := make([]byte, 32)
	bytesRead, errPlain = io.ReadFull(input, buf[:])
	if errPlain != nil {
		bufCopy := buf // copy to avoid buf escaping to the heap here.
		err, _ = errorsWithData.NewErrorWithData_params[common.ReadErrorData](errPlain, "",
			"PartialRead", bytesRead != 0 && bytesRead != 32,
			"BytesRead", bytesRead,
			"ActuallyRead", bufCopy[0:bytesRead],
			"IoError", true,
			"ValueType", "Uint256 and prefix",
		)
		return
	}

	// Write from buf to z
	byteOrder.Uint256_array(&buf, (*[4]uint64)(z))

	// read out the top prefixLength many bits.
	prefix = common.PrefixBits(z[3] >> (64 - prefixLength))

	// clear those bits from z
	var bitmask_remaining uint64 = 0xFFFFFFFF_FFFFFFFF >> prefixLength
	z[3] &= bitmask_remaining

	return
}

// DeserializeAndGetPrefix_Buffer is the special-cased version [DeserializeAndGetPrefix] for input of type [*bytes.Buffer].
//
// It is more efficient than the general case.
//
// Note that it has the same behaviour under errors as [DeserializeAndGetPrefix]
func (z *Uint256) DeserializeAndGetPrefix_Buffer(input *bytes.Buffer, prefixLength uint8, byteOrder FieldElementEndianness) (bytesRead int, prefix common.PrefixBits, err common.DeserializationError) {
	if prefixLength > common.MaxLengthPrefixBits { // prefixLength > 8
		err, _ = errorsWithData.NewErrorWithData_params[common.ReadErrorData](errPrefixLengthInvalid_Deserialize, "",
			"ValueType", "Uint256 and prefix",
			"PrefixLength", prefixLength,
		)
		return
	}

	// check that the input has sufficient size. If not, mimick the behaviour of DeserializeAndGetPrefix.
	if input.Len() < 32 {
		bytesRead, err = handleTooSMallBuffer(input, errorsWithData.ParamMap{
			"ValueType":   "Uint256 and prefix",
			"ErrorPrefix": ErrorPrefix,
		})
		// prefix is zero-initialized.
		return
	}

	// NOTE: We cannot fail if we get here.

	// Optimization: Instead of copying the data into buf, we read directly from the underlying wrapped []byte via Next().

	// Write to z directly.
	byteOrder.Uint256_indirect(input.Next(32), (*[4]uint64)(z))
	bytesRead = 32

	// read out the top prefixLength many bits.
	prefix = common.PrefixBits(z[3] >> (64 - prefixLength))

	// clear those bits from z
	var bitmask_remaining uint64 = 0xFFFFFFFF_FFFFFFFF >> prefixLength
	z[3] &= bitmask_remaining

	return
}

// DeserializeAndGetPrefix_Bytes is the variant of [DeserializeAndGetPrefix] that can read directly from input of type []byte.
//
// The error behaviour is similar to [DeserializeAndGetPrefix]; possible errors are errors wrapping
//
// [ErrPrefixLengthInvalid], [ErrTooSmalBytesSlice] or [ErrEmptyByteSlice] if either the prefixLenght or input were bade.
// On error, we do not modify *z and the returned prefix is meaningless.
func (z *Uint256) DeserializeAndGetPrefix_Bytes(input []byte, prefixLength uint8, byteOrder FieldElementEndianness) (bytesRead int, prefix common.PrefixBits, err common.DeserializationError) {
	if prefixLength > common.MaxLengthPrefixBits { // prefixLength > 8
		err, _ = errorsWithData.NewErrorWithData_params[common.ReadErrorData](errPrefixLengthInvalid_Deserialize, "",
			"ValueType", "Uint256 and prefix",
			"PrefixLength", prefixLength,
		)
		return
	}

	if len(input) < 32 {
		bytesRead, err = handleTooSmallByteSlice_deserialize(input, errorsWithData.ParamMap{
			"ValueType":    "Uint256 and Prefix",
			"RequiredSize": 32,
			"ErrorPrefix":  ErrorPrefix,
		})
		return
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
// If the prefix is not present, we return an error wrapping [ErrPrefixMismatch].
// On any error, we do not write to z.
//
// NOTE: On error, err's BytesRead and ActuallyRead accurately reflect what and how much was read by this method.
// NOTE2: In the big endian case, we might either read 32 or only read 1 byte (which contains the prefix) in case of a prefix-mismatch.
// This is an implementation detail and the user is advised to check the returned bytesRead.
func (z *Uint256) DeserializeWithExpectedPrefix(input io.Reader, expectedPrefix BitHeader, byteOrder FieldElementEndianness) (bytesRead int, err common.DeserializationError) {
	// var fieldElementBuffer bsFieldElement_64
	var buf [32]byte // for receiving the input of io.ReadFull

	expectedPrefixLength := expectedPrefix.PrefixLen()
	expectedPrefixBits := expectedPrefix.PrefixBits()

	var errIO error
	bytesRead, errIO = io.ReadFull(input, buf[0:32])
	if errIO != nil {
		bufCopy := buf // escape anlysis: to avoid heap-allocating buf.
		err, _ = errorsWithData.NewErrorWithData_map[common.ReadErrorData](errIO, "", errorsWithData.ParamMap{
			"PartialRead":    bytesRead != 0 && bytesRead != 32,
			"BytesRead":      bytesRead,
			"ActuallyRead":   bufCopy[0:bytesRead],
			"IoError":        true,
			"ValueType":      "Uint256 with expected prefix",
			"ExpectedPrefix": byte(expectedPrefixBits),
		})
		return
	}

	// NOTE: We could check the prefix in buf directly, which would avoid this copy.
	// However, since FieldElementEndianness may be changed to an interface type at some point,
	// this would require a GetMSB method.
	// zTemp = byteOrder.Uint256(buf[:])
	// endianness and IO no longer play a role. We have everything in zTemp now.
	// readPrefixBits := common.PrefixBits(zTemp[3] >> (64 - expectedPrefixLength))

	// Implemented a GetMSB method:
	readPrefixBits := common.PrefixBits(byteOrder.GetMSB(&buf)) >> (8 - expectedPrefixLength)

	if readPrefixBits != expectedPrefixBits {
		bufCopy := buf
		err, _ = errorsWithData.NewErrorWithData_map[common.ReadErrorData](ErrPrefixMismatch, "", errorsWithData.ParamMap{
			"PartialRead":    false,
			"BytesRead":      bytesRead, // == 32
			"ActuallyRead":   bufCopy[0:bytesRead],
			"IoError":        false,
			"Prefix":         byte(readPrefixBits),
			"ExpectedPrefix": byte(expectedPrefixBits),
			"ValueType":      "Uint256",
		})
		return // bytesRead == 32
	}

	byteOrder.Uint256_array(&buf, (*[4]uint64)(z))
	// remove prefix from read data z.
	var bitmask_remaining uint64 = 0xFFFFFFFF_FFFFFFFF >> expectedPrefixLength
	z[3] &= bitmask_remaining

	return
}

// DeserializeWithExpectedPrefix_Buffer is the variant of [DeserializeWithExpectedPrefix] for the special case where input is of type [*bytes.Buffer].
//
// It reads up to 32 bytes from the buffer, interpreting them as a 256-bit number according to byteOrder. It then checks whether the most significant expectedPrefix.PrefixLen() many bits
// exactly match expectedPrefix.PrefixBits(). If not, we return an error.
// If the prefix matches, we strip the prefix from the read number and set z to this stripped number.
//
// If any error occurs, this function does not modify z.
func (z *Uint256) DeserializeWithExpectedPrefix_Buffer(input *bytes.Buffer, expectedPrefix BitHeader, byteOrder FieldElementEndianness) (bytesRead int, err common.DeserializationError) {
	if input.Len() < 32 {
		return handleTooSMallBuffer(input, errorsWithData.ParamMap{
			"ValueType":   "Uint256 with expected prefix",
			"ErrorPrefix": ErrorPrefix,
		})
	}
	// read bytes; note that input.Next avoids copying: the returned slice references the internal storage of input.
	buf := input.Next(32)
	bytesRead = 32

	// Check prefix:
	expectedPrefixLength := expectedPrefix.PrefixLen()
	expectedPrefixBits := expectedPrefix.PrefixBits()
	readPrefixBits := common.PrefixBits(byteOrder.GetMSB((*[32]byte)(buf))) >> (8 - expectedPrefixLength)

	// handle wrong prefix
	if readPrefixBits != expectedPrefixBits {
		var bufCopy [32]byte
		copy(bufCopy[:], buf[:])
		err, _ = errorsWithData.NewErrorWithData_map[common.ReadErrorData](ErrPrefixMismatch, "", errorsWithData.ParamMap{
			"PartialRead":    false,
			"BytesRead":      bytesRead, // == 32
			"ActuallyRead":   bufCopy[0:bytesRead],
			"IoError":        false,
			"Prefix":         byte(readPrefixBits),
			"ExpectedPrefix": byte(expectedPrefixBits),
			"ValueType":      "Uint256 with expected prefix",
			"ErrorPrefix":    ErrorPrefix,
		})
		return // bytesRead == 32
	}

	// no error if we get here:
	byteOrder.Uint256_indirect(buf, (*[4]uint64)(z))
	var bitmask_remaining uint64 = 0xFFFFFFFF_FFFFFFFF >> expectedPrefixLength
	z[3] &= bitmask_remaining
	return

}

// DeserializeWithExpectedPrefix_Bytes is the variant of [DeserializeWithExpectedPrefix] for the special case where input is a byte slice.
func (z *Uint256) DeserializeWithExpectedPrefix_Bytes(input []byte, expectedPrefix BitHeader, byteOrder FieldElementEndianness) (bytesRead int, err common.DeserializationError) {

	if len(input) < 32 {
		return handleTooSmallByteSlice_deserialize(input, errorsWithData.ParamMap{
			"ValueType":    "Uint256 with expected prefix",
			"Errorprefix":  ErrorPrefix,
			"RequiredSize": 32,
		})
	}
	bytesRead = 32

	// Check prefix:
	expectedPrefixLength := expectedPrefix.PrefixLen()
	expectedPrefixBits := expectedPrefix.PrefixBits()
	readPrefixBits := common.PrefixBits(byteOrder.GetMSB((*[32]byte)(input))) >> (8 - expectedPrefixLength)

	// handle wrong prefix
	if readPrefixBits != expectedPrefixBits {
		var bufCopy [32]byte
		copy(bufCopy[:], input[0:32])
		err, _ = errorsWithData.NewErrorWithData_map[common.ReadErrorData](ErrPrefixMismatch, "", errorsWithData.ParamMap{
			"PartialRead":    false,
			"BytesRead":      bytesRead, // == 32
			"ActuallyRead":   bufCopy[0:bytesRead],
			"IoError":        false,
			"Prefix":         byte(readPrefixBits),
			"ExpectedPrefix": byte(expectedPrefixBits),
			"ValueType":      "Uint256 with expected prefix",
			"ErrorPrefix":    ErrorPrefix,
		})
		return // bytesRead == 32
	}

	// no error if we get here:
	byteOrder.Uint256_indirect(input[0:32], (*[4]uint64)(z))
	var bitmask_remaining uint64 = 0xFFFFFFFF_FFFFFFFF >> expectedPrefixLength
	z[3] &= bitmask_remaining
	return

}
