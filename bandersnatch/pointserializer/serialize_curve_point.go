package pointserializer

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/bandersnatchErrors"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/common"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/curvePoints"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/errorsWithData"
)

type CurvePointDeserializer interface {
	DeserializeCurvePoint(inputStream io.Reader, trustLevel common.IsInputTrusted, outputPoint curvePoints.CurvePointPtrInterfaceWrite) (bytesRead int, err bandersnatchErrors.DeserializationError)
	IsSubgroupOnly() bool                             // Can be called on nil pointers of concrete type, indicates whether the deserializer is only for subgroup points.
	OutputLength() int32                              // returns the length in bytes that this serializer will try at most to read per curve point.
	SliceOutputLength(numPoints int32) (int32, error) // returns the length in bytes that this serializer will try at most to read if deserializing a slice of numPoints many points.

	GetParameter(parameterName string) interface{} // obtains a parameter (such as endianness. parameterName is case-insensitive.
	GetFieldElementEndianness() common.FieldElementEndianness
	Validate() // internal self-check function. Users need not call this.

	// DeserializePoints(inputStream io.Reader, outputPoints bandersnatch.CurvePointSlice) (bytesRead int, err bandersnatchErrors.BatchSerializationError)
	// DeserializeBatch(inputStream io.Reader, trustLevel common.IsInputTrusted, outputPoints ...curvePoints.CurvePointPtrInterfaceWrite) (bytesRead int, err bandersnatchErrors.DeserializationError)

	// Matches SerializeSlice
	// DeserializeSlice(inputStream io.Reader) (outputPoints bandersnatch.CurvePointSlice, bytesRead int, err bandersnatchErrors.BatchSerializationError)
	// DeserializeSliceToBuffer(inputStream io.Reader, outputPoints bandersnatch.CurvePointSlice) (bytesRead int, pointsRead int32, err bandersnatchErrors.BatchSerializationError)
}

// Note: WithParameter, WithEndianness and Clone "forget" their types.
// The reason is these interfaces are exported and the user should not need to care about the type.

type CurvePointDeserializerModifyable interface {
	CurvePointDeserializer
	WithParameter(parameterName string, newParam any) CurvePointDeserializerModifyable
	WithEndianness(newEndianness binary.ByteOrder) CurvePointDeserializerModifyable
	Clone() CurvePointDeserializerModifyable
}

type CurvePointSerializer interface {

	// similar to curvePointSerializer_basic. We repeat everthing because of go-doc

	DeserializeCurvePoint(inputStream io.Reader, trustLevel common.IsInputTrusted, outputPoint curvePoints.CurvePointPtrInterfaceWrite) (bytesRead int, err bandersnatchErrors.DeserializationError)
	IsSubgroupOnly() bool                             // Can be called on nil pointers of concrete type, indicates whether the deserializer is only for subgroup points.
	OutputLength() int32                              // returns the length in bytes that this serializer will try to read/write per curve point.
	SliceOutputLength(numPoints int32) (int32, error) // returns the length in bytes that this serializer will try to read/write if serializing a slice of numPoints many points.

	GetParameter(parameterName string) interface{} // obtains a parameter (such as endianness. parameterName is case-insensitive.
	GetFieldElementEndianness() common.FieldElementEndianness
	Validate() // internal self-check function. Users need not call this.

	SerializeCurvePoint(outputStream io.Writer, inputPoint curvePoints.CurvePointPtrInterfaceRead) (bytesWritten int, err bandersnatchErrors.SerializationError)

	// DeserializePoints(inputStream io.Reader, outputPoints bandersnatch.CurvePointSlice) (bytesRead int, err bandersnatchErrors.BatchSerializationError)
	// DeserializeBatch(inputStream io.Reader, outputPoints ...bandersnatch.CurvePointPtrInterfaceWrite) (bytesRead int, err bandersnatchErrors.BatchSerializationError)

	// DeserializeSlice(inputStream io.Reader) (outputPoints bandersnatch.CurvePointSlice, bytesRead int, err bandersnatchErrors.BatchSerializationError)
	// DeserializeSliceToBuffer(inputStream io.Reader, outputPoints bandersnatch.CurvePointSlice) (bytesRead int, pointsRead int32, err bandersnatchErrors.BatchSerializationError)

	// SerializePoints(outputStream io.Writer, inputPoints bandersnatch.CurvePointSlice) (bytesWritten int, err bandersnatchErrors.BatchSerializationError) // SerializeBatch(os, points) is equivalent (if no error occurs) to calling Serialize(os, point[i]) for all i. NOTE: This provides the same functionality as SerializePoints, but with a different argument type.
	// SerializeBatch(outputStream io.Writer, inputPoints ...bandersnatch.CurvePointPtrInterfaceRead) (bytesWritten int, err error) // SerializePoints(os, &x1, &x2, ...) is equivalent (if not error occurs, at least) to Serialize(os, &x1), Serialize(os, &x1), ... NOTE: Using SerializePoints(os, points...) with ...-notation might not work due to the need to convert []concrete Point type to []CurvePointPtrInterface. Use SerializeBatch to avoid this.
	// SerializeSlice(outputStream io.Writer, inputSlice bandersnatch.CurvePointSlice) (bytesWritten int, err bandersnatchErrors.BatchSerializationError)   // SerializeSlice(os, points) serializes a slice of points to outputStream. As opposed to SerializeBatch and SerializePoints, the number of points written is stored in the output stream and can NOT be read back individually, but only by DeserializeSlice
}

// Note: WithParameter, WithEndianness and Clone "forget" their types.
// The reason is these interfaces are exported and the user should not need to care about the type.

type CurvePointSerializerModifyable interface {
	CurvePointSerializer
	// modifyableSerializer[SelfValue, SelfPtr]
	WithParameter(parameterName string, newParam any) CurvePointSerializerModifyable
	WithEndianness(newEndianness binary.ByteOrder) CurvePointSerializerModifyable
	Clone() CurvePointSerializerModifyable
}

// this definition crashes staticcheck (my linter) -- bug reported
// unfortunately, expanding the generics does not help. The issue seems to be with interface{*BasicValue; something refering to BasicValue in function signatures} in general.

type multiDeserializer[BasicValue any, BasicPtr interface {
	*BasicValue
	modifyableDeserializer_basic[BasicValue, BasicPtr]
}] struct {
	basicDeserializer  BasicValue
	headerDeserializer simpleHeaderDeserializer // we could do struct embeding here (well, not with generics...), but some methods are defined on both members, so we prefer explicit forwarding for clarity.
}

type multiSerializer[BasicValue any, BasicPtr interface {
	*BasicValue
	modifyableSerializer_basic[BasicValue, BasicPtr]
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

func (md *multiDeserializer[BasicValue, BasicPtr]) Validate() {
	basicDeserializerPtr := BasicPtr(&md.basicDeserializer) // required to tell Go to use the interface constraints for BasicPtr
	basicDeserializerPtr.Validate()
	md.headerDeserializer.Validate()

	// overflow check for output length:
	var singleOutputLength64 int64 = int64(md.headerDeserializer.SinglePointHeaderOverhead()) + int64(basicDeserializerPtr.OutputLength())
	if singleOutputLength64 > math.MaxInt32 {
		panic(fmt.Errorf(ErrorPrefix+"Output length of deserializer for single point is %v, which does not fit into int32", singleOutputLength64))
	}
}

func (md *multiSerializer[BasicValue, BasicPtr]) Validate() {
	basicSerializerPtr := BasicPtr(&md.basicSerializer)
	basicSerializerPtr.Validate()
	md.headerSerializer.Validate()

	// overflow check for output length:
	var singleOutputLength64 int64 = int64(md.headerSerializer.SinglePointHeaderOverhead()) + int64(basicSerializerPtr.OutputLength())
	if singleOutputLength64 > math.MaxInt32 {
		panic(fmt.Errorf(ErrorPrefix+"Output length of deserializer for single point is %v, which does not fit into int32", singleOutputLength64))
	}
}

func (md *multiDeserializer[BasicValue, BasicPtr]) Clone() CurvePointDeserializerModifyable {
	mdCopy := md.makeCopy()
	return &mdCopy
}

func (md *multiSerializer[BasicValue, BasicPtr]) Clone() CurvePointSerializerModifyable {
	mdCopy := md.makeCopy()
	return &mdCopy
}

// WithParameter and GetParameter are complicated by the fact that we cannot struct-embed generic type parameters.

func (md *multiDeserializer[BasicValue, BasicPtr]) WithParameter(parameterName string, newParam any) CurvePointDeserializerModifyable {
	mdCopy := md.makeCopy()
	var basicDeserializationPtr = BasicPtr(&mdCopy.basicDeserializer)
	if hasParameter(basicDeserializationPtr, parameterName) {
		mdCopy.basicDeserializer = makeCopyWithParams(basicDeserializationPtr, parameterName, newParam)
	} else {
		mdCopy.headerDeserializer = makeCopyWithParams(&mdCopy.headerDeserializer, parameterName, newParam)
	}
	mdCopy.Validate()
	return &mdCopy
}

func (md *multiSerializer[BasicValue, BasicPtr]) WithParameter(parameterName string, newParam any) CurvePointSerializerModifyable {
	mdCopy := md.makeCopy()
	var basicSerializationPtr = BasicPtr(&mdCopy.basicSerializer)
	if hasParameter(basicSerializationPtr, parameterName) {
		mdCopy.basicSerializer = makeCopyWithParams(basicSerializationPtr, parameterName, newParam)
	} else {
		mdCopy.headerSerializer = makeCopyWithParams(&mdCopy.headerSerializer, parameterName, newParam)
	}
	mdCopy.Validate()
	return &mdCopy
}

func (md *multiDeserializer[BasicValue, BasicPtr]) GetParameter(parameterName string) any {
	basicPointer := BasicPtr(&md.basicDeserializer)
	if hasParameter(basicPointer, parameterName) {
		return basicPointer.GetParameter(parameterName)
	} else {
		return getSerializerParam(&md.headerDeserializer, parameterName)
	}
}

func (md *multiSerializer[BasicValue, BasicPtr]) GetParameter(parameterName string) any {
	basicPointer := BasicPtr(&md.basicSerializer)
	if hasParameter(basicPointer, parameterName) {
		return basicPointer.GetParameter(parameterName)
	} else {
		return getSerializerParam(&md.headerSerializer, parameterName)
	}
}

func (md *multiDeserializer[BasicValue, BasicPtr]) WithEndianness(newEndianness binary.ByteOrder) CurvePointDeserializerModifyable {
	mdcopy := md.makeCopy()
	mdcopy.basicDeserializer = BasicPtr(&mdcopy.basicDeserializer).WithEndianness(newEndianness)
	mdcopy.Validate()
	return &mdcopy
}

func (md *multiSerializer[BasicValue, BasicPtr]) WithEndianness(newEndianness binary.ByteOrder) CurvePointSerializerModifyable {
	mdcopy := md.makeCopy()
	mdcopy.basicSerializer = BasicPtr(&mdcopy.basicSerializer).WithEndianness(newEndianness)
	mdcopy.Validate()
	return &mdcopy
}

func (md *multiDeserializer[BasicValue, BasicPtr]) DeserializeCurvePoint(inputStream io.Reader, trustLevel common.IsInputTrusted, outputPoint curvePoints.CurvePointPtrInterfaceWrite) (bytesRead int, err bandersnatchErrors.DeserializationError) {
	return BasicPtr(&md.basicDeserializer).DeserializeCurvePoint(inputStream, trustLevel, outputPoint)
}

func (md *multiSerializer[BasicValue, BasicPtr]) SerializeCurvePoint(outputStream io.Writer, inputPoint curvePoints.CurvePointPtrInterfaceRead) (bytesWritten int, err bandersnatchErrors.SerializationError) {
	return BasicPtr(&md.basicSerializer).SerializeCurvePoint(outputStream, inputPoint)
}

func (md *multiSerializer[BasicValue, BasicPtr]) DeserializeCurvePoint(inputStream io.Reader, trustLevel common.IsInputTrusted, outputPoint curvePoints.CurvePointPtrInterfaceWrite) (bytesRead int, err bandersnatchErrors.DeserializationError) {
	return BasicPtr(&md.basicSerializer).DeserializeCurvePoint(inputStream, trustLevel, outputPoint)
}

func (md *multiDeserializer[BasicValue, BasicPtr]) GetFieldElementEndianness() common.FieldElementEndianness {
	return BasicPtr(&md.basicDeserializer).GetEndianness()
}

func (md *multiSerializer[BasicValue, BasicPtr]) GetFieldElementEndianness() common.FieldElementEndianness {
	return BasicPtr(&md.basicSerializer).GetEndianness()
}

func (md *multiDeserializer[BasicValue, BasicPtr]) IsSubgroupOnly() bool {
	return BasicPtr(&md.basicDeserializer).IsSubgroupOnly()
}

func (md *multiSerializer[BasicValue, BasicPtr]) IsSubgroupOnly() bool {
	return BasicPtr(&md.basicSerializer).IsSubgroupOnly()
}

func (md *multiDeserializer[BasicValue, BasicPtr]) OutputLength() int32 {
	// Validate ensures this does not overflow
	return md.headerDeserializer.SinglePointHeaderOverhead() + BasicPtr(&md.basicDeserializer).OutputLength()
}

func (md *multiSerializer[BasicValue, BasicPtr]) OutputLength() int32 {
	// Validate ensures this does not overflow
	return md.headerSerializer.SinglePointHeaderOverhead() + BasicPtr(&md.basicSerializer).OutputLength()
}

// SliceOutputLength returns the length in bytes that this deserializer will try to read if deserializing a slice of numPoints many points.
// error is set on int32 overflow.
func (md *multiDeserializer[BasicValue, BasicPtr]) SliceOutputLength(numPoints int32) (int32, error) {
	basicPtr := BasicPtr(&md.basicDeserializer)
	var pointCost64 int64 = int64(numPoints) * int64(basicPtr.OutputLength())
	overhead, errOverhead := md.headerDeserializer.MultiPointHeaderOverhead(numPoints)
	if errOverhead != nil {
		OverheadSize, OverheadSizeExists := errorsWithData.GetParameterFromError(errOverhead, "Size") // the error might (in fact, does) contain the actual size in a non-int32. We give better error messages then.
		if OverheadSizeExists {
			err := fmt.Errorf(ErrorPrefix+"requested SliceOutputLength exceeds MaxInt32 by overhead alone. Overhead size is %v, actual points would use another %v", OverheadSize, pointCost64)
			return 0, err
		} else {
			err := fmt.Errorf(ErrorPrefix+"requested SliceOutputLength exceeds MaxInt32 by overhead alone. Actual points would use another %v", pointCost64)
			return 0, err
		}
	}
	var ret64 int64 = int64(overhead) + pointCost64 // Cannot overflow, because it is bounded by MaxInt32^2 + MaxInt32
	if ret64 > math.MaxInt32 {
		err := errorsWithData.NewErrorWithParameters(nil, ErrorPrefix+"SliceOutputLength would return %v{Size}, which exceeds MaxInt32", "Size", ret64)
		return 0, err
	}
	return int32(ret64), nil
}

// SliceOutputLength returns the length in bytes that this Deserializer will try to read at most if deserializing a slice of numPoints many points.
// error is set on int32 overflow.
func (md *multiSerializer[BasicValue, BasicPtr]) SliceOutputLength(numPoints int32) (int32, error) {
	basicPtr := BasicPtr(&md.basicSerializer)
	var pointCost64 int64 = int64(numPoints) * int64(basicPtr.OutputLength())
	overhead, errOverhead := md.headerSerializer.MultiPointHeaderOverhead(numPoints)
	if errOverhead != nil {
		OverheadSize, OverheadSizeExists := errorsWithData.GetParameterFromError(errOverhead, "Size") // the error might (in fact, does) contain the actual size in a non-int32. We give better error messages then.
		if OverheadSizeExists {
			err := fmt.Errorf(ErrorPrefix+"requested SliceOutputLength exceeds MaxInt32 by overhead alone. Overhead size is %v, actual points would use another %v", OverheadSize, pointCost64)
			return 0, err
		} else {
			err := fmt.Errorf(ErrorPrefix+"requested SliceOutputLength exceeds MaxInt32 by overhead alone. Actual points would use another %v", pointCost64)
			return 0, err
		}
	}
	var ret64 int64 = int64(overhead) + pointCost64 // Cannot overflow, because it is bounded by MaxInt32^2 + MaxInt32
	if ret64 > math.MaxInt32 {
		err := errorsWithData.NewErrorWithParameters(nil, ErrorPrefix+"SliceOutputLength would return %v{Size}, which exceeds MaxInt32", "Size", ret64)
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
