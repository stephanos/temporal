package evaluationcontract

import (
	"bytes"
	"fmt"
	"slices"
	"strconv"
	"strings"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
)

type contractValidator struct {
	contract     *umpirespb.EvaluationContract
	limits       *umpirespb.EvaluationLimits
	bindings     map[string]string
	sources      map[string]struct{}
	kinds        map[string]map[string]umpirespb.ValueKind
	digestPolicy map[string]struct{}
}

func validateContract(contract *umpirespb.EvaluationContract, checksumOptional bool) error {
	if contract == nil {
		return admissionError(ErrorMalformedValue, "$", "contract is required")
	}
	validator := &contractValidator{
		contract:     contract,
		limits:       contract.GetLimits(),
		bindings:     make(map[string]string),
		sources:      make(map[string]struct{}),
		kinds:        make(map[string]map[string]umpirespb.ValueKind),
		digestPolicy: make(map[string]struct{}),
	}
	if err := validator.validateVersion(); err != nil {
		return err
	}
	if !validDefinitionID(contract.GetContractId()) {
		return admissionError(ErrorMalformedValue, "$.contractId", "definition ID %q is invalid", contract.GetContractId())
	}
	if checksumLength := len(contract.GetArtifactChecksum()); checksumLength != 32 && (!checksumOptional || checksumLength != 0) {
		return admissionError(ErrorChecksum, "$.artifactChecksum", "checksum has %d bytes; want 32", checksumLength)
	}
	if err := validator.validateArtifactBinding(contract.GetExperiment(), "$.experiment", artifactv2.ExperimentFormat); err != nil {
		return err
	}
	if err := validator.validateArtifactBinding(contract.GetRuntimeConfig(), "$.runtimeConfig", artifactv2.RuntimeConfigurationFormat); err != nil {
		return err
	}
	if err := validator.validateBinding(contract.GetTest(), "$.test"); err != nil {
		return err
	}
	if err := validator.validateBinding(contract.GetQuery(), "$.query"); err != nil {
		return err
	}
	if contract.GetQuery().GetBehaviorFingerprint() != contract.GetExperiment().GetBehaviorFingerprint() {
		return admissionError(ErrorBinding, "$.query", "query fingerprint is crossed with the Experiment binding")
	}
	if err := validator.validateLimits(); err != nil {
		return err
	}
	if err := validator.validateObservation(); err != nil {
		return err
	}
	if err := validator.validateLink(); err != nil {
		return err
	}
	if err := validator.validateProperties(); err != nil {
		return err
	}
	if err := validator.validateKnownGaps(); err != nil {
		return err
	}
	return validator.validateLocations(contract.GetProvenance(), "$.provenance", true)
}

func (v *contractValidator) validateVersion() error {
	version := v.contract.GetVersion()
	if version == nil {
		return admissionError(ErrorUnsupportedVersion, "$.version", "version is required")
	}
	if version.GetMajor() != SupportedMajorVersion || version.GetMinor() > SupportedMinorVersion || version.GetMinor() < 0 {
		return admissionError(ErrorUnsupportedVersion, "$.version", "got %d.%d; reader supports %d.%d",
			version.GetMajor(), version.GetMinor(), SupportedMajorVersion, SupportedMinorVersion)
	}
	return nil
}

func (v *contractValidator) validateLimits() error {
	if v.limits == nil {
		return admissionError(ErrorLimit, "$.limits", "limits are required")
	}
	for _, limit := range []struct {
		path    string
		value   int64
		maximum int64
	}{
		{path: "maxContractBytes", value: v.limits.GetMaxContractBytes(), maximum: MaximumContractBytes},
		{path: "maxInputBytes", value: v.limits.GetMaxInputBytes(), maximum: MaximumInputBytes},
		{path: "maxEvidenceRecords", value: v.limits.GetMaxEvidenceRecords(), maximum: MaximumEvidenceRecords},
		{path: "maxExpressionDepth", value: v.limits.GetMaxExpressionDepth(), maximum: MaximumExpressionDepth},
		{path: "maxCollectionItems", value: v.limits.GetMaxCollectionItems(), maximum: MaximumCollectionItems},
		{path: "maxEvaluationWork", value: v.limits.GetMaxEvaluationWork(), maximum: MaximumEvaluationWork},
		{path: "maxDiagnosticBytes", value: v.limits.GetMaxDiagnosticBytes(), maximum: MaximumDiagnosticBytes},
		{path: "maxResultBytes", value: v.limits.GetMaxResultBytes(), maximum: MaximumResultBytes},
		{path: "maxTotalDurationMilliseconds", value: v.limits.GetMaxTotalDurationMilliseconds(), maximum: MaximumDurationMillis},
	} {
		if limit.value <= 0 || limit.value > limit.maximum {
			return admissionError(ErrorLimit, "$.limits."+limit.path,
				"limit is %d; allowed range is 1..%d", limit.value, limit.maximum)
		}
	}
	if !validNatural(v.limits.GetMaxNatural()) {
		return admissionError(ErrorLimit, "$.limits.maxNatural", "natural bound %q is not canonical", v.limits.GetMaxNatural())
	}
	if v.limits.GetMaxDiagnosticBytes() > v.limits.GetMaxResultBytes() {
		return admissionError(ErrorLimit, "$.limits.maxDiagnosticBytes", "diagnostic limit exceeds result limit")
	}
	return nil
}

func (v *contractValidator) validateArtifactBinding(
	binding *umpirespb.ArtifactBinding,
	path string,
	format string,
) error {
	if binding == nil {
		return admissionError(ErrorBinding, path, "artifact binding is required")
	}
	if binding.GetFormatVersion() != format {
		return admissionError(ErrorBinding, path+".formatVersion", "got %q; want %q", binding.GetFormatVersion(), format)
	}
	for _, field := range []struct {
		name  string
		value string
	}{
		{name: "artifactChecksum", value: binding.GetArtifactChecksum()},
		{name: "behaviorFingerprint", value: binding.GetBehaviorFingerprint()},
		{name: "provenanceChecksum", value: binding.GetProvenanceChecksum()},
	} {
		if !artifactv2.ValidDigest(field.value) {
			return admissionError(ErrorBinding, path+"."+field.name, "digest %q is invalid", field.value)
		}
	}
	return nil
}

func (v *contractValidator) validateBinding(binding *umpirespb.DefinitionBinding, path string) error {
	if binding == nil || !validDefinitionID(binding.GetDefinitionId()) || !artifactv2.ValidDigest(binding.GetBehaviorFingerprint()) {
		return admissionError(ErrorBinding, path, "definition binding is malformed")
	}
	if fingerprint, ok := v.bindings[binding.GetDefinitionId()]; ok && fingerprint != binding.GetBehaviorFingerprint() {
		return admissionError(ErrorBinding, path, "definition ID %q is crossed with another fingerprint", binding.GetDefinitionId())
	}
	v.bindings[binding.GetDefinitionId()] = binding.GetBehaviorFingerprint()
	return nil
}

func (v *contractValidator) validateObservation() error {
	observation := v.contract.GetObservation()
	if observation == nil {
		return admissionError(ErrorMalformedValue, "$.observation", "Observation program is required")
	}
	if err := v.validateBinding(observation.GetDefinition(), "$.observation.definition"); err != nil {
		return err
	}
	if err := v.validateLocation(observation.GetSource(), "$.observation.source"); err != nil {
		return err
	}
	if err := v.validateBinding(observation.GetMapping(), "$.observation.mapping"); err != nil {
		return err
	}
	if observation.GetMappingVersion() != 1 {
		return admissionError(ErrorUnsupportedVersion, "$.observation.mappingVersion", "got %d; want 1", observation.GetMappingVersion())
	}
	if err := v.validateEvidenceProfile(observation.GetProfile()); err != nil {
		return err
	}
	emitIDs, err := v.validateEmits(observation.GetEmits())
	if err != nil {
		return err
	}
	return v.validateEmitOrdering(observation.GetOrdering(), emitIDs)
}

func (v *contractValidator) validateEmits(emits []*umpirespb.Emit) (map[string]struct{}, error) {
	if err := v.collection("$.observation.emits", len(emits), true); err != nil {
		return nil, err
	}
	if err := requireSortedUnique(emits, func(emit *umpirespb.Emit) string {
		return emit.GetDefinitionId()
	}, "$.observation.emits"); err != nil {
		return nil, err
	}
	emitIDs := make(map[string]struct{}, len(emits))
	coordinates := make(map[string]struct{}, len(emits))
	for index, emit := range emits {
		path := fmt.Sprintf("$.observation.emits[%d]", index)
		if err := v.validateEmit(emit, path, coordinates); err != nil {
			return nil, err
		}
		emitIDs[emit.GetDefinitionId()] = struct{}{}
	}
	return emitIDs, nil
}

func (v *contractValidator) validateEmit(emit *umpirespb.Emit, path string, coordinates map[string]struct{}) error {
	if emit == nil || !validDefinitionID(emit.GetDefinitionId()) {
		return admissionError(ErrorMalformedValue, path, "emit definition ID is invalid")
	}
	if _, ok := v.kinds[emit.GetSourceKindDefinitionId()]; !ok {
		return admissionError(ErrorBinding, path+".sourceKindDefinitionId", "source kind is not declared")
	}
	if err := v.validateBinding(emit.GetOutputDefinition(), path+".outputDefinition"); err != nil {
		return err
	}
	if emit.GetOutputKind() == umpirespb.DEFINITION_KIND_UNSPECIFIED {
		return admissionError(ErrorUnsupportedEnum, path+".outputKind", "definition kind is unspecified")
	}
	if err := validateCoordinate(emit.GetCoordinate(), path+".coordinate"); err != nil {
		return err
	}
	coordinate := coordinateKey(emit.GetCoordinate())
	if _, duplicate := coordinates[coordinate]; duplicate {
		return admissionError(ErrorDuplicate, path+".coordinate", "duplicate emitted coordinate")
	}
	coordinates[coordinate] = struct{}{}
	if err := v.validateExpression(emit.GetCondition(), path+".condition", 1); err != nil {
		return err
	}
	return v.validateExpression(emit.GetValue(), path+".value", 1)
}

func (v *contractValidator) validateEmitOrdering(orderingValues []*umpirespb.EmitOrdering, emitIDs map[string]struct{}) error {
	if err := v.collection("$.observation.ordering", len(orderingValues), false); err != nil {
		return err
	}
	if err := requireSortedUnique(orderingValues, func(ordering *umpirespb.EmitOrdering) string {
		return ordering.GetPredecessorEmitDefinitionId() + "\x00" + ordering.GetSuccessorEmitDefinitionId()
	}, "$.observation.ordering"); err != nil {
		return err
	}
	for index, ordering := range orderingValues {
		path := fmt.Sprintf("$.observation.ordering[%d]", index)
		if ordering == nil || ordering.GetPredecessorEmitDefinitionId() == ordering.GetSuccessorEmitDefinitionId() {
			return admissionError(ErrorBinding, path, "emit ordering is malformed")
		}
		if _, ok := emitIDs[ordering.GetPredecessorEmitDefinitionId()]; !ok {
			return admissionError(ErrorBinding, path+".predecessorEmitDefinitionId", "emit is not declared")
		}
		if _, ok := emitIDs[ordering.GetSuccessorEmitDefinitionId()]; !ok {
			return admissionError(ErrorBinding, path+".successorEmitDefinitionId", "emit is not declared")
		}
	}
	if hasEmitOrderingCycle(orderingValues, emitIDs) {
		return admissionError(ErrorOrdering, "$.observation.ordering", "emit ordering contains a cycle")
	}
	return nil
}

func hasEmitOrderingCycle(orderingValues []*umpirespb.EmitOrdering, emitIDs map[string]struct{}) bool {
	adjacency := make(map[string][]string, len(emitIDs))
	indegree := make(map[string]int, len(emitIDs))
	for emitID := range emitIDs {
		indegree[emitID] = 0
	}
	for _, ordering := range orderingValues {
		predecessor := ordering.GetPredecessorEmitDefinitionId()
		successor := ordering.GetSuccessorEmitDefinitionId()
		adjacency[predecessor] = append(adjacency[predecessor], successor)
		indegree[successor]++
	}
	ready := make([]string, 0, len(emitIDs))
	for emitID, degree := range indegree {
		if degree == 0 {
			ready = append(ready, emitID)
		}
	}
	slices.Sort(ready)
	visited := 0
	for len(ready) > 0 {
		current := ready[0]
		ready = ready[1:]
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

func (v *contractValidator) validateEvidenceProfile(profile *umpirespb.EvidenceProfile) error {
	if profile == nil {
		return admissionError(ErrorMalformedValue, "$.observation.profile", "Evidence profile is required")
	}
	if err := v.validateBinding(profile.GetDefinition(), "$.observation.profile.definition"); err != nil {
		return err
	}
	if profile.GetVersion() != 1 {
		return admissionError(ErrorUnsupportedVersion, "$.observation.profile.version", "got %d; want 1", profile.GetVersion())
	}
	if err := v.collection("$.observation.profile.sources", len(profile.GetSources()), true); err != nil {
		return err
	}
	if err := requireSortedUnique(profile.GetSources(), func(source *umpirespb.EvidenceSourceDeclaration) string {
		return source.GetSourceDefinitionId()
	}, "$.observation.profile.sources"); err != nil {
		return err
	}
	for index, source := range profile.GetSources() {
		path := fmt.Sprintf("$.observation.profile.sources[%d].sourceDefinitionId", index)
		if source == nil || !validDefinitionID(source.GetSourceDefinitionId()) {
			return admissionError(ErrorMalformedValue, path, "source definition ID is invalid")
		}
		v.sources[source.GetSourceDefinitionId()] = struct{}{}
	}
	if err := v.validateDigestPolicies(profile.GetDigestPolicies()); err != nil {
		return err
	}
	if err := v.validateEvidenceKinds(profile.GetKinds()); err != nil {
		return err
	}
	if err := v.validateCardinalities(profile.GetCardinalities()); err != nil {
		return err
	}
	return v.validateCorrelationSlots(profile.GetCorrelationSlots())
}

func (v *contractValidator) validateDigestPolicies(policies []*umpirespb.DigestPolicy) error {
	path := "$.observation.profile.digestPolicies"
	if err := v.collection(path, len(policies), false); err != nil {
		return err
	}
	if err := requireSortedUnique(policies, func(policy *umpirespb.DigestPolicy) string {
		return policy.GetDefinitionId()
	}, path); err != nil {
		return err
	}
	for index, policy := range policies {
		itemPath := fmt.Sprintf("%s[%d]", path, index)
		if policy == nil || !validDefinitionID(policy.GetDefinitionId()) {
			return admissionError(ErrorMalformedValue, itemPath, "digest policy definition ID is invalid")
		}
		if policy.GetAlgorithm() != umpirespb.DIGEST_ALGORITHM_SYNTHETIC_DIGEST_V1 {
			return admissionError(ErrorUnsupportedEnum, itemPath+".algorithm", "digest algorithm is unsupported")
		}
		v.digestPolicy[policy.GetDefinitionId()] = struct{}{}
	}
	return nil
}

func (v *contractValidator) validateEvidenceKinds(kinds []*umpirespb.EvidenceKindDeclaration) error {
	path := "$.observation.profile.kinds"
	if err := v.collection(path, len(kinds), true); err != nil {
		return err
	}
	if err := requireSortedUnique(kinds, func(kind *umpirespb.EvidenceKindDeclaration) string {
		return kind.GetKindDefinitionId()
	}, path); err != nil {
		return err
	}
	for index, kind := range kinds {
		itemPath := fmt.Sprintf("%s[%d]", path, index)
		if err := v.validateEvidenceKind(kind, itemPath); err != nil {
			return err
		}
	}
	return nil
}

func (v *contractValidator) validateEvidenceKind(kind *umpirespb.EvidenceKindDeclaration, path string) error {
	if kind == nil || !validDefinitionID(kind.GetKindDefinitionId()) {
		return admissionError(ErrorMalformedValue, path, "kind definition ID is invalid")
	}
	if _, ok := v.sources[kind.GetSourceDefinitionId()]; !ok {
		return admissionError(ErrorBinding, path+".sourceDefinitionId", "source is not declared")
	}
	if err := v.collection(path+".fields", len(kind.GetFields()), true); err != nil {
		return err
	}
	if err := requireSortedUnique(kind.GetFields(), func(field *umpirespb.EvidenceFieldDeclaration) string {
		return field.GetFieldDefinitionId()
	}, path+".fields"); err != nil {
		return err
	}
	fields := make(map[string]umpirespb.ValueKind, len(kind.GetFields()))
	for index, field := range kind.GetFields() {
		if err := v.validateEvidenceField(field, fmt.Sprintf("%s.fields[%d]", path, index)); err != nil {
			return err
		}
		fields[field.GetFieldDefinitionId()] = field.GetValueKind()
	}
	v.kinds[kind.GetKindDefinitionId()] = fields
	return nil
}

func (v *contractValidator) validateEvidenceField(field *umpirespb.EvidenceFieldDeclaration, path string) error {
	if field == nil || !validDefinitionID(field.GetFieldDefinitionId()) {
		return admissionError(ErrorMalformedValue, path, "field definition ID is invalid")
	}
	if field.GetValueKind() == umpirespb.VALUE_KIND_UNSPECIFIED || field.GetDisposition() == umpirespb.FIELD_DISPOSITION_KIND_UNSPECIFIED {
		return admissionError(ErrorUnsupportedEnum, path, "field kind or disposition is unspecified")
	}
	if field.GetDisposition() == umpirespb.FIELD_DISPOSITION_KIND_HASH {
		if _, ok := v.digestPolicy[field.GetDigestPolicyDefinitionId()]; !ok {
			return admissionError(ErrorBinding, path+".digestPolicyDefinitionId", "digest policy is not declared")
		}
	} else if field.GetDigestPolicyDefinitionId() != "" {
		return admissionError(ErrorBinding, path+".digestPolicyDefinitionId", "only hashed fields may name a digest policy")
	}
	return nil
}

func (v *contractValidator) validateCardinalities(cardinalities []*umpirespb.EvidenceCardinality) error {
	path := "$.observation.profile.cardinalities"
	if err := v.collection(path, len(cardinalities), false); err != nil {
		return err
	}
	if err := requireSortedUnique(cardinalities, func(cardinality *umpirespb.EvidenceCardinality) string {
		return cardinality.GetKindDefinitionId()
	}, path); err != nil {
		return err
	}
	for index, cardinality := range cardinalities {
		itemPath := fmt.Sprintf("%s[%d]", path, index)
		if cardinality == nil {
			return admissionError(ErrorMalformedValue, itemPath, "cardinality is required")
		}
		if _, ok := v.kinds[cardinality.GetKindDefinitionId()]; !ok {
			return admissionError(ErrorBinding, itemPath+".kindDefinitionId", "kind is not declared")
		}
		if cardinality.GetMinimum() < 0 || cardinality.GetMaximum() < cardinality.GetMinimum() ||
			cardinality.GetMaximum() > v.limits.GetMaxEvidenceRecords() {
			return admissionError(ErrorLimit, itemPath, "cardinality is outside Evidence limits")
		}
	}
	return nil
}

func (v *contractValidator) validateCorrelationSlots(slots []*umpirespb.CorrelationSlot) error {
	path := "$.observation.profile.correlationSlots"
	if err := v.collection(path, len(slots), false); err != nil {
		return err
	}
	if err := requireSortedUnique(slots, func(slot *umpirespb.CorrelationSlot) string {
		return slot.GetDefinitionId()
	}, path); err != nil {
		return err
	}
	for index, slot := range slots {
		itemPath := fmt.Sprintf("%s[%d]", path, index)
		if slot == nil || !validDefinitionID(slot.GetDefinitionId()) {
			return admissionError(ErrorMalformedValue, itemPath, "correlation slot definition ID is invalid")
		}
		if slot.GetKind() == umpirespb.CORRELATION_SLOT_KIND_UNSPECIFIED {
			return admissionError(ErrorUnsupportedEnum, itemPath+".kind", "correlation slot kind is unspecified")
		}
		if err := v.collection(itemPath+".fields", len(slot.GetFields()), true); err != nil {
			return err
		}
		if err := requireSortedUnique(slot.GetFields(), fieldReferenceKey, itemPath+".fields"); err != nil {
			return err
		}
		for fieldIndex, field := range slot.GetFields() {
			if err := v.validateFieldReference(field, fmt.Sprintf("%s.fields[%d]", itemPath, fieldIndex)); err != nil {
				return err
			}
		}
	}
	return nil
}

func (v *contractValidator) validateExpression(expression *umpirespb.ObservationExpression, path string, depth int64) error {
	if expression == nil {
		return admissionError(ErrorUnsupportedOperator, path, "Observation operator is required")
	}
	if depth > v.limits.GetMaxExpressionDepth() {
		return admissionError(ErrorLimit, path, "expression depth %d exceeds limit %d", depth, v.limits.GetMaxExpressionDepth())
	}
	switch operator := expression.GetOperator().(type) {
	case *umpirespb.ObservationExpression_LiteralText:
		if operator.LiteralText == nil {
			return admissionError(ErrorMalformedValue, path, "literal_text payload is required")
		}
	case *umpirespb.ObservationExpression_LiteralNatural:
		if operator.LiteralNatural == nil || !validNatural(operator.LiteralNatural.GetValue()) ||
			compareNatural(operator.LiteralNatural.GetValue(), v.limits.GetMaxNatural()) > 0 {
			return admissionError(ErrorMalformedValue, path, "literal natural is noncanonical or out of range")
		}
	case *umpirespb.ObservationExpression_Field:
		return v.validateFieldReference(operator.Field, path+".field")
	case *umpirespb.ObservationExpression_NaturalRenderV1:
		if operator.NaturalRenderV1 == nil {
			return admissionError(ErrorMalformedValue, path, "natural_render_v1 payload is required")
		}
		return v.validateExpression(operator.NaturalRenderV1.GetOperand(), path+".naturalRenderV1.operand", depth+1)
	case *umpirespb.ObservationExpression_Present:
		if operator.Present == nil {
			return admissionError(ErrorMalformedValue, path, "present payload is required")
		}
		return v.validateExpression(operator.Present.GetOperand(), path+".present.operand", depth+1)
	case *umpirespb.ObservationExpression_Equals:
		if operator.Equals == nil {
			return admissionError(ErrorMalformedValue, path, "equals payload is required")
		}
		if err := v.validateExpression(operator.Equals.GetLeft(), path+".equals.left", depth+1); err != nil {
			return err
		}
		return v.validateExpression(operator.Equals.GetRight(), path+".equals.right", depth+1)
	case *umpirespb.ObservationExpression_All:
		if operator.All == nil {
			return admissionError(ErrorMalformedValue, path, "all payload is required")
		}
		return v.validateOperands(operator.All.GetOperands(), path+".all.operands", depth)
	case *umpirespb.ObservationExpression_Any:
		if operator.Any == nil {
			return admissionError(ErrorMalformedValue, path, "any payload is required")
		}
		return v.validateOperands(operator.Any.GetOperands(), path+".any.operands", depth)
	default:
		return admissionError(ErrorUnsupportedOperator, path, "Observation operator is unspecified or unsupported")
	}
	return nil
}

func (v *contractValidator) validateOperands(operands []*umpirespb.ObservationExpression, path string, depth int64) error {
	if err := v.collection(path, len(operands), true); err != nil {
		return err
	}
	for index, operand := range operands {
		if err := v.validateExpression(operand, fmt.Sprintf("%s[%d]", path, index), depth+1); err != nil {
			return err
		}
	}
	return nil
}

func (v *contractValidator) validateFieldReference(reference *umpirespb.EvidenceFieldReference, path string) error {
	if reference == nil {
		return admissionError(ErrorBinding, path, "field reference is required")
	}
	fields, ok := v.kinds[reference.GetKindDefinitionId()]
	if !ok {
		return admissionError(ErrorBinding, path+".kindDefinitionId", "kind is not declared")
	}
	if _, ok := fields[reference.GetFieldDefinitionId()]; !ok {
		return admissionError(ErrorBinding, path+".fieldDefinitionId", "field is not declared for kind")
	}
	return nil
}

func (v *contractValidator) validateLink() error {
	link := v.contract.GetImplementationLink()
	if link == nil {
		return admissionError(ErrorUnsupportedOperator, "$.implementationLink", "rename_exact link is required")
	}
	if err := v.validateBinding(link.GetDefinition(), "$.implementationLink.definition"); err != nil {
		return err
	}
	if err := v.validateLocation(link.GetSource(), "$.implementationLink.source"); err != nil {
		return err
	}
	if err := v.validateBinding(link.GetSourceTarget(), "$.implementationLink.sourceTarget"); err != nil {
		return err
	}
	if err := v.validateBinding(link.GetDestinationTarget(), "$.implementationLink.destinationTarget"); err != nil {
		return err
	}
	if link.GetSourceTarget().GetDefinitionId() == link.GetDestinationTarget().GetDefinitionId() {
		return admissionError(ErrorBinding, "$.implementationLink", "source and destination targets must differ")
	}
	if err := validateLimit(link.GetApplicationLimit(), "$.implementationLink.applicationLimit", applicationLimitUnit); err != nil {
		return err
	}
	if link.GetApplicationLimit().GetValue() > v.limits.GetMaxEvaluationWork() {
		return admissionError(ErrorLimit, "$.implementationLink.applicationLimit", "application limit exceeds evaluation work limit")
	}
	if err := v.validateRenameEntries(link.GetEntries()); err != nil {
		return err
	}
	return v.validateDefinitionRenameEntries(link.GetDefinitionEntries())
}

func (v *contractValidator) validateRenameEntries(entries []*umpirespb.RenameExactEntry) error {
	if err := v.collection("$.implementationLink.entries", len(entries), true); err != nil {
		return err
	}
	if err := requireSortedUnique(entries, renameEntryKey, "$.implementationLink.entries"); err != nil {
		return err
	}
	sources := make(map[string]struct{}, len(entries))
	for index, entry := range entries {
		path := fmt.Sprintf("$.implementationLink.entries[%d]", index)
		if entry == nil {
			return admissionError(ErrorMalformedValue, path, "rename entry is required")
		}
		if err := v.validateModelValue(entry.GetSource(), path+".source"); err != nil {
			return err
		}
		source := modelValueKey(entry.GetSource())
		if _, duplicate := sources[source]; duplicate {
			return admissionError(ErrorDuplicate, path+".source", "source has duplicate or contradictory mappings")
		}
		sources[source] = struct{}{}
		if err := v.validateModelValue(entry.GetDestination(), path+".destination"); err != nil {
			return err
		}
	}
	return nil
}

func (v *contractValidator) validateDefinitionRenameEntries(entries []*umpirespb.DefinitionRenameEntry) error {
	if err := v.collection("$.implementationLink.definitionEntries", len(entries), false); err != nil {
		return err
	}
	if err := requireSortedUnique(entries, definitionRenameEntryKey,
		"$.implementationLink.definitionEntries"); err != nil {
		return err
	}
	sources := make(map[string]struct{}, len(entries))
	for index, entry := range entries {
		path := fmt.Sprintf("$.implementationLink.definitionEntries[%d]", index)
		if entry == nil {
			return admissionError(ErrorMalformedValue, path, "definition rename entry is required")
		}
		if err := v.validateBinding(entry.GetSource(), path+".source"); err != nil {
			return err
		}
		if entry.GetKind() == umpirespb.DEFINITION_KIND_UNSPECIFIED {
			return admissionError(ErrorUnsupportedEnum, path+".kind", "definition kind is unspecified")
		}
		source := bindingKey(entry.GetSource()) + "\x00" + strconv.FormatInt(int64(entry.GetKind()), 10)
		if _, duplicate := sources[source]; duplicate {
			return admissionError(ErrorDuplicate, path+".source", "source has duplicate or contradictory mappings")
		}
		sources[source] = struct{}{}
		if err := v.validateBinding(entry.GetDestination(), path+".destination"); err != nil {
			return err
		}
	}
	return nil
}

func (v *contractValidator) validateProperties() error {
	properties := v.contract.GetProperties()
	if err := v.collection("$.properties", len(properties), true); err != nil {
		return err
	}
	if err := requireSortedUnique(properties, func(property *umpirespb.Property) string {
		return bindingKey(property.GetDefinition())
	}, "$.properties"); err != nil {
		return err
	}
	for index, property := range properties {
		path := fmt.Sprintf("$.properties[%d]", index)
		if err := v.validateProperty(property, path); err != nil {
			return err
		}
	}
	return nil
}

func (v *contractValidator) validateProperty(property *umpirespb.Property, path string) error {
	if property == nil {
		return admissionError(ErrorMalformedValue, path, "property is required")
	}
	if err := v.validateBinding(property.GetDefinition(), path+".definition"); err != nil {
		return err
	}
	if err := v.validateLocation(property.GetSource(), path+".source"); err != nil {
		return err
	}
	if err := v.collection(path+".requirements", len(property.GetRequirements()), false); err != nil {
		return err
	}
	if err := requireSortedUnique(property.GetRequirements(), bindingKey, path+".requirements"); err != nil {
		return err
	}
	for index, requirement := range property.GetRequirements() {
		if err := v.validateBinding(requirement, fmt.Sprintf("%s.requirements[%d]", path, index)); err != nil {
			return err
		}
	}
	if err := v.collection(path+".clauses", len(property.GetClauses()), true); err != nil {
		return err
	}
	if err := requireSortedUnique(property.GetClauses(), func(clause *umpirespb.PropertyClause) string {
		return clause.GetDefinitionId()
	}, path+".clauses"); err != nil {
		return err
	}
	for index, clause := range property.GetClauses() {
		if err := v.validateClause(clause, fmt.Sprintf("%s.clauses[%d]", path, index)); err != nil {
			return err
		}
	}
	return nil
}

func (v *contractValidator) validateClause(clause *umpirespb.PropertyClause, path string) error {
	if clause == nil || !validDefinitionID(clause.GetDefinitionId()) {
		return admissionError(ErrorMalformedValue, path, "clause definition ID is invalid")
	}
	if clause.GetProvenance() == umpirespb.PROPERTY_CLAUSE_PROVENANCE_UNSPECIFIED {
		return admissionError(ErrorUnsupportedEnum, path+".provenance", "clause provenance is unspecified")
	}
	operator := clause.GetPerStepImplies()
	if operator == nil {
		return admissionError(ErrorUnsupportedOperator, path+".perStepImplies", "per_step_implies operator is required")
	}
	if err := v.validatePattern(operator.GetTrigger(), path+".perStepImplies.trigger"); err != nil {
		return err
	}
	return v.validatePattern(operator.GetRequired(), path+".perStepImplies.required")
}

func (v *contractValidator) validatePattern(pattern *umpirespb.Pattern, path string) error {
	if pattern == nil || pattern.GetField() == umpirespb.TRACE_FIELD_UNSPECIFIED {
		return admissionError(ErrorMalformedValue, path, "pattern field is required")
	}
	if err := v.validateBinding(pattern.GetDefinition(), path+".definition"); err != nil {
		return err
	}
	switch operator := pattern.GetOperator().(type) {
	case *umpirespb.Pattern_EqualsText:
		if operator.EqualsText == nil {
			return admissionError(ErrorMalformedValue, path, "equals_text payload is required")
		}
	case *umpirespb.Pattern_NaturalAtMost:
		if operator.NaturalAtMost == nil || !validNatural(operator.NaturalAtMost.GetBound()) ||
			compareNatural(operator.NaturalAtMost.GetBound(), v.limits.GetMaxNatural()) > 0 {
			return admissionError(ErrorMalformedValue, path, "natural_at_most bound is noncanonical or out of range")
		}
	default:
		return admissionError(ErrorUnsupportedOperator, path, "Property pattern operator is unspecified or unsupported")
	}
	return nil
}

func (v *contractValidator) validateKnownGaps() error {
	gaps := v.contract.GetKnownGaps()
	if err := v.collection("$.knownGaps", len(gaps), false); err != nil {
		return err
	}
	if err := requireSortedUnique(gaps, knownGapKey, "$.knownGaps"); err != nil {
		return err
	}
	for index, gap := range gaps {
		path := fmt.Sprintf("$.knownGaps[%d]", index)
		if gap == nil || gap.GetKind() == umpirespb.KNOWN_GAP_KIND_UNSPECIFIED || !validDefinitionID(gap.GetCode()) {
			return admissionError(ErrorMalformedValue, path, "Known Gap is malformed")
		}
		if gap.GetSubject() != "" && !validDefinitionID(gap.GetSubject()) {
			return admissionError(ErrorMalformedValue, path+".subject", "subject definition ID is invalid")
		}
	}
	return nil
}

func (v *contractValidator) validateLocations(locations []*umpirespb.SourceLocation, path string, required bool) error {
	if err := v.collection(path, len(locations), required); err != nil {
		return err
	}
	if err := requireSortedUnique(locations, sourceLocationKey, path); err != nil {
		return err
	}
	for index, location := range locations {
		if err := v.validateLocation(location, fmt.Sprintf("%s[%d]", path, index)); err != nil {
			return err
		}
	}
	return nil
}

func (*contractValidator) validateLocation(location *umpirespb.SourceLocation, path string) error {
	if location == nil || strings.TrimSpace(location.GetPath()) == "" || location.GetLine() <= 0 ||
		location.GetColumn() <= 0 || strings.TrimSpace(location.GetProvenance()) == "" {
		return admissionError(ErrorMalformedValue, path, "source location is malformed")
	}
	return nil
}

func (v *contractValidator) validateModelValue(value *umpirespb.ModelValue, path string) error {
	if value == nil {
		return admissionError(ErrorMalformedValue, path, "model value is required")
	}
	if err := v.validateBinding(value.GetDefinition(), path+".definition"); err != nil {
		return err
	}
	if value.GetKind() == umpirespb.DEFINITION_KIND_UNSPECIFIED {
		return admissionError(ErrorUnsupportedEnum, path+".kind", "definition kind is unspecified")
	}
	if value.GetValue() == nil || value.GetValue().GetValue() == nil {
		return admissionError(ErrorMalformedValue, path+".value", "tagged value is required")
	}
	if natural, ok := value.GetValue().GetValue().(*umpirespb.Value_Natural); ok {
		if !validNatural(natural.Natural) || compareNatural(natural.Natural, v.limits.GetMaxNatural()) > 0 {
			return admissionError(ErrorMalformedValue, path+".value.natural", "natural is noncanonical or out of range")
		}
	}
	return nil
}

func (v *contractValidator) collection(path string, length int, required bool) error {
	if required && length == 0 {
		return admissionError(ErrorMalformedValue, path, "collection must not be empty")
	}
	if int64(length) > v.limits.GetMaxCollectionItems() {
		return admissionError(ErrorLimit, path, "collection has %d items; limit is %d", length, v.limits.GetMaxCollectionItems())
	}
	return nil
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

func validateLimit(limit *umpirespb.Limit, path, unit string) error {
	if limit == nil || limit.GetValue() <= 0 || limit.GetUnit() != unit {
		return admissionError(ErrorLimit, path, "positive typed Limit is required")
	}
	return nil
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
	for _, character := range []byte(value) {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

func compareNatural(left, right string) int {
	if len(left) != len(right) {
		return len(left) - len(right)
	}
	return strings.Compare(left, right)
}

func fieldReferenceKey(reference *umpirespb.EvidenceFieldReference) string {
	if reference == nil {
		return ""
	}
	return reference.GetKindDefinitionId() + "\x00" + reference.GetFieldDefinitionId()
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
	switch tagged := value.GetValue().(type) {
	case *umpirespb.Value_Text:
		return "text\x00" + tagged.Text
	case *umpirespb.Value_Natural:
		return "natural\x00" + tagged.Natural
	case *umpirespb.Value_BoolValue:
		return "boolean\x00" + strconv.FormatBool(tagged.BoolValue)
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
	return bindingKey(entry.GetSource()) + "\x00" + strconv.FormatInt(int64(entry.GetKind()), 10) + "\x00" + bindingKey(entry.GetDestination())
}

func coordinateKey(coordinate *umpirespb.ModelCoordinate) string {
	if coordinate == nil {
		return ""
	}
	return fmt.Sprintf("%010d\x00%020d\x00%020d", coordinate.GetField(), coordinate.GetStep(), coordinate.GetPosition())
}

func knownGapKey(gap *umpirespb.KnownGap) string {
	if gap == nil {
		return ""
	}
	return fmt.Sprintf("%010d\x00%s\x00%s\x00%s", gap.GetKind(), gap.GetCode(), gap.GetSubject(), gap.GetDetail())
}

func sourceLocationKey(location *umpirespb.SourceLocation) string {
	if location == nil {
		return ""
	}
	var encoded bytes.Buffer
	encoded.WriteString(location.GetPath())
	encoded.WriteByte(0)
	_, _ = fmt.Fprintf(&encoded, "%020d\x00%020d\x00", location.GetLine(), location.GetColumn())
	encoded.WriteString(location.GetProvenance())
	return encoded.String()
}
