package determinism

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

const EventOrderSchema = "gomad.wasm-event-order/v1"
const EventOrderPrefix = "gomad-wasm-event-order-v1 "
const MaximumEventOrderBytes = 32 << 20
const MaximumOrderedEvents = 131072

type OrderedEvent struct {
	Order       uint64          `json:"order"`
	Kind        EventKind       `json:"kind"`
	Instance    string          `json:"instance"`
	Incarnation uint64          `json:"incarnation"`
	Ordinal     uint64          `json:"ordinal"`
	Trace       uint64          `json:"trace"`
	Decision    uint64          `json:"decision"`
	TimeNanos   int64           `json:"time_nanos"`
	Value       json.RawMessage `json:"value"`
}

type EventOrder struct {
	Schema     string          `json:"schema"`
	Complete   bool            `json:"complete"`
	Events     []OrderedEvent  `json:"events"`
	Boundaries []EventBoundary `json:"boundaries"`
	Bytes      uint64          `json:"bytes"`
	Reason     string          `json:"reason,omitempty"`
}

type EventBoundary struct {
	Order           uint64 `json:"order"`
	Instance        string `json:"instance"`
	Incarnation     uint64 `json:"incarnation"`
	Trace           uint64 `json:"trace"`
	Decision        uint64 `json:"decision"`
	TimeNanos       int64  `json:"time_nanos"`
	InspectionOrder uint64 `json:"inspection_order"`
}

func strictEventJSON(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return errors.New("trailing event-order JSON")
	}
	return nil
}

// ParseEventOrder extracts supplemental diagnostics from helper stderr. Other
// telemetry stays outside this channel. A truncated or bounded log never
// establishes complete event ordering.
func ParseEventOrder(stderr []byte) (EventOrder, error) {
	var log EventOrder
	var counted uint64
	ended := false
	for len(stderr) != 0 {
		line, rest, newline := bytes.Cut(stderr, []byte{'\n'})
		stderr = rest
		payload, eventLine := bytes.CutPrefix(line, []byte(EventOrderPrefix))
		if !eventLine {
			continue
		}
		if !newline {
			log.Reason = "truncated event-order frame"
			return log, nil
		}
		if ended {
			return EventOrder{}, errors.New("event-order frame follows terminal")
		}
		var tag struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(payload, &tag); err != nil {
			return EventOrder{}, err
		}
		switch tag.Type {
		case "header":
			var header struct {
				Type   string `json:"type"`
				Schema string `json:"schema"`
			}
			if err := strictEventJSON(payload, &header); err != nil {
				return EventOrder{}, err
			}
			if log.Schema != "" || header.Schema != EventOrderSchema {
				return EventOrder{}, errors.New("duplicate or unsupported event-order header")
			}
			log.Schema = header.Schema
		case "event":
			if log.Schema == "" {
				return EventOrder{}, errors.New("event-order event precedes header")
			}
			var frame struct {
				Type string `json:"type"`
				OrderedEvent
			}
			if err := strictEventJSON(payload, &frame); err != nil {
				return EventOrder{}, err
			}
			if len(log.Events) >= MaximumOrderedEvents || frame.Order != uint64(len(log.Events))+1 || !knownEventKind(frame.Kind) || len(frame.Value) == 0 {
				return EventOrder{}, errors.New("invalid event-order event or event bound")
			}
			var value bytes.Buffer
			if err := json.Compact(&value, frame.Value); err != nil {
				return EventOrder{}, err
			}
			frame.Value = value.Bytes()
			log.Events = append(log.Events, frame.OrderedEvent)
		case "boundary":
			var frame struct {
				Type string `json:"type"`
				EventBoundary
			}
			if err := strictEventJSON(payload, &frame); err != nil {
				return EventOrder{}, err
			}
			if log.Schema == "" || frame.Order != uint64(len(log.Events)) || frame.Instance == "" || frame.Incarnation == 0 || len(log.Boundaries) >= MaximumOrderedEvents {
				return EventOrder{}, errors.New("invalid event-order boundary marker")
			}
			log.Boundaries = append(log.Boundaries, frame.EventBoundary)
		case "complete":
			var terminal struct {
				Type   string `json:"type"`
				Events uint64 `json:"events"`
				Bytes  uint64 `json:"bytes"`
			}
			if err := strictEventJSON(payload, &terminal); err != nil {
				return EventOrder{}, err
			}
			if log.Schema == "" || terminal.Events != uint64(len(log.Events)) || terminal.Bytes != counted {
				return EventOrder{}, errors.New("event-order terminal count or byte commitment differs")
			}
			log.Complete, log.Bytes, ended = true, counted, true
			continue
		case "incomplete":
			var terminal struct {
				Type   string `json:"type"`
				Reason string `json:"reason"`
			}
			if err := strictEventJSON(payload, &terminal); err != nil {
				return EventOrder{}, err
			}
			log.Reason, ended = terminal.Reason, true
			continue
		default:
			return EventOrder{}, fmt.Errorf("unknown event-order frame kind %q", tag.Type)
		}
		counted += uint64(len(line) + 1)
		if counted > MaximumEventOrderBytes {
			return EventOrder{}, errors.New("event-order byte bound exceeded")
		}
	}
	if !log.Complete && log.Reason == "" {
		log.Reason = "missing complete supplemental event-order diagnostic"
	}
	return log, nil
}

func knownEventKind(kind EventKind) bool {
	switch kind {
	case KindChoice, KindDrawCounter, KindClock, KindEntropy, KindWASIOp, KindModelTransition, KindOutput, KindWorldOrder:
		return true
	default:
		return false
	}
}

func compareEventOrder(expected, actual Run, canonicalDifference bool) (Report, error) {
	for _, run := range []Run{expected, actual} {
		if run.OrderLog == nil || run.OrderLog.Schema != EventOrderSchema || !run.OrderLog.Complete || len(run.Incarnations) != 0 && len(run.OrderLog.Events) == 0 {
			reason := "missing complete supplemental event-order diagnostic"
			if run.OrderLog != nil && run.OrderLog.Reason != "" {
				reason += ": " + run.OrderLog.Reason
			}
			return Report{Schema: ReportSchema, Status: "undiagnosable", Reason: reason}, nil
		}
	}
	index, different := firstDifference(expected.OrderLog.Events, actual.OrderLog.Events, func(a, b OrderedEvent) bool {
		return a.Order == b.Order && a.Kind == b.Kind && a.Instance == b.Instance && a.Incarnation == b.Incarnation && a.Ordinal == b.Ordinal && a.Trace == b.Trace && a.Decision == b.Decision && a.TimeNanos == b.TimeNanos && bytes.Equal(a.Value, b.Value)
	})
	if !different {
		if canonicalDifference {
			return Report{Schema: ReportSchema, Status: "undiagnosable", Reason: "canonical evidence differs without a matching event-order diagnostic difference"}, nil
		}
		return Report{Schema: ReportSchema, Status: "identical"}, nil
	}
	e, a := at(expected.OrderLog.Events, index), at(actual.OrderLog.Events, index)
	ref := e
	if index >= len(expected.OrderLog.Events) {
		ref = a
	}
	value := func(event OrderedEvent, present bool) (string, error) {
		if !present {
			return "absent", nil
		}
		data, err := json.Marshal(event)
		return string(data), err
	}
	left, err := value(e, index < len(expected.OrderLog.Events))
	if err != nil {
		return Report{}, err
	}
	right, err := value(a, index < len(actual.OrderLog.Events))
	if err != nil {
		return Report{}, err
	}
	first := Divergence{Kind: ref.Kind, Instance: ref.Instance, Incarnation: ref.Incarnation, Ordinal: ref.Ordinal,
		At: Coordinate{Shared: ref.Order, ChoiceOrdinal: ref.Trace, VirtualTime: ref.TimeNanos}, Field: "ordered-event", Expected: left, Actual: right}
	var selected *ReplayBoundary
	for _, marker := range expected.OrderLog.Boundaries {
		if marker.Order >= ref.Order {
			continue
		}
		for index := range expected.Boundaries {
			boundary := &expected.Boundaries[index]
			if boundary.At.Shared == marker.InspectionOrder && boundary.Instance == marker.Instance && boundary.Incarnation == marker.Incarnation && boundary.Trace == marker.Trace && boundary.Decision == marker.Decision && boundary.At.VirtualTime == marker.TimeNanos {
				if selected == nil || selected.At.Shared < boundary.At.Shared {
					selected = boundary
				}
				break
			}
		}
	}
	var replay *Reproducer
	if selected == nil {
		replay = makeReproducer(expected, actual, 0, ref.Instance, ref.Incarnation, "no authenticated inspection boundary precedes the first differing event")
	} else {
		replay = makeReproducer(expected, actual, selected.Decision, selected.Instance, selected.Incarnation, "")
	}
	known := true
	return Report{Schema: ReportSchema, Status: "diverged", Ordering: "helper-event", VirtualTimeAvailable: &known, First: &first, Reproducer: replay}, nil
}
