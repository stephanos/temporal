package runner

import (
	"context"
	"errors"
	"os"

	backend "go.temporal.io/server/tools/gomad3/backends"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/target"
)

func campaignPreparer(config campaignRequest) Preparer {
	if config.Preparer != nil {
		return config.Preparer
	}
	return config.Backend
}

func backendValidator(config campaignRequest) func(target.Spec, target.Prepared, []string) error {
	if config.Backend == nil {
		return nil
	}
	return config.Backend.ValidatePrepared
}

func recordedBackend(prepared target.Prepared, result backend.ExecutionResult) *record.BackendMetadata {
	metadata := record.CloneBackendMetadata(prepared.Backend)
	if metadata != nil && len(result.BackendEvidence) != 0 {
		metadata.Evidence = &record.BackendPayload{Schema: "backend-evidence/v1", File: "backend/evidence.bin", SHA256: record.HashBytes(result.BackendEvidence), Bytes: record.Uint64String(len(result.BackendEvidence))}
	}
	return metadata
}

func backendChoiceRequest(capability *backend.ChoiceCapability) *backend.ChoiceRequest {
	if capability == nil {
		return nil
	}
	return &backend.ChoiceRequest{Mode: capability.Mode, ExecutionIdentity: capability.ExecutionIdentity, Limit: capability.Limit, Tape: capability.ReplayPlan}
}

func runBackend(ctx context.Context, provider backend.Provider, request backend.Request, stdout, stderr *os.File) (backend.ExecutionResult, error) {
	if provider == nil {
		return backend.ExecutionResult{}, errors.New("backend provider is required")
	}
	observed, err := provider.Run(ctx, request)
	if err != nil {
		return backend.ExecutionResult{}, err
	}
	return backend.ProjectObservedResult(request, observed, stdout, stderr)
}
