package pointserializer

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/bandersnatchErrors"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/common"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/curvePoints"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/errorsWithData"
	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
)

// *** PURPOSE OF THIS PACAKGE ***:
// This package contains interfaces/methods/struct/functions used to (de)serialize bandersnatch curve points.
// Generally, we define structs/interfaces that define some (de)serializer (named, say s)
// s holds all the metadata (such as e.g. endianness) needed to define the serialization format.
// Modifying such metadata contained in a (de)serializer is done by making a copy with modified parameters.
// The (de)serializers themselves are considered (shallowly) immutable.
// To actually serialize a curve point, the user calls some appropropriate (de)serialization method, e.g.
// s.SomeSerializeMethod(outputStream io.Writer, curvePoint ...) (bytesWritten int, err SomeErrorInterfaceType)
// or
// s.SomeDeserializeMethod(inputStream io.Reader, targetCurvePoint, ...) (bytesRead int, err SomeErrorInterfaceType)
// Note here that deserializers do not usually return points, but rather take some buffer to hold points as input.
// The reason is that targetCurvePoint's type is essentially used to pass the intended type to the deserialization method.
//
//
// Note here that io targets are io.Writer or io.Reader, not []byte. Use bytes.Buffer to wrap this.
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
// The user is provided with an interface type for serializers of single curve points or multiple curve points; the latter is defined in this file.
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

// TODO: Rename this vs. the *_Modifyable variants?

type CurvePointDeserializer interface {
	DeserializeCurvePoint(inputStream io.Reader, trustLevel common.IsInputTrusted, outputPoint curvePoints.CurvePointPtrInterfaceWrite) (bytesRead int, err bandersnatchErrors.DeserializationError)
	IsSubgroupOnly() bool                             // Can be called on nil pointers of concrete type, indicates whether the deserializer is only for subgroup points.
	OutputLength() int32                              // returns the length in bytes that this serializer will try at most to read per curve point.
	SliceOutputLength(numPoints int32) (int32, error) // returns the length in bytes that this serializer will try at most to read if deserializing a slice of numPoints many points.

	GetParameter(parameterName string) interface{} // obtains a parameter (such as endianness. parameterName is case-insensitive.
	GetFieldElementEndianness() common.FieldElementEndianness
	Validate() // internal self-check function. Users need not call this.

	RecognizedParameters() []string
	HasParameter(parameterName string) bool

	DeserializeCurvePoints(inputStream io.Reader, trustLevel common.IsInputTrusted, outputPoints curvePoints.CurvePointSlice) (bytesRead int, err BatchDeserializationError)
	DeserializeSlice(inputStream io.Reader, trustLevel common.IsInputTrusted, sliceMaker DeserializeSliceMaker) (output any, bytesRead int, err BatchDeserializationError)
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

	DeserializeCurvePoints(inputStream io.Reader, trustLevel common.IsInputTrusted, outputPoints curvePoints.CurvePointSlice) (bytesRead int, err BatchDeserializationError)
	DeserializeSlice(inputStream io.Reader, trustLevel common.IsInputTrusted, sliceMaker DeserializeSliceMaker) (output any, bytesRead int, err BatchDeserializationError)

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
	basicDeserializer  BasicValue               // Due to immutability, having a pointer would be fine as well.
	headerDeserializer simpleHeaderDeserializer // we could do struct embeding here (well, not with generics...), but some methods are defined on both members, so we prefer explicit forwarding for clarity.
}

type multiSerializer[BasicValue any, BasicPtr interface {
	*BasicValue
	modifyableSerializer_basic[BasicValue, BasicPtr]
}] struct {
	basicSerializer  BasicValue             // Due to immutability, having a pointer would be fine as well.
	headerSerializer simpleHeaderSerializer // we could do struct embeding here (well, not with generics...), but some methods are defined on both members, so we prefer explicit forwarding for clarity.
}

type BatchSerializationErrorData struct {
	bandersnatchErrors.WriteErrorData
	PointsSerialized int
}

type BatchDeserializationErrorData struct {
	bandersnatchErrors.ReadErrorData
	PointsDeserialized int
}

const FIELDNAME_POINTSDESERIALIZED = "PointsDeserialized"
const FIELDNAME_POINTSSERIALIZED = "PointsSerialized"

func init() {
	errorsWithData.CheckParameterForStruct[BatchDeserializationErrorData](FIELDNAME_POINTSDESERIALIZED)
	errorsWithData.CheckParameterForStruct[BatchSerializationErrorData](FIELDNAME_POINTSSERIALIZED)
	errorsWithData.CheckParameterForStruct[BatchDeserializationErrorData]("PointsDeserialized")
	errorsWithData.CheckParameterForStruct[BatchSerializationErrorData]("PointsSerialized")
}

type BatchSerializationError = errorsWithData.ErrorWithGuaranteedParameters[BatchSerializationErrorData]
type BatchDeserializationError = errorsWithData.ErrorWithGuaranteedParameters[BatchDeserializationErrorData]

// ErrInsufficientBufferForDeserialization is the (base) error output when DeserializeSliceToBuffer is called with a buffer of insufficient size.
//
// Note that the actual error returned wraps this error (and the error message reports the actual sizes)
var ErrInsufficientBufferForDeserialization BatchDeserializationError = errorsWithData.NewErrorWithParametersFromData(nil,
	ErrorPrefix+"The provided buffer is too small to store the curve point slice",
	&BatchDeserializationErrorData{
		PointsDeserialized: 0,
		ReadErrorData: bandersnatchErrors.ReadErrorData{
			PartialRead: true,
		}})

// ***********************************************************************************************************************************************************

// makeCopy is basically a variant of Clone() that returns a non-pointer and does not throw away the concrete struct type.
//
// This distinction is made here, because here Clone() actually returns an interface (as opposed to essentially everywhere else).
// The latter is done to avoid users having to see our generics.
func (md *multiDeserializer[BasicValue, BasicPtr]) makeCopy() multiDeserializer[BasicValue, BasicPtr] {
	var ret multiDeserializer[BasicValue, BasicPtr]
	ret.basicDeserializer = *BasicPtr(&md.basicDeserializer).Clone()
	ret.headerDeserializer = *md.headerDeserializer.Clone()
	return ret
}

// makeCopy is basically a variant of Clone() that returns a non-pointer and does not throw away the concrete struct type.
//
// This distinction is made here, because here Clone() actually returns an interface (as opposed to essentially everywhere else).
// The latter is done to avoid users having to see our generics.
func (md *multiSerializer[BasicValue, BasicPtr]) makeCopy() multiSerializer[BasicValue, BasicPtr] {
	var ret multiSerializer[BasicValue, BasicPtr]
	ret.basicSerializer = *BasicPtr(&md.basicSerializer).Clone()
	ret.headerSerializer = *md.headerSerializer.Clone()
	return ret
}

// Validate checks the internal data of the deserializer for validity.
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

// Validate checks the internal data of the serializer for validity.
//
// It panics on error
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

// Clone() returns a copy of itself (as a pointer inside an interface)
//
// Note that for the sake of simplicity (to avoid exporting a generic type), Clone throws away the concrete type here.
func (md *multiDeserializer[BasicValue, BasicPtr]) Clone() CurvePointDeserializerModifyable {
	mdCopy := md.makeCopy()
	return &mdCopy
}

// Clone() returns a copy of itself (as a pointer inside an interface)
//
// Note that for the sake of simplicity (to avoid exporting a generic type), Clone throws away the concrete type here.
func (md *multiSerializer[BasicValue, BasicPtr]) Clone() CurvePointSerializerModifyable {
	mdCopy := md.makeCopy()
	return &mdCopy
}

// RecognizedParameters returns a list of parameters that can be queried/modified via WithParameter / GetParameter
func (md *multiDeserializer[BasicValue, BasicPtr]) RecognizedParameters() []string {

	// return the union of parameters from its components.
	list1 := BasicPtr(&md.basicDeserializer).RecognizedParameters()
	list2 := md.headerDeserializer.RecognizedParameters()
	return concatParameterList(list1, list2)
}

// RecognizedParameters returns a list of parameters that can be queried/modified via WithParameter / GetParameter
func (md *multiSerializer[BasicValue, BasicPtr]) RecognizedParameters() []string {

	// return the union of parameters from its components.
	list1 := BasicPtr(&md.basicSerializer).RecognizedParameters()
	list2 := md.headerSerializer.RecognizedParameters()
	return concatParameterList(list1, list2)
}

// HasParameter tells whether a given parameterName is the name of a valid parameter for this deserializer.
func (md *multiDeserializer[BasicValue, BasicPtr]) HasParameter(parameterName string) bool {
	return BasicPtr(&md.basicDeserializer).HasParameter(parameterName) || md.headerDeserializer.HasParameter(parameterName)
}

// HasParameter tells whether a given parameterName is the name of a valid parameter for this serializer.
func (md *multiSerializer[BasicValue, BasicPtr]) HasParameter(parameterName string) bool {
	return BasicPtr(&md.basicSerializer).HasParameter(parameterName) || md.headerSerializer.HasParameter(parameterName)
}

// WithParameter and GetParameter are complicated by the fact that we cannot struct-embed generic type parameters.

// WithParameter returns a new Deserializer with the parameter determined by parameterName changed to newParam.
//
// Note: This function will panic if the parameter does not exists or the new deserializer would have invalid parameters.
func (md *multiDeserializer[BasicValue, BasicPtr]) WithParameter(parameterName string, newParam any) CurvePointDeserializerModifyable {
	mdCopy := md.makeCopy()
	var basicDeserializationPtr = BasicPtr(&mdCopy.basicDeserializer)
	var found bool = false

	// Check which one of the components have a parameter with the given parameterName and change it.
	if basicDeserializationPtr.HasParameter(parameterName) {
		mdCopy.basicDeserializer = makeCopyWithParameters(basicDeserializationPtr, parameterName, newParam)
		found = true
	}
	if md.headerDeserializer.HasParameter(parameterName) {
		mdCopy.headerDeserializer = makeCopyWithParameters(&mdCopy.headerDeserializer, parameterName, newParam)
		found = true
	}
	if !found {
		panic(fmt.Errorf(ErrorPrefix+"Trying to set parameter %v that does not exist for this deserializer", parameterName))
	}
	mdCopy.Validate()
	return &mdCopy
}

// WithParameter returns a new Serializer with the parameter determined by parameterName changed to newParam.
//
// Note: This function will panic if the parameter does not exists or the new serializer would have invalid parameters.
func (md *multiSerializer[BasicValue, BasicPtr]) WithParameter(parameterName string, newParam any) CurvePointSerializerModifyable {
	mdCopy := md.makeCopy()
	var basicSerializationPtr = BasicPtr(&mdCopy.basicSerializer)
	var found bool = false

	// Check which of the components have a parameter with the given parameterName and change it.
	if basicSerializationPtr.HasParameter(parameterName) {
		mdCopy.basicSerializer = makeCopyWithParameters(basicSerializationPtr, parameterName, newParam)
		found = true
	}

	if md.headerSerializer.HasParameter(parameterName) {
		mdCopy.headerSerializer = makeCopyWithParameters(&mdCopy.headerSerializer, parameterName, newParam)
		found = true
	}
	if !found {
		panic(fmt.Errorf(ErrorPrefix+"Trying to set parameter %v that does not exist for this serializer", parameterName))
	}
	mdCopy.Validate()
	return &mdCopy
}

// GetParameter returns the value stored under the given parameterName for this deserializer.
//
// Note that this method panics if the parameterName is not valid. Check with HasParameter first, if needed.
func (md *multiDeserializer[BasicValue, BasicPtr]) GetParameter(parameterName string) any {
	basicPointer := BasicPtr(&md.basicDeserializer)

	// If parameterName is contained in both, preference is given to basicPointer

	if basicPointer.HasParameter(parameterName) {
		return basicPointer.GetParameter(parameterName)
	} else {
		return getSerializerParameter(&md.headerDeserializer, parameterName)
	}
}

// GetParameter returns the value stored under the given parameterName for this serializer.
//
// Note that this method panics if the parameterName is not valid. Check with HasParameter first, if needed.
func (md *multiSerializer[BasicValue, BasicPtr]) GetParameter(parameterName string) any {
	basicPointer := BasicPtr(&md.basicSerializer)

	// If parameterName is contained in both, preference is given to basicPointer
	if basicPointer.HasParameter(parameterName) {
		return basicPointer.GetParameter(parameterName)
	} else {
		return getSerializerParameter(&md.headerSerializer, parameterName)
	}
}

// WithEndianness returns a copy of the given deserializer with the endianness used for field element (de)serialization changed.
//
// NOTE: The endianness for (de)serialization of slice size headers is NOT affected by this method.
// NOTE2: newEndianness must be either literal binary.BigEndian, binary.LittleEndian or a common.FieldElementEndianness. In particular it must be non-nil. Failure to do so will panic.
func (md *multiDeserializer[BasicValue, BasicPtr]) WithEndianness(newEndianness binary.ByteOrder) CurvePointDeserializerModifyable {
	mdcopy := md.makeCopy()
	mdcopy.basicDeserializer = BasicPtr(&mdcopy.basicDeserializer).WithEndianness(newEndianness)
	mdcopy.Validate()
	return &mdcopy
}

// WithEndianness returns a copy of the given serializer with the endianness used for field element (de)serialization changed.
//
// NOTE: The endianness for (de)serialization of slice size headers is NOT affected by this method.
// NOTE2: newEndianness must be either literal binary.BigEndian, binary.LittleEndian or a common.FieldElementEndianness. In particular it must be non-nil. Failure to do so will panic.
func (md *multiSerializer[BasicValue, BasicPtr]) WithEndianness(newEndianness binary.ByteOrder) CurvePointSerializerModifyable {
	mdcopy := md.makeCopy()
	mdcopy.basicSerializer = BasicPtr(&mdcopy.basicSerializer).WithEndianness(newEndianness)
	mdcopy.Validate()
	return &mdcopy
}

// TODO: We would prefer to specify that on error, outputPoint is unchanged.
// Unfortunately, this is quite hard to achieve -- we would need to store a backup copy of the point
// in case the footer deserializer returns an error.
// Unfortunately, the CurvePointPtr interface does not really allow to do that at the moment:
// The issue is that backing up the point with .Clone() and restoring with .SetFrom(...) is not really correct:
// outputPoint might (in fact likely is!) a NaP and we have no guarantee that Clone() or SetFrom works for those.
// We would need a raw ToBytes / FromBytes interface here.

// DeserializeCurvePoint deserializes a single curve point from input stream, (over-)writing to ouputPoint.
// trustLevel indicates whether the input is to be trusted that the data represents any (subgroup)point at all.
//
// On error, outputPoint may or may not be changed.
func (md *multiDeserializer[BasicValue, BasicPtr]) DeserializeCurvePoint(inputStream io.Reader, trustLevel common.IsInputTrusted, outputPoint curvePoints.CurvePointPtrInterfaceWrite) (bytesRead int, err bandersnatchErrors.DeserializationError) {
	bytesRead, err = md.headerDeserializer.deserializeSinglePointHeader(inputStream)
	if err != nil {
		return
	}

	// originalPoint := outputPoint.Clone() // needed to undo changes on error.

	bytesJustRead, err := BasicPtr(&md.basicDeserializer).DeserializeCurvePoint(inputStream, trustLevel, outputPoint)
	bytesRead += bytesJustRead
	if err != nil {
		return
	}
	bytesJustRead, err = md.headerDeserializer.deserializeSinglePointFooter(inputStream)
	bytesRead += bytesJustRead
	// if err != nil {
	// 		outputPoint.SetFrom(originalPoint)
	//	}
	return
}

// DeserializeCurvePoint deserializes a single curve point from input stream, (over-)writing to ouputPoint.
// trustLevel indicates whether the input is to be trusted that the data represents any (subgroup)point at all.
//
// On error, outputPoint may or may not be changed.
func (md *multiSerializer[BasicValue, BasicPtr]) DeserializeCurvePoint(inputStream io.Reader, trustLevel common.IsInputTrusted, outputPoint curvePoints.CurvePointPtrInterfaceWrite) (bytesRead int, err bandersnatchErrors.DeserializationError) {
	bytesRead, err = md.headerSerializer.deserializeSinglePointHeader(inputStream)
	if err != nil {
		return
	}

	// originalPoint := outputPoint.Clone() // needed to undo changes on error.

	bytesJustRead, err := BasicPtr(&md.basicSerializer).DeserializeCurvePoint(inputStream, trustLevel, outputPoint)
	bytesRead += bytesJustRead
	if err != nil {
		return
	}
	bytesJustRead, err = md.headerSerializer.deserializeSinglePointFooter(inputStream)
	bytesRead += bytesJustRead
	// if err != nil {
	//	outputPoint.SetFrom(originalPoint)
	//}
	return
}

// SerializeCurvePoint serializes the given input point to the outputStream.
func (md *multiSerializer[BasicValue, BasicPtr]) SerializeCurvePoint(outputStream io.Writer, inputPoint curvePoints.CurvePointPtrInterfaceRead) (bytesWritten int, err bandersnatchErrors.SerializationError) {
	bytesWritten, err = md.headerSerializer.serializeSinglePointHeader(outputStream)
	if err != nil {
		return
	}
	bytesJustWritten, err := BasicPtr(&md.basicSerializer).SerializeCurvePoint(outputStream, inputPoint)
	bytesWritten += bytesJustWritten
	if err != nil {
		return
	}
	bytesJustWritten, err = md.headerSerializer.serializeSinglePointFooter(outputStream)
	bytesWritten += bytesJustWritten
	return
}

// GetFieldElementEndianness returns the endianness this deserializer used to (de)serialize field elements.
//
// Note that the value is returned as a common.FieldElementEndianness, which is an interface extending binary.ByteOrder.
func (md *multiDeserializer[BasicValue, BasicPtr]) GetFieldElementEndianness() common.FieldElementEndianness {
	return BasicPtr(&md.basicDeserializer).GetEndianness()
}

// GetFieldElementEndianness returns the endianness this serializer used to (de)serialize field elements.
//
// Note that the value is returned as a common.FieldElementEndianness, which is an interface extending binary.ByteOrder.
func (md *multiSerializer[BasicValue, BasicPtr]) GetFieldElementEndianness() common.FieldElementEndianness {
	return BasicPtr(&md.basicSerializer).GetEndianness()
}

// IsSubgroupOnly returns whether this deserializer only works for subgroup elements.
func (md *multiDeserializer[BasicValue, BasicPtr]) IsSubgroupOnly() bool {
	return BasicPtr(&md.basicDeserializer).IsSubgroupOnly()
}

// IsSubgroupOnly returns whether this serializer only works for subgroup elements.
func (md *multiSerializer[BasicValue, BasicPtr]) IsSubgroupOnly() bool {
	return BasicPtr(&md.basicSerializer).IsSubgroupOnly()
}

// OutputLength returns an upper bound on the size (in bytes) that this deserializer will read when deserializing a single curve point.
//
// Note: We can only hope to get an upper bound, because a deserializer that is *not* also a serializer might work (and autodetect) multiple serialization formats;
// these different formats may have different lengths.
func (md *multiDeserializer[BasicValue, BasicPtr]) OutputLength() int32 {
	// Validate ensures this does not overflow
	return md.headerDeserializer.SinglePointHeaderOverhead() + BasicPtr(&md.basicDeserializer).OutputLength()
}

// OutputLength returns the size (in bytes) that this serializer will read or write when (de)serializing a single curve point.
func (md *multiSerializer[BasicValue, BasicPtr]) OutputLength() int32 {
	// Validate ensures this does not overflow
	return md.headerSerializer.SinglePointHeaderOverhead() + BasicPtr(&md.basicSerializer).OutputLength()
}

// SliceOutputLength returns the length in bytes that this deserializer will try to read at most if deserializing a slice of numPoints many points.
// Note that this is an upper bound (for the same reason as with OutputLength)
// error is set on int32 overflow.
func (md *multiDeserializer[BasicValue, BasicPtr]) SliceOutputLength(numPoints int32) (int32, error) {
	basicPtr := BasicPtr(&md.basicDeserializer)
	// Get size used by the actual points (upper bound) and for the headers:
	var pointCost64 int64 = int64(numPoints) * int64(basicPtr.OutputLength())          // guaranteed to not overflow
	overhead, errOverhead := md.headerDeserializer.MultiPointHeaderOverhead(numPoints) // overhead is what is used by the headers, including the size written in-band.

	var err error // returned error value by this method

	// MultiPointHeaderOverhead return an error on overflow. Handle that:
	if errOverhead != nil {

		// TODO: Guarantee Size parameter via the type system to make that check obsolete?

		OverheadSizeExists := errorsWithData.HasParameter(errOverhead, "Size") // Check whether the error has an embedded "Size" datum (which is an int64 containing the true non-overflown value)
		if OverheadSizeExists {
			err = errorsWithData.NewErrorWithParameters(errOverhead, ErrorPrefix+"requested SliceOutputLength exceeds MaxInt32 by overhead alone. Overhead size is %v{Data}, actual points would use another %v{PointSize}", "PointSize", pointCost64)
			err = errorsWithData.DeleteParameterFromError(err, "Size") // Delete parameter, because it only relates to the header size.
			// TODO: Update with "Size" + pointCost64?
		} else {
			err = errorsWithData.NewErrorWithParameters(errOverhead, ErrorPrefix+"requested SliceOutputLength exceeds MaxInt32 by overhead alone. Actual points would use another %v{PointSize}", "PointSize", pointCost64)
		}
		return -1, err // we return -1 for the int32 in case of error here, as the actual value is meaningless. Note that that overhead might well be negative at this point anyway.

	}
	var ret64 int64 = int64(overhead) + pointCost64 // Cannot overflow, because it is bounded by MaxInt32^2 + MaxInt32
	if ret64 > math.MaxInt32 {
		err = errorsWithData.NewErrorWithParameters(nil, ErrorPrefix+"SliceOutputLength would return %v{Size}, which exceeds MaxInt32", "Size", ret64)
		return -1, err
	}
	return int32(ret64), nil
}

// SliceOutputLength returns the length in bytes that this Deserializer will (try to) read/write if deserializing a slice of numPoints many points.
// error is set on int32 overflow.
func (md *multiSerializer[BasicValue, BasicPtr]) SliceOutputLength(numPoints int32) (int32, error) {
	basicPtr := BasicPtr(&md.basicSerializer)

	// Get size used by the actual points (upper bound) and for the headers:
	var pointCost64 int64 = int64(numPoints) * int64(basicPtr.OutputLength())        // guaranteed to not overflow
	overhead, errOverhead := md.headerSerializer.MultiPointHeaderOverhead(numPoints) // overhead is what is used by the headers, including the size written in-band.

	var err error // returned error value

	// MultiPointHeaderOverhead return an error on overflow. Handle that:
	if errOverhead != nil {

		// TODO: Guarantee Size parameter via the type system to make that check obsolete?

		OverheadSizeExists := errorsWithData.HasParameter(errOverhead, "Size") // Check whether the error has an embedded "Size" datum (which is an int64 containing the true non-overflown value)
		if OverheadSizeExists {
			err = errorsWithData.NewErrorWithParameters(errOverhead, ErrorPrefix+"requested SliceOutputLength exceeds MaxInt32 by overhead alone. Overhead size is %v{Data}, actual points would use another %v{PointSize}", "PointSize", pointCost64)
			err = errorsWithData.DeleteParameterFromError(err, "Size") // Delete parameter, because it only relates to the header size.
			// TODO: Update with "Size" + pointCost64?
		} else {
			err = errorsWithData.NewErrorWithParameters(errOverhead, ErrorPrefix+"requested SliceOutputLength exceeds MaxInt32 by overhead alone. Actual points would use another %v{PointSize}", "PointSize", pointCost64)
		}
		return -1, err

	}

	var ret64 int64 = int64(overhead) + pointCost64 // Cannot overflow, because it is bounded by MaxInt32^2 + MaxInt32
	if ret64 > math.MaxInt32 {
		err = errorsWithData.NewErrorWithParameters(nil, ErrorPrefix+"SliceOutputLength would return %v{Size}, which exceeds MaxInt32", "Size", ret64)
		return -1, err
	}
	return int32(ret64), nil
}

// *******************************************************************************
//
// Multi-IO routines
//
// ********************************************************************************

// DeserializeCurvePoints(inputStream, trustLevel, outputPoints...) will deserialize from inputStream and write to the output points in order.
// If no error occurs, DeserializeCurvePoints(inputStream, trustLevel, outputPoint1, outputPoint2, ...) is equivalent to calling
// DeserializeCurvePoint(inputStream, trustLevel, ouputPoint1), DeserializeCurvePoint(inputStream, trustLevel, outputPoint2,), ... in order.
//
// DeserializeCurvePoints will always try to deserialize L := outputPoints.Len() many points or until the first error.
// L times deserializer.OutputLenght() must fit into an int32, else we panic.
// On error, the BatchDeserialization error contains (among other data) via the errorsWithData framework fields PointsDeserialized and PartialRead.
//
// PointsDeserialized is the number of points that were *successfully* deserialized (i.e. actually written to outputPoints).
// If we read from the inputStream, but do not write because the read data fails a subgroup check, this is not counted in PointsDeserialized.
// PartialRead is set to true if we encountered a read error that is not aligned with data encoding points.
//
// NOTE: If you have a slice buf of type []PointType to hold the output, call this with curvePoints.AsCurvePointSlice(buf) to create a view of buf with the appropriate type.
// Be aware that whether PointType is restricted to points in the subgroup or not may control whether we perform subgroup checks on deserialization!
//
// NOTE: When using this method to deserialize AT MOST L points into a buffer, but don't know how many points are in the stream, you need to check that the error wraps either io.EOF/io.UnexpectedEOF and PartialRead is false.
// We provide a convenience function DeserializeCurvePoints_Bounded that handles this case.
// We also provide a convenience variadic version DeserialiveCurvePoints_Variadic.
// These are both functions, not methods.
func (md *multiDeserializer[BasicValue, BasicPtr]) DeserializeCurvePoints(inputStream io.Reader, trustLevel common.IsInputTrusted, outputPoints curvePoints.CurvePointSlice) (bytesRead int, err BatchDeserializationError) {
	L := outputPoints.Len()
	if L > math.MaxInt32 {
		panic(fmt.Errorf(ErrorPrefix+"trying to batch-deserialize %v points, which is more than MaxInt32, with DeserializeBatch", L))
	}
	if int64(L)*int64(md.OutputLength()) > math.MaxInt32 {
		panic(fmt.Errorf(ErrorPrefix+"trying to batch-deserialize %v points, each reading potentially %v bytes. The total number of bytes read might exceed MaxInt32. Bailing out", L, md.OutputLength()))
	}
	for i := 0; i < L; i++ {
		outputPoint := outputPoints.GetByIndex(i) // returns pointer, wrapped in interface
		bytesJustRead, errSingle := md.DeserializeCurvePoint(inputStream, trustLevel, outputPoint)
		bytesRead += bytesJustRead
		if errSingle != nil {
			// Turns an EOF into an UnexpectedEOF if i != 0.
			if i != 0 {
				bandersnatchErrors.UnexpectEOF2(&errSingle)
			}

			// the index i gives the correct value for the PointsDeserialized error data. The other data (including PartialRead) is actually correct.
			err = errorsWithData.NewErrorWithGuaranteedParameters[BatchDeserializationErrorData](errSingle, ErrorPrefix+"batch deserialization failed after deserializing %{PointsDeserialized} points with error %w", "PointsDeserialized", i)
			return
		}
	}
	return
}

// DeserializeCurvePoints(inputStream, trustLevel, outputPoints...) will deserialize from inputStream and write to the output points in order.
// If no error occurs, DeserializeCurvePoints(inputStream, trustLevel, outputPoint1, outputPoint2, ...) is equivalent to calling
// DeserializeCurvePoint(inputStream, trustLevel, ouputPoint1), DeserializeCurvePoint(inputStream, trustLevel, outputPoint2,), ... in order.
//
// DeserializeCurvePoints will always try to deserialize L := outputPoints.Len() many points or until the first error. L times deserializer.OutputLenght() must fit into an int32, else we panic.
// On error, the BatchDeserialization error contains (among other data) via the errorsWithData framework fields PointsDeserialized and PartialRead.
//
// PointsDeserialized is the number of points that were *successfully* deserialized (i.e. actually written to outputPoints).
// If we read from the inputStream, but do not write because the read data fails a subgroup check, this is not counted in PointsDeserialized.
// PartialRead is set to true if we encountered a read error that is not aligned with data encoding points.
//
// NOTE: If you have a slice buf of type []PointType to hold the output, call this with curvePoints.AsCurvePointSlice(buf) to create a view of buf with the appropriate type.
// Be aware that whether PointType is restricted to points in the subgroup or not may control whether we perform subgroup checks on deserialization!
//
// NOTE: When using this method to deserialize AT MOST L points into a buffer, but don't know how many points are in the stream, you need to check that the error wraps either io.EOF/io.UnexpectedEOF and PartialRead is false.
// We provide a convenience function DeserializeCurvePoints_Bounded that handles this case.
// We also provide a convenience variadic version DeserialiveCurvePoints_Variadic.
// These are both functions, not methods.
func (md *multiSerializer[BasicValue, BasicPtr]) DeserializeCurvePoints(inputStream io.Reader, trustLevel common.IsInputTrusted, outputPoints curvePoints.CurvePointSlice) (bytesRead int, err BatchDeserializationError) {
	L := outputPoints.Len()
	if L > math.MaxInt32 {
		panic(fmt.Errorf(ErrorPrefix+"trying to batch-deserialize %v, which is more than MaxInt32 points with DeserializeBatch", L))
	}
	if int64(L)*int64(md.OutputLength()) > math.MaxInt32 {
		panic(fmt.Errorf(ErrorPrefix+"trying to batch-deserialize %v points, each reading potentially %v bytes. The total number of bytes read might exceed MaxInt32. Bailing out", L, md.OutputLength()))
	}
	for i := 0; i < L; i++ {
		outputPoint := outputPoints.GetByIndex(i) // returns pointer, wrapped in interface
		bytesJustRead, errSingle := md.DeserializeCurvePoint(inputStream, trustLevel, outputPoint)
		bytesRead += bytesJustRead
		if errSingle != nil {
			// Turns an EOF into an UnexpectedEOF if i != 0.
			if i != 0 {
				bandersnatchErrors.UnexpectEOF2(&errSingle)
			}

			// the index i gives the correct value for the PointsDeserialized error data. The other data (including PartialRead) is actually correct.
			err = errorsWithData.NewErrorWithGuaranteedParameters[BatchDeserializationErrorData](errSingle, ErrorPrefix+"batch deserialization failed after deserializing %{PointsDeserialized} points with error %w", "PointsDeserialized", i)
			return
		}
	}
	return
}

// DeserializeCurvePoints_Bounded is a variant of the DeserializeCurvePoints method of our (de)serializers.
//
// While the DeserializeCurvePoints method will always try to deserialize exactly outputPoints.Len() many points and report and error if it could not,
// this version will report no error if fewer points were present in the inputStream (and no other error occurred).
// It reports the number of points actually written to outputPoints.
func DeserializeCurvePoints_Bounded(deserializer CurvePointDeserializer, inputStream io.Reader, trustLevel common.IsInputTrusted, outputPoints curvePoints.CurvePointSlice) (bytesRead int, pointsWritten int, err BatchDeserializationError) {
	bytesRead, err = deserializer.DeserializeCurvePoints(inputStream, trustLevel, outputPoints)
	if err == nil {
		pointsWritten = outputPoints.Len()
		return
	}
	// err != nil at this point
	errData := err.GetData()
	pointsWritten = errData.PointsDeserialized
	if errData.PartialRead {
		return
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		err = nil
	}
	return
}

// DeserializeCurvePoints_Variadic is a variadic version of the DeserializeCurvePoints method of our (de)serializers.
//
// Usage: DeserializeCurvePoints_Variadic(deserializer, inputStream, trustLevel, &point_1, &point_2, ...)
// Here point_i are the points to be written to.
// Note that due to the way, Go's variadics work, the (static) type of all &point_i's must be the same (possibly an interface type).
// There is no need to use curvePoints.AsCurvePointsSlice.
func DeserializeCurvePoints_Variadic[PtrType curvePoints.CurvePointPtrInterface](deserializer CurvePointDeserializer, inputStream io.Reader, trustLevel common.IsInputTrusted, outputPoints ...PtrType) (bytesRead int, err BatchDeserializationError) {
	return deserializer.DeserializeCurvePoints(inputStream, trustLevel, curvePoints.AsCurvePointPtrSlice(outputPoints))
}

// main loop of DeserializeSlice, separate function for historical reasons.

func deserializeSlice_mainloop(inputStream io.Reader, trustLevel common.IsInputTrusted, targetSlice curvePoints.CurvePointSlice, deserializer_header headerDeserializer, deserializer_point curvePointDeserializer_basic, size32 int32) (bytesRead int, err BatchDeserializationError) {
	var bytesJustRead int
	var errNonBatch bandersnatchErrors.DeserializationError
	size := int(size32) // i in the loop below should be int (because of type-unsafe inclusion in BatchDeserializationErrorData)
	for i := 0; i < size; i++ {
		// Read/consume per-point header
		bytesJustRead, errNonBatch = deserializer_header.deserializePerPointHeader(inputStream)
		bytesRead += bytesJustRead
		if errNonBatch != nil {
			err = errorsWithData.NewErrorWithGuaranteedParameters[BatchDeserializationErrorData](errNonBatch, ErrorPrefix+"slice deserialization failed when reading per-point header after reading %v{PointsDeserialized} points. Errors was %w", "PointsDeserialized", i, FIELDNAME_PARTIAL_READ, true)
			return
		}
		// Read/consume actual point:
		bytesJustRead, errNonBatch = deserializer_point.DeserializeCurvePoint(inputStream, trustLevel, targetSlice.GetByIndex(i))
		bytesRead += bytesJustRead
		if errNonBatch != nil {
			err = errorsWithData.NewErrorWithGuaranteedParameters[BatchDeserializationErrorData](errNonBatch, ErrorPrefix+"slice deserialization failed after successfully reading %v{PointsDeserialized} points. The error was %w", "PointsDeserialized", i, FIELDNAME_PARTIAL_READ, true)
			// NOTE: PartialRead is always set to true here.
			// TODO: Fix that? We would need to know whether headerDeserializer has a 0-length footer
			return
		}
		// Read/consume per-point footer. Note that PointsDeserialized is set to i+1.
		bytesJustRead, errNonBatch = deserializer_header.deserializePerPointFooter(inputStream)
		bytesRead += bytesJustRead
		if errNonBatch != nil {
			err = errorsWithData.NewErrorWithGuaranteedParameters[BatchDeserializationErrorData](errNonBatch, ErrorPrefix+"slice deserialization failed when reading per-point footer after reading %v{PointsDeserialized} points. Errors was %w", "PointsDeserialized", i+1, FIELDNAME_PARTIAL_READ, true)
			return
		}
	}
	return
}

// NOTE: CreateNewSlice and UseExistingSlice are generic functions. This whole thing really is a workaround for the lack of generic methods in Go1.19.

// DeserializeSlice reads a slice of curve points from inputSteam.
// As opposed to DeserializeCurvePoints, the slice length is contained in-band and the slice is treated as a single (de)serialization object.
//
// the passed sliceMaker argument has type func(length int32) (output any, slice CurvePointSlice, err error) and is called exactly once with an appropriate length.
// The slice return value is where DeserializeSlice will write into. The output return value of sliceMaker is the output return value of DeserializeSlice.
// Note that the type(s) contained in slice influence whether DeserializeSlice performs subgroup checks.
// See the specification of DeserializeSliceMaker for details.
//
// Use sliceMaker == CreateNewSlice[PointType] to have DeserializeSlice create a slice of points. output will have type []PointType.
// Use sliceMaker == UseExistingSlice(existingSlice) to use existingSlice as a buffer to hold the result of deserialization. In this case, output will have type int and equals the number of points written on success.
//
// On error, at least for the two DeserializeSliceMaker's above, output has the correct type, but is meaningless (possibly a nil slice).
// error contains as data (accessible via errorsWithData) a PointsDeserialized field.
// This indicates how many points were successfully writen to slice.
func (md *multiDeserializer[BasicValue, BasicPtr]) DeserializeSlice(inputStream io.Reader, trustLevel common.IsInputTrusted, sliceMaker DeserializeSliceMaker) (output any, bytesRead int, err BatchDeserializationError) {
	var size int32                                          // size of the slice
	var errNonBatch bandersnatchErrors.DeserializationError // error returned from individual deserialization routines

	// read slice header, including the size of the slice to be deserialized.
	bytesRead, size, errNonBatch = md.headerDeserializer.deserializeGlobalSliceHeader(inputStream)

	// If reading the slice header fails, bail out
	if errNonBatch != nil {
		err = errorsWithData.NewErrorWithGuaranteedParameters[BatchDeserializationErrorData](errNonBatch, ErrorPrefix+" slice deserialization could not read header (including size). Error was: %w", FIELDNAME_PARTIAL_READ, bytesRead != 0, FIELDNAME_POINTSDESERIALIZED, 0)

		output, _, _ = sliceMaker(-1) // create a dummy value for output
		return
	}

	// Make sure the total number of bytes that we will read from will not overflow int32. If it does, we bail out early.
	_, overflowErr := md.SliceOutputLength(size)
	if overflowErr != nil {
		err = errorsWithData.NewErrorWithParametersFromData(overflowErr, ErrorPrefix+"when deserializing a slice, the slice header indicated a length for which the number of bytesRead during deserialization may overflow int32: %w", &BatchDeserializationErrorData{PointsDeserialized: 0, ReadErrorData: bandersnatchErrors.ReadErrorData{PartialRead: true}})
		output, _, _ = sliceMaker(-1) // create a dummy value for output
		return
	}

	// Create a slice to hold the result. Note that this may be a view on an existing buffer, depending on what sliceMaker does.
	var outputPointSlice curvePoints.CurvePointSlice
	var errSliceCreate error
	output, outputPointSlice, errSliceCreate = sliceMaker(size)
	if errSliceCreate != nil {
		err = errorsWithData.NewErrorWithParametersFromData(errSliceCreate, "%w", &BatchDeserializationErrorData{
			ReadErrorData:      bandersnatchErrors.ReadErrorData{PartialRead: true},
			PointsDeserialized: 0,
		})
		return
	}

	// Actually deserialize the into the slice now.
	var bytesJustRead int
	bytesJustRead, err = deserializeSlice_mainloop(inputStream, trustLevel, outputPointSlice, &md.headerDeserializer, BasicPtr(&md.basicDeserializer), size)
	bytesRead += bytesJustRead
	if err != nil {
		return
	}

	// consume the global footer
	bytesJustRead, errNonBatch = md.headerDeserializer.deserializeGlobalSliceFooter(inputStream)
	bytesRead += bytesJustRead
	if errNonBatch != nil {
		err = errorsWithData.NewErrorWithGuaranteedParameters[BatchDeserializationErrorData](errNonBatch, ErrorPrefix+" slice deserialization could not read footer. Error was: %w", FIELDNAME_POINTSDESERIALIZED, int(size))
		return
	}

	// Note: Due to the check on _, overFlowErr := md.SliceOutputLength(size) above, this is not supposed to be possible to fail.
	testutils.Assert(bytesRead <= math.MaxInt32)

	return
}

// DeserializeSlice reads a slice of curve points from inputSteam.
// As opposed to DeserializeCurvePoints, the slice length is contained in-band and the slice is treated as a single (de)serialization object.
//
// the passes sliceMaker argument has type func(length int32) (output any, slice CurvePointSlice, err error) and is called exactly once an appropriate length.
// The slice return value is where DeserializeSlice will write into. The output return value of sliceMaker is the output return value of DeserializeSlice.
// Note that the type(s) contained in slice influence whether DeserializeSlice performs subgroup checks.
// See the specification of DeserializeSliceMaker for details.
//
// Use sliceMaker = CreateNewSlice[PointType] to have DeserializeSlice create a slice of points. output will have type []PointType.
// Use sliceMaker = UseExistingSlice(existingSlice) to use existingSlice as a buffer to hold the result of deserialization. output will have type int and equals the number of points written on success.
//
// On error, at least for the two DeserializeSliceMaker's above, output has the correct type, but is meaningless (possibly a nil slice).
// error contains as data (accessible via errorsWithData) a PointsDeserialized field. This indicates how many points were successfully writen to slice.
func (md *multiSerializer[BasicValue, BasicPtr]) DeserializeSlice(inputStream io.Reader, trustLevel common.IsInputTrusted, sliceCreater DeserializeSliceMaker) (output any, bytesRead int, err BatchDeserializationError) {
	var size int32                                          // size of the slice
	var errNonBatch bandersnatchErrors.DeserializationError // error returned from individual deserialization routines

	bytesRead, size, errNonBatch = md.headerSerializer.deserializeGlobalSliceHeader(inputStream)
	if errNonBatch != nil {
		err = errorsWithData.NewErrorWithGuaranteedParameters[BatchDeserializationErrorData](errNonBatch, ErrorPrefix+" slice deserialization could not read header (including size). Error was: %w", FIELDNAME_PARTIAL_READ, bytesRead != 0, FIELDNAME_POINTSDESERIALIZED, 0)

		output, _, _ = sliceCreater(-1)
		return
	}
	_, overflowErr := md.SliceOutputLength(size)
	if overflowErr != nil {
		err = errorsWithData.NewErrorWithParametersFromData(overflowErr, ErrorPrefix+"when deserializing a slice, the slice header indicated a length for which the number of bytesRead during deserialization may overflow int32: %w", &BatchDeserializationErrorData{PointsDeserialized: 0, ReadErrorData: bandersnatchErrors.ReadErrorData{PartialRead: true}})
		output, _, _ = sliceCreater(-1)
		return
	}

	var outputPointSlice curvePoints.CurvePointSlice
	var errSliceCreate error
	output, outputPointSlice, errSliceCreate = sliceCreater(size)
	if errSliceCreate != nil {
		err = errorsWithData.NewErrorWithParametersFromData(errSliceCreate, "%w", &BatchDeserializationErrorData{
			ReadErrorData:      bandersnatchErrors.ReadErrorData{PartialRead: true},
			PointsDeserialized: 0,
		})
		return
	}

	var bytesJustRead int
	bytesJustRead, err = deserializeSlice_mainloop(inputStream, trustLevel, outputPointSlice, &md.headerSerializer, BasicPtr(&md.basicSerializer), size)
	bytesRead += bytesJustRead
	if err != nil {
		return
	}
	bytesJustRead, errNonBatch = md.headerSerializer.deserializeGlobalSliceFooter(inputStream)
	bytesRead += bytesJustRead
	if errNonBatch != nil {
		err = errorsWithData.NewErrorWithGuaranteedParameters[BatchDeserializationErrorData](errNonBatch, ErrorPrefix+" slice deserialization could not read footer. Error was: %w", FIELDNAME_POINTSDESERIALIZED, int(size))
		return
	}

	testutils.Assert(bytesRead <= math.MaxInt32)

	return
}

// DeserializeSliceMaker is the function type passed to DeserializeSlice and is used to determine where DeserializeSlice will write its output.
//
// Notably, DeserializeSlice will call the given function once with the requested length.
// This length may be -1, indicating that DeserializeSlice has detected an error beforehand.
// In this case, the slice and err return variable are ignored. We recommend setting output to a value of appropriate type,
// so the caller of DeserializeSlice can always safely type-assert on output.
//
// Otherwise, length is non-negative. If the DeserializeSliceMaker returns a non-nil err, DeserializeSlice will stop and return
// an error wrapping this err.
//
// Otherwise, DeserializeSlice will forward output to the caller and (try to) write to slice.
type DeserializeSliceMaker = func(length int32) (output any, slice curvePoints.CurvePointSlice, err error)

// CreateNewSlice is a generic function, whose instantiations are of type SliceCreater.
// Instantiations CreateNewSlice[PointType] are supposed to be used as arguments to DeserializeSlice to select the PointType holding the returned slice.
// output is returned to the caller of DeserializeSlice and holds the actual slice of type []PointType.
func CreateNewSlice[PointType any, PointTypePtr interface {
	*PointType
	curvePoints.CurvePointPtrInterface
}](length int32) (output any, slice curvePoints.CurvePointSlice, err error) {
	// length == -1 indicates there was an error in the caller beforehand. We just ensure the output has the correct type.
	if length == -1 {
		output = []PointType(nil)
		return
	}
	var out []PointType = make([]PointType, length)
	output = out
	slice = curvePoints.AsCurvePointSlice[PointType, PointTypePtr](out)
	err = nil
	return
}

// UseExistingSlice(existingSlice) returns a function of type DeserializeSliceMaker.
// The returned function is supposed to be used as argument to DeserializeSlice to use existingSlice as a buffer to hold the output.
// output is returned to the caller of DeserializeSlice and holds the number of points that were written on success of type int.
//
// If the buffer is too small, we return an error and make no write attempts.
// Note that if DeserializeSlice returns an error, output.(int) should be ignored.
func UseExistingSlice[PointType any, PointTypePtr interface {
	*PointType
	curvePoints.CurvePointPtrInterface
}](existingSlice []PointType) DeserializeSliceMaker {
	return func(length int32) (output any, slice curvePoints.CurvePointSlice, err error) {
		// length == -1 indicates there was an error in the caller beforehand. We just ensure the output has the correct type.
		if length == -1 {
			output = int(0)
			return
		}
		// ensure the provided existing slice is large enough. Note that we check for size, not capacity;
		var targetSliceLen int = len(existingSlice)
		if targetSliceLen < int(length) {
			output = int(0)
			// The error message depends on whether the capacity is too small as well.
			if cap(existingSlice) < int(length) {
				err = errorsWithData.NewErrorWithGuaranteedParameters[BatchDeserializationErrorData](ErrInsufficientBufferForDeserialization,
					"%w: in UseExistingSlice, The length of the given buffer was %v{BufferSize}, but the slice read would have size %v{ReadSliceLen}",
					"BufferSize", targetSliceLen,
					"ReadSliceLen", int(length),
					"BufferCapacity", cap(existingSlice))
			} else {
				err = errorsWithData.NewErrorWithGuaranteedParameters[BatchDeserializationErrorData](ErrInsufficientBufferForDeserialization,
					"%w: in UseExistingSlice, the length of the given buffer was %v{BufferSize}, but the slice read would have size %v{ReadSliceLen}. Note that the given buffer would have had sufficient capacity %v{BufferCapacity}",
					"BufferSize", targetSliceLen,
					"ReadSliceLen", int(length),
					"BufferCapacity", cap(existingSlice))
			}
			return
		}
		output = int(length)
		err = nil
		slice = curvePoints.AsCurvePointSlice[PointType, PointTypePtr](existingSlice[0:length])
		return
	}
}

func (md *multiSerializer[BasicValue, BasicPtr]) SerializeCurvePoints(outputStream io.Writer, inputPoints curvePoints.CurvePointSlice) (bytesWritten int, err BatchSerializationError) {
	var _ BatchSerializationErrorData
	panic(0)
	// return
}
