package hooks

import "github.com/temporalio/gomad/gomadruntime"

func Os_runtime_args() []string {
	// TODO: make this configurable / fetch this from the machine API?
	return []string{"gomadapp"}
}

func Os_sigpipe() {
	panic("gomad not implemented")
}

func Os_runtime_beforeExit(exitCode int) {
	panic("gomad not implemented")
}

func Os_runtime_rand() uint64 {
	return gomadruntime.Fastrand64()
}

func Os_checkClonePidfd() {
	panic("gomad not implemented")
}

func Os_ignoreSIGSYS() {
	panic("gomad not implemented")
}

func Os_restoreSIGSYS() {
	panic("gomad not implemented")
}
