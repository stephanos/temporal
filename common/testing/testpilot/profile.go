package testpilot

import (
	"cmp"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"unicode/utf8"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/internal/execution"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

// Catalog freezes the descriptor graph; it contains no channels or credentials.
type Catalog struct{ catalog *ir.Catalog }

// NewCatalog admits an immutable descriptor graph. Rejections expose
// *PreparationError through errors.As.
func NewCatalog(source *descriptorpb.FileDescriptorSet) (*Catalog, error) {
	catalog, err := ir.NewCatalog(source)
	if err != nil {
		return nil, preparationError(err, "catalog")
	}
	return &Catalog{catalog: catalog}, nil
}

func (c *Catalog) Identity() string {
	if c == nil || c.catalog == nil {
		return ""
	}
	return c.catalog.Identity()
}

// InstructionOpcode is the Opcode one declared instruction requires, or zero when the
// instruction is unset or outside the version-one table. Callers deriving a Profile from a Case
// read it rather than restating the mapping.
func InstructionOpcode(instruction *testpilotspb.Instruction) Opcode {
	return execution.InstructionOpcode(instruction)
}

// EnvironmentBindingIDs is the symbolic binding graph a Program references, the set Prepare resolves
// against a Profile. Callers deriving a Profile from a Case read it rather than restating it.
func EnvironmentBindingIDs(program *testpilotspb.Program) []string {
	return execution.EnvironmentBindingIDs(program)
}

// CheckMethod reports whether this catalog admits one unary gRPC method by its full path.
// Rejections expose *PreparationError through errors.As.
func (c *Catalog) CheckMethod(name string) error {
	if c == nil || c.catalog == nil {
		return preparationError(errors.New("catalog is required"), "catalog")
	}
	if _, err := c.catalog.Method(name); err != nil {
		return preparationError(err, "catalog")
	}
	return nil
}

// Profile supplies static authorization only. Snapshot must not perform target I/O.
// Identity must change whenever authorization, reservation carrier policy, resource ceilings,
// instruction defaults or role bindings change; rotating credentials for the same authorized identity
// does not change it.
type Profile interface{ Snapshot() ProfileSpec }

// ProfileSpec is one Profile snapshot. Its limits are the resource ceilings every admitted Case runs
// under: a Case declares none of them, only the bounds that carry its behavior, which admission
// checks against these ceilings. An instruction that writes no timeout or attempts takes
// InstructionDefaults. CorrelatedLimits is required only to admit a correlated contract.
type ProfileSpec struct {
	Identity            string
	Catalog             *Catalog
	Roles               []RolePolicy
	Opcodes             []Opcode
	EnvironmentBindings []EnvironmentBinding
	ProgramLimits       *testpilotspb.ProgramLimits
	ContractLimits      *testpilotspb.ContractLimits
	CorrelatedLimits    *testpilotspb.CorrelatedLimits
	InstructionDefaults InstructionDefaults
}

func (p ProfileSpec) Snapshot() ProfileSpec {
	snapshot := p
	snapshot.ProgramLimits = proto.CloneOf(p.ProgramLimits)
	snapshot.ContractLimits = proto.CloneOf(p.ContractLimits)
	snapshot.CorrelatedLimits = proto.CloneOf(p.CorrelatedLimits)
	snapshot.Opcodes = slices.Clone(p.Opcodes)
	snapshot.EnvironmentBindings = slices.Clone(p.EnvironmentBindings)
	snapshot.Roles = slices.Clone(p.Roles)
	for i, role := range p.Roles {
		snapshot.Roles[i].Methods = slices.Clone(role.Methods)
		snapshot.Roles[i].ReservationCarriers = slices.Clone(role.ReservationCarriers)
		for j, carrier := range role.ReservationCarriers {
			snapshot.Roles[i].ReservationCarriers[j].Shapes = slices.Clone(carrier.Shapes)
		}
	}
	return snapshot
}

// BindingFingerprint validates and identifies the complete environment binding snapshot.
// Rejections are malformed Profile preparation errors, including binding ceiling failures.
func (p ProfileSpec) BindingFingerprint() (string, error) {
	if len(p.EnvironmentBindings) == 0 {
		return "", nil
	}
	if p.ProgramLimits == nil {
		return "", preparationError(errors.New("Profile Program limits are required"), "profile.program_limits")
	}
	if len(p.EnvironmentBindings) > 10000 {
		return "", preparationError(errors.New("Profile environment binding collection ceiling exceeded"), "profile.environment_bindings")
	}
	bindings := slices.Clone(p.EnvironmentBindings)
	slices.SortFunc(bindings, func(a, b EnvironmentBinding) int { return cmp.Compare(a.ID, b.ID) })
	var total int64
	for i, binding := range bindings {
		if !validEnvironmentID(binding.ID) || !utf8.ValidString(binding.ID) {
			return "", preparationError(fmt.Errorf("Profile environment binding %d has an invalid identity", i), "profile.environment_bindings")
		}
		if binding.Value == "" || !utf8.ValidString(binding.Value) {
			return "", preparationError(fmt.Errorf("Profile environment binding %q has an invalid value", binding.ID), "profile.environment_bindings")
		}
		if i > 0 && bindings[i-1].ID == binding.ID {
			return "", preparationError(fmt.Errorf("Profile environment binding %q is duplicated", binding.ID), "profile.environment_bindings")
		}
		bytes := int64(len(binding.ID) + len(binding.Value))
		if bytes > p.ProgramLimits.MaxRequestBytes-total {
			return "", preparationError(errors.New("Profile environment binding byte ceiling exceeded"), "profile.environment_bindings")
		}
		total += bytes
	}
	canonical := []byte("testpilot.environment-bindings/v1")
	for _, binding := range bindings {
		canonical = binary.BigEndian.AppendUint64(canonical, uint64(len(binding.ID)))
		canonical = append(canonical, binding.ID...)
		canonical = binary.BigEndian.AppendUint64(canonical, uint64(len(binding.Value)))
		canonical = append(canonical, binding.Value...)
	}
	fingerprint := sha256.Sum256(canonical)
	return hex.EncodeToString(fingerprint[:]), nil
}

func validEnvironmentID(id string) bool {
	if len(id) == 0 || len(id) > 256 {
		return false
	}
	for _, c := range id {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '_', c == '-', c == '.':
		default:
			return false
		}
	}
	return true
}
