package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"go.temporal.io/api/serviceerror"
	sdkclient "go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/server/tests/umpire3/canary"
	"go.temporal.io/server/tests/umpire3/environment"
	"go.temporal.io/server/tests/umpire3/protocol"
	umpire3runtime "go.temporal.io/server/tests/umpire3/runtime"
	umpire3temporal "go.temporal.io/server/tests/umpire3/temporal"
)

const maxRequestBytes = 16 << 20

func main() {
	if err := run(context.Background(), os.Stdin, os.Stdout); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, input io.Reader, output io.Writer) error {
	request, err := decodeRequest(input)
	if err != nil {
		return err
	}
	clientOptions, err := umpire3temporal.ClientOptions(request.Profile.Endpoint, request.Profile.Namespace,
		"umpire3-canary/"+request.Approval.Identifier, os.Getenv("UMPIRE3_TEMPORAL_API_KEY"))
	if err != nil {
		return err
	}
	client, err := sdkclient.Dial(clientOptions)
	if err != nil {
		return fmt.Errorf("dial Temporal: %w", err)
	}
	defer client.Close()
	var response canary.WorkerResponse
	switch request.Operation {
	case canary.OperationExecute:
		response, err = execute(ctx, client, request)
	case canary.OperationCleanup:
		response, err = cleanup(ctx, client, request)
	default:
		err = fmt.Errorf("unknown canary worker operation %q", request.Operation)
	}
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("encode canary worker response: %w", err)
	}
	_, err = output.Write(append(encoded, '\n'))
	return err
}

func execute(
	ctx context.Context,
	client sdkclient.Client,
	request canary.WorkerRequest,
) (canary.WorkerResponse, error) {
	if len(request.Experiment.Faults) != 0 {
		return canary.WorkerResponse{}, errors.New("production canary fault realization is not configured")
	}
	sdkWorker := worker.New(client, request.Profile.TaskQueue, worker.Options{})
	workflowID, err := canaryWorkflowID(request.Experiment)
	if err != nil {
		return canary.WorkerResponse{}, err
	}
	factory, err := umpire3temporal.NewSDKFactory(umpire3temporal.SDKFactoryOptions{
		Client: client, Registry: sdkWorker, Namespace: request.Profile.Namespace,
		TaskQueue: request.Profile.TaskQueue, BuildID: request.Profile.Environment.BuildID,
		ConfigurationIdentity: request.Profile.Environment.ConfigurationIdentity,
		ProfileName:           "production-canary", EvidenceProfile: environment.EvidenceProfilePublicGRPCHistory,
		DrivingAuthority:     "approved-production-worker",
		ObservationAuthority: "production-public-history",
		FaultAuthority:       "none", HardExecutionBudget: true,
		CleanupTimeout: request.Approval.CleanupTimeout,
		WorkflowID:     func(protocol.Experiment) string { return workflowID },
	})
	if err != nil {
		return canary.WorkerResponse{}, err
	}
	session, err := factory.Prepare(ctx, request.Experiment)
	if err != nil {
		return canary.WorkerResponse{}, err
	}
	if err := sdkWorker.Start(); err != nil {
		cleanupResult := session.Cleanup(ctx)
		startErr := fmt.Errorf("start canary SDK worker: %w", err)
		if cleanupResult.Error != "" {
			return canary.WorkerResponse{}, errors.Join(startErr, errors.New(cleanupResult.Error))
		}
		return canary.WorkerResponse{}, startErr
	}
	defer sdkWorker.Stop()
	prepared, err := environment.PrepareOnce(factory.Capabilities(), session)
	if err != nil {
		return canary.WorkerResponse{}, err
	}
	result, err := umpire3runtime.Run(ctx, umpire3runtime.Request{
		Experiment: request.Experiment, Environment: prepared,
		Limits: umpire3runtime.Limits{
			PrepareTimeout: request.Approval.MaxDuration,
			ActionTimeout:  request.Approval.MaxDuration, ObserveTimeout: request.Approval.MaxDuration,
			CleanupTimeout: request.Approval.CleanupTimeout,
			MaxActions:     request.Approval.MaxActions, MaxEvidenceBytes: request.Approval.MaxEvidenceBytes,
		},
	})
	if err != nil {
		return canary.WorkerResponse{}, err
	}
	return canary.WorkerResponse{
		FormatVersion: canary.FormatVersion, Result: result,
		Resources:       map[string]string{"workflow": workflowID, "namespace": request.Profile.Namespace},
		CleanupComplete: result.Cleanup.Complete,
	}, nil
}

func cleanup(
	ctx context.Context,
	client sdkclient.Client,
	request canary.WorkerRequest,
) (canary.WorkerResponse, error) {
	workflowID := request.Recovery.Resources["workflow"]
	if workflowID == "" {
		var err error
		workflowID, err = canaryWorkflowID(request.Experiment)
		if err != nil {
			return canary.WorkerResponse{}, err
		}
	}
	err := client.TerminateWorkflow(ctx, workflowID, "", "umpire3 canary cleanup")
	if err != nil {
		var notFound *serviceerror.NotFound
		var failedPrecondition *serviceerror.FailedPrecondition
		if !errors.As(err, &notFound) && !errors.As(err, &failedPrecondition) {
			return canary.WorkerResponse{}, fmt.Errorf("terminate canary workflow: %w", err)
		}
	}
	return canary.WorkerResponse{
		FormatVersion:   canary.FormatVersion,
		Resources:       map[string]string{"workflow": workflowID, "namespace": request.Profile.Namespace},
		CleanupComplete: true,
	}, nil
}

func decodeRequest(input io.Reader) (canary.WorkerRequest, error) {
	encoded, err := io.ReadAll(io.LimitReader(input, maxRequestBytes+1))
	if err != nil {
		return canary.WorkerRequest{}, fmt.Errorf("read canary worker request: %w", err)
	}
	if len(encoded) > maxRequestBytes {
		return canary.WorkerRequest{}, errors.New("canary worker request exceeds input budget")
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var request canary.WorkerRequest
	if err := decoder.Decode(&request); err != nil {
		return canary.WorkerRequest{}, fmt.Errorf("decode canary worker request: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return canary.WorkerRequest{}, errors.New("canary worker request must contain one JSON document")
	}
	if request.FormatVersion != canary.FormatVersion || request.Operation == "" ||
		request.Profile.Namespace == "" || request.Profile.TaskQueue == "" ||
		request.Profile.Endpoint == "" || request.Approval.Identifier == "" {
		return canary.WorkerRequest{}, errors.New("canary worker request is incomplete")
	}
	return request, nil
}

func canaryWorkflowID(experiment protocol.Experiment) (string, error) {
	digest, err := experiment.Digest()
	if err != nil {
		return "", err
	}
	return "umpire3-canary-" + strings.TrimPrefix(digest, "sha256:")[:32], nil
}
