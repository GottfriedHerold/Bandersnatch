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

// TODO: Should the returned error capture the actually read bytes?

// consumeExpectRead reads and consumes len(expectToRead) bytes from input and reports an error if the read bytes differ from expectToRead.
// This is intended to read headers. Remember to use errors.Is to check the returned errors rather than == due to error wrapping.
//
// NOTES:
// Returns an error wrapping io.ErrUnexpectedEOF or io.EOF on end-of-file (io.EOF if the io.Reader was in EOF state to start with, io.ErrUnexpectedEOF if we encounter EOF after reading >0 bytes)
// On mismatch of expectToRead vs. actually read values, returns an error wrapping ErrDidNotReadExpectedString
func consumeExpectRead(input io.Reader, expectToRead []byte) (bytes_read int, err error) {
	l := len(expectToRead)
	if l > math.MaxInt32 {
		panic(fmt.Errorf("bandersnatch / serialization: Trying to read from io.Reader, expecting to read %v bytes, which is more than MaxInt32", l))
	}
	if l == 0 {
		return 0, nil
	}
	var buf []byte = make([]byte, l)
	bytes_read, err = io.ReadFull(input, buf)
	if err != nil {
		if errors.Is(err, io.ErrUnexpectedEOF) {
			// Replaces expectToRead by a copy of itself.
			// The reason is that we do not want the returned error to retain a copy of the passed expectToRead.
			// Otherwise, if this is used in an implementation of io.Writer, it would break part of the contract.
			deepCopyMaybeNilByteSlice(expectToRead, expectToRead)
			err = fmt.Errorf("bandersnatch / deserialization: Unexpected EOF after reading %v out of %v bytes when reading header.\nReported error was %w.\nBytes expected were 0x%x, got 0x%x", bytes_read, len(expectToRead), err, expectToRead, buf[0:bytes_read])
		}
		if errors.Is(err, io.EOF) {
			// Replaces expectToRead by a copy of itself.
			// The reason is that we do not want the returned error to retain a copy of the passed expectToRead.
			// Otherwise, if this is used in an implementation of io.Writer, it would break part of the contract.
			deepCopyMaybeNilByteSlice(expectToRead, expectToRead)
			err = bandersnatchErrors.NewWrappedError(err, fmt.Sprintf("bandersnatch / deserialization: Unexpected EOF when trying to read buffer.\nExpected to read 0x%x, got EOF instead", expectToRead))
		}
		return
	}
	if !bytes.Equal(expectToRead, buf) {
		err = bandersnatchErrors.NewWrappedError(bandersnatchErrors.ErrDidNotReadExpectedString, fmt.Sprintf("bandersnatch / deserialization: Unexpected Header encountered upon deserialization. Expected 0x%x, got 0x%x", expectToRead, buf))
	}
	return
}

var byteOrderType reflect.Type = reflect.TypeOf((*binary.ByteOrder)(nil)).Elem() // reflect.Type of binary.ByteOrder. Since it is an interface type, we need this roundabout way.

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

// hasParam(serializer, param) checks whether the type of serializer has setter and getter methods for parameter param.
// The name of these getter and setter methods is looked up via the serializerParams map.
// param is case-insensitive
func hasParam[ValueType any, PtrType *ValueType](serializer PtrType, param string) bool {
	param = strings.ToLower(param) // make params case-insensitive
	paramInfo, ok := serializerParams[param]
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

// replaced by generic version below, which has better compile-time guards.
/*

// makeCopyWithParams takes a serializer and returns a copy of it with param modified to newParam.
// It uses reflection and requires serializer to have a Clone() - method. To modify params, we look up getters / setters in serializerParams.
// If such a setter is not found or some other error is encountered, we panic.
func makeCopyWithParams(serializer interface{}, param string, newParam interface{}) (ret interface{}) {
	param = strings.ToLower(param) // make params case-insensitive
	paramInfo, ok := serializerParams[param]
	if !ok {
		panic("bandersnatch / serialization: makeCopyWithParams called with unrecognized parameter name")
	}

	serializerValue := reflect.ValueOf(serializer)
	// serializerType := reflect.TypeOf(serializer)
	cloneMethod := serializerValue.MethodByName("Clone")
	if !cloneMethod.IsValid() {
		panic("bandersnatch / serialization: makeCopyWithParams called with non-clonable serializer")
	}
	cloneMethodType := cloneMethod.Type()
	if cloneMethodType.NumIn() != 0 {
		panic("bandersnatch / serialization: makeCopyWithParams called with type whose Clone() method has >0 args")
	}
	if cloneMethodType.NumOut() != 1 {
		panic("bandersnatch / serialization: makeCopyWithParams called with type whose Clone() method returns != 1 args")
	}
	serializerClone := cloneMethod.Call([]reflect.Value{})[0]
	// serializerClone.Type() ought to be the same as serializerValue.Type(), up to pointer indirection. We care only about the result of clone(), which should be
	// a pointer (since we need to modify the result)
	serializerType := serializerClone.Type()
	if serializerType.Kind() != reflect.Ptr {
		// We could take the adress and work with values as well, but none of our serializers does that
		panic("bandersnatch / serialization: makeCopyWithParams calles with type whole Clone() method returns non-pointer type")
	}
	setterMethod := serializerClone.MethodByName(paramInfo.setter)
	if !setterMethod.IsValid() {
		panic(fmt.Errorf("bandersnatch / serialization: makeCopyWithParams called with type lacking a setter method %v for the requested parameter %v", paramInfo.setter, param))
	}
	newParamValue := reflect.ValueOf(newParam)
	newParamType := newParamValue.Type()
	if !newParamType.AssignableTo(paramInfo.vartype) {
		panic(fmt.Errorf("bandersnatch / serialization: makeCopyWithParams called with wrong type of argument %v. Expected argument type was %v", testutils.GetReflectName(newParamType), testutils.GetReflectName(paramInfo.vartype)))
	}
	setterMethod.Call([]reflect.Value{newParamValue})
	return serializerClone.Elem().Interface()
}
*/

// TODO: Delete above and rename the new version.

// makeCopyWithParamsNew(serializer, param, newParam) takes a serializer (anything with a Clone-method, really) and returns an
// indepepdent copy with param replaced by newParam. The serializer argument is a pointer, but the returned value is not.
// param is looked up in the global serializerParams map to obtain getter/setter method names. The function panics on failure.
func makeCopyWithParamsNew[SerializerType any, SerializerPtr interface {
	*SerializerType
	utils.Clonable[*SerializerType]
}](serializer SerializerPtr, param string, newParam any) SerializerType {
	param = strings.ToLower(param)
	paramInfo, ok := serializerParams[param]
	if !ok {
		panic("bandersnatch / serialization: makeCopyWithParams called with unrecognized parameter name")
	}
	var clone SerializerPtr = serializer.Clone()
	cloneValue := reflect.ValueOf(clone)
	cloneType := cloneValue.Type()
	var typeName string = testutils.GetReflectName(cloneType) // name of parameter type. This is used for better error messages

	// should be guananteed by restrictions on type parameters.
	testutils.Assert(cloneType.Kind() == reflect.Pointer)

	setterMethod := cloneValue.MethodByName(paramInfo.setter)
	if !setterMethod.IsValid() {
		panic(fmt.Errorf("bandersnatch / serialization: makeCopyWithParams called with type %v lacking a setter method %v for the requested parameter %v", typeName, paramInfo.setter, param))
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
	return *clone
}

// getSerializerParam takes a serializer and returns the parameter stored under the key param. The type of the return value depends on param.
func getSerializerParam[ValueType any, PtrType *ValueType](serializer PtrType, param string) interface{} {
	param = strings.ToLower(param)
	paramInfo, ok := serializerParams[param]
	serializerType := reflect.TypeOf(serializer)
	receiverName := testutils.GetReflectName(serializerType)
	if !ok {
		panic(fmt.Errorf("bandersnatch / serialization: getSerializerParam called on %v with unrecognized parameter name %v (lowercased)", receiverName, param))
	}

	getterName := paramInfo.getter
	serializerValue := reflect.ValueOf(serializer)
	getterMethod := serializerValue.MethodByName(getterName)
	if !getterMethod.IsValid() {
		panic(fmt.Errorf("bandersnatch / serialization: getSerializeParam called on %v with parameter %v, but that type does not have a %v method", receiverName, param, getterName))
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
