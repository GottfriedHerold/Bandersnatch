package pointserializer

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/bandersnatchErrors"
	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

// TODO: Should header errors be data-carrying wrappers?

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
	deserializeGlobalSliceHeader(input io.Reader) (bytes_read int, size int32, err error)
	deserializeGlobalSliceFooter(input io.Reader) (bytes_read int, err error)
	deserializeSinglePointHeader(input io.Reader) (bytes_read int, err error)
	deserializeSinglePointFooter(input io.Reader) (bytes_read int, err error)
	deserializePerPointHeader(input io.Reader) (bytes_read int, err error)
	deserializePerPointFooter(input io.Reader) (bytes_read int, err error)
	SinglePointHeaderOverhead() int32                                                           // returns the size taken up by headers and footers for single-point
	MultiPointHeaderOverhead(numPoints int32) (int32, *bandersnatchErrors.ErrorWithData[int64]) // returns the size taken up by headers and footers for slice of given size. error is set on int32 overflow.
}

// headerSerializer extends headerDeserializer by also providing serialization routines.

type headerSerializer interface {
	headerDeserializer
	serializeGlobalSliceHeader(output io.Writer, size int32) (bytes_written int, err error)
	serializeGlobalSliceFooter(output io.Writer) (bytes_written int, err error)
	serializeSinglePointHeader(output io.Writer) (bytes_written int, err error)
	serializeSinglePointFooter(output io.Writer) (bytes_written int, err error)
	serializePerPointHeader(output io.Writer) (bytes_written int, err error)
	serializePerPointFooter(output io.Writer) (bytes_written int, err error)
}

const simpleHeaderSliceLengthOverhead = 4 // size taken up in bytes for serializing slice lengths.

// simpleHeaderDeserializer is a headerDeserializer where all headers are just constant []byte's and the size of slices is written into 4 bytes after the slice header.
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
type simpleHeaderSerializer struct {
	simpleHeaderDeserializer
}

func (shd *simpleHeaderDeserializer) Clone() *simpleHeaderDeserializer {
	var ret simpleHeaderDeserializer
	ret.headerSlice = copyByteSlice(shd.headerSlice)
	ret.headerPerCurvePoint = copyByteSlice(shd.headerPerCurvePoint)
	ret.headerSingleCurvePoint = copyByteSlice(shd.headerSingleCurvePoint)
	ret.footerSlice = copyByteSlice(shd.footerSlice)
	ret.footerPerCurvePoint = copyByteSlice(shd.footerPerCurvePoint)
	ret.footerSingleCurvePoint = copyByteSlice(shd.footerSingleCurvePoint)
	ret.sliceSizeEndianness = shd.sliceSizeEndianness
	return &ret
}

func (shs *simpleHeaderSerializer) Clone() *simpleHeaderSerializer {
	var ret simpleHeaderSerializer
	ret.simpleHeaderDeserializer = *shs.simpleHeaderDeserializer.Clone()
	return &ret
}

// fixNilEntries replaces any nil []byte entries by length-0 []bytes.
func (shd *simpleHeaderDeserializer) fixNilEntries() {
	for _, arg := range [][]byte{shd.headerSlice, shd.headerPerCurvePoint, shd.headerSingleCurvePoint, shd.footerPerCurvePoint, shd.footerSlice, shd.footerPerCurvePoint, shd.footerSingleCurvePoint} {
		if arg == nil {
			arg = make([]byte, 0)
		}
	}
}

// this must be called after all setters.

// ensureInt32Constraints fixes any nil entries (replacing them by length-0 slices) and ensures that
// relevant overhead lengths fit into int32's
func (shd *simpleHeaderDeserializer) ensureInt32Constraints() {
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
// The only way for users to modify a serializer is to create a modified copy. Returning the contained slice would break that.

func (shd *simpleHeaderDeserializer) SetGlobalSliceHeader(v []byte) {
	shd.headerSlice = copyByteSlice(v)
	shd.ensureInt32Constraints()
}

func (shd *simpleHeaderDeserializer) GetGlobalSliceHeader() []byte {
	return copyByteSlice(shd.headerSlice)
}

func (shd *simpleHeaderDeserializer) deserializeGlobalSliceHeader(input io.Reader) (bytes_read int, size int32, err error) {
	bytes_read, err = consumeExpectRead(input, shd.headerSlice[:])
	if err != nil {
		return
	}
	var buf [simpleHeaderSliceLengthOverhead]byte
	bytesJustRead, err := io.ReadFull(input, buf[:])
	bytes_read += bytesJustRead // ensureInt32Constrains ensures this fits into int32
	if err != nil {
		utils.UnexpectEOF(&err) // turn io.EOF into io.ErrUnexpectedEOF
		return
	}
	var sizeUInt32 uint32 = shd.sliceSizeEndianness.Uint32(buf[:])
	if sizeUInt32 > math.MaxInt32 {
		err = fmt.Errorf("%w. Size read when deserializing was %v", bandersnatchErrors.ErrSizeDoesNotFitInt32, sizeUInt32)
		return
	}
	size = int32(sizeUInt32)
	return
}

func (shs *simpleHeaderSerializer) serializeGlobalSliceHeader(output io.Writer, size int32) (bytesWritten int, err error) {
	if size < 0 {
		panic(fmt.Errorf("bandersnatch / serializers: called simpleHeaderSerializer.serializeGlobalSliceHeader with negative size %v", size))
	}
	bytesWritten, err = output.Write(shs.headerSlice[:])
	if err != nil {
		return
	}

	var buf [simpleHeaderSliceLengthOverhead]byte
	shs.sliceSizeEndianness.PutUint32(buf[:], uint32(size))
	bytesJustWritten, err := output.Write(buf[:])
	bytesWritten += bytesJustWritten // ensureInt32Constrains ensures this fits into int32
	if err != nil {
		utils.UnexpectEOF(&err)
		return
	}
	return
}

func (shd *simpleHeaderDeserializer) SetGlobalSliceFooter(v []byte) {
	shd.footerSlice = copyByteSlice(v)
	shd.ensureInt32Constraints()
}

func (shd *simpleHeaderDeserializer) GetGlobalSliceFooter() []byte {
	return copyByteSlice(shd.footerSlice)
}

func (shd *simpleHeaderDeserializer) deserializeGlobalSliceFooter(input io.Reader) (bytesRead int, err error) {
	return consumeExpectRead(input, shd.footerSlice) // ensureInt32Constrains ensures bytesRead fits into int32
}

func (shs *simpleHeaderSerializer) serializeGlobalSliceFooter(output io.Writer) (bytesWritten int, err error) {
	return output.Write(shs.footerSlice)
}

func (shd *simpleHeaderDeserializer) SetPerPointHeader(v []byte) {
	shd.headerPerCurvePoint = copyByteSlice(v)
	shd.ensureInt32Constraints()
}

func (shd *simpleHeaderDeserializer) GetPerPointHeader() []byte {
	return copyByteSlice(shd.headerPerCurvePoint)
}

func (shd *simpleHeaderDeserializer) deserializePerPointHeader(input io.Reader) (bytesRead int, err error) {
	return consumeExpectRead(input, shd.headerPerCurvePoint) // ensureInt32Constrains ensures bytesRead fits into int32
}

func (shs *simpleHeaderSerializer) serializePerPointHeader(output io.Writer) (bytesWritten int, err error) {
	return output.Write(shs.headerPerCurvePoint)
}

func (shd *simpleHeaderDeserializer) SetPerPointFooter(v []byte) {
	shd.footerPerCurvePoint = copyByteSlice(v)
	shd.ensureInt32Constraints()
}

func (shd *simpleHeaderDeserializer) GetPerPointFooter() []byte {
	return copyByteSlice(shd.footerPerCurvePoint)
}

func (shd *simpleHeaderDeserializer) deserializePerPointFooter(input io.Reader) (bytesRead int, err error) {
	return consumeExpectRead(input, shd.footerPerCurvePoint) // ensureInt32Constrains ensures bytesRead fits into int32
}

func (shs *simpleHeaderSerializer) serializePerPointFooter(output io.Writer) (bytesWritten int, err error) {
	return output.Write(shs.footerPerCurvePoint)
}

func (shd *simpleHeaderDeserializer) SetSinglePointHeader(v []byte) {
	shd.headerSingleCurvePoint = copyByteSlice(v)
	shd.ensureInt32Constraints()
}

func (shd *simpleHeaderDeserializer) GetSinglePointHeader() []byte {
	return copyByteSlice(shd.headerSingleCurvePoint)
}

func (shd *simpleHeaderDeserializer) deserializeSinglePointHeader(input io.Reader) (bytesRead int, err error) {
	return consumeExpectRead(input, shd.headerSingleCurvePoint) // ensureInt32Constrains ensures bytesRead fits into int32
}

func (shs *simpleHeaderSerializer) serializeSinglePointHeader(output io.Writer) (bytesWritten int, err error) {
	return output.Write(shs.headerSingleCurvePoint)
}

func (shd *simpleHeaderDeserializer) SetSinglePointFooter(v []byte) {
	shd.footerSingleCurvePoint = copyByteSlice(v)
	shd.ensureInt32Constraints()
}

func (shd *simpleHeaderDeserializer) GetSinglePointFooter() []byte {
	return copyByteSlice(shd.footerSingleCurvePoint)
}

func (shd *simpleHeaderDeserializer) deserializeSinglePointFooter(input io.Reader) (bytesRead int, err error) {
	return consumeExpectRead(input, shd.footerSingleCurvePoint) // ensureInt32Constrains ensures bytesRead fits into int32
}

func (shs *simpleHeaderSerializer) serializeSinglePointFooter(output io.Writer) (bytesWritten int, err error) {
	return output.Write(shs.footerSingleCurvePoint)
}

func (shd *simpleHeaderDeserializer) SinglePointHeaderOverhead() int32 {
	// ensureInt32Contrains ensures this does not overflow int32
	return int32(len(shd.headerSingleCurvePoint) + len(shd.footerSingleCurvePoint))
}

// Should the return type be a plain error? (The caller can just type-assert back)

func (shd *simpleHeaderDeserializer) MultiPointHeaderOverhead(numPoints int32) (ret int32, err *bandersnatchErrors.ErrorWithData[int64]) {
	var ret64 int64
	// shd.fixNilEntries()
	if numPoints < 0 {
		panic(fmt.Errorf("bandersnatch / serializer: Querying overhead size for slice (de)serialization for negative length %v", numPoints))
	}
	ret64 = int64(numPoints) * int64(len(shd.headerPerCurvePoint)+len(shd.footerPerCurvePoint)) // both factors are guaranteed to fit into int32, so no overflow here.
	ret64 += simpleHeaderSliceLengthOverhead                                                    // for writing the size
	ret64 += int64(len(shd.headerSlice) + len(shd.footerSlice))                                 // term added is guaranteed to fit into int32
	// NOTE: ret64 is guaranteed to not have overflown an int64, since it is at most (2^31-1) * (2^31-1) + 4 + (2^31-1), which is smaller than 2^63-1
	if ret64 > math.MaxInt32 {
		err = bandersnatchErrors.NewErrorWithData(fmt.Errorf("MultiPointOverhead does not fit into int32, size was %v", ret64), "", ret64)
	}
	ret = int32(ret64)
	return
}
