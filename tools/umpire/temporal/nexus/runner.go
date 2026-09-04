package nexus

import (
	"errors"
	"reflect"

	"go.temporal.io/server/tools/umpire/artifact"
	umpireruntime "go.temporal.io/server/tools/umpire/runtime"
)

// Binding is the sole current System-owned authority/program adapter available
// to the generated local runner. Explicit construction supplies its Temporal
// authority without accepting a semantic override.
type Binding struct {
	factory umpireruntime.EnvironmentFactory
}

// NewBinding binds the caller-supplied Temporal environment factory.
func NewBinding(factory umpireruntime.EnvironmentFactory) (Binding, error) {
	if isNilEnvironmentFactory(factory) {
		return Binding{}, errors.New("nexus binding requires an environment factory")
	}
	return Binding{factory: factory}, nil
}

// CheckRequest constructs the exact model-owned caller-closure program and
// binds it to the already-admitted two-member input before any runtime IO.
func (Binding) CheckRequest(
	admitted artifact.AdmittedSet,
	runIdentity string,
) (umpireruntime.CheckedRunRequest, error) {
	return CheckRequest(admitted, runIdentity)
}

// EnvironmentFactory returns the supplied authority.
func (b Binding) EnvironmentFactory() umpireruntime.EnvironmentFactory {
	return b.factory
}

func isNilEnvironmentFactory(factory umpireruntime.EnvironmentFactory) bool {
	if factory == nil {
		return true
	}
	value := reflect.ValueOf(factory)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
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
