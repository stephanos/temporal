package adapterregen

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestRunCheckoutRevalidation(t *testing.T) {
	for _, test := range []struct {
		name    string
		want    string
		symlink bool
	}{
		{name: "unchanged-regular"},
		{name: "same-byte-regular-replacement"},
		{name: "changed-regular", want: generatedFile + " changed"},
		{name: "same-byte-symlink-replacement", want: generatedFile + " changed", symlink: true},
		{name: "new-symlink", want: "toolchain/version/added.txt appeared", symlink: true},
		{name: "initial-symlink", want: "checkout file " + filepath.FromSlash(generatedFile) + " is not a regular file", symlink: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := newFixture(t)
			review := fixture.dryRun(goodVersion)
			original := fixture.read(generatedFile)
			path := filepath.Join(fixture.root, filepath.FromSlash(generatedFile))
			external := filepath.Join(t.TempDir(), "external.txt")
			if err := os.WriteFile(external, []byte(original), 0o644); err != nil {
				t.Fatal(err)
			}
			if test.symlink {
				if err := os.Symlink(external, filepath.Join(t.TempDir(), "probe")); err != nil {
					t.Skipf("host cannot create a real symlink: %v", err)
				}
			}
			replaceWithSymlink := func() error {
				if err := os.Remove(path); err != nil {
					return err
				}
				return os.Symlink(external, path)
			}
			if test.name == "initial-symlink" {
				if err := replaceWithSymlink(); err != nil {
					t.Fatal(err)
				}
			}
			before := fixture.snapshot()
			spec := fixture.spec(goodVersion, review.Regeneration.ApprovalSHA256)
			spec.Verifiers = [][]string{{"test", "-s", generatedFile}}
			staged := false
			publishing := false
			spec.beforeApplyFile = func(int) error {
				publishing = true
				return nil
			}
			spec.afterStage = func() error {
				staged = true
				switch test.name {
				case "same-byte-regular-replacement":
					previous, err := os.Lstat(path)
					if err != nil {
						return err
					}
					replacement := path + ".replacement"
					if err := os.WriteFile(replacement, []byte(original), 0o644); err != nil {
						return err
					}
					if err := os.Rename(replacement, path); err != nil {
						return err
					}
					current, err := os.Lstat(path)
					if err != nil {
						return err
					}
					if !current.Mode().IsRegular() || os.SameFile(previous, current) {
						t.Fatal("same-byte regular replacement did not replace the original file")
					}
				case "changed-regular":
					if err := os.WriteFile(path, []byte("user edit\n"), 0o644); err != nil {
						return err
					}
				case "same-byte-symlink-replacement":
					if err := replaceWithSymlink(); err != nil {
						return err
					}
				case "new-symlink":
					if err := os.Symlink(external, filepath.Join(fixture.root, "toolchain", "version", "added.txt")); err != nil {
						return err
					}
				}
				before = fixture.snapshot()
				return nil
			}
			result, err := Run(t.Context(), spec)
			if staged != (test.name != "initial-symlink") {
				t.Errorf("afterStage reached = %v", staged)
			}
			if test.want == "" {
				if err != nil || !result.Applied || len(result.Published) == 0 {
					t.Fatalf("regular checkout publication = %+v, %v", result, err)
				}
				for _, relative := range []string{"deterministicio/sentry_adapter.go", "deterministicio/testdata/sentry/go.mod", "deterministicio/testdata/sentry/go.sum", "toolchain/version/version.json", generatedFile} {
					if !slices.Contains(result.Published, relative) {
						t.Errorf("published %v, want %s", result.Published, relative)
					}
				}
				descriptor := fixture.read("toolchain/version/version.json")
				if generated := fixture.read(generatedFile); generated == original || !strings.HasPrefix(generated, strings.TrimPrefix(digest([]byte(descriptor)), "sha256:")) {
					t.Errorf("generated output %q does not match the published descriptor", generated)
				}
			} else {
				if err == nil || !strings.HasSuffix(err.Error(), test.want) {
					t.Errorf("checkout refusal = %v, want %q", err, test.want)
				}
				if test.name != "initial-symlink" {
					var blocked *BlockedError
					if !errors.As(err, &blocked) {
						t.Errorf("checkout drift lost BlockedError classification: %v", err)
					}
				}
				if result.Applied || len(result.Published) != 0 || len(result.Staged) != 0 || publishing {
					t.Errorf("refused checkout published: applied = %v, paths = %v, staged = %d, publication started = %v", result.Applied, result.Published, len(result.Staged), publishing)
				}
			}
			requireNoJournal(t, fixture.root)
			if test.symlink {
				link := path
				if test.name == "new-symlink" {
					link = filepath.Join(fixture.root, "toolchain", "version", "added.txt")
				}
				if target, err := os.Readlink(link); err != nil || target != external {
					t.Errorf("user symlink was changed: target = %q, error = %v", target, err)
				}
			}
			if contents, err := os.ReadFile(external); err != nil || string(contents) != original {
				t.Errorf("external file was changed: contents = %q, error = %v", contents, err)
			}
			if test.want != "" {
				requireUnchanged(t, before, fixture.snapshot())
			}
		})
	}
}
