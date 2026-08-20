package campaignstore

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"go.temporal.io/server/tools/gomadv3/evidence"
	"go.temporal.io/server/tools/gomadv3/internal/hostfs"
)

type ResumeState struct {
	Plan CampaignPlan
	Runs []ExecutionRecord
}

func ResumeCampaignJournal(ctx context.Context, path string) (_ *CampaignJournal, _ ResumeState, retErr error) {
	if ctx == nil {
		ctx = context.Background()
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, ResumeState{}, fmt.Errorf("resolve resumable batch path: %w", err)
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
	if plan.Schema == CampaignPlanSchema || plan.Schema == PreviousCampaignPlanSchema {
		return resumeSegmentedCampaignJournal(ctx, path, plan, lock)
	}
	runsPath := filepath.Join(path, "runs.jsonl")
	runsBytes, err := readResumeRuns(runsPath)
	if err != nil {
		return nil, ResumeState{}, err
	}
	runs, err := decodeExecutions(runsBytes)
	if err != nil {
		return nil, ResumeState{}, err
	}
	retained, err := validateResumeExecutions(path, plan, runs)
	if err != nil {
		return nil, ResumeState{}, err
	}
	retainedBytes, err := encodeExecutionRecords(retained)
	if err != nil {
		return nil, ResumeState{}, err
	}
	if err := archiveResumeState(ctx, path, runsBytes, len(retained) != len(runs), plan.Strategy == "choice-frontier"); err != nil {
		return nil, ResumeState{}, err
	}
	if len(retained) != len(runs) || !fileExists(runsPath) {
		if err := atomicWriteContext(ctx, runsPath, retainedBytes); err != nil {
			return nil, ResumeState{}, fmt.Errorf("rewrite resumable runs journal: %w", err)
		}
	}
	runsFile, err := openAppendOnly(runsPath)
	if err != nil {
		return nil, ResumeState{}, err
	}
	hasher := sha256.New()
	if _, err := hasher.Write(retainedBytes); err != nil {
		runsFile.Close()
		return nil, ResumeState{}, fmt.Errorf("hash resumable runs journal: %w", err)
	}
	strategy := plan.Strategy
	if strategy == "" {
		strategy = "seed"
	}
	journal := &CampaignJournal{
		ctx: ctx,
		config: CampaignConfig{
			Root: filepath.Dir(filepath.Dir(path)), CampaignID: filepath.Base(path), PlanSHA256: plan.PlanSHA256, Shard: cloneCampaignShard(plan.Shard), Strategy: strategy, Selection: plan.Selection, SelectionCount: uint64(plan.SelectionCount),
		},
		path: path, runsFile: runsFile, runsHasher: hasher, runsWriter: io.MultiWriter(runsFile, hasher), runsBytes: uint64(len(retainedBytes)), resumeLock: lock,
	}
	lifecycle, err := InspectCampaignLifecycle(path)
	if err != nil {
		return nil, ResumeState{}, errors.Join(err, journal.Close())
	}
	journal.lifecycle = lifecycle.State
	journal.lastStableLifecycle = lifecycle.LastStableState
	if lifecycle.State == LifecycleRunning {
		journal.lifecycle = LifecycleRecoverableFailure
		journal.lastStableLifecycle = LifecycleRunning
	}
	if err := journal.transitionLifecycle(LifecycleRunning, "", nil); err != nil {
		return nil, ResumeState{}, errors.Join(err, journal.Close())
	}
	return journal, ResumeState{Plan: plan, Runs: retained}, nil
}

func resumeSegmentedCampaignJournal(ctx context.Context, path string, plan CampaignPlan, lock *hostfs.Lock) (*CampaignJournal, ResumeState, error) {
	limits := runJournalLimitsFromPlan(*plan.Journal)
	index, closedRuns, activeRuns, err := readResumableRunJournal(path, limits)
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
			return nil, ResumeState{}, newIntegrityError(errors.New("closed run segment contains a discardable resume record"))
		}
	}
	if err := archiveResumeState(ctx, path, nil, false, plan.Strategy == "choice-frontier"); err != nil {
		return nil, ResumeState{}, err
	}
	if err := makePrivateDirectoriesContext(ctx, filepath.Join(path, ".partial", "runs")); err != nil {
		return nil, ResumeState{}, err
	}
	segmented := &segmentedRunJournal{
		ctx: ctx, batchPath: path, limits: limits, segments: append([]runJournalSegment(nil), index.Segments...),
		totalRecords: uint64(index.Records), totalBytes: uint64(index.Bytes),
	}
	if err := segmented.writeIndex(); err != nil {
		return nil, ResumeState{}, err
	}
	for _, run := range retained[len(closedRuns):] {
		encoded, err := evidence.CanonicalJSON(run)
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
	strategy := plan.Strategy
	if strategy == "" {
		strategy = "seed"
	}
	artifacts := *plan.Artifacts
	journal := &CampaignJournal{
		ctx: ctx,
		config: CampaignConfig{
			Root: filepath.Dir(filepath.Dir(path)), CampaignID: filepath.Base(path), PlanSHA256: plan.PlanSHA256, Shard: cloneCampaignShard(plan.Shard), Strategy: strategy,
			Selection: plan.Selection, SelectionCount: uint64(plan.SelectionCount), MaxRuns: uint64(plan.MaxRuns), Parallel: uint64(plan.Parallel), Journal: limits,
		},
		path: path, segmentedRuns: segmented, artifactPlan: &artifacts, resumeLock: lock,
	}
	if err := restoreRunningLifecycle(journal, path); err != nil {
		return nil, ResumeState{}, errors.Join(err, journal.Close())
	}
	return journal, ResumeState{Plan: plan, Runs: retained}, nil
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

func readResumeRuns(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return []byte{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("inspect resumable runs journal: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0o600 || info.Size() > maximumRunsBytes {
		return nil, fmt.Errorf("resumable runs journal metadata is invalid")
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read resumable runs journal: %w", err)
	}
	return contents, nil
}

func validateResumeExecutions(batchPath string, plan CampaignPlan, runs []ExecutionRecord) ([]ExecutionRecord, error) {
	strategy := plan.Strategy
	if strategy == "" {
		strategy = "seed"
	}
	ordinalLimit := uint64(plan.SelectionCount)
	if strategy == "choice-frontier" {
		ordinalLimit = uint64(plan.MaxRuns)
	}
	ordinals := make(map[uint64]struct{}, len(runs))
	candidates := make(map[evidence.SHA256]struct{}, len(runs))
	observedProbes := make(map[string]struct{})
	observedChoiceFeatures := make(map[string]struct{})
	retained := make([]ExecutionRecord, 0, len(runs))
	var retainedSuccesses, retainedSuccessBytes uint64
	failureArtifacts := make(map[evidence.SHA256]uint64)
	var failureArtifactBytes uint64
	for index, run := range runs {
		ordinal := uint64(run.SelectionOrdinal)
		if ordinal >= ordinalLimit {
			return nil, fmt.Errorf("resumable run %d selection ordinal is out of range", index+1)
		}
		if plan.Shard != nil && ordinal%uint64(plan.Shard.Count) != uint64(plan.Shard.Index) {
			return nil, fmt.Errorf("resumable run %d is outside its shard assignment", index+1)
		}
		if strategy == "choice-frontier" && run.Strategy != "choice-frontier" {
			return nil, fmt.Errorf("resumable frontier run %d strategy is invalid", index+1)
		}
		if strategy == "choice-frontier" {
			baseSeed, parseErr := strconv.ParseUint(plan.Selection, 10, 64)
			if parseErr != nil || uint64(run.Seed) != baseSeed || ordinal != uint64(index) {
				return nil, fmt.Errorf("resumable frontier run %d seed or logical ordinal is invalid", index+1)
			}
			if err := validateFrontierExecutionSummary(run, candidates); err != nil {
				return nil, fmt.Errorf("resumable frontier run %d: %w", index+1, err)
			}
		} else if run.Strategy != "" {
			return nil, fmt.Errorf("resumable seed run %d contains strategy evidence", index+1)
		}
		if _, duplicate := ordinals[ordinal]; duplicate {
			return nil, fmt.Errorf("resumable selection ordinal is duplicated: %d", ordinal)
		}
		ordinals[ordinal] = struct{}{}
		if run.Reason == "" || (run.IOTranscriptSHA256 == nil) != (run.IOTranscriptRecords == nil) {
			return nil, fmt.Errorf("resumable run %d identity is incomplete", index+1)
		}
		if run.IOTranscriptSHA256 != nil && !validRecordSHA256(*run.IOTranscriptSHA256) {
			return nil, fmt.Errorf("resumable run %d transcript digest is invalid", index+1)
		}
		if err := validateChoiceExecutionSummary(run); err != nil {
			return nil, fmt.Errorf("resumable run %d: %w", index+1, err)
		}
		if err := validateSemanticProbeLists(run.SemanticProbes, run.NovelSemanticProbes); err != nil {
			return nil, fmt.Errorf("resumable run %d: %w", index+1, err)
		}
		if err := validateChoiceFeatureLists(run.ChoiceFeatures, run.NovelChoiceFeatures); err != nil {
			return nil, fmt.Errorf("resumable run %d: %w", index+1, err)
		}
		switch run.Domain {
		case "success":
			if run.Termination != "exit" || run.FailureSignature != nil || run.Artifact != nil {
				return nil, fmt.Errorf("successful resumable run %d has failure evidence", index+1)
			}
			if (run.SuccessArtifact == nil) != (run.SuccessArtifactBytes == nil) {
				return nil, fmt.Errorf("successful resumable run %d has incomplete retained success evidence", index+1)
			}
			if run.SuccessArtifact == nil {
				if len(run.NovelSemanticProbes) != 0 || len(run.NovelChoiceFeatures) != 0 || plan.KeepSuccesses == "all" {
					return nil, fmt.Errorf("successful resumable run %d violates its retention policy", index+1)
				}
			} else {
				if plan.KeepSuccesses == "none" || plan.KeepSuccesses == "novel" && len(run.NovelSemanticProbes) == 0 && len(run.NovelChoiceFeatures) == 0 || plan.KeepSuccesses == "all" && (len(run.NovelSemanticProbes) != 0 || len(run.NovelChoiceFeatures) != 0) {
					return nil, fmt.Errorf("retained successful run %d violates its retention policy", index+1)
				}
				for _, probe := range run.NovelSemanticProbes {
					if _, found := observedProbes[probe]; found {
						return nil, fmt.Errorf("retained successful run %d claims previously observed probe %q", index+1, probe)
					}
				}
				for _, feature := range run.NovelChoiceFeatures {
					if _, found := observedChoiceFeatures[feature]; found {
						return nil, fmt.Errorf("retained successful run %d claims previously observed choice feature %q", index+1, feature)
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
					return nil, fmt.Errorf("retained successes exceed the batch plan capacity")
				}
			}
			retained = append(retained, run)
		case "target", "watchdog":
			artifact, err := validateResumeArtifact(batchPath, plan, run)
			if err != nil {
				return nil, fmt.Errorf("validate resumable run %d: %w", index+1, err)
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
			return nil, fmt.Errorf("resumable run %d domain is invalid: %s", index+1, run.Domain)
		}
		if limits := plan.Artifacts; limits != nil {
			if uint64(len(failureArtifacts)) > uint64(limits.FailureArtifacts) || failureArtifactBytes > uint64(limits.FailureBytes) || retainedSuccessBytes > ^uint64(0)-failureArtifactBytes || failureArtifactBytes+retainedSuccessBytes > uint64(limits.TotalBytes) {
				return nil, errors.New("retained artifacts exceed the batch plan capacity")
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
	if manifest.ReplayMode != evidence.ReplayExact {
		return fmt.Errorf("retained success artifact run identity does not match its journal")
	}
	if manifest.Outcome.Reason != run.Reason || manifest.Outcome.Termination != run.Termination {
		return fmt.Errorf("retained success artifact outcome does not match its journal")
	}
	targetMatches, targetErr := evidence.SameTargetIdentity(manifest.Target, plan.Prepared.Target)
	if targetErr != nil || manifest.Runner.RunnerBuild != plan.RunnerBuild || manifest.Toolchain != plan.Toolchain || !targetMatches {
		return fmt.Errorf("retained success artifact target identity does not match its batch plan")
	}
	if !choiceProfileMatchesPlan(plan, manifest) {
		return fmt.Errorf("retained success artifact choice profile does not match its batch plan")
	}
	return nil
}

func validateResumeArtifact(batchPath string, plan CampaignPlan, run ExecutionRecord) (RetainedEvidence, error) {
	retained, err := ResolveRetainedEvidence(batchPath, filepath.Base(batchPath), run)
	if err != nil {
		return RetainedEvidence{}, err
	}
	manifest := retained.Manifest
	targetMatches, targetErr := evidence.SameTargetIdentity(manifest.Target, plan.Prepared.Target)
	if targetErr != nil || manifest.Runner.RunnerBuild != plan.RunnerBuild || manifest.Toolchain != plan.Toolchain || !targetMatches {
		return RetainedEvidence{}, fmt.Errorf("failure artifact target identity does not match its batch plan")
	}
	if !choiceProfileMatchesPlan(plan, manifest) {
		return RetainedEvidence{}, fmt.Errorf("failure artifact choice profile does not match its batch plan")
	}
	return retained, nil
}

func choiceProfileMatchesPlan(plan CampaignPlan, manifest evidence.ExecutionRecord) bool {
	if plan.ChoiceProfile == nil || manifest.ChoiceProfile == nil {
		return plan.ChoiceProfile == nil && manifest.ChoiceProfile == nil
	}
	return plan.ChoiceProfile.Name == manifest.ChoiceProfile.Name &&
		plan.ChoiceProfile.ImplementationSHA256 == manifest.ChoiceProfile.ImplementationSHA256 &&
		plan.ChoiceProfile.Limit == manifest.ChoiceProfile.Trace.Limit
}

func encodeExecutionRecords(runs []ExecutionRecord) ([]byte, error) {
	values := make([]any, len(runs))
	for index := range runs {
		values[index] = runs[index]
	}
	return evidence.CanonicalJSONLines(values)
}

func archiveResumeState(ctx context.Context, path string, runs []byte, archiveRuns, preserveFrontier bool) error {
	partialRoot := filepath.Join(path, ".partial")
	entries, err := os.ReadDir(partialRoot)
	if err != nil {
		return fmt.Errorf("read partial batch state: %w", err)
	}
	toArchive := make([]os.DirEntry, 0)
	for _, entry := range entries {
		if entry.Name() == "batch" || entry.Name() == "resume" || preserveFrontier && entry.Name() == "frontier" {
			continue
		}
		info, err := entry.Info()
		if err != nil || info.Mode()&os.ModeSymlink != 0 || info.IsDir() && info.Mode().Perm() != 0o700 || !info.IsDir() && (info.Mode().Perm() != 0o600 || !strings.HasPrefix(entry.Name(), ".tmp-")) {
			return errors.Join(fmt.Errorf("partial batch entry %q is invalid", entry.Name()), err)
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
		if err := atomicWriteContext(ctx, filepath.Join(attemptPath, "runs.jsonl"), runs); err != nil {
			return err
		}
	}
	for _, entry := range toArchive {
		if err := renameContext(ctx, filepath.Join(partialRoot, entry.Name()), filepath.Join(partialsPath, entry.Name()), "resume-archive"); err != nil {
			return fmt.Errorf("archive partial batch entry %q: %w", entry.Name(), err)
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

func openAppendOnly(path string) (*os.File, error) {
	before, err := os.Lstat(path)
	if err != nil || !before.Mode().IsRegular() || before.Mode()&os.ModeSymlink != 0 || before.Mode().Perm() != 0o600 {
		return nil, errors.Join(fmt.Errorf("resumable runs journal metadata is invalid"), err)
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, err
	}
	after, err := file.Stat()
	if err != nil || !os.SameFile(before, after) {
		return nil, errors.Join(fmt.Errorf("resumable runs journal changed while opening"), err, file.Close())
	}
	return file, nil
}

func fileExists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}
