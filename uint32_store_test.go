// Copyright 2021 The Sensible Code Company Ltd
// Author: Duncan Harris

package faststringmap_test

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"testing"

	"github.com/sensiblecodeio/faststringmap"
)

func TestFastStringToUint32Empty(t *testing.T) {
	ms := mapSliceN(map[string]uint32{"": 1, "a": 2, "foo": 3, "ß": 4}, 0)
	checkWithMapSlice(t, ms)
}

func TestFastStringToUint32BigSpan(t *testing.T) {
	ms := mapSliceN(map[string]uint32{"a!": 1, "a~": 2}, 2)
	checkWithMapSlice(t, ms)
}

func TestFastStringToUint32(t *testing.T) {
	const nStrs = 8192
	m := randomSmallStrings(nStrs, 8)
	checkWithMapSlice(t, mapSliceN(m, len(m)/2))
}

func checkWithMapSlice(t *testing.T, ms mapSlice) {
	fm := faststringmap.NewUint32Store(ms)

	for _, k := range ms.in {
		check := func(actV uint32, ok bool) {
			if !ok {
				t.Errorf("%q not present", k)
			} else if actV != ms.m[k] {
				t.Errorf("got %d want %d for %q", actV, ms.m[k], k)
			}
		}
		check(fm.LookupString(k))
		check(fm.LookupBytes([]byte(k)))
	}

	for _, k := range ms.out {
		check := func(actV uint32, ok bool) {
			if ok {
				t.Errorf("%q present when not expected, got %d", k, actV)
			}
		}
		check(fm.LookupString(k))
		check(fm.LookupBytes([]byte(k)))
	}
}

type mapSlice struct {
	m   map[string]uint32
	in  []string
	out []string
}

func mapSliceN(m map[string]uint32, n int) mapSlice {
	if n < 0 || n > len(m) {
		panic(fmt.Sprintf("n value %d out of range for map size %d", n, len(m)))
	}
	in := make([]string, 0, n)
	out := make([]string, 0, len(m)-n)
	nAdded := 0

	for k := range m {
		if nAdded < n {
			nAdded++
			in = append(in, k)
		} else {
			out = append(out, k)
		}
	}
	return mapSlice{m: m, in: in, out: out}
}

func (m mapSlice) AppendKeys(a []string) []string { return append(a, m.in...) }
func (m mapSlice) Get(s string) uint32            { return m.m[s] }

func randomSmallStrings(nStrs int, maxLen uint8) map[string]uint32 {
	m := map[string]uint32{"": 0}
	for len(m) < nStrs {
		s := randomSmallString(maxLen)
		if _, ok := m[s]; !ok {
			m[s] = uint32(len(m))
		}
	}
	return m
}

func randomSmallString(maxLen uint8) string {
	var sb strings.Builder
	n := rand.Intn(int(maxLen) + 1)
	for i := 0; i <= n; i++ {
		sb.WriteRune(rand.Int31n(94) + 33)
	}
	return sb.String()
}

func typicalCodeStrings(n int) mapSlice {
	m := make(map[string]uint32, n)
	keys := make([]string, 0, n)
	add := func(s string) {
		m[s] = uint32(len(m))
		keys = append(keys, s)
	}
	for i := 1; i < n; i++ {
		add(strconv.Itoa(i))
	}
	add("-9")
	return mapSlice{m: m, in: keys}
}

const nStrsBench = 1000

func BenchmarkUint32Store(b *testing.B) {
	m := typicalCodeStrings(nStrsBench)
	fm := faststringmap.NewUint32Store(m)
	b.ResetTimer()
	for bi := 0; bi < b.N; bi++ {
		for si, n := uint32(0), uint32(len(m.in)); si < n; si++ {
			v, ok := fm.LookupString(m.in[si])
			if !ok || v != si {
				b.Fatalf("ok=%v, value got %d want %d", ok, v, si)
			}
		}
	}
}

func BenchmarkGoStringToUint32(b *testing.B) {
	m := typicalCodeStrings(nStrsBench)
	b.ResetTimer()
	for bi := 0; bi < b.N; bi++ {
		for si, n := uint32(0), uint32(len(m.in)); si < n; si++ {
			v, ok := m.m[m.in[si]]
			if !ok || v != si {
				b.Fatalf("ok=%v, value got %d want %d", ok, v, si)
			}
		}
	}
}
