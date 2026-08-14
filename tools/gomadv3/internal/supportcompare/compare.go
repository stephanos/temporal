package supportcompare

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"go.temporal.io/server/tools/gomadv3/internal/capabilityanalysis"
	"go.temporal.io/server/tools/gomadv3/internal/choicewire"
	"go.temporal.io/server/tools/gomadv3/internal/qualificationset"
	"go.temporal.io/server/tools/gomadv3/internal/record"
)

const Schema = "gomadv3.support-comparison/v1"

const BoundaryDiffDomain = "gomadv3.support-boundary-diff/v1"

type Classification string

const (
	ClassificationClean        Classification = "clean"
	ClassificationImproved     Classification = "improved"
	ClassificationRegressed    Classification = "regressed"
	ClassificationIncomparable Classification = "incomparable"
)

type Input struct {
	Baseline             qualificationset.SetReport
	Candidate            qualificationset.SetReport
	ApprovedBoundaryDiff record.SHA256
}

type Result struct {
	Schema              string               `json:"schema"`
	Classification      Classification       `json:"classification"`
	ReviewRequired      bool                 `json:"review_required"`
	IdentityDifferences []IdentityDifference `json:"identity_differences"`
	Boundary            BoundaryComparison   `json:"boundary"`
	Workloads           []WorkloadChange     `json:"workloads"`
}

type IdentityDifference struct {
	Field     string `json:"field"`
	Baseline  string `json:"baseline"`
	Candidate string `json:"candidate"`
}

type BoundaryComparison struct {
	Changed          bool          `json:"changed"`
	Approved         bool          `json:"approved"`
	DiffSHA256       record.SHA256 `json:"diff_sha256,omitempty"`
	BaselineVersion  string        `json:"baseline_version"`
	BaselineSHA256   record.SHA256 `json:"baseline_sha256"`
	CandidateVersion string        `json:"candidate_version"`
	CandidateSHA256  record.SHA256 `json:"candidate_sha256"`
}

type WorkloadChange struct {
	ID                      string            `json:"id"`
	BaselineClassification  string            `json:"baseline_classification,omitempty"`
	CandidateClassification string            `json:"candidate_classification,omitempty"`
	SupportTransition       string            `json:"support_transition,omitempty"`
	ExpectationChanged      bool              `json:"expectation_changed"`
	BlockersAdded           []BlockerIdentity `json:"blockers_added"`
	BlockersRemoved         []BlockerIdentity `json:"blockers_removed"`
	ReplayRegressed         bool              `json:"replay_regressed"`
	ReplayImproved          bool              `json:"replay_improved"`
	ElapsedNanosDelta       int64             `json:"elapsed_nanos_delta"`
	ArtifactBytesDelta      int64             `json:"artifact_bytes_delta"`
	TraceBytesDelta         int64             `json:"trace_bytes_delta"`
	Choice                  ChoiceChange      `json:"choice"`
}

type BlockerIdentity struct {
	SHA256  record.SHA256              `json:"sha256"`
	Blocker capabilityanalysis.Blocker `json:"blocker"`
}

type ChoiceChange struct {
	Comparable bool                 `json:"comparable"`
	Reason     string               `json:"reason,omitempty"`
	Added      []choicewire.Feature `json:"added"`
	Removed    []choicewire.Feature `json:"removed"`
}

func Compare(input Input) (Result, error) {
	if input.Baseline.Schema != qualificationset.ReportSchema || input.Candidate.Schema != qualificationset.ReportSchema {
		return Result{}, errors.New("support comparison requires normalized qualification-set reports")
	}
	result := Result{
		Schema: Schema, IdentityDifferences: []IdentityDifference{}, Workloads: []WorkloadChange{},
		Boundary: compareBoundary(input.Baseline, input.Candidate, input.ApprovedBoundaryDiff),
	}
	incomparable := compareIdentities(&result, input.Baseline, input.Candidate)
	workloads := compareWorkloadSets(input.Baseline, input.Candidate)
	result.Workloads = workloads.changes
	if incomparable || workloads.membershipChanged {
		result.Classification = ClassificationIncomparable
	} else if workloads.regressed {
		result.Classification = ClassificationRegressed
	} else if workloads.improved {
		result.Classification = ClassificationImproved
	} else {
		result.Classification = ClassificationClean
	}
	result.ReviewRequired = result.Classification == ClassificationRegressed || result.Boundary.Changed && !result.Boundary.Approved
	return result, nil
}

type workloadComparison struct {
	changes           []WorkloadChange
	membershipChanged bool
	regressed         bool
	improved          bool
}

func compareWorkloadSets(baselineReport, candidateReport qualificationset.SetReport) workloadComparison {
	result := workloadComparison{changes: []WorkloadChange{}}
	baseline := workloadsByID(baselineReport.Suites)
	candidate := workloadsByID(candidateReport.Suites)
	for _, id := range workloadIdentities(baseline, candidate) {
		baselineWorkload, baselineFound := baseline[id]
		candidateWorkload, candidateFound := candidate[id]
		comparison := compareWorkload(
			id, baselineWorkload, baselineFound, candidateWorkload, candidateFound, baselineReport.Dimensions.Choice, candidateReport.Dimensions.Choice,
		)
		result.changes = append(result.changes, comparison.change)
		result.membershipChanged = result.membershipChanged || comparison.membershipChanged
		result.regressed = result.regressed || comparison.regressed
		result.improved = result.improved || comparison.improved
	}
	return result
}

func workloadIdentities(baseline, candidate map[string]qualificationset.SuiteReport) []string {
	identities := make([]string, 0, len(baseline)+len(candidate))
	seen := make(map[string]struct{}, len(baseline)+len(candidate))
	for id := range baseline {
		identities = append(identities, id)
		seen[id] = struct{}{}
	}
	for id := range candidate {
		if _, found := seen[id]; !found {
			identities = append(identities, id)
		}
	}
	sort.Strings(identities)
	return identities
}

type workloadChangeResult struct {
	change            WorkloadChange
	membershipChanged bool
	regressed         bool
	improved          bool
}

func compareWorkload(id string, baseline qualificationset.SuiteReport, baselineFound bool, candidate qualificationset.SuiteReport, candidateFound bool, baselineChoice, candidateChoice bool) workloadChangeResult {
	result := workloadChangeResult{change: WorkloadChange{
		ID: id, BlockersAdded: []BlockerIdentity{}, BlockersRemoved: []BlockerIdentity{},
		Choice: ChoiceChange{Added: []choicewire.Feature{}, Removed: []choicewire.Feature{}},
	}}
	if baselineFound {
		result.change.BaselineClassification = baseline.Classification
	}
	if candidateFound {
		result.change.CandidateClassification = candidate.Classification
	}
	switch {
	case !baselineFound:
		result.change.SupportTransition = "added"
		result.membershipChanged = true
	case !candidateFound:
		result.change.SupportTransition = "removed"
		result.membershipChanged = true
	default:
		baselineRank := supportRank(baseline.Classification)
		candidateRank := supportRank(candidate.Classification)
		if candidateRank > baselineRank {
			result.change.SupportTransition = "improved"
			result.improved = true
		} else if candidateRank < baselineRank {
			result.change.SupportTransition = "regressed"
			result.regressed = true
		}
		result.change.ExpectationChanged = baseline.Expected != candidate.Expected
		result.regressed = result.regressed || result.change.ExpectationChanged
		result.change.BlockersAdded, result.change.BlockersRemoved = compareBlockers(baseline.Blockers, candidate.Blockers)
		result.regressed = result.regressed || len(result.change.BlockersAdded) != 0 || len(result.change.BlockersRemoved) != 0
		result.change.ReplayRegressed, result.change.ReplayImproved = compareReplay(baseline.Seeds, candidate.Seeds)
		result.regressed = result.regressed || result.change.ReplayRegressed
		result.improved = result.improved || result.change.ReplayImproved
		result.change.ElapsedNanosDelta = signedDelta(sumElapsed(candidate.Seeds), sumElapsed(baseline.Seeds))
		result.change.ArtifactBytesDelta = signedDelta(sumArtifacts(candidate.Seeds), sumArtifacts(baseline.Seeds))
		result.change.TraceBytesDelta = signedDelta(sumTrace(candidate.Seeds), sumTrace(baseline.Seeds))
		result.change.Choice = compareChoice(baselineChoice, candidateChoice, baseline.Choice, candidate.Choice)
	}
	return result
}

func compareIdentities(result *Result, baseline, candidate qualificationset.SetReport) bool {
	differences := []IdentityDifference{}
	add := func(field, left, right string) {
		if left != right {
			differences = append(differences, IdentityDifference{Field: field, Baseline: left, Candidate: right})
		}
	}
	if !baseline.Dimensions.PortableV3 || !candidate.Dimensions.PortableV3 {
		add("dimensions.portable_v3", fmt.Sprint(baseline.Dimensions.PortableV3), fmt.Sprint(candidate.Dimensions.PortableV3))
		if baseline.Dimensions.PortableV3 == candidate.Dimensions.PortableV3 {
			differences = append(differences, IdentityDifference{Field: "dimensions.portable_v3", Baseline: "false", Candidate: "false"})
		}
	}
	add("corpus.name", baseline.Name, candidate.Name)
	add("module.path", baseline.Module.Path, candidate.Module.Path)
	add("module.go_mod_sha256", string(baseline.Module.GoModSHA256), string(candidate.Module.GoModSHA256))
	add("platform.goos", baseline.Platform.GOOS, candidate.Platform.GOOS)
	add("platform.goarch", baseline.Platform.GOARCH, candidate.Platform.GOARCH)
	add("toolchain.go_version", baseline.Toolchain.GoVersion, candidate.Toolchain.GoVersion)
	add("toolchain.build_key", baseline.Toolchain.BuildKey, candidate.Toolchain.BuildKey)
	add("io_profile.name", baseline.IOProfile.Name, candidate.IOProfile.Name)
	add("io_profile.implementation_sha256", string(baseline.IOProfile.ImplementationSHA256), string(candidate.IOProfile.ImplementationSHA256))
	add("io_profile.inventory_sha256", string(baseline.IOProfile.InventorySHA256), string(candidate.IOProfile.InventorySHA256))
	sort.Slice(differences, func(i, j int) bool { return differences[i].Field < differences[j].Field })
	result.IdentityDifferences = differences
	return !baseline.Dimensions.PortableV3 || !candidate.Dimensions.PortableV3 || baseline.Name != candidate.Name || baseline.Module.Path != candidate.Module.Path
}

func compareBoundary(baseline, candidate qualificationset.SetReport, approval record.SHA256) BoundaryComparison {
	result := BoundaryComparison{
		BaselineVersion: baseline.Toolchain.BoundaryManifestVersion, BaselineSHA256: baseline.Toolchain.BoundaryManifestSHA256,
		CandidateVersion: candidate.Toolchain.BoundaryManifestVersion, CandidateSHA256: candidate.Toolchain.BoundaryManifestSHA256,
	}
	result.Changed = result.BaselineVersion != result.CandidateVersion || result.BaselineSHA256 != result.CandidateSHA256
	if !result.Changed {
		return result
	}
	payload, _ := record.CanonicalJSON(struct {
		BaselineVersion  string        `json:"baseline_version"`
		BaselineSHA256   record.SHA256 `json:"baseline_sha256"`
		CandidateVersion string        `json:"candidate_version"`
		CandidateSHA256  record.SHA256 `json:"candidate_sha256"`
	}{result.BaselineVersion, result.BaselineSHA256, result.CandidateVersion, result.CandidateSHA256})
	result.DiffSHA256 = record.DomainHash(BoundaryDiffDomain, payload)
	result.Approved = approval != "" && approval == result.DiffSHA256
	return result
}

func workloadsByID(workloads []qualificationset.SuiteReport) map[string]qualificationset.SuiteReport {
	result := make(map[string]qualificationset.SuiteReport, len(workloads))
	for _, workload := range workloads {
		result[workload.ID] = workload
	}
	return result
}

func supportRank(classification string) int {
	switch classification {
	case "qualified":
		return 2
	case "unsupported_target":
		return 1
	default:
		return 0
	}
}

func compareBlockers(baseline, candidate []capabilityanalysis.Blocker) (added []BlockerIdentity, removed []BlockerIdentity) {
	left := blockerIdentities(baseline)
	right := blockerIdentities(candidate)
	added = []BlockerIdentity{}
	removed = []BlockerIdentity{}
	for digest, blocker := range right {
		if _, found := left[digest]; !found {
			added = append(added, blocker)
		}
	}
	for digest, blocker := range left {
		if _, found := right[digest]; !found {
			removed = append(removed, blocker)
		}
	}
	sortBlockers(added)
	sortBlockers(removed)
	return added, removed
}

func blockerIdentities(blockers []capabilityanalysis.Blocker) map[record.SHA256]BlockerIdentity {
	result := make(map[record.SHA256]BlockerIdentity, len(blockers))
	for _, blocker := range blockers {
		encoded, _ := record.CanonicalJSON(blocker)
		digest := record.DomainHash("gomadv3.support-blocker/v1", encoded)
		result[digest] = BlockerIdentity{SHA256: digest, Blocker: blocker}
	}
	return result
}

func sortBlockers(blockers []BlockerIdentity) {
	sort.Slice(blockers, func(i, j int) bool { return blockers[i].SHA256 < blockers[j].SHA256 })
}

func compareReplay(baseline, candidate []qualificationset.SeedReport) (regressed bool, improved bool) {
	left := replayBySeed(baseline)
	right := replayBySeed(candidate)
	for seed, baselineMatch := range left {
		candidateMatch, found := right[seed]
		if baselineMatch && (!found || !candidateMatch) {
			regressed = true
		}
		if !baselineMatch && found && candidateMatch {
			improved = true
		}
	}
	return regressed, improved
}

func replayBySeed(seeds []qualificationset.SeedReport) map[record.Uint64String]bool {
	result := make(map[record.Uint64String]bool, len(seeds))
	for _, seed := range seeds {
		result[seed.Seed] = seed.Replayed && seed.ReplayMatch
	}
	return result
}

func compareChoice(baselineDimension, candidateDimension bool, baseline, candidate qualificationset.ChoiceCoverage) ChoiceChange {
	result := ChoiceChange{Added: []choicewire.Feature{}, Removed: []choicewire.Feature{}}
	if !baselineDimension || !candidateDimension || !baseline.Available || !candidate.Available {
		result.Reason = "dimension_unavailable"
		return result
	}
	if baseline.Profile != candidate.Profile || baseline.ImplementationSHA256 != candidate.ImplementationSHA256 || baseline.Limit != candidate.Limit {
		result.Reason = "identity_mismatch"
		return result
	}
	result.Comparable = true
	left := make(map[choicewire.Feature]struct{}, len(baseline.Features))
	right := make(map[choicewire.Feature]struct{}, len(candidate.Features))
	for _, feature := range baseline.Features {
		left[feature] = struct{}{}
	}
	for _, feature := range candidate.Features {
		right[feature] = struct{}{}
	}
	for feature := range right {
		if _, found := left[feature]; !found {
			result.Added = append(result.Added, feature)
		}
	}
	for feature := range left {
		if _, found := right[feature]; !found {
			result.Removed = append(result.Removed, feature)
		}
	}
	sortFeatures(result.Added)
	sortFeatures(result.Removed)
	return result
}

func sortFeatures(features []choicewire.Feature) {
	sort.Slice(features, func(i, j int) bool {
		if features[i].Kind != features[j].Kind {
			return features[i].Kind < features[j].Kind
		}
		return features[i].Value < features[j].Value
	})
}

func sumElapsed(seeds []qualificationset.SeedReport) uint64 {
	var result uint64
	for _, seed := range seeds {
		result += uint64(seed.ElapsedNanos)
	}
	return result
}

func sumArtifacts(seeds []qualificationset.SeedReport) uint64 {
	var result uint64
	for _, seed := range seeds {
		result += uint64(seed.ArtifactBytes)
	}
	return result
}

func sumTrace(seeds []qualificationset.SeedReport) uint64 {
	var result uint64
	for _, seed := range seeds {
		result += uint64(seed.TraceBytes)
	}
	return result
}

func signedDelta(candidate, baseline uint64) int64 {
	if candidate >= baseline {
		if candidate-baseline > uint64(^uint64(0)>>1) {
			return int64(^uint64(0) >> 1)
		}
		return int64(candidate - baseline)
	}
	if baseline-candidate > uint64(^uint64(0)>>1) {
		return -int64(^uint64(0) >> 1)
	}
	return -int64(baseline - candidate)
}

func FormatText(result Result) string {
	var output strings.Builder
	fmt.Fprintf(&output, "support comparison: %s review-required=%t\n", result.Classification, result.ReviewRequired)
	for _, difference := range result.IdentityDifferences {
		fmt.Fprintf(&output, "identity: %s baseline=%s candidate=%s\n", difference.Field, difference.Baseline, difference.Candidate)
	}
	if result.Boundary.Changed {
		fmt.Fprintf(&output, "boundary: changed digest=%s approved=%t\n", result.Boundary.DiffSHA256, result.Boundary.Approved)
	}
	for _, workload := range result.Workloads {
		fmt.Fprintf(&output, "workload: %s baseline=%s candidate=%s", workload.ID, workload.BaselineClassification, workload.CandidateClassification)
		if workload.SupportTransition != "" {
			fmt.Fprintf(&output, " transition=%s", workload.SupportTransition)
		}
		output.WriteByte('\n')
	}
	return output.String()
}
