package evaluationcontract

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

func TestPackProducesStableDeterministicContract(t *testing.T) {
	contract := testContract()
	canonicalJSON, err := CanonicalProtoJSON(contract)
	require.NoError(t, err)

	first, err := Pack(canonicalJSON)
	require.NoError(t, err)
	second, err := Pack(canonicalJSON)
	require.NoError(t, err)
	require.Equal(t, first, second)

	admitted, err := Admit(first)
	require.NoError(t, err)
	require.Len(t, admitted.GetArtifactChecksum(), 32)
	canonicalSealedJSON, err := CanonicalProtoJSON(admitted)
	require.NoError(t, err)
	repacked, err := Pack(canonicalSealedJSON)
	require.NoError(t, err)
	require.Equal(t, first, repacked)
}

func TestPackRejectsProtoJSONOutsideTheCanonicalVocabulary(t *testing.T) {
	contract := testContract()
	canonicalJSON, err := CanonicalProtoJSON(contract)
	require.NoError(t, err)

	unknownField := bytes.Replace(canonicalJSON, []byte("\n}"), []byte(",\n  \"futureField\": true\n}"), 1)
	_, err = Pack(unknownField)
	requireAdmissionCode(t, err, ErrorUnknownField)

	noncanonical := bytes.Replace(canonicalJSON, []byte("  \"contractId\""), []byte("    \"contractId\""), 1)
	_, err = Pack(noncanonical)
	requireAdmissionCode(t, err, ErrorNoncanonical)

	require.Equal(t, 1, bytes.Count(canonicalJSON, []byte("\"equals\"")))
	unsupportedOperator := bytes.Replace(canonicalJSON, []byte("\"equals\""), []byte("\"futureOperator\""), 1)
	_, err = Pack(unsupportedOperator)
	requireAdmissionCode(t, err, ErrorUnknownField)
}

func TestAdmitRejectsOneFieldStructuralMutations(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		code   ErrorCode
		mutate func(*umpirespb.EvaluationContract)
	}{
		{
			name: "missing identity",
			code: ErrorMalformedValue,
			mutate: func(contract *umpirespb.EvaluationContract) {
				contract.ContractId = ""
			},
		},
		{
			name: "unsupported major version",
			code: ErrorUnsupportedVersion,
			mutate: func(contract *umpirespb.EvaluationContract) {
				contract.Version.Major++
			},
		},
		{
			name: "invalid enum",
			code: ErrorUnsupportedEnum,
			mutate: func(contract *umpirespb.EvaluationContract) {
				contract.Observation.Emits[0].OutputKind = umpirespb.DefinitionKind(999)
			},
		},
		{
			name: "unsupported operator",
			code: ErrorUnsupportedOperator,
			mutate: func(contract *umpirespb.EvaluationContract) {
				contract.Observation.Emits[0].Condition = &umpirespb.ObservationExpression{}
			},
		},
		{
			name: "invalid limit",
			code: ErrorLimit,
			mutate: func(contract *umpirespb.EvaluationContract) {
				contract.Limits.MaxEvaluationWork = 0
			},
		},
		{
			name: "invalid application limit unit",
			code: ErrorLimit,
			mutate: func(contract *umpirespb.EvaluationContract) {
				contract.ImplementationLink.ApplicationLimit.Unit = " link-entries "
			},
		},
		{
			name: "crossed query binding",
			code: ErrorBinding,
			mutate: func(contract *umpirespb.EvaluationContract) {
				contract.Query.BehaviorFingerprint = testDigest('f')
			},
		},
		{
			name: "repeated fingerprint drift",
			code: ErrorBinding,
			mutate: func(contract *umpirespb.EvaluationContract) {
				contract.Properties[0].Clauses[0].PerStepImplies.Required.Definition.BehaviorFingerprint = testDigest('e')
			},
		},
		{
			name: "noncanonical collection order",
			code: ErrorOrdering,
			mutate: func(contract *umpirespb.EvaluationContract) {
				fields := contract.Observation.Profile.Kinds[0].Fields
				fields[0], fields[1] = fields[1], fields[0]
			},
		},
		{
			name: "duplicate identity",
			code: ErrorDuplicate,
			mutate: func(contract *umpirespb.EvaluationContract) {
				contract.Observation.Profile.Sources = append(
					contract.Observation.Profile.Sources,
					proto.CloneOf(contract.Observation.Profile.Sources[0]),
				)
			},
		},
		{
			name: "contradictory link mapping",
			code: ErrorDuplicate,
			mutate: func(contract *umpirespb.EvaluationContract) {
				entry := proto.CloneOf(contract.ImplementationLink.Entries[0])
				entry.Destination = testModelValue("zz.observation.delivery-count", testDigest('3'),
					umpirespb.DEFINITION_KIND_OBSERVATION,
					&umpirespb.Value{Value: &umpirespb.Value_Natural{Natural: "1"}})
				contract.ImplementationLink.Entries = append(contract.ImplementationLink.Entries, entry)
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			contract := proto.CloneOf(testContract())
			testCase.mutate(contract)
			encoded := encodeUnchecked(t, contract)

			_, err := Admit(encoded)
			requireAdmissionCode(t, err, testCase.code)
		})
	}
}

func TestAdmitRejectsInvalidCoordinateBoundaries(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		coordinate *umpirespb.ModelCoordinate
	}{
		{name: "initial state with step", coordinate: &umpirespb.ModelCoordinate{Field: umpirespb.TRACE_FIELD_INITIAL_STATE, Step: 1}},
		{name: "step field at zero", coordinate: &umpirespb.ModelCoordinate{Field: umpirespb.TRACE_FIELD_SELECTED_ACTION}},
		{name: "step field with position", coordinate: &umpirespb.ModelCoordinate{Field: umpirespb.TRACE_FIELD_SELECTED_ACTION, Step: 1, Position: 1}},
		{name: "observation at step zero", coordinate: &umpirespb.ModelCoordinate{Field: umpirespb.TRACE_FIELD_OBSERVATION, Position: 1}},
		{name: "observation at position zero", coordinate: &umpirespb.ModelCoordinate{Field: umpirespb.TRACE_FIELD_OBSERVATION, Step: 1}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			contract := testContract()
			contract.Observation.Emits[0].Coordinate = testCase.coordinate

			_, err := Admit(encodeUnchecked(t, contract))
			requireAdmissionCode(t, err, ErrorMalformedValue)
		})
	}
}

func TestAdmitRejectsCyclicEmitOrdering(t *testing.T) {
	acyclic := testContractWithThreeEmits()
	acyclic.Observation.Ordering = []*umpirespb.EmitOrdering{
		{PredecessorEmitDefinitionId: "system.emit.a", SuccessorEmitDefinitionId: "system.emit.b"},
		{PredecessorEmitDefinitionId: "system.emit.b", SuccessorEmitDefinitionId: "system.emit.delivery-count"},
	}
	_, err := Admit(packTestContract(t, acyclic))
	require.NoError(t, err)

	for _, testCase := range []struct {
		name     string
		ordering []*umpirespb.EmitOrdering
	}{
		{
			name: "two-node cycle",
			ordering: []*umpirespb.EmitOrdering{
				{PredecessorEmitDefinitionId: "system.emit.a", SuccessorEmitDefinitionId: "system.emit.b"},
				{PredecessorEmitDefinitionId: "system.emit.b", SuccessorEmitDefinitionId: "system.emit.a"},
			},
		},
		{
			name: "longer cycle",
			ordering: []*umpirespb.EmitOrdering{
				{PredecessorEmitDefinitionId: "system.emit.a", SuccessorEmitDefinitionId: "system.emit.b"},
				{PredecessorEmitDefinitionId: "system.emit.b", SuccessorEmitDefinitionId: "system.emit.delivery-count"},
				{PredecessorEmitDefinitionId: "system.emit.delivery-count", SuccessorEmitDefinitionId: "system.emit.a"},
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			contract := testContractWithThreeEmits()
			contract.Observation.Ordering = testCase.ordering

			_, admissionErr := Admit(encodeUnchecked(t, contract))
			requireAdmissionCode(t, admissionErr, ErrorOrdering)
		})
	}
}

func TestAdmitRejectsChecksumUnknownFieldAndNoncanonicalWire(t *testing.T) {
	canonical := packTestContract(t, testContract())

	var checksumDrift umpirespb.EvaluationContract
	require.NoError(t, proto.Unmarshal(canonical, &checksumDrift))
	checksumDrift.ArtifactChecksum[0] ^= 1
	checksumBytes, err := deterministicMarshal.Marshal(&checksumDrift)
	require.NoError(t, err)
	_, err = Admit(checksumBytes)
	requireAdmissionCode(t, err, ErrorChecksum)

	unknown := protowire.AppendTag(bytes.Clone(canonical), 1000, protowire.VarintType)
	unknown = protowire.AppendVarint(unknown, 1)
	_, err = Admit(unknown)
	requireAdmissionCode(t, err, ErrorUnknownField)

	noncanonical := protowire.AppendTag(bytes.Clone(canonical), 2, protowire.BytesType)
	noncanonical = protowire.AppendString(noncanonical, testContract().GetContractId())
	_, err = Admit(noncanonical)
	requireAdmissionCode(t, err, ErrorNoncanonical)
}

func TestAdmitEnforcesCollectionLimitAtNAndNPlusOne(t *testing.T) {
	atN := testContract()
	atN.Limits.MaxCollectionItems = 2
	_, err := Admit(packTestContract(t, atN))
	require.NoError(t, err)

	atNPlusOne := proto.CloneOf(atN)
	atNPlusOne.Observation.Profile.Kinds[0].Fields = append(
		atNPlusOne.Observation.Profile.Kinds[0].Fields,
		&umpirespb.EvidenceFieldDeclaration{
			FieldDefinitionId: "evidence.field.z-last",
			ValueKind:         umpirespb.VALUE_KIND_TEXT,
			Disposition:       umpirespb.FIELD_DISPOSITION_KIND_RETAIN,
		},
	)
	_, err = Admit(encodeUnchecked(t, atNPlusOne))
	requireAdmissionCode(t, err, ErrorLimit)
}

func testContract() *umpirespb.EvaluationContract {
	queryFingerprint := testDigest('1')
	countField := &umpirespb.EvidenceFieldReference{
		KindDefinitionId:  "evidence.kind.operation",
		FieldDefinitionId: "evidence.field.count",
	}
	statusField := &umpirespb.EvidenceFieldReference{
		KindDefinitionId:  "evidence.kind.operation",
		FieldDefinitionId: "evidence.field.status",
	}
	observationValue := testModelValue("system.observation.delivery-count", testDigest('9'),
		umpirespb.DEFINITION_KIND_OBSERVATION, &umpirespb.Value{Value: &umpirespb.Value_Natural{Natural: "1"}})
	linkedValue := testModelValue("feature.observation.delivery-count", testDigest('a'),
		umpirespb.DEFINITION_KIND_OBSERVATION, &umpirespb.Value{Value: &umpirespb.Value_Natural{Natural: "1"}})

	return &umpirespb.EvaluationContract{
		Version:          &umpirespb.FormatVersion{Major: 1, Minor: 0},
		ContractId:       "umpire.contract.caller-closure",
		ArtifactChecksum: nil,
		Experiment: &umpirespb.ArtifactBinding{
			FormatVersion:       "umpire-experiment/v2",
			ArtifactChecksum:    testDigest('2'),
			BehaviorFingerprint: queryFingerprint,
			ProvenanceChecksum:  testDigest('3'),
		},
		RuntimeConfig: &umpirespb.ArtifactBinding{
			FormatVersion:       "umpire-runtime-configuration/v2",
			ArtifactChecksum:    testDigest('4'),
			BehaviorFingerprint: testDigest('5'),
			ProvenanceChecksum:  testDigest('6'),
		},
		Test:  testBinding("umpire.test.caller-closure", '7'),
		Query: &umpirespb.DefinitionBinding{DefinitionId: "workflow-nexus.query.caller-closure", BehaviorFingerprint: queryFingerprint},
		Limits: &umpirespb.EvaluationLimits{
			MaxContractBytes:             MaximumContractBytes,
			MaxInputBytes:                MaximumInputBytes,
			MaxEvidenceRecords:           10,
			MaxExpressionDepth:           8,
			MaxCollectionItems:           4,
			MaxNatural:                   "100",
			MaxEvaluationWork:            1_000,
			MaxDiagnosticBytes:           1_024,
			MaxResultBytes:               16_384,
			MaxTotalDurationMilliseconds: 10_000,
		},
		Observation: &umpirespb.ObservationProgram{
			Definition:     testBinding("system.observation.caller-closure", '8'),
			Source:         testSourceLocation("model/Temporal/System/Nexus/Observation.lean", 10),
			Mapping:        testBinding("system.mapping.caller-closure", 'b'),
			MappingVersion: 1,
			Profile: &umpirespb.EvidenceProfile{
				Definition: testBinding("system.evidence-profile.caller-closure", 'c'),
				Version:    1,
				Sources: []*umpirespb.EvidenceSourceDeclaration{{
					SourceDefinitionId: "evidence.source.history",
				}},
				Kinds: []*umpirespb.EvidenceKindDeclaration{{
					KindDefinitionId:   "evidence.kind.operation",
					SourceDefinitionId: "evidence.source.history",
					Fields: []*umpirespb.EvidenceFieldDeclaration{
						{
							FieldDefinitionId: "evidence.field.count",
							ValueKind:         umpirespb.VALUE_KIND_NATURAL,
							Disposition:       umpirespb.FIELD_DISPOSITION_KIND_RETAIN,
						},
						{
							FieldDefinitionId: "evidence.field.status",
							ValueKind:         umpirespb.VALUE_KIND_TEXT,
							Disposition:       umpirespb.FIELD_DISPOSITION_KIND_RETAIN,
						},
					},
				}},
				Cardinalities: []*umpirespb.EvidenceCardinality{{
					KindDefinitionId: "evidence.kind.operation", Minimum: 1, Maximum: 2,
				}},
				CorrelationSlots: []*umpirespb.CorrelationSlot{{
					DefinitionId: "evidence.correlation.operation",
					Kind:         umpirespb.CORRELATION_SLOT_KIND_OPERATION,
					Fields:       []*umpirespb.EvidenceFieldReference{statusField},
				}},
			},
			Emits: []*umpirespb.Emit{{
				DefinitionId:           "system.emit.delivery-count",
				SourceKindDefinitionId: "evidence.kind.operation",
				OutputDefinition:       observationValue.Definition,
				OutputKind:             observationValue.Kind,
				Coordinate: &umpirespb.ModelCoordinate{
					Field: umpirespb.TRACE_FIELD_OBSERVATION, Step: 1, Position: 1,
				},
				Condition: &umpirespb.ObservationExpression{Operator: &umpirespb.ObservationExpression_Equals{
					Equals: &umpirespb.Equals{
						Left: &umpirespb.ObservationExpression{Operator: &umpirespb.ObservationExpression_Field{
							Field: statusField,
						}},
						Right: &umpirespb.ObservationExpression{Operator: &umpirespb.ObservationExpression_LiteralText{
							LiteralText: &umpirespb.LiteralText{Value: "closed"},
						}},
					},
				}},
				Value: &umpirespb.ObservationExpression{Operator: &umpirespb.ObservationExpression_Field{
					Field: countField,
				}},
			}},
		},
		ImplementationLink: &umpirespb.RenameExactLink{
			Definition:        testBinding("implementation-link.caller-closure", 'd'),
			Source:            testSourceLocation("model/Temporal/System/Nexus/ImplementationLink.lean", 20),
			SourceTarget:      testBinding("system.target.nexus", 'e'),
			DestinationTarget: testBinding("feature.target.nexus", 'f'),
			Entries: []*umpirespb.RenameExactEntry{{
				Source: observationValue, Destination: linkedValue,
			}},
			ApplicationLimit: &umpirespb.Limit{Value: 10, Unit: applicationLimitUnit},
		},
		Properties: []*umpirespb.Property{{
			Definition: testBinding("workflow-nexus.property.caller-closure", '0'),
			Source:     testSourceLocation("model/Temporal/Feature/Nexus/CallerClosure.lean", 30),
			Clauses: []*umpirespb.PropertyClause{{
				DefinitionId: "workflow-nexus.property.caller-closure.input-output",
				Provenance:   umpirespb.PROPERTY_CLAUSE_PROVENANCE_INPUT_OUTPUT,
				PerStepImplies: &umpirespb.PerStepImplies{
					Trigger: &umpirespb.Pattern{
						Field:      umpirespb.TRACE_FIELD_SELECTED_ACTION,
						Definition: testBinding("workflow-nexus.action.close", '1'),
						Operator: &umpirespb.Pattern_EqualsText{
							EqualsText: &umpirespb.EqualsText{Value: "close"},
						},
					},
					Required: &umpirespb.Pattern{
						Field:      umpirespb.TRACE_FIELD_OBSERVATION,
						Definition: linkedValue.Definition,
						Operator: &umpirespb.Pattern_NaturalAtMost{
							NaturalAtMost: &umpirespb.NaturalAtMost{Bound: "1"},
						},
					},
				},
			}},
		}},
		Provenance: []*umpirespb.SourceLocation{
			testSourceLocation("model/Temporal/Tool/PortableEvaluationContract.lean", 40),
		},
	}
}

func testContractWithThreeEmits() *umpirespb.EvaluationContract {
	contract := testContract()
	first := proto.CloneOf(contract.Observation.Emits[0])
	first.DefinitionId = "system.emit.a"
	first.Coordinate.Position = 2
	second := proto.CloneOf(contract.Observation.Emits[0])
	second.DefinitionId = "system.emit.b"
	second.Coordinate.Position = 3
	contract.Observation.Emits = []*umpirespb.Emit{first, second, contract.Observation.Emits[0]}
	return contract
}

func testBinding(definitionID string, fingerprintByte byte) *umpirespb.DefinitionBinding {
	return &umpirespb.DefinitionBinding{DefinitionId: definitionID, BehaviorFingerprint: testDigest(fingerprintByte)}
}

func testDigest(character byte) string {
	return "sha256:" + string(bytes.Repeat([]byte{character}, 64))
}

func testSourceLocation(path string, line int64) *umpirespb.SourceLocation {
	return &umpirespb.SourceLocation{Path: path, Line: line, Column: 1, Provenance: "handwritten"}
}

func testModelValue(
	definitionID string,
	fingerprint string,
	kind umpirespb.DefinitionKind,
	value *umpirespb.Value,
) *umpirespb.ModelValue {
	return &umpirespb.ModelValue{
		Definition: &umpirespb.DefinitionBinding{DefinitionId: definitionID, BehaviorFingerprint: fingerprint},
		Kind:       kind,
		Value:      value,
	}
}

func packTestContract(t *testing.T, contract *umpirespb.EvaluationContract) []byte {
	t.Helper()
	canonicalJSON, err := CanonicalProtoJSON(contract)
	require.NoError(t, err)
	encoded, err := Pack(canonicalJSON)
	require.NoError(t, err)
	return encoded
}

func encodeUnchecked(t *testing.T, contract *umpirespb.EvaluationContract) []byte {
	t.Helper()
	contract.ArtifactChecksum = nil
	checksum, err := expectedChecksum(contract)
	require.NoError(t, err)
	contract.ArtifactChecksum = checksum
	encoded, err := deterministicMarshal.Marshal(contract)
	require.NoError(t, err)
	return encoded
}

func requireAdmissionCode(t *testing.T, err error, expected ErrorCode) {
	t.Helper()
	require.Error(t, err)
	actual, ok := CodeOf(err)
	require.True(t, ok)
	require.Equal(t, expected, actual)
}
