package main

import (
	"fmt"
	"os/user"
)

func main() {
	current, err := user.Current()
	if current != nil || err == nil {
		panic(fmt.Sprintf("user.Current() = %#v, %v", current, err))
	}
	fmt.Println("ok")
}
