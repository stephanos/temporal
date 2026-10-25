package snapshot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadPreservesPayloadAndRefusesPendingOrMalformed(t *testing.T) {
	body := "\nfirst\n---\n\nlast\n\n"
	path := filepath.Join(t.TempDir(), "control.txt.snap")
	for _, candidate := range []struct {
		content string
		valid   bool
	}{
		{"---\nsource: control.rs\n---\n" + body, true},
		{"plain text", false},
		{"---\nsource: control.rs", false},
		{"---\nwrong: field\n---\nbody", false},
		{"---\nsource: control.rs\nsnapshot_kind: binary\n---\nbody", false},
	} {
		if err := os.WriteFile(path, []byte(candidate.content), 0o600); err != nil {
			t.Fatal(err)
		}
		got, err := Read(path)
		if candidate.valid && (err != nil || got != body) {
			t.Fatalf("body changed: %q, %v", got, err)
		}
		if !candidate.valid && err == nil {
			t.Fatalf("accepted malformed snapshot %q", candidate.content)
		}
	}
	if err := os.WriteFile(path, []byte("---\nsource: control.rs\n---\n"+body), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path+".new", []byte("candidate"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(path); err == nil || !strings.Contains(err.Error(), "pending snapshot") {
		t.Fatalf("pending snapshot was not refused: %v", err)
	}
}
