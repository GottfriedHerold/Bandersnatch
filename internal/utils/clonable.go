package utils

type Clonable[K any] interface {
	Clone() K
}
