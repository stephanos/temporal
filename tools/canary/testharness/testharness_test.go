//go:build canary_harness

package testharness

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/canary/authority"
	"go.temporal.io/server/tools/canary/policy"
	"google.golang.org/grpc/credentials/insecure"
)

func environment(values map[string]string) authority.Lookup {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}

// writePolicy writes the committed policy, edited, in its canonical form.
func writePolicy(t *testing.T, edit func(*policy.Policy)) string {
	t.Helper()
	canary, err := policy.Embedded()
	require.NoError(t, err)
	canary.EvaluationProfile = ProfileName
	canary.AuthorityClass = policy.AuthorityHarness
	canary.Coordinates = policy.DigestsOf("127.0.0.1:7233", "harness", "queue", "handler", "endpoint")
	if edit != nil {
		edit(canary)
	}
	encoded, err := json.MarshalIndent(canary, "", "  ")
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "policy.json")
	require.NoError(t, os.WriteFile(path, append(encoded, '\n'), 0o600))
	return path
}

func TestLoadPolicyReadsAHarnessPolicy(t *testing.T) {
	canary, profile, err := LoadPolicy(environment(map[string]string{VariablePolicy: writePolicy(t, nil)}))
	require.NoError(t, err)
	require.Equal(t, ProfileName, canary.EvaluationProfile)
	require.Equal(t, ProfileName, profile.Name)
	require.Equal(t, "test-cluster-harness", profile.Trust)
}

// A harness policy that names production's Evaluation Profile or authority class, or is missing
// or unreadable, is refused: no harness run can produce a production receipt.
func TestLoadPolicyRefusesAnythingButAHarnessPolicy(t *testing.T) {
	for name, path := range map[string]string{
		"production's Profile":        writePolicy(t, func(p *policy.Policy) { p.EvaluationProfile = "production-canary" }),
		"another Profile":             writePolicy(t, func(p *policy.Policy) { p.EvaluationProfile = "local-ephemeral" }),
		"production's authority":      writePolicy(t, func(p *policy.Policy) { p.AuthorityClass = policy.AuthorityProtectedWorkflow }),
		"a policy that does not read": filepath.Join(t.TempDir(), "absent.json"),
	} {
		t.Run(name, func(t *testing.T) {
			_, _, err := LoadPolicy(environment(map[string]string{VariablePolicy: path}))
			require.Error(t, err)
		})
	}
	_, _, err := LoadPolicy(environment(nil))
	require.ErrorContains(t, err, VariablePolicy)

	noncanonical := filepath.Join(t.TempDir(), "policy.json")
	encoded, err := os.ReadFile(writePolicy(t, nil))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(noncanonical, append([]byte(" "), encoded...), 0o600))
	_, _, err = LoadPolicy(environment(map[string]string{VariablePolicy: noncanonical}))
	require.Error(t, err, "the harness reads a policy as strictly as production does")
}

// The harness transport is plaintext and needs no credential; its Redactor knows every coordinate.
func TestAuthorityIsPlaintextWithoutACredential(t *testing.T) {
	loaded, err := Authority(environment(map[string]string{
		authority.VariableGRPC: "127.0.0.1:7233", authority.VariableNamespace: "harness", authority.VariableTaskQueue: "queue",
		authority.VariableHandlerQueue: "handler", authority.VariableEndpoint: "endpoint",
	}))
	require.NoError(t, err)
	require.Equal(t, insecure.NewCredentials().Info().SecurityProtocol, loaded.Transport.Credentials.Info().SecurityProtocol)
	require.Nil(t, loaded.Transport.ClientTLS)
	require.Equal(t, authority.Redacted+" "+authority.Redacted, loaded.Redactor.Redact("harness 127.0.0.1"))
	_, err = Authority(environment(nil))
	require.Error(t, err, "the coordinates are still required")
}

// A pause hook returns once its file exists; a hook for another phase does nothing.
func TestThePauseHookWaitsForItsFile(t *testing.T) {
	release := filepath.Join(t.TempDir(), "release")
	hooked := hook(environment(map[string]string{VariablePause: "leased:" + release}))
	hooked("run-opened")
	done := make(chan struct{})
	go func() {
		hooked("leased")
		close(done)
	}()
	select {
	case <-done:
		t.Fatal("the pause returned before its file existed")
	default:
	}
	require.NoError(t, os.WriteFile(release, nil, 0o600))
	<-done
}
