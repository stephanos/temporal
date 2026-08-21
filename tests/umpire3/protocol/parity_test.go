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
	require.Equal(t, ResultClassEvidenceResolved, ledger.ResultClass)
	require.True(t, ledger.TrustBadge.valid())
	require.Equal(t, ledger.SemanticHash, ledger.SourceDigest)
	require.True(t, validHash(ledger.DependencyDigest))
	require.True(t, validHash(ledger.ArtifactDigest))
	require.Len(t, ledger.Entries, 20)

	properties := 0
	targets := 0
	equivalent := 0
	for _, entry := range ledger.Entries {
		require.Equal(t, ParityEquivalent, entry.Disposition)
		require.Equal(t, MetadataPresent, entry.EvidenceStatus)
		require.Equal(t, FidelityExact, entry.Fidelity)
		require.Equal(t, EvidenceLocalIntegration, entry.EvidenceLevel)
		for _, evidence := range []ResolvedDeclaration{
			entry.Evidence.Proof,
			entry.Evidence.Executable,
			entry.Evidence.Monitor,
			entry.Evidence.NegativeControl,
		} {
			require.NotEmpty(t, evidence.Declaration, entry.LegacyName)
			require.NotEmpty(t, evidence.Type, entry.LegacyName)
			require.True(t, validHash(evidence.TypeHash), entry.LegacyName)
			require.NotContains(t, evidence.Axioms, "sorryAx", entry.LegacyName)
		}
		equivalent++
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
	require.Equal(t, 20, equivalent)
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
	require.Equal(t, "umpire3/parity-ledger/v4", raw.FormatVersion)
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
	ledger.Entries[equivalent].Evidence.Monitor = ResolvedDeclaration{}
	require.ErrorContains(t, ledger.Validate(), "complete evidence")
}

func TestParityLedgerRejectsForgedResolvedEvidenceAndArtifact(t *testing.T) {
	ledger, err := DefaultParityLedger()
	require.NoError(t, err)
	ledger.Entries[0].Evidence.Proof.Type = "True"
	require.ErrorContains(t, ledger.Validate(), "type hash")

	ledger, err = DefaultParityLedger()
	require.NoError(t, err)
	ledger.Entries[0].Evidence.Proof.Axioms = []string{"sorryAx"}
	require.ErrorContains(t, ledger.Validate(), "invalid axiom")

	ledger, err = DefaultParityLedger()
	require.NoError(t, err)
	ledger.ArtifactDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	require.ErrorContains(t, ledger.Validate(), "artifact digest")
}

func TestParityLedgerRejectsDispositionFidelityAndEvidenceContradictions(t *testing.T) {
	ledger, err := DefaultParityLedger()
	require.NoError(t, err)
	equivalent := -1
	for index, entry := range ledger.Entries {
		if equivalent < 0 && entry.Disposition == ParityEquivalent {
			equivalent = index
		}
	}
	require.NotEqual(t, -1, equivalent)
	incomplete := equivalent
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
			mutate:     func(entry *ParityEntry) { entry.EvidenceStatus = MetadataMissing },
			errorMatch: "missing evidence metadata",
		},
		{
			name: "incomplete exact fidelity", entry: incomplete,
			mutate: func(entry *ParityEntry) {
				entry.Disposition = ParityNotYetImplemented
				entry.EvidenceStatus = MetadataMissing
				entry.Fidelity = FidelityExact
			},
			errorMatch: "incomplete parity entry",
		},
		{
			name: "incomplete complete exploration", entry: incomplete,
			mutate: func(entry *ParityEntry) {
				entry.Disposition = ParityNotYetImplemented
				entry.Fidelity = FidelityPartial
				entry.EvidenceStatus = MetadataPresent
			},
			errorMatch: "claims present evidence metadata",
		},
		{
			name: "unknown fidelity", entry: incomplete,
			mutate: func(entry *ParityEntry) {
				entry.Disposition = ParityNotYetImplemented
				entry.EvidenceStatus = MetadataMissing
				entry.Fidelity = "unknown"
			},
			errorMatch: "unknown parity fidelity",
		},
		{
			name: "unknown evidence level", entry: incomplete,
			mutate: func(entry *ParityEntry) {
				entry.Disposition = ParityNotYetImplemented
				entry.EvidenceStatus = MetadataMissing
				entry.Fidelity = FidelityPartial
				entry.EvidenceLevel = "unknown"
			},
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
