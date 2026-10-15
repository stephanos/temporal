package wasi

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"sync"

	"go.temporal.io/server/tools/gomad/internal/model/clock"
	"go.temporal.io/server/tools/gomad/internal/model/world"
)

type BrokerNode struct {
	ID          string
	Incarnation uint64
}

type BrokerLink struct {
	From, To   string
	Enabled    bool
	DelayNanos uint64
}

type BrokerConfig struct {
	Nodes          []BrokerNode
	Seed           uint64
	Limits         world.Limits
	Network        *NetworkModelV2
	Addresses      map[string]string
	Faults         []BrokerNetworkFault
	Links          []BrokerLink
	StrictTopology bool
	Diagnostics    bool
	// WorldProfile optionally carries the binary snapshot schema in retained identities.
	WorldProfile string `json:",omitempty"`
}

// Broker serializes isolated helper continuations. The shared simulation clock
// owns progress accounting; World owns the externally visible delivery order.
type Broker struct {
	mu                sync.Mutex
	clock             *clock.BrokerClock
	world             *world.Model
	sessions          map[string]*BrokerSession
	order             []string
	active            *BrokerSession
	preparing         *BrokerSession
	changed           chan struct{}
	network           *brokerNetwork
	restartHolds      map[string]*clock.BrokerParticipant
	crashOutputs      map[string][]byte
	crashSeen         map[string]bool
	diagnostics       bool
	diagnosticLog     []byte
	idleAt, idleCount uint64
	worldProfile      string
	worldFinalized    bool
}

func (s *BrokerSession) acquirePreparation() (bool, <-chan struct{}, error) {
	b := s.broker
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.sessions[s.Node.ID] != s || s.state == "finished" || s.state == "revoked" {
		return false, nil, invalid("stale broker preparation")
	}
	if b.preparing == nil || b.preparing == s {
		b.preparing = s
		return true, nil, nil
	}
	return false, b.changed, nil
}

// Prepared releases only the host compilation slot, not a guest continuation.
func (s *BrokerSession) Prepared() {
	b := s.broker
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.preparing == s {
		b.preparing = nil
		b.signal()
	}
}

type BrokerSession struct {
	broker              *Broker
	Node                BrokerNode
	participant         *clock.BrokerParticipant
	state               string
	arrivals            uint32
	request, generation uint64
	pending             world.RequestID
	resume              *clock.BrokerTimeResponse
	err                 error
	slot                uint64
	crashMarker         []byte
	stopMarker          []byte
	termination         string
	environment         *Environment
	// localCalls marks split ownership, where the helper answers local calls.
	localCalls bool
	// afterHandle is a test-only hook between a handled call and its
	// broker completion, where a concurrent revocation can land.
	afterHandle func()
}

func NewBroker(config BrokerConfig) (*Broker, error) {
	if config.WorldProfile != "" && config.WorldProfile != world.CompressedSnapshotSchema {
		return nil, invalid("broker World profile")
	}
	if err := world.ValidateCompressedSnapshotConfiguration(world.Config{Seed: world.Seed(config.Seed), Limits: config.Limits}, world.CompressedSnapshotProfile()); err != nil {
		return nil, err
	}
	model, err := world.New(world.Config{Seed: world.Seed(config.Seed), Limits: config.Limits})
	if err != nil {
		return nil, err
	}
	if len(config.Nodes) == 0 {
		return nil, invalid("broker nodes")
	}
	b := &Broker{clock: clock.NewBrokerClock(), world: model, sessions: make(map[string]*BrokerSession), changed: make(chan struct{}), restartHolds: make(map[string]*clock.BrokerParticipant), diagnostics: config.Diagnostics, worldProfile: config.WorldProfile}
	for _, node := range config.Nodes {
		if node.ID == "" || node.Incarnation == 0 || b.sessions[node.ID] != nil {
			return nil, invalid("broker node identity")
		}
		p, err := b.clock.Register(fmt.Sprintf("%s/%d", node.ID, node.Incarnation))
		if err != nil {
			return nil, err
		}
		b.sessions[node.ID] = &BrokerSession{broker: b, Node: node, participant: p, state: "runnable"}
		b.order = append(b.order, node.ID)
	}
	sort.Strings(b.order)
	for index, id := range b.order {
		b.sessions[id].slot = uint64(index + 1)
	}
	b.network, err = newBrokerNetwork(config, b)
	if err != nil {
		return nil, err
	}
	b.schedule()
	return b, nil
}

func (b *Broker) Session(node string) *BrokerSession {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.sessions[node]
}

func (b *Broker) Snapshot() world.Snapshot {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.world.Snapshot()
}

// EncodeWorldEvidence transfers compressed terminal state only after every
// helper has finished. Serialization runs outside both Broker and World locks.
func (b *Broker) EncodeWorldEvidence() ([]byte, error) {
	b.mu.Lock()
	if b.worldFinalized {
		b.mu.Unlock()
		return nil, invalid("broker World finalized")
	}
	for _, session := range b.sessions {
		if session.state != "finished" {
			b.mu.Unlock()
			return nil, invalid("broker World participant is not finished")
		}
	}
	detached, err := b.world.DetachCompressedSnapshot(world.CompressedSnapshotProfile())
	if err == nil {
		b.worldFinalized = true
	}
	b.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return detached.Encode()
}

func (b *Broker) DiagnosticCardinality() world.DiagnosticCardinality {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.world.DiagnosticCardinality()
}

func (b *Broker) Current() uint64 { return uint64(b.clock.Current() - int64(world.InitialTime)) }

func (b *Broker) signal() { close(b.changed); b.changed = make(chan struct{}) }

func (b *Broker) schedule() {
	if b.active != nil {
		return
	}
	for _, id := range b.order {
		s := b.sessions[id]
		if s.state == "runnable" {
			s.state = "active"
			b.active = s
			b.signal()
			return
		}
	}
	ready := false
	for _, id := range b.order {
		s := b.sessions[id]
		if s.state == "idle" {
			if s.resume == nil && s.err == nil {
				return
			}
			ready = true
		}
	}
	if ready {
		if b.network != nil {
			if err := b.network.advance(b.Current()); err != nil {
				for _, id := range b.order {
					if b.sessions[id].state == "idle" {
						b.sessions[id].err = err
					}
				}
			}
		}
		for _, id := range b.order {
			s := b.sessions[id]
			if s.state == "idle" {
				s.state = "runnable"
			}
		}
		b.schedule()
	}
}

// Crashed distinguishes a modeled incarnation revocation from a host watchdog.
func (s *BrokerSession) Crashed() bool {
	b := s.broker
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.restartHolds[s.Node.ID] != nil && b.sessions[s.Node.ID] == s
}

func (s *BrokerSession) Stopped() bool {
	b := s.broker
	b.mu.Lock()
	defer b.mu.Unlock()
	return s.termination == "stop" && b.sessions[s.Node.ID] == s
}

func (s *BrokerSession) register(op string, payload []byte) error {
	digest := sha256.Sum256(payload)
	id, err := s.broker.world.Register(world.Request{Kind: op, Resource: world.ResourceID{Adapter: "wasm-broker", Kind: "incarnation", Key: fmt.Sprintf("%s/%d", s.Node.ID, s.Node.Incarnation)}, Payload: digest[:]})
	if err == nil {
		s.pending = id
	}
	return err
}

func (s *BrokerSession) ready(op string) error {
	b := s.broker
	if _, err := b.world.Ready(world.Readiness{RequestID: s.pending, At: world.LogicalTime(b.clock.Current()), Kind: op}); err != nil {
		return err
	}
	q, err := b.world.Quiesce()
	if err != nil {
		return err
	}
	if len(q.Deliveries) != 1 || q.Deliveries[0].RequestID != s.pending {
		return invalid("broker delivery order")
	}
	s.pending = 0
	return nil
}

func brokerVisible(call Call) bool {
	return call.Op == "runtime_idle" || call.Op == "runtime_net_stream"
}

// currentErrorLocked preserves the modeled boundary for a revoked current
// incarnation, without accepting work from finished or replaced incarnations.
func (s *BrokerSession) currentErrorLocked() error {
	if s.broker.sessions[s.Node.ID] != s || s.state == "finished" {
		return invalid("stale broker incarnation")
	}
	if s.state == "revoked" {
		if s.err != nil {
			return s.err
		}
		return invalid("stale broker incarnation")
	}
	return nil
}

func (s *BrokerSession) beginCall(e *Environment, call Call) error {
	b := s.broker
	b.mu.Lock()
	if err := s.currentErrorLocked(); err != nil {
		b.mu.Unlock()
		return err
	}
	// Split ownership answers local calls in the helper, so remote call
	// IDs advance by at least one rather than exactly one.
	if call.ID != s.request+1 && (!s.localCalls || call.ID <= s.request) {
		b.mu.Unlock()
		return invalid("broker request generation")
	}
	s.request = call.ID
	s.environment = e
	if call.Op != "runtime_idle" {
		if err := b.clock.Accepted(s.participant, call.ID, s.arrivals); err != nil {
			b.mu.Unlock()
			return err
		}
		s.arrivals = 0
	}
	visible := brokerVisible(call)
	if visible {
		if err := s.register(call.Op, call.Input); err != nil {
			b.mu.Unlock()
			return err
		}
	}
	b.mu.Unlock()
	if b.network != nil {
		if err := b.network.attach(s, e); err != nil {
			return err
		}
	}
	e.now = max(e.now, b.Current())
	return nil
}

func (s *BrokerSession) finishCall(e *Environment, call Call) error {
	b := s.broker
	visible := brokerVisible(call)
	b.mu.Lock()
	if err := s.currentErrorLocked(); err != nil {
		b.mu.Unlock()
		return err
	}
	crash := len(s.crashMarker) != 0 && bytes.Contains(e.stdout, s.crashMarker)
	stop := len(s.stopMarker) != 0 && bytes.Contains(e.stdout, s.stopMarker)
	var crashNodes []string
	if marker := b.crashOutputs[s.Node.ID]; len(marker) != 0 && bytes.Contains(e.stdout, marker) {
		b.crashSeen[s.Node.ID] = true
		all := true
		for id := range b.crashOutputs {
			if !b.crashSeen[id] {
				all = false
			}
		}
		if all {
			for _, id := range b.order {
				if b.crashOutputs[id] != nil {
					crashNodes = append(crashNodes, id)
				}
			}
			b.crashOutputs = nil
		}
	}
	b.mu.Unlock()
	if crash {
		if err := b.Crash(s.Node.ID); err != nil {
			return err
		}
		return &BoundaryError{Kind: "crash", Dimension: "broker output trigger"}
	}
	if len(crashNodes) != 0 {
		for _, id := range crashNodes {
			if err := b.Crash(id); err != nil {
				return err
			}
		}
		return &BoundaryError{Kind: "crash", Dimension: "broker output barrier"}
	}
	if stop {
		if err := b.Stop(s.Node.ID); err != nil {
			return err
		}
		return &BoundaryError{Kind: "stop", Dimension: "broker output trigger"}
	}
	if visible {
		b.mu.Lock()
		err := s.currentErrorLocked()
		if err == nil {
			err = s.ready(call.Op)
		}
		b.mu.Unlock()
		return err
	}
	return nil
}

func (s *BrokerSession) prepareIdle(deadline uint64) (clock.BrokerTimeRequest, error) {
	if deadline == 0 {
		deadline = uint64(math.MaxInt64 - int64(world.InitialTime))
	}
	if deadline > uint64(math.MaxInt64-int64(world.InitialTime)) {
		return clock.BrokerTimeRequest{}, invalid("broker deadline")
	}
	b := s.broker
	b.mu.Lock()
	if err := s.currentErrorLocked(); err != nil {
		b.mu.Unlock()
		return clock.BrokerTimeRequest{}, err
	}
	if b.active != s {
		b.mu.Unlock()
		return clock.BrokerTimeRequest{}, invalid("broker continuation")
	}
	if b.network != nil {
		if next := b.network.deadline(); next != 0 {
			deadline = min(deadline, next)
		}
		if len(b.network.model.Ready()) != 0 {
			deadline = min(deadline, max(uint64(1), b.Current()))
		}
	}
	if b.diagnostics {
		if at := b.Current(); at == b.idleAt {
			b.idleCount++
		} else {
			b.idleAt = at
			b.idleCount = 1
		}
		if b.idleCount == 1000 {
			if err := b.debugIdle(); err != nil {
				b.mu.Unlock()
				return clock.BrokerTimeRequest{}, err
			}
		}
	}
	s.generation++
	request := clock.BrokerTimeRequest{Generation: s.generation, Current: b.clock.Current(), Deadline: int64(world.InitialTime) + int64(deadline), Arrivals: s.arrivals}
	s.arrivals = 0
	s.state = "idle"
	s.resume = nil
	b.active = nil
	b.schedule()
	b.mu.Unlock()
	return request, nil
}

func (s *BrokerSession) resumeIdle(response clock.BrokerTimeResponse, err error) error {
	b := s.broker
	b.mu.Lock()
	if s.state == "revoked" {
		err = s.err
		b.mu.Unlock()
		return err
	}
	if s.state != "finished" {
		if err != nil {
			s.err = err
		} else {
			s.resume = &response
		}
		b.schedule()
		b.signal()
	}
	b.mu.Unlock()
	return err
}

func (b *Broker) debugIdle() error {
	encoded, err := b.diagnosticSnapshot()
	if err != nil {
		return err
	}
	b.diagnosticLog = append(b.diagnosticLog, encoded...)
	b.diagnosticLog = append(b.diagnosticLog, '\n')
	return nil
}

func (b *Broker) diagnosticSnapshot() ([]byte, error) {
	type participant struct {
		Node          string
		Slot          uint64
		State         string
		LocalRequests []uint64
	}
	var participants []participant
	for _, id := range b.order {
		s := b.sessions[id]
		p := participant{Node: id, Slot: s.slot, State: s.state}
		// The active helper may be changing its local request map outside the
		// broker lock. Only quiescent participants have a stable local view.
		if s != b.active && s.environment != nil && s.environment.networkV2 != nil {
			for request := range s.environment.networkV2.requests {
				p.LocalRequests = append(p.LocalRequests, request)
			}
			sort.Slice(p.LocalRequests, func(i, j int) bool { return p.LocalRequests[i] < p.LocalRequests[j] })
		}
		participants = append(participants, p)
	}
	var ready []uint64
	if b.network != nil {
		ready = b.network.model.Ready()
	}
	return json.Marshal(struct {
		At, RepeatedIdle uint64
		Ready            []uint64
		Participants     []participant
		World            world.DiagnosticCardinality
	}{uint64(b.clock.Current() - int64(world.InitialTime)), b.idleCount, ready, participants, b.world.DiagnosticCardinality()})
}

func (b *Broker) Diagnostics() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	data := bytes.Clone(b.diagnosticLog)
	if b.diagnostics {
		current, err := b.diagnosticSnapshot()
		if err == nil {
			data = append(data, current...)
			data = append(data, '\n')
		}
	}
	return data
}

func (s *BrokerSession) Delivered(call Call) error {
	b := s.broker
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := s.currentErrorLocked(); err != nil {
		return err
	}
	if call.Op == "runtime_idle" {
		return nil
	}
	if call.ID != s.request || s.arrivals != 0 {
		return invalid("broker completion generation")
	}
	if err := b.clock.Delivered(s.participant, call.ID); err != nil {
		return err
	}
	s.arrivals = 1
	return nil
}

func (s *BrokerSession) localContinuation() (uint64, error) {
	b := s.broker
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := s.currentErrorLocked(); err != nil {
		return 0, err
	}
	return b.Current(), nil
}

func (s *BrokerSession) Finished() error {
	b := s.broker
	b.mu.Lock()
	defer b.mu.Unlock()
	if s.state == "finished" {
		return nil
	}
	var err error
	if s.pending != 0 {
		_, err = b.world.Cancel(s.pending)
		s.pending = 0
	}
	if s.state != "revoked" {
		err = errors.Join(err, b.clock.Remove(s.participant))
		if b.network != nil {
			err = errors.Join(err, b.network.revoke(s, true))
		}
	}
	s.state = "finished"
	if b.preparing == s {
		b.preparing = nil
	}
	if hold := b.restartHolds[s.Node.ID]; hold != nil {
		err = errors.Join(err, b.clock.Remove(hold))
		delete(b.restartHolds, s.Node.ID)
	}
	if b.active == s {
		b.active = nil
	}
	b.schedule()
	b.signal()
	return err
}

func (b *Broker) Crash(node string) error {
	return b.revoke(node, "crash", false)
}

func (b *Broker) Stop(node string) error {
	return b.revoke(node, "stop", true)
}

func (b *Broker) revoke(node, termination string, graceful bool) error {
	s := b.Session(node)
	if s == nil {
		return invalid("unknown broker node")
	}
	b.mu.Lock()
	if b.restartHolds[node] != nil || s.state == "finished" || s.state == "revoked" {
		b.mu.Unlock()
		return invalid("broker crash lifecycle")
	}
	hold, err := b.clock.Register(fmt.Sprintf("restart/%s/%d", node, s.Node.Incarnation))
	if err != nil {
		b.mu.Unlock()
		return err
	}
	b.restartHolds[node] = hold
	err = b.clock.Remove(s.participant)
	if b.network != nil {
		err = errors.Join(err, b.network.revoke(s, graceful))
	}
	if s.pending != 0 {
		_, cancelErr := b.world.Cancel(s.pending)
		err = errors.Join(err, cancelErr)
		s.pending = 0
	}
	s.state = "revoked"
	s.termination = termination
	s.err = &BoundaryError{Kind: termination, Dimension: "broker incarnation revoked"}
	b.signal()
	b.mu.Unlock()
	return err
}

func (b *Broker) CrashAfterOutputs(nodes []string, marker []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(nodes) == 0 || len(marker) == 0 || len(marker) > 4096 || b.crashOutputs != nil {
		return invalid("broker output crash barrier")
	}
	outputs := make(map[string][]byte, len(nodes))
	for _, id := range nodes {
		if b.sessions[id] == nil || outputs[id] != nil {
			return invalid("broker crash barrier nodes")
		}
		outputs[id] = bytes.Clone(marker)
	}
	b.crashOutputs = outputs
	b.crashSeen = map[string]bool{}
	return nil
}

func (s *BrokerSession) CrashOnOutput(marker []byte) error {
	return s.terminateOnOutput(marker, false)
}

func (s *BrokerSession) StopOnOutput(marker []byte) error {
	return s.terminateOnOutput(marker, true)
}

func (s *BrokerSession) terminateOnOutput(marker []byte, stop bool) error {
	if len(marker) == 0 || len(marker) > 4096 {
		return invalid("broker crash marker")
	}
	b := s.broker
	b.mu.Lock()
	defer b.mu.Unlock()
	if s.state == "finished" || s.state == "revoked" || len(s.crashMarker) != 0 || len(s.stopMarker) != 0 {
		return invalid("broker crash trigger lifecycle")
	}
	if stop {
		s.stopMarker = bytes.Clone(marker)
	} else {
		s.crashMarker = bytes.Clone(marker)
	}
	return nil
}

func (b *Broker) Restart(node string) (*BrokerSession, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.worldFinalized {
		return nil, invalid("broker World finalized")
	}
	old := b.sessions[node]
	if old == nil || old.state != "finished" || old.Node.Incarnation == math.MaxUint64 {
		return nil, invalid("broker restart lifecycle")
	}
	n := BrokerNode{node, old.Node.Incarnation + 1}
	if old.slot+uint64(len(b.order)) >= 1<<16 {
		return nil, capacity("broker incarnation slots")
	}
	p, err := b.clock.Register(fmt.Sprintf("%s/%d", n.ID, n.Incarnation))
	if err != nil {
		return nil, err
	}
	s := &BrokerSession{broker: b, Node: n, participant: p, state: "runnable", slot: old.slot + uint64(len(b.order))}
	if b.network != nil {
		if err := b.network.register(s); err != nil {
			return nil, err
		}
	}
	b.sessions[node] = s
	if hold := b.restartHolds[node]; hold != nil {
		if err := b.clock.Remove(hold); err != nil {
			return nil, err
		}
		delete(b.restartHolds, node)
	}
	b.schedule()
	b.signal()
	return s, nil
}
