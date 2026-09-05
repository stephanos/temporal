package server

import (
	"context"
	"errors"
	"io"
	"maps"
	"net/http"
	"net/url"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/nexus-rpc/sdk-go/nexus"
	commonpb "go.temporal.io/api/common/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	commonnexus "go.temporal.io/server/common/nexus"
	"go.temporal.io/server/common/nexus/nexusrpc"
	"go.temporal.io/server/tools/umpire"
	"google.golang.org/protobuf/proto"
)

// CompletionInfo is supplied only by trusted Host wiring at worker activation. It must never
// be put in a Case, ordinary Slot, Run event or diagnostic.
type CompletionInfo struct {
	URL            string
	Header         nexus.Header
	OperationToken string
	StartTime      time.Time
}

type completionCapability struct {
	session   *Session
	origin    umpire.Coordinate
	info      CompletionInfo
	used      bool
	published string
}

type capabilitySlot struct {
	ready      chan struct{}
	capability *completionCapability
	claim      *completionClaim
}

type completionClaim struct {
	capability *completionCapability
	context    context.Context
	released   atomic.Bool
}

// NewCompletionCapability is the injection seam for composite Host wiring. The worker can
// receive this bound function without importing the server package. Minting does no target I/O.
func (s *Session) NewCompletionCapability(ctx context.Context, origin umpire.Coordinate, info CompletionInfo) (umpire.OpaqueCapability, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	if origin.RunID != s.runID || origin.ActivationID == "" || len(origin.ActivationID) > 256 || s.entries[origin.EntrypointID] != umpirespb.ENTRYPOINT_CONTEXT_NEXUS_HANDLER {
		return nil, errUnauthorized
	}
	target, err := url.Parse(info.URL)
	if err == nil && (info.URL == commonnexus.SystemCallbackURL || info.URL == commonnexus.PathCompletionCallbackNoIdentifier) && s.host.systemCallbackBaseURL != nil {
		target = s.host.systemCallbackBaseURL.ResolveReference(&url.URL{Path: commonnexus.PathCompletionCallbackNoIdentifier})
		info.URL = target.String()
	}
	if err != nil || target.Host == "" || (target.Scheme != "http" && target.Scheme != "https") || target.User != nil || target.Fragment != "" || info.OperationToken == "" {
		return nil, errInvalid
	}
	size := len(info.URL) + len(info.OperationToken)
	for k, v := range info.Header {
		size += len(k) + len(v)
	}
	if int64(size) > s.host.profile.ProgramLimits.MaxRequestBytes {
		return nil, errCapacity
	}
	if err := s.host.mu.LockContext(ctx); err != nil {
		return nil, err
	}
	defer s.host.mu.Unlock()
	if s.closed {
		return nil, errClosed
	}
	if s.minted >= s.host.profile.ProgramLimits.MaxActivations {
		return nil, errCapacity
	}
	info.Header = maps.Clone(info.Header)
	capability := &completionCapability{session: s, origin: origin, info: info}
	s.capabilities[capability] = struct{}{}
	s.minted++
	return capability, nil
}

func (s *Session) CompleteNexusOperation(ctx context.Context, c umpire.Coordinate, opaque umpire.OpaqueCapability, value *umpirespb.Value) (umpire.EffectHandle, error) {
	claim, ok := opaque.(*completionClaim)
	if !ok || claim == nil || claim.capability == nil || claim.capability.session != s {
		return nil, errUnauthorized
	}
	accepted := false
	defer func() {
		if !accepted {
			claim.released.Store(true)
		}
	}()
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	n, err := s.controllerNode(c)
	if err != nil {
		return nil, err
	}
	if n.GetInstruction().GetCompleteNexusOperation() == nil || value == nil || value.Value == nil {
		return nil, errUnauthorized
	}
	if int64(proto.Size(value)) > s.host.profile.ProgramLimits.MaxRequestBytes {
		return nil, errCapacity
	}
	data, err := proto.Marshal(value)
	if err != nil {
		return nil, errInvalid
	}
	if err := s.host.mu.LockContext(ctx); err != nil {
		return nil, err
	}
	defer s.host.mu.Unlock()
	capability := claim.capability
	slot := s.slots[capability.published]
	if capability.used || claim.released.Load() || claim.context.Err() != nil || slot == nil || slot.claim != claim {
		return nil, errUnauthorized
	}
	if _, exists := s.capabilities[capability]; !exists {
		return nil, errUnauthorized
	}
	info := capability.info
	handle, err := s.startLocked(ctx, c, n.Bounds, func(ctx context.Context) umpire.EffectResult {
		return s.complete(ctx, info, data, min(n.Bounds.MaxResponseBytes, s.host.profile.ProgramLimits.MaxResponseBytes))
	})
	if err != nil {
		return nil, err
	}
	accepted = true
	capability.used = true
	capability.info = CompletionInfo{}
	delete(s.capabilities, capability)
	return handle, nil
}

func (s *Session) complete(ctx context.Context, info CompletionInfo, data []byte, maxResponseBytes int64) umpire.EffectResult {
	var body *boundedBody
	var protocolCode int
	caller := func(request *http.Request) (*http.Response, error) {
		response, err := s.host.httpClient.Do(request)
		if err != nil {
			return nil, err
		}
		body = &boundedBody{ReadCloser: response.Body, remaining: maxResponseBytes}
		response.Body = body
		protocolCode = response.StatusCode
		return response, nil
	}
	client := nexusrpc.NewCompletionHTTPClient(nexusrpc.CompletionHTTPClientOptions{HTTPCaller: caller, Serializer: commonnexus.PayloadSerializer})
	payload := &commonpb.Payload{Metadata: map[string][]byte{"encoding": []byte("binary/protobuf"), "messageType": []byte("temporal.server.api.umpire.v1.Value")}, Data: data}
	err := client.CompleteOperation(ctx, info.URL, nexusrpc.CompleteOperationOptions{Header: info.Header, OperationToken: info.OperationToken, StartTime: info.StartTime, Result: payload})
	if body != nil {
		closeErr := body.Close()
		if err == nil {
			err = closeErr
		}
		if body.readErr != nil {
			err = body.readErr
		}
		if body.exceeded {
			err = errCapacity
		}
	}
	outcome := &umpirespb.InstructionOutcome{Status: umpirespb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, ProtocolCode: "ok"}
	if err != nil {
		outcome.Status = umpirespb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_NON_SUCCESS
		outcome.ProtocolCode = "transport_failure"
		if protocolCode != 0 {
			outcome.ProtocolCode = "http_" + strconv.Itoa(protocolCode)
		}
		if errors.Is(err, errCapacity) {
			outcome.ProtocolCode = "resource_exhausted"
		}
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			outcome.Status = umpirespb.INSTRUCTION_OUTCOME_STATUS_TIMED_OUT
			outcome.ProtocolCode = "deadline_exceeded"
		}
		if errors.Is(ctx.Err(), context.Canceled) {
			outcome.Status = umpirespb.INSTRUCTION_OUTCOME_STATUS_CANCELED
			outcome.ProtocolCode = "canceled"
		}
	}
	return umpire.EffectResult{Outcome: outcome}
}

type boundedBody struct {
	readErr error
	io.ReadCloser
	remaining        int64
	exceeded, closed bool
}

func (b *boundedBody) Read(p []byte) (int, error) {
	if int64(len(p)) > b.remaining+1 {
		p = p[:b.remaining+1]
	}
	n, err := b.ReadCloser.Read(p)
	if err != nil && err != io.EOF {
		b.readErr = err
	}
	if int64(n) > b.remaining {
		b.exceeded = true
		return 0, errCapacity
	}
	b.remaining -= int64(n)
	return n, err
}
func (b *boundedBody) Close() error {
	if b.closed {
		return nil
	}
	b.closed = true
	return b.ReadCloser.Close()
}

func (s *Session) Bridge(ctx context.Context) (umpire.CapabilityBridge, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	if err := s.host.mu.LockContext(ctx); err != nil {
		return nil, err
	}
	defer s.host.mu.Unlock()
	if s.closed {
		return nil, errClosed
	}
	return s, nil
}
func (s *Session) Publish(ctx context.Context, c umpire.Coordinate, slotID string, opaque umpire.OpaqueCapability) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	if err := s.host.mu.LockContext(ctx); err != nil {
		return err
	}
	defer s.host.mu.Unlock()
	if s.closed {
		return errClosed
	}
	slot := s.slots[slotID]
	capability, ok := opaque.(*completionCapability)
	if slot == nil || !ok || capability == nil || capability.session != s || capability.origin != c || capability.used {
		return errUnauthorized
	}
	if _, exists := s.capabilities[capability]; !exists {
		return errUnauthorized
	}
	if slot.claim != nil || slot.capability != nil && slot.capability != capability || capability.published != "" && capability.published != slotID {
		return errInvalid
	}
	if slot.capability == nil {
		slot.capability = capability
		capability.published = slotID
		close(slot.ready)
	}
	return nil
}
func (s *Session) Await(ctx context.Context, slotID string) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	if err := s.host.mu.LockContext(ctx); err != nil {
		return err
	}
	if s.closed {
		s.host.mu.Unlock()
		return errClosed
	}
	slot := s.slots[slotID]
	s.host.mu.Unlock()
	if slot == nil {
		return errUnauthorized
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.closedSignal:
		return errClosed
	case <-slot.ready:
		if err := s.host.mu.LockContext(ctx); err != nil {
			return err
		}
		defer s.host.mu.Unlock()
		if s.closed {
			return errClosed
		}
		if slot.capability.used {
			return errInvalid
		}
		return nil
	}
}
func (s *Session) Consume(ctx context.Context, slotID string) (umpire.OpaqueCapability, error) {
	if err := s.host.mu.LockContext(ctx); err != nil {
		return nil, err
	}
	defer s.host.mu.Unlock()
	if s.closed {
		return nil, errClosed
	}
	slot := s.slots[slotID]
	if slot == nil || slot.capability == nil || slot.capability.used {
		return nil, errUnauthorized
	}
	if old := slot.claim; old != nil && !old.released.Load() && old.context.Err() == nil {
		return nil, errUnauthorized
	}
	claim := &completionClaim{capability: slot.capability, context: ctx}
	slot.claim = claim
	return claim, nil
}
