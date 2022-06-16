//go:build callcounters

package fieldElement

import (
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/callcounters"
)

// This file is only compiled if tags=callcounters is set, otherwise
// callcounters_inactive is used.
// The difference is just that the functions defined here are replaced by no-ops

// CallCountersActive is a constant whose value depends on build flags;
// it is true if CallCounters are active, which means we profile the number of calls to certain functions.
const CallCountersActive = true

// IncrementCallCounter increments the given call counter if callcounters are active (via build tags)
// It is a NoOp if callcounters are inactive
func IncrementCallCounter(id callcounters.Id) {
	id.Increment()
}

// This might go to internal/callcounters

func BenchmarkWithCallCounters(b *testing.B) {
	b.StopTimer()
	reports := callcounters.ReportCallCounters(true, false)
	for _, item := range reports {
		b.ReportMetric(float64(item.Calls)/float64(b.N), item.Tag+"/op")
	}
}
