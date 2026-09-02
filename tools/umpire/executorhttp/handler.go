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
	maximumRequestBytes          = evaluationcontract.MaximumContractBytes +
		2*artifact.MaximumDocumentBytes + maximumProtobufEnvelopeBytes
	maximumResponseBytes   = evaluationcontract.MaximumResultBytes
	maximumRequestDuration = time.Duration(evaluationcontract.MaximumDurationMillis) * time.Millisecond
)

var deterministicMarshal = proto.MarshalOptions{Deterministic: true}

type executeFunc func(context.Context, *umpirespb.ExecuteRequest) (*umpirespb.ExecuteResponse, error)

// Executor is the transport-independent resident execution seam.
type Executor interface {
	Execute(context.Context, *umpirespb.ExecuteRequest) (*umpirespb.ExecuteResponse, error)
}

type handler struct {
	execute              executeFunc
	maximumRequestBytes  int64
	maximumResponseBytes int64
	maximumDuration      time.Duration
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
		maximumResponseBytes,
		maximumRequestDuration,
	)
}

func newHandler(
	execute executeFunc,
	requestBytes int64,
	responseBytes int64,
	duration time.Duration,
) http.Handler {
	return &handler{
		execute: execute, maximumRequestBytes: requestBytes,
		maximumResponseBytes: responseBytes, maximumDuration: duration,
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
	if h == nil || h.execute == nil || h.maximumRequestBytes <= 0 ||
		h.maximumResponseBytes <= 0 || h.maximumDuration <= 0 {
		writeTransportStatus(writer, http.StatusInternalServerError)
		return
	}
	executionRequest, status := h.decodeRequest(writer, request)
	if status != 0 {
		writeTransportStatus(writer, status)
		return
	}

	ctx, cancel := context.WithTimeout(request.Context(), h.maximumDuration)
	defer cancel()
	response, err := h.execute(ctx, executionRequest)
	if request.Context().Err() != nil {
		return
	}
	encoded, status := h.encodeResponse(ctx, response, err)
	if status != 0 {
		writeTransportStatus(writer, status)
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
	canonical, err := deterministicMarshal.Marshal(executionRequest)
	if err != nil || !bytes.Equal(encoded, canonical) {
		return nil, http.StatusBadRequest
	}
	return executionRequest, 0
}

func (h *handler) encodeResponse(
	ctx context.Context,
	response *umpirespb.ExecuteResponse,
	executeErr error,
) ([]byte, int) {
	if executeErr != nil || response == nil || response.GetResult() == nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, http.StatusGatewayTimeout
		}
		return nil, http.StatusInternalServerError
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) &&
		(response.GetResult().GetToolingStatus() != umpirespb.TOOLING_STATUS_CANCELED ||
			response.GetResult().GetDecision() != umpirespb.CANARY_DECISION_INCONCLUSIVE) {
		return nil, http.StatusGatewayTimeout
	}
	if !validProtoSemantics(response.ProtoReflect()) {
		return nil, http.StatusInternalServerError
	}
	encoded, err := deterministicMarshal.Marshal(response)
	if err != nil || int64(len(encoded)) > h.maximumResponseBytes {
		return nil, http.StatusInternalServerError
	}
	return encoded, 0
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
