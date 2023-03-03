package testutils

import (
	"sync"
	"testing"
)

// NOTE:
// Technically, in a sequence of calls MakeVariableEscape(&x); MakeVariableEscape(&y)
// the compiler could elide the intermediate unlock - lock calls to the mutex without violating any guarantees, so there is now a single protection code section.
// Then the store of &x could be elided and optimzed away.
//
// Still, given that the compiler is doing a generally poor job at optimizing (the whole language was not designed for optimizations anyway), fixing
// this is probably not worth it.

var DumpGeneral struct {
	x any // a single slot only. This is actually enough.
	m sync.Mutex
}

// MakeVariableEscape(b, &x) registers a cleanup function that keeps &x in a globally accessible variable, where b is a benchmark.
//
// The purpose of this is that calling MakeVariableEscape(b, &x) will prevent the compiler from optimizing away writes to x.
func MakeVariableEscape[T any, Ptr *T](b *testing.B, arg Ptr) {
	b.Cleanup(func() {
		DumpGeneral.m.Lock()
		DumpGeneral.x = arg
		DumpGeneral.m.Unlock()
	})
}
