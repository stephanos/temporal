package verification

import (
	"context"
	"maps"
	"slices"
	"strconv"
	"unicode/utf8"

	celpb "cel.dev/expr"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/proto"
)

type admittedCorrelatedEvidence struct {
	*testpilotspb.CorrelatedEvidence
	supportingEventSequences []int64
}

type correlatedObligation struct {
	remaining int64
	status    testpilotspb.RuleVerdictStatus
	// violatedBy is the evidence whose release resolved the obligation violated, kept so that a
	// replay can name it: the releasing event may be later than the event that carried it.
	violatedBy *admittedCorrelatedEvidence
}

// retainedCapture is one occurrence a declared capture kept. Entries are only appended, so an
// ordinal already recorded keeps the value it was admitted with rather than being replaced by a
// later match.
type retainedCapture struct {
	capture string
	ordinal int64
	value   *celpb.Value
}

type correlatedOperation struct {
	state       string
	last        *testpilotspb.CorrelatedIdentity
	obligations [][]correlatedObligation
	captures    []retainedCapture
}
type correlatedMonitor struct {
	prepared *PreparedContract
	// limits is the Profile's correlated ceiling snapshot, shared read-only by every clone.
	limits                                                                        *testpilotspb.CorrelatedLimits
	scope                                                                         []*testpilotspb.NamedValue
	accepted                                                                      []*admittedCorrelatedEvidence
	processed                                                                     []*testpilotspb.CorrelatedIdentity
	support                                                                       [][]*testpilotspb.CorrelatedIdentity
	operations                                                                    map[string]*correlatedOperation
	ruleSupport                                                                   [][]int64
	transitions, obligations, projectionWork, obligationWork, retainedStepSupport int64
	capturedValues                                                                int64
}

func newCorrelated(p *PreparedContract) *correlatedMonitor {
	s := p.source.Correlated
	if s == nil {
		return nil
	}
	return &correlatedMonitor{prepared: p, limits: p.correlatedLimits, operations: map[string]*correlatedOperation{}, ruleSupport: make([][]int64, len(s.Rules))}
}
func (r *correlatedMonitor) clone() *correlatedMonitor {
	n := *r
	n.accepted = slices.Clone(r.accepted)
	n.processed = slices.Clone(r.processed)
	n.support = slices.Clone(r.support)
	n.operations = map[string]*correlatedOperation{}
	for k, v := range r.operations {
		op := *v
		op.obligations = make([][]correlatedObligation, len(v.obligations))
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
func identityEqual(a, b *testpilotspb.CorrelatedIdentity) bool { return proto.Equal(a, b) }
func identityIndex(ids []*testpilotspb.CorrelatedIdentity, id *testpilotspb.CorrelatedIdentity) int {
	return slices.IndexFunc(ids, func(x *testpilotspb.CorrelatedIdentity) bool { return identityEqual(x, id) })
}
func (r *correlatedMonitor) event(id *testpilotspb.CorrelatedIdentity) *admittedCorrelatedEvidence {
	i := slices.IndexFunc(r.accepted, func(e *admittedCorrelatedEvidence) bool { return identityEqual(e.Identity, id) })
	if i < 0 {
		return nil
	}
	return r.accepted[i]
}
func sourceBefore(a, b *testpilotspb.CorrelatedIdentity) bool {
	return a.EvidenceSource == b.EvidenceSource && a.Ordinal < b.Ordinal
}
func orderingEdge(a *testpilotspb.CorrelatedIdentity, b *admittedCorrelatedEvidence) bool {
	return identityIndex(b.Parents, a) >= 0 || sourceBefore(a, b.Identity)
}
func (r *correlatedMonitor) reaches(before, after *testpilotspb.CorrelatedIdentity) bool {
	visited := []*testpilotspb.CorrelatedIdentity{before}
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
func (r *correlatedMonitor) validateGraph() error {
	for _, e := range r.accepted {
		for _, id := range e.Parents {
			if p := r.event(id); p != nil && p.Operation != e.Operation {
				return invalid(ir.Malformed, "cross-operation causal parent")
			}
		}
	}
	removed := []*testpilotspb.CorrelatedIdentity{}
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
func (r *correlatedMonitor) ready(e *admittedCorrelatedEvidence) bool {
	for _, id := range e.Parents {
		if identityIndex(r.processed, id) < 0 {
			return false
		}
	}
	return e.Identity.Ordinal == 0 || slices.ContainsFunc(r.processed, func(id *testpilotspb.CorrelatedIdentity) bool {
		return id.EvidenceSource == e.Identity.EvidenceSource && id.Ordinal+1 == e.Identity.Ordinal
	})
}
func identitySize(id *testpilotspb.CorrelatedIdentity) int64 {
	n := int64(utf8.RuneCountInString(id.EvidenceSource) + 1)
	for _, b := range id.Scope {
		n += int64(utf8.RuneCountInString(b.FieldId) + utf8.RuneCountInString(b.GetValue().GetStringValue()))
	}
	return n
}
func evidenceSize(e *admittedCorrelatedEvidence) int64 {
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
			switch v := f.Value.Kind.(type) {
			case *celpb.Value_StringValue:
				n += int64(utf8.RuneCountInString(v.StringValue))
			case *celpb.Value_Uint64Value:
				n += int64(len(strconv.FormatUint(v.Uint64Value, 10)))
			case *celpb.Value_BoolValue:
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
func projectionRule(s *testpilotspb.CorrelatedContract, kind string) *testpilotspb.CorrelatedProjectionRule {
	for _, r := range s.ProjectionRules {
		if r.Kind == kind {
			return r
		}
	}
	return nil
}
func (r *correlatedMonitor) validate(s *testpilotspb.CorrelatedContract, e *admittedCorrelatedEvidence, sequence int64) error {
	if err := ir.CheckSurface(e.CorrelatedEvidence, ir.DefaultLimits()); err != nil {
		return err
	}
	if e.Identity == nil || e.Operation == "" || evidenceSize(e) > r.limits.MaxEventBytes {
		return invalid(ir.Malformed, "invalid correlated evidence size or identity")
	}
	if len(e.Identity.Scope) != len(s.ScopeFields) {
		return invalid(ir.Malformed, "wrong correlated bindings")
	}
	for i, b := range e.Identity.Scope {
		if b.FieldId != s.ScopeFields[i] || b.GetValue().GetStringValue() == "" {
			return invalid(ir.Malformed, "wrong correlated bindings")
		}
	}
	if r.scope != nil && !slices.EqualFunc(r.scope, e.Identity.Scope, func(a, b *testpilotspb.NamedValue) bool { return proto.Equal(a, b) }) {
		return invalid(ir.Malformed, "changed correlated bindings")
	}
	for _, id := range append(slices.Clone(e.Parents), e.Identity) {
		if id == nil || !slices.EqualFunc(id.Scope, e.Identity.Scope, func(a, b *testpilotspb.NamedValue) bool { return proto.Equal(a, b) }) || !slices.Contains(s.Sources, id.EvidenceSource) || id.Ordinal < 0 || id.Ordinal >= r.limits.MaxEvents {
			return invalid(ir.Malformed, "invalid correlated source identity")
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
		return invalid(ir.Unknown, "unsupported correlated evidence kind")
	}
	seen := map[string]bool{}
	for _, f := range e.Fields {
		i := slices.IndexFunc(rule.Fields, func(p *testpilotspb.CorrelatedFieldPolicy) bool { return p.FieldId == f.FieldId })
		if i < 0 || seen[f.FieldId] {
			return invalid(ir.Malformed, "unauthorized evidence field")
		}
		seen[f.FieldId] = true
		p := rule.Fields[i]
		if p.Disposition == testpilotspb.CORRELATED_FIELD_DISPOSITION_REJECT || p.Disposition == testpilotspb.CORRELATED_FIELD_DISPOSITION_REDACT && f.Value != nil || p.Disposition == testpilotspb.CORRELATED_FIELD_DISPOSITION_RETAIN && f.Value == nil {
			return invalid(ir.Malformed, "field disposition mismatch")
		}
		if f.Value != nil {
			ok := false
			switch f.Value.Kind.(type) {
			case *celpb.Value_StringValue:
				ok = p.GetType().GetKind() == testpilotspb.SCALAR_KIND_TEXT
			case *celpb.Value_BoolValue:
				ok = p.GetType().GetKind() == testpilotspb.SCALAR_KIND_BOOLEAN
			case *celpb.Value_Uint64Value:
				ok = p.GetType().GetKind() == testpilotspb.SCALAR_KIND_UINT64
			default:
				return invalid(ir.TypeMismatch, "unsupported correlated scalar")
			}
			if !ok {
				return invalid(ir.TypeMismatch, "correlated field type mismatch")
			}
		}
	}
	for _, f := range rule.Fields {
		if f.Disposition != testpilotspb.CORRELATED_FIELD_DISPOSITION_REJECT && !seen[f.FieldId] {
			return invalid(ir.Malformed, "missing declared evidence field")
		}
	}
	return nil
}
func appendIdentity(ids []*testpilotspb.CorrelatedIdentity, id *testpilotspb.CorrelatedIdentity) []*testpilotspb.CorrelatedIdentity {
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
func (r *correlatedMonitor) sequences(ids []*testpilotspb.CorrelatedIdentity) []int64 {
	var out []int64
	for _, id := range ids {
		if e := r.event(id); e != nil {
			out = unionSequences(out, e.supportingEventSequences)
		}
	}
	return out
}

// stepValues supplies all values of a definition to native CEL membership and size operations.
func stepValues(ref ir.Reference, tr *testpilotspb.CorrelatedResult, state *testpilotspb.CorrelatedState) *celpb.Value {
	var values []*testpilotspb.ModelValue
	switch testpilotspb.CorrelatedStepField(ref.Field) {
	case testpilotspb.CORRELATED_STEP_FIELD_ACTION:
		values = []*testpilotspb.ModelValue{tr.Action}
	case testpilotspb.CORRELATED_STEP_FIELD_OUTCOME:
		values = []*testpilotspb.ModelValue{tr.Outcome}
	case testpilotspb.CORRELATED_STEP_FIELD_STATE:
		values = append([]*testpilotspb.ModelValue{state.Atom}, state.Fields...)
	case testpilotspb.CORRELATED_STEP_FIELD_FACT:
		values = tr.Facts
	default:
		return nil
	}
	list := &celpb.ListValue{}
	for _, value := range values {
		if value.DefinitionId == ref.ID {
			list.Values = append(list.Values, &celpb.Value{Kind: &celpb.Value_StringValue{StringValue: value.Value}})
		}
	}
	return &celpb.Value{Kind: &celpb.Value_ListValue{ListValue: list}}
}
func evidenceField(e *admittedCorrelatedEvidence, id string) *celpb.Value {
	i := slices.IndexFunc(e.Fields, func(f *testpilotspb.NamedValue) bool { return f.FieldId == id })
	if i < 0 {
		return nil
	}
	return e.Fields[i].Value
}

func (r *correlatedMonitor) evaluate(ctx context.Context, expression *ir.Expression, out *testpilotspb.CorrelatedResult, e *admittedCorrelatedEvidence, op *correlatedOperation) (bool, error) {
	resolve := func(ref ir.Reference) *celpb.Value {
		switch ref.Kind {
		case ir.CorrelatedStepReference:
			return stepValues(ref, out, r.prepared.correlatedStates[out.StateId])
		case ir.EvidenceFieldReference:
			return evidenceField(e, ref.ID)
		case ir.CorrelatedCaptureReference:
			for _, entry := range op.captures {
				if entry.capture == ref.ID && entry.ordinal == ref.Ordinal {
					return entry.value
				}
			}
		}
		return nil
	}
	value, work, err := expression.Evaluate(ctx, resolve, r.limits.MaxObligationWork-r.obligationWork)
	if err != nil {
		return false, err
	}
	if err := add(&r.obligationWork, work, r.limits.MaxObligationWork); err != nil {
		return false, err
	}
	return value.GetBoolValue(), nil
}

// retain keeps this step's occurrence of every declared capture. A step supplying no value at the
// declared field records nothing; an operation already holding its declared lifetime rejects.
func (r *correlatedMonitor) retain(s *testpilotspb.CorrelatedContract, e *admittedCorrelatedEvidence, op *correlatedOperation) error {
	for _, c := range s.Rules {
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
			if err := add(&r.capturedValues, 1, r.limits.MaxCaptures); err != nil {
				return err
			}
			op.captures = append(op.captures, retainedCapture{capture: d.CaptureId, ordinal: ordinal, value: value})
		}
	}
	return nil
}

func (r *correlatedMonitor) release(ctx context.Context, s *testpilotspb.CorrelatedContract, e *admittedCorrelatedEvidence) error {
	rule := projectionRule(s, e.Kind)
	var support []*testpilotspb.CorrelatedIdentity
	for _, p := range e.Parents {
		i := identityIndex(r.processed, p)
		for _, ancestor := range r.support[i] {
			support = appendIdentity(support, ancestor)
		}
	}
	support = appendIdentity(support, e.Identity)
	r.processed = append(r.processed, e.Identity)
	r.support = append(r.support, support)
	if rule.Meaning != testpilotspb.CORRELATED_EVIDENCE_MEANING_CONFIRMED {
		return nil
	}
	operationCount := int64(len(r.operations))
	op := r.operations[e.Operation]
	if op == nil {
		op = &correlatedOperation{state: s.InitialStateId, obligations: make([][]correlatedObligation, len(s.Rules))}
		r.operations[e.Operation] = op
	}
	if op.last != nil && !r.reaches(op.last, e.Identity) {
		return invalid(ir.Malformed, "incomparable operation transitions")
	}
	if rule.Submission != nil && !slices.ContainsFunc(e.Parents, func(id *testpilotspb.CorrelatedIdentity) bool {
		p := r.event(id)
		return p != nil && projectionRule(s, p.Kind).Meaning == testpilotspb.CORRELATED_EVIDENCE_MEANING_SUBMISSION && proto.Equal(projectionRule(s, p.Kind).Submission, rule.Submission)
	}) {
		return invalid(ir.Malformed, "missing causal submission")
	}
	for _, resultID := range rule.ResultIds {
		out := r.prepared.correlatedResults[resultID]
		direct := appendIdentity(slices.Clone(e.Parents), e.Identity)
		r.retainedStepSupport += int64(len(direct) + len(support) + len(r.sequences(direct)) + len(r.sequences(support)))
		if !slices.ContainsFunc(s.Transitions, func(tr *testpilotspb.CorrelatedTransition) bool {
			return op.state == tr.PriorStateId && tr.ResultId == resultID
		}) {
			return invalid(ir.Malformed, "unauthorized operation transition")
		}
		// A declared correlation decides which authorized steps are this operation's semantic steps
		// at all. It reads this step's own evidence together with what the operation already
		// retained, so an occurrence binds only after an earlier step admitted it.
		for i, c := range s.Rules {
			if c.Correlation == nil {
				continue
			}
			holds, err := r.evaluate(ctx, r.prepared.correlatedRules[i].correlation, out, e, op)
			if err != nil {
				return err
			}
			if !holds {
				return invalid(ir.Malformed, "correlation rejected this operation's step")
			}
		}
		if r.transitions >= r.limits.MaxSemanticTransitions {
			return invalid(ir.LimitExceeded, "semantic transition ceiling exceeded")
		}
		maximumFacts, candidates := int64(0), int64(0)
		for _, row := range s.Transitions {
			result := r.prepared.correlatedResults[row.ResultId]
			maximumFacts = max(maximumFacts, int64(len(result.Facts)))
			if op.state == row.PriorStateId && proto.Equal(result.Action, out.Action) {
				candidates++
			}
		}
		cost := int64(16)
		for _, factor := range []int64{int64(len(s.Rules)), r.transitions + 1, 1 + maximumFacts} {
			if cost > r.limits.MaxObligationWork/factor {
				return invalid(ir.LimitExceeded, "obligation work exhausted")
			}
			cost *= factor
		}
		for _, extra := range []int64{r.obligations, operationCount, candidates} {
			if err := add(&cost, extra, r.limits.MaxObligationWork); err != nil {
				return err
			}
		}
		if err := add(&r.obligationWork, cost, r.limits.MaxObligationWork); err != nil {
			return err
		}
		for i, c := range s.Rules {
			triggered, err := r.evaluate(ctx, r.prepared.correlatedRules[i].trigger, out, e, op)
			if err != nil {
				return err
			}
			responded, err := r.evaluate(ctx, r.prepared.correlatedRules[i].response, out, e, op)
			if err != nil {
				return err
			}
			if triggered {
				if err := add(&r.obligations, 1, r.limits.MaxObligations); err != nil {
					return err
				}
				op.obligations[i] = append(op.obligations[i], correlatedObligation{remaining: c.Bound, status: testpilotspb.RULE_VERDICT_STATUS_PENDING})
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
					o.violatedBy = e
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
		op.state = out.StateId
		operationCount = int64(len(r.operations))
		r.transitions++
	}
	op.last = e.Identity
	return nil
}
func (r *correlatedMonitor) stage(ctx context.Context, s *testpilotspb.CorrelatedContract, e *admittedCorrelatedEvidence, sequence int64) (*correlatedMonitor, int64, error) {
	if err := r.validate(s, e, sequence); err != nil {
		return nil, 0, err
	}
	previous := r.event(e.Identity)
	if previous != nil && !proto.Equal(previous.CorrelatedEvidence, e.CorrelatedEvidence) {
		return nil, 0, invalid(ir.Malformed, "conflicting source identity")
	}
	if previous == nil && (len(e.supportingEventSequences) != 1 || e.supportingEventSequences[0] != sequence) {
		return nil, 0, invalid(ir.Malformed, "new evidence must name its actual emitting Run Event")
	}
	n := r.clone()
	if previous == nil {
		n.accepted = append(n.accepted, &admittedCorrelatedEvidence{CorrelatedEvidence: proto.CloneOf(e.CorrelatedEvidence), supportingEventSequences: slices.Clone(e.supportingEventSequences)})
	}
	if n.scope == nil {
		n.scope = proto.CloneOf(e.Identity).Scope
	}
	l := r.limits
	if int64(len(n.accepted)) > l.MaxEvents {
		return nil, 0, invalid(ir.LimitExceeded, "correlated events exhausted")
	}
	keys := map[string]bool{}
	size := int64(1)
	for _, event := range n.accepted {
		keys[event.Operation] = true
		size += evidenceSize(event)
	}
	if int64(len(keys)) > l.MaxKeys {
		return nil, 0, invalid(ir.LimitExceeded, "correlated keys exhausted")
	}
	outputs := int64(len(s.Transitions) + 1)
	for _, rule := range s.ProjectionRules {
		outputs += int64(len(rule.Fields) + max(1, len(rule.ResultIds)))
		for _, id := range rule.ResultIds {
			result := r.prepared.correlatedResults[id]
			if err := add(&outputs, int64(proto.Size(result))+int64(proto.Size(r.prepared.correlatedStates[result.StateId])), l.MaxProjectionWork); err != nil {
				return nil, 0, err
			}
		}
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
	slices.SortFunc(ordered, func(a, b *admittedCorrelatedEvidence) int {
		if a.Identity.EvidenceSource < b.Identity.EvidenceSource {
			return -1
		}
		if a.Identity.EvidenceSource > b.Identity.EvidenceSource {
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
				if err := n.release(ctx, s, event); err != nil {
					return nil, 0, err
				}
			}
		}
	}
	if int64(len(n.accepted)-len(n.processed)) > l.MaxBuffered {
		return nil, 0, invalid(ir.LimitExceeded, "correlated buffer exhausted")
	}
	retained := n.retainedStepSupport
	for _, ids := range n.support {
		retained += int64(len(ids))
	}
	if retained > l.MaxSupport {
		return nil, 0, invalid(ir.LimitExceeded, "correlated support exhausted")
	}
	return n, work + n.obligationWork - r.obligationWork, nil
}

// violation is the evidence that resolved the rule's first violated obligation, operations walked
// in key order so the choice is the same on every reading, or nil when no obligation of the rule
// is violated by evidence: an obligation answer reports violated at closure stays pending here.
func (r *correlatedMonitor) violation(index int) *admittedCorrelatedEvidence {
	if r == nil {
		return nil
	}
	// Operations are keyed in a map; the first violated obligation must be the same one on every
	// reading, so the keys are walked in order.
	for _, key := range slices.Sorted(maps.Keys(r.operations)) {
		for _, o := range r.operations[key].obligations[index] {
			if o.status == testpilotspb.RULE_VERDICT_STATUS_VIOLATED {
				return o.violatedBy
			}
		}
	}
	return nil
}

func (r *correlatedMonitor) answer(s *testpilotspb.CorrelatedContract, index int, closed, incomplete bool) testpilotspb.RuleVerdictStatus {
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
		if closed && s.Rules[index].Ending == testpilotspb.TRACE_ENDING_FINAL {
			return testpilotspb.RULE_VERDICT_STATUS_VIOLATED
		}
		return testpilotspb.RULE_VERDICT_STATUS_INCONCLUSIVE
	}
	return testpilotspb.RULE_VERDICT_STATUS_SATISFIED
}
