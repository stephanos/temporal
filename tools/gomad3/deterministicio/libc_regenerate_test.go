package deterministicio

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLibcRegenerationKeepsExactSourceAndPreparedPins(t *testing.T) {
	if !containsString(RegenerableAdapters(), libcModulePath) {
		t.Fatal("modernc libc is absent from adapter regeneration")
	}
	root := t.TempDir()
	directory := filepath.Join(root, "deterministicio")
	if err := os.Mkdir(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	regeneration := AdapterRegeneration{Module: libcModulePath}
	var source strings.Builder
	source.WriteString("package deterministicio\nconst (\n")
	for index, name := range libcRegenerationSources {
		old, next := "old-"+name, "new-"+name
		regeneration.Previous.Rewrites = append(regeneration.Previous.Rewrites, RewriteAnchors{Path: name, SourceSHA256: old})
		regeneration.Proposed.Rewrites = append(regeneration.Proposed.Rewrites, RewriteAnchors{Path: name, SourceSHA256: next})
		source.WriteString("pin" + string(rune('A'+index)) + " = \"" + old + "\"\n")
	}
	regeneration.Previous.PreparedSourceSetSHA256 = map[string]string{"darwin/arm64": "old-darwin", "linux/amd64": "old-linux"}
	regeneration.Proposed.PreparedSourceSetSHA256 = map[string]string{"darwin/arm64": "new-darwin", "linux/amd64": "new-linux"}
	source.WriteString("darwin = \"old-darwin\"\nlinux = \"old-linux\"\n)\n")
	path := filepath.Join(directory, "libc_adapter.go")
	if err := os.WriteFile(path, []byte(source.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	edits, err := regeneration.SourceEdits(root)
	if err != nil {
		t.Fatal(err)
	}
	updated := string(edits["deterministicio/libc_adapter.go"])
	if strings.Contains(updated, "old-") || !strings.Contains(updated, "new-linux") {
		t.Fatalf("libc pins were not fully refreshed: %s", updated)
	}
	if err := os.WriteFile(path, []byte(strings.Replace(source.String(), "old-darwin", "drifted", 1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := regeneration.SourceEdits(root); err == nil || !strings.Contains(err.Error(), "occurs 0 times") {
		t.Fatalf("changed pinned source was accepted: %v", err)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
