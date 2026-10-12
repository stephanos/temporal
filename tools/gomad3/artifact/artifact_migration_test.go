package artifact

import (
	"os"
	"testing"

	"go.temporal.io/server/tools/gomad3/record"
)

func TestOpenPreMoveMinimizedArtifactUsesRecordedChoiceIdentity(t *testing.T) {
	path := os.Getenv("GOMAD_PRE_MOVE_MINIMIZED_ARTIFACT")
	if path == "" {
		t.Skip("requires the retained pre-move minimized artifact")
	}
	opened, err := OpenArtifact(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := opened.Close(); err != nil {
			t.Error(err)
		}
	})
	manifest := opened.Manifest()
	if manifest.RecordHash != record.SHA256("sha256:5e999d872d7a6c6e612c1df16546404200160cc761807d3d77e0b4e8d5a327bf") {
		t.Fatalf("historical record identity changed: %s", manifest.RecordHash)
	}
	if manifest.ChoiceProfile == nil || manifest.ChoiceProfile.ImplementationSHA256 != record.SHA256("sha256:2ec7b626144ac127543110e6bfe075df74d0fc09a606790df7e20a2c295075ad") {
		t.Fatalf("historical choice identity changed: %+v", manifest.ChoiceProfile)
	}
}
