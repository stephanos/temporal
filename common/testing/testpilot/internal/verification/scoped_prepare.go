package verification

import (
	"fmt"
	"slices"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/proto"
)

func scopedValue(v *testpilotspb.ScopedValue) bool { return v != nil && validID(v.DefinitionId) }
func sameResult(a, b *testpilotspb.ScopedTransition) bool {
	return proto.Equal(a.Action, b.Action) && proto.Equal(a.ResultingState, b.ResultingState) && proto.Equal(a.Outcome, b.Outcome) && slices.EqualFunc(a.Facts, b.Facts, func(x, y *testpilotspb.ScopedValue) bool { return proto.Equal(x, y) })
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
func validPredicate(p *testpilotspb.ScopedPredicate, trigger bool) bool {
	if p == nil || !validID(p.DefinitionId) {
		return false
	}
	if trigger && p.Field != testpilotspb.SCOPED_PREDICATE_FIELD_ACTION || !trigger && (p.Field < testpilotspb.SCOPED_PREDICATE_FIELD_OUTCOME || p.Field > testpilotspb.SCOPED_PREDICATE_FIELD_FACT) {
		return false
	}
	switch c := p.Constraint.(type) {
	case *testpilotspb.ScopedPredicate_Present:
		return c.Present
	case *testpilotspb.ScopedPredicate_EqualsText:
		return true
	default:
		return false
	}
}
func (a *admission) bindScoped(seen map[string]bool) error {
	s := a.prepared.source.Scoped
	if s == nil {
		return nil
	}
	if s.Version != 1 {
		return invalid(ir.Unknown, "unsupported scoped capability version")
	}
	if !validID(s.ProjectionId) || s.ProjectionFingerprint == "" || !validID(s.OperationField) || !uniqueIDs(s.ScopeFields) || slices.Contains(s.ScopeFields, s.OperationField) || !uniqueIDs(s.Sources) || !scopedValue(s.InitialState) {
		return invalid(ir.Malformed, "invalid scoped projection binding")
	}
	typ, ok := a.prepared.observations[s.EvidenceObservationId]
	if !ok || typ.Cardinality() != ir.Singular || !ir.SameMessage(typ.Message(), (&testpilotspb.ScopedEvidence{}).ProtoReflect().Descriptor()) {
		return invalid(ir.TypeMismatch, "scoped evidence requires exact declared ScopedEvidence Observation")
	}
	l := s.Limits
	if l == nil {
		return invalid(ir.Malformed, "scoped limits required")
	}
	fields := l.ProtoReflect().Descriptor().Fields()
	for i := 0; i < fields.Len(); i++ {
		f := fields.Get(i)
		// The capture ceilings are required only by a capability that declares captures or a
		// correlation; one that declares neither leaves both unset and keeps its exact encoding.
		if f.Name() == "max_captures" || f.Name() == "max_correlation_depth" {
			continue
		}
		if l.ProtoReflect().Get(f).Int() <= 0 {
			return invalid(ir.LimitExceeded, "scoped limits must be positive")
		}
	}
	capturesDeclared := slices.ContainsFunc(s.Clauses, func(c *testpilotspb.ScopedClause) bool { return len(c.Captures) > 0 })
	correlationDeclared := slices.ContainsFunc(s.Clauses, func(c *testpilotspb.ScopedClause) bool { return c.Correlation != nil })
	if l.MaxCaptures < 0 || l.MaxCorrelationDepth < 0 || capturesDeclared && l.MaxCaptures <= 0 || correlationDeclared && l.MaxCorrelationDepth <= 0 {
		return invalid(ir.LimitExceeded, "scoped capture limits must be positive when declared")
	}
	limits := a.prepared.source.Limits
	if l.MaxEvents > a.prepared.program.Limits().MaxRunEvents || l.MaxBuffered > l.MaxEvents || l.MaxKeys > l.MaxEvents || l.MaxEventBytes > a.prepared.program.Limits().MaxResponseBytes || l.MaxProjectionWork > limits.MaxTotalWork || l.MaxObligationWork > limits.MaxTotalWork || l.MaxSemanticTransitions > limits.MaxTransitions {
		return invalid(ir.LimitExceeded, "incompatible scoped limits")
	}
	if err := add(&a.captures, l.MaxObligations, limits.MaxCaptures); err != nil {
		return err
	}
	if err := add(&a.captures, l.MaxCaptures, limits.MaxCaptures); err != nil {
		return err
	}
	// Retained evidence, support references and countdowns share the Contract's capture budget.
	bytes := int64(0)
	for _, pair := range [][2]int64{{l.MaxEvents, l.MaxEventBytes}, {l.MaxSupport, 8}, {l.MaxObligations, 16}, {l.MaxCaptures, l.MaxEventBytes}} {
		if pair[0] == 0 {
			continue
		}
		if pair[0] > (limits.MaxCaptureBytes-bytes)/pair[1] {
			return invalid(ir.LimitExceeded, "scoped retention exceeds capture bytes")
		}
		bytes += pair[0] * pair[1]
	}
	if err := add(&a.captureBytes, bytes, limits.MaxCaptureBytes); err != nil {
		return err
	}
	if len(s.Transitions) == 0 || len(s.ProjectionRules) == 0 || len(s.Clauses) == 0 {
		return invalid(ir.Malformed, "scoped transition table, projection and clauses required")
	}
	if err := add(&a.transitions, int64(len(s.Transitions)), limits.MaxTransitions); err != nil {
		return err
	}
	states := map[[2]string]bool{{s.InitialState.DefinitionId, s.InitialState.Value}: true}
	for _, tr := range s.Transitions {
		if !scopedValue(tr.PriorState) || !scopedValue(tr.Action) || !scopedValue(tr.ResultingState) || !scopedValue(tr.Outcome) {
			return invalid(ir.Malformed, "invalid scoped transition value")
		}
		states[[2]string{tr.PriorState.DefinitionId, tr.PriorState.Value}] = true
		states[[2]string{tr.ResultingState.DefinitionId, tr.ResultingState.Value}] = true
		for _, fact := range tr.Facts {
			if !scopedValue(fact) {
				return invalid(ir.Malformed, "invalid scoped fact")
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
			return invalid(ir.Malformed, "invalid or repeated scoped evidence kind")
		}
		kinds[r.Kind] = true
		switch r.Meaning {
		case testpilotspb.SCOPED_EVIDENCE_MEANING_IRRELEVANT:
			if r.Submission != nil || len(r.Outputs) > 0 {
				return invalid(ir.Malformed, "irrelevant evidence has semantic outputs")
			}
		case testpilotspb.SCOPED_EVIDENCE_MEANING_SUBMISSION:
			if !scopedValue(r.Submission) || len(r.Outputs) > 0 || !slices.ContainsFunc(s.Transitions, func(tr *testpilotspb.ScopedTransition) bool { return proto.Equal(tr.Action, r.Submission) }) {
				return invalid(ir.Malformed, "invalid submission")
			}
		case testpilotspb.SCOPED_EVIDENCE_MEANING_CONFIRMED:
			if len(r.Outputs) == 0 {
				return invalid(ir.Malformed, "confirmed evidence requires outputs")
			}
			if r.Submission != nil && !slices.ContainsFunc(s.ProjectionRules, func(other *testpilotspb.ScopedProjectionRule) bool {
				return other.Meaning == testpilotspb.SCOPED_EVIDENCE_MEANING_SUBMISSION && proto.Equal(other.Submission, r.Submission)
			}) {
				return invalid(ir.Malformed, "missing submission mapping")
			}
			for _, out := range r.Outputs {
				if out == nil || out.PriorState != nil || !slices.ContainsFunc(s.Transitions, func(tr *testpilotspb.ScopedTransition) bool { return sameResult(tr, out) }) {
					return invalid(ir.Malformed, "projection output absent from transition table")
				}
			}
		default:
			return invalid(ir.Unknown, "unsupported evidence meaning")
		}
		fields := map[string]bool{}
		for _, f := range r.Fields {
			if !validID(f.FieldId) || fields[f.FieldId] || f.GetType().GetKind() < testpilotspb.SCALAR_KIND_TEXT || f.GetType().GetKind() > testpilotspb.SCALAR_KIND_BOOLEAN || f.Disposition < testpilotspb.SCOPED_FIELD_DISPOSITION_RETAIN || f.Disposition > testpilotspb.SCOPED_FIELD_DISPOSITION_REJECT {
				return invalid(ir.Malformed, "invalid field policy")
			}
			fields[f.FieldId] = true
			// Only a field this projection actually retains carries a value a capture or a
			// correlation operand can read.
			if f.Disposition == testpilotspb.SCOPED_FIELD_DISPOSITION_RETAIN {
				if declared, seen := retained[f.FieldId]; seen && declared != f.GetType().GetKind() {
					ambiguous[f.FieldId] = true
				}
				retained[f.FieldId] = f.GetType().GetKind()
			}
		}
	}
	if int64(len(seen)+len(s.Clauses)) > limits.MaxRules {
		return invalid(ir.LimitExceeded, "combined rule count exceeds ceiling")
	}
	// Capture identities are one namespace across the capability: a retained occurrence is named by
	// its capture id and ordinal alone, so two clauses declaring one id would alias the same stream.
	declared := map[string]bool{}
	for _, c := range s.Clauses {
		if !validID(c.ClauseId) || seen[c.ClauseId] {
			return invalid(ir.Malformed, "invalid clause provenance")
		}
		seen[c.ClauseId] = true
		if c.Clock != testpilotspb.SCOPED_CLOCK_OPERATION_TRANSITIONS || c.Bound < 0 || c.Endpoint < testpilotspb.SCOPED_ENDPOINT_RUNTIME_PREFIX || c.Endpoint > testpilotspb.SCOPED_ENDPOINT_DELIBERATELY_CLOSED || !validPredicate(c.Trigger, true) || !validPredicate(c.Response, false) {
			return invalid(ir.Unknown, fmt.Sprintf("unsupported scoped clause %s", c.ClauseId))
		}
		captures := map[string]scopedCapture{}
		for _, d := range c.Captures {
			kind, ok := retained[d.FieldId]
			if !validID(d.CaptureId) || declared[d.CaptureId] || !validID(d.FieldId) || d.Lifetime <= 0 || !ok || ambiguous[d.FieldId] {
				return invalid(ir.Malformed, fmt.Sprintf("invalid capture declaration in scoped clause %s", c.ClauseId))
			}
			declared[d.CaptureId] = true
			captures[d.CaptureId] = scopedCapture{lifetime: d.Lifetime, kind: kind}
		}
		if c.Correlation != nil {
			if err := validCorrelation(c.Correlation, retained, ambiguous, captures, l.MaxCorrelationDepth); err != nil {
				return err
			}
		}
	}
	return nil
}

func validScopedLiteral(v *testpilotspb.Value) bool {
	switch literal := v.GetValue().(type) {
	case *testpilotspb.Value_Text, *testpilotspb.Value_BoolValue:
		return true
	case *testpilotspb.Value_Natural:
		text := literal.Natural
		if text != "0" && (len(text) == 0 || text[0] < '1' || text[0] > '9') {
			return false
		}
		for _, c := range text {
			if c < '0' || c > '9' {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// scopedCapture is one clause's declared capture: how many occurrences an operation retains and the
// declared scalar kind every occurrence carries.
type scopedCapture struct {
	lifetime int64
	kind     testpilotspb.ScalarKind
}

func scopedLiteralKind(v *testpilotspb.Value) testpilotspb.ScalarKind {
	switch v.GetValue().(type) {
	case *testpilotspb.Value_Text:
		return testpilotspb.SCALAR_KIND_TEXT
	case *testpilotspb.Value_Natural:
		return testpilotspb.SCALAR_KIND_NATURAL
	case *testpilotspb.Value_BoolValue:
		return testpilotspb.SCALAR_KIND_BOOLEAN
	default:
		return testpilotspb.SCALAR_KIND_UNSPECIFIED
	}
}

// validOperand reports the operand's declared scalar kind, so a comparison is checked against the
// types the projection declares instead of comparing values of different kinds.
func validOperand(o *testpilotspb.ScopedOperand, retained map[string]testpilotspb.ScalarKind, ambiguous map[string]bool, captures map[string]scopedCapture) (testpilotspb.ScalarKind, error) {
	switch v := o.GetOperand().(type) {
	case *testpilotspb.ScopedOperand_Literal:
		if !validScopedLiteral(v.Literal) {
			return 0, invalid(ir.TypeMismatch, "unsupported correlation literal")
		}
		return scopedLiteralKind(v.Literal), nil
	case *testpilotspb.ScopedOperand_FieldId:
		kind, ok := retained[v.FieldId]
		if !validID(v.FieldId) || !ok {
			return 0, invalid(ir.Malformed, "unretained correlation field operand")
		}
		if ambiguous[v.FieldId] {
			return 0, invalid(ir.TypeMismatch, "ambiguous retained field type")
		}
		return kind, nil
	case *testpilotspb.ScopedOperand_Capture:
		declaration, ok := captures[v.Capture.GetCaptureId()]
		if !ok || v.Capture.GetOrdinal() < 0 || v.Capture.GetOrdinal() >= declaration.lifetime {
			return 0, invalid(ir.Malformed, "unbound capture reference")
		}
		return declaration.kind, nil
	default:
		return 0, invalid(ir.Unknown, "unsupported correlation operand")
	}
}

// validCorrelation checks the whole condition under the declared depth ceiling. An exhausted depth
// is an explicit rejection, never a silently truncated condition.
func validCorrelation(c *testpilotspb.ScopedCorrelation, retained map[string]testpilotspb.ScalarKind, ambiguous map[string]bool, captures map[string]scopedCapture, depth int64) error {
	if depth <= 0 {
		return invalid(ir.LimitExceeded, "correlation depth exhausted")
	}
	switch v := c.GetCondition().(type) {
	case *testpilotspb.ScopedCorrelation_Predicate:
		if !validPredicate(v.Predicate, true) && !validPredicate(v.Predicate, false) {
			return invalid(ir.Unknown, "unsupported correlation predicate")
		}
		return nil
	case *testpilotspb.ScopedCorrelation_Comparison:
		if v.Comparison.GetOperator() < testpilotspb.SCOPED_COMPARISON_OPERATOR_EQUAL || v.Comparison.GetOperator() > testpilotspb.SCOPED_COMPARISON_OPERATOR_NOT_EQUAL {
			return invalid(ir.Unknown, "unsupported comparison operator")
		}
		left, err := validOperand(v.Comparison.GetLeft(), retained, ambiguous, captures)
		if err != nil {
			return err
		}
		right, err := validOperand(v.Comparison.GetRight(), retained, ambiguous, captures)
		if err != nil {
			return err
		}
		if left != right {
			return invalid(ir.TypeMismatch, "incompatible correlation operand types")
		}
		return nil
	case *testpilotspb.ScopedCorrelation_All, *testpilotspb.ScopedCorrelation_Any:
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
