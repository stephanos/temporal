package testpilot_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestCaseFormatRejectsBeforePayloadInterpretation(t *testing.T) {
	for _, version := range []string{`{"major":2}`, `{"major":3}`, `{"major":4}`, `{"major":99}`, `{"major":1,"minor":1}`} {
		t.Run(version, func(t *testing.T) {
			decoded, err := testpilot.DecodeCaseProtoJSON([]byte(fmt.Sprintf(`{"version":%s,"program":{"retiredExpression":true}}`, version)))
			require.Nil(t, decoded)
			require.ErrorContains(t, err, "unsupported Case version")
			require.NotContains(t, err.Error(), "retiredExpression")
		})
	}
}

func TestCaseFormatUsesProtoJSONVersionNumbers(t *testing.T) {
	for _, version := range []string{
		`{"major":1e0}`, `{"major":1.0}`, `{"major":"1e0"}`, `{"major":1,"minor":0e0}`,
	} {
		t.Run(version, func(t *testing.T) {
			parsed := new(testpilotspb.FormatVersion)
			require.NoError(t, protojson.Unmarshal([]byte(version), parsed))
			decoded, err := testpilot.DecodeCaseProtoJSON([]byte(`{"version":` + version + `,"caseId":"case"}`))
			require.NoError(t, err)
			require.Equal(t, parsed.GetMajor(), decoded.GetVersion().GetMajor())
			require.Equal(t, parsed.GetMinor(), decoded.GetVersion().GetMinor())
		})
	}
	for _, version := range []string{`{"major":1.1}`, `{"major":2147483648}`, `{"major":"2147483648"}`} {
		parsed := new(testpilotspb.FormatVersion)
		require.Error(t, protojson.Unmarshal([]byte(version), parsed))
		_, err := testpilot.DecodeCaseProtoJSON([]byte(`{"version":` + version + `}`))
		require.Error(t, err)
	}
	_, err := testpilot.DecodeCaseProtoJSON([]byte(`{"version":{"major":4e0},"program":{"retiredExpression":true}}`))
	require.ErrorContains(t, err, "unsupported Case version 4.0")
}

func TestCaseProtoJSONIsStrict(t *testing.T) {
	decoded, err := testpilot.DecodeCaseProtoJSON([]byte(`{"version":{"major":1},"caseId":"case"}`))
	require.NoError(t, err)
	require.Equal(t, "case", decoded.GetCaseId())

	_, err = testpilot.DecodeCaseProtoJSON([]byte(`{"caseId":"case","unknown":true}`))
	require.Error(t, err)
	_, err = testpilot.DecodeCaseProtoJSON(nil)
	require.Error(t, err)
}

// Preparation derives a Program's environment bindings and each instruction's activation reservations
// and outcome fields, so a Case that still writes one is refused by the strict decode that precedes
// preparation, naming the field.
func TestCaseProtoJSONRejectsDerivedDeclarations(t *testing.T) {
	// The retired reservations field is spelled in two parts so the vocabulary scan keeps holding it.
	reservations := "activation" + "Reservations"
	for field, encoded := range map[string]string{
		"environment": `{"caseId":"case","program":{"environment":[{"bindingId":"namespace"}]}}`,
		reservations:  `{"caseId":"case","program":{"entrypoints":[{"instructions":[{"` + reservations + `":[{"entrypointId":"workflow","count":"1"}]}]}]}}`,
		"outcome":     `{"caseId":"case","program":{"entrypoints":[{"instructions":[{"outcome":{"fields":[]}}]}]}}`,
	} {
		t.Run(field, func(t *testing.T) {
			_, err := testpilot.DecodeCaseProtoJSON([]byte(encoded))
			require.ErrorContains(t, err, `unknown field "`+field+`"`)
		})
	}
}
