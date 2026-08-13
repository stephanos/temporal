package guide

import (
	"fmt"
	"sort"

	"go.temporal.io/server/tools/gomadv3/internal/ioprofile"
	"go.temporal.io/server/tools/gomadv3/internal/record"
)

const (
	CorpusSchema          = "gomadv3.guide-corpus/v1"
	SemanticFeatureSchema = "gomadv3.semantic-features/v1"

	FeatureFailure       = "failure"
	FeatureInvariant     = "invariant"
	FeatureTerminal      = "terminal"
	FeatureOutcome       = "outcome"
	FeatureWorld         = "world"
	FeatureIOOutcome     = "io_outcome"
	FeatureOperationPair = "operation_pair"
	FeatureBoundaryProbe = "boundary_probe"
	FeatureCodeEdge      = "code_edge"
	FeatureSmaller       = "smaller_reproduction"
)

const (
	MaximumEntries = 1024
	MaximumBytes   = 1 << 30
)

type Identity struct {
	TargetSHA256           record.SHA256    `json:"target_sha256"`
	Toolchain              record.Toolchain `json:"toolchain"`
	BoundaryVersion        string           `json:"boundary_version"`
	BoundarySHA256         record.SHA256    `json:"boundary_sha256"`
	InstrumentationSchema  string           `json:"instrumentation_schema"`
	InstrumentationSHA256  record.SHA256    `json:"instrumentation_sha256"`
	ManifestSchemaVersion  uint32           `json:"manifest_schema_version"`
	ManifestRecordContract string           `json:"manifest_record_contract"`
}

type Feature struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

type ReplayResult struct {
	Verified   bool   `json:"verified"`
	Match      bool   `json:"match"`
	Diagnostic bool   `json:"diagnostic"`
	Divergence string `json:"divergence,omitempty"`
}

type CapturedInputs struct {
	IOTranscriptSHA256     record.SHA256       `json:"io_transcript_sha256"`
	IOTranscriptRecords    record.Uint64String `json:"io_transcript_records"`
	WorldTranscriptSHA256  record.SHA256       `json:"world_transcript_sha256"`
	WorldTransitionRecords record.Uint64String `json:"world_transition_records"`
	WorldInitialSHA256     record.SHA256       `json:"world_initial_sha256"`
	ReadOnlyMountsSHA256   *record.SHA256      `json:"read_only_mounts_sha256,omitempty"`
}

type Entry struct {
	Seed           record.Uint64String        `json:"seed"`
	RecordHash     record.SHA256              `json:"record_hash"`
	Artifact       string                     `json:"artifact"`
	StoredBytes    record.Uint64String        `json:"stored_bytes"`
	PayloadBytes   record.Uint64String        `json:"payload_bytes"`
	Coverage       ioprofile.SemanticCoverage `json:"semantic_coverage"`
	Features       []Feature                  `json:"features"`
	NoveltyReasons []Feature                  `json:"novelty_reasons"`
	Inputs         CapturedInputs             `json:"captured_inputs"`
	Replay         ReplayResult               `json:"replay"`
}

type Snapshot struct {
	Schema         string              `json:"schema"`
	Identity       Identity            `json:"identity"`
	Generation     record.Uint64String `json:"generation"`
	Entries        []Entry             `json:"entries"`
	CoverageSHA256 record.SHA256       `json:"coverage_sha256"`
	SnapshotSHA256 record.SHA256       `json:"snapshot_sha256"`
}

type targetProjection struct {
	Kind          string                     `json:"kind"`
	SHA256        record.SHA256              `json:"sha256"`
	Size          record.Uint64String        `json:"size"`
	Argv          []string                   `json:"argv"`
	BuildTags     []string                   `json:"build_tags"`
	Adapters      []record.TargetAdapter     `json:"adapters"`
	Compatibility []record.CompatibilityPack `json:"compatibility"`
	BuildInfo     record.BuildInfo           `json:"build_info"`
}

func IdentityFor(target record.Target, toolchain record.Toolchain, boundaryVersion string, boundarySHA256 record.SHA256) (Identity, error) {
	projected := targetProjection{
		Kind: target.Kind, SHA256: target.SHA256, Size: target.Size, Argv: append([]string(nil), target.Argv...), BuildTags: append([]string(nil), target.BuildTags...),
		Adapters: append([]record.TargetAdapter(nil), target.Adapters...), Compatibility: append([]record.CompatibilityPack(nil), target.Compatibility...), BuildInfo: target.BuildInfo,
	}
	encoded, err := record.CanonicalJSON(projected)
	if err != nil {
		return Identity{}, fmt.Errorf("encode guided target identity: %w", err)
	}
	return Identity{
		TargetSHA256: record.DomainHash("gomadv3-guide-target-v1", encoded), Toolchain: toolchain,
		BoundaryVersion: boundaryVersion, BoundarySHA256: boundarySHA256,
		InstrumentationSchema: SemanticFeatureSchema, InstrumentationSHA256: semanticInstrumentationIdentity(),
		ManifestSchemaVersion: record.SchemaVersion, ManifestRecordContract: record.RecordContract,
	}, nil
}

func semanticInstrumentationIdentity() record.SHA256 {
	identity := SemanticFeatureSchema + "\x00" + ioprofile.SemanticCoverageSchema + "\x00" + string(ioprofile.SemanticInstrumentationIdentity()) + "\x00"
	return record.DomainHash("gomadv3-guide-instrumentation-v1", []byte(identity))
}

func (snapshot Snapshot) PrioritizedSeeds() []uint64 {
	frequencies := make(map[Feature]int)
	for _, entry := range snapshot.Entries {
		for _, feature := range entry.Features {
			frequencies[feature]++
		}
	}
	entries := append([]Entry(nil), snapshot.Entries...)
	sort.Slice(entries, func(i, j int) bool {
		return entryLess(entries[i], entries[j], frequencies)
	})
	seeds := make([]uint64, 0, len(entries))
	seen := make(map[uint64]struct{}, len(entries))
	for _, entry := range entries {
		seed := uint64(entry.Seed)
		if _, found := seen[seed]; found {
			continue
		}
		seen[seed] = struct{}{}
		seeds = append(seeds, seed)
	}
	return seeds
}

func entryLess(left, right Entry, frequencies map[Feature]int) bool {
	for rank := 0; rank <= 5; rank++ {
		leftFrequency := rarestFeatureFrequency(left.Features, rank, frequencies)
		rightFrequency := rarestFeatureFrequency(right.Features, rank, frequencies)
		if leftFrequency != rightFrequency {
			return leftFrequency < rightFrequency
		}
	}
	if left.PayloadBytes != right.PayloadBytes {
		return left.PayloadBytes < right.PayloadBytes
	}
	if left.Seed != right.Seed {
		return left.Seed < right.Seed
	}
	return left.RecordHash < right.RecordHash
}

func rarestFeatureFrequency(features []Feature, rank int, frequencies map[Feature]int) int {
	minimum := int(^uint(0) >> 1)
	for _, feature := range features {
		if featureRank(feature.Kind)/10 == rank && frequencies[feature] < minimum {
			minimum = frequencies[feature]
		}
	}
	return minimum
}

func featureRank(kind string) int {
	switch kind {
	case FeatureFailure:
		return 0
	case FeatureInvariant:
		return 10
	case FeatureTerminal:
		return 11
	case FeatureOutcome:
		return 20
	case FeatureWorld:
		return 21
	case FeatureIOOutcome:
		return 22
	case FeatureOperationPair:
		return 30
	case FeatureBoundaryProbe:
		return 40
	case FeatureCodeEdge:
		return 50
	case FeatureSmaller:
		return 60
	default:
		return 100
	}
}
