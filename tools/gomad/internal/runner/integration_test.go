package runner

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"hash/crc32"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	backend "go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/evidence/artifact"
	captured "go.temporal.io/server/tools/gomad/internal/evidence/artifact/capture"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
	"go.temporal.io/server/tools/gomad/internal/runner/inspect"
	"go.temporal.io/server/tools/gomad/internal/runner/internal/campaign"
	"go.temporal.io/server/tools/gomad/internal/runner/internal/corpus"
	"go.temporal.io/server/tools/gomad/internal/runner/internal/logstore"
	campaignoptions "go.temporal.io/server/tools/gomad/internal/runner/options"
)

func transcriptSHA256(value [sha256.Size]byte) record.SHA256 {
	return record.SHA256("sha256:" + hex.EncodeToString(value[:]))
}

func pointerToReadOnlyMounts(mounts record.ReadOnlyMounts) *record.ReadOnlyMounts {
	return &mounts
}

func replayRecordedCapturedInputs(manifest captured.CapturedInputsManifest) record.ReadOnlyMounts {
	return record.ReadOnlyMounts{
		Schema: manifest.Schema, File: manifest.File, SHA256: record.SHA256(manifest.SHA256), Bytes: record.Uint64String(manifest.Bytes),
		Entries: record.Uint64String(manifest.Entries), NotExist: record.Uint64String(manifest.NotExist), TotalBytes: record.Uint64String(manifest.TotalBytes), Mappings: append([]string(nil), manifest.Mappings...),
		Limits: record.ReadOnlyMountLimits{
			PathBytes: record.Uint64String(manifest.Limits.PathBytes), Requests: record.Uint64String(manifest.Limits.Requests), Files: record.Uint64String(manifest.Limits.Files),
			DirectoryEntries: record.Uint64String(manifest.Limits.DirectoryEntries), SingleFileBytes: record.Uint64String(manifest.Limits.SingleFileBytes), TotalBytes: record.Uint64String(manifest.Limits.TotalBytes),
		},
	}
}

// inspectLogFrame encodes one log record as the log store frames it.
func inspectLogFrame(kind uint16, payload []byte) []byte {
	table := crc32.MakeTable(crc32.Castagnoli)
	frame := make([]byte, 16+len(payload)+4)
	copy(frame, "GMLG")
	frame[4] = logstore.FrameVersion
	binary.LittleEndian.PutUint16(frame[6:], kind)
	binary.LittleEndian.PutUint32(frame[8:], uint32(len(payload)))
	binary.LittleEndian.PutUint32(frame[12:], crc32.Checksum(frame[:12], table))
	copy(frame[16:], payload)
	binary.LittleEndian.PutUint32(frame[16+len(payload):], crc32.Update(crc32.Checksum(frame[:16], table), table, payload))
	return frame
}

// writeInspectStore writes a private directory holding a log store whose log
// is exactly log.
func writeInspectStore(t *testing.T, log []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "store-owner")
	store := filepath.Join(path, "store")
	if err := os.MkdirAll(store, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(store, logstore.LogFile), log, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(store, logstore.LockFile), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func readInspectLog(t *testing.T, path string) []byte {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join(path, "store", logstore.LogFile))
	if err != nil {
		t.Fatal(err)
	}
	return contents
}

func appendInspectLog(t *testing.T, path string, data []byte) {
	t.Helper()
	file, err := os.OpenFile(filepath.Join(path, "store", logstore.LogFile), os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write(data); err != nil {
		t.Fatal(errors.Join(err, file.Close()))
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

// rewriteInspectEnvelope changes the first record without disturbing the
// records that follow it.
func rewriteInspectEnvelope(t *testing.T, path string, change func(map[string]any)) {
	t.Helper()
	log := readInspectLog(t, path)
	length := int(binary.LittleEndian.Uint32(log[8:]))
	end := 16 + length + 4
	var envelope map[string]any
	if err := json.Unmarshal(log[16:16+length], &envelope); err != nil {
		t.Fatal(err)
	}
	change(envelope)
	payload, err := logstore.EncodeJSON(envelope)
	if err != nil {
		t.Fatal(err)
	}
	rewritten := append(inspectLogFrame(uint16(binary.LittleEndian.Uint16(log[6:])), payload), log[end:]...)
	if err := os.WriteFile(filepath.Join(path, "store", logstore.LogFile), rewritten, 0o600); err != nil {
		t.Fatal(err)
	}
}

// removeInspectRecord removes the first complete record selected by remove
// without changing any other frame.
func removeInspectRecord(t *testing.T, path string, remove func(kind uint16, payload []byte) bool) {
	t.Helper()
	log := readInspectLog(t, path)
	rewritten := make([]byte, 0, len(log))
	removed := false
	for offset := 0; offset < len(log); {
		length := int(binary.LittleEndian.Uint32(log[offset+8:]))
		end := offset + 16 + length + 4
		kind := binary.LittleEndian.Uint16(log[offset+6:])
		payload := log[offset+16 : offset+16+length]
		if !removed && remove(kind, payload) {
			removed = true
		} else {
			rewritten = append(rewritten, log[offset:end]...)
		}
		offset = end
	}
	if !removed {
		t.Fatal("inspection log has no matching record to remove")
	}
	if err := os.WriteFile(filepath.Join(path, "store", logstore.LogFile), rewritten, 0o600); err != nil {
		t.Fatal(err)
	}
}

// inspectFrameBoundaries returns the offset of every frame of a valid log and
// its end.
func inspectFrameBoundaries(log []byte) []int64 {
	boundaries := []int64{0}
	for offset := int64(0); offset < int64(len(log)); {
		offset += 16 + int64(binary.LittleEndian.Uint32(log[offset+8:])) + 4
		boundaries = append(boundaries, offset)
	}
	return boundaries
}

// A campaign inspects from its log while its writer holds the lock and
// appends: every inspection reports a validated snapshot of the committed
// outcomes and changes nothing, and the published campaign reports the same
// log it replayed.
func TestInspectReportsCampaignLogStateWhileAWriterAppends(t *testing.T) {
	const executions = 24
	journal, err := campaign.NewCampaignJournal(context.Background(), campaign.CampaignConfig{
		Root: t.TempDir(), CampaignID: "run-inspect-live", Selection: "0-23", SelectionCount: executions,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func(journal *campaign.CampaignJournal) {
		if err := journal.Close(); err != nil {
			t.Error(err)
		}
	}(journal)
	if err := journal.StartExecutions(); err != nil {
		t.Fatal(err)
	}
	appendDone := make(chan error, 1)
	go func() {
		for ordinal := range uint64(executions) {
			if err := journal.AppendExecution(campaign.ExecutionRecord{
				SelectionOrdinal: record.Uint64String(ordinal), Seed: record.Uint64String(ordinal), Domain: "success", Reason: "success", Termination: "exit", ElapsedNanos: 5,
			}); err != nil {
				appendDone <- err
				return
			}
		}
		appendDone <- nil
	}()
	var appendErr error
	appending := true
	var observed uint64
	for observed < executions {
		report, err := inspect.Inspect(journal.Path(), inspect.InspectOptions{})
		if err != nil {
			if appending {
				appendErr = <-appendDone
			}
			t.Fatalf("inspect.Inspect() while appending after %d outcomes: %v (append: %v)", observed, err, appendErr)
		}
		log, committed := report.Log, report.Committed
		if report.Kind != "campaign" || report.Campaign != nil || report.Lifecycle == nil || report.Lifecycle.State != "running" || log == nil || committed == nil ||
			log.Format != campaign.CampaignLogFormat || log.FrameVersion != logstore.FrameVersion || log.Tail != "none" || log.State != "valid" || log.ValidBytes != log.SnapshotBytes || log.RepairedBytes != 0 {
			t.Fatalf("live campaign inspection = %#v", report)
		}
		if committed.Outcomes < observed || committed.Outcomes > executions || committed.Limits.MaximumExecutions != executions {
			t.Fatalf("committed outcomes went from %d to %#v", observed, committed)
		}
		observed = committed.Outcomes
		select {
		case appendErr = <-appendDone:
			appending = false
		default:
		}
		if appendErr != nil {
			break
		}
	}
	if appending {
		appendErr = <-appendDone
	}
	if appendErr != nil {
		t.Fatal(appendErr)
	}
	before := readInspectLog(t, journal.Path())
	if err := journal.Publish(campaign.CampaignSummary{Attempted: executions, Succeeded: executions, StopReason: "seeds_exhausted"}); err != nil {
		t.Fatal(err)
	}
	published := readInspectLog(t, journal.Path())
	if !bytes.HasPrefix(published, before) {
		t.Fatal("publication rewrote the committed log")
	}
	report, err := inspect.Inspect(journal.Path(), inspect.InspectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if report.Campaign == nil || report.Campaign.Journal == nil || report.Campaign.Journal.Records != executions || report.Committed.Outcomes != executions ||
		report.Committed.OutcomeBytes != report.Campaign.Journal.Bytes || report.Log.SnapshotBytes != uint64(len(published)) || report.Lifecycle.State != "published" {
		t.Fatalf("published campaign inspection = %#v", report)
	}
	if after := readInspectLog(t, journal.Path()); !bytes.Equal(after, published) {
		t.Fatal("inspection changed the campaign log")
	}
}

// An interrupted final record is observed and reported beside the committed
// state it follows; inspection neither truncates it nor needs the writer lock.
func TestInspectObservesATornCampaignTailWithoutRepairingIt(t *testing.T) {
	for _, test := range []struct {
		name string
		tail func(frame []byte) []byte
		want string
	}{
		{name: "short frame", tail: func(frame []byte) []byte { return frame[:len(frame)-3] }, want: "short-frame"},
		{name: "payload checksum", tail: func(frame []byte) []byte { frame[20] ^= 0xff; return frame }, want: "payload-checksum"},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := writeInspectBatchForChoiceTest(t)
			clean, err := inspect.Inspect(path, inspect.InspectOptions{})
			if err != nil {
				t.Fatal(err)
			}
			torn := test.tail(inspectLogFrame(2, []byte(`{"state":"running"}`)))
			appendInspectLog(t, path, torn)
			before := readInspectLog(t, path)
			report, err := inspect.Inspect(path, inspect.InspectOptions{})
			if err != nil {
				t.Fatal(err)
			}
			log := report.Log
			if log.State != "torn-tail" || log.Tail != test.want || log.TailBytes != uint64(len(torn)) || log.RepairedBytes != 0 || log.Records != clean.Log.Records ||
				log.ValidBytes != clean.Log.ValidBytes || log.SnapshotBytes != clean.Log.SnapshotBytes+uint64(len(torn)) || log.Invalid != nil {
				t.Fatalf("torn log inspection = %#v", log)
			}
			if report.Campaign == nil || report.Campaign.Attempted != clean.Campaign.Attempted || report.Committed.Outcomes != clean.Committed.Outcomes {
				t.Fatalf("torn campaign inspection = %#v", report)
			}
			if after := readInspectLog(t, path); !bytes.Equal(after, before) {
				t.Fatal("inspection repaired the torn tail")
			}
		})
	}
}

const inspectCampaignEnvelope = `{"format":"gomad.campaign-log/v1","campaign_id":"run-damaged","strategy":"seed","selection":"7","selection_count":"1"}`

func inspectCorpusEnvelope() []byte {
	return []byte(`{"format":"gomad.guide-corpus-log/v1","identity":{"environment_sha256":"sha256:` + strings.Repeat("e", 64) + `","target_sha256":"sha256:` + strings.Repeat("a", 64) + `","toolchain":{"go_version":"go1.27.1","build_key":"key","target_goos":"linux","target_goarch":"arm64"},"boundary_version":"v1","boundary_sha256":"sha256:` + strings.Repeat("b", 64) + `","instrumentation_schema":"gomad.semantic-features/v1","instrumentation_sha256":"sha256:` + strings.Repeat("c", 64) + `","manifest_schema_version":1,"manifest_record_contract":"gomad.execution-record/v1"}}`)
}

// Damage that is not a torn final record, an envelope of another lineage, or
// a log without its envelope is an invalid store: inspection reports where its
// validated prefix ends and why, reports no state, and changes nothing.
// Missing or invalid store metadata and earlier layouts keep their errors.
func TestInspectDiagnosesADamagedLogAsAnInvalidStore(t *testing.T) {
	published := readInspectLog(t, writeInspectBatchForChoiceTest(t))
	boundaries := inspectFrameBoundaries(published)
	flipped := bytes.Clone(published)
	flipped[boundaries[2]+18] ^= 0xff
	unknownKind := append(inspectLogFrame(1, []byte(inspectCampaignEnvelope)), inspectLogFrame(99, []byte(`{}`))...)
	invalidTransition := append(inspectLogFrame(1, []byte(inspectCampaignEnvelope)), inspectLogFrame(2, []byte(`{"state":"published"}`))...)
	envelopeBytes := uint64(len(inspectLogFrame(1, []byte(inspectCampaignEnvelope))))
	corpusEnvelope := inspectCorpusEnvelope()
	corpusEnvelopeBytes := uint64(len(inspectLogFrame(1, corpusEnvelope)))
	for _, test := range []struct {
		name    string
		log     []byte
		kind    string
		format  string
		problem string
		offset  uint64
		records uint64
		reason  string
	}{
		{name: "earlier record damage", log: flipped, kind: "campaign", format: campaign.CampaignLogFormat, problem: "corrupt", offset: uint64(boundaries[2]), records: 2, reason: "checksum"},
		{name: "unknown record kind", log: unknownKind, kind: "campaign", format: campaign.CampaignLogFormat, problem: "corrupt", offset: envelopeBytes, records: 1, reason: "kind 99"},
		{name: "invalid typed state", log: invalidTransition, kind: "campaign", format: campaign.CampaignLogFormat, problem: "corrupt", offset: envelopeBytes, records: 1, reason: "published"},
		{name: "damaged envelope", log: append([]byte("GMLX"), published[4:]...), kind: "store", problem: "corrupt", reason: "magic"},
		{name: "earlier lineage", log: inspectLogFrame(1, []byte(`{"format":"gomad.campaign-log/v0","campaign_id":"x"}`)), kind: "store", format: "gomad.campaign-log/v0", problem: "unsupported-lineage", reason: `"gomad.campaign-log/v0"`},
		{name: "envelope without lineage", log: inspectLogFrame(1, []byte(`{"campaign_id":"x"}`)), kind: "store", problem: "unsupported-lineage", reason: "does not name a format"},
		{name: "empty log", log: nil, kind: "store", problem: "incomplete", reason: "no envelope"},
		{name: "torn envelope", log: inspectLogFrame(1, []byte(inspectCampaignEnvelope))[:10], kind: "store", problem: "incomplete", reason: "no envelope"},
		{name: "corrupt corpus admission", log: append(inspectLogFrame(1, corpusEnvelope), inspectLogFrame(2, []byte(`{"generation":"7"}`))...), kind: "corpus", format: corpus.LogFormat, problem: "corrupt", offset: corpusEnvelopeBytes, records: 1, reason: "generation"},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := writeInspectStore(t, test.log)
			report, err := inspect.Inspect(path, inspect.InspectOptions{})
			var invalid *inspect.InvalidStoreError
			if !errors.As(err, &invalid) || !strings.Contains(err.Error(), test.reason) {
				t.Fatalf("inspect.Inspect() = %#v, %v", report, err)
			}
			log := invalid.Report.Log
			if invalid.Report.Kind != test.kind || invalid.Report.Path != path || invalid.Report.Schema != "gomad.inspect/v8" || log == nil || log.State != "invalid" || log.Format != test.format || log.Invalid == nil ||
				log.Invalid.Problem != test.problem || log.Invalid.Offset != test.offset || log.Invalid.Records != test.records || log.SnapshotBytes != uint64(len(test.log)) || log.RepairedBytes != 0 {
				t.Fatalf("invalid store diagnosis = %#v %#v", invalid.Report, log)
			}
			if report.Kind != invalid.Report.Kind || report.Log != invalid.Report.Log || report.Campaign != nil || report.Lifecycle != nil || report.Committed != nil || report.Corpus != nil || report.Merged != nil {
				t.Fatalf("invalid store reported state: %#v", report)
			}
			if after := readInspectLog(t, path); !bytes.Equal(after, test.log) {
				t.Fatal("inspection changed the damaged log")
			}
		})
	}

	t.Run("store metadata and earlier layouts keep their errors", func(t *testing.T) {
		missingLog := writeInspectStore(t, published)
		if err := os.Remove(filepath.Join(missingLog, "store", logstore.LogFile)); err != nil {
			t.Fatal(err)
		}
		openStore := writeInspectStore(t, published)
		if err := os.Chmod(filepath.Join(openStore, "store"), 0o755); err != nil {
			t.Fatal(err)
		}
		legacyCampaign, legacyCorpus := t.TempDir(), t.TempDir()
		for path, name := range map[string]string{filepath.Join(legacyCampaign, "campaign.json"): "{}", filepath.Join(legacyCorpus, "corpus.lock"): ""} {
			if err := os.WriteFile(path, []byte(name), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		manifestOnly := t.TempDir()
		if err := os.WriteFile(filepath.Join(manifestOnly, "corpus.json"), []byte("{}"), 0o600); err != nil {
			t.Fatal(err)
		}
		for path, matches := range map[string]func(error) bool{
			missingLog:     func(err error) bool { return errors.Is(err, os.ErrNotExist) },
			openStore:      func(err error) bool { return errors.Is(err, logstore.ErrInvalidStore) },
			legacyCampaign: func(err error) bool { return errors.Is(err, campaign.ErrUnsupportedFormat) },
			legacyCorpus:   func(err error) bool { return errors.Is(err, corpus.ErrUnsupportedFormat) },
			manifestOnly:   func(err error) bool { return err != nil && !errors.Is(err, corpus.ErrUnsupportedFormat) },
		} {
			_, err := inspect.Inspect(path, inspect.InspectOptions{})
			var invalid *inspect.InvalidStoreError
			if errors.As(err, &invalid) || !matches(err) {
				t.Fatalf("inspect.Inspect(%s) error = %v", path, err)
			}
		}
	})
}

func TestInspectClassifiesInvalidCurrentCorpusHeadersAsCorruption(t *testing.T) {
	for _, test := range []struct {
		name    string
		payload string
		reason  string
	}{
		{name: "invalid identity", payload: `{"format":"gomad.guide-corpus-log/v1","identity":{}}`, reason: "identity is invalid"},
		{name: "unknown field", payload: `{"format":"gomad.guide-corpus-log/v1","identity":{},"future":true}`, reason: "unknown field"},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := writeInspectStore(t, inspectLogFrame(1, []byte(test.payload)))
			_, err := inspect.Inspect(path, inspect.InspectOptions{})
			var invalid *inspect.InvalidStoreError
			if !errors.As(err, &invalid) || invalid.Report.Kind != "corpus" || invalid.Report.Log == nil || invalid.Report.Log.Invalid == nil ||
				invalid.Report.Log.Invalid.Problem != "corrupt" || invalid.Report.Log.Invalid.Offset != 0 || invalid.Report.Log.Invalid.Records != 0 || !strings.Contains(err.Error(), test.reason) {
				t.Fatalf("inspect.Inspect() error = %#v", err)
			}
		})
	}
}

func TestInspectRejectsANonPrivateCorpusRootWithoutMutation(t *testing.T) {
	log := inspectLogFrame(1, []byte(`{"format":"gomad.guide-corpus-log/v1","identity":{}}`))
	path := writeInspectStore(t, log)
	if err := os.Chmod(path, 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := inspect.Inspect(path, inspect.InspectOptions{})
	var invalid *inspect.InvalidStoreError
	if err == nil || errors.As(err, &invalid) || !strings.Contains(err.Error(), "private directory") {
		t.Fatalf("inspect.Inspect() error = %v", err)
	}
	if after := readInspectLog(t, path); !bytes.Equal(after, log) {
		t.Fatal("inspection changed a non-private corpus")
	}
}

// A guided corpus inspects through the same dispatcher as a campaign: its log
// lineage, live entries and capacity come from one read-only replay, and the
// corpus is left exactly as it was.
func TestInspectReportsAGuidedCorpusFromItsLog(t *testing.T) {
	root := filepath.Join(t.TempDir(), "corpus")
	replayer := &matchingReplayer{}
	executor := &fakeExecutor{result: func(seed uint64) backend.ExecutionResult {
		result := processResult(0, "", "")
		probe := "stdlib.os.openfile"
		if seed%2 == 1 {
			probe = "stdlib.net.dialtcp"
		}
		result.IOTranscript = semanticTranscript(t, probe)
		return result
	}}
	config, configDependencies := testConfig(t, newFakePreparer(t), executor, "0-3", PolicyAll, 1)
	config.Coverage = campaignoptions.CoverageSemantic
	config.Guide = true
	config.Corpus = root
	config.Replayer = replayer
	summary, err := exploreWith(context.Background(), config, configDependencies)
	if err != nil {
		t.Fatal(err)
	}
	before := runnerTreeSnapshot(t, root)
	report, err := inspect.Inspect(root, inspect.InspectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	inspected, log := report.Corpus, report.Log
	if report.Kind != "corpus" || inspected == nil || log == nil || report.Campaign != nil || report.Lifecycle != nil || log.Format != corpus.LogFormat || log.State != "valid" ||
		log.Records != summary.CorpusEntries+1 || log.ValidBytes != log.SnapshotBytes || inspected.Generation != summary.CorpusEntries || inspected.Entries != summary.CorpusEntries || summary.CorpusEntries != 2 ||
		inspected.MaximumEntries != 1024 || inspected.MaximumBytes != 1<<30 || inspected.Schema != corpus.CorpusSchema || inspected.SnapshotSHA256 == "" || inspected.TargetSHA256 == "" {
		t.Fatalf("corpus inspection = %#v %#v", inspected, log)
	}
	var stored uint64
	for _, entry := range inspected.Cases {
		if _, err := os.Stat(filepath.Join(entry.Path, "manifest.json")); err != nil || len(entry.NoveltyReasons) == 0 || entry.StoredBytes == 0 {
			t.Fatalf("corpus case = %#v: %v", entry, err)
		}
		stored += entry.StoredBytes
	}
	if stored != inspected.EntryStoredBytes {
		t.Fatalf("entry stored bytes = %d, cases sum to %d", inspected.EntryStoredBytes, stored)
	}
	if _, err := inspect.Inspect(root, inspect.InspectOptions{Choices: true}); err == nil || !strings.Contains(err.Error(), "traced artifact") {
		t.Fatalf("corpus choice inspection error = %v", err)
	}
	torn := inspectLogFrame(2, []byte(`{}`))[:7]
	appendInspectLog(t, root, torn)
	tornReport, err := inspect.Inspect(root, inspect.InspectOptions{})
	if err != nil || tornReport.Log.State != "torn-tail" || tornReport.Log.TailBytes != uint64(len(torn)) || tornReport.Corpus.Generation != inspected.Generation {
		t.Fatalf("torn corpus inspection = %#v, %v", tornReport.Log, err)
	}
	if err := os.Truncate(filepath.Join(root, "store", logstore.LogFile), int64(log.SnapshotBytes)); err != nil {
		t.Fatal(err)
	}
	if after := runnerTreeSnapshot(t, root); !maps.Equal(after, before) {
		t.Fatal("corpus inspection changed the corpus")
	}
}

// inspectedRounds reads a campaign's exploration rounds through public
// inspection; it reports false for a seed campaign.
func inspectedRounds(path string) (inspect.ExplorationInspection, bool, error) {
	report, err := inspect.Inspect(path, inspect.InspectOptions{})
	if err != nil || report.Exploration == nil {
		return inspect.ExplorationInspection{}, false, err
	}
	return *report.Exploration, true, nil
}

func TestInspectReportsArtifactIdentityWithoutPresentation(t *testing.T) {
	published := publishInspectArtifact(t)
	report, err := inspect.Inspect(published.Path, inspect.InspectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if report.Kind != "artifact" || report.Artifact == nil || report.Campaign != nil {
		t.Fatalf("report = %#v", report)
	}
	observed := report.Artifact
	if observed.Seed != 7 || observed.Target.Source != "./target" || observed.Outcome.Reason != "nonzero_exit" || observed.Transcript == nil || observed.Transcript.Records != 3 {
		t.Fatalf("artifact report = %#v", observed)
	}
	if !observed.Stdout.Truncated {
		t.Fatalf("artifact details = %#v", observed)
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "replay_command") {
		t.Fatalf("inspection contains CLI presentation: %s", encoded)
	}
}

func TestOpenReportsSimulationExplorationEvidence(t *testing.T) {
	preparer := newFakePreparer(t)
	limit := choiceTraceLimit(t, 1)
	executor := &simulationExplorationExecutor{t: t, buildKey: preparer.prepared.BuildKey, limit: limit, fail: true}
	config, configDependencies := testConfig(t, preparer, executor, "7", PolicyAll, 1)
	config.Strategy = StrategySimulationExploration
	config.ChoiceTraceLimit = limit
	config.MaxExecutions = 1
	config.MaxForcedDecisions = 1
	config.MaxExplorationBytes = 1 << 20
	config.MaxExplorationResultBytes = 1 << 20
	config.SimulationDimensionLimits = SimulationDimensionLimits{Runtime: 1, Scenario: 1, Network: 1, Storage: 1, Fault: 1, Crash: 1}
	configDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)

	summary, err := exploreWith(context.Background(), config, configDependencies)
	if err != nil {
		t.Fatal(err)
	}
	report, err := inspect.Inspect(summary.Artifacts[0], inspect.InspectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	simulation := report.Artifact.Simulation
	if simulation == nil || simulation.ControllerSHA256 == "" || simulation.ExecutionSHA256 == "" || simulation.CandidateSHA256 == "" || simulation.OutcomeSHA256 == "" || simulation.FailureSHA256 == "" || simulation.Plan.Bytes == 0 || simulation.Plan.SHA256 == "" || simulation.Record.Bytes == 0 || simulation.Record.SHA256 == "" || simulation.Record.Limit != 128<<20 {
		t.Fatalf("simulation inspection = %#v", simulation)
	}
}

func TestOpenWithOptionsProjectsValidatedChoiceTrace(t *testing.T) {
	published := publishInspectArtifact(t)
	report, err := inspect.Inspect(published.Path, inspect.InspectOptions{Choices: true})
	if err != nil {
		t.Fatal(err)
	}
	choices := report.Artifact.Choices
	if choices == nil || choices.Records != 1 || choices.BranchingRecords != 1 || choices.Runnable != 1 || choices.SelectPoll != 0 || choices.SelectResult != 0 || len(choices.Sites) != 1 {
		t.Fatalf("choice inspection = %#v", choices)
	}
	if choices.Sites[0].Kind != "runnable" || choices.Sites[0].MaximumAlternatives != 2 || choices.Sites[0].Fingerprint == "" {
		t.Fatalf("choice site = %#v", choices.Sites)
	}
}

func TestOpenWithOptionsRejectsBatch(t *testing.T) {
	batch := writeInspectBatchForChoiceTest(t)
	if _, err := inspect.Inspect(batch, inspect.InspectOptions{Choices: true}); err == nil || !strings.Contains(err.Error(), "traced artifact") {
		t.Fatalf("batch choice inspection error = %v", err)
	}
}

func TestOpenReportsValidatedBatchRuns(t *testing.T) {
	journal, err := campaign.NewCampaignJournal(context.Background(), campaign.CampaignConfig{
		Root: t.TempDir(), CampaignID: "run-inspect", Selection: "7", SelectionCount: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func(journal *campaign.CampaignJournal) {
		if err := journal.Close(); err != nil {
			t.Error(err)
		}
	}(journal)
	if err := journal.StartExecutions(); err != nil {
		t.Fatal(err)
	}
	if err := journal.AppendExecution(campaign.ExecutionRecord{
		SelectionOrdinal: 0, Seed: 7, Domain: "success", Reason: "success", Termination: "exit", ElapsedNanos: 5,
	}); err != nil {
		t.Fatal(err)
	}
	if err := journal.Publish(campaign.CampaignSummary{Attempted: 1, Succeeded: 1, StopReason: "seeds_exhausted"}); err != nil {
		t.Fatal(err)
	}
	report, err := inspect.Inspect(journal.Path(), inspect.InspectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if report.Kind != "campaign" || report.Campaign == nil || report.Lifecycle == nil || report.Lifecycle.State != "published" || report.Artifact != nil || report.Campaign.CampaignID != "run-inspect" || len(report.Campaign.Executions) != 1 || report.Campaign.Journal == nil || report.Campaign.Journal.Records != 1 {
		t.Fatalf("report = %#v", report)
	}
}

func TestOpenReportsSimulationExplorationBoundsAndRemainingWork(t *testing.T) {
	preparer := newFakePreparer(t)
	limit := choiceTraceLimit(t, 1)
	executor := &simulationExplorationExecutor{t: t, buildKey: preparer.prepared.BuildKey, limit: limit}
	config, configDependencies := testConfig(t, preparer, executor, "7", PolicyAll, 1)
	config.Strategy = StrategySimulationExploration
	config.ChoiceTraceLimit = limit
	config.MaxExecutions = 4
	config.MaxForcedDecisions = 1
	config.MaxExplorationBytes = 1 << 20
	config.MaxExplorationResultBytes = 1 << 20
	config.SimulationDimensionLimits = SimulationDimensionLimits{Runtime: 2, Scenario: 1, Network: 2, Storage: 2, Fault: 2, Crash: 2}
	configDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)

	summary, err := exploreWith(context.Background(), config, configDependencies)
	if err != nil {
		t.Fatal(err)
	}
	report, err := inspect.Inspect(summary.CampaignPath, inspect.InspectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	combined := report.Campaign.SimulationExploration
	if combined == nil || combined.LogicalExecutions != 2 || combined.Pending != 0 || combined.Limits.Scenario != 1 || report.Campaign.SimulationExplorationImplementationSHA256 == "" || report.Campaign.SimulationExplorationChainSHA256 == "" {
		t.Fatalf("simulation exploration inspection = %#v", report.Campaign)
	}
	if len(report.Campaign.Executions) != 2 || report.Campaign.Executions[0].Strategy != string(StrategySimulationExploration) || report.Campaign.Executions[1].ForcedDepth == nil || *report.Campaign.Executions[1].ForcedDepth != 1 {
		t.Fatalf("simulation exploration runs = %#v", report.Campaign.Executions)
	}
}

func TestOpenReportsInterruptedBatchLifecycle(t *testing.T) {
	journal, err := campaign.NewCampaignJournal(context.Background(), campaign.CampaignConfig{
		Root: t.TempDir(), CampaignID: "run-interrupted-inspect", Selection: "7", SelectionCount: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := journal.Close(); err != nil {
			t.Error(err)
		}
	})
	report, err := inspect.Inspect(journal.Path(), inspect.InspectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if report.Kind != "campaign" || report.Campaign != nil || report.Artifact != nil || report.Lifecycle == nil || report.Lifecycle.State != "planned" || report.Lifecycle.Resumable || report.Lifecycle.Published {
		t.Fatalf("interrupted batch report = %#v", report)
	}
}

// An interrupted simulation exploration reports its committed rounds, its
// pending candidates and the round it started without committing, read from
// its log without changing it.
func TestOpenReportsInterruptedSimulationExplorationPendingAndStagedWork(t *testing.T) {
	preparer := newFakePreparer(t)
	limit := choiceTraceLimit(t, 1)
	baseExecutor := &simulationExplorationExecutor{t: t, buildKey: preparer.prepared.BuildKey, limit: limit}
	config, configDependencies := testConfig(t, preparer, simulationExplorationInterruptExecutor{exploration: baseExecutor}, "7", PolicyAll, 1)
	config.Strategy = StrategySimulationExploration
	config.ChoiceTraceLimit = limit
	config.MaxExecutions = 4
	config.MaxForcedDecisions = 2
	config.MaxExplorationBytes = 1 << 20
	config.MaxExplorationResultBytes = 1 << 20
	config.SimulationDimensionLimits = SimulationDimensionLimits{Runtime: 2, Scenario: 2, Network: 2, Storage: 2, Fault: 2, Crash: 2}
	partial, err := exploreWith(context.Background(), config, configDependencies)
	if err == nil {
		t.Fatal("interrupted simulation exploration succeeded")
	}
	logPath := filepath.Join(partial.CampaignPath, "store", "log")
	before, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	report, err := inspect.Inspect(partial.CampaignPath, inspect.InspectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	combined, rounds := report.SimulationExploration, report.Exploration
	if report.Campaign != nil || combined == nil || rounds == nil || combined.Summary.Pending != 1 || len(combined.Pending) != 1 || combined.StagedRound == nil || combined.StagedRound.Index != 1 || combined.StagedRound.Attempted != 1 {
		t.Fatalf("interrupted simulation exploration report = %#v", report)
	}
	if rounds.Strategy != string(StrategySimulationExploration) || len(rounds.Rounds) != 1 || rounds.StagedRound == nil || rounds.StagedRound.Candidates != 1 || rounds.ChainSHA256 != combined.ChainSHA256 || rounds.Rounds[0].RoundBytes == 0 {
		t.Fatalf("interrupted exploration rounds = %#v", rounds)
	}
	after, err := os.ReadFile(logPath)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("inspection changed the campaign log: %v", err)
	}
}

func TestOpenReportsAndValidatesRetainedSuccessfulRuns(t *testing.T) {
	journal, err := campaign.NewCampaignJournal(context.Background(), campaign.CampaignConfig{
		Root: t.TempDir(), CampaignID: "run-inspect-success", Selection: "7", SelectionCount: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func(journal *campaign.CampaignJournal) {
		if err := journal.Close(); err != nil {
			t.Error(err)
		}
	}(journal)
	retained := publishInspectArtifactAt(t, journal.SuccessesPath(), "run-inspect-success", true)
	relative, err := filepath.Rel(journal.Path(), retained.Path)
	if err != nil {
		t.Fatal(err)
	}
	bytes := record.Uint64String(retained.StoredBytes)
	choiceTrace := retained.Manifest.ChoiceProfile.Trace
	if err := journal.StartExecutions(); err != nil {
		t.Fatal(err)
	}
	if err := journal.AppendExecution(campaign.ExecutionRecord{
		SelectionOrdinal: 0, Seed: 7, Domain: "success", Reason: "success", Termination: "exit", ElapsedNanos: 5,
		SuccessArtifact: &relative, SuccessArtifactBytes: &bytes, SemanticProbes: []string{"stdlib.os.openfile"}, NovelSemanticProbes: []string{"stdlib.os.openfile"},
		ChoiceTraceSHA256: &choiceTrace.SHA256, ChoiceTraceRecords: &choiceTrace.Records, ChoiceTraceBranchingRecords: &choiceTrace.BranchingRecords,
		ChoiceTraceTerminalState: &choiceTrace.TerminalState, ChoiceTapeSHA256: &choiceTrace.TapeSHA256, ChoiceDecisions: &choiceTrace.Decisions,
	}); err != nil {
		t.Fatal(err)
	}
	if err := journal.Publish(campaign.CampaignSummary{Attempted: 1, Succeeded: 1, RetainedSuccesses: 1, RetainedSuccessBytes: retained.StoredBytes, StopReason: "seeds_exhausted"}); err != nil {
		t.Fatal(err)
	}
	report, err := inspect.Inspect(journal.Path(), inspect.InspectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if report.Campaign == nil || report.Campaign.RetainedSuccesses != 1 || report.Campaign.RetainedSuccessBytes != retained.StoredBytes || len(report.Campaign.SuccessArtifacts) != 1 || report.Campaign.SuccessArtifacts[0].Path != retained.Path {
		t.Fatalf("batch report = %#v", report.Campaign)
	}
}

func publishInspectArtifact(t *testing.T) artifact.Artifact {
	t.Helper()
	return publishInspectArtifactAt(t, t.TempDir(), "batch-inspect", false)
}

func publishInspectArtifactAt(t *testing.T, root, batchID string, success bool) artifact.Artifact {
	t.Helper()
	targetPath := filepath.Join(t.TempDir(), "target")
	if err := os.WriteFile(targetPath, []byte("target"), 0o700); err != nil {
		t.Fatal(err)
	}
	world, payloads := record.NoneWorld()
	exitCode := record.Uint64String(2)
	artifactKind := record.ArtifactTargetFailure
	domain := "target"
	reason := "nonzero_exit"
	if success {
		exitCode = 0
		artifactKind = record.ArtifactSuccess
		domain = "success"
		reason = "success"
	}
	transcript := []byte(strings.Repeat("x", 128*3))
	first := sha256.Sum256([]byte("first choice alternative"))
	second := sha256.Sum256([]byte("second choice alternative"))
	decision, err := choice.CanonicalDecision(0, choice.KindRunnable, 24, false, [][sha256.Size]byte{first, second}, second, 0)
	if err != nil {
		t.Fatal(err)
	}
	choiceTrace, err := choice.BuildTrace([]choice.Record{decision.Record()}, choice.TerminalComplete)
	if err != nil {
		t.Fatal(err)
	}
	choicePayload := choiceTrace.Bytes
	choiceLimit, err := choice.TraceBytes(1)
	if err != nil {
		t.Fatal(err)
	}
	choiceImplementation, err := choice.ImplementationIdentity(strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	targetSHA256, err := record.HashBytes([]byte("target")).Bytes()
	if err != nil {
		t.Fatal(err)
	}
	choiceReplayPlan, err := choice.ProjectReplayPlan(choiceTrace, choice.ExecutionIdentity{
		TargetSHA256: targetSHA256, ToolchainBuildKey: strings.Repeat("a", 64),
		GOOS: "darwin", GOARCH: "arm64", ImplementationSHA256: choiceImplementation,
	})
	if err != nil {
		t.Fatal(err)
	}
	manifest := record.ExecutionRecord{
		SchemaVersion: record.SchemaVersion, ArtifactKind: artifactKind, CreatedAt: "2026-08-12T12:00:00Z", CampaignID: batchID, SelectionOrdinal: 0, Seed: 7, ReplayMode: record.ReplayExact,
		Runner:        record.Runner{RecordContract: record.RecordContract, RunnerBuild: "sha256:runner", HostOS: "darwin", HostArch: "arm64"},
		Toolchain:     record.Toolchain{GoVersion: "go1.26.4", BuildKey: strings.Repeat("a", 64), TargetGOOS: "darwin", TargetGOARCH: "arm64"},
		Target:        record.Target{Kind: "go-test", Source: "./target", SHA256: record.HashBytes([]byte("target")), Size: 6, Argv: []string{"gomad-target"}, BuildTags: []string{"gomad_fixture"}, Adapters: []record.TargetAdapter{}, Compatibility: []record.CompatibilityPack{}, BuildInfo: record.BuildInfo{GoVersion: "go1.26.4", Path: "example.com/target"}},
		IOProfile:     record.IOProfile{Name: "gomad-deterministic/v1", ImplementationSHA256: record.HashBytes([]byte("implementation")), Inventory: "{}", InventorySHA256: record.HashBytes([]byte("{}")), Transcript: &record.IOTranscript{Schema: "gomad.io-transcript/v1", SHA256: record.HashBytes(transcript), Bytes: record.Uint64String(len(transcript)), Records: 3}},
		ChoiceProfile: &record.ChoiceProfile{Name: choice.Profile, ImplementationSHA256: record.SHA256FromSum(choiceImplementation), Trace: record.ChoiceTrace{Schema: "gomad.choice-trace/v3", SHA256: record.SHA256FromSum(choiceTrace.SHA256), Bytes: record.Uint64String(len(choicePayload)), Records: 1, BranchingRecords: 1, TerminalState: "complete", Limit: record.Uint64String(choiceLimit), TapeSHA256: record.SHA256FromSum(choiceReplayPlan.SHA256), Decisions: 1}},
		Environment:   []record.Environment{{Name: "GOMADSEED", Value: "7"}, {Name: "GOMAD_CHOICE_PROFILE", Value: choice.Profile}, {Name: "GOMAD_IO_PROFILE", Value: "gomad-deterministic/v1"}, {Name: "TZ", Value: "UTC"}}, Limits: record.Limits{ExecutionTimeoutNanos: 1, OverallTimeoutNanos: 2, OutputBytes: 4, WorldTransitionBytes: 64, IOTranscriptBytes: 1 << 20, ChoiceTraceBytes: record.Uint64String(choiceLimit)}, World: world,
		Outcome: record.Outcome{Domain: domain, Reason: reason, Termination: "exit", ExitCode: &exitCode},
		Streams: record.Streams{
			Stdout: record.Stream{FullSHA256: record.HashBytes([]byte("long output")), TotalBytes: 11, RetainedBytes: 4, DiscardedBytes: 7, Truncated: true},
			Stderr: record.Stream{FullSHA256: record.HashBytes(nil), TotalBytes: 0, RetainedBytes: 0},
		},
		Host: record.Host{StartedAt: "2026-08-12T12:00:00Z", FinishedAt: "2026-08-12T12:00:01Z", ElapsedNanos: 1},
	}
	published, err := artifact.PublishArtifact(artifact.Store{Root: root}, artifact.ArtifactInput{
		Manifest: manifest, TargetPath: targetPath, Stdout: []byte("long"), Stderr: nil, IOTranscript: transcript, ChoiceTrace: choicePayload, World: payloads,
	})
	if err != nil {
		t.Fatal(err)
	}
	return published
}

func writeInspectBatchForChoiceTest(t *testing.T) string {
	t.Helper()
	journal, err := campaign.NewCampaignJournal(context.Background(), campaign.CampaignConfig{Root: t.TempDir(), CampaignID: "run-choice-batch", Selection: "7", SelectionCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = journal.Close() })
	if err := journal.StartExecutions(); err != nil {
		t.Fatal(err)
	}
	if err := journal.AppendExecution(campaign.ExecutionRecord{SelectionOrdinal: 0, Seed: 7, Domain: "success", Reason: "success", Termination: "exit"}); err != nil {
		t.Fatal(err)
	}
	if err := journal.Publish(campaign.CampaignSummary{Attempted: 1, Succeeded: 1, StopReason: "seeds_exhausted"}); err != nil {
		t.Fatal(err)
	}
	return journal.Path()
}

func TestInspectCleanupPreservesNoChoiceArtifact(t *testing.T) {
	published := publishInspectCleanupArtifact(t)
	exitCode := uint64(0)
	want := inspect.Inspection{
		Schema: "gomad.inspect/v8", Kind: "artifact", Path: published.Path,
		Artifact: &inspect.ArtifactInspection{
			ArtifactKind: "gomad.success/v1", RecordHash: published.Manifest.RecordHash, CampaignID: "inspect-cleanup", Seed: 7, ReplayMode: "exact",
			Runner:    record.Runner{RecordContract: record.RecordContract, RunnerBuild: "sha256:runner", HostOS: "darwin", HostArch: "arm64"},
			Toolchain: record.Toolchain{GoVersion: "go1.26.4", BuildKey: strings.Repeat("a", 64), TargetGOOS: "darwin", TargetGOARCH: "arm64"},
			Target: inspect.TargetReport{
				Kind: "go-test", Source: "./target", SHA256: record.HashBytes([]byte("target")), Size: 6,
				Argv: []string{"gomad-target"}, BuildTags: []string{"gomad_fixture"},
				BuildInfo: record.BuildInfo{GoVersion: "go1.26.4", Path: "example.com/target"}, CapabilityMode: "closure", Sharing: "private",
			},
			Outcome:    inspect.OutcomeReport{Domain: "success", Reason: "success", Termination: "exit", ExitCode: &exitCode, FailureSignature: published.Manifest.Outcome.FailureSignature},
			Transcript: &inspect.Transcript{Schema: "gomad.io-transcript/v1", SHA256: record.HashBytes([]byte(strings.Repeat("x", 384))), Bytes: 384, Records: 3},
			Stdout:     inspect.StreamReport{FullSHA256: record.HashBytes([]byte("long output")), TotalBytes: 11, RetainedBytes: 4, DiscardedBytes: 7, Truncated: true},
			Stderr:     inspect.StreamReport{FullSHA256: record.HashBytes(nil)},
		},
	}
	wantBytes, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	for _, sharing := range []string{"private", "shared"} {
		t.Run(sharing, func(t *testing.T) {
			if sharing == "shared" {
				if err := os.Link(filepath.Join(published.Path, "target"), filepath.Join(t.TempDir(), "shared-target")); err != nil {
					t.Fatal(err)
				}
				want.Artifact.Target.Sharing = "shared"
				wantBytes, err = json.Marshal(want)
				if err != nil {
					t.Fatal(err)
				}
			}
			report, err := inspect.Inspect(published.Path, inspect.InspectOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(report, want) {
				t.Fatalf("inspection = %#v, artifact = %#v, want %#v", report, report.Artifact, want.Artifact)
			}
			encoded, err := json.Marshal(report)
			if err != nil {
				t.Fatal(err)
			}
			if string(encoded) != string(wantBytes) {
				t.Fatalf("canonical inspection = %s, want %s", encoded, wantBytes)
			}
		})
	}
	report, err := inspect.Inspect(published.Path, inspect.InspectOptions{Choices: true})
	if err == nil || err.Error() != "artifact has no choice trace" || reflect.TypeOf(err) != reflect.TypeOf(errors.New("")) || !reflect.DeepEqual(report, inspect.Inspection{}) {
		t.Fatalf("choice inspection = %#v, error = %v", report, err)
	}
}

func TestInspectCleanupPreservesEarlierErrors(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	report, err := inspect.Inspect(missing, inspect.InspectOptions{Choices: true})
	pathError, direct := err.(*os.PathError)
	if !direct || !errors.Is(err, os.ErrNotExist) || pathError.Op != "lstat" || pathError.Path != missing || !reflect.DeepEqual(report, inspect.Inspection{}) {
		t.Fatalf("missing inspection = %#v, error = %v", report, err)
	}
	published := publishInspectCleanupArtifact(t)
	if err := os.Mkdir(filepath.Join(published.Path, "store"), 0o700); err != nil {
		t.Fatal(err)
	}
	report, err = inspect.Inspect(published.Path, inspect.InspectOptions{Choices: true})
	if err == nil || err.Error() != "inspection path contains conflicting artifact records" || reflect.TypeOf(err) != reflect.TypeOf(errors.New("")) || !reflect.DeepEqual(report, inspect.Inspection{}) {
		t.Fatalf("conflicting inspection = %#v, error = %v", report, err)
	}
}

func publishInspectCleanupArtifact(t *testing.T) artifact.Artifact {
	t.Helper()
	published := publishInspectArtifactAt(t, t.TempDir(), "inspect-cleanup", true)
	manifest := published.Manifest
	manifest.ChoiceProfile = nil
	manifest.Environment = slices.DeleteFunc(manifest.Environment, func(entry record.Environment) bool { return entry.Name == "GOMAD_CHOICE_PROFILE" })
	manifest.Limits.ChoiceTraceBytes = 0
	world, payloads := record.NoneWorld()
	manifest.World = world
	result, err := artifact.PublishArtifact(artifact.Store{Root: t.TempDir()}, artifact.ArtifactInput{
		Manifest: manifest, TargetPath: filepath.Join(published.Path, "target"), Stdout: []byte("long"),
		IOTranscript: []byte(strings.Repeat("x", 384)), World: payloads,
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}
