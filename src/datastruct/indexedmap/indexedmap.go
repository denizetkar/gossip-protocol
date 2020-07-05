package indexedmap

import "fmt"

// AnyKeyType is a placeholder for any type of key in Go language.
type AnyKeyType interface{}

// AnyValueType is a placeholder for any type of value in Go language.
type AnyValueType interface{}

// ValueAndIndex is exactly what is sounds like.
type ValueAndIndex struct {
	value AnyValueType
	index int
}

// IndexedMap is a convenience type for maps whose keys are indexable.
// ALWAYS USE THE CONSTRUCTOR FOR A NEW INDEXED MAP!
type IndexedMap struct {
	m       map[AnyKeyType]ValueAndIndex
	keyList []AnyKeyType
}

// New is the constructor function for type IndexedMap.
func New() *IndexedMap {
	return &IndexedMap{m: map[AnyKeyType]ValueAndIndex{}}
}

// Put is the function for putting a key-value pair into the map.
func (indexedMap *IndexedMap) Put(key AnyKeyType, value AnyValueType) {
	valueAndIndex, isMember := indexedMap.m[key]
	if !isMember {
		indexedMap.m[key] = ValueAndIndex{value: value, index: len(indexedMap.keyList)}
		indexedMap.keyList = append(indexedMap.keyList, key)
	} else {
		indexedMap.m[key] = ValueAndIndex{value: value, index: valueAndIndex.index}
	}
}

// Remove is the function for removing keys from the map.
func (indexedMap *IndexedMap) Remove(key AnyKeyType) {
	if valueAndIndex, isMember := indexedMap.m[key]; isMember {
		lastIndex := len(indexedMap.keyList) - 1
		lastKey := indexedMap.keyList[lastIndex]
		indexedMap.keyList[valueAndIndex.index] = lastKey
		indexedMap.keyList = indexedMap.keyList[:lastIndex]
		// Change the index inside the last ValueAndIndex for correctness
		lastValueAndIndex := indexedMap.m[lastKey]
		lastValueAndIndex.index = valueAndIndex.index
		indexedMap.m[lastKey] = lastValueAndIndex
		delete(indexedMap.m, key)
	}
}

// IsMember is the function for checking if the key is in the map.
func (indexedMap *IndexedMap) IsMember(key AnyKeyType) bool {
	_, isMember := indexedMap.m[key]
	return isMember
}

// KeyAtIndex is the function for getting the key at the index
// as stored in 'indexedMap.keyList'.
func (indexedMap *IndexedMap) KeyAtIndex(index int) (AnyKeyType, error) {
	if index < 0 || index >= len(indexedMap.keyList) {
		return nil, fmt.Errorf("key index out of bounds for the map: %d", index)
	}

	return indexedMap.keyList[index], nil
}

// Len is the function to get the number of keys in the map.
func (indexedMap *IndexedMap) Len() int {
	return len(indexedMap.keyList)
}

// KeyIndex is the function to get the index of a key as stored
// in 'indexedMap.keyList'. If key is not found, it returns -1.
func (indexedMap *IndexedMap) KeyIndex(key AnyKeyType) int {
	if valueAndIndex, isMember := indexedMap.m[key]; isMember {
		return valueAndIndex.index
	}
	return -1
}

// GetValue is the function to get the value of the key.
func (indexedMap *IndexedMap) GetValue(key AnyKeyType) AnyValueType {
	valueAndIndex, ok := indexedMap.m[key]
	if ok {
		return valueAndIndex.value
	}
	return nil
}
