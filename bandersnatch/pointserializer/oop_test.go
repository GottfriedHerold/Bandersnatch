package pointserializer

import (
	"encoding/binary"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
)

// keys in the global serializerParams act case-insensitve, which is implemented via normalization to lowercase. So the entries in the map must be lowercase.
func TestParamsLowercase(t *testing.T) {
	for key := range serializerParams {
		if key != normalizeParameter(key) {
			t.Fatalf("serializerParams has non-lowercased key %v", key)
		}
	}
}

type dummyGetterOnly struct{}

func (*dummyGetterOnly) GetEndianness() binary.ByteOrder { return binary.LittleEndian }

type dummySetterOnly struct{}

func (*dummySetterOnly) SetEndianness(b binary.ByteOrder) {}

type dummyGetterAndSetter struct {
	dummyGetterOnly
	dummySetterOnly
}

func TestHasParameters(t *testing.T) {
	var nilEndianness *FieldElementEndianness = nil
	if !testutils.CheckPanic(hasParameter, nilEndianness, "invalidParameter") {
		t.Fatalf("hasParameter did not panic on unrecognized parameter")
	}
	if hasParameter(nilEndianness, "SubgroupOnly") {
		t.Fatalf("hasParameter returned true when it should not")
	}
	if !hasParameter(nilEndianness, "endianness") {
		t.Fatalf("hasParameter returned false when it should not")
	}
	var getterOnly *dummyGetterOnly = nil
	var setterOnly *dummySetterOnly = nil
	var setterAndGetter *dummyGetterAndSetter = nil
	if hasParameter(getterOnly, "endianness") {
		t.Fatalf("hasParameter returned true for struct with getter only")
	}
	if hasParameter(setterOnly, "endianness") {
		t.Fatalf("hasParameter returned true for struct with setter only")
	}
	if !hasParameter(setterAndGetter, "endianness") {
		t.Fatalf("hasParamter returned false for struct with both getter and setter")
	}
}
