package pointserializer

import (
	"encoding/binary"
	"reflect"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

// TOOD: This does not check argument types for getters and setters!
// NOTE: Has been moved to test, will probably be removed completely.

// hasSetterAndGetterForParameter(serializer, parameterName) checks whether the (dynamic) type of serializer has setter and getter methods for the given parameter.
// The name of these getter and setter methods is looked up via the serializerParams map.
// parameterName is case-insensitive
//
// Note that the serializer argument is only used to derive the generic parameters and may be a nil pointer of the appropriate type.
//
// Note: This internal function does NOT look at RecognizedParameters().
// It instead uses reflection to check the presence of methods.
// It panics if called on invalid parameter strings not in the serializerParams map.
//
// DEPRECATED
func hasSetterAndGetterForParameter(serializer any, parameterName string) bool {
	parameterName = normalizeParameter(parameterName) // make parameterName case-insensitive
	paramInfo, ok := default_serializerParamFuns[parameterName]

	// Technically, we could just meaningfully return false if parameterName is not found in serializerParams.
	// However, this is an internal function and we never intend to call hasParameter on anything but a fixed string which is supposed to be a key of the serializerParams map.
	// Hence, this can only occur due to a bug (e.g. a typo in the parameterName string).
	if !ok {
		panic(ErrorPrefix + "hasParameter called with unrecognized parameter name")
	}

	serializerType := reflect.TypeOf(serializer)
	_, ok = serializerType.MethodByName(paramInfo.getter)
	if !ok {
		return false
	}
	_, ok = serializerType.MethodByName(paramInfo.setter)
	return ok
}

// keys in the global serializerParams act case-insensitve, which is implemented via normalization to lowercase. So the entries in the map must be lowercase.
func TestParamsNormalized(t *testing.T) {
	for key := range default_serializerParamFuns {
		if key != normalizeParameter(key) {
			t.Fatalf("serializerParams has non-normalized key %v", key)
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
	if !testutils.CheckPanic(hasSetterAndGetterForParameter, nilEndianness, "invalidParameter") {
		t.Fatalf("hasParameter did not panic on unrecognized parameter")
	}
	if hasSetterAndGetterForParameter(nilEndianness, "SubgroupOnly") {
		t.Fatalf("hasParameter returned true when it should not")
	}
	if !hasSetterAndGetterForParameter(nilEndianness, "Endianness") {
		t.Fatalf("hasParameter returned false when it should not")
	}
	var getterOnly *dummyGetterOnly = nil
	var setterOnly *dummySetterOnly = nil
	var setterAndGetter *dummyGetterAndSetter = nil
	if hasSetterAndGetterForParameter(getterOnly, "Endianness") {
		t.Fatalf("hasParameter returned true for struct with getter only")
	}
	if hasSetterAndGetterForParameter(setterOnly, "Endianness") {
		t.Fatalf("hasParameter returned true for struct with setter only")
	}
	if !hasSetterAndGetterForParameter(setterAndGetter, "Endianness") {
		t.Fatalf("hasParamter returned false for struct with both getter and setter")
	}
}

func TestConcatParameterList(t *testing.T) {
	emptyList := []string{}
	emptyListSum := concatParameterList(emptyList, emptyList)
	if len(emptyListSum) != 0 {
		t.Fatalf("Concatenatio of empty lists non-empty")
	}
	list1 := []string{"A", "AB", "A", "DUP", "B"}
	list2 := []string{"B", "C", "DUP", "D"}
	concat := concatParameterList(list1, list2)
	expectedconcat := []string{"A", "AB", "DUP", "B", "C", "D"}
	if len(concat) != len(expectedconcat) {
		t.Fatalf("List concatentation has unexpected length")
	}
	for i, v := range concat {
		if v != expectedconcat[i] {
			t.Fatalf("List concatenation not as expected")
		}
	}

}

func ensureParamsAreValidForSerializer(serializer parameterAware, t *testing.T) {

	params := serializer.RecognizedParameters()
	for _, param := range params {
		validateGetter(serializer, param)
		validateSetter(serializer, param)
		if !serializer.HasParameter(param) {
			t.Fatalf("Serializer of type %v does not have parameter %v recognized by HasParameter", utils.GetReflectName(reflect.TypeOf(serializer)), param)
		}
		if serializer.HasParameter("InvalidParameter") {
			t.Fatalf("Serializer of type %v does recognize an invalid paramter as valid", utils.GetReflectName(reflect.TypeOf(serializer)))
		}
	}

}
