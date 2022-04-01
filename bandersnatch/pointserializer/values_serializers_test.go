package pointserializer

import (
	"bytes"
	"math/rand"
	"reflect"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch"
	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
)

var allValuesSerializers []interface{} = []interface{}{
	&valuesSerializerFeFe{fieldElementEndianness: defaultEndianness},
	&valuesSerializerHeaderFeHeaderFe{fieldElementEndianness: defaultEndianness},
	&valuesSerializerFe{fieldElementEndianness: defaultEndianness},
	&valuesSerializerFeCompressedBit{fieldElementEndianness: defaultEndianness},
	&valuesSerializerHeaderFe{fieldElementEndianness: defaultEndianness},
}

func TestValueSerializersHasClonable(t *testing.T) {
	for _, basicSerializer := range allValuesSerializers {
		serializerType := reflect.TypeOf(basicSerializer)
		ok, reason := testutils.DoesMethodExist(serializerType, "Clone", []reflect.Type{}, []reflect.Type{serializerType})
		if !ok {
			t.Error(reason)
		}
	}
}

// This tests that all valuesSerializer types have SerializeValues and DeserializeValues methods that actually roundtrip elements.
func TestValuesSerializersRountrip(t *testing.T) {
	var drng *rand.Rand = rand.New(rand.NewSource(1024))
	for _, valuesSerializer := range allValuesSerializers {
		serializerV := reflect.ValueOf(valuesSerializer)
		serializerT := reflect.TypeOf(valuesSerializer)
		var typeName string = testutils.GetReflectName(serializerT)
		if serializerT.Kind() != reflect.Ptr {
			t.Fatal("values serializer is not a pointer receiver")
		}
		valueSerializerFun := serializerV.MethodByName("SerializeValues")
		valueDeserializerFun := serializerV.MethodByName("DeserializeValues")
		if !valueSerializerFun.IsValid() {
			t.Fatalf("%v has no SerializeValues method", typeName)
		}
		if !valueDeserializerFun.IsValid() {
			t.Fatalf("%v has no DeserializeValues method", typeName)
		}
		var err error
		var bytesWritten int
		valuesSerializerFunType := valueSerializerFun.Type()
		numInputs := valuesSerializerFunType.NumIn()
		var inputs []reflect.Value = make([]reflect.Value, numInputs)
		var inputTypes []reflect.Type = make([]reflect.Type, numInputs)
		for j := 0; j < numInputs; j++ {
			inputTypes[j] = valuesSerializerFunType.In(j)
		}
		feType := reflect.TypeOf(&bandersnatch.FieldElement{})
		boolType := reflect.TypeOf(bool(true))
		const iterations = 20
		var buf bytes.Buffer
		inputs[0] = reflect.ValueOf(&buf)
		for i := 0; i < iterations; i++ {
			expectedLen := 0
			for j := 1; j < numInputs; j++ {
				switch inputTypes[j] {
				case feType:
					var fe bandersnatch.FieldElement
					fe.SetRandomUnsafe(drng)
					inputs[j] = reflect.ValueOf(&fe)
					expectedLen += 32

				case boolType:
					var bit bool = (drng.Intn(2) == 0)
					inputs[j] = reflect.ValueOf(bit)
				default:
					// not really an error; it's just that the test cannot accomodate this and needs to be extended.
					panic("unrecognized type to be serialized")
				}

			}
			buf.Reset()
			outputs := valueSerializerFun.Call(inputs)
			if len(outputs) != 2 {
				t.Fatalf("%v's SerializeValues method does not return 2 values", typeName)
			}
			bytesWritten = outputs[0].Interface().(int)
			if outputs[1].Interface() == nil {
				err = nil
			} else {
				err = outputs[1].Interface().(error)
			}
			if err != nil {
				t.Fatalf("%v's SerializeValues method returned err == %v", typeName, err)
			}
			if bytesWritten != expectedLen {
				t.Fatalf("%v's SerializeValues method returned %v for bytesWritten, expected %v", typeName, bytesWritten, expectedLen)
			}
			// bytes.Buffer has seperate read and write positions.
			outputs = valueDeserializerFun.Call([]reflect.Value{reflect.ValueOf(&buf)})
			if len(outputs) < 2 {
				t.Fatalf("%v's DeserializeValues methods returns <2 values", typeName)
			}

			if outputs[1].Interface() != nil {
				err = outputs[1].Interface().(error)
				t.Fatalf("For %v, Writing via SerializeValues and reading back via DeserializeValues gave error %v", typeName, err)
			}
			bytesRead := outputs[0].Interface().(int)
			if bytesRead != bytesWritten {
				t.Fatalf("%v's DeserializeValues and SerializeValues method differ in #bytes read/written", typeName)
			}
			if len(outputs)-1 != numInputs {
				t.Fatalf("For %v, DeserializeValues and SerializeValues do not have matching number of relevant arguments", typeName)
			}
			for j := 1; j < numInputs; j++ {
				switch inputTypes[j] {
				case feType:
					var feIn *bandersnatch.FieldElement = inputs[j].Interface().(*bandersnatch.FieldElement)
					var feGot bandersnatch.FieldElement = outputs[j+1].Interface().(bandersnatch.FieldElement)
					if !feIn.IsEqual(&feGot) {
						t.Fatalf("For %v, did not get back %v'th value (starting at 1) of type FieldElement via DeserializeValues that were serialized vie SerializeValues", typeName, j)
					}
				case boolType:
					var bitIn bool = inputs[j].Interface().(bool)
					var bitOut bool = outputs[j+1].Interface().(bool)
					if bitIn != bitOut {
						t.Fatalf("For %v, did not get back %v'th value (starting at 1) of type bool via DeserializeValues that were serialized vie SerializeValues", typeName, j)
					}
				default:
					// not really an error; it's just that the test cannot accomodate this and needs to be extended.
					panic("unrecognized type to be deserialized")
				}
			}
		}

	}
}
