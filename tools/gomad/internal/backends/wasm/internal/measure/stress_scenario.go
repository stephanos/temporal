package measure

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.temporal.io/server/tools/gomad/internal/backends/target"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/backend"
	simulation "go.temporal.io/server/tools/gomad/internal/backends/wasm/cluster"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	model "go.temporal.io/server/tools/gomad/internal/model/cluster"
	netmodel "go.temporal.io/server/tools/gomad/internal/model/network"
	"go.temporal.io/server/tools/gomad/internal/model/volume"
)

var stressClusterArgs = []string{"-test.v", "-test.count=1", "-test.timeout=1h", "-test.run=^TestIndependentClusterReplicationRecovery$", "-persistenceType=sql", "-persistenceDriver=sqlite"}

func stressClusterNetwork() *wasi.NetworkModelV2 {
	return &wasi.NetworkModelV2{Profile: wasi.NetworkStreamProfile, Limits: netmodel.Limits{Listeners: 128, Connections: 4096, Bytes: 128 << 20, ConnectionBytes: 4 << 20, StallNanos: 10_000_000_000, Transitions: 1 << 20}}
}

func stressClusterScenario(storage *wasi.StorageModel) (simulation.OperatorScenario, error) {
	if storage == nil || len(storage.Volumes) != 1 {
		return simulation.OperatorScenario{}, errors.New("two-cluster stress requires one durable volume per node")
	}
	scenario := simulation.OperatorScenario{Schema: simulation.OperatorSchema, Spec: model.Spec{Schema: model.SpecSchema, Backend: model.BackendWASM, Fidelity: model.FidelitySimulationModel, Limits: model.DefaultLimits()}}
	for index, name := range []string{"active", "standby"} {
		initial := model.NodeHandle{Node: model.NodeID(name), Incarnation: 1}
		reopened := model.NodeHandle{Node: model.NodeID(name), Incarnation: 2}
		volumeConfig := *storage
		volumeConfig.Volumes = append([]volume.VolumeConfig(nil), storage.Volumes...)
		volumeConfig.Volumes[0].ID = name
		for _, phase := range []string{"commit", "reopen"} {
			scenario.Boots = append(scenario.Boots, simulation.OperatorBoot{ID: model.BootID(name + "-" + phase), Args: append([]string(nil), stressClusterArgs...), Storage: &volumeConfig})
		}
		scenario.Spec.Nodes = append(scenario.Spec.Nodes, model.NodeSpec{ID: initial.Node, Boot: model.BootID(name + "-commit"), Address: fmt.Sprintf("10.0.0.%d", index+1), Volumes: []model.VolumeMount{{Volume: model.VolumeID(name), Path: storage.Volumes[0].Path}}})
		scenario.Spec.Volumes = append(scenario.Spec.Volumes, model.VolumeSpec{ID: model.VolumeID(name), CapacityBytes: storage.Volumes[0].CapacityBytes})
		scenario.OutputCrashes = append(scenario.OutputCrashes, simulation.OutputCrash{Handle: initial, Marker: []byte(temporalObservationPrefix)})
		scenario.IncarnationBoots = append(scenario.IncarnationBoots, simulation.IncarnationBoot{Handle: reopened, Boot: model.BootID(name + "-reopen")})
	}
	scenario.Spec.Links = []model.LinkSpec{{From: "active", To: "standby", Enabled: true}, {From: "standby", To: "active", Enabled: true}}
	scenario.NetworkFaults = temporalBrokerConfiguration(stressClusterNetwork()).Faults
	for _, phase := range []struct {
		action      string
		incarnation uint64
	}{{"start", 1}, {"crash", 1}, {"restart", 2}, {"wait", 2}} {
		for _, name := range []model.NodeID{"active", "standby"} {
			scenario.Actions = append(scenario.Actions, simulation.OperatorAction{Action: phase.action, Handle: model.NodeHandle{Node: name, Incarnation: phase.incarnation}})
		}
	}
	encoded, err := json.Marshal(scenario)
	if err != nil {
		return scenario, err
	}
	_, _, err = simulation.NormalizeOperator(encoded)
	return scenario, err
}

func stressBindCluster(base wasi.Request, spec model.Spec, provider *simulation.Provider) (model.Spec, error) {
	for id, request := range provider.Boots {
		name, phase, ok := strings.Cut(string(id), "-")
		if !ok || name != "active" && name != "standby" || phase != "commit" && phase != "reopen" {
			return spec, fmt.Errorf("unknown stress boot %s", id)
		}
		peer := "10.0.0.1"
		if name == "active" {
			peer = "10.0.0.2"
		}
		request.Config.Environment = append(append([]string(nil), base.Config.Environment...), "GOMAD_TEMPORAL_CLUSTER="+name, "GOMAD_TEMPORAL_PHASE="+phase, "GOMAD_TEMPORAL_REMOTE_FRONTEND="+peer+":7233", "GOMAD_TEMPORAL_REMOTE_HTTP="+peer+":7243", "GOMAD_TEMPORAL_REMOTE_HISTORY="+peer+":7234")
		request.Config.EntropyKey = sha256.Sum256([]byte("gomad-wasm-temporal-replication/v1/" + name))
		provider.Boots[id] = request
	}
	provider.OutputCrashes = nil
	provider.OutputCrashGroups = []simulation.OutputCrashGroup{{Handles: []model.NodeHandle{{Node: "active", Incarnation: 1}, {Node: "standby", Incarnation: 1}}, Marker: []byte(temporalObservationPrefix)}}
	for index, node := range spec.Nodes {
		configuration, err := simulation.BootConfiguration(provider.Boots[node.Boot])
		if err != nil {
			return spec, err
		}
		spec.Nodes[index].Config = configuration
	}
	return spec, model.ValidateSpec(spec)
}

func stressTarget(ctx context.Context, workload string, request wasi.Request) (target.Spec, error) {
	repo := scopeFor(ctx).options.Repo
	spec := target.Spec{Backend: backend.Name, Kind: target.KindGoTest, WorkingDir: repo, BuildTags: backend.TemporalGuestBuildTags()}
	switch workload {
	case "runtime":
		spec.Kind = target.KindGoRun
		spec.WorkingDir = filepath.Join(repo, "tools/gomad")
		spec.Source = "./internal/backends/wasm/testdata/runtime_probes"
		spec.BuildTags = nil
	case "workflow", "activity":
		spec.Source = "./tests"
		spec.Args = append([]string(nil), request.Config.Args[1:]...)
	case "two-cluster":
		storage, err := os.ReadFile(filepath.Join(repo, "tools/gomad/internal/backends/wasm/testdata/temporal_storage/recovery_test.go"))
		if err != nil {
			return spec, err
		}
		replication, err := os.ReadFile(filepath.Join(repo, "tools/gomad/internal/backends/wasm/testdata/temporal_broker/replication_test.go"))
		if err != nil {
			return spec, err
		}
		spec.Source = "./" + filepath.ToSlash(filepath.Join(".tmp", "wasm-backend-temporal", "temporal_broker-"+digest(append(storage, replication...))[:16]))
		spec.Args = append([]string(nil), stressClusterArgs...)
		scenario, err := stressClusterScenario(request.Config.Storage)
		if err != nil {
			return spec, err
		}
		spec.Scenario, err = json.Marshal(scenario)
		if err != nil {
			return spec, err
		}
	default:
		return spec, fmt.Errorf("unknown stress workload %q", workload)
	}
	return spec, nil
}

func stressClusterOracles(ctx context.Context, root string, incarnations []simulation.IncarnationResult) error {
	if len(incarnations) != 4 {
		return fmt.Errorf("two-cluster stress executed %d incarnations, expected four", len(incarnations))
	}
	originals := map[model.NodeID]temporalRecoveryObservation{}
	var history []byte
	identities := map[string]bool{}
	for _, incarnation := range incarnations {
		r := incarnation.Result
		value, err := temporalRecoveryValue(ctx, r.Stdout)
		if err != nil {
			return err
		}
		if incarnation.Handle.Incarnation == 1 {
			if r.Termination != "crash" || !r.Reaped || r.StorageImage == nil {
				return fmt.Errorf("%v did not retain crashed durable state", incarnation.Handle)
			}
			if len(history) != 0 && !bytes.Equal(history, value.History) {
				return errors.New("independent histories did not converge after heal")
			}
			history = value.History
			if identities[value.ClusterID] {
				return errors.New("independent clusters have the same identity")
			}
			identities[value.ClusterID] = true
			originals[incarnation.Handle.Node] = value
		}
	}
	for _, incarnation := range incarnations {
		if incarnation.Handle.Incarnation != 2 {
			continue
		}
		if err := requireExit(ctx, string(incarnation.Handle.Node), incarnation.Result, 0); err != nil {
			return err
		}
		value, err := temporalRecoveryValue(ctx, incarnation.Result.Stdout)
		if err != nil {
			return err
		}
		original, ok := originals[incarnation.Handle.Node]
		if !ok {
			return errors.New("reopened node lacks a committed incarnation")
		}
		if err := temporalHistoryOracles(ctx, root, "temporal.replication."+string(incarnation.Handle.Node)+"-recovered", original, value); err != nil {
			return err
		}
		if !bytes.Equal(history, value.History) || original.ClusterID != value.ClusterID {
			return errors.New("committed history or cluster identity changed after whole-cluster crash")
		}
	}
	return nil
}
