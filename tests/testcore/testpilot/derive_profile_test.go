package testpilot_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	temporaldriver "go.temporal.io/server/common/testing/testpilot/temporal"
	testpilotfixture "go.temporal.io/server/tests/testcore/testpilot"
	"google.golang.org/protobuf/proto"
)

func loadCase(t testing.TB, name string) *testpilotspb.Case {
	t.Helper()
	encoded, err := os.ReadFile(filepath.Join("testdata", name+"-case.json"))
	require.NoError(t, err)
	decoded, err := testpilot.DecodeCaseProtoJSON(encoded)
	require.NoError(t, err)
	return decoded
}

func asyncNexusEnvironment() temporaldriver.Environment {
	return temporaldriver.Environment{
		Identity: "async-nexus-profile", Namespace: "namespace",
		TaskQueue: "task-queue", NexusEndpoint: "nexus-endpoint",
	}
}

// The hand-written Profile is the derivation oracle: if deriving the same Case does not reproduce
// it field for field, the derivation is guessing rather than reading the Case.
func TestDeriveProfileEqualsTheHandWrittenAsyncNexusProfile(t *testing.T) {
	source := loadCase(t, "async-nexus")
	catalog, err := temporaldriver.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	environment := asyncNexusEnvironment()

	derived, err := temporaldriver.DeriveProfile(source, catalog, environment)
	require.NoError(t, err)
	oracle := testpilotfixture.AsyncNexusProfile(catalog, source, testpilotfixture.AsyncNexusEnvironment{
		Namespace: environment.Namespace, TaskQueue: environment.TaskQueue, NexusEndpoint: environment.NexusEndpoint,
	})
	require.Equal(t, oracle.Identity, derived.Identity)
	require.Equal(t, oracle.Roles, derived.Roles)
	require.Equal(t, oracle.Capabilities, derived.Capabilities)
	require.Equal(t, oracle.EnvironmentBindings, derived.EnvironmentBindings)
	require.Same(t, oracle.Catalog, derived.Catalog)
	require.True(t, proto.Equal(oracle.ProgramLimits, derived.ProgramLimits))
	require.True(t, proto.Equal(oracle.ContractLimits, derived.ContractLimits))
}

// The typed Cases pin the parts the async-nexus shape cannot: two reserved Nexus handler
// activations on one carrier, and a Case with no Nexus handler at all.
func TestDeriveProfileEqualsTheHandWrittenTypedProfiles(t *testing.T) {
	catalog, err := temporaldriver.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	nexus := loadCase(t, "typed-nexus")
	derived, err := temporaldriver.DeriveProfile(nexus, catalog, temporaldriver.Environment{
		Identity: "typed-nexus-profile", Namespace: "namespace", TaskQueue: "task-queue", NexusEndpoint: "nexus-endpoint",
	})
	require.NoError(t, err)
	oracle := testpilotfixture.TypedNexusProfile(catalog, nexus, testpilotfixture.TypedNexusEnvironment{
		Namespace: "namespace", TaskQueue: "task-queue", NexusEndpoint: "nexus-endpoint",
	})
	require.Equal(t, oracle.Roles, derived.Roles)
	require.Equal(t, oracle.Capabilities, derived.Capabilities)
	require.Equal(t, oracle.EnvironmentBindings, derived.EnvironmentBindings)

	unary := loadCase(t, "typed-unary")
	derivedUnary, err := temporaldriver.DeriveProfile(unary, catalog, temporaldriver.Environment{
		Identity: "typed-unary-profile", Namespace: "namespace", TaskQueue: "task-queue",
	})
	require.NoError(t, err)
	unaryOracle := testpilotfixture.TypedUnaryProfile(catalog, unary, testpilotfixture.TypedUnaryEnvironment{
		Namespace: "namespace", TaskQueue: "task-queue",
	})
	require.Equal(t, unaryOracle.Roles, derivedUnary.Roles)
	require.Equal(t, unaryOracle.Capabilities, derivedUnary.Capabilities)
	require.Equal(t, unaryOracle.EnvironmentBindings, derivedUnary.EnvironmentBindings)
}

func TestDeriveProfileNeverWidensBeyondTheCase(t *testing.T) {
	catalog, err := temporaldriver.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	source := loadCase(t, "async-nexus")
	derived, err := temporaldriver.DeriveProfile(source, catalog, asyncNexusEnvironment())
	require.NoError(t, err)

	// Every derived method, carrier and capability is one the Case itself references.
	methods, capabilities := map[string]bool{}, map[testpilot.Capability]bool{}
	reserving := map[string]bool{}
	program := source.GetProgram()
	plans := append(program.GetEntrypoints(), &testpilotspb.EntrypointDefinition{
		EntrypointId: program.GetCleanup().GetEntrypointId(), Instructions: program.GetCleanup().GetInstructions(),
	})
	for _, entrypoint := range plans {
		for _, instruction := range entrypoint.GetInstructions() {
			capabilities[testpilot.InstructionCapability(instruction.GetInstruction())] = true
			if rpc := instruction.GetInstruction().GetInvokeRpc(); rpc != nil {
				methods[rpc.GetEndpointRoleId()+rpc.GetMethod()] = true
				if len(instruction.GetActivationReservations()) > 0 {
					reserving[rpc.GetEndpointRoleId()+rpc.GetMethod()] = true
				}
			}
		}
	}
	for _, capability := range derived.Capabilities {
		require.True(t, capabilities[capability])
	}
	for _, role := range derived.Roles {
		for _, method := range role.Methods {
			require.True(t, methods[role.ID+method])
		}
		for _, carrier := range role.ReservationCarriers {
			require.True(t, reserving[role.ID+carrier.Method])
		}
	}

	// A Case with no worker roles yields no worker policy, no carriers, and no bindings.
	bare := proto.CloneOf(source)
	bare.Program.Roles = nil
	bare.Program.Environment = nil
	bare.Program.Entrypoints = bare.Program.Entrypoints[:1]
	bare.Program.Entrypoints[0].Instructions = nil
	bare.Program.Cleanup.Instructions = nil
	derivedBare, err := temporaldriver.DeriveProfile(bare, catalog, asyncNexusEnvironment())
	require.NoError(t, err)
	require.Empty(t, derivedBare.Roles)
	require.Empty(t, derivedBare.Capabilities)
	require.Empty(t, derivedBare.EnvironmentBindings)
}

func TestDeriveProfileRejectsWhatItCannotRead(t *testing.T) {
	catalog, err := temporaldriver.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	for name, mutate := range map[string]func(*testpilotspb.Case){
		"unknown method": func(c *testpilotspb.Case) {
			c.Program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().Method = "/temporal.api.workflowservice.v1.WorkflowService/Missing"
		},
		"unknown role kind": func(c *testpilotspb.Case) { c.Program.Roles[0].Kind = 99 },
		"duplicate role":    func(c *testpilotspb.Case) { c.Program.Roles = append(c.Program.Roles, c.Program.Roles[0]) },
		"unset instruction": func(c *testpilotspb.Case) {
			c.Program.Entrypoints[0].Instructions[0].Instruction = &testpilotspb.Instruction{}
		},
		"undeclared reserved entrypoint": func(c *testpilotspb.Case) {
			c.Program.Entrypoints[0].Instructions[0].ActivationReservations[0].EntrypointId = "missing"
		},
		"unclaimed environment binding": func(c *testpilotspb.Case) {
			c.Program.Environment = append(c.Program.Environment, &testpilotspb.EnvironmentDefinition{BindingId: "orphan"})
		},
		"unset activation": func(c *testpilotspb.Case) { c.Program.Entrypoints[0].Activation = nil },
	} {
		t.Run(name, func(t *testing.T) {
			source := loadCase(t, "async-nexus")
			mutate(source)
			_, err := temporaldriver.DeriveProfile(source, catalog, asyncNexusEnvironment())
			require.Error(t, err)
		})
	}
	_, err = temporaldriver.DeriveProfile(nil, catalog, asyncNexusEnvironment())
	require.Error(t, err)
	_, err = temporaldriver.DeriveProfile(loadCase(t, "async-nexus"), catalog, temporaldriver.Environment{})
	require.Error(t, err)
}
