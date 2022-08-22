package pointserializer

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/bandersnatchErrors"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/errorsWithData"
	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

// headerDeserializer and headerSerializer are abstractions used to (de)serialize headers / footers for multiple points.
//
// When (de)serializing a single point, we call
// - (de)serializeSinglePointHeader
// - actual point (de)serialization
// - (de)deserializeSinglePointFooter
//
// When (de)serializing a slice (this includes its length in-band), we call
// - (de)serializeGlobalSliceHeader : This reads/writes the slice length
// - for each element:
//    -- (de)serializePerPointHeader
//    -- actual point (de)serialization
//    -- (de)serializePerPointFooter
// - (de)serializeGlobalSliceFooter
//
// NOTE: While the return type is int (for consistency with the standard library), we promise that all bytes_read / bytes_written fit into an int32.
// Too big reads/write will panic. This is to ensure consistency for 32-bit and 64-bit users.
type headerDeserializer interface {
	deserializeGlobalSliceHeader(input io.Reader) (bytes_read int, size int32, err bandersnatchErrors.DeserializationError)
	deserializeGlobalSliceFooter(input io.Reader) (bytes_read int, err bandersnatchErrors.DeserializationError)
	deserializeSinglePointHeader(input io.Reader) (bytes_read int, err bandersnatchErrors.DeserializationError)
	deserializeSinglePointFooter(input io.Reader) (bytes_read int, err bandersnatchErrors.DeserializationError)
	deserializePerPointHeader(input io.Reader) (bytes_read int, err bandersnatchErrors.DeserializationError)
	deserializePerPointFooter(input io.Reader) (bytes_read int, err bandersnatchErrors.DeserializationError)
	SinglePointHeaderOverhead() int32                                         // returns the size taken up by headers and footers for single-point
	MultiPointHeaderOverhead(numPoints int32) (size int32, overflowErr error) // returns the size taken up by headers and footers for slice of given size. error is set on int32 overflow.
	RecognizedParameters() []string
	HasParameter(parameterName string) bool
}

// these are the parameter names accepted by simpleHeaderDeserializer. This is returned by RecognizedParameters()
var headerSerializerParams = []string{"GlobalSliceHeader", "GlobalSliceFooter", "SinglePointHeader", "SinglePointFooter", "PerPointHeader", "PerPointFooter"}

// headerSerializer extends headerDeserializer by also providing serialization routines.
type headerSerializer interface {
	headerDeserializer
	serializeGlobalSliceHeader(output io.Writer, size int32) (bytes_written int, err bandersnatchErrors.SerializationError)
	serializeGlobalSliceFooter(output io.Writer) (bytes_written int, err bandersnatchErrors.SerializationError)
	serializeSinglePointHeader(output io.Writer) (bytes_written int, err bandersnatchErrors.SerializationError)
	serializeSinglePointFooter(output io.Writer) (bytes_written int, err bandersnatchErrors.SerializationError)
	serializePerPointHeader(output io.Writer) (bytes_written int, err bandersnatchErrors.SerializationError)
	serializePerPointFooter(output io.Writer) (bytes_written int, err bandersnatchErrors.SerializationError)
}

const simpleHeaderSliceLengthOverhead = 4 // size taken up in bytes for serializing slice lengths.

// simpleHeaderDeserializer is a headerDeserializer where all headers are just constant []byte's and the size of slices is written into 4 bytes after the slice header.
//
// Note that the zero value is invalid and does not pass Validate() due to sliceSizeEndianness being nil.
type simpleHeaderDeserializer struct {
	headerSlice            []byte
	headerPerCurvePoint    []byte
	headerSingleCurvePoint []byte
	footerSingleCurvePoint []byte
	footerPerCurvePoint    []byte
	footerSlice            []byte

	sliceSizeEndianness binary.ByteOrder // endianness for writing the size of slices.
}

// simpleHeaderSerializer extends simpleHeaderDeserializer by also providing write methods.
//
// As with simpleHeaderDeserializer, the zero value of this type is invalid.
type simpleHeaderSerializer struct {
	simpleHeaderDeserializer
}

var (
	// basicSimpleHeaderDeserializer is a valid simpleHeaderDeserializer with trivial headers/footers. Note that nil []byte-slices are changed by init() below
	basicSimpleHeaderDeserializer simpleHeaderDeserializer = simpleHeaderDeserializer{sliceSizeEndianness: binary.LittleEndian}
	// basicSimpleHeaderSerializer is a valid simpleHeaderSerializer with trivial headers/footers. Note that nil []byte-slices are changed by init() below
	basicSimpleHeaderSerializer simpleHeaderSerializer = simpleHeaderSerializer{simpleHeaderDeserializer: *basicSimpleHeaderDeserializer.Clone()}
)

// Validate the above; this also changes nil entries to empty []byte
func init() {
	basicSimpleHeaderDeserializer.Validate()
	basicSimpleHeaderSerializer.Validate()
}

// RecognizedParameters returns a list of all parameter names that header (de)serializers support for querying and modifying.
func (*simpleHeaderDeserializer) RecognizedParameters() []string {
	return headerSerializerParams // defined above. Note that this is essentiall a global constant not supposed to be modified.
}

// HasParameter checks whether a given parameter is supported for this type
func (shd *simpleHeaderDeserializer) HasParameter(parameterName string) bool {
	return utils.ElementInList(parameterName, headerSerializerParams, normalizeParameter)
}

// Clone returns an independent copy of the receivedHeaderDeserializer (as a pointer)
func (shd *simpleHeaderDeserializer) Clone() *simpleHeaderDeserializer {
	var ret simpleHeaderDeserializer

	// We copy the byte slices to avoid aliasing. This is actually not needed, since headerDeserializers are immutable.
	// While Clone is used internally to create modified copies (by first cloning and then changing parameters), the latter change
	// does not modify the existing slice.
	ret.headerSlice = copyByteSlice(shd.headerSlice)
	ret.headerPerCurvePoint = copyByteSlice(shd.headerPerCurvePoint)
	ret.headerSingleCurvePoint = copyByteSlice(shd.headerSingleCurvePoint)
	ret.footerSlice = copyByteSlice(shd.footerSlice)
	ret.footerPerCurvePoint = copyByteSlice(shd.footerPerCurvePoint)
	ret.footerSingleCurvePoint = copyByteSlice(shd.footerSingleCurvePoint)

	// Copy the endianness. While this is an interface possibly holding a pointer, we do not expect this to be modifyable.
	ret.sliceSizeEndianness = shd.sliceSizeEndianness
	return &ret
}

// Clone returns an independent copy of the receivedHeaderSerializer (as a pointer)
func (shs *simpleHeaderSerializer) Clone() *simpleHeaderSerializer {
	var ret simpleHeaderSerializer
	ret.simpleHeaderDeserializer = *shs.simpleHeaderDeserializer.Clone()
	return &ret
}

// fixNilEntries replaces any nil []byte entries by length-0 []bytes.
func (shd *simpleHeaderDeserializer) fixNilEntries() {
	for _, arg := range []*[]byte{&shd.headerSlice, &shd.headerPerCurvePoint, &shd.headerSingleCurvePoint, &shd.footerPerCurvePoint, &shd.footerSlice, &shd.footerPerCurvePoint, &shd.footerSingleCurvePoint} {
		if *arg == nil {
			*arg = make([]byte, 0)
		}
	}
}

// this must be called after all setters.
// (Note: We call this from the setters, but this is actually redundant, as the external caller is actually responsible)

// Validate fixes any nil entries (replacing them by length-0 slices) and ensures that
// relevant overhead lengths fit into int32's
func (shd *simpleHeaderDeserializer) Validate() {
	if shd.sliceSizeEndianness == nil {
		panic(ErrorPrefix + "serializer does not have endianness set to serialize the length of slices")
	}
	shd.fixNilEntries()
	l1 := len(shd.headerSingleCurvePoint)
	l2 := len(shd.footerSingleCurvePoint)
	if l1 > math.MaxInt32 {
		panic(fmt.Errorf("bandersnatch / serialization: serializer has single-point header length of %v, which exceeds MaxInt32", l1))
	}
	if l2 > math.MaxInt32 {
		panic(fmt.Errorf("bandersnatch / serialization: serializer has single-point footer length of %v, which exceeds MaxInt32", l2))
	}
	var sum int64 = int64(l1) + int64(l2)
	if sum > math.MaxInt32 {
		panic(fmt.Errorf("bandersnatch / serialization: serializer has single-point overhead of %v, which exceeds MaxInt32", sum))
	}
	l1 = len(shd.headerSlice)
	l2 = len(shd.footerSlice)
	if l1 > math.MaxInt32 {
		panic(fmt.Errorf("bandersnatch / serialization: serializer has slice serialization header of length %v, which exceeds MaxInt32", l1))
	}
	if l2 > math.MaxInt32 {
		panic(fmt.Errorf("bandersnatch / serialization: serializer has slice serialization footer of length %v, which exceeds MaxInt32", l2))
	}
	sum = int64(l1) + int64(l2) + simpleHeaderSliceLengthOverhead
	if sum > math.MaxInt32 {
		panic(fmt.Errorf("bandersnatch / serialization: serializer has fixed overhead for slice serialization of length %v, which exceeds MaxInt32", sum))
	}
	l1 = len(shd.headerPerCurvePoint)
	l2 = len(shd.footerPerCurvePoint)
	if l1 > math.MaxInt32 {
		panic(fmt.Errorf("bandersnatch / serialization: serializer has per-point header for slice serialization of length %v, which exceeds MaxInt32", l1))
	}
	if l2 > math.MaxInt32 {
		panic(fmt.Errorf("bandersnatch / serialization: serializer has per-point footer for slice serialization of length %v, which exceeds MaxInt32", l2))
	}
	sum = int64(l1) + int64(l2)
	if sum > math.MaxInt32 {
		panic(fmt.Errorf("bandersnatch / serialization: serializer has per-point overhead for slice serialization of length %v, which exceeds MaxInt32", sum))
	}
}

// NOTE: Getters return a copy, by design. This is because all serializers are read-only.
// The only way for users to modify a serializer is to create a modified copy.
// Returning the contained slice would break that.
//
// Similarly, Setters put a copy of the slice in the

func (shd *simpleHeaderDeserializer) SetGlobalSliceHeader(v []byte) {
	shd.headerSlice = copyByteSlice(v)
	shd.Validate()
}

func (shd *simpleHeaderDeserializer) GetGlobalSliceHeader() []byte {
	return copyByteSlice(shd.headerSlice)
}

func (shd *simpleHeaderDeserializer) deserializeGlobalSliceHeader(input io.Reader) (bytesRead int, size int32, err bandersnatchErrors.DeserializationError) {
	// var err error // might need to declare, because := in consumeExpectRead gives more powerful interface
	bytesRead, errCER := consumeExpectRead(input, shd.headerSlice[:])
	if errCER != nil {
		err = errorsWithData.IncludeGuaranteedParametersInError[bandersnatchErrors.ReadErrorData](errCER, FIELDNAME_PARTIAL_READ, bytesRead != 0)
		return
	}
	var buf [simpleHeaderSliceLengthOverhead]byte
	bytesJustRead, errPlain := io.ReadFull(input, buf[:])
	bytesRead += bytesJustRead // Validate ensures this fits into int32
	if errPlain != nil {
		bandersnatchErrors.UnexpectEOF(&errPlain) // turn io.EOF into io.ErrUnexpectedEOF
		err = errorsWithData.IncludeGuaranteedParametersInError[bandersnatchErrors.ReadErrorData](errPlain, FIELDNAME_PARTIAL_READ, bytesJustRead != simpleHeaderSliceLengthOverhead, FIELDNAME_ACTUALLY_READ, buf[:])
		return
	}

	var sizeUInt32 uint32 = shd.sliceSizeEndianness.Uint32(buf[:])
	if sizeUInt32 > math.MaxInt32 {
		errPlain = errorsWithData.NewErrorWithParameters(bandersnatchErrors.ErrSizeDoesNotFitInt32, "%w. Size read when deserializing was %v{Size}", "Size", sizeUInt32)
		err = errorsWithData.NewErrorWithParametersFromData(errPlain, "%w", &bandersnatchErrors.ReadErrorData{
			PartialRead:  false,
			BytesRead:    bytesJustRead,
			ActuallyRead: buf[:],
		})
		return
	}
	size = int32(sizeUInt32)
	return bytesRead, size, nil
}

func (shs *simpleHeaderSerializer) serializeGlobalSliceHeader(output io.Writer, size int32) (bytesWritten int, err bandersnatchErrors.SerializationError) {
	if size < 0 {
		// this should be unreachable
		panic(fmt.Errorf(ErrorPrefix+"called simpleHeaderSerializer.serializeGlobalSliceHeader with negative size %v", size))
	}
	// var err error
	bytesWritten, errPlain := output.Write(shs.headerSlice[:])
	if errPlain != nil {
		err = errorsWithData.NewErrorWithParametersFromData(errPlain, "%w", &bandersnatchErrors.WriteErrorData{
			BytesWritten: bytesWritten,
			PartialWrite: bytesWritten != 0,
		})
		return
	}

	var buf [simpleHeaderSliceLengthOverhead]byte
	shs.sliceSizeEndianness.PutUint32(buf[:], uint32(size))
	bytesJustWritten, errPlain := output.Write(buf[:])
	bytesWritten += bytesJustWritten // ensureInt32Constrains ensures this fits into int32
	if errPlain != nil {
		bandersnatchErrors.UnexpectEOF(&errPlain)
		err = errorsWithData.NewErrorWithParametersFromData(errPlain, "%w", &bandersnatchErrors.WriteErrorData{
			BytesWritten: bytesJustWritten,
			PartialWrite: bytesJustWritten != simpleHeaderSliceLengthOverhead,
		})
		return
	}
	return
}

func (shd *simpleHeaderDeserializer) SetGlobalSliceFooter(v []byte) {
	shd.footerSlice = copyByteSlice(v)
	shd.Validate()
}

func (shd *simpleHeaderDeserializer) GetGlobalSliceFooter() []byte {
	return copyByteSlice(shd.footerSlice)
}

// addPartialReadInfo is a helper function that just "downcasts" the extra data type for the error
func fixReadErrorType(errIn errorsWithData.ErrorWithGuaranteedParameters[headerRead]) (errOut bandersnatchErrors.DeserializationError) {
	return errorsWithData.AsErrorWithData[bandersnatchErrors.ReadErrorData](errIn)
}

func (shd *simpleHeaderDeserializer) deserializeGlobalSliceFooter(input io.Reader) (bytesRead int, err bandersnatchErrors.DeserializationError) {
	bytesRead, errPlain := consumeExpectRead(input, shd.footerSlice) // Validate ensures bytesRead fits into int32
	err = fixReadErrorType(errPlain)
	return
}

func (shs *simpleHeaderSerializer) serializeGlobalSliceFooter(output io.Writer) (bytesWritten int, err bandersnatchErrors.SerializationError) {
	bytesWritten, err = writeFull(output, shs.footerSlice)
	return
}

func (shd *simpleHeaderDeserializer) SetPerPointHeader(v []byte) {
	shd.headerPerCurvePoint = copyByteSlice(v)
	shd.Validate()
}

func (shd *simpleHeaderDeserializer) GetPerPointHeader() []byte {
	return copyByteSlice(shd.headerPerCurvePoint)
}

func (shd *simpleHeaderDeserializer) deserializePerPointHeader(input io.Reader) (bytesRead int, err bandersnatchErrors.DeserializationError) {
	bytesRead, errPlain := consumeExpectRead(input, shd.headerPerCurvePoint) // Validate ensures bytesRead fits into int32
	err = fixReadErrorType(errPlain)
	return
}

func (shs *simpleHeaderSerializer) serializePerPointHeader(output io.Writer) (bytesWritten int, err bandersnatchErrors.SerializationError) {
	bytesWritten, err = writeFull(output, shs.headerPerCurvePoint)
	return
}

func (shd *simpleHeaderDeserializer) SetPerPointFooter(v []byte) {
	shd.footerPerCurvePoint = copyByteSlice(v)
	shd.Validate()
}

func (shd *simpleHeaderDeserializer) GetPerPointFooter() []byte {
	return copyByteSlice(shd.footerPerCurvePoint)
}

func (shd *simpleHeaderDeserializer) deserializePerPointFooter(input io.Reader) (bytesRead int, err bandersnatchErrors.DeserializationError) {
	bytesRead, errPlain := consumeExpectRead(input, shd.footerPerCurvePoint) // Validate ensures bytesRead fits into int32
	err = fixReadErrorType(errPlain)
	return
}

func (shs *simpleHeaderSerializer) serializePerPointFooter(output io.Writer) (bytesWritten int, err bandersnatchErrors.SerializationError) {
	bytesWritten, err = writeFull(output, shs.footerPerCurvePoint)
	return
}

func (shd *simpleHeaderDeserializer) SetSinglePointHeader(v []byte) {
	shd.headerSingleCurvePoint = copyByteSlice(v)
	shd.Validate()
}

func (shd *simpleHeaderDeserializer) GetSinglePointHeader() []byte {
	return copyByteSlice(shd.headerSingleCurvePoint)
}

func (shd *simpleHeaderDeserializer) deserializeSinglePointHeader(input io.Reader) (bytesRead int, err bandersnatchErrors.DeserializationError) {
	bytesRead, errPlain := consumeExpectRead(input, shd.headerSingleCurvePoint) // Validate ensures bytesRead fits into int32
	err = fixReadErrorType(errPlain)
	return
}

func (shs *simpleHeaderSerializer) serializeSinglePointHeader(output io.Writer) (bytesWritten int, err bandersnatchErrors.SerializationError) {
	bytesWritten, err = writeFull(output, shs.headerSingleCurvePoint)
	return
}

func (shd *simpleHeaderDeserializer) SetSinglePointFooter(v []byte) {
	shd.footerSingleCurvePoint = copyByteSlice(v)
	shd.Validate()
}

func (shd *simpleHeaderDeserializer) GetSinglePointFooter() []byte {
	return copyByteSlice(shd.footerSingleCurvePoint)
}

func (shd *simpleHeaderDeserializer) deserializeSinglePointFooter(input io.Reader) (bytesRead int, err bandersnatchErrors.DeserializationError) {
	bytesRead, errPlain := consumeExpectRead(input, shd.footerSingleCurvePoint) // Validate ensures bytesRead fits into int32
	err = fixReadErrorType(errPlain)
	return
}

func (shs *simpleHeaderSerializer) serializeSinglePointFooter(output io.Writer) (bytesWritten int, err bandersnatchErrors.SerializationError) {
	bytesWritten, err = writeFull(output, shs.footerSingleCurvePoint)
	return
}

func (shd *simpleHeaderDeserializer) SinglePointHeaderOverhead() int32 {
	// Validate ensures this does not overflow int32
	return int32(len(shd.headerSingleCurvePoint) + len(shd.footerSingleCurvePoint))
}

// MultiPointHeaderOverhead returns the size taken up by headers and footers for slice of given size.
// This includes everything except for actually writing the points.
// error is set on int32 overflow.
func (shd *simpleHeaderDeserializer) MultiPointHeaderOverhead(numPoints int32) (ret int32, err error) {
	var ret64 int64
	// shd.fixNilEntries()
	if numPoints < 0 {
		panic(fmt.Errorf(ErrorPrefix+"Querying overhead size for slice (de)serialization for negative length %v", numPoints))
	}
	ret64 = int64(numPoints) * int64(len(shd.headerPerCurvePoint)+len(shd.footerPerCurvePoint)) // both factors are guaranteed to fit into int32, so no overflow here.
	ret64 += simpleHeaderSliceLengthOverhead                                                    // for writing the size
	ret64 += int64(len(shd.headerSlice) + len(shd.footerSlice))                                 // term added is guaranteed to fit into int32
	// NOTE: ret64 is guaranteed to not have overflown an int64, since it is at most (2^31-1) * (2^31-1) + 4 + (2^31-1), which is smaller than 2^63-1
	if ret64 > math.MaxInt32 {
		err = errorsWithData.NewErrorWithParameters(nil, "MultiPointOverhead does not fit into int32, size was %v{Size}", "Size", ret64)
	}
	ret = int32(ret64)
	return
}
