package wasi

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"sort"
	"strings"
	"syscall"
	"unicode/utf8"

	netmodel "go.temporal.io/server/tools/gomad/internal/model/network"
	"go.temporal.io/server/tools/gomad/internal/model/world"
)

const NetworkStreamProfile = netmodel.Profile
const networkStreamHeader = 64

// NetworkModelV2 declares the native hold/timeout stream profile.
type NetworkModelV2 struct {
	Profile string           `json:"profile"`
	Limits  netmodel.Limits  `json:"limits"`
	Faults  []NetworkFaultV2 `json:"faults,omitempty"`
}
type NetworkFaultV2 struct {
	Name       string `json:"name"`
	Action     string `json:"action"`
	AtNanos    uint64 `json:"at_nanos"`
	Port       uint32 `json:"port,omitempty"`
	DelayNanos uint64 `json:"delay_nanos,omitempty"`
}
type networkV2State struct {
	config         NetworkModelV2
	model          *netmodel.Model
	world          *world.Model
	client, server netmodel.Domain
	handles        map[uint64]netmodel.Domain
	ports          map[uint64]uint32
	requests       map[uint64]netmodel.Domain
	worldRequests  map[uint64]world.RequestID
	worldReady     map[uint64]bool
	requestBase    uint64
	fault          int
	log            []byte
}

func validateNetworkModelV2(config *NetworkModelV2) error {
	if config == nil {
		return nil
	}
	if config.Profile != NetworkStreamProfile {
		return invalid("network v2 profile")
	}
	limits := config.Limits
	if limits.Listeners == 0 || limits.Listeners > 4096 || limits.Connections == 0 || limits.Connections > 65536 || limits.Bytes == 0 || limits.Bytes > 128<<20 || limits.Transitions == 0 || limits.Transitions > 1<<20 {
		return invalid("network v2 limits")
	}
	if limits.ConnectionBytes == 0 || limits.ConnectionBytes > 1<<30 || limits.StallNanos == 0 || limits.StallNanos > math.MaxInt64-1 {
		return invalid("network v2 byte or stall limits")
	}
	names := map[string]bool{}
	if len(config.Faults) > int(limits.Transitions) {
		return invalid("network v2 fault capacity")
	}
	for i, f := range config.Faults {
		if f.Name == "" || len(f.Name) > 1024 || !utf8.ValidString(f.Name) || strings.IndexByte(f.Name, 0) >= 0 || names[f.Name] || f.AtNanos > math.MaxInt64 || f.DelayNanos > math.MaxInt64 || f.Port > 65535 || i > 0 && config.Faults[i-1].AtNanos > f.AtNanos {
			return invalid("network v2 fault schedule")
		}
		names[f.Name] = true
		switch f.Action {
		case "partition", "heal", "reset", "delay":
		default:
			return invalid("network v2 fault action")
		}
		if f.Action != "reset" && f.Port != 0 {
			return invalid("network v2 topology action cannot select a port")
		}
		if f.Action != "delay" && f.DelayNanos != 0 {
			return invalid("network v2 delay applies only to delay action")
		}
	}
	return nil
}
func newNetworkV2State(config *NetworkModelV2, epoch uint64) (*networkV2State, error) {
	if config == nil {
		return nil, nil
	}
	if err := validateNetworkModelV2(config); err != nil {
		return nil, err
	}
	ceiling, err := networkV2ClockCeiling(epoch)
	if err != nil {
		return nil, err
	}
	m, err := netmodel.New(1, netmodel.Config{Nodes: []netmodel.Node{{Node: "guest.client", Address: "127.0.0.2"}, {Node: "guest.server", Address: "127.0.0.1"}}, Links: []netmodel.Link{{From: "guest.client", To: "guest.server", Enabled: true}, {From: "guest.server", To: "guest.client", Enabled: true}}, Limits: config.Limits})
	if err != nil {
		return nil, err
	}
	if err := m.LimitClockCeiling(ceiling); err != nil {
		return nil, err
	}
	w, err := world.New(networkV2WorldConfig(config.Limits))
	if err != nil {
		return nil, err
	}
	n := &networkV2State{config: *config, model: m, world: w, client: netmodel.Domain{Run: 1, Node: "guest.client", Incarnation: 1, Address: "127.0.0.2"}, server: netmodel.Domain{Run: 1, Node: "guest.server", Incarnation: 1, Address: "127.0.0.1"}, handles: map[uint64]netmodel.Domain{}, ports: map[uint64]uint32{}, requests: map[uint64]netmodel.Domain{}, worldRequests: map[uint64]world.RequestID{}, worldReady: map[uint64]bool{}}
	n.config.Faults = slices.Clone(config.Faults)
	if err = m.Register(n.client); err != nil {
		return nil, err
	}
	if err = m.Register(n.server); err != nil {
		return nil, err
	}
	return n, nil
}

// networkV2WorldConfig is a node's network World; the cluster evidence decoder
// replays a helper-produced World against the same configuration.
func networkV2WorldConfig(limits netmodel.Limits) world.Config {
	return world.Config{Limits: world.Limits{MaxRequests: limits.Transitions * 2, MaxEvents: limits.Transitions * 2, MaxQueuedEvents: min(netmodel.MaximumPendingOperations, limits.Transitions*2), MaxTransitions: limits.Transitions * 8, MaxPayloadBytes: limits.Bytes * 8, MaxStringBytes: 1024}}
}

func networkV2ClockCeiling(epoch uint64) (uint64, error) {
	if epoch >= math.MaxInt64 {
		return 0, invalid("network v2 clock epoch")
	}
	return min(uint64(math.MaxInt64-int64(world.InitialTime)), math.MaxInt64-epoch), nil
}

func (e *Environment) runtimeNetworkStream(input json.RawMessage) (bytesOutput, error) {
	var request bytesOutput
	if err := decodeInput(input, &request, "data_base64"); err != nil {
		return bytesOutput{}, err
	}
	data := slices.Clone(request.Data)
	if len(data) < networkStreamHeader || len(data) > networkStreamHeader+netmodel.MaximumChunkBytes || binary.BigEndian.Uint32(data) != 2 || !allZero(data[40:48]) || !allZero(data[52:56]) || !allZero(data[60:64]) {
		return bytesOutput{}, invalid("network v2 stream layout")
	}
	op := binary.BigEndian.Uint32(data[4:])
	id := binary.BigEndian.Uint64(data[8:])
	handle := binary.BigEndian.Uint64(data[16:])
	deadline := binary.BigEndian.Uint64(data[24:])
	port := binary.BigEndian.Uint32(data[32:])
	count := binary.BigEndian.Uint32(data[36:])
	if op == 0 {
		if len(data) != 64 || id != 0 || handle != 0 || deadline != 0 || port != 0 || count != 0 {
			return bytesOutput{}, invalid("network v2 query")
		}
		if e.networkV2 == nil {
			binary.BigEndian.PutUint32(data[40:], 2)
		}
		return bytesOutput{Data: data}, nil
	}
	n := e.networkV2
	if op == 14 {
		if n == nil || e.runtime == nil || !e.runtime.configured || e.runtime.finished || !allZero(data[8:36]) || !allZero(data[40:64]) || count > 128 || len(data) != networkStreamHeader+8*int(count) {
			return bytesOutput{}, invalid("network v2 readiness probe layout")
		}
		seen := make(map[uint64]bool, count)
		for offset := networkStreamHeader; offset < len(data); offset += 8 {
			requestID := binary.BigEndian.Uint64(data[offset:])
			domain, exists := n.requests[requestID]
			if requestID == 0 || requestID >= 1<<48 || seen[requestID] || !exists || domain != n.client && domain != n.server || !n.model.Pending(domain, n.requestBase+requestID) {
				return bytesOutput{}, invalid("network v2 readiness probe request")
			}
			seen[requestID] = true
		}
		if err := e.networkV2Advance(e.now); err != nil {
			return bytesOutput{}, err
		}
		for offset := networkStreamHeader; offset < len(data); offset += 8 {
			if !n.worldReady[binary.BigEndian.Uint64(data[offset:])] {
				binary.BigEndian.PutUint64(data[offset:], 0)
			}
		}
		return bytesOutput{Data: data}, nil
	}
	if n == nil || e.runtime == nil || !e.runtime.configured || e.runtime.finished || id == 0 || id >= 1<<48 || count > netmodel.MaximumChunkBytes || port > 65535 {
		return bytesOutput{}, invalid("network v2 operation")
	}
	if err := e.networkV2Advance(e.now); err != nil {
		return bytesOutput{}, err
	}
	names := []string{"query", "listen", "dial", "accept", "read", "write", "close", "close_read", "close_write", "deadline", "read_deadline", "write_deadline", "poll", "cancel"}
	if int(op) >= len(names) {
		return bytesOutput{}, invalid("network v2 opcode")
	}
	if op == 5 {
		if len(data) != 64+int(count) {
			return bytesOutput{}, invalid("network v2 write payload")
		}
	} else if op == 4 && len(data) != 64+int(count) || op != 4 && op != 5 && op != 12 && len(data) != 64 {
		return bytesOutput{}, invalid("network v2 payload")
	}
	domain := n.handles[handle]
	switch op {
	case 1:
		domain = n.server
	case 2:
		domain = n.client
	case 12, 13:
		domain = n.requests[id]
	}
	host := n.server.Address
	if op == 2 {
		address := binary.BigEndian.Uint32(data[56:])
		if address != 0 && address>>24 != 127 {
			host = fmt.Sprintf("%d.%d.%d.%d", byte(address>>24), byte(address>>16), byte(address>>8), byte(address))
		}
	}
	r := netmodel.Request{Domain: domain, Op: names[op], Handle: handle, Host: host, Port: int(port), Count: count, Deadline: deadline, RequestID: n.requestBase + id}
	if op == 5 {
		r.Bytes = slices.Clone(data[64:])
	}
	var result netmodel.Result
	if op == 13 {
		err := n.model.Cancel(domain, r.RequestID)
		result = netmodel.Result{RequestID: r.RequestID, State: "completed"}
		if err != nil {
			// Cancel's closed owner returns only the unwrapped stale-handle
			// sentinel. Do not consult a platform error formatter.
			result.Error = "stale file handle"
		} else {
			n.model.Dispatch(netmodel.Request{Domain: domain, Op: "poll", RequestID: r.RequestID})
		}
		if wid := n.worldRequests[id]; wid != 0 {
			if _, err = n.world.Cancel(wid); err != nil {
				return bytesOutput{}, err
			}
		}
	} else {
		result = n.model.Dispatch(r)
	}
	result.RequestID = id
	if len(result.Bytes) > len(data)-networkStreamHeader {
		return bytesOutput{}, invalid("network v2 completed bytes exceed caller payload")
	}
	clear(data[36:56])
	if result.State == "pending" {
		n.requests[id] = domain
		if n.worldRequests[id] == 0 {
			wid, err := n.world.Register(world.Request{Kind: "network", Resource: world.ResourceID{Adapter: "wasm", Kind: "stream", Key: fmt.Sprint(id)}})
			if err != nil {
				return bytesOutput{}, err
			}
			n.worldRequests[id] = wid
		}
		binary.BigEndian.PutUint32(data[40:], 1)
	} else {
		if result.Handle != 0 {
			n.handles[result.Handle] = domain
			n.ports[result.Handle] = uint32(result.Remote.Port)
			if op == 1 || op == 3 {
				n.ports[result.Handle] = uint32(result.Local.Port)
			}
		}
		delete(n.requests, id)
		delete(n.worldRequests, id)
		delete(n.worldReady, id)
		binary.BigEndian.PutUint64(data[16:], result.Handle)
		binary.BigEndian.PutUint32(data[36:], uint32(result.Count))
		binary.BigEndian.PutUint32(data[44:], networkV2Errno(result.Error))
		binary.BigEndian.PutUint32(data[48:], uint32(result.Local.Port))
		binary.BigEndian.PutUint32(data[52:], uint32(result.Remote.Port))
		if op == 4 || op == 12 {
			copy(data[64:], result.Bytes)
		}
	}
	if err := n.appendLog(e, networkV2Operation{e.now, r, result}); err != nil {
		return bytesOutput{}, err
	}
	return bytesOutput{Data: data}, nil
}

// networkV2Operation is one request/result entry of the operations log; the
// cluster evidence decoder rebuilds entries with the same type.
type networkV2Operation struct {
	At      uint64           `json:"at"`
	Request netmodel.Request `json:"request"`
	Result  netmodel.Result  `json:"result"`
}

func networkV2Errno(message string) uint32 {
	switch message {
	case "", "EOF":
		return 0
	case "context canceled":
		return 11
	case "connection timed out":
		return netmodel.StreamStallTimeout
	case "context deadline exceeded", "i/o timeout", "file closing", "deadline exceeded":
		return 73
	case netmodel.ErrConnectionRefused.Error():
		return 14
	case netmodel.ErrResourceExhausted.Error():
		return 42
	case netmodel.ErrClosed.Error():
		return 15
	default:
		if strings.Contains(message, "deadline") {
			return 73
		}
		return 8
	}
}
func (n *networkV2State) appendLog(e *Environment, value any) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if uint64(len(n.log)+len(encoded)+1) > e.config.Limits.TranscriptBytes {
		return capacity("network-v2-transcript-bytes")
	}
	n.log = append(n.log, encoded...)
	n.log = append(n.log, '\n')
	return nil
}

func (e *Environment) networkV2Advance(now uint64) error {
	n := e.networkV2
	if n == nil {
		return nil
	}
	for n.fault < len(n.config.Faults) && n.config.Faults[n.fault].AtNanos <= now {
		f := n.config.Faults[n.fault]
		if _, err := n.model.AdvanceBefore(f.AtNanos); err != nil {
			return err
		}
		var err error
		switch f.Action {
		case "partition":
			err = n.model.Partition(n.client.Node, n.server.Node, true)
		case "heal":
			err = n.model.Heal(n.client.Node, n.server.Node, true)
		case "delay":
			err = n.model.Delay(n.client.Node, n.server.Node, f.DelayNanos, true)
		case "reset":
			handles := make([]uint64, 0, len(n.handles))
			for handle := range n.handles {
				if f.Port == 0 || n.ports[handle] == f.Port {
					handles = append(handles, handle)
				}
			}
			sort.Slice(handles, func(i, j int) bool { return handles[i] < handles[j] })
			for _, handle := range handles {
				resetErr := n.model.Reset(handle)
				if resetErr != nil && resetErr != syscall.ESTALE {
					err = resetErr
					break
				}
			}
		}
		if err != nil {
			return err
		}
		n.fault++
		if err = n.appendLog(e, struct {
			Fault NetworkFaultV2 `json:"fault"`
		}{f}); err != nil {
			return err
		}
	}
	if _, err := n.model.AdvanceBefore(now); err != nil {
		return err
	}
	ids, err := n.model.Advance(now)
	if err != nil {
		return err
	}
	newReadiness := false
	for _, globalID := range ids {
		if globalID < n.requestBase || globalID-n.requestBase >= 1<<48 {
			continue
		}
		id := globalID - n.requestBase
		if n.worldReady[id] {
			continue
		}
		wid := n.worldRequests[id]
		if wid == 0 {
			continue
		}
		if _, err = n.world.Ready(world.Readiness{RequestID: wid, At: world.InitialTime + world.LogicalTime(now), Kind: "network"}); err != nil {
			return err
		}
		n.worldReady[id] = true
		newReadiness = true
	}
	if newReadiness {
		q, err := n.world.Quiesce()
		if err != nil {
			return err
		}
		if q.After != world.InitialTime+world.LogicalTime(now) {
			return invalid("network readiness advanced outside idle")
		}
		if err = n.appendLog(e, q); err != nil {
			return err
		}
	}
	return nil
}
func (e *Environment) networkV2Deadline() uint64 {
	n := e.networkV2
	if n == nil {
		return 0
	}
	next := n.model.NextDeadline()
	if n.fault < len(n.config.Faults) {
		f := n.config.Faults[n.fault].AtNanos
		if next == 0 || f < next {
			next = f
		}
	}
	return next
}
func (e *Environment) NetworkV2Transcript() []byte {
	data, _ := e.NetworkV2Evidence()
	return data
}

func (e *Environment) NetworkV2Evidence() ([]byte, error) {
	if e.networkV2 == nil {
		return nil, nil
	}
	n := e.networkV2
	record, err := n.model.Record()
	if err != nil {
		return nil, err
	}
	return networkV2Document(n.log, record, n.world.Snapshot(), e.config.Limits.TranscriptBytes)
}

// networkV2Document is the per-instance evidence document, shared with the
// cluster evidence decoder so both produce identical bytes.
func networkV2Document(log []byte, record netmodel.Record, snapshot world.Snapshot, limit uint64) ([]byte, error) {
	encoded, err := json.Marshal(struct {
		Profile    string          `json:"profile"`
		Operations json.RawMessage `json:"operations"`
		Native     netmodel.Record `json:"native"`
		World      world.Snapshot  `json:"world"`
	}{NetworkStreamProfile, encodeNetworkOperations(log), record, snapshot})
	if err != nil {
		return nil, err
	}
	if uint64(len(encoded)) > limit {
		return nil, capacity("network-v2-evidence-bytes")
	}
	return encoded, nil
}
func encodeNetworkOperations(log []byte) json.RawMessage {
	lines := slices.DeleteFunc(strings.Split(strings.TrimSpace(string(log)), "\n"), func(s string) bool { return s == "" })
	return json.RawMessage("[" + strings.Join(lines, ",") + "]")
}

func (e *Environment) networkV2Idle(data []byte, deadline uint64) (bytesOutput, error) {
	if err := e.networkV2Advance(e.now); err != nil {
		return bytesOutput{}, err
	}
	n := e.networkV2
	next := e.networkV2Deadline()
	if next != 0 && (deadline == 0 || next < deadline) {
		deadline = next
	}
	if deadline > uint64(math.MaxInt64-int64(world.InitialTime)) || deadline > math.MaxInt64-e.config.Clock.EpochNanos {
		return bytesOutput{}, invalid("network v2 idle deadline overflow")
	}
	if deadline != 0 {
		id, err := n.world.Register(world.Request{Kind: "idle", Resource: world.ResourceID{Adapter: "wasm", Kind: "clock", Key: fmt.Sprint(e.now)}})
		if err != nil {
			return bytesOutput{}, err
		}
		if _, err = n.world.Ready(world.Readiness{RequestID: id, At: world.InitialTime + world.LogicalTime(max(e.now, deadline)), Kind: "idle"}); err != nil {
			return bytesOutput{}, err
		}
	}
	q, err := n.world.Quiesce()
	if err != nil {
		return bytesOutput{}, err
	}
	now := uint64(q.After - world.InitialTime)
	if now > e.now {
		e.advances++
	}
	e.now = max(e.now, now)
	if err = e.networkV2Advance(e.now); err != nil {
		return bytesOutput{}, err
	}
	// Network readiness may precede the guest's timer deadline.
	binary.BigEndian.PutUint32(data[4:], 1)
	if deadline == 0 && len(q.Deliveries) == 0 {
		binary.BigEndian.PutUint32(data[4:], 2)
	}
	binary.BigEndian.PutUint64(data[8:], e.now)
	if err = n.appendLog(e, q); err != nil {
		return bytesOutput{}, err
	}
	return bytesOutput{Data: data}, nil
}
