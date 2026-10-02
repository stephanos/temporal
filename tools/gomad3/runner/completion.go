package runner

import (
	"fmt"

	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/internal/execution"
	"go.temporal.io/server/tools/gomad3/target"
	"go.temporal.io/server/tools/gomad3/world"
)

// completedExecution is what every strategy reads from one captured execution
// once its World record is validated: the coverage it observed and its outcome.
type completedExecution struct {
	coverage         deterministicio.SemanticCoverage
	choiceFeatures   []string
	choiceProjection *choice.FeatureProjection
	outcome          execution.Classification
}

// assessWorld validates the World record a captured execution left for the
// seed it ran with. An execution that recorded none has the none World.
func assessWorld(result execution.Result, seed, transitionLimit uint64) (execution.Bundle, error) {
	worldBundle := noneWorldBundle()
	if len(result.WorldRecord) == 0 {
		return worldBundle, nil
	}
	recording, err := world.DecodeRecording(result.WorldRecord)
	if err == nil {
		worldBundle, err = execution.ComposeRecording(recording, transitionLimit)
	}
	if err == nil {
		initialWorld, _, validateErr := execution.Validate(worldBundle.Manifest, worldBundle.Payloads)
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
func assessCompletion(result execution.Result, terminal record.WorldTerminal, mode CoverageMode, prepared target.Prepared) (completedExecution, *HostError) {
	coverage, err := deterministicio.SummarizeSemanticProbes(nil)
	if err == nil && coverageHasSemantic(mode) {
		coverage, err = deterministicio.DecodeSemanticCoverage(result.IOTranscript.Bytes)
	}
	if err != nil {
		return completedExecution{}, &HostError{Reason: "semantic_coverage", Err: err}
	}
	assessed := completedExecution{coverage: coverage, choiceFeatures: []string{}}
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
	assessed.outcome = execution.Classify(result, false, terminal)
	return assessed, nil
}
