package wasi

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	netmodel "go.temporal.io/server/tools/gomad/internal/model/network"
	"go.temporal.io/server/tools/gomad/internal/model/world"
)

const networkClusterCorpusDirectory = "../../../rust/wasmhost/tests/testdata/network_cluster_cases"

type networkClusterInput struct {
	Action          string              `json:"action"`
	Node            string              `json:"node,omitempty"`
	Incarnation     uint64              `json:"incarnation,omitempty"`
	Graceful        bool                `json:"graceful,omitempty"`
	Nanos           uint64              `json:"nanos,omitempty"`
	LinkAction      string              `json:"link_action,omitempty"`
	From            string              `json:"from,omitempty"`
	To              string              `json:"to,omitempty"`
	Symmetric       bool                `json:"symmetric,omitempty"`
	Request         *netmodel.Request   `json:"request,omitempty"`
	Data            []byte              `json:"data,omitempty"`
	Inactive        bool                `json:"inactive,omitempty"`
	TranscriptBytes uint64              `json:"transcript_bytes,omitempty"`
	Epoch           uint64              `json:"epoch,omitempty"`
	Operation       *networkV2Operation `json:"operation,omitempty"`
	Quiescence      *world.Quiescence   `json:"quiescence,omitempty"`
}

type networkClusterStep struct {
	Input             networkClusterInput `json:"input"`
	Expected          json.RawMessage     `json:"expected"`
	ResourceExhausted bool                `json:"resource_exhausted"`
}

type networkClusterCorpus struct {
	Schema                string               `json:"schema"`
	Implementation        string               `json:"implementation_sha256"`
	NetworkImplementation string               `json:"network_implementation_sha256"`
	Name                  string               `json:"name"`
	Config                any                  `json:"config"`
	Nodes                 []string             `json:"nodes"`
	CreationError         string               `json:"creation_error"`
	Initial               json.RawMessage      `json:"initial,omitempty"`
	Steps                 []networkClusterStep `json:"steps"`
}

type networkClusterBuilder struct {
	t            *testing.T
	broker       *Broker
	environments map[string]*Environment
	corpus       networkClusterCorpus
}

func newNetworkClusterCorpus(t *testing.T, name string, config BrokerConfig) *networkClusterBuilder {
	t.Helper()
	links := make([]map[string]any, 0, len(config.Links))
	for _, link := range config.Links {
		links = append(links, map[string]any{"from": link.From, "to": link.To, "enabled": link.Enabled, "delay_nanos": link.DelayNanos})
	}
	faults := make([]map[string]any, 0, len(config.Faults))
	for _, fault := range config.Faults {
		faults = append(faults, map[string]any{"name": fault.Name, "action": fault.Action, "from": fault.From, "to": fault.To, "at_nanos": fault.AtNanos, "delay_nanos": fault.DelayNanos, "symmetric": fault.Symmetric})
	}
	b := &networkClusterBuilder{t: t, environments: map[string]*Environment{}, corpus: networkClusterCorpus{
		Schema: "gomad.network-cluster-corpus/v1", Implementation: ImplementationSHA256(), NetworkImplementation: netmodel.ImplementationSHA256(), Name: name,
		Config: map[string]any{"model": config.Network, "addresses": config.Addresses, "links": links, "faults": faults, "strict_topology": config.StrictTopology},
	}}
	for _, node := range config.Nodes {
		b.corpus.Nodes = append(b.corpus.Nodes, node.ID)
	}
	var err error
	b.broker, err = NewBroker(config)
	if err != nil {
		b.corpus.CreationError = err.Error()
		return b
	}
	for _, node := range b.broker.order {
		e := streamEnvironment(t, config.Network)
		if err := b.broker.network.attach(b.broker.sessions[node], e); err != nil {
			t.Fatal(err)
		}
		b.environments[node] = e
	}
	b.corpus.Initial = b.observe(nil, nil, nil)
	return b
}

func (b *networkClusterBuilder) observe(err error, result *netmodel.Result, data []byte) json.RawMessage {
	b.t.Helper()
	record, recordErr := b.broker.NetworkRecord()
	if recordErr != nil {
		b.t.Fatal(recordErr)
	}
	wantSnapshot, wantSnapshotErr := b.broker.network.model.Snapshot()
	if wantSnapshotErr != nil {
		b.t.Fatal(wantSnapshotErr)
	}
	snapshot, snapshotErr := b.broker.NetworkSnapshot()
	if snapshotErr != nil || !reflect.DeepEqual(snapshot, wantSnapshot) {
		b.t.Fatalf("broker NetworkSnapshot differs from the chronological owner snapshot: got %+v, want %+v, error %v", snapshot, wantSnapshot, snapshotErr)
	}
	message := ""
	if err != nil {
		message = err.Error()
	}
	value := map[string]any{"error": message, "record": record, "next_deadline": b.broker.network.deadline(), "model_deadline": b.broker.network.model.NextDeadline()}
	if result != nil {
		value["result"] = result
	}
	if data != nil {
		value["data"] = data
	}
	encoded, marshalErr := json.Marshal(value)
	if marshalErr != nil {
		b.t.Fatal(marshalErr)
	}
	return encoded
}

func (b *networkClusterBuilder) step(input networkClusterInput) *netmodel.Result {
	b.t.Helper()
	var err error
	var result *netmodel.Result
	var data []byte
	var entry []byte
	var entryTag byte
	session := b.broker.sessions[input.Node]
	switch input.Action {
	case "advance":
		err = b.broker.network.advance(input.Nanos)
	case "change_link":
		err = b.broker.ChangeLink(input.LinkAction, input.From, input.To, input.Nanos, input.Symmetric)
	case "partition_domains":
		err = b.broker.network.model.Partition(input.From, input.To, input.Symmetric)
	case "register":
		session.Node.Incarnation = input.Incarnation
		err = b.broker.network.register(session)
	case "revoke":
		err = b.broker.network.revoke(session, input.Graceful)
	case "dispatch":
		r := b.broker.network.model.Dispatch(*input.Request)
		result = &r
	case "attach_epoch":
		config := testConfig()
		config.Profile, config.Clock = CooperativeProfile, ClockPolicy{EpochNanos: input.Epoch}
		config.NetworkV2 = b.environments[input.Node].config.NetworkV2
		var environment *Environment
		environment, err = NewEnvironment(config)
		if err == nil {
			err = b.broker.network.attach(session, environment)
			data = []byte{0}
			if environment.networkV2.model == b.broker.network.model {
				data[0] = 1
			}
		}
	case "serve":
		e := b.environments[input.Node]
		e.now = input.Nanos
		e.runtime.configured = !input.Inactive
		if input.TranscriptBytes != 0 {
			e.config.Limits.TranscriptBytes = input.TranscriptBytes
		}
		encoded, marshalErr := json.Marshal(bytesOutput{Data: input.Data})
		if marshalErr != nil {
			b.t.Fatal(marshalErr)
		}
		output, callErr := e.runtimeNetworkStream(encoded)
		err = callErr
		if err == nil {
			data = output.Data
		}
	case "operation_entry":
		entry, err = json.Marshal(input.Operation)
		entryTag = 1
	case "quiescence_entry":
		entry, err = json.Marshal(input.Quiescence)
		entryTag = 3
	default:
		b.t.Fatalf("unknown network cluster action %q", input.Action)
	}
	resourceExhausted := err == netmodel.ErrResourceExhausted
	expected := b.observe(err, result, data)
	if entry != nil {
		var value map[string]any
		if err := json.Unmarshal(expected, &value); err != nil {
			b.t.Fatal(err)
		}
		value["entry_json"], value["entry_tag"] = string(entry), entryTag
		encoded, err := json.Marshal(value)
		if err != nil {
			b.t.Fatal(err)
		}
		expected = encoded
	}
	b.corpus.Steps = append(b.corpus.Steps, networkClusterStep{Input: input, Expected: expected, ResourceExhausted: resourceExhausted})
	return result
}

func (b *networkClusterBuilder) request(request netmodel.Request) netmodel.Result {
	return *b.step(networkClusterInput{Action: "dispatch", Request: &request})
}

func (b *networkClusterBuilder) serve(node string, data []byte) []byte {
	b.t.Helper()
	b.step(networkClusterInput{Action: "serve", Node: node, Nanos: 1, Data: data})
	var output struct {
		Data  []byte `json:"data"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(b.corpus.Steps[len(b.corpus.Steps)-1].Expected, &output); err != nil {
		b.t.Fatal(err)
	}
	if output.Error != "" {
		b.t.Fatal(output.Error)
	}
	return output.Data
}

func networkClusterCorpora(t *testing.T) []networkClusterCorpus {
	var corpora []networkClusterCorpus
	add := func(name string, config BrokerConfig, run func(*networkClusterBuilder)) {
		b := newNetworkClusterCorpus(t, name, config)
		if b.broker != nil {
			run(b)
		}
		corpora = append(corpora, b.corpus)
	}
	config := func() BrokerConfig {
		c := brokerTestConfig()
		c.Network = streamConfig()
		c.Addresses = map[string]string{}
		return c
	}
	add("default-addresses-and-topology", config(), func(*networkClusterBuilder) {})
	add("own-server-dial-and-bytes", config(), func(b *networkClusterBuilder) {
		listen := b.serve("a", streamFrame(1, 1, 0, 0, 0, 0, nil))
		dial := b.serve("a", streamFrame(2, 2, 0, 0, binary.BigEndian.Uint32(listen[48:]), 0, nil))
		accept := b.serve("a", streamFrame(3, 3, binary.BigEndian.Uint64(listen[16:]), 0, 0, 0, nil))
		b.serve("a", streamFrame(5, 4, binary.BigEndian.Uint64(dial[16:]), 0, 0, 5, []byte("hello")))
		b.serve("a", streamFrame(4, 5, binary.BigEndian.Uint64(accept[16:]), 0, 0, 5, nil))
	})
	add("stream-stale-cancel", config(), func(b *networkClusterBuilder) {
		b.serve("a", streamFrame(13, 1, 0, 0, 0, 0, nil))
	})
	crossNode := config()
	crossNode.Addresses["b"] = "10.0.7.1"
	add("cross-node-ready-poll-and-stale-cancel-binary-fixture", crossNode, func(b *networkClusterBuilder) {
		listen := b.serve("b", streamFrame(1, 1, 0, 0, 0, 0, nil))
		dial := streamFrame(2, 1, 0, 0, binary.BigEndian.Uint32(listen[48:]), 0, nil)
		copy(dial[56:60], []byte{10, 0, 7, 1})
		client := b.serve("a", dial)
		server := b.serve("b", streamFrame(3, 2, binary.BigEndian.Uint64(listen[16:]), 0, 0, 0, nil))
		b.serve("b", streamFrame(4, 3, binary.BigEndian.Uint64(server[16:]), 0, 0, 4, nil))
		b.serve("a", streamFrame(5, 2, binary.BigEndian.Uint64(client[16:]), 0, 0, 4, []byte("ping")))
		poll := b.serve("b", streamFrame(12, 3, 0, 0, 0, 4, nil))
		if !bytes.Equal(poll[64:], []byte("ping")) {
			b.t.Fatalf("cross-node corpus poll lost payload: %x", poll)
		}
		b.serve("a", streamFrame(13, 9, 0, 0, 0, 0, nil))
	})
	add("stream-pending-read-cancel", config(), func(b *networkClusterBuilder) {
		listen := b.serve("a", streamFrame(1, 1, 0, 0, 0, 0, nil))
		b.serve("a", streamFrame(2, 2, 0, 0, binary.BigEndian.Uint32(listen[48:]), 0, nil))
		accept := b.serve("a", streamFrame(3, 3, binary.BigEndian.Uint64(listen[16:]), 0, 0, 0, nil))
		b.serve("a", streamFrame(4, 4, binary.BigEndian.Uint64(accept[16:]), 0, 0, 4, nil))
		b.serve("a", streamFrame(13, 4, 0, 0, 0, 0, nil))
	})
	add("go-log-formfeed-escaping", config(), func(b *networkClusterBuilder) {
		b.step(networkClusterInput{Action: "operation_entry", Operation: &networkV2Operation{At: 3, Request: netmodel.Request{Domain: netmodel.Domain{Run: 1, Node: "a.client", Incarnation: 1, Address: "10.1.0.2"}, Op: "read", RequestID: 1}, Result: netmodel.Result{RequestID: 1, State: "completed", Error: "\f"}}})
	})
	strict := config()
	strict.StrictTopology = true
	strict.Links = []BrokerLink{{From: "a", To: "b", DelayNanos: 7}}
	add("strict-directional-disabled-link", strict, func(*networkClusterBuilder) {})
	for _, mutation := range []string{"local-fault", "ipv4-range", "ipv4-leading-zero", "self-link", "unknown-from", "unknown-to", "duplicate-link", "unordered-faults", "duplicate-fault", "unknown-action", "unknown-fault-endpoint", "profile", "zero-listeners"} {
		c := config()
		fault := BrokerNetworkFault{Name: "x", Action: "partition", From: "a", To: "b", AtNanos: 2}
		switch mutation {
		case "local-fault":
			c.Network.Faults = []NetworkFaultV2{{Name: "local", Action: "partition", AtNanos: 1}}
		case "ipv4-range":
			c.Addresses["a"] = "10.0.0.256"
		case "ipv4-leading-zero":
			c.Addresses["a"] = "10.00.0.1"
		case "self-link":
			c.Links = []BrokerLink{{From: "a", To: "a"}}
		case "unknown-from":
			c.Links = []BrokerLink{{From: "z", To: "b"}}
		case "unknown-to":
			c.Links = []BrokerLink{{From: "a", To: "z"}}
		case "duplicate-link":
			c.Links = []BrokerLink{{From: "a", To: "b"}, {From: "a", To: "b"}}
		case "unordered-faults":
			c.Faults = []BrokerNetworkFault{fault, fault}
			c.Faults[0].AtNanos = 9
			c.Faults[1].Name = "early"
		case "duplicate-fault":
			c.Faults = []BrokerNetworkFault{fault, fault}
		case "unknown-action":
			fault.Action = "reset"
			c.Faults = []BrokerNetworkFault{fault}
		case "unknown-fault-endpoint":
			fault.To = "z"
			c.Faults = []BrokerNetworkFault{fault}
		case "profile":
			c.Network.Profile = "other"
		case "zero-listeners":
			c.Network.Limits.Listeners = 0
		}
		add("reject-"+mutation, c, func(*networkClusterBuilder) {})
	}
	for _, symmetric := range []bool{false, true} {
		c := config()
		c.Faults = []BrokerNetworkFault{{Name: "cut", Action: "partition", From: "a", To: "b", AtNanos: 5, Symmetric: true}, {Name: "mend", Action: "heal", From: "a", To: "b", AtNanos: 9, Symmetric: true}}
		add(fmt.Sprintf("scheduled-and-live-four-pairs-symmetric-%t", symmetric), c, func(b *networkClusterBuilder) {
			b.step(networkClusterInput{Action: "advance", Nanos: 6})
			b.step(networkClusterInput{Action: "advance", Nanos: 9})
			for _, action := range []string{"partition", "delay", "heal", "reset"} {
				b.step(networkClusterInput{Action: "change_link", LinkAction: action, From: "a", To: "b", Nanos: 10, Symmetric: symmetric})
			}
			b.step(networkClusterInput{Action: "change_link", LinkAction: "heal", From: "a", To: "a", Symmetric: true})
		})
	}
	for _, graceful := range []bool{false, true} {
		add(fmt.Sprintf("revoke-both-domains-graceful-%t", graceful), config(), func(b *networkClusterBuilder) {
			b.step(networkClusterInput{Action: "serve", Node: "b", Nanos: 1, Data: streamFrame(1, 1, 0, 0, 0, 0, nil)})
			b.step(networkClusterInput{Action: "serve", Node: "b", Nanos: 1, Data: streamFrame(2, 2, 0, 0, 20000, 0, nil)})
			b.step(networkClusterInput{Action: "revoke", Node: "b", Graceful: graceful})
			b.step(networkClusterInput{Action: "revoke", Node: "b", Graceful: graceful})
			b.step(networkClusterInput{Action: "register", Node: "b", Incarnation: 2})
		})
	}
	for _, mutation := range []string{"reserved", "query", "bad-query", "zero-id", "large-id", "inactive", "opcode", "short-write", "trailing-listen", "transcript-capacity"} {
		add("stream-"+mutation, config(), func(b *networkClusterBuilder) {
			input := networkClusterInput{Action: "serve", Node: "a", Nanos: 1, Data: streamFrame(1, 1, 0, 0, 0, 0, nil)}
			switch mutation {
			case "reserved":
				input.Data[41] = 1
			case "query":
				input.Data = streamFrame(0, 0, 0, 0, 0, 0, nil)
			case "bad-query":
				input.Data = streamFrame(0, 1, 0, 0, 0, 0, nil)
			case "zero-id":
				input.Data = streamFrame(1, 0, 0, 0, 0, 0, nil)
			case "large-id":
				input.Data = streamFrame(1, 1<<48, 0, 0, 0, 0, nil)
			case "inactive":
				input.Inactive = true
			case "opcode":
				input.Data = streamFrame(14, 1, 0, 0, 0, 0, nil)
			case "short-write":
				input.Data = streamFrame(5, 1, 0, 0, 0, 3, []byte("abc"))
				input.Data = input.Data[:len(input.Data)-1]
			case "trailing-listen":
				input.Data = append(input.Data, 0)
			case "transcript-capacity":
				input.TranscriptBytes = 16
			}
			b.step(input)
		})
	}
	networkClusterBrokerCorpora(add, config)
	add("go-log-entry-json-and-wire-tags", config(), func(b *networkClusterBuilder) {
		b.step(networkClusterInput{Action: "operation_entry", Operation: &networkV2Operation{At: 3, Request: netmodel.Request{Domain: netmodel.Domain{Run: 1, Node: "a.client", Incarnation: 1, Address: "10.1.0.2"}, Op: "write", Handle: 7, Host: "10.0.0.2", Bytes: []byte("hi"), Count: 2, RequestID: 1<<48 | 4}, Result: netmodel.Result{RequestID: 4, State: "completed", Count: 2, Error: "<&>"}}})
		b.step(networkClusterInput{Action: "quiescence_entry", Quiescence: &world.Quiescence{Kind: world.QuiescenceDelivered, Before: 5, After: 6, Deliveries: []world.Delivery{{RequestID: 1, EventID: 2, At: 6, Kind: "network"}}}})
	})
	return corpora
}

func networkClusterBrokerCorpora(add func(string, BrokerConfig, func(*networkClusterBuilder)), config func() BrokerConfig) {
	pair := func(b *networkClusterBuilder) (netmodel.Domain, netmodel.Domain, uint64, uint64) {
		client, _ := b.broker.network.domains(b.broker.sessions["a"])
		_, server := b.broker.network.domains(b.broker.sessions["b"])
		listener := b.request(netmodel.Request{Domain: server, Op: "listen", RequestID: 1})
		x := b.request(netmodel.Request{Domain: client, Op: "dial", Host: server.Address, Port: listener.Local.Port, RequestID: 2})
		y := b.request(netmodel.Request{Domain: server, Op: "accept", Handle: listener.Handle, RequestID: 3})
		return client, server, x.Handle, y.Handle
	}
	ceilingConfig := config()
	ceilingConfig.Network.Limits.StallNanos = math.MaxInt64 - 1
	add("broker-clock-ceiling-pending-dial-atomic-rejection", ceilingConfig, func(b *networkClusterBuilder) {
		client, _ := b.broker.network.domains(b.broker.sessions["a"])
		_, server := b.broker.network.domains(b.broker.sessions["b"])
		listener := b.request(netmodel.Request{Domain: server, Op: "listen", RequestID: 1})
		b.step(networkClusterInput{Action: "partition_domains", From: client.Node, To: server.Node})
		b.request(netmodel.Request{Domain: client, Op: "dial", Host: server.Address, Port: listener.Local.Port, RequestID: 2})
	})
	for _, pending := range []bool{false, true} {
		c := config()
		c.Network.Limits.StallNanos = 100
		add(fmt.Sprintf("guest-attachment-clock-ceiling-pending-%t", pending), c, func(b *networkClusterBuilder) {
			client, server := b.broker.network.domains(b.broker.sessions["a"])
			listener := b.request(netmodel.Request{Domain: server, Op: "listen", RequestID: 1})
			b.step(networkClusterInput{Action: "partition_domains", From: client.Node, To: server.Node})
			if pending {
				b.request(netmodel.Request{Domain: client, Op: "dial", Host: server.Address, Port: listener.Local.Port, RequestID: 2})
			}
			b.step(networkClusterInput{Action: "attach_epoch", Node: "a", Epoch: math.MaxInt64 - 100})
			if !pending {
				b.request(netmodel.Request{Domain: client, Op: "dial", Host: server.Address, Port: listener.Local.Port, RequestID: 3})
			}
		})
	}
	for _, scheduled := range []bool{false, true} {
		for _, action := range []string{"partition", "heal", "delay"} {
			c := config()
			c.Network.Limits.Transitions = 1
			if scheduled {
				c.Faults = []BrokerNetworkFault{{Name: "change", Action: action, From: "a", To: "b", AtNanos: 100, DelayNanos: 10, Symmetric: true}}
			}
			add(fmt.Sprintf("group-rejection-atomic-%s-scheduled-%t", action, scheduled), c, func(b *networkClusterBuilder) {
				if scheduled {
					b.step(networkClusterInput{Action: "advance", Nanos: 100})
				} else {
					b.step(networkClusterInput{Action: "change_link", LinkAction: action, From: "a", To: "b", Nanos: 10, Symmetric: true})
				}
			})
		}
		c := config()
		c.Network.Limits.StallNanos = 10
		if scheduled {
			c.Faults = []BrokerNetworkFault{{Name: "heal", Action: "heal", From: "a", To: "b", AtNanos: 100, Symmetric: true}}
		}
		add(fmt.Sprintf("leap-preserves-dial-deadline-scheduled-heal-%t", scheduled), c, func(b *networkClusterBuilder) {
			client, _ := b.broker.network.domains(b.broker.sessions["a"])
			_, server := b.broker.network.domains(b.broker.sessions["b"])
			listener := b.request(netmodel.Request{Domain: server, Op: "listen", RequestID: 1})
			b.step(networkClusterInput{Action: "change_link", LinkAction: "partition", From: "a", To: "b", Symmetric: true})
			b.request(netmodel.Request{Domain: client, Op: "dial", Host: server.Address, Port: listener.Local.Port, Deadline: 5, RequestID: 2})
			b.step(networkClusterInput{Action: "advance", Nanos: 100})
			b.request(netmodel.Request{Domain: client, Op: "poll", RequestID: 2})
		})
	}
	for _, transitions := range []uint64{12, 13} {
		c := config()
		c.Network.Limits.Transitions, c.Network.Limits.StallNanos = transitions, 10
		add(fmt.Sprintf("group-reservation-ceiling-%d", transitions), c, func(b *networkClusterBuilder) {
			client, _, x, _ := pair(b)
			b.step(networkClusterInput{Action: "change_link", LinkAction: "delay", From: "a", To: "b", Nanos: 200, Symmetric: true})
			b.request(netmodel.Request{Domain: client, Op: "write", Handle: x, RequestID: 4, Bytes: []byte("held")})
			b.step(networkClusterInput{Action: "change_link", LinkAction: "partition", From: "a", To: "b", Symmetric: true})
			if transitions == 12 {
				b.step(networkClusterInput{Action: "change_link", LinkAction: "delay", From: "a", To: "b", Nanos: 200, Symmetric: true})
				return
			}
			b.step(networkClusterInput{Action: "change_link", LinkAction: "delay", From: "a", To: "b", Nanos: 1, Symmetric: true})
			b.step(networkClusterInput{Action: "advance", Nanos: 12})
		})
	}
	for _, healAt := range []uint64{0, 111, 112} {
		c := config()
		c.Network.Limits.StallNanos = 10
		c.Faults = []BrokerNetworkFault{{Name: "partition", Action: "partition", From: "a", To: "b", AtNanos: 100, Symmetric: true}}
		if healAt != 0 {
			c.Faults = append(c.Faults, BrokerNetworkFault{Name: "heal", Action: "heal", From: "a", To: "b", AtNanos: healAt, Symmetric: true})
		}
		add(fmt.Sprintf("scheduled-partition-fault-time-heal-%d", healAt), c, func(b *networkClusterBuilder) {
			client, server, x, y := pair(b)
			b.step(networkClusterInput{Action: "change_link", LinkAction: "delay", From: "a", To: "b", Nanos: 200, Symmetric: true})
			b.request(netmodel.Request{Domain: client, Op: "write", Handle: x, RequestID: 4, Bytes: []byte("x")})
			b.step(networkClusterInput{Action: "advance", Nanos: max(110, healAt)})
			b.request(netmodel.Request{Domain: server, Op: "read", Handle: y, RequestID: 5, Count: 1})
			if healAt != 111 {
				return
			}
			b.step(networkClusterInput{Action: "advance", Nanos: 310})
			b.request(netmodel.Request{Domain: server, Op: "poll", RequestID: 5})
			b.step(networkClusterInput{Action: "advance", Nanos: 311})
			result := b.request(netmodel.Request{Domain: server, Op: "poll", RequestID: 5})
			if result.Error != "" || result.State != "completed" || string(result.Bytes) != "x" {
				b.t.Fatalf("scheduled heal must deliver the delayed payload: %+v", result)
			}
		})
	}
}

func TestNetworkClusterCorpusIsCurrent(t *testing.T) {
	regenerate := os.Getenv("GOMAD_NETWORK_CLUSTER_CORPUS_REGENERATE") == "1"
	pinPath := "../../../rust/wasmhost/src/cluster/network/PINNED_GO_WASI_SHA256"
	pin := []byte(ImplementationSHA256() + "\n")
	if regenerate {
		if err := os.WriteFile(pinPath, pin, 0o644); err != nil {
			t.Fatal(err)
		}
	} else {
		got, err := os.ReadFile(pinPath)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, pin) {
			t.Fatal("stale cluster Go WASI implementation pin")
		}
	}
	if regenerate {
		if err := os.MkdirAll(networkClusterCorpusDirectory, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	wantNames := map[string]bool{}
	for _, corpus := range networkClusterCorpora(t) {
		name := corpus.Name + ".json"
		if wantNames[name] {
			t.Fatalf("duplicate network cluster corpus %s", name)
		}
		wantNames[name] = true
		data, err := json.Marshal(corpus)
		if err != nil {
			t.Fatal(err)
		}
		data = append(data, '\n')
		path := filepath.Join(networkClusterCorpusDirectory, name)
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
			t.Errorf("stale network cluster corpus %s", corpus.Name)
		}
	}
	entries, err := os.ReadDir(networkClusterCorpusDirectory)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".json" && !wantNames[entry.Name()] {
			t.Errorf("unexpected network cluster corpus %s", entry.Name())
		}
	}
}

func TestNetworkClusterCorpus(t *testing.T) {
	files, err := filepath.Glob(filepath.Join(networkClusterCorpusDirectory, "*.json"))
	if err != nil || len(files) == 0 {
		t.Fatalf("network cluster corpus selection: %v, files=%d", err, len(files))
	}
	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			data, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			var corpus networkClusterCorpus
			if err := json.Unmarshal(data, &corpus); err != nil {
				t.Fatal(err)
			}
			if corpus.Schema != "gomad.network-cluster-corpus/v1" || corpus.Implementation != ImplementationSHA256() || corpus.NetworkImplementation != netmodel.ImplementationSHA256() {
				t.Fatal("network cluster corpus identity")
			}
			var envelope struct {
				Config struct {
					Model     *NetworkModelV2   `json:"model"`
					Addresses map[string]string `json:"addresses"`
					Links     []struct {
						From    string `json:"from"`
						To      string `json:"to"`
						Enabled bool   `json:"enabled"`
						Delay   uint64 `json:"delay_nanos"`
					} `json:"links"`
					Faults []struct {
						Name      string `json:"name"`
						Action    string `json:"action"`
						From      string `json:"from"`
						To        string `json:"to"`
						At        uint64 `json:"at_nanos"`
						Delay     uint64 `json:"delay_nanos"`
						Symmetric bool   `json:"symmetric"`
					} `json:"faults"`
					Strict bool `json:"strict_topology"`
				} `json:"config"`
			}
			if err := json.Unmarshal(data, &envelope); err != nil {
				t.Fatal(err)
			}
			config := brokerTestConfig()
			config.Nodes = nil
			for _, id := range corpus.Nodes {
				config.Nodes = append(config.Nodes, BrokerNode{ID: id, Incarnation: 1})
			}
			config.Network, config.Addresses = envelope.Config.Model, envelope.Config.Addresses
			config.Links, config.Faults, config.StrictTopology = nil, nil, envelope.Config.Strict
			for _, link := range envelope.Config.Links {
				config.Links = append(config.Links, BrokerLink{From: link.From, To: link.To, Enabled: link.Enabled, DelayNanos: link.Delay})
			}
			for _, fault := range envelope.Config.Faults {
				config.Faults = append(config.Faults, BrokerNetworkFault{Name: fault.Name, Action: fault.Action, From: fault.From, To: fault.To, AtNanos: fault.At, DelayNanos: fault.Delay, Symmetric: fault.Symmetric})
			}
			b := newNetworkClusterCorpus(t, corpus.Name, config)
			if b.corpus.CreationError != corpus.CreationError {
				t.Fatalf("creation: got %q want %q", b.corpus.CreationError, corpus.CreationError)
			}
			if b.broker == nil {
				if len(corpus.Steps) != 0 || len(corpus.Initial) != 0 {
					t.Fatal("rejected creation has execution observations")
				}
				return
			}
			compareNetworkCorpusJSON(t, b.corpus.Initial, corpus.Initial)
			for _, step := range corpus.Steps {
				b.step(step.Input)
				if got := b.corpus.Steps[len(b.corpus.Steps)-1].ResourceExhausted; got != step.ResourceExhausted {
					t.Fatalf("resource exhausted got %t want %t", got, step.ResourceExhausted)
				}
				compareNetworkCorpusJSON(t, b.corpus.Steps[len(b.corpus.Steps)-1].Expected, step.Expected)
			}
			if corpus.Name == "cross-node-ready-poll-and-stale-cancel-binary-fixture" {
				for id, environment := range b.environments {
					encoded, err := os.ReadFile(filepath.Join("../../../rust/wasmhost/tests/testdata/cluster_network", id+".gmnetev1"))
					if err != nil {
						t.Fatal(err)
					}
					decoded, err := DecodeClusterNetworkEvidence(encoded, ClusterNetworkLimits{Network: config.Network.Limits, Instance: *config.Network, TranscriptBytes: 1 << 20})
					if err != nil {
						t.Fatal(err)
					}
					document, err := environment.NetworkV2Evidence()
					if err != nil {
						t.Fatal(err)
					}
					if !bytes.Equal(decoded.Document, document) {
						t.Fatalf("%s corpus network binary document differs from Go", id)
					}
					record, err := b.broker.NetworkRecord()
					if err != nil {
						t.Fatal(err)
					}
					if !reflect.DeepEqual(decoded.Record, record) {
						t.Fatalf("%s corpus native record differs", id)
					}
				}
			}
		})
	}
}

func compareNetworkCorpusJSON(t *testing.T, got, want []byte) {
	t.Helper()
	decode := func(data []byte) any {
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.UseNumber()
		var value any
		if err := decoder.Decode(&value); err != nil {
			t.Fatal(err)
		}
		return value
	}
	if !reflect.DeepEqual(decode(got), decode(want)) {
		t.Fatalf("network corpus observation differs:\ngot %s\nwant %s", got, want)
	}
}
