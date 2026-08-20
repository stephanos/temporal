package main

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestTransportPresenceDistinguishesAbsentAndPresentDefault(t *testing.T) {
	message := dynamicMessage(t, "temporal.api.failure.v1.Failure")
	field := message.Descriptor().Fields().ByName("cause")
	require.False(t, message.Has(field))

	message.Set(field, protoreflect.ValueOfMessage(dynamicpb.NewMessage(field.Message())))
	require.True(t, message.Has(field))
}

func TestTransportPreservesUnknownEnumNumber(t *testing.T) {
	message := dynamicMessage(t, "temporal.api.taskqueue.v1.TaskQueue")
	field := message.Descriptor().Fields().ByName("kind")
	message.Set(field, protoreflect.ValueOfEnum(123456))

	require.Equal(t, protoreflect.EnumNumber(123456), message.Get(field).Enum())
}

func TestTransportOneofReplacementKeepsLastValue(t *testing.T) {
	message := dynamicMessage(t, "temporal.api.failure.v1.Failure")
	application := message.Descriptor().Fields().ByName("application_failure_info")
	timeout := message.Descriptor().Fields().ByName("timeout_failure_info")
	message.Set(application, protoreflect.ValueOfMessage(dynamicpb.NewMessage(application.Message())))
	require.True(t, message.Has(application))

	message.Set(timeout, protoreflect.ValueOfMessage(dynamicpb.NewMessage(timeout.Message())))
	require.False(t, message.Has(application))
	require.True(t, message.Has(timeout))
}

func TestTransportMapEncodingIsDeterministic(t *testing.T) {
	first := payloadWithMetadata(t, []string{"z", "a"})
	second := payloadWithMetadata(t, []string{"a", "z"})
	options := proto.MarshalOptions{Deterministic: true}
	firstEncoded, err := options.Marshal(first)
	require.NoError(t, err)
	secondEncoded, err := options.Marshal(second)
	require.NoError(t, err)
	require.Equal(t, firstEncoded, secondEncoded)
}

func TestTransportRepeatedFieldPreservesSourceOrder(t *testing.T) {
	message := dynamicMessage(t, "temporal.api.common.v1.RetryPolicy")
	field := message.Descriptor().Fields().ByName("non_retryable_error_types")
	list := message.Mutable(field).List()
	list.Append(protoreflect.ValueOfString("second"))
	list.Append(protoreflect.ValueOfString("first"))

	encoded, err := proto.Marshal(message)
	require.NoError(t, err)
	roundTripped := dynamicpb.NewMessage(message.Descriptor())
	require.NoError(t, proto.Unmarshal(encoded, roundTripped))
	require.Equal(t, "second", roundTripped.Get(field).List().Get(0).String())
	require.Equal(t, "first", roundTripped.Get(field).List().Get(1).String())
}

func TestTransportDurationRetainsNegativeAndRejectsOverflow(t *testing.T) {
	require.NoError(t, durationpb.New(-1).CheckValid())
	require.Error(t, (&durationpb.Duration{Seconds: 315576000001}).CheckValid())
}

func dynamicMessage(t *testing.T, fullName protoreflect.FullName) *dynamicpb.Message {
	t.Helper()
	descriptor, err := protoregistry.GlobalFiles.FindDescriptorByName(fullName)
	require.NoError(t, err)
	messageDescriptor, ok := descriptor.(protoreflect.MessageDescriptor)
	require.True(t, ok)
	return dynamicpb.NewMessage(messageDescriptor)
}

func payloadWithMetadata(t *testing.T, keys []string) *dynamicpb.Message {
	t.Helper()
	message := dynamicMessage(t, "temporal.api.common.v1.Payload")
	field := message.Descriptor().Fields().ByName("metadata")
	metadata := message.Mutable(field).Map()
	for _, key := range keys {
		metadata.Set(protoreflect.ValueOfString(key).MapKey(), protoreflect.ValueOfBytes([]byte(key)))
	}
	return message
}
