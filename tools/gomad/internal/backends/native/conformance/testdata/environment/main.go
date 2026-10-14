package main

import (
	"fmt"
	"os"
)

var initialized []string

func init() {
	initialized = os.Environ()
}

func main() {
	fmt.Printf("init=%q main=%q\n", initialized, os.Environ())
}
