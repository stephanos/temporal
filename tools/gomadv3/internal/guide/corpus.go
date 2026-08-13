package guide

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"go.temporal.io/server/tools/gomadv3/internal/artifact"
	"go.temporal.io/server/tools/gomadv3/internal/ioprofile"
	"go.temporal.io/server/tools/gomadv3/internal/record"
	"go.temporal.io/server/tools/gomadv3/internal/safefile"
)

const maximumCorpusJSONBytes = 16 << 20

type Corpus struct {
	ctx      context.Context
	path     string
	identity Identity
	snapshot Snapshot
	lock     *os.File
}

func Open(ctx context.Context, path string, identity Identity) (_ *Corpus, retErr error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if path == "" {
		return nil, errors.New("guided corpus path is required")
	}
	if err := validateIdentity(identity); err != nil {
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
	if err := os.Chmod(absolute, 0o700); err != nil {
		return nil, fmt.Errorf("make guided corpus private: %w", err)
	}
	lock, err := acquireLock(filepath.Join(absolute, "corpus.lock"))
	if err != nil {
		return nil, err
	}
	corpus := &Corpus{ctx: ctx, path: absolute, identity: identity, lock: lock}
	defer func() {
		if retErr != nil {
			retErr = errors.Join(retErr, corpus.Close())
		}
	}()
	if err := os.MkdirAll(corpus.CasesPath(), 0o700); err != nil {
		return nil, fmt.Errorf("create guided corpus case store: %w", err)
	}
	casesInfo, err := os.Lstat(corpus.CasesPath())
	if err != nil || !casesInfo.IsDir() || casesInfo.Mode()&os.ModeSymlink != 0 {
		return nil, errors.Join(errors.New("guided corpus case store must be a directory"), err)
	}
	if err := os.Chmod(corpus.CasesPath(), 0o700); err != nil {
		return nil, fmt.Errorf("make guided corpus case store private: %w", err)
	}
	corpus.snapshot, err = corpus.readSnapshot()
	if err != nil {
		return nil, err
	}
	if err := corpus.cleanupCases(); err != nil {
		return nil, err
	}
	return corpus, nil
}

func (corpus *Corpus) Path() string {
	return corpus.path
}

func (corpus *Corpus) CasesPath() string {
	return filepath.Join(corpus.path, "cases")
}

func (corpus *Corpus) Snapshot() Snapshot {
	return cloneSnapshot(corpus.snapshot)
}

func (corpus *Corpus) Close() error {
	if corpus == nil || corpus.lock == nil {
		return nil
	}
	err := releaseLock(corpus.lock)
	corpus.lock = nil
	return err
}

func (corpus *Corpus) Interesting(features []Feature, storedBytes uint64) bool {
	features = canonicalFeatures(features)
	covered := featureSet(corpus.snapshot.Entries)
	for _, feature := range features {
		if _, found := covered[feature]; !found {
			return storedBytes == 0 || storedBytes <= MaximumBytes
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

func (corpus *Corpus) Merge(published artifact.Artifact, coverage ioprofile.SemanticCoverage, features []Feature, replay ReplayResult) (bool, error) {
	if corpus == nil || corpus.lock == nil {
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
	covered := featureSet(corpus.snapshot.Entries)
	for _, feature := range entry.Features {
		if _, found := covered[feature]; !found {
			entry.NoveltyReasons = append(entry.NoveltyReasons, feature)
		}
	}
	if len(entry.NoveltyReasons) == 0 {
		for _, feature := range entry.Features {
			if feature.Kind != FeatureFailure {
				continue
			}
			for _, existing := range corpus.snapshot.Entries {
				if containsFeature(existing.Features, feature) && entry.PayloadBytes < existing.PayloadBytes && featureSuperset(entry.Features, existing.Features) {
					entry.NoveltyReasons = []Feature{{Kind: FeatureSmaller, Value: string(existing.RecordHash)}}
					break
				}
			}
		}
	}
	if len(entry.NoveltyReasons) == 0 {
		return false, corpus.removeUnreferencedCase(entry.Artifact)
	}
	entry.NoveltyReasons = canonicalFeatures(entry.NoveltyReasons)
	if err := corpus.validateEntry(entry); err != nil {
		return false, err
	}
	pool := append(cloneEntries(corpus.snapshot.Entries), entry)
	selected := boundedEntries(pool)
	retained := false
	for _, candidate := range selected {
		if candidate.RecordHash == entry.RecordHash {
			retained = true
			break
		}
	}
	if !retained {
		return false, corpus.removeUnreferencedCase(entry.Artifact)
	}
	next := Snapshot{Schema: CorpusSchema, Identity: corpus.identity, Generation: corpus.snapshot.Generation + 1, Entries: selected}
	next, encoded, err := finalizeSnapshot(next)
	if err != nil {
		return false, err
	}
	if err := writeAtomic(corpus.ctx, filepath.Join(corpus.path, "corpus.json"), encoded); err != nil {
		return false, fmt.Errorf("publish guided corpus snapshot: %w", err)
	}
	corpus.snapshot = next
	if err := corpus.cleanupCases(); err != nil {
		return true, err
	}
	return true, nil
}

func (corpus *Corpus) Discard(published artifact.Artifact) error {
	if corpus == nil || corpus.lock == nil {
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

func (corpus *Corpus) entryFor(published artifact.Artifact, coverage ioprofile.SemanticCoverage, features []Feature, replay ReplayResult) (Entry, error) {
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
	}
	return entry, nil
}

func (corpus *Corpus) readSnapshot() (Snapshot, error) {
	path := filepath.Join(corpus.path, "corpus.json")
	file, info, err := safefile.OpenPath(path)
	if os.IsNotExist(err) {
		snapshot, _, finalizeErr := finalizeSnapshot(Snapshot{Schema: CorpusSchema, Identity: corpus.identity, Entries: []Entry{}})
		return snapshot, finalizeErr
	}
	if err != nil {
		return Snapshot{}, fmt.Errorf("open guided corpus snapshot: %w", err)
	}
	defer file.Close()
	if info.Mode().Perm() != 0o600 || info.Size() > maximumCorpusJSONBytes {
		return Snapshot{}, errors.New("guided corpus snapshot mode or size is invalid")
	}
	contents, err := io.ReadAll(io.LimitReader(file, maximumCorpusJSONBytes+1))
	if err != nil {
		return Snapshot{}, fmt.Errorf("read guided corpus snapshot: %w", err)
	}
	var snapshot Snapshot
	if err := record.DecodeCanonicalJSON(contents, &snapshot); err != nil {
		return Snapshot{}, fmt.Errorf("decode guided corpus snapshot: %w", err)
	}
	finalized, _, err := finalizeSnapshot(snapshot)
	if err != nil {
		return Snapshot{}, err
	}
	if finalized.SnapshotSHA256 != snapshot.SnapshotSHA256 || finalized.CoverageSHA256 != snapshot.CoverageSHA256 {
		return Snapshot{}, errors.New("guided corpus snapshot identity mismatch")
	}
	if snapshot.Identity != corpus.identity {
		return Snapshot{}, errors.New("guided corpus identity does not match the prepared target and toolchain")
	}
	if len(snapshot.Entries) > MaximumEntries {
		return Snapshot{}, errors.New("guided corpus entry capacity is exceeded")
	}
	var total uint64
	for _, entry := range snapshot.Entries {
		if err := corpus.validateEntry(entry); err != nil {
			return Snapshot{}, err
		}
		if uint64(entry.StoredBytes) > MaximumBytes-total {
			return Snapshot{}, errors.New("guided corpus byte capacity is exceeded")
		}
		total += uint64(entry.StoredBytes)
	}
	return snapshot, nil
}

func (corpus *Corpus) validateEntry(entry Entry) error {
	if _, err := record.ParseSHA256(string(entry.RecordHash)); err != nil || !validCaseReference(entry.Artifact) || entry.StoredBytes == 0 || entry.PayloadBytes == 0 || !entry.Replay.Verified || !entry.Replay.Match || entry.Replay.Divergence != "" {
		return errors.New("guided corpus entry identity is invalid")
	}
	coverage, err := ioprofile.SummarizeSemanticProbes(entry.Coverage.Probes)
	if err != nil || coverage.Schema != entry.Coverage.Schema || coverage.Digest != entry.Coverage.Digest || !slices.Equal(coverage.Probes, entry.Coverage.Probes) {
		return errors.Join(errors.New("guided corpus semantic coverage is invalid"), err)
	}
	if !featuresEqual(entry.Features, canonicalFeatures(entry.Features)) || len(entry.Features) == 0 || !featuresEqual(entry.NoveltyReasons, canonicalFeatures(entry.NoveltyReasons)) || len(entry.NoveltyReasons) == 0 {
		return errors.New("guided corpus features or novelty reasons are invalid")
	}
	for _, reason := range entry.NoveltyReasons {
		if reason.Kind != FeatureSmaller && !containsFeature(entry.Features, reason) {
			return errors.New("guided corpus novelty reason was not observed")
		}
	}
	opened, err := artifact.Open(filepath.Join(corpus.path, filepath.FromSlash(entry.Artifact)))
	if err != nil {
		return fmt.Errorf("open guided corpus case: %w", err)
	}
	defer opened.Close()
	manifest := opened.Manifest
	if manifest.RecordHash != entry.RecordHash || manifest.Seed != entry.Seed || opened.StoredBytes != uint64(entry.StoredBytes) {
		return errors.New("guided corpus case identity does not match its entry")
	}
	payloadBytes, err := artifactPayloadBytes(manifest)
	if err != nil || payloadBytes != uint64(entry.PayloadBytes) {
		return errors.Join(errors.New("guided corpus payload size mismatch"), err)
	}
	profile := ioprofile.Default()
	if manifest.IOProfile.Name != profile.Name() || manifest.IOProfile.ImplementationSHA256 != profile.ImplementationSHA256() || manifest.IOProfile.InventorySHA256 != profile.InventorySHA256() {
		return errors.New("guided corpus case boundary identity does not match this Runner")
	}
	targetIdentity, err := IdentityFor(manifest.Target, manifest.Toolchain, corpus.identity.BoundaryVersion, corpus.identity.BoundarySHA256)
	if err != nil || targetIdentity != corpus.identity {
		return errors.Join(errors.New("guided corpus case target identity mismatch"), err)
	}
	if manifest.IOProfile.Transcript == nil || manifest.IOProfile.Transcript.SHA256 != entry.Inputs.IOTranscriptSHA256 || manifest.IOProfile.Transcript.Records != entry.Inputs.IOTranscriptRecords || manifest.World.Transitions.TranscriptDigest != entry.Inputs.WorldTranscriptSHA256 || manifest.World.Transitions.Count != entry.Inputs.WorldTransitionRecords || manifest.World.Initial.SemanticDigest != entry.Inputs.WorldInitialSHA256 {
		return errors.New("guided corpus captured input identity mismatch")
	}
	if mounts := manifest.IOProfile.ReadOnlyMounts; mounts == nil && entry.Inputs.ReadOnlyMountsSHA256 != nil || mounts != nil && (entry.Inputs.ReadOnlyMountsSHA256 == nil || mounts.SHA256 != *entry.Inputs.ReadOnlyMountsSHA256) {
		return errors.New("guided corpus read-only mount identity mismatch")
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
	featureBytes, err := record.CanonicalJSON(canonicalFeatures(features))
	if err != nil {
		return Snapshot{}, nil, err
	}
	snapshot.CoverageSHA256 = record.DomainHash("gomadv3-guide-coverage-v1", featureBytes)
	snapshot.SnapshotSHA256 = ""
	projection, err := record.CanonicalJSON(snapshot)
	if err != nil {
		return Snapshot{}, nil, err
	}
	snapshot.SnapshotSHA256 = record.DomainHash("gomadv3-guide-snapshot-v1", projection)
	encoded, err := record.CanonicalJSON(snapshot)
	return snapshot, encoded, err
}

func boundedEntries(entries []Entry) []Entry {
	frequencies := featureFrequencies(entries)
	sort.Slice(entries, func(i, j int) bool { return entryLess(entries[i], entries[j], frequencies) })
	selected := make([]Entry, 0, min(len(entries), MaximumEntries))
	var bytes uint64
	for _, entry := range entries {
		if len(selected) == MaximumEntries || uint64(entry.StoredBytes) > MaximumBytes-bytes {
			continue
		}
		selected = append(selected, entry)
		bytes += uint64(entry.StoredBytes)
	}
	sort.Slice(selected, func(i, j int) bool { return selected[i].RecordHash < selected[j].RecordHash })
	return selected
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
		result[index].Coverage.Probes = append([]string(nil), result[index].Coverage.Probes...)
		result[index].Features = append([]Feature(nil), result[index].Features...)
		result[index].NoveltyReasons = append([]Feature(nil), result[index].NoveltyReasons...)
		if result[index].Inputs.ReadOnlyMountsSHA256 != nil {
			digest := *result[index].Inputs.ReadOnlyMountsSHA256
			result[index].Inputs.ReadOnlyMountsSHA256 = &digest
		}
	}
	return result
}

func validCaseReference(path string) bool {
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))
	return clean == path && strings.HasPrefix(path, "cases/sha256-") && !strings.Contains(path, "..")
}

func artifactPayloadBytes(manifest record.Manifest) (uint64, error) {
	var total uint64
	for _, file := range manifest.Files {
		if uint64(file.Size) > ^uint64(0)-total {
			return 0, errors.New("guided corpus payload size overflows")
		}
		total += uint64(file.Size)
	}
	return total, nil
}

func validateIdentity(identity Identity) error {
	_, targetErr := record.ParseSHA256(string(identity.TargetSHA256))
	_, boundaryErr := record.ParseSHA256(string(identity.BoundarySHA256))
	if targetErr != nil || identity.Toolchain.GoVersion == "" || identity.Toolchain.BuildKey == "" || identity.Toolchain.TargetGOOS == "" || identity.Toolchain.TargetGOARCH == "" || identity.BoundaryVersion == "" || boundaryErr != nil || identity.InstrumentationSchema != SemanticFeatureSchema || identity.InstrumentationSHA256 != semanticInstrumentationIdentity() || identity.ManifestSchemaVersion != record.SchemaVersion || identity.ManifestRecordContract != record.RecordContract {
		return errors.New("guided corpus identity is invalid")
	}
	return nil
}

func (corpus *Corpus) cleanupCases() error {
	referenced := make(map[string]struct{}, len(corpus.snapshot.Entries))
	for _, entry := range corpus.snapshot.Entries {
		referenced[filepath.Base(entry.Artifact)] = struct{}{}
	}
	entries, err := os.ReadDir(corpus.CasesPath())
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if _, found := referenced[entry.Name()]; found {
			continue
		}
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "sha256-") && !strings.HasPrefix(entry.Name(), ".publish-") {
			return fmt.Errorf("guided corpus contains unexpected case entry %s", entry.Name())
		}
		if err := os.RemoveAll(filepath.Join(corpus.CasesPath(), entry.Name())); err != nil {
			return fmt.Errorf("remove unreferenced guided case %s: %w", entry.Name(), err)
		}
	}
	return syncDirectory(corpus.CasesPath())
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
	return syncDirectory(corpus.CasesPath())
}

func writeAtomic(ctx context.Context, path string, contents []byte) (retErr error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".corpus-")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer func() {
		if removeErr := os.Remove(temporaryPath); removeErr != nil && !os.IsNotExist(removeErr) {
			retErr = errors.Join(retErr, removeErr)
		}
	}()
	if err := temporary.Chmod(0o600); err != nil {
		return errors.Join(err, temporary.Close())
	}
	if _, err := temporary.Write(contents); err != nil {
		return errors.Join(err, temporary.Close())
	}
	if err := temporary.Sync(); err != nil {
		return errors.Join(err, temporary.Close())
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return err
	}
	return syncDirectory(filepath.Dir(path))
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	return errors.Join(directory.Sync(), directory.Close())
}
