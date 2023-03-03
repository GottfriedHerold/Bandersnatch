package testutils

import (
	"fmt"
	"math/rand"
	"sync"
)

// This file defines and exports a PrecomputedCache functionality used mainly for testing.
// Precomputed caches are, in essence, (pseudorandom) lists []ElementType for some appropriate ElementType
// that our tests run against. Since we may need multiple, potentially different such lists, we actually key this and instead
// consider (at least from a user's point of view) a map[KeyType] -> []ElementType. (*)
// (KeyType and ElementType are generic parameters)
//
// We imagine the KeyType to hold an rng seed (+ potentially some tags).
// Then, we can ask a cache to get the first n elements for a given key k.
// Subsequently asking the first m elements for the same key k will give back a fresh copy of appropriate elements, where
// one list is a prefix of the other.
//
// The point here is that
//   a) creating the elements may be expensive (for curve points, creating elements freshly for each test would dominate the running time and be quite slow otherwise),
//      so we instead cache them and only extend the internal per-key cache as needed.
//   b) the whole thing is actually supposed to be thread-safe
//
// To actually populate the cache, we can either fill it with a given slice or provide some functions as arguments
// to MakePrecomputedCache that control the process.
//
// (*) We could instead just support a datastructure for a single list and have the user create a map[KeyType] -> this datastructure.
// However, then the caller would need to deal with concurrency issues when accessing/adding new keys.
// Implementing this *once* properly is better.

// precomputedCachePage is the data structure holding the cache for a single given key
type precomputedCachePage[MapType comparable, ElementType any] struct {
	rng       *rand.Rand    // rng state used to extend the elements list. This may be nil
	elements  []ElementType // actual cache of elements.
	key       MapType       // We store a copy of the map key for this page. We maintain somePrecomputedCache[someKey].key == someKey.
	pageMutex sync.RWMutex  // mutex
}

// PrecomputedCache stores, for each key of type MapType, a precomputed cache (i.e. a list) of ElementType.
// These can be cheaply retrieved.
type PrecomputedCache[MapType comparable, ElementType any] struct {
	tableMutex         sync.RWMutex                                            // mutex for the table itself
	data               map[MapType]*precomputedCachePage[MapType, ElementType] // actual cache(s) NOTE: The element type is pointer here. This is not really needed and adds extra indirection, but makes arguing about thread-safety much easier: Once an entry data[key] has been created, this entry (i.e. the pointer) never changes, even though what is pointed to may.
	createRandFromSeed func(key MapType) *rand.Rand                            // function that is used to initialize the rng from the key
	creationFun        func(*rand.Rand, MapType) ElementType                   // function that is used to extend the cache for a given key. THIS MAY BE NIL, in which case extension will fail. Note: This function is the same for all keys for simplicity. If needed, users can always include the actual function as a field of KeyType.
	copyFun            func(ElementType) ElementType                           // function used to (appropriately deep-) copy elements. We always output copies, since users should have no way to modify the cache.
}

// MakePrecomputedCache is used to create a ready-to-use [PrecomputedCache] for the given KeyType and ElementType.
//
// We expect the user to provide 3 functions to create the rng seed from the key, to create elements from the rng, and to copy elements.
// Each of these 3 functions can be nil with the following meaning:
//   - if the createRandFromSeed function is nil, creationFun will always be called with a nil argument
//   - if the creationFun function is nil, we cannot randomly sample and only prepoulated keys work.
//   - if the copyFun function is nil, we perform a trivial copy operation.
func MakePrecomputedCache[KeyType comparable, ElementType any](createRandFromSeed func(KeyType) *rand.Rand, creationFun func(*rand.Rand, KeyType) ElementType, copyFun func(ElementType) ElementType) (ret PrecomputedCache[KeyType, ElementType]) {
	ret.data = make(map[KeyType]*precomputedCachePage[KeyType, ElementType])

	// If createRandFromSeed is nil, we populate it with a valid function that just returns nil.
	// USING the created *rand.Rand will likely fail.
	// We need this, because our implementation will still call createRandFromSeed, even if we use a prepolutatedCache.
	// So we need this to be appropriately callable even if the rng seed is likely never used anyway in such a case.
	if createRandFromSeed == nil {
		ret.createRandFromSeed = func(KeyType) *rand.Rand { return nil }
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

// PrepopulateCache pre-populates (or pre-prefixes) the cache under the given key with the given list of entries.
// Retrieving elements under this key will then return (a prefix of) (copies of) this list of entries.
// If we want to retrieve more elements than len(cacheEntries), the PrecomputedCache will extend the cache automatically via its usual method.
//
// IMPORTANT:
//   - This method ONLY works if no entries have ever been retrieved under the given key and we did not prepopulate the key before. We panic otherwise.
//   - the given entries are copied into the cache into a fresh backing array.
func (pc *PrecomputedCache[KeyType, ElementType]) PrepopulateCache(key KeyType, cacheEntries []ElementType) {
	copyFun := pc.copyFun
	pc.tableMutex.Lock()
	page, ok := pc.data[key]
	if ok {
		pc.tableMutex.Unlock()
		panic(fmt.Errorf("trying to populate cache under key %v, which already exists", key))
	}
	Assert(page == nil)
	page = pc.newCachePage(key)
	pc.data[key] = page
	page.pageMutex.Lock() // cannot block
	pc.tableMutex.Unlock()
	for _, entry := range cacheEntries {
		page.elements = append(page.elements, copyFun(entry))
	}
	page.pageMutex.Unlock()

}

// GetElements returns (a copy of) the first amount many precomputed elements that are stored under the given key.
// If there are not enough or no elements stored, we extend the cache automatically.
func (pc *PrecomputedCache[KeyType, ElementType]) GetElements(key KeyType, amount int) (ret []ElementType) {
	if amount == 0 {
		return make([]ElementType, 0)
	}
	// writing down the types on purpose
	var page *precomputedCachePage[KeyType, ElementType]
	var ok bool

	// Read existing entry from cache
	pc.tableMutex.RLock()
	page, ok = pc.data[key]
	pc.tableMutex.RUnlock()

	// If the key was not in the map, we need to create it.
	if !ok {
		// create new entry
		page = pc.newCachePage(key)

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

// newCachePage is an internal function used to populate the actual map entries (pointer-to-page) stored inside PrecomputedCache.
// This is used to make sure all these entries are in a valid state.
func (pc *PrecomputedCache[KeyType, ElementType]) newCachePage(key KeyType) (ret *precomputedCachePage[KeyType, ElementType]) {
	ret = new(precomputedCachePage[KeyType, ElementType])
	ret.key = key
	ret.elements = make([]ElementType, 0)
	ret.rng = pc.createRandFromSeed(key)
	return
}

// getElement is a helper function to get the actual elements from the given cachepage.
func (pc *precomputedCachePage[KeyType, ElementType]) getElements(
	amount int, source *PrecomputedCache[KeyType, ElementType]) (ret []ElementType) {
	ret = make([]ElementType, amount)
	if amount == 0 {
		// unreachable from outside package, already caught by caller.
		return
	}

	copyFun := source.copyFun // avoid indirection through source at each access.

	// Check if the cache contains enough elements:
	pc.pageMutex.RLock()
	currentLen := len(pc.elements)

	// If the cache contains enough elements, just copy them and be done
	if currentLen >= amount {
		for i := 0; i < amount; i++ {
			ret[i] = copyFun(pc.elements[i])
		}
		pc.pageMutex.RUnlock()
		return
	}

	// Otherwise, we (may) need to create more entries
	pc.pageMutex.RUnlock()
	// sync.RWMutex cannot atomically upgrade read-locks to rw-locks. So pc.elements may get extended in the meantime. This is fine.
	creationFun := source.creationFun
	pc.pageMutex.Lock()
	// extend as appropriate
	for len(pc.elements) < amount {
		pc.elements = append(pc.elements, creationFun(pc.rng, pc.key))
	}
	pc.pageMutex.Unlock()

	// We are now guaranteed to read the desired amount
	pc.pageMutex.RLock()
	for i := 0; i < amount; i++ {
		ret[i] = copyFun(pc.elements[i])
	}
	pc.pageMutex.RUnlock()
	return
}

// DefaultCreateRandFromSeed is a default function that can be used as an argument to [MakePrecomputedCache] if the KeyType is int64.
// Note that this differs from given a nil argument to [MakePrecomputedCache].
func DefaultCreateRandFromSeed(key int64) *rand.Rand {
	return rand.New(rand.NewSource(key))
}
