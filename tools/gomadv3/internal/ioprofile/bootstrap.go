package ioprofile

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"go.temporal.io/server/tools/gomadv3/internal/record"
	"go.temporal.io/server/tools/gomadv3/internal/target"
)

const bootstrapFrameBytes = 212

var bootstrapMagic = [8]byte{'G', 'O', 'M', 'A', 'D', 'I', 'O', 1}

type Bootstrap struct {
	Profile              string
	InventorySHA256      record.SHA256
	ImplementationSHA256 record.SHA256
	TargetSHA256         string
	RunnerSHA256         string
	ArgvSHA256           record.SHA256
	Seed                 uint64
}

func (profile Profile) BootstrapFrame(prepared target.Prepared, runnerSHA256 string, seed uint64) ([]byte, error) {
	if profile.Name != TemporalActivityAPIBatchCancel {
		return nil, fmt.Errorf("unknown I/O profile %q", profile.Name)
	}
	resolved, err := Resolve(profile.Name)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(profile.Inventory, resolved.Inventory) || profile.InventorySHA256 != resolved.InventorySHA256 || profile.ImplementationSHA256 != resolved.ImplementationSHA256 {
		return nil, fmt.Errorf("I/O profile %q identity is invalid", profile.Name)
	}
	argv, err := record.CanonicalJSON(prepared.Argv)
	if err != nil {
		return nil, fmt.Errorf("encode target argv identity: %w", err)
	}
	digests := []string{string(profile.InventorySHA256), string(profile.ImplementationSHA256), prepared.SHA256, runnerSHA256, string(record.HashBytes(argv))}
	frame := make([]byte, bootstrapFrameBytes)
	copy(frame[:8], bootstrapMagic[:])
	binary.BigEndian.PutUint16(frame[8:10], 1)
	binary.BigEndian.PutUint16(frame[10:12], 1)
	offset := 12
	for _, value := range digests {
		decoded, decodeErr := parseSHA256(value)
		if decodeErr != nil {
			return nil, decodeErr
		}
		copy(frame[offset:offset+sha256.Size], decoded)
		offset += sha256.Size
	}
	binary.BigEndian.PutUint64(frame[offset:offset+8], seed)
	offset += 8
	checksum := sha256.Sum256(frame[:offset])
	copy(frame[offset:], checksum[:])
	return frame, nil
}

func DecodeBootstrapFrame(frame []byte) (Bootstrap, error) {
	if len(frame) != bootstrapFrameBytes || !bytes.Equal(frame[:8], bootstrapMagic[:]) || binary.BigEndian.Uint16(frame[8:10]) != 1 || binary.BigEndian.Uint16(frame[10:12]) != 1 {
		return Bootstrap{}, errors.New("invalid I/O profile bootstrap frame")
	}
	checksum := sha256.Sum256(frame[:bootstrapFrameBytes-sha256.Size])
	if !bytes.Equal(checksum[:], frame[bootstrapFrameBytes-sha256.Size:]) {
		return Bootstrap{}, errors.New("I/O profile bootstrap frame checksum mismatch")
	}
	digests := make([]string, 5)
	offset := 12
	for index := range digests {
		digests[index] = "sha256:" + hex.EncodeToString(frame[offset:offset+sha256.Size])
		offset += sha256.Size
	}
	seed := binary.BigEndian.Uint64(frame[offset : offset+8])
	profile, err := Resolve(TemporalActivityAPIBatchCancel)
	if err != nil {
		return Bootstrap{}, err
	}
	if record.SHA256(digests[0]) != profile.InventorySHA256 || record.SHA256(digests[1]) != profile.ImplementationSHA256 {
		return Bootstrap{}, errors.New("I/O profile bootstrap frame identity mismatch")
	}
	return Bootstrap{
		Profile: profile.Name, InventorySHA256: record.SHA256(digests[0]), ImplementationSHA256: record.SHA256(digests[1]),
		TargetSHA256: digests[2], RunnerSHA256: digests[3], ArgvSHA256: record.SHA256(digests[4]), Seed: seed,
	}, nil
}

func parseSHA256(value string) ([]byte, error) {
	hexValue, found := strings.CutPrefix(value, "sha256:")
	if !found || len(hexValue) != sha256.Size*2 {
		return nil, fmt.Errorf("invalid SHA-256 identity %q", value)
	}
	decoded, err := hex.DecodeString(hexValue)
	if err != nil {
		return nil, fmt.Errorf("invalid SHA-256 identity %q: %w", value, err)
	}
	return decoded, nil
}
