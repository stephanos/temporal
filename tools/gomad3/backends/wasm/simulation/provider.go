package simulation

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"sync"
	"time"

	"go.temporal.io/server/tools/gomad3/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/hostexec"
	"go.temporal.io/server/tools/gomad3/volume"
	"go.temporal.io/server/tools/gomad3/world"
	"go.temporal.io/server/tools/gomad3sim"
)

type Provider struct {
	Boots             map[gomad3sim.BootID]wasi.Request
	IncarnationBoots  []IncarnationBoot
	Broker            wasi.BrokerConfig
	OutputCrashes     []OutputCrash
	OutputCrashGroups []OutputCrashGroup
	Oracles           *OraclePlan
	OutputStops       []OutputCrash
	mu                sync.Mutex
	latest            *execution
}

// IncarnationBoot selects a sealed fresh boot configuration for a restart.
// Recovery images remain lifecycle-owned, never supplied by this mapping.
type IncarnationBoot struct {
	Handle gomad3sim.NodeHandle `json:"handle"`
	Boot   gomad3sim.BootID     `json:"boot"`
}

type OutputCrash struct {
	Handle gomad3sim.NodeHandle `json:"handle"`
	Marker []byte               `json:"marker"`
}

// OutputCrashGroup is a broker-owned acknowledgment barrier for initial guests.
type OutputCrashGroup struct {
	Handles []gomad3sim.NodeHandle `json:"handles"`
	Marker  []byte                 `json:"marker"`
}

type IncarnationResult struct {
	Handle           gomad3sim.NodeHandle
	Result           wasi.Result
	WallElapsedNanos uint64
}

// IncarnationSummary contains detached terminal and cost counters, not guest
// tapes, traces, output, storage images or crash enumeration payloads.
type IncarnationSummary struct {
	Handle                       gomad3sim.NodeHandle
	WallElapsedNanos             uint64
	Termination                  string
	ExitCode                     *uint32
	Reaped                       bool
	FuelRemaining                uint64
	PeakMemoryBytes              uint64
	VirtualTimeElapsedNanos      uint64
	AuthorizationToTerminalNanos uint64
	WASITraceBytes               uint64
	NetworkTraceBytes            uint64
	StorageTraceBytes            uint64
	ChoiceTraceBytes             uint64
	DiagnosticTraceBytes         uint64
	Telemetry                    *wasi.Telemetry
	HelperResourceUsage          *hostexec.ResourceUsage
}

func (p *Provider) CompletedSummaries() []IncarnationSummary {
	p.mu.Lock()
	e := p.latest
	p.mu.Unlock()
	if e == nil {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	results := make([]IncarnationSummary, 0, len(e.results))
	for handle, r := range e.results {
		summary := IncarnationSummary{Handle: handle, WallElapsedNanos: e.elapsed[handle], Termination: r.Termination, Reaped: r.Reaped, FuelRemaining: r.FuelRemaining, PeakMemoryBytes: r.PeakMemoryBytes, VirtualTimeElapsedNanos: r.VirtualTimeElapsedNanos, AuthorizationToTerminalNanos: r.AuthorizationToTerminalNanos, WASITraceBytes: uint64(len(r.Transcript)), NetworkTraceBytes: uint64(len(r.NetworkTranscript)), StorageTraceBytes: uint64(len(r.StorageTranscript)), ChoiceTraceBytes: uint64(len(r.ChoiceTrace.Bytes)) + uint64(len(r.InterruptedChoiceBytes)), DiagnosticTraceBytes: uint64(len(r.DiagnosticTrace.Bytes)) + uint64(len(r.InterruptedDiagnosticBytes))}
		if r.ExitCode != nil {
			code := *r.ExitCode
			summary.ExitCode = &code
		}
		if r.Telemetry != nil {
			telemetry := *r.Telemetry
			summary.Telemetry = &telemetry
		}
		if r.HelperResourceUsage != nil {
			usage := *r.HelperResourceUsage
			summary.HelperResourceUsage = &usage
		}
		results = append(results, summary)
	}
	sort.Slice(results, func(i, j int) bool {
		a, b := results[i].Handle, results[j].Handle
		return a.Node < b.Node || a.Node == b.Node && a.Incarnation < b.Incarnation
	})
	return results
}

// CompletedResults exposes detached helper observations for existing evidence
// collectors. Host telemetry is deliberately outside canonical cluster replay.
func (p *Provider) CompletedResults() []IncarnationResult {
	p.mu.Lock()
	e := p.latest
	p.mu.Unlock()
	if e == nil {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	results := make([]IncarnationResult, 0, len(e.results))
	for handle, result := range e.results {
		result.Stdout = bytes.Clone(result.Stdout)
		result.Stderr = bytes.Clone(result.Stderr)
		result.Transcript = bytes.Clone(result.Transcript)
		result.NetworkTranscript = bytes.Clone(result.NetworkTranscript)
		result.StorageTranscript = bytes.Clone(result.StorageTranscript)
		result.InterruptedChoiceBytes = bytes.Clone(result.InterruptedChoiceBytes)
		result.InterruptedDiagnosticBytes = bytes.Clone(result.InterruptedDiagnosticBytes)
		result.ChoiceTrace.Bytes = bytes.Clone(result.ChoiceTrace.Bytes)
		result.ChoiceTrace.Records = append([]choice.Record(nil), result.ChoiceTrace.Records...)
		result.DiagnosticTrace.Bytes = bytes.Clone(result.DiagnosticTrace.Bytes)
		result.DiagnosticTrace.Records = slices.Clone(result.DiagnosticTrace.Records)
		if result.ExitCode != nil {
			code := *result.ExitCode
			result.ExitCode = &code
		}
		if result.Started != nil {
			started := *result.Started
			started.Imports = slices.Clone(started.Imports)
			started.Exports = slices.Clone(started.Exports)
			result.Started = &started
		}
		if result.ChoiceDivergence != nil {
			divergence := *result.ChoiceDivergence
			if divergence.Expected != nil {
				expected := *divergence.Expected
				divergence.Expected = &expected
			}
			if divergence.Observed != nil {
				observed := *divergence.Observed
				divergence.Observed = &observed
			}
			result.ChoiceDivergence = &divergence
		}
		result.StorageSnapshots = slices.Clone(result.StorageSnapshots)
		for i := range result.StorageSnapshots {
			result.StorageSnapshots[i].Persisted = slices.Clone(result.StorageSnapshots[i].Persisted)
			result.StorageSnapshots[i].Volatile = slices.Clone(result.StorageSnapshots[i].Volatile)
		}
		if result.StorageOperations != nil {
			operations := make(map[string][]volume.Operation, len(result.StorageOperations))
			for id, values := range result.StorageOperations {
				cloned := slices.Clone(values)
				for i := range cloned {
					cloned[i].Dependencies = slices.Clone(cloned[i].Dependencies)
				}
				operations[id] = cloned
			}
			result.StorageOperations = operations
		}
		if result.StorageCrashEnumeration != nil {
			enumerations := make(map[string]volume.CrashEnumeration, len(result.StorageCrashEnumeration))
			for id, enumeration := range result.StorageCrashEnumeration {
				enumeration.States = slices.Clone(enumeration.States)
				for i := range enumeration.States {
					state := &enumeration.States[i]
					state.SelectedOperations = slices.Clone(state.SelectedOperations)
					state.Entries = slices.Clone(state.Entries)
					for j := range state.Entries {
						state.Entries[j].Data = bytes.Clone(state.Entries[j].Data)
					}
				}
				if enumeration.ChoiceExploration != nil {
					exploration := *enumeration.ChoiceExploration
					exploration.Cursor = bytes.Clone(exploration.Cursor)
					exploration.Seen = slices.Clone(exploration.Seen)
					enumeration.ChoiceExploration = &exploration
				}
				enumerations[id] = enumeration
			}
			result.StorageCrashEnumeration = enumerations
		}
		if result.Telemetry != nil {
			telemetry := *result.Telemetry
			result.Telemetry = &telemetry
		}
		if result.HelperResourceUsage != nil {
			usage := *result.HelperResourceUsage
			result.HelperResourceUsage = &usage
		}
		// StorageImage exposes immutable sealed data only; recovery clones its
		// namespace/FS through the volume owner, never through this projection.
		results = append(results, IncarnationResult{Handle: handle, Result: result, WallElapsedNanos: e.elapsed[handle]})
	}
	sort.Slice(results, func(i, j int) bool {
		a, b := results[i].Handle, results[j].Handle
		return a.Node < b.Node || a.Node == b.Node && a.Incarnation < b.Incarnation
	})
	return results
}

type bootIdentity struct {
	Module, Helper, Environment   string
	Implementation                string
	Engine                        wasi.EngineIdentity
	Memory, Fuel                  uint64
	Runtime                       choice.ExecutionIdentity
	Imports                       []wasi.Import
	InitialMemoryPages            uint64
	Timeout                       time.Duration
	ChoiceLimit                   uint64
	Diagnostics, DigestTranscript bool
	StorageCrashSelection         map[string][]uint64
	StorageCrashLimits            *volume.CrashEnumerationLimits
}

func BootConfiguration(request wasi.Request) ([]byte, error) {
	if request.Runtime == nil || request.Runtime.Choice == nil {
		return nil, errors.New("WASM simulation requires cooperative control")
	}
	environment, err := wasi.NewEnvironment(request.Config)
	if err != nil {
		return nil, err
	}
	return json.Marshal(bootIdentity{Module: request.ModuleSHA256, Helper: request.HelperSHA256, Environment: environment.Identity(), Implementation: ImplementationSHA256(), Engine: request.Engine, Memory: request.MemoryBytes, Fuel: request.Fuel, Runtime: request.Runtime.Choice.ExecutionIdentity, Imports: request.Imports, InitialMemoryPages: request.InitialMemoryPages, Timeout: request.Timeout, ChoiceLimit: request.Runtime.Choice.Limit, Diagnostics: request.Runtime.Diagnostics, DigestTranscript: request.DigestTranscript, StorageCrashSelection: request.StorageCrashSelection, StorageCrashLimits: request.StorageCrashLimits})
}

type node struct {
	handle  gomad3sim.NodeHandle
	request wasi.Request
	cancel  context.CancelFunc
	done    chan struct{}
	result  wasi.Result
	err     error
}

type execution struct {
	mu                    sync.Mutex
	ctx                   context.Context
	provider              Provider
	broker                *wasi.Broker
	nodes                 map[gomad3sim.NodeID]*node
	results               map[gomad3sim.NodeHandle]wasi.Result
	requests              map[gomad3sim.NodeHandle]wasi.Request
	elapsed               map[gomad3sim.NodeHandle]uint64
	replay                *gomad3sim.WASMExecutionEvidence
	configurationIdentity string
	limits                gomad3sim.Limits
}

func (p *Provider) Open(ctx context.Context, spec gomad3sim.Spec) (gomad3sim.WASMExecution, error) {
	if p == nil || ctx == nil || spec.Backend != gomad3sim.BackendWASM {
		return nil, &gomad3sim.BackendUnavailableError{Backend: gomad3sim.BackendWASM}
	}
	if err := validateOraclePlan(p.Oracles, spec); err != nil {
		return nil, err
	}
	var oraclePlan *OraclePlan
	if p.Oracles != nil {
		data, err := json.Marshal(p.Oracles)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(data, &oraclePlan); err != nil {
			return nil, err
		}
	}
	config := p.Broker
	config.Nodes = nil
	config.Seed = spec.Seed
	config.Addresses = map[string]string{}
	config.StrictTopology = true
	config.Links = nil
	for _, link := range spec.Links {
		config.Links = append(config.Links, wasi.BrokerLink{From: string(link.From), To: string(link.To), Enabled: link.Enabled, DelayNanos: link.DelayNanos})
	}
	volumes := map[gomad3sim.VolumeID]gomad3sim.VolumeSpec{}
	for _, v := range spec.Volumes {
		volumes[v.ID] = v
	}
	volumeOwners := map[gomad3sim.VolumeID]gomad3sim.NodeID{}
	if config.Limits.MaxRequests == 0 {
		if config.WorldProfile != "" {
			return nil, errors.New("explicit WASM World profile requires explicit configured capacities")
		}
		config.Limits = world.Limits{MaxRequests: spec.Limits.NetworkTransitions * 2, MaxEvents: spec.Limits.NetworkTransitions * 2, MaxQueuedEvents: spec.Limits.NetworkDeliveries, MaxTransitions: spec.Limits.NetworkTransitions * 8, MaxPayloadBytes: spec.Limits.ObservationBytes, MaxStringBytes: 1024}
	}
	boots := make(map[gomad3sim.BootID]wasi.Request, len(p.Boots))
	for id, request := range p.Boots {
		boots[id] = request
	}
	for _, n := range spec.Nodes {
		request, ok := boots[n.Boot]
		if !ok {
			return nil, fmt.Errorf("WASM boot %q has no verified guest", n.Boot)
		}
		if request.Config.Profile != wasi.CooperativeProfile || request.Config.Clock.ReadStepNanos != 0 || request.Broker != nil || request.StorageImage != nil {
			return nil, errors.New("WASM node must have a fresh cooperative frozen-clock environment")
		}
		identity, err := BootConfiguration(request)
		if err != nil {
			return nil, err
		}
		if !bytes.Equal(n.Config, identity) {
			return nil, fmt.Errorf("WASM node %q execution inputs differ from its bound boot configuration", n.ID)
		}
		if request.Runtime.Seed != spec.Seed {
			return nil, errors.New("WASM node seed differs from simulation seed")
		}
		var mounts []volume.VolumeConfig
		if request.Config.Storage != nil {
			mounts = request.Config.Storage.Volumes
		}
		if len(mounts) != len(n.Volumes) {
			return nil, errors.New("WASM durable mounts differ from node specification")
		}
		for _, mount := range mounts {
			id := gomad3sim.VolumeID(mount.ID)
			v, ok := volumes[id]
			if !ok || v.CapacityBytes != mount.CapacityBytes || volumeOwners[id] != "" {
				return nil, errors.New("WASM durable volumes must have independent specified ownership and capacity")
			}
			matched := false
			for _, declared := range n.Volumes {
				matched = matched || declared.Volume == id && declared.Path == mount.Path
			}
			if !matched {
				return nil, errors.New("WASM durable mount differs from specified node mount")
			}
			volumeOwners[id] = n.ID
		}
		config.Nodes = append(config.Nodes, wasi.BrokerNode{ID: string(n.ID), Incarnation: 1})
		config.Addresses[string(n.ID)] = n.Address
		if config.Network == nil {
			config.Network = request.Config.NetworkV2
		}
	}
	semanticConfig := config
	semanticConfig.Diagnostics = false
	bindings := append([]IncarnationBoot(nil), p.IncarnationBoots...)
	for i, binding := range bindings {
		var initial *wasi.Request
		for _, n := range spec.Nodes {
			if n.ID == binding.Handle.Node {
				r := boots[n.Boot]
				initial = &r
			}
		}
		r, ok := boots[binding.Boot]
		if initial == nil || binding.Handle.Incarnation < 2 || !ok || r.Config.Profile != wasi.CooperativeProfile || r.Config.Clock.ReadStepNanos != 0 || r.Broker != nil || r.StorageImage != nil || r.Runtime == nil || r.Runtime.Seed != spec.Seed {
			return nil, errors.New("invalid WASM incarnation boot binding")
		}
		storage, _ := json.Marshal(r.Config.Storage)
		originalStorage, _ := json.Marshal(initial.Config.Storage)
		if !bytes.Equal(storage, originalStorage) {
			return nil, errors.New("WASM restart durable model differs from initial boot")
		}
		for j := 0; j < i; j++ {
			if bindings[j].Handle == binding.Handle {
				return nil, errors.New("duplicate WASM incarnation boot binding")
			}
		}
	}
	type boundBoot struct {
		ID            gomad3sim.BootID
		Configuration []byte
	}
	ids := make([]gomad3sim.BootID, 0, len(boots))
	for id := range boots {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	var configurations []boundBoot
	for _, id := range ids {
		identity, err := BootConfiguration(boots[id])
		if err != nil {
			return nil, err
		}
		configurations = append(configurations, boundBoot{ID: id, Configuration: identity})
	}
	boundaries := append(append([]OutputCrash(nil), p.OutputCrashes...), p.OutputStops...)
	if len(p.OutputCrashGroups) > 1 {
		return nil, errors.New("WASM supports one initial output crash group")
	}
	for _, group := range p.OutputCrashGroups {
		if len(group.Handles) < 2 {
			return nil, errors.New("WASM output crash group requires independent guests")
		}
		for _, handle := range group.Handles {
			if handle.Incarnation != 1 {
				return nil, errors.New("WASM output crash group requires initial incarnations")
			}
			boundaries = append(boundaries, OutputCrash{Handle: handle, Marker: group.Marker})
		}
	}
	for _, boundary := range boundaries {
		known := false
		for _, n := range spec.Nodes {
			known = known || n.ID == boundary.Handle.Node
		}
		if !known || boundary.Handle.Incarnation == 0 || len(boundary.Marker) == 0 || len(boundary.Marker) > 4096 {
			return nil, errors.New("invalid WASM output crash boundaries")
		}
	}
	for i := range boundaries {
		for j := 0; j < i; j++ {
			if boundaries[i].Handle == boundaries[j].Handle {
				return nil, errors.New("duplicate WASM output termination boundary")
			}
		}
	}
	encodedConfig, err := json.Marshal(struct {
		Broker            wasi.BrokerConfig
		OutputCrashes     []OutputCrash
		OutputStops       []OutputCrash
		IncarnationBoots  []IncarnationBoot
		Boots             []boundBoot
		OutputCrashGroups []OutputCrashGroup
		Oracles           *OraclePlan
	}{semanticConfig, p.OutputCrashes, p.OutputStops, bindings, configurations, p.OutputCrashGroups, oraclePlan})
	if err != nil {
		return nil, err
	}
	configurationIdentity := fmt.Sprintf("sha256:%x", sha256.Sum256(encodedConfig))
	if spec.Replay != nil && spec.Replay.WASM != nil && spec.Replay.WASM.ConfigurationSHA256 != configurationIdentity {
		return nil, errors.New("WASM broker configuration differs from retained execution")
	}
	b, err := wasi.NewBroker(config)
	if err != nil {
		return nil, err
	}
	for _, group := range p.OutputCrashGroups {
		var ids []string
		for _, handle := range group.Handles {
			ids = append(ids, string(handle.Node))
		}
		if err := b.CrashAfterOutputs(ids, group.Marker); err != nil {
			return nil, err
		}
	}
	var replay *gomad3sim.WASMExecutionEvidence
	if spec.Replay != nil {
		replay = spec.Replay.WASM
	}
	e := &execution{ctx: ctx, provider: Provider{Boots: boots, IncarnationBoots: bindings, Broker: config, OutputCrashes: p.OutputCrashes, OutputCrashGroups: p.OutputCrashGroups, OutputStops: p.OutputStops, Oracles: oraclePlan}, broker: b, nodes: map[gomad3sim.NodeID]*node{}, results: map[gomad3sim.NodeHandle]wasi.Result{}, requests: map[gomad3sim.NodeHandle]wasi.Request{}, elapsed: map[gomad3sim.NodeHandle]uint64{}, replay: replay, limits: spec.Limits, configurationIdentity: configurationIdentity}
	p.mu.Lock()
	p.latest = e
	p.mu.Unlock()
	return e, nil
}

func (e *execution) Start(spec gomad3sim.NodeSpec, handle gomad3sim.NodeHandle) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	request := e.provider.Boots[spec.Boot]
	for _, binding := range e.provider.IncarnationBoots {
		if binding.Handle == handle {
			request = e.provider.Boots[binding.Boot]
		}
	}
	if e.replay != nil {
		var retained *gomad3sim.WASMIncarnationEvidence
		for i := range e.replay.Incarnations {
			if e.replay.Incarnations[i].Handle == handle {
				retained = &e.replay.Incarnations[i]
				break
			}
		}
		if retained == nil || retained.ModuleSHA256 != request.ModuleSHA256 || retained.HelperSHA256 != request.HelperSHA256 {
			return errors.New("WASM incarnation lacks matching retained execution inputs")
		}
		tape := choice.ReplayPlan{Bytes: bytes.Clone(retained.ChoiceTape), SHA256: sha256.Sum256(retained.ChoiceTape)}
		var err error
		validate := choice.ValidateReplayPlan
		if retained.Interrupted {
			validate = choice.ValidatePrefixReplayPlan
		}
		tape, err = validate(tape, request.Runtime.Choice.ExecutionIdentity)
		if err != nil {
			return err
		}
		control := *request.Runtime
		choiceRequest := *control.Choice
		choiceRequest.Mode, choiceRequest.Tape = choice.ModeReplay, &tape
		if retained.Interrupted {
			choiceRequest.Mode = choice.ModePrefix
		}
		control.Choice = &choiceRequest
		request.Runtime = &control
	}
	session := e.broker.Session(string(handle.Node))
	if old := e.nodes[handle.Node]; old != nil {
		select {
		case <-old.done:
		default:
			return gomad3sim.ErrInvalidTransition
		}
		// Restart schedules the next incarnation; reject before it can hold
		// peers behind a session no helper will own.
		if session == nil || session.Node.Incarnation+1 != handle.Incarnation {
			return gomad3sim.ErrStaleIncarnation
		}
		request.StorageImage = old.result.StorageImage
		if request.Config.Storage != nil && request.StorageImage == nil {
			return errors.New("WASM restart lacks recovered durable storage")
		}
		var err error
		session, err = e.broker.Restart(string(handle.Node))
		if err != nil {
			return err
		}
	}
	if session == nil || session.Node.Incarnation != handle.Incarnation {
		return gomad3sim.ErrStaleIncarnation
	}
	request.Broker = session
	for _, boundary := range e.provider.OutputStops {
		if boundary.Handle == handle {
			if err := session.StopOnOutput(boundary.Marker); err != nil {
				return err
			}
		}
	}
	for _, boundary := range e.provider.OutputCrashes {
		if boundary.Handle == handle {
			if err := session.CrashOnOutput(boundary.Marker); err != nil {
				return err
			}
		}
	}
	ctx, cancel := context.WithCancel(e.ctx)
	n := &node{handle: handle, request: request, cancel: cancel, done: make(chan struct{})}
	e.nodes[handle.Node] = n
	e.requests[handle] = request
	go func() {
		began := time.Now()
		n.result, n.err = wasi.Run(ctx, request)
		elapsed := uint64(time.Since(began))
		e.mu.Lock()
		e.results[handle] = n.result
		e.elapsed[handle] = elapsed
		e.mu.Unlock()
		close(n.done)
	}()
	return nil
}

func (e *execution) lookup(handle gomad3sim.NodeHandle) (*node, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	n := e.nodes[handle.Node]
	if n == nil || n.handle != handle {
		return nil, gomad3sim.ErrStaleIncarnation
	}
	return n, nil
}

func terminal(n *node) (gomad3sim.WASMTerminal, error) {
	if n.err != nil {
		return gomad3sim.WASMTerminal{}, n.err
	}
	if !n.result.Reaped {
		return gomad3sim.WASMTerminal{}, errors.New("WASM helper was not reaped")
	}
	failed := n.result.Termination != "exit" || n.result.ExitCode == nil || *n.result.ExitCode != 0
	reason := n.result.Message
	if failed && reason == "" {
		reason = "WASM guest termination: " + n.result.Termination
		if n.result.ExitCode != nil {
			reason = fmt.Sprintf("WASM guest exited with code %d", *n.result.ExitCode)
		}
	}
	return gomad3sim.WASMTerminal{Failed: failed, Reason: reason, Termination: n.result.Termination}, nil
}

// Shutdown is host cleanup, not a modeled lifecycle action. It must reap every
// helper even if a marker can no longer be reached or the scenario was canceled.
func (e *execution) Shutdown(ctx context.Context) error {
	e.mu.Lock()
	var nodes []*node
	for _, n := range e.nodes {
		nodes = append(nodes, n)
	}
	e.mu.Unlock()
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].handle.Node < nodes[j].handle.Node })
	var failures error
	revocations := map[gomad3sim.NodeHandle]error{}
	// Revoke every live incarnation before waiting for any one helper. A
	// canceled scenario must not leave peers blocked behind cleanup ordering.
	for _, n := range nodes {
		select {
		case <-n.done:
		default:
			revocations[n.handle] = e.broker.Stop(string(n.handle.Node))
		}
	}
	for _, n := range nodes {
		select {
		case <-n.done:
			continue
		default:
		}
		revokeErr := revocations[n.handle]
		// Permit the typed interruption terminal (and its real phase metrics)
		// before the hard cancellation fallback used by mandatory host cleanup.
		grace := time.NewTimer(100 * time.Millisecond)
		select {
		case <-n.done:
			grace.Stop()
			if !n.result.Reaped {
				failures = errors.Join(failures, errors.New("WASM cleanup helper not reaped"))
			}
			failures = errors.Join(failures, n.err)
			continue
		case <-grace.C:
		}
		n.cancel()
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		select {
		case <-n.done:
			if !n.result.Reaped {
				failures = errors.Join(failures, errors.New("WASM cleanup helper not reaped"))
			}
			if n.err != nil {
				failures = errors.Join(failures, n.err)
			}
		case <-cleanup.Done():
			failures = errors.Join(failures, revokeErr, errors.New("WASM cleanup reap deadline exceeded"))
		}
		cancel()
	}
	return failures
}

func (e *execution) Wait(ctx context.Context, handle gomad3sim.NodeHandle) (gomad3sim.WASMTerminal, error) {
	n, err := e.lookup(handle)
	if err != nil {
		return gomad3sim.WASMTerminal{}, err
	}
	select {
	case <-n.done:
		return terminal(n)
	case <-ctx.Done():
		return gomad3sim.WASMTerminal{}, ctx.Err()
	}
}

func (e *execution) Stop(ctx context.Context, handle gomad3sim.NodeHandle) (gomad3sim.WASMTerminal, error) {
	n, err := e.lookup(handle)
	if err != nil {
		return gomad3sim.WASMTerminal{}, err
	}
	for _, boundary := range e.provider.OutputStops {
		if boundary.Handle != handle {
			continue
		}
		select {
		case <-n.done:
			if n.result.Termination != "stop" || !n.result.Reaped {
				return gomad3sim.WASMTerminal{}, errors.New("WASM output stop boundary was not reached and reaped")
			}
			return gomad3sim.WASMTerminal{}, n.err
		case <-ctx.Done():
			return gomad3sim.WASMTerminal{}, ctx.Err()
		}
	}
	select {
	case <-n.done:
		if !n.result.Reaped {
			return gomad3sim.WASMTerminal{}, errors.New("WASM stop did not reap helper")
		}
		return gomad3sim.WASMTerminal{}, n.err
	default:
	}
	if err := e.broker.Stop(string(handle.Node)); err != nil {
		return gomad3sim.WASMTerminal{}, err
	}
	n.cancel()
	select {
	case <-n.done:
		if !n.result.Reaped {
			return gomad3sim.WASMTerminal{}, errors.New("WASM stop did not reap helper")
		}
		return gomad3sim.WASMTerminal{}, n.err
	case <-ctx.Done():
		return gomad3sim.WASMTerminal{}, ctx.Err()
	}
}

func (e *execution) Crash(ctx context.Context, handle gomad3sim.NodeHandle) (gomad3sim.WASMTerminal, error) {
	n, err := e.lookup(handle)
	if err != nil {
		return gomad3sim.WASMTerminal{}, err
	}
	bound := false
	for _, boundary := range e.provider.OutputCrashes {
		bound = bound || boundary.Handle == handle
	}
	for _, group := range e.provider.OutputCrashGroups {
		for _, member := range group.Handles {
			bound = bound || member == handle
		}
	}
	if bound {
		select {
		case <-n.done:
			if n.result.Termination != "crash" || !n.result.Reaped {
				return gomad3sim.WASMTerminal{}, errors.New("WASM output crash boundary was not reached and reaped")
			}
			return gomad3sim.WASMTerminal{}, n.err
		case <-ctx.Done():
			return gomad3sim.WASMTerminal{}, ctx.Err()
		}
	}
	if storageCrashBound(n.request) {
		select {
		case <-n.done:
			if n.result.Termination != "storage_crash" || !n.result.RuntimeInterrupted || !n.result.Reaped || n.result.StorageImage == nil {
				return gomad3sim.WASMTerminal{}, errors.New("WASM storage crash boundary was not reached with durable recovery evidence")
			}
			return gomad3sim.WASMTerminal{}, n.err
		case <-ctx.Done():
			return gomad3sim.WASMTerminal{}, ctx.Err()
		}
	}
	if err = e.broker.Crash(string(handle.Node)); err != nil {
		return gomad3sim.WASMTerminal{}, err
	}
	n.cancel()
	select {
	case <-n.done:
		if !n.result.Reaped {
			return gomad3sim.WASMTerminal{}, errors.New("WASM crash did not reap helper")
		}
		return gomad3sim.WASMTerminal{}, n.err
	case <-ctx.Done():
		return gomad3sim.WASMTerminal{}, ctx.Err()
	}
}

func storageCrashBound(request wasi.Request) bool {
	if request.Config.Storage != nil {
		for _, fault := range request.Config.Storage.Faults {
			if fault.Action == "crash-before" || fault.Action == "crash-after" {
				return true
			}
		}
	}
	return false
}

func (e *execution) Link(action string, from, to gomad3sim.NodeID, delay uint64, symmetric bool) error {
	return e.broker.ChangeLink(action, string(from), string(to), delay, symmetric)
}

func (e *execution) Finish() (gomad3sim.WASMEvidence, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	var evidence gomad3sim.WASMEvidence
	evidence.Execution = &gomad3sim.WASMExecutionEvidence{ConfigurationSHA256: e.configurationIdentity, VirtualElapsedNanos: e.broker.Current(), WorldEncoding: e.provider.Broker.WorldProfile}
	for _, n := range e.nodes {
		select {
		case <-n.done:
		default:
			return evidence, errors.New("WASM simulation has an unreaped node")
		}
	}
	handles := make([]gomad3sim.NodeHandle, 0, len(e.results))
	for handle := range e.results {
		handles = append(handles, handle)
	}
	sort.Slice(handles, func(i, j int) bool {
		a, b := handles[i], handles[j]
		return a.Node < b.Node || a.Node == b.Node && a.Incarnation < b.Incarnation
	})
	for _, handle := range handles {
		result := e.results[handle]
		request := e.requests[handle]
		tape, err := choice.ProjectReplayPlan(result.ChoiceTrace, request.Runtime.Choice.ExecutionIdentity)
		trace, diagnostic := result.ChoiceTrace.Bytes, result.DiagnosticTrace.Bytes
		if result.RuntimeInterrupted {
			tape, err = choice.ProjectInterruptedReplayPlan(result.InterruptedChoiceBytes, request.Runtime.Choice.Limit, request.Runtime.Choice.ExecutionIdentity)
			trace, diagnostic = result.InterruptedChoiceBytes, result.InterruptedDiagnosticBytes
		}
		if err != nil {
			return evidence, fmt.Errorf("retain WASM %s/%d choice control: %w", handle.Node, handle.Incarnation, err)
		}
		retained := gomad3sim.WASMIncarnationEvidence{Handle: handle, ModuleSHA256: request.ModuleSHA256, HelperSHA256: request.HelperSHA256, EnvironmentIdentity: result.EnvironmentIdentity, ChoiceTape: bytes.Clone(tape.Bytes), ChoiceTrace: bytes.Clone(trace), DiagnosticTrace: bytes.Clone(diagnostic), Interrupted: result.RuntimeInterrupted, WASISHA256: fmt.Sprintf("sha256:%x", result.TranscriptSHA256), NetworkSHA256: fmt.Sprintf("sha256:%x", sha256.Sum256(result.NetworkTranscript)), StorageSHA256: fmt.Sprintf("sha256:%x", sha256.Sum256(result.StorageTranscript))}
		if result.StorageImage != nil {
			retained.StorageImageIdentity = result.StorageImage.Identity()
		}
		evidence.Execution.Incarnations = append(evidence.Execution.Incarnations, retained)
		for _, output := range []struct {
			stream gomad3sim.OutputStream
			data   []byte
		}{{gomad3sim.OutputStdout, result.Stdout}, {gomad3sim.OutputStderr, result.Stderr}} {
			if len(output.data) == 0 {
				continue
			}
			evidence.Outputs = append(evidence.Outputs, gomad3sim.OutputObservation{Handle: handle, Stream: output.stream, Bytes: bytes.Clone(output.data), FullSHA256: fmt.Sprintf("sha256:%x", sha256.Sum256(output.data)), TotalBytes: uint64(len(output.data)), RetainedBytes: uint64(len(output.data))})
		}
	}
	// Infrastructure failures with missing control evidence fail before a full
	// canonical World snapshot can duplicate a large model in memory.
	global, err := e.broker.EncodeWorldEvidence()
	if err != nil {
		return evidence, err
	}
	evidence.Execution.World = global
	var states []gomad3sim.VolumeStateSnapshot
	var transitions []gomad3sim.VolumeTransition
	seen := map[string]uint64{}
	for _, handle := range handles {
		result := e.results[handle]
		ids := make([]string, 0, len(result.StorageOperations))
		for id := range result.StorageOperations {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			key := string(handle.Node) + "\x00" + id
			for _, op := range result.StorageOperations[id] {
				if op.ID <= seen[key] {
					continue
				}
				seen[key] = op.ID
				kind := gomad3sim.VolumeTransitionKind(op.Kind)
				if op.Kind == "alloc" {
					kind = gomad3sim.VolumeOperationAllocate
				}
				transitions = append(transitions, gomad3sim.VolumeTransition{Kind: kind, Handle: handle, Volume: gomad3sim.VolumeID(id), Operation: op.ID, Dependencies: append([]uint64(nil), op.Dependencies...), Path: op.Path, Destination: op.Destination, Inode: op.Inode, Offset: op.Offset, Bytes: op.Bytes, PayloadSHA256: op.PayloadSHA256, EffectSHA256: op.EffectSHA256, Outcome: gomad3sim.VolumeOutcomeOK})
			}
		}
	}
	for id, n := range e.nodes {
		for _, snapshot := range n.result.StorageSnapshots {
			states = append(states, gomad3sim.VolumeStateSnapshot{Volume: gomad3sim.VolumeID(snapshot.ID), Node: id, Mount: snapshot.Mount, CapacityBytes: snapshot.CapacityBytes, Persisted: entries(snapshot.Persisted), Volatile: entries(snapshot.Volatile), PendingOperations: snapshot.PendingOperations, PendingSHA256: snapshot.PendingSHA256, NextOperation: snapshot.NextOperation})
		}
	}
	evidence.Volumes, err = gomad3sim.BindWASMVolumeEvidence(transitions, states, e.limits)
	if err != nil {
		return evidence, err
	}
	if e.provider.Broker.Network != nil {
		record, err := e.broker.NetworkRecord()
		if err != nil {
			return evidence, err
		}
		encoded, err := json.Marshal(record)
		if err != nil {
			return evidence, err
		}
		if err = json.Unmarshal(encoded, &evidence.Network); err != nil {
			return evidence, err
		}
	}
	return evidence, nil
}

func entries(source []volume.SnapshotEntry) []gomad3sim.VolumeEntrySnapshot {
	if source == nil {
		return nil
	}
	result := make([]gomad3sim.VolumeEntrySnapshot, len(source))
	for i, v := range source {
		result[i] = gomad3sim.VolumeEntrySnapshot{Path: v.Path, Mode: v.Mode, Kind: v.Kind, ModTime: v.ModTime, Size: v.Size, DataSHA256: v.DataSHA256}
	}
	return result
}
