package gomadfs

func ClosedProcessHandleForTest() *Handle {
	return &Handle{implementation: &processHandle{closed: true}}
}
