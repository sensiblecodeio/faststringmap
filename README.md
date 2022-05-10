# faststringmap

## v2 : Latest for Go 1.18 onwards
**v2** is the latest which uses generics and runs on Go 1.18. See [v2/README.md](v2/README.md) for details.

## v1 : for Go 1.17 and earlier

`faststringmap` is a fast read-only string keyed map for Go (golang).
For our use case it is approximately 5 times faster than using Go's
built-in map type with a string key. It also has the following advantages:

* look up strings and byte slices without use of the `unsafe` package
* minimal impact on GC due to lack of pointers in the data structure
* data structure can be trivially serialized to disk or network

The code provided implements a map from string to `uint32` which fits our
use case, but you can easily substitute other value types.

`faststringmap` is a variant of a data structure called a [Trie](https://en.wikipedia.org/wiki/Trie).
At each level we use a slice to hold the next possible byte values.
This slice is of length one plus the difference between the lowest and highest
possible next bytes of strings in the map. Not all the entries in the slice are
valid next bytes. `faststringmap` is thus more space efficient for keys using a
small set of nearby runes, for example those using a lot of digits.

## Example

Example usage can be found in [``uint32_store_example_test.go``](uint32_store_example_test.go).

## Motivation

I created `faststringmap` in order to improve the speed of parsing CSV
where the fields were category codes from survey data. The majority of these
were numeric (`"1"`, `"2"`, `"3"`...) plus a distinct code for "not applicable".
I was struck that in the simplest possible cases (e.g. `"1"` ... `"5"`) the map
should be a single slice lookup.

Our fast CSV parser provides fields as byte slices into the read buffer to
avoid creating string objects. So I also wanted to facilitate key lookup from a
`[]byte` rather than a string. This is not possible using a built-in Go map without
use of the `unsafe` package.

## Benchmarks

Below are example benchmarks from my laptop which are for looking up every element
in a map of size 1000. So approximate times are 25ns per lookup for the Go native map
and 5ns per lookup for the ``faststringmap``.
```
cpu: Intel(R) Core(TM) i7-6700HQ CPU @ 2.60GHz
BenchmarkUint32Store
BenchmarkUint32Store-8        	  218463	      4959 ns/op
BenchmarkGoStringToUint32
BenchmarkGoStringToUint32-8   	   49279	     24483 ns/op
```

## Improvements

[v2](v2/README.md) features a version which has improved performance by using a slice for
the `next` fields. It is also built using generics so you can easily use any value type.