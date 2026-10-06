package ir

// A Property's origin is inert: it traces a generated Property to the capability Property it was
// expanded from. That no fingerprint, answer, lowering or exploration identity reads it is held where
// those layers meet, in lower's origin identity test.

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/common/testing/protorequire"
	"google.golang.org/protobuf/proto"
)

// The reader admits a Property with an origin and one without; WithoutOrigins clears only the
// origins, and leaves its argument alone.
func TestWithoutOriginsClearsOnlyOrigins(t *testing.T) {
	plain, err := Load(filepath.Join("..", "..", "..", "model", "ir", "activity-standalone.json"))
	require.NoError(t, err)
	traced := proto.CloneOf(plain)
	require.NotEmpty(t, traced.GetProperties())
	for _, p := range traced.GetProperties() {
		require.Nil(t, p.GetOrigin(), p.GetName())
		p.Origin = &umpirespb.PropertyOrigin{Name: "temporal.capabilities.Traced." + p.GetName(), Position: proto.CloneOf(p.GetPosition())}
	}
	require.NoError(t, Validate(traced))
	given := proto.CloneOf(traced)
	protorequire.ProtoEqual(t, plain, WithoutOrigins(traced))
	protorequire.ProtoEqual(t, given, traced)
}
