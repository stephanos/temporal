package canonicaljson_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomad3/world"
)

type streamEmbedded struct {
	Value string `json:"value"`
}

type streamMarshaler string

func (v streamMarshaler) MarshalJSON() ([]byte, error) {
	return []byte(`{"z":2,"a":"<>&雪"}`), nil
}

type streamPointerMarshaler struct{ Value string }

func (v *streamPointerMarshaler) MarshalJSON() ([]byte, error) {
	return json.Marshal("pointer:" + v.Value)
}

type streamTextKey struct{ Value string }

func (v streamTextKey) MarshalText() ([]byte, error) {
	return []byte(v.Value), nil
}

func TestWriteCanonicalJSONMatchesLegacy(t *testing.T) {
	text := "<>&雪😀\u2028\u2029\x00\b\f\n\r\t\"\\"
	duplicate := reflect.New(reflect.StructOf([]reflect.StructField{
		{Name: "A", Type: reflect.TypeFor[string](), Tag: `json:"same"`},
		{Name: "B", Type: reflect.TypeFor[int](), Tag: `json:"same"`},
	})).Elem().Interface()
	invalidTag := reflect.New(reflect.StructOf([]reflect.StructField{
		{Name: "Value", Type: reflect.TypeFor[string](), Tag: `json:"bad\\name"`},
	})).Elem()
	invalidTag.Field(0).SetString(text)
	for _, tc := range []struct {
		name  string
		value any
	}{
		{"nil", nil},
		{"strings", []string{text, strings.Repeat(text, 1000)}},
		{"integer-limits", []any{int64(math.MinInt64), uint64(math.MaxUint64), float64(2), math.Copysign(0, -1), json.Number("-0"), json.Number("")}},
		{"bytes", [][]byte{nil, {}, {0, 1, 255}, bytes.Repeat([]byte{1, 2, 3, 4}, 5000)}},
		{"array", [2]string{"a", text}},
		{"map", map[string]any{"雪": []int{1, 2}, "a": (*int)(nil), "z": map[string]string{}}},
		{"integer-map", map[int]string{10: "ten", 2: "two", -1: "minus"}},
		{"nil-unsupported-map", (map[bool]string)(nil)},
		{"fields", struct {
			Z       string `json:"z,omitempty"`
			A       string `json:"a"`
			Zero    int    `json:"zero,omitempty"`
			Pointer *int   `json:"pointer,omitempty"`
			Bytes   []byte `json:"bytes,omitempty"`
			Hidden  string `json:"-"`
			hidden  string
		}{A: text}},
		{"quoted-fields", struct {
			N int    `json:"n,string"`
			S string `json:"s,string"`
			B bool   `json:"b,string"`
		}{12, text, true}},
		{"embedded-fields", struct {
			streamEmbedded
			Z string
		}{streamEmbedded{"ok"}, "last"}},
		{"duplicate-fields", duplicate},
		{"omitzero", struct {
			A int    `json:"a,omitzero"`
			B string `json:"b,omitzero"`
		}{B: "present"}},
		{"invalid-tag-name", invalidTag.Interface()},
		{"nil-container", []any{([]byte)(nil), ([]int)(nil), (map[string]int)(nil), (*streamPointerMarshaler)(nil)}},
		{"text-map", map[streamTextKey]string{{Value: "雪"}: "unicode", {Value: "a"}: "first"}},
		{"addressable-pointer-hook", &struct{ Value streamPointerMarshaler }{streamPointerMarshaler{text}}},
		{"unaddressable-pointer-hook", struct{ Value streamPointerMarshaler }{streamPointerMarshaler{text}}},
		{"pointer-hook-slice", []streamPointerMarshaler{{text}}},
		{"unaddressable-map-value", map[string]streamPointerMarshaler{"key": {text}}},
		{"raw-message", json.RawMessage(` {"z":2,"a":"\u003c雪"} `)},
		{"custom-json", streamMarshaler("ignored")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			want, err := canonicaljson.CanonicalJSON(tc.value)
			if err != nil {
				t.Fatal(err)
			}
			var got bytes.Buffer
			if err := canonicaljson.WriteCanonicalJSON(&got, tc.value); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got.Bytes(), want) {
				t.Fatalf("stream=%q legacy=%q", got.Bytes(), want)
			}
		})
	}
	var literal bytes.Buffer
	if err := canonicaljson.WriteCanonicalJSON(&literal, map[string]any{"z": []byte{1, 2}, "a": text}); err != nil {
		t.Fatal(err)
	}
	if got, want := literal.String(), "{\"a\":\"<>&雪😀\\u2028\\u2029\\u0000\\b\\f\\n\\r\\t\\\"\\\\\",\"z\":\"AQI=\"}"; got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestWriteCanonicalJSONWorldSnapshot(t *testing.T) {
	model, err := world.New(world.Config{Seed: 7, Limits: world.Limits{MaxRequests: 20, MaxEvents: 20, MaxQueuedEvents: 20, MaxTransitions: 100, MaxPayloadBytes: 1 << 20, MaxStringBytes: 4096}})
	if err != nil {
		t.Fatal(err)
	}
	id, err := model.Register(world.Request{Kind: "雪<>&", Resource: world.ResourceID{Adapter: "test", Kind: "message", Key: "😀"}, Payload: []byte{0, 255}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := model.Ready(world.Readiness{RequestID: id, At: world.InitialTime + 1, Kind: "ready", Payload: []byte("payload")}); err != nil {
		t.Fatal(err)
	}
	if _, err := model.Quiesce(); err != nil {
		t.Fatal(err)
	}
	snapshot := model.Snapshot()
	want, err := canonicaljson.CanonicalJSON(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	var got bytes.Buffer
	if err := canonicaljson.WriteCanonicalJSON(&got, snapshot); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got.Bytes(), want) {
		t.Fatalf("stream=%s legacy=%s", got.Bytes(), want)
	}
}

type streamFailWriter struct {
	remaining int
	err       error
}

func (w *streamFailWriter) Write(p []byte) (int, error) {
	if len(p) > w.remaining {
		n := w.remaining
		w.remaining = 0
		return n, w.err
	}
	w.remaining -= len(p)
	return len(p), nil
}

func TestWriteCanonicalJSONWriterFailures(t *testing.T) {
	sentinel := errors.New("write failed")
	for _, offset := range []int{0, 1, 4, 10, 100} {
		if err := canonicaljson.WriteCanonicalJSON(&streamFailWriter{offset, sentinel}, []string{strings.Repeat("雪", 1000)}); !errors.Is(err, sentinel) {
			t.Fatalf("offset %d: %v", offset, err)
		}
		if err := canonicaljson.WriteCanonicalJSON(&streamFailWriter{offset, nil}, []byte(strings.Repeat("bytes", 1000))); !errors.Is(err, io.ErrShortWrite) {
			t.Fatalf("short write offset %d: %v", offset, err)
		}
	}
}

func TestWriteCanonicalJSONRejectsInvalidValues(t *testing.T) {
	cycle := map[string]any{}
	cycle["self"] = cycle
	for _, value := range []any{1.5, json.Number("1e2"), json.Number("18446744073709551616"), json.RawMessage(`{"a":1,"a":2}`), "\xff", struct {
		Hidden string `json:"-"`
	}{"\xff"}, cycle, make(chan int)} {
		if err := canonicaljson.WriteCanonicalJSON(io.Discard, value); err == nil {
			legacy, legacyErr := canonicaljson.CanonicalJSON(value)
			t.Fatalf("accepted %T (legacy=%s, %v)", value, legacy, legacyErr)
		}
	}
}

type streamErrorMarshaler struct {
	Text  string
	Calls *int
	Err   error
}

func (v *streamErrorMarshaler) MarshalJSON() ([]byte, error) {
	*v.Calls++
	return nil, v.Err
}

func TestWriteCanonicalJSONPreservesHookErrorsAndValidationPrecedence(t *testing.T) {
	sentinel := errors.New("marshal failed")
	for _, text := range []string{"valid", "\xff"} {
		calls := 0
		value := &streamErrorMarshaler{Text: text, Calls: &calls, Err: sentinel}
		err := canonicaljson.WriteCanonicalJSON(io.Discard, value)
		if text == "valid" {
			var cause *json.MarshalerError
			if !errors.Is(err, sentinel) || !errors.As(err, &cause) || calls != 1 {
				t.Fatalf("err=%v calls=%d", err, calls)
			}
		} else if err == nil || err.Error() != "JSON string is not valid UTF-8" || calls != 0 {
			t.Fatalf("err=%v calls=%d", err, calls)
		}
	}
}

func TestWriteCanonicalJSONSharedAliasesAndCycles(t *testing.T) {
	text := "valid"
	sequence := []any{&text}
	sequence = append(sequence, sequence[:0])
	value := []any{&text, &text, sequence, sequence}
	want, err := canonicaljson.CanonicalJSON(value)
	if err != nil {
		t.Fatal(err)
	}
	var got bytes.Buffer
	if err := canonicaljson.WriteCanonicalJSON(&got, value); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got.Bytes(), want) {
		t.Fatalf("got=%s want=%s", got.Bytes(), want)
	}
	type node struct{ Next *node }
	cycle := &node{}
	cycle.Next = cycle
	cycleSlice := make([]any, 1)
	cycleSlice[0] = cycleSlice
	for _, value := range []any{cycle, cycleSlice} {
		var cause *json.UnsupportedValueError
		if err := canonicaljson.WriteCanonicalJSON(io.Discard, value); !errors.As(err, &cause) {
			t.Fatalf("cycle %T: %v", value, err)
		}
	}
	storage := []string{"valid", "\xff"}
	if err := canonicaljson.WriteCanonicalJSON(io.Discard, [2][]string{storage[:1], storage}); err == nil {
		t.Fatal("accepted invalid UTF-8 through shared alias")
	}
}

type streamBoundWriter struct {
	total   int
	largest int
}

func (w *streamBoundWriter) Write(p []byte) (int, error) {
	w.total += len(p)
	if len(p) > w.largest {
		w.largest = len(p)
	}
	return len(p), nil
}

func TestWriteCanonicalJSONBoundsPayloadChunks(t *testing.T) {
	w := &streamBoundWriter{}
	if err := canonicaljson.WriteCanonicalJSON(w, struct {
		Text  string
		Bytes []byte
	}{strings.Repeat("雪<>&", 20000), bytes.Repeat([]byte{255}, 200000)}); err != nil {
		t.Fatal(err)
	}
	if w.total < 300000 || w.largest > 4096 {
		t.Fatalf("total=%d largest=%d", w.total, w.largest)
	}
}

func TestImplementationSHA256BindsAllProductionSources(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".go") && !strings.HasSuffix(entry.Name(), "_test.go") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	var inventory strings.Builder
	inventory.WriteString("gomad3-canonicaljson-implementation/v1\n")
	for _, name := range names {
		source, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(source)
		inventory.WriteString(name + ":" + hex.EncodeToString(digest[:]) + "\n")
	}
	digest := sha256.Sum256([]byte(inventory.String()))
	if got, want := canonicaljson.ImplementationSHA256(), "sha256:"+hex.EncodeToString(digest[:]); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
