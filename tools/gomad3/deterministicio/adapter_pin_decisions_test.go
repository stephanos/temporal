package deterministicio

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAdapterRegistryPortablePinDecisions(t *testing.T) {
	registry := Default().definition.adapters
	for _, test := range []struct {
		name    string
		version string
		sum     string
		extra   string
		want    string
	}{
		{name: "exact", version: "v0.46.0"},
		{name: "version bump", version: "v0.47.0", sum: "h1:rVufXyMD3nGx09C8z0nSYlPQhvRWj0/iKMJHHeZ5yOw=", want: "unsupported github.com/getsentry/sentry-go version"},
		{name: "changed sum", version: "v0.46.0", sum: "h1:O5gORYN3OQb0jnU86eetnKVrCU2y7mRXrFqf9WWxgVs=", want: "target module sum for github.com/getsentry/sentry-go@v0.46.0 is missing or modified"},
		{name: "replacement", version: "v0.46.0", extra: "\nreplace github.com/getsentry/sentry-go => ./sentry\n", want: "already replaces github.com/getsentry/sentry-go"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			goMod := fmt.Sprintf("module example.test/pinimpact\n\ngo 1.27.0\n\nrequire github.com/getsentry/sentry-go %s\n%s", test.version, test.extra)
			goSum := "github.com/getsentry/sentry-go v0.46.0 h1:mbdDaarbUdOt9X+dx6kDdntkShLEX3/+KyOsVDTPDj0=\n"
			if test.sum != "" {
				goSum = "github.com/getsentry/sentry-go " + test.version + " " + test.sum + "\n"
			}
			for name, contents := range map[string]string{"go.mod": goMod, "go.sum": goSum} {
				if err := os.WriteFile(filepath.Join(root, name), []byte(contents), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			selected, _, err := registry.selected(root)
			if test.sum != "" && err == nil {
				_, err = requireAdapterSums(root, selected)
			}
			if test.want != "" {
				if !IsInvalidBuildAdapterConfiguration(err) || !strings.Contains(err.Error(), test.want) {
					t.Fatalf("registry selection = %v, want %q", err, test.want)
				}
			} else {
				if err != nil || len(selected) != 1 || selected[0].identity.Module != "github.com/getsentry/sentry-go" {
					t.Fatalf("registry selection = %+v, %v", selected, err)
				}
				if _, err := requireAdapterSums(root, selected); err != nil {
					t.Fatal(err)
				}
			}
			for name, want := range map[string]string{"go.mod": goMod, "go.sum": goSum} {
				got, err := os.ReadFile(filepath.Join(root, name))
				if err != nil || string(got) != want {
					t.Fatalf("%s changed: %q, %v", name, got, err)
				}
			}
		})
	}
}
