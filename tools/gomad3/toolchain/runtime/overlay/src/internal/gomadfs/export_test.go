package gomadfs

func ClosedProcessHandleForTest() *Handle {
	return &Handle{fs: processFilesystem, closed: true}
}
