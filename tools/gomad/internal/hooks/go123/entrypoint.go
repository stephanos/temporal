package go123

import (
	"github.com/temporalio/gomad/gomadruntime"
	"github.com/temporalio/gomad/internal/simulation"
	"github.com/temporalio/gomad/internal/simulation/syscallabi"
	"github.com/temporalio/gomad/internal/testing"
)

func Runtime() gomadruntime.Runtime {
	return runtimeImpl{}
}

type runtimeImpl struct{}

var _ gomadruntime.Runtime = runtimeImpl{}

func (r runtimeImpl) Run(fn func()) {
	simulation.Runtime(fn)
}

func (r runtimeImpl) Setup() {
	syscallabi.Setup()
}

func (r runtimeImpl) TestEntrypoint(match string, skip string, tests []gomadruntime.Test) bool {
	return testing.Entrypoint(match, skip, tests)
}
