// Package server owns authorized controller transports for the Temporal Host.
package server

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"time"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

// Endpoint is Host configuration, never a Case or retained Run value.
type Endpoint struct {
	Target            string
	Credentials       credentials.TransportCredentials
	PerRPCCredentials credentials.PerRPCCredentials
	Metadata          metadata.MD
}

type Options struct {
	Profile               umpire.ProfileSpec
	Endpoints             map[string]Endpoint
	SystemCallbackBaseURL string
	// HTTPClient and its transport must honor request cancellation. Redirects are disabled.
	HTTPClient *http.Client
}

type endpoint struct {
	connection *grpc.ClientConn
	metadata   metadata.MD
	methods    map[string]bool
}

// Host shares channels and finite capacity across independent logical sessions.
// Profile MaxActivations bounds live sessions; MaxAttempts bounds all unfinished effects,
// including quarantine, across those sessions.
type Host struct {
	profile               umpire.ProfileSpec
	endpoints             map[string]endpoint
	httpClient            http.Client
	systemCallbackBaseURL *url.URL
	mu                    hostMutex
	sessions              map[string]*Session
	effects               int64
	closed                bool
}

var (
	errInvalid      = errors.New("invalid server Host input")
	errClosed       = errors.New("server Host session is closed")
	errCapacity     = errors.New("server Host capacity exhausted")
	errUnauthorized = errors.New("server Host operation is not authorized")
)

func New(options Options) (*Host, error) {
	p := options.Profile
	l := p.ProgramLimits
	if !validProfile(p) {
		return nil, errInvalid
	}
	callbackBaseURL, err := parseSystemCallbackBaseURL(options.SystemCallbackBaseURL)
	if err != nil {
		return nil, errInvalid
	}
	h := &Host{mu: make(hostMutex, 1), profile: cloneProfile(p), endpoints: make(map[string]endpoint), sessions: make(map[string]*Session), systemCallbackBaseURL: callbackBaseURL}
	if options.HTTPClient != nil {
		h.httpClient = *options.HTTPClient
	}
	transport := h.httpClient.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	if standard, ok := transport.(*http.Transport); ok && standard != nil {
		owned := standard.Clone()
		owned.MaxResponseHeaderBytes = l.MaxResponseBytes
		owned.MaxConnsPerHost = int(l.MaxAttempts)
		owned.MaxIdleConns = int(l.MaxAttempts)
		owned.MaxIdleConnsPerHost = int(l.MaxAttempts)
		h.httpClient.Transport = owned
	} else if nilValue(transport) {
		return nil, errInvalid
	}
	h.httpClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	h.httpClient.Timeout = time.Duration(l.MaxTotalDurationMilliseconds) * time.Millisecond
	for _, role := range p.Roles {
		if role.Kind != umpirespb.SYMBOLIC_ROLE_KIND_ENDPOINT || len(role.Methods) == 0 {
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

func parseSystemCallbackBaseURL(raw string) (*url.URL, error) {
	if raw == "" {
		return nil, nil
	}
	base, err := url.Parse(raw)
	if err != nil || base.Host == "" || (base.Scheme != "http" && base.Scheme != "https") || base.User != nil || base.Fragment != "" || base.RawQuery != "" || base.ForceQuery || base.Opaque != "" || base.RawPath != "" || (base.Path != "" && base.Path != "/") {
		return nil, errInvalid
	}
	base.Path = ""
	return base, nil
}

func validProfile(p umpire.ProfileSpec) bool {
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

func cloneProfile(p umpire.ProfileSpec) umpire.ProfileSpec {
	p.Roles = slices.Clone(p.Roles)
	for i := range p.Roles {
		p.Roles[i].Methods = slices.Clone(p.Roles[i].Methods)
	}
	p.Capabilities = slices.Clone(p.Capabilities)
	p.ProgramLimits = proto.CloneOf(p.ProgramLimits)
	p.ContractLimits = proto.CloneOf(p.ContractLimits)
	return p
}

func (h *Host) Snapshot() umpire.ProfileSpec { return cloneProfile(h.profile) }
func (h *Host) Identity(ctx context.Context) (umpire.HostIdentity, error) {
	if err := contextError(ctx); err != nil {
		return umpire.HostIdentity{}, err
	}
	return umpire.HostIdentity{Profile: h.profile.Identity, Catalog: h.profile.Catalog.Identity()}, nil
}

func (h *Host) Open(ctx context.Context, runID string, program umpire.PreparedProgram) (umpire.Session, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	return h.open(ctx, runID, program.Snapshot())
}

// OpenSession retains the concrete server session for composite Host wiring.
func (h *Host) OpenSession(ctx context.Context, runID string, program umpire.PreparedProgram) (*Session, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	return h.open(ctx, runID, program.Snapshot())
}

func (h *Host) open(ctx context.Context, runID string, program *umpirespb.Program) (*Session, error) {
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
	s := &Session{host: h, runID: runID, effects: make(map[*effect]struct{}), started: make(map[umpire.Coordinate]struct{}), entries: make(map[string]umpirespb.EntrypointContext), nodes: make(map[nodeKey]*umpirespb.InstructionNode), slots: make(map[string]*capabilitySlot), capabilities: make(map[*completionCapability]struct{}), closedSignal: make(chan struct{})}
	for _, entry := range program.Entrypoints {
		s.entries[entry.EntrypointId] = entry.Context
		for _, node := range entry.Nodes {
			s.nodes[nodeKey{entry.EntrypointId, node.InstructionId}] = proto.CloneOf(node)
		}
	}
	if cleanup := program.Cleanup; cleanup != nil {
		s.entries[cleanup.EntrypointId] = cleanup.Context
		for _, node := range cleanup.Nodes {
			s.nodes[nodeKey{cleanup.EntrypointId, node.InstructionId}] = proto.CloneOf(node)
		}
	}
	for _, slot := range program.Slots {
		if slot.GetType().GetSingular().GetOpaqueCapability() != nil {
			s.slots[slot.SlotId] = &capabilitySlot{ready: make(chan struct{})}
		}
	}
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	h.sessions[runID] = s
	return s, nil
}

func (h *Host) Close(ctx context.Context) error {
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
	h.httpClient.CloseIdleConnections()
	return h.closeConnections()
}

func (h *Host) closeConnections() error {
	var result error
	for _, e := range h.endpoints {
		if err := e.connection.Close(); err != nil {
			result = errors.New("server Host channel close failed")
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

// Host operations can abandon serialization without a timeout goroutine. Internal completion
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
