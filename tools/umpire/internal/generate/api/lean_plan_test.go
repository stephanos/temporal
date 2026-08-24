package api

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLeanPlanAllocatesCollidingMembersFromStableProtobufIdentity(t *testing.T) {
	t.Parallel()

	fields := []fieldProjection{
		{FullName: "temporal.api.test.v1.Collision.foo_bar", Name: "foo_bar", Number: 2, Kind: "string"},
		{FullName: "temporal.api.test.v1.Collision.fooBar", Name: "fooBar", Number: 3, Kind: "string"},
		{FullName: "temporal.api.test.v1.Collision.foo_bar2", Name: "foo_bar2", Number: 4, Kind: "string"},
		{FullName: "temporal.api.test.v1.Collision.not_set", Name: "not_set", Number: 5, Kind: "string", Oneof: "choice"},
	}
	document := projection{Messages: []messageProjection{{
		FullName: "temporal.api.test.v1.Collision", Name: "Collision", Package: "temporal.api.test.v1",
		Source: sourcePublic, Fields: fields,
		Oneofs: []oneofProjection{{
			FullName: "temporal.api.test.v1.Collision.choice", Name: "choice",
			FieldNames: []string{"temporal.api.test.v1.Collision.not_set"},
		}},
	}}}
	first, err := buildLeanPlan(document)
	require.NoError(t, err)
	slices.Reverse(document.Messages[0].Fields)
	second, err := buildLeanPlan(document)
	require.NoError(t, err)

	for _, fullName := range []string{
		"temporal.api.test.v1.Collision.foo_bar",
		"temporal.api.test.v1.Collision.fooBar",
		"temporal.api.test.v1.Collision.foo_bar2",
		"temporal.api.test.v1.Collision.not_set",
	} {
		require.Equal(t, first.fields[fullName].Name, second.fields[fullName].Name)
	}
	require.Equal(t, "fooBar2", first.fields["temporal.api.test.v1.Collision.foo_bar2"].Name)
	require.NotEqual(t,
		first.fields["temporal.api.test.v1.Collision.foo_bar"].Name,
		first.fields["temporal.api.test.v1.Collision.fooBar"].Name,
	)
	require.Equal(t, "notSet5", first.fields["temporal.api.test.v1.Collision.not_set"].Name)
}

func TestLeanPlanPreservesNestedOwnershipAndQualifiesCrossPackageReferences(t *testing.T) {
	t.Parallel()

	document := projection{
		Messages: []messageProjection{
			{
				FullName: "example.common.v1.External", Name: "External", Package: "example.common.v1",
				Source: sourceExternal, Fields: []fieldProjection{}, Oneofs: []oneofProjection{},
			},
			{
				FullName: "temporal.api.test.v1.Outer", Name: "Outer", Package: "temporal.api.test.v1",
				Source: sourcePublic, Fields: []fieldProjection{}, Oneofs: []oneofProjection{},
			},
			{
				FullName: "temporal.api.test.v1.Outer.Inner", Name: "Inner", Package: "temporal.api.test.v1",
				Parent: "temporal.api.test.v1.Outer", Source: sourcePublic,
				Fields: []fieldProjection{}, Oneofs: []oneofProjection{},
			},
			{
				FullName: "temporal.api.test.v1.Holder", Name: "Holder", Package: "temporal.api.test.v1",
				Source: sourcePublic,
				Fields: []fieldProjection{
					{FullName: "temporal.api.test.v1.Holder.local", Name: "local", Number: 1, Kind: "message", TypeName: "temporal.api.test.v1.Outer.Inner", Presence: true},
					{FullName: "temporal.api.test.v1.Holder.external", Name: "external", Number: 2, Kind: "message", TypeName: "example.common.v1.External", Presence: true},
				},
				Oneofs: []oneofProjection{},
			},
		},
	}
	plan, err := buildLeanPlan(document)
	require.NoError(t, err)
	require.Equal(t, "Temporal.Api.Test.V1.Outer.Inner", plan.names["temporal.api.test.v1.Outer.Inner"].String())
	require.Equal(t, "Option Outer.Inner", renderLeanType(plan.fields["temporal.api.test.v1.Holder.local"].Type))
	require.Equal(t, "Option Example.Common.V1.External", renderLeanType(plan.fields["temporal.api.test.v1.Holder.external"].Type))

	generated := string(renderTypes(plan))
	require.Contains(t, generated, "namespace Temporal.Api.Test.V1")
	require.Contains(t, generated, "structure Outer.Inner where")
	require.Contains(t, generated, "local : Option Outer.Inner")
	require.Contains(t, generated, "external : Option Example.Common.V1.External")
}

func TestLeanPlanUsesMessageReferencesOnlyWithinRecursiveComponents(t *testing.T) {
	t.Parallel()

	document := projection{Messages: []messageProjection{
		messageProjectionWithReference("A", "B"),
		messageProjectionWithReference("B", "A"),
		messageProjectionWithReference("C", "A"),
	}}
	plan, err := buildLeanPlan(document)
	require.NoError(t, err)

	require.True(t, plan.fields["temporal.api.test.v1.A.value"].Recursive)
	require.True(t, plan.fields["temporal.api.test.v1.B.value"].Recursive)
	require.False(t, plan.fields["temporal.api.test.v1.C.value"].Recursive)
	require.Equal(t, "Option Temporal.Proto.MessageRef", renderLeanType(plan.fields["temporal.api.test.v1.A.value"].Type))
	require.Equal(t, "Option A", renderLeanType(plan.fields["temporal.api.test.v1.C.value"].Type))

	order := make(map[string]int, len(plan.Messages))
	for index, message := range plan.Messages {
		order[message.Projection.Name] = index
	}
	require.Less(t, order["A"], order["C"])
}

func TestLeanPlanRejectsUnknownNamedTypesWithFieldContext(t *testing.T) {
	t.Parallel()

	document := projection{Messages: []messageProjection{{
		FullName: "temporal.api.test.v1.Request", Name: "Request", Package: "temporal.api.test.v1",
		Source: sourcePublic,
		Fields: []fieldProjection{{
			FullName: "temporal.api.test.v1.Request.missing", Name: "missing", Number: 1,
			Kind: "message", TypeName: "temporal.api.missing.v1.Unknown", Presence: true,
		}},
		Oneofs: []oneofProjection{},
	}}}

	_, err := buildLeanPlan(document)
	require.ErrorContains(t, err, "temporal.api.test.v1.Request.missing")
	require.ErrorContains(t, err, "temporal.api.missing.v1.Unknown")
}

func TestLeanPlanValidationRejectsDeclarationsBeforeTheirDependencies(t *testing.T) {
	t.Parallel()

	document := projection{Messages: []messageProjection{
		messageProjectionWithReference("A", "B"),
		messageProjectionWithReference("B", "A"),
		messageProjectionWithReference("C", "A"),
	}}
	plan, err := buildLeanPlan(document)
	require.NoError(t, err)
	cIndex := slices.IndexFunc(plan.Messages, func(message leanMessagePlan) bool {
		return message.Projection.Name == "C"
	})
	require.NotEqual(t, -1, cIndex)
	plan.Messages[0], plan.Messages[cIndex] = plan.Messages[cIndex], plan.Messages[0]

	err = validateLeanPlan(document, plan)
	require.ErrorContains(t, err, "precedes dependency")
}

func messageProjectionWithReference(name, target string) messageProjection {
	fullName := "temporal.api.test.v1." + name
	return messageProjection{
		FullName: fullName, Name: name, Package: "temporal.api.test.v1", Source: sourcePublic,
		Fields: []fieldProjection{{
			FullName: fullName + ".value", Name: "value", Number: 1, Kind: "message",
			TypeName: "temporal.api.test.v1." + target, Presence: true,
		}},
		Oneofs: []oneofProjection{},
	}
}
