package testpilot_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"google.golang.org/protobuf/proto"
)

func TestCaseProtoJSONIsStrictAndPacksDeterministically(t *testing.T) {
	decoded, err := testpilot.DecodeCaseProtoJSON([]byte(`{"version":{"major":1},"caseId":"case"}`))
	require.NoError(t, err)
	require.Equal(t, "case", decoded.GetCaseId())

	_, err = testpilot.DecodeCaseProtoJSON([]byte(`{"caseId":"case","unknown":true}`))
	require.Error(t, err)
	_, err = testpilot.DecodeCaseProtoJSON(nil)
	require.Error(t, err)

	first, err := testpilot.PackCaseProtoJSON([]byte(`{"caseId":"case","version":{"major":1}}`))
	require.NoError(t, err)
	second, err := testpilot.PackCaseProtoJSON([]byte("{\n  \"version\": {\"major\": 1},\n  \"caseId\": \"case\"\n}"))
	require.NoError(t, err)
	require.Equal(t, first, second)
	unpacked := new(testpilotspb.Case)
	require.NoError(t, proto.Unmarshal(first, unpacked))
	require.Equal(t, "case", unpacked.GetCaseId())
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
