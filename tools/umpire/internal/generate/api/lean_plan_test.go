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

func TestLeanPlanDisambiguatesPackageAndDeclarationCollisions(t *testing.T) {
	t.Parallel()

	document := projection{Messages: []messageProjection{
		{FullName: "foo_bar.v1.Record", Name: "Record", Package: "foo_bar.v1", Source: sourceExternal},
		{FullName: "fooBar.v1.Record", Name: "Record", Package: "fooBar.v1", Source: sourceExternal},
		{FullName: "temporal.api.test.v1.Foo_Bar", Name: "Foo_Bar", Package: "temporal.api.test.v1", Source: sourcePublic},
		{FullName: "temporal.api.test.v1.FooBar", Name: "FooBar", Package: "temporal.api.test.v1", Source: sourcePublic},
		{FullName: "temporal.api.test.v1.Outer", Name: "Outer", Package: "temporal.api.test.v1", Source: sourcePublic},
		{FullName: "temporal.api.test.v1.Outer.Foo_Bar", Name: "Foo_Bar", Package: "temporal.api.test.v1", Parent: "temporal.api.test.v1.Outer", Source: sourcePublic},
		{FullName: "temporal.api.test.v1.Outer.FooBar", Name: "FooBar", Package: "temporal.api.test.v1", Parent: "temporal.api.test.v1.Outer", Source: sourcePublic},
	}}
	first, err := buildLeanPlan(document)
	require.NoError(t, err)
	slices.Reverse(document.Messages)
	second, err := buildLeanPlan(document)
	require.NoError(t, err)

	identities := []string{
		"foo_bar.v1.Record",
		"fooBar.v1.Record",
		"temporal.api.test.v1.Foo_Bar",
		"temporal.api.test.v1.FooBar",
		"temporal.api.test.v1.Outer.Foo_Bar",
		"temporal.api.test.v1.Outer.FooBar",
	}
	for _, identity := range identities {
		require.Equal(t, first.names[identity], second.names[identity])
	}
	require.NotEqual(t, first.names[identities[0]], first.names[identities[1]])
	require.NotEqual(t, first.names[identities[2]], first.names[identities[3]])
	require.NotEqual(t, first.names[identities[4]], first.names[identities[5]])
}

func TestLeanPlanDisambiguatesEnumValuesAndMethods(t *testing.T) {
	t.Parallel()

	document := projection{
		Enums: []enumProjection{{
			FullName: "temporal.api.test.v1.State", Name: "State", Package: "temporal.api.test.v1", Source: sourcePublic,
			Values: []enumValueProjection{
				{FullName: "temporal.api.test.v1.FOO_BAR", Name: "FOO_BAR", Number: 1},
				{FullName: "temporal.api.test.v1.FooBar", Name: "FooBar", Number: 2},
			},
		}},
		Messages: []messageProjection{
			{FullName: "temporal.api.test.v1.Request", Name: "Request", Package: "temporal.api.test.v1", Source: sourcePublic},
			{FullName: "temporal.api.test.v1.Response", Name: "Response", Package: "temporal.api.test.v1", Source: sourcePublic},
		},
		Services: []serviceProjection{{
			FullName: "temporal.api.test.v1.TestService", Name: "TestService", Package: "temporal.api.test.v1", Source: sourcePublic,
			Methods: []methodProjection{
				{FullName: "temporal.api.test.v1.TestService.Do_Thing", Name: "Do_Thing", InputType: "temporal.api.test.v1.Request", OutputType: "temporal.api.test.v1.Response"},
				{FullName: "temporal.api.test.v1.TestService.DoThing", Name: "DoThing", InputType: "temporal.api.test.v1.Request", OutputType: "temporal.api.test.v1.Response"},
			},
		}},
	}
	first, err := buildLeanPlan(document)
	require.NoError(t, err)
	slices.Reverse(document.Enums[0].Values)
	slices.Reverse(document.Services[0].Methods)
	second, err := buildLeanPlan(document)
	require.NoError(t, err)

	require.NotEqual(t, first.Enums[0].Values[0].Name, first.Enums[0].Values[1].Name)
	for _, fullName := range []string{"temporal.api.test.v1.FOO_BAR", "temporal.api.test.v1.FooBar"} {
		require.Equal(t, enumValueName(first, fullName), enumValueName(second, fullName))
	}
	for _, fullName := range []string{
		"temporal.api.test.v1.TestService.Do_Thing",
		"temporal.api.test.v1.TestService.DoThing",
	} {
		require.Equal(t, methodName(first, fullName), methodName(second, fullName))
	}
	require.NotEqual(t,
		methodName(first, "temporal.api.test.v1.TestService.Do_Thing"),
		methodName(first, "temporal.api.test.v1.TestService.DoThing"),
	)
}

func TestLeanPlanSeparatesPackageNamespacesFromDeclarationNamespaces(t *testing.T) {
	t.Parallel()

	document := projection{Messages: []messageProjection{
		{FullName: "foo.Bar", Name: "Bar", Package: "foo", Source: sourceExternal},
		{FullName: "foo.Bar.Record", Name: "Record", Package: "foo", Parent: "foo.Bar", Source: sourceExternal},
		{FullName: "foo.bar.Record", Name: "Record", Package: "foo.bar", Source: sourceExternal},
	}}
	plan, err := buildLeanPlan(document)
	require.NoError(t, err)

	require.NotEqual(t, plan.names["foo.Bar.Record"], plan.names["foo.bar.Record"])
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
				FullName: "temporal.api.test.v1.sub.External", Name: "External", Package: "temporal.api.test.v1.sub",
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
					{FullName: "temporal.api.test.v1.Holder.subpackage", Name: "subpackage", Number: 3, Kind: "message", TypeName: "temporal.api.test.v1.sub.External", Presence: true},
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
	require.Equal(t, "Option Temporal.Api.Test.V1.Sub.External", renderLeanType(plan.fields["temporal.api.test.v1.Holder.subpackage"].Type))

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
	slices.Reverse(document.Messages)
	permuted, err := buildLeanPlan(document)
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
	require.Equal(t, plannedMessageNames(plan), plannedMessageNames(permuted))
	for _, fullName := range []string{
		"temporal.api.test.v1.A.value",
		"temporal.api.test.v1.B.value",
		"temporal.api.test.v1.C.value",
	} {
		require.Equal(t, plan.fields[fullName].Recursive, permuted.fields[fullName].Recursive)
	}
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

func TestLeanPlanRejectsFieldsWhoseOneofIsMissing(t *testing.T) {
	t.Parallel()

	document := projection{Messages: []messageProjection{{
		FullName: "temporal.api.test.v1.Request", Name: "Request", Package: "temporal.api.test.v1",
		Source: sourcePublic,
		Fields: []fieldProjection{{
			FullName: "temporal.api.test.v1.Request.value", Name: "value", Number: 1,
			Kind: "string", Oneof: "choice",
		}},
	}}}

	_, err := buildLeanPlan(document)
	require.ErrorContains(t, err, "temporal.api.test.v1.Request.value")
	require.ErrorContains(t, err, "choice")
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

func TestLeanPlanValidationRejectsIncompleteModuleImports(t *testing.T) {
	t.Parallel()

	plan, err := buildLeanPlan(projection{})
	require.NoError(t, err)
	plan.Sources[0].CatalogModule.Imports = nil

	err = validateLeanPlan(projection{}, plan)
	require.ErrorContains(t, err, "incomplete modules")
}

func TestLeanPlanValidationRejectsIncompleteNamespaceOwnership(t *testing.T) {
	t.Parallel()

	document := projection{Messages: []messageProjection{{
		FullName: "temporal.api.test.v1.Request", Name: "Request", Package: "temporal.api.test.v1", Source: sourcePublic,
	}}}
	plan, err := buildLeanPlan(document)
	require.NoError(t, err)
	plan.Namespaces = nil

	err = validateLeanPlan(document, plan)
	require.ErrorContains(t, err, "namespace ownership")
}

func TestLeanPlanValidationRejectsDuplicateSourceOwnership(t *testing.T) {
	t.Parallel()

	document := projection{Messages: []messageProjection{
		{FullName: "temporal.api.test.v1.Public", Name: "Public", Package: "temporal.api.test.v1", Source: sourcePublic},
		{FullName: "temporal.server.api.test.v1.Internal", Name: "Internal", Package: "temporal.server.api.test.v1", Source: sourceInternal},
	}}
	plan, err := buildLeanPlan(document)
	require.NoError(t, err)
	public := slices.IndexFunc(plan.Sources, func(source leanSourcePlan) bool { return source.Source == sourcePublic })
	internal := slices.IndexFunc(plan.Sources, func(source leanSourcePlan) bool { return source.Source == sourceInternal })
	require.NotEqual(t, -1, public)
	require.NotEqual(t, -1, internal)
	plan.Sources[internal].Messages[0] = plan.Sources[public].Messages[0]

	err = validateLeanPlan(document, plan)
	require.ErrorContains(t, err, "source partition")
}

func enumValueName(plan leanPlan, fullName string) string {
	for _, enum := range plan.Enums {
		for _, value := range enum.Values {
			if value.Projection.FullName == fullName {
				return value.Name
			}
		}
	}
	return ""
}

func methodName(plan leanPlan, fullName string) string {
	for _, service := range plan.services {
		for _, method := range service.Methods {
			if method.Projection.FullName == fullName {
				return method.Name
			}
		}
	}
	return ""
}

func plannedMessageNames(plan leanPlan) []string {
	result := make([]string, 0, len(plan.Messages))
	for _, message := range plan.Messages {
		result = append(result, message.Projection.FullName)
	}
	return result
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
