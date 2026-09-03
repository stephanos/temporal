package testplan

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/artifact"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	SupportedMajorVersion int32 = 1
	SupportedMinorVersion int32 = 0

	MaximumPlanBytes                 int64 = 1 << 20
	MaximumNestingDepth              int64 = 256
	MaximumCollectionItems           int64 = 10_000
	MaximumOperatorCount             int64 = 100_000
	MaximumActionCount               int64 = 1
	MaximumFaultCount                int64 = 1
	MaximumPhaseAttempts             int64 = 1
	MaximumPhaseDurationMilliseconds int64 = 30_000
	MaximumTotalDurationMilliseconds int64 = 120_000
	MaximumEvidenceRecords           int64 = 100_000
	MaximumEvidenceBytes             int64 = 16 << 20
	MaximumEvidenceSources           int64 = 10_000
	MaximumExpressionDepth           int64 = 64
	MaximumEvaluationWork            int64 = 10_000_000
	MaximumDiagnosticBytes           int64 = 64 << 10
	MaximumResultBytes               int64 = 4 << 20
)

var expectedPhases = [...]struct {
	phase    umpirespb.ExecutionPhase
	duration int64
	records  int64
	bytes    int64
}{
	{umpirespb.EXECUTION_PHASE_PREPARATION, 30_000, 128, 1 << 20},
	{umpirespb.EXECUTION_PHASE_REALIZATION, 30_000, 128, 1 << 20},
	{umpirespb.EXECUTION_PHASE_OBSERVATION, 30_000, 3_584, 12 << 20},
	{umpirespb.EXECUTION_PHASE_ISOLATION, 15_000, 128, 1 << 20},
	{umpirespb.EXECUTION_PHASE_CLEANUP, 15_000, 128, 1 << 20},
}

type surfaceStats struct {
	maxDepth      int64
	maxCollection int64
}

type validator struct {
	plan      *umpirespb.PortableTestPlan
	limits    *umpirespb.PortableTestPlanLimits
	bindings  map[string]string
	operators int64
}

func validatePlan(plan *umpirespb.PortableTestPlan, checksumOptional bool) error {
	if plan == nil {
		return admissionError(ErrorMalformedValue, "$", "plan is required")
	}
	stats := surfaceStats{}
	if err := inspectProtoSurface(plan.ProtoReflect(), "$", 1, &stats); err != nil {
		return err
	}
	validator := &validator{
		plan: plan, limits: plan.GetLimits(), bindings: make(map[string]string),
	}
	if err := validator.validateVersion(); err != nil {
		return err
	}
	if !validDefinitionID(plan.GetPlanId()) || len(plan.GetPlanId()) > artifact.MaximumIdentityBytes {
		return admissionError(ErrorMalformedValue, "$.planId", "plan identity is malformed")
	}
	checksumLength := len(plan.GetPlanChecksum())
	if checksumLength != sha256Size && (!checksumOptional || checksumLength != 0) {
		return admissionError(ErrorChecksum, "$.planChecksum", "checksum has %d bytes; want %d", checksumLength, sha256Size)
	}
	if err := validator.validateLimits(stats); err != nil {
		return err
	}
	if err := validator.validateProvenance(); err != nil {
		return err
	}
	if err := validator.validateExecution(); err != nil {
		return err
	}
	if err := validator.validateVerification(); err != nil {
		return err
	}
	if err := validator.validateKnownGaps(); err != nil {
		return err
	}
	if err := validator.validateObligations(); err != nil {
		return err
	}
	if validator.operators > validator.limits.GetStructural().GetMaxOperatorCount() {
		return admissionError(ErrorLimit, "$.limits.structural.maxOperatorCount",
			"plan has %d operators; limit is %d", validator.operators,
			validator.limits.GetStructural().GetMaxOperatorCount())
	}
	if int64(proto.Size(plan)) > validator.limits.GetStructural().GetMaxPlanBytes() {
		return admissionError(ErrorByteLimit, "$.limits.structural.maxPlanBytes",
			"plan has %d bytes; limit is %d", proto.Size(plan),
			validator.limits.GetStructural().GetMaxPlanBytes())
	}
	if !checksumOptional {
		mandatory := mandatoryResult(plan)
		diagnosticBytes := int64(proto.Size(mandatory.GetDiagnostics()[0]))
		if diagnosticBytes > validator.limits.GetOutput().GetMaxDiagnosticBytes() {
			return admissionError(ErrorLimit, "$.limits.output.maxDiagnosticBytes",
				"mandatory diagnostic has %d bytes; limit is %d", diagnosticBytes,
				validator.limits.GetOutput().GetMaxDiagnosticBytes())
		}
		mandatoryBytes := int64(proto.Size(mandatory))
		if mandatoryBytes > validator.limits.GetOutput().GetMaxResultBytes() {
			return admissionError(ErrorLimit, "$.limits.output.maxResultBytes",
				"mandatory result has %d bytes; limit is %d", mandatoryBytes,
				validator.limits.GetOutput().GetMaxResultBytes())
		}
	}
	return nil
}

const sha256Size = 32

func inspectProtoSurface(message protoreflect.Message, path string, depth int64, stats *surfaceStats) error {
	if depth > MaximumNestingDepth {
		return admissionError(ErrorLimit, path, "protobuf nesting exceeds hard maximum %d", MaximumNestingDepth)
	}
	stats.maxDepth = max(stats.maxDepth, depth)
	if len(message.GetUnknown()) != 0 {
		return admissionError(ErrorUnknownField, path, "protobuf contains unknown fields")
	}
	var validationErr error
	message.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		fieldPath := path + "." + field.JSONName()
		if field.IsMap() {
			validationErr = admissionError(ErrorUnsupportedOperator, fieldPath, "maps are not in the plan vocabulary")
			return false
		}
		if field.IsList() {
			list := value.List()
			stats.maxCollection = max(stats.maxCollection, int64(list.Len()))
			if int64(list.Len()) > MaximumCollectionItems {
				validationErr = admissionError(ErrorLimit, fieldPath, "collection exceeds hard maximum %d", MaximumCollectionItems)
				return false
			}
			for index := 0; index < list.Len(); index++ {
				if err := inspectProtoValue(field, list.Get(index), fmt.Sprintf("%s[%d]", fieldPath, index), depth, stats); err != nil {
					validationErr = err
					return false
				}
			}
			return true
		}
		if err := inspectProtoValue(field, value, fieldPath, depth, stats); err != nil {
			validationErr = err
			return false
		}
		return true
	})
	return validationErr
}

func inspectProtoValue(
	field protoreflect.FieldDescriptor,
	value protoreflect.Value,
	path string,
	depth int64,
	stats *surfaceStats,
) error {
	switch field.Kind() {
	case protoreflect.EnumKind:
		if field.Enum().Values().ByNumber(value.Enum()) == nil {
			return admissionError(ErrorUnsupportedEnum, path, "enum value %d is not declared", value.Enum())
		}
	case protoreflect.MessageKind, protoreflect.GroupKind:
		if !value.Message().IsValid() {
			return admissionError(ErrorMalformedValue, path, "message is invalid")
		}
		return inspectProtoSurface(value.Message(), path, depth+1, stats)
	default:
	}
	return nil
}

func (v *validator) validateVersion() error {
	version := v.plan.GetVersion()
	if version == nil || version.GetMajor() != SupportedMajorVersion ||
		version.GetMinor() < 0 || version.GetMinor() > SupportedMinorVersion {
		return admissionError(ErrorUnsupportedVersion, "$.version", "reader supports version %d.%d",
			SupportedMajorVersion, SupportedMinorVersion)
	}
	return nil
}

func (v *validator) validateLimits(stats surfaceStats) error {
	if v.limits == nil || v.limits.GetStructural() == nil || v.limits.GetExecution() == nil ||
		v.limits.GetEvidence() == nil || v.limits.GetEvaluation() == nil || v.limits.GetOutput() == nil {
		return admissionError(ErrorLimit, "$.limits", "all independent Limit groups are required")
	}
	structural := v.limits.GetStructural()
	for _, limit := range []struct {
		path    string
		value   int64
		maximum int64
	}{
		{"structural.maxPlanBytes", structural.GetMaxPlanBytes(), MaximumPlanBytes},
		{"structural.maxNestingDepth", structural.GetMaxNestingDepth(), MaximumNestingDepth},
		{"structural.maxCollectionItems", structural.GetMaxCollectionItems(), MaximumCollectionItems},
		{"structural.maxOperatorCount", structural.GetMaxOperatorCount(), MaximumOperatorCount},
		{"execution.maxActions", v.limits.GetExecution().GetMaxActions(), MaximumActionCount},
		{"execution.maxFaults", v.limits.GetExecution().GetMaxFaults(), MaximumFaultCount},
		{"execution.maxPhaseAttempts", v.limits.GetExecution().GetMaxPhaseAttempts(), MaximumPhaseAttempts},
		{"execution.maxPhaseDurationMilliseconds", v.limits.GetExecution().GetMaxPhaseDurationMilliseconds(), MaximumPhaseDurationMilliseconds},
		{"execution.maxTotalDurationMilliseconds", v.limits.GetExecution().GetMaxTotalDurationMilliseconds(), MaximumTotalDurationMilliseconds},
		{"evidence.maxRecords", v.limits.GetEvidence().GetMaxRecords(), MaximumEvidenceRecords},
		{"evidence.maxBytes", v.limits.GetEvidence().GetMaxBytes(), MaximumEvidenceBytes},
		{"evidence.maxSources", v.limits.GetEvidence().GetMaxSources(), MaximumEvidenceSources},
		{"evaluation.maxExpressionDepth", v.limits.GetEvaluation().GetMaxExpressionDepth(), MaximumExpressionDepth},
		{"evaluation.maxWork", v.limits.GetEvaluation().GetMaxWork(), MaximumEvaluationWork},
		{"output.maxDiagnosticBytes", v.limits.GetOutput().GetMaxDiagnosticBytes(), MaximumDiagnosticBytes},
		{"output.maxResultBytes", v.limits.GetOutput().GetMaxResultBytes(), MaximumResultBytes},
	} {
		if limit.value <= 0 || limit.value > limit.maximum {
			return admissionError(ErrorLimit, "$.limits."+limit.path,
				"limit is %d; allowed range is 1..%d", limit.value, limit.maximum)
		}
	}
	if structural.GetMaxNestingDepth() < stats.maxDepth {
		return admissionError(ErrorLimit, "$.limits.structural.maxNestingDepth",
			"plan depth is %d; limit is %d", stats.maxDepth, structural.GetMaxNestingDepth())
	}
	if structural.GetMaxCollectionItems() < stats.maxCollection {
		return admissionError(ErrorLimit, "$.limits.structural.maxCollectionItems",
			"largest collection has %d items; limit is %d", stats.maxCollection, structural.GetMaxCollectionItems())
	}
	if !validNatural(v.limits.GetEvaluation().GetMaxNatural()) {
		return admissionError(ErrorLimit, "$.limits.evaluation.maxNatural", "natural bound is not canonical uint64 text")
	}
	if v.limits.GetOutput().GetMaxDiagnosticBytes() > v.limits.GetOutput().GetMaxResultBytes() {
		return admissionError(ErrorLimit, "$.limits.output.maxDiagnosticBytes", "diagnostic limit exceeds result limit")
	}
	return nil
}

func (v *validator) validateProvenance() error {
	switch provenance := v.plan.GetProvenance().(type) {
	case *umpirespb.PortableTestPlan_External:
		if provenance.External == nil {
			return admissionError(ErrorMalformedValue, "$.external", "external provenance is required")
		}
		return validateLocations(provenance.External.GetSources(), "$.external.sources")
	case *umpirespb.PortableTestPlan_ModelCompiled:
		model := provenance.ModelCompiled
		if model == nil {
			return admissionError(ErrorMalformedValue, "$.modelCompiled", "model provenance is required")
		}
		if err := v.validateBinding(model.GetTest(), "$.modelCompiled.test"); err != nil {
			return err
		}
		if err := v.validateBinding(model.GetQuery(), "$.modelCompiled.query"); err != nil {
			return err
		}
		if err := validateArtifactBinding(model.GetExperiment(), "$.modelCompiled.experiment", artifactv2.ExperimentFormat); err != nil {
			return err
		}
		if err := validateArtifactBinding(model.GetRuntimeConfig(), "$.modelCompiled.runtimeConfig", artifactv2.RuntimeConfigurationFormat); err != nil {
			return err
		}
		if model.GetExperiment().GetBehaviorFingerprint() != model.GetQuery().GetBehaviorFingerprint() {
			return admissionError(ErrorBinding, "$.modelCompiled.experiment", "Experiment behavior is crossed with the model query")
		}
		if model.GetRuntimeConfig().GetBehaviorFingerprint() != v.plan.GetExecution().GetRuntime().GetConfig().GetBehaviorFingerprint() {
			return admissionError(ErrorBinding, "$.modelCompiled.runtimeConfig", "RuntimeConfiguration behavior is crossed with the execution runtime")
		}
		if err := v.validateBindingCollection(model.GetProperties(), "$.modelCompiled.properties", true); err != nil {
			return err
		}
		if err := v.validateBinding(model.GetCompilerContract(), "$.modelCompiled.compilerContract"); err != nil {
			return err
		}
		return validateLocations(model.GetSources(), "$.modelCompiled.sources")
	default:
		return admissionError(ErrorMalformedValue, "$.provenance", "exactly one provenance kind is required")
	}
}

func (v *validator) validateExecution() error {
	execution := v.plan.GetExecution()
	if execution == nil {
		return admissionError(ErrorMalformedValue, "$.execution", "execution program is required")
	}
	checks := []func() error{
		func() error { return v.validateExecutionBindings(execution) },
		func() error { return v.validateExecutionRoles(execution) },
		func() error { return v.validateExecutionTrace(execution) },
		func() error {
			return v.validateBindingCollection(execution.GetCapabilityRequirements(), "$.execution.capabilityRequirements", true)
		},
		func() error { return v.validateCheckpoints(execution.GetCheckpoints()) },
		func() error { return v.validateRuntime(execution.GetRuntime()) },
	}
	for _, check := range checks {
		if err := check(); err != nil {
			return err
		}
	}
	return nil
}

func (v *validator) validateExecutionBindings(execution *umpirespb.ExecutionProgram) error {
	bindings := []struct {
		path    string
		binding *umpirespb.DefinitionBinding
	}{
		{"setup", execution.GetSetup()},
		{"query", execution.GetQuery()},
		{"behavior", execution.GetBehavior()},
		{"target", execution.GetTarget()},
		{"kernel", execution.GetKernel()},
	}
	for _, item := range bindings {
		if err := v.validateBinding(item.binding, "$.execution."+item.path); err != nil {
			return err
		}
	}
	if model := v.plan.GetModelCompiled(); model != nil && !proto.Equal(model.GetQuery(), execution.GetQuery()) {
		return admissionError(ErrorBinding, "$.modelCompiled.query", "model query is crossed with execution query")
	}
	return nil
}

func (v *validator) validateExecutionRoles(execution *umpirespb.ExecutionProgram) error {
	if len(execution.GetRoleBindings()) != 1 || len(execution.GetSymbolicRoles()) != 1 ||
		len(execution.GetRuntimeBindingSlots()) == 0 || len(execution.GetPreconditions()) == 0 {
		return admissionError(ErrorMalformedValue, "$.execution", "roles, runtime slots, and preconditions are required")
	}
	role := execution.GetRoleBindings()[0]
	if role == nil {
		return admissionError(ErrorMalformedValue, "$.execution.roleBindings[0]", "role binding is required")
	}
	if err := v.validateBinding(role.GetRole(), "$.execution.roleBindings[0].role"); err != nil {
		return err
	}
	if err := v.validateModelValue(role.GetValue(), "$.execution.roleBindings[0].value"); err != nil {
		return err
	}
	symbolic := execution.GetSymbolicRoles()[0]
	if symbolic == nil || symbolic.GetValueKind() == umpirespb.PORTABLE_VALUE_KIND_UNSPECIFIED {
		return admissionError(ErrorMalformedValue, "$.execution.symbolicRoles[0]", "symbolic role is malformed")
	}
	if err := v.validateBinding(symbolic.GetDefinition(), "$.execution.symbolicRoles[0].definition"); err != nil {
		return err
	}
	if !proto.Equal(role.GetRole(), symbolic.GetDefinition()) {
		return admissionError(ErrorBinding, "$.execution.roleBindings[0].role", "role is crossed with its symbolic declaration")
	}
	if err := v.validateBindingSlots(execution.GetRuntimeBindingSlots()); err != nil {
		return err
	}
	for index, precondition := range execution.GetPreconditions() {
		if err := v.validatePrecondition(precondition, fmt.Sprintf("$.execution.preconditions[%d]", index)); err != nil {
			return err
		}
	}
	return nil
}

func (v *validator) validateExecutionTrace(execution *umpirespb.ExecutionProgram) error {
	if err := v.validateModelValue(execution.GetInitialState(), "$.execution.initialState"); err != nil {
		return err
	}
	if len(execution.GetRequestedActions()) != 1 || int64(len(execution.GetRequestedActions())) > v.limits.GetExecution().GetMaxActions() {
		return admissionError(ErrorLimit, "$.execution.requestedActions", "version one requires exactly one action")
	}
	if len(execution.GetModelOutcomes()) != 1 || len(execution.GetResultingStates()) != 1 || len(execution.GetOccurrences()) != 1 {
		return admissionError(ErrorLimit, "$.execution", "version one requires one outcome, resulting state, and occurrence")
	}
	if len(execution.GetSelectedChoices()) == 0 || len(execution.GetSelectedVariants()) == 0 {
		return admissionError(ErrorMalformedValue, "$.execution", "selected choices and variants are required")
	}
	if int64(len(execution.GetRequestedFaults())) > v.limits.GetExecution().GetMaxFaults() {
		return admissionError(ErrorLimit, "$.execution.requestedFaults", "fault count exceeds the declared limit")
	}
	collections := []struct {
		name   string
		values []*umpirespb.ModelValue
	}{
		{"requestedActions", execution.GetRequestedActions()},
		{"modelOutcomes", execution.GetModelOutcomes()},
		{"resultingStates", execution.GetResultingStates()},
		{"selectedChoices", execution.GetSelectedChoices()},
		{"selectedVariants", execution.GetSelectedVariants()},
		{"requestedFaults", execution.GetRequestedFaults()},
	}
	for _, collection := range collections {
		if err := v.validateModelValues(collection.values, "$.execution."+collection.name); err != nil {
			return err
		}
	}
	occurrence := execution.GetOccurrences()[0]
	if occurrence == nil || !validDefinitionID(occurrence.GetActionDefinitionId()) || occurrence.GetPosition() != 1 ||
		occurrence.GetAuthoredDefinitionId() == "" || occurrence.GetAuthoredDefinitionId() != occurrence.GetDefinition().GetDefinitionId() {
		return admissionError(ErrorOrdering, "$.execution.occurrences[0]", "occurrence identity or order is invalid")
	}
	if err := v.validateBinding(occurrence.GetDefinition(), "$.execution.occurrences[0].definition"); err != nil {
		return err
	}
	if occurrence.GetActionDefinitionId() != execution.GetRequestedActions()[0].GetDefinition().GetDefinitionId() {
		return admissionError(ErrorBinding, "$.execution.occurrences[0].actionDefinitionId", "occurrence is crossed with requested action")
	}
	return nil
}

func (v *validator) validateCheckpoints(checkpoints []*umpirespb.ExecutionCheckpoint) error {
	if len(checkpoints) == 0 {
		return admissionError(ErrorMalformedValue, "$.execution.checkpoints", "at least one checkpoint is required")
	}
	for index, checkpoint := range checkpoints {
		path := fmt.Sprintf("$.execution.checkpoints[%d]", index)
		if checkpoint == nil || checkpoint.GetTransition() != int64(index+1) {
			return admissionError(ErrorOrdering, path, "checkpoint order is invalid")
		}
		if err := v.validateModelValues(checkpoint.GetObservations(), path+".observations"); err != nil {
			return err
		}
	}
	return nil
}

func (v *validator) validateBindingSlots(slots []*umpirespb.RuntimeBindingSlot) error {
	seen := make(map[string]struct{}, len(slots))
	for index, slot := range slots {
		path := fmt.Sprintf("$.execution.runtimeBindingSlots[%d]", index)
		if slot == nil || slot.GetValueKind() == umpirespb.PORTABLE_VALUE_KIND_UNSPECIFIED {
			return admissionError(ErrorMalformedValue, path, "runtime binding slot is malformed")
		}
		if err := v.validateBinding(slot.GetDefinition(), path+".definition"); err != nil {
			return err
		}
		id := slot.GetDefinition().GetDefinitionId()
		if _, duplicate := seen[id]; duplicate {
			return admissionError(ErrorDuplicate, path, "duplicate runtime binding slot")
		}
		seen[id] = struct{}{}
	}
	return nil
}

func (v *validator) validatePrecondition(precondition *umpirespb.ExecutionPrecondition, path string) error {
	if precondition == nil || precondition.GetOperator() != umpirespb.PRECONDITION_OPERATOR_EQUALS {
		return admissionError(ErrorUnsupportedOperator, path, "version one supports only equals")
	}
	v.operators++
	if err := v.validateBinding(precondition.GetDefinition(), path+".definition"); err != nil {
		return err
	}
	if err := v.validateExecutionOperand(precondition.GetLeft(), path+".left"); err != nil {
		return err
	}
	return v.validateExecutionOperand(precondition.GetRight(), path+".right")
}

func (v *validator) validateExecutionOperand(operand *umpirespb.ExecutionOperand, path string) error {
	if operand == nil {
		return admissionError(ErrorUnsupportedOperator, path, "operand is required")
	}
	switch value := operand.GetOperand().(type) {
	case *umpirespb.ExecutionOperand_Literal:
		return v.validateModelValue(value.Literal, path+".literal")
	case *umpirespb.ExecutionOperand_Binding:
		if err := v.validateBinding(value.Binding, path+".binding"); err != nil {
			return err
		}
		for _, slot := range v.plan.GetExecution().GetRuntimeBindingSlots() {
			if proto.Equal(value.Binding, slot.GetDefinition()) {
				return nil
			}
		}
		return admissionError(ErrorBinding, path+".binding", "operand is crossed with the declared runtime binding slots")
	default:
		return admissionError(ErrorUnsupportedOperator, path, "operand kind is required")
	}
}

func (v *validator) validateRuntime(runtime *umpirespb.RuntimeProgram) error {
	if runtime == nil {
		return admissionError(ErrorMalformedValue, "$.execution.runtime", "runtime program is required")
	}
	if err := v.validateRuntimeBindings(runtime); err != nil {
		return err
	}
	if len(runtime.GetParticipantBindings()) != 1 {
		return admissionError(ErrorLimit, "$.execution.runtime.participantBindings", "version one requires exactly one participant")
	}
	if err := v.validateParticipant(runtime.GetParticipantBindings()[0]); err != nil {
		return err
	}
	if err := v.validateObservationConfig(runtime.GetObservationConfig()); err != nil {
		return err
	}
	if err := v.validatePhaseLimits(runtime.GetPhaseLimits()); err != nil {
		return err
	}
	return v.validateTerminationAndCleanup(runtime)
}

func (v *validator) validateRuntimeBindings(runtime *umpirespb.RuntimeProgram) error {
	bindings := []struct {
		path    string
		binding *umpirespb.DefinitionBinding
	}{
		{"authorityProfile", runtime.GetAuthorityProfile()},
		{"config", runtime.GetConfig()},
	}
	for _, item := range bindings {
		if err := v.validateBinding(item.binding, "$.execution.runtime."+item.path); err != nil {
			return err
		}
	}
	return nil
}

func (v *validator) validateParticipant(participant *umpirespb.PortableParticipantBinding) error {
	const path = "$.execution.runtime.participantBindings[0]"
	if participant == nil || participant.GetProtocolVersion() != 2 {
		return admissionError(ErrorUnsupportedVersion, path, "participant protocol version 2 is required")
	}
	bindings := []struct {
		name    string
		binding *umpirespb.DefinitionBinding
	}{
		{"participant", participant.GetParticipant()},
		{"protocol", participant.GetProtocol()},
		{"program", participant.GetProgram()},
	}
	for _, item := range bindings {
		if err := v.validateBinding(item.binding, path+"."+item.name); err != nil {
			return err
		}
	}
	if err := v.validateBindingCollection(participant.GetCapabilities(), path+".capabilities", true); err != nil {
		return err
	}
	if !equalBindings(participant.GetCapabilities(), v.plan.GetExecution().GetCapabilityRequirements()) {
		return admissionError(ErrorBinding, path+".capabilities", "participant capabilities are crossed with execution requirements")
	}
	return nil
}

func (v *validator) validateObservationConfig(observation *umpirespb.PortableObservationConfig) error {
	const path = "$.execution.runtime.observationConfig"
	if observation == nil {
		return admissionError(ErrorBinding, path, "observation config is required")
	}
	bindings := []struct {
		name    string
		binding *umpirespb.DefinitionBinding
	}{
		{"profile", observation.GetProfile()},
		{"program", observation.GetProgram()},
		{"mapping", observation.GetMapping()},
	}
	for _, item := range bindings {
		if err := v.validateBinding(item.binding, path+"."+item.name); err != nil {
			return err
		}
	}
	return nil
}

func (v *validator) validatePhaseLimits(limits []*umpirespb.ExecutionPhaseLimit) error {
	if len(limits) != len(expectedPhases) {
		return admissionError(ErrorOrdering, "$.execution.runtime.phaseLimits", "the fixed five phases are required")
	}
	var totalDuration int64
	for index, expected := range expectedPhases {
		limit := limits[index]
		path := fmt.Sprintf("$.execution.runtime.phaseLimits[%d]", index)
		if limit == nil || limit.GetPhase() != expected.phase {
			return admissionError(ErrorOrdering, path, "phase order is invalid")
		}
		if limit.GetDurationMilliseconds() != expected.duration || limit.GetMaxAttempts() != 1 ||
			limit.GetMaxRecords() != expected.records || limit.GetMaxBytes() != expected.bytes {
			return admissionError(ErrorLimit, path, "phase Limit is not the supported version-one value")
		}
		if limit.GetDurationMilliseconds() > v.limits.GetExecution().GetMaxPhaseDurationMilliseconds() ||
			limit.GetMaxAttempts() > v.limits.GetExecution().GetMaxPhaseAttempts() ||
			limit.GetMaxRecords() > v.limits.GetEvidence().GetMaxRecords() ||
			limit.GetMaxBytes() > v.limits.GetEvidence().GetMaxBytes() {
			return admissionError(ErrorLimit, path, "phase Limit exceeds its independent plan Limit")
		}
		totalDuration += limit.GetDurationMilliseconds()
	}
	if totalDuration > v.limits.GetExecution().GetMaxTotalDurationMilliseconds() {
		return admissionError(ErrorLimit, "$.limits.execution.maxTotalDurationMilliseconds", "phase durations exceed total duration Limit")
	}
	return nil
}

func (v *validator) validateTerminationAndCleanup(runtime *umpirespb.RuntimeProgram) error {
	if runtime.GetTermination() == nil || runtime.GetCleanup() == nil {
		return admissionError(ErrorMalformedValue, "$.execution.runtime", "termination and cleanup obligations are required")
	}
	if err := v.validateBinding(runtime.GetTermination().GetDefinition(), "$.execution.runtime.termination.definition"); err != nil {
		return err
	}
	return v.validateBinding(runtime.GetCleanup().GetDefinition(), "$.execution.runtime.cleanup.definition")
}

func (v *validator) validateVerification() error {
	verification := v.plan.GetVerification()
	if verification == nil || verification.GetEvidence() == nil || verification.GetObservation() == nil {
		return admissionError(ErrorMalformedValue, "$.verification", "Evidence and Observation programs are required")
	}
	checks := []func() error{
		func() error { return v.validateEvidenceProfile(verification.GetEvidence(), "$.verification.evidence") },
		func() error {
			return v.validateObservation(verification.GetObservation(), "$.verification.observation")
		},
		func() error { return v.validateVerificationBindings(verification) },
		func() error { return v.validateTraceProjection(verification) },
	}
	for _, check := range checks {
		if err := check(); err != nil {
			return err
		}
	}
	propertyBindings, err := v.validateProperties(verification.GetProperties())
	if err != nil {
		return err
	}
	if model := v.plan.GetModelCompiled(); model != nil && !equalBindings(model.GetProperties(), propertyBindings) {
		return admissionError(ErrorBinding, "$.modelCompiled.properties", "model Property bindings are crossed with verification")
	}
	if verification.GetDecision().GetKind() != umpirespb.DECISION_POLICY_KIND_STRICT_V1 {
		return admissionError(ErrorUnsupportedOperator, "$.verification.decision", "strict version-one decision policy is required")
	}
	v.operators++
	return nil
}

func (v *validator) validateVerificationBindings(verification *umpirespb.VerificationProgram) error {
	if !proto.Equal(verification.GetEvidence(), verification.GetObservation().GetProfile()) {
		return admissionError(ErrorBinding, "$.verification.observation.profile", "Observation profile is crossed with Evidence profile")
	}
	runtimeObservation := v.plan.GetExecution().GetRuntime().GetObservationConfig()
	if !proto.Equal(runtimeObservation.GetProfile(), verification.GetEvidence().GetDefinition()) ||
		!proto.Equal(runtimeObservation.GetProgram(), verification.GetObservation().GetDefinition()) ||
		!proto.Equal(runtimeObservation.GetMapping(), verification.GetObservation().GetMapping()) {
		return admissionError(ErrorBinding, "$.execution.runtime.observationConfig", "runtime Observation bindings are crossed with verification")
	}
	return nil
}

func (v *validator) validateTraceProjection(verification *umpirespb.VerificationProgram) error {
	switch projection := verification.GetTraceProjection().(type) {
	case *umpirespb.VerificationProgram_DirectPlanTrace:
		if projection.DirectPlanTrace == nil {
			return admissionError(ErrorUnsupportedOperator, "$.verification.directPlanTrace", "direct projection marker is required")
		}
	case *umpirespb.VerificationProgram_RenameExactLink:
		if projection.RenameExactLink == nil {
			return admissionError(ErrorUnsupportedOperator, "$.verification.renameExactLink", "exact rename link is required")
		}
		if err := v.validateRenameLink(projection.RenameExactLink); err != nil {
			return err
		}
	default:
		return admissionError(ErrorUnsupportedOperator, "$.verification.traceProjection", "explicit direct or exact-rename projection is required")
	}
	v.operators++
	return nil
}

func (v *validator) validateProperties(properties []*umpirespb.Property) ([]*umpirespb.DefinitionBinding, error) {
	if len(properties) == 0 {
		return nil, admissionError(ErrorMalformedValue, "$.verification.properties", "at least one Property is required")
	}
	if err := requireSortedUnique(properties, func(property *umpirespb.Property) string {
		return bindingKey(property.GetDefinition())
	}, "$.verification.properties"); err != nil {
		return nil, err
	}
	propertyBindings := make([]*umpirespb.DefinitionBinding, 0, len(properties))
	for index, property := range properties {
		path := fmt.Sprintf("$.verification.properties[%d]", index)
		if err := v.validateProperty(property, path); err != nil {
			return nil, err
		}
		propertyBindings = append(propertyBindings, property.GetDefinition())
	}
	return propertyBindings, nil
}

func (v *validator) validateProperty(property *umpirespb.Property, path string) error {
	if property == nil {
		return admissionError(ErrorMalformedValue, path, "Property is required")
	}
	if err := v.validateBinding(property.GetDefinition(), path+".definition"); err != nil {
		return err
	}
	if err := validateLocation(property.GetSource(), path+".source"); err != nil {
		return err
	}
	if err := v.validateBindingCollection(property.GetRequirements(), path+".requirements", false); err != nil {
		return err
	}
	return v.validateClauses(property.GetClauses(), path+".clauses")
}

func (v *validator) validateClauses(clauses []*umpirespb.PropertyClause, path string) error {
	if len(clauses) == 0 {
		return admissionError(ErrorMalformedValue, path, "Property clauses are required")
	}
	if err := requireSortedUnique(clauses, func(clause *umpirespb.PropertyClause) string {
		return clause.GetDefinitionId()
	}, path); err != nil {
		return err
	}
	for index, clause := range clauses {
		clausePath := fmt.Sprintf("%s[%d]", path, index)
		if clause == nil || !validDefinitionID(clause.GetDefinitionId()) ||
			clause.GetProvenance() == umpirespb.PROPERTY_CLAUSE_PROVENANCE_UNSPECIFIED || clause.GetPerStepImplies() == nil {
			return admissionError(ErrorUnsupportedOperator, clausePath, "portable per-step clause is malformed")
		}
		v.operators++
		if err := v.validatePattern(clause.GetPerStepImplies().GetTrigger(), clausePath+".trigger"); err != nil {
			return err
		}
		if err := v.validatePattern(clause.GetPerStepImplies().GetRequired(), clausePath+".required"); err != nil {
			return err
		}
	}
	return nil
}

func (v *validator) validateEvidenceProfile(profile *umpirespb.EvidenceProfile, path string) error {
	if profile.GetVersion() != 1 {
		return admissionError(ErrorUnsupportedVersion, path+".version", "Evidence profile version 1 is required")
	}
	if err := v.validateBinding(profile.GetDefinition(), path+".definition"); err != nil {
		return err
	}
	sources, err := v.validateEvidenceSources(profile.GetSources(), path+".sources")
	if err != nil {
		return err
	}
	digestPolicies, err := validateDigestPolicies(profile.GetDigestPolicies(), path+".digestPolicies")
	if err != nil {
		return err
	}
	kinds, err := v.validateEvidenceKinds(profile.GetKinds(), sources, digestPolicies, path+".kinds")
	if err != nil {
		return err
	}
	if err := validateEvidenceCardinalities(profile.GetCardinalities(), kinds, v.limits.GetEvidence().GetMaxRecords(), path+".cardinalities"); err != nil {
		return err
	}
	return validateCorrelationSlots(profile.GetCorrelationSlots(), kinds, path+".correlationSlots")
}

func (v *validator) validateEvidenceSources(sources []*umpirespb.EvidenceSourceDeclaration, path string) (map[string]struct{}, error) {
	if len(sources) == 0 || int64(len(sources)) > v.limits.GetEvidence().GetMaxSources() {
		return nil, admissionError(ErrorLimit, path, "Evidence source count is outside the declared Limit")
	}
	if err := requireSortedUnique(sources, func(source *umpirespb.EvidenceSourceDeclaration) string {
		return source.GetSourceDefinitionId()
	}, path); err != nil {
		return nil, err
	}
	seen := make(map[string]struct{}, len(sources))
	for index, source := range sources {
		id := source.GetSourceDefinitionId()
		if !validDefinitionID(id) {
			return nil, admissionError(ErrorMalformedValue, fmt.Sprintf("%s[%d]", path, index), "Evidence source identity is malformed")
		}
		if _, duplicate := seen[id]; duplicate {
			return nil, admissionError(ErrorDuplicate, fmt.Sprintf("%s[%d]", path, index), "duplicate Evidence source")
		}
		seen[id] = struct{}{}
	}
	return seen, nil
}

func validateDigestPolicies(policies []*umpirespb.DigestPolicy, path string) (map[string]struct{}, error) {
	if err := requireSortedUnique(policies, func(policy *umpirespb.DigestPolicy) string {
		return policy.GetDefinitionId()
	}, path); err != nil {
		return nil, err
	}
	seen := make(map[string]struct{}, len(policies))
	for index, policy := range policies {
		policyPath := fmt.Sprintf("%s[%d]", path, index)
		if policy == nil || !validDefinitionID(policy.GetDefinitionId()) {
			return nil, admissionError(ErrorMalformedValue, policyPath, "digest policy identity is malformed")
		}
		if policy.GetAlgorithm() != umpirespb.DIGEST_ALGORITHM_SYNTHETIC_DIGEST_V1 {
			return nil, admissionError(ErrorUnsupportedOperator, policyPath+".algorithm", "digest algorithm is unsupported")
		}
		seen[policy.GetDefinitionId()] = struct{}{}
	}
	return seen, nil
}

func (v *validator) validateEvidenceKinds(
	kinds []*umpirespb.EvidenceKindDeclaration,
	sources map[string]struct{},
	digestPolicies map[string]struct{},
	path string,
) (map[string]map[string]struct{}, error) {
	if len(kinds) == 0 {
		return nil, admissionError(ErrorMalformedValue, path, "Evidence kind declarations are required")
	}
	if err := requireSortedUnique(kinds, func(kind *umpirespb.EvidenceKindDeclaration) string {
		return kind.GetKindDefinitionId()
	}, path); err != nil {
		return nil, err
	}
	seen := make(map[string]map[string]struct{}, len(kinds))
	for index, kind := range kinds {
		kindPath := fmt.Sprintf("%s[%d]", path, index)
		if kind == nil || !validDefinitionID(kind.GetKindDefinitionId()) {
			return nil, admissionError(ErrorMalformedValue, kindPath, "Evidence kind identity is malformed")
		}
		if _, ok := sources[kind.GetSourceDefinitionId()]; !ok {
			return nil, admissionError(ErrorBinding, kindPath+".sourceDefinitionId", "Evidence kind has an undeclared source")
		}
		fields, err := validateEvidenceFields(kind.GetFields(), digestPolicies, kindPath+".fields")
		if err != nil {
			return nil, err
		}
		seen[kind.GetKindDefinitionId()] = fields
	}
	return seen, nil
}

func validateEvidenceFields(
	fields []*umpirespb.EvidenceFieldDeclaration,
	digestPolicies map[string]struct{},
	path string,
) (map[string]struct{}, error) {
	if len(fields) == 0 {
		return nil, admissionError(ErrorMalformedValue, path, "Evidence fields are required")
	}
	if err := requireSortedUnique(fields, func(field *umpirespb.EvidenceFieldDeclaration) string {
		return field.GetFieldDefinitionId()
	}, path); err != nil {
		return nil, err
	}
	seen := make(map[string]struct{}, len(fields))
	for index, field := range fields {
		fieldPath := fmt.Sprintf("%s[%d]", path, index)
		if field == nil || !validDefinitionID(field.GetFieldDefinitionId()) ||
			field.GetValueKind() == umpirespb.VALUE_KIND_UNSPECIFIED ||
			field.GetDisposition() == umpirespb.FIELD_DISPOSITION_KIND_UNSPECIFIED {
			return nil, admissionError(ErrorMalformedValue, fieldPath, "Evidence field declaration is malformed")
		}
		if field.GetDisposition() == umpirespb.FIELD_DISPOSITION_KIND_HASH {
			if _, ok := digestPolicies[field.GetDigestPolicyDefinitionId()]; !ok {
				return nil, admissionError(ErrorBinding, fieldPath+".digestPolicyDefinitionId", "digest policy is not declared")
			}
		} else if field.GetDigestPolicyDefinitionId() != "" {
			return nil, admissionError(ErrorBinding, fieldPath+".digestPolicyDefinitionId", "only hashed fields may name a digest policy")
		}
		seen[field.GetFieldDefinitionId()] = struct{}{}
	}
	return seen, nil
}

func validateEvidenceCardinalities(
	cardinalities []*umpirespb.EvidenceCardinality,
	kinds map[string]map[string]struct{},
	maxRecords int64,
	path string,
) error {
	if err := requireSortedUnique(cardinalities, func(cardinality *umpirespb.EvidenceCardinality) string {
		return cardinality.GetKindDefinitionId()
	}, path); err != nil {
		return err
	}
	for index, cardinality := range cardinalities {
		cardinalityPath := fmt.Sprintf("%s[%d]", path, index)
		if cardinality == nil || cardinality.GetMinimum() < 0 || cardinality.GetMaximum() < cardinality.GetMinimum() ||
			cardinality.GetMaximum() > maxRecords {
			return admissionError(ErrorLimit, cardinalityPath, "Evidence cardinality is malformed")
		}
		if _, ok := kinds[cardinality.GetKindDefinitionId()]; !ok {
			return admissionError(ErrorBinding, cardinalityPath, "Evidence cardinality has an undeclared kind")
		}
	}
	return nil
}

func validateCorrelationSlots(
	slots []*umpirespb.CorrelationSlot,
	kinds map[string]map[string]struct{},
	path string,
) error {
	if err := requireSortedUnique(slots, func(slot *umpirespb.CorrelationSlot) string {
		return slot.GetDefinitionId()
	}, path); err != nil {
		return err
	}
	for index, slot := range slots {
		slotPath := fmt.Sprintf("%s[%d]", path, index)
		if slot == nil || !validDefinitionID(slot.GetDefinitionId()) || slot.GetKind() == umpirespb.CORRELATION_SLOT_KIND_UNSPECIFIED {
			return admissionError(ErrorMalformedValue, slotPath, "correlation slot is malformed")
		}
		if len(slot.GetFields()) == 0 {
			return admissionError(ErrorMalformedValue, slotPath+".fields", "correlation fields are required")
		}
		if err := requireSortedUnique(slot.GetFields(), evidenceFieldReferenceKey, slotPath+".fields"); err != nil {
			return err
		}
		for fieldIndex, field := range slot.GetFields() {
			if !isEvidenceFieldDeclared(kinds, field) {
				return admissionError(ErrorBinding, fmt.Sprintf("%s.fields[%d]", slotPath, fieldIndex), "correlation field is not declared")
			}
		}
	}
	return nil
}

func (v *validator) validateObservation(observation *umpirespb.ObservationProgram, path string) error {
	if observation.GetMappingVersion() != 1 || len(observation.GetEmits()) == 0 {
		return admissionError(ErrorUnsupportedVersion, path, "Observation mapping version 1 with emits is required")
	}
	if err := v.validateBinding(observation.GetDefinition(), path+".definition"); err != nil {
		return err
	}
	if err := v.validateBinding(observation.GetMapping(), path+".mapping"); err != nil {
		return err
	}
	if err := validateLocation(observation.GetSource(), path+".source"); err != nil {
		return err
	}
	if err := requireSortedUnique(observation.GetEmits(), func(emit *umpirespb.Emit) string {
		return emit.GetDefinitionId()
	}, path+".emits"); err != nil {
		return err
	}
	seen := make(map[string]struct{}, len(observation.GetEmits()))
	coordinates := make(map[string]struct{}, len(observation.GetEmits()))
	for index, emit := range observation.GetEmits() {
		emitPath := fmt.Sprintf("%s.emits[%d]", path, index)
		if emit == nil || !validDefinitionID(emit.GetDefinitionId()) || !validDefinitionID(emit.GetSourceKindDefinitionId()) ||
			emit.GetOutputKind() == umpirespb.DEFINITION_KIND_UNSPECIFIED || emit.GetCoordinate() == nil {
			return admissionError(ErrorMalformedValue, emitPath, "Emit is malformed")
		}
		if !v.isDeclaredEvidenceKind(emit.GetSourceKindDefinitionId()) {
			return admissionError(ErrorBinding, emitPath+".sourceKindDefinitionId", "Emit source kind is not declared")
		}
		seen[emit.GetDefinitionId()] = struct{}{}
		if err := validateCoordinate(emit.GetCoordinate(), emitPath+".coordinate"); err != nil {
			return err
		}
		coordinate := coordinateKey(emit.GetCoordinate())
		if _, duplicate := coordinates[coordinate]; duplicate {
			return admissionError(ErrorDuplicate, emitPath+".coordinate", "duplicate emitted coordinate")
		}
		coordinates[coordinate] = struct{}{}
		if err := v.validateBinding(emit.GetOutputDefinition(), emitPath+".outputDefinition"); err != nil {
			return err
		}
		if err := v.validateObservationExpression(emit.GetCondition(), emitPath+".condition", 1); err != nil {
			return err
		}
		if err := v.validateObservationExpression(emit.GetValue(), emitPath+".value", 1); err != nil {
			return err
		}
	}
	return validateEmitOrdering(observation.GetOrdering(), seen, path+".ordering")
}

func (v *validator) validateObservationExpression(expression *umpirespb.ObservationExpression, path string, depth int64) error {
	if expression == nil || depth > v.limits.GetEvaluation().GetMaxExpressionDepth() {
		return admissionError(ErrorLimit, path, "Observation expression exceeds its depth Limit")
	}
	v.operators++
	switch operator := expression.GetOperator().(type) {
	case *umpirespb.ObservationExpression_LiteralText:
		if operator.LiteralText == nil {
			return admissionError(ErrorMalformedValue, path, "literal text is required")
		}
	case *umpirespb.ObservationExpression_LiteralNatural:
		if operator.LiteralNatural == nil || !v.validBoundedNatural(operator.LiteralNatural.GetValue()) {
			return admissionError(ErrorMalformedValue, path, "literal natural is malformed")
		}
	case *umpirespb.ObservationExpression_Field:
		if operator.Field == nil || !validDefinitionID(operator.Field.GetKindDefinitionId()) || !validDefinitionID(operator.Field.GetFieldDefinitionId()) {
			return admissionError(ErrorBinding, path, "Evidence field reference is malformed")
		}
		if !v.isDeclaredEvidenceField(operator.Field) {
			return admissionError(ErrorBinding, path, "Evidence field reference is crossed with the declared profile")
		}
	case *umpirespb.ObservationExpression_NaturalRenderV1:
		return v.validateObservationExpression(operator.NaturalRenderV1.GetOperand(), path+".naturalRenderV1", depth+1)
	case *umpirespb.ObservationExpression_Present:
		return v.validateObservationExpression(operator.Present.GetOperand(), path+".present", depth+1)
	case *umpirespb.ObservationExpression_Equals:
		if err := v.validateObservationExpression(operator.Equals.GetLeft(), path+".equals.left", depth+1); err != nil {
			return err
		}
		return v.validateObservationExpression(operator.Equals.GetRight(), path+".equals.right", depth+1)
	case *umpirespb.ObservationExpression_All:
		return v.validateExpressionList(operator.All.GetOperands(), path+".all", depth+1)
	case *umpirespb.ObservationExpression_Any:
		return v.validateExpressionList(operator.Any.GetOperands(), path+".any", depth+1)
	default:
		return admissionError(ErrorUnsupportedOperator, path, "Observation operator is required")
	}
	return nil
}

func (v *validator) isDeclaredEvidenceField(reference *umpirespb.EvidenceFieldReference) bool {
	profile := v.plan.GetVerification().GetEvidence()
	for _, kind := range profile.GetKinds() {
		if kind.GetKindDefinitionId() != reference.GetKindDefinitionId() {
			continue
		}
		for _, field := range kind.GetFields() {
			if field.GetFieldDefinitionId() == reference.GetFieldDefinitionId() {
				return true
			}
		}
	}
	return false
}

func (v *validator) isDeclaredEvidenceKind(kindID string) bool {
	for _, kind := range v.plan.GetVerification().GetEvidence().GetKinds() {
		if kind.GetKindDefinitionId() == kindID {
			return true
		}
	}
	return false
}

func validateEmitOrdering(ordering []*umpirespb.EmitOrdering, emitIDs map[string]struct{}, path string) error {
	if err := requireSortedUnique(ordering, func(edge *umpirespb.EmitOrdering) string {
		return edge.GetPredecessorEmitDefinitionId() + "\x00" + edge.GetSuccessorEmitDefinitionId()
	}, path); err != nil {
		return err
	}
	for index, edge := range ordering {
		edgePath := fmt.Sprintf("%s[%d]", path, index)
		if edge == nil || edge.GetPredecessorEmitDefinitionId() == edge.GetSuccessorEmitDefinitionId() {
			return admissionError(ErrorBinding, edgePath, "Emit ordering is malformed")
		}
		if _, ok := emitIDs[edge.GetPredecessorEmitDefinitionId()]; !ok {
			return admissionError(ErrorBinding, edgePath+".predecessorEmitDefinitionId", "Emit is not declared")
		}
		if _, ok := emitIDs[edge.GetSuccessorEmitDefinitionId()]; !ok {
			return admissionError(ErrorBinding, edgePath+".successorEmitDefinitionId", "Emit is not declared")
		}
	}
	if hasEmitOrderingCycle(ordering, emitIDs) {
		return admissionError(ErrorOrdering, path, "Emit ordering contains a cycle")
	}
	return nil
}

func hasEmitOrderingCycle(ordering []*umpirespb.EmitOrdering, emitIDs map[string]struct{}) bool {
	adjacency := make(map[string][]string, len(emitIDs))
	indegree := make(map[string]int, len(emitIDs))
	for emitID := range emitIDs {
		indegree[emitID] = 0
	}
	for _, edge := range ordering {
		predecessor := edge.GetPredecessorEmitDefinitionId()
		successor := edge.GetSuccessorEmitDefinitionId()
		adjacency[predecessor] = append(adjacency[predecessor], successor)
		indegree[successor]++
	}
	ready := make([]string, 0, len(emitIDs))
	for emitID, degree := range indegree {
		if degree == 0 {
			ready = append(ready, emitID)
		}
	}
	visited := 0
	for len(ready) > 0 {
		current := ready[len(ready)-1]
		ready = ready[:len(ready)-1]
		visited++
		for _, successor := range adjacency[current] {
			indegree[successor]--
			if indegree[successor] == 0 {
				ready = append(ready, successor)
			}
		}
	}
	return visited != len(emitIDs)
}

func (v *validator) validateExpressionList(expressions []*umpirespb.ObservationExpression, path string, depth int64) error {
	if len(expressions) == 0 {
		return admissionError(ErrorUnsupportedOperator, path, "operator operands are required")
	}
	for index, expression := range expressions {
		if err := v.validateObservationExpression(expression, fmt.Sprintf("%s[%d]", path, index), depth); err != nil {
			return err
		}
	}
	return nil
}

func (v *validator) validatePattern(pattern *umpirespb.Pattern, path string) error {
	if pattern == nil || pattern.GetField() == umpirespb.TRACE_FIELD_UNSPECIFIED {
		return admissionError(ErrorMalformedValue, path, "Property pattern is malformed")
	}
	if err := v.validateBinding(pattern.GetDefinition(), path+".definition"); err != nil {
		return err
	}
	if v.plan.GetVerification().GetDirectPlanTrace() != nil && !v.isDirectTraceBinding(pattern.GetField(), pattern.GetDefinition()) {
		return admissionError(ErrorBinding, path+".definition", "Property pattern is crossed with the direct execution trace")
	}
	v.operators++
	switch operator := pattern.GetOperator().(type) {
	case *umpirespb.Pattern_EqualsText:
		if operator.EqualsText == nil {
			return admissionError(ErrorUnsupportedOperator, path, "equals-text operand is required")
		}
	case *umpirespb.Pattern_NaturalAtMost:
		if operator.NaturalAtMost == nil || !v.validBoundedNatural(operator.NaturalAtMost.GetBound()) {
			return admissionError(ErrorMalformedValue, path, "natural bound is malformed")
		}
	default:
		return admissionError(ErrorUnsupportedOperator, path, "Property pattern operator is required")
	}
	return nil
}

func (v *validator) isDirectTraceBinding(field umpirespb.TraceField, binding *umpirespb.DefinitionBinding) bool {
	execution := v.plan.GetExecution()
	var values []*umpirespb.ModelValue
	switch field {
	case umpirespb.TRACE_FIELD_INITIAL_STATE, umpirespb.TRACE_FIELD_PRIOR_STATE:
		values = []*umpirespb.ModelValue{execution.GetInitialState()}
	case umpirespb.TRACE_FIELD_SELECTED_ACTION:
		values = execution.GetRequestedActions()
	case umpirespb.TRACE_FIELD_MODEL_OUTCOME:
		values = execution.GetModelOutcomes()
	case umpirespb.TRACE_FIELD_RESULTING_STATE:
		values = execution.GetResultingStates()
	case umpirespb.TRACE_FIELD_OBSERVATION:
		for _, checkpoint := range execution.GetCheckpoints() {
			values = append(values, checkpoint.GetObservations()...)
		}
	default:
		return false
	}
	for _, value := range values {
		if proto.Equal(value.GetDefinition(), binding) {
			return true
		}
	}
	return false
}

func (v *validator) validateRenameLink(link *umpirespb.RenameExactLink) error {
	const path = "$.verification.renameExactLink"
	bindings := []struct {
		name    string
		binding *umpirespb.DefinitionBinding
	}{
		{"definition", link.GetDefinition()},
		{"sourceTarget", link.GetSourceTarget()},
		{"destinationTarget", link.GetDestinationTarget()},
	}
	for _, item := range bindings {
		if err := v.validateBinding(item.binding, path+"."+item.name); err != nil {
			return err
		}
	}
	if proto.Equal(link.GetSourceTarget(), link.GetDestinationTarget()) || len(link.GetEntries()) == 0 {
		return admissionError(ErrorBinding, path, "exact rename targets and entries are required")
	}
	if link.GetApplicationLimit() == nil || link.GetApplicationLimit().GetValue() <= 0 || link.GetApplicationLimit().GetUnit() != "semantic-transitions" {
		return admissionError(ErrorLimit, path+".applicationLimit", "positive semantic-transitions Limit is required")
	}
	if link.GetApplicationLimit().GetValue() > v.limits.GetEvaluation().GetMaxWork() {
		return admissionError(ErrorLimit, path+".applicationLimit", "application Limit exceeds evaluation work Limit")
	}
	if err := validateLocation(link.GetSource(), path+".source"); err != nil {
		return err
	}
	if err := v.validateRenameEntries(link.GetEntries(), path+".entries"); err != nil {
		return err
	}
	return v.validateDefinitionRenameEntries(link.GetDefinitionEntries(), path+".definitionEntries")
}

func (v *validator) validateRenameEntries(entries []*umpirespb.RenameExactEntry, path string) error {
	if err := requireSortedUnique(entries, renameEntryKey, path); err != nil {
		return err
	}
	sources := make(map[string]struct{}, len(entries))
	for index, entry := range entries {
		entryPath := fmt.Sprintf("%s[%d]", path, index)
		if entry == nil {
			return admissionError(ErrorMalformedValue, entryPath, "rename entry is required")
		}
		if err := v.validateModelValue(entry.GetSource(), entryPath+".source"); err != nil {
			return err
		}
		source := modelValueKey(entry.GetSource())
		if _, duplicate := sources[source]; duplicate {
			return admissionError(ErrorDuplicate, entryPath+".source", "rename source has duplicate or contradictory mappings")
		}
		sources[source] = struct{}{}
		if err := v.validateModelValue(entry.GetDestination(), entryPath+".destination"); err != nil {
			return err
		}
	}
	return nil
}

func (v *validator) validateDefinitionRenameEntries(entries []*umpirespb.DefinitionRenameEntry, path string) error {
	if err := requireSortedUnique(entries, definitionRenameEntryKey, path); err != nil {
		return err
	}
	sources := make(map[string]struct{}, len(entries))
	for index, entry := range entries {
		entryPath := fmt.Sprintf("%s[%d]", path, index)
		if entry == nil || entry.GetKind() == umpirespb.DEFINITION_KIND_UNSPECIFIED {
			return admissionError(ErrorMalformedValue, entryPath, "definition rename entry is malformed")
		}
		if err := v.validateBinding(entry.GetSource(), entryPath+".source"); err != nil {
			return err
		}
		source := bindingKey(entry.GetSource()) + "\x00" + strconv.FormatInt(int64(entry.GetKind()), 10)
		if _, duplicate := sources[source]; duplicate {
			return admissionError(ErrorDuplicate, entryPath+".source", "definition rename source has duplicate or contradictory mappings")
		}
		sources[source] = struct{}{}
		if err := v.validateBinding(entry.GetDestination(), entryPath+".destination"); err != nil {
			return err
		}
	}
	return nil
}

func (v *validator) validateKnownGaps() error {
	seen := make(map[string]struct{}, len(v.plan.GetKnownGaps()))
	for index, gap := range v.plan.GetKnownGaps() {
		path := fmt.Sprintf("$.knownGaps[%d]", index)
		if gap == nil || gap.GetKind() == umpirespb.KNOWN_GAP_KIND_UNSPECIFIED || !validDefinitionID(gap.GetCode()) ||
			len(gap.GetDetail()) > artifact.MaximumDiagnosticBytes {
			return admissionError(ErrorMalformedValue, path, "Known Gap is malformed")
		}
		if _, duplicate := seen[gap.GetCode()]; duplicate {
			return admissionError(ErrorDuplicate, path, "duplicate Known Gap")
		}
		seen[gap.GetCode()] = struct{}{}
	}
	return nil
}

func (v *validator) validateObligations() error {
	seen := make(map[string]struct{}, len(v.plan.GetExternalObligations()))
	for index, obligation := range v.plan.GetExternalObligations() {
		path := fmt.Sprintf("$.externalObligations[%d]", index)
		if obligation == nil || obligation.GetKind() == umpirespb.EXTERNAL_VERIFICATION_OBLIGATION_KIND_UNSPECIFIED ||
			obligation.GetStatement() == "" || len(obligation.GetStatement()) > artifact.MaximumDiagnosticBytes {
			return admissionError(ErrorMalformedValue, path, "external verification obligation is malformed")
		}
		if err := v.validateBinding(obligation.GetDefinition(), path+".definition"); err != nil {
			return err
		}
		if err := validateLocation(obligation.GetSource(), path+".source"); err != nil {
			return err
		}
		id := obligation.GetDefinition().GetDefinitionId()
		if _, duplicate := seen[id]; duplicate {
			return admissionError(ErrorDuplicate, path, "duplicate external verification obligation")
		}
		seen[id] = struct{}{}
	}
	return nil
}

func (v *validator) validateBinding(binding *umpirespb.DefinitionBinding, path string) error {
	if binding == nil || !validDefinitionID(binding.GetDefinitionId()) ||
		len(binding.GetDefinitionId()) > artifact.MaximumIdentityBytes ||
		!artifactv2.ValidDigest(binding.GetBehaviorFingerprint()) {
		return admissionError(ErrorBinding, path, "definition binding is malformed")
	}
	if fingerprint, ok := v.bindings[binding.GetDefinitionId()]; ok && fingerprint != binding.GetBehaviorFingerprint() {
		return admissionError(ErrorBinding, path, "definition identity is crossed with another fingerprint")
	}
	v.bindings[binding.GetDefinitionId()] = binding.GetBehaviorFingerprint()
	return nil
}

func (v *validator) validateBindingCollection(bindings []*umpirespb.DefinitionBinding, path string, required bool) error {
	if required && len(bindings) == 0 {
		return admissionError(ErrorMalformedValue, path, "binding collection is required")
	}
	if err := requireSortedUnique(bindings, bindingKey, path); err != nil {
		return err
	}
	for index, binding := range bindings {
		bindingPath := fmt.Sprintf("%s[%d]", path, index)
		if err := v.validateBinding(binding, bindingPath); err != nil {
			return err
		}
	}
	return nil
}

func (v *validator) validateModelValues(values []*umpirespb.ModelValue, path string) error {
	seen := make(map[string]struct{}, len(values))
	for index, value := range values {
		valuePath := fmt.Sprintf("%s[%d]", path, index)
		if err := v.validateModelValue(value, valuePath); err != nil {
			return err
		}
		id := value.GetDefinition().GetDefinitionId()
		if _, duplicate := seen[id]; duplicate {
			return admissionError(ErrorDuplicate, valuePath, "duplicate model value identity")
		}
		seen[id] = struct{}{}
	}
	return nil
}

func (v *validator) validateModelValue(value *umpirespb.ModelValue, path string) error {
	if value == nil || value.GetKind() == umpirespb.DEFINITION_KIND_UNSPECIFIED || value.GetValue() == nil {
		return admissionError(ErrorMalformedValue, path, "model value is malformed")
	}
	if err := v.validateBinding(value.GetDefinition(), path+".definition"); err != nil {
		return err
	}
	switch scalar := value.GetValue().GetValue().(type) {
	case *umpirespb.Value_Text:
		if len(scalar.Text) > artifact.MaximumDiagnosticBytes {
			return admissionError(ErrorByteLimit, path+".value.text", "text value exceeds its byte maximum")
		}
	case *umpirespb.Value_Natural:
		if !v.validBoundedNatural(scalar.Natural) {
			return admissionError(ErrorMalformedValue, path+".value.natural", "natural is malformed or out of range")
		}
	case *umpirespb.Value_BoolValue:
	default:
		return admissionError(ErrorMalformedValue, path+".value", "scalar value kind is required")
	}
	return nil
}

func (v *validator) validBoundedNatural(value string) bool {
	return validNatural(value) && compareNatural(value, v.limits.GetEvaluation().GetMaxNatural()) <= 0
}

func validateArtifactBinding(binding *umpirespb.ArtifactBinding, path, format string) error {
	if binding == nil || binding.GetFormatVersion() != format || !artifactv2.ValidDigest(binding.GetArtifactChecksum()) ||
		!artifactv2.ValidDigest(binding.GetBehaviorFingerprint()) || !artifactv2.ValidDigest(binding.GetProvenanceChecksum()) {
		return admissionError(ErrorBinding, path, "artifact binding is malformed")
	}
	return nil
}

func validateLocations(locations []*umpirespb.SourceLocation, path string) error {
	if len(locations) == 0 {
		return admissionError(ErrorMalformedValue, path, "source locations are required")
	}
	if !slices.IsSortedFunc(locations, func(left, right *umpirespb.SourceLocation) int {
		return strings.Compare(locationKey(left), locationKey(right))
	}) {
		return admissionError(ErrorOrdering, path, "source locations are not in canonical order")
	}
	for index, location := range locations {
		if err := validateLocation(location, fmt.Sprintf("%s[%d]", path, index)); err != nil {
			return err
		}
		if index > 0 && locationKey(locations[index-1]) == locationKey(location) {
			return admissionError(ErrorDuplicate, fmt.Sprintf("%s[%d]", path, index), "duplicate source location")
		}
	}
	return nil
}

func validateLocation(location *umpirespb.SourceLocation, path string) error {
	if location == nil || location.GetPath() == "" || location.GetLine() <= 0 || location.GetColumn() <= 0 ||
		location.GetProvenance() == "" || len(location.GetPath()) > artifact.MaximumIdentityBytes ||
		len(location.GetProvenance()) > artifact.MaximumDiagnosticBytes {
		return admissionError(ErrorMalformedValue, path, "source location is malformed")
	}
	return nil
}

func locationKey(location *umpirespb.SourceLocation) string {
	if location == nil {
		return ""
	}
	return fmt.Sprintf("%s\x00%020d\x00%020d\x00%s", location.GetPath(), location.GetLine(), location.GetColumn(), location.GetProvenance())
}

func equalBindings(left, right []*umpirespb.DefinitionBinding) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if !proto.Equal(left[index], right[index]) {
			return false
		}
	}
	return true
}

func requireSortedUnique[T any](values []T, key func(T) string, path string) error {
	if !slices.IsSortedFunc(values, func(left, right T) int {
		return strings.Compare(key(left), key(right))
	}) {
		return admissionError(ErrorOrdering, path, "collection is not in canonical order")
	}
	for index := 1; index < len(values); index++ {
		if key(values[index-1]) == key(values[index]) {
			return admissionError(ErrorDuplicate, fmt.Sprintf("%s[%d]", path, index), "duplicate identity")
		}
	}
	return nil
}

func isEvidenceFieldDeclared(
	kinds map[string]map[string]struct{},
	reference *umpirespb.EvidenceFieldReference,
) bool {
	if reference == nil {
		return false
	}
	fields, ok := kinds[reference.GetKindDefinitionId()]
	if !ok {
		return false
	}
	_, ok = fields[reference.GetFieldDefinitionId()]
	return ok
}

func evidenceFieldReferenceKey(reference *umpirespb.EvidenceFieldReference) string {
	if reference == nil {
		return ""
	}
	return reference.GetKindDefinitionId() + "\x00" + reference.GetFieldDefinitionId()
}

func validateCoordinate(coordinate *umpirespb.ModelCoordinate, path string) error {
	if coordinate == nil {
		return admissionError(ErrorMalformedValue, path, "Model coordinate is malformed")
	}
	switch coordinate.GetField() {
	case umpirespb.TRACE_FIELD_INITIAL_STATE:
		if coordinate.GetStep() == 0 && coordinate.GetPosition() == 0 {
			return nil
		}
	case umpirespb.TRACE_FIELD_PRIOR_STATE,
		umpirespb.TRACE_FIELD_SELECTED_ACTION,
		umpirespb.TRACE_FIELD_MODEL_OUTCOME,
		umpirespb.TRACE_FIELD_RESULTING_STATE:
		if coordinate.GetStep() > 0 && coordinate.GetPosition() == 0 {
			return nil
		}
	case umpirespb.TRACE_FIELD_OBSERVATION:
		if coordinate.GetStep() > 0 && coordinate.GetPosition() > 0 {
			return nil
		}
	default:
	}
	return admissionError(ErrorMalformedValue, path, "Model coordinate shape is invalid")
}

func coordinateKey(coordinate *umpirespb.ModelCoordinate) string {
	if coordinate == nil {
		return ""
	}
	return fmt.Sprintf("%010d\x00%020d\x00%020d", coordinate.GetField(), coordinate.GetStep(), coordinate.GetPosition())
}

func bindingKey(binding *umpirespb.DefinitionBinding) string {
	if binding == nil {
		return ""
	}
	return binding.GetDefinitionId() + "\x00" + binding.GetBehaviorFingerprint()
}

func modelValueKey(value *umpirespb.ModelValue) string {
	if value == nil {
		return ""
	}
	return bindingKey(value.GetDefinition()) + "\x00" + strconv.FormatInt(int64(value.GetKind()), 10) + "\x00" + valueKey(value.GetValue())
}

func valueKey(value *umpirespb.Value) string {
	if value == nil {
		return ""
	}
	switch scalar := value.GetValue().(type) {
	case *umpirespb.Value_Text:
		return "text\x00" + scalar.Text
	case *umpirespb.Value_Natural:
		return "natural\x00" + scalar.Natural
	case *umpirespb.Value_BoolValue:
		return "boolean\x00" + strconv.FormatBool(scalar.BoolValue)
	default:
		return ""
	}
}

func renameEntryKey(entry *umpirespb.RenameExactEntry) string {
	if entry == nil {
		return ""
	}
	return modelValueKey(entry.GetSource()) + "\x00" + modelValueKey(entry.GetDestination())
}

func definitionRenameEntryKey(entry *umpirespb.DefinitionRenameEntry) string {
	if entry == nil {
		return ""
	}
	return bindingKey(entry.GetSource()) + "\x00" + strconv.FormatInt(int64(entry.GetKind()), 10) +
		"\x00" + bindingKey(entry.GetDestination())
}

func validDefinitionID(value string) bool {
	segments := strings.Split(value, ".")
	if len(segments) < 2 {
		return false
	}
	for _, segment := range segments {
		if segment == "" {
			return false
		}
		for _, character := range []byte(segment) {
			if !isASCIIAlphanumeric(character) && character != '-' && character != '_' {
				return false
			}
		}
	}
	return true
}

func isASCIIAlphanumeric(character byte) bool {
	return character >= 'a' && character <= 'z' ||
		character >= 'A' && character <= 'Z' ||
		character >= '0' && character <= '9'
}

func validNatural(value string) bool {
	if value == "" || len(value) > 1 && value[0] == '0' {
		return false
	}
	_, err := strconv.ParseUint(value, 10, 64)
	return err == nil
}

func compareNatural(left, right string) int {
	if len(left) != len(right) {
		return len(left) - len(right)
	}
	return strings.Compare(left, right)
}
