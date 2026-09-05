package umpire_test

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type alternateHost struct{}

func (alternateHost) Identity(context.Context) (umpire.HostIdentity, error) {
	return umpire.HostIdentity{Profile: "alternate"}, nil
}
func (alternateHost) Open(ctx context.Context, _ string, program umpire.PreparedProgram) (umpire.Session, error) {
	for _, entry := range program.Entrypoints() {
		nodes := entry.Instructions()
		for _, index := range entry.Order() {
			node := nodes[index]
			if guard := node.Guard(); guard != nil {
				_, _, err := guard.Evaluate(ctx, func(ref umpire.ValueReference) *umpirespb.Value {
					switch ref.Kind {
					case umpire.SlotReference, umpire.OutcomeReference:
						return &umpirespb.Value{Value: &umpirespb.Value_BoolValue{BoolValue: true}}
					default:
						return nil
					}
				}, 1000)
				if err != nil {
					return nil, err
				}
			}
		}
	}
	return alternateSession{}, nil
}

type alternateSession struct{}

func (alternateSession) Reserve(context.Context, umpire.ReservationRequest) ([]umpire.ReservationHandle, error) {
	return nil, errors.New("no reservation")
}
func (alternateSession) InvokeRPC(context.Context, umpire.Coordinate, string, protoreflect.MethodDescriptor, proto.Message) (umpire.EffectHandle, error) {
	return nil, errors.New("no endpoint")
}
func (alternateSession) CompleteNexusOperation(context.Context, umpire.Coordinate, umpire.OpaqueCapability, *umpirespb.Value) (umpire.EffectHandle, error) {
	return nil, errors.New("no capability")
}
func (alternateSession) Bridge(context.Context) (umpire.CapabilityBridge, error) {
	return nil, errors.New("no capability")
}
func (alternateSession) Quarantine(context.Context, umpire.EffectHandle) error            { return nil }
func (alternateSession) Close(context.Context) error                                      { return nil }
func (alternateSession) Diagnose(context.Context, string, *umpirespb.RunDiagnostic) error { return nil }

var _ umpire.Host = alternateHost{}
var _ umpire.Session = alternateSession{}

func TestRootImportBoundary(t *testing.T) {
	command := exec.CommandContext(t.Context(), "go", "list", "-tags", "test_dep", "-deps", "go.temporal.io/server/tools/umpire")
	output, err := command.Output()
	require.NoError(t, err)
	for _, dependency := range strings.Fields(string(output)) {
		for _, prefix := range []string{"go.temporal.io/server/tools/umpire/temporal", "go.temporal.io/sdk"} {
			require.False(t, dependency == prefix || strings.HasPrefix(dependency, prefix+"/"), "forbidden dependency: %s", dependency)
		}
	}
}
