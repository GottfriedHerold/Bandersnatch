package testutils

import (
	"fmt"
	"math/rand"
	"sync"
)

type PrecomputedCacheEntry[MapKey comparable, ElementType any] struct {
	rng        *rand.Rand
	elements   []ElementType
	key        MapKey
	entryMutex sync.RWMutex
}

type PrecomputedCache[MapKey comparable, ElementType any] struct {
	tableMutex         sync.RWMutex
	data               map[MapKey]*PrecomputedCacheEntry[MapKey, ElementType]
	createRandFromSeed func(key MapKey) *rand.Rand
	creationFun        func(*rand.Rand, MapKey) ElementType
	copyFun            func(ElementType) ElementType
}

func MakePrecomputedCache[MapKey comparable, ElementType any](createRandFromSeed func(MapKey) *rand.Rand, creationFun func(*rand.Rand, MapKey) ElementType, copyFun func(ElementType) ElementType) (ret PrecomputedCache[MapKey, ElementType]) {
	ret.data = make(map[MapKey]*PrecomputedCacheEntry[MapKey, ElementType])

	// If createRandFromSeed is nil, we populate it with a valid function that just returns nil.
	// USING the created *rand.Rand will likely fail.
	// This is needed to make prepopulated PrecomputedCache's work, where the output rng seed is likely never used anyway.
	if createRandFromSeed == nil {
		ret.createRandFromSeed = func(MapKey) *rand.Rand { return nil }
	} else {
		ret.createRandFromSeed = createRandFromSeed
	}

	// creationFun may be nil. In this case, automatic extension of the precomputed caches will fail and only prepopulated one are meaningful.
	ret.creationFun = creationFun

	// nil copyFuns get replaced by a trivial one that just naively copies the element.
	// The only reason why we even need copyFun is when ElementType is, say, a pointer (possibly in an interface)
	if copyFun == nil {
		ret.copyFun = func(in ElementType) ElementType { return in }
	} else {
		ret.copyFun = copyFun
	}
	return
}

func (pc *PrecomputedCache[MapKey, ElementType]) PrePopulateCache(key MapKey, cacheEntries []ElementType) {
	copyFun := pc.copyFun
	pc.tableMutex.Lock()
	page, ok := pc.data[key]
	if ok {
		pc.tableMutex.Unlock()
		panic(fmt.Errorf("trying to populate cache under key %v, which already exists", key))
	}
	Assert(page == nil)
	page = pc.newCacheEntry(key)
	pc.data[key] = page
	page.entryMutex.Lock() // cannot block
	pc.tableMutex.Unlock()
	for _, entry := range cacheEntries {
		page.elements = append(page.elements, copyFun(entry))
	}
	page.entryMutex.Unlock()

}

func (pc *PrecomputedCache[MapKey, ElementType]) GetElements(key MapKey, amount int) (ret []ElementType) {
	if amount == 0 {
		return make([]ElementType, 0)
	}
	// writing down the types on purpose
	var page *PrecomputedCacheEntry[MapKey, ElementType]
	var ok bool

	// Read existing entry from cache
	pc.tableMutex.RLock()
	page, ok = pc.data[key]
	pc.tableMutex.RUnlock()

	// If the key was not in the map, we need to create it.
	if !ok {
		// create new entry
		page = pc.newCacheEntry(key)

		// try to write it to the map:
		pc.tableMutex.Lock()
		existingPage, ok := pc.data[key] // check again if data was present
		if ok {
			page = existingPage // if some other goroutine alread filled the entry, discard our work and use that. The first goroutine to write "wins".
		} else {
			pc.data[key] = page
		}
		pc.tableMutex.Unlock()
	}
	// in any case, we now have a valid entry. Get ret from it.

	return page.getElements(amount, pc)
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

	pc.entryMutex.RLock()
	currentLen := len(pc.elements)

	if currentLen >= amount {
		for i := 0; i < amount; i++ {
			ret[i] = copyFun(pc.elements[i])
		}
		pc.entryMutex.RUnlock()
		return
	}

	// We need to create more entries
	pc.entryMutex.RUnlock()
	// sync.RWMutex cannot atomically upgrade read-locks to rw-locks.
	creationFun := source.creationFun
	pc.entryMutex.Lock()
	for len(pc.elements) < amount {
		pc.elements = append(pc.elements, creationFun(pc.rng, pc.key))
	}
	pc.entryMutex.Unlock()
	pc.entryMutex.RLock()
	for i := 0; i < amount; i++ {
		ret[i] = copyFun(pc.elements[i])
	}
	pc.entryMutex.RUnlock()
	return
}
