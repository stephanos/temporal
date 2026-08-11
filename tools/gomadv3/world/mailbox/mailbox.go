// Package mailbox is an explicit deterministic adapter pilot. It models
// mailbox receive completion without performing host I/O or creating a
// dispatcher goroutine.
package mailbox

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"slices"
	"sort"
	"unicode/utf8"

	"go.temporal.io/server/tools/gomadv3/world"
)

const SchemaVersion uint32 = 1

type ReceiveState string

const (
	ReceivePending   ReceiveState = "pending"
	ReceiveQueued    ReceiveState = "queued"
	ReceiveCanceled  ReceiveState = "canceled"
	ReceiveDelivered ReceiveState = "delivered"
)

type ReceiveSnapshot struct {
	RequestID world.RequestID `json:"requestId"`
	EventID   world.EventID   `json:"eventId,omitempty"`
	Mailbox   string          `json:"mailbox"`
	Priority  world.Priority  `json:"priority"`
	State     ReceiveState    `json:"state"`
}

type Snapshot struct {
	SchemaVersion uint32            `json:"schemaVersion"`
	Receives      []ReceiveSnapshot `json:"receives"`
	Digest        world.Digest      `json:"digest"`
}

type Adapter struct {
	core *world.World
}

func New(core *world.World) (*Adapter, error) {
	if core == nil {
		return nil, fmt.Errorf("mailbox World is required")
	}
	if _, err := DeriveSnapshot(core.Snapshot()); err != nil {
		return nil, err
	}
	return &Adapter{core: core}, nil
}

func Restore(core *world.World, snapshot Snapshot) (*Adapter, error) {
	if core == nil || snapshot.SchemaVersion != SchemaVersion || snapshot.Digest != snapshotDigest(snapshot) {
		return nil, fmt.Errorf("invalid mailbox snapshot identity")
	}
	derived, err := DeriveSnapshot(core.Snapshot())
	if err != nil {
		return nil, err
	}
	if snapshot.Digest != derived.Digest || !slices.Equal(snapshot.Receives, derived.Receives) {
		return nil, fmt.Errorf("mailbox snapshot does not match World")
	}
	return &Adapter{core: core}, nil
}

func (adapter *Adapter) Receive(mailbox string, priority world.Priority, payload []byte) (world.RequestID, error) {
	if mailbox == "" || !utf8.ValidString(mailbox) {
		return 0, fmt.Errorf("invalid mailbox identity")
	}
	id, err := adapter.core.Register(world.Request{
		Kind: "receive", Resource: world.ResourceID{Adapter: "mailbox", Kind: "queue", Key: mailbox}, Priority: priority, Payload: payload,
	})
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (adapter *Adapter) MessageReady(requestID world.RequestID, at world.LogicalTime, payload []byte) (world.EventID, error) {
	if _, err := findReceive(adapter.core.Snapshot(), requestID); err != nil {
		return 0, fmt.Errorf("unknown mailbox receive %d", requestID)
	}
	eventID, err := adapter.core.Ready(world.Readiness{RequestID: requestID, At: at, Kind: "message", Payload: payload})
	if err != nil {
		return 0, err
	}
	return eventID, nil
}

func (adapter *Adapter) Cancel(requestID world.RequestID) (world.Cancellation, error) {
	if _, err := findReceive(adapter.core.Snapshot(), requestID); err != nil {
		return world.Cancellation{}, fmt.Errorf("unknown mailbox receive %d", requestID)
	}
	cancellation, err := adapter.core.Cancel(requestID)
	if err != nil {
		return world.Cancellation{}, err
	}
	return cancellation, nil
}

func (adapter *Adapter) Drive() (world.Quiescence, error) {
	quiescence, err := adapter.core.Quiesce()
	if err != nil {
		return world.Quiescence{}, err
	}
	snapshot := adapter.core.Snapshot()
	for _, delivery := range quiescence.Deliveries {
		receive, findErr := findReceive(snapshot, delivery.RequestID)
		if findErr != nil || receive.EventID != delivery.EventID {
			return world.Quiescence{}, fmt.Errorf("World delivered unknown mailbox event %d", delivery.EventID)
		}
	}
	return quiescence, nil
}

func (adapter *Adapter) Snapshot() (Snapshot, error) {
	return DeriveSnapshot(adapter.core.Snapshot())
}

func DeriveSnapshot(core world.Snapshot) (Snapshot, error) {
	snapshot := Snapshot{SchemaVersion: SchemaVersion, Receives: []ReceiveSnapshot{}}
	for _, request := range core.Requests {
		if request.Request.Resource.Adapter != "mailbox" {
			continue
		}
		if request.Request.Kind != "receive" || request.Request.Resource.Kind != "queue" || request.Request.Resource.Key == "" {
			return Snapshot{}, fmt.Errorf("World contains an incompatible mailbox-owned request %d", request.ID)
		}
		state, err := receiveState(request.State)
		if err != nil {
			return Snapshot{}, err
		}
		snapshot.Receives = append(snapshot.Receives, ReceiveSnapshot{
			RequestID: request.ID, EventID: request.EventID, Mailbox: request.Request.Resource.Key, Priority: request.Request.Priority, State: state,
		})
	}
	sort.Slice(snapshot.Receives, func(i, j int) bool { return snapshot.Receives[i].RequestID < snapshot.Receives[j].RequestID })
	snapshot.Digest = snapshotDigest(snapshot)
	return snapshot, nil
}

func receiveState(state world.RequestState) (ReceiveState, error) {
	switch state {
	case world.RequestPending:
		return ReceivePending, nil
	case world.RequestQueued:
		return ReceiveQueued, nil
	case world.RequestCanceled:
		return ReceiveCanceled, nil
	case world.RequestDelivered:
		return ReceiveDelivered, nil
	default:
		return "", fmt.Errorf("invalid mailbox-owned request state %q", state)
	}
}

func findReceive(snapshot world.Snapshot, requestID world.RequestID) (ReceiveSnapshot, error) {
	derived, err := DeriveSnapshot(snapshot)
	if err != nil {
		return ReceiveSnapshot{}, err
	}
	for _, receive := range derived.Receives {
		if receive.RequestID == requestID {
			return receive, nil
		}
	}
	return ReceiveSnapshot{}, fmt.Errorf("unknown mailbox receive %d", requestID)
}

func snapshotDigest(snapshot Snapshot) world.Digest {
	data := []byte("gomadv3/world/mailbox-snapshot/v1\x00")
	data = appendUint32(data, snapshot.SchemaVersion)
	data = appendUint64(data, uint64(len(snapshot.Receives)))
	for _, receive := range snapshot.Receives {
		data = appendUint64(data, uint64(receive.RequestID))
		data = appendUint64(data, uint64(receive.EventID))
		data = appendString(data, receive.Mailbox)
		data = appendUint32(data, uint32(receive.Priority))
		data = appendString(data, string(receive.State))
	}
	digest := sha256.Sum256(data)
	return world.Digest(hex.EncodeToString(digest[:]))
}

func appendUint64(data []byte, value uint64) []byte {
	var encoded [8]byte
	binary.BigEndian.PutUint64(encoded[:], value)
	return append(data, encoded[:]...)
}

func appendUint32(data []byte, value uint32) []byte {
	var encoded [4]byte
	binary.BigEndian.PutUint32(encoded[:], value)
	return append(data, encoded[:]...)
}

func appendString(data []byte, value string) []byte {
	data = appendUint64(data, uint64(len(value)))
	return append(data, value...)
}
