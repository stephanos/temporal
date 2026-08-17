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
		{Backend: "in_process", CaseID: "restart-durable-and-volatile", Fidelity: "simulation_model", ID: "restart", Package: "./tools/gomadv3sim", Status: "prototype", Test: "TestPrototypeRestart"},
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
