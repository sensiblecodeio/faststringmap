// Copyright 2022 The Sensible Code Company Ltd
// Author: Duncan Harris

package faststringmap_test

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sensiblecodeio/faststringmap/v2"
)

func Example() {
	m := faststringmap.MapSource[string, int]{
		"key1": 42,
		"key2": 27644437,
		"l":    2,
	}

	fm := faststringmap.NewMap[string, int](m)

	// add an entry that is not in the fast map
	m["m"] = 4

	// sort the keys so output is the same for each test run
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// lookup every key in the fast map and print the corresponding value
	for _, k := range keys {
		v, ok := fm.LookupString(k)
		fmt.Printf("%q: %d, %v\n", k, v, ok)
	}

	// Dump out the store to aid in understanding the implementation
	fmt.Println()
	dump := fmt.Sprintf("%+v", fm)
	dump = strings.ReplaceAll(dump, "}", "}\n")
	dump = strings.ReplaceAll(dump, "[", "[\n ")
	dump = strings.ReplaceAll(dump, " ", "\t")
	fmt.Println(dump)

	// Output:
	//
	// "key1": 42, true
	// "key2": 27644437, true
	// "l": 2, true
	// "m": 0, false
	//
	// {store:[
	// 	{nextLo:1	nextLen:2	nextOffset:107	valid:false	value:0}
	// 	{nextLo:3	nextLen:1	nextOffset:101	valid:false	value:0}
	// 	{nextLo:0	nextLen:0	nextOffset:0	valid:true	value:2}
	// 	{nextLo:4	nextLen:1	nextOffset:121	valid:false	value:0}
	// 	{nextLo:5	nextLen:2	nextOffset:49	valid:false	value:0}
	// 	{nextLo:0	nextLen:0	nextOffset:0	valid:true	value:42}
	// 	{nextLo:0	nextLen:0	nextOffset:0	valid:true	value:27644437}
	// ]}
}
