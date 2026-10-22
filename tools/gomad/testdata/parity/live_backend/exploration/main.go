package main
import ("fmt"; "os")
func choose() int {
	a, b := make(chan int, 1), make(chan int, 1)
	a <- 0; b <- 1
	select { case n := <-a: return n; case n := <-b: return n }
}
func main() {
	a, b := choose(), choose()
	fmt.Printf("choices=%d,%d\n", a, b)
	if a == 1 && b == 1 { os.Exit(17) }
}
