package main

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

func TestConcreteSchemaProjectionRetainsPresenceAndRecursiveTypes(t *testing.T) {
	document, err := buildProjection(basicDescriptorSet(t))
	require.NoError(t, err)
	artifacts, err := generateArtifacts(fixtureTestConfiguration, document)
	require.NoError(t, err)
	api := string(artifacts["Fixture/API.lean"])
	require.Contains(t, api, "valueShape := some (.message")
	require.Contains(t, api, "(.message \"fixture.messaging.shared.v1.Right\")")
	require.Contains(t, api, "(.enumeration false [0, 1])")
}

func TestConcreteFixtureMatchesDecodedProtobufPresenceBytesEnumsAndMaps(t *testing.T) {
	files, err := protodesc.NewFiles(basicDescriptorSet(t))
	require.NoError(t, err)
	descriptor, err := files.FindDescriptorByName("fixture.messaging.public.v1.Message")
	require.NoError(t, err)
	message := dynamicpb.NewMessage(descriptor.(protoreflect.MessageDescriptor))
	raw := []byte{
		18, 12, 10, 1, 'z', 18, 7, 10, 5, 'f', 'i', 'r', 's', 't',
		18, 10, 10, 1, 'a', 18, 5, 10, 3, 't', 'w', 'o',
		18, 11, 10, 1, 'z', 18, 6, 10, 4, 'l', 'a', 's', 't',
		42, 0, 50, 2, 0, 255, 58, 2, 8, 99,
	}
	require.NoError(t, proto.Unmarshal(raw, message))
	fields := message.Descriptor().Fields()
	require.True(t, message.Has(fields.ByName("note")))
	require.Empty(t, message.Get(fields.ByName("note")).String())
	require.Equal(t, []byte{0, 255}, message.Get(fields.ByName("payload")).Bytes())
	nested := message.Get(fields.ByName("nested")).Message()
	require.Equal(t, protoreflect.EnumNumber(99), nested.Get(nested.Descriptor().Fields().ByName("state")).Enum())
	attributes := message.Get(fields.ByName("attributes")).Map()
	require.Equal(t, 2, attributes.Len())
	last := attributes.Get(protoreflect.ValueOfString("z").MapKey()).Message()
	require.Equal(t, "last", last.Get(last.Descriptor().Fields().ByName("id")).String())
	encoded, err := (proto.MarshalOptions{Deterministic: true}).Marshal(message)
	require.NoError(t, err)
	require.Equal(t, []byte{
		18, 10, 10, 1, 'a', 18, 5, 10, 3, 't', 'w', 'o',
		18, 11, 10, 1, 'z', 18, 6, 10, 4, 'l', 'a', 's', 't',
		42, 0, 50, 2, 0, 255, 58, 2, 8, 99,
	}, encoded)
	empty := dynamicpb.NewMessage(message.Descriptor())
	require.False(t, empty.Has(fields.ByName("note")))
	empty.Set(fields.ByName("payload"), protoreflect.ValueOfBytes([]byte{255, 0}))
	require.NotEqual(t, message.Get(fields.ByName("payload")).Bytes(), empty.Get(fields.ByName("payload")).Bytes())
}
