package architecture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDependencyInitialization(t *testing.T) {
	for _, test := range []struct {
		name, imports, source, child string
		effect                       bool
	}{
		{"called-dependency", `"example.invalid/initialization/internal/canonicaljson"`, `import "time";func init(){_=time.Now()};func Clean(){}`, "", true},
		{"blank-import", `_ "example.invalid/initialization/internal/canonicaljson"`, `import "time";func init(){_=time.Now()}`, "", true},
		{"variable-initializer", `_ "example.invalid/initialization/internal/canonicaljson"`, `import "time";var stamp=time.Now()`, "", true},
		{"second-init", `_ "example.invalid/initialization/internal/canonicaljson"`, `import "time";func init(){_=time.Now()};func init(){}`, "", true},
		{"transitive-import", `_ "example.invalid/initialization/internal/canonicaljson"`, `import _ "example.invalid/initialization/internal/canonicaljson/child"`, `import "time";func init(){_=time.Now()}`, true},
		{"pure-initializers", `_ "example.invalid/initialization/internal/canonicaljson"`, `var n=func()int{return 1}();func init(){n++}`, "", false},
		{"once-callback", `_ "example.invalid/initialization/internal/canonicaljson"`, `import("time";"sync");var once sync.Once;func init(){once.Do(func(){_=time.Now()})}`, "", true},
		{"pure-once", `_ "example.invalid/initialization/internal/canonicaljson"`, `import "sync";var once sync.Once;var n int;func init(){once.Do(func(){n++})}`, "", false},
		{"standard-startup", `_ "time";_ "crypto/sha256";_ "encoding/json";_ "net";_ "crypto/rand"`, `func Clean(){}`, "", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			files := map[string]string{
				"go.mod": "module example.invalid/initialization\n\ngo 1.27.1\n",
				"record/pure.go": "package record;import(" + test.imports + ");func Check(){" + func() string {
					if test.name == "called-dependency" {
						return "helper.Clean()"
					}
					return ""
				}() + "}",
				"internal/canonicaljson/helper.go": "package helper;" + test.source,
			}
			if test.child != "" {
				files["internal/canonicaljson/child/child.go"] = "package child;" + test.child
			}
			for name, source := range files {
				path := filepath.Join(root, name)
				if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(source), 0600); err != nil {
					t.Fatal(err)
				}
			}
			for _, platform := range []Platform{{"linux", "amd64"}, {"darwin", "arm64"}} {
				program, err := Load(root, "go", platform)
				if err != nil {
					t.Fatal(err)
				}
				var packages []Package
				for _, metadata := range program.Metadata {
					packages = append(packages, metadata)
				}
				if findings := PackageEdges(program.Module, packages); len(findings) != 0 {
					t.Fatalf("invalid fixture edges: %v", findings)
				}
				findings := program.Effects()
				if !test.effect {
					if len(findings) != 0 {
						t.Fatalf("pure startup rejected %s: %v", platform, findings)
					}
					continue
				}
				found := false
				for _, finding := range findings {
					if finding.Category == "host-effect" && strings.Contains(finding.Detail, "initializer") && strings.Contains(finding.Detail, "time.Now") {
						found = true
					}
				}
				if !found {
					t.Fatalf("dependency clock escaped %s: %v", platform, findings)
				}
			}
		})
	}
}

func TestThirdPartyInitialization(t *testing.T) {
	for _, effect := range []bool{true, false} {
		t.Run(map[bool]string{true: "clock", false: "pure"}[effect], func(t *testing.T) {
			root, dependency := t.TempDir(), t.TempDir()
			initializer := `var n=1;func init(){n++}`
			if effect {
				initializer = `import "time";func init(){_=time.Now()}`
			}
			for path, source := range map[string]string{
				filepath.Join(root, "go.mod"):              "module example.invalid/initialization\n\ngo 1.27.1\nrequire example.invalid/dependency v0.0.0\nreplace example.invalid/dependency => " + dependency + "\n",
				filepath.Join(root, "record", "pure.go"):   `package record;import _ "example.invalid/dependency";func Check(){}`,
				filepath.Join(dependency, "go.mod"):        "module example.invalid/dependency\n\ngo 1.27.1\n",
				filepath.Join(dependency, "dependency.go"): "package dependency;" + initializer,
			} {
				if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(source), 0600); err != nil {
					t.Fatal(err)
				}
			}
			for _, platform := range []Platform{{"linux", "amd64"}, {"darwin", "arm64"}} {
				program, err := Load(root, "go", platform)
				if err != nil {
					t.Fatal(err)
				}
				findings := program.Effects()
				if !effect {
					if len(findings) != 0 {
						t.Fatalf("pure dependency rejected: %v", findings)
					}
					continue
				}
				found := false
				for _, finding := range findings {
					if finding.Category == "host-effect" && strings.Contains(finding.Detail, "example.invalid/dependency:initializer") && strings.Contains(finding.Detail, "time.Now") {
						found = true
					}
				}
				if !found {
					t.Fatalf("third-party initializer escaped: %v", findings)
				}
			}
		})
	}
}

func TestStandardStartupIdentity(t *testing.T) {
	root := t.TempDir()
	for name, source := range map[string]string{
		"go.mod":         "module example.invalid/initialization\n\ngo 1.27.1\n",
		"record/pure.go": `package record;import _ "time";func Check(){}`,
	} {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	program, err := Load(root, "go", Platform{"linux", "amd64"})
	if err != nil {
		t.Fatal(err)
	}
	metadata := program.Metadata["time"]
	metadata.Dir = t.TempDir()
	if err := os.WriteFile(filepath.Join(metadata.Dir, "changed.go"), []byte("package time"), 0600); err != nil {
		t.Fatal(err)
	}
	program.Metadata["time"] = metadata
	for _, finding := range program.Effects() {
		if finding.Category == "unresolved-effect" && strings.Contains(finding.Detail, "pinned standard startup source changed: time") {
			return
		}
	}
	t.Fatal("changed standard startup source was accepted")
}
