package deterministicio

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRegenerateSourceRewriteRequiresExactlyOneAnchor(t *testing.T) {
	rewrite := sourceRewrite{path: "source.go", rewrites: []anchorRewrite{{anchor: []byte("target"), replacement: []byte("replacement")}}}
	for _, test := range []struct {
		name, source, want string
	}{
		{name: "moved", source: "changed", want: "anchor mismatch"},
		{name: "duplicated", source: "target target", want: "anchor mismatch"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := regenerateSourceRewrite("example.com/module", rewrite, []byte(test.source))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("regenerateSourceRewrite() error = %v, want %q", err, test.want)
			}
		})
	}
	contents, err := regenerateSourceRewrite("example.com/module", rewrite, []byte("a target b"))
	if err != nil || string(contents) != "a replacement b" {
		t.Fatalf("regenerateSourceRewrite() = %q, %v", contents, err)
	}
}

func TestRegenerateRewrittenModuleRejectsMissingSource(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "other.go"), []byte("package example\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	recipe := rewrittenModule{module: "example.com/module", replacementDirectory: "replacement", preparedPackage: "example.com/module", rewrites: []sourceRewrite{{path: "missing.go", rewrites: []anchorRewrite{{anchor: []byte("target")}}}}}
	_, err := regenerateRewrittenModule(root, t.TempDir(), recipe)
	if err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("regenerateRewrittenModule() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "missing.go")); !os.IsNotExist(err) {
		t.Fatalf("unexpected source: %v", err)
	}
}

func TestRegenerateRewrittenModuleBindsInventory(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "source.go"), []byte("package example\n\nconst target = 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	recipe := rewrittenModule{module: "example.com/module", replacementDirectory: "replacement", preparedPackage: "example.com/module", rewrites: []sourceRewrite{{path: "source.go", rewrites: []anchorRewrite{{anchor: []byte("target"), replacement: []byte("changed")}}}}}
	result, err := regenerateRewrittenModule(root, t.TempDir(), recipe)
	if err != nil || len(result.Sources) != 1 || !strings.HasPrefix(result.OriginalInventorySHA256, "sha256:") || !strings.HasPrefix(result.ReplacementInventorySHA256, "sha256:") || result.OriginalInventorySHA256 == result.ReplacementInventorySHA256 {
		t.Fatalf("regenerateRewrittenModule() = %#v, %v", result, err)
	}
}
