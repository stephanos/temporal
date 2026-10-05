package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/lint"
	umpiremodel "go.temporal.io/server/tools/umpire/model"
)

// fixture is a lifter fixture with machines, Properties and find Queries and no realization, small
// enough to lint in a test.
const fixture = "../../../../model/irgen/testdata/lifts/expected/declarations.json"

// answer is what one run printed and its exit status.
type answer struct {
	status      int
	out, errors string
}

func lintRun(arguments ...string) answer {
	var out, errors bytes.Buffer
	status := run(arguments, &out, &errors)
	return answer{status, out.String(), errors.String()}
}

// copied is the fixture copied into a directory of its own, so acceptance files can be written
// beside it.
func copied(t *testing.T) string {
	t.Helper()
	encoded, err := os.ReadFile(fixture)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "declarations.json")
	require.NoError(t, os.WriteFile(path, encoded, 0o644))
	return path
}

// findings is what lint finds in the IR file, read as the command reads it.
func findings(t *testing.T, path string) []lint.Finding {
	t.Helper()
	m, err := lint.Read(path, lowering, lint.Options{})
	require.NoError(t, err)
	result, err := m.Lint()
	require.NoError(t, err)
	return result.Findings()
}

// accepting is an acceptance of each finding, under one reason, and of the extra acceptances.
func accepting(found []lint.Finding, because string, extra ...lint.Acceptance) lint.Accepted {
	var a lint.Accepted
	for _, f := range found {
		if n := len(a.Accepted); n > 0 && a.Accepted[n-1].Kind == f.Kind && a.Accepted[n-1].Owner == f.Owner {
			a.Accepted[n-1].Subjects = append(a.Accepted[n-1].Subjects, f.Subject)
			continue
		}
		a.Accepted = append(a.Accepted, lint.Acceptance{Kind: f.Kind, Owner: f.Owner, Subjects: []string{f.Subject}, Because: because})
	}
	a.Accepted = append(a.Accepted, extra...)
	return a
}

func writeAccepted(t *testing.T, path string, a lint.Accepted) {
	t.Helper()
	encoded, err := json.MarshalIndent(a, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, encoded, 0o644))
}

// A finding nothing accepts fails the run and is printed; accepted with a reason, it passes and is
// printed with that reason.
func TestAFindingFailsTheRunUntilItIsAcceptedWithAReason(t *testing.T) {
	path := copied(t)
	found := findings(t, path)
	require.NotEmpty(t, found, "the fixture must give a finding, or this test checks nothing")
	unaccepted := lintRun(path)
	require.Equal(t, 1, unaccepted.status, unaccepted.errors)
	require.True(t, strings.HasPrefix(unaccepted.out, "lint "+path+"\n"), unaccepted.out)
	for _, f := range found {
		require.Contains(t, unaccepted.out, f.Message)
	}
	require.NotContains(t, unaccepted.out, "accepted: ")
	require.Contains(t, unaccepted.errors, "umpire-lint: "+strconv.Itoa(len(found))+" unaccepted findings, 0 stale acceptances and 0 errors; fix the Model")

	writeAccepted(t, lint.AcceptedPath(path), accepting(found, "the fixture declares it so"))
	accepted := lintRun(path)
	require.Equal(t, 0, accepted.status, accepted.errors)
	require.Empty(t, accepted.errors)
	require.Equal(t, len(found), strings.Count(accepted.out, "accepted: the fixture declares it so\n"), accepted.out)
}

// An acceptance that matches no finding is stale and fails the run, every finding accepted.
func TestAStaleAcceptanceFailsTheRun(t *testing.T) {
	path := copied(t)
	stale := lint.Acceptance{Kind: lint.UnaskedProperty, Owner: "nowhere", Subjects: []string{"nothing"}, Because: "kept"}
	writeAccepted(t, lint.AcceptedPath(path), accepting(findings(t, path), "the fixture declares it so", stale))
	a := lintRun(path)
	require.Equal(t, 1, a.status)
	require.Contains(t, a.out, "  stale acceptance: unasked-property nowhere \"nothing\" matches no finding\n")
	require.Contains(t, a.errors, "umpire-lint: 0 unaccepted findings, 1 stale acceptances and 0 errors;")
}

// An acceptance with no reason is an error of the acceptance file: the file's findings are not
// judged, and the run fails.
func TestAnAcceptanceWithNoReasonFailsTheRun(t *testing.T) {
	path := copied(t)
	writeAccepted(t, lint.AcceptedPath(path), accepting(nil, "", lint.Acceptance{Kind: lint.UnaskedProperty, Owner: "disk", Subjects: []string{"x"}}))
	a := lintRun(path)
	require.Equal(t, 1, a.status)
	require.Empty(t, a.out)
	require.Contains(t, a.errors, lint.AcceptedPath(path)+": acceptance 0 of unasked-property disk gives no reason\n")
	require.Contains(t, a.errors, " and 1 errors;")
}

// An acceptance file beside no IR file accepts nothing, and fails the run.
func TestAnOrphanAcceptanceFileFailsTheRun(t *testing.T) {
	path := copied(t)
	writeAccepted(t, lint.AcceptedPath(path), accepting(findings(t, path), "the fixture declares it so"))
	orphan := filepath.Join(filepath.Dir(path), "gone"+lint.AcceptedSuffix)
	writeAccepted(t, orphan, accepting(nil, "", lint.Acceptance{Kind: lint.UnaskedProperty, Owner: "gone", Subjects: []string{"x"}, Because: "kept"}))
	a := lintRun(path)
	require.Equal(t, 1, a.status)
	require.Contains(t, a.errors, orphan+" accepts findings of "+filepath.Join(filepath.Dir(path), "gone.json")+", which does not exist")
	require.Contains(t, a.errors, "0 unaccepted findings, 0 stale acceptances and 1 errors;")
}

// A malformed IR file is reported with the reader's error and gives no findings; the other files
// are linted all the same.
func TestAMalformedIRFileFailsWithTheReadersError(t *testing.T) {
	path := copied(t)
	malformed := filepath.Join(filepath.Dir(path), "malformed.json")
	require.NoError(t, os.WriteFile(malformed, []byte(`{"machines": [{"unknown": 1}]}`), 0o644))
	_, readerError := umpiremodel.Load(malformed)
	require.Error(t, readerError)

	a := lintRun(path, malformed)
	require.Equal(t, 1, a.status)
	require.Contains(t, a.errors, readerError.Error()+"\n")
	require.NotContains(t, a.out, malformed)
	require.Contains(t, a.out, "lint "+path+"\n")
}

// The output is byte-stable: the same files print the same bytes, in the same order whatever
// order they are named in, with the tables as without them.
func TestTheOutputIsByteStable(t *testing.T) {
	path := copied(t)
	other := filepath.Join(filepath.Dir(path), "presence.json")
	encoded, err := os.ReadFile(filepath.Join(filepath.Dir(fixture), "presence.json"))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(other, encoded, 0o644))

	first := lintRun(path, other)
	require.Equal(t, first, lintRun(other, path))
	require.Equal(t, first, lintRun(path, other))
	require.Less(t, strings.Index(first.out, "lint "+path+"\n"), strings.Index(first.out, "lint "+other+"\n"))
	tables := lintRun("--tables", path, other)
	require.Equal(t, tables, lintRun("--tables", path, other))
	require.Equal(t, first.status, tables.status)
}

func TestAnUnknownFlagIsBadUsage(t *testing.T) {
	a := lintRun("--verbose")
	require.Equal(t, 2, a.status)
	require.Contains(t, a.errors, "usage: umpire-lint [--update] [--must-not-pinned] [--tables] [ir files...]")
	require.Empty(t, a.out)
}

// capabilities is the lifter fixture whose capability declarations write a law sidecar, with a law
// overridden and a law excepted.
const capabilities = "../../../../model/irgen/testdata/lifts/expected/capabilities.json"

// copiedWithLaws is the capabilities fixture and its law sidecar copied into a directory of their own.
func copiedWithLaws(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "capabilities.json")
	for from, to := range map[string]string{capabilities: path, umpiremodel.LawSidecarPath(capabilities): umpiremodel.LawSidecarPath(path)} {
		encoded, err := os.ReadFile(from)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(to, encoded, 0o644))
	}
	return path
}

// The reasons of the laws a sidecar waives reach the accepted findings only by an update, which
// forwards them keyed `<machine>.<law>` beside the acceptances an author wrote; a check fails on
// accepted findings that do not carry them, and on one a waiver no longer backs.
func TestAnUpdateForwardsTheLawWaiversAndACheckHoldsThem(t *testing.T) {
	path := copiedWithLaws(t)
	var authored []lint.Finding
	for _, f := range findings(t, path) {
		if f.Kind != lint.WaivedLaw {
			authored = append(authored, f)
		}
	}
	writeAccepted(t, lint.AcceptedPath(path), accepting(authored, "the fixture declares it so"))

	check := lintRun(path)
	require.Equal(t, 1, check.status)
	require.Contains(t, check.errors, lint.AcceptedPath(path)+" does not carry the law waivers of "+umpiremodel.LawSidecarPath(path)+
		": rerun make umpire-gen-model, whose update forwards them (umpire-lint --update)\n")
	require.Contains(t, check.errors, "2 unaccepted findings, 0 stale acceptances and 1 errors;")

	update := lintRun("--update", path)
	require.Equal(t, 0, update.status, update.errors)
	require.Contains(t, update.out, "accepted: a fixture's waiver: the legacy job keeps no status\n")
	accepted, err := lint.ReadAccepted(path)
	require.NoError(t, err)
	require.Equal(t, []lint.Acceptance{
		{Kind: lint.WaivedLaw, Owner: "legacyJob", Subjects: []string{"legacyJob.closedIsRejectedUniformly"},
			Because: "a fixture's override: the legacy job answers a closed job as it likes"},
		{Kind: lint.WaivedLaw, Owner: "legacyJob", Subjects: []string{"legacyJob.terminalStatesAreFinal"},
			Because: "a fixture's waiver: the legacy job keeps no status"},
	}, accepted.Accepted[len(accepted.Accepted)-2:])
	require.Equal(t, update.out, lintRun(path).out, "a check after the update prints what the update did")
	require.Equal(t, 0, lintRun(path).status)

	// A waiver the sidecar no longer records leaves its forwarded acceptance stale.
	laws, err := umpiremodel.ReadLawSidecar(path)
	require.NoError(t, err)
	laws.Waivers = laws.Waivers[:1]
	encoded, err := json.Marshal(laws)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(umpiremodel.LawSidecarPath(path), encoded, 0o644))
	gone := lintRun(path)
	require.Equal(t, 1, gone.status)
	require.Contains(t, gone.out, "  stale acceptance: waived-law legacyJob \"legacyJob.terminalStatesAreFinal\" matches no finding\n")
	require.Contains(t, gone.errors, " does not carry the law waivers of ")
}

// A law's instantiating machines are counted across every sidecar of the directories a run lints,
// so a law with one machine in each of two files is no finding in either.
func TestInstantiatingMachinesAreCountedAcrossTheDirectory(t *testing.T) {
	path := copiedWithLaws(t)
	lonely := func(path string) []string {
		var out []string
		m, err := lint.Read(path, lowering, lint.Options{Instances: instances([]string{filepath.Dir(path)})})
		require.NoError(t, err)
		result, err := m.Lint()
		require.NoError(t, err)
		for _, f := range result.Findings() {
			if f.Kind == lint.LawWithOneInstance {
				out = append(out, f.Subject)
			}
		}
		return out
	}
	alone := lonely(path)
	require.Contains(t, alone, "terminalStatesAreFinal", "the fixture's laws have one instantiating machine each")

	laws, err := umpiremodel.ReadLawSidecar(path)
	require.NoError(t, err)
	for i := range laws.Catalog {
		laws.Catalog[i].Instantiating = []umpiremodel.LawInstance{{Machine: "elsewhere", State: "fixture.Elsewhere"}}
	}
	laws.Claims, laws.Waivers = nil, nil
	encoded, err := json.Marshal(laws)
	require.NoError(t, err)
	other := filepath.Join(filepath.Dir(path), "other.json")
	require.NoError(t, os.WriteFile(umpiremodel.LawSidecarPath(other), encoded, 0o644))
	require.NoError(t, os.WriteFile(other, []byte("{}"), 0o644))
	require.Empty(t, lonely(path))
}
