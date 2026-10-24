package controller

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/protobuf/types/known/anypb"
	runv1 "umpire.local/model/umpire/run/v1"
	"umpire.local/model/umpire/runner"
)

// ErrOutsideFence is a start the fence does not name: another activity, or any other execution.
var ErrOutsideFence = errors.New("the canary starts only the activity its fence names")

// activityPrefix begins every activity ID the canary fences, a fresh UUID after it.
const activityPrefix = "umpire-canary-"

// newActivityID is a fresh activity ID for one iteration.
func newActivityID() string { return activityPrefix + uuid.NewString() }

// IsActivityID reports whether id is one the canary fences: its prefix and a UUID.
func IsActivityID(id string) bool {
	rest, ok := strings.CutPrefix(id, activityPrefix)
	if !ok {
		return false
	}
	parsed, err := uuid.Parse(rest)
	return err == nil && parsed.String() == rest
}

// startPrefix begins the full name of every WorkflowService request that creates an execution.
const startPrefix = "temporal.api.workflowservice.v1.Start"

// startActivity is the one start the fence admits, for its own activity ID.
var startActivity = (&workflowservice.StartActivityExecutionRequest{}).ProtoReflect().Descriptor().FullName()

// fencedDriver wraps one Run's Driver so the Run can start nothing but the activity the lease
// already names: cleanup and reconcile act only on the fence's IDs, so they can never miss an
// execution the canary started.
type fencedDriver struct {
	runner.Driver
	activityID string
}

// fenced makes newDriver's Drivers fenced to activityID.
func fenced(newDriver runner.NewDriver, activityID string) runner.NewDriver {
	return func(ctx context.Context) (runner.Driver, error) {
		driver, err := newDriver(ctx)
		if err != nil {
			return nil, err
		}
		return &fencedDriver{Driver: driver, activityID: activityID}, nil
	}
}

func (f *fencedDriver) Start(ctx context.Context, request uint64, payload *anypb.Any) (runner.Exchange, error) {
	if err := f.within(payload); err != nil {
		return nil, err
	}
	return f.Driver.Start(ctx, request, payload)
}

// Close closes the wrapped Driver, when it has anything to close.
func (f *fencedDriver) Close() error {
	if closer, ok := f.Driver.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// Redact keeps the wrapped Driver's redaction, when it has one.
func (f *fencedDriver) Redact(event *runv1.Event) bool {
	if redactor, ok := f.Driver.(runner.Redactor); ok {
		return redactor.Redact(event)
	}
	return false
}

func (f *fencedDriver) within(payload *anypb.Any) error {
	name := payload.MessageName()
	if !strings.HasPrefix(string(name), startPrefix) {
		return nil
	}
	if name != startActivity {
		return fmt.Errorf("%w: %s", ErrOutsideFence, name)
	}
	start := new(workflowservice.StartActivityExecutionRequest)
	if err := payload.UnmarshalTo(start); err != nil {
		return err
	}
	if start.GetActivityId() != f.activityID {
		return fmt.Errorf("%w: the start names another activity", ErrOutsideFence)
	}
	return nil
}
