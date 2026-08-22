package main

import (
	"fmt"
	"maps"
	"math"
	"os"
	"strings"

	"gomadv3.test/internal/layout"
)

type emptyInterfaceKey struct {
	index int
	label string
}

type nonEmptyInterface interface {
	index() int
}

type nonEmptyInterfaceKey int

func (key nonEmptyInterfaceKey) index() int {
	return int(key)
}

type structKey struct {
	index uint64
	label string
}

func appendMap[K comparable](output *strings.Builder, prefix string, values map[K]int, formatKey func(K) string) {
	fmt.Fprint(output, prefix, ":")
	for key, value := range values {
		fmt.Fprintf(output, "%s=%d,", formatKey(key), value)
	}
	output.WriteByte('\n')
}

func requireMap[K comparable](values map[K]int, expected map[K]int) {
	if len(values) != len(expected) {
		panic("map has an invalid cardinality")
	}
	for key, value := range expected {
		if observed, present := values[key]; !present || observed != value {
			panic("map has an invalid logical entry")
		}
	}
}

func main() {
	padding := layout.New(os.Args[1:])
	var output strings.Builder
	values := make(map[string]int)
	for value := range 24 {
		values[fmt.Sprintf("k%02d", value)] = value
	}
	expectedStrings := make(map[string]int)
	for value := range 24 {
		expectedStrings[fmt.Sprintf("k%02d", value)] = value
	}
	requireMap(values, expectedStrings)
	formatString := func(key string) string { return key }
	appendMap(&output, "create", values, formatString)
	appendMap(&output, "string", values, formatString)
	cloned := maps.Clone(values)
	requireMap(cloned, expectedStrings)
	appendMap(&output, "clone", cloned, formatString)
	delete(values, "k00")
	values["k00"] = 100
	if cloned["k00"] != 0 || values["k00"] != 100 {
		panic("map clone did not preserve an independent lifecycle")
	}
	appendMap(&output, "delete-reinsert", values, formatString)
	clear(values)
	expectedStrings = make(map[string]int)
	for value := range 24 {
		values[fmt.Sprintf("r%02d", value)] = value
		expectedStrings[fmt.Sprintf("r%02d", value)] = value
	}
	requireMap(values, expectedStrings)
	appendMap(&output, "clear", values, formatString)

	uint8s := make(map[uint8]int)
	uint16s := make(map[uint16]int)
	uint32s := make(map[uint32]int)
	uint64s := make(map[uint64]int)
	expectedUint8s := make(map[uint8]int)
	expectedUint16s := make(map[uint16]int)
	expectedUint32s := make(map[uint32]int)
	expectedUint64s := make(map[uint64]int)
	for value := range 24 {
		key8 := uint8(value * 11)
		key16 := uint16(value * 2731)
		key32 := uint32(value) * 2654435761
		key64 := uint64(value) * 11400714819323198485
		uint8s[key8], expectedUint8s[key8] = value, value
		uint16s[key16], expectedUint16s[key16] = value, value
		uint32s[key32], expectedUint32s[key32] = value, value
		uint64s[key64], expectedUint64s[key64] = value, value
	}
	requireMap(uint8s, expectedUint8s)
	requireMap(uint16s, expectedUint16s)
	requireMap(uint32s, expectedUint32s)
	requireMap(uint64s, expectedUint64s)
	appendMap(&output, "uint8", uint8s, func(key uint8) string { return fmt.Sprintf("%02x", key) })
	appendMap(&output, "uint16", uint16s, func(key uint16) string { return fmt.Sprintf("%04x", key) })
	appendMap(&output, "uint32", uint32s, func(key uint32) string { return fmt.Sprintf("%08x", key) })
	appendMap(&output, "uint64", uint64s, func(key uint64) string { return fmt.Sprintf("%016x", key) })

	float32s := make(map[float32]int)
	float64s := make(map[float64]int)
	complex64s := make(map[complex64]int)
	complex128s := make(map[complex128]int)
	for value := range 24 {
		float32Key := float32(value)
		float64Key := float64(value)
		if value >= 12 {
			float32Key = math.Float32frombits(0x7fc00000 + uint32(value))
			float64Key = math.Float64frombits(0x7ff8000000000000 + uint64(value))
		}
		float32s[float32Key] = value
		float64s[float64Key] = value
		complex64s[complex(float32Key, float32(value+1))] = value
		complex128s[complex(float64Key, float64(value+1))] = value
	}
	if len(float32s) != 24 || len(float64s) != 24 || len(complex64s) != 24 || len(complex128s) != 24 {
		panic("floating map has an invalid cardinality")
	}
	appendMap(&output, "float32", float32s, func(key float32) string {
		return fmt.Sprintf("%08x", math.Float32bits(key))
	})
	appendMap(&output, "float64", float64s, func(key float64) string {
		return fmt.Sprintf("%016x", math.Float64bits(key))
	})
	appendMap(&output, "complex64", complex64s, func(key complex64) string {
		return fmt.Sprintf("%08x-%08x", math.Float32bits(real(key)), math.Float32bits(imag(key)))
	})
	appendMap(&output, "complex128", complex128s, func(key complex128) string {
		return fmt.Sprintf("%016x-%016x", math.Float64bits(real(key)), math.Float64bits(imag(key)))
	})

	emptyInterfaces := make(map[any]int)
	nonEmptyInterfaces := make(map[nonEmptyInterface]int)
	arrays := make(map[[2]uint64]int)
	structs := make(map[structKey]int)
	for value := range 24 {
		emptyInterfaces[emptyInterfaceKey{index: value, label: fmt.Sprintf("e%02d", value)}] = value
		nonEmptyInterfaces[nonEmptyInterfaceKey(value)] = value
		arrays[[2]uint64{uint64(value), uint64(value * value)}] = value
		structs[structKey{index: uint64(value), label: fmt.Sprintf("s%02d", value)}] = value
	}
	if len(emptyInterfaces) != 24 || len(nonEmptyInterfaces) != 24 || len(arrays) != 24 || len(structs) != 24 {
		panic("composite map has an invalid cardinality")
	}
	for value := range 24 {
		if emptyInterfaces[emptyInterfaceKey{index: value, label: fmt.Sprintf("e%02d", value)}] != value ||
			nonEmptyInterfaces[nonEmptyInterfaceKey(value)] != value ||
			arrays[[2]uint64{uint64(value), uint64(value * value)}] != value ||
			structs[structKey{index: uint64(value), label: fmt.Sprintf("s%02d", value)}] != value {
			panic("composite map has an invalid logical entry")
		}
	}
	appendMap(&output, "empty-interface", emptyInterfaces, func(key any) string {
		value := key.(emptyInterfaceKey)
		return fmt.Sprintf("%d-%s", value.index, value.label)
	})
	appendMap(&output, "non-empty-interface", nonEmptyInterfaces, func(key nonEmptyInterface) string {
		return fmt.Sprintf("%d", key.index())
	})
	appendMap(&output, "array", arrays, func(key [2]uint64) string {
		return fmt.Sprintf("%d-%d", key[0], key[1])
	})
	appendMap(&output, "struct", structs, func(key structKey) string {
		return fmt.Sprintf("%d-%s", key.index, key.label)
	})

	lifecycle := make(map[int]int)
	for value := range 4096 {
		lifecycle[value] = value * value
	}
	for value := 0; value < 4096; value += 2 {
		delete(lifecycle, value)
	}
	for value := 0; value < 4096; value += 2 {
		lifecycle[value] = -value
	}
	if len(lifecycle) != 4096 {
		panic("map lifecycle changed its cardinality")
	}
	for value := range 4096 {
		expected := value * value
		if value%2 == 0 {
			expected = -value
		}
		if lifecycle[value] != expected {
			panic("map lifecycle returned an invalid value")
		}
	}
	growthDigest := uint64(14695981039346656037)
	for key, value := range lifecycle {
		growthDigest ^= uint64(uint32(key))<<32 | uint64(uint32(value))
		growthDigest *= 1099511628211
	}
	fmt.Fprintf(&output, "growth:%016x\n", growthDigest)

	small := map[int]int{0: 0, 1: 1, 2: 4, 3: 9}
	requireMap(small, map[int]int{0: 0, 1: 1, 2: 4, 3: 9})
	appendMap(&output, "small", small, func(key int) string { return fmt.Sprintf("%d", key) })

	nans := make(map[float64]int)
	for value := range 8 {
		key := math.Float64frombits(0x7ff8000000000000 + uint64(value))
		nans[key] = value
	}
	if len(nans) != 8 {
		panic("NaN map has an invalid cardinality")
	}
	seenNaNs := make([]bool, 8)
	for key, value := range nans {
		if value < 0 || value >= len(seenNaNs) || seenNaNs[value] || math.Float64bits(key) != 0x7ff8000000000000+uint64(value) {
			panic("NaN map has an invalid logical entry")
		}
		seenNaNs[value] = true
	}
	output.WriteString("nan:")
	for key, value := range nans {
		fmt.Fprintf(&output, "%x=%d,", math.Float64bits(key), value)
	}
	output.WriteByte('\n')
	output.WriteString("maps-oracle:ok\n")
	fmt.Print(output.String())
	padding.Finish()
}
