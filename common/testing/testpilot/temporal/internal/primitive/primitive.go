// Package primitive holds the primitives the Temporal Drivers share. Each Driver package keeps its
// own sentinel errors and passes them in.
package primitive

import (
	"context"
	"reflect"

	"github.com/nexus-rpc/sdk-go/nexus"
	"go.temporal.io/server/common/testing/testpilot"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// StartWorkflowPath is the full gRPC method path of WorkflowService.StartWorkflowExecution.
const StartWorkflowPath = "/temporal.api.workflowservice.v1.WorkflowService/StartWorkflowExecution"

// NilValue reports whether value is nil or an interface holding a nil reference.
func NilValue(value any) bool {
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

// ContextError returns invalid for a nil context and the context's error otherwise.
func ContextError(ctx context.Context, invalid error) error {
	if NilValue(ctx) {
		return invalid
	}
	return ctx.Err()
}

// Mutex lets Driver operations abandon serialization without a timeout goroutine. Internal
// completion uses Lock so canceled callers cannot prevent capacity from being released.
type Mutex chan struct{}

func NewMutex() Mutex { return make(Mutex, 1) }

func (m Mutex) Lock()   { m <- struct{}{} }
func (m Mutex) Unlock() { <-m }

// LockContext acquires m unless ctx ends first; a nil ctx is rejected with invalid.
func (m Mutex) LockContext(ctx context.Context, invalid error) error {
	if err := ContextError(ctx, invalid); err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case m <- struct{}{}:
	}
	if err := ctx.Err(); err != nil {
		m.Unlock()
		return err
	}
	return nil
}

func CloneEffectResult(result testpilot.EffectResult) testpilot.EffectResult {
	return testpilot.EffectResult{Outcome: proto.CloneOf(result.Outcome), Response: proto.Clone(result.Response)}
}

// NexusHeaderBytes is the byte size of header's names and values.
func NexusHeaderBytes(header nexus.Header) int {
	size := 0
	for key, value := range header {
		size += len(key) + len(value)
	}
	return size
}

// MethodPath is the full gRPC path of method, as Programs name it.
func MethodPath(method protoreflect.MethodDescriptor) string {
	return "/" + string(method.Parent().FullName()) + "/" + string(method.Name())
}

// HasWorkerEntrypoint reports whether any plan is realized by an SDK worker.
func HasWorkerEntrypoint(plans []testpilot.EntrypointPlan) bool {
	for _, plan := range plans {
		if plan.Kind() == testpilot.WorkflowEntrypoint || plan.Kind() == testpilot.ActivityEntrypoint || plan.Kind() == testpilot.NexusHandlerEntrypoint {
			return true
		}
	}
	return false
}
