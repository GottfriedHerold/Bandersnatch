package callcounters

// This package contains code for call counters.
// Call counters are just benchmarking counters that are intended to be used
// to count how often certain functions are called and display the output
// in an organized fashion.

/*
Usage example:
var _ = AddNewCallCounter("ExpensiveOperations", "", "")
var _ = AddNewCallCounter("SquareRoot", "Square Root", "ExpensiveOperations")
var _ = AddNewCallCounter("CubeRoot", "Cubic Root", "ExpensiveOperations")
func SquareRoot() {
	IncrementCallCounter("SquareRoot")
	...
}

[Note:  The calls to AddNewCallCounter(NewCounter, Display Name, Parent) can be in any order here.
		You can create a counter with a reference to the parent before creating the parent.
		Of course, you can also use an init() function instead of the form above.]
*/

// Call counters are typically refered to (for external callers) by their Id, which is a string.
// (This string should contain no whitespace due to limitations of Go's benchmarking framework).
// Call counters are organized in a tree-like structure for displaying and grouping:
// e.g.
// ExpensiveOperations (root): count = 5
// 	- SquareRoot (child of ExpensiveOperations) count = 3
//  - CubeRoot (dito)  count = 2
// in the example above
//
// There are marked "display root" counters that get displayed by default along with all of their recursive children etc.
// Note that the graph structure related to displaying actually does not have to be a tree (but it is by default):
// A node can have multiple parents (if you want something displayed multiple times).
// The "display as child"-graph just needs to be a directed acyclic graph.
// Furthermore, the set of display roots does not need to coincide with the set of roots of this graph.
// Roots of the graph not set as display roots just won't get printed by default.
// Setting intermediate nodes as (additional) display roots will (additinally) print out subtrees.

// In addition to the display-as-child relationship, some adding to some counters automatically also (recursively or not) add to or subtracts from
// other counters. By default, every child counter recursively adds to all its parents, so the adds-to and the display-as-child
// graphs are identical, but this need not be the case. Of course, the recursively adds-to graph also needs to be a directed acyclic graph.
// The adds-to relationship comes with a weight; setting this to -1 will cause subtraction.
// The general use-case is the (fairly common) situation
// ExpensiveOps
// -- ExpensiveOp1
// -- ExpensiveOp2
// where ExpensiveOp1 and ExpensiveOp2 are both exported and we want to count how often they are externally called
// internally, however, ExpensiveOp1 calls ExpensiveOp2 and we want to correct for those dependencies, and display e.g.
// ExpensiveOps
// -- ExpensiveOp1
// -- ExpensiveOp2
// ---- ExpensiveOp2CalledFromExpensiveOp1
// ---- ExpensiveOp2CalledDirectly
// and have the parent ExpensiveOps just count the Sum ExpensiveOp1 and ExpensiveOp2CalledDirectly

// In the external interface, call counters are always refered to by the id string.
// A CallCounter with a given id needs to be created and registered exactly once by one the provided function before it can be used.
// We assume that all call counters are created before any increments happen.
// Call counters can be refered to by id and be created later, so initialization order does not matter.
// If a call counter is refered, but never actually created, a dummy entry with sensible defaults is used.
// (This is not recommended). Note that CallCounterExists returns false for those dummies.

type Id string

type CallCounter struct {
	id               Id                   // string id used to external refer to this CallCounter. Never ""
	display          bool                 // Should we ever display this CallCounter and its children.
	displayName      string               // Display name of this CallCounter. Defaults to id if set to the empty string.
	subcounters      []*CallCounter       // children of the display-as-tree relationship
	addTo            map[*CallCounter]int // adding to the call counter will also non-recursively add to other call counter cc with multiplicity addTo[cc]
	addToRecursive   map[*CallCounter]int // adding to the call counter will also recursively add to other call counter cc with multiplicity addTo[cc]
	count_direct     int                  // Incrementing the counter just increments this without considering addTo or addToRecursive
	count_modified   int                  // This variable holds the counters when adjusted for addTo and addToRecursive. We adjust when reading from the counters, not when writing to them.
	initialized      bool                 // Was this counter initialized? We need to sometimes create dummy counters to make the user not need to care about the order of creating counters.
	displayRootNode  bool                 // Should this counter and its children be the starting point of displaying.
	displayRemaining *CallCounter         // If non-nil we automatically add an child that displays the difference between this node and the other children. This child is not contained in the callCounters map.
}

// TODO: Protect by Mutex?
var callCounters map[Id]*CallCounter = make(map[Id]*CallCounter)

// This function checks whether a call counter with the given id exists and was initialized.
func (id Id) Exists() (ret bool) {
	if id == "" {
		return false
	}
	var cc *CallCounter
	cc, ret = callCounters[id]
	if ret {
		ret = cc.initialized
	}
	return
}

// Note: The external interface refers to individual CallCounters only by Id strings.
// Internally, we cross-link them using pointers instead of strings (for efficiency reasons)
// and keep a map callCounters that map Id strings -> pointers for existing CallCounters.
// As users may register CallCounters in an non-dependency-compatible order,
// we just add a dummy entry into the map in order to be able to take an adress.
// This dummy is the later replaced (while keeping the adress) when the user request actual creation later.

// addDummyCounter registers an (marked as uninitialized) dummy entry
// into the callCounters map and returns a pointer to it.
func addDummyCounter(id Id) *CallCounter {
	if id == "" {
		panic("callCounters: Trying to create a dummy call counter with empty id")
	}
	cc, exists := callCounters[id]
	if exists {
		if cc.initialized {
			panic("callCounters: Trying to add overwrite existing call counter with a dummy")
		} else {
			// We just return the existing dummy counter. Still, this case should not actually happen.
			return cc
		}
	}
	cc = &CallCounter{id: id}
	cc.addTo = make(map[*CallCounter]int)
	cc.addToRecursive = make(map[*CallCounter]int)
	cc.subcounters = make([]*CallCounter, 0) // probably not really needed, as append to nil works.
	callCounters[id] = cc
	return cc
}

// getCounter translates from id to *CallCounter.
// If id did not exist in the global table yet, a (dummy) entry is created.
func getCounter(id Id) (ret *CallCounter, already_existed bool) {
	if id == "" {
		panic("callCounters: called getCounter with empty id")
	}
	ret, already_existed = callCounters[id]
	if !already_existed {
		// We just create a dummy entry, to be properly initialized later.
		ret = addDummyCounter(id)
		if ret == nil {
			panic("callCounters: addDummyCounter gave back a nil entry")
		}
	}
	if ret == nil {
		panic("callcounters: callCounters map contained nil entry")
	}
	return
}

// CreateHierarchicalCallCounter(id, displayName, parentId) creates a new call counter with the given id and displayName and returns a pointer to it.
// If parentId is not the empty string, it sets that callCounter as its parent wrt the display-as-children relation.
// If parent did not exist yet or is never created, a dummy parent is created. displayRootNode is set by default for roots.
func CreateHierarchicalCallCounter(id Id, displayName string, parentId Id) *CallCounter {
	if id == "" {
		panic("callCounters: called AddNewCounter with empty string")
	}
	var cc *CallCounter
	var alreadyExists bool
	cc, alreadyExists = getCounter(id)
	if alreadyExists && cc.initialized {
		panic("callCounters: Added the same counter twice")
	}
	cc.display = true
	if displayName != "" {
		cc.displayName = displayName
	} else {
		cc.displayName = string(id)
	}

	cc.initialized = true
	if parentId != "" {
		cc.displayRootNode = false
		var parentcc *CallCounter
		parentcc, alreadyExists = getCounter(parentId)
		if !alreadyExists {
			// The default dummy parent should have these settings in case it is never "properly" created.
			parentcc.displayRootNode = true
			parentcc.display = true
			parentcc.displayName = string(parentId) + "(dummy)"
		}
		linkAsChildParent(cc, parentcc)
		// Note: +=1 instead of = 1 in case someone creates and modifies a dummy entry before calling AddNewCallCounter (which is a bad idea, but anyway)
		cc.addToRecursive[parentcc] += 1 // this creates the map entry if it did not exist before
		if cc.addToRecursive[parentcc] == 0 {
			delete(cc.addToRecursive, parentcc)
		}
	} else {
		cc.displayRootNode = true
	}
	return cc
}

// This function creates a parent-child relation for the subcounter relationship.
// Use this instead of adding to subcounters directly, as we need to adjusts displayRemaining.
func linkAsChildParent(child *CallCounter, parent *CallCounter) {
	parent.subcounters = append(parent.subcounters, child)
	if parent.displayRemaining != nil {
		child.addTo[parent.displayRemaining] = -1
	}
}

// CreateNewCallCounter(id, displayName, doDisplay, attachTo, displayRootNode) creates and registers a
// new call counter with the given id and parameters. It returns a pointer to the newly created call counter.
// doDisplay controls whether it (and its children) gets displayed at all. attachTo, if not "", makes attachTo a parent
// wrt the display-as-child relation, but does NOT automatically add to it. displayRootNode control whether is a root node for
// displaying purposes.
func CreateNewCallCounter(id Id, displayName string, doDisplay bool, attachTo Id, displayRootNode bool) (cc *CallCounter) {
	if id == "" {
		panic("callCounter: called CreateNewCounter with empty id string")
	}
	cc, _ = getCounter(id)
	if cc.initialized {
		panic("callcounters: trying to create a call counter for an already used id")
	}
	cc.displayName = displayName
	cc.id = id
	cc.initialized = true
	cc.displayRootNode = displayRootNode
	cc.display = doDisplay
	if attachTo != "" {
		parentcc, _ := getCounter(attachTo)
		linkAsChildParent(cc, parentcc)
	}
	return
}

// CreateAttachedCallCounter(id, displayName, attachTo) creates a and registers a new call counter with the given parameters.
// If attachTo is not the empty string, it is created as a child of attachTo, but does not add to it.
// It also makes the attached-to parent display the difference between itself and its children.
func CreateAttachedCallCounter(id Id, displayName string, attachTo Id) (cc *CallCounter) {
	cc = CreateNewCallCounter(id, displayName, true, attachTo, false)
	attachTo.SetDisplayRemaining(true)
	return
}

// SetDisplayRemaining sets (or unsets) the displayremaining flag on a call counter. If this flag is set
// and the call counter has child counters, then an additional implicit child counter is added
// for displaying purposes that counts the difference between the sum of the child nodes and the parent.
func (id Id) SetDisplayRemaining(displayremaining bool) *CallCounter {
	if id == "" {
		panic("callCounter: called SetDisplayRemaining with empty string")
	}
	cc, _ := getCounter(id)
	cc.SetDisplayRemaining(displayremaining)
	return cc
}

// SetDisplayRemaining sets (or unsets) the displayremaining flag on a call counter. If this flag is set
// and the call counter has child counters, then an additional implicit child counter is added
// for displaying purposes that counts the difference between the sum of the child nodes and the parent.
func (cc *CallCounter) SetDisplayRemaining(displayRemaining bool) *CallCounter {
	if displayRemaining {
		if cc.displayRemaining != nil {
			return cc // displayRemaining was already set, nothing to do.
		}
		var counterForRemains CallCounter = CallCounter{id: cc.id + "Other", display: true, displayName: string(cc.id) + "(other)"}
		counterForRemains.subcounters = make([]*CallCounter, 0)
		counterForRemains.initialized = true

		counterForRemains.addTo = make(map[*CallCounter]int) // addTo would not work anyway; leaving it at nil would actually be fine.
		counterForRemains.addToRecursive = make(map[*CallCounter]int)
		cc.displayRemaining = &counterForRemains
		cc.subcounters = append(cc.subcounters, &counterForRemains)
		cc.addTo[&counterForRemains] = +1
		for _, child_cc := range cc.subcounters {
			child_cc.addTo[&counterForRemains] = -1
		}
	} else { // unset displayRemaining
		if cc.displayRemaining == nil {
			return cc // displayRemaining was already unset, nothing to do.
		}
		delete(cc.addTo, cc.displayRemaining)
		for _, child_cc := range cc.subcounters {
			delete(child_cc.addTo, cc.displayRemaining)
		}
		numChilds := len(cc.subcounters)
		for i, child_cc := range cc.subcounters {
			if child_cc == cc.displayRemaining {
				cc.subcounters[i], cc.subcounters[numChilds-1] = cc.subcounters[numChilds-1], cc.subcounters[i]
				cc.subcounters[numChilds-1] = nil
				cc.subcounters = cc.subcounters[0 : numChilds-1]
				break
			} else if i == numChilds-1 {
				panic("callCounters: internal error: counter for remaining not found as subcounter")
			}
		}
		cc.displayRemaining = nil
	}
	return cc
}

// AddThisToTarget(targetId, multiplier) causes all increments to the receiver to also recursively add to targetId with multiplicity multiplier.
// If there is already such a link present, the multipliers get added together (instead of overwritten).
// It returns the receiver for chaining.
func (cc *CallCounter) AddThisToTarget(targetId Id, multiplier int) *CallCounter {
	if multiplier == 0 {
		return cc
	}
	if targetId == "" {
		panic("callCounters: called AddThisToTarget with empty target id")
	}
	target, _ := getCounter(targetId)
	cc.addToRecursive[target] += multiplier
	if cc.addToRecursive[target] == 0 {
		delete(cc.addToRecursive, target)
	}
	return cc
}

// AddToThisFromSource(sourceId, multiplier) is similar to AddThisToTarget, but called on the target of the link rather than on the source.
func (cc *CallCounter) AddToThisFromSource(sourceId Id, multiplier int) *CallCounter {
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

// Can call the above also on the Id directly:

// AddThisToTarget(targetId, multiplier) causes all increments to the receiver to also recursively add to targetId with multiplicity multiplier.
// If there is already such a link present, the multipliers get added together (instead of overwritten).
// It returns the call counter for chaining.
func (id Id) AddThisToTarget(targetId Id, multiplier int) *CallCounter {
	if id == "" {
		panic("callCounters: called AddThisToTarget on empty id")
	}
	cc, _ := getCounter(id)
	return cc.AddThisToTarget(targetId, multiplier)
}

// AddToThisFromSource(sourceId, multiplier) is similar to AddThisToTarget, but called on the target of the link rather than on the source.
func (id Id) AddToThisFromSource(sourceId Id, multiplier int) *CallCounter {
	if id == "" {
		panic("callCounters: called AddToThisFromSource on empty id")
	}
	cc, _ := getCounter(id)
	return cc.AddToThisFromSource(sourceId, multiplier)
}

// CCReport is the type used as output of Queries to read out call counters
// It is used to create report
type CCReport struct {
	Tag   string       // a string tag describing the call counter, typically the id or display string.
	CC    *CallCounter // a pointer to the actual call counter.
	Calls int          // The value the counter holds
	depth int          // depth in the subcounter-tree. Not always meaningful
}

func getCallCountersBelowNode(cc *CallCounter, onlyPositive bool, onlyDisplay bool, useDisplayName bool, addDepth int) (ret []CCReport) {
	ret = make([]CCReport, 0)
	if !cc.display && onlyDisplay {
		return
	}
	if cc.count_modified == 0 && onlyPositive {
		return
	}
	var report CCReport = CCReport{CC: cc, depth: addDepth, Calls: cc.count_modified}
	if useDisplayName {
		report.Tag = cc.displayName
	} else {
		report.Tag = string(cc.id)
	}
	ret = append(ret, report)
	for _, child_cc := range cc.subcounters {
		childreport := getCallCountersBelowNode(child_cc, onlyPositive, onlyDisplay, useDisplayName, addDepth+1)
		ret = append(ret, childreport...)
	}
	return
}

func ReportCallCounters(onlyPositive bool, useDisplayName bool) (ret []CCReport) {
	correctDependencies()
	ret = make([]CCReport, 0)
	for _, rootNode := range callCounters {
		if rootNode.displayRootNode {
			ret = append(ret, getCallCountersBelowNode(rootNode, onlyPositive, true, useDisplayName, 0)...)
		}
	}
	return
}

// Q: Should this function take an io.Writer rather than return a string?

// GetCallCounterStructureReport creates a string that can be printed to show the structure of all created call counters.
// Indent is the indent used to show the subtree-structure
func GetCallCounterStructureReport(indent string) (ret string) {
	var seen_cc map[*CallCounter]bool = make(map[*CallCounter]bool)
	var hidebelow int = -1
	for _, rootNode := range callCounters {
		if rootNode.displayRootNode {
			report := getCallCountersBelowNode(rootNode, false, false, false, 0)
			for _, item := range report {
				for i := 0; i < item.depth; i++ {
					ret += indent
				}
				ret += item.Tag
				if !item.CC.display {
					ret += " (hidden)"
					if hidebelow > item.depth {
						hidebelow = item.depth
					}
				} else if hidebelow == -1 {
					// do nothing
				} else if item.depth > hidebelow {
					ret += " (recursively hidden)"
				} else {
					hidebelow = -1
				}
				if !item.CC.initialized {
					ret += " (counter was only refered to, never actually created)"
				}
				ret += "\n"
				seen_cc[item.CC] = true
			}
		}
	}
	var anyunseen bool
	for _, node := range callCounters {
		if !seen_cc[node] {
			if !anyunseen {
				ret += "Call counters not below any display root node\n"
				anyunseen = true
			}
			ret += indent
			ret += string(node.id)
			if !node.display {
				ret += " (hidden)"
			}
			if !node.initialized {
				ret += " (counter was only refered to, never actually created)"
			}
			ret += "\n"
		}
	}
	return
}

// ResetCounters resets all callCounters to 0
func ResetAllCounters() {
	for _, cc := range callCounters {
		cc.count_direct = 0
		// count_direct of displayRemaining-counters is not touched, but there are always 0 anyway.
	}
}

// Reset resets the counter to 0
func (cc *CallCounter) Reset() {
	cc.count_direct = 0
}

// Reset resets the counter to 0
func (id Id) Reset() {
	cc, _ := getCounter(id)
	cc.Reset()
}

// correctDependencies takes care of the addTo and addToRecursiveRelation. This needs to be called before every read.
func correctDependencies() {
	// set all count_modified to 0
	for _, cc := range callCounters {
		cc.count_modified = 0
		if cc.displayRemaining != nil {
			cc.displayRemaining.count_modified = 0
		}
	}
	for _, cc := range callCounters {
		if cc.count_direct != 0 {
			cc.recursivelyModifyBy(cc.count_direct)
		}
	}
}

// recursivelyModifyBy is the recursive part of correctDependencies
func (cc *CallCounter) recursivelyModifyBy(amount int) {
	cc.count_modified += amount

	for additionTarget, multiplier := range cc.addTo {
		additionTarget.count_modified += amount * multiplier
	}
	for additionTarget, multiplier := range cc.addToRecursive {
		additionTarget.recursivelyModifyBy(amount * multiplier)
	}
}

func (cc *CallCounter) Get() (ret int, ok bool) {
	correctDependencies()
	ret = cc.count_modified
	ok = true
	return
}

func (id Id) Get() (ret int, ok bool) {
	if id == "" {
		panic("callCounters: called Get with empty string")
	}
	correctDependencies()
	cc, ok := callCounters[id]
	ret = cc.count_modified
	return
}

func (id Id) Increment() {
	if id == "" {
		panic("callCounters: Increment called with empty id")
	}
	cc := callCounters[id]
	if cc == nil {
		panic("callCounters: Trying to increment non-existent call counter")
	}
	if !cc.initialized {
		panic("callCounters: Trying to increment uninitialized dummy counter")
	}
	cc.count_direct++
}
