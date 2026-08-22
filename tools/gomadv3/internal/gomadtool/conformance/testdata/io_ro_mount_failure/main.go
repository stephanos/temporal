package main

import (
	"fmt"
	"os"
)

func main() {
	contents, err := os.ReadFile("/mounted/schema.sql")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(3)
	}
	fmt.Print(string(contents))
	os.Exit(2)
}
