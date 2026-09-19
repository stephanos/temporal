package verification

import (
	"fmt"
	"slices"
	"strconv"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/proto"
)

func validModelValue(v *testpilotspb.ModelValue) bool {
	return v != nil && validID(v.DefinitionId)
}
func sameResult(a, b *testpilotspb.CorrelatedTransition) bool {
	return proto.Equal(a.Action, b.Action) && proto.Equal(a.State, b.State) && proto.Equal(a.Outcome, b.Outcome) && slices.EqualFunc(a.Facts, b.Facts, func(x, y *testpilotspb.ModelValue) bool { return proto.Equal(x, y) })
}
func uniqueIDs(ids []string) bool {
	seen := map[string]bool{}
	for _, id := range ids {
		if !validID(id) || seen[id] {
			return false
		}
		seen[id] = true
	}
	return len(ids) > 0
}

// stepCondition is the shape a correlated trigger, response and correlation predicate take: a
// correlated step reference tested for presence, or compared EQUAL with a text literal.
type stepCondition struct {
	field        testpilotspb.CorrelatedStepField
	definitionID string
	equals       bool
	text         string
	// path locates the step reference within its condition.
	path string
}

// readStepCondition reads e as a step condition, reporting false for any other shape.
func readStepCondition(e *testpilotspb.Expression) (stepCondition, bool) {
	switch v := e.GetExpression().(type) {
	case *testpilotspb.Expression_Present:
		step := v.Present.GetOperand().GetReference().GetCorrelatedStep()
		return stepCondition{field: step.GetField(), definitionID: step.GetDefinitionId(), path: ".present.reference.correlated_step"}, step != nil
	case *testpilotspb.Expression_Compare:
		step := v.Compare.GetLeft().GetReference().GetCorrelatedStep()
		text, isText := v.Compare.GetRight().GetLiteral().GetValue().(*testpilotspb.Value_TextValue)
		if step == nil || !isText || v.Compare.GetOperator() != testpilotspb.COMPARISON_OPERATOR_EQUAL {
			return stepCondition{}, false
		}
		return stepCondition{field: step.GetField(), definitionID: step.GetDefinitionId(), equals: true, text: text.TextValue, path: ".compare.left.reference.correlated_step"}, true
	default:
		return stepCondition{}, false
	}
}

// admitRuleCondition admits a trigger or response located at path: a step condition, over a
// reference its context admits, reading one of fields. A trigger reads only the step's action and a
// response only its outcome, state or facts, and a condition reading another part rejects at its step
// reference.
func admitRuleCondition(e *testpilotspb.Expression, path, detail string, fields ...testpilotspb.CorrelatedStepField) (bool, error) {
	if err := ir.AdmitReferences(ir.Site{Context: ir.CorrelatedContext, Path: path}, e); err != nil {
		return false, err
	}
	condition, ok := readStepCondition(e)
	if !ok || !validID(condition.definitionID) {
		return false, nil
	}
	if !slices.Contains(fields, condition.field) {
		located := path + condition.path + ".field"
		if len(located) > 256 {
			located = located[:256]
		}
		return false, &ir.Error{Category: ir.Unknown, Path: located, Detail: detail}
	}
	return true, nil
}

var (
	triggerFields  = []testpilotspb.CorrelatedStepField{testpilotspb.CORRELATED_STEP_FIELD_ACTION}
	responseFields = []testpilotspb.CorrelatedStepField{testpilotspb.CORRELATED_STEP_FIELD_OUTCOME, testpilotspb.CORRELATED_STEP_FIELD_STATE, testpilotspb.CORRELATED_STEP_FIELD_FACT}
)

func (a *admission) bindCorrelated(seen map[string]bool) error {
	s := a.prepared.source.Correlated
	if s == nil {
		return nil
	}
	if !validID(s.ProjectionId) || s.ProjectionFingerprint == "" || !validID(s.OperationField) || !uniqueIDs(s.ScopeFields) || slices.Contains(s.ScopeFields, s.OperationField) || !uniqueIDs(s.Sources) || !validModelValue(s.InitialState) {
		return invalid(ir.Malformed, "invalid correlated projection binding")
	}
	typ, ok := a.prepared.observations[s.EvidenceObservationId]
	if !ok || typ.Cardinality() != ir.Singular || !ir.SameMessage(typ.Message(), (&testpilotspb.CorrelatedEvidence{}).ProtoReflect().Descriptor()) {
		return invalid(ir.TypeMismatch, "correlated evidence requires exact declared CorrelatedEvidence Observation")
	}
	l := a.prepared.correlatedLimits
	if l == nil {
		return invalid(ir.Malformed, "Profile correlated limits required")
	}
	if err := ir.CheckSurface(l, ir.DefaultLimits()); err != nil {
		return err
	}
	fields := l.ProtoReflect().Descriptor().Fields()
	for i := 0; i < fields.Len(); i++ {
		f := fields.Get(i)
		// The capture ceilings are required only by a capability that declares captures or a
		// correlation, so a Profile that admits neither may leave both unset.
		if f.Name() == "max_captures" || f.Name() == "max_correlation_depth" {
			continue
		}
		if l.ProtoReflect().Get(f).Int() <= 0 {
			return invalid(ir.LimitExceeded, "correlated limits must be positive")
		}
	}
	capturesDeclared := slices.ContainsFunc(s.Rules, func(c *testpilotspb.CorrelatedRule) bool { return len(c.Captures) > 0 })
	correlationDeclared := slices.ContainsFunc(s.Rules, func(c *testpilotspb.CorrelatedRule) bool { return c.Correlation != nil })
	if l.MaxCaptures < 0 || l.MaxCorrelationDepth < 0 || capturesDeclared && l.MaxCaptures <= 0 || correlationDeclared && l.MaxCorrelationDepth <= 0 {
		return invalid(ir.LimitExceeded, "correlated capture limits must be positive when declared")
	}
	limits := a.prepared.limits
	if l.MaxEvents > a.prepared.program.Limits().MaxRunEvents || l.MaxBuffered > l.MaxEvents || l.MaxKeys > l.MaxEvents || l.MaxEventBytes > a.prepared.program.Limits().MaxResponseBytes || l.MaxProjectionWork > limits.MaxTotalWork || l.MaxObligationWork > limits.MaxTotalWork || l.MaxSemanticTransitions > limits.MaxTransitions {
		return invalid(ir.LimitExceeded, "incompatible correlated limits")
	}
	if err := add(&a.captures, l.MaxObligations, limits.MaxCaptures); err != nil {
		return err
	}
	// Only a correlated contract that declares captures can retain captured values, so only it
	// reserves the Profile's capture ceiling.
	captures := int64(0)
	if capturesDeclared {
		captures = l.MaxCaptures
	}
	if err := add(&a.captures, captures, limits.MaxCaptures); err != nil {
		return err
	}
	// Retained evidence, support references and countdowns share the Contract's capture budget.
	bytes := int64(0)
	for _, pair := range [][2]int64{{l.MaxEvents, l.MaxEventBytes}, {l.MaxSupport, 8}, {l.MaxObligations, 16}, {captures, l.MaxEventBytes}} {
		if pair[0] == 0 {
			continue
		}
		if pair[0] > (limits.MaxCaptureBytes-bytes)/pair[1] {
			return invalid(ir.LimitExceeded, "correlated retention exceeds capture bytes")
		}
		bytes += pair[0] * pair[1]
	}
	if err := add(&a.captureBytes, bytes, limits.MaxCaptureBytes); err != nil {
		return err
	}
	if len(s.Transitions) == 0 || len(s.ProjectionRules) == 0 || len(s.Rules) == 0 {
		return invalid(ir.Malformed, "correlated transition table, projection and clauses required")
	}
	if err := add(&a.transitions, int64(len(s.Transitions)), limits.MaxTransitions); err != nil {
		return err
	}
	states := map[[2]string]bool{{s.InitialState.DefinitionId, s.InitialState.Value}: true}
	for _, tr := range s.Transitions {
		if !validModelValue(tr.PriorState) || !validModelValue(tr.Action) || !validModelValue(tr.State) || !validModelValue(tr.Outcome) {
			return invalid(ir.Malformed, "invalid correlated transition value")
		}
		states[[2]string{tr.PriorState.DefinitionId, tr.PriorState.Value}] = true
		states[[2]string{tr.State.DefinitionId, tr.State.Value}] = true
		for _, fact := range tr.Facts {
			if !validModelValue(fact) {
				return invalid(ir.Malformed, "invalid correlated fact")
			}
		}
	}
	if err := add(&a.states, int64(len(states)), limits.MaxStates); err != nil {
		return err
	}
	// A field two rules declare at different kinds has no single type a comparison could be checked
	// against, so reading it rejects rather than comparing values of different kinds.
	kinds, retained, ambiguous := map[string]bool{}, map[string]testpilotspb.ScalarKind{}, map[string]bool{}
	for _, r := range s.ProjectionRules {
		if !validID(r.Kind) || kinds[r.Kind] {
			return invalid(ir.Malformed, "invalid or repeated correlated evidence kind")
		}
		kinds[r.Kind] = true
		switch r.Meaning {
		case testpilotspb.CORRELATED_EVIDENCE_MEANING_IRRELEVANT:
			if r.Submission != nil || len(r.Outputs) > 0 {
				return invalid(ir.Malformed, "irrelevant evidence has semantic outputs")
			}
		case testpilotspb.CORRELATED_EVIDENCE_MEANING_SUBMISSION:
			if !validModelValue(r.Submission) || len(r.Outputs) > 0 || !slices.ContainsFunc(s.Transitions, func(tr *testpilotspb.CorrelatedTransition) bool { return proto.Equal(tr.Action, r.Submission) }) {
				return invalid(ir.Malformed, "invalid submission")
			}
		case testpilotspb.CORRELATED_EVIDENCE_MEANING_CONFIRMED:
			if len(r.Outputs) == 0 {
				return invalid(ir.Malformed, "confirmed evidence requires outputs")
			}
			if r.Submission != nil && !slices.ContainsFunc(s.ProjectionRules, func(other *testpilotspb.CorrelatedProjectionRule) bool {
				return other.Meaning == testpilotspb.CORRELATED_EVIDENCE_MEANING_SUBMISSION && proto.Equal(other.Submission, r.Submission)
			}) {
				return invalid(ir.Malformed, "missing submission mapping")
			}
			for _, out := range r.Outputs {
				if out == nil || out.PriorState != nil || !slices.ContainsFunc(s.Transitions, func(tr *testpilotspb.CorrelatedTransition) bool { return sameResult(tr, out) }) {
					return invalid(ir.Malformed, "projection output absent from transition table")
				}
			}
		default:
			return invalid(ir.Unknown, "unsupported evidence meaning")
		}
		fields := map[string]bool{}
		for _, f := range r.Fields {
			if !validID(f.FieldId) || fields[f.FieldId] || !correlatedFieldKind(f.GetType().GetKind()) || f.Disposition < testpilotspb.CORRELATED_FIELD_DISPOSITION_RETAIN || f.Disposition > testpilotspb.CORRELATED_FIELD_DISPOSITION_REJECT {
				return invalid(ir.Malformed, "invalid field policy")
			}
			fields[f.FieldId] = true
			// Only a field this projection actually retains carries a value a capture or a
			// correlation operand can read.
			if f.Disposition == testpilotspb.CORRELATED_FIELD_DISPOSITION_RETAIN {
				if declared, seen := retained[f.FieldId]; seen && declared != f.GetType().GetKind() {
					ambiguous[f.FieldId] = true
				}
				retained[f.FieldId] = f.GetType().GetKind()
			}
		}
	}
	if int64(len(seen)+len(s.Rules)) > limits.MaxRules {
		return invalid(ir.LimitExceeded, "combined rule count exceeds ceiling")
	}
	// Capture identities are one namespace across the capability: a retained occurrence is named by
	// its capture id and ordinal alone, so two clauses declaring one id would alias the same stream.
	declared := map[string]bool{}
	for _, c := range s.Rules {
		if !validID(c.RuleId) || seen[c.RuleId] {
			return invalid(ir.Malformed, "invalid clause provenance")
		}
		seen[c.RuleId] = true
		path := fmt.Sprintf("contract.correlated.rules[%s]", c.RuleId)
		trigger, err := admitRuleCondition(c.Trigger, path+".trigger", "a trigger reads only the step's action", triggerFields...)
		if err != nil {
			return err
		}
		response, err := admitRuleCondition(c.Response, path+".response", "a response reads only the step's outcome, state or facts", responseFields...)
		if err != nil {
			return err
		}
		if c.Clock != testpilotspb.CORRELATED_CLOCK_OPERATION_TRANSITIONS || c.Bound < 0 || c.Ending < testpilotspb.TRACE_ENDING_PARTIAL || c.Ending > testpilotspb.TRACE_ENDING_FINAL || !trigger || !response {
			return invalid(ir.Unknown, fmt.Sprintf("unsupported correlated rule %s", c.RuleId))
		}
		captures := map[string]correlatedCapture{}
		for _, d := range c.Captures {
			kind, ok := retained[d.FieldId]
			if !validID(d.CaptureId) || declared[d.CaptureId] || !validID(d.FieldId) || d.Lifetime <= 0 || !ok || ambiguous[d.FieldId] {
				return invalid(ir.Malformed, fmt.Sprintf("invalid capture declaration in correlated rule %s", c.RuleId))
			}
			declared[d.CaptureId] = true
			captures[d.CaptureId] = correlatedCapture{lifetime: d.Lifetime, kind: kind}
		}
		if c.Correlation != nil {
			if err := ir.AdmitReferences(ir.Site{Context: ir.CorrelatedContext, Path: path + ".correlation"}, c.Correlation); err != nil {
				return err
			}
			if err := validCorrelation(c.Correlation, retained, ambiguous, captures, l.MaxCorrelationDepth); err != nil {
				return err
			}
		}
	}
	return nil
}

func validCorrelatedLiteral(v *testpilotspb.Value) bool {
	switch literal := v.GetValue().(type) {
	case *testpilotspb.Value_TextValue, *testpilotspb.Value_BoolValue:
		return true
	case *testpilotspb.Value_UnsignedIntegerValue:
		return canonicalUint64(literal.UnsignedIntegerValue)
	default:
		return false
	}
}

// correlatedFieldKind reports whether kind is one the portable evidence domain declares: text,
// unsigned integer or boolean.
func correlatedFieldKind(kind testpilotspb.ScalarKind) bool {
	return kind == testpilotspb.SCALAR_KIND_TEXT || kind == testpilotspb.SCALAR_KIND_UINT64 || kind == testpilotspb.SCALAR_KIND_BOOLEAN
}

// canonicalUint64 reports whether text is an unsigned 64-bit integer in canonical base-10 text.
func canonicalUint64(text string) bool {
	parsed, err := strconv.ParseUint(text, 10, 64)
	return err == nil && strconv.FormatUint(parsed, 10) == text
}

// correlatedCapture is one clause's declared capture: how many occurrences an operation retains and the
// declared scalar kind every occurrence carries.
type correlatedCapture struct {
	lifetime int64
	kind     testpilotspb.ScalarKind
}

func correlatedLiteralKind(v *testpilotspb.Value) testpilotspb.ScalarKind {
	switch v.GetValue().(type) {
	case *testpilotspb.Value_TextValue:
		return testpilotspb.SCALAR_KIND_TEXT
	case *testpilotspb.Value_UnsignedIntegerValue:
		return testpilotspb.SCALAR_KIND_UINT64
	case *testpilotspb.Value_BoolValue:
		return testpilotspb.SCALAR_KIND_BOOLEAN
	default:
		return testpilotspb.SCALAR_KIND_UNSPECIFIED
	}
}

// validOperand reports the operand's declared scalar kind, so a comparison is checked against the
// types the projection declares instead of comparing values of different kinds. An operand is a
// literal, one declared evidence field of the step being admitted, or one retained earlier
// occurrence of a declared capture.
func validOperand(o *testpilotspb.Expression, retained map[string]testpilotspb.ScalarKind, ambiguous map[string]bool, captures map[string]correlatedCapture) (testpilotspb.ScalarKind, error) {
	if literal, ok := o.GetExpression().(*testpilotspb.Expression_Literal); ok {
		if !validCorrelatedLiteral(literal.Literal) {
			return 0, invalid(ir.TypeMismatch, "unsupported correlation literal")
		}
		return correlatedLiteralKind(literal.Literal), nil
	}
	switch v := o.GetReference().GetReference().(type) {
	case *testpilotspb.Reference_EvidenceFieldId:
		kind, ok := retained[v.EvidenceFieldId]
		if !validID(v.EvidenceFieldId) || !ok {
			return 0, invalid(ir.Malformed, "unretained correlation field operand")
		}
		if ambiguous[v.EvidenceFieldId] {
			return 0, invalid(ir.TypeMismatch, "ambiguous retained field type")
		}
		return kind, nil
	case *testpilotspb.Reference_CorrelatedCapture:
		declaration, ok := captures[v.CorrelatedCapture.GetCaptureId()]
		if !ok || v.CorrelatedCapture.GetOrdinal() < 0 || v.CorrelatedCapture.GetOrdinal() >= declaration.lifetime {
			return 0, invalid(ir.Malformed, "unbound capture reference")
		}
		return declaration.kind, nil
	default:
		return 0, invalid(ir.Unknown, "unsupported correlation operand")
	}
}

// validCorrelation checks the whole condition under the declared depth ceiling. An exhausted depth
// is an explicit rejection, never a silently truncated condition. A step condition and a comparison
// each count as one level, as does every all and any; the expression nodes inside a step condition
// or a comparison do not.
func validCorrelation(c *testpilotspb.Expression, retained map[string]testpilotspb.ScalarKind, ambiguous map[string]bool, captures map[string]correlatedCapture, depth int64) error {
	if depth <= 0 {
		return invalid(ir.LimitExceeded, "correlation depth exhausted")
	}
	if condition, ok := readStepCondition(c); ok {
		if !validID(condition.definitionID) || condition.field < testpilotspb.CORRELATED_STEP_FIELD_ACTION || condition.field > testpilotspb.CORRELATED_STEP_FIELD_FACT {
			return invalid(ir.Unknown, "unsupported correlation predicate")
		}
		return nil
	}
	switch v := c.GetExpression().(type) {
	case *testpilotspb.Expression_Compare:
		if v.Compare.GetOperator() < testpilotspb.COMPARISON_OPERATOR_EQUAL || v.Compare.GetOperator() > testpilotspb.COMPARISON_OPERATOR_NOT_EQUAL {
			return invalid(ir.Unknown, "unsupported comparison operator")
		}
		left, err := validOperand(v.Compare.GetLeft(), retained, ambiguous, captures)
		if err != nil {
			return err
		}
		right, err := validOperand(v.Compare.GetRight(), retained, ambiguous, captures)
		if err != nil {
			return err
		}
		if left != right {
			return invalid(ir.TypeMismatch, "incompatible correlation operand types")
		}
		return nil
	case *testpilotspb.Expression_All, *testpilotspb.Expression_Any:
		operands := c.GetAll().GetOperands()
		if c.GetAny() != nil {
			operands = c.GetAny().GetOperands()
		}
		if len(operands) == 0 {
			return invalid(ir.Malformed, "empty correlation group")
		}
		for _, operand := range operands {
			if err := validCorrelation(operand, retained, ambiguous, captures, depth-1); err != nil {
				return err
			}
		}
		return nil
	default:
		return invalid(ir.Unknown, "unsupported correlation condition")
	}
}
