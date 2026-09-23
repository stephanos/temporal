// Package replay turns one admitted violated Run of a produced Case into separate answers: whether
// its Contract-relative violation recurs, which prefix steps one sweep can drop, and what expected
// behavior to propose. This file is the recorded Run, the file shape every writer of a closed Run
// shares, which lives in tools/umpire/recordedrun so that qualification admission and the canary
// read it without importing the replay bridge; replay keeps its names.
package replay

import (
	"errors"
	"fmt"
	"os"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/tools/umpire/recordedrun"
)

// RecordedIdentity is the Profile identity a Run was prepared under.
type RecordedIdentity = recordedrun.Identity

// RecordedRun is one closed Run with the identity of the canonical Case it ran and the Profile
// identity it was prepared under.
type RecordedRun = recordedrun.Record

// DecodedRun is a recorded Run read back.
type DecodedRun = recordedrun.Decoded

// ErrRecordNamesNoCase says a recorded Run names no Case identity: a record from before the
// identity was recorded.
var ErrRecordNamesNoCase = recordedrun.ErrNoCase

// CaseIdentity is the hex SHA-256 of a Case's canonical bytes, recovered from its canonical or
// persisted form: what a recorded Run names the Case it ran by.
func CaseIdentity(input []byte) (string, error) {
	return recordedrun.CaseIdentity(input)
}

// EncodeRecordedRun renders a closed Run with the identity of the Case it ran and the Profile
// identity it was prepared under, one JSON document ended with a newline.
func EncodeRecordedRun(caseIdentity string, identity testpilot.DriverIdentity, run *testpilotspb.Run) ([]byte, error) {
	return recordedrun.Encode(caseIdentity, identity, run)
}

// WriteRecordedRun writes the recorded Run of a Case, given in its canonical or persisted form, to
// path, creating it exclusively: an existing file is never replaced.
func WriteRecordedRun(path string, caseBytes []byte, identity testpilot.DriverIdentity, run *testpilotspb.Run) error {
	caseIdentity, err := CaseIdentity(caseBytes)
	if err != nil {
		return fmt.Errorf("write recorded Run: the Case has no identity: %w", err)
	}
	document, err := EncodeRecordedRun(caseIdentity, identity, run)
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

// DecodeRecordedRun reads a recorded Run strictly: exactly spelled keys, each once, no unknown
// field, one document. A record naming no Case is ErrRecordNamesNoCase.
func DecodeRecordedRun(document []byte) (DecodedRun, error) {
	return recordedrun.Decode(document)
}
