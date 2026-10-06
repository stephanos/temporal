package check

// The claims capability declarations generate, lifted from
// model/irgen/testdata/lifts/Capabilities.scala: each law the catalog brings a job, a pair of
// jobs, a legacy job and a job under the fixture's own catalog, named `<machine>.<law>`, with the
// total the lifter computed for its Query.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/ir"
)

// Each generated Query is admitted with the total the lifter computed, which Go recounts, and each
// has its answer: the job keeps every law, its functional laws are found from `reach`, the legacy
// job's override and the fixture catalog's keeps-law hold, and the pair keeps the pair's law.
func TestCapabilitiesGeneratedClaims(t *testing.T) {
	m := lifted(t, "capabilities")
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
		"query job job.terminalStatesAreFinal":                Verified,
		"query job job.closedIsRejectedUniformly":             Verified,
		"query job job.pausedIsNotDispatched":                 Verified,
		"query job job.terminateSettles":                      Found,
		"query job job.cancelIsRequested":                     Found,
		"query legacyJob legacyJob.closedIsRejectedUniformly": Verified,
		"query keptJob keptJob.statusStaysClosed":             Verified,
		"query pair pair.pausedIsNotDispatched":               Verified,
		"query pair rightNeverHeld":                           Verified,
		"query job killedWhileQueued":                         Found,
		"query rogueJob rogueJob.pausedIsNotDispatched":       Counterexample,
	}, kinds(c.report))
}

// A Model that breaks a law is rejected with the law's name and the entity's bindings (fn-122 R11):
// the rogue job's poll dispatches a paused job, so its generated pausedIsNotDispatched fails, and the
// report reads the law, both capabilities and their bindings from the sidecar.
func TestCapabilitiesViolationNamesTheLawAndItsBindings(t *testing.T) {
	path := filepath.Join("..", "..", "..", "model", "irgen", "testdata", "lifts", "expected", "capabilities.json")
	c, err := checkedOnce(lifted(t, "capabilities"))
	require.NoError(t, err)
	sidecar, err := ReadLawSidecar(path)
	require.NoError(t, err)
	require.NotNil(t, sidecar)
	violations := LawViolations(c.report, sidecar)
	require.Len(t, violations, 1)
	require.True(t, strings.HasPrefix(violations[0], "rogueJob.pausedIsNotDispatched breaks the law "+
		"pausedIsNotDispatched of Pausable and Pollable (paused = fixture.capabilities.Jobs$.paused, "+
		"running = fixture.capabilities.Jobs$.running): "), violations[0])
	witness := receiptOf(t, c.report, "query rogueJob rogueJob.pausedIsNotDispatched").Witness
	require.Equal(t, []string{"pause", "poll"}, taken(witness))
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

// The standalone Nexus operation, the laws' second entity, receives them without listing them:
// its terminal statuses are final, its own reading of closed rejection (a repeated request id is
// answered OK) holds, and its functional laws are found from its start (fn-122 R6, R11).
func TestNexusOperationReceivesTheLaws(t *testing.T) {
	path := filepath.Join("..", "..", "..", "model", "ir", "nexus-standalone.json")
	m, err := ir.Load(path)
	require.NoError(t, err)
	c, err := checkedOnce(m)
	require.NoError(t, err)
	require.Equal(t, map[string]ReceiptKind{
		"query nexusOperation nexusOperation.terminalStatesAreFinal":    Verified,
		"query nexusOperation nexusOperation.closedIsRejectedUniformly": Verified,
		"query nexusOperation nexusOperation.terminateSettles":          Found,
		"query nexusOperation nexusOperation.cancelIsRequested":         Found,
	}, kinds(c.report))
	sidecar, err := ReadLawSidecar(path)
	require.NoError(t, err)
	require.NotNil(t, sidecar)
	require.Empty(t, LawViolations(c.report, sidecar))
}
