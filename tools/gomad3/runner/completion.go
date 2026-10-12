package runner

import (
	"context"
	"fmt"

	backend "go.temporal.io/server/tools/gomad3/backends"
	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/record"

	"go.temporal.io/server/tools/gomad3/target"
	"go.temporal.io/server/tools/gomad3/world"
)

// completedExecution is what every strategy reads from one captured execution
// once its World record is validated: the coverage it observed and its outcome.
type completedExecution struct {
	coverage         record.SemanticCoverage
	choiceFeatures   []string
	choiceProjection *choice.FeatureProjection
	outcome          backend.Classification
}

// assessWorld validates the World record a captured execution left for the
// seed it ran with. An execution that recorded none has the none World.
func assessWorld(result backend.ExecutionResult, seed, transitionLimit uint64) (backend.Bundle, error) {
	worldBundle := noneWorldBundle()
	if len(result.WorldRecord) == 0 {
		return worldBundle, nil
	}
	recording, err := world.DecodeRecording(result.WorldRecord)
	if err == nil {
		worldBundle, err = backend.ComposeRecording(recording, transitionLimit)
	}
	if err == nil {
		initialWorld, _, validateErr := backend.Validate(worldBundle.Manifest, worldBundle.Payloads)
		if validateErr != nil {
			err = validateErr
		} else if worldBundle.Manifest.Initial.Schema != "gomad3.world.snapshot/v1" || uint64(initialWorld.Config.Seed) != seed {
			err = fmt.Errorf("World record seed or schema does not match seed %d", seed)
		}
	}
	return worldBundle, err
}

// assessCompletion projects the coverage a captured execution observed and
// classifies its outcome against the terminal of its validated World record.
// Semantic coverage is decoded before choice features are projected, and the
// first failure is the reported one.
func assessCompletion(ctx context.Context, provider backend.Provider, result backend.ExecutionResult, terminal record.WorldTerminal, mode CoverageMode, prepared target.Prepared) (completedExecution, *HostError) {
	request := backend.EvidenceRequest{}
	if coverageHasSemantic(mode) {
		request.Transcript = result.IOTranscript.Bytes
	}
	evidence, err := provider.Evidence(ctx, request)
	if err != nil {
		return completedExecution{}, &HostError{Reason: "semantic_coverage", Err: err}
	}
	assessed := completedExecution{coverage: evidence.Coverage, choiceFeatures: []string{}}
	// A target the watchdog or a cancellation killed wrote no choice trace to
	// project; the termination is its outcome.
	if coverageHasChoice(mode) && choiceTraceObserved(result) {
		projection, features, err := projectChoiceFeatures(result.ChoiceTrace, prepared)
		if err != nil {
			return completedExecution{}, &HostError{Reason: "choice_coverage", Err: err}
		}
		assessed.choiceProjection = &projection
		assessed.choiceFeatures = features
	}
	assessed.outcome = backend.Classify(result, false, terminal)
	if prepared.Backend != nil && assessed.outcome.ReplayMode == record.ReplayExact {
		assessed.outcome.ReplayMode = prepared.Backend.ReplayMode
	}
	return assessed, nil
}
