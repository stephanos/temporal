package deployment

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"

	environment "go.temporal.io/server/tools/umpire3/execution"
	protocolcatalog "go.temporal.io/server/tools/umpire3/protocol/catalog"
	protocolexperiment "go.temporal.io/server/tools/umpire3/protocol/experiment"
)

type Kind string

const (
	KindLocal    Kind = "local-in-process"
	KindCI       Kind = "ci-test-cluster"
	KindRemote   Kind = "remote-deployment"
	KindBlackBox Kind = "grpc-only-black-box"
	KindCanary   Kind = "production-canary"
)

type Spec struct {
	Kind                Kind                           `json:"kind"`
	Endpoint            string                         `json:"endpoint,omitempty"`
	AuthToken           string                         `json:"-"`
	BuildID             string                         `json:"buildID"`
	Namespace           string                         `json:"namespace"`
	TaskQueue           string                         `json:"taskQueue"`
	WorkerCommand       []string                       `json:"-"`
	HardExecutionBudget bool                           `json:"hardExecutionBudget"`
	Capabilities        []protocolcatalog.CapabilityID `json:"capabilities,omitempty"`
}

type Attestation struct {
	BuildID             string `json:"buildID"`
	ConfigurationDigest string `json:"configurationDigest"`
	EndpointIdentity    string `json:"endpointIdentity,omitempty"`
}

type Profile struct {
	Kind          Kind                            `json:"kind"`
	Environment   environment.EnvironmentIdentity `json:"environment"`
	Capabilities  []protocolcatalog.CapabilityID  `json:"capabilities"`
	Attestation   Attestation                     `json:"attestation"`
	Endpoint      string                          `json:"endpoint,omitempty"`
	Namespace     string                          `json:"namespace"`
	TaskQueue     string                          `json:"taskQueue"`
	workerCommand []string
}

func Local(buildID, namespace, taskQueue string) Spec {
	return Spec{Kind: KindLocal, BuildID: buildID, Namespace: namespace, TaskQueue: taskQueue}
}

func CI(buildID, namespace, taskQueue string) Spec {
	return Spec{Kind: KindCI, BuildID: buildID, Namespace: namespace, TaskQueue: taskQueue}
}

func Remote(endpoint, authToken, buildID, namespace, taskQueue string) Spec {
	return Spec{
		Kind: KindRemote, Endpoint: endpoint, AuthToken: authToken, BuildID: buildID,
		Namespace: namespace, TaskQueue: taskQueue,
	}
}

func BlackBox(endpoint, authToken, buildID, namespace, taskQueue string) Spec {
	config := Remote(endpoint, authToken, buildID, namespace, taskQueue)
	config.Kind = KindBlackBox
	return config
}

func Canary(endpoint, authToken, buildID, namespace, taskQueue string, workerCommand []string) Spec {
	config := Remote(endpoint, authToken, buildID, namespace, taskQueue)
	config.Kind = KindCanary
	config.WorkerCommand = append([]string(nil), workerCommand...)
	config.HardExecutionBudget = true
	return config
}

func Define(config Spec) (Profile, error) {
	if config.BuildID == "" || config.Namespace == "" || config.TaskQueue == "" {
		return Profile{}, errors.New("build, namespace, and task queue identities are required")
	}
	if config.HardExecutionBudget && len(config.WorkerCommand) == 0 {
		return Profile{}, errors.New("hard execution budget requires a killable worker command")
	}
	if err := validateEndpoint(config); err != nil {
		return Profile{}, err
	}

	identity := environment.EnvironmentIdentity{
		Name: config.Kind.String(), BuildID: config.BuildID,
		ConfigurationIdentity: configurationDigest(config),
		IsolationIdentity:     config.Namespace + "/" + config.TaskQueue,
		RetentionClass:        "semantic-redacted", HardExecutionBudget: config.HardExecutionBudget,
	}
	capabilities := catalogCapabilities()
	switch config.Kind {
	case KindLocal:
		identity.EvidenceProfile = environment.EvidenceProfileInProcessHooks
		identity.DrivingAuthority = "local-test-authority"
		identity.ObservationAuthority = "local-server-hooks"
		identity.FaultAuthority = "isolated-local-faults"
	case KindCI:
		identity.EvidenceProfile = environment.EvidenceProfilePublicGRPCHistory
		identity.DrivingAuthority = "ci-test-cluster"
		identity.ObservationAuthority = "ci-public-history"
		identity.FaultAuthority = "isolated-ci-faults"
	case KindRemote:
		identity.EvidenceProfile = environment.EvidenceProfilePublicGRPCHistory
		identity.DrivingAuthority = "remote-api"
		identity.ObservationAuthority = "remote-public-history"
		identity.FaultAuthority = "remote-approved-faults"
		capabilities = withoutCapabilities(capabilities,
			protocolcatalog.CapabilityIDFaultProcess, protocolcatalog.CapabilityIDFaultPersistence)
	case KindBlackBox:
		identity.EvidenceProfile = environment.EvidenceProfilePublicGRPC
		identity.DrivingAuthority = "public-grpc"
		identity.ObservationAuthority = "public-grpc"
		identity.FaultAuthority = "none"
		capabilities = withoutCapabilities(capabilities,
			protocolcatalog.CapabilityIDFaultProcess, protocolcatalog.CapabilityIDFaultNetwork,
			protocolcatalog.CapabilityIDFaultClock, protocolcatalog.CapabilityIDFaultPersistence,
			protocolcatalog.CapabilityIDFailoverControl)
	case KindCanary:
		identity.EvidenceProfile = environment.EvidenceProfilePublicGRPCHistory
		identity.DrivingAuthority = "approved-production-worker"
		identity.ObservationAuthority = "production-public-history"
		identity.FaultAuthority = "approved-production-fault-controller"
		capabilities = withoutCapabilities(capabilities,
			protocolcatalog.CapabilityIDFaultProcess, protocolcatalog.CapabilityIDFaultNetwork,
			protocolcatalog.CapabilityIDFaultClock, protocolcatalog.CapabilityIDFaultPersistence)
	default:
		return Profile{}, fmt.Errorf("unknown deployment profile %q", config.Kind)
	}
	if config.Capabilities != nil {
		capabilities = intersectCapabilities(capabilities, config.Capabilities)
	}
	identity.Capabilities = append([]protocolcatalog.CapabilityID(nil), capabilities...)
	if err := identity.Validate(); err != nil {
		return Profile{}, err
	}
	return Profile{
		Kind: config.Kind, Environment: identity, Capabilities: capabilities,
		Attestation: Attestation{
			BuildID: config.BuildID, ConfigurationDigest: identity.ConfigurationIdentity,
			EndpointIdentity: digest(config.Endpoint),
		},
		Endpoint: config.Endpoint, Namespace: config.Namespace, TaskQueue: config.TaskQueue,
		workerCommand: append([]string(nil), config.WorkerCommand...),
	}, nil
}

func (k Kind) String() string {
	return string(k)
}

func (d Profile) String() string {
	encoded, _ := json.Marshal(d)
	return string(encoded)
}

func (d Profile) Digest() (string, error) {
	encoded, err := json.Marshal(d)
	if err != nil {
		return "", fmt.Errorf("encode profile definition: %w", err)
	}
	return digest(string(encoded)), nil
}

func (d Profile) WorkerCommand() []string {
	return append([]string(nil), d.workerCommand...)
}

func validateEndpoint(config Spec) error {
	switch config.Kind {
	case KindLocal, KindCI:
		if config.Endpoint != "" || config.AuthToken != "" {
			return errors.New("local profiles do not accept remote endpoint credentials")
		}
		return nil
	case KindRemote, KindBlackBox, KindCanary:
		if config.Endpoint == "" || config.AuthToken == "" {
			return errors.New("remote endpoint and authentication are required")
		}
		parsed, err := url.Parse(config.Endpoint)
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" {
			return errors.New("remote endpoint must be an HTTPS origin without credentials or query parameters")
		}
		return nil
	default:
		return nil
	}
}

func configurationDigest(config Spec) string {
	capabilities := append([]protocolcatalog.CapabilityID(nil), config.Capabilities...)
	slices.Sort(capabilities)
	return digest(strings.Join([]string{
		string(config.Kind), config.Endpoint, config.BuildID, config.Namespace, config.TaskQueue,
		fmt.Sprint(config.HardExecutionBudget), strings.Join(config.WorkerCommand, "\x00"),
		strings.Join(capabilityStrings(capabilities), "\x00"),
	}, "\x00"))
}

func digest(value string) string {
	if value == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func catalogCapabilities() []protocolcatalog.CapabilityID {
	catalog, err := protocolcatalog.DefaultCatalog()
	if err != nil {
		return nil
	}
	result := make([]protocolcatalog.CapabilityID, len(catalog.Capabilities))
	for index, capability := range catalog.Capabilities {
		result[index] = capability.Identifier
	}
	slices.Sort(result)
	return result
}

func withoutCapabilities(values []protocolcatalog.CapabilityID, removed ...protocolcatalog.CapabilityID) []protocolcatalog.CapabilityID {
	blocked := make(map[protocolcatalog.CapabilityID]struct{}, len(removed))
	for _, capability := range removed {
		blocked[capability] = struct{}{}
	}
	result := make([]protocolcatalog.CapabilityID, 0, len(values))
	for _, value := range values {
		if _, remove := blocked[value]; !remove {
			result = append(result, value)
		}
	}
	return result
}

type boundFactory struct {
	profile      Profile
	underlying   environment.Factory
	capabilities []protocolcatalog.CapabilityID
}

func Bind(profile Profile, underlying environment.Factory) (environment.Factory, error) {
	if underlying == nil {
		return nil, errors.New("underlying environment factory is required")
	}
	if err := profile.Environment.Validate(); err != nil {
		return nil, fmt.Errorf("validate profile: %w", err)
	}
	if profile.Environment.HardExecutionBudget {
		return nil, errors.New("hard-budget profile must execute through its killable worker command")
	}
	capabilities := intersectCapabilities(profile.Capabilities, underlying.Capabilities())
	return &boundFactory{profile: profile, underlying: underlying, capabilities: capabilities}, nil
}

func (f *boundFactory) Capabilities() []protocolcatalog.CapabilityID {
	return append([]protocolcatalog.CapabilityID(nil), f.capabilities...)
}

func (f *boundFactory) Prepare(ctx context.Context, experiment protocolexperiment.Experiment) (environment.PreparedEnvironment, error) {
	if missing := missingCapabilities(experiment, f.capabilities); len(missing) != 0 {
		return environment.PreparedEnvironment{}, fmt.Errorf("unsupported capabilities: %v", missing)
	}
	prepared, err := f.underlying.Prepare(ctx, experiment)
	if prepared.Session == nil {
		return prepared, err
	}
	identity := f.profile.Environment
	identity.Capabilities = append([]protocolcatalog.CapabilityID(nil), f.capabilities...)
	if prepared.Identity.FaultAuthority == "none" || !hasFaultCapability(f.capabilities) {
		identity.FaultAuthority = "none"
	}
	prepared.Identity = identity
	return prepared, err
}

func intersectCapabilities(left, right []protocolcatalog.CapabilityID) []protocolcatalog.CapabilityID {
	rightSet := make(map[protocolcatalog.CapabilityID]struct{}, len(right))
	for _, capability := range right {
		rightSet[capability] = struct{}{}
	}
	var result []protocolcatalog.CapabilityID
	for _, capability := range left {
		if _, exists := rightSet[capability]; exists {
			result = append(result, capability)
		}
	}
	slices.Sort(result)
	return slices.Compact(result)
}

func missingCapabilities(experiment protocolexperiment.Experiment, available []protocolcatalog.CapabilityID) []string {
	have := make(map[protocolcatalog.CapabilityID]struct{}, len(available))
	for _, capability := range available {
		have[capability] = struct{}{}
	}
	var missing []string
	for _, action := range experiment.Actions {
		for _, capability := range action.RequiredCapabilities {
			if _, exists := have[protocolcatalog.CapabilityID(capability)]; !exists {
				missing = append(missing, capability)
			}
		}
	}
	for _, fault := range experiment.Faults {
		for _, capability := range fault.RequiredCapabilities {
			if _, exists := have[protocolcatalog.CapabilityID(capability)]; !exists {
				missing = append(missing, capability)
			}
		}
	}
	slices.Sort(missing)
	return slices.Compact(missing)
}

func hasFaultCapability(capabilities []protocolcatalog.CapabilityID) bool {
	for _, capability := range capabilities {
		if strings.HasPrefix(string(capability), "fault-") || capability == protocolcatalog.CapabilityIDFailoverControl {
			return true
		}
	}
	return false
}

func capabilityStrings(capabilities []protocolcatalog.CapabilityID) []string {
	values := make([]string, len(capabilities))
	for index, capability := range capabilities {
		values[index] = string(capability)
	}
	return values
}
