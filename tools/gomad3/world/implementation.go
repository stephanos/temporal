package world

import (
	"crypto/sha256"
	"embed"
	"fmt"
)

//go:embed choice.go codec.go compressed.go compressed_decode.go compressed_wire.go errors.go implementation.go queue.go recording.go replay.go snapshot.go types.go world.go
var implementationSources embed.FS

// DiagnosticCardinality is a bounded read-only summary, never a canonical
// replay snapshot or a replacement for complete successful execution evidence.
type DiagnosticCardinality struct {
	Now                                                       LogicalTime `json:"now"`
	Requests, Events, QueuedEvents, Transitions, PayloadBytes uint64
	TranscriptDigest                                          Digest `json:"transcript_digest"`
}

func (w *Model) DiagnosticCardinality() DiagnosticCardinality {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.detached {
		return w.detachedCardinality
	}
	return DiagnosticCardinality{Now: w.now, Requests: uint64(len(w.requests)), Events: uint64(len(w.events)), QueuedEvents: uint64(len(w.queue)), Transitions: uint64(len(w.history)), PayloadBytes: w.payloadBytes, TranscriptDigest: w.transcript}
}

// ImplementationSHA256 binds the World owner used by detached engine replay.
func ImplementationSHA256() string {
	data := []byte("gomad3-world-implementation/v1\n")
	for _, name := range []string{"choice.go", "codec.go", "compressed.go", "compressed_decode.go", "compressed_wire.go", "errors.go", "implementation.go", "queue.go", "recording.go", "replay.go", "snapshot.go", "types.go", "world.go"} {
		source, err := implementationSources.ReadFile(name)
		if err != nil {
			return ""
		}
		data = fmt.Appendf(data, "%s:%x\n", name, sha256.Sum256(source))
	}
	return fmt.Sprintf("sha256:%x", sha256.Sum256(data))
}
