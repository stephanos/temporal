package authoring

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	compatibility "go.temporal.io/server/tools/gomad/internal/backends/native/compatibilitypack"
	"go.temporal.io/server/tools/gomad/internal/backends/target"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
)

func TestPublishRequestUsesStdlibEncoding(t *testing.T) {
	request := validRequest()
	path := filepath.Join(t.TempDir(), "request.json")
	if err := PublishRequest(path, request); err != nil {
		t.Fatal(err)
	}
	encoded, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	want = append(want, '\n')
	if !bytes.Equal(encoded, want) {
		t.Fatalf("published request = %s, want stdlib bytes %s", encoded, want)
	}
}

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

	type mutation struct {
		name    string
		request func(*testing.T) Request
		mutate  func(*Request)
	}
	fromFixture := func(name string) func(*testing.T) Request {
		return func(t *testing.T) Request { return retainedRequest(t, name) }
	}
	base := func(*testing.T) Request { return validRequest() }
	mutations := []mutation{
		{"request ID", base, func(request *Request) { request.ID = "another-pack" }},
		{"target kind", base, func(request *Request) { request.Target.Kind = target.KindGoRun }},
		{"target package", base, func(request *Request) { request.Target.Package = "./another-fixture" }},
		{"target argument", base, func(request *Request) { request.Target.TestArguments[1] = "^TestAnother$" }},
		{"target build tag", base, func(request *Request) { request.Target.BuildTags[0] = "another_tag" }},
		{"expected module", base, func(request *Request) { request.Target.ExpectedModule = "example.com/another" }},
		{"activation path", base, func(request *Request) {
			request.Activation[0].Path = "example.com/another-dependency"
			request.Activation[0].Evidence.Path = request.Activation[0].Path
		}},
		{"activation version", base, func(request *Request) { request.Activation[0].Evidence.Version = "v1.2.4" }},
		{"activation sum", base, func(request *Request) { request.Activation[0].Evidence.Sum = moduleSum("B") }},
		{"package identity", base, func(request *Request) {
			request.Packages[0].ImportPath = "example.com/dependency/internal/another"
			request.Packages[0].Evidence.ImportPath = request.Packages[0].ImportPath
		}},
		{"package module path", base, func(request *Request) { request.Packages[0].Evidence.Module.Path = "example.com/another-dependency" }},
		{"package module version", base, func(request *Request) { request.Packages[0].Evidence.Module.Version = "v1.2.4" }},
		{"package module sum", base, func(request *Request) { request.Packages[0].Evidence.Module.Sum = moduleSum("B") }},
		{"Go source name", base, func(request *Request) {
			request.Packages[0].Evidence.GoSources[0].Name = "another.go"
			refreshSourceSet(request, 0)
		}},
		{"Go source identity", base, func(request *Request) {
			request.Packages[0].Evidence.GoSources[0].SHA256 = digestOf("e")
			refreshSourceSet(request, 0)
		}},
		{"capability fact", base, func(request *Request) { request.Packages[0].Facts[0].Capability = "import:reflect" }},
		{"fact disposition", base, func(request *Request) { request.Packages[0].Facts[0].Disposition = DispositionDeny }},
		{"owner", base, func(request *Request) { request.Owner = "another-team" }},
		{"review time", base, func(request *Request) { request.ReviewedAt = "2026-08-16T00:00:00Z" }},
		{"justification", base, func(request *Request) { request.Justification += " Additional reason." }},
		{"workload", base, func(request *Request) { request.Workloads[0] = "another-workload" }},
		{"platform", base, func(request *Request) { request.Platforms[0] = "linux/amd64" }},

		{"foreign source kind", fromFixture("modernc-libc-xsys-v047-linux-amd64"), func(request *Request) {
			pkg, source := firstForeignSource(request)
			source.Kind = "assembler"
			refreshSourceSet(request, pkg)
		}},
		{"foreign source name", fromFixture("modernc-libc-xsys-v047-linux-amd64"), func(request *Request) {
			pkg, source := firstForeignSource(request)
			source.Name = "another.s"
			refreshSourceSet(request, pkg)
		}},
		{"foreign source identity", fromFixture("modernc-libc-xsys-v047-linux-amd64"), func(request *Request) {
			pkg, source := firstForeignSource(request)
			source.SHA256 = digestOf("e")
			refreshSourceSet(request, pkg)
		}},

		{"linkname source", fromFixture("modernc-libc-xsys-v047-linux-amd64"), func(request *Request) { firstLinknameFact(request).Source = "another.go" }},
		{"linkname source identity", fromFixture("modernc-libc-xsys-v047-linux-amd64"), func(request *Request) { firstLinknameFact(request).SHA256 = digestOf("e") }},
		{"linkname directives", fromFixture("modernc-libc-xsys-v047-linux-amd64"), func(request *Request) { firstLinknameFact(request).Directives[0] = "local another.symbol" }},
		{"linkname disposition", fromFixture("modernc-libc-xsys-v047-linux-amd64"), func(request *Request) { firstLinknameFact(request).Disposition = DispositionDeny }},

		{"adapter profile", fromFixture("modernc-libc-xsys-v041"), func(request *Request) { firstAdapter(request).ProfileName = "another-profile/v1" }},
		{"adapter implementation", fromFixture("modernc-libc-xsys-v041"), func(request *Request) { firstAdapter(request).ProfileImplementationSHA256 = digestOf("e") }},
		{"adapter module", fromFixture("modernc-libc-xsys-v041"), func(request *Request) { firstAdapter(request).Module = "example.com/adapter" }},
		{"adapter version", fromFixture("modernc-libc-xsys-v041"), func(request *Request) { firstAdapter(request).Version = "v1.72.4" }},
		{"adapter sum", fromFixture("modernc-libc-xsys-v041"), func(request *Request) { firstAdapter(request).Sum = moduleSum("B") }},
		{"adapter original inventory", fromFixture("modernc-libc-xsys-v041"), func(request *Request) { firstAdapter(request).OriginalSourceInventorySHA256 = digestOf("e") }},
		{"adapter replacement inventory", fromFixture("modernc-libc-xsys-v041"), func(request *Request) { firstAdapter(request).ReplacementSourceInventorySHA256 = digestOf("e") }},
		{"adapter prepared source set", fromFixture("modernc-libc-xsys-v041"), func(request *Request) { firstAdapter(request).PreparedSourceSetSHA256 = digestOf("e") }},
	}
	for _, test := range mutations {
		t.Run(test.name, func(t *testing.T) {
			request := test.request(t)
			digest, err := ApprovalSHA256(request)
			if err != nil {
				t.Fatal(err)
			}
			test.mutate(&request)
			changedDigest, err := ApprovalSHA256(request)
			if err != nil {
				t.Fatalf("mutation is not a valid reviewed request: %v", err)
			}
			if changedDigest == digest {
				t.Fatalf("mutation did not change approval digest: %q", changedDigest)
			}
		})
	}
}

func retainedRequest(t *testing.T, name string) Request {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "requests", name+".json"))
	if err != nil {
		t.Fatal(err)
	}
	request, err := DecodeRequest(data)
	if err != nil {
		t.Fatal(err)
	}
	return request
}

func firstForeignSource(request *Request) (int, *compatibility.PackForeignSource) {
	for index := range request.Packages {
		if len(request.Packages[index].Evidence.ForeignSources) != 0 {
			return index, &request.Packages[index].Evidence.ForeignSources[0]
		}
	}
	panic("request has no foreign source")
}

func firstLinknameFact(request *Request) *Fact {
	for packageIndex := range request.Packages {
		for factIndex := range request.Packages[packageIndex].Facts {
			fact := &request.Packages[packageIndex].Facts[factIndex]
			if fact.Kind == FactLinkname {
				return fact
			}
		}
	}
	panic("request has no linkname fact")
}

func firstAdapter(request *Request) *compatibility.PackAdapter {
	for index := range request.Activation {
		if adapter := request.Activation[index].Evidence.Replacement.Adapter; adapter != nil {
			return adapter
		}
	}
	for index := range request.Packages {
		if adapter := request.Packages[index].Evidence.Module.Replacement.Adapter; adapter != nil {
			return adapter
		}
	}
	panic("request has no adapter")
}

func refreshSourceSet(request *Request, packageIndex int) {
	rule := &request.Packages[packageIndex].Evidence
	sources := make([]compatibility.Source, 0, len(rule.GoSources)+len(rule.ForeignSources))
	for _, source := range rule.GoSources {
		sources = append(sources, compatibility.Source(source))
	}
	for _, source := range rule.ForeignSources {
		sources = append(sources, compatibility.Source{Name: source.Kind + ":" + source.Name, SHA256: source.SHA256})
	}
	rule.SourceSetSHA256 = compatibility.DigestSources(sources)
}

func digestOf(character string) string {
	return "sha256:" + strings.Repeat(character, 64)
}

func moduleSum(character string) string {
	return "h1:" + strings.Repeat(character, 43) + "="
}

func TestStdlibEncodingDerivedCompatibilityIdentities(t *testing.T) {
	request := validRequest()
	approval, err := ApprovalSHA256(request)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := Generate(root, request, approval); err != nil {
		t.Fatal(err)
	}
	requestBytes, err := os.ReadFile(filepath.Join(root, "requests", request.ID+".json"))
	if err != nil {
		t.Fatal(err)
	}
	packBytes, err := os.ReadFile(filepath.Join(root, "packs", request.ID+".json"))
	if err != nil {
		t.Fatal(err)
	}
	stateBytes, err := os.ReadFile(filepath.Join(root, "generation.json"))
	if err != nil {
		t.Fatal(err)
	}
	pack, err := compatibility.DecodePack(packBytes)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(record.DomainHash("gomad.compatibility-pack-request/v1", requestBytes)); got != pack.RequestSHA256 {
		t.Fatalf("request projection identity = %s, pack records %s", got, pack.RequestSHA256)
	}
	checks := map[string][2]string{
		"approval":         {approval, "sha256:099f7e6b230df2a539e68da1dadbe9430e781b94ebeca6712de17a5e77f67236"},
		"request":          {pack.RequestSHA256, "sha256:337b3bf3a5d550188c495ba78ad268b88a8cd57d84a4cc27392e001146176821"},
		"pack bytes":       {string(record.HashBytes(packBytes)), "sha256:12b6cfc8f4acc4eecf6e04329fdafcd3005c75163a69eb3b114be4d0dba3e61e"},
		"generation bytes": {string(record.HashBytes(stateBytes)), "sha256:3614888e35a5987d537769eca3ae7edb3741afbcd04b52b6ed86fb4183026780"},
	}
	for name, check := range checks {
		if check[0] != check[1] {
			t.Errorf("%s identity = %s, want %s", name, check[0], check[1])
		}
	}
}

func TestRequestDecodingSeparatesSpellingFromStrictAndPolicyFailures(t *testing.T) {
	encoded, err := json.Marshal(validRequest())
	if err != nil {
		t.Fatal(err)
	}
	spelling := bytes.Replace(encoded, []byte(`{"schema":`), []byte("{\n  \"SCHEMA\" : "), 1)
	decoded, err := DecodeRequest(spelling)
	if err != nil || decoded.ID != validRequest().ID {
		t.Fatalf("spelling-only request = %#v, %v", decoded, err)
	}
	invalidUTF8 := append([]byte(`{"schema":"`), 0xff)
	invalidUTF8 = append(invalidUTF8, []byte(`"}`)...)
	for _, test := range []struct {
		name string
		data []byte
	}{
		{"unknown field", bytes.Replace(encoded, []byte(`{"schema":`), []byte(`{"unknown":true,"schema":`), 1)},
		{"duplicate field", bytes.Replace(encoded, []byte(`{"schema":`), []byte(`{"schema":"gomad.compatibility-pack-request/v1","schema":`), 1)},
		{"trailing input", append(append([]byte{}, encoded...), []byte(` {}`)...)},
		{"invalid UTF-8", invalidUTF8},
		{"capacity", bytes.Repeat([]byte{' '}, MaximumRequestBytes+1)},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := DecodeRequest(test.data); err == nil {
				t.Fatal("DecodeRequest() accepted invalid input")
			}
		})
	}
}

func TestRequestValidationRejectsInvalidOriginalStrings(t *testing.T) {
	for _, mutate := range []func(*Request){
		func(request *Request) { request.Owner = "\xff" },
		func(request *Request) { request.Target.Package = "./\xff" },
		func(request *Request) { request.Target.TestArguments[0] = "\xff" },
		func(request *Request) { request.Target.BuildTags[0] = "\xff" },
		func(request *Request) { request.Packages[0].Facts[0].Capability = "import:\xff" },
	} {
		request := validRequest()
		mutate(&request)
		if err := ValidateRequest(request); err == nil {
			t.Fatalf("ValidateRequest() accepted invalid original strings: %#v", request)
		}
	}
}

func TestDecodeDraftRequestAcceptsSelectorsWithoutDiscoveredEvidence(t *testing.T) {
	draft := validRequest()
	draft.Activation[0].Evidence = compatibility.PackModule{}
	draft.Packages[0].Evidence = compatibility.PackRule{}
	encoded, err := json.Marshal(draft)
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
			encoded, err := json.Marshal(request)
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
				SourceSetSHA256: "sha256:6610ebeabfa4432215e0233a5bc29e63db3d180d036979bff63bb23764345ff4",
				GoSources:       []compatibility.PackSource{{Name: "runtime.go", SHA256: "sha256:" + strings.Repeat("4", 64)}},
				ForeignSources:  []compatibility.PackForeignSource{}, Capabilities: []string{}, Linknames: []compatibility.PackLinkname{},
			},
		}},
		Owner: "runtime-team", ReviewedAt: "2026-08-15T00:00:00Z",
		Justification: "Allows one reviewed dependency capability.",
		Workloads:     []string{"core-fixture"}, Platforms: []string{"darwin/arm64"}, ApprovalSHA256: "",
	}
}
