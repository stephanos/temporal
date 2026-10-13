package gomad3sim

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStateInvariantFailureIdentityIsSeedIndependent(t *testing.T) {
	evidence := []OracleEvidence{{Label: "state", Value: []byte(`{"queue":2}`)}}
	first, err := StateInvariant("matching.queue-drained", false, evidence, 1024)
	require.NoError(t, err)
	second, err := StateInvariant("matching.queue-drained", false, evidence, 1024)
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.NotEmpty(t, first.FailureIdentity)
	require.NotEmpty(t, first.Identity)

	passed, err := StateInvariant("matching.queue-drained", true, evidence, 1024)
	require.NoError(t, err)
	require.Empty(t, passed.FailureIdentity)
	require.NotEqual(t, first.Identity, passed.Identity)
}

func TestSoftAssertionOracleRetainsStaticInstanceSignature(t *testing.T) {
	first, err := SoftAssertion("history/1", []string{"invalid state"}, false, false, 4096)
	require.NoError(t, err)
	require.False(t, first.Passed)
	require.True(t, first.SoftAssertion)
	require.NoError(t, validateOracleResult(first, 4096))
	other, err := SoftAssertion("history/2", []string{"invalid state"}, false, false, 4096)
	require.NoError(t, err)
	require.NotEqual(t, first.FailureIdentity, other.FailureIdentity)
	reported, err := SoftAssertion("history/1", []string{"invalid state"}, false, true, 4096)
	require.NoError(t, err)
	require.True(t, reported.Passed)
	require.True(t, reported.ReportOnly)
	require.NoError(t, validateOracleResult(reported, 4096))
	incomplete, err := SoftAssertion("history/1", nil, true, true, 4096)
	require.NoError(t, err)
	require.False(t, incomplete.Passed)
	require.True(t, incomplete.Inconclusive)
	require.NoError(t, validateOracleResult(incomplete, 4096))
}

func TestExactHistoryAndDuplicateLostOracles(t *testing.T) {
	expected := []HistoryOperation{
		{ID: "op-1", Actor: "client-a", Kind: "enqueue", Invocation: 1, Completion: 2, Input: []byte("a"), Output: []byte("ok")},
		{ID: "op-2", Actor: "worker-a", Kind: "complete", Invocation: 3, Completion: 4, Input: []byte("a"), Output: []byte("ok")},
	}
	actual := cloneHistoryOperations(expected)
	exact, err := ExactHistory("queue.history", expected, actual, 4096)
	require.NoError(t, err)
	require.True(t, exact.Passed)

	actual = append(actual, cloneHistoryOperation(actual[1]))
	actual[2].ID = "op-3"
	actual[2].Invocation = 5
	actual[2].Completion = 6
	exact, err = ExactHistory("queue.history", expected, actual, 4096)
	require.NoError(t, err)
	require.False(t, exact.Passed)
	require.NotEmpty(t, exact.FailureIdentity)

	delivery, err := NoDuplicateOrLost("queue.delivery", []string{"a", "b"}, []string{"a", "a"}, 4096)
	require.NoError(t, err)
	require.False(t, delivery.Passed)
	require.NotEmpty(t, delivery.FailureIdentity)
}

func TestEventualConvergenceAndOracleEvidenceBounds(t *testing.T) {
	result, err := EventualConvergence("replicas.converged", map[string][]byte{
		"history-a": []byte("state-7"),
		"history-b": []byte("state-7"),
	}, 1024)
	require.NoError(t, err)
	require.True(t, result.Passed)

	result, err = EventualConvergence("replicas.converged", map[string][]byte{
		"history-a": []byte("state-7"),
		"history-b": []byte("state-6"),
	}, 1024)
	require.NoError(t, err)
	require.False(t, result.Passed)

	_, err = StateInvariant("bounded", false, []OracleEvidence{{Label: "large", Value: make([]byte, 5)}}, 4)
	var capacityErr *CapacityError
	require.ErrorAs(t, err, &capacityErr)
	require.Equal(t, "oracle_evidence_bytes", capacityErr.Resource)
}

func TestHistoryValidationRejectsMutableOrAmbiguousOperations(t *testing.T) {
	tests := []HistoryOperation{
		{},
		{ID: "op", Actor: "client", Kind: "write", Invocation: 2, Completion: 1},
		{ID: "op", Actor: "client", Kind: "write", Invocation: 1, Completion: 2, Error: "failed", Output: []byte("ok")},
	}
	for _, operation := range tests {
		require.Error(t, ValidateHistory([]HistoryOperation{operation}, 1024))
	}
	require.Error(t, ValidateHistory([]HistoryOperation{
		{ID: "op", Actor: "client", Kind: "write", Invocation: 1, Completion: 2},
		{ID: "op", Actor: "client", Kind: "write", Invocation: 3, Completion: 4},
	}, 1024))
}
