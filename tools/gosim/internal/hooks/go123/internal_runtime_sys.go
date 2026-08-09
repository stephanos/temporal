package go123

func InternalRuntimeSys_EnableDIT() bool {
	return false
}

func InternalRuntimeSys_DITEnabled() bool {
	return false
}

func InternalRuntimeSys_DisableDIT() {}

func InternalRuntimeSys_GetCallerPC() uintptr {
	return 0
}

func InternalRuntimeSys_GetCallerSP() uintptr {
	return 0
}

func InternalRuntimeSys_GetClosurePtr() uintptr {
	return 0
}
