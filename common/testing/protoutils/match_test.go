package protoutils_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.temporal.io/server/api/matchingservice/v1"
	"go.temporal.io/server/common/testing/protoutils"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestMatchesPartialProto(t *testing.T) {
	tests := []struct {
		name     string
		partial  *matchingservice.AddWorkflowTaskRequest
		actual   *matchingservice.AddWorkflowTaskRequest
		expected bool
	}{
		{
			name: "exact match - single field",
			partial: &matchingservice.AddWorkflowTaskRequest{
				Speculative: true,
			},
			actual: &matchingservice.AddWorkflowTaskRequest{
				Speculative: true,
			},
			expected: true,
		},
		{
			name: "no match - different value",
			partial: &matchingservice.AddWorkflowTaskRequest{
				Speculative: true,
			},
			actual: &matchingservice.AddWorkflowTaskRequest{
				Speculative: false,
			},
			expected: false,
		},
		{
			name: "match with extra fields in actual",
			partial: &matchingservice.AddWorkflowTaskRequest{
				Speculative: true,
			},
			actual: &matchingservice.AddWorkflowTaskRequest{
				NamespaceId: "test-namespace",
				Speculative: true,
				Stamp:       42,
			},
			expected: true,
		},
		{
			name:    "match - empty partial matches anything",
			partial: &matchingservice.AddWorkflowTaskRequest{},
			actual: &matchingservice.AddWorkflowTaskRequest{
				NamespaceId: "test-namespace",
				Speculative: true,
			},
			expected: true,
		},
		{
			name: "match - multiple fields",
			partial: &matchingservice.AddWorkflowTaskRequest{
				NamespaceId: "test-namespace",
				Speculative: true,
			},
			actual: &matchingservice.AddWorkflowTaskRequest{
				NamespaceId: "test-namespace",
				Speculative: true,
				Stamp:       42,
			},
			expected: true,
		},
		{
			name: "no match - one field different",
			partial: &matchingservice.AddWorkflowTaskRequest{
				NamespaceId: "test-namespace",
				Speculative: true,
			},
			actual: &matchingservice.AddWorkflowTaskRequest{
				NamespaceId: "different-namespace",
				Speculative: true,
			},
			expected: false,
		},
		{
			name: "match - message field",
			partial: &matchingservice.AddWorkflowTaskRequest{
				ScheduleToStartTimeout: durationpb.New(10),
			},
			actual: &matchingservice.AddWorkflowTaskRequest{
				ScheduleToStartTimeout: durationpb.New(10),
			},
			expected: true,
		},
		{
			name: "no match - different message field",
			partial: &matchingservice.AddWorkflowTaskRequest{
				ScheduleToStartTimeout: durationpb.New(10),
			},
			actual: &matchingservice.AddWorkflowTaskRequest{
				ScheduleToStartTimeout: durationpb.New(20),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := protoutils.MatchesPartialProto(tt.partial, tt.actual)
			assert.Equal(t, tt.expected, result, "MatchesPartialProto result mismatch")
		})
	}
}
