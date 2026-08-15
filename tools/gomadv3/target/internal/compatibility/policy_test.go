package compatibility

import (
	"testing"
)

func TestSelectBindsExactPackIdentityAndCapabilities(t *testing.T) {
	selection, err := Select([]Package{
		{ImportPath: "modernc.org/libc", Module: Module{Path: "modernc.org/libc", Version: "v1.72.3", Replaced: true, LocalReplacement: true}},
		{ImportPath: "golang.org/x/sys/unix", Module: Module{Path: "golang.org/x/sys", Version: "v0.41.0", Sum: "h1:Ivj+2Cp/ylzLiEU89QhWblYnOE9zerudt9Ftecq2C6k="}},
	})
	if err != nil {
		t.Fatal(err)
	}
	identities := selection.Identities()
	if len(identities) != 1 || identities[0].ID != "modernc-libc-xsys-v041" || identities[0].SHA256 == "" {
		t.Fatalf("identities = %#v", identities)
	}
	if !selection.AllowsCapability(Package{ImportPath: "golang.org/x/sys/unix", Module: Module{Path: "golang.org/x/sys", Version: "v0.41.0", Sum: "h1:Ivj+2Cp/ylzLiEU89QhWblYnOE9zerudt9Ftecq2C6k="}}, "import:syscall") {
		t.Fatal("exact x/sys import capability was not authorized")
	}
	if selection.AllowsCapability(Package{ImportPath: "golang.org/x/sys/unix", Module: Module{Path: "golang.org/x/sys", Version: "v0.42.0", Sum: "h1:changed"}}, "import:syscall") {
		t.Fatal("unknown x/sys identity was authorized")
	}
	libc := Package{
		ImportPath:      "modernc.org/libc",
		Module:          Module{Path: "modernc.org/libc", Version: "v1.72.3", Replaced: true, LocalReplacement: true},
		SourceSetSHA256: "sha256:86528a49d1159917b064c458409f43c9094cca0bb1212d77e157cc05b7457749",
	}
	if !selection.AllowsCapability(libc, "import:syscall") {
		t.Fatal("exact local adapter source set was not authorized")
	}
	libc.SourceSetSHA256 = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	if selection.AllowsCapability(libc, "import:syscall") {
		t.Fatal("modified local adapter source set was authorized")
	}
}

func TestSelectRequiresExactLinknameSourceIdentity(t *testing.T) {
	selection, err := Select([]Package{{
		ImportPath: "github.com/modern-go/reflect2",
		Module:     Module{Path: "github.com/modern-go/reflect2", Version: "v1.0.3-0.20250322232337-35a7c28c31ee", Sum: "h1:W5t00kpgFdJifH4BDsTlE89Zl93FEloxaWZfGcifgq8="},
	}})
	if err != nil {
		t.Fatal(err)
	}
	pkg := Package{ImportPath: "github.com/modern-go/reflect2", Module: Module{Path: "github.com/modern-go/reflect2", Version: "v1.0.3-0.20250322232337-35a7c28c31ee", Sum: "h1:W5t00kpgFdJifH4BDsTlE89Zl93FEloxaWZfGcifgq8="}}
	directives := []string{"mapiterinit reflect.mapiterinit"}
	if !selection.AllowsLinkname(pkg, "go_above_118.go", "sha256:b41d841d561da73b0ab54f9f2830d7f9437561b831faad1fa22f738ea99ad805", directives) {
		t.Fatal("exact reflect2 source was not authorized")
	}
	if selection.AllowsLinkname(pkg, "go_above_118.go", "sha256:0000000000000000000000000000000000000000000000000000000000000000", directives) {
		t.Fatal("modified reflect2 source was authorized")
	}
}

func TestVerifyIdentitiesRejectsUnknownOrModifiedPacks(t *testing.T) {
	selection, err := Select([]Package{{
		ImportPath: "github.com/modern-go/reflect2",
		Module:     Module{Path: "github.com/modern-go/reflect2", Version: "v1.0.3-0.20250322232337-35a7c28c31ee", Sum: "h1:W5t00kpgFdJifH4BDsTlE89Zl93FEloxaWZfGcifgq8="},
	}})
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
