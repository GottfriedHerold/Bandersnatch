package pointserializer

import (
	"bytes"
	"math/rand"
	"reflect"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/common"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/curvePoints"
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

var (
	allTestMultiSerializers   []CurvePointSerializerModifyable   = []CurvePointSerializerModifyable{BanderwagonShort, BanderwagonLong, VerboseBanderwagonLong, VerboseBanderwagonShort}
	allTestMultiDeserializers []CurvePointDeserializerModifyable = []CurvePointDeserializerModifyable{BanderwagonShort_OnlyDeserializer, BanderwagonLong_OnlyDeserializer, VerboseBanderwagonLong_OnlyDeserializer, VerboseBanderwagonShort_OnlyDeserializer}
)

var DeserializerFromSerializer = map[CurvePointSerializerModifyable]CurvePointDeserializerModifyable{
	BanderwagonShort:        BanderwagonShort_OnlyDeserializer,
	BanderwagonLong:         BanderwagonLong_OnlyDeserializer,
	VerboseBanderwagonLong:  VerboseBanderwagonLong_OnlyDeserializer,
	VerboseBanderwagonShort: VerboseBanderwagonShort_OnlyDeserializer,
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
