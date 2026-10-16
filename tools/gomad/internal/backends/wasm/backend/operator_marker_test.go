package backend

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/backends/target"
	simulation "go.temporal.io/server/tools/gomad/internal/backends/wasm/cluster"
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
