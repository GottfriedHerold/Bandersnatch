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

var (
	BanderwagonLong_OnlyDeserializer  = newMultiDeserializer(basicBanderwagonLong, trivialSimpleHeaderDeserializer)
	BanderwagonShort_OnlyDeserializer = newMultiDeserializer(basicBanderwagonShort, trivialSimpleHeaderDeserializer)
)

var (
	allTestMultiSerializers   []CurvePointSerializerModifyable = []CurvePointSerializerModifyable{BanderwagonShort, BanderwagonLong}
	allTestMultiDeserializers []CurvePointDeserializer         = []CurvePointDeserializer{BanderwagonShort_OnlyDeserializer, BanderwagonLong_OnlyDeserializer}
)

var DeserializerFromSerializer = map[CurvePointSerializerModifyable]CurvePointDeserializer{
	BanderwagonShort: BanderwagonShort_OnlyDeserializer,
	BanderwagonLong:  BanderwagonLong_OnlyDeserializer,
}

type (
	someMultiSerializerType   = multiSerializer[*pointSerializerXY, *simpleHeaderSerializer, pointSerializerXY, simpleHeaderSerializer]
	someMultiDeserializerType = multiDeserializer[*pointSerializerYXTimesSignY, *simpleHeaderDeserializer, pointSerializerYXTimesSignY, simpleHeaderDeserializer]
)

func TestEnsureExportedSerializersValidate(t *testing.T) {
	BanderwagonLong.Validate()
	BanderwagonShort.Validate()
}

func TestClonePreservesDynamicType(t *testing.T) {
	for _, serializer := range allTestMultiSerializers {
		serializerType := reflect.TypeOf(serializer)
		testutils.Assert(serializerType.Kind() == reflect.Pointer)

		serializerValue := reflect.ValueOf(serializer)
		CloneMethod := serializerValue.MethodByName("Clone")
		cloneValue := CloneMethod.Call([]reflect.Value{})[0]
		// this looks stupid, but cloneValue reflects a variable of type interface, so this reaches into the interface and changes it to the dynamic type.
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
		// this looks stupid, but cloneValue reflects a variable of type interface, so this reaches into the interface and changes it to the dynamic type.
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
