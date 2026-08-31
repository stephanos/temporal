package nexus

import (
	"go.temporal.io/server/tools/umpire/artifact"
	umpireruntime "go.temporal.io/server/tools/umpire/runtime"
	"go.temporal.io/server/tools/umpire/temporal/local"
)

// Binding is the sole current System-owned authority/program adapter available
// to the generated local runner. Its zero value is complete and accepts no
// endpoint, namespace, credential, executable, or semantic override.
type Binding struct{}

// CheckRequest constructs the exact model-owned caller-closure program and
// binds it to the already-admitted two-member input before any runtime IO.
func (Binding) CheckRequest(
	admitted artifact.AdmittedSet,
	runIdentity string,
) (umpireruntime.CheckedRunRequest, error) {
	return CheckRequest(admitted, runIdentity)
}

// EnvironmentFactory returns the invocation-owned loopback authority.
func (Binding) EnvironmentFactory() umpireruntime.EnvironmentFactory {
	return local.NewFactory()
}

// NewParticipant constructs the one checked SDK participant.
func (Binding) NewParticipant(
	request umpireruntime.CheckedRunRequest,
) (umpireruntime.Participant, error) {
	return NewParticipant(request)
}

// ValidateOutput proves the exact four-member operational closure without
// evaluating an Observation or Property.
func (Binding) ValidateOutput(
	request umpireruntime.CheckedRunRequest,
	output umpireruntime.Output,
) error {
	executable, ok := request.AdmittedSet().Executable()
	if !ok {
		return errExecutionClosure
	}
	if err := validateExecutionClosure(
		executable,
		output.AdmittedSet(),
		output.ExperimentRun(),
		output.RawEvidence(),
	); err != nil {
		return classifyExecutionClosure(err)
	}
	return nil
}
