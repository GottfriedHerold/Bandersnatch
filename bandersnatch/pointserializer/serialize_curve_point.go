package pointserializer

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/bandersnatchErrors"
)

type CurvePointDeserializer interface {
	DeserializeCurvePoint(inputStream io.Reader, trustLevel bandersnatch.IsInputTrusted, outputPoint bandersnatch.CurvePointPtrInterfaceWrite) (bytesRead int, err error)
	IsSubgroupOnly() bool                             // Can be called on nil pointers of concrete type, indicates whether the deserializer is only for subgroup points.
	OutputLength() int32                              // returns the length in bytes that this serializer will try at most to read per curve point.
	SliceOutputLength(numPoints int32) (int32, error) // returns the length in bytes that this serializer will try at most to read if deserializing a slice of numPoints many points.

	GetParameter(parameterName string) interface{} // obtains a parameter (such as endianness. parameterName is case-insensitive.
	GetFieldElementEndianness() binary.ByteOrder
	Verifier // TODO: Remove

	// DeserializePoints(inputStream io.Reader, outputPoints bandersnatch.CurvePointSlice) (bytesRead int, err bandersnatchErrors.BatchSerializationError)
	DeserializeBatch(inputStream io.Reader, trustLevel bandersnatch.IsInputTrusted, outputPoints ...bandersnatch.CurvePointPtrInterfaceWrite) (bytesRead int, err bandersnatchErrors.BatchSerializationError)

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

	DeserializeCurvePoint(inputStream io.Reader, trustLevel bandersnatch.IsInputTrusted, outputPoint bandersnatch.CurvePointPtrInterfaceWrite) (bytesRead int, err error)
	IsSubgroupOnly() bool                             // Can be called on nil pointers of concrete type, indicates whether the deserializer is only for subgroup points.
	OutputLength() int32                              // returns the length in bytes that this serializer will try to read/write per curve point.
	SliceOutputLength(numPoints int32) (int32, error) // returns the length in bytes that this serializer will try to read/write if serializing a slice of numPoints many points.

	GetParameter(parameterName string) interface{} // obtains a parameter (such as endianness. parameterName is case-insensitive.
	GetFieldElementEndianness() binary.ByteOrder
	Verifier // TODO: Remove

	SerializeCurvePoint(outputStream io.Writer, inputPoint bandersnatch.CurvePointPtrInterfaceRead) (bytesWritten int, err error)

	// DeserializePoints(inputStream io.Reader, outputPoints bandersnatch.CurvePointSlice) (bytesRead int, err bandersnatchErrors.BatchSerializationError)
	DeserializeBatch(inputStream io.Reader, outputPoints ...bandersnatch.CurvePointPtrInterfaceWrite) (bytesRead int, err bandersnatchErrors.BatchSerializationError)

	// DeserializeSlice(inputStream io.Reader) (outputPoints bandersnatch.CurvePointSlice, bytesRead int, err bandersnatchErrors.BatchSerializationError)
	// DeserializeSliceToBuffer(inputStream io.Reader, outputPoints bandersnatch.CurvePointSlice) (bytesRead int, pointsRead int32, err bandersnatchErrors.BatchSerializationError)

	// SerializePoints(outputStream io.Writer, inputPoints bandersnatch.CurvePointSlice) (bytesWritten int, err bandersnatchErrors.BatchSerializationError) // SerializeBatch(os, points) is equivalent (if no error occurs) to calling Serialize(os, point[i]) for all i. NOTE: This provides the same functionality as SerializePoints, but with a different argument type.
	SerializeBatch(outputStream io.Writer, inputPoints ...bandersnatch.CurvePointPtrInterfaceRead) (bytesWritten int, err error) // SerializePoints(os, &x1, &x2, ...) is equivalent (if not error occurs, at least) to Serialize(os, &x1), Serialize(os, &x1), ... NOTE: Using SerializePoints(os, points...) with ...-notation might not work due to the need to convert []concrete Point type to []CurvePointPtrInterface. Use SerializeBatch to avoid this.
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

// makeCopy is basically a variant of Clone() that returns a non-pointer and does not throw away the concrete struct type.
func (md *multiDeserializer[BasicValue, BasicPtr]) makeCopy() multiDeserializer[BasicValue, BasicPtr] {
	var ret multiDeserializer[BasicValue, BasicPtr]
	ret.basicDeserializer = *BasicPtr(&md.basicDeserializer).Clone()
	ret.headerDeserializer = *md.headerDeserializer.Clone()
	return ret
}

// makeCopy is basically a variant of Clone() that returns a non-pointer and does not throw away the concrete struct type.
func (md *multiSerializer[BasicValue, BasicPtr]) makeCopy() multiSerializer[BasicValue, BasicPtr] {
	var ret multiSerializer[BasicValue, BasicPtr]
	ret.basicSerializer = *BasicPtr(&md.basicSerializer).Clone()
	ret.headerSerializer = *md.headerSerializer.Clone()
	return ret
}

func (md *multiDeserializer[BasicValue, BasicPtr]) Verify() {
	basicDeserializerPtr := BasicPtr(&md.basicDeserializer) // required to tell Go to use the interface constraints
	basicDeserializerPtr.Verify()
	md.headerDeserializer.Verify()

	// overflow check for output length:
	var singleOutputLength64 int64 = int64(md.headerDeserializer.SinglePointHeaderOverhead()) + int64(basicDeserializerPtr.OutputLength())
	if singleOutputLength64 > math.MaxInt32 {
		panic(fmt.Errorf("Output length of deserializer for single point is %v, which does not fit into int32", singleOutputLength64))
	}
}

func (md *multiSerializer[BasicValue, BasicPtr]) Verify() {
	basicSerializerPtr := BasicPtr(&md.basicSerializer)
	basicSerializerPtr.Verify()
	md.headerSerializer.Verify()

	// overflow check for output length:
	var singleOutputLength64 int64 = int64(md.headerSerializer.SinglePointHeaderOverhead()) + int64(basicSerializerPtr.OutputLength())
	if singleOutputLength64 > math.MaxInt32 {
		panic(fmt.Errorf("Output length of deserializer for single point is %v, which does not fit into int32", singleOutputLength64))
	}
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
	if hasParameter(basicDeserializationPtr, parameterName) {
		mdCopy.basicDeserializer = makeCopyWithParamsNew(basicDeserializationPtr, parameterName, newParam)
	} else {
		mdCopy.headerDeserializer = makeCopyWithParamsNew(&mdCopy.headerDeserializer, parameterName, newParam)
	}
	return &mdCopy
}

func (md *multiSerializer[BasicValue, BasicPtr]) WithParameter(parameterName string, newParam any) CurvePointSerializer {
	mdCopy := md.makeCopy()
	var basicSerializationPtr = BasicPtr(&mdCopy.basicSerializer)
	if hasParameter(basicSerializationPtr, parameterName) {
		mdCopy.basicSerializer = makeCopyWithParamsNew(basicSerializationPtr, parameterName, newParam)
	} else {
		mdCopy.headerSerializer = makeCopyWithParamsNew(&mdCopy.headerSerializer, parameterName, newParam)
	}
	return &mdCopy
}

func (md *multiDeserializer[BasicValue, BasicPtr]) GetParameter(parameterName string) any {
	basicPointer := BasicPtr(&md.basicDeserializer)
	if hasParameter(basicPointer, parameterName) {
		return basicPointer.GetParam(parameterName)
	} else {
		return getSerializerParam(&md.headerDeserializer, parameterName)
	}
}

func (md *multiSerializer[BasicValue, BasicPtr]) GetParameter(parameterName string) any {
	basicPointer := BasicPtr(&md.basicSerializer)
	if hasParameter(basicPointer, parameterName) {
		return basicPointer.GetParam(parameterName)
	} else {
		return getSerializerParam(&md.headerSerializer, parameterName)
	}
}

func (md *multiDeserializer[BasicValue, BasicPtr]) DeserializeCurvePoint(inputStream io.Reader, trustLevel bandersnatch.IsInputTrusted, outputPoint bandersnatch.CurvePointPtrInterfaceWrite) (bytesRead int, err error) {
	return BasicPtr(&md.basicDeserializer).DeserializeCurvePoint(inputStream, trustLevel, outputPoint)
}

func (md *multiSerializer[BasicValue, BasicPtr]) SerializeCurvePoint(outputStream io.Writer, inputPoint bandersnatch.CurvePointPtrInterfaceRead) (bytesWritten int, err error) {
	return BasicPtr(&md.basicSerializer).SerializeCurvePoint(outputStream, inputPoint)
}

func (md *multiSerializer[BasicValue, BasicPtr]) DeserializeCurvePoint(inputStream io.Reader, trustLevel bandersnatch.IsInputTrusted, outputPoint bandersnatch.CurvePointPtrInterfaceWrite) (bytesRead int, err error) {
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

func (md *multiDeserializer[BasicValue, BasicPtr]) OutputLength() int32 {
	// Verify ensures this does not overflow
	return md.headerDeserializer.SinglePointHeaderOverhead() + BasicPtr(&md.basicDeserializer).OutputLength()
}

func (md *multiSerializer[BasicValue, BasicPtr]) OutputLength() int32 {
	// Verify ensures this does not overflow
	return md.headerSerializer.SinglePointHeaderOverhead() + BasicPtr(&md.basicSerializer).OutputLength()
}

// SliceOutputLength returns the length in bytes that this deserializer will try to read if deserializing a slice of numPoints many points.
// error is set on int32 overflow.
func (md *multiDeserializer[BasicValue, BasicPtr]) SliceOutputLength(numPoints int32) (int32, error) {
	basicPtr := BasicPtr(&md.basicDeserializer)
	var pointCost64 int64 = int64(numPoints) * int64(basicPtr.OutputLength())
	overhead, errOverhead := md.headerDeserializer.MultiPointHeaderOverhead(numPoints)
	if errOverhead != nil {
		err := fmt.Errorf("bandernsnatch / serialization: requested sliceOutputLength exceeds MaxInt32 by overhead alone. Overhead size is %v, actual points would use another %v", errOverhead.Data, pointCost64)
		return 0, err
	}
	var ret64 int64 = int64(overhead) + pointCost64 // Cannot overflow, because it is bounded by MaxInt32^2 + MaxInt32
	if ret64 > math.MaxInt32 {
		err := bandersnatchErrors.NewErrorWithData(fmt.Errorf("bandersnatch / serialization: sliceOutputLength would return %v, which exceeds MaxInt32", ret64), "", ret64)
		return 0, err
	}
	return int32(ret64), nil
}

// SliceOutputLength returns the length in bytes that this serializer will try to read/write if serializing/deserialization a slice of numPoints many points.
// error is set on int32 overflow.
func (md *multiSerializer[BasicValue, BasicPtr]) SliceOutputLength(numPoints int32) (int32, error) {
	basicPtr := BasicPtr(&md.basicSerializer)
	var pointCost64 int64 = int64(numPoints) * int64(basicPtr.OutputLength())
	overhead, errOverhead := md.headerSerializer.MultiPointHeaderOverhead(numPoints)
	if errOverhead != nil {
		err := fmt.Errorf("bandernsnatch / serialization: requested sliceOutputLength exceeds MaxInt32 by overhead alone. Overhead size is %v, actual points would use another %v", errOverhead.Data, pointCost64)
		return 0, err
	}
	var ret64 int64 = int64(overhead) + pointCost64 // Cannot overflow, because it is bounded by MaxInt32^2 + MaxInt32
	if ret64 > math.MaxInt32 {
		err := bandersnatchErrors.NewErrorWithData(fmt.Errorf("bandersnatch / serialization: sliceOutputLength would return %v, which exceeds MaxInt32", ret64), "", ret64)
		return 0, err
	}
	return int32(ret64), nil
}

/*
DeserializePoints(inputStream io.Reader, outputPoints bandersnatch.CurvePointSlice) (bytesRead int, err bandersnatchErrors.BatchSerializationError)
	DeserializeBatch(inputStream io.Reader, outputPoints ...bandersnatch.CurvePointPtrInterfaceWrite) (bytesRead int, err bandersnatchErrors.BatchSerializationError)

	DeserializeSlice(inputStream io.Reader) (outputPoints bandersnatch.CurvePointSlice, bytesRead int, err bandersnatchErrors.BatchSerializationError)
	DeserializeSliceToBuffer(inputStream io.Reader, outputPoints bandersnatch.CurvePointSlice) (bytesRead int, pointsRead int32, err bandersnatchErrors.BatchSerializationError)

	SerializePoints(outputStream io.Writer, inputPoints bandersnatch.CurvePointSlice) (bytesWritten int, err bandersnatchErrors.BatchSerializationError) // SerializeBatch(os, points) is equivalent (if no error occurs) to calling Serialize(os, point[i]) for all i. NOTE: This provides the same functionality as SerializePoints, but with a different argument type.
	SerializeBatch(outputStream io.Writer, inputPoints ...bandersnatch.CurvePointPtrInterfaceRead) (bytesWritten int, err error)                         // SerializePoints(os, &x1, &x2, ...) is equivalent (if not error occurs, at least) to Serialize(os, &x1), Serialize(os, &x1), ... NOTE: Using SerializePoints(os, points...) with ...-notation might not work due to the need to convert []concrete Point type to []CurvePointPtrInterface. Use SerializeBatch to avoid this.
	SerializeSlice(outputStream io.Writer, inputSlice bandersnatch.CurvePointSlice) (bytesWritten int, err bandersnatchErrors.BatchSerializationError)   // SerializeSlice(os, points) serializes a slice of points to outputStream. As opposed to SerializeBatch and SerializePoints, the number of points written is stored in the output stream and can NOT be read back individually, but only by DeserializeSlice

*/

/*

func (md *multiSerializer[BasicValue, BasicPtr]) DeserializeBatch(inputStream io.Reader, trustLevel bandersnatch.IsPointTrusted, outputPoints ...bandersnatch.CurvePointPtrInterfaceWrite) (bytesRead int, err bandersnatchErrors.BatchSerializationError) {
	L := len(outputPoints)
	LEN := md.OutputLength()
	if L > math.MaxInt32 {
		panic("bandersnatch / serialization: trying to batach-deserialize more than MaxInt32 points")
	}
	for i := 0; i < L; i++ {
		bytesJustRead, individualErr := md.DeserializeCurvePoint(inputStream, trustLevel, outputPoints[i])
		bytesRead += bytesJustRead
		if individualErr != nil {

		}

	}
	return
}

*/

// TODO: Overwrite GetParam and GetEndianness

// func (md *multiDeserializer) DeseriaizePoints()
