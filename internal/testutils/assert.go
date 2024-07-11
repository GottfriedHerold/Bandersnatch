package testutils

import (
	"fmt"
	"runtime/debug"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/errorsWithData"
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

// CheckErrorValidity checks that err.ValidateError_Finall() returns nil if err is non-nil/
//
// This is just provided to avoid having to write the same diagnostic message for the failed test a lot of times.
func CheckErrorValidity(t *testing.T, err errorsWithData.ErrorWithData_any) {
	if err == nil {
		return
	}
	validate := err.ValidateError_Final()
	if validate != nil {
		t.Fatalf("error %v failed ValidateError_Final.\nThe validation result was\n%v", err, validate)
	}

}

// CheckErrorMessage checks that err.Error() matches the result of fmt.Sprintf(expectedMessageFmtString, arg...)
//
// If err is an ErrorWithData_any, we also run [CheckErrorValidity].
//
// This mostly serves to write tests that check for a particular error message concisely.
func CheckErrorMessage(t *testing.T, err error, expectedMessageFmtString string, args ...any) {
	expectedMessage := fmt.Sprintf(expectedMessageFmtString, args...)
	if err == nil {
		t.Fatalf("Unexpected nil error rather than an error with message\n%v", expectedMessage)
	}

	// Run that before the check below.
	// Note that if this fails, the check below will also likely fail (due to in-band diagnostics inside errorMessage),
	// but the failure report from CheckErrorValidity is more useful.
	if errWithData, ok := err.(errorsWithData.ErrorWithData_any); ok {
		CheckErrorValidity(t, errWithData)
	}

	errorMessage := err.Error()
	if errorMessage != expectedMessage {
		debug.PrintStack()
		t.Fatalf("Unexpected error message: Expected\n%v\nGot:\n%v", expectedMessage, errorMessage)
	}
}
