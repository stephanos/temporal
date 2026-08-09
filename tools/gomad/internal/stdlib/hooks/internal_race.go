package hooks

import (
	"unsafe"

	simrace "github.com/temporalio/gomad/internal/race"
)

func InternalRace_Acquire(addr unsafe.Pointer) {
	simrace.Acquire(addr)
}

func InternalRace_Release(addr unsafe.Pointer) {
	simrace.Release(addr)
}

func InternalRace_ReleaseMerge(addr unsafe.Pointer) {
	simrace.ReleaseMerge(addr)
}

func InternalRace_Disable() {
	simrace.Disable()
}

func InternalRace_Enable() {
	simrace.Enable()
}

func InternalRace_Read(addr unsafe.Pointer) {
	simrace.Read(addr)
}

func InternalRace_ReadPC(addr unsafe.Pointer, _, _ uintptr) {
	simrace.Read(addr)
}

func InternalRace_ReadObjectPC(_ unsafe.Pointer, addr unsafe.Pointer, _, _ uintptr) {
	simrace.Read(addr)
}

func InternalRace_Write(addr unsafe.Pointer) {
	simrace.Write(addr)
}

func InternalRace_WritePC(addr unsafe.Pointer, _, _ uintptr) {
	simrace.Write(addr)
}

func InternalRace_WriteObjectPC(_ unsafe.Pointer, addr unsafe.Pointer, _, _ uintptr) {
	simrace.Write(addr)
}

func InternalRace_ReadRange(addr unsafe.Pointer, length int) {
	simrace.ReadRange(addr, length)
}

func InternalRace_WriteRange(addr unsafe.Pointer, length int) {
	simrace.WriteRange(addr, length)
}

func InternalRace_Errors() int {
	return simrace.Errors()
}
