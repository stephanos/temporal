// Package runner binds one exact admitted input to a domain adapter and owns
// the reusable bounded execution handoff used by ordinary generated Go tests.
package runner

import (
	"context"
	"errors"
	"slices"

	"go.temporal.io/server/tools/umpire/artifact"
	"go.temporal.io/server/tools/umpire/internal/runtimeengine"
	umpireruntime "go.temporal.io/server/tools/umpire/runtime"
)

// InputBinding is the complete generated identity of one executable Artifact
// set. Generated tests retain these values as literals so admission cannot be
// replaced by a fresh compilation of behavioral intent.
type InputBinding struct {
	ArtifactSetIdentity                      string
	ArtifactSetChecksum                      string
	ManifestSHA256                           string
	ExperimentArtifactChecksum               string
	ExperimentBehaviorFingerprint            string
	RuntimeConfigurationArtifactChecksum     string
	RuntimeConfigurationBehaviorFingerprint  string
	AuthorityRequiredCapabilityDefinitionIDs []string
}

type bindingFailure struct {
	kind    string
	code    string
	message string
}

type runFailure struct {
	cause             error
	executionOccurred bool
}

func (failure *runFailure) Error() string {
	if failure == nil || failure.cause == nil {
		return ""
	}
	return failure.cause.Error()
}

func (failure *runFailure) Unwrap() error {
	if failure == nil {
		return nil
	}
	return failure.cause
}

func (failure *runFailure) ExecutionOccurred() bool {
	return failure != nil && failure.executionOccurred
}

func (failure *bindingFailure) Error() string {
	if failure == nil {
		return ""
	}
	return failure.message
}

func (failure *bindingFailure) Kind() string {
	if failure == nil {
		return ""
	}
	return failure.kind
}

func (*bindingFailure) Phase() string {
	return "admission"
}

func (failure *bindingFailure) Code() string {
	if failure == nil {
		return ""
	}
	return failure.code
}

// Adapter supplies one closed authority/program binding below the reusable
// runner. It receives only an already-admitted set and checked runtime values.
type Adapter interface {
	CheckRequest(artifact.AdmittedSet, string) (umpireruntime.CheckedRunRequest, error)
	EnvironmentFactory() umpireruntime.EnvironmentFactory
	NewParticipant(umpireruntime.CheckedRunRequest) (umpireruntime.Participant, error)
	ValidateOutput(umpireruntime.CheckedRunRequest, umpireruntime.Output) error
}

// Run verifies the generated input binding before asking the adapter to
// construct authority, then verifies that binding before allowing participant
// or environment construction and executes wholly in memory.
// It never reads bytes, publishes output, or interprets evidence.
func Run(
	ctx context.Context,
	admitted artifact.AdmittedSet,
	binding InputBinding,
	runIdentity string,
	adapter Adapter,
) (umpireruntime.Output, error) {
	if err := validateInputBinding(admitted, binding); err != nil {
		return umpireruntime.Output{}, classifyRunFailure(err, false)
	}
	if ctx == nil {
		return umpireruntime.Output{}, classifyRunFailure(
			errors.New("umpire runner requires a context"), false,
		)
	}
	if adapter == nil {
		return umpireruntime.Output{}, classifyRunFailure(
			errors.New("umpire runner requires an adapter"), false,
		)
	}
	request, err := adapter.CheckRequest(admitted, runIdentity)
	if err != nil {
		return umpireruntime.Output{}, classifyRunFailure(err, false)
	}
	if err := validateAuthorityBinding(binding, request); err != nil {
		return umpireruntime.Output{}, classifyRunFailure(err, false)
	}
	return runChecked(ctx, request, adapter)
}

// runChecked executes one request that has already passed generated binding
// and adapter preflight. It exists so focused adapter tests can retain their
// checked-request seam without adding a second execution surface.
func runChecked(
	ctx context.Context,
	request umpireruntime.CheckedRunRequest,
	adapter Adapter,
) (umpireruntime.Output, error) {
	if ctx == nil {
		return umpireruntime.Output{}, classifyRunFailure(
			errors.New("umpire runner requires a context"), false,
		)
	}
	if adapter == nil {
		return umpireruntime.Output{}, classifyRunFailure(
			errors.New("umpire runner requires an adapter"), false,
		)
	}
	participant, err := adapter.NewParticipant(request)
	if err != nil {
		return umpireruntime.Output{}, classifyRunFailure(err, false)
	}
	factory := adapter.EnvironmentFactory()
	if factory == nil || participant == nil {
		return umpireruntime.Output{}, classifyRunFailure(
			errors.New("umpire runner adapter is incomplete"), false,
		)
	}
	output, err := runtimeengine.Run(ctx, request, factory, participant)
	if err != nil {
		return umpireruntime.Output{}, classifyRunFailure(err, executionOccurred(err))
	}
	if err := adapter.ValidateOutput(request, output); err != nil {
		return umpireruntime.Output{}, classifyRunFailure(err, true)
	}
	return output, nil
}

func classifyRunFailure(err error, executionOccurred bool) error {
	if err == nil {
		return nil
	}
	return &runFailure{cause: err, executionOccurred: executionOccurred}
}

func executionOccurred(err error) bool {
	var classified interface {
		ExecutionOccurred() bool
	}
	return errors.As(err, &classified) && classified.ExecutionOccurred()
}

func validateInputBinding(admitted artifact.AdmittedSet, expected InputBinding) error {
	executable, ok := admitted.Executable()
	if !ok {
		return &bindingFailure{
			kind:    "input-binding",
			code:    "umpire.runner.input-binding.invalid",
			message: "umpire runner requires an exact two-member executable set",
		}
	}
	experiment := executable.Experiment()
	configuration := executable.RuntimeConfiguration()
	if admitted.Identity() != expected.ArtifactSetIdentity ||
		admitted.Checksum() != expected.ArtifactSetChecksum ||
		admitted.ManifestSHA256() != expected.ManifestSHA256 ||
		experiment.ArtifactChecksum != expected.ExperimentArtifactChecksum ||
		experiment.QueryBehaviorFingerprint != expected.ExperimentBehaviorFingerprint ||
		configuration.ArtifactChecksum != expected.RuntimeConfigurationArtifactChecksum ||
		configuration.BehaviorFingerprint != expected.RuntimeConfigurationBehaviorFingerprint {
		return &bindingFailure{
			kind:    "input-binding",
			code:    "umpire.runner.input-binding.drift",
			message: "umpire runner generated input binding does not match the admitted set",
		}
	}
	return nil
}

func validateAuthorityBinding(
	expected InputBinding,
	request umpireruntime.CheckedRunRequest,
) error {
	if !slices.Equal(
		request.Authority().RequiredCapabilityDefinitionIDs(),
		expected.AuthorityRequiredCapabilityDefinitionIDs,
	) {
		return &bindingFailure{
			kind:    "authority-binding",
			code:    "umpire.runner.authority-binding.unauthorized",
			message: "umpire runner generated authority binding does not match the checked request",
		}
	}
	return nil
}
