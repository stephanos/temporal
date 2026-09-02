package executorhttp

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

func TestNewConnectsTheResidentExecutor(t *testing.T) {
	handler := New(testExecutor{execute: func(context.Context, *umpirespb.ExecuteRequest) (*umpirespb.ExecuteResponse, error) {
		return toolingResponse(umpirespb.TOOLING_STATUS_INVALID_CONTRACT), nil
	}})

	recorder := serve(handler, http.MethodPost, ExecutePath, ProtobufContentType, marshalProto(t, testRequest()))

	response := decodeResponse(t, recorder)
	require.Equal(t, umpirespb.TOOLING_STATUS_INVALID_CONTRACT, response.GetResult().GetToolingStatus())
	require.Equal(t, umpirespb.CANARY_DECISION_INCONCLUSIVE, response.GetResult().GetDecision())
}

type testExecutor struct {
	execute executeFunc
}

func (executor testExecutor) Execute(
	ctx context.Context,
	request *umpirespb.ExecuteRequest,
) (*umpirespb.ExecuteResponse, error) {
	return executor.execute(ctx, request)
}

func TestHandlerExchangesCanonicalProtobuf(t *testing.T) {
	request := testRequest()
	encodedRequest := marshalProto(t, request)
	want := toolingResponse(umpirespb.TOOLING_STATUS_SUCCEEDED)
	handler := newHandler(func(ctx context.Context, got *umpirespb.ExecuteRequest) (*umpirespb.ExecuteResponse, error) {
		require.NoError(t, ctx.Err())
		require.True(t, proto.Equal(request, got))
		return want, nil
	}, int64(len(encodedRequest)), int64(len(marshalProto(t, want))), time.Second)

	recorder := serve(handler, http.MethodPost, ExecutePath, ProtobufContentType, encodedRequest)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, ProtobufContentType, recorder.Header().Get("Content-Type"))
	require.Equal(t, "nosniff", recorder.Header().Get("X-Content-Type-Options"))
	require.Equal(t, marshalProto(t, want), recorder.Body.Bytes())
}

func TestHandlerRejectsInvalidTransportBeforeExecution(t *testing.T) {
	valid := marshalProto(t, testRequest())
	unknown := append(bytes.Clone(valid), protowire.AppendTag(nil, 63, protowire.VarintType)...)
	unknown = protowire.AppendVarint(unknown, 1)
	nestedInput := marshalProto(t, testRequest().GetInput())
	nestedInput = append(nestedInput, protowire.AppendTag(nil, 63, protowire.VarintType)...)
	nestedInput = protowire.AppendVarint(nestedInput, 1)
	nestedUnknown := testRequest()
	nestedUnknown.Input = new(umpirespb.EvaluationInput)
	require.NoError(t, proto.Unmarshal(nestedInput, nestedUnknown.Input))
	noncanonical := append(bytes.Clone(valid), valid...)
	tests := []struct {
		name        string
		method      string
		path        string
		contentType string
		body        []byte
		wantStatus  int
	}{
		{
			name: "wrong path", method: http.MethodPost, path: ExecutePath + "/",
			contentType: ProtobufContentType, body: valid, wantStatus: http.StatusNotFound,
		},
		{
			name: "wrong method", method: http.MethodGet, path: ExecutePath,
			contentType: ProtobufContentType, body: valid, wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name: "wrong content type", method: http.MethodPost, path: ExecutePath,
			contentType: "application/octet-stream", body: valid, wantStatus: http.StatusUnsupportedMediaType,
		},
		{
			name: "malformed protobuf", method: http.MethodPost, path: ExecutePath,
			contentType: ProtobufContentType, body: []byte{0x80}, wantStatus: http.StatusBadRequest,
		},
		{
			name: "unknown protobuf field", method: http.MethodPost, path: ExecutePath,
			contentType: ProtobufContentType, body: unknown, wantStatus: http.StatusBadRequest,
		},
		{
			name: "nested unknown protobuf field", method: http.MethodPost, path: ExecutePath,
			contentType: ProtobufContentType, body: marshalProto(t, nestedUnknown), wantStatus: http.StatusBadRequest,
		},
		{
			name: "noncanonical protobuf", method: http.MethodPost, path: ExecutePath,
			contentType: ProtobufContentType, body: noncanonical, wantStatus: http.StatusBadRequest,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var calls atomic.Int32
			handler := newHandler(func(context.Context, *umpirespb.ExecuteRequest) (*umpirespb.ExecuteResponse, error) {
				calls.Add(1)
				return toolingResponse(umpirespb.TOOLING_STATUS_SUCCEEDED), nil
			}, int64(len(test.body)), 1<<20, time.Second)

			recorder := serve(handler, test.method, test.path, test.contentType, test.body)

			require.Equal(t, test.wantStatus, recorder.Code)
			require.Empty(t, recorder.Body.Bytes())
			require.Zero(t, calls.Load())
			if test.wantStatus == http.StatusMethodNotAllowed {
				require.Equal(t, http.MethodPost, recorder.Header().Get("Allow"))
			}
		})
	}
}

func TestHandlerAdmitsExactRequestLimitAndRejectsLimitPlusOne(t *testing.T) {
	encoded := marshalProto(t, testRequest())
	var calls atomic.Int32
	handler := newHandler(func(context.Context, *umpirespb.ExecuteRequest) (*umpirespb.ExecuteResponse, error) {
		calls.Add(1)
		return toolingResponse(umpirespb.TOOLING_STATUS_SUCCEEDED), nil
	}, int64(len(encoded)), 1<<20, time.Second)

	exact := serveStreaming(handler, encoded)
	over := serveStreaming(handler, append(bytes.Clone(encoded), 0))

	require.Equal(t, http.StatusOK, exact.Code)
	require.Equal(t, http.StatusRequestEntityTooLarge, over.Code)
	require.Empty(t, over.Body.Bytes())
	require.Equal(t, int32(1), calls.Load())
}

func TestHandlerPreservesToolingFailuresAsInconclusiveResults(t *testing.T) {
	for _, status := range []umpirespb.ToolingStatus{
		umpirespb.TOOLING_STATUS_BUSY,
		umpirespb.TOOLING_STATUS_POISONED,
		umpirespb.TOOLING_STATUS_CANCELED,
		umpirespb.TOOLING_STATUS_INTERNAL_ERROR,
	} {
		t.Run(status.String(), func(t *testing.T) {
			want := toolingResponse(status)
			handler := newHandler(func(context.Context, *umpirespb.ExecuteRequest) (*umpirespb.ExecuteResponse, error) {
				return want, nil
			}, 1<<20, 1<<20, time.Second)

			recorder := serve(handler, http.MethodPost, ExecutePath, ProtobufContentType, marshalProto(t, testRequest()))

			require.Equal(t, http.StatusOK, recorder.Code)
			var got umpirespb.ExecuteResponse
			require.NoError(t, proto.Unmarshal(recorder.Body.Bytes(), &got))
			require.Equal(t, status, got.GetResult().GetToolingStatus())
			require.Equal(t, umpirespb.CANARY_DECISION_INCONCLUSIVE, got.GetResult().GetDecision())
		})
	}
}

func TestHandlerRejectsExecutorAndResponseTransportFailures(t *testing.T) {
	tests := []struct {
		name     string
		execute  executeFunc
		response int64
	}{
		{
			name: "executor error",
			execute: func(context.Context, *umpirespb.ExecuteRequest) (*umpirespb.ExecuteResponse, error) {
				return nil, errors.New("executor failed")
			},
			response: 1 << 20,
		},
		{
			name: "missing response",
			execute: func(context.Context, *umpirespb.ExecuteRequest) (*umpirespb.ExecuteResponse, error) {
				return nil, nil
			},
			response: 1 << 20,
		},
		{
			name: "oversized response",
			execute: func(context.Context, *umpirespb.ExecuteRequest) (*umpirespb.ExecuteResponse, error) {
				return toolingResponse(umpirespb.TOOLING_STATUS_SUCCEEDED), nil
			},
			response: 1,
		},
		{
			name: "unknown response enum",
			execute: func(context.Context, *umpirespb.ExecuteRequest) (*umpirespb.ExecuteResponse, error) {
				return toolingResponse(umpirespb.ToolingStatus(99)), nil
			},
			response: 1 << 20,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := newHandler(test.execute, 1<<20, test.response, time.Second)

			recorder := serve(handler, http.MethodPost, ExecutePath, ProtobufContentType, marshalProto(t, testRequest()))

			require.Equal(t, http.StatusInternalServerError, recorder.Code)
			require.Empty(t, recorder.Body.Bytes())
		})
	}
}

func TestHandlerPropagatesItsDeadlineAsCanceledAndInconclusive(t *testing.T) {
	deadlineObserved := make(chan time.Duration, 1)
	handler := newHandler(func(ctx context.Context, _ *umpirespb.ExecuteRequest) (*umpirespb.ExecuteResponse, error) {
		deadline, ok := ctx.Deadline()
		require.True(t, ok)
		deadlineObserved <- time.Until(deadline)
		<-ctx.Done()
		return toolingResponse(umpirespb.TOOLING_STATUS_CANCELED), nil
	}, 1<<20, 1<<20, time.Millisecond)

	recorder := serve(handler, http.MethodPost, ExecutePath, ProtobufContentType, marshalProto(t, testRequest()))

	require.LessOrEqual(t, <-deadlineObserved, 10*time.Millisecond)
	require.Equal(t, http.StatusOK, recorder.Code)
	var got umpirespb.ExecuteResponse
	require.NoError(t, proto.Unmarshal(recorder.Body.Bytes(), &got))
	require.Equal(t, umpirespb.TOOLING_STATUS_CANCELED, got.GetResult().GetToolingStatus())
	require.Equal(t, umpirespb.CANARY_DECISION_INCONCLUSIVE, got.GetResult().GetDecision())
}

func TestHandlerClientCancellationCannotPublishPartialSuccess(t *testing.T) {
	started := make(chan struct{})
	handler := newHandler(func(ctx context.Context, _ *umpirespb.ExecuteRequest) (*umpirespb.ExecuteResponse, error) {
		close(started)
		<-ctx.Done()
		return toolingResponse(umpirespb.TOOLING_STATUS_SUCCEEDED), nil
	}, 1<<20, 1<<20, time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	request := httptest.NewRequest(http.MethodPost, ExecutePath, bytes.NewReader(marshalProto(t, testRequest()))).WithContext(ctx)
	request.Header.Set("Content-Type", ProtobufContentType)
	recorder := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		handler.ServeHTTP(recorder, request)
		close(done)
	}()

	<-started
	cancel()
	<-done

	require.Empty(t, recorder.Body.Bytes())
	require.Empty(t, recorder.Header())
}

func TestHandlerReusesOneExecutorSequentially(t *testing.T) {
	var calls atomic.Int32
	handler := newHandler(func(context.Context, *umpirespb.ExecuteRequest) (*umpirespb.ExecuteResponse, error) {
		call := calls.Add(1)
		response := toolingResponse(umpirespb.TOOLING_STATUS_SUCCEEDED)
		response.Result.RunIdentity = "run-" + string(rune('0'+call))
		return response, nil
	}, 1<<20, 1<<20, time.Second)

	first := serve(handler, http.MethodPost, ExecutePath, ProtobufContentType, marshalProto(t, testRequest()))
	second := serve(handler, http.MethodPost, ExecutePath, ProtobufContentType, marshalProto(t, testRequest()))

	require.Equal(t, "run-1", decodeResponse(t, first).GetResult().GetRunIdentity())
	require.Equal(t, "run-2", decodeResponse(t, second).GetResult().GetRunIdentity())
	require.Equal(t, int32(2), calls.Load())
}

func TestHandlerOverlapReturnsBusyBeforeRuntimeIO(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	firstDone := make(chan *httptest.ResponseRecorder, 1)
	var active atomic.Bool
	var runtimeCalls atomic.Int32
	handler := newHandler(func(context.Context, *umpirespb.ExecuteRequest) (*umpirespb.ExecuteResponse, error) {
		if !active.CompareAndSwap(false, true) {
			return toolingResponse(umpirespb.TOOLING_STATUS_BUSY), nil
		}
		runtimeCalls.Add(1)
		close(entered)
		<-release
		active.Store(false)
		return toolingResponse(umpirespb.TOOLING_STATUS_SUCCEEDED), nil
	}, 1<<20, 1<<20, time.Second)
	body := marshalProto(t, testRequest())
	go func() {
		firstDone <- serve(handler, http.MethodPost, ExecutePath, ProtobufContentType, body)
	}()
	<-entered

	overlap := serve(handler, http.MethodPost, ExecutePath, ProtobufContentType, body)

	require.Equal(t, umpirespb.TOOLING_STATUS_BUSY, decodeResponse(t, overlap).GetResult().GetToolingStatus())
	require.Equal(t, umpirespb.CANARY_DECISION_INCONCLUSIVE, decodeResponse(t, overlap).GetResult().GetDecision())
	require.Equal(t, int32(1), runtimeCalls.Load())
	close(release)
	require.Equal(t, umpirespb.TOOLING_STATUS_SUCCEEDED, decodeResponse(t, <-firstDone).GetResult().GetToolingStatus())
}

func serve(handler http.Handler, method, path, contentType string, body []byte) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, bytes.NewReader(body))
	request.Header.Set("Content-Type", contentType)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func serveStreaming(handler http.Handler, body []byte) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, ExecutePath, io.NopCloser(bytes.NewReader(body)))
	request.Header.Set("Content-Type", ProtobufContentType)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func marshalProto(t *testing.T, message proto.Message) []byte {
	t.Helper()
	encoded, err := (proto.MarshalOptions{Deterministic: true}).Marshal(message)
	require.NoError(t, err)
	return encoded
}

func decodeResponse(t *testing.T, recorder *httptest.ResponseRecorder) *umpirespb.ExecuteResponse {
	t.Helper()
	require.Equal(t, http.StatusOK, recorder.Code)
	var response umpirespb.ExecuteResponse
	require.NoError(t, proto.Unmarshal(recorder.Body.Bytes(), &response))
	return &response
}

func testRequest() *umpirespb.ExecuteRequest {
	return &umpirespb.ExecuteRequest{
		EvaluationContract: []byte("contract"),
		Input: &umpirespb.EvaluationInput{
			Experiment: []byte("experiment"), RuntimeConfig: []byte("runtime"),
		},
	}
}

func toolingResponse(status umpirespb.ToolingStatus) *umpirespb.ExecuteResponse {
	decision := umpirespb.CANARY_DECISION_INCONCLUSIVE
	if status == umpirespb.TOOLING_STATUS_SUCCEEDED {
		decision = umpirespb.CANARY_DECISION_PASS
	}
	return &umpirespb.ExecuteResponse{Result: &umpirespb.EvaluationResult{
		ToolingStatus: status, Decision: decision,
	}}
}
