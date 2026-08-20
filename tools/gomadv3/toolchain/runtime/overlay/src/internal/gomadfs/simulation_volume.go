// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gomadfs

import (
	"errors"
	"sort"
	"strconv"
	"sync"
	"syscall"
	_ "unsafe"

	"internal/gomadchoicewire"
	"internal/gomadsim"
)

var ErrSimulationVolumeReplayDiverged = errors.New("simulation volume replay diverged")

type simulationVolumeConfig struct {
	Seed   uint64
	Limits simulationVolumeLimits
	Nodes  []simulationVolumeNodeConfig
	Replay *simulationVolumeRecord
}

type simulationVolumeLimits struct {
	Operations  uint64
	Transitions uint64
}

type simulationVolumeNodeConfig struct {
	Node    string
	Volumes []VolumeConfig
}

type simulationVolumeHandle struct {
	Node        string
	Incarnation uint64
}

type simulationVolumeTransition struct {
	Ordinal            uint64
	Kind               string
	Handle             simulationVolumeHandle
	Volume             string
	Operation          uint64
	Dependencies       []uint64
	SelectedOperations []uint64
	Path               string
	Destination        string
	Inode              uint64
	Offset             uint64
	Bytes              uint64
	PayloadSHA256      string
	EffectSHA256       string
	Outcome            string
}

type simulationVolumeStateSnapshot struct {
	Volume            string
	Node              string
	Mount             string
	CapacityBytes     uint64
	Persisted         []SnapshotEntry
	Volatile          []SnapshotEntry
	PendingOperations uint64
	PendingSHA256     string
	NextOperation     uint64
	Identity          string
}

type simulationVolumeSnapshot struct {
	Volumes          []simulationVolumeStateSnapshot
	TransitionSHA256 string
	Identity         string
}

type simulationVolumeRecord struct {
	Transitions []simulationVolumeTransition
	Snapshot    simulationVolumeSnapshot
}

type simulationVolumeBridgeError struct {
	Kind           string
	Message        string
	Ordinal        uint64
	ExpectedSHA256 string
	ActualSHA256   string
	Expected       *simulationVolumeTransition
	Actual         *simulationVolumeTransition
}

type simulationVolumeNode struct {
	node          string
	filesystem    *FS
	activeDomain  uint64
	incarnation   uint64
	domainHistory []uint64
}

type simulationVolumeRun struct {
	sync.Mutex
	run         uint64
	seed        uint64
	limits      simulationVolumeLimits
	nodes       map[string]*simulationVolumeNode
	ordered     []string
	transitions []simulationVolumeTransition
	replay      *simulationVolumeRecord
	replayLanes map[string][]simulationVolumeTransition
	replayNext  map[string]int
	divergence  *simulationVolumeBridgeError
	finished    bool
}

type simulationVolumeObserver struct {
	run         *simulationVolumeRun
	node        string
	incarnation uint64
}

var simulationVolumeState = struct {
	sync.Mutex
	runs map[uint64]*simulationVolumeRun
}{runs: make(map[uint64]*simulationVolumeRun)}

//go:linkname BeginSimulationVolumes
func BeginSimulationVolumes(run uint64, encoded []byte) ([]byte, bool) {
	config, err := decodeSimulationVolumeConfig(encoded)
	if err != nil {
		return encodeSimulationVolumeBridgeError(err), false
	}
	volumes, err := newSimulationVolumeRun(run, config)
	if err != nil {
		return encodeSimulationVolumeBridgeError(err), false
	}
	simulationVolumeState.Lock()
	defer simulationVolumeState.Unlock()
	if run == 0 || simulationVolumeState.runs[run] != nil {
		return encodeSimulationVolumeBridgeError(errors.New("simulation volume run is invalid or duplicated")), false
	}
	simulationVolumeState.runs[run] = volumes
	return nil, true
}

//go:linkname RegisterSimulationVolumes
func RegisterSimulationVolumes(domainToken uint64) ([]byte, bool) {
	domain, ok := gomadsim.DescribeNetworkDomain(domainToken)
	if !ok {
		return encodeSimulationVolumeBridgeError(syscall.ESTALE), false
	}
	run := simulationVolumeRunFor(domain.Run)
	if run == nil {
		return encodeSimulationVolumeBridgeError(errors.New("simulation volume run is unavailable")), false
	}
	if err := run.register(domain); err != nil {
		return encodeSimulationVolumeBridgeError(err), false
	}
	return nil, true
}

//go:linkname RevokeSimulationVolumes
func RevokeSimulationVolumes(domainToken uint64, graceful, persistedOnly bool) ([]byte, bool) {
	domain, ok := gomadsim.DescribeNetworkDomain(domainToken)
	if !ok {
		return encodeSimulationVolumeBridgeError(syscall.ESTALE), false
	}
	run := simulationVolumeRunFor(domain.Run)
	if run == nil {
		return encodeSimulationVolumeBridgeError(errors.New("simulation volume run is unavailable")), false
	}
	if graceful && persistedOnly {
		return encodeSimulationVolumeBridgeError(errors.New("graceful volume revocation cannot select persisted-only crash state")), false
	}
	if err := run.revoke(domain, graceful, persistedOnly); err != nil {
		return encodeSimulationVolumeBridgeError(err), false
	}
	return nil, true
}

//go:linkname EnumerateSimulationVolume
func EnumerateSimulationVolume(domainToken uint64, volume string, states, operations, depth, bytes, wallNanos uint64, encodedFrontier []byte) ([]byte, bool) {
	domain, ok := gomadsim.DescribeNetworkDomain(domainToken)
	if !ok {
		return encodeSimulationVolumeBridgeError(syscall.ESTALE), false
	}
	run := simulationVolumeRunFor(domain.Run)
	if run == nil {
		return encodeSimulationVolumeBridgeError(errors.New("simulation volume run is unavailable")), false
	}
	frontier, err := decodeSimulationCrashFrontier(encodedFrontier)
	if err != nil {
		return encodeSimulationVolumeBridgeError(err), false
	}
	page, err := run.enumerate(domain, volume, CrashEnumerationLimits{States: states, Operations: operations, Depth: depth, Bytes: bytes, WallNanos: wallNanos}, frontier)
	if err != nil {
		return encodeSimulationVolumeBridgeError(err), false
	}
	encoded, err := encodeSimulationCrashEnumeration(page)
	if err != nil {
		return encodeSimulationVolumeBridgeError(err), false
	}
	return encoded, true
}

//go:linkname FinishSimulationVolumes
func FinishSimulationVolumes(runToken uint64) ([]byte, bool) {
	simulationVolumeState.Lock()
	run := simulationVolumeState.runs[runToken]
	if run != nil {
		delete(simulationVolumeState.runs, runToken)
	}
	simulationVolumeState.Unlock()
	if run == nil {
		return encodeSimulationVolumeBridgeError(errors.New("simulation volume run is unavailable")), false
	}
	record, finishErr := run.finish()
	var bridge *simulationVolumeBridgeError
	if finishErr != nil && !errors.As(finishErr, &bridge) {
		bridge = &simulationVolumeBridgeError{Kind: "runtime", Message: finishErr.Error()}
	}
	encoded, err := encodeSimulationVolumeFinish(record, bridge)
	if err != nil {
		return encodeSimulationVolumeBridgeError(err), false
	}
	return encoded, true
}

func newSimulationVolumeRun(run uint64, config simulationVolumeConfig) (*simulationVolumeRun, error) {
	if run == 0 || config.Limits.Operations == 0 || config.Limits.Transitions == 0 {
		return nil, errors.New("simulation volume limits are invalid")
	}
	result := &simulationVolumeRun{
		run: run, seed: config.Seed, limits: config.Limits,
		nodes: make(map[string]*simulationVolumeNode, len(config.Nodes)), replay: config.Replay,
	}
	if config.Replay != nil {
		result.replayLanes = make(map[string][]simulationVolumeTransition)
		result.replayNext = make(map[string]int)
		for _, transition := range config.Replay.Transitions {
			lane := simulationVolumeTransitionLane(transition)
			result.replayLanes[lane] = append(result.replayLanes[lane], transition)
		}
	}
	previous := ""
	for _, configNode := range config.Nodes {
		if configNode.Node <= previous {
			return nil, errors.New("simulation volume nodes are invalid or unordered")
		}
		filesystem := NewSimulation()
		if err := filesystem.ConfigureVolumes(configNode.Volumes, VolumeLimits{PendingOperations: config.Limits.Operations, Transitions: config.Limits.Transitions}); err != nil {
			return nil, err
		}
		result.nodes[configNode.Node] = &simulationVolumeNode{node: configNode.Node, filesystem: filesystem}
		result.ordered = append(result.ordered, configNode.Node)
		previous = configNode.Node
	}
	return result, nil
}

func simulationVolumeRunFor(run uint64) *simulationVolumeRun {
	simulationVolumeState.Lock()
	defer simulationVolumeState.Unlock()
	return simulationVolumeState.runs[run]
}

func (run *simulationVolumeRun) register(domain gomadsim.NetworkDomain) error {
	run.Lock()
	defer run.Unlock()
	if run.finished {
		return syscall.ESTALE
	}
	node := run.nodes[domain.Node]
	if node == nil || node.activeDomain != 0 || domain.Incarnation <= node.incarnation {
		return syscall.ESTALE
	}
	observer := &simulationVolumeObserver{run: run, node: domain.Node, incarnation: domain.Incarnation}
	node.filesystem.SetVolumeObserver(observer)
	if !registerSimulationFilesystem(domain.Token, node.filesystem) {
		return syscall.ESTALE
	}
	node.activeDomain = domain.Token
	node.incarnation = domain.Incarnation
	node.domainHistory = append(node.domainHistory, domain.Token)
	return nil
}

func (run *simulationVolumeRun) revoke(domain gomadsim.NetworkDomain, graceful, persistedOnly bool) error {
	run.Lock()
	node := run.nodes[domain.Node]
	if run.finished || node == nil || node.activeDomain != domain.Token || node.incarnation != domain.Incarnation {
		run.Unlock()
		return syscall.ESTALE
	}
	filesystem := node.filesystem
	seed := run.crashSeed(domain.Node, domain.Incarnation)
	diverged := run.divergence != nil
	run.Unlock()

	revoked, ok := revokeSimulationFilesystem(domain.Token)
	if !ok || revoked != filesystem {
		return syscall.ESTALE
	}
	var selections map[string][]uint64
	if !graceful {
		if persistedOnly {
			selections = make(map[string][]uint64)
		} else {
			selections = filesystem.SelectCrashOperations(seed)
		}
	}
	var next *FS
	var err error
	if diverged {
		next, err = filesystem.AdvanceVolumeLifecycleAfterDivergence(graceful, selections)
	} else {
		next, err = filesystem.AdvanceVolumeLifecycle(graceful, selections)
	}
	if err != nil {
		if !restoreSimulationFilesystem(domain.Token, filesystem) {
			return errors.Join(err, errors.New("restore simulation volume binding"))
		}
		return err
	}
	revokeProcessVolumeResources(domain.Token)
	run.Lock()
	defer run.Unlock()
	if node.activeDomain != domain.Token || node.filesystem != filesystem {
		return syscall.ESTALE
	}
	node.filesystem = next
	node.activeDomain = 0
	return nil
}

func (run *simulationVolumeRun) crashSeed(node string, incarnation uint64) uint64 {
	hasher := gomadchoicewire.NewHasher()
	writeHashString(hasher, "gomadv3-volume-crash-selection/v1")
	writeHashUint64(hasher, run.seed)
	writeHashString(hasher, node)
	writeHashUint64(hasher, incarnation)
	digest := hasher.Sum()
	return uint64(digest[0]) | uint64(digest[1])<<8 | uint64(digest[2])<<16 | uint64(digest[3])<<24 |
		uint64(digest[4])<<32 | uint64(digest[5])<<40 | uint64(digest[6])<<48 | uint64(digest[7])<<56
}

func (run *simulationVolumeRun) enumerate(domain gomadsim.NetworkDomain, volume string, limits CrashEnumerationLimits, frontier *CrashFrontier) (CrashEnumeration, error) {
	run.Lock()
	node := run.nodes[domain.Node]
	if run.finished || node == nil || node.activeDomain != domain.Token || node.incarnation != domain.Incarnation {
		run.Unlock()
		return CrashEnumeration{}, syscall.ESTALE
	}
	filesystem := node.filesystem
	run.Unlock()
	return filesystem.EnumerateCrashStates(volume, limits, frontier)
}

func (observer *simulationVolumeObserver) BeforeVolumeOperations(volume string, operations []Operation) error {
	transitions := make([]simulationVolumeTransition, 0, len(operations))
	for _, operation := range operations {
		kind := operation.Kind
		if kind == "alloc" {
			kind = "allocate"
		}
		transitions = append(transitions, simulationVolumeTransition{
			Kind: kind, Handle: simulationVolumeHandle{Node: observer.node, Incarnation: observer.incarnation}, Volume: volume,
			Operation: operation.ID, Dependencies: append([]uint64(nil), operation.Dependencies...), Path: operation.Path,
			Destination: operation.Destination, Inode: operation.Inode, Offset: operation.Offset, Bytes: operation.Bytes,
			PayloadSHA256: operation.PayloadSHA256, EffectSHA256: operation.EffectSHA256, Outcome: "ok",
		})
	}
	return observer.run.commitTransitions(transitions)
}

func (observer *simulationVolumeObserver) BeforeVolumeControl(volume, kind string, selected []uint64) error {
	return observer.run.commitTransitions([]simulationVolumeTransition{{
		Kind: kind, Handle: simulationVolumeHandle{Node: observer.node, Incarnation: observer.incarnation}, Volume: volume,
		SelectedOperations: append([]uint64(nil), selected...), Outcome: "ok",
	}})
}

func (run *simulationVolumeRun) commitTransitions(transitions []simulationVolumeTransition) error {
	run.Lock()
	defer run.Unlock()
	if run.finished {
		return syscall.ESTALE
	}
	required := uint64(len(run.transitions)) + uint64(len(transitions))
	if required > run.limits.Transitions {
		return &VolumeCapacityError{Resource: "transitions", Required: required, Maximum: run.limits.Transitions}
	}
	next := make(map[string]int, len(run.replayNext))
	for lane, consumed := range run.replayNext {
		next[lane] = consumed
	}
	if run.replay != nil {
		for index := range transitions {
			transition := transitions[index]
			lane := simulationVolumeTransitionLane(transition)
			expectedLane := run.replayLanes[lane]
			consumed := next[lane]
			var expected simulationVolumeTransition
			if consumed < len(expectedLane) {
				expected = expectedLane[consumed]
			}
			if consumed >= len(expectedLane) || !sameSimulationVolumeTransition(expected, transition) {
				ordinal := expected.Ordinal
				if expected.Kind == "" {
					ordinal = uint64(len(run.replay.Transitions))
				}
				transition.Ordinal = ordinal
				replayErr := newSimulationVolumeReplayError(ordinal, expected, transition)
				run.divergence = replayErr
				return replayErr
			}
			next[lane] = consumed + 1
		}
	}
	run.replayNext = next
	for index := range transitions {
		transitions[index].Ordinal = 0
		run.transitions = append(run.transitions, transitions[index])
	}
	return nil
}

func (run *simulationVolumeRun) finish() (simulationVolumeRecord, error) {
	run.Lock()
	defer run.Unlock()
	if run.finished {
		return simulationVolumeRecord{}, errors.New("simulation volume run is already finished")
	}
	run.finished = true
	var finishErr error
	if run.divergence != nil {
		finishErr = run.divergence
	}
	canonicalizeSimulationVolumeTransitions(run.transitions)
	if finishErr == nil && run.replay != nil && len(run.transitions) != len(run.replay.Transitions) {
		expected := run.firstMissingReplayTransitionLocked()
		finishErr = newSimulationVolumeReplayError(expected.Ordinal, expected, simulationVolumeTransition{})
	}
	snapshot := simulationVolumeSnapshot{}
	for _, nodeID := range run.ordered {
		node := run.nodes[nodeID]
		for _, volume := range node.filesystem.VolumeSnapshots() {
			snapshot.Volumes = append(snapshot.Volumes, simulationVolumeStateSnapshot{
				Volume: volume.ID, Node: nodeID, Mount: volume.Mount, CapacityBytes: volume.CapacityBytes,
				Persisted: volume.Persisted, Volatile: volume.Volatile, PendingOperations: volume.PendingOperations,
				PendingSHA256: volume.PendingSHA256, NextOperation: volume.NextOperation, Identity: volume.Identity,
			})
		}
	}
	snapshot.TransitionSHA256 = simulationVolumeTransitionsIdentity(run.transitions)
	snapshot.Identity = simulationVolumeSnapshotIdentity(snapshot)
	record := simulationVolumeRecord{Transitions: append([]simulationVolumeTransition(nil), run.transitions...), Snapshot: snapshot}
	if finishErr == nil && run.replay != nil && run.replay.Snapshot.Identity != snapshot.Identity {
		finishErr = &simulationVolumeBridgeError{
			Kind: "replay", Message: ErrSimulationVolumeReplayDiverged.Error(), Ordinal: uint64(len(run.transitions)),
			ExpectedSHA256: run.replay.Snapshot.Identity, ActualSHA256: snapshot.Identity,
		}
	}
	var domains []uint64
	for _, nodeID := range run.ordered {
		domains = append(domains, run.nodes[nodeID].domainHistory...)
	}
	removeSimulationFilesystems(domains)
	return record, finishErr
}

func canonicalizeSimulationVolumeTransitions(transitions []simulationVolumeTransition) {
	sort.SliceStable(transitions, func(left, right int) bool {
		return simulationVolumeTransitionLane(transitions[left]) < simulationVolumeTransitionLane(transitions[right])
	})
	for index := range transitions {
		transitions[index].Ordinal = uint64(index)
	}
}

func simulationVolumeTransitionLane(transition simulationVolumeTransition) string {
	return transition.Handle.Node + "\x00" + transition.Volume
}

func sameSimulationVolumeTransition(left, right simulationVolumeTransition) bool {
	left.Ordinal = 0
	right.Ordinal = 0
	if left.Kind != right.Kind || left.Handle != right.Handle || left.Volume != right.Volume || left.Operation != right.Operation || left.Path != right.Path || left.Destination != right.Destination || left.Inode != right.Inode || left.Offset != right.Offset || left.Bytes != right.Bytes || left.PayloadSHA256 != right.PayloadSHA256 || left.EffectSHA256 != right.EffectSHA256 || left.Outcome != right.Outcome {
		return false
	}
	return equalUint64s(left.Dependencies, right.Dependencies) && equalUint64s(left.SelectedOperations, right.SelectedOperations)
}

func equalUint64s(left, right []uint64) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func (run *simulationVolumeRun) firstMissingReplayTransitionLocked() simulationVolumeTransition {
	remaining := make(map[string]int, len(run.replayNext))
	for lane, consumed := range run.replayNext {
		remaining[lane] = consumed
	}
	for _, transition := range run.replay.Transitions {
		lane := simulationVolumeTransitionLane(transition)
		if remaining[lane] != 0 {
			remaining[lane]--
			continue
		}
		return transition
	}
	return simulationVolumeTransition{Ordinal: uint64(len(run.replay.Transitions))}
}

func newSimulationVolumeReplayError(ordinal uint64, expected, actual simulationVolumeTransition) *simulationVolumeBridgeError {
	bridge := &simulationVolumeBridgeError{
		Kind: "replay", Message: ErrSimulationVolumeReplayDiverged.Error(), Ordinal: ordinal,
		ExpectedSHA256: simulationVolumeTransitionIdentity(expected), ActualSHA256: simulationVolumeTransitionIdentity(actual),
	}
	if expected.Kind != "" {
		value := expected
		bridge.Expected = &value
	}
	if actual.Kind != "" {
		value := actual
		bridge.Actual = &value
	}
	return bridge
}

func simulationVolumeTransitionIdentity(transition simulationVolumeTransition) string {
	hasher := gomadchoicewire.NewHasher()
	writeHashString(hasher, "gomadv3-volume-transition/v1")
	writeSimulationVolumeTransitionHash(hasher, transition)
	return hashDigest(hasher)
}

func simulationVolumeTransitionsIdentity(transitions []simulationVolumeTransition) string {
	hasher := gomadchoicewire.NewHasher()
	writeHashString(hasher, "gomadv3-volume-transitions/v1")
	writeHashUint64(hasher, uint64(len(transitions)))
	for _, transition := range transitions {
		writeSimulationVolumeTransitionHash(hasher, transition)
	}
	return hashDigest(hasher)
}

func writeSimulationVolumeTransitionHash(hasher *gomadchoicewire.Hasher, transition simulationVolumeTransition) {
	writeHashUint64(hasher, transition.Ordinal)
	writeHashString(hasher, transition.Kind)
	writeHashString(hasher, transition.Handle.Node)
	writeHashUint64(hasher, transition.Handle.Incarnation)
	writeHashString(hasher, transition.Volume)
	writeHashUint64(hasher, transition.Operation)
	writeHashUint64(hasher, uint64(len(transition.Dependencies)))
	for _, dependency := range transition.Dependencies {
		writeHashUint64(hasher, dependency)
	}
	writeHashUint64(hasher, uint64(len(transition.SelectedOperations)))
	for _, selected := range transition.SelectedOperations {
		writeHashUint64(hasher, selected)
	}
	writeHashString(hasher, transition.Path)
	writeHashString(hasher, transition.Destination)
	writeHashUint64(hasher, transition.Inode)
	writeHashUint64(hasher, transition.Offset)
	writeHashUint64(hasher, transition.Bytes)
	writeHashString(hasher, transition.PayloadSHA256)
	writeHashString(hasher, transition.EffectSHA256)
	writeHashString(hasher, transition.Outcome)
}

func simulationVolumeSnapshotIdentity(snapshot simulationVolumeSnapshot) string {
	hasher := gomadchoicewire.NewHasher()
	writeHashString(hasher, "gomadv3-volume-run-snapshot/v1")
	writeHashString(hasher, snapshot.TransitionSHA256)
	writeHashUint64(hasher, uint64(len(snapshot.Volumes)))
	for _, volume := range snapshot.Volumes {
		writeHashString(hasher, volume.Node)
		writeHashString(hasher, volume.Volume)
		writeHashString(hasher, volume.Identity)
	}
	return hashDigest(hasher)
}

func (err *simulationVolumeBridgeError) Error() string {
	if err.Message != "" {
		return err.Message
	}
	return "simulation volume bridge error " + strconv.FormatUint(err.Ordinal, 10)
}

func (err *simulationVolumeBridgeError) GomadSimulationVolumeReplayDivergence() []byte {
	if err.Kind != "replay" {
		return nil
	}
	return encodeSimulationVolumeBridgeError(err)
}
