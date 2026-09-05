// Package temporal composes controller transports and SDK workers behind one Umpire Host.
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
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire"
	"go.temporal.io/server/tools/umpire/temporal/internal/delivery"
	"go.temporal.io/server/tools/umpire/temporal/server"
	workerhost "go.temporal.io/server/tools/umpire/temporal/worker"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var ErrInvalid = errors.New("invalid composite Temporal Host input")

type Endpoint = server.Endpoint
type RoleBinding = workerhost.RoleBinding

type Options struct {
	Profile               umpire.ProfileSpec
	ServerEndpoints       map[string]Endpoint
	SystemCallbackBaseURL string
	HTTPClient            *http.Client
	SDKClient             client.Client
	Namespace             string
	WorkerRoleID          string
	TaskQueues            []RoleBinding
	NexusEndpoints        []RoleBinding
	WorkerStopTimeout     time.Duration
}

// Host keeps server transport authority and SDK worker authority in their owning packages.
type Host struct {
	profile    umpire.ProfileSpec
	controller *server.Host
	worker     *workerhost.Host
}

func New(options Options) (*Host, error) {
	controller, err := server.New(server.Options{
		Profile: options.Profile, Endpoints: options.ServerEndpoints, SystemCallbackBaseURL: options.SystemCallbackBaseURL, HTTPClient: options.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	workers, err := workerhost.New(workerhost.Options{
		Profile: options.Profile, Client: options.SDKClient, Namespace: options.Namespace,
		WorkerRoleID: options.WorkerRoleID, TaskQueues: options.TaskQueues,
		Endpoints: options.NexusEndpoints, WorkerStopTimeout: options.WorkerStopTimeout,
	})
	if err != nil {
		closeErr := controller.Close(context.Background())
		return nil, errors.Join(err, closeErr)
	}
	return &Host{profile: options.Profile.Snapshot(), controller: controller, worker: workers}, nil
}

func (h *Host) Snapshot() umpire.ProfileSpec {
	if h == nil {
		return umpire.ProfileSpec{}
	}
	return h.profile.Snapshot()
}

func (h *Host) Identity(ctx context.Context) (umpire.HostIdentity, error) {
	if h == nil || ctx == nil {
		return umpire.HostIdentity{}, ErrInvalid
	}
	return h.controller.Identity(ctx)
}

func (h *Host) Open(ctx context.Context, runID string, program umpire.PreparedProgram) (umpire.Session, error) {
	if h == nil || ctx == nil || runID == "" {
		return nil, ErrInvalid
	}
	controller, err := h.controller.OpenSession(ctx, runID, program)
	if err != nil {
		return nil, err
	}
	if !hasWorkerEntrypoint(program.Snapshot()) {
		return newPreparedCompositeSession(controller, nil, program), nil
	}
	bridge, err := controller.Bridge(ctx)
	if err != nil {
		return nil, errors.Join(err, controller.Close(context.Background()))
	}
	worker, err := h.worker.OpenSession(ctx, runID, program, workerhost.SessionOptions{
		Bridge: bridge,
		NewCompletionCapability: func(ctx context.Context, origin umpire.Coordinate, info workerhost.CompletionInfo) (umpire.OpaqueCapability, error) {
			return controller.NewCompletionCapability(ctx, origin, server.CompletionInfo{
				URL: info.URL, Header: info.Header, OperationToken: info.OperationToken, StartTime: info.StartTime,
			})
		},
		Diagnose:   controller.Diagnose,
		Quarantine: quarantineWorkerHandle,
	})
	if err != nil {
		return nil, errors.Join(err, controller.Close(context.Background()))
	}
	return newPreparedCompositeSession(controller, worker, program), nil
}

func hasWorkerEntrypoint(program *umpirespb.Program) bool {
	for _, entrypoint := range program.GetEntrypoints() {
		if entrypoint.GetContext() == umpirespb.ENTRYPOINT_CONTEXT_WORKFLOW ||
			entrypoint.GetContext() == umpirespb.ENTRYPOINT_CONTEXT_ACTIVITY ||
			entrypoint.GetContext() == umpirespb.ENTRYPOINT_CONTEXT_NEXUS_HANDLER {
			return true
		}
	}
	return false
}

func (h *Host) Close(ctx context.Context) error {
	if h == nil || ctx == nil {
		return ErrInvalid
	}
	return h.controller.Close(ctx)
}

type workerSession interface {
	Reserve(context.Context, umpire.ReservationRequest) ([]umpire.ReservationHandle, error)
	Quarantine(context.Context, umpire.EffectHandle) error
	Close(context.Context) error
	Diagnose(context.Context, string, *umpirespb.RunDiagnostic) error
}

type carrierSession interface {
	CreateCarrier(context.Context, umpire.Coordinate, umpire.ReservationCarrierPlan, workerhost.WorkflowBinding, []umpire.ReservationHandle) (*workerhost.Carrier, error)
}

type compositeSession struct {
	controller   umpire.Session
	worker       workerSession
	program      umpire.PreparedProgram
	carrierPlan  func(string, string) (umpire.ReservationCarrierPlan, bool)
	mu           sync.Mutex
	reservations map[umpire.Coordinate][]umpire.ReservationHandle
}

func newCompositeSession(controller umpire.Session, worker workerSession, program umpire.PreparedProgram) *compositeSession {
	return &compositeSession{controller: controller, worker: worker, program: program, reservations: make(map[umpire.Coordinate][]umpire.ReservationHandle)}
}

func newPreparedCompositeSession(controller umpire.Session, worker workerSession, program umpire.PreparedProgram) *compositeSession {
	session := newCompositeSession(controller, worker, program)
	session.carrierPlan = program.ReservationCarrier
	return session
}

func (s *compositeSession) Reserve(ctx context.Context, request umpire.ReservationRequest) ([]umpire.ReservationHandle, error) {
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

func (s *compositeSession) InvokeRPC(ctx context.Context, coordinate umpire.Coordinate, role string, method protoreflect.MethodDescriptor, request proto.Message) (umpire.EffectHandle, error) {
	var plan umpire.ReservationCarrierPlan
	var carrierRequired bool
	if s.carrierPlan != nil {
		plan, carrierRequired = s.carrierPlan(coordinate.EntrypointID, coordinate.InstructionID)
	}
	if !carrierRequired {
		return s.controller.InvokeRPC(ctx, coordinate, role, method, request)
	}
	worker, ok := s.worker.(carrierSession)
	if !ok || method == nil || request == nil {
		return nil, ErrInvalid
	}
	binding, err := workflowBinding(request)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	handles := append([]umpire.ReservationHandle(nil), s.reservations[coordinate]...)
	s.mu.Unlock()
	carrier, err := worker.CreateCarrier(ctx, coordinate, plan, binding, handles)
	if err != nil {
		return nil, err
	}
	maximum := s.program.Snapshot().GetLimits().GetMaxRequestBytes()
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
	cleanupTimeout := time.Duration(s.program.Snapshot().GetLimits().GetMaxCleanupDurationMilliseconds()) * time.Millisecond
	return &carrierEffect{EffectHandle: handle, carrier: carrier, cleanupTimeout: cleanupTimeout}, nil
}

func workflowBinding(request proto.Message) (workerhost.WorkflowBinding, error) {
	wire, err := proto.Marshal(request)
	if err != nil {
		return workerhost.WorkflowBinding{}, err
	}
	var start workflowservice.StartWorkflowExecutionRequest
	if err := proto.Unmarshal(wire, &start); err != nil {
		return workerhost.WorkflowBinding{}, ErrInvalid
	}
	if start.GetNamespace() == "" || start.GetWorkflowId() == "" || start.GetWorkflowType().GetName() == "" || start.GetTaskQueue().GetName() == "" {
		return workerhost.WorkflowBinding{}, ErrInvalid
	}
	return workerhost.WorkflowBinding{
		Namespace: start.GetNamespace(), WorkflowID: start.GetWorkflowId(),
		WorkflowType: start.GetWorkflowType().GetName(), TaskQueue: start.GetTaskQueue().GetName(),
	}, nil
}

func (s *compositeSession) CompleteNexusOperation(ctx context.Context, coordinate umpire.Coordinate, capability umpire.OpaqueCapability, value *umpirespb.Value) (umpire.EffectHandle, error) {
	return s.controller.CompleteNexusOperation(ctx, coordinate, capability, value)
}
func (s *compositeSession) Bridge(ctx context.Context) (umpire.CapabilityBridge, error) {
	return s.controller.Bridge(ctx)
}
func (s *compositeSession) Quarantine(ctx context.Context, handle umpire.EffectHandle) error {
	if carried, ok := handle.(*carrierEffect); ok {
		return s.controller.Quarantine(ctx, carried.EffectHandle)
	}
	if _, ok := handle.(umpire.ReservationHandle); ok && s.worker != nil {
		return s.worker.Quarantine(ctx, handle)
	}
	return s.controller.Quarantine(ctx, handle)
}
func (s *compositeSession) Close(ctx context.Context) error {
	var workerErr error
	if s.worker != nil {
		workerErr = s.worker.Close(ctx)
	}
	return errors.Join(workerErr, s.controller.Close(ctx))
}
func (s *compositeSession) Diagnose(ctx context.Context, runID string, diagnostic *umpirespb.RunDiagnostic) error {
	return s.controller.Diagnose(ctx, runID, diagnostic)
}

type carrierEffect struct {
	umpire.EffectHandle
	carrier        terminalCarrier
	cleanupTimeout time.Duration
	mu             sync.Mutex
	finished       bool
	disposition    delivery.TriggerDisposition
	response       *workflowservice.StartWorkflowExecutionResponse
}

func (e *carrierEffect) Wait(ctx context.Context) (umpire.EffectResult, error) {
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
	PinStartResponse(context.Context, *workflowservice.StartWorkflowExecutionResponse) error
	TriggerTerminal(context.Context, delivery.TriggerDisposition) (int, error)
}

func (e *carrierEffect) finish(result umpire.EffectResult, waitErr error) error {
	disposition := delivery.TriggerUncertain
	var response *workflowservice.StartWorkflowExecutionResponse
	var responseErr error
	if waitErr == nil && result.Outcome != nil {
		switch result.Outcome.GetStatus() {
		case umpirespb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED:
			response, responseErr = startResponse(result.Response)
			if responseErr == nil {
				disposition = delivery.TriggerSucceeded
			}
		case umpirespb.INSTRUCTION_OUTCOME_STATUS_CANCELED:
			disposition = delivery.TriggerCanceled
		default:
			disposition = delivery.TriggerNonSuccess
		}
	}
	return errors.Join(responseErr, e.finalize(disposition, response))
}

func (e *carrierEffect) trigger(disposition delivery.TriggerDisposition) error {
	return e.finalize(disposition, nil)
}

func (e *carrierEffect) finalize(disposition delivery.TriggerDisposition, response *workflowservice.StartWorkflowExecutionResponse) error {
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
		e.response = proto.CloneOf(response)
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

func startResponse(response proto.Message) (*workflowservice.StartWorkflowExecutionResponse, error) {
	if response == nil {
		return nil, ErrInvalid
	}
	wire, err := proto.Marshal(response)
	if err != nil {
		return nil, err
	}
	var start workflowservice.StartWorkflowExecutionResponse
	if err := proto.Unmarshal(wire, &start); err != nil || start.GetRunId() == "" {
		return nil, ErrInvalid
	}
	return &start, nil
}

func quarantineWorkerHandle(ctx context.Context, handle umpire.EffectHandle, complete func()) error {
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

var _ umpire.Profile = (*Host)(nil)
var _ umpire.Host = (*Host)(nil)
var _ umpire.Session = (*compositeSession)(nil)
