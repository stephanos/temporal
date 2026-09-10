package temporal

import (
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"google.golang.org/protobuf/proto"
)

// Environment names the physical resources one Case's symbolic bindings resolve to, plus the
// identity the derived Profile carries. A Case that binds no Nexus endpoint leaves it empty.
type Environment struct {
	Identity      string
	Namespace     string
	TaskQueue     string
	NexusEndpoint string
}

// DeriveProfile returns the minimal authorization the Case implies: the roles it declares, the
// methods it invokes, the reservation carriers its instructions actually use, the capabilities its
// capabilities require, and the environment values its declared bindings resolve to. Nothing is widened
// beyond what the Case references, and anything the Case names that the catalog does not know is
// an error rather than a silently authorized surface.
//
// The Profile stays an authorization snapshot, so the derived value is returned for the caller to
// review and tighten before Prepare rather than applied on its behalf.
func DeriveProfile(source *testpilotspb.Case, catalog *testpilot.Catalog, environment Environment) (testpilot.ProfileSpec, error) {
	program := source.GetProgram()
	if program == nil || catalog == nil || environment.Identity == "" {
		return testpilot.ProfileSpec{}, ErrInvalid
	}
	contexts, err := entrypointKinds(program)
	if err != nil {
		return testpilot.ProfileSpec{}, err
	}
	usage, err := deriveUsage(program, contexts, catalog)
	if err != nil {
		return testpilot.ProfileSpec{}, err
	}
	roles, err := deriveRoles(program, usage)
	if err != nil {
		return testpilot.ProfileSpec{}, err
	}
	bindings, err := deriveBindings(program, environment)
	if err != nil {
		return testpilot.ProfileSpec{}, err
	}
	return testpilot.ProfileSpec{
		Identity:            environment.Identity,
		Catalog:             catalog,
		Roles:               roles,
		Opcodes:             usage.capabilities(),
		EnvironmentBindings: bindings,
		ProgramLimits:       proto.CloneOf(program.GetLimits()),
		ContractLimits:      proto.CloneOf(source.GetContract().GetLimits()),
	}, nil
}

func entrypointKinds(program *testpilotspb.Program) (map[string]testpilotspb.EntrypointKind, error) {
	contexts := make(map[string]testpilotspb.EntrypointKind, len(program.GetEntrypoints()))
	for _, entrypoint := range program.GetEntrypoints() {
		var kind testpilotspb.EntrypointKind
		switch entrypoint.GetActivation().(type) {
		case *testpilotspb.EntrypointDefinition_Controller:
			kind = testpilotspb.ENTRYPOINT_KIND_CONTROLLER
		case *testpilotspb.EntrypointDefinition_Workflow:
			kind = testpilotspb.ENTRYPOINT_KIND_WORKFLOW
		case *testpilotspb.EntrypointDefinition_Activity:
			kind = testpilotspb.ENTRYPOINT_KIND_ACTIVITY
		case *testpilotspb.EntrypointDefinition_NexusHandler:
			kind = testpilotspb.ENTRYPOINT_KIND_NEXUS_HANDLER
		default:
			return nil, ErrInvalid
		}
		if _, duplicate := contexts[entrypoint.GetEntrypointId()]; duplicate {
			return nil, ErrInvalid
		}
		contexts[entrypoint.GetEntrypointId()] = kind
	}
	return contexts, nil
}

// carrierKey names one reservation carrier: the method, on the endpoint role, whose instructions
// reserve activations.
type carrierKey struct{ role, method string }

// programUsage is what the Case actually references, in the order it references it. Order is kept
// so a derived Profile compares equal to a hand-written one rather than only equivalent.
type programUsage struct {
	methods      map[string][]string
	methodSeen   map[carrierKey]bool
	carrierOrder map[string][]string
	shapes       map[carrierKey]map[testpilotspb.EntrypointKind]int64
	opcodes      map[testpilot.Opcode]bool
}

func (u *programUsage) capabilities() []testpilot.Opcode {
	result := make([]testpilot.Opcode, 0, len(u.opcodes))
	for capability := testpilot.InvokeRPC; capability <= testpilot.MaxOpcode; capability++ {
		if u.opcodes[capability] {
			result = append(result, capability)
		}
	}
	return result
}

func deriveUsage(program *testpilotspb.Program, contexts map[string]testpilotspb.EntrypointKind, catalog *testpilot.Catalog) (*programUsage, error) {
	usage := &programUsage{
		methods:      map[string][]string{},
		methodSeen:   map[carrierKey]bool{},
		carrierOrder: map[string][]string{},
		shapes:       map[carrierKey]map[testpilotspb.EntrypointKind]int64{},
		opcodes:      map[testpilot.Opcode]bool{},
	}
	for _, entrypoint := range program.GetEntrypoints() {
		for _, instruction := range entrypoint.GetInstructions() {
			if err := usage.add(instruction, contexts, catalog); err != nil {
				return nil, err
			}
		}
	}
	for _, instruction := range program.GetCleanup().GetInstructions() {
		if err := usage.add(instruction, contexts, catalog); err != nil {
			return nil, err
		}
	}
	return usage, nil
}

func (u *programUsage) add(instruction *testpilotspb.InstructionDefinition, contexts map[string]testpilotspb.EntrypointKind, catalog *testpilot.Catalog) error {
	capability := testpilot.InstructionCapability(instruction.GetInstruction())
	if capability == 0 {
		return ErrInvalid
	}
	u.opcodes[capability] = true
	rpc := instruction.GetInstruction().GetInvokeRpc()
	if rpc == nil {
		return nil
	}
	if err := catalog.CheckMethod(rpc.GetMethod()); err != nil {
		return err
	}
	key := carrierKey{role: rpc.GetEndpointRoleId(), method: rpc.GetMethod()}
	if !u.methodSeen[key] {
		u.methodSeen[key] = true
		u.methods[key.role] = append(u.methods[key.role], key.method)
	}
	if len(instruction.GetActivationReservations()) == 0 {
		return nil
	}
	if u.shapes[key] == nil {
		u.shapes[key] = map[testpilotspb.EntrypointKind]int64{}
		u.carrierOrder[key.role] = append(u.carrierOrder[key.role], key.method)
	}
	// Carrier shapes are checked per reserving node, so the ceiling one carrier needs is the
	// largest single node's reservation of that context, never the sum across nodes: summing
	// would authorize more than any one instruction can ask for.
	node := map[testpilotspb.EntrypointKind]int64{}
	for _, reservation := range instruction.GetActivationReservations() {
		kind, declared := contexts[reservation.GetEntrypointId()]
		if !declared || reservation.GetCount() <= 0 {
			return ErrInvalid
		}
		node[kind] += reservation.GetCount()
	}
	for kind, count := range node {
		u.shapes[key][kind] = max(u.shapes[key][kind], count)
	}
	return nil
}

func deriveRoles(program *testpilotspb.Program, usage *programUsage) ([]testpilot.RolePolicy, error) {
	roles := make([]testpilot.RolePolicy, 0, len(program.GetRoles()))
	seen := make(map[string]struct{}, len(program.GetRoles()))
	for _, role := range program.GetRoles() {
		if role.GetKind() < testpilotspb.ROLE_KIND_ENDPOINT || role.GetKind() > testpilotspb.ROLE_KIND_PARTICIPANT {
			return nil, ErrInvalid
		}
		if _, duplicate := seen[role.GetRoleId()]; duplicate {
			return nil, ErrInvalid
		}
		seen[role.GetRoleId()] = struct{}{}
		policy := testpilot.RolePolicy{ID: role.GetRoleId(), Kind: role.GetKind(), Methods: usage.methods[role.GetRoleId()]}
		for _, method := range usage.carrierOrder[role.GetRoleId()] {
			counts := usage.shapes[carrierKey{role: role.GetRoleId(), method: method}]
			shapes := make([]testpilot.ReservationCarrierShape, 0, len(counts))
			for kind := testpilotspb.ENTRYPOINT_KIND_CONTROLLER; kind <= testpilotspb.ENTRYPOINT_KIND_NEXUS_HANDLER; kind++ {
				if count := counts[kind]; count > 0 {
					shapes = append(shapes, testpilot.ReservationCarrierShape{Kind: kind, MaximumCount: count})
				}
			}
			policy.ReservationCarriers = append(policy.ReservationCarriers, testpilot.ReservationCarrierPolicy{Method: method, Shapes: shapes})
		}
		roles = append(roles, policy)
	}
	// A method attributed to a role the Program never declared would authorize a surface the Case
	// cannot reach, so it is an error rather than a dropped entry.
	for role := range usage.methods {
		if _, declared := seen[role]; !declared {
			return nil, ErrInvalid
		}
	}
	return roles, nil
}

// deriveBindings resolves each declared environment binding through the role that references it.
// A binding no role claims has no derivable value, so it rejects rather than resolving to empty.
func deriveBindings(program *testpilotspb.Program, environment Environment) ([]testpilot.EnvironmentBinding, error) {
	values := map[string]string{}
	for _, role := range program.GetRoles() {
		switch role.GetKind() {
		case testpilotspb.ROLE_KIND_WORKER, testpilotspb.ROLE_KIND_TASK_QUEUE:
			if id := role.GetNamespaceBindingId(); id != "" {
				values[id] = environment.Namespace
			}
			if id := role.GetResourceBindingId(); id != "" && role.GetKind() == testpilotspb.ROLE_KIND_TASK_QUEUE {
				values[id] = environment.TaskQueue
			}
		case testpilotspb.ROLE_KIND_ENDPOINT:
			if id := role.GetResourceBindingId(); id != "" {
				values[id] = environment.NexusEndpoint
			}
		default:
		}
	}
	bindings := make([]testpilot.EnvironmentBinding, 0, len(program.GetEnvironment()))
	for _, declaration := range program.GetEnvironment() {
		value, resolved := values[declaration.GetBindingId()]
		if !resolved || value == "" {
			return nil, ErrInvalid
		}
		bindings = append(bindings, testpilot.EnvironmentBinding{ID: declaration.GetBindingId(), Value: value})
	}
	return bindings, nil
}
