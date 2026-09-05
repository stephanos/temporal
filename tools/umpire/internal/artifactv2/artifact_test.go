package artifactv2

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodeExperimentAcceptsCanonicalSwitchAndNexusV2(t *testing.T) {
	for _, relative := range []string{
		"model/Umpire/Artifact/Tests/Fixtures/SwitchExperimentSpecV2.json",
		"model/Umpire/Examples/Fixtures/SwitchCompiledArtifact.json",
		"model/Umpire/Examples/testdata/switch-experiment-spec.json",
		"model/Temporal/Feature/Nexus/Fixtures/OperationsAsyncStartArtifact.json",
		"model/Temporal/Feature/Nexus/Fixtures/OperationsCancellationArtifact.json",
		"model/Temporal/Feature/Nexus/Fixtures/OperationsSuccessfulCompletionArtifact.json",
	} {
		t.Run(filepath.Base(relative), func(t *testing.T) {
			document, err := DecodeExperiment(readRepositoryFile(t, relative))
			require.NoError(t, err)
			require.Equal(t, ExperimentFormat, document.FormatVersion)
			require.Equal(t, DrivePlanFormat, document.Plan.FormatVersion)
		})
	}
}

func TestDecodeExperimentAcceptsLeanNaturalAboveUint64(t *testing.T) {
	encodedNatural := readRepositoryFile(t, "model/Umpire/Planning/Tests/Fixtures/NaturalAboveUint64.json")
	var natural Natural
	require.NoError(t, json.Unmarshal(bytes.TrimSuffix(encodedNatural, []byte{'\n'}), &natural))
	require.Equal(t, Natural("18446744073709551616"), natural)

	encoded, err := json.Marshal(natural)
	require.NoError(t, err)
	require.Equal(t, bytes.TrimSuffix(encodedNatural, []byte{'\n'}), encoded)

	document, err := DecodeExperiment(readRepositoryFile(t,
		"model/Umpire/Examples/testdata/switch-experiment-spec.json"))
	require.NoError(t, err)
	document.Plan.ExpandedLimits.Behavior.Transitions.Value = natural
	document, err = SealExperiment(document)
	require.NoError(t, err)
	canonical, err := CanonicalExperimentBytes(document)
	require.NoError(t, err)

	decoded, err := DecodeExperiment(canonical)
	require.NoError(t, err)
	require.Equal(t, natural, decoded.Plan.ExpandedLimits.Behavior.Transitions.Value)
}

func TestCanonicalExperimentBytesUsesStablePrettyJSON(t *testing.T) {
	var document Experiment
	require.NoError(t, json.Unmarshal(readRepositoryFile(t,
		"model/Umpire/Examples/testdata/switch-experiment-spec.json"), &document))

	canonical, err := CanonicalExperimentBytes(document)
	require.NoError(t, err)
	require.True(t, bytes.HasPrefix(canonical,
		[]byte("{\n  \"formatVersion\": \"umpire-experiment/v2\",\n")))
	require.True(t, bytes.HasSuffix(canonical, []byte("\n")))
	require.False(t, bytes.HasSuffix(canonical, []byte("\n\n")))
}

func TestCanonicalJSONEscapingMatchesLean(t *testing.T) {
	type escapingProbe struct {
		Value string `json:"value"`
	}

	canonical, err := encodeJSONLine(escapingProbe{Value: string([]rune{
		0, 1, 8, 9, 10, 11, 12, 13, 31, 34, 92, 0x03bb, 0x2028, 0x2029,
	})})
	require.NoError(t, err)
	const expected = "{\n  \"value\": \"\\u0000\\u0001\\b\\t\\n\\u000b\\f\\r\\u001f\\\"\\\\λ\\u2028\\u2029\"\n}\n"
	// This is a byte-spelling golden; semantic JSON equality would hide escaping drift.
	//nolint:testifylint
	require.Equal(t,
		expected,
		string(canonical),
	)
}

func TestExpectedChecksumsUseExactPrettyPreimages(t *testing.T) {
	var document Experiment
	require.NoError(t, json.Unmarshal(readRepositoryFile(t,
		"model/Umpire/Examples/testdata/switch-experiment-spec.json"), &document))

	planChecksum, err := ExpectedDrivePlanChecksum(document.Plan)
	require.NoError(t, err)
	require.Equal(t,
		"sha256:a695f9f6cc79ba49a721d1764519e2167b5fe66278666238c6da862b1a33b835",
		planChecksum,
	)
	document.Plan.ArtifactChecksum = planChecksum
	experimentChecksum, err := ExpectedExperimentChecksum(document)
	require.NoError(t, err)
	require.Equal(t,
		"sha256:ac3fde668a79ff0433106e28f8ec9579a36f9f7d0ab09845d01b563289b560fd",
		experimentChecksum,
	)
}

func TestDecodeExperimentClassifiesV1BeforeV2Fields(t *testing.T) {
	encoded := []byte(`{"formatVersion":"umpire-experiment/v1","semanticIdentity":"legacy","plan":null}` + "\n")
	_, err := DecodeExperiment(encoded)
	require.EqualError(t, err, `unsupported format "umpire-experiment/v1"`)
}

func TestDecodeExperimentRejectsNoncanonicalEncodings(t *testing.T) {
	canonical := readRepositoryFile(t, "model/Umpire/Examples/testdata/switch-experiment-spec.json")
	withoutTerminalLF := bytes.TrimSuffix(canonical, []byte{'\n'})
	lines := bytes.Split(withoutTerminalLF, []byte{'\n'})
	require.Greater(t, len(lines), 3)
	lines[1], lines[2] = lines[2], lines[1]
	reordered := bytes.Join(lines, []byte{'\n'})
	var compact bytes.Buffer
	require.NoError(t, json.Compact(&compact, canonical))
	compact.WriteByte('\n')
	differentIndentation := bytes.Replace(withoutTerminalLF,
		[]byte("  \"formatVersion\""), []byte("    \"formatVersion\""), 1)
	lineTrailingSpace := bytes.Replace(withoutTerminalLF, []byte("{\n"), []byte("{ \n"), 1)
	alternateEscape := bytes.Replace(withoutTerminalLF, []byte("switch.query.exact-action"), []byte(`switch.query.exact\u002daction`), 1)
	exponent := bytes.Replace(withoutTerminalLF, []byte(`"position": 1`), []byte(`"position": 1e0`), 1)
	legacyKey := bytes.Replace(withoutTerminalLF, []byte(`"queryDefinitionId"`), []byte(`"queryIdentity"`), 1)
	unknownKey := bytes.Replace(withoutTerminalLF, []byte("{\n  \"formatVersion\":"),
		[]byte("{\n  \"unknown\": true,\n  \"formatVersion\":"), 1)
	caseCollision := bytes.Replace(withoutTerminalLF, []byte(`"queryDefinitionId"`), []byte(`"QueryDefinitionId"`), 1)
	duplicateKey := bytes.Replace(
		withoutTerminalLF,
		[]byte(`  "queryBehaviorFingerprint": `),
		[]byte("  \"queryBehaviorFingerprint\": \"sha256:0000000000000000000000000000000000000000000000000000000000000000\",\n  \"queryBehaviorFingerprint\": "),
		1,
	)
	malformedFingerprint := bytes.Replace(withoutTerminalLF, []byte("sha256:c296"), []byte("sha256:C296"), 1)
	malformedChecksum := bytes.Replace(withoutTerminalLF, []byte("sha256:ac3fde668a79ff0433106e28f8ec9579a36f9f7d0ab09845d01b563289b560fd"), []byte("sha256:1234"), 1)

	cases := map[string][]byte{
		"reordered object fields":        append(reordered, '\n'),
		"leading whitespace":             append([]byte{' '}, canonical...),
		"trailing whitespace":            append(append([]byte(nil), withoutTerminalLF...), ' ', '\n'),
		"compact whitespace":             compact.Bytes(),
		"different indentation":          append(differentIndentation, '\n'),
		"line trailing space":            append(lineTrailingSpace, '\n'),
		"missing terminal LF":            withoutTerminalLF,
		"extra terminal LF":              append(append([]byte(nil), canonical...), '\n'),
		"alternate string escaping":      append(alternateEscape, '\n'),
		"alternate numeric encoding":     append(exponent, '\n'),
		"legacy key":                     append(legacyKey, '\n'),
		"unknown key":                    append(unknownKey, '\n'),
		"case-colliding key":             append(caseCollision, '\n'),
		"duplicate key":                  append(duplicateKey, '\n'),
		"trailing JSON data":             append(append([]byte(nil), canonical...), []byte("{}")...),
		"malformed JSON":                 []byte("{\n"),
		"malformed behavior fingerprint": append(malformedFingerprint, '\n'),
		"malformed artifact checksum":    append(malformedChecksum, '\n'),
	}
	for name, encoded := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := DecodeExperiment(encoded)
			require.Error(t, err)
		})
	}
}

func TestDecodeExperimentVerifiesNestedAndOuterChecksumsIndependently(t *testing.T) {
	canonical := readRepositoryFile(t, "model/Umpire/Examples/testdata/switch-experiment-spec.json")
	cases := map[string]struct {
		encoded []byte
		want    string
	}{
		"nested": {encoded: bytes.Replace(canonical,
			[]byte("sha256:a695f9f6cc79ba49a721d1764519e2167b5fe66278666238c6da862b1a33b835"),
			[]byte("sha256:2caad30cc09a2006600917465e4f9223529afbba7acf734c3a629b0e3723ba7d"), 1), want: "nested"},
		"outer": {encoded: bytes.Replace(canonical,
			[]byte("sha256:ac3fde668a79ff0433106e28f8ec9579a36f9f7d0ab09845d01b563289b560fd"),
			[]byte("sha256:d7fc19d59b8b97922df475596bc45022e97c19d051149aa0c9aabe82dff18179"), 1), want: "ExperimentSpec"},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := DecodeExperiment(test.encoded)
			require.ErrorContains(t, err, test.want)
		})
	}
}

func TestDecodeExperimentRejectsInvalidPersistedKnownGaps(t *testing.T) {
	canonical := readRepositoryFile(t, "model/Umpire/Examples/testdata/switch-experiment-spec.json")
	cases := map[string]struct {
		mutate func(*Experiment)
		reseal bool
		want   string
	}{
		"malformed": {
			mutate: func(document *Experiment) {
				invalid := "unnamespaced"
				document.Plan.KnownGaps[0].Subject = &invalid
			},
			reseal: true,
			want:   "known gap subject",
		},
		"reordered": {
			mutate: func(document *Experiment) {
				document.Plan.KnownGaps[0], document.Plan.KnownGaps[1] =
					document.Plan.KnownGaps[1], document.Plan.KnownGaps[0]
			},
			reseal: true,
			want:   "known gaps are not in canonical order",
		},
		"duplicate": {
			mutate: func(document *Experiment) {
				gaps := document.Plan.KnownGaps
				document.Plan.KnownGaps = append([]KnownGap{gaps[0], gaps[0]}, gaps[1:]...)
			},
			reseal: true,
			want:   "duplicate or conflicting known gap",
		},
		"conflicting": {
			mutate: func(document *Experiment) {
				gaps := document.Plan.KnownGaps
				conflicting := gaps[0]
				detail := "changed"
				conflicting.Detail = &detail
				document.Plan.KnownGaps = append([]KnownGap{gaps[0], conflicting}, gaps[1:]...)
			},
			reseal: true,
			want:   "duplicate or conflicting known gap",
		},
		"stale": {
			mutate: func(document *Experiment) {
				document.QueryBehaviorFingerprint =
					"sha256:0000000000000000000000000000000000000000000000000000000000000000"
			},
			reseal: true,
			want:   "query behavior fingerprint differs from nested plan",
		},
		"checksum inconsistent": {
			mutate: func(document *Experiment) {
				detail := "changed"
				document.Plan.KnownGaps[0].Detail = &detail
			},
			want: "nested plan artifact checksum mismatch",
		},
	}

	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			document, err := DecodeExperiment(canonical)
			require.NoError(t, err)
			test.mutate(&document)
			if test.reseal {
				document, err = SealExperiment(document)
				require.NoError(t, err)
			}
			encoded, err := CanonicalExperimentBytes(document)
			require.NoError(t, err)

			_, err = DecodeExperiment(encoded)
			require.ErrorContains(t, err, test.want)
		})
	}
}

func TestExperimentV2HooksPreserveDecodeExperimentContract(t *testing.T) {
	canonical := readRepositoryFile(t, "model/Umpire/Examples/testdata/switch-experiment-spec.json")
	document, err := DecodeExperiment(canonical)
	require.NoError(t, err)
	require.NoError(t, ValidateExperiment(document))
	require.NoError(t, ValidateExperimentClosure(document))
	require.NoError(t, VerifyExperimentChecksums(document))

	document.ArtifactChecksum = "sha256:d7fc19d59b8b97922df475596bc45022e97c19d051149aa0c9aabe82dff18179"
	hookErr := VerifyExperimentChecksums(document)
	require.Error(t, hookErr)
	mutated, err := CanonicalExperimentBytes(document)
	require.NoError(t, err)
	_, decodeErr := DecodeExperiment(mutated)
	require.EqualError(t, decodeErr, hookErr.Error())
}

func TestDecodeExperimentRejectsResealedMalformedV2Values(t *testing.T) {
	value := ModelValue{DefinitionID: "switch.state.power", Value: "off"}
	cases := map[string]struct {
		mutate func(*Experiment)
		want   string
	}{
		"selection reason enum": {
			mutate: func(document *Experiment) { document.Plan.SelectionReason = "arbitrary" },
			want:   "selection reason",
		},
		"transitions limit unit": {
			mutate: func(document *Experiment) { document.Plan.ExpandedLimits.Behavior.Transitions.Unit = "arbitrary" },
			want:   "behavior transitions limit unit",
		},
		"selected actions limit unit": {
			mutate: func(document *Experiment) { document.Plan.ExpandedLimits.Behavior.SelectedActions.Unit = "arbitrary" },
			want:   "behavior selected actions limit unit",
		},
		"search limit unit": {
			mutate: func(document *Experiment) { document.Plan.ExpandedLimits.Search.Unit = "arbitrary" },
			want:   "search limit unit",
		},
		"operand kind enum": {
			mutate: func(document *Experiment) { document.Plan.ModelPreconditions[0].Left.Kind = "arbitrary" },
			want:   "operand kind",
		},
		"role operand carries value": {
			mutate: func(document *Experiment) { document.Plan.ModelPreconditions[0].Left.Value = &value },
			want:   "role operand is malformed",
		},
		"value operand carries role": {
			mutate: func(document *Experiment) {
				document.Plan.ModelPreconditions[0].Right.DefinitionID = "switch.role.subject"
			},
			want: "value operand is malformed",
		},
		"value operand missing payload": {
			mutate: func(document *Experiment) { document.Plan.ModelPreconditions[0].Right.Value = nil },
			want:   "value operand is malformed",
		},
		"property requirements null": {
			mutate: func(document *Experiment) { document.Properties[0].RequirementDefinitionIDs = nil },
			want:   "requirement definition IDs must not be null",
		},
		"checkpoint observations null": {
			mutate: func(document *Experiment) { document.Plan.Checkpoints[0].Observations = nil },
			want:   "observations must not be null",
		},
		"symbolic role value kind enum": {
			mutate: func(document *Experiment) {
				document.Plan.SymbolicRoles = []Role{{DefinitionID: "switch.role.pending", ValueKind: "arbitrary"}}
			},
			want: "symbolic role value kind",
		},
		"precondition relation enum": {
			mutate: func(document *Experiment) { document.Plan.ModelPreconditions[0].Relation = "arbitrary" },
			want:   "model precondition relation",
		},
		"required definition ID": {
			mutate: func(document *Experiment) { document.Plan.InitialState.DefinitionID = "unnamespaced" },
			want:   "initial state definition ID",
		},
		"noncanonical bindings": {
			mutate: func(document *Experiment) {
				document.Plan.Bindings = append(document.Plan.Bindings, Binding{RoleDefinitionID: "aaa.role", Value: value})
			},
			want: "bindings are not in canonical order",
		},
		"noncanonical property requirements": {
			mutate: func(document *Experiment) {
				document.Properties[0].RequirementDefinitionIDs = []string{"z.requirement", "a.requirement"}
			},
			want: "property requirement definition IDs are not in canonical order",
		},
		"noncanonical observation requirements": {
			mutate: func(document *Experiment) {
				document.ObservationRequirementDefinitionIDs = []string{"z.observation", "a.observation"}
			},
			want: "observation requirement definition IDs are not in canonical order",
		},
		"duplicate capability requirements": {
			mutate: func(document *Experiment) {
				document.Plan.CapabilityRequirementDefinitionIDs = []string{"a.capability", "a.capability"}
			},
			want: "duplicate capability requirement definition ID",
		},
		"unnamespaced capability requirement": {
			mutate: func(document *Experiment) {
				document.Plan.CapabilityRequirementDefinitionIDs = []string{"unnamespaced"}
			},
			want: "capability requirement definition ID",
		},
		"non-ASCII property requirement": {
			mutate: func(document *Experiment) {
				document.Properties[0].RequirementDefinitionIDs = []string{"switch.réquirement"}
			},
			want: "property requirement definition ID",
		},
		"unnamespaced observation requirement": {
			mutate: func(document *Experiment) {
				document.ObservationRequirementDefinitionIDs = []string{"unnamespaced"}
			},
			want: "observation requirement definition ID",
		},
	}

	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			canonical := readRepositoryFile(t, "model/Umpire/Examples/testdata/switch-experiment-spec.json")
			document, err := DecodeExperiment(canonical)
			require.NoError(t, err)
			test.mutate(&document)
			sealed, err := SealExperiment(document)
			require.NoError(t, err)
			encoded, err := CanonicalExperimentBytes(sealed)
			require.NoError(t, err)

			_, err = DecodeExperiment(encoded)
			require.ErrorContains(t, err, test.want)
		})
	}
}

func TestDecodeExperimentAcceptsResealedLeanRecordValues(t *testing.T) {
	cases := map[string]func(*Experiment){
		"zero limits and independent trace lists": func(document *Experiment) {
			document.Plan.ExpandedLimits.Behavior.Transitions.Value = Natural("0")
			document.Plan.ExpandedLimits.Behavior.SelectedActions.Value = Natural("0")
			document.Plan.ExpandedLimits.Search.Value = Natural("0")
			document.Plan.ModelOutcomes = []ModelValue{}
			document.Plan.LinearExtension[0].Position = Natural("0")
			document.Plan.Checkpoints[0].Transition = Natural("0")
		},
		"record ordered roles and preconditions": func(document *Experiment) {
			document.Plan.SymbolicRoles = []Role{
				{DefinitionID: "z.role", ValueKind: "state"},
				{DefinitionID: "a.role", ValueKind: "action"},
			}
			first := document.Plan.ModelPreconditions[0]
			first.DefinitionID = "z.precondition"
			second := first
			second.DefinitionID = "a.precondition"
			document.Plan.ModelPreconditions = []Precondition{first, second}
		},
		"sorted bindings and properties retain duplicates": func(document *Experiment) {
			document.Plan.Bindings = append(document.Plan.Bindings, document.Plan.Bindings[0])
			document.Properties = append(document.Properties, document.Properties[0])
		},
	}

	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			canonical := readRepositoryFile(t, "model/Umpire/Examples/testdata/switch-experiment-spec.json")
			document, err := DecodeExperiment(canonical)
			require.NoError(t, err)
			mutate(&document)
			sealed, err := SealExperiment(document)
			require.NoError(t, err)
			encoded, err := CanonicalExperimentBytes(sealed)
			require.NoError(t, err)

			_, err = DecodeExperiment(encoded)
			require.NoError(t, err)
		})
	}
}

func TestGoSHA256MatchesLeanGoldens(t *testing.T) {
	require.Equal(t,
		"sha256:8c09aa7f7eec82e39e6f28406acc4f640dac30a2b3bf861acfaad8d701275870",
		BehaviorFingerprint([]byte(`{"definitionId":"example.target","behavior":"start->done"}`)),
	)
	require.Equal(t,
		"sha256:3f40af6e8524a50317e0e116514d05bae3a2aef6cdbf47acc8faf071e24a9a9b",
		derive(drivePlanChecksumDomain, []byte(`{"formatVersion":"umpire-drive-plan/v2","definitionId":"example.query"}`)),
	)
	require.NotEqual(t,
		derive(experimentChecksumDomain, []byte(`{"formatVersion":"umpire-drive-plan/v2","definitionId":"example.query"}`)),
		derive(drivePlanChecksumDomain, []byte(`{"formatVersion":"umpire-drive-plan/v2","definitionId":"example.query"}`)),
	)
}

func readRepositoryFile(t *testing.T, relative string) []byte {
	t.Helper()
	root := filepath.Clean(filepath.Join("..", "..", "..", ".."))
	encoded, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
	require.NoError(t, err)
	require.True(t, strings.HasSuffix(string(encoded), "\n"))
	return encoded
}
