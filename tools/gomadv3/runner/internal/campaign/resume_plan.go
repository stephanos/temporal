package campaign

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go.temporal.io/server/tools/gomadv3/choice"
	"go.temporal.io/server/tools/gomadv3/deterministicio"
	"go.temporal.io/server/tools/gomadv3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomadv3/record"
	choiceengine "go.temporal.io/server/tools/gomadv3/runner/internal/exploration/choice"
	simulationengine "go.temporal.io/server/tools/gomadv3/runner/internal/exploration/simulation"
)

const CampaignPlanSchema = "gomadv3.campaign-plan/v1"

const maximumCampaignPlanBytes = 1 << 20

type PreparedTargetPlan struct {
	Path   string        `json:"path"`
	Target record.Target `json:"target"`
}

type ChoiceProfilePlan struct {
	Name                 string              `json:"name"`
	ImplementationSHA256 record.SHA256       `json:"implementation_sha256"`
	Limit                record.Uint64String `json:"limit"`
}

type GuidancePlan struct {
	Corpus         string        `json:"corpus"`
	SnapshotSHA256 record.SHA256 `json:"snapshot_sha256"`
}

type CampaignShard struct {
	Index record.Uint64String `json:"index"`
	Count record.Uint64String `json:"count"`
}

type CampaignPlan struct {
	Schema                                    string                           `json:"schema"`
	PlanSHA256                                record.SHA256                    `json:"plan_sha256,omitempty"`
	Shard                                     *CampaignShard                   `json:"shard,omitempty"`
	Strategy                                  string                           `json:"strategy,omitempty"`
	Selection                                 string                           `json:"selection"`
	SelectionCount                            record.Uint64String              `json:"selection_count"`
	Parallel                                  record.Uint64String              `json:"parallel"`
	Journal                                   *ExecutionJournalPlan            `json:"journal,omitempty"`
	Artifacts                                 *ArtifactCapacityPlan            `json:"artifacts,omitempty"`
	MaxExecutions                             record.Uint64String              `json:"max_executions,omitempty"`
	MaxChoiceDepth                            record.Uint64String              `json:"max_choice_depth,omitempty"`
	MaxForcedDecisions                        record.Uint64String              `json:"max_forced_decisions,omitempty"`
	MaxExplorationBytes                       record.Uint64String              `json:"max_exploration_bytes,omitempty"`
	MaxExplorationResultBytes                 record.Uint64String              `json:"max_exploration_result_bytes,omitempty"`
	SimulationDimensionLimits                 simulationengine.DimensionLimits `json:"simulation_dimension_limits,omitempty"`
	ChoiceExplorationImplementationSHA256     record.SHA256                    `json:"choice_exploration_implementation_sha256,omitempty"`
	SimulationExplorationImplementationSHA256 record.SHA256                    `json:"simulation_exploration_implementation_sha256,omitempty"`
	ExecutionTimeoutNanos                     record.Uint64String              `json:"execution_timeout_nanos"`
	OverallTimeoutNanos                       record.Uint64String              `json:"overall_timeout_nanos"`
	TerminateGraceNanos                       record.Uint64String              `json:"terminate_grace_nanos"`
	OnFailure                                 string                           `json:"on_failure"`
	FailureBudget                             record.Uint64String              `json:"failure_budget"`
	OutputBytes                               record.Uint64String              `json:"output_bytes"`
	WorldTransitionBytes                      record.Uint64String              `json:"world_transition_bytes"`
	RunnerBuild                               string                           `json:"runner_build"`
	Toolchain                                 record.Toolchain                 `json:"toolchain"`
	Prepared                                  PreparedTargetPlan               `json:"prepared"`
	IOProfile                                 deterministicio.Contract         `json:"io_profile"`
	ChoiceProfile                             *ChoiceProfilePlan               `json:"choice_profile,omitempty"`
	Environment                               []record.Environment             `json:"environment"`
	IOROMounts                                []string                         `json:"io_ro_mounts"`
	IOROMountLimits                           record.ReadOnlyMountLimits       `json:"io_ro_mount_limits"`
	Coverage                                  string                           `json:"coverage"`
	RequiredSemanticProbes                    []string                         `json:"required_semantic_probes"`
	KeepSuccesses                             string                           `json:"keep_successes"`
	SuccessArtifactLimit                      record.Uint64String              `json:"success_artifact_limit"`
	SuccessBytesLimit                         record.Uint64String              `json:"success_bytes_limit"`
	Guidance                                  *GuidancePlan                    `json:"guidance,omitempty"`
}

func (journal *CampaignJournal) RecordPlan(plan CampaignPlan) error {
	if err := validateCampaignPlan(plan); err != nil {
		return err
	}
	if plan.Selection != journal.config.Selection || uint64(plan.SelectionCount) != journal.config.SelectionCount {
		return fmt.Errorf("campaign plan selection does not match journal")
	}
	if plan.PlanSHA256 != journal.config.PlanSHA256 || !equalCampaignShard(plan.Shard, journal.config.Shard) {
		return fmt.Errorf("campaign plan shard identity does not match journal")
	}
	if plan.Journal == nil || *plan.Journal != journal.ExecutionJournalPlan() {
		return fmt.Errorf("campaign plan journal limits do not match journal")
	}
	root, err := os.OpenRoot(journal.path)
	if err != nil {
		return fmt.Errorf("pin campaign directory: %w", err)
	}
	defer root.Close()
	digest, size, err := hashValidatedFile(root, filepath.FromSlash(plan.Prepared.Path), 0o500, uint64(plan.Prepared.Target.Size))
	if err != nil || digest != plan.Prepared.Target.SHA256 || size != uint64(plan.Prepared.Target.Size) {
		return errors.Join(fmt.Errorf("prepared target identity does not match campaign plan"), err)
	}
	encoded, err := canonicaljson.CanonicalJSON(plan)
	if err != nil {
		return fmt.Errorf("encode campaign plan: %w", err)
	}
	path := filepath.Join(journal.PreparedPath(), "plan.json")
	if _, err := os.Lstat(path); err == nil {
		return fmt.Errorf("campaign plan already exists")
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
		return CampaignPlan{}, fmt.Errorf("open resumable campaign directory: %w", err)
	}
	if !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 || rootInfo.Mode().Perm() != 0o700 {
		return CampaignPlan{}, fmt.Errorf("resumable campaign path must be a private directory")
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return CampaignPlan{}, fmt.Errorf("pin resumable campaign directory: %w", err)
	}
	defer root.Close()
	pinnedInfo, err := root.Stat(".")
	if err != nil || !os.SameFile(rootInfo, pinnedInfo) {
		return CampaignPlan{}, errors.Join(fmt.Errorf("resumable campaign directory changed while opening"), err)
	}
	if _, err := root.Stat("campaign.json"); err == nil {
		return CampaignPlan{}, fmt.Errorf("published campaign cannot be resumed")
	} else if !os.IsNotExist(err) {
		return CampaignPlan{}, fmt.Errorf("inspect campaign publication state: %w", err)
	}
	if _, err := root.Stat(filepath.Join(".partial", "preparation")); err == nil {
		return CampaignPlan{}, fmt.Errorf("campaign preparation did not complete")
	} else if !os.IsNotExist(err) {
		return CampaignPlan{}, fmt.Errorf("inspect campaign preparation state: %w", err)
	}
	if err := validateResumeLifecycle(root, filepath.Base(path)); err != nil {
		return CampaignPlan{}, err
	}
	planBytes, err := readValidatedFile(root, filepath.Join(".prepared", "plan.json"), 0o600, maximumCampaignPlanBytes)
	if err != nil {
		return CampaignPlan{}, fmt.Errorf("read campaign plan: %w", err)
	}
	var plan CampaignPlan
	if err := canonicaljson.DecodeCanonicalJSON(planBytes, &plan); err != nil {
		return CampaignPlan{}, fmt.Errorf("decode campaign plan: %w", err)
	}
	if err := validateCampaignPlan(plan); err != nil {
		return CampaignPlan{}, err
	}
	digest, size, err := hashValidatedFile(root, filepath.FromSlash(plan.Prepared.Path), 0o500, uint64(plan.Prepared.Target.Size))
	if err != nil || digest != plan.Prepared.Target.SHA256 || size != uint64(plan.Prepared.Target.Size) {
		return CampaignPlan{}, errors.Join(fmt.Errorf("prepared target identity does not match campaign plan"), err)
	}
	return plan, nil
}

func validateResumeLifecycle(root *os.Root, campaignID string) error {
	contents, err := readValidatedFile(root, filepath.Join(".partial", "campaign", "partial.json"), 0o600, maximumCampaignPlanBytes)
	if err != nil {
		return fmt.Errorf("read campaign lifecycle: %w", err)
	}
	lifecycle, err := decodeLifecycleRecord(contents, campaignID)
	if err != nil {
		return err
	}
	if lifecycle.State != LifecyclePrepared && lifecycle.State != LifecycleRunning && lifecycle.State != LifecycleCommitting &&
		(lifecycle.State != LifecycleRecoverableFailure || lifecycle.LastStableState != LifecyclePrepared && lifecycle.LastStableState != LifecycleRunning) {
		return fmt.Errorf("campaign lifecycle is not resumable")
	}
	return nil
}

func validateCampaignPlan(plan CampaignPlan) error {
	if plan.Schema != CampaignPlanSchema || plan.Selection == "" || plan.SelectionCount == 0 || plan.Parallel == 0 {
		return fmt.Errorf("campaign plan identity is invalid")
	}
	if plan.Journal == nil || validateExecutionJournalPlan(*plan.Journal, plan) != nil || plan.Artifacts == nil {
		return fmt.Errorf("campaign plan journal limits are invalid")
	}
	artifacts, err := DeriveArtifactCapacityPlan(plan)
	if err != nil || artifacts != *plan.Artifacts {
		return errors.Join(fmt.Errorf("campaign plan artifact limits are invalid"), err)
	}
	if (plan.PlanSHA256 == "") != (plan.Shard == nil) {
		return fmt.Errorf("campaign plan portable identity is incomplete")
	}
	if plan.Shard != nil {
		if !validRecordSHA256(plan.PlanSHA256) || plan.Shard.Count == 0 || plan.Shard.Index >= plan.Shard.Count || plan.Strategy != "seed" || plan.Guidance != nil || plan.OnFailure != "all" {
			return fmt.Errorf("campaign plan shard identity is invalid")
		}
	}
	switch plan.Strategy {
	case "seed":
		if hasCampaignPlanExplorationFields(plan) {
			return fmt.Errorf("seed campaign plan contains exploration bounds")
		}
	case "choice-exploration":
		if plan.SelectionCount != 1 || plan.MaxExecutions == 0 || plan.MaxChoiceDepth == 0 || plan.MaxExplorationBytes == 0 || plan.MaxForcedDecisions != 0 || plan.MaxExplorationResultBytes != 0 || plan.SimulationDimensionLimits != (simulationengine.DimensionLimits{}) || plan.Guidance != nil || plan.ChoiceProfile == nil || plan.ChoiceExplorationImplementationSHA256 != choiceengine.ImplementationSHA256() || plan.SimulationExplorationImplementationSHA256 != "" {
			return fmt.Errorf("choice-exploration campaign plan is invalid")
		}
	case "simulation-exploration":
		if plan.SelectionCount != 1 || plan.MaxExecutions == 0 || plan.MaxChoiceDepth != 0 || plan.MaxForcedDecisions == 0 || plan.MaxExplorationBytes == 0 || plan.MaxExplorationResultBytes == 0 || plan.Guidance != nil || plan.ChoiceProfile == nil || plan.ChoiceExplorationImplementationSHA256 != "" || plan.SimulationExplorationImplementationSHA256 != simulationengine.ImplementationSHA256() || !validSimulationDimensionLimits(plan.SimulationDimensionLimits) {
			return fmt.Errorf("simulation-exploration campaign plan is invalid")
		}
	default:
		return fmt.Errorf("campaign plan strategy is invalid")
	}
	if plan.ExecutionTimeoutNanos == 0 || plan.OverallTimeoutNanos == 0 || plan.OutputBytes == 0 || plan.WorldTransitionBytes == 0 || plan.TerminateGraceNanos > plan.ExecutionTimeoutNanos || plan.TerminateGraceNanos > plan.OverallTimeoutNanos {
		return fmt.Errorf("campaign plan limits are invalid")
	}
	if plan.OnFailure != "first" && plan.OnFailure != "budget" && plan.OnFailure != "all" || plan.FailureBudget == 0 {
		return fmt.Errorf("campaign plan failure policy is invalid")
	}
	if plan.RunnerBuild == "" || plan.Toolchain.GoVersion == "" || plan.Toolchain.BuildKey == "" || plan.Toolchain.TargetGOOS == "" || plan.Toolchain.TargetGOARCH == "" {
		return fmt.Errorf("campaign plan build identity is invalid")
	}
	preparedPath := filepath.ToSlash(filepath.Clean(filepath.FromSlash(plan.Prepared.Path)))
	if preparedPath != plan.Prepared.Path || !strings.HasPrefix(preparedPath, ".prepared/") || strings.Contains(preparedPath, "..") {
		return fmt.Errorf("campaign plan prepared target path is invalid")
	}
	target := plan.Prepared.Target
	if target.Kind == "" || target.Source == "" || !validRecordSHA256(target.SHA256) || target.Size == 0 || len(target.Argv) == 0 || target.Argv[0] == "" {
		return fmt.Errorf("campaign plan prepared target identity is invalid")
	}
	if err := record.ValidateCompatibilityPacks(target.Compatibility); err != nil {
		return fmt.Errorf("campaign plan prepared target: %w", err)
	}
	if err := record.ValidateTargetAdapters(target.Adapters); err != nil {
		return fmt.Errorf("campaign plan prepared target: %w", err)
	}
	if err := record.ValidateCurrentTargetCapability(target); err != nil {
		return fmt.Errorf("campaign plan prepared target: %w", err)
	}
	if plan.IOProfile.Name == "" || !validRecordSHA256(record.SHA256(plan.IOProfile.ImplementationSHA256)) || !validRecordSHA256(record.SHA256(plan.IOProfile.InventorySHA256)) {
		return fmt.Errorf("campaign plan I/O profile identity is invalid")
	}
	if choices := plan.ChoiceProfile; choices != nil {
		implementation, err := choice.ImplementationIdentity(plan.Toolchain.BuildKey)
		if err != nil || choices.Name != choice.Profile || choices.ImplementationSHA256 != record.SHA256FromSum(implementation) || choices.Limit < choice.MinimumTraceBytes || choices.Limit > choice.MaximumTraceBytes {
			return fmt.Errorf("campaign plan choice profile identity is invalid")
		}
	}
	if plan.Coverage != "none" && plan.Coverage != "semantic" && plan.Coverage != "choice" && plan.Coverage != "semantic+choice" ||
		(plan.Coverage == "none" || plan.Coverage == "choice") && len(plan.RequiredSemanticProbes) != 0 ||
		(plan.Coverage == "choice" || plan.Coverage == "semantic+choice") && plan.ChoiceProfile == nil {
		return fmt.Errorf("campaign plan coverage policy is invalid")
	}
	if !sort.StringsAreSorted(plan.RequiredSemanticProbes) {
		return fmt.Errorf("campaign plan required semantic probes are not sorted")
	}
	for index, probe := range plan.RequiredSemanticProbes {
		if probe == "" || index > 0 && plan.RequiredSemanticProbes[index-1] == probe {
			return fmt.Errorf("campaign plan required semantic probes are invalid")
		}
	}
	switch plan.KeepSuccesses {
	case "none":
		if plan.SuccessArtifactLimit != 0 || plan.SuccessBytesLimit != 0 {
			return fmt.Errorf("campaign plan disabled success retention has capacity")
		}
	case "novel":
		if plan.Coverage == "none" || plan.SuccessArtifactLimit == 0 || plan.SuccessBytesLimit == 0 {
			return fmt.Errorf("campaign plan novel success retention policy is invalid")
		}
	case "all":
		if plan.SuccessArtifactLimit == 0 || plan.SuccessBytesLimit == 0 {
			return fmt.Errorf("campaign plan success retention capacity is invalid")
		}
	default:
		return fmt.Errorf("campaign plan success retention policy is invalid")
	}
	if plan.Guidance != nil {
		corpus := filepath.Clean(plan.Guidance.Corpus)
		if !filepath.IsAbs(corpus) || corpus != plan.Guidance.Corpus || corpus == filepath.VolumeName(corpus)+string(filepath.Separator) || !validRecordSHA256(plan.Guidance.SnapshotSHA256) || plan.Coverage == "none" {
			return fmt.Errorf("campaign plan guidance identity is invalid")
		}
	}
	return nil
}

func hasCampaignPlanExplorationFields(plan CampaignPlan) bool {
	return plan.MaxExecutions != 0 || plan.MaxChoiceDepth != 0 || plan.MaxForcedDecisions != 0 || plan.MaxExplorationBytes != 0 || plan.MaxExplorationResultBytes != 0 || plan.SimulationDimensionLimits != (simulationengine.DimensionLimits{}) || plan.ChoiceExplorationImplementationSHA256 != "" || plan.SimulationExplorationImplementationSHA256 != ""
}

func validSimulationDimensionLimits(limits simulationengine.DimensionLimits) bool {
	return limits.Runtime != 0 && limits.Scenario != 0 && limits.Network != 0 && limits.Storage != 0 && limits.Fault != 0 && limits.Crash != 0
}

func ValidateCampaignPlan(plan CampaignPlan) error {
	return validateCampaignPlan(plan)
}
