package marklist

import "fmt"

// AnyType is any type in Go language.
type AnyType interface{}

// MarkList the data structure for storing elements that will not
// change in size, but will be sometimes marked and unmarked.
type MarkList struct {
	list       []AnyType
	markedSize int
	totalSize  int
}

// New is the constructor function for struct type MarkList.
func New(capacity int) *MarkList {
	return &MarkList{list: make([]AnyType, capacity), markedSize: 0, totalSize: 0}
}

// MarkedLen returns the size of marked elements which are
// stored in indexes [0, m.MarkedLen()].
func (m *MarkList) MarkedLen() int {
	return m.markedSize
}

// Len is the method which returns the current length of elements.
// Unmarked elements are stored in indexes [m.MarkedLen(), m.Len()].
func (m *MarkList) Len() int {
	return m.totalSize
}

// Cap is the method for getting the capacity. One must not add
// more elements than the capacity.
func (m *MarkList) Cap() int {
	return len(m.list)
}

// ElemAtIndex is the method for getting element at the index.
func (m *MarkList) ElemAtIndex(index int) AnyType {
	if index >= m.totalSize {
		panic(fmt.Sprintf("index %d is more than the total size %d", index, m.totalSize))
	}
	return m.list[index]
}

// Add is the method for adding new elements up to the capacity.
// If capacity does not allow, it will panic.
func (m *MarkList) Add(elem AnyType) {
	if m.totalSize >= len(m.list) {
		panic(fmt.Sprintf("adding an element exceeds the capacity %d", len(m.list)))
	}
	m.list[m.totalSize] = elem
	m.totalSize++
}

// Mark is the method for marking an unmarked element. If the index
// corresponds to an already marked element, no change happens.
//
// Returns the new index.
func (m *MarkList) Mark(index int) int {
	elem := m.ElemAtIndex(index)
	if index >= m.markedSize {
		m.list[index] = m.list[m.markedSize]
		m.list[m.markedSize] = elem
		m.markedSize++
		return m.markedSize - 1
	}
	return -1
}

// Unmark is the method for unmarking a marked element. If the index
// corresponds to an already unmarked element, no change happens.
//
// Returns the new index.
func (m *MarkList) Unmark(index int) int {
	elem := m.ElemAtIndex(index)
	if index < m.markedSize {
		m.markedSize--
		m.list[index] = m.list[m.markedSize]
		m.list[m.markedSize] = elem
		return m.markedSize
	}
	return -1
}

// RemoveAt is the method for removing an element at the index.
func (m *MarkList) RemoveAt(index int) {
	if newIndex := m.Unmark(index); newIndex >= 0 {
		index = newIndex
	}
	m.list[index] = m.list[m.totalSize-1]
	m.totalSize--
}
