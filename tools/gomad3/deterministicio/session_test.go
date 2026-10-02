package deterministicio

import (
	"crypto/sha256"
	"errors"
	"reflect"
	"strings"
	"testing"

	iowire "go.temporal.io/server/tools/gomad3/deterministicio/internal/wire"
)

func TestSessionCollectTerminal(t *testing.T) {
	complete := iowire.EncodeTerminal(iowire.Terminal{State: iowire.TerminalComplete, MappingBytes: iowire.TranscriptHeaderBytes, PayloadHash: sha256.Sum256(nil)})
	checksum := complete
	checksum[len(checksum)-1] ^= 1
	digest := iowire.EncodeTerminal(iowire.Terminal{State: iowire.TerminalComplete, MappingBytes: iowire.TranscriptHeaderBytes, PayloadHash: sha256.Sum256([]byte("corrupt"))})
	overflow := iowire.EncodeTerminal(iowire.Terminal{State: iowire.TerminalOverflow, MappingBytes: iowire.TranscriptHeaderBytes})
	oversized := iowire.EncodeTerminal(iowire.Terminal{State: iowire.TerminalComplete, MappingBytes: iowire.TranscriptHeaderBytes + iowire.TranscriptRecordBytes, Records: 1})
	diverged := iowire.EncodeTerminal(iowire.Terminal{State: iowire.TerminalReplayDivergence, MappingBytes: iowire.TranscriptHeaderBytes, PayloadHash: sha256.Sum256(nil), DivergentOrdinal: 3})
	for _, test := range []struct {
		name       string
		frame      []byte
		message    string
		divergence bool
	}{
		{name: "absent", message: "I/O transcript terminal is absent"},
		{name: "truncated", frame: complete[:len(complete)-1], message: "invalid I/O terminal frame"},
		{name: "malformed", frame: make([]byte, len(complete)), message: "invalid I/O terminal frame"},
		{name: "checksum", frame: checksum[:], message: "I/O terminal frame checksum mismatch"},
		{name: "digest", frame: digest[:], message: "I/O transcript digest mismatch"},
		{name: "overflow", frame: overflow[:], message: "I/O transcript did not complete"},
		{name: "capacity", frame: oversized[:], message: "invalid I/O transcript length"},
		{name: "complete", frame: complete[:]},
		{name: "replay_divergence", frame: diverged[:], divergence: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			limit := uint64(iowire.TranscriptHeaderBytes)
			if test.name == "absent" {
				limit += iowire.TranscriptRecordBytes
			}
			session, err := NewSession(SessionSpec{Limit: limit})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := session.Close(); err != nil {
					t.Error(err)
				}
			})
			if _, err := session.Files().Terminal.Write(test.frame); err != nil {
				t.Fatal(err)
			}
			if err := session.Files().Terminal.Close(); err != nil {
				t.Fatal(err)
			}
			if test.name == "absent" {
				if _, err := session.Files().Transcript.WriteAt(make([]byte, iowire.TranscriptRecordBytes), iowire.TranscriptHeaderBytes); err != nil {
					t.Fatal(err)
				}
			}
			observed, err := session.Collect()
			if errors.Is(err, ErrTranscriptUnterminated) != (test.name == "absent") {
				t.Fatalf("unterminated classification = %v for %s", err, test.name)
			}
			if test.message != "" {
				if err == nil || !strings.Contains(err.Error(), test.message) {
					t.Fatalf("Collect error = %v, want %s", err, test.message)
				}
				if !reflect.DeepEqual(observed, Transcript{}) {
					t.Fatalf("incomplete transcript = %#v", observed)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !observed.Complete || observed.Records != 0 || len(observed.Bytes) != 0 || observed.SHA256 != sha256.Sum256(nil) {
				t.Fatalf("complete transcript = %#v", observed)
			}
			if test.divergence {
				if observed.ReplayDivergence == nil || *observed.ReplayDivergence != 3 {
					t.Fatalf("divergence = %v", observed.ReplayDivergence)
				}
			} else if observed.ReplayDivergence != nil {
				t.Fatalf("unexpected divergence = %v", observed.ReplayDivergence)
			}
		})
	}
}
