package execution

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClaimKindsRemainDistinct(t *testing.T) {
	claims := []ClaimKind{
		ClaimProved,
		ClaimBoundedSafe,
		ClaimCounterexample,
		ClaimConforming,
		ClaimViolating,
		ClaimUnsupported,
		ClaimInconclusive,
		ClaimEvidenceFailure,
	}
	seen := make(map[ClaimKind]struct{}, len(claims))
	for _, claim := range claims {
		_, duplicate := seen[claim]
		require.False(t, duplicate)
		seen[claim] = struct{}{}
	}
}
