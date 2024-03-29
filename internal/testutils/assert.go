package testutils

import (
	"runtime/debug"
	"testing"
)

// TODO: Not really an assert (since the check is actually performed). Maybe rename to PanicUnless?

// Assert(condition) panics if condition is false; Assert(condition, error) panics if condition is false with panic(error).
func Assert(condition bool, err ...interface{}) {
	if len(err) > 1 {
		panic("bandersnatch / testutils: Assert can only handle 1 extra error argument")
	}
	if !condition {
		if len(err) == 0 {
			panic("This is not supposed to be possible")
		} else {
			panic(err[0])
		}
	}
}

// FatalUnless is used in testing functions. It checks if condition is satisfied; if not, the test is aborted with failure.
// formatString and args are used to construct the failure message.
//
// Note that this function does *NOT* panic on failure, but (manually) prints a stack dump in what looks like a panic at first glance.
// This is neccessary to pinpoint the line/file of the failing caller.
func FatalUnless(t *testing.T, condition bool, formatstring string, args ...any) {
	if !condition {
		debug.PrintStack()
		t.Fatalf(formatstring, args...)
	}
}
