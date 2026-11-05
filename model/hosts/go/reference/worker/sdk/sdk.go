// Package sdk follows activity programs using one dynamic Go SDK activity handler.
package sdk

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	genericworker "github.com/temporalio/umpire/hosts/go/worker"
	failurepb "go.temporal.io/api/failure/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	sdklog "go.temporal.io/sdk/log"
	"go.temporal.io/sdk/temporal"
	sdkworker "go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
	"umpire.local/temporal/reference/worker"
)

const Kind = "sdk"
const Version = "go.temporal.io/sdk@v1.48.0"

type scenarioKey struct{}
type propagator struct{}

// ScenarioPropagator must be installed on conn to expose the task header to the handler.
func ScenarioPropagator() workflow.ContextPropagator                   { return propagator{} }
func (propagator) Inject(context.Context, workflow.HeaderWriter) error { return nil }
func (propagator) Extract(ctx context.Context, h workflow.HeaderReader) (context.Context, error) {
	p, _ := h.Get(worker.ScenarioHeader)
	return context.WithValue(ctx, scenarioKey{}, worker.Scenario(p)), nil
}
func (propagator) InjectFromWorkflow(workflow.Context, workflow.HeaderWriter) error { return nil }
func (propagator) ExtractToWorkflow(ctx workflow.Context, _ workflow.HeaderReader) (workflow.Context, error) {
	return ctx, nil
}

// Validate checks every SDK response conversion before startup.
func Validate(program []byte) error {
	table, err := worker.Decode(program)
	if err != nil {
		return err
	}
	return validateTable(table)
}

func validateTable(table *worker.Program) error {
	for _, row := range table.Rows() {
		if row.Response.MessageIs(&workflowservice.RecordActivityTaskHeartbeatRequest{}) {
			heartbeat := new(workflowservice.RecordActivityTaskHeartbeatRequest)
			if err := row.Response.UnmarshalTo(heartbeat); err != nil {
				return err
			}
			if err := sendable(heartbeat, "details"); err != nil {
				return err
			}
			if heartbeat.Details != nil && len(heartbeat.Details.Payloads) == 0 {
				return errors.New("SDK cannot preserve present empty heartbeat details")
			}
			continue
		}
		if row.Response.MessageIs(&workflowservice.RespondActivityTaskCanceledRequest{}) {
			notified := false
			for _, bit := range row.Key.GetHeartbeatHistory() {
				notified = notified || bit
			}
			if !notified {
				return errors.New("SDK canceled leaf requires an observed true heartbeat")
			}
		}
		if _, err := convert(row.Response); err != nil {
			return err
		}
	}
	return nil
}

type terminalExpectation struct {
	ticket   *worker.ActivityTicket
	deadline time.Time
}

// Follow joins its handlers, native worker and terminal coordinator before returning.
func Follow(ctx context.Context, conn client.Client, program []byte, bridge *worker.Bridge) error {
	return followWithOptions(ctx, conn, program, bridge, sdkworker.Options{})
}

func followWithOptions(ctx context.Context, conn client.Client, program []byte, bridge *worker.Bridge, options sdkworker.Options) error {
	table, err := worker.Decode(program)
	if err != nil {
		return err
	}
	if err := validateTable(table); err != nil {
		return err
	}
	if conn == nil || bridge == nil {
		return errors.New("SDK executor requires a connection and transport bridge")
	}
	run, cancel := context.WithCancel(ctx)
	defer cancel()
	bound, err := client.NewClientFromExistingWithContext(run, conn, client.Options{
		Namespace: table.Namespace, ContextPropagators: []workflow.ContextPropagator{ScenarioPropagator()},
		Logger: sdklog.NewStructuredLogger(slog.New(slog.DiscardHandler)),
	})
	if err != nil {
		return err
	}
	defer bound.Close()
	var stateMu sync.Mutex
	var failure error
	accepting := true
	var handlers sync.WaitGroup
	fail := func(err error) {
		if err == nil {
			return
		}
		stateMu.Lock()
		if failure == nil {
			failure = err
		}
		stateMu.Unlock()
		cancel()
	}
	capacity := len(table.Rows())
	if capacity == 0 {
		capacity = 1
	}
	terminals := make(chan terminalExpectation, capacity)
	coordinatorDone := make(chan struct{})
	go func() { defer close(coordinatorDone); coordinateTerminals(run, ctx, bridge, terminals, fail) }()
	options.DisableWorkflowWorker = true
	options.WorkerStopTimeout = time.Second
	options.BackgroundActivityContext = run
	w := sdkworker.New(bound, table.TaskQueue, options)
	w.RegisterDynamicActivity(func(activityCtx context.Context, _ converter.EncodedValues) (converter.RawValue, error) {
		stateMu.Lock()
		if !accepting {
			stateMu.Unlock()
			return converter.RawValue{}, run.Err()
		}
		handlers.Add(1)
		stateMu.Unlock()
		defer handlers.Done()
		result, err := followTask(run, activityCtx, table, bridge, terminals, fail)
		if err != nil {
			if !shutdownCancellation(run, ctx, err) {
				fail(err)
			}
			return converter.RawValue{}, err
		}
		return result.value, result.err
	}, activity.DynamicRegisterOptions{})
	if err := w.Start(); err != nil {
		cancel()
		close(terminals)
		<-coordinatorDone
		return err
	}
	select {
	case <-run.Done():
	case <-bridge.Failed():
		fail(bridge.Err())
	case <-bridge.Done():
		if run.Err() == nil {
			fail(errors.New("SDK transport bridge closed"))
		}
	}
	stateMu.Lock()
	accepting = false
	stateMu.Unlock()
	cancel()
	w.Stop()
	handlers.Wait()
	close(terminals)
	<-coordinatorDone
	stateMu.Lock()
	defer stateMu.Unlock()
	return failure
}

func shutdownCancellation(run, owner context.Context, err error) bool {
	return (run.Err() != nil || owner.Err() != nil) && cancellationOnly(err)
}

func cancellationOnly(err error) bool {
	if err == context.Canceled || err == context.DeadlineExceeded {
		return true
	}
	switch wrapped := err.(type) {
	case interface{ Unwrap() []error }:
		causes := wrapped.Unwrap()
		if len(causes) == 0 {
			return false
		}
		for _, cause := range causes {
			if !cancellationOnly(cause) {
				return false
			}
		}
		return true
	case interface{ Unwrap() error }:
		return cancellationOnly(wrapped.Unwrap())
	default:
		return false
	}
}

func coordinateTerminals(run, owner context.Context, bridge *worker.Bridge, terminals <-chan terminalExpectation, fail func(error)) {
	failed, closed := bridge.Failed(), bridge.Done()
	for {
		select {
		case terminal, ok := <-terminals:
			if !ok {
				return
			}
			terminalCtx, terminalCancel := context.WithDeadline(run, terminal.deadline)
			err := terminal.ticket.Wait(terminalCtx)
			terminalCancel()
			if err != nil && shutdownCancellation(run, owner, err) {
				continue
			}
			if err != nil {
				fail(fmt.Errorf("SDK terminal completion: %w", err))
			}
		case <-failed:
			fail(bridge.Err())
			failed = nil
		case <-closed:
			if run.Err() == nil && owner.Err() == nil {
				fail(errors.New("SDK transport bridge closed"))
			}
			closed = nil
		}
	}
}

type taskLifetime struct {
	ctx                     context.Context
	cancel                  context.CancelFunc
	runJoined, originalDone chan struct{}
	stopRun, stopOriginal   func() bool
	cause                   error
}

func newTaskLifetime(run, activityCtx context.Context, fail func(error)) (*taskLifetime, error) {
	deadline, ok := activityCtx.Deadline()
	if !ok {
		return nil, errors.New("SDK activity deadline is missing")
	}
	heartbeatCtx, cancel := context.WithDeadline(context.WithoutCancel(activityCtx), deadline)
	lifetime := &taskLifetime{ctx: heartbeatCtx, cancel: cancel, runJoined: make(chan struct{}), originalDone: make(chan struct{})}
	lifetime.stopRun = context.AfterFunc(run, func() { cancel(); close(lifetime.runJoined) })
	lifetime.stopOriginal = context.AfterFunc(activityCtx, func() {
		defer close(lifetime.originalDone)
		lifetime.cause = context.Cause(activityCtx)
		if !temporal.IsCanceledError(lifetime.cause) && run.Err() == nil {
			fail(fmt.Errorf("SDK activity cancellation without server heartbeat: %w", lifetime.cause))
		}
	})
	return lifetime, nil
}

func (l *taskLifetime) close() {
	l.cancel()
	if !l.stopRun() {
		<-l.runJoined
	}
	if !l.stopOriginal() {
		<-l.originalDone
	}
}

func followTask(run, activityCtx context.Context, table *worker.Program, bridge *worker.Bridge, terminals chan<- terminalExpectation, fail func(error)) (delivery, error) {
	info := activity.GetInfo(activityCtx)
	scenario, ok := activityCtx.Value(scenarioKey{}).(string)
	if !ok {
		return delivery{}, errors.New("SDK task scenario context is missing")
	}
	lifetime, err := newTaskLifetime(run, activityCtx, fail)
	if err != nil {
		return delivery{}, err
	}
	defer lifetime.close()
	session, err := table.NewSession(bridge, worker.TaskIdentity{
		Namespace: info.Namespace, TaskQueue: info.TaskQueue, ActivityID: info.ActivityID, ActivityRunID: info.ActivityRunID,
		WorkflowID: info.WorkflowExecution.ID, WorkflowRunID: info.WorkflowExecution.RunID,
		ActivityType: info.ActivityType.Name, Scenario: scenario, Attempt: info.Attempt, Token: info.TaskToken,
	}, "", func(ctx context.Context, bit bool) error {
		if !bit {
			return nil
		}
		return awaitSDKCancellation(ctx, bridge, lifetime.originalDone, &lifetime.cause)
	})
	if err != nil {
		return delivery{}, err
	}
	lifetime.ctx = session.Context(lifetime.ctx)
	heartbeatCtx := lifetime.ctx
	for {
		if err := heartbeatCtx.Err(); err != nil {
			return delivery{}, err
		}
		ticket, err := session.Begin()
		if err != nil {
			return delivery{}, err
		}
		if ticket.Role() == genericworker.Continue {
			if err := nativeHeartbeat(heartbeatCtx, ticket); err != nil {
				return delivery{}, err
			}
			continue
		}
		return enqueueTerminal(heartbeatCtx, bridge, ticket, terminals)
	}
}

func awaitSDKCancellation(ctx context.Context, bridge *worker.Bridge, canceled <-chan struct{}, cause *error) error {
	select {
	case <-canceled:
		if !temporal.IsCanceledError(*cause) {
			return errors.New("SDK true heartbeat lacks server cancellation cause")
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-bridge.Failed():
		return bridge.Err()
	case <-bridge.Done():
		return errors.New("SDK transport bridge closed")
	}
}

func enqueueTerminal(ctx context.Context, bridge *worker.Bridge, ticket *worker.ActivityTicket, terminals chan<- terminalExpectation) (delivery, error) {
	if ticket.Role() != genericworker.Finish {
		return delivery{}, errors.New("SDK terminal requires a Finish ticket")
	}
	result, err := convert(ticket.Template())
	if err != nil {
		return delivery{}, err
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		return delivery{}, errors.New("SDK terminal deadline is missing")
	}
	select {
	case terminals <- terminalExpectation{ticket: ticket, deadline: deadline}:
		return result, nil
	case <-ctx.Done():
		return delivery{}, ctx.Err()
	case <-bridge.Failed():
		return delivery{}, bridge.Err()
	case <-bridge.Done():
		return delivery{}, errors.New("SDK transport bridge closed")
	}
}

func nativeHeartbeat(ctx context.Context, ticket *worker.ActivityTicket) error {
	if ticket.Role() != genericworker.Continue {
		return errors.New("SDK heartbeat requires a Continue ticket")
	}
	heartbeat := new(workflowservice.RecordActivityTaskHeartbeatRequest)
	if err := ticket.Request().UnmarshalTo(heartbeat); err != nil {
		return err
	}
	details := make([]any, 0, len(heartbeat.GetDetails().GetPayloads()))
	for _, payload := range heartbeat.GetDetails().GetPayloads() {
		details = append(details, converter.NewRawValue(payload))
	}
	activity.RecordHeartbeat(ctx, details...)
	return ticket.Wait(ctx)
}

type delivery struct {
	value converter.RawValue
	err   error
}

func convert(response *anypb.Any) (delivery, error) {
	switch {
	case response.MessageIs(&workflowservice.RespondActivityTaskCompletedRequest{}):
		completed := new(workflowservice.RespondActivityTaskCompletedRequest)
		if err := response.UnmarshalTo(completed); err != nil {
			return delivery{}, err
		}
		if err := sendable(completed, "result"); err != nil {
			return delivery{}, err
		}
		payloads := completed.GetResult().GetPayloads()
		if len(payloads) != 1 {
			return delivery{}, fmt.Errorf("SDK returns one result payload, not %d", len(payloads))
		}
		return delivery{value: converter.NewRawValue(payloads[0])}, nil
	case response.MessageIs(&workflowservice.RespondActivityTaskFailedRequest{}):
		failed := new(workflowservice.RespondActivityTaskFailedRequest)
		if err := response.UnmarshalTo(failed); err != nil {
			return delivery{}, err
		}
		if err := sendable(failed, "failure"); err != nil {
			return delivery{}, err
		}
		if failed.GetFailure() == nil {
			return delivery{}, errors.New("failure response carries no failure")
		}
		converted := temporal.GetDefaultFailureConverter().FailureToError(failed.GetFailure())
		if temporal.IsCanceledError(converted) {
			return delivery{}, errors.New("SDK failed response cannot carry canceled failure")
		}
		return delivery{err: converted}, nil
	case response.MessageIs(&workflowservice.RespondActivityTaskCanceledRequest{}):
		canceled := new(workflowservice.RespondActivityTaskCanceledRequest)
		if err := response.UnmarshalTo(canceled); err != nil {
			return delivery{}, err
		}
		if err := sendable(canceled, "details"); err != nil {
			return delivery{}, err
		}
		failure := &failurepb.Failure{FailureInfo: &failurepb.Failure_CanceledFailureInfo{CanceledFailureInfo: &failurepb.CanceledFailureInfo{Details: canceled.GetDetails()}}}
		return delivery{err: temporal.GetDefaultFailureConverter().FailureToError(failure)}, nil
	default:
		return delivery{}, fmt.Errorf("SDK cannot send %q", response.GetTypeUrl())
	}
}
func sendable(request proto.Message, returned protoreflect.Name) error {
	var err error
	request.ProtoReflect().Range(func(fd protoreflect.FieldDescriptor, _ protoreflect.Value) bool {
		switch fd.Name() {
		case returned, "task_token", "namespace", "identity":
		default:
			err = fmt.Errorf("SDK cannot send response field %s", fd.Name())
		}
		return err == nil
	})
	return err
}
