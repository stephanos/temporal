package testpilot

import (
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	temporaldriver "go.temporal.io/server/common/testing/testpilot/temporal"
	"google.golang.org/protobuf/proto"
)

func asyncNexusEnvironment() temporaldriver.Environment {
	return temporaldriver.Environment{
		Identity: "nexus-caller-profile", Namespace: "namespace",
		TaskQueue: "task-queue", HandlerTaskQueue: "task-queue-handler", NexusEndpoint: "nexus-endpoint",
	}
}

// The hand-written Profile is the derivation oracle: if deriving the same Case does not reproduce
// it field for field, the derivation is guessing rather than reading the Case.
func TestDeriveProfileEqualsTheHandWrittenAsyncNexusProfile(t *testing.T) {
	source := loadLeanCase(t, NexusCallerAsyncCompletionFixture)
	catalog, err := temporaldriver.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	environment := asyncNexusEnvironment()

	derived, err := temporaldriver.DeriveProfile(source, catalog, environment)
	require.NoError(t, err)
	oracle := NexusCallerProfile(catalog, NexusCallerEnvironment{
		Namespace: environment.Namespace, TaskQueue: environment.TaskQueue,
		HandlerTaskQueue: environment.HandlerTaskQueue, NexusEndpoint: environment.NexusEndpoint,
	})
	require.Equal(t, oracle.Identity, derived.Identity)
	require.Equal(t, oracle.Roles, derived.Roles)
	require.Equal(t, oracle.Opcodes, derived.Opcodes)
	require.Equal(t, oracle.EnvironmentBindings, derived.EnvironmentBindings)
	require.Same(t, oracle.Catalog, derived.Catalog)
	require.True(t, proto.Equal(oracle.ProgramLimits, derived.ProgramLimits))
	require.True(t, proto.Equal(oracle.ContractLimits, derived.ContractLimits))
}

// The typed Cases pin the parts the caller set's shape cannot: two reserved Nexus handler
// activations on one carrier, and a Case with no Nexus handler at all.
func TestDeriveProfileEqualsTheHandWrittenTypedProfiles(t *testing.T) {
	catalog, err := temporaldriver.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	nexus := loadLeanCase(t, "typed-nexus")
	derived, err := temporaldriver.DeriveProfile(nexus, catalog, temporaldriver.Environment{
		Identity: "typed-nexus-profile", Namespace: "namespace", TaskQueue: "task-queue", NexusEndpoint: "nexus-endpoint",
	})
	require.NoError(t, err)
	oracle := TypedNexusProfile(catalog, TypedNexusEnvironment{
		Namespace: "namespace", TaskQueue: "task-queue", NexusEndpoint: "nexus-endpoint",
	})
	require.Equal(t, oracle.Roles, derived.Roles)
	require.Equal(t, oracle.Opcodes, derived.Opcodes)
	require.Equal(t, oracle.EnvironmentBindings, derived.EnvironmentBindings)

	start := loadLeanCase(t, WorkflowStartFixture)
	derivedStart, err := temporaldriver.DeriveProfile(start, catalog, temporaldriver.Environment{
		Identity: "workflow-start-profile", Namespace: "namespace", TaskQueue: "task-queue",
	})
	require.NoError(t, err)
	startOracle := WorkflowStartProfile(catalog, WorkflowStartEnvironment{
		Namespace: "namespace", TaskQueue: "task-queue",
	})
	require.Equal(t, startOracle.Roles, derivedStart.Roles)
	require.Equal(t, startOracle.Opcodes, derivedStart.Opcodes)
	require.Equal(t, startOracle.EnvironmentBindings, derivedStart.EnvironmentBindings)
}

// The environment's dynamic configuration -- a switch value, a bound setup parameter -- is recorded
// on the Profile in the catalog's spelling, sorted, and is part of the binding fingerprint, so a
// Case run under two switch values runs under two Profiles over the same bytes.
func TestDeriveProfileRecordsTheEnvironmentsConfiguration(t *testing.T) {
	source := loadLeanCase(t, NexusCallerAsyncCompletionFixture)
	catalog, err := temporaldriver.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	environment := asyncNexusEnvironment()
	environment.DynamicConfig = map[string]string{
		"nexusoperation.enableChasmWorkflowOperations": "true",
		"history.enableChasm":                          "true",
	}

	derived, err := temporaldriver.DeriveProfile(source, catalog, environment)
	require.NoError(t, err)
	require.Equal(t, []testpilot.ConfigurationValue{
		{Key: "history.enablechasm", Value: "true"},
		{Key: "nexusoperation.enablechasmworkflowoperations", Value: "true"},
	}, derived.Configuration)

	plain, err := temporaldriver.DeriveProfile(source, catalog, asyncNexusEnvironment())
	require.NoError(t, err)
	require.Empty(t, plain.Configuration)
	plainFingerprint, err := plain.BindingFingerprint()
	require.NoError(t, err)
	configuredFingerprint, err := derived.BindingFingerprint()
	require.NoError(t, err)
	require.NotEqual(t, plainFingerprint, configuredFingerprint)

	// The same configuration spelled with a different case is the same Profile.
	respelled := asyncNexusEnvironment()
	respelled.DynamicConfig = map[string]string{
		"NexusOperation.EnableChasmWorkflowOperations": "true",
		"History.EnableChasm":                          "true",
	}
	derivedRespelled, err := temporaldriver.DeriveProfile(source, catalog, respelled)
	require.NoError(t, err)
	require.Equal(t, derived.Configuration, derivedRespelled.Configuration)

	for name, configuration := range map[string]map[string]string{
		"empty key":   {"": "true"},
		"empty value": {"history.enableChasm": ""},
		"two spellings of one key": {
			"history.enableChasm": "true", "history.enablechasm": "false",
		},
	} {
		t.Run(name, func(t *testing.T) {
			invalid := asyncNexusEnvironment()
			invalid.DynamicConfig = configuration
			_, err := temporaldriver.DeriveProfile(source, catalog, invalid)
			require.Error(t, err)
		})
	}
}

func TestDeriveProfileNeverWidensBeyondTheCase(t *testing.T) {
	catalog, err := temporaldriver.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	source := loadLeanCase(t, NexusCallerAsyncCompletionFixture)
	derived, err := temporaldriver.DeriveProfile(source, catalog, asyncNexusEnvironment())
	require.NoError(t, err)

	// Every derived method, carrier and opcode is one the Case itself references.
	methods, opcodes := map[string]bool{}, map[testpilot.Opcode]bool{}
	program := source.GetProgram()
	plans := append(program.GetEntrypoints(), &testpilotspb.Entrypoint{
		EntrypointId: program.GetCleanup().GetEntrypointId(), Instructions: program.GetCleanup().GetInstructions(),
	})
	for _, entrypoint := range plans {
		for _, instruction := range entrypoint.GetInstructions() {
			opcodes[testpilot.InstructionOpcode(instruction.GetInstruction())] = true
			if rpc := instruction.GetInstruction().GetInvokeRpc(); rpc != nil {
				methods[rpc.GetEndpointRoleId()+rpc.GetMethod()] = true
			}
		}
	}
	for _, opcode := range derived.Opcodes {
		require.True(t, opcodes[opcode])
	}
	for _, role := range derived.Roles {
		for _, method := range role.Methods {
			require.True(t, methods[role.ID+method])
		}
		for _, carrier := range role.ReservationCarriers {
			require.True(t, methods[role.ID+carrier.Method])
		}
	}

	// A carrier's shapes are the entrypoints its reservations reach, so a second node on the same
	// carrier must not raise the ceiling: it needs the same room, not twice it.
	twoNodes := proto.CloneOf(source)
	controller := twoNodes.Program.Entrypoints[0]
	second := proto.CloneOf(controller.Instructions[0])
	second.InstructionId = second.GetInstructionId() + "-again"
	controller.Instructions = append(controller.Instructions, second)
	derivedTwoNodes, err := temporaldriver.DeriveProfile(twoNodes, catalog, asyncNexusEnvironment())
	require.NoError(t, err)
	require.Equal(t, derived.Roles[0].ReservationCarriers, derivedTwoNodes.Roles[0].ReservationCarriers)

	// A Case with no worker roles yields no worker policy, no carriers, and no bindings.
	bare := proto.CloneOf(source)
	bare.Program.Roles = nil
	bare.Program.Entrypoints = bare.Program.Entrypoints[:1]
	bare.Program.Entrypoints[0].Instructions = nil
	bare.Program.Cleanup.Instructions = nil
	derivedBare, err := temporaldriver.DeriveProfile(bare, catalog, asyncNexusEnvironment())
	require.NoError(t, err)
	require.Empty(t, derivedBare.Roles)
	require.Empty(t, derivedBare.Opcodes)
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
		"unclaimed environment binding": func(c *testpilotspb.Case) {
			c.Program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().RequestAssignments[0].Value = &testpilotspb.Expression{Expression: &testpilotspb.Expression_Reference{Reference: &testpilotspb.Reference{Reference: &testpilotspb.Reference_EnvironmentBindingId{EnvironmentBindingId: "orphan"}}}}
		},
		"unset activation": func(c *testpilotspb.Case) { c.Program.Entrypoints[0].Activation = nil },
	} {
		t.Run(name, func(t *testing.T) {
			source := loadLeanCase(t, NexusCallerAsyncCompletionFixture)
			mutate(source)
			_, err := temporaldriver.DeriveProfile(source, catalog, asyncNexusEnvironment())
			require.Error(t, err)
		})
	}
	_, err = temporaldriver.DeriveProfile(nil, catalog, asyncNexusEnvironment())
	require.Error(t, err)
	_, err = temporaldriver.DeriveProfile(loadLeanCase(t, NexusCallerAsyncCompletionFixture), catalog, temporaldriver.Environment{})
	require.Error(t, err)
}
