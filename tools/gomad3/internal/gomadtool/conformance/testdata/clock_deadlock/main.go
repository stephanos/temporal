// clock_deadlock blocks forever with no timers or other goroutines, so the
// runtime must report the deadlock instead of advancing virtual time.
package main

func main() {
	select {}
}
