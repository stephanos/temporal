//go:build gomadv3_integration

package gomadv3integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/gomadv3sim"
)

func TestSimulationParityContractMatchesHarness(t *testing.T) {
	contents, err := os.ReadFile(filepath.Join("..", "gomadv3", "simulation", "parity", "manifest.json"))
	require.NoError(t, err)
	var manifest struct {
		Schema        string            `json:"schema"`
		HarnessSchema string            `json:"harness_schema"`
		Limits        gomadv3sim.Limits `json:"limits"`
		Cases         []struct {
			ID           string `json:"id"`
			Requirements []struct {
				Backends []string `json:"backends"`
				Fidelity string   `json:"fidelity"`
			} `json:"requirements"`
		} `json:"cases"`
		Prototypes []struct {
			Backend  string `json:"backend"`
			CaseID   string `json:"case_id"`
			Fidelity string `json:"fidelity"`
			ID       string `json:"id"`
			Package  string `json:"package"`
			Status   string `json:"status"`
			Test     string `json:"test"`
		} `json:"prototypes"`
	}
	require.NoError(t, json.Unmarshal(contents, &manifest))
	require.Equal(t, "gomadv3.simulation-parity/v1", manifest.Schema)
	require.Equal(t, gomadv3sim.SpecSchema, manifest.HarnessSchema)
	require.Equal(t, gomadv3sim.DefaultLimits(), manifest.Limits)
	caseIDs := make([]string, 0, len(manifest.Cases))
	cases := make(map[string][]struct {
		Backends []string `json:"backends"`
		Fidelity string   `json:"fidelity"`
	}, len(manifest.Cases))
	for _, parityCase := range manifest.Cases {
		caseIDs = append(caseIDs, parityCase.ID)
		cases[parityCase.ID] = parityCase.Requirements
	}
	require.Len(t, caseIDs, 13)
	require.True(t, slices.IsSorted(caseIDs))
	require.Equal(t, []struct {
		Backend  string `json:"backend"`
		CaseID   string `json:"case_id"`
		Fidelity string `json:"fidelity"`
		ID       string `json:"id"`
		Package  string `json:"package"`
		Status   string `json:"status"`
		Test     string `json:"test"`
	}{
		{Backend: "in_process", CaseID: "different-seed-diversity", Fidelity: "simulation_model", ID: "different-seed-diversity", Package: "./tools/gomadv3sim", Status: "prototype", Test: "TestScenarioCompositionSameSeedEqualityAndDifferentSeedDiversity"},
		{Backend: "in_process", CaseID: "enumerate-crash-states", Fidelity: "simulation_model", ID: "enumerate-crash-states", Package: "./tools/gomadv3sim", Status: "prototype", Test: "TestVolumeCrashEnumerationRestartAndExactReplay"},
		{Backend: "in_process", CaseID: "file-directory-sync", Fidelity: "simulation_model", ID: "file-directory-sync", Package: "./tools/gomadv3sim", Status: "prototype", Test: "TestVolumeFileAndDirectorySyncParity"},
		{Backend: "in_process", CaseID: "fixed-link-latency", Fidelity: "simulation_model", ID: "fixed-link-latency", Package: "./tools/gomadv3sim", Status: "prototype", Test: "TestNetworkFixedDelayIsRecordedAndExactlyReplayable"},
		{Backend: "in_process", CaseID: "graceful-stop-vs-crash-connection", Fidelity: "simulation_model", ID: "graceful-stop-vs-crash-connection", Package: "./tools/gomadv3sim", Status: "prototype", Test: "TestNetworkGracefulStopReturnsEOFAndCrashResetsConnection"},
		{Backend: "in_process", CaseID: "independent-node-identity-lifecycle", Fidelity: "simulation_model", ID: "independent-node-identity-lifecycle", Package: "./tools/gomadv3sim", Status: "prototype", Test: "TestRunProducesExactReplayableLifecycleRecord"},
		{Backend: "in_process", CaseID: "nemesis-partition-restart", Fidelity: "simulation_model", ID: "nemesis-partition-restart", Package: "./tools/gomadv3sim", Status: "prototype", Test: "TestFaultPlanPartitionRestartEvidenceAndExactReplay"},
		{Backend: "in_process", CaseID: "partial-crash-persistence", Fidelity: "simulation_model", ID: "partial-crash-persistence", Package: "./tools/gomadv3sim", Status: "prototype", Test: "TestVolumeCrashEnumerationRestartAndExactReplay"},
		{Backend: "in_process", CaseID: "partition-timeout-heal-reconnect", Fidelity: "simulation_model", ID: "partition-timeout-heal-reconnect", Package: "./tools/gomadv3sim", Status: "prototype", Test: "TestNetworkPartitionTimeoutHealReconnect"},
		{Backend: "process", CaseID: "two-node-request-response", Fidelity: "simulation_model", ID: "process-shared-network", Package: "./tools/gomadv3sim", Status: "prototype", Test: "TestProcessBackendRoutesTCPThroughSharedHostModel"},
		{Backend: "process", CaseID: "restart-durable-and-volatile", Fidelity: "simulation_model", ID: "process-volume-restart", Package: "./tools/gomadv3sim", Status: "prototype", Test: "TestProcessBackendPreservesHostVolumeAcrossRestart"},
		{Backend: "in_process", CaseID: "rename-truncate-crash-dependencies", Fidelity: "simulation_model", ID: "rename-truncate-crash-dependencies", Package: "./tools/gomadv3sim", Status: "prototype", Test: "TestVolumeRenameAndTruncateCrashDependenciesParity"},
		{Backend: "in_process", CaseID: "restart-durable-and-volatile", Fidelity: "simulation_model", ID: "restart", Package: "./tools/gomadv3sim", Status: "prototype", Test: "TestPrototypeRestart"},
		{Backend: "process", CaseID: "restart-durable-and-volatile", Fidelity: "hard_isolation", ID: "restart-hard-isolation", Package: "./tools/gomadv3sim", Status: "prototype", Test: "TestProcessBackendResetsGlobalsDescriptorsAndGoroutines"},
		{Backend: "in_process", CaseID: "same-seed-equality", Fidelity: "simulation_model", ID: "same-seed-equality", Package: "./tools/gomadv3sim", Status: "prototype", Test: "TestScenarioCompositionSameSeedEqualityAndDifferentSeedDiversity"},
		{Backend: "in_process", CaseID: "two-node-request-response", Fidelity: "simulation_model", ID: "two-node-request-response", Package: "./tools/gomadv3sim", Status: "prototype", Test: "TestPrototypeTwoNodeRequestResponse"},
	}, manifest.Prototypes)
	for _, prototype := range manifest.Prototypes {
		matched := false
		for _, requirement := range cases[prototype.CaseID] {
			if requirement.Fidelity == prototype.Fidelity && slices.Contains(requirement.Backends, prototype.Backend) {
				matched = true
				break
			}
		}
		require.True(t, matched, "prototype %q widens case %q", prototype.ID, prototype.CaseID)
	}
}
