//go:build test_dep && integration

package tests

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.temporal.io/server/tests/testcore"
	"go.temporal.io/server/tools/umpire/artifact"
	umpireruntime "go.temporal.io/server/tools/umpire/runtime"
	"go.temporal.io/server/tools/umpire/temporal/local"
	"go.temporal.io/server/tools/umpire/temporal/nexus"
)

func newUmpireTestEnvironment(
	t *testing.T,
) (*testcore.TestEnv, umpireruntime.EnvironmentFactory) {
	t.Helper()
	env := testcore.NewEnv(t, testcore.WithInMemorySQLitePersistence())
	factory, err := local.NewAttachedFactory(testEnvAuthority{
		client:    env.SdkClient(),
		namespace: env.Namespace().String(),
		endpoint:  env.FrontendGRPCAddress(),
	})
	require.NoError(t, err)
	return env, factory
}

func newUmpireNexusBinding(
	t *testing.T,
	factory umpireruntime.EnvironmentFactory,
) nexus.Binding {
	t.Helper()
	binding, err := nexus.NewBinding(factory)
	require.NoError(t, err)
	return binding
}

func loadUmpireCallerClosureInputSet(t *testing.T, name string) artifact.AdmittedSet {
	t.Helper()
	files := make(map[string][]byte, 3)
	for _, relative := range []string{
		"manifest.json",
		"artifacts/experiment.json",
		"artifacts/runtime-configuration.json",
	} {
		files[relative] = loadUmpireCallerClosureArtifact(t, name, relative)
	}
	admitted, err := artifact.AdmitSetFiles(files)
	require.NoError(t, err)
	return admitted
}

func loadUmpireCallerClosureArtifact(t *testing.T, name string, relative string) []byte {
	t.Helper()
	encoded, err := os.ReadFile(filepath.Join(
		"..", "tools", "umpire", "temporal", "nexus", "testdata", name,
		filepath.FromSlash(relative),
	))
	require.NoError(t, err)
	return encoded
}

func umpireTestArtifactChecksum(t *testing.T, domain string, value any) string {
	t.Helper()
	var encoded bytes.Buffer
	encoder := json.NewEncoder(&encoded)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	require.NoError(t, encoder.Encode(value))
	hasher := sha256.New()
	_, err := hasher.Write([]byte(domain + "\n"))
	require.NoError(t, err)
	_, err = hasher.Write(encoded.Bytes())
	require.NoError(t, err)
	return "sha256:" + hex.EncodeToString(hasher.Sum(nil))
}

type testEnvAuthority struct {
	client    client.Client
	namespace string
	endpoint  string
}

func (a testEnvAuthority) SDKClient() client.Client { return a.client }
func (a testEnvAuthority) Namespace() string        { return a.namespace }
func (a testEnvAuthority) Endpoint() string         { return a.endpoint }
