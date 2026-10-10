package runner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"go.temporal.io/server/tools/gomad3/artifact"
	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/backend"
	"go.temporal.io/server/tools/gomad3/runner/internal/execution"
)

func replayBackend(ctx context.Context, config ReplaySpec, opened *artifact.Opened) (result ReplayResult, retErr error) {
	prepared, err := config.Backend.ValidateReplay(ctx, opened)
	if err != nil {
		return ReplayResult{}, &ReplayPreflightError{Err: err}
	}
	manifest := opened.Manifest()
	payloads, err := readWorldPayloads(opened)
	if err != nil {
		return ReplayResult{}, &ReplayPreflightError{Err: err}
	}
	if _, _, err := execution.Validate(manifest.World, payloads); err != nil {
		return ReplayResult{}, &ReplayPreflightError{Err: err}
	}
	if manifest.World.Initial.Schema != "gomad3.world.snapshot/none" {
		return ReplayResult{}, &ReplayPreflightError{Err: errors.New("external observed profile does not support a World replay plan")}
	}
	result = ReplayResult{Artifact: opened.Snapshot(), Verified: true, ChoiceReplayStatus: ChoiceReplayNone, ObservedRepetition: manifest.ReplayMode == record.ReplayObserved}
	choiceCapability, choiceUnavailable, err := choiceCapabilityForArtifact(opened)
	if err != nil {
		return ReplayResult{}, &ReplayPreflightError{Err: err}
	}
	if choiceUnavailable {
		result.ChoiceReplayStatus = ChoiceReplayUnavailable
		result.Divergence = "choice_profile.replay_unavailable"
		return result, nil
	}
	if choiceCapability != nil {
		result.ChoiceReplayStatus = ChoiceReplayAvailable
	}
	if config.VerifyOnly {
		return result, nil
	}
	work, err := os.MkdirTemp("", "gomad3-replay-")
	if err != nil {
		return ReplayResult{}, err
	}
	defer func() { retErr = errors.Join(retErr, os.RemoveAll(work)) }()
	prepared.Path = filepath.Join(work, "target")
	if err := opened.CopyPayload(manifest.Target.File, prepared.Path, 0o500); err != nil {
		return ReplayResult{}, err
	}
	timeout, err := duration(manifest.Limits.ExecutionTimeoutNanos)
	if err != nil {
		return ReplayResult{}, err
	}
	observed, err := runBackend(ctx, config.Backend, backend.Request{Target: prepared.CloneBackend(), Seed: uint64(manifest.Seed), Environment: replayEnvironment(manifest.Environment), Timeout: timeout, OutputBytes: uint64(manifest.Limits.OutputBytes), TranscriptBytes: uint64(manifest.Limits.IOTranscriptBytes), Choice: backendChoiceRequest(choiceCapability)}, nil, nil)
	if err != nil && choiceCapability != nil && backendChoiceReplayDivergence(err) {
		result.ChoiceReplayStatus = ChoiceReplayDiverged
		result.Divergence = "choice_profile.divergence"
		var failure *backend.FailureError
		if errors.As(err, &failure) && failure.ChoiceDivergence != nil {
			divergence := failure.ChoiceDivergence
			result.Divergence = fmt.Sprintf("choice_profile.divergence.ordinal[%d].%s", divergence.Ordinal, choice.DivergenceReasonName(divergence.Reason))
		}
		return result, nil
	}
	if config.ObservedDir != "" {
		err = errors.Join(err, dumpObservedStreams(config.ObservedDir, observed))
		if err == nil {
			err = os.WriteFile(filepath.Join(config.ObservedDir, "backend-evidence.bin"), observed.BackendEvidence, 0o600)
		}
	}
	if err != nil {
		return ReplayResult{}, err
	}
	result.Divergence = replayDivergence(manifest, observed, nil)
	if result.Divergence == "" && (manifest.Target.Backend.Evidence == nil || record.HashBytes(observed.BackendEvidence) != manifest.Target.Backend.Evidence.SHA256) {
		result.Divergence = "backend.evidence.sha256"
	}
	result.Match = result.Divergence == ""
	if choiceCapability != nil {
		result.ChoiceReplayStatus = ChoiceReplayExact
		if choiceTraceDivergence(manifest, observed) != "" {
			result.ChoiceReplayStatus = ChoiceReplayDiverged
		}
	}
	return result, nil
}

func backendChoiceReplayDivergence(err error) bool {
	for err != nil {
		if failure, ok := err.(*backend.FailureError); ok && failure.Termination == backend.Divergence {
			return true
		}
		err = errors.Unwrap(err)
	}
	return false
}
