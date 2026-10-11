package wasi

import (
	"encoding/binary"
	"encoding/json"
	"math"
	"slices"
)

const (
	networkQuery   = 0
	networkConnect = 1
	networkWrite   = 2

	networkDeliver = 0
	networkRefuse  = 1
)

// NetworkModel declares the simulated in-guest network of the cooperative
// profile. The guest's fake TCP asks the model before each stream connect and
// write. Delays are drawn from a seeded stream in operation order; faults apply
// at virtual monotonic time. Port 0 in a fault matches every connection.
type NetworkModel struct {
	Seed          uint64         `json:"seed"`
	MaxDelayNanos uint64         `json:"max_delay_nanos"`
	Faults        []NetworkFault `json:"faults,omitempty"`
}

// NetworkFault is one scheduled action. A partition refuses connects and
// resets writes on matching connections until a heal for the same port. A
// reset fails the next write on every matching connection established before it.
type NetworkFault struct {
	AtNanos uint64 `json:"at_nanos"`
	Action  string `json:"action"`
	Port    uint32 `json:"port"`
}

// NetworkEvent is one model decision, retained as replay evidence.
type NetworkEvent struct {
	Sequence   uint64 `json:"seq"`
	AtNanos    uint64 `json:"at"`
	Op         string `json:"op"`
	Local      uint32 `json:"local"`
	Remote     uint32 `json:"remote"`
	Bytes      uint64 `json:"bytes,omitempty"`
	Action     string `json:"action"`
	DelayNanos uint64 `json:"delay,omitempty"`
}

type connectionKey struct{ low, high uint32 }

type networkState struct {
	model       NetworkModel
	nextFault   int
	partitioned map[uint32]bool
	resetting   map[connectionKey]bool
	open        map[connectionKey]bool
	sequence    uint64
	log         []byte
}

func validateNetworkModel(model *NetworkModel) error {
	if model == nil {
		return nil
	}
	if model.MaxDelayNanos > uint64(math.MaxInt64)/2 || len(model.Faults) > 4096 {
		return invalid("network model bounds")
	}
	for i, fault := range model.Faults {
		if fault.Action != "partition" && fault.Action != "heal" && fault.Action != "reset" || fault.AtNanos > math.MaxInt64 || i > 0 && fault.AtNanos < model.Faults[i-1].AtNanos {
			return invalid("network fault schedule")
		}
	}
	return nil
}

func newNetworkState(model *NetworkModel) *networkState {
	if model == nil {
		return nil
	}
	cloned := *model
	cloned.Faults = slices.Clone(model.Faults)
	return &networkState{model: cloned, partitioned: map[uint32]bool{}, resetting: map[connectionKey]bool{}, open: map[connectionKey]bool{}}
}

func keyFor(local, remote uint32) connectionKey {
	return connectionKey{min(local, remote), max(local, remote)}
}

func (n *networkState) matches(port uint32, key connectionKey) bool {
	return port == 0 || key.low == port || key.high == port
}

func (n *networkState) partitionedFor(key connectionKey) bool {
	return n.partitioned[0] || n.partitioned[key.low] || n.partitioned[key.high]
}

func (n *networkState) advance(now uint64) {
	for n.nextFault < len(n.model.Faults) && n.model.Faults[n.nextFault].AtNanos <= now {
		fault := n.model.Faults[n.nextFault]
		n.nextFault++
		switch fault.Action {
		case "partition":
			n.partitioned[fault.Port] = true
		case "heal":
			delete(n.partitioned, fault.Port)
		case "reset":
			for key := range n.open {
				if n.matches(fault.Port, key) {
					n.resetting[key] = true
				}
			}
		}
	}
}

// delay draws the next seeded delay with splitmix64 over the operation sequence.
func (n *networkState) delay() uint64 {
	if n.model.MaxDelayNanos == 0 {
		return 0
	}
	z := n.model.Seed + (n.sequence+1)*0x9e3779b97f4a7c15
	z = (z ^ z>>30) * 0xbf58476d1ce4e5b9
	z = (z ^ z>>27) * 0x94d049bb133111eb
	z ^= z >> 31
	return z % (n.model.MaxDelayNanos + 1)
}

func (e *Environment) runtimeNet(input json.RawMessage) (bytesOutput, error) {
	if !e.runtime.configured || e.runtime.finished {
		return bytesOutput{}, invalid("runtime network outside active lifetime")
	}
	var request bytesOutput
	if err := decodeInput(input, &request, "data_base64"); err != nil {
		return bytesOutput{}, err
	}
	data := slices.Clone(request.Data)
	if len(data) != 32 || binary.BigEndian.Uint32(data) != 1 || !allZero(data[24:]) {
		return bytesOutput{}, invalid("runtime network layout")
	}
	op := binary.BigEndian.Uint32(data[4:])
	local, remote := binary.BigEndian.Uint32(data[8:]), binary.BigEndian.Uint32(data[12:])
	count := binary.BigEndian.Uint64(data[16:])
	n := e.network
	if op == networkQuery {
		if local != 0 || remote != 0 || count != 0 {
			return bytesOutput{}, invalid("runtime network query")
		}
		if n != nil {
			binary.BigEndian.PutUint32(data[4:], networkRefuse)
		}
		return bytesOutput{Data: data}, nil
	}
	if n == nil || op != networkConnect && op != networkWrite || op == networkConnect && count != 0 {
		return bytesOutput{}, invalid("runtime network operation")
	}
	n.advance(e.now)
	key := keyFor(local, remote)
	event := NetworkEvent{Sequence: n.sequence, AtNanos: e.now, Local: local, Remote: remote, Bytes: count, Action: "deliver", DelayNanos: n.delay()}
	action := uint32(networkDeliver)
	switch op {
	case networkConnect:
		event.Op = "connect"
		if n.partitionedFor(key) {
			action, event.Action, event.DelayNanos = networkRefuse, "refuse", 0
		} else {
			n.open[key] = true
			delete(n.resetting, key)
		}
	case networkWrite:
		event.Op = "write"
		if n.partitionedFor(key) || n.resetting[key] {
			action, event.Action, event.DelayNanos = networkRefuse, "reset", 0
			delete(n.resetting, key)
			delete(n.open, key)
		} else {
			n.open[key] = true
		}
	}
	line, err := json.Marshal(event)
	if err != nil {
		return bytesOutput{}, err
	}
	if uint64(len(n.log)+len(line)+1) > e.config.Limits.TranscriptBytes {
		return bytesOutput{}, capacity("network-transcript-bytes")
	}
	n.log = append(append(n.log, line...), '\n')
	n.sequence++
	binary.BigEndian.PutUint32(data[4:], action)
	binary.BigEndian.PutUint64(data[24:], event.DelayNanos)
	return bytesOutput{Data: data}, nil
}

// NetworkTranscript returns the model decisions in guest operation order.
func (e *Environment) NetworkTranscript() []byte {
	if e.network == nil {
		return nil
	}
	return slices.Clone(e.network.log)
}
