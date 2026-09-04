package local

import (
	"context"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	umpireruntime "go.temporal.io/server/tools/umpire/runtime"
)

// AttachedAuthority supplies the borrowed SDK client and namespace identity
// needed to run in a caller-owned Temporal environment.
type AttachedAuthority interface {
	SDKClient() client.Client
	Namespace() string
	Endpoint() string
}

// NewAttachedFactory binds one borrowed Temporal authority for sequential,
// isolated runs. Umpire never owns or closes the supplied client or cluster.
func NewAttachedFactory(authority AttachedAuthority) (umpireruntime.EnvironmentFactory, error) {
	return newAttachedFactory(authority, newSDKAttachedWorker)
}

type attachedWorker interface {
	worker.Registry
	Start() error
	Stop() error
}

type attachedWorkerFactory func(client.Client, string, worker.Options) (attachedWorker, error)

type attachedBinding struct {
	authority AttachedAuthority
	client    client.Client
	namespace string
	endpoint  string
	workers   attachedWorkerFactory

	mu     sync.Mutex
	active bool
}

func newAttachedFactory(
	authority AttachedAuthority,
	workers attachedWorkerFactory,
) (*factory, error) {
	binding, err := bindAttachedAuthority(authority, workers)
	if err != nil {
		return nil, err
	}
	return newFactory(attachedStarter{binding: binding}), nil
}

func bindAttachedAuthority(
	authority AttachedAuthority,
	workers attachedWorkerFactory,
) (*attachedBinding, error) {
	if isNilInterface(authority) || workers == nil {
		return nil, errors.New("attached Temporal authority is incomplete")
	}
	binding := &attachedBinding{
		authority: authority,
		client:    authority.SDKClient(),
		namespace: authority.Namespace(),
		endpoint:  authority.Endpoint(),
		workers:   workers,
	}
	if isNilInterface(binding.client) || strings.TrimSpace(binding.namespace) == "" ||
		strings.TrimSpace(binding.endpoint) == "" {
		return nil, errors.New("attached Temporal authority is incomplete")
	}
	if !reflect.ValueOf(binding.client).Comparable() {
		return nil, errors.New("attached Temporal client identity is not comparable")
	}
	return binding, nil
}

func (b *attachedBinding) validate(ctx context.Context) error {
	if ctx == nil || ctx.Err() != nil {
		return contextError(ctx)
	}
	if isNilInterface(b.authority) {
		return errors.New("attached Temporal authority binding drifted")
	}
	currentClient := b.authority.SDKClient()
	currentNamespace := b.authority.Namespace()
	currentEndpoint := b.authority.Endpoint()
	if isNilInterface(currentClient) || !sameClient(b.client, currentClient) ||
		b.namespace != currentNamespace || b.endpoint != currentEndpoint {
		return errors.New("attached Temporal authority binding drifted")
	}
	return nil
}

func (b *attachedBinding) claim(ctx context.Context) error {
	if err := b.validate(ctx); err != nil {
		return err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.active {
		return errors.New("attached Temporal authority already has an active run")
	}
	b.active = true
	return nil
}

func (b *attachedBinding) release() {
	b.mu.Lock()
	b.active = false
	b.mu.Unlock()
}

func sameClient(left client.Client, right client.Client) bool {
	if isNilInterface(left) || isNilInterface(right) {
		return isNilInterface(left) && isNilInterface(right)
	}
	leftValue := reflect.ValueOf(left)
	rightValue := reflect.ValueOf(right)
	return leftValue.Type() == rightValue.Type() && leftValue.Comparable() &&
		leftValue.Interface() == rightValue.Interface()
}

func isNilInterface(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}

type attachedStarter struct {
	binding *attachedBinding
}

func (s attachedStarter) Start(ctx context.Context) (temporalAuthority, error) {
	if s.binding == nil {
		return nil, errors.New("attached Temporal authority is incomplete")
	}
	if err := s.binding.claim(ctx); err != nil {
		return nil, err
	}
	return &attachedTemporalAuthority{binding: s.binding}, nil
}

type attachedTemporalAuthority struct {
	binding *attachedBinding

	mu          sync.Mutex
	worker      attachedWorker
	startDone   chan struct{}
	startErr    error
	stopAttempt *attachedStopAttempt
	closed      bool
	release     sync.Once
}

type attachedStopAttempt struct {
	done chan struct{}
	err  error
}

func (a *attachedTemporalAuthority) isolationProbe(
	workflowIdentity string,
) executionIsolationProbe {
	if a == nil || a.binding == nil {
		return nil
	}
	return attachedExecutionIsolationProbe{
		client: a.binding.client, workflowIdentity: workflowIdentity,
	}
}

type attachedExecutionIsolationProbe struct {
	client           client.Client
	workflowIdentity string
}

func (p attachedExecutionIsolationProbe) Verify(ctx context.Context, workflowIdentity string) error {
	if ctx == nil || ctx.Err() != nil {
		return contextError(ctx)
	}
	if isNilInterface(p.client) || p.workflowIdentity == "" || workflowIdentity != p.workflowIdentity {
		return errors.New("attached Temporal execution isolation binding drifted")
	}
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		response, err := p.client.ListWorkflow(ctx, &workflowservice.ListWorkflowExecutionsRequest{
			PageSize: 2,
			Query:    "WorkflowId = " + strconv.Quote(p.workflowIdentity),
		})
		if err != nil {
			return err
		}
		if response != nil && len(response.Executions) == 1 && len(response.NextPageToken) == 0 {
			execution := response.Executions[0].GetExecution()
			if execution != nil && execution.GetWorkflowId() == p.workflowIdentity {
				return nil
			}
			return errors.New("attached Temporal authority contains an unexpected execution")
		}
		if response != nil && (len(response.Executions) > 1 || len(response.NextPageToken) != 0) {
			return errors.New("attached Temporal authority contains duplicate run executions")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (a *attachedTemporalAuthority) Connect(ctx context.Context) error {
	if a == nil || a.binding == nil {
		return errors.New("attached Temporal authority is incomplete")
	}
	return a.binding.validate(ctx)
}

func (a *attachedTemporalAuthority) SDKClient() client.Client {
	if a == nil || a.binding == nil {
		return nil
	}
	return a.binding.client
}

func (a *attachedTemporalAuthority) StartWorker(
	ctx context.Context,
	taskQueue string,
	identity string,
	registration WorkerRegistration,
) error {
	if a == nil || a.binding == nil {
		return errors.New("attached Temporal authority is incomplete")
	}
	if err := a.binding.validate(ctx); err != nil {
		return err
	}
	if registration == nil || taskQueue == "" || identity == "" {
		return errors.New("attached Temporal worker binding is incomplete")
	}
	a.mu.Lock()
	if a.worker != nil || a.closed {
		a.mu.Unlock()
		return errors.New("attached Temporal worker already exists")
	}
	owned, err := a.binding.workers(a.binding.client, taskQueue, worker.Options{
		Identity:          identity,
		WorkerStopTimeout: umpireruntime.CanonicalPhaseLimits()[4].Duration(),
	})
	if err != nil {
		a.mu.Unlock()
		return err
	}
	if isNilInterface(owned) {
		a.mu.Unlock()
		return errors.New("attached Temporal worker factory returned no worker")
	}
	a.worker = owned
	a.startDone = make(chan struct{})
	registration.Register(owned)
	startDone := a.startDone
	a.mu.Unlock()

	go func() {
		err := owned.Start()
		a.mu.Lock()
		a.startErr = err
		close(startDone)
		a.mu.Unlock()
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-startDone:
		if err := ctx.Err(); err != nil {
			return err
		}
		a.mu.Lock()
		defer a.mu.Unlock()
		return a.startErr
	}
}

func (a *attachedTemporalAuthority) Stop(ctx context.Context) error {
	if a == nil || a.binding == nil {
		return errors.New("attached Temporal authority is incomplete")
	}
	if ctx == nil || ctx.Err() != nil {
		return contextError(ctx)
	}
	a.mu.Lock()
	if a.worker == nil {
		a.closed = true
		a.mu.Unlock()
		a.releaseLease()
		return nil
	}
	if a.stopAttempt == nil {
		attempt := &attachedStopAttempt{done: make(chan struct{})}
		a.stopAttempt = attempt
		owned := a.worker
		startDone := a.startDone
		go func() {
			// SDK worker lifecycle calls are context-free. Waiting outside the
			// caller lets its deadline remain authoritative while ownership stays
			// live until a started worker has actually stopped.
			<-startDone
			err := owned.Stop()
			release := false
			a.mu.Lock()
			attempt.err = err
			if err == nil {
				a.worker = nil
				a.closed = true
				release = true
			}
			if a.stopAttempt == attempt {
				a.stopAttempt = nil
			}
			close(attempt.done)
			a.mu.Unlock()
			if release {
				a.releaseLease()
			}
		}()
	}
	attempt := a.stopAttempt
	a.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-attempt.done:
		if err := ctx.Err(); err != nil {
			return err
		}
		return attempt.err
	}
}

func (a *attachedTemporalAuthority) releaseLease() {
	a.release.Do(a.binding.release)
}

func (a *attachedTemporalAuthority) OwnedResources() []ownedResource {
	if a == nil {
		return []ownedResource{}
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closed {
		return []ownedResource{}
	}
	// This marker is the per-run environment wrapper; it never represents or
	// stops the borrowed Temporal cluster or client.
	resources := []ownedResource{{kind: ownedEnvironment}}
	if a.worker != nil {
		resources = append(resources, ownedResource{kind: ownedWorker})
	}
	return resources
}

func (a *attachedTemporalAuthority) Namespace() string {
	if a == nil || a.binding == nil {
		return ""
	}
	return a.binding.namespace
}

func (a *attachedTemporalAuthority) Endpoint() string {
	if a == nil || a.binding == nil {
		return ""
	}
	return a.binding.endpoint
}

type sdkAttachedWorker struct {
	worker.Worker
}

func newSDKAttachedWorker(
	sdkClient client.Client,
	taskQueue string,
	options worker.Options,
) (attachedWorker, error) {
	if isNilInterface(sdkClient) || taskQueue == "" || options.Identity == "" {
		return nil, errors.New("attached Temporal worker binding is incomplete")
	}
	owned := worker.New(sdkClient, taskQueue, options)
	if isNilInterface(owned) {
		return nil, errors.New("attached Temporal worker construction failed")
	}
	return &sdkAttachedWorker{Worker: owned}, nil
}

func (w *sdkAttachedWorker) Stop() error {
	if w == nil || isNilInterface(w.Worker) {
		return errors.New("attached Temporal worker is incomplete")
	}
	w.Worker.Stop()
	return nil
}

var _ temporalAuthority = (*attachedTemporalAuthority)(nil)
