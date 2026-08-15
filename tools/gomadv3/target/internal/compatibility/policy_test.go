package compatibility

import (
	"strings"
	"testing"
)

const validPackV2 = `{
  "schema": "gomadv3.compatibility-pack/v2",
  "id": "example-pack",
  "request_sha256": "sha256:1111111111111111111111111111111111111111111111111111111111111111",
  "governance": {
    "owner": "runtime-team",
    "reviewed_at": "2026-08-15T00:00:00Z",
    "justification": "Allows one exact reviewed import for the synthetic workload.",
    "workloads": ["core-fixture"],
    "platforms": ["darwin/arm64"],
    "approval_sha256": "sha256:2222222222222222222222222222222222222222222222222222222222222222"
  },
  "activation": [{
    "path": "example.com/dependency",
    "version": "v1.2.3",
    "sum": "h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
    "replacement": {"kind": "none"}
  }],
  "rules": [{
    "import_path": "example.com/dependency/internal/runtime",
    "module": {
      "path": "example.com/dependency",
      "version": "v1.2.3",
      "sum": "h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
      "replacement": {"kind": "none"}
    },
    "source_set_sha256": "sha256:3b010f66c93c97f26f632675bca3d51939d7c5e559ee214a292eac38c744e496",
    "go_sources": [{"name": "runtime.go", "sha256": "sha256:4444444444444444444444444444444444444444444444444444444444444444"}],
    "foreign_sources": [],
    "capabilities": ["import:syscall"],
    "linknames": []
  }]
}`

func TestDecodePackV2RejectsWeakOrNonCanonicalPolicy(t *testing.T) {
	decoded, err := DecodePack([]byte(validPackV2))
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Schema != PackSchema || decoded.ID != "example-pack" || decoded.Activation[0].Replacement.Kind != ReplacementNone {
		t.Fatalf("pack = %#v", decoded)
	}

	tests := map[string]string{
		"legacy schema":        strings.Replace(validPackV2, "gomadv3.compatibility-pack/v2", "gomadv3.compatibility-pack/v1", 1),
		"local replacement":    strings.Replace(validPackV2, `"kind": "none"`, `"kind": "local"`, 1),
		"missing sum":          strings.Replace(validPackV2, `"sum": "h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",`, `"sum": "",`, 1),
		"duplicate capability": strings.Replace(validPackV2, `"capabilities": ["import:syscall"]`, `"capabilities": ["import:syscall", "import:syscall"]`, 1),
		"source-set mismatch":  strings.Replace(validPackV2, "sha256:3b010f66c93c97f26f632675bca3d51939d7c5e559ee214a292eac38c744e496", "sha256:3333333333333333333333333333333333333333333333333333333333333333", 1),
		"unknown field":        strings.Replace(validPackV2, `"id": "example-pack",`, `"id": "example-pack", "unknown": true,`, 1),
		"trailing data":        validPackV2 + `{}`,
	}
	for name, encoded := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodePack([]byte(encoded)); err == nil {
				t.Fatal("DecodePack() succeeded")
			}
		})
	}
}

func TestSelectValidatedPackRequiresCompleteSourceInventory(t *testing.T) {
	validated, err := LoadPack([]byte(validPackV2))
	if err != nil {
		t.Fatal(err)
	}
	pkg := Package{
		ImportPath:      "example.com/dependency/internal/runtime",
		Module:          Module{Path: "example.com/dependency", Version: "v1.2.3", Sum: "h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="},
		SourceSetSHA256: "sha256:3b010f66c93c97f26f632675bca3d51939d7c5e559ee214a292eac38c744e496",
		GoSources:       []Source{{Name: "runtime.go", SHA256: "sha256:4444444444444444444444444444444444444444444444444444444444444444"}},
		ForeignSources:  []ForeignSource{},
	}
	selection, err := SelectPacks([]ValidatedPack{validated}, []Package{pkg})
	if err != nil {
		t.Fatal(err)
	}
	if !selection.AllowsCapability(pkg, "import:syscall") {
		t.Fatal("exact complete source inventory was not authorized")
	}
	projected := selection.Evidence()
	if len(projected) != 1 || projected[0].RequestSHA256 != "sha256:1111111111111111111111111111111111111111111111111111111111111111" || len(projected[0].Rules) != 1 || len(projected[0].Rules[0].GoSources) != 1 {
		t.Fatalf("pack evidence = %#v", projected)
	}

	pkg.GoSources[0].SHA256 = "sha256:5555555555555555555555555555555555555555555555555555555555555555"
	selection, err = SelectPacks([]ValidatedPack{validated}, []Package{pkg})
	if err != nil {
		t.Fatal(err)
	}
	if selection.AllowsCapability(pkg, "import:syscall") {
		t.Fatal("changed source inventory was authorized")
	}
}

func TestSelectValidatedPackRequiresApprovedPlatform(t *testing.T) {
	validated, err := LoadPack([]byte(validPackV2))
	if err != nil {
		t.Fatal(err)
	}
	selection, err := SelectPacksForPlatform([]ValidatedPack{validated}, nil, "linux/amd64")
	if err != nil {
		t.Fatal(err)
	}
	if len(selection.Identities()) != 0 {
		t.Fatalf("wrong-platform identities = %#v", selection.Identities())
	}
}

func TestSelectValidatedPackRequiresApplicableRule(t *testing.T) {
	validated, err := LoadPack([]byte(validPackV2))
	if err != nil {
		t.Fatal(err)
	}
	selection, err := SelectPacksForPlatform([]ValidatedPack{validated}, []Package{{
		ImportPath: "example.com/dependency/other",
		Module:     Module{Path: "example.com/dependency", Version: "v1.2.3", Sum: "h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="},
	}}, "darwin/arm64")
	if err != nil {
		t.Fatal(err)
	}
	if len(selection.Identities()) != 0 {
		t.Fatalf("inapplicable pack identities = %#v", selection.Identities())
	}
}

func TestVerifyValidatedPackIdentitiesRejectsUnavailableOrModifiedPacks(t *testing.T) {
	validated, err := LoadPack([]byte(validPackV2))
	if err != nil {
		t.Fatal(err)
	}
	identities := []Identity{{ID: validated.pack.ID, SHA256: validated.digest}}
	if err := VerifyPackIdentities([]ValidatedPack{validated}, identities); err != nil {
		t.Fatal(err)
	}
	identities[0].SHA256 = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	if err := VerifyPackIdentities([]ValidatedPack{validated}, identities); err == nil {
		t.Fatal("VerifyPackIdentities() accepted a changed pack digest")
	}
}

func TestSelectBindsExactPackIdentityAndCapabilities(t *testing.T) {
	validated := loadGeneratedPackForTest(t, "modernc-libc-xsys-v041")
	packages := generatedExactPackages(validated.pack)
	selection, err := Select(packages)
	if err != nil {
		t.Fatal(err)
	}
	identities := selection.Identities()
	if len(identities) != 1 || identities[0].ID != "modernc-libc-xsys-v041" || identities[0].SHA256 == "" {
		t.Fatalf("identities = %#v", identities)
	}
	xsys := generatedPackageForTest(t, packages, "golang.org/x/sys/unix")
	if !selection.AllowsCapability(xsys, "import:syscall") {
		t.Fatal("exact x/sys import capability was not authorized")
	}
	xsys.Module.Version = "v0.42.0"
	if selection.AllowsCapability(xsys, "import:syscall") {
		t.Fatal("unknown x/sys identity was authorized")
	}
	libc := generatedPackageForTest(t, packages, "modernc.org/libc")
	if !selection.AllowsCapability(libc, "import:syscall") {
		t.Fatal("exact local adapter source set was not authorized")
	}
	libc.SourceSetSHA256 = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	if selection.AllowsCapability(libc, "import:syscall") {
		t.Fatal("modified local adapter source set was authorized")
	}
}

func TestSelectRequiresExactLinknameSourceIdentity(t *testing.T) {
	validated := loadGeneratedPackForTest(t, "reflect2-go126")
	packages := generatedExactPackages(validated.pack)
	selection, err := Select(packages)
	if err != nil {
		t.Fatal(err)
	}
	pkg := generatedPackageForTest(t, packages, "github.com/modern-go/reflect2")
	directives := []string{"mapiterinit reflect.mapiterinit"}
	if !selection.AllowsLinkname(pkg, "go_above_118.go", "sha256:b41d841d561da73b0ab54f9f2830d7f9437561b831faad1fa22f738ea99ad805", directives) {
		t.Fatal("exact reflect2 source was not authorized")
	}
	if selection.AllowsLinkname(pkg, "go_above_118.go", "sha256:0000000000000000000000000000000000000000000000000000000000000000", directives) {
		t.Fatal("modified reflect2 source was authorized")
	}
}

func TestVerifyIdentitiesRejectsUnknownOrModifiedPacks(t *testing.T) {
	validated := loadGeneratedPackForTest(t, "reflect2-go126")
	selection, err := Select(generatedExactPackages(validated.pack))
	if err != nil {
		t.Fatal(err)
	}
	identities := selection.Identities()
	if err := VerifyIdentities(identities); err != nil {
		t.Fatal(err)
	}
	identities[0].SHA256 = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	if err := VerifyIdentities(identities); err == nil {
		t.Fatal("VerifyIdentities() accepted a modified pack")
	}
}

func loadGeneratedPackForTest(t *testing.T, id string) ValidatedPack {
	t.Helper()
	contents, err := packFiles.ReadFile("packs/" + id + ".json")
	if err != nil {
		t.Fatal(err)
	}
	validated, err := LoadPack(contents)
	if err != nil {
		t.Fatal(err)
	}
	return validated
}

func generatedPackageForTest(t *testing.T, packages []Package, importPath string) Package {
	t.Helper()
	for _, pkg := range packages {
		if pkg.ImportPath == importPath {
			return pkg
		}
	}
	t.Fatalf("generated packages do not contain %s", importPath)
	return Package{}
}
