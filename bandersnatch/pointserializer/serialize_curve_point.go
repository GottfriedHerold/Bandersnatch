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
	"github.com/GottfriedHerold/Bandersnatch/internal/errorTransform"
)

// *** PURPOSE OF THIS PACAKGE ***:
// This package contains interfaces/methods/struct/functions used to (de)serialize bandersnatch curve points.
// Generally, we define structs/interfaces that define some (de)serializer (named, say s)
// s holds all the metadata (such as e.g. endianness) needed to define the serialization format.
// Modifying such metadata contained in a (de)serializer is done by making a copy with modified parameters.
// The (de)serializers themselves are considered immutable.
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
	IsSubgroupOnly() bool                             // Equivalent to GetParameter("SubgroupOnly") This indicates whether the deserializer is restricted to subgroup point. Note: If the target curve point type can only hold subgroup elements, this serializer flag is irrelevant and this is the preferred method.
	OutputLength() int32                              // returns the length in bytes that this serializer will try at most to read per curve point.
	SliceOutputLength(numPoints int32) (int32, error) // returns the length in bytes that this serializer will try at most to read if deserializing a slice of numPoints many points.

	GetParameter(parameterName string) interface{}            // obtains a parameter (such as endianness. parameterName is case-insensitive.
	GetFieldElementEndianness() common.FieldElementEndianness // Equivalent to GetParameter("Endianness").(common.FieldElementEndianness)

	// TODO: Can we Remove this?
	Validate() // internal self-check function. Users never have a reason to call this.

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
	IsSubgroupOnly() bool                             // Equivalent to GetParameter("SubgroupOnly").(bool)
	OutputLength() int32                              // returns the length in bytes that this serializer will try to read/write per curve point.
	SliceOutputLength(numPoints int32) (int32, error) // returns the length in bytes that this serializer will try to read/write if serializing a slice of numPoints many points.

	GetParameter(parameterName string) interface{}            // obtains a parameter (such as endianness. parameterName is case-insensitive.
	GetFieldElementEndianness() common.FieldElementEndianness // Equivalent to GetParameter("Endianness").(common.FieldElementEndianness)

	// TODO: Can we Remove this?
	Validate() // internal self-check function. Users never have a reason to call this.

	RecognizedParameters() []string
	HasParameter(parameterName string) bool

	SerializeCurvePoint(outputStream io.Writer, inputPoint curvePoints.CurvePointPtrInterfaceRead) (bytesWritten int, err bandersnatchErrors.SerializationError)

	DeserializeCurvePoints(inputStream io.Reader, trustLevel common.IsInputTrusted, outputPoints curvePoints.CurvePointSlice) (bytesRead int, err BatchDeserializationError)
	DeserializeSlice(inputStream io.Reader, trustLevel common.IsInputTrusted, sliceMaker DeserializeSliceMaker) (output any, bytesRead int, err BatchDeserializationError)

	SerializeCurvePoints(outputStream io.Writer, inputPoints curvePoints.CurvePointSlice) (bytesWritten int, err BatchSerializationError) // SerializePoints(os, points) is equivalent (if no error occurs) to calling Serialize(os, point[i]) for all i.
}

// Note: WithParameter, WithEndianness and Clone "forget" their types.
// The reason is these interfaces are exported and the user should not need to care about the type.

type CurvePointSerializerModifyable interface {
	CurvePointSerializer
	WithParameter(parameterName string, newParam any) CurvePointSerializerModifyable
	WithEndianness(newEndianness binary.ByteOrder) CurvePointSerializerModifyable
	Clone() CurvePointSerializerModifyable
	AsDeserializer() CurvePointDeserializerModifyable
}

var (
	BanderwagonShort CurvePointSerializerModifyable = newMultiSerializer(basicBanderwagonShort, trivialSimpleHeaderSerializer)
	BanderwagonLong  CurvePointSerializerModifyable = newMultiSerializer(basicBanderwagonLong, trivialSimpleHeaderSerializer)
)

// Note: We cannot directly use variables of interface type inside the struct, but rather use generics for two reasons:
//   a) Albeit a minor issue (we could just not support this), handling nils is somewhat different.
//      In particular, nil pointers to multiDeserializer[A,B] contain information about the types A and B.
//      We have functions on A, B that do work with nil receivers of type A resp. B and we want to have the same behaviour on multiDeserializer[A,B] for constency.
//      This is only possible with generics
//   b) At least for BasicPtr, The actual interface satisfied is generic due to Clone() functions retaining the type.
//      Since we need either generics or reflection anyway, we might as well make both types generic.
//

type multiDeserializer[BasicPtr interface {
	modifyableDeserializer_basic[BasicPtr]
	*BasicValue
}, HeaderPtr interface {
	headerDeserializerInterface
	Clone() HeaderPtr
	WithParameter(string, any) HeaderPtr
	*HeaderValue
}, BasicValue any, HeaderValue any] struct {
	basicDeserializer  BasicPtr  // pointer to basic deserializer. Due to immutability, having a pointer is fine.
	headerDeserializer HeaderPtr // pointer to header deserializer. Again, a pointer is fine.
}

type multiSerializer[BasicPtr interface {
	modifyableSerializer_basic[BasicPtr]
	*BasicValue
}, HeaderPtr interface {
	headerSerializerInterface
	Clone() HeaderPtr
	WithParameter(string, any) HeaderPtr
	*HeaderValue
}, BasicValue any, HeaderValue any] struct {
	basicSerializer  BasicPtr  // pointer to basic serializer. Due to immutability, having a pointer is fine.
	headerSerializer HeaderPtr // pointer to header serializer. Again, a pointer is fine.
}

// these functions just serve as workarounds to Go's inability to derive type parameters for variable declarations.

func newMultiDeserializer[BasicPtr interface {
	modifyableDeserializer_basic[BasicPtr]
	*BasicValue
}, HeaderPtr interface {
	headerDeserializerInterface
	Clone() HeaderPtr
	WithParameter(string, any) HeaderPtr
	*HeaderValue
}, BasicValue any, HeaderValue any](basicDeserializer BasicPtr, headerDeserializer HeaderPtr) (ret *multiDeserializer[BasicPtr, HeaderPtr, BasicValue, HeaderValue]) {
	ret = &multiDeserializer[BasicPtr, HeaderPtr, BasicValue, HeaderValue]{basicDeserializer: basicDeserializer, headerDeserializer: headerDeserializer}
	ret.Validate()
	return
}

func newMultiSerializer[BasicPtr interface {
	modifyableSerializer_basic[BasicPtr]
	*BasicValue
}, HeaderPtr interface {
	headerSerializerInterface
	Clone() HeaderPtr
	WithParameter(string, any) HeaderPtr
	*HeaderValue
}, BasicValue any, HeaderValue any](basicSerializer BasicPtr, headerSerializer HeaderPtr) (ret *multiSerializer[BasicPtr, HeaderPtr, BasicValue, HeaderValue]) {

	ret = &multiSerializer[BasicPtr, HeaderPtr, BasicValue, HeaderValue]{basicSerializer: basicSerializer, headerSerializer: headerSerializer}
	ret.Validate()
	return
}

func (md *multiSerializer[BasicPtr, HeaderPtr, BasicValue, HeaderValue]) AsDeserializer() CurvePointDeserializerModifyable {
	return &multiDeserializer[BasicPtr, HeaderPtr, BasicValue, HeaderValue]{basicDeserializer: md.basicSerializer, headerDeserializer: md.headerSerializer}
}

// ErrInsufficientBufferForDeserialization is the (base) error output when DeserializeSliceToBuffer is called with a buffer of insufficient size.
//
// Note that the actual error returned wraps this error (and the error message of the wrapping error reports the actual sizes)
var ErrInsufficientBufferForDeserialization BatchDeserializationError = errorsWithData.NewErrorWithParametersFromData(nil,
	ErrorPrefix+"The provided buffer is too small to store the curve point slice",
	&BatchDeserializationErrorData{
		PointsDeserialized: 0, // We check this before we do any IO on the actual points
		ReadErrorData: bandersnatchErrors.ReadErrorData{
			PartialRead: true, // We still performed IO prior to this error, because we needed to determine the required size of the buffer from the in-band information.
		}})

// ***********************************************************************************************************************************************************

// general note: We check for nil in most methods here. The first reason is that for some methods, we actually at (least try to) comply with the callers request as far as possible.
// (which may or not work, depending on whether the individual point and header serializers handle nil -- but at least some methods of those do by design)
// The second reason is that this type is "pointer-based", i.e. we think of as the pointers as the correct objects and we want to give better error messages for nils.
// NOTE: Only an interface type is actually exported, so users should not be able to create nil pointers anyway (without reflect or unsafe packages). As such these are all internal errors.

// Validate checks the internal data of the deserializer for validity.
//
// It panics on error. Note that this is an internal function.
// We actually provide no easy way for users to generate an instance where this would fail.
//   - nil interfaces don't count, because they are not even instances of this type
//   - nil pointers cannot be easily created because the type is not exported
//   - any modification / copying with modifications of existing instances runs through Validate before the new instance is given to the user
func (md *multiDeserializer[_, _, _, _]) Validate() {
	// would panic anyway, but we prefer a custom error
	if md == nil {
		panic(ErrorPrefix + "Validate called nil pointer of type *multiDeserializer[...]")
	}
	if md.basicDeserializer == nil {
		panic(ErrorPrefix + "Validate called on multiDeserializer whose underlying basic deserializer is nil.")
	}
	if md.headerDeserializer == nil {
		panic(ErrorPrefix + "Validate called on multiDeserializer whose underlying header deserializer is nil.")
	}

	md.basicDeserializer.Validate()
	md.headerDeserializer.Validate()

	// overflow check for output length:
	var singleOutputLength64 int64 = int64(md.headerDeserializer.SinglePointHeaderOverhead()) + int64(md.basicDeserializer.OutputLength())
	if singleOutputLength64 > math.MaxInt32 {
		panic(fmt.Errorf(ErrorPrefix+"Output length of deserializer for single point is %v, which does not fit into int32", singleOutputLength64))
	}
}

// Validate checks the internal data of the serializer for validity.
//
// It panics on error. Note that this is an internal function.
// We actually provide no easy way for users to generate an instance where this would fail.
//   - nil interfaces don't count, because they are not even instances of this type
//   - nil pointers cannot be easily created because the type is not exported
//   - any modification / copying with modifications of existing instances runs through Validate before the new instance is given to the user
func (md *multiSerializer[_, _, _, _]) Validate() {
	// would panic anyway, but we prefer a custom error
	if md == nil {
		panic(ErrorPrefix + "Validate called nil pointer of type *multiSerializer[...]")
	}
	if md.basicSerializer == nil {
		panic(ErrorPrefix + "Validate called on multiSerializer whose underlying basic serializer is nil.")
	}
	if md.headerSerializer == nil {
		panic(ErrorPrefix + "Validate called on multiSerializer whose underlying header serializer is nil.")
	}

	md.basicSerializer.Validate()
	md.headerSerializer.Validate()

	// overflow check for output length:
	var singleOutputLength64 int64 = int64(md.headerSerializer.SinglePointHeaderOverhead()) + int64(md.basicSerializer.OutputLength())
	if singleOutputLength64 > math.MaxInt32 {
		panic(fmt.Errorf(ErrorPrefix+"Output length of deserializer for single point is %v, which does not fit into int32", singleOutputLength64))
	}
}

// Clone() returns a copy of itself (as a pointer inside an interface)
//
// Note that for the sake of simplicity (to avoid exporting a generic type), Clone throws away the concrete type here.
func (md *multiDeserializer[BasicPtr, HeaderPtr, BasicValue, HeaderValue]) Clone() CurvePointDeserializerModifyable {
	return &multiDeserializer[BasicPtr, HeaderPtr, BasicValue, HeaderValue]{basicDeserializer: md.basicDeserializer.Clone(), headerDeserializer: md.headerDeserializer.Clone()}
}

// Clone() returns a copy of itself (as a pointer inside an interface)
//
// Note that for the sake of simplicity (to avoid exporting a generic type), Clone throws away the concrete type here.
func (md *multiSerializer[BasicPtr, HeaderPtr, BasicValue, HeaderValue]) Clone() CurvePointSerializerModifyable {
	return &multiSerializer[BasicPtr, HeaderPtr, BasicValue, HeaderValue]{basicSerializer: md.basicSerializer.Clone(), headerSerializer: md.headerSerializer.Clone()}
}

// RecognizedParameters returns a list of parameters that can be queried/modified via WithParameter / GetParameter
func (md *multiDeserializer[BasicPtr, HeaderPtr, _, _]) RecognizedParameters() []string {
	// We just return the union of parameters from its components.
	// However, if md is nil, we need to handle this explicitly by treating each component as nil.

	var list1 []string
	var list2 []string
	if md == nil {
		list1 = BasicPtr(nil).RecognizedParameters()
		list2 = HeaderPtr(nil).RecognizedParameters()
	} else {

		list1 = md.basicDeserializer.RecognizedParameters()
		list2 = md.headerDeserializer.RecognizedParameters()
	}

	return concatenateParameterList(list1, list2)
}

// RecognizedParameters returns a list of parameters that can be queried/modified via WithParameter / GetParameter
func (md *multiSerializer[BasicPtr, HeaderPtr, _, _]) RecognizedParameters() []string {

	var list1, list2 []string
	if md == nil {
		list1 = BasicPtr(nil).RecognizedParameters()
		list2 = HeaderPtr(nil).RecognizedParameters()
	} else {
		list1 = md.basicSerializer.RecognizedParameters()
		list2 = md.headerSerializer.RecognizedParameters()
	}

	return concatenateParameterList(list1, list2)
}

// HasParameter tells whether a given parameterName is the name of a valid parameter for this deserializer.
func (md *multiDeserializer[BasicPtr, HeaderPtr, _, _]) HasParameter(parameterName string) bool {
	if md == nil {
		return BasicPtr(nil).HasParameter(parameterName) || HeaderPtr(nil).HasParameter(parameterName)
	} else {
		return md.basicDeserializer.HasParameter(parameterName) || md.headerDeserializer.HasParameter(parameterName)
	}
}

// HasParameter tells whether a given parameterName is the name of a valid parameter for this serializer.
func (md *multiSerializer[BasicPtr, HeaderPtr, _, _]) HasParameter(parameterName string) bool {
	if md == nil {
		return BasicPtr(nil).HasParameter(parameterName) || HeaderPtr(nil).HasParameter(parameterName)
	} else {
		return md.basicSerializer.HasParameter(parameterName) || md.headerSerializer.HasParameter(parameterName)
	}
}

// WithParameter and GetParameter are complicated by the fact that we cannot struct-embed generic type parameters.

// WithParameter returns a new Deserializer with the parameter determined by parameterName changed to newParameter.
// The receiver is unchanged and the new Deserializer is independent.
//
// Note: This function will panic if the parameter does not exists or the new deserializer would have invalid parameters.
func (md *multiDeserializer[BasicPtr, HeaderPtr, BasicValue, HeaderValue]) WithParameter(parameterName string, newParameter any) CurvePointDeserializerModifyable {
	// Just for better error message
	if md == nil {
		panic(fmt.Errorf(ErrorPrefix+"Called WithParameter on nil receiver of type %T", md))
	}

	var ret multiDeserializer[BasicPtr, HeaderPtr, BasicValue, HeaderValue]

	// Check which component has the parameter. Note that it could be both
	foundBasic := md.basicDeserializer.HasParameter(parameterName)
	foundHeader := md.headerDeserializer.HasParameter(parameterName)

	// make sure the parameter actually exists.
	if !(foundBasic || foundHeader) {
		panic(fmt.Errorf(ErrorPrefix+"Trying to set parameter %v that does not exist for this deserializer.\nValid parameters are %v", parameterName, md.RecognizedParameters()))
	}

	// Case distinction: If the paramter is present in the component, use WithParameter, otherwise Clone()
	if foundBasic {
		ret.basicDeserializer = md.basicDeserializer.WithParameter(parameterName, newParameter)
	} else {
		ret.basicDeserializer = md.basicDeserializer.Clone() // NOTE: We could copy the pointer. However, that is error-prone
	}
	if foundHeader {
		ret.headerDeserializer = md.headerDeserializer.WithParameter(parameterName, newParameter)
	} else {
		ret.headerDeserializer = md.headerDeserializer.Clone() // NOTE: We could copy the pointer. However, that is error-prone
	}

	// We need to do that here, since we have additional validity constaint that go beyond each comonent being valid.
	ret.Validate()

	return &ret
}

// WithParameter returns a new Deserializer with the parameter determined by parameterName changed to newParameter.
// The receiver is unchanged and the new Deserializer is independent.
//
// Note: This function will panic if the parameter does not exists or the new deserializer would have invalid parameters.
func (md *multiSerializer[BasicPtr, HeaderPtr, BasicValue, HeaderValue]) WithParameter(parameterName string, newParameter any) CurvePointSerializerModifyable {
	// Just for better error message
	if md == nil {
		panic(fmt.Errorf(ErrorPrefix+"Called WithParameter on nil receiver of type %T", md))
	}

	var ret multiSerializer[BasicPtr, HeaderPtr, BasicValue, HeaderValue]

	// Check which component has the parameter. Note that it could be both
	foundBasic := md.basicSerializer.HasParameter(parameterName)
	foundHeader := md.headerSerializer.HasParameter(parameterName)

	// make sure the parameter actually exists.
	if !(foundBasic || foundHeader) {
		panic(fmt.Errorf(ErrorPrefix+"Trying to set parameter %v that does not exist for this serializer.\nValid parameters are %v", parameterName, md.RecognizedParameters()))
	}

	// Case distinction: If the paramter is present in the component, use WithParameter, otherwise Clone()
	if foundBasic {
		ret.basicSerializer = md.basicSerializer.WithParameter(parameterName, newParameter)
	} else {
		ret.basicSerializer = md.basicSerializer.Clone() // NOTE: We could copy the pointer. However, that is error-prone
	}
	if foundHeader {
		ret.headerSerializer = md.headerSerializer.WithParameter(parameterName, newParameter)
	} else {
		ret.headerSerializer = md.headerSerializer.Clone() // NOTE: We could copy the pointer. However, that is error-prone
	}

	// We need to do that here, since we have additional validity constaint that go beyond each comonent being valid.
	ret.Validate()

	return &ret
}

// GetParameter returns the value stored under the given parameterName for this deserializer.
//
// Note that this method panics if the parameterName is not valid. Check with HasParameter first, if needed.
func (md *multiDeserializer[BasicPtr, HeaderPtr, BasicValue, HeaderValue]) GetParameter(parameterName string) any {

	// Treat nil md as md containing nils. This may or may not work, depending on the involved types and on parameterName
	// Note that we change (the local variable) md, not *md -- this is not visible to the called.
	if md == nil {
		md = &multiDeserializer[BasicPtr, HeaderPtr, BasicValue, HeaderValue]{}
	}

	// If parameterName is contained in both components, preference is given to basicPointer

	if md.basicDeserializer.HasParameter(parameterName) {
		return md.basicDeserializer.GetParameter(parameterName)
	} else {
		return md.headerDeserializer.GetParameter(parameterName)
	}
}

// GetParameter returns the value stored under the given parameterName for this serializer.
//
// Note that this method panics if the parameterName is not valid. Check with HasParameter first, if needed.
func (md *multiSerializer[BasicPtr, HeaderPtr, BasicValue, HeaderValue]) GetParameter(parameterName string) any {

	// Treat nil md as md containing nils. This may or may not work, depending on the involved types and on parameterName
	// Note that we change (the local variable) md, not *md -- this is not visible to the called.
	if md == nil {
		md = &multiSerializer[BasicPtr, HeaderPtr, BasicValue, HeaderValue]{}
	}

	// If parameterName is contained in both components, preference is given to basicPointer
	if md.basicSerializer.HasParameter(parameterName) {
		return md.basicSerializer.GetParameter(parameterName)
	} else {
		return md.headerSerializer.GetParameter(parameterName)
	}
}

// WithEndianness returns a copy of the given deserializer with the endianness used for field element (de)serialization changed.
//
// NOTE: The endianness for (de)serialization of slice size headers is NOT affected by this method.
// NOTE2: newEndianness must be either literal binary.BigEndian, binary.LittleEndian or a common.FieldElementEndianness. In particular it must be non-nil. Failure to do so will panic.
func (md *multiDeserializer[_, _, _, _]) WithEndianness(newEndianness binary.ByteOrder) CurvePointDeserializerModifyable {

	if md == nil {
		panic(fmt.Errorf(ErrorPrefix+"Called WithEndianness on nil receiver of type %T", md))
	}
	return md.WithParameter("Endianness", newEndianness)
}

// WithEndianness returns a copy of the given serializer with the endianness used for field element (de)serialization changed.
//
// NOTE: The endianness for (de)serialization of slice size headers is NOT affected by this method.
// NOTE2: newEndianness must be either literal binary.BigEndian, binary.LittleEndian or a common.FieldElementEndianness. In particular it must be non-nil. Failure to do so will panic.
func (md *multiSerializer[_, _, _, _]) WithEndianness(newEndianness binary.ByteOrder) CurvePointSerializerModifyable {
	if md == nil {
		panic(fmt.Errorf(ErrorPrefix+"Called WithEndianness on nil receiver of type %T", md))
	}

	return md.WithParameter("Endianness", newEndianness)
}

// TODO: We would prefer to specify that on error, outputPoint is unchanged.
// Unfortunately, this is quite hard to achieve -- either we would need to store a backup copy of the point
// in case the footer deserializer returns an error or we would need to write to a temporary point.
// For the first solution: Unfortunately, the CurvePointPtr interface does not really allow to do that at the moment:
// The issue is that backing up the point with .Clone() and restoring with .SetFrom(...) is not really correct:
// outputPoint might be (in fact likely is!) a NaP and we have no guarantee that Clone() or SetFrom works for those.
// We would need a raw ToBytes / FromBytes interface here.
//
// Deserializing to a temporary and only writing to outputPoint on success has the problem that the dynamic type passed to
// basicDeserializer.DeserializeCurvePoint is relevant for this method's behaviour. So our temporary point would need to have the same dynamic type
// as outputPoint -- and we have no good way of doing that (Clone may not work for NaPs, reflection is not a "good way").
//
// -> TODO: Either add a raw ToBytes / FromBytes or specify that Clone works for NaPs.

// DeserializeCurvePoint deserializes a single curve point from input stream, (over-)writing to ouputPoint.
// trustLevel indicates whether the input is to be trusted that the data represents any (subgroup)point at all.
//
// On error, outputPoint may or may not be changed.
func (md *multiDeserializer[_, _, _, _]) DeserializeCurvePoint(inputStream io.Reader, trustLevel common.IsInputTrusted, outputPoint curvePoints.CurvePointPtrInterfaceWrite) (bytesRead int, err bandersnatchErrors.DeserializationError) {
	bytesRead, err = md.headerDeserializer.deserializeSinglePointHeader(inputStream)
	if err != nil {
		if bytesRead > 0 {
			errorTransform.UnexpectEOF2(&err) // not really doing anything, since bytesRead > 0 contradicts err is EOF.
			err = errorsWithData.NewErrorWithGuaranteedParameters[bandersnatchErrors.ReadErrorData](err, "%w", FIELDNAME_PARTIAL_READ, true)
		}
		return
	}

	// originalPoint := outputPoint.Clone() // needed to undo changes on error.

	bytesJustRead, err := md.basicDeserializer.DeserializeCurvePoint(inputStream, trustLevel, outputPoint)
	bytesRead += bytesJustRead
	if err != nil {
		// if we either read something before or there is data following, the correct error is UnexpectedEOF rather than EOF.
		if bytesRead != bytesJustRead || !md.headerDeserializer.trivialSinglePointFooter() {
			errorTransform.UnexpectEOF2(&err)
		}

		// If the footer is non-trivial, we always have a partial read.
		// So assume the footer is trivial. If err does not alreay contain the partial read_flag, bytesJustRead is either 0 or everything was read.
		// In the latter case, we really have no partial read; if bytesJustRead, we have a partial read situation if there was a header.
		if (bytesJustRead != 0 && bytesRead > 0) || !md.headerDeserializer.trivialSinglePointFooter() {
			err = errorsWithData.NewErrorWithGuaranteedParameters[bandersnatchErrors.ReadErrorData](err, "%w", FIELDNAME_PARTIAL_READ, true)
		}
		return
	}
	bytesJustRead, err = md.headerDeserializer.deserializeSinglePointFooter(inputStream)
	bytesRead += bytesJustRead
	if err != nil {
		errorTransform.UnexpectEOF2(&err)
		if bytesJustRead == 0 {
			err = errorsWithData.NewErrorWithGuaranteedParameters[bandersnatchErrors.ReadErrorData](err, "%w", FIELDNAME_PARTIAL_READ, true)
		}
		// outputPoint.SetFrom(originalPoint)
	}
	return
}

// DeserializeCurvePoint deserializes a single curve point from input stream, (over-)writing to ouputPoint.
// trustLevel indicates whether the input is to be trusted that the data represents any (subgroup)point at all.
//
// On error, outputPoint may or may not be changed.
func (md *multiSerializer[_, _, _, _]) DeserializeCurvePoint(inputStream io.Reader, trustLevel common.IsInputTrusted, outputPoint curvePoints.CurvePointPtrInterfaceWrite) (bytesRead int, err bandersnatchErrors.DeserializationError) {
	bytesRead, err = md.headerSerializer.deserializeSinglePointHeader(inputStream)
	if err != nil {
		if bytesRead > 0 {
			errorTransform.UnexpectEOF2(&err) // not really doing anything, since bytesRead > 0 contradicts err is EOF.
			err = errorsWithData.NewErrorWithGuaranteedParameters[bandersnatchErrors.ReadErrorData](err, "%w", FIELDNAME_PARTIAL_READ, true)
		}
		return
	}

	// originalPoint := outputPoint.Clone() // needed to undo changes on error.

	bytesJustRead, err := md.basicSerializer.DeserializeCurvePoint(inputStream, trustLevel, outputPoint)
	bytesRead += bytesJustRead
	if err != nil {
		// if we either read something before or there is data following, the correct error is UnexpectedEOF rather than EOF.
		if bytesRead != bytesJustRead || !md.headerSerializer.trivialSinglePointFooter() {
			errorTransform.UnexpectEOF2(&err)
		}

		// If the footer is non-trivial, we always have a partial read.
		// So assume the footer is trivial. If err does not alreay contain the partial read_flag, bytesJustRead is either 0 or everything was read.
		// In the latter case, we really have no partial read; if bytesJustRead, we have a partial read situation if there was a header.
		if (bytesJustRead != 0 && bytesRead > 0) || !md.headerSerializer.trivialSinglePointFooter() {
			err = errorsWithData.NewErrorWithGuaranteedParameters[bandersnatchErrors.ReadErrorData](err, "%w", FIELDNAME_PARTIAL_READ, true)
		}
		return
	}
	bytesJustRead, err = md.headerSerializer.deserializeSinglePointFooter(inputStream)
	bytesRead += bytesJustRead
	if err != nil {
		errorTransform.UnexpectEOF2(&err)
		if bytesJustRead == 0 {
			err = errorsWithData.NewErrorWithGuaranteedParameters[bandersnatchErrors.ReadErrorData](err, "%w", FIELDNAME_PARTIAL_READ, true)
		}
		// outputPoint.SetFrom(originalPoint)
	}
	return

}

// SerializeCurvePoint serializes the given input point to the outputStream.
func (md *multiSerializer[_, _, _, _]) SerializeCurvePoint(outputStream io.Writer, inputPoint curvePoints.CurvePointPtrInterfaceRead) (bytesWritten int, err bandersnatchErrors.SerializationError) {
	bytesWritten, err = md.headerSerializer.serializeSinglePointHeader(outputStream)
	if err != nil {
		if bytesWritten > 0 {
			errorTransform.UnexpectEOF2(&err) // does nothing, actually, because err cannot be EOF for bytesWritten > 0
			err = errorsWithData.NewErrorWithGuaranteedParameters[bandersnatchErrors.WriteErrorData](err, "%w", FIELDNAME_PARTIAL_WRITE, true)
		}
		return
	}
	bytesJustWritten, err := md.basicSerializer.SerializeCurvePoint(outputStream, inputPoint)
	bytesWritten += bytesJustWritten
	if err != nil {
		if bytesWritten > 0 && bytesWritten < int(md.OutputLength()) {
			errorTransform.UnexpectEOF2(&err)
			err = errorsWithData.NewErrorWithGuaranteedParameters[bandersnatchErrors.WriteErrorData](err, "%w", FIELDNAME_PARTIAL_WRITE, true)
		}
		return
	}
	bytesJustWritten, err = md.headerSerializer.serializeSinglePointFooter(outputStream)
	bytesWritten += bytesJustWritten
	if err != nil {
		if bytesWritten > 0 && bytesWritten < int(md.OutputLength()) {
			errorTransform.UnexpectEOF2(&err)
			err = errorsWithData.NewErrorWithGuaranteedParameters[bandersnatchErrors.WriteErrorData](err, "%w", FIELDNAME_PARTIAL_WRITE, true)
		}
	}
	return
}

// GetFieldElementEndianness returns the endianness this deserializer used to (de)serialize field elements.
//
// Note that the value is returned as a common.FieldElementEndianness, which is an interface extending binary.ByteOrder.
func (md *multiDeserializer[BasicPtr, _, _, _]) GetFieldElementEndianness() common.FieldElementEndianness {
	if md == nil {
		return BasicPtr(nil).GetEndianness()
	}
	return md.basicDeserializer.GetEndianness()
}

// GetFieldElementEndianness returns the endianness this serializer used to (de)serialize field elements.
//
// Note that the value is returned as a common.FieldElementEndianness, which is an interface extending binary.ByteOrder.
func (md *multiSerializer[BasicPtr, _, _, _]) GetFieldElementEndianness() common.FieldElementEndianness {
	if md == nil {
		return BasicPtr(nil).GetEndianness()
	}
	return md.basicSerializer.GetEndianness()
}

// IsSubgroupOnly returns whether this deserializer only works for subgroup elements.
func (md *multiDeserializer[BasicPtr, _, _, _]) IsSubgroupOnly() bool {
	if md == nil {
		return BasicPtr(nil).IsSubgroupOnly()
	}
	return md.basicDeserializer.IsSubgroupOnly()
}

// IsSubgroupOnly returns whether this serializer only works for subgroup elements.
func (md *multiSerializer[BasicPtr, _, _, _]) IsSubgroupOnly() bool {
	if md == nil {
		return BasicPtr(nil).IsSubgroupOnly()
	}
	return md.basicSerializer.IsSubgroupOnly()
}

// OutputLength returns an upper bound on the size (in bytes) that this deserializer will read when deserializing a single curve point.
//
// Note: We can only hope to get an upper bound, because a deserializer that is *not* also a serializer might work (and autodetect) multiple serialization formats;
// these different formats may have different lengths.
func (md *multiDeserializer[BasicPtr, HeaderPtr, _, _]) OutputLength() int32 {
	if md == nil {

		// These calls may well panic (especially the HeaderPtr one)
		actualPoint := BasicPtr(nil).OutputLength()
		overhead := HeaderPtr(nil).SinglePointHeaderOverhead()

		if int64(actualPoint)+int64(overhead) > math.MaxInt32 {
			panic(fmt.Errorf(ErrorPrefix+"OutputLength overflows int32 when called on nil receiver of type %T", md))
		}
	}
	// Validate ensures this does not overflow
	return md.headerDeserializer.SinglePointHeaderOverhead() + md.basicDeserializer.OutputLength()
}

// OutputLength returns the size (in bytes) that this serializer will read or write when (de)serializing a single curve point.
func (md *multiSerializer[BasicPtr, HeaderPtr, _, _]) OutputLength() int32 {
	if md == nil {

		// These calls may well panic (especially the HeaderPtr one)
		actualPoint := BasicPtr(nil).OutputLength()
		overhead := HeaderPtr(nil).SinglePointHeaderOverhead()

		if int64(actualPoint)+int64(overhead) > math.MaxInt32 {
			panic(fmt.Errorf(ErrorPrefix+"OutputLength overflows int32 when called on nil receiver of type %T", md))
		}
	}
	// Validate ensures this does not overflow
	return md.headerSerializer.SinglePointHeaderOverhead() + md.basicSerializer.OutputLength()
}

// SliceOutputLength returns the length in bytes that this deserializer will try to read at most if deserializing a slice of numPoints many points.
// Note that this is an upper bound (for the same reason as with OutputLength)
// error is set on int32 overflow.
func (md *multiDeserializer[_, _, _, _]) SliceOutputLength(numPoints int32) (int32, error) {
	// Get size used by the actual points (upper bound) and for the headers:
	var pointCost64 int64 = int64(numPoints) * int64(md.basicDeserializer.OutputLength()) // guaranteed to not overflow
	overhead, errOverhead := md.headerDeserializer.MultiPointHeaderOverhead(numPoints)    // overhead is what is used by the headers, including the size written in-band.

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
func (md *multiSerializer[_, _, _, _]) SliceOutputLength(numPoints int32) (int32, error) {

	// Get size used by the actual points (upper bound) and for the headers:
	var pointCost64 int64 = int64(numPoints) * int64(md.basicSerializer.OutputLength()) // guaranteed to not overflow
	overhead, errOverhead := md.headerSerializer.MultiPointHeaderOverhead(numPoints)    // overhead is what is used by the headers, including the size written in-band.

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
