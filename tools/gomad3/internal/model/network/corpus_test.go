package network

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"syscall"
	"testing"
)

const networkCorpusDirectory = "../../backends/wasm/wasmhost/tests/testdata/network_v2"

// Inputs are concrete Model calls, including resolved handles and ports, so a
// consumer needs neither Go's scenario builders nor the net_stream ABI.
type networkCorpusInput struct {
	Action    string   `json:"action"`
	Request   *Request `json:"request,omitempty"`
	Domain    *Domain  `json:"domain,omitempty"`
	RequestID uint64   `json:"request_id,omitempty"`
	Handle    uint64   `json:"handle,omitempty"`
	Left      string   `json:"left,omitempty"`
	Right     string   `json:"right,omitempty"`
	Symmetric bool     `json:"symmetric,omitempty"`
	Graceful  bool     `json:"graceful,omitempty"`
	Nanos     uint64   `json:"nanos,omitempty"`
}

type networkCorpusPending struct {
	RequestID uint64 `json:"request_id"`
	State     string `json:"state"`
	Accepted  int    `json:"accepted"`
}

type networkCorpusState struct {
	Now             uint64                 `json:"now"`
	PendingBytes    uint64                 `json:"pending_bytes"`
	HistoryReserved uint64                 `json:"history_reserved"`
	Live            []Domain               `json:"live_domains"`
	Pending         []networkCorpusPending `json:"pending_operations"`
}

type networkCorpusOutput struct {
	Result      *Result            `json:"result,omitempty"`
	Error       string             `json:"error"`
	Ready       []uint64           `json:"ready"`
	Deadline    uint64             `json:"next_deadline"`
	State       networkCorpusState `json:"state"`
	Snapshot    Snapshot           `json:"snapshot"`
	Transitions []Transition       `json:"transitions"`
}

type networkCorpusStep struct {
	Input    networkCorpusInput  `json:"input"`
	Expected networkCorpusOutput `json:"expected"`
}

type networkCorpus struct {
	Schema         string              `json:"schema"`
	Implementation string              `json:"implementation_sha256"`
	Name           string              `json:"name"`
	Run            uint64              `json:"run"`
	Seed           uint64              `json:"seed,omitempty"`
	Config         Config              `json:"config"`
	CreationError  string              `json:"creation_error"`
	Steps          []networkCorpusStep `json:"steps"`
	Final          *Record             `json:"final,omitempty"`
	FinalError     string              `json:"final_error"`
}

type networkCorpusBuilder struct {
	t       *testing.T
	model   *Model
	corpus  networkCorpus
	ids     map[uint64]uint64
	history int
}

func newNetworkCorpus(t *testing.T, name string, config Config) *networkCorpusBuilder {
	t.Helper()
	m, err := New(1, config)
	b := &networkCorpusBuilder{t: t, model: m, ids: make(map[uint64]uint64), corpus: networkCorpus{
		Schema: "gomad3.network-operation-corpus/v1", Implementation: ImplementationSHA256(), Name: name, Run: 1, Config: config,
	}}
	if err != nil {
		b.corpus.CreationError = err.Error()
	}
	return b
}

func networkCorpusConfig(limits Limits) Config {
	return Config{Nodes: []Node{{Node: "a", Address: "a"}, {Node: "b", Address: "b"}}, Links: []Link{{From: "a", To: "b", Enabled: true}, {From: "b", To: "a", Enabled: true}}, Limits: limits}
}

func (b *networkCorpusBuilder) domains() (Domain, Domain) {
	a := Domain{Run: 1, Node: "a", Address: "a", Incarnation: 1}
	c := Domain{Run: 1, Node: "b", Address: "b", Incarnation: 1}
	b.step(networkCorpusInput{Action: "register", Domain: &a})
	b.step(networkCorpusInput{Action: "register", Domain: &c})
	return a, c
}

func (b *networkCorpusBuilder) request(r Request, slot uint64) Result {
	if r.RequestID == 0 {
		b.ids[slot]++
		r.RequestID = slot<<48 | b.ids[slot]
	}
	return *b.step(networkCorpusInput{Action: "dispatch", Request: &r}).Result
}

func (b *networkCorpusBuilder) pair(a, c Domain, slot uint64) (uint64, uint64, uint64, int) {
	b.t.Helper()
	l := b.request(Request{Domain: c, Op: "listen"}, slot)
	x := b.request(Request{Domain: a, Op: "dial", Host: c.Address, Port: l.Local.Port}, slot)
	y := b.request(Request{Domain: c, Op: "accept", Handle: l.Handle}, slot)
	for _, r := range []Result{l, x, y} {
		if r.Error != "" || r.State != "completed" {
			b.t.Fatalf("%s pair: %+v", b.corpus.Name, r)
		}
	}
	return l.Handle, x.Handle, y.Handle, l.Local.Port
}

func (b *networkCorpusBuilder) io(domain Domain, op string, handle uint64, data string, count uint32) Result {
	return b.request(Request{Domain: domain, Op: op, Handle: handle, Bytes: []byte(data), Count: count}, 0)
}

func (b *networkCorpusBuilder) link(action string, left, right Domain, nanos uint64, symmetric bool) {
	b.step(networkCorpusInput{Action: action, Left: left.Node, Right: right.Node, Nanos: nanos, Symmetric: symmetric})
}

func (b *networkCorpusBuilder) advance(now uint64) {
	b.step(networkCorpusInput{Action: "advance", Nanos: now})
}

func (b *networkCorpusBuilder) poll(domain Domain, id uint64) Result {
	return b.request(Request{Domain: domain, Op: "poll", RequestID: id}, 0)
}

func (b *networkCorpusBuilder) step(input networkCorpusInput) networkCorpusOutput {
	b.t.Helper()
	if b.model == nil {
		b.t.Fatalf("%s model creation: %s", b.corpus.Name, b.corpus.CreationError)
	}
	m := b.model
	output := networkCorpusOutput{}
	var err error
	switch input.Action {
	case "register":
		err = m.Register(*input.Domain)
	case "revoke":
		err = m.Revoke(*input.Domain, input.Graceful)
	case "dispatch":
		result := m.Dispatch(*input.Request)
		output.Result = &result
	case "cancel":
		err = m.Cancel(*input.Domain, input.RequestID)
	case "partition":
		err = m.Partition(input.Left, input.Right, input.Symmetric)
	case "heal":
		err = m.Heal(input.Left, input.Right, input.Symmetric)
	case "delay":
		err = m.Delay(input.Left, input.Right, input.Nanos, input.Symmetric)
	case "reset":
		err = m.Reset(input.Handle)
	case "advance":
		output.Ready, err = m.Advance(input.Nanos)
	case "ready":
		output.Ready = m.Ready()
	case "finish":
		_, err = m.Finish()
	default:
		b.t.Fatalf("unknown corpus action %q", input.Action)
	}
	if err != nil {
		output.Error = err.Error()
		if err == syscall.ESTALE {
			output.Error = modelErrorMessage(err)
		}
	}
	// Observe settled requests without running Ready: an implicit retry between
	// heal and repartition would change the serialized owner's behavior.
	if input.Action != "ready" && input.Action != "advance" {
		for id, p := range m.pending {
			if p.result != nil {
				output.Ready = append(output.Ready, id)
			}
		}
		sort.Slice(output.Ready, func(i, j int) bool { return output.Ready[i] < output.Ready[j] })
	}
	output.Deadline = m.NextDeadline()
	output.State = networkCorpusState{Now: uint64(m.network.now.UnixNano()), PendingBytes: m.network.pendingBytes, HistoryReserved: m.network.historyReserved}
	for _, domain := range m.network.live {
		output.State.Live = append(output.State.Live, domain)
	}
	sort.Slice(output.State.Live, func(i, j int) bool { return output.State.Live[i].Node < output.State.Live[j].Node })
	for id, p := range m.pending {
		state := "pending"
		if p.result != nil {
			state = "completed"
		}
		output.State.Pending = append(output.State.Pending, networkCorpusPending{RequestID: id, State: state, Accepted: p.accepted})
	}
	sort.Slice(output.State.Pending, func(i, j int) bool { return output.State.Pending[i].RequestID < output.State.Pending[j].RequestID })
	output.Snapshot, err = m.Snapshot()
	if err != nil {
		b.t.Fatal(err)
	}
	output.Transitions = append([]Transition(nil), m.network.transitions[b.history:]...)
	b.history = len(m.network.transitions)
	b.corpus.Steps = append(b.corpus.Steps, networkCorpusStep{Input: input, Expected: output})
	return output
}

func (b *networkCorpusBuilder) finish() networkCorpus {
	b.t.Helper()
	if b.model == nil {
		return b.corpus
	}
	r, err := b.model.Record()
	if err != nil {
		b.t.Fatal(err)
	}
	b.corpus.Final = &r
	if !b.model.network.finished {
		if _, err := b.model.Finish(); err != nil {
			b.corpus.FinalError = err.Error()
		}
	}
	return b.corpus
}

func networkCorpusScenarios(t *testing.T) []networkCorpus {
	t.Helper()
	var corpora []networkCorpus
	add := func(name string, limits Limits, run func(*networkCorpusBuilder, Domain, Domain)) networkCorpus {
		b := newNetworkCorpus(t, name, networkCorpusConfig(limits))
		a, c := b.domains()
		run(b, a, c)
		corpus := b.finish()
		corpora = append(corpora, corpus)
		return corpus
	}
	limits := testLimits()
	limits.StallNanos = 10
	for _, healFirst := range []bool{false, true} {
		add(fmt.Sprintf("equal-time-heal-first-%t", healFirst), limits, func(b *networkCorpusBuilder, a, c Domain) {
			_, x, y, _ := b.pair(a, c, 0)
			b.link("partition", a, c, 0, false)
			b.advance(1000)
			b.io(a, "write", x, "held", 0)
			b.advance(1010)
			b.link("partition", a, c, 0, false)
			if healFirst {
				b.link("heal", a, c, 0, false)
			}
			b.advance(1011)
			if !healFirst {
				b.link("heal", a, c, 0, false)
			}
			b.io(c, "read", y, "", 32)
		})
	}
	add("stall-epoch-invalidation", limits, func(b *networkCorpusBuilder, a, c Domain) {
		_, x, _, _ := b.pair(a, c, 0)
		b.link("partition", a, c, 0, false)
		b.io(a, "write", x, "held", 0)
		b.advance(5)
		b.link("heal", a, c, 0, false)
		b.link("partition", a, c, 0, false)
		b.advance(11)
		b.advance(16)
	})
	reserved := limits
	reserved.StallNanos, reserved.Transitions = 100, 8
	add("autonomous-expiry-reserved-terminal-slot", reserved, func(b *networkCorpusBuilder, a, c Domain) {
		_, x, y, _ := b.pair(a, c, 0)
		b.link("partition", a, c, 0, false)
		b.io(a, "write", x, "held", 0)
		for _, delay := range []uint64{1, 1, 2} {
			b.link("delay", a, c, delay, false)
		}
		b.advance(100)
		b.advance(101)
		for _, end := range []struct {
			domain Domain
			handle uint64
		}{{a, x}, {c, y}} {
			b.request(Request{Domain: end.domain, Op: "deadline", Handle: end.handle}, 0)
			b.io(end.domain, "read", end.handle, "", 1)
			b.io(end.domain, "write", end.handle, "x", 0)
		}
	})
	reserved.Transitions = 11
	timeout := add("simultaneous-pair-expiry", reserved, func(b *networkCorpusBuilder, a, c Domain) {
		_, x, _, _ := b.pair(a, c, 0)
		_, y, _, _ := b.pair(a, c, 0)
		b.link("partition", a, c, 0, true)
		b.io(a, "write", x, "held", 0)
		b.io(a, "write", y, "held", 0)
		b.advance(101)
	})
	for _, ceiling := range []uint64{5, 6} {
		bounded := limits
		bounded.Transitions = ceiling
		add(fmt.Sprintf("held-reservation-ceiling-%d", ceiling), bounded, func(b *networkCorpusBuilder, a, c Domain) {
			_, x, _, _ := b.pair(a, c, 0)
			b.link("partition", a, c, 0, false)
			b.io(a, "write", x, "held", 0)
			b.link("delay", a, c, 1, false)
			b.link("heal", a, c, 0, false)
		})
	}
	bounded := limits
	bounded.Transitions = 5
	add("partition-reservation-atomic-rejection", bounded, func(b *networkCorpusBuilder, a, c Domain) {
		_, x, y, _ := b.pair(a, c, 0)
		b.io(a, "write", x, "queued", 0)
		b.link("partition", a, c, 0, false)
		b.io(c, "read", y, "", 32)
	})
	for _, queued := range []bool{false, true} {
		overflow := limits
		overflow.StallNanos = math.MaxInt64 - 1
		add(fmt.Sprintf("stall-overflow-queued-%t", queued), overflow, func(b *networkCorpusBuilder, a, c Domain) {
			_, x, _, _ := b.pair(a, c, 0)
			if queued {
				b.io(a, "write", x, "queued", 0)
			}
			b.advance(2)
			b.link("partition", a, c, 0, false)
			if !queued {
				b.io(a, "write", x, "held", 0)
			}
		})
	}
	networkCorpusFIFO(add)
	networkCorpusDials(add, limits)
	networkCorpusCapacity(add, limits)
	// Replay inputs are sealed records emitted by the same owner, never hand
	// constructed identities. Mutation cases exercise admission before any call.
	for _, mutation := range []string{"valid", "connection", "kind", "incarnation", "ordinal", "snapshot"} {
		data, err := json.Marshal(timeout.Final)
		if err != nil {
			t.Fatal(err)
		}
		var r Record
		if err := json.Unmarshal(data, &r); err != nil {
			t.Fatal(err)
		}
		var last *Transition
		for i := range r.Transitions {
			if r.Transitions[i].Kind == "reset" && r.Transitions[i].Outcome == "timeout" {
				last = &r.Transitions[i]
				break
			}
		}
		if last == nil {
			t.Fatal("simultaneous expiry corpus has no timeout reset")
		}
		switch mutation {
		case "valid":
		case "connection":
			last.Connection = 0
		case "kind":
			last.Kind = "unknown"
		case "incarnation":
			last.Destination.Incarnation = 0
		case "ordinal":
			last.Ordinal++
		case "snapshot":
			r.Snapshot.Identity = "sha256:corrupted"
		}
		config := timeout.Config
		config.Replay = &r
		b := newNetworkCorpus(t, "held-timeout-replay-admission-"+mutation, config)
		if mutation == "valid" {
			a, c := b.domains()
			_, x, _, _ := b.pair(a, c, 0)
			_, y, _, _ := b.pair(a, c, 0)
			b.link("partition", a, c, 0, true)
			b.io(a, "write", x, "held", 0)
			b.io(a, "write", y, "held", 0)
			b.advance(101)
		} else if b.model != nil {
			t.Fatalf("invalid replay admitted: %s", mutation)
		}
		corpora = append(corpora, b.finish())
	}
	source := add("payload-free-history", limits, func(b *networkCorpusBuilder, a, c Domain) {
		_, x, y, _ := b.pair(a, c, 0)
		b.io(a, "write", x, "abc", 0)
		b.io(c, "read", y, "", 3)
	})
	for _, payload := range []string{"xyz", "xy"} {
		config := source.Config
		config.Replay = source.Final
		b := newNetworkCorpus(t, "payload-free-replay-"+payload, config)
		a, c := b.domains()
		_, x, y, _ := b.pair(a, c, 0)
		b.io(a, "write", x, payload, 0)
		if len(payload) == 3 {
			b.io(c, "read", y, "", 3)
		}
		corpora = append(corpora, b.finish())
	}
	corpora = append(corpora, networkCorpusBroker(t)...)
	corpora = append(corpora, networkCorpusRandom(t)...)
	return corpora
}

type networkCorpusAdd func(string, Limits, func(*networkCorpusBuilder, Domain, Domain)) networkCorpus

func networkCorpusFIFO(add networkCorpusAdd) {
	add("fifo-partition-delay-redundant-heal", testLimits(), func(b *networkCorpusBuilder, a, c Domain) {
		_, x, y, _ := b.pair(a, c, 0)
		b.link("delay", a, c, 10, false)
		b.io(a, "write", x, "before", 0)
		b.advance(20)
		b.link("partition", a, c, 0, false)
		b.io(a, "write", x, "after", 0)
		b.io(c, "write", y, "reverse", 0)
		b.io(a, "read", x, "", 32)
		r := b.io(c, "read", y, "", 32)
		b.link("heal", a, c, 0, false)
		b.io(a, "write", x, "fresh", 0)
		b.advance(25)
		b.link("heal", a, c, 0, false)
		b.advance(29)
		b.poll(c, r.RequestID)
		b.advance(30)
		b.poll(c, r.RequestID)
		b.io(c, "read", y, "", 32)
		b.io(c, "read", y, "", 32)
	})
	for _, op := range []string{"close", "close_write", "close_read"} {
		add("delayed-drain-"+op, testLimits(), func(b *networkCorpusBuilder, a, c Domain) {
			_, x, y, _ := b.pair(a, c, 0)
			b.link("delay", a, c, 10, false)
			b.io(a, "write", x, "delayed", 0)
			b.io(a, op, x, "", 0)
			r := b.io(c, "read", y, "", 32)
			b.advance(10)
			b.poll(c, r.RequestID)
			b.io(c, "read", y, "", 32)
		})
	}
	for _, op := range []string{"close", "close_write", "reset"} {
		add("held-suffix-"+op, testLimits(), func(b *networkCorpusBuilder, a, c Domain) {
			_, x, y, _ := b.pair(a, c, 0)
			b.io(a, "write", x, "received", 0)
			b.io(c, "read", y, "", 2)
			b.link("partition", a, c, 0, false)
			b.io(a, "write", x, "held", 0)
			if op == "reset" {
				b.step(networkCorpusInput{Action: "reset", Handle: x})
				b.step(networkCorpusInput{Action: "reset", Handle: x})
			} else {
				b.io(a, op, x, "", 0)
			}
			b.link("heal", a, c, 0, false)
			b.advance(100)
			b.io(c, "read", y, "", 32)
			b.io(c, "read", y, "", 32)
			b.io(c, "read", y, "", 32)
		})
	}
	add("received-tail-repartition-delayed-suffix", testLimits(), func(b *networkCorpusBuilder, a, c Domain) {
		_, x, y, _ := b.pair(a, c, 0)
		b.io(a, "write", x, "received", 0)
		b.io(c, "read", y, "", 2)
		b.link("delay", a, c, 10, false)
		b.io(a, "write", x, "suffix", 0)
		b.link("partition", a, c, 0, false)
		b.io(c, "read", y, "", 32)
		b.link("heal", a, c, 0, false)
		b.advance(5)
		b.link("partition", a, c, 0, false)
		b.advance(20)
		b.link("heal", a, c, 0, false)
		r := b.io(c, "read", y, "", 32)
		b.advance(30)
		b.poll(c, r.RequestID)
	})
	for _, op := range []string{"close", "close_write"} {
		add("close-before-partition-"+op, testLimits(), func(b *networkCorpusBuilder, a, c Domain) {
			_, x, y, _ := b.pair(a, c, 0)
			b.io(a, "write", x, "received", 0)
			b.io(c, "read", y, "", 2)
			b.link("delay", a, c, 10, false)
			b.io(a, "write", x, "delayed", 0)
			b.io(a, op, x, "", 0)
			b.link("partition", a, c, 0, false)
			b.io(c, "read", y, "", 32)
			b.link("heal", a, c, 0, false)
			b.advance(10)
			b.io(c, "read", y, "", 32)
			b.io(c, "read", y, "", 32)
		})
	}
	for _, symmetric := range []bool{false, true} {
		bounded := testLimits()
		bounded.Transitions = 6
		add(fmt.Sprintf("closed-partition-atomic-rejection-symmetric-%t", symmetric), bounded, func(b *networkCorpusBuilder, a, c Domain) {
			_, x, y, _ := b.pair(a, c, 0)
			b.link("delay", a, c, 10, false)
			b.io(a, "write", x, "delayed", 0)
			b.io(a, "close", x, "", 0)
			b.link("partition", a, c, 0, symmetric)
			b.advance(10)
			b.io(c, "read", y, "", 32)
			b.io(c, "read", y, "", 32)
		})
	}
}

func networkCorpusDials(add networkCorpusAdd, limits Limits) {
	for _, caller := range []uint64{0, 5, 100} {
		add(fmt.Sprintf("pending-dial-caller-%d", caller), limits, func(b *networkCorpusBuilder, a, c Domain) {
			l := b.request(Request{Domain: c, Op: "listen"}, 0)
			b.link("partition", a, c, 0, false)
			r := b.request(Request{Domain: a, Op: "dial", Host: c.Address, Port: l.Local.Port, Deadline: caller}, 0)
			b.advance(b.model.NextDeadline())
			b.poll(a, r.RequestID)
		})
	}
	add("pending-dial-heal-repartition-epoch", limits, func(b *networkCorpusBuilder, a, c Domain) {
		l := b.request(Request{Domain: c, Op: "listen"}, 0)
		b.link("partition", a, c, 0, false)
		r := b.request(Request{Domain: a, Op: "dial", Host: c.Address, Port: l.Local.Port}, 0)
		b.advance(5)
		b.link("partition", a, c, 0, false)
		b.link("heal", a, c, 0, false)
		b.link("partition", a, c, 0, false)
		b.advance(11)
		b.poll(a, r.RequestID)
		b.link("heal", a, c, 0, false)
		b.step(networkCorpusInput{Action: "ready"})
		b.poll(a, r.RequestID)
	})
	for _, terminal := range []string{"cancel", "revoke", "refused", "capacity"} {
		bounded := limits
		if terminal == "capacity" {
			bounded.Connections = 1
		}
		add("pending-dial-heal-"+terminal, bounded, func(b *networkCorpusBuilder, a, c Domain) {
			l, _, _, port := b.pair(a, c, 0)
			b.link("partition", a, c, 0, false)
			r := b.request(Request{Domain: a, Op: "dial", Host: c.Address, Port: port}, 0)
			switch terminal {
			case "cancel":
				b.step(networkCorpusInput{Action: "cancel", Domain: &a, RequestID: r.RequestID})
			case "revoke":
				b.step(networkCorpusInput{Action: "revoke", Domain: &a})
			case "refused":
				b.io(c, "close", l, "", 0)
			case "capacity":
			}
			b.link("heal", a, c, 0, false)
			b.step(networkCorpusInput{Action: "ready"})
			b.poll(a, r.RequestID)
			b.advance(11)
		})
	}
	for _, overflow := range []bool{false, true} {
		bounded := limits
		bounded.Transitions = 2
		if overflow {
			bounded.Transitions, bounded.StallNanos = 100, math.MaxInt64-1
		}
		add(fmt.Sprintf("pending-dial-reservation-overflow-%t", overflow), bounded, func(b *networkCorpusBuilder, a, c Domain) {
			l := b.request(Request{Domain: c, Op: "listen"}, 0)
			b.advance(2)
			b.link("partition", a, c, 0, false)
			b.request(Request{Domain: a, Op: "dial", Host: c.Address, Port: l.Local.Port}, 0)
		})
	}
	add("self-dial-and-refused-dial", limits, func(b *networkCorpusBuilder, a, c Domain) {
		l := b.request(Request{Domain: a, Op: "listen"}, 0)
		b.request(Request{Domain: a, Op: "dial", Host: a.Address, Port: l.Local.Port}, 0)
		b.request(Request{Domain: a, Op: "dial", Host: c.Address, Port: 1}, 0)
		b.advance(5)
		b.request(Request{Domain: a, Op: "dial", Host: a.Address, Port: l.Local.Port, Deadline: 5}, 0)
	})
}

func networkCorpusCapacity(add networkCorpusAdd, limits Limits) {
	bounded := limits
	bounded.ConnectionBytes = 2
	add("partial-write-stall-timeout", bounded, func(b *networkCorpusBuilder, a, c Domain) {
		_, x, y, _ := b.pair(a, c, 0)
		b.link("partition", a, c, 0, false)
		r := b.request(Request{Domain: a, Op: "write", Handle: x, Bytes: []byte("prefix"), Deadline: 100}, 0)
		b.advance(11)
		b.poll(a, r.RequestID)
		b.io(c, "read", y, "", 8)
	})
	add("partial-write-backpressure-drain", bounded, func(b *networkCorpusBuilder, a, c Domain) {
		_, x, y, _ := b.pair(a, c, 0)
		r := b.io(a, "write", x, "prefix", 0)
		for range 3 {
			b.io(c, "read", y, "", 2)
			b.step(networkCorpusInput{Action: "ready"})
		}
		b.poll(a, r.RequestID)
	})
	bounded = limits
	bounded.Bytes = 3
	add("global-capacity-and-read-deadline", bounded, func(b *networkCorpusBuilder, a, c Domain) {
		_, x, y, _ := b.pair(a, c, 0)
		b.io(a, "write", x, "four", 0)
		b.request(Request{Domain: c, Op: "read_deadline", Handle: y, Deadline: 10}, 0)
		r := b.io(c, "read", y, "", 1)
		b.advance(10)
		b.poll(c, r.RequestID)
		b.request(Request{Domain: a, Op: "write_deadline", Handle: x, Deadline: 10}, 0)
		b.io(a, "write", x, "abc", 0)
		b.io(c, "read", y, "", 3)
		b.link("delay", a, c, 20, false)
		b.io(a, "write", x, "abc", 0)
		b.step(networkCorpusInput{Action: "reset", Handle: x})
		b.io(c, "read", y, "", 3)
	})
	bounded = limits
	bounded.Listeners, bounded.Connections = 1, 1
	add("listener-connection-and-request-limits", bounded, func(b *networkCorpusBuilder, a, c Domain) {
		_, x, y, port := b.pair(a, c, 0)
		b.request(Request{Domain: c, Op: "listen"}, 0)
		b.request(Request{Domain: a, Op: "dial", Host: c.Address, Port: port}, 0)
		b.request(Request{Domain: c, Op: "read", Handle: y, Count: MaximumChunkBytes + 1}, 0)
		b.request(Request{Domain: a, Op: "write", Handle: x, Bytes: make([]byte, MaximumChunkBytes+1)}, 0)
		b.request(Request{Domain: a, Op: "deadline", Handle: x, Deadline: math.MaxUint64}, 0)
		b.advance(math.MaxUint64)
		b.advance(2)
		b.advance(1)
		b.link("delay", a, c, math.MaxUint64, false)
		b.step(networkCorpusInput{Action: "cancel", Domain: &a, RequestID: 999})
	})
	for _, graceful := range []bool{false, true} {
		add(fmt.Sprintf("revoke-restart-graceful-%t", graceful), limits, func(b *networkCorpusBuilder, a, c Domain) {
			_, x, y, _ := b.pair(a, c, 0)
			b.link("partition", a, c, 0, false)
			b.io(a, "write", x, "held", 0)
			r := b.io(c, "read", y, "", 1)
			b.step(networkCorpusInput{Action: "revoke", Domain: &c, Graceful: graceful})
			b.step(networkCorpusInput{Action: "ready"})
			b.poll(c, r.RequestID)
			b.step(networkCorpusInput{Action: "register", Domain: &c})
			c.Incarnation++
			b.step(networkCorpusInput{Action: "register", Domain: &c})
			b.io(c, "read", y, "", 1)
			b.advance(11)
			b.io(a, "read", x, "", 1)
			b.step(networkCorpusInput{Action: "finish"})
			b.io(a, "read", x, "", 1)
			b.step(networkCorpusInput{Action: "register", Domain: &c})
		})
	}
}

func networkCorpusRandom(t *testing.T) []networkCorpus {
	t.Helper()
	var result []networkCorpus
	for seed := uint64(1); seed <= 32; seed++ {
		for _, negative := range []bool{false, true} {
			limits := testLimits()
			if negative {
				limits.Bytes = 3
			}
			b := newNetworkCorpus(t, fmt.Sprintf("random-%02d-negative-%t", seed, negative), networkCorpusConfig(limits))
			b.corpus.Seed = seed
			a, c := b.domains()
			_, x, y, _ := b.pair(a, c, 0)
			random := seed
			cycles := 64
			if negative {
				cycles = 8
			}
			for range cycles {
				random = random*6364136223846793005 + 1
				payload := bytes.Repeat([]byte{byte(random >> 32)}, int(random%127+1))
				if negative {
					payload = []byte{byte(random >> 32), byte(random >> 40), byte(random >> 48)}
				}
				b.request(Request{Domain: a, Op: "write", Handle: x, Bytes: payload}, 0)
				if negative {
					stale := a
					stale.Incarnation++
					requests := []Request{{Domain: c, Op: "read", Handle: y, Count: MaximumChunkBytes + 1}, {Domain: stale, Op: "write", Handle: x, Bytes: []byte("x")}, {Domain: a, Op: "dial", Host: c.Address, Port: 1}}
					for step := uint64(0); step < 3; step++ {
						b.request(requests[(step+random%3)%3], 0)
					}
					b.io(a, "write", x, "x", 0)
				}
				b.request(Request{Domain: c, Op: "read", Handle: y, Count: uint32(len(payload))}, 0)
			}
			first := b.finish()
			config := first.Config
			config.Replay = first.Final
			replay := newNetworkCorpus(t, first.Name, config)
			for _, step := range first.Steps {
				if got := replay.step(step.Input); !reflect.DeepEqual(got, step.Expected) {
					t.Fatalf("%s replay differs at %s", first.Name, step.Input.Action)
				}
			}
			if got := replay.finish(); got.FinalError != "" || !reflect.DeepEqual(got.Final, first.Final) {
				t.Fatalf("%s exact replay differs", first.Name)
			}
			result = append(result, first)
		}
	}
	return result
}

func networkCorpusBroker(t *testing.T) []networkCorpus {
	t.Helper()
	var corpora []networkCorpus
	for _, graceful := range []bool{false, true} {
		config := Config{Limits: testLimits()}
		var domains []Domain
		for i, node := range []string{"alpha", "beta", "gamma"} {
			for _, side := range []string{"client", "server"} {
				address := fmt.Sprintf("10.0.0.%d", i+2)
				if side == "client" {
					address = fmt.Sprintf("10.1.0.%d", i+2)
				}
				domains = append(domains, Domain{Run: 1, Node: node + "." + side, Address: address, Incarnation: 1})
			}
		}
		for i, from := range domains {
			config.Nodes = append(config.Nodes, Node{Node: from.Node, Address: from.Address})
			for j, to := range domains {
				// Strict topology admits alpha <-> beta and beta <-> gamma;
				// alpha <-> gamma is deliberately absent, including all sides.
				if i != j && (i/2 == j/2 || i/2-j/2 == 1 || j/2-i/2 == 1) {
					config.Links = append(config.Links, Link{From: from.Node, To: to.Node, Enabled: true})
				}
			}
		}
		b := newNetworkCorpus(t, fmt.Sprintf("broker-strict-topology-graceful-%t", graceful), config)
		for _, domain := range domains {
			b.step(networkCorpusInput{Action: "register", Domain: &domain})
		}
		brokerPair := func(client, server Domain, clientSlot, serverSlot uint64) (uint64, uint64, int) {
			l := b.request(Request{Domain: server, Op: "listen"}, serverSlot)
			x := b.request(Request{Domain: client, Op: "dial", Host: server.Address, Port: l.Local.Port}, clientSlot)
			y := b.request(Request{Domain: server, Op: "accept", Handle: l.Handle}, serverSlot)
			for _, result := range []Result{l, x, y} {
				if result.Error != "" || result.State != "completed" {
					t.Fatalf("%s broker pair: %+v", b.corpus.Name, result)
				}
			}
			return x.Handle, y.Handle, l.Local.Port
		}
		type brokerPairHandles struct {
			client, server Domain
			x, y           uint64
		}
		var pairs []brokerPairHandles
		var port int
		for _, client := range domains[:2] {
			for _, server := range domains[2:4] {
				x, y, p := brokerPair(client, server, 1, 2)
				pairs = append(pairs, brokerPairHandles{client, server, x, y})
				if server == domains[3] {
					port = p
				}
			}
		}
		z, v, _ := brokerPair(domains[2], domains[5], 2, 3)
		b.request(Request{Domain: domains[4], Op: "dial", Host: domains[3].Address, Port: port}, 3)
		b.request(Request{Domain: domains[0], Op: "dial", Host: domains[5].Address, Port: 20000, Deadline: 5}, 1)
		for _, action := range []string{"delay", "partition", "heal"} {
			for _, left := range domains[:2] {
				for _, right := range domains[2:4] {
					b.link(action, left, right, 10, true)
				}
			}
			if action == "partition" {
				for _, pair := range pairs {
					b.request(Request{Domain: pair.client, Op: "write", Handle: pair.x, Bytes: []byte("alpha-held")}, 1)
					b.request(Request{Domain: pair.server, Op: "read", Handle: pair.y, Count: 32}, 2)
				}
				b.request(Request{Domain: domains[2], Op: "write", Handle: z, Bytes: []byte("beta-unaffected")}, 2)
				b.request(Request{Domain: domains[5], Op: "read", Handle: v, Count: 32}, 3)
			}
		}
		b.advance(10)
		b.step(networkCorpusInput{Action: "ready"})
		for _, domain := range domains[2:4] {
			b.step(networkCorpusInput{Action: "revoke", Domain: &domain, Graceful: graceful})
		}
		b.request(Request{Domain: domains[2], Op: "write", Handle: z, Bytes: []byte("stale")}, 2)
		for _, domain := range domains[2:4] {
			domain.Incarnation++
			b.step(networkCorpusInput{Action: "register", Domain: &domain})
		}
		b.advance(100)
		for _, pair := range pairs {
			b.request(Request{Domain: pair.client, Op: "read", Handle: pair.x, Count: 32}, 1)
		}
		b.request(Request{Domain: domains[5], Op: "read", Handle: v, Count: 32}, 3)
		corpora = append(corpora, b.finish())
	}
	return corpora
}

func TestNetworkCorpusIsCurrent(t *testing.T) {
	regenerate := os.Getenv("GOMAD_NETWORK_CORPUS_REGENERATE") == "1"
	if regenerate {
		if err := os.MkdirAll(networkCorpusDirectory, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	wantNames := make(map[string]bool)
	for _, corpus := range networkCorpusScenarios(t) {
		name := corpus.Name + ".json"
		if wantNames[name] {
			t.Fatalf("duplicate corpus %s", name)
		}
		wantNames[name] = true
		data, err := json.Marshal(corpus)
		if err != nil {
			t.Fatal(err)
		}
		data = append(data, '\n')
		path := filepath.Join(networkCorpusDirectory, name)
		if regenerate {
			if err := os.WriteFile(path, data, 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, data) {
			t.Errorf("network corpus %s is stale; review Go-owner changes, then run GOMAD_NETWORK_CORPUS_REGENERATE=1 go test -tags test_dep -run '^TestNetworkCorpusIsCurrent$' ./internal/model/network", name)
		}
	}
	entries, err := os.ReadDir(networkCorpusDirectory)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".json" && !wantNames[entry.Name()] {
			t.Errorf("unexpected network corpus %s", entry.Name())
		}
	}
}
