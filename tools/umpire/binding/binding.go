// Package binding binds Cases to one Temporal deployment the way `umpire-run` does, split into
// what a campaign opens once and what each candidate opens for itself.
//
// A campaign binding dials the frontend, provisions the named resources when asked, and builds the
// method catalog; it is released once, after the last candidate. A candidate binding derives the
// Profile one Case implies, prepares the Case's unchanged bytes, and opens one composite Driver
// with its own SDK worker; it is released after that Case's Run. Preparation is decided before any
// Driver opens, so a rejected Case creates no Run and holds no resource. Nothing here enters the
// Case: addresses and credentials stay outside the bytes, and no option widens a declared Limit.
package binding

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.temporal.io/api/operatorservice/v1"
	"go.temporal.io/api/workflowservice/v1"
	sdkclient "go.temporal.io/sdk/client"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	testpilotdriver "go.temporal.io/server/common/testing/testpilot/temporal"
	"go.temporal.io/server/common/testing/testpilot/temporal/provision"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	// TeardownTimeout bounds each released resource on its own, so a worker that will not stop
	// cannot starve the namespace deletion that follows it.
	TeardownTimeout   = 30 * time.Second
	WorkerStopTimeout = 10 * time.Second
	WorkflowRole      = "temporal.workflow-service"
	WorkerRole        = "temporal.worker"
)

// Deployment is what the caller names: where the frontend is, which resources a Case binds to,
// and whether to create them. HandlerTaskQueue is the queue a Case's Nexus handler polls when the
// Case binds one apart from the caller's; empty derives `<task-queue>-handler` for such a Case.
type Deployment struct {
	GRPCAddress      string
	HTTPAddress      string
	Namespace        string
	TaskQueue        string
	NexusEndpoint    string
	HandlerTaskQueue string
	Create           bool
}

// HandlerQueueFor is the queue a Case's Nexus handler polls under this deployment: the named one,
// or `<task-queue>-handler`, when the Program binds a handler queue apart from the workflow's, and
// "" when it does not.
func HandlerQueueFor(deployment Deployment, program *testpilotspb.Program) string {
	if testpilotdriver.HandlerTaskQueueBindingID(program) == "" {
		return ""
	}
	return handlerQueue(deployment)
}

func handlerQueue(deployment Deployment) string {
	if deployment.HandlerTaskQueue != "" {
		return deployment.HandlerTaskQueue
	}
	return deployment.TaskQueue + "-handler"
}

// Campaign is one deployment bound for a sequence of Cases: the connection, the provisioned
// resources and the catalog every candidate prepares against.
type Campaign struct {
	deployment Deployment
	connection *grpc.ClientConn
	catalog    *testpilot.Catalog
	releases   []func(context.Context) error
}

// Open binds the deployment once. handlerQueue is the queue the Nexus endpoint routes to when the
// Cases' handlers poll one of their own ("" routes it to the task queue); a caller binding one Case
// takes it from HandlerQueueFor, a campaign names it once for every candidate. With Create the
// named resources are registered and released with the campaign. A binding that fails rolls back
// what it had opened before returning.
func Open(ctx context.Context, deployment Deployment, handlerQueue string) (*Campaign, error) {
	connection, err := grpc.NewClient(deployment.GRPCAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial %q: %w", deployment.GRPCAddress, err)
	}
	campaign := &Campaign{
		deployment: deployment,
		connection: connection,
		releases:   []func(context.Context) error{func(context.Context) error { return connection.Close() }},
	}
	// A binding that failed rolls back on a context the failure cannot have cancelled.
	fail := func(err error) (*Campaign, error) {
		return nil, errors.Join(err, ReleaseAll(context.WithoutCancel(ctx), campaign.releases))
	}
	if deployment.Create {
		cleanup, err := provision.Create(ctx, provision.Clients{
			Workflow: workflowservice.NewWorkflowServiceClient(connection),
			Operator: operatorservice.NewOperatorServiceClient(connection),
		}, provision.Resources{
			Namespace:      deployment.Namespace,
			TaskQueue:      deployment.TaskQueue,
			NexusEndpoint:  deployment.NexusEndpoint,
			NexusTaskQueue: handlerQueue,
		})
		if err != nil {
			return fail(err)
		}
		campaign.releases = append(campaign.releases, cleanup)
	}
	catalog, err := testpilotdriver.NewWorkflowServiceCatalog()
	if err != nil {
		return fail(fmt.Errorf("build method catalog: %w", err))
	}
	campaign.catalog = catalog
	return campaign, nil
}

// Catalog is the method catalog every candidate prepares against.
func (c *Campaign) Catalog() *testpilot.Catalog { return c.catalog }

// Deployment is what the campaign was bound to.
func (c *Campaign) Deployment() Deployment { return c.deployment }

// Close releases what Open opened: the provisioned resources, then the connection. It names every
// resource it could not remove.
func (c *Campaign) Close(ctx context.Context) error {
	if c == nil {
		return nil
	}
	return ReleaseAll(ctx, c.releases)
}

// Bound is one prepared Case with the Driver that runs it.
type Bound struct {
	prepared *testpilot.PreparedCase
	driver   *testpilotdriver.Driver
	releases []func(context.Context) error
}

// Bind prepares one Case under this deployment and opens its Driver. identity names the Profile
// the Case runs under, which the prepared Case's Driver identity carries. A Case the deployment
// cannot bind rejects before any Driver opens: a Profile that cannot be derived, or a
// *testpilot.PreparationError, holds nothing and creates no Run.
func (c *Campaign) Bind(ctx context.Context, identity string, source *testpilotspb.Case) (*Bound, error) {
	if c == nil || source == nil {
		return nil, errors.New("campaign binding and Case are required")
	}
	profile, err := testpilotdriver.DeriveProfile(source, c.catalog, testpilotdriver.Environment{
		Identity:         identity,
		Namespace:        c.deployment.Namespace,
		TaskQueue:        c.deployment.TaskQueue,
		HandlerTaskQueue: HandlerQueueFor(c.deployment, source.GetProgram()),
		NexusEndpoint:    c.deployment.NexusEndpoint,
	})
	if err != nil {
		return nil, fmt.Errorf("derive Profile for Case %q: %w", source.GetCaseId(), err)
	}
	prepared, err := testpilot.Prepare(source, profile)
	if err != nil {
		return nil, err
	}
	bound := &Bound{prepared: prepared}
	fail := func(err error) (*Bound, error) {
		return nil, errors.Join(err, ReleaseAll(context.WithoutCancel(ctx), bound.releases))
	}
	caseClient, err := sdkclient.Dial(sdkclient.Options{
		HostPort: c.deployment.GRPCAddress, Namespace: c.deployment.Namespace,
	})
	if err != nil {
		return fail(fmt.Errorf("open SDK client: %w", err))
	}
	bound.releases = append(bound.releases, func(context.Context) error { caseClient.Close(); return nil })
	driver, err := testpilotdriver.New(testpilotdriver.Options{
		Profile: profile,
		ServerEndpoints: map[string]testpilotdriver.Endpoint{
			WorkflowRole: {Target: c.deployment.GRPCAddress, Credentials: insecure.NewCredentials()},
		},
		SystemCallbackBaseURL: "http://" + c.deployment.HTTPAddress,
		SDKClient:             caseClient,
		WorkerRoleID:          WorkerRole,
		WorkerStopTimeout:     WorkerStopTimeout,
	})
	if err != nil {
		return fail(fmt.Errorf("open Driver: %w", err))
	}
	bound.driver = driver
	bound.releases = append(bound.releases, driver.Close)
	return bound, nil
}

// Prepared is the admitted Case.
func (b *Bound) Prepared() *testpilot.PreparedCase { return b.prepared }

// Run executes the prepared Case once against the Driver the binding opened. The Run it returns
// has observed its cleanup.
func (b *Bound) Run(ctx context.Context) (*testpilotspb.Run, *testpilotspb.Verdict, error) {
	return b.prepared.Run(ctx, b.driver)
}

// Release is best-effort teardown of the Driver and the SDK client, run whatever the Run did. It
// names every resource it could not remove.
func (b *Bound) Release(ctx context.Context) error {
	if b == nil {
		return nil
	}
	return ReleaseAll(ctx, b.releases)
}

// ReleaseAll releases in reverse order and keeps going after a failure, so one stuck resource never
// hides the others. Each release gets its own budget for the same reason.
func ReleaseAll(ctx context.Context, releases []func(context.Context) error) error {
	var failures []error
	for index := len(releases) - 1; index >= 0; index-- {
		each, cancel := context.WithTimeout(ctx, TeardownTimeout)
		err := releases[index](each)
		cancel()
		if err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}

// IsPreparationRejection reports whether a Bind error is the Case's own static rejection, which
// created no Run, rather than a deployment or Driver failure.
func IsPreparationRejection(err error) (*testpilot.PreparationError, bool) {
	var rejection *testpilot.PreparationError
	if errors.As(err, &rejection) {
		return rejection, true
	}
	return nil, false
}
