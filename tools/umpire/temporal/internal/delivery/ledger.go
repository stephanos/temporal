package delivery

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"sync/atomic"

	"go.temporal.io/api/workflowservice/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire"
	"google.golang.org/protobuf/proto"
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

type WorkflowBinding struct {
	Namespace, WorkflowID, WorkflowType, TaskQueue string
}

type Ledger struct {
	mu            ledgerMutex
	config        Config
	bundles       map[uint64]*bundleState
	routes        map[string]*routeState
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
	id                 uint64
	origin             umpire.Coordinate
	plan               umpire.ReservationCarrierPlan
	binding            binding
	workflow           *routeState
	nexus              map[sourceKey]*routeState
	routes             []*routeState
	responseRunID      string
	triggerDisposition TriggerDisposition
	triggerFinal       bool
	triggerCanceled    []*routeState
	parentReleased     bool
	parentCanceled     []*routeState
	active             int
}

type sourceKey struct {
	workflowEntrypoint string
	workflowOrdinal    int64
	sourceInstruction  string
}

type routeState struct {
	bundle     *bundleState
	identity   umpire.ReservationIdentity
	retained   *retainedReservation
	kind       routeKind
	source     sourceKey
	authority  authority
	activation activationData
}

type activationData struct {
	coordinate    umpire.Coordinate
	temporalRunID string
	requestID     string
}

type Bundle struct {
	ledger  *Ledger
	id      uint64
	handles []umpire.EffectHandle
}

func (b Bundle) Handles() []umpire.EffectHandle { return slices.Clone(b.handles) }

type Activation struct {
	ledger *Ledger
	state  *routeState
	data   activationData
	replay bool
}

func (a Activation) Coordinate() umpire.Coordinate { return a.data.coordinate }
func (a Activation) Reservation() umpire.ReservationIdentity {
	if a.state == nil {
		return umpire.ReservationIdentity{}
	}
	return a.state.identity
}
func (a Activation) TemporalRunID() string { return a.data.temporalRunID }
func (a Activation) RequestID() string     { return a.data.requestID }
func (a Activation) Replay() bool          { return a.replay }
func (a Activation) Handle() umpire.EffectHandle {
	if a.state == nil {
		return nil
	}
	return a.state.retained.proxy
}

type Release struct{ unused int }

func (r Release) Unused() int { return r.unused }

type TriggerDisposition uint8

type CompletionFunc func()
type QuarantineFunc func(context.Context, umpire.EffectHandle, CompletionFunc) error

const (
	TriggerSucceeded TriggerDisposition = iota + 1
	TriggerRejected
	TriggerCanceled
	TriggerNonSuccess
	TriggerUncertain
)

func (d TriggerDisposition) String() string {
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
	return &Ledger{mu: make(ledgerMutex, 1), config: config, bundles: make(map[uint64]*bundleState), routes: make(map[string]*routeState), retained: make(map[string]*retainedReservation)}, nil
}

// RetainReservation attaches lifecycle accounting before the handle is returned from Session.Reserve.
// The returned proxy is the handle that the Executor must retain and later Wait, Cancel or Drain.
func (l *Ledger) RetainReservation(ctx context.Context, handle umpire.ReservationHandle) (umpire.ReservationHandle, error) {
	if nilValue(handle) {
		return handle, ErrInvalid
	}
	cleanup := newCleanupProxy(handle)
	if err := contextError(ctx); err != nil {
		return cleanup, err
	}
	identity := handle.Identity()
	if identity.Origin.RunID != l.config.RunID || !validCoordinate(identity.Origin) || !validReservation(identity) {
		return cleanup, ErrRouteConflict
	}
	if err := l.mu.LockContext(ctx); err != nil {
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

func (l *Ledger) CreateBundle(ctx context.Context, origin umpire.Coordinate, plan umpire.ReservationCarrierPlan, workflowBinding WorkflowBinding, handles []umpire.ReservationHandle) (Bundle, error) {
	cleanup := cleanupBundle(handles)
	if err := contextError(ctx); err != nil {
		return cleanup, err
	}
	validated, ordered, err := validateBundle(l.config.RunID, origin, plan, binding(workflowBinding), handles, l.config.Limits)
	if err != nil {
		return cleanup, err
	}
	if err := l.mu.LockContext(ctx); err != nil {
		return cleanup, err
	}
	defer l.mu.Unlock()
	if l.stopped {
		return cleanup, ErrRouteStale
	}
	if l.activeRoutes > l.config.Limits.MaxRoutes-len(ordered) {
		return cleanup, ErrCapacity
	}
	if len(l.bundles) >= l.config.Limits.MaxRoutes {
		return cleanup, ErrCapacity
	}
	proxies := make([]*reservationProxy, 0, len(ordered))
	for _, handle := range ordered {
		proxy, ok := handle.(*reservationProxy)
		if !ok || proxy == nil || proxy.retained == nil || proxy.retained.ledger != l || proxy.retained.completed || proxy.retained.route != nil || l.retained[handle.Identity().ID] != proxy.retained {
			return cleanup, ErrRouteConflict
		}
		if _, exists := l.routes[handle.Identity().ID]; exists {
			return cleanup, ErrRouteConflict
		}
		proxies = append(proxies, proxy)
	}
	l.nextBundle++
	state := &bundleState{id: l.nextBundle, origin: origin, plan: clonePlan(plan), binding: binding(workflowBinding), nexus: make(map[sourceKey]*routeState), active: len(ordered)}
	byIdentity := make(map[reservationKey]*routeState, len(ordered))
	for _, proxy := range proxies {
		identity := proxy.Identity()
		routeState := &routeState{bundle: state, identity: identity, retained: proxy.retained, authority: reserved}
		proxy.retained.route = routeState
		entrypointContext := validated[reservationKey{entrypoint: identity.EntrypointID, ordinal: identity.Ordinal}]
		if entrypointContext == umpirespb.ENTRYPOINT_CONTEXT_WORKFLOW {
			routeState.kind = workflowRoute
			state.workflow = routeState
		} else {
			routeState.kind = nexusRoute
		}
		state.routes = append(state.routes, routeState)
		byIdentity[reservationKey{entrypoint: identity.EntrypointID, ordinal: identity.Ordinal}] = routeState
		l.routes[identity.ID] = routeState
	}
	for _, route := range plan.Routes {
		key := sourceKey{workflowEntrypoint: route.WorkflowEntrypointID, workflowOrdinal: route.WorkflowOrdinal, sourceInstruction: route.SourceInstructionID}
		handler := byIdentity[reservationKey{entrypoint: route.HandlerEntrypointID, ordinal: route.HandlerOrdinal}]
		handler.source = key
		state.nexus[key] = handler
	}
	l.bundles[state.id] = state
	l.activeRoutes += len(ordered)
	bundle := Bundle{ledger: l, id: state.id, handles: make([]umpire.EffectHandle, len(state.routes))}
	for i, route := range state.routes {
		bundle.handles[i] = route.retained.proxy
	}
	return bundle, nil
}

type reservationKey struct {
	entrypoint string
	ordinal    int64
}

func validateBundle(runID string, origin umpire.Coordinate, plan umpire.ReservationCarrierPlan, workflowBinding binding, handles []umpire.ReservationHandle, limits Limits) (map[reservationKey]umpirespb.EntrypointContext, []umpire.ReservationHandle, error) {
	if origin.RunID != runID || !validCoordinate(origin) || !validRouteText(plan.EndpointRoleID) || plan.Method != startWorkflowPath || !validBinding(workflowBinding) || len(plan.Reservations) > limits.MaxRoutes || len(plan.Routes) > limits.MaxRoutes {
		return nil, nil, ErrInvalid
	}
	expected, handlerCount, err := validateTopology(plan, limits)
	if err != nil {
		return nil, nil, err
	}
	ordered, err := validateHandles(origin, expected, handles)
	if err != nil {
		return nil, nil, err
	}
	if err := validateRoutes(plan, expected, handlerCount); err != nil {
		return nil, nil, err
	}
	return expected, ordered, nil
}

func validateTopology(plan umpire.ReservationCarrierPlan, limits Limits) (map[reservationKey]umpirespb.EntrypointContext, int, error) {
	expected := make(map[reservationKey]umpirespb.EntrypointContext)
	workflowCount := 0
	handlerCount := 0
	for _, topology := range plan.Reservations {
		if !validRouteText(topology.EntrypointID) || topology.Count <= 0 || topology.Count > int64(limits.MaxRoutes-len(expected)) {
			return nil, 0, ErrInvalid
		}
		if topology.Context != umpirespb.ENTRYPOINT_CONTEXT_WORKFLOW && topology.Context != umpirespb.ENTRYPOINT_CONTEXT_NEXUS_HANDLER {
			return nil, 0, ErrInvalid
		}
		for ordinal := int64(0); ordinal < topology.Count; ordinal++ {
			key := reservationKey{entrypoint: topology.EntrypointID, ordinal: ordinal}
			if _, duplicate := expected[key]; duplicate {
				return nil, 0, ErrInvalid
			}
			expected[key] = topology.Context
		}
		if topology.Context == umpirespb.ENTRYPOINT_CONTEXT_WORKFLOW {
			workflowCount += int(topology.Count)
		} else {
			handlerCount += int(topology.Count)
		}
	}
	if workflowCount != 1 {
		return nil, 0, ErrInvalid
	}
	return expected, handlerCount, nil
}

func validateHandles(origin umpire.Coordinate, expected map[reservationKey]umpirespb.EntrypointContext, handles []umpire.ReservationHandle) ([]umpire.ReservationHandle, error) {
	if len(handles) != len(expected) {
		return nil, ErrInvalid
	}
	ordered := make([]umpire.ReservationHandle, 0, len(handles))
	seenKeys := make(map[reservationKey]bool, len(handles))
	seenIDs := make(map[string]bool, len(handles))
	for _, handle := range handles {
		if nilValue(handle) {
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

func validateRoutes(plan umpire.ReservationCarrierPlan, expected map[reservationKey]umpirespb.EntrypointContext, handlerCount int) error {
	seenSources := make(map[sourceKey]bool, len(plan.Routes))
	seenHandlers := make(map[reservationKey]bool, len(plan.Routes))
	for _, planned := range plan.Routes {
		source := sourceKey{workflowEntrypoint: planned.WorkflowEntrypointID, workflowOrdinal: planned.WorkflowOrdinal, sourceInstruction: planned.SourceInstructionID}
		handler := reservationKey{entrypoint: planned.HandlerEntrypointID, ordinal: planned.HandlerOrdinal}
		if expected[reservationKey{entrypoint: planned.WorkflowEntrypointID, ordinal: planned.WorkflowOrdinal}] != umpirespb.ENTRYPOINT_CONTEXT_WORKFLOW || expected[handler] != umpirespb.ENTRYPOINT_CONTEXT_NEXUS_HANDLER || !validRouteText(planned.SourceInstructionID) || seenSources[source] || seenHandlers[handler] {
			return ErrRouteConflict
		}
		seenSources[source] = true
		seenHandlers[handler] = true
	}
	if len(seenHandlers) != handlerCount {
		return ErrInvalid
	}
	return nil
}

func clonePlan(plan umpire.ReservationCarrierPlan) umpire.ReservationCarrierPlan {
	plan.Reservations = slices.Clone(plan.Reservations)
	plan.Routes = slices.Clone(plan.Routes)
	return plan
}

func cleanupBundle(handles []umpire.ReservationHandle) Bundle {
	result := Bundle{}
	for _, handle := range handles {
		if !nilValue(handle) {
			result.handles = append(result.handles, handle)
		}
	}
	return result
}

func (l *Ledger) PinStartResponse(ctx context.Context, bundle Bundle, response *workflowservice.StartWorkflowExecutionResponse) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	if nilValue(response) || !validRouteText(response.GetRunId()) {
		return ErrInvalid
	}
	if err := l.mu.LockContext(ctx); err != nil {
		return err
	}
	defer l.mu.Unlock()
	state, err := l.bundleLocked(bundle)
	if err != nil {
		return err
	}
	if l.stopped || state.triggerFinal && state.triggerDisposition != TriggerSucceeded {
		l.diagnoseLocked()
		return ErrRouteStale
	}
	runID := response.GetRunId()
	if state.responseRunID != "" && state.responseRunID != runID || state.workflow.activation.temporalRunID != "" && state.workflow.activation.temporalRunID != runID {
		return ErrRouteConflict
	}
	state.responseRunID = runID
	return nil
}

func (l *Ledger) TriggerTerminal(ctx context.Context, bundle Bundle, disposition TriggerDisposition) (Release, error) {
	if err := contextError(ctx); err != nil {
		return Release{}, err
	}
	if disposition < TriggerSucceeded || disposition > TriggerUncertain {
		return Release{}, ErrInvalid
	}
	if err := l.mu.LockContext(ctx); err != nil {
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
		if state.triggerDisposition != disposition {
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
		state.triggerDisposition = disposition
		l.retireBundleLocked(state)
		l.mu.Unlock()
		return Release{}, nil
	}
	state.triggerFinal = true
	state.triggerDisposition = disposition
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
	if err := contextError(ctx); err != nil {
		return Release{}, err
	}
	if err := l.mu.LockContext(ctx); err != nil {
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
	pending := l.pendingCancellationLocked(routes)
	l.mu.Unlock()
	return release, l.cancel(ctx, pending)
}

func (l *Ledger) Stop(ctx context.Context) (Release, error) {
	if err := contextError(ctx); err != nil {
		return Release{}, err
	}
	if err := l.mu.LockContext(ctx); err != nil {
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

func (l *Ledger) Quarantine(ctx context.Context, handle umpire.EffectHandle, quarantine QuarantineFunc) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	if nilValue(handle) || nilValue(quarantine) {
		return ErrInvalid
	}
	proxy, ok := handle.(*reservationProxy)
	if !ok || proxy == nil || proxy.retained == nil || proxy.retained.ledger != l {
		return ErrRouteCrossed
	}
	if err := l.mu.LockContext(ctx); err != nil {
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
	if err := contextError(ctx); err != nil {
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
	identity     umpire.ReservationIdentity
	handle       umpire.ReservationHandle
	proxy        *reservationProxy
	route        *routeState
	completed    bool
	canceling    atomic.Bool
	cancelSent   atomic.Bool
	quarantining atomic.Bool
	quarantined  atomic.Bool
}

type reservationProxy struct{ retained *retainedReservation }

func newCleanupProxy(handle umpire.ReservationHandle) *reservationProxy {
	retained := &retainedReservation{identity: handle.Identity(), handle: handle}
	proxy := &reservationProxy{retained: retained}
	retained.proxy = proxy
	return proxy
}

func (h *reservationProxy) Identity() umpire.ReservationIdentity { return h.retained.identity }

func (h *reservationProxy) Consume(context.Context) (umpire.Coordinate, error) {
	return umpire.Coordinate{}, ErrInvalid
}

func (h *reservationProxy) Wait(ctx context.Context) (umpire.EffectResult, error) {
	if err := contextError(ctx); err != nil {
		return umpire.EffectResult{}, err
	}
	result, err := h.retained.handle.Wait(ctx)
	if contextErr := ctx.Err(); contextErr != nil {
		return copyEffectResult(result), contextErr
	}
	if completeErr := h.complete(ctx); completeErr != nil && err == nil {
		err = completeErr
	}
	return copyEffectResult(result), err
}

func (h *reservationProxy) Cancel(ctx context.Context) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	l := h.retained.ledger
	if l == nil {
		return h.retained.handle.Cancel(ctx)
	}
	if err := l.mu.LockContext(ctx); err != nil {
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
	if err := contextError(ctx); err != nil {
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
	if err := l.mu.LockContext(ctx); err != nil {
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

func copyEffectResult(result umpire.EffectResult) umpire.EffectResult {
	return umpire.EffectResult{Outcome: proto.CloneOf(result.Outcome), Response: proto.Clone(result.Response)}
}

func nilValue(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice, reflect.UnsafePointer:
		return reflected.IsNil()
	default:
		return false
	}
}

func contextError(ctx context.Context) error {
	if nilValue(ctx) {
		return ErrInvalid
	}
	return ctx.Err()
}

type ledgerMutex chan struct{}

func (m ledgerMutex) Lock()   { m <- struct{}{} }
func (m ledgerMutex) Unlock() { <-m }
func (m ledgerMutex) LockContext(ctx context.Context) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case m <- struct{}{}:
	}
	if err := ctx.Err(); err != nil {
		m.Unlock()
		return err
	}
	return nil
}
