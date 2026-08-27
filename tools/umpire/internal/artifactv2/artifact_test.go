package artifactv2

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodeExperimentAcceptsCanonicalSwitchAndNexusV2(t *testing.T) {
	for _, relative := range []string{
		"model/Umpire/Examples/testdata/switch-experiment-spec.json",
		"model/Temporal/Feature/Nexus/Experimental/testdata/nexus-caller-closure-experiment-spec.json",
	} {
		t.Run(filepath.Base(relative), func(t *testing.T) {
			document, err := DecodeExperiment(readRepositoryFile(t, relative))
			require.NoError(t, err)
			require.Equal(t, ExperimentFormat, document.FormatVersion)
			require.Equal(t, DrivePlanFormat, document.Plan.FormatVersion)
		})
	}
}

func TestDecodeExperimentClassifiesV1BeforeV2Fields(t *testing.T) {
	encoded := []byte(`{"formatVersion":"umpire-experiment/v1","semanticIdentity":"legacy","plan":null}` + "\n")
	_, err := DecodeExperiment(encoded)
	require.EqualError(t, err, `unsupported format "umpire-experiment/v1"`)
}

func TestDecodeExperimentRejectsNoncanonicalEncodings(t *testing.T) {
	canonical := readRepositoryFile(t, "model/Umpire/Examples/testdata/switch-experiment-spec.json")
	oneLine := bytes.TrimSuffix(canonical, []byte{'\n'})
	reordered := bytes.Replace(
		oneLine,
		[]byte(`{"formatVersion":"umpire-experiment/v2","queryBehaviorFingerprint":`),
		[]byte(`{"queryBehaviorFingerprint":`),
		1,
	)
	reordered = bytes.Replace(
		reordered,
		[]byte(`,"plan":`),
		[]byte(`,"formatVersion":"umpire-experiment/v2","plan":`),
		1,
	)
	pretty := bytes.Replace(oneLine, []byte(`,"queryBehaviorFingerprint"`), []byte(",\n  \"queryBehaviorFingerprint\""), 1)
	alternateEscape := bytes.Replace(oneLine, []byte("switch.query.exact-action"), []byte(`switch.query.exact\u002daction`), 1)
	exponent := bytes.Replace(oneLine, []byte(`"position":1`), []byte(`"position":1e0`), 1)
	legacyKey := bytes.Replace(oneLine, []byte(`"queryDefinitionId"`), []byte(`"queryIdentity"`), 1)
	unknownKey := bytes.Replace(oneLine, []byte(`{"formatVersion":`), []byte(`{"unknown":true,"formatVersion":`), 1)
	caseCollision := bytes.Replace(oneLine, []byte(`"queryDefinitionId"`), []byte(`"QueryDefinitionId"`), 1)
	duplicateKey := bytes.Replace(
		oneLine,
		[]byte(`"queryBehaviorFingerprint":`),
		[]byte(`"queryBehaviorFingerprint":"sha256:0000000000000000000000000000000000000000000000000000000000000000","queryBehaviorFingerprint":`),
		1,
	)
	malformedFingerprint := bytes.Replace(oneLine, []byte("sha256:d915"), []byte("sha256:D915"), 1)
	malformedChecksum := bytes.Replace(oneLine, []byte("sha256:9533fdb58edf1ef3702c9f909ea62a3546d65d0bf864e1a224706bb18925d984"), []byte("sha256:1234"), 1)

	cases := map[string][]byte{
		"reordered object fields":        append(reordered, '\n'),
		"leading whitespace":             append([]byte{' '}, canonical...),
		"trailing whitespace":            append(append([]byte(nil), oneLine...), ' ', '\n'),
		"pretty whitespace":              append(pretty, '\n'),
		"missing terminal LF":            oneLine,
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
			[]byte("sha256:bfa6866e94636af51a7c0cc39b8637a896b2866c3e7f0214395f0d0d803a2d72"),
			[]byte("sha256:afa6866e94636af51a7c0cc39b8637a896b2866c3e7f0214395f0d0d803a2d72"), 1), want: "nested"},
		"outer": {encoded: bytes.Replace(canonical,
			[]byte("sha256:9533fdb58edf1ef3702c9f909ea62a3546d65d0bf864e1a224706bb18925d984"),
			[]byte("sha256:8533fdb58edf1ef3702c9f909ea62a3546d65d0bf864e1a224706bb18925d984"), 1), want: "ExperimentSpec"},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := DecodeExperiment(test.encoded)
			require.ErrorContains(t, err, test.want)
		})
	}
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
			document.Plan.ExpandedLimits.Behavior.Transitions.Value = 0
			document.Plan.ExpandedLimits.Behavior.SelectedActions.Value = 0
			document.Plan.ExpandedLimits.Search.Value = 0
			document.Plan.ModelOutcomes = []ModelValue{}
			document.Plan.LinearExtension[0].Position = 0
			document.Plan.Checkpoints[0].Transition = 0
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
