//go:build !callcounters

// This file is part of the fieldElements package. See the documentation of field_element.go for general remarks.

// This file contains (dummy) implementations of the CallCounters functions that can be used to count how often certain functions are called.
// The idea is to avoid having any runtime impact.

// NOTE: Callcounters themselves are a candidate for deprecation, so we don't use them consistently atm.

package fieldElements

import (
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/callcounters"
)

// NOTE: Go documentation tools do not always seem to recognize build tags properly, so this might show up twice.

// CallCountersActive is a constant whose value depends on build flags;
// it is true if CallCounters are active, which means we profile the number of calls to certain functions.
const CallCountersActive = false

// IncrementCallCounter increments the given call counter if callcounters are active (via build tags)
// It is a NoOp if callcounters are inactive
func IncrementCallCounter(id callcounters.Id) {
}

// TODO: Should be stop the timer? The version with active callcounters does. -- although this function is never actually called anyway.

// BenchmarkWithCallCounters stops the benchmark timing and includes callcounters in the report as custom fields.
// If callcounters are inactive, is a no-op.
func BenchmarkWithCallCounters(b *testing.B) {
}
