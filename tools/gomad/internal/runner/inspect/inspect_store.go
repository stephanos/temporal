package inspect

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"

	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	"go.temporal.io/server/tools/gomad/internal/runner/internal/campaign"
	"go.temporal.io/server/tools/gomad/internal/runner/internal/corpus"
	"go.temporal.io/server/tools/gomad/internal/runner/internal/logstore"
)

// Log states an inspection reports. A valid log ends on a record boundary; a
// torn-tail log ends in an interrupted final record that inspection observed
// and left in place; an invalid log failed replay, and only its validated
// prefix is described.
const (
	logStateValid    = "valid"
	logStateTornTail = "torn-tail"
	logStateInvalid  = "invalid"
)

// Invalid-store problems.
const (
	problemCorrupt            = "corrupt"
	problemUnsupportedLineage = "unsupported-lineage"
	problemIncomplete         = "incomplete"
)

// LogInspection describes the append-only log of a campaign, merged campaign
// or guided corpus as one read-only replay observed it. SnapshotBytes is the
// log size the replay captured when it opened the log; Records and ValidBytes
// describe the validated prefix, and ValidBytes is the offset of its end.
// TailBytes are the bytes of an observed torn final record. RepairedBytes is
// always zero: inspection takes no lock and truncates nothing, so a torn tail
// stays until a writer opens the store.
type LogInspection struct {
	Format        string                `json:"format,omitempty"`
	FrameVersion  uint8                 `json:"frame_version"`
	State         string                `json:"state"`
	SnapshotBytes uint64                `json:"snapshot_bytes"`
	Records       uint64                `json:"records"`
	ValidBytes    uint64                `json:"valid_bytes"`
	Tail          string                `json:"tail"`
	TailBytes     uint64                `json:"tail_bytes"`
	RepairedBytes uint64                `json:"repaired_bytes"`
	Invalid       *InvalidLogInspection `json:"invalid,omitempty"`
}

// InvalidLogInspection locates why replay rejected a log: Offset is the end
// of the validated prefix and the start of the rejected record, Records the
// records before it. Problem is "corrupt" for damage that is not a torn final
// record or a record whose payload or state is invalid, "unsupported-lineage"
// for an envelope that names another storage format, and "incomplete" for a
// log that ends before the records its envelope requires.
type InvalidLogInspection struct {
	Problem string `json:"problem"`
	Offset  uint64 `json:"offset"`
	Records uint64 `json:"records"`
	Reason  string `json:"reason"`
}

// CommittedCampaignInspection is the committed state a campaign log replays
// to, published or not: the outcomes it holds and their encoded bytes, the
// execution limits that bound them, and the durable admission frontier of a
// seed campaign. Admitted are the reserved ordinals without a committed
// outcome; a receipt records an admitted attempt's actual completion.
type CommittedCampaignInspection struct {
	Outcomes     uint64                           `json:"outcomes"`
	OutcomeBytes uint64                           `json:"outcome_bytes"`
	Limits       ExecutionJournalLimitsInspection `json:"limits"`
	Admitted     []AdmittedOrdinalInspection      `json:"admitted"`
	Available    uint64                           `json:"available"`
	PendingStop  string                           `json:"pending_stop,omitempty"`
	Stopped      string                           `json:"stopped,omitempty"`
}

type AdmittedOrdinalInspection struct {
	Ordinal uint64                      `json:"ordinal"`
	Seed    uint64                      `json:"seed"`
	Receipt *AdmissionReceiptInspection `json:"receipt,omitempty"`
}

type AdmissionReceiptInspection struct {
	Domain      string   `json:"domain"`
	Reason      string   `json:"reason"`
	Termination string   `json:"termination"`
	Artifacts   []string `json:"artifacts"`
	Diagnostics uint64   `json:"diagnostics"`
}

// CorpusInspection reports a guided corpus as its log replays: the live
// entries of the latest admission and the identity they share. Its capacity
// is the live corpus's: MaximumEntries cases and MaximumBytes retained bytes.
// EntryStoredBytes sums each live entry's stored bytes and so counts a target
// the cases share once per case; the capacity charges a shared pool target
// once. Historical log bytes are reported by the log and charged against
// nothing.
type CorpusInspection struct {
	Schema                string                 `json:"schema"`
	Generation            uint64                 `json:"generation"`
	TargetSHA256          record.SHA256          `json:"target_sha256"`
	EnvironmentSHA256     record.SHA256          `json:"environment_sha256"`
	ToolchainBuildKey     string                 `json:"toolchain_build_key"`
	InstrumentationSchema string                 `json:"instrumentation_schema"`
	CoverageSHA256        record.SHA256          `json:"coverage_sha256"`
	SnapshotSHA256        record.SHA256          `json:"snapshot_sha256"`
	Entries               uint64                 `json:"entries"`
	MaximumEntries        uint64                 `json:"maximum_entries"`
	EntryStoredBytes      uint64                 `json:"entry_stored_bytes"`
	MaximumBytes          uint64                 `json:"maximum_bytes"`
	Cases                 []CorpusCaseInspection `json:"cases"`
}

type CorpusCaseInspection struct {
	Seed           uint64                    `json:"seed"`
	RecordHash     record.SHA256             `json:"record_hash"`
	Path           string                    `json:"path"`
	StoredBytes    uint64                    `json:"stored_bytes"`
	PayloadBytes   uint64                    `json:"payload_bytes"`
	NoveltyReasons []CorpusFeatureInspection `json:"novelty_reasons"`
}

type CorpusFeatureInspection struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

// InvalidStoreError reports a recognized log store whose read-only replay
// failed: damage that is not a torn final record, an envelope of another
// storage lineage, or a merged log that ends early. Report is the diagnosis:
// the store's kind and path and its log's validated prefix. Nothing a partial
// replay produced is reported as state.
type InvalidStoreError struct {
	Report Inspection
	Err    error
}

func (err *InvalidStoreError) Error() string {
	return err.Err.Error()
}

func (err *InvalidStoreError) Unwrap() error {
	return err.Err
}

var (
	errEnvelopeRead = errors.New("log envelope read")
	errNoEnvelope   = errors.New("the log holds no envelope record")
)

// logStoreDirectory names the log store below a campaign, merged campaign or
// corpus directory; each owner keeps its log there.
const logStoreDirectory = "store"

// readStoreLineage reads only the envelope of the log store below path and
// returns the storage format it names. Every campaign, merged campaign and
// corpus log begins with its envelope, so the first record is read with only
// the envelope kinds admitted.
func readStoreLineage(path string) (string, logstore.Report, error) {
	kinds := campaign.EnvelopeKinds()
	for kind, bound := range corpus.EnvelopeKinds() {
		kinds[kind] = max(kinds[kind], bound)
	}
	var envelope []byte
	report, err := logstore.Replay(filepath.Join(path, logStoreDirectory), logstore.Options{Kinds: kinds}, func(entry logstore.Record) error {
		envelope = entry.Payload
		return errEnvelopeRead
	})
	if envelope == nil {
		if err == nil {
			err = errNoEnvelope
		}
		return "", report, err
	}
	var lineage struct {
		Format *string `json:"format"`
	}
	if err := json.Unmarshal(envelope, &lineage); err != nil || lineage.Format == nil || *lineage.Format == "" {
		return "", report, &unsupportedLineageError{}
	}
	return *lineage.Format, report, nil
}

// unsupportedLineageError refuses a log whose envelope names no storage
// format this build opens.
type unsupportedLineageError struct {
	Format string
}

func (err *unsupportedLineageError) Error() string {
	if err.Format == "" {
		return fmt.Sprintf("unsupported log storage format: the log envelope does not name a format; this Gomad opens only %q, %q and %q, and no migration exists", campaign.CampaignLogFormat, campaign.MergedCampaignLogFormat, corpus.LogFormat)
	}
	return fmt.Sprintf("unsupported log storage format %q: this Gomad opens only %q, %q and %q, and no migration exists", err.Format, campaign.CampaignLogFormat, campaign.MergedCampaignLogFormat, corpus.LogFormat)
}

// projectLog describes a replay that succeeded.
func projectLog(format string, report logstore.Report) *LogInspection {
	state := logStateValid
	if report.Tail != logstore.TailNone {
		state = logStateTornTail
	}
	return &LogInspection{
		Format: format, FrameVersion: logstore.FrameVersion, State: state, SnapshotBytes: uint64(report.SnapshotBytes),
		Records: report.Records, ValidBytes: uint64(report.ValidBytes), Tail: report.Tail.String(), TailBytes: uint64(report.ObservedTailBytes),
		RepairedBytes: uint64(report.RepairedBytes),
	}
}

// invalidStore turns a failed replay of a recognized log into the invalid
// store diagnosis when the failure is the log's own: corruption, an envelope
// of another lineage, or an incomplete log. Every other failure, such as
// missing or invalid store metadata, a changed snapshot or a missing
// referenced artifact, is returned unchanged with its existing precedence.
func invalidStore(kind, path, format string, report logstore.Report, err error) (Inspection, error) {
	var corruption *logstore.CorruptionError
	var lineage *unsupportedLineageError
	var corpusFormat *corpus.UnsupportedFormatError
	invalid := InvalidLogInspection{Problem: problemCorrupt, Offset: uint64(report.ValidBytes), Records: report.Records, Reason: err.Error()}
	switch {
	case errors.As(err, &lineage):
		invalid.Problem = problemUnsupportedLineage
	case errors.Is(err, errNoEnvelope), errors.Is(err, campaign.ErrIncompleteLog):
		invalid.Problem = problemIncomplete
	case errors.As(err, &corruption):
		invalid.Offset, invalid.Records, invalid.Reason = uint64(corruption.Offset), corruption.Records, corruption.Err.Error()
		if errors.Is(err, campaign.ErrUnsupportedFormat) || errors.As(err, &corpusFormat) && corpusFormat.Malformed == nil {
			invalid.Problem = problemUnsupportedLineage
		}
	case campaign.IsMergedLogCorruption(err):
		invalid.Offset, invalid.Records = 0, 0
	case errors.Is(err, logstore.ErrCorrupt):
	default:
		return Inspection{}, err
	}
	log := projectLog(format, report)
	log.State, log.Invalid = logStateInvalid, &invalid
	inspection := Inspection{Schema: reportSchema, Kind: kind, Path: path, Log: log}
	return inspection, &InvalidStoreError{Report: inspection, Err: err}
}
