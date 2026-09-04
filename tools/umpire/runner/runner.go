// Package runner binds one exact admitted input to a domain adapter and owns
// the reusable bounded execution handoff used by ordinary generated Go tests.
package runner

import (
	"context"
	"errors"
	"slices"
	"strconv"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/artifact"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
	"go.temporal.io/server/tools/umpire/internal/runtimeengine"
	umpireruntime "go.temporal.io/server/tools/umpire/runtime"
	"google.golang.org/protobuf/proto"
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
	RuntimeBindingSlots                      []*umpirespb.RuntimeBindingSlot
}

// RuntimeBindingValue is one adapter-resolved scalar for a declared runtime slot.
type RuntimeBindingValue struct {
	Definition *umpirespb.DefinitionBinding
	Value      *umpirespb.Value
}

// RuntimeBindingResolver fills only the runtime slots declared by a portable plan.
type RuntimeBindingResolver interface {
	ResolveRuntimeBindings(
		umpireruntime.CheckedRunRequest,
		[]*umpirespb.RuntimeBindingSlot,
	) ([]RuntimeBindingValue, error)
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
	if err := resolveRuntimeBindings(binding, request, adapter); err != nil {
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
	factory := adapter.EnvironmentFactory()
	if factory == nil {
		return umpireruntime.Output{}, classifyRunFailure(
			errors.New("umpire runner adapter is incomplete"), false,
		)
	}
	participant, err := adapter.NewParticipant(request)
	if err != nil {
		return umpireruntime.Output{}, classifyRunFailure(err, false)
	}
	if participant == nil {
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

func resolveRuntimeBindings(
	expected InputBinding,
	request umpireruntime.CheckedRunRequest,
	adapter Adapter,
) error {
	if len(expected.RuntimeBindingSlots) == 0 {
		return nil
	}
	resolver, ok := adapter.(RuntimeBindingResolver)
	if !ok {
		return runtimeBindingFailure("runtime binding resolver is required")
	}
	slots := make([]*umpirespb.RuntimeBindingSlot, len(expected.RuntimeBindingSlots))
	for index, slot := range expected.RuntimeBindingSlots {
		slots[index] = proto.CloneOf(slot)
	}
	bindings, err := resolver.ResolveRuntimeBindings(request, slots)
	if err != nil {
		return runtimeBindingFailure("runtime binding resolution failed: " + err.Error())
	}
	values, err := checkedRuntimeBindingValues(expected.RuntimeBindingSlots, bindings)
	if err != nil {
		return err
	}
	return validateRuntimePreconditions(request.Experiment().Plan, values)
}

type checkedRuntimeBindingValue struct {
	value string
}

func checkedRuntimeBindingValues(
	slots []*umpirespb.RuntimeBindingSlot,
	bindings []RuntimeBindingValue,
) (map[string]checkedRuntimeBindingValue, error) {
	if len(bindings) != len(slots) {
		return nil, runtimeBindingFailure("runtime binding resolution is incomplete")
	}
	expected := make(map[string]*umpirespb.RuntimeBindingSlot, len(slots))
	for _, slot := range slots {
		if slot == nil || slot.GetDefinition() == nil {
			return nil, runtimeBindingFailure("runtime binding slot is malformed")
		}
		expected[slot.GetDefinition().GetDefinitionId()] = slot
	}
	values := make(map[string]checkedRuntimeBindingValue, len(bindings))
	for _, binding := range bindings {
		if binding.Definition == nil {
			return nil, runtimeBindingFailure("runtime binding definition is required")
		}
		id := binding.Definition.GetDefinitionId()
		slot, declared := expected[id]
		if !declared || !proto.Equal(binding.Definition, slot.GetDefinition()) {
			return nil, runtimeBindingFailure("runtime binding crossed its declared slot")
		}
		if _, duplicate := values[id]; duplicate {
			return nil, runtimeBindingFailure("runtime binding was resolved more than once")
		}
		kind, value, valid := runtimeScalar(binding.Value)
		if !valid || kind != slot.GetValueKind() {
			return nil, runtimeBindingFailure("runtime binding has a crossed scalar kind")
		}
		values[id] = checkedRuntimeBindingValue{value: value}
	}
	return values, nil
}

func runtimeScalar(value *umpirespb.Value) (umpirespb.PortableValueKind, string, bool) {
	if value == nil {
		return umpirespb.PORTABLE_VALUE_KIND_UNSPECIFIED, "", false
	}
	switch scalar := value.GetValue().(type) {
	case *umpirespb.Value_Text:
		if len(scalar.Text) > artifact.MaximumDiagnosticBytes {
			return umpirespb.PORTABLE_VALUE_KIND_UNSPECIFIED, "", false
		}
		return umpirespb.PORTABLE_VALUE_KIND_TEXT, scalar.Text, true
	case *umpirespb.Value_Natural:
		if !canonicalNatural(scalar.Natural) {
			return umpirespb.PORTABLE_VALUE_KIND_UNSPECIFIED, "", false
		}
		return umpirespb.PORTABLE_VALUE_KIND_NATURAL, scalar.Natural, true
	case *umpirespb.Value_BoolValue:
		return umpirespb.PORTABLE_VALUE_KIND_BOOLEAN, strconv.FormatBool(scalar.BoolValue), true
	default:
		return umpirespb.PORTABLE_VALUE_KIND_UNSPECIFIED, "", false
	}
}

func canonicalNatural(value string) bool {
	if value == "" || (len(value) > 1 && value[0] == '0') || len(value) > artifact.MaximumDiagnosticBytes {
		return false
	}
	for _, digit := range value {
		if digit < '0' || digit > '9' {
			return false
		}
	}
	return true
}

func validateRuntimePreconditions(
	plan artifactv2.DrivePlan,
	bindings map[string]checkedRuntimeBindingValue,
) error {
	for _, precondition := range plan.ModelPreconditions {
		left, leftRuntime, leftOK := runtimeOperandValue(plan, precondition.Left, bindings)
		right, rightRuntime, rightOK := runtimeOperandValue(plan, precondition.Right, bindings)
		if !leftRuntime && !rightRuntime {
			continue
		}
		if !leftOK || !rightOK {
			return runtimeBindingFailure("runtime binding precondition is unresolved")
		}
		equal := left == right
		if (precondition.Relation == "equal" && !equal) ||
			(precondition.Relation == "different" && equal) {
			return runtimeBindingFailure("runtime binding precondition is not satisfied")
		}
	}
	return nil
}

func runtimeOperandValue(
	plan artifactv2.DrivePlan,
	operand artifactv2.Operand,
	bindings map[string]checkedRuntimeBindingValue,
) (value string, runtimeBound bool, ok bool) {
	if operand.Kind == "value" && operand.Value != nil {
		return operand.Value.Value, false, true
	}
	if operand.Kind != "role" {
		return "", false, false
	}
	if binding, ok := bindings[operand.DefinitionID]; ok {
		return binding.value, true, true
	}
	for _, binding := range plan.Bindings {
		if binding.RoleDefinitionID == operand.DefinitionID {
			return binding.Value.Value, false, true
		}
	}
	return "", false, false
}

func runtimeBindingFailure(message string) error {
	return &bindingFailure{
		kind: "runtime-binding", code: "umpire.runner.runtime-binding.unsatisfied", message: message,
	}
}
