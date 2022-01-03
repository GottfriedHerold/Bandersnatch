//go:build ignore

package bandersnatch

import "strconv"

const CallCountersActive = true

func (v *callCounter) reset() {
	v.count_direct = 0
	v.count_modified = 0
}

func (v *callCounter) resetEval() {
	v.count_modified = 0
}

func (v *callCounter) recursivelyModifyBy(amount int) {
	v.count_modified += amount
	// var recurse *callCounter
	for recurse, multiplier := range v.addTo {
		recurse.count_modified += amount * multiplier
	}
	for recurse, multiplier := range v.addToRecursive {
		recurse.recursivelyModifyBy(amount * multiplier)
	}
}

func correctDependencies() {
	for _, cc := range callCounters {
		cc.resetEval()
	}
	for _, cc := range callCounters {
		if cc.count_direct != 0 {
			cc.recursivelyModifyBy(cc.count_direct)
		}
	}
}

func ResetCallCounters() {
	for _, cc := range callCounters {
		cc.reset()
	}
}

func (id CallCounterId) Get() (ret int, ok bool) {
	if id == "" {
		panic("callCounters: called Get with empty string")
	}
	correctDependencies()
	cc, ok := callCounters[id]
	if ok {
		ret = cc.count_modified
	}
	return
}

// getStringsBelowNode assumes correctDependencies has been called

func (cc *callCounter) getStringsBelowNode(indent string) (ret string, number int) {
	if !cc.display {
		if len(cc.subcounters) > 1 {
			panic("Non-displayed node has >= 2 subcounters")
		} else if len(cc.subcounters) == 1 && cc.displayremaining {
			panic("Non-display node has 1 subcounter and should display remaining")
		} else if len(cc.subcounters) == 1 {
			return cc.subcounters[0].getStringsBelowNode(indent)
		} else {
			return "", 0
		}
	}

	ret += indent
	number = cc.count_modified
	ret += strconv.Itoa(number)
	var s string = cc.displayName
	if s == "" {
		s = string(cc.id)
	}
	ret += " x " + s + "\n"

	indent += "  "
	childsum := 0
	haschildren := false
	for _, child := range cc.subcounters {
		childstring, childnumber := child.getStringsBelowNode(indent)
		ret += childstring
		childsum += childnumber
		if childstring != "" {
			haschildren = true
		}
	}
	if cc.displayremaining && haschildren {
		ret += indent
		ret += strconv.Itoa(cc.count_modified - childsum)
		ret += " x remaining, not covered by above\n"
	}
	return
}

func CallPointersStringToPrint() string {
	correctDependencies()
	var ret string
	for _, cc := range callCounters {
		if !cc.rootnode || !cc.display {
			continue
		}
		strtoadd, _ := cc.getStringsBelowNode("")
		ret += strtoadd
	}
	return ret
}

func IncrementCallCounter(id CallCounterId) {
	if id == "" {
		panic("callCounters: IncrementCallCounter called with empty id")
	}
	cc := callCounters[id]
	// We will panic (dereference nil) by design if id does not exist in callCounters.
	if !cc.initialized {
		panic("callCounters: IncrementCallCounter called on inintialized counter")
	}
	cc.count_direct++
}
