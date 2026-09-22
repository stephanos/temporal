// Package replay turns one admitted violated Run of a produced Case into separate answers: whether
// its Contract-relative violation recurs, which prefix steps one sweep can drop, and what expected
// behavior to propose. This file is the recorded Run, the file shape every writer of a closed Run
// shares: the Run with its Verdict and the Profile identity it was prepared under, which the Run
// proto itself does not carry.
package replay

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"google.golang.org/protobuf/encoding/protojson"
)

// RecordedIdentity is the Profile identity a Run was prepared under: the Profile's name, the
// catalog fingerprint and the environment-binding fingerprint, none of them secret.
type RecordedIdentity struct {
	Profile  string `json:"profile"`
	Catalog  string `json:"catalog"`
	Bindings string `json:"bindings"`
}

// RecordedRun is one closed Run with its Verdict and the identity it was prepared under, as
// umpire-run --record, umpire-fuzz --record-root and the live suite write it: a local file, not an
// artifact family.
type RecordedRun struct {
	Identity RecordedIdentity `json:"identity"`
	Run      json.RawMessage  `json:"run"`
}

// EncodeRecordedRun renders a closed Run with the identity it was prepared under, one JSON
// document ended with a newline. The Run is canonical ProtoJSON, deterministic across writers.
func EncodeRecordedRun(identity testpilot.DriverIdentity, run *testpilotspb.Run) ([]byte, error) {
	if run == nil {
		return nil, errors.New("closed Run required")
	}
	encoded, err := protojson.MarshalOptions{UseProtoNames: false}.Marshal(run)
	if err != nil {
		return nil, fmt.Errorf("encode Run: %w", err)
	}
	// protojson's spacing is deliberately unstable; the record carries the compact form.
	var compact bytes.Buffer
	if err := json.Compact(&compact, encoded); err != nil {
		return nil, fmt.Errorf("compact Run: %w", err)
	}
	document, err := json.Marshal(RecordedRun{
		Identity: RecordedIdentity{Profile: identity.Profile, Catalog: identity.Catalog, Bindings: identity.Bindings},
		Run:      json.RawMessage(compact.Bytes()),
	})
	if err != nil {
		return nil, err
	}
	return append(document, '\n'), nil
}

// WriteRecordedRun writes EncodeRecordedRun's document to path, creating it exclusively: an
// existing file is never replaced.
func WriteRecordedRun(path string, identity testpilot.DriverIdentity, run *testpilotspb.Run) error {
	document, err := EncodeRecordedRun(identity, run)
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

// DecodeRecordedRun reads a recorded Run: the identity, and the Run decoded strictly (an unknown
// field is a different protocol, not a Run to admit).
func DecodeRecordedRun(document []byte) (testpilot.DriverIdentity, *testpilotspb.Run, error) {
	if len(document) == 0 {
		return testpilot.DriverIdentity{}, nil, errors.New("recorded Run is required")
	}
	decoder := json.NewDecoder(bytes.NewReader(document))
	decoder.DisallowUnknownFields()
	var recorded RecordedRun
	if err := decoder.Decode(&recorded); err != nil {
		return testpilot.DriverIdentity{}, nil, fmt.Errorf("decode recorded Run: %w", err)
	}
	// One document and nothing after it: a second document or trailing bytes are not a record.
	if err := decoder.Decode(new(json.RawMessage)); !errors.Is(err, io.EOF) {
		return testpilot.DriverIdentity{}, nil, errors.New("decode recorded Run: bytes after the document")
	}
	if len(recorded.Run) == 0 {
		return testpilot.DriverIdentity{}, nil, errors.New("recorded Run carries no Run")
	}
	run := new(testpilotspb.Run)
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(recorded.Run, run); err != nil {
		return testpilot.DriverIdentity{}, nil, fmt.Errorf("decode recorded Run: %w", err)
	}
	identity := testpilot.DriverIdentity{Profile: recorded.Identity.Profile, Catalog: recorded.Identity.Catalog, Bindings: recorded.Identity.Bindings}
	return identity, run, nil
}
