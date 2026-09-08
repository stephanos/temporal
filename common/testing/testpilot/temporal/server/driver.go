// Package server owns authorized controller transports for the Temporal Driver.
package server

import (
	"context"
	"errors"
	"reflect"
	"slices"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

// Endpoint is Driver configuration, never a Case or retained Run value.
type Endpoint struct {
	Target            string
	Credentials       credentials.TransportCredentials
	PerRPCCredentials credentials.PerRPCCredentials
	Metadata          metadata.MD
}

type Options struct {
	Profile   testpilot.ProfileSpec
	Endpoints map[string]Endpoint
}

type endpoint struct {
	connection *grpc.ClientConn
	metadata   metadata.MD
	methods    map[string]bool
}

// Driver shares channels and finite capacity across independent logical sessions.
// Profile MaxActivations bounds live sessions; MaxAttempts bounds all unfinished effects,
// including quarantine, across those sessions.
type Driver struct {
	profile   testpilot.ProfileSpec
	endpoints map[string]endpoint
	mu        hostMutex
	sessions  map[string]*Session
	effects   int64
	closed    bool
}

var (
	errInvalid      = errors.New("invalid server Driver input")
	errClosed       = errors.New("server Driver session is closed")
	errCapacity     = errors.New("server Driver capacity exhausted")
	errUnauthorized = errors.New("server Driver operation is not authorized")
)

func New(options Options) (*Driver, error) {
	p := options.Profile
	l := p.ProgramLimits
	if !validProfile(p) {
		return nil, errInvalid
	}
	h := &Driver{mu: make(hostMutex, 1), profile: cloneProfile(p), endpoints: make(map[string]endpoint), sessions: make(map[string]*Session)}
	for _, role := range p.Roles {
		if role.Kind != testpilotspb.ROLE_KIND_ENDPOINT || len(role.Methods) == 0 {
			continue
		}
		config, ok := options.Endpoints[role.ID]
		if !ok || config.Target == "" || nilValue(config.Credentials) || len(role.Methods) > 10000 {
			return nil, errors.Join(errInvalid, h.closeConnections())
		}
		if _, duplicate := h.endpoints[role.ID]; duplicate {
			return nil, errors.Join(errInvalid, h.closeConnections())
		}
		opts := []grpc.DialOption{grpc.WithTransportCredentials(config.Credentials.Clone()), grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(int(l.MaxRequestBytes)), grpc.MaxCallRecvMsgSize(int(l.MaxResponseBytes)))}
		if config.PerRPCCredentials != nil {
			if nilValue(config.PerRPCCredentials) {
				return nil, errors.Join(errInvalid, h.closeConnections())
			}
			opts = append(opts, grpc.WithPerRPCCredentials(config.PerRPCCredentials))
		}
		connection, err := grpc.NewClient(config.Target, opts...)
		if err != nil {
			return nil, errors.Join(errInvalid, h.closeConnections())
		}
		methods := make(map[string]bool, len(role.Methods))
		for _, method := range role.Methods {
			methods[method] = true
		}
		h.endpoints[role.ID] = endpoint{connection: connection, metadata: config.Metadata.Copy(), methods: methods}
	}
	return h, nil
}

func validProfile(p testpilot.ProfileSpec) bool {
	l := p.ProgramLimits
	if p.Identity == "" || len(p.Identity) > 256 || len(p.Capabilities) > 7 || p.Catalog.Identity() == "" || l == nil || l.MaxActivations <= 0 || l.MaxActivations > 100000 || l.MaxAttempts <= 0 || l.MaxAttempts > 100000 || l.MaxNodes <= 0 || l.MaxNodes > 10000 || l.MaxRequestBytes <= 0 || l.MaxRequestBytes > 16<<20 || l.MaxResponseBytes <= 0 || l.MaxResponseBytes > 16<<20 || l.MaxTotalDurationMilliseconds <= 0 || l.MaxTotalDurationMilliseconds > 86400000 || l.MaxCleanupDurationMilliseconds <= 0 || l.MaxCleanupDurationMilliseconds > 86400000 || len(p.Roles) > 10000 {
		return false
	}
	total := 0
	for _, role := range p.Roles {
		if len(role.Methods) > 10000 || len(role.Methods) > 100000-total {
			return false
		}
		total += len(role.Methods)
	}
	return true
}

func cloneProfile(p testpilot.ProfileSpec) testpilot.ProfileSpec {
	p.Roles = slices.Clone(p.Roles)
	for i := range p.Roles {
		p.Roles[i].Methods = slices.Clone(p.Roles[i].Methods)
	}
	p.Capabilities = slices.Clone(p.Capabilities)
	p.EnvironmentBindings = slices.Clone(p.EnvironmentBindings)
	p.ProgramLimits = proto.CloneOf(p.ProgramLimits)
	p.ContractLimits = proto.CloneOf(p.ContractLimits)
	return p
}

func (h *Driver) Snapshot() testpilot.ProfileSpec { return cloneProfile(h.profile) }
func (h *Driver) Identity(ctx context.Context) (testpilot.DriverIdentity, error) {
	if err := contextError(ctx); err != nil {
		return testpilot.DriverIdentity{}, err
	}
	fingerprint, err := h.profile.BindingFingerprint()
	if err != nil {
		return testpilot.DriverIdentity{}, err
	}
	return testpilot.DriverIdentity{Profile: h.profile.Identity, Catalog: h.profile.Catalog.Identity(), Bindings: fingerprint}, nil
}

func (h *Driver) Validate(ctx context.Context, program testpilot.PreparedProgram) error {
	if h == nil {
		return errInvalid
	}
	if err := contextError(ctx); err != nil {
		return err
	}
	if program.Snapshot() == nil {
		return errInvalid
	}
	return nil
}

func (h *Driver) Open(ctx context.Context, runID string, program testpilot.PreparedProgram) (testpilot.Session, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	return h.open(ctx, runID, program.Snapshot())
}

// OpenSession retains the concrete server session for composite Driver wiring.
func (h *Driver) OpenSession(ctx context.Context, runID string, program testpilot.PreparedProgram) (*Session, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	return h.open(ctx, runID, program.Snapshot())
}

func (h *Driver) open(ctx context.Context, runID string, program *testpilotspb.Program) (*Session, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	if runID == "" || len(runID) > 256 || program == nil || program.Limits == nil {
		return nil, errInvalid
	}
	if err := h.mu.LockContext(ctx); err != nil {
		return nil, err
	}
	defer h.mu.Unlock()
	if h.closed {
		return nil, errClosed
	}
	if _, exists := h.sessions[runID]; exists {
		return nil, errInvalid
	}
	if int64(len(h.sessions)) >= h.profile.ProgramLimits.MaxActivations {
		return nil, errCapacity
	}
	s := &Session{host: h, runID: runID, effects: make(map[*effect]struct{}), started: make(map[testpilot.Coordinate]struct{}), entries: make(map[string]struct{}), controllers: make(map[string]struct{}), nodes: make(map[nodeKey]*testpilotspb.InstructionDefinition), slots: make(map[string]*capabilitySlot), capabilities: make(map[*opaqueCapability]struct{}), closedSignal: make(chan struct{})}
	for _, entry := range program.Entrypoints {
		s.entries[entry.EntrypointId] = struct{}{}
		if entry.GetController() != nil {
			s.controllers[entry.EntrypointId] = struct{}{}
		}
		for _, node := range entry.Instructions {
			s.nodes[nodeKey{entry.EntrypointId, node.InstructionId}] = proto.CloneOf(node)
		}
	}
	if cleanup := program.Cleanup; cleanup != nil {
		s.entries[cleanup.EntrypointId] = struct{}{}
		s.controllers[cleanup.EntrypointId] = struct{}{}
		for _, node := range cleanup.Instructions {
			s.nodes[nodeKey{cleanup.EntrypointId, node.InstructionId}] = proto.CloneOf(node)
		}
	}
	for _, slot := range program.Slots {
		if slot.GetOpaqueCapability() != nil {
			s.slots[slot.SlotId] = &capabilitySlot{ready: make(chan struct{})}
		}
	}
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	h.sessions[runID] = s
	return s, nil
}

func (h *Driver) Close(ctx context.Context) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	if err := h.mu.LockContext(ctx); err != nil {
		return err
	}
	if h.closed {
		h.mu.Unlock()
		return nil
	}
	h.closed = true
	for _, session := range h.sessions {
		session.closeLocked()
	}
	h.mu.Unlock()
	return h.closeConnections()
}

func (h *Driver) closeConnections() error {
	var result error
	for _, e := range h.endpoints {
		if err := e.connection.Close(); err != nil {
			result = errors.New("server Driver channel close failed")
		}
	}
	return result
}

func nilValue(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice, reflect.UnsafePointer:
		return v.IsNil()
	default:
		return false
	}
}
func contextError(ctx context.Context) error {
	if nilValue(ctx) {
		return errInvalid
	}
	return ctx.Err()
}

// Driver operations can abandon serialization without a timeout goroutine. Internal completion
// uses Lock so canceled callers cannot prevent capacity from being released.
type hostMutex chan struct{}

func (m hostMutex) Lock()   { m <- struct{}{} }
func (m hostMutex) Unlock() { <-m }
func (m hostMutex) LockContext(ctx context.Context) error {
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
