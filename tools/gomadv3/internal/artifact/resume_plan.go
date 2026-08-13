package artifact

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go.temporal.io/server/tools/gomadv3/internal/record"
)

const BatchPlanSchema = "gomadv3.batch-plan/v1"

const maximumBatchPlanBytes = 1 << 20

type PreparedTargetPlan struct {
	Path   string        `json:"path"`
	Target record.Target `json:"target"`
}

type IOProfilePlan struct {
	Name                 string        `json:"name"`
	ImplementationSHA256 record.SHA256 `json:"implementation_sha256"`
	InventorySHA256      record.SHA256 `json:"inventory_sha256"`
}

type GuidancePlan struct {
	Corpus         string        `json:"corpus"`
	SnapshotSHA256 record.SHA256 `json:"snapshot_sha256"`
}

type BatchPlan struct {
	Schema                 string                     `json:"schema"`
	Selection              string                     `json:"selection"`
	SelectionCount         record.Uint64String        `json:"selection_count"`
	Parallel               record.Uint64String        `json:"parallel"`
	RunTimeoutNanos        record.Uint64String        `json:"run_timeout_nanos"`
	OverallTimeoutNanos    record.Uint64String        `json:"overall_timeout_nanos"`
	TerminateGraceNanos    record.Uint64String        `json:"terminate_grace_nanos"`
	OnFailure              string                     `json:"on_failure"`
	FailureBudget          record.Uint64String        `json:"failure_budget"`
	OutputBytes            record.Uint64String        `json:"output_bytes"`
	WorldTransitionBytes   record.Uint64String        `json:"world_transition_bytes"`
	RunnerBuild            string                     `json:"runner_build"`
	Toolchain              record.Toolchain           `json:"toolchain"`
	Prepared               PreparedTargetPlan         `json:"prepared"`
	IOProfile              IOProfilePlan              `json:"io_profile"`
	Environment            []record.Environment       `json:"environment"`
	IOROMounts             []string                   `json:"io_ro_mounts"`
	IOROMountLimits        record.ReadOnlyMountLimits `json:"io_ro_mount_limits"`
	Coverage               string                     `json:"coverage"`
	RequiredSemanticProbes []string                   `json:"required_semantic_probes"`
	KeepSuccesses          string                     `json:"keep_successes"`
	SuccessArtifactLimit   record.Uint64String        `json:"success_artifact_limit"`
	SuccessBytesLimit      record.Uint64String        `json:"success_bytes_limit"`
	Guidance               *GuidancePlan              `json:"guidance,omitempty"`
}

func (journal *BatchJournal) RecordPlan(plan BatchPlan) error {
	if err := validateBatchPlan(plan); err != nil {
		return err
	}
	if plan.Selection != journal.config.Selection || uint64(plan.SelectionCount) != journal.config.SelectionCount {
		return fmt.Errorf("batch plan selection does not match journal")
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
	encoded, err := record.CanonicalJSON(plan)
	if err != nil {
		return fmt.Errorf("encode batch plan: %w", err)
	}
	path := filepath.Join(journal.PreparedPath(), "plan.json")
	if _, err := os.Lstat(path); err == nil {
		return fmt.Errorf("batch plan already exists")
	} else if !os.IsNotExist(err) {
		return err
	}
	return atomicWriteContext(journal.ctx, path, encoded)
}

func ReadResumePlan(path string) (BatchPlan, error) {
	rootInfo, err := os.Lstat(path)
	if err != nil {
		return BatchPlan{}, fmt.Errorf("open resumable batch directory: %w", err)
	}
	if !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 || rootInfo.Mode().Perm() != 0o700 {
		return BatchPlan{}, fmt.Errorf("resumable batch path must be a private directory")
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return BatchPlan{}, fmt.Errorf("pin resumable batch directory: %w", err)
	}
	defer root.Close()
	pinnedInfo, err := root.Stat(".")
	if err != nil || !os.SameFile(rootInfo, pinnedInfo) {
		return BatchPlan{}, errors.Join(fmt.Errorf("resumable batch directory changed while opening"), err)
	}
	if _, err := root.Stat("batch.json"); err == nil {
		return BatchPlan{}, fmt.Errorf("published batch cannot be resumed")
	} else if !os.IsNotExist(err) {
		return BatchPlan{}, fmt.Errorf("inspect batch publication state: %w", err)
	}
	if _, err := root.Stat(filepath.Join(".partial", "preparation")); err == nil {
		return BatchPlan{}, fmt.Errorf("batch preparation did not complete")
	} else if !os.IsNotExist(err) {
		return BatchPlan{}, fmt.Errorf("inspect batch preparation state: %w", err)
	}
	if err := validateResumeLifecycle(root); err != nil {
		return BatchPlan{}, err
	}
	planBytes, err := readValidatedFile(root, filepath.Join(".prepared", "plan.json"), 0o600, maximumBatchPlanBytes)
	if err != nil {
		return BatchPlan{}, fmt.Errorf("read batch plan: %w", err)
	}
	var plan BatchPlan
	if err := record.StrictDecode(planBytes, &plan); err != nil {
		return BatchPlan{}, fmt.Errorf("decode batch plan: %w", err)
	}
	canonical, err := record.CanonicalJSON(plan)
	if err != nil || !bytes.Equal(canonical, planBytes) {
		return BatchPlan{}, errors.Join(fmt.Errorf("batch plan is not canonical"), err)
	}
	if err := validateBatchPlan(plan); err != nil {
		return BatchPlan{}, err
	}
	digest, size, err := hashValidatedFile(root, filepath.FromSlash(plan.Prepared.Path), 0o500, uint64(plan.Prepared.Target.Size))
	if err != nil || digest != plan.Prepared.Target.SHA256 || size != uint64(plan.Prepared.Target.Size) {
		return BatchPlan{}, errors.Join(fmt.Errorf("prepared target identity does not match batch plan"), err)
	}
	return plan, nil
}

func validateResumeLifecycle(root *os.Root) error {
	contents, err := readValidatedFile(root, filepath.Join(".partial", "batch", "partial.json"), 0o600, maximumBatchPlanBytes)
	if err != nil {
		return fmt.Errorf("read batch lifecycle: %w", err)
	}
	var lifecycle struct {
		SchemaVersion uint32  `json:"schema_version"`
		State         string  `json:"state"`
		Reason        *string `json:"reason"`
		Detail        *string `json:"detail"`
	}
	if err := record.StrictDecode(contents, &lifecycle); err != nil {
		return fmt.Errorf("decode batch lifecycle: %w", err)
	}
	canonical, err := record.CanonicalJSON(lifecycle)
	if err != nil || !bytes.Equal(canonical, contents) {
		return errors.Join(fmt.Errorf("batch lifecycle is not canonical"), err)
	}
	if lifecycle.SchemaVersion != record.SchemaVersion || lifecycle.State != "running" && lifecycle.State != "failed" {
		return fmt.Errorf("batch lifecycle is not resumable")
	}
	return nil
}

func validateBatchPlan(plan BatchPlan) error {
	if plan.Schema != BatchPlanSchema || plan.Selection == "" || plan.SelectionCount == 0 || plan.Parallel == 0 {
		return fmt.Errorf("batch plan identity is invalid")
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
	if err := record.ValidateCompatibilityPacks(target.Compatibility); err != nil {
		return fmt.Errorf("batch plan prepared target: %w", err)
	}
	if err := record.ValidateTargetAdapters(target.Adapters); err != nil {
		return fmt.Errorf("batch plan prepared target: %w", err)
	}
	if plan.IOProfile.Name == "" || !validRecordSHA256(plan.IOProfile.ImplementationSHA256) || !validRecordSHA256(plan.IOProfile.InventorySHA256) {
		return fmt.Errorf("batch plan I/O profile identity is invalid")
	}
	if plan.Coverage != "none" && plan.Coverage != "semantic" || plan.Coverage == "none" && len(plan.RequiredSemanticProbes) != 0 {
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
		if plan.Coverage != "semantic" || plan.SuccessArtifactLimit == 0 || plan.SuccessBytesLimit == 0 {
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
		if !filepath.IsAbs(corpus) || corpus != plan.Guidance.Corpus || corpus == filepath.VolumeName(corpus)+string(filepath.Separator) || !validRecordSHA256(plan.Guidance.SnapshotSHA256) || plan.Coverage != "semantic" {
			return fmt.Errorf("batch plan guidance identity is invalid")
		}
	}
	return nil
}
