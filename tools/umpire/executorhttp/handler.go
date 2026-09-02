// Package executorhttp exposes the resident Umpire executor through bounded protobuf HTTP.
package executorhttp

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/artifact"
	"go.temporal.io/server/tools/umpire/evaluationcontract"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	ExecutePath         = "/umpire/v1/execute"
	ProtobufContentType = "application/x-protobuf"

	maximumProtobufEnvelopeBytes = 32
	maximumResponseEnvelopeBytes = 5
	maximumRequestBytes          = evaluationcontract.MaximumContractBytes +
		2*artifact.MaximumDocumentBytes + maximumProtobufEnvelopeBytes
	maximumResultBytes     = evaluationcontract.MaximumResultBytes
	maximumRequestDuration = time.Duration(evaluationcontract.MaximumDurationMillis) * time.Millisecond
)

var deterministicMarshal = proto.MarshalOptions{Deterministic: true}

type executeFunc func(context.Context, *umpirespb.ExecuteRequest) (*umpirespb.ExecuteResponse, error)

// Executor is the transport-independent resident execution seam.
type Executor interface {
	Execute(context.Context, *umpirespb.ExecuteRequest) (*umpirespb.ExecuteResponse, error)
}

type handler struct {
	execute             executeFunc
	marshal             func(proto.Message) ([]byte, error)
	maximumRequestBytes int64
	maximumResultBytes  int64
	maximumDuration     time.Duration
}

// New returns the fixed HTTP handler for one resident executor.
func New(resident Executor) http.Handler {
	var execute executeFunc
	if resident != nil {
		execute = resident.Execute
	}
	return newHandler(
		execute,
		maximumRequestBytes,
		maximumResultBytes,
		maximumRequestDuration,
	)
}

func newHandler(
	execute executeFunc,
	requestBytes int64,
	resultBytes int64,
	duration time.Duration,
) http.Handler {
	return &handler{
		execute: execute, marshal: deterministicMarshal.Marshal,
		maximumRequestBytes: requestBytes, maximumResultBytes: resultBytes,
		maximumDuration: duration,
	}
}

func (h *handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if request.URL.Path != ExecutePath {
		writeTransportStatus(writer, http.StatusNotFound)
		return
	}
	if request.Method != http.MethodPost {
		writer.Header().Set("Allow", http.MethodPost)
		writeTransportStatus(writer, http.StatusMethodNotAllowed)
		return
	}
	if request.Header.Get("Content-Type") != ProtobufContentType {
		writeTransportStatus(writer, http.StatusUnsupportedMediaType)
		return
	}
	if h == nil || h.execute == nil || h.marshal == nil || h.maximumRequestBytes <= 0 ||
		h.maximumResultBytes <= 0 || h.maximumDuration <= 0 {
		writeTransportStatus(writer, http.StatusInternalServerError)
		return
	}
	clientContext := request.Context()
	ctx, cancel := context.WithTimeout(clientContext, h.maximumDuration)
	defer cancel()
	request = request.WithContext(ctx)
	if deadline, ok := ctx.Deadline(); ok {
		finishDeadlines := setHTTPDeadlines(writer, deadline)
		defer finishDeadlines()
	}
	executionRequest, status := h.decodeRequest(writer, request)
	if status != 0 {
		publishTransportFailure(clientContext, ctx, writer, status)
		return
	}

	response, err := h.execute(ctx, executionRequest)
	if clientContext.Err() != nil {
		return
	}
	encoded, status := h.encodeResponse(ctx, response, err)
	if status != 0 {
		publishTransportFailure(clientContext, ctx, writer, status)
		return
	}
	if clientContext.Err() != nil {
		return
	}
	if contextDeadlineExceeded(ctx) {
		publishTransportFailure(clientContext, ctx, writer, http.StatusGatewayTimeout)
		return
	}

	writer.Header().Set("Content-Type", ProtobufContentType)
	writer.Header().Set("Content-Length", strconv.Itoa(len(encoded)))
	writer.Header().Set("X-Content-Type-Options", "nosniff")
	writer.WriteHeader(http.StatusOK)
	if _, err := writer.Write(encoded); err != nil {
		return
	}
}

func (h *handler) decodeRequest(
	writer http.ResponseWriter,
	request *http.Request,
) (*umpirespb.ExecuteRequest, int) {
	if request.ContentLength > h.maximumRequestBytes {
		return nil, http.StatusRequestEntityTooLarge
	}
	request.Body = http.MaxBytesReader(writer, request.Body, h.maximumRequestBytes)
	encoded, err := io.ReadAll(request.Body)
	if closeErr := request.Body.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		if contextDeadlineExceeded(request.Context()) {
			return nil, http.StatusRequestTimeout
		}
		var maximumBytesError *http.MaxBytesError
		if errors.As(err, &maximumBytesError) {
			return nil, http.StatusRequestEntityTooLarge
		}
		return nil, http.StatusBadRequest
	}

	executionRequest := new(umpirespb.ExecuteRequest)
	if err := (proto.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(encoded, executionRequest); err != nil ||
		!validProtoSemantics(executionRequest.ProtoReflect()) {
		return nil, http.StatusBadRequest
	}
	canonical, err := h.marshal(executionRequest)
	if err != nil || !bytes.Equal(encoded, canonical) {
		return nil, http.StatusBadRequest
	}
	if contextDeadlineExceeded(request.Context()) {
		return nil, http.StatusRequestTimeout
	}
	return executionRequest, 0
}

func (h *handler) encodeResponse(
	ctx context.Context,
	response *umpirespb.ExecuteResponse,
	executeErr error,
) ([]byte, int) {
	if executeErr != nil || response == nil || response.GetResult() == nil {
		if contextDeadlineExceeded(ctx) {
			return nil, http.StatusGatewayTimeout
		}
		return nil, http.StatusInternalServerError
	}
	if contextDeadlineExceeded(ctx) &&
		(response.GetResult().GetToolingStatus() != umpirespb.TOOLING_STATUS_CANCELED ||
			response.GetResult().GetDecision() != umpirespb.CANARY_DECISION_INCONCLUSIVE) {
		return nil, http.StatusGatewayTimeout
	}
	if !validProtoSemantics(response.ProtoReflect()) {
		return nil, http.StatusInternalServerError
	}
	result, err := h.marshal(response.GetResult())
	if err != nil || int64(len(result)) > h.maximumResultBytes {
		return nil, http.StatusInternalServerError
	}
	encoded, err := h.marshal(response)
	if err != nil || int64(len(encoded)) > h.maximumResultBytes+maximumResponseEnvelopeBytes {
		return nil, http.StatusInternalServerError
	}
	if contextDeadlineExceeded(ctx) {
		return nil, http.StatusGatewayTimeout
	}
	return encoded, 0
}

func setHTTPDeadlines(writer http.ResponseWriter, deadline time.Time) func() {
	controller := http.NewResponseController(writer)
	readSet := controller.SetReadDeadline(deadline) == nil
	writeSet := controller.SetWriteDeadline(deadline) == nil
	return func() {
		resetHTTPDeadline(readSet, controller.SetReadDeadline)
		resetHTTPDeadline(writeSet, controller.SetWriteDeadline)
	}
}

func resetHTTPDeadline(set bool, setDeadline func(time.Time) error) {
	if !set {
		return
	}
	if err := setDeadline(time.Time{}); err != nil {
		return
	}
}

func abortConnection(writer http.ResponseWriter) bool {
	connection, _, err := http.NewResponseController(writer).Hijack()
	if err != nil {
		return false
	}
	if err := connection.Close(); err != nil {
		return true
	}
	return true
}

func contextDeadlineExceeded(ctx context.Context) bool {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return true
	}
	deadline, ok := ctx.Deadline()
	return ok && !time.Now().Before(deadline)
}

func publishTransportFailure(
	clientContext context.Context,
	requestContext context.Context,
	writer http.ResponseWriter,
	status int,
) {
	deadlineExceeded := contextDeadlineExceeded(requestContext)
	if clientContext.Err() != nil && !deadlineExceeded {
		return
	}
	if deadlineExceeded && abortConnection(writer) {
		return
	}
	writeTransportStatus(writer, status)
}

func writeTransportStatus(writer http.ResponseWriter, status int) {
	writer.Header().Set("Content-Length", "0")
	writer.Header().Set("X-Content-Type-Options", "nosniff")
	writer.WriteHeader(status)
}

func validProtoSemantics(message protoreflect.Message) bool {
	if !message.IsValid() || len(message.GetUnknown()) != 0 {
		return false
	}
	valid := true
	message.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		switch {
		case field.IsList():
			list := value.List()
			for index := 0; index < list.Len(); index++ {
				if !validProtoValue(field, list.Get(index)) {
					valid = false
					return false
				}
			}
		case field.IsMap():
			value.Map().Range(func(_ protoreflect.MapKey, item protoreflect.Value) bool {
				valid = validProtoValue(field.MapValue(), item)
				return valid
			})
		default:
			valid = validProtoValue(field, value)
		}
		return valid
	})
	return valid
}

func validProtoValue(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
	switch field.Kind() {
	case protoreflect.EnumKind:
		return field.Enum().Values().ByNumber(value.Enum()) != nil
	case protoreflect.MessageKind, protoreflect.GroupKind:
		return validProtoSemantics(value.Message())
	default:
		return true
	}
}
