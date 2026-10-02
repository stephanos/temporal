package temporal

import (
	"context"
	"errors"
	"fmt"
	"sync"

	enumspb "go.temporal.io/api/enums/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal/control"
	"go.temporal.io/server/common/testing/testpilot/temporal/internal/delivery"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// ErrNoDeliveryControl refuses a Program that holds a delivery on a Driver whose environment
// supplies no delivery control, such as one that reaches a server it does not run.
var ErrNoDeliveryControl = errors.New("the environment supplies no delivery control")

// ActivityPoller is the poll a released delivery reaches admission through, as a worker's would.
type ActivityPoller func(context.Context, *workflowservice.PollActivityTaskQueueRequest) (*workflowservice.PollActivityTaskQueueResponse, error)

const startActivityMethod = "temporal.api.workflowservice.v1.WorkflowService.StartActivityExecution"

func deliveryFault(kind testpilotspb.FaultKind) bool {
	return kind == testpilotspb.FAULT_KIND_DELIVERY_HOLD || kind == testpilotspb.FAULT_KIND_DELIVERY_RELEASE
}

type deliveryQueue struct{ namespace, queue string }

// planDeliveries is the queue of each task-queue role a delivery fault of the Program names. A
// release needs a hold of the same role before it, so a Program that releases what it never holds
// is refused, and so is any delivery fault where the environment supplies no control.
func planDeliveries(program testpilot.PreparedProgram, available bool) (map[string]deliveryQueue, error) {
	roles := map[string]testpilot.PreparedRole{}
	for _, role := range program.Roles() {
		roles[role.ID] = role
	}
	plan := map[string]deliveryQueue{}
	held := map[string]bool{}
	for _, entrypoint := range append(program.Entrypoints(), cleanupPlans(program)...) {
		for _, instruction := range entrypoint.Instructions() {
			fault := instruction.Source().GetInstruction().GetInjectFault()
			if fault == nil || !deliveryFault(fault.GetKind()) {
				continue
			}
			at := entrypoint.ID() + "/" + instruction.Source().GetInstructionId()
			if !available {
				return nil, fmt.Errorf("%w: %s requests %s", ErrNoDeliveryControl, at, fault.GetKind())
			}
			role, ok := roles[fault.GetRoleId()]
			if !ok || role.Kind != testpilotspb.ROLE_KIND_TASK_QUEUE || role.Resource == "" || role.Namespace == "" {
				return nil, fmt.Errorf("%w: %s holds no declared task queue", ErrInvalid, at)
			}
			if fault.GetKind() == testpilotspb.FAULT_KIND_DELIVERY_HOLD {
				held[role.ID] = true
			} else if !held[role.ID] {
				return nil, fmt.Errorf("%w: %s releases a delivery no earlier instruction holds", ErrInvalid, at)
			}
			plan[role.ID] = deliveryQueue{namespace: role.Namespace, queue: role.Resource}
		}
	}
	return plan, nil
}

func cleanupPlans(program testpilot.PreparedProgram) []testpilot.EntrypointPlan {
	if cleanup, ok := program.Cleanup(); ok {
		return []testpilot.EntrypointPlan{cleanup}
	}
	return nil
}

// deliverySession is one Run's delivery control: it arms a hold when the Run starts the activity
// a held queue serves, and realizes the hold and release faults on it.
type deliverySession struct {
	deliveries *control.Deliveries
	poll       ActivityPoller
	plan       map[string]deliveryQueue
	mu         sync.Mutex
	held       map[string]*control.Held
}

func newDeliverySession(deliveries *control.Deliveries, poll ActivityPoller, plan map[string]deliveryQueue) *deliverySession {
	if len(plan) == 0 {
		return nil
	}
	return &deliverySession{deliveries: deliveries, poll: poll, plan: plan, held: map[string]*control.Held{}}
}

// arm holds the activity a start request names when its queue is one the Program holds, before the
// request is sent, so no dispatch of the activity passes unheld.
func (s *deliverySession) arm(method protoreflect.MethodDescriptor, request proto.Message) error {
	if s == nil || method == nil || string(method.FullName()) != startActivityMethod || request == nil {
		return nil
	}
	binding, _, err := delivery.ActivityStartBinding(request.ProtoReflect())
	if err != nil {
		return errors.Join(ErrInvalid, err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for role, queue := range s.plan {
		if queue.namespace != binding.Namespace || queue.queue != binding.TaskQueue {
			continue
		}
		if _, armed := s.held[role]; armed {
			return fmt.Errorf("%w: role %s holds one activity, and the Run starts another", ErrInvalid, role)
		}
		held, err := s.deliveries.Hold(binding.ActivityID)
		if err != nil {
			return err
		}
		s.held[role] = held
	}
	return nil
}

func (s *deliverySession) inject(ctx context.Context, roleID string, kind testpilotspb.FaultKind) (testpilot.EffectHandle, error) {
	if s == nil {
		return nil, ErrInvalid
	}
	queue, declared := s.plan[roleID]
	if !declared {
		return nil, ErrInvalid
	}
	s.mu.Lock()
	held := s.held[roleID]
	s.mu.Unlock()
	if held == nil {
		return &deliveryEffect{work: func(context.Context) (*testpilotspb.InstructionOutcome, error) {
			return nil, errors.New("the Run started no activity on the held queue")
		}}, nil
	}
	if kind == testpilotspb.FAULT_KIND_DELIVERY_HOLD {
		return &deliveryEffect{work: func(ctx context.Context) (*testpilotspb.InstructionOutcome, error) {
			_, err := held.Await(ctx)
			return &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}, err
		}}, nil
	}
	return &deliveryEffect{work: func(ctx context.Context) (*testpilotspb.InstructionOutcome, error) {
		admission, err := held.Release(ctx, func(ctx context.Context) error { return s.deliver(ctx, queue) })
		return &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, DeliveryAdmission: admission}, err
	}}, nil
}

// deliver polls the held queue until canceled, so the released dispatch is matched and admission
// is asked; a task the poll is handed is admission's, and the observation reports it.
func (s *deliverySession) deliver(ctx context.Context, queue deliveryQueue) error {
	for {
		_, err := s.poll(ctx, &workflowservice.PollActivityTaskQueueRequest{
			Namespace: queue.namespace,
			TaskQueue: &taskqueuepb.TaskQueue{Name: queue.queue, Kind: enumspb.TASK_QUEUE_KIND_NORMAL},
			Identity:  "testpilot-delivery-release",
		})
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil {
			return err
		}
	}
}

func (s *deliverySession) close() {
	if s == nil {
		return
	}
	s.mu.Lock()
	held := s.held
	s.held = map[string]*control.Held{}
	s.mu.Unlock()
	for _, h := range held {
		h.Close()
	}
}

// deliveryEffect runs its work once, on the first Wait. A deadline is the scheduler's to classify;
// any other failure is a delivery control the Driver could not realize, which the Run carries as a
// non-success outcome naming the reason.
type deliveryEffect struct {
	work    func(context.Context) (*testpilotspb.InstructionOutcome, error)
	once    sync.Once
	outcome *testpilotspb.InstructionOutcome
	err     error
}

func (e *deliveryEffect) Wait(ctx context.Context) (testpilot.EffectResult, error) {
	if ctx == nil {
		return testpilot.EffectResult{}, ErrInvalid
	}
	e.once.Do(func() {
		e.outcome, e.err = e.work(ctx)
	})
	if e.err == nil {
		return testpilot.EffectResult{Outcome: e.outcome}, nil
	}
	if ctx.Err() != nil {
		return testpilot.EffectResult{}, e.err
	}
	return testpilot.EffectResult{Outcome: &testpilotspb.InstructionOutcome{
		Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_FAILURE, ProtocolCode: "delivery_not_realized", Detail: e.err.Error(),
	}}, nil
}

func (e *deliveryEffect) Cancel(ctx context.Context) error {
	if ctx == nil {
		return ErrInvalid
	}
	return ctx.Err()
}

func (e *deliveryEffect) Drain(ctx context.Context) error {
	if ctx == nil {
		return ErrInvalid
	}
	return ctx.Err()
}
