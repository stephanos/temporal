package gomad

import (
	"time"

	"github.com/temporalio/gomad/gomadruntime"
	"github.com/temporalio/gomad/internal/simulation"
)

// IsSim returns if the program is running inside of a gomad simulation.
//
// To include or exclude code or tests only if running inside of a gomad the
// gomad tool also supports a build tag. Use the `//go:build gomad` build
// constraint at the top of a go file to only include it in simulated builds or
// `//go:build !gomad` to exclude it from simulated from builds.
func IsSim() bool {
	return gomadruntime.IsSim()
}

// Yield yields the processor, allowing other goroutines to run. Similar to
// [runtime.Gosched] but for gomad's scheduler.
//
// TODO: Replace with runtime.Gosched() instead?
func Yield() {
	gomadruntime.Yield()
}

/*
type Simulation struct{}

func CurrentSimulation() Simulation {
	return Simulation{}
}

// SetConnected configures the network connectivity between addresses a and b.
//
// TODO: How will this fail? Unrouteable or rejected
// TODO: Support asymmetry
func (s Simulation) SetConnected(a, b string, connected bool) error {
	return simulation.SyscallSetConnected(a, b, connected)
}

// SetDelay configures the network latency between addresses a and b.
//
// TODO: Support asymmetry
func (s Simulation) SetDelay(a, b string, delay time.Duration) error {
	return simulation.SyscallSetDelay(a, b, delay)
}

// SetSimulationTimeout resets the simulation timeout to the given duration.
// After the simulated duration the simululation will fail with a test timeout
// error.
func (s Simulation) SetSimulationTimeout(timeout time.Duration) {
	err := simulation.SyscallSetSimulationTimeout(timeout)
	if err != nil {
		panic(err)
	}
}
*/

// SetConnected configures the network connectivity between addresses a and b.
//
// TODO: How will this fail? Unrouteable or rejected
// TODO: Support asymmetry
func SetConnected(a, b string, connected bool) error {
	return simulation.SyscallSetConnected(a, b, connected)
}

// SetDelay configures the network latency between addresses a and b.
//
// TODO: Support asymmetry
func SetDelay(a, b string, delay time.Duration) error {
	return simulation.SyscallSetDelay(a, b, delay)
}

// SetSimulationTimeout resets the simulation timeout to the given duration.
// After the simulated duration the simululation will fail with a test timeout
// error.
func SetSimulationTimeout(d time.Duration) {
	err := simulation.SyscallSetSimulationTimeout(d)
	if err != nil {
		panic(err)
	}
}
