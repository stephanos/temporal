package worker

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nexus-rpc/sdk-go/nexus"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/converter"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	commonnexus "go.temporal.io/server/common/nexus"
	"google.golang.org/protobuf/proto"
)

func callbackValue() *testpilotspb.Value {
	return &testpilotspb.Value{Value: &testpilotspb.Value_Text{Text: "result"}}
}

func callbackLimits() *testpilotspb.ProgramLimits {
	return &testpilotspb.ProgramLimits{MaxAttempts: 8, MaxRequestBytes: 4096, MaxResponseBytes: 4096, MaxTotalDurationMilliseconds: 1000}
}

func TestCompletionTransportPreservesPayloadAndCredentials(t *testing.T) {
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
	transport, err := newCompletionTransport(nil, "", callbackLimits())
	require.NoError(t, err)
	header := nexus.Header{"authorization": "completion-secret"}
	effect, err := transport.newEffect(completionInfo{URL: target.URL, Header: header, OperationToken: "operation-secret"})
	require.NoError(t, err)
	header["authorization"] = "changed"

	result := effect.Invoke(t.Context(), callbackValue(), 4096)
	require.Equal(t, testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, result.Outcome.Status)
	require.Nil(t, result.Response)
	require.Empty(t, result.Outcome.Detail)
	req := <-requests
	require.NoError(t, req.err)
	require.Equal(t, "completion-secret", req.credential)
	require.Equal(t, "operation-secret", req.token)
	var payload *commonpb.Payload
	require.NoError(t, commonnexus.PayloadSerializer.Deserialize(&nexus.Content{Header: nexus.Header{"type": req.contentType}, Data: req.data}, &payload))
	var decoded *testpilotspb.Value
	require.NoError(t, converter.GetDefaultDataConverter().FromPayload(payload, &decoded))
	require.True(t, proto.Equal(callbackValue(), decoded))
	require.NotContains(t, fmt.Sprint(result), "secret")
}

func TestCompletionEffectAcceptsOnlyCompletionInstructionsAndValues(t *testing.T) {
	transport, err := newCompletionTransport(nil, "", callbackLimits())
	require.NoError(t, err)
	effect, err := transport.newEffect(completionInfo{URL: "http://localhost", OperationToken: "token"})
	require.NoError(t, err)
	completion := &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_CompleteNexusOperation{CompleteNexusOperation: &testpilotspb.CompleteNexusOperation{}}}
	other := &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_InvokeRpc{InvokeRpc: &testpilotspb.InvokeRPC{}}}
	require.True(t, effect.Accepts(t.Context(), completion, callbackValue()))
	require.False(t, effect.Accepts(t.Context(), other, callbackValue()))
	require.False(t, effect.Accepts(t.Context(), completion, &testpilotspb.InstructionOutcome{}))
}

func TestCompletionTransportRejectsInvalidAuthorityAndBounds(t *testing.T) {
	transport, err := newCompletionTransport(nil, "", callbackLimits())
	require.NoError(t, err)
	for _, info := range []completionInfo{
		{URL: "file:///secret", OperationToken: "token"},
		{URL: "http://user:secret@localhost", OperationToken: "token"},
	} {
		effect, err := transport.newEffect(info)
		require.ErrorIs(t, err, ErrInvalid)
		require.Nil(t, effect)
		require.NotContains(t, err.Error(), "secret")
	}
	effect, err := transport.newEffect(completionInfo{URL: "http://localhost", OperationToken: strings.Repeat("x", 5000)})
	require.ErrorIs(t, err, ErrCapacity)
	require.Nil(t, effect)

	limits := callbackLimits()
	limits.MaxRequestBytes = int64(len(commonnexus.SystemCallbackURL) + len("token"))
	transport, err = newCompletionTransport(nil, "https://"+strings.Repeat("a", 64)+".invalid", limits)
	require.NoError(t, err)
	effect, err = transport.newEffect(completionInfo{URL: commonnexus.SystemCallbackURL, OperationToken: "token"})
	require.ErrorIs(t, err, ErrCapacity)
	require.Nil(t, effect)
}

func TestCompletionTransportClassifiesFailures(t *testing.T) {
	for _, mode := range []string{"failure", "timeout", "oversized", "redirect", "truncated"} {
		t.Run(mode, func(t *testing.T) {
			target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch mode {
				case "failure":
					http.Error(w, "completion-secret", http.StatusForbidden)
				case "timeout":
					if _, err := io.Copy(io.Discard, r.Body); err != nil {
						t.Errorf("read request body: %v", err)
						return
					}
					<-r.Context().Done()
				case "oversized":
					if _, err := io.WriteString(w, strings.Repeat("x", 5000)); err != nil {
						t.Errorf("write oversized response: %v", err)
					}
				case "redirect":
					w.Header().Set("Location", "http://127.0.0.1:1/credential-leak")
					w.WriteHeader(http.StatusTemporaryRedirect)
				case "truncated":
					w.Header().Set("Content-Length", "100")
					if _, err := io.WriteString(w, "short"); err != nil {
						t.Errorf("write truncated response: %v", err)
					}
				default:
					t.Errorf("unknown mode %q", mode)
				}
			}))
			defer target.Close()
			transport, err := newCompletionTransport(nil, "", callbackLimits())
			require.NoError(t, err)
			effect, err := transport.newEffect(completionInfo{URL: target.URL, Header: nexus.Header{"authorization": "completion-secret"}, OperationToken: "token"})
			require.NoError(t, err)
			ctx := t.Context()
			if mode == "timeout" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 20*time.Millisecond)
				defer cancel()
			}
			result := effect.Invoke(ctx, callbackValue(), 4096)
			want := testpilotspb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_NON_SUCCESS
			if mode == "timeout" {
				want = testpilotspb.INSTRUCTION_OUTCOME_STATUS_TIMED_OUT
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

func TestCompletionTransportResolvesOnlyTrustedSystemCallbacks(t *testing.T) {
	paths := make(chan string, 1)
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths <- r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer target.Close()
	transport, err := newCompletionTransport(nil, target.URL, callbackLimits())
	require.NoError(t, err)
	effect, err := transport.newEffect(completionInfo{URL: commonnexus.SystemCallbackURL, OperationToken: "token"})
	require.NoError(t, err)
	result := effect.Invoke(t.Context(), callbackValue(), 4096)
	require.Equal(t, testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, result.Outcome.Status)
	require.Equal(t, commonnexus.PathCompletionCallbackNoIdentifier, <-paths)

	for _, callbackURL := range []string{"//attacker.invalid/nexus/callback", "/../nexus/callback", "/nexus/callback#fragment", "nexus/callback"} {
		denied, err := transport.newEffect(completionInfo{URL: callbackURL, OperationToken: "token"})
		require.ErrorIs(t, err, ErrInvalid)
		require.Nil(t, denied)
	}
}
