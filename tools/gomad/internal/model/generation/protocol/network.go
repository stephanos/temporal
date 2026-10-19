package protocol

import (
	"encoding/json"
	"fmt"
	"os"
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

func readNetworkWireSchema(path string) (networkWireSchema, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return networkWireSchema{}, fmt.Errorf("read network wire schema: %w", err)
	}
	var definition networkWireSchema
	if err := json.Unmarshal(contents, &definition); err != nil {
		return networkWireSchema{}, fmt.Errorf("decode network wire schema: %w", err)
	}
	if len(definition.ConfigMagic) != 8 || len(definition.FinishMagic) != 8 || len(definition.ErrorMagic) != 8 ||
		definition.MaximumBytes == 0 || definition.MaximumStringBytes == 0 ||
		definition.MaximumStringBytes > definition.MaximumBytes || definition.MaximumNodes == 0 ||
		definition.MaximumLinks == 0 || definition.MaximumListeners == 0 || definition.MaximumConnections == 0 ||
		definition.MaximumTransitions == 0 ||
		definition.MaximumNetworkBytes != 1<<30 || definition.MaximumStallNanos != 9223372036854775806 {
		return networkWireSchema{}, fmt.Errorf("network wire schema has invalid framing or limits")
	}
	return definition, nil
}
