package testutils

import (
	"sync"
	"testing"
)

var DumpGeneral struct {
	x any
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
