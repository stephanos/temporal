package runner

import (
	"context"
	"crypto/sha256"
	"errors"
	"testing"

	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/runner/internal/campaign"
	"go.temporal.io/server/tools/gomad3/runner/internal/execution"
)

func TestRunEarlyCompletionCountersRemainUnclassified(t *testing.T) {
	for _, test := range []struct {
		name   string
		config func(*testing.T) CampaignSpec
		reason string
		want   campaign.CampaignStatistics
	}{
		{
			name: "supervision failure",
			config: func(t *testing.T) CampaignSpec {
				config, _ := completionCampaign(t, StrategySeed, CoverageNone, nil, execution.ErrChoiceTraceMalformed)
				return config
			},
			reason: "choice_trace_malformed", want: campaign.CampaignStatistics{Attempted: 1},
		},
		{
			name: "prepared target integrity",
			config: func(t *testing.T) CampaignSpec {
				return testConfig(t, newFakePreparer(t), mutatingExecutor{}, "1", PolicyAll, 1)
			},
			reason: "prepared_target_integrity", want: campaign.CampaignStatistics{Attempted: 1},
		},
		{
			name: "world evidence",
			config: func(t *testing.T) CampaignSpec {
				config, _ := completionCampaign(t, StrategySeed, CoverageNone, func(result *execution.Result) {
					result.WorldRecord = completionWorldRecord(t, 8)
				}, nil)
				return config
			},
			reason: "world_record", want: campaign.CampaignStatistics{Attempted: 1, DistinctFailures: 1},
		},
		{
			name: "coverage evidence",
			config: func(t *testing.T) CampaignSpec {
				config, _ := completionCampaign(t, StrategySeed, CoverageSemantic, func(result *execution.Result) {
					payload := []byte("not an I/O transcript")
					result.IOTranscript = deterministicio.Transcript{Bytes: payload, SHA256: sha256.Sum256(payload), Records: 1, Complete: true}
				}, nil)
				return config
			},
			reason: "semantic_coverage", want: campaign.CampaignStatistics{Attempted: 1},
		},
		{
			name: "success publication",
			config: func(t *testing.T) CampaignSpec {
				config := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "1", PolicyAll, 1)
				config.KeepSuccesses = KeepSuccessesAll
				config.SuccessArtifactLimit = 1
				config.SuccessBytesLimit = 1 << 20
				return config
			},
			reason: "success_artifact_publication", want: campaign.CampaignStatistics{Attempted: 1},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			summary, err := Explore(context.Background(), test.config(t))
			var hostError *HostError
			if !errors.As(err, &hostError) || hostError.Reason != test.reason {
				t.Fatalf("Explore() error = %v, want %q", err, test.reason)
			}
			got := campaign.CampaignStatistics{
				Attempted: summary.Attempted, Succeeded: summary.Succeeded, Failures: summary.Failures,
				Watchdogs: summary.Watchdogs, ReplayDivergences: summary.ReplayDivergences,
				Cancelled: summary.Cancelled, DistinctFailures: summary.DistinctFailures,
				StopReason: campaign.ControllerStopReason(summary.StopReason),
			}
			if got != test.want {
				t.Fatalf("statistics = %#v, want %#v", got, test.want)
			}
		})
	}
}
