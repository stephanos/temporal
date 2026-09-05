// Package umpire prepares bounded Cases for execution through authorized Hosts.
package umpire

import (
	"context"
	"errors"
	"reflect"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/internal/execution"
	"go.temporal.io/server/tools/umpire/verification"
	"google.golang.org/protobuf/proto"
)

// PreparedCase owns immutable admission products, never a live Profile or Host.
type PreparedCase struct {
	source   *umpirespb.Case
	program  *execution.PreparedProgram
	factory  execution.MonitorFactory
	identity HostIdentity
}

func PrepareCase(source *umpirespb.Case, profile Profile) (*PreparedCase, error) {
	if isNil(profile) {
		return nil, errors.New("Profile is required")
	}
	spec := profile.Snapshot()
	if spec.Catalog == nil || spec.Catalog.catalog == nil {
		return nil, errors.New("Profile catalog is required")
	}
	policy := execution.Policy{Identity: spec.Identity, CatalogIdentity: spec.Catalog.Identity(), Roles: spec.Roles, Capabilities: spec.Capabilities, Limits: spec.ProgramLimits}
	program, err := execution.Prepare(source, spec.Catalog.catalog, policy)
	if err != nil {
		return nil, err
	}
	contract, err := verification.Prepare(source.Contract, spec.Catalog.catalog, program.View(), spec.ContractLimits)
	if err != nil {
		return nil, err
	}
	return &PreparedCase{source: proto.CloneOf(source), program: program, factory: contract, identity: HostIdentity{Profile: policy.Identity, Catalog: policy.CatalogIdentity}}, nil
}
func (p *PreparedCase) Snapshot() *umpirespb.Case { return proto.CloneOf(p.source) }
func (p *PreparedCase) Identity() HostIdentity    { return p.identity }

func (p *PreparedCase) preflight(ctx context.Context, host Host) (execution.Driver, execution.Monitor, error) {
	if p == nil || p.program == nil || isNil(ctx) || isNil(host) || isNil(p.factory) {
		return nil, nil, errors.New("prepared Case, context, Host and Contract factory are required")
	}
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	identity, err := host.Identity(ctx)
	if err != nil {
		return nil, nil, err
	}
	if identity != p.identity {
		return nil, nil, errors.New("Host Profile or catalog identity changed")
	}
	monitor, err := execution.NewMonitor(ctx, p.factory, p.program.View())
	if err != nil {
		return nil, nil, err
	}
	return driver{host: host}, monitor, nil
}
func isNil(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice, reflect.UnsafePointer:
		return v.IsNil()
	default:
		return false
	}
}
