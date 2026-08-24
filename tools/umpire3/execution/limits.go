package execution

import (
	"context"
	"time"

	protocolexperiment "go.temporal.io/server/tools/umpire3/protocol/experiment"
)

func finalizeCleanup(result *Result, session Session, timeout time.Duration) {
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

func (limits Limits) withDefaults(experiment protocolexperiment.Experiment) Limits {
	if limits.PrepareTimeout <= 0 {
		limits.PrepareTimeout = 10 * time.Second
	}
	if limits.ActionTimeout <= 0 {
		limits.ActionTimeout = 5 * time.Second
	}
	if limits.ObserveTimeout <= 0 {
		limits.ObserveTimeout = 5 * time.Second
	}
	if limits.FaultTimeout <= 0 {
		limits.FaultTimeout = limits.ActionTimeout
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

func waitForActionRate(ctx context.Context, previous *time.Time, perSecond int) error {
	if perSecond <= 0 {
		return nil
	}
	interval := time.Second / time.Duration(perSecond)
	if previous.IsZero() || interval <= 0 {
		*previous = time.Now()
		return nil
	}
	wait := time.Until(previous.Add(interval))
	if wait <= 0 {
		*previous = time.Now()
		return nil
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		*previous = time.Now()
		return nil
	}
}
