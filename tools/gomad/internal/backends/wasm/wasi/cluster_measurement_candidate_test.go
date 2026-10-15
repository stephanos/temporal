//go:build gomad_cluster_candidate

package wasi

import (
	"testing"
	"time"
)

// The candidate adapter is enabled only after the single-helper client joins.
func clusterMeasurementCandidateProcess(t *testing.T, result *ClusterResult) map[string]any {
	t.Helper()
	if result == nil || result.Metrics == nil || result.Scheduler == nil || result.HelperResourceUsage == nil || result.AuthorizationStartedAt.IsZero() || !result.TerminalObservedAt.After(result.AuthorizationStartedAt) || result.AuthorizationToTerminalNanos == 0 || result.WallElapsedNanos == 0 {
		t.Fatal("candidate measurement requires an actual completed and owned-reaped cluster process")
	}
	if result.Frames.GuestCallbacks != 0 || !result.Metrics.ResourceUsageAvailable || result.Metrics.PeakRSSBytes == 0 || result.Scheduler.OperationSamples == 0 || result.Scheduler.RunnableSamples > result.Scheduler.OperationSamples || result.Scheduler.OverlapNanos > result.Scheduler.ActiveNanos {
		t.Fatal("candidate scheduler, ownership or resource observation invalid")
	}
	return map[string]any{
		"network_owner": "in-process-rust", "network_owner_implementation_sha256": ImplementationSHA256(),
		"authorization_started_at": result.AuthorizationStartedAt, "classified_reaped_at": result.TerminalObservedAt, "authorization_to_classified_reaped_nanos": result.TerminalObservedAt.Sub(result.AuthorizationStartedAt).Nanoseconds(),
		"cluster_authorization_to_terminal_nanos": result.AuthorizationToTerminalNanos, "cluster_wall_elapsed_nanos": result.WallElapsedNanos,
		"metrics": result.Metrics, "helper_process_usage": result.HelperResourceUsage, "frames": result.Frames, "scheduler": result.Scheduler,
		"scope":         "one completed helper process; cluster authorization and elapsed time include control/oracle pauses and preparation according to their own boundaries; never sum per-incarnation copies of these metrics",
		"guest_scope":   "execute wall minus timed host-import spans, an upper bound including ABI and unattributed runtime cost; not guest-only CPU",
		"handoff_scope": "handoff_nanos measures grant-to-wake latency; divide only by matching handoff_samples, not guest idle handoff count; fibers decision uses idle handoff count times current host switch cost",
	}
}

func clusterMeasurementCandidateEnvelope(t *testing.T, results []*ClusterResult) map[string]any {
	t.Helper()
	if len(results) != 3 {
		t.Fatal("actual crash/reopen, prefix and full-replay helper processes required")
	}
	var first, last time.Time
	var processes []map[string]any
	for _, result := range results {
		processes = append(processes, clusterMeasurementCandidateProcess(t, result))
		if first.IsZero() || result.AuthorizationStartedAt.Before(first) {
			first = result.AuthorizationStartedAt
		}
		if result.TerminalObservedAt.After(last) {
			last = result.TerminalObservedAt
		}
	}
	return map[string]any{"first_authorization_at": first, "last_classified_reaped_at": last, "authorization_to_classified_reaped_nanos": last.Sub(first).Nanoseconds(), "processes": processes}
}
