package bandersnatch

import "strconv"

type callCounter struct {
	id               string
	display          bool
	displayName      string
	subcounters      []*callCounter
	addTo            []*callCounter
	addToRecursive   []*callCounter
	subFrom          []*callCounter
	subFromRecursive []*callCounter
	count_direct     int
	count_modified   int
	initialized      bool
	rootnode         bool
	displayremaining bool
}

func (v *callCounter) reset() {
	v.count_direct = 0
	v.count_modified = 0
}

func (v *callCounter) resetEval() {
	v.count_modified = 0
}

func (v *callCounter) recursivelyModifyBy(amount int) {
	v.count_modified += amount
	var recurse *callCounter
	for _, recurse = range v.addTo {
		recurse.count_modified += amount
	}
	for _, recurse = range v.subFrom {
		recurse.count_modified -= amount
	}
	for _, recurse = range v.addToRecursive {
		recurse.recursivelyModifyBy(amount)
	}
	for _, recurse = range v.subFromRecursive {
		recurse.recursivelyModifyBy(-amount)
	}
}

func correctDependencies() {
	for _, cc := range callCounters {
		cc.resetEval()
	}
	for _, cc := range callCounters {
		cc.recursivelyModifyBy(cc.count_direct)
	}
}

var callCounters map[string]*callCounter = make(map[string]*callCounter)

// TODO: Mutex?

func Reset() {
	for _, cc := range callCounters {
		cc.reset()
	}
}

func Get(id string) (ret int, ok bool) {
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
	ret += " x " + cc.id + ": "
	ret += cc.displayName + "\n"

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

func StringToPrint() string {
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

func CounterExists(id string) (ret bool) {
	if id == "" {
		return false
	}
	_, ret = callCounters[id]
	return
}

func addDummyCounter(id string) *callCounter {
	if CounterExists(id) {
		panic("Trying to add existing counter as dummy")
	}
	cc := callCounter{id: id}
	callCounters[id] = &cc
	return &cc
}

func getCounter(id string) (ret *callCounter, ok bool) {
	if id == "" {
		panic("Counter id is empty")
	}
	ret, ok = callCounters[id]
	if !ok {
		if ret != nil {
			panic(0)
		}
		ret = addDummyCounter(id)
	}
	if ok && ret == nil {
		panic(0)
	}
	return
}

func AddNewCounter(id string, displayName string, parentName string) *callCounter {
	if id == "" {
		panic("callCounter: called AddNewCounter with empty string")
	}
	var cc *callCounter
	var alreadyExists bool
	cc, alreadyExists = getCounter(id)
	if alreadyExists && cc.initialized {
		panic("Added the same counter twice")
	}
	cc.display = true
	cc.displayName = displayName
	cc.initialized = true
	if parentName != "" {
		cc.rootnode = false
		var parentcc *callCounter
		parentcc, alreadyExists = getCounter(parentName)
		if !alreadyExists || !parentcc.initialized {
			parentcc.rootnode = true
			parentcc.display = true
			parentcc.initialized = true
		}
		parentcc.subcounters = append(parentcc.subcounters, cc)
		cc.addToRecursive = append(cc.addToRecursive, parentcc)
	} else {
		cc.rootnode = true
	}
	return cc
}

func SetDisplayRemaining(id string, displayremaining bool) {
	if id == "" {
		panic("callCounter: called SetDisplayRemaining with empty string")
	}
	cc, _ := getCounter(id)
	cc.displayremaining = displayremaining
}

func CreateNewCounter(id string, displayName string, doDisplay bool, attach string, root bool) {
	if id == "" {
		panic("callCounter: called CreateNewCounter with empty id string")
	}
	cc, _ := getCounter(id)
	cc.displayName = displayName
	cc.id = id
	cc.initialized = true
	cc.rootnode = root
	if attach != "" {
		parentcc, _ := getCounter(attach)
		parentcc.subcounters = append(parentcc.subcounters, cc)
	}
}

func IncrementCallCounter(id string) {
	if id == "" {
		panic("callCounters: IncrementCallCounter called with empty id")
	}
	cc, _ := callCounters[id]
	cc.count_direct++ // This will panic (dereference nil) by design if id does not exist in callCounters.
}

// TODO: Set Add/SubRecursive
