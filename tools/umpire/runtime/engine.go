package runtime

import (
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
		admitted:    admitted,
		run:         artifactv2.CopyExperimentRun(run),
		rawEvidence: artifactv2.CopyRawEvidence(rawEvidence),
	}
}

func (o Output) AdmittedSet() artifact.AdmittedSet { return o.admitted }

func (o Output) ExperimentRun() artifactv2.ExperimentRun {
	return artifactv2.CopyExperimentRun(o.run)
}

func (o Output) RawEvidence() artifactv2.RawEvidence {
	return artifactv2.CopyRawEvidence(o.rawEvidence)
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
