package set

import "fmt"

// AnyType is a placeholder for any type in Go language.
type AnyType interface{}

// Set is a convenience type for sets.
// ALWAYS USE THE CONSTRUCTOR FOR A NEW SET!
type Set struct {
	s        map[AnyType]int
	elemList []AnyType
}

// New is the constructor function for type Set.
func New() *Set {
	return &Set{s: map[AnyType]int{}}
}

// Add is the function for adding elements into the set.
func (set *Set) Add(elem AnyType) {
	if _, isMember := set.s[elem]; !isMember {
		set.s[elem] = len(set.elemList)
		set.elemList = append(set.elemList, elem)
	}
}

// Remove is the function for removing elements from the set.
func (set *Set) Remove(elem AnyType) {
	if i, isMember := set.s[elem]; isMember {
		lastIndex := len(set.elemList) - 1
		lastElem := set.elemList[lastIndex]
		set.elemList[i] = lastElem
		set.elemList = set.elemList[:lastIndex]
		set.s[lastElem] = i
		delete(set.s, elem)
	}
}

// IsMember is the function for checking if the element is in the set.
func (set *Set) IsMember(elem AnyType) bool {
	_, isMember := set.s[elem]
	return isMember
}

// ElemAtIndex is the function for getting the element at the index
// as stored in 'set.elemList'.
func (set *Set) ElemAtIndex(index int) (AnyType, error) {
	if index < 0 || index >= len(set.elemList) {
		return nil, fmt.Errorf("element index out of bounds for the set: %d", index)
	}

	return set.elemList[index], nil
}

// Len is the function to get the number of elements in the set.
func (set *Set) Len() int {
	return len(set.elemList)
}

// ElemIndex is the function to get the index of an element as
// stored in 'set.elemList'. If element is not found, it returns -1.
func (set *Set) ElemIndex(elem AnyType) int {
	if i, isMember := set.s[elem]; isMember {
		return i
	}
	return -1
}
