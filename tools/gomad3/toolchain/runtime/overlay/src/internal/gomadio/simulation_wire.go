// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gomadio

import (
	"errors"
	"fmt"
)

const (
	maximumSimulationWireStringBytes = 1 << 20
	maximumSimulationNodes           = 64
	maximumSimulationLinks           = 4096
	maximumSimulationListeners       = 4096
	maximumSimulationConnections     = 65536
	maximumSimulationDeliveries      = 65536
	maximumSimulationTransitions     = 1 << 20
)

const (
	simulationConfigMagic = "GOMADNC1"
	simulationFinishMagic = "GOMADNR1"
	simulationErrorMagic  = "GOMADNE1"
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
	if len(encoder.data) > maximumSimulationConfigBytes {
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
	if len(encoder.data) > maximumSimulationConfigBytes {
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
	if len(encoded) == 0 || len(encoded) > maximumSimulationConfigBytes {
		return simulationNetworkConfig{}, errors.New("simulation network configuration is invalid")
	}
	decoder := newSimulationWireDecoder(encoded, simulationConfigMagic)
	config := simulationNetworkConfig{}
	for range decoder.count(maximumSimulationNodes) {
		config.Nodes = append(config.Nodes, simulationNodeConfig{Node: decoder.string(), Address: decoder.string()})
	}
	for range decoder.count(maximumSimulationLinks) {
		config.Links = append(config.Links, simulationLink{
			From: decoder.string(), To: decoder.string(), Enabled: decoder.boolean(), DelayNanos: decoder.uint64(),
		})
	}
	config.Limits = simulationLimits{
		Listeners: decoder.uint64(), Connections: decoder.uint64(), Deliveries: decoder.uint64(), Bytes: decoder.uint64(), Transitions: decoder.uint64(),
	}
	if decoder.boolean() {
		replay := decodeSimulationRecord(decoder)
		config.Replay = &replay
	}
	if err := decoder.finish(); err != nil {
		return simulationNetworkConfig{}, fmt.Errorf("decode simulation network configuration: %w", err)
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
		record.Transitions = append(record.Transitions, decodeSimulationTransition(decoder))
	}
	record.Snapshot = decodeSimulationSnapshot(decoder)
	return record
}

func encodeSimulationTransition(encoder *simulationWireEncoder, transition simulationTransition) {
	encoder.uint64(transition.Ordinal)
	encoder.string(transition.Kind)
	encodeSimulationEndpoint(encoder, transition.Source)
	encodeSimulationEndpoint(encoder, transition.Destination)
	encoder.uint64(transition.Connection)
	encoder.uint64(transition.Delivery)
	encoder.uint64(transition.Bytes)
	encoder.uint64(transition.Count)
	encoder.uint64(transition.DelayNanos)
	encoder.string(transition.Outcome)
	encoder.string(transition.PayloadSHA256)
}

func decodeSimulationTransition(decoder *simulationWireDecoder) simulationTransition {
	transition := simulationTransition{Ordinal: decoder.uint64(), Kind: decoder.string()}
	transition.Source = decodeSimulationEndpoint(decoder)
	transition.Destination = decodeSimulationEndpoint(decoder)
	transition.Connection = decoder.uint64()
	transition.Delivery = decoder.uint64()
	transition.Bytes = decoder.uint64()
	transition.Count = decoder.uint64()
	transition.DelayNanos = decoder.uint64()
	transition.Outcome = decoder.string()
	transition.PayloadSHA256 = decoder.string()
	return transition
}

func encodeSimulationEndpoint(encoder *simulationWireEncoder, endpoint simulationEndpoint) {
	encoder.string(endpoint.Node)
	encoder.uint64(endpoint.Incarnation)
	encoder.string(endpoint.Address)
	encoder.uint64(endpoint.Port)
}

func decodeSimulationEndpoint(decoder *simulationWireDecoder) simulationEndpoint {
	return simulationEndpoint{Node: decoder.string(), Incarnation: decoder.uint64(), Address: decoder.string(), Port: decoder.uint64()}
}

func encodeSimulationSnapshot(encoder *simulationWireEncoder, snapshot simulationSnapshot) {
	encoder.uint64(uint64(len(snapshot.Nodes)))
	for _, node := range snapshot.Nodes {
		encoder.string(node.Node)
		encoder.string(node.Address)
		encoder.uint64(node.LastIncarnation)
		encoder.uint64(node.NextListenerPort)
		encoder.uint64(node.NextClientPort)
	}
	encoder.uint64(uint64(len(snapshot.Links)))
	for _, link := range snapshot.Links {
		encoder.string(link.From)
		encoder.string(link.To)
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
		snapshot.Nodes = append(snapshot.Nodes, simulationNodeSnapshot{
			Node: decoder.string(), Address: decoder.string(), LastIncarnation: decoder.uint64(),
			NextListenerPort: decoder.uint64(), NextClientPort: decoder.uint64(),
		})
	}
	for range decoder.count(maximumSimulationLinks) {
		snapshot.Links = append(snapshot.Links, simulationLink{
			From: decoder.string(), To: decoder.string(), Enabled: decoder.boolean(), DelayNanos: decoder.uint64(),
		})
	}
	for range decoder.count(maximumSimulationListeners) {
		snapshot.Listeners = append(snapshot.Listeners, simulationListenerSnapshot{Endpoint: decodeSimulationEndpoint(decoder), Closed: decoder.boolean()})
	}
	for range decoder.count(maximumSimulationConnections) {
		snapshot.Connections = append(snapshot.Connections, simulationConnectionSnapshot{
			Identity: decoder.uint64(), Client: decodeSimulationEndpoint(decoder), Server: decodeSimulationEndpoint(decoder),
			Closed: decoder.boolean(), Reset: decoder.boolean(),
		})
	}
	for range decoder.count(maximumSimulationDeliveries) {
		snapshot.Deliveries = append(snapshot.Deliveries, simulationDeliverySnapshot{
			Identity: decoder.uint64(), Connection: decoder.uint64(), Source: decodeSimulationEndpoint(decoder), Destination: decodeSimulationEndpoint(decoder),
			Bytes: decoder.uint64(), DelayNanos: decoder.uint64(),
		})
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

func readSimulationUint64(source []byte) uint64 {
	return uint64(source[0]) |
		uint64(source[1])<<8 |
		uint64(source[2])<<16 |
		uint64(source[3])<<24 |
		uint64(source[4])<<32 |
		uint64(source[5])<<40 |
		uint64(source[6])<<48 |
		uint64(source[7])<<56
}

func appendSimulationUint64(destination []byte, value uint64) []byte {
	return append(destination,
		byte(value), byte(value>>8), byte(value>>16), byte(value>>24),
		byte(value>>32), byte(value>>40), byte(value>>48), byte(value>>56),
	)
}
