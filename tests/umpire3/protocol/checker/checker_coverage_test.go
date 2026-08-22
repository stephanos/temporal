package checker

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultCheckerCoverageDispositionsEveryPrimaryChecker(t *testing.T) {
	manifest, err := DefaultCheckerCoverage()
	require.NoError(t, err)
	composition, err := DefaultComposition()
	require.NoError(t, err)

	pairs := 0
	for _, target := range composition.Targets {
		pairs += len(target.Properties)
	}
	require.Len(t, manifest.Entries, pairs*3)

	checked := map[CheckerKind]int{}
	unsupported := map[CheckerKind]int{}
	for _, entry := range manifest.Entries {
		switch entry.Status {
		case CheckerCoverageChecked:
			checked[entry.Checker]++
		case CheckerCoverageNotSupported:
			unsupported[entry.Checker]++
		default:
			require.Failf(t, "unknown checker status", "%q", entry.Status)
		}
	}
	require.Equal(t, pairs, checked[CheckerExact])
	require.Equal(t, 1, checked[CheckerNative])
	require.Equal(t, 1, checked[CheckerVeil])
	require.Equal(t, pairs-1, unsupported[CheckerNative])
	require.Equal(t, pairs-1, unsupported[CheckerVeil])
}

func TestCheckerCoverageRejectsUnearnedClaims(t *testing.T) {
	manifest, err := DefaultCheckerCoverage()
	require.NoError(t, err)

	tests := map[string]func(*CheckerCoverageManifest){
		"unsupported exact": func(value *CheckerCoverageManifest) {
			entry := checkerCoverageEntry(t, value, CheckerExact, TargetIDFoundationBacklogAck)
			entry.Status = CheckerCoverageNotSupported
			entry.Reason = "not implemented"
			entry.SemanticHash = ""
			entry.Claims = nil
			entry.Evidence = nil
		},
		"checked without evidence": func(value *CheckerCoverageManifest) {
			entry := checkerCoverageEntry(t, value, CheckerNative, TargetIDNexusCancellation)
			entry.Evidence = nil
		},
		"unsupported with claims": func(value *CheckerCoverageManifest) {
			entry := checkerCoverageEntry(t, value, CheckerNative, TargetIDFoundationBacklogAck)
			entry.Claims = []CheckerClaim{{
				Job: "exhaustive", ResultClass: ResultClassFiniteExhaustive,
				TrustBadge: TrustBadgeCheckedCertificate, Exact: true,
			}}
		},
		"missing target checker": func(value *CheckerCoverageManifest) {
			value.Entries = value.Entries[1:]
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			changed := manifest.Clone()
			mutate(&changed)
			require.Error(t, changed.Validate())
		})
	}
}

func checkerCoverageEntry(
	t *testing.T,
	manifest *CheckerCoverageManifest,
	checker CheckerKind,
	target TargetID,
) *CheckerCoverageEntry {
	t.Helper()
	for index := range manifest.Entries {
		entry := &manifest.Entries[index]
		if entry.Checker == checker && entry.Target == target {
			return entry
		}
	}
	require.FailNowf(t, "missing checker coverage entry", "%s/%s", checker, target)
	return nil
}
