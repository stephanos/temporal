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
	result                 Result
	err                    error
	terminal               []byte
	readyOnce              sync.Once
	activationStarted      bool
	activationAcknowledged bool
}

type simulationCoordinator struct {
	mu      sync.Mutex
	request Spec
	nodes   map[string]*simulationNodeProcess
	model   *simulationModelTransport
	closing bool
}

func newSimulationCoordinator(request Spec) *simulationCoordinator {
	return &simulationCoordinator{request: request, nodes: make(map[string]*simulationNodeProcess)}
}

func (coordinator *simulationCoordinator) handle(ctx context.Context, frame simulationFrame) (simulationFrame, error) {
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
	nodeCtx, cancel := context.WithCancel(ctx)
	node := &simulationNodeProcess{
		node: frame.Node, incarnation: frame.Incarnation, cancel: cancel, hardCrash: make(chan struct{}), ready: make(chan struct{}), activate: make(chan struct{}), activated: make(chan struct{}), done: make(chan struct{}),
	}
	coordinator.nodes[key] = node
	coordinator.mu.Unlock()

	request := coordinator.request
	request.Simulation = &SimulationCapability{
		Role: SimulationRoleNode, Bootstrap: append([]byte(nil), frame.Payload...), hardCrash: node.hardCrash,
		handler: func(childCtx context.Context, child simulationFrame) (simulationFrame, error) {
			return coordinator.handleNodeFrame(childCtx, node, child)
		},
	}
	go func() {
		node.result, node.err = Run(nodeCtx, request)
		close(node.done)
		node.readyOnce.Do(func() { close(node.ready) })
	}()

	select {
	case <-node.ready:
		select {
		case <-node.done:
			if node.err != nil {
				return simulationFrame{}, fmt.Errorf("start simulation node: %w", node.err)
			}
			return simulationFrame{}, errors.New("simulation node exited before reporting readiness")
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
		return simulationFrame{}, errors.New("simulation node exited before activation")
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
		return simulationFrame{}, errors.New("simulation node exited before acknowledging activation")
	case <-ctx.Done():
		return simulationFrame{}, ctx.Err()
	}
}

func (coordinator *simulationCoordinator) handleNodeFrame(ctx context.Context, node *simulationNodeProcess, frame simulationFrame) (simulationFrame, error) {
	if frame.Node != node.node || frame.Incarnation != node.incarnation {
		return simulationFrame{}, errors.New("simulation node frame identity changed")
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
		close(node.activated)
		coordinator.mu.Unlock()
	case simulationFrameModel:
		if coordinator.model == nil {
			return simulationFrame{}, errors.New("simulation model transport is unavailable")
		}
		response, err := coordinator.model.exchange(ctx, frame)
		if err != nil {
			return simulationFrame{}, err
		}
		if response.Error != "" {
			return simulationFrame{}, errors.New(response.Error)
		}
		return simulationFrame{Node: node.node, Incarnation: node.incarnation, Payload: append([]byte(nil), response.Payload...)}, nil
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
		close(node.hardCrash)
	} else {
		node.cancel()
	}
	response, waitErr := coordinator.waitNode(ctx, node, hard)
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
	response, waitErr := coordinator.waitNode(ctx, node, false)
	if waitErr == nil {
		coordinator.removeNode(node)
	}
	return response, waitErr
}

func (coordinator *simulationCoordinator) removeNode(node *simulationNodeProcess) {
	coordinator.mu.Lock()
	delete(coordinator.nodes, simulationNodeKey(node.node, node.incarnation))
	coordinator.mu.Unlock()
}

func (coordinator *simulationCoordinator) waitNode(ctx context.Context, node *simulationNodeProcess, crashed bool) (simulationFrame, error) {
	select {
	case <-node.done:
	case <-ctx.Done():
		return simulationFrame{}, ctx.Err()
	}
	if node.err != nil {
		return simulationFrame{}, node.err
	}
	if !node.result.GroupGone {
		return simulationFrame{}, errors.New("simulation node process group remains after completion")
	}
	if crashed {
		if node.result.Termination != TerminationSignal {
			return simulationFrame{}, errors.New("simulation crash did not terminate the node with a signal")
		}
		return simulationFrame{Node: node.node, Incarnation: node.incarnation}, nil
	}
	if node.result.ExitCode != 0 || node.result.Termination != TerminationExit {
		return simulationFrame{}, fmt.Errorf("simulation node terminated as %s exit=%d signal=%s", node.result.Termination, node.result.ExitCode, node.result.Signal)
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
	return result
}

func simulationNodeKey(node string, incarnation uint64) string {
	return fmt.Sprintf("%s/%d", node, incarnation)
}
