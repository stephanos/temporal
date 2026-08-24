package campaign

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"unicode/utf8"

	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomad3/record"
)

const (
	lifecycleSchema       = "gomad3.campaign-lifecycle/v1"
	maximumLifecycleBytes = 64 << 10
	maximumLifecycleText  = 4 << 10
)

type LifecycleState string

const (
	LifecyclePlanned            LifecycleState = "planned"
	LifecyclePrepared           LifecycleState = "prepared"
	LifecycleRunning            LifecycleState = "running"
	LifecycleCommitting         LifecycleState = "committing"
	LifecyclePublished          LifecycleState = "published"
	LifecycleRecoverableFailure LifecycleState = "recoverable-failure"
)

type LifecycleStatus struct {
	State           LifecycleState `json:"state"`
	LastStableState LifecycleState `json:"last_stable_state"`
	Reason          string         `json:"reason,omitempty"`
	Detail          string         `json:"detail,omitempty"`
	Published       bool           `json:"published"`
	Resumable       bool           `json:"resumable"`
	Repairable      bool           `json:"repairable"`
	Action          RecoveryAction `json:"action"`
}

type lifecycleRecord struct {
	Schema          string         `json:"schema"`
	SchemaVersion   uint32         `json:"schema_version"`
	CampaignID      string         `json:"campaign_id"`
	State           LifecycleState `json:"state"`
	LastStableState LifecycleState `json:"last_stable_state"`
	Reason          *string        `json:"reason"`
	Detail          *string        `json:"detail"`
}

func InspectCampaignLifecycle(path string) (_ LifecycleStatus, retErr error) {
	defer func() {
		retErr = classifyIntegrityError(retErr)
	}()
	absolute, err := filepath.Abs(path)
	if err != nil {
		return LifecycleStatus{}, fmt.Errorf("resolve campaign lifecycle path: %w", err)
	}
	info, err := os.Lstat(absolute)
	if err != nil {
		return LifecycleStatus{}, fmt.Errorf("inspect campaign lifecycle directory: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0o700 {
		return LifecycleStatus{}, errors.New("campaign lifecycle directory metadata is invalid")
	}
	manifestPath := filepath.Join(absolute, "campaign.json")
	if _, err := os.Lstat(manifestPath); err == nil {
		if _, err := OpenCampaign(absolute); err != nil {
			return LifecycleStatus{}, fmt.Errorf("validate published campaign lifecycle: %w", err)
		}
		stale, err := publishedPrivateStatePresent(absolute)
		if err != nil {
			return LifecycleStatus{}, err
		}
		status := LifecycleStatus{State: LifecyclePublished, LastStableState: LifecyclePublished, Published: true}
		if stale {
			status.Repairable = true
			status.Action = RecoveryFinalizePublication
		}
		return status, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return LifecycleStatus{}, fmt.Errorf("inspect published campaign lifecycle: %w", err)
	}
	record, err := readLifecycleRecord(absolute)
	if err != nil {
		return LifecycleStatus{}, err
	}
	status := LifecycleStatus{State: record.State, LastStableState: record.LastStableState}
	if record.Reason != nil {
		status.Reason = *record.Reason
	}
	if record.Detail != nil {
		status.Detail = *record.Detail
	}
	resumeState := record.State == LifecyclePrepared || record.State == LifecycleRunning ||
		record.State == LifecycleRecoverableFailure && (record.LastStableState == LifecyclePrepared || record.LastStableState == LifecycleRunning)
	if resumeState {
		_, statusErr := ReadResumePlan(absolute)
		status.Resumable = statusErr == nil
	}
	if record.State == LifecycleCommitting {
		if _, statusErr := ReadResumePlan(absolute); statusErr == nil {
			status.Repairable = true
			status.Action = RecoveryRestoreRunning
		}
	}
	return status, nil
}

func publishedPrivateStatePresent(path string) (bool, error) {
	present := false
	for _, privatePath := range []string{filepath.Join(path, ".prepared"), filepath.Join(path, ".partial", "campaign"), filepath.Join(path, ".partial", "preparation")} {
		info, err := os.Lstat(privatePath)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return false, fmt.Errorf("inspect published private state: %w", err)
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0o700 {
			return false, errors.New("published private state metadata is invalid")
		}
		present = true
	}
	return present, nil
}

func readLifecycleRecord(path string) (_ lifecycleRecord, retErr error) {
	root, err := os.OpenRoot(path)
	if err != nil {
		return lifecycleRecord{}, fmt.Errorf("pin campaign lifecycle directory: %w", err)
	}
	defer func() {
		retErr = errors.Join(retErr, root.Close())
	}()
	contents, err := readValidatedFile(root, filepath.Join(".partial", "campaign", "partial.json"), 0o600, maximumLifecycleBytes)
	if err != nil {
		return lifecycleRecord{}, fmt.Errorf("read campaign lifecycle: %w", err)
	}
	record, err := decodeLifecycleRecord(contents, filepath.Base(path))
	if err != nil {
		return lifecycleRecord{}, err
	}
	return record, nil
}

func decodeLifecycleRecord(contents []byte, campaignID string) (lifecycleRecord, error) {
	var record lifecycleRecord
	if err := canonicaljson.DecodeCanonicalJSON(contents, &record); err != nil {
		return lifecycleRecord{}, fmt.Errorf("decode campaign lifecycle: %w", err)
	}
	if err := validateLifecycleRecord(record, campaignID); err != nil {
		return lifecycleRecord{}, err
	}
	return record, nil
}

func validateLifecycleRecord(lifecycle lifecycleRecord, campaignID string) error {
	if lifecycle.Schema != lifecycleSchema || lifecycle.SchemaVersion != record.SchemaVersion || lifecycle.CampaignID != campaignID || !validLifecycleState(lifecycle.State) || !validRecoveryState(lifecycle.LastStableState) {
		return errors.New("campaign lifecycle identity is invalid")
	}
	if lifecycle.State == LifecycleRecoverableFailure {
		if lifecycle.Reason == nil || *lifecycle.Reason == "" {
			return errors.New("recoverable campaign lifecycle reason is missing")
		}
	} else if lifecycle.State == LifecycleCommitting && lifecycle.LastStableState != LifecycleRunning || lifecycle.State != LifecycleCommitting && lifecycle.State != lifecycle.LastStableState || lifecycle.Reason != nil || lifecycle.Detail != nil {
		return errors.New("stable campaign lifecycle contains failure evidence")
	}
	if lifecycle.Reason != nil && !validLifecycleText(*lifecycle.Reason) || lifecycle.Detail != nil && !validLifecycleText(*lifecycle.Detail) {
		return errors.New("campaign lifecycle failure evidence is invalid")
	}
	return nil
}

func validLifecycleState(state LifecycleState) bool {
	return validRecoveryState(state) || state == LifecycleCommitting || state == LifecycleRecoverableFailure
}

func validRecoveryState(state LifecycleState) bool {
	return state == LifecyclePlanned || state == LifecyclePrepared || state == LifecycleRunning
}

func validLifecycleText(value string) bool {
	return len(value) <= maximumLifecycleText && utf8.ValidString(value)
}

func (journal *CampaignJournal) transitionLifecycle(next LifecycleState, reason string, cause error) error {
	return journal.transitionLifecycleContext(journal.ctx, next, reason, cause)
}

func (journal *CampaignJournal) transitionLifecycleContext(ctx context.Context, next LifecycleState, reason string, cause error) error {
	if !validLifecycleTransition(journal.lifecycle, next) {
		return fmt.Errorf("invalid campaign lifecycle transition %q -> %q", journal.lifecycle, next)
	}
	record := lifecycleRecord{
		Schema: lifecycleSchema, SchemaVersion: record.SchemaVersion, CampaignID: journal.config.CampaignID,
		State: next, LastStableState: next,
	}
	if next == LifecycleCommitting {
		record.LastStableState = journal.lifecycle
	}
	if next == LifecycleRecoverableFailure {
		stable := journal.lifecycle
		if stable == LifecycleRecoverableFailure {
			stable = journal.lastStableLifecycle
		}
		if stable == LifecycleCommitting {
			stable = journal.lastStableLifecycle
		}
		if !validRecoveryState(stable) {
			return errors.New("campaign lifecycle has no stable recovery state")
		}
		if !validLifecycleText(reason) || reason == "" {
			return errors.New("campaign lifecycle failure reason is invalid")
		}
		record.LastStableState = stable
		record.Reason = &reason
		if cause != nil {
			detail := cause.Error()
			if len(detail) > maximumLifecycleText {
				detail = detail[:maximumLifecycleText]
				for !utf8.ValidString(detail) {
					detail = detail[:len(detail)-1]
				}
			}
			record.Detail = &detail
		}
	}
	encoded, err := canonicaljson.CanonicalJSON(record)
	if err != nil {
		return fmt.Errorf("encode campaign lifecycle: %w", err)
	}
	if len(encoded) > maximumLifecycleBytes {
		return fmt.Errorf("campaign lifecycle exceeds %d bytes", maximumLifecycleBytes)
	}
	if err := atomicWriteContext(ctx, filepath.Join(journal.path, ".partial", "campaign", "partial.json"), encoded); err != nil {
		return fmt.Errorf("publish campaign lifecycle: %w", err)
	}
	journal.lifecycle = next
	journal.lastStableLifecycle = record.LastStableState
	return nil
}

func validLifecycleTransition(current, next LifecycleState) bool {
	if current == "" {
		return next == LifecyclePlanned
	}
	if next == LifecycleRecoverableFailure {
		return current == LifecyclePlanned || current == LifecyclePrepared || current == LifecycleRunning || current == LifecycleCommitting || current == LifecycleRecoverableFailure
	}
	if current == LifecycleRecoverableFailure {
		return next == LifecycleRunning
	}
	return current == LifecyclePlanned && next == LifecyclePrepared ||
		current == LifecyclePrepared && next == LifecycleRunning ||
		current == LifecycleRunning && next == LifecycleCommitting
}
