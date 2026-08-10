package gomadmain

import (
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/temporalio/gomad/internal/gomadtool"
)

func TestBuildOptionsApplyUserTags(t *testing.T) {
	flags := flag.NewFlagSet("test", flag.ContinueOnError)
	var options buildOptions
	options.register(flags)
	if err := flags.Parse([]string{"-race", "-tags=test_dep,integration"}); err != nil {
		t.Fatal(err)
	}

	cfg := gomadtool.BuildConfig{}
	options.apply(&cfg)
	if diff := cmp.Diff("gomad,test_dep,integration,race", cfg.PackageTags()); diff != "" {
		t.Errorf("configured package tags mismatch (-want +got):\n%s", diff)
	}
}

func TestTranslatedBuildFlagsIncludeUserTags(t *testing.T) {
	cfg := gomadtool.BuildConfig{
		UserTags: gomadtool.ParseBuildTags("test_dep,integration"),
	}

	if diff := cmp.Diff([]string{
		"-ldflags=-checklinkname=0",
		"-tags=linkname,test_dep,integration",
	}, translatedBuildFlags(cfg)); diff != "" {
		t.Errorf("translatedBuildFlags() mismatch (-want +got):\n%s", diff)
	}
}

func TestGroup(t *testing.T) {
	grouped := batchPackagesWithDifferentNames([]string{
		"hello",
		"github.com/bar",
		"github.com/foo/baz",
		"github.com/foo/bar",
		"github.com/foo/hello",
		"github.com/ok",
		"goodbye/hello",
	})

	if diff := cmp.Diff(grouped, [][]string{
		{
			"hello",
			"github.com/bar",
			"github.com/foo/baz",
			"github.com/ok",
		},
		{
			"github.com/foo/bar",
			"github.com/foo/hello",
		},
		{
			"goodbye/hello",
		},
	}); diff != "" {
		t.Error(diff)
	}
}

func TestConfigureGoBuildCacheDefault(t *testing.T) {
	unsetenv(t, "GOCACHE")
	modDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(modDir, "go.mod"), []byte("module test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	workDir := filepath.Join(modDir, "subdir")
	if err := os.Mkdir(workDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(workDir)

	if err := configureGoBuildCache("test"); err != nil {
		t.Fatal(err)
	}

	want := filepath.Join(modDir, ".gomad", "go-build")
	if got := os.Getenv("GOCACHE"); got != want {
		t.Fatalf("GOCACHE = %q, want %q", got, want)
	}
	output, err := exec.Command("go", "env", "GOCACHE").Output()
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(output)); got != want {
		t.Fatalf("child GOCACHE = %q, want %q", got, want)
	}
}

func TestConfigureGoBuildCachePreservesExplicitValue(t *testing.T) {
	want := filepath.Join(t.TempDir(), "custom-cache")
	t.Setenv("GOCACHE", want)
	t.Chdir(t.TempDir())

	if err := configureGoBuildCache("test"); err != nil {
		t.Fatal(err)
	}

	if got := os.Getenv("GOCACHE"); got != want {
		t.Fatalf("GOCACHE = %q, want %q", got, want)
	}
}

func TestConfigureGoBuildCacheRequiresModule(t *testing.T) {
	unsetenv(t, "GOCACHE")
	t.Chdir(t.TempDir())

	if err := configureGoBuildCache("test"); err == nil {
		t.Fatal("configureGoBuildCache() succeeded outside a Go module")
	}
	if _, ok := os.LookupEnv("GOCACHE"); ok {
		t.Fatal("configureGoBuildCache() set GOCACHE after failing")
	}
}

func TestConfigureGoBuildCacheCommands(t *testing.T) {
	modDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(modDir, "go.mod"), []byte("module test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(modDir)
	want := filepath.Join(modDir, ".gomad", "go-build")

	for _, command := range []string{"translate", "test", "build-tests", "debug", "prepare-selftest"} {
		t.Run(command, func(t *testing.T) {
			unsetenv(t, "GOCACHE")
			if err := configureGoBuildCache(command); err != nil {
				t.Fatal(err)
			}
			if got := os.Getenv("GOCACHE"); got != want {
				t.Fatalf("GOCACHE = %q, want %q", got, want)
			}
		})
	}
}

func TestConfigureGoBuildCacheIgnoresNonBuildCommands(t *testing.T) {
	unsetenv(t, "GOCACHE")
	t.Chdir(t.TempDir())

	if err := configureGoBuildCache("help"); err != nil {
		t.Fatalf("configureGoBuildCache(%q) failed: %v", "help", err)
	}
	if _, ok := os.LookupEnv("GOCACHE"); ok {
		t.Fatal("configureGoBuildCache() set GOCACHE for a non-build command")
	}
}

func unsetenv(t *testing.T, key string) {
	t.Helper()

	value, ok := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if ok {
			if err := os.Setenv(key, value); err != nil {
				t.Error(err)
			}
		} else if err := os.Unsetenv(key); err != nil {
			t.Error(err)
		}
	})
}
