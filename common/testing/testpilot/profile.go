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

	enumspb "go.temporal.io/api/enums/v1"
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
var InstructionOpcode = execution.InstructionOpcode

// EnvironmentBindingIDs is the symbolic binding graph a Program references, the set Prepare resolves
// against a Profile. Callers deriving a Profile from a Case read it rather than restating it.
var EnvironmentBindingIDs = execution.EnvironmentBindingIDs

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

// Profile supplies static authorization only. Snapshot must not perform target I/O, and returns a
// spec its caller owns: no collection or limit in it is shared with the Profile or a prior snapshot.
// Identity must change whenever authorization, reservation carrier policy, resource ceilings,
// instruction defaults or role bindings change; rotating credentials for the same authorized identity
// does not change it.
type Profile interface{ Snapshot() ProfileSpec }

// ConfigurationValue is one dynamic configuration value the environment a Profile describes runs
// under: a machine's setup parameter bound through the realization's key, or one value of a switch a
// functional set repeats over. The Case bytes do not depend on it; the Profile records it, so two
// Runs of one Case under two values carry two Profiles.
type ConfigurationValue struct {
	Key   string
	Value string
}

// ProfileSpec is one Profile snapshot. Its limits are the resource ceilings every admitted Case runs
// under: a Case declares none of them, only the bounds that carry its behavior, which admission
// checks against these ceilings. An instruction that writes no timeout or attempts takes
// InstructionDefaults. CorrelatedLimits is required only to admit a correlated contract.
// Configuration is the dynamic configuration the environment sets, and is part of the binding
// fingerprint the prepared Case's identity carries.
type ProfileSpec struct {
	Identity string
	Catalog  *Catalog
	Roles    []RolePolicy
	Opcodes  []Opcode
	// CommandTypes are the workflow command types a WorkflowCommand may carry: the ones the Driver
	// realizes through its SDK, as DeriveProfile records them.
	CommandTypes        []enumspb.CommandType
	EnvironmentBindings []EnvironmentBinding
	Configuration       []ConfigurationValue
	ProgramLimits       *testpilotspb.ProgramLimits
	ContractLimits      *testpilotspb.ContractLimits
	CorrelatedLimits    *testpilotspb.CorrelatedLimits
	InstructionDefaults InstructionDefaults
}

// WithinProgramCeiling reports whether every ProgramLimits field is positive and within the ceiling
// admission enforces. Drivers check a Profile's limits with it once, at construction.
func WithinProgramCeiling(limits *testpilotspb.ProgramLimits) bool {
	return ir.CheckCeilings(limits, execution.ProgramCeiling(), func(string) error { return errOutsideCeiling }) == nil
}

var errOutsideCeiling = errors.New("program limit outside the ceiling")

func (p ProfileSpec) Snapshot() ProfileSpec {
	snapshot := p
	snapshot.ProgramLimits = proto.CloneOf(p.ProgramLimits)
	snapshot.ContractLimits = proto.CloneOf(p.ContractLimits)
	snapshot.CorrelatedLimits = proto.CloneOf(p.CorrelatedLimits)
	snapshot.Opcodes = slices.Clone(p.Opcodes)
	snapshot.CommandTypes = slices.Clone(p.CommandTypes)
	snapshot.EnvironmentBindings = slices.Clone(p.EnvironmentBindings)
	snapshot.Configuration = slices.Clone(p.Configuration)
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

// BindingFingerprint validates and identifies the complete environment binding snapshot, the
// configuration the environment runs under included. Rejections are malformed Profile preparation
// errors, including binding ceiling failures. A Profile with no bindings and no configuration has
// no fingerprint, so a Profile that carried neither before keeps the identity it had.
func (p ProfileSpec) BindingFingerprint() (string, error) {
	configuration, err := p.canonicalConfiguration()
	if err != nil {
		return "", err
	}
	if len(p.EnvironmentBindings) == 0 && len(configuration) == 0 {
		return "", nil
	}
	if p.ProgramLimits == nil {
		return "", preparationError(errors.New("Profile Program limits are required"), "profile.program_limits")
	}
	var total int64
	bindings, err := sortedByID(p.EnvironmentBindings, "Profile environment binding", "identity", "profile.environment_bindings",
		func(binding EnvironmentBinding) (string, string) { return binding.ID, binding.Value },
		func(binding EnvironmentBinding) error {
			bytes := int64(len(binding.ID) + len(binding.Value))
			if bytes > p.ProgramLimits.MaxRequestBytes-total {
				return errors.New("Profile environment binding byte ceiling exceeded")
			}
			total += bytes
			return nil
		})
	if err != nil {
		return "", err
	}
	canonical := []byte("testpilot.environment-bindings/v1")
	for _, binding := range bindings {
		canonical = binary.BigEndian.AppendUint64(canonical, uint64(len(binding.ID)))
		canonical = append(canonical, binding.ID...)
		canonical = binary.BigEndian.AppendUint64(canonical, uint64(len(binding.Value)))
		canonical = append(canonical, binding.Value...)
	}
	// The configuration section is appended only when there is one, so a Profile that sets no
	// configuration fingerprints exactly as it did before configuration was recorded.
	if len(configuration) > 0 {
		canonical = append(canonical, "testpilot.configuration/v1"...)
		for _, value := range configuration {
			canonical = binary.BigEndian.AppendUint64(canonical, uint64(len(value.Key)))
			canonical = append(canonical, value.Key...)
			canonical = binary.BigEndian.AppendUint64(canonical, uint64(len(value.Value)))
			canonical = append(canonical, value.Value...)
		}
	}
	fingerprint := sha256.Sum256(canonical)
	return hex.EncodeToString(fingerprint[:]), nil
}

// canonicalConfiguration validates the configuration values and returns them sorted by key. A key
// is spelled the way the dynamic configuration catalog spells it; a value is any non-empty text.
func (p ProfileSpec) canonicalConfiguration() ([]ConfigurationValue, error) {
	if len(p.Configuration) == 0 {
		return nil, nil
	}
	return sortedByID(p.Configuration, "Profile configuration", "key", "profile.configuration",
		func(value ConfigurationValue) (string, string) { return value.Key, value.Value }, nil)
}

// sortedByID clones entries sorted by identity and admits each in that order: an identity spelled
// like an environment identity, a non-empty UTF-8 value, no identity twice, then admit when given.
// noun names an entry and idNoun its identity in the errors, which carry path.
func sortedByID[T any](entries []T, noun, idNoun, path string, fields func(T) (id, value string), admit func(T) error) ([]T, error) {
	if len(entries) > 10000 {
		return nil, preparationError(fmt.Errorf("%s collection ceiling exceeded", noun), path)
	}
	sorted := slices.Clone(entries)
	slices.SortFunc(sorted, func(a, b T) int {
		left, _ := fields(a)
		right, _ := fields(b)
		return cmp.Compare(left, right)
	})
	previous := ""
	for i, entry := range sorted {
		id, value := fields(entry)
		if !ir.ValidID(id) || !utf8.ValidString(id) {
			return nil, preparationError(fmt.Errorf("%s %d has an invalid %s", noun, i, idNoun), path)
		}
		if value == "" || !utf8.ValidString(value) {
			return nil, preparationError(fmt.Errorf("%s %q has an invalid value", noun, id), path)
		}
		if i > 0 && previous == id {
			return nil, preparationError(fmt.Errorf("%s %q is duplicated", noun, id), path)
		}
		previous = id
		if admit != nil {
			if err := admit(entry); err != nil {
				return nil, preparationError(err, path)
			}
		}
	}
	return sorted, nil
}
