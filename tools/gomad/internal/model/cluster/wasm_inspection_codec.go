package cluster

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"strings"
	"unicode/utf8"

	"go.temporal.io/server/tools/gomad/internal/model/strictjson"
)

const (
	WASMInspectionBoundaryEncoding  = "gomad.inspection-boundaries-binary/v1"
	WASMInspectionLifecycleEncoding = "gomad.inspection-lifecycle-binary/v1"
	WASMInspectionLifecycleHeader   = "GMWIL\x01\x00\x00"
	inspectionBoundaryHeader        = "GMWIB\x01\x00\x00"
)

type inspectionWire struct {
	Schema            string                      `json:"schema"`
	Complete          bool                        `json:"complete"`
	WorldEncoding     string                      `json:"world_encoding"`
	StateEncoding     string                      `json:"state_encoding"`
	BoundaryEncoding  string                      `json:"boundary_encoding"`
	LifecycleEncoding string                      `json:"lifecycle_encoding"`
	World             []byte                      `json:"world"`
	Lifecycle         []byte                      `json:"lifecycle"`
	Incarnations      []WASMInspectionIncarnation `json:"incarnations"`
	BoundaryFrames    []byte                      `json:"boundary_frames"`
}

func (e WASMInspectionEvidence) MarshalJSON() ([]byte, error) {
	frames, err := encodeInspectionBoundaries(e.Boundaries)
	if err != nil {
		return nil, err
	}
	return json.Marshal(inspectionWire{Schema: e.Schema, Complete: e.Complete, WorldEncoding: e.WorldEncoding, StateEncoding: WASMInspectionStateEncoding, BoundaryEncoding: WASMInspectionBoundaryEncoding, LifecycleEncoding: WASMInspectionLifecycleEncoding, World: e.World, Lifecycle: e.Lifecycle, Incarnations: e.Incarnations, BoundaryFrames: frames})
}

func (e *WASMInspectionEvidence) UnmarshalJSON(data []byte) error {
	if len(data) > MaximumWASMInspectionBytes {
		return ErrWASMInspectionUnsupported
	}
	var wire inspectionWire
	if err := strictjson.Decode(data, &wire); err != nil {
		return err
	}
	if wire.BoundaryEncoding != WASMInspectionBoundaryEncoding || wire.StateEncoding != WASMInspectionStateEncoding || wire.LifecycleEncoding != WASMInspectionLifecycleEncoding {
		return inspectionInvalid("compact evidence encoding")
	}
	boundaries, err := decodeInspectionBoundaries(wire.BoundaryFrames)
	if err != nil {
		return err
	}
	*e = WASMInspectionEvidence{Schema: wire.Schema, Complete: wire.Complete, WorldEncoding: wire.WorldEncoding, World: wire.World, Lifecycle: wire.Lifecycle, Incarnations: wire.Incarnations, Boundaries: boundaries}
	return nil
}

type inspectionDecoder struct {
	data   []byte
	offset int
	err    error
}

func (d *inspectionDecoder) take(n uint64) []byte {
	if d.err != nil {
		return nil
	}
	if n > uint64(len(d.data)-d.offset) {
		d.err = inspectionInvalid("truncated binary evidence")
		return nil
	}
	part := d.data[d.offset : d.offset+int(n)]
	d.offset += int(n)
	return part
}
func (d *inspectionDecoder) u32() uint32 {
	b := d.take(4)
	if len(b) != 4 {
		return 0
	}
	return binary.LittleEndian.Uint32(b)
}
func (d *inspectionDecoder) u64() uint64 {
	b := d.take(8)
	if len(b) != 8 {
		return 0
	}
	return binary.LittleEndian.Uint64(b)
}
func validInspectionNode(node string) bool {
	return len(node) > 0 && len(node) <= 1024 && utf8.ValidString(node) && !strings.ContainsAny(node, "\x00\r\n")
}
func (d *inspectionDecoder) handle(allowController bool) NodeHandle {
	n := d.u32()
	if n == 0 || n > 1024 {
		d.err = inspectionInvalid("binary handle node length")
		return NodeHandle{}
	}
	node := string(d.take(uint64(n)))
	incarnation := d.u64()
	if !validInspectionNode(node) || incarnation == 0 && (!allowController || node != "controller") {
		d.err = inspectionInvalid("binary handle")
	}
	return NodeHandle{Node: NodeID(node), Incarnation: incarnation}
}
func appendInspectionHandle(data []byte, handle NodeHandle) ([]byte, error) {
	if !validInspectionNode(string(handle.Node)) || handle.Incarnation == 0 {
		return nil, inspectionInvalid("binary handle")
	}
	data = binary.LittleEndian.AppendUint32(data, uint32(len(handle.Node)))
	data = append(data, string(handle.Node)...)
	return binary.LittleEndian.AppendUint64(data, handle.Incarnation), nil
}

func decodeInspectionWitness(raw []byte) (*WASMRuntimeWitness, error) {
	if len(raw) != 128 {
		return nil, inspectionInvalid("binary runtime witness length")
	}
	u64 := func(at int) uint64 { return binary.BigEndian.Uint64(raw[at:]) }
	records, choices, goroutines := u64(0), u64(104), u64(112)
	if choices > records || goroutines == 0 {
		return nil, inspectionInvalid("binary runtime witness counters")
	}
	w := &WASMRuntimeWitness{RuntimeState: bytes.Clone(raw), Allocations: u64(16), AllocatedBytes: u64(96), GCCycles: uint64(binary.BigEndian.Uint32(raw[24:])), GCPhase: uint64(raw[28]), Goroutines: goroutines, RunQueue: uint64(binary.BigEndian.Uint32(raw[32:])), GlobalRunQueue: u64(120), Choices: choices, Observations: records - choices}
	for i := range w.SeededDrawCounters {
		w.SeededDrawCounters[i] = u64(40 + i*8)
		w.SeededDraws += w.SeededDrawCounters[i]
	}
	return w, nil
}

func encodeInspectionBoundaries(boundaries []WASMInspectionBoundary) ([]byte, error) {
	if len(boundaries) > MaximumWASMInspectionBoundaries {
		return nil, inspectionInvalid("binary boundary count")
	}
	data := binary.LittleEndian.AppendUint32([]byte(inspectionBoundaryHeader), uint32(len(boundaries)))
	for _, boundary := range boundaries {
		payload, err := appendInspectionHandle(nil, boundary.Handle)
		if err != nil {
			return nil, err
		}
		for _, value := range []uint64{boundary.Decision, boundary.Trace, boundary.Order, boundary.TimeNanos, boundary.WorldBytes, boundary.LifecycleBytes} {
			payload = binary.LittleEndian.AppendUint64(payload, value)
		}
		if len(boundary.Cuts) > MaximumWASMInspectionBoundaries {
			return nil, inspectionInvalid("binary incarnation cut count")
		}
		payload = binary.LittleEndian.AppendUint32(payload, uint32(len(boundary.Cuts)))
		for _, cut := range boundary.Cuts {
			if cut.StateEncoding != WASMInspectionStateEncoding || len(cut.NetworkState) != 32 || len(cut.StorageState) != 32 || len(cut.RecoveryState) != 32 || len(cut.ClockRun) > 1024 || cut.Witness == nil {
				return nil, inspectionInvalid("binary incarnation state")
			}
			witness, err := decodeInspectionWitness(cut.Witness.RuntimeState)
			if err != nil {
				return nil, err
			}
			if !runtimeWitnessEqual(*witness, *cut.Witness) {
				return nil, inspectionInvalid("runtime witness projection differs from raw state")
			}
			payload, err = appendInspectionHandle(payload, cut.Handle)
			if err != nil {
				return nil, err
			}
			for _, value := range []uint64{cut.ChoiceRecords, cut.ChoiceBytes, cut.WASIBytes, cut.NetworkBytes, cut.StorageBytes, cut.StdoutBytes, cut.StderrBytes} {
				payload = binary.LittleEndian.AppendUint64(payload, value)
			}
			payload = binary.LittleEndian.AppendUint32(payload, uint32(len(cut.ClockRun)))
			payload = append(payload, cut.ClockRun...)
			payload = append(payload, cut.NetworkState...)
			payload = append(payload, cut.StorageState...)
			payload = append(payload, cut.RecoveryState...)
			payload = append(payload, witness.RuntimeState...)
			if len(payload) > MaximumWASMInspectionBytes {
				return nil, inspectionInvalid("binary boundary byte capacity")
			}
		}
		if len(data)+4+len(payload) > MaximumWASMInspectionBytes {
			return nil, inspectionInvalid("binary boundary byte capacity")
		}
		data = binary.LittleEndian.AppendUint32(data, uint32(len(payload)))
		data = append(data, payload...)
	}
	return data, nil
}

func decodeInspectionBoundaries(data []byte) ([]WASMInspectionBoundary, error) {
	if len(data) > MaximumWASMInspectionBytes || !bytes.HasPrefix(data, []byte(inspectionBoundaryHeader)) {
		return nil, inspectionInvalid("binary boundary header")
	}
	d := inspectionDecoder{data: data, offset: len(inspectionBoundaryHeader)}
	count := d.u32()
	if count > MaximumWASMInspectionBoundaries || d.err != nil || uint64(count)*69 > uint64(len(data)-d.offset) {
		return nil, inspectionInvalid("binary boundary count")
	}
	boundaries := make([]WASMInspectionBoundary, 0, count)
	for range count {
		payload := d.take(uint64(d.u32()))
		if d.err != nil {
			return nil, d.err
		}
		frame := inspectionDecoder{data: payload}
		boundary := WASMInspectionBoundary{Handle: frame.handle(false), Decision: frame.u64(), Trace: frame.u64(), Order: frame.u64(), TimeNanos: frame.u64(), WorldBytes: frame.u64(), LifecycleBytes: frame.u64()}
		cuts := frame.u32()
		if frame.err != nil || cuts > MaximumWASMInspectionBoundaries || uint64(cuts)*297 > uint64(len(payload)-frame.offset) {
			return nil, inspectionInvalid("binary incarnation cut count")
		}
		boundary.Cuts = make([]WASMInspectionCut, 0, cuts)
		for range cuts {
			cut := WASMInspectionCut{StateEncoding: WASMInspectionStateEncoding, Handle: frame.handle(false), ChoiceRecords: frame.u64(), ChoiceBytes: frame.u64(), WASIBytes: frame.u64(), NetworkBytes: frame.u64(), StorageBytes: frame.u64(), StdoutBytes: frame.u64(), StderrBytes: frame.u64()}
			clock := frame.u32()
			if clock > 1024 {
				return nil, inspectionInvalid("binary folded clock state length")
			}
			cut.ClockRun = bytes.Clone(frame.take(uint64(clock)))
			cut.NetworkState, cut.StorageState, cut.RecoveryState = bytes.Clone(frame.take(32)), bytes.Clone(frame.take(32)), bytes.Clone(frame.take(32))
			var err error
			cut.Witness, err = decodeInspectionWitness(frame.take(128))
			if err != nil {
				return nil, err
			}
			boundary.Cuts = append(boundary.Cuts, cut)
		}
		if frame.err != nil {
			return nil, frame.err
		}
		if frame.offset != len(payload) {
			return nil, inspectionInvalid("binary boundary trailing bytes")
		}
		boundaries = append(boundaries, boundary)
	}
	if d.err != nil {
		return nil, d.err
	}
	if d.offset != len(data) {
		return nil, inspectionInvalid("binary boundary trailing material")
	}
	return boundaries, nil
}

type inspectionLifecycleCut struct {
	order  uint64
	action byte
	handle NodeHandle
}

func inspectionLifecycleCuts(data []byte) (map[uint64]inspectionLifecycleCut, error) {
	if !bytes.HasPrefix(data, []byte(WASMInspectionLifecycleHeader)) || len(data) > MaximumWASMInspectionBytes {
		return nil, inspectionInvalid("binary lifecycle header")
	}
	d := inspectionDecoder{data: data, offset: len(WASMInspectionLifecycleHeader)}
	cuts := make(map[uint64]inspectionLifecycleCut)
	var order uint64
	for d.offset < len(data) && d.err == nil {
		action := d.take(1)[0]
		if action < 1 || action > 10 {
			return nil, inspectionInvalid("binary lifecycle action")
		}
		handle := d.handle(action == 7)
		if action == 7 && (handle.Node != "controller" || handle.Incarnation != 0) {
			return nil, inspectionInvalid("binary controller lifecycle handle")
		}
		order++
		cuts[uint64(d.offset)] = inspectionLifecycleCut{order: order, action: action, handle: handle}
	}
	if d.err != nil {
		return nil, d.err
	}
	return cuts, nil
}
