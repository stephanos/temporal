// Package sdk follows activity programs using one dynamic Go SDK activity handler.
package sdk

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

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
	for _, response := range table.Responses() {
		if _, err := convert(response); err != nil {
			return err
		}
	}
	return nil
}

// Follow starts one dynamic activity handler and joins it before returning on cancellation.
// conn remains caller-owned; the program selects the namespace on its shared transport.
func Follow(ctx context.Context, conn client.Client, program []byte) error {
	table, err := worker.Decode(program)
	if err != nil {
		return err
	}
	for _, response := range table.Responses() {
		if _, err := convert(response); err != nil {
			return err
		}
	}
	if conn == nil {
		return errors.New("sdk executor requires a connection")
	}
	bound, err := client.NewClientFromExistingWithContext(ctx, conn, client.Options{
		Namespace: table.Namespace, ContextPropagators: []workflow.ContextPropagator{ScenarioPropagator()},
		Logger: sdklog.NewStructuredLogger(slog.New(slog.DiscardHandler)),
	})
	if err != nil {
		return err
	}
	defer bound.Close()
	w := sdkworker.New(bound, table.TaskQueue, sdkworker.Options{DisableWorkflowWorker: true, WorkerStopTimeout: time.Second, BackgroundActivityContext: ctx})
	w.RegisterDynamicActivity(func(ctx context.Context, _ converter.EncodedValues) (converter.RawValue, error) {
		info := activity.GetInfo(ctx)
		scenario, _ := ctx.Value(scenarioKey{}).(string)
		response := table.Match(scenario, info.Attempt, info.ActivityType.Name)
		if response == nil {
			return converter.RawValue{}, temporal.GetDefaultFailureConverter().FailureToError(worker.UnprogrammedFailure())
		}
		got, err := convert(response)
		if err != nil {
			return converter.RawValue{}, err
		}
		return got.value, got.err
	}, activity.DynamicRegisterOptions{})
	if err := w.Start(); err != nil {
		return err
	}
	<-ctx.Done()
	w.Stop()
	return nil
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
		return delivery{err: temporal.GetDefaultFailureConverter().FailureToError(failed.GetFailure())}, nil
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
