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
	"log/slog"
	"os"
	"time"

	"go.temporal.io/api/operatorservice/v1"
	"go.temporal.io/api/workflowservice/v1"
	sdkclient "go.temporal.io/sdk/client"
	sdklog "go.temporal.io/sdk/log"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	testpilotdriver "go.temporal.io/server/common/testing/testpilot/temporal"
	"go.temporal.io/server/common/testing/testpilot/temporal/provision"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	// teardownTimeout bounds each released resource on its own, so a worker that will not stop
	// cannot starve the namespace deletion that follows it.
	teardownTimeout   = 30 * time.Second
	workerStopTimeout = 10 * time.Second
	workflowRole      = "temporal.workflow-service"
	workerRole        = "temporal.worker"
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

// HandlerQueue is the queue a Nexus handler polls under this deployment when a Case binds one
// apart from the caller's: the named one, or `<task-queue>-handler`.
func HandlerQueue(deployment Deployment) string {
	if deployment.HandlerTaskQueue != "" {
		return deployment.HandlerTaskQueue
	}
	return deployment.TaskQueue + "-handler"
}

// HandlerQueueFor is the queue one Case's Nexus handler polls: HandlerQueue when the Program binds
// a handler queue apart from the workflow's, and "" when it does not.
func HandlerQueueFor(deployment Deployment, program *testpilotspb.Program) string {
	if testpilotdriver.HandlerTaskQueueBindingID(program) == "" {
		return ""
	}
	return HandlerQueue(deployment)
}

// Campaign is one deployment bound for a sequence of Cases: the connection, the provisioned
// resources and the catalog every candidate prepares against.
type Campaign struct {
	deployment   Deployment
	handlerQueue string
	connection   *grpc.ClientConn
	catalog      *testpilot.Catalog
	releases     []func(context.Context) error
}

// Open binds the deployment once. handlerQueue is the queue the Nexus endpoint routes to and every
// Case's handler polls when a Case binds a handler queue of its own ("" routes the endpoint to
// the task queue and binds no handler queue); a caller binding one Case takes it from
// HandlerQueueFor, a campaign takes HandlerQueue once for every candidate. With Create the named
// resources are registered and released with the campaign. A binding that fails rolls back what
// it had opened before returning.
func Open(ctx context.Context, deployment Deployment, handlerQueue string) (*Campaign, error) {
	connection, err := grpc.NewClient(deployment.GRPCAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial %q: %w", deployment.GRPCAddress, err)
	}
	campaign := &Campaign{
		deployment:   deployment,
		handlerQueue: handlerQueue,
		connection:   connection,
		releases:     []func(context.Context) error{func(context.Context) error { return connection.Close() }},
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

// Prepared is the prepared Case the binding runs.
func (b *Bound) Prepared() *testpilot.PreparedCase { return b.prepared }

// Identity is the Profile identity the prepared Case runs under.
func (b *Bound) Identity() testpilot.DriverIdentity { return b.prepared.Identity() }

// Prepared is one Case prepared under a deployment's names with no connection to it: what
// admission and offline replay need, and what Bind opens a Driver for.
type Prepared struct {
	Case    *testpilot.PreparedCase
	Profile testpilot.ProfileSpec
}

// Prepare derives the Profile one Case implies under the deployment's names and prepares it,
// touching no deployment: it builds the method catalog itself, opens no connection, provisions
// nothing and opens no Driver. identity names the Profile the Case is prepared under, which the
// prepared Case's Driver identity carries beside the catalog and binding fingerprints. A Case the
// deployment cannot bind rejects here: a handler queue the Case binds that handlerQueue does not
// route to (or the reverse), a Profile that cannot be derived, or a *testpilot.PreparationError.
func Prepare(deployment Deployment, handlerQueue, identity string, source *testpilotspb.Case) (*Prepared, error) {
	catalog, err := testpilotdriver.NewWorkflowServiceCatalog()
	if err != nil {
		return nil, fmt.Errorf("build method catalog: %w", err)
	}
	return PrepareWith(catalog, deployment, handlerQueue, identity, source)
}

// PrepareWith is Prepare against a catalog the caller already holds, the campaign's for every
// candidate.
func PrepareWith(catalog *testpilot.Catalog, deployment Deployment, handlerQueue, identity string, source *testpilotspb.Case) (*Prepared, error) {
	if source == nil || catalog == nil {
		return nil, errors.New("the Case and the catalog are required")
	}
	// The handler queue is the one the deployment routes its endpoint to, so the endpoint's route
	// and the handler's poll never disagree: a Case that binds a handler queue of its own under a
	// deployment bound without one, or the reverse, would poll one queue while the endpoint routes
	// to another, and is refused here rather than left to time out.
	if bindsHandlerQueue := testpilotdriver.HandlerTaskQueueBindingID(source.GetProgram()) != ""; bindsHandlerQueue != (handlerQueue != "") {
		return nil, fmt.Errorf("case %q binds a Nexus handler queue of its own (%t) but the campaign was opened with handler queue %q",
			source.GetCaseId(), bindsHandlerQueue, handlerQueue)
	}
	profile, err := testpilotdriver.DeriveProfile(source, catalog, testpilotdriver.Environment{
		Identity:         identity,
		Namespace:        deployment.Namespace,
		TaskQueue:        deployment.TaskQueue,
		HandlerTaskQueue: handlerQueue,
		NexusEndpoint:    deployment.NexusEndpoint,
	})
	if err != nil {
		return nil, fmt.Errorf("derive Profile for Case %q: %w", source.GetCaseId(), err)
	}
	prepared, err := testpilot.Prepare(source, profile)
	if err != nil {
		return nil, err
	}
	return &Prepared{Case: prepared, Profile: profile}, nil
}

// Bind prepares one Case under this deployment and opens its Driver. identity names the Profile
// the Case runs under, which the prepared Case's Driver identity carries. A Case the deployment
// cannot bind rejects before any Driver opens, as Prepare decides it: it holds nothing and
// creates no Run.
func (c *Campaign) Bind(ctx context.Context, identity string, source *testpilotspb.Case) (*Bound, error) {
	if c == nil || source == nil {
		return nil, errors.New("campaign binding and Case are required")
	}
	prepared, err := PrepareWith(c.catalog, c.deployment, c.handlerQueue, identity, source)
	if err != nil {
		return nil, err
	}
	profile := prepared.Profile
	bound := &Bound{prepared: prepared.Case}
	fail := func(err error) (*Bound, error) {
		return nil, errors.Join(err, ReleaseAll(context.WithoutCancel(ctx), bound.releases))
	}
	// The SDK's default logger writes to stdout, where a command writes its one report; its
	// worker lines go to stderr with the rest of the progress.
	caseClient, err := sdkclient.DialContext(ctx, sdkclient.Options{
		HostPort: c.deployment.GRPCAddress, Namespace: c.deployment.Namespace,
		Logger: sdklog.NewStructuredLogger(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))),
	})
	if err != nil {
		return fail(fmt.Errorf("open SDK client: %w", err))
	}
	bound.releases = append(bound.releases, func(context.Context) error { caseClient.Close(); return nil })
	driver, err := testpilotdriver.New(testpilotdriver.Options{
		Profile: profile,
		ServerEndpoints: map[string]testpilotdriver.Endpoint{
			workflowRole: {Target: c.deployment.GRPCAddress, Credentials: insecure.NewCredentials()},
		},
		SystemCallbackBaseURL: "http://" + c.deployment.HTTPAddress,
		SDKClient:             caseClient,
		WorkerRoleID:          workerRole,
		WorkerStopTimeout:     workerStopTimeout,
	})
	if err != nil {
		return fail(fmt.Errorf("open Driver: %w", err))
	}
	bound.driver = driver
	bound.releases = append(bound.releases, driver.Close)
	return bound, nil
}

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
		each, cancel := context.WithTimeout(ctx, teardownTimeout)
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
