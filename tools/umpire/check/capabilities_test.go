package check

// Claims brought by capability companions, with their query bounds and inert origins.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/ir"
)

// Generated Queries use the bounds their queries sections state, including a per-Property
// override. Shared sections keep their waivers, and the paired Property needs both capabilities.
func TestCapabilitiesGeneratedClaims(t *testing.T) {
	m := lifted(t, "capabilitySections")
	recounted, err := ir.WithTotals(m)
	require.NoError(t, err)
	for i, q := range m.GetQueries() {
		require.NotNil(t, q.GetTotal(), q.GetName())
		require.Equal(t, recounted.GetQueries()[i].GetTotal().GetValue(), q.GetTotal().GetValue(), q.GetName())
	}
	c, err := checkedOnce(m)
	require.NoError(t, err)
	require.Empty(t, c.report.Unsupported())
	require.Equal(t, map[string]ReceiptKind{
		"query chore chore.sealedIsRefused":             Verified,
		"query task heldStaysUntaken":                   Verified,
		"query mirrorPair mirrorPair.sealedStaysSealed": Verified,
		"query task task.heldIsNotTaken":                Verified,
		"query task task.sealedIsRefused":               Verified,
		"query task task.sealedStaysSealed":             Verified,
		"query taskPair taskPair.heldIsNotTaken":        Verified,
		"query taskPair taskPair.sealedStaysSealed":     Verified,
	}, kinds(c.report))
}

func TestCapabilitiesGeneratedPropertiesKeepTheirOrigin(t *testing.T) {
	m := lifted(t, "capabilitySections")
	origins := map[string]string{}
	for _, property := range m.GetProperties() {
		if property.GetOrigin() != nil {
			origins[property.GetName()] = property.GetOrigin().GetName()
			require.NotEmpty(t, property.GetOrigin().GetPosition().GetFile(), property.GetName())
		}
	}
	require.Equal(t, map[string]string{
		"chore.sealedIsRefused":        "fixture.capabilitysections.Sealable.sealedIsRefused",
		"mirrorPair.sealedStaysSealed": "fixture.capabilitysections.Sealable.sealedStaysSealed",
		"task.heldIsNotTaken":          "fixture.capabilitysections.Holdable.heldIsNotTaken",
		"task.sealedIsRefused":         "fixture.capabilitysections.Sealable.sealedIsRefused",
		"task.sealedStaysSealed":       "fixture.capabilitysections.Sealable.sealedStaysSealed",
		"taskPair.heldIsNotTaken":      "fixture.capabilitysections.Holdable.heldIsNotTaken",
		"taskPair.sealedStaysSealed":   "fixture.capabilitysections.Sealable.sealedStaysSealed",
	}, origins)
}

// The law sidecar the lifter writes beside an IR file, and the accepted lint findings an author
// writes there, are JSON and no Model: every reader of a directory of IR files, the Case generator,
// the exploration bridge and lint among them, lists them apart.
func TestIRPathsLeaveOutLawSidecarsAndAcceptedFindings(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"activity-standalone.json", "activity-standalone.laws.json", "activity-standalone.lint.json", "nexus.json", "notes.txt"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("{}"), 0o600))
	}
	paths, err := ir.IRPaths(dir)
	require.NoError(t, err)
	require.Equal(t, []string{filepath.Join(dir, "activity-standalone.json"), filepath.Join(dir, "nexus.json")}, paths)
}

// The standalone Nexus operation receives its companions' Properties without listing them:
// its terminal statuses are final, its own reading of closed rejection (a repeated request id is
// answered OK) holds, and its functional laws are found from its start (fn-122 R6, R11).
func TestNexusOperationReceivesTheLaws(t *testing.T) {
	path := filepath.Join("..", "..", "..", "model", "ir", "nexus-standalone.json")
	m, err := ir.Load(path)
	require.NoError(t, err)
	c, err := checkedOnce(m)
	require.NoError(t, err)
	require.Equal(t, map[string]ReceiptKind{
		"query nexusSystem nexusSystem.terminalStatesAreFinal":    Verified,
		"query nexusSystem terminalHolds":                         Verified,
		"refinement nexusSystem nexusProduct":                     Verified,
		"query nexusSystem nexusSystem.closedIsRejectedUniformly": Verified,
		"query nexusSystem nexusSystem.terminateSettles":          Found,
		"query nexusSystem nexusSystem.cancelIsRequested":         Found,
	}, kinds(c.report))
	origins := map[string]string{}
	for _, property := range m.GetProperties() {
		if property.GetOrigin() != nil {
			origins[property.GetName()] = property.GetOrigin().GetName()
			require.NotEmpty(t, property.GetOrigin().GetPosition().GetFile(), property.GetName())
		}
	}
	require.Equal(t, map[string]string{
		"nexusSystem.terminalStatesAreFinal":    "temporal.capabilities.Closable.terminalStatesAreFinal",
		"nexusSystem.closedIsRejectedUniformly": "temporal.capabilities.Closable.closedIsRejectedUniformly",
		"nexusSystem.terminateSettles":          "temporal.capabilities.Terminable.terminateSettles",
		"nexusSystem.cancelIsRequested":         "temporal.capabilities.Cancelable.cancelIsRequested",
	}, origins)
}
