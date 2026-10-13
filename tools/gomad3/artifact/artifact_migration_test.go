package artifact

import (
	"errors"
	"os"
	"testing"
)

// The retained pre-move minimized artifact predates storage lineage. It is a
// negative control: the current build refuses it with the named unsupported
// format instead of decoding or migrating it.
func TestOpenPreMoveMinimizedArtifactIsRefusedAsUnsupportedStorageFormat(t *testing.T) {
	path := os.Getenv("GOMAD_PRE_MOVE_MINIMIZED_ARTIFACT")
	if path == "" {
		t.Skip("requires the retained pre-move minimized artifact")
	}
	opened, err := OpenArtifact(path)
	var unsupported *UnsupportedFormatError
	if opened != nil || !errors.Is(err, ErrUnsupportedFormat) || !errors.As(err, &unsupported) || unsupported.Format != PreLineageFormat {
		t.Fatalf("OpenArtifact() = %v, %v; want the pre-lineage refusal", opened, err)
	}
}
