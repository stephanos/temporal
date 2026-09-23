package canary_test

import (
	"os/exec"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	harnessPackage = "go.temporal.io/server/tools/canary/testharness"
	commandPackage = "go.temporal.io/server/tools/canary/cmd/umpire-canary"
	// plaintextPackage is gRPC's insecure transport, which only the harness build and tests use.
	plaintextPackage = "google.golang.org/grpc/credentials/insecure"
)

// goList runs `go list` from the repository root and returns its lines.
func goList(t *testing.T, arguments ...string) []string {
	t.Helper()
	command := exec.Command("go", append([]string{"list"}, arguments...)...)
	command.Dir = repositoryRoot(t)
	output, err := command.Output()
	require.NoError(t, err, "go list %s", strings.Join(arguments, " "))
	return strings.Split(strings.TrimSpace(string(output)), "\n")
}

// The untagged binary the protected workflow runs has no override path: it contains no harness
// package, and no untagged, non-test canary package builds a plaintext transport. The harness
// build does contain the harness, so the check is not vacuous.
func TestTheUntaggedBuildHasNoHarnessAndNoPlaintext(t *testing.T) {
	untagged := goList(t, "-deps", commandPackage)
	require.NotContains(t, untagged, harnessPackage)
	harness := goList(t, "-tags", "canary_harness", "-deps", commandPackage)
	require.Contains(t, harness, harnessPackage)

	for _, line := range goList(t, "-f", "{{.ImportPath}}|{{join .Imports \",\"}}", "./tools/canary/...") {
		name, imports, _ := strings.Cut(line, "|")
		require.False(t, slices.Contains(strings.Split(imports, ","), plaintextPackage),
			"%s builds a plaintext transport in the untagged build", name)
	}
	var importsPlaintext []string
	for _, line := range goList(t, "-tags", "canary_harness", "-f", "{{.ImportPath}}|{{join .Imports \",\"}}", "./tools/canary/...") {
		name, imports, _ := strings.Cut(line, "|")
		if slices.Contains(strings.Split(imports, ","), plaintextPackage) {
			importsPlaintext = append(importsPlaintext, name)
		}
	}
	require.Equal(t, []string{harnessPackage}, importsPlaintext, "only the harness builds a plaintext transport")
}
