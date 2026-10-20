package backend

import (
	"crypto/sha256"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/gomad/internal/backends"
	simulation "go.temporal.io/server/tools/gomad/internal/backends/wasm/cluster"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	model "go.temporal.io/server/tools/gomad/internal/model/cluster"
)

func TestStressHostConfigurationCannotChangeDeclaredInputs(t *testing.T) {
	request := wasi.Request{Engine: wasi.PinnedEngine()}
	before, err := json.Marshal(request)
	require.NoError(t, err)
	p := &Provider{options: Options{StressExecution: &StressExecution{Padding: strings.Repeat("x", 4096), Jitter: true}}}
	require.NoError(t, p.stressRequest(&request))
	after, err := json.Marshal(request)
	require.NoError(t, err)
	require.Equal(t, before, after)
	require.Equal(t, wasi.PinnedEngine().Configuration+"-stress-jitter", request.DiagnosticConfigurationRevision())
	p.options.StressExecution.ConfigureRequest = func(r *wasi.Request) { r.Fuel++ }
	require.ErrorContains(t, p.stressRequest(&request), "changed declared")
}

func TestStressScenarioBindingsAllowOnlyAuthenticatedBootInputs(t *testing.T) {
	for _, mutation := range []string{"environment", "fuel", "topology", "missing-config", "group", "wrong-group"} {
		t.Run(mutation, func(t *testing.T) {
			options := portableProviderOptions(wasi.CooperativeProfile)
			options.Config.Clock.ReadStepNanos = 0
			base := wasi.Request{Config: options.Config, Runtime: &wasi.RuntimeControl{}}
			base.Config.Args = []string{"gomad-target"}
			base.Runtime.Choice = &backends.ChoiceRequest{}
			handle := model.NodeHandle{Node: "a", Incarnation: 1}
			scenario := simulation.OperatorScenario{Schema: simulation.OperatorSchema, Spec: model.Spec{Schema: model.SpecSchema, Backend: model.BackendWASM, Fidelity: model.FidelitySimulationModel, Limits: model.DefaultLimits(), Nodes: []model.NodeSpec{{ID: "a", Boot: "main", Address: "10.0.0.1"}}}, Boots: []simulation.OperatorBoot{{ID: "main"}}, OutputCrashes: []simulation.OutputCrash{{Handle: handle, Marker: []byte("done")}}}
			spec, provider, err := scenario.Bind(base)
			require.NoError(t, err)
			p := &Provider{options: Options{StressExecution: &StressExecution{BindScenario: func(_ wasi.Request, spec model.Spec, provider *simulation.Provider) (model.Spec, error) {
				boot := provider.Boots["main"]
				boot.Config.Environment = append(boot.Config.Environment, "CLUSTER=a")
				boot.Config.EntropyKey = sha256.Sum256([]byte("a"))
				if mutation == "fuel" {
					boot.Fuel++
				}
				provider.Boots["main"] = boot
				if mutation != "missing-config" {
					spec.Nodes[0].Config, err = simulation.BootConfiguration(boot)
					require.NoError(t, err)
				}
				if mutation == "topology" {
					spec.Nodes[0].Address = "10.0.0.2"
				}
				if mutation == "group" || mutation == "wrong-group" {
					provider.OutputCrashes = nil
					marker := []byte("done")
					if mutation == "wrong-group" {
						marker = []byte("early")
					}
					provider.OutputCrashGroups = []simulation.OutputCrashGroup{{Handles: []model.NodeHandle{handle}, Marker: marker}}
				}
				return spec, nil
			}}}}
			_, err = p.bindStressScenario(base, spec, provider)
			if mutation == "environment" || mutation == "group" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
