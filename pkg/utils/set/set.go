package set

// Set implements a unique, unordered set of homogeneous element types.
type Set[T comparable] map[T]struct{}

// FromSlice creates a Set from a slice of values.
func FromSlice[T comparable](values []T) Set[T] {
	set := make(Set[T], len(values))
	set.Add(values...)
	return set
}

// Add inserts a value to the set, without retaining insertion order.
func (s Set[T]) Add(values ...T) {
	for _, value := range values {
		s[value] = struct{}{}
	}
}

// Delete removes all matching elements from the set.
// return bool whether the element is found and deleted.
func (s Set[T]) Delete(value T) bool {
	if _, ok := s[value]; ok {
		delete(s, value)
		return true
	}
	return false
}

// Has whether the value exists in the Set.
func (s Set[T]) Has(value T) bool {
	_, ok := s[value]
	return ok
}

// Len returns the number of items in the Set.
func (s Set[T]) Len() int {
	return len(s)
}

// Foreach give a iteration callback to access each element in the Set.
func (s Set[T]) Foreach(it func(T)) {
	for value := range s {
		it(value)
	}
}

// Copy returns a copy of the Set.
func (s Set[T]) Copy() Set[T] {
	set := make(Set[T])
	for value := range s {
		set[value] = struct{}{}
	}
	return set
}

// All return then slice of Set.
func (s Set[T]) All() []T {
	result := make([]T, len(s))
	i := 0
	for value := range s {
		result[i] = value
		i++
	}
	return result
}
