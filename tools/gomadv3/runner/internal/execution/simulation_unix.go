//go:build unix

package execution

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

type simulationNodeProcess struct {
	node                   string
	incarnation            uint64
	cancel                 context.CancelFunc
	hardCrash              chan struct{}
	ready                  chan struct{}
	activate               chan struct{}
	activated              chan struct{}
	done                   chan struct{}
	reaped                 chan struct{}
	result                 Result
	err                    error
	terminal               []byte
	readyOnce              sync.Once
	activationStarted      bool
	activationAcknowledged bool
	hardCrashStarted       bool
	completionRelease      chan bool
	completionPending      bool
	time                   *simulationTimeParticipant
}

type simulationCoordinator struct {
	mu          sync.Mutex
	request     Spec
	nodes       map[string]*simulationNodeProcess
	model       *simulationModelTransport
	time        *simulationTimeArbiter
	coordinator *simulationTimeParticipant
	responses   map[uint64]uint64
	closing     bool
}

func newSimulationCoordinator(request Spec) (*simulationCoordinator, error) {
	timeArbiter := newSimulationTimeArbiter()
	participant, err := timeArbiter.register("coordinator")
	if err != nil {
		return nil, err
	}
	timeArbiter.activate(participant)
	return &simulationCoordinator{
		request: request, nodes: make(map[string]*simulationNodeProcess), time: timeArbiter, coordinator: participant,
		responses: make(map[uint64]uint64),
	}, nil
}

func (coordinator *simulationCoordinator) handle(ctx context.Context, frame simulationFrame) (simulationFrame, error) {
	if frame.Kind == simulationFrameWait {
		if err := coordinator.time.acknowledgeExternal(coordinator.coordinator, frame.Arrivals); err != nil {
			return simulationFrame{}, err
		}
		coordinator.time.runnable(coordinator.coordinator)
	} else if frame.Kind == simulationFrameStop || frame.Kind == simulationFrameCrash {
		node, err := coordinator.node(frame)
		if err != nil {
			if barrierErr := coordinator.beginResponseBarrier(frame.Request, frame.Arrivals); barrierErr != nil {
				return simulationFrame{}, errors.Join(err, barrierErr)
			}
			return simulationFrame{}, err
		}
		if err := coordinator.beginForwardedResponseBarrier(frame.Request, frame.Arrivals, node); err != nil {
			return simulationFrame{}, err
		}
	} else {
		if err := coordinator.beginResponseBarrier(frame.Request, frame.Arrivals); err != nil {
			return simulationFrame{}, err
		}
	}
	switch frame.Kind {
	case simulationFrameStart:
		return coordinator.start(ctx, frame)
	case simulationFrameActivate:
		return coordinator.activate(ctx, frame)
	case simulationFrameStop:
		return coordinator.stop(ctx, frame, false)
	case simulationFrameCrash:
		return coordinator.stop(ctx, frame, true)
	case simulationFrameWait:
		return coordinator.wait(ctx, frame)
	default:
		return simulationFrame{}, fmt.Errorf("simulation coordinator cannot handle %q", frame.Kind)
	}
}

func (coordinator *simulationCoordinator) handleCoordinatorTime(ctx context.Context, request simulationTimeRequest) (simulationTimeResponse, error) {
	return coordinator.time.quiesce(ctx, coordinator.coordinator, request)
}

func (coordinator *simulationCoordinator) handleCoordinatorDelivery(frame simulationFrame) {
	coordinator.mu.Lock()
	_, pending := coordinator.responses[frame.Request]
	delete(coordinator.responses, frame.Request)
	coordinator.mu.Unlock()
	if pending {
		coordinator.time.deliverExternal(coordinator.coordinator)
	}
}

func (coordinator *simulationCoordinator) handleCoordinatorResponse(simulationFrame) {
	coordinator.time.runnable(coordinator.coordinator)
}

func (coordinator *simulationCoordinator) handleModelArrival(frame simulationFrame) error {
	node, err := coordinator.node(frame)
	if err != nil {
		return err
	}
	return coordinator.time.transferExternalArrival(coordinator.coordinator, frame.Arrivals, node.time)
}

func (coordinator *simulationCoordinator) start(ctx context.Context, frame simulationFrame) (simulationFrame, error) {
	if frame.Node == "" || frame.Incarnation == 0 || len(frame.Payload) == 0 {
		return simulationFrame{}, errors.New("simulation node start request is incomplete")
	}
	key := simulationNodeKey(frame.Node, frame.Incarnation)
	coordinator.mu.Lock()
	if coordinator.closing {
		coordinator.mu.Unlock()
		return simulationFrame{}, errors.New("simulation coordinator is closing")
	}
	if _, exists := coordinator.nodes[key]; exists {
		coordinator.mu.Unlock()
		return simulationFrame{}, errors.New("simulation node incarnation already exists")
	}
	timeParticipant, err := coordinator.time.register(key)
	if err != nil {
		coordinator.mu.Unlock()
		return simulationFrame{}, err
	}
	nodeCtx, cancel := context.WithCancel(ctx)
	node := &simulationNodeProcess{
		node: frame.Node, incarnation: frame.Incarnation, cancel: cancel, hardCrash: make(chan struct{}), ready: make(chan struct{}), activate: make(chan struct{}), activated: make(chan struct{}), done: make(chan struct{}), reaped: make(chan struct{}),
		time: timeParticipant,
	}
	coordinator.nodes[key] = node
	coordinator.mu.Unlock()

	request := coordinator.request
	request.Simulation = &SimulationCapability{
		Role: SimulationRoleNode, Bootstrap: append([]byte(nil), frame.Payload...), hardCrash: node.hardCrash,
		reaped: node.reaped,
		handler: func(childCtx context.Context, child simulationFrame) (simulationFrame, error) {
			return coordinator.handleNodeFrame(childCtx, node, child)
		},
		time: func(childCtx context.Context, child simulationTimeRequest) (simulationTimeResponse, error) {
			return coordinator.time.quiesce(childCtx, node.time, child)
		},
		delivering: func(simulationFrame) {
			coordinator.time.deliverExternal(node.time)
		},
		responded: func(simulationFrame) { coordinator.time.runnable(node.time) },
		arrived: func(arrivals uint32) error {
			return coordinator.time.acknowledgeExternal(node.time, arrivals)
		},
	}
	go func() {
		node.result, node.err = Run(nodeCtx, request)
		coordinator.mu.Lock()
		completionRelease := node.completionRelease
		coordinator.mu.Unlock()
		if completionRelease == nil {
			coordinator.retainNodeCompletion(node)
			close(node.done)
		} else {
			close(node.done)
			if !<-completionRelease {
				coordinator.retainNodeCompletion(node)
			}
		}
		coordinator.time.remove(node.time)
		node.readyOnce.Do(func() { close(node.ready) })
	}()

	select {
	case <-node.ready:
		select {
		case <-node.done:
			barrierErr := coordinator.retainCompletionUntilResponse(frame.Request, node)
			if node.err != nil {
				return simulationFrame{}, fmt.Errorf("start simulation node: %w", errors.Join(node.err, barrierErr))
			}
			return simulationFrame{}, errors.Join(errors.New("simulation node exited before reporting readiness"), barrierErr)
		default:
			return simulationFrame{Node: frame.Node, Incarnation: frame.Incarnation}, nil
		}
	case <-ctx.Done():
		cancel()
		return simulationFrame{}, ctx.Err()
	}
}

func (coordinator *simulationCoordinator) activate(ctx context.Context, frame simulationFrame) (simulationFrame, error) {
	node, err := coordinator.node(frame)
	if err != nil {
		return simulationFrame{}, err
	}
	select {
	case <-node.ready:
	case <-node.done:
		return simulationFrame{}, errors.Join(errors.New("simulation node exited before activation"), coordinator.retainCompletionUntilResponse(frame.Request, node))
	case <-ctx.Done():
		return simulationFrame{}, ctx.Err()
	}
	coordinator.mu.Lock()
	if node.activationStarted {
		coordinator.mu.Unlock()
		return simulationFrame{}, errors.New("simulation node activation is duplicated")
	}
	node.activationStarted = true
	close(node.activate)
	coordinator.mu.Unlock()
	select {
	case <-node.activated:
		return simulationFrame{Node: node.node, Incarnation: node.incarnation}, nil
	case <-node.done:
		return simulationFrame{}, errors.Join(errors.New("simulation node exited before acknowledging activation"), coordinator.retainCompletionUntilResponse(frame.Request, node))
	case <-ctx.Done():
		return simulationFrame{}, ctx.Err()
	}
}

func (coordinator *simulationCoordinator) handleNodeFrame(ctx context.Context, node *simulationNodeProcess, frame simulationFrame) (simulationFrame, error) {
	if frame.Node != node.node || frame.Incarnation != node.incarnation {
		return simulationFrame{}, errors.New("simulation node frame identity changed")
	}
	modelRequest := frame.Kind == simulationFrameModel
	var err error
	if modelRequest {
		err = coordinator.time.forwardExternalAfterArrivals(node.time, frame.Arrivals, coordinator.coordinator)
	} else {
		err = coordinator.time.beginExternalAfterArrivals(node.time, frame.Arrivals)
	}
	if err != nil {
		return simulationFrame{}, err
	}
	switch frame.Kind {
	case simulationFrameReady:
		node.readyOnce.Do(func() { close(node.ready) })
		select {
		case <-node.activate:
		case <-ctx.Done():
			return simulationFrame{}, ctx.Err()
		}
	case simulationFrameActivated:
		coordinator.mu.Lock()
		if !node.activationStarted || node.activationAcknowledged {
			coordinator.mu.Unlock()
			return simulationFrame{}, errors.New("simulation node activation acknowledgement is invalid")
		}
		node.activationAcknowledged = true
		current := coordinator.time.activate(node.time)
		close(node.activated)
		coordinator.mu.Unlock()
		return simulationFrame{Node: node.node, Incarnation: node.incarnation, Payload: encodeSimulationActivationTime(current)}, nil
	case simulationFrameModel:
		if coordinator.model == nil {
			coordinator.time.endExternal(coordinator.coordinator)
			return simulationFrame{}, errors.New("simulation model transport is unavailable")
		}
		response, err := coordinator.model.exchange(ctx, frame)
		if err != nil {
			return simulationFrame{}, err
		}
		if response.Error != "" {
			return simulationFrame{}, errors.New(response.Error)
		}
		return simulationFrame{Node: node.node, Incarnation: node.incarnation, Time: coordinator.time.currentTime(), Payload: append([]byte(nil), response.Payload...)}, nil
	case simulationFrameTerminal:
		coordinator.mu.Lock()
		node.terminal = append([]byte(nil), frame.Payload...)
		coordinator.mu.Unlock()
	default:
		return simulationFrame{}, fmt.Errorf("simulation node cannot send %q", frame.Kind)
	}
	return simulationFrame{Node: node.node, Incarnation: node.incarnation}, nil
}

func (coordinator *simulationCoordinator) stop(ctx context.Context, frame simulationFrame, hard bool) (simulationFrame, error) {
	node, err := coordinator.node(frame)
	if err != nil {
		return simulationFrame{}, err
	}
	if hard {
		coordinator.mu.Lock()
		if node.hardCrashStarted {
			coordinator.mu.Unlock()
			return simulationFrame{}, errors.New("simulation node hard crash is duplicated")
		}
		node.hardCrashStarted = true
		close(node.hardCrash)
		coordinator.mu.Unlock()
		select {
		case <-node.reaped:
			return simulationFrame{Node: node.node, Incarnation: node.incarnation}, nil
		case <-node.done:
			barrierErr := coordinator.retainCompletionUntilResponse(frame.Request, node)
			select {
			case <-node.reaped:
				return simulationFrame{Node: node.node, Incarnation: node.incarnation}, barrierErr
			default:
			}
			return simulationFrame{}, errors.Join(errors.New("simulation node ended before hard-crash containment was confirmed"), node.err, barrierErr)
		case <-ctx.Done():
			return simulationFrame{}, ctx.Err()
		}
	}
	node.cancel()
	response, waitErr := coordinator.waitNode(ctx, node, false)
	if nodeCompleted(node) {
		waitErr = errors.Join(waitErr, coordinator.retainCompletionUntilResponse(frame.Request, node))
	}
	if waitErr == nil {
		coordinator.removeNode(node)
	}
	return response, waitErr
}

func (coordinator *simulationCoordinator) wait(ctx context.Context, frame simulationFrame) (simulationFrame, error) {
	node, err := coordinator.node(frame)
	if err != nil {
		return simulationFrame{}, err
	}
	completionRelease, err := coordinator.registerCompletionWait(node)
	if err != nil {
		return simulationFrame{}, err
	}
	coordinator.mu.Lock()
	crashed := node.hardCrashStarted
	coordinator.mu.Unlock()
	response, waitErr := coordinator.waitNode(ctx, node, crashed)
	completed := nodeCompleted(node)
	if completed {
		waitErr = errors.Join(waitErr, coordinator.retainCompletionUntilResponse(frame.Request, node))
	}
	waitErr = errors.Join(waitErr, coordinator.releaseCompletionWait(node, completionRelease, completed))
	if waitErr == nil {
		coordinator.removeNode(node)
	}
	return response, waitErr
}

func (coordinator *simulationCoordinator) registerCompletionWait(node *simulationNodeProcess) (chan bool, error) {
	coordinator.mu.Lock()
	defer coordinator.mu.Unlock()
	if node.completionRelease != nil {
		return nil, errors.New("simulation node completion already has a waiter")
	}
	release := make(chan bool, 1)
	node.completionRelease = release
	return release, nil
}

func (coordinator *simulationCoordinator) releaseCompletionWait(node *simulationNodeProcess, release chan bool, retained bool) error {
	coordinator.mu.Lock()
	defer coordinator.mu.Unlock()
	if node.completionRelease != release {
		return errors.New("simulation node completion waiter changed")
	}
	node.completionRelease = nil
	release <- retained
	return nil
}

func (coordinator *simulationCoordinator) retainNodeCompletion(node *simulationNodeProcess) {
	coordinator.mu.Lock()
	node.completionPending = true
	coordinator.mu.Unlock()
}

func (coordinator *simulationCoordinator) retainCompletionUntilResponse(request uint64, node *simulationNodeProcess) error {
	coordinator.mu.Lock()
	responsePending := coordinator.responses[request] != 0
	if node.completionPending {
		node.completionPending = false
		if !responsePending {
			coordinator.responses[request] = 1
		}
		coordinator.mu.Unlock()
		if !responsePending {
			coordinator.time.beginExternal(coordinator.coordinator)
		}
		return nil
	}
	coordinator.mu.Unlock()
	if responsePending {
		return nil
	}
	return coordinator.beginResponseBarrier(request, 0)
}

func (coordinator *simulationCoordinator) beginResponseBarrier(request uint64, arrivals uint32) error {
	if err := coordinator.time.beginExternalAfterArrivals(coordinator.coordinator, arrivals); err != nil {
		return err
	}
	coordinator.mu.Lock()
	defer coordinator.mu.Unlock()
	if coordinator.responses[request] != 0 {
		coordinator.time.endExternal(coordinator.coordinator)
		return errors.New("simulation response barrier is duplicated")
	}
	coordinator.responses[request] = 1
	return nil
}

func (coordinator *simulationCoordinator) beginForwardedResponseBarrier(request uint64, arrivals uint32, node *simulationNodeProcess) error {
	if err := coordinator.time.forwardExternalAfterArrivals(coordinator.coordinator, arrivals, node.time); err != nil {
		return err
	}
	coordinator.mu.Lock()
	if coordinator.responses[request] != 0 {
		coordinator.mu.Unlock()
		coordinator.time.endExternal(coordinator.coordinator)
		coordinator.time.endExternal(node.time)
		return errors.New("simulation response barrier is duplicated")
	}
	coordinator.responses[request] = 1
	coordinator.mu.Unlock()
	return nil
}

func nodeCompleted(node *simulationNodeProcess) bool {
	select {
	case <-node.done:
		return true
	default:
		return false
	}
}

func (coordinator *simulationCoordinator) removeNode(node *simulationNodeProcess) {
	coordinator.mu.Lock()
	delete(coordinator.nodes, simulationNodeKey(node.node, node.incarnation))
	coordinator.mu.Unlock()
	coordinator.time.remove(node.time)
}

func (coordinator *simulationCoordinator) waitNode(ctx context.Context, node *simulationNodeProcess, crashed bool) (simulationFrame, error) {
	select {
	case <-node.done:
	case <-ctx.Done():
		return simulationFrame{}, ctx.Err()
	}
	if !node.result.GroupGone {
		return simulationFrame{}, errors.Join(errors.New("simulation node process group remains after completion"), node.err)
	}
	if crashed {
		if node.result.Termination != TerminationSignal {
			return simulationFrame{}, errors.Join(errors.New("simulation crash did not terminate the node with a signal"), node.err)
		}
		return simulationFrame{Node: node.node, Incarnation: node.incarnation}, nil
	}
	if node.err != nil {
		return simulationFrame{}, node.err
	}
	if node.result.ExitCode != 0 || node.result.Termination != TerminationExit {
		stderr := node.result.Stderr.Bytes
		if len(stderr) > 1024 {
			stderr = stderr[:1024]
		}
		return simulationFrame{}, fmt.Errorf("simulation node terminated as %s exit=%d signal=%s stderr=%q", node.result.Termination, node.result.ExitCode, node.result.Signal, stderr)
	}
	coordinator.mu.Lock()
	terminal := append([]byte(nil), node.terminal...)
	coordinator.mu.Unlock()
	if len(terminal) == 0 {
		return simulationFrame{}, errors.New("simulation node omitted its terminal frame")
	}
	return simulationFrame{Node: node.node, Incarnation: node.incarnation, Payload: terminal}, nil
}

func (coordinator *simulationCoordinator) node(frame simulationFrame) (*simulationNodeProcess, error) {
	if frame.Node == "" || frame.Incarnation == 0 {
		return nil, errors.New("simulation node request identity is incomplete")
	}
	coordinator.mu.Lock()
	defer coordinator.mu.Unlock()
	node := coordinator.nodes[simulationNodeKey(frame.Node, frame.Incarnation)]
	if node == nil {
		return nil, errors.New("simulation node incarnation is unknown")
	}
	return node, nil
}

func (coordinator *simulationCoordinator) close() error {
	coordinator.mu.Lock()
	coordinator.closing = true
	nodes := make([]*simulationNodeProcess, 0, len(coordinator.nodes))
	for _, node := range coordinator.nodes {
		nodes = append(nodes, node)
	}
	coordinator.mu.Unlock()
	for _, node := range nodes {
		select {
		case <-node.done:
		default:
			node.cancel()
			<-node.done
		}
	}
	var result error
	for _, node := range nodes {
		if !node.result.GroupGone {
			result = errors.Join(result, fmt.Errorf("simulation node %s/%d process group remains", node.node, node.incarnation))
		}
	}
	coordinator.time.remove(coordinator.coordinator)
	return result
}

func simulationNodeKey(node string, incarnation uint64) string {
	return fmt.Sprintf("%s/%d", node, incarnation)
}
