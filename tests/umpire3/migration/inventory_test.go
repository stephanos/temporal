package migration

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/protocol"
	"go.temporal.io/server/tests/umpire3/scenario"
)

func TestCheckedLedgerMatchesMechanicalRootInventory(t *testing.T) {
	t.Parallel()

	ledger, err := Build("../..")
	require.NoError(t, err)
	generated, err := ledger.CanonicalJSON()
	require.NoError(t, err)
	checked, err := os.ReadFile("ledger.json")
	require.NoError(t, err)
	require.JSONEq(t, string(generated), string(checked))
	require.Len(t, ledger.Entries, 28)
	for _, entry := range ledger.Entries {
		require.NotEmpty(t, entry.ModelTarget)
		require.NotEmpty(t, entry.Properties)
		require.NotEmpty(t, entry.Entities)
		require.NotEmpty(t, entry.Actions)
		require.NotEmpty(t, entry.Relations)
		require.NotEmpty(t, entry.Evidence)
		require.NotEqual(t, "typed-sparse-intent", entry.Scenario)
		require.True(t, entry.ArtifactReplay)
		require.NotEmpty(t, entry.ExecutedContracts)
		for _, executed := range entry.ExecutedContracts {
			require.NotEmpty(t, executed.ScenarioDigest)
			require.NotEmpty(t, executed.ExperimentDigests)
			require.Equal(t, executed.ScenarioDigest, executed.Explain.ScenarioDigest)
			require.Equal(t, scenario.ExplainFormatVersion, executed.Explain.FormatVersion)
		}
	}
}

func TestCheckedLedgerClassifiesCurrentBehaviorFidelity(t *testing.T) {
	ledger, err := Build("../..")
	require.NoError(t, err)
	encoded, err := ledger.CanonicalJSON()
	require.NoError(t, err)
	var raw struct {
		FormatVersion string `json:"formatVersion"`
		Entries       []struct {
			Behavior      string `json:"behavior"`
			Fidelity      string `json:"fidelity"`
			EvidenceLevel string `json:"evidenceLevel"`
		} `json:"entries"`
	}
	require.NoError(t, json.Unmarshal(encoded, &raw))
	require.Equal(t, "umpire3/migration-ledger/v3", raw.FormatVersion)
	counts := make(map[string]int)
	for _, entry := range raw.Entries {
		require.Equal(t, "local-integration", entry.EvidenceLevel, entry.Behavior)
		require.Contains(t, []string{"exact", "semantic-equivalent"}, entry.Fidelity, entry.Behavior)
		counts[entry.Fidelity]++
	}
	require.Zero(t, counts["partial"])
	require.Equal(t, 23, counts["exact"])
	require.Equal(t, 5, counts["semantic-equivalent"])
}

func TestLedgerRejectsUnsupportedFidelityAndEvidenceClaims(t *testing.T) {
	ledger, err := Build("../..")
	require.NoError(t, err)
	tests := []struct {
		name       string
		mutate     func(*Entry)
		errorMatch string
	}{
		{
			name:       "inventory-only executable behavior",
			mutate:     func(entry *Entry) { entry.Fidelity = protocol.FidelityInventoryOnly },
			errorMatch: "inventory-only fidelity",
		},
		{
			name:       "unknown fidelity",
			mutate:     func(entry *Entry) { entry.Fidelity = "unknown" },
			errorMatch: "unknown fidelity",
		},
		{
			name:       "equivalence without live integration",
			mutate:     func(entry *Entry) { entry.EvidenceLevel = protocol.EvidenceModelProof },
			errorMatch: "requires live integration evidence",
		},
		{
			name:       "unknown evidence level",
			mutate:     func(entry *Entry) { entry.EvidenceLevel = "unknown" },
			errorMatch: "unknown evidence level",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidate := ledger
			candidate.Entries = append([]Entry(nil), ledger.Entries...)
			test.mutate(&candidate.Entries[0])
			_, encodeErr := candidate.CanonicalJSON()
			require.ErrorContains(t, encodeErr, test.errorMatch)
		})
	}
}

func TestBehaviorContractsUseGeneratedSemanticVocabulary(t *testing.T) {
	t.Parallel()

	catalog, err := protocol.DefaultCatalog()
	require.NoError(t, err)
	targets := make(map[string]struct{}, len(catalog.Targets))
	properties := make(map[string]struct{}, len(catalog.Properties))
	entities := make(map[string]struct{}, len(catalog.Entities))
	actions := make(map[string]struct{}, len(catalog.Actions))
	faults := make(map[string]struct{}, len(catalog.Faults))
	capabilities := make(map[string]struct{}, len(catalog.Capabilities))
	for _, value := range catalog.Targets {
		targets[value.Identifier] = struct{}{}
	}
	for _, value := range catalog.Properties {
		properties[value.Identifier] = struct{}{}
	}
	for _, value := range catalog.Entities {
		entities[value.Identifier] = struct{}{}
	}
	for _, value := range catalog.Actions {
		actions[value.Identifier] = struct{}{}
	}
	for _, value := range catalog.Faults {
		faults[value.Identifier] = struct{}{}
	}
	for _, value := range catalog.Capabilities {
		capabilities[string(value.Identifier)] = struct{}{}
	}
	contracts := behaviorContracts()
	require.Len(t, contracts, 28)
	for behavior, contract := range contracts {
		require.Equal(t, behavior, contract.Behavior)
		require.Contains(t, targets, string(contract.ModelTarget), behavior)
		require.Contains(t, properties, string(contract.Property), behavior)
		for _, value := range contract.Entities {
			require.Contains(t, entities, string(value), behavior)
		}
		for _, value := range contract.Actions {
			require.Contains(t, actions, string(value), behavior)
		}
		for _, value := range contract.Faults {
			require.Contains(t, faults, string(value), behavior)
		}
		for _, value := range contract.RequiredCapabilities {
			require.Contains(t, capabilities, string(value), behavior)
		}
	}
}

func TestBehaviorContractsPreserveFaultSemanticsAndScopes(t *testing.T) {
	t.Parallel()

	faultAction, exists := Contract("ProbeNexusFaultAction")
	require.True(t, exists)
	require.Equal(t, []protocol.FaultKind{"hold-release"}, faultAction.Faults)
	httpFault, exists := Contract("ProbeNexusHTTPFaultSeam")
	require.True(t, exists)
	require.Equal(t, []protocol.FaultKind{"hold-release"}, httpFault.Faults)

	authored, err := Scenario("SparseRegressionCancellationRetry", "")
	require.NoError(t, err)
	suite, err := scenario.Compile(context.Background(), authored, scenario.Limits{
		MaxPaths: 1, MaxActions: 16, MaxStates: 64, MaxMemoryBytes: 1 << 20, MaxTime: time.Second,
	})
	require.NoError(t, err)
	require.Len(t, suite.Experiments, 1)
	require.Len(t, suite.Experiments[0].Faults, 1)
	require.Equal(t, []string{"nexus"}, suite.Experiments[0].Faults[0].Scope.Services)
	require.Equal(t, []string{"/service/operation"}, suite.Experiments[0].Faults[0].Scope.Routes)
}
