package temporal

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"go.temporal.io/server/tests/umpire3/execution/participant"
)

func (r *SDKParticipantAdapter) Stop(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.started || r.stopped {
		return nil
	}
	var stopErr error
	if !r.workflowClosed {
		if err := r.options.Client.SignalWorkflow(ctx, r.run.GetID(), r.run.GetRunID(), SDKFinishSignalName, nil); err != nil {
			stopErr = r.terminateAfterStopFailure(err)
		} else {
			var results []participant.Result
			if err := r.run.Get(ctx, &results); err != nil {
				stopErr = r.terminateAfterStopFailure(err)
			} else {
				r.workflowClosed = true
				r.stopped = true
			}
		}
	} else {
		r.stopped = true
	}
	if r.options.CallbackDriver != nil {
		if err := r.options.CallbackDriver.CleanupCompletionCallbacks(ctx); err != nil {
			stopErr = errors.Join(stopErr, fmt.Errorf("cleanup SDK completion callbacks: %w", err))
		}
	}
	if r.options.NexusDriver != nil {
		if err := r.options.NexusDriver.CleanupNexus(ctx); err != nil {
			stopErr = errors.Join(stopErr, fmt.Errorf("cleanup dedicated Nexus driver: %w", err))
		}
	}
	return stopErr
}

func (r *SDKParticipantAdapter) queryResult(ctx context.Context, commandID string) (participant.Result, error) {
	encoded, err := r.options.Client.QueryWorkflow(ctx, r.run.GetID(), r.run.GetRunID(), SDKStateQueryName)
	if err != nil {
		return participant.Result{}, err
	}
	var state SDKWorkflowState
	if err := encoded.Get(&state); err != nil {
		return participant.Result{}, fmt.Errorf("decode SDK participant state: %w", err)
	}
	return state.Results[commandID], nil
}

func (r *SDKParticipantAdapter) waitForResult(ctx context.Context, commandID string) (participant.Result, error) {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		result, err := r.queryResult(ctx, commandID)
		if err != nil || result.CommandID != "" {
			return result, err
		}
		select {
		case <-ctx.Done():
			return participant.Result{}, ctx.Err()
		case <-ticker.C:
		}
	}
}

func (r *SDKParticipantAdapter) qualifyResult(operation participant.Operation, result participant.Result) (participant.Result, error) {
	if result.CommandID != operation.CommandID || result.Status == "" {
		return participant.Result{}, errors.New("SDK participant returned incomplete command result")
	}
	if result.Source == "" {
		switch operation.SemanticAction {
		case "create-speculative-workflow-task", "commit-speculative-workflow-task":
			result.Source = "temporal-sdk-speculative-update"
		case "dispatch-assurance-workflow-task", "progress-entity":
			result.Source = "temporal-sdk-workflow-progress"
		default:
			result.Source = "temporal-sdk-participant"
		}
	}
	if result.SourceIdentity == "" {
		result.SourceIdentity = r.options.WorkflowType
	}
	if result.WorkflowID == "" {
		result.WorkflowID = r.run.GetID()
	}
	if result.RunID == "" {
		result.RunID = r.run.GetRunID()
	}
	if result.Reference == "" {
		switch result.Source {
		case "temporal-sdk-speculative-update":
			result.Reference = result.WorkflowID + "/" + result.RunID +
				"/speculative-update/" + operation.CommandID
		case "temporal-sdk-workflow-progress":
			result.Reference = result.WorkflowID + "/" + result.RunID +
				"/workflow-progress/" + operation.CommandID
		default:
			result.Reference = result.WorkflowID + "/" + result.RunID + "/" + operation.CommandID
		}
	}
	lineage := []string{r.plan.ProgramID, result.WorkflowID}
	if len(result.Lineage) == 0 {
		lineage = append(lineage, result.RunID)
	} else {
		lineage = append(lineage, result.Lineage...)
	}
	result.Lineage = slices.Compact(lineage)
	encoded, err := json.Marshal(operation)
	if err != nil {
		return participant.Result{}, fmt.Errorf("encode SDK operation receipt: %w", err)
	}
	digest := sha256.Sum256(encoded)
	result.PayloadDigest = "sha256:" + hex.EncodeToString(digest[:])
	return result, nil
}
