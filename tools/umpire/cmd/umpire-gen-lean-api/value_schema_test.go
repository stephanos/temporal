package main

import (
	"testing"

	"github.com/stretchr/testify/require"
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
