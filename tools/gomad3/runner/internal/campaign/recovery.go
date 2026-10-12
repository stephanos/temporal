package campaign

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type recoveryAction string

const (
	recoveryFinalizePublication recoveryAction = "finalize-publication"
	recoveryRestoreRunning      recoveryAction = "restore-running"
)

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

func PreflightResume(path string) (ResumePreflight, error) {
	lifecycle, err := InspectCampaignLifecycle(path)
	if err != nil {
		return ResumePreflight{}, err
	}
	if lifecycle.Published || !lifecycle.Resumable && lifecycle.Action != recoveryRestoreRunning {
		return ResumePreflight{}, newIntegrityError(fmt.Errorf("campaign lifecycle %q is not resumable", lifecycle.State))
	}
	plan, err := ReadResumePlan(path)
	if err != nil {
		return ResumePreflight{}, classifyIntegrityError(err)
	}
	return ResumePreflight{Lifecycle: lifecycle, Plan: plan}, nil
}

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
	if before.Action == "" && !before.Published && !before.Resumable {
		return recoveryResult{}, newIntegrityError(fmt.Errorf("campaign lifecycle %q is not recoverable", before.State))
	}
	lock, err := acquireResumeLock(ctx, filepath.Join(absolute, ".resume.lock"))
	if err != nil {
		return recoveryResult{}, err
	}
	defer func() {
		retErr = errors.Join(retErr, releaseResumeLock(lock))
	}()
	before, err = InspectCampaignLifecycle(absolute)
	if err != nil {
		return recoveryResult{}, err
	}
	if err := ctx.Err(); err != nil {
		return recoveryResult{}, err
	}
	changed, err := applyRecoveryAction(ctx, absolute, before)
	if err != nil {
		return recoveryResult{}, err
	}
	result := recoveryResult{Action: before.Action, Changed: changed, Before: before, After: before}
	after, err := InspectCampaignLifecycle(absolute)
	if err != nil {
		return recoveryResult{}, err
	}
	result.After = after
	return result, nil
}

func applyRecoveryAction(ctx context.Context, path string, before LifecycleStatus) (bool, error) {
	switch before.Action {
	case recoveryFinalizePublication:
		for _, private := range []struct {
			path      string
			operation string
		}{
			{path: filepath.Join(path, ".prepared"), operation: "prepared-target"},
			{path: filepath.Join(path, ".partial", "preparation"), operation: "preparation"},
			{path: filepath.Join(path, ".partial", "campaign"), operation: "campaign-lifecycle"},
		} {
			if err := removeCompletedPartialContext(ctx, private.path, private.operation); err != nil {
				return false, err
			}
		}
		if err := syncExistingDirectory(ctx, filepath.Join(path, ".partial")); err != nil {
			return false, err
		}
		if err := syncDirectoryContext(ctx, path); err != nil {
			return false, err
		}
		return true, nil
	case recoveryRestoreRunning:
		if err := restoreInterruptedCommit(ctx, path); err != nil {
			return false, err
		}
		return true, nil
	default:
		if !before.Published && !before.Resumable {
			return false, newIntegrityError(fmt.Errorf("campaign lifecycle %q is not recoverable", before.State))
		}
		return false, nil
	}
}

func restoreInterruptedCommit(ctx context.Context, path string) error {
	journal := &CampaignJournal{
		ctx: ctx, config: CampaignConfig{CampaignID: filepath.Base(path)}, path: path,
		lifecycle: lifecycleCommitting, lastStableLifecycle: lifecycleRunning,
	}
	return journal.transitionLifecycle(lifecycleRecoverableFailure, "commit_interrupted", errors.New("final campaign manifest was not published"))
}

func syncExistingDirectory(ctx context.Context, path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0o700 {
		return errors.New("recovery directory metadata is invalid")
	}
	return syncDirectoryContext(ctx, path)
}
