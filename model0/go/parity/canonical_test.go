package parity

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/model/go/nexuscaller"
	"go.temporal.io/server/model/go/umpire"
)

// semanticOf slices the semantic object out of a canonical-metadata string without re-encoding
// it, so the comparison is byte for byte.
func semanticOf(t *testing.T, metadata string) string {
	t.Helper()
	const prefix = `{"semantic":`
	require.True(t, strings.HasPrefix(metadata, prefix))
	depth := 0
	for i := len(prefix); i < len(metadata); i++ {
		switch metadata[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return metadata[len(prefix) : i+1]
			}
		default:
		}
	}
	t.Fatal("unterminated semantic object")
	return ""
}

func requireSameString(t *testing.T, what, want, got string) {
	t.Helper()
	if want == got {
		return
	}
	i := 0
	for i < min(len(want), len(got)) && want[i] == got[i] {
		i++
	}
	lo := max(0, i-120)
	t.Fatalf("%s differs at byte %d:\n lean: …%s\n   go: …%s", what, i,
		want[lo:min(len(want), i+120)], got[lo:min(len(got), i+120)])
}

func TestTargetFingerprint(t *testing.T) {
	encoded := leanDump(t, "canonical-target-nexusProtocol.txt")
	tb, err := nexuscaller.NexusProtocol.Table()
	require.NoError(t, err)
	requireSameString(t, "target semantic", semanticOf(t, strings.TrimSpace(string(encoded))), tb.TargetSemantic())
	require.Equal(t, "sha256:b38647500819c04cd82e179972ed23c76609d69e99a424748596295ce067af14", tb.TargetFingerprint())
}

func TestScenarioPropertyAndQueryFingerprints(t *testing.T) {
	tb, err := nexuscaller.NexusProtocol.Table()
	require.NoError(t, err)
	for _, q := range nexuscaller.FunctionalQueries {
		t.Run(q.Name, func(t *testing.T) {
			want := readLean[map[string]string](t, "canonical-"+q.Name+".json")

			scenario := q.Scenario.ScenarioSemantic(tb)
			requireSameString(t, "scenario semantic", semanticOf(t, want["scenario"]), scenario)
			require.Equal(t, want["scenarioFingerprint"], umpire.Fingerprint(scenario))

			groups, err := q.Property.Lower()
			require.NoError(t, err)
			property := tb.PropertySemantic(q.Property.PropertyID(tb), groups)
			requireSameString(t, "property semantic", semanticOf(t, want["property"]), property)
			require.Equal(t, want["propertyFingerprint"], umpire.Fingerprint(property))

			query := q.QueryCanonical(tb, umpire.Fingerprint(property))
			requireSameString(t, "query canonical", want["queryCanonical"], query)
			require.Equal(t, want["queryFingerprint"], umpire.Fingerprint(query))
		})
	}
}
