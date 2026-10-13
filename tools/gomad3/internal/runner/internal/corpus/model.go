package corpus

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"unicode/utf8"

	"go.temporal.io/server/tools/gomad3/internal/backends"
	"go.temporal.io/server/tools/gomad3/internal/evidence/artifact"
	"go.temporal.io/server/tools/gomad3/internal/evidence/record"
	"go.temporal.io/server/tools/gomad3/internal/model/choice"
)

const (
	CorpusSchema          = "gomad3.guide-corpus/v2"
	SemanticFeatureSchema = "gomad3.semantic-features/v1"
	ChoiceFeatureSchema   = "gomad3.semantic-choice-features/v1"

	FeatureFailure       = "failure"
	FeatureInvariant     = "invariant"
	FeatureTerminal      = "terminal"
	FeatureOutcome       = "outcome"
	FeatureWorld         = "world"
	FeatureIOOutcome     = "io_outcome"
	FeatureOperationPair = "operation_pair"
	FeatureBoundaryProbe = "boundary_probe"
	FeatureChoice        = "choice"
	FeatureCodeEdge      = "code_edge"
	FeatureSmaller       = "smaller_reproduction"
)

const (
	maximumEntries = 1024
	maximumBytes   = 1 << 30
)

// capacity bounds the live corpus: its entry count and its retained bytes
// under the stored-bytes rule. Historical log bytes are not part of it.
type capacity struct {
	entries int
	bytes   uint64
}

var defaultCapacity = capacity{entries: maximumEntries, bytes: maximumBytes}

type Identity struct {
	EnvironmentSHA256      record.SHA256          `json:"environment_sha256"`
	TargetSHA256           record.SHA256          `json:"target_sha256"`
	Toolchain              record.Toolchain       `json:"toolchain"`
	BoundaryVersion        string                 `json:"boundary_version"`
	BoundarySHA256         record.SHA256          `json:"boundary_sha256"`
	InstrumentationSchema  string                 `json:"instrumentation_schema"`
	InstrumentationSHA256  record.SHA256          `json:"instrumentation_sha256"`
	ManifestSchemaVersion  uint32                 `json:"manifest_schema_version"`
	ManifestRecordContract string                 `json:"manifest_record_contract"`
	ChoiceProfile          *ChoiceProfileIdentity `json:"choice_profile,omitempty"`
}

type ChoiceProfileIdentity struct {
	Profile              string              `json:"profile"`
	ImplementationSHA256 record.SHA256       `json:"implementation_sha256"`
	Limit                record.Uint64String `json:"limit"`
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
	Seed           record.Uint64String     `json:"seed"`
	RecordHash     record.SHA256           `json:"record_hash"`
	Artifact       string                  `json:"artifact"`
	StoredBytes    record.Uint64String     `json:"stored_bytes"`
	PayloadBytes   record.Uint64String     `json:"payload_bytes"`
	Coverage       record.SemanticCoverage `json:"semantic_coverage"`
	Features       []Feature               `json:"features"`
	NoveltyReasons []Feature               `json:"novelty_reasons"`
	Inputs         CapturedInputs          `json:"captured_inputs"`
	Replay         ReplayResult            `json:"replay"`
	// sharedTarget is the part of StoredBytes the case has in common with the
	// other cases of its corpus. It is read from the case on disk, so a case
	// that owns a private copy of the target counts in full against the cap.
	sharedTarget artifact.SharedTarget
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

func IdentityFor(target record.Target, toolchain record.Toolchain, description backends.Description, environment []record.Environment) (Identity, error) {
	return identityFor(target, toolchain, description, environment, nil)
}

func IdentityForChoice(target record.Target, toolchain record.Toolchain, description backends.Description, environment []record.Environment, choiceProfile ChoiceProfileIdentity) (Identity, error) {
	implementation, err := choice.ImplementationIdentity(toolchain.BuildKey)
	if err != nil || choiceProfile.Profile != choice.Profile || choiceProfile.ImplementationSHA256 != record.SHA256FromSum(implementation) || choiceProfile.Limit < choice.MinimumTraceBytes || choiceProfile.Limit > choice.MaximumTraceBytes {
		return Identity{}, errors.New("guided choice profile identity is invalid")
	}
	return identityFor(target, toolchain, description, environment, &choiceProfile)
}

func identityFor(target record.Target, toolchain record.Toolchain, description backends.Description, environment []record.Environment, choice *ChoiceProfileIdentity) (Identity, error) {
	projected := targetProjection{
		Kind: target.Kind, SHA256: target.SHA256, Size: target.Size, Argv: append([]string(nil), target.Argv...), BuildTags: append([]string(nil), target.BuildTags...),
		Adapters: append([]record.TargetAdapter(nil), target.Adapters...), Compatibility: append([]record.CompatibilityPack(nil), target.Compatibility...), BuildInfo: target.BuildInfo,
	}
	baseEnvironment := make([]record.Environment, 0, len(environment))
	for _, entry := range environment {
		switch entry.Name {
		case "GOMADSEED", "GOMAD3_IO_PROFILE", "GOMAD3_CHOICE_PROFILE":
			continue
		}
		baseEnvironment = append(baseEnvironment, entry)
	}
	if err := validateIdentityInputStrings(projected, toolchain, description, baseEnvironment, choice); err != nil {
		return Identity{}, err
	}
	//gomad:dynamic reviewed 2026-10-13 stdlib JSON marshals only validated corpus target identity inputs
	encoded, err := json.Marshal(projected)
	if err != nil {
		return Identity{}, fmt.Errorf("encode guided target identity: %w", err)
	}
	sort.Slice(baseEnvironment, func(i, j int) bool { return baseEnvironment[i].Name < baseEnvironment[j].Name })
	//gomad:dynamic reviewed 2026-10-13 stdlib JSON marshals only validated corpus environment identity inputs
	environmentBytes, err := json.Marshal(baseEnvironment)
	if err != nil {
		return Identity{}, fmt.Errorf("encode guided environment identity: %w", err)
	}
	result := Identity{
		EnvironmentSHA256: record.DomainHash("gomad3-guide-environment-v1", environmentBytes),
		TargetSHA256:      record.DomainHash("gomad3-guide-target-v1", encoded), Toolchain: toolchain,
		BoundaryVersion: description.BoundaryVersion, BoundarySHA256: description.BoundarySHA256,
		InstrumentationSchema: SemanticFeatureSchema, InstrumentationSHA256: semanticInstrumentationIdentity(description.SemanticInstrumentationSHA256),
		ManifestSchemaVersion: record.SchemaVersion, ManifestRecordContract: record.RecordContract,
	}
	if choice != nil {
		profile := *choice
		result.ChoiceProfile = &profile
		result.InstrumentationSchema = ChoiceFeatureSchema
		choiceIdentity, err := choiceInstrumentationIdentity(profile, description.SemanticInstrumentationSHA256)
		if err != nil {
			return Identity{}, err
		}
		result.InstrumentationSHA256 = choiceIdentity
	}
	return result, nil
}

func choiceInstrumentationIdentity(profile ChoiceProfileIdentity, instrumentation record.SHA256) (record.SHA256, error) {
	//gomad:dynamic reviewed 2026-10-13 stdlib JSON marshals only validated corpus choice identity inputs
	choiceBytes, err := json.Marshal(profile)
	if err != nil {
		return "", fmt.Errorf("encode guided choice profile identity: %w", err)
	}
	material := append([]byte(string(semanticInstrumentationIdentity(instrumentation))+"\x00"), choiceBytes...)
	return record.DomainHash("gomad3-guide-choice-instrumentation-v1", material), nil
}

func validateIdentityInputStrings(target targetProjection, toolchain record.Toolchain, description backends.Description, environment []record.Environment, choice *ChoiceProfileIdentity) error {
	values := []string{
		target.Kind, string(target.SHA256), target.BuildInfo.GoVersion, target.BuildInfo.Path, target.BuildInfo.MainModule,
		toolchain.GoVersion, toolchain.BuildKey, toolchain.TargetGOOS, toolchain.TargetGOARCH,
		description.BoundaryVersion, string(description.BoundarySHA256), string(description.SemanticInstrumentationSHA256),
	}
	values = append(values, target.Argv...)
	values = append(values, target.BuildTags...)
	for _, adapter := range target.Adapters {
		values = append(values, adapter.Module, adapter.Version, adapter.Sum)
	}
	for _, compatibility := range target.Compatibility {
		values = append(values, compatibility.ID, string(compatibility.SHA256))
	}
	for _, setting := range target.BuildInfo.Settings {
		values = append(values, setting.Key, setting.Value)
	}
	for _, entry := range environment {
		values = append(values, entry.Name, entry.Value)
	}
	if choice != nil {
		values = append(values, choice.Profile, string(choice.ImplementationSHA256))
	}
	for _, value := range values {
		if !utf8.ValidString(value) {
			return errors.New("guided corpus identity input contains invalid UTF-8")
		}
	}
	return nil
}

func semanticInstrumentationIdentity(instrumentation record.SHA256) record.SHA256 {
	identity := SemanticFeatureSchema + "\x00" + record.SemanticCoverageSchema + "\x00" + string(instrumentation) + "\x00"
	return record.DomainHash("gomad3-guide-instrumentation-v1", []byte(identity))
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
	case FeatureChoice:
		return 31
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
