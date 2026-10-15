package wasi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"path/filepath"
	"sort"
	"sync"
	"time"

	hostexec "go.temporal.io/server/tools/gomad/internal/host/exec"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
	netmodel "go.temporal.io/server/tools/gomad/internal/model/network"
	"go.temporal.io/server/tools/gomad/internal/model/world"
)

type ClusterInstance struct {
	ID      string
	Request Request
}

type ClusterBoundary struct {
	Node        string `json:"node"`
	Incarnation uint64 `json:"incarnation"`
	Marker      []byte `json:"marker_base64"`
}

type ClusterCrashGroup struct {
	Nodes  []string `json:"nodes"`
	Marker []byte   `json:"marker_base64"`
}

type ClusterRequest struct {
	Broker                     BrokerConfig
	Instances                  []ClusterInstance
	Boots                      map[string]Request
	OutputCrashes, OutputStops []ClusterBoundary
	OutputCrashGroup           *ClusterCrashGroup
}

type ClusterFrameCounts struct {
	GuestCallbacks, Controls, Results uint64
}

type ClusterSchedulerMetrics struct {
	Handoffs         uint64 `json:"handoffs"`
	HandoffSamples   uint64 `json:"handoff_samples"`
	HandoffNanos     uint64 `json:"handoff_nanos"`
	ActiveNanos      uint64 `json:"active_nanos"`
	OverlapNanos     uint64 `json:"overlap_nanos"`
	OperationSamples uint64 `json:"operation_samples"`
	RunnableSamples  uint64 `json:"runnable_samples"`
	GuestNanos       uint64 `json:"guest_nanos"`
	HostNanos        uint64 `json:"host_nanos"`
}

type ClusterResult struct {
	World                        []byte
	Network                      netmodel.Record
	VirtualElapsedNanos          uint64
	Metrics                      *Telemetry
	HelperResourceUsage          *hostexec.ResourceUsage
	Frames                       ClusterFrameCounts
	Scheduler                    *ClusterSchedulerMetrics
	WallElapsedNanos             uint64
	AuthorizationToTerminalNanos uint64
	AuthorizationStartedAt       time.Time `json:"-"`
	TerminalObservedAt           time.Time `json:"-"`
}

type clusterBoot struct {
	request Request
	config  clusterInstanceFrame
	inputs  []byte
	started *Started
}

type clusterState struct {
	sink                   *frameSink
	network                []byte
	progress               hash.Hash
	progressBytes          uint64
	result                 Result
	err                    error
	done                   chan struct{}
	authorizationStartedAt time.Time
	executionClassifiedAt  time.Time
}

type ClusterSession struct {
	mu                    sync.Mutex
	writeMu               sync.Mutex
	config                ClusterRequest
	boots                 map[string]*clusterBoot
	states                map[BrokerNode]*clusterState
	current               map[string]uint64
	pending               map[uint64]chan error
	sequence              uint64
	counts                ClusterFrameCounts
	input                 *helperInput
	ctx                   context.Context
	cancel                context.CancelFunc
	ready                 chan struct{}
	done                  chan struct{}
	err                   error
	result                ClusterResult
	terminal              bool
	finishing             bool
	worldWire             []byte
	networkWire           []byte
	networkSource         string
	began                 time.Time
	authorizedAt          time.Time
	executionClassifiedAt time.Time
}

type clusterInstanceFrame struct {
	ID             string                `json:"id"`
	ModulePath     string                `json:"module_path,omitempty"`
	ModuleSHA256   string                `json:"module_sha256,omitempty"`
	MemoryBytes    uint64                `json:"memory_bytes"`
	Fuel           uint64                `json:"fuel"`
	OutputBytes    uint64                `json:"output_bytes"`
	LocalModel     *helperModelConfig    `json:"local_model"`
	Namespace      *namespaceDeclaration `json:"namespace"`
	Storage        *helperStorageRequest `json:"storage,omitempty"`
	Network        *NetworkModelV2       `json:"network_v2,omitempty"`
	StdoutProgress bool                  `json:"stdout_progress,omitempty"`
}

type clusterBrokerFrame struct {
	Seed   uint64 `json:"seed"`
	Limits struct {
		Requests    uint64 `json:"max_requests"`
		Events      uint64 `json:"max_events"`
		Queued      uint64 `json:"max_queued_events"`
		Transitions uint64 `json:"max_transitions"`
		Payload     uint64 `json:"max_payload_bytes"`
		Strings     uint32 `json:"max_string_bytes"`
	} `json:"limits"`
}

func prepareClusterBoot(id string, request Request) (*clusterBoot, error) {
	if request.Broker != nil || request.StorageImage != nil || request.Config.Profile != CooperativeProfile || request.Runtime == nil || request.Runtime.Choice == nil || request.Config.Clock.ReadStepNanos != 0 || request.Engine != PinnedEngine() || request.Timeout <= 0 || request.InitialMemoryPages == 0 || request.MemoryBytes == 0 || request.MemoryBytes > 1<<32 || request.MemoryBytes%65536 != 0 || request.Fuel == 0 {
		return nil, invalid("cluster boot configuration")
	}
	if err := validateInputIdentity(request.ModulePath, request.ModuleSHA256); err != nil {
		return nil, err
	}
	if request.CompileCache != "" && !filepath.IsAbs(request.CompileCache) {
		return nil, invalid("compile cache directory")
	}
	e, err := NewEnvironment(request.Config)
	if err != nil {
		return nil, err
	}
	e.runtime, err = newRuntimeSession(request.Runtime)
	if err != nil {
		return nil, err
	}
	e.helperOwned, e.namespaceOwned = true, true
	e.mirror = request.StdoutMirror
	image, declaration, err := declareNamespaceImage(e)
	if err != nil {
		return nil, err
	}
	boot := &clusterBoot{request: request, config: clusterInstanceFrame{ID: id, ModulePath: request.ModulePath, ModuleSHA256: request.ModuleSHA256, MemoryBytes: request.MemoryBytes, Fuel: request.Fuel, OutputBytes: request.Config.Limits.OutputBytes, LocalModel: helperConfig(e, request.DigestTranscript), Namespace: declaration, Network: request.Config.NetworkV2, StdoutProgress: request.StdoutMirror != nil}}
	var storage []byte
	if request.Config.Storage != nil {
		boot.config.Storage, storage, err = helperStorageConfig(e, request)
		if err != nil {
			return nil, err
		}
	}
	boot.inputs, err = appendHelperInputs(nil, e)
	if err != nil {
		return nil, err
	}
	var namespace bytes.Buffer
	sink := &frameSink{environment: e, request: request, namespaceImage: image, storageImage: storage, input: &namespace}
	if err := sink.sendHelperNamespace(); err != nil {
		return nil, err
	}
	boot.inputs = append(boot.inputs, namespace.Bytes()...)
	return boot, nil
}

func OpenCluster(ctx context.Context, request ClusterRequest) (*ClusterSession, error) {
	began := time.Now()
	if ctx == nil || len(request.Instances) == 0 || len(request.Instances) > 64 {
		return nil, invalid("cluster instances")
	}
	instances := append([]ClusterInstance(nil), request.Instances...)
	sort.Slice(instances, func(i, j int) bool { return instances[i].ID < instances[j].ID })
	request.Instances = instances
	first := instances[0].Request
	for _, instance := range instances {
		first.Timeout = max(first.Timeout, instance.Request.Timeout)
	}
	for _, boot := range request.Boots {
		first.Timeout = max(first.Timeout, boot.Timeout)
	}
	if err := verifyFile(first.HelperPath, first.HelperSHA256); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, first.Timeout)
	session := &ClusterSession{config: request, boots: map[string]*clusterBoot{}, states: map[BrokerNode]*clusterState{}, current: map[string]uint64{}, pending: map[uint64]chan error{}, ctx: ctx, cancel: cancel, ready: make(chan struct{}), done: make(chan struct{})}
	session.began = began
	var configs, boots []clusterInstanceFrame
	networks, cooperative := map[string]*NetworkModelV2{}, map[string]bool{}
	var order []string
	for _, instance := range instances {
		if _, exists := session.boots[instance.ID]; exists {
			cancel()
			return nil, invalid("duplicate cluster instance")
		}
		if instance.Request.HelperPath != first.HelperPath || instance.Request.HelperSHA256 != first.HelperSHA256 || instance.Request.Engine != first.Engine || instance.Request.CompileCache != first.CompileCache {
			cancel()
			return nil, invalid("cluster helper identity")
		}
		boot, err := prepareClusterBoot(instance.ID, instance.Request)
		if err != nil {
			cancel()
			return nil, err
		}
		session.boots[instance.ID] = boot
		configs = append(configs, boot.config)
		order = append(order, instance.ID)
		networks[instance.ID], cooperative[instance.ID] = instance.Request.Config.NetworkV2, true
	}
	ids := make([]string, 0, len(request.Boots))
	for id := range request.Boots {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if _, exists := session.boots[id]; exists {
			cancel()
			return nil, invalid("duplicate cluster boot")
		}
		r := request.Boots[id]
		if r.HelperPath != first.HelperPath || r.HelperSHA256 != first.HelperSHA256 || r.Engine != first.Engine || r.CompileCache != first.CompileCache {
			cancel()
			return nil, invalid("cluster helper identity")
		}
		boot, err := prepareClusterBoot(id, r)
		if err != nil {
			cancel()
			return nil, err
		}
		session.boots[id] = boot
		boots = append(boots, boot.config)
		order = append(order, id)
	}
	network, err := ClusterNetworkFrame(request.Broker, networks, cooperative)
	if err != nil {
		cancel()
		return nil, err
	}
	broker := clusterBrokerFrame{Seed: request.Broker.Seed}
	broker.Limits.Requests, broker.Limits.Events, broker.Limits.Queued = request.Broker.Limits.MaxRequests, request.Broker.Limits.MaxEvents, request.Broker.Limits.MaxQueuedEvents
	broker.Limits.Transitions, broker.Limits.Payload, broker.Limits.Strings = request.Broker.Limits.MaxTransitions, request.Broker.Limits.MaxPayloadBytes, request.Broker.Limits.MaxStringBytes
	initial, err := json.Marshal(struct {
		Schema           string                 `json:"schema"`
		Type             string                 `json:"type"`
		ModulePath       string                 `json:"module_path"`
		ModuleSHA256     string                 `json:"module_sha256"`
		CompileCache     string                 `json:"compile_cache,omitempty"`
		ManualStart      bool                   `json:"manual_start"`
		Instances        []clusterInstanceFrame `json:"instances"`
		Boots            []clusterInstanceFrame `json:"boots,omitempty"`
		Broker           clusterBrokerFrame     `json:"broker"`
		Network          *ClusterNetworkConfig  `json:"network,omitempty"`
		OutputCrashes    []ClusterBoundary      `json:"output_crashes,omitempty"`
		OutputStops      []ClusterBoundary      `json:"output_stops,omitempty"`
		OutputCrashGroup *ClusterCrashGroup     `json:"output_crash_group,omitempty"`
	}{Schema: "gomad3.wasm-host/v1", Type: "cluster_execute", ModulePath: first.ModulePath, ModuleSHA256: first.ModuleSHA256, CompileCache: first.CompileCache, ManualStart: true, Instances: configs, Boots: boots, Broker: broker, Network: network, OutputCrashes: request.OutputCrashes, OutputStops: request.OutputStops, OutputCrashGroup: request.OutputCrashGroup})
	if err != nil {
		cancel()
		return nil, err
	}
	initial = append(initial, '\n')
	for _, id := range order {
		for _, frame := range bytes.Split(bytes.TrimSpace(session.boots[id].inputs), []byte{'\n'}) {
			initial = appendClusterTag(initial, id, 1, frame)
		}
		session.boots[id].inputs = nil
		session.boots[id].config = clusterInstanceFrame{}
	}
	session.input, err = newHelperInput()
	if err != nil {
		cancel()
		return nil, err
	}
	session.input.start(initial)
	go session.run(first)
	select {
	case <-session.ready:
		return session, nil
	case <-session.done:
		return nil, session.err
	case <-ctx.Done():
		cancel()
		session.input.closeInput()
		<-session.done
		return nil, session.err
	}
}

func appendClusterTag(output []byte, node string, incarnation uint64, frame []byte) []byte {
	tag, _ := json.Marshal(node)
	output = append(output, `{"instance":`...)
	output = append(output, tag...)
	if incarnation > 1 {
		output = fmt.Appendf(output, `,"incarnation":%d`, incarnation)
	}
	output = append(output, ',')
	output = append(output, bytes.TrimSpace(frame)[1:]...)
	return append(output, '\n')
}

func (session *ClusterSession) run(request Request) {
	stream := &clusterStream{session: session}
	host, runErr := hostexec.Run(session.ctx, hostexec.Request{Command: []string{request.HelperPath}, Dir: filepath.Dir(request.HelperPath), Env: []string{}, Stdin: session.input.reader, StdoutSink: stream, StdoutDone: session.input.closeInput, Timeout: request.Timeout, TerminateGrace: 100 * time.Millisecond, OutputLimit: 64 << 10, CollectResourceUsage: true})
	session.input.closeInput()
	closeErr := session.input.finish()
	session.mu.Lock()
	defer session.mu.Unlock()
	session.result.HelperResourceUsage = host.ResourceUsage
	session.result.Frames = session.counts
	if session.err == nil {
		if runErr != nil || closeErr != nil || !host.GroupGone || host.Termination != hostexec.TerminationExit || host.ExitCode != 0 {
			session.err = fmt.Errorf("cluster helper infrastructure: %w", errors.Join(runErr, closeErr, errors.New(string(host.Stderr.RawBytes))))
		} else if !session.terminal || len(stream.buffer) != 0 {
			session.err = invalid("incomplete cluster frames")
		}
	}
	for _, state := range session.states {
		select {
		case <-state.done:
		default:
			state.err = session.err
			if state.err == nil {
				state.err = invalid("missing incarnation result")
			}
			state.result.Reaped = host.GroupGone
			state.executionClassifiedAt = time.Now()
			close(state.done)
		}
	}
	session.executionClassifiedAt = time.Now()
	session.result.WallElapsedNanos = uint64(session.executionClassifiedAt.Sub(session.began))
	session.result.AuthorizationStartedAt = session.authorizedAt
	session.result.TerminalObservedAt = session.executionClassifiedAt
	close(session.done)
	session.cancel()
}

func (session *ClusterSession) FrameCounts() ClusterFrameCounts {
	session.mu.Lock()
	defer session.mu.Unlock()
	return session.counts
}

// CompletedResult returns detached shared process costs, without canonical payloads.
func (session *ClusterSession) CompletedResult() *ClusterResult {
	session.mu.Lock()
	defer session.mu.Unlock()
	select {
	case <-session.done:
	default:
		return nil
	}
	result := ClusterResult{VirtualElapsedNanos: session.result.VirtualElapsedNanos, Frames: session.counts, WallElapsedNanos: session.result.WallElapsedNanos, AuthorizationToTerminalNanos: session.result.AuthorizationToTerminalNanos, AuthorizationStartedAt: session.result.AuthorizationStartedAt, TerminalObservedAt: session.result.TerminalObservedAt}
	if session.result.Metrics != nil {
		metrics := *session.result.Metrics
		result.Metrics = &metrics
	}
	if session.result.HelperResourceUsage != nil {
		usage := *session.result.HelperResourceUsage
		result.HelperResourceUsage = &usage
	}
	if session.result.Scheduler != nil {
		scheduler := *session.result.Scheduler
		result.Scheduler = &scheduler
	}
	return &result
}

func (session *ClusterSession) command(ctx context.Context, frame map[string]any, suffix []byte, timing *clusterState) error {
	session.writeMu.Lock()
	session.mu.Lock()
	if session.terminal || session.err != nil {
		err := session.err
		session.mu.Unlock()
		session.writeMu.Unlock()
		if err == nil {
			err = invalid("cluster already finished")
		}
		return err
	}
	session.sequence++
	id := session.sequence
	frame["id"] = id
	wait := make(chan error, 1)
	session.pending[id] = wait
	session.counts.Controls++
	session.mu.Unlock()
	data, err := json.Marshal(frame)
	if err == nil {
		data = append(append(data, '\n'), suffix...)
		if timing != nil {
			session.mu.Lock()
			timing.authorizationStartedAt = time.Now()
			timing.sink.authorizedAt = timing.authorizationStartedAt
			session.mu.Unlock()
		}
		var n int
		n, err = session.input.Write(data)
		if err == nil && n != len(data) {
			err = io.ErrShortWrite
		}
	}
	session.writeMu.Unlock()
	if err != nil {
		session.cancel()
		return err
	}
	select {
	case err := <-wait:
		return err
	case <-session.done:
		return session.err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (session *ClusterSession) Start(ctx context.Context, node string, incarnation uint64, bootID string, runtime *RuntimeControl) error {
	session.mu.Lock()
	boot := session.boots[bootID]
	if boot == nil || incarnation == 0 || session.current[node]+1 != incarnation || session.finishing {
		session.mu.Unlock()
		return invalid("cluster incarnation boot")
	}
	previous := session.states[BrokerNode{ID: node, Incarnation: incarnation - 1}]
	if previous != nil {
		select {
		case <-previous.done:
		default:
			session.mu.Unlock()
			return invalid("cluster restart before result")
		}
	}
	request := boot.request
	request.Runtime = runtime
	e, err := NewEnvironment(request.Config)
	if err == nil && previous != nil && request.Config.Storage != nil {
		image := previous.result.StorageImage
		model, encodeErr := json.Marshal(request.Config.Storage)
		if encodeErr != nil {
			err = encodeErr
		} else if image == nil || !bytes.Equal(model, image.model) {
			err = invalid("recovery storage model identity")
		} else {
			e.identity = recoveredEnvironmentIdentity(e.identity, image.Identity())
		}
	}
	if err == nil {
		e.runtime, err = newRuntimeSession(runtime)
	}
	if err != nil {
		session.mu.Unlock()
		return err
	}
	e.helperOwned, e.namespaceOwned = true, true
	e.mirror = nil
	sink := &frameSink{request: request, environment: e, started: boot.started, authorizedAt: time.Now()}
	state := &clusterState{sink: sink, done: make(chan struct{})}
	handle := BrokerNode{ID: node, Incarnation: incarnation}
	session.states[handle], session.current[node] = state, incarnation
	session.mu.Unlock()
	var tape bytes.Buffer
	sink.input = &tape
	if err := sink.sendHelperTape(); err != nil {
		return err
	}
	var tagged []byte
	for _, frame := range bytes.Split(bytes.TrimSpace(tape.Bytes()), []byte{'\n'}) {
		if len(frame) != 0 {
			tagged = appendClusterTag(tagged, node, incarnation, frame)
		}
	}
	kind := "start"
	if incarnation > 1 {
		kind = "restart"
	}
	model := helperConfig(e, request.DigestTranscript)
	return session.command(ctx, map[string]any{"type": kind, "node": node, "incarnation": incarnation, "boot": bootID, "runtime": struct {
		Mode       choice.Mode `json:"mode"`
		TapeChunks uint64      `json:"tape_chunks"`
	}{model.Mode, model.TapeChunks}}, tagged, state)
}

func (session *ClusterSession) Wait(ctx context.Context, node string, incarnation uint64) (Result, error) {
	session.mu.Lock()
	state := session.states[BrokerNode{ID: node, Incarnation: incarnation}]
	session.mu.Unlock()
	if state == nil {
		return Result{}, invalid("unknown cluster incarnation")
	}
	select {
	case <-state.done:
		return state.result, state.err
	case <-ctx.Done():
		return Result{}, ctx.Err()
	}
}

func (session *ClusterSession) Revoke(ctx context.Context, action, node string, incarnation uint64) error {
	return session.command(ctx, map[string]any{"type": "control", "command": action, "node": node, "incarnation": incarnation}, nil, nil)
}

func (session *ClusterSession) Link(ctx context.Context, action, from, to string, delay uint64, symmetric bool) error {
	return session.command(ctx, map[string]any{"type": "control", "command": "link", "action": action, "from": from, "to": to, "delay_nanos": delay, "symmetric": symmetric}, nil, nil)
}

func (session *ClusterSession) Finish(ctx context.Context) (ClusterResult, error) {
	session.mu.Lock()
	if session.finishing {
		session.mu.Unlock()
		select {
		case <-session.done:
			return session.result, session.err
		case <-ctx.Done():
			return ClusterResult{}, ctx.Err()
		}
	}
	for _, state := range session.states {
		select {
		case <-state.done:
		default:
			session.mu.Unlock()
			return ClusterResult{}, invalid("cluster finish has a live incarnation")
		}
	}
	session.finishing = true
	session.mu.Unlock()
	if err := session.command(ctx, map[string]any{"type": "finish"}, nil, nil); err != nil {
		session.mu.Lock()
		if !session.terminal && session.err == nil {
			session.finishing = false
		}
		session.mu.Unlock()
		return ClusterResult{}, err
	}
	select {
	case <-session.done:
		return session.result, session.err
	case <-ctx.Done():
		return ClusterResult{}, ctx.Err()
	}
}

func (session *ClusterSession) Shutdown(ctx context.Context) error {
	session.mu.Lock()
	var live []BrokerNode
	for handle, state := range session.states {
		select {
		case <-state.done:
		default:
			live = append(live, handle)
		}
	}
	session.mu.Unlock()
	for _, handle := range live {
		if err := session.Revoke(ctx, "stop", handle.ID, handle.Incarnation); err != nil {
			if _, waitErr := session.Wait(ctx, handle.ID, handle.Incarnation); waitErr == nil {
				continue
			}
			session.cancel()
			session.input.closeInput()
			<-session.done
			return err
		}
	}
	for _, handle := range live {
		if _, err := session.Wait(ctx, handle.ID, handle.Incarnation); err != nil {
			session.cancel()
			session.input.closeInput()
			<-session.done
			return err
		}
	}
	_, err := session.Finish(ctx)
	return err
}

type clusterStream struct {
	session *ClusterSession
	buffer  []byte
}

func (stream *clusterStream) Write(data []byte) (int, error) {
	written := len(data)
	for len(data) > 0 {
		n := bytes.IndexByte(data, '\n')
		complete := n >= 0
		if complete {
			n++
		} else {
			n = len(data)
		}
		if len(stream.buffer)+n > frameLimit {
			stream.fail(invalid("cluster protocol frame size"))
			return written, nil
		}
		stream.buffer = append(stream.buffer, data[:n]...)
		data = data[n:]
		if complete {
			if err := stream.frame(stream.buffer); err != nil {
				stream.fail(err)
				return written, nil
			}
			stream.buffer = stream.buffer[:0]
		}
	}
	return written, nil
}

func (stream *clusterStream) fail(err error) {
	s := stream.session
	s.mu.Lock()
	if s.err == nil {
		s.err = err
	}
	s.mu.Unlock()
	s.input.closeInput()
	s.cancel()
}

func (stream *clusterStream) frame(data []byte) error {
	frame, err := newValidatedFrame(data)
	if err != nil {
		return err
	}
	var kind string
	if err := json.Unmarshal(frame.fields["type"], &kind); err != nil {
		return invalid("cluster frame type")
	}
	s := stream.session
	s.mu.Lock()
	defer s.mu.Unlock()
	if kind == "call" || kind == "sync" {
		s.counts.GuestCallbacks++
		return invalid("guest callback in cluster transport")
	}
	if s.terminal {
		return invalid("cluster frame after terminal")
	}
	if raw, tagged := frame.fields["instance"]; tagged {
		var node string
		if json.Unmarshal(raw, &node) != nil || node == "" {
			return invalid("cluster instance tag")
		}
		incarnation := uint64(1)
		if raw, exists := frame.fields["incarnation"]; exists {
			if json.Unmarshal(raw, &incarnation) != nil || incarnation < 2 {
				return invalid("cluster incarnation tag")
			}
		}
		state := s.states[BrokerNode{ID: node, Incarnation: incarnation}]
		if state == nil {
			return invalid("unattributable cluster frame")
		}
		select {
		case <-state.done:
			return invalid("frame after incarnation terminal")
		default:
		}
		delete(frame.fields, "instance")
		delete(frame.fields, "incarnation")
		frame.data, err = json.Marshal(frame.fields)
		if err != nil {
			return err
		}
		if kind == "progress" {
			var progress struct {
				Type   string `json:"type"`
				Stream string `json:"stream"`
				Data   []byte `json:"data_base64"`
			}
			if err := frame.decode(&progress, "type", "stream", "data_base64"); err != nil {
				return err
			}
			if progress.Stream != "stdout" || state.sink.request.StdoutMirror == nil || len(progress.Data) == 0 || len(progress.Data) > 64<<10 || uint64(len(progress.Data)) > state.sink.request.Config.Limits.OutputBytes-state.progressBytes {
				return invalid("cluster progress frame")
			}
			if state.progress == nil {
				state.progress = sha256.New()
			}
			_, _ = state.progress.Write(progress.Data)
			state.progressBytes += uint64(len(progress.Data))
			_, _ = state.sink.request.StdoutMirror.Write(progress.Data)
			return nil
		}
		if kind == "evidence" {
			var evidence struct {
				Type   string `json:"type"`
				Stream string `json:"stream"`
				Data   []byte `json:"data_base64"`
			}
			if err := frame.decode(&evidence, "type", "stream", "data_base64"); err != nil {
				return err
			}
			if evidence.Stream == "network" {
				if len(evidence.Data) == 0 || len(evidence.Data) > 64<<10 || state.sink.request.Config.NetworkV2 == nil || uint64(len(state.network)+len(evidence.Data)) > state.sink.request.Config.Limits.TranscriptBytes {
					return invalid("cluster network evidence bounds")
				}
				state.network = append(state.network, evidence.Data...)
				return nil
			}
			return state.sink.receiveHelperEvidence(frame)
		}
		if kind != "result" {
			return invalid("unexpected instance frame")
		}
		if err := state.collect(frame, s.config.Broker); err != nil {
			return err
		}
		state.sink = nil
		state.network = nil
		state.progress = nil
		s.counts.Results++
		close(state.done)
		return nil
	}
	switch kind {
	case "cluster_network":
		if !s.finishing || s.config.Broker.Network == nil {
			return invalid("unsolicited final cluster network")
		}
		var evidence struct {
			Type   string `json:"type"`
			Source string `json:"source"`
			Data   []byte `json:"data_base64"`
		}
		if err := frame.decode(&evidence, "type", "source", "data_base64"); err != nil {
			return err
		}
		boot := s.boots[evidence.Source]
		if boot == nil || boot.request.Config.NetworkV2 == nil || s.networkSource != "" && s.networkSource != evidence.Source || len(evidence.Data) == 0 || len(evidence.Data) > 64<<10 || uint64(len(s.networkWire)+len(evidence.Data)) > boot.request.Config.Limits.TranscriptBytes {
			return invalid("final cluster network bounds")
		}
		s.networkSource = evidence.Source
		s.networkWire = append(s.networkWire, evidence.Data...)
		return nil
	case "boot_started":
		var bootID string
		if err := json.Unmarshal(frame.fields["boot"], &bootID); err != nil {
			return invalid("cluster boot started tag")
		}
		boot := s.boots[bootID]
		if boot == nil || boot.started != nil {
			return invalid("unknown or duplicate cluster boot started")
		}
		delete(frame.fields, "boot")
		frame.fields["type"] = json.RawMessage(`"started"`)
		frame.data, err = json.Marshal(frame.fields)
		if err != nil {
			return err
		}
		var started Started
		if err := frame.decode(&started, "schema", "type", "engine", "module_sha256", "imports", "exports", "initial_memory_pages"); err != nil {
			return err
		}
		if err := validateClusterStarted(started, boot.request); err != nil {
			return err
		}
		boot.started = &started
		return nil
	case "started":
		select {
		case <-s.ready:
			return invalid("duplicate cluster started")
		default:
		}
		var started Started
		if err := frame.decode(&started, "schema", "type", "engine", "module_sha256", "imports", "exports", "initial_memory_pages"); err != nil {
			return err
		}
		for _, boot := range s.boots {
			if boot.started == nil {
				return invalid("missing cluster boot identity")
			}
		}
		if err := validateClusterStarted(started, s.config.Instances[0].Request); err != nil {
			return err
		}
		value, _ := json.Marshal(map[string]any{"type": "authorize", "module_sha256": started.ModuleSHA256, "configuration": started.Engine.Configuration})
		s.authorizedAt = time.Now()
		if _, err := s.input.Write(append(value, '\n')); err != nil {
			return err
		}
		close(s.ready)
		return nil
	case "control_result":
		var ack struct {
			Type        string `json:"type"`
			ID          uint64 `json:"id"`
			Termination string `json:"termination"`
			Message     string `json:"message"`
		}
		if err := frame.decode(&ack, "type", "id", "termination", "message"); err != nil {
			return err
		}
		if len(ack.Message) > 65536 || ack.ID == 0 {
			return invalid("cluster control acknowledgement limits")
		}
		pending := s.pending[ack.ID]
		if pending == nil {
			return invalid("unknown cluster control acknowledgement")
		}
		delete(s.pending, ack.ID)
		if ack.Termination != "" {
			pending <- invalid("cluster control: " + ack.Termination + ": " + ack.Message)
		} else {
			pending <- nil
		}
		return nil
	case "evidence":
		if !s.finishing {
			return invalid("cluster World evidence before finish")
		}
		var evidence struct {
			Type   string `json:"type"`
			Stream string `json:"stream"`
			Data   []byte `json:"data_base64"`
		}
		if err := frame.decode(&evidence, "type", "stream", "data_base64"); err != nil {
			return err
		}
		if evidence.Stream != "world" || len(evidence.Data) == 0 || len(evidence.Data) > 64<<10 || uint64(len(s.worldWire)+len(evidence.Data)) > world.CompressedSnapshotProfile().MaxExpandedBytes {
			return invalid("cluster World evidence bounds")
		}
		s.worldWire = append(s.worldWire, evidence.Data...)
		return nil
	case "cluster_result":
		return s.collect(frame)
	default:
		return invalid("unknown cluster frame")
	}
}

func validateClusterStarted(started Started, request Request) error {
	if started.Schema != "gomad3.wasm-host/v1" || started.Engine != request.Engine || started.ModuleSHA256 != request.ModuleSHA256 || started.InitialMemoryPages != request.InitialMemoryPages || !sameImports(started.Imports, request.Imports) {
		return invalid("cluster engine/module/import identity mismatch before authorization")
	}
	memory, start := false, false
	for _, export := range started.Exports {
		memory = memory || export.Name == "memory" && export.Kind == "memory"
		start = start || export.Name == "_start" && export.Kind == "func"
	}
	if !memory || !start {
		return invalid("cluster guest exports")
	}
	return nil
}

func (state *clusterState) collect(frame *validatedFrame, config BrokerConfig) error {
	var terminal resultFrame
	keys := []string{"type", "termination", "message", "fuel_remaining", "peak_memory_bytes", "output_bytes"}
	for _, key := range []string{"exit_code", "local_model", "divergence", "metrics"} {
		if _, ok := frame.fields[key]; ok {
			keys = append(keys, key)
		}
	}
	if err := frame.decode(&terminal, keys...); err != nil {
		return err
	}
	request := state.sink.request
	switch terminal.Termination {
	case "exit":
		if terminal.ExitCode == nil {
			return invalid("cluster exit code")
		}
	case "trap", "capacity", "unsupported", "invalid", "divergence", "storage_crash", "crash", "stop":
		if terminal.ExitCode != nil {
			return invalid("cluster non-exit code")
		}
	default:
		return invalid("cluster incarnation termination")
	}
	if len(terminal.Message) > 65536 || terminal.FuelRemaining > request.Fuel || terminal.PeakMemoryBytes > request.MemoryBytes || terminal.OutputBytes > request.Config.Limits.OutputBytes || terminal.OutputBytes < uint64(len(state.sink.helperEvidence.stdout)+len(state.sink.helperEvidence.stderr)) {
		return invalid("cluster terminal limits")
	}
	state.sink.terminal = &terminal
	result := Result{Termination: terminal.Termination, ExitCode: terminal.ExitCode, Message: terminal.Message, FuelRemaining: terminal.FuelRemaining, PeakMemoryBytes: terminal.PeakMemoryBytes, OfferedOutputBytes: terminal.OutputBytes, EnvironmentIdentity: state.sink.environment.Identity(), Started: state.sink.started, Reaped: true, Telemetry: terminal.Metrics, AuthorizationToTerminalNanos: uint64(time.Since(state.sink.authorizedAt))}
	result.Stdout, result.Stderr = state.sink.guestOutput()
	if state.sink.request.StdoutMirror != nil {
		digest := sha256.Sum256(result.Stdout)
		if state.progressBytes != uint64(len(result.Stdout)) || state.progress != nil && !bytes.Equal(state.progress.Sum(nil), digest[:]) {
			return invalid("cluster progress does not match stdout")
		}
	}
	interrupted := terminal.Termination == "crash" || terminal.Termination == "stop"
	if interrupted {
		result.Termination = "storage_crash"
	}
	if err := state.sink.collectHelper(&result); err != nil {
		return err
	}
	result.Termination = terminal.Termination
	if interrupted {
		if terminal.LocalModel == nil || terminal.LocalModel.Finished {
			return invalid("cluster completed interrupted runtime")
		}
		if _, err := choice.ProjectInterruptedReplayPlan(state.sink.helperEvidence.choice, request.Runtime.Choice.Limit, request.Runtime.Choice.ExecutionIdentity); err != nil {
			return err
		}
		result.RuntimeInterrupted = true
		result.InterruptedChoiceBytes = state.sink.helperEvidence.choice
		result.InterruptedDiagnosticBytes = state.sink.helperEvidence.diagnostic
		state.sink.terminal.Termination = result.Termination
	}
	if request.Config.Storage != nil {
		if err := state.sink.collectHelperStorage(&result); err != nil {
			return err
		}
	}
	if request.Config.NetworkV2 != nil {
		evidence, err := DecodeClusterNetworkEvidence(state.network, ClusterNetworkLimits{Network: config.Network.Limits, Instance: *request.Config.NetworkV2, TranscriptBytes: request.Config.Limits.TranscriptBytes})
		if err != nil {
			return err
		}
		result.NetworkTranscript = evidence.Document
	}
	state.result = result
	state.executionClassifiedAt = time.Now()
	state.result.AuthorizationToTerminalNanos = uint64(state.executionClassifiedAt.Sub(state.authorizationStartedAt))
	return nil
}

func (s *ClusterSession) collect(frame *validatedFrame) error {
	var terminal struct {
		Type        string `json:"type"`
		Termination string `json:"termination"`
		Message     string `json:"message"`
		Instances   []struct {
			ID          string `json:"id"`
			Incarnation uint64 `json:"incarnation,omitempty"`
			Termination string `json:"termination"`
		} `json:"instances"`
		VirtualElapsedNanos uint64                   `json:"virtual_elapsed_nanos"`
		Metrics             *Telemetry               `json:"metrics,omitempty"`
		Scheduler           *ClusterSchedulerMetrics `json:"scheduler,omitempty"`
	}
	keys := []string{"type", "termination", "message", "instances", "virtual_elapsed_nanos"}
	if _, ok := frame.fields["metrics"]; ok {
		keys = append(keys, "metrics")
	}
	if _, ok := frame.fields["scheduler"]; ok {
		keys = append(keys, "scheduler")
	}
	if err := frame.decode(&terminal, keys...); err != nil {
		return err
	}
	if terminal.Termination != "exit" {
		switch terminal.Termination {
		case "invalid", "infrastructure", "capacity", "unsupported":
			return &BoundaryError{Kind: terminal.Termination, Dimension: "cluster run: " + terminal.Message}
		default:
			return invalid("cluster run termination")
		}
	}
	if !s.finishing || len(terminal.Instances) != len(s.states) {
		return invalid("cluster terminal incarnation count")
	}
	seen := map[BrokerNode]bool{}
	for _, instance := range terminal.Instances {
		incarnation := instance.Incarnation
		if incarnation == 0 {
			incarnation = 1
		}
		handle := BrokerNode{ID: instance.ID, Incarnation: incarnation}
		state := s.states[handle]
		if state == nil || seen[handle] {
			return invalid("cluster terminal incarnation identity")
		}
		seen[handle] = true
		select {
		case <-state.done:
		default:
			return invalid("cluster missing instance result")
		}
		if instance.Termination != state.result.Termination {
			return invalid("cluster terminal incarnation classification")
		}
	}
	snapshot, err := world.DecodeExpandedSnapshot(s.worldWire, world.Config{Seed: world.Seed(s.config.Broker.Seed), Limits: s.config.Broker.Limits})
	if err != nil {
		return invalid("cluster World: " + err.Error())
	}
	model, err := world.Restore(snapshot, nil)
	if err != nil {
		return err
	}
	detached, err := model.DetachCompressedSnapshot(world.CompressedSnapshotProfile())
	if err != nil {
		return err
	}
	encoded, err := detached.Encode()
	if err != nil {
		return err
	}
	s.result = ClusterResult{World: encoded, VirtualElapsedNanos: terminal.VirtualElapsedNanos, Metrics: terminal.Metrics, Scheduler: terminal.Scheduler}
	if s.config.Broker.Network != nil {
		boot := s.boots[s.networkSource]
		if boot == nil {
			return invalid("missing final cluster network evidence")
		}
		evidence, err := DecodeClusterNetworkEvidence(s.networkWire, ClusterNetworkLimits{Network: s.config.Broker.Network.Limits, Instance: *boot.request.Config.NetworkV2, TranscriptBytes: boot.request.Config.Limits.TranscriptBytes})
		if err != nil {
			return err
		}
		s.result.Network = evidence.Record
	}
	s.terminal = true
	s.result.AuthorizationToTerminalNanos = uint64(time.Since(s.authorizedAt))
	return nil
}
