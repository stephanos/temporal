package simulation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	model "go.temporal.io/server/tools/gomad/internal/model/cluster"
	netmodel "go.temporal.io/server/tools/gomad/internal/model/network"
	"go.temporal.io/server/tools/gomad/internal/model/strictjson"
	"go.temporal.io/server/tools/gomad/internal/model/world"
)

const OperatorSchema = "gomad-wasm.scenario/v1"

type OperatorScenario struct {
	Schema           string                    `json:"schema"`
	Spec             model.Spec                `json:"spec"`
	Boots            []OperatorBoot            `json:"boots"`
	Actions          []OperatorAction          `json:"actions"`
	IncarnationBoots []IncarnationBoot         `json:"incarnation_boots,omitempty"`
	OutputCrashes    []OutputCrash             `json:"output_crashes,omitempty"`
	OutputStops      []OutputCrash             `json:"output_stops,omitempty"`
	NetworkFaults    []wasi.BrokerNetworkFault `json:"network_faults,omitempty"`
}

type OperatorBoot struct {
	ID      model.BootID       `json:"id"`
	Args    []string           `json:"args"`
	Storage *wasi.StorageModel `json:"storage,omitempty"`
}

type OperatorAction struct {
	Action string           `json:"action"`
	Handle model.NodeHandle `json:"handle"`
}

func NormalizeOperator(data []byte) (OperatorScenario, []byte, error) {
	var s OperatorScenario
	if len(data) == 0 || len(data) > 256<<10 {
		return s, nil, errors.New("WASM scenario size exceeds bounds")
	}
	if err := strictjson.Decode(data, &s); err != nil {
		return s, nil, err
	}
	if s.Schema != OperatorSchema || s.Spec.Backend != model.BackendWASM || s.Spec.Seed != 0 || s.Spec.Replay != nil || s.Spec.Exploration != nil || s.Spec.ScenarioChoices != nil || s.Spec.Faults != nil {
		return s, nil, errors.New("unsupported WASM scenario schema or control plan")
	}
	if err := model.ValidateSpec(s.Spec); err != nil {
		return s, nil, err
	}
	if uint64(len(s.NetworkFaults)) > s.Spec.Limits.FaultActions {
		return s, nil, &model.CapacityError{Resource: "fault_actions", Required: uint64(len(s.NetworkFaults)), Maximum: s.Spec.Limits.FaultActions}
	}
	if len(s.Boots) == 0 || uint64(len(s.Boots)) > model.MaximumScenarioActions || len(s.Actions) == 0 || uint64(len(s.Actions)) > s.Spec.Limits.ScenarioActions {
		return s, nil, errors.New("WASM scenario boot/action capacity")
	}
	boots := map[model.BootID]OperatorBoot{}
	for _, b := range s.Boots {
		if b.ID == "" {
			return s, nil, errors.New("empty WASM boot ID")
		}
		if _, ok := boots[b.ID]; ok {
			return s, nil, errors.New("duplicate WASM boot ID")
		}
		boots[b.ID] = b
	}
	nodes := map[model.NodeID]model.NodeSpec{}
	for _, n := range s.Spec.Nodes {
		if len(n.Config) != 0 {
			return s, nil, errors.New("WASM boot configuration is preparation-owned")
		}
		if _, ok := boots[n.Boot]; !ok {
			return s, nil, errors.New("missing WASM boot")
		}
		nodes[n.ID] = n
	}
	bindings := map[model.NodeHandle]model.BootID{}
	for _, b := range s.IncarnationBoots {
		_, known := nodes[b.Handle.Node]
		_, boot := boots[b.Boot]
		if !known || !boot || b.Handle.Incarnation < 2 || bindings[b.Handle] != "" {
			return s, nil, errors.New("invalid WASM incarnation boot")
		}
		bindings[b.Handle] = b.Boot
	}
	boundaries := map[model.NodeHandle]string{}
	for _, set := range []struct {
		kind  string
		items []OutputCrash
	}{{"crash", s.OutputCrashes}, {"stop", s.OutputStops}} {
		for _, b := range set.items {
			if _, ok := nodes[b.Handle.Node]; !ok || b.Handle.Incarnation == 0 || len(b.Marker) == 0 || len(b.Marker) > 4096 || boundaries[b.Handle] != "" {
				return s, nil, errors.New("invalid WASM lifecycle boundary")
			}
			boundaries[b.Handle] = set.kind
		}
	}
	current := map[model.NodeID]uint64{}
	active := map[model.NodeID]bool{}
	used := map[model.NodeHandle]bool{}
	usedBindings := map[model.NodeHandle]bool{}
	for _, a := range s.Actions {
		if _, ok := nodes[a.Handle.Node]; !ok {
			return s, nil, errors.New("invalid WASM lifecycle reference")
		}
		n := a.Handle.Node
		switch a.Action {
		case "start":
			if current[n] != 0 || a.Handle.Incarnation != 1 {
				return s, nil, errors.New("invalid WASM start order")
			}
			current[n] = 1
			active[n] = true
		case "restart":
			if current[n] == 0 || active[n] || a.Handle.Incarnation != current[n]+1 {
				return s, nil, errors.New("invalid WASM restart order")
			}
			current[n]++
			active[n] = true
			if bindings[a.Handle] != "" {
				usedBindings[a.Handle] = true
			}
		case "wait", "crash", "stop":
			if !active[n] || current[n] != a.Handle.Incarnation {
				return s, nil, errors.New("invalid WASM terminal order")
			}
			if a.Action != "wait" {
				boot := nodes[n].Boot
				if selected := bindings[a.Handle]; selected != "" {
					boot = selected
				}
				storageBound := a.Action == "crash" && storageCrashBound(wasi.Request{Config: wasi.Config{Storage: boots[boot].Storage}})
				if boundaries[a.Handle] != a.Action && !storageBound {
					return s, nil, errors.New("undeclared WASM lifecycle boundary")
				}
				if boundaries[a.Handle] == a.Action {
					used[a.Handle] = true
				}
			}
			active[n] = false
		default:
			return s, nil, fmt.Errorf("unsupported WASM scenario action %q", a.Action)
		}
	}
	if len(used) != len(boundaries) || len(usedBindings) != len(bindings) {
		return s, nil, errors.New("unused WASM lifecycle binding")
	}
	for n := range nodes {
		if current[n] == 0 || active[n] {
			return s, nil, errors.New("WASM scenario must terminate every node")
		}
	}
	canonical, err := json.Marshal(s)
	if len(canonical) > 256<<10 {
		return s, nil, errors.New("normalized WASM scenario size exceeds bounds")
	}
	return s, canonical, err
}

func (s OperatorScenario) Bind(base wasi.Request) (model.Spec, *Provider, error) {
	spec := s.Spec
	if base.Runtime == nil || base.Runtime.Choice == nil || len(base.Config.Args) == 0 || base.Config.Profile != wasi.CooperativeProfile || base.Config.Clock.ReadStepNanos != 0 {
		return spec, nil, errors.New("WASM scenario requires fresh cooperative boot inputs")
	}
	spec.Seed = base.Runtime.Seed
	p := &Provider{Boots: map[model.BootID]wasi.Request{}, IncarnationBoots: s.IncarnationBoots, OutputCrashes: s.OutputCrashes, OutputStops: s.OutputStops}
	p.Broker.Faults = s.NetworkFaults
	for _, b := range s.Boots {
		r := base
		r.Config = base.Config
		r.Config.Storage = b.Storage
		if b.Args != nil {
			r.Config.Args = append([]string{base.Config.Args[0]}, b.Args...)
		}
		if _, err := wasi.NewEnvironment(r.Config); err != nil {
			return spec, nil, err
		}
		if err := validateBootNetworkLimits(r.Config.NetworkV2, spec.Limits); err != nil {
			return spec, nil, err
		}
		p.Boots[b.ID] = r
	}
	spec.Nodes = append([]model.NodeSpec(nil), spec.Nodes...)
	for i, n := range spec.Nodes {
		r := p.Boots[n.Boot]
		if len(n.Volumes) != storageMountCount(r.Config.Storage) {
			return spec, nil, errors.New("WASM boot durable mounts differ from scenario")
		}
		for _, m := range n.Volumes {
			found := false
			for _, v := range r.Config.Storage.Volumes {
				for _, declared := range spec.Volumes {
					found = found || (m.Volume == model.VolumeID(v.ID) && m.Path == v.Path && declared.ID == m.Volume && declared.CapacityBytes == v.CapacityBytes)
				}
			}
			if !found {
				return spec, nil, errors.New("WASM boot durable volume differs from scenario")
			}
		}
		configuration, err := BootConfiguration(r)
		if err != nil {
			return spec, nil, err
		}
		spec.Nodes[i].Config = configuration
	}
	for _, binding := range s.IncarnationBoots {
		initial := p.Boots[nodesBoot(spec, binding.Handle.Node)]
		restart := p.Boots[binding.Boot]
		a, _ := json.Marshal(initial.Config.Storage)
		b, _ := json.Marshal(restart.Config.Storage)
		if string(a) != string(b) {
			return spec, nil, errors.New("WASM restart durable model differs from initial boot")
		}
	}
	broker := p.Broker
	broker.Network = base.Config.NetworkV2
	broker.Addresses = map[string]string{}
	broker.StrictTopology = true
	broker.Limits = world.Limits{MaxRequests: spec.Limits.NetworkTransitions * 2, MaxEvents: spec.Limits.NetworkTransitions * 2, MaxQueuedEvents: min(netmodel.MaximumPendingOperations, spec.Limits.NetworkTransitions*2), MaxTransitions: spec.Limits.NetworkTransitions * 8, MaxPayloadBytes: spec.Limits.ObservationBytes, MaxStringBytes: 1024}
	for _, n := range spec.Nodes {
		broker.Nodes = append(broker.Nodes, wasi.BrokerNode{ID: string(n.ID), Incarnation: 1})
		broker.Addresses[string(n.ID)] = n.Address
	}
	for _, link := range spec.Links {
		broker.Links = append(broker.Links, wasi.BrokerLink{From: string(link.From), To: string(link.To), Enabled: link.Enabled, DelayNanos: link.DelayNanos})
	}
	if _, err := wasi.NewBroker(broker); err != nil {
		return spec, nil, err
	}
	return spec, p, model.ValidateSpec(spec)
}

func storageMountCount(s *wasi.StorageModel) int {
	if s == nil {
		return 0
	}
	return len(s.Volumes)
}
func nodesBoot(s model.Spec, n model.NodeID) model.BootID {
	for _, node := range s.Nodes {
		if node.ID == n {
			return node.Boot
		}
	}
	return ""
}

func (s OperatorScenario) Run(ctx context.Context, c model.Cluster) error {
	for _, a := range s.Actions {
		var err error
		switch a.Action {
		case "start":
			_, err = c.Start(ctx, a.Handle.Node)
		case "restart":
			_, err = c.Restart(ctx, a.Handle.Node)
		case "wait":
			var result model.NodeResult
			result, err = c.Wait(ctx, a.Handle)
			if err == nil && result.State == model.NodeStateFailed {
				return nil
			}
		case "crash":
			err = c.Crash(ctx, a.Handle)
		case "stop":
			err = c.Stop(ctx, a.Handle)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
