package gomadv3sim

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/netip"
	"path"
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	MaximumNodes                 uint64 = 64
	MaximumDirectionalLinks      uint64 = 4096
	MaximumVolumes               uint64 = 64
	MaximumScenarioActions       uint64 = 4096
	MaximumScenarioDecisions     uint64 = 4096
	MaximumHistoryOperations     uint64 = 1 << 16
	MaximumObservations          uint64 = 1 << 16
	MaximumOracleResults         uint64 = 4096
	MaximumScenarioEvidenceBytes uint64 = 64 << 20
	MaximumBootConfigBytes       uint64 = 1 << 20
	MaximumVolumeCapacityBytes   uint64 = 1 << 30
	MaximumObservationBytes      uint64 = 64 << 20
	MaximumNetworkListeners      uint64 = 4096
	MaximumNetworkConnections    uint64 = 65536
	MaximumNetworkDeliveries     uint64 = 65536
	MaximumNetworkBytes          uint64 = 1 << 30
	MaximumNetworkTransitions    uint64 = 1 << 20
	MaximumVolumeOperations      uint64 = 1 << 20
	MaximumVolumeTransitions     uint64 = 1 << 20
	MaximumCrashStates           uint64 = 1 << 16
	MaximumCrashDepth            uint64 = 256
	MaximumCrashBytes            uint64 = 128 << 20
	MaximumCrashWallNanos        uint64 = 60_000_000_000
	MaximumSpecJSONBytes                = 128 << 20
	MaximumMountPathBytes               = 4096
)

var ErrCapacity = errors.New("simulation capacity exhausted")

var identifierPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)

type CapacityError struct {
	Resource string
	Required uint64
	Maximum  uint64
}

func (err *CapacityError) Error() string {
	return fmt.Sprintf("%v: resource=%s required=%d maximum=%d", ErrCapacity, err.Resource, err.Required, err.Maximum)
}

func (err *CapacityError) Unwrap() error {
	return ErrCapacity
}

type Limits struct {
	Nodes                 uint64 `json:"nodes"`
	DirectionalLinks      uint64 `json:"directional_links"`
	Volumes               uint64 `json:"volumes"`
	ScenarioActions       uint64 `json:"scenario_actions"`
	ScenarioDecisions     uint64 `json:"scenario_decisions"`
	FaultActions          uint64 `json:"fault_actions"`
	HistoryOperations     uint64 `json:"history_operations"`
	Observations          uint64 `json:"observations"`
	OracleResults         uint64 `json:"oracle_results"`
	ScenarioEvidenceBytes uint64 `json:"scenario_evidence_bytes"`
	BootConfigBytes       uint64 `json:"boot_config_bytes"`
	VolumeCapacityBytes   uint64 `json:"volume_capacity_bytes"`
	ObservationBytes      uint64 `json:"observation_bytes"`
	NetworkListeners      uint64 `json:"network_listeners"`
	NetworkConnections    uint64 `json:"network_connections"`
	NetworkDeliveries     uint64 `json:"network_deliveries"`
	NetworkBytes          uint64 `json:"network_bytes"`
	NetworkTransitions    uint64 `json:"network_transitions"`
	VolumeOperations      uint64 `json:"volume_operations"`
	VolumeTransitions     uint64 `json:"volume_transitions"`
	CrashStates           uint64 `json:"crash_states"`
	CrashDepth            uint64 `json:"crash_depth"`
	CrashBytes            uint64 `json:"crash_bytes"`
	CrashWallNanos        uint64 `json:"crash_wall_nanos"`
}

type Spec struct {
	Schema   string       `json:"schema"`
	Backend  Backend      `json:"backend"`
	Fidelity Fidelity     `json:"fidelity"`
	Seed     uint64       `json:"seed"`
	Limits   Limits       `json:"limits"`
	Nodes    []NodeSpec   `json:"nodes"`
	Links    []LinkSpec   `json:"links,omitempty"`
	Volumes  []VolumeSpec `json:"volumes,omitempty"`
	Faults   *FaultPlan   `json:"faults,omitempty"`
	Replay   *ReplayPlan  `json:"replay,omitempty"`
}

type NodeSpec struct {
	ID      NodeID        `json:"id"`
	Boot    BootID        `json:"boot"`
	Address string        `json:"address"`
	Config  []byte        `json:"config,omitempty"`
	Volumes []VolumeMount `json:"volumes,omitempty"`
}

type LinkSpec struct {
	From       NodeID `json:"from"`
	To         NodeID `json:"to"`
	Enabled    bool   `json:"enabled"`
	DelayNanos uint64 `json:"delay_nanos"`
}

type VolumeSpec struct {
	ID            VolumeID `json:"id"`
	CapacityBytes uint64   `json:"capacity_bytes"`
}

type VolumeMount struct {
	Volume VolumeID `json:"volume"`
	Path   string   `json:"path"`
}

func DefaultLimits() Limits {
	return Limits{
		Nodes:                 MaximumNodes,
		DirectionalLinks:      MaximumDirectionalLinks,
		Volumes:               MaximumVolumes,
		ScenarioActions:       MaximumScenarioActions,
		ScenarioDecisions:     MaximumScenarioDecisions,
		FaultActions:          MaximumFaultActions,
		HistoryOperations:     MaximumHistoryOperations,
		Observations:          MaximumObservations,
		OracleResults:         MaximumOracleResults,
		ScenarioEvidenceBytes: MaximumScenarioEvidenceBytes,
		BootConfigBytes:       MaximumBootConfigBytes,
		VolumeCapacityBytes:   MaximumVolumeCapacityBytes,
		ObservationBytes:      MaximumObservationBytes,
		NetworkListeners:      MaximumNetworkListeners,
		NetworkConnections:    MaximumNetworkConnections,
		NetworkDeliveries:     MaximumNetworkDeliveries,
		NetworkBytes:          MaximumNetworkBytes,
		NetworkTransitions:    MaximumNetworkTransitions,
		VolumeOperations:      MaximumVolumeOperations,
		VolumeTransitions:     MaximumVolumeTransitions,
		CrashStates:           MaximumCrashStates,
		CrashDepth:            MaximumCrashDepth,
		CrashBytes:            MaximumCrashBytes,
		CrashWallNanos:        MaximumCrashWallNanos,
	}
}

func DecodeSpec(data []byte) (Spec, error) {
	if len(data) == 0 || len(data) > MaximumSpecJSONBytes {
		return Spec{}, fmt.Errorf("simulation spec must be between 1 and %d bytes", MaximumSpecJSONBytes)
	}
	if !utf8.Valid(data) {
		return Spec{}, errors.New("simulation spec is not valid UTF-8")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var spec Spec
	if err := decoder.Decode(&spec); err != nil {
		return Spec{}, fmt.Errorf("decode simulation spec: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return Spec{}, errors.New("simulation spec contains trailing JSON")
		}
		return Spec{}, fmt.Errorf("decode trailing simulation spec data: %w", err)
	}
	canonical, err := json.Marshal(spec)
	if err != nil {
		return Spec{}, fmt.Errorf("canonicalize simulation spec: %w", err)
	}
	if !bytes.Equal(data, canonical) {
		return Spec{}, errors.New("simulation spec is not canonical JSON")
	}
	if err := ValidateSpec(spec); err != nil {
		return Spec{}, err
	}
	return spec, nil
}

func ValidateSpec(spec Spec) error {
	if spec.Schema != SpecSchema {
		return fmt.Errorf("simulation spec schema = %q, want %q", spec.Schema, SpecSchema)
	}
	switch spec.Backend {
	case BackendInProcess, BackendProcess:
	default:
		return fmt.Errorf("invalid simulation backend %q", spec.Backend)
	}
	switch spec.Fidelity {
	case FidelitySimulationModel:
	case FidelityHardIsolation:
		if spec.Backend != BackendProcess {
			return errors.New("hard isolation fidelity requires the process backend")
		}
	default:
		return fmt.Errorf("invalid simulation fidelity %q", spec.Fidelity)
	}
	if err := validateLimits(spec.Limits); err != nil {
		return err
	}
	if err := checkCapacity("nodes", uint64(len(spec.Nodes)), spec.Limits.Nodes); err != nil {
		return err
	}
	if err := checkCapacity("directional_links", uint64(len(spec.Links)), spec.Limits.DirectionalLinks); err != nil {
		return err
	}
	if err := checkCapacity("volumes", uint64(len(spec.Volumes)), spec.Limits.Volumes); err != nil {
		return err
	}
	if len(spec.Nodes) == 0 {
		return errors.New("simulation spec must contain at least one node")
	}
	if spec.Replay != nil {
		if err := validateReplayPlan(*spec.Replay, spec); err != nil {
			return err
		}
	}
	if spec.Faults != nil {
		if err := validateFaultPlan(*spec.Faults); err != nil {
			return err
		}
		if err := checkCapacity("fault_actions", uint64(len(spec.Faults.Actions)), spec.Limits.FaultActions); err != nil {
			return err
		}
	}

	volumes, err := validateVolumes(spec.Volumes, spec.Limits.VolumeCapacityBytes)
	if err != nil {
		return err
	}
	nodes, err := validateNodes(spec.Nodes, volumes, spec.Limits.BootConfigBytes)
	if err != nil {
		return err
	}
	if err := validateLinks(spec.Links, nodes); err != nil {
		return err
	}
	return validateFaultReferences(spec.Faults, nodes)
}

func validateFaultReferences(plan *FaultPlan, nodes map[NodeID]struct{}) error {
	if plan == nil {
		return nil
	}
	prior := make(map[FaultID]FaultAction, len(plan.Actions))
	for _, action := range plan.Actions {
		checkNode := func(node NodeID) error {
			if _, ok := nodes[node]; !ok {
				return fmt.Errorf("fault %q refers to unknown node %q", action.ID, node)
			}
			return nil
		}
		if action.Node != "" {
			if err := checkNode(action.Node); err != nil {
				return err
			}
		}
		for _, node := range action.Candidates {
			if err := checkNode(node); err != nil {
				return err
			}
		}
		for _, node := range append(append(append([]NodeID(nil), action.Left...), action.Right...), action.From, action.To) {
			if node != "" {
				if err := checkNode(node); err != nil {
					return err
				}
			}
		}
		if action.Match.Node != "" {
			if err := checkNode(action.Match.Node); err != nil {
				return err
			}
		}
		if action.TargetFrom != "" {
			referenced, ok := prior[action.TargetFrom]
			if !ok || referenced.Kind != FaultGracefulStop && referenced.Kind != FaultHarshCrash && referenced.Kind != FaultRestart {
				return fmt.Errorf("fault %q has invalid prior target reference %q", action.ID, action.TargetFrom)
			}
		}
		prior[action.ID] = action
	}
	return nil
}

func validateLimits(limits Limits) error {
	configured := []struct {
		name    string
		value   uint64
		maximum uint64
	}{
		{name: "nodes", value: limits.Nodes, maximum: MaximumNodes},
		{name: "directional_links", value: limits.DirectionalLinks, maximum: MaximumDirectionalLinks},
		{name: "volumes", value: limits.Volumes, maximum: MaximumVolumes},
		{name: "scenario_actions", value: limits.ScenarioActions, maximum: MaximumScenarioActions},
		{name: "scenario_decisions", value: limits.ScenarioDecisions, maximum: MaximumScenarioDecisions},
		{name: "fault_actions", value: limits.FaultActions, maximum: MaximumFaultActions},
		{name: "history_operations", value: limits.HistoryOperations, maximum: MaximumHistoryOperations},
		{name: "observations", value: limits.Observations, maximum: MaximumObservations},
		{name: "oracle_results", value: limits.OracleResults, maximum: MaximumOracleResults},
		{name: "scenario_evidence_bytes", value: limits.ScenarioEvidenceBytes, maximum: MaximumScenarioEvidenceBytes},
		{name: "boot_config_bytes", value: limits.BootConfigBytes, maximum: MaximumBootConfigBytes},
		{name: "volume_capacity_bytes", value: limits.VolumeCapacityBytes, maximum: MaximumVolumeCapacityBytes},
		{name: "observation_bytes", value: limits.ObservationBytes, maximum: MaximumObservationBytes},
		{name: "network_listeners", value: limits.NetworkListeners, maximum: MaximumNetworkListeners},
		{name: "network_connections", value: limits.NetworkConnections, maximum: MaximumNetworkConnections},
		{name: "network_deliveries", value: limits.NetworkDeliveries, maximum: MaximumNetworkDeliveries},
		{name: "network_bytes", value: limits.NetworkBytes, maximum: MaximumNetworkBytes},
		{name: "network_transitions", value: limits.NetworkTransitions, maximum: MaximumNetworkTransitions},
		{name: "volume_operations", value: limits.VolumeOperations, maximum: MaximumVolumeOperations},
		{name: "volume_transitions", value: limits.VolumeTransitions, maximum: MaximumVolumeTransitions},
		{name: "crash_states", value: limits.CrashStates, maximum: MaximumCrashStates},
		{name: "crash_depth", value: limits.CrashDepth, maximum: MaximumCrashDepth},
		{name: "crash_bytes", value: limits.CrashBytes, maximum: MaximumCrashBytes},
		{name: "crash_wall_nanos", value: limits.CrashWallNanos, maximum: MaximumCrashWallNanos},
	}
	for _, limit := range configured {
		if limit.value == 0 || limit.value > limit.maximum {
			return fmt.Errorf("simulation %s limit = %d, want between 1 and %d", limit.name, limit.value, limit.maximum)
		}
	}
	return nil
}

func validateVolumes(specs []VolumeSpec, maximumBytes uint64) (map[VolumeID]struct{}, error) {
	volumes := make(map[VolumeID]struct{}, len(specs))
	var capacityBytes uint64
	var previous VolumeID
	for _, spec := range specs {
		if err := validateID("volume ID", string(spec.ID)); err != nil {
			return nil, err
		}
		if spec.ID <= previous {
			return nil, fmt.Errorf("volume IDs must be strictly sorted: %q after %q", spec.ID, previous)
		}
		if spec.CapacityBytes == 0 {
			return nil, fmt.Errorf("volume %q capacity must be nonzero", spec.ID)
		}
		capacityBytes = saturatingAdd(capacityBytes, spec.CapacityBytes)
		if err := checkCapacity("volume_capacity_bytes", capacityBytes, maximumBytes); err != nil {
			return nil, err
		}
		volumes[spec.ID] = struct{}{}
		previous = spec.ID
	}
	return volumes, nil
}

func validateNodes(specs []NodeSpec, volumes map[VolumeID]struct{}, maximumConfigBytes uint64) (map[NodeID]struct{}, error) {
	nodes := make(map[NodeID]struct{}, len(specs))
	addresses := make(map[netip.Addr]NodeID, len(specs))
	volumeOwners := make(map[VolumeID]NodeID, len(volumes))
	var configBytes uint64
	var previous NodeID
	for _, spec := range specs {
		if err := validateID("node ID", string(spec.ID)); err != nil {
			return nil, err
		}
		if spec.ID <= previous {
			return nil, fmt.Errorf("node IDs must be strictly sorted: %q after %q", spec.ID, previous)
		}
		if err := validateID("boot ID", string(spec.Boot)); err != nil {
			return nil, fmt.Errorf("node %q: %w", spec.ID, err)
		}
		address, err := netip.ParseAddr(spec.Address)
		if len(spec.Address) > 128 || err != nil || address.Zone() != "" || address.IsUnspecified() || address.Unmap().String() != spec.Address {
			return nil, fmt.Errorf("node %q has invalid address %q", spec.ID, spec.Address)
		}
		if owner, ok := addresses[address]; ok {
			return nil, fmt.Errorf("nodes %q and %q share address %q", owner, spec.ID, address)
		}
		if err := validateMounts(spec.ID, spec.Volumes, volumes, volumeOwners); err != nil {
			return nil, err
		}
		configBytes = saturatingAdd(configBytes, uint64(len(spec.Config)))
		if err := checkCapacity("boot_config_bytes", configBytes, maximumConfigBytes); err != nil {
			return nil, err
		}
		nodes[spec.ID] = struct{}{}
		addresses[address] = spec.ID
		previous = spec.ID
	}
	if len(volumeOwners) != len(volumes) {
		return nil, errors.New("every simulation volume must be mounted by exactly one node")
	}
	return nodes, nil
}

func validateMounts(nodeID NodeID, mounts []VolumeMount, volumes map[VolumeID]struct{}, owners map[VolumeID]NodeID) error {
	seenVolumes := make(map[VolumeID]struct{}, len(mounts))
	previousPath := ""
	for _, mount := range mounts {
		if _, ok := volumes[mount.Volume]; !ok {
			return fmt.Errorf("node %q mounts unknown volume %q", nodeID, mount.Volume)
		}
		if len(mount.Path) == 0 || len(mount.Path) > MaximumMountPathBytes || !utf8.ValidString(mount.Path) || strings.IndexByte(mount.Path, 0) >= 0 || mount.Path == "/" || !path.IsAbs(mount.Path) || path.Clean(mount.Path) != mount.Path || strings.Contains(mount.Path, `\`) {
			return fmt.Errorf("node %q has invalid volume path %q", nodeID, mount.Path)
		}
		if mount.Path <= previousPath {
			return fmt.Errorf("node %q volume paths must be strictly sorted: %q after %q", nodeID, mount.Path, previousPath)
		}
		if _, ok := seenVolumes[mount.Volume]; ok {
			return fmt.Errorf("node %q mounts volume %q more than once", nodeID, mount.Volume)
		}
		if owner := owners[mount.Volume]; owner != "" {
			return fmt.Errorf("nodes %q and %q both mount per-node volume %q", owner, nodeID, mount.Volume)
		}
		seenVolumes[mount.Volume] = struct{}{}
		owners[mount.Volume] = nodeID
		previousPath = mount.Path
	}
	return nil
}

func validateLinks(specs []LinkSpec, nodes map[NodeID]struct{}) error {
	previous := ""
	for _, spec := range specs {
		if _, ok := nodes[spec.From]; !ok {
			return fmt.Errorf("link refers to unknown source node %q", spec.From)
		}
		if _, ok := nodes[spec.To]; !ok {
			return fmt.Errorf("link refers to unknown destination node %q", spec.To)
		}
		if spec.From == spec.To {
			return fmt.Errorf("link from node %q to itself is invalid", spec.From)
		}
		key := string(spec.From) + "\x00" + string(spec.To)
		if key <= previous {
			return errors.New("directional links must be strictly sorted")
		}
		previous = key
	}
	return nil
}

func validateID(name string, value string) error {
	if len(value) == 0 || len(value) > 128 || !identifierPattern.MatchString(value) {
		return fmt.Errorf("invalid %s %q", name, value)
	}
	return nil
}

func checkCapacity(resource string, required uint64, maximum uint64) error {
	if required > maximum {
		return &CapacityError{Resource: resource, Required: required, Maximum: maximum}
	}
	return nil
}

func saturatingAdd(left uint64, right uint64) uint64 {
	if right > math.MaxUint64-left {
		return math.MaxUint64
	}
	return left + right
}
