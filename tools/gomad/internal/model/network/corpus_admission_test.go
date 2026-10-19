package network

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"testing"
)

func networkCorpusSeal(t *testing.T, record *Record) {
	t.Helper()
	encoded, err := encodeSimulationTransitions(record.Transitions)
	if err != nil {
		t.Fatal(err)
	}
	record.Snapshot.TransitionSHA256 = fmt.Sprintf("sha256:%x", sha256.Sum256(append([]byte("gomad-simulation-network-transitions/v3\x00"), encoded...)))
	record.Snapshot.Identity = ""
	encoded, err = encodeSimulationSnapshotIdentity(record.Snapshot)
	if err != nil {
		t.Fatal(err)
	}
	record.Snapshot.Identity = fmt.Sprintf("sha256:%x", sha256.Sum256(append([]byte("gomad-simulation-network-snapshot/v3\x00"), encoded...)))
}

func networkCorpusCreationError(t *testing.T, b *networkCorpusBuilder, want string) {
	t.Helper()
	if b.corpus.CreationError != want {
		t.Fatalf("%s creation error: got %q want %q", b.corpus.Name, b.corpus.CreationError, want)
	}
}

func networkCorpusFourConfig() Config {
	return Config{Nodes: []Node{{Node: "a", Address: "a"}, {Node: "b", Address: "b"}, {Node: "c", Address: "c"}, {Node: "d", Address: "d"}}, Links: []Link{{From: "a", To: "b", Enabled: true}, {From: "b", To: "a", Enabled: true}, {From: "c", To: "d", Enabled: true}, {From: "d", To: "c", Enabled: true}}, Limits: Limits{Listeners: 128, Connections: 128, Bytes: 1024, ConnectionBytes: 64, StallNanos: 10, Transitions: 8192}}
}

func networkCorpusFour(t *testing.T, name string, config Config) (*networkCorpusBuilder, []Domain) {
	t.Helper()
	b := newNetworkCorpus(t, "admission-"+name, config)
	var domains []Domain
	if b.model == nil {
		return b, nil
	}
	for _, node := range config.Nodes {
		domain := Domain{Run: 1, Node: node.Node, Address: node.Address, Incarnation: 1}
		domains = append(domains, domain)
		b.step(networkCorpusInput{Action: "register", Domain: &domain})
	}
	return b, domains
}

func networkCorpusBacklog(b *networkCorpusBuilder, a, c Domain) uint64 {
	l := b.request(Request{Domain: c, Op: "listen", RequestID: 1}, 0)
	for id := uint64(100); id < 164; id++ {
		b.request(Request{Domain: a, Op: "dial", RequestID: id, Host: c.Address, Port: l.Local.Port}, 0)
	}
	y := b.request(Request{Domain: c, Op: "accept", RequestID: 2, Handle: l.Handle}, 0)
	b.request(Request{Domain: a, Op: "dial", RequestID: 164, Host: c.Address, Port: l.Local.Port}, 0)
	return y.Handle
}

func networkCorpusAdmission(t *testing.T) []networkCorpus {
	var corpora []networkCorpus
	add := func(name string, config Config, run func(*networkCorpusBuilder, []Domain)) networkCorpus {
		b, d := networkCorpusFour(t, name, config)
		if b.model != nil {
			run(b, d)
		}
		corpus := b.finish()
		corpora = append(corpora, corpus)
		return corpus
	}
	config := networkCorpusFourConfig()
	add("two-pair-slot-handles-and-lifecycle", config, func(b *networkCorpusBuilder, d []Domain) {
		listeners := []Result{b.request(Request{Domain: d[1], Op: "listen", RequestID: 2<<48 | 1}, 0), b.request(Request{Domain: d[3], Op: "listen", RequestID: 4<<48 | 1}, 0)}
		for i, left := range []int{0, 2} {
			b.request(Request{Domain: d[left+1], Op: "accept", RequestID: uint64(left+2)<<48 | 2, Handle: listeners[i].Handle}, 0)
		}
		x := b.request(Request{Domain: d[0], Op: "dial", RequestID: 1<<48 | 1, Host: "b", Port: 20000}, 0)
		z := b.request(Request{Domain: d[2], Op: "dial", RequestID: 3<<48 | 1, Host: "d", Port: 20000}, 0)
		b.step(networkCorpusInput{Action: "ready"})
		y := b.poll(d[1], 2<<48|2)
		v := b.poll(d[3], 4<<48|2)
		b.link("partition", d[0], d[1], 0, false)
		b.request(Request{Domain: d[0], Op: "write", RequestID: 1<<48 | 2, Handle: x.Handle, Bytes: []byte("held")}, 0)
		b.request(Request{Domain: d[2], Op: "write", RequestID: 3<<48 | 2, Handle: z.Handle, Bytes: []byte("live")}, 0)
		b.request(Request{Domain: d[3], Op: "read", RequestID: 4<<48 | 3, Handle: v.Handle, Count: 8}, 0)
		b.request(Request{Domain: d[1], Op: "read", RequestID: 2<<48 | 3, Handle: y.Handle, Count: 8}, 0)
		b.step(networkCorpusInput{Action: "cancel", Domain: &d[3], RequestID: 2<<48 | 3})
		b.step(networkCorpusInput{Action: "cancel", Domain: &d[1], RequestID: 2<<48 | 3})
		b.poll(d[1], 2<<48|3)
		b.link("heal", d[0], d[1], 0, false)
		b.request(Request{Domain: d[1], Op: "read", RequestID: 2<<48 | 4, Handle: y.Handle, Count: 8}, 0)
		b.step(networkCorpusInput{Action: "revoke", Domain: &d[0]})
		b.request(Request{Domain: d[1], Op: "read", RequestID: 2<<48 | 5, Handle: y.Handle, Count: 1}, 0)
		fresh := d[0]
		fresh.Incarnation++
		b.step(networkCorpusInput{Action: "register", Domain: &fresh})
		b.request(Request{Domain: fresh, Op: "close", RequestID: 1<<48 | 3, Handle: x.Handle}, 0)
		b.request(Request{Domain: fresh, Op: "dial", RequestID: 1<<48 | 4, Host: "b", Port: 20000}, 0)
		b.step(networkCorpusInput{Action: "revoke", Domain: &d[3], Graceful: true})
		b.request(Request{Domain: d[2], Op: "read", RequestID: 3<<48 | 3, Handle: z.Handle, Count: 1}, 0)
	})
	add("backlog-blocks-before-connection-allocation", config, func(b *networkCorpusBuilder, d []Domain) {
		l := b.request(Request{Domain: d[1], Op: "listen", RequestID: 1}, 0)
		for id := uint64(100); id < 164; id++ {
			b.request(Request{Domain: d[0], Op: "dial", RequestID: id, Host: "b", Port: 20000}, 0)
		}
		b.request(Request{Domain: d[0], Op: "dial", RequestID: 1000, Host: "b", Port: 20000}, 0)
		b.step(networkCorpusInput{Action: "ready"})
		b.request(Request{Domain: d[1], Op: "accept", RequestID: 2, Handle: l.Handle}, 0)
		b.step(networkCorpusInput{Action: "ready"})
		b.poll(d[0], 1000)
	})
	global := config
	global.Limits.ConnectionBytes, global.Limits.Bytes = 2, 1
	add("global-capacity-precedes-ready-overflow", global, func(b *networkCorpusBuilder, d []Domain) {
		_, x, _, _ := b.pair(d[0], d[1], 0)
		b.advance(1)
		b.io(d[0], "write", x, "a", 0)
		b.link("delay", d[0], d[1], math.MaxInt64, false)
		b.io(d[0], "write", x, "b", 0)
	})
	add("full-close-snapshot-marker-until-reset", config, func(b *networkCorpusBuilder, d []Domain) {
		_, x, y, _ := b.pair(d[0], d[1], 0)
		b.io(d[0], "close", x, "", 0)
		b.io(d[1], "close", y, "", 0)
		b.step(networkCorpusInput{Action: "reset", Handle: x})
	})
	add("stale-domain-registration-and-revocation", config, func(b *networkCorpusBuilder, d []Domain) {
		for _, bad := range []Domain{{Run: 2, Node: "a", Address: "a", Incarnation: 1}, {Run: 1, Node: "a", Address: "wrong", Incarnation: 1}, {Run: 1, Node: "missing", Address: "missing", Incarnation: 1}, d[0]} {
			b.step(networkCorpusInput{Action: "register", Domain: &bad})
		}
		stale := d[0]
		stale.Incarnation = 2
		b.step(networkCorpusInput{Action: "revoke", Domain: &stale})
	})
	add("oversize-chunks-precede-stale-domain", config, func(b *networkCorpusBuilder, d []Domain) {
		stale := d[0]
		stale.Incarnation++
		b.request(Request{Domain: stale, Op: "read", Count: MaximumChunkBytes + 1}, 0)
		b.request(Request{Domain: stale, Op: "write", Bytes: make([]byte, MaximumChunkBytes+1)}, 0)
	})
	add("clock-ceiling-monotone-and-caller-admission", config, func(b *networkCorpusBuilder, d []Domain) {
		for _, ceiling := range []uint64{0, 5, 100} {
			b.step(networkCorpusInput{Action: "clock_ceiling", Nanos: ceiling})
		}
		b.request(Request{Domain: d[0], Op: "listen", Deadline: 6}, 0)
		b.advance(6)
	})
	add("clock-ceiling-cannot-exclude-queue-or-listener-deadline", config, func(b *networkCorpusBuilder, d []Domain) {
		l, x, y, _ := b.pair(d[0], d[1], 0)
		b.link("delay", d[0], d[1], 10, false)
		b.io(d[0], "write", x, "x", 0)
		b.step(networkCorpusInput{Action: "clock_ceiling", Nanos: 5})
		b.advance(6)
		b.request(Request{Domain: d[1], Op: "deadline", Handle: l, Deadline: 20}, 0)
		b.step(networkCorpusInput{Action: "clock_ceiling", Nanos: 15})
		b.advance(16)
		b.request(Request{Domain: d[1], Op: "deadline", Handle: l}, 0)
		b.io(d[1], "read", y, "", 1)
		b.step(networkCorpusInput{Action: "clock_ceiling", Nanos: 16})
	})
	add("advance-before-preserves-equal-ready-time", config, func(b *networkCorpusBuilder, d []Domain) {
		_, x, y, _ := b.pair(d[0], d[1], 0)
		b.link("delay", d[0], d[1], 10, false)
		b.io(d[0], "write", x, "x", 0)
		r := b.io(d[1], "read", y, "", 1)
		b.step(networkCorpusInput{Action: "advance_before", Nanos: 10})
		b.poll(d[1], r.RequestID)
		b.advance(10)
		b.poll(d[1], r.RequestID)
	})
	limited := config
	limited.Limits.Listeners = 1
	add("listen-collision-precedes-history-capacity", limited, func(b *networkCorpusBuilder, d []Domain) {
		for _, port := range []int{20000, 20000, 20001} {
			b.request(Request{Domain: d[0], Op: "listen", Port: port}, 0)
		}
	})
	for _, transitions := range []uint64{67, 8192} {
		c := config
		c.Limits.Transitions = transitions
		add(fmt.Sprintf("cancel-backlog-dial-history-%d", transitions), c, func(b *networkCorpusBuilder, d []Domain) {
			networkCorpusBacklog(b, d[0], d[1])
			id := uint64(1<<48 | 1000)
			b.request(Request{Domain: d[0], Op: "dial", RequestID: id, Host: d[1].Address, Port: 20000}, 0)
			b.step(networkCorpusInput{Action: "cancel", Domain: &d[0], RequestID: id})
			b.poll(d[0], id)
		})
	}
	add("group-duration-ignored-unless-delay", config, func(b *networkCorpusBuilder, _ []Domain) {
		for _, action := range []string{"partition", "heal", "delay"} {
			b.step(networkCorpusInput{Action: "group", LeftNodes: []string{"a"}, RightNodes: []string{"b"}, LinkAction: action, Nanos: math.MaxUint64, Symmetric: action != "partition"})
		}
	})
	add("reset-after-finish-keeps-pair-and-records-crash", config, func(b *networkCorpusBuilder, d []Domain) {
		_, x, _, _ := b.pair(d[0], d[1], 0)
		b.step(networkCorpusInput{Action: "finish"})
		b.step(networkCorpusInput{Action: "reset", Handle: x})
	})
	for _, before := range []bool{false, true} {
		add(fmt.Sprintf("dial-caller-deadline-advance-before-%t", before), config, func(b *networkCorpusBuilder, d []Domain) {
			b.link("partition", d[0], d[1], 0, false)
			r := b.request(Request{Domain: d[0], Op: "dial", Host: d[1].Address, Port: 20000, Deadline: 5}, 0)
			action := "advance"
			if before {
				action = "advance_before"
			}
			b.step(networkCorpusInput{Action: action, Nanos: 100})
			b.poll(d[0], r.RequestID)
		})
	}
	add("dial-clock-ceiling-heal-repartition-rejection", config, func(b *networkCorpusBuilder, d []Domain) {
		b.step(networkCorpusInput{Action: "clock_ceiling", Nanos: 20})
		b.link("partition", d[0], d[1], 0, false)
		b.request(Request{Domain: d[0], Op: "dial", Host: d[1].Address, Port: 20000}, 0)
		b.step(networkCorpusInput{Action: "clock_ceiling", Nanos: 10})
		b.advance(10)
		b.link("heal", d[0], d[1], 0, false)
		b.link("partition", d[0], d[1], 0, false)
	})
	add("pending-dial-poll-keeps-old-wait-until-ready", config, func(b *networkCorpusBuilder, d []Domain) {
		b.link("partition", d[0], d[1], 0, false)
		r := b.request(Request{Domain: d[0], Op: "dial", Host: d[1].Address, Port: 20000}, 0)
		b.advance(5)
		b.link("heal", d[0], d[1], 0, false)
		b.link("partition", d[0], d[1], 0, false)
		b.poll(d[0], r.RequestID)
		b.step(networkCorpusInput{Action: "ready"})
		b.poll(d[0], r.RequestID)
	})
	for _, caller := range []bool{false, true} {
		c := config
		c.Limits.ConnectionBytes = 2
		add(fmt.Sprintf("partial-write-caller-versus-connection-deadline-%t", caller), c, func(b *networkCorpusBuilder, d []Domain) {
			_, x, _, _ := b.pair(d[0], d[1], 0)
			b.advance(5)
			if !caller {
				b.request(Request{Domain: d[0], Op: "write_deadline", Handle: x, Deadline: 5}, 0)
			}
			request := Request{Domain: d[0], Op: "write", Handle: x, Bytes: []byte("abcdef")}
			if caller {
				request.Deadline = 5
			}
			b.request(request, 0)
		})
	}
	for _, capacity := range []bool{false, true} {
		for _, consume := range []bool{false, true} {
			c := config
			c.Limits.Connections = 1
			add(fmt.Sprintf("completed-dial-no-rearm-capacity-%t-consumed-%t", capacity, consume), c, func(b *networkCorpusBuilder, d []Domain) {
				if capacity {
					b.pair(d[0], d[1], 0)
				}
				b.link("partition", d[0], d[1], 0, false)
				r := b.request(Request{Domain: d[0], Op: "dial", Host: d[1].Address, Port: 20000}, 0)
				b.link("heal", d[0], d[1], 0, false)
				b.step(networkCorpusInput{Action: "ready"})
				if consume {
					b.poll(d[0], r.RequestID)
				}
				b.link("partition", d[0], d[1], 0, false)
				b.advance(100)
				if !consume {
					b.poll(d[0], r.RequestID)
				}
			})
		}
	}
	for _, transitions := range []uint64{6, 7} {
		c := config
		c.Limits.Transitions = transitions
		c.Links = append(c.Links, Link{From: "c", To: "b"})
		sort.Slice(c.Links, func(i, j int) bool {
			return c.Links[i].From+"\x00"+c.Links[i].To < c.Links[j].From+"\x00"+c.Links[j].To
		})
		add(fmt.Sprintf("mixed-dial-pair-atomic-partition-%d", transitions), c, func(b *networkCorpusBuilder, d []Domain) {
			_, x, _, _ := b.pair(d[0], d[1], 1)
			b.request(Request{Domain: d[2], Op: "dial", RequestID: 3<<48 | 1, Host: d[1].Address, Port: 20000}, 0)
			b.request(Request{Domain: d[0], Op: "write", RequestID: 1<<48 | 4, Handle: x, Bytes: []byte("held")}, 0)
			b.link("partition", d[0], d[1], 0, false)
			if transitions == 7 {
				b.advance(11)
			}
		})
	}
	add("two-pair-stalls-revocation-and-global-request-ids", config, func(b *networkCorpusBuilder, d []Domain) {
		for _, left := range []int{0, 2} {
			slot := uint64(left + 1)
			l := b.request(Request{Domain: d[left+1], Op: "listen", RequestID: (slot+1)<<48 | 1}, 0)
			x := b.request(Request{Domain: d[left], Op: "dial", RequestID: slot<<48 | 1, Host: d[left+1].Address, Port: l.Local.Port}, 0)
			y := b.request(Request{Domain: d[left+1], Op: "accept", RequestID: (slot+1)<<48 | 2, Handle: l.Handle}, 0)
			b.link("partition", d[left], d[left+1], 0, false)
			b.request(Request{Domain: d[left], Op: "write", RequestID: slot<<48 | 2, Handle: x.Handle, Bytes: []byte("held")}, 0)
			b.request(Request{Domain: d[left+1], Op: "read", RequestID: (slot+1)<<48 | 3, Handle: y.Handle, Count: 4}, 0)
		}
		b.step(networkCorpusInput{Action: "revoke", Domain: &d[0]})
		fresh := d[0]
		fresh.Incarnation = 2
		b.step(networkCorpusInput{Action: "register", Domain: &fresh})
		b.advance(11)
		b.poll(d[1], 2<<48|3)
		b.poll(d[3], 4<<48|3)
	})
	source := add("replay-source", config, func(b *networkCorpusBuilder, d []Domain) {
		_, x, y, _ := b.pair(d[0], d[1], 0)
		b.io(d[0], "write", x, "abc", 0)
		b.io(d[1], "read", y, "", 3)
	})
	for _, mutation := range []string{"transition-outcome", "listener-timeout", "listener-reset", "node-port", "link-order", "listener-port-zero", "connection-incarnation", "reset-retains-bytes", "connection-bytes-capacity"} {
		var record Record
		encoded, err := json.Marshal(source.Final)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(encoded, &record); err != nil {
			t.Fatal(err)
		}
		wantError := "simulation network replay transition is invalid or unordered"
		switch mutation {
		case "transition-outcome":
			record.Transitions[2].Outcome = "invented"
		case "listener-timeout":
			record.Transitions[0].Outcome = "timeout"
		case "listener-reset":
			record.Transitions[0].Kind = "reset"
		case "node-port":
			record.Snapshot.Nodes[0].NextClientPort = 0
			wantError = "simulation network replay node snapshot is invalid or unordered"
		case "link-order":
			record.Snapshot.Links[0], record.Snapshot.Links[1] = record.Snapshot.Links[1], record.Snapshot.Links[0]
			wantError = "simulation network replay link snapshot is invalid or unordered"
		case "listener-port-zero":
			record.Snapshot.Listeners[0].Endpoint.Port = 0
			wantError = "simulation network replay listener snapshot is invalid or unordered"
		case "connection-incarnation":
			record.Snapshot.Connections[0].Client.Incarnation = 0
			wantError = "simulation network replay connection snapshot is invalid or unordered"
		case "reset-retains-bytes":
			record.Snapshot.Connections[0].Closed = true
			record.Snapshot.Connections[0].Reset = true
			record.Snapshot.Connections[0].ClientBytes = 1
			wantError = "simulation network replay reset connection retains bytes"
		case "connection-bytes-capacity":
			record.Snapshot.Connections[0].ClientBytes = 1 << 30
			wantError = "network resources exhausted"
		}
		networkCorpusSeal(t, &record)
		c := config
		c.Replay = &record
		b := newNetworkCorpus(t, "admission-resealed-"+mutation, c)
		networkCorpusCreationError(t, b, wantError)
		if b.model != nil {
			t.Fatalf("invalid resealed record admitted: %s", mutation)
		}
		corpora = append(corpora, b.finish())
	}
	for _, closed := range [][]bool{{true, true}, {true, false}, {true, true, true, false}, {false, true}, {false, false}} {
		record := *source.Final
		record.Snapshot.Listeners = nil
		for _, value := range closed {
			listener := source.Final.Snapshot.Listeners[0]
			listener.Closed = value
			record.Snapshot.Listeners = append(record.Snapshot.Listeners, listener)
		}
		networkCorpusSeal(t, &record)
		c := config
		c.Replay = &record
		b := newNetworkCorpus(t, fmt.Sprintf("admission-replay-listener-history-%v", closed), c)
		wantError := ""
		if !closed[0] {
			wantError = "simulation network replay listener snapshot is invalid or unordered"
		}
		networkCorpusCreationError(t, b, wantError)
		corpora = append(corpora, b.finish())
	}
	for _, port := range []uint64{0, 65536, math.MaxUint64} {
		record := *source.Final
		record.Snapshot.Listeners = append([]simulationListenerSnapshot(nil), source.Final.Snapshot.Listeners...)
		record.Snapshot.Listeners[0].Endpoint.Port = port
		networkCorpusSeal(t, &record)
		c := config
		c.Replay = &record
		b := newNetworkCorpus(t, fmt.Sprintf("admission-replay-listener-port-%d", port), c)
		networkCorpusCreationError(t, b, "simulation network replay listener snapshot is invalid or unordered")
		corpora = append(corpora, b.finish())
	}
	record := *source.Final
	first := source.Final.Snapshot.Listeners[0]
	first.Endpoint.Port = 7234
	first.Closed = true
	second := first
	second.Endpoint.Port = 7233
	record.Snapshot.Listeners = []simulationListenerSnapshot{first, second}
	networkCorpusSeal(t, &record)
	unordered := config
	unordered.Replay = &record
	bUnordered := newNetworkCorpus(t, "admission-replay-listener-history-unordered", unordered)
	networkCorpusCreationError(t, bUnordered, "simulation network replay listener snapshot is invalid or unordered")
	corpora = append(corpora, bUnordered.finish())
	for _, divergent := range []bool{false, true} {
		c := config
		c.Links = append([]Link(nil), config.Links...)
		c.Links[0].Enabled = false
		admit := func(b *networkCorpusBuilder, d []Domain) {
			for _, id := range []uint64{10, 11} {
				b.request(Request{Domain: d[0], Op: "dial", RequestID: id, Host: d[1].Address, Port: 20000 + int(id-10)}, 0)
			}
		}
		b, d := networkCorpusFour(t, "expiry-source", c)
		admit(b, d)
		b.advance(11)
		record := *b.finish().Final
		if divergent {
			for i := range record.Transitions {
				if record.Transitions[i].Destination.Port == 20001 {
					record.Transitions[i].Outcome = "deadline"
				}
			}
			networkCorpusSeal(t, &record)
		}
		c.Replay = &record
		expiry := add(fmt.Sprintf("expiry-revoke-id-reuse-divergent-%t", divergent), c, func(b *networkCorpusBuilder, d []Domain) {
			admit(b, d)
			b.advance(11)
			wantError := ""
			if divergent {
				wantError = "simulation network replay diverged"
			}
			if got := b.corpus.Steps[len(b.corpus.Steps)-1].Expected.Error; got != wantError {
				t.Fatalf("%s expiry error: got %q want %q", b.corpus.Name, got, wantError)
			}
			b.step(networkCorpusInput{Action: "revoke", Domain: &d[0]})
			fresh := d[0]
			fresh.Incarnation = 2
			b.step(networkCorpusInput{Action: "register", Domain: &fresh})
			b.request(Request{Domain: fresh, Op: "dial", RequestID: 10, Host: d[1].Address, Port: 20000}, 0)
		})
		if expiry.CreationError != "" {
			t.Fatalf("%s must reach replay execution: %s", expiry.Name, expiry.CreationError)
		}
	}
	corpora = append(corpora, networkCorpusReplayGroups(t)...)
	corpora = append(corpora, networkCorpusSeparators(t)...)
	return corpora
}

func networkCorpusReplayGroups(t *testing.T) []networkCorpus {
	var corpora []networkCorpus
	for _, partial := range []bool{false, true} {
		config := networkCorpusFourConfig()
		if partial {
			config.Links = append(config.Links, Link{From: "a", To: "d", Enabled: true}, Link{From: "c", To: "b", Enabled: true})
			sort.Slice(config.Links, func(i, j int) bool {
				return config.Links[i].From+"\x00"+config.Links[i].To < config.Links[j].From+"\x00"+config.Links[j].To
			})
		}
		setup := func(name string, config Config) (*networkCorpusBuilder, []Domain, uint64) {
			b, d := networkCorpusFour(t, name, config)
			_, x, _, _ := b.pair(d[0], d[1], 0)
			if partial {
				b.io(d[0], "write", x, "held", 0)
			}
			return b, d, x
		}
		b, _, _ := setup("group-source", config)
		left := []string{"a"}
		if partial {
			left = append(left, "c")
		}
		b.step(networkCorpusInput{Action: "group", LeftNodes: left, RightNodes: []string{"b"}, LinkAction: "partition"})
		config.Replay = b.finish().Final
		b, _, _ = setup(fmt.Sprintf("group-replay-rollback-partial-%t", partial), config)
		wrong := networkCorpusInput{Action: "group", LeftNodes: []string{"a"}, RightNodes: []string{"b"}, LinkAction: "heal"}
		if partial {
			wrong.LinkAction = "partition"
			wrong.RightNodes = append(wrong.RightNodes, "d")
		}
		b.step(wrong)
		b.step(networkCorpusInput{Action: "group", LeftNodes: left, RightNodes: []string{"b"}, LinkAction: "partition"})
		corpora = append(corpora, b.finish())
	}
	config := networkCorpusFourConfig()
	b, d := networkCorpusFour(t, "reset-replay-source", config)
	y := networkCorpusBacklog(b, d[0], d[1])
	b.request(Request{Domain: d[0], Op: "write", Handle: 2, RequestID: 200, Bytes: []byte("abc")}, 0)
	b.request(Request{Domain: d[1], Op: "read", Handle: y, RequestID: 201, Count: 3}, 0)
	b.step(networkCorpusInput{Action: "reset", Handle: 2})
	config.Replay = b.finish().Final
	b, d = networkCorpusFour(t, "reset-wakes-backlog-after-replay-divergence", config)
	networkCorpusBacklog(b, d[0], d[1])
	b.request(Request{Domain: d[0], Op: "dial", RequestID: 1000, Host: d[1].Address, Port: 20000}, 0)
	b.request(Request{Domain: d[0], Op: "write", Handle: 2, RequestID: 200, Bytes: []byte("x")}, 0)
	b.step(networkCorpusInput{Action: "ready"})
	b.step(networkCorpusInput{Action: "reset", Handle: 2})
	b.step(networkCorpusInput{Action: "ready"})
	b.poll(d[0], 1000)
	corpora = append(corpora, b.finish())
	return corpora
}

func networkCorpusSeparators(t *testing.T) []networkCorpus {
	var corpora []networkCorpus
	config := networkCorpusFourConfig()
	config.Nodes = []Node{{Node: "a", Address: "a"}, {Node: "a\x00", Address: "a0"}, {Node: "z", Address: "z"}}
	config.Links = []Link{{From: "a\x00", To: "z", Enabled: true}, {From: "a", To: "z", Enabled: true}}
	b, d := networkCorpusFour(t, "separator-node-snapshot-order", config)
	b.request(Request{Domain: d[0], Op: "listen"}, 0)
	b.request(Request{Domain: d[1], Op: "listen"}, 0)
	corpora = append(corpora, b.finish())
	config.Nodes = []Node{{Node: "a", Address: "1"}, {Node: "a\x00b", Address: "2"}, {Node: "b\x00c", Address: "3"}, {Node: "c", Address: "4"}}
	config.Links = []Link{{From: "a", To: "b\x00c", Enabled: true}}
	b, _ = networkCorpusFour(t, "separator-link-alias-identity", config)
	b.step(networkCorpusInput{Action: "group", LeftNodes: []string{"a\x00b"}, RightNodes: []string{"c"}, LinkAction: "partition"})
	corpora = append(corpora, b.finish())
	return corpora
}
