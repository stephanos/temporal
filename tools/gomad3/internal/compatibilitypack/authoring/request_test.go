package authoring

import (
	"fmt"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomad3/internal/compatibilitypack"
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
			request.Packages[0].Evidence.SourceSetSHA256 = "sha256:d7284b6ed361460152190946e4b35247f1d0d6823c8f55bfdd40adc88d59ce4b"
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
				SourceSetSHA256: "sha256:3b010f66c93c97f26f632675bca3d51939d7c5e559ee214a292eac38c744e496",
				GoSources:       []compatibility.PackSource{{Name: "runtime.go", SHA256: "sha256:" + strings.Repeat("4", 64)}},
				ForeignSources:  []compatibility.PackForeignSource{}, Capabilities: []string{}, Linknames: []compatibility.PackLinkname{},
			},
		}},
		Owner: "runtime-team", ReviewedAt: "2026-08-15T00:00:00Z",
		Justification: "Allows one reviewed dependency capability.",
		Workloads:     []string{"core-fixture"}, Platforms: []string{"darwin/arm64"}, ApprovalSHA256: "",
	}
}
