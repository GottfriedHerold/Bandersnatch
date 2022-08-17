package pointserializer

import (
	"encoding/binary"
	"io"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/bandersnatchErrors"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/common"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/curvePoints"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/errorsWithData"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/fieldElements"
	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

// This file is part of the serialization-for-curve-points package.
// This package defines types that act as (de)serializers. These types hold metadata (such as e.g. endianness) about the serialization format.
// (De)serializers then have methods that are called with the actual curve point(s) as arguments to (de)serialize them.

// This file defines basic serializers that serialize and deserialize a single curve point.

// TODO: Allow serializing points at infinity.

// curvePointDeserializer_basic is a deserializer for single curve points
//
// Note that methods for parameter modification are contained in the generic modifyableSerializer interface.
// This is done in order to have the non-generic interface separate.
type curvePointDeserializer_basic interface {
	// DeserializeCurvePoint deserializes a single curve point from the inputStream. The output is written to output point.
	// TrustLevel determines whether we trust the input to be a valid representation of a curve point.
	// (The latter includes subgroup checks if outputPoint can only store subgroup points)
	// On error, outputPoint is kept unchanged.
	DeserializeCurvePoint(inputStream io.Reader, trustLevel common.IsInputTrusted, outputPoint curvePoints.CurvePointPtrInterfaceWrite) (bytesRead int, err bandersnatchErrors.DeserializationError)
	IsSubgroupOnly() bool // Can be called on nil pointers of concrete type. This indicates whether the deserializer is only for subgroup points.
	OutputLength() int32  // returns the length in bytes that this serializer will try to read/write per curve point. For deserializers without serializers, it is an upper bound.

	GetParameter(parameterName string) any        // obtains a parameter (such as endianness. parameterName is case-insensitive.
	GetEndianness() common.FieldElementEndianness // returns the endianness used for field element serialization.
	Validate()                                    // internal self-check of parameters; this is exported because of reflect usage
	RecognizedParameters() []string               // gives a list of recognized parameters
	HasParameter(parameterName string) bool       // checks whether a given parameter is recognized
}

// modifyableSerializer is the interface part contains the generic methods used to modify parameters.
// The relevant methods return a modified copy, whose type depends on the original, hence the need for generics.
type modifyableSerializer[SelfValue any, SelfPtr interface{ *SelfValue }] interface {
	WithParameter(parameterName string, newParam any) SelfValue // WithParameter returns an independent copy of the serializer with parameter given by paramName changed to newParam
	WithEndianness(newEndianness binary.ByteOrder) SelfValue    // WithEndianness is equivalent to WithParameter("Endianness, newEndianness)")
	utils.Clonable[SelfPtr]                                     // gives a Clone() SelfPtr function to make copies of itself

}

// modifyableDeserializer_basic is the interface for deserializer of single curve points that allow parameter modifications.
type modifyableDeserializer_basic[SelfValue any, SelfPtr interface{ *SelfValue }] interface {
	curvePointDeserializer_basic
	modifyableSerializer[SelfValue, SelfPtr]
}

// curvePointSerializer_basic is a serializer+deserializer for single curve points.
type curvePointSerializer_basic interface {
	curvePointDeserializer_basic
	SerializeCurvePoint(outputStream io.Writer, inputPoint curvePoints.CurvePointPtrInterfaceRead) (bytesWritten int, err bandersnatchErrors.SerializationError)
}

// modifyableDeserializer_basic is the interface for a serializer+deserializer of single curve points that allow parameter modifications.
type modifyableSerializer_basic[SelfValue any, SelfPtr interface{ *SelfValue }] interface {
	curvePointSerializer_basic
	modifyableSerializer[SelfValue, SelfPtr]
}

// Q: Separate into separate checks?

// checkPointSerializability verifies that the point is not a NaP or infinite. If performSubgroupCheck is set to true, also ensures that the point is in the p253-prime order subgroup.
// If everything is fine, returns nil. These correspond to the points that we usually want to serialize.
//
// Note: This function is is typically called before serializing (not for deserializing), where we do not have a trustLevel argument.
// This means that we always check whether the point is in the subgroup for any writes if the serializer is subgroup-only.
// Note for efficiency that this check is actually trivial if the type of point can only represent subgroup elements;
// we assume that this is the most common usage scenario.
func checkPointSerializability(point curvePoints.CurvePointPtrInterfaceRead, performSubgroupCheck bool) (err error) {
	if point.IsNaP() {
		err = bandersnatchErrors.ErrCannotSerializeNaP
		return
	}
	if point.IsAtInfinity() {
		err = bandersnatchErrors.ErrCannotSerializePointAtInfinity
		return
	}
	if performSubgroupCheck {
		if !point.IsInSubgroup() {
			err = bandersnatchErrors.ErrWillNotSerializePointOutsideSubgroup
			return
		}
	}
	return nil
}

// type alias to non-exported type for struct embedding

type subgroupRestriction = common.SubgroupRestriction
type subgroupOnly = common.SubgroupOnly

// addErrorDataNoWrite turns an arbitrary error into a SerializationError; the additional data added is trivial.
// this is supposed to be used if serialization fails before any io is even attempted.
func addErrorDataNoWrite(err error) bandersnatchErrors.SerializationError {
	if err == nil {
		return nil
	}
	return errorsWithData.NewErrorWithParametersFromData(err, "", &bandersnatchErrors.WriteErrorData{
		PartialWrite: false,
		BytesWritten: 0,
	})
}

// ***********************************************************************************************************************************************************

// we now define some "basic" serializers, basic being in the sense that they only allow (de)serializing a single point.
// They also do not allow headers (except possibly embedded)

// pointSerializerXY is a simple serializer that works by just writing / reading both the affine X and Y coordinates.
// If subgroupOnly is set to true, it will only work for points in the subgroup.
//
// NOTE: This cannot serialize points at infinity atm, even if subgroupRestriction is set to false
type pointSerializerXY struct {
	valuesSerializerHeaderFeHeaderFe
	subgroupRestriction // wraps a bool
}

// SerializeCurvePoint writes a single curve point to the given output.
// Since the output format relies on affine coordinates, this currently fails for points at infinity, which might change in the future.
//
// The format is X||Y for affine X and Y coordinates.
func (s *pointSerializerXY) SerializeCurvePoint(output io.Writer, point curvePoints.CurvePointPtrInterfaceRead) (bytesWritten int, err bandersnatchErrors.SerializationError) {
	var errPlain error = checkPointSerializability(point, s.IsSubgroupOnly())
	if errPlain != nil {
		err = addErrorDataNoWrite(errPlain)
		return
	}
	X, Y := point.XY_affine()
	bytesWritten, err = s.valuesSerializerHeaderFeHeaderFe.SerializeValues(output, &X, &Y)
	return
}

// DeserializeCurvePoint reads from input, interprets it and overwrites point.
// On error, point is untouched.
//
// The format is X||Y for affine X and Y coordinates.
func (s *pointSerializerXY) DeserializeCurvePoint(input io.Reader, trustLevel common.IsInputTrusted, point curvePoints.CurvePointPtrInterfaceWrite) (bytesRead int, err bandersnatchErrors.DeserializationError) {
	var X, Y fieldElements.FieldElement
	// var errPlain error
	bytesRead, err, X, Y = s.DeserializeValues(input)
	if err != nil {
		return
	}
	if s.IsSubgroupOnly() || point.CanOnlyRepresentSubgroup() {
		// using a temporary P here to ensure P is unchanged on error
		var P curvePoints.Point_axtw_subgroup
		P, errPlain := curvePoints.CurvePointFromXYAffine_subgroup(&X, &Y, trustLevel)
		if errPlain != nil {
			err = errorsWithData.NewErrorWithParametersFromData(errPlain, "", &bandersnatchErrors.ReadErrorData{
				PartialRead:  false,
				BytesRead:    int(s.OutputLength()),
				ActuallyRead: nil,
			})
			if trustLevel.Bool() {
				panic(err)
			}
			return
		}
		point.SetFrom(&P)
	} else {
		// using a temporary P here to ensure P is unchanged on error
		var P curvePoints.Point_axtw_full
		P, errPlain := curvePoints.CurvePointFromXYAffine_full(&X, &Y, trustLevel)
		if errPlain != nil {
			err = errorsWithData.NewErrorWithParametersFromData(errPlain, "", &bandersnatchErrors.ReadErrorData{
				PartialRead:  false,
				BytesRead:    int(s.OutputLength()),
				ActuallyRead: nil,
			})
			if trustLevel.Bool() {
				panic(err)
			}
			return
		}
		point.SetFrom(&P)
	}
	return
}

// Validate perfoms a self-check of the internal parameters stored for the given serializer.
// It panics on failure.
//
// Note that users are not expected to call this; this is provided for internal usage to unify parameter setting functions.
// It is exported for cross-package/reflect usage.
func (s *pointSerializerXY) Validate() {
	s.valuesSerializerHeaderFeHeaderFe.Validate()
	s.subgroupRestriction.Validate()
}

// Clone creates an independent copy of the received serializer, returning a pointer.
//
// Note that since serializers are immutable, library users should never need to call this;
// this is an internal function that is exported due to cross-package and reflect usage.
func (s *pointSerializerXY) Clone() (ret *pointSerializerXY) {
	var sCopy pointSerializerXY = *s
	ret = &sCopy
	return
}

// WithParameter(param, newParam) creates a modified copy of the received serializer with the parameter determined by param replaced by newParam.
// Invalid inputs cause a panic.
//
// Recognized params are: "Endianness", "SubgroupOnly", "BitHeader", "BitHeader2"
func (s *pointSerializerXY) WithParameter(param string, newParam interface{}) (newSerializer pointSerializerXY) {
	return makeCopyWithParameters(s, param, newParam)
}

// WithEndianness creates a modified copy of the received serializer with the prescribed endianness for field element serialization.
// It accepts only literal binary.LittleEndian, binary.BigEndian or any newEndianness satisfying the common.FieldElementEnianness interface (which extends binary.ByteOrder).
//
// Invalid inputs cause a panic.
func (s *pointSerializerXY) WithEndianness(newEndianness binary.ByteOrder) pointSerializerXY {
	return s.WithParameter("Endianness", newEndianness)
}

// OutputLength returns the number of bytes read/written per curve point.
//
// It returns 64 for this serializer type.
func (s *pointSerializerXY) OutputLength() int32 { return 64 }

// GetParameter returns the value of the internal parameter determined by parameterName
//
// recognized parameterNames are: "Endianness", "SubgroupOnly"., "BitHeader", "BitHeader2"
func (s *pointSerializerXY) GetParameter(parameterName string) interface{} {
	return getSerializerParam(s, parameterName)
}

func (s *pointSerializerXY) RecognizedParameters() []string {
	return concatParameterList(s.valuesSerializerHeaderFeHeaderFe.RecognizedParameters(), s.subgroupRestriction.RecognizedParameters())
}

func (s *pointSerializerXY) HasParameter(parameterName string) bool {
	return utils.ElementInList(parameterName, s.RecognizedParameters(), normalizeParameter)
}

// ***********************************************************************************************************************************************************

// pointSerializerXAndSignY is a Serialializer that serializes the affine X coordinate and the sign of the Y coordinate. (Note that the latter is never 0)
//
// More precisely, we write a 1 bit into the msb of the output (if interpreteed as 256bit-number) if the sign of Y is negative.
type pointSerializerXAndSignY struct {
	valuesSerializerFeCompressedBit
	subgroupRestriction
}

// SerializeCurvePoint writes a single curve point to the given output.
// Since the output format relies on affine coordinates, this currently fails for points at infinity, which might change in the future.
//
// The format written is Sign(Y)||X, with the sign bit (1 for negative, 0 for positive) embedded in the msb of X to save space.
func (s *pointSerializerXAndSignY) SerializeCurvePoint(output io.Writer, point curvePoints.CurvePointPtrInterfaceRead) (bytesWritten int, err bandersnatchErrors.SerializationError) {
	var errPlain error = checkPointSerializability(point, s.IsSubgroupOnly())
	if errPlain != nil {
		bytesWritten = 0
		err = addErrorDataNoWrite(errPlain)
		return
	}

	X, Y := point.XY_affine()
	var SignY bool = Y.Sign() < 0 // canot be == 0
	bytesWritten, err = s.SerializeValues(output, &X, SignY)
	return
}

// DeserializeCurvePoint reads from input, interprets it and overwrites point.
// On error, point is untouched.
//
// The format expected is Sign(Y)||X, with the sign bit (1 for negative, 0 for positive) embedded in the msb of X.
func (s *pointSerializerXAndSignY) DeserializeCurvePoint(input io.Reader, trustLevel common.IsInputTrusted, point curvePoints.CurvePointPtrInterfaceWrite) (bytesRead int, err bandersnatchErrors.DeserializationError) {
	var X fieldElements.FieldElement
	var signBit bool
	bytesRead, err, X, signBit = s.DeserializeValues(input)
	if err != nil {
		return
	}

	//  convert boolean sign bit to +/-1 - valued sign
	var signInt int
	if signBit {
		signInt = -1
	} else {
		signInt = +1
	}

	if s.IsSubgroupOnly() || point.CanOnlyRepresentSubgroup() {
		var P curvePoints.Point_axtw_subgroup
		P, errCurvePoint := curvePoints.CurvePointFromXAndSignY_subgroup(&X, signInt, trustLevel)
		if errCurvePoint != nil {
			err = errorsWithData.NewErrorWithParametersFromData(errCurvePoint, "%w", &bandersnatchErrors.ReadErrorData{
				PartialRead:  false,
				BytesRead:    int(s.OutputLength()),
				ActuallyRead: nil,
			})
			if trustLevel.Bool() {
				panic(err) // should not happen, because CurvePointFromXAndSignY panics.
			}

			return
		}
		point.SetFrom(&P)
	} else {
		var P curvePoints.Point_axtw_full
		P, errCurvePoint := curvePoints.CurvePointFromXAndSignY_full(&X, signInt, trustLevel)
		if errCurvePoint != nil {
			err = errorsWithData.NewErrorWithParametersFromData(errCurvePoint, "%w", &bandersnatchErrors.ReadErrorData{
				PartialRead:  false,
				BytesRead:    int(s.OutputLength()),
				ActuallyRead: nil,
			})
			if trustLevel.Bool() {
				panic(err) // should not happen, because CurvePointFromXAndSignY panics.
			}
			return
		}
		point.SetFrom(&P)
	}
	return
}

// Clone creates an independent copy of the received serializer, returning a pointer.
//
// Note that since serializers are immutable, library users should never need to call this;
// this is an internal function that is exported due to cross-package and reflect usage.
func (s *pointSerializerXAndSignY) Clone() (ret *pointSerializerXAndSignY) {
	var sCopy pointSerializerXAndSignY = *s
	ret = &sCopy
	return
}

// WithParameter(param, newParam) creates a modified copy of the received serializer with the parameter determined by param replaced by newParam.
// Invalid input cause a panic.
//
// Recognized params are: "Endianness", "SubgroupOnly"
func (s *pointSerializerXAndSignY) WithParameter(param string, newParam interface{}) (newSerializer pointSerializerXAndSignY) {
	return makeCopyWithParameters(s, param, newParam)
}

// WithEndianness creates a modified copy of the received serializer with the prescribed endianness for field element serialization.
// It accepts only literal binary.LittleEndian, binary.BigEndian or any newEndianness satisfying the common.FieldElementEnianness interface (which extends binary.ByteOrder).
//
// Invalid inputs cause a panic.
func (s *pointSerializerXAndSignY) WithEndianness(newEndianness binary.ByteOrder) pointSerializerXAndSignY {
	return s.WithParameter("endianness", newEndianness)
}

// OutputLength returns the number of bytes read/written per curve point.
//
// It returns 32 for this serializer type.
func (s *pointSerializerXAndSignY) OutputLength() int32 { return 32 }

// GetParameter returns the value of the internal parameter determined by parameterName
//
// recognized parameterNames are: "Endianness", "SubgroupOnly".
// GetParameter returns the value of the given parameterName.
//
// Accepted values for parameterName are "Endiannness", "SubgroupOnly"
func (s *pointSerializerXAndSignY) GetParameter(parameterName string) interface{} {
	return getSerializerParam(s, parameterName)
}

// Validate perfoms a self-check of the internal parameters stored for the given serializer.
// It panics on failure.
//
// Note that users are not expected to call this; this is provided for internal usage to unify parameter setting functions.
// It is exported for cross-package/reflect usage.
func (s *pointSerializerXAndSignY) Validate() {
	s.valuesSerializerFeCompressedBit.Validate()
	s.fieldElementEndianness.Validate()
}

func (s *pointSerializerXAndSignY) RecognizedParameters() []string {
	return concatParameterList(s.valuesSerializerFeCompressedBit.RecognizedParameters(), s.subgroupRestriction.RecognizedParameters())
}

func (s *pointSerializerXAndSignY) HasParameter(parameterName string) bool {
	return utils.ElementInList(parameterName, s.RecognizedParameters(), normalizeParameter)
}

// ***********************************************************************************************************************************************************

// pointSerializerYAndSignX serializes a point via its Y coordinate and the sign of X. (For X==0, we do not set the sign bit)
type pointSerializerYAndSignX struct {
	valuesSerializerFeCompressedBit
	subgroupRestriction
}

// Validate perfoms a self-check of the internal parameters stored for the given serializer.
// It panics on failure.
//
// Note that users are not expected to call this; this is provided for internal usage to unify parameter setting functions.
// It is exported for cross-package/reflect usage.
func (s *pointSerializerYAndSignX) Validate() {
	s.valuesSerializerFeCompressedBit.Validate()
	s.subgroupRestriction.Validate()
}

// SerializeCurvePoint writes a single curve point to the given output.
// Since the output format relies on affine coordinates, this currently fails for points at infinity, which might change in the future.
//
// The output format is Sign(X)||Y, where Sign(X) is a bit (set iff X<0) stored inside the msb of Y for compression.
func (s *pointSerializerYAndSignX) SerializeCurvePoint(output io.Writer, point curvePoints.CurvePointPtrInterfaceRead) (bytesWritten int, err bandersnatchErrors.SerializationError) {
	errPlain := checkPointSerializability(point, s.IsSubgroupOnly())
	if errPlain != nil {
		err = addErrorDataNoWrite(errPlain)
		bytesWritten = 0
		return
	}
	X, Y := point.XY_affine()
	var SignX bool = X.Sign() < 0 // for X==0, we want the sign bit to be NOT set.
	bytesWritten, err = s.SerializeValues(output, &Y, SignX)
	return
}

// DeserializeCurvePoint reads from input, interprets it and overwrites point.
// On error, point is untouched.
//
// The format expected is Sign(X)||Y, where Sign(X) is a bit (0b1 iff X<0) stored inside the msb of Y for compression.
func (s *pointSerializerYAndSignX) DeserializeCurvePoint(input io.Reader, trustLevel common.IsInputTrusted, point curvePoints.CurvePointPtrInterfaceWrite) (bytesRead int, err bandersnatchErrors.DeserializationError) {
	var Y fieldElements.FieldElement
	var signBit bool
	bytesRead, err, Y, signBit = s.DeserializeValues(input)
	if err != nil {
		return
	}
	var signInt int
	if signBit {
		signInt = -1
	} else {
		signInt = +1
	}

	errData := bandersnatchErrors.ReadErrorData{PartialRead: false, BytesRead: int(s.OutputLength()), ActuallyRead: nil}

	// Note: CurvePointFromYAndSignX_* accepts any sign for Y=+/-1.
	// We need to correct this to ensure uniqueness of the serialized representation.

	if s.IsSubgroupOnly() || point.CanOnlyRepresentSubgroup() {
		var P curvePoints.Point_axtw_subgroup
		P, errConvertToPoint := curvePoints.CurvePointFromYAndSignX_subgroup(&Y, signInt, trustLevel)
		if errConvertToPoint != nil {
			err = errorsWithData.NewErrorWithParametersFromData(errConvertToPoint, "", utils.AddressOfCopy(errData))
			if trustLevel.Bool() {
				panic(err) // not supposed to be reachable
			}
			return
		}

		// Handle Y = +/- 1 cases:
		// Y = -1 is already accounted for (not in subgroup)
		// For Y = +1, we only accept signBit = false, as that's what we write when serializing.
		if P.IsNeutralElement() && signBit {
			err = errorsWithData.NewErrorWithParametersFromData(bandersnatchErrors.ErrUnexpectedNegativeZero, "", utils.AddressOfCopy(errData))
			if trustLevel.Bool() {
				panic(err) // This is actually reachable
			}
			return
		}

		point.SetFrom(&P)
	} else {
		var P curvePoints.Point_axtw_full
		P, errConvertToPoint := curvePoints.CurvePointFromYAndSignX_full(&Y, signInt, trustLevel)
		if errConvertToPoint != nil {
			err = errorsWithData.NewErrorWithParametersFromData(errConvertToPoint, "", utils.AddressOfCopy(errData))
			if trustLevel.Bool() {
				panic(err) // not supposed to be reachable
			}
			return
		}

		// Special case for Y = +/-1: We have X=0. In that case, we only accept signBit = false, as that's what we write when serializing.
		{
			var X fieldElements.FieldElement = P.X_decaf_affine()
			if X.IsZero() && signBit {
				err = errorsWithData.NewErrorWithParametersFromData(bandersnatchErrors.ErrUnexpectedNegativeZero, "", utils.AddressOfCopy(errData))
				if trustLevel.Bool() {
					panic(err) // This is actually reachable
				}
				return
			}
		}
		point.SetFrom(&P)
	}
	return
}

// Clone creates an independent copy of the received serializer, returning a pointer.
//
// Note that since serializers are immutable, library users should never need to call this;
// this is an internal function that is exported due to cross-package and reflect usage.
func (s *pointSerializerYAndSignX) Clone() (ret *pointSerializerYAndSignX) {
	var sCopy pointSerializerYAndSignX
	sCopy.fieldElementEndianness = s.fieldElementEndianness
	sCopy.subgroupRestriction = s.subgroupRestriction
	ret = &sCopy
	return
}

// WithParameter(param, newParam) creates a modified copy of the received serializer with the parameter determined by param replaced by newParam.
//
// Recognized params are: "Endianness", "SubgroupOnly"
func (s *pointSerializerYAndSignX) WithParameter(param string, newParam interface{}) (newSerializer pointSerializerYAndSignX) {
	return makeCopyWithParameters(s, param, newParam)
}

// WithEndianness creates a modified copy of the received serializer with the prescribed endianness for field element serialization.
// It accepts only literal binary.LittleEndian, binary.BigEndian or any newEndianness satisfying the common.FieldElementEnianness interface (which extends binary.ByteOrder).
//
// Invalid inputs cause a panic.
func (s *pointSerializerYAndSignX) WithEndianness(newEndianness binary.ByteOrder) pointSerializerYAndSignX {
	return s.WithParameter("endianness", newEndianness)
}

// OutputLength returns the number of bytes read/written per curve point.
//
// It returns 32 for this serializer type.
func (s *pointSerializerYAndSignX) OutputLength() int32 { return 32 }

// GetParameter returns the value of the internal parameter determined by parameterName
//
// recognized parameterNames are: "Endianness", "SubgroupOnly".
func (s *pointSerializerYAndSignX) GetParameter(parameterName string) interface{} {
	return getSerializerParam(s, parameterName)
}

func (s *pointSerializerYAndSignX) RecognizedParameters() []string {
	return concatParameterList(s.valuesSerializerFeCompressedBit.RecognizedParameters(), s.subgroupRestriction.RecognizedParameters())
}

func (s *pointSerializerYAndSignX) HasParameter(parameterName string) bool {
	return utils.ElementInList(parameterName, s.RecognizedParameters(), normalizeParameter)
}

// ***********************************************************************************************************************************************************

// pointSerializerXTimesSignY is a basic serializer that serializes via X * Sign(Y).
// Note that this only works for points in the subgroup, as the information of being in the subgroup
// is needed to deserialize uniquely.
type pointSerializerXTimesSignY struct {
	valuesSerializerHeaderFe
	subgroupOnly
}

// Validate perfoms a self-check of the internal parameters stored for the given serializer.
// It panics on failure.
//
// Note that users are not expected to call this; this is provided for internal usage to unify parameter setting functions.
// It is exported for cross-package/reflect usage.
func (s *pointSerializerXTimesSignY) Validate() {
	s.valuesSerializerHeaderFe.Validate()
	s.subgroupOnly.Validate()
}

// SerializeCurvePoint writes a single curve point to the given output.
// Since the output format relies on affine coordinates, this currently fails for points at infinity, which might change in the future.
//
// The format written is X*Sign(Y), where Sign(Y) is +1 or -1.
func (s *pointSerializerXTimesSignY) SerializeCurvePoint(output io.Writer, point curvePoints.CurvePointPtrInterfaceRead) (bytesWritten int, err bandersnatchErrors.SerializationError) {
	errPlain := checkPointSerializability(point, true)
	if errPlain != nil {
		err = addErrorDataNoWrite(errPlain)
		bytesWritten = 0
		return
	}
	X := point.X_decaf_affine()
	Y := point.Y_decaf_affine()
	var SignY int = Y.Sign()
	if SignY < 0 {
		X.NegEq()
	}
	bytesWritten, err = s.SerializeValues(output, &X)
	return
}

// DeserializeCurvePoint reads from input, interprets it and overwrites point.
// On error, point is untouched.
//
// The format expected is X*Sign(Y), where Sign(Y) is +1 or -1.
func (s *pointSerializerXTimesSignY) DeserializeCurvePoint(input io.Reader, trustLevel common.IsInputTrusted, point curvePoints.CurvePointPtrInterfaceWrite) (bytesRead int, err bandersnatchErrors.DeserializationError) {
	var XSignY fieldElements.FieldElement
	bytesRead, err, XSignY = s.DeserializeValues(input)
	if err != nil {
		return
	}
	var P curvePoints.Point_axtw_subgroup
	P, errConversionToCurvePoint := curvePoints.CurvePointFromXTimesSignY_subgroup(&XSignY, trustLevel)
	if errConversionToCurvePoint != nil {
		err = errorsWithData.NewErrorWithParametersFromData(errConversionToCurvePoint, "%w", &bandersnatchErrors.ReadErrorData{
			PartialRead:  false,
			BytesRead:    int(s.OutputLength()),
			ActuallyRead: nil,
		})
		if trustLevel.Bool() {
			panic(err) // not supposed to be reachable
		}
		return
	}
	point.SetFrom(&P)
	return
}

// Clone creates an independent copy of the received serializer, returning a pointer.
//
// Note that since serializers are immutable, library users should never need to call this;
// this is an internal function that is exported due to cross-package and reflect usage.
func (s *pointSerializerXTimesSignY) Clone() (ret *pointSerializerXTimesSignY) {
	var sCopy pointSerializerXTimesSignY = *s
	return &sCopy
}

// WithParameter(param, newParam) creates a modified copy of the received serializer with the parameter determined by param replaced by newParam.
//
// Recognized params are: "Endianness", "SubgroupOnly"
// Note that "SubgroupOnly" only accepts true.
func (s *pointSerializerXTimesSignY) WithParameter(param string, newParam interface{}) (newSerializer pointSerializerXTimesSignY) {
	return makeCopyWithParameters(s, param, newParam)
}

// WithEndianness creates a modified copy of the received serializer with the prescribed endianness for field element serialization.
// It accepts only literal binary.LittleEndian, binary.BigEndian or any newEndianness satisfying the common.FieldElementEnianness interface (which extends binary.ByteOrder).
//
// Invalid inputs cause a panic.
func (s *pointSerializerXTimesSignY) WithEndianness(newEndianness binary.ByteOrder) pointSerializerXTimesSignY {
	return s.WithParameter("endianness", newEndianness)
}

// OutputLength returns the number of bytes read/written per curve point.
//
// It returns 32 for this serializer type.
func (s *pointSerializerXTimesSignY) OutputLength() int32 { return 32 }

// GetParameter returns the value of the internal parameter determined by parameterName
//
// recognized parameterNames are: "Endianness", "SubgroupOnly".
func (s *pointSerializerXTimesSignY) GetParameter(parameterName string) interface{} {
	return getSerializerParam(s, parameterName)
}

func (s *pointSerializerXTimesSignY) RecognizedParameters() []string {
	return concatParameterList(s.valuesSerializerHeaderFe.RecognizedParameters(), s.subgroupOnly.RecognizedParameters())
}

func (s *pointSerializerXTimesSignY) HasParameter(parameterName string) bool {
	return utils.ElementInList(parameterName, s.RecognizedParameters(), normalizeParameter)
}

// ***********************************************************************************************************************************************************

// pointSerializerYXTimesSignY is a serializer that uses X*Sign(Y), Y*Sign(Y).
// This serializer only works for subgroup elements:
// The fact of being in the subgroup is needed to uniquely deserialize.
type pointSerializerYXTimesSignY struct {
	valuesSerializerHeaderFeHeaderFe
	subgroupOnly
}

// Validate perfoms a self-check of the internal parameters stored for the given serializer.
// It panics on failure.
//
// Note that users are not expected to call this; this is provided for internal usage to unify parameter setting functions.
// It is exported for cross-package/reflect usage.
func (s *pointSerializerYXTimesSignY) Validate() {
	s.valuesSerializerHeaderFeHeaderFe.Validate()
	s.subgroupOnly.Validate()
}

// SerializeCurvePoint writes a single curve point to the given output.
// Since the output format relies on affine coordinates, this currently fails for points at infinity, which might change in the future.
//
// The format written is Y*Sign(Y)||X*Sign(Y), where  Sign(Y) = +1 or -1
func (s *pointSerializerYXTimesSignY) SerializeCurvePoint(output io.Writer, point curvePoints.CurvePointPtrInterfaceRead) (bytesWritten int, err bandersnatchErrors.SerializationError) {
	errPlain := checkPointSerializability(point, true)
	if errPlain != nil {
		err = addErrorDataNoWrite(errPlain)
		bytesWritten = 0
		return
	}
	X := point.X_decaf_affine()
	Y := point.Y_decaf_affine()
	var SignY int = Y.Sign()
	if SignY < 0 {
		X.NegEq()
		Y.NegEq()
	}
	bytesWritten, err = s.SerializeValues(output, &Y, &X)
	return
}

// DeserializeCurvePoint reads from input, interprets it and overwrites point.
// On error, point is untouched.
//
// The format expected is Y*Sign(Y)||X*Sign(Y), with Sign(Y)=+1 or -1.
func (s *pointSerializerYXTimesSignY) DeserializeCurvePoint(input io.Reader, trustLevel common.IsInputTrusted, point curvePoints.CurvePointPtrInterfaceWrite) (bytesRead int, err bandersnatchErrors.DeserializationError) {
	var XSignY, YSignY fieldElements.FieldElement
	bytesRead, err, YSignY, XSignY = s.DeserializeValues(input)
	if err != nil {
		return
	}

	var P curvePoints.Point_axtw_subgroup
	P, errConversionToCurvePoint := curvePoints.CurvePointFromXYTimesSignY_subgroup(&XSignY, &YSignY, trustLevel)
	if errConversionToCurvePoint != nil {
		err = errorsWithData.NewErrorWithParametersFromData(errConversionToCurvePoint, "", &bandersnatchErrors.ReadErrorData{
			PartialRead:  false,
			BytesRead:    int(s.OutputLength()),
			ActuallyRead: nil,
		})
		if trustLevel.Bool() {
			panic(err) // not supposed to be reachable
		}
		return
	}
	point.SetFrom(&P)
	/*
		-- removed : P's type ensures this
		ok := point.SetFromSubgroupPoint(&P, bandersnatch.TrustedInput) // P is trusted at this point
		if !ok {
			// This is supposed to be impossible to happen (unless the user lied wrt trusted-ness of input)
			// Actually, even then, SetFromSubgroupPoint does not make checks for trusted input, so it ought to be unreachable; of course, this depends on the dynamic type of point, so don't
			// want to make this assumption.
			// The error message is unspecific, because we can not guarantee that the previous steps produced valid outputs.
			panic(fmt.Errorf(ErrorPrefix+"When deserializing trusted input from (X,Y)*SignOfY, an unexpected error happened during conversion to the desired curve point. XSignY = %v, YSignY = %v", XSignY, YSignY))
		}
	*/
	return
}

// Clone creates an independent copy of the received serializer, returning a pointer.
//
// Note that since serializers are immutable, library users should never need to call this;
// this is an internal function that is exported due to cross-package and reflect usage.
func (s *pointSerializerYXTimesSignY) Clone() (ret *pointSerializerYXTimesSignY) {
	var sCopy pointSerializerYXTimesSignY = *s
	ret = &sCopy
	return
}

// WithParameter(param, newParam) creates a modified copy of the received serializer with the parameter determined by param replaced by newParam.
//
// Recognized params are: "Endianness", "SubgroupOnly"
// Note that SubgroupOnly only accepts true.
func (s *pointSerializerYXTimesSignY) WithParameter(param string, newParam interface{}) (newSerializer pointSerializerYXTimesSignY) {
	return makeCopyWithParameters(s, param, newParam)
}

// WithEndianness creates a modified copy of the received serializer with the prescribed endianness for field element serialization.
// It accepts only literal binary.LittleEndian, binary.BigEndian or any newEndianness satisfying the common.FieldElementEnianness interface (which extends binary.ByteOrder).
//
// Invalid inputs cause a panic.
func (s *pointSerializerYXTimesSignY) WithEndianness(newEndianness binary.ByteOrder) pointSerializerYXTimesSignY {
	return s.WithParameter("endianness", newEndianness)
}

// OutputLength returns the number of bytes read/written per curve point.
//
// It returns 64 for this serializer type.
func (s *pointSerializerYXTimesSignY) OutputLength() int32 { return 64 }

// GetParameter returns the value of the internal parameter determined by parameterName
//
// recognized parameterNames are: "Endianness", "SubgroupOnly".
func (s *pointSerializerYXTimesSignY) GetParameter(parameterName string) interface{} {
	return getSerializerParam(s, parameterName)
}

func (s *pointSerializerYXTimesSignY) RecognizedParameters() []string {
	return concatParameterList(s.valuesSerializerHeaderFeHeaderFe.RecognizedParameters(), s.subgroupOnly.RecognizedParameters())
}

func (s *pointSerializerYXTimesSignY) HasParameter(parameterName string) bool {
	return utils.ElementInList(parameterName, s.RecognizedParameters(), normalizeParameter)
}

// ***********************************************************************************************************************************************************

// Note: This selection of bitHeaders is not the original spec, but this has the advantage that an all-zeroes input actually causes an error (rather than be interpreted as the neutral element)

var bitHeaderBanderwagonX common.BitHeader = common.MakeBitHeader(common.PrefixBits(0b1), 1)
var bitHeaderBanderwagonY common.BitHeader = common.MakeBitHeader(common.PrefixBits(0b00), 2)

var basicBanderwagonShort = pointSerializerXTimesSignY{valuesSerializerHeaderFe: valuesSerializerHeaderFe{fieldElementEndianness: common.DefaultEndian, bitHeader: bitHeaderBanderwagonX}, subgroupOnly: subgroupOnly{}}
var basicBanderwagonLong = pointSerializerYXTimesSignY{valuesSerializerHeaderFeHeaderFe: valuesSerializerHeaderFeHeaderFe{fieldElementEndianness: common.DefaultEndian, bitHeader: bitHeaderBanderwagonY, bitHeader2: bitHeaderBanderwagonX}, subgroupOnly: subgroupOnly{}}

func init() {
	bitHeaderBanderwagonX.Validate()
	bitHeaderBanderwagonY.Validate()
	basicBanderwagonShort.Validate()
	basicBanderwagonLong.Validate()
}
