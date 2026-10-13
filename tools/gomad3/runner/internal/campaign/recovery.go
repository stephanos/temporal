package campaign

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"go.temporal.io/server/tools/gomad3/artifact"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/internal/logstore"
)

type recoveryAction string

// recoveryFinalizePublication removes the prepared target a published
// campaign no longer needs, when publication's cleanup was interrupted.
const recoveryFinalizePublication recoveryAction = "finalize-publication"

type LifecycleStatus struct {
	State           lifecycleState `json:"state"`
	LastStableState lifecycleState `json:"last_stable_state"`
	Reason          string         `json:"reason,omitempty"`
	Detail          string         `json:"detail,omitempty"`
	Published       bool           `json:"published"`
	Resumable       bool           `json:"resumable"`
	Repairable      bool           `json:"repairable"`
	Action          recoveryAction `json:"action"`
}

type recoveryResult struct {
	Action  recoveryAction  `json:"action"`
	Changed bool            `json:"changed"`
	Before  LifecycleStatus `json:"before"`
	After   LifecycleStatus `json:"after"`
}

type ResumePreflight struct {
	Lifecycle LifecycleStatus
	Plan      CampaignPlan
}

// ResumeState is what a resumed campaign continues from: its frozen plan,
// committed outcomes and, for a campaign that admits ordinals, its durable
// admission frontier and receipts.
type ResumeState struct {
	Plan       CampaignPlan
	Executions []ExecutionRecord
	Frontier   Frontier
}

// InspectCampaignLifecycle replays a campaign log read-only and reports its
// lifecycle. It takes no lock and mutates nothing; a torn final record is
// observed, not repaired.
func InspectCampaignLifecycle(path string) (LifecycleStatus, error) {
	inspected, err := InspectCampaignStore(path)
	return inspected.Lifecycle, err
}

func lifecycleStatus(path string, state *campaignState) (LifecycleStatus, error) {
	status, _, err := publishedLifecycle(path, state)
	return status, err
}

// publishedLifecycle projects the replayed lifecycle and, for a published
// campaign, validates and returns the published campaign.
func publishedLifecycle(path string, state *campaignState) (LifecycleStatus, *Campaign, error) {
	status := state.status()
	if status.Published {
		opened, err := openPublishedState(path, state)
		if err != nil {
			return LifecycleStatus{}, nil, fmt.Errorf("validate published campaign lifecycle: %w", err)
		}
		stale, err := privateDirectoryPresent(filepath.Join(path, ".prepared"))
		if err != nil {
			return LifecycleStatus{}, nil, err
		}
		if stale {
			status.Repairable = true
			status.Action = recoveryFinalizePublication
		}
		return status, &opened, nil
	}
	if state.resumable() == nil {
		_, err := resumePlan(path, state)
		status.Resumable = err == nil
	}
	return status, nil, nil
}

func privateDirectoryPresent(path string) (bool, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("inspect published private state: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0o700 {
		return false, errors.New("published private state metadata is invalid")
	}
	return true, nil
}

// ReadResumePlan returns the frozen plan of a resumable campaign after
// checking the prepared target it names.
func ReadResumePlan(path string) (CampaignPlan, error) {
	if err := validateCampaignDirectory(path); err != nil {
		return CampaignPlan{}, fmt.Errorf("resumable campaign path must be a private directory: %w", err)
	}
	state, _, err := replayCampaign(path)
	if err != nil {
		return CampaignPlan{}, err
	}
	return resumePlan(path, state)
}

func resumePlan(path string, state *campaignState) (_ CampaignPlan, retErr error) {
	if err := state.resumable(); err != nil {
		return CampaignPlan{}, err
	}
	plan := *state.plan
	root, err := os.OpenRoot(path)
	if err != nil {
		return CampaignPlan{}, fmt.Errorf("pin resumable campaign directory: %w", err)
	}
	defer func() {
		retErr = errors.Join(retErr, root.Close())
	}()
	digest, size, err := hashValidatedFile(root, filepath.FromSlash(plan.Prepared.Path), 0o500, uint64(plan.Prepared.Target.Size))
	if err != nil || digest != plan.Prepared.Target.SHA256 || size != uint64(plan.Prepared.Target.Size) {
		return CampaignPlan{}, errors.Join(errors.New("prepared target identity does not match campaign plan"), err)
	}
	return plan, nil
}

func PreflightResume(path string) (ResumePreflight, error) {
	lifecycle, err := InspectCampaignLifecycle(path)
	if err != nil {
		return ResumePreflight{}, err
	}
	if lifecycle.Published || !lifecycle.Resumable {
		return ResumePreflight{}, newIntegrityError(fmt.Errorf("campaign lifecycle %q is not resumable", lifecycle.State))
	}
	plan, err := ReadResumePlan(path)
	if err != nil {
		return ResumePreflight{}, classifyIntegrityError(err)
	}
	return ResumePreflight{Lifecycle: lifecycle, Plan: plan}, nil
}

// openCampaignWriter takes the campaign log's exclusive lock and replays it,
// truncating a recognized torn final record. Callers validate the log
// read-only first, so an unsupported or corrupt log fails before this repair.
func openCampaignWriter(ctx context.Context, path string) (*logstore.Writer, *campaignState, logstore.Report, error) {
	store, err := campaignStorePath(path)
	if err != nil {
		return nil, nil, logstore.Report{}, err
	}
	state := newCampaignState()
	writer, report, err := logstore.OpenWriter(store, campaignLogOptions(logStoreHook(ctx), false), state.consume)
	if errors.Is(err, logstore.ErrInUse) {
		return nil, nil, report, fmt.Errorf("campaign is already being resumed: %w", err)
	}
	if err != nil {
		return nil, nil, report, classifyStoreError(fmt.Errorf("open campaign log: %w", err))
	}
	if !state.opened {
		return nil, nil, report, errors.Join(newIntegrityError(errors.New("campaign log has no envelope")), writer.Close())
	}
	return writer, state, report, nil
}

// ResumeCampaignJournal reopens an interrupted campaign for further
// executions. It validates the log read-only, takes the exclusive lock,
// validates every committed outcome and its evidence, commits the resume,
// and only then reclaims the owned payloads and attempt work that the
// committed state no longer references.
func ResumeCampaignJournal(ctx context.Context, path string) (_ *CampaignJournal, _ ResumeState, retErr error) {
	if ctx == nil {
		ctx = context.Background()
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, ResumeState{}, fmt.Errorf("resolve resumable campaign path: %w", err)
	}
	preflight, err := PreflightResume(path)
	if err != nil {
		return nil, ResumeState{}, err
	}
	writer, state, _, err := openCampaignWriter(ctx, path)
	if err != nil {
		return nil, ResumeState{}, err
	}
	journal := &CampaignJournal{ctx: ctx, path: path, writer: writer, state: state, started: true}
	defer func() {
		if retErr != nil {
			retErr = errors.Join(retErr, journal.Close())
		}
	}()
	plan, err := resumePlan(path, state)
	if err != nil {
		return nil, ResumeState{}, classifyIntegrityError(err)
	}
	if !samePlanIdentity(plan, preflight.Plan) {
		return nil, ResumeState{}, newIntegrityError(errors.New("campaign plan changed while acquiring its resume lock"))
	}
	retained, err := validateResumeExecutions(path, plan, state.executions, state.stopGroupMember)
	if err != nil {
		return nil, ResumeState{}, classifyIntegrityError(err)
	}
	journal.config = CampaignConfig{
		Root: filepath.Dir(filepath.Dir(path)), CampaignID: state.open.CampaignID, PlanSHA256: plan.PlanSHA256, Shard: cloneCampaignShard(plan.Shard), Strategy: plan.Strategy,
		Guidance: plan.Guidance, Selection: plan.Selection, SelectionCount: uint64(plan.SelectionCount), MaxExecutions: uint64(plan.MaxExecutions), Parallel: uint64(plan.Parallel),
		Journal: state.limits,
	}
	if err := validateLiveReferences(path, state); err != nil {
		return nil, ResumeState{}, err
	}
	if err := journal.transition(ctx, lifecycleRunning, "", nil); err != nil {
		return nil, ResumeState{}, err
	}
	if len(retained) != len(state.executions) {
		return nil, ResumeState{}, newIntegrityError(errors.New("resumed campaign outcomes differ from their validated set"))
	}
	if _, err := reclaimOwnedPayloads(ctx, path, state, true); err != nil {
		return nil, ResumeState{}, err
	}
	return journal, ResumeState{Plan: plan, Executions: append([]ExecutionRecord(nil), state.executions...), Frontier: state.frontierSnapshot()}, nil
}

// classifyStoreError marks damaged or unsupported storage as an integrity
// failure and leaves host I/O failures, which a retry may clear, unmarked.
func classifyStoreError(err error) error {
	if errors.Is(err, logstore.ErrCorrupt) || errors.Is(err, logstore.ErrInvalidStore) || errors.Is(err, ErrUnsupportedFormat) || errors.Is(err, artifact.ErrCorruptReference) {
		return newIntegrityError(err)
	}
	return err
}

func samePlanIdentity(left, right CampaignPlan) bool {
	leftBytes, leftErr := logstore.EncodeJSON(left)
	rightBytes, rightErr := logstore.EncodeJSON(right)
	return leftErr == nil && rightErr == nil && string(leftBytes) == string(rightBytes)
}

// RecoverCampaign repairs a campaign under its exclusive lock: it truncates a
// recognized torn final record, validates every live reference, reclaims the
// owned payloads the committed state no longer references and, for a
// published campaign, finishes removing its prepared target. Repeating it
// changes nothing further.
func RecoverCampaign(ctx context.Context, path string) (_ recoveryResult, retErr error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return recoveryResult{}, err
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return recoveryResult{}, fmt.Errorf("resolve recoverable campaign path: %w", err)
	}
	before, err := InspectCampaignLifecycle(absolute)
	if err != nil {
		return recoveryResult{}, err
	}
	if err := ctx.Err(); err != nil {
		return recoveryResult{}, err
	}
	if !before.Published && !before.Resumable {
		return recoveryResult{}, newIntegrityError(fmt.Errorf("campaign lifecycle %q is not recoverable", before.State))
	}
	writer, state, report, err := openCampaignWriter(ctx, absolute)
	if err != nil {
		return recoveryResult{}, err
	}
	defer func() {
		retErr = errors.Join(retErr, writer.Close())
	}()
	if err := ctx.Err(); err != nil {
		return recoveryResult{}, err
	}
	if state.published != nil {
		if _, err := openPublishedState(absolute, state); err != nil {
			return recoveryResult{}, err
		}
	} else {
		plan, err := resumePlan(absolute, state)
		if err != nil {
			return recoveryResult{}, classifyIntegrityError(err)
		}
		if _, err := validateResumeExecutions(absolute, plan, state.executions, state.stopGroupMember); err != nil {
			return recoveryResult{}, classifyIntegrityError(err)
		}
	}
	removed, err := reclaimOwnedPayloads(ctx, absolute, state, false)
	if err != nil {
		return recoveryResult{}, err
	}
	if state.published != nil {
		present, err := privateDirectoryPresent(filepath.Join(absolute, ".prepared"))
		if err != nil {
			return recoveryResult{}, err
		}
		if present {
			if err := removeCompletedPartialContext(ctx, filepath.Join(absolute, ".prepared"), "prepared-target"); err != nil {
				return recoveryResult{}, err
			}
			if err := syncDirectoryContext(ctx, absolute); err != nil {
				return recoveryResult{}, err
			}
			removed++
		}
	}
	after, err := lifecycleStatus(absolute, state)
	if err != nil {
		return recoveryResult{}, err
	}
	return recoveryResult{Action: before.Action, Changed: report.RepairedBytes != 0 || removed != 0, Before: before, After: after}, nil
}

// liveReferences projects the artifacts the committed campaign state
// references in each owned payload root.
func (state *campaignState) liveReferences() map[string]artifact.References {
	live := map[string]artifact.References{"failures": {}, "successes": {}}
	seen := map[string]struct{}{}
	add := func(reference *string) {
		if reference == nil {
			return
		}
		root, name := filepath.Split(filepath.FromSlash(*reference))
		root = filepath.Clean(root)
		references, owned := live[root]
		if !owned {
			return
		}
		if _, duplicate := seen[*reference]; duplicate {
			return
		}
		seen[*reference] = struct{}{}
		references.Artifacts = append(references.Artifacts, name)
		live[root] = references
	}
	for _, run := range state.executions {
		add(run.Artifact)
		add(run.SuccessArtifact)
	}
	state.pendingLiveReferences(add)
	live["diagnostics"] = artifact.References{Files: state.liveDiagnostics()}
	return live
}

// validateLiveReferences validates every payload the committed state and
// its durable receipts reference: owned failure and success artifacts and
// the diagnostic sidecars outcomes adopted or receipts still hold.
func validateLiveReferences(path string, state *campaignState) error {
	live := state.liveReferences()
	for _, root := range ownedPayloadRoots {
		if err := artifact.ValidateReferences(filepath.Join(path, root), live[root]); err != nil {
			return newIntegrityError(err)
		}
	}
	return nil
}

// ownedPayloadRoots are the campaign directory's exclusively owned payload
// roots: failure and success artifacts and diagnostic sidecars.
var ownedPayloadRoots = []string{"failures", "successes", "diagnostics"}

var executionWorkPattern = regexp.MustCompile(`^[0-9]{20}-[0-9]+$`)

// reclaimOwnedPayloads validates every live payload of the campaign's owned
// failure, success and diagnostic roots before it removes anything, then
// removes what neither a committed outcome nor a durable receipt references. With work it also removes the
// work directories of attempts whose outcomes were never committed. Shared
// target pools and exploration round stores are not swept here.
func reclaimOwnedPayloads(ctx context.Context, path string, state *campaignState, work bool) (int, error) {
	if err := validateLiveReferences(path, state); err != nil {
		return 0, err
	}
	live := state.liveReferences()
	removed := 0
	for _, root := range ownedPayloadRoots {
		if err := observeMutation(ctx, mutationDelete, "owned-payload-sweep"); err != nil {
			return removed, err
		}
		names, err := artifact.SweepOwnedRoot(filepath.Join(path, root), live[root])
		removed += len(names)
		if err != nil {
			return removed, classifyStoreError(fmt.Errorf("reclaim unreferenced %s: %w", root, err))
		}
	}
	if !work {
		return removed, nil
	}
	partialRoot := filepath.Join(path, ".partial")
	entries, err := os.ReadDir(partialRoot)
	if errors.Is(err, os.ErrNotExist) {
		return removed, nil
	}
	if err != nil {
		return removed, fmt.Errorf("read attempt work: %w", err)
	}
	reclaimed := 0
	for _, entry := range entries {
		if !entry.IsDir() || !executionWorkPattern.MatchString(entry.Name()) {
			continue
		}
		if err := removeCompletedPartialContext(ctx, filepath.Join(partialRoot, entry.Name()), "execution-partial"); err != nil {
			return removed, err
		}
		reclaimed++
	}
	if reclaimed != 0 {
		if err := syncDirectoryContext(ctx, partialRoot); err != nil {
			return removed, err
		}
	}
	return removed + reclaimed, nil
}

// validateResumeExecutions validates committed outcomes and their evidence
// and returns the ones a resumed campaign reuses. keepCancelled reports a
// cancellation committed in a stop group, which is reused; other Runner
// outcomes run again.
func validateResumeExecutions(batchPath string, plan CampaignPlan, runs []ExecutionRecord, keepCancelled func(uint64) bool) ([]ExecutionRecord, error) {
	strategy := plan.Strategy
	ordinalLimit := uint64(plan.SelectionCount)
	if strategy == "choice-exploration" || strategy == "simulation-exploration" {
		ordinalLimit = uint64(plan.MaxExecutions)
	}
	ordinals := make(map[uint64]struct{}, len(runs))
	candidates := make(map[record.SHA256]struct{}, len(runs))
	observedProbes := make(map[string]struct{})
	observedChoiceFeatures := make(map[string]struct{})
	retained := make([]ExecutionRecord, 0, len(runs))
	var retainedSuccesses, retainedSuccessBytes uint64
	failureArtifacts := make(map[record.SHA256]uint64)
	var failureArtifactBytes uint64
	for index, run := range runs {
		ordinal := uint64(run.SelectionOrdinal)
		if ordinal >= ordinalLimit {
			return nil, fmt.Errorf("resumable execution %d selection ordinal is out of range", index+1)
		}
		if plan.Shard != nil && ordinal%uint64(plan.Shard.Count) != uint64(plan.Shard.Index) {
			return nil, fmt.Errorf("resumable execution %d is outside its shard assignment", index+1)
		}
		if (strategy == "choice-exploration" || strategy == "simulation-exploration") && run.Strategy != strategy {
			return nil, fmt.Errorf("resumable exploration execution %d strategy is invalid", index+1)
		}
		if strategy == "choice-exploration" {
			baseSeed, parseErr := strconv.ParseUint(plan.Selection, 10, 64)
			if parseErr != nil || uint64(run.Seed) != baseSeed || ordinal != uint64(index) {
				return nil, fmt.Errorf("resumable exploration execution %d seed or logical ordinal is invalid", index+1)
			}
			if err := validateExplorationExecutionSummary(run, candidates); err != nil {
				return nil, fmt.Errorf("resumable exploration execution %d: %w", index+1, err)
			}
		} else if strategy == "simulation-exploration" {
			baseSeed, parseErr := strconv.ParseUint(plan.Selection, 10, 64)
			if parseErr != nil || uint64(run.Seed) != baseSeed || ordinal != uint64(index) {
				return nil, fmt.Errorf("resumable simulation exploration execution %d seed or logical ordinal is invalid", index+1)
			}
			if err := validateSimulationExplorationExecutionSummary(run, candidates); err != nil {
				return nil, fmt.Errorf("resumable simulation exploration execution %d: %w", index+1, err)
			}
		} else if run.Strategy != "" || run.Divergence != nil {
			return nil, fmt.Errorf("resumable seed execution %d contains strategy evidence", index+1)
		}
		if _, duplicate := ordinals[ordinal]; duplicate {
			return nil, fmt.Errorf("resumable selection ordinal is duplicated: %d", ordinal)
		}
		ordinals[ordinal] = struct{}{}
		if run.Reason == "" || (run.IOTranscriptSHA256 == nil) != (run.IOTranscriptRecords == nil) {
			return nil, fmt.Errorf("resumable execution %d identity is incomplete", index+1)
		}
		if run.IOTranscriptSHA256 != nil && !validRecordSHA256(*run.IOTranscriptSHA256) {
			return nil, fmt.Errorf("resumable execution %d transcript digest is invalid", index+1)
		}
		if err := validateChoiceExecutionSummary(run); err != nil {
			return nil, fmt.Errorf("resumable execution %d: %w", index+1, err)
		}
		if err := validateSemanticProbeLists(run.SemanticProbes, run.NovelSemanticProbes); err != nil {
			return nil, fmt.Errorf("resumable execution %d: %w", index+1, err)
		}
		if err := validateChoiceFeatureLists(run.ChoiceFeatures, run.NovelChoiceFeatures); err != nil {
			return nil, fmt.Errorf("resumable execution %d: %w", index+1, err)
		}
		switch run.Domain {
		case "success":
			if run.Termination != "exit" || run.FailureSignature != nil || run.Artifact != nil {
				return nil, fmt.Errorf("successful resumable execution %d has failure evidence", index+1)
			}
			if (run.SuccessArtifact == nil) != (run.SuccessArtifactBytes == nil) {
				return nil, fmt.Errorf("successful resumable execution %d has incomplete retained success evidence", index+1)
			}
			if run.SuccessArtifact == nil {
				if len(run.NovelSemanticProbes) != 0 || len(run.NovelChoiceFeatures) != 0 || plan.KeepSuccesses == "all" {
					return nil, fmt.Errorf("successful resumable execution %d violates its retention policy", index+1)
				}
			} else {
				if plan.KeepSuccesses == "none" || plan.KeepSuccesses == "novel" && len(run.NovelSemanticProbes) == 0 && len(run.NovelChoiceFeatures) == 0 || plan.KeepSuccesses == "all" && (len(run.NovelSemanticProbes) != 0 || len(run.NovelChoiceFeatures) != 0) {
					return nil, fmt.Errorf("retained successful execution %d violates its retention policy", index+1)
				}
				for _, probe := range run.NovelSemanticProbes {
					if _, found := observedProbes[probe]; found {
						return nil, fmt.Errorf("retained successful execution %d claims previously observed probe %q", index+1, probe)
					}
				}
				for _, feature := range run.NovelChoiceFeatures {
					if _, found := observedChoiceFeatures[feature]; found {
						return nil, fmt.Errorf("retained successful execution %d claims previously observed choice feature %q", index+1, feature)
					}
				}
				if err := validateResumeSuccessArtifact(batchPath, plan, run); err != nil {
					return nil, fmt.Errorf("validate resumable retained success %d: %w", index+1, err)
				}
				retainedSuccesses++
				if uint64(*run.SuccessArtifactBytes) > ^uint64(0)-retainedSuccessBytes {
					return nil, fmt.Errorf("retained success byte count overflows")
				}
				retainedSuccessBytes += uint64(*run.SuccessArtifactBytes)
				if retainedSuccesses > uint64(plan.SuccessArtifactLimit) || retainedSuccessBytes > uint64(plan.SuccessBytesLimit) {
					return nil, fmt.Errorf("retained successes exceed the campaign plan capacity")
				}
			}
			retained = append(retained, run)
		case "target", "watchdog":
			artifact, err := validateResumeArtifact(batchPath, plan, run)
			if err != nil {
				return nil, fmt.Errorf("validate resumable execution %d: %w", index+1, err)
			}
			if _, found := failureArtifacts[*run.FailureSignature]; !found {
				if artifact.StoredBytes > ^uint64(0)-failureArtifactBytes {
					return nil, errors.New("retained failure byte count overflows")
				}
				failureArtifacts[*run.FailureSignature] = artifact.StoredBytes
				failureArtifactBytes += artifact.StoredBytes
			}
			retained = append(retained, run)
		case "runner":
			if strategy == "choice-exploration" && run.Divergence != nil {
				retained = append(retained, run)
				continue
			}
			if run.Artifact != nil || run.FailureSignature != nil {
				artifact, err := validateResumeArtifact(batchPath, plan, run)
				if err != nil {
					return nil, fmt.Errorf("validate resumable Runner record %d: %w", index+1, err)
				}
				if _, found := failureArtifacts[*run.FailureSignature]; !found {
					if artifact.StoredBytes > ^uint64(0)-failureArtifactBytes {
						return nil, errors.New("retained failure byte count overflows")
					}
					failureArtifacts[*run.FailureSignature] = artifact.StoredBytes
					failureArtifactBytes += artifact.StoredBytes
				}
			} else if run.Reason != "runner_cancelled" || run.Termination != "none" {
				return nil, fmt.Errorf("resumable Runner record %d is invalid", index+1)
			} else if keepCancelled(ordinal) {
				retained = append(retained, run)
				continue
			}
			delete(ordinals, ordinal)
		default:
			return nil, fmt.Errorf("resumable execution %d domain is invalid: %s", index+1, run.Domain)
		}
		if limits := plan.Artifacts; limits != nil {
			if uint64(len(failureArtifacts)) > uint64(limits.FailureArtifacts) || failureArtifactBytes > uint64(limits.FailureBytes) || retainedSuccessBytes > ^uint64(0)-failureArtifactBytes || failureArtifactBytes+retainedSuccessBytes > uint64(limits.TotalBytes) {
				return nil, errors.New("retained artifacts exceed the campaign plan capacity")
			}
		}
		for _, probe := range run.SemanticProbes {
			observedProbes[probe] = struct{}{}
		}
		for _, feature := range run.ChoiceFeatures {
			observedChoiceFeatures[feature] = struct{}{}
		}
	}
	return retained, nil
}

func validateResumeSuccessArtifact(batchPath string, plan CampaignPlan, run ExecutionRecord) error {
	retained, err := ResolveRetainedEvidence(batchPath, filepath.Base(batchPath), run)
	if err != nil {
		return err
	}
	manifest := retained.Manifest
	if manifest.ReplayMode != record.ReplayExact && (manifest.Target.Backend == nil || manifest.ReplayMode != record.ReplayObserved || manifest.Target.Backend.ReplayMode != record.ReplayObserved) {
		return fmt.Errorf("retained success artifact execution identity does not match its journal")
	}
	if manifest.Outcome.Reason != run.Reason || manifest.Outcome.Termination != run.Termination {
		return fmt.Errorf("retained success artifact outcome does not match its journal")
	}
	targetMatches, targetErr := record.SamePreparedTargetIdentity(manifest.Target, plan.Prepared.Target)
	if targetErr != nil || manifest.Runner.RunnerBuild != plan.RunnerBuild || manifest.Toolchain != plan.Toolchain || !targetMatches {
		return fmt.Errorf("retained success artifact target identity does not match its campaign plan")
	}
	if !choiceProfileMatchesPlan(plan, manifest) {
		return fmt.Errorf("retained success artifact choice profile does not match its campaign plan")
	}
	return nil
}

func validateResumeArtifact(batchPath string, plan CampaignPlan, run ExecutionRecord) (retainedEvidence, error) {
	retained, err := ResolveRetainedEvidence(batchPath, filepath.Base(batchPath), run)
	if err != nil {
		return retainedEvidence{}, err
	}
	manifest := retained.Manifest
	targetMatches, targetErr := record.SamePreparedTargetIdentity(manifest.Target, plan.Prepared.Target)
	if targetErr != nil || manifest.Runner.RunnerBuild != plan.RunnerBuild || manifest.Toolchain != plan.Toolchain || !targetMatches {
		return retainedEvidence{}, fmt.Errorf("failure artifact target identity does not match its campaign plan")
	}
	if !choiceProfileMatchesPlan(plan, manifest) {
		return retainedEvidence{}, fmt.Errorf("failure artifact choice profile does not match its campaign plan")
	}
	return retained, nil
}

func choiceProfileMatchesPlan(plan CampaignPlan, manifest record.ExecutionRecord) bool {
	if plan.ChoiceProfile == nil || manifest.ChoiceProfile == nil {
		return plan.ChoiceProfile == nil && manifest.ChoiceProfile == nil
	}
	return plan.ChoiceProfile.Name == manifest.ChoiceProfile.Name &&
		plan.ChoiceProfile.ImplementationSHA256 == manifest.ChoiceProfile.ImplementationSHA256 &&
		plan.ChoiceProfile.Limit == manifest.ChoiceProfile.Trace.Limit
}
