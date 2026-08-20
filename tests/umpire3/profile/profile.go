package profile

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

	"go.temporal.io/server/tests/umpire3/environment"
	"go.temporal.io/server/tests/umpire3/protocol"
)

type Kind string

const (
	KindLocal    Kind = "local-in-process"
	KindCI       Kind = "ci-test-cluster"
	KindRemote   Kind = "remote-deployment"
	KindBlackBox Kind = "grpc-only-black-box"
	KindCanary   Kind = "production-canary"
)

type Config struct {
	Kind                Kind     `json:"kind"`
	Endpoint            string   `json:"endpoint,omitempty"`
	AuthToken           string   `json:"-"`
	BuildID             string   `json:"buildID"`
	Namespace           string   `json:"namespace"`
	TaskQueue           string   `json:"taskQueue"`
	WorkerCommand       []string `json:"-"`
	HardExecutionBudget bool     `json:"hardExecutionBudget"`
	Capabilities        []string `json:"capabilities,omitempty"`
}

type Attestation struct {
	BuildID             string `json:"buildID"`
	ConfigurationDigest string `json:"configurationDigest"`
	EndpointIdentity    string `json:"endpointIdentity,omitempty"`
}

type Definition struct {
	Kind          Kind                `json:"kind"`
	Environment   environment.Profile `json:"environment"`
	Capabilities  []string            `json:"capabilities"`
	Attestation   Attestation         `json:"attestation"`
	Endpoint      string              `json:"endpoint,omitempty"`
	Namespace     string              `json:"namespace"`
	TaskQueue     string              `json:"taskQueue"`
	workerCommand []string
}

func Local(buildID, namespace, taskQueue string) Config {
	return Config{Kind: KindLocal, BuildID: buildID, Namespace: namespace, TaskQueue: taskQueue}
}

func CI(buildID, namespace, taskQueue string) Config {
	return Config{Kind: KindCI, BuildID: buildID, Namespace: namespace, TaskQueue: taskQueue}
}

func Remote(endpoint, authToken, buildID, namespace, taskQueue string) Config {
	return Config{
		Kind: KindRemote, Endpoint: endpoint, AuthToken: authToken, BuildID: buildID,
		Namespace: namespace, TaskQueue: taskQueue,
	}
}

func BlackBox(endpoint, authToken, buildID, namespace, taskQueue string) Config {
	config := Remote(endpoint, authToken, buildID, namespace, taskQueue)
	config.Kind = KindBlackBox
	return config
}

func Canary(endpoint, authToken, buildID, namespace, taskQueue string, workerCommand []string) Config {
	config := Remote(endpoint, authToken, buildID, namespace, taskQueue)
	config.Kind = KindCanary
	config.WorkerCommand = append([]string(nil), workerCommand...)
	config.HardExecutionBudget = true
	return config
}

func Define(config Config) (Definition, error) {
	if config.BuildID == "" || config.Namespace == "" || config.TaskQueue == "" {
		return Definition{}, errors.New("build, namespace, and task queue identities are required")
	}
	if config.HardExecutionBudget && len(config.WorkerCommand) == 0 {
		return Definition{}, errors.New("hard execution budget requires a killable worker command")
	}
	if err := validateEndpoint(config); err != nil {
		return Definition{}, err
	}

	profile := environment.Profile{
		Name: config.Kind.String(), BuildID: config.BuildID,
		ConfigurationIdentity: configurationDigest(config),
		IsolationIdentity:     config.Namespace + "/" + config.TaskQueue,
		RetentionClass:        "semantic-redacted", HardExecutionBudget: config.HardExecutionBudget,
	}
	capabilities := catalogCapabilities()
	switch config.Kind {
	case KindLocal:
		profile.EvidenceProfile = environment.EvidenceProfileInProcessHooks
		profile.DrivingAuthority = "local-test-authority"
		profile.ObservationAuthority = "local-server-hooks"
		profile.FaultAuthority = "isolated-local-faults"
	case KindCI:
		profile.EvidenceProfile = environment.EvidenceProfilePublicGRPCHistory
		profile.DrivingAuthority = "ci-test-cluster"
		profile.ObservationAuthority = "ci-public-history"
		profile.FaultAuthority = "isolated-ci-faults"
	case KindRemote:
		profile.EvidenceProfile = environment.EvidenceProfilePublicGRPCHistory
		profile.DrivingAuthority = "remote-api"
		profile.ObservationAuthority = "remote-public-history"
		profile.FaultAuthority = "remote-approved-faults"
		capabilities = withoutCapabilities(capabilities,
			protocol.CapabilityIDFaultProcess, protocol.CapabilityIDFaultPersistence)
	case KindBlackBox:
		profile.EvidenceProfile = environment.EvidenceProfilePublicGRPC
		profile.DrivingAuthority = "public-grpc"
		profile.ObservationAuthority = "public-grpc"
		profile.FaultAuthority = "none"
		capabilities = withoutCapabilities(capabilities,
			protocol.CapabilityIDFaultProcess, protocol.CapabilityIDFaultNetwork,
			protocol.CapabilityIDFaultClock, protocol.CapabilityIDFaultPersistence,
			protocol.CapabilityIDFailoverControl)
	case KindCanary:
		profile.EvidenceProfile = environment.EvidenceProfilePublicGRPCHistory
		profile.DrivingAuthority = "approved-production-worker"
		profile.ObservationAuthority = "production-public-history"
		profile.FaultAuthority = "approved-production-fault-controller"
		capabilities = withoutCapabilities(capabilities,
			protocol.CapabilityIDFaultProcess, protocol.CapabilityIDFaultNetwork,
			protocol.CapabilityIDFaultClock, protocol.CapabilityIDFaultPersistence)
	default:
		return Definition{}, fmt.Errorf("unknown deployment profile %q", config.Kind)
	}
	if config.Capabilities != nil {
		capabilities = intersectCapabilities(capabilities, config.Capabilities)
	}
	if err := profile.Validate(); err != nil {
		return Definition{}, err
	}
	return Definition{
		Kind: config.Kind, Environment: profile, Capabilities: capabilities,
		Attestation: Attestation{
			BuildID: config.BuildID, ConfigurationDigest: profile.ConfigurationIdentity,
			EndpointIdentity: digest(config.Endpoint),
		},
		Endpoint: config.Endpoint, Namespace: config.Namespace, TaskQueue: config.TaskQueue,
		workerCommand: append([]string(nil), config.WorkerCommand...),
	}, nil
}

func (k Kind) String() string {
	return string(k)
}

func (d Definition) String() string {
	encoded, _ := json.Marshal(d)
	return string(encoded)
}

func (d Definition) Digest() (string, error) {
	encoded, err := json.Marshal(d)
	if err != nil {
		return "", fmt.Errorf("encode profile definition: %w", err)
	}
	return digest(string(encoded)), nil
}

func (d Definition) WorkerCommand() []string {
	return append([]string(nil), d.workerCommand...)
}

func validateEndpoint(config Config) error {
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

func configurationDigest(config Config) string {
	capabilities := append([]string(nil), config.Capabilities...)
	slices.Sort(capabilities)
	return digest(strings.Join([]string{
		string(config.Kind), config.Endpoint, config.BuildID, config.Namespace, config.TaskQueue,
		fmt.Sprint(config.HardExecutionBudget), strings.Join(config.WorkerCommand, "\x00"),
		strings.Join(capabilities, "\x00"),
	}, "\x00"))
}

func digest(value string) string {
	if value == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func catalogCapabilities() []string {
	catalog, err := protocol.DefaultCatalog()
	if err != nil {
		return nil
	}
	result := make([]string, len(catalog.Capabilities))
	for index, capability := range catalog.Capabilities {
		result[index] = string(capability.Identifier)
	}
	slices.Sort(result)
	return result
}

func withoutCapabilities(values []string, removed ...protocol.CapabilityID) []string {
	blocked := make(map[string]struct{}, len(removed))
	for _, capability := range removed {
		blocked[string(capability)] = struct{}{}
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, remove := blocked[value]; !remove {
			result = append(result, value)
		}
	}
	return result
}

type boundFactory struct {
	definition   Definition
	underlying   environment.Factory
	capabilities []string
}

func Bind(definition Definition, underlying environment.Factory) (environment.Factory, error) {
	if underlying == nil {
		return nil, errors.New("underlying environment factory is required")
	}
	if err := definition.Environment.Validate(); err != nil {
		return nil, fmt.Errorf("validate profile: %w", err)
	}
	if definition.Environment.HardExecutionBudget {
		return nil, errors.New("hard-budget profile must execute through its killable worker command")
	}
	capabilities := intersectCapabilities(definition.Capabilities, underlying.Capabilities())
	return &boundFactory{definition: definition, underlying: underlying, capabilities: capabilities}, nil
}

func (f *boundFactory) Capabilities() []string {
	return append([]string(nil), f.capabilities...)
}

func (f *boundFactory) Prepare(ctx context.Context, experiment protocol.Experiment) (environment.Session, error) {
	if missing := missingCapabilities(experiment, f.capabilities); len(missing) != 0 {
		return nil, fmt.Errorf("unsupported capabilities: %v", missing)
	}
	session, err := f.underlying.Prepare(ctx, experiment)
	if err != nil || session == nil {
		return session, err
	}
	return &profiledSession{Session: session, profile: f.definition.Environment}, nil
}

type profiledSession struct {
	environment.Session
	profile environment.Profile
}

func (s *profiledSession) Profile() environment.Profile {
	return s.profile
}

func intersectCapabilities(left, right []string) []string {
	rightSet := make(map[string]struct{}, len(right))
	for _, capability := range right {
		rightSet[capability] = struct{}{}
	}
	var result []string
	for _, capability := range left {
		if _, exists := rightSet[capability]; exists {
			result = append(result, capability)
		}
	}
	slices.Sort(result)
	return slices.Compact(result)
}

func missingCapabilities(experiment protocol.Experiment, available []string) []string {
	have := make(map[string]struct{}, len(available))
	for _, capability := range available {
		have[capability] = struct{}{}
	}
	var missing []string
	for _, action := range experiment.Actions {
		for _, capability := range action.RequiredCapabilities {
			if _, exists := have[capability]; !exists {
				missing = append(missing, capability)
			}
		}
	}
	for _, fault := range experiment.Faults {
		for _, capability := range fault.RequiredCapabilities {
			if _, exists := have[capability]; !exists {
				missing = append(missing, capability)
			}
		}
	}
	slices.Sort(missing)
	return slices.Compact(missing)
}

type Dimensions map[string][]string

type Assignment map[string]string

func Pairwise(dimensions Dimensions) ([]Assignment, error) {
	names, domains, err := normalizeDimensions(dimensions)
	if err != nil {
		return nil, err
	}
	assignments := cartesian(names, domains)
	uncovered := allPairs(names, domains)
	var selected []Assignment
	for len(uncovered) != 0 {
		best := -1
		bestCoverage := -1
		for index, assignment := range assignments {
			coverage := coveredPairCount(names, assignment, uncovered)
			if coverage > bestCoverage {
				best, bestCoverage = index, coverage
			}
		}
		if best < 0 || bestCoverage == 0 {
			return nil, errors.New("pairwise matrix could not cover remaining pairs")
		}
		chosen := cloneAssignment(assignments[best])
		selected = append(selected, chosen)
		removeCoveredPairs(names, chosen, uncovered)
		assignments = append(assignments[:best], assignments[best+1:]...)
	}
	return selected, nil
}

func CoversEveryPair(dimensions Dimensions, assignments []Assignment) bool {
	names, domains, err := normalizeDimensions(dimensions)
	if err != nil {
		return false
	}
	uncovered := allPairs(names, domains)
	for _, assignment := range assignments {
		removeCoveredPairs(names, assignment, uncovered)
	}
	return len(uncovered) == 0
}

type pair struct {
	leftName, leftValue, rightName, rightValue string
}

func normalizeDimensions(dimensions Dimensions) ([]string, map[string][]string, error) {
	if len(dimensions) < 2 {
		return nil, nil, errors.New("at least two pairwise dimensions are required")
	}
	names := make([]string, 0, len(dimensions))
	domains := make(map[string][]string, len(dimensions))
	for name, values := range dimensions {
		if name == "" || len(values) == 0 {
			return nil, nil, errors.New("every pairwise dimension requires a name and values")
		}
		values = append([]string(nil), values...)
		slices.Sort(values)
		values = slices.Compact(values)
		if slices.Contains(values, "") {
			return nil, nil, fmt.Errorf("dimension %q contains an empty value", name)
		}
		names = append(names, name)
		domains[name] = values
	}
	slices.Sort(names)
	return names, domains, nil
}

func cartesian(names []string, domains map[string][]string) []Assignment {
	result := []Assignment{{}}
	for _, name := range names {
		var next []Assignment
		for _, assignment := range result {
			for _, value := range domains[name] {
				candidate := cloneAssignment(assignment)
				candidate[name] = value
				next = append(next, candidate)
			}
		}
		result = next
	}
	return result
}

func allPairs(names []string, domains map[string][]string) map[pair]struct{} {
	result := make(map[pair]struct{})
	for left := range names {
		for right := left + 1; right < len(names); right++ {
			for _, leftValue := range domains[names[left]] {
				for _, rightValue := range domains[names[right]] {
					result[pair{names[left], leftValue, names[right], rightValue}] = struct{}{}
				}
			}
		}
	}
	return result
}

func coveredPairCount(names []string, assignment Assignment, uncovered map[pair]struct{}) int {
	count := 0
	for left := range names {
		for right := left + 1; right < len(names); right++ {
			if _, exists := uncovered[pair{names[left], assignment[names[left]], names[right], assignment[names[right]]}]; exists {
				count++
			}
		}
	}
	return count
}

func removeCoveredPairs(names []string, assignment Assignment, uncovered map[pair]struct{}) {
	for left := range names {
		for right := left + 1; right < len(names); right++ {
			delete(uncovered, pair{names[left], assignment[names[left]], names[right], assignment[names[right]]})
		}
	}
}

func cloneAssignment(value Assignment) Assignment {
	result := make(Assignment, len(value))
	for key, item := range value {
		result[key] = item
	}
	return result
}
