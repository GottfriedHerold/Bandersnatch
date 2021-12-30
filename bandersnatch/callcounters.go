package bandersnatch

import (
	"strconv"
)

type CallCounterId string

type callCounter struct {
	id               CallCounterId
	display          bool
	displayName      string
	subcounters      []*callCounter
	addTo            map[*callCounter]int
	addToRecursive   map[*callCounter]int
	count_direct     int
	count_modified   int
	initialized      bool
	rootnode         bool
	displayremaining bool
}

var callCounters map[CallCounterId]*callCounter = make(map[CallCounterId]*callCounter)

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

// TODO: Mutex?

func ResetCallCounters() {
	for _, cc := range callCounters {
		cc.reset()
	}
}

func (id CallCounterId) Get(ret int, ok bool) {
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

func (id CallCounterId) CallCounterExists() (ret bool) {
	if id == "" {
		return false
	}
	_, ret = callCounters[id]
	return
}

func addDummyCounter(id CallCounterId) *callCounter {
	if id.CallCounterExists() {
		panic("Trying to add existing counter as dummy")
	}
	cc := callCounter{id: id}
	cc.addTo = make(map[*callCounter]int)
	cc.addToRecursive = make(map[*callCounter]int)
	callCounters[id] = &cc
	return &cc
}

func getCounter(id CallCounterId) (ret *callCounter, already_existed bool) {
	if id == "" {
		panic("callCounters: called getCounter with empty id")
	}
	ret, already_existed = callCounters[id]
	if !already_existed {
		if ret != nil {
			panic(0)
		}
		ret = addDummyCounter(id)
	}
	if ret == nil {
		panic(0)
	}
	return
}

func AddNewCallCounter(id CallCounterId, displayName string, parentName CallCounterId) *callCounter {
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
		cc.addToRecursive[parentcc] += 1 // this creates the map entry if it did not exist before
		if cc.addToRecursive[parentcc] == 0 {
			delete(cc.addToRecursive, parentcc)
		}
	} else {
		cc.rootnode = true
	}
	return cc
}

func (id CallCounterId) SetDisplayRemaining(displayremaining bool) {
	if id == "" {
		panic("callCounter: called SetDisplayRemaining with empty string")
	}
	cc, _ := getCounter(id)
	cc.displayremaining = displayremaining
}

func (cc *callCounter) SetDisplayRemaining(displayremaining bool) *callCounter {
	if cc == nil {
		panic("calling SetDisplayRemaining with nil receiver")
	}
	cc.displayremaining = displayremaining
	return cc
}

func CreateNewCallCounter(id CallCounterId, displayName string, doDisplay bool, attach CallCounterId, root bool) (cc *callCounter) {
	if id == "" {
		panic("callCounter: called CreateNewCounter with empty id string")
	}
	cc, _ = getCounter(id)
	cc.displayName = displayName
	cc.id = id
	cc.initialized = true
	cc.rootnode = root
	if attach != "" {
		parentcc, _ := getCounter(attach)
		parentcc.subcounters = append(parentcc.subcounters, cc)
	}
	return
}

func CreateAttachedCallCounter(id CallCounterId, displayName string, attachTo CallCounterId) (cc *callCounter) {
	cc = CreateNewCallCounter(id, displayName, true, attachTo, false)
	attachTo.SetDisplayRemaining(true)
	return
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

func (cc *callCounter) AddThisToTarget(targetId CallCounterId, multiplier int) *callCounter {
	if multiplier == 0 {
		return cc
	}
	if targetId == "" {
		panic("callCounter: called AddThisToTarget with empty target id")
	}
	target, _ := getCounter(targetId)
	cc.addToRecursive[target] += multiplier
	if cc.addToRecursive[target] == 0 {
		delete(cc.addToRecursive, target)
	}
	return cc
}

func (cc *callCounter) AddToThisFromSource(sourceId CallCounterId, multiplier int) *callCounter {
	if multiplier == 0 {
		return cc
	}
	if sourceId == "" {
		panic("callCounter: called AddToThisFromSource with empty source id")
	}
	source, _ := getCounter(sourceId)
	source.addToRecursive[cc] += multiplier
	if source.addToRecursive[cc] == 0 {
		delete(source.addToRecursive, cc)
	}
	return cc
}

// TODO: Set Add/SubRecursive
