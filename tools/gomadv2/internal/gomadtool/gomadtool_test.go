package gomadtool

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestBuildConfigTags(t *testing.T) {
	tests := []struct {
		name        string
		tags        string
		race        bool
		packageTags string
		buildTags   string
	}{
		{
			name:        "internal tags",
			packageTags: "gomad",
			buildTags:   "linkname",
		},
		{
			name:        "user tags",
			tags:        "test_dep,integration",
			packageTags: "gomad,test_dep,integration",
			buildTags:   "linkname,test_dep,integration",
		},
		{
			name:        "duplicates",
			tags:        "gomad, test_dep,test_dep,linkname",
			packageTags: "gomad,test_dep,linkname",
			buildTags:   "linkname,gomad,test_dep",
		},
		{
			name:        "race",
			tags:        "test_dep,race",
			race:        true,
			packageTags: "gomad,test_dep,race",
			buildTags:   "linkname,test_dep,race",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := BuildConfig{
				Race:     test.race,
				UserTags: ParseBuildTags(test.tags),
			}

			if diff := cmp.Diff(test.packageTags, cfg.PackageTags()); diff != "" {
				t.Errorf("PackageTags() mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.buildTags, cfg.BuildTags()); diff != "" {
				t.Errorf("BuildTags() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFindGomadExecutable(t *testing.T) {
	t.Run("environment override", func(t *testing.T) {
		t.Setenv("GOMADTOOL", "/path/to/gomad")

		path, err := findGomadExecutable()

		if err != nil {
			t.Fatal(err)
		}
		if path != "/path/to/gomad" {
			t.Fatalf("findGomadExecutable() = %q, want %q", path, "/path/to/gomad")
		}
	})

	t.Run("path", func(t *testing.T) {
		t.Setenv("GOMADTOOL", "")
		binDir := t.TempDir()
		binPath := filepath.Join(binDir, "gomad")
		if err := os.WriteFile(binPath, nil, 0o755); err != nil {
			t.Fatal(err)
		}
		t.Setenv("PATH", binDir)

		path, err := findGomadExecutable()

		if err != nil {
			t.Fatal(err)
		}
		if path != binPath {
			t.Fatalf("findGomadExecutable() = %q, want %q", path, binPath)
		}
	})

	t.Run("missing", func(t *testing.T) {
		t.Setenv("GOMADTOOL", "")
		t.Setenv("PATH", t.TempDir())

		_, err := findGomadExecutable()

		if err == nil {
			t.Fatal("findGomadExecutable() succeeded, want error")
		}
	})
}
