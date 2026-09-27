// Package verification prepares bounded Contracts over immutable Program observations.
package verification

import (
	"errors"
	"fmt"
	"iter"
	"maps"
	"slices"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/internal/execution"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type PreparedContract struct {
	source *testpilotspb.Contract
	// limits and correlatedLimits are the Profile's ceiling snapshots; a Contract declares none.
	limits           *testpilotspb.ContractLimits
	correlatedLimits *testpilotspb.CorrelatedLimits
	catalog          *ir.Catalog
	observations     map[string]ir.Type
	program          execution.ProgramView
	rules            []*machine
	workPerEvent     int64
}
type machine struct {
	source       *testpilotspb.ContractRule
	initial      int
	states       map[string]int
	captures     map[string]int
	captureTypes []ir.Type
	transitions  []*ir.Expression
	outgoing     []map[testpilotspb.RunEventKind][]int
	// instanceValues types each value the Rule's instances assign; a predicate reads one as always
	// available.
	instanceValues map[string]ir.Type
	// instances are the Rule's instances in declaration order; a plain Rule has none.
	instances []ruleInstance
}

// ruleInstance is one evaluation of a Rule, concluding under its own rule ID.
type ruleInstance struct {
	ruleID string
	values map[string]*testpilotspb.Value
	// work is, per instance value, the binding work its inlined literal costs over its reference.
	work map[string]int64
}

// readWork is what the instance value reads cost this instance beyond their references' binding.
func (r *ruleInstance) readWork(reads map[string]int64) int64 {
	var work int64
	for id, count := range reads {
		work += count * r.work[id]
	}
	return work
}

// ledgerEntry is admission charges a Rule's binding made to one counter, replayed per further instance.
type ledgerEntry struct {
	total          *int64
	value, ceiling int64
	reads          map[string]int64
}

// ledger records the admission charges a Rule's binding makes, in order.
type ledger []ledgerEntry

// record appends a charge. Consecutive charges to one counter merge: every value is nonnegative, so
// their sum exceeds the ceiling exactly when one of them would, with the same error.
func (l *ledger) record(total *int64, value, ceiling int64, reads map[string]int64) {
	if last := len(*l) - 1; last >= 0 && (*l)[last].total == total {
		entry := &(*l)[last]
		entry.value += value
		for id, n := range reads {
			if entry.reads == nil {
				entry.reads = map[string]int64{}
			}
			entry.reads[id] += n
		}
		return
	}
	*l = append(*l, ledgerEntry{total: total, value: value, ceiling: ceiling, reads: maps.Clone(reads)})
}

type admission struct {
	prepared                                    *PreparedContract
	catalog                                     *ir.Catalog
	scope                                       map[ir.Reference]ir.Binding
	boolean                                     ir.Type
	limits                                      ir.Limits
	work                                        int64
	states, transitions, captures, captureBytes int64
	// ruleCount counts Rule instances, a plain Rule as one, as the Contract's expansion counts rules.
	ruleCount int64
	// instance prices instance value reads while its Rule binds. While a Rule with instances binds,
	// recorded holds each charge, so every further instance is charged exactly as its expanded copy
	// would be.
	instance *ruleInstance
	recorded *ledger
}

func (p *PreparedContract) Snapshot() *testpilotspb.Contract   { return proto.CloneOf(p.source) }
func (p *PreparedContract) ProgramView() execution.ProgramView { return p.program }
func invalid(category ir.ErrorCategory, detail string) error {
	return &ir.Error{Category: category, Path: "contract", Detail: detail}
}

// invalidAt is invalid located at path, truncated to the bound every located path keeps.
func invalidAt(category ir.ErrorCategory, path, detail string) error {
	if len(path) > 256 {
		path = path[:256]
	}
	return &ir.Error{Category: category, Path: path, Detail: detail}
}
func validID(id string) bool {
	if len(id) == 0 || len(id) > 256 {
		return false
	}
	for _, c := range id {
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') && c != '_' && c != '-' && c != '.' {
			return false
		}
	}
	return true
}

// hardLimits is the Driver ceiling every Profile's Contract ceiling must fit under. MaxWorkPerEvent
// is sized for a correlated capability rather than for expression evaluation alone: an evidence event
// charges the correlated stage's conservative reservation into the same per-event bucket, and that
// reservation is cubic in the accepted evidence count, so the expression-evaluation ceiling it used
// to carry made every multi-operation correlated Case reject at its own first evidence event. The
// capability is still bounded by the Profile's max_projection_work and max_obligation_work, which
// admission bounds by the Profile's Contract total, and every Profile still declares its own smaller
// per-event value.
//
// CONSIDER(umpire): charge the reservation's per-event increment instead of recomputing the cube of
// the whole accepted set on every event, then restore a per-event ceiling that means expression
// evaluation again.
func hardLimits() *testpilotspb.ContractLimits {
	return &testpilotspb.ContractLimits{MaxRules: 10000, MaxStates: 10000, MaxTransitions: 10000, MaxExpressionDepth: 64, MaxWorkPerEvent: 100000000, MaxTotalWork: 1000000000000, MaxCaptures: 10000, MaxCaptureBytes: 16 << 20}
}
func checkLimits(limits, ceiling *testpilotspb.ContractLimits) error {
	if limits == nil || ceiling == nil {
		return invalid(ir.Malformed, "Contract limits and Driver ceilings are required")
	}
	if err := ir.CheckSurface(limits, ir.DefaultLimits()); err != nil {
		return err
	}
	fields := limits.ProtoReflect().Descriptor().Fields()
	for i := 0; i < fields.Len(); i++ {
		f := fields.Get(i)
		value := limits.ProtoReflect().Get(f).Int()
		if value <= 0 || value > ceiling.ProtoReflect().Get(f).Int() {
			return invalid(ir.LimitExceeded, "limit outside positive Driver ceiling: "+string(f.Name()))
		}
	}
	return nil
}
func add(total *int64, value, ceiling int64) error {
	if value < 0 || value > ceiling-*total {
		return invalid(ir.LimitExceeded, "count, byte, or work ceiling exceeded")
	}
	*total += value
	return nil
}
func (a *admission) charge(value int64) error {
	return a.count(&a.work, value, ir.DefaultLimits().Work, nil)
}

// count adds value to total under ceiling, with each instance value read priced at the current
// Rule instance's inlined literal, and records the charge while a Rule with instances binds.
func (a *admission) count(total *int64, value, ceiling int64, reads map[string]int64) error {
	if a.recorded != nil {
		a.recorded.record(total, value, ceiling, reads)
	}
	if a.instance != nil {
		value += a.instance.readWork(reads)
	}
	return add(total, value, ceiling)
}

func tally(reads []string) map[string]int64 {
	if len(reads) == 0 {
		return nil
	}
	counts := map[string]int64{}
	for _, id := range reads {
		counts[id]++
	}
	return counts
}

// Prepare admits static machines under the Profile's Contract and correlated ceilings; each
// evaluator will own fresh state and capture values. A Contract without a correlated contract needs
// no correlated ceiling.
func Prepare(source *testpilotspb.Contract, catalog *ir.Catalog, program execution.ProgramView, ceiling *testpilotspb.ContractLimits, correlatedCeiling *testpilotspb.CorrelatedLimits) (*PreparedContract, error) {
	if catalog == nil || program.ProgramID() == "" || program.CatalogIdentity() != catalog.Identity() {
		return nil, invalid(ir.Malformed, "prepared Program and matching catalog are required")
	}
	if err := ir.CheckSurface(source, ir.DefaultLimits()); err != nil {
		return nil, err
	}
	if slices.ContainsFunc(source.Rules, func(rule *testpilotspb.ContractRule) bool { return len(rule.Instances) > 0 }) {
		// The expansion's surface is charged too, so an instanced Contract is not admitted where its
		// expansion is rejected on surface size.
		if err := ir.CheckExpandedSurface(source, ir.DefaultLimits(), expandRuleInstances); err != nil {
			return nil, err
		}
	}
	if !validID(source.ContractId) || len(source.Rules) == 0 && source.Correlated == nil {
		return nil, invalid(ir.Malformed, "Contract identity and rules are required")
	}
	if err := checkLimits(ceiling, hardLimits()); err != nil {
		return nil, err
	}
	// Rule instances count as the rules of the expansion, checked before any is allocated.
	var ruleCount int64
	for _, rule := range source.Rules {
		ruleCount += max(1, int64(len(rule.Instances)))
	}
	if ruleCount > ceiling.MaxRules {
		return nil, invalid(ir.LimitExceeded, "rule count exceeds ceiling")
	}
	p := &PreparedContract{source: proto.CloneOf(source), limits: proto.CloneOf(ceiling), correlatedLimits: proto.CloneOf(correlatedCeiling), program: program, catalog: catalog, observations: map[string]ir.Type{}}
	for _, observation := range program.Observations() {
		p.observations[observation.ID] = observation.Type
	}
	a := &admission{prepared: p, catalog: catalog, scope: map[ir.Reference]ir.Binding{}, limits: ir.DefaultLimits(), ruleCount: ruleCount}
	a.limits.Depth = p.limits.MaxExpressionDepth
	a.limits.Fanout = program.Limits().MaxPathFanout
	var err error
	a.boolean, err = catalog.BindType(scalarType(testpilotspb.SCALAR_KIND_BOOLEAN))
	if err != nil {
		return nil, err
	}
	if err := a.bindScope(); err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, rule := range p.source.Rules {
		if !validID(rule.RuleId) || seen[rule.RuleId] {
			return nil, invalid(ir.Malformed, "invalid or duplicate rule identity")
		}
		seen[rule.RuleId] = true
		m, bindErr := a.bindRule(rule, seen)
		if bindErr != nil {
			return nil, fmt.Errorf("rule %s: %w", rule.RuleId, bindErr)
		}
		p.rules = append(p.rules, m)
	}
	if err := a.bindCorrelated(seen); err != nil {
		return nil, err
	}
	if err := a.boundWork(); err != nil {
		return nil, err
	}
	return p, nil
}
func scalarType(kind testpilotspb.ScalarKind) *testpilotspb.ValueType {
	return &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Scalar{Scalar: &testpilotspb.ScalarType{Kind: kind}}}}}
}

// bindRule admits a Rule and its instances. The Rule binds once, as its first instance would, and
// each further instance is charged what binding its expanded copy would cost, so the Contract is
// admitted exactly when its expansion is and rejects on the same ceiling.
func (a *admission) bindRule(rule *testpilotspb.ContractRule, seen map[string]bool) (*machine, error) {
	values, instances, err := a.bindInstances(rule, seen)
	if err != nil {
		return nil, err
	}
	if err := checkInstanceValueReads(rule, values); err != nil {
		return nil, err
	}
	recorded := &ledger{}
	if len(instances) > 0 {
		a.instance, a.recorded = &instances[0], recorded
	}
	m, err := a.bindMachine(rule, values)
	a.recorded = nil
	defer func() { a.instance = nil }()
	if err != nil {
		return nil, err
	}
	m.instances = instances
	for i := 1; i < len(instances); i++ {
		a.instance = &instances[i]
		for _, entry := range *recorded {
			if err := a.count(entry.total, entry.value, entry.ceiling, entry.reads); err != nil {
				return nil, fmt.Errorf("instance %s: %w", instances[i].ruleID, err)
			}
		}
	}
	return m, nil
}

// bindInstances checks a Rule's declared instance values and its instances before binding: every
// instance assigns every declared value exactly once, in declaration order, with a value of its
// declared type, and names a rule ID no other rule, instance or correlated rule uses.
func (a *admission) bindInstances(rule *testpilotspb.ContractRule, seen map[string]bool) (map[string]ir.Type, []ruleInstance, error) {
	path := fmt.Sprintf("contract.rules[%s]", rule.RuleId)
	if len(rule.InstanceValues) > 0 && len(rule.Instances) == 0 {
		return nil, nil, invalidAt(ir.Malformed, path+".instances", "declared instance values require Rule instances")
	}
	if len(rule.Instances) > 0 && len(rule.InstanceValues) == 0 {
		return nil, nil, invalidAt(ir.Malformed, path+".instance_values", "Rule instances require declared instance values")
	}
	types := make(map[string]ir.Type, len(rule.InstanceValues))
	for _, declared := range rule.InstanceValues {
		located := fmt.Sprintf("%s.instance_values[%s]", path, declared.InstanceValueId)
		if _, duplicate := types[declared.InstanceValueId]; duplicate || !validID(declared.InstanceValueId) {
			return nil, nil, invalidAt(ir.Malformed, located, "invalid or duplicate instance value identity")
		}
		typ, err := a.instanceValueType(declared.Type, located+".type")
		if err != nil {
			return nil, nil, err
		}
		types[declared.InstanceValueId] = typ
	}
	instances := make([]ruleInstance, 0, len(rule.Instances))
	for _, instance := range rule.Instances {
		located := fmt.Sprintf("%s.instances[%s]", path, instance.RuleId)
		if !validID(instance.RuleId) || seen[instance.RuleId] {
			return nil, nil, invalidAt(ir.Malformed, located, "invalid or duplicate rule identity")
		}
		seen[instance.RuleId] = true
		bound := ruleInstance{ruleID: instance.RuleId, values: map[string]*testpilotspb.Value{}, work: map[string]int64{}}
		for position, assignment := range instance.Assignments {
			id := assignment.InstanceValueId
			at := fmt.Sprintf("%s.assignments[%s]", located, id)
			typ, declared := types[id]
			if !declared {
				return nil, nil, invalidAt(ir.Unknown, at, "assignment names an undeclared instance value")
			}
			if _, repeated := bound.values[id]; repeated {
				return nil, nil, invalidAt(ir.Malformed, at, "instance value is assigned more than once")
			}
			// Every earlier assignment named a distinct declared value, so position indexes a declaration.
			if rule.InstanceValues[position].InstanceValueId != id {
				return nil, nil, invalidAt(ir.Malformed, at, "assignments follow the instance values' declaration order")
			}
			if assignment.Value == nil {
				return nil, nil, invalidAt(ir.Malformed, at, "assignment value is required")
			}
			if err := a.catalog.CheckLiteral(assignment.Value, typ, a.limits); err != nil {
				return nil, nil, relocate(err, at)
			}
			work, err := a.catalog.InstanceValueWork(id, assignment.Value, typ)
			if err != nil {
				return nil, nil, relocate(err, at)
			}
			bound.values[id], bound.work[id] = assignment.Value, work
		}
		if len(bound.values) < len(rule.InstanceValues) {
			missing := rule.InstanceValues[len(bound.values)].InstanceValueId
			return nil, nil, invalidAt(ir.Malformed, fmt.Sprintf("%s.assignments[%s]", located, missing), "instance omits a declared instance value")
		}
		instances = append(instances, bound)
	}
	return types, instances, nil
}

// instanceValueType binds a declared instance value type, which is text, integer or enum. A boolean
// is not admitted: capture analysis prunes paths on boolean literals, which a Rule analyzed once for
// all of its instances cannot do per instance.
func (a *admission) instanceValueType(declared *testpilotspb.SingularType, path string) (ir.Type, error) {
	switch typ := declared.GetType().(type) {
	case *testpilotspb.SingularType_Scalar:
		kind := typ.Scalar.GetKind()
		if kind == testpilotspb.SCALAR_KIND_BOOLEAN {
			return ir.Type{}, invalidAt(ir.Malformed, path, "an instance value cannot be boolean")
		}
		if kind != testpilotspb.SCALAR_KIND_TEXT && (kind < testpilotspb.SCALAR_KIND_INT32 || kind > testpilotspb.SCALAR_KIND_SFIXED64) {
			return ir.Type{}, invalidAt(ir.Malformed, path, "instance value requires a text, integer or enum type")
		}
	case *testpilotspb.SingularType_Enumeration:
	default:
		return ir.Type{}, invalidAt(ir.Malformed, path, "instance value requires a text, integer or enum type")
	}
	bound, err := a.catalog.BindType(&testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: proto.CloneOf(declared)}})
	if err != nil {
		return ir.Type{}, relocate(err, path)
	}
	return bound, nil
}

// checkInstanceValueReads locates, at its predicate, an instance value reference that is empty or
// names a value the Rule does not declare, which binding would report unlocated.
func checkInstanceValueReads(rule *testpilotspb.ContractRule, declared map[string]ir.Type) error {
	for _, tr := range rule.Transitions {
		err := ir.WalkReferences(predicatePath(rule, tr), tr.GetPredicate(), func(path string, reference *testpilotspb.Reference) error {
			read, ok := reference.GetReference().(*testpilotspb.Reference_InstanceValueId)
			if !ok {
				return nil
			}
			located := path + ".reference.instance_value_id"
			if read.InstanceValueId == "" {
				return invalidAt(ir.Malformed, located, "instance value reference names no instance value")
			}
			if _, exists := declared[read.InstanceValueId]; !exists {
				return invalidAt(ir.Unknown, located, "instance value is not declared by the rule")
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

var contractRules = (&testpilotspb.Contract{}).ProtoReflect().Descriptor().Fields().ByName("rules")

// expandRuleInstances writes a Rule with instances out as its expansion's plain Rules: one per
// instance, under the instance's rule ID, with each instance value the instance assigns inlined as a
// literal.
func expandRuleInstances(field protoreflect.FieldDescriptor, element protoreflect.Message) (int, iter.Seq[proto.Message], bool) {
	rule, ok := element.Interface().(*testpilotspb.ContractRule)
	if field != contractRules || !ok || len(rule.Instances) == 0 {
		return 0, nil, false
	}
	return len(rule.Instances), func(yield func(proto.Message) bool) {
		template := proto.CloneOf(rule)
		template.InstanceValues, template.Instances = nil, nil
		for _, instance := range rule.Instances {
			copied := proto.CloneOf(template)
			copied.RuleId = instance.RuleId
			values := make(map[string]*testpilotspb.Value, len(instance.Assignments))
			for _, assignment := range instance.Assignments {
				values[assignment.InstanceValueId] = assignment.Value
			}
			for _, tr := range copied.Transitions {
				inlineInstanceValues(tr.GetPredicate(), values)
			}
			if !yield(copied) {
				return
			}
		}
	}, true
}

// inlineInstanceValues replaces each instance value reference in e with the literal values assigns
// it. A reference values assigns nothing is left for preparation to reject at its location.
func inlineInstanceValues(e *testpilotspb.Expression, values map[string]*testpilotspb.Value) {
	switch v := e.GetExpression().(type) {
	case *testpilotspb.Expression_Reference:
		if read, ok := v.Reference.GetReference().(*testpilotspb.Reference_InstanceValueId); ok && values[read.InstanceValueId] != nil {
			e.Expression = &testpilotspb.Expression_Literal{Literal: values[read.InstanceValueId]}
		}
	case *testpilotspb.Expression_Path:
		inlineInstanceValues(v.Path.GetOperand(), values)
	case *testpilotspb.Expression_Present:
		inlineInstanceValues(v.Present.GetOperand(), values)
	case *testpilotspb.Expression_Not:
		inlineInstanceValues(v.Not.GetOperand(), values)
	case *testpilotspb.Expression_Compare:
		inlineInstanceValues(v.Compare.GetLeft(), values)
		inlineInstanceValues(v.Compare.GetRight(), values)
	case *testpilotspb.Expression_All:
		for _, operand := range v.All.GetOperands() {
			inlineInstanceValues(operand, values)
		}
	case *testpilotspb.Expression_Any:
		for _, operand := range v.Any.GetOperands() {
			inlineInstanceValues(operand, values)
		}
	default:
		// A literal holds no instance value.
	}
}

// relocate places an unlocated ir diagnostic at path.
func relocate(err error, path string) error {
	var diagnostic *ir.Error
	if !errors.As(err, &diagnostic) {
		return err
	}
	return invalidAt(diagnostic.Category, path, diagnostic.Detail)
}

func (a *admission) bindMachine(rule *testpilotspb.ContractRule, instanceValues map[string]ir.Type) (*machine, error) {
	limits := a.prepared.limits
	if err := a.count(&a.states, int64(len(rule.States)), limits.MaxStates, nil); err != nil {
		return nil, err
	}
	if err := a.count(&a.transitions, int64(len(rule.Transitions)), limits.MaxTransitions, nil); err != nil {
		return nil, err
	}
	if len(rule.States) == 0 || len(rule.Transitions) == 0 {
		return nil, invalid(ir.Malformed, "states and transitions are required")
	}
	if rule.Kind != testpilotspb.CONTRACT_RULE_KIND_SAFETY && rule.Kind != testpilotspb.CONTRACT_RULE_KIND_BOUNDED_LIVENESS {
		return nil, invalid(ir.Unknown, "unsupported rule kind")
	}
	m := &machine{source: rule, states: map[string]int{}, captures: map[string]int{}, outgoing: make([]map[testpilotspb.RunEventKind][]int, len(rule.States)), instanceValues: instanceValues}
	if err := a.bindStates(m); err != nil {
		return nil, err
	}
	if err := a.bindCaptures(m); err != nil {
		return nil, err
	}
	if err := a.charge(int64(len(a.scope)) + int64(len(m.captures))); err != nil {
		return nil, err
	}
	scope := a.scopeFor(m, nil, true)
	if err := a.bindTransitions(m, scope); err != nil {
		return nil, err
	}
	if err := a.analyzeCaptures(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (a *admission) bind(conditions []ir.Condition, value *testpilotspb.Expression, path string, expected *ir.Type, scope map[ir.Reference]ir.Binding) (*ir.Expression, error) {
	// The binding is bounded by the hard work ceiling alone and then charged against what remains, so
	// admission rejects on the remaining work at the charge, where a Rule instance's charge can be
	// replayed, rather than midway through the expression.
	bound, err := a.catalog.BindConditionedExpression(conditions, ir.Site{Context: ir.ContractContext, Path: path}, value, expected, scope, a.limits)
	if err != nil {
		return nil, err
	}
	if err := a.count(&a.work, bound.BindingWork(), ir.DefaultLimits().Work, tally(bound.InstanceValueReads())); err != nil {
		return nil, err
	}
	return bound, nil
}

func (a *admission) bindScope() error {
	for _, observation := range a.prepared.program.Observations() {
		a.scope[ir.Reference{Kind: ir.ObservationReference, ID: observation.ID}] = ir.Binding{Type: observation.Type}
	}
	// RUN_ID follows SOURCE_ID and stays Program-only. Kind-specific data is not a field: a
	// transition declares the payload arms its event kinds may carry.
	for field := testpilotspb.RUN_EVENT_FIELD_SEQUENCE; field <= testpilotspb.RUN_EVENT_FIELD_SOURCE_ID; field++ {
		kind := testpilotspb.SCALAR_KIND_TEXT
		if field == testpilotspb.RUN_EVENT_FIELD_SEQUENCE || field == testpilotspb.RUN_EVENT_FIELD_ELAPSED_MILLISECONDS || field == testpilotspb.RUN_EVENT_FIELD_ATTEMPT {
			kind = testpilotspb.SCALAR_KIND_INT64
		}
		typ, bindErr := a.catalog.BindType(scalarType(kind))
		if bindErr != nil {
			return bindErr
		}
		if enumeration, ok := eventFieldEnumerations[field]; ok {
			typ, bindErr = a.catalog.BindType(&testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Enumeration{Enumeration: &testpilotspb.NamedType{ProtobufType: string(enumeration)}}}}})
			if bindErr != nil {
				return bindErr
			}
		}
		a.scope[ir.Reference{Kind: ir.EventReference, Field: int32(field)}] = ir.Binding{Type: typ, Available: true}
	}
	return nil
}

// predicatePath locates a transition predicate by rule and transition identity rather than by index.
func predicatePath(rule *testpilotspb.ContractRule, transition *testpilotspb.ContractTransition) string {
	return fmt.Sprintf("contract.rules[%s].transitions[%s].predicate", rule.RuleId, transition.TransitionId)
}

// The Run Event fields whose declared type is an enumeration rather than a scalar.
var eventFieldEnumerations = map[testpilotspb.RunEventField]protoreflect.FullName{
	testpilotspb.RUN_EVENT_FIELD_KIND: testpilotspb.RunEventKind(0).Descriptor().FullName(),
}

func payloadReference(arm protoreflect.Name) ir.Reference {
	return ir.Reference{Kind: ir.EventPayloadReference, ID: string(arm)}
}

// declarePayloads declares in scope, for structural checking, each Run Event payload arm some
// kind in kinds may carry, and removes every other arm. A path into an arm no kind of the
// transition's filter can carry therefore rejects at preparation.
func (a *admission) declarePayloads(scope map[ir.Reference]ir.Binding, kinds []testpilotspb.RunEventKind) error {
	for kind := testpilotspb.RUN_EVENT_KIND_RUN_OPENED; kind <= ir.MaxRunEventKind; kind++ {
		if arm := ir.RunEventPayloadOf(kind).Arm; arm != "" {
			delete(scope, payloadReference(arm))
		}
	}
	for _, kind := range kinds {
		if err := a.declarePayload(scope, ir.RunEventPayloadOf(kind).Arm, true); err != nil {
			return err
		}
	}
	return nil
}

// evaluatedPayloads declares in scope every Run Event payload arm for one evaluated event kind.
// Only the arm the kind requires is available, so a read of an arm the event may lack is absent: a
// comparison over it is false there, and any other use needs a presence check to guard it.
func (a *admission) evaluatedPayloads(scope map[ir.Reference]ir.Binding, kind testpilotspb.RunEventKind) error {
	carried := ir.RunEventPayloadOf(kind)
	for other := testpilotspb.RUN_EVENT_KIND_RUN_OPENED; other <= ir.MaxRunEventKind; other++ {
		arm := ir.RunEventPayloadOf(other).Arm
		if err := a.declarePayload(scope, arm, carried.Required && carried.Arm == arm); err != nil {
			return err
		}
	}
	return nil
}

func (a *admission) declarePayload(scope map[ir.Reference]ir.Binding, arm protoreflect.Name, available bool) error {
	if arm == "" {
		return nil
	}
	typ, known := a.catalog.RunEventPayloadType(arm)
	if !known {
		return invalid(ir.Malformed, "Run Event payload table names an undeclared arm")
	}
	scope[payloadReference(arm)] = ir.Binding{Type: typ, Available: available}
	return nil
}

func (a *admission) bindStates(m *machine) error {
	rule := m.source
	for i, state := range rule.States {
		if !validID(state.StateId) {
			return invalid(ir.Malformed, "invalid state identity")
		}
		if _, exists := m.states[state.StateId]; exists {
			return invalid(ir.Malformed, "duplicate state")
		}
		if state.Status < testpilotspb.CONTRACT_STATE_STATUS_PENDING || state.Status > testpilotspb.CONTRACT_STATE_STATUS_VIOLATED {
			return invalid(ir.Unknown, "invalid terminal state kind")
		}
		m.states[state.StateId] = i
		m.outgoing[i] = map[testpilotspb.RunEventKind][]int{}
	}
	initial, ok := m.states[rule.InitialStateId]
	if !ok {
		return invalid(ir.Unknown, "initial state is not declared")
	}
	m.initial = initial
	if rule.States[initial].Status != testpilotspb.CONTRACT_STATE_STATUS_PENDING {
		return invalid(ir.Malformed, "initial state must be nonterminal")
	}
	if rule.Kind == testpilotspb.CONTRACT_RULE_KIND_BOUNDED_LIVENESS {
		target, exists := m.states[rule.Deadline.GetViolationStateId()]
		path := fmt.Sprintf("contract.rules[%s].deadline", rule.RuleId)
		// The bound oneof carries one bound, so a rule never expires on two clocks at once.
		switch bound := rule.Deadline.GetBound().(type) {
		case *testpilotspb.Deadline_RuleEvents:
			if bound.RuleEvents <= 0 {
				return invalidAt(ir.Malformed, path+".rule_events", "liveness deadline bound must be positive")
			}
		case *testpilotspb.Deadline_ElapsedMilliseconds:
			if bound.ElapsedMilliseconds <= 0 {
				return invalidAt(ir.Malformed, path+".elapsed_milliseconds", "liveness deadline bound must be positive")
			}
		default:
			return invalidAt(ir.Malformed, path, "liveness deadline requires a bound")
		}
		if !exists || rule.States[target].Status != testpilotspb.CONTRACT_STATE_STATUS_VIOLATED {
			return invalid(ir.Malformed, "liveness requires a violated deadline target")
		}
	} else if rule.Deadline != nil {
		return invalid(ir.Malformed, "safety rule cannot declare a liveness deadline")
	}
	return nil
}

func (a *admission) bindTransitions(m *machine, scope map[ir.Reference]ir.Binding) error {
	rule := m.source
	seen := map[string]bool{}
	for i, tr := range rule.Transitions {
		if err := a.charge(1 + int64(len(tr.EventFilter.GetKinds()))); err != nil {
			return err
		}
		from, fromOK := m.states[tr.SourceStateId]
		_, toOK := m.states[tr.TargetStateId]
		if !validID(tr.TransitionId) || seen[tr.TransitionId] || !fromOK || !toOK {
			return invalid(ir.Malformed, "invalid, duplicate, or undeclared transition identity/state")
		}
		seen[tr.TransitionId] = true
		if rule.States[from].Status != testpilotspb.CONTRACT_STATE_STATUS_PENDING {
			return invalid(ir.Malformed, "terminal states cannot have outgoing transitions")
		}
		if tr.SupportKind != testpilotspb.CONTRACT_SUPPORT_KIND_NONE && tr.SupportKind != testpilotspb.CONTRACT_SUPPORT_KIND_MATCHING_EVENT {
			return invalid(ir.Unknown, "invalid supporting event policy")
		}
		if len(tr.EventFilter.GetKinds()) == 0 {
			return invalid(ir.Malformed, "transition event kinds required")
		}
		for _, kind := range tr.EventFilter.Kinds {
			if kind < testpilotspb.RUN_EVENT_KIND_RUN_OPENED || kind > ir.MaxRunEventKind || slices.Contains(m.outgoing[from][kind], i) {
				return invalid(ir.Unknown, "invalid or duplicate event kind")
			}
			m.outgoing[from][kind] = append(m.outgoing[from][kind], i)
		}
		if err := a.declarePayloads(scope, tr.EventFilter.Kinds); err != nil {
			return err
		}
		bound, err := a.bind(nil, tr.Predicate, predicatePath(rule, tr), &a.boolean, scope)
		if err != nil {
			return err
		}
		m.transitions = append(m.transitions, bound)
		if err := a.checkAssignments(m, tr); err != nil {
			return err
		}
	}
	return nil
}
