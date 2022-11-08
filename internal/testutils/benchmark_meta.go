package testutils

import (
	"math/rand"
	"sync"
)

type PrecomputedCacheEntry[MapKey comparable, ElementType any] struct {
	rng      *rand.Rand
	elements []ElementType
	key      MapKey
	mut      sync.RWMutex
}

type PrecomputedCache[MapKey comparable, ElementType any] struct {
	mut                sync.RWMutex
	data               map[MapKey]*PrecomputedCacheEntry[MapKey, ElementType]
	createRandFromSeed func(key MapKey) *rand.Rand
	copyFun            func(ElementType) ElementType
	creationFun        func(*rand.Rand, MapKey) ElementType
}

func (pc *PrecomputedCache[MapKey, ElementType]) GetElements(key MapKey, amount int) (ret []ElementType) {
	if amount == 0 {
		return make([]ElementType, 0)
	}
	// writing down the types on purpose
	var entry *PrecomputedCacheEntry[MapKey, ElementType]
	var ok bool

	// Read existing entry from cache
	pc.mut.RLock()
	entry, ok = pc.data[key]
	pc.mut.RUnlock()

	// If the key was not in the map, we need to create it.
	if !ok {
		// create new entry
		entry = pc.newCacheEntry(key)

		// try to write it to the map:
		pc.mut.Lock()
		altentry, ok := pc.data[key] // check again if data was present
		if ok {
			entry = altentry // if some other goroutine alread filled the entry, discard our work and use that. The first goroutine to write "wins".
		} else {
			pc.data[key] = entry
		}
		pc.mut.Unlock()
	}
	// in any case, we now have a valid entry. Get ret from it.

	return entry.getElements(amount, pc)
}

func (pc *PrecomputedCache[MapKey, ElementType]) newCacheEntry(key MapKey) (ret *PrecomputedCacheEntry[MapKey, ElementType]) {
	ret = new(PrecomputedCacheEntry[MapKey, ElementType])
	ret.key = key
	ret.elements = make([]ElementType, 0)
	ret.rng = pc.createRandFromSeed(key)
	return
}

func (pc *PrecomputedCacheEntry[MapKey, ElementType]) getElements(
	amount int, source *PrecomputedCache[MapKey, ElementType]) (ret []ElementType) {
	ret = make([]ElementType, amount)
	if amount == 0 {
		// unreachable from outside package, already caught by caller.
		return
	}

	copyFun := source.copyFun
	if copyFun == nil {
		copyFun = func(in ElementType) ElementType {
			return in
		}
	}

	pc.mut.RLock()
	currentLen := len(pc.elements)

	if currentLen >= amount {
		for i := 0; i < amount; i++ {
			ret[i] = copyFun(pc.elements[i])
		}
		pc.mut.RUnlock()
		return
	}

	// We need to create more entries
	pc.mut.RUnlock()
	// sync.RWMutex cannot atomically upgrade read-locks to rw-locks.
	creationFun := source.creationFun
	pc.mut.Lock()
	currentLen = len(pc.elements) // reload due to the above non-atomicity
	for currentLen < amount {
		pc.elements = append(pc.elements, creationFun(pc.rng, pc.key))
	}
	pc.mut.Unlock()
	pc.mut.RLock()
	for i := 0; i < amount; i++ {
		ret[i] = copyFun(pc.elements[i])
	}
	pc.mut.RUnlock()
	return
}

func (pc *PrecomputedCache[MapKey, ElementType]) Validate() {
	pc.mut.RLock()
	if pc.createRandFromSeed == nil {
		panic("Function to create rng seed uninitialized")
	}
	if pc.creationFun == nil {
		panic("Function to create elements unintialized")
	}
	pc.mut.RUnlock()
}
