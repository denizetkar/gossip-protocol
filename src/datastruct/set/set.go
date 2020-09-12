// Package set contains an implementation of a set data structure.
package set

// AnyType is a placeholder for any type in Go language.
type AnyType interface{}

type void struct{}

// Set is a convenience type for sets.
// ALWAYS USE THE CONSTRUCTOR FOR A NEW Set!
type Set map[AnyType]void

// New is the constructor function for type IndexedSet.
func New() Set {
	return map[AnyType]void{}
}

// Add is the function for adding elements into the set.
func (set Set) Add(elem AnyType) Set {
	set[elem] = void{}
	return set
}

// Remove is the function for removing elements from the set.
func (set Set) Remove(elem AnyType) Set {
	delete(set, elem)
	return set
}

// IsMember is the function for checking if the element is in the set.
func (set Set) IsMember(elem AnyType) bool {
	_, isMember := set[elem]
	return isMember
}

// Len is the function to get the number of elements in the set.
func (set Set) Len() int {
	return len(set)
}

// Iterate is the method for iterating over the set:
//
// for elem := range set.Iterate() ...
func (set Set) Iterate() Set {
	return set
}
