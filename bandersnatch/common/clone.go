package common

import (
	"reflect"

	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

var anyType = utils.TypeOfType[any]()

// Untested:

func Clone[T any, Ptr interface{ *T }](p Ptr) Ptr {
	switch p := any(p).(type) {
	case interface{ Clone() Ptr }:
		return p.Clone()
	case interface{ Clone() any }:
		return p.Clone().(Ptr)
	default:
		pValue := reflect.ValueOf(p)
		pType := pValue.Type()
		if ok, errMsg := utils.DoesMethodExist(pType, "Clone", []reflect.Type{}, []reflect.Type{anyType}); !ok {
			panic(ErrorPrefix + "Clone free function called with receiver lacking appropriate Clone method:\n" + errMsg)
		}
		cloneMethod := reflect.ValueOf(p).MethodByName("Clone")
		result := cloneMethod.Call([]reflect.Value{})[0]
		return result.Interface().(Ptr)

	}
}
