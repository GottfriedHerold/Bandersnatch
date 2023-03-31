package errorsWithData

import "fmt"

func EnsureTestsValid_Final(errs ...ErrorWithData_any) {
	for _, err := range errs {
		if internalIssue := err.ValidateError_Final(); internalIssue != nil {
			panic(fmt.Errorf("issue in error: %w", internalIssue))
		}
	}
}

func EnsureTestsValid_Base(errs ...ErrorWithData_any) {
	for _, err := range errs {
		if internalIssue := err.ValidateError_Final(); internalIssue != nil {
			panic(fmt.Errorf("issue in error: %w", internalIssue))
		}
	}
}

func EnsureTestsValid_Syntax(errs ...ErrorWithData_any) {
	for _, err := range errs {
		if internalIssue := err.ValidateError_Final(); internalIssue != nil {
			panic(fmt.Errorf("issue in error: %w", internalIssue))
		}
	}
}
