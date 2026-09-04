package testplan

import (
	"bytes"
	"context"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestGeneratedUnaryExecutorContract(t *testing.T) {
	service := umpirespb.File_temporal_server_api_umpire_v1_service_proto.Services().ByName("UmpireExecutor")
	require.NotNil(t, service)
	require.Equal(t, 1, service.Methods().Len())
	method := service.Methods().Get(0)
	require.Equal(t, "Execute", string(method.Name()))
	require.Equal(t, "temporal.server.api.umpire.v1.PortableTestPlan", string(method.Input().FullName()))
	require.Equal(t, "temporal.server.api.umpire.v1.ExecutionResult", string(method.Output().FullName()))
	require.Len(t, umpirespb.UmpireExecutor_ServiceDesc.Methods, 1)
}

func TestPortablePlanSchemaHasNoOpaqueDocuments(t *testing.T) {
	var byteFields []string
	messages := umpirespb.File_temporal_server_api_umpire_v1_portable_test_plan_proto.Messages()
	for index := 0; index < messages.Len(); index++ {
		collectByteFields(messages.Get(index), &byteFields)
	}
	slices.Sort(byteFields)
	require.Equal(t, []string{
		"temporal.server.api.umpire.v1.ExecutionResult.plan_checksum",
		"temporal.server.api.umpire.v1.PortableTestPlan.plan_checksum",
	}, byteFields)
	roleKinds := umpirespb.PORTABLE_DEFINITION_KIND_UNSPECIFIED.Descriptor().Values()
	for _, name := range []protoreflect.Name{
		"PORTABLE_DEFINITION_KIND_STATE",
		"PORTABLE_DEFINITION_KIND_ACTION",
		"PORTABLE_DEFINITION_KIND_OUTCOME",
		"PORTABLE_DEFINITION_KIND_OBSERVATION",
		"PORTABLE_DEFINITION_KIND_RELATION",
		"PORTABLE_DEFINITION_KIND_CAPABILITY",
		"PORTABLE_DEFINITION_KIND_PROVIDER",
		"PORTABLE_DEFINITION_KIND_LAW",
		"PORTABLE_DEFINITION_KIND_CONNECTOR",
		"PORTABLE_DEFINITION_KIND_TARGET",
		"PORTABLE_DEFINITION_KIND_KERNEL",
	} {
		require.NotNil(t, roleKinds.ByName(name), name)
	}
}

func TestSealAndAdmitCallerNeutralPlan(t *testing.T) {
	external := testPlan()
	sealedExternal, err := Seal(external)
	require.NoError(t, err)

	modelCompiled := testPlan()
	modelCompiled.Provenance = &umpirespb.PortableTestPlan_ModelCompiled{
		ModelCompiled: testModelProvenance(),
	}
	modelCompiled.GetExecution().SymbolicRoles = nil
	modelCompiled.GetExecution().RuntimeBindingSlots = nil
	modelCompiled.GetExecution().SelectedChoices = nil
	modelCompiled.GetExecution().SelectedVariants = nil
	modelCompiled.GetExecution().GetPreconditions()[0].Left = &umpirespb.ExecutionOperand{
		Operand: &umpirespb.ExecutionOperand_Role{Role: proto.CloneOf(modelCompiled.GetExecution().GetRoleBindings()[0].GetRole())},
	}
	modelCompiled.GetExecution().GetPreconditions()[0].Right.GetLiteral().Kind = umpirespb.PORTABLE_DEFINITION_KIND_STATE
	sealedModel, err := Seal(modelCompiled)
	require.NoError(t, err)

	for _, plan := range []*umpirespb.PortableTestPlan{sealedExternal, sealedModel} {
		admitted, err := Admit(plan)
		require.NoError(t, err)
		require.Equal(t, plan.GetPlanChecksum(), admitted.Checksum())
		require.True(t, proto.Equal(plan, admitted.Plan()))
		require.NotZero(t, admitted.MandatoryResultBytes())
	}
	require.IsType(t, &umpirespb.ExecutionProgram{}, sealedExternal.GetExecution())
	require.IsType(t, &umpirespb.ExecutionProgram{}, sealedModel.GetExecution())
	require.IsType(t, &umpirespb.VerificationProgram{}, sealedExternal.GetVerification())
	require.IsType(t, &umpirespb.VerificationProgram{}, sealedModel.GetVerification())
}

func TestChecksumUsesDecodedPlanValue(t *testing.T) {
	sealed, err := Seal(testPlan())
	require.NoError(t, err)
	canonical, err := proto.MarshalOptions{Deterministic: true}.Marshal(sealed)
	require.NoError(t, err)

	versionBytes, err := proto.Marshal(sealed.GetVersion())
	require.NoError(t, err)
	versionField := protowire.AppendTag(nil, 1, protowire.BytesType)
	versionField = protowire.AppendBytes(versionField, versionBytes)
	noncanonical := append(bytes.Clone(canonical[len(versionField):]), versionField...)
	decoded := new(umpirespb.PortableTestPlan)
	require.NoError(t, proto.Unmarshal(noncanonical, decoded))

	admitted, err := Admit(decoded)
	require.NoError(t, err)
	require.Equal(t, sealed.GetPlanChecksum(), admitted.Checksum())
}

func TestAdmissionRejectsStructuralAndAuthorityMutations(t *testing.T) {
	testCases := []struct {
		name   string
		code   ErrorCode
		mutate func(*umpirespb.PortableTestPlan)
	}{
		{
			name: "unknown top-level field", code: ErrorUnknownField,
			mutate: func(plan *umpirespb.PortableTestPlan) {
				plan.ProtoReflect().SetUnknown(protowire.AppendVarint(protowire.AppendTag(nil, 1000, protowire.VarintType), 1))
			},
		},
		{
			name: "unknown nested field", code: ErrorUnknownField,
			mutate: func(plan *umpirespb.PortableTestPlan) {
				plan.GetExecution().GetRuntime().ProtoReflect().SetUnknown(
					protowire.AppendVarint(protowire.AppendTag(nil, 1000, protowire.VarintType), 1),
				)
			},
		},
		{
			name: "unknown enum", code: ErrorUnsupportedEnum,
			mutate: func(plan *umpirespb.PortableTestPlan) {
				plan.GetVerification().GetDecision().Kind = 99
			},
		},
		{
			name: "unsupported version", code: ErrorUnsupportedVersion,
			mutate: func(plan *umpirespb.PortableTestPlan) {
				plan.GetVersion().Major = 2
			},
		},
		{
			name: "missing provenance", code: ErrorMalformedValue,
			mutate: func(plan *umpirespb.PortableTestPlan) {
				plan.Provenance = nil
			},
		},
		{
			name: "missing trace projection", code: ErrorUnsupportedOperator,
			mutate: func(plan *umpirespb.PortableTestPlan) {
				plan.GetVerification().TraceProjection = nil
			},
		},
		{
			name: "duplicate property", code: ErrorDuplicate,
			mutate: func(plan *umpirespb.PortableTestPlan) {
				plan.GetVerification().Properties = append(
					plan.GetVerification().Properties,
					proto.CloneOf(plan.GetVerification().GetProperties()[0]),
				)
			},
		},
		{
			name: "crossed model query", code: ErrorBinding,
			mutate: func(plan *umpirespb.PortableTestPlan) {
				plan.Provenance = &umpirespb.PortableTestPlan_ModelCompiled{ModelCompiled: testModelProvenance()}
				plan.GetModelCompiled().GetQuery().BehaviorFingerprint =
					"sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
			},
		},
		{
			name: "crossed model experiment", code: ErrorBinding,
			mutate: func(plan *umpirespb.PortableTestPlan) {
				plan.Provenance = &umpirespb.PortableTestPlan_ModelCompiled{ModelCompiled: testModelProvenance()}
				plan.GetModelCompiled().GetExperiment().BehaviorFingerprint =
					"sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
			},
		},
		{
			name: "crossed model runtime config", code: ErrorBinding,
			mutate: func(plan *umpirespb.PortableTestPlan) {
				plan.Provenance = &umpirespb.PortableTestPlan_ModelCompiled{ModelCompiled: testModelProvenance()}
				plan.GetModelCompiled().GetRuntimeConfig().BehaviorFingerprint =
					"sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
			},
		},
		{
			name: "unordered model properties", code: ErrorOrdering,
			mutate: func(plan *umpirespb.PortableTestPlan) {
				plan.Provenance = &umpirespb.PortableTestPlan_ModelCompiled{ModelCompiled: testModelProvenance()}
				plan.GetModelCompiled().Properties = []*umpirespb.DefinitionBinding{
					testBinding("umpire.property.zed"),
					testBinding("umpire.property.basic"),
				}
			},
		},
		{
			name: "crossed initial state kind", code: ErrorBinding,
			mutate: func(plan *umpirespb.PortableTestPlan) {
				plan.GetExecution().GetInitialState().Kind = umpirespb.PORTABLE_DEFINITION_KIND_ACTION
			},
		},
		{
			name: "role both bound and symbolic", code: ErrorDuplicate,
			mutate: func(plan *umpirespb.PortableTestPlan) {
				plan.GetExecution().GetRoleBindings()[0].Role = proto.CloneOf(
					plan.GetExecution().GetSymbolicRoles()[0].GetDefinition(),
				)
			},
		},
		{
			name: "crossed runtime slot", code: ErrorBinding,
			mutate: func(plan *umpirespb.PortableTestPlan) {
				plan.GetExecution().GetPreconditions()[0].Left = &umpirespb.ExecutionOperand{
					Operand: &umpirespb.ExecutionOperand_RuntimeBindingSlot{RuntimeBindingSlot: testBinding("umpire.runtime-slot.crossed")},
				}
			},
		},
		{
			name: "crossed runtime slot scalar kind", code: ErrorBinding,
			mutate: func(plan *umpirespb.PortableTestPlan) {
				plan.GetExecution().GetRuntimeBindingSlots()[0].ValueKind = umpirespb.PORTABLE_VALUE_KIND_NATURAL
			},
		},
		{
			name: "crossed bound role scalar kind", code: ErrorBinding,
			mutate: func(plan *umpirespb.PortableTestPlan) {
				role := plan.GetExecution().GetRoleBindings()[0]
				role.GetValue().Value = &umpirespb.Value{Value: &umpirespb.Value_BoolValue{BoolValue: true}}
				precondition := plan.GetExecution().GetPreconditions()[0]
				precondition.Left = &umpirespb.ExecutionOperand{
					Operand: &umpirespb.ExecutionOperand_Role{Role: proto.CloneOf(role.GetRole())},
				}
				precondition.GetRight().GetLiteral().Kind = role.GetValue().GetKind()
			},
		},
		{
			name: "duplicate participant capability", code: ErrorDuplicate,
			mutate: func(plan *umpirespb.PortableTestPlan) {
				capability := plan.GetExecution().GetRuntime().GetParticipantBindings()[0].GetCapabilities()[0]
				plan.GetExecution().GetRuntime().GetParticipantBindings()[0].Capabilities = []*umpirespb.DefinitionBinding{
					capability, proto.CloneOf(capability),
				}
			},
		},
		{
			name: "crossed evidence field", code: ErrorBinding,
			mutate: func(plan *umpirespb.PortableTestPlan) {
				plan.GetVerification().GetObservation().GetEmits()[0].GetValue().Operator =
					&umpirespb.ObservationExpression_Field{Field: &umpirespb.EvidenceFieldReference{
						KindDefinitionId:  "umpire.evidence.kind.basic",
						FieldDefinitionId: "umpire.evidence.field.crossed",
					}}
			},
		},
		{
			name: "crossed evidence kind", code: ErrorBinding,
			mutate: func(plan *umpirespb.PortableTestPlan) {
				plan.GetVerification().GetObservation().GetEmits()[0].SourceKindDefinitionId =
					"umpire.evidence.kind.crossed"
			},
		},
		{
			name: "crossed emit condition scalar kind", code: ErrorBinding,
			mutate: func(plan *umpirespb.PortableTestPlan) {
				plan.GetVerification().GetObservation().GetEmits()[0].Condition = &umpirespb.ObservationExpression{
					Operator: &umpirespb.ObservationExpression_LiteralText{LiteralText: &umpirespb.LiteralText{Value: "true"}},
				}
			},
		},
		{
			name: "crossed emit value scalar kind", code: ErrorBinding,
			mutate: func(plan *umpirespb.PortableTestPlan) {
				plan.GetVerification().GetObservation().GetEmits()[0].Value = &umpirespb.ObservationExpression{
					Operator: &umpirespb.ObservationExpression_LiteralNatural{LiteralNatural: &umpirespb.LiteralNatural{Value: "1"}},
				}
			},
		},
		{
			name: "crossed emit semantic kind", code: ErrorBinding,
			mutate: func(plan *umpirespb.PortableTestPlan) {
				plan.GetVerification().GetObservation().GetEmits()[0].OutputKind = umpirespb.DEFINITION_KIND_STATE
			},
		},
		{
			name: "unresolved digest policy", code: ErrorBinding,
			mutate: func(plan *umpirespb.PortableTestPlan) {
				profile := plan.GetVerification().GetEvidence()
				field := profile.GetKinds()[0].GetFields()[0]
				field.Disposition = umpirespb.FIELD_DISPOSITION_KIND_HASH
				field.DigestPolicyDefinitionId = "umpire.digest.missing"
				plan.GetVerification().GetObservation().Profile = proto.CloneOf(profile)
			},
		},
		{
			name: "crossed correlation field", code: ErrorBinding,
			mutate: func(plan *umpirespb.PortableTestPlan) {
				profile := plan.GetVerification().GetEvidence()
				profile.CorrelationSlots = []*umpirespb.CorrelationSlot{{
					DefinitionId: "umpire.correlation.basic",
					Kind:         umpirespb.CORRELATION_SLOT_KIND_RUN,
					Fields: []*umpirespb.EvidenceFieldReference{{
						KindDefinitionId:  "umpire.evidence.kind.basic",
						FieldDefinitionId: "umpire.evidence.field.crossed",
					}},
				}}
				plan.GetVerification().GetObservation().Profile = proto.CloneOf(profile)
			},
		},
		{
			name: "cyclic emit ordering", code: ErrorOrdering,
			mutate: func(plan *umpirespb.PortableTestPlan) {
				observation := plan.GetVerification().GetObservation()
				second := proto.CloneOf(observation.GetEmits()[0])
				second.DefinitionId = "umpire.emit.second"
				second.OutputDefinition = testBinding("umpire.observation.second")
				second.Coordinate.Position = 2
				observation.Emits = append(observation.Emits, second)
				plan.GetExecution().GetCheckpoints()[0].Observations = append(
					plan.GetExecution().GetCheckpoints()[0].Observations,
					&umpirespb.PortableModelValue{
						Definition: testBinding("umpire.observation.second"), Kind: umpirespb.PORTABLE_DEFINITION_KIND_OBSERVATION,
						Value: &umpirespb.Value{Value: &umpirespb.Value_Text{Text: "observed"}},
					},
				)
				observation.Ordering = []*umpirespb.EmitOrdering{
					{PredecessorEmitDefinitionId: "umpire.emit.basic", SuccessorEmitDefinitionId: "umpire.emit.second"},
					{PredecessorEmitDefinitionId: "umpire.emit.second", SuccessorEmitDefinitionId: "umpire.emit.basic"},
				}
			},
		},
		{
			name: "malformed exact rename entry", code: ErrorMalformedValue,
			mutate: func(plan *umpirespb.PortableTestPlan) {
				plan.GetVerification().TraceProjection = &umpirespb.VerificationProgram_RenameExactLink{
					RenameExactLink: &umpirespb.RenameExactLink{
						Definition:        testBinding("umpire.rename.basic"),
						Source:            &umpirespb.SourceLocation{Path: "rename.proto", Line: 1, Column: 1, Provenance: "fixture"},
						SourceTarget:      testBinding("umpire.target.source"),
						DestinationTarget: proto.CloneOf(plan.GetExecution().GetTarget()),
						Entries: []*umpirespb.RenameExactEntry{nil, {
							Source: &umpirespb.ModelValue{
								Definition: testBinding("umpire.observation.expected"), Kind: umpirespb.DEFINITION_KIND_OBSERVATION,
								Value: &umpirespb.Value{Value: &umpirespb.Value_Text{Text: "observed"}},
							},
							Destination: &umpirespb.ModelValue{
								Definition: testBinding("umpire.observation.destination"), Kind: umpirespb.DEFINITION_KIND_OBSERVATION,
								Value: &umpirespb.Value{Value: &umpirespb.Value_Text{Text: "observed"}},
							},
						}},
						ApplicationLimit: &umpirespb.Limit{Value: 1, Unit: "semantic-transitions"},
					},
				}
			},
		},
		{
			name: "crossed direct trace property", code: ErrorBinding,
			mutate: func(plan *umpirespb.PortableTestPlan) {
				plan.GetVerification().GetProperties()[0].GetClauses()[0].GetPerStepImplies().GetTrigger().Definition =
					testBinding("umpire.action.crossed")
			},
		},
		{
			name: "crossed Property scalar kind", code: ErrorBinding,
			mutate: func(plan *umpirespb.PortableTestPlan) {
				plan.GetVerification().GetProperties()[0].GetClauses()[0].GetPerStepImplies().GetTrigger().Operator =
					&umpirespb.Pattern_NaturalAtMost{NaturalAtMost: &umpirespb.NaturalAtMost{Bound: "1"}}
			},
		},
		{
			name: "second checkpoint", code: ErrorLimit,
			mutate: func(plan *umpirespb.PortableTestPlan) {
				checkpoint := proto.CloneOf(plan.GetExecution().GetCheckpoints()[0])
				checkpoint.Transition = 2
				plan.GetExecution().Checkpoints = append(plan.GetExecution().Checkpoints, checkpoint)
			},
		},
		{
			name: "duplicate precondition", code: ErrorDuplicate,
			mutate: func(plan *umpirespb.PortableTestPlan) {
				plan.GetExecution().Preconditions = append(
					plan.GetExecution().Preconditions,
					proto.CloneOf(plan.GetExecution().GetPreconditions()[0]),
				)
			},
		},
		{
			name: "unordered preconditions", code: ErrorOrdering,
			mutate: func(plan *umpirespb.PortableTestPlan) {
				first := plan.GetExecution().GetPreconditions()[0]
				first.Definition = testBinding("umpire.precondition.zed")
				second := proto.CloneOf(first)
				second.Definition = testBinding("umpire.precondition.alpha")
				plan.GetExecution().Preconditions = append(plan.GetExecution().Preconditions, second)
			},
		},
		{
			name: "malformed Known Gap subject", code: ErrorMalformedValue,
			mutate: func(plan *umpirespb.PortableTestPlan) {
				plan.GetKnownGaps()[0].Subject = "not a definition id"
			},
		},
		{
			name: "unordered Known Gaps", code: ErrorOrdering,
			mutate: func(plan *umpirespb.PortableTestPlan) {
				plan.KnownGaps = append(plan.KnownGaps, &umpirespb.KnownGap{
					Kind: umpirespb.KNOWN_GAP_KIND_INPUT, Code: "umpire.gap.input",
				})
			},
		},
		{
			name: "reordered phases", code: ErrorOrdering,
			mutate: func(plan *umpirespb.PortableTestPlan) {
				phases := plan.GetExecution().GetRuntime().PhaseLimits
				phases[0], phases[1] = phases[1], phases[0]
			},
		},
		{
			name: "second action", code: ErrorLimit,
			mutate: func(plan *umpirespb.PortableTestPlan) {
				plan.GetExecution().RequestedActions = append(
					plan.GetExecution().RequestedActions,
					proto.CloneOf(plan.GetExecution().GetRequestedActions()[0]),
				)
			},
		},
		{
			name: "second participant", code: ErrorLimit,
			mutate: func(plan *umpirespb.PortableTestPlan) {
				plan.GetExecution().GetRuntime().ParticipantBindings = append(
					plan.GetExecution().GetRuntime().ParticipantBindings,
					proto.CloneOf(plan.GetExecution().GetRuntime().GetParticipantBindings()[0]),
				)
			},
		},
		{
			name: "second fault", code: ErrorLimit,
			mutate: func(plan *umpirespb.PortableTestPlan) {
				fault := &umpirespb.PortableModelValue{
					Definition: testBinding("umpire.fault.basic"), Kind: umpirespb.PORTABLE_DEFINITION_KIND_RELATION,
					Value: &umpirespb.Value{Value: &umpirespb.Value_Text{Text: "fault"}},
				}
				plan.GetExecution().RequestedFaults = []*umpirespb.PortableModelValue{fault, proto.CloneOf(fault)}
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			plan := testPlan()
			testCase.mutate(plan)
			_, err := Seal(plan)
			requirePlanError(t, err, testCase.code)
		})
	}

	sealed, err := Seal(testPlan())
	require.NoError(t, err)
	sealed.PlanId = "umpire.plan.external.mutated"
	_, err = Admit(sealed)
	requirePlanError(t, err, ErrorChecksum)
}

func TestAdmissionUsesCompleteKnownGapIdentity(t *testing.T) {
	plan := testPlan()
	second := &umpirespb.KnownGap{
		Kind: umpirespb.KNOWN_GAP_KIND_CLAIM, Code: "umpire.gap.external-fixture",
		Subject: "umpire.plan.zzz", Detail: "second scoped gap",
	}
	plan.KnownGaps = append(plan.KnownGaps, second)
	plan.GetExecution().GetArtifactProjection().ExperimentKnownGaps = append(
		plan.GetExecution().GetArtifactProjection().GetExperimentKnownGaps(), proto.CloneOf(second),
	)
	_, err := Seal(plan)
	require.NoError(t, err)
}

func TestAdmissionEnforcesIndependentLimitBoundaries(t *testing.T) {
	testCases := []struct {
		name   string
		mutate func(*umpirespb.PortableTestPlan, int64)
		limit  int64
	}{
		{"plan bytes", func(plan *umpirespb.PortableTestPlan, value int64) {
			plan.GetLimits().GetStructural().MaxPlanBytes = value
		}, 1 << 20},
		{"nesting depth", func(plan *umpirespb.PortableTestPlan, value int64) {
			plan.GetLimits().GetStructural().MaxNestingDepth = value
		}, 256},
		{"collection items", func(plan *umpirespb.PortableTestPlan, value int64) {
			plan.GetLimits().GetStructural().MaxCollectionItems = value
		}, 10_000},
		{"operator count", func(plan *umpirespb.PortableTestPlan, value int64) {
			plan.GetLimits().GetStructural().MaxOperatorCount = value
		}, 100_000},
		{"actions", func(plan *umpirespb.PortableTestPlan, value int64) {
			plan.GetLimits().GetExecution().MaxActions = value
		}, 1},
		{"faults", func(plan *umpirespb.PortableTestPlan, value int64) { plan.GetLimits().GetExecution().MaxFaults = value }, 1},
		{"phase attempts", func(plan *umpirespb.PortableTestPlan, value int64) {
			plan.GetLimits().GetExecution().MaxPhaseAttempts = value
		}, 1},
		{"phase duration", func(plan *umpirespb.PortableTestPlan, value int64) {
			plan.GetLimits().GetExecution().MaxPhaseDurationMilliseconds = value
		}, 30_000},
		{"total duration", func(plan *umpirespb.PortableTestPlan, value int64) {
			plan.GetLimits().GetExecution().MaxTotalDurationMilliseconds = value
		}, 120_000},
		{"evidence records", func(plan *umpirespb.PortableTestPlan, value int64) { plan.GetLimits().GetEvidence().MaxRecords = value }, 100_000},
		{"evidence bytes", func(plan *umpirespb.PortableTestPlan, value int64) { plan.GetLimits().GetEvidence().MaxBytes = value }, 16 << 20},
		{"evidence sources", func(plan *umpirespb.PortableTestPlan, value int64) { plan.GetLimits().GetEvidence().MaxSources = value }, 10_000},
		{"expression depth", func(plan *umpirespb.PortableTestPlan, value int64) {
			plan.GetLimits().GetEvaluation().MaxExpressionDepth = value
		}, 64},
		{"evaluation work", func(plan *umpirespb.PortableTestPlan, value int64) { plan.GetLimits().GetEvaluation().MaxWork = value }, 10_000_000},
		{"diagnostic bytes", func(plan *umpirespb.PortableTestPlan, value int64) {
			plan.GetLimits().GetOutput().MaxDiagnosticBytes = value
		}, 64 << 10},
		{"result bytes", func(plan *umpirespb.PortableTestPlan, value int64) {
			plan.GetLimits().GetOutput().MaxResultBytes = value
		}, 4 << 20},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			atLimit := testPlan()
			testCase.mutate(atLimit, testCase.limit)
			_, err := Seal(atLimit)
			require.NoError(t, err)

			beyondLimit := testPlan()
			testCase.mutate(beyondLimit, testCase.limit+1)
			_, err = Seal(beyondLimit)
			requirePlanError(t, err, ErrorLimit)
		})
	}
}

func TestAdmissionRejectsDeclaredContentAndResultNPlusOne(t *testing.T) {
	plan := testPlan()
	plan.GetLimits().GetStructural().MaxCollectionItems = 5
	_, err := Seal(plan)
	require.NoError(t, err)

	plan.GetVerification().GetEvidence().Sources = append(
		plan.GetVerification().GetEvidence().Sources,
		&umpirespb.EvidenceSourceDeclaration{SourceDefinitionId: "umpire.evidence.source.two"},
		&umpirespb.EvidenceSourceDeclaration{SourceDefinitionId: "umpire.evidence.source.three"},
		&umpirespb.EvidenceSourceDeclaration{SourceDefinitionId: "umpire.evidence.source.four"},
		&umpirespb.EvidenceSourceDeclaration{SourceDefinitionId: "umpire.evidence.source.five"},
		&umpirespb.EvidenceSourceDeclaration{SourceDefinitionId: "umpire.evidence.source.six"},
	)
	_, err = Seal(plan)
	requirePlanError(t, err, ErrorLimit)

	for _, testCase := range []struct {
		name       string
		provenance func(*umpirespb.PortableTestPlan)
		verifier   ModelProvenanceVerifier
		outcome    umpirespb.ProvenanceOutcome
		scope      umpirespb.ClaimScope
	}{
		{
			name: "external", provenance: func(*umpirespb.PortableTestPlan) {},
			outcome: umpirespb.PROVENANCE_OUTCOME_EXTERNAL, scope: umpirespb.CLAIM_SCOPE_PLAN_LOCAL,
		},
		{
			name: "model compiled", provenance: func(plan *umpirespb.PortableTestPlan) {
				plan.Provenance = &umpirespb.PortableTestPlan_ModelCompiled{ModelCompiled: testModelProvenance()}
			},
			verifier: acceptExactModelProvenance,
			outcome:  umpirespb.PROVENANCE_OUTCOME_MODEL_VERIFIED, scope: umpirespb.CLAIM_SCOPE_MODEL_BOUND,
		},
	} {
		t.Run(testCase.name+" mandatory result", func(t *testing.T) {
			resultPlan := testPlan()
			testCase.provenance(resultPlan)
			requireExactMandatoryResultLimit(t, resultPlan, testCase.verifier, testCase.outcome, testCase.scope)
		})
	}

	diagnosticPlan := testPlan()
	diagnosticBytes := int64(proto.Size(mandatoryResult(diagnosticPlan).GetDiagnostics()[0]))
	diagnosticPlan.GetLimits().GetOutput().MaxDiagnosticBytes = diagnosticBytes
	_, err = Seal(diagnosticPlan)
	require.NoError(t, err)
	diagnosticPlan.GetLimits().GetOutput().MaxDiagnosticBytes--
	_, err = Seal(diagnosticPlan)
	requirePlanError(t, err, ErrorLimit)
}

func TestAdmissionUsesExactDeclaredStructuralBounds(t *testing.T) {
	plan := testPlan()
	plan.GetLimits().GetStructural().MaxNestingDepth = 8
	plan.GetLimits().GetStructural().MaxCollectionItems = 5
	plan.GetLimits().GetStructural().MaxOperatorCount = 9
	plan.GetLimits().GetEvidence().MaxSources = 1
	sealed, err := Seal(plan)
	require.NoError(t, err)

	tooDeep := proto.CloneOf(plan)
	tooDeep.GetLimits().GetStructural().MaxNestingDepth--
	_, err = Seal(tooDeep)
	requirePlanError(t, err, ErrorLimit)

	tooManyOperators := proto.CloneOf(plan)
	tooManyOperators.GetLimits().GetStructural().MaxOperatorCount--
	_, err = Seal(tooManyOperators)
	requirePlanError(t, err, ErrorLimit)

	tooManySources := proto.CloneOf(plan)
	tooManySources.GetVerification().GetEvidence().Sources = append(
		tooManySources.GetVerification().GetEvidence().Sources,
		&umpirespb.EvidenceSourceDeclaration{SourceDefinitionId: "umpire.evidence.source.extra"},
	)
	tooManySources.GetVerification().GetObservation().Profile = proto.CloneOf(
		tooManySources.GetVerification().GetEvidence(),
	)
	_, err = Seal(tooManySources)
	requirePlanError(t, err, ErrorLimit)

	exactBytes := proto.CloneOf(plan)
	for {
		exactBytes.GetLimits().GetStructural().MaxPlanBytes = int64(proto.Size(sealed))
		sealed, err = Seal(exactBytes)
		require.NoError(t, err)
		if exactBytes.GetLimits().GetStructural().GetMaxPlanBytes() == int64(proto.Size(sealed)) {
			break
		}
	}
	exactBytes.GetLimits().GetStructural().MaxPlanBytes--
	_, err = Seal(exactBytes)
	requirePlanError(t, err, ErrorByteLimit)

	unsupportedPhaseBytes := proto.CloneOf(plan)
	unsupportedPhaseBytes.GetExecution().GetRuntime().GetPhaseLimits()[0].MaxBytes++
	_, err = Seal(unsupportedPhaseBytes)
	requirePlanError(t, err, ErrorLimit)
}

func requireExactMandatoryResultLimit(
	t *testing.T,
	plan *umpirespb.PortableTestPlan,
	verifier ModelProvenanceVerifier,
	outcome umpirespb.ProvenanceOutcome,
	scope umpirespb.ClaimScope,
) {
	t.Helper()
	plan.GetLimits().GetOutput().MaxDiagnosticBytes = 256
	sealed, err := Seal(plan)
	require.NoError(t, err)
	admitted, err := Admit(sealed)
	require.NoError(t, err)
	for plan.GetLimits().GetOutput().GetMaxResultBytes() != int64(admitted.MandatoryResultBytes()) {
		plan.GetLimits().GetOutput().MaxResultBytes = int64(admitted.MandatoryResultBytes())
		sealed, err = Seal(plan)
		require.NoError(t, err)
		admitted, err = Admit(sealed)
		require.NoError(t, err)
	}
	authorized, err := Authorize(context.Background(), admitted, verifier)
	require.NoError(t, err)
	reserved := authorized.ResultLimitExceeded()
	require.Equal(t, outcome, reserved.GetProvenanceOutcome())
	require.Equal(t, scope, reserved.GetClaimScope())
	require.Equal(t, admitted.MandatoryResultBytes(), proto.Size(reserved))
	plan.GetLimits().GetOutput().MaxResultBytes--
	_, err = Seal(plan)
	requirePlanError(t, err, ErrorLimit)
}

func collectByteFields(message protoreflect.MessageDescriptor, fields *[]string) {
	messageFields := message.Fields()
	for index := 0; index < messageFields.Len(); index++ {
		field := messageFields.Get(index)
		if field.Kind() == protoreflect.BytesKind {
			*fields = append(*fields, string(field.FullName()))
		}
	}
	nested := message.Messages()
	for index := 0; index < nested.Len(); index++ {
		collectByteFields(nested.Get(index), fields)
	}
}

func requirePlanError(t *testing.T, err error, want ErrorCode) {
	t.Helper()
	require.Error(t, err)
	code, ok := CodeOf(err)
	require.True(t, ok)
	require.Equal(t, want, code)
}

func testPlan() *umpirespb.PortableTestPlan {
	return &umpirespb.PortableTestPlan{
		Version:      &umpirespb.FormatVersion{Major: 1},
		PlanId:       "umpire.plan.external.basic",
		Provenance:   &umpirespb.PortableTestPlan_External{External: testExternalProvenance()},
		Execution:    testExecutionProgram(),
		Verification: testVerificationProgram(),
		Limits:       testLimits(),
		KnownGaps:    []*umpirespb.KnownGap{testKnownGap()},
		ExternalObligations: []*umpirespb.ExternalVerificationObligation{{
			Definition: testBinding("umpire.obligation.basic"),
			Kind:       umpirespb.EXTERNAL_VERIFICATION_OBLIGATION_KIND_ADVISORY,
			Source: &umpirespb.SourceLocation{
				Path: "clients/example/plan.proto", Line: 2, Column: 1, Provenance: "external fixture",
			},
			Statement: "Verify the environment-specific follow-up independently.",
		}},
	}
}

func testExternalProvenance() *umpirespb.ExternalPlanProvenance {
	return &umpirespb.ExternalPlanProvenance{Sources: []*umpirespb.SourceLocation{{
		Path: "clients/example/plan.proto", Line: 1, Column: 1, Provenance: "external fixture",
	}}}
}

func testModelProvenance() *umpirespb.ModelCompiledPlanProvenance {
	return &umpirespb.ModelCompiledPlanProvenance{
		Test:             testBinding("umpire.test.basic"),
		Query:            testBinding("umpire.query.basic"),
		Experiment:       testArtifactBinding("umpire-experiment/v2"),
		RuntimeConfig:    testArtifactBinding("umpire-runtime-configuration/v2"),
		Properties:       []*umpirespb.DefinitionBinding{testBinding("umpire.property.basic")},
		CompilerContract: testBinding("umpire.compiler.portable-plan.v1"),
		Sources: []*umpirespb.SourceLocation{{
			Path: "Temporal/Tool/PortableTestPlan.lean", Line: 1, Column: 1, Provenance: "Lean fixture",
		}},
	}
}

func testBinding(id string) *umpirespb.DefinitionBinding {
	return &umpirespb.DefinitionBinding{
		DefinitionId:        id,
		BehaviorFingerprint: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}
}

func testArtifactBinding(format string) *umpirespb.ArtifactBinding {
	return &umpirespb.ArtifactBinding{
		FormatVersion:       format,
		ArtifactChecksum:    "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		BehaviorFingerprint: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ProvenanceChecksum:  "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
	}
}

func testExecutionProgram() *umpirespb.ExecutionProgram {
	return &umpirespb.ExecutionProgram{
		Setup:    testBinding("umpire.setup.basic"),
		Query:    testBinding("umpire.query.basic"),
		Behavior: testBinding("umpire.behavior.basic"),
		Target:   testBinding("umpire.target.basic"),
		Kernel:   testBinding("umpire.kernel.basic"),
		RoleBindings: []*umpirespb.RoleBinding{{
			Role: testBinding("umpire.role.basic"),
			Value: &umpirespb.PortableModelValue{
				Definition: testBinding("umpire.binding.basic"), Kind: umpirespb.PORTABLE_DEFINITION_KIND_STATE,
				Value: &umpirespb.Value{Value: &umpirespb.Value_Text{Text: "participant"}},
			},
		}},
		SymbolicRoles: []*umpirespb.SymbolicRole{{
			Definition: testBinding("umpire.role.symbolic"), Kind: umpirespb.PORTABLE_DEFINITION_KIND_STATE,
		}},
		RuntimeBindingSlots: []*umpirespb.RuntimeBindingSlot{{
			Definition: testBinding("umpire.runtime-slot.workflow"), ValueKind: umpirespb.PORTABLE_VALUE_KIND_TEXT,
		}},
		Preconditions: []*umpirespb.ExecutionPrecondition{{
			Definition: testBinding("umpire.precondition.basic"), Operator: umpirespb.PRECONDITION_OPERATOR_EQUALS,
			Left: &umpirespb.ExecutionOperand{Operand: &umpirespb.ExecutionOperand_RuntimeBindingSlot{
				RuntimeBindingSlot: testBinding("umpire.runtime-slot.workflow"),
			}},
			Right: &umpirespb.ExecutionOperand{Operand: &umpirespb.ExecutionOperand_Literal{
				Literal: &umpirespb.PortableModelValue{
					Definition: testBinding("umpire.workflow.expected"), Kind: umpirespb.PORTABLE_DEFINITION_KIND_SETUP,
					Value: &umpirespb.Value{Value: &umpirespb.Value_Text{Text: "fixture"}},
				},
			}},
		}},
		InitialState: &umpirespb.PortableModelValue{
			Definition: testBinding("umpire.state.initial"), Kind: umpirespb.PORTABLE_DEFINITION_KIND_STATE,
			Value: &umpirespb.Value{Value: &umpirespb.Value_Text{Text: "initial"}},
		},
		RequestedActions: []*umpirespb.PortableModelValue{{
			Definition: testBinding("umpire.action.basic"), Kind: umpirespb.PORTABLE_DEFINITION_KIND_ACTION,
			Value: &umpirespb.Value{Value: &umpirespb.Value_Text{Text: "run"}},
		}},
		ModelOutcomes: []*umpirespb.PortableModelValue{{
			Definition: testBinding("umpire.outcome.basic"), Kind: umpirespb.PORTABLE_DEFINITION_KIND_OUTCOME,
			Value: &umpirespb.Value{Value: &umpirespb.Value_Text{Text: "done"}},
		}},
		ResultingStates: []*umpirespb.PortableModelValue{{
			Definition: testBinding("umpire.state.done"), Kind: umpirespb.PORTABLE_DEFINITION_KIND_STATE,
			Value: &umpirespb.Value{Value: &umpirespb.Value_Text{Text: "done"}},
		}},
		Occurrences: []*umpirespb.PlannedOccurrence{{
			Definition:           testBinding("umpire.occurrence.basic"),
			ActionDefinitionId:   "umpire.action.basic",
			Position:             1,
			AuthoredDefinitionId: "umpire.occurrence.basic",
		}},
		SelectedChoices: []*umpirespb.PortableModelValue{{
			Definition: testBinding("umpire.choice.basic"), Kind: umpirespb.PORTABLE_DEFINITION_KIND_RELATION,
			Value: &umpirespb.Value{Value: &umpirespb.Value_Text{Text: "choice"}},
		}},
		SelectedVariants: []*umpirespb.PortableModelValue{{
			Definition: testBinding("umpire.variant.basic"), Kind: umpirespb.PORTABLE_DEFINITION_KIND_RELATION,
			Value: &umpirespb.Value{Value: &umpirespb.Value_Text{Text: "variant"}},
		}},
		CapabilityRequirements: []*umpirespb.DefinitionBinding{testBinding("umpire.capability.semantic")},
		Checkpoints: []*umpirespb.ExecutionCheckpoint{{
			Transition: 1,
			Observations: []*umpirespb.PortableModelValue{{
				Definition: testBinding("umpire.observation.expected"), Kind: umpirespb.PORTABLE_DEFINITION_KIND_OBSERVATION,
				Value: &umpirespb.Value{Value: &umpirespb.Value_Text{Text: "observed"}},
			}},
		}},
		Runtime: &umpirespb.RuntimeProgram{
			AuthorityProfile: testBinding("umpire.authority.basic"),
			Config:           testBinding("umpire.runtime-configuration.basic"),
			ParticipantBindings: []*umpirespb.PortableParticipantBinding{{
				Participant: testBinding("umpire.participant.basic"),
				Protocol:    testBinding("umpire.protocol.basic"), ProtocolVersion: 2,
				Program:      testBinding("umpire.program.basic"),
				Capabilities: []*umpirespb.DefinitionBinding{testBinding("umpire.capability.participant")},
			}},
			ObservationConfig: &umpirespb.PortableObservationConfig{
				Profile: testBinding("umpire.evidence-profile.basic"),
				Program: testBinding("umpire.observation-program.basic"),
				Mapping: testBinding("umpire.observation-mapping.basic"),
			},
			PhaseLimits: testPhaseLimits(),
			Termination: &umpirespb.TerminationObligation{Definition: testBinding("umpire.termination.basic")},
			Cleanup:     &umpirespb.CleanupObligation{Definition: testBinding("umpire.cleanup.basic")},
			AuthorityRequiredCapabilities: []*umpirespb.DefinitionBinding{
				testBinding("umpire.capability.authority"),
			},
		},
		ArtifactProjection: testArtifactProjection(),
	}
}

func testArtifactProjection() *umpirespb.PlanArtifactProjection {
	return &umpirespb.PlanArtifactProjection{
		ExpandedLimits: &umpirespb.PlanSearchLimits{
			MaxSemanticTransitions: 1, MaxSelectedActions: 1, MaxCandidateEvaluations: 10,
		},
		SelectionReason: umpirespb.PLAN_SELECTION_REASON_BEHAVIOR_SELECTION,
		Explored: &umpirespb.PlanExploredCounts{
			Setups: 1, Traces: 1, Transitions: 1, PropertyEvaluations: 1,
		},
		ExperimentKnownGaps: []*umpirespb.KnownGap{testKnownGap()},
		ExperimentProvenance: &umpirespb.PlanArtifactProvenance{
			SourceDefinitionIds: []string{
				"umpire.behavior.basic", "umpire.kernel.basic", "umpire.property.basic",
				"umpire.query.basic", "umpire.target.basic",
			},
			SourceLocations: []*umpirespb.SourceLocation{{
				Path: "clients/example/plan.proto", Line: 1, Column: 1, Provenance: "external fixture",
			}},
		},
		RuntimeKnownGaps: []*umpirespb.KnownGap{},
		RuntimeProvenance: &umpirespb.PlanArtifactProvenance{
			SourceDefinitionIds: []string{
				"umpire.authority.basic", "umpire.evidence-profile.basic", "umpire.observation-mapping.basic",
				"umpire.observation-program.basic", "umpire.runtime-configuration.basic",
			},
			SourceLocations: []*umpirespb.SourceLocation{{
				Path: "clients/example/runtime.proto", Line: 1, Column: 1, Provenance: "external fixture",
			}},
		},
		ExperimentObservationRequirementDefinitionIds: []string{"umpire.observation.expected"},
		RuntimeObservationConfig: &umpirespb.PortableObservationConfig{
			Profile: testBinding("umpire.evidence-profile.basic"),
			Program: testBinding("umpire.observation-program.basic"),
			Mapping: testBinding("umpire.observation-mapping.basic"),
		},
	}
}

func testKnownGap() *umpirespb.KnownGap {
	return &umpirespb.KnownGap{
		Kind: umpirespb.KNOWN_GAP_KIND_CLAIM,
		Code: "umpire.gap.external-fixture", Subject: "umpire.plan.external.basic", Detail: "fixture gap",
	}
}

func testPhaseLimits() []*umpirespb.ExecutionPhaseLimit {
	return []*umpirespb.ExecutionPhaseLimit{
		{Phase: umpirespb.EXECUTION_PHASE_PREPARATION, DurationMilliseconds: 30_000, MaxAttempts: 1, MaxRecords: 128, MaxBytes: 1 << 20},
		{Phase: umpirespb.EXECUTION_PHASE_REALIZATION, DurationMilliseconds: 30_000, MaxAttempts: 1, MaxRecords: 128, MaxBytes: 1 << 20},
		{Phase: umpirespb.EXECUTION_PHASE_OBSERVATION, DurationMilliseconds: 30_000, MaxAttempts: 1, MaxRecords: 3_584, MaxBytes: 12 << 20},
		{Phase: umpirespb.EXECUTION_PHASE_ISOLATION, DurationMilliseconds: 15_000, MaxAttempts: 1, MaxRecords: 128, MaxBytes: 1 << 20},
		{Phase: umpirespb.EXECUTION_PHASE_CLEANUP, DurationMilliseconds: 15_000, MaxAttempts: 1, MaxRecords: 128, MaxBytes: 1 << 20},
	}
}

func testVerificationProgram() *umpirespb.VerificationProgram {
	profile := &umpirespb.EvidenceProfile{
		Definition: testBinding("umpire.evidence-profile.basic"), Version: 1,
		Sources: []*umpirespb.EvidenceSourceDeclaration{{SourceDefinitionId: "umpire.evidence.source.basic"}},
		Kinds: []*umpirespb.EvidenceKindDeclaration{{
			KindDefinitionId: "umpire.evidence.kind.basic", SourceDefinitionId: "umpire.evidence.source.basic",
			Fields: []*umpirespb.EvidenceFieldDeclaration{{
				FieldDefinitionId: "umpire.evidence.field.basic",
				ValueKind:         umpirespb.VALUE_KIND_TEXT,
				Disposition:       umpirespb.FIELD_DISPOSITION_KIND_RETAIN,
			}},
		}},
		Cardinalities: []*umpirespb.EvidenceCardinality{{
			KindDefinitionId: "umpire.evidence.kind.basic", Minimum: 0, Maximum: 1,
		}},
	}
	return &umpirespb.VerificationProgram{
		Evidence: profile,
		Observation: &umpirespb.ObservationProgram{
			Definition: testBinding("umpire.observation-program.basic"),
			Source:     &umpirespb.SourceLocation{Path: "observation.proto", Line: 1, Column: 1, Provenance: "fixture"},
			Mapping:    testBinding("umpire.observation-mapping.basic"), MappingVersion: 1,
			Profile: proto.CloneOf(profile),
			Emits: []*umpirespb.Emit{{
				DefinitionId:           "umpire.emit.basic",
				SourceKindDefinitionId: "umpire.evidence.kind.basic",
				OutputDefinition:       testBinding("umpire.observation.expected"),
				OutputKind:             umpirespb.DEFINITION_KIND_OBSERVATION,
				Coordinate: &umpirespb.ModelCoordinate{
					Field: umpirespb.TRACE_FIELD_OBSERVATION, Step: 1, Position: 1,
				},
				Condition: &umpirespb.ObservationExpression{Operator: &umpirespb.ObservationExpression_Present{
					Present: &umpirespb.Present{Operand: &umpirespb.ObservationExpression{
						Operator: &umpirespb.ObservationExpression_Field{Field: &umpirespb.EvidenceFieldReference{
							KindDefinitionId:  "umpire.evidence.kind.basic",
							FieldDefinitionId: "umpire.evidence.field.basic",
						}},
					}},
				}},
				Value: &umpirespb.ObservationExpression{Operator: &umpirespb.ObservationExpression_Field{
					Field: &umpirespb.EvidenceFieldReference{
						KindDefinitionId:  "umpire.evidence.kind.basic",
						FieldDefinitionId: "umpire.evidence.field.basic",
					},
				}},
			}},
		},
		TraceProjection: &umpirespb.VerificationProgram_DirectPlanTrace{
			DirectPlanTrace: &umpirespb.DirectPlanTrace{},
		},
		Properties: []*umpirespb.Property{{
			Definition: testBinding("umpire.property.basic"),
			Source:     &umpirespb.SourceLocation{Path: "property.proto", Line: 1, Column: 1, Provenance: "fixture"},
			Clauses: []*umpirespb.PropertyClause{{
				DefinitionId: "umpire.property.clause.basic",
				Provenance:   umpirespb.PROPERTY_CLAUSE_PROVENANCE_INPUT_OUTPUT,
				PerStepImplies: &umpirespb.PerStepImplies{
					Trigger: &umpirespb.Pattern{
						Field:      umpirespb.TRACE_FIELD_SELECTED_ACTION,
						Definition: testBinding("umpire.action.basic"),
						Operator:   &umpirespb.Pattern_EqualsText{EqualsText: &umpirespb.EqualsText{Value: "run"}},
					},
					Required: &umpirespb.Pattern{
						Field:      umpirespb.TRACE_FIELD_RESULTING_STATE,
						Definition: testBinding("umpire.state.done"),
						Operator:   &umpirespb.Pattern_EqualsText{EqualsText: &umpirespb.EqualsText{Value: "done"}},
					},
				},
			}},
		}},
		Decision: &umpirespb.DecisionPolicy{Kind: umpirespb.DECISION_POLICY_KIND_STRICT_V1},
	}
}

func testLimits() *umpirespb.PortableTestPlanLimits {
	return &umpirespb.PortableTestPlanLimits{
		Structural: &umpirespb.StructuralLimits{
			MaxPlanBytes: 1 << 20, MaxNestingDepth: 256, MaxCollectionItems: 10_000, MaxOperatorCount: 100_000,
		},
		Execution: &umpirespb.ExecutionLimits{
			MaxActions: 1, MaxFaults: 1, MaxPhaseAttempts: 1,
			MaxPhaseDurationMilliseconds: 30_000, MaxTotalDurationMilliseconds: 120_000,
		},
		Evidence: &umpirespb.EvidenceLimits{MaxRecords: 100_000, MaxBytes: 16 << 20, MaxSources: 10_000},
		Evaluation: &umpirespb.PortableEvaluationLimits{
			MaxExpressionDepth: 64, MaxNatural: "18446744073709551615", MaxWork: 10_000_000,
		},
		Output: &umpirespb.OutputLimits{MaxDiagnosticBytes: 64 << 10, MaxResultBytes: 4 << 20},
	}
}
