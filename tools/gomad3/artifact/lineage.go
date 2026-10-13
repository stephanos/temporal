package artifact

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

// StorageFormat is the artifact storage lineage this build writes and the
// only one it opens. It names the on-disk layout of an artifact directory
// (storage.json, manifest.json and the files the manifest lists), not the
// execution record: record.SchemaVersion and record.RecordContract keep their
// own versions inside the manifest.
const StorageFormat = "gomad3.artifact-storage/v1"

// PreLineageFormat names the artifacts that earlier Gomad builds wrote
// without a storage envelope. They are refused, never decoded or migrated.
const PreLineageFormat = "pre-lineage"

// storageEnvelopeFile is the storage envelope of an artifact directory. Like
// manifest.json it is reserved: the manifest never lists it and no payload
// may use its name. It is written and synced before the manifest and read
// before the manifest is decoded.
const storageEnvelopeFile = "storage.json"

const maximumStorageEnvelopeBytes = 4 << 10

type storageEnvelope struct {
	Format string `json:"format"`
}

// storageEnvelopeBytes is the envelope publication writes.
var storageEnvelopeBytes = func() []byte {
	encoded, err := json.Marshal(storageEnvelope{Format: StorageFormat})
	if err != nil {
		panic(err)
	}
	return append(encoded, '\n')
}()

// ErrUnsupportedFormat matches every UnsupportedFormatError.
var ErrUnsupportedFormat = errors.New("unsupported artifact storage format")

// UnsupportedFormatError is the refusal of an artifact whose storage lineage
// is not StorageFormat: an artifact of an earlier Gomad without a storage
// envelope (Format is PreLineageFormat), an envelope that does not decode
// (Malformed is set), or an envelope that declares another format. The
// manifest of such an artifact is never decoded. errors.Is matches
// ErrUnsupportedFormat. The CLI reports it with exit status 2, as other
// invalid input: `gomad inspect` prints "inspect PATH: <error>" and
// `gomad replay` and `gomad minimize` print "incompatible replay artifact:
// <error>".
type UnsupportedFormatError struct {
	// Format is the storage format the artifact declares, PreLineageFormat
	// when it declares none, and empty when its envelope is malformed.
	Format string
	// Malformed is why a present envelope does not decode.
	Malformed error
}

func (err *UnsupportedFormatError) Error() string {
	switch {
	case err.Malformed != nil:
		return fmt.Sprintf("%s: malformed %s (%v); this Gomad reads only %q artifacts and does not migrate others", ErrUnsupportedFormat, storageEnvelopeFile, err.Malformed, StorageFormat)
	case err.Format == PreLineageFormat:
		return fmt.Sprintf("%s %q: the artifact has no %s; this Gomad reads only %q artifacts and does not migrate others", ErrUnsupportedFormat, PreLineageFormat, storageEnvelopeFile, StorageFormat)
	default:
		return fmt.Sprintf("%s %q: this Gomad reads only %q artifacts and does not migrate others", ErrUnsupportedFormat, err.Format, StorageFormat)
	}
}

func (err *UnsupportedFormatError) Is(target error) bool {
	return target == ErrUnsupportedFormat
}

// checkStorageLineage reads the storage envelope of the pinned artifact
// directory and accepts only StorageFormat. It runs before the manifest is
// read, so no other lineage reaches the manifest decoder.
func checkStorageLineage(root *os.Root) error {
	data, err := readValidatedFile(root, storageEnvelopeFile, 0o600, maximumStorageEnvelopeBytes)
	if errors.Is(err, os.ErrNotExist) {
		return &UnsupportedFormatError{Format: PreLineageFormat}
	}
	if err != nil {
		return fmt.Errorf("read artifact storage envelope: %w", err)
	}
	envelope, err := decodeStorageEnvelope(data)
	if err != nil {
		return &UnsupportedFormatError{Malformed: err}
	}
	if envelope.Format != StorageFormat {
		return &UnsupportedFormatError{Format: envelope.Format}
	}
	return nil
}

func decodeStorageEnvelope(data []byte) (storageEnvelope, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var envelope storageEnvelope
	if err := decoder.Decode(&envelope); err != nil {
		return storageEnvelope{}, err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return storageEnvelope{}, errors.New("trailing data after the envelope")
	}
	if envelope.Format == "" {
		return storageEnvelope{}, errors.New("no format")
	}
	return envelope, nil
}
