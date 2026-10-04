// Package testpilot prepares bounded Cases for execution through authorized Drivers.
package testpilot

import (
	"context"
	"errors"
	"strings"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/internal/execution"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"go.temporal.io/server/common/testing/testpilot/internal/verification"
	"google.golang.org/protobuf/proto"
)

// PreparedCase owns immutable admission products, never a live Profile or Driver.
type PreparedCase struct {
	source  *testpilotspb.Case
	program *execution.PreparedProgram
	factory execution.MonitorFactory
	// contract is factory's own prepared Contract, which Evaluate replays a closed Run through.
	contract *verification.PreparedContract
	identity DriverIdentity
}

// Prepare snapshots and admits a Case without Driver I/O. Rejections expose
// *PreparationError through errors.As.
func Prepare(source *testpilotspb.Case, profile Profile) (*PreparedCase, error) {
	if ir.IsNil(profile) {
		return nil, preparationError(errors.New("Profile is required"), "profile")
	}
	// The snapshot is Prepare's one clone of the Profile; admission holds its values as given.
	spec := profile.Snapshot()
	if spec.Catalog == nil || spec.Catalog.catalog == nil {
		return nil, preparationError(errors.New("Profile catalog is required"), "profile.catalog")
	}
	fingerprint, err := spec.BindingFingerprint()
	if err != nil {
		return nil, preparationError(err, "profile.environment_bindings")
	}
	// The one boundary copy: execution cannot import the facade, and ProfileSpec's public shape stays.
	policy := execution.Profile{Identity: spec.Identity, CatalogIdentity: spec.Catalog.Identity(), Roles: spec.Roles, Opcodes: spec.Opcodes, CommandTypes: spec.CommandTypes, EnvironmentBindings: spec.EnvironmentBindings, Limits: spec.ProgramLimits, InstructionDefaults: spec.InstructionDefaults, BoundScale: spec.BoundScale, DeliveryControl: spec.DeliveryControl, Configuration: configurationByKey(spec.Configuration)}
	program, err := execution.Prepare(source, spec.Catalog.catalog, policy)
	if err != nil {
		return nil, preparationError(err, "program")
	}
	contract, err := verification.Prepare(source.Contract, spec.Catalog.catalog, program.View(), spec.ContractLimits, spec.CorrelatedLimits)
	if err != nil {
		return nil, preparationError(err, "contract")
	}
	return &PreparedCase{source: proto.CloneOf(source), program: program, factory: contract, contract: contract, identity: DriverIdentity{Profile: policy.Identity, Catalog: policy.CatalogIdentity, Bindings: fingerprint}}, nil
}

// configurationByKey is the Profile's configuration by lower-case key, which BindingFingerprint has
// already admitted: each key valid and once.
func configurationByKey(values []ConfigurationValue) map[string]string {
	byKey := make(map[string]string, len(values))
	for _, value := range values {
		byKey[strings.ToLower(value.Key)] = value.Value
	}
	return byKey
}

func (p *PreparedCase) Snapshot() *testpilotspb.Case { return proto.CloneOf(p.source) }
func (p *PreparedCase) Identity() DriverIdentity     { return p.identity }

func (p *PreparedCase) preflight(ctx context.Context, driver Driver) (execution.Driver, execution.Monitor, error) {
	if p == nil || p.program == nil || ir.IsNil(ctx) || ir.IsNil(driver) || ir.IsNil(p.factory) {
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
		return nil, nil, errors.New("Driver Profile, catalog or binding identity changed")
	}
	if err := driver.Validate(ctx, PreparedProgram{program: p.program}); err != nil {
		return nil, nil, err
	}
	monitor, err := execution.NewMonitor(ctx, p.factory, p.program.View())
	if err != nil {
		return nil, nil, err
	}
	return driverAdapter{driver: driver}, monitor, nil
}
