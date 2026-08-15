package layout

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
)

const maximumPadding = 4 << 20

type Padding struct {
	blocks []*[64]byte
	marker *[64]byte
}

func New() Padding {
	value, present := os.LookupEnv("GOMADV3_ADDRESS_PADDING")
	if !present {
		return Padding{}
	}
	size, err := parse(value)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GOMADV3_ADDRESS_PADDING must be a decimal byte count up to %d\n", maximumPadding)
		os.Exit(2)
	}
	padding := Padding{blocks: make([]*[64]byte, int((size+63)/64))}
	for index := range padding.blocks {
		padding.blocks[index] = new([64]byte)
	}
	padding.marker = new([64]byte)
	return padding
}

func (padding Padding) Finish() {
	if padding.marker != nil {
		fmt.Printf("GOMADV3_ADDRESS %p\n", padding.marker)
	}
	runtime.KeepAlive(padding)
}

func parse(value string) (uint64, error) {
	if value == "" {
		return 0, strconv.ErrSyntax
	}
	for _, digit := range value {
		if digit < '0' || digit > '9' {
			return 0, strconv.ErrSyntax
		}
	}
	size, err := strconv.ParseUint(value, 10, 23)
	if err != nil || size > maximumPadding {
		return 0, strconv.ErrRange
	}
	return size, nil
}
