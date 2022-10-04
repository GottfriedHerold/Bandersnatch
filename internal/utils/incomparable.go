package utils

// MakeIncomparable is type designed for struct-embedding. Embed this at the start of a struct (due to alignment) in a type to make it incomparable without taking up memory.
type MakeIncomparable = [0]func()
