package campaignstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"unicode/utf8"

	"go.temporal.io/server/tools/gomadv3/evidence"
)

const (
	lifecycleSchema       = "gomadv3.batch-lifecycle/v1"
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
		return LifecycleStatus{}, fmt.Errorf("resolve batch lifecycle path: %w", err)
	}
	info, err := os.Lstat(absolute)
	if err != nil {
		return LifecycleStatus{}, fmt.Errorf("inspect batch lifecycle directory: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0o700 {
		return LifecycleStatus{}, errors.New("batch lifecycle directory metadata is invalid")
	}
	manifestPath := filepath.Join(absolute, "batch.json")
	if _, err := os.Lstat(manifestPath); err == nil {
		if _, err := OpenCampaign(absolute); err != nil {
			return LifecycleStatus{}, fmt.Errorf("validate published batch lifecycle: %w", err)
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
		return LifecycleStatus{}, fmt.Errorf("inspect published batch lifecycle: %w", err)
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
	for _, privatePath := range []string{filepath.Join(path, ".prepared"), filepath.Join(path, ".partial", "batch"), filepath.Join(path, ".partial", "preparation")} {
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
		return lifecycleRecord{}, fmt.Errorf("pin batch lifecycle directory: %w", err)
	}
	defer func() {
		retErr = errors.Join(retErr, root.Close())
	}()
	contents, err := readValidatedFile(root, filepath.Join(".partial", "batch", "partial.json"), 0o600, maximumLifecycleBytes)
	if err != nil {
		return lifecycleRecord{}, fmt.Errorf("read batch lifecycle: %w", err)
	}
	record, err := decodeLifecycleRecord(contents, filepath.Base(path))
	if err != nil {
		return lifecycleRecord{}, err
	}
	return record, nil
}

func decodeLifecycleRecord(contents []byte, campaignID string) (lifecycleRecord, error) {
	var members map[string]json.RawMessage
	if err := json.Unmarshal(contents, &members); err != nil {
		return lifecycleRecord{}, fmt.Errorf("decode batch lifecycle: %w", err)
	}
	if _, current := members["schema"]; current {
		var record lifecycleRecord
		if err := evidence.DecodeCanonicalJSON(contents, &record); err != nil {
			return lifecycleRecord{}, fmt.Errorf("decode batch lifecycle: %w", err)
		}
		if err := validateLifecycleRecord(record, campaignID); err != nil {
			return lifecycleRecord{}, err
		}
		return record, nil
	}
	var legacy struct {
		SchemaVersion uint32  `json:"schema_version"`
		State         string  `json:"state"`
		Reason        *string `json:"reason"`
		Detail        *string `json:"detail"`
	}
	if err := evidence.DecodeCanonicalJSON(contents, &legacy); err != nil {
		return lifecycleRecord{}, fmt.Errorf("decode legacy batch lifecycle: %w", err)
	}
	if legacy.SchemaVersion != evidence.SchemaVersion || legacy.State != "running" && legacy.State != "failed" {
		return lifecycleRecord{}, errors.New("legacy batch lifecycle identity is invalid")
	}
	record := lifecycleRecord{
		Schema: lifecycleSchema, SchemaVersion: evidence.SchemaVersion, CampaignID: campaignID,
		State: LifecycleRunning, LastStableState: LifecycleRunning,
	}
	if legacy.State == "failed" {
		record.State = LifecycleRecoverableFailure
		record.Reason = legacy.Reason
		record.Detail = legacy.Detail
	}
	if err := validateLifecycleRecord(record, campaignID); err != nil {
		return lifecycleRecord{}, err
	}
	return record, nil
}

func validateLifecycleRecord(record lifecycleRecord, campaignID string) error {
	if record.Schema != lifecycleSchema || record.SchemaVersion != evidence.SchemaVersion || record.CampaignID != campaignID || !validLifecycleState(record.State) || !validRecoveryState(record.LastStableState) {
		return errors.New("batch lifecycle identity is invalid")
	}
	if record.State == LifecycleRecoverableFailure {
		if record.Reason == nil || *record.Reason == "" {
			return errors.New("recoverable batch lifecycle reason is missing")
		}
	} else if record.State == LifecycleCommitting && record.LastStableState != LifecycleRunning || record.State != LifecycleCommitting && record.State != record.LastStableState || record.Reason != nil || record.Detail != nil {
		return errors.New("stable batch lifecycle contains failure evidence")
	}
	if record.Reason != nil && !validLifecycleText(*record.Reason) || record.Detail != nil && !validLifecycleText(*record.Detail) {
		return errors.New("batch lifecycle failure evidence is invalid")
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
		return fmt.Errorf("invalid batch lifecycle transition %q -> %q", journal.lifecycle, next)
	}
	record := lifecycleRecord{
		Schema: lifecycleSchema, SchemaVersion: evidence.SchemaVersion, CampaignID: journal.config.CampaignID,
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
			return errors.New("batch lifecycle has no stable recovery state")
		}
		if !validLifecycleText(reason) || reason == "" {
			return errors.New("batch lifecycle failure reason is invalid")
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
	encoded, err := evidence.CanonicalJSON(record)
	if err != nil {
		return fmt.Errorf("encode batch lifecycle: %w", err)
	}
	if len(encoded) > maximumLifecycleBytes {
		return fmt.Errorf("batch lifecycle exceeds %d bytes", maximumLifecycleBytes)
	}
	if err := atomicWriteContext(ctx, filepath.Join(journal.path, ".partial", "batch", "partial.json"), encoded); err != nil {
		return fmt.Errorf("publish batch lifecycle: %w", err)
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
