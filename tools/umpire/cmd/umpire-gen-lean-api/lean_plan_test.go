package main

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLeanPlanAllocatesCollidingMembersFromStableProtobufIdentity(t *testing.T) {
	t.Parallel()

	fields := []fieldProjection{
		{FullName: "fixture.messaging.test.v1.Collision.foo_bar", Name: "foo_bar", Number: 2, Kind: "string"},
		{FullName: "fixture.messaging.test.v1.Collision.fooBar", Name: "fooBar", Number: 3, Kind: "string"},
		{FullName: "fixture.messaging.test.v1.Collision.foo_bar2", Name: "foo_bar2", Number: 4, Kind: "string"},
		{FullName: "fixture.messaging.test.v1.Collision.not_set", Name: "not_set", Number: 5, Kind: "string", Oneof: "choice"},
	}
	document := projection{Messages: []messageProjection{{
		FullName: "fixture.messaging.test.v1.Collision", Name: "Collision", Package: "fixture.messaging.test.v1",
		Fields: fields,
		Oneofs: []oneofProjection{{
			FullName: "fixture.messaging.test.v1.Collision.choice", Name: "choice",
			FieldNames: []string{"fixture.messaging.test.v1.Collision.not_set"},
		}},
	}}}
	first, err := buildLeanPlan(document, testGenerationConfig("Fixture"))
	require.NoError(t, err)
	slices.Reverse(document.Messages[0].Fields)
	second, err := buildLeanPlan(document, testGenerationConfig("Fixture"))
	require.NoError(t, err)

	for _, fullName := range []string{
		"fixture.messaging.test.v1.Collision.foo_bar",
		"fixture.messaging.test.v1.Collision.fooBar",
		"fixture.messaging.test.v1.Collision.foo_bar2",
		"fixture.messaging.test.v1.Collision.not_set",
	} {
		require.Equal(t, first.fields[fullName].Name, second.fields[fullName].Name)
	}
	require.Equal(t, "fooBar2", first.fields["fixture.messaging.test.v1.Collision.foo_bar2"].Name)
	require.NotEqual(t,
		first.fields["fixture.messaging.test.v1.Collision.foo_bar"].Name,
		first.fields["fixture.messaging.test.v1.Collision.fooBar"].Name,
	)
	require.Equal(t, "notSet5", first.fields["fixture.messaging.test.v1.Collision.not_set"].Name)
}

func TestLeanPlanDisambiguatesPackageAndDeclarationCollisions(t *testing.T) {
	t.Parallel()

	document := projection{Messages: []messageProjection{
		{FullName: "foo_bar.v1.Record", Name: "Record", Package: "foo_bar.v1"},
		{FullName: "fooBar.v1.Record", Name: "Record", Package: "fooBar.v1"},
		{FullName: "fixture.messaging.test.v1.Foo_Bar", Name: "Foo_Bar", Package: "fixture.messaging.test.v1"},
		{FullName: "fixture.messaging.test.v1.FooBar", Name: "FooBar", Package: "fixture.messaging.test.v1"},
		{FullName: "fixture.messaging.test.v1.Outer", Name: "Outer", Package: "fixture.messaging.test.v1"},
		{FullName: "fixture.messaging.test.v1.Outer.Foo_Bar", Name: "Foo_Bar", Package: "fixture.messaging.test.v1", Parent: "fixture.messaging.test.v1.Outer"},
		{FullName: "fixture.messaging.test.v1.Outer.FooBar", Name: "FooBar", Package: "fixture.messaging.test.v1", Parent: "fixture.messaging.test.v1.Outer"},
	}}
	first, err := buildLeanPlan(document, testGenerationConfig("Fixture"))
	require.NoError(t, err)
	slices.Reverse(document.Messages)
	second, err := buildLeanPlan(document, testGenerationConfig("Fixture"))
	require.NoError(t, err)

	identities := []string{
		"foo_bar.v1.Record",
		"fooBar.v1.Record",
		"fixture.messaging.test.v1.Foo_Bar",
		"fixture.messaging.test.v1.FooBar",
		"fixture.messaging.test.v1.Outer.Foo_Bar",
		"fixture.messaging.test.v1.Outer.FooBar",
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
			FullName: "fixture.messaging.test.v1.State", Name: "State", Package: "fixture.messaging.test.v1",
			Values: []enumValueProjection{
				{FullName: "fixture.messaging.test.v1.FOO_BAR", Name: "FOO_BAR", Number: 1},
				{FullName: "fixture.messaging.test.v1.FooBar", Name: "FooBar", Number: 2},
			},
		}},
		Messages: []messageProjection{
			{FullName: "fixture.messaging.test.v1.Request", Name: "Request", Package: "fixture.messaging.test.v1"},
			{FullName: "fixture.messaging.test.v1.Response", Name: "Response", Package: "fixture.messaging.test.v1"},
		},
		Services: []serviceProjection{{
			FullName: "fixture.messaging.test.v1.TestService", Name: "TestService", Package: "fixture.messaging.test.v1",
			Methods: []methodProjection{
				{FullName: "fixture.messaging.test.v1.TestService.Do_Thing", Name: "Do_Thing", InputType: "fixture.messaging.test.v1.Request", OutputType: "fixture.messaging.test.v1.Response"},
				{FullName: "fixture.messaging.test.v1.TestService.DoThing", Name: "DoThing", InputType: "fixture.messaging.test.v1.Request", OutputType: "fixture.messaging.test.v1.Response"},
			},
		}},
	}
	first, err := buildLeanPlan(document, testGenerationConfig("Fixture"))
	require.NoError(t, err)
	slices.Reverse(document.Enums[0].Values)
	slices.Reverse(document.Services[0].Methods)
	second, err := buildLeanPlan(document, testGenerationConfig("Fixture"))
	require.NoError(t, err)

	require.NotEqual(t, first.Enums[0].Values[0].Name, first.Enums[0].Values[1].Name)
	for _, fullName := range []string{"fixture.messaging.test.v1.FOO_BAR", "fixture.messaging.test.v1.FooBar"} {
		require.Equal(t, enumValueName(first, fullName), enumValueName(second, fullName))
	}
	for _, fullName := range []string{
		"fixture.messaging.test.v1.TestService.Do_Thing",
		"fixture.messaging.test.v1.TestService.DoThing",
	} {
		require.Equal(t, methodName(first, fullName), methodName(second, fullName))
	}
	require.NotEqual(t,
		methodName(first, "fixture.messaging.test.v1.TestService.Do_Thing"),
		methodName(first, "fixture.messaging.test.v1.TestService.DoThing"),
	)
}

func TestLeanPlanSeparatesPackageNamespacesFromDeclarationNamespaces(t *testing.T) {
	t.Parallel()

	document := projection{Messages: []messageProjection{
		{FullName: "foo.Bar", Name: "Bar", Package: "foo"},
		{FullName: "foo.Bar.Record", Name: "Record", Package: "foo", Parent: "foo.Bar"},
		{FullName: "foo.bar.Record", Name: "Record", Package: "foo.bar"},
	}}
	plan, err := buildLeanPlan(document, testGenerationConfig("Fixture"))
	require.NoError(t, err)

	require.NotEqual(t, plan.names["foo.Bar.Record"], plan.names["foo.bar.Record"])
}

func TestLeanPlanPreservesNestedOwnershipAndQualifiesCrossPackageReferences(t *testing.T) {
	t.Parallel()

	document := projection{
		Messages: []messageProjection{
			{
				FullName: "example.common.v1.External", Name: "External", Package: "example.common.v1",
				Fields: []fieldProjection{}, Oneofs: []oneofProjection{},
			},
			{
				FullName: "fixture.messaging.test.v1.sub.External", Name: "External", Package: "fixture.messaging.test.v1.sub",
				Fields: []fieldProjection{}, Oneofs: []oneofProjection{},
			},
			{
				FullName: "fixture.messaging.test.v1.Outer", Name: "Outer", Package: "fixture.messaging.test.v1",
				Fields: []fieldProjection{}, Oneofs: []oneofProjection{},
			},
			{
				FullName: "fixture.messaging.test.v1.Outer.Inner", Name: "Inner", Package: "fixture.messaging.test.v1",
				Parent: "fixture.messaging.test.v1.Outer",
				Fields: []fieldProjection{}, Oneofs: []oneofProjection{},
			},
			{
				FullName: "fixture.messaging.test.v1.Holder", Name: "Holder", Package: "fixture.messaging.test.v1",
				Fields: []fieldProjection{
					{FullName: "fixture.messaging.test.v1.Holder.local", Name: "local", Number: 1, Kind: "message", TypeName: "fixture.messaging.test.v1.Outer.Inner", Presence: true},
					{FullName: "fixture.messaging.test.v1.Holder.external", Name: "external", Number: 2, Kind: "message", TypeName: "example.common.v1.External", Presence: true},
					{FullName: "fixture.messaging.test.v1.Holder.subpackage", Name: "subpackage", Number: 3, Kind: "message", TypeName: "fixture.messaging.test.v1.sub.External", Presence: true},
				},
				Oneofs: []oneofProjection{},
			},
		},
	}
	plan, err := buildLeanPlan(document, testGenerationConfig("Fixture"))
	require.NoError(t, err)
	require.Equal(t, "Fixture.Messaging.Test.V1.Outer.Inner", plan.names["fixture.messaging.test.v1.Outer.Inner"].String())
	require.Equal(t, "Option Outer.Inner", renderLeanType(plan.fields["fixture.messaging.test.v1.Holder.local"].Type))
	require.Equal(t, "Option Example.Common.V1.External", renderLeanType(plan.fields["fixture.messaging.test.v1.Holder.external"].Type))
	require.Equal(t, "Option Fixture.Messaging.Test.V1.Sub.External", renderLeanType(plan.fields["fixture.messaging.test.v1.Holder.subpackage"].Type))

	generated := string(renderTypes(plan))
	require.Contains(t, generated, "namespace Fixture.Messaging.Test.V1")
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
	plan, err := buildLeanPlan(document, testGenerationConfig("Fixture"))
	require.NoError(t, err)
	slices.Reverse(document.Messages)
	permuted, err := buildLeanPlan(document, testGenerationConfig("Fixture"))
	require.NoError(t, err)

	require.True(t, plan.fields["fixture.messaging.test.v1.A.value"].Recursive)
	require.True(t, plan.fields["fixture.messaging.test.v1.B.value"].Recursive)
	require.False(t, plan.fields["fixture.messaging.test.v1.C.value"].Recursive)
	require.Equal(t, "Option Fixture.API.Proto.MessageRef", renderLeanType(plan.fields["fixture.messaging.test.v1.A.value"].Type))
	require.Equal(t, "Option A", renderLeanType(plan.fields["fixture.messaging.test.v1.C.value"].Type))

	order := make(map[string]int, len(plan.Messages))
	for index, message := range plan.Messages {
		order[message.Projection.Name] = index
	}
	require.Less(t, order["A"], order["C"])
	require.Equal(t, plannedMessageNames(plan), plannedMessageNames(permuted))
	for _, fullName := range []string{
		"fixture.messaging.test.v1.A.value",
		"fixture.messaging.test.v1.B.value",
		"fixture.messaging.test.v1.C.value",
	} {
		require.Equal(t, plan.fields[fullName].Recursive, permuted.fields[fullName].Recursive)
	}
}

func TestLeanPlanRejectsUnknownNamedTypesWithFieldContext(t *testing.T) {
	t.Parallel()

	document := projection{Messages: []messageProjection{{
		FullName: "fixture.messaging.test.v1.Request", Name: "Request", Package: "fixture.messaging.test.v1",
		Fields: []fieldProjection{{
			FullName: "fixture.messaging.test.v1.Request.missing", Name: "missing", Number: 1,
			Kind: "message", TypeName: "fixture.messaging.missing.v1.Unknown", Presence: true,
		}},
		Oneofs: []oneofProjection{},
	}}}

	_, err := buildLeanPlan(document, testGenerationConfig("Fixture"))
	require.ErrorContains(t, err, "fixture.messaging.test.v1.Request.missing")
	require.ErrorContains(t, err, "fixture.messaging.missing.v1.Unknown")
}

func TestLeanPlanRejectsFieldsWhoseOneofIsMissing(t *testing.T) {
	t.Parallel()

	document := projection{Messages: []messageProjection{{
		FullName: "fixture.messaging.test.v1.Request", Name: "Request", Package: "fixture.messaging.test.v1",
		Fields: []fieldProjection{{
			FullName: "fixture.messaging.test.v1.Request.value", Name: "value", Number: 1,
			Kind: "string", Oneof: "choice",
		}},
	}}}

	_, err := buildLeanPlan(document, testGenerationConfig("Fixture"))
	require.ErrorContains(t, err, "fixture.messaging.test.v1.Request.value")
	require.ErrorContains(t, err, "choice")
}

func TestLeanPlanValidationRejectsDeclarationsBeforeTheirDependencies(t *testing.T) {
	t.Parallel()

	document := projection{Messages: []messageProjection{
		messageProjectionWithReference("A", "B"),
		messageProjectionWithReference("B", "A"),
		messageProjectionWithReference("C", "A"),
	}}
	configuration := testGenerationConfig("Fixture")
	plan, err := buildLeanPlan(document, configuration)
	require.NoError(t, err)
	cIndex := slices.IndexFunc(plan.Messages, func(message leanMessagePlan) bool {
		return message.Projection.Name == "C"
	})
	require.NotEqual(t, -1, cIndex)
	plan.Messages[0], plan.Messages[cIndex] = plan.Messages[cIndex], plan.Messages[0]

	err = validateLeanPlan(document, plan, configuration)
	require.ErrorContains(t, err, "precedes dependency")
}

func TestLeanPlanValidationRejectsIncompleteModuleImports(t *testing.T) {
	t.Parallel()

	configuration := testGenerationConfig("Fixture")
	plan, err := buildLeanPlan(projection{}, configuration)
	require.NoError(t, err)
	plan.APIModule.Imports = nil

	err = validateLeanPlan(projection{}, plan, configuration)
	require.ErrorContains(t, err, "API module is incomplete")
}

func TestLeanPlanValidationRejectsIncompleteNamespaceOwnership(t *testing.T) {
	t.Parallel()

	document := projection{Messages: []messageProjection{{
		FullName: "fixture.messaging.test.v1.Request", Name: "Request", Package: "fixture.messaging.test.v1",
	}}}
	configuration := testGenerationConfig("Fixture")
	plan, err := buildLeanPlan(document, configuration)
	require.NoError(t, err)
	plan.Namespaces = nil

	err = validateLeanPlan(document, plan, configuration)
	require.ErrorContains(t, err, "namespace ownership")
}

func TestLeanPlanValidationRejectsChangedServiceOrder(t *testing.T) {
	t.Parallel()

	document := projection{Services: []serviceProjection{
		{FullName: "fixture.messaging.test.v1.Alpha", Name: "Alpha", Package: "fixture.messaging.test.v1"},
		{FullName: "fixture.messaging.test.v1.Beta", Name: "Beta", Package: "fixture.messaging.test.v1"},
	}}
	configuration := testGenerationConfig("Fixture")
	plan, err := buildLeanPlan(document, configuration)
	require.NoError(t, err)
	require.Equal(t, []string{"fixture.messaging.test.v1.Alpha", "fixture.messaging.test.v1.Beta"}, []string{
		plan.Services[0].Projection.FullName,
		plan.Services[1].Projection.FullName,
	})
	plan.Services[0], plan.Services[1] = plan.Services[1], plan.Services[0]

	err = validateLeanPlan(document, plan, configuration)
	require.ErrorContains(t, err, "service order")
}

func TestLeanPlanUsesConfiguredThreeModuleBoundary(t *testing.T) {
	configuration := testGenerationConfig("Acme.Model")
	document := projection{Messages: []messageProjection{{
		FullName: "example.v1.Record", Name: "Record", Package: "example.v1",
		Fields: []fieldProjection{{
			FullName: "example.v1.Record.payload", Name: "payload", Number: 1, Kind: "bytes",
		}},
	}}}

	plan, err := buildLeanPlan(document, configuration)
	require.NoError(t, err)
	require.Equal(t, leanModulePlan{Path: "Acme/Model/API/Proto.lean", Imports: []string{}}, plan.ProtoModule)
	require.Equal(t, leanModulePlan{Path: "Acme/Model/API/Types.lean", Imports: []string{"Acme.Model.API.Proto"}}, plan.TypesModule)
	require.Equal(t, leanModulePlan{Path: "Acme/Model/API.lean", Imports: []string{"Acme.Model.API.Proto", "Acme.Model.API.Types"}}, plan.APIModule)
	require.Equal(t, "Acme.Model.API.Proto.Bytes", renderLeanType(plan.fields["example.v1.Record.payload"].Type))
}

func TestLeanPlanRejectsDeclarationsCollidingWithGeneratedSupport(t *testing.T) {
	configuration := testGenerationConfig("Acme.Model")
	document := projection{Messages: []messageProjection{{
		FullName: "acme.model.a_p_i.proto.Bytes", Name: "Bytes", Package: "acme.model.a_p_i.proto",
	}}}

	_, err := buildLeanPlan(document, configuration)
	require.ErrorContains(t, err, `"acme.model.a_p_i.proto.Bytes"`)
	require.ErrorContains(t, err, `generated support declaration "Acme.Model.API.Proto.Bytes"`)
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
	for _, service := range plan.Services {
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
	fullName := "fixture.messaging.test.v1." + name
	return messageProjection{
		FullName: fullName, Name: name, Package: "fixture.messaging.test.v1",
		Fields: []fieldProjection{{
			FullName: fullName + ".value", Name: "value", Number: 1, Kind: "message",
			TypeName: "fixture.messaging.test.v1." + target, Presence: true,
		}},
		Oneofs: []oneofProjection{},
	}
}
