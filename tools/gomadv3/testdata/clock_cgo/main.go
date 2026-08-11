package main

/*
int gomad_cgo_value(void) { return 42; }
*/
import "C"

import "fmt"

func main() {
	fmt.Println(C.gomad_cgo_value())
}
