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
		{FullName: "fixture.messaging.test.v1.Collision.match", Name: "match", Number: 6, Kind: "string"},
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
		"fixture.messaging.test.v1.Collision.match",
	} {
		require.Equal(t, first.fields[fullName].Name, second.fields[fullName].Name)
	}
	require.Equal(t, "fooBar_e298b0d6", first.fields["fixture.messaging.test.v1.Collision.foo_bar"].Name)
	require.Equal(t, "fooBar2", first.fields["fixture.messaging.test.v1.Collision.foo_bar2"].Name)
	require.NotEqual(t,
		first.fields["fixture.messaging.test.v1.Collision.foo_bar"].Name,
		first.fields["fixture.messaging.test.v1.Collision.fooBar"].Name,
	)
	require.Equal(t, "notSet5", first.fields["fixture.messaging.test.v1.Collision.not_set"].Name)
	require.Equal(t, "matchValue", first.fields["fixture.messaging.test.v1.Collision.match"].Name)
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
	require.Equal(t, leanType{Kind: leanTypeOption, Arguments: []leanType{namedLeanType("Outer.Inner")}},
		plan.fields["fixture.messaging.test.v1.Holder.local"].Type)
	require.Equal(t, leanType{Kind: leanTypeOption, Arguments: []leanType{namedLeanType("Example.Common.V1.External")}},
		plan.fields["fixture.messaging.test.v1.Holder.external"].Type)
	require.Equal(t, leanType{Kind: leanTypeOption, Arguments: []leanType{namedLeanType("Fixture.Messaging.Test.V1.Sub.External")}},
		plan.fields["fixture.messaging.test.v1.Holder.subpackage"].Type)
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
	require.Equal(t, leanType{Kind: leanTypeOption, Arguments: []leanType{namedLeanType("Fixture.API.Proto.MessageRef")}},
		plan.fields["fixture.messaging.test.v1.A.value"].Type)
	require.Equal(t, leanType{Kind: leanTypeOption, Arguments: []leanType{namedLeanType("A")}},
		plan.fields["fixture.messaging.test.v1.C.value"].Type)

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

func TestBuildLeanPlanRejectsInvalidInputs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		document      projection
		configuration generationConfig
		want          string
	}{
		{
			name: "empty package",
			document: projection{Messages: []messageProjection{{
				FullName: "Record", Name: "Record",
			}}},
			want: "build Lean package names: empty protobuf package",
		},
		{
			name: "duplicate identity",
			document: projection{
				Enums:    []enumProjection{{FullName: "fixture.v1.Record", Name: "Record", Package: "fixture.v1"}},
				Messages: []messageProjection{{FullName: "fixture.v1.Record", Name: "Record", Package: "fixture.v1"}},
			},
			want: `build Lean declaration names: duplicate protobuf identity "fixture.v1.Record"`,
		},
		{
			name: "duplicate field",
			document: projection{Messages: []messageProjection{{
				FullName: "fixture.v1.Record", Name: "Record", Package: "fixture.v1",
				Fields: []fieldProjection{
					{FullName: "fixture.v1.Record.value", Name: "value", Number: 1, Kind: "string"},
					{FullName: "fixture.v1.Record.value", Name: "other", Number: 2, Kind: "string"},
				},
			}}},
			want: `plan message "fixture.v1.Record": duplicate field "fixture.v1.Record.value"`,
		},
		{
			name: "unresolved parent",
			document: projection{Messages: []messageProjection{{
				FullName: "fixture.v1.Child", Name: "Child", Package: "fixture.v1", Parent: "fixture.v1.Missing",
			}}},
			want: `build Lean declarations: unresolved parent for "fixture.v1.Child"`,
		},
		{
			name: "unknown named type",
			document: projection{Messages: []messageProjection{{
				FullName: "fixture.v1.Record", Name: "Record", Package: "fixture.v1",
				Fields: []fieldProjection{{
					FullName: "fixture.v1.Record.value", Name: "value", Number: 1,
					Kind: "message", TypeName: "fixture.v1.Missing", Presence: true,
				}},
			}}},
			want: `plan field "fixture.v1.Record.value": unknown protobuf type "fixture.v1.Missing"`,
		},
		{
			name: "missing oneof",
			document: projection{Messages: []messageProjection{{
				FullName: "fixture.v1.Record", Name: "Record", Package: "fixture.v1",
				Fields: []fieldProjection{{
					FullName: "fixture.v1.Record.value", Name: "value", Number: 1, Kind: "string", Oneof: "choice",
				}},
			}}},
			want: `plan field "fixture.v1.Record.value": unresolved oneof "choice"`,
		},
		{
			name: "mismatched oneof",
			document: projection{Messages: []messageProjection{{
				FullName: "fixture.v1.Record", Name: "Record", Package: "fixture.v1",
				Fields: []fieldProjection{{
					FullName: "fixture.v1.Record.value", Name: "value", Number: 1, Kind: "string", Oneof: "other",
				}},
				Oneofs: []oneofProjection{{
					FullName: "fixture.v1.Record.choice", Name: "choice", FieldNames: []string{"fixture.v1.Record.value"},
				}},
			}}},
			want: `plan oneof "fixture.v1.Record.choice": field "fixture.v1.Record.value" belongs to "other"`,
		},
		{
			name: "unsupported scalar kind",
			document: projection{Messages: []messageProjection{{
				FullName: "fixture.v1.Record", Name: "Record", Package: "fixture.v1",
				Fields: []fieldProjection{{
					FullName: "fixture.v1.Record.value", Name: "value", Number: 1, Kind: "group",
				}},
			}}},
			want: `plan field "fixture.v1.Record.value": unsupported protobuf kind "group"`,
		},
		{
			name: "support name collision",
			document: projection{Messages: []messageProjection{{
				FullName: "acme.model.a_p_i.proto.Bytes", Name: "Bytes", Package: "acme.model.a_p_i.proto",
			}}},
			configuration: testGenerationConfig("Acme.Model"),
			want:          `protobuf declaration "acme.model.a_p_i.proto.Bytes" collides with generated support declaration "Acme.Model.API.Proto.Bytes"`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configuration := test.configuration
			if configuration.Layout.RootModule == "" {
				configuration = testGenerationConfig("Fixture")
			}
			plan, err := buildLeanPlan(test.document, configuration)
			require.EqualError(t, err, test.want)
			require.Equal(t, leanPlan{}, plan)
		})
	}
}

func TestBuildLeanPlanPreservesFailurePrecedence(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		document projection
		want     string
	}{
		{
			name: "message graph before packages",
			document: projection{Messages: []messageProjection{
				{FullName: "Record", Name: "Record"},
				{FullName: "Record", Name: "Record"},
			}},
			want: `build message graph: duplicate message "Record"`,
		},
		{
			name: "packages before declaration parents",
			document: projection{Messages: []messageProjection{{
				FullName: "Child", Name: "Child", Parent: "Missing",
			}}},
			want: "build Lean package names: empty protobuf package",
		},
		{
			name: "support collisions before field types",
			document: projection{Messages: []messageProjection{
				{FullName: "fixture.a_p_i.proto.Bytes", Name: "Bytes", Package: "fixture.a_p_i.proto"},
				{
					FullName: "fixture.v1.Record", Name: "Record", Package: "fixture.v1",
					Fields: []fieldProjection{{
						FullName: "fixture.v1.Record.value", Name: "value", Number: 1,
						Kind: "message", TypeName: "fixture.v1.Missing",
					}},
				},
			}},
			want: `protobuf declaration "fixture.a_p_i.proto.Bytes" collides with generated support declaration "Fixture.API.Proto.Bytes"`,
		},
		{
			name: "field types before oneof ownership",
			document: projection{Messages: []messageProjection{{
				FullName: "fixture.v1.Record", Name: "Record", Package: "fixture.v1",
				Fields: []fieldProjection{{
					FullName: "fixture.v1.Record.value", Name: "value", Number: 1,
					Kind: "message", TypeName: "fixture.v1.Missing", Oneof: "choice",
				}},
			}}},
			want: `plan field "fixture.v1.Record.value": unknown protobuf type "fixture.v1.Missing"`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plan, err := buildLeanPlan(test.document, testGenerationConfig("Fixture"))
			require.EqualError(t, err, test.want)
			require.Equal(t, leanPlan{}, plan)
		})
	}
}

func TestLeanPlanValidationRejectsMalformedStructuralTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		typeValue leanType
		want      string
	}{
		{name: "named", typeValue: leanType{Kind: leanTypeNamed}, want: "invalid named type"},
		{name: "option", typeValue: leanType{Kind: leanTypeOption}, want: "type constructor requires one argument"},
		{name: "product", typeValue: leanType{Kind: leanTypeProduct, Arguments: []leanType{namedLeanType("String")}}, want: "product type requires two arguments"},
		{name: "unknown", typeValue: leanType{Kind: leanTypeKind(255)}, want: "unknown type constructor 255"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			document := projection{Messages: []messageProjection{{
				FullName: "fixture.v1.Record", Name: "Record", Package: "fixture.v1",
				Fields: []fieldProjection{{
					FullName: "fixture.v1.Record.value", Name: "value", Number: 1, Kind: "string",
				}},
			}}}
			configuration := testGenerationConfig("Fixture")
			plan, err := buildLeanPlan(document, configuration)
			require.NoError(t, err)
			plan.Messages[0].StructureFields[0].Type = test.typeValue

			err = validateLeanPlan(document, plan, configuration)
			require.EqualError(t, err, `validate Lean plan: message "fixture.v1.Record" field "value": `+test.want)
		})
	}
}

func TestLeanPlanValidationRejectsStructuralInconsistencies(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*leanPlan)
		want   string
	}{
		{
			name:   "incomplete imports",
			mutate: func(plan *leanPlan) { plan.APIModule.Imports = nil },
			want:   "validate Lean plan: API module is incomplete",
		},
		{
			name:   "support namespace mismatch",
			mutate: func(plan *leanPlan) { plan.supportNamespace = "Fixture.API.Other" },
			want:   "validate Lean plan: support namespace is incomplete",
		},
		{
			name:   "namespace mismatch",
			mutate: func(plan *leanPlan) { plan.Namespaces = nil },
			want:   "validate Lean plan: namespace ownership mismatch",
		},
		{
			name:   "dependency order",
			mutate: func(plan *leanPlan) { plan.Messages[0], plan.Messages[1] = plan.Messages[1], plan.Messages[0] },
			want:   `validate Lean plan: message "fixture.v1.Record" precedes dependency "fixture.v1.Dependency"`,
		},
		{
			name:   "service order",
			mutate: func(plan *leanPlan) { plan.Services[0], plan.Services[1] = plan.Services[1], plan.Services[0] },
			want:   "validate Lean plan: service order mismatch",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			document := leanPlanValidationProjection()
			configuration := testGenerationConfig("Fixture")
			plan, err := buildLeanPlan(document, configuration)
			require.NoError(t, err)
			test.mutate(&plan)

			err = validateLeanPlan(document, plan, configuration)
			require.EqualError(t, err, test.want)
		})
	}
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
	require.Equal(t, "Acme.Model.API.Proto", plan.supportNamespace)
	require.Equal(t, leanModulePlan{Path: "Acme/Model/API/Proto.lean", Imports: []string{}}, plan.ProtoModule)
	require.Equal(t, leanModulePlan{Path: "Acme/Model/API/Types.lean", Imports: []string{"Acme.Model.API.Proto"}}, plan.TypesModule)
	require.Equal(t, leanModulePlan{Path: "Acme/Model/API.lean", Imports: []string{"Acme.Model.API.Proto", "Acme.Model.API.Types"}}, plan.APIModule)
	require.Equal(t, namedLeanType("Acme.Model.API.Proto.Bytes"), plan.fields["example.v1.Record.payload"].Type)
}

func TestLeanPlanNormalizesCompositeFieldsAndStreamingMethods(t *testing.T) {
	t.Parallel()

	document := projection{
		Messages: []messageProjection{{
			FullName: "fixture.v1.Request", Name: "Request", Package: "fixture.v1",
			Fields: []fieldProjection{
				{
					FullName: "fixture.v1.Request.labels", Name: "labels", Number: 1,
					Map: true, MapKey: "string", MapValue: "string",
				},
				{FullName: "fixture.v1.Request.payloads", Name: "payloads", Number: 2, Kind: "bytes", Repeated: true},
				{FullName: "fixture.v1.Request.note", Name: "note", Number: 3, Kind: "string", Presence: true},
			},
		}},
		Services: []serviceProjection{{
			FullName: "fixture.v1.Stream", Name: "Stream", Package: "fixture.v1",
			Methods: []methodProjection{{
				FullName: "fixture.v1.Stream.Exchange", Name: "Exchange",
				InputType: "fixture.v1.Request", OutputType: "fixture.v1.Request",
				ClientStreaming: true, ServerStreaming: true,
			}},
		}},
	}
	plan, err := buildLeanPlan(document, testGenerationConfig("Fixture"))
	require.NoError(t, err)

	require.Equal(t, leanType{
		Kind: leanTypeList,
		Arguments: []leanType{{
			Kind: leanTypeProduct, Arguments: []leanType{namedLeanType("String"), namedLeanType("String")},
		}},
	}, plan.fields["fixture.v1.Request.labels"].Type)
	require.Equal(t, leanType{Kind: leanTypeList, Arguments: []leanType{namedLeanType("Fixture.API.Proto.Bytes")}},
		plan.fields["fixture.v1.Request.payloads"].Type)
	require.Equal(t, leanType{Kind: leanTypeOption, Arguments: []leanType{namedLeanType("String")}},
		plan.fields["fixture.v1.Request.note"].Type)
	require.Equal(t, leanMethodPlan{
		Projection:      document.Services[0].Methods[0],
		Name:            "exchange",
		QualifiedName:   leanName{"Fixture", "V1", "Stream", "exchange"},
		InputType:       namedLeanType("Request"),
		OutputType:      namedLeanType("Request"),
		FullName:        "fixture.v1.Stream.Exchange",
		ClientStreaming: true,
		ServerStreaming: true,
	}, plan.Services[0].Methods[0])
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

func leanPlanValidationProjection() projection {
	return projection{
		Messages: []messageProjection{
			{FullName: "fixture.v1.Dependency", Name: "Dependency", Package: "fixture.v1"},
			{
				FullName: "fixture.v1.Record", Name: "Record", Package: "fixture.v1",
				Fields: []fieldProjection{{
					FullName: "fixture.v1.Record.dependency", Name: "dependency", Number: 1,
					Kind: "message", TypeName: "fixture.v1.Dependency", Presence: true,
				}},
			},
		},
		Services: []serviceProjection{
			{
				FullName: "fixture.v1.Alpha", Name: "Alpha", Package: "fixture.v1",
				Methods: []methodProjection{{
					FullName: "fixture.v1.Alpha.Call", Name: "Call",
					InputType: "fixture.v1.Dependency", OutputType: "fixture.v1.Record",
				}},
			},
			{FullName: "fixture.v1.Beta", Name: "Beta", Package: "fixture.v1"},
		},
	}
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
