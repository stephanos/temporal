package verification

import (
	"context"
	"slices"
	"strconv"
	"unicode/utf8"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/proto"
)

type admittedScopedEvidence struct {
	*testpilotspb.ScopedEvidence
	supportingEventSequences []int64
}

type scopedObligation struct {
	remaining int64
	status    testpilotspb.RuleVerdictStatus
}

// retainedCapture is one occurrence a declared capture kept. Entries are only appended, so an
// ordinal already recorded keeps the value it was admitted with rather than being replaced by a
// later match.
type retainedCapture struct {
	capture string
	ordinal int64
	value   *testpilotspb.Value
}
type scopedOperation struct {
	state       *testpilotspb.ScopedValue
	last        *testpilotspb.ScopedIdentity
	obligations [][]scopedObligation
	captures    []retainedCapture
}
type scopedRun struct {
	scope                                                                         []*testpilotspb.ScopedBinding
	accepted                                                                      []*admittedScopedEvidence
	processed                                                                     []*testpilotspb.ScopedIdentity
	support                                                                       [][]*testpilotspb.ScopedIdentity
	operations                                                                    map[string]*scopedOperation
	ruleSupport                                                                   [][]int64
	transitions, obligations, projectionWork, obligationWork, retainedStepSupport int64
	capturedValues                                                                int64
}

func newScoped(s *testpilotspb.ScopedContract) *scopedRun {
	if s == nil {
		return nil
	}
	return &scopedRun{operations: map[string]*scopedOperation{}, ruleSupport: make([][]int64, len(s.Clauses))}
}
func (r *scopedRun) clone() *scopedRun {
	n := *r
	n.accepted = slices.Clone(r.accepted)
	n.processed = slices.Clone(r.processed)
	n.support = slices.Clone(r.support)
	n.operations = map[string]*scopedOperation{}
	for k, v := range r.operations {
		op := *v
		op.obligations = make([][]scopedObligation, len(v.obligations))
		for i, obs := range v.obligations {
			op.obligations[i] = slices.Clone(obs)
		}
		op.captures = slices.Clone(v.captures)
		n.operations[k] = &op
	}
	n.ruleSupport = make([][]int64, len(r.ruleSupport))
	for i, v := range r.ruleSupport {
		n.ruleSupport[i] = slices.Clone(v)
	}
	return &n
}
func identityEqual(a, b *testpilotspb.ScopedIdentity) bool { return proto.Equal(a, b) }
func identityIndex(ids []*testpilotspb.ScopedIdentity, id *testpilotspb.ScopedIdentity) int {
	return slices.IndexFunc(ids, func(x *testpilotspb.ScopedIdentity) bool { return identityEqual(x, id) })
}
func (r *scopedRun) event(id *testpilotspb.ScopedIdentity) *admittedScopedEvidence {
	i := slices.IndexFunc(r.accepted, func(e *admittedScopedEvidence) bool { return identityEqual(e.Identity, id) })
	if i < 0 {
		return nil
	}
	return r.accepted[i]
}
func sourceBefore(a, b *testpilotspb.ScopedIdentity) bool {
	return a.Source == b.Source && a.Ordinal < b.Ordinal
}
func orderingEdge(a *testpilotspb.ScopedIdentity, b *admittedScopedEvidence) bool {
	return identityIndex(b.Parents, a) >= 0 || sourceBefore(a, b.Identity)
}
func (r *scopedRun) reaches(before, after *testpilotspb.ScopedIdentity) bool {
	visited := []*testpilotspb.ScopedIdentity{before}
	for i := 0; i < len(visited); i++ {
		if identityEqual(visited[i], after) {
			return true
		}
		for _, e := range r.accepted {
			if identityIndex(visited, e.Identity) < 0 && orderingEdge(visited[i], e) {
				visited = append(visited, e.Identity)
			}
		}
	}
	return false
}
func (r *scopedRun) validateGraph() error {
	for _, e := range r.accepted {
		for _, id := range e.Parents {
			if p := r.event(id); p != nil && p.Operation != e.Operation {
				return invalid(ir.Malformed, "cross-operation causal parent")
			}
		}
	}
	removed := []*testpilotspb.ScopedIdentity{}
	for range r.accepted {
		for _, e := range r.accepted {
			if identityIndex(removed, e.Identity) >= 0 {
				continue
			}
			ready := true
			for _, p := range r.accepted {
				if orderingEdge(p.Identity, e) && identityIndex(removed, p.Identity) < 0 {
					ready = false
					break
				}
			}
			if ready {
				removed = append(removed, e.Identity)
			}
		}
	}
	if len(removed) != len(r.accepted) {
		return invalid(ir.Malformed, "causal/source-order cycle")
	}
	return nil
}
func (r *scopedRun) ready(e *admittedScopedEvidence) bool {
	for _, id := range e.Parents {
		if identityIndex(r.processed, id) < 0 {
			return false
		}
	}
	return e.Identity.Ordinal == 0 || slices.ContainsFunc(r.processed, func(id *testpilotspb.ScopedIdentity) bool {
		return id.Source == e.Identity.Source && id.Ordinal+1 == e.Identity.Ordinal
	})
}
func identitySize(id *testpilotspb.ScopedIdentity) int64 {
	n := int64(utf8.RuneCountInString(id.Source) + 1)
	for _, b := range id.Scope {
		n += int64(utf8.RuneCountInString(b.FieldId) + utf8.RuneCountInString(b.Value))
	}
	return n
}
func evidenceSize(e *admittedScopedEvidence) int64 {
	n := identitySize(e.Identity) + int64(utf8.RuneCountInString(e.Operation)+utf8.RuneCountInString(e.Kind))
	for _, seq := range e.supportingEventSequences {
		n += int64(len(strconv.FormatInt(seq, 10)))
	}
	for _, id := range e.Parents {
		n += identitySize(id)
	}
	for _, f := range e.Fields {
		n += int64(utf8.RuneCountInString(f.FieldId))
		if f.Value == nil {
			n++
		} else {
			switch v := f.Value.Value.(type) {
			case *testpilotspb.Value_Text:
				n += int64(utf8.RuneCountInString(v.Text))
			case *testpilotspb.Value_Natural:
				n += int64(len(v.Natural))
			case *testpilotspb.Value_BoolValue:
				if v.BoolValue {
					n += 4
				} else {
					n += 5
				}
			default:
				n += int64(proto.Size(f.Value))
			}
		}
	}
	return n
}
func projectionRule(s *testpilotspb.ScopedContract, kind string) *testpilotspb.ScopedProjectionRule {
	for _, r := range s.ProjectionRules {
		if r.Kind == kind {
			return r
		}
	}
	return nil
}
func (r *scopedRun) validate(s *testpilotspb.ScopedContract, e *admittedScopedEvidence, sequence int64) error {
	if e.Identity == nil || e.Operation == "" || evidenceSize(e) > s.Limits.MaxEventBytes {
		return invalid(ir.Malformed, "invalid scoped evidence size or identity")
	}
	if len(e.Identity.Scope) != len(s.ScopeFields) {
		return invalid(ir.Malformed, "wrong scoped bindings")
	}
	for i, b := range e.Identity.Scope {
		if b.FieldId != s.ScopeFields[i] || b.Value == "" {
			return invalid(ir.Malformed, "wrong scoped bindings")
		}
	}
	if r.scope != nil && !slices.EqualFunc(r.scope, e.Identity.Scope, func(a, b *testpilotspb.ScopedBinding) bool { return proto.Equal(a, b) }) {
		return invalid(ir.Malformed, "changed scoped bindings")
	}
	for _, id := range append(slices.Clone(e.Parents), e.Identity) {
		if id == nil || !slices.EqualFunc(id.Scope, e.Identity.Scope, func(a, b *testpilotspb.ScopedBinding) bool { return proto.Equal(a, b) }) || !slices.Contains(s.Sources, id.Source) || id.Ordinal < 0 || id.Ordinal >= s.Limits.MaxEvents {
			return invalid(ir.Malformed, "invalid scoped source identity")
		}
	}
	for i, p := range e.Parents {
		if identityIndex(e.Parents[:i], p) >= 0 {
			return invalid(ir.Malformed, "duplicate causal parent")
		}
	}
	if len(e.supportingEventSequences) == 0 {
		return invalid(ir.Malformed, "missing evidence support")
	}
	for i, seq := range e.supportingEventSequences {
		if seq <= 0 || seq > sequence || slices.Contains(e.supportingEventSequences[:i], seq) {
			return invalid(ir.Malformed, "invalid evidence support")
		}
	}
	rule := projectionRule(s, e.Kind)
	if rule == nil {
		return invalid(ir.Unknown, "unsupported scoped evidence kind")
	}
	seen := map[string]bool{}
	for _, f := range e.Fields {
		i := slices.IndexFunc(rule.Fields, func(p *testpilotspb.ScopedFieldPolicy) bool { return p.FieldId == f.FieldId })
		if i < 0 || seen[f.FieldId] {
			return invalid(ir.Malformed, "unauthorized evidence field")
		}
		seen[f.FieldId] = true
		p := rule.Fields[i]
		if p.Disposition == testpilotspb.SCOPED_FIELD_DISPOSITION_REJECT || p.Disposition == testpilotspb.SCOPED_FIELD_DISPOSITION_REDACT && f.Value != nil || p.Disposition == testpilotspb.SCOPED_FIELD_DISPOSITION_RETAIN && f.Value == nil {
			return invalid(ir.Malformed, "field disposition mismatch")
		}
		if f.Value != nil {
			ok := false
			switch v := f.Value.Value.(type) {
			case *testpilotspb.Value_Text:
				ok = p.GetType().GetKind() == testpilotspb.SCALAR_KIND_TEXT
			case *testpilotspb.Value_BoolValue:
				ok = p.GetType().GetKind() == testpilotspb.SCALAR_KIND_BOOLEAN
			case *testpilotspb.Value_Natural:
				ok = p.GetType().GetKind() == testpilotspb.SCALAR_KIND_NATURAL && (v.Natural == "0" || len(v.Natural) > 0 && v.Natural[0] >= '1' && v.Natural[0] <= '9')
				for _, c := range v.Natural {
					ok = ok && c >= '0' && c <= '9'
				}
			default:
				return invalid(ir.TypeMismatch, "unsupported scoped scalar")
			}
			if !ok {
				return invalid(ir.TypeMismatch, "scoped field type mismatch")
			}
		}
	}
	for _, f := range rule.Fields {
		if f.Disposition != testpilotspb.SCOPED_FIELD_DISPOSITION_REJECT && !seen[f.FieldId] {
			return invalid(ir.Malformed, "missing declared evidence field")
		}
	}
	return nil
}
func appendIdentity(ids []*testpilotspb.ScopedIdentity, id *testpilotspb.ScopedIdentity) []*testpilotspb.ScopedIdentity {
	if identityIndex(ids, id) < 0 {
		return append(ids, id)
	}
	return ids
}
func unionSequences(a, b []int64) []int64 {
	for _, v := range b {
		if !slices.Contains(a, v) {
			a = append(a, v)
		}
	}
	slices.Sort(a)
	return a
}
func (r *scopedRun) sequences(ids []*testpilotspb.ScopedIdentity) []int64 {
	var out []int64
	for _, id := range ids {
		if e := r.event(id); e != nil {
			out = unionSequences(out, e.supportingEventSequences)
		}
	}
	return out
}
func predicate(p *testpilotspb.ScopedPredicate, tr *testpilotspb.ScopedTransition) bool {
	var values []*testpilotspb.ScopedValue
	switch p.Field {
	case testpilotspb.SCOPED_PREDICATE_FIELD_ACTION:
		values = []*testpilotspb.ScopedValue{tr.Action}
	case testpilotspb.SCOPED_PREDICATE_FIELD_OUTCOME:
		values = []*testpilotspb.ScopedValue{tr.Outcome}
	case testpilotspb.SCOPED_PREDICATE_FIELD_RESULTING_STATE:
		values = []*testpilotspb.ScopedValue{tr.ResultingState}
	case testpilotspb.SCOPED_PREDICATE_FIELD_FACT:
		values = tr.Facts
	default:
		return false
	}
	return slices.ContainsFunc(values, func(v *testpilotspb.ScopedValue) bool {
		if v.DefinitionId != p.DefinitionId {
			return false
		}
		switch c := p.Constraint.(type) {
		case *testpilotspb.ScopedPredicate_Present:
			return c.Present
		case *testpilotspb.ScopedPredicate_EqualsText:
			return v.Value == c.EqualsText
		default:
			return false
		}
	})
}
func evidenceField(e *admittedScopedEvidence, id string) *testpilotspb.Value {
	i := slices.IndexFunc(e.Fields, func(f *testpilotspb.ScopedEvidenceField) bool { return f.FieldId == id })
	if i < 0 {
		return nil
	}
	return e.Fields[i].Value
}

// operandValue reads only declared evidence. An occurrence the operation never retained -- a future
// ordinal, or one belonging to a different operation -- has no value here, so admission fails rather
// than binding the nearest match.
func operandValue(o *testpilotspb.ScopedOperand, e *admittedScopedEvidence, op *scopedOperation) (*testpilotspb.Value, error) {
	switch v := o.GetOperand().(type) {
	case *testpilotspb.ScopedOperand_Literal:
		return v.Literal, nil
	case *testpilotspb.ScopedOperand_FieldId:
		if value := evidenceField(e, v.FieldId); value != nil {
			return value, nil
		}
		return nil, invalid(ir.Malformed, "missing correlation field operand")
	case *testpilotspb.ScopedOperand_Capture:
		for _, entry := range op.captures {
			if entry.capture == v.Capture.GetCaptureId() && entry.ordinal == v.Capture.GetOrdinal() {
				return entry.value, nil
			}
		}
		return nil, invalid(ir.Malformed, "missing retained capture occurrence")
	default:
		return nil, invalid(ir.Unknown, "unsupported correlation operand")
	}
}

// correlationHolds evaluates groups left to right and stops at the first decisive operand, so an
// operand an earlier one made irrelevant is never read and cannot fail admission.
func correlationHolds(c *testpilotspb.ScopedCorrelation, out *testpilotspb.ScopedTransition, e *admittedScopedEvidence, op *scopedOperation) (bool, error) {
	switch v := c.GetCondition().(type) {
	case *testpilotspb.ScopedCorrelation_Predicate:
		return predicate(v.Predicate, out), nil
	case *testpilotspb.ScopedCorrelation_Comparison:
		left, err := operandValue(v.Comparison.GetLeft(), e, op)
		if err != nil {
			return false, err
		}
		right, err := operandValue(v.Comparison.GetRight(), e, op)
		if err != nil {
			return false, err
		}
		return proto.Equal(left, right) == (v.Comparison.GetOperator() == testpilotspb.SCOPED_COMPARISON_OPERATOR_EQUAL), nil
	case *testpilotspb.ScopedCorrelation_All:
		for _, operand := range v.All.GetOperands() {
			holds, err := correlationHolds(operand, out, e, op)
			if err != nil || !holds {
				return false, err
			}
		}
		return true, nil
	case *testpilotspb.ScopedCorrelation_Any:
		for _, operand := range v.Any.GetOperands() {
			holds, err := correlationHolds(operand, out, e, op)
			if err != nil {
				return false, err
			}
			if holds {
				return true, nil
			}
		}
		return false, nil
	default:
		return false, invalid(ir.Unknown, "unsupported correlation condition")
	}
}

// retain keeps this step's occurrence of every declared capture. A step supplying no value at the
// declared field records nothing; an operation already holding its declared lifetime rejects.
func (r *scopedRun) retain(s *testpilotspb.ScopedContract, e *admittedScopedEvidence, op *scopedOperation) error {
	for _, c := range s.Clauses {
		for _, d := range c.Captures {
			value := evidenceField(e, d.FieldId)
			if value == nil {
				continue
			}
			ordinal := int64(0)
			for _, entry := range op.captures {
				if entry.capture == d.CaptureId {
					ordinal++
				}
			}
			if ordinal >= d.Lifetime {
				return invalid(ir.LimitExceeded, "capture lifetime exhausted")
			}
			if err := add(&r.capturedValues, 1, s.Limits.MaxCaptures); err != nil {
				return err
			}
			op.captures = append(op.captures, retainedCapture{capture: d.CaptureId, ordinal: ordinal, value: value})
		}
	}
	return nil
}

func (r *scopedRun) release(s *testpilotspb.ScopedContract, e *admittedScopedEvidence) error {
	rule := projectionRule(s, e.Kind)
	var support []*testpilotspb.ScopedIdentity
	for _, p := range e.Parents {
		i := identityIndex(r.processed, p)
		for _, ancestor := range r.support[i] {
			support = appendIdentity(support, ancestor)
		}
	}
	support = appendIdentity(support, e.Identity)
	r.processed = append(r.processed, e.Identity)
	r.support = append(r.support, support)
	if rule.Meaning != testpilotspb.SCOPED_EVIDENCE_MEANING_CONFIRMED {
		return nil
	}
	operationCount := int64(len(r.operations))
	op := r.operations[e.Operation]
	if op == nil {
		op = &scopedOperation{state: s.InitialState, obligations: make([][]scopedObligation, len(s.Clauses))}
		r.operations[e.Operation] = op
	}
	if op.last != nil && !r.reaches(op.last, e.Identity) {
		return invalid(ir.Malformed, "incomparable operation transitions")
	}
	if rule.Submission != nil && !slices.ContainsFunc(e.Parents, func(id *testpilotspb.ScopedIdentity) bool {
		p := r.event(id)
		return p != nil && projectionRule(s, p.Kind).Meaning == testpilotspb.SCOPED_EVIDENCE_MEANING_SUBMISSION && proto.Equal(projectionRule(s, p.Kind).Submission, rule.Submission)
	}) {
		return invalid(ir.Malformed, "missing causal submission")
	}
	for _, out := range rule.Outputs {
		direct := appendIdentity(slices.Clone(e.Parents), e.Identity)
		r.retainedStepSupport += int64(len(direct) + len(support) + len(r.sequences(direct)) + len(r.sequences(support)))
		if !slices.ContainsFunc(s.Transitions, func(tr *testpilotspb.ScopedTransition) bool {
			return proto.Equal(tr.PriorState, op.state) && sameResult(tr, out)
		}) {
			return invalid(ir.Malformed, "unauthorized operation transition")
		}
		// A declared correlation decides which authorized steps are this operation's semantic steps
		// at all. It reads this step's own evidence together with what the operation already
		// retained, so an occurrence binds only after an earlier step admitted it.
		for _, c := range s.Clauses {
			if c.Correlation == nil {
				continue
			}
			holds, err := correlationHolds(c.Correlation, out, e, op)
			if err != nil {
				return err
			}
			if !holds {
				return invalid(ir.Malformed, "correlation rejected this operation's step")
			}
		}
		if r.transitions >= s.Limits.MaxSemanticTransitions {
			return invalid(ir.LimitExceeded, "semantic transition ceiling exceeded")
		}
		maximumFacts, candidates := int64(0), int64(0)
		for _, row := range s.Transitions {
			maximumFacts = max(maximumFacts, int64(len(row.Facts)))
			if proto.Equal(row.PriorState, op.state) && proto.Equal(row.Action, out.Action) {
				candidates++
			}
		}
		cost := int64(16)
		for _, factor := range []int64{int64(len(s.Clauses)), r.transitions + 1, 1 + maximumFacts} {
			if cost > s.Limits.MaxObligationWork/factor {
				return invalid(ir.LimitExceeded, "obligation work exhausted")
			}
			cost *= factor
		}
		for _, extra := range []int64{r.obligations, operationCount, candidates} {
			if err := add(&cost, extra, s.Limits.MaxObligationWork); err != nil {
				return err
			}
		}
		if err := add(&r.obligationWork, cost, s.Limits.MaxObligationWork); err != nil {
			return err
		}
		for i, c := range s.Clauses {
			triggered, responded := predicate(c.Trigger, out), predicate(c.Response, out)
			if triggered {
				if err := add(&r.obligations, 1, s.Limits.MaxObligations); err != nil {
					return err
				}
				op.obligations[i] = append(op.obligations[i], scopedObligation{remaining: c.Bound, status: testpilotspb.RULE_VERDICT_STATUS_PENDING})
			}
			for j := range op.obligations[i] {
				o := &op.obligations[i][j]
				if o.status != testpilotspb.RULE_VERDICT_STATUS_PENDING {
					continue
				}
				if responded {
					o.status = testpilotspb.RULE_VERDICT_STATUS_SATISFIED
				} else if o.remaining == 0 {
					o.status = testpilotspb.RULE_VERDICT_STATUS_VIOLATED
				} else {
					o.remaining--
				}
			}
			r.ruleSupport[i] = unionSequences(r.ruleSupport[i], r.sequences(support))
		}
		// This step's own occurrences are retained only once its whole append was admitted, so no
		// correlation can read the occurrence its own step creates.
		if err := r.retain(s, e, op); err != nil {
			return err
		}
		op.state = out.ResultingState
		operationCount = int64(len(r.operations))
		r.transitions++
	}
	op.last = e.Identity
	return nil
}
func (r *scopedRun) stage(ctx context.Context, s *testpilotspb.ScopedContract, e *admittedScopedEvidence, sequence int64) (*scopedRun, int64, error) {
	if err := r.validate(s, e, sequence); err != nil {
		return nil, 0, err
	}
	previous := r.event(e.Identity)
	if previous != nil && !proto.Equal(previous.ScopedEvidence, e.ScopedEvidence) {
		return nil, 0, invalid(ir.Malformed, "conflicting source identity")
	}
	if previous == nil && (len(e.supportingEventSequences) != 1 || e.supportingEventSequences[0] != sequence) {
		return nil, 0, invalid(ir.Malformed, "new evidence must name its actual emitting Run Event")
	}
	n := r.clone()
	if previous == nil {
		n.accepted = append(n.accepted, &admittedScopedEvidence{ScopedEvidence: proto.CloneOf(e.ScopedEvidence), supportingEventSequences: slices.Clone(e.supportingEventSequences)})
	}
	if n.scope == nil {
		n.scope = proto.CloneOf(e.Identity).Scope
	}
	l := s.Limits
	if int64(len(n.accepted)) > l.MaxEvents {
		return nil, 0, invalid(ir.LimitExceeded, "scoped events exhausted")
	}
	keys := map[string]bool{}
	size := int64(1)
	for _, event := range n.accepted {
		keys[event.Operation] = true
		size += evidenceSize(event)
	}
	if int64(len(keys)) > l.MaxKeys {
		return nil, 0, invalid(ir.LimitExceeded, "scoped keys exhausted")
	}
	outputs := int64(len(s.Transitions) + 1)
	for _, rule := range s.ProjectionRules {
		outputs += int64(len(rule.Fields) + max(1, len(rule.Outputs)))
	}
	work := size
	for _, factor := range []int64{int64(len(n.accepted) + 1), int64(len(n.accepted) + 1), int64(len(n.accepted) + 1), outputs} {
		if work > (l.MaxProjectionWork-n.projectionWork)/factor {
			return nil, 0, invalid(ir.LimitExceeded, "projection work exhausted")
		}
		work *= factor
	}
	n.projectionWork += work
	if previous != nil {
		return n, work, nil
	}
	if err := n.validateGraph(); err != nil {
		return nil, 0, err
	}
	ordered := slices.Clone(n.accepted)
	slices.SortFunc(ordered, func(a, b *admittedScopedEvidence) int {
		if a.Identity.Source < b.Identity.Source {
			return -1
		}
		if a.Identity.Source > b.Identity.Source {
			return 1
		}
		if a.Identity.Ordinal < b.Identity.Ordinal {
			return -1
		}
		if a.Identity.Ordinal > b.Identity.Ordinal {
			return 1
		}
		return 0
	})
	for range n.accepted {
		for _, event := range ordered {
			if err := ctx.Err(); err != nil {
				return nil, 0, err
			}
			if identityIndex(n.processed, event.Identity) < 0 && n.ready(event) {
				if err := n.release(s, event); err != nil {
					return nil, 0, err
				}
			}
		}
	}
	if int64(len(n.accepted)-len(n.processed)) > l.MaxBuffered {
		return nil, 0, invalid(ir.LimitExceeded, "scoped buffer exhausted")
	}
	retained := n.retainedStepSupport
	for _, ids := range n.support {
		retained += int64(len(ids))
	}
	if retained > l.MaxSupport {
		return nil, 0, invalid(ir.LimitExceeded, "scoped support exhausted")
	}
	return n, work + n.obligationWork - r.obligationWork, nil
}
func (r *scopedRun) answer(s *testpilotspb.ScopedContract, index int, closed, incomplete bool) testpilotspb.RuleVerdictStatus {
	pending := false
	for _, op := range r.operations {
		for _, o := range op.obligations[index] {
			if o.status == testpilotspb.RULE_VERDICT_STATUS_VIOLATED {
				return o.status
			}
			pending = pending || o.status == testpilotspb.RULE_VERDICT_STATUS_PENDING
		}
	}
	// An empty evidence stream observed nothing. The model kernel reads an empty obligation list as
	// vacuous satisfaction because a model trace is total; a recorded evidence stream is not, so
	// silence resolves inconclusive here rather than manufacturing a satisfied answer.
	if incomplete || len(r.accepted) == 0 || len(r.accepted) != len(r.processed) {
		return testpilotspb.RULE_VERDICT_STATUS_INCONCLUSIVE
	}
	if pending {
		if closed && s.Clauses[index].Endpoint == testpilotspb.SCOPED_ENDPOINT_DELIBERATELY_CLOSED {
			return testpilotspb.RULE_VERDICT_STATUS_VIOLATED
		}
		return testpilotspb.RULE_VERDICT_STATUS_INCONCLUSIVE
	}
	return testpilotspb.RULE_VERDICT_STATUS_SATISFIED
}
