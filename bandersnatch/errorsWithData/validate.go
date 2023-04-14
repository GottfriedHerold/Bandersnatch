package errorsWithData

import "fmt"

// EnsureTestsValid_Final runs ValidateError_Final on each of its arguments and panics if there is an issue.
func EnsureTestsValid_Final(errs ...ErrorWithData_any) {
	for _, err := range errs {
		if internalIssue := err.ValidateError_Final(); internalIssue != nil {
			panic(fmt.Errorf("issue in error: %w", internalIssue))
		}
	}
}

// EnsureTestsValid_Final runs ValidateError_Base on each of its arguments and panics if there is an issue.
func EnsureTestsValid_Base(errs ...ErrorWithData_any) {
	for _, err := range errs {
		if internalIssue := err.ValidateError_Base(); internalIssue != nil {
			panic(fmt.Errorf("issue in error: %w", internalIssue))
		}
	}
}

// EnsureTestsValid_Syntax runs ValidateSyntax on each of its arguments and panics if there is an issue.
func EnsureTestsValid_Syntax(errs ...ErrorWithData_any) {
	for _, err := range errs {
		if internalIssue := err.ValidateSyntax(); internalIssue != nil {
			panic(fmt.Errorf("issue in error: %w", internalIssue))
		}
	}
}
