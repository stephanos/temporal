package authoring

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	compatibility "go.temporal.io/server/tools/gomad3/internal/compatibilitypack"
)

func TestGenerateStandaloneConsumerPackage(t *testing.T) {
	root := t.TempDir()
	request := validRequest()
	approval, err := ApprovalSHA256(request)
	if err != nil {
		t.Fatal(err)
	}
	if err := Generate(root, request, approval); err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filepath.Join(root, "packs_generated_test.go"), nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	configuration := types.Config{}
	if _, err := configuration.Check("example.com/consumer/compatibility", fset, []*ast.File{file}, nil); err != nil {
		t.Fatalf("generated standalone consumer package does not compile: %v", err)
	}
}

func TestGenerateRequiresExactApprovalAndCheckDetectsDrift(t *testing.T) {
	root := t.TempDir()
	request := validRequest()
	approval, err := ApprovalSHA256(request)
	if err != nil {
		t.Fatal(err)
	}
	if err := Generate(root, request, "sha256:0000000000000000000000000000000000000000000000000000000000000000"); err == nil {
		t.Fatal("Generate() accepted the wrong approval")
	}
	if _, err := os.Stat(filepath.Join(root, "packs", "example-pack.json")); !os.IsNotExist(err) {
		t.Fatalf("rejected generation published a pack: %v", err)
	}

	if err := Generate(root, request, approval); err != nil {
		t.Fatal(err)
	}
	if err := Check(root); err != nil {
		t.Fatal(err)
	}
	packPath := filepath.Join(root, "packs", "example-pack.json")
	pack, err := os.ReadFile(packPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(packPath, append(pack, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Check(root); err == nil {
		t.Fatal("Check() accepted a modified generated pack")
	}
}

func TestRegenerateUsesOnlyRecordedExactApprovals(t *testing.T) {
	root := t.TempDir()
	request := validRequest()
	approval, err := ApprovalSHA256(request)
	if err != nil {
		t.Fatal(err)
	}
	if err := Generate(root, request, approval); err != nil {
		t.Fatal(err)
	}
	packPath := filepath.Join(root, "packs", request.ID+".json")
	if err := os.WriteFile(packPath, []byte("stale\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Regenerate(root); err != nil {
		t.Fatal(err)
	}
	if err := Check(root); err != nil {
		t.Fatal(err)
	}
}

func TestGenerateAdmissionRejectsFiveHostImportsBeforePublication(t *testing.T) {
	for _, test := range []struct{ capability, approval, want string }{
		{"import:os/exec", "", "compatibility-pack capability import:os/exec is never admitted"},
		{"import:os/signal", "", "compatibility-pack capability import:os/signal is never admitted"},
		{"import:os/user", "", "compatibility-pack capability import:os/user is never admitted"},
		{"import:plugin", "sha256:4498c8a9f3a04185d698117b02156da5efe90800e896962b7469f1165ed84df3", "compatibility-pack capability import:plugin is never admitted"},
		{"import:runtime/cgo", "sha256:87db10412554e878db7f86b0208c28f8526e80ef11fe0783b6c2c743065a2988", "compatibility-pack capability import:runtime/cgo is never admitted"},
	} {
		t.Run(test.capability, func(t *testing.T) {
			root := t.TempDir()
			request := validRequest()
			approval, err := ApprovalSHA256(request)
			if err != nil {
				t.Fatal(err)
			}
			if err := Generate(root, request, approval); err != nil {
				t.Fatal(err)
			}
			before := generationSnapshot(t, root)
			request.Packages[0].Facts[0].Capability = test.capability
			if test.approval != "" {
				approval = test.approval
			}
			request.ApprovalSHA256 = approval
			err = Generate(root, request, approval)
			if after := generationSnapshot(t, root); !maps.Equal(before, after) {
				t.Fatal("rejected generation changed the populated root")
			}
			requireRequestAdmissionError(t, err, test.want)
			if err := Check(root); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestGenerateAdmissionRetainsFiveDeniedHostImportsWithExactSyscallGrant(t *testing.T) {
	for _, capability := range []string{"import:os/exec", "import:os/signal", "import:os/user", "import:plugin", "import:runtime/cgo"} {
		t.Run(capability, func(t *testing.T) {
			request := validRequest()
			request.Packages[0].Facts = []Fact{
				{Kind: FactCapability, Capability: capability, Disposition: DispositionDeny},
				{Kind: FactCapability, Capability: "import:syscall", Disposition: DispositionAllow},
			}
			if err := ValidateRequest(request); err != nil {
				t.Fatal(err)
			}
			encoded, err := canonicaljson.CanonicalJSON(request)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := DecodeRequest(encoded)
			if err != nil || decoded.Packages[0].Facts[0].Disposition != DispositionDeny || decoded.Packages[0].Facts[0].Capability != capability {
				t.Fatalf("denied request round trip = %#v, %v", decoded, err)
			}
			report, approval, err := RenderReview(request)
			if err != nil {
				t.Fatal(err)
			}
			denial := "- `" + capability + "`: **deny**"
			if !strings.Contains(string(report), denial) {
				t.Fatalf("review omitted denial %q", denial)
			}
			root := t.TempDir()
			if err := Generate(root, request, approval); err != nil {
				t.Fatal(err)
			}
			if err := Check(root); err != nil {
				t.Fatal(err)
			}
			stored, err := os.ReadFile(filepath.Join(root, "requests", request.ID+".json"))
			if err != nil {
				t.Fatal(err)
			}
			retained, err := DecodeRequest(stored)
			if err != nil || retained.Packages[0].Facts[0].Disposition != DispositionDeny || retained.Packages[0].Facts[0].Capability != capability {
				t.Fatalf("generated request lost denial = %#v, %v", retained, err)
			}
			storedReport, err := os.ReadFile(filepath.Join(root, "reports", request.ID+".md"))
			if err != nil || string(storedReport) != string(report) {
				t.Fatalf("generated report differs from reviewed denial: %v", err)
			}
			packs, err := compatibility.LoadPackDirectory(filepath.Join(root, "packs"))
			if err != nil || len(packs) != 1 {
				t.Fatalf("generated packs = %d, %v", len(packs), err)
			}
			pack := packs[0].Pack()
			if len(pack.Rules) != 1 || !slices.Equal(pack.Rules[0].Capabilities, []string{"import:syscall"}) || len(pack.Rules[0].Linknames) != 0 {
				t.Fatalf("generated grants = %#v", pack.Rules)
			}
			pkg := compatibility.Package{
				ImportPath:      "example.com/dependency/internal/runtime",
				Module:          compatibility.Module{Path: "example.com/dependency", Version: "v1.2.3", Sum: "h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="},
				SourceSetSHA256: "sha256:8ae49dab0499a1c49b23aac2cde0cd0c4edeb8e291faf0e53c0461ebd8416859",
				GoSources:       []compatibility.Source{{Name: "runtime.go", SHA256: "sha256:4444444444444444444444444444444444444444444444444444444444444444"}},
				ForeignSources:  []compatibility.ForeignSource{},
			}
			selection, err := compatibility.SelectPacksForPlatform(packs, []compatibility.Package{pkg}, "darwin/arm64")
			if err != nil {
				t.Fatal(err)
			}
			allowed := selection.Evaluate(pkg, compatibility.Fact{Kind: compatibility.FactCapability, Capability: "import:syscall"})
			if allowed != (compatibility.Decision{Allowed: true, Disposition: "allowed_by_exact_pack", PackID: "example-pack"}) {
				t.Fatalf("exact syscall decision = %#v", allowed)
			}
			denied := selection.Evaluate(pkg, compatibility.Fact{Kind: compatibility.FactCapability, Capability: capability})
			if denied != (compatibility.Decision{Disposition: "denied", Remediation: "remain_unsupported"}) {
				t.Fatalf("prohibited import decision = %#v", denied)
			}
		})
	}
}

func generationSnapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	files := make(map[string]string)
	if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		var contents []byte
		if !entry.IsDir() {
			contents, err = os.ReadFile(path)
			if err != nil {
				return err
			}
		}
		files[path] = fmt.Sprintf("%s\x00%s", info.Mode(), contents)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return files
}
