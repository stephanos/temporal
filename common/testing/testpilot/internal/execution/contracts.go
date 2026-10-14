package execution

import (
	"context"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/contract"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/proto"
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
	if ir.IsNil(ctx) || ir.IsNil(factory) || view.programID == "" {
		return nil, ir.Invalid(ir.Malformed, "monitor", "context, factory and prepared Program view are required")
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
	if ir.IsNil(monitor) {
		return nil, ir.Invalid(ir.Malformed, "monitor", "factory returned no Monitor")
	}
	return monitor, nil
}

// EventObserver is given the events a Monitor is, and answers nothing: it has no way to stop, fail
// or decide a Run. It is called inside the Monitor's callback, so an observer takes its snapshot
// and returns; anything slower belongs on a goroutine of its own.
type EventObserver interface {
	Observe(context.Context, *testpilotspb.RunEvent)
}

// Observed puts observer beside monitor. Each event whose evaluation monitor committed reaches
// observer as its own snapshot; the decision, the error and the Verdict stay monitor's alone.
func Observed(monitor Monitor, observer EventObserver) (Monitor, error) {
	if ir.IsNil(monitor) || ir.IsNil(observer) {
		return nil, ir.Invalid(ir.Malformed, "monitor", "Monitor and observer are required")
	}
	return observedMonitor{Monitor: monitor, observer: observer}, nil
}

type observedMonitor struct {
	Monitor
	observer EventObserver
}

func (m observedMonitor) Observe(ctx context.Context, event *testpilotspb.RunEvent) (Decision, error) {
	// The Monitor owns the event it is handed and may change it, so the snapshot is taken first.
	snapshot := proto.CloneOf(event)
	decision, err := m.Monitor.Observe(ctx, event)
	if err == nil {
		m.observer.Observe(ctx, snapshot)
	}
	return decision, err
}

// Driver is the one Driver call Run makes. Open and all Session methods must honor caller bounds;
// the Executor never wraps them in goroutines.
type Driver interface {
	Open(context.Context, string, *PreparedProgram) (contract.Session, error)
}
