//go:build !callcounters

package bandersnatch

import (
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/callcounters"
)

// This file is only compiled if tags=callcounters is set, otherwise
// callcounters_inactive is used.
// The difference is just that the functions defined here are replaced by no-ops

const CallCountersActive = true

func IncrementCallCounter(id callcounters.Id) {
	id.Increment()
}

// This might go to internal/callcounters

func BenchmarkWithCallCounters(b *testing.B) {
	b.StopTimer()
	reports := callcounters.ReportCallCounters(true, false)
	for _, item := range reports {
		b.ReportMetric(float64(item.Calls)/float64(b.N), item.Tag)
	}
}
