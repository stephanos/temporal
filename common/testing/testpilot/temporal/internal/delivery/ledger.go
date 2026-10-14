package delivery

import (
	"cmp"
	"context"
	"errors"
	"slices"
	"sync/atomic"

	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal/internal/primitive"
)

var (
	ErrInvalid         = errors.New("invalid reservation delivery input")
	ErrCapacity        = errors.New("reservation delivery capacity exhausted")
	ErrBindingMismatch = errors.New("workflow delivery binding does not match")
	ErrRouteConflict   = errors.New("reservation delivery identity conflicts with its admission")
	ErrRouteStale      = errors.New("reservation delivery route is no longer active")
	ErrReservedHeader  = errors.New("reserved delivery header is already present")
	ErrLifecycle       = errors.New("reservation handle lifecycle operation failed")
)

type Limits struct {
	MaxRoutes      int
	MaxHeaderBytes int
	MaxHandles     int
	MaxDiagnostics int
}

type Config struct {
	RunID, SessionID string
	Limits           Limits
}

// A Ledger keeps each activity bundle in operations, and the attempts of each activity a workflow
// schedules in scheduled, under the reservation its route names on the wire, for as long as the
// bundle lives: the attempts' own routes leave routes as their reservations end.
type Ledger struct {
	mu            primitive.Mutex
	config        Config
	bundles       map[uint64]*bundleState
	routes        map[string]*routeState
	operations    map[string]*bundleState
	scheduled     map[string][]*routeState
	retained      map[string]*retainedReservation
	nextBundle    uint64
	activeRoutes  int
	activeHandles int
	diagnostics   int
	stopped       bool
}

type authority uint8

const (
	reserved authority = iota
	admitted
	terminal
	canceled
)

type bundleState struct {
	id              uint64
	origin          testpilot.Coordinate
	plan            testpilot.ReservationCarrierPlan
	binding         WorkflowBinding
	activityBinding ActivityBinding
	workflow        *routeState
	attempts        []*routeState
	nexus           map[sourceKey]*routeState
	// scheduled holds, per schedule command of the workflow, the attempts of the activity it
	// reaches, in attempt order.
	scheduled       map[sourceKey][]*routeState
	routes          []*routeState
	responseRunID   string
	triggerStatus   TriggerStatus
	triggerFinal    bool
	triggerCanceled []*routeState
	parentReleased  bool
	parentCanceled  []*routeState
	active          int
}

// started is the route the carrier's own start request names. The bundle of a carrier that starts
// a standalone activity holds activityBinding and attempts, one route per attempt its script
// declares in attempt order, where any other holds binding and workflow; an activity start names
// its first attempt's route, which stands for the whole operation on the wire.
func (b *bundleState) started() *routeState {
	if len(b.attempts) > 0 {
		return b.attempts[0]
	}
	return b.workflow
}

// startedRunID is the run a delivery already admitted for the started entity: the workflow's, or
// the activity run every admitted attempt belongs to.
func (b *bundleState) startedRunID() string {
	for _, attempt := range b.attempts {
		if attempt.activation.temporalRunID != "" {
			return attempt.activation.temporalRunID
		}
	}
	if b.workflow != nil {
		return b.workflow.activation.temporalRunID
	}
	return ""
}

type sourceKey struct {
	workflowEntrypoint string
	workflowOrdinal    int64
	sourceInstruction  string
}

type routeState struct {
	bundle     *bundleState
	identity   testpilot.ReservationIdentity
	retained   *retainedReservation
	kind       routeKind
	source     sourceKey
	authority  authority
	activation activationData
}

// An activity attempt's activationData also holds its SDK identities: attempt is the server's
// attempt number, and deliveryID the delivery that first carried it.
//
// An attempt of an activity a workflow scheduled also holds the activity ID the SDK gave it, which
// every attempt of that activity shares.
type activationData struct {
	coordinate    testpilot.Coordinate
	temporalRunID string
	requestID     string
	attempt       int32
	deliveryID    string
	activityID    string
}

type Bundle struct {
	ledger *Ledger
	id     uint64
}

type Activation struct {
	ledger *Ledger
	state  *routeState
	data   activationData
	replay bool
}

func (a Activation) Coordinate() testpilot.Coordinate { return a.data.coordinate }
func (a Activation) Reservation() testpilot.ReservationIdentity {
	if a.state == nil {
		return testpilot.ReservationIdentity{}
	}
	return a.state.identity
}
func (a Activation) TemporalRunID() string { return a.data.temporalRunID }
func (a Activation) RequestID() string     { return a.data.requestID }
func (a Activation) Attempt() int32        { return a.data.attempt }
func (a Activation) DeliveryID() string    { return a.data.deliveryID }
func (a Activation) Replay() bool          { return a.replay }

type Release struct {
	unused    int
	notNeeded []testpilot.ReservationIdentity
}

func (r Release) Unused() int { return r.unused }

// NotNeeded are the declared attempts a released workflow's activities will never be delivered,
// for the caller to settle as not needed.
func (r Release) NotNeeded() []testpilot.ReservationIdentity { return slices.Clone(r.notNeeded) }

type TriggerStatus uint8

type CompletionFunc func()
type QuarantineFunc func(context.Context, testpilot.EffectHandle, CompletionFunc) error

const (
	TriggerSucceeded TriggerStatus = iota + 1
	TriggerRejected
	TriggerCanceled
	TriggerNonSuccess
	TriggerUncertain
)

func (d TriggerStatus) String() string {
	switch d {
	case TriggerSucceeded:
		return "succeeded"
	case TriggerRejected:
		return "rejected"
	case TriggerCanceled:
		return "canceled"
	case TriggerNonSuccess:
		return "non-success"
	case TriggerUncertain:
		return "uncertain"
	default:
		return "unknown"
	}
}

func New(config Config) (*Ledger, error) {
	limits := config.Limits
	if !validRouteText(config.RunID) || !validRouteText(config.SessionID) || limits.MaxRoutes <= 0 || limits.MaxRoutes > 100000 || limits.MaxHeaderBytes <= 0 || limits.MaxHeaderBytes > 16<<20 || limits.MaxHandles <= 0 || limits.MaxHandles > 100000 || limits.MaxDiagnostics <= 0 || limits.MaxDiagnostics > 100000 {
		return nil, ErrInvalid
	}
	return &Ledger{mu: primitive.NewMutex(), config: config, bundles: make(map[uint64]*bundleState), routes: make(map[string]*routeState), operations: make(map[string]*bundleState), scheduled: make(map[string][]*routeState), retained: make(map[string]*retainedReservation)}, nil
}

// RetainReservation attaches lifecycle accounting before the handle is returned from Session.Reserve.
// The returned proxy is the handle that the Executor must retain and later Wait, Cancel or Drain.
func (l *Ledger) RetainReservation(ctx context.Context, handle testpilot.ReservationHandle) (testpilot.ReservationHandle, error) {
	if primitive.NilValue(handle) {
		return handle, ErrInvalid
	}
	cleanup := newCleanupProxy(handle)
	if err := primitive.ContextError(ctx, ErrInvalid); err != nil {
		return cleanup, err
	}
	identity := handle.Identity()
	if identity.Origin.RunID != l.config.RunID || !validCoordinate(identity.Origin) || !validReservation(identity) {
		return cleanup, ErrRouteConflict
	}
	if err := l.mu.LockContext(ctx, ErrInvalid); err != nil {
		return cleanup, err
	}
	defer l.mu.Unlock()
	if l.stopped {
		return cleanup, ErrRouteStale
	}
	if l.activeHandles >= l.config.Limits.MaxHandles {
		return cleanup, ErrCapacity
	}
	if _, duplicate := l.retained[identity.ID]; duplicate {
		return cleanup, ErrRouteConflict
	}
	retained := &retainedReservation{ledger: l, identity: identity, handle: handle}
	retained.proxy = &reservationProxy{retained: retained}
	l.retained[identity.ID] = retained
	l.activeHandles++
	return retained.proxy, nil
}

// startBinding is the physical entity a carrier's start request names, by the kind of route that
// start activates: a workflow, or a standalone activity.
type startBinding struct {
	kind     routeKind
	workflow WorkflowBinding
	activity ActivityBinding
}

// Carried names the reservation carriers the Temporal Driver realizes and the entrypoint kinds each
// delivers, the one table the Driver's Profile derivation and this ledger read: a workflow start
// carries the reservations of the workflow it starts and of the Nexus handlers and activities that
// workflow schedules, and an activity start the reservations of the activity it starts; an
// activity's are one per attempt.
var Carried = map[string][]testpilot.EntrypointKind{
	primitive.StartWorkflowPath: {testpilot.WorkflowEntrypoint, testpilot.NexusHandlerEntrypoint, testpilot.ActivityEntrypoint},
	StartActivityPath:           {testpilot.ActivityEntrypoint},
}

// carries reports whether plan is one the start can carry: its method is the start's, and each of its
// reservations of a kind Carried says the method delivers; an activity start carries the one activity
// it starts, with no route.
func (b startBinding) carries(plan testpilot.ReservationCarrierPlan) bool {
	for _, reservation := range plan.Reservations {
		if !slices.Contains(Carried[plan.Method], reservation.Kind) || reservation.Restart != 0 && b.kind != activityRoute {
			return false
		}
	}
	switch b.kind {
	case workflowRoute:
		return plan.Method == primitive.StartWorkflowPath && validBinding(b.workflow)
	case activityRoute:
		return plan.Method == StartActivityPath && validActivityBinding(b.activity) && len(plan.Routes) == 0 &&
			len(plan.Reservations) == 1 && plan.Reservations[0].Count >= 1 &&
			plan.Reservations[0].Restart >= 0 && plan.Reservations[0].Restart < plan.Reservations[0].Count
	default:
		return false
	}
}

func (l *Ledger) CreateBundle(ctx context.Context, origin testpilot.Coordinate, plan testpilot.ReservationCarrierPlan, workflowBinding WorkflowBinding, handles []testpilot.ReservationHandle) (Bundle, error) {
	return l.createBundle(ctx, origin, plan, startBinding{kind: workflowRoute, workflow: workflowBinding}, handles)
}

// CreateActivityBundle is CreateBundle for a carrier that starts a standalone activity. Its
// reservations are the activity's own attempts, so the bundle holds one route per attempt.
func (l *Ledger) CreateActivityBundle(ctx context.Context, origin testpilot.Coordinate, plan testpilot.ReservationCarrierPlan, activityBinding ActivityBinding, handles []testpilot.ReservationHandle) (Bundle, error) {
	return l.createBundle(ctx, origin, plan, startBinding{kind: activityRoute, activity: activityBinding}, handles)
}

func (l *Ledger) createBundle(ctx context.Context, origin testpilot.Coordinate, plan testpilot.ReservationCarrierPlan, start startBinding, handles []testpilot.ReservationHandle) (Bundle, error) {
	if err := primitive.ContextError(ctx, ErrInvalid); err != nil {
		return Bundle{}, err
	}
	validated, ordered, err := validateBundle(l.config.RunID, origin, plan, start, handles, l.config.Limits)
	if err != nil {
		return Bundle{}, err
	}
	if err := l.mu.LockContext(ctx, ErrInvalid); err != nil {
		return Bundle{}, err
	}
	defer l.mu.Unlock()
	if l.stopped {
		return Bundle{}, ErrRouteStale
	}
	if l.activeRoutes > l.config.Limits.MaxRoutes-len(ordered) {
		return Bundle{}, ErrCapacity
	}
	if len(l.bundles) >= l.config.Limits.MaxRoutes {
		return Bundle{}, ErrCapacity
	}
	proxies := make([]*reservationProxy, 0, len(ordered))
	for _, handle := range ordered {
		proxy, ok := handle.(*reservationProxy)
		if !ok || proxy == nil || proxy.retained == nil || proxy.retained.ledger != l || proxy.retained.completed || proxy.retained.route != nil || l.retained[handle.Identity().ID] != proxy.retained {
			return Bundle{}, ErrRouteConflict
		}
		if _, exists := l.routes[handle.Identity().ID]; exists {
			return Bundle{}, ErrRouteConflict
		}
		proxies = append(proxies, proxy)
	}
	l.nextBundle++
	state := &bundleState{id: l.nextBundle, origin: origin, plan: clonePlan(plan), binding: start.workflow, activityBinding: start.activity, nexus: make(map[sourceKey]*routeState), scheduled: make(map[sourceKey][]*routeState), active: len(ordered)}
	byIdentity := make(map[reservationKey]*routeState, len(ordered))
	scheduledAttempts := make(map[string][]*routeState)
	for _, proxy := range proxies {
		identity := proxy.Identity()
		routeState := &routeState{bundle: state, identity: identity, retained: proxy.retained, authority: reserved}
		proxy.retained.route = routeState
		entrypointKind := validated[reservationKey{entrypoint: identity.EntrypointID, ordinal: identity.Ordinal}]
		switch entrypointKind {
		case testpilot.WorkflowEntrypoint:
			routeState.kind = workflowRoute
			state.workflow = routeState
		case testpilot.ActivityEntrypoint:
			if start.kind == workflowRoute {
				routeState.kind = scheduledActivityRoute
				scheduledAttempts[identity.EntrypointID] = append(scheduledAttempts[identity.EntrypointID], routeState)
				break
			}
			routeState.kind = activityRoute
			state.attempts = append(state.attempts, routeState)
		default:
			routeState.kind = nexusRoute
		}
		state.routes = append(state.routes, routeState)
		byIdentity[reservationKey{entrypoint: identity.EntrypointID, ordinal: identity.Ordinal}] = routeState
		l.routes[identity.ID] = routeState
	}
	for _, route := range plan.Routes {
		key := sourceKey{workflowEntrypoint: route.WorkflowEntrypointID, workflowOrdinal: route.WorkflowOrdinal, sourceInstruction: route.SourceInstructionID}
		if attempts := scheduledAttempts[route.HandlerEntrypointID]; len(attempts) > 0 {
			slices.SortFunc(attempts, func(left, right *routeState) int { return cmp.Compare(left.identity.Ordinal, right.identity.Ordinal) })
			for _, attempt := range attempts {
				attempt.source = key
			}
			state.scheduled[key] = attempts
			l.scheduled[attempts[0].identity.ID] = attempts
			continue
		}
		handler := byIdentity[reservationKey{entrypoint: route.HandlerEntrypointID, ordinal: route.HandlerOrdinal}]
		handler.source = key
		state.nexus[key] = handler
	}
	if len(state.attempts) > 0 {
		slices.SortFunc(state.attempts, func(left, right *routeState) int { return cmp.Compare(left.identity.Ordinal, right.identity.Ordinal) })
		l.operations[state.attempts[0].identity.ID] = state
	}
	l.bundles[state.id] = state
	l.activeRoutes += len(ordered)
	return Bundle{ledger: l, id: state.id}, nil
}

type reservationKey struct {
	entrypoint string
	ordinal    int64
}

func validateBundle(runID string, origin testpilot.Coordinate, plan testpilot.ReservationCarrierPlan, start startBinding, handles []testpilot.ReservationHandle, limits Limits) (map[reservationKey]testpilot.EntrypointKind, []testpilot.ReservationHandle, error) {
	if origin.RunID != runID || !validCoordinate(origin) || !start.carries(plan) || len(plan.Reservations) > limits.MaxRoutes || len(plan.Routes) > limits.MaxRoutes {
		return nil, nil, ErrInvalid
	}
	expected, err := expectedHandles(plan, limits)
	if err != nil {
		return nil, nil, err
	}
	ordered, err := validateHandles(origin, expected, handles)
	if err != nil {
		return nil, nil, err
	}
	return expected, ordered, nil
}

// expectedHandles is the handle each reservation of the admitted plan expects, bounded by the
// runtime route limit. Preparation compiled the plan's shape, so it is not re-checked here.
func expectedHandles(plan testpilot.ReservationCarrierPlan, limits Limits) (map[reservationKey]testpilot.EntrypointKind, error) {
	expected := make(map[reservationKey]testpilot.EntrypointKind)
	for _, topology := range plan.Reservations {
		if topology.Count > int64(limits.MaxRoutes-len(expected)) {
			return nil, ErrInvalid
		}
		for ordinal := int64(0); ordinal < topology.Count; ordinal++ {
			expected[reservationKey{entrypoint: topology.EntrypointID, ordinal: ordinal}] = topology.Kind
		}
	}
	return expected, nil
}

func validateHandles(origin testpilot.Coordinate, expected map[reservationKey]testpilot.EntrypointKind, handles []testpilot.ReservationHandle) ([]testpilot.ReservationHandle, error) {
	if len(handles) != len(expected) {
		return nil, ErrInvalid
	}
	ordered := make([]testpilot.ReservationHandle, 0, len(handles))
	seenKeys := make(map[reservationKey]bool, len(handles))
	seenIDs := make(map[string]bool, len(handles))
	for _, handle := range handles {
		if primitive.NilValue(handle) {
			return nil, ErrInvalid
		}
		identity := handle.Identity()
		key := reservationKey{entrypoint: identity.EntrypointID, ordinal: identity.Ordinal}
		if identity.Origin != origin || !validReservation(identity) || seenKeys[key] || seenIDs[identity.ID] {
			return nil, ErrRouteConflict
		}
		if _, exists := expected[key]; !exists {
			return nil, ErrRouteConflict
		}
		seenKeys[key] = true
		seenIDs[identity.ID] = true
		ordered = append(ordered, handle)
	}
	if len(seenKeys) != len(expected) {
		return nil, ErrInvalid
	}
	return ordered, nil
}

func clonePlan(plan testpilot.ReservationCarrierPlan) testpilot.ReservationCarrierPlan {
	plan.Reservations = slices.Clone(plan.Reservations)
	plan.Routes = slices.Clone(plan.Routes)
	return plan
}

// StartResponse is what a carried start answers with: the run it started, of a workflow or of a
// standalone activity.
type StartResponse interface{ GetRunId() string }

func (l *Ledger) PinStartResponse(ctx context.Context, bundle Bundle, response StartResponse) error {
	if err := primitive.ContextError(ctx, ErrInvalid); err != nil {
		return err
	}
	if primitive.NilValue(response) || !validRouteText(response.GetRunId()) {
		return ErrInvalid
	}
	if err := l.mu.LockContext(ctx, ErrInvalid); err != nil {
		return err
	}
	defer l.mu.Unlock()
	state, err := l.bundleLocked(bundle)
	if err != nil {
		return err
	}
	if l.stopped || state.triggerFinal && state.triggerStatus != TriggerSucceeded {
		l.diagnoseLocked()
		return ErrRouteStale
	}
	runID := response.GetRunId()
	if state.responseRunID != "" && state.responseRunID != runID || state.startedRunID() != "" && state.startedRunID() != runID {
		return ErrRouteConflict
	}
	state.responseRunID = runID
	return nil
}

func (l *Ledger) TriggerTerminal(ctx context.Context, bundle Bundle, disposition TriggerStatus) (Release, error) {
	if err := primitive.ContextError(ctx, ErrInvalid); err != nil {
		return Release{}, err
	}
	if disposition < TriggerSucceeded || disposition > TriggerUncertain {
		return Release{}, ErrInvalid
	}
	if err := l.mu.LockContext(ctx, ErrInvalid); err != nil {
		return Release{}, err
	}
	state, err := l.bundleLocked(bundle)
	if err != nil {
		l.mu.Unlock()
		return Release{}, err
	}
	if l.stopped {
		l.diagnoseLocked()
		l.mu.Unlock()
		return Release{}, ErrRouteStale
	}
	if state.triggerFinal {
		if state.triggerStatus != disposition {
			l.mu.Unlock()
			return Release{}, ErrRouteConflict
		}
		if disposition == TriggerSucceeded {
			l.mu.Unlock()
			return Release{}, nil
		}
		pending := l.pendingCancellationLocked(state.triggerCanceled)
		l.mu.Unlock()
		return Release{}, l.cancel(ctx, pending)
	}
	if disposition == TriggerSucceeded {
		if state.responseRunID == "" {
			l.mu.Unlock()
			return Release{}, ErrRouteConflict
		}
		state.triggerFinal = true
		state.triggerStatus = disposition
		l.retireBundleLocked(state)
		l.mu.Unlock()
		return Release{}, nil
	}
	state.triggerFinal = true
	state.triggerStatus = disposition
	release := Release{}
	for _, route := range state.routes {
		if route.authority == reserved {
			release.unused++
		}
		if route.authority == reserved || route.authority == admitted {
			route.authority = canceled
		}
		if route.authority == canceled && !route.retained.completed {
			state.triggerCanceled = append(state.triggerCanceled, route)
		}
	}
	pending := l.pendingCancellationLocked(state.triggerCanceled)
	l.retireBundleLocked(state)
	l.mu.Unlock()
	return release, l.cancel(ctx, pending)
}

func (l *Ledger) ParentTerminal(ctx context.Context, workflow Activation) (Release, error) {
	if err := primitive.ContextError(ctx, ErrInvalid); err != nil {
		return Release{}, err
	}
	if err := l.mu.LockContext(ctx, ErrInvalid); err != nil {
		return Release{}, err
	}
	state, err := l.activationLocked(workflow, workflowRoute)
	if err != nil {
		l.mu.Unlock()
		return Release{}, err
	}
	if l.stopped {
		l.diagnoseLocked()
		l.mu.Unlock()
		return Release{}, ErrRouteStale
	}
	bundle := state.bundle
	if bundle.parentReleased {
		pending := l.pendingCancellationLocked(bundle.parentCanceled)
		l.mu.Unlock()
		return Release{}, l.cancel(ctx, pending)
	}
	bundle.parentReleased = true
	state.authority = terminal
	release := Release{}
	var routes []*routeState
	for _, candidate := range bundle.routes {
		if candidate.kind == nexusRoute && candidate.authority == reserved {
			candidate.authority = canceled
			release.unused++
			routes = append(routes, candidate)
			bundle.parentCanceled = append(bundle.parentCanceled, candidate)
		}
	}
	// A closed workflow's activities get no further attempt, so the attempts declared after the
	// last one delivered are not needed. An activity none of whose attempts was delivered is not
	// explained by the workflow closing, so its attempts stay reserved.
	for _, attempts := range bundle.scheduled {
		release.notNeeded = append(release.notNeeded, releaseUndeliveredAttempts(attempts)...)
	}
	pending := l.pendingCancellationLocked(routes)
	l.mu.Unlock()
	return release, l.cancel(ctx, pending)
}

func (l *Ledger) Stop(ctx context.Context) (Release, error) {
	if err := primitive.ContextError(ctx, ErrInvalid); err != nil {
		return Release{}, err
	}
	if err := l.mu.LockContext(ctx, ErrInvalid); err != nil {
		return Release{}, err
	}
	if l.stopped {
		pending := l.pendingRetainedCancellationLocked()
		l.mu.Unlock()
		return Release{}, l.cancel(ctx, pending)
	}
	l.stopped = true
	release := Release{}
	for _, route := range l.routes {
		if route.authority == reserved {
			release.unused++
		}
		if route.authority == reserved || route.authority == admitted {
			route.authority = canceled
		}
	}
	for _, retained := range l.retained {
		if retained.route == nil && !retained.completed {
			release.unused++
		}
	}
	pending := l.pendingRetainedCancellationLocked()
	l.mu.Unlock()
	return release, l.cancel(ctx, pending)
}

func (l *Ledger) Quarantine(ctx context.Context, handle testpilot.EffectHandle, quarantine QuarantineFunc) error {
	if err := primitive.ContextError(ctx, ErrInvalid); err != nil {
		return err
	}
	if primitive.NilValue(handle) || primitive.NilValue(quarantine) {
		return ErrInvalid
	}
	proxy, ok := handle.(*reservationProxy)
	if !ok || proxy == nil || proxy.retained == nil || proxy.retained.ledger != l {
		return ErrRouteCrossed
	}
	if err := l.mu.LockContext(ctx, ErrInvalid); err != nil {
		return err
	}
	if proxy.retained.quarantined.Load() {
		l.mu.Unlock()
		return nil
	}
	if proxy.retained.quarantining.Load() {
		l.mu.Unlock()
		return ErrLifecycle
	}
	if proxy.retained.completed || l.retained[proxy.retained.identity.ID] != proxy.retained {
		l.mu.Unlock()
		return ErrRouteStale
	}
	if !proxy.retained.quarantining.CompareAndSwap(false, true) {
		l.mu.Unlock()
		return ErrLifecycle
	}
	raw := proxy.retained.handle
	l.mu.Unlock()
	if err := quarantine(ctx, raw, func() { _ = proxy.complete(context.Background()) }); err != nil {
		proxy.retained.quarantining.Store(false)
		if contextErr := ctx.Err(); contextErr != nil {
			return contextErr
		}
		return ErrLifecycle
	}
	proxy.retained.quarantined.Store(true)
	proxy.retained.quarantining.Store(false)
	return nil
}

func (l *Ledger) bundleLocked(bundle Bundle) (*bundleState, error) {
	if bundle.ledger != l || bundle.id == 0 {
		return nil, ErrRouteCrossed
	}
	state := l.bundles[bundle.id]
	if state == nil {
		l.diagnoseLocked()
		return nil, ErrRouteStale
	}
	return state, nil
}

func (l *Ledger) activationLocked(activation Activation, kind routeKind) (*routeState, error) {
	if activation.ledger != l || activation.state == nil || activation.state.kind != kind || activation.data != activation.state.activation {
		return nil, ErrRouteCrossed
	}
	if activation.state.bundle == nil || l.bundles[activation.state.bundle.id] != activation.state.bundle {
		l.diagnoseLocked()
		return nil, ErrRouteStale
	}
	return activation.state, nil
}

func (l *Ledger) retireBundleLocked(state *bundleState) {
	if state.triggerFinal && state.active == 0 && l.bundles[state.id] == state {
		delete(l.bundles, state.id)
		if len(state.attempts) > 0 {
			delete(l.operations, state.attempts[0].identity.ID)
		}
		for _, attempts := range state.scheduled {
			delete(l.scheduled, attempts[0].identity.ID)
		}
	}
}

func (l *Ledger) pendingCancellationLocked(routes []*routeState) []*retainedReservation {
	result := make([]*retainedReservation, 0, len(routes))
	for _, route := range routes {
		if route != nil && route.retained != nil && !route.retained.completed && !route.retained.cancelSent.Load() && route.retained.canceling.CompareAndSwap(false, true) {
			result = append(result, route.retained)
		}
	}
	return result
}

func (l *Ledger) pendingRetainedCancellationLocked() []*retainedReservation {
	result := make([]*retainedReservation, 0, len(l.retained))
	for _, retained := range l.retained {
		if !retained.completed && !retained.cancelSent.Load() && retained.canceling.CompareAndSwap(false, true) {
			result = append(result, retained)
		}
	}
	return result
}

func (l *Ledger) cancel(ctx context.Context, retained []*retainedReservation) error {
	var result error
	for _, reservation := range retained {
		err := reservation.handle.Cancel(ctx)
		if err == nil {
			reservation.cancelSent.Store(true)
		}
		reservation.canceling.Store(false)
		if err != nil {
			result = ErrLifecycle
		}
	}
	if err := primitive.ContextError(ctx, ErrInvalid); err != nil {
		return err
	}
	return result
}

func (l *Ledger) diagnoseLocked() {
	if l.diagnostics < l.config.Limits.MaxDiagnostics {
		l.diagnostics++
	}
}

type retainedReservation struct {
	ledger       *Ledger
	identity     testpilot.ReservationIdentity
	handle       testpilot.ReservationHandle
	proxy        *reservationProxy
	route        *routeState
	completed    bool
	canceling    atomic.Bool
	cancelSent   atomic.Bool
	quarantining atomic.Bool
	quarantined  atomic.Bool
}

type reservationProxy struct{ retained *retainedReservation }

func newCleanupProxy(handle testpilot.ReservationHandle) *reservationProxy {
	retained := &retainedReservation{identity: handle.Identity(), handle: handle}
	proxy := &reservationProxy{retained: retained}
	retained.proxy = proxy
	return proxy
}

func (h *reservationProxy) Identity() testpilot.ReservationIdentity { return h.retained.identity }

func (h *reservationProxy) Consume(context.Context) (testpilot.Coordinate, error) {
	return testpilot.Coordinate{}, ErrInvalid
}

func (h *reservationProxy) Wait(ctx context.Context) (testpilot.EffectResult, error) {
	if err := primitive.ContextError(ctx, ErrInvalid); err != nil {
		return testpilot.EffectResult{}, err
	}
	result, err := h.retained.handle.Wait(ctx)
	if contextErr := ctx.Err(); contextErr != nil {
		return primitive.CloneEffectResult(result), contextErr
	}
	if completeErr := h.complete(ctx); completeErr != nil && err == nil {
		err = completeErr
	}
	return primitive.CloneEffectResult(result), err
}

func (h *reservationProxy) Cancel(ctx context.Context) error {
	if err := primitive.ContextError(ctx, ErrInvalid); err != nil {
		return err
	}
	l := h.retained.ledger
	if l == nil {
		return h.retained.handle.Cancel(ctx)
	}
	if err := l.mu.LockContext(ctx, ErrInvalid); err != nil {
		return err
	}
	if h.retained.completed || h.retained.cancelSent.Load() {
		l.mu.Unlock()
		return nil
	}
	if !h.retained.canceling.CompareAndSwap(false, true) {
		l.mu.Unlock()
		return ErrLifecycle
	}
	l.mu.Unlock()
	return l.cancel(ctx, []*retainedReservation{h.retained})
}

func (h *reservationProxy) Drain(ctx context.Context) error {
	if err := primitive.ContextError(ctx, ErrInvalid); err != nil {
		return err
	}
	err := h.retained.handle.Drain(ctx)
	if err == nil {
		err = h.complete(ctx)
	}
	return err
}

func (h *reservationProxy) complete(ctx context.Context) error {
	l := h.retained.ledger
	if l == nil {
		return nil
	}
	if err := l.mu.LockContext(ctx, ErrInvalid); err != nil {
		return err
	}
	retained := h.retained
	if !retained.completed {
		retained.completed = true
		l.activeHandles--
		delete(l.retained, retained.identity.ID)
		if state := retained.route; state != nil {
			state.authority = terminal
			l.activeRoutes--
			state.bundle.active--
			delete(l.routes, state.identity.ID)
			l.retireBundleLocked(state.bundle)
		}
	}
	l.mu.Unlock()
	return nil
}
