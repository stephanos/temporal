package wasi

import (
	"encoding/json"
	"math"
)

type Event struct {
	Userdata uint64 `json:"userdata"`
	Errno    uint16 `json:"errno"`
	Type     uint8  `json:"type"`
	Nbytes   uint64 `json:"nbytes"`
	Flags    uint16 `json:"flags"`
}
type eventsOutput struct {
	Events []Event `json:"events"`
}
type subscription struct {
	Userdata  uint64 `json:"userdata"`
	Type      uint8  `json:"type"`
	ClockID   uint32 `json:"clock_id"`
	Timeout   uint64 `json:"timeout"`
	Precision uint64 `json:"precision"`
	Flags     uint16 `json:"flags"`
	FD        uint32 `json:"fd"`
}

func (e *Environment) clockCall(call Call) (any, uint16, string, error) {
	if call.Op == "clock_time_get" {
		var input struct {
			ClockID   uint32 `json:"clock_id"`
			Precision uint64 `json:"precision"`
		}
		if err := decodeInput(call.Input, &input, "clock_id", "precision"); err != nil {
			return nil, 0, "", err
		}
		if input.ClockID > 1 {
			return nil, 0, "CPU clocks outside stock WASI profile", nil
		}
		if e.config.Clock.EpochNanos > math.MaxInt64-e.now || e.config.Clock.ReadStepNanos > math.MaxInt64-e.config.Clock.EpochNanos-e.now {
			return nil, 0, "", capacity("clock-nanoseconds")
		}
		timestamp := e.now
		if input.ClockID == 0 {
			timestamp += e.config.Clock.EpochNanos
		}
		e.now += e.config.Clock.ReadStepNanos
		return timestampOutput{timestamp}, 0, "", nil
	}
	var input struct {
		Subscriptions []json.RawMessage `json:"subscriptions"`
	}
	if err := decodeInput(call.Input, &input, "subscriptions"); err != nil {
		return nil, 0, "", err
	}
	if len(input.Subscriptions) == 0 {
		return nil, errnoInval, "", nil
	}
	if uint64(len(input.Subscriptions)) > e.config.Limits.PendingEvents {
		return nil, 0, "", capacity("pending-events")
	}
	events := make([]Event, len(input.Subscriptions))
	deadlines := make([]uint64, len(input.Subscriptions))
	earliest := uint64(math.MaxUint64)
	for i, raw := range input.Subscriptions {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(raw, &fields); err != nil {
			return nil, 0, "", invalid("subscription")
		}
		var kind uint8
		if err := json.Unmarshal(fields["type"], &kind); err != nil {
			return nil, 0, "", invalid("subscription type")
		}
		keys := []string{"userdata", "type", "fd"}
		if kind == 0 {
			keys = []string{"userdata", "type", "clock_id", "timeout", "precision", "flags"}
		}
		var sub subscription
		if err := decodeInput(raw, &sub, keys...); err != nil {
			return nil, 0, "", err
		}
		event := Event{Userdata: sub.Userdata, Type: sub.Type}
		at := e.now
		switch sub.Type {
		case 0:
			if sub.ClockID > 1 {
				return nil, 0, "CPU poll clock outside stock WASI profile", nil
			}
			if sub.Flags&^uint16(1) != 0 {
				return nil, errnoInval, "", nil
			}
			if sub.Flags&1 != 0 {
				at = sub.Timeout
				if sub.ClockID == 0 {
					if at < e.config.Clock.EpochNanos {
						at = 0
					} else {
						at -= e.config.Clock.EpochNanos
					}
				}
				at = max(at, e.now)
			} else {
				if sub.Timeout > math.MaxInt64-e.config.Clock.EpochNanos-e.now {
					return nil, errnoInval, "", nil
				}
				at += sub.Timeout
			}
			if at > math.MaxInt64-e.config.Clock.EpochNanos {
				return nil, errnoInval, "", nil
			}
		case 1, 2:
			right := rightPoll
			if sub.Type == 1 {
				right |= rightRead
			} else {
				right |= rightWrite
			}
			d, errno := e.getFD(sub.FD, right)
			event.Errno = errno
			if errno == 0 {
				if d.node.directory {
					event.Errno = errnoInval
				} else if sub.Type == 1 {
					if d.offset < uint64(len(d.node.data)) {
						event.Nbytes = uint64(len(d.node.data)) - d.offset
					} else {
						event.Flags = 1
					}
				}
			}
		default:
			return nil, errnoInval, "", nil
		}
		events[i] = event
		deadlines[i] = at
		earliest = min(earliest, at)
	}
	// A blocking import advances this exploratory profile; it makes no assertion about runnable guest goroutines.
	e.now = earliest
	ready := make([]Event, 0, len(events))
	for i, event := range events {
		if deadlines[i] <= e.now {
			ready = append(ready, event)
		}
	}
	return eventsOutput{ready}, 0, "", nil
}
