package corpus

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/backends/native/transcript"
	"go.temporal.io/server/tools/gomad/internal/evidence/artifact"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	"go.temporal.io/server/tools/gomad/internal/runner/internal/logstore"
)

const (
	probeA = "stdlib.os.openfile"
	probeB = "stdlib.os.file.read"
	probeX = "stdlib.os.file.close"
)

func matchingReplay(context.Context, string) (ReplayResult, error) {
	return ReplayResult{Verified: true, Match: true}, nil
}

// probedCandidate is a case of one seed whose features differ from other
// seeds only by the boundary probes it covers.
func probedCandidate(t *testing.T, seed uint64, probes ...string) Candidate {
	t.Helper()
	input, _, _ := guideArtifactInput(t, seed)
	coverage, err := transcript.SummarizeSemanticProbes(probes)
	if err != nil {
		t.Fatal(err)
	}
	return Candidate{Artifact: input, Coverage: coverage}
}

func openTest(t *testing.T, root string, identity Identity, options openOptions) *Corpus {
	t.Helper()
	if options.limits == (capacity{}) {
		options.limits = defaultCapacity
	}
	corpus, err := open(context.Background(), root, identity, corpusDescription(), corpusEvidenceProvider{}, options)
	if err != nil {
		t.Fatal(err)
	}
	return corpus
}

func closeTest(t *testing.T, corpus *Corpus) {
	t.Helper()
	if err := corpus.Close(); err != nil {
		t.Fatal(err)
	}
}

func admitTest(t *testing.T, corpus *Corpus, candidate Candidate) {
	t.Helper()
	if added, err := corpus.Admit(context.Background(), candidate, matchingReplay); err != nil || !added {
		t.Fatalf("Admit(seed %d) = %t, %v", candidate.Artifact.Manifest.Seed, added, err)
	}
}

func liveSeeds(snapshot Snapshot) []uint64 {
	seeds := make([]uint64, 0, len(snapshot.Entries))
	for _, entry := range snapshot.Entries {
		seeds = append(seeds, uint64(entry.Seed))
	}
	slices.Sort(seeds)
	return seeds
}

func caseNames(t *testing.T, root string) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(root, "cases"))
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}

func logBytes(t *testing.T, root string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, StoreDirectory, logstore.LogFile))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// A capacity of one makes each novel case evict the live one. Seed 1 is
// evicted by seed 3, which seed 2 evicts, and seed 1 then evicts seed 2:
// within a pair, a case with a probe only it covers ranks first, and equal
// ranks fall back to the lower seed.
func TestCorpusLogReplaysEvictionAndReadmission(t *testing.T) {
	root := filepath.Join(t.TempDir(), "corpus")
	first := probedCandidate(t, 1, probeA)
	identity := guideIdentity(t, first.Artifact.Manifest)
	limits := capacity{entries: 1, bytes: maximumBytes}
	corpus := openTest(t, root, identity, openOptions{limits: limits})
	for _, step := range []struct {
		candidate Candidate
		live      uint64
	}{
		{first, 1},
		{probedCandidate(t, 3, probeA, probeB), 3},
		{probedCandidate(t, 2, probeB, probeX), 2},
		{probedCandidate(t, 1, probeA), 1},
	} {
		admitTest(t, corpus, step.candidate)
		snapshot := corpus.Snapshot()
		if seeds := liveSeeds(snapshot); !reflect.DeepEqual(seeds, []uint64{step.live}) {
			t.Fatalf("live seeds after admitting seed %d = %v, want [%d]", step.candidate.Artifact.Manifest.Seed, seeds, step.live)
		}
		if cases := caseNames(t, root); len(cases) != 1 || cases[0] != path.Base(snapshot.Entries[0].Artifact) {
			t.Fatalf("cases after a committed eviction = %v, want only the live case", cases)
		}
		if targets := corpusTargets(t, root); len(targets) != 1 {
			t.Fatalf("shared targets after a committed eviction = %v, want the live case's one", targets)
		}
	}
	readmitted := corpus.Snapshot()
	if readmitted.Generation != 4 {
		t.Fatalf("generation = %d, want 4 admissions", readmitted.Generation)
	}
	if again, err := corpus.Admit(context.Background(), probedCandidate(t, 1, probeA), func(context.Context, string) (ReplayResult, error) {
		t.Fatal("a live record hash was replayed again")
		return ReplayResult{}, nil
	}); err != nil || again {
		t.Fatalf("Admit(live hash) = %t, %v, want a no-op", again, err)
	}
	history := corpus.History()
	if history.Records != 5 || history.Bytes != int64(len(logBytes(t, root))) {
		t.Fatalf("history = %#v, want a header and four admissions spanning the log", history)
	}
	read, err := ReadLog(root)
	if err != nil || read.Snapshot.SnapshotSHA256 != readmitted.SnapshotSHA256 || read.History != history || read.Report.Tail != logstore.TailNone {
		t.Fatalf("ReadLog() = %#v, %v", read, err)
	}
	closeTest(t, corpus)

	for name, limits := range map[string]capacity{"same capacity": limits, "default capacity": defaultCapacity} {
		t.Run(name, func(t *testing.T) {
			reopened := openTest(t, root, identity, openOptions{limits: limits})
			defer closeTest(t, reopened)
			if !reflect.DeepEqual(reopened.snapshot, readmitted) {
				t.Fatalf("replayed snapshot = %#v, want %#v", reopened.snapshot, readmitted)
			}
			if len(reopened.live) != 1 || reopened.History() != history {
				t.Fatalf("replayed index = %v, history = %#v", reopened.live, reopened.History())
			}
		})
	}
}

func TestCorpusLogFaultsPreservePriorSnapshot(t *testing.T) {
	for _, test := range []struct {
		name      string
		point     logstore.PointKind
		fault     error
		committed bool
	}{
		// A torn frame is truncated on reopen; the case is reclaimed.
		{name: "partial write", point: logstore.PointWrite, fault: &logstore.PartialWrite{Bytes: 24}},
		// A complete frame whose sync failed may survive; replay decides.
		{name: "sync", point: logstore.PointSync, fault: errors.New("injected sync failure"), committed: true},
		{name: "after sync", point: logstore.PointSynced, fault: errors.New("injected acknowledgment failure"), committed: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), "corpus")
			first := probedCandidate(t, 1, probeA)
			identity := guideIdentity(t, first.Artifact.Manifest)
			armed := false
			corpus := openTest(t, root, identity, openOptions{hook: func(point logstore.Point) error {
				if armed && point.Kind == test.point && point.Operation == "append" {
					return test.fault
				}
				return nil
			}})
			admitTest(t, corpus, first)
			before := corpus.Snapshot()
			armed = true
			second := probedCandidate(t, 2, probeB)
			if added, err := corpus.Admit(context.Background(), second, matchingReplay); added || !errors.Is(err, logstore.ErrPoisoned) {
				t.Fatalf("Admit(fault) = %t, %v, want a poisoned append", added, err)
			}
			if !reflect.DeepEqual(corpus.Snapshot(), before) {
				t.Fatal("a failed append advanced the live snapshot")
			}
			if cases := caseNames(t, root); len(cases) != 2 {
				t.Fatalf("cases after a failed append = %v, want the uncertain case kept", cases)
			}
			armed = false
			if added, err := corpus.Admit(context.Background(), probedCandidate(t, 3, probeX), matchingReplay); added || !errors.Is(err, logstore.ErrPoisoned) {
				t.Fatalf("Admit(after fault) = %t, %v, want the poisoned writer refused", added, err)
			}
			closeTest(t, corpus)

			reopened := openTest(t, root, identity, openOptions{})
			defer closeTest(t, reopened)
			want := []uint64{1}
			if test.committed {
				want = []uint64{1, 2}
			}
			if seeds := liveSeeds(reopened.Snapshot()); !reflect.DeepEqual(seeds, want) {
				t.Fatalf("replayed live seeds = %v, want %v", seeds, want)
			}
			if !test.committed && !reflect.DeepEqual(reopened.Snapshot(), before) {
				t.Fatal("reopen changed the prior committed snapshot")
			}
			if cases := caseNames(t, root); len(cases) != len(want) {
				t.Fatalf("cases after reopen = %v, want one per live entry", cases)
			}
		})
	}
}

func TestCorpusReclaimsUncommittedPublicationOnReopen(t *testing.T) {
	root := filepath.Join(t.TempDir(), "corpus")
	first := probedCandidate(t, 1, probeA)
	identity := guideIdentity(t, first.Artifact.Manifest)
	corpus := openTest(t, root, identity, openOptions{})
	admitTest(t, corpus, first)
	committed := corpus.Snapshot()
	orphan, _, _ := guideArtifactInput(t, 2)
	if _, err := artifact.PublishArtifact(artifact.Store{Root: corpus.casesPath(), Context: context.Background(), MaximumBytes: maximumBytes, Key: artifact.StoreKeyRecord, TargetPool: corpus.targetPool()}, orphan); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "cases", ".publish-crashed"), 0o700); err != nil {
		t.Fatal(err)
	}
	closeTest(t, corpus)

	reopened := openTest(t, root, identity, openOptions{})
	defer closeTest(t, reopened)
	if !reflect.DeepEqual(reopened.Snapshot(), committed) {
		t.Fatal("an uncommitted publication changed the live snapshot")
	}
	if cases := caseNames(t, root); len(cases) != 1 || cases[0] != path.Base(committed.Entries[0].Artifact) {
		t.Fatalf("cases after reopen = %v, want only the live case", cases)
	}
}

func TestCorpusMissingLiveCaseIsCorruptionAndPreventsCleanup(t *testing.T) {
	root := filepath.Join(t.TempDir(), "corpus")
	first := probedCandidate(t, 1, probeA)
	identity := guideIdentity(t, first.Artifact.Manifest)
	corpus := openTest(t, root, identity, openOptions{})
	admitTest(t, corpus, first)
	live := filepath.Join(root, filepath.FromSlash(corpus.Snapshot().Entries[0].Artifact))
	orphan, _, _ := guideArtifactInput(t, 2)
	published, err := artifact.PublishArtifact(artifact.Store{Root: corpus.casesPath(), Context: context.Background(), MaximumBytes: maximumBytes, Key: artifact.StoreKeyRecord, TargetPool: corpus.targetPool()}, orphan)
	if err != nil {
		t.Fatal(err)
	}
	closeTest(t, corpus)
	if err := os.RemoveAll(live); err != nil {
		t.Fatal(err)
	}
	before := logBytes(t, root)
	if _, err := Open(context.Background(), root, identity, corpusDescription(), corpusEvidenceProvider{}); !errors.Is(err, artifact.ErrCorruptReference) {
		t.Fatalf("Open(missing live case) = %v, want a corrupt live reference", err)
	}
	if _, err := os.Stat(published.Path); err != nil {
		t.Fatalf("corruption reclaimed an unreferenced case: %v", err)
	}
	if !bytes.Equal(logBytes(t, root), before) {
		t.Fatal("corruption changed the log")
	}
}

type rawRecord struct {
	kind    logstore.Kind
	payload []byte
}

func jsonRecord(t *testing.T, kind logstore.Kind, value any) rawRecord {
	t.Helper()
	payload, err := logstore.EncodeJSON(value)
	if err != nil {
		t.Fatal(err)
	}
	return rawRecord{kind: kind, payload: payload}
}

func headerRecord(t *testing.T, identity Identity) rawRecord {
	return jsonRecord(t, kindHeader, logHeader{Format: LogFormat, Identity: identity})
}

// writeLog writes records to the log store of root, accepting one kind the
// corpus does not know.
func writeLog(t *testing.T, root string, records ...rawRecord) {
	t.Helper()
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	kinds := map[logstore.Kind]uint32{kindHeader: maximumRecordBytes, kindAdmission: maximumRecordBytes, 3: 64}
	writer, _, err := logstore.OpenWriter(filepath.Join(root, StoreDirectory), logstore.Options{Kinds: kinds, Create: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
	}()
	for _, raw := range records {
		if err := writer.Append(raw.kind, raw.payload); err != nil {
			t.Fatal(err)
		}
	}
}

// admissionAfter is the record admitting entry into live, evicting evicted,
// with the snapshot identity of the result.
func admissionAfter(t *testing.T, identity Identity, generation uint64, live []Entry, entry Entry, evicted ...Entry) admissionRecord {
	t.Helper()
	next := []Entry{entry}
	for _, existing := range live {
		if !containsRecordHash(evicted, existing.RecordHash) {
			next = append(next, existing)
		}
	}
	snapshot, _, err := finalizeSnapshot(Snapshot{Schema: CorpusSchema, Identity: identity, Generation: record.Uint64String(generation), Entries: next})
	if err != nil {
		t.Fatal(err)
	}
	hashes := []record.SHA256{}
	for _, retired := range evicted {
		hashes = append(hashes, retired.RecordHash)
	}
	return admissionRecord{Generation: record.Uint64String(generation), Entry: entry, Evicted: hashes, CoverageSHA256: snapshot.CoverageSHA256, SnapshotSHA256: snapshot.SnapshotSHA256}
}

type corpusPathState struct {
	mode     fs.FileMode
	size     int64
	contents string
}

func corpusStorageTree(t *testing.T, root string) map[string]corpusPathState {
	t.Helper()
	state := make(map[string]corpusPathState)
	if err := filepath.WalkDir(root, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		stored := corpusPathState{mode: info.Mode(), size: info.Size()}
		if info.Mode().IsRegular() {
			contents, err := os.ReadFile(current)
			if err != nil {
				return err
			}
			stored.contents = string(contents)
		}
		relative, err := filepath.Rel(root, current)
		if err != nil {
			return err
		}
		state[relative] = stored
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return state
}

// expectRefusedBeforeMutation opens root and checks that the refusal left
// every stored object's contents and metadata unchanged.
func expectRefusedBeforeMutation(t *testing.T, root string, identity Identity, limits capacity) error {
	t.Helper()
	before := corpusStorageTree(t, root)
	corpus, err := open(context.Background(), root, identity, corpusDescription(), corpusEvidenceProvider{}, openOptions{limits: limits})
	if err == nil {
		closeTest(t, corpus)
		t.Fatal("Open accepted the corpus")
	}
	after := corpusStorageTree(t, root)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("a refused open changed corpus storage:\nbefore: %#v\nafter:  %#v", before, after)
	}
	return err
}

func TestCorpusRejectsLegacyAndUnknownLineageBeforeMutation(t *testing.T) {
	input, _, _ := guideArtifactInput(t, 7)
	identity := guideIdentity(t, input.Manifest)
	other := identity
	other.TargetSHA256 = record.HashBytes([]byte("another target"))
	current := headerRecord(t, identity)
	for _, test := range []struct {
		name     string
		legacy   string
		records  []rawRecord
		format   string
		want     error
		rootMode fs.FileMode
	}{
		{name: "corpus.json snapshot", legacy: "corpus.json", format: LegacySnapshotFormat, want: ErrUnsupportedFormat},
		{name: "corpus.lock", legacy: "corpus.lock", format: LegacySnapshotFormat, want: ErrUnsupportedFormat},
		{name: "missing header", records: []rawRecord{}, want: ErrUnsupportedFormat},
		{name: "future format before identity", records: []rawRecord{jsonRecord(t, kindHeader, map[string]any{"format": "gomad3.guide-corpus-log/v999", "identity": other, "future": true})}, format: "gomad3.guide-corpus-log/v999", want: ErrUnsupportedFormat, rootMode: 0o750},
		{name: "missing format", records: []rawRecord{jsonRecord(t, kindHeader, map[string]any{"identity": identity})}, want: ErrUnsupportedFormat},
		{name: "malformed header", records: []rawRecord{{kind: kindHeader, payload: []byte(`{"format":`)}}, want: ErrUnsupportedFormat, rootMode: 0o750},
		{name: "unknown header field", records: []rawRecord{{kind: kindHeader, payload: append(bytes.TrimSuffix(current.payload, []byte("}")), []byte(`,"future":true}`)...)}}, want: ErrUnsupportedFormat},
		{name: "trailing header data", records: []rawRecord{{kind: kindHeader, payload: append(append([]byte(nil), current.payload...), []byte(` {}`)...)}}, want: ErrUnsupportedFormat},
		{name: "admission before header", records: []rawRecord{{kind: kindAdmission, payload: []byte(`{}`)}}, want: ErrUnsupportedFormat},
		{name: "repeated header", records: []rawRecord{current, current}, want: logstore.ErrCorrupt},
		{name: "unknown kind", records: []rawRecord{current, {kind: 3, payload: []byte(`{}`)}}, want: logstore.ErrCorrupt},
		{name: "identity", records: []rawRecord{headerRecord(t, other)}, want: errIdentityMismatch},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), "corpus")
			if test.legacy != "" {
				if err := os.MkdirAll(root, 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(root, test.legacy), []byte(`{"schema":"gomad3.guide-corpus/v2"}`), 0o600); err != nil {
					t.Fatal(err)
				}
			} else {
				writeLog(t, root, test.records...)
			}
			if test.rootMode != 0 {
				if err := os.Chmod(root, test.rootMode); err != nil {
					t.Fatal(err)
				}
			}
			err := expectRefusedBeforeMutation(t, root, identity, defaultCapacity)
			if !errors.Is(err, test.want) {
				t.Fatalf("Open() = %v, want %v", err, test.want)
			}
			var unsupported *UnsupportedFormatError
			if errors.As(err, &unsupported) && test.format != "" && unsupported.Format != test.format {
				t.Fatalf("unsupported format = %q, want %q", unsupported.Format, test.format)
			}
			if test.legacy != "" {
				if _, statErr := os.Stat(filepath.Join(root, StoreDirectory)); !errors.Is(statErr, os.ErrNotExist) {
					t.Fatalf("a legacy corpus gained a log store: %v", statErr)
				}
				if _, readErr := ReadLog(root); !errors.Is(readErr, ErrUnsupportedFormat) {
					t.Fatalf("ReadLog(legacy) = %v", readErr)
				}
				// Inspection keys on the lock alone: a bare corpus.json may be
				// an unrelated qualification manifest.
				if lockErr := RefuseLegacyLock(root); (test.legacy == "corpus.lock") != errors.Is(lockErr, ErrUnsupportedFormat) {
					t.Fatalf("RefuseLegacyLock(%s) = %v", test.legacy, lockErr)
				}
			}
		})
	}
}

func TestCorpusReplayRejectsInvalidAdmissionsBeforeApplying(t *testing.T) {
	source := filepath.Join(t.TempDir(), "source")
	first := probedCandidate(t, 1, probeA)
	identity := guideIdentity(t, first.Artifact.Manifest)
	corpus := openTest(t, source, identity, openOptions{})
	admitTest(t, corpus, first)
	admitTest(t, corpus, probedCandidate(t, 2, probeA, probeB))
	entries := corpus.Snapshot().Entries
	closeTest(t, corpus)
	a, b := entries[0], entries[1]
	diverged := a
	diverged.Replay.Match = false
	mismatched := admissionAfter(t, identity, 1, nil, a)
	mismatched.SnapshotSHA256 = record.HashBytes([]byte("another snapshot"))
	unknownField := jsonRecord(t, kindAdmission, admissionAfter(t, identity, 1, nil, a))
	unknownField.payload = append(bytes.TrimSuffix(unknownField.payload, []byte("}")), []byte(`,"future":true}`)...)
	header := headerRecord(t, identity)
	admit := func(value admissionRecord) rawRecord { return jsonRecord(t, kindAdmission, value) }
	for _, test := range []struct {
		name    string
		limits  capacity
		records []rawRecord
	}{
		{name: "generation gap", records: []rawRecord{header, admit(admissionAfter(t, identity, 2, nil, a))}},
		{name: "repeated live hash", records: []rawRecord{header, admit(admissionAfter(t, identity, 1, nil, a)), admit(admissionAfter(t, identity, 2, []Entry{a}, a))}},
		{name: "evicts a record that is not live", records: []rawRecord{header, admit(admissionAfter(t, identity, 1, nil, a, b))}},
		{name: "evicts twice", records: []rawRecord{header, admit(admissionAfter(t, identity, 1, nil, a)), admit(admissionAfter(t, identity, 2, []Entry{a}, b, a, a))}},
		{name: "entry capacity", limits: capacity{entries: 1, bytes: maximumBytes}, records: []rawRecord{header, admit(admissionAfter(t, identity, 1, nil, a)), admit(admissionAfter(t, identity, 2, []Entry{a}, b))}},
		{name: "unverified entry", records: []rawRecord{header, admit(admissionAfter(t, identity, 1, nil, diverged))}},
		{name: "unknown field", records: []rawRecord{header, unknownField}},
		{name: "trailing data", records: []rawRecord{header, {kind: kindAdmission, payload: append(admit(admissionAfter(t, identity, 1, nil, a)).payload, []byte(` {}`)...)}}},
		{name: "snapshot identity", records: []rawRecord{header, admit(mismatched)}},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), "corpus")
			writeLog(t, root, test.records...)
			if test.name == "snapshot identity" {
				replayed, err := ReadLog(root)
				if !errors.Is(err, logstore.ErrCorrupt) || replayed.Report.Records != 1 || replayed.Report.ValidBytes != int64(len(header.payload)+20) {
					t.Fatalf("ReadLog(snapshot identity) = %#v, %v", replayed.Report, err)
				}
			}
			limits := test.limits
			if limits == (capacity{}) {
				limits = defaultCapacity
			}
			if err := expectRefusedBeforeMutation(t, root, identity, limits); !errors.Is(err, logstore.ErrCorrupt) {
				t.Fatalf("Open() = %v, want corruption", err)
			}
		})
	}
}

func TestCorpusTornTailIsReportedReadOnlyAndRepairedByOpen(t *testing.T) {
	root := filepath.Join(t.TempDir(), "corpus")
	first := probedCandidate(t, 1, probeA)
	identity := guideIdentity(t, first.Artifact.Manifest)
	corpus := openTest(t, root, identity, openOptions{})
	admitTest(t, corpus, first)
	committed, history := corpus.Snapshot(), corpus.History()
	closeTest(t, corpus)
	logPath := filepath.Join(root, StoreDirectory, logstore.LogFile)
	torn := append(logBytes(t, root), "GMLG"...)
	if err := os.WriteFile(logPath, torn, 0o600); err != nil {
		t.Fatal(err)
	}
	read, err := ReadLog(root)
	if err != nil || read.Report.Tail != logstore.TailShortFrame || read.History != history || read.Snapshot.SnapshotSHA256 != committed.SnapshotSHA256 {
		t.Fatalf("ReadLog(torn) = %#v, %v", read, err)
	}
	if !bytes.Equal(logBytes(t, root), torn) {
		t.Fatal("read-only replay repaired the torn tail")
	}
	reopened := openTest(t, root, identity, openOptions{})
	defer closeTest(t, reopened)
	if !reflect.DeepEqual(reopened.Snapshot(), committed) || reopened.History() != history || int64(len(logBytes(t, root))) != history.Bytes {
		t.Fatalf("reopened after a torn tail = %#v, %#v", reopened.Snapshot(), reopened.History())
	}
}

// Read-only replay refuses a corpus root that is not a directory before it
// reads anything below it.
func TestReadLogRefusesARootThatIsNotADirectory(t *testing.T) {
	parent := t.TempDir()
	directory := filepath.Join(parent, "corpus")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(parent, "link")
	if err := os.Symlink(directory, link); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(parent, "file")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, root := range []string{link, file} {
		if _, err := ReadLog(root); err == nil || !strings.Contains(err.Error(), "private directory") {
			t.Fatalf("ReadLog(%s) = %v", root, err)
		}
	}
	if _, err := ReadLog(filepath.Join(parent, "missing")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("ReadLog(missing) = %v", err)
	}
}
