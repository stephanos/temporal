package wasi

import (
	"encoding/json"
	"testing"
)

func testConfig() Config {
	return Config{Profile: StockProfile, Args: []string{"guest", "literal $arg"}, Environment: []string{"PWD=/workspace", "TZ=UTC", "ONLY=captured"}, WorkingDirectory: "/workspace", WritableDirectories: []string{"/workspace", "/tmp"}, EntropyKey: [32]byte{1, 2, 3}, Clock: ClockPolicy{EpochNanos: 946684800000000000, ReadStepNanos: 1000}, Limits: Limits{OutputBytes: 1 << 20, TranscriptBytes: 8 << 20, Calls: 10000, Descriptors: 64, Files: 1000, FilesystemBytes: 8 << 20, PendingEvents: 4096}}
}

func call(t *testing.T, environment *Environment, op, input string) Reply {
	t.Helper()
	reply, err := environment.Handle(Call{Type: "call", ID: environment.nextCall, Op: op, Input: json.RawMessage(input)})
	if err != nil {
		t.Fatal(err)
	}
	return reply
}

func output(t *testing.T, reply Reply, target any) {
	t.Helper()
	if reply.Errno != 0 || reply.Unsupported {
		t.Fatalf("reply = %#v", reply)
	}
	if err := json.Unmarshal(reply.Output, target); err != nil {
		t.Fatal(err)
	}
}
