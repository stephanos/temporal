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
const fixture = "../../../../model/lifter/testdata/lifts/expected/declarations.json"

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
	if len(found) == 0 {
		t.Skip("no kind reports a finding of the fixture")
	}
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
	require.Contains(t, a.errors, "usage: umpire-lint [--must-not-pinned] [--tables] [ir files...]")
	require.Empty(t, a.out)
}
