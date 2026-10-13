// Code generated from simulation/schema/networkwire.go.tmpl; DO NOT EDIT.

package gomad3sim

import (
	"encoding/hex"
	"errors"
	"fmt"
)

const (
	maximumRuntimeNetworkWireBytes        = 134217728
	maximumRuntimeNetworkWireStringBytes  = 1048576
	runtimeNetworkConfigMagic             = "GOMADNC2"
	runtimeNetworkFinishMagic             = "GOMADNR1"
	runtimeNetworkErrorMagic              = "GOMADNE1"
	maximumRuntimeNetworkWireNodes        = 64
	maximumRuntimeNetworkWireLinks        = 4096
	maximumRuntimeNetworkWireListeners    = 4096
	maximumRuntimeNetworkWireConnections  = 65536
	maximumRuntimeNetworkWireNetworkBytes = 1073741824
	maximumRuntimeNetworkWireStallNanos   = 9223372036854775806
	maximumRuntimeNetworkWireDeliveries   = 65536
	maximumRuntimeNetworkWireTransitions  = 1048576
)

type runtimeNetworkWireEncoder struct {
	data []byte
	err  error
}

func (encoder *runtimeNetworkWireEncoder) uint64(value uint64) {
	if encoder.err != nil {
		return
	}
	encoder.data = appendRuntimeNetworkUint64(encoder.data, value)
	if len(encoder.data) > maximumRuntimeNetworkWireBytes {
		encoder.err = errors.New("simulation runtime network wire value exceeds its byte limit")
	}
}

func (encoder *runtimeNetworkWireEncoder) boolean(value bool) {
	if value {
		encoder.uint64(1)
		return
	}
	encoder.uint64(0)
}

func (encoder *runtimeNetworkWireEncoder) string(value string) {
	if encoder.err != nil {
		return
	}
	if len(value) > maximumRuntimeNetworkWireStringBytes {
		encoder.err = errors.New("simulation runtime network wire string exceeds its byte limit")
		return
	}
	encoder.uint64(uint64(len(value)))
	encoder.data = append(encoder.data, value...)
	if len(encoder.data) > maximumRuntimeNetworkWireBytes {
		encoder.err = errors.New("simulation runtime network wire value exceeds its byte limit")
	}
}

type runtimeNetworkWireDecoder struct {
	data   []byte
	offset uint64
	err    error
}

func newRuntimeNetworkWireDecoder(data []byte, magic string) *runtimeNetworkWireDecoder {
	decoder := &runtimeNetworkWireDecoder{data: data}
	if len(data) < len(magic) || string(data[:len(magic)]) != magic {
		decoder.err = errors.New("simulation runtime network wire header is invalid")
		return decoder
	}
	decoder.offset = uint64(len(magic))
	return decoder
}

func (decoder *runtimeNetworkWireDecoder) uint64() uint64 {
	if decoder.err != nil {
		return 0
	}
	if uint64(len(decoder.data))-decoder.offset < 8 {
		decoder.err = errors.New("simulation runtime network wire value is truncated")
		return 0
	}
	value := readRuntimeNetworkUint64(decoder.data[decoder.offset : decoder.offset+8])
	decoder.offset += 8
	return value
}

func (decoder *runtimeNetworkWireDecoder) boolean() bool {
	value := decoder.uint64()
	if decoder.err == nil && value > 1 {
		decoder.err = errors.New("simulation runtime network wire boolean is invalid")
	}
	return value == 1
}

func (decoder *runtimeNetworkWireDecoder) string() string {
	length := decoder.uint64()
	if decoder.err != nil {
		return ""
	}
	if length > maximumRuntimeNetworkWireStringBytes || length > uint64(len(decoder.data))-decoder.offset {
		decoder.err = errors.New("simulation runtime network wire string is invalid")
		return ""
	}
	value := string(decoder.data[decoder.offset : decoder.offset+length])
	decoder.offset += length
	return value
}

func (decoder *runtimeNetworkWireDecoder) count(maximum uint64) uint64 {
	value := decoder.uint64()
	if decoder.err == nil && value > maximum {
		decoder.err = errors.New("simulation runtime network wire count exceeds its limit")
	}
	if decoder.err != nil {
		return 0
	}
	return value
}

func (decoder *runtimeNetworkWireDecoder) finish() error {
	if decoder.err != nil {
		return decoder.err
	}
	if decoder.offset != uint64(len(decoder.data)) {
		return errors.New("simulation runtime network wire value contains trailing data")
	}
	return nil
}

func encodeRuntimeNetworkConfig(spec Spec) ([]byte, error) {
	if err := validateRuntimeNetworkLimits(spec.Limits); err != nil {
		return nil, err
	}
	encoder := &runtimeNetworkWireEncoder{data: []byte(runtimeNetworkConfigMagic)}
	encoder.uint64(uint64(len(spec.Nodes)))
	for _, node := range spec.Nodes {
		encoder.string(string(node.ID))
		encoder.string(node.Address)
	}
	encoder.uint64(uint64(len(spec.Links)))
	for _, link := range spec.Links {
		encoder.string(string(link.From))
		encoder.string(string(link.To))
		encoder.boolean(link.Enabled)
		encoder.uint64(link.DelayNanos)
	}
	encoder.uint64(spec.Limits.NetworkListeners)
	encoder.uint64(spec.Limits.NetworkConnections)
	encoder.uint64(spec.Limits.NetworkDeliveries)
	encoder.uint64(spec.Limits.NetworkBytes)
	encoder.uint64(spec.Limits.NetworkConnectionBytes)
	encoder.uint64(spec.Limits.NetworkStallNanos)
	encoder.uint64(spec.Limits.NetworkTransitions)
	encoder.boolean(spec.Replay != nil)
	if spec.Replay != nil {
		encodeRuntimeNetworkRecord(encoder, spec.Replay.Network)
	}
	if encoder.err != nil {
		return nil, encoder.err
	}
	return encoder.data, nil
}

func decodeRuntimeNetworkFinish(encoded []byte) (NetworkRecord, *runtimeNetworkError, error) {
	if len(encoded) == 0 || len(encoded) > maximumRuntimeNetworkWireBytes {
		return NetworkRecord{}, nil, errors.New("simulation runtime network record is invalid")
	}
	decoder := newRuntimeNetworkWireDecoder(encoded, runtimeNetworkFinishMagic)
	record := decodeRuntimeNetworkRecord(decoder)
	var runtimeErr *runtimeNetworkError
	if decoder.boolean() {
		value := decodeRuntimeNetworkErrorBody(decoder)
		runtimeErr = &value
	}
	if err := decoder.finish(); err != nil {
		return NetworkRecord{}, nil, fmt.Errorf("decode simulation runtime network record: %w", err)
	}
	return record, runtimeErr, nil
}

func decodeRuntimeNetworkErrorWire(encoded []byte) (runtimeNetworkError, error) {
	if len(encoded) == 0 || len(encoded) > maximumRuntimeNetworkWireBytes {
		return runtimeNetworkError{}, errors.New("simulation runtime network error is invalid")
	}
	decoder := newRuntimeNetworkWireDecoder(encoded, runtimeNetworkErrorMagic)
	runtimeErr := decodeRuntimeNetworkErrorBody(decoder)
	if err := decoder.finish(); err != nil {
		return runtimeNetworkError{}, err
	}
	if runtimeErr.Message == "" {
		return runtimeNetworkError{}, errors.New("simulation runtime network error message is empty")
	}
	return runtimeErr, nil
}

func validateRuntimeNetworkLimits(limits Limits) error {
	if limits.NetworkConnectionBytes == 0 || limits.NetworkConnectionBytes > maximumRuntimeNetworkWireNetworkBytes {
		return errors.New("simulation runtime network connection byte limit is invalid")
	}
	if limits.NetworkStallNanos == 0 || limits.NetworkStallNanos > maximumRuntimeNetworkWireStallNanos {
		return errors.New("simulation runtime network stall duration limit is invalid")
	}
	return nil
}

func validNetworkTransition(transition NetworkTransition) bool {
	switch transition.Kind {
	case "listen", "dial", "accept", "write", "deliver", "close", "listener_close", "stop", "crash", "reset":
		if !validNetworkEndpoint(transition.Source, true) {
			return false
		}
	case "partition", "heal", "delay", "disconnect", "reconnect", "directional_delay":
		if !validNetworkEndpoint(transition.Source, false) || !validNetworkEndpoint(transition.Destination, false) || transition.Source.Node == transition.Destination.Node {
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
	if transition.PayloadSHA256 != "" && !validNetworkPayloadSHA256(transition.PayloadSHA256) {
		return false
	}
	if transition.Kind == "reset" || transition.Outcome == "held" || transition.Outcome == "timeout" && transition.Kind == "write" {
		if transition.Kind != "reset" && transition.Kind != "write" ||
			transition.Connection == 0 || !validNetworkEndpoint(transition.Destination, true) ||
			transition.Source.Node == transition.Destination.Node || transition.Source.Port == 0 || transition.Source.Port > 65535 || transition.Destination.Port == 0 || transition.Destination.Port > 65535 {
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
		if transition.Kind != "dial" || transition.Connection != 0 || transition.Delivery != 0 ||
			!validNetworkEndpoint(transition.Destination, false) || transition.Source.Node == transition.Destination.Node || transition.Destination.Port == 0 || transition.Destination.Port > 65535 {
			return false
		}
	}
	return true
}

func validNetworkEndpoint(endpoint NetworkEndpoint, requireIncarnation bool) bool {
	if endpoint.Node == "" || endpoint.Address == "" {
		return false
	}
	return !requireIncarnation || endpoint.Incarnation != 0
}

func validNetworkPayloadSHA256(value string) bool {
	if len(value) != 71 || value[:7] != "sha256:" {
		return false
	}
	_, err := hex.DecodeString(value[7:])
	return err == nil
}

func networkTransitionLane(transition NetworkTransition) string {
	switch transition.Kind {
	case "partition", "heal", "delay", "disconnect", "reconnect", "directional_delay":
		return "topology"
	case "listen", "listener_close":
		return "listener\x00" + networkEndpointKey(transition.Source)
	case "dial", "accept", "write", "deliver", "close", "reset":
		if transition.Connection != 0 {
			return fmt.Sprintf("connection\x00%020d", transition.Connection)
		}
		return "operation\x00" + string(transition.Kind) + "\x00" + networkEndpointKey(transition.Source) + "\x00" + networkEndpointKey(transition.Destination)
	case "stop", "crash":
		return "lifecycle\x00" + networkEndpointKey(transition.Source)
	default:
		return "unknown\x00" + string(transition.Kind)
	}
}

func encodeRuntimeNetworkRecord(encoder *runtimeNetworkWireEncoder, record NetworkRecord) {
	encoder.uint64(uint64(len(record.Transitions)))
	for _, transition := range record.Transitions {
		encodeRuntimeNetworkTransition(encoder, transition)
	}
	encodeRuntimeNetworkSnapshot(encoder, record.Snapshot)
}

func decodeRuntimeNetworkRecord(decoder *runtimeNetworkWireDecoder) NetworkRecord {
	record := NetworkRecord{}
	for range decoder.count(maximumRuntimeNetworkWireTransitions) {
		if decoder.err != nil {
			break
		}
		record.Transitions = append(record.Transitions, decodeRuntimeNetworkTransition(decoder))
	}
	record.Snapshot = decodeRuntimeNetworkSnapshot(decoder)
	return record
}

func encodeRuntimeNetworkTransition(encoder *runtimeNetworkWireEncoder, transition NetworkTransition) {
	encoder.uint64(transition.Ordinal)
	encoder.string(string(transition.Kind))
	encodeRuntimeNetworkEndpoint(encoder, transition.Source)
	encodeRuntimeNetworkEndpoint(encoder, transition.Destination)
	encoder.uint64(transition.Connection)
	encoder.uint64(transition.Delivery)
	encoder.uint64(transition.Bytes)
	encoder.uint64(transition.Count)
	encoder.uint64(transition.DelayNanos)
	encoder.string(string(transition.Outcome))
	encoder.string(transition.PayloadSHA256)
}

func decodeRuntimeNetworkTransition(decoder *runtimeNetworkWireDecoder) NetworkTransition {
	transition := NetworkTransition{Ordinal: decoder.uint64(), Kind: NetworkTransitionKind(decoder.string())}
	transition.Source = decodeRuntimeNetworkEndpoint(decoder)
	transition.Destination = decodeRuntimeNetworkEndpoint(decoder)
	transition.Connection = decoder.uint64()
	transition.Delivery = decoder.uint64()
	transition.Bytes = decoder.uint64()
	transition.Count = decoder.uint64()
	transition.DelayNanos = decoder.uint64()
	transition.Outcome = NetworkOutcome(decoder.string())
	transition.PayloadSHA256 = decoder.string()
	return transition
}

func encodeRuntimeNetworkEndpoint(encoder *runtimeNetworkWireEncoder, endpoint NetworkEndpoint) {
	encoder.string(string(endpoint.Node))
	encoder.uint64(endpoint.Incarnation)
	encoder.string(endpoint.Address)
	encoder.uint64(endpoint.Port)
}

func decodeRuntimeNetworkEndpoint(decoder *runtimeNetworkWireDecoder) NetworkEndpoint {
	return NetworkEndpoint{Node: NodeID(decoder.string()), Incarnation: decoder.uint64(), Address: decoder.string(), Port: decoder.uint64()}
}

func encodeRuntimeNetworkSnapshot(encoder *runtimeNetworkWireEncoder, snapshot NetworkSnapshot) {
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
		encodeRuntimeNetworkEndpoint(encoder, listener.Endpoint)
		encoder.boolean(listener.Closed)
	}
	encoder.uint64(uint64(len(snapshot.Connections)))
	for _, connection := range snapshot.Connections {
		encoder.uint64(connection.Identity)
		encodeRuntimeNetworkEndpoint(encoder, connection.Client)
		encodeRuntimeNetworkEndpoint(encoder, connection.Server)
		encoder.boolean(connection.Closed)
		encoder.boolean(connection.Reset)
	}
	encoder.uint64(uint64(len(snapshot.Deliveries)))
	for _, delivery := range snapshot.Deliveries {
		encoder.uint64(delivery.Identity)
		encoder.uint64(delivery.Connection)
		encodeRuntimeNetworkEndpoint(encoder, delivery.Source)
		encodeRuntimeNetworkEndpoint(encoder, delivery.Destination)
		encoder.uint64(delivery.Bytes)
		encoder.uint64(delivery.DelayNanos)
	}
	encoder.uint64(snapshot.NextConnection)
	encoder.uint64(snapshot.NextDelivery)
	encoder.string(snapshot.TransitionSHA256)
	encoder.string(snapshot.Identity)
}

func decodeRuntimeNetworkSnapshot(decoder *runtimeNetworkWireDecoder) NetworkSnapshot {
	snapshot := NetworkSnapshot{}
	for range decoder.count(maximumRuntimeNetworkWireNodes) {
		if decoder.err != nil {
			break
		}
		snapshot.Nodes = append(snapshot.Nodes, NetworkNodeSnapshot{
			Node: NodeID(decoder.string()), Address: decoder.string(), LastIncarnation: decoder.uint64(),
			NextListenerPort: decoder.uint64(), NextClientPort: decoder.uint64(),
		})
	}
	for range decoder.count(maximumRuntimeNetworkWireLinks) {
		if decoder.err != nil {
			break
		}
		snapshot.Links = append(snapshot.Links, NetworkLinkSnapshot{
			From: NodeID(decoder.string()), To: NodeID(decoder.string()), Enabled: decoder.boolean(), DelayNanos: decoder.uint64(),
		})
	}
	for range decoder.count(maximumRuntimeNetworkWireListeners) {
		if decoder.err != nil {
			break
		}
		snapshot.Listeners = append(snapshot.Listeners, NetworkListenerSnapshot{Endpoint: decodeRuntimeNetworkEndpoint(decoder), Closed: decoder.boolean()})
	}
	for range decoder.count(maximumRuntimeNetworkWireConnections) {
		if decoder.err != nil {
			break
		}
		snapshot.Connections = append(snapshot.Connections, NetworkConnectionSnapshot{
			Identity: decoder.uint64(), Client: decodeRuntimeNetworkEndpoint(decoder), Server: decodeRuntimeNetworkEndpoint(decoder),
			Closed: decoder.boolean(), Reset: decoder.boolean(),
		})
	}
	for range decoder.count(maximumRuntimeNetworkWireDeliveries) {
		if decoder.err != nil {
			break
		}
		snapshot.Deliveries = append(snapshot.Deliveries, NetworkDeliverySnapshot{
			Identity: decoder.uint64(), Connection: decoder.uint64(), Source: decodeRuntimeNetworkEndpoint(decoder), Destination: decodeRuntimeNetworkEndpoint(decoder),
			Bytes: decoder.uint64(), DelayNanos: decoder.uint64(),
		})
	}
	snapshot.NextConnection = decoder.uint64()
	snapshot.NextDelivery = decoder.uint64()
	snapshot.TransitionSHA256 = decoder.string()
	snapshot.Identity = decoder.string()
	return snapshot
}

func decodeRuntimeNetworkErrorBody(decoder *runtimeNetworkWireDecoder) runtimeNetworkError {
	runtimeErr := runtimeNetworkError{
		Kind: decoder.string(), Message: decoder.string(), Ordinal: decoder.uint64(),
		ExpectedSHA256: decoder.string(), ActualSHA256: decoder.string(),
	}
	if decoder.boolean() {
		expected := decodeRuntimeNetworkTransition(decoder)
		runtimeErr.Expected = &expected
	}
	if decoder.boolean() {
		actual := decodeRuntimeNetworkTransition(decoder)
		runtimeErr.Actual = &actual
	}
	return runtimeErr
}

func encodeRuntimeNetworkTransitions(transitions []NetworkTransition) ([]byte, error) {
	encoder := &runtimeNetworkWireEncoder{}
	encoder.uint64(uint64(len(transitions)))
	for _, transition := range transitions {
		encodeRuntimeNetworkTransition(encoder, transition)
	}
	return encoder.data, encoder.err
}

func encodeRuntimeNetworkSnapshotIdentity(snapshot NetworkSnapshot) ([]byte, error) {
	encoder := &runtimeNetworkWireEncoder{}
	encodeRuntimeNetworkSnapshot(encoder, snapshot)
	return encoder.data, encoder.err
}

func appendRuntimeNetworkUint64(destination []byte, value uint64) []byte {
	return append(destination,
		byte(value), byte(value>>8), byte(value>>16), byte(value>>24),
		byte(value>>32), byte(value>>40), byte(value>>48), byte(value>>56),
	)
}

func readRuntimeNetworkUint64(source []byte) uint64 {
	return uint64(source[0]) |
		uint64(source[1])<<8 |
		uint64(source[2])<<16 |
		uint64(source[3])<<24 |
		uint64(source[4])<<32 |
		uint64(source[5])<<40 |
		uint64(source[6])<<48 |
		uint64(source[7])<<56
}
