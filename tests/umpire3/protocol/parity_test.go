package protocol

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultParityLedgerDispositionsEntireUmpire2Inventory(t *testing.T) {
	ledger, err := DefaultParityLedger()
	require.NoError(t, err)
	require.Len(t, ledger.Entries, 20)

	properties := 0
	targets := 0
	equivalent := 0
	incomplete := 0
	equivalentNames := map[string]struct{}{
		"foundation-backlog-ack":                {},
		"NexusOperationClosure":                 {},
		"NexusActivityLinkConsistency":          {},
		"NexusOperationTimeoutSemantics":        {},
		"CallbackReferenceConsistency":          {},
		"CallbackResponseConsistency":           {},
		"integration-nexus-activity":            {},
		"integration-callback-nexus":            {},
		"integration-callback-workflow":         {},
		"foundation-ownership-fencing":          {},
		"SpeculativeTaskCreation":               {},
		"feature-workflow-speculative-delivery": {},
		"WorkflowTaskStarvation":                {},
		"EntityProgress":                        {},
		"foundation-delivery-safety":            {},
		"integration-workflow-delivery":         {},
	}
	for _, entry := range ledger.Entries {
		if _, equivalentEntry := equivalentNames[entry.LegacyName]; equivalentEntry {
			require.Equal(t, ParityEquivalent, entry.Disposition)
			require.Equal(t, "complete", entry.ExplorationStatus)
			require.Equal(t, FidelityExact, entry.Fidelity)
			require.Equal(t, EvidenceLocalIntegration, entry.EvidenceLevel)
			equivalent++
		} else {
			require.Equal(t, ParityNotYetImplemented, entry.Disposition, entry.LegacyName)
			require.Equal(t, "incomplete", entry.ExplorationStatus, entry.LegacyName)
			incomplete++
		}
		switch entry.Category {
		case ParityProperty:
			properties++
		case ParityTarget:
			targets++
		default:
			require.FailNow(t, "unknown parity category", entry.Category)
		}
	}
	require.Equal(t, 8, properties)
	require.Equal(t, 12, targets)
	require.Equal(t, 16, equivalent)
	require.Equal(t, 4, incomplete)
}

func TestGeneratedParityEntriesDeclareFidelityAndEvidenceLevel(t *testing.T) {
	encoded, err := os.ReadFile("generated/parity-ledger.json")
	require.NoError(t, err)
	var raw struct {
		FormatVersion string `json:"formatVersion"`
		Entries       []struct {
			LegacyName    string `json:"legacyName"`
			Fidelity      string `json:"fidelity"`
			EvidenceLevel string `json:"evidenceLevel"`
		} `json:"entries"`
	}
	require.NoError(t, json.Unmarshal(encoded, &raw))
	require.Equal(t, "umpire3/parity-ledger/v2", raw.FormatVersion)
	for _, entry := range raw.Entries {
		require.NotEmpty(t, entry.Fidelity, entry.LegacyName)
		require.NotEmpty(t, entry.EvidenceLevel, entry.LegacyName)
	}
}

func TestParityLedgerRejectsEquivalentEntryWithoutEvidence(t *testing.T) {
	ledger, err := DefaultParityLedger()
	require.NoError(t, err)
	equivalent := -1
	for index, entry := range ledger.Entries {
		if entry.Disposition == ParityEquivalent {
			equivalent = index
			break
		}
	}
	require.NotEqual(t, -1, equivalent)
	ledger.Entries[equivalent].Evidence.Monitor = ""
	require.ErrorContains(t, ledger.Validate(), "complete evidence")
}

func TestParityLedgerRejectsDispositionFidelityAndEvidenceContradictions(t *testing.T) {
	ledger, err := DefaultParityLedger()
	require.NoError(t, err)
	equivalent := -1
	incomplete := -1
	for index, entry := range ledger.Entries {
		if equivalent < 0 && entry.Disposition == ParityEquivalent {
			equivalent = index
		}
		if incomplete < 0 && entry.Disposition == ParityNotYetImplemented {
			incomplete = index
		}
	}
	require.NotEqual(t, -1, equivalent)
	require.NotEqual(t, -1, incomplete)
	tests := []struct {
		name       string
		entry      int
		mutate     func(*ParityEntry)
		errorMatch string
	}{
		{
			name: "equivalent partial fidelity", entry: equivalent,
			mutate:     func(entry *ParityEntry) { entry.Fidelity = FidelityPartial },
			errorMatch: "claims equivalence with fidelity",
		},
		{
			name: "equivalent inventory evidence", entry: equivalent,
			mutate:     func(entry *ParityEntry) { entry.EvidenceLevel = EvidenceInventory },
			errorMatch: "inventory-only evidence",
		},
		{
			name: "equivalent incomplete exploration", entry: equivalent,
			mutate:     func(entry *ParityEntry) { entry.ExplorationStatus = "incomplete" },
			errorMatch: "incomplete exploration",
		},
		{
			name: "incomplete exact fidelity", entry: incomplete,
			mutate:     func(entry *ParityEntry) { entry.Fidelity = FidelityExact },
			errorMatch: "incomplete parity entry",
		},
		{
			name: "incomplete complete exploration", entry: incomplete,
			mutate:     func(entry *ParityEntry) { entry.ExplorationStatus = "complete" },
			errorMatch: "claims complete exploration",
		},
		{
			name: "unknown fidelity", entry: incomplete,
			mutate:     func(entry *ParityEntry) { entry.Fidelity = "unknown" },
			errorMatch: "unknown parity fidelity",
		},
		{
			name: "unknown evidence level", entry: incomplete,
			mutate:     func(entry *ParityEntry) { entry.EvidenceLevel = "unknown" },
			errorMatch: "unknown parity evidence level",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidate := ledger
			candidate.Entries = append([]ParityEntry(nil), ledger.Entries...)
			test.mutate(&candidate.Entries[test.entry])
			require.ErrorContains(t, candidate.Validate(), test.errorMatch)
		})
	}
}

func TestParityLedgerRejectsUnknownSemanticIdentity(t *testing.T) {
	ledger, err := DefaultParityLedger()
	require.NoError(t, err)
	ledger.Entries[0].SemanticIdentifier = "missing-property"
	require.ErrorContains(t, ledger.Validate(), "unknown property")
}
