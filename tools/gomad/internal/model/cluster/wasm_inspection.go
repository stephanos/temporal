package cluster

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"
	"unicode/utf8"

	"go.temporal.io/server/tools/gomad/internal/model/strictjson"
)

const (
	WASMInspectionSchema               = "gomad.wasm-inspection/v3"
	WASMInspectionWorldEncoding        = "gomad.inspection-world-sha256/v2"
	WASMInspectionStateEncoding        = "gomad.inspection-state-sha256/v1"
	WASMInspectionWorldHeader          = "gomad.inspection-world-commitments/v2\x00"
	MaximumWASMInspectionBytes         = 64 << 20
	MaximumWASMInspectionBoundaries    = 65536
	MaximumWASMInspectionRegistrations = 256
	MaximumWASMInspectionNameBytes     = 128
)

var ErrWASMInspectionUnsupported = errors.New("unsupported or incomplete WASM inspection evidence")

type WASMInspectionEvidence struct {
	Schema        string                      `json:"schema"`
	WorldEncoding string                      `json:"world_encoding"`
	Complete      bool                        `json:"complete"`
	World         []byte                      `json:"world"`
	Lifecycle     []byte                      `json:"lifecycle"`
	Incarnations  []WASMInspectionIncarnation `json:"incarnations"`
	Boundaries    []WASMInspectionBoundary    `json:"-"`
}

type WASMInspectionIncarnation struct {
	Handle        NodeHandle             `json:"handle"`
	StartedOrder  uint64                 `json:"started_order"`
	TerminalOrder uint64                 `json:"terminal_order"`
	FailureOrder  *uint64                `json:"failure_order,omitempty"`
	Registry      WASMInspectionRegistry `json:"registry"`
	Streams       WASMInspectionStreams  `json:"streams"`
}

type WASMInspectionRegistry struct {
	Closed      bool                         `json:"closed"`
	ClosedOrder uint64                       `json:"closed_order"`
	Entries     []WASMInspectionRegistration `json:"entries"`
}

type WASMInspectionRegistration struct {
	ID             uint32 `json:"id"`
	Kind           string `json:"kind"`
	Name           string `json:"name"`
	AvailableOrder uint64 `json:"available_order"`
	Monotonic      bool   `json:"monotonic"`
}

type WASMInspectionStreams struct {
	NetworkEncoding string `json:"network_encoding,omitempty"`
	Choice          []byte `json:"choice"`
	WASI            []byte `json:"wasi"`
	Network         []byte `json:"network"`
	Storage         []byte `json:"storage"`
	Stdout          []byte `json:"stdout"`
	Stderr          []byte `json:"stderr"`
}

type WASMInspectionBoundary struct {
	Handle         NodeHandle          `json:"handle"`
	Decision       uint64              `json:"decision"`
	Trace          uint64              `json:"trace"`
	Order          uint64              `json:"order"`
	TimeNanos      uint64              `json:"time_nanos"`
	WorldBytes     uint64              `json:"world_bytes"`
	LifecycleBytes uint64              `json:"lifecycle_bytes"`
	Cuts           []WASMInspectionCut `json:"cuts"`
}

type WASMInspectionCut struct {
	StateEncoding string              `json:"state_encoding"`
	Handle        NodeHandle          `json:"handle"`
	ChoiceRecords uint64              `json:"choice_records"`
	ChoiceBytes   uint64              `json:"choice_bytes"`
	WASIBytes     uint64              `json:"wasi_bytes"`
	NetworkBytes  uint64              `json:"network_bytes"`
	StorageBytes  uint64              `json:"storage_bytes"`
	StdoutBytes   uint64              `json:"stdout_bytes"`
	StderrBytes   uint64              `json:"stderr_bytes"`
	ClockRun      []byte              `json:"clock_run"`
	NetworkState  []byte              `json:"network_state"`
	StorageState  []byte              `json:"storage_state"`
	RecoveryState []byte              `json:"recovery_state"`
	Witness       *WASMRuntimeWitness `json:"witness"`
}

type WASMRuntimeWitness struct {
	RuntimeState       []byte    `json:"runtime_state"`
	Allocations        uint64    `json:"allocations"`
	AllocatedBytes     uint64    `json:"allocated_bytes"`
	GCCycles           uint64    `json:"gc_cycles"`
	GCPhase            uint64    `json:"gc_phase"`
	Goroutines         uint64    `json:"goroutines"`
	RunQueue           uint64    `json:"run_queue"`
	SeededDraws        uint64    `json:"seeded_draws"`
	SeededDrawCounters [7]uint64 `json:"seeded_draw_counters"`
	GlobalRunQueue     uint64    `json:"global_run_queue"`
	Choices            uint64    `json:"choices"`
	Observations       uint64    `json:"observations"`
}

type WASMInspectionStop struct {
	Kind  string `json:"kind"`
	Value uint64 `json:"value"`
}

type WASMInspectionRequest struct {
	Target NodeHandle         `json:"target"`
	Stop   WASMInspectionStop `json:"stop"`
	Dumps  []string           `json:"dumps"`
	Probes []string           `json:"probes"`
}

type WASMInspectionResolution struct {
	Requested WASMInspectionRequest  `json:"requested"`
	Target    NodeHandle             `json:"target"`
	Boundary  WASMInspectionBoundary `json:"boundary"`
	ProbeIDs  []uint32               `json:"probe_ids"`
}

func DecodeWASMInspection(data []byte) (*WASMInspectionEvidence, error) {
	if len(data) == 0 || len(data) > MaximumWASMInspectionBytes {
		return nil, ErrWASMInspectionUnsupported
	}
	var evidence WASMInspectionEvidence
	if err := strictjson.Decode(data, &evidence); err != nil {
		return nil, fmt.Errorf("inspection metadata: %w", err)
	}
	if err := ValidateWASMInspection(&evidence, nil); err != nil {
		return nil, err
	}
	return &evidence, nil
}

func CloneWASMInspection(source *WASMInspectionEvidence) *WASMInspectionEvidence {
	if source == nil {
		return nil
	}
	cloned := *source
	cloned.World, cloned.Lifecycle = bytes.Clone(source.World), bytes.Clone(source.Lifecycle)
	cloned.Incarnations = slices.Clone(source.Incarnations)
	for i := range cloned.Incarnations {
		n := &cloned.Incarnations[i]
		if n.FailureOrder != nil {
			order := *n.FailureOrder
			n.FailureOrder = &order
		}
		n.Registry.Entries = slices.Clone(n.Registry.Entries)
		n.Streams = WASMInspectionStreams{NetworkEncoding: n.Streams.NetworkEncoding, Choice: bytes.Clone(n.Streams.Choice), WASI: bytes.Clone(n.Streams.WASI), Network: bytes.Clone(n.Streams.Network), Storage: bytes.Clone(n.Streams.Storage), Stdout: bytes.Clone(n.Streams.Stdout), Stderr: bytes.Clone(n.Streams.Stderr)}
	}
	cloned.Boundaries = slices.Clone(source.Boundaries)
	for i := range cloned.Boundaries {
		cloned.Boundaries[i] = cloneInspectionBoundary(source.Boundaries[i])
	}
	return &cloned
}

func cloneInspectionBoundary(source WASMInspectionBoundary) WASMInspectionBoundary {
	source.Cuts = slices.Clone(source.Cuts)
	for i := range source.Cuts {
		cut := &source.Cuts[i]
		cut.ClockRun, cut.StorageState, cut.RecoveryState = bytes.Clone(cut.ClockRun), bytes.Clone(cut.StorageState), bytes.Clone(cut.RecoveryState)
		cut.NetworkState = bytes.Clone(cut.NetworkState)
		if cut.Witness != nil {
			witness := *cut.Witness
			witness.RuntimeState = bytes.Clone(witness.RuntimeState)
			cut.Witness = &witness
		}
	}
	return source
}

func ValidateWASMInspection(evidence *WASMInspectionEvidence, execution *WASMExecutionEvidence) error {
	if evidence == nil || evidence.Schema != WASMInspectionSchema || len(evidence.Incarnations) == 0 || len(evidence.Incarnations) > MaximumWASMInspectionBoundaries || len(evidence.Boundaries) > MaximumWASMInspectionBoundaries {
		return ErrWASMInspectionUnsupported
	}
	if evidence.WorldEncoding != WASMInspectionWorldEncoding {
		return inspectionInvalid("World commitment encoding")
	}
	var lifecycleCuts map[uint64]inspectionLifecycleCut
	if evidence.Complete || len(evidence.Boundaries) != 0 || len(evidence.Lifecycle) != 0 {
		var err error
		lifecycleCuts, err = inspectionLifecycleCuts(evidence.Lifecycle)
		if err != nil {
			return err
		}
	}
	if len(evidence.Boundaries) != 0 || evidence.Complete || len(evidence.World) != 0 {
		if err := validateInspectionWorldCommitments(evidence); err != nil {
			return err
		}
	}
	if evidence.Complete && (len(evidence.World) == 0 || len(evidence.Lifecycle) == 0) {
		return inspectionInvalid("missing shared prefix material")
	}
	encoded, err := json.Marshal(evidence)
	if err != nil {
		return fmt.Errorf("%w: compact metadata encoding: %v", ErrWASMInspectionUnsupported, err)
	}
	if len(encoded) > MaximumWASMInspectionBytes {
		return fmt.Errorf("%w: metadata capacity", ErrWASMInspectionUnsupported)
	}
	incarnations := make(map[NodeHandle]*WASMInspectionIncarnation, len(evidence.Incarnations))
	for i := range evidence.Incarnations {
		n := &evidence.Incarnations[i]
		if n.Streams.NetworkEncoding != "" && n.Streams.NetworkEncoding != "gomad.cluster-network-binary/v1" {
			return inspectionInvalid("network stream encoding")
		}
		if n.Streams.NetworkEncoding == "gomad.cluster-network-binary/v1" && !bytes.HasPrefix(n.Streams.Network, []byte("GMNETEV1")) {
			return inspectionInvalid("network binary stream header")
		}
		if n.Handle.Node == "" || n.Handle.Incarnation == 0 || incarnations[n.Handle] != nil || (evidence.Complete || n.TerminalOrder != 0) && n.TerminalOrder < n.StartedOrder || n.FailureOrder != nil && *n.FailureOrder > n.TerminalOrder {
			return inspectionInvalid("incarnation coordinates")
		}
		incarnations[n.Handle] = n
		if err := validateInspectionRegistry(n); err != nil {
			return err
		}
	}
	lastDecision := make(map[NodeHandle]uint64)
	previousCuts := make(map[NodeHandle]WASMInspectionCut)
	for i, boundary := range evidence.Boundaries {
		event, exists := lifecycleCuts[boundary.LifecycleBytes]
		if !exists || event.order != boundary.Order || event.action != 2 || event.handle != boundary.Handle {
			return inspectionInvalid("binary lifecycle boundary cut")
		}
		n := incarnations[boundary.Handle]
		if n == nil || boundary.Order < n.StartedOrder || (evidence.Complete || n.TerminalOrder != 0) && boundary.Order > n.TerminalOrder || boundary.Decision != lastDecision[boundary.Handle] {
			return inspectionInvalid("decision coordinates")
		}
		lastDecision[boundary.Handle]++
		if i > 0 {
			previous := evidence.Boundaries[i-1]
			if boundary.Order <= previous.Order || boundary.TimeNanos < previous.TimeNanos || boundary.WorldBytes < previous.WorldBytes || boundary.LifecycleBytes < previous.LifecycleBytes {
				return inspectionInvalid("shared boundary order")
			}
		}
		if boundary.WorldBytes == 0 || boundary.LifecycleBytes == 0 || boundary.WorldBytes > uint64(len(evidence.World)) || boundary.LifecycleBytes > uint64(len(evidence.Lifecycle)) {
			return inspectionInvalid("shared stream cuts")
		}
		seen := make(map[NodeHandle]bool, len(boundary.Cuts))
		for _, cut := range boundary.Cuts {
			if cut.StateEncoding != WASMInspectionStateEncoding || len(cut.NetworkState) != sha256.Size || len(cut.StorageState) != sha256.Size || len(cut.RecoveryState) != sha256.Size {
				return inspectionInvalid("incarnation state commitments")
			}
			owner := incarnations[cut.Handle]
			if owner == nil || seen[cut.Handle] || owner.StartedOrder > boundary.Order || cut.Witness == nil || cut.Witness.Goroutines == 0 || len(cut.Witness.RuntimeState) != 128 {
				return inspectionInvalid("missing incarnation cut or runtime witness")
			}
			seen[cut.Handle] = true
			streams := owner.Streams
			if cut.ChoiceBytes > uint64(len(streams.Choice)) {
				return inspectionInvalid("choice stream cut")
			}
			if cut.WASIBytes > uint64(len(streams.WASI)) || cut.NetworkBytes > uint64(len(streams.Network)) || cut.StorageBytes > uint64(len(streams.Storage)) || cut.StdoutBytes > uint64(len(streams.Stdout)) || cut.StderrBytes > uint64(len(streams.Stderr)) {
				return inspectionInvalid("incarnation stream cuts")
			}
			if cut.NetworkBytes < uint64(len(streams.Network)) && len(cut.NetworkState) == 0 || cut.StorageBytes < uint64(len(streams.Storage)) && len(cut.StorageState) == 0 {
				return inspectionInvalid("missing network or storage prefix state")
			}
			if owner.TerminalOrder != 0 && owner.TerminalOrder < boundary.Order && (cut.ChoiceBytes != uint64(len(streams.Choice)) || cut.WASIBytes != uint64(len(streams.WASI)) || cut.NetworkBytes != uint64(len(streams.Network)) || cut.StorageBytes != uint64(len(streams.Storage)) || cut.StdoutBytes != uint64(len(streams.Stdout)) || cut.StderrBytes != uint64(len(streams.Stderr))) {
				return inspectionInvalid("incomplete terminal incarnation cut")
			}
			if cut.Handle == boundary.Handle && cut.ChoiceRecords != boundary.Trace {
				return inspectionInvalid("predecision trace cut")
			}
			if prior, ok := previousCuts[cut.Handle]; ok {
				if cut.ChoiceRecords < prior.ChoiceRecords || cut.ChoiceBytes < prior.ChoiceBytes || cut.WASIBytes < prior.WASIBytes || cut.NetworkBytes < prior.NetworkBytes || cut.StorageBytes < prior.StorageBytes || cut.StdoutBytes < prior.StdoutBytes || cut.StderrBytes < prior.StderrBytes {
					return inspectionInvalid("regressing incarnation cut")
				}
				if owner.TerminalOrder != 0 && owner.TerminalOrder < boundary.Order && i > 0 && owner.TerminalOrder < evidence.Boundaries[i-1].Order && !inspectionCutsEqual(prior, cut) {
					return inspectionInvalid("changed terminal incarnation cut")
				}
			}
			previousCuts[cut.Handle] = cut
		}
		for handle, owner := range incarnations {
			if owner.StartedOrder <= boundary.Order && !seen[handle] {
				return inspectionInvalid("omitted executed incarnation")
			}
		}
	}
	if execution != nil && evidence.Complete {
		return validateInspectionExecution(execution, incarnations)
	}
	return nil
}

func inspectionInvalid(reason string) error {
	return fmt.Errorf("%w: %s", ErrWASMInspectionUnsupported, reason)
}

func validateInspectionRegistry(n *WASMInspectionIncarnation) error {
	registry := n.Registry
	if len(registry.Entries) > MaximumWASMInspectionRegistrations || registry.Closed && (registry.ClosedOrder < n.StartedOrder || n.TerminalOrder != 0 && registry.ClosedOrder > n.TerminalOrder) {
		return inspectionInvalid("registry closure")
	}
	names := map[string]bool{}
	for i, entry := range registry.Entries {
		if entry.ID != uint32(i) || entry.Kind != "dump" && entry.Kind != "probe" && entry.Kind != "check" || entry.Name == "" || len(entry.Name) > MaximumWASMInspectionNameBytes || !utf8.ValidString(entry.Name) || strings.ContainsAny(entry.Name, "\x00\r\n") || names[entry.Name] || entry.AvailableOrder < n.StartedOrder || n.TerminalOrder != 0 && entry.AvailableOrder > n.TerminalOrder || registry.Closed && entry.AvailableOrder > registry.ClosedOrder || entry.Monotonic && entry.Kind != "check" || entry.Name == "stacks" || entry.Name == "heap" {
			return inspectionInvalid("registry descriptor")
		}
		if i > 0 && entry.AvailableOrder < registry.Entries[i-1].AvailableOrder {
			return inspectionInvalid("registry availability order")
		}
		names[entry.Name] = true
	}
	return nil
}

func validateInspectionExecution(execution *WASMExecutionEvidence, incarnations map[NodeHandle]*WASMInspectionIncarnation) error {
	if len(execution.Incarnations) != len(incarnations) {
		return inspectionInvalid("executed incarnation inventory")
	}
	for _, retained := range execution.Incarnations {
		n := incarnations[retained.Handle]
		if n == nil {
			return inspectionInvalid("executed incarnation inventory")
		}
		if !bytes.Equal(n.Streams.Choice, retained.ChoiceTrace) {
			return inspectionInvalid("retained choice stream identity")
		}
		for _, stream := range []struct {
			bytes  []byte
			digest string
		}{{n.Streams.WASI, retained.WASISHA256}, {n.Streams.Storage, retained.StorageSHA256}} {
			if fmt.Sprintf("sha256:%x", sha256.Sum256(stream.bytes)) != stream.digest {
				return inspectionInvalid("canonical retained stream identity")
			}
		}
		// Encoded cluster network streams are decoded and bound by the backend
		// owner with its authenticated network limits before admission.
		if n.Streams.NetworkEncoding == "" && fmt.Sprintf("sha256:%x", sha256.Sum256(n.Streams.Network)) != retained.NetworkSHA256 {
			return inspectionInvalid("canonical retained network identity")
		}
	}
	return nil
}

func validateInspectionOutputs(evidence *WASMInspectionEvidence, outputs []OutputObservation) error {
	if evidence == nil || !evidence.Complete {
		return nil
	}
	for _, incarnation := range evidence.Incarnations {
		for _, stream := range []struct {
			kind OutputStream
			data []byte
		}{{OutputStdout, incarnation.Streams.Stdout}, {OutputStderr, incarnation.Streams.Stderr}} {
			found := false
			for _, output := range outputs {
				if output.Handle != incarnation.Handle || output.Stream != stream.kind {
					continue
				}
				if output.Truncated || output.TotalBytes != uint64(len(stream.data)) || !bytes.Equal(output.Bytes, stream.data) {
					return inspectionInvalid("canonical output identity")
				}
				found = true
			}
			if !found && len(stream.data) != 0 {
				return inspectionInvalid("missing canonical output")
			}
		}
	}
	return nil
}

func ResolveWASMInspection(execution *WASMExecutionEvidence, request WASMInspectionRequest, cluster bool) (WASMInspectionResolution, error) {
	if execution == nil || execution.Inspection == nil || !execution.Inspection.Complete {
		return WASMInspectionResolution{}, ErrWASMInspectionUnsupported
	}
	if err := ValidateWASMInspection(execution.Inspection, execution); err != nil {
		return WASMInspectionResolution{}, err
	}
	return resolveWASMInspection(execution.Inspection, request, cluster)
}

func resolveWASMInspection(evidence *WASMInspectionEvidence, request WASMInspectionRequest, cluster bool) (WASMInspectionResolution, error) {
	resolution := WASMInspectionResolution{Requested: request, Target: request.Target}
	if resolution.Target == (NodeHandle{}) && !cluster && len(evidence.Incarnations) == 1 {
		resolution.Target = evidence.Incarnations[0].Handle
	}
	if resolution.Target.Node == "" || resolution.Target.Incarnation == 0 {
		return resolution, errors.New("inspection requires an explicit instance and incarnation")
	}
	var target *WASMInspectionIncarnation
	for i := range evidence.Incarnations {
		if evidence.Incarnations[i].Handle == resolution.Target {
			target = &evidence.Incarnations[i]
			break
		}
	}
	if target == nil {
		return resolution, errors.New("unknown inspection instance or incarnation")
	}
	var selected *WASMInspectionBoundary
	for i := range evidence.Boundaries {
		b := &evidence.Boundaries[i]
		if b.Handle != resolution.Target {
			continue
		}
		switch request.Stop.Kind {
		case "choice":
			if b.Decision == request.Stop.Value {
				selected = b
			}
		case "time":
			if selected == nil && b.TimeNanos >= request.Stop.Value {
				selected = b
			}
		case "failure":
			if request.Stop.Value != 0 {
				return resolution, errors.New("failure stop cannot carry a value")
			}
			if target.FailureOrder != nil && b.Order <= *target.FailureOrder {
				selected = b
			}
		default:
			return resolution, errors.New("invalid inspection stop selector")
		}
	}
	if selected == nil {
		return resolution, errors.New("inspection stop is outside retained inspectable boundaries")
	}
	resolution.Boundary = cloneInspectionBoundary(*selected)
	if err := resolveInspectionNames(target, selected.Order, request, &resolution); err != nil {
		return resolution, err
	}
	resolution.Requested.Dumps, resolution.Requested.Probes = slices.Clone(request.Dumps), slices.Clone(request.Probes)
	return resolution, nil
}

func resolveInspectionNames(target *WASMInspectionIncarnation, order uint64, request WASMInspectionRequest, resolution *WASMInspectionResolution) error {
	if (len(request.Probes) != 0 || len(request.Dumps) != 0) && !target.Registry.Closed {
		for _, name := range request.Dumps {
			if name != "stacks" && name != "heap" {
				return inspectionInvalid("registry is not closed")
			}
		}
		if len(request.Probes) != 0 {
			return inspectionInvalid("registry is not closed")
		}
	}
	for _, name := range request.Dumps {
		if name == "stacks" || name == "heap" {
			continue
		}
		found := false
		for _, entry := range target.Registry.Entries {
			found = found || entry.Kind == "dump" && entry.Name == name && entry.AvailableOrder <= order
		}
		if !found {
			return fmt.Errorf("unknown or unavailable inspection dump %q", name)
		}
	}
	enabled := map[uint32]bool{}
	for _, pattern := range request.Probes {
		if len(pattern) == 0 || len(pattern) > MaximumWASMInspectionNameBytes {
			return errors.New("invalid inspection probe pattern")
		}
		found := false
		for _, entry := range target.Registry.Entries {
			matched, err := path.Match(pattern, entry.Name)
			if err != nil {
				return fmt.Errorf("invalid inspection probe pattern %q: %w", pattern, err)
			}
			if matched && entry.Kind == "probe" && entry.AvailableOrder <= order {
				found = true
				enabled[entry.ID] = true
			}
		}
		if !found {
			return fmt.Errorf("unknown or unavailable inspection probe pattern %q", pattern)
		}
	}
	for _, entry := range target.Registry.Entries {
		if enabled[entry.ID] {
			resolution.ProbeIDs = append(resolution.ProbeIDs, entry.ID)
		}
	}
	return nil
}

func inspectionCutsEqual(a, b WASMInspectionCut) bool {
	if a.Witness == nil || b.Witness == nil {
		return false
	}
	aw, bw := *a.Witness, *b.Witness
	if !bytes.Equal(aw.RuntimeState, bw.RuntimeState) {
		return false
	}
	aw.RuntimeState, bw.RuntimeState = nil, nil
	return a.StateEncoding == b.StateEncoding && a.Handle == b.Handle && a.ChoiceRecords == b.ChoiceRecords && a.ChoiceBytes == b.ChoiceBytes && a.WASIBytes == b.WASIBytes && a.NetworkBytes == b.NetworkBytes && a.StorageBytes == b.StorageBytes && a.StdoutBytes == b.StdoutBytes && a.StderrBytes == b.StderrBytes && bytes.Equal(a.ClockRun, b.ClockRun) && bytes.Equal(a.NetworkState, b.NetworkState) && bytes.Equal(a.StorageState, b.StorageState) && bytes.Equal(a.RecoveryState, b.RecoveryState) && runtimeWitnessEqual(aw, bw)
}

func validateInspectionWorldCommitments(evidence *WASMInspectionEvidence) error {
	if !bytes.HasPrefix(evidence.World, []byte(WASMInspectionWorldHeader)) {
		return inspectionInvalid("World commitment header")
	}
	offset := len(WASMInspectionWorldHeader)
	for _, boundary := range evidence.Boundaries {
		if len(evidence.World)-offset < 8+sha256.Size || binary.LittleEndian.Uint64(evidence.World[offset:]) != sha256.Size {
			return inspectionInvalid("World commitment frame")
		}
		offset += 8 + sha256.Size
		if boundary.WorldBytes != uint64(offset) {
			return inspectionInvalid("World commitment boundary cut")
		}
	}
	if offset != len(evidence.World) {
		return inspectionInvalid("World commitment trailing material")
	}
	return nil
}

// WASMInspectionStateCommitment binds canonical owner bytes at a recorded cut.
// Callers must obtain material from its state owner, never from replay claims.
func WASMInspectionStateCommitment(kind string, material []byte) ([sha256.Size]byte, error) {
	if kind != "storage" && kind != "recovery" && kind != "network" {
		return [sha256.Size]byte{}, errors.New("unknown inspection state commitment kind")
	}
	domain := WASMInspectionStateEncoding + "/" + kind + "\x00"
	hash := sha256.New()
	var length [8]byte
	binary.LittleEndian.PutUint64(length[:], uint64(len(material)))
	for _, part := range [][]byte{[]byte(domain), length[:], material} {
		if _, err := hash.Write(part); err != nil {
			return [sha256.Size]byte{}, err
		}
	}
	var digest [sha256.Size]byte
	copy(digest[:], hash.Sum(nil))
	return digest, nil
}

func runtimeWitnessEqual(a, b WASMRuntimeWitness) bool {
	return a.Allocations == b.Allocations && a.AllocatedBytes == b.AllocatedBytes && a.GCCycles == b.GCCycles && a.GCPhase == b.GCPhase && a.Goroutines == b.Goroutines && a.RunQueue == b.RunQueue && a.GlobalRunQueue == b.GlobalRunQueue && a.SeededDraws == b.SeededDraws && a.SeededDrawCounters == b.SeededDrawCounters && a.Choices == b.Choices && a.Observations == b.Observations
}

var ErrWASMInstrumentationDivergence = errors.New("WASM instrumentation divergence")

func VerifyWASMInspectionPrefix(retained, observed *WASMInspectionEvidence, boundary WASMInspectionBoundary) error {
	if retained == nil || !retained.Complete {
		return ErrWASMInspectionUnsupported
	}
	if err := ValidateWASMInspection(retained, nil); err != nil {
		return err
	}
	if err := ValidateWASMInspection(observed, nil); err != nil {
		return fmt.Errorf("%w: %v", ErrWASMInstrumentationDivergence, err)
	}
	if retained.WorldEncoding != observed.WorldEncoding {
		return fmt.Errorf("%w: World commitment encoding", ErrWASMInstrumentationDivergence)
	}
	diverged := func(reason string) error { return fmt.Errorf("%w: %s", ErrWASMInstrumentationDivergence, reason) }
	var expected, actual *WASMInspectionBoundary
	for i := range retained.Boundaries {
		b := &retained.Boundaries[i]
		if b.Handle == boundary.Handle && b.Decision == boundary.Decision {
			expected = b
			break
		}
	}
	for i := range observed.Boundaries {
		b := &observed.Boundaries[i]
		if b.Handle == boundary.Handle && b.Decision == boundary.Decision {
			actual = b
			break
		}
	}
	if expected == nil || actual == nil {
		return diverged("validated stop was not reached")
	}
	if boundary.Handle != expected.Handle || boundary.Decision != expected.Decision || boundary.Trace != expected.Trace || boundary.Order != expected.Order || boundary.TimeNanos != expected.TimeNanos || actual.Trace != expected.Trace || actual.Order != expected.Order || actual.TimeNanos != expected.TimeNanos || actual.WorldBytes != expected.WorldBytes || actual.LifecycleBytes != expected.LifecycleBytes {
		return diverged("resolved boundary coordinates")
	}
	if !bytes.Equal(retained.World[:expected.WorldBytes], observed.World[:actual.WorldBytes]) || !bytes.Equal(retained.Lifecycle[:expected.LifecycleBytes], observed.Lifecycle[:actual.LifecycleBytes]) {
		return diverged("shared World or lifecycle prefix")
	}
	if len(expected.Cuts) != len(actual.Cuts) {
		return diverged("executed incarnation inventory")
	}
	lookup := func(e *WASMInspectionEvidence, handle NodeHandle) *WASMInspectionIncarnation {
		for i := range e.Incarnations {
			if e.Incarnations[i].Handle == handle {
				return &e.Incarnations[i]
			}
		}
		return nil
	}
	for _, cut := range expected.Cuts {
		var matched *WASMInspectionCut
		for i := range actual.Cuts {
			if actual.Cuts[i].Handle == cut.Handle {
				matched = &actual.Cuts[i]
				break
			}
		}
		if matched == nil || !inspectionCutsEqual(cut, *matched) {
			return diverged("runtime witness or incarnation cut")
		}
		a, b := lookup(retained, cut.Handle), lookup(observed, cut.Handle)
		if a == nil || b == nil || a.StartedOrder != b.StartedOrder || cut.NetworkBytes != 0 && a.Streams.NetworkEncoding != b.Streams.NetworkEncoding {
			return diverged("incarnation start")
		}
		if !bytes.Equal(a.Streams.Choice[:cut.ChoiceBytes], b.Streams.Choice[:cut.ChoiceBytes]) {
			return diverged("incarnation choice prefix")
		}
		if !bytes.Equal(a.Streams.WASI[:cut.WASIBytes], b.Streams.WASI[:cut.WASIBytes]) || !bytes.Equal(a.Streams.Network[:cut.NetworkBytes], b.Streams.Network[:cut.NetworkBytes]) || !bytes.Equal(a.Streams.Storage[:cut.StorageBytes], b.Streams.Storage[:cut.StorageBytes]) || !bytes.Equal(a.Streams.Stdout[:cut.StdoutBytes], b.Streams.Stdout[:cut.StdoutBytes]) || !bytes.Equal(a.Streams.Stderr[:cut.StderrBytes], b.Streams.Stderr[:cut.StderrBytes]) {
			return diverged("incarnation canonical prefix")
		}
		for _, entry := range a.Registry.Entries {
			if entry.AvailableOrder > boundary.Order {
				break
			}
			if int(entry.ID) >= len(b.Registry.Entries) || entry != b.Registry.Entries[entry.ID] {
				return diverged("registration prefix")
			}
		}
		available := func(registry WASMInspectionRegistry) int {
			n := 0
			for _, entry := range registry.Entries {
				if entry.AvailableOrder <= boundary.Order {
					n++
				}
			}
			return n
		}
		if available(a.Registry) != available(b.Registry) {
			return diverged("registration prefix count")
		}
		aClosed, bClosed := a.Registry.Closed && a.Registry.ClosedOrder <= boundary.Order, b.Registry.Closed && b.Registry.ClosedOrder <= boundary.Order
		if aClosed != bClosed || aClosed && a.Registry.ClosedOrder != b.Registry.ClosedOrder {
			return diverged("registry closure")
		}
	}
	return nil
}
