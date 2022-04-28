package pointserializer

import (
	"encoding/binary"
	"io"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch"
)

type CurvePointDeserializer interface {
	DeserializeCurvePoint(inputStream io.Reader, trustLevel bandersnatch.IsPointTrusted, outputPoint bandersnatch.CurvePointPtrInterfaceWrite) (bytesRead int, err error)
	IsSubgroupOnly() bool // Can be called on nil pointers of concrete type, indicates whether the deserializer is only for subgroup points.
	OutputLength() int    // returns the length in bytes that this serializer will try to read/write per curve point.

	GetParameter(parameterName string) interface{} // obtains a parameter (such as endianness. parameterName is a case-insesitive.
	GetFieldElementEndianness() binary.ByteOrder

	// DeserializePoints(inputStream io.Reader, outputPoints bandersnatch.CurvePointSlice) (bytesRead int, err bandersnatchErrors.BatchSerializationError)
	// DeserializeBatch(inputStream io.Reader, outputPoints ...bandersnatch.CurvePointPtrInterfaceWrite) (bytesRead int, err bandersnatchErrors.BatchSerializationError)

	// Matches SerializeSlice
	// DeserializeSlice(inputStream io.Reader) (outputPoints bandersnatch.CurvePointSlice, bytesRead int, err bandersnatchErrors.BatchSerializationError)
	// DeserializeSliceToBuffer(inputStream io.Reader, outputPoints bandersnatch.CurvePointSlice) (bytesRead int, pointsRead int32, err bandersnatchErrors.BatchSerializationError)
}

type CurvePointDeserializerModifyable[SelfValue any] interface {
	CurvePointDeserializer
	modifyableSerializer[SelfValue]
}

type CurvePointSerializer interface {

	// similar to curvePointSerializer_basic

	DeserializeCurvePoint(inputStream io.Reader, trustLevel bandersnatch.IsPointTrusted, outputPoint bandersnatch.CurvePointPtrInterfaceWrite) (bytesRead int, err error)
	IsSubgroupOnly() bool // Can be called on nil pointers of concrete type, indicates whether the deserializer is only for subgroup points.
	OutputLength() int    // returns the length in bytes that this serializer will try to read/write per curve point.

	GetParameter(parameterName string) interface{} // obtains a parameter (such as endianness. parameterName is a case-insesitive.
	GetFieldElementEndianness() binary.ByteOrder

	SerializeCurvePoint(outputStream io.Writer, inputPoint bandersnatch.CurvePointPtrInterfaceRead) (bytesWritten int, err error)

	// DeserializePoints(inputStream io.Reader, outputPoints bandersnatch.CurvePointSlice) (bytesRead int, err bandersnatchErrors.BatchSerializationError)
	// DeserializeBatch(inputStream io.Reader, outputPoints ...bandersnatch.CurvePointPtrInterfaceWrite) (bytesRead int, err bandersnatchErrors.BatchSerializationError)

	// DeserializeSlice(inputStream io.Reader) (outputPoints bandersnatch.CurvePointSlice, bytesRead int, err bandersnatchErrors.BatchSerializationError)
	// DeserializeSliceToBuffer(inputStream io.Reader, outputPoints bandersnatch.CurvePointSlice) (bytesRead int, pointsRead int32, err bandersnatchErrors.BatchSerializationError)

	// SerializePoints(outputStream io.Writer, inputPoints bandersnatch.CurvePointSlice) (bytesWritten int, err bandersnatchErrors.BatchSerializationError) // SerializeBatch(os, points) is equivalent (if no error occurs) to calling Serialize(os, point[i]) for all i. NOTE: This provides the same functionality as SerializePoints, but with a different argument type.
	// SerializeBatch(outputStream io.Writer, inputPoints ...bandersnatch.CurvePointPtrInterfaceRead) (bytesWritten int, err error)                         // SerializePoints(os, &x1, &x2, ...) is equivalent (if not error occurs, at least) to Serialize(os, &x1), Serialize(os, &x1), ... NOTE: Using SerializePoints(os, points...) with ...-notation might not work due to the need to convert []concrete Point type to []CurvePointPtrInterface. Use SerializeBatch to avoid this.
	// SerializeSlice(outputStream io.Writer, inputSlice bandersnatch.CurvePointSlice) (bytesWritten int, err bandersnatchErrors.BatchSerializationError)   // SerializeSlice(os, points) serializes a slice of points to outputStream. As opposed to SerializeBatch and SerializePoints, the number of points written is stored in the output stream and can NOT be read back individually, but only by DeserializeSlice
}
type CurvePointSerializerModifyable[SelfValue any] interface {
	CurvePointSerializer
	modifyableSerializer[SelfValue]
}

type multiDeserializer[BasicValue any, BasicPtr interface {
	*BasicValue
	modifyableDeserializer_basic[BasicValue]
}] struct {
	basicDeserializer  BasicValue
	headerDeserializer simpleHeaderDeserializer // we could do struct embeding here (well, not with generics...), but some methods are defined on both members, so we prefer explicit forwarding for clarity.
}

type multiSerializer[BasicValue any, BasicPtr interface {
	*BasicValue
	modifyableSerializer_basic[BasicValue]
}] struct {
	basicSerializer  BasicValue
	headerSerializer simpleHeaderSerializer // we could do struct embeding here (well, not with generics...), but some methods are defined on both members, so we prefer explicit forwarding for clarity.
}

func (md *multiDeserializer[BasicValue, BasicPtr]) makeCopy() multiDeserializer[BasicValue, BasicPtr] {
	var ret multiDeserializer[BasicValue, BasicPtr]
	ret.basicDeserializer = *BasicPtr(&md.basicDeserializer).Clone()
	ret.headerDeserializer = *md.headerDeserializer.Clone()
	return ret
}

func (md *multiSerializer[BasicValue, BasicPtr]) makeCopy() multiSerializer[BasicValue, BasicPtr] {
	var ret multiSerializer[BasicValue, BasicPtr]
	ret.basicSerializer = *BasicPtr(&md.basicSerializer).Clone()
	ret.headerSerializer = *md.headerSerializer.Clone()
	return ret
}

func (md *multiDeserializer[BasicValue, BasicPtr]) Clone() CurvePointDeserializer {
	mdCopy := md.makeCopy()
	return &mdCopy
}

func (md *multiSerializer[BasicValue, BasicPtr]) Clone() CurvePointSerializer {
	mdCopy := md.makeCopy()
	return &mdCopy
}

func (md *multiDeserializer[BasicValue, BasicPtr]) WithParameter(parameterName string, newParam any) CurvePointDeserializer {
	mdCopy := md.makeCopy()
	var basicDeserializationPtr = BasicPtr(&mdCopy.basicDeserializer)
	if hasParam(basicDeserializationPtr, parameterName) {
		mdCopy.basicDeserializer = makeCopyWithParamsNew(basicDeserializationPtr, parameterName, newParam)
	} else {
		mdCopy.headerDeserializer = makeCopyWithParamsNew(&mdCopy.headerDeserializer, parameterName, newParam)
	}
	return &mdCopy
}

func (md *multiSerializer[BasicValue, BasicPtr]) WithParameter(parameterName string, newParam any) CurvePointSerializer {
	mdCopy := md.makeCopy()
	var basicSerializationPtr = BasicPtr(&mdCopy.basicSerializer)
	if hasParam(basicSerializationPtr, parameterName) {
		mdCopy.basicSerializer = makeCopyWithParamsNew(basicSerializationPtr, parameterName, newParam)
	} else {
		mdCopy.headerSerializer = makeCopyWithParamsNew(&mdCopy.headerSerializer, parameterName, newParam)
	}
	return &mdCopy
}

func (md *multiDeserializer[BasicValue, BasicPtr]) GetParameter(parameterName string) any {
	basicPointer := BasicPtr(&md.basicDeserializer)
	if hasParam(basicPointer, parameterName) {
		return basicPointer.GetParam(parameterName)
	} else {
		return getSerializerParam(&md.headerDeserializer, parameterName)
	}
}

func (md *multiSerializer[BasicValue, BasicPtr]) GetParameter(parameterName string) any {
	basicPointer := BasicPtr(&md.basicSerializer)
	if hasParam(basicPointer, parameterName) {
		return basicPointer.GetParam(parameterName)
	} else {
		return getSerializerParam(&md.headerSerializer, parameterName)
	}
}

func (md *multiDeserializer[BasicValue, BasicPtr]) DeserializeCurvePoint(inputStream io.Reader, trustLevel bandersnatch.IsPointTrusted, outputPoint bandersnatch.CurvePointPtrInterfaceWrite) (bytesRead int, err error) {
	return BasicPtr(&md.basicDeserializer).DeserializeCurvePoint(inputStream, trustLevel, outputPoint)
}

func (md *multiSerializer[BasicValue, BasicPtr]) SerializeCurvePoint(outputStream io.Writer, inputPoint bandersnatch.CurvePointPtrInterfaceRead) (bytesWritten int, err error) {
	return BasicPtr(&md.basicSerializer).SerializeCurvePoint(outputStream, inputPoint)
}

func (md *multiSerializer[BasicValue, BasicPtr]) DeserializeCurvePoint(inputStream io.Reader, trustLevel bandersnatch.IsPointTrusted, outputPoint bandersnatch.CurvePointPtrInterfaceWrite) (bytesRead int, err error) {
	return BasicPtr(&md.basicSerializer).DeserializeCurvePoint(inputStream, trustLevel, outputPoint)
}

func (md *multiDeserializer[BasicValue, BasicPtr]) GetFieldElementEndianness() binary.ByteOrder {
	return BasicPtr(&md.basicDeserializer).GetEndianness()
}

func (md *multiSerializer[BasicValue, BasicPtr]) GetFieldElementEndianness() binary.ByteOrder {
	return BasicPtr(&md.basicSerializer).GetEndianness()
}

func (md *multiDeserializer[BasicValue, BasicPtr]) IsSubgroupOnly() bool {
	return BasicPtr(&md.basicDeserializer).IsSubgroupOnly()
}

func (md *multiSerializer[BasicValue, BasicPtr]) IsSubgroupOnly() bool {
	return BasicPtr(&md.basicSerializer).IsSubgroupOnly()
}

func (md *multiDeserializer[BasicValue, BasicPtr]) OutputLength() int {
	return md.headerDeserializer.SinglePointHeaderOverhead() + BasicPtr(&md.basicDeserializer).OutputLength()
}

func (md *multiSerializer[BasicValue, BasicPtr]) OutputLength() int {
	return md.headerSerializer.SinglePointHeaderOverhead() + BasicPtr(&md.basicSerializer).OutputLength()
}

// TODO: Overwrite GetParam and GetEndianness

// func (md *multiDeserializer) DeseriaizePoints()
