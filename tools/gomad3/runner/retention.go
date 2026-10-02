package runner

import (
	"errors"

	"go.temporal.io/server/tools/gomad3/artifact"
	"go.temporal.io/server/tools/gomad3/deterministicio/readonlymount"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/internal/campaign"
	"go.temporal.io/server/tools/gomad3/runner/internal/execution"
	"go.temporal.io/server/tools/gomad3/target"
)

// successRetention is the decision to keep one successful execution: whether
// to, what made it novel under the novel policy, and how many bytes of the
// campaign's success bound its artifact may take.
type successRetention struct {
	retain       bool
	novelProbes  []string
	novelChoices []string
	maximumBytes uint64
}

// decideSuccessRetention judges one assessed success against the novelty and
// the retained count and bytes its strategy has committed. It reads that state
// and advances none of it: the strategy does, at its own commit point.
func decideSuccessRetention(config CampaignSpec, assessed completedExecution, transcriptComplete bool, seenProbes, seenChoices map[string]struct{}, retained, retainedBytes uint64) (successRetention, *HostError) {
	decision := successRetention{retain: config.KeepSuccesses == KeepSuccessesAll}
	if config.KeepSuccesses == KeepSuccessesNovel {
		decision.novelProbes = novelStrings(assessed.coverage.Probes, seenProbes)
		decision.novelChoices = novelStrings(assessed.choiceFeatures, seenChoices)
		decision.retain = len(decision.novelProbes) != 0 || len(decision.novelChoices) != 0
	}
	if !decision.retain {
		return decision, nil
	}
	if !transcriptComplete {
		return decision, &HostError{Reason: "success_artifact_publication", Err: errors.New("retained success requires a complete I/O transcript for exact replay")}
	}
	if retained >= config.SuccessArtifactLimit || retainedBytes >= config.SuccessBytesLimit {
		return decision, &HostError{Reason: "success_retention_capacity", Err: errors.New("successful-execution retention capacity is exhausted")}
	}
	decision.maximumBytes = config.SuccessBytesLimit - retainedBytes
	return decision, nil
}

// annotate records on the journal record of a kept success where its artifact
// is stored and, under the novel policy, what made it novel.
func (decision successRetention) annotate(run *campaign.ExecutionRecord, relative string, storedBytes uint64) {
	bytes := record.Uint64String(storedBytes)
	run.SuccessArtifact = &relative
	run.SuccessArtifactBytes = &bytes
	run.NovelSemanticProbes = append([]string(nil), decision.novelProbes...)
	run.NovelChoiceFeatures = append([]string(nil), decision.novelChoices...)
}

// successPublicationFailure classifies a failure to publish a kept success: a
// store that is out of the bytes it was offered is a capacity failure.
func successPublicationFailure(err error) *HostError {
	reason := "success_artifact_publication"
	var capacity *artifact.CapacityError
	if errors.As(err, &capacity) {
		reason = "success_retention_capacity"
	}
	return &HostError{Reason: reason, Err: err}
}

// executionArtifactInput composes what every published execution artifact
// carries. The strategies that record simulation payloads add them.
func executionArtifactInput(manifest record.ExecutionRecord, prepared target.Prepared, result execution.Result, mountArtifact *readonlymount.CapturedInputs, worldBundle execution.Bundle) artifact.ArtifactInput {
	return artifact.ArtifactInput{
		Manifest: manifest, TargetPath: prepared.Path, Stdout: result.Stdout.Bytes, Stderr: result.Stderr.Bytes,
		IOTranscript: result.IOTranscript.Bytes, ChoiceTrace: result.ChoiceTrace.Trace.Bytes, ReadOnlyMounts: mountArtifact, World: worldBundle.Payloads,
	}
}
