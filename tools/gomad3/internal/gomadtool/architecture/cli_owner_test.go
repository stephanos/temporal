package architecture

import "testing"

func TestGomadCommandFacadeHasExactCLIOwnership(t *testing.T) {
	const module = "go.temporal.io/server/tools/gomad3"
	for path, want := range map[string]string{
		"cmd/gomad":                     "cli",
		"cmd/gomad/api":                 "cli",
		"cmd/gomad/api/nested":          "",
		"cmd/gomad/apiextra":            "",
		"cmd/gomad/internal/cli":        "cli",
		"cmd/gomad/internal/cli/nested": "cli",
		"cmd/gomad/internal/other":      "",
		"cmd/gomadtool":                 "developer",
		"runner/backend":                "runner",
	} {
		if got := Owner(module, module+"/"+path); got != want {
			t.Errorf("Owner(%q)=%q want=%q", path, got, want)
		}
	}
}
