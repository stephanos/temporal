package campaignstore

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go.temporal.io/server/tools/gomadv3/choice"
	"go.temporal.io/server/tools/gomadv3/deterministicio"
	"go.temporal.io/server/tools/gomadv3/evidence"
	"go.temporal.io/server/tools/gomadv3/runner/internal/frontier"
)

const (
	CampaignPlanSchema         = "gomadv3.batch-plan/v5"
	PreviousCampaignPlanSchema = "gomadv3.batch-plan/v4"
	PriorCampaignPlanSchema    = "gomadv3.batch-plan/v3"
	EarlierCampaignPlanSchema  = "gomadv3.batch-plan/v2"
	LegacyBatchPlanSchema      = "gomadv3.batch-plan/v1"
)

const maximumBatchPlanBytes = 1 << 20

type PreparedTargetPlan struct {
	Path   string          `json:"path"`
	Target evidence.Target `json:"target"`
}

type IOProfilePlan = deterministicio.Contract

type ChoiceProfilePlan struct {
	Name                 string                `json:"name"`
	ImplementationSHA256 evidence.SHA256       `json:"implementation_sha256"`
	Limit                evidence.Uint64String `json:"limit"`
}

type GuidancePlan struct {
	Corpus         string          `json:"corpus"`
	SnapshotSHA256 evidence.SHA256 `json:"snapshot_sha256"`
}

type CampaignShard struct {
	Index evidence.Uint64String `json:"index"`
	Count evidence.Uint64String `json:"count"`
}

type CampaignPlan struct {
	Schema                       string                       `json:"schema"`
	PlanSHA256                   evidence.SHA256              `json:"plan_sha256,omitempty"`
	Shard                        *CampaignShard               `json:"shard,omitempty"`
	Strategy                     string                       `json:"strategy,omitempty"`
	Selection                    string                       `json:"selection"`
	SelectionCount               evidence.Uint64String        `json:"selection_count"`
	Parallel                     evidence.Uint64String        `json:"parallel"`
	Journal                      *RunJournalPlan              `json:"journal,omitempty"`
	Artifacts                    *ArtifactCapacityPlan        `json:"artifacts,omitempty"`
	MaxRuns                      evidence.Uint64String        `json:"max_runs,omitempty"`
	MaxChoiceDepth               evidence.Uint64String        `json:"max_choice_depth,omitempty"`
	MaxFrontierBytes             evidence.Uint64String        `json:"max_frontier_bytes,omitempty"`
	FrontierImplementationSHA256 evidence.SHA256              `json:"frontier_implementation_sha256,omitempty"`
	RunTimeoutNanos              evidence.Uint64String        `json:"run_timeout_nanos"`
	OverallTimeoutNanos          evidence.Uint64String        `json:"overall_timeout_nanos"`
	TerminateGraceNanos          evidence.Uint64String        `json:"terminate_grace_nanos"`
	OnFailure                    string                       `json:"on_failure"`
	FailureBudget                evidence.Uint64String        `json:"failure_budget"`
	OutputBytes                  evidence.Uint64String        `json:"output_bytes"`
	WorldTransitionBytes         evidence.Uint64String        `json:"world_transition_bytes"`
	RunnerBuild                  string                       `json:"runner_build"`
	Toolchain                    evidence.Toolchain           `json:"toolchain"`
	Prepared                     PreparedTargetPlan           `json:"prepared"`
	IOProfile                    IOProfilePlan                `json:"io_profile"`
	ChoiceProfile                *ChoiceProfilePlan           `json:"choice_profile,omitempty"`
	Environment                  []evidence.Environment       `json:"environment"`
	IOROMounts                   []string                     `json:"io_ro_mounts"`
	IOROMountLimits              evidence.ReadOnlyMountLimits `json:"io_ro_mount_limits"`
	Coverage                     string                       `json:"coverage"`
	RequiredSemanticProbes       []string                     `json:"required_semantic_probes"`
	KeepSuccesses                string                       `json:"keep_successes"`
	SuccessArtifactLimit         evidence.Uint64String        `json:"success_artifact_limit"`
	SuccessBytesLimit            evidence.Uint64String        `json:"success_bytes_limit"`
	Guidance                     *GuidancePlan                `json:"guidance,omitempty"`
}

func (journal *CampaignJournal) RecordPlan(plan CampaignPlan) error {
	if err := validateCampaignPlan(plan); err != nil {
		return err
	}
	if plan.Selection != journal.config.Selection || uint64(plan.SelectionCount) != journal.config.SelectionCount {
		return fmt.Errorf("batch plan selection does not match journal")
	}
	if plan.PlanSHA256 != journal.config.PlanSHA256 || !equalCampaignShard(plan.Shard, journal.config.Shard) {
		return fmt.Errorf("batch plan shard identity does not match journal")
	}
	if (plan.Schema == CampaignPlanSchema || plan.Schema == PreviousCampaignPlanSchema) && (plan.Journal == nil || *plan.Journal != journal.RunJournalPlan()) {
		return fmt.Errorf("batch plan journal limits do not match journal")
	}
	root, err := os.OpenRoot(journal.path)
	if err != nil {
		return fmt.Errorf("pin batch directory: %w", err)
	}
	defer root.Close()
	digest, size, err := hashValidatedFile(root, filepath.FromSlash(plan.Prepared.Path), 0o500, uint64(plan.Prepared.Target.Size))
	if err != nil || digest != plan.Prepared.Target.SHA256 || size != uint64(plan.Prepared.Target.Size) {
		return errors.Join(fmt.Errorf("prepared target identity does not match batch plan"), err)
	}
	encoded, err := evidence.CanonicalJSON(plan)
	if err != nil {
		return fmt.Errorf("encode batch plan: %w", err)
	}
	path := filepath.Join(journal.PreparedPath(), "plan.json")
	if _, err := os.Lstat(path); err == nil {
		return fmt.Errorf("batch plan already exists")
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := atomicWriteContext(journal.ctx, path, encoded); err != nil {
		return err
	}
	if plan.Artifacts != nil {
		artifacts := *plan.Artifacts
		journal.artifactPlan = &artifacts
	}
	return nil
}

func equalCampaignShard(left, right *CampaignShard) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func ReadResumePlan(path string) (CampaignPlan, error) {
	rootInfo, err := os.Lstat(path)
	if err != nil {
		return CampaignPlan{}, fmt.Errorf("open resumable batch directory: %w", err)
	}
	if !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 || rootInfo.Mode().Perm() != 0o700 {
		return CampaignPlan{}, fmt.Errorf("resumable batch path must be a private directory")
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return CampaignPlan{}, fmt.Errorf("pin resumable batch directory: %w", err)
	}
	defer root.Close()
	pinnedInfo, err := root.Stat(".")
	if err != nil || !os.SameFile(rootInfo, pinnedInfo) {
		return CampaignPlan{}, errors.Join(fmt.Errorf("resumable batch directory changed while opening"), err)
	}
	if _, err := root.Stat("batch.json"); err == nil {
		return CampaignPlan{}, fmt.Errorf("published batch cannot be resumed")
	} else if !os.IsNotExist(err) {
		return CampaignPlan{}, fmt.Errorf("inspect batch publication state: %w", err)
	}
	if _, err := root.Stat(filepath.Join(".partial", "preparation")); err == nil {
		return CampaignPlan{}, fmt.Errorf("batch preparation did not complete")
	} else if !os.IsNotExist(err) {
		return CampaignPlan{}, fmt.Errorf("inspect batch preparation state: %w", err)
	}
	if err := validateResumeLifecycle(root, filepath.Base(path)); err != nil {
		return CampaignPlan{}, err
	}
	planBytes, err := readValidatedFile(root, filepath.Join(".prepared", "plan.json"), 0o600, maximumBatchPlanBytes)
	if err != nil {
		return CampaignPlan{}, fmt.Errorf("read batch plan: %w", err)
	}
	var plan CampaignPlan
	if err := evidence.DecodeCanonicalJSON(planBytes, &plan); err != nil {
		return CampaignPlan{}, fmt.Errorf("decode batch plan: %w", err)
	}
	if err := validateCampaignPlan(plan); err != nil {
		return CampaignPlan{}, err
	}
	digest, size, err := hashValidatedFile(root, filepath.FromSlash(plan.Prepared.Path), 0o500, uint64(plan.Prepared.Target.Size))
	if err != nil || digest != plan.Prepared.Target.SHA256 || size != uint64(plan.Prepared.Target.Size) {
		return CampaignPlan{}, errors.Join(fmt.Errorf("prepared target identity does not match batch plan"), err)
	}
	if plan.Schema == EarlierCampaignPlanSchema || plan.Schema == LegacyBatchPlanSchema {
		plan.Prepared.Target.CapabilityMode = "closure"
	}
	return plan, nil
}

func validateResumeLifecycle(root *os.Root, campaignID string) error {
	contents, err := readValidatedFile(root, filepath.Join(".partial", "batch", "partial.json"), 0o600, maximumBatchPlanBytes)
	if err != nil {
		return fmt.Errorf("read batch lifecycle: %w", err)
	}
	lifecycle, err := decodeLifecycleRecord(contents, campaignID)
	if err != nil {
		return err
	}
	if lifecycle.State != LifecyclePrepared && lifecycle.State != LifecycleRunning && lifecycle.State != LifecycleCommitting &&
		(lifecycle.State != LifecycleRecoverableFailure || lifecycle.LastStableState != LifecyclePrepared && lifecycle.LastStableState != LifecycleRunning) {
		return fmt.Errorf("batch lifecycle is not resumable")
	}
	return nil
}

func validateCampaignPlan(plan CampaignPlan) error {
	if plan.Schema != CampaignPlanSchema && plan.Schema != PreviousCampaignPlanSchema && plan.Schema != PriorCampaignPlanSchema && plan.Schema != EarlierCampaignPlanSchema && plan.Schema != LegacyBatchPlanSchema || plan.Selection == "" || plan.SelectionCount == 0 || plan.Parallel == 0 {
		return fmt.Errorf("batch plan identity is invalid")
	}
	if plan.Schema != CampaignPlanSchema && plan.Schema != PreviousCampaignPlanSchema && (plan.Journal != nil || plan.Artifacts != nil) {
		return fmt.Errorf("historical batch plan contains segmented journal or artifact limits")
	}
	if plan.Schema == CampaignPlanSchema || plan.Schema == PreviousCampaignPlanSchema {
		if plan.Journal == nil || validateRunJournalPlan(*plan.Journal, plan) != nil || plan.Artifacts == nil {
			return fmt.Errorf("batch plan journal limits are invalid")
		}
		artifacts, err := DeriveArtifactCapacityPlan(plan)
		if err != nil || artifacts != *plan.Artifacts {
			return errors.Join(fmt.Errorf("batch plan artifact limits are invalid"), err)
		}
	}
	if plan.Schema != CampaignPlanSchema && (plan.PlanSHA256 != "" || plan.Shard != nil) {
		return fmt.Errorf("historical batch plan contains portable plan identity")
	}
	if (plan.PlanSHA256 == "") != (plan.Shard == nil) {
		return fmt.Errorf("batch plan portable identity is incomplete")
	}
	if plan.Shard != nil {
		if !validRecordSHA256(plan.PlanSHA256) || plan.Shard.Count == 0 || plan.Shard.Index >= plan.Shard.Count || plan.Strategy != "seed" || plan.Guidance != nil || plan.OnFailure != "all" {
			return fmt.Errorf("batch plan shard identity is invalid")
		}
	}
	if plan.Schema == LegacyBatchPlanSchema {
		if plan.Strategy != "" || plan.MaxRuns != 0 || plan.MaxChoiceDepth != 0 || plan.MaxFrontierBytes != 0 || plan.FrontierImplementationSHA256 != "" {
			return fmt.Errorf("legacy batch plan contains frontier fields")
		}
	} else {
		switch plan.Strategy {
		case "seed":
			if plan.MaxRuns != 0 || plan.MaxChoiceDepth != 0 || plan.MaxFrontierBytes != 0 || plan.FrontierImplementationSHA256 != "" {
				return fmt.Errorf("seed batch plan contains frontier bounds")
			}
		case "choice-frontier":
			if plan.SelectionCount != 1 || plan.MaxRuns == 0 || plan.MaxChoiceDepth == 0 || plan.MaxFrontierBytes == 0 || plan.Guidance != nil || plan.ChoiceProfile == nil || plan.FrontierImplementationSHA256 != frontier.ImplementationSHA256() {
				return fmt.Errorf("choice-frontier batch plan is invalid")
			}
		default:
			return fmt.Errorf("batch plan strategy is invalid")
		}
	}
	if plan.RunTimeoutNanos == 0 || plan.OverallTimeoutNanos == 0 || plan.OutputBytes == 0 || plan.WorldTransitionBytes == 0 || plan.TerminateGraceNanos > plan.RunTimeoutNanos || plan.TerminateGraceNanos > plan.OverallTimeoutNanos {
		return fmt.Errorf("batch plan limits are invalid")
	}
	if plan.OnFailure != "first" && plan.OnFailure != "budget" && plan.OnFailure != "all" || plan.FailureBudget == 0 {
		return fmt.Errorf("batch plan failure policy is invalid")
	}
	if plan.RunnerBuild == "" || plan.Toolchain.GoVersion == "" || plan.Toolchain.BuildKey == "" || plan.Toolchain.TargetGOOS == "" || plan.Toolchain.TargetGOARCH == "" {
		return fmt.Errorf("batch plan build identity is invalid")
	}
	preparedPath := filepath.ToSlash(filepath.Clean(filepath.FromSlash(plan.Prepared.Path)))
	if preparedPath != plan.Prepared.Path || !strings.HasPrefix(preparedPath, ".prepared/") || strings.Contains(preparedPath, "..") {
		return fmt.Errorf("batch plan prepared target path is invalid")
	}
	target := plan.Prepared.Target
	if target.Kind == "" || target.Source == "" || !validRecordSHA256(target.SHA256) || target.Size == 0 || len(target.Argv) == 0 || target.Argv[0] == "" {
		return fmt.Errorf("batch plan prepared target identity is invalid")
	}
	if err := evidence.ValidateCompatibilityPacks(target.Compatibility); err != nil {
		return fmt.Errorf("batch plan prepared target: %w", err)
	}
	if err := evidence.ValidateTargetAdapters(target.Adapters); err != nil {
		return fmt.Errorf("batch plan prepared target: %w", err)
	}
	if plan.Schema == CampaignPlanSchema || plan.Schema == PreviousCampaignPlanSchema || plan.Schema == PriorCampaignPlanSchema {
		if err := evidence.ValidateCurrentTargetCapability(target); err != nil {
			return fmt.Errorf("batch plan prepared target: %w", err)
		}
	} else if target.CapabilityMode != "" || target.CapabilityManifest != nil {
		return fmt.Errorf("historical batch plan contains linked capability evidence")
	}
	if plan.IOProfile.Name == "" || !validRecordSHA256(evidence.SHA256(plan.IOProfile.ImplementationSHA256)) || !validRecordSHA256(evidence.SHA256(plan.IOProfile.InventorySHA256)) {
		return fmt.Errorf("batch plan I/O profile identity is invalid")
	}
	if choices := plan.ChoiceProfile; choices != nil {
		implementation, err := choice.ImplementationIdentity(plan.Toolchain.BuildKey)
		if err != nil || choices.Name != choice.Profile || choices.ImplementationSHA256 != evidence.SHA256FromSum(implementation) || choices.Limit < choice.MinimumTraceBytes || choices.Limit > choice.MaximumTraceBytes {
			return fmt.Errorf("batch plan choice profile identity is invalid")
		}
	}
	if plan.Coverage != "none" && plan.Coverage != "semantic" && plan.Coverage != "choice" && plan.Coverage != "semantic+choice" ||
		(plan.Coverage == "none" || plan.Coverage == "choice") && len(plan.RequiredSemanticProbes) != 0 ||
		(plan.Coverage == "choice" || plan.Coverage == "semantic+choice") && plan.ChoiceProfile == nil {
		return fmt.Errorf("batch plan coverage policy is invalid")
	}
	if !sort.StringsAreSorted(plan.RequiredSemanticProbes) {
		return fmt.Errorf("batch plan required semantic probes are not sorted")
	}
	for index, probe := range plan.RequiredSemanticProbes {
		if probe == "" || index > 0 && plan.RequiredSemanticProbes[index-1] == probe {
			return fmt.Errorf("batch plan required semantic probes are invalid")
		}
	}
	switch plan.KeepSuccesses {
	case "none":
		if plan.SuccessArtifactLimit != 0 || plan.SuccessBytesLimit != 0 {
			return fmt.Errorf("batch plan disabled success retention has capacity")
		}
	case "novel":
		if plan.Coverage == "none" || plan.SuccessArtifactLimit == 0 || plan.SuccessBytesLimit == 0 {
			return fmt.Errorf("batch plan novel success retention policy is invalid")
		}
	case "all":
		if plan.SuccessArtifactLimit == 0 || plan.SuccessBytesLimit == 0 {
			return fmt.Errorf("batch plan success retention capacity is invalid")
		}
	default:
		return fmt.Errorf("batch plan success retention policy is invalid")
	}
	if plan.Guidance != nil {
		corpus := filepath.Clean(plan.Guidance.Corpus)
		if !filepath.IsAbs(corpus) || corpus != plan.Guidance.Corpus || corpus == filepath.VolumeName(corpus)+string(filepath.Separator) || !validRecordSHA256(plan.Guidance.SnapshotSHA256) || plan.Coverage == "none" {
			return fmt.Errorf("batch plan guidance identity is invalid")
		}
	}
	return nil
}

func ValidateCampaignPlan(plan CampaignPlan) error {
	return validateCampaignPlan(plan)
}
