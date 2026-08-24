// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gomadio

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"sort"
	"sync"
	"syscall"
	_ "unsafe"

	"internal/gomadsim"
)

const maximumSimulationConfigBytes = 128 << 20

var ErrReplayDiverged = errors.New("simulation network replay diverged")

type simulationNetworkConfig struct {
	Nodes  []simulationNodeConfig `json:"nodes"`
	Links  []simulationLink       `json:"links"`
	Limits simulationLimits       `json:"limits"`
	Replay *simulationRecord      `json:"replay,omitempty"`
}

type simulationLimits struct {
	Listeners   uint64 `json:"listeners"`
	Connections uint64 `json:"connections"`
	Deliveries  uint64 `json:"deliveries"`
	Bytes       uint64 `json:"bytes"`
	Transitions uint64 `json:"transitions"`
}

type simulationNodeConfig struct {
	Node    string `json:"node"`
	Address string `json:"address"`
}

type simulationLink struct {
	From       string `json:"from"`
	To         string `json:"to"`
	Enabled    bool   `json:"enabled"`
	DelayNanos uint64 `json:"delay_nanos"`
}

type simulationEndpoint struct {
	Node        string `json:"node,omitempty"`
	Incarnation uint64 `json:"incarnation,omitempty"`
	Address     string `json:"address,omitempty"`
	Port        uint64 `json:"port,omitempty"`
}

type simulationTransition struct {
	Ordinal       uint64             `json:"ordinal"`
	Kind          string             `json:"kind"`
	Source        simulationEndpoint `json:"source,omitempty"`
	Destination   simulationEndpoint `json:"destination,omitempty"`
	Connection    uint64             `json:"connection,omitempty"`
	Delivery      uint64             `json:"delivery,omitempty"`
	Bytes         uint64             `json:"bytes,omitempty"`
	Count         uint64             `json:"count,omitempty"`
	DelayNanos    uint64             `json:"delay_nanos,omitempty"`
	Outcome       string             `json:"outcome"`
	PayloadSHA256 string             `json:"payload_sha256,omitempty"`
}

type simulationNodeSnapshot struct {
	Node             string `json:"node"`
	Address          string `json:"address"`
	LastIncarnation  uint64 `json:"last_incarnation"`
	NextListenerPort uint64 `json:"next_listener_port"`
	NextClientPort   uint64 `json:"next_client_port"`
}

type simulationListenerSnapshot struct {
	Endpoint simulationEndpoint `json:"endpoint"`
	Closed   bool               `json:"closed"`
}

type simulationConnectionSnapshot struct {
	Identity uint64             `json:"identity"`
	Client   simulationEndpoint `json:"client"`
	Server   simulationEndpoint `json:"server"`
	Closed   bool               `json:"closed"`
	Reset    bool               `json:"reset"`
}

type simulationDeliverySnapshot struct {
	Identity    uint64             `json:"identity"`
	Connection  uint64             `json:"connection"`
	Source      simulationEndpoint `json:"source"`
	Destination simulationEndpoint `json:"destination"`
	Bytes       uint64             `json:"bytes"`
	DelayNanos  uint64             `json:"delay_nanos"`
}

type simulationSnapshot struct {
	Nodes            []simulationNodeSnapshot       `json:"nodes"`
	Links            []simulationLink               `json:"links"`
	Listeners        []simulationListenerSnapshot   `json:"listeners"`
	Connections      []simulationConnectionSnapshot `json:"connections"`
	Deliveries       []simulationDeliverySnapshot   `json:"deliveries"`
	NextConnection   uint64                         `json:"next_connection"`
	NextDelivery     uint64                         `json:"next_delivery"`
	TransitionSHA256 string                         `json:"transition_sha256"`
	Identity         string                         `json:"identity"`
}

type simulationRecord struct {
	Transitions []simulationTransition `json:"transitions"`
	Snapshot    simulationSnapshot     `json:"snapshot"`
}

type simulationBridgeError struct {
	Kind           string                `json:"kind"`
	Message        string                `json:"message"`
	Ordinal        uint64                `json:"ordinal,omitempty"`
	ExpectedSHA256 string                `json:"expected_sha256,omitempty"`
	ActualSHA256   string                `json:"actual_sha256,omitempty"`
	Expected       *simulationTransition `json:"expected,omitempty"`
	Actual         *simulationTransition `json:"actual,omitempty"`
}

type simulationNode struct {
	node             string
	address          string
	lastIncarnation  uint64
	nextListenerPort int
	nextClientPort   int
}

type simulationConnection struct {
	identity    uint64
	client      simulationEndpoint
	server      simulationEndpoint
	clientState *connState
	serverState *connState
	closed      bool
	reset       bool
}

type simulationDelivery struct {
	identity    uint64
	connection  uint64
	source      simulationEndpoint
	destination simulationEndpoint
	bytes       uint64
	delayNanos  uint64
}

type simulationNetwork struct {
	sync.Mutex
	run             uint64
	limits          simulationLimits
	nodes           map[string]*simulationNode
	addresses       map[string]*simulationNode
	links           map[string]simulationLink
	listeners       map[Address]*Listener
	listenerHistory []*Listener
	connections     map[uint64]*simulationConnection
	deliveries      map[uint64]simulationDelivery
	nextConnection  uint64
	nextDelivery    uint64
	pendingBytes    uint64
	transitions     []simulationTransition
	replay          *simulationRecord
	replayLanes     map[string][]simulationTransition
	replayNext      map[string]int
	divergence      *simulationBridgeError
	changed         chan struct{}
	finished        bool
}

var simulationNetworkState = struct {
	sync.Mutex
	runs map[uint64]*simulationNetwork
}{runs: make(map[uint64]*simulationNetwork)}

//go:linkname BeginSimulation
func BeginSimulation(run uint64, encoded []byte) ([]byte, bool) {
	config, err := decodeSimulationConfig(encoded)
	if err != nil {
		return encodeSimulationBridgeError(err), false
	}
	network, err := newSimulationNetwork(run, config)
	if err != nil {
		return encodeSimulationBridgeError(err), false
	}
	simulationNetworkState.Lock()
	defer simulationNetworkState.Unlock()
	if run == 0 || simulationNetworkState.runs[run] != nil {
		return encodeSimulationBridgeError(errors.New("simulation network run is invalid or duplicated")), false
	}
	simulationNetworkState.runs[run] = network
	return nil, true
}

//go:linkname PartitionSimulation
func PartitionSimulation(run uint64, left, right string, symmetric bool) ([]byte, bool) {
	if symmetric {
		return changeSimulationPair(run, left, right, "partition", false, 0)
	}
	return changeSimulationLink(run, left, right, "disconnect", false, 0)
}

//go:linkname HealSimulation
func HealSimulation(run uint64, left, right string, symmetric bool) ([]byte, bool) {
	if symmetric {
		return changeSimulationPair(run, left, right, "heal", true, 0)
	}
	return changeSimulationLink(run, left, right, "reconnect", true, 0)
}

//go:linkname DelaySimulation
func DelaySimulation(run uint64, left, right string, delayNanos uint64, symmetric bool) ([]byte, bool) {
	if symmetric {
		return changeSimulationPair(run, left, right, "delay", true, delayNanos)
	}
	return changeSimulationLink(run, left, right, "directional_delay", true, delayNanos)
}

//go:linkname ChangeSimulationGroup
func ChangeSimulationGroup(run uint64, left, right []string, enabled bool) ([]byte, bool) {
	network := simulationNetworkForRun(run)
	if network == nil {
		return encodeSimulationBridgeError(errors.New("simulation network run is unavailable")), false
	}
	if err := network.changeGroup(left, right, enabled); err != nil {
		return encodeSimulationBridgeError(err), false
	}
	return nil, true
}

//go:linkname RevokeSimulation
func RevokeSimulation(domainToken uint64, graceful bool) ([]byte, bool) {
	domain, ok := gomadsim.DescribeNetworkDomain(domainToken)
	if !ok {
		return encodeSimulationBridgeError(syscall.ESTALE), false
	}
	network := simulationNetworkForRun(domain.Run)
	if network == nil {
		return encodeSimulationBridgeError(errors.New("simulation network run is unavailable")), false
	}
	if err := network.revoke(domain, graceful); err != nil {
		return encodeSimulationBridgeError(err), false
	}
	revokeProcessNetworkResources(domainToken)
	return nil, true
}

//go:linkname FinishSimulation
func FinishSimulation(run uint64) ([]byte, bool) {
	simulationNetworkState.Lock()
	network := simulationNetworkState.runs[run]
	if network != nil {
		delete(simulationNetworkState.runs, run)
	}
	simulationNetworkState.Unlock()
	if network == nil {
		return encodeSimulationBridgeError(errors.New("simulation network run is unavailable")), false
	}
	record, err := network.finish()
	var bridge *simulationBridgeError
	if err != nil {
		if !errors.As(err, &bridge) {
			bridge = &simulationBridgeError{Kind: "runtime", Message: err.Error()}
		}
	}
	encoded, err := encodeSimulationFinishResponse(record, bridge)
	if err != nil {
		return encodeSimulationBridgeError(fmt.Errorf("encode simulation network record: %w", err)), false
	}
	return encoded, true
}

func newSimulationNetwork(run uint64, config simulationNetworkConfig) (*simulationNetwork, error) {
	if run == 0 || config.Limits.Listeners == 0 || config.Limits.Connections == 0 || config.Limits.Deliveries == 0 || config.Limits.Bytes == 0 || config.Limits.Transitions == 0 {
		return nil, errors.New("simulation network limits are invalid")
	}
	network := &simulationNetwork{
		run:         run,
		limits:      config.Limits,
		nodes:       make(map[string]*simulationNode, len(config.Nodes)),
		addresses:   make(map[string]*simulationNode, len(config.Nodes)),
		links:       make(map[string]simulationLink, len(config.Links)),
		listeners:   make(map[Address]*Listener),
		connections: make(map[uint64]*simulationConnection),
		deliveries:  make(map[uint64]simulationDelivery),
		changed:     make(chan struct{}),
		replay:      config.Replay,
	}
	if config.Replay != nil {
		network.replayLanes = make(map[string][]simulationTransition)
		network.replayNext = make(map[string]int)
		for _, transition := range config.Replay.Transitions {
			lane := simulationTransitionLane(transition)
			network.replayLanes[lane] = append(network.replayLanes[lane], transition)
		}
	}
	previous := ""
	for _, configNode := range config.Nodes {
		if configNode.Node <= previous || configNode.Address == "" || network.addresses[configNode.Address] != nil {
			return nil, errors.New("simulation network nodes are invalid or unordered")
		}
		node := &simulationNode{node: configNode.Node, address: configNode.Address, nextListenerPort: firstListenerPort, nextClientPort: firstClientPort}
		network.nodes[node.node] = node
		network.addresses[node.address] = node
		previous = configNode.Node
	}
	previous = ""
	for _, link := range config.Links {
		key := simulationLinkKey(link.From, link.To)
		if key <= previous || network.nodes[link.From] == nil || network.nodes[link.To] == nil || link.From == link.To {
			return nil, errors.New("simulation network links are invalid or unordered")
		}
		network.links[key] = link
		previous = key
	}
	return network, nil
}

func simulationNetworkForRun(run uint64) *simulationNetwork {
	simulationNetworkState.Lock()
	defer simulationNetworkState.Unlock()
	return simulationNetworkState.runs[run]
}

func currentSimulationNetwork() (*simulationNetwork, gomadsim.NetworkDomain, error, bool) {
	domain, err, handled := gomadsim.CurrentNetworkDomain()
	if !handled || err != nil {
		return nil, domain, err, handled
	}
	network := simulationNetworkForRun(domain.Run)
	if network == nil {
		return nil, domain, syscall.ESTALE, true
	}
	network.Lock()
	node := network.nodes[domain.Node]
	valid := node != nil && node.address == domain.Address
	if valid && domain.Incarnation > node.lastIncarnation {
		node.lastIncarnation = domain.Incarnation
	}
	network.Unlock()
	if !valid {
		return nil, domain, syscall.ESTALE, true
	}
	return network, domain, nil, true
}

func (network *simulationNetwork) listen(networkName, host string, requestedPort int, domain gomadsim.NetworkDomain) (*Listener, error) {
	network.Lock()
	defer network.Unlock()
	if network.finished || host != "" && host != "0.0.0.0" && host != domain.Address {
		return nil, ErrUnsupported
	}
	node := network.nodes[domain.Node]
	if node == nil || node.address != domain.Address {
		return nil, syscall.ESTALE
	}
	port := requestedPort
	if port == 0 {
		port = node.nextListenerPort
		for port <= maximumPort && network.listeners[Address{IP: domain.Address, Port: port}] != nil {
			port++
		}
		if port > maximumPort {
			return nil, network.recordFailureLocked("listen", simulationEndpoint{Node: domain.Node, Incarnation: domain.Incarnation, Address: domain.Address}, simulationEndpoint{}, "capacity")
		}
	}
	address := Address{IP: domain.Address, Port: port}
	if network.listeners[address] != nil {
		return nil, network.recordFailureLocked("listen", simulationEndpoint{Node: domain.Node, Incarnation: domain.Incarnation, Address: domain.Address, Port: uint64(port)}, simulationEndpoint{}, "closed")
	}
	if uint64(len(network.listenerHistory)) >= network.limits.Listeners {
		return nil, network.recordFailureLocked("listen", simulationEndpoint{Node: domain.Node, Incarnation: domain.Incarnation, Address: domain.Address, Port: uint64(port)}, simulationEndpoint{}, "capacity")
	}
	owner := simulationEndpoint{Node: domain.Node, Incarnation: domain.Incarnation, Address: domain.Address, Port: uint64(port)}
	if err := network.commitTransitionLocked(simulationTransition{Kind: "listen", Source: owner, Outcome: "ok"}); err != nil {
		return nil, err
	}
	if requestedPort == 0 {
		node.nextListenerPort = port + 1
	}
	listener := &Listener{address: address, owner: owner, network: network, changed: make(chan struct{})}
	network.listeners[address] = listener
	network.listenerHistory = append(network.listenerHistory, listener)
	network.signalLocked()
	return listener, nil
}

func (network *simulationNetwork) dial(ctx context.Context, networkName, host string, port int, domain gomadsim.NetworkDomain) (*Conn, error) {
	if host == "" {
		return nil, ErrUnsupported
	}
	for {
		if err := ctx.Err(); err != nil {
			network.Lock()
			source := simulationEndpoint{Node: domain.Node, Incarnation: domain.Incarnation, Address: domain.Address}
			destination := simulationEndpoint{Address: host, Port: uint64(port)}
			if target := network.addresses[host]; target != nil {
				destination.Node = target.node
			}
			recordErr := network.commitTransitionLocked(simulationTransition{Kind: "dial", Source: source, Destination: destination, Outcome: "deadline"})
			network.Unlock()
			if recordErr != nil {
				return nil, recordErr
			}
			return nil, err
		}
		network.Lock()
		if network.finished {
			network.Unlock()
			return nil, syscall.ESTALE
		}
		sourceNode := network.nodes[domain.Node]
		targetNode := network.addresses[host]
		source := simulationEndpoint{Node: domain.Node, Incarnation: domain.Incarnation, Address: domain.Address}
		destination := simulationEndpoint{Address: host, Port: uint64(port)}
		if targetNode != nil {
			destination.Node = targetNode.node
		}
		if sourceNode == nil || sourceNode.address != domain.Address || targetNode == nil {
			err := network.commitTransitionLocked(simulationTransition{Kind: "dial", Source: source, Destination: destination, Outcome: "refused"})
			network.Unlock()
			if err != nil {
				return nil, err
			}
			return nil, ErrConnectionRefused
		}
		link, linked := network.links[simulationLinkKey(domain.Node, targetNode.node)]
		if !linked || !link.Enabled {
			changed := network.changed
			network.Unlock()
			select {
			case <-changed:
			case <-ctx.Done():
			}
			continue
		}
		listener := network.listeners[Address{IP: host, Port: port}]
		if listener == nil || listener.closed {
			err := network.commitTransitionLocked(simulationTransition{Kind: "dial", Source: source, Destination: destination, Outcome: "refused"})
			network.Unlock()
			if err != nil {
				return nil, err
			}
			return nil, ErrConnectionRefused
		}
		listener.mu.Lock()
		if len(listener.pending) >= maximumPendingConns {
			changed := listener.changed
			listener.mu.Unlock()
			network.Unlock()
			select {
			case <-changed:
			case <-ctx.Done():
			}
			continue
		}
		if uint64(len(network.connections)) >= network.limits.Connections || sourceNode.nextClientPort > maximumPort {
			listener.mu.Unlock()
			err := network.commitTransitionLocked(simulationTransition{Kind: "dial", Source: source, Destination: destination, Outcome: "capacity"})
			network.Unlock()
			if err != nil {
				return nil, err
			}
			return nil, ErrResourceExhausted
		}
		clientPort := sourceNode.nextClientPort
		clientEndpoint := source
		clientEndpoint.Port = uint64(clientPort)
		serverEndpoint := listener.owner
		identity := network.nextConnection + 1
		transition := simulationTransition{Kind: "dial", Source: clientEndpoint, Destination: serverEndpoint, Connection: identity, DelayNanos: link.DelayNanos, Outcome: "ok"}
		if err := network.commitTransitionLocked(transition); err != nil {
			listener.mu.Unlock()
			network.Unlock()
			return nil, err
		}
		sourceNode.nextClientPort++
		network.nextConnection = identity
		clientState, serverState := newConnStates()
		clientAddress := Address{IP: domain.Address, Port: clientPort}
		serverAddress := listener.address
		client := &Conn{local: clientAddress, remote: serverAddress, owner: clientEndpoint, target: serverEndpoint, network: network, identity: identity, state: clientState, peer: serverState}
		server := &Conn{local: serverAddress, remote: clientAddress, owner: serverEndpoint, target: clientEndpoint, network: network, identity: identity, state: serverState, peer: clientState}
		network.connections[identity] = &simulationConnection{identity: identity, client: clientEndpoint, server: serverEndpoint, clientState: clientState, serverState: serverState}
		listener.pending = append(listener.pending, server)
		listener.signal()
		listener.mu.Unlock()
		network.signalLocked()
		network.Unlock()
		return client, nil
	}
}

func (network *simulationNetwork) accept(listener *Listener) (*Conn, error) {
	for {
		if err := validateSimulationEndpoint(network, listener.owner); err != nil {
			return nil, err
		}
		network.Lock()
		listener.mu.Lock()
		if len(listener.pending) != 0 {
			connection := listener.pending[0]
			transition := simulationTransition{Kind: "accept", Source: listener.owner, Destination: connection.target, Connection: connection.identity, Outcome: "ok"}
			if err := network.commitTransitionLocked(transition); err != nil {
				listener.mu.Unlock()
				network.Unlock()
				return nil, err
			}
			listener.pending = listener.pending[1:]
			listener.signal()
			listener.mu.Unlock()
			network.Unlock()
			return connection, nil
		}
		if listener.closed {
			listener.mu.Unlock()
			network.Unlock()
			return nil, ErrClosed
		}
		deadline := listener.deadline
		if deadlineExpired(deadline) {
			if err := network.commitTransitionLocked(simulationTransition{Kind: "accept", Source: listener.owner, Outcome: "deadline"}); err != nil {
				listener.mu.Unlock()
				network.Unlock()
				return nil, err
			}
			listener.mu.Unlock()
			network.Unlock()
			return nil, os.ErrDeadlineExceeded
		}
		changed := listener.changed
		listener.mu.Unlock()
		network.Unlock()
		waitForChange(changed, deadline)
	}
}

func (network *simulationNetwork) closeListener(listener *Listener) error {
	if err := validateSimulationEndpoint(network, listener.owner); err != nil {
		return err
	}
	network.Lock()
	defer network.Unlock()
	listener.mu.Lock()
	defer listener.mu.Unlock()
	if listener.closed {
		return ErrClosed
	}
	if err := network.commitTransitionLocked(simulationTransition{Kind: "listener_close", Source: listener.owner, Outcome: "ok"}); err != nil {
		return err
	}
	if network.listeners[listener.address] == listener {
		delete(network.listeners, listener.address)
	}
	listener.closed = true
	listener.signal()
	network.signalLocked()
	return nil
}

func validateSimulationEndpoint(network *simulationNetwork, endpoint simulationEndpoint) error {
	current, err, handled := gomadsim.CurrentNetworkDomain()
	if !handled || err != nil {
		if err != nil {
			return err
		}
		return ErrUnsupported
	}
	if current.Run != network.run || current.Node != endpoint.Node || current.Address != endpoint.Address || current.Incarnation != endpoint.Incarnation {
		return syscall.ESTALE
	}
	return nil
}

func (network *simulationNetwork) recordFailureLocked(kind string, source, destination simulationEndpoint, outcome string) error {
	if err := network.commitTransitionLocked(simulationTransition{Kind: kind, Source: source, Destination: destination, Outcome: outcome}); err != nil {
		return err
	}
	switch outcome {
	case "capacity":
		return ErrResourceExhausted
	case "closed":
		return ErrAddressInUse
	default:
		return ErrUnsupported
	}
}

func changeSimulationPair(run uint64, left, right, kind string, enabled bool, delayNanos uint64) ([]byte, bool) {
	network := simulationNetworkForRun(run)
	if network == nil {
		return encodeSimulationBridgeError(errors.New("simulation network run is unavailable")), false
	}
	if err := network.changePair(left, right, kind, enabled, delayNanos); err != nil {
		return encodeSimulationBridgeError(err), false
	}
	return nil, true
}

func changeSimulationLink(run uint64, from, to, kind string, enabled bool, delayNanos uint64) ([]byte, bool) {
	network := simulationNetworkForRun(run)
	if network == nil {
		return encodeSimulationBridgeError(errors.New("simulation network run is unavailable")), false
	}
	if err := network.changeLink(from, to, kind, enabled, delayNanos); err != nil {
		return encodeSimulationBridgeError(err), false
	}
	return nil, true
}

func (network *simulationNetwork) changeLink(from, to, kind string, enabled bool, delayNanos uint64) error {
	network.Lock()
	defer network.Unlock()
	if network.finished || from == to || network.nodes[from] == nil || network.nodes[to] == nil {
		return ErrUnsupported
	}
	key := simulationLinkKey(from, to)
	link, ok := network.links[key]
	if !ok {
		return ErrUnsupported
	}
	transition := simulationTransition{
		Kind:        kind,
		Source:      simulationEndpoint{Node: from, Address: network.nodes[from].address},
		Destination: simulationEndpoint{Node: to, Address: network.nodes[to].address},
		Count:       1,
		DelayNanos:  delayNanos,
		Outcome:     "ok",
	}
	if err := network.commitTransitionLocked(transition); err != nil {
		return err
	}
	switch kind {
	case "disconnect", "reconnect":
		link.Enabled = enabled
	case "directional_delay":
		link.DelayNanos = delayNanos
	default:
		return ErrUnsupported
	}
	network.links[key] = link
	network.signalLocked()
	return nil
}

func (network *simulationNetwork) changeGroup(left, right []string, enabled bool) error {
	network.Lock()
	defer network.Unlock()
	if network.finished || !validSimulationNodeGroup(left) || !validSimulationNodeGroup(right) {
		return ErrUnsupported
	}
	for _, leftNode := range left {
		if network.nodes[leftNode] == nil {
			return ErrUnsupported
		}
		for _, rightNode := range right {
			if leftNode == rightNode || network.nodes[rightNode] == nil {
				return ErrUnsupported
			}
			if _, ok := network.links[simulationLinkKey(leftNode, rightNode)]; !ok {
				return ErrUnsupported
			}
			if _, ok := network.links[simulationLinkKey(rightNode, leftNode)]; !ok {
				return ErrUnsupported
			}
		}
	}
	required := uint64(len(left) * len(right))
	if uint64(len(network.transitions))+required > network.limits.Transitions {
		return ErrResourceExhausted
	}
	kind := "partition"
	if enabled {
		kind = "heal"
	}
	transitions := make([]simulationTransition, 0, required)
	for _, leftNode := range left {
		for _, rightNode := range right {
			transitions = append(transitions, simulationTransition{
				Kind:        kind,
				Source:      simulationEndpoint{Node: leftNode, Address: network.nodes[leftNode].address},
				Destination: simulationEndpoint{Node: rightNode, Address: network.nodes[rightNode].address},
				Count:       2, Outcome: "ok",
			})
		}
	}
	start := len(network.transitions)
	replayNext := make(map[string]int, len(network.replayNext))
	for lane, next := range network.replayNext {
		replayNext[lane] = next
	}
	for _, transition := range transitions {
		if err := network.commitTransitionLocked(transition); err != nil {
			network.transitions = network.transitions[:start]
			network.replayNext = replayNext
			return err
		}
	}
	for _, leftNode := range left {
		for _, rightNode := range right {
			forwardKey := simulationLinkKey(leftNode, rightNode)
			reverseKey := simulationLinkKey(rightNode, leftNode)
			forward := network.links[forwardKey]
			reverse := network.links[reverseKey]
			forward.Enabled = enabled
			reverse.Enabled = enabled
			network.links[forwardKey] = forward
			network.links[reverseKey] = reverse
		}
	}
	network.signalLocked()
	return nil
}

func validSimulationNodeGroup(nodes []string) bool {
	if len(nodes) == 0 {
		return false
	}
	for index, node := range nodes {
		if node == "" || index != 0 && nodes[index-1] >= node {
			return false
		}
	}
	return true
}

func (network *simulationNetwork) changePair(left, right, kind string, enabled bool, delayNanos uint64) error {
	network.Lock()
	defer network.Unlock()
	if network.finished || left == right || network.nodes[left] == nil || network.nodes[right] == nil {
		return ErrUnsupported
	}
	forwardKey := simulationLinkKey(left, right)
	reverseKey := simulationLinkKey(right, left)
	forward, forwardOK := network.links[forwardKey]
	reverse, reverseOK := network.links[reverseKey]
	if !forwardOK || !reverseOK {
		return ErrUnsupported
	}
	transition := simulationTransition{
		Kind:        kind,
		Source:      simulationEndpoint{Node: left, Address: network.nodes[left].address},
		Destination: simulationEndpoint{Node: right, Address: network.nodes[right].address},
		Count:       2,
		DelayNanos:  delayNanos,
		Outcome:     "ok",
	}
	if err := network.commitTransitionLocked(transition); err != nil {
		return err
	}
	switch kind {
	case "partition", "heal":
		forward.Enabled = enabled
		reverse.Enabled = enabled
	case "delay":
		forward.DelayNanos = delayNanos
		reverse.DelayNanos = delayNanos
	default:
		return ErrUnsupported
	}
	network.links[forwardKey] = forward
	network.links[reverseKey] = reverse
	network.signalLocked()
	return nil
}

func (network *simulationNetwork) revoke(domain gomadsim.NetworkDomain, graceful bool) error {
	network.Lock()
	defer network.Unlock()
	if network.finished {
		return syscall.ESTALE
	}
	if node := network.nodes[domain.Node]; node != nil && domain.Incarnation > node.lastIncarnation {
		node.lastIncarnation = domain.Incarnation
	}
	endpoint := simulationEndpoint{Node: domain.Node, Incarnation: domain.Incarnation, Address: domain.Address}
	var count uint64
	var listeners uint64
	var pendingBytes uint64
	for _, connection := range network.connections {
		if sameSimulationIncarnation(connection.client, endpoint) || sameSimulationIncarnation(connection.server, endpoint) {
			count++
			for _, chunk := range connection.clientState.incoming {
				pendingBytes += uint64(len(chunk.bytes))
			}
			for _, chunk := range connection.serverState.incoming {
				pendingBytes += uint64(len(chunk.bytes))
			}
		}
	}
	for _, listener := range network.listeners {
		if sameSimulationIncarnation(listener.owner, endpoint) {
			listeners++
		}
	}
	kind := "crash"
	outcome := "reset"
	if graceful {
		kind = "stop"
		outcome = "ok"
	}
	if count != 0 || listeners != 0 {
		if err := network.commitTransitionLocked(simulationTransition{Kind: kind, Source: endpoint, Count: count + listeners, Bytes: pendingBytes, Outcome: outcome}); err != nil {
			return err
		}
	}
	for address, listener := range network.listeners {
		if sameSimulationIncarnation(listener.owner, endpoint) {
			listener.mu.Lock()
			listener.closed = true
			listener.signal()
			listener.mu.Unlock()
			delete(network.listeners, address)
		}
	}
	for _, connection := range network.connections {
		if !sameSimulationIncarnation(connection.client, endpoint) && !sameSimulationIncarnation(connection.server, endpoint) {
			continue
		}
		connection.clientState.shared.Lock()
		network.dropStateDeliveriesLocked(connection.clientState)
		network.dropStateDeliveriesLocked(connection.serverState)
		if graceful {
			local := connection.serverState
			if sameSimulationIncarnation(connection.client, endpoint) {
				local = connection.clientState
			}
			local.readClosed = true
			local.writeClosed = true
			connection.closed = connection.clientState.readClosed && connection.clientState.writeClosed && connection.serverState.readClosed && connection.serverState.writeClosed
		} else {
			connection.closed = true
			connection.reset = true
			connection.clientState.readClosed = true
			connection.clientState.writeClosed = true
			connection.serverState.readClosed = true
			connection.serverState.writeClosed = true
			connection.clientState.reset = true
			connection.serverState.reset = true
		}
		connection.clientState.shared.signal()
		connection.clientState.shared.Unlock()
	}
	network.signalLocked()
	return nil
}

func (network *simulationNetwork) dropStateDeliveriesLocked(state *connState) {
	for _, chunk := range state.incoming {
		delete(network.deliveries, chunk.identity)
		network.pendingBytes -= uint64(len(chunk.bytes))
	}
	state.incoming = nil
}

func (network *simulationNetwork) finish() (simulationRecord, error) {
	network.Lock()
	defer network.Unlock()
	if network.finished {
		return simulationRecord{}, errors.New("simulation network is already finished")
	}
	network.finished = true
	canonicalizeSimulationTransitions(network.transitions)
	var finishErr error
	if network.divergence != nil {
		finishErr = network.divergence
	}
	if finishErr == nil && network.replay != nil && len(network.transitions) != len(network.replay.Transitions) {
		expected := network.firstMissingReplayTransitionLocked()
		ordinal := expected.Ordinal
		replayErr, err := newSimulationReplayError(ordinal, expected, simulationTransition{})
		if err != nil {
			return simulationRecord{}, err
		}
		finishErr = replayErr
	}
	snapshot, err := network.snapshotLocked()
	if err != nil {
		return simulationRecord{}, err
	}
	record := simulationRecord{Transitions: append([]simulationTransition(nil), network.transitions...), Snapshot: snapshot}
	if finishErr == nil && network.replay != nil && network.replay.Snapshot.Identity != snapshot.Identity {
		finishErr = &simulationBridgeError{
			Kind: "replay", Message: ErrReplayDiverged.Error(), Ordinal: uint64(len(network.transitions)),
			ExpectedSHA256: network.replay.Snapshot.Identity, ActualSHA256: snapshot.Identity,
		}
	}
	return record, finishErr
}

func (network *simulationNetwork) snapshotLocked() (simulationSnapshot, error) {
	encoded, err := encodeSimulationTransitions(network.transitions)
	if err != nil {
		return simulationSnapshot{}, fmt.Errorf("encode simulation network transitions: %w", err)
	}
	payload := append([]byte("gomad3-simulation-network-transitions/v2\x00"), encoded...)
	digest := sha256.Sum256(payload)
	snapshot := simulationSnapshot{
		TransitionSHA256: "sha256:" + hex.EncodeToString(digest[:]),
		NextConnection:   network.nextConnection,
		NextDelivery:     network.nextDelivery,
	}
	for _, node := range network.nodes {
		snapshot.Nodes = append(snapshot.Nodes, simulationNodeSnapshot{
			Node: node.node, Address: node.address, LastIncarnation: node.lastIncarnation,
			NextListenerPort: uint64(node.nextListenerPort), NextClientPort: uint64(node.nextClientPort),
		})
	}
	sort.Slice(snapshot.Nodes, func(left, right int) bool { return snapshot.Nodes[left].Node < snapshot.Nodes[right].Node })
	for _, link := range network.links {
		snapshot.Links = append(snapshot.Links, link)
	}
	sort.Slice(snapshot.Links, func(left, right int) bool {
		return simulationLinkKey(snapshot.Links[left].From, snapshot.Links[left].To) < simulationLinkKey(snapshot.Links[right].From, snapshot.Links[right].To)
	})
	for _, listener := range network.listenerHistory {
		snapshot.Listeners = append(snapshot.Listeners, simulationListenerSnapshot{Endpoint: listener.owner, Closed: listener.closed})
	}
	sort.Slice(snapshot.Listeners, func(left, right int) bool {
		return simulationEndpointKey(snapshot.Listeners[left].Endpoint) < simulationEndpointKey(snapshot.Listeners[right].Endpoint)
	})
	for _, connection := range network.connections {
		snapshot.Connections = append(snapshot.Connections, simulationConnectionSnapshot{
			Identity: connection.identity, Client: connection.client, Server: connection.server, Closed: connection.closed, Reset: connection.reset,
		})
	}
	sort.Slice(snapshot.Connections, func(left, right int) bool {
		return snapshot.Connections[left].Identity < snapshot.Connections[right].Identity
	})
	for _, delivery := range network.deliveries {
		snapshot.Deliveries = append(snapshot.Deliveries, simulationDeliverySnapshot{
			Identity: delivery.identity, Connection: delivery.connection, Source: delivery.source, Destination: delivery.destination,
			Bytes: delivery.bytes, DelayNanos: delivery.delayNanos,
		})
	}
	sort.Slice(snapshot.Deliveries, func(left, right int) bool {
		return snapshot.Deliveries[left].Identity < snapshot.Deliveries[right].Identity
	})
	encodedSnapshot, err := encodeSimulationSnapshotIdentity(snapshot)
	if err != nil {
		return simulationSnapshot{}, fmt.Errorf("encode simulation network snapshot: %w", err)
	}
	snapshotDigest := sha256.Sum256(append([]byte("gomad3-simulation-network-snapshot/v2\x00"), encodedSnapshot...))
	snapshot.Identity = "sha256:" + hex.EncodeToString(snapshotDigest[:])
	return snapshot, nil
}

func (network *simulationNetwork) commitTransitionLocked(transition simulationTransition) error {
	ordinal := uint64(len(network.transitions))
	if ordinal >= network.limits.Transitions {
		return ErrResourceExhausted
	}
	if network.replay != nil {
		lane := simulationTransitionLane(transition)
		expectedLane := network.replayLanes[lane]
		next := network.replayNext[lane]
		var expected simulationTransition
		if next < len(expectedLane) {
			expected = expectedLane[next]
		}
		if next >= len(expectedLane) || !sameSimulationTransition(expected, transition) {
			actualOrdinal := expected.Ordinal
			if expected.Kind == "" {
				actualOrdinal = uint64(len(network.replay.Transitions))
			}
			transition.Ordinal = actualOrdinal
			replayErr, err := newSimulationReplayError(actualOrdinal, expected, transition)
			if err != nil {
				return err
			}
			network.divergence = replayErr
			return replayErr
		}
		network.replayNext[lane] = next + 1
	}
	transition.Ordinal = 0
	network.transitions = append(network.transitions, transition)
	return nil
}

func sameSimulationTransition(left, right simulationTransition) bool {
	left.Ordinal = 0
	right.Ordinal = 0
	return left == right
}

func canonicalizeSimulationTransitions(transitions []simulationTransition) {
	sort.SliceStable(transitions, func(left, right int) bool {
		return simulationTransitionLane(transitions[left]) < simulationTransitionLane(transitions[right])
	})
	for index := range transitions {
		transitions[index].Ordinal = uint64(index)
	}
}

func simulationTransitionLane(transition simulationTransition) string {
	switch transition.Kind {
	case "partition", "heal", "delay":
		return "topology"
	case "listen", "listener_close":
		return "listener\x00" + simulationEndpointKey(transition.Source)
	case "dial", "accept", "write", "deliver", "close":
		if transition.Connection != 0 {
			return fmt.Sprintf("connection\x00%020d", transition.Connection)
		}
		return "operation\x00" + transition.Kind + "\x00" + simulationEndpointKey(transition.Source) + "\x00" + simulationEndpointKey(transition.Destination)
	case "stop", "crash":
		return "lifecycle\x00" + simulationEndpointKey(transition.Source)
	default:
		return "unknown\x00" + transition.Kind
	}
}

func (network *simulationNetwork) firstMissingReplayTransitionLocked() simulationTransition {
	remaining := make(map[string]int, len(network.replayNext))
	for lane, consumed := range network.replayNext {
		remaining[lane] = consumed
	}
	for _, transition := range network.replay.Transitions {
		lane := simulationTransitionLane(transition)
		if remaining[lane] != 0 {
			remaining[lane]--
			continue
		}
		return transition
	}
	return simulationTransition{Ordinal: uint64(len(network.replay.Transitions))}
}

func newSimulationReplayError(ordinal uint64, expected, actual simulationTransition) (*simulationBridgeError, error) {
	expectedSHA256, err := simulationTransitionSHA256(expected)
	if err != nil {
		return nil, err
	}
	actualSHA256, err := simulationTransitionSHA256(actual)
	if err != nil {
		return nil, err
	}
	bridgeErr := &simulationBridgeError{
		Kind: "replay", Message: ErrReplayDiverged.Error(), Ordinal: ordinal,
		ExpectedSHA256: expectedSHA256, ActualSHA256: actualSHA256,
	}
	if expected.Kind != "" {
		bridgeErr.Expected = &expected
	}
	if actual.Kind != "" {
		bridgeErr.Actual = &actual
	}
	return bridgeErr, nil
}

func simulationTransitionSHA256(transition simulationTransition) (string, error) {
	encoded, err := encodeSimulationTransitionIdentity(transition)
	if err != nil {
		return "", fmt.Errorf("encode simulation network transition digest: %w", err)
	}
	digest := sha256.Sum256(append([]byte("gomad3-simulation-network-transition/v2\x00"), encoded...))
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func simulationPayloadSHA256(payload []byte) string {
	digest := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func (err *simulationBridgeError) Error() string {
	return err.Message
}

func (err *simulationBridgeError) GomadSimulationNetworkReplayDivergence() []byte {
	if err.Kind != "replay" {
		return nil
	}
	return encodeSimulationBridgeError(err)
}

func simulationLinkKey(from, to string) string {
	return from + "\x00" + to
}

func simulationEndpointKey(endpoint simulationEndpoint) string {
	return fmt.Sprintf("%s\x00%020d\x00%s\x00%05d", endpoint.Node, endpoint.Incarnation, endpoint.Address, endpoint.Port)
}

func sameSimulationIncarnation(left, right simulationEndpoint) bool {
	return left.Node == right.Node && left.Incarnation == right.Incarnation
}

func (network *simulationNetwork) signalLocked() {
	close(network.changed)
	network.changed = make(chan struct{})
}
