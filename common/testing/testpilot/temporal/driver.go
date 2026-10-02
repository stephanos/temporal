// Package temporal composes controller transports and SDK workers behind one Testpilot Driver.
package temporal

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal/internal/delivery"
	"go.temporal.io/server/common/testing/testpilot/temporal/internal/primitive"
	"go.temporal.io/server/common/testing/testpilot/temporal/server"
	workerhost "go.temporal.io/server/common/testing/testpilot/temporal/worker"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var ErrInvalid = errors.New("invalid composite Temporal Driver input")

type Endpoint = server.Endpoint

type Options struct {
	Profile               testpilot.ProfileSpec
	ServerEndpoints       map[string]Endpoint
	SystemCallbackBaseURL string
	HTTPClient            *http.Client
	SDKClient             client.Client
	WorkerRoleID          string
	WorkerStopTimeout     time.Duration
	// Deliveries holds activity deliveries inside the server under test. Only an environment that
	// runs that server can supply it; without it, a Program that holds a delivery is refused at
	// Validate, before any I/O.
	Deliveries DeliveryControl
}

// Driver keeps server transport authority and SDK worker authority in their owning packages.
type Driver struct {
	profile    testpilot.ProfileSpec
	controller *server.Driver
	worker     *workerhost.Driver
	deliveries DeliveryControl
	poll       ActivityPoller
}

func New(options Options) (*Driver, error) {
	// Refused here, as the wiring mistake it is: left to Validate, it would read as an environment
	// that supplies no delivery control.
	if options.Deliveries != nil && primitive.NilValue(options.SDKClient) {
		return nil, fmt.Errorf("%w: a delivery control needs the SDK client its release polls through", ErrInvalid)
	}
	controller, err := server.New(server.Options{
		Profile: options.Profile, Endpoints: options.ServerEndpoints,
	})
	if err != nil {
		return nil, err
	}
	workers, err := workerhost.New(workerhost.Options{
		Profile: options.Profile, Client: options.SDKClient, WorkerRoleID: options.WorkerRoleID,
		WorkerStopTimeout: options.WorkerStopTimeout, SystemCallbackBaseURL: options.SystemCallbackBaseURL, HTTPClient: options.HTTPClient,
	})
	if err != nil {
		closeErr := controller.Close(context.Background())
		return nil, errors.Join(err, closeErr)
	}
	driver := &Driver{profile: options.Profile.Snapshot(), controller: controller, worker: workers, deliveries: options.Deliveries}
	if options.Deliveries != nil {
		service := options.SDKClient.WorkflowService()
		driver.poll = func(ctx context.Context, request *workflowservice.PollActivityTaskQueueRequest) (*workflowservice.PollActivityTaskQueueResponse, error) {
			return service.PollActivityTaskQueue(ctx, request)
		}
	}
	return driver, nil
}

func (h *Driver) Snapshot() testpilot.ProfileSpec {
	if h == nil {
		return testpilot.ProfileSpec{}
	}
	return h.profile.Snapshot()
}

func (h *Driver) Identity(ctx context.Context) (testpilot.DriverIdentity, error) {
	if h == nil || ctx == nil {
		return testpilot.DriverIdentity{}, ErrInvalid
	}
	return h.controller.Identity(ctx)
}

func (h *Driver) Validate(ctx context.Context, program testpilot.PreparedProgram) error {
	if h == nil || ctx == nil {
		return ErrInvalid
	}
	if err := h.controller.Validate(ctx, program); err != nil {
		return err
	}
	if _, err := planDeliveries(program, h.poll != nil); err != nil {
		return err
	}
	return h.worker.Validate(ctx, program)
}

func (h *Driver) Open(ctx context.Context, runID string, program testpilot.PreparedProgram) (testpilot.Session, error) {
	if h == nil || ctx == nil || runID == "" {
		return nil, ErrInvalid
	}
	plan, err := planDeliveries(program, h.poll != nil)
	if err != nil {
		return nil, err
	}
	controller, err := h.controller.OpenSession(ctx, runID, program)
	if err != nil {
		return nil, err
	}
	if !primitive.HasWorkerEntrypoint(program.Entrypoints()) && !hasFaultInstruction(program) {
		session := newPreparedCompositeSession(controller, nil, program)
		session.deliveries = newDeliverySession(h.deliveries, h.poll, plan)
		return session, nil
	}
	bridge, err := controller.Bridge(ctx)
	if err != nil {
		return nil, errors.Join(err, controller.Close(context.Background()))
	}
	worker, err := h.worker.OpenSession(ctx, runID, program, workerhost.SessionOptions{
		Bridge: bridge, NewHandle: controller.NewHandle,
		Diagnose:   controller.Diagnose,
		Quarantine: quarantineWorkerHandle,
	})
	if err != nil {
		return nil, errors.Join(err, controller.Close(context.Background()))
	}
	session := newPreparedCompositeSession(controller, worker, program)
	session.deliveries = newDeliverySession(h.deliveries, h.poll, plan)
	return session, nil
}

// A fault is realized by the worker Driver, so a Program that requests one is routed there even
// when every other worker use is absent; the worker Driver decides whether such a Program is
// realizable at all.
func hasFaultInstruction(program testpilot.PreparedProgram) bool {
	return workerhost.DeclaresFault(workerhost.ProgramPlans(program))
}

func (h *Driver) Close(ctx context.Context) error {
	if h == nil || ctx == nil {
		return ErrInvalid
	}
	return errors.Join(h.controller.Close(ctx), h.worker.Close(ctx))
}

type workerSession interface {
	Reserve(context.Context, testpilot.ReservationRequest) ([]testpilot.ReservationHandle, error)
	InjectFault(context.Context, testpilot.Coordinate, string, testpilotspb.FaultKind) (testpilot.EffectHandle, error)
	Quarantine(context.Context, testpilot.EffectHandle) error
	Close(context.Context) error
	Diagnose(context.Context, string, *testpilotspb.RunDiagnostic) error
}

type carrierSession interface {
	CreateCarrier(context.Context, testpilot.Coordinate, testpilot.ReservationCarrierPlan, delivery.WorkflowBinding, []testpilot.ReservationHandle) (*workerhost.Carrier, error)
	CreateActivityCarrier(context.Context, testpilot.Coordinate, testpilot.ReservationCarrierPlan, delivery.ActivityBinding, []testpilot.ReservationHandle) (*workerhost.Carrier, error)
}

type compositeSession struct {
	controller   testpilot.Session
	worker       workerSession
	program      testpilot.PreparedProgram
	mu           sync.Mutex
	reservations map[testpilot.Coordinate][]testpilot.ReservationHandle
	deliveries   *deliverySession
}

func newPreparedCompositeSession(controller testpilot.Session, worker workerSession, program testpilot.PreparedProgram) *compositeSession {
	return &compositeSession{controller: controller, worker: worker, program: program, reservations: make(map[testpilot.Coordinate][]testpilot.ReservationHandle)}
}

func (s *compositeSession) Reserve(ctx context.Context, request testpilot.ReservationRequest) ([]testpilot.ReservationHandle, error) {
	if s.worker == nil {
		return nil, ErrInvalid
	}
	handles, err := s.worker.Reserve(ctx, request)
	if err != nil {
		return handles, fmt.Errorf("reserve %s from %+v: %w", request.EntrypointID, request.Origin, err)
	}
	s.mu.Lock()
	s.reservations[request.Origin] = append(s.reservations[request.Origin], handles...)
	s.mu.Unlock()
	return handles, nil
}

func (s *compositeSession) PollRPC(ctx context.Context, coordinate testpilot.Coordinate, role string, method protoreflect.MethodDescriptor, request proto.Message, interval time.Duration, satisfied testpilot.PollPredicate) (testpilot.EffectHandle, error) {
	return s.controller.PollRPC(ctx, coordinate, role, method, request, interval, satisfied)
}

func (s *compositeSession) InvokeRPC(ctx context.Context, coordinate testpilot.Coordinate, role string, method protoreflect.MethodDescriptor, request proto.Message) (testpilot.EffectHandle, error) {
	if err := s.deliveries.arm(method, request); err != nil {
		return nil, err
	}
	plan, carrierRequired := s.program.ReservationCarrier(coordinate.EntrypointID, coordinate.InstructionID)
	if !carrierRequired {
		return s.controller.InvokeRPC(ctx, coordinate, role, method, request)
	}
	worker, ok := s.worker.(carrierSession)
	if !ok || method == nil || primitive.NilValue(request) {
		return nil, ErrInvalid
	}
	s.mu.Lock()
	handles := append([]testpilot.ReservationHandle(nil), s.reservations[coordinate]...)
	s.mu.Unlock()
	carrier, err := createCarrier(ctx, worker, coordinate, plan, request, handles)
	if err != nil {
		return nil, err
	}
	maximum := s.program.Limits().GetMaxRequestBytes()
	prepared, err := carrier.PrepareRPC(ctx, role, method, request, maximum)
	if err != nil {
		_, terminalErr := carrier.TriggerTerminal(ctx, delivery.TriggerRejected)
		return nil, errors.Join(err, terminalErr)
	}
	handle, err := s.controller.InvokeRPC(ctx, coordinate, role, method, prepared)
	if err != nil {
		_, terminalErr := carrier.TriggerTerminal(ctx, delivery.TriggerRejected)
		return nil, errors.Join(err, terminalErr)
	}
	s.mu.Lock()
	delete(s.reservations, coordinate)
	s.mu.Unlock()
	cleanupTimeout := time.Duration(s.program.Limits().GetMaxCleanupDurationMilliseconds()) * time.Millisecond
	return &carrierEffect{EffectHandle: handle, carrier: carrier, cleanupTimeout: cleanupTimeout}, nil
}

// createCarrier makes the worker carrier of the start the plan names: a StartActivityExecution
// carries the activation of the standalone activity it starts, and any other carrier is a
// StartWorkflow request.
func createCarrier(ctx context.Context, worker carrierSession, coordinate testpilot.Coordinate, plan testpilot.ReservationCarrierPlan, request proto.Message, handles []testpilot.ReservationHandle) (*workerhost.Carrier, error) {
	if plan.Method == delivery.StartActivityPath {
		binding, err := activityCarrierBinding(request)
		if err != nil {
			return nil, err
		}
		return worker.CreateActivityCarrier(ctx, coordinate, plan, binding, handles)
	}
	binding, err := carrierBinding(request)
	if err != nil {
		return nil, err
	}
	return worker.CreateCarrier(ctx, coordinate, plan, binding, handles)
}

// activityCarrierBinding is the activity binding a carried StartActivityExecution request names;
// every field of it must be set.
func activityCarrierBinding(request proto.Message) (delivery.ActivityBinding, error) {
	binding, _, err := delivery.ActivityStartBinding(request.ProtoReflect())
	if err != nil {
		return delivery.ActivityBinding{}, errors.Join(ErrInvalid, err)
	}
	if binding.Namespace == "" || binding.ActivityID == "" || binding.ActivityType == "" || binding.TaskQueue == "" {
		return delivery.ActivityBinding{}, ErrInvalid
	}
	return binding, nil
}

// carrierBinding is the workflow binding a carried StartWorkflow request names; every field of it
// must be set.
func carrierBinding(request proto.Message) (delivery.WorkflowBinding, error) {
	binding, _, err := delivery.StartBinding(request.ProtoReflect())
	if err != nil {
		return delivery.WorkflowBinding{}, errors.Join(ErrInvalid, err)
	}
	if binding.Namespace == "" || binding.WorkflowID == "" || binding.WorkflowType == "" || binding.TaskQueue == "" {
		return delivery.WorkflowBinding{}, ErrInvalid
	}
	return binding, nil
}

func (s *compositeSession) InvokeHandle(ctx context.Context, coordinate testpilot.Coordinate, handle testpilot.OpaqueHandle, value proto.Message) (testpilot.EffectHandle, error) {
	return s.controller.InvokeHandle(ctx, coordinate, handle, value)
}

// A worker-lifecycle outage is the worker Session's to realize; a Program with no worker use has
// no worker Session and no queue to stop. A delivery control is the delivery session's.
func (s *compositeSession) InjectFault(ctx context.Context, coordinate testpilot.Coordinate, roleID string, kind testpilotspb.FaultKind) (testpilot.EffectHandle, error) {
	if s != nil && deliveryFault(kind) {
		return s.deliveries.inject(ctx, roleID, kind)
	}
	if s == nil || s.worker == nil {
		return nil, ErrInvalid
	}
	return s.worker.InjectFault(ctx, coordinate, roleID, kind)
}
func (s *compositeSession) Bridge(ctx context.Context) (testpilot.HandleBridge, error) {
	return s.controller.Bridge(ctx)
}
func (s *compositeSession) Quarantine(ctx context.Context, handle testpilot.EffectHandle) error {
	if carried, ok := handle.(*carrierEffect); ok {
		return s.controller.Quarantine(ctx, carried.EffectHandle)
	}
	if _, ok := handle.(testpilot.ReservationHandle); ok && s.worker != nil {
		return s.worker.Quarantine(ctx, handle)
	}
	return s.controller.Quarantine(ctx, handle)
}
func (s *compositeSession) Close(ctx context.Context) error {
	s.deliveries.close()
	var workerErr error
	if s.worker != nil {
		workerErr = s.worker.Close(ctx)
	}
	return errors.Join(workerErr, s.controller.Close(ctx))
}
func (s *compositeSession) Diagnose(ctx context.Context, runID string, diagnostic *testpilotspb.RunDiagnostic) error {
	return s.controller.Diagnose(ctx, runID, diagnostic)
}

type carrierEffect struct {
	testpilot.EffectHandle
	carrier        terminalCarrier
	cleanupTimeout time.Duration
	mu             sync.Mutex
	finished       bool
	disposition    delivery.TriggerStatus
	response       delivery.StartResponse
}

func (e *carrierEffect) Wait(ctx context.Context) (testpilot.EffectResult, error) {
	result, waitErr := e.EffectHandle.Wait(ctx)
	return result, errors.Join(waitErr, e.finish(result, waitErr))
}

func (e *carrierEffect) Cancel(ctx context.Context) error {
	err := e.EffectHandle.Cancel(ctx)
	return errors.Join(err, e.trigger(delivery.TriggerCanceled))
}

func (e *carrierEffect) Drain(ctx context.Context) error {
	drainErr := e.EffectHandle.Drain(ctx)
	if drainErr != nil {
		return errors.Join(drainErr, e.trigger(delivery.TriggerUncertain))
	}
	result, waitErr := e.EffectHandle.Wait(ctx)
	return errors.Join(waitErr, e.finish(result, waitErr))
}

type terminalCarrier interface {
	PinStartResponse(context.Context, delivery.StartResponse) error
	TriggerTerminal(context.Context, delivery.TriggerStatus) (int, error)
}

func (e *carrierEffect) finish(result testpilot.EffectResult, waitErr error) error {
	disposition := delivery.TriggerUncertain
	var response delivery.StartResponse
	var responseErr error
	if waitErr == nil && result.Outcome != nil {
		switch result.Outcome.GetStatus() {
		case testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED:
			response, responseErr = startResponse(result.Response)
			if responseErr == nil {
				disposition = delivery.TriggerSucceeded
			}
		case testpilotspb.INSTRUCTION_OUTCOME_STATUS_CANCELED:
			disposition = delivery.TriggerCanceled
		default:
			disposition = delivery.TriggerNonSuccess
		}
	}
	return errors.Join(responseErr, e.finalize(disposition, response))
}

func (e *carrierEffect) trigger(disposition delivery.TriggerStatus) error {
	return e.finalize(disposition, nil)
}

func (e *carrierEffect) finalize(disposition delivery.TriggerStatus, response delivery.StartResponse) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.finished {
		return nil
	}
	if e.disposition == 0 {
		e.disposition = disposition
	} else {
		disposition = e.disposition
	}
	if response != nil {
		e.response = response
	}
	ctx, cancel := context.WithTimeout(context.Background(), e.cleanupTimeout)
	defer cancel()
	if disposition == delivery.TriggerSucceeded {
		if e.response == nil {
			return ErrInvalid
		}
		if err := e.carrier.PinStartResponse(ctx, e.response); err != nil {
			return err
		}
	}
	_, err := e.carrier.TriggerTerminal(ctx, disposition)
	if err == nil {
		e.finished = true
	}
	return err
}

// startResponse decodes what a carried start answered with, a StartWorkflowExecution or a
// StartActivityExecution response, into the run it started. The response is its own copy.
func startResponse(response proto.Message) (delivery.StartResponse, error) {
	if response == nil {
		return nil, ErrInvalid
	}
	var start interface {
		proto.Message
		delivery.StartResponse
	}
	switch response.ProtoReflect().Descriptor().FullName() {
	case (&workflowservice.StartWorkflowExecutionResponse{}).ProtoReflect().Descriptor().FullName():
		start = &workflowservice.StartWorkflowExecutionResponse{}
	case (&workflowservice.StartActivityExecutionResponse{}).ProtoReflect().Descriptor().FullName():
		start = &workflowservice.StartActivityExecutionResponse{}
	default:
		return nil, ErrInvalid
	}
	wire, err := proto.Marshal(response)
	if err != nil {
		return nil, err
	}
	if err := proto.Unmarshal(wire, start); err != nil || start.GetRunId() == "" {
		return nil, ErrInvalid
	}
	return start, nil
}

func quarantineWorkerHandle(ctx context.Context, handle testpilot.EffectHandle, complete func()) error {
	if ctx == nil || handle == nil || complete == nil {
		return ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	go func() {
		defer complete()
		_, _ = handle.Wait(context.Background())
	}()
	return nil
}

var _ testpilot.Profile = (*Driver)(nil)
var _ testpilot.Driver = (*Driver)(nil)
var _ testpilot.Session = (*compositeSession)(nil)
