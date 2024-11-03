// This package collects testing utilities that are used to test correctness of errors and their error messages
// for errors returned utilizing the [errorsWithData] package.
//
// NOTE: [errorsWithData] imports internal/testutils for its own (internal) tests.
// Since the utility functions defined here need to import [errorWithData], we need to put them in a separate package
// to avoid circular imports.
//
// We might consider merging this internal package into [errorsWithData] itself.
package testutils_errors

import (
	"fmt"
	"runtime/debug"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/errorsWithData"
)

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
