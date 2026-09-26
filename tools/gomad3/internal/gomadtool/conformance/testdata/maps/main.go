// maps prints the iteration order of one map per hashing family so the driver
// can check that the order is seeded, repeatable, and diverse.
package main

import (
	"fmt"
	"maps"
	"math"
	"os"
	"strings"

	"gomad3.test/internal/perturb"
)

type point struct {
	X, Y int
}

type label string

func (l label) String() string { return string(l) }

func main() {
	perturb.Apply(os.Args[1:])
	ok := true
	report := func(family string, iterated []string, want int) {
		if len(iterated) != want {
			ok = false
		}
		fmt.Printf("%s:%s\n", family, strings.Join(iterated, ","))
	}

	created := make(map[int]int, 16)
	for key := range 16 {
		created[key] = key * key
	}
	report("create", keysOf(created), 16)

	strs := make(map[string]int, 16)
	for key := range 16 {
		strs[fmt.Sprintf("key-%02d", key)] = key
	}
	report("string", keysOf(strs), 16)

	report("clone", keysOf(maps.Clone(created)), 16)

	reinserted := make(map[int]int, 16)
	for key := range 16 {
		reinserted[key] = key
	}
	for key := 0; key < 16; key += 2 {
		delete(reinserted, key)
	}
	for key := 0; key < 16; key += 2 {
		reinserted[key] = -key
	}
	report("delete-reinsert", keysOf(reinserted), 16)

	cleared := make(map[int]int, 16)
	for key := range 16 {
		cleared[key] = key
	}
	clear(cleared)
	for key := range 12 {
		cleared[key+100] = key
	}
	report("clear", keysOf(cleared), 12)

	report("uint8", keysOf(numbered[uint8](16)), 16)
	report("uint16", keysOf(numbered[uint16](16)), 16)
	report("uint32", keysOf(numbered[uint32](16)), 16)
	report("uint64", keysOf(numbered[uint64](16)), 16)
	report("float32", keysOf(numbered[float32](16)), 16)
	report("float64", keysOf(numbered[float64](16)), 16)

	complex64s := make(map[complex64]int, 16)
	for key := range 16 {
		complex64s[complex(float32(key), float32(-key))] = key
	}
	report("complex64", keysOf(complex64s), 16)
	complex128s := make(map[complex128]int, 16)
	for key := range 16 {
		complex128s[complex(float64(key), float64(-key))] = key
	}
	report("complex128", keysOf(complex128s), 16)

	empty := make(map[any]int, 16)
	for key := range 8 {
		empty[key] = key
		empty[fmt.Sprintf("s%d", key)] = key
	}
	report("empty-interface", keysOf(empty), 16)

	stringers := make(map[fmt.Stringer]int, 16)
	for key := range 16 {
		stringers[label(fmt.Sprintf("l%02d", key))] = key
	}
	report("non-empty-interface", keysOf(stringers), 16)

	arrays := make(map[[2]int]int, 16)
	for key := range 16 {
		arrays[[2]int{key, key % 4}] = key
	}
	report("array", keysOf(arrays), 16)

	structs := make(map[point]int, 16)
	for key := range 16 {
		structs[point{X: key, Y: 16 - key}] = key
	}
	report("struct", keysOf(structs), 16)

	grown := make(map[int]int)
	for key := range 64 {
		grown[key] = key
	}
	report("growth", keysOf(grown), 64)

	small := map[int]int{1: 1, 2: 2, 3: 3, 4: 4}
	report("small", keysOf(small), 4)

	nans := make(map[float64]int, 6)
	for value := range 6 {
		nans[math.NaN()] = value
	}
	var nanValues []string
	for _, value := range nans {
		nanValues = append(nanValues, fmt.Sprint(value))
	}
	report("nan", nanValues, 6)

	if ok {
		fmt.Println("maps-oracle:ok")
	} else {
		fmt.Println("maps-oracle:a family iterated the wrong number of keys")
	}
	perturb.Marker()
}

func numbered[K uint8 | uint16 | uint32 | uint64 | float32 | float64](count int) map[K]int {
	result := make(map[K]int, count)
	for key := range count {
		result[K(key)] = key
	}
	return result
}

func keysOf[K comparable, V any](source map[K]V) []string {
	keys := make([]string, 0, len(source))
	for key := range source {
		keys = append(keys, fmt.Sprint(key))
	}
	return keys
}
