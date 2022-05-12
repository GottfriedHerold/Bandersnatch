package pointserializer

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"reflect"
	"strings"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/bandersnatchErrors"
	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

type Verifier interface {
	Verify()
}

// consumeExpectRead reads and consumes len(expectToRead) bytes from input and reports an error if the read bytes differ from expectToRead.
// This is intended to read headers. Remember to use errors.Is to check the returned errors rather than == due to error wrapping.
//
// NOTES:
// Returns an error wrapping io.ErrUnexpectedEOF or io.EOF on end-of-file (io.EOF if the io.Reader was in EOF state to start with, io.ErrUnexpectedEOF if we encounter EOF after reading >0 bytes)
// On mismatch of expectToRead vs. actually read values, returns an error wrapping bandersnatchErrors.ErrDidNotReadExpectedString
// The returned error type satisfies the error interface and, if non-nil, contains a .Data member of type []byte and length bytes_read with the actually read bytes.
//
// Panics if expectToRead has length >MaxInt32. The function always (tries to) consume len(expectToRead) bytes, even if a mismatch is already early in the stream.
// Panics if expectToRead is nil or input is nil (unless len(expectToRead)==0)
func consumeExpectRead(input io.Reader, expectToRead []byte) (bytes_read int, returnedError *bandersnatchErrors.ErrorWithData[[]byte]) {
	if expectToRead == nil {
		panic("bandernatch / serialization: consumeExpectRead called with nil input for expectToRead")
	}
	l := len(expectToRead)
	if l > math.MaxInt32 {
		// should we return an error instead of panicking?
		panic(fmt.Errorf("bandersnatch / serialization: trying to read from io.Reader, expecting to read %v bytes, which is more than MaxInt32", l))
	}
	if l == 0 {
		return 0, nil
	}
	if input == nil {
		panic("bandersnatch / serialization: consumeExpectRead was called on nil reader")
	}
	var err error
	var buf []byte = make([]byte, l)
	bytes_read, err = io.ReadFull(input, buf)
	if err != nil {
		if errors.Is(err, io.ErrUnexpectedEOF) {
			data := buf[0:bytes_read]
			message := fmt.Sprintf("bandersnatch / deserialization: Unexpected EOF after reading %v out of %v bytes when reading header.\nReported error was %v.\nBytes expected were 0x%x, got 0x%x", bytes_read, len(expectToRead), err, expectToRead, data)
			returnedError = bandersnatchErrors.NewErrorWithData(err, message, data)
		} else if errors.Is(err, io.EOF) {
			expectToRead = copyByteSlice(expectToRead)
			message := fmt.Sprintf("bandersnatch / deserialization: Unexpected EOF when trying to read buffer.\nExpected to read 0x%x, got EOF instead", expectToRead)
			testutils.Assert(bytes_read == 0)
			data := make([]byte, 0) // no need to retain buf's underlying array.
			returnedError = bandersnatchErrors.NewErrorWithData(err, message, data)
		} else {
			data := buf[0:bytes_read]
			// empty message means that err.Error() is used as error message.
			returnedError = bandersnatchErrors.NewErrorWithData(err, "", data)
		}
		return
	}
	if !bytes.Equal(expectToRead, buf) {
		err = bandersnatchErrors.ErrDidNotReadExpectedString
		message := fmt.Sprintf("bandersnatch / deserialization: Unexpected Header encountered upon deserialization. Expected 0x%x, got 0x%x", expectToRead, buf)
		data := buf
		returnedError = bandersnatchErrors.NewErrorWithData(err, message, data)
	}
	// returnedError == nil iff no error occured so far
	return
}

// btyeOrderType is the reflect.Type of binary.ByteOrder. Since it is an interface type, we need this roundabout way of writing it down.
var byteOrderType reflect.Type = reflect.TypeOf((*binary.ByteOrder)(nil)).Elem()

// serializerParams is a global constant map that is used to lookup the names of setter and getter methods (which are called via reflection)
var serializerParams = map[string]struct {
	getter  string
	setter  string
	vartype reflect.Type
}{
	"endianness":   {getter: "GetEndianness", setter: "SetEndianness", vartype: byteOrderType},
	"bitheader":    {getter: "GetBitHeader", setter: "SetBitHeader", vartype: reflect.TypeOf(bitHeader{})},
	"bitheader2":   {getter: "GetBitHeader2", setter: "SetBitHeader2", vartype: reflect.TypeOf(bitHeader{})},
	"subgrouponly": {getter: "IsSubgroupOnly", setter: "SetSubgroupRestriction", vartype: reflect.TypeOf(bool(false))},
}

// hasParameter(serializer, parameterName) checks whether the type of serializer has setter and getter methods for the given parameter.
// The name of these getter and setter methods is looked up via the serializerParams map.
// parameterName is case-insensitive
func hasParameter[ValueType any, PtrType *ValueType](serializer PtrType, parameterName string) bool {
	parameterName = strings.ToLower(parameterName) // make parameterName case-insensitive
	paramInfo, ok := serializerParams[parameterName]

	// Technically, we could just meaningufully return false if parameterName is not found in serializerParams.
	// However, this is an internal function and we never intend to call hasParameter on anything but a fixed string which is supposed to be a key of the serializerParams map.
	// Hence, this can only occur due to a bug (e.g. a typo in the parameterName string).
	if !ok {
		panic("bandersnatch / serialization: makeCopyWithParams called with unrecognized parameter name")
	}

	serializerValue := reflect.ValueOf(serializer)
	// serializerType := reflect.TypeOf(serializer)
	setterMethod := serializerValue.MethodByName(paramInfo.setter)
	if !setterMethod.IsValid() {
		return false
	}
	getterMethod := serializerValue.MethodByName(paramInfo.getter)
	return getterMethod.IsValid()
}

// .With(...) forwards to this

// makeCopyWithParamsNew(serializer, parameterName, newParam) takes a serializer (anything with a Clone-method, really) and returns an
// indepepdent copy with the parameter given by parameterName replaced by newParam. The serializer argument is a pointer, but the returned value is not.
// parameterName is looked up in the global serializerParams map to obtain getter/setter method names. The function panics on failure.
func makeCopyWithParamsNew[SerializerType any, SerializerPtr interface {
	*SerializerType
	utils.Clonable[*SerializerType]
	Verifier
}](serializer SerializerPtr, parameterName string, newParam any) SerializerType {
	parameterName = strings.ToLower(parameterName) // make parameterName case-insensitive. The map keys are all lower-case
	paramInfo, ok := serializerParams[parameterName]
	if !ok {
		panic("bandersnatch / serialization: makeCopyWithParams called with unrecognized parameter name")
	}
	var clone SerializerPtr = serializer.Clone()
	cloneValue := reflect.ValueOf(clone)
	cloneType := cloneValue.Type()
	var typeName string = testutils.GetReflectName(cloneType) // name of parameter type. This is used for better error messages

	// This should be guaranteed by restrictions on type parameters.
	testutils.Assert(cloneType.Kind() == reflect.Pointer)

	setterMethod := cloneValue.MethodByName(paramInfo.setter)
	if !setterMethod.IsValid() {
		panic(fmt.Errorf("bandersnatch / serialization: makeCopyWithParams called with type %v lacking a setter method %v for the requested parameter %v", typeName, paramInfo.setter, parameterName))
	}
	if setterMethod.Type().NumOut() != 0 {
		panic(fmt.Errorf("bandersnatch / serialization: makeCopyWithParams called with type %v whose serializer %v returns a non-zero number of return values", typeName, paramInfo.setter))
	}
	newParamValue := reflect.ValueOf(newParam)
	newParamType := newParamValue.Type()
	if !newParamType.AssignableTo(paramInfo.vartype) {
		panic(fmt.Errorf("bandersnatch / serialization: makeCopyWithParams called with wrong type of argument %v. Expected argument type was %v", testutils.GetReflectName(newParamType), testutils.GetReflectName(paramInfo.vartype)))
	}
	setterMethod.Call([]reflect.Value{newParamValue})
	clone.Verify()
	return *clone
}

// getSerializerParam takes a serializer and returns the parameter stored under the key parameterName. The type of the return value depends on parameterName.
// parameterName is case-insensitive.
func getSerializerParam[ValueType any, PtrType *ValueType](serializer PtrType, parameterName string) interface{} {
	serializerType := reflect.TypeOf(serializer)
	receiverName := testutils.GetReflectName(serializerType) // used for diagnostics.

	parameterName = strings.ToLower(parameterName)
	paramInfo, ok := serializerParams[parameterName]
	if !ok {
		panic(fmt.Errorf("bandersnatch / serialization: getSerializerParam called on %v with unrecognized parameter name %v (lowercased)", receiverName, parameterName))
	}

	getterName := paramInfo.getter
	serializerValue := reflect.ValueOf(serializer)
	getterMethod := serializerValue.MethodByName(getterName)
	if !getterMethod.IsValid() {
		panic(fmt.Errorf("bandersnatch / serialization: getSerializeParam called on %v with parameter %v, but that type does not have a %v method", receiverName, parameterName, getterName))
	}
	getterType := getterMethod.Type()
	if getterType.NumIn() != 0 {
		panic(fmt.Errorf("bandersnatch / serialization: Getter Method %v called via getSerializeParam on %v takes >0 arguments", getterName, receiverName))
	}
	if getterType.NumOut() != 1 {
		panic(fmt.Errorf("bandersnatch / serialization: Getter Method %v called via getSerializeParam on %v returns %v != 1 arguments", getterName, receiverName, getterType.NumOut()))
	}
	retValue := getterMethod.Call([]reflect.Value{})[0]
	return retValue.Interface()
}

// Note: This returns a copy (by design). For v==nil, we return a fresh, empty non-nil slice.

// copyByteSlice returns a copy of the given byte slice. For nil inputs, returns an empty byte slice.
func copyByteSlice(v []byte) (ret []byte) {
	if v == nil {
		ret = make([]byte, 0)
		return
	}
	ret = make([]byte, len(v))
	L := copy(ret, v)
	testutils.Assert(L == len(v))
	return
}
