package go123

import "unsafe"

func Weak_runtime_registerWeakPointer(pointer unsafe.Pointer) unsafe.Pointer {
	return pointer
}

func Weak_runtime_makeStrongFromWeak(pointer unsafe.Pointer) unsafe.Pointer {
	return pointer
}
