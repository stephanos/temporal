package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/nexus-rpc/sdk-go/nexus"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/converter"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	commonnexus "go.temporal.io/server/common/nexus"
	"go.temporal.io/server/tools/umpire"
	"google.golang.org/protobuf/proto"
)

func nexusSession(t *testing.T, h *Host, source *umpirespb.Case, run string) (*Session, umpire.Coordinate) {
	t.Helper()
	program := proto.CloneOf(source.Program)
	node := rpcNode("complete", "")
	node.Instruction = &umpirespb.Instruction{Instruction: &umpirespb.Instruction_CompleteNexusOperation{CompleteNexusOperation: &umpirespb.CompleteNexusOperation{CapabilitySlotId: "capability"}}}
	program.Entrypoints[0].Nodes = append(program.Entrypoints[0].Nodes, node)
	program.Entrypoints = append(program.Entrypoints, &umpirespb.Entrypoint{EntrypointId: "handler", Context: umpirespb.ENTRYPOINT_CONTEXT_NEXUS_HANDLER})
	for _, id := range []string{"capability", "other"} {
		program.Slots = append(program.Slots, &umpirespb.SlotSchema{SlotId: id, Kind: umpirespb.SLOT_KIND_OPAQUE_CAPABILITY, Type: &umpirespb.ValueType{Shape: &umpirespb.ValueType_Singular{Singular: &umpirespb.SingularType{Type: &umpirespb.SingularType_OpaqueCapability{OpaqueCapability: &umpirespb.OpaqueCapabilityType{}}}}}})
	}
	s, err := h.open(t.Context(), run, program)
	require.NoError(t, err)
	return s, umpire.Coordinate{RunID: run, EntrypointID: "handler", ActivationID: "activation", InstructionID: "respond", Attempt: 1}
}
func completionValue() *umpirespb.Value {
	return &umpirespb.Value{Value: &umpirespb.Value_Text{Text: "result"}}
}

func TestNexusCompletionPayloadAndOpaqueOwnership(t *testing.T) {
	type received struct {
		data                           []byte
		contentType, token, credential string
		err                            error
	}
	requests := make(chan received, 1)
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		requests <- received{data, r.Header.Get("Content-Type"), r.Header.Get("Nexus-Operation-Token"), r.Header.Get("authorization"), err}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer target.Close()
	h, source, _ := fixture(t, "127.0.0.1:1")
	s, origin := nexusSession(t, h, source, "run")
	info := CompletionInfo{URL: target.URL, Header: nexus.Header{"authorization": "completion-secret"}, OperationToken: "operation-secret"}
	capability, err := s.NewCompletionCapability(t.Context(), origin, info)
	require.NoError(t, err)
	info.Header["authorization"] = "changed"
	bridge, err := s.Bridge(t.Context())
	require.NoError(t, err)
	require.NoError(t, bridge.Publish(t.Context(), origin, "capability", capability))
	require.NoError(t, bridge.Publish(t.Context(), origin, "capability", capability))
	require.Error(t, bridge.Publish(t.Context(), origin, "other", capability))
	wrong := origin
	wrong.ActivationID = "foreign"
	require.Error(t, bridge.Publish(t.Context(), wrong, "capability", capability))
	require.NoError(t, bridge.Await(t.Context(), "capability"))
	consumed, err := bridge.Consume(t.Context(), "capability")
	require.NoError(t, err)
	require.Same(t, capability, consumed.(*completionClaim).capability)
	_, err = bridge.Consume(t.Context(), "capability")
	require.Error(t, err)
	foreign, _ := nexusSession(t, h, source, "foreign")
	denied, err := foreign.CompleteNexusOperation(t.Context(), coordinate("foreign", "complete"), consumed, completionValue())
	require.Error(t, err)
	require.Nil(t, denied)
	handle, err := s.CompleteNexusOperation(t.Context(), coordinate("run", "complete"), consumed, completionValue())
	require.NoError(t, err)
	result, err := handle.Wait(t.Context())
	require.NoError(t, err)
	require.Equal(t, umpirespb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, result.Outcome.Status)
	require.Nil(t, result.Response)
	require.Empty(t, result.Outcome.Detail)
	denied, err = s.CompleteNexusOperation(t.Context(), coordinate("run", "complete"), consumed, completionValue())
	require.Error(t, err)
	require.Nil(t, denied)
	req := <-requests
	require.NoError(t, req.err)
	require.Equal(t, "completion-secret", req.credential)
	require.Equal(t, "operation-secret", req.token)
	var payload *commonpb.Payload
	require.NoError(t, commonnexus.PayloadSerializer.Deserialize(&nexus.Content{Header: nexus.Header{"type": req.contentType}, Data: req.data}, &payload))
	var decoded *umpirespb.Value
	require.NoError(t, converter.GetDefaultDataConverter().FromPayload(payload, &decoded))
	require.True(t, proto.Equal(completionValue(), decoded))
	require.NotContains(t, fmt.Sprint(result), "secret")
	require.NoError(t, s.Close(t.Context()))
	require.Error(t, bridge.Await(t.Context(), "capability"))
	require.Error(t, bridge.Publish(t.Context(), origin, "capability", capability))
}
func TestNexusFailureTimeoutSizeAndRedirect(t *testing.T) {
	for _, mode := range []string{"failure", "timeout", "oversized", "redirect", "truncated"} {
		t.Run(mode, func(t *testing.T) {
			target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch mode {
				case "failure":
					http.Error(w, "completion-secret", http.StatusForbidden)
				case "timeout":
					if _, err := io.Copy(io.Discard, r.Body); err != nil {
						return
					}
					<-r.Context().Done()
				case "oversized":
					_, err := io.WriteString(w, strings.Repeat("x", 5000))
					if err != nil {
						return
					}
				case "redirect":
					w.Header().Set("Location", "http://127.0.0.1:1/credential-leak")
					w.WriteHeader(http.StatusTemporaryRedirect)
				case "truncated":
					w.Header().Set("Content-Length", "100")
					if _, err := io.WriteString(w, "short"); err != nil {
						return
					}
				default:
					http.Error(w, "invalid fixture", http.StatusInternalServerError)
				}
			}))
			defer target.Close()
			h, source, _ := fixture(t, "127.0.0.1:1")
			s, origin := nexusSession(t, h, source, "run")
			if mode == "timeout" {
				s.nodes[nodeKey{"controller", "complete"}].Bounds.TimeoutMilliseconds = 20
			}
			capability, err := s.NewCompletionCapability(t.Context(), origin, CompletionInfo{URL: target.URL, Header: nexus.Header{"authorization": "completion-secret"}, OperationToken: "token"})
			require.NoError(t, err)
			require.NoError(t, s.Publish(t.Context(), origin, "capability", capability))
			claim, err := s.Consume(t.Context(), "capability")
			require.NoError(t, err)
			handle, err := s.CompleteNexusOperation(t.Context(), coordinate("run", "complete"), claim, completionValue())
			require.NoError(t, err)
			result, err := handle.Wait(t.Context())
			require.NoError(t, err)
			want := umpirespb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_NON_SUCCESS
			if mode == "timeout" {
				want = umpirespb.INSTRUCTION_OUTCOME_STATUS_TIMED_OUT
			}
			require.Equal(t, want, result.Outcome.Status)
			require.NotContains(t, fmt.Sprint(result), "completion-secret")
			require.Empty(t, result.Outcome.Detail)
			if mode == "oversized" {
				require.Equal(t, "resource_exhausted", result.Outcome.ProtocolCode)
			}
		})
	}
}
func TestCapabilityClosureAndLimits(t *testing.T) {
	h, source, _ := fixture(t, "127.0.0.1:1")
	s, origin := nexusSession(t, h, source, "run")
	for _, info := range []CompletionInfo{{URL: "file:///secret", OperationToken: "x"}, {URL: "http://user:secret@localhost", OperationToken: "x"}, {URL: "http://localhost", OperationToken: strings.Repeat("x", 5000)}} {
		capability, err := s.NewCompletionCapability(t.Context(), origin, info)
		require.Error(t, err)
		require.Nil(t, capability)
		require.NotContains(t, err.Error(), "secret")
	}
	h.profile.ProgramLimits.MaxActivations = 1
	info := CompletionInfo{URL: "http://localhost", OperationToken: "secret"}
	capability, err := s.NewCompletionCapability(t.Context(), origin, info)
	require.NoError(t, err)
	_, err = s.NewCompletionCapability(t.Context(), origin, info)
	require.ErrorIs(t, err, errCapacity)
	ctx, cancel := context.WithTimeout(t.Context(), time.Millisecond)
	defer cancel()
	require.ErrorIs(t, s.Await(ctx, "capability"), context.DeadlineExceeded)
	waiting := make(chan error, 1)
	go func() { waiting <- s.Await(t.Context(), "capability") }()
	require.NoError(t, s.Close(t.Context()))
	require.ErrorIs(t, <-waiting, errClosed)
	require.Empty(t, capability.(*completionCapability).info.OperationToken)
	require.NoError(t, s.Diagnose(t.Context(), "run", &umpirespb.RunDiagnostic{}))
	require.Error(t, s.Diagnose(t.Context(), "foreign", nil))
}

func TestSystemRelativeCompletionCallbackResolution(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer target.Close()
	h, source, _ := fixture(t, "127.0.0.1:1")
	base, err := url.Parse(target.URL)
	require.NoError(t, err)
	h.systemCallbackBaseURL = base
	s, origin := nexusSession(t, h, source, "run")

	for _, callbackURL := range []string{commonnexus.SystemCallbackURL, commonnexus.PathCompletionCallbackNoIdentifier} {
		capability, err := s.NewCompletionCapability(t.Context(), origin, CompletionInfo{URL: callbackURL, OperationToken: "token"})
		require.NoError(t, err)
		require.Equal(t, target.URL+commonnexus.PathCompletionCallbackNoIdentifier, capability.(*completionCapability).info.URL)
	}

	h.systemCallbackBaseURL = &url.URL{Scheme: "https", Host: strings.Repeat("a", 64) + ".invalid"}
	h.profile.ProgramLimits.MaxRequestBytes = int64(len(commonnexus.SystemCallbackURL) + len("token"))
	denied, err := s.NewCompletionCapability(t.Context(), origin, CompletionInfo{URL: commonnexus.SystemCallbackURL, OperationToken: "token"})
	require.ErrorIs(t, err, errCapacity)
	require.Nil(t, denied)

	for _, callbackURL := range []string{
		"//attacker.invalid/nexus/callback",
		"/../nexus/callback",
		"/nexus/callback/..",
		"/nexus/callback#fragment",
		"nexus/callback",
	} {
		denied, err := s.NewCompletionCapability(t.Context(), origin, CompletionInfo{URL: callbackURL, OperationToken: "token"})
		require.ErrorIs(t, err, errInvalid)
		require.Nil(t, denied)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestNexusQuarantineRetainsCapacityUntilTransportReturns(t *testing.T) {
	h, source, _ := fixture(t, "127.0.0.1:1")
	entered, release := make(chan struct{}), make(chan struct{})
	h.httpClient.Transport = roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		close(entered)
		<-release
		return nil, r.Context().Err()
	})
	h.profile.ProgramLimits.MaxAttempts = 1
	s, origin := nexusSession(t, h, source, "run")
	capability, err := s.NewCompletionCapability(t.Context(), origin, CompletionInfo{URL: "http://localhost", OperationToken: "secret"})
	require.NoError(t, err)
	require.NoError(t, s.Publish(t.Context(), origin, "capability", capability))
	claim, err := s.Consume(t.Context(), "capability")
	require.NoError(t, err)
	handle, err := s.CompleteNexusOperation(t.Context(), coordinate("run", "complete"), claim, completionValue())
	require.NoError(t, err)
	<-entered
	require.NoError(t, handle.Cancel(t.Context()))
	ctx, cancel := context.WithTimeout(t.Context(), time.Millisecond)
	defer cancel()
	_, err = handle.Wait(ctx)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.ErrorIs(t, handle.Drain(ctx), context.DeadlineExceeded)
	require.NoError(t, s.Quarantine(t.Context(), handle))
	require.NoError(t, s.Close(t.Context()))
	h.mu.Lock()
	require.EqualValues(t, 1, h.effects)
	h.mu.Unlock()
	close(release)
	require.NoError(t, handle.Drain(t.Context()))
	result, err := handle.Wait(t.Context())
	require.NoError(t, err)
	require.Equal(t, umpirespb.INSTRUCTION_OUTCOME_STATUS_CANCELED, result.Outcome.Status)
	h.mu.Lock()
	require.Zero(t, h.effects)
	h.mu.Unlock()
}
