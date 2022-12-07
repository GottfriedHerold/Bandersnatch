//go:build callcounters

package fieldElements

import (
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/callcounters"
)

// This file is part of the fieldElements package. See the documentation of field_element.go for general remarks.

// This file is only compiled if tags=callcounters is set, otherwise callcounters_inactive is used.
// The difference is just that the functions defined here are replaced by no-ops

// NOTE: Callcounters themselves are a candidate for deprecation, so we don't use them consistently atm.

// CallCountersActive is a constant whose value depends on build flags;
// it is true if CallCounters are active, which means we profile the number of calls to certain functions.
const CallCountersActive = true

// IncrementCallCounter increments the given call counter if callcounters are active (via build tags)
// It is a NoOp if callcounters are inactive
func IncrementCallCounter(id callcounters.Id) {
	id.Increment()
}

// This might go to internal/callcounters

// TODO: Revisit

// BenchmarkWithCallCounters stops the benchmark timing and includes callcounters in the report as custom fields.
// If callcounters are inactive, is a no-op.
func BenchmarkWithCallCounters(b *testing.B) {
	b.StopTimer()
	reports := callcounters.ReportCallCounters(true, false)
	for _, item := range reports {
		b.ReportMetric(float64(item.Calls)/float64(b.N), item.Tag+"/op")
	}
}
