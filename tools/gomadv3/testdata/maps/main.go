package main

import (
	"fmt"
	"maps"
	"math"
	"os"
	"runtime"
	"strconv"
	"strings"
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

const maxAddressPadding = 4 << 20

func appendMap[K comparable](output *strings.Builder, prefix string, values map[K]int, formatKey func(K) string) {
	fmt.Fprint(output, prefix, ":")
	for key, value := range values {
		fmt.Fprintf(output, "%s=%d,", formatKey(key), value)
	}
	output.WriteByte('\n')
}

func invalidAddressPadding() {
	fmt.Fprintf(os.Stderr, "GOMADV3_MAP_PADDING must be a decimal byte count up to %d\n", maxAddressPadding)
	os.Exit(2)
}

func addressPadding() ([]*[64]byte, *[64]byte) {
	value, present := os.LookupEnv("GOMADV3_MAP_PADDING")
	if !present {
		return nil, nil
	}
	if value == "" {
		invalidAddressPadding()
	}
	for _, digit := range value {
		if digit < '0' || digit > '9' {
			invalidAddressPadding()
		}
	}
	size, err := strconv.ParseUint(value, 10, 23)
	if err != nil || size > maxAddressPadding {
		invalidAddressPadding()
	}
	padding := make([]*[64]byte, int((size+63)/64))
	for index := range padding {
		padding[index] = new([64]byte)
	}
	marker := new([64]byte)
	return padding, marker
}

func main() {
	padding, marker := addressPadding()
	var output strings.Builder
	values := make(map[string]int)
	for value := range 24 {
		values[fmt.Sprintf("k%02d", value)] = value
	}
	formatString := func(key string) string { return key }
	appendMap(&output, "create", values, formatString)
	appendMap(&output, "string", values, formatString)
	appendMap(&output, "clone", maps.Clone(values), formatString)
	clear(values)
	for value := range 24 {
		values[fmt.Sprintf("r%02d", value)] = value
	}
	appendMap(&output, "clear", values, formatString)

	uint32s := make(map[uint32]int)
	uint64s := make(map[uint64]int)
	for value := range 24 {
		uint32s[uint32(value)*2654435761] = value
		uint64s[uint64(value)*11400714819323198485] = value
	}
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

	nans := make(map[float64]int)
	for value := range 8 {
		key := math.Float64frombits(0x7ff8000000000000 + uint64(value))
		nans[key] = value
	}
	output.WriteString("nan:")
	for key, value := range nans {
		fmt.Fprintf(&output, "%x=%d,", math.Float64bits(key), value)
	}
	output.WriteByte('\n')
	if marker != nil {
		fmt.Fprintf(&output, "GOMADV3_MAP_ADDRESS %p\n", marker)
	}
	fmt.Print(output.String())
	runtime.KeepAlive(padding)
	runtime.KeepAlive(marker)
}
