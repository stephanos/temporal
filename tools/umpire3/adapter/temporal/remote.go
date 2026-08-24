package temporal

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	sdkclient "go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/server/tools/umpire3/deployment"
	"go.temporal.io/server/tools/umpire3/execution"
	protocolcatalog "go.temporal.io/server/tools/umpire3/protocol/catalog"
	protocolexperiment "go.temporal.io/server/tools/umpire3/protocol/experiment"
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

func Execute(parent context.Context, experiment protocolexperiment.Experiment, options Options) (execution.Result, error) {
	deploymentProfile, err := Validate(options)
	if err != nil {
		return execution.Result{}, err
	}
	ctx, cancel := context.WithTimeout(parent, options.Timeout)
	defer cancel()
	clientOptions, err := ClientOptions(options.Address, options.Namespace,
		"umpire3/"+experiment.ExperimentID, options.APIKey)
	if err != nil {
		return execution.Result{}, err
	}
	client, err := sdkclient.Dial(clientOptions)
	if err != nil {
		return execution.Result{}, fmt.Errorf("dial Temporal: %w", err)
	}
	defer client.Close()
	sdkWorker := worker.New(client, options.TaskQueue, worker.Options{})
	factory, err := NewSDKFactory(SDKFactoryOptions{
		Client: client, Registry: sdkWorker, Deployment: deploymentProfile, Namespace: options.Namespace,
		TaskQueue: options.TaskQueue, CleanupTimeout: options.Timeout,
		WorkflowID:    func(experiment protocolexperiment.Experiment) string { return "umpire3-" + experiment.ExperimentID },
		NexusEndpoint: options.NexusEndpoint, NexusService: options.NexusService,
		NexusOperation: options.NexusOperation,
	})
	if err != nil {
		return execution.Result{}, err
	}
	bound, err := deployment.Bind(deploymentProfile, factory)
	if err != nil {
		return execution.Result{}, err
	}
	owned, err := OwnWorkerLifecycle(bound, sdkWorker, options.Timeout)
	if err != nil {
		return execution.Result{}, err
	}
	return execution.Run(ctx, execution.Request{
		Experiment: experiment, Environment: owned,
		AllowRestrictedFaults: options.AllowRestrictedFaults,
		Limits: execution.Limits{
			PrepareTimeout: options.Timeout, ActionTimeout: options.Timeout,
			ObserveTimeout: options.Timeout, FaultTimeout: options.Timeout,
			CleanupTimeout: options.Timeout,
		},
	})
}

type WorkerLifecycle interface {
	Start() error
	Stop()
}

type workerEnvironment struct {
	underlying     execution.Factory
	worker         WorkerLifecycle
	cleanupTimeout time.Duration
}

func OwnWorkerLifecycle(
	underlying execution.Factory,
	lifecycle WorkerLifecycle,
	cleanupTimeout time.Duration,
) (execution.Factory, error) {
	if underlying == nil || lifecycle == nil || cleanupTimeout <= 0 {
		return nil, errors.New("worker environment requires a factory, worker, and positive cleanup timeout")
	}
	return &workerEnvironment{underlying: underlying, worker: lifecycle, cleanupTimeout: cleanupTimeout}, nil
}

func (f *workerEnvironment) Capabilities() []protocolcatalog.CapabilityID {
	return f.underlying.Capabilities()
}

func (f *workerEnvironment) Prepare(
	ctx context.Context,
	experiment protocolexperiment.Experiment,
) (execution.PreparedEnvironment, error) {
	prepared, err := f.underlying.Prepare(ctx, experiment)
	if err != nil {
		if prepared.Session == nil {
			return prepared, err
		}
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), f.cleanupTimeout)
		cleanup := prepared.Session.Cleanup(cleanupCtx)
		cancel()
		if cleanup.Error != "" {
			return execution.PreparedEnvironment{}, errors.Join(err, errors.New(cleanup.Error))
		}
		return execution.PreparedEnvironment{}, err
	}
	if prepared.Session == nil {
		return prepared, err
	}
	if err := f.worker.Start(); err != nil {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), f.cleanupTimeout)
		cleanup := prepared.Session.Cleanup(cleanupCtx)
		cancel()
		startErr := fmt.Errorf("start SDK worker: %w", err)
		if cleanup.Error != "" {
			return execution.PreparedEnvironment{}, errors.Join(startErr, errors.New(cleanup.Error))
		}
		return execution.PreparedEnvironment{}, startErr
	}
	prepared.Session = &workerSession{Session: prepared.Session, worker: f.worker}
	return prepared, nil
}

type workerSession struct {
	execution.Session
	worker  WorkerLifecycle
	cleanup sync.Once
	result  execution.CleanupResult
}

func (s *workerSession) Cleanup(ctx context.Context) execution.CleanupResult {
	s.cleanup.Do(func() {
		s.result = s.Session.Cleanup(ctx)
		s.worker.Stop()
	})
	return s.result
}

func Validate(options Options) (deployment.Profile, error) {
	if options.Address == "" || options.Namespace == "" || options.TaskQueue == "" ||
		options.BuildID == "" || options.Timeout <= 0 {
		return deployment.Profile{}, errors.New("address, namespace, task queue, build, and positive timeout are required")
	}
	nexusValues := 0
	for _, value := range []string{options.NexusEndpoint, options.NexusService, options.NexusOperation} {
		if value != "" {
			nexusValues++
		}
	}
	if nexusValues != 0 && nexusValues != 3 {
		return deployment.Profile{}, errors.New("nexus endpoint, service, and operation must be supplied together")
	}
	var spec deployment.Spec
	switch options.Profile {
	case "local-in-process":
		spec = deployment.Local(options.BuildID, options.Namespace, options.TaskQueue)
	case "ci-test-cluster":
		spec = deployment.CI(options.BuildID, options.Namespace, options.TaskQueue)
	case "remote-deployment", "grpc-only-black-box":
		endpoint := options.Address
		if !strings.Contains(endpoint, "://") {
			endpoint = "https://" + endpoint
		}
		if options.Profile == "remote-deployment" {
			spec = deployment.Remote(endpoint, options.APIKey, options.BuildID, options.Namespace, options.TaskQueue)
		} else {
			spec = deployment.BlackBox(endpoint, options.APIKey, options.BuildID, options.Namespace, options.TaskQueue)
		}
	default:
		return deployment.Profile{}, fmt.Errorf("unsupported execution profile %q", options.Profile)
	}
	return deployment.Define(spec)
}
