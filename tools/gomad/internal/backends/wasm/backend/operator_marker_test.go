package backend

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/backends/target"
	simulation "go.temporal.io/server/tools/gomad/internal/backends/wasm/cluster"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
	model "go.temporal.io/server/tools/gomad/internal/model/cluster"
)

func TestScenarioPreservesChoiceCapacityBeforeDeclaredMarker(t *testing.T) {
	p, source := inspectionArtifactFixture(t)
	handle := model.NodeHandle{Node: "a", Incarnation: 1}
	for _, action := range []string{"crash", "stop"} {
		t.Run(action, func(t *testing.T) {
			s := simulation.OperatorScenario{Schema: simulation.OperatorSchema, Spec: model.Spec{Schema: model.SpecSchema, Backend: model.BackendWASM, Fidelity: model.FidelitySimulationModel, Limits: model.DefaultLimits(), Nodes: []model.NodeSpec{{ID: handle.Node, Boot: "main", Address: "10.0.0.1"}}}, Boots: []simulation.OperatorBoot{{ID: "main"}}, Actions: []simulation.OperatorAction{{Action: "start", Handle: handle}, {Action: action, Handle: handle}}}
			boundary := []simulation.OutputCrash{{Handle: handle, Marker: []byte("after initial runtime choices")}}
			if action == "crash" {
				s.OutputCrashes = boundary
			} else {
				s.OutputStops = boundary
			}
			data, err := json.Marshal(s)
			require.NoError(t, err)
			prepared, err := p.Prepare(t.Context(), target.Spec{Backend: Name, Kind: target.KindGoRun, Source: ".", WorkingDir: source, PreparationRoot: t.TempDir(), BuildTags: []string{"gomad", "test_dep"}, Scenario: data})
			require.NoError(t, err)
			_, err = p.Execute(t.Context(), backends.ExecutionRequest{Target: prepared, OutputLimit: 1 << 20, ExecutionTimeout: time.Minute, Choice: &backends.ChoiceCapability{Mode: choice.ModeRecord, Limit: choice.MinimumTraceBytes}})
			var failure *backends.FailureError
			require.ErrorAs(t, err, &failure)
			require.Equal(t, backends.Capacity, failure.Termination)
			require.Contains(t, failure.Message, "runtime-choice-bytes")
		})
	}
}

func TestScenarioMarkerClassificationsPreserveReplayPrecedence(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
		want backends.Termination
	}{
		{"unsupported", &wasi.BoundaryError{Kind: "unsupported", Dimension: "CPU clocks outside stock WASI profile"}, backends.Unsupported},
		{"selected divergence", &wasi.BoundaryError{Kind: "divergence", Dimension: "runtime tape differs"}, backends.Divergence},
		{"invalid", &wasi.BoundaryError{Kind: "invalid", Dimension: "malformed evidence"}, backends.Infrastructure},
		{"infrastructure", &wasi.BoundaryError{Kind: "infrastructure", Dimension: "transport failed"}, backends.Infrastructure},
		{"runtime replay", &wasi.RuntimeReplayError{}, backends.Divergence},
		{"model replay precedes unsupported", errors.Join(&wasi.BoundaryError{Kind: "unsupported"}, &model.ReplayDivergenceError{}), backends.Divergence},
		{"runtime replay precedes capacity", errors.Join(&wasi.BoundaryError{Kind: "capacity"}, &wasi.RuntimeReplayError{}), backends.Divergence},
	} {
		t.Run(test.name, func(t *testing.T) {
			var failure *backends.FailureError
			require.ErrorAs(t, scenarioFailure(test.err), &failure)
			require.Equal(t, test.want, failure.Termination)
			require.Contains(t, failure.Message, test.err.Error())
		})
	}
}

func TestScenarioPreservesUnsupportedBeforeDeclaredMarker(t *testing.T) {
	p, _ := inspectionArtifactFixture(t)
	source, err := filepath.Abs("../testdata/unsupported_clock")
	require.NoError(t, err)
	handle := model.NodeHandle{Node: "a", Incarnation: 1}
	for _, action := range []string{"crash", "stop"} {
		t.Run(action, func(t *testing.T) {
			s := simulation.OperatorScenario{Schema: simulation.OperatorSchema, Spec: model.Spec{Schema: model.SpecSchema, Backend: model.BackendWASM, Fidelity: model.FidelitySimulationModel, Limits: model.DefaultLimits(), Nodes: []model.NodeSpec{{ID: handle.Node, Boot: "main", Address: "10.0.0.1"}}}, Boots: []simulation.OperatorBoot{{ID: "main"}}, Actions: []simulation.OperatorAction{{Action: "start", Handle: handle}, {Action: action, Handle: handle}}}
			boundary := []simulation.OutputCrash{{Handle: handle, Marker: []byte("after CPU clock call")}}
			if action == "crash" {
				s.OutputCrashes = boundary
			} else {
				s.OutputStops = boundary
			}
			data, err := json.Marshal(s)
			require.NoError(t, err)
			prepared, err := p.Prepare(t.Context(), target.Spec{Backend: Name, Kind: target.KindGoRun, Source: ".", WorkingDir: source, PreparationRoot: t.TempDir(), BuildTags: []string{"gomad", "test_dep"}, Scenario: data})
			require.NoError(t, err)
			_, err = p.Execute(t.Context(), backends.ExecutionRequest{Target: prepared, OutputLimit: 1 << 20, ExecutionTimeout: time.Minute})
			var failure *backends.FailureError
			require.ErrorAs(t, err, &failure)
			require.Equal(t, backends.Unsupported, failure.Termination)
			require.Contains(t, failure.Message, "CPU clocks outside stock WASI profile")
		})
	}
}
