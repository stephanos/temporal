package regression

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	environment "go.temporal.io/server/tools/umpire3/execution"
	"go.temporal.io/server/tools/umpire3/execution/observation"
	protocolcatalog "go.temporal.io/server/tools/umpire3/protocol/catalog"
	protocolexperiment "go.temporal.io/server/tools/umpire3/protocol/experiment"
	"go.temporal.io/server/tools/umpire3/scenario"
)

type fakeTestingT struct {
	name        string
	helperCalls int
}

func (t *fakeTestingT) Helper() { t.helperCalls++ }
func (t *fakeTestingT) Name() string {
	return t.name
}
func (t *fakeTestingT) Fatalf(format string, arguments ...any) {
	panic(fmt.Sprintf(format, arguments...))
}

type facadeFactory struct {
	capabilities []protocolcatalog.CapabilityID
	prepareCount int
	violating    bool
}

func (f *facadeFactory) Capabilities() []protocolcatalog.CapabilityID { return f.capabilities }
func (f *facadeFactory) Prepare(
	context.Context,
	protocolexperiment.Experiment,
) (environment.PreparedEnvironment, error) {
	f.prepareCount++
	return environment.PreparedEnvironment{
		Session: facadeSession{violating: f.violating},
		Identity: environment.EnvironmentIdentity{
			Name: "test", BuildID: "build", ConfigurationIdentity: "configuration",
			EvidenceProfile: environment.EvidenceProfileInProcessHooks, DrivingAuthority: "driver",
			ObservationAuthority: "observer", FaultAuthority: "none",
			IsolationIdentity: "namespace/queue", RetentionClass: "semantic-redacted",
			Capabilities: f.Capabilities(),
		},
	}, nil
}

type facadeSession struct {
	violating bool
}

func (facadeSession) Realize(context.Context, protocolexperiment.Action, environment.Bindings) (environment.ActionEvidence, error) {
	return environment.ActionEvidence{
		Source: "facade-test", Outcome: protocolexperiment.ActionOutcomeApplied, Reference: "action",
	}, nil
}
func (s facadeSession) ObserveFacts(
	_ context.Context,
	checkpoint protocolexperiment.Checkpoint,
	_ environment.Bindings,
) ([]observation.Fact, error) {
	kinds := []string{
		observation.CallbackRegistered,
		observation.CallbackOperationSettled,
		observation.CallbackResponseRecorded,
	}
	if s.violating {
		kinds = []string{observation.CallbackResponseConflict}
	}
	facts := make([]observation.Fact, len(kinds), len(kinds)+1)
	for index, kind := range kinds {
		sequence := int64(index + 1)
		facts[index] = observation.Fact{
			Identifier: "facade/" + kind,
			Source: observation.Source{
				Identity: "facade-source", ClockDomain: "facade-sequence", Sequence: sequence,
				Reference: "facade/reference/" + kind, CausalReferences: []string{"facade/cause"},
				EntityIdentity: "callback", Lineage: []string{"namespace", "callback"},
			},
			History: &observation.HistoryEvent{
				EventType: kind, EventID: sequence, WorkflowID: "workflow", RunID: "run",
			},
		}
	}
	sequence := int64(len(kinds) + 1)
	facts = append(facts, observation.Fact{
		Identifier: "facade/window/" + checkpoint.Observation,
		Source: observation.Source{
			Identity: "facade-source", ClockDomain: "facade-sequence", Sequence: sequence,
			Reference: "facade/reference/window", CausalReferences: []string{"facade/cause"},
			EntityIdentity: "callback", Lineage: []string{"namespace", "callback"},
		},
		Window: &observation.EvidenceWindow{
			Purpose: checkpoint.Observation, Closed: true, ThroughSequence: sequence,
		},
	})
	return facts, nil
}
func (facadeSession) Cleanup(context.Context) environment.CleanupResult {
	return environment.CleanupResult{Complete: true}
}
func (facadeSession) RecoveryMetadata() map[string]string { return nil }

func TestRequireRegressionRunsTypedScenario(t *testing.T) {
	t.Parallel()

	test := &fakeTestingT{name: "TestRequireRegressionRunsTypedScenario"}
	factory := &facadeFactory{capabilities: []protocolcatalog.CapabilityID{protocolcatalog.CapabilityIDHistoryObservation}}
	authored := scenario.ProtocolAtomicScenario("callback-response",
		[]scenario.Resource{scenario.Callback("callback")},
		scenario.OnePath(
			scenario.RecordCallbackResponse("respond"),
			scenario.RequireCallbackResponseConsistency(),
		),
	)

	RequireRegression(test, authored, WithEnvironment(factory), WithCompilerLimits(testCompilerLimits()))
	require.Positive(t, test.helperCalls)
	require.Equal(t, 1, factory.prepareCount)
}

func TestRequireRegressionAcceptsExpectedViolation(t *testing.T) {
	t.Parallel()

	test := &fakeTestingT{name: "TestRequireRegressionAcceptsExpectedViolation"}
	factory := &facadeFactory{
		capabilities: []protocolcatalog.CapabilityID{protocolcatalog.CapabilityIDHistoryObservation},
		violating:    true,
	}
	authored := scenario.ProtocolAtomicScenario("callback-response",
		[]scenario.Resource{scenario.Callback("callback")},
		scenario.OnePath(
			scenario.RecordCallbackResponse("respond"),
			scenario.RequireCallbackResponseConsistency(),
		),
	)

	RequireRegression(test, authored, WithEnvironment(factory), ExpectViolation(),
		WithCompilerLimits(testCompilerLimits()))
	require.Positive(t, test.helperCalls)
	require.Equal(t, 1, factory.prepareCount)
}

func TestRequireRegressionReportsStableCompileCategory(t *testing.T) {
	t.Parallel()

	test := &fakeTestingT{name: "TestRequireRegressionReportsStableCompileCategory"}
	require.PanicsWithValue(t,
		"Umpire3 scenario compilation failed: invalid-intent: scenario identifier and resources are required",
		func() {
			RequireRegression(test, scenario.Scenario{}, WithCompilerLimits(testCompilerLimits()))
		})
}

func TestRequireRegressionRequiresExecutionFactory(t *testing.T) {
	t.Parallel()

	test := &fakeTestingT{name: "TestRequireRegressionRequiresExecutionFactory"}
	authored := scenario.ProtocolAtomicScenario("callback-response",
		[]scenario.Resource{scenario.Callback("callback")},
		scenario.OnePath(
			scenario.RecordCallbackResponse("respond"),
			scenario.RequireCallbackResponseConsistency(),
		),
	)

	require.PanicsWithValue(t, "Umpire3 regression requires an execution factory", func() {
		RequireRegression(test, authored, WithCompilerLimits(testCompilerLimits()))
	})
}

func TestRequireRegressionReportsUnsupportedBeforeAllocation(t *testing.T) {
	t.Parallel()

	test := &fakeTestingT{name: "TestUnsupported/Profile"}
	factory := &facadeFactory{}
	authored := scenario.ProtocolAtomicScenario("callback-response",
		[]scenario.Resource{scenario.Callback("callback")},
		scenario.OnePath(
			scenario.RecordCallbackResponse("respond"),
			scenario.RequireCallbackResponseConsistency(),
		),
	)

	require.PanicsWithValue(t,
		"Umpire3 regression failed\nclaim: unsupported\nreason: missing capabilities: [history-observation]\npath: [respond]\ngrounded bindings: map[]\nomissions: []\ncleanup: {Complete:false Error: RecoverableResources:map[]}\nartifact: not retained\nreplay: go test -run '^TestUnsupported/Profile$'",
		func() {
			RequireRegression(test, authored, WithEnvironment(factory), WithCompilerLimits(testCompilerLimits()))
		})
	require.Zero(t, factory.prepareCount)
}

func testCompilerLimits() CompilerLimits {
	return CompilerLimits{
		MaxPaths: 2, MaxActions: 8, MaxStates: 32, MaxMemoryBytes: 1 << 20, MaxTime: time.Second,
	}
}
