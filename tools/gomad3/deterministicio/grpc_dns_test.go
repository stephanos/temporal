package deterministicio

import (
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"
)

func TestGRPCDNSRewrite(t *testing.T) {
	downloadPinnedModule(t, grpcModulePath, grpcVersion)
	rewrite := grpcDNSRewrites[0]
	moduleRoot := filepath.Join(pinnedModuleCache(t), "google.golang.org", "grpc@"+grpcVersion)
	source, err := readGRPCAdapterSource(moduleRoot, rewrite.path)
	if err != nil {
		t.Fatal(err)
	}
	replacement, err := rewriteAdapterSource(grpcModulePath, rewrite, source)
	if err != nil {
		t.Fatal(err)
	}
	comments := func(contents []byte) []string {
		t.Helper()
		file, err := parser.ParseFile(token.NewFileSet(), rewrite.path, contents, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		var result []string
		for _, group := range file.Comments {
			for _, comment := range group.List {
				result = append(result, comment.Text)
			}
		}
		return result
	}
	if !reflect.DeepEqual(comments(source), comments(replacement)) {
		t.Fatal("gRPC DNS adapter changed original comments")
	}
	for _, test := range []struct {
		name, want string
		change     func(*sourceRewrite, *[]byte)
	}{
		{name: "source", want: "source identity mismatch", change: func(_ *sourceRewrite, source *[]byte) { *source = append(*source, '\n') }},
		{name: "missing-anchor", want: "anchor mismatch", change: func(rewrite *sourceRewrite, _ *[]byte) {
			rewrite.rewrites[0].anchor = []byte("absent resolver factory")
		}},
		{name: "ambiguous-anchor", want: "anchor mismatch", change: func(rewrite *sourceRewrite, _ *[]byte) { rewrite.rewrites[0].anchor = []byte("return nil") }},
		{name: "replacement", want: "replacement identity mismatch", change: func(rewrite *sourceRewrite, _ *[]byte) { rewrite.replacementSHA256 = rewrite.sourceSHA256 }},
	} {
		t.Run(test.name, func(t *testing.T) {
			changed := rewrite
			changed.rewrites = append([]anchorRewrite(nil), rewrite.rewrites...)
			contents := append([]byte(nil), source...)
			test.change(&changed, &contents)
			if _, err := rewriteAdapterSource(grpcModulePath, changed, contents); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("gRPC DNS rewrite: %v, want %s", err, test.want)
			}
		})
	}
}

func TestGRPCDNSConsumer(t *testing.T) {
	downloadPinnedModule(t, grpcModulePath, grpcVersion)
	prepared, err := prepareGRPC(pinnedModuleCache(t), t.TempDir(), gomadversion.AdapterIdentity{
		Module: grpcModulePath, Version: grpcVersion, Sum: grpcSum,
	})
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := os.ReadFile(filepath.Join("testdata", "grpcdns", "dns_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(prepared.replacement, "internal", "resolver", "dns", "gomad_dns_test.go"), fixture, 0o600); err != nil {
		t.Fatal(err)
	}
	goCommand, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	command := exec.CommandContext(t.Context(), goCommand, "test", "-v", "-p=2", "-mod=readonly", "-run=^TestGomadDNS", "./internal/resolver/dns")
	command.Dir = prepared.replacement
	command.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("prepared gRPC DNS consumer: %v\n%s", err, output)
	}
	t.Logf("prepared gRPC DNS consumer:\n%s", output)
}
