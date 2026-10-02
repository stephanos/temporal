//go:build unix

package artifact

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenRejectsAnotherLinkToAPayloadThatIsNotTheTarget(t *testing.T) {
	owner := t.TempDir()
	store := Store{Root: filepath.Join(owner, "failures"), TargetPool: TargetPool(owner)}
	published, err := store.PublishArtifact(artifactInput(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Link(filepath.Join(published.Path, "stdout"), filepath.Join(owner, "stdout-alias")); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenArtifact(published.Path); err == nil || !strings.Contains(err.Error(), "link count") {
		t.Fatalf("OpenArtifact() error = %v", err)
	}
}
