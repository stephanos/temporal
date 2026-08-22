package main

import (
	"fmt"
	"os"
)

func main() {
	file := os.NewFile(5, "reserved-by-caller")
	if file == nil {
		panic("fd 5 is unavailable")
	}
	if _, err := file.WriteString("preserved"); err != nil {
		panic(fmt.Sprintf("write fd 5: %v", err))
	}
}
