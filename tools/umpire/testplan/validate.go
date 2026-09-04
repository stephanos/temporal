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
	if err := validator.validateKnownGaps(); err != nil {
		return err
	}
	if err := validator.validateExecution(); err != nil {
		return err
	}
	if err := validator.validateVerification(); err != nil {
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
		mandatoryBytes := int64(mandatoryResultBytes(plan))
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
		func() error { return v.validateArtifactProjection(execution.GetArtifactProjection()) },
	}
	for _, check := range checks {
		if err := check(); err != nil {
			return err
		}
	}
	return nil
}

func (v *validator) validateArtifactProjection(projection *umpirespb.PlanArtifactProjection) error {
	if projection == nil || projection.GetExpandedLimits() == nil || projection.GetExplored() == nil ||
		projection.GetExperimentProvenance() == nil || projection.GetRuntimeProvenance() == nil {
		return admissionError(ErrorMalformedValue, "$.execution.artifactProjection", "complete artifact projection is required")
	}
	if projection.GetSelectionReason() == umpirespb.PLAN_SELECTION_REASON_UNSPECIFIED {
		return admissionError(ErrorMalformedValue, "$.execution.artifactProjection.selectionReason", "selection reason is required")
	}
	if err := validateArtifactProjectionBounds(projection, v.plan.GetExecution()); err != nil {
		return err
	}
	if err := validateKnownGapList(projection.GetExperimentKnownGaps(), "$.execution.artifactProjection.experimentKnownGaps"); err != nil {
		return err
	}
	if err := validateKnownGapList(projection.GetRuntimeKnownGaps(), "$.execution.artifactProjection.runtimeKnownGaps"); err != nil {
		return err
	}
	if err := validateArtifactProvenance(projection.GetExperimentProvenance(), "$.execution.artifactProjection.experimentProvenance"); err != nil {
		return err
	}
	if err := validateArtifactProvenance(projection.GetRuntimeProvenance(), "$.execution.artifactProjection.runtimeProvenance"); err != nil {
		return err
	}
	if err := validateDefinitionIDs(
		projection.GetExperimentObservationRequirementDefinitionIds(),
		"$.execution.artifactProjection.experimentObservationRequirementDefinitionIds",
	); err != nil {
		return err
	}
	if err := v.validateArtifactRuntimeObservation(projection.GetRuntimeObservationConfig()); err != nil {
		return err
	}
	return validateArtifactKnownGapProjection(projection, v.plan.GetKnownGaps())
}

func validateArtifactProjectionBounds(
	projection *umpirespb.PlanArtifactProjection,
	execution *umpirespb.ExecutionProgram,
) error {
	limits := projection.GetExpandedLimits()
	if limits.GetMaxSemanticTransitions() != int64(len(execution.GetOccurrences())) ||
		limits.GetMaxSelectedActions() != int64(len(execution.GetRequestedActions())) ||
		limits.GetMaxCandidateEvaluations() <= 0 ||
		limits.GetMaxCandidateEvaluations() > MaximumEvaluationWork {
		return admissionError(ErrorLimit, "$.execution.artifactProjection.expandedLimits", "artifact search limits are inconsistent with the portable plan")
	}
	explored := projection.GetExplored()
	if explored.GetSetups() <= 0 || explored.GetTraces() <= 0 || explored.GetTransitions() <= 0 ||
		explored.GetPropertyEvaluations() <= 0 ||
		explored.GetTransitions() > limits.GetMaxSemanticTransitions() ||
		explored.GetPropertyEvaluations() > limits.GetMaxCandidateEvaluations() {
		return admissionError(ErrorLimit, "$.execution.artifactProjection.explored", "artifact explored counts are outside the expanded limits")
	}
	return nil
}

func (v *validator) validateArtifactRuntimeObservation(runtimeObservation *umpirespb.PortableObservationConfig) error {
	if runtimeObservation == nil {
		return admissionError(ErrorMalformedValue, "$.execution.artifactProjection.runtimeObservationConfig", "runtime artifact Observation configuration is required")
	}
	for _, binding := range []struct {
		path  string
		value *umpirespb.DefinitionBinding
	}{
		{"profile", runtimeObservation.GetProfile()},
		{"program", runtimeObservation.GetProgram()},
		{"mapping", runtimeObservation.GetMapping()},
	} {
		if err := v.validateBinding(binding.value, "$.execution.artifactProjection.runtimeObservationConfig."+binding.path); err != nil {
			return err
		}
	}
	return nil
}

func validateArtifactKnownGapProjection(
	projection *umpirespb.PlanArtifactProjection,
	want []*umpirespb.KnownGap,
) error {
	projectedGaps := append([]*umpirespb.KnownGap{}, projection.GetExperimentKnownGaps()...)
	projectedGaps = append(projectedGaps, projection.GetRuntimeKnownGaps()...)
	slices.SortFunc(projectedGaps, func(left, right *umpirespb.KnownGap) int {
		return strings.Compare(knownGapKey(left), knownGapKey(right))
	})
	projectedGaps = slices.CompactFunc(projectedGaps, func(left, right *umpirespb.KnownGap) bool {
		return knownGapKey(left) == knownGapKey(right)
	})
	if !equalKnownGaps(projectedGaps, want) {
		return admissionError(ErrorBinding, "$.execution.artifactProjection", "artifact Known Gaps do not project to the portable plan")
	}
	return nil
}

func validateArtifactProvenance(provenance *umpirespb.PlanArtifactProvenance, path string) error {
	if err := validateDefinitionIDs(provenance.GetSourceDefinitionIds(), path+".sourceDefinitionIds"); err != nil {
		return err
	}
	return validateLocations(provenance.GetSourceLocations(), path+".sourceLocations")
}

func validateDefinitionIDs(ids []string, path string) error {
	if len(ids) == 0 || !slices.IsSorted(ids) {
		return admissionError(ErrorOrdering, path, "definition identities must be non-empty and canonically ordered")
	}
	for index, id := range ids {
		if !validDefinitionID(id) || len(id) > artifact.MaximumIdentityBytes {
			return admissionError(ErrorMalformedValue, fmt.Sprintf("%s[%d]", path, index), "definition identity is malformed")
		}
		if index > 0 && ids[index-1] == id {
			return admissionError(ErrorDuplicate, fmt.Sprintf("%s[%d]", path, index), "duplicate definition identity")
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
	roleIDs, err := v.validateRoleBindings(execution.GetRoleBindings())
	if err != nil {
		return err
	}
	if err := v.validateSymbolicRoles(execution.GetSymbolicRoles(), roleIDs); err != nil {
		return err
	}
	if err := v.validateBindingSlots(execution.GetRuntimeBindingSlots()); err != nil {
		return err
	}
	if err := requireSortedUnique(execution.GetPreconditions(), func(precondition *umpirespb.ExecutionPrecondition) string {
		return bindingKey(precondition.GetDefinition())
	}, "$.execution.preconditions"); err != nil {
		return err
	}
	for index, precondition := range execution.GetPreconditions() {
		if err := v.validatePrecondition(precondition, fmt.Sprintf("$.execution.preconditions[%d]", index)); err != nil {
			return err
		}
	}
	return nil
}

func (v *validator) validateRoleBindings(bindings []*umpirespb.RoleBinding) (map[string]struct{}, error) {
	seen := make(map[string]struct{}, len(bindings))
	for index, role := range bindings {
		path := fmt.Sprintf("$.execution.roleBindings[%d]", index)
		if role == nil {
			return nil, admissionError(ErrorMalformedValue, path, "role binding is required")
		}
		if err := v.validateBinding(role.GetRole(), path+".role"); err != nil {
			return nil, err
		}
		if err := v.validatePortableModelValue(role.GetValue(), umpirespb.PORTABLE_DEFINITION_KIND_UNSPECIFIED, path+".value"); err != nil {
			return nil, err
		}
		id := role.GetRole().GetDefinitionId()
		if _, duplicate := seen[id]; duplicate {
			return nil, admissionError(ErrorDuplicate, path+".role", "duplicate bound role")
		}
		seen[id] = struct{}{}
	}
	return seen, nil
}

func (v *validator) validateSymbolicRoles(roles []*umpirespb.SymbolicRole, bound map[string]struct{}) error {
	seen := make(map[string]struct{}, len(roles))
	for index, role := range roles {
		path := fmt.Sprintf("$.execution.symbolicRoles[%d]", index)
		if role == nil || role.GetKind() == umpirespb.PORTABLE_DEFINITION_KIND_UNSPECIFIED {
			return admissionError(ErrorMalformedValue, path, "symbolic role is malformed")
		}
		if err := v.validateBinding(role.GetDefinition(), path+".definition"); err != nil {
			return err
		}
		id := role.GetDefinition().GetDefinitionId()
		if _, duplicate := seen[id]; duplicate {
			return admissionError(ErrorDuplicate, path, "duplicate symbolic role")
		}
		if _, duplicate := bound[id]; duplicate {
			return admissionError(ErrorDuplicate, path, "role is both bound and symbolic")
		}
		seen[id] = struct{}{}
	}
	return nil
}

func (v *validator) validateExecutionTrace(execution *umpirespb.ExecutionProgram) error {
	if err := v.validatePortableModelValue(execution.GetInitialState(), umpirespb.PORTABLE_DEFINITION_KIND_STATE, "$.execution.initialState"); err != nil {
		return err
	}
	if len(execution.GetRequestedActions()) != 1 || int64(len(execution.GetRequestedActions())) > v.limits.GetExecution().GetMaxActions() {
		return admissionError(ErrorLimit, "$.execution.requestedActions", "version one requires exactly one action")
	}
	if len(execution.GetModelOutcomes()) != 1 || len(execution.GetResultingStates()) != 1 || len(execution.GetOccurrences()) != 1 {
		return admissionError(ErrorLimit, "$.execution", "version one requires one outcome, resulting state, and occurrence")
	}
	if int64(len(execution.GetRequestedFaults())) > v.limits.GetExecution().GetMaxFaults() {
		return admissionError(ErrorLimit, "$.execution.requestedFaults", "fault count exceeds the declared limit")
	}
	collections := []struct {
		name         string
		values       []*umpirespb.PortableModelValue
		expectedKind umpirespb.PortableDefinitionKind
	}{
		{"requestedActions", execution.GetRequestedActions(), umpirespb.PORTABLE_DEFINITION_KIND_ACTION},
		{"modelOutcomes", execution.GetModelOutcomes(), umpirespb.PORTABLE_DEFINITION_KIND_OUTCOME},
		{"resultingStates", execution.GetResultingStates(), umpirespb.PORTABLE_DEFINITION_KIND_STATE},
		{"selectedChoices", execution.GetSelectedChoices(), umpirespb.PORTABLE_DEFINITION_KIND_UNSPECIFIED},
		{"selectedVariants", execution.GetSelectedVariants(), umpirespb.PORTABLE_DEFINITION_KIND_UNSPECIFIED},
		{"requestedFaults", execution.GetRequestedFaults(), umpirespb.PORTABLE_DEFINITION_KIND_UNSPECIFIED},
	}
	for _, collection := range collections {
		if err := v.validatePortableModelValues(collection.values, collection.expectedKind, "$.execution."+collection.name); err != nil {
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
	if len(checkpoints) != len(v.plan.GetExecution().GetOccurrences()) {
		return admissionError(ErrorLimit, "$.execution.checkpoints", "checkpoint count must equal the trace transition count")
	}
	for index, checkpoint := range checkpoints {
		path := fmt.Sprintf("$.execution.checkpoints[%d]", index)
		if checkpoint == nil || checkpoint.GetTransition() != int64(index+1) {
			return admissionError(ErrorOrdering, path, "checkpoint order is invalid")
		}
		if err := v.validatePortableModelValues(
			checkpoint.GetObservations(), umpirespb.PORTABLE_DEFINITION_KIND_OBSERVATION, path+".observations",
		); err != nil {
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
	if precondition == nil ||
		(precondition.GetOperator() != umpirespb.PRECONDITION_OPERATOR_EQUALS &&
			precondition.GetOperator() != umpirespb.PRECONDITION_OPERATOR_NOT_EQUALS) {
		return admissionError(ErrorUnsupportedOperator, path, "version one supports equals and not-equals")
	}
	v.operators++
	if err := v.validateBinding(precondition.GetDefinition(), path+".definition"); err != nil {
		return err
	}
	left, err := v.validateExecutionOperand(precondition.GetLeft(), path+".left")
	if err != nil {
		return err
	}
	right, err := v.validateExecutionOperand(precondition.GetRight(), path+".right")
	if err != nil {
		return err
	}
	if !compatibleOperandTypes(left, right) {
		return admissionError(ErrorBinding, path, "precondition operands have crossed types")
	}
	return nil
}

type operandType struct {
	definition umpirespb.PortableDefinitionKind
	scalar     umpirespb.PortableValueKind
}

func (v *validator) validateExecutionOperand(operand *umpirespb.ExecutionOperand, path string) (operandType, error) {
	if operand == nil {
		return operandType{}, admissionError(ErrorUnsupportedOperator, path, "operand is required")
	}
	switch value := operand.GetOperand().(type) {
	case *umpirespb.ExecutionOperand_Literal:
		if err := v.validatePortableModelValue(value.Literal, umpirespb.PORTABLE_DEFINITION_KIND_UNSPECIFIED, path+".literal"); err != nil {
			return operandType{}, err
		}
		return operandType{definition: value.Literal.GetKind(), scalar: portableScalarKind(value.Literal.GetValue())}, nil
	case *umpirespb.ExecutionOperand_Role:
		if err := v.validateBinding(value.Role, path+".role"); err != nil {
			return operandType{}, err
		}
		for _, role := range v.plan.GetExecution().GetRoleBindings() {
			if proto.Equal(value.Role, role.GetRole()) {
				return operandType{
					definition: role.GetValue().GetKind(),
					scalar:     portableScalarKind(role.GetValue().GetValue()),
				}, nil
			}
		}
		for _, role := range v.plan.GetExecution().GetSymbolicRoles() {
			if proto.Equal(value.Role, role.GetDefinition()) {
				return operandType{definition: role.GetKind()}, nil
			}
		}
		return operandType{}, admissionError(ErrorBinding, path+".role", "operand is crossed with the declared roles")
	case *umpirespb.ExecutionOperand_RuntimeBindingSlot:
		if err := v.validateBinding(value.RuntimeBindingSlot, path+".runtimeBindingSlot"); err != nil {
			return operandType{}, err
		}
		for _, slot := range v.plan.GetExecution().GetRuntimeBindingSlots() {
			if proto.Equal(value.RuntimeBindingSlot, slot.GetDefinition()) {
				return operandType{scalar: slot.GetValueKind()}, nil
			}
		}
		return operandType{}, admissionError(ErrorBinding, path+".runtimeBindingSlot", "operand is crossed with the declared runtime binding slots")
	default:
		return operandType{}, admissionError(ErrorUnsupportedOperator, path, "operand kind is required")
	}
}

func compatibleOperandTypes(left, right operandType) bool {
	compared := false
	if left.definition != umpirespb.PORTABLE_DEFINITION_KIND_UNSPECIFIED &&
		right.definition != umpirespb.PORTABLE_DEFINITION_KIND_UNSPECIFIED {
		compared = true
		if left.definition != right.definition {
			return false
		}
	}
	if left.scalar != umpirespb.PORTABLE_VALUE_KIND_UNSPECIFIED && right.scalar != umpirespb.PORTABLE_VALUE_KIND_UNSPECIFIED {
		compared = true
		if left.scalar != right.scalar {
			return false
		}
	}
	return compared
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
	return v.validateBindingCollection(
		runtime.GetAuthorityRequiredCapabilities(),
		"$.execution.runtime.authorityRequiredCapabilities",
		false,
	)
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
	return v.validateBindingCollection(participant.GetCapabilities(), path+".capabilities", true)
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
	seen, err := v.validateEmits(observation.GetEmits(), path+".emits")
	if err != nil {
		return err
	}
	return validateEmitOrdering(observation.GetOrdering(), seen, path+".ordering")
}

func (v *validator) validateEmits(emits []*umpirespb.Emit, path string) (map[string]struct{}, error) {
	seen := make(map[string]struct{}, len(emits))
	coordinates := make(map[string]struct{}, len(emits))
	for index, emit := range emits {
		emitPath := fmt.Sprintf("%s[%d]", path, index)
		if err := v.validateEmit(emit, emitPath, coordinates); err != nil {
			return nil, err
		}
		seen[emit.GetDefinitionId()] = struct{}{}
	}
	return seen, nil
}

func (v *validator) validateEmit(emit *umpirespb.Emit, path string, coordinates map[string]struct{}) error {
	if emit == nil || !validDefinitionID(emit.GetDefinitionId()) || !validDefinitionID(emit.GetSourceKindDefinitionId()) ||
		emit.GetOutputKind() == umpirespb.DEFINITION_KIND_UNSPECIFIED || emit.GetCoordinate() == nil {
		return admissionError(ErrorMalformedValue, path, "Emit is malformed")
	}
	if !v.isDeclaredEvidenceKind(emit.GetSourceKindDefinitionId()) {
		return admissionError(ErrorBinding, path+".sourceKindDefinitionId", "Emit source kind is not declared")
	}
	if err := validateCoordinate(emit.GetCoordinate(), path+".coordinate"); err != nil {
		return err
	}
	coordinate := coordinateKey(emit.GetCoordinate())
	if _, duplicate := coordinates[coordinate]; duplicate {
		return admissionError(ErrorDuplicate, path+".coordinate", "duplicate emitted coordinate")
	}
	coordinates[coordinate] = struct{}{}
	if err := v.validateBinding(emit.GetOutputDefinition(), path+".outputDefinition"); err != nil {
		return err
	}
	conditionKind, err := v.validateObservationExpression(emit.GetCondition(), path+".condition", 1)
	if err != nil {
		return err
	}
	if conditionKind != umpirespb.PORTABLE_VALUE_KIND_BOOLEAN {
		return admissionError(ErrorBinding, path+".condition", "Emit condition is not Boolean")
	}
	valueKind, err := v.validateObservationExpression(emit.GetValue(), path+".value", 1)
	if err != nil {
		return err
	}
	return v.validateEmitType(emit, valueKind, path)
}

func (v *validator) validateObservationExpression(
	expression *umpirespb.ObservationExpression,
	path string,
	depth int64,
) (umpirespb.PortableValueKind, error) {
	if expression == nil || depth > v.limits.GetEvaluation().GetMaxExpressionDepth() {
		return umpirespb.PORTABLE_VALUE_KIND_UNSPECIFIED,
			admissionError(ErrorLimit, path, "Observation expression exceeds its depth Limit")
	}
	v.operators++
	switch operator := expression.GetOperator().(type) {
	case *umpirespb.ObservationExpression_LiteralText:
		if operator.LiteralText == nil {
			return umpirespb.PORTABLE_VALUE_KIND_UNSPECIFIED,
				admissionError(ErrorMalformedValue, path, "literal text is required")
		}
		return umpirespb.PORTABLE_VALUE_KIND_TEXT, nil
	case *umpirespb.ObservationExpression_LiteralNatural:
		if operator.LiteralNatural == nil || !v.validBoundedNatural(operator.LiteralNatural.GetValue()) {
			return umpirespb.PORTABLE_VALUE_KIND_UNSPECIFIED,
				admissionError(ErrorMalformedValue, path, "literal natural is malformed")
		}
		return umpirespb.PORTABLE_VALUE_KIND_NATURAL, nil
	case *umpirespb.ObservationExpression_Field:
		if operator.Field == nil || !validDefinitionID(operator.Field.GetKindDefinitionId()) || !validDefinitionID(operator.Field.GetFieldDefinitionId()) {
			return umpirespb.PORTABLE_VALUE_KIND_UNSPECIFIED,
				admissionError(ErrorBinding, path, "Evidence field reference is malformed")
		}
		kind := v.declaredEvidenceFieldKind(operator.Field)
		if kind == umpirespb.PORTABLE_VALUE_KIND_UNSPECIFIED {
			return kind, admissionError(ErrorBinding, path, "Evidence field reference is crossed with the declared profile")
		}
		return kind, nil
	case *umpirespb.ObservationExpression_NaturalRenderV1:
		kind, err := v.validateObservationExpression(operator.NaturalRenderV1.GetOperand(), path+".naturalRenderV1", depth+1)
		if err != nil {
			return umpirespb.PORTABLE_VALUE_KIND_UNSPECIFIED, err
		}
		if kind != umpirespb.PORTABLE_VALUE_KIND_NATURAL {
			return umpirespb.PORTABLE_VALUE_KIND_UNSPECIFIED,
				admissionError(ErrorBinding, path, "natural-render operand is not Natural")
		}
		return umpirespb.PORTABLE_VALUE_KIND_TEXT, nil
	case *umpirespb.ObservationExpression_Present:
		if _, err := v.validateObservationExpression(operator.Present.GetOperand(), path+".present", depth+1); err != nil {
			return umpirespb.PORTABLE_VALUE_KIND_UNSPECIFIED, err
		}
		return umpirespb.PORTABLE_VALUE_KIND_BOOLEAN, nil
	case *umpirespb.ObservationExpression_Equals:
		left, err := v.validateObservationExpression(operator.Equals.GetLeft(), path+".equals.left", depth+1)
		if err != nil {
			return umpirespb.PORTABLE_VALUE_KIND_UNSPECIFIED, err
		}
		right, err := v.validateObservationExpression(operator.Equals.GetRight(), path+".equals.right", depth+1)
		if err != nil {
			return umpirespb.PORTABLE_VALUE_KIND_UNSPECIFIED, err
		}
		if left != right {
			return umpirespb.PORTABLE_VALUE_KIND_UNSPECIFIED,
				admissionError(ErrorBinding, path, "equals operands have crossed scalar kinds")
		}
		return umpirespb.PORTABLE_VALUE_KIND_BOOLEAN, nil
	case *umpirespb.ObservationExpression_All:
		return v.validateBooleanExpressionList(operator.All.GetOperands(), path+".all", depth+1)
	case *umpirespb.ObservationExpression_Any:
		return v.validateBooleanExpressionList(operator.Any.GetOperands(), path+".any", depth+1)
	default:
		return umpirespb.PORTABLE_VALUE_KIND_UNSPECIFIED,
			admissionError(ErrorUnsupportedOperator, path, "Observation operator is required")
	}
}

func (v *validator) declaredEvidenceFieldKind(reference *umpirespb.EvidenceFieldReference) umpirespb.PortableValueKind {
	profile := v.plan.GetVerification().GetEvidence()
	for _, kind := range profile.GetKinds() {
		if kind.GetKindDefinitionId() != reference.GetKindDefinitionId() {
			continue
		}
		for _, field := range kind.GetFields() {
			if field.GetFieldDefinitionId() == reference.GetFieldDefinitionId() {
				if field.GetDisposition() != umpirespb.FIELD_DISPOSITION_KIND_RETAIN {
					return umpirespb.PORTABLE_VALUE_KIND_UNSPECIFIED
				}
				return portableEvidenceValueKind(field.GetValueKind())
			}
		}
	}
	return umpirespb.PORTABLE_VALUE_KIND_UNSPECIFIED
}

func (v *validator) isDeclaredEvidenceKind(kindID string) bool {
	for _, kind := range v.plan.GetVerification().GetEvidence().GetKinds() {
		if kind.GetKindDefinitionId() == kindID {
			return true
		}
	}
	return false
}

func (v *validator) validateEmitType(emit *umpirespb.Emit, scalar umpirespb.PortableValueKind, path string) error {
	wantDefinition := traceDefinitionKind(emit.GetCoordinate().GetField())
	if wantDefinition == umpirespb.DEFINITION_KIND_UNSPECIFIED || emit.GetOutputKind() != wantDefinition {
		return admissionError(ErrorBinding, path+".outputKind", "Emit output kind is crossed with its coordinate")
	}
	wantScalar := v.coordinateValueKind(emit.GetCoordinate(), emit.GetOutputDefinition())
	if v.plan.GetVerification().GetRenameExactLink() != nil {
		wantScalar = v.renameValueKind(emit.GetCoordinate().GetField(), emit.GetOutputDefinition(), false)
	}
	if wantScalar == umpirespb.PORTABLE_VALUE_KIND_UNSPECIFIED {
		return admissionError(ErrorBinding, path+".outputDefinition", "Emit output is crossed with the projected trace")
	}
	if scalar != wantScalar {
		return admissionError(ErrorBinding, path+".value", "Emit value has a crossed scalar kind")
	}
	return nil
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

func (v *validator) validateBooleanExpressionList(
	expressions []*umpirespb.ObservationExpression,
	path string,
	depth int64,
) (umpirespb.PortableValueKind, error) {
	if len(expressions) == 0 {
		return umpirespb.PORTABLE_VALUE_KIND_UNSPECIFIED,
			admissionError(ErrorUnsupportedOperator, path, "operator operands are required")
	}
	for index, expression := range expressions {
		kind, err := v.validateObservationExpression(expression, fmt.Sprintf("%s[%d]", path, index), depth)
		if err != nil {
			return umpirespb.PORTABLE_VALUE_KIND_UNSPECIFIED, err
		}
		if kind != umpirespb.PORTABLE_VALUE_KIND_BOOLEAN {
			return umpirespb.PORTABLE_VALUE_KIND_UNSPECIFIED,
				admissionError(ErrorBinding, fmt.Sprintf("%s[%d]", path, index), "Boolean operand is required")
		}
	}
	return umpirespb.PORTABLE_VALUE_KIND_BOOLEAN, nil
}

func (v *validator) validatePattern(pattern *umpirespb.Pattern, path string) error {
	if pattern == nil || pattern.GetField() == umpirespb.TRACE_FIELD_UNSPECIFIED {
		return admissionError(ErrorMalformedValue, path, "Property pattern is malformed")
	}
	if err := v.validateBinding(pattern.GetDefinition(), path+".definition"); err != nil {
		return err
	}
	scalar := v.traceValueKind(pattern.GetField(), pattern.GetDefinition())
	if v.plan.GetVerification().GetRenameExactLink() != nil {
		scalar = v.renameValueKind(pattern.GetField(), pattern.GetDefinition(), true)
	}
	if scalar == umpirespb.PORTABLE_VALUE_KIND_UNSPECIFIED {
		return admissionError(ErrorBinding, path+".definition", "Property pattern is crossed with the projected execution trace")
	}
	v.operators++
	switch operator := pattern.GetOperator().(type) {
	case *umpirespb.Pattern_EqualsText:
		if operator.EqualsText == nil {
			return admissionError(ErrorUnsupportedOperator, path, "equals-text operand is required")
		}
		if scalar != umpirespb.PORTABLE_VALUE_KIND_TEXT {
			return admissionError(ErrorBinding, path, "equals-text requires a Text trace value")
		}
	case *umpirespb.Pattern_NaturalAtMost:
		if operator.NaturalAtMost == nil || !v.validBoundedNatural(operator.NaturalAtMost.GetBound()) {
			return admissionError(ErrorMalformedValue, path, "natural bound is malformed")
		}
		if scalar != umpirespb.PORTABLE_VALUE_KIND_NATURAL {
			return admissionError(ErrorBinding, path, "natural-at-most requires a Natural trace value")
		}
	default:
		return admissionError(ErrorUnsupportedOperator, path, "Property pattern operator is required")
	}
	return nil
}

func (v *validator) traceValueKind(
	field umpirespb.TraceField,
	binding *umpirespb.DefinitionBinding,
) umpirespb.PortableValueKind {
	execution := v.plan.GetExecution()
	var values []*umpirespb.PortableModelValue
	switch field {
	case umpirespb.TRACE_FIELD_INITIAL_STATE, umpirespb.TRACE_FIELD_PRIOR_STATE:
		values = []*umpirespb.PortableModelValue{execution.GetInitialState()}
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
		return umpirespb.PORTABLE_VALUE_KIND_UNSPECIFIED
	}
	for _, value := range values {
		if proto.Equal(value.GetDefinition(), binding) {
			return portableScalarKind(value.GetValue())
		}
	}
	return umpirespb.PORTABLE_VALUE_KIND_UNSPECIFIED
}

func (v *validator) coordinateValueKind(
	coordinate *umpirespb.ModelCoordinate,
	binding *umpirespb.DefinitionBinding,
) umpirespb.PortableValueKind {
	execution := v.plan.GetExecution()
	step := int(coordinate.GetStep())
	position := int(coordinate.GetPosition())
	var value *umpirespb.PortableModelValue
	switch coordinate.GetField() {
	case umpirespb.TRACE_FIELD_INITIAL_STATE:
		value = execution.GetInitialState()
	case umpirespb.TRACE_FIELD_PRIOR_STATE:
		if step == 1 {
			value = execution.GetInitialState()
		} else if step > 1 && step-2 < len(execution.GetResultingStates()) {
			value = execution.GetResultingStates()[step-2]
		}
	case umpirespb.TRACE_FIELD_SELECTED_ACTION:
		if step > 0 && step <= len(execution.GetRequestedActions()) {
			value = execution.GetRequestedActions()[step-1]
		}
	case umpirespb.TRACE_FIELD_MODEL_OUTCOME:
		if step > 0 && step <= len(execution.GetModelOutcomes()) {
			value = execution.GetModelOutcomes()[step-1]
		}
	case umpirespb.TRACE_FIELD_RESULTING_STATE:
		if step > 0 && step <= len(execution.GetResultingStates()) {
			value = execution.GetResultingStates()[step-1]
		}
	case umpirespb.TRACE_FIELD_OBSERVATION:
		if step > 0 && step <= len(execution.GetCheckpoints()) {
			observations := execution.GetCheckpoints()[step-1].GetObservations()
			if position > 0 && position <= len(observations) {
				value = observations[position-1]
			}
		}
	default:
	}
	if value != nil && proto.Equal(value.GetDefinition(), binding) {
		return portableScalarKind(value.GetValue())
	}
	return umpirespb.PORTABLE_VALUE_KIND_UNSPECIFIED
}

func (v *validator) renameValueKind(
	field umpirespb.TraceField,
	binding *umpirespb.DefinitionBinding,
	destination bool,
) umpirespb.PortableValueKind {
	wantKind := traceDefinitionKind(field)
	for _, entry := range v.plan.GetVerification().GetRenameExactLink().GetEntries() {
		value := entry.GetSource()
		if destination {
			value = entry.GetDestination()
		}
		if value.GetKind() == wantKind && proto.Equal(value.GetDefinition(), binding) {
			return portableScalarKind(value.GetValue())
		}
	}
	return umpirespb.PORTABLE_VALUE_KIND_UNSPECIFIED
}

func traceDefinitionKind(field umpirespb.TraceField) umpirespb.DefinitionKind {
	switch field {
	case umpirespb.TRACE_FIELD_INITIAL_STATE,
		umpirespb.TRACE_FIELD_PRIOR_STATE,
		umpirespb.TRACE_FIELD_RESULTING_STATE:
		return umpirespb.DEFINITION_KIND_STATE
	case umpirespb.TRACE_FIELD_SELECTED_ACTION:
		return umpirespb.DEFINITION_KIND_ACTION
	case umpirespb.TRACE_FIELD_MODEL_OUTCOME:
		return umpirespb.DEFINITION_KIND_OUTCOME
	case umpirespb.TRACE_FIELD_OBSERVATION:
		return umpirespb.DEFINITION_KIND_OBSERVATION
	default:
		return umpirespb.DEFINITION_KIND_UNSPECIFIED
	}
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
	if !proto.Equal(link.GetDestinationTarget(), v.plan.GetExecution().GetTarget()) {
		return admissionError(ErrorBinding, path+".destinationTarget", "exact rename destination target is crossed with execution target")
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
		if err := v.validateLegacyModelValue(entry.GetSource(), entryPath+".source"); err != nil {
			return err
		}
		source := modelValueKey(entry.GetSource())
		if _, duplicate := sources[source]; duplicate {
			return admissionError(ErrorDuplicate, entryPath+".source", "rename source has duplicate or contradictory mappings")
		}
		sources[source] = struct{}{}
		if err := v.validateLegacyModelValue(entry.GetDestination(), entryPath+".destination"); err != nil {
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
	return validateKnownGapList(v.plan.GetKnownGaps(), "$.knownGaps")
}

func validateKnownGapList(gaps []*umpirespb.KnownGap, path string) error {
	if err := requireSortedUnique(gaps, knownGapKey, path); err != nil {
		return err
	}
	for index, gap := range gaps {
		gapPath := fmt.Sprintf("%s[%d]", path, index)
		if gap == nil || gap.GetKind() == umpirespb.KNOWN_GAP_KIND_UNSPECIFIED || !validDefinitionID(gap.GetCode()) ||
			len(gap.GetDetail()) > artifact.MaximumDiagnosticBytes {
			return admissionError(ErrorMalformedValue, gapPath, "Known Gap is malformed")
		}
		if gap.GetSubject() != "" && !validDefinitionID(gap.GetSubject()) {
			return admissionError(ErrorMalformedValue, gapPath+".subject", "Known Gap subject is malformed")
		}
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

func (v *validator) validatePortableModelValues(
	values []*umpirespb.PortableModelValue,
	expectedKind umpirespb.PortableDefinitionKind,
	path string,
) error {
	seen := make(map[string]struct{}, len(values))
	for index, value := range values {
		valuePath := fmt.Sprintf("%s[%d]", path, index)
		if err := v.validatePortableModelValue(value, expectedKind, valuePath); err != nil {
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

func (v *validator) validatePortableModelValue(
	value *umpirespb.PortableModelValue,
	expectedKind umpirespb.PortableDefinitionKind,
	path string,
) error {
	if value == nil || value.GetKind() == umpirespb.PORTABLE_DEFINITION_KIND_UNSPECIFIED || value.GetValue() == nil {
		return admissionError(ErrorMalformedValue, path, "portable model value is malformed")
	}
	if err := v.validateBinding(value.GetDefinition(), path+".definition"); err != nil {
		return err
	}
	if expectedKind != umpirespb.PORTABLE_DEFINITION_KIND_UNSPECIFIED && value.GetKind() != expectedKind {
		return admissionError(ErrorBinding, path+".kind", "portable model value has a crossed definition kind")
	}
	return v.validateScalarValue(value.GetValue(), path+".value")
}

func (v *validator) validateLegacyModelValue(value *umpirespb.ModelValue, path string) error {
	if value == nil || value.GetKind() == umpirespb.DEFINITION_KIND_UNSPECIFIED || value.GetValue() == nil {
		return admissionError(ErrorMalformedValue, path, "model value is malformed")
	}
	if err := v.validateBinding(value.GetDefinition(), path+".definition"); err != nil {
		return err
	}
	return v.validateScalarValue(value.GetValue(), path+".value")
}

func (v *validator) validateScalarValue(value *umpirespb.Value, path string) error {
	switch scalar := value.GetValue().(type) {
	case *umpirespb.Value_Text:
		if len(scalar.Text) > artifact.MaximumDiagnosticBytes {
			return admissionError(ErrorByteLimit, path+".text", "text value exceeds its byte maximum")
		}
	case *umpirespb.Value_Natural:
		if !v.validBoundedNatural(scalar.Natural) {
			return admissionError(ErrorMalformedValue, path+".natural", "natural is malformed or out of range")
		}
	case *umpirespb.Value_BoolValue:
	default:
		return admissionError(ErrorMalformedValue, path+".value", "scalar value kind is required")
	}
	return nil
}

func portableScalarKind(value *umpirespb.Value) umpirespb.PortableValueKind {
	if value == nil {
		return umpirespb.PORTABLE_VALUE_KIND_UNSPECIFIED
	}
	switch value.GetValue().(type) {
	case *umpirespb.Value_Text:
		return umpirespb.PORTABLE_VALUE_KIND_TEXT
	case *umpirespb.Value_Natural:
		return umpirespb.PORTABLE_VALUE_KIND_NATURAL
	case *umpirespb.Value_BoolValue:
		return umpirespb.PORTABLE_VALUE_KIND_BOOLEAN
	default:
		return umpirespb.PORTABLE_VALUE_KIND_UNSPECIFIED
	}
}

func portableEvidenceValueKind(kind umpirespb.ValueKind) umpirespb.PortableValueKind {
	switch kind {
	case umpirespb.VALUE_KIND_TEXT:
		return umpirespb.PORTABLE_VALUE_KIND_TEXT
	case umpirespb.VALUE_KIND_NATURAL:
		return umpirespb.PORTABLE_VALUE_KIND_NATURAL
	case umpirespb.VALUE_KIND_BOOLEAN:
		return umpirespb.PORTABLE_VALUE_KIND_BOOLEAN
	default:
		return umpirespb.PORTABLE_VALUE_KIND_UNSPECIFIED
	}
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

func knownGapKey(gap *umpirespb.KnownGap) string {
	if gap == nil {
		return ""
	}
	return fmt.Sprintf("%010d\x00%s\x00%s\x00%s", gap.GetKind(), gap.GetCode(), gap.GetSubject(), gap.GetDetail())
}

func equalKnownGaps(left, right []*umpirespb.KnownGap) bool {
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
