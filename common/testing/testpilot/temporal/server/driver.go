// Package server owns authorized controller transports for the Temporal Driver.
package server

import (
	"context"
	"errors"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal/internal/primitive"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
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
	mu        primitive.Mutex
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
	h := &Driver{mu: primitive.NewMutex(), profile: p.Snapshot(), endpoints: make(map[string]endpoint), sessions: make(map[string]*Session)}
	for _, role := range p.Roles {
		if role.Kind != testpilotspb.ROLE_KIND_ENDPOINT || len(role.Methods) == 0 {
			continue
		}
		config, ok := options.Endpoints[role.ID]
		if !ok || config.Target == "" || primitive.NilValue(config.Credentials) || len(role.Methods) > 10000 {
			return nil, errors.Join(errInvalid, h.closeConnections())
		}
		if _, duplicate := h.endpoints[role.ID]; duplicate {
			return nil, errors.Join(errInvalid, h.closeConnections())
		}
		opts := []grpc.DialOption{grpc.WithTransportCredentials(config.Credentials.Clone()), grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(int(l.MaxRequestBytes)), grpc.MaxCallRecvMsgSize(int(l.MaxResponseBytes)))}
		if config.PerRPCCredentials != nil {
			if primitive.NilValue(config.PerRPCCredentials) {
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
	if p.Identity == "" || len(p.Identity) > 256 || len(p.Opcodes) > int(testpilot.MaxOpcode) || p.Catalog.Identity() == "" || l == nil || len(p.Roles) > 10000 || p.BoundScale.Percent() < 100 {
		return false
	}
	// The scaled ceilings are the ones preparation admits a Program under and a Session caps at.
	if !testpilot.WithinProgramCeiling(p.BoundScale.Ceilings(l)) {
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

func (h *Driver) Snapshot() testpilot.ProfileSpec { return h.profile.Snapshot() }
func (h *Driver) Identity(ctx context.Context) (testpilot.DriverIdentity, error) {
	if err := primitive.ContextError(ctx, errInvalid); err != nil {
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
	if err := primitive.ContextError(ctx, errInvalid); err != nil {
		return err
	}
	if program.Snapshot() == nil {
		return errInvalid
	}
	return nil
}

func (h *Driver) Open(ctx context.Context, runID string, program testpilot.PreparedProgram) (testpilot.Session, error) {
	s, err := h.OpenSession(ctx, runID, program)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// OpenSession retains the concrete server session for composite Driver wiring. The session indexes
// the prepared instruction plans, so it reads node bounds and defaults as preparation resolved them.
func (h *Driver) OpenSession(ctx context.Context, runID string, program testpilot.PreparedProgram) (*Session, error) {
	if err := primitive.ContextError(ctx, errInvalid); err != nil {
		return nil, err
	}
	source := program.Snapshot()
	if runID == "" || len(runID) > 256 || source == nil {
		return nil, errInvalid
	}
	s := &Session{host: h, runID: runID, effects: make(map[*effect]struct{}), started: make(map[testpilot.Coordinate]struct{}), entries: make(map[string]struct{}), controllers: make(map[string]struct{}), instructions: make(map[nodeKey]testpilot.InstructionPlan), slots: make(map[string]*handleSlot), handles: make(map[*opaqueHandle]struct{}), closedSignal: make(chan struct{})}
	index := func(entry testpilot.EntrypointPlan, controller bool) {
		s.entries[entry.ID()] = struct{}{}
		if controller {
			s.controllers[entry.ID()] = struct{}{}
		}
		for _, plan := range entry.Instructions() {
			s.instructions[nodeKey{entry.ID(), plan.Source().GetInstructionId()}] = plan
		}
	}
	for _, entry := range program.Entrypoints() {
		index(entry, entry.Kind() == testpilot.ControllerEntrypoint)
	}
	if cleanup, ok := program.Cleanup(); ok {
		index(cleanup, true)
	}
	for _, slot := range source.Slots {
		if slot.GetOpaqueHandle() != nil {
			s.slots[slot.SlotId] = &handleSlot{ready: make(chan struct{})}
		}
	}
	if err := h.mu.LockContext(ctx, errInvalid); err != nil {
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
	if err := primitive.ContextError(ctx, errInvalid); err != nil {
		return nil, err
	}
	h.sessions[runID] = s
	return s, nil
}

func (h *Driver) Close(ctx context.Context) error {
	if err := primitive.ContextError(ctx, errInvalid); err != nil {
		return err
	}
	if err := h.mu.LockContext(ctx, errInvalid); err != nil {
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
