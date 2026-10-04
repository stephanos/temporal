package consumer

import (
	"context"
	"encoding/json"
	"encoding/json/jsontext"

	"go.temporal.io/server/tools/gomad3/runner"
	"go.temporal.io/server/tools/gomad3/target"
	"go.temporal.io/server/tools/gomad3/upgrade/pinimpact"
	"go.temporal.io/server/tools/gomad3/world"
)

type preparer struct{}

func (preparer) Prepare(context.Context, target.Spec) (target.Prepared, error) {
	return target.Prepared{}, nil
}

func ConstructPublishedEvidence() {
	_ = runner.CampaignPlanInspection{Journal: runner.ExecutionJournalLimitsInspection{MaximumExecutions: 1, MaximumBytes: 2, SegmentBytes: 3, SegmentRecords: 4, MaximumSegments: 5, MaximumPartialExecutions: 6, CapacityOutcome: "infrastructure_failure"}, ArtifactCapacity: runner.ArtifactCapacityInspection{FailureArtifacts: 1, FailureBytes: 2, SuccessArtifacts: 3, SuccessBytes: 4, TotalBytes: 5, TranscriptBytes: 6, FailureOutcome: "infrastructure_failure", SuccessOutcome: "infrastructure_failure"}}
	_ = runner.CampaignInspection{Journal: &runner.ExecutionJournalInspection{Limits: runner.ExecutionJournalLimitsInspection{}}, ArtifactCapacity: &runner.ArtifactCapacityInspection{}}
	_ = target.CompatibilityPackEvidence{Governance: &target.CompatibilityPackGovernance{Workloads: []string{"unit"}, Platforms: []string{"linux/amd64"}}, Activation: []target.CompatibilityModuleEvidence{{Adapter: &target.CompatibilityPackAdapter{ProfileName: "adapter"}}}, Rules: []target.CompatibilityPackageRuleEvidence{{Module: target.CompatibilityModuleEvidence{}, GoSources: []target.CompatibilityPackSource{{Name: "source.go"}}, ForeignSources: []target.CompatibilityPackForeignSource{{Kind: "asm"}}, Capabilities: []string{"pure"}, Linknames: []target.CompatibilityLinknameEvidence{{Directives: []string{"link"}}}}}}
	_ = pinimpact.Spec{PacksDirectory: "authoring/packs"}
	var raw json.RawMessage = []byte(`{"ok":true}`)
	var option jsontext.Options = jsontext.AllowDuplicateNames(false)
	_ = raw.IsValid(option)
	var recorder *world.Recorder
	_ = recorder.FinishTerminal
}

type replayer struct{}

func (replayer) Replay(context.Context, runner.ReplaySpec) (runner.ReplayResult, error) {
	return runner.ReplayResult{}, nil
}

func ConstructRequests() {
	_ = runner.CampaignSpec{Preparer: preparer{}, Replayer: replayer{}}
	_ = runner.CampaignShardSpec{Replayer: replayer{}}
	_ = runner.ReplaySpec{}
	_ = runner.MinimizeSpec{Replayer: replayer{}}
	_ = runner.ResumeSpec{Replayer: replayer{}}
}
