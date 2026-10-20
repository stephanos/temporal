package simulation

import (
	"crypto/sha256"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	model "go.temporal.io/server/tools/gomad/internal/model/cluster"
)

func TestStressRetainedReopenPreservesBoundInputsAndRestoresHostOptIn(t *testing.T) {
	for _, changeDeclared := range []bool{false, true} {
		t.Run(map[bool]string{false: "private-host-opt-in", true: "reject-declared-mutation"}[changeDeclared], func(t *testing.T) {
			module := []byte("retained-module")
			installed := installedHelperFile(t, []byte("stress-helper"))
			moduleID := strings.TrimPrefix(string(record.HashBytes(module)), "sha256:")
			config := wasi.Config{Environment: []string{"CLUSTER=active", "PHASE=reopen"}, EntropyKey: sha256.Sum256([]byte("active"))}
			request := wasi.Request{ModuleSHA256: moduleID, HelperSHA256: installed.SHA256, Engine: installed.Engine, Config: config, Fuel: 100}
			group := []OutputCrashGroup{{Handles: []model.NodeHandle{{Node: "active", Incarnation: 1}, {Node: "standby", Incarnation: 1}}, Marker: []byte("observed")}}
			provider := &Provider{Boots: map[model.BootID]wasi.Request{"reopen": request}, OutputCrashGroups: group}
			admission := &retainedAdmission{bundle: retainedExecution{Boots: []retainedBoot{{ID: "reopen"}}, ModuleSHA256: moduleID}, provider: provider}
			installed.ConfigureRequest = func(request *wasi.Request) error {
				request.SetStressHostConditions("", false)
				if changeDeclared {
					request.Fuel++
				}
				return nil
			}
			_, reopened, err := admission.reopen(retainedModuleBytes(t, module), t.TempDir(), installed)
			if changeDeclared {
				require.ErrorContains(t, err, "changed retained WASM inputs")
				return
			}
			require.NoError(t, err)
			require.Equal(t, config, reopened.Boots["reopen"].Config)
			require.Equal(t, group, reopened.OutputCrashGroups)
			require.Equal(t, wasi.PinnedEngine().Configuration+"-stress-jitter", reopened.Boots["reopen"].DiagnosticConfigurationRevision())
		})
	}
}
