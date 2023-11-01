package utils

// Identity is the identity function, i.e. Identity(x) returns (a copy of) x.
func Identity[T any](x T) T {
	return x
}

// ElementInList checks whether the given list contains the given element.
// normalizer is an optional argument of type func(T) T. If given, the comparison is made modulo normalizer,
// where we assume normalizer is idempotent (i.e. normalizer(normalizer(x)) == normalizer(x)  )
func ElementInList[T comparable](element T, list []T, normalizer ...func(T) T) bool {
	if len(normalizer) > 1 {
		panic("Can only provide 1 optional function argument for normalization")
	}
	if len(normalizer) == 1 {
		normalizerfun := normalizer[0]
		element = normalizerfun(element)
		for _, v := range list {
			if element == normalizerfun(v) {
				return true
			}
		}
	} else {
		for _, v := range list {
			if element == v {
				return true
			}
		}
	}
	return false
}

// ConcatenateListsWithoutDuplicates takes 2 lists list1 and list2 and returns a new list that contains every element from both list,
// but with duplicates removed. The normalizer argument (of type func(T) T) is optional; if given, it should be an involution and we consider duplicates
// modulo normalizer.
//
// Note: The current implementation is naive and has O(N^2) running time, where N is the length of the lists. This is fine for the use-case.
func ConcatenateListsWithoutDuplicates[T comparable](list1 []T, list2 []T, normalizer ...func(T) T) []T {
	if len(normalizer) > 1 {
		panic("Can only provide 1 optional function argument for normalization")
	}
	// Not terribly efficient. This has O(N^2), when N is the length of the input lists.
	// It's fine for our purpose, though.

	// naive implementation: Just checks for every element from list1 if it already appears; if not, append it.
	// Then repeat with list2.

	var ret []T = make([]T, 0, len(list1)+len(list2))

	if len(normalizer) == 1 {
		normalizerfun := normalizer[0]

	loop1:
		for _, val := range list1 {
			for _, alreadyIn := range ret {
				if normalizerfun(alreadyIn) == normalizerfun(val) {
					continue loop1
				}
			}
			ret = append(ret, val)
		}
	loop2:
		for _, val := range list2 {
			for _, alreadyIn := range ret {
				if normalizerfun(alreadyIn) == normalizerfun(val) {
					continue loop2
				}
			}
			ret = append(ret, val)
		}
		return ret
	} else {
		// no normalizer
	loop3:
		for _, val := range list1 {
			for _, alreadyIn := range ret {
				if alreadyIn == val {
					continue loop3
				}
			}
			ret = append(ret, val)
		}
	loop4:
		for _, val := range list2 {
			for _, alreadyIn := range ret {
				if alreadyIn == val {
					continue loop4
				}
			}
			ret = append(ret, val)
		}
		return ret

	}
}
