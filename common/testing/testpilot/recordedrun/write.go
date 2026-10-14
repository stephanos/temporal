package recordedrun

import (
	"errors"
	"fmt"
	"os"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
)

// Write writes the recorded Run of a Case, given in its canonical or persisted form, to
// path, creating it exclusively: an existing file is never replaced.
func Write(path string, caseBytes []byte, identity testpilot.DriverIdentity, run *testpilotspb.Run) error {
	caseIdentity, err := CaseIdentity(caseBytes)
	if err != nil {
		return fmt.Errorf("write recorded Run: the Case has no identity: %w", err)
	}
	document, err := Encode(caseIdentity, identity, run)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("write recorded Run: %w", err)
	}
	if _, err := file.Write(document); err != nil {
		return errors.Join(fmt.Errorf("write recorded Run: %w", err), file.Close())
	}
	return file.Close()
}
