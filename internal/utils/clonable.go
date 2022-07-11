package utils

type Clonable[K any] interface {
	Clone() K
}

func AddressOfCopy[K any, KPtr interface{ *K }](in K) KPtr {
	return &in
}
