// clock_cgo links C code; the seeded runtime refuses cgo binaries.
package main

/*
static int answer(void) { return 42; }
*/
import "C"

import "fmt"

func main() {
	fmt.Println(int(C.answer()))
}
