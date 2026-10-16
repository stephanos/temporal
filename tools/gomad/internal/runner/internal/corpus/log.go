package corpus

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math"
	"os"
	"path/filepath"

	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	"go.temporal.io/server/tools/gomad/internal/runner/internal/logstore"
)

// LogFormat is the storage lineage of a guided corpus. The first record of
// the corpus log declares it, and Open accepts no other.
const LogFormat = "gomad.guide-corpus-log/v1"

// StoreDirectory names the corpus's log store below the corpus root. It holds
// only the log and its writer lock; cases and their shared targets stay
// beside it as files.
const StoreDirectory = "store"

// LegacySnapshotFormat names the corpora that earlier Gomad builds kept as a
// corpus.json snapshot with a corpus.lock. They are refused, never decoded or
// migrated.
const LegacySnapshotFormat = "corpus.json"

const (
	kindHeader    logstore.Kind = 1
	kindAdmission logstore.Kind = 2

	// maximumRecordBytes bounds each record by the envelope of the former
	// corpus.json snapshot, which held the identity and every live entry, so
	// no record outgrows what one snapshot could hold.
	maximumRecordBytes = 16 << 20
)

var logKinds = map[logstore.Kind]uint32{kindHeader: maximumRecordBytes, kindAdmission: maximumRecordBytes}

// EnvelopeKinds returns the record kind and bound that begin a corpus log:
// its header, which names the corpus format.
func EnvelopeKinds() map[logstore.Kind]uint32 {
	return map[logstore.Kind]uint32{kindHeader: maximumRecordBytes}
}

// MaximumEntries and MaximumBytes are the live corpus capacity: its entry
// count and its retained bytes under the stored-bytes rule. Historical log
// bytes are not charged against them.
const (
	MaximumEntries = maximumEntries
	MaximumBytes   = maximumBytes
)

// logHeader is the first record of a corpus log: its lineage and the
// identity every case must match.
type logHeader struct {
	Format   string   `json:"format"`
	Identity Identity `json:"identity"`
}

// admissionRecord is one admission transaction: the replay-verified entry,
// the live entries its bounded selection evicts, and the identity of the
// snapshot they produce.
type admissionRecord struct {
	Generation     record.Uint64String `json:"generation"`
	Entry          Entry               `json:"entry"`
	Evicted        []record.SHA256     `json:"evicted"`
	CoverageSHA256 record.SHA256       `json:"coverage_sha256"`
	SnapshotSHA256 record.SHA256       `json:"snapshot_sha256"`
}

// History describes the corpus log. Its bytes grow with every admission and
// are not charged against the live payload capacity.
type History struct {
	Records uint64
	Bytes   int64
}

// ErrUnsupportedFormat matches every UnsupportedFormatError.
var ErrUnsupportedFormat = errors.New("unsupported guided corpus format")

// UnsupportedFormatError refuses a corpus whose storage lineage is not
// LogFormat: a corpus.json corpus of an earlier Gomad (Format is
// LegacySnapshotFormat), a log whose header is missing or does not decode
// (Malformed is set), or a header that declares another format.
type UnsupportedFormatError struct {
	Format    string
	Malformed error
}

func (err *UnsupportedFormatError) Error() string {
	switch {
	case err.Malformed != nil:
		return fmt.Sprintf("%s: malformed corpus log header (%v); this Gomad reads only %q corpora and does not migrate others", ErrUnsupportedFormat, err.Malformed, LogFormat)
	case err.Format == LegacySnapshotFormat:
		return fmt.Sprintf("%s %q: the corpus holds a corpus.json snapshot of an earlier Gomad; this Gomad reads only %q corpora and does not migrate others", ErrUnsupportedFormat, LegacySnapshotFormat, LogFormat)
	default:
		return fmt.Sprintf("%s %q: this Gomad reads only %q corpora and does not migrate others", ErrUnsupportedFormat, err.Format, LogFormat)
	}
}

func (err *UnsupportedFormatError) Is(target error) bool {
	return target == ErrUnsupportedFormat
}

var errIdentityMismatch = errors.New("guided corpus identity does not match the prepared target and toolchain")

// refuseLegacyLayout fails before any mutation when root holds a corpus of
// the corpus.json layout.
func refuseLegacyLayout(root string) error {
	for _, name := range []string{"corpus.json", "corpus.lock"} {
		_, err := os.Lstat(filepath.Join(root, name))
		if err == nil {
			return &UnsupportedFormatError{Format: LegacySnapshotFormat}
		}
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("inspect guided corpus layout: %w", err)
		}
	}
	return nil
}

// preflightExistingLog validates an existing store's lineage and state using
// read-only replay before Open can chmod the corpus root, repair a tail, or
// append anything. Only an absent store is eligible for header bootstrap.
func preflightExistingLog(root string, identity Identity, limits capacity) (bool, error) {
	store := filepath.Join(root, StoreDirectory)
	_, err := os.Lstat(store)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return true, fmt.Errorf("inspect guided corpus log store: %w", err)
	}
	state := newReplayState(&identity, limits)
	if _, err := logstore.Replay(store, logstore.Options{Kinds: logKinds}, state.consume); err != nil {
		return true, openLogError(err)
	}
	if !state.started {
		return true, &UnsupportedFormatError{Malformed: errors.New("the corpus log has no committed header")}
	}
	if _, err := state.result(); err != nil {
		return true, err
	}
	return true, nil
}

// RefuseLegacyLock fails with an UnsupportedFormatError when root holds the
// writer lock of a corpus.json corpus. Inspection keys on the lock alone: a
// directory that merely holds a file named corpus.json may hold an unrelated
// qualification manifest.
func RefuseLegacyLock(root string) error {
	_, err := os.Lstat(filepath.Join(root, "corpus.lock"))
	if err == nil {
		return &UnsupportedFormatError{Format: LegacySnapshotFormat}
	}
	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect guided corpus layout: %w", err)
	}
	return nil
}

// replayState rebuilds the live hash index from a streamed corpus log. It
// retains only the live entries, never the evicted history.
type replayState struct {
	// identity is the identity the header must declare; nil accepts the
	// header's own identity.
	identity *Identity
	limits   capacity
	started  bool
	live     map[record.SHA256]Entry
	// generation counts admissions; coverage and snapshot are the identities
	// the latest admission committed.
	generation uint64
	coverage   record.SHA256
	snapshot   record.SHA256
}

func newReplayState(identity *Identity, limits capacity) *replayState {
	return &replayState{identity: identity, limits: limits, live: make(map[record.SHA256]Entry)}
}

// consume validates one record completely before applying any of it.
func (state *replayState) consume(entry logstore.Record) error {
	switch {
	case !state.started && entry.Kind != kindHeader:
		return &UnsupportedFormatError{Malformed: errors.New("the corpus log does not begin with its header")}
	case !state.started:
		header, err := decodeHeader(entry.Payload)
		if err != nil {
			return err
		}
		if state.identity != nil && !identitiesEqual(header.Identity, *state.identity) {
			return errIdentityMismatch
		}
		state.identity = &header.Identity
		state.started = true
		return nil
	case entry.Kind == kindHeader:
		return errors.New("guided corpus log repeats its header")
	default:
		return state.admit(entry.Payload)
	}
}

func decodeHeader(payload []byte) (logHeader, error) {
	var lineage struct {
		Format string `json:"format"`
	}
	if err := json.Unmarshal(payload, &lineage); err != nil {
		return logHeader{}, &UnsupportedFormatError{Malformed: err}
	}
	if lineage.Format == "" {
		return logHeader{}, &UnsupportedFormatError{Malformed: errors.New("no format")}
	}
	if lineage.Format != LogFormat {
		return logHeader{}, &UnsupportedFormatError{Format: lineage.Format}
	}
	var header logHeader
	if err := logstore.DecodeJSON(payload, &header); err != nil {
		return logHeader{}, &UnsupportedFormatError{Malformed: err}
	}
	if err := validateStoredIdentity(header.Identity); err != nil {
		return logHeader{}, err
	}
	return header, nil
}

func (state *replayState) admit(payload []byte) error {
	var admission admissionRecord
	if err := logstore.DecodeJSON(payload, &admission); err != nil {
		return fmt.Errorf("decode guided corpus admission: %w", err)
	}
	if state.generation == math.MaxUint64 || uint64(admission.Generation) != state.generation+1 {
		return fmt.Errorf("guided corpus admission generation %d does not follow %d", admission.Generation, state.generation)
	}
	if err := checkEntryIdentity(admission.Entry); err != nil {
		return err
	}
	if err := checkEntryFeatures(admission.Entry); err != nil {
		return err
	}
	if _, err := record.ParseSHA256(string(admission.CoverageSHA256)); err != nil {
		return errors.New("guided corpus admission coverage identity is invalid")
	}
	if _, err := record.ParseSHA256(string(admission.SnapshotSHA256)); err != nil {
		return errors.New("guided corpus admission snapshot identity is invalid")
	}
	hash := admission.Entry.RecordHash
	if _, live := state.live[hash]; live {
		return fmt.Errorf("guided corpus admission repeats live record %s", hash)
	}
	evicted := make(map[record.SHA256]struct{}, len(admission.Evicted))
	for _, retired := range admission.Evicted {
		if _, live := state.live[retired]; !live {
			return fmt.Errorf("guided corpus admission evicts record %s that is not live", retired)
		}
		if _, repeated := evicted[retired]; repeated {
			return fmt.Errorf("guided corpus admission evicts record %s twice", retired)
		}
		evicted[retired] = struct{}{}
	}
	if len(state.live)-len(evicted) >= state.limits.entries {
		return errors.New("guided corpus entry capacity is exceeded")
	}
	next := maps.Clone(state.live)
	for retired := range evicted {
		delete(next, retired)
	}
	next[hash] = admission.Entry
	generation := state.generation + 1
	snapshot, err := replaySnapshot(*state.identity, generation, next)
	if err != nil {
		return err
	}
	if snapshot.SnapshotSHA256 != admission.SnapshotSHA256 || snapshot.CoverageSHA256 != admission.CoverageSHA256 {
		return fmt.Errorf("%w: guided corpus snapshot identity mismatch", logstore.ErrCorrupt)
	}
	state.live = next
	state.generation = generation
	state.coverage, state.snapshot = admission.CoverageSHA256, admission.SnapshotSHA256
	return nil
}

// result finalizes the replayed live entries and checks them against the
// snapshot identity the latest admission committed.
func (state *replayState) result() (Snapshot, error) {
	snapshot, err := replaySnapshot(*state.identity, state.generation, state.live)
	if err != nil {
		return Snapshot{}, err
	}
	if state.generation != 0 && (snapshot.SnapshotSHA256 != state.snapshot || snapshot.CoverageSHA256 != state.coverage) {
		return Snapshot{}, fmt.Errorf("%w: guided corpus snapshot identity mismatch", logstore.ErrCorrupt)
	}
	return snapshot, nil
}

func replaySnapshot(identity Identity, generation uint64, live map[record.SHA256]Entry) (Snapshot, error) {
	entries := make([]Entry, 0, len(live))
	for _, entry := range live {
		entries = append(entries, entry)
	}
	snapshot, _, err := finalizeSnapshot(Snapshot{Schema: CorpusSchema, Identity: identity, Generation: record.Uint64String(generation), Entries: entries})
	if err != nil {
		return Snapshot{}, err
	}
	return snapshot, nil
}

// Replayed is a read-only view of a corpus log.
type Replayed struct {
	// Snapshot is the replayed live snapshot. Its cases are not opened.
	Snapshot Snapshot
	History  History
	Report   logstore.Report
}

// ReadLog replays the corpus log below root without its writer lock. It
// creates, repairs and removes nothing, and it does not open the cases; Open
// validates those. A torn tail is reported, not repaired.
func ReadLog(root string) (Replayed, error) {
	info, err := os.Lstat(root)
	if err != nil {
		return Replayed{}, fmt.Errorf("inspect guided corpus: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0o700 {
		return Replayed{}, errors.New("guided corpus must be a private directory")
	}
	if err := refuseLegacyLayout(root); err != nil {
		return Replayed{}, err
	}
	state := newReplayState(nil, defaultCapacity)
	report, err := logstore.Replay(filepath.Join(root, StoreDirectory), logstore.Options{Kinds: logKinds}, state.consume)
	if err != nil {
		return Replayed{Report: report}, err
	}
	if !state.started {
		return Replayed{Report: report}, &UnsupportedFormatError{Malformed: errors.New("the corpus log has no committed header")}
	}
	snapshot, err := state.result()
	if err != nil {
		return Replayed{Report: report}, err
	}
	return Replayed{Snapshot: snapshot, History: History{Records: report.Records, Bytes: report.ValidBytes}, Report: report}, nil
}
