package testpilot

import (
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
)

func TestCaseEnvironmentBindingSchema(t *testing.T) {
	// A Program declares no environment bindings: preparation derives them from its roles and
	// references.
	require.Nil(t, (&testpilotspb.Program{}).ProtoReflect().Descriptor().Fields().ByName("environment"))
	roleFields := (&testpilotspb.Role{}).ProtoReflect().Descriptor().Fields()
	require.EqualValues(t, 3, roleFields.ByName("namespace_binding_id").Number())
	require.EqualValues(t, 4, roleFields.ByName("resource_binding_id").Number())
	referenceEnvironment := (&testpilotspb.Reference{}).ProtoReflect().Descriptor().
		Fields().ByName("environment_binding_id")
	require.NotNil(t, referenceEnvironment)
	require.EqualValues(t, 4, referenceEnvironment.Number())

}
