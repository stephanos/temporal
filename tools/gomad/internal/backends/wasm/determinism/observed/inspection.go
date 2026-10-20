// Package observed decodes retained WASM evidence for determinism comparison.
package observed

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"go.temporal.io/server/tools/gomad/internal/backends/wasm/determinism"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
	"go.temporal.io/server/tools/gomad/internal/model/cluster"
)

// FromInspection preserves fn-180's actual shared order and separate trace and
// decision ordinals. Within a boundary interval the recorded streams have no
// shared event sequence, so the comparison is explicitly boundary-scoped.
func FromInspection(run determinism.Run, metadata []byte) (determinism.Run, error) {
	evidence, err := cluster.DecodeWASMInspection(metadata)
	if err != nil {
		return determinism.Run{}, err
	}
	if !evidence.Complete {
		return determinism.Run{}, errors.New("incomplete inspection evidence")
	}
	return fromEvidence(run, evidence)
}

func fromEvidence(run determinism.Run, evidence *cluster.WASMInspectionEvidence) (determinism.Run, error) {
	diagnostics := map[cluster.NodeHandle][]byte{}
	for _, incarnation := range run.Incarnations {
		diagnostics[cluster.NodeHandle{Node: cluster.NodeID(incarnation.Instance), Incarnation: incarnation.Incarnation}] = incarnation.Diagnostic
	}
	run.BoundaryScoped = true
	run.WorldOrder, run.Boundaries = nil, nil
	for _, boundary := range evidence.Boundaries {
		at := determinism.Coordinate{Shared: boundary.Order, ChoiceOrdinal: boundary.Trace, VirtualTime: int64(boundary.TimeNanos)}
		run.WorldOrder = append(run.WorldOrder, determinism.WorldStep{Order: boundary.Order, Instance: string(boundary.Handle.Node), Incarnation: boundary.Handle.Incarnation, ChoiceOrdinal: boundary.Trace, VirtualTime: int64(boundary.TimeNanos), Digest: digest(evidence.World[:boundary.WorldBytes])})
		run.Boundaries = append(run.Boundaries, determinism.ReplayBoundary{Instance: string(boundary.Handle.Node), Incarnation: boundary.Handle.Incarnation, Trace: boundary.Trace, Decision: boundary.Decision, At: at})
	}
	incarnations := make([]determinism.Incarnation, 0, len(evidence.Incarnations))
	for _, retained := range evidence.Incarnations {
		incarnation := determinism.Incarnation{Instance: string(retained.Handle.Node), Incarnation: retained.Handle.Incarnation, Diagnostic: diagnostics[retained.Handle]}
		if len(evidence.Incarnations) == 1 && len(run.Incarnations) == 1 {
			incarnation.Diagnostic = run.Incarnations[0].Diagnostic
		}
		count, err := choice.TraceRecordCount(retained.Streams.Choice)
		if err != nil {
			return determinism.Run{}, err
		}
		limit, err := choice.TraceBytes(count)
		if err != nil {
			return determinism.Run{}, err
		}
		trace, err := choice.DecodeStoredTrace(choice.Profile, retained.Streams.Choice, choice.TerminalMetadata{State: choice.TerminalComplete, Limit: limit, Records: count, SHA256: sha256.Sum256(retained.Streams.Choice)})
		if err != nil {
			return determinism.Run{}, err
		}
		terminal := determinism.Coordinate{Shared: retained.TerminalOrder, ChoiceOrdinal: count}
		position := func(included func(cluster.WASMInspectionCut) bool) determinism.Coordinate {
			for _, boundary := range evidence.Boundaries {
				for _, cut := range boundary.Cuts {
					if cut.Handle == retained.Handle && included(cut) {
						return determinism.Coordinate{Shared: boundary.Order, ChoiceOrdinal: cut.ChoiceRecords, VirtualTime: int64(boundary.TimeNanos)}
					}
				}
			}
			return terminal
		}
		for _, record := range trace.Records {
			at := position(func(cut cluster.WASMInspectionCut) bool { return cut.ChoiceRecords >= record.Ordinal })
			incarnation.Choices = append(incarnation.Choices, determinism.ChoiceEvent{At: at, Record: record})
		}
		records, err := wasi.DecodeTranscript(retained.Streams.WASI)
		if err != nil {
			return determinism.Run{}, err
		}
		offset := 8
		for _, record := range records {
			if len(retained.Streams.WASI)-offset < 16 {
				return determinism.Run{}, errors.New("truncated WASI frame")
			}
			offset += 16 + int(binary.LittleEndian.Uint32(retained.Streams.WASI[offset+12:]))
			end := uint64(offset)
			encoded, err := json.Marshal(record)
			if err != nil {
				return determinism.Run{}, err
			}
			incarnation.Operations = append(incarnation.Operations, determinism.Operation{At: position(func(cut cluster.WASMInspectionCut) bool { return cut.WASIBytes >= end }), Op: record.Op, Input: string(record.Input), Output: string(encoded), Repeat: record.Count})
		}
		var stdout, stderr uint64
		for _, boundary := range evidence.Boundaries {
			for _, cut := range boundary.Cuts {
				if cut.Handle != retained.Handle {
					continue
				}
				at := determinism.Coordinate{Shared: boundary.Order, ChoiceOrdinal: cut.ChoiceRecords, VirtualTime: int64(boundary.TimeNanos)}
				if len(cut.ClockRun) != 0 {
					incarnation.Operations = append(incarnation.Operations, determinism.Operation{At: at, Op: "clock_time_get", Input: "folded-run-at-boundary", Output: hex.EncodeToString(cut.ClockRun), Repeat: 1})
				}
				appendOutput(&incarnation, at, "stdout", retained.Streams.Stdout, stdout, cut.StdoutBytes)
				appendOutput(&incarnation, at, "stderr", retained.Streams.Stderr, stderr, cut.StderrBytes)
				stdout, stderr = cut.StdoutBytes, cut.StderrBytes
				for _, model := range []struct{ name, value string }{{"network", hex.EncodeToString(cut.NetworkState)}, {"network-transcript", digest(retained.Streams.Network[:cut.NetworkBytes])}, {"volume", hex.EncodeToString(cut.StorageState)}, {"volume-transcript", digest(retained.Streams.Storage[:cut.StorageBytes])}, {"namespace", digest(retained.Streams.WASI[:cut.WASIBytes])}, {"world", digest(evidence.World[:boundary.WorldBytes])}} {
					incarnation.Transitions = append(incarnation.Transitions, determinism.Transition{At: at, Model: model.name, Digest: model.value})
				}
			}
		}
		appendOutput(&incarnation, terminal, "stdout", retained.Streams.Stdout, stdout, uint64(len(retained.Streams.Stdout)))
		appendOutput(&incarnation, terminal, "stderr", retained.Streams.Stderr, stderr, uint64(len(retained.Streams.Stderr)))
		for _, model := range []struct {
			name string
			data []byte
		}{{"network-transcript", retained.Streams.Network}, {"volume-transcript", retained.Streams.Storage}, {"namespace", retained.Streams.WASI}, {"world", evidence.World}} {
			incarnation.Transitions = append(incarnation.Transitions, determinism.Transition{At: terminal, Model: model.name, Digest: digest(model.data)})
		}
		slices.SortStableFunc(incarnation.Operations, func(a, b determinism.Operation) int {
			if a.At.Shared < b.At.Shared {
				return -1
			}
			if a.At.Shared > b.At.Shared {
				return 1
			}
			return 0
		})
		incarnations = append(incarnations, incarnation)
	}
	run.Incarnations = incarnations
	return run, nil
}

func appendOutput(incarnation *determinism.Incarnation, at determinism.Coordinate, stream string, data []byte, start, end uint64) {
	if end > start {
		incarnation.Operations = append(incarnation.Operations, determinism.Operation{At: at, Op: "output:" + stream, Input: fmt.Sprint(start), Output: hex.EncodeToString(data[start:end]), Repeat: 1})
	}
}

func digest(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}
