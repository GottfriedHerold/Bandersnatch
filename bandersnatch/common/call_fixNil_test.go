package common

import (
	"reflect"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
)

func TestCallFunc_FixNil(t *testing.T) {
	f := func(a any, b *int, c int) (any, int, int) {
		var ret int
		if a == nil {
			ret += 2
		}
		if b == nil {
			ret += 1
		}
		return a, ret, c
	}
	fValue := reflect.ValueOf(f)
	b := 5
	in := []reflect.Value{reflect.ValueOf(3), reflect.ValueOf(&b), reflect.ValueOf(b)}
	out := CallFunction_FixNil(fValue, in)
	testutils.FatalUnless(t, out[0].Interface() == 3, "failure (1), argument 1")
	testutils.FatalUnless(t, out[1].Interface() == 0, "failure (1), argument 2")
	testutils.FatalUnless(t, out[2].Interface() == 5, "failure (1), argument 3")

	in = []reflect.Value{reflect.ValueOf(nil), reflect.ValueOf(&b), reflect.ValueOf(b)}
	out = CallFunction_FixNil(fValue, in)
	testutils.FatalUnless(t, out[0].Interface() == nil, "failure (2), argument 1")
	testutils.FatalUnless(t, out[1].Interface() == 2, "failure (2), argument 2")
	testutils.FatalUnless(t, out[2].Interface() == 5, "failure (2), argument 3")

	in = []reflect.Value{reflect.ValueOf(nil), reflect.ValueOf(nil), reflect.ValueOf(b)}
	out = CallFunction_FixNil(fValue, in)
	testutils.FatalUnless(t, out[0].Interface() == nil, "failure (3), argument 1")
	testutils.FatalUnless(t, out[1].Interface() == 3, "failure (3), argument 2")
	testutils.FatalUnless(t, out[2].Interface() == 5, "failure (3), argument 3")
	testutils.FatalUnless(t, in[0].Interface() == nil, "faillure (3), input change 1")
	testutils.FatalUnless(t, in[1].Interface() == (*int)(nil), "faillure (3), input change 1")

	in = []reflect.Value{reflect.ValueOf(nil), reflect.ValueOf(nil), reflect.ValueOf(nil)}
	didPanic := testutils.CheckPanic(CallFunction_FixNil, fValue, in)
	testutils.FatalUnless(t, didPanic, "CallFunc_FixNil did not panic(1)")

	in = []reflect.Value{reflect.ValueOf(3), reflect.ValueOf(&b), reflect.ValueOf(b)}
	didPanic = testutils.CheckPanic(CallFunction_FixNil, reflect.ValueOf(10), in)
	testutils.FatalUnless(t, didPanic, "CallFunc_FixNil did not panic(2)")
}
