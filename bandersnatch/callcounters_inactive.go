//go:build !ignore

package bandersnatch

const CallCountersActive = false

func ResetCallCounters() {
}

func (id CallCounterId) Get() (ret int, ok bool) {
	_, ok = callCounters[id]
	return // ret == 0

}

func CallPointersStringToPrint() string {
	return "Call Counters inactive\n"
}

func IncrementCallCounter(id CallCounterId) {
}
