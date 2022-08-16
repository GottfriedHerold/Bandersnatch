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

// *** PURPOSE OF THIS PACAKGE ***:
// This package contains interfaces/methods/struct/functions used to (de)serialize bandersnatch curve points.
// Generally, we define structs/interfaces that define some (de)serializer (named, say s)
// s holds all the metadata (such as e.g. endianness) needed to define the serialization format.
// Modifying such metadata contained in a (de)serializer is done by making a copy with modified parameters.
// the (de)serializers themselves are considered (shallowly) immutable.
// To actually serialize a curve point, the user calls some appropropriate (de)serialization method, e.g.
// s.SomeSerializeMethod(outputStream io.Writer, curvePoint ...) (bytesWritten int, err SomeErrorInterfaceType)
//
//
// Note here that io targets are io.Writer or io.Reader, not []byte.
// Deserialization function also typically have a trustLevel argument to distinguish trusted input from untrusted input.
// Serializers and Deserializers are separate types:
// This is annoying, but the reason is that some methods have slightly differnt signatures:
// While every Serializer actually also "is" a deserializer that allows to read back in what was written,
// a deserializer might support several serialization formats. In this case, the deserializer does not have serialization routines at all,
// but we rather have several serializers and one compatible deserializer.
// Unfortunately, Serializers are not an interface extension of Deserializers.
// (<rant>The sole reason being that method signatures to make copies of serializers need to either return a serializer or a deserializer --
// and interfaces in Go have no co- or contravariance, meaing that even if interface A extends B, a method Foo() B does not satisfy an interface containing Foo() A.
// Note that there is basically no good reason for this behaviour in Go and it outright invalidates several good coding paradigms (be generous in what you accept and restrictive in what you return)
// The main argument why this was not implemented in Go is efficiency;
// while that is true, note, however, that this mostly applies for the similar case where B is a struct satisfying interface A.
// In this case, Go's implicit interface design would require a worst-case call overhead and passing everything as interfaces, which would indeed by a massive performance killer.
// This is the case usually discussed and encountered by people who want to assign a slice of B's to a slice of A's.
// If both A and B are already interfaces, the argument does not really apply and this is mostly a choice between simplicity for the compiler writers and a type system that sucks a little bit less.
// </rant>)

// *** STRUCTURE OF THIS PACKAGE ***
//
// --SerializeCurvePoints: (this file)
//   |--SerializeHeaders (headerSerializer.go)
//   |--SerializeSinglePoints (basic_serializers.go)
//      |--Translate Points from/into sequences of sign bits / field elements [using appropriate functions from curvePoints package] (basic_serializers.go)
//      |--Serialize those values (values_serializers.go)
// --general helper functions (utils.go)
// --helper functions to pass parameter setting up/down this hierarchical structure (oop.go)
//
// The user is provided with an interface type for serializers of single curve points or multiple curve points; this an an implementation is defined in this file.
// The implementation splits this task into a serializer for headers (which includes slice size for serializing multiple points) and serializing single curve points.
// The former is what we call headerSerializers, the latter is called basic_serializers.
// basic_serializers in turn split their task into translating a curve point to a sequence of values (sign or field elements)
// the latter are then serialized by a so-called values serializer.
// Note that we have multiple basic_serializers, depending on what the point is split into (e.g. pointSerializerXY for X and Y coo vs. pointSerializerXAndSignY for X and sign(Y))
// We also have separate appropriate values serializers (e.g. a values serializer for a pair of field elements vs. one for a (field element, sign bit)-pair).

// *** THIS FILE ***
// This file contains the actual serializers that we export to users.
// The user only sees that interface for (De)serializers, providing an API for
// -(De)serializing individual curve points or slices of curve points
// -Creating new (de)serializers by changing parameters (such as Headers or Endianness)
//
// The way the actual implementation works is that we have a (generic, parameterized by the curve point (de)serializer) struct type that combines
// a (de)serializer for single curve points with a (de)serializer for headers.
// Parameter modifications are forwarded to the appropriate part (point (de)serializer / header (de)serializer)

// Since we do not know which parameters these parts might have and different choices result in actually different parameters,
// we cannot put those in the interface definition unless we know the parameter exists for all instantiations.
// So we use a unified API for parameter changes in the form of a WithParameter(parameterName string, newParam any) method
// Its return type is an interface and this actually creates a new (De)Serializer without modifying the receiver:
// To make life simpler and less error-prone, we make (De)serializers (shallowly) immutable objects.
//
// To implement parameter modification itself, we could work with a map[string]any holding all parameters or we translate parameterNames -> properties and use reflection.
// Since the parameters we can set are fairly limited, we opt for the latter;
// this has some serious downsides, but it allows (or rather enforces, actually) setters and getters.
// Notably, we have a global translation map parameterName -> Name of Setter/Getter (and possibly argument type) and use reflection to call those getters/setters by name;
// the advantage is that this composes with struct embedding and actually makes use of getter/setters.
// The point here is that those getters/setters are ususally non-trivial:
// We have validity restrictions on some parameters and immutability means that when setting/getter e.g. a header of type []byte,
// we actually need to make a copy inside the setter and getter.

// TODO: Rename this vs. *Modifyable

type CurvePointDeserializer interface {
	DeserializeCurvePoint(inputStream io.Reader, trustLevel common.IsInputTrusted, outputPoint curvePoints.CurvePointPtrInterfaceWrite) (bytesRead int, err bandersnatchErrors.DeserializationError)
	IsSubgroupOnly() bool                             // Can be called on nil pointers of concrete type, indicates whether the deserializer is only for subgroup points.
	OutputLength() int32                              // returns the length in bytes that this serializer will try at most to read per curve point.
	SliceOutputLength(numPoints int32) (int32, error) // returns the length in bytes that this serializer will try at most to read if deserializing a slice of numPoints many points.

	GetParameter(parameterName string) interface{} // obtains a parameter (such as endianness. parameterName is case-insensitive.
	GetFieldElementEndianness() common.FieldElementEndianness
	Validate() // internal self-check function. Users need not call this.

	RecognizedParameters() []string

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

	RecognizedParameters() []string

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

// ***********************************************************************************************************************************************************

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

func (md *multiDeserializer[BasicValue, BasicPtr]) RecognizedParameters() []string {
	list1 := BasicPtr(&md.basicDeserializer).RecognizedParameters()
	list2 := md.headerDeserializer.RecognizedParameters()
	return concatParameterList(list1, list2)
}

func (md *multiSerializer[BasicValue, BasicPtr]) RecognizedParameters() []string {
	list1 := BasicPtr(&md.basicSerializer).RecognizedParameters()
	list2 := md.headerSerializer.RecognizedParameters()
	return concatParameterList(list1, list2)
}

// WithParameter and GetParameter are complicated by the fact that we cannot struct-embed generic type parameters.

func (md *multiDeserializer[BasicValue, BasicPtr]) WithParameter(parameterName string, newParam any) CurvePointDeserializerModifyable {
	mdCopy := md.makeCopy()
	var basicDeserializationPtr = BasicPtr(&mdCopy.basicDeserializer)
	if hasParameter(basicDeserializationPtr, parameterName) {
		mdCopy.basicDeserializer = makeCopyWithParameters(basicDeserializationPtr, parameterName, newParam)
	} else {
		mdCopy.headerDeserializer = makeCopyWithParameters(&mdCopy.headerDeserializer, parameterName, newParam)
	}
	mdCopy.Validate()
	return &mdCopy
}

func (md *multiSerializer[BasicValue, BasicPtr]) WithParameter(parameterName string, newParam any) CurvePointSerializerModifyable {
	mdCopy := md.makeCopy()
	var basicSerializationPtr = BasicPtr(&mdCopy.basicSerializer)
	if hasParameter(basicSerializationPtr, parameterName) {
		mdCopy.basicSerializer = makeCopyWithParameters(basicSerializationPtr, parameterName, newParam)
	} else {
		mdCopy.headerSerializer = makeCopyWithParameters(&mdCopy.headerSerializer, parameterName, newParam)
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
