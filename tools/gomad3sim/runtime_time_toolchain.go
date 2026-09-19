//go:build gomad3_toolchain

package gomad3sim

import (
	"errors"
	_ "unsafe"
)

//go:linkname gomadSimulationTimeAdvance runtime.gomadSimulationTimeAdvance
func gomadSimulationTimeAdvance(int64) bool

//go:linkname gomadSimulationTimeTakeArrivals runtime.gomadSimulationTimeTakeArrivals
func gomadSimulationTimeTakeArrivals() uint32

func runtimeProcessTimeAdvance(current int64) error {
	if !gomadSimulationTimeAdvance(current) {
		return errors.New("advance process simulation time")
	}
	return nil
}

func runtimeProcessTimeArrivals() uint32 {
	return gomadSimulationTimeTakeArrivals()
}
