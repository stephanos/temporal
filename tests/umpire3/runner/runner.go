package runner

import (
	"context"
	"errors"
	"fmt"
	"time"

	sdkclient "go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/server/tests/umpire3/environment"
	"go.temporal.io/server/tests/umpire3/protocol"
	umpire3runtime "go.temporal.io/server/tests/umpire3/runtime"
	umpire3temporal "go.temporal.io/server/tests/umpire3/temporal"
)

type Options struct {
	Address               string
	Namespace             string
	TaskQueue             string
	BuildID               string
	Profile               string
	NexusEndpoint         string
	NexusService          string
	NexusOperation        string
	APIKey                string
	Timeout               time.Duration
	AllowRestrictedFaults bool
}

func Execute(parent context.Context, experiment protocol.Experiment, options Options) (umpire3runtime.Result, error) {
	profile, err := Validate(options)
	if err != nil {
		return umpire3runtime.Result{}, err
	}
	ctx, cancel := context.WithTimeout(parent, options.Timeout)
	defer cancel()
	clientOptions, err := umpire3temporal.ClientOptions(options.Address, options.Namespace,
		"umpire3-run/"+experiment.ExperimentID, options.APIKey)
	if err != nil {
		return umpire3runtime.Result{}, err
	}
	client, err := sdkclient.Dial(clientOptions)
	if err != nil {
		return umpire3runtime.Result{}, fmt.Errorf("dial Temporal: %w", err)
	}
	defer client.Close()
	sdkWorker := worker.New(client, options.TaskQueue, worker.Options{})
	factory, err := umpire3temporal.NewSDKFactory(umpire3temporal.SDKFactoryOptions{
		Client: client, Registry: sdkWorker, Namespace: options.Namespace,
		TaskQueue: options.TaskQueue, BuildID: options.BuildID,
		ProfileName: options.Profile, EvidenceProfile: profile.EvidenceProfile,
		DrivingAuthority: profile.DrivingAuthority, ObservationAuthority: profile.ObservationAuthority,
		FaultAuthority: profile.FaultAuthority, CleanupTimeout: options.Timeout,
		WorkflowID:    func(experiment protocol.Experiment) string { return "umpire3-" + experiment.ExperimentID },
		NexusEndpoint: options.NexusEndpoint, NexusService: options.NexusService,
		NexusOperation: options.NexusOperation,
	})
	if err != nil {
		return umpire3runtime.Result{}, err
	}
	session, err := factory.Prepare(ctx, experiment)
	if err != nil {
		return umpire3runtime.Result{}, err
	}
	if err := sdkWorker.Start(); err != nil {
		cleanup := session.Cleanup(ctx)
		startErr := fmt.Errorf("start SDK worker: %w", err)
		if cleanup.Error != "" {
			return umpire3runtime.Result{}, errors.Join(startErr, errors.New(cleanup.Error))
		}
		return umpire3runtime.Result{}, startErr
	}
	defer sdkWorker.Stop()
	prepared, err := environment.PrepareOnce(factory.Capabilities(), session)
	if err != nil {
		return umpire3runtime.Result{}, err
	}
	return umpire3runtime.Run(ctx, umpire3runtime.Request{
		Experiment: experiment, Environment: prepared,
		AllowRestrictedFaults: options.AllowRestrictedFaults,
		Limits: umpire3runtime.Limits{
			PrepareTimeout: options.Timeout, ActionTimeout: options.Timeout,
			ObserveTimeout: options.Timeout, FaultTimeout: options.Timeout,
			CleanupTimeout: options.Timeout,
		},
	})
}

type Profile struct {
	EvidenceProfile      string
	DrivingAuthority     string
	ObservationAuthority string
	FaultAuthority       string
}

func Validate(options Options) (Profile, error) {
	if options.Address == "" || options.Namespace == "" || options.TaskQueue == "" ||
		options.BuildID == "" || options.Timeout <= 0 {
		return Profile{}, errors.New("address, namespace, task queue, build, and positive timeout are required")
	}
	nexusValues := 0
	for _, value := range []string{options.NexusEndpoint, options.NexusService, options.NexusOperation} {
		if value != "" {
			nexusValues++
		}
	}
	if nexusValues != 0 && nexusValues != 3 {
		return Profile{}, errors.New("nexus endpoint, service, and operation must be supplied together")
	}
	switch options.Profile {
	case "local-in-process":
		return Profile{
			EvidenceProfile:  environment.EvidenceProfilePublicGRPCHistory,
			DrivingAuthority: "local-sdk", ObservationAuthority: "local-public-history",
			FaultAuthority: "none",
		}, nil
	case "ci-test-cluster":
		return Profile{
			EvidenceProfile:  environment.EvidenceProfilePublicGRPCHistory,
			DrivingAuthority: "ci-test-cluster", ObservationAuthority: "ci-public-history",
			FaultAuthority: "none",
		}, nil
	case "remote-deployment":
		return Profile{
			EvidenceProfile:  environment.EvidenceProfilePublicGRPCHistory,
			DrivingAuthority: "remote-api", ObservationAuthority: "remote-public-history",
			FaultAuthority: "none",
		}, nil
	case "grpc-only-black-box":
		return Profile{
			EvidenceProfile:  environment.EvidenceProfilePublicGRPC,
			DrivingAuthority: "public-grpc", ObservationAuthority: "public-grpc",
			FaultAuthority: "none",
		}, nil
	default:
		return Profile{}, fmt.Errorf("unsupported execution profile %q", options.Profile)
	}
}
