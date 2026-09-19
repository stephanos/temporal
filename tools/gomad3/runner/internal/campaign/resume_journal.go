package campaign

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomad3/internal/hostfs"
	"go.temporal.io/server/tools/gomad3/record"
)

type ResumeState struct {
	Plan       CampaignPlan
	Executions []ExecutionRecord
}

func ResumeCampaignJournal(ctx context.Context, path string) (_ *CampaignJournal, _ ResumeState, retErr error) {
	if ctx == nil {
		ctx = context.Background()
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, ResumeState{}, fmt.Errorf("resolve resumable campaign path: %w", err)
	}
	lock, err := acquireResumeLock(ctx, filepath.Join(path, ".resume.lock"))
	if err != nil {
		return nil, ResumeState{}, err
	}
	defer func() {
		if retErr != nil {
			retErr = errors.Join(retErr, releaseResumeLock(lock))
		}
	}()
	preflight, err := PreflightResume(path)
	if err != nil {
		return nil, ResumeState{}, err
	}
	if preflight.Lifecycle.Action == RecoveryRestoreRunning {
		if err := restoreInterruptedCommit(ctx, path); err != nil {
			return nil, ResumeState{}, err
		}
		preflight, err = PreflightResume(path)
		if err != nil {
			return nil, ResumeState{}, err
		}
	}
	plan := preflight.Plan
	return resumeSegmentedCampaignJournal(ctx, path, plan, lock)
}

func resumeSegmentedCampaignJournal(ctx context.Context, path string, plan CampaignPlan, lock *hostfs.Lock) (*CampaignJournal, ResumeState, error) {
	limits := executionJournalLimitsFromPlan(*plan.Journal)
	index, closedRuns, activeRuns, err := readResumableExecutionJournal(path, limits)
	if err != nil {
		return nil, ResumeState{}, err
	}
	allRuns := append(append([]ExecutionRecord(nil), closedRuns...), activeRuns...)
	retained, err := validateResumeExecutions(path, plan, allRuns)
	if err != nil {
		return nil, ResumeState{}, err
	}
	for index := range closedRuns {
		if index >= len(retained) || !reflect.DeepEqual(retained[index], closedRuns[index]) {
			return nil, ResumeState{}, newIntegrityError(errors.New("closed execution segment contains a discardable resume record"))
		}
	}
	if err := archiveResumeState(ctx, path, nil, false, plan.Strategy); err != nil {
		return nil, ResumeState{}, err
	}
	if err := makePrivateDirectoriesContext(ctx, filepath.Join(path, ".partial", "executions")); err != nil {
		return nil, ResumeState{}, err
	}
	segmented := &segmentedExecutionJournal{
		ctx: ctx, campaignPath: path, limits: limits, segments: append([]executionJournalSegment(nil), index.Segments...),
		totalRecords: uint64(index.Records), totalBytes: uint64(index.Bytes),
	}
	if err := segmented.writeIndex(); err != nil {
		return nil, ResumeState{}, err
	}
	for _, run := range retained[len(closedRuns):] {
		encoded, err := canonicaljson.CanonicalJSON(run)
		if err != nil {
			return nil, ResumeState{}, errors.Join(err, segmented.close())
		}
		if err := segmented.append(append(encoded, '\n')); err != nil {
			return nil, ResumeState{}, errors.Join(err, segmented.close())
		}
	}
	if len(retained) != len(closedRuns) {
		if err := segmented.seal(); err != nil {
			return nil, ResumeState{}, errors.Join(err, segmented.close())
		}
	}
	artifacts := *plan.Artifacts
	journal := &CampaignJournal{
		ctx: ctx,
		config: CampaignConfig{
			Root: filepath.Dir(filepath.Dir(path)), CampaignID: filepath.Base(path), PlanSHA256: plan.PlanSHA256, Shard: cloneCampaignShard(plan.Shard), Strategy: plan.Strategy,
			Selection: plan.Selection, SelectionCount: uint64(plan.SelectionCount), MaxExecutions: uint64(plan.MaxExecutions), Parallel: uint64(plan.Parallel), Journal: limits,
		},
		path: path, segmentedRuns: segmented, artifactPlan: &artifacts, resumeLock: lock,
	}
	if err := restoreRunningLifecycle(journal, path); err != nil {
		return nil, ResumeState{}, errors.Join(err, journal.Close())
	}
	return journal, ResumeState{Plan: plan, Executions: retained}, nil
}

func restoreRunningLifecycle(journal *CampaignJournal, path string) error {
	lifecycle, err := InspectCampaignLifecycle(path)
	if err != nil {
		return err
	}
	journal.lifecycle = lifecycle.State
	journal.lastStableLifecycle = lifecycle.LastStableState
	if lifecycle.State == LifecycleRunning {
		journal.lifecycle = LifecycleRecoverableFailure
		journal.lastStableLifecycle = LifecycleRunning
	}
	return journal.transitionLifecycle(LifecycleRunning, "", nil)
}

func validateResumeExecutions(batchPath string, plan CampaignPlan, runs []ExecutionRecord) ([]ExecutionRecord, error) {
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
		} else if run.Strategy != "" {
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
	if manifest.ReplayMode != record.ReplayExact {
		return fmt.Errorf("retained success artifact execution identity does not match its journal")
	}
	if manifest.Outcome.Reason != run.Reason || manifest.Outcome.Termination != run.Termination {
		return fmt.Errorf("retained success artifact outcome does not match its journal")
	}
	targetMatches, targetErr := record.SameTargetIdentity(manifest.Target, plan.Prepared.Target)
	if targetErr != nil || manifest.Runner.RunnerBuild != plan.RunnerBuild || manifest.Toolchain != plan.Toolchain || !targetMatches {
		return fmt.Errorf("retained success artifact target identity does not match its campaign plan")
	}
	if !choiceProfileMatchesPlan(plan, manifest) {
		return fmt.Errorf("retained success artifact choice profile does not match its campaign plan")
	}
	return nil
}

func validateResumeArtifact(batchPath string, plan CampaignPlan, run ExecutionRecord) (RetainedEvidence, error) {
	retained, err := ResolveRetainedEvidence(batchPath, filepath.Base(batchPath), run)
	if err != nil {
		return RetainedEvidence{}, err
	}
	manifest := retained.Manifest
	targetMatches, targetErr := record.SameTargetIdentity(manifest.Target, plan.Prepared.Target)
	if targetErr != nil || manifest.Runner.RunnerBuild != plan.RunnerBuild || manifest.Toolchain != plan.Toolchain || !targetMatches {
		return RetainedEvidence{}, fmt.Errorf("failure artifact target identity does not match its campaign plan")
	}
	if !choiceProfileMatchesPlan(plan, manifest) {
		return RetainedEvidence{}, fmt.Errorf("failure artifact choice profile does not match its campaign plan")
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

func archiveResumeState(ctx context.Context, path string, runs []byte, archiveRuns bool, strategy string) error {
	partialRoot := filepath.Join(path, ".partial")
	entries, err := os.ReadDir(partialRoot)
	if err != nil {
		return fmt.Errorf("read partial campaign state: %w", err)
	}
	toArchive := make([]os.DirEntry, 0)
	for _, entry := range entries {
		if entry.Name() == "campaign" || entry.Name() == "resume" || strategy == "choice-exploration" && entry.Name() == "choice-exploration" || strategy == "simulation-exploration" && entry.Name() == "simulation-exploration" {
			continue
		}
		info, err := entry.Info()
		if err != nil || info.Mode()&os.ModeSymlink != 0 || info.IsDir() && info.Mode().Perm() != 0o700 || !info.IsDir() && (info.Mode().Perm() != 0o600 || !strings.HasPrefix(entry.Name(), ".tmp-")) {
			return errors.Join(fmt.Errorf("partial campaign entry %q is invalid", entry.Name()), err)
		}
		toArchive = append(toArchive, entry)
	}
	if len(toArchive) == 0 && !archiveRuns {
		return nil
	}
	resumeRoot := filepath.Join(partialRoot, "resume")
	if err := makePrivateDirectoriesContext(ctx, resumeRoot); err != nil {
		return err
	}
	attempt, err := nextResumeAttempt(resumeRoot)
	if err != nil {
		return err
	}
	attemptPath := filepath.Join(resumeRoot, fmt.Sprintf("%06d", attempt))
	partialsPath := filepath.Join(attemptPath, "partials")
	if err := makePrivateDirectoriesContext(ctx, partialsPath); err != nil {
		return err
	}
	if archiveRuns {
		if err := atomicWriteContext(ctx, filepath.Join(attemptPath, "executions.jsonl"), runs); err != nil {
			return err
		}
	}
	for _, entry := range toArchive {
		if err := renameContext(ctx, filepath.Join(partialRoot, entry.Name()), filepath.Join(partialsPath, entry.Name()), "resume-archive"); err != nil {
			return fmt.Errorf("archive partial campaign entry %q: %w", entry.Name(), err)
		}
	}
	return syncDirectoryContext(ctx, partialRoot)
}

func nextResumeAttempt(root string) (uint64, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return 0, err
	}
	var maximum uint64
	for _, entry := range entries {
		if !entry.IsDir() {
			return 0, fmt.Errorf("resume archive entry %q is not a directory", entry.Name())
		}
		value, err := strconv.ParseUint(entry.Name(), 10, 64)
		if err != nil || value == 0 {
			return 0, fmt.Errorf("resume archive entry %q is invalid", entry.Name())
		}
		if value > maximum {
			maximum = value
		}
	}
	return maximum + 1, nil
}
