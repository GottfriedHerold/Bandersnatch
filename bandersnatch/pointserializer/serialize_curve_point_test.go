package pointserializer

import (
	"bytes"
	"errors"
	"io"
	"math"
	"math/rand"
	"reflect"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/bandersnatchErrors"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/common"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/curvePoints"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/errorsWithData"
	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
)

var VerboseBanderwagonLong = BanderwagonLong.
	WithParameter("SinglePointHeader", "SinglePointHeader").
	WithParameter("SinglePointFooter", "SinglePointFooter").
	WithParameter("GlobalSliceHeader", "GlobalSliceHeader").
	WithParameter("GlobalSliceFooter", "GlobalSliceFooter").
	WithParameter("PerPointHeader", "PerPointHeader").
	WithParameter("PerPointFooter", "PerPointFooter")

var VerboseBanderwagonShort = BanderwagonShort.
	WithParameter("SinglePointHeader", "SinglePointHeader").
	WithParameter("SinglePointFooter", "SinglePointFooter").
	WithParameter("GlobalSliceHeader", "GlobalSliceHeader").
	WithParameter("GlobalSliceFooter", "GlobalSliceFooter").
	WithParameter("PerPointHeader", "PerPointHeader").
	WithParameter("PerPointFooter", "PerPointFooter")

var (
	BanderwagonLong_OnlyDeserializer         = BanderwagonLong.AsDeserializer()
	BanderwagonShort_OnlyDeserializer        = BanderwagonShort.AsDeserializer()
	VerboseBanderwagonLong_OnlyDeserializer  = VerboseBanderwagonLong.AsDeserializer()
	VerboseBanderwagonShort_OnlyDeserializer = VerboseBanderwagonShort.AsDeserializer()
)

var WideTestSerializer = newMultiSerializer(basicXYSerializer, trivialSimpleHeaderSerializer).WithParameter("SubgroupOnly", false)
var WideTestDeserializer = newMultiDeserializer(basicXYSerializer, trivialSimpleHeaderDeserializer).WithParameter("SubgroupOnly", false)

var (
	allTestMultiSerializers   []CurvePointSerializerModifyable   = []CurvePointSerializerModifyable{BanderwagonShort, BanderwagonLong, VerboseBanderwagonLong, VerboseBanderwagonShort}
	allTestMultiDeserializers []CurvePointDeserializerModifyable = []CurvePointDeserializerModifyable{BanderwagonShort_OnlyDeserializer, BanderwagonLong_OnlyDeserializer, VerboseBanderwagonLong_OnlyDeserializer, VerboseBanderwagonShort_OnlyDeserializer}
)

var DeserializerFromSerializer = map[CurvePointSerializerModifyable]CurvePointDeserializerModifyable{
	BanderwagonShort:        BanderwagonShort_OnlyDeserializer,
	BanderwagonLong:         BanderwagonLong_OnlyDeserializer,
	VerboseBanderwagonLong:  VerboseBanderwagonLong_OnlyDeserializer,
	VerboseBanderwagonShort: VerboseBanderwagonShort_OnlyDeserializer,
	WideTestSerializer:      WideTestDeserializer,
}

type (
	someMultiSerializerType   = multiSerializer[*pointSerializerXY, *simpleHeaderSerializer, pointSerializerXY, simpleHeaderSerializer]
	someMultiDeserializerType = multiDeserializer[*pointSerializerYXTimesSignY, *simpleHeaderDeserializer, pointSerializerYXTimesSignY, simpleHeaderDeserializer]
)

// make sure everyything we actually export validates

func TestEnsureExportedSerializersValidate(t *testing.T) {
	BanderwagonLong.Validate()
	BanderwagonShort.Validate()
}

// ensure Clone preserves the dynamic type.

func TestClonePreservesDynamicType(t *testing.T) {
	for _, serializer := range allTestMultiSerializers {
		serializerType := reflect.TypeOf(serializer)
		testutils.Assert(serializerType.Kind() == reflect.Pointer)

		// Check the (dynamic) type of everything from allTestMultiSerializers,
		// reflect-Call Clone() on it and check that the (dynamic) type remains the same

		serializerValue := reflect.ValueOf(serializer)
		CloneMethod := serializerValue.MethodByName("Clone")
		cloneValue := CloneMethod.Call([]reflect.Value{})[0]
		// this looks stupid, but cloneValue reflects a variable of type interface: cloneValue.Type().Kind() is interface since cloneValue.Type() is the static type returned by Clone.
		// So we need this to reach into the interface{} and changes it to the dynamic type.
		cloneValue = reflect.ValueOf(cloneValue.Interface())
		cloneType := cloneValue.Type()

		testutils.FatalUnless(t, serializerType == cloneType, "Clone changes dynamic type for %T", serializer)
	}

	for _, serializer := range allTestMultiDeserializers {
		serializerType := reflect.TypeOf(serializer)
		testutils.Assert(serializerType.Kind() == reflect.Pointer)

		serializerValue := reflect.ValueOf(serializer)
		CloneMethod := serializerValue.MethodByName("Clone")
		cloneValue := CloneMethod.Call([]reflect.Value{})[0]
		cloneValue = reflect.ValueOf(cloneValue.Interface())
		cloneType := cloneValue.Type()

		testutils.FatalUnless(t, serializerType == cloneType, "Clone changes dynamic type for %T", serializer)
	}
}

func TestRecognizedParamsForMultiSerializers(t *testing.T) {
	for _, serializer := range allTestMultiDeserializers {
		ensureRecognizedParamsAreFine(t, serializer)
	}
	for _, serializer := range allTestMultiSerializers {
		ensureRecognizedParamsAreFine(t, serializer)
	}
}

func TestNilCallabilityForMultiSerializers(t *testing.T) {
	var zeroValue1 *someMultiSerializerType = nil
	var zeroValue2 *someMultiDeserializerType = nil

	// Note: OutputLength for nil's are not required to work as far as the spec is concerned.

	// _ = zeroValue1.OutputLength()
	// _ = zeroValue2.OutputLength()

	_ = zeroValue1.RecognizedParameters()
	_ = zeroValue2.RecognizedParameters()

	// Note that the real tests here are that .HasParameter(...) does not panic. Checking the results is just an extra.
	testutils.FatalUnless(t, !zeroValue1.HasParameter("Invalid"), "nil serializer accepts invalid parameter")
	testutils.FatalUnless(t, !zeroValue2.HasParameter("Invalid"), "nil deserializer accepts invalid parameter")
	testutils.FatalUnless(t, zeroValue1.HasParameter("Endianness"), "nil serializer does not accept endianness parameter")
	testutils.FatalUnless(t, zeroValue2.HasParameter("Endianness"), "nil serializer does not accept endianness parameter")
}

// Make sure the Banderwagon serializers are set to subgroup only and that cannot be changed.

func TestBanderwagonSubgroupOnly(t *testing.T) {
	testutils.FatalUnless(t, BanderwagonLong.IsSubgroupOnly(), "Banderwagon must be subgroup only")
	testutils.FatalUnless(t, BanderwagonShort.IsSubgroupOnly(), "Banderwagon must be subgroup only")
	testutils.FatalUnless(t, BanderwagonLong_OnlyDeserializer.IsSubgroupOnly(), "Banderwagon must be subgroup only")
	testutils.FatalUnless(t, BanderwagonShort_OnlyDeserializer.IsSubgroupOnly(), "Banderwagon must be subgroup only")
	for _, serializer := range []any{BanderwagonLong, BanderwagonShort, BanderwagonLong_OnlyDeserializer, BanderwagonShort_OnlyDeserializer} {
		for _, newSubgroupOnly := range []bool{true, false} {
			didPanic, errorMessage := testutils.CheckPanic2(func() {
				_ = testutils.CallMethodByName(serializer, "WithParameter", "SubgroupOnly", newSubgroupOnly)
			})
			if newSubgroupOnly {
				testutils.FatalUnless(t, !didPanic, "Setting SubgroupOnly to true (keeping it) did unexpectedly panic: error was %v", errorMessage)
			} else {
				testutils.FatalUnless(t, didPanic, "Setting SubgroupOnly to false did not panic")
			}
		}
	}
}

// Ensure roundtrip when writing with serializer and deserializing with appropriate deserializer
func TestMultiSerializerAndDeserializerConsistency(t *testing.T) {
	var buf bytes.Buffer
	drng := rand.New(rand.NewSource(1))

	const num = 200
	var point [num + 1]curvePoints.Point_xtw_subgroup
	var numbytes [num + 1]int
	var err error

	for serializer, deserializer := range DeserializerFromSerializer {
		for i := 0; i < num; i++ {
			point[i] = curvePoints.MakeRandomPointUnsafe_xtw_subgroup(drng)
			numbytes[i], err = serializer.SerializeCurvePoint(&buf, &point[i])
			testutils.FatalUnless(t, err == nil, "Error during serialization: %v", err)
		}
		point[num].SetNeutral()
		numbytes[num], err = serializer.SerializeCurvePoint(&buf, &point[num])
		testutils.FatalUnless(t, err == nil, "Error during serialization: %v", err)

		for i := 0; i < num+1; i++ {
			var readBack curvePoints.Point_xtw_subgroup
			bytesRead, err := deserializer.DeserializeCurvePoint(&buf, common.UntrustedInput, &readBack)
			testutils.FatalUnless(t, err == nil, "Error during deserialization: %v", err)
			testutils.FatalUnless(t, bytesRead == numbytes[i], "Did not read back same number of bytes as written")
			testutils.FatalUnless(t, readBack.IsEqual(&point[i]), "Did not read back same point as written")
		}
	}
}

func TestRetrieveParamsViaMultiSerializer(t *testing.T) {
	for _, serializer := range allTestMultiSerializers {
		recognizedParams := serializer.RecognizedParameters()
		for _, param := range recognizedParams {
			testutils.FatalUnless(t, serializer.HasParameter(param), "Test is meaningless. Recognized Parameters should have failed anyway.")
			_ = serializer.GetParameter(param)
		}
	}

	for _, deserializer := range allTestMultiDeserializers {
		recognizedParams := deserializer.RecognizedParameters()
		for _, param := range recognizedParams {
			testutils.FatalUnless(t, deserializer.HasParameter(param), "Test is meaningless. Recognized Parameters should have failed anyway.")
			_ = deserializer.GetParameter(param)
		}
	}

	for _, param := range headerSerializerParams {
		// This is not a strict requirement, but our test is only meaningful unter that assumption
		testutils.FatalUnless(t, VerboseBanderwagonLong.HasParameter(param), "VerboseBanderwagonLong should contain headerSerializerParams")
		testutils.FatalUnless(t, VerboseBanderwagonShort.HasParameter(param), "VerboseBanderwagonLong should contain headerSerializerParams")

		val1 := VerboseBanderwagonLong.GetParameter(param).([]byte)
		val1String := string(val1)
		testutils.FatalUnless(t, param == val1String, "VerboseBanderwagonLong should have param == paramName for its header params")

		val2 := VerboseBanderwagonLong_OnlyDeserializer.GetParameter(param).([]byte)
		val2String := string(val2)
		testutils.FatalUnless(t, val1String == val2String, "DeserializerParams differ from serializer params for VerboseBanderwagonLong")

		val2 = VerboseBanderwagonShort.GetParameter(param).([]byte)
		val2String = string(val2)
		testutils.FatalUnless(t, val1String == val2String, "DeserializerParams different for VerboseBanderwagonShort")

		val2 = VerboseBanderwagonShort_OnlyDeserializer.GetParameter(param).([]byte)
		val2String = string(val2)
		testutils.FatalUnless(t, val1String == val2String, "DeserializerParams different for VerboseBanderwagonShort (Deserializer)")
	}
}

func TestWithEndianness(t *testing.T) {
	orginalEndianness := BanderwagonLong.GetFieldElementEndianness()
	originalEndianness2 := BanderwagonLong.AsDeserializer().GetFieldElementEndianness()

	// NOTE: This compares pointers inside an interface, actually.
	testutils.FatalUnless(t, orginalEndianness == originalEndianness2, "Endianness choice for serializer and deserializers differ")

	ser := BanderwagonLong.WithEndianness(common.BigEndian)
	testutils.FatalUnless(t, BanderwagonLong.GetFieldElementEndianness() == orginalEndianness, "WithEndianness changes original")
	testutils.FatalUnless(t, ser.GetFieldElementEndianness() == common.BigEndian, "Setting Endianness failed")

	deser := BanderwagonLong.AsDeserializer().WithEndianness(common.BigEndian)
	testutils.FatalUnless(t, BanderwagonLong.GetFieldElementEndianness() == orginalEndianness, "WithEndianness changes original")
	testutils.FatalUnless(t, deser.GetFieldElementEndianness() == common.BigEndian, "Setting Endianness failed")

	ser = BanderwagonLong.WithEndianness(common.LittleEndian)
	testutils.FatalUnless(t, BanderwagonLong.GetFieldElementEndianness() == orginalEndianness, "WithEndianness changes original")
	testutils.FatalUnless(t, ser.GetFieldElementEndianness() == common.LittleEndian, "Setting Endianness failed")

	deser = BanderwagonLong.AsDeserializer().WithEndianness(common.LittleEndian)
	testutils.FatalUnless(t, BanderwagonLong.GetFieldElementEndianness() == orginalEndianness, "WithEndianness changes original")
	testutils.FatalUnless(t, deser.GetFieldElementEndianness() == common.LittleEndian, "Setting Endianness failed")
}

func TestOutputLengthForMultiSerializers(t *testing.T) {
	testutils.FatalUnless(t, BanderwagonLong.OutputLength() == 64, "Unexpected output length for long banderwagon format")
	testutils.FatalUnless(t, BanderwagonShort.OutputLength() == 32, "Unexpected output length for long banderwagon format")

	ser := BanderwagonLong.
		WithParameter("SinglePointHeader", make([]byte, 10)).
		WithParameter("SinglePointFooter", make([]byte, 100)).
		WithParameter("PerPointHeader", make([]byte, 1000)).
		WithParameter("PerPointFooter", make([]byte, 10000))
	testutils.FatalUnless(t, ser.OutputLength() == 110+64, "Unexpected length after changing headers")
	for _, ser := range allTestMultiSerializers {
		testutils.FatalUnless(t, ser.OutputLength() == ser.AsDeserializer().OutputLength(), "Output length differs for serializer vs. deserializer")
	}
}

func TestOutputSliceLengthForMultiSerializer(t *testing.T) {
	x, err := BanderwagonLong.SliceOutputLength(0)
	testutils.FatalUnless(t, err == nil, "unexpected error: %v", err)
	testutils.FatalUnless(t, x == 4, "Unexpected Slice serialization size")

	x, err = BanderwagonLong.SliceOutputLength(100)
	testutils.FatalUnless(t, err == nil, "unexpected error: %v", err)
	testutils.FatalUnless(t, x == 100*64+4, "Unexpected Slice serialization size")

	ser := BanderwagonLong.
		WithParameter("GlobalSliceHeader", make([]byte, 100)).
		WithParameter("GlobalSliceFooter", make([]byte, 50)).
		WithParameter("PerPointHeader", make([]byte, 1000)).
		WithParameter("PerPointFooter", make([]byte, 500))

	x, err = ser.SliceOutputLength(0)
	testutils.FatalUnless(t, err == nil, "unexpected error: %v", err)
	testutils.FatalUnless(t, x == 4+100+50, "Unexpected Slice serialization size")

	for i := int32(0); i < 10; i++ {
		x1, err1 := ser.SliceOutputLength(i)
		x2, err2 := ser.AsDeserializer().SliceOutputLength(i)
		testutils.FatalUnless(t, err1 == nil, "unexpected error %v", err1)
		testutils.FatalUnless(t, err2 == nil, "unexpected error %v", err2)
		testutils.FatalUnless(t, x1 == x2, "SliceOutputLength differs for serializer vs. deserializer")

	}

	x, err = ser.SliceOutputLength(7)
	testutils.FatalUnless(t, err == nil, "unexpected error: %v", err)
	testutils.FatalUnless(t, x == 4+100+50+7*(1000+500+64), "Unexpected Slice serialization size")

	serBig := BanderwagonLong.WithParameter("GlobalSliceHeader", make([]byte, math.MaxInt32-4))

	x, err = serBig.SliceOutputLength(0)
	testutils.FatalUnless(t, err == nil, "unexpected error: %v", err)
	testutils.FatalUnless(t, x == math.MaxInt32, "unexpected Slice serialization size")
	_, err = serBig.SliceOutputLength(1)
	testutils.FatalUnless(t, err != nil, "NO error")
	x64, ok := errorsWithData.GetParameter(err, "Size")
	testutils.FatalUnless(t, ok, "internal error")
	testutils.FatalUnless(t, x64.(int64) == math.MaxInt32+64, "")

}

// Test roundtrip for single-point serialization
func TestRoundtripSingleMultiSerializers(t *testing.T) {
	var buf bytes.Buffer
	drng := rand.New(rand.NewSource(1))

	// create num many points
	const num = 200
	var point [num]curvePoints.Point_xtw_subgroup
	for i := 0; i < num-1; i++ {
		point[i] = curvePoints.MakeRandomPointUnsafe_xtw_subgroup(drng)
	}
	point[num-1].SetNeutral()

	for _, ser := range allTestMultiSerializers {
		expectedWrite := ser.OutputLength()
		buf.Reset()
		for i := 0; i < num; i++ {
			bytesWritten, err := ser.SerializeCurvePoint(&buf, &point[i])
			testutils.FatalUnless(t, err == nil, "Unexpected Write error %v", err)
			testutils.FatalUnless(t, bytesWritten == int(expectedWrite), "Unexpected number of bytes written")
		}
		for i := 0; i < num; i++ {
			var readBack curvePoints.Point_efgh_subgroup
			bytesRead, err := ser.DeserializeCurvePoint(&buf, common.UntrustedInput, &readBack)
			testutils.FatalUnless(t, err == nil, "Unexpected Read error %v", err)
			testutils.FatalUnless(t, bytesRead == int(expectedWrite), "Unexpected number of bytes read")
			testutils.FatalUnless(t, readBack.IsEqual(&point[i]), "Did not read back point")
		}
	}
}

func TestErrorBehaviourSingleSerializeMultiSerializer(t *testing.T) {
	var HEADER []byte = []byte("HEADER")
	var FOOTER []byte = []byte("FOOTER")

	// Try header and footer length 0--2. This is because length 0, length 1 and length 2 behave somewhat differently wrt whether
	// partially writing something is concerned.
	for headerLen := 0; headerLen < 3; headerLen++ {
		for footerLen := 0; footerLen < 3; footerLen++ {

			// Test a serializer with the given header and footer length
			overhead := headerLen + footerLen
			ser := WideTestSerializer.WithParameter("SinglePointHeader", HEADER[0:headerLen]).WithParameter("SinglePointFooter", FOOTER[0:footerLen])
			testutils.Assert(ser.OutputLength() == int32(overhead+64))

			// Obtain a []byte-serialization of an arbitrary point serialized with ser.
			var goodBuf bytes.Buffer
			ser.SerializeCurvePoint(&goodBuf, &curvePoints.SubgroupGenerator_xtw_subgroup)
			correctBytes := goodBuf.Bytes()

			designatedError := errors.New("Designated") // IO error that we will throw from faulty bufs
			// Try writing to a buf that throws an error after i steps:
			for i := 0; i < overhead+64; i++ {
				faultyBuf := testutils.NewFaultyBuffer(i, designatedError)
				bytesWritten, err := ser.SerializeCurvePoint(faultyBuf, &curvePoints.SubgroupGenerator_xtw_subgroup)
				testutils.FatalUnless(t, bytesWritten == i, "Did not write until error")
				testutils.FatalUnless(t, err != nil, "Did not get error on faulty buffer")
				errData := err.GetData_struct()
				testutils.FatalUnless(t, errData.PartialWrite == (i != 0), "Invalid PartialWrite flag")
				testutils.FatalUnless(t, errors.Is(err, designatedError), "Unexpected Error")
			}

			for i := 0; i < overhead+64; i++ {

				var p curvePoints.Point_axtw_subgroup

				faultyBuf := testutils.NewFaultyBuffer(i, designatedError)
				faultyBuf.SetContent(correctBytes)

				bytesRead, err := ser.DeserializeCurvePoint(faultyBuf, UntrustedInput, &p)

				testutils.FatalUnless(t, bytesRead == i, "Did not read until error")
				testutils.FatalUnless(t, err != nil, "Did not get read error on faulty buffer")
				testutils.FatalUnless(t, errors.Is(err, designatedError), "Did not get expected error")
				var errData bandersnatchErrors.ReadErrorData = err.GetData_struct()
				testutils.FatalUnless(t, errData.PartialRead == (i != 0), "Invalid PartialRead flag")

				faultyBuf = testutils.NewFaultyBuffer(i, designatedError)
				faultyBuf.SetContent(correctBytes)

				bytesRead, err = ser.AsDeserializer().DeserializeCurvePoint(faultyBuf, UntrustedInput, &p)

				testutils.FatalUnless(t, bytesRead == i, "Did not read until error")
				testutils.FatalUnless(t, err != nil, "Did not get read error on faulty buffer")
				testutils.FatalUnless(t, errors.Is(err, designatedError), "Did not get expected error")
				errData = err.GetData_struct()
				testutils.FatalUnless(t, errData.PartialRead == (i != 0), "Invalid PartialRead flag")

				eofBuf := bytes.NewBuffer(copyByteSlice(correctBytes)[0:i])

				bytesRead, err = ser.DeserializeCurvePoint(eofBuf, UntrustedInput, &p)

				testutils.FatalUnless(t, bytesRead == i, "Did not read until error")
				testutils.FatalUnless(t, err != nil, "Did not get read error on EOF buffer")
				if i == 0 {
					testutils.FatalUnless(t, errors.Is(err, io.EOF), "Did not get expected EOF error: Got instead:\n%v", err)
				} else {
					testutils.FatalUnless(t, errors.Is(err, io.ErrUnexpectedEOF), "Did not get ErrUnexpectedEOF error. Got instead:\n%v\n %v %v %v", err, headerLen, footerLen, i)
				}

				errData = err.GetData_struct()
				testutils.FatalUnless(t, errData.PartialRead == (i != 0), "Invalid PartialRead flag")

				eofBuf = bytes.NewBuffer(copyByteSlice(correctBytes)[0:i])

				bytesRead, err = ser.AsDeserializer().DeserializeCurvePoint(eofBuf, UntrustedInput, &p)

				testutils.FatalUnless(t, bytesRead == i, "Did not read until error")
				testutils.FatalUnless(t, err != nil, "Did not get read error on EOF buffer")
				if i == 0 {
					testutils.FatalUnless(t, errors.Is(err, io.EOF), "Did not get expected EOF error: Got instead:\n%v", err)
				} else {
					testutils.FatalUnless(t, errors.Is(err, io.ErrUnexpectedEOF), "Did not get ErrUnexpectedEOF error. Got instead:\n%v\n %v %v %v", err, headerLen, footerLen, i)
				}

				errData = err.GetData_struct()
				testutils.FatalUnless(t, errData.PartialRead == (i != 0), "Invalid PartialRead flag")

			}
		}
	}
}
