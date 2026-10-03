package deterministicio

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/target"
	gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"
	"golang.org/x/mod/modfile"
)

func memberlistAdapterIdentity() gomadversion.AdapterIdentity {
	return gomadversion.AdapterIdentity{Module: memberlistModulePath, Version: memberlistVersion, Sum: memberlistSum}
}

func TestMemberlistSuppliedTCPConsumer(t *testing.T) {
	workingDirectory := os.Getenv("GOMAD_MEMBERLIST_TCP_CONSUMER_DIR")
	if workingDirectory == "" {
		t.Skip("set GOMAD_MEMBERLIST_TCP_CONSUMER_DIR to a consumer checkout for its real TCP membership lifecycle")
	}
	modulePath := os.Getenv("GOMAD_MEMBERLIST_TCP_CONSUMER_MODULE")
	moduleDirectory := filepath.Join(workingDirectory, modulePath)
	downloadPinnedModule(t, memberlistModulePath, memberlistVersion)
	registry, err := newAdapterRegistry([]gomadversion.AdapterIdentity{memberlistAdapterIdentity()}, []adapterImplementation{
		{module: memberlistModulePath, prepare: prepareMemberlist},
	})
	if err != nil {
		t.Fatal(err)
	}
	spec, adapters, err := registry.prepare(target.Spec{
		Kind: target.KindGoTest, Source: "./testutil", WorkingDir: moduleDirectory, PreparationRoot: t.TempDir(),
	}, pinnedModuleCache(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(adapters) != 1 || adapters[0].Module != memberlistModulePath {
		t.Fatalf("TCP consumer adapters = %#v", adapters)
	}
	contents, err := os.ReadFile(spec.BuildModFile)
	if err != nil {
		t.Fatal(err)
	}
	moduleFile, err := modfile.Parse(spec.BuildModFile, contents, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, replacement := range moduleFile.Replace {
		if replacement.New.Version == "" && !filepath.IsAbs(replacement.New.Path) {
			if err := moduleFile.AddReplace(replacement.Old.Path, replacement.Old.Version, filepath.Join(spec.WorkingDir, replacement.New.Path), ""); err != nil {
				t.Fatal(err)
			}
		}
	}
	contents, err = moduleFile.Format()
	if err != nil {
		t.Fatal(err)
	}
	wrapperModFile := filepath.Join(t.TempDir(), "wrapper.mod")
	if err := os.WriteFile(wrapperModFile, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	sums, err := os.ReadFile(strings.TrimSuffix(spec.BuildModFile, ".mod") + ".sum")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(strings.TrimSuffix(wrapperModFile, ".mod")+".sum", sums, 0o600); err != nil {
		t.Fatal(err)
	}
	spec.BuildModFile = wrapperModFile
	goCommand, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	selection := exec.CommandContext(t.Context(), goCommand, "list", "-mod=readonly", "-modfile="+spec.BuildModFile, "-json", memberlistModulePath)
	selection.Dir = spec.WorkingDir
	selection.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=")
	output, err := selection.CombinedOutput()
	if err != nil {
		t.Fatalf("select prepared Memberlist in real TCP consumer: %v\n%s", err, output)
	}
	var pkg struct {
		Dir    string
		Module struct{ Path, Version string }
	}
	if err := json.Unmarshal(output, &pkg); err != nil {
		t.Fatal(err)
	}
	if pkg.Dir != adapters[0].ReplacementRoot || pkg.Module.Path != memberlistModulePath || pkg.Module.Version != memberlistVersion {
		t.Fatalf("real TCP consumer selected %#v; want %s@%s from %s", pkg, memberlistModulePath, memberlistVersion, adapters[0].ReplacementRoot)
	}
	t.Logf("real TCP consumer selected %s@%s, replacement inventory %s, source set %s", pkg.Module.Path, pkg.Module.Version, adapters[0].ReplacementSourceInventorySHA256, adapters[0].PreparedSourceSetSHA256)
	mise, err := exec.LookPath("mise")
	if err != nil {
		t.Fatal(err)
	}
	command := exec.CommandContext(t.Context(), mise, "run", "test", "--run", "^TestTCPTransport_MembershipUpdateFailureAndRejoin$", "--tags=test_dep,hashicorpmetrics", "-t", "3m", "-o", "fn107-memberlist-tcp.log", "./"+filepath.ToSlash(filepath.Join(modulePath, "testutil")))
	command.Dir = workingDirectory
	command.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=-p=2 -modfile="+spec.BuildModFile)
	output, err = command.CombinedOutput()
	if err != nil {
		t.Fatalf("prepared real TCP memberlist lifecycle: %v\n%s", err, output)
	}
	t.Logf("prepared real TCP memberlist lifecycle:\n%s", output)
}
