package main
import ("fmt"; "os")
func choose() int {
	a, b := make(chan int, 1), make(chan int, 1)
	a <- 0; b <- 1
	select { case n := <-a: return n; case n := <-b: return n }
}
func main() {
	failure := choose() == 1
	choose(); choose()
	if failure { fmt.Println("essential early choice failed"); os.Exit(17) }
	fmt.Println("success")
}
