package stack

const ErrorPrefix = "bandersnatch / stack "

type Stack[T any] struct {
	s []T
}

func MakeStack[T any]() Stack[T] {
	return Stack[T]{s: make([]T, 0)}
}

func NewStack[T any]() *Stack[T] {
	return &Stack[T]{s: make([]T, 0)}
}

func (st *Stack[T]) Len() int {
	return len(st.s)
}

func (st *Stack[T]) IsEmpty() bool {
	return len(st.s) == 0
}

func (st *Stack[T]) Pop() (t T) {
	l := len(st.s)
	if l == 0 {
		panic(ErrorPrefix + "Trying to Pop from empty stack")
	}
	t = st.s[l-1]
	st.s = st.s[0 : l-1]
	return
}

func (st *Stack[T]) Push(t T) {
	st.s = append(st.s, t)
}

func (st *Stack[T]) PushPtr(t *T) {
	st.s = append(st.s, *t)
}

func (st *Stack[T]) Top() *T {
	l := len(st.s)
	if l == 0 {
		panic(ErrorPrefix + "Called Top on empty stack")
	}
	return &st.s[l-1]
}
