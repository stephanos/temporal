// Package provision creates and removes the physical resources one Case's symbolic bindings
// resolve to: a namespace, and the Nexus endpoint that routes to its task queue.
//
// It reaches the server over the public workflow and operator services only. It takes no test
// environment, no *testing.T and no server-internal package, which is what lets a live test, a
// one-shot CLI and a future canary all provision the same way. Registering a namespace through
// the frontend is asynchronous in the namespace cache, so Create does not return until a
// DescribeNamespace call actually serves it.
package provision

import (
	"context"
	"errors"
	"fmt"
	"time"

	nexuspb "go.temporal.io/api/nexus/v1"
	"go.temporal.io/api/operatorservice/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/protobuf/types/known/durationpb"
)

// ErrInvalid rejects a request this package cannot act on, before any server call.
var ErrInvalid = errors.New("invalid Testpilot provisioning request")

// Clients are the two public services provisioning needs. Both are required.
type Clients struct {
	Workflow workflowservice.WorkflowServiceClient
	Operator operatorservice.OperatorServiceClient
}

// Resources names what to create. A Case that binds no Nexus endpoint leaves NexusEndpoint empty
// and none is created.
type Resources struct {
	Namespace     string
	TaskQueue     string
	NexusEndpoint string
	// Retention is the namespace's workflow execution retention. Zero takes DefaultRetention.
	Retention time.Duration
	// ReadyTimeout bounds the wait for the namespace cache to serve the new namespace. Zero takes
	// DefaultReadyTimeout.
	ReadyTimeout time.Duration
	// ReadyInterval is how often that wait polls. Zero takes DefaultReadyInterval.
	ReadyInterval time.Duration
	// RetainNamespace keeps the namespace after cleanup. Deleting one is a server-side workflow
	// that takes tens of seconds, and a cluster that is discarded wholesale buys nothing by
	// waiting for it. A caller running against a deployment that outlives the Run leaves this
	// false, so `--create` never leaves a namespace behind.
	RetainNamespace bool
}

const (
	DefaultRetention     = 24 * time.Hour
	DefaultReadyTimeout  = 30 * time.Second
	DefaultReadyInterval = 250 * time.Millisecond
)

// Cleanup releases what Create created, in reverse order. It is safe to call with a context whose
// deadline is shorter than the one Create used, and reports every resource it could not remove.
type Cleanup func(ctx context.Context) error

// Create registers the namespace, waits until it is served, and creates the Nexus endpoint when one
// is named. On any failure it releases whatever it had already created before returning, so a
// caller never has to clean up after an error.
func Create(ctx context.Context, clients Clients, resources Resources) (Cleanup, error) {
	if ctx == nil || clients.Workflow == nil || clients.Operator == nil {
		return nil, fmt.Errorf("%w: both service clients are required", ErrInvalid)
	}
	if resources.Namespace == "" {
		return nil, fmt.Errorf("%w: a namespace name is required", ErrInvalid)
	}
	if resources.NexusEndpoint != "" && resources.TaskQueue == "" {
		return nil, fmt.Errorf("%w: a Nexus endpoint needs the task queue it routes to", ErrInvalid)
	}

	var releases []release
	if err := registerNamespace(ctx, clients.Workflow, resources); err != nil {
		return nil, err
	}
	if !resources.RetainNamespace {
		releases = append(releases, release{
			what: "namespace " + resources.Namespace,
			release: func(ctx context.Context) error {
				_, err := clients.Operator.DeleteNamespace(ctx, &operatorservice.DeleteNamespaceRequest{
					Namespace: resources.Namespace,
				})
				return err
			},
		})
	}

	if resources.NexusEndpoint != "" {
		created, err := clients.Operator.CreateNexusEndpoint(ctx, &operatorservice.CreateNexusEndpointRequest{
			Spec: &nexuspb.EndpointSpec{
				Name: resources.NexusEndpoint,
				Target: &nexuspb.EndpointTarget{Variant: &nexuspb.EndpointTarget_Worker_{
					Worker: &nexuspb.EndpointTarget_Worker{
						Namespace: resources.Namespace, TaskQueue: resources.TaskQueue,
					},
				}},
			},
		})
		if err != nil {
			// Roll back on a context the failure cannot have cancelled: a caller must never be left
			// holding a half-built environment.
			return nil, errors.Join(fmt.Errorf("create Nexus endpoint %q: %w", resources.NexusEndpoint, err),
				runReleases(context.WithoutCancel(ctx), releases))
		}
		endpoint := created.GetEndpoint()
		releases = append(releases, release{
			what: "Nexus endpoint " + resources.NexusEndpoint,
			release: func(ctx context.Context) error {
				_, err := clients.Operator.DeleteNexusEndpoint(ctx, &operatorservice.DeleteNexusEndpointRequest{
					Id: endpoint.GetId(), Version: endpoint.GetVersion(),
				})
				return err
			},
		})
	}

	return func(ctx context.Context) error { return runReleases(ctx, releases) }, nil
}

type release struct {
	what    string
	release func(ctx context.Context) error
}

// runReleases releases in reverse creation order and keeps going after a failure, so one stuck
// resource never hides the others. Every failure names the resource it could not remove.
func runReleases(ctx context.Context, releases []release) error {
	var failures []error
	for index := len(releases) - 1; index >= 0; index-- {
		if err := releases[index].release(ctx); err != nil {
			failures = append(failures, fmt.Errorf("delete %s: %w", releases[index].what, err))
		}
	}
	return errors.Join(failures...)
}

func registerNamespace(ctx context.Context, workflow workflowservice.WorkflowServiceClient, resources Resources) error {
	retention := resources.Retention
	if retention <= 0 {
		retention = DefaultRetention
	}
	_, err := workflow.RegisterNamespace(ctx, &workflowservice.RegisterNamespaceRequest{
		Namespace:                        resources.Namespace,
		WorkflowExecutionRetentionPeriod: durationpb.New(retention),
	})
	if err != nil {
		return fmt.Errorf("register namespace %q: %w", resources.Namespace, err)
	}
	return awaitNamespace(ctx, workflow, resources)
}

// awaitNamespace polls until the namespace cache serves the namespace the frontend just recorded.
// A Case whose first RPC raced that refresh would fail as a missing namespace rather than as
// anything about the Case.
func awaitNamespace(ctx context.Context, workflow workflowservice.WorkflowServiceClient, resources Resources) error {
	timeout := resources.ReadyTimeout
	if timeout <= 0 {
		timeout = DefaultReadyTimeout
	}
	interval := resources.ReadyInterval
	if interval <= 0 {
		interval = DefaultReadyInterval
	}
	deadline, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	var last error
	for {
		_, last = workflow.DescribeNamespace(deadline, &workflowservice.DescribeNamespaceRequest{
			Namespace: resources.Namespace,
		})
		if last == nil {
			return nil
		}
		var missing *serviceerror.NamespaceNotFound
		if !errors.As(last, &missing) {
			return fmt.Errorf("describe namespace %q: %w", resources.Namespace, last)
		}
		select {
		case <-deadline.Done():
			return fmt.Errorf("namespace %q was not served within %s: %w", resources.Namespace, timeout, last)
		case <-ticker.C:
		}
	}
}
