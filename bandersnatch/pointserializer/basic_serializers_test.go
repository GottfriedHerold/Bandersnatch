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

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/bandersnatchErrors"
	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
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

var testBitHeader = bitHeader{prefixLen: 1, prefixBits: 0b1}

var ps_XY = pointSerializerXY{valuesSerializerHeaderFeHeaderFe{fieldElementEndianness: defaultEndianness, bitHeader: testBitHeader}, subgroupRestriction{subgroupOnly: false}}
var ps_XY_sub = ps_XY.WithParameter("SubgroupOnly", true)
var ps_XSY = pointSerializerXAndSignY{valuesSerializerFeCompressedBit{fieldElementEndianness: defaultEndianness}, subgroupRestriction{subgroupOnly: false}}
var ps_XSY_sub = ps_XSY.WithParameter("SubgroupOnly", true)
var ps_YSX = pointSerializerYAndSignX{valuesSerializerFeCompressedBit{fieldElementEndianness: defaultEndianness}, subgroupRestriction{subgroupOnly: false}}
var ps_YSX_sub = ps_YSX.WithParameter("SubgroupOnly", true)
var ps_XxSY = basicBanderwagonShort
var ps_XYxSY = basicBanderwagonLong

var allBasicSerializers []curvePointSerializer_basic = []curvePointSerializer_basic{
	&ps_XY,
	&ps_XY_sub,
	&ps_XSY,
	&ps_XSY_sub,
	&ps_YSX,
	&ps_YSX_sub,
	&ps_XxSY,
	&ps_XYxSY,
}

var allSubgroupOnlySerializers []curvePointSerializer_basic = []curvePointSerializer_basic{
	&ps_XxSY,
	&ps_XYxSY,
}

var allSerializersWithModifyableSubgroupOnly []curvePointSerializer_basic = []curvePointSerializer_basic{
	&ps_XY,
	&ps_XSY,
	&ps_YSX,
}

func TestBasicSerializersHasClonable(t *testing.T) {
	for _, basicSerializer := range allBasicSerializers {
		serializerType := reflect.TypeOf(basicSerializer)
		ok, reason := testutils.DoesMethodExist(serializerType, "Clone", []reflect.Type{}, []reflect.Type{serializerType})
		if !ok {
			t.Error(reason)
		}
	}
}

func TestBasicSerializersHaveWithParams(t *testing.T) {
	for _, basicSerializer := range allBasicSerializers {
		serializerType := reflect.TypeOf(basicSerializer)
		serializerNonPointerType := serializerType.Elem()
		var arg1Type reflect.Type = reflect.TypeOf("abc")
		var dummy *interface{}
		var arg2Type reflect.Type = reflect.TypeOf(dummy).Elem()
		ok, reason := testutils.DoesMethodExist(serializerType, "WithParameter", []reflect.Type{arg1Type, arg2Type}, []reflect.Type{serializerNonPointerType})
		if !ok {
			t.Error(reason)
		}
	}
}

func TestBasicSerializerHasWithEndianness(t *testing.T) {
	endiannessType := byteOrderType
	_ = ps_XSY.WithEndianness(binary.BigEndian)
	for _, basicSerializer := range allBasicSerializers {
		serializerType := reflect.TypeOf(basicSerializer)
		serializerValueType := serializerType.Elem()
		ok, reason := testutils.DoesMethodExist(serializerType, "WithEndianness", []reflect.Type{endiannessType}, []reflect.Type{serializerValueType})
		if !ok {
			t.Error(reason)
		}
		_ = testutils.CallMethodByName(basicSerializer, "WithEndianness", binary.BigEndian)
		_ = testutils.CallMethodByName(basicSerializer, "WithEndianness", binary.LittleEndian)
	}
}

func TestBasicSerializersCannotChangeAwayFromSubgroupOnly(t *testing.T) {
	for _, basicSerializer := range allSubgroupOnlySerializers {
		var typeName string = testutils.GetReflectName(reflect.TypeOf(basicSerializer))

		funSubgroupOnly := func(val bool) {
			_ = testutils.CallMethodByName(basicSerializer, "WithParameter", "SubgroupOnly", val)
		}
		funSubgroupOnly(true)
		didPanic := bandersnatch.CheckPanic(funSubgroupOnly, false)
		if !didPanic {
			t.Fatalf("%v did not panic when trying to set it to non-subgroup-only", typeName)
		}
	}
}

func TestBasicSerializeNAPs(t *testing.T) {
	for _, basicSerializer := range allBasicSerializers {
		var typeName string = testutils.GetReflectName(reflect.TypeOf(basicSerializer))
		var P bandersnatch.Point_xtw_subgroup
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

// Test Roundtrip for basic serialializers.
// We also check correct error behaviour on reading EOF / unexpected EOF.

func TestBasicSerializersRoundtrip(t *testing.T) {
	var drng *rand.Rand = rand.New(rand.NewSource(1024))
	for _, basicSerializer := range allBasicSerializers {
		outputLen := basicSerializer.OutputLength()
		const iterations = 20
		for i := 0; i < iterations+outputLen; i++ {
			serializerName := testutils.GetReflectName(reflect.TypeOf(basicSerializer))
			var subgroupOnly bool = basicSerializer.IsSubgroupOnly()
			var inputPoint bandersnatch.CurvePointPtrInterface
			if subgroupOnly {
				var point bandersnatch.Point_xtw_subgroup = bandersnatch.MakeRandomPointUnsafe_xtw_subgroup(drng)
				inputPoint = &point
			} else {
				var point bandersnatch.Point_xtw_full = bandersnatch.MakeRandomPointUnsafe_xtw_full(drng)
				inputPoint = &point
			}
			var buf bytes.Buffer
			bytesWritten, err := basicSerializer.SerializeCurvePoint(&buf, inputPoint)
			if err != nil {
				t.Fatal(fmt.Errorf("Error when using %v's SerializeCurvePoint Method: %w", serializerName, err))
			}
			if bytesWritten != basicSerializer.OutputLength() {
				t.Fatalf("Error when using %v's SerializeCurvePoint Method: bytesWritten == %v, expected output length == %v", serializerName, bytesWritten, basicSerializer.OutputLength())
			}

			// write extra byte, to ensure reading stops at the correct position.
			buf.WriteByte(42)

			var truncate bool = (i >= iterations)
			var truncateTo int
			if truncate {
				truncateTo = i - iterations
				buf.Truncate(truncateTo)
			}

			// distinguish type to ensure subgroup checks are performed when reading from buf
			var outputPoint bandersnatch.CurvePointPtrInterface
			if subgroupOnly {
				outputPoint = &bandersnatch.Point_xtw_subgroup{}
			} else {
				outputPoint = &bandersnatch.Point_xtw_full{}
			}

			bytesRead, err := basicSerializer.DeserializeCurvePoint(&buf, bandersnatch.UntrustedInput, outputPoint)
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
