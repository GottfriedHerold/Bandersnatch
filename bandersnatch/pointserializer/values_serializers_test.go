package pointserializer

import (
	"bytes"
	"errors"
	"math/rand"
	"reflect"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/bandersnatchErrors"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/common"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/errorsWithData"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/fieldElements"
	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

var defaultEndianness = common.DefaultEndian

var allValuesSerializers []valuesSerializer = []valuesSerializer{
	&valuesSerializerFeFe{fieldElementEndianness: defaultEndianness},
	&valuesSerializerHeaderFeHeaderFe{fieldElementEndianness: defaultEndianness},
	&valuesSerializerFe{fieldElementEndianness: defaultEndianness},
	&valuesSerializerFeCompressedBit{fieldElementEndianness: defaultEndianness},
	&valuesSerializerHeaderFe{fieldElementEndianness: defaultEndianness},
}

// for debugging reflect issues with unexported stuff. Currently unused:
type ValuesSerializerFeFe struct {
	valuesSerializerFeFe
}

// for debugging reflect issues with unexported stuff. Currently unused:
type ValuesSerializerHeaderFeHeaderFe struct {
	valuesSerializerHeaderFeHeaderFe
}

// for debugging reflect issues with unexported stuff. Currently unused:
type ValuesSerializerFe struct {
	valuesSerializerFe
}

// for debugging reflect issues with unexported stuff. Currently unused:
type ValuesSerializerFeCompressedBit struct {
	valuesSerializerFeCompressedBit
}

// for debugging reflect issues with unexported stuff. Currently unused:
type ValuesSerializerHeaderFe struct {
	valuesSerializerHeaderFe
}

// for debugging reflect issues with unexported stuff. Currently unused:
var allValuesSerializersExported []valuesSerializer = []valuesSerializer{
	&ValuesSerializerFeFe{valuesSerializerFeFe{fieldElementEndianness: defaultEndianness}},
	&ValuesSerializerHeaderFeHeaderFe{valuesSerializerHeaderFeHeaderFe{fieldElementEndianness: defaultEndianness}},
	&ValuesSerializerFe{valuesSerializerFe{fieldElementEndianness: defaultEndianness}},
	&ValuesSerializerFeCompressedBit{valuesSerializerFeCompressedBit{fieldElementEndianness: defaultEndianness}},
	&ValuesSerializerHeaderFe{valuesSerializerHeaderFe{fieldElementEndianness: defaultEndianness}},
}

var allValueSerializerTypes []reflect.Type = []reflect.Type{
	utils.TypeOfType[valuesSerializerFe](),
	utils.TypeOfType[valuesSerializerHeaderFe](),
	utils.TypeOfType[valuesSerializerFeFe](),
	utils.TypeOfType[valuesSerializerHeaderFeHeaderFe](),
	utils.TypeOfType[valuesSerializerFeCompressedBit](),
}

// TODO: Check that this works with the oop.go - functions.

// This tests that all value serializer types satisfy some constraints on the type such as having a Clone() - function.
// Since the return type of Clone depends on the type of the value serializer itself, we need reflection to express that
// (We could do it with generics, but then we could not write it as a loop over a global array that is shared across tests)
func TestValueSerializersSatisfyImplicitInterface(t *testing.T) {
	for _, valueSerializerType := range allValueSerializerTypes {
		// methods are defined on the pointer types
		valueSerializerPtrType := reflect.PtrTo(valueSerializerType)
		ok, reason := testutils.DoesMethodExist(valueSerializerPtrType, "Clone", []reflect.Type{}, []reflect.Type{valueSerializerPtrType})
		if !ok {
			t.Error(reason)
		}
		// NOTE: We might also check for SerializeValues and DeserializeValues here.
		// Currently, this is done in the TestValuesSerializersRountrip test.
	}
}

// TODO: Check that general SetParameter functions work and merge these two tests:

func TestRegognizedParameters(t *testing.T) {
	for _, valueSerializer := range allValuesSerializers {
		recognizedParams := valueSerializer.RecognizedParameters()
		for _, recognizedParam := range recognizedParams {
			_ = getSerializerParameter(valueSerializer, recognizedParam)
			if !valueSerializer.HasParameter(recognizedParam) {
				t.Fatalf("Parameter not reported as recognized")
			}
		}
		if valueSerializer.HasParameter("InvalidParameter") {
			t.Fatalf("Invalid Parameter was recognized as value")
		}
	}
}

func TestParameterSettings(t *testing.T) {
	for _, valueSerializer := range allValuesSerializers {
		name := testutils.GetReflectName(reflect.TypeOf(valueSerializer)) // name of the type of the serialzer. Used for error reporting.
		var params []string = valueSerializer.RecognizedParameters()
		for _, param := range params {
			if !hasSetterAndGetterForParameter(valueSerializer, param) {
				t.Errorf("%v does not have parameter named %v as claimed", name, param)
			}
		}
	}
}

// This tests whether OutputLength, RecognizedParameters and HasParameter can be called on nil pointers of the appropriate type
func TestQueryFunctionsCallableOnNil(t *testing.T) {
	for _, valueSerializerType := range allValueSerializerTypes {
		valueSerializerType = reflect.PtrTo(valueSerializerType)
		zeroValue := reflect.Zero(valueSerializerType).Interface().(valuesSerializer)
		_ = zeroValue.OutputLength()
		_ = zeroValue.RecognizedParameters()
		_ = zeroValue.HasParameter("foo")
	}
}

// This Test runs Validate on all values Serializers that we defined above.
func TestAllValuesSerializersValidate(t *testing.T) {
	for _, valueSerializer := range allValuesSerializers {
		valueSerializer.(validater).Validate()
	}
}

// Checks that Clone methods are callable
func TestAllValuesSerializersClonable(t *testing.T) {
	for _, valueSerializer := range allValuesSerializers {
		// TestValuesSerializersSatisfyImplicitInterface takes care about whether a Clone method exists with the correct type.
		// (It is even more restrictive than this test).
		// The purpose here is to actually CALL the Clone method and Validate the result.
		clone := testutils.CallMethodByName(valueSerializer, "Clone")[0]
		if clone == nil {
			t.Fatalf("Clone returned nil")
		}
		cloneSerializer, ok := clone.(valuesSerializer)
		if !ok {
			t.Fatalf("Cloning a valuesSerializer did not work. See output of TestValuesSerializersSatisfyImplicitInterface.")
		}
		cloneSerializer.Validate()
	}
}

// This tests that all valuesSerializer types have SerializeValues and DeserializeValues methods that actually roundtrip elements.
func TestValuesSerializersRoundtrip(t *testing.T) {
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
		feType := reflect.TypeOf(&fieldElements.FieldElement{})
		boolType := reflect.TypeOf(bool(true))
		const iterations = 20
		var buf bytes.Buffer
		inputs[0] = reflect.ValueOf(&buf)
		for i := 0; i < iterations; i++ {
			expectedLen := 0
			for j := 1; j < numInputs; j++ {
				switch inputTypes[j] {
				case feType:
					var fe fieldElements.FieldElement
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
			if valuesSerializer.OutputLength() != int32(expectedLen) {
				t.Fatalf("%v's OutputLength does not return expected value", typeName)
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
					var feIn *fieldElements.FieldElement = inputs[j].Interface().(*fieldElements.FieldElement)
					var feGot fieldElements.FieldElement = outputs[j+1].Interface().(fieldElements.FieldElement)
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
			// Try behaviour with faulty buf:
			designatedErr := errors.New("some error")
			for faultPos := 0; faultPos < expectedLen; faultPos++ {
				faultyBuf := testutils.NewFaultyBuffer(faultPos, designatedErr) // Note: faultyBuf is a pointer (buf above was not)
				inputs[0] = reflect.ValueOf(faultyBuf)
				outputs = valueSerializerFun.Call(inputs)
				inputs[0] = reflect.ValueOf(&buf)
				bytesWritten = outputs[0].Interface().(int)
				if outputs[1].Interface() == nil {
					t.Fatalf("For %v, writing to a faulty buffer did not result in an error", typeName)
				}
				err = outputs[1].Interface().(error)
				if bytesWritten != faultPos {
					t.Fatalf("Could not write to faulty buffer until error for %v", typeName)
				}
				if !errors.Is(err, designatedErr) {
					t.Fatalf("Did not get expected designated error upon writing for %v", typeName)
				}
				Partial := errorsWithData.GetDataFromError[bandersnatchErrors.WriteErrorData](err).PartialWrite
				if Partial != (faultPos != 0) {
					t.Fatalf("Did not set PartialWrite correctly on error for %v", typeName)
				}

				// Read back:
				outputs = valueDeserializerFun.Call([]reflect.Value{reflect.ValueOf(faultyBuf)})
				bytesRead = outputs[0].Interface().(int)
				if bytesRead != bytesWritten {
					t.Fatalf("Did not read as much as written for faulty buffer for %v", typeName)
				}
				if outputs[1].Interface() == nil {
					t.Fatalf("For %v, reading from faulty buf gave no error", typeName)
				}
				err = outputs[1].Interface().(error)
				if !errors.Is(err, designatedErr) {
					t.Fatalf("Did not get expceted designated error reading from faulty buffer for %v", typeName)
				}
				Partial = errorsWithData.GetDataFromError[bandersnatchErrors.ReadErrorData](err).PartialRead
				if Partial != (faultPos != 0) {
					t.Fatalf("Did not set PartialRead correctly on error for %v", typeName)
				}

			}
		}
	}
}
