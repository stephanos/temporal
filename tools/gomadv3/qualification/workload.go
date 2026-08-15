package qualification

import (
	"context"
	"errors"
	"fmt"

	"go.temporal.io/server/tools/gomadv3/deterministicio"
	"go.temporal.io/server/tools/gomadv3/evidence"
	"go.temporal.io/server/tools/gomadv3/runner"
	"go.temporal.io/server/tools/gomadv3/target"
)

const maximumQualificationRepeats = 32

type WorkloadProgress struct {
	Iteration uint64
	Repeat    uint64
}

type WorkloadSpec struct {
	Command         []string
	Seed            uint64
	Repeat          uint64
	ArtifactRoot    string
	Campaign        runner.CampaignSpec
	Replay          runner.ReplaySpec
	ReplaySuccesses bool
	Progress        func(WorkloadProgress) error
	Explore         func(context.Context, runner.CampaignSpec) (runner.CampaignResult, error)
	ReplayArtifact  func(context.Context, runner.ReplaySpec) (runner.ReplayResult, error)
	Write           func(string, QualificationReport) (string, error)
}

type WorkloadResult struct {
	Report      QualificationReport
	ReportPath  string
	ChoiceTrace *runner.ChoiceTraceSummary
}

func RunWorkload(ctx context.Context, spec WorkloadSpec) (WorkloadResult, error) {
	var err error
	spec, err = normalizeWorkloadSpec(spec)
	if err != nil {
		return WorkloadResult{}, err
	}

	runs := make([]QualificationExecution, 0, spec.Repeat)
	for iteration := uint64(1); iteration <= spec.Repeat; iteration++ {
		if spec.Progress != nil {
			if err := spec.Progress(WorkloadProgress{Iteration: iteration, Repeat: spec.Repeat}); err != nil {
				return WorkloadResult{}, err
			}
		}
		summary, err := spec.Explore(ctx, spec.Campaign)
		if err != nil {
			if summary.ExecutionEvidence != nil {
				runs = append(runs, workloadExecution(summary))
			}
			failure := workloadFailure(err, iteration)
			return retainWorkloadFailure(spec, runs, failure, summary.ChoiceTrace)
		}
		if summary.ExecutionEvidence == nil {
			failure := QualificationFailure{Classification: "runner_failure", Message: "Runner omitted bounded qualification evidence", Iteration: evidence.Uint64String(iteration)}
			return retainWorkloadFailure(spec, runs, failure, summary.ChoiceTrace)
		}
		run := workloadExecution(summary)
		if summary.ExecutionEvidence.Outcome.Domain != "success" && run.ArtifactPath == "" {
			failure := QualificationFailure{Classification: "runner_failure", Message: "Runner omitted the retained failure artifact", Iteration: evidence.Uint64String(iteration)}
			return retainWorkloadFailure(spec, runs, failure, summary.ChoiceTrace)
		}
		if spec.ReplaySuccesses && summary.ExecutionEvidence.Outcome.Domain == "success" && (summary.RetainedSuccesses != 1 || len(summary.SuccessArtifacts) != 1 || run.ArtifactPath == "") {
			failure := QualificationFailure{Classification: "runner_failure", Message: "Runner did not retain exactly one successful replay artifact", Iteration: evidence.Uint64String(iteration)}
			return retainWorkloadFailure(spec, runs, failure, summary.ChoiceTrace)
		}
		runs = append(runs, run)
	}

	for index := range runs {
		if runs[index].ArtifactPath == "" || runs[index].Evidence.Outcome.Domain == "success" && !spec.ReplaySuccesses {
			continue
		}
		replaySpec := spec.Replay
		replaySpec.ArtifactPath = runs[index].ArtifactPath
		replayed, err := spec.ReplayArtifact(ctx, replaySpec)
		runs[index].Replay = &QualificationReplay{ArtifactPath: runs[index].ArtifactPath, Attempted: true}
		if err != nil {
			runs[index].Replay.Divergence = err.Error()
			return retainWorkloadFailure(spec, runs, workloadReplayFailure(err, uint64(index)+1), nil)
		}
		runs[index].Replay.Match = replayed.Match
		runs[index].Replay.Diagnostic = replayed.Diagnostic
		runs[index].Replay.Divergence = replayed.Divergence
		runs[index].Replay.ChoiceReplayStatus = replayed.ChoiceReplayStatus
	}
	report, err := BuildQualificationReport(QualificationInput{Command: spec.Command, Runs: runs})
	if err != nil {
		return WorkloadResult{}, err
	}
	return publishWorkload(spec, report, nil)
}

func normalizeWorkloadSpec(spec WorkloadSpec) (WorkloadSpec, error) {
	if spec.Repeat < 2 || spec.Repeat > maximumQualificationRepeats {
		return WorkloadSpec{}, fmt.Errorf("qualification repeat must be between 2 and %d", maximumQualificationRepeats)
	}
	if len(spec.Command) == 0 || spec.Command[0] == "" {
		return WorkloadSpec{}, errors.New("qualification command is required")
	}
	if spec.ArtifactRoot == "" {
		return WorkloadSpec{}, errors.New("qualification artifact root is required")
	}
	if spec.Explore == nil {
		spec.Explore = runner.Explore
	}
	if spec.ReplayArtifact == nil {
		spec.ReplayArtifact = runner.Replay
	}
	if spec.Write == nil {
		spec.Write = WriteQualificationReport
	}
	return spec, nil
}

func workloadExecution(summary runner.CampaignResult) QualificationExecution {
	run := QualificationExecution{CampaignPath: summary.CampaignPath, Evidence: *summary.ExecutionEvidence}
	if summary.ExecutionEvidence.Outcome.Domain == "success" && len(summary.SuccessArtifacts) != 0 {
		run.ArtifactPath = summary.SuccessArtifacts[0]
	} else if len(summary.Artifacts) != 0 {
		run.ArtifactPath = summary.Artifacts[0]
	}
	return run
}

func retainWorkloadFailure(spec WorkloadSpec, runs []QualificationExecution, failure QualificationFailure, trace *runner.ChoiceTraceSummary) (WorkloadResult, error) {
	report, err := BuildQualificationFailure(spec.Command, spec.Seed, spec.Repeat, runs, failure)
	if err != nil {
		return WorkloadResult{ChoiceTrace: trace}, err
	}
	return publishWorkload(spec, report, trace)
}

func publishWorkload(spec WorkloadSpec, report QualificationReport, trace *runner.ChoiceTraceSummary) (WorkloadResult, error) {
	path, err := spec.Write(spec.ArtifactRoot, report)
	return WorkloadResult{Report: report, ReportPath: path, ChoiceTrace: trace}, err
}

func workloadReplayFailure(err error, iteration uint64) QualificationFailure {
	classification := "runner_failure"
	if errors.Is(err, context.Canceled) {
		classification = "cancelled"
	} else if errors.Is(err, context.DeadlineExceeded) {
		classification = "overall_timeout"
	}
	return QualificationFailure{Classification: classification, Message: err.Error(), Iteration: evidence.Uint64String(iteration)}
}

func workloadFailure(err error, iteration uint64) QualificationFailure {
	failure := QualificationFailure{Classification: classifyWorkloadError(err), Message: err.Error(), Iteration: evidence.Uint64String(iteration)}
	var unsupported *target.UnsupportedCapabilityError
	if errors.As(err, &unsupported) {
		failure.ImportPath = unsupported.ImportPath
		failure.Capability = unsupported.Capability
	}
	return failure
}

func classifyWorkloadError(err error) string {
	var missing *deterministicio.MissingSemanticProbesError
	if errors.As(err, &missing) {
		return "semantic_coverage_failure"
	}
	var unsupported *target.UnsupportedCapabilityError
	if errors.As(err, &unsupported) {
		return "unsupported_target"
	}
	var hostError *runner.HostError
	if errors.As(err, &hostError) {
		if hostError.Reason == "cancelled" || hostError.Reason == "overall_timeout" {
			return hostError.Reason
		}
		return "runner_failure"
	}
	return "invalid_input"
}
