package runtime

import (
	"slices"

	"go.temporal.io/server/tools/umpire/artifact"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
)

// Output is one admitted in-memory execution closure. It has not been published.
type Output struct {
	admitted    artifact.AdmittedSet
	run         artifactv2.ExperimentRun
	rawEvidence artifactv2.RawEvidence
}

// NewOutput retains one admitted execution closure for the internal engine.
func NewOutput(
	admitted artifact.AdmittedSet,
	run artifactv2.ExperimentRun,
	rawEvidence artifactv2.RawEvidence,
) Output {
	return Output{
		admitted: admitted, run: cloneExperimentRun(run), rawEvidence: cloneRawEvidence(rawEvidence),
	}
}

func (o Output) AdmittedSet() artifact.AdmittedSet { return o.admitted }

func (o Output) ExperimentRun() artifactv2.ExperimentRun {
	return cloneExperimentRun(o.run)
}

func (o Output) RawEvidence() artifactv2.RawEvidence {
	return cloneRawEvidence(o.rawEvidence)
}

// InvariantError is one sanitized post-start engine or admission failure.
type InvariantError struct {
	phase             Phase
	code              string
	executionOccurred bool
}

// NewInvariantError retains one sanitized internal engine failure.
func NewInvariantError(phase Phase, code string, executionOccurred bool) *InvariantError {
	return &InvariantError{phase: phase, code: code, executionOccurred: executionOccurred}
}

func (e *InvariantError) Error() string {
	if e == nil {
		return ""
	}
	return e.code
}

func (e *InvariantError) Phase() Phase {
	if e == nil {
		return ""
	}
	return e.phase
}

func (e *InvariantError) Code() string {
	if e == nil {
		return ""
	}
	return e.code
}

func (e *InvariantError) ExecutionOccurred() bool {
	return e != nil && e.executionOccurred
}

func cloneExperimentRun(run artifactv2.ExperimentRun) artifactv2.ExperimentRun {
	cloned := run
	cloned.PhaseOutcomes = slices.Clone(run.PhaseOutcomes)
	for index := range cloned.PhaseOutcomes {
		cloned.PhaseOutcomes[index].StartedAtUnixMillis = cloneEngineNatural(run.PhaseOutcomes[index].StartedAtUnixMillis)
		cloned.PhaseOutcomes[index].FinishedAtUnixMillis = cloneEngineNatural(run.PhaseOutcomes[index].FinishedAtUnixMillis)
		cloned.PhaseOutcomes[index].Code = cloneEngineString(run.PhaseOutcomes[index].Code)
	}
	cloned.ControlAttempts = slices.Clone(run.ControlAttempts)
	for index := range cloned.ControlAttempts {
		cloned.ControlAttempts[index].ReceiptFactDefinitionID = cloneEngineString(run.ControlAttempts[index].ReceiptFactDefinitionID)
		cloned.ControlAttempts[index].Code = cloneEngineString(run.ControlAttempts[index].Code)
	}
	cloned.SourceClosures = slices.Clone(run.SourceClosures)
	cloned.Cleanup.Code = cloneEngineString(run.Cleanup.Code)
	cloned.Limits = slices.Clone(run.Limits)
	cloned.KnownGaps = cloneEngineGaps(run.KnownGaps)
	cloned.Provenance.SourceDefinitionIDs = slices.Clone(run.Provenance.SourceDefinitionIDs)
	cloned.Provenance.SourceLocations = slices.Clone(run.Provenance.SourceLocations)
	return cloned
}

func cloneRawEvidence(document artifactv2.RawEvidence) artifactv2.RawEvidence {
	cloned := document
	cloned.Sources = slices.Clone(document.Sources)
	cloned.Facts = slices.Clone(document.Facts)
	for index := range cloned.Facts {
		cloned.Facts[index].CausalFactDefinitionIDs = slices.Clone(document.Facts[index].CausalFactDefinitionIDs)
		cloned.Facts[index].Fields = slices.Clone(document.Facts[index].Fields)
	}
	cloned.KnownGaps = cloneEngineGaps(document.KnownGaps)
	cloned.Provenance.SourceDefinitionIDs = slices.Clone(document.Provenance.SourceDefinitionIDs)
	cloned.Provenance.SourceLocations = slices.Clone(document.Provenance.SourceLocations)
	return cloned
}

func cloneEngineGaps(gaps []artifactv2.KnownGap) []artifactv2.KnownGap {
	cloned := slices.Clone(gaps)
	for index := range cloned {
		cloned[index].Subject = cloneEngineString(cloned[index].Subject)
		cloned[index].Detail = cloneEngineString(cloned[index].Detail)
	}
	return cloned
}

func cloneEngineNatural(value *artifactv2.Natural) *artifactv2.Natural {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneEngineString(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
