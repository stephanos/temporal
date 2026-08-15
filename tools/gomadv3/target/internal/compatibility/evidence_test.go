package compatibility

import (
	"reflect"
	"testing"
)

func TestSelectionProjectsExactActivationAndAllowanceEvidence(t *testing.T) {
	packages := []Package{
		{
			ImportPath:      "modernc.org/libc",
			Module:          Module{Path: "modernc.org/libc", Version: "v1.72.3", Replaced: true, LocalReplacement: true},
			SourceSetSHA256: "sha256:86528a49d1159917b064c458409f43c9094cca0bb1212d77e157cc05b7457749",
		},
		{
			ImportPath: "golang.org/x/sys/unix",
			Module:     Module{Path: "golang.org/x/sys", Version: "v0.41.0", Sum: "h1:Ivj+2Cp/ylzLiEU89QhWblYnOE9zerudt9Ftecq2C6k="},
		},
	}
	selection, err := Select(packages)
	requireTestNoError(t, err)

	evidence := selection.Evidence()
	if len(evidence) != 1 {
		t.Fatalf("evidence = %#v", evidence)
	}
	requireTestEqual(t, "modernc-libc-xsys-v041", evidence[0].ID)
	if evidence[0].SHA256 == "" {
		t.Fatal("pack evidence has no SHA-256")
	}
	requireTestEqual(t, []ModuleEvidence{
		{Path: "golang.org/x/sys", Version: "v0.41.0", Sum: "h1:Ivj+2Cp/ylzLiEU89QhWblYnOE9zerudt9Ftecq2C6k=", Replacement: "none"},
		{Path: "modernc.org/libc", Version: "v1.72.3", Replacement: "local"},
	}, evidence[0].Activation)
	requireTestEqual(t, []string{"golang.org/x/sys/unix", "modernc.org/libc"}, []string{evidence[0].Rules[0].ImportPath, evidence[0].Rules[1].ImportPath})
	if !contains(evidence[0].Rules[0].Capabilities, "import:syscall") || !contains(evidence[0].Rules[1].Capabilities, "import:os/exec") {
		t.Fatalf("rule evidence = %#v", evidence[0].Rules)
	}

	allowed := selection.Evaluate(packages[1], Fact{Kind: FactCapability, Capability: "import:syscall"})
	if !allowed.Allowed {
		t.Fatalf("decision = %#v", allowed)
	}
	requireTestEqual(t, DispositionAllowedExactPack, allowed.Disposition)
	requireTestEqual(t, "modernc-libc-xsys-v041", allowed.PackID)

	nearMiss := packages[1]
	nearMiss.Module.Version = "v0.42.0"
	denied := selection.Evaluate(nearMiss, Fact{Kind: FactCapability, Capability: "import:syscall"})
	if denied.Allowed {
		t.Fatalf("decision = %#v", denied)
	}
	requireTestEqual(t, DispositionDenied, denied.Disposition)
	requireTestEqual(t, RemediationAddExactPack, denied.Remediation)
}

func TestSelectionEvaluatesClosedRemediationCategories(t *testing.T) {
	selection := Selection{}
	local := Package{ImportPath: "modernc.org/libc", Module: Module{Path: "modernc.org/libc", Version: "v1", LocalReplacement: true}}
	versioned := Package{ImportPath: "example.com/dependency", Module: Module{Path: "example.com/dependency", Version: "v1.0.0", Sum: "h1:sum"}}

	tests := map[string]struct {
		pkg  Package
		fact Fact
		want RemediationCategory
	}{
		"adapter":     {pkg: local, fact: Fact{Kind: FactCapability, Capability: "foreign:c:file.c"}, want: RemediationAddAdapter},
		"exact pack":  {pkg: versioned, fact: Fact{Kind: FactLinkname}, want: RemediationAddExactPack},
		"model":       {pkg: Package{ImportPath: "example.com/main"}, fact: Fact{Kind: FactCapability, Capability: "import:syscall"}, want: RemediationModelOperation},
		"remove":      {pkg: versioned, fact: Fact{Kind: FactNoReviewedGoSource}, want: RemediationRemoveDependency},
		"unsupported": {pkg: versioned, fact: Fact{Kind: FactMalformedLinkname}, want: RemediationRemainUnsupported},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			requireTestEqual(t, test.want, selection.Evaluate(test.pkg, test.fact).Remediation)
		})
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func requireTestNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func requireTestEqual(t *testing.T, want, got any) {
	t.Helper()
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}
