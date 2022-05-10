// Copyright 2022 The Sensible Code Company Ltd
// Author: Duncan Harris

package faststringmap

import (
	"sort"
)

type (
	// Map is a fast read only map from a string type to T.
	// Lookups are about 5x faster than the built-in Go map type.
	// A Map instance can also be directly persisted to disk.
	Map[_ ~string, T any] struct {
		store []byteValue[T]
	}

	byteValue[T any] struct {
		nextLo     uint32 // index in store of next byteValues
		nextLen    byte   // number of byteValues in store used for next possible bytes
		nextOffset byte   // offset from zero byte value of first element of range of byteValues
		valid      bool   // is the byte sequence with no more bytes in the map?
		value      T      // value for byte sequence with no more bytes
	}

	// MapFaster is a faster read only map from a string type to T.
	// Unlike Map it can't be directly persisted to disk.
	MapFaster[_ ~string, T any] struct {
		store []byteValueSlice[T]
	}

	byteValueSlice[T any] struct {
		next       []byteValueSlice[T]
		nextOffset byte // offset from zero byte value of first element of next
		valid      bool // is the byte sequence with no more bytes in the map?
		value      T    // value for byte sequence with no more bytes
	}

	// builder is used only during construction
	builder[K ~string, T any] struct {
		all [][]byteValue[T]
		src Source[K, T]
		len int
	}

	// Source is for supplying data to initialise Map
	Source[K ~string, T any] interface {
		// AppendKeys should append the keys of the map to the supplied slice and return the resulting slice
		AppendKeys([]K) []K
		// Get should return the value for the supplied key
		Get(K) T
	}

	// MapSource is an adaptor from a Go map to a Source
	MapSource[K ~string, T any] map[K]T
)

func (m MapSource[K, _]) AppendKeys(a []K) []K {
	if cap(a)-len(a) < len(m) {
		a = append(make([]K, 0, len(a)+len(m)), a...)
	}
	for k := range m {
		a = append(a, k)
	}
	return a
}

func (m MapSource[K, T]) Get(s K) T { return m[s] }

// NewMap creates a map which can be persisted to disk easily but
// is slightly slower than the "faster" version owing to an unavoidable bounds check.
func NewMap[K ~string, T any](srcMap Source[K, T]) Map[K, T] {
	if keys := srcMap.AppendKeys([]K(nil)); len(keys) > 0 {
		sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
		return Map[K, T]{store: build[K, T](keys, srcMap)}
	}
	return Map[K, T]{store: []byteValue[T]{{}}}
}

// build constructs the map by allocating memory in blocks
// and then copying into the eventual slice at the end.
// This is more efficient than continually using append.
func build[K ~string, T any](keys []K, src Source[K, T]) []byteValue[T] {
	b := builder[K, T]{
		all: [][]byteValue[T]{make([]byteValue[T], 1, firstBufSize(len(keys)))},
		src: src,
		len: 1,
	}
	b.makeByteValue(&b.all[0][0], keys, 0)
	// copy all blocks to one slice
	s := make([]byteValue[T], 0, b.len)
	for _, a := range b.all {
		s = append(s, a...)
	}
	return s
}

// makeByteValue will initialise the supplied byteValue for
// the sorted strings in slice a considering bytes at byteIndex in the strings
func (b *builder[K, T]) makeByteValue(bv *byteValue[T], a []K, byteIndex int) {
	// if there is a string with no-more bytes then it is always first because they are sorted
	if len(a[0]) == byteIndex {
		bv.valid = true
		bv.value = b.src.Get(a[0])
		a = a[1:]
	}
	if len(a) == 0 {
		return
	}
	bv.nextOffset = a[0][byteIndex]       // lowest value for next byte
	bv.nextLen = a[len(a)-1][byteIndex] - // highest value for next byte
		bv.nextOffset + 1 // minus lowest value +1 = number of possible next bytes
	bv.nextLo = uint32(b.len)   // first byteValue struct in eventual built slice
	next := b.alloc(bv.nextLen) // new byteValues default to "not valid"

	for i, n := 0, len(a); i < n; {
		// find range of strings starting with the same byte
		iSameByteHi := i + 1
		for iSameByteHi < n && a[iSameByteHi][byteIndex] == a[i][byteIndex] {
			iSameByteHi++
		}
		b.makeByteValue(&next[(a[i][byteIndex]-bv.nextOffset)], a[i:iSameByteHi], byteIndex+1)
		i = iSameByteHi
	}
}

const maxBuildBufSize = 1 << 20

func firstBufSize(mapSize int) int {
	size := 1 << 4
	for size < mapSize && size < maxBuildBufSize {
		size <<= 1
	}
	return size
}

// alloc will grab space in the current block if available or allocate a new one if not
func (b *builder[_, T]) alloc(nByteValues byte) []byteValue[T] {
	n := int(nByteValues)
	b.len += n
	cur := &b.all[len(b.all)-1] // current
	curCap, curLen := cap(*cur), len(*cur)
	if curCap-curLen >= n { // enough space in current
		*cur = (*cur)[: curLen+n : curCap]
		return (*cur)[curLen:]
	}
	newCap := curCap * 2
	for newCap < n {
		newCap *= 2
	}
	if newCap > maxBuildBufSize {
		newCap = maxBuildBufSize
	}
	a := make([]byteValue[T], n, newCap)
	b.all = append(b.all, a)
	return a
}

// NewMapFaster creates a map which is faster than Map
// but can't be directly persisted to disk
func NewMapFaster[K ~string, T any](srcMap Map[K, T]) MapFaster[K, T] {
	m := MapFaster[K, T]{store: make([]byteValueSlice[T], len(srcMap.store))}
	for i := range srcMap.store {
		v, sv := &m.store[i], &srcMap.store[i]
		v.nextOffset = sv.nextOffset
		v.valid = sv.valid
		v.value = sv.value
		v.next = m.store[sv.nextLo : sv.nextLo+uint32(sv.nextLen)]
	}
	return m
}

// LookupString looks up the supplied string in the map
func (m Map[K, T]) LookupString(s K) (T, bool) {
	bv := &m.store[0]
	for i, n := 0, len(s); i < n; i++ {
		b := s[i]
		if b < bv.nextOffset {
			var r T
			return r, false
		}
		ni := b - bv.nextOffset
		if ni >= bv.nextLen {
			var r T
			return r, false
		}
		bv = &m.store[bv.nextLo+uint32(ni)]
	}
	return bv.value, bv.valid
}

func (m Map[_, _]) Empty() bool {
	return len(m.store) == 1 && !m.store[0].valid
}

// AppendSortedKeys appends the keys in the map to the supplied slice in sorted order
func (m Map[K, _]) AppendSortedKeys(a []K) []K {
	buf := make([]byte, 0, 256) // initially allocate for reasonable max key length, but this is not a maximum
	m.appendKeysFrom(0, &buf, &a)
	return a
}

func (m Map[K, _]) appendKeysFrom(storeIndex uint32, prefix *[]byte, a *[]K) {
	bv := &m.store[storeIndex]
	if bv.valid {
		*a = append(*a, K(*prefix))
	}
	for i := byte(0); i < bv.nextLen; i++ {
		*prefix = append(*prefix, bv.nextOffset+i)
		m.appendKeysFrom(bv.nextLo+uint32(i), prefix, a)
		*prefix = (*prefix)[:len(*prefix)-1]
	}
}

// LookupBytes looks up the supplied byte slice in the map
func (m Map[_, T]) LookupBytes(s []byte) (T, bool) {
	bv := &m.store[0]
	for i, n := 0, len(s); i < n; i++ {
		b := s[i]
		if b < bv.nextOffset {
			var r T
			return r, false
		}
		ni := b - bv.nextOffset
		if ni >= bv.nextLen {
			var r T
			return r, false
		}
		bv = &m.store[bv.nextLo+uint32(ni)]
	}
	return bv.value, bv.valid
}

// LookupString looks up the supplied string in the map
func (m MapFaster[_, T]) LookupString(s string) (T, bool) {
	bv := &m.store[0]
	for i, n := 0, len(s); i < n; i++ {
		b := s[i]
		if b < bv.nextOffset {
			var r T
			return r, false
		}
		// careful to avoid bounds check
		ni := int(b - bv.nextOffset)
		if ni >= len(bv.next) {
			var r T
			return r, false
		}
		bv = &bv.next[ni]
	}
	return bv.value, bv.valid
}

// LookupBytes looks up the supplied byte slice in the map
func (m MapFaster[_, T]) LookupBytes(s []byte) (T, bool) {
	bv := &m.store[0]
	for i, n := 0, len(s); i < n; i++ {
		b := s[i]
		if b < bv.nextOffset {
			var r T
			return r, false
		}
		// careful to avoid bounds check
		ni := int(b - bv.nextOffset)
		if ni >= len(bv.next) {
			var r T
			return r, false
		}
		bv = &bv.next[ni]
	}
	return bv.value, bv.valid
}
