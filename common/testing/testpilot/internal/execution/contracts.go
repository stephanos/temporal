package execution

import (
	"context"
	"reflect"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/contract"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
)

type Decision uint8

const (
	Continue Decision = iota
	Stop
)

// Monitor callbacks receive independent event/Run snapshots. Every callback must return when
// its Executor-bounded context is canceled; execution never manufactures a goroutine timeout.
type Monitor interface {
	Observe(context.Context, *testpilotspb.RunEvent) (Decision, error)
	Close(context.Context, *testpilotspb.Run) (*testpilotspb.Verdict, error)
}

// MonitorFactory creates fresh evaluation state before Run creation or target effects.
// Its prepared Contract can inspect ProgramView, but cannot access scheduling or Slot state.
type MonitorFactory interface {
	New(context.Context, ProgramView) (Monitor, error)
}

func NewMonitor(ctx context.Context, factory MonitorFactory, view ProgramView) (Monitor, error) {
	if isNil(ctx) || isNil(factory) || view.programID == "" {
		return nil, invalid(ir.Malformed, "monitor", "context, factory and prepared Program view are required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	monitor, err := factory.New(ctx, view)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if isNil(monitor) {
		return nil, invalid(ir.Malformed, "monitor", "factory returned no Monitor")
	}
	return monitor, nil
}

// Driver identity is a non-secret snapshot and may be read without target I/O. Open and all
// Session methods must honor caller bounds; the Executor never wraps them in goroutines.
type Driver interface {
	Identity(context.Context) (contract.DriverIdentity, error)
	Validate(context.Context, *PreparedProgram) error
	Open(context.Context, string, *PreparedProgram) (contract.Session, error)
}

func isNil(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice, reflect.UnsafePointer:
		return reflected.IsNil()
	default:
		return false
	}
}
