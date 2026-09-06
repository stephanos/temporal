package testpilot

import (
	"slices"

	testpilotpb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/internal/execution"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

// Catalog freezes the descriptor graph; it contains no channels or credentials.
type Catalog struct{ catalog *ir.Catalog }

func NewCatalog(source *descriptorpb.FileDescriptorSet) (*Catalog, error) {
	catalog, err := ir.NewCatalog(source)
	if err != nil {
		return nil, err
	}
	return &Catalog{catalog: catalog}, nil
}

func (c *Catalog) Identity() string {
	if c == nil || c.catalog == nil {
		return ""
	}
	return c.catalog.Identity()
}

type Capability uint8

const (
	InvokeRPC Capability = iota + 1
	AwaitSlot
	CompleteNexusOperation
	StartNexusOperation
	Await
	Finish
	RespondNexus
)

type RolePolicy struct {
	ID                  string
	Kind                testpilotpb.RoleKind
	Methods             []string
	ReservationCarriers []ReservationCarrierPolicy
}

type ReservationCarrierPolicy struct {
	Method string
	Shapes []ReservationCarrierShape
}

type ReservationCarrierShape struct {
	Context      testpilotpb.EntrypointKind
	MaximumCount int64
}

// Profile supplies static authorization only. Snapshot must not perform target I/O.
// Identity must change whenever authorization, reservation carrier policy, resource ceilings or
// role bindings change; rotating credentials for the same authorized identity does not change it.
type Profile interface{ Snapshot() ProfileSpec }

type ProfileSpec struct {
	Identity       string
	Catalog        *Catalog
	Roles          []RolePolicy
	Capabilities   []Capability
	ProgramLimits  *testpilotpb.ProgramLimits
	ContractLimits *testpilotpb.ContractLimits
}

func (p ProfileSpec) policy() execution.Policy {
	roles := make([]execution.RolePolicy, len(p.Roles))
	for i, role := range p.Roles {
		carriers := make([]execution.ReservationCarrierPolicy, len(role.ReservationCarriers))
		for j, carrier := range role.ReservationCarriers {
			shapes := make([]execution.ReservationCarrierShape, len(carrier.Shapes))
			for k, shape := range carrier.Shapes {
				shapes[k] = execution.ReservationCarrierShape{Context: shape.Context, MaximumCount: shape.MaximumCount}
			}
			carriers[j] = execution.ReservationCarrierPolicy{Method: carrier.Method, Shapes: shapes}
		}
		roles[i] = execution.RolePolicy{ID: role.ID, Kind: role.Kind, Methods: slices.Clone(role.Methods), ReservationCarriers: carriers}
	}
	capabilities := make([]execution.Opcode, len(p.Capabilities))
	for i, capability := range p.Capabilities {
		capabilities[i] = execution.Opcode(capability)
	}
	return execution.Policy{Identity: p.Identity, CatalogIdentity: p.Catalog.Identity(), Roles: roles, Capabilities: capabilities, Limits: proto.CloneOf(p.ProgramLimits)}
}

func (p ProfileSpec) Snapshot() ProfileSpec {
	snapshot := p
	snapshot.ProgramLimits = proto.CloneOf(p.ProgramLimits)
	snapshot.ContractLimits = proto.CloneOf(p.ContractLimits)
	snapshot.Capabilities = slices.Clone(p.Capabilities)
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
