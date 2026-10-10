package verification

import (
	"fmt"
	"slices"

	celpb "cel.dev/expr"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/proto"
)

func validModelValue(v *testpilotspb.ModelValue) bool {
	return v != nil && ir.ValidID(v.DefinitionId)
}
func validModelValues(vs []*testpilotspb.ModelValue) bool {
	return !slices.ContainsFunc(vs, func(v *testpilotspb.ModelValue) bool { return !validModelValue(v) })
}
func uniqueIDs(ids []string) bool {
	seen := map[string]bool{}
	for _, id := range ids {
		if !ir.ValidID(id) || seen[id] {
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
	call := e.GetCel().GetExpr().GetCallExpr()
	var variable string
	condition := stepCondition{}
	if call.GetFunction() == "@in" && len(call.Args) == 2 {
		text, ok := call.Args[0].GetConstExpr().GetConstantKind().(*celpb.Constant_StringValue)
		if !ok {
			return condition, false
		}
		variable = stepVariable(call.Args[1])
		condition.equals, condition.text = true, text.StringValue
	} else if call.GetFunction() == "_>_" && len(call.Args) == 2 {
		size := call.Args[0].GetCallExpr()
		zero, ok := call.Args[1].GetConstExpr().GetConstantKind().(*celpb.Constant_Int64Value)
		if size.GetFunction() != "size" || len(size.Args) != 1 || !ok || zero.Int64Value != 0 {
			return condition, false
		}
		variable = stepVariable(size.Args[0])
	} else {
		return condition, false
	}
	if variable == "" || len(e.Bindings) != 1 {
		return condition, false
	}
	binding := e.Bindings[0]
	step := binding.GetReference().GetCorrelatedStep()
	if binding.GetVariable() != variable || binding.GetPath() != "" || step == nil {
		return condition, false
	}
	condition.field, condition.definitionID, condition.path = step.Field, step.DefinitionId, ".bindings[0].reference.correlated_step"
	return condition, true
}

func stepVariable(e *celpb.Expr) string {
	call := e.GetCallExpr()
	if call.GetFunction() == "value" && len(call.Args) == 0 {
		return call.GetTarget().GetIdentExpr().GetName()
	}
	return ""
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
	if !ok || !ir.ValidID(condition.definitionID) {
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
	if !ir.ValidID(s.ProjectionId) || s.ProjectionFingerprint == "" || !ir.ValidID(s.OperationField) || !uniqueIDs(s.ScopeFields) || slices.Contains(s.ScopeFields, s.OperationField) || !uniqueIDs(s.Sources) {
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
	// The capture ceilings are required only by a capability that declares captures or a
	// correlation, so a Profile that admits neither may leave both unset.
	if err := ir.CheckCeilings(l, nil, func(string) error {
		return invalid(ir.LimitExceeded, "correlated limits must be positive")
	}, "max_captures", "max_correlation_depth"); err != nil {
		return err
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
	if err := a.bindCorrelatedTables(s); err != nil {
		return err
	}
	// A field two rules declare at different kinds has no single type a comparison could be checked
	// against, so reading it rejects rather than comparing values of different kinds.
	kinds, retained, ambiguous := map[string]bool{}, map[string]testpilotspb.ScalarKind{}, map[string]bool{}
	for _, r := range s.ProjectionRules {
		if !ir.ValidID(r.Kind) || kinds[r.Kind] {
			return invalid(ir.Malformed, "invalid or repeated correlated evidence kind")
		}
		kinds[r.Kind] = true
		switch r.Meaning {
		case testpilotspb.CORRELATED_EVIDENCE_MEANING_IRRELEVANT:
			if r.Submission != nil || len(r.ResultIds) > 0 {
				return invalid(ir.Malformed, "irrelevant evidence has semantic outputs")
			}
		case testpilotspb.CORRELATED_EVIDENCE_MEANING_SUBMISSION:
			if !validModelValue(r.Submission) || len(r.ResultIds) > 0 || !slices.ContainsFunc(s.Results, func(tr *testpilotspb.CorrelatedResult) bool { return proto.Equal(tr.Action, r.Submission) }) {
				return invalid(ir.Malformed, "invalid submission")
			}
		case testpilotspb.CORRELATED_EVIDENCE_MEANING_CONFIRMED:
			if len(r.ResultIds) == 0 {
				return invalid(ir.Malformed, "confirmed evidence requires outputs")
			}
			if r.Submission != nil && !slices.ContainsFunc(s.ProjectionRules, func(other *testpilotspb.CorrelatedProjectionRule) bool {
				return other.Meaning == testpilotspb.CORRELATED_EVIDENCE_MEANING_SUBMISSION && proto.Equal(other.Submission, r.Submission)
			}) {
				return invalid(ir.Malformed, "missing submission mapping")
			}
			for _, id := range r.ResultIds {
				if a.prepared.correlatedResults[id] == nil || !slices.ContainsFunc(s.Transitions, func(tr *testpilotspb.CorrelatedTransition) bool { return tr.ResultId == id }) {
					return invalid(ir.Malformed, "projection output absent from transition table")
				}
			}
		default:
			return invalid(ir.Unknown, "unsupported evidence meaning")
		}
		fields := map[string]bool{}
		for _, f := range r.Fields {
			if !ir.ValidID(f.FieldId) || fields[f.FieldId] || !correlatedFieldKind(f.GetType().GetKind()) || f.Disposition < testpilotspb.CORRELATED_FIELD_DISPOSITION_RETAIN || f.Disposition > testpilotspb.CORRELATED_FIELD_DISPOSITION_REJECT {
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
	if a.ruleCount+int64(len(s.Rules)) > limits.MaxRules {
		return invalid(ir.LimitExceeded, "combined rule count exceeds ceiling")
	}
	// Capture identities are one namespace across the capability: a retained occurrence is named by
	// its capture id and ordinal alone, so two clauses declaring one id would alias the same stream.
	declared := map[string]bool{}
	for _, c := range s.Rules {
		if !ir.ValidID(c.RuleId) || seen[c.RuleId] {
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
		if c.Bound < 0 || c.Ending < testpilotspb.TRACE_ENDING_PARTIAL || c.Ending > testpilotspb.TRACE_ENDING_FINAL || !trigger || !response {
			return invalid(ir.Unknown, fmt.Sprintf("unsupported correlated rule %s", c.RuleId))
		}
		captures := map[string]correlatedCapture{}
		for _, d := range c.Captures {
			kind, ok := retained[d.FieldId]
			if !ir.ValidID(d.CaptureId) || declared[d.CaptureId] || !ir.ValidID(d.FieldId) || d.Lifetime <= 0 || !ok || ambiguous[d.FieldId] {
				return invalid(ir.Malformed, fmt.Sprintf("invalid capture declaration in correlated rule %s", c.RuleId))
			}
			declared[d.CaptureId] = true
			captures[d.CaptureId] = correlatedCapture{lifetime: d.Lifetime, kind: kind}
		}
		bound, err := a.bindCorrelatedPredicates(c, path, retained, ambiguous, captures, l.MaxCorrelationDepth)
		if err != nil {
			return err
		}
		a.prepared.correlatedRules = append(a.prepared.correlatedRules, bound)
	}
	return nil
}

// correlatedFieldKind reports whether kind is one the portable evidence domain declares: text,
// unsigned integer or boolean.
func correlatedFieldKind(kind testpilotspb.ScalarKind) bool {
	return kind == testpilotspb.SCALAR_KIND_TEXT || kind == testpilotspb.SCALAR_KIND_UINT64 || kind == testpilotspb.SCALAR_KIND_BOOLEAN
}

// correlatedCapture is one clause's declared capture: how many occurrences an operation retains and the
// declared scalar kind every occurrence carries.
type correlatedCapture struct {
	lifetime int64
	kind     testpilotspb.ScalarKind
}

type correlatedPredicates struct{ trigger, response, correlation *ir.Expression }

func (a *admission) bindCorrelatedPredicates(rule *testpilotspb.CorrelatedRule, path string, retained map[string]testpilotspb.ScalarKind, ambiguous map[string]bool, captures map[string]correlatedCapture, depth int64) (correlatedPredicates, error) {
	var result correlatedPredicates
	for _, item := range []struct {
		name   string
		source *testpilotspb.Expression
		target **ir.Expression
	}{{"trigger", rule.Trigger, &result.trigger}, {"response", rule.Response, &result.response}, {"correlation", rule.Correlation, &result.correlation}} {
		if item.source == nil {
			continue
		}
		if err := ir.AdmitReferences(ir.Site{Context: ir.CorrelatedContext, Path: path + "." + item.name}, item.source); err != nil {
			return result, err
		}
		scope := map[ir.Reference]ir.Binding{}
		for _, binding := range item.source.Bindings {
			if binding.GetReference() == nil {
				continue
			}
			var ref ir.Reference
			var declared *testpilotspb.ValueType
			switch v := binding.GetReference().GetReference().(type) {
			case *testpilotspb.Reference_CorrelatedStep:
				step := v.CorrelatedStep
				if !ir.ValidID(step.GetDefinitionId()) || step.GetField() < testpilotspb.CORRELATED_STEP_FIELD_ACTION || step.GetField() > testpilotspb.CORRELATED_STEP_FIELD_FACT {
					return result, invalid(ir.Unknown, "unsupported correlation predicate")
				}
				ref = ir.Reference{Kind: ir.CorrelatedStepReference, ID: step.DefinitionId, Field: int32(step.Field)}
				declared = &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Repeated{Repeated: &testpilotspb.RepeatedType{Element: scalarType(testpilotspb.SCALAR_KIND_TEXT).GetSingular()}}}
			case *testpilotspb.Reference_EvidenceFieldId:
				kind, ok := retained[v.EvidenceFieldId]
				if !ir.ValidID(v.EvidenceFieldId) || !ok {
					return result, invalid(ir.Malformed, "unretained correlation field operand")
				}
				if ambiguous[v.EvidenceFieldId] {
					return result, invalid(ir.TypeMismatch, "ambiguous retained field type")
				}
				ref, declared = ir.Reference{Kind: ir.EvidenceFieldReference, ID: v.EvidenceFieldId}, scalarType(kind)
			case *testpilotspb.Reference_CorrelatedCapture:
				capture := v.CorrelatedCapture
				declaration, ok := captures[capture.GetCaptureId()]
				if !ok || capture.GetOrdinal() < 0 || capture.GetOrdinal() >= declaration.lifetime {
					return result, invalid(ir.Malformed, "unbound capture reference")
				}
				ref, declared = ir.Reference{Kind: ir.CorrelatedCaptureReference, ID: capture.CaptureId, Ordinal: capture.Ordinal}, scalarType(declaration.kind)
			default:
				return result, invalid(ir.Unknown, "unsupported correlation operand")
			}
			typ, err := a.catalog.BindType(declared)
			if err != nil {
				return result, err
			}
			scope[ref] = ir.Binding{Type: typ, Available: true}
		}
		limits := a.limits
		if item.name == "correlation" {
			limits.Depth = min(limits.Depth, depth)
		}
		bound, err := a.catalog.BindExpression(ir.Site{Context: ir.CorrelatedContext, Path: path + "." + item.name}, item.source, &a.boolean, scope, limits)
		if err != nil {
			return result, err
		}
		if err := a.charge(bound.BindingWork()); err != nil {
			return result, err
		}
		*item.target = bound
	}
	return result, nil
}

func (a *admission) bindCorrelatedTables(s *testpilotspb.CorrelatedContract) error {
	limits := a.prepared.limits
	if err := add(&a.states, int64(len(s.States)), limits.MaxStates); err != nil {
		return err
	}
	if int64(len(s.Results)) > limits.MaxTransitions {
		return invalid(ir.LimitExceeded, "correlated result count exceeds ceiling")
	}
	// Catalog construction and expanded payload visits are priced before allocating either index.
	for _, count := range []int64{int64(len(s.States)), int64(len(s.Results)), int64(len(s.Transitions)), int64(len(s.ProjectionRules))} {
		if err := a.charge(count); err != nil {
			return err
		}
	}
	var expanded int64
	scan := func(count, width int64) error {
		if count > 0 && width > (a.limits.Work-a.work)/count {
			return invalid(ir.LimitExceeded, "correlated expanded lookup work exceeds ceiling")
		}
		return a.charge(count * width)
	}
	if err := scan(int64(len(s.Transitions)), 2*int64(len(s.States))+int64(len(s.Results))); err != nil {
		return err
	}
	for _, projection := range s.ProjectionRules {
		if err := scan(int64(len(projection.GetResultIds())), int64(len(s.States))+int64(len(s.Results))); err != nil {
			return err
		}
	}
	for _, tr := range s.Transitions {
		prior := slices.IndexFunc(s.States, func(state *testpilotspb.CorrelatedState) bool { return state.GetStateId() == tr.GetPriorStateId() })
		result := slices.IndexFunc(s.Results, func(result *testpilotspb.CorrelatedResult) bool { return result.GetResultId() == tr.GetResultId() })
		if prior < 0 || result < 0 {
			return invalid(ir.Malformed, "invalid or duplicate correlated transition")
		}
		state := slices.IndexFunc(s.States, func(state *testpilotspb.CorrelatedState) bool {
			return state.GetStateId() == s.Results[result].GetStateId()
		})
		if state < 0 {
			return invalid(ir.Malformed, "invalid or duplicate correlated result")
		}
		for _, message := range []proto.Message{s.States[prior], s.Results[result], s.States[state]} {
			if err := add(&expanded, int64(proto.Size(message))+1, a.limits.Work-a.work); err != nil {
				return err
			}
		}
	}
	for _, projection := range s.ProjectionRules {
		for _, id := range projection.GetResultIds() {
			result := slices.IndexFunc(s.Results, func(result *testpilotspb.CorrelatedResult) bool { return result.GetResultId() == id })
			if result < 0 {
				return invalid(ir.Malformed, "projection output absent from result table")
			}
			state := slices.IndexFunc(s.States, func(state *testpilotspb.CorrelatedState) bool {
				return state.GetStateId() == s.Results[result].GetStateId()
			})
			if state < 0 {
				return invalid(ir.Malformed, "invalid or duplicate correlated result")
			}
			for _, message := range []proto.Message{s.Results[result], s.States[state]} {
				if err := add(&expanded, int64(proto.Size(message))+1, a.limits.Work-a.work); err != nil {
					return err
				}
			}
		}
	}
	if err := a.charge(expanded); err != nil {
		return err
	}
	states, results := map[string]*testpilotspb.CorrelatedState{}, map[string]*testpilotspb.CorrelatedResult{}
	complete := map[string]bool{}
	for _, state := range s.States {
		if !ir.ValidID(state.GetStateId()) || states[state.StateId] != nil || !validModelValue(state.Atom) || !validModelValues(state.Fields) {
			return invalid(ir.Malformed, "invalid or duplicate correlated state")
		}
		key, err := proto.MarshalOptions{Deterministic: true}.Marshal(&testpilotspb.CorrelatedState{Atom: state.Atom, Fields: state.Fields})
		if err != nil {
			return err
		}
		if complete[string(key)] {
			return invalid(ir.Malformed, "duplicate complete correlated state")
		}
		complete[string(key)], states[state.StateId] = true, state
	}
	if states[s.InitialStateId] == nil {
		return invalid(ir.Malformed, "undeclared initial correlated state")
	}
	completeResults := map[string]bool{}
	for _, result := range s.Results {
		if !ir.ValidID(result.GetResultId()) || results[result.ResultId] != nil || states[result.StateId] == nil || !validModelValue(result.Action) || !validModelValue(result.Outcome) || !validModelValues(result.Facts) {
			return invalid(ir.Malformed, "invalid or duplicate correlated result")
		}
		key, err := proto.MarshalOptions{Deterministic: true}.Marshal(&testpilotspb.CorrelatedResult{Action: result.Action, StateId: result.StateId, Outcome: result.Outcome, Facts: result.Facts})
		if err != nil {
			return err
		}
		if completeResults[string(key)] {
			return invalid(ir.Malformed, "duplicate complete correlated result")
		}
		completeResults[string(key)] = true
		results[result.ResultId] = result
	}
	seen := map[[2]string]bool{}
	for _, tr := range s.Transitions {
		prior, result := states[tr.GetPriorStateId()], results[tr.GetResultId()]
		key := [2]string{tr.GetPriorStateId(), tr.GetResultId()}
		if prior == nil || result == nil || seen[key] {
			return invalid(ir.Malformed, "invalid or duplicate correlated transition")
		}
		seen[key] = true
	}
	for _, projection := range s.ProjectionRules {
		for _, id := range projection.GetResultIds() {
			result := results[id]
			if result == nil {
				return invalid(ir.Malformed, "projection output absent from result table")
			}
		}
	}
	a.prepared.correlatedStates, a.prepared.correlatedResults = states, results
	return nil
}
