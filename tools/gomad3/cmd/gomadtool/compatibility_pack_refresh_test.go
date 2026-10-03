package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/compatibilitypack/authoring"
	"go.temporal.io/server/tools/gomad3/upgrade"
	"golang.org/x/mod/modfile"
)

func TestCompatibilityPackRefreshUsesEachMappedModuleAndPreservesPartialApproval(t *testing.T) {
	root := t.TempDir()
	compatibilityRoot := filepath.Join(root, "internal", "compatibilitypack")
	if err := os.MkdirAll(filepath.Join(compatibilityRoot, "requests"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(compatibilityRoot, "reports"), 0o700); err != nil {
		t.Fatal(err)
	}
	ids := []string{"modernc-libc-xsys-v047", "reflect2-go126"}
	modules := []struct{ name, version string }{{"golang.org/x/sys", "v0.48.0"}, {"github.com/modern-go/reflect2", "v1.0.4"}}
	for index, id := range ids {
		original, err := os.ReadFile(filepath.Join("..", "..", "internal", "compatibilitypack", "requests", id+".json"))
		if err != nil {
			t.Fatal(err)
		}
		request, err := authoring.DecodeRequest(original)
		if err != nil {
			t.Fatal(err)
		}
		if err := authoring.PublishRequest(filepath.Join(compatibilityRoot, "requests", id+".json"), request); err != nil {
			t.Fatal(err)
		}
		moduleDir := filepath.Join(root, id)
		if err := os.Mkdir(moduleDir, 0o700); err != nil {
			t.Fatal(err)
		}
		goMod := "module " + request.Target.ExpectedModule + "\n\ngo 1.27.1\n\nrequire " + modules[index].name + " " + modules[index].version + "\n"
		if err := os.WriteFile(filepath.Join(moduleDir, "go.mod"), []byte(goMod), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	table := "darwin/arm64 modernc-libc-xsys-v047 modernc-libc-xsys-v047\ndarwin/arm64 reflect2-go126 reflect2-go126\n"
	if err := os.WriteFile(filepath.Join(compatibilityRoot, "targets.tsv"), []byte(table), 0o600); err != nil {
		t.Fatal(err)
	}
	targets, err := readPackTargets(filepath.Join(compatibilityRoot, "targets.tsv"), root)
	if err != nil {
		t.Fatal(err)
	}
	impact := upgrade.PinImpact{Schema: "gomad3.pin-impact/v1", Pins: []upgrade.PinResult{
		{Class: "pack_rule", ID: ids[0] + ":golang.org/x/sys/unix", Status: "invalidated"},
		{Class: "pack_rule", ID: ids[1] + ":github.com/modern-go/reflect2", Status: "invalidated"},
	}}
	invalidated, err := invalidatedPackRequests(impact, targets)
	if err != nil {
		t.Fatal(err)
	}
	discover := func(request authoring.Request, working, _ string) (authoring.Request, string, error) {
		data, err := os.ReadFile(filepath.Join(working, "go.mod"))
		if err != nil {
			return authoring.Request{}, "", err
		}
		parsed, err := modfile.Parse("go.mod", data, nil)
		if err != nil {
			return authoring.Request{}, "", err
		}
		fresh := request
		fresh.ApprovalSHA256 = ""
		fresh.Activation = append([]authoring.Activation(nil), request.Activation...)
		fresh.Packages = append([]authoring.Package(nil), request.Packages...)
		for _, requirement := range parsed.Require {
			for index := range fresh.Activation {
				if fresh.Activation[index].Path == requirement.Mod.Path {
					fresh.Activation[index].Evidence.Version = requirement.Mod.Version
				}
			}
			for index := range fresh.Packages {
				if fresh.Packages[index].Evidence.Module.Path == requirement.Mod.Path {
					fresh.Packages[index].Evidence.Module.Version = requirement.Mod.Version
				}
			}
		}
		digest, err := authoring.ApprovalSHA256(fresh)
		return fresh, digest, err
	}
	var stdout, stderr bytes.Buffer
	if status := refreshCompatibilityPacks(root, compatibilityRoot, "darwin/arm64", invalidated, targets, discover, &stdout, &stderr); status != 0 {
		t.Fatalf("refresh status %d: %s", status, stderr.String())
	}
	for index, id := range ids {
		data, err := os.ReadFile(filepath.Join(compatibilityRoot, "requests", id+".json"))
		if err != nil {
			t.Fatal(err)
		}
		request, err := authoring.DecodeRequest(data)
		if err != nil {
			t.Fatal(err)
		}
		if request.ApprovalSHA256 != "" || !strings.Contains(stdout.String(), id+": review ") {
			t.Fatalf("request %s retained stale approval or lost digest: %s", id, stdout.String())
		}
		selected := false
		for _, activation := range request.Activation {
			if activation.Path == modules[index].name {
				selected = activation.Evidence.Version == modules[index].version
			}
		}
		if !selected {
			t.Fatalf("request %s did not use mapped module candidate %s", id, modules[index].version)
		}
	}
	approvedPath := filepath.Join(compatibilityRoot, "requests", ids[0]+".json")
	data, err := os.ReadFile(approvedPath)
	if err != nil {
		t.Fatal(err)
	}
	approved, err := authoring.DecodeRequest(data)
	if err != nil {
		t.Fatal(err)
	}
	approval, err := authoring.ApprovalSHA256(approved)
	if err != nil {
		t.Fatal(err)
	}
	if err := authoring.Generate(compatibilityRoot, approved, approval); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if status := refreshCompatibilityPacks(root, compatibilityRoot, "darwin/arm64", invalidated, targets, discover, &stdout, &stderr); status != 0 {
		t.Fatalf("rerun status %d: %s", status, stderr.String())
	}
	if strings.Contains(stdout.String(), ids[0]) || !strings.Contains(stdout.String(), ids[1]+": review ") {
		t.Fatalf("rerun did not preserve partial progress: %s", stdout.String())
	}
}

func TestCompatibilityPackRefreshRejectsUnmappedRequest(t *testing.T) {
	_, err := invalidatedPackRequests(upgrade.PinImpact{Pins: []upgrade.PinResult{{Class: "pack_rule", ID: "missing:example.com/pkg", Status: "invalidated"}}}, map[string]packTarget{})
	if err == nil || !strings.Contains(err.Error(), "no target mapping") {
		t.Fatalf("unmapped request error = %v", err)
	}
}

func TestCompatibilityPackRefreshLeavesOtherPlatformUntouched(t *testing.T) {
	root := t.TempDir()
	compatibilityRoot := filepath.Join(root, "internal", "compatibilitypack")
	requestPath := filepath.Join(compatibilityRoot, "requests", "modernc-libc-xsys-v047-linux-amd64.json")
	if err := os.MkdirAll(filepath.Dir(requestPath), 0o700); err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(filepath.Join("..", "..", "internal", "compatibilitypack", "requests", "modernc-libc-xsys-v047-linux-amd64.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(requestPath, original, 0o600); err != nil {
		t.Fatal(err)
	}
	targets := map[string]packTarget{"modernc-libc-xsys-v047-linux-amd64": {platform: "linux/amd64", working: root}}
	var stdout, stderr bytes.Buffer
	discover := func(authoring.Request, string, string) (authoring.Request, string, error) {
		t.Fatal("other-platform request was evaluated")
		return authoring.Request{}, "", nil
	}
	if status := refreshCompatibilityPacks(root, compatibilityRoot, "darwin/arm64", []string{"modernc-libc-xsys-v047-linux-amd64"}, targets, discover, &stdout, &stderr); status != 0 {
		t.Fatalf("refresh status %d: %s", status, stderr.String())
	}
	after, err := os.ReadFile(requestPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(original, after) || !strings.Contains(stdout.String(), "not evaluable") {
		t.Fatalf("other-platform request changed: %s", stdout.String())
	}
}
