package go123

import (
	"unsafe"

	"github.com/temporalio/gomad/gomadruntime"
)

func InternalSynctest_Run(f func()) {
	f()
}

func InternalSynctest_Wait() {
	gomadruntime.Yield()
}

func InternalSynctest_IsInBubble() bool {
	return false
}

func InternalSynctest_associate(unsafe.Pointer) int {
	return 0
}

func InternalSynctest_disassociate(unsafe.Pointer) {}

func InternalSynctest_isAssociated(unsafe.Pointer) bool {
	return false
}

func InternalSynctest_acquire() any {
	return nil
}

func InternalSynctest_release(any) {}

func InternalSynctest_inBubble(_ any, f func()) {
	f()
}
