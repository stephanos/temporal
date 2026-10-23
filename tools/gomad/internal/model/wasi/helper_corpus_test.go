package wasi

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
	"unicode/utf8"
)

// The helper transcript corpus is shared with wasmhost/tests/helper_corpus.rs,
// which owns regeneration (GOMAD_HELPER_CORPUS_REGENERATE=1), derived inputs,
// the diagnostic-build expected blocks and the cross-case relations. This
// test, on the default build,
// replays every case against the same helper, applies the same projection,
// requires the identical expected block, and decodes the raw started and
// result frames through the Go client's own frame validation.
const helperCorpusDirectory = "../../../rust/wasmhost/tests/testdata/helper_transcript"

const helperCorpusSchema = "gomad.helper-transcript-corpus/v1"

var helperCorpusKeys = map[string]bool{"schema": true, "intent": true, "replaces": true, "guest": true, "request": true, "inputs": true, "close_input": true, "replies": true, "reply_policy": true, "builds": true, "derived": true, "expected": true, "expected_builds": true}

type helperCorpusRun struct {
	frames   []any
	raw      [][]byte
	stderr   string
	exitCode int
}

func TestHelperCorpus(t *testing.T) {
	helper := helperPath(t)
	files, err := filepath.Glob(filepath.Join(helperCorpusDirectory, "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("empty helper transcript corpus")
	}
	sort.Strings(files)
	for _, file := range files {
		t.Run(strings.TrimSuffix(filepath.Base(file), ".json"), func(t *testing.T) {
			t.Parallel()
			data, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			kase, err := decodeCorpusJSON(data)
			if err != nil {
				t.Fatal(err)
			}
			fields, ok := kase.(map[string]any)
			if !ok || fields["schema"] != helperCorpusSchema {
				t.Fatalf("schema must be %s", helperCorpusSchema)
			}
			for key := range fields {
				if !helperCorpusKeys[key] {
					t.Fatalf("unknown case field %s", key)
				}
			}
			name := strings.TrimSuffix(filepath.Base(file), ".json")
			if err := helperCorpusResidualInput(name, fields); err != nil {
				t.Fatal(err)
			}
			expected, ok := fields["expected"]
			if !ok {
				t.Fatal("no expected block; regenerate with the Rust loader and review")
			}
			var actual []any
			for range 2 {
				run, err := runHelperCorpusCase(helper, fields)
				if err != nil {
					t.Fatal(err)
				}
				if fields["request"].(map[string]any)["type"] == "execute" {
					if err := decodeHelperCorpusFrames(run.raw); err != nil {
						t.Fatal(err)
					}
				}
				projected, err := helperCorpusExpected(run)
				if err != nil {
					t.Fatal(err)
				}
				if err := helperCorpusRefusalLiteral(strings.TrimSuffix(filepath.Base(file), ".json"), projected); err != nil {
					t.Fatal(err)
				}
				if err := helperCorpusResidualActual(name, projected); err != nil {
					t.Fatal(err)
				}
				actual = append(actual, projected)
			}
			if !reflect.DeepEqual(actual[0], actual[1]) {
				t.Fatal("output varies across two runs")
			}
			if !reflect.DeepEqual(expected, actual[0]) {
				want, _ := json.Marshal(expected)
				got, _ := json.Marshal(actual[0])
				t.Fatalf("differs from expected:\nwant %s\n got %s", want, got)
			}
		})
	}
}

func decodeCorpusJSON(data []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	if decoder.More() {
		return nil, errors.New("trailing JSON")
	}
	return value, nil
}

func helperCorpusOutputFrame(line []byte) (any, error) {
	frame, err := decodeCorpusJSON(line)
	if err != nil {
		return nil, err
	}
	fields, ok := frame.(map[string]any)
	if !ok || fields["type"] != "call" || fields["op"] != "fd_write" && fields["op"] != "fd_pwrite" {
		return frame, nil
	}
	number := func(value any) (string, error) {
		n, ok := value.(json.Number)
		if !ok {
			return "", errors.New("invalid unsigned write callback field")
		}
		value64, err := strconv.ParseUint(n.String(), 10, 64)
		if err != nil {
			return "", err
		}
		return strconv.FormatUint(value64, 10), nil
	}
	quote := func(value string) string {
		encoded, _ := helperCorpusMarshal(value)
		encoded = bytes.ReplaceAll(encoded, []byte(`\u2028`), []byte{0xe2, 0x80, 0xa8})
		encoded = bytes.ReplaceAll(encoded, []byte(`\u2029`), []byte{0xe2, 0x80, 0xa9})
		return string(encoded)
	}
	id, err := number(fields["id"])
	if err != nil {
		return nil, err
	}
	input, ok := fields["input"].(map[string]any)
	if !ok {
		return nil, errors.New("missing write callback input")
	}
	fd, err := number(input["fd"])
	if err != nil {
		return nil, err
	}
	if _, err := strconv.ParseUint(fd, 10, 32); err != nil {
		return nil, err
	}
	data, ok := input["data_base64"].(string)
	if !ok {
		return nil, errors.New("missing write callback bytes")
	}
	offset := ""
	if fields["op"] == "fd_pwrite" {
		value, err := number(input["offset"])
		if err != nil {
			return nil, err
		}
		offset = `,"offset":` + value
	}
	state := ""
	if value, present := fields["local_state"]; present {
		local, ok := value.(map[string]any)
		if !ok {
			return nil, errors.New("invalid callback local_state")
		}
		var parts []string
		for _, key := range []string{"advances", "now", "transcript_bytes"} {
			value, err := number(local[key])
			if err != nil {
				return nil, err
			}
			parts = append(parts, quote(key)+":"+value)
		}
		if offered, present := local["offered_output_bytes"]; present {
			value, err := number(offered)
			if err != nil {
				return nil, err
			}
			parts = append(parts, `"offered_output_bytes":`+value)
		}
		state = `,"local_state":{` + strings.Join(parts, ",") + `}`
	}
	instance := ""
	if value, present := fields["instance"]; present {
		name, ok := value.(string)
		if !ok {
			return nil, errors.New("invalid callback instance")
		}
		instance = `"instance":` + quote(name) + ","
	}
	instance, err = helperCorpusClusterIncarnationPrefix(fields, instance)
	if err != nil {
		return nil, err
	}
	expected := fmt.Sprintf(`{%s"id":%s,"input":{"data_base64":%s,"fd":%s%s}%s,"op":%s,"type":"call"}`, instance, id, quote(data), fd, offset, state, quote(fields["op"].(string)))
	if !bytes.Equal(line, []byte(expected)) {
		return nil, errors.New("write callback raw literal key order or bytes differ")
	}
	return frame, nil
}

func TestHelperCorpusWriteRawLinesPreserveLiteralKeyOrderAndBytes(t *testing.T) {
	for _, literal := range []string{
		`{"id":1,"input":{"data_base64":"AP8=","fd":13},"local_state":{"advances":0,"now":1,"transcript_bytes":0},"op":"fd_write","type":"call"}`,
		`{"id":2,"input":{"data_base64":"","fd":4294967295,"offset":18446744073709551615},"op":"fd_pwrite","type":"call"}`,
	} {
		if _, err := helperCorpusOutputFrame([]byte(literal)); err != nil {
			t.Fatal(err)
		}
		reordered := strings.ReplaceAll(strings.ReplaceAll(literal, `{"id":`, `{"type":"call","id":`), `,"type":"call"}`, `}`)
		original, err := decodeCorpusJSON([]byte(literal))
		if err != nil {
			t.Fatal(err)
		}
		changed, err := decodeCorpusJSON([]byte(reordered))
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(original, changed) {
			t.Fatal("mutation changed parsed fields")
		}
		for _, wrong := range []string{reordered, strings.Replace(literal, ":", ": ", 1)} {
			if _, err := helperCorpusOutputFrame([]byte(wrong)); err == nil {
				t.Fatalf("raw callback byte/order drift accepted: %s", wrong)
			}
		}
	}
	for _, wrong := range []string{
		`{"id":1,"input":{"data_base64":"AP\u0038=","fd":13},"op":"fd_write","type":"call"}`,
		`{"id":1,"id":1,"input":{"data_base64":"","fd":13},"op":"fd_write","type":"call"}`,
	} {
		if _, err := helperCorpusOutputFrame([]byte(wrong)); err == nil {
			t.Fatalf("raw escape/duplicate drift accepted: %s", wrong)
		}
	}
}

func helperCorpusGuest(guest any) ([]byte, error) {
	fields, ok := guest.(map[string]any)
	if !ok || len(fields) != 1 {
		return nil, errors.New("guest needs exactly one of wat, path, wat_parts")
	}
	if text, ok := fields["wat"].(string); ok {
		return []byte(text), nil
	}
	if name, ok := fields["path"].(string); ok {
		if filepath.IsAbs(name) || strings.Contains(name, "..") {
			return nil, fmt.Errorf("guest.path %s escapes the corpus directory", name)
		}
		return os.ReadFile(filepath.Join(helperCorpusDirectory, name))
	}
	parts, ok := fields["wat_parts"].([]any)
	if !ok {
		return nil, errors.New("guest needs wat, path or wat_parts")
	}
	var text strings.Builder
	for _, part := range parts {
		switch part := part.(type) {
		case string:
			text.WriteString(part)
		case map[string]any:
			count, err := part["repeat"].(json.Number).Int64()
			piece, ok := part["text"].(string)
			if err != nil || !ok || len(part) != 2 {
				return nil, fmt.Errorf("invalid guest part %v", part)
			}
			text.WriteString(strings.Repeat(piece, int(count)))
		default:
			return nil, fmt.Errorf("invalid guest part %v", part)
		}
	}
	return []byte(text.String()), nil
}

// helperCorpusGuestPathBytes is the byte length of every guest path. Helper
// refusals quote request columns, so the path's length (not its text) would
// otherwise leak into output; the Rust loader builds the same shape.
const helperCorpusGuestPathBytes = 160

var helperCorpusNext atomic.Uint64

func helperCorpusGuestPath() (string, error) {
	unique := uint64(os.Getpid())*1_000_000 + helperCorpusNext.Add(1) - 1
	base := filepath.Join(os.TempDir(), fmt.Sprintf("gomad-helper-corpus-%020d-", unique))
	padding := helperCorpusGuestPathBytes - len(base) - len("/guest.wat")
	if padding < 0 {
		return "", errors.New("temp directory too long for the fixed-length guest path")
	}
	directory := base + strings.Repeat("x", padding)
	if err := os.Mkdir(directory, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(directory, "guest.wat"), nil
}

func helperCorpusSubstitute(value any, guestPath, guestSHA256 string, trigger any) (any, error) {
	switch value := value.(type) {
	case string:
		if !strings.HasPrefix(value, "$") {
			return value, nil
		}
		switch value {
		case "$guest_path":
			return guestPath, nil
		case "$guest_sha256":
			return guestSHA256, nil
		}
		path, ok := strings.CutPrefix(value, "$trigger.")
		if !ok {
			return nil, fmt.Errorf("unknown placeholder %s", value)
		}
		if trigger == nil {
			return nil, fmt.Errorf("%s outside a reply", value)
		}
		found := trigger
		for _, key := range strings.Split(path, ".") {
			object, ok := found.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("%s missing from trigger", value)
			}
			if found, ok = object[key]; !ok {
				return nil, fmt.Errorf("%s missing from trigger", value)
			}
		}
		return found, nil
	case []any:
		out := make([]any, len(value))
		for index, item := range value {
			resolved, err := helperCorpusSubstitute(item, guestPath, guestSHA256, trigger)
			if err != nil {
				return nil, err
			}
			out[index] = resolved
		}
		return out, nil
	case map[string]any:
		out := make(map[string]any, len(value))
		for key, item := range value {
			resolved, err := helperCorpusSubstitute(item, guestPath, guestSHA256, trigger)
			if err != nil {
				return nil, err
			}
			out[key] = resolved
		}
		return out, nil
	}
	return value, nil
}

func helperCorpusMatches(pattern, frame any) bool {
	want, ok := pattern.(map[string]any)
	if !ok {
		return reflect.DeepEqual(pattern, frame)
	}
	have, ok := frame.(map[string]any)
	if !ok {
		return false
	}
	for key, value := range want {
		got, found := have[key]
		if !found || !helperCorpusMatches(value, got) {
			return false
		}
	}
	return true
}

func runHelperCorpusCase(helper string, kase map[string]any) (_ *helperCorpusRun, retErr error) {
	guest, err := helperCorpusGuest(kase["guest"])
	if err != nil {
		return nil, err
	}
	guestPath, err := helperCorpusGuestPath()
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(filepath.Dir(guestPath)) }()
	if err := os.WriteFile(guestPath, guest, 0o600); err != nil {
		return nil, err
	}
	kase, err = helperCorpusClusterAssets(kase, guestPath)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(guest)
	guestSHA256 := hex.EncodeToString(digest[:])
	replies, _ := kase["replies"].([]any)
	deny := false
	switch policy := kase["reply_policy"]; policy {
	case nil, "fail":
	case "deny":
		deny = true
	default:
		return nil, fmt.Errorf("unknown reply_policy %v", policy)
	}
	for _, frames := range []any{kase["request"], kase["inputs"], kase["replies"]} {
		if _, err := helperCorpusWireMarshal(frames); err != nil {
			return nil, err
		}
	}
	command := exec.Command(helper)
	stdin, err := command.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := command.StdoutPipe()
	if err != nil {
		return nil, err
	}
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		return nil, err
	}
	defer func() {
		if command.ProcessState == nil {
			retErr = errors.Join(retErr, command.Process.Kill(), command.Wait())
		}
	}()
	timer := time.AfterFunc(2*time.Minute, func() { _ = command.Process.Kill() })
	defer timer.Stop()
	open := true
	// A helper that already refused may have closed its input; the refusal is
	// in its output, so a broken pipe is not a corpus failure.
	write := func(frame any, trigger any) error {
		resolved, err := helperCorpusSubstitute(frame, guestPath, guestSHA256, trigger)
		if err != nil {
			return err
		}
		line, err := helperCorpusInput(resolved)
		if err != nil {
			return err
		}
		if open {
			if _, err := stdin.Write(line); err != nil {
				open = false
			}
		}
		return nil
	}
	closeInput := func() {
		if open {
			open = false
			_ = stdin.Close()
		}
	}
	defer closeInput()
	if err := write(kase["request"], nil); err != nil {
		return nil, err
	}
	inputs, _ := kase["inputs"].([]any)
	for _, frame := range inputs {
		if err := write(frame, nil); err != nil {
			return nil, err
		}
	}
	// true closes input after the initial frames, "after_replies" once the
	// last scripted reply is sent; otherwise input stays open to the end.
	closeAfterReplies := false
	switch when := kase["close_input"]; when {
	case nil, false:
	case true:
		closeInput()
	case "after_replies":
		closeAfterReplies = true
		if len(replies) == 0 {
			closeInput()
		}
	default:
		return nil, fmt.Errorf("invalid close_input %v", when)
	}
	run := &helperCorpusRun{}
	var failure error
	cursor := 0
	reader := bufio.NewReaderSize(stdout, 1<<20)
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) == 0 && errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		frame, err := helperCorpusOutputFrame(bytes.TrimSuffix(line, []byte("\n")))
		if err != nil {
			return nil, fmt.Errorf("helper emitted non-JSON %q: %w", line, err)
		}
		run.frames = append(run.frames, frame)
		run.raw = append(run.raw, bytes.TrimSuffix(line, []byte("\n")))
		if failure != nil {
			continue
		}
		kind := frame.(map[string]any)["type"]
		if cursor < len(replies) && helperCorpusMatches(replies[cursor].(map[string]any)["after"], frame) {
			send, _ := replies[cursor].(map[string]any)["send"].([]any)
			for _, item := range send {
				if err := write(item, frame); err != nil {
					return nil, err
				}
			}
			cursor++
			if closeAfterReplies && cursor == len(replies) {
				closeInput()
			}
		} else if kind == "call" || kind == "sync" {
			if deny {
				denial := map[string]any{"type": "reply", "id": frame.(map[string]any)["id"], "errno": 0, "output": map[string]any{}, "unsupported": true, "message": "not modeled"}
				if err := write(denial, nil); err != nil {
					return nil, err
				}
			} else {
				failure = fmt.Errorf("unscripted helper request %s", line)
				closeInput()
			}
		}
	}
	closeInput()
	err = command.Wait()
	if err := helperProfileStderr(stderr.Bytes()); err != nil {
		return nil, err
	}
	var exit *exec.ExitError
	switch {
	case err == nil:
	case errors.As(err, &exit) && exit.Exited():
	default:
		return nil, fmt.Errorf("helper did not exit normally: %w; stderr: %s", err, stderr.String())
	}
	if status, ok := command.ProcessState.Sys().(syscall.WaitStatus); ok && status.Signaled() {
		return nil, fmt.Errorf("helper killed by %v", status.Signal())
	}
	if failure != nil {
		return nil, failure
	}
	if cursor != len(replies) {
		return nil, fmt.Errorf("scripted reply %d never triggered", cursor)
	}
	run.stderr, run.exitCode = stderr.String(), command.ProcessState.ExitCode()
	return run, nil
}

func TestHelperCorpusInputFrames(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		frame   map[string]any
		want    []byte
		invalid bool
	}{
		{name: "raw LF and binary", frame: map[string]any{"raw_base64": "AAoA"}, want: []byte{0, '\n', 0}},
		{name: "JSON line", frame: map[string]any{"type": "authorize"}, want: []byte("{\"type\":\"authorize\"}\n")},
		{name: "invalid base64", frame: map[string]any{"raw_base64": "!"}, invalid: true},
		{name: "nonstring", frame: map[string]any{"raw_base64": 3}, invalid: true},
		{name: "extra field", frame: map[string]any{"raw_base64": "AAoA", "type": "authorize"}, invalid: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := helperCorpusInput(test.frame)
			if test.invalid {
				if err == nil {
					t.Fatal("accepted malformed raw corpus input")
				}
				return
			}
			if err != nil || !bytes.Equal(got, test.want) {
				t.Fatalf("corpus wire=%q want=%q err=%v", got, test.want, err)
			}
		})
	}
}

func helperCorpusInput(frame any) ([]byte, error) {
	if fields, ok := frame.(map[string]any); ok {
		if raw, found := fields["raw_base64"]; found {
			encoded, ok := raw.(string)
			if !ok || len(fields) != 1 {
				return nil, errors.New("raw_base64 input must be a single string field")
			}
			return base64.StdEncoding.DecodeString(encoded)
		}
	}
	line, err := helperCorpusLine(frame)
	if err != nil {
		return nil, err
	}
	return append(line, '\n'), nil
}

// helperCorpusLine encodes a sent frame; cluster frames carry their instance
// tag first, as the helper's transport routes by that prefix.
func helperCorpusLine(frame any) ([]byte, error) {
	fields, ok := frame.(map[string]any)
	instance, tagged := fields["instance"]
	if !ok || !tagged {
		return helperCorpusWireMarshal(frame)
	}
	rest := make(map[string]any, len(fields))
	for key, value := range fields {
		if key != "instance" {
			rest[key] = value
		}
	}
	tag, err := helperCorpusMarshal(instance)
	if err != nil {
		return nil, err
	}
	body, err := helperCorpusWireMarshal(rest)
	if err != nil {
		return nil, err
	}
	if len(rest) == 0 {
		return []byte(`{"instance":` + string(tag) + `}`), nil
	}
	return []byte(`{"instance":` + string(tag) + `,` + string(body[1:])), nil
}

func helperCorpusWireMarshal(value any) ([]byte, error) {
	switch fields := value.(type) {
	case map[string]any:
		if raw, found := fields["raw_json"]; found {
			text, ok := raw.(string)
			trimmed := strings.TrimSpace(text)
			if !ok || len(fields) != 1 || strings.ContainsAny(text, "\r\n") || !strings.HasPrefix(trimmed, "{") || !json.Valid([]byte(text)) {
				return nil, errors.New("raw_json needs one JSON object without line breaks or sibling fields")
			}
			return []byte(text), nil
		}
		keys := make([]string, 0, len(fields))
		for key := range fields {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		var output bytes.Buffer
		output.WriteByte('{')
		for index, key := range keys {
			if index != 0 {
				output.WriteByte(',')
			}
			name, err := helperCorpusMarshal(key)
			if err != nil {
				return nil, err
			}
			item, err := helperCorpusWireMarshal(fields[key])
			if err != nil {
				return nil, err
			}
			output.Write(name)
			output.WriteByte(':')
			output.Write(item)
		}
		output.WriteByte('}')
		return output.Bytes(), nil
	case []any:
		var output bytes.Buffer
		output.WriteByte('[')
		for index, field := range fields {
			if index != 0 {
				output.WriteByte(',')
			}
			item, err := helperCorpusWireMarshal(field)
			if err != nil {
				return nil, err
			}
			output.Write(item)
		}
		output.WriteByte(']')
		return output.Bytes(), nil
	default:
		return helperCorpusMarshal(value)
	}
}

// helperCorpusMarshal encodes like serde_json: sorted keys, no HTML escaping.
func helperCorpusMarshal(value any) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(buffer.Bytes(), []byte("\n")), nil
}

// helperCorpusMetrics mirrors the Rust projection of host telemetry (metrics,
// cluster scheduler): durations keep only whether they are zero, process
// counters only that they were measured.
func helperCorpusMetrics(metrics any) any {
	fields, ok := metrics.(map[string]any)
	if !ok {
		return helperCorpusProject(metrics)
	}
	out := make(map[string]any, len(fields))
	for key, value := range fields {
		number, isNumber := value.(json.Number)
		unsigned := false
		if isNumber {
			_, err := strconv.ParseUint(number.String(), 10, 64)
			unsigned = err == nil
		}
		switch {
		case unsigned && (key == "process_user_nanos" || key == "process_system_nanos" || key == "peak_rss_bytes"):
			out[key] = "measured"
		case unsigned && strings.HasSuffix(key, "_nanos"):
			if number.String() == "0" {
				out[key] = json.Number("0")
			} else {
				out[key] = "nonzero"
			}
		default:
			out[key] = helperCorpusProject(value)
		}
	}
	return out
}

func helperCorpusProject(value any) any {
	switch value := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(value))
		for key, item := range value {
			switch engine, isObject := item.(map[string]any); {
			case key == "metrics" || key == "scheduler":
				out[key] = helperCorpusMetrics(item)
			case key == "engine" && isObject:
				projected := helperCorpusProject(engine).(map[string]any)
				for _, host := range []string{"host_os", "host_arch"} {
					if _, ok := projected[host].(string); ok {
						projected[host] = "$" + host
					}
				}
				out[key] = projected
			default:
				out[key] = helperCorpusProject(item)
			}
		}
		return out
	case []any:
		out := make([]any, len(value))
		for index, item := range value {
			out[index] = helperCorpusProject(item)
		}
		return out
	}
	return value
}

func helperCorpusExpected(run *helperCorpusRun) (any, error) {
	streams := map[string][]byte{}
	frames := make([]any, len(run.frames))
	termination := any(nil)
	for index, frame := range run.frames {
		fields := frame.(map[string]any)
		frames[index] = helperCorpusProject(frame)
		termination = fields["termination"]
		if fields["type"] != "evidence" {
			continue
		}
		stream, ok := fields["stream"].(string)
		encoded, encodedOK := fields["data_base64"].(string)
		if !ok || !encodedOK {
			return nil, errors.New("evidence without stream or data")
		}
		if instance, ok := fields["instance"].(string); ok {
			stream = instance + "/" + stream
		}
		data, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return nil, err
		}
		streams[stream] = append(streams[stream], data...)
	}
	evidence := map[string]any{}
	for name, data := range streams {
		digest := sha256.Sum256(data)
		summary := map[string]any{"bytes": json.Number(fmt.Sprint(len(data))), "sha256": hex.EncodeToString(digest[:])}
		// Text streams stay readable, as the Rust loader records them.
		if stream := name[strings.LastIndex(name, "/")+1:]; (stream == "storage" || stream == "stdout" || stream == "stderr") && utf8.Valid(data) {
			summary["text"] = string(data)
		}
		evidence[name] = summary
	}
	return map[string]any{
		"exit_code":   json.Number(fmt.Sprint(run.exitCode)),
		"termination": termination,
		"frames":      frames,
		"evidence":    evidence,
		"stderr":      run.stderr,
	}, nil
}

// decodeHelperCorpusFrames decodes started and result frames with the key
// sets and strict types the Go helper client uses.
func decodeHelperCorpusFrames(raw [][]byte) error {
	for _, line := range raw {
		frame, err := newValidatedFrame(line)
		if err != nil {
			return fmt.Errorf("%s: %w", line, err)
		}
		var kind string
		if err := json.Unmarshal(frame.fields["type"], &kind); err != nil {
			return err
		}
		switch kind {
		case "started":
			var started Started
			if err := frame.decode(&started, "schema", "type", "engine", "module_sha256", "imports", "exports", "initial_memory_pages"); err != nil {
				return fmt.Errorf("Go client refuses started %s: %w", line, err)
			}
		case "result":
			keys := []string{"type", "termination", "message", "fuel_remaining", "peak_memory_bytes", "output_bytes"}
			for _, key := range []string{"exit_code", "local_model", "divergence", "metrics", "error", "check"} {
				if _, found := frame.fields[key]; found {
					keys = append(keys, key)
				}
			}
			var result resultFrame
			if err := frame.decode(&result, keys...); err != nil {
				return fmt.Errorf("Go client refuses result %s: %w", line, err)
			}
		}
	}
	return nil
}
