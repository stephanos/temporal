package wasi

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"sort"
	"strconv"
	"strings"

	netmodel "go.temporal.io/server/tools/gomad/internal/model/network"
	"go.temporal.io/server/tools/gomad/internal/model/world"
)

type BrokerNetworkFault struct {
	Name, Action, From, To string
	AtNanos, DelayNanos    uint64
	Symmetric              bool
}

type brokerNetwork struct {
	model     *netmodel.Model
	addresses map[string]string
	clients   map[string]string
	faults    []BrokerNetworkFault
	fault     int
}

func newBrokerNetwork(config BrokerConfig, b *Broker) (*brokerNetwork, error) {
	if config.Network == nil {
		if len(config.Faults) != 0 {
			return nil, invalid("broker faults without network")
		}
		return nil, nil
	}
	if err := validateNetworkModelV2(config.Network); err != nil {
		return nil, err
	}
	if len(config.Network.Faults) != 0 {
		return nil, invalid("broker requires central fault schedule")
	}
	n := &brokerNetwork{addresses: map[string]string{}, clients: map[string]string{}, faults: slices.Clone(config.Faults)}
	var nodes []netmodel.Node
	owners := map[string]string{}
	for i, id := range b.order {
		address := config.Addresses[id]
		if address == "" {
			address = fmt.Sprintf("10.0.%d.%d", (i+1)/254, (i+1)%254+1)
		}
		client := fmt.Sprintf("10.1.%d.%d", (i+1)/254, (i+1)%254+1)
		if !brokerIPv4(address) {
			return nil, invalid("broker IPv4 address")
		}
		n.addresses[id] = address
		n.clients[id] = client
		nodes = append(nodes, netmodel.Node{Node: id + ".server", Address: address}, netmodel.Node{Node: id + ".client", Address: client})
		owners[id+".server"], owners[id+".client"] = id, id
	}
	var links []netmodel.Link
	topology := map[string]BrokerLink{}
	for _, link := range config.Links {
		key := link.From + "\x00" + link.To
		if link.From == link.To || n.addresses[link.From] == "" || n.addresses[link.To] == "" {
			return nil, invalid("broker link endpoint")
		}
		if _, ok := topology[key]; ok {
			return nil, invalid("broker duplicate link")
		}
		topology[key] = link
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].Node < nodes[j].Node })
	for _, from := range nodes {
		for _, to := range nodes {
			if from.Node != to.Node {
				link, ok := topology[owners[from.Node]+"\x00"+owners[to.Node]]
				if owners[from.Node] == owners[to.Node] || !config.StrictTopology && !ok {
					link.Enabled = true
				} else if !ok {
					continue
				}
				links = append(links, netmodel.Link{From: from.Node, To: to.Node, Enabled: link.Enabled, DelayNanos: link.DelayNanos})
			}
		}
	}
	var err error
	n.model, err = netmodel.New(1, netmodel.Config{Nodes: nodes, Links: links, Limits: config.Network.Limits})
	if err != nil {
		return nil, err
	}
	if err := n.model.LimitClockCeiling(uint64(math.MaxInt64 - int64(world.InitialTime))); err != nil {
		return nil, err
	}
	for _, id := range b.order {
		if err = n.register(b.sessions[id]); err != nil {
			return nil, err
		}
	}
	seen := map[string]bool{}
	for i, f := range n.faults {
		if f.Name == "" || seen[f.Name] || n.addresses[f.From] == "" || n.addresses[f.To] == "" || i > 0 && n.faults[i-1].AtNanos > f.AtNanos {
			return nil, invalid("broker fault schedule")
		}
		seen[f.Name] = true
		switch f.Action {
		case "partition", "heal", "delay":
		default:
			return nil, invalid("broker fault action")
		}
	}
	return n, n.advance(b.Current())
}

func brokerIPv4(address string) bool {
	parts := strings.Split(address, ".")
	if len(parts) != 4 {
		return false
	}
	for _, part := range parts {
		if len(part) == 0 || len(part) > 3 || len(part) > 1 && part[0] == '0' {
			return false
		}
		for _, digit := range part {
			if digit < '0' || digit > '9' {
				return false
			}
		}
		value, err := strconv.ParseUint(part, 10, 8)
		if err != nil || value > 255 {
			return false
		}
	}
	return true
}

func (n *brokerNetwork) domains(s *BrokerSession) (netmodel.Domain, netmodel.Domain) {
	return netmodel.Domain{Run: 1, Node: s.Node.ID + ".client", Incarnation: s.Node.Incarnation, Address: n.clients[s.Node.ID]}, netmodel.Domain{Run: 1, Node: s.Node.ID + ".server", Incarnation: s.Node.Incarnation, Address: n.addresses[s.Node.ID]}
}

func (n *brokerNetwork) register(s *BrokerSession) error {
	c, v := n.domains(s)
	return errors.Join(n.model.Register(c), n.model.Register(v))
}

func (n *brokerNetwork) attach(s *BrokerSession, e *Environment) error {
	if e.networkV2 == nil {
		return invalid("broker node requires stream network profile")
	}
	if e.networkV2.model == n.model {
		return n.advance(s.broker.Current())
	}
	if len(e.networkV2.config.Faults) != 0 {
		return invalid("broker node local fault schedule")
	}
	ceiling, err := networkV2ClockCeiling(e.config.Clock.EpochNanos)
	if err != nil {
		return err
	}
	if err := n.model.LimitClockCeiling(ceiling); err != nil {
		return err
	}
	e.networkV2.model = n.model
	e.networkV2.client, e.networkV2.server = n.domains(s)
	e.networkV2.requestBase = s.slot << 48
	return n.advance(s.broker.Current())
}

func (n *brokerNetwork) revoke(s *BrokerSession, graceful bool) error {
	c, v := n.domains(s)
	return errors.Join(n.model.Revoke(c, graceful), n.model.Revoke(v, graceful))
}

func (n *brokerNetwork) deadline() uint64 {
	deadline := n.model.NextDeadline()
	if n.fault < len(n.faults) {
		next := n.faults[n.fault].AtNanos
		if deadline == 0 || next < deadline {
			deadline = next
		}
	}
	return deadline
}

func (n *brokerNetwork) advance(now uint64) error {
	for n.fault < len(n.faults) && n.faults[n.fault].AtNanos <= now {
		f := n.faults[n.fault]
		if _, err := n.model.AdvanceBefore(f.AtNanos); err != nil {
			return err
		}
		if err := n.changeLink(f.Action, f.From, f.To, f.DelayNanos, f.Symmetric); err != nil {
			return err
		}
		n.fault++
	}
	if _, err := n.model.AdvanceBefore(now); err != nil {
		return err
	}
	_, err := n.model.Advance(now)
	return err
}

func (b *Broker) Address(node string) string {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.network == nil {
		return ""
	}
	return b.network.addresses[node]
}

func (b *Broker) NetworkSnapshot() (netmodel.Snapshot, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.network == nil {
		return netmodel.Snapshot{}, invalid("broker network unavailable")
	}
	return b.network.model.Snapshot()
}

func (b *Broker) NetworkRecord() (netmodel.Record, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.network == nil {
		return netmodel.Record{}, invalid("broker network unavailable")
	}
	return b.network.model.Record()
}

func (b *Broker) ChangeLink(action, from, to string, delay uint64, symmetric bool) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.network == nil || b.network.addresses[from] == "" || b.network.addresses[to] == "" || from == to {
		return invalid("broker link")
	}
	return b.network.changeLink(action, from, to, delay, symmetric)
}

func (n *brokerNetwork) changeLink(action, from, to string, delay uint64, symmetric bool) error {
	switch action {
	case "partition", "heal", "delay":
	default:
		return invalid("broker link action")
	}
	return n.model.ChangeGroup([]string{from + ".client", from + ".server"}, []string{to + ".client", to + ".server"}, action, delay, symmetric)
}
