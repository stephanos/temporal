package world

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

const frozenWorldWireDigest = "170566849ca1809a57f609eb981d70af4a10552eaf08f5ccc44402d146195bed"

func populateWorldWireValue(value reflect.Value, text string, numeric, collection int, fields map[string]bool) {
	switch value.Kind() {
	case reflect.Struct:
		for index := 0; index < value.NumField(); index++ {
			field := value.Type().Field(index)
			fields[value.Type().Name()+"."+field.Name+" "+string(field.Tag)] = true
			populateWorldWireValue(value.Field(index), text, numeric, collection, fields)
		}
	case reflect.Pointer:
		if collection == 0 {
			return
		}
		value.Set(reflect.New(value.Type().Elem()))
		populateWorldWireValue(value.Elem(), text, numeric, collection, fields)
	case reflect.Slice:
		if collection == 0 {
			return
		}
		length := 0
		if collection == 2 {
			length = 2
		}
		if value.Type().Elem().Kind() == reflect.Uint8 && collection == 2 {
			length = 256
		}
		value.Set(reflect.MakeSlice(value.Type(), length, length))
		for index := range length {
			if value.Type().Elem().Kind() == reflect.Uint8 {
				value.Index(index).SetUint(uint64(index))
				continue
			}
			populateWorldWireValue(value.Index(index), text, numeric, collection, fields)
		}
	case reflect.String:
		value.SetString(text)
	case reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		var number uint64
		if numeric == 1 {
			number = 1
		}
		if numeric == 2 {
			number = ^uint64(0) >> (64 - value.Type().Bits())
		}
		value.SetUint(number)
	case reflect.Int64:
		var number int64
		if numeric == 1 {
			number = int64(InitialTime)
		}
		if numeric == 2 {
			number = int64(^uint64(0) >> 1)
		}
		value.SetInt(number)
	default:
		panic(fmt.Sprintf("uncovered World wire field kind %s", value.Kind()))
	}
}

func encodeFrozenWorldValue(value any) ([]byte, error) {
	switch value := value.(type) {
	case Snapshot:
		return encodeWorldJSON(value)
	case ReplayPlan:
		return encodeWorldJSON(value)
	case Transition:
		return encodeWorldJSON(value)
	case Terminal:
		return encodeWorldJSON(value)
	case RequestSnapshot:
		return encodeWorldJSON(value)
	case EventSnapshot:
		return encodeWorldJSON(value)
	case Quiescence:
		return encodeWorldJSON(value)
	case Cancellation:
		return encodeWorldJSON(value)
	case LogicalTime:
		return encodeWorldJSON(value)
	default:
		return nil, fmt.Errorf("uncovered World wire root %T", value)
	}
}

func TestWorldJSONWireCompatibility(t *testing.T) {
	count := 0
	digest := sha256.New()
	compare := func(name string, value any) []byte {
		t.Helper()
		encoded, err := encodeFrozenWorldValue(value)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if bytes.HasSuffix(encoded, []byte{'\n'}) {
			t.Fatalf("%s has an extra newline", name)
		}
		if _, err := fmt.Fprintf(digest, "%d:%s:%d:", len(name), name, len(encoded)); err != nil {
			t.Fatal(err)
		}
		if _, err := digest.Write(encoded); err != nil {
			t.Fatal(err)
		}
		count++
		return encoded
	}

	fields := make(map[string]bool)
	texts := []string{"", "ASCII", "<>&", "\"\\\b\f\n\r\t\x00\x01\x1f\x7f", "é世界🙂", "\u2028\u2029", "literal \\u003c and \\ud800"}
	for character := range 128 {
		texts = append(texts, string(rune(character)))
	}
	roots := []reflect.Type{reflect.TypeFor[ReplayPlan](), reflect.TypeFor[Terminal](), reflect.TypeFor[Transition]()}
	for _, root := range roots {
		for textIndex, text := range texts {
			for numeric := range 3 {
				for collection := range 3 {
					value := reflect.New(root).Elem()
					populateWorldWireValue(value, text, numeric, collection, fields)
					compare(fmt.Sprintf("structural/%s/text%d/number%d/collection%d", root.Name(), textIndex, numeric, collection), value.Interface())
				}
			}
		}
	}
	for _, kind := range []TerminalKind{TerminalNone, TerminalDelivered, TerminalIdle, TerminalDeadlock, TerminalCapacity, TerminalReplayDivergence, TerminalInvalidInput} {
		terminal := Terminal{Kind: kind}
		if kind == TerminalCapacity || kind == TerminalReplayDivergence || kind == TerminalInvalidInput {
			terminal.Detail = "<>&\u2028\u2029"
		}
		if err := validateTerminal(terminal); err != nil {
			t.Fatal(err)
		}
		compare("terminal/"+string(kind), terminal)
	}
	for _, state := range []RequestState{RequestPending, RequestQueued, RequestCanceled, RequestDelivered} {
		compare("request-state/"+string(state), RequestSnapshot{State: state})
	}
	for _, state := range []EventState{EventQueued, EventCanceled, EventDelivered} {
		compare("event-state/"+string(state), EventSnapshot{State: state})
	}
	for _, kind := range []QuiescenceKind{QuiescenceDelivered, QuiescenceDeadlock, QuiescenceIdle} {
		compare("quiescence/"+string(kind), Quiescence{Kind: kind})
	}
	for _, status := range []CancelStatus{CancelWon, CancelAlreadyCanceled, CancelAlreadyDelivered} {
		compare("cancel/"+string(status), Cancellation{Status: status})
	}
	compare("negative-logical-time-encoding-only", LogicalTime(-1))
	compare("min-logical-time-encoding-only", LogicalTime(-1<<63))
	assertWorldModelWire(t, texts[:7], compare)

	if count != 3751 {
		t.Fatalf("wire comparison count = %d, want 3751", count)
	}
	if len(fields) != 46 {
		t.Fatalf("covered serialized fields = %d, want 46", len(fields))
	}
	if got := fmt.Sprintf("%x", digest.Sum(nil)); got != frozenWorldWireDigest {
		t.Fatalf("World wire digest = %s, want %s", got, frozenWorldWireDigest)
	}
}

func assertWorldModelWire(t *testing.T, texts []string, compare func(string, any) []byte) {
	t.Helper()
	for textIndex, text := range texts {
		model := newTestWorld(t, ^Seed(0))
		initial := model.Snapshot()
		checkSnapshot := func(label string) {
			snapshot := model.Snapshot()
			got, err := EncodeSnapshot(snapshot)
			if err != nil {
				t.Fatalf("snapshot %s: %v", label, err)
			}
			if _, err := DecodeSnapshot(got); err != nil {
				t.Fatalf("decode snapshot %s: %v", label, err)
			}
			if !bytes.HasPrefix(got, compressedEnvelopeMagic[:]) {
				t.Fatalf("snapshot %s is not a binary envelope", label)
			}
		}
		checkSnapshot("initial")
		request := Request{Kind: "wait" + text, Resource: ResourceID{Adapter: "memory", Kind: "cell", Key: text}, Priority: ^Priority(0), Payload: []byte{0, 1, 127, 128, 255}}
		requestID, err := model.Register(request)
		if err != nil {
			t.Fatal(err)
		}
		checkSnapshot("pending")
		if _, err := model.Quiesce(); err != nil {
			t.Fatal(err)
		}
		checkSnapshot("deadlock")
		if _, err := model.Ready(Readiness{RequestID: requestID, At: InitialTime + 100, Kind: "ready" + text, Payload: []byte{0, 255}, EquivalenceClass: text}); err != nil {
			t.Fatal(err)
		}
		checkSnapshot("queued")
		if _, err := model.Quiesce(); err != nil {
			t.Fatal(err)
		}
		checkSnapshot("delivered")
		if _, err := model.Cancel(requestID); err != nil {
			t.Fatal(err)
		}
		requestID, err = model.Register(request)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := model.Ready(Readiness{RequestID: requestID, At: InitialTime + 200, Kind: "done" + text}); err != nil {
			t.Fatal(err)
		}
		if _, err := model.Cancel(requestID); err != nil {
			t.Fatal(err)
		}
		checkSnapshot("canceled")
		if _, err := model.Cancel(requestID); err != nil {
			t.Fatal(err)
		}
		if _, err := model.Quiesce(); err != nil {
			t.Fatal(err)
		}
		checkSnapshot("idle")

		final := model.Snapshot()
		plan := ReplayPlan{SchemaVersion: SchemaVersion, InitialDigest: initial.StateDigest, FinalDigest: final.StateDigest, Transitions: final.Transitions}
		wantPlan := compare(fmt.Sprintf("model%d/replay", textIndex), plan)
		gotPlan, err := EncodeReplayPlan(plan)
		if err != nil || !bytes.Equal(gotPlan, wantPlan) {
			t.Fatalf("replay plan = %s, %v; want %s", gotPlan, err, wantPlan)
		}
		if _, err := DecodeReplayPlan(wantPlan); err != nil {
			t.Fatal(err)
		}
		var wantLines bytes.Buffer
		for index, transition := range final.Transitions {
			wantLines.Write(compare(fmt.Sprintf("model%d/transition%d/%s", textIndex, index, transition.Kind), transition))
			wantLines.WriteByte('\n')
		}
		gotLines, err := EncodeTransitions(final.Transitions)
		if err != nil || !bytes.Equal(gotLines, wantLines.Bytes()) {
			t.Fatalf("transition JSONL = %s, %v; want %s", gotLines, err, wantLines.Bytes())
		}

		recording := Recording{Initial: initial, Final: final, Terminal: Terminal{Kind: TerminalIdle}}
		gotRecording, err := EncodeRecording(recording)
		if err != nil {
			t.Fatal(err)
		}
		wantRecording := append([]byte(nil), recordingMagic[:]...)
		initialPart, err := EncodeSnapshot(recording.Initial)
		if err != nil {
			t.Fatal(err)
		}
		finalPart, err := EncodeSnapshot(recording.Final)
		if err != nil {
			t.Fatal(err)
		}
		terminalPart := compare(fmt.Sprintf("model%d/recording-terminal", textIndex), recording.Terminal)
		for _, part := range [][]byte{initialPart, finalPart, terminalPart} {
			wantRecording = binary.BigEndian.AppendUint64(wantRecording, uint64(len(part)))
			wantRecording = append(wantRecording, part...)
		}
		if !bytes.Equal(gotRecording, wantRecording) {
			t.Fatal("complete recording frame changed")
		}
		if _, err := DecodeRecording(wantRecording); err != nil {
			t.Fatal(err)
		}
	}
}

func TestWorldEncodersRejectInvalidOriginalStrings(t *testing.T) {
	bad := string([]byte{0xff})
	digest := emptyDigest()
	baseRequest := Request{Kind: "wait", Resource: ResourceID{Adapter: "memory", Kind: "cell", Key: "key"}}
	baseReadiness := Readiness{RequestID: 1, At: InitialTime, Kind: "ready", EquivalenceClass: "class"}
	tests := []struct {
		name       string
		transition Transition
	}{
		{name: "request kind", transition: Transition{Sequence: 1, Kind: "register", PreviousDigest: digest, Register: &RegisterTransition{Request: func() Request { value := baseRequest; value.Kind = bad; return value }(), RequestID: 1}}},
		{name: "request adapter", transition: Transition{Sequence: 1, Kind: "register", PreviousDigest: digest, Register: &RegisterTransition{Request: func() Request { value := baseRequest; value.Resource.Adapter = bad; return value }(), RequestID: 1}}},
		{name: "request resource kind", transition: Transition{Sequence: 1, Kind: "register", PreviousDigest: digest, Register: &RegisterTransition{Request: func() Request { value := baseRequest; value.Resource.Kind = bad; return value }(), RequestID: 1}}},
		{name: "request key", transition: Transition{Sequence: 1, Kind: "register", PreviousDigest: digest, Register: &RegisterTransition{Request: func() Request { value := baseRequest; value.Resource.Key = bad; return value }(), RequestID: 1}}},
		{name: "readiness kind", transition: Transition{Sequence: 1, Kind: "ready", PreviousDigest: digest, Ready: &ReadyTransition{Readiness: func() Readiness { value := baseReadiness; value.Kind = bad; return value }(), EventID: 1}}},
		{name: "readiness equivalence class", transition: Transition{Sequence: 1, Kind: "ready", PreviousDigest: digest, Ready: &ReadyTransition{Readiness: func() Readiness { value := baseReadiness; value.EquivalenceClass = bad; return value }(), EventID: 1}}},
		{name: "cancellation status", transition: Transition{Sequence: 1, Kind: "cancel", PreviousDigest: digest, Cancel: &CancelTransition{RequestID: 1, Cancellation: Cancellation{RequestID: 1, Status: CancelStatus(bad)}}}},
		{name: "quiescence kind", transition: Transition{Sequence: 1, Kind: "quiesce", PreviousDigest: digest, Quiesce: &QuiesceTransition{Result: Quiescence{Kind: QuiescenceKind(bad)}}}},
		{name: "delivery kind", transition: Transition{Sequence: 1, Kind: "quiesce", PreviousDigest: digest, Quiesce: &QuiesceTransition{Result: Quiescence{Kind: QuiescenceDelivered, Deliveries: []Delivery{{RequestID: 1, EventID: 1, At: InitialTime, Kind: bad}}}}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.transition.Digest = transitionDigest(test.transition)
			if _, err := EncodeTransitions([]Transition{test.transition}); err == nil || !strings.Contains(err.Error(), "invalid UTF-8") {
				t.Fatalf("EncodeTransitions() error = %v", err)
			}
			plan := ReplayPlan{SchemaVersion: SchemaVersion, InitialDigest: digest, Transitions: []Transition{test.transition}, FinalDigest: digest}
			if _, err := EncodeReplayPlan(plan); err == nil || !strings.Contains(err.Error(), "invalid UTF-8") {
				t.Fatalf("EncodeReplayPlan() error = %v", err)
			}
		})
	}

	transition := tests[0].transition
	transition.Digest = transitionDigest(transition)
	plan := ReplayPlan{SchemaVersion: 0, InitialDigest: digest, Transitions: []Transition{transition}, FinalDigest: digest}
	if _, err := EncodeReplayPlan(plan); !errors.Is(err, ErrInvalidSnapshot) || !strings.Contains(err.Error(), "replay plan identity") {
		t.Fatalf("EncodeReplayPlan() error order = %v", err)
	}

	snapshot := newTestWorld(t, 0).Snapshot()
	recording := Recording{Initial: snapshot, Final: snapshot, Terminal: Terminal{Kind: TerminalInvalidInput, Detail: bad}}
	if _, err := EncodeRecording(recording); err == nil || !strings.Contains(err.Error(), "invalid UTF-8") {
		t.Fatalf("EncodeRecording() error = %v", err)
	}
}

func TestWorldDecodersRetainStrictAndExactWireRefusal(t *testing.T) {
	model := newTestWorld(t, 0)
	initial := model.Snapshot()
	if _, err := model.Register(Request{Kind: "wait<", Resource: ResourceID{Adapter: "memory", Kind: "cell", Key: "key"}}); err != nil {
		t.Fatal(err)
	}
	final := model.Snapshot()
	encoded, err := EncodeReplayPlan(ReplayPlan{SchemaVersion: SchemaVersion, InitialDigest: initial.StateDigest, Transitions: final.Transitions, FinalDigest: final.StateDigest})
	if err != nil {
		t.Fatal(err)
	}
	invalidUTF8 := bytes.Replace(encoded, []byte("wait<"), []byte{'w', 'a', 'i', 't', 0xff}, 1)
	for name, malformed := range map[string][]byte{
		"unknown":          append([]byte(`{"unknown":true,`), encoded[1:]...),
		"duplicate":        bytes.Replace(encoded, []byte(`"schemaVersion":1`), []byte(`"schemaVersion":1,"schemaVersion":1`), 1),
		"trailing":         append(append([]byte(nil), encoded...), []byte(` {}`)...),
		"raw invalid UTF8": invalidUTF8,
		"alternate escape": bytes.Replace(encoded, []byte("wait<"), []byte(`wait\u003c`), 1),
		"case alias":       bytes.Replace(encoded, []byte(`"schemaVersion"`), []byte(`"SchemaVersion"`), 1),
		"whitespace":       append([]byte{' '}, encoded...),
		"lone surrogate":   bytes.Replace(encoded, []byte("wait<"), []byte(`wait\ud800`), 1),
	} {
		t.Run("replay "+name, func(t *testing.T) {
			if _, err := DecodeReplayPlan(malformed); err == nil {
				t.Fatal("DecodeReplayPlan() succeeded")
			}
		})
	}

	snapshot := newTestWorld(t, 0).Snapshot()
	initialJSON, err := EncodeSnapshot(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	frame := func(terminal []byte) []byte {
		result := append([]byte(nil), recordingMagic[:]...)
		result = binary.BigEndian.AppendUint64(result, uint64(len(initialJSON)))
		result = append(result, initialJSON...)
		result = binary.BigEndian.AppendUint64(result, uint64(len(initialJSON)))
		result = append(result, initialJSON...)
		result = binary.BigEndian.AppendUint64(result, uint64(len(terminal)))
		return append(result, terminal...)
	}
	validFrame := frame([]byte(`{"detail":"x","kind":"invalid-input"}`))
	malformedTerminal := append([]byte(nil), validFrame...)
	malformedTerminal[len(malformedTerminal)-1] ^= 1
	if _, err := DecodeRecording(malformedTerminal); err == nil || err.Error() != "decode World terminal: invalid character '|' after object key:value pair" {
		t.Fatalf("DecodeRecording() malformed terminal error = %v", err)
	}
	oversizedInitial := append([]byte(nil), validFrame...)
	binary.BigEndian.PutUint64(oversizedInitial[len(recordingMagic):], MaximumSnapshotJSONBytes+1)
	oversizedTerminal := append([]byte(nil), validFrame...)
	terminalLengthOffset := len(recordingMagic) + 8 + len(initialJSON) + 8 + len(initialJSON)
	binary.BigEndian.PutUint64(oversizedTerminal[terminalLengthOffset:], 1<<20+1)
	for name, malformed := range map[string][]byte{
		"unknown terminal":          frame([]byte(`{"detail":"x","extra":true,"kind":"invalid-input"}`)),
		"duplicate terminal":        frame([]byte(`{"detail":"x","detail":"y","kind":"invalid-input"}`)),
		"trailing terminal":         frame([]byte(`{"detail":"x","kind":"invalid-input"} {}`)),
		"raw invalid UTF8 terminal": frame([]byte("{\"detail\":\"\xff\",\"kind\":\"invalid-input\"}")),
		"alternate terminal order":  frame([]byte(`{"kind":"invalid-input","detail":"x"}`)),
		"wrong magic":               append([]byte("BADMAGIC"), validFrame[8:]...),
		"oversized initial length":  oversizedInitial,
		"oversized terminal length": oversizedTerminal,
		"truncated frame":           validFrame[:len(validFrame)-1],
	} {
		t.Run("recording "+name, func(t *testing.T) {
			if _, err := DecodeRecording(malformed); err == nil {
				t.Fatal("DecodeRecording() succeeded")
			}
		})
	}
}

func TestDecodeTransitionsPreservesExactWireAndRefusesNonCanonicalInput(t *testing.T) {
	model := newTestWorld(t, 0)
	if _, err := model.Register(Request{Kind: "wait<", Resource: ResourceID{Adapter: "memory", Kind: "cell", Key: "key"}}); err != nil {
		t.Fatal(err)
	}
	transitions := model.Snapshot().Transitions
	encoded, err := EncodeTransitions(transitions)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeTransitions(encoded, uint64(len(encoded)))
	if err != nil {
		t.Fatal(err)
	}
	reencoded, err := EncodeTransitions(decoded)
	if err != nil || !reflect.DeepEqual(decoded, transitions) || !bytes.Equal(reencoded, encoded) {
		t.Fatalf("transition round trip = %#v, %q, %v; want %#v, %q", decoded, reencoded, err, transitions, encoded)
	}
	if empty, err := DecodeTransitions(nil, 1); err != nil || len(empty) != 0 {
		t.Fatalf("DecodeTransitions(nil) = %#v, %v", empty, err)
	}

	invalidUTF8 := bytes.Replace(encoded, []byte("wait<"), []byte{'w', 'a', 'i', 't', 0xff}, 1)
	numericString := bytes.Replace(encoded, []byte(`"sequence":"0"`), []byte(`"sequence":"00"`), 1)
	if bytes.Equal(numericString, encoded) {
		numericString = bytes.Replace(encoded, []byte(`"sequence":"1"`), []byte(`"sequence":"01"`), 1)
	}
	if bytes.Equal(numericString, encoded) {
		t.Fatalf("transition fixture has no sequence field: %s", encoded)
	}
	for name, malformed := range map[string][]byte{
		"missing final newline": bytes.TrimSuffix(encoded, []byte{'\n'}),
		"empty line":            append(append([]byte(nil), encoded...), '\n'),
		"malformed":             append(append([]byte(nil), encoded[:len(encoded)-2]...), '\n'),
		"unknown":               append([]byte(`{"unknown":true,`), encoded[1:]...),
		"duplicate":             bytes.Replace(encoded, []byte(`"kind":"register"`), []byte(`"kind":"register","kind":"register"`), 1),
		"trailing":              append(append([]byte(nil), encoded[:len(encoded)-1]...), []byte(` {}`+"\n")...),
		"raw invalid UTF8":      invalidUTF8,
		"alternate escape":      bytes.Replace(encoded, []byte("wait<"), []byte(`wait\u003c`), 1),
		"whitespace":            append([]byte{' '}, encoded...),
		"numeric string":        numericString,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodeTransitions(malformed, uint64(len(malformed))); err == nil {
				t.Fatal("DecodeTransitions() succeeded")
			}
		})
	}
	if _, err := DecodeTransitions(encoded, uint64(len(encoded)-1)); err == nil {
		t.Fatal("DecodeTransitions() accepted an oversized payload")
	}
}
