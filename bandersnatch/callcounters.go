package bandersnatch

/*
import (
	"fmt"
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

// TODO: Mutex?
var callCounters map[CallCounterId]*callCounter = make(map[CallCounterId]*callCounter)

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
	cc.display = doDisplay
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

// debug function
func printCallCounterStructure() {
	for id, cc := range callCounters {
		fmt.Println("\nCallCounter map key ", id)
		fmt.Println("id: ", cc.id, " display as: ", cc.displayName)
		fmt.Println("Initialized:", cc.initialized, "  displayed:", cc.display, "  Is root node:", cc.rootnode, "  Display remaining:", cc.displayremaining)
		fmt.Println("currentVal:", cc.count_direct, "  adjusted: ", cc.count_modified)
		fmt.Println(len(cc.subcounters), " subcounters:")
		for _, subcc := range cc.subcounters {
			fmt.Print("    ", subcc.id)
		}
		if len(cc.subcounters) > 0 {
			fmt.Println()
		}
		fmt.Println(len(cc.addTo), " counters added to:")
		for subid, multiplier := range cc.addTo {
			fmt.Print("    ", subid.id, " x ", multiplier)
		}
		if len(cc.addTo) > 0 {
			fmt.Println()
		}
		fmt.Println(len(cc.addToRecursive), " counters recursively added to:")
		for subid, multiplier := range cc.addToRecursive {
			fmt.Print("    ", subid.id, " x ", multiplier)
		}
		if len(cc.addToRecursive) > 0 {
			fmt.Println()
		}
	}
}

type ccreport struct {
	tag    string
	number int
}

func getActiveCallCountersBelowNode(cc *callCounter) (ret []ccreport, value int) {
	ret = make([]ccreport, 0)
	if !cc.display || cc.count_modified == 0 {
		return
	}
	value = cc.count_modified
	childsum := 0
	haschildren := false

	ret = append(ret, ccreport{tag: string(cc.id), number: cc.count_modified})
	for _, child_cc := range cc.subcounters {
		childreport, childvalue := getActiveCallCountersBelowNode(child_cc)
		if childvalue == 0 {
			continue
		}
		haschildren = true
		childsum += childvalue
		ret = append(ret, childreport...)
	}
	if haschildren && cc.displayremaining && childsum < value {
		ret = append(ret, ccreport{tag: "Other" + string(cc.id), number: value - childsum})
	}
	return
}

func PrintCallCounters() {
	fmt.Println(CallPointersStringToPrint())
}
*/
