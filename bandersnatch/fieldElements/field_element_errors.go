package fieldElements

import (
	"errors"
	"fmt"
	"strings"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/errorsWithData"
)

// ErrorPrefix is the prefix used by all error message strings originating from this package.
const ErrorPrefix = "bandersnatch / field element: "

var ErrCannotRepresentAsUInt64 = errors.New(ErrorPrefix + "cannot represent field element as a uint64")
var ErrDivisionByZero = errors.New(ErrorPrefix + "division by zero")

// These are the errors that can occur during (de)serialization.
var (
	ErrPrefixDoesNotFit             error = errors.New(ErrorPrefix + "while trying to serialize a field element with a prefix, the prefix did not fit, because the number was too large")
	ErrPrefixLengthInvalid          error = errors.New(ErrorPrefix + "in FieldElement deserializitation, an invalid prefix length > 8 was requested")
	ErrPrefixMismatch               error = errors.New(ErrorPrefix + "during deserialization, the read prefix did not match the expected one")
	ErrNonNormalizedDeserialization error = errors.New(ErrorPrefix + "during FieldElement deserialization, the read number was not the minimal representative modulo BaseFieldSize")
)

// MultiInversionError is an interface extending error.
// It is used to indicate errors in multiinversion algorithms.
type MultiInversionError = errorsWithData.ErrorWithGuaranteedParameters[MultiInversionErrorData]

// IMPORTANT NOTE: Some callers in the bandersnatch package actually recover the panic and rely on the fact that no changes to args are made on panic.

// TODO: Base on ErrorWithData; this is a separate class for historic reasons.
// Needs to be redone anyway;

// errMultiInversionEncounteredZero is a (stateful) error either returned by or provided as argument to panic by functions that perform
// simultaneous inversion of multiple field elements.
//
// It contains information about which elements were zero.
//
// Satisfies the ErrorWithGuaranteedData[MultiInversionErrorData] interface
type errMultiInversionEncounteredZero struct {
	ZeroIndices         []bool // indices (starting from 0) of the field elements that were zero, i.e. in a call (ignoring argument types) MultiInvertEq(0, 1, 2, 0, 0), we would have ZeroIndices = [0, 3, 4]
	NumberOfZeroIndices int    // number of field elements that were zero when multi-inversion was requested. In the above example, would be 3
	s                   string // internal: (static) error string that is to be displayed by Error(). Note that Error() also outputs additional information about ZeroIndices etc.
}

// Just for IDE; this must correspond to field names of errMultiInversionEncounteredZero.
const argNameZeroIndices = "ZeroIndices"
const argNameNumberOfZeroIndices = "NumberOfZeroIndices"

func (err *errMultiInversionEncounteredZero) Unwrap() error {
	return ErrDivisionByZero
}

func (err *errMultiInversionEncounteredZero) GetParameter(parameterName string) (value any, wasPresent bool) {
	switch parameterName {
	case argNameZeroIndices:
		return err.ZeroIndices, true
	case argNameNumberOfZeroIndices:
		return err.NumberOfZeroIndices, true
	default:
		return nil, false
	}
}

func (err *errMultiInversionEncounteredZero) HasParameter(parameterName string) bool {
	switch parameterName {
	case argNameNumberOfZeroIndices, argNameZeroIndices:
		return true
	default:
		return false
	}
}

func (err *errMultiInversionEncounteredZero) GetData() MultiInversionErrorData {
	return MultiInversionErrorData{ZeroIndices: err.ZeroIndices, NumberOfZeroIndices: err.NumberOfZeroIndices}
}

func (err *errMultiInversionEncounteredZero) GetAllParameters() map[string]any {
	return map[string]any{argNameNumberOfZeroIndices: err.NumberOfZeroIndices, argNameZeroIndices: err.ZeroIndices}
}

// Error is provided to satisfy the error interface (for pointer receivers). We report the stored string s together with information about ZeroIndices.
func (err *errMultiInversionEncounteredZero) Error() string {
	var b strings.Builder
	b.WriteString(err.s)
	if err.NumberOfZeroIndices <= 0 {
		fmt.Fprintf(&b, "\nThe number of zero indices stored as metadata is %v <= 0. This should only occur if you are creating uninitialized ErrMultiInversionEncounteredZero errors manually.", err.NumberOfZeroIndices)
		return b.String()
	}
	if err.NumberOfZeroIndices == 1 {
		for i := 0; i < len(err.ZeroIndices); i++ {
			if err.ZeroIndices[i] {
				fmt.Fprintf(&b, "\nThe %v'th argument (counting from 0) was the only one that was zero.", i)
				return b.String()
			}
		}
		fmt.Fprintf(&b, "\nInternal bug: the number of zero indices stored as metadata is 1, but no zero index was contained in the metadata.")
		return b.String()
	}
	var indices []int = make([]int, 0, err.NumberOfZeroIndices)
	for i := 0; i < len(err.ZeroIndices); i++ {
		if err.ZeroIndices[i] {
			indices = append(indices, i)
		}
	}
	if len(indices) != err.NumberOfZeroIndices {
		fmt.Fprintf(&b, "\nInternal bug: the number of zero indices stored as metadata does not match the number of field elements that were reported as zero. Error reporting may be unreliable.")
	}
	fmt.Fprintf(&b, "\nThere were %v numbers (starting indexing with 0) that were 0 in the call. Those are:\n %v", err.NumberOfZeroIndices, indices)
	return b.String()
}

// generateMultiDivisionByZeroPanic is a helper function for MultiInvertEq and MultiInvertSliceEq.
//
// It creates the actual non-nil error that includes diagnostics which field Elements were zero.
func generateMultiDivisionByZeroPanic(fieldElements []*bsFieldElement_64, s string) errorsWithData.ErrorWithGuaranteedParameters[MultiInversionErrorData] {
	var err errMultiInversionEncounteredZero
	err.s = s
	err.NumberOfZeroIndices = 0
	err.ZeroIndices = make([]bool, len(fieldElements))
	for i, fe := range fieldElements {
		if fe.IsZero() {
			err.ZeroIndices[i] = true
			err.NumberOfZeroIndices++
		}
	}
	return &err
}
