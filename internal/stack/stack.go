package stack

import "fmt"

const ErrorPrefix = "bandersnatch / stack "

// Stack is a simple generic stack, parameterized by the element type.
// Note that values put on the stack / returned are copied.
// The zero value of Stack is invalid. Use [MakeStack] or [NewStack].
//
// Values of type Stack should not be copied unless the source is no longer using it.
type Stack[T any] struct {
	s []T
}

// MakeStack creates a new empty stack
func MakeStack[T any]() Stack[T] {
	return Stack[T]{s: make([]T, 0)}
}

// NewStack creates a (pointer to a) new stack
func NewStack[T any]() *Stack[T] {
	return &Stack[T]{s: make([]T, 0)}
}

// Len returns the number of elements on the stack
func (st *Stack[T]) Len() int {
	return len(st.s)
}

// IsEmpty is used to query whether the stack is empty
func (st *Stack[T]) IsEmpty() bool {
	return len(st.s) == 0
}

// Pop removes the top element from the stack and returns it.
func (st *Stack[T]) Pop() (t T) {
	l := len(st.s)
	if l == 0 {
		panic(ErrorPrefix + "Trying to Pop from empty stack")
	}
	t = st.s[l-1]
	st.s = st.s[0 : l-1]
	return
}

// Push puts a (copy of) the given element on the stack
func (st *Stack[T]) Push(t T) {
	st.s = append(st.s, t)
}

// NOTE: This avoids 1 extra copying of *t compared to the equivalent st.Push(*t).
// (unless inlined, which is likely to happen, but Go is unpredictable here)

// PushPtr puts a (copy of) *t on the stack.
func (st *Stack[T]) PushPtr(t *T) {
	st.s = append(st.s, *t)
}

// Top returns a pointer to the top element of the stack.
// This is intended for direct manipulation of the element.
func (st *Stack[T]) Top() *T {
	l := len(st.s)
	if l == 0 {
		panic(ErrorPrefix + "Called Top on empty stack")
	}
	return &st.s[l-1]
}

// Peek returns a pointer to the pos'th element of the stack.
// pos == 0 corresponds to the top element, pos == st.Len()-1 correspond to the bottom.
func (st *Stack[T]) Peek(pos int) *T {
	l := len(st.s)
	if (pos < 0) || (pos >= l) {
		panic(fmt.Errorf(ErrorPrefix+"trying to peek into stack at invalid index %v, whereas the given stack has size %v", pos, l))
	}
	return &st.s[l-1-pos]
}
