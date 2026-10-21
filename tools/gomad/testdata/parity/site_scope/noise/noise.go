package noise

//go:noinline
func Choose() int {
	left, right := make(chan int, 1), make(chan int, 1)
	left <- 1
	right <- 2
	select {
	case value := <-left:
		return value
	case value := <-right:
		return value
	}
}

//go:noinline
func Start(done chan<- int, value int) {
	go func() { done <- value }()
}
