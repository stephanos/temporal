package umpire3test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/environment"
	"go.temporal.io/server/tests/umpire3/protocol"
	"go.temporal.io/server/tests/umpire3/regress"
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
	capabilities []string
	prepareCount int
}

func (f *facadeFactory) Capabilities() []string { return f.capabilities }
func (f *facadeFactory) Prepare(context.Context, protocol.Experiment) (environment.Session, error) {
	f.prepareCount++
	return facadeSession{}, nil
}

type facadeSession struct{}

func (facadeSession) Realize(context.Context, protocol.Action, environment.Bindings) (environment.ActionEvidence, error) {
	return environment.ActionEvidence{Source: "facade-test", Reference: "action"}, nil
}
func (facadeSession) Observe(_ context.Context, checkpoint protocol.Checkpoint, _ environment.Bindings) (environment.Observation, error) {
	return environment.Observation{
		CheckpointID: checkpoint.Identifier, Kind: checkpoint.Observation, Satisfied: true,
		Source: "facade-test", SourceIdentity: "facade-source", ClockDomain: "facade-sequence",
		SourceSequence: 1, Reference: "facade/" + checkpoint.Identifier, CausalReference: "cause",
		EntityIdentity: "callback", Lineage: []string{"namespace", "callback"},
	}, nil
}
func (facadeSession) Cleanup(context.Context) environment.CleanupResult {
	return environment.CleanupResult{Complete: true}
}
func (facadeSession) RecoveryMetadata() map[string]string { return nil }

func TestRequireRegressionRunsTypedScenario(t *testing.T) {
	t.Parallel()

	test := &fakeTestingT{name: "TestRequireRegressionRunsTypedScenario"}
	factory := &facadeFactory{capabilities: []string{"history-observation"}}
	scenario := regress.NewScenario("callback-response", protocol.TargetIDProtocolAtomic,
		[]regress.Resource{regress.Callback("callback")},
		regress.OnePath(
			regress.RecordCallbackResponse("respond"),
			regress.RequireCallbackResponseConsistency(),
		),
	)

	RequireRegression(test, scenario, WithEnvironment(factory), WithCompilerLimits(testCompilerLimits()))
	require.Positive(t, test.helperCalls)
	require.Equal(t, 1, factory.prepareCount)
}

func TestRequireRegressionReportsStableCompileCategory(t *testing.T) {
	t.Parallel()

	test := &fakeTestingT{name: "TestRequireRegressionReportsStableCompileCategory"}
	require.PanicsWithValue(t,
		"Umpire3 scenario compilation failed: invalid-intent: scenario identifier and resources are required",
		func() {
			RequireRegression(test, regress.Scenario{}, WithCompilerLimits(testCompilerLimits()))
		})
}

func TestRequireRegressionReportsUnsupportedBeforeAllocation(t *testing.T) {
	t.Parallel()

	test := &fakeTestingT{name: "TestUnsupported/Profile"}
	factory := &facadeFactory{}
	scenario := regress.NewScenario("callback-response", protocol.TargetIDProtocolAtomic,
		[]regress.Resource{regress.Callback("callback")},
		regress.OnePath(
			regress.RecordCallbackResponse("respond"),
			regress.RequireCallbackResponseConsistency(),
		),
	)

	require.PanicsWithValue(t,
		"Umpire3 regression failed\nclaim: unsupported\nreason: missing capabilities: [history-observation]\npath: [respond]\ngrounded bindings: map[]\nomissions: []\ncleanup: {Complete:false Error: RecoverableResources:map[]}\nartifact: not retained\nreplay: go test -run '^TestUnsupported/Profile$'",
		func() {
			RequireRegression(test, scenario, WithEnvironment(factory), WithCompilerLimits(testCompilerLimits()))
		})
	require.Zero(t, factory.prepareCount)
}

func testCompilerLimits() CompilerLimits {
	return CompilerLimits{
		MaxPaths: 2, MaxActions: 8, MaxStates: 32, MaxMemoryBytes: 1 << 20, MaxTime: time.Second,
	}
}
