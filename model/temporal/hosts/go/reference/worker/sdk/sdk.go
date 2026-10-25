// Package sdk follows activity programs using one dynamic Go SDK activity handler.
package sdk

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

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
	wait     *worker.TerminalWait
	expected *anypb.Any
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
	go func() { defer close(coordinatorDone); coordinateTerminals(run, bridge, terminals, fail) }()
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
			if run.Err() == nil {
				fail(err)
			}
			return converter.RawValue{}, err
		}
		return result.value, result.err
	}, activity.DynamicRegisterOptions{})
	if err := w.Start(); err != nil {
		cancel()
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
	<-coordinatorDone
	stateMu.Lock()
	defer stateMu.Unlock()
	return failure
}

func coordinateTerminals(run context.Context, bridge *worker.Bridge, terminals <-chan terminalExpectation, fail func(error)) {

	for {
		select {
		case terminal := <-terminals:
			terminalCtx, terminalCancel := context.WithDeadline(run, terminal.deadline)
			completion, err := terminal.wait.Wait(terminalCtx)
			terminalCancel()
			if err == nil {
				err = worker.TerminalResult(terminal.expected, completion.RPC())
			}
			if err != nil {
				if run.Err() == nil {
					fail(fmt.Errorf("SDK terminal completion: %w", err))
				}
				return
			}
		case <-run.Done():
			return
		case <-bridge.Failed():
			fail(bridge.Err())
			return
		case <-bridge.Done():
			if run.Err() == nil {
				fail(errors.New("SDK transport bridge closed"))
			}
			return
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

func newTaskLifetime(run, activityCtx context.Context, assignment *worker.Assignment, fail func(error)) (*taskLifetime, error) {
	deadline, ok := activityCtx.Deadline()
	if !ok {
		return nil, errors.New("SDK activity deadline is missing")
	}
	heartbeatCtx, cancel := context.WithDeadline(context.WithoutCancel(activityCtx), deadline)
	lifetime := &taskLifetime{ctx: assignment.HeartbeatContext(heartbeatCtx), cancel: cancel, runJoined: make(chan struct{}), originalDone: make(chan struct{})}
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
	assignment, err := bridge.Bind(worker.TaskIdentity{
		Namespace: info.Namespace, TaskQueue: info.TaskQueue, ActivityID: info.ActivityID, ActivityRunID: info.ActivityRunID,
		WorkflowID: info.WorkflowExecution.ID, WorkflowRunID: info.WorkflowExecution.RunID,
		ActivityType: info.ActivityType.Name, Scenario: scenario, Attempt: info.Attempt, Token: info.TaskToken,
	})
	if err != nil {
		return delivery{}, err
	}
	lifetime, err := newTaskLifetime(run, activityCtx, assignment, fail)
	if err != nil {
		return delivery{}, err
	}
	defer lifetime.close()
	heartbeatCtx := lifetime.ctx
	history := make([]bool, 0, table.MaxHeartbeats())
	for decisions := 0; decisions <= table.MaxHeartbeats(); decisions++ {
		if err := heartbeatCtx.Err(); err != nil {
			return delivery{}, err
		}
		response := table.Match(scenario, info.Attempt, info.ActivityType.Name, history...)
		if response == nil {
			if len(history) != 0 {
				return delivery{}, errors.New("SDK task has no exact observable-history row")
			}
			response, err = anypb.New(&workflowservice.RespondActivityTaskFailedRequest{Failure: worker.UnprogrammedFailure()})
			if err != nil {
				return delivery{}, err
			}
		}
		if response.MessageIs(&workflowservice.RecordActivityTaskHeartbeatRequest{}) {
			bit, err := nativeHeartbeat(heartbeatCtx, assignment, response, info.Namespace, info.TaskToken)
			if err != nil {
				return delivery{}, err
			}
			if bit {
				err = awaitSDKCancellation(heartbeatCtx, bridge, lifetime.originalDone, &lifetime.cause)
			}
			if err != nil {
				return delivery{}, err
			}
			history = append(history, bit)
			continue
		}
		return enqueueTerminal(heartbeatCtx, bridge, assignment, response, info.Namespace, info.TaskToken, terminals)
	}
	return delivery{}, errors.New("SDK task exhausted its admitted heartbeat depth")
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

func enqueueTerminal(ctx context.Context, bridge *worker.Bridge, assignment *worker.Assignment, response *anypb.Any, namespace string, token []byte, terminals chan<- terminalExpectation) (delivery, error) {
	result, err := convert(response)
	if err != nil {
		return delivery{}, err
	}
	expected, method, err := terminalRequest(response, namespace, token)
	if err != nil {
		return delivery{}, err
	}
	wait, err := assignment.BeginTerminal(method)
	if err != nil {
		return delivery{}, err
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		return delivery{}, errors.New("SDK terminal deadline is missing")
	}
	select {
	case terminals <- terminalExpectation{wait: wait, expected: expected, deadline: deadline}:
		return result, nil
	case <-ctx.Done():
		return delivery{}, ctx.Err()
	case <-bridge.Failed():
		return delivery{}, bridge.Err()
	case <-bridge.Done():
		return delivery{}, errors.New("SDK transport bridge closed")
	}
}

func nativeHeartbeat(ctx context.Context, assignment *worker.Assignment, response *anypb.Any, namespace string, token []byte) (bool, error) {
	heartbeat := new(workflowservice.RecordActivityTaskHeartbeatRequest)
	if err := response.UnmarshalTo(heartbeat); err != nil {
		return false, err
	}
	heartbeat.Namespace, heartbeat.TaskToken = namespace, append([]byte(nil), token...)
	expected, err := anypb.New(heartbeat)
	if err != nil {
		return false, err
	}
	wait, err := assignment.BeginHeartbeat()
	if err != nil {
		return false, err
	}
	details := make([]any, 0, len(heartbeat.GetDetails().GetPayloads()))
	for _, payload := range heartbeat.GetDetails().GetPayloads() {
		details = append(details, converter.NewRawValue(payload))
	}
	activity.RecordHeartbeat(ctx, details...)
	completion, err := wait.Wait(ctx)
	if err != nil {
		return false, err
	}
	return worker.HeartbeatResult(expected, completion.RPC())
}

func terminalRequest(response *anypb.Any, namespace string, token []byte) (*anypb.Any, string, error) {
	var message proto.Message
	var method string
	switch {
	case response.MessageIs(&workflowservice.RespondActivityTaskCompletedRequest{}):
		request := new(workflowservice.RespondActivityTaskCompletedRequest)
		if err := response.UnmarshalTo(request); err != nil {
			return nil, "", err
		}
		request.Namespace, request.TaskToken = namespace, append([]byte(nil), token...)
		message, method = request, workflowservice.WorkflowService_RespondActivityTaskCompleted_FullMethodName
	case response.MessageIs(&workflowservice.RespondActivityTaskFailedRequest{}):
		request := new(workflowservice.RespondActivityTaskFailedRequest)
		if err := response.UnmarshalTo(request); err != nil {
			return nil, "", err
		}
		request.Namespace, request.TaskToken = namespace, append([]byte(nil), token...)
		message, method = request, workflowservice.WorkflowService_RespondActivityTaskFailed_FullMethodName
	case response.MessageIs(&workflowservice.RespondActivityTaskCanceledRequest{}):
		request := new(workflowservice.RespondActivityTaskCanceledRequest)
		if err := response.UnmarshalTo(request); err != nil {
			return nil, "", err
		}
		request.Namespace, request.TaskToken = namespace, append([]byte(nil), token...)
		message, method = request, workflowservice.WorkflowService_RespondActivityTaskCanceled_FullMethodName
	default:
		return nil, "", errors.New("SDK terminal response kind is unsupported")
	}
	packed, err := anypb.New(message)
	return packed, method, err
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
