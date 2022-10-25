package pointserializer

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"reflect"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/bandersnatchErrors"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/common"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/curvePoints"
	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

var _ curvePointDeserializer_basic = &pointSerializerXY{}
var _ curvePointDeserializer_basic = &pointSerializerXAndSignY{}
var _ curvePointDeserializer_basic = &pointSerializerYAndSignX{}
var _ curvePointDeserializer_basic = &pointSerializerXTimesSignY{}
var _ curvePointDeserializer_basic = &pointSerializerYXTimesSignY{}

var _ curvePointSerializer_basic = &pointSerializerXY{}
var _ curvePointSerializer_basic = &pointSerializerXAndSignY{}
var _ curvePointSerializer_basic = &pointSerializerYAndSignX{}
var _ curvePointSerializer_basic = &pointSerializerXTimesSignY{}
var _ curvePointSerializer_basic = &pointSerializerYXTimesSignY{}

var _ modifyableSerializer[pointSerializerXY, *pointSerializerXY] = &pointSerializerXY{}
var _ modifyableSerializer[pointSerializerXAndSignY, *pointSerializerXAndSignY] = &pointSerializerXAndSignY{}
var _ modifyableSerializer[pointSerializerYAndSignX, *pointSerializerYAndSignX] = &pointSerializerYAndSignX{}
var _ modifyableSerializer[pointSerializerXTimesSignY, *pointSerializerXTimesSignY] = &pointSerializerXTimesSignY{}
var _ modifyableSerializer[pointSerializerYXTimesSignY, *pointSerializerYXTimesSignY] = &pointSerializerYXTimesSignY{}

var testBitHeader = common.MakeBitHeader(common.PrefixBits(0b1), 1)

var ps_XY = &pointSerializerXY{valuesSerializerHeaderFeHeaderFe{fieldElementEndianness: defaultEndianness, bitHeader: testBitHeader}, subgroupRestriction{}}
var ps_XY_sub = ps_XY.WithParameter("SubgroupOnly", true)
var ps_XSY = &pointSerializerXAndSignY{valuesSerializerFeCompressedBit{fieldElementEndianness: defaultEndianness}, subgroupRestriction{}}
var ps_XSY_sub = ps_XSY.WithParameter("SubgroupOnly", true)
var ps_YSX = &pointSerializerYAndSignX{valuesSerializerFeCompressedBit{fieldElementEndianness: defaultEndianness}, subgroupRestriction{}}
var ps_YSX_sub = ps_YSX.WithParameter("SubgroupOnly", true)
var ps_XxSY = basicBanderwagonShort
var ps_XYxSY = basicBanderwagonLong

var allBasicSerializers []curvePointSerializer_basic = []curvePointSerializer_basic{
	ps_XY,
	ps_XY_sub,
	ps_XSY,
	ps_XSY_sub,
	ps_YSX,
	ps_YSX_sub,
	ps_XxSY,
	ps_XYxSY,
}

var allSubgroupOnlySerializers []curvePointSerializer_basic = []curvePointSerializer_basic{
	ps_XxSY,
	ps_XYxSY,
}

var allSerializersWithModifyableSubgroupOnly []curvePointSerializer_basic = []curvePointSerializer_basic{
	ps_XY,
	ps_XSY,
	ps_YSX,
}

var allBasicSerializerTypes = []reflect.Type{
	utils.TypeOfType[pointSerializerXY](),
	utils.TypeOfType[pointSerializerXAndSignY](),
	utils.TypeOfType[pointSerializerYAndSignX](),
	utils.TypeOfType[pointSerializerXTimesSignY](),
	utils.TypeOfType[pointSerializerYXTimesSignY](),
}

// Ensure that the serializers work with our default_* implementation of WithParameter and GetParameter.
// Note that this might change and we might just not use the default_ functions for a given type.

var pointSerializerTypesUsingDefaultParameterChangeMethods = allBasicSerializerTypes

func TestDefaultParameterHandlingIsSaneForBasicSerializers(t *testing.T) {
	for _, basicSerializerType := range pointSerializerTypesUsingDefaultParameterChangeMethods {
		ensureDefaultSettersAndGettersWorkForSerializer(basicSerializerType, t)
	}
}

// This test is more strict than neccessary on the return type of Clone(), but all pointSerializers satisfy that

func TestBasicSerializersHasClonable(t *testing.T) {
	for _, basicSerializer := range allBasicSerializers {
		serializerType := reflect.TypeOf(basicSerializer)
		ok, reason := utils.DoesMethodExist(serializerType, "Clone", []reflect.Type{}, []reflect.Type{serializerType})
		if !ok {
			t.Error(reason)
		}
	}
}

// This test is more strict than neccessary on the return type of WithParamter, but all our pointSerializer satisfy that.

func TestBasicSerializersHaveWithParams(t *testing.T) {
	for _, basicSerializer := range allBasicSerializers {
		serializerType := reflect.TypeOf(basicSerializer)
		ok, reason := utils.DoesMethodExist(serializerType, "WithParameter", []reflect.Type{stringType, anyType}, []reflect.Type{serializerType})
		if !ok {
			t.Error(reason)
		}
	}
}

// Mostly superseded by generics, but kept
func TestBasicSerializerHasWithEndianness(t *testing.T) {
	_ = ps_XSY.WithEndianness(binary.BigEndian)
	for _, basicSerializer := range allBasicSerializers {
		serializerType := reflect.TypeOf(basicSerializer)
		ok, reason := utils.DoesMethodExist(serializerType, "WithEndianness", []reflect.Type{utils.TypeOfType[binary.ByteOrder]()}, []reflect.Type{serializerType})
		if !ok {
			t.Error(reason)
		}
		_ = testutils.CallMethodByName(basicSerializer, "WithEndianness", binary.BigEndian)
		_ = testutils.CallMethodByName(basicSerializer, "WithEndianness", binary.LittleEndian)
	}
}

func TestQueryFunctionsCallableOnNilForBasicSerializers(t *testing.T) {
	for _, basicSerializerType := range allBasicSerializerTypes {
		// sanity check: allBasicSerializerTypes should contain the non-pointer types.
		testutils.FatalUnless(t, basicSerializerType.Kind() != reflect.Pointer, "basicSerializerType contains a pointer")
		basicSerializerType = reflect.PtrTo(basicSerializerType)

		// create nil pointer of concrete type, stored in appropriate interface.
		zeroValue := reflect.Zero(basicSerializerType).Interface().(curvePointDeserializer_basic)

		// NOTE: This is contained here because it currently happens to work. Callability on nil-pointers is not a requirement.
		_ = zeroValue.OutputLength()

		_ = zeroValue.RecognizedParameters()
		_ = zeroValue.HasParameter("foo")

	}

	// loop is over actual interfaces, not reflect.Types and already has pointer types.
	for _, basicSerializer := range allSubgroupOnlySerializers {
		basicSerializerType := reflect.TypeOf(basicSerializer)
		// this time, we already should have pointers.
		testutils.FatalUnless(t, basicSerializerType.Kind() == reflect.Pointer, "basicSerializerType is non-pointer")

		// create nil pointer of concrete type, stored in appropriate interface.
		zeroValue := reflect.Zero(basicSerializerType).Interface().(curvePointDeserializer_basic)

		v := zeroValue.IsSubgroupOnly()
		testutils.FatalUnless(t, v, "IsSubgroupOnly() called on nil value did not return true when we expected that")
		v = zeroValue.GetParameter("SubgroupOnly").(bool)
		testutils.FatalUnless(t, v, "WithParameter('SubgroupOnly') called on nil value did not return true when we expected that")
	}
}

// This test checks whether the behaviour on changing the SubgroupOnly parameter is as we expect.

func TestBasicSerializersCannotChangeAwayFromSubgroupOnly(t *testing.T) {
	for _, basicSerializer := range allSubgroupOnlySerializers {
		var typeName string = utils.GetReflectName(reflect.TypeOf(basicSerializer))

		funSubgroupOnly := func(val bool) {
			newSerializer := testutils.CallMethodByName(basicSerializer, "WithParameter", "SubgroupOnly", val)[0].(curvePointDeserializer_basic)
			if newSerializer.IsSubgroupOnly() != val {
				t.Fatalf("Changing SubgroupOnly not reflected by IsSubgroupOnly for %v", typeName)
			}

		}
		funSubgroupOnly(true)
		didPanic := testutils.CheckPanic(funSubgroupOnly, false)
		if !didPanic {
			t.Fatalf("%v did not panic when trying to set it to non-subgroup-only", typeName)
		}
	}
}

// This test checks whether RecognizedParameters gives a reasonable result

func TestRecognizedParameterForBasicSerializers(t *testing.T) {
	for _, basicSerializer := range allBasicSerializers {
		// ensure consistency with HasParameter
		ensureRecognizedParamsAreFine(t, basicSerializer)
		params := basicSerializer.RecognizedParameters()
		testutils.FatalUnless(t, utils.ElementInList("Endianness", params, normalizeParameter), "Type %T has no Endianness parameter", basicSerializer)
		testutils.FatalUnless(t, utils.ElementInList("SubgroupOnly", params, normalizeParameter), "Type %T has no SubgroupOnly parameter", basicSerializer)
	}

}

// This test verifies that trying to serialize NAPs behaves as expected (fail with the correct error, don't write anyying)

func TestBasicSerializeNAPs(t *testing.T) {
	for _, basicSerializer := range allBasicSerializers {
		var typeName string = utils.GetReflectName(reflect.TypeOf(basicSerializer))
		var P curvePoints.Point_xtw_subgroup
		if !P.IsNaP() {
			t.Fatalf("Uninitialized Point is no NAP. This is not supposed not happen") // well, it's not really a problem semantically, but the test here would not work.
		}
		var buf bytes.Buffer
		bytesWritten, err := basicSerializer.SerializeCurvePoint(&buf, &P)
		if err == nil {
			t.Fatalf("Serializing NAP with %v gave no error", typeName)
		}
		if !errors.Is(err, bandersnatchErrors.ErrCannotSerializeNaP) {
			t.Fatalf("Serializing NAP with %v gave from error. Expected %v, got %v", typeName, bandersnatchErrors.ErrCannotSerializeNaP, err)
		}
		if bytesWritten != 0 {
			t.Fatalf("When trying to serialize NAP with %v, something was actually written", typeName)
		}
	}
}

// This ensures that trying to serialize points outside the subgroup fails with the correct error.
// We only try with the affine 2-torsion point.

func TestBasicSerializersNonSubgroup(t *testing.T) {
	for _, basicSerializer := range allSubgroupOnlySerializers {
		var typeName string = utils.GetReflectName(reflect.TypeOf(basicSerializer))
		var P curvePoints.Point_xtw_full
		P.SetAffineTwoTorsion() // not in subgroup
		var buf bytes.Buffer
		bytesWritten, err := basicSerializer.SerializeCurvePoint(&buf, &P)
		if err == nil {
			t.Fatalf("Serializing point outside subgroup gave no error for %v", typeName)
		}
		if !errors.Is(err, bandersnatchErrors.ErrWillNotSerializePointOutsideSubgroup) {
			t.Fatalf("Serializing point outside subgroup gave wrong error for %v. Error gotten was %v", typeName, err)
		}
		if bytesWritten != 0 {
			t.Fatalf("Serializing point outside subgroup actually wrote something for %v", typeName)
		}
	}
}

// TODO:
// Test behaviour for serializing points at infinity. Note that expected behaviour is subject to change.

// Test Roundtrip for basic serialializers.
// We also check correct error behaviour on reading EOF / unexpected EOF.

func TestBasicSerializersRoundtrip(t *testing.T) {
	var drng *rand.Rand = rand.New(rand.NewSource(1024))
	for _, basicSerializer := range allBasicSerializers {
		outputLen := int(basicSerializer.OutputLength())
		const iterations = 20
		for i := 0; i < iterations+outputLen; i++ {
			var err error
			serializerName := utils.GetReflectName(reflect.TypeOf(basicSerializer))
			var subgroupOnly bool = basicSerializer.IsSubgroupOnly()
			var inputPoint curvePoints.CurvePointPtrInterface
			if subgroupOnly {
				var point curvePoints.Point_xtw_subgroup = curvePoints.MakeRandomPointUnsafe_xtw_subgroup(drng)
				inputPoint = &point
			} else {
				var point curvePoints.Point_xtw_full = curvePoints.MakeRandomPointUnsafe_xtw_full(drng)
				inputPoint = &point
			}
			var buf bytes.Buffer
			bytesWritten, err := basicSerializer.SerializeCurvePoint(&buf, inputPoint)
			if err != nil {
				t.Fatal(fmt.Errorf("Error when using %v's SerializeCurvePoint Method: %w", serializerName, err))
			}
			if bytesWritten != int(basicSerializer.OutputLength()) {
				t.Fatalf("Error when using %v's SerializeCurvePoint Method: bytesWritten == %v, expected output length == %v", serializerName, bytesWritten, basicSerializer.OutputLength())
			}

			// write extra byte with arbitrary value. This is make sure that the fact that reading stops at the correct position is for non EOF-reasons.
			buf.WriteByte(42)

			var truncate bool = (i >= iterations)
			var truncateTo int
			if truncate {
				truncateTo = i - iterations
				buf.Truncate(truncateTo)
			}

			// distinguish type to ensure subgroup checks are performed when reading from buf
			var outputPoint curvePoints.CurvePointPtrInterface
			if subgroupOnly {
				outputPoint = &curvePoints.Point_xtw_subgroup{}
			} else {
				outputPoint = &curvePoints.Point_xtw_full{}
			}

			bytesRead, err := basicSerializer.DeserializeCurvePoint(&buf, common.UntrustedInput, outputPoint)
			if truncate {
				if err == nil {
					t.Fatalf("Deserializing on truncated buffer gave no error for %v when it should have", serializerName)
				}
				if bytesRead != truncateTo {
					t.Fatalf("Deserializing on buffer truncated to %v read only %v for %v", truncateTo, bytesRead, serializerName)
				}
				if truncateTo == 0 {
					if !errors.Is(err, io.EOF) {
						t.Fatalf("Error when using %v's Deserialization on truncated input. Did not get EOF. Got %v", serializerName, err)
					}
				} else {
					if !errors.Is(err, io.ErrUnexpectedEOF) {
						t.Fatalf("Error when using %v's Deserialization on truncated input. Did not get ErrUnexpectedEOF. Got %v", serializerName, err)
					}
				}
				continue // no more checks for this iteration
			}
			if err != nil {
				t.Fatal(fmt.Errorf("Error when using %v's DeserializeCurvePoint Method: %w", serializerName, err))
			}
			if bytesRead != bytesWritten {
				t.Fatalf("Error for %v: Serializing and Deserializing gives mismatching bytesWritten == %v and bytesRead == %v", serializerName, bytesWritten, bytesRead)
			}
			if !inputPoint.IsEqual(outputPoint) {
				t.Fatalf("Error for %v: Roundtrip error", serializerName)
			}

		}
	}

}
