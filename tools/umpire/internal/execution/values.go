package execution

import (
	"context"
	"math"
	"math/bits"
	"sync"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/internal/ir"
)

type valueStore struct {
	mu          sync.Mutex
	program     *PreparedProgram
	runID       string
	sealed      bool
	changed     chan struct{}
	attempts    int64
	activations map[string]*activationValues
	controllers map[string]bool
	slots       map[string]*umpirespb.Value
}
type activationValues struct {
	store    *valueStore
	graph    *graph
	id       string
	slots    map[string]*umpirespb.Value
	outcomes map[Coordinate]*valueBatch
	latest   map[string]*valueBatch
}
type valueBatch struct {
	owner      *activationValues
	coordinate Coordinate
	outcome    *umpirespb.InstructionOutcome
	fields     map[umpirespb.InstructionOutcomeField]*umpirespb.Value
	writes     map[string]*umpirespb.Value
	facts      []projectionFact
}
type projectionFact struct {
	projection, index int64
	observations      []*umpirespb.ObservationValue
}

func newValueStore(program *PreparedProgram, runID string) (*valueStore, error) {
	if program == nil || !validID(runID) {
		return nil, invalid(ir.Malformed, "values", "prepared Program and Run identity required")
	}
	return &valueStore{program: program, runID: runID, changed: make(chan struct{}), activations: map[string]*activationValues{}, controllers: map[string]bool{}, slots: map[string]*umpirespb.Value{}}, nil
}
func (s *valueStore) activate(entrypoint, id string) (*activationValues, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sealed || !validID(id) {
		return nil, invalid(ir.Unavailable, "values", "closed store or invalid activation")
	}
	if _, exists := s.activations[id]; exists {
		return nil, invalid(ir.Malformed, "values", "duplicate activation identity")
	}
	if int64(len(s.activations)) >= s.program.source.Limits.MaxActivations {
		return nil, invalid(ir.LimitExceeded, "values", "activation ceiling exceeded")
	}
	var selected *graph
	for _, g := range s.program.graphs {
		if g.id == entrypoint {
			selected = g
			break
		}
	}
	if selected == nil {
		return nil, invalid(ir.Unknown, "values", "unknown entrypoint")
	}
	if selected.context == umpirespb.ENTRYPOINT_CONTEXT_CONTROLLER && s.controllers[entrypoint] {
		return nil, invalid(ir.Malformed, "values", "controller already activated")
	}
	a := &activationValues{store: s, graph: selected, id: id, slots: map[string]*umpirespb.Value{}, outcomes: map[Coordinate]*valueBatch{}, latest: map[string]*valueBatch{}}
	if selected.context == umpirespb.ENTRYPOINT_CONTEXT_CONTROLLER {
		a.slots = s.slots
		s.controllers[entrypoint] = true
	}
	s.activations[id] = a
	return a, nil
}
func (s *valueStore) seal() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.sealed {
		s.sealed = true
		close(s.changed)
	}
}
func (a *activationValues) instruction(c Coordinate) (*node, error) {
	a.store.mu.Lock()
	sealed := a.store.sealed
	a.store.mu.Unlock()
	if sealed {
		return nil, invalid(ir.Unavailable, "values", "store sealed")
	}
	if c.RunID != a.store.runID || c.EntrypointID != a.graph.id || c.ActivationID != a.id {
		return nil, invalid(ir.TypeMismatch, "values", "crossed Run or activation owner")
	}
	index, exists := a.graph.index[c.InstructionID]
	if !exists {
		return nil, invalid(ir.Unknown, "values", "unknown instruction")
	}
	n := a.graph.nodes[index]
	if c.Attempt <= 0 || c.Attempt > n.source.Bounds.MaxAttempts {
		return nil, invalid(ir.LimitExceeded, "values", "invalid attempt")
	}
	return n, nil
}
func (a *activationValues) commit(ctx context.Context, batch *valueBatch) error {
	s := a.store
	s.mu.Lock()
	defer s.mu.Unlock()
	if ctx == nil {
		return invalid(ir.Malformed, "values", "context required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.sealed {
		return invalid(ir.Unavailable, "values", "store sealed")
	}
	if batch == nil || batch.owner != a {
		return invalid(ir.TypeMismatch, "values", "crossed staged batch owner")
	}
	if _, exists := a.outcomes[batch.coordinate]; exists {
		return invalid(ir.Malformed, "values", "attempt already assigned")
	}
	if previous := a.latest[batch.coordinate.InstructionID]; previous != nil && previous.coordinate.Attempt >= batch.coordinate.Attempt {
		return invalid(ir.Malformed, "values", "out-of-order attempt")
	}
	if s.attempts >= s.program.source.Limits.MaxAttempts {
		return invalid(ir.LimitExceeded, "values", "attempt ceiling exceeded")
	}
	for id := range batch.writes {
		if _, exists := a.slots[id]; exists {
			return invalid(ir.Malformed, "values", "Slot already assigned")
		}
	}
	for id, value := range batch.writes {
		a.slots[id] = value
	}
	a.outcomes[batch.coordinate] = batch
	a.latest[batch.coordinate.InstructionID] = batch
	s.attempts++
	close(s.changed)
	s.changed = make(chan struct{})
	return nil
}

// Runtime work scales with admitted operations and payload bounds, independently of binding work.
// Each operation may validate, encode/decode and copy values at every admitted path depth.
func (a *activationValues) workLimit() int64 { return a.graph.runtimeWork }
func runtimeWorkLimit(g *graph, limits *umpirespb.ProgramLimits) int64 {
	operations := int64(1)
	for _, n := range g.nodes {
		operations += int64(len(n.assignments)+len(n.outcomes)+1) + expressionNodes(n.guard) + expressionNodes(n.input)
		for _, assignment := range n.assignments {
			operations += expressionNodes(assignment.value)
		}
		for _, p := range n.projections {
			operations += int64(len(p.sinks) + 1)
		}
	}
	result := max(limits.MaxRequestBytes, limits.MaxResponseBytes)
	if result >= math.MaxInt64-limits.MaxPathFanout {
		return math.MaxInt64
	}
	result += limits.MaxPathFanout + 1
	for _, factor := range []int64{limits.MaxExpressionDepth + 1, operations, int64(bits.Len64(uint64(limits.MaxPathFanout))) + 1, 32} {
		if result > math.MaxInt64/factor {
			return math.MaxInt64
		}
		result *= factor
	}
	return result
}

func expressionNodes(expression *ir.Expression) int64 {
	if expression == nil {
		return 0
	}
	count := int64(1)
	for _, child := range expression.Children() {
		count += expressionNodes(child)
	}
	return count
}

type valueWork struct {
	ctx    context.Context
	limits ir.Limits
	work   int64
}

func (a *activationValues) newWork(ctx context.Context, limit int64) (*valueWork, error) {
	return newValueWork(ctx, a.store.program.source.Limits, a.workLimit(), limit)
}
func newValueWork(ctx context.Context, p *umpirespb.ProgramLimits, ceiling, limit int64) (*valueWork, error) {
	if ctx == nil || limit <= 0 || limit > ceiling {
		return nil, invalid(ir.LimitExceeded, "values", "invalid runtime work ceiling")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &valueWork{ctx: ctx, limits: ir.Limits{Depth: p.MaxExpressionDepth, Work: limit, Bytes: max(p.MaxRequestBytes, p.MaxResponseBytes), Fanout: p.MaxPathFanout}}, nil
}
func (w *valueWork) remaining(bytes int64) ir.Limits {
	l := w.limits
	l.Work -= w.work
	l.Bytes = bytes
	return l
}
func (w *valueWork) charge(count int64) error {
	if err := w.ctx.Err(); err != nil {
		return err
	}
	if count < 0 || count > w.limits.Work-w.work {
		return invalid(ir.LimitExceeded, "values", "runtime work ceiling exceeded")
	}
	w.work += count
	return nil
}
func (w *valueWork) copy(value *umpirespb.Value, typ ir.Type) (*umpirespb.Value, error) {
	snapshot, work, err := ir.SnapshotValue(w.ctx, value, typ, w.remaining(w.limits.Bytes))
	w.work += work
	return snapshot, err
}
func (a *activationValues) evaluate(w *valueWork, e *ir.Expression) (*umpirespb.Value, error) {
	s := a.store
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sealed {
		return nil, invalid(ir.Unavailable, "values", "store sealed")
	}
	value, work, err := e.EvaluateExecution(w.ctx, func(ref ir.Reference) *umpirespb.Value {
		switch ref.Kind {
		case ir.EventReference:
			if ref.Field == int32(umpirespb.RUN_EVENT_FIELD_RUN_ID) {
				return textValue(s.runID)
			}
		case ir.SlotReference:
			return a.slots[ref.ID]
		case ir.OutcomeReference:
			if ref.Entrypoint == a.graph.id {
				if batch := a.latest[ref.ID]; batch != nil {
					return batch.fields[umpirespb.InstructionOutcomeField(ref.Field)]
				}
			}
		default:
		}
		return nil
	}, w.limits.Work-w.work)
	w.work += work
	return value, err
}

func (a *activationValues) awaitSlot(ctx context.Context, id string) error {
	for {
		a.store.mu.Lock()
		if a.store.sealed {
			a.store.mu.Unlock()
			return invalid(ir.Unavailable, "values", "store sealed")
		}
		if a.slots[id] != nil {
			a.store.mu.Unlock()
			return nil
		}
		changed := a.store.changed
		a.store.mu.Unlock()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-changed:
		}
	}
}
