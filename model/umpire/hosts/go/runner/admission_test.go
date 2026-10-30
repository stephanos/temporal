package runner

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"
	modelv1 "umpire.local/model/gen/umpire/model/v1"
	runv1 "umpire.local/model/gen/umpire/run/v1"
)

func TestResetContractAdmissionPrecedesDriver(t *testing.T) {
	for _, tc := range []struct {
		name     string
		contract byte
		want     modelv1.RefusalCode
	}{
		{"missing", 0, modelv1.RefusalCode_REFUSAL_CODE_MALFORMED_REQUEST},
		{"unknown", 99, modelv1.RefusalCode_REFUSAL_CODE_UNSUPPORTED},
		{"current", 1, modelv1.RefusalCode_REFUSAL_CODE_UNSPECIFIED},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			wasm := resetContractGuest(tc.contract)
			r := New(wasm, func(context.Context) (Driver, error) {
				calls++
				return nil, errors.New("admitted driver")
			})
			run, err := r.Run(context.Background(), ModeExplore, Options{Seed: 1})
			if err != nil {
				t.Fatal(err)
			}
			if tc.contract == 1 {
				if calls != 1 || run.GetEvents()[0].GetReset_().GetResponse() == nil {
					t.Fatalf("actual Reset not recorded before driver: calls=%d run=%v", calls, run)
				}
				return
			}
			if calls != 0 || len(run.GetEvents()) != 1 || run.GetEvents()[0].GetRefusal().GetRefusal().GetCode() != tc.want || run.GetVerdict().GetRefused() == nil {
				t.Fatalf("incompatible actual Reset reached Driver: calls=%d run=%v", calls, run)
			}
		})
	}
}

func TestRunnerOwnsLoadedGuestBytes(t *testing.T) {
	wasm := resetContractGuest(1)
	original := append([]byte(nil), wasm...)
	sum := sha256.Sum256(original)
	r := New(wasm, func(context.Context) (Driver, error) { return nil, errors.New("admitted driver") })
	clear(wasm)
	run, err := r.Run(context.Background(), ModeExplore, Options{Seed: 1})
	if err != nil {
		t.Fatalf("caller mutation changed loaded guest: %v", err)
	}
	if run.GetModelHash() != hex.EncodeToString(sum[:]) || r.ModelHash() != run.GetModelHash() || len(run.GetEvents()) != 2 || run.GetEvents()[0].GetReset_().GetResponse() == nil {
		t.Fatalf("loaded guest/hash diverged after caller mutation: %v", run)
	}
	wire, err := proto.Marshal(run.GetEvents()[0].GetReset_().GetResponse())
	if err != nil || string(wire) != string([]byte{0x10, 1}) {
		t.Fatalf("actual current Reset response = %x, %v", wire, err)
	}
}

// This guest only returns an actual Reset response through the ordinary ABI.
func resetContractGuest(contract byte) []byte {
	payload := []byte{0x22, 0}
	if contract != 0 {
		payload = []byte{0x22, 2, 0x10, contract}
	}
	wasm := []byte{0, 97, 115, 109, 1, 0, 0, 0}
	u := func(n uint64) []byte { return binary.AppendUvarint(nil, n) }
	section := func(id byte, body []byte) {
		wasm = append(wasm, id)
		wasm = append(wasm, u(uint64(len(body)))...)
		wasm = append(wasm, body...)
	}
	section(1, []byte{3, 0x60, 1, 0x7f, 1, 0x7f, 0x60, 2, 0x7f, 0x7f, 0, 0x60, 3, 0x7f, 0x7f, 0x7f, 1, 0x7e})
	section(3, []byte{3, 0, 1, 2})
	section(5, []byte{1, 0, 1})
	exports := []byte{4}
	for i, name := range []string{"umpire_buffer_alloc", "umpire_buffer_release", "umpire_call", "memory"} {
		exports = append(exports, byte(len(name)))
		exports = append(exports, name...)
		if i == 3 {
			exports = append(exports, 2, 0)
		} else {
			exports = append(exports, 0, byte(i))
		}
	}
	section(7, exports)
	// The packed response pointer (2048) and length, encoded as signed LEB128.
	section(10, []byte{3, 5, 0, 0x41, 0x80, 8, 0x0b, 2, 0, 0x0b, 9, 0, 0x42, 0x80, 0x90, 0x80, 0x80, 0x80 | byte(len(payload)<<4), 0, 0x0b})
	section(11, append([]byte{1, 0, 0x41, 0x80, 0x10, 0x0b, byte(len(payload))}, payload...))
	return wasm
}

func TestCurrentRecordingAdmission(t *testing.T) {
	wasm := resetContractGuest(1)
	calls := 0
	r := New(wasm, func(context.Context) (Driver, error) {
		calls++
		return nil, errors.New("admitted driver")
	})
	recorded, err := r.Run(t.Context(), ModeExplore, Options{Seed: 1})
	if err != nil {
		t.Fatal(err)
	}
	if recorded.GetProtocolVersion() != 2 {
		t.Fatalf("recording format = %d, want 2", recorded.GetProtocolVersion())
	}
	for _, tc := range []struct {
		name       string
		alter      func(*Run)
		wantReason string
	}{
		{"old format", func(run *Run) { run.ProtocolVersion = 1 }, ""},
		{"missing contract", func(run *Run) {
			run.Events[0].GetReset_().Response.ProofContract = modelv1.ProofContract_PROOF_CONTRACT_UNSPECIFIED
		}, "current whole-prefix proof contract"},
		{"unknown contract", func(run *Run) { run.Events[0].GetReset_().Response.ProofContract = 99 }, "current whole-prefix proof contract"},
		{"missing reset response", func(run *Run) { run.Events[0].GetReset_().Response = nil }, "current whole-prefix proof contract"},
		{"other guest", func(run *Run) { run.ModelHash = "other" }, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, mode := range []Mode{ModeReplay, ModeQuestion, ModeReduce} {
				altered := proto.CloneOf(recorded)
				tc.alter(altered)
				if mode == ModeReduce {
					altered.Verdict = &runv1.Verdict{Outcome: &runv1.Verdict_Unexplained{Unexplained: &runv1.Unexplained{Event: 1}}}
				}
				before := mustMarshal(t, altered)
				run, err := r.Run(t.Context(), mode, Options{Recorded: altered, Prefix: 1})
				if err == nil && run.GetVerdict().GetRefused() == nil {
					t.Fatalf("%s admitted incompatible record: %v", mode, run)
				}
				if tc.wantReason != "" {
					reason := run.GetVerdict().GetRefused().GetReason()
					if err != nil {
						reason = err.Error()
					}
					if !strings.Contains(reason, tc.wantReason) {
						t.Fatalf("%s did not reject the recorded contract at admission: %s", mode, reason)
					}
				}
				if calls != 1 {
					t.Fatalf("%s reached Driver: calls=%d", mode, calls)
				}
				if string(before) != string(mustMarshal(t, altered)) {
					t.Fatalf("%s mutated recording", mode)
				}
			}
		})
	}
}
