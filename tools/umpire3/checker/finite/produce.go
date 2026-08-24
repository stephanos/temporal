package finite

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"

	protocolcatalog "go.temporal.io/server/tools/umpire3/protocol/catalog"
	protocolchecker "go.temporal.io/server/tools/umpire3/protocol/checker"
)

type Options struct {
	Workers    int
	Replicas   int
	Limits     SearchLimits
	Checkpoint func(Checkpoint) error
}

type ResourceError struct {
	Resource string
	Limit    int
}

func (e *ResourceError) Error() string {
	return fmt.Sprintf("native search exhausted %s limit %d", e.Resource, e.Limit)
}

type CounterexampleError struct {
	Replica int
	Actions []protocolcatalog.ActionKind
	State   protocolchecker.FirstOrderState
}

func (e *CounterexampleError) Error() string {
	return fmt.Sprintf("native search found a property violation in replica %d", e.Replica)
}

type proposal struct {
	replica int
	parent  int
	action  protocolcatalog.ActionKind
	state   protocolchecker.FirstOrderState
	key     string
}

type expansion struct {
	proposals []proposal
	err       error
}

func Produce(
	ctx context.Context,
	view protocolchecker.FirstOrderView,
	options Options,
	resume *Checkpoint,
) (Certificate, error) {
	if options.Workers <= 0 || options.Replicas <= 0 || options.Replicas > 10 ||
		options.Limits.MaxDepth <= 0 || options.Limits.MaxStates <= 0 ||
		options.Limits.MaxTransitions <= 0 || options.Limits.MaxStateBytes <= 0 {
		return Certificate{}, errors.New("positive native workers, replicas, and resource limits are required")
	}
	machine, err := NewMachine(view)
	if err != nil {
		return Certificate{}, err
	}
	viewDigest, err := firstOrderViewDigest(view)
	if err != nil {
		return Certificate{}, err
	}
	checkpoint, err := seedCheckpoint(machine, view, viewDigest, options)
	if err != nil {
		return Certificate{}, err
	}
	if resume != nil {
		checkpoint = *resume
		if err := validateCheckpoint(checkpoint, machine, view, viewDigest, options); err != nil {
			return Certificate{}, err
		}
	}

	seen, err := checkpointSeen(machine, checkpoint)
	if err != nil {
		return Certificate{}, err
	}
	for len(checkpoint.Frontier) != 0 {
		if err := ctx.Err(); err != nil {
			return Certificate{}, err
		}
		expansions, err := expandFrontier(ctx, machine, checkpoint.Nodes, checkpoint.Frontier, options.Workers)
		if err != nil {
			return Certificate{}, err
		}
		proposals := make([]proposal, 0)
		layerTransitions := 0
		for _, expanded := range expansions {
			if expanded.err != nil {
				return Certificate{}, expanded.err
			}
			layerTransitions += len(expanded.proposals)
			proposals = append(proposals, expanded.proposals...)
		}
		if checkpoint.Transitions+layerTransitions > options.Limits.MaxTransitions {
			return Certificate{}, publishResourceCheckpoint(checkpoint, options,
				&ResourceError{Resource: "transitions", Limit: options.Limits.MaxTransitions})
		}
		slices.SortFunc(proposals, compareProposal)
		newNodes := make([]ExpandedNode, 0)
		newKeys := make(map[string]struct{})
		for _, candidate := range proposals {
			compoundKey := expandedStateKey(candidate.replica, candidate.key)
			if _, duplicate := seen[compoundKey]; duplicate {
				continue
			}
			if _, duplicate := newKeys[compoundKey]; duplicate {
				continue
			}
			parent := checkpoint.Nodes[candidate.parent]
			if parent.Depth >= options.Limits.MaxDepth {
				return Certificate{}, publishResourceCheckpoint(checkpoint, options,
					&ResourceError{Resource: "depth", Limit: options.Limits.MaxDepth})
			}
			safe, invariantErr := machine.Invariant(candidate.state)
			if invariantErr != nil {
				return Certificate{}, invariantErr
			}
			if !safe {
				return Certificate{}, &CounterexampleError{
					Replica: candidate.replica,
					Actions: append(actionsFor(checkpoint.Nodes, candidate.parent), candidate.action),
					State:   candidate.state,
				}
			}
			newKeys[compoundKey] = struct{}{}
			newNodes = append(newNodes, ExpandedNode{
				Replica: candidate.replica, State: candidate.state, Parent: candidate.parent,
				Action: candidate.action, Depth: parent.Depth + 1,
			})
		}
		if len(checkpoint.Nodes)+len(newNodes) > options.Limits.MaxStates {
			return Certificate{}, publishResourceCheckpoint(checkpoint, options,
				&ResourceError{Resource: "states", Limit: options.Limits.MaxStates})
		}
		additionalBytes := 0
		for _, node := range newNodes {
			size, sizeErr := stateEncodedSize(node.State)
			if sizeErr != nil {
				return Certificate{}, sizeErr
			}
			additionalBytes += size
		}
		if checkpoint.StateBytes+additionalBytes > options.Limits.MaxStateBytes {
			return Certificate{}, publishResourceCheckpoint(checkpoint, options,
				&ResourceError{Resource: "state-bytes", Limit: options.Limits.MaxStateBytes})
		}
		frontier := make([]int, len(newNodes))
		for index, node := range newNodes {
			frontier[index] = len(checkpoint.Nodes) + index
			key, keyErr := machine.StateKey(node.State)
			if keyErr != nil {
				return Certificate{}, keyErr
			}
			seen[expandedStateKey(node.Replica, key)] = struct{}{}
		}
		checkpoint.Nodes = append(checkpoint.Nodes, newNodes...)
		checkpoint.Frontier = frontier
		checkpoint.Transitions += layerTransitions
		checkpoint.StateBytes += additionalBytes
		checkpoint.CompletedDepth++
		if err := checkpoint.seal(); err != nil {
			return Certificate{}, err
		}
		if options.Checkpoint != nil {
			if err := options.Checkpoint(checkpoint); err != nil {
				return Certificate{}, fmt.Errorf("publish native checkpoint: %w", err)
			}
		}
	}
	return compactCertificate(machine, view, viewDigest, checkpoint)
}

func seedCheckpoint(
	machine Machine,
	view protocolchecker.FirstOrderView,
	viewDigest string,
	options Options,
) (Checkpoint, error) {
	initials, err := machine.InitialStates()
	if err != nil {
		return Checkpoint{}, err
	}
	type initial struct {
		replica int
		state   protocolchecker.FirstOrderState
		key     string
	}
	seeds := make([]initial, 0, len(initials)*options.Replicas)
	for replica := 0; replica < options.Replicas; replica++ {
		for _, state := range initials {
			key, keyErr := machine.StateKey(state)
			if keyErr != nil {
				return Checkpoint{}, keyErr
			}
			seeds = append(seeds, initial{replica: replica, state: state, key: key})
		}
	}
	slices.SortFunc(seeds, func(left, right initial) int {
		if left.replica != right.replica {
			return left.replica - right.replica
		}
		return compareStrings(left.key, right.key)
	})
	if len(seeds) > options.Limits.MaxStates {
		return Checkpoint{}, &ResourceError{Resource: "states", Limit: options.Limits.MaxStates}
	}
	nodes := make([]ExpandedNode, len(seeds))
	frontier := make([]int, len(seeds))
	stateBytes := 0
	for index, seed := range seeds {
		nodes[index] = ExpandedNode{Replica: seed.replica, State: seed.state, Parent: -1}
		frontier[index] = index
		size, sizeErr := stateEncodedSize(seed.state)
		if sizeErr != nil {
			return Checkpoint{}, sizeErr
		}
		stateBytes += size
	}
	if stateBytes > options.Limits.MaxStateBytes {
		return Checkpoint{}, &ResourceError{Resource: "state-bytes", Limit: options.Limits.MaxStateBytes}
	}
	checkpoint := Checkpoint{
		FormatVersion: CheckpointFormatVersion, ViewDigest: viewDigest, SemanticHash: view.SemanticHash,
		Replicas: options.Replicas, Limits: options.Limits, CompletedDepth: -1,
		Nodes: nodes, Frontier: frontier, StateBytes: stateBytes,
	}
	if err := checkpoint.seal(); err != nil {
		return Checkpoint{}, err
	}
	return checkpoint, nil
}

func validateCheckpoint(
	checkpoint Checkpoint,
	machine Machine,
	view protocolchecker.FirstOrderView,
	viewDigest string,
	options Options,
) error {
	if err := checkpoint.validateDigest(); err != nil {
		return err
	}
	if checkpoint.ViewDigest != viewDigest || checkpoint.SemanticHash != view.SemanticHash ||
		checkpoint.Replicas != options.Replicas || checkpoint.Limits != options.Limits {
		return errors.New("native checkpoint does not match the selected view, replicas, and limits")
	}
	seen := make(map[string]struct{}, len(checkpoint.Nodes))
	for index, node := range checkpoint.Nodes {
		if node.Replica < 0 || node.Replica >= checkpoint.Replicas {
			return fmt.Errorf("checkpoint node %d has an invalid replica", index)
		}
		key, err := machine.StateKey(node.State)
		if err != nil {
			return err
		}
		compoundKey := expandedStateKey(node.Replica, key)
		if _, duplicate := seen[compoundKey]; duplicate {
			return fmt.Errorf("checkpoint has duplicate node %d", index)
		}
		seen[compoundKey] = struct{}{}
		if node.Parent == -1 {
			if node.Action != "" || node.Depth != 0 {
				return fmt.Errorf("checkpoint root node %d has predecessor evidence", index)
			}
			continue
		}
		if node.Parent < 0 || node.Parent >= index || node.Action == "" ||
			node.Replica != checkpoint.Nodes[node.Parent].Replica ||
			node.Depth != checkpoint.Nodes[node.Parent].Depth+1 {
			return fmt.Errorf("checkpoint node %d has an invalid predecessor", index)
		}
	}
	for _, index := range checkpoint.Frontier {
		if index < 0 || index >= len(checkpoint.Nodes) ||
			checkpoint.Nodes[index].Depth != checkpoint.CompletedDepth+1 {
			return errors.New("native checkpoint frontier is not the next complete BFS layer")
		}
	}
	return nil
}

func checkpointSeen(machine Machine, checkpoint Checkpoint) (map[string]struct{}, error) {
	seen := make(map[string]struct{}, len(checkpoint.Nodes))
	for _, node := range checkpoint.Nodes {
		key, err := machine.StateKey(node.State)
		if err != nil {
			return nil, err
		}
		seen[expandedStateKey(node.Replica, key)] = struct{}{}
	}
	return seen, nil
}

func expandFrontier(
	ctx context.Context,
	machine Machine,
	nodes []ExpandedNode,
	frontier []int,
	workers int,
) ([]expansion, error) {
	results := make([]expansion, len(frontier))
	jobs := make(chan int)
	workerCount := min(workers, len(frontier))
	var wait sync.WaitGroup
	wait.Add(workerCount)
	for range workerCount {
		go func() {
			defer wait.Done()
			for position := range jobs {
				if ctx.Err() != nil {
					continue
				}
				parentIndex := frontier[position]
				parent := nodes[parentIndex]
				steps, err := machine.Successors(parent.State)
				if err != nil {
					results[position].err = err
					continue
				}
				proposals := make([]proposal, len(steps))
				for index, step := range steps {
					key, keyErr := machine.StateKey(step.State)
					if keyErr != nil {
						results[position].err = keyErr
						break
					}
					proposals[index] = proposal{
						replica: parent.Replica, parent: parentIndex,
						action: step.Action, state: step.State, key: key,
					}
				}
				results[position].proposals = proposals
			}
		}()
	}
	for position := range frontier {
		select {
		case <-ctx.Done():
			close(jobs)
			wait.Wait()
			return nil, ctx.Err()
		case jobs <- position:
		}
	}
	close(jobs)
	wait.Wait()
	return results, nil
}

func compactCertificate(
	machine Machine,
	view protocolchecker.FirstOrderView,
	viewDigest string,
	checkpoint Checkpoint,
) (Certificate, error) {
	representatives := make([]CompactNode, 0, len(checkpoint.Nodes)/checkpoint.Replicas)
	globalToCompact := make(map[int]int)
	for globalIndex, node := range checkpoint.Nodes {
		if node.Replica != 0 {
			continue
		}
		parent := -1
		if node.Parent >= 0 {
			var found bool
			parent, found = globalToCompact[node.Parent]
			if !found {
				return Certificate{}, errors.New("replica representative parent was not emitted first")
			}
		}
		globalToCompact[globalIndex] = len(representatives)
		representatives = append(representatives, CompactNode{
			State: node.State, Parent: parent, Action: node.Action, Depth: node.Depth,
		})
	}
	closureTransitions := 0
	maxDepth := 0
	for _, node := range representatives {
		steps, err := machine.Successors(node.State)
		if err != nil {
			return Certificate{}, err
		}
		closureTransitions += len(steps)
		maxDepth = max(maxDepth, node.Depth)
	}
	certificate := Certificate{
		FormatVersion: CertificateFormatVersion, ViewVersion: view.FormatVersion,
		ViewDigest: viewDigest, Target: view.Target, Property: view.Property,
		World: view.World, Variant: view.Variant, SemanticHash: view.SemanticHash,
		Termination: "exhausted", Nodes: representatives,
		Closure: ClosureCertificate{
			Kind: closureRecomputedSuccessors, ClosedRepresentatives: len(representatives),
			RecomputedTransitions: closureTransitions,
		},
		Symmetry: SymmetryCertificate{
			Kind: symmetryReplicatedWorlds, Replicas: checkpoint.Replicas,
			Representatives: len(representatives), ExpandedStates: len(checkpoint.Nodes),
		},
		Statistics: Statistics{
			ExpandedStates: len(checkpoint.Nodes), RepresentativeStates: len(representatives),
			Transitions: checkpoint.Transitions, StateBytes: checkpoint.StateBytes, MaxDepth: maxDepth,
		},
	}
	if err := certificate.seal(); err != nil {
		return Certificate{}, err
	}
	if err := certificate.Validate(view); err != nil {
		return Certificate{}, err
	}
	return certificate, nil
}

func publishResourceCheckpoint(checkpoint Checkpoint, options Options, resourceErr error) error {
	if options.Checkpoint == nil {
		return resourceErr
	}
	if err := options.Checkpoint(checkpoint); err != nil {
		return errors.Join(resourceErr, fmt.Errorf("publish native checkpoint: %w", err))
	}
	return resourceErr
}

func actionsFor(nodes []ExpandedNode, index int) []protocolcatalog.ActionKind {
	var reversed []protocolcatalog.ActionKind
	for index >= 0 && nodes[index].Parent >= 0 {
		reversed = append(reversed, nodes[index].Action)
		index = nodes[index].Parent
	}
	slices.Reverse(reversed)
	return reversed
}

func compareProposal(left, right proposal) int {
	if left.replica != right.replica {
		return left.replica - right.replica
	}
	if order := compareStrings(left.key, right.key); order != 0 {
		return order
	}
	if left.parent != right.parent {
		return left.parent - right.parent
	}
	return compareStrings(string(left.action), string(right.action))
}

func compareStrings(left, right string) int {
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}

func expandedStateKey(replica int, stateKey string) string {
	return fmt.Sprintf("%d\x00%s", replica, stateKey)
}
