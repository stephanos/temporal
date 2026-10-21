package temporal

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func coverageFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"go.mod": "module go.temporal.io/server\n",
		"chasm/lib/activity/frontend.go": `package activity
import helpers "go.temporal.io/server/common/helpers"
type wrapped struct { *helpers.Validator }
type unknownValidator interface { Validate() }
type frontendHandler struct { validator *wrapped; unknown unknownValidator }
func (h *frontendHandler) StartActivityExecution() { wrapper(); h.validator.Check(); h.validateReceiver(); h.unknown.Validate() }
func (h *frontendHandler) DescribeActivityExecution() {}
func (h *frontendHandler) PollActivityExecution() {}
func (h *frontendHandler) RequestCancelActivityExecution() {}
func wrapper() { helpers.ExportedHelper() }
func (h *frontendHandler) validateReceiver() {}
func validateReceiver() {}
`,
		"common/helpers/validator.go": `package helpers
func ExportedHelper() { validateLeaf() }
func validateLeaf() {}
func DeclaredBehindTransport() { validateBehindTransport() }
func validateBehindTransport() {}
type Validator struct{}
func (*Validator) Check() {}
`,
		"service/frontend/handler.go": `package frontend
import "go.temporal.io/api/serviceerror"
type WorkflowHandler struct{}
func (*WorkflowHandler) PollActivityTaskQueue() {}
func (*WorkflowHandler) RespondActivityTaskCompleted() { serviceerror.NewInvalidArgument("inline") }
func (*WorkflowHandler) RespondActivityTaskFailed() {}
func (*WorkflowHandler) RespondActivityTaskCanceled() {}
`,
		"common/helpers/validator_test.go":              "package helpers\nfunc OnlyInTest() {}\n",
		".flow/tmp/copied/common/helpers/validator.go":  "package helpers\nfunc Removed() {}\n",
		".worktrees/copied/common/helpers/validator.go": "package helpers\nfunc Removed() {}\n",
	}
	for name, body := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestReverseCoverageRepository(t *testing.T) {
	uncovered, unresolved, err := ReverseCoverage("../../../..", []string{
		"go.temporal.io/server/chasm/lib/activity.validateAndNormalizeStartRequest",
		"go.temporal.io/server/chasm/lib/activity.handler.StartActivityExecution",
		"go.temporal.io/server/common/retrypolicy.Validate",
		"go.temporal.io/server/service/frontend.WorkflowHandler.RespondActivityTaskFailed",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{
		"go.temporal.io/server/common/tqid.NormalizeAndValidateUserDefined",
		"go.temporal.io/server/common/priorities.Validate",
		"go.temporal.io/server/common/links.Validator.ValidateRequestWithCallbacks",
		"go.temporal.io/server/common/primitives/timestamp.ValidateAndCapProtoDuration",
	} {
		if !slices.Contains(uncovered, target) {
			t.Fatalf("missing actual imported validation seam %s", target)
		}
	}
	t.Logf("actual repository: %d uncovered seams, %d unresolved seams", len(uncovered), len(unresolved))
	for _, seam := range unresolved {
		t.Log(seam)
	}
}

func TestReverseCoverageTracesDefinitionsAndInlineSeams(t *testing.T) {
	root := coverageFixture(t)
	covered := "go.temporal.io/server/chasm/lib/activity.frontendHandler.validateReceiver"
	uncovered, unresolved, err := ReverseCoverage(root, []string{covered})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"go.temporal.io/server/common/helpers.Validator.Check",
		"go.temporal.io/server/common/helpers.validateLeaf",
	} {
		if !slices.Contains(uncovered, want) {
			t.Fatalf("missing reached validator %s in %v", want, uncovered)
		}
	}
	if slices.Contains(uncovered, covered) || slices.Contains(uncovered, "go.temporal.io/server/chasm/lib/activity.validateReceiver") {
		t.Fatalf("receiver and free function identities conflated: %v", uncovered)
	}
	if !strings.Contains(strings.Join(uncovered, "\n"), "WorkflowHandler.RespondActivityTaskCompleted#inline:") {
		t.Fatalf("inline request check omitted: %v", uncovered)
	}
	if !strings.Contains(strings.Join(unresolved, "\n"), "h.unknown.Validate") {
		t.Fatalf("unresolved validator silently omitted: %v", unresolved)
	}
	if !slices.IsSorted(uncovered) || !slices.IsSorted(unresolved) {
		t.Fatal("coverage output is not sorted")
	}
}

func TestReverseCoverageRejectsVanishedOrTestOnlyTarget(t *testing.T) {
	root := coverageFixture(t)
	for _, target := range []string{"go.temporal.io/server/common/helpers.Removed", "go.temporal.io/server/common/helpers.OnlyInTest", "go.temporal.io/server/common/helpers.Other.Check"} {
		_, _, err := ReverseCoverage(root, []string{target})
		if err == nil || !strings.Contains(err.Error(), target) {
			t.Fatalf("missing target %q must fail with identity, got %v", target, err)
		}
	}
	if err := os.Remove(filepath.Join(root, "common/helpers/validator.go")); err != nil {
		t.Fatal(err)
	}
	_, _, err := ReverseCoverage(root, []string{"go.temporal.io/server/common/helpers.ExportedHelper"})
	if err == nil {
		t.Fatal("removed real target passed")
	}
}

func TestReverseCoverageRejectsPackageDirectoryAliases(t *testing.T) {
	for _, destination := range []string{".flow/tmp/alias-source", ".worktrees/alias-source", "outside"} {
		t.Run(destination, func(t *testing.T) {
			root := coverageFixture(t)
			original := filepath.Join(root, "common/helpers")
			copyPath := filepath.Join(root, destination)
			if destination == "outside" {
				copyPath = filepath.Join(t.TempDir(), "helpers")
			}
			if err := os.MkdirAll(filepath.Dir(copyPath), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.Rename(original, copyPath); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(copyPath, original); err != nil {
				t.Fatal(err)
			}
			target := "go.temporal.io/server/common/helpers.ExportedHelper"
			if _, _, err := ReverseCoverage(root, []string{target}); err == nil {
				t.Fatal("copied function satisfied a listed target through a package-directory alias")
			}
			_, unresolved, err := ReverseCoverage(root, nil)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(strings.Join(unresolved, "\n"), "package directory identity") {
				t.Fatalf("reached alias was not reported honestly: %v", unresolved)
			}
		})
	}
}

func TestReverseCoverageNormalizesRepositoryRootSymlink(t *testing.T) {
	root := coverageFixture(t)
	alias := filepath.Join(t.TempDir(), "repository")
	if err := os.Symlink(root, alias); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ReverseCoverage(alias, []string{"go.temporal.io/server/common/helpers.ExportedHelper"}); err != nil {
		t.Fatal(err)
	}
}

func TestReverseCoverageTracesListedTargetsBehindDynamicTransport(t *testing.T) {
	root := coverageFixture(t)
	uncovered, _, err := ReverseCoverage(root, []string{"go.temporal.io/server/common/helpers.DeclaredBehindTransport"})
	if err != nil {
		t.Fatal(err)
	}
	want := "go.temporal.io/server/common/helpers.validateBehindTransport"
	if !slices.Contains(uncovered, want) {
		t.Fatalf("listed target's helper was not traced behind dynamic transport: %v", uncovered)
	}
}
