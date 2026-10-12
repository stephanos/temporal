package compatibility

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
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

func TestExternalPackAdmissionRejectsFiveHostImportsWithoutPartialResults(t *testing.T) {
	for _, test := range []struct{ capability, want string }{
		{"import:os/exec", "external compatibility pack z-invalid.json: compatibility pack rule 0: capability import:os/exec is never admitted"},
		{"import:os/signal", "external compatibility pack z-invalid.json: compatibility pack rule 0: capability import:os/signal is never admitted"},
		{"import:os/user", "external compatibility pack z-invalid.json: compatibility pack rule 0: capability import:os/user is never admitted"},
		{"import:plugin", "external compatibility pack z-invalid.json: compatibility pack rule 0: capability import:plugin is never admitted"},
		{"import:runtime/cgo", "external compatibility pack z-invalid.json: compatibility pack rule 0: capability import:runtime/cgo is never admitted"},
	} {
		t.Run(test.capability, func(t *testing.T) {
			directory := t.TempDir()
			writeExternalPack(t, directory, "a-valid.json", externalPackFixture(t, "a-valid", nil))
			valid, err := LoadPackDirectory(directory)
			requireTestNoError(t, err)
			requireTestEqual(t, 1, len(valid))
			invalid, err := DecodePack(externalPackFixture(t, "z-invalid", nil))
			requireTestNoError(t, err)
			invalid.Rules[0].Capabilities = []string{test.capability}
			encoded, err := canonicaljson.CanonicalJSON(invalid)
			requireTestNoError(t, err)
			writeExternalPack(t, directory, "z-invalid.json", encoded)
			packs, err := LoadPackDirectory(directory)
			requireAdmissionError(t, err, test.want)
			if packs != nil {
				t.Fatalf("LoadPackDirectory returned partial packs: %#v", packs)
			}
			t.Setenv(ExternalPacksEnvironment, directory)
			packs, err = LoadPacks()
			requireAdmissionError(t, err, test.want)
			if packs != nil {
				t.Fatalf("LoadPacks returned partial packs: %#v", packs)
			}
		})
	}
}
