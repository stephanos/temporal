package recovery

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateUpdateAndReadBack(t *testing.T) {
	path := filepath.Join(t.TempDir(), "recovery.json")
	store, err := Create(path, "1234-1")
	require.NoError(t, err)
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, fs.FileMode(0o600), info.Mode().Perm())

	read, err := Read(path)
	require.NoError(t, err)
	require.Equal(t, Record{Version: Version, InvocationID: "1234-1", Phase: PhaseStarted, Iterations: []Iteration{}}, *read)

	require.NoError(t, store.Update(func(r *Record) {
		r.Phase = PhaseRunning
		r.Lease = &Lease{WorkflowID: "umpire-canary-lease", RunID: "fence", Held: HeldTook}
		r.CurrentRunID = "run-1"
		r.Iterations = append(r.Iterations, Iteration{RunID: "run-1"})
	}))
	info, err = os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, fs.FileMode(0o600), info.Mode().Perm(), "an update keeps the mode")
	read, err = Read(path)
	require.NoError(t, err)
	require.Equal(t, store.Snapshot(), *read)
	require.Equal(t, HeldTook, read.Lease.Held)

	snapshot := store.Snapshot()
	snapshot.Lease.RunID = "changed"
	snapshot.Iterations[0].Published = true
	require.Equal(t, "fence", store.Snapshot().Lease.RunID, "a snapshot is a copy")
	require.False(t, store.Snapshot().Iterations[0].Published)

	entries, err := os.ReadDir(filepath.Dir(path))
	require.NoError(t, err)
	require.Len(t, entries, 1, "an update leaves no temporary file")

	_, err = Create(path, "1234-1")
	require.Error(t, err, "a job writes one record, once")
}

// An update that would make the record invalid changes neither the file nor the store.
func TestAnInvalidUpdateChangesNothing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "recovery.json")
	store, err := Create(path, "1234-1")
	require.NoError(t, err)
	before, err := os.ReadFile(path)
	require.NoError(t, err)
	for name, change := range map[string]func(*Record){
		"an unknown phase":               func(r *Record) { r.Phase = "paused" },
		"a lease with no run":            func(r *Record) { r.Lease = &Lease{WorkflowID: "lease", Held: HeldTook} },
		"a lease neither took nor found": func(r *Record) { r.Lease = &Lease{WorkflowID: "lease", RunID: "run", Held: "borrowed"} },
		"an iteration with no Run":       func(r *Record) { r.Iterations = []Iteration{{}} },
		"no invocation":                  func(r *Record) { r.InvocationID = "" },
	} {
		t.Run(name, func(t *testing.T) {
			require.Error(t, store.Update(change))
			after, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, before, after)
			require.Equal(t, PhaseStarted, store.Snapshot().Phase)
		})
	}
}

// A record is refused unless it is exactly canonical, valid, regular and mode 0600.
func TestReadRefusesAnythingButTheCanonicalRecord(t *testing.T) {
	canonical, err := Encode(&Record{Version: Version, InvocationID: "1234-1", Phase: PhaseLeasing,
		Lease: &Lease{WorkflowID: "umpire-canary-lease", RunID: "fence", Held: HeldFound}})
	require.NoError(t, err)
	decoded, err := Decode(canonical)
	require.NoError(t, err)
	require.Equal(t, HeldFound, decoded.Lease.Held)

	text := string(canonical)
	for name, encoded := range map[string]string{
		"an unknown key":         strings.Replace(text, `"version": 1,`, `"version": 1,`+"\n"+`  "extra": true,`, 1),
		"a repeated key":         strings.Replace(text, `"version": 1,`, `"version": 1,`+"\n"+`  "version": 1,`, 1),
		"a case-folded key":      strings.Replace(text, `"invocationId"`, `"InvocationId"`, 1),
		"another version":        strings.Replace(text, `"version": 1`, `"version": 2`, 1),
		"another spacing":        strings.ReplaceAll(text, "  ", "    "),
		"null iterations":        strings.Replace(text, `"iterations": []`, `"iterations": null`, 1),
		"no final newline":       strings.TrimSuffix(text, "\n"),
		"a lease held otherwise": strings.Replace(text, `"found"`, `"borrowed"`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			require.NotEqual(t, text, encoded)
			_, err := Decode([]byte(encoded))
			require.Error(t, err)
		})
	}

	dir := t.TempDir()
	loose := filepath.Join(dir, "loose.json")
	require.NoError(t, os.WriteFile(loose, canonical, 0o644))
	require.NoError(t, os.Chmod(loose, 0o644))
	_, err = Read(loose)
	require.ErrorContains(t, err, "mode")

	link := filepath.Join(dir, "link.json")
	target := filepath.Join(dir, "target.json")
	require.NoError(t, os.WriteFile(target, canonical, 0o600))
	require.NoError(t, os.Symlink(target, link))
	_, err = Read(link)
	require.ErrorContains(t, err, "not a regular file")

	large := filepath.Join(dir, "large.json")
	require.NoError(t, os.WriteFile(large, []byte(strings.Repeat(" ", maxRecordBytes+1)), 0o600))
	_, err = Read(large)
	require.ErrorContains(t, err, "exceeds")

	_, err = Read(filepath.Join(dir, "absent.json"))
	require.ErrorIs(t, err, fs.ErrNotExist, "a missing record is nothing to reconcile")
}
