package layout

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
)

const maximumPadding = 4 << 20
const argumentPrefix = "-gomad-address-padding="

type Padding struct {
	blocks []*[64]byte
	marker *[64]byte
}

func New(arguments []string) Padding {
	size, present, err := parseArguments(arguments)
	if !present {
		return Padding{}
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s must be a decimal byte count up to %d\n", argumentPrefix[:len(argumentPrefix)-1], maximumPadding)
		os.Exit(2)
	}
	padding := Padding{blocks: make([]*[64]byte, int((size+63)/64))}
	for index := range padding.blocks {
		padding.blocks[index] = new([64]byte)
	}
	padding.marker = new([64]byte)
	return padding
}

func parseArguments(arguments []string) (uint64, bool, error) {
	var value string
	present := false
	for _, argument := range arguments {
		current, found := strings.CutPrefix(argument, argumentPrefix)
		if !found {
			continue
		}
		if present {
			return 0, true, errors.New("duplicate address padding argument")
		}
		value = current
		present = true
	}
	if !present {
		return 0, false, nil
	}
	size, err := parse(value)
	return size, true, err
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
