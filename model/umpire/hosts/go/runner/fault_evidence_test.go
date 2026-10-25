package runner

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
	runv1 "umpire.local/model/gen/umpire/run/v1"
)

func TestDriverFaultEvidenceUsesFreshWireTag(t *testing.T) {
	fields := new(runv1.DriverFault).ProtoReflect().Descriptor().Fields()
	for name, number := range map[string]int{"reason": 1, "evidence": 2} {
		field := fields.ByNumber(protoreflect.FieldNumber(number))
		if field == nil || string(field.Name()) != name {
			t.Fatalf("DriverFault tag %d = %v, want %s", number, field, name)
		}
	}
	fault := &runv1.DriverFault{Reason: "r", Evidence: &anypb.Any{TypeUrl: "x", Value: []byte{1, 2}}}
	const wire = "0a017212070a017812020102"
	encoded := mustMarshal(t, fault)
	if hex.EncodeToString(encoded) != wire {
		t.Fatalf("fault wire = %x, want %s", encoded, wire)
	}
	var decoded runv1.DriverFault
	if err := proto.Unmarshal(encoded, &decoded); err != nil || !proto.Equal(fault, &decoded) {
		t.Fatalf("fault round trip = %v, %v", &decoded, err)
	}
}

type evidenceError struct{ evidence *anypb.Any }

func (e *evidenceError) Error() string        { return "held response" }
func (e *evidenceError) Evidence() *anypb.Any { return e.evidence }

func TestDriverFaultEvidenceIsRetainedRedactedAndReplayed(t *testing.T) {
	wasm := statefulGuest(t)
	for _, source := range []string{"event", "close error"} {
		for _, redact := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/redact=%t", source, redact), func(t *testing.T) {
				evidence := &anypb.Any{TypeUrl: "example.net/Evidence", Value: []byte("wire")}
				h := newHarness(t, wasm, func(int) *fakeDriver {
					if source == "event" {
						return &fakeDriver{script: map[byte][]step{0: {{kind: EventFault, fault: "held response", payload: evidence}}}}
					}
					return &fakeDriver{script: good, closeErr: fmt.Errorf("wrapped: %w", &evidenceError{evidence: evidence})}
				})
				opts := configured(1)
				if redact {
					opts.Redact = func(event *runv1.Event) bool {
						if evidence := event.GetDriverFault().GetEvidence(); evidence != nil {
							evidence.Value = nil
							return true
						}
						return false
					}
				}
				run := h.run(ModeExplore, opts)
				var fault *runv1.Event
				for _, event := range run.GetEvents() {
					if event.GetDriverFault() != nil {
						fault = event
						break
					}
				}
				if fault == nil {
					t.Fatal("no Driver fault recorded")
				}
				requireRefused(t, run, fault.GetSequence(), "held response")
				got := fault.GetDriverFault().GetEvidence()
				if got == nil || got.GetTypeUrl() != evidence.GetTypeUrl() {
					t.Fatalf("fault evidence = %v, want %v", got, evidence)
				}
				if redact {
					if len(got.GetValue()) != 0 || !fault.GetRedacted() || string(evidence.GetValue()) != "wire" {
						t.Fatalf("redaction: fault=%v, source=%v", fault, evidence)
					}
					replayed, err := New(wasm, nil).Run(context.Background(), ModeReplay, Options{Recorded: run})
					if err != nil {
						t.Fatal(err)
					}
					requireRefused(t, replayed, fault.GetSequence(), "redacted")
					return
				}
				if !proto.Equal(got, evidence) {
					t.Fatalf("fault evidence = %v, want %v", got, evidence)
				}
				evidence.Value = []byte("changed")
				if string(got.GetValue()) != "wire" {
					t.Fatal("source mutation changed the Run's fault evidence")
				}
				requireReplays(t, wasm, run)
			})
		}
	}
}

type lateCloseEvidenceDriver struct {
	*fakeDriver
	evidence        *anypb.Any
	closedExchanges bool
}

func (d *lateCloseEvidenceDriver) Close() error {
	exchanges := d.all()
	d.closedExchanges = len(exchanges) != 0
	for _, exchange := range exchanges {
		exchange.mu.Lock()
		closed := exchange.closed
		exchange.mu.Unlock()
		d.closedExchanges = d.closedExchanges && closed && exchange.joined()
	}
	if !d.closedExchanges {
		return errors.New("driver closed before its exchanges joined")
	}
	if d.evidence == nil {
		return errors.New("late driver close")
	}
	d.evidence.Value = []byte("captured-after-exchange-close")
	return fmt.Errorf("late driver close: %w", &evidenceError{evidence: d.evidence})
}

func TestDriverClosePreservesEvidenceCapturedAfterExchangeClose(t *testing.T) {
	wasm := statefulGuest(t)
	for _, withEvidence := range []bool{false, true} {
		t.Run(fmt.Sprintf("evidence=%t", withEvidence), func(t *testing.T) {
			driver := &lateCloseEvidenceDriver{fakeDriver: &fakeDriver{script: good}}
			if withEvidence {
				driver.evidence = &anypb.Any{TypeUrl: "example.net/LateEvidence"}
			}
			runtime := New(wasm, func(context.Context) (Driver, error) { return driver, nil })
			run, err := runtime.Run(t.Context(), ModeExplore, configured(1))
			if err != nil {
				t.Fatal(err)
			}
			if !driver.closedExchanges {
				t.Fatal("Driver.Close ran before Exchange.Close joined")
			}
			var fault *runv1.Event
			for _, event := range run.GetEvents() {
				if event.GetDriverFault() != nil {
					fault = event
				}
			}
			if fault == nil {
				t.Fatal("Driver.Close error was not recorded")
			}
			if !strings.Contains(fault.GetDriverFault().GetReason(), "driver close: late driver close") {
				t.Fatalf("wrong Driver.Close fault: %v", fault)
			}
			requireExplained(t, run, runv1.Termination_TERMINATION_QUIESCENT)
			evidence := fault.GetDriverFault().GetEvidence()
			if withEvidence {
				if !proto.Equal(evidence, driver.evidence) || string(evidence.GetValue()) != "captured-after-exchange-close" {
					t.Fatalf("late Driver.Close evidence lost: %v", evidence)
				}
				driver.evidence.Value = []byte("mutated")
				if string(evidence.GetValue()) != "captured-after-exchange-close" {
					t.Fatal("Driver.Close evidence was not copied")
				}
			} else if evidence != nil {
				t.Fatalf("plain Driver.Close invented evidence: %v", evidence)
			}
		})
	}
}
