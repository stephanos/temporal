package wire

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strings"

	"go.temporal.io/api/serviceerror"
	_ "go.temporal.io/api/workflowservice/v1" // Register selected workflow-service descriptors.
	protocolcatalog "go.temporal.io/server/tools/umpire3/protocol/catalog"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type MutationKind string

const (
	MutationEmptyString      MutationKind = "empty-string"
	MutationNegativeDuration MutationKind = "negative-duration"
	MutationOverflowDuration MutationKind = "overflow-duration"
	MutationUnknownEnum      MutationKind = "unknown-enum-number"
)

type Mutation struct {
	Message     string       `json:"message"`
	Field       string       `json:"field"`
	Kind        MutationKind `json:"kind"`
	Disposition string       `json:"disposition"`
}

type Provenance struct {
	DescriptorDigest string       `json:"descriptorDigest"`
	Message          string       `json:"message"`
	Field            string       `json:"field"`
	Mutation         MutationKind `json:"mutation"`
	Disposition      string       `json:"disposition"`
	RequestDigest    string       `json:"requestDigest"`
}

type ResponseKind string

const (
	ResponseAccepted        ResponseKind = "accepted"
	ResponseRejected        ResponseKind = "rejected"
	ResponseUnsupported     ResponseKind = "unsupported"
	ResponseTimedOut        ResponseKind = "timed-out"
	ResponseTransportFailed ResponseKind = "transport-failed"
)

type Result struct {
	Provenance Provenance   `json:"provenance"`
	Response   ResponseKind `json:"response"`
	Code       string       `json:"code,omitempty"`
	Error      string       `json:"error,omitempty"`
}

type Invoke func(context.Context, proto.Message) (proto.Message, error)

func Catalog(message string) ([]Mutation, error) {
	inventory, err := protocolcatalog.DefaultProtobufInventory()
	if err != nil {
		return nil, err
	}
	prefix := message + "."
	var mutations []Mutation
	for _, field := range inventory.Fields {
		if !strings.HasPrefix(field.FullName, prefix) || field.Disposition != "interpreted" {
			continue
		}
		name := strings.TrimPrefix(field.FullName, prefix)
		switch {
		case field.Kind == "string":
			mutations = append(mutations, Mutation{
				Message: message, Field: name, Kind: MutationEmptyString, Disposition: field.Disposition,
			})
		case field.Kind == "enum":
			mutations = append(mutations, Mutation{
				Message: message, Field: name, Kind: MutationUnknownEnum, Disposition: field.Disposition,
			})
		case field.TypeName == "google.protobuf.Duration":
			mutations = append(mutations,
				Mutation{Message: message, Field: name, Kind: MutationNegativeDuration, Disposition: field.Disposition},
				Mutation{Message: message, Field: name, Kind: MutationOverflowDuration, Disposition: field.Disposition})
		default:
		}
	}
	slices.SortFunc(mutations, func(left, right Mutation) int {
		return strings.Compare(left.Field+"\x00"+string(left.Kind), right.Field+"\x00"+string(right.Kind))
	})
	if len(mutations) == 0 {
		return nil, fmt.Errorf("message %q has no generated interpreted mutations", message)
	}
	return mutations, nil
}

func Apply(base proto.Message, mutation Mutation) (proto.Message, Provenance, error) {
	if base == nil {
		return nil, Provenance{}, errors.New("protobuf mutation requires a base request")
	}
	messageName := string(base.ProtoReflect().Descriptor().FullName())
	if mutation.Message != messageName {
		return nil, Provenance{}, fmt.Errorf("mutation message %q does not match request %q", mutation.Message, messageName)
	}
	generated, err := Catalog(messageName)
	if err != nil {
		return nil, Provenance{}, err
	}
	if !slices.Contains(generated, mutation) {
		return nil, Provenance{}, fmt.Errorf("mutation %s.%s/%s was not generated for the selected field",
			mutation.Message, mutation.Field, mutation.Kind)
	}
	request := proto.Clone(base)
	reflection := request.ProtoReflect()
	field := reflection.Descriptor().Fields().ByName(protoreflect.Name(mutation.Field))
	if field == nil {
		return nil, Provenance{}, fmt.Errorf("request has no field %q", mutation.Field)
	}
	if err := applyMutation(reflection, field, mutation.Kind); err != nil {
		return nil, Provenance{}, err
	}
	encoded, err := proto.MarshalOptions{Deterministic: true}.Marshal(request)
	if err != nil {
		return nil, Provenance{}, fmt.Errorf("encode mutated request: %w", err)
	}
	digest := sha256.Sum256(encoded)
	inventory, err := protocolcatalog.DefaultProtobufInventory()
	if err != nil {
		return nil, Provenance{}, err
	}
	return request, Provenance{
		DescriptorDigest: inventory.DescriptorDigest, Message: mutation.Message, Field: mutation.Field,
		Mutation: mutation.Kind, Disposition: mutation.Disposition,
		RequestDigest: "sha256:" + hex.EncodeToString(digest[:]),
	}, nil
}

func Drive(ctx context.Context, base proto.Message, mutation Mutation, invoke Invoke) (Result, error) {
	if invoke == nil {
		return Result{}, errors.New("protobuf case requires an invocation")
	}
	request, provenance, err := Apply(base, mutation)
	if err != nil {
		return Result{}, err
	}
	_, invokeErr := invoke(ctx, request)
	result := Result{Provenance: provenance, Response: classifyResponse(invokeErr)}
	if invokeErr != nil {
		result.Code = serviceerror.ToStatus(invokeErr).Code().String()
		result.Error = invokeErr.Error()
	}
	return result, nil
}

func applyMutation(message protoreflect.Message, field protoreflect.FieldDescriptor, kind MutationKind) error {
	switch kind {
	case MutationEmptyString:
		if field.Kind() != protoreflect.StringKind {
			return errors.New("empty-string mutation requires a string field")
		}
		message.Set(field, protoreflect.ValueOfString(""))
	case MutationUnknownEnum:
		if field.Kind() != protoreflect.EnumKind {
			return errors.New("unknown-enum mutation requires an enum field")
		}
		message.Set(field, protoreflect.ValueOfEnum(protoreflect.EnumNumber(2147483647)))
	case MutationNegativeDuration, MutationOverflowDuration:
		if field.Message() == nil || field.Message().FullName() != "google.protobuf.Duration" {
			return errors.New("duration mutation requires a google.protobuf.Duration field")
		}
		duration := message.Mutable(field).Message()
		seconds := duration.Descriptor().Fields().ByName("seconds")
		nanos := duration.Descriptor().Fields().ByName("nanos")
		if kind == MutationNegativeDuration {
			duration.Set(seconds, protoreflect.ValueOfInt64(0))
			duration.Set(nanos, protoreflect.ValueOfInt32(-1))
		} else {
			duration.Set(seconds, protoreflect.ValueOfInt64(315576000001))
			duration.Set(nanos, protoreflect.ValueOfInt32(0))
		}
	default:
		return fmt.Errorf("unknown protobuf mutation %q", kind)
	}
	return nil
}

func classifyResponse(err error) ResponseKind {
	if err == nil {
		return ResponseAccepted
	}
	switch serviceerror.ToStatus(err).Code() {
	case codes.InvalidArgument, codes.NotFound, codes.AlreadyExists, codes.FailedPrecondition,
		codes.PermissionDenied, codes.Unauthenticated:
		return ResponseRejected
	case codes.Unimplemented:
		return ResponseUnsupported
	case codes.DeadlineExceeded, codes.Canceled:
		return ResponseTimedOut
	default:
		return ResponseTransportFailed
	}
}
