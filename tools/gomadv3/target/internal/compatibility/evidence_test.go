package compatibility

import (
	"reflect"
	"testing"
)

func TestSelectionProjectsExactActivationAndAllowanceEvidence(t *testing.T) {
	validated := loadGeneratedPackForTest(t, "modernc-libc-xsys-v041")
	packages := generatedExactPackages(validated.pack)
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
	if evidence[0].RequestSHA256 == "" || evidence[0].Governance == nil {
		t.Fatalf("pack governance evidence = %#v", evidence[0])
	}
	if len(evidence[0].Activation) != 2 || evidence[0].Activation[0].Replacement != "none" || evidence[0].Activation[1].Replacement != "adapter" || evidence[0].Activation[1].Adapter == nil {
		t.Fatalf("activation evidence = %#v", evidence[0].Activation)
	}
	if len(evidence[0].Rules) != 5 {
		t.Fatalf("rule evidence = %#v", evidence[0].Rules)
	}
	requireTestEqual(t, []string{"github.com/mattn/go-isatty", "github.com/remyoudompheng/bigfft", "golang.org/x/sys/unix", "modernc.org/libc", "modernc.org/memory"}, []string{evidence[0].Rules[0].ImportPath, evidence[0].Rules[1].ImportPath, evidence[0].Rules[2].ImportPath, evidence[0].Rules[3].ImportPath, evidence[0].Rules[4].ImportPath})
	if len(evidence[0].Rules[2].GoSources) == 0 || len(evidence[0].Rules[2].ForeignSources) == 0 || !contains(evidence[0].Rules[2].Capabilities, "import:syscall") || !contains(evidence[0].Rules[3].Capabilities, "import:os/exec") {
		t.Fatalf("rule evidence = %#v", evidence[0].Rules)
	}

	xsys := generatedPackageForTest(t, packages, "golang.org/x/sys/unix")
	allowed := selection.Evaluate(xsys, Fact{Kind: FactCapability, Capability: "import:syscall"})
	if !allowed.Allowed {
		t.Fatalf("decision = %#v", allowed)
	}
	requireTestEqual(t, DispositionAllowedExactPack, allowed.Disposition)
	requireTestEqual(t, "modernc-libc-xsys-v041", allowed.PackID)

	nearMiss := xsys
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
