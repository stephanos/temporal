package testpilot

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// A scaled Profile is its own identity, so a Driver built from the unscaled spec cannot run a Case
// prepared under the scaled one; an unscaled Profile, at zero or 100%, fingerprints as it did
// before scales existed.
func TestBindingFingerprintCoversTheBoundScale(t *testing.T) {
	source, profile := facadeFixture(t)
	unbound, err := profile.BindingFingerprint()
	require.NoError(t, err)
	require.Empty(t, unbound)
	profile.BoundScale = 100
	hundred, err := profile.BindingFingerprint()
	require.NoError(t, err)
	require.Empty(t, hundred)

	scaled := profile.Snapshot()
	scaled.BoundScale = 150
	scaledFingerprint, err := scaled.BindingFingerprint()
	require.NoError(t, err)
	require.NotEmpty(t, scaledFingerprint, "a scale alone is an identity")
	prepared, err := Prepare(source, scaled)
	require.NoError(t, err)
	require.Equal(t, scaledFingerprint, prepared.Identity().Bindings)
	other := scaled.Snapshot()
	other.BoundScale = 200
	otherFingerprint, err := other.BindingFingerprint()
	require.NoError(t, err)
	require.NotEqual(t, scaledFingerprint, otherFingerprint)

	bound := profile.Snapshot()
	bound.EnvironmentBindings = []EnvironmentBinding{{ID: "alpha", Value: "v:1"}}
	bound.BoundScale = 0
	zero, err := bound.BindingFingerprint()
	require.NoError(t, err)
	bound.BoundScale = 100
	hundred, err = bound.BindingFingerprint()
	require.NoError(t, err)
	// TestBindingFingerprintGolden pins the unscaled bytes; zero and 100 must both keep them.
	require.Equal(t, zero, hundred)
	bound.BoundScale = 150
	boundScaled, err := bound.BindingFingerprint()
	require.NoError(t, err)
	require.NotEqual(t, zero, boundScaled)

	for _, scale := range []BoundScale{-1, 50, 99} {
		invalid := profile.Snapshot()
		invalid.BoundScale = scale
		_, err := invalid.BindingFingerprint()
		var preparation *PreparationError
		require.ErrorAs(t, err, &preparation)
		require.Equal(t, "profile.bound_scale", preparation.Path)
		_, err = Prepare(source, invalid)
		require.ErrorAs(t, err, &preparation)
		require.Equal(t, "profile.bound_scale", preparation.Path)
	}
}
