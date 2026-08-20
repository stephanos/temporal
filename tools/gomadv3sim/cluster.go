package gomadv3sim

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"
)

type bootCompletion struct {
	done chan struct{}
	err  error
}

type nodeOperation struct {
	action LifecycleAction
	done   chan struct{}
	err    error
}

type transitionEntry struct {
	transition LifecycleTransition
	resolved   bool
}

type clusterNode struct {
	spec         NodeSpec
	state        NodeState
	handle       NodeHandle
	boot         BootFunc
	cancel       context.CancelFunc
	completion   *bootCompletion
	operation    *nodeOperation
	domain       uint64
	reason       string
	modelActive  uint64
	modelStarted uint64
	modelClosing bool
}

type inProcessCluster struct {
	mu                    sync.Mutex
	backend               Backend
	seed                  uint64
	limits                Limits
	nodes                 map[NodeID]*clusterNode
	ordered               []NodeID
	transitions           []*transitionEntry
	incarnations          []NodeResult
	leaks                 []LeakDiagnostic
	replay                *ReplayPlan
	specSHA256            string
	runtimeRun            uint64
	closing               bool
	activeCalls           uint64
	pendingOps            uint64
	activity              chan struct{}
	replayFailure         error
	faultPlan             FaultPlan
	faults                []FaultRealization
	faultPending          bool
	faultOccurrences      map[string]uint64
	scenarios             []ScenarioDecision
	scenarioOccurrences   map[string]uint64
	history               []HistoryOperation
	observations          []Observation
	oracles               []OracleResult
	scenarioEvidenceBytes uint64
	processOutputs        []OutputObservation
}

func Run(ctx context.Context, spec Spec, scenario Scenario) (Result, error) {
	if ctx == nil {
		return Result{}, errors.New("simulation context is nil")
	}
	if err := ValidateSpec(spec); err != nil {
		return Result{}, err
	}
	if runPrivateProcessNodeIfPresent(ctx, spec) {
		return Result{}, nil
	}
	if spec.Backend == BackendProcess && (!processBackendAvailable() || processBackendRole() != processRoleCoordinator) {
		return Result{}, &BackendUnavailableError{Backend: spec.Backend}
	}
	if scenario == nil {
		return Result{}, errors.New("simulation scenario is nil")
	}
	cluster, err := newInProcessCluster(spec)
	if err != nil {
		return Result{}, err
	}
	if !runtimeDomainAvailable() {
		return Result{}, ErrRuntimeUnavailable
	}
	runtimeRun, err := runtimeDomainBegin(spec.Limits.ObservationBytes, spec.Limits.ScenarioActions)
	if err != nil {
		return Result{}, err
	}
	cluster.runtimeRun = runtimeRun
	networkConfig, err := encodeRuntimeNetworkConfig(spec)
	if err != nil {
		_, finishErr := runtimeDomainFinish(runtimeRun)
		return Result{}, errors.Join(err, finishErr)
	}
	if err := runtimeNetworkBegin(runtimeRun, networkConfig); err != nil {
		_, finishErr := runtimeDomainFinish(runtimeRun)
		return Result{}, errors.Join(fmt.Errorf("begin simulation network: %w", err), finishErr)
	}
	volumeConfig, err := encodeRuntimeVolumeConfig(spec)
	if err != nil {
		_, networkFinishErr := runtimeNetworkFinish(runtimeRun)
		_, finishErr := runtimeDomainFinish(runtimeRun)
		return Result{}, errors.Join(err, networkFinishErr, finishErr)
	}
	if err := runtimeVolumeBegin(runtimeRun, volumeConfig); err != nil {
		_, networkFinishErr := runtimeNetworkFinish(runtimeRun)
		_, finishErr := runtimeDomainFinish(runtimeRun)
		return Result{}, errors.Join(fmt.Errorf("begin simulation volumes: %w", err), networkFinishErr, finishErr)
	}
	stopProcessModelBroker := func() error { return nil }
	if spec.Backend == BackendProcess {
		stopProcessModelBroker, err = beginProcessModelBroker(cluster.handleProcessModelOperation)
		if err != nil {
			_, volumeFinishErr := runtimeVolumeFinish(runtimeRun)
			_, networkFinishErr := runtimeNetworkFinish(runtimeRun)
			_, finishErr := runtimeDomainFinish(runtimeRun)
			return Result{}, errors.Join(err, volumeFinishErr, networkFinishErr, finishErr)
		}
	}
	scenarioErr := scenario(ctx, cluster)
	controllerErr := cluster.finishControllers()
	scenarioErr = errors.Join(scenarioErr, controllerErr)
	cleanupErr := cluster.shutdown(ctx)
	modelBrokerErr := stopProcessModelBroker()
	volumes, volumeFinishErr := runtimeVolumeFinish(runtimeRun)
	network, networkFinishErr := runtimeNetworkFinish(runtimeRun)
	outputs, finishErr := runtimeDomainFinish(runtimeRun)
	if err := errors.Join(cleanupErr, modelBrokerErr, finishErr); err != nil {
		return Result{}, errors.Join(fmt.Errorf("finish simulation: %w", err), scenarioErr)
	}
	result, err := cluster.result()
	if err != nil {
		return Result{}, err
	}
	result.Outputs = outputs
	result.Outputs = append(result.Outputs, cluster.processOutputs...)
	sort.Slice(result.Outputs, func(left, right int) bool { return outputBefore(result.Outputs[left], result.Outputs[right]) })
	result.Network = network
	result.Volumes = volumes
	if !applyReplayDivergence(&result, scenarioErr) && !applyReplayDivergence(&result, volumeFinishErr) && !applyReplayDivergence(&result, networkFinishErr) {
		if volumeFinishErr != nil {
			return Result{}, fmt.Errorf("finish simulation volumes: %w", volumeFinishErr)
		}
		if networkFinishErr != nil {
			return Result{}, fmt.Errorf("finish simulation network: %w", networkFinishErr)
		}
		if scenarioErr != nil {
			result.Outcome = OutcomeScenarioFailed
			result.Reason = boundedTerminalText(scenarioErr.Error())
			result.FailureIdentity, err = normalizedFailureIdentity(result.Outcome, result.Reason, "")
			if err != nil {
				return Result{}, err
			}
		} else if failed := firstFailedOracle(result.Oracles); failed != nil {
			result.Outcome = OutcomeOracleFailed
			result.Reason = boundedTerminalText("oracle " + failed.Name + " failed")
			result.FailureIdentity = failed.FailureIdentity
		} else {
			result.Outcome = OutcomeCompleted
		}
		if err := cluster.finishReplay(result); err != nil {
			if !applyReplayDivergence(&result, err) {
				return Result{}, err
			}
		}
	}
	record, err := buildClusterRecord(spec, cluster.specSHA256, result)
	if err != nil {
		return Result{}, err
	}
	if _, err := EncodeClusterRecord(record); err != nil {
		return Result{}, err
	}
	result.Record = record
	return result, nil
}

func (cluster *inProcessCluster) handleProcessModelOperation(request processModelRequest) ([]byte, error) {
	domain, err := cluster.beginProcessModelOperation(request.Handle)
	if err != nil {
		return nil, err
	}
	cluster.mu.Lock()
	node := cluster.nodes[request.Handle.Node]
	cluster.mu.Unlock()
	defer cluster.finishProcessModelOperation(node)
	previous, err := runtimeDomainEnter(domain)
	if err != nil {
		return nil, err
	}
	defer runtimeDomainLeave(previous)
	if len(request.Payload) < 10 {
		return nil, errors.New("process simulation model operation is unsupported")
	}
	switch request.Payload[9] {
	case 1:
		return runtimeProcessNetworkOperation(domain, request.Payload)
	case 2:
		return runtimeProcessVolumeOperation(domain, request.Payload)
	default:
		return nil, errors.New("process simulation model operation is unsupported")
	}
}

func (cluster *inProcessCluster) beginProcessModelOperation(handle NodeHandle) (uint64, error) {
	cluster.mu.Lock()
	node, err := cluster.currentNode(handle, NodeStateRunning)
	if err != nil {
		cluster.mu.Unlock()
		return 0, err
	}
	if node.modelClosing {
		cluster.mu.Unlock()
		return 0, ErrInvalidTransition
	}
	node.modelActive++
	node.modelStarted++
	domain := node.domain
	cluster.mu.Unlock()
	cluster.notifyActivity()
	return domain, nil
}

func (cluster *inProcessCluster) finishProcessModelOperation(node *clusterNode) {
	cluster.mu.Lock()
	if node.modelActive != 0 {
		node.modelActive--
	}
	cluster.mu.Unlock()
	cluster.notifyActivity()
}

func (cluster *inProcessCluster) waitForProcessModelOperations(node *clusterNode) {
	for {
		cluster.mu.Lock()
		active := node.modelActive != 0
		cluster.mu.Unlock()
		if !active {
			return
		}
		<-cluster.activity
	}
}

func applyReplayDivergence(result *Result, source error) bool {
	divergenceErr := replayDivergence(source)
	if divergenceErr == nil {
		return false
	}
	result.Outcome = OutcomeReplayDiverged
	result.Reason = divergenceErr.Error()
	divergence := divergenceErr.Divergence
	result.Divergence = &divergence
	result.FailureIdentity, _ = normalizedFailureIdentity(OutcomeReplayDiverged, result.Reason, divergenceIdentity(divergence))
	return true
}

func replayDivergence(source error) *ReplayDivergenceError {
	var divergenceErr *ReplayDivergenceError
	if errors.As(source, &divergenceErr) {
		return divergenceErr
	}
	return nil
}

func (cluster *inProcessCluster) retainReplayFailureLocked(source error) error {
	if cluster.replayFailure != nil {
		return cluster.replayFailure
	}
	var classified error
	if divergenceErr := replayDivergence(source); divergenceErr != nil {
		classified = divergenceErr
	}
	if classified == nil {
		classified = runtimeVolumeReplayDivergence(source)
	}
	if classified == nil {
		classified = runtimeNetworkReplayDivergence(source)
	}
	cluster.replayFailure = classified
	return classified
}

func newInProcessCluster(spec Spec) (*inProcessCluster, error) {
	specSHA256, err := hashSpec(spec)
	if err != nil {
		return nil, err
	}
	cluster := &inProcessCluster{
		backend:             spec.Backend,
		seed:                spec.Seed,
		limits:              spec.Limits,
		nodes:               make(map[NodeID]*clusterNode, len(spec.Nodes)),
		ordered:             make([]NodeID, 0, len(spec.Nodes)),
		specSHA256:          specSHA256,
		activity:            make(chan struct{}, 1),
		faultOccurrences:    make(map[string]uint64),
		scenarioOccurrences: make(map[string]uint64),
	}
	if spec.Faults == nil {
		cluster.faultPlan, err = NewFaultPlan(nil)
		if err != nil {
			return nil, err
		}
	} else {
		cluster.faultPlan = *spec.Faults
		cluster.faultPlan.Actions = cloneFaultActions(spec.Faults.Actions)
	}
	if spec.Replay != nil {
		if spec.Replay.SpecSHA256 != specSHA256 {
			return nil, errors.New("simulation replay plan does not match the specification")
		}
		replay := cloneReplayPlan(*spec.Replay)
		cluster.replay = &replay
	}
	for _, nodeSpec := range spec.Nodes {
		boot, ok := RegisteredBoot(nodeSpec.Boot)
		if !ok {
			return nil, fmt.Errorf("node %q uses unregistered boot %q", nodeSpec.ID, nodeSpec.Boot)
		}
		config := append([]byte(nil), nodeSpec.Config...)
		volumes := append([]VolumeMount(nil), nodeSpec.Volumes...)
		nodeSpec.Config = config
		nodeSpec.Volumes = volumes
		cluster.nodes[nodeSpec.ID] = &clusterNode{spec: nodeSpec, state: NodeStateDefined, handle: NodeHandle{Node: nodeSpec.ID}, boot: boot}
		cluster.ordered = append(cluster.ordered, nodeSpec.ID)
	}
	return cluster, nil
}

func (cluster *inProcessCluster) Start(ctx context.Context, id NodeID) (NodeHandle, error) {
	if err := cluster.beginCall(ctx); err != nil {
		return NodeHandle{}, err
	}
	defer cluster.endCall()
	cluster.mu.Lock()
	defer cluster.mu.Unlock()
	return cluster.startLocked(ctx, id, LifecycleStart)
}

func (cluster *inProcessCluster) Wait(ctx context.Context, handle NodeHandle) (NodeResult, error) {
	if err := cluster.beginCall(ctx); err != nil {
		return NodeResult{}, err
	}
	defer cluster.endCall()
	if cluster.backend == BackendProcess {
		return cluster.waitProcess(ctx, handle)
	}
	cluster.mu.Lock()
	node, err := cluster.currentNode(handle, NodeStateRunning)
	if err != nil {
		cluster.mu.Unlock()
		return NodeResult{}, err
	}
	if node.operation != nil {
		cluster.mu.Unlock()
		return NodeResult{}, ErrInvalidTransition
	}
	operation := &nodeOperation{action: LifecycleWait, done: make(chan struct{})}
	node.operation = operation
	completion := node.completion
	cluster.mu.Unlock()

	select {
	case <-completion.done:
	case <-ctx.Done():
		cluster.mu.Lock()
		if node.operation == operation {
			node.operation = nil
		}
		cluster.mu.Unlock()
		return NodeResult{}, ctx.Err()
	}

	cluster.mu.Lock()
	defer cluster.mu.Unlock()
	node, err = cluster.currentNode(handle, NodeStateRunning)
	if err != nil || node.operation != operation {
		if err != nil {
			return NodeResult{}, err
		}
		return NodeResult{}, ErrInvalidTransition
	}
	if divergenceErr := cluster.retainReplayFailureLocked(completion.err); divergenceErr != nil {
		node.operation = nil
		return NodeResult{}, divergenceErr
	}
	to := NodeStateExited
	reason := ""
	if completion.err != nil {
		to = NodeStateFailed
		reason = boundedTerminalText(completion.err.Error())
	}
	transition, err := cluster.prepareTransition(LifecycleWait, handle, NodeStateRunning, to)
	if err != nil {
		node.operation = nil
		return NodeResult{}, err
	}
	if err := runtimeVolumeRevoke(node.domain, true, false); err != nil {
		node.operation = nil
		return NodeResult{}, err
	}
	if err := runtimeNetworkRevoke(node.domain, true); err != nil {
		node.operation = nil
		return NodeResult{}, err
	}
	if err := runtimeDomainRevoke(node.domain); err != nil {
		node.operation = nil
		return NodeResult{}, err
	}
	cluster.commitTransition(transition)
	result := cluster.commitTerminal(node, to, reason)
	node.operation = nil
	close(operation.done)
	return result, nil
}

func (cluster *inProcessCluster) waitProcess(ctx context.Context, handle NodeHandle) (NodeResult, error) {
	cluster.mu.Lock()
	node, err := cluster.currentNode(handle, NodeStateRunning)
	if err != nil {
		cluster.mu.Unlock()
		return NodeResult{}, err
	}
	if node.operation != nil {
		cluster.mu.Unlock()
		return NodeResult{}, ErrInvalidTransition
	}
	operation := &nodeOperation{action: LifecycleWait, done: make(chan struct{})}
	node.operation = operation
	cluster.pendingOps++
	cluster.mu.Unlock()

	terminal, terminalErr := waitProcessNode(handle)
	cluster.mu.Lock()
	if terminalErr != nil {
		operation.err = terminalErr
		cluster.finishOperationLocked(operation)
		cluster.mu.Unlock()
		return NodeResult{}, terminalErr
	}
	to := NodeStateExited
	reason := ""
	if terminal.Error != "" {
		to = NodeStateFailed
		reason = boundedTerminalText(terminal.Error)
	}
	transition, err := cluster.prepareTransition(LifecycleWait, handle, NodeStateRunning, to)
	if err != nil {
		operation.err = err
		cluster.finishOperationLocked(operation)
		cluster.mu.Unlock()
		return NodeResult{}, err
	}
	node.modelClosing = true
	revokeErr := errors.Join(runtimeVolumeRevoke(node.domain, true, false), runtimeNetworkRevoke(node.domain, true), runtimeDomainRevoke(node.domain))
	cluster.mu.Unlock()
	cluster.waitForProcessModelOperations(node)
	cluster.mu.Lock()
	if revokeErr != nil {
		operation.err = revokeErr
		cluster.finishOperationLocked(operation)
		cluster.mu.Unlock()
		return NodeResult{}, revokeErr
	}
	cluster.processOutputs = append(cluster.processOutputs, terminal.Outputs...)
	cluster.commitTransition(transition)
	result := cluster.commitTerminal(node, to, reason)
	cluster.finishOperationLocked(operation)
	cluster.mu.Unlock()
	return result, nil
}

func (cluster *inProcessCluster) Stop(ctx context.Context, handle NodeHandle) error {
	if err := cluster.beginCall(ctx); err != nil {
		return err
	}
	defer cluster.endCall()
	if cluster.backend == BackendProcess {
		return cluster.stopProcess(ctx, handle)
	}
	cluster.mu.Lock()
	node, err := cluster.currentNode(handle, NodeStateRunning)
	if err != nil {
		cluster.mu.Unlock()
		return err
	}
	operation := node.operation
	if operation != nil {
		cluster.mu.Unlock()
		return ErrInvalidTransition
	}
	entry, err := cluster.reserveStopTransition(handle)
	if err != nil {
		cluster.mu.Unlock()
		return err
	}
	operation = &nodeOperation{action: LifecycleStop, done: make(chan struct{})}
	node.operation = operation
	cluster.pendingOps++
	cancel := node.cancel
	cancel()
	cluster.mu.Unlock()
	cluster.finishStop(node, handle, operation, entry)
	return operation.err
}

func (cluster *inProcessCluster) stopProcess(ctx context.Context, handle NodeHandle) error {
	cluster.mu.Lock()
	node, err := cluster.currentNode(handle, NodeStateRunning)
	if err != nil {
		cluster.mu.Unlock()
		return err
	}
	if node.operation != nil {
		cluster.mu.Unlock()
		return ErrInvalidTransition
	}
	entry, err := cluster.reserveStopTransition(handle)
	if err != nil {
		cluster.mu.Unlock()
		return err
	}
	operation := &nodeOperation{action: LifecycleStop, done: make(chan struct{})}
	node.operation = operation
	node.modelClosing = true
	cluster.pendingOps++
	revokeErr := errors.Join(runtimeVolumeRevoke(node.domain, true, false), runtimeNetworkRevoke(node.domain, true), runtimeDomainRevoke(node.domain))
	cluster.mu.Unlock()
	cluster.waitForProcessModelOperations(node)

	terminal, terminalErr := stopProcessNode(handle)
	cluster.mu.Lock()
	defer cluster.mu.Unlock()
	if terminalErr != nil {
		cluster.transitions = cluster.transitions[:len(cluster.transitions)-1]
		operation.err = errors.Join(revokeErr, terminalErr)
		cluster.finishOperationLocked(operation)
		return operation.err
	}
	to := NodeStateStopped
	reason := ""
	if terminal.Error != "" {
		to = NodeStateFailed
		reason = boundedTerminalText(terminal.Error)
	}
	actual := entry.transition
	actual.To = to
	if cluster.replay != nil && entry.transition != actual {
		err := &ReplayDivergenceError{Divergence: ReplayDivergence{Dimension: ReplayDimensionTransition, Ordinal: actual.Ordinal, Expected: entry.transition, Actual: actual}}
		cluster.transitions = cluster.transitions[:len(cluster.transitions)-1]
		cluster.replayFailure = err
		operation.err = err
		cluster.finishOperationLocked(operation)
		return err
	}
	entry.transition = actual
	entry.resolved = true
	if revokeErr != nil {
		operation.err = revokeErr
		cluster.finishOperationLocked(operation)
		return revokeErr
	}
	cluster.processOutputs = append(cluster.processOutputs, terminal.Outputs...)
	cluster.commitTerminal(node, to, reason)
	if to == NodeStateFailed {
		operation.err = fmt.Errorf("stop node %q: %s", handle.Node, terminal.Error)
	}
	result := operation.err
	cluster.finishOperationLocked(operation)
	return result
}

func (cluster *inProcessCluster) finishStop(node *clusterNode, handle NodeHandle, operation *nodeOperation, entry *transitionEntry) {
	<-node.completion.done
	bootErr := node.completion.err
	to := NodeStateStopped
	reason := ""
	if bootErr != nil && !errors.Is(bootErr, context.Canceled) {
		to = NodeStateFailed
		reason = boundedTerminalText(bootErr.Error())
	}
	actual := entry.transition
	actual.To = to

	cluster.mu.Lock()
	defer cluster.mu.Unlock()
	if node.operation != operation || node.handle != handle || node.state != NodeStateRunning {
		operation.err = ErrInvalidTransition
		cluster.finishOperationLocked(operation)
		return
	}
	if divergenceErr := cluster.retainReplayFailureLocked(bootErr); divergenceErr != nil {
		operation.err = divergenceErr
		cluster.transitions = cluster.transitions[:len(cluster.transitions)-1]
		cluster.finishOperationLocked(operation)
		return
	}
	if cluster.replay != nil && entry.transition != actual {
		operation.err = &ReplayDivergenceError{Divergence: ReplayDivergence{
			Dimension: ReplayDimensionTransition,
			Ordinal:   actual.Ordinal,
			Expected:  entry.transition,
			Actual:    actual,
		}}
		cluster.transitions = cluster.transitions[:len(cluster.transitions)-1]
		cluster.replayFailure = operation.err
		cluster.finishOperationLocked(operation)
		return
	}
	entry.transition = actual
	entry.resolved = true
	if err := runtimeVolumeRevoke(node.domain, true, false); err != nil {
		operation.err = errors.Join(operation.err, err)
		cluster.finishOperationLocked(operation)
		return
	}
	if err := runtimeNetworkRevoke(node.domain, true); err != nil {
		operation.err = errors.Join(operation.err, err)
		cluster.finishOperationLocked(operation)
		return
	}
	if err := runtimeDomainRevoke(node.domain); err != nil {
		operation.err = errors.Join(operation.err, err)
		cluster.finishOperationLocked(operation)
		return
	}
	cluster.commitTerminal(node, to, reason)
	if to == NodeStateFailed && operation.err == nil {
		operation.err = fmt.Errorf("stop node %q: %w", handle.Node, bootErr)
	}
	cluster.finishOperationLocked(operation)
}

func (cluster *inProcessCluster) finishOperationLocked(operation *nodeOperation) {
	for _, node := range cluster.nodes {
		if node.operation == operation {
			node.operation = nil
			break
		}
	}
	cluster.pendingOps--
	close(operation.done)
	cluster.notifyActivity()
}

func (cluster *inProcessCluster) Crash(ctx context.Context, handle NodeHandle) error {
	return cluster.crash(ctx, handle, false)
}

func (cluster *inProcessCluster) crash(ctx context.Context, handle NodeHandle, persistedOnly bool) error {
	if err := cluster.beginCall(ctx); err != nil {
		return err
	}
	defer cluster.endCall()
	cluster.mu.Lock()
	node, err := cluster.currentNode(handle, NodeStateRunning)
	if err != nil {
		cluster.mu.Unlock()
		return err
	}
	if node.operation != nil {
		cluster.mu.Unlock()
		return ErrInvalidTransition
	}
	if cluster.backend == BackendProcess {
		err := cluster.crashProcessLocked(node, handle, persistedOnly)
		cluster.mu.Unlock()
		return err
	}
	defer cluster.mu.Unlock()
	transition, err := cluster.prepareTransition(LifecycleCrash, handle, NodeStateRunning, NodeStateCrashed)
	if err != nil {
		return err
	}
	if err := runtimeVolumeRevoke(node.domain, false, persistedOnly); err != nil {
		return err
	}
	if err := runtimeNetworkRevoke(node.domain, false); err != nil {
		return err
	}
	if err := runtimeDomainRevoke(node.domain); err != nil {
		return err
	}
	cluster.commitTransition(transition)
	if cluster.backend == BackendInProcess {
		cluster.leaks = append(cluster.leaks, LeakDiagnostic{Handle: handle, Kind: LeakRevokedGoroutineMayRemain})
	}
	cluster.commitTerminal(node, NodeStateCrashed, "")
	return nil
}

func (cluster *inProcessCluster) crashProcessLocked(node *clusterNode, handle NodeHandle, persistedOnly bool) error {
	transition, err := cluster.prepareTransition(LifecycleCrash, handle, NodeStateRunning, NodeStateCrashed)
	if err != nil {
		return err
	}
	entry := &transitionEntry{transition: transition}
	cluster.transitions = append(cluster.transitions, entry)
	operation := &nodeOperation{action: LifecycleCrash, done: make(chan struct{})}
	node.operation = operation
	node.modelClosing = true
	cluster.pendingOps++
	if err := crashProcessNode(handle); err != nil {
		cluster.transitions = cluster.transitions[:len(cluster.transitions)-1]
		node.modelClosing = false
		operation.err = err
		cluster.finishOperationLocked(operation)
		return err
	}
	revokeErr := errors.Join(runtimeVolumeRevoke(node.domain, false, persistedOnly), runtimeNetworkRevoke(node.domain, false), runtimeDomainRevoke(node.domain))
	cluster.mu.Unlock()
	cluster.waitForProcessModelOperations(node)
	reapErr := waitCrashedProcessNode(handle)
	cluster.mu.Lock()
	if reapErr != nil {
		operation.err = errors.Join(revokeErr, reapErr)
		cluster.finishOperationLocked(operation)
		return operation.err
	}
	entry.resolved = true
	cluster.commitTerminal(node, NodeStateCrashed, "")
	operation.err = revokeErr
	cluster.finishOperationLocked(operation)
	return revokeErr
}

func (cluster *inProcessCluster) Restart(ctx context.Context, id NodeID) (NodeHandle, error) {
	if err := cluster.beginCall(ctx); err != nil {
		return NodeHandle{}, err
	}
	defer cluster.endCall()
	cluster.mu.Lock()
	defer cluster.mu.Unlock()
	node, ok := cluster.nodes[id]
	if !ok || node.operation != nil || node.state != NodeStateStopped && node.state != NodeStateCrashed && node.state != NodeStateExited && node.state != NodeStateFailed {
		return NodeHandle{}, ErrInvalidTransition
	}
	from := node.state
	handle := NodeHandle{Node: id, Incarnation: node.handle.Incarnation + 1}
	transition, err := cluster.prepareTransition(LifecycleRestart, handle, from, NodeStateRunning)
	if err != nil {
		return NodeHandle{}, err
	}
	domain, err := cluster.prepareIncarnation(node, handle)
	if err != nil {
		return NodeHandle{}, err
	}
	cluster.commitTransition(transition)
	if cluster.backend == BackendProcess {
		cluster.launchProcess(node, handle, domain)
	} else {
		cluster.launch(ctx, node, handle, domain)
	}
	return handle, nil
}

func (cluster *inProcessCluster) Partition(ctx context.Context, left, right NodeID) error {
	if err := cluster.beginCall(ctx); err != nil {
		return err
	}
	defer cluster.endCall()
	return runtimeNetworkPartition(cluster.runtimeRun, left, right, true)
}

func (cluster *inProcessCluster) Heal(ctx context.Context, left, right NodeID) error {
	if err := cluster.beginCall(ctx); err != nil {
		return err
	}
	defer cluster.endCall()
	return runtimeNetworkHeal(cluster.runtimeRun, left, right, true)
}

func (cluster *inProcessCluster) SetDelay(ctx context.Context, left, right NodeID, delayNanos uint64) error {
	if err := cluster.beginCall(ctx); err != nil {
		return err
	}
	defer cluster.endCall()
	return runtimeNetworkDelay(cluster.runtimeRun, left, right, delayNanos, true)
}

func (cluster *inProcessCluster) EnumerateCrashStates(ctx context.Context, handle NodeHandle, volume VolumeID, limits VolumeCrashEnumerationLimits, frontier *VolumeCrashFrontier) (VolumeCrashEnumeration, error) {
	if err := cluster.beginCall(ctx); err != nil {
		return VolumeCrashEnumeration{}, err
	}
	defer cluster.endCall()
	cluster.mu.Lock()
	node, err := cluster.currentNode(handle, NodeStateRunning)
	if err != nil {
		cluster.mu.Unlock()
		return VolumeCrashEnumeration{}, err
	}
	domain := node.domain
	cluster.mu.Unlock()
	return runtimeVolumeEnumerate(domain, volume, limits, frontier)
}

func (cluster *inProcessCluster) startLocked(ctx context.Context, id NodeID, action LifecycleAction) (NodeHandle, error) {
	node, ok := cluster.nodes[id]
	if !ok || node.state != NodeStateDefined || node.operation != nil {
		return NodeHandle{}, ErrInvalidTransition
	}
	handle := NodeHandle{Node: id, Incarnation: node.handle.Incarnation + 1}
	transition, err := cluster.prepareTransition(action, handle, NodeStateDefined, NodeStateRunning)
	if err != nil {
		return NodeHandle{}, err
	}
	domain, err := cluster.prepareIncarnation(node, handle)
	if err != nil {
		return NodeHandle{}, err
	}
	cluster.commitTransition(transition)
	if cluster.backend == BackendProcess {
		cluster.launchProcess(node, handle, domain)
	} else {
		cluster.launch(ctx, node, handle, domain)
	}
	return handle, nil
}

func (cluster *inProcessCluster) prepareIncarnation(node *clusterNode, handle NodeHandle) (uint64, error) {
	if cluster.backend == BackendProcess {
		if err := cluster.startProcessNode(node, handle); err != nil {
			return 0, err
		}
	}
	domain, err := runtimeDomainRegister(cluster.runtimeRun, handle.Node, node.spec.Address, handle.Incarnation)
	if err != nil {
		if cluster.backend == BackendProcess {
			return 0, errors.Join(err, crashProcessNode(handle))
		}
		return 0, err
	}
	if err := runtimeVolumeRegister(domain); err != nil {
		if cluster.backend == BackendProcess {
			return 0, errors.Join(err, crashProcessNode(handle), revokeRegisteredDomain(domain))
		}
		return 0, errors.Join(err, revokeRegisteredDomain(domain))
	}
	if cluster.backend == BackendProcess {
		if err := activateProcessNode(handle); err != nil {
			return 0, errors.Join(err, crashProcessNode(handle), runtimeVolumeRevoke(domain, false, true), runtimeNetworkRevoke(domain, false), revokeRegisteredDomain(domain))
		}
	}
	return domain, nil
}

func revokeRegisteredDomain(domain uint64) error {
	if err := runtimeDomainRevoke(domain); err != nil {
		return fmt.Errorf("revoke simulation domain after volume registration failure: %w", err)
	}
	return nil
}

func (cluster *inProcessCluster) beginCall(ctx context.Context) error {
	if err := actionContextError(ctx); err != nil {
		return err
	}
	cluster.mu.Lock()
	defer cluster.mu.Unlock()
	if cluster.closing {
		return ErrInvalidTransition
	}
	if cluster.replayFailure != nil {
		return cluster.replayFailure
	}
	cluster.activeCalls++
	return nil
}

func (cluster *inProcessCluster) endCall() {
	cluster.mu.Lock()
	cluster.activeCalls--
	cluster.mu.Unlock()
	cluster.notifyActivity()
}

func actionContextError(ctx context.Context) error {
	if ctx == nil {
		return errors.New("simulation lifecycle context is nil")
	}
	return ctx.Err()
}

func (cluster *inProcessCluster) launch(ctx context.Context, node *clusterNode, handle NodeHandle, domain uint64) {
	bootCtx, cancel := context.WithCancel(ctx)
	completion := &bootCompletion{done: make(chan struct{})}
	boot := node.boot
	nodeContext := NodeContext{NodeHandle: handle, Address: node.spec.Address, Config: append([]byte(nil), node.spec.Config...)}
	node.state = NodeStateRunning
	node.handle = handle
	node.cancel = cancel
	node.completion = completion
	node.domain = domain
	node.reason = ""
	go func() {
		previous, err := runtimeDomainEnter(domain)
		if err == nil {
			defer runtimeDomainLeave(previous)
			err = boot(bootCtx, nodeContext)
		}
		completion.err = err
		close(completion.done)
		cluster.notifyActivity()
	}()
}

func (cluster *inProcessCluster) launchProcess(node *clusterNode, handle NodeHandle, domain uint64) {
	node.state = NodeStateRunning
	node.handle = handle
	node.cancel = nil
	node.completion = nil
	node.domain = domain
	node.reason = ""
	node.modelActive = 0
	node.modelStarted = 0
	node.modelClosing = false
}

func (cluster *inProcessCluster) currentNode(handle NodeHandle, state NodeState) (*clusterNode, error) {
	node, ok := cluster.nodes[handle.Node]
	if !ok {
		return nil, ErrInvalidTransition
	}
	if handle.Incarnation != node.handle.Incarnation {
		return nil, ErrStaleIncarnation
	}
	if node.handle != handle || node.state != state {
		return nil, ErrInvalidTransition
	}
	return node, nil
}

func (cluster *inProcessCluster) prepareTransition(action LifecycleAction, handle NodeHandle, from, to NodeState) (LifecycleTransition, error) {
	if len(cluster.transitions) != 0 && !cluster.transitions[len(cluster.transitions)-1].resolved {
		return LifecycleTransition{}, ErrInvalidTransition
	}
	if err := cluster.checkCommitCapacity(); err != nil {
		return LifecycleTransition{}, err
	}
	transition := LifecycleTransition{
		Ordinal: uint64(len(cluster.transitions)),
		Action:  action,
		Handle:  handle,
		From:    from,
		To:      to,
	}
	if cluster.replay != nil {
		expected := cluster.expectedTransition(transition.Ordinal)
		if expected != transition {
			return LifecycleTransition{}, transitionDivergence(transition.Ordinal, expected, transition)
		}
	}
	return transition, nil
}

func (cluster *inProcessCluster) reserveStopTransition(handle NodeHandle) (*transitionEntry, error) {
	if len(cluster.transitions) != 0 && !cluster.transitions[len(cluster.transitions)-1].resolved {
		return nil, ErrInvalidTransition
	}
	if err := cluster.checkCommitCapacity(); err != nil {
		return nil, err
	}
	transition := LifecycleTransition{
		Ordinal: uint64(len(cluster.transitions)),
		Action:  LifecycleStop,
		Handle:  handle,
		From:    NodeStateRunning,
		To:      NodeStateStopped,
	}
	if cluster.replay != nil {
		expected := cluster.expectedTransition(transition.Ordinal)
		if expected.Ordinal != transition.Ordinal || expected.Action != transition.Action || expected.Handle != transition.Handle || expected.From != transition.From || expected.To != NodeStateStopped && expected.To != NodeStateFailed {
			return nil, transitionDivergence(transition.Ordinal, expected, transition)
		}
		transition = expected
	}
	entry := &transitionEntry{transition: transition}
	cluster.transitions = append(cluster.transitions, entry)
	return entry, nil
}

func transitionDivergence(ordinal uint64, expected, actual LifecycleTransition) error {
	return &ReplayDivergenceError{Divergence: ReplayDivergence{
		Dimension: ReplayDimensionTransition,
		Ordinal:   ordinal,
		Expected:  expected,
		Actual:    actual,
	}}
}

func (cluster *inProcessCluster) expectedTransition(ordinal uint64) LifecycleTransition {
	if ordinal < uint64(len(cluster.replay.Transitions)) {
		return cluster.replay.Transitions[ordinal]
	}
	return LifecycleTransition{}
}

func (cluster *inProcessCluster) commitTransition(transition LifecycleTransition) {
	cluster.transitions = append(cluster.transitions, &transitionEntry{transition: transition, resolved: true})
}

func (cluster *inProcessCluster) checkCommitCapacity() error {
	return checkCapacity("scenario_actions", uint64(len(cluster.transitions))+1, cluster.limits.ScenarioActions)
}

func (cluster *inProcessCluster) commitTerminal(node *clusterNode, state NodeState, reason string) NodeResult {
	result := NodeResult{Handle: node.handle, State: state, Reason: reason}
	cluster.incarnations = append(cluster.incarnations, result)
	node.state = state
	node.cancel = nil
	node.completion = nil
	node.domain = 0
	node.reason = reason
	return result
}

func (cluster *inProcessCluster) shutdown(ctx context.Context) error {
	if cluster.backend == BackendProcess {
		return cluster.shutdownProcess(ctx)
	}
	cluster.mu.Lock()
	cluster.closing = true
	for _, id := range cluster.ordered {
		node := cluster.nodes[id]
		if node.state == NodeStateRunning && node.cancel != nil {
			node.cancel()
		}
	}
	cluster.mu.Unlock()
	if err := cluster.waitForQuiescence(ctx); err != nil {
		return err
	}

	cluster.mu.Lock()
	defer cluster.mu.Unlock()
	var cleanupErr error
	for _, id := range cluster.ordered {
		node := cluster.nodes[id]
		if node.state != NodeStateRunning {
			continue
		}
		if err := runtimeVolumeRevoke(node.domain, true, false); err != nil {
			cleanupErr = errors.Join(cleanupErr, err)
			continue
		}
		if err := runtimeNetworkRevoke(node.domain, true); err != nil {
			cleanupErr = errors.Join(cleanupErr, err)
			continue
		}
		if err := runtimeDomainRevoke(node.domain); err != nil {
			cleanupErr = errors.Join(cleanupErr, err)
			continue
		}
		to := NodeStateStopped
		reason := ""
		if node.completion.err != nil && !errors.Is(node.completion.err, context.Canceled) {
			to = NodeStateFailed
			reason = boundedTerminalText(node.completion.err.Error())
		}
		cluster.commitTerminal(node, to, reason)
	}
	return cleanupErr
}

func (cluster *inProcessCluster) shutdownProcess(ctx context.Context) error {
	cluster.mu.Lock()
	cluster.closing = true
	for _, id := range cluster.ordered {
		if node := cluster.nodes[id]; node.state == NodeStateRunning {
			node.modelClosing = true
		}
	}
	cluster.mu.Unlock()
	if err := cluster.waitForQuiescence(ctx); err != nil {
		return err
	}
	cluster.mu.Lock()
	handles := make([]NodeHandle, 0, len(cluster.ordered))
	for _, id := range cluster.ordered {
		if node := cluster.nodes[id]; node.state == NodeStateRunning {
			handles = append(handles, node.handle)
		}
	}
	cluster.mu.Unlock()
	var cleanupErr error
	for _, handle := range handles {
		cluster.mu.Lock()
		node, currentErr := cluster.currentNode(handle, NodeStateRunning)
		if currentErr == nil {
			currentErr = errors.Join(runtimeVolumeRevoke(node.domain, true, false), runtimeNetworkRevoke(node.domain, true), runtimeDomainRevoke(node.domain))
		}
		cluster.mu.Unlock()
		if node != nil {
			cluster.waitForProcessModelOperations(node)
		}
		terminal, err := stopProcessNode(handle)
		if err != nil {
			crashErr := crashProcessNode(handle)
			cleanupErr = errors.Join(cleanupErr, currentErr, err, crashErr)
			continue
		}
		cluster.mu.Lock()
		if currentErr == nil {
			cluster.processOutputs = append(cluster.processOutputs, terminal.Outputs...)
			to := NodeStateStopped
			reason := ""
			if terminal.Error != "" {
				to = NodeStateFailed
				reason = boundedTerminalText(terminal.Error)
			}
			cluster.commitTerminal(node, to, reason)
		}
		cluster.mu.Unlock()
		cleanupErr = errors.Join(cleanupErr, currentErr)
	}
	return cleanupErr
}

func (cluster *inProcessCluster) waitForQuiescence(ctx context.Context) error {
	for {
		cluster.mu.Lock()
		active := cluster.activeCalls != 0 || cluster.pendingOps != 0
		var completion <-chan struct{}
		if !active {
			for _, id := range cluster.ordered {
				node := cluster.nodes[id]
				if node.state == NodeStateRunning && node.completion != nil {
					select {
					case <-node.completion.done:
					default:
						completion = node.completion.done
					}
					if completion != nil {
						break
					}
				}
			}
		}
		cluster.mu.Unlock()
		if !active && completion == nil {
			return nil
		}
		select {
		case <-cluster.activity:
		case <-completion:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (cluster *inProcessCluster) notifyActivity() {
	select {
	case cluster.activity <- struct{}{}:
	default:
	}
}

func (cluster *inProcessCluster) result() (Result, error) {
	cluster.mu.Lock()
	defer cluster.mu.Unlock()
	transitions := make([]LifecycleTransition, len(cluster.transitions))
	for index, entry := range cluster.transitions {
		if !entry.resolved {
			return Result{}, errors.New("simulation lifecycle transition remains unresolved")
		}
		transitions[index] = entry.transition
	}
	incarnations := append([]NodeResult(nil), cluster.incarnations...)
	sort.Slice(incarnations, func(left, right int) bool {
		if incarnations[left].Handle.Node != incarnations[right].Handle.Node {
			return incarnations[left].Handle.Node < incarnations[right].Handle.Node
		}
		return incarnations[left].Handle.Incarnation < incarnations[right].Handle.Incarnation
	})
	result := Result{
		Nodes:        make([]NodeResult, 0, len(cluster.ordered)),
		Leaks:        append([]LeakDiagnostic(nil), cluster.leaks...),
		Faults:       cloneFaultRealizations(cluster.faults),
		Scenarios:    cloneScenarioDecisions(cluster.scenarios),
		History:      cloneHistoryOperations(cluster.history),
		Observations: cloneObservations(cluster.observations),
		Oracles:      cloneOracleResults(cluster.oracles),
		Record:       ClusterRecord{Nodes: incarnations, Transitions: transitions},
	}
	for _, id := range cluster.ordered {
		node := cluster.nodes[id]
		result.Nodes = append(result.Nodes, NodeResult{Handle: node.handle, State: node.state, Reason: node.reason})
	}
	return result, nil
}

func (cluster *inProcessCluster) finishReplay(result Result) error {
	if cluster.replay == nil {
		return nil
	}
	actualTransitions := result.Record.Transitions
	if len(actualTransitions) != len(cluster.replay.Transitions) {
		ordinal := uint64(len(actualTransitions))
		var expected LifecycleTransition
		if ordinal < uint64(len(cluster.replay.Transitions)) {
			expected = cluster.replay.Transitions[ordinal]
		}
		return transitionDivergence(ordinal, expected, LifecycleTransition{})
	}
	if cluster.replay.Outcome != result.Outcome || cluster.replay.Reason != result.Reason || cluster.replay.FailureIdentity != result.FailureIdentity || !equalNodeResults(cluster.replay.Nodes, result.Record.Nodes) || !equalOutputs(cluster.replay.Outputs, result.Outputs) || !equalLeaks(cluster.replay.Leaks, result.Leaks) {
		expected, err := hashCanonical("gomadv3-cluster-replay-terminal/v1", replayTerminal{
			Outcome:         cluster.replay.Outcome,
			Reason:          cluster.replay.Reason,
			FailureIdentity: cluster.replay.FailureIdentity,
			Nodes:           cluster.replay.Nodes,
			Outputs:         cluster.replay.Outputs,
			Leaks:           cluster.replay.Leaks,
		})
		if err != nil {
			return err
		}
		actual, err := hashCanonical("gomadv3-cluster-replay-terminal/v1", replayTerminal{
			Outcome:         result.Outcome,
			Reason:          result.Reason,
			FailureIdentity: result.FailureIdentity,
			Nodes:           result.Record.Nodes,
			Outputs:         result.Outputs,
			Leaks:           result.Leaks,
		})
		if err != nil {
			return err
		}
		return &ReplayDivergenceError{Divergence: ReplayDivergence{
			Dimension:      ReplayDimensionTerminal,
			Ordinal:        uint64(len(actualTransitions)),
			ExpectedSHA256: expected,
			ActualSHA256:   actual,
		}}
	}
	return nil
}

type replayTerminal struct {
	Outcome         Outcome             `json:"outcome"`
	Reason          string              `json:"reason,omitempty"`
	FailureIdentity string              `json:"failure_identity,omitempty"`
	Nodes           []NodeResult        `json:"nodes"`
	Outputs         []OutputObservation `json:"outputs"`
	Leaks           []LeakDiagnostic    `json:"leaks"`
}

func boundedTerminalText(value string) string {
	value = strings.ToValidUTF8(value, "�")
	if len(value) <= MaximumTerminalReasonBytes {
		return value
	}
	value = value[:MaximumTerminalReasonBytes]
	for !utf8.ValidString(value) {
		value = value[:len(value)-1]
	}
	return value
}

func equalLeaks(left, right []LeakDiagnostic) bool {
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

func equalOutputs(left, right []OutputObservation) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index].Handle != right[index].Handle || left[index].Stream != right[index].Stream || left[index].FullSHA256 != right[index].FullSHA256 || left[index].TotalBytes != right[index].TotalBytes || left[index].RetainedBytes != right[index].RetainedBytes || left[index].DiscardedBytes != right[index].DiscardedBytes || left[index].Truncated != right[index].Truncated || string(left[index].Bytes) != string(right[index].Bytes) {
			return false
		}
	}
	return true
}

func equalNodeResults(left, right []NodeResult) bool {
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
