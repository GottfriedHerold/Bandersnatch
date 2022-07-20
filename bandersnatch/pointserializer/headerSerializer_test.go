package pointserializer

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

var _ headerDeserializer = &simpleHeaderDeserializer{}
var _ headerSerializer = &simpleHeaderSerializer{}
var _ headerDeserializer = &simpleHeaderSerializer{}

var _ utils.Clonable[*simpleHeaderDeserializer] = &simpleHeaderDeserializer{}
var _ utils.Clonable[*simpleHeaderSerializer] = &simpleHeaderSerializer{}

func TestFixNilEntries(t *testing.T) {
	var shd simpleHeaderDeserializer
	shd.sliceSizeEndianness = binary.LittleEndian
	shd.fixNilEntries()
	for _, arg := range [][]byte{shd.headerSlice, shd.footerSlice, shd.headerPerCurvePoint, shd.footerPerCurvePoint, shd.headerSingleCurvePoint, shd.footerSingleCurvePoint} {
		if arg == nil {
			t.Fatalf("fixNilEntries kept nil")
		}
	}
	shd.Validate()
}

func TestRecognizeParameterNames(t *testing.T) {
	for _, arg := range headerSerializerParams {
		arg = strings.ToLower(arg)
		_, ok := serializerParams[arg]
		if !ok {
			t.Fatalf("serializer parameter named %v not recognized by global parameter lookup table", arg)
		}
	}
}

// getParamDirectlyForSimpleHeaderDeserializer returns the []byte stored in a simpleHeaderDeserializer bypassing the getter.
func getParamDirectlyForSimpleHeaderDeserializer(shd *simpleHeaderDeserializer, paramName string) []byte {
	switch paramName {
	case "GlobalSliceHeader":
		return shd.headerSlice
	case "GlobalSliceFooter":
		return shd.footerSlice
	case "SinglePointHeader":
		return shd.headerSingleCurvePoint
	case "SinglePointFooter":
		return shd.footerSingleCurvePoint
	case "PerPointHeader":
		return shd.headerPerCurvePoint
	case "PerPointFooter":
		return shd.footerPerCurvePoint
	default:
		panic("unrecognized paramName")
	}
}

func TestSettersAndGetters(t *testing.T) {
	var shd simpleHeaderDeserializer
	shd.sliceSizeEndianness = binary.LittleEndian
	shd.Validate()

	var m map[string][]byte = make(map[string][]byte)

	for _, paramName := range headerSerializerParams {
		m[paramName] = []byte(paramName)
		shd = makeCopyWithParams(&shd, paramName, m[paramName])
	}
	for _, paramName := range headerSerializerParams {
		arg := getSerializerParam(&shd, paramName).([]byte)
		if arg == nil {
			t.Fatalf("Getter returned nil")
		}
		if testutils.CheckSliceAlias(arg, m[paramName]) {
			t.Fatalf("Getter returned alias to value that was set")
		}
		if !bytes.Equal(arg, m[paramName]) {
			t.Fatalf("Getter did not return value that was set")
		}
		arg2 := getParamDirectlyForSimpleHeaderDeserializer(&shd, paramName)
		if testutils.CheckSliceAlias(arg, arg2) {
			t.Fatalf("Getter returns value that aliases stored value")
		}
		if !bytes.Equal(arg, arg2) {
			t.Fatalf("Getter does not return stored value")
		}

	}

}
