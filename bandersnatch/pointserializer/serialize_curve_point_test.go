package pointserializer

import (
	"reflect"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
)

var allTestMultiSerializers []CurvePointSerializerModifyable = []CurvePointSerializerModifyable{BanderwagonShort, BanderwagonLong}
var allTestMultiDeserializers []CurvePointDeserializer = []CurvePointDeserializer{BanderwagonShort, BanderwagonLong}

func TestEnsureExportedSerializersValidate(t *testing.T) {
	BanderwagonLong.Validate()
	BanderwagonShort.Validate()
}

func TestClonePreservesDynamicType(t *testing.T) {
	for _, serializer := range []any{BanderwagonLong, basicBanderwagonShort} {
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

func TestRecognizedParamsForExportedSerializers(t *testing.T) {
	for _, serializer := range []CurvePointSerializerModifyable{BanderwagonLong, BanderwagonShort} {
		ensureRecognizedParamsAreFine(t, serializer)
	}
}

type (
	someMultiSerializerType   = multiSerializer[*pointSerializerXY, *simpleHeaderSerializer, pointSerializerXY, simpleHeaderSerializer]
	someMultiDeserializerType = multiDeserializer[*pointSerializerYXTimesSignY, *simpleHeaderDeserializer, pointSerializerYXTimesSignY, simpleHeaderDeserializer]
)

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

func TestBanderwagonSubgroupOnly(t *testing.T) {
	testutils.FatalUnless(t, BanderwagonLong.IsSubgroupOnly(), "Banderwagon must be subgroup only")
	testutils.FatalUnless(t, BanderwagonShort.IsSubgroupOnly(), "Banderwagon must be subgroup only")
}
