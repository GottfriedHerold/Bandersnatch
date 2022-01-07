//go:build callcounters

// This file contains (dummy) implementations of the

package bandersnatch

const CallCountersActive = false

func IncrementCallCounter(id callcounters.Id) {
}

func BenchmarkWithCallCounters(b *testing.B) {
}
