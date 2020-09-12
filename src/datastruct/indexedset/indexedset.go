// Package indexedset contains an implementation of a set that is
// indexable with O(1) complexity. Indexing happens on the elements
// of the set and the order of addition is not necessarily preserved
// in the element list.
package indexedset

// AnyType is a placeholder for any type in Go language.
type AnyType interface{}

// IndexedSet is a convenience type for sets that are indexed.
// ALWAYS USE THE CONSTRUCTOR FOR A NEW IndexedSet!
type IndexedSet struct {
	s        map[AnyType]int
	elemList []AnyType
}

// New is the constructor function for type IndexedSet.
func New() *IndexedSet {
	return &IndexedSet{s: map[AnyType]int{}}
}

// Add is the function for adding elements into the indexed set.
func (indexedSet *IndexedSet) Add(elem AnyType) *IndexedSet {
	if _, isMember := indexedSet.s[elem]; !isMember {
		indexedSet.s[elem] = len(indexedSet.elemList)
		indexedSet.elemList = append(indexedSet.elemList, elem)
	}
	return indexedSet
}

// Remove is the function for removing elements from the indexed set.
func (indexedSet *IndexedSet) Remove(elem AnyType) *IndexedSet {
	if i, isMember := indexedSet.s[elem]; isMember {
		lastIndex := len(indexedSet.elemList) - 1
		lastElem := indexedSet.elemList[lastIndex]
		indexedSet.elemList[i] = lastElem
		indexedSet.elemList = indexedSet.elemList[:lastIndex]
		indexedSet.s[lastElem] = i
		delete(indexedSet.s, elem)
	}
	return indexedSet
}

// IsMember is the function for checking if the element is in the indexed set.
func (indexedSet *IndexedSet) IsMember(elem AnyType) bool {
	_, isMember := indexedSet.s[elem]
	return isMember
}

// ElemAtIndex is the function for getting the element at the index
// as stored in 'set.elemList'.
func (indexedSet *IndexedSet) ElemAtIndex(index int) AnyType {
	return indexedSet.elemList[index]
}

// Len is the function to get the number of elements in the indexed set.
func (indexedSet *IndexedSet) Len() int {
	return len(indexedSet.elemList)
}

// ElemIndex is the function to get the index of an element as
// stored in 'set.elemList'. If element is not found, it returns -1.
func (indexedSet *IndexedSet) ElemIndex(elem AnyType) int {
	if i, isMember := indexedSet.s[elem]; isMember {
		return i
	}
	return -1
}

// Iterate is the method for iterating over the indexed set:
//
// for elem, i := range set.Iterate() {} OR
// for elem := range set.Iterate() {}
func (indexedSet *IndexedSet) Iterate() map[AnyType]int {
	return indexedSet.s
}
