//nolint:revive // The package name is the public Umpire3 runtime.Run seam.
package runtime

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"go.temporal.io/server/tests/umpire3/environment"
	"go.temporal.io/server/tests/umpire3/protocol"
)

type ClaimKind = protocol.ClaimKind

const (
	ClaimConforming   = protocol.ClaimConforming
	ClaimViolating    = protocol.ClaimViolating
	ClaimUnsupported  = protocol.ClaimUnsupported
	ClaimInconclusive = protocol.ClaimInconclusive
)

type Limits struct {
	PrepareTimeout   time.Duration
	ActionTimeout    time.Duration
	ObserveTimeout   time.Duration
	CleanupTimeout   time.Duration
	MaxActions       int
	MaxObservations  int
	MaxResources     int
	MaxEvidenceBytes int64
}

type Request struct {
	Experiment  protocol.Experiment
	Environment environment.Factory
	Limits      Limits
}

type Claim struct {
	Kind       ClaimKind `json:"kind"`
	Property   string    `json:"property"`
	Checkpoint string    `json:"checkpoint,omitempty"`
	Reason     string    `json:"reason,omitempty"`
}

type ActionResult struct {
	Identifier string                     `json:"identifier"`
	Kind       string                     `json:"kind"`
	Evidence   environment.ActionEvidence `json:"evidence"`
	Error      string                     `json:"error,omitempty"`
}

type CheckpointResult struct {
	Identifier string `json:"identifier"`
	Satisfied  bool   `json:"satisfied"`
	Qualified  bool   `json:"qualified"`
	Reason     string `json:"reason,omitempty"`
}

type Result struct {
	FormatVersion    string                    `json:"formatVersion"`
	ExperimentDigest string                    `json:"experimentDigest"`
	Environment      EnvironmentProfile        `json:"environment"`
	Actions          []ActionResult            `json:"actions"`
	Bindings         environment.Bindings      `json:"bindings"`
	Observations     []environment.Observation `json:"observations"`
	Omissions        []string                  `json:"omissions"`
	Checkpoints      []CheckpointResult        `json:"checkpoints"`
	Claim            Claim                     `json:"claim"`
	Cleanup          environment.CleanupResult `json:"cleanup"`
}

type EnvironmentProfile struct {
	Name                  string   `json:"name,omitempty"`
	BuildID               string   `json:"buildID,omitempty"`
	ConfigurationIdentity string   `json:"configurationIdentity,omitempty"`
	EvidenceProfile       string   `json:"evidenceProfile,omitempty"`
	Capabilities          []string `json:"capabilities"`
}

func Run(ctx context.Context, request Request) (result Result, retErr error) {
	if err := request.Experiment.Validate(); err != nil {
		return Result{}, fmt.Errorf("validate experiment: %w", err)
	}
	if request.Environment == nil {
		return Result{}, errors.New("environment is required")
	}
	digest, err := request.Experiment.Digest()
	if err != nil {
		return Result{}, err
	}
	limits := request.Limits.withDefaults(request.Experiment)
	if len(request.Experiment.Actions) > limits.MaxActions ||
		len(request.Experiment.Checkpoints) > limits.MaxObservations ||
		len(request.Experiment.Resources) > limits.MaxResources {
		return Result{}, errors.New("experiment exceeds runtime count budget")
	}

	capabilities := uniqueSorted(request.Environment.Capabilities())
	result = Result{
		FormatVersion:    protocol.FormatVersion,
		ExperimentDigest: digest,
		Environment:      EnvironmentProfile{Capabilities: capabilities},
		Bindings:         make(environment.Bindings),
		Claim: Claim{
			Kind:     ClaimInconclusive,
			Property: request.Experiment.Property.Identifier,
		},
	}
	if missing := missingCapabilities(request.Experiment, capabilities); len(missing) != 0 {
		result.Claim.Kind = ClaimUnsupported
		result.Claim.Reason = "missing capabilities: " + fmt.Sprint(missing)
		return result, nil
	}

	prepareCtx, cancelPrepare := context.WithTimeout(ctx, limits.PrepareTimeout)
	session, err := request.Environment.Prepare(prepareCtx, request.Experiment)
	cancelPrepare()
	if session != nil {
		defer func() {
			finalizeCleanup(&result, session, limits.CleanupTimeout)
		}()
	}
	if err != nil {
		result.Claim.Reason = "prepare environment: " + err.Error()
		return result, nil
	}
	if session == nil {
		result.Claim.Reason = "prepare environment returned no session"
		return result, nil
	}
	if provider, ok := session.(environment.ProfileProvider); ok {
		profile := provider.Profile()
		result.Environment.Name = profile.Name
		result.Environment.BuildID = profile.BuildID
		result.Environment.ConfigurationIdentity = profile.ConfigurationIdentity
		result.Environment.EvidenceProfile = profile.EvidenceProfile
	}

	checkpointByID := make(map[string]protocol.Checkpoint, len(request.Experiment.Checkpoints))
	for _, checkpoint := range request.Experiment.Checkpoints {
		checkpointByID[checkpoint.Identifier] = checkpoint
	}
	observed := make(map[string]struct{}, len(checkpointByID))
	observe := func(identifier string) bool {
		if identifier == "" {
			return true
		}
		if _, exists := observed[identifier]; exists {
			return true
		}
		checkpoint := checkpointByID[identifier]
		observeCtx, cancelObserve := context.WithTimeout(ctx, limits.ObserveTimeout)
		observation, observeErr := session.Observe(observeCtx, checkpoint, result.Bindings)
		cancelObserve()
		if observeErr != nil {
			result.Omissions = append(result.Omissions, identifier+": "+observeErr.Error())
			if checkpoint.OmissionPolicy == "optional" {
				result.Checkpoints = append(result.Checkpoints, CheckpointResult{
					Identifier: identifier,
					Qualified:  true,
					Reason:     "optional observation omitted",
				})
				observed[identifier] = struct{}{}
				return true
			}
			result.Checkpoints = append(result.Checkpoints, CheckpointResult{
				Identifier: identifier,
				Reason:     observeErr.Error(),
			})
			return false
		}
		result.Observations = append(result.Observations, observation)
		qualified, reason := qualifyObservation(checkpoint, observation)
		result.Checkpoints = append(result.Checkpoints, CheckpointResult{
			Identifier: identifier,
			Satisfied:  observation.Satisfied,
			Qualified:  qualified,
			Reason:     reason,
		})
		observed[identifier] = struct{}{}
		if !qualified {
			result.Omissions = append(result.Omissions, identifier+": "+reason)
			return false
		}
		if !observation.Satisfied {
			result.Claim.Kind = ClaimViolating
			result.Claim.Checkpoint = identifier
			result.Claim.Reason = "checkpoint contradicted: " + identifier
			return false
		}
		return true
	}

	completeEvidence := true
	var evidenceBytes int64
	for _, action := range request.Experiment.Actions {
		if !observe(action.PreCheckpoint) {
			completeEvidence = false
			break
		}
		actionCtx, cancelAction := context.WithTimeout(ctx, limits.ActionTimeout)
		evidence, actionErr := session.Realize(actionCtx, action, result.Bindings)
		cancelAction()
		actionResult := ActionResult{Identifier: action.Identifier, Kind: action.Kind, Evidence: evidence}
		if actionErr != nil {
			actionResult.Error = actionErr.Error()
			result.Actions = append(result.Actions, actionResult)
			result.Claim.Reason = "realize action " + action.Identifier + ": " + actionErr.Error()
			completeEvidence = false
			break
		}
		evidenceBytes += actionEvidenceSize(evidence)
		if evidenceBytes > limits.MaxEvidenceBytes {
			actionResult.Error = "evidence budget exhausted"
			result.Actions = append(result.Actions, actionResult)
			result.Omissions = append(result.Omissions, "evidence budget exhausted")
			result.Claim.Reason = "evidence budget exhausted"
			completeEvidence = false
			break
		}
		for symbolic, concrete := range evidence.GroundedBindings {
			if existing, exists := result.Bindings[symbolic]; exists && existing != concrete {
				result.Omissions = append(result.Omissions, "conflicting binding: "+symbolic)
				completeEvidence = false
				continue
			}
			result.Bindings[symbolic] = concrete
		}
		result.Actions = append(result.Actions, actionResult)
		if !observe(action.PostCheckpoint) {
			completeEvidence = false
			if result.Claim.Kind == ClaimViolating {
				break
			}
		}
	}
	if result.Claim.Kind != ClaimViolating {
		if completeEvidence && len(observed) == len(request.Experiment.Checkpoints) {
			result.Claim.Kind = ClaimConforming
			result.Claim.Reason = "all required checkpoints qualified"
		} else {
			result.Claim.Kind = ClaimInconclusive
			if result.Claim.Reason == "" {
				result.Claim.Reason = "required evidence is incomplete"
			}
		}
	}
	return result, nil
}

func finalizeCleanup(result *Result, session environment.Session, timeout time.Duration) {
	cleanupCtx, cancelCleanup := context.WithTimeout(context.Background(), timeout)
	defer cancelCleanup()
	result.Cleanup = session.Cleanup(cleanupCtx)
	if result.Cleanup.Complete {
		return
	}
	if len(result.Cleanup.RecoverableResources) == 0 {
		result.Cleanup.RecoverableResources = session.RecoveryMetadata()
	}
	result.Omissions = append(result.Omissions, "cleanup incomplete")
	if result.Claim.Kind == ClaimConforming {
		result.Claim.Kind = ClaimInconclusive
		result.Claim.Reason = "cleanup incomplete"
	}
}

func (limits Limits) withDefaults(experiment protocol.Experiment) Limits {
	if limits.PrepareTimeout <= 0 {
		limits.PrepareTimeout = 10 * time.Second
	}
	if limits.ActionTimeout <= 0 {
		limits.ActionTimeout = 5 * time.Second
	}
	if limits.ObserveTimeout <= 0 {
		limits.ObserveTimeout = 5 * time.Second
	}
	if limits.CleanupTimeout <= 0 {
		limits.CleanupTimeout = 5 * time.Second
	}
	if limits.MaxActions <= 0 {
		limits.MaxActions = experiment.Scope.Bounds.MaxDepth
	}
	if limits.MaxObservations <= 0 {
		limits.MaxObservations = experiment.Scope.Bounds.MaxResults
	}
	if limits.MaxResources <= 0 {
		limits.MaxResources = len(experiment.Resources)
	}
	if limits.MaxEvidenceBytes <= 0 {
		limits.MaxEvidenceBytes = experiment.Retention.MaxArtifactBytes
	}
	return limits
}

func actionEvidenceSize(evidence environment.ActionEvidence) int64 {
	size := len(evidence.Source) + len(evidence.Reference)
	for key, value := range evidence.GroundedBindings {
		size += len(key) + len(value)
	}
	return int64(size)
}

func uniqueSorted(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		seen[value] = struct{}{}
	}
	result := make([]string, 0, len(seen))
	for value := range seen {
		result = append(result, value)
	}
	slices.Sort(result)
	return result
}

func missingCapabilities(experiment protocol.Experiment, available []string) []string {
	have := make(map[string]struct{}, len(available))
	for _, capability := range available {
		have[capability] = struct{}{}
	}
	missing := make(map[string]struct{})
	for _, action := range experiment.Actions {
		for _, capability := range action.RequiredCapabilities {
			if _, exists := have[capability]; !exists {
				missing[capability] = struct{}{}
			}
		}
	}
	return uniqueSortedMap(missing)
}

func uniqueSortedMap(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	slices.Sort(result)
	return result
}

func qualifyObservation(checkpoint protocol.Checkpoint, observation environment.Observation) (bool, string) {
	if observation.CheckpointID != checkpoint.Identifier || observation.Kind != checkpoint.Observation {
		return false, "observation identity does not match checkpoint"
	}
	if observation.Source == "" {
		return false, "observation source is missing"
	}
	switch checkpoint.Ordering {
	case "causal":
		if observation.CausalReference == "" {
			return false, "causal reference is missing"
		}
	case "source-sequence":
		if observation.SourceSequence <= 0 {
			return false, "source sequence is missing"
		}
	default:
		if checkpoint.Ordering != "none" {
			return false, "unknown ordering requirement"
		}
	}
	return true, ""
}
