package backend

import (
	"bytes"
	"encoding/json"
	"errors"
	"maps"
	"reflect"
	"slices"

	"go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/backends/target"
	simulation "go.temporal.io/server/tools/gomad/internal/backends/wasm/cluster"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	model "go.temporal.io/server/tools/gomad/internal/model/cluster"
)

type StressExecution struct {
	Padding          string
	Jitter           bool
	Diagnostics      bool
	ConfigureRequest func(*wasi.Request)
	BindScenario     func(wasi.Request, model.Spec, *simulation.Provider) (model.Spec, error)
}

type ExecutionObservation struct {
	Target               target.Prepared
	DeclaredInputsSHA256 record.SHA256
	Spec                 *model.Spec
	Incarnations         []simulation.IncarnationResult
	Cluster              *wasi.ClusterResult
}

func stressScenarioIdentity(spec model.Spec, provider *simulation.Provider) (record.SHA256, error) {
	boots := make(map[model.BootID]json.RawMessage, len(provider.Boots))
	for id, request := range provider.Boots {
		configuration, err := simulation.BootConfiguration(request)
		if err != nil {
			return "", err
		}
		boots[id] = configuration
	}
	encoded, err := json.Marshal(struct {
		Spec                       model.Spec
		Boots                      map[model.BootID]json.RawMessage
		Broker                     wasi.BrokerConfig
		IncarnationBoots           []simulation.IncarnationBoot
		OutputCrashes, OutputStops []simulation.OutputCrash
		OutputCrashGroups          []simulation.OutputCrashGroup
	}{spec, boots, provider.Broker, provider.IncarnationBoots, provider.OutputCrashes, provider.OutputStops, provider.OutputCrashGroups})
	if err != nil {
		return "", err
	}
	return record.HashBytes(encoded), nil
}

func (p *Provider) ForStressReplay() (backends.Provider, error) {
	if p.options.Config.Profile != wasi.CooperativeProfile {
		return nil, errors.New("stress replay requires a cooperative WASM installation")
	}
	options := p.options
	options.StressExecution = &StressExecution{}
	options.ObserveResults = nil
	return New(options)
}

func (p *Provider) stressRequest(request *wasi.Request) error {
	stress := p.options.StressExecution
	if stress == nil {
		return nil
	}
	before, err := json.Marshal(request)
	if err != nil {
		return err
	}
	request.SetStressHostConditions(stress.Padding, stress.Jitter)
	if stress.ConfigureRequest != nil {
		stress.ConfigureRequest(request)
	}
	after, err := json.Marshal(request)
	if err != nil {
		return err
	}
	if !bytes.Equal(before, after) {
		return errors.New("stress host configuration changed declared WASM inputs")
	}
	return nil
}

func (p *Provider) bindStressScenario(base wasi.Request, spec model.Spec, provider *simulation.Provider) (model.Spec, error) {
	encoded, err := json.Marshal(spec)
	if err != nil {
		return spec, err
	}
	var original model.Spec
	if err := json.Unmarshal(encoded, &original); err != nil {
		return spec, err
	}
	boots := make(map[model.BootID][]byte, len(provider.Boots))
	for _, id := range slices.Sorted(maps.Keys(provider.Boots)) {
		request := provider.Boots[id]
		request.Config.Environment = nil
		request.Config.EntropyKey = [32]byte{}
		boots[id], err = json.Marshal(request)
		if err != nil {
			return spec, err
		}
	}
	broker, err := json.Marshal(provider.Broker)
	if err != nil {
		return spec, err
	}
	bindings, err := json.Marshal(struct {
		Boots []simulation.IncarnationBoot
		Stops []simulation.OutputCrash
	}{provider.IncarnationBoots, provider.OutputStops})
	if err != nil {
		return spec, err
	}
	crashes := append([]simulation.OutputCrash(nil), provider.OutputCrashes...)
	spec, err = p.options.StressExecution.BindScenario(base, spec, provider)
	if err != nil {
		return spec, err
	}
	if err := model.ValidateSpec(spec); err != nil {
		return spec, err
	}
	if len(boots) != len(provider.Boots) || len(spec.Nodes) != len(original.Nodes) || provider.Oracles != nil {
		return spec, errors.New("stress scenario changed prepared topology or oracle policy")
	}
	for _, id := range slices.Sorted(maps.Keys(provider.Boots)) {
		request := provider.Boots[id]
		if _, err := wasi.NewEnvironment(request.Config); err != nil {
			return spec, err
		}
		request.Config.Environment = nil
		request.Config.EntropyKey = [32]byte{}
		encoded, err := json.Marshal(request)
		if err != nil {
			return spec, err
		}
		if !bytes.Equal(boots[id], encoded) {
			return spec, errors.New("stress scenario changed prepared boot inputs outside environment and entropy")
		}
	}
	for i, node := range spec.Nodes {
		configuration, err := simulation.BootConfiguration(provider.Boots[node.Boot])
		if err != nil {
			return spec, err
		}
		if !bytes.Equal(node.Config, configuration) {
			return spec, errors.New("stress scenario node configuration differs from bound boot")
		}
		original.Nodes[i].Config = node.Config
	}
	if !reflect.DeepEqual(original, spec) {
		return spec, errors.New("stress scenario changed prepared topology or lifecycle inputs")
	}
	afterBroker, err := json.Marshal(provider.Broker)
	if err != nil {
		return spec, err
	}
	afterBindings, err := json.Marshal(struct {
		Boots []simulation.IncarnationBoot
		Stops []simulation.OutputCrash
	}{provider.IncarnationBoots, provider.OutputStops})
	if err != nil {
		return spec, err
	}
	if !bytes.Equal(broker, afterBroker) || !bytes.Equal(bindings, afterBindings) {
		return spec, errors.New("stress scenario changed prepared broker or lifecycle bindings")
	}
	boundaries := make(map[model.NodeHandle][]byte, len(crashes))
	for _, crash := range crashes {
		boundaries[crash.Handle] = crash.Marker
	}
	consume := func(handle model.NodeHandle, marker []byte) bool {
		expected, exists := boundaries[handle]
		if !exists || !bytes.Equal(expected, marker) {
			return false
		}
		delete(boundaries, handle)
		return true
	}
	for _, crash := range provider.OutputCrashes {
		if !consume(crash.Handle, crash.Marker) {
			return spec, errors.New("stress scenario changed prepared output crash boundary")
		}
	}
	for _, group := range provider.OutputCrashGroups {
		if len(group.Handles) == 0 {
			return spec, errors.New("stress scenario has an empty output crash group")
		}
		for _, handle := range group.Handles {
			if !consume(handle, group.Marker) {
				return spec, errors.New("stress scenario changed prepared output crash boundary")
			}
		}
	}
	if len(boundaries) != 0 {
		return spec, errors.New("stress scenario omitted a prepared output crash boundary")
	}
	return spec, nil
}
