package world

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
)

const (
	SchemaVersion uint32      = 1
	InitialTime   LogicalTime = 946684800000000000
)

type LogicalTime int64
type Seed uint64
type RequestID uint64
type EventID uint64
type Sequence uint64
type Priority uint16
type Digest string

type Config struct {
	Seed   Seed   `json:"seed"`
	Limits Limits `json:"limits"`
}

type Limits struct {
	MaxRequests     uint64 `json:"maxRequests"`
	MaxEvents       uint64 `json:"maxEvents"`
	MaxQueuedEvents uint64 `json:"maxQueuedEvents"`
	MaxTransitions  uint64 `json:"maxTransitions"`
	MaxPayloadBytes uint64 `json:"maxPayloadBytes"`
	MaxStringBytes  uint32 `json:"maxStringBytes"`
}

type ResourceID struct {
	Adapter string `json:"adapter"`
	Kind    string `json:"kind"`
	Key     string `json:"key"`
}

type Request struct {
	Kind     string     `json:"kind"`
	Resource ResourceID `json:"resource"`
	Priority Priority   `json:"priority"`
	Payload  []byte     `json:"payload,omitempty"`
}

type Readiness struct {
	RequestID        RequestID   `json:"requestId"`
	At               LogicalTime `json:"at"`
	Kind             string      `json:"kind"`
	Payload          []byte      `json:"payload,omitempty"`
	EquivalenceClass string      `json:"equivalenceClass,omitempty"`
}

type CancelStatus string

const (
	CancelWon              CancelStatus = "won"
	CancelAlreadyCanceled  CancelStatus = "already-canceled"
	CancelAlreadyDelivered CancelStatus = "already-delivered"
)

type Cancellation struct {
	RequestID RequestID    `json:"requestId"`
	EventID   EventID      `json:"eventId,omitempty"`
	Status    CancelStatus `json:"status"`
}

type Delivery struct {
	RequestID RequestID   `json:"requestId"`
	EventID   EventID     `json:"eventId"`
	At        LogicalTime `json:"at"`
	Kind      string      `json:"kind"`
	Payload   []byte      `json:"payload,omitempty"`
}

type QuiescenceKind string

const (
	QuiescenceDelivered QuiescenceKind = "delivered"
	QuiescenceDeadlock  QuiescenceKind = "deadlock"
	QuiescenceIdle      QuiescenceKind = "idle"
)

type Quiescence struct {
	Kind       QuiescenceKind `json:"kind"`
	Before     LogicalTime    `json:"before"`
	After      LogicalTime    `json:"after"`
	Deliveries []Delivery     `json:"deliveries,omitempty"`
	Blocked    []RequestID    `json:"blocked,omitempty"`
}

type ReplayProgress struct {
	Cursor   uint64 `json:"cursor"`
	Expected uint64 `json:"expected"`
}

func (value Seed) MarshalJSON() ([]byte, error)      { return marshalUint64(uint64(value)), nil }
func (value RequestID) MarshalJSON() ([]byte, error) { return marshalUint64(uint64(value)), nil }
func (value EventID) MarshalJSON() ([]byte, error)   { return marshalUint64(uint64(value)), nil }
func (value Sequence) MarshalJSON() ([]byte, error)  { return marshalUint64(uint64(value)), nil }
func (value LogicalTime) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(strconv.FormatInt(int64(value), 10))), nil
}

func (value *Seed) UnmarshalJSON(data []byte) error {
	parsed, err := unmarshalUint64(data, true)
	*value = Seed(parsed)
	return err
}

func (value *RequestID) UnmarshalJSON(data []byte) error {
	parsed, err := unmarshalUint64(data, false)
	*value = RequestID(parsed)
	return err
}

func (value *EventID) UnmarshalJSON(data []byte) error {
	parsed, err := unmarshalUint64(data, false)
	*value = EventID(parsed)
	return err
}

func (value *Sequence) UnmarshalJSON(data []byte) error {
	parsed, err := unmarshalUint64(data, false)
	*value = Sequence(parsed)
	return err
}

func (value *LogicalTime) UnmarshalJSON(data []byte) error {
	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return fmt.Errorf("logical time must be a decimal string")
	}
	if text == "" || len(text) > 1 && text[0] == '0' || text[0] == '-' || text[0] == '+' {
		return fmt.Errorf("logical time is not canonical")
	}
	for _, character := range text {
		if character < '0' || character > '9' {
			return fmt.Errorf("logical time is not canonical")
		}
	}
	parsed, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return fmt.Errorf("logical time is out of range")
	}
	*value = LogicalTime(parsed)
	return nil
}

func marshalUint64(value uint64) []byte {
	return []byte(strconv.Quote(strconv.FormatUint(value, 10)))
}

func unmarshalUint64(data []byte, zeroAllowed bool) (uint64, error) {
	var text string
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&text); err != nil {
		return 0, fmt.Errorf("identity must be a decimal string")
	}
	if text == "" || len(text) > 1 && text[0] == '0' || text[0] == '-' || text[0] == '+' {
		return 0, fmt.Errorf("identity is not canonical")
	}
	for _, character := range text {
		if character < '0' || character > '9' {
			return 0, fmt.Errorf("identity is not canonical")
		}
	}
	parsed, err := strconv.ParseUint(text, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("identity is out of range")
	}
	if parsed == 0 && !zeroAllowed {
		return 0, fmt.Errorf("identity zero is invalid")
	}
	return parsed, nil
}
