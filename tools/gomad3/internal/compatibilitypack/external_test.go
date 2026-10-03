package compatibility

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const externalFixtureSource = "reflect2-go126"

// externalPackFixture rewrites an embedded pack under a new ID so it can stand
// in for a pack a downstream module reviewed and owns.
func externalPackFixture(t *testing.T, id string, mutate func([]byte) []byte) []byte {
	t.Helper()
	contents, err := packFiles.ReadFile("packs/" + externalFixtureSource + ".json")
	if err != nil {
		t.Fatal(err)
	}
	renamed := bytes.Replace(contents, []byte(`"id":"`+externalFixtureSource+`"`), []byte(`"id":"`+id+`"`), 1)
	if bytes.Equal(renamed, contents) {
		t.Fatal("external fixture did not rename the pack")
	}
	if mutate != nil {
		renamed = mutate(renamed)
	}
	return renamed
}

func writeExternalPack(t *testing.T, directory, name string, contents []byte) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(directory, name), contents, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestExternalPacksLoadNextToEmbeddedPacks(t *testing.T) {
	directory := t.TempDir()
	writeExternalPack(t, directory, "downstream-fixture.json", externalPackFixture(t, "downstream-fixture", nil))
	t.Setenv(ExternalPacksEnvironment, directory)

	packs, err := loadPacksV2()
	if err != nil {
		t.Fatal(err)
	}
	var external *ValidatedPack
	for index := range packs {
		if packs[index].pack.ID == "downstream-fixture" {
			external = &packs[index]
		}
	}
	if external == nil {
		t.Fatal("external pack was not loaded")
	}
	identity := []Identity{{ID: external.pack.ID, SHA256: external.digest}}
	if err := VerifyIdentities(identity); err != nil {
		t.Fatalf("VerifyIdentities(external) = %v", err)
	}

	// Replay and resume verify the recorded identity; without the directory the
	// pack is unknown and verification fails closed.
	t.Setenv(ExternalPacksEnvironment, "")
	if err := VerifyIdentities(identity); err == nil {
		t.Fatal("VerifyIdentities() accepted an external pack that is no longer available")
	}
}

func TestExternalPacksFailClosed(t *testing.T) {
	for name, test := range map[string]struct {
		setup func(t *testing.T) string
		want  string
	}{
		"relative directory": {
			setup: func(t *testing.T) string { return "packs" },
			want:  "absolute, clean path",
		},
		"missing directory": {
			setup: func(t *testing.T) string { return filepath.Join(t.TempDir(), "absent") },
			want:  "read external compatibility packs",
		},
		"collision with embedded pack": {
			setup: func(t *testing.T) string {
				directory := t.TempDir()
				contents, err := packFiles.ReadFile("packs/" + externalFixtureSource + ".json")
				if err != nil {
					t.Fatal(err)
				}
				writeExternalPack(t, directory, externalFixtureSource+".json", contents)
				return directory
			},
			want: "collides with an embedded pack",
		},
		"file name does not match ID": {
			setup: func(t *testing.T) string {
				directory := t.TempDir()
				writeExternalPack(t, directory, "other-name.json", externalPackFixture(t, "downstream-fixture", nil))
				return directory
			},
			want: "does not match ID",
		},
		"non-pack entry": {
			setup: func(t *testing.T) string {
				directory := t.TempDir()
				writeExternalPack(t, directory, "README.md", []byte("notes\n"))
				return directory
			},
			want: "entry README.md is invalid",
		},
		"admits an unadmittable capability": {
			setup: func(t *testing.T) string {
				directory := t.TempDir()
				writeExternalPack(t, directory, "downstream-fixture.json", externalPackFixture(t, "downstream-fixture", func(contents []byte) []byte {
					return bytes.Replace(contents, []byte(`"foreign:assembly:relfect2_ppc64x.s"],`), []byte(`"foreign:assembly:relfect2_ppc64x.s","import:os/exec"],`), 1)
				}))
				return directory
			},
			want: "import:os/exec",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Setenv(ExternalPacksEnvironment, test.setup(t))
			_, err := loadPacksV2()
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("loadPacksV2() error = %v, want it to mention %q", err, test.want)
			}
		})
	}
}

// TestLoadPackDirectoryReadsOnlyThatDirectory loads an authoring root's packs
// without the embedded ones or GOMAD3_COMPATIBILITY_PACKS, and an absent
// directory as none.
func TestLoadPackDirectoryReadsOnlyThatDirectory(t *testing.T) {
	directory := t.TempDir()
	writeExternalPack(t, directory, "downstream-fixture.json", externalPackFixture(t, "downstream-fixture", nil))
	t.Setenv(ExternalPacksEnvironment, t.TempDir())
	packs, err := LoadPackDirectory(directory)
	if err != nil || len(packs) != 1 || packs[0].pack.ID != "downstream-fixture" {
		t.Fatalf("LoadPackDirectory() = %d packs, %v", len(packs), err)
	}
	packs, err = LoadPackDirectory(filepath.Join(directory, "missing"))
	if err != nil || len(packs) != 0 {
		t.Fatalf("LoadPackDirectory(missing) = %d packs, %v", len(packs), err)
	}
	writeExternalPack(t, directory, "renamed.json", externalPackFixture(t, "downstream-other", nil))
	if _, err := LoadPackDirectory(directory); err == nil {
		t.Fatal("LoadPackDirectory() accepted a pack not named by its ID")
	}
}
