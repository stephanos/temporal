package protocol

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	hostfs "go.temporal.io/server/tools/gomad3/internal/host/fs"
)

type networkWireSchema struct {
	ConfigMagic         string `json:"config_magic"`
	FinishMagic         string `json:"finish_magic"`
	ErrorMagic          string `json:"error_magic"`
	MaximumBytes        uint64 `json:"maximum_bytes"`
	MaximumStringBytes  uint64 `json:"maximum_string_bytes"`
	MaximumNodes        uint64 `json:"maximum_nodes"`
	MaximumLinks        uint64 `json:"maximum_links"`
	MaximumListeners    uint64 `json:"maximum_listeners"`
	MaximumConnections  uint64 `json:"maximum_connections"`
	MaximumNetworkBytes uint64 `json:"maximum_network_bytes"`
	MaximumStallNanos   uint64 `json:"maximum_stall_nanos"`
	MaximumTransitions  uint64 `json:"maximum_transitions"`
}

type networkWireTemplateData struct {
	Package        string
	Host           bool
	ErrorPrefix    string
	NodeID         string
	TransitionKind string
	Outcome        string
	Names          map[string]string
	Schema         networkWireSchema
}

func networkWireConsumer(host bool, definition networkWireSchema) networkWireTemplateData {
	data := networkWireTemplateData{
		Package: "gomadio", ErrorPrefix: "simulation network", NodeID: "string",
		TransitionKind: "string", Outcome: "string", Schema: definition,
		Names: map[string]string{
			"maximumRuntimeNetworkWireBytes":       "maximumSimulationWireBytes",
			"maximumRuntimeNetworkWireStringBytes": "maximumSimulationWireStringBytes",
			"runtimeNetworkConfigMagic":            "simulationConfigMagic",
			"runtimeNetworkFinishMagic":            "simulationFinishMagic",
			"runtimeNetworkErrorMagic":             "simulationErrorMagic",
			"MaximumNodes":                         "maximumSimulationNodes",
			"MaximumDirectionalLinks":              "maximumSimulationLinks",
			"MaximumNetworkListeners":              "maximumSimulationListeners",
			"MaximumNetworkConnections":            "maximumSimulationConnections",
			"MaximumNetworkBytes":                  "maximumSimulationNetworkBytes",
			"MaximumNetworkStallNanos":             "maximumSimulationStallNanos",
			"MaximumNetworkTransitions":            "maximumSimulationTransitions",
			"runtimeNetworkWireEncoder":            "simulationWireEncoder",
			"runtimeNetworkWireDecoder":            "simulationWireDecoder",
			"newRuntimeNetworkWireDecoder":         "newSimulationWireDecoder",
			"appendRuntimeNetworkUint64":           "appendSimulationUint64",
			"readRuntimeNetworkUint64":             "readSimulationUint64",
			"encodeRuntimeNetworkRecord":           "encodeSimulationRecord",
			"decodeRuntimeNetworkRecord":           "decodeSimulationRecord",
			"encodeRuntimeNetworkTransition":       "encodeSimulationTransition",
			"decodeRuntimeNetworkTransition":       "decodeSimulationTransition",
			"encodeRuntimeNetworkEndpoint":         "encodeSimulationEndpoint",
			"decodeRuntimeNetworkEndpoint":         "decodeSimulationEndpoint",
			"encodeRuntimeNetworkSnapshot":         "encodeSimulationSnapshot",
			"decodeRuntimeNetworkSnapshot":         "decodeSimulationSnapshot",
			"encodeRuntimeNetworkTransitions":      "encodeSimulationTransitions",
			"encodeRuntimeNetworkSnapshotIdentity": "encodeSimulationSnapshotIdentity",
			"NetworkRecord":                        "simulationRecord",
			"NetworkTransition":                    "simulationTransition",
			"NetworkEndpoint":                      "simulationEndpoint",
			"NetworkSnapshot":                      "simulationSnapshot",
			"NetworkNodeSnapshot":                  "simulationNodeSnapshot",
			"NetworkLinkSnapshot":                  "simulationLink",
			"NetworkListenerSnapshot":              "simulationListenerSnapshot",
			"NetworkConnectionSnapshot":            "simulationConnectionSnapshot",
			"Limits":                               "simulationLimits",
			"NetworkConnectionBytes":               "ConnectionBytes",
			"NetworkStallNanos":                    "StallNanos",
			"validateRuntimeNetworkLimits":         "validateSimulationLimits",
			"validNetworkTransition":               "validSimulationTransition",
			"validNetworkEndpoint":                 "validSimulationWireEndpoint",
			"networkTransitionLane":                "simulationTransitionLane",
			"networkEndpointKey":                   "simulationEndpointKey",
		},
	}
	if host {
		data.Package, data.Host, data.ErrorPrefix = "wire", true, "simulation runtime network"
		data.NodeID, data.TransitionKind, data.Outcome = "NodeID", "NetworkTransitionKind", "NetworkOutcome"
		for name := range data.Names {
			data.Names[name] = name
		}
		data.Names["MaximumNodes"] = "maximumRuntimeNetworkWireNodes"
		data.Names["MaximumDirectionalLinks"] = "maximumRuntimeNetworkWireLinks"
		data.Names["MaximumNetworkListeners"] = "maximumRuntimeNetworkWireListeners"
		data.Names["MaximumNetworkConnections"] = "maximumRuntimeNetworkWireConnections"
		data.Names["MaximumNetworkBytes"] = "maximumRuntimeNetworkWireNetworkBytes"
		data.Names["MaximumNetworkStallNanos"] = "maximumRuntimeNetworkWireStallNanos"
		data.Names["MaximumNetworkTransitions"] = "maximumRuntimeNetworkWireTransitions"
	}
	return data
}

func generateNetworkProtocols(root string, check bool) error {
	contents, err := os.ReadFile(filepath.Join(root, "internal/model/schema/networkwire.json"))
	if err != nil {
		return fmt.Errorf("read network wire schema: %w", err)
	}
	var definition networkWireSchema
	if err := json.Unmarshal(contents, &definition); err != nil {
		return fmt.Errorf("decode network wire schema: %w", err)
	}
	if len(definition.ConfigMagic) != 8 || len(definition.FinishMagic) != 8 || len(definition.ErrorMagic) != 8 ||
		definition.MaximumBytes == 0 || definition.MaximumStringBytes == 0 ||
		definition.MaximumStringBytes > definition.MaximumBytes || definition.MaximumNodes == 0 ||
		definition.MaximumLinks == 0 || definition.MaximumListeners == 0 || definition.MaximumConnections == 0 ||
		definition.MaximumTransitions == 0 ||
		definition.MaximumNetworkBytes != 1<<30 || definition.MaximumStallNanos != 9223372036854775806 {
		return fmt.Errorf("network wire schema has invalid framing or limits")
	}
	for _, target := range []struct {
		Path string
		Host bool
	}{
		{"internal/model/cluster/internal/wire/network.go", true},
		{"internal/backends/native/toolchain/runtime/overlay/src/internal/gomadio/simulation_wire.go", false},
	} {
		generated, err := generate(filepath.Join(root, "internal/model/schema/networkwire.go.tmpl"), networkWireConsumer(target.Host, definition))
		if err != nil {
			return err
		}
		if target.Host {
			generated, err = clusterWireSource(generated)
			if err != nil {
				return err
			}
		}
		path := filepath.Join(root, filepath.FromSlash(target.Path))
		if check {
			current, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(current, generated) {
				return fmt.Errorf("generated network wire codec is stale: %s", target.Path)
			}
		} else if err := hostfs.Replace(path, generated, 0o644); err != nil {
			return fmt.Errorf("write generated network wire codec: %w", err)
		}
	}
	return nil
}
