package testsupport

import (
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/cel"
)

func LiteralExpressions(values []*testpilotspb.NamedValue) []*testpilotspb.NamedExpression {
	result := make([]*testpilotspb.NamedExpression, len(values))
	for index, value := range values {
		if value != nil {
			result[index] = &testpilotspb.NamedExpression{FieldId: value.FieldId, Value: cel.Literal(value.Value)}
		}
	}
	return result
}
