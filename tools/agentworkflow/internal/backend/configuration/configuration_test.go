package configuration

import (
	"os"
	"strings"
	"testing"
)

func TestDigestBindsResolvedCommandAndSettings(t *testing.T) {
	first, err := Digest([]string{os.Args[0], "wrapper"}, map[string]any{"model": "one", "qualified": true})
	if err != nil {
		t.Fatal(err)
	}
	repeated, err := Digest([]string{os.Args[0], "wrapper"}, map[string]any{"model": "one", "qualified": true})
	if err != nil {
		t.Fatal(err)
	}
	changed, err := Digest([]string{os.Args[0], "wrapper"}, map[string]any{"model": "two", "qualified": true})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(first, "sha256:") || first != repeated || first == changed {
		t.Fatalf("digests = %q, %q, %q", first, repeated, changed)
	}
	if _, err := Digest([]string{"missing-agentworkflow-executable"}, struct{}{}); err == nil {
		t.Fatal("missing executable was accepted")
	}
}

func TestEnvironmentSelectsOnlyCommonAndDeclaredCredentialNames(t *testing.T) {
	t.Setenv("PATH", "/bin")
	t.Setenv("AGENTWORKFLOW_TEST_CREDENTIAL", "allowed")
	t.Setenv("AGENTWORKFLOW_UNDECLARED_SECRET", "secret")
	selected := Environment("AGENTWORKFLOW_TEST_CREDENTIAL", "AGENTWORKFLOW_TEST_CREDENTIAL", "bad=name")
	joined := strings.Join(selected, "\n")
	if !strings.Contains(joined, "PATH=/bin") || !strings.Contains(joined, "AGENTWORKFLOW_TEST_CREDENTIAL=allowed") {
		t.Fatalf("selected environment = %v", selected)
	}
	if strings.Contains(joined, "AGENTWORKFLOW_UNDECLARED_SECRET") || strings.Count(joined, "AGENTWORKFLOW_TEST_CREDENTIAL=") != 1 {
		t.Fatalf("environment was not minimal = %v", selected)
	}
}
