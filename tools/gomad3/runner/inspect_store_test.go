package runner

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"hash/crc32"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	backend "go.temporal.io/server/tools/gomad3/backends"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/internal/campaign"
	"go.temporal.io/server/tools/gomad3/runner/internal/corpus"
	"go.temporal.io/server/tools/gomad3/runner/internal/logstore"
)

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
		report, err := Inspect(journal.Path(), InspectOptions{})
		if err != nil {
			if appending {
				appendErr = <-appendDone
			}
			t.Fatalf("Inspect() while appending after %d outcomes: %v (append: %v)", observed, err, appendErr)
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
	report, err := Inspect(journal.Path(), InspectOptions{})
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
			clean, err := Inspect(path, InspectOptions{})
			if err != nil {
				t.Fatal(err)
			}
			torn := test.tail(inspectLogFrame(2, []byte(`{"state":"running"}`)))
			appendInspectLog(t, path, torn)
			before := readInspectLog(t, path)
			report, err := Inspect(path, InspectOptions{})
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

const inspectCampaignEnvelope = `{"format":"gomad3.campaign-log/v1","campaign_id":"run-damaged","strategy":"seed","selection":"7","selection_count":"1"}`

func inspectCorpusEnvelope() []byte {
	return []byte(`{"format":"gomad3.guide-corpus-log/v1","identity":{"environment_sha256":"sha256:` + strings.Repeat("e", 64) + `","target_sha256":"sha256:` + strings.Repeat("a", 64) + `","toolchain":{"go_version":"go1.27.1","build_key":"key","target_goos":"linux","target_goarch":"arm64"},"boundary_version":"v1","boundary_sha256":"sha256:` + strings.Repeat("b", 64) + `","instrumentation_schema":"gomad3.semantic-features/v1","instrumentation_sha256":"sha256:` + strings.Repeat("c", 64) + `","manifest_schema_version":1,"manifest_record_contract":"gomad3.execution-record/v1"}}`)
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
		{name: "earlier lineage", log: inspectLogFrame(1, []byte(`{"format":"gomad3.campaign-log/v0","campaign_id":"x"}`)), kind: "store", format: "gomad3.campaign-log/v0", problem: "unsupported-lineage", reason: `"gomad3.campaign-log/v0"`},
		{name: "envelope without lineage", log: inspectLogFrame(1, []byte(`{"campaign_id":"x"}`)), kind: "store", problem: "unsupported-lineage", reason: "does not name a format"},
		{name: "empty log", log: nil, kind: "store", problem: "incomplete", reason: "no envelope"},
		{name: "torn envelope", log: inspectLogFrame(1, []byte(inspectCampaignEnvelope))[:10], kind: "store", problem: "incomplete", reason: "no envelope"},
		{name: "corrupt corpus admission", log: append(inspectLogFrame(1, corpusEnvelope), inspectLogFrame(2, []byte(`{"generation":"7"}`))...), kind: "corpus", format: corpus.LogFormat, problem: "corrupt", offset: corpusEnvelopeBytes, records: 1, reason: "generation"},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := writeInspectStore(t, test.log)
			report, err := Inspect(path, InspectOptions{})
			var invalid *InvalidStoreError
			if !errors.As(err, &invalid) || !strings.Contains(err.Error(), test.reason) {
				t.Fatalf("Inspect() = %#v, %v", report, err)
			}
			log := invalid.Report.Log
			if invalid.Report.Kind != test.kind || invalid.Report.Path != path || invalid.Report.Schema != reportSchema || log == nil || log.State != "invalid" || log.Format != test.format || log.Invalid == nil ||
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
			_, err := Inspect(path, InspectOptions{})
			var invalid *InvalidStoreError
			if errors.As(err, &invalid) || !matches(err) {
				t.Fatalf("Inspect(%s) error = %v", path, err)
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
		{name: "invalid identity", payload: `{"format":"gomad3.guide-corpus-log/v1","identity":{}}`, reason: "identity is invalid"},
		{name: "unknown field", payload: `{"format":"gomad3.guide-corpus-log/v1","identity":{},"future":true}`, reason: "unknown field"},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := writeInspectStore(t, inspectLogFrame(1, []byte(test.payload)))
			_, err := Inspect(path, InspectOptions{})
			var invalid *InvalidStoreError
			if !errors.As(err, &invalid) || invalid.Report.Kind != "corpus" || invalid.Report.Log == nil || invalid.Report.Log.Invalid == nil ||
				invalid.Report.Log.Invalid.Problem != "corrupt" || invalid.Report.Log.Invalid.Offset != 0 || invalid.Report.Log.Invalid.Records != 0 || !strings.Contains(err.Error(), test.reason) {
				t.Fatalf("Inspect() error = %#v", err)
			}
		})
	}
}

func TestInspectRejectsANonPrivateCorpusRootWithoutMutation(t *testing.T) {
	log := inspectLogFrame(1, []byte(`{"format":"gomad3.guide-corpus-log/v1","identity":{}}`))
	path := writeInspectStore(t, log)
	if err := os.Chmod(path, 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := Inspect(path, InspectOptions{})
	var invalid *InvalidStoreError
	if err == nil || errors.As(err, &invalid) || !strings.Contains(err.Error(), "private directory") {
		t.Fatalf("Inspect() error = %v", err)
	}
	if after := readInspectLog(t, path); !bytes.Equal(after, log) {
		t.Fatal("inspection changed a non-private corpus")
	}
}

func TestAppendOnlyStoreInspectionUsesVersionedSchema(t *testing.T) {
	if reportSchema != "gomad3.inspect/v8" {
		t.Fatalf("append-only store report schema = %q, want gomad3.inspect/v8", reportSchema)
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
	config.Coverage = CoverageSemantic
	config.Guide = true
	config.Corpus = root
	config.Replayer = replayer
	summary, err := exploreWith(context.Background(), config, configDependencies)
	if err != nil {
		t.Fatal(err)
	}
	before := runnerTreeSnapshot(t, root)
	report, err := Inspect(root, InspectOptions{})
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
	if _, err := Inspect(root, InspectOptions{Choices: true}); err == nil || !strings.Contains(err.Error(), "traced artifact") {
		t.Fatalf("corpus choice inspection error = %v", err)
	}
	torn := inspectLogFrame(2, []byte(`{}`))[:7]
	appendInspectLog(t, root, torn)
	tornReport, err := Inspect(root, InspectOptions{})
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
func inspectedRounds(path string) (ExplorationInspection, bool, error) {
	report, err := Inspect(path, InspectOptions{})
	if err != nil || report.Exploration == nil {
		return ExplorationInspection{}, false, err
	}
	return *report.Exploration, true, nil
}
