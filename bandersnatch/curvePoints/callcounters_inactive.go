//go:build !callcounters

// This file contains (dummy) implementations of the IncrementCallCounter and BenchmarkWithCallCounters
// functions. This file is selected if callcounters is not an active build tag
// and ensures we have no overhead from these benchmark-utils.

package bandersnatch

import (
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/callcounters"
)

// NOTE: Godoc does not seem to recognize build tags properly, so this might show up twice.

// CallCountersActive is a constant whose value depends on build flags;
// it is true if CallCounters are active, which means we profile the number of calls to certain functions.
const CallCountersActive = false

// IncrementCallCounter increments the given call counter if callcounters are active (via build tags)
// It is a NoOp if callcounters are inactive
func IncrementCallCounter(id callcounters.Id) {
}

func BenchmarkWithCallCounters(b *testing.B) {
}
