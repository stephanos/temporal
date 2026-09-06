package testpilot_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	testpilotpb "go.temporal.io/server/api/testpilot/v1"
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
	unpacked := new(testpilotpb.Case)
	require.NoError(t, proto.Unmarshal(first, unpacked))
	require.Equal(t, "case", unpacked.GetCaseId())
}
