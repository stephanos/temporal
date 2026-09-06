// Package testpilot prepares bounded Cases for execution through authorized Drivers.
package testpilot

import (
	"context"
	"errors"
	"reflect"

	testpilotpb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/internal/execution"
	"go.temporal.io/server/common/testing/testpilot/internal/verification"
	"google.golang.org/protobuf/proto"
)

// PreparedCase owns immutable admission products, never a live Profile or Driver.
type PreparedCase struct {
	source   *testpilotpb.Case
	program  *execution.PreparedProgram
	factory  execution.MonitorFactory
	identity DriverIdentity
}

func Prepare(source *testpilotpb.Case, profile Profile) (*PreparedCase, error) {
	if isNil(profile) {
		return nil, errors.New("Profile is required")
	}
	spec := profile.Snapshot()
	if spec.Catalog == nil || spec.Catalog.catalog == nil {
		return nil, errors.New("Profile catalog is required")
	}
	policy := spec.policy()
	program, err := execution.Prepare(source, spec.Catalog.catalog, policy)
	if err != nil {
		return nil, err
	}
	contract, err := verification.Prepare(source.Contract, spec.Catalog.catalog, program.View(), spec.ContractLimits)
	if err != nil {
		return nil, err
	}
	return &PreparedCase{source: proto.CloneOf(source), program: program, factory: contract, identity: DriverIdentity{Profile: policy.Identity, Catalog: policy.CatalogIdentity}}, nil
}

func (p *PreparedCase) Snapshot() *testpilotpb.Case { return proto.CloneOf(p.source) }
func (p *PreparedCase) Identity() DriverIdentity    { return p.identity }

func (p *PreparedCase) preflight(ctx context.Context, driver Driver) (execution.Driver, execution.Monitor, error) {
	if p == nil || p.program == nil || isNil(ctx) || isNil(driver) || isNil(p.factory) {
		return nil, nil, errors.New("prepared Case, context, Driver and Contract factory are required")
	}
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	identity, err := driver.Identity(ctx)
	if err != nil {
		return nil, nil, err
	}
	if identity != p.identity {
		return nil, nil, errors.New("Driver Profile or catalog identity changed")
	}
	monitor, err := execution.NewMonitor(ctx, p.factory, p.program.View())
	if err != nil {
		return nil, nil, err
	}
	return driverAdapter{driver: driver}, monitor, nil
}

func isNil(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice, reflect.UnsafePointer:
		return reflected.IsNil()
	default:
		return false
	}
}
