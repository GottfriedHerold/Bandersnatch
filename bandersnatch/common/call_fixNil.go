package common

import "reflect"

// CallFunction_FixNil is a wrapper around [reflect.Value] Call(...) method that treats zero [reflect.Value] as nil
//
// For a [reflect.Value] fun (that is supposed to be callable), CallFunction_FixNil(fun, in) is equivalent to fun.Call(in), except that
// a zero [reflect.Value] appearing as input to fun -- which is the result of [reflect.ValueOf](nil) -- is actually treated as nil rather than a panic-causing invalid [reflect.Value].
// This function modfifies the given `in` slice in-place to replace zero [reflect.Value]s by appropriate replacements.
//
// NOTE: This method actually works around a design flaw in the way reflect handles nil, cf.  https://github.com/golang/go/issues/51649
// There are serious considerations to actually change the behaviour of the standard library, which would make this function obsolete.
// Potentially, such changes might actually break this function.
// This function may or may not work for variadic functions.
func CallFunction_FixNil(fun reflect.Value, in []reflect.Value) (out []reflect.Value) {
	if fun.Kind() != reflect.Func {
		panic(ErrorPrefix + "CallMethod_FixNil expects a reflect.Value wrapping a function as first argument")
	}
	funType := fun.Type()
	for pos, inputArg := range in {
		if !inputArg.IsValid() {
			InType := funType.In(pos) // type of input argument
			switch InType.Kind() {
			case reflect.Interface,
				reflect.Pointer,
				reflect.Chan,
				reflect.Func,
				reflect.Slice,
				reflect.Map:
				in[pos] = reflect.Zero(InType)
			default:
				panic(ErrorPrefix + "CallMethod_FixNil asked to call function, where argument list contains nil; however, nil is not valid for the function's expected argument type.")
			}
		}
	}
	return fun.Call(in)
}
