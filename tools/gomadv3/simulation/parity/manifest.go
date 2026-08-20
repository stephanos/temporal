package parity

import (
	_ "embed"
	"errors"
	"fmt"
	"path"
	"regexp"
	"slices"
	"strings"

	"go.temporal.io/server/tools/gomadv3/evidence"
)

const ManifestSchema = "gomadv3.simulation-parity/v1"
const HarnessSpecSchema = "gomadv3.simulation-spec/v5"
const MaximumManifestBytes = 1 << 20

const (
	BackendInProcess Backend = "in_process"
	BackendProcess   Backend = "process"
)

const (
	FidelitySimulationModel Fidelity = "simulation_model"
	FidelityHardIsolation   Fidelity = "hard_isolation"
)

const (
	DispositionPreserved Disposition = "preserved"
	DispositionReplaced  Disposition = "replaced"
)

const (
	StatusImplemented Status = "implemented"
	StatusPlanned     Status = "planned"
	StatusPrototype   Status = "prototype"
)

const (
	StageSIM1 Stage = "SIM-1"
	StageSIM2 Stage = "SIM-2"
	StageSIM3 Stage = "SIM-3"
	StageSIM4 Stage = "SIM-4"
)

const (
	MaximumNodes                 uint64 = 64
	MaximumDirectionalLinks      uint64 = 4096
	MaximumVolumes               uint64 = 64
	MaximumScenarioActions       uint64 = 4096
	MaximumScenarioDecisions     uint64 = 4096
	MaximumFaultActions          uint64 = 4096
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
)

const (
	maximumCases          = 64
	maximumPrototypes     = 16
	maximumSourcesPerCase = 16
	maximumTestsPerSource = 64
	maximumRequirements   = 4
	maximumTextBytes      = 4096
	maximumTestNameBytes  = 256
)

var identifierPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)
var testPattern = regexp.MustCompile(`^Test[A-Za-z0-9_]+$`)

//go:embed manifest.json
var currentManifestBytes []byte

type Backend string
type Fidelity string
type Disposition string
type Status string
type Stage string

type Manifest struct {
	Schema        string      `json:"schema"`
	HarnessSchema string      `json:"harness_schema"`
	Limits        Limits      `json:"limits"`
	Cases         []Case      `json:"cases"`
	Prototypes    []Prototype `json:"prototypes"`
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

type Case struct {
	ID           string            `json:"id"`
	Stage        Stage             `json:"stage"`
	Contract     string            `json:"contract"`
	Disposition  Disposition       `json:"disposition"`
	Replacement  string            `json:"replacement,omitempty"`
	Status       Status            `json:"status"`
	Sources      []SourceReference `json:"sources"`
	Requirements []Requirement     `json:"requirements"`
}

type SourceReference struct {
	Path  string   `json:"path"`
	Tests []string `json:"tests"`
}

type Requirement struct {
	Fidelity Fidelity  `json:"fidelity"`
	Backends []Backend `json:"backends"`
}

type Prototype struct {
	ID       string   `json:"id"`
	CaseID   string   `json:"case_id"`
	Package  string   `json:"package"`
	Test     string   `json:"test"`
	Backend  Backend  `json:"backend"`
	Fidelity Fidelity `json:"fidelity"`
	Status   Status   `json:"status"`
}

func CurrentBytes() []byte {
	return append([]byte(nil), currentManifestBytes...)
}

func Current() (Manifest, error) {
	return Decode(currentManifestBytes)
}

func Decode(data []byte) (Manifest, error) {
	if len(data) == 0 || len(data) > MaximumManifestBytes {
		return Manifest{}, fmt.Errorf("simulation parity manifest must be between 1 and %d bytes", MaximumManifestBytes)
	}
	var manifest Manifest
	if err := evidence.DecodeCanonicalJSON(data, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("decode simulation parity manifest: %w", err)
	}
	if err := manifest.Validate(); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func (manifest Manifest) Validate() error {
	if manifest.Schema != ManifestSchema {
		return fmt.Errorf("simulation parity schema = %q, want %q", manifest.Schema, ManifestSchema)
	}
	if manifest.HarnessSchema != HarnessSpecSchema {
		return fmt.Errorf("simulation harness schema = %q, want %q", manifest.HarnessSchema, HarnessSpecSchema)
	}
	wantLimits := Limits{
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
	if manifest.Limits != wantLimits {
		return fmt.Errorf("simulation parity limits = %+v, want %+v", manifest.Limits, wantLimits)
	}
	if len(manifest.Cases) == 0 || len(manifest.Cases) > maximumCases {
		return fmt.Errorf("simulation parity case count = %d, want between 1 and %d", len(manifest.Cases), maximumCases)
	}
	cases := make(map[string]Case, len(manifest.Cases))
	previousID := ""
	for index, parityCase := range manifest.Cases {
		if err := validateCase(parityCase); err != nil {
			return fmt.Errorf("simulation parity case %d: %w", index, err)
		}
		if parityCase.ID <= previousID {
			return fmt.Errorf("simulation parity case IDs must be strictly sorted: %q after %q", parityCase.ID, previousID)
		}
		cases[parityCase.ID] = parityCase
		previousID = parityCase.ID
	}
	if len(manifest.Prototypes) == 0 || len(manifest.Prototypes) > maximumPrototypes {
		return fmt.Errorf("simulation prototype count = %d, want between 1 and %d", len(manifest.Prototypes), maximumPrototypes)
	}
	previousID = ""
	for index, prototype := range manifest.Prototypes {
		if err := validatePrototype(prototype, cases); err != nil {
			return fmt.Errorf("simulation prototype %d: %w", index, err)
		}
		if prototype.ID <= previousID {
			return fmt.Errorf("simulation prototype IDs must be strictly sorted: %q after %q", prototype.ID, previousID)
		}
		previousID = prototype.ID
	}
	encoded, err := evidence.CanonicalJSON(manifest)
	if err != nil {
		return fmt.Errorf("canonicalize simulation parity manifest: %w", err)
	}
	if len(encoded) > MaximumManifestBytes {
		return fmt.Errorf("simulation parity manifest requires %d bytes, maximum is %d", len(encoded), MaximumManifestBytes)
	}
	return nil
}

func validateCase(parityCase Case) error {
	if err := validateIdentifier("case ID", parityCase.ID); err != nil {
		return err
	}
	switch parityCase.Stage {
	case StageSIM1, StageSIM2, StageSIM3, StageSIM4:
	default:
		return fmt.Errorf("case %q has invalid stage %q", parityCase.ID, parityCase.Stage)
	}
	if err := validateText("contract", parityCase.Contract, maximumTextBytes); err != nil {
		return fmt.Errorf("case %q contract must be between 1 and %d bytes", parityCase.ID, maximumTextBytes)
	}
	switch parityCase.Disposition {
	case DispositionPreserved:
		if parityCase.Replacement != "" {
			return fmt.Errorf("preserved case %q has replacement text", parityCase.ID)
		}
	case DispositionReplaced:
		if err := validateText("replacement", parityCase.Replacement, maximumTextBytes); err != nil {
			return fmt.Errorf("replaced case %q replacement must be between 1 and %d bytes", parityCase.ID, maximumTextBytes)
		}
	default:
		return fmt.Errorf("case %q has invalid disposition %q", parityCase.ID, parityCase.Disposition)
	}
	if parityCase.Status != StatusImplemented && parityCase.Status != StatusPlanned {
		return fmt.Errorf("case %q has invalid status %q", parityCase.ID, parityCase.Status)
	}
	if len(parityCase.Sources) == 0 || len(parityCase.Sources) > maximumSourcesPerCase {
		return fmt.Errorf("case %q source count = %d, want between 1 and %d", parityCase.ID, len(parityCase.Sources), maximumSourcesPerCase)
	}
	previousPath := ""
	for _, source := range parityCase.Sources {
		if err := validateSource(source); err != nil {
			return fmt.Errorf("case %q: %w", parityCase.ID, err)
		}
		if source.Path <= previousPath {
			return fmt.Errorf("case %q source paths must be strictly sorted: %q after %q", parityCase.ID, source.Path, previousPath)
		}
		previousPath = source.Path
	}
	if len(parityCase.Requirements) == 0 || len(parityCase.Requirements) > maximumRequirements {
		return fmt.Errorf("case %q requirement count = %d, want between 1 and %d", parityCase.ID, len(parityCase.Requirements), maximumRequirements)
	}
	previousFidelity := Fidelity("")
	for _, requirement := range parityCase.Requirements {
		if err := validateRequirement(requirement); err != nil {
			return fmt.Errorf("case %q: %w", parityCase.ID, err)
		}
		if requirement.Fidelity <= previousFidelity {
			return fmt.Errorf("case %q requirements must be strictly sorted by fidelity", parityCase.ID)
		}
		previousFidelity = requirement.Fidelity
	}
	return nil
}

func validateSource(source SourceReference) error {
	if len(source.Path) == 0 || len(source.Path) > maximumTextBytes || strings.IndexByte(source.Path, 0) >= 0 || !strings.HasPrefix(source.Path, "tools/gomadv2/") || path.Clean(source.Path) != source.Path || path.Ext(source.Path) != ".go" || strings.Contains(source.Path, `\`) {
		return fmt.Errorf("invalid v2 source path %q", source.Path)
	}
	if len(source.Tests) == 0 || len(source.Tests) > maximumTestsPerSource {
		return fmt.Errorf("source %q test count = %d, want between 1 and %d", source.Path, len(source.Tests), maximumTestsPerSource)
	}
	if !slices.IsSorted(source.Tests) {
		return fmt.Errorf("source %q test names are not sorted", source.Path)
	}
	for index, testName := range source.Tests {
		if len(testName) > maximumTestNameBytes || !testPattern.MatchString(testName) {
			return fmt.Errorf("source %q has invalid test name %q", source.Path, testName)
		}
		if index > 0 && testName == source.Tests[index-1] {
			return fmt.Errorf("source %q has duplicate test name %q", source.Path, testName)
		}
	}
	return nil
}

func validateRequirement(requirement Requirement) error {
	switch requirement.Fidelity {
	case FidelitySimulationModel, FidelityHardIsolation:
	default:
		return fmt.Errorf("invalid fidelity %q", requirement.Fidelity)
	}
	if len(requirement.Backends) == 0 || len(requirement.Backends) > 2 {
		return fmt.Errorf("fidelity %q backend count = %d, want between 1 and 2", requirement.Fidelity, len(requirement.Backends))
	}
	if !slices.IsSorted(requirement.Backends) {
		return fmt.Errorf("fidelity %q backends are not sorted", requirement.Fidelity)
	}
	for index, backend := range requirement.Backends {
		switch backend {
		case BackendInProcess, BackendProcess:
		default:
			return fmt.Errorf("invalid backend %q", backend)
		}
		if index > 0 && backend == requirement.Backends[index-1] {
			return fmt.Errorf("fidelity %q has duplicate backend %q", requirement.Fidelity, backend)
		}
		if requirement.Fidelity == FidelityHardIsolation && backend != BackendProcess {
			return errors.New("hard isolation requires the process backend")
		}
	}
	return nil
}

func validatePrototype(prototype Prototype, cases map[string]Case) error {
	if err := validateIdentifier("prototype ID", prototype.ID); err != nil {
		return err
	}
	parityCase, ok := cases[prototype.CaseID]
	if !ok {
		return fmt.Errorf("prototype %q refers to unknown case %q", prototype.ID, prototype.CaseID)
	}
	if prototype.Package != "./tools/gomadv3sim" {
		return fmt.Errorf("prototype %q package = %q, want %q", prototype.ID, prototype.Package, "./tools/gomadv3sim")
	}
	if len(prototype.Test) > maximumTestNameBytes || !testPattern.MatchString(prototype.Test) {
		return fmt.Errorf("prototype %q has invalid test %q", prototype.ID, prototype.Test)
	}
	if prototype.Status != StatusPrototype {
		return fmt.Errorf("prototype %q status = %q, want %q", prototype.ID, prototype.Status, StatusPrototype)
	}
	if err := validateRequirement(Requirement{Fidelity: prototype.Fidelity, Backends: []Backend{prototype.Backend}}); err != nil {
		return err
	}
	for _, requirement := range parityCase.Requirements {
		if requirement.Fidelity == prototype.Fidelity && slices.Contains(requirement.Backends, prototype.Backend) {
			return nil
		}
	}
	return fmt.Errorf("prototype %q backend %q and fidelity %q are not required by case %q", prototype.ID, prototype.Backend, prototype.Fidelity, prototype.CaseID)
}

func validateIdentifier(name string, value string) error {
	if len(value) == 0 || len(value) > 128 || !identifierPattern.MatchString(value) {
		return fmt.Errorf("invalid %s %q", name, value)
	}
	return nil
}

func validateText(name string, value string, maximum int) error {
	if len(value) == 0 || len(value) > maximum || strings.IndexByte(value, 0) >= 0 {
		return fmt.Errorf("%s must be between 1 and %d bytes and contain no NUL", name, maximum)
	}
	return nil
}
