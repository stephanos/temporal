// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Code generated from the native Gomad network model; DO NOT EDIT.

package network

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"sync"
	"syscall"
	"time"
)

var (
	ErrAddressInUse      = errors.New("address already in use")
	ErrClosed            = errors.New("use of closed network connection")
	ErrConnectionRefused = errors.New("connection refused")
	ErrStallTimeout      = syscall.ETIMEDOUT
	ErrResourceExhausted = errors.New("network resources exhausted")
	ErrUnsupported       = errors.New("unsupported Gomad network operation")
	errNetworkWouldBlock = errors.New("Gomad network would block")
)

const (
	firstListenerPort    = 20000
	firstClientPort      = 40000
	maximumPort          = 65535
	maximumPendingConns  = 64
	maximumPendingChunks = 64
	maximumChunkBytes    = 64 << 10
)

type Address struct {
	IP   string
	Port int
}
type listenerImplementation interface {
	Accept() (*Conn, error)
	Close() error
	Address() Address
	SetDeadline(time.Time) error
}
type connImplementation interface {
	Read([]byte) (int, error)
	Write([]byte) (int, error)
	Close() error
	CloseRead() error
	CloseWrite() error
	LocalAddress() Address
	RemoteAddress() Address
	SetDeadline(time.Time) error
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
}
type Listener struct{ implementation listenerImplementation }
type Conn struct {
	implementation connImplementation
	readMu         sync.Mutex
	writeMu        sync.Mutex
}

func (listener *Listener) Accept() (*Conn, error) {
	return listener.implementation.Accept()
}
func (listener *Listener) Close() error {
	return listener.implementation.Close()
}
func (listener *Listener) Address() Address {
	return listener.implementation.Address()
}
func (listener *Listener) SetDeadline(deadline time.Time) error {
	return listener.implementation.SetDeadline(deadline)
}
func (connection *Conn) Read(destination []byte) (int, error) {
	connection.readMu.Lock()
	defer connection.readMu.Unlock()
	return connection.implementation.Read(destination)
}
func (connection *Conn) Write(source []byte) (int, error) {
	connection.writeMu.Lock()
	defer connection.writeMu.Unlock()
	return connection.implementation.Write(source)
}
func (connection *Conn) Close() error {
	return connection.implementation.Close()
}
func (connection *Conn) CloseRead() error {
	return connection.implementation.CloseRead()
}
func (connection *Conn) CloseWrite() error {
	return connection.implementation.CloseWrite()
}
func (connection *Conn) LocalAddress() Address {
	return connection.implementation.LocalAddress()
}
func (connection *Conn) RemoteAddress() Address {
	return connection.implementation.RemoteAddress()
}
func (connection *Conn) SetDeadline(deadline time.Time) error {
	return connection.implementation.SetDeadline(deadline)
}
func (connection *Conn) SetReadDeadline(deadline time.Time) error {
	return connection.implementation.SetReadDeadline(deadline)
}
func (connection *Conn) SetWriteDeadline(deadline time.Time) error {
	return connection.implementation.SetWriteDeadline(deadline)
}
func newConnStates() (*connState, *connState) {
	shared := &connShared{changed: make(chan struct{})}
	return &connState{shared: shared}, &connState{shared: shared}
}
func earliestDeadline(left, right time.Time) time.Time {
	if left.IsZero() || right.Before(left) {
		return right
	}
	return left
}
func (shared *connShared) signal() {
	close(shared.changed)
	shared.changed = make(chan struct{})
}

type pairedConn struct {
	local    Address
	remote   Address
	state    *connState
	peer     *connState
	network  *simulationNetwork
	owner    simulationEndpoint
	target   simulationEndpoint
	identity uint64
}
type connState struct {
	shared        *connShared
	incoming      []networkChunk
	reset         bool
	terminal      error
	pending       []byte
	retainedBytes uint64
	closeCalled   bool
	readClosed    bool
	writeClosed   bool
	readDeadline  time.Time
	writeDeadline time.Time
}
type connShared struct {
	sync.Mutex
	changed chan struct{}
}
type networkChunk struct {
	identity    uint64
	connection  uint64
	source      simulationEndpoint
	destination simulationEndpoint
	bytes       []byte
	ready       time.Time
	delayNanos  uint64
}

func (connection *pairedConn) validate() error {
	if connection.network != nil {
		return validateSimulationEndpoint(connection.network, connection.owner)
	}
	return nil
}
func (connection *pairedConn) nowTime() time.Time {
	return connection.network.nowTime()
}
func (connection *pairedConn) deadlineExpired(deadline time.Time) bool {
	return !deadline.IsZero() && !connection.nowTime().Before(deadline)
}
func (connection *pairedConn) terminalError() error {
	if connection.state.terminal != nil {
		return connection.state.terminal
	}
	if connection.peer.terminal != nil {
		return connection.peer.terminal
	}
	if connection.state.reset || connection.peer.reset {
		return ErrClosed
	}
	return nil
}
func (connection *pairedConn) lockState() {
	if connection.network != nil {
		connection.network.Lock()
	}
	connection.state.shared.Lock()
}
func (connection *pairedConn) unlockState() {
	connection.state.shared.Unlock()
	if connection.network != nil {
		connection.network.Unlock()
	}
}
func (connection *pairedConn) Read(destination []byte) (int, error) {
	if len(destination) == 0 {
		return 0, nil
	}
	if err := connection.validate(); err != nil {
		return 0, err
	}
	for {
		connection.lockState()
		length, err, ready, freed := connection.tryReadLocked(destination)
		if err != errNetworkWouldBlock {
			connection.unlockState()
			if freed {
			}
			return length, err
		}
		deadline := connection.state.readDeadline
		if !ready.IsZero() {
			changed := connection.state.shared.changed
			connection.unlockState()
			panic(waiting{Changed: changed, Deadline: earliestDeadline(deadline, ready)})
		}
		if connection.deadlineExpired(deadline) {
			connection.unlockState()
			return 0, os.ErrDeadlineExceeded
		}
		changed := connection.state.shared.changed
		connection.unlockState()
		panic(waiting{Changed: changed, Deadline: deadline})
	}
}
func (connection *pairedConn) Write(source []byte) (int, error) {
	if err := connection.validate(); err != nil {
		return 0, err
	}
	written := 0
	for len(source) != 0 {
		connection.lockState()
		length, err := connection.tryWriteLocked(source)
		if err != errNetworkWouldBlock {
			connection.unlockState()
			written += length
			if length != 0 {
			}
			if err != nil {
				return written, err
			}
			source = source[length:]
			continue
		}
		deadline := connection.state.writeDeadline
		if connection.deadlineExpired(deadline) {
			connection.unlockState()
			return written, os.ErrDeadlineExceeded
		}
		changed := connection.state.shared.changed
		connection.unlockState()
		panic(waiting{Changed: changed, Deadline: deadline, Accepted: written})
	}
	return written, nil
}
func (connection *pairedConn) Close() error {
	if err := connection.validate(); err != nil {
		return err
	}
	connection.lockState()
	if connection.state.closeCalled {
		connection.unlockState()
		return ErrClosed
	}
	if connection.network != nil {
		if err := connection.network.commitTransitionLocked(simulationTransition{Kind: "close", Source: connection.owner, Destination: connection.target, Connection: connection.identity, Outcome: "ok"}); err != nil {
			connection.unlockState()
			return err
		}
	}
	connection.state.closeCalled = true
	connection.shutdownLocked(true, true)
	connection.unlockState()
	return nil
}
func (connection *pairedConn) CloseRead() error {
	if err := connection.validate(); err != nil {
		return err
	}
	connection.lockState()
	if connection.state.readClosed {
		connection.unlockState()
		return ErrClosed
	}
	connection.shutdownLocked(true, false)
	connection.unlockState()
	return nil
}
func (connection *pairedConn) CloseWrite() error {
	if err := connection.validate(); err != nil {
		return err
	}
	connection.lockState()
	if connection.state.writeClosed {
		connection.unlockState()
		return ErrClosed
	}
	connection.shutdownLocked(false, true)
	connection.unlockState()
	return nil
}
func (connection *pairedConn) LocalAddress() Address {
	return connection.local
}
func (connection *pairedConn) RemoteAddress() Address {
	return connection.remote
}
func (connection *pairedConn) SetDeadline(deadline time.Time) error {
	if err := connection.validate(); err != nil {
		return err
	}
	connection.lockState()
	connection.state.readDeadline = deadline
	connection.state.writeDeadline = deadline
	connection.state.shared.signal()
	connection.unlockState()
	return nil
}
func (connection *pairedConn) SetReadDeadline(deadline time.Time) error {
	if err := connection.validate(); err != nil {
		return err
	}
	connection.lockState()
	connection.state.readDeadline = deadline
	connection.state.shared.signal()
	connection.unlockState()
	return nil
}
func (connection *pairedConn) SetWriteDeadline(deadline time.Time) error {
	if err := connection.validate(); err != nil {
		return err
	}
	connection.lockState()
	connection.state.writeDeadline = deadline
	connection.state.shared.signal()
	connection.unlockState()
	return nil
}
func (connection *pairedConn) tryReadLocked(destination []byte) (int, error, time.Time, bool) {
	if len(destination) == 0 {
		return 0, nil, time.Time{}, false
	}
	if err := connection.terminalError(); err != nil {
		return 0, err, time.Time{}, false
	}
	if connection.state.readClosed {
		return 0, ErrClosed, time.Time{}, false
	}
	if len(connection.state.pending) == 0 && len(connection.state.incoming) != 0 {
		chunk := connection.state.incoming[0]
		if connection.nowTime().Before(chunk.ready) {
			return 0, errNetworkWouldBlock, chunk.ready, false
		}
		if connection.network != nil {
			transition := simulationTransition{Kind: "deliver", Source: chunk.source, Destination: chunk.destination, Connection: chunk.connection, Delivery: chunk.identity, Bytes: uint64(len(chunk.bytes)), DelayNanos: chunk.delayNanos, Outcome: "ok", PayloadSHA256: simulationPayloadSHA256(chunk.bytes)}
			if err := connection.network.commitTransitionLocked(transition); err != nil {
				return 0, err, time.Time{}, false
			}
			delete(connection.network.deliveries, chunk.identity)
		}
		connection.state.pending = chunk.bytes
		connection.state.incoming[0] = networkChunk{}
		connection.state.incoming = connection.state.incoming[1:]
	}
	if len(connection.state.pending) != 0 {
		length := copy(destination, connection.state.pending)
		connection.state.pending = connection.state.pending[length:]
		connection.state.retainedBytes -= uint64(length)
		if connection.network != nil {
			connection.network.pendingBytes -= uint64(length)
		}
		connection.state.shared.signal()
		return length, nil, time.Time{}, true
	}
	if connection.peer.writeClosed {
		return 0, io.EOF, time.Time{}, false
	}
	return 0, errNetworkWouldBlock, time.Time{}, false
}
func (connection *pairedConn) tryWriteLocked(source []byte) (int, error) {
	if len(source) == 0 {
		return 0, nil
	}
	if err := connection.terminalError(); err != nil {
		return 0, err
	}
	if connection.state.writeClosed || connection.peer.readClosed {
		return 0, ErrClosed
	}
	capacity := uint64(4 << 20)
	if connection.network != nil {
		capacity = connection.network.limits.ConnectionBytes
	}
	if connection.peer.retainedBytes >= capacity {
		return 0, errNetworkWouldBlock
	}
	length := min(len(source), maximumChunkBytes, int(capacity-connection.peer.retainedBytes))
	chunk := networkChunk{bytes: append([]byte(nil), source[:length]...)}
	if connection.network != nil {
		link, ok := connection.network.links[simulationLinkKey(connection.owner.Node, connection.target.Node)]
		outcome := "ok"
		if !ok || !link.Enabled {
			outcome = "partition_drop"
		}
		transition := simulationTransition{Kind: "write", Source: connection.owner, Destination: connection.target, Connection: connection.identity, Bytes: uint64(length), DelayNanos: link.DelayNanos, Outcome: outcome, PayloadSHA256: simulationPayloadSHA256(source[:length])}
		if outcome == "ok" && (uint64(len(connection.network.deliveries)) >= connection.network.limits.Deliveries || uint64(length) > connection.network.limits.Bytes-connection.network.pendingBytes) {
			transition.Outcome = "capacity"
			if err := connection.network.commitTransitionLocked(transition); err != nil {
				return 0, err
			}
			return 0, ErrResourceExhausted
		}
		delivery := connection.network.nextDelivery + 1
		transition.Delivery = delivery
		if err := connection.network.commitTransitionLocked(transition); err != nil {
			return 0, err
		}
		connection.network.nextDelivery = delivery
		if outcome == "partition_drop" {
			return length, nil
		}
		chunk.identity = delivery
		chunk.connection = connection.identity
		chunk.source = connection.owner
		chunk.destination = connection.target
		chunk.ready = connection.nowTime().Add(time.Duration(link.DelayNanos))
		chunk.delayNanos = link.DelayNanos
		connection.network.deliveries[delivery] = simulationDelivery{identity: delivery, connection: connection.identity, source: connection.owner, destination: connection.target, bytes: uint64(length), delayNanos: link.DelayNanos}
		connection.network.pendingBytes += uint64(length)
	}
	connection.peer.incoming = append(connection.peer.incoming, chunk)
	connection.peer.retainedBytes += uint64(length)
	connection.state.shared.signal()
	return length, nil
}
func (connection *pairedConn) shutdownLocked(read, write bool) {
	if read {
		connection.state.readClosed = true
		if connection.network != nil {
			connection.network.dropStateDeliveriesLocked(connection.state)
		} else {
			connection.state.retainedBytes = 0
			connection.state.incoming = nil
			connection.state.pending = nil
		}
	}
	if write {
		connection.state.writeClosed = true
	}
	connection.state.shared.signal()
}

const maximumSimulationConfigBytes = 128 << 20

var ErrReplayDiverged = errors.New("simulation network replay diverged")

type simulationNetworkConfig struct {
	Nodes  []simulationNodeConfig `json:"nodes"`
	Links  []simulationLink       `json:"links"`
	Limits simulationLimits       `json:"limits"`
	Replay *simulationRecord      `json:"replay,omitempty"`
}
type simulationLimits struct {
	Listeners       uint64 `json:"listeners"`
	Connections     uint64 `json:"connections"`
	Deliveries      uint64 `json:"deliveries"`
	Bytes           uint64 `json:"bytes"`
	ConnectionBytes uint64 `json:"connection_bytes"`
	StallNanos      uint64 `json:"stall_nanos"`
	Transitions     uint64 `json:"transitions"`
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
	listeners       map[Address]*simulationListener
	listenerHistory []*simulationListener
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
	current         Domain
	now             time.Time
	live            map[string]Domain
}

func newSimulationNetwork(run uint64, config simulationNetworkConfig) (*simulationNetwork, error) {
	if run == 0 || config.Limits.Listeners == 0 || config.Limits.Connections == 0 || config.Limits.Deliveries == 0 || config.Limits.Bytes == 0 || config.Limits.Transitions == 0 {
		return nil, errors.New("simulation network limits are invalid")
	}
	if err := validateSimulationLimits(config.Limits); err != nil {
		return nil, err
	}
	if config.Replay != nil {
		for index, transition := range config.Replay.Transitions {
			if transition.Ordinal != uint64(index) || !validSimulationTransition(transition) || index != 0 && simulationTransitionLane(config.Replay.Transitions[index-1]) > simulationTransitionLane(transition) {
				return nil, errors.New("simulation network replay transition is invalid or unordered")
			}
		}
	}
	network := &simulationNetwork{run: run, limits: config.Limits, nodes: make(map[string]*simulationNode, len(config.Nodes)), addresses: make(map[string]*simulationNode, len(config.Nodes)), links: make(map[string]simulationLink, len(config.Links)), listeners: make(map[Address]*simulationListener), connections: make(map[uint64]*simulationConnection), deliveries: make(map[uint64]simulationDelivery), changed: make(chan struct{}), replay: config.Replay}
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
func (network *simulationNetwork) listen(networkName, host string, requestedPort int, domain Domain) (*Listener, error) {
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
	listener := &simulationListener{address: address, owner: owner, network: network, changed: make(chan struct{})}
	network.listeners[address] = listener
	network.listenerHistory = append(network.listenerHistory, listener)
	network.signalLocked()
	return &Listener{implementation: listener}, nil
}
func (network *simulationNetwork) dial(ctx context.Context, networkName, host string, port int, domain Domain) (*Conn, error) {
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
			panic(waiting{Changed: changed, Deadline: deadlineOf(ctx)})
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
			panic(waiting{Changed: changed, Deadline: deadlineOf(ctx)})
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
		client := &simulationConn{pairedConn: pairedConn{local: clientAddress, remote: serverAddress, state: clientState, peer: serverState, owner: clientEndpoint, target: serverEndpoint, network: network, identity: identity}}
		server := &simulationConn{pairedConn: pairedConn{local: serverAddress, remote: clientAddress, state: serverState, peer: clientState, owner: serverEndpoint, target: clientEndpoint, network: network, identity: identity}}
		network.connections[identity] = &simulationConnection{identity: identity, client: clientEndpoint, server: serverEndpoint, clientState: clientState, serverState: serverState}
		listener.pending = append(listener.pending, server)
		listener.signal()
		listener.mu.Unlock()
		network.signalLocked()
		network.Unlock()
		return &Conn{implementation: client}, nil
	}
}
func (network *simulationNetwork) accept(listener *simulationListener) (*simulationConn, error) {
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
		if network.deadlineExpired(deadline) {
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
		panic(waiting{Changed: changed, Deadline: deadline})
	}
}
func (network *simulationNetwork) closeListener(listener *simulationListener) error {
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
	return network.validateEndpoint(endpoint)
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
	transition := simulationTransition{Kind: kind, Source: simulationEndpoint{Node: from, Address: network.nodes[from].address}, Destination: simulationEndpoint{Node: to, Address: network.nodes[to].address}, Count: 1, DelayNanos: delayNanos, Outcome: "ok"}
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
			transitions = append(transitions, simulationTransition{Kind: kind, Source: simulationEndpoint{Node: leftNode, Address: network.nodes[leftNode].address}, Destination: simulationEndpoint{Node: rightNode, Address: network.nodes[rightNode].address}, Count: 2, Outcome: "ok"})
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
	transition := simulationTransition{Kind: kind, Source: simulationEndpoint{Node: left, Address: network.nodes[left].address}, Destination: simulationEndpoint{Node: right, Address: network.nodes[right].address}, Count: 2, DelayNanos: delayNanos, Outcome: "ok"}
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
func (network *simulationNetwork) revoke(domain Domain, graceful bool) error {
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
	}
	network.pendingBytes -= state.retainedBytes
	state.retainedBytes = 0
	state.pending = nil
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
		finishErr = &simulationBridgeError{Kind: "replay", Message: ErrReplayDiverged.Error(), Ordinal: uint64(len(network.transitions)), ExpectedSHA256: network.replay.Snapshot.Identity, ActualSHA256: snapshot.Identity}
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
	snapshot := simulationSnapshot{TransitionSHA256: "sha256:" + hex.EncodeToString(digest[:]), NextConnection: network.nextConnection, NextDelivery: network.nextDelivery}
	for _, node := range network.nodes {
		snapshot.Nodes = append(snapshot.Nodes, simulationNodeSnapshot{Node: node.node, Address: node.address, LastIncarnation: node.lastIncarnation, NextListenerPort: uint64(node.nextListenerPort), NextClientPort: uint64(node.nextClientPort)})
	}
	sort.Slice(snapshot.Nodes, func(left, right int) bool {
		return snapshot.Nodes[left].Node < snapshot.Nodes[right].Node
	})
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
		snapshot.Connections = append(snapshot.Connections, simulationConnectionSnapshot{Identity: connection.identity, Client: connection.client, Server: connection.server, Closed: connection.closed, Reset: connection.reset})
	}
	sort.Slice(snapshot.Connections, func(left, right int) bool {
		return snapshot.Connections[left].Identity < snapshot.Connections[right].Identity
	})
	for _, delivery := range network.deliveries {
		snapshot.Deliveries = append(snapshot.Deliveries, simulationDeliverySnapshot{Identity: delivery.identity, Connection: delivery.connection, Source: delivery.source, Destination: delivery.destination, Bytes: delivery.bytes, DelayNanos: delivery.delayNanos})
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
	if !validSimulationTransition(transition) {
		return errors.New("simulation network transition is invalid")
	}
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
	bridgeErr := &simulationBridgeError{Kind: "replay", Message: ErrReplayDiverged.Error(), Ordinal: ordinal, ExpectedSHA256: expectedSHA256, ActualSHA256: actualSHA256}
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

type simulationListener struct {
	address  Address
	owner    simulationEndpoint
	network  *simulationNetwork
	mu       sync.Mutex
	pending  []*simulationConn
	closed   bool
	deadline time.Time
	changed  chan struct{}
}
type simulationConn struct{ pairedConn }

func (listener *simulationListener) Accept() (*Conn, error) {
	connection, err := listener.network.accept(listener)
	if err != nil {
		return nil, err
	}
	return &Conn{implementation: connection}, nil
}
func (listener *simulationListener) Close() error {
	err := listener.network.closeListener(listener)
	return err
}
func (listener *simulationListener) Address() Address {
	return listener.address
}
func (listener *simulationListener) SetDeadline(deadline time.Time) error {
	if err := validateSimulationEndpoint(listener.network, listener.owner); err != nil {
		return err
	}
	listener.mu.Lock()
	listener.deadline = deadline
	listener.signal()
	listener.mu.Unlock()
	return nil
}
func (listener *simulationListener) signal() {
	close(listener.changed)
	listener.changed = make(chan struct{})
}

const (
	maximumSimulationWireBytes       = 134217728
	maximumSimulationWireStringBytes = 1048576
	simulationConfigMagic            = "GOMADNC2"
	simulationFinishMagic            = "GOMADNR1"
	simulationErrorMagic             = "GOMADNE1"
	maximumSimulationNodes           = 64
	maximumSimulationLinks           = 4096
	maximumSimulationListeners       = 4096
	maximumSimulationConnections     = 65536
	maximumSimulationNetworkBytes    = 1073741824
	maximumSimulationStallNanos      = 9223372036854775806
	maximumSimulationDeliveries      = 65536
	maximumSimulationTransitions     = 1048576
)

type simulationWireEncoder struct {
	data []byte
	err  error
}

func (encoder *simulationWireEncoder) uint64(value uint64) {
	if encoder.err != nil {
		return
	}
	encoder.data = appendSimulationUint64(encoder.data, value)
	if len(encoder.data) > maximumSimulationWireBytes {
		encoder.err = errors.New("simulation network wire value exceeds its byte limit")
	}
}
func (encoder *simulationWireEncoder) boolean(value bool) {
	if value {
		encoder.uint64(1)
		return
	}
	encoder.uint64(0)
}
func (encoder *simulationWireEncoder) string(value string) {
	if encoder.err != nil {
		return
	}
	if len(value) > maximumSimulationWireStringBytes {
		encoder.err = errors.New("simulation network wire string exceeds its byte limit")
		return
	}
	encoder.uint64(uint64(len(value)))
	encoder.data = append(encoder.data, value...)
	if len(encoder.data) > maximumSimulationWireBytes {
		encoder.err = errors.New("simulation network wire value exceeds its byte limit")
	}
}

type simulationWireDecoder struct {
	data   []byte
	offset uint64
	err    error
}

func newSimulationWireDecoder(data []byte, magic string) *simulationWireDecoder {
	decoder := &simulationWireDecoder{data: data}
	if len(data) < len(magic) || string(data[:len(magic)]) != magic {
		decoder.err = errors.New("simulation network wire header is invalid")
		return decoder
	}
	decoder.offset = uint64(len(magic))
	return decoder
}
func (decoder *simulationWireDecoder) uint64() uint64 {
	if decoder.err != nil {
		return 0
	}
	if uint64(len(decoder.data))-decoder.offset < 8 {
		decoder.err = errors.New("simulation network wire value is truncated")
		return 0
	}
	value := readSimulationUint64(decoder.data[decoder.offset : decoder.offset+8])
	decoder.offset += 8
	return value
}
func (decoder *simulationWireDecoder) boolean() bool {
	value := decoder.uint64()
	if decoder.err == nil && value > 1 {
		decoder.err = errors.New("simulation network wire boolean is invalid")
	}
	return value == 1
}
func (decoder *simulationWireDecoder) string() string {
	length := decoder.uint64()
	if decoder.err != nil {
		return ""
	}
	if length > maximumSimulationWireStringBytes || length > uint64(len(decoder.data))-decoder.offset {
		decoder.err = errors.New("simulation network wire string is invalid")
		return ""
	}
	value := string(decoder.data[decoder.offset : decoder.offset+length])
	decoder.offset += length
	return value
}
func (decoder *simulationWireDecoder) count(maximum uint64) uint64 {
	value := decoder.uint64()
	if decoder.err == nil && value > maximum {
		decoder.err = errors.New("simulation network wire count exceeds its limit")
	}
	if decoder.err != nil {
		return 0
	}
	return value
}
func (decoder *simulationWireDecoder) finish() error {
	if decoder.err != nil {
		return decoder.err
	}
	if decoder.offset != uint64(len(decoder.data)) {
		return errors.New("simulation network wire value contains trailing data")
	}
	return nil
}
func decodeSimulationConfig(encoded []byte) (simulationNetworkConfig, error) {
	if len(encoded) == 0 || len(encoded) > maximumSimulationWireBytes {
		return simulationNetworkConfig{}, errors.New("simulation network configuration is invalid")
	}
	decoder := newSimulationWireDecoder(encoded, simulationConfigMagic)
	config := simulationNetworkConfig{}
	for range decoder.count(maximumSimulationNodes) {
		if decoder.err != nil {
			break
		}
		config.Nodes = append(config.Nodes, simulationNodeConfig{Node: decoder.string(), Address: decoder.string()})
	}
	for range decoder.count(maximumSimulationLinks) {
		if decoder.err != nil {
			break
		}
		config.Links = append(config.Links, simulationLink{From: decoder.string(), To: decoder.string(), Enabled: decoder.boolean(), DelayNanos: decoder.uint64()})
	}
	config.Limits = simulationLimits{Listeners: decoder.uint64(), Connections: decoder.uint64(), Deliveries: decoder.uint64(), Bytes: decoder.uint64(), ConnectionBytes: decoder.uint64(), StallNanos: decoder.uint64(), Transitions: decoder.uint64()}
	if decoder.boolean() {
		replay := decodeSimulationRecord(decoder)
		config.Replay = &replay
	}
	if err := decoder.finish(); err != nil {
		return simulationNetworkConfig{}, fmt.Errorf("decode simulation network configuration: %w", err)
	}
	if err := validateSimulationLimits(config.Limits); err != nil {
		return simulationNetworkConfig{}, err
	}
	return config, nil
}
func encodeSimulationFinishResponse(record simulationRecord, bridge *simulationBridgeError) ([]byte, error) {
	encoder := &simulationWireEncoder{data: []byte(simulationFinishMagic)}
	encodeSimulationRecord(encoder, record)
	encoder.boolean(bridge != nil)
	if bridge != nil {
		encodeSimulationError(encoder, bridge)
	}
	if encoder.err != nil {
		return nil, encoder.err
	}
	return encoder.data, nil
}
func encodeSimulationBridgeError(source error) []byte {
	var bridge *simulationBridgeError
	if !errors.As(source, &bridge) {
		bridge = &simulationBridgeError{Kind: "runtime", Message: source.Error()}
	}
	encoder := &simulationWireEncoder{data: []byte(simulationErrorMagic)}
	encodeSimulationError(encoder, bridge)
	if encoder.err == nil {
		return encoder.data
	}
	fallback := &simulationWireEncoder{data: []byte(simulationErrorMagic)}
	encodeSimulationError(fallback, &simulationBridgeError{Kind: "runtime", Message: "encode simulation network error"})
	return fallback.data
}
func validateSimulationLimits(limits simulationLimits) error {
	if limits.ConnectionBytes == 0 || limits.ConnectionBytes > maximumSimulationNetworkBytes {
		return errors.New("simulation network connection byte limit is invalid")
	}
	if limits.StallNanos == 0 || limits.StallNanos > maximumSimulationStallNanos {
		return errors.New("simulation network stall duration limit is invalid")
	}
	return nil
}
func validSimulationTransition(transition simulationTransition) bool {
	switch transition.Kind {
	case "listen", "dial", "accept", "write", "deliver", "close", "listener_close", "stop", "crash", "reset":
		if !validSimulationWireEndpoint(transition.Source, true) {
			return false
		}
	case "partition", "heal", "delay", "disconnect", "reconnect", "directional_delay":
		if !validSimulationWireEndpoint(transition.Source, false) || !validSimulationWireEndpoint(transition.Destination, false) || transition.Source.Node == transition.Destination.Node {
			return false
		}
	default:
		return false
	}
	switch transition.Outcome {
	case "ok", "closed", "refused", "deadline", "partition_drop", "stale_drop", "reset", "capacity", "unsupported", "held", "timeout":
	default:
		return false
	}
	if transition.PayloadSHA256 != "" && !validSimulationWirePayloadSHA256(transition.PayloadSHA256) {
		return false
	}
	if transition.Kind == "reset" || transition.Outcome == "held" || transition.Outcome == "timeout" && transition.Kind == "write" {
		if transition.Kind != "reset" && transition.Kind != "write" || transition.Connection == 0 || !validSimulationWireEndpoint(transition.Destination, true) || transition.Source.Node == transition.Destination.Node || transition.Source.Port == 0 || transition.Source.Port > 65535 || transition.Destination.Port == 0 || transition.Destination.Port > 65535 {
			return false
		}
		if transition.Kind == "reset" && (transition.Outcome != "reset" && transition.Outcome != "timeout" || transition.Delivery != 0 || transition.PayloadSHA256 != "") {
			return false
		}
		if transition.Outcome == "held" && transition.Bytes == 0 {
			return false
		}
	}
	if transition.Outcome == "timeout" && transition.Kind != "write" && transition.Kind != "reset" {
		if transition.Kind != "dial" || transition.Connection != 0 || transition.Delivery != 0 || transition.Source.Port > 65535 || !validSimulationWireEndpoint(transition.Destination, false) || transition.Source.Node == transition.Destination.Node || transition.Destination.Port == 0 || transition.Destination.Port > 65535 {
			return false
		}
	}
	return true
}
func validSimulationWireEndpoint(endpoint simulationEndpoint, requireIncarnation bool) bool {
	if endpoint.Node == "" || endpoint.Address == "" {
		return false
	}
	return !requireIncarnation || endpoint.Incarnation != 0
}
func validSimulationWirePayloadSHA256(value string) bool {
	if len(value) != 71 || value[:7] != "sha256:" {
		return false
	}
	_, err := hex.DecodeString(value[7:])
	return err == nil
}
func simulationTransitionLane(transition simulationTransition) string {
	switch transition.Kind {
	case "partition", "heal", "delay", "disconnect", "reconnect", "directional_delay":
		return "topology"
	case "listen", "listener_close":
		return "listener\x00" + simulationEndpointKey(transition.Source)
	case "dial", "accept", "write", "deliver", "close", "reset":
		if transition.Connection != 0 {
			return fmt.Sprintf("connection\x00%020d", transition.Connection)
		}
		return "operation\x00" + string(transition.Kind) + "\x00" + simulationEndpointKey(transition.Source) + "\x00" + simulationEndpointKey(transition.Destination)
	case "stop", "crash":
		return "lifecycle\x00" + simulationEndpointKey(transition.Source)
	default:
		return "unknown\x00" + string(transition.Kind)
	}
}
func encodeSimulationRecord(encoder *simulationWireEncoder, record simulationRecord) {
	encoder.uint64(uint64(len(record.Transitions)))
	for _, transition := range record.Transitions {
		encodeSimulationTransition(encoder, transition)
	}
	encodeSimulationSnapshot(encoder, record.Snapshot)
}
func decodeSimulationRecord(decoder *simulationWireDecoder) simulationRecord {
	record := simulationRecord{}
	for range decoder.count(maximumSimulationTransitions) {
		if decoder.err != nil {
			break
		}
		record.Transitions = append(record.Transitions, decodeSimulationTransition(decoder))
	}
	record.Snapshot = decodeSimulationSnapshot(decoder)
	return record
}
func encodeSimulationTransition(encoder *simulationWireEncoder, transition simulationTransition) {
	encoder.uint64(transition.Ordinal)
	encoder.string(string(transition.Kind))
	encodeSimulationEndpoint(encoder, transition.Source)
	encodeSimulationEndpoint(encoder, transition.Destination)
	encoder.uint64(transition.Connection)
	encoder.uint64(transition.Delivery)
	encoder.uint64(transition.Bytes)
	encoder.uint64(transition.Count)
	encoder.uint64(transition.DelayNanos)
	encoder.string(string(transition.Outcome))
	encoder.string(transition.PayloadSHA256)
}
func decodeSimulationTransition(decoder *simulationWireDecoder) simulationTransition {
	transition := simulationTransition{Ordinal: decoder.uint64(), Kind: string(decoder.string())}
	transition.Source = decodeSimulationEndpoint(decoder)
	transition.Destination = decodeSimulationEndpoint(decoder)
	transition.Connection = decoder.uint64()
	transition.Delivery = decoder.uint64()
	transition.Bytes = decoder.uint64()
	transition.Count = decoder.uint64()
	transition.DelayNanos = decoder.uint64()
	transition.Outcome = string(decoder.string())
	transition.PayloadSHA256 = decoder.string()
	return transition
}
func encodeSimulationEndpoint(encoder *simulationWireEncoder, endpoint simulationEndpoint) {
	encoder.string(string(endpoint.Node))
	encoder.uint64(endpoint.Incarnation)
	encoder.string(endpoint.Address)
	encoder.uint64(endpoint.Port)
}
func decodeSimulationEndpoint(decoder *simulationWireDecoder) simulationEndpoint {
	return simulationEndpoint{Node: string(decoder.string()), Incarnation: decoder.uint64(), Address: decoder.string(), Port: decoder.uint64()}
}
func encodeSimulationSnapshot(encoder *simulationWireEncoder, snapshot simulationSnapshot) {
	encoder.uint64(uint64(len(snapshot.Nodes)))
	for _, node := range snapshot.Nodes {
		encoder.string(string(node.Node))
		encoder.string(node.Address)
		encoder.uint64(node.LastIncarnation)
		encoder.uint64(node.NextListenerPort)
		encoder.uint64(node.NextClientPort)
	}
	encoder.uint64(uint64(len(snapshot.Links)))
	for _, link := range snapshot.Links {
		encoder.string(string(link.From))
		encoder.string(string(link.To))
		encoder.boolean(link.Enabled)
		encoder.uint64(link.DelayNanos)
	}
	encoder.uint64(uint64(len(snapshot.Listeners)))
	for _, listener := range snapshot.Listeners {
		encodeSimulationEndpoint(encoder, listener.Endpoint)
		encoder.boolean(listener.Closed)
	}
	encoder.uint64(uint64(len(snapshot.Connections)))
	for _, connection := range snapshot.Connections {
		encoder.uint64(connection.Identity)
		encodeSimulationEndpoint(encoder, connection.Client)
		encodeSimulationEndpoint(encoder, connection.Server)
		encoder.boolean(connection.Closed)
		encoder.boolean(connection.Reset)
	}
	encoder.uint64(uint64(len(snapshot.Deliveries)))
	for _, delivery := range snapshot.Deliveries {
		encoder.uint64(delivery.Identity)
		encoder.uint64(delivery.Connection)
		encodeSimulationEndpoint(encoder, delivery.Source)
		encodeSimulationEndpoint(encoder, delivery.Destination)
		encoder.uint64(delivery.Bytes)
		encoder.uint64(delivery.DelayNanos)
	}
	encoder.uint64(snapshot.NextConnection)
	encoder.uint64(snapshot.NextDelivery)
	encoder.string(snapshot.TransitionSHA256)
	encoder.string(snapshot.Identity)
}
func decodeSimulationSnapshot(decoder *simulationWireDecoder) simulationSnapshot {
	snapshot := simulationSnapshot{}
	for range decoder.count(maximumSimulationNodes) {
		if decoder.err != nil {
			break
		}
		snapshot.Nodes = append(snapshot.Nodes, simulationNodeSnapshot{Node: string(decoder.string()), Address: decoder.string(), LastIncarnation: decoder.uint64(), NextListenerPort: decoder.uint64(), NextClientPort: decoder.uint64()})
	}
	for range decoder.count(maximumSimulationLinks) {
		if decoder.err != nil {
			break
		}
		snapshot.Links = append(snapshot.Links, simulationLink{From: string(decoder.string()), To: string(decoder.string()), Enabled: decoder.boolean(), DelayNanos: decoder.uint64()})
	}
	for range decoder.count(maximumSimulationListeners) {
		if decoder.err != nil {
			break
		}
		snapshot.Listeners = append(snapshot.Listeners, simulationListenerSnapshot{Endpoint: decodeSimulationEndpoint(decoder), Closed: decoder.boolean()})
	}
	for range decoder.count(maximumSimulationConnections) {
		if decoder.err != nil {
			break
		}
		snapshot.Connections = append(snapshot.Connections, simulationConnectionSnapshot{Identity: decoder.uint64(), Client: decodeSimulationEndpoint(decoder), Server: decodeSimulationEndpoint(decoder), Closed: decoder.boolean(), Reset: decoder.boolean()})
	}
	for range decoder.count(maximumSimulationDeliveries) {
		if decoder.err != nil {
			break
		}
		snapshot.Deliveries = append(snapshot.Deliveries, simulationDeliverySnapshot{Identity: decoder.uint64(), Connection: decoder.uint64(), Source: decodeSimulationEndpoint(decoder), Destination: decodeSimulationEndpoint(decoder), Bytes: decoder.uint64(), DelayNanos: decoder.uint64()})
	}
	snapshot.NextConnection = decoder.uint64()
	snapshot.NextDelivery = decoder.uint64()
	snapshot.TransitionSHA256 = decoder.string()
	snapshot.Identity = decoder.string()
	return snapshot
}
func encodeSimulationError(encoder *simulationWireEncoder, bridge *simulationBridgeError) {
	encoder.string(bridge.Kind)
	encoder.string(bridge.Message)
	encoder.uint64(bridge.Ordinal)
	encoder.string(bridge.ExpectedSHA256)
	encoder.string(bridge.ActualSHA256)
	encoder.boolean(bridge.Expected != nil)
	if bridge.Expected != nil {
		encodeSimulationTransition(encoder, *bridge.Expected)
	}
	encoder.boolean(bridge.Actual != nil)
	if bridge.Actual != nil {
		encodeSimulationTransition(encoder, *bridge.Actual)
	}
}
func encodeSimulationTransitions(transitions []simulationTransition) ([]byte, error) {
	encoder := &simulationWireEncoder{}
	encoder.uint64(uint64(len(transitions)))
	for _, transition := range transitions {
		encodeSimulationTransition(encoder, transition)
	}
	return encoder.data, encoder.err
}
func encodeSimulationSnapshotIdentity(snapshot simulationSnapshot) ([]byte, error) {
	encoder := &simulationWireEncoder{}
	encodeSimulationSnapshot(encoder, snapshot)
	return encoder.data, encoder.err
}
func encodeSimulationTransitionIdentity(transition simulationTransition) ([]byte, error) {
	encoder := &simulationWireEncoder{}
	encodeSimulationTransition(encoder, transition)
	return encoder.data, encoder.err
}
func appendSimulationUint64(destination []byte, value uint64) []byte {
	return append(destination, byte(value), byte(value>>8), byte(value>>16), byte(value>>24), byte(value>>32), byte(value>>40), byte(value>>48), byte(value>>56))
}
func readSimulationUint64(source []byte) uint64 {
	return uint64(source[0]) | uint64(source[1])<<8 | uint64(source[2])<<16 | uint64(source[3])<<24 | uint64(source[4])<<32 | uint64(source[5])<<40 | uint64(source[6])<<48 | uint64(source[7])<<56
}

func (network *simulationNetwork) resetConnection(identity uint64) error {
	network.Lock()
	defer network.Unlock()
	if network.finished {
		return syscall.ESTALE
	}
	found := false
	for _, connection := range network.connections {
		if connection.identity != identity {
			continue
		}
		found = true
		connection.clientState.shared.Lock()
		network.dropStateDeliveriesLocked(connection.clientState)
		network.dropStateDeliveriesLocked(connection.serverState)
		connection.closed = true
		connection.reset = true
		connection.clientState.readClosed = true
		connection.clientState.writeClosed = true
		connection.serverState.readClosed = true
		connection.serverState.writeClosed = true
		connection.clientState.reset = true
		connection.serverState.reset = true
		connection.clientState.shared.signal()
		connection.clientState.shared.Unlock()
	}
	if !found {
		return syscall.ESTALE
	}
	network.signalLocked()
	return nil
}
