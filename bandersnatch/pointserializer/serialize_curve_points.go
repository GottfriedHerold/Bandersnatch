package pointserializer

import (
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/bandersnatchErrors"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/common"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/curvePoints"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/errorsWithData"
	"github.com/GottfriedHerold/Bandersnatch/internal/errorTransform"
	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
)

// BatchSerializationErrorData is the struct that hold additional data contained in errors reported by batch serialization methods.
// This data is obtainable via the [errorsWithData] framework.
// Concretely, we extend our usual serialization error data by an int field indication how many points were actually fully serialized before the error occurred.
type BatchSerializationErrorData struct {
	bandersnatchErrors.WriteErrorData     // Note [errorsWithData]'s behaviour for struct embedding
	PointsSerialized                  int // Number of points fully serialized
}

// BatchDeserializationErrorData is the struct that hold additional data contained in errors reported by batch deserialization methods.
// This data is obtainable via the [errorsWithData] framework.
// Concretely, we extend our usual deserialization error data by an int field indicating how many points were actually fully deserialized before the error occurred.
// Note that in this context, "fully deserialized" includes potential subgroup and validity checks and actually writing to the target buffer.
// We guarantee that this is equal to the number of buffer elements that were written to.
type BatchDeserializationErrorData struct {
	bandersnatchErrors.ReadErrorData     // Note [errorsWithData]'s behaviour for struct embedding
	PointsDeserialized               int // Number of points fully deserialized
}

const FIELDNAME_POINTSDESERIALIZED = "PointsDeserialized"
const FIELDNAME_POINTSSERIALIZED = "PointsSerialized"

func init() {
	errorsWithData.CheckParameterForStruct[BatchDeserializationErrorData](FIELDNAME_POINTSDESERIALIZED)
	errorsWithData.CheckParameterForStruct[BatchSerializationErrorData](FIELDNAME_POINTSSERIALIZED)
	errorsWithData.CheckParameterForStruct[BatchDeserializationErrorData]("PointsDeserialized")
	errorsWithData.CheckParameterForStruct[BatchSerializationErrorData]("PointsSerialized")
}

// BatchSerializationError is the error type returned by Serialization methods that serialize multiple points at once.
// errors of this type contain an instance of [BatchSerializationErrorData].
type BatchSerializationError = errorsWithData.ErrorWithGuaranteedParameters[BatchSerializationErrorData]

// BatchDeserializationError is the error type returned by Deserialization methods that deserialize multiple points at once.
// errors of this type contain an instance of [BatchDeserializationErrorData].
type BatchDeserializationError = errorsWithData.ErrorWithGuaranteedParameters[BatchDeserializationErrorData]

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
// On error, the returned error of type [BatchDeserializationError] contains (among other data) via the [errorsWithData] framework fields PointsDeserialized and PartialRead.
//
// PointsDeserialized is the number of points that were *successfully* deserialized (i.e. actually written to outputPoints).
// If we read from the inputStream, but do not write because the read data fails a subgroup check, this is not counted in PointsDeserialized.
// PartialRead is set to true if we encountered a read error that is not aligned with data encoding points.
//
// NOTE: If you have a slice buf of type []PointType to hold the output, call this method with curvePoints.AsCurvePointSlice(buf) to create a view of buf with the appropriate type.
// Be aware that whether PointType is restricted to points in the subgroup or not may control whether we perform subgroup checks on untrusted deserialization!
// (If the serializer only works for subgroup elements anyway, PointType is ignored)
//
// NOTE: When using this method to deserialize AT MOST L points into a buffer, but without knowing how many points are in the stream,
// you need to check that the error wraps either io.EOF or io.UnexpectedEOF and PartialRead is false.
// We provide a convenience function [DeserializeCurvePoints_Bounded] that handles this case.
// We also provide a convenience variadic version [DeserialiveCurvePoints_Variadic].
// These are both functions, not methods (due to using generics).
func (md *multiDeserializer[_, _, _, _]) DeserializeCurvePoints(inputStream io.Reader, trustLevel IsInputTrusted, outputPoints curvePoints.CurvePointSlice) (bytesRead int, err BatchDeserializationError) {
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
				errorTransform.UnexpectEOF2(&errSingle)
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
// NOTE: When using this method to deserialize AT MOST L points into a buffer, but don't know how many points are in the stream,
// you need to check that the error wraps either io.EOF or io.UnexpectedEOF and that PartialRead is false.
// We provide a convenience function DeserializeCurvePoints_Bounded that handles this case.
// We also provide a convenience variadic version DeserialiveCurvePoints_Variadic.
// These are both functions, not methods.
func (md *multiSerializer[_, _, _, _]) DeserializeCurvePoints(inputStream io.Reader, trustLevel common.IsInputTrusted, outputPoints curvePoints.CurvePointSlice) (bytesRead int, err BatchDeserializationError) {
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
				errorTransform.UnexpectEOF2(&errSingle)
			}

			// the index i gives the correct value for the PointsDeserialized error data. The other data (including PartialRead) is actually correct.
			err = errorsWithData.NewErrorWithGuaranteedParameters[BatchDeserializationErrorData](errSingle,
				ErrorPrefix+"batch deserialization failed after deserializing %{PointsDeserialized} points with error %w",
				"PointsDeserialized", i)
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

func deserializeSlice_mainloop(inputStream io.Reader, trustLevel common.IsInputTrusted, targetSlice curvePoints.CurvePointSlice, deserializer_header headerDeserializerInterface, deserializer_point curvePointDeserializer_basic, size32 int32) (bytesRead int, err BatchDeserializationError) {
	var bytesJustRead int
	var errNonBatch bandersnatchErrors.DeserializationError
	size := int(size32) // i in the loop below should be int (because of type-unsafe inclusion in BatchDeserializationErrorData)
	for i := 0; i < size; i++ {
		// Read/consume per-point header
		bytesJustRead, errNonBatch = deserializer_header.deserializePerPointHeader(inputStream)
		bytesRead += bytesJustRead
		if errNonBatch != nil {
			err = errorsWithData.NewErrorWithGuaranteedParameters[BatchDeserializationErrorData](errNonBatch,
				ErrorPrefix+"slice deserialization failed when reading per-point header after reading %v{PointsDeserialized} points. Errors was %w",
				"PointsDeserialized", i,
				FIELDNAME_PARTIAL_READ, true)
			return
		}
		// Read/consume actual point:
		bytesJustRead, errNonBatch = deserializer_point.DeserializeCurvePoint(inputStream, trustLevel, targetSlice.GetByIndex(i))
		bytesRead += bytesJustRead
		if errNonBatch != nil {
			if i != size || !deserializer_header.trivialPerPointFooter() || !deserializer_header.trivialPerPointFooter() || bytesJustRead == 0 {
				err = errorsWithData.NewErrorWithGuaranteedParameters[BatchDeserializationErrorData](errNonBatch,
					ErrorPrefix+"slice deserialization failed after successfully reading %v{PointsDeserialized} points. The error was %w",
					"PointsDeserialized", i,
					FIELDNAME_PARTIAL_READ, true)
			} else {
				err = errorsWithData.NewErrorWithGuaranteedParameters[BatchDeserializationErrorData](errNonBatch,
					ErrorPrefix+"slice deserialization failed after successfully reading %v{PointsDeserialized} points. The error was %w",
					"PointsDeserialized", i)
			}
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
func (md *multiDeserializer[_, _, _, _]) DeserializeSlice(inputStream io.Reader, trustLevel common.IsInputTrusted, sliceMaker DeserializeSliceMaker) (output any, bytesRead int, err BatchDeserializationError) {
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
	bytesJustRead, err = deserializeSlice_mainloop(inputStream, trustLevel, outputPointSlice, md.headerDeserializer, md.basicDeserializer, size)
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
// the passed sliceMaker argument has type func(length int32) (output any, slice CurvePointSlice, err error) and is called exactly once with an appropriate length.
// The slice return value is where DeserializeSlice will write into. The output return value of sliceMaker is the output return value of DeserializeSlice.
// Note that the type(s) contained in slice influence whether DeserializeSlice performs subgroup checks.
// See the specification of DeserializeSliceMaker for details.
//
// Use sliceMaker = CreateNewSlice[PointType] to have DeserializeSlice create a slice of points. output will have type []PointType.
// Use sliceMaker = UseExistingSlice(existingSlice) to use existingSlice as a buffer to hold the result of deserialization. output will have type int and equals the number of points written on success.
//
// On error, at least for the two DeserializeSliceMaker's above, output has the correct type, but is meaningless (possibly a nil slice).
// error contains as data (accessible via errorsWithData) a PointsDeserialized field. This indicates how many points were successfully writen to slice.
func (md *multiSerializer[_, _, _, _]) DeserializeSlice(inputStream io.Reader, trustLevel common.IsInputTrusted, sliceMaker DeserializeSliceMaker) (output any, bytesRead int, err BatchDeserializationError) {
	var size int32                                          // size of the slice
	var errNonBatch bandersnatchErrors.DeserializationError // error returned from individual deserialization routines

	bytesRead, size, errNonBatch = md.headerSerializer.deserializeGlobalSliceHeader(inputStream)
	if errNonBatch != nil {
		err = errorsWithData.NewErrorWithGuaranteedParameters[BatchDeserializationErrorData](errNonBatch, ErrorPrefix+" slice deserialization could not read header (including size). Error was: %w", FIELDNAME_PARTIAL_READ, bytesRead != 0, FIELDNAME_POINTSDESERIALIZED, 0)

		output, _, _ = sliceMaker(-1)
		return
	}
	_, overflowErr := md.SliceOutputLength(size)
	if overflowErr != nil {
		err = errorsWithData.NewErrorWithParametersFromData(overflowErr, ErrorPrefix+"when deserializing a slice, the slice header indicated a length for which the number of bytesRead during deserialization may overflow int32: %w", &BatchDeserializationErrorData{PointsDeserialized: 0, ReadErrorData: bandersnatchErrors.ReadErrorData{PartialRead: true}})
		output, _, _ = sliceMaker(-1)
		return
	}

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

	var bytesJustRead int
	bytesJustRead, err = deserializeSlice_mainloop(inputStream, trustLevel, outputPointSlice, md.headerSerializer, md.basicSerializer, size)
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

// SerializeCurvePoints will (try to) serialize the points from inputPoints to the given outputStream.
// The total number of bytes written must fit into an int32, else we panic.
// If no error occurs, it is equivalent to calling SerializeCurvePoint on each input point separately (but possibly more efficient).
//
// Note that this method treats each point separately and does not write the slice size in-band. Use SerializeSlice for that.
//
// SerializeCurvePoints will stop at the first error. The error value contains data (accessible via the errorsWithData framework)
//   - PointsSerialized (int) is the number of points successfully written
//   - PartialWrite (bool) indicates whether some write operation wrote data that is not aligned with actually encoding points (e.g. due to some IO error in the middle of writing a point)
func (md *multiSerializer[_, _, _, _]) SerializeCurvePoints(outputStream io.Writer, inputPoints curvePoints.CurvePointSlice) (bytesWritten int, err BatchSerializationError) {
	// var _ BatchSerializationErrorData

	// for efficiency, we batch-normalize the points, if supported.
	normalizeable, ok := inputPoints.(curvePoints.BatchNormalizerForZ)
	if ok {
		_ = normalizeable.BatchNormalizeForZ() // Note: errors are ignored: Some points might not be normalized.
		// Without knowing about the capabilities of the underlying serializers and point type, we cannot do anything about that.
	}

	L := inputPoints.Len()
	if L > math.MaxInt32 {
		panic(fmt.Errorf(ErrorPrefix+"SerializeCurvePoints was asked to serialize %v points, which exceeds MaxInt32. Bailing out", L))
	}
	if int64(L)*int64(md.OutputLength()) > math.MaxInt32 {
		panic(fmt.Errorf(ErrorPrefix+"trying to batch-serialize %v points, each expected to write %v bytes. The total number of bytes might exceed MaxInt32. Bailing out", L, md.OutputLength()))
	}
	for i := 0; i < L; i++ {
		inputPoint := inputPoints.GetByIndex(i) // pointer, wrapped in interface
		bytesJustWriten, errSingle := md.SerializeCurvePoint(outputStream, inputPoint)
		bytesWritten += bytesJustWriten
		if errSingle != nil {
			if i != 0 {
				errorTransform.UnexpectEOF2(&errSingle)
			}
			err = errorsWithData.NewErrorWithGuaranteedParameters[BatchSerializationErrorData](errSingle,
				ErrorPrefix+"batch serialization failed after deserializing %{PointsSerialized} many points with error %w",
				"PointsSerialized", i)
			return
		}
	}
	return
}

func (md *multiSerializer[_, _, _, _]) SerializeSlice(outputStream io.Writer, inputPoints curvePoints.CurvePointSlice) (bytesWritten int, err BatchSerializationError) {
	// Get slice length
	LInt := inputPoints.Len()
	if LInt > math.MaxInt32 {
		panic(fmt.Errorf(ErrorPrefix+"called SerializeSlice with a slice of length %v > MaxInt32. This is not supported. Bailing out", LInt))
	}
	L := int32(LInt)

	// ensure the total number of bytes written will not overflow MaxInt32. Notet that we don't need expectedSize itself, actually.
	// We only use it to make an internal self-check after everything.
	expectedSize, errOverflow := md.SliceOutputLength(L)
	if errOverflow != nil {
		// var noWriteAttempt :=
		overflowErrData := BatchSerializationErrorData{WriteErrorData: bandersnatchErrors.NoWriteAttempt, PointsSerialized: 0}
		err = errorsWithData.NewErrorWithParametersFromData(bandersnatchErrors.ErrSizeDoesNotFitInt32, ErrorPrefix+"called SerializeSlice with a slice that would require more than MaxInt32 bytes to serialize", &overflowErrData)
		return
	}

	// write length header
	bytesJustWritten, errHeader := md.headerSerializer.serializeGlobalSliceHeader(outputStream, L)
	bytesWritten += bytesJustWritten
	if errHeader != nil {
		err = errorsWithData.NewErrorWithGuaranteedParameters[BatchSerializationErrorData](errHeader,
			ErrorPrefix+"failed to write slice header. Error was %w",
			"PointsSerialized", 0,
			"PartialWrite", bytesWritten != 0)
		return
	}

	// for efficiency, we batch-normalize the points, if supported.
	normalizeable, ok := inputPoints.(curvePoints.BatchNormalizerForZ)
	if ok {
		_ = normalizeable.BatchNormalizeForZ() // Note: errors are ignored: Some points might not be normalized.
		// Without knowing about the capabilities of the underlying serializers and point type, we cannot do anything about that.
	}

	var errNonBatch bandersnatchErrors.SerializationError
	// write each point. Note that i is int, not int32 -- this is important to include it as parameter in errors.
	for i := 0; i < LInt; i++ {
		// write per-point-header
		bytesJustWritten, errNonBatch = md.headerSerializer.serializePerPointHeader(outputStream)
		bytesWritten += bytesJustWritten
		if errNonBatch != nil {
			err = errorsWithData.NewErrorWithGuaranteedParameters[BatchSerializationErrorData](errNonBatch,
				ErrorPrefix+"slice serialization failed after successfully writing %v{PointsSerialized} points. The error was: %w",
				"PointsSerialized", i,
				FIELDNAME_PARTIAL_WRITE, true)
			return
		}

		// write point
		bytesJustWritten, errNonBatch = md.basicSerializer.SerializeCurvePoint(outputStream, inputPoints.GetByIndex(i))
		bytesWritten += bytesJustWritten
		if errNonBatch != nil {
			err = errorsWithData.NewErrorWithGuaranteedParameters[BatchSerializationErrorData](errNonBatch,
				ErrorPrefix+"slice serialization failed after successfully writing %v{PointsSerialized} points. The error was: %w",
				"PointsSerialized", i,
				FIELDNAME_PARTIAL_WRITE, true)
			return
		}

		// write per-point footer. Note that PointsSerialized is set to i+1 here (this is debatable, but done for constency with failing slice reads)
		bytesJustWritten, errNonBatch = md.headerSerializer.serializePerPointFooter(outputStream)
		bytesWritten += bytesJustWritten
		if errNonBatch != nil {
			err = errorsWithData.NewErrorWithGuaranteedParameters[BatchSerializationErrorData](errNonBatch,
				ErrorPrefix+"slice serialization failed after successfully writing %v{PointsSerialized} points. The error was: %w",
				"PointsSerialized", i+1,
				FIELDNAME_PARTIAL_WRITE, true)
			return
		}
	}

	// write slice footer
	bytesJustWritten, errNonBatch = md.headerSerializer.serializeGlobalSliceFooter(outputStream)
	bytesWritten += bytesJustWritten
	if errNonBatch != nil {
		err = errorsWithData.NewErrorWithGuaranteedParameters[BatchSerializationErrorData](errNonBatch,
			ErrorPrefix+"slice serialization failed after successfully writing %v{PointsSerialized} points. The error was: %w",
			"PointsSerialized", LInt, // Note: PointsSerialized needs type int, not int32
			FIELDNAME_PARTIAL_WRITE, true)
		return
	}

	if bytesWritten != int(expectedSize) {
		panic(fmt.Errorf(ErrorPrefix+"Slice serialization for slice of length %v was successful, but the number of bytes written was not what we expected: bytesWritten = %v, but we expected %v", LInt, bytesWritten, expectedSize))
	}
	return
}
