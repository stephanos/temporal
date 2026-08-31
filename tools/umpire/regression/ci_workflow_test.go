package regression

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const hermeticCITestCommand = "mise exec -- go test -count=1 -tags test_dep ./tools/umpire/temporal/nexus/... -run '^TestHermeticCIPortability$'"

type ciWorkflow struct {
	Name        string                       `yaml:"name"`
	On          map[string]ciWorkflowTrigger `yaml:"on"`
	Permissions map[string]string            `yaml:"permissions"`
	Concurrency ciWorkflowConcurrency        `yaml:"concurrency"`
	Jobs        map[string]ciWorkflowJob     `yaml:"jobs"`
}

type ciWorkflowTrigger struct {
	Branches []string `yaml:"branches"`
}

type ciWorkflowConcurrency struct {
	Group            string `yaml:"group"`
	CancelInProgress bool   `yaml:"cancel-in-progress"`
}

type ciWorkflowJob struct {
	RunsOn         string           `yaml:"runs-on"`
	TimeoutMinutes int              `yaml:"timeout-minutes"`
	Steps          []ciWorkflowStep `yaml:"steps"`
}

type ciWorkflowStep struct {
	Name string         `yaml:"name"`
	Uses string         `yaml:"uses"`
	With map[string]any `yaml:"with"`
	Run  string         `yaml:"run"`
}

func TestHermeticCIWorkflowDelegatesToOrdinaryPinnedTest(t *testing.T) {
	repositoryRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	require.NoError(t, err)

	workflowBytes, err := os.ReadFile(filepath.Join(repositoryRoot, ".github", "workflows", "umpire.yml"))
	require.NoError(t, err)
	var workflow ciWorkflow
	decoder := yaml.NewDecoder(bytes.NewReader(workflowBytes))
	decoder.KnownFields(true)
	require.NoError(t, decoder.Decode(&workflow))

	require.Equal(t, ciWorkflow{
		Name: "Umpire",
		On: map[string]ciWorkflowTrigger{
			"pull_request": {},
			"push": {
				Branches: []string{"main", "stephanos/umpire"},
			},
		},
		Permissions: map[string]string{"contents": "read"},
		Concurrency: ciWorkflowConcurrency{
			Group:            "umpire-${{ github.head_ref || github.run_id }}",
			CancelInProgress: true,
		},
		Jobs: map[string]ciWorkflowJob{
			"portability": {
				RunsOn:         "ubuntu-24.04",
				TimeoutMinutes: 15,
				Steps: []ciWorkflowStep{
					{Uses: "actions/checkout@df4cb1c069e1874edd31b4311f1884172cec0e10"},
					{
						Uses: "actions/setup-go@4a3601121dd01d1626a1e23e37211e3254c1c06c",
						With: map[string]any{
							"go-version-file": "go.mod",
							"check-latest":    false,
							"cache":           false,
						},
					},
					{
						Uses: "jdx/mise-action@dba19683ed58901619b14f395a24841710cb4925",
						With: map[string]any{
							"version":           "2026.8.16",
							"sha256":            "cff4832ded79af2951e800bddcb5a22acac58630d765a2d062c1180680a0bb35",
							"working_directory": "model",
							"cache":             false,
						},
					},
					{Name: "Run the ordinary Umpire test", Run: hermeticCITestCommand},
				},
			},
		},
	}, workflow)

	command := exec.Command("make", "--no-print-directory", "-n", "umpire-check-regression")
	command.Dir = repositoryRoot
	dryRun, err := command.Output()
	require.NoError(t, err)
	normalizedDryRun := strings.Join(strings.Fields(strings.ReplaceAll(string(dryRun), "\\\n", " ")), " ")
	require.Equal(t, 1, strings.Count(normalizedDryRun, hermeticCITestCommand))
}
