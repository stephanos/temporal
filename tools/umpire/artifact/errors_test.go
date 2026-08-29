package artifact

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStrictJSONErrorCodesAreStableThroughWrapping(t *testing.T) {
	cases := []struct {
		code     ErrorCode
		sentinel error
	}{
		{ErrorByteLimit, ErrByteLimit},
		{ErrorSyntax, ErrSyntax},
		{ErrorTokenLimit, ErrTokenLimit},
		{ErrorDepthLimit, ErrDepthLimit},
		{ErrorDuplicateKey, ErrDuplicateKey},
		{ErrorCaseCollision, ErrCaseCollision},
		{ErrorUnsupportedFormat, ErrUnsupportedFormat},
		{ErrorWrongFamily, ErrWrongFamily},
		{ErrorUnknownField, ErrUnknownField},
		{ErrorCollectionLimit, ErrCollectionLimit},
		{ErrorStringLimit, ErrStringLimit},
		{ErrorPayloadLimit, ErrPayloadLimit},
		{ErrorMalformedValue, ErrMalformedValue},
		{ErrorNoncanonical, ErrNoncanonical},
		{ErrorProvenanceChecksum, ErrProvenanceChecksum},
		{ErrorArtifactChecksum, ErrArtifactChecksum},
		{ErrorClosure, ErrClosure},
	}
	for _, test := range cases {
		t.Run(string(test.code), func(t *testing.T) {
			err := wrapAdmission(test.code, errors.New("detail"))
			require.ErrorIs(t, err, test.sentinel)
			code, ok := CodeOf(err)
			require.True(t, ok)
			require.Equal(t, test.code, code)
		})
	}
}
