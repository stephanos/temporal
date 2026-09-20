package worker

import (
	"context"
	"errors"
	"io"
	"maps"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/nexus-rpc/sdk-go/nexus"
	commonpb "go.temporal.io/api/common/v1"
	failurepb "go.temporal.io/api/failure/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	commonnexus "go.temporal.io/server/common/nexus"
	"go.temporal.io/server/common/nexus/nexusrpc"
	"go.temporal.io/server/common/testing/testpilot"
	"google.golang.org/protobuf/proto"
)

type completionInfo struct {
	URL            string
	Header         nexus.Header
	OperationToken string
	StartTime      time.Time
}

type completionTransport struct {
	httpClient            http.Client
	systemCallbackBaseURL *url.URL
	maxRequestBytes       int64
}

type completionEffect struct {
	transport *completionTransport
	info      completionInfo
}

func newCompletionTransport(client *http.Client, rawBaseURL string, limits *testpilotspb.ProgramLimits) (*completionTransport, error) {
	baseURL, err := parseSystemCallbackBaseURL(rawBaseURL)
	if err != nil || limits == nil {
		return nil, ErrInvalid
	}
	transport := &completionTransport{systemCallbackBaseURL: baseURL, maxRequestBytes: limits.GetMaxRequestBytes()}
	if client != nil {
		transport.httpClient = *client
	}
	roundTripper := transport.httpClient.Transport
	if roundTripper == nil {
		roundTripper = http.DefaultTransport
	}
	if standard, ok := roundTripper.(*http.Transport); ok && standard != nil {
		owned := standard.Clone()
		owned.MaxResponseHeaderBytes = limits.GetMaxResponseBytes()
		owned.MaxConnsPerHost = int(limits.GetMaxAttempts())
		owned.MaxIdleConns = int(limits.GetMaxAttempts())
		owned.MaxIdleConnsPerHost = int(limits.GetMaxAttempts())
		transport.httpClient.Transport = owned
	} else if nilValue(roundTripper) {
		return nil, ErrInvalid
	}
	transport.httpClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	transport.httpClient.Timeout = time.Duration(limits.GetMaxTotalDurationMilliseconds()) * time.Millisecond
	return transport, nil
}

func parseSystemCallbackBaseURL(raw string) (*url.URL, error) {
	if raw == "" {
		return nil, nil
	}
	base, err := url.Parse(raw)
	if err != nil || base.Host == "" || (base.Scheme != "http" && base.Scheme != "https") || base.User != nil || base.Fragment != "" || base.RawQuery != "" || base.ForceQuery || base.Opaque != "" || base.RawPath != "" || (base.Path != "" && base.Path != "/") {
		return nil, ErrInvalid
	}
	base.Path = ""
	return base, nil
}

func (t *completionTransport) newEffect(info completionInfo) (testpilot.CapabilityEffect, error) {
	target, err := url.Parse(info.URL)
	if err == nil && (info.URL == commonnexus.SystemCallbackURL || info.URL == commonnexus.PathCompletionCallbackNoIdentifier) && t.systemCallbackBaseURL != nil {
		target = t.systemCallbackBaseURL.ResolveReference(&url.URL{Path: commonnexus.PathCompletionCallbackNoIdentifier})
		info.URL = target.String()
	}
	if err != nil || target.Host == "" || (target.Scheme != "http" && target.Scheme != "https") || target.User != nil || target.Fragment != "" || info.OperationToken == "" {
		return nil, ErrInvalid
	}
	size := len(info.URL) + len(info.OperationToken)
	for key, value := range info.Header {
		size += len(key) + len(value)
	}
	if int64(size) > t.maxRequestBytes {
		return nil, ErrCapacity
	}
	info.Header = maps.Clone(info.Header)
	return completionEffect{transport: t, info: info}, nil
}

// Accepts admits the typed completion with the payload or failure it carries.
func (completionEffect) Accepts(_ context.Context, instruction *testpilotspb.Instruction, input proto.Message) bool {
	switch input.(type) {
	case *commonpb.Payload:
		return instruction.GetNexusOperationCompletion().GetPayload() != nil
	case *failurepb.Failure:
		return instruction.GetNexusOperationCompletion().GetFailure() != nil
	default:
		return false
	}
}

// Invoke delivers the completion: an interpreter value as a protobuf payload of its own type, a
// carried payload as it is, and a carried failure as the operation error it denotes.
func (e completionEffect) Invoke(ctx context.Context, input proto.Message, maxResponseBytes int64) testpilot.EffectResult {
	invalidArgument := testpilot.EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_FAILURE, ProtocolCode: "invalid_argument"}}
	switch typed := input.(type) {
	case *testpilotspb.Value:
		data, err := proto.Marshal(typed)
		if err != nil {
			return invalidArgument
		}
		payload := &commonpb.Payload{Metadata: map[string][]byte{"encoding": []byte("binary/protobuf"), "messageType": []byte("temporal.server.api.testpilot.v1.Value")}, Data: data}
		return e.transport.complete(ctx, e.info, payload, nil, maxResponseBytes)
	case *commonpb.Payload:
		return e.transport.complete(ctx, e.info, proto.CloneOf(typed), nil, maxResponseBytes)
	case *failurepb.Failure:
		failure, err := operationError(typed)
		if err != nil {
			return invalidArgument
		}
		return e.transport.complete(ctx, e.info, nil, failure, maxResponseBytes)
	default:
		return invalidArgument
	}
}

func (t *completionTransport) complete(ctx context.Context, info completionInfo, result *commonpb.Payload, failure *nexus.OperationError, maxResponseBytes int64) testpilot.EffectResult {
	var body *boundedBody
	var protocolCode int
	caller := func(request *http.Request) (*http.Response, error) {
		response, err := t.httpClient.Do(request)
		if err != nil {
			return nil, err
		}
		body = &boundedBody{ReadCloser: response.Body, remaining: maxResponseBytes}
		response.Body = body
		protocolCode = response.StatusCode
		return response, nil
	}
	client := nexusrpc.NewCompletionHTTPClient(nexusrpc.CompletionHTTPClientOptions{HTTPCaller: caller, Serializer: commonnexus.PayloadSerializer})
	options := nexusrpc.CompleteOperationOptions{Header: info.Header, OperationToken: info.OperationToken, StartTime: info.StartTime, Result: result, Error: failure}
	if failure != nil {
		options.Result = nil
	}
	err := client.CompleteOperation(ctx, info.URL, options)
	if body != nil {
		closeErr := body.Close()
		if err == nil {
			err = closeErr
		}
		if body.readErr != nil {
			err = body.readErr
		}
		if body.exceeded {
			err = ErrCapacity
		}
	}
	outcome := &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, ProtocolCode: "ok"}
	if err != nil {
		outcome.Status = testpilotspb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_FAILURE
		outcome.ProtocolCode = "transport_failure"
		if protocolCode != 0 {
			outcome.ProtocolCode = "http_" + strconv.Itoa(protocolCode)
		}
		if errors.Is(err, ErrCapacity) {
			outcome.ProtocolCode = "resource_exhausted"
		}
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			outcome.Status = testpilotspb.INSTRUCTION_OUTCOME_STATUS_TIMED_OUT
			outcome.ProtocolCode = "deadline_exceeded"
		}
		if errors.Is(ctx.Err(), context.Canceled) {
			outcome.Status = testpilotspb.INSTRUCTION_OUTCOME_STATUS_CANCELED
			outcome.ProtocolCode = "canceled"
		}
	}
	return testpilot.EffectResult{Outcome: outcome}
}

func (t *completionTransport) close() {
	t.httpClient.CloseIdleConnections()
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
		return 0, ErrCapacity
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
