package casefile

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCanonicalAcceptsTheCompactAndPersistedFormsOnly(t *testing.T) {
	compact := []byte(`{"caseId":"c","program":{"programId":"p","roles":[]}}`)
	persisted, err := Persisted(compact)
	require.NoError(t, err)
	require.JSONEq(t, "{\n  \"caseId\": \"c\",\n  \"program\": {\n    \"programId\": \"p\",\n    \"roles\": []\n  }\n}\n", string(persisted))
	for name, input := range map[string][]byte{
		"compact":              compact,
		"compact with newline": append(append([]byte(nil), compact...), '\n'),
		"persisted":            persisted,
	} {
		recovered, err := Canonical(input)
		require.NoError(t, err, name)
		require.Equal(t, string(compact), string(recovered), name)
	}
	for name, input := range map[string][]byte{
		"four-space indent":         []byte("{\n    \"caseId\": \"c\"\n}\n"),
		"two newlines":              append(append([]byte(nil), compact...), '\n', '\n'),
		"spaced":                    []byte(`{"caseId": "c"}`),
		"persisted without newline": persisted[:len(persisted)-1],
	} {
		_, err := Canonical(input)
		require.ErrorIs(t, err, ErrNoncanonical, name)
	}
	_, err = Canonical([]byte("{not json"))
	require.Error(t, err)
	require.NotErrorIs(t, err, ErrNoncanonical)
	again, err := Persisted(persisted)
	require.NoError(t, err)
	require.Equal(t, persisted, again, "the persisted form is idempotent")
}
