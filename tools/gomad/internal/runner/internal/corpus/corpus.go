package corpus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strings"
	"unicode/utf8"

	"go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/evidence/artifact"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	hostfs "go.temporal.io/server/tools/gomad/internal/host/fs"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
	"go.temporal.io/server/tools/gomad/internal/runner/internal/logstore"
)

// Corpus is an open guided corpus: its log writer and the live snapshot
// replayed from it, with a hash index of the live entries.
type Corpus struct {
	ctx         context.Context
	path        string
	identity    Identity
	snapshot    Snapshot
	live        map[record.SHA256]struct{}
	writer      *logstore.Writer
	limits      capacity
	description backends.Description
	evidence    EvidenceProvider
}

type EvidenceProvider interface {
	Evidence(context.Context, backends.EvidenceRequest) (backends.Evidence, error)
}

// openOptions are the seams tests use: a smaller live capacity and log
// fault injection.
type openOptions struct {
	limits capacity
	hook   logstore.Hook
}

// Open takes the corpus's exclusive writer lock, replays its log into the
// live hash index, validates every live case, and reclaims the case files no
// live entry references. A corpus of another storage lineage, a log of
// another identity, and a missing or corrupt live case fail before anything
// is reclaimed.
func Open(ctx context.Context, path string, identity Identity, description backends.Description, evidence EvidenceProvider) (*Corpus, error) {
	return open(ctx, path, identity, description, evidence, openOptions{limits: defaultCapacity})
}

func open(ctx context.Context, path string, identity Identity, description backends.Description, evidence EvidenceProvider, options openOptions) (_ *Corpus, retErr error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if path == "" {
		return nil, errors.New("guided corpus path is required")
	}
	if evidence == nil {
		return nil, errors.New("guided corpus evidence provider is required")
	}
	if err := validateIdentity(identity, description.SemanticInstrumentationSHA256); err != nil {
		return nil, err
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve guided corpus path: %w", err)
	}
	volumeRoot := filepath.VolumeName(absolute) + string(filepath.Separator)
	if filepath.Clean(absolute) == volumeRoot {
		return nil, errors.New("guided corpus cannot use a filesystem root")
	}
	if err := os.MkdirAll(absolute, 0o700); err != nil {
		return nil, fmt.Errorf("create guided corpus: %w", err)
	}
	info, err := os.Lstat(absolute)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.Join(errors.New("guided corpus must be a private directory"), err)
	}
	if err := refuseLegacyLayout(absolute); err != nil {
		return nil, err
	}
	storeExisted, err := preflightExistingLog(absolute, identity, options.limits)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(absolute, 0o700); err != nil {
		return nil, fmt.Errorf("make guided corpus private: %w", err)
	}
	state := newReplayState(&identity, options.limits)
	writer, _, err := logstore.OpenWriter(filepath.Join(absolute, StoreDirectory), logstore.Options{Kinds: logKinds, Create: true, Hook: options.hook}, state.consume)
	if err != nil {
		return nil, openLogError(err)
	}
	corpus := &Corpus{ctx: ctx, path: absolute, identity: identity, writer: writer, limits: options.limits, description: description, evidence: evidence}
	defer func() {
		if retErr != nil {
			retErr = errors.Join(retErr, corpus.Close())
		}
	}()
	if !state.started {
		if storeExisted {
			return nil, &UnsupportedFormatError{Malformed: errors.New("the corpus log has no committed header")}
		}
		if err := writer.AppendJSON(kindHeader, logHeader{Format: LogFormat, Identity: identity}); err != nil {
			return nil, fmt.Errorf("commit guided corpus header: %w", err)
		}
		state.started = true
	}
	snapshot, err := state.result()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(corpus.casesPath(), 0o700); err != nil {
		return nil, fmt.Errorf("create guided corpus case store: %w", err)
	}
	casesInfo, err := os.Lstat(corpus.casesPath())
	if err != nil || !casesInfo.IsDir() || casesInfo.Mode()&os.ModeSymlink != 0 {
		return nil, errors.Join(errors.New("guided corpus case store must be a directory"), err)
	}
	if err := os.Chmod(corpus.casesPath(), 0o700); err != nil {
		return nil, fmt.Errorf("make guided corpus case store private: %w", err)
	}
	if err := corpus.validateLive(snapshot.Entries); err != nil {
		return nil, err
	}
	corpus.setSnapshot(snapshot)
	if err := corpus.cleanupCases(); err != nil {
		return nil, err
	}
	return corpus, nil
}

// openLogError presents the lineage and identity refusals a replay consumer
// raised as themselves rather than as damage at a log offset.
func openLogError(err error) error {
	var unsupported *UnsupportedFormatError
	switch {
	case errors.As(err, &unsupported):
		return unsupported
	case errors.Is(err, errIdentityMismatch):
		return errIdentityMismatch
	case errors.Is(err, logstore.ErrInUse):
		return fmt.Errorf("guided corpus is already in use: %w", err)
	default:
		return fmt.Errorf("open guided corpus log: %w", err)
	}
}

// validateLive checks every replayed live entry against its case on disk
// and the live byte capacity, recording the target each case shares. A
// missing or corrupt case is corruption of the committed state.
func (corpus *Corpus) validateLive(entries []Entry) error {
	var retained artifact.RetainedBytes
	for index, entry := range entries {
		shared, err := corpus.validateEntry(entry)
		if err != nil {
			return &artifact.ReferenceError{Reference: entry.Artifact, Err: err}
		}
		entries[index].sharedTarget = shared
		if _, err := retained.Add(uint64(entry.StoredBytes), shared); err != nil || retained.Total() > corpus.limits.bytes {
			return errors.Join(errors.New("guided corpus byte capacity is exceeded"), err)
		}
	}
	return nil
}

func (corpus *Corpus) setSnapshot(snapshot Snapshot) {
	corpus.snapshot = snapshot
	corpus.live = make(map[record.SHA256]struct{}, len(snapshot.Entries))
	for _, entry := range snapshot.Entries {
		corpus.live[entry.RecordHash] = struct{}{}
	}
}

func (corpus *Corpus) Path() string {
	return corpus.path
}

func (corpus *Corpus) casesPath() string {
	return filepath.Join(corpus.path, "cases")
}

func (corpus *Corpus) targetPool() string {
	return artifact.TargetPool(corpus.path)
}

func (corpus *Corpus) Snapshot() Snapshot {
	return cloneSnapshot(corpus.snapshot)
}

// History reports the committed log; it is zero for a closed corpus.
func (corpus *Corpus) History() History {
	if corpus == nil || corpus.writer == nil {
		return History{}
	}
	return History{Records: corpus.writer.Records(), Bytes: corpus.writer.Offset()}
}

func (corpus *Corpus) Close() error {
	if corpus == nil || corpus.writer == nil {
		return nil
	}
	err := corpus.writer.Close()
	corpus.writer = nil
	return err
}

func (corpus *Corpus) interesting(features []Feature, storedBytes uint64) bool {
	features = canonicalFeatures(features)
	covered := featureSet(corpus.snapshot.Entries)
	for _, feature := range features {
		if _, found := covered[feature]; !found {
			return storedBytes == 0 || storedBytes <= corpus.limits.bytes
		}
	}
	for _, feature := range features {
		if feature.Kind != FeatureFailure {
			continue
		}
		for _, entry := range corpus.snapshot.Entries {
			if containsFeature(entry.Features, feature) && (storedBytes == 0 || storedBytes < uint64(entry.StoredBytes)) {
				return true
			}
		}
	}
	return false
}

// merge commits a replay-verified case: one appended record holds the entry,
// the live entries the bounded selection evicts, and the new snapshot
// identity. Memory advances only after the append is durable, and evicted
// cases are removed only after that.
func (corpus *Corpus) merge(published artifact.Artifact, coverage record.SemanticCoverage, features []Feature, replay ReplayResult) (bool, error) {
	if corpus == nil || corpus.writer == nil {
		return false, errors.New("guided corpus is not open")
	}
	if err := corpus.ctx.Err(); err != nil {
		return false, err
	}
	if !replay.Verified || !replay.Match || replay.Divergence != "" {
		return false, errors.New("guided corpus requires a verified matching replay")
	}
	entry, err := corpus.entryFor(published, coverage, canonicalFeatures(features), replay)
	if err != nil {
		return false, err
	}
	entry.NoveltyReasons = noveltyReasons(entry, corpus.snapshot.Entries)
	if len(entry.NoveltyReasons) == 0 {
		return false, corpus.removeUnreferencedCase(entry.Artifact)
	}
	if _, err := corpus.validateEntry(entry); err != nil {
		return false, err
	}
	pool := append(cloneEntries(corpus.snapshot.Entries), entry)
	selected, err := boundedEntries(pool, corpus.limits)
	if err != nil {
		return false, err
	}
	if !containsRecordHash(selected, entry.RecordHash) {
		return false, corpus.removeUnreferencedCase(entry.Artifact)
	}
	retained := make(map[record.SHA256]struct{}, len(selected))
	for _, kept := range selected {
		retained[kept.RecordHash] = struct{}{}
	}
	var evicted []Entry
	evictedHashes := []record.SHA256{}
	for _, existing := range corpus.snapshot.Entries {
		if _, kept := retained[existing.RecordHash]; !kept {
			evicted = append(evicted, existing)
			evictedHashes = append(evictedHashes, existing.RecordHash)
		}
	}
	next := Snapshot{Schema: CorpusSchema, Identity: corpus.identity, Generation: corpus.snapshot.Generation + 1, Entries: selected}
	next, _, err = finalizeSnapshot(next)
	if err != nil {
		return false, err
	}
	if err := corpus.ctx.Err(); err != nil {
		return false, err
	}
	admission := admissionRecord{Generation: next.Generation, Entry: entry, Evicted: evictedHashes, CoverageSHA256: next.CoverageSHA256, SnapshotSHA256: next.SnapshotSHA256}
	if err := corpus.writer.AppendJSON(kindAdmission, admission); err != nil {
		// The case stays: a complete frame may survive the failed append,
		// and the next Open either replays it or reclaims the case.
		return false, fmt.Errorf("commit guided corpus admission: %w", err)
	}
	corpus.setSnapshot(next)
	return true, corpus.retire(evicted)
}

// retire removes the cases a committed eviction released and the corpus
// targets no remaining case shares.
func (corpus *Corpus) retire(evicted []Entry) error {
	if len(evicted) == 0 {
		return nil
	}
	for _, entry := range evicted {
		if err := os.RemoveAll(filepath.Join(corpus.path, filepath.FromSlash(entry.Artifact))); err != nil {
			return fmt.Errorf("remove evicted guided case %s: %w", entry.Artifact, err)
		}
	}
	if err := hostfs.SyncDirectory(corpus.casesPath()); err != nil {
		return err
	}
	return corpus.pruneTargets()
}

func noveltyReasons(entry Entry, entries []Entry) []Feature {
	covered := featureSet(entries)
	reasons := make([]Feature, 0, len(entry.Features))
	for _, feature := range entry.Features {
		if _, found := covered[feature]; !found {
			reasons = append(reasons, feature)
		}
	}
	if len(reasons) != 0 {
		return canonicalFeatures(reasons)
	}
	for _, feature := range entry.Features {
		if feature.Kind != FeatureFailure {
			continue
		}
		for _, existing := range entries {
			if containsFeature(existing.Features, feature) && entry.PayloadBytes < existing.PayloadBytes && featureSuperset(entry.Features, existing.Features) {
				return []Feature{{Kind: FeatureSmaller, Value: string(existing.RecordHash)}}
			}
		}
	}
	return []Feature{}
}

func containsRecordHash(entries []Entry, hash record.SHA256) bool {
	for _, entry := range entries {
		if entry.RecordHash == hash {
			return true
		}
	}
	return false
}

func (corpus *Corpus) discard(published artifact.Artifact) error {
	if corpus == nil || corpus.writer == nil {
		return errors.New("guided corpus is not open")
	}
	relative, err := filepath.Rel(corpus.path, published.Path)
	if err != nil {
		return err
	}
	relative = filepath.ToSlash(relative)
	if !validCaseReference(relative) {
		return errors.New("guided case is outside the corpus")
	}
	return corpus.removeUnreferencedCase(relative)
}

func (corpus *Corpus) entryFor(published artifact.Artifact, coverage record.SemanticCoverage, features []Feature, replay ReplayResult) (Entry, error) {
	relative, err := filepath.Rel(corpus.path, published.Path)
	if err != nil {
		return Entry{}, fmt.Errorf("make guided case path relative: %w", err)
	}
	relative = filepath.ToSlash(relative)
	if !validCaseReference(relative) {
		return Entry{}, fmt.Errorf("guided case path is outside the corpus: %s", relative)
	}
	manifest := published.Manifest
	if manifest.IOProfile.Transcript == nil {
		return Entry{}, errors.New("guided case requires a complete I/O transcript")
	}
	inputs := CapturedInputs{
		IOTranscriptSHA256: manifest.IOProfile.Transcript.SHA256, IOTranscriptRecords: manifest.IOProfile.Transcript.Records,
		WorldTranscriptSHA256: manifest.World.Transitions.TranscriptDigest, WorldTransitionRecords: manifest.World.Transitions.Count,
		WorldInitialSHA256: manifest.World.Initial.SemanticDigest,
	}
	if manifest.IOProfile.ReadOnlyMounts != nil {
		digest := manifest.IOProfile.ReadOnlyMounts.SHA256
		inputs.ReadOnlyMountsSHA256 = &digest
	}
	payloadBytes, err := artifactPayloadBytes(manifest)
	if err != nil {
		return Entry{}, err
	}
	entry := Entry{
		Seed: manifest.Seed, RecordHash: manifest.RecordHash, Artifact: relative, StoredBytes: record.Uint64String(published.StoredBytes),
		PayloadBytes: record.Uint64String(payloadBytes), Coverage: coverage, Features: features, Inputs: inputs, Replay: replay,
		sharedTarget: artifact.PoolTarget(corpus.targetPool(), published.Path, manifest),
	}
	return entry, nil
}

// validateEntry checks an entry against its case on disk and returns the
// target that case shares with the corpus.
func (corpus *Corpus) validateEntry(entry Entry) (result artifact.SharedTarget, retErr error) {
	if err := checkEntryIdentity(entry); err != nil {
		return artifact.SharedTarget{}, err
	}
	evidence, err := corpus.evidence.Evidence(corpus.ctx, backends.EvidenceRequest{Probes: entry.Coverage.Probes})
	coverage := evidence.Coverage
	if err != nil || coverage.Schema != entry.Coverage.Schema || coverage.Digest != entry.Coverage.Digest || !slices.Equal(coverage.Probes, entry.Coverage.Probes) {
		return artifact.SharedTarget{}, errors.Join(errors.New("guided corpus semantic coverage is invalid"), err)
	}
	if err := checkEntryFeatures(entry); err != nil {
		return artifact.SharedTarget{}, err
	}
	casePath := filepath.Join(corpus.path, filepath.FromSlash(entry.Artifact))
	opened, err := artifact.OpenArtifact(casePath)
	if err != nil {
		return artifact.SharedTarget{}, fmt.Errorf("open guided corpus case: %w", err)
	}
	defer func() {
		if closeErr := opened.Close(); closeErr != nil {
			result = artifact.SharedTarget{}
			if retErr == nil {
				retErr = closeErr
			} else {
				retErr = errors.Join(retErr, closeErr)
			}
		}
	}()
	manifest := opened.Manifest()
	if manifest.RecordHash != entry.RecordHash || manifest.Seed != entry.Seed || opened.StoredBytes() != uint64(entry.StoredBytes) {
		return artifact.SharedTarget{}, errors.New("guided corpus case identity does not match its entry")
	}
	payloadBytes, err := artifactPayloadBytes(manifest)
	if err != nil || payloadBytes != uint64(entry.PayloadBytes) {
		return artifact.SharedTarget{}, errors.Join(errors.New("guided corpus payload size mismatch"), err)
	}
	profile := corpus.description.IOProfile
	if profile.Name != manifest.IOProfile.Name || profile.ImplementationSHA256 != manifest.IOProfile.ImplementationSHA256 || profile.InventorySHA256 != manifest.IOProfile.InventorySHA256 || profile.Inventory != manifest.IOProfile.Inventory {
		return artifact.SharedTarget{}, errors.New("guided corpus case boundary identity does not match this Runner")
	}
	targetIdentity, err := IdentityFor(manifest.Target, manifest.Toolchain, corpus.description, manifest.Environment)
	if corpus.identity.ChoiceProfile != nil {
		if manifest.ChoiceProfile == nil || manifest.ChoiceProfile.Name != corpus.identity.ChoiceProfile.Profile || manifest.ChoiceProfile.ImplementationSHA256 != corpus.identity.ChoiceProfile.ImplementationSHA256 || manifest.ChoiceProfile.Trace.Limit != corpus.identity.ChoiceProfile.Limit {
			return artifact.SharedTarget{}, errors.New("guided corpus case choice profile identity mismatch")
		}
		targetIdentity, err = IdentityForChoice(manifest.Target, manifest.Toolchain, corpus.description, manifest.Environment, *corpus.identity.ChoiceProfile)
	}
	if err != nil || !identitiesEqual(targetIdentity, corpus.identity) {
		return artifact.SharedTarget{}, errors.Join(errors.New("guided corpus case target identity mismatch"), err)
	}
	if manifest.IOProfile.Transcript == nil || manifest.IOProfile.Transcript.SHA256 != entry.Inputs.IOTranscriptSHA256 || manifest.IOProfile.Transcript.Records != entry.Inputs.IOTranscriptRecords || manifest.World.Transitions.TranscriptDigest != entry.Inputs.WorldTranscriptSHA256 || manifest.World.Transitions.Count != entry.Inputs.WorldTransitionRecords || manifest.World.Initial.SemanticDigest != entry.Inputs.WorldInitialSHA256 {
		return artifact.SharedTarget{}, errors.New("guided corpus captured input identity mismatch")
	}
	if mounts := manifest.IOProfile.ReadOnlyMounts; mounts == nil && entry.Inputs.ReadOnlyMountsSHA256 != nil || mounts != nil && (entry.Inputs.ReadOnlyMountsSHA256 == nil || mounts.SHA256 != *entry.Inputs.ReadOnlyMountsSHA256) {
		return artifact.SharedTarget{}, errors.New("guided corpus read-only mount identity mismatch")
	}
	return artifact.PoolTarget(corpus.targetPool(), casePath, manifest), nil
}

// checkEntryIdentity checks the entry fields that need no case or provider.
func checkEntryIdentity(entry Entry) error {
	if _, err := record.ParseSHA256(string(entry.RecordHash)); err != nil || !validCaseReference(entry.Artifact) || entry.StoredBytes == 0 || entry.PayloadBytes == 0 || !entry.Replay.Verified || !entry.Replay.Match || entry.Replay.Divergence != "" {
		return errors.New("guided corpus entry identity is invalid")
	}
	return nil
}

func checkEntryFeatures(entry Entry) error {
	if !featuresEqual(entry.Features, canonicalFeatures(entry.Features)) || len(entry.Features) == 0 || !featuresEqual(entry.NoveltyReasons, canonicalFeatures(entry.NoveltyReasons)) || len(entry.NoveltyReasons) == 0 {
		return errors.New("guided corpus features or novelty reasons are invalid")
	}
	for _, reason := range entry.NoveltyReasons {
		if reason.Kind != FeatureSmaller && !containsFeature(entry.Features, reason) {
			return errors.New("guided corpus novelty reason was not observed")
		}
	}
	return nil
}

func finalizeSnapshot(snapshot Snapshot) (Snapshot, []byte, error) {
	if snapshot.Schema != CorpusSchema {
		return Snapshot{}, nil, errors.New("guided corpus schema is invalid")
	}
	sort.Slice(snapshot.Entries, func(i, j int) bool { return snapshot.Entries[i].RecordHash < snapshot.Entries[j].RecordHash })
	features := make([]Feature, 0)
	for _, entry := range snapshot.Entries {
		features = append(features, entry.Features...)
	}
	if err := validateSnapshotIdentityStrings(snapshot); err != nil {
		return Snapshot{}, nil, err
	}
	//gomad:dynamic reviewed 2026-10-13 stdlib JSON marshals only validated corpus feature identities
	featureBytes, err := json.Marshal(canonicalFeatures(features))
	if err != nil {
		return Snapshot{}, nil, err
	}
	snapshot.CoverageSHA256 = record.DomainHash("gomad3-guide-coverage-v1", featureBytes)
	snapshot.SnapshotSHA256 = ""
	//gomad:dynamic reviewed 2026-10-13 stdlib JSON marshals only validated corpus snapshot identities
	projection, err := json.Marshal(snapshot)
	if err != nil {
		return Snapshot{}, nil, err
	}
	snapshot.SnapshotSHA256 = record.DomainHash("gomad3-guide-snapshot-v1", projection)
	//gomad:dynamic reviewed 2026-10-13 stdlib JSON marshals only validated corpus snapshots
	encoded, err := json.Marshal(snapshot)
	return snapshot, encoded, err
}

// boundedEntries keeps the entries of highest priority that fit the corpus
// caps. The byte cap follows the stored-bytes rule: the target the cases share
// counts once, and each case that shares it adds only its own files.
func boundedEntries(entries []Entry, limits capacity) ([]Entry, error) {
	frequencies := featureFrequencies(entries)
	sort.Slice(entries, func(i, j int) bool { return entryLess(entries[i], entries[j], frequencies) })
	selected := make([]Entry, 0, min(len(entries), limits.entries))
	var retained artifact.RetainedBytes
	for _, entry := range entries {
		cost, err := retained.Cost(uint64(entry.StoredBytes), entry.sharedTarget)
		if err != nil {
			return nil, err
		}
		if len(selected) == limits.entries || cost > limits.bytes-retained.Total() {
			continue
		}
		if _, err := retained.Add(uint64(entry.StoredBytes), entry.sharedTarget); err != nil {
			return nil, err
		}
		selected = append(selected, entry)
	}
	sort.Slice(selected, func(i, j int) bool { return selected[i].RecordHash < selected[j].RecordHash })
	return selected, nil
}

func featureFrequencies(entries []Entry) map[Feature]int {
	result := make(map[Feature]int)
	for _, entry := range entries {
		for _, feature := range entry.Features {
			result[feature]++
		}
	}
	return result
}

func featureSet(entries []Entry) map[Feature]struct{} {
	result := make(map[Feature]struct{})
	for _, entry := range entries {
		for _, feature := range entry.Features {
			result[feature] = struct{}{}
		}
	}
	return result
}

func featureSuperset(left, right []Feature) bool {
	leftSet := make(map[Feature]struct{}, len(left))
	for _, feature := range left {
		leftSet[feature] = struct{}{}
	}
	for _, feature := range right {
		if _, found := leftSet[feature]; !found {
			return false
		}
	}
	return true
}

func containsFeature(features []Feature, want Feature) bool {
	for _, feature := range features {
		if feature == want {
			return true
		}
	}
	return false
}

func featuresEqual(left, right []Feature) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func cloneSnapshot(snapshot Snapshot) Snapshot {
	snapshot.Entries = cloneEntries(snapshot.Entries)
	return snapshot
}

func cloneEntries(entries []Entry) []Entry {
	result := append([]Entry(nil), entries...)
	for index := range result {
		// slices.Clone keeps an empty slice distinct from a nil one, so a
		// cloned entry projects into the snapshot identity as its record does.
		result[index].Coverage.Probes = slices.Clone(result[index].Coverage.Probes)
		result[index].Features = slices.Clone(result[index].Features)
		result[index].NoveltyReasons = slices.Clone(result[index].NoveltyReasons)
		if result[index].Inputs.ReadOnlyMountsSHA256 != nil {
			digest := *result[index].Inputs.ReadOnlyMountsSHA256
			result[index].Inputs.ReadOnlyMountsSHA256 = &digest
		}
	}
	return result
}

func validCaseReference(path string) bool {
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))
	return clean == path && strings.HasPrefix(path, "cases/sha256-") && strings.Count(path, "/") == 1 && !strings.Contains(path, "..")
}

func artifactPayloadBytes(manifest record.ExecutionRecord) (uint64, error) {
	var total uint64
	for _, file := range manifest.Files {
		if uint64(file.Size) > ^uint64(0)-total {
			return 0, errors.New("guided corpus payload size overflows")
		}
		total += uint64(file.Size)
	}
	return total, nil
}

func validateIdentity(identity Identity, instrumentation record.SHA256) error {
	if err := validateStoredIdentity(identity); err != nil {
		return err
	}
	_, instrumentationErr := record.ParseSHA256(string(instrumentation))
	validInstrumentation := instrumentationErr == nil && identity.ChoiceProfile == nil && identity.InstrumentationSchema == SemanticFeatureSchema && identity.InstrumentationSHA256 == semanticInstrumentationIdentity(instrumentation)
	if identity.ChoiceProfile != nil {
		implementation, err := choice.ImplementationIdentity(identity.Toolchain.BuildKey)
		profileIdentity, profileErr := choiceInstrumentationIdentity(*identity.ChoiceProfile, instrumentation)
		validInstrumentation = instrumentationErr == nil && err == nil && profileErr == nil && identity.ChoiceProfile.Profile == choice.Profile && identity.ChoiceProfile.ImplementationSHA256 == record.SHA256FromSum(implementation) && identity.ChoiceProfile.Limit >= choice.MinimumTraceBytes && identity.ChoiceProfile.Limit <= choice.MaximumTraceBytes && identity.InstrumentationSchema == ChoiceFeatureSchema && identity.InstrumentationSHA256 == profileIdentity
	}
	if !validInstrumentation {
		return errors.New("guided corpus identity is invalid")
	}
	return nil
}

func validateStoredIdentity(identity Identity) error {
	if err := validateCorpusIdentityStrings(identity); err != nil {
		return err
	}
	_, targetErr := record.ParseSHA256(string(identity.TargetSHA256))
	_, environmentErr := record.ParseSHA256(string(identity.EnvironmentSHA256))
	_, boundaryErr := record.ParseSHA256(string(identity.BoundarySHA256))
	_, instrumentationErr := record.ParseSHA256(string(identity.InstrumentationSHA256))
	validInstrumentation := instrumentationErr == nil && identity.ChoiceProfile == nil && identity.InstrumentationSchema == SemanticFeatureSchema
	if identity.ChoiceProfile != nil {
		implementation, err := choice.ImplementationIdentity(identity.Toolchain.BuildKey)
		validInstrumentation = instrumentationErr == nil && err == nil && identity.ChoiceProfile.Profile == choice.Profile && identity.ChoiceProfile.ImplementationSHA256 == record.SHA256FromSum(implementation) && identity.ChoiceProfile.Limit >= choice.MinimumTraceBytes && identity.ChoiceProfile.Limit <= choice.MaximumTraceBytes && identity.InstrumentationSchema == ChoiceFeatureSchema
	}
	if targetErr != nil || environmentErr != nil || identity.Toolchain.GoVersion == "" || identity.Toolchain.BuildKey == "" || identity.Toolchain.TargetGOOS == "" || identity.Toolchain.TargetGOARCH == "" || identity.BoundaryVersion == "" || boundaryErr != nil || !validInstrumentation || identity.ManifestSchemaVersion != record.SchemaVersion || identity.ManifestRecordContract != record.RecordContract {
		return errors.New("guided corpus identity is invalid")
	}
	return nil
}

func identitiesEqual(left, right Identity) bool {
	return reflect.DeepEqual(left, right)
}

func validateCorpusIdentityStrings(identity Identity) error {
	values := []string{
		string(identity.EnvironmentSHA256), string(identity.TargetSHA256),
		identity.Toolchain.GoVersion, identity.Toolchain.BuildKey, identity.Toolchain.TargetGOOS, identity.Toolchain.TargetGOARCH,
		identity.BoundaryVersion, string(identity.BoundarySHA256), identity.InstrumentationSchema, string(identity.InstrumentationSHA256), identity.ManifestRecordContract,
	}
	if identity.ChoiceProfile != nil {
		values = append(values, identity.ChoiceProfile.Profile, string(identity.ChoiceProfile.ImplementationSHA256))
	}
	for _, value := range values {
		if !utf8.ValidString(value) {
			return errors.New("guided corpus identity contains invalid UTF-8")
		}
	}
	return nil
}

func validateSnapshotIdentityStrings(snapshot Snapshot) error {
	if !utf8.ValidString(snapshot.Schema) {
		return errors.New("guided corpus snapshot contains invalid UTF-8")
	}
	if err := validateCorpusIdentityStrings(snapshot.Identity); err != nil {
		return err
	}
	values := []string{string(snapshot.CoverageSHA256), string(snapshot.SnapshotSHA256)}
	for _, entry := range snapshot.Entries {
		values = append(values,
			string(entry.RecordHash), entry.Artifact, entry.Coverage.Schema, string(entry.Coverage.Digest),
			string(entry.Inputs.IOTranscriptSHA256), string(entry.Inputs.WorldTranscriptSHA256), string(entry.Inputs.WorldInitialSHA256), entry.Replay.Divergence,
		)
		if entry.Inputs.ReadOnlyMountsSHA256 != nil {
			values = append(values, string(*entry.Inputs.ReadOnlyMountsSHA256))
		}
		values = append(values, entry.Coverage.Probes...)
		for _, feature := range entry.Features {
			values = append(values, feature.Kind, feature.Value)
		}
		for _, feature := range entry.NoveltyReasons {
			values = append(values, feature.Kind, feature.Value)
		}
	}
	for _, value := range values {
		if !utf8.ValidString(value) {
			return errors.New("guided corpus snapshot contains invalid UTF-8")
		}
	}
	return nil
}

// cleanupCases reclaims the case directories and publication leftovers no
// live entry references. SweepOwnedRoot validates every live case first and
// removes nothing when one is missing or corrupt.
func (corpus *Corpus) cleanupCases() error {
	live := artifact.References{Artifacts: make([]string, 0, len(corpus.snapshot.Entries))}
	for _, entry := range corpus.snapshot.Entries {
		live.Artifacts = append(live.Artifacts, path.Base(entry.Artifact))
	}
	if _, err := artifact.SweepOwnedRoot(corpus.casesPath(), live); err != nil {
		return fmt.Errorf("reclaim unreferenced guided cases: %w", err)
	}
	return corpus.pruneTargets()
}

// pruneTargets drops the corpus target no remaining case shares. The corpus
// lock makes this process the pool's only writer.
func (corpus *Corpus) pruneTargets() error {
	if err := artifact.PruneTargetPool(corpus.targetPool(), true); err != nil {
		return fmt.Errorf("prune guided corpus targets: %w", err)
	}
	return nil
}

func (corpus *Corpus) removeUnreferencedCase(relative string) error {
	for _, entry := range corpus.snapshot.Entries {
		if entry.Artifact == relative {
			return nil
		}
	}
	if err := os.RemoveAll(filepath.Join(corpus.path, filepath.FromSlash(relative))); err != nil {
		return err
	}
	if err := hostfs.SyncDirectory(corpus.casesPath()); err != nil {
		return err
	}
	return corpus.pruneTargets()
}
