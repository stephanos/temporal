package deterministicio

import (
	"encoding/json"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	compatibility "go.temporal.io/server/tools/gomad3/internal/compatibilitypack"
	"go.temporal.io/server/tools/gomad3/target"
	gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"
)

func validatorAdapterIdentity() gomadversion.AdapterIdentity {
	return gomadversion.AdapterIdentity{Module: validatorModulePath, Version: validatorVersion, Sum: validatorSum}
}

func TestValidatorAdapterConsumer(t *testing.T) {
	downloadPinnedModule(t, validatorModulePath, validatorVersion)
	workingDirectory := t.TempDir()
	for _, name := range []string{"go.mod", "go.sum", "validator_test.go"} {
		contents, err := os.ReadFile(filepath.Join("testdata", "validator", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(workingDirectory, name), contents, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	registry, err := newAdapterRegistry([]gomadversion.AdapterIdentity{validatorAdapterIdentity()}, []adapterImplementation{
		{module: validatorModulePath, prepare: prepareValidator},
	})
	if err != nil {
		t.Fatal(err)
	}
	spec, adapters, err := registry.prepare(target.Spec{
		Kind: target.KindGoTest, Source: ".", WorkingDir: workingDirectory, PreparationRoot: t.TempDir(),
	}, pinnedModuleCache(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(adapters) != 1 || adapters[0].Module != validatorModulePath {
		t.Fatalf("Validator consumer adapter selection = %#v", adapters)
	}
	goCommand, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	run := func(dir string, environment []string, args ...string) []byte {
		t.Helper()
		command := exec.CommandContext(t.Context(), goCommand, args...)
		command.Dir = dir
		command.Env = append(os.Environ(), append([]string{"GOWORK=off", "GOFLAGS="}, environment...)...)
		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("prepared Validator consumer %v: %v\n%s", args, err, output)
		}
		return output
	}
	output := run(workingDirectory, nil, "test", "-v", "-p=2", "-buildvcs=false", "-mod=readonly", "-modfile="+spec.BuildModFile, ".")
	t.Logf("prepared stock Validator consumer:\n%s", output)
	for _, platform := range []struct{ goos, goarch string }{{"darwin", "arm64"}, {"linux", "amd64"}} {
		environment := []string{"GOOS=" + platform.goos, "GOARCH=" + platform.goarch, "CGO_ENABLED=0"}
		output := run(workingDirectory, environment, "list", "-mod=readonly", "-modfile="+spec.BuildModFile, "-json", validatorModulePath)
		var pkg struct {
			Dir     string
			GoFiles []string
		}
		if err := json.Unmarshal(output, &pkg); err != nil {
			t.Fatal(err)
		}
		slices.Sort(pkg.GoFiles)
		sources := make([]compatibility.Source, len(pkg.GoFiles))
		for index, name := range pkg.GoFiles {
			contents, err := os.ReadFile(filepath.Join(pkg.Dir, name))
			if err != nil {
				t.Fatal(err)
			}
			sources[index] = compatibility.Source{Name: name, SHA256: digestBytes(contents)}
		}
		if got := compatibility.DigestSources(sources); got != validatorPreparedSourceSetSHA256 {
			t.Fatalf("prepared Validator source set on %s/%s = %s, want %s", platform.goos, platform.goarch, got, validatorPreparedSourceSetSHA256)
		}
		t.Logf("prepared %s/%s source set %s", platform.goos, platform.goarch, validatorPreparedSourceSetSHA256)
	}
}

func TestValidatorRejectsChangedIdentity(t *testing.T) {
	for _, identity := range []gomadversion.AdapterIdentity{
		{Module: "github.com/go-playground/validator/v10/other", Version: validatorVersion, Sum: validatorSum},
		{Module: validatorModulePath, Version: "v0.0.0", Sum: validatorSum},
		{Module: validatorModulePath, Version: validatorVersion, Sum: "h1:changed"},
	} {
		if _, err := prepareValidator(t.TempDir(), t.TempDir(), identity); err == nil || !strings.Contains(err.Error(), "identity mismatch") {
			t.Fatalf("changed Validator identity %#v: %v", identity, err)
		}
	}
}

func TestValidatorRewriteRejectsDrift(t *testing.T) {
	downloadPinnedModule(t, validatorModulePath, validatorVersion)
	moduleRoot := filepath.Join(pinnedModuleCache(t), "github.com", "go-playground", "validator", "v10@"+validatorVersion)
	source, err := readAdapterSource(validatorModulePath, moduleRoot, validatorBakedInPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name, want string
		change     func(*sourceRewrite, *[]byte)
	}{
		{name: "upstream-file", want: "source identity mismatch", change: func(_ *sourceRewrite, source *[]byte) { *source = append(*source, '\n') }},
		{name: "missing-anchor", want: "anchor mismatch", change: func(rewrite *sourceRewrite, _ *[]byte) {
			rewrite.rewrites[0].anchor = []byte("absent resolver callback")
		}},
		{name: "ambiguous-anchor", want: "anchor mismatch", change: func(rewrite *sourceRewrite, _ *[]byte) { rewrite.rewrites[0].anchor = []byte("fl") }},
		{name: "replacement-digest", want: "replacement identity mismatch", change: func(rewrite *sourceRewrite, _ *[]byte) { rewrite.replacementSHA256 = validatorBakedInSourceSHA256 }},
	} {
		t.Run(test.name, func(t *testing.T) {
			rewrite := validatorRewrites[0]
			rewrite.rewrites = append([]anchorRewrite(nil), rewrite.rewrites...)
			contents := append([]byte(nil), source...)
			test.change(&rewrite, &contents)
			if _, err := rewriteAdapterSource(validatorModulePath, rewrite, contents); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Validator rewrite: %v, want %s", err, test.want)
			}
		})
	}
}

func TestValidatorRewritePreservesComments(t *testing.T) {
	downloadPinnedModule(t, validatorModulePath, validatorVersion)
	moduleRoot := filepath.Join(pinnedModuleCache(t), "github.com", "go-playground", "validator", "v10@"+validatorVersion)
	source, err := readAdapterSource(validatorModulePath, moduleRoot, validatorBakedInPath)
	if err != nil {
		t.Fatal(err)
	}
	replacement, err := rewriteAdapterSource(validatorModulePath, validatorRewrites[0], source)
	if err != nil {
		t.Fatal(err)
	}
	comments := func(contents []byte) []string {
		t.Helper()
		file, err := parser.ParseFile(token.NewFileSet(), validatorBakedInPath, contents, parser.ParseComments)
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
		t.Fatal("Validator adapter changed original comments")
	}
}

func TestValidatorRejectsUnrewrittenInventoryDrift(t *testing.T) {
	downloadPinnedModule(t, validatorModulePath, validatorVersion)
	moduleCache := t.TempDir()
	moduleRoot := filepath.Join(moduleCache, "github.com", "go-playground", "validator", "v10@"+validatorVersion)
	if err := os.MkdirAll(filepath.Dir(moduleRoot), 0o700); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(pinnedModuleCache(t), "github.com", "go-playground", "validator", "v10@"+validatorVersion)
	if err := copyAdapterModule(source, moduleRoot, nil, defaultAdapterCopyLimits); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(moduleRoot, "validator.go")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(contents, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := prepareValidator(moduleCache, t.TempDir(), validatorAdapterIdentity()); err == nil || !strings.Contains(err.Error(), "source inventory identity mismatch") {
		t.Fatalf("changed Validator module inventory: %v", err)
	}
}
