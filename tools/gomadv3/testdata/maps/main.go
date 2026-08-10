package main

import (
	"fmt"
	"maps"
	"math"
	"strings"
)

func appendMap(output *strings.Builder, prefix string, values map[string]int) {
	fmt.Fprint(output, prefix, ":")
	for key, value := range values {
		fmt.Fprintf(output, "%s=%d,", key, value)
	}
	output.WriteByte('\n')
}

func main() {
	var output strings.Builder
	values := make(map[string]int)
	for value := range 24 {
		values[fmt.Sprintf("k%02d", value)] = value
	}
	appendMap(&output, "create", values)
	appendMap(&output, "clone", maps.Clone(values))
	clear(values)
	for value := range 24 {
		values[fmt.Sprintf("r%02d", value)] = value
	}
	appendMap(&output, "clear", values)

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
	fmt.Print(output.String())
}
