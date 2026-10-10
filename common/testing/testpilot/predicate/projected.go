// Package predicate evaluates neutral predicates under an explicit descriptor catalog.
package predicate

import (
	"context"

	celpb "cel.dev/expr"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/types/descriptorpb"
)

// EvaluateProjected checks a Boolean predicate using the same restricted engine as Testpilot.
// The explicit catalog owns the projected value's descriptors. Work includes input snapshots,
// native CEL cost and result ownership, and errors never represent a successful match.
func EvaluateProjected(ctx context.Context, source *testpilotspb.Expression, projected *celpb.Value, schema *testpilotspb.ValueType, descriptors *descriptorpb.FileDescriptorSet, work int64) (*celpb.Value, int64, error) {
	catalog, err := ir.NewCatalog(descriptors)
	if err != nil {
		return nil, 0, err
	}
	typ, err := catalog.BindType(schema)
	if err != nil {
		return nil, 0, err
	}
	boolean, err := catalog.BindType(&testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Scalar{Scalar: &testpilotspb.ScalarType{Kind: testpilotspb.SCALAR_KIND_BOOLEAN}}}}})
	if err != nil {
		return nil, 0, err
	}
	reference := ir.Reference{Kind: ir.ProjectedValueReference}
	bound, err := catalog.BindExpression(ir.Site{Context: ir.EvidenceLiftContext, Path: "evidence.guard"}, source, &boolean, map[ir.Reference]ir.Binding{reference: {Type: typ, Available: true}}, ir.DefaultLimits())
	if err != nil {
		return nil, 0, err
	}
	return bound.Evaluate(ctx, func(ir.Reference) *celpb.Value { return projected }, work)
}
