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

type headerDeserializer interface {
	deserializeGlobalSliceHeader(input io.Reader) (bytes_read int, size int32, err error)
	deserializeGlobalSliceFooter(input io.Reader) (bytes_read int, err error)
	deserializeSinglePointHeader(input io.Reader) (bytes_read int, err error)
	deserializeSinglePointFooter(input io.Reader) (bytes_read int, err error)
	deserializePerPointHeader(input io.Reader) (bytes_read int, err error)
	deserializePerPointFooter(input io.Reader) (bytes_read int, err error)
	SinglePointHeaderOverhead() int               // returns the size taken up by headers and footers for single-point
	MultiPointHeaderOverhead(numPoints int32) int // returns the size taken up by headers and footers for slice of given size
}

type headerSerializer interface {
	headerDeserializer
	serializeGlobalSliceHeader(output io.Writer, size int32) (bytes_written int, err error)
	serializeGlobalSliceFooter(output io.Writer) (bytes_written int, err error)
	serializeSinglePointHeader(output io.Writer) (bytes_written int, err error)
	serializeSinglePointFooter(output io.Writer) (bytes_written int, err error)
	serializePerPointHeader(output io.Writer) (bytes_written int, err error)
	serializePerPointFooter(output io.Writer) (bytes_written int, err error)
}

type simpleHeaderDeserializer struct {
	headerSlice            []byte
	headerPerCurvePoint    []byte
	headerSingleCurvePoint []byte
	footerSingleCurvePoint []byte
	footerPerCurvePoint    []byte
	footerSlice            []byte

	sliceSizeEndianness binary.ByteOrder
}

type simpleHeaderSerializer struct {
	simpleHeaderDeserializer
}

func (shd *simpleHeaderDeserializer) Clone() *simpleHeaderDeserializer {
	var ret simpleHeaderDeserializer
	deepCopyMaybeNilByteSlice(ret.headerSlice, shd.headerSlice)
	deepCopyMaybeNilByteSlice(ret.headerPerCurvePoint, shd.headerPerCurvePoint)
	deepCopyMaybeNilByteSlice(ret.headerSingleCurvePoint, shd.headerSingleCurvePoint)
	deepCopyMaybeNilByteSlice(ret.footerSlice, shd.footerSlice)
	deepCopyMaybeNilByteSlice(ret.footerPerCurvePoint, shd.footerPerCurvePoint)
	deepCopyMaybeNilByteSlice(ret.footerSingleCurvePoint, shd.footerSingleCurvePoint)
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

func (shd *simpleHeaderDeserializer) SetGlobalSliceHeader(v []byte) {
	deepCopyMaybeNilByteSlice(shd.headerSlice, v)
}

func (shd *simpleHeaderDeserializer) GetGlobalSliceHeader() []byte {
	return getHeaderByteSlice(shd.headerSlice)
}

func (shd *simpleHeaderDeserializer) deserializeGlobalSliceHeader(input io.Reader) (bytes_read int, size int32, err error) {
	header := shd.GetGlobalSliceHeader()
	bytes_read, err = consumeExpectRead(input, header)
	if err != nil {
		return
	}
	var buf [4]byte
	bytesJustRead, err := io.ReadFull(input, buf[:])
	bytes_read += bytesJustRead
	if err != nil {
		utils.UnexpectEOF(&err)
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
	header := shs.GetGlobalSliceHeader()
	bytesWritten, err = output.Write(header[:])
	if err != nil {
		return
	}

	var buf [4]byte
	shs.sliceSizeEndianness.PutUint32(buf[:], uint32(size))
	bytesJustWritten, err := output.Write(buf[:])
	bytesWritten += bytesJustWritten
	if err != nil {
		utils.UnexpectEOF(&err)
		return
	}
	return
}

func (shd *simpleHeaderDeserializer) SetGlobalSliceFooter(v []byte) {
	deepCopyMaybeNilByteSlice(shd.footerSlice, v)
}

func (shd *simpleHeaderDeserializer) GetGlobalSliceFooter() []byte {
	return getHeaderByteSlice(shd.footerSlice)
}

func (shd *simpleHeaderDeserializer) deserializeGlobalSliceFooter(input io.Reader) (bytesRead int, err error) {
	footer := shd.GetGlobalSliceFooter()
	return consumeExpectRead(input, footer)
}

func (shs *simpleHeaderSerializer) serializeGlobalSliceFooter(output io.Writer) (bytesWritten int, err error) {
	footer := shs.GetGlobalSliceFooter()
	return output.Write(footer)
}

func (shd *simpleHeaderDeserializer) SetPerPointHeader(v []byte) {
	deepCopyMaybeNilByteSlice(shd.headerPerCurvePoint, v)
}

func (shd *simpleHeaderDeserializer) GetPerPointHeader() []byte {
	return getHeaderByteSlice(shd.headerPerCurvePoint)
}

func (shd *simpleHeaderDeserializer) deserializePerPointHeader(input io.Reader) (bytesRead int, err error) {
	perPointHeader := shd.GetPerPointHeader()
	return consumeExpectRead(input, perPointHeader)
}

func (shs *simpleHeaderSerializer) serializePerPointHeader(output io.Writer) (bytesWritten int, err error) {
	perPointHeader := shs.GetPerPointHeader()
	return output.Write(perPointHeader)
}

func (shd *simpleHeaderDeserializer) SetPerPointFooter(v []byte) {
	deepCopyMaybeNilByteSlice(shd.footerPerCurvePoint, v)
}

func (shd *simpleHeaderDeserializer) GetPerPointFooter() []byte {
	return getHeaderByteSlice(shd.footerPerCurvePoint)
}

func (shd *simpleHeaderDeserializer) deserializePerPointFooter(input io.Reader) (bytesRead int, err error) {
	perPointFooter := shd.GetPerPointFooter()
	return consumeExpectRead(input, perPointFooter)
}

func (shs *simpleHeaderSerializer) serializePerPointFooter(output io.Writer) (bytesWritten int, err error) {
	perPointFooter := shs.GetPerPointFooter()
	return output.Write(perPointFooter)
}

func (shd *simpleHeaderDeserializer) SetSinglePointHeader(v []byte) {
	deepCopyMaybeNilByteSlice(shd.headerSingleCurvePoint, v)
}

func (shd *simpleHeaderDeserializer) GetSinglePointHeader() []byte {
	return getHeaderByteSlice(shd.headerSingleCurvePoint)
}

func (shd *simpleHeaderDeserializer) deserializeSinglePointHeader(input io.Reader) (bytesRead int, err error) {
	singlePointHeader := shd.GetSinglePointHeader()
	return consumeExpectRead(input, singlePointHeader)
}

func (shs *simpleHeaderSerializer) serializeSinglePointHeader(output io.Writer) (bytesWritten int, err error) {
	singlePointHeader := shs.GetSinglePointHeader()
	return output.Write(singlePointHeader)
}

func (shd *simpleHeaderDeserializer) SetSinglePointFooter(v []byte) {
	deepCopyMaybeNilByteSlice(shd.footerSingleCurvePoint, v)
}

func (shd *simpleHeaderDeserializer) GetSinglePointFooter() []byte {
	return getHeaderByteSlice(shd.footerSingleCurvePoint)
}

func (shd *simpleHeaderDeserializer) deserializeSinglePointFooter(input io.Reader) (bytesRead int, err error) {
	singlePointFooter := shd.GetSinglePointFooter()
	return consumeExpectRead(input, singlePointFooter)
}

func (shs *simpleHeaderSerializer) serializeSinglePointFooter(output io.Writer) (bytesWritten int, err error) {
	singlePointFooter := shs.GetSinglePointFooter()
	return output.Write(singlePointFooter)
}

func (shd *simpleHeaderDeserializer) SinglePointHeaderOverhead() int {
	shd.fixNilEntries()
	return len(shd.headerSingleCurvePoint) + len(shd.footerSingleCurvePoint)
}

func (shs *simpleHeaderSerializer) SinglePointHeaderOverhead() int {
	shs.fixNilEntries()
	return len(shs.headerSingleCurvePoint) + len(shs.footerSingleCurvePoint)
}

// TODO: Overflow protection

func (shd *simpleHeaderDeserializer) MultiPointHeaderOverhead(numPoints int32) (ret int) {
	shd.fixNilEntries()
	if numPoints < 0 {
		panic("bandersnatch / serializer: Querying Overhead size for slice deserialization for negative length")
	}
	ret = 4 // for writing the size
	ret += len(shd.headerSlice) + len(shd.footerSlice)
	ret += int(numPoints) * (len(shd.headerPerCurvePoint) + len(shd.footerPerCurvePoint))
	return
}

func (shd *simpleHeaderSerializer) MultiPointHeaderOverhead(numPoints int32) (ret int) {
	shd.fixNilEntries()
	if numPoints < 0 {
		panic("bandersnatch / serializer: Querying Overhead size for slice deserialization for negative length")
	}
	ret = 4 // for writing the size
	ret += len(shd.headerSlice) + len(shd.footerSlice)
	ret += int(numPoints) * (len(shd.headerPerCurvePoint) + len(shd.footerPerCurvePoint))
	return
}
