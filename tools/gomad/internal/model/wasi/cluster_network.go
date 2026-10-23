package wasi

import (
	"bytes"
	"encoding/json"
	"slices"
	"unicode/utf8"

	netmodel "go.temporal.io/server/tools/gomad/internal/model/network"
	"go.temporal.io/server/tools/gomad/internal/model/world"
)

// Cluster networking (fn-178.13): Go stays the configuration authority for the
// in-process cluster's shared Rust network owner, and rebuilds each instance's
// broker-mode evidence document from the helper's binary stream.

// ClusterNetworkConfig is the cluster execute frame's `network` object.
type ClusterNetworkConfig struct {
	Model          NetworkModelV2        `json:"model"`
	Addresses      map[string]string     `json:"addresses,omitempty"`
	Links          []ClusterNetworkLink  `json:"links,omitempty"`
	StrictTopology bool                  `json:"strict_topology,omitempty"`
	Faults         []ClusterNetworkFault `json:"faults,omitempty"`
}

type ClusterNetworkLink struct {
	From       string `json:"from"`
	To         string `json:"to"`
	Enabled    bool   `json:"enabled"`
	DelayNanos uint64 `json:"delay_nanos"`
}

type ClusterNetworkFault struct {
	Name       string `json:"name"`
	Action     string `json:"action"`
	From       string `json:"from"`
	To         string `json:"to"`
	AtNanos    uint64 `json:"at_nanos"`
	DelayNanos uint64 `json:"delay_nanos"`
	Symmetric  bool   `json:"symmetric"`
}

// ClusterNetworkFrame maps a broker configuration and each node's stream
// profile into the cluster frame, after refusing everything the Go broker
// refuses: `newBrokerNetwork` runs on the same configuration, and per node the
// checks `brokerNetwork.attach` applies at a node's first call. A cluster also
// refuses a node profile without a cluster network and a node without the
// cooperative runtime. A nil network config with no node profiles maps to nil.
func ClusterNetworkFrame(config BrokerConfig, nodes map[string]*NetworkModelV2, cooperative map[string]bool) (*ClusterNetworkConfig, error) {
	if _, err := NewBroker(config); err != nil {
		return nil, err
	}
	for _, node := range config.Nodes {
		// The helper's shared owner registers every instance at incarnation 1.
		if node.Incarnation != 1 {
			return nil, invalid("cluster node incarnation")
		}
	}
	if config.Network == nil {
		for _, model := range nodes {
			if model != nil {
				return nil, invalid("broker node network without cluster network")
			}
		}
		return nil, nil
	}
	for _, node := range config.Nodes {
		model := nodes[node.ID]
		if model == nil {
			return nil, invalid("broker node requires stream network profile")
		}
		if err := validateNetworkModelV2(model); err != nil {
			return nil, err
		}
		if len(model.Faults) != 0 {
			return nil, invalid("broker node local fault schedule")
		}
		if !cooperative[node.ID] {
			return nil, invalid("broker node requires cooperative runtime")
		}
	}
	// newBrokerNetwork has refused a node-local schedule in config.Network.
	frame := &ClusterNetworkConfig{Model: *config.Network, StrictTopology: config.StrictTopology}
	// The WASI package's pure-import rule excludes `maps`.
	if len(config.Addresses) != 0 {
		frame.Addresses = make(map[string]string, len(config.Addresses))
		for id, address := range config.Addresses {
			frame.Addresses[id] = address
		}
	}
	for _, link := range config.Links {
		frame.Links = append(frame.Links, ClusterNetworkLink(link))
	}
	for _, fault := range config.Faults {
		frame.Faults = append(frame.Faults, ClusterNetworkFault(fault))
	}
	return frame, nil
}

// ClusterNetworkLimits bind a decode to the configuration the run used.
type ClusterNetworkLimits struct {
	// Network is the cluster model's limits, which bound the shared Record.
	Network netmodel.Limits
	// Instance is the instance's own stream profile, which sizes its network World.
	Instance NetworkModelV2
	// TranscriptBytes bounds the stream, the operations log and the document.
	TranscriptBytes uint64
}

// ClusterNetworkEvidence is one cluster instance's network evidence rebuilt
// from the helper's `GMNETEV1` stream.
type ClusterNetworkEvidence struct {
	// Document is byte-identical to Environment.NetworkV2Evidence in broker mode.
	Document []byte
	// Record is the shared owner's Record at the instance's terminal, with
	// identities recomputed by the generated network codec.
	Record netmodel.Record
	// World is the instance's network World, replayed by the generated decoder.
	World world.Snapshot
}

const clusterNetworkMagic = "GMNETEV1"

// DecodeClusterNetworkEvidence strictly decodes one instance's binary network
// evidence (layout: wasmhost `src/network/cluster/evidence.rs`): truncation,
// trailing bytes, unknown tags, non-minimal varints, invalid UTF-8 and
// oversize streams are `invalid`. It rebuilds the operations log with Go's
// codecs and requires the helper's Go-JSON byte count, recomputes the Record's
// identities and replays the World. A rebuilt document over TranscriptBytes is
// the instance's `network-v2-evidence-bytes` capacity, as in broker mode.
func DecodeClusterNetworkEvidence(data []byte, limits ClusterNetworkLimits) (ClusterNetworkEvidence, error) {
	if uint64(len(data)) > limits.TranscriptBytes {
		return ClusterNetworkEvidence{}, invalid("cluster network evidence size")
	}
	d := &clusterNetworkDecoder{data: data}
	if !bytes.HasPrefix(data, []byte(clusterNetworkMagic)) {
		return ClusterNetworkEvidence{}, invalid("cluster network evidence magic")
	}
	d.offset = len(clusterNetworkMagic)
	if d.varint() != 1 || d.text() != NetworkStreamProfile {
		d.fail("header")
		return ClusterNetworkEvidence{}, d.err
	}
	declaredLog := d.varint()
	entries := d.count()
	var log []byte
	for range entries {
		var line []byte
		var err error
		switch d.byte() {
		case 1:
			line, err = json.Marshal(d.operation())
		case 3:
			line, err = json.Marshal(d.quiescence())
		default:
			d.fail("operation tag")
			return ClusterNetworkEvidence{}, d.err
		}
		if d.err != nil {
			return ClusterNetworkEvidence{}, d.err
		}
		if err != nil {
			return ClusterNetworkEvidence{}, err
		}
		log = append(append(log, line...), '\n')
		if uint64(len(log)) > limits.TranscriptBytes {
			return ClusterNetworkEvidence{}, invalid("cluster network evidence log")
		}
	}
	if d.err == nil && uint64(len(log)) != declaredLog {
		return ClusterNetworkEvidence{}, invalid("cluster network evidence log accounting")
	}
	record := d.record()
	worldWire := []byte(nil)
	if d.flag() {
		worldWire = d.bytes()
	} else {
		d.fail("world")
	}
	if d.err != nil {
		return ClusterNetworkEvidence{}, d.err
	}
	if d.offset != len(d.data) {
		return ClusterNetworkEvidence{}, invalid("cluster network evidence trailing bytes")
	}
	if err := validateClusterNetworkRecord(record, limits.Network); err != nil {
		return ClusterNetworkEvidence{}, err
	}
	snapshot, err := world.DecodeExpandedSnapshot(worldWire, networkV2WorldConfig(limits.Instance.Limits))
	if err != nil {
		return ClusterNetworkEvidence{}, invalid("cluster network evidence world")
	}
	document, err := networkV2Document(log, record, snapshot, limits.TranscriptBytes)
	if err != nil {
		return ClusterNetworkEvidence{}, err
	}
	return ClusterNetworkEvidence{Document: document, Record: record, World: snapshot}, nil
}

// validateClusterNetworkRecord recomputes the Record's transition digest and
// snapshot identity through the generated network codec, and bounds it by the
// cluster model's limits.
func validateClusterNetworkRecord(record netmodel.Record, limits netmodel.Limits) error {
	if uint64(len(record.Transitions)) > limits.Transitions || uint64(len(record.Snapshot.Listeners)) > limits.Listeners || uint64(len(record.Snapshot.Connections)) > limits.Connections {
		return invalid("cluster network evidence record")
	}
	if err := netmodel.VerifyRecordIdentity(record); err != nil {
		return invalid("cluster network evidence record")
	}
	return nil
}

type clusterNetworkDecoder struct {
	data   []byte
	offset int
	err    error
}

// fail records the first decode failure; later reads become no-ops.
func (d *clusterNetworkDecoder) fail(field string) {
	if d.err == nil {
		d.err = invalid("cluster network evidence " + field)
	}
}

func (d *clusterNetworkDecoder) byte() byte {
	if d.err != nil || d.offset >= len(d.data) {
		d.fail("truncated")
		return 0
	}
	value := d.data[d.offset]
	d.offset++
	return value
}

// varint reads a minimal unsigned LEB128 value of at most ten bytes.
func (d *clusterNetworkDecoder) varint() uint64 {
	var value uint64
	for shift := uint(0); shift < 70; shift += 7 {
		b := d.byte()
		if d.err != nil {
			return 0
		}
		if shift == 63 && b > 1 {
			d.fail("varint overflow")
			return 0
		}
		value |= uint64(b&0x7f) << shift
		if b&0x80 == 0 {
			if b == 0 && shift != 0 {
				d.fail("non-minimal varint")
				return 0
			}
			return value
		}
	}
	d.fail("varint overflow")
	return 0
}

// count reads a list length; every element takes at least one byte.
func (d *clusterNetworkDecoder) count() uint64 {
	n := d.varint()
	if d.err == nil && n > uint64(len(d.data)-d.offset) {
		d.fail("count")
		return 0
	}
	return n
}

func (d *clusterNetworkDecoder) bytes() []byte {
	n := d.varint()
	if d.err != nil || n > uint64(len(d.data)-d.offset) {
		d.fail("truncated")
		return nil
	}
	value := slices.Clone(d.data[d.offset : d.offset+int(n)])
	d.offset += int(n)
	if len(value) == 0 {
		return nil
	}
	return value
}

func (d *clusterNetworkDecoder) text() string {
	value := d.bytes()
	if !utf8.Valid(value) {
		d.fail("string")
		return ""
	}
	return string(value)
}

func (d *clusterNetworkDecoder) flag() bool {
	switch d.byte() {
	case 0:
		return false
	case 1:
		return true
	default:
		d.fail("flag")
		return false
	}
}

func (d *clusterNetworkDecoder) int() int {
	value := d.varint()
	if value > uint64(^uint(0)>>1) {
		d.fail("integer")
		return 0
	}
	return int(value)
}

func (d *clusterNetworkDecoder) uint32() uint32 {
	value := d.varint()
	if value > uint64(^uint32(0)) {
		d.fail("integer")
		return 0
	}
	return uint32(value)
}

func (d *clusterNetworkDecoder) operation() networkV2Operation {
	var operation networkV2Operation
	operation.At = d.varint()
	r := &operation.Request
	r.Domain = netmodel.Domain{Run: d.varint(), Node: d.text(), Incarnation: d.varint(), Address: d.text()}
	r.Op, r.Handle, r.Host, r.Port = d.text(), d.varint(), d.text(), d.int()
	r.Bytes, r.Count, r.Deadline, r.RequestID = d.bytes(), d.uint32(), d.varint(), d.varint()
	result := &operation.Result
	result.RequestID, result.State, result.Handle = d.varint(), d.text(), d.varint()
	result.Local = netmodel.Address{IP: d.text(), Port: d.int()}
	result.Remote = netmodel.Address{IP: d.text(), Port: d.int()}
	result.Bytes, result.Count, result.Deadline, result.Error = d.bytes(), d.int(), d.varint(), d.text()
	return operation
}

func (d *clusterNetworkDecoder) quiescence() world.Quiescence {
	var q world.Quiescence
	q.Kind = world.QuiescenceKind(d.text())
	q.Before, q.After = world.LogicalTime(d.varint()), world.LogicalTime(d.varint())
	for range d.count() {
		q.Deliveries = append(q.Deliveries, world.Delivery{RequestID: world.RequestID(d.varint()), EventID: world.EventID(d.varint()), At: world.LogicalTime(d.varint()), Kind: d.text(), Payload: d.bytes()})
	}
	for range d.count() {
		q.Blocked = append(q.Blocked, world.RequestID(d.varint()))
	}
	switch q.Kind {
	case world.QuiescenceDelivered, world.QuiescenceDeadlock, world.QuiescenceIdle:
	default:
		d.fail("quiescence kind")
	}
	return q
}

// grow returns n zero elements of the slice's element type, non-nil even when
// n is zero, so presence survives JSON (null versus []).
func grow[S ~[]E, E any](s S, n uint64) S {
	return slices.Grow(s[:0:0], int(n)+1)[:n]
}

func (d *clusterNetworkDecoder) endpoint(node *string, incarnation *uint64, address *string, port *uint64) {
	*node, *incarnation, *address, *port = d.text(), d.varint(), d.text(), d.varint()
}

func (d *clusterNetworkDecoder) record() netmodel.Record {
	var record netmodel.Record
	if d.flag() {
		record.Transitions = grow(record.Transitions, d.count())
		for i := range record.Transitions {
			t := &record.Transitions[i]
			t.Ordinal, t.Kind = d.varint(), d.text()
			d.endpoint(&t.Source.Node, &t.Source.Incarnation, &t.Source.Address, &t.Source.Port)
			d.endpoint(&t.Destination.Node, &t.Destination.Incarnation, &t.Destination.Address, &t.Destination.Port)
			t.Connection, t.Bytes, t.Count, t.DelayNanos, t.Outcome = d.varint(), d.varint(), d.varint(), d.varint(), d.text()
		}
	}
	snapshot := &record.Snapshot
	if d.flag() {
		snapshot.Nodes = grow(snapshot.Nodes, d.count())
		for i := range snapshot.Nodes {
			n := &snapshot.Nodes[i]
			n.Node, n.Address, n.LastIncarnation, n.NextListenerPort, n.NextClientPort = d.text(), d.text(), d.varint(), d.varint(), d.varint()
		}
	}
	if d.flag() {
		snapshot.Links = grow(snapshot.Links, d.count())
		for i := range snapshot.Links {
			l := &snapshot.Links[i]
			l.From, l.To, l.Enabled, l.DelayNanos = d.text(), d.text(), d.flag(), d.varint()
		}
	}
	if d.flag() {
		snapshot.Listeners = grow(snapshot.Listeners, d.count())
		for i := range snapshot.Listeners {
			l := &snapshot.Listeners[i]
			d.endpoint(&l.Endpoint.Node, &l.Endpoint.Incarnation, &l.Endpoint.Address, &l.Endpoint.Port)
			l.Closed = d.flag()
		}
	}
	if d.flag() {
		snapshot.Connections = grow(snapshot.Connections, d.count())
		for i := range snapshot.Connections {
			c := &snapshot.Connections[i]
			c.ClientBytes, c.ServerBytes, c.Identity = d.varint(), d.varint(), d.varint()
			d.endpoint(&c.Client.Node, &c.Client.Incarnation, &c.Client.Address, &c.Client.Port)
			d.endpoint(&c.Server.Node, &c.Server.Incarnation, &c.Server.Address, &c.Server.Port)
			c.Closed, c.Reset = d.flag(), d.flag()
		}
	}
	snapshot.NextConnection, snapshot.TransitionSHA256, snapshot.Identity = d.varint(), d.text(), d.text()
	return record
}
