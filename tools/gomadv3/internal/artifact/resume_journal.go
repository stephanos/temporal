package artifact

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"go.temporal.io/server/tools/gomadv3/internal/record"
)

type ResumeState struct {
	Plan BatchPlan
	Runs []RunRecord
}

func ResumeBatchJournal(ctx context.Context, path string) (_ *BatchJournal, _ ResumeState, retErr error) {
	if ctx == nil {
		ctx = context.Background()
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, ResumeState{}, fmt.Errorf("resolve resumable batch path: %w", err)
	}
	lock, err := acquireResumeLock(filepath.Join(path, ".resume.lock"))
	if err != nil {
		return nil, ResumeState{}, err
	}
	defer func() {
		if retErr != nil {
			retErr = errors.Join(retErr, releaseResumeLock(lock))
		}
	}()
	plan, err := ReadResumePlan(path)
	if err != nil {
		return nil, ResumeState{}, err
	}
	runsPath := filepath.Join(path, "runs.jsonl")
	runsBytes, err := readResumeRuns(runsPath)
	if err != nil {
		return nil, ResumeState{}, err
	}
	runs, err := decodeRuns(runsBytes)
	if err != nil {
		return nil, ResumeState{}, err
	}
	retained, err := validateResumeRuns(path, plan, runs)
	if err != nil {
		return nil, ResumeState{}, err
	}
	retainedBytes, err := encodeRunRecords(retained)
	if err != nil {
		return nil, ResumeState{}, err
	}
	if err := archiveResumeState(path, runsBytes, len(retained) != len(runs)); err != nil {
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
	journal := &BatchJournal{
		ctx: ctx,
		config: BatchConfig{
			Root: filepath.Dir(filepath.Dir(path)), RunID: filepath.Base(path), Selection: plan.Selection, SelectionCount: uint64(plan.SelectionCount),
		},
		path: path, runsFile: runsFile, runsHasher: hasher, runsWriter: io.MultiWriter(runsFile, hasher), runsBytes: uint64(len(retainedBytes)), resumeLock: lock,
	}
	if err := journal.writeLifecycle(filepath.Join(path, ".partial", "batch"), "running", "", nil); err != nil {
		journal.Close()
		return nil, ResumeState{}, err
	}
	return journal, ResumeState{Plan: plan, Runs: retained}, nil
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

func validateResumeRuns(batchPath string, plan BatchPlan, runs []RunRecord) ([]RunRecord, error) {
	ordinals := make(map[uint64]struct{}, len(runs))
	observedProbes := make(map[string]struct{})
	retained := make([]RunRecord, 0, len(runs))
	var retainedSuccesses, retainedSuccessBytes uint64
	for index, run := range runs {
		ordinal := uint64(run.SelectionOrdinal)
		if ordinal >= uint64(plan.SelectionCount) {
			return nil, fmt.Errorf("resumable run %d selection ordinal is out of range", index+1)
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
		if err := validateChoiceRunSummary(run); err != nil {
			return nil, fmt.Errorf("resumable run %d: %w", index+1, err)
		}
		if err := validateSemanticProbeLists(run.SemanticProbes, run.NovelSemanticProbes); err != nil {
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
				if len(run.NovelSemanticProbes) != 0 || plan.KeepSuccesses == "all" {
					return nil, fmt.Errorf("successful resumable run %d violates its retention policy", index+1)
				}
			} else {
				if plan.KeepSuccesses == "none" || plan.KeepSuccesses == "novel" && len(run.NovelSemanticProbes) == 0 || plan.KeepSuccesses == "all" && len(run.NovelSemanticProbes) != 0 {
					return nil, fmt.Errorf("retained successful run %d violates its retention policy", index+1)
				}
				for _, probe := range run.NovelSemanticProbes {
					if _, found := observedProbes[probe]; found {
						return nil, fmt.Errorf("retained successful run %d claims previously observed probe %q", index+1, probe)
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
			if err := validateResumeArtifact(batchPath, plan, run); err != nil {
				return nil, fmt.Errorf("validate resumable run %d: %w", index+1, err)
			}
			retained = append(retained, run)
		case "runner":
			if run.Artifact != nil || run.FailureSignature != nil {
				if err := validateResumeArtifact(batchPath, plan, run); err != nil {
					return nil, fmt.Errorf("validate resumable Runner record %d: %w", index+1, err)
				}
			} else if run.Reason != "runner_cancelled" || run.Termination != "none" {
				return nil, fmt.Errorf("resumable Runner record %d is invalid", index+1)
			}
			delete(ordinals, ordinal)
		default:
			return nil, fmt.Errorf("resumable run %d domain is invalid: %s", index+1, run.Domain)
		}
		for _, probe := range run.SemanticProbes {
			observedProbes[probe] = struct{}{}
		}
	}
	return retained, nil
}

func validateResumeSuccessArtifact(batchPath string, plan BatchPlan, run RunRecord) error {
	evidence, err := ResolveRetainedEvidence(batchPath, filepath.Base(batchPath), run)
	if err != nil {
		return err
	}
	manifest := evidence.Manifest
	if manifest.ReplayMode != record.ReplayExact {
		return fmt.Errorf("retained success artifact run identity does not match its journal")
	}
	if manifest.Outcome.Reason != run.Reason || manifest.Outcome.Termination != run.Termination {
		return fmt.Errorf("retained success artifact outcome does not match its journal")
	}
	if manifest.Runner.RunnerBuild != plan.RunnerBuild || manifest.Toolchain != plan.Toolchain || manifest.Target.SHA256 != plan.Prepared.Target.SHA256 || manifest.Target.Size != plan.Prepared.Target.Size {
		return fmt.Errorf("retained success artifact target identity does not match its batch plan")
	}
	if !choiceProfileMatchesPlan(plan, manifest) {
		return fmt.Errorf("retained success artifact choice profile does not match its batch plan")
	}
	return nil
}

func validateResumeArtifact(batchPath string, plan BatchPlan, run RunRecord) error {
	evidence, err := ResolveRetainedEvidence(batchPath, filepath.Base(batchPath), run)
	if err != nil {
		return err
	}
	manifest := evidence.Manifest
	if manifest.Runner.RunnerBuild != plan.RunnerBuild || manifest.Toolchain != plan.Toolchain || manifest.Target.SHA256 != plan.Prepared.Target.SHA256 || manifest.Target.Size != plan.Prepared.Target.Size {
		return fmt.Errorf("failure artifact target identity does not match its batch plan")
	}
	if !choiceProfileMatchesPlan(plan, manifest) {
		return fmt.Errorf("failure artifact choice profile does not match its batch plan")
	}
	return nil
}

func choiceProfileMatchesPlan(plan BatchPlan, manifest record.Manifest) bool {
	if plan.ChoiceProfile == nil || manifest.ChoiceProfile == nil {
		return plan.ChoiceProfile == nil && manifest.ChoiceProfile == nil
	}
	return plan.ChoiceProfile.Name == manifest.ChoiceProfile.Name &&
		plan.ChoiceProfile.ImplementationSHA256 == manifest.ChoiceProfile.ImplementationSHA256 &&
		plan.ChoiceProfile.Limit == manifest.ChoiceProfile.Trace.Limit
}

func encodeRunRecords(runs []RunRecord) ([]byte, error) {
	values := make([]any, len(runs))
	for index := range runs {
		values[index] = runs[index]
	}
	return record.CanonicalJSONLines(values)
}

func archiveResumeState(path string, runs []byte, archiveRuns bool) error {
	partialRoot := filepath.Join(path, ".partial")
	entries, err := os.ReadDir(partialRoot)
	if err != nil {
		return fmt.Errorf("read partial batch state: %w", err)
	}
	toArchive := make([]os.DirEntry, 0)
	for _, entry := range entries {
		if entry.Name() == "batch" || entry.Name() == "resume" {
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
	if err := makePrivateDirectories(resumeRoot); err != nil {
		return err
	}
	attempt, err := nextResumeAttempt(resumeRoot)
	if err != nil {
		return err
	}
	attemptPath := filepath.Join(resumeRoot, fmt.Sprintf("%06d", attempt))
	partialsPath := filepath.Join(attemptPath, "partials")
	if err := makePrivateDirectories(partialsPath); err != nil {
		return err
	}
	if archiveRuns {
		if err := atomicWrite(filepath.Join(attemptPath, "runs.jsonl"), runs); err != nil {
			return err
		}
	}
	for _, entry := range toArchive {
		if err := os.Rename(filepath.Join(partialRoot, entry.Name()), filepath.Join(partialsPath, entry.Name())); err != nil {
			return fmt.Errorf("archive partial batch entry %q: %w", entry.Name(), err)
		}
	}
	return syncDirectory(partialRoot)
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
