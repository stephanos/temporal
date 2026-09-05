package umpire

import (
	"slices"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/internal/execution"
	"go.temporal.io/server/tools/umpire/internal/ir"
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

type Capability = execution.Opcode

const (
	InvokeRPC              = execution.InvokeRPC
	AwaitSlot              = execution.AwaitSlot
	CompleteNexusOperation = execution.CompleteNexusOperation
	StartNexusOperation    = execution.StartNexusOperation
	Await                  = execution.Await
	Finish                 = execution.Finish
	RespondNexus           = execution.RespondNexus
)

type RolePolicy = execution.RolePolicy
type ReservationCarrierPolicy = execution.ReservationCarrierPolicy
type ReservationCarrierShape = execution.ReservationCarrierShape

// Profile supplies static authorization only. Snapshot must not perform target I/O.
// Identity must change whenever authorization, reservation carrier policy, resource ceilings or
// role bindings change; rotating credentials for the same authorized identity does not change it.
type Profile interface{ Snapshot() ProfileSpec }
type ProfileSpec struct {
	Identity       string
	Catalog        *Catalog
	Roles          []RolePolicy
	Capabilities   []Capability
	ProgramLimits  *umpirespb.ProgramLimits
	ContractLimits *umpirespb.ContractLimits
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
