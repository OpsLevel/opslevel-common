package opslevel_common

// Stack
// A LIFO stack implementation that supports generics
type Stack[T any] struct {
	Push   func(T)
	Peek   func() T
	Pop    func() T
	Length func() int
}

// NewStack
// Usage:
// myStack := NewStack[string]("")
// myStack.Push("one")
// myStack.Push("two")
// if myStack.Peek() == "two" && myStack.Length() == 2 {
//   current := myStack.Pop()
// }
func NewStack[T any](defaultValue T) Stack[T] {
	slice := make([]T, 0)
	return Stack[T]{
		Push: func(i T) {
			slice = append(slice, i)
		},
		Peek: func() T {
			if len(slice) == 0 {
				return defaultValue
			}
			return slice[len(slice)-1]
		},
		Pop: func() T {
			if len(slice) == 0 {
				return defaultValue
			}
			res := slice[len(slice)-1]
			slice = slice[:len(slice)-1]
			return res
		},
		Length: func() int {
			return len(slice)
		},
	}
}
