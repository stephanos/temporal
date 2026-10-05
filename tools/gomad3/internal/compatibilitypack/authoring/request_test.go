package authoring

import (
	"fmt"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	compatibility "go.temporal.io/server/tools/gomad3/internal/compatibilitypack"
	"go.temporal.io/server/tools/gomad3/target"
)

func TestApprovalSHA256BindsCompleteReviewedRequest(t *testing.T) {
	request := validRequest()
	digest, err := ApprovalSHA256(request)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(digest, "sha256:") || len(digest) != len("sha256:")+64 {
		t.Fatalf("approval digest = %q", digest)
	}
	request.ApprovalSHA256 = "sha256:" + strings.Repeat("f", 64)
	withApproval, err := ApprovalSHA256(request)
	if err != nil {
		t.Fatal(err)
	}
	if withApproval != digest {
		t.Fatalf("approval field changed digest: got %q, want %q", withApproval, digest)
	}

	mutations := map[string]func(*Request){
		"owner":         func(request *Request) { request.Owner = "another-team" },
		"review time":   func(request *Request) { request.ReviewedAt = "2026-08-16T00:00:00Z" },
		"justification": func(request *Request) { request.Justification += " Additional reason." },
		"workload":      func(request *Request) { request.Workloads[0] = "another-workload" },
		"platform":      func(request *Request) { request.Platforms[0] = "linux/amd64" },
		"source": func(request *Request) {
			request.Packages[0].Evidence.GoSources[0].SHA256 = "sha256:" + strings.Repeat("e", 64)
			request.Packages[0].Evidence.SourceSetSHA256 = "sha256:226df64c94c787464495fbef20913adac635ce6817e582562d1b9aa0c7e333f2"
		},
		"fact disposition": func(request *Request) { request.Packages[0].Facts[0].Disposition = DispositionDeny },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			changed := validRequest()
			mutate(&changed)
			changedDigest, err := ApprovalSHA256(changed)
			if err != nil {
				t.Fatal(err)
			}
			if changedDigest == digest {
				t.Fatalf("mutation did not change approval digest: %q", changedDigest)
			}
		})
	}
}

func TestDecodeDraftRequestAcceptsSelectorsWithoutDiscoveredEvidence(t *testing.T) {
	draft := validRequest()
	draft.Activation[0].Evidence = compatibility.PackModule{}
	draft.Packages[0].Evidence = compatibility.PackRule{}
	encoded, err := canonicaljson.CanonicalJSON(draft)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeDraftRequest(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.ID != draft.ID || decoded.Packages[0].ImportPath != draft.Packages[0].ImportPath {
		t.Fatalf("decoded draft = %#v", decoded)
	}
}

func TestValidateRequestRejectsOperationalPathsAndCollectionOverflow(t *testing.T) {
	request := validRequest()
	request.Target.TestArguments = []string{"/private/target"}
	if err := ValidateRequest(request); err == nil {
		t.Fatal("ValidateRequest() accepted an absolute operational argument")
	}

	request = validRequest()
	request.Target.BuildTags = make([]string, 65)
	for index := range request.Target.BuildTags {
		request.Target.BuildTags[index] = fmt.Sprintf("tag%02d", index)
	}
	if err := ValidateRequest(request); err == nil {
		t.Fatal("ValidateRequest() accepted too many build tags")
	}
}

func TestValidateRequestRejectsUnadmittableCapabilities(t *testing.T) {
	for _, capability := range []string{"import:os/exec", "import:os/signal", "import:os/user"} {
		t.Run(capability, func(t *testing.T) {
			request := validRequest()
			request.Packages[0].Facts = []Fact{{Kind: FactCapability, Capability: capability, Disposition: DispositionAllow}}
			if err := ValidateRequest(request); err == nil || !strings.Contains(err.Error(), capability+" is never admitted") {
				t.Fatalf("ValidateRequest() = %v", err)
			}
			request.Packages[0].Facts = []Fact{{Kind: FactCapability, Capability: capability, Disposition: DispositionDeny}}
			if err := ValidateRequest(request); err != nil {
				t.Fatalf("ValidateRequest() rejected a denied %s: %v", capability, err)
			}
		})
	}
}

func TestRequestAdmissionRejectsFiveAllowedHostImports(t *testing.T) {
	for _, test := range []struct{ capability, want string }{
		{"import:os/exec", "compatibility-pack capability import:os/exec is never admitted"},
		{"import:os/signal", "compatibility-pack capability import:os/signal is never admitted"},
		{"import:os/user", "compatibility-pack capability import:os/user is never admitted"},
		{"import:plugin", "compatibility-pack capability import:plugin is never admitted"},
		{"import:runtime/cgo", "compatibility-pack capability import:runtime/cgo is never admitted"},
	} {
		t.Run(test.capability, func(t *testing.T) {
			request := validRequest()
			request.Packages[0].Facts = []Fact{{Kind: FactCapability, Capability: test.capability, Disposition: DispositionAllow}}
			requireRequestAdmissionError(t, ValidateRequest(request), test.want)
			encoded, err := canonicaljson.CanonicalJSON(request)
			if err != nil {
				t.Fatal(err)
			}
			_, err = DecodeRequest(encoded)
			requireRequestAdmissionError(t, err, test.want)
			_, err = ApprovalSHA256(request)
			requireRequestAdmissionError(t, err, test.want)
			_, _, err = RenderReview(request)
			requireRequestAdmissionError(t, err, test.want)
		})
	}
}

func TestRequestAdmissionPreservesFactValidationPriority(t *testing.T) {
	for _, test := range []struct {
		name  string
		facts []Fact
		want  string
	}{
		{"admission-before-inventory", []Fact{{Kind: FactCapability, Capability: "import:plugin", Disposition: DispositionAllow}}, "compatibility-pack capability import:plugin is never admitted"},
		{"malformed-before-admission", []Fact{{Kind: FactCapability, Capability: "import:plugin", Source: "runtime.go", Disposition: DispositionAllow}}, "compatibility-pack capability fact is invalid"},
		{"disposition-before-admission", []Fact{{Kind: FactCapability, Capability: "import:plugin", Disposition: "future"}}, "compatibility-pack request fact disposition is invalid"},
		{"earlier-fact-before-admission", []Fact{
			{Kind: FactCapability, Capability: "import:a", Source: "runtime.go", Disposition: DispositionDeny},
			{Kind: FactCapability, Capability: "import:plugin", Disposition: DispositionAllow},
		}, "compatibility-pack capability fact is invalid"},
		{"admission-before-later-fact", []Fact{
			{Kind: FactCapability, Capability: "import:plugin", Disposition: DispositionAllow},
			{Kind: FactCapability, Capability: "import:syscall", Source: "runtime.go", Disposition: DispositionDeny},
		}, "compatibility-pack capability import:plugin is never admitted"},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := validRequest()
			request.Packages[0].Facts = test.facts
			request.Packages[0].Evidence.GoSources = nil
			requireRequestAdmissionError(t, ValidateRequest(request), test.want)
		})
	}
}

func requireRequestAdmissionError(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil || err.Error() != want {
		t.Fatalf("error = %v, want %q", err, want)
	}
}

func validRequest() Request {
	module := compatibility.PackModule{
		Path: "example.com/dependency", Version: "v1.2.3", Sum: "h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		Replacement: compatibility.PackReplacement{Kind: compatibility.ReplacementNone},
	}
	return Request{
		Schema: RequestSchema, ID: "example-pack",
		Target:     Target{Kind: target.KindGoTest, Package: "./fixture", TestArguments: []string{"-test.run", "^TestFixture$"}, BuildTags: []string{"test_dep"}, ExpectedModule: "example.com/main"},
		Activation: []Activation{{Path: "example.com/dependency", Evidence: module}},
		Packages: []Package{{
			ImportPath: "example.com/dependency/internal/runtime",
			Facts:      []Fact{{Kind: FactCapability, Capability: "import:syscall", Disposition: DispositionAllow}},
			Evidence: compatibility.PackRule{
				ImportPath: "example.com/dependency/internal/runtime", Module: module,
				SourceSetSHA256: "sha256:8ae49dab0499a1c49b23aac2cde0cd0c4edeb8e291faf0e53c0461ebd8416859",
				GoSources:       []compatibility.PackSource{{Name: "runtime.go", SHA256: "sha256:" + strings.Repeat("4", 64)}},
				ForeignSources:  []compatibility.PackForeignSource{}, Capabilities: []string{}, Linknames: []compatibility.PackLinkname{},
			},
		}},
		Owner: "runtime-team", ReviewedAt: "2026-08-15T00:00:00Z",
		Justification: "Allows one reviewed dependency capability.",
		Workloads:     []string{"core-fixture"}, Platforms: []string{"darwin/arm64"}, ApprovalSHA256: "",
	}
}
