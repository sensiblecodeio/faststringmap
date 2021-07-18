// Copyright 2021 The Sensible Code Company Ltd
// Author: Duncan Harris

package faststringmap

import (
	"sort"
)

type (
	// Uint32Store is a fast read only map from string to uint32
	// Lookups are about 5x faster than the built-in Go map type
	Uint32Store struct {
		store []byteValue
	}

	byteValue struct {
		nextLo     uint32 // index in store of next byteValues
		nextLen    byte   // number of byteValues in store used for next possible bytes
		nextOffset byte   // offset from zero byte value of first element of range of byteValues
		valid      bool   // is the byte sequence with no more bytes in the map?
		value      uint32 // value for byte sequence with no more bytes
	}

	// Uint32Source is for supplying data to initialise Uint32Store
	Uint32Source interface {
		// AppendKeys should append the keys of the maps to the supplied slice and return the resulting slice
		AppendKeys([]string) []string
		// Get should return the value for the supplied key
		Get(string) uint32
	}
)

// NewUint32Store creates from the data supplied in srcMap
func NewUint32Store(srcMap Uint32Source) Uint32Store {
	m := Uint32Store{store: make([]byteValue, 1)}
	if keys := srcMap.AppendKeys([]string(nil)); len(keys) > 0 {
		sort.Strings(keys)
		m.makeByteValue(&m.store[0], keys, 0, srcMap)
	}
	return m
}

// makeByteValue will initialise the supplied byteValue for
// the sorted strings in slice a considering bytes at byteIndex in the strings
func (m *Uint32Store) makeByteValue(bv *byteValue, a []string, byteIndex int, srcMap Uint32Source) {
	// if there is a string with no more bytes then it is always first because they are sorted
	if len(a[0]) == byteIndex {
		bv.valid = true
		bv.value = srcMap.Get(a[0])
		a = a[1:]
	}
	if len(a) == 0 {
		return
	}
	bv.nextOffset = a[0][byteIndex]       // lowest value for next byte
	bv.nextLen = a[len(a)-1][byteIndex] - // highest value for next byte
		bv.nextOffset + 1 // minus lowest value +1 = number of possible next bytes
	bv.nextLo = uint32(len(m.store)) // first byteValue struct to use

	// allocate enough byteValue structs - they default to "not valid"
	m.store = append(m.store, make([]byteValue, bv.nextLen)...)

	for i, n := 0, len(a); i < n; {
		// find range of strings starting with the same byte
		iSameByteHi := i + 1
		for iSameByteHi < n && a[iSameByteHi][byteIndex] == a[i][byteIndex] {
			iSameByteHi++
		}
		nextStoreIndex := bv.nextLo + uint32(a[i][byteIndex]-bv.nextOffset)
		m.makeByteValue(&m.store[nextStoreIndex], a[i:iSameByteHi], byteIndex+1, srcMap)
		i = iSameByteHi
	}
}

// LookupString looks up the supplied string in the map
func (m *Uint32Store) LookupString(s string) (uint32, bool) {
	bv := &m.store[0]
	for i, n := 0, len(s); i < n; i++ {
		b := s[i]
		if b < bv.nextOffset {
			return 0, false
		}
		ni := b - bv.nextOffset
		if ni >= bv.nextLen {
			return 0, false
		}
		bv = &m.store[bv.nextLo+uint32(ni)]
	}
	return bv.value, bv.valid
}

// LookupBytes looks up the supplied byte slice in the map
func (m *Uint32Store) LookupBytes(s []byte) (uint32, bool) {
	bv := &m.store[0]
	for _, b := range s {
		if b < bv.nextOffset {
			return 0, false
		}
		ni := b - bv.nextOffset
		if ni >= bv.nextLen {
			return 0, false
		}
		bv = &m.store[bv.nextLo+uint32(ni)]
	}
	return bv.value, bv.valid
}
