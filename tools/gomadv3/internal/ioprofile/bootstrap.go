package ioprofile

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"go.temporal.io/server/tools/gomadv3/internal/iowire"
	"go.temporal.io/server/tools/gomadv3/internal/record"
	"go.temporal.io/server/tools/gomadv3/internal/target"
)

const bootstrapFrameBytes = iowire.BootstrapFrameBytes

type Bootstrap struct {
	Profile              string
	InventorySHA256      record.SHA256
	ImplementationSHA256 record.SHA256
	TargetSHA256         string
	RunnerSHA256         string
	ArgvSHA256           record.SHA256
	Seed                 uint64
}

func (profile ProfileSpec) BootstrapFrame(prepared target.Prepared, runnerSHA256 string, seed uint64) ([]byte, error) {
	definition, err := profile.validated()
	if err != nil {
		return nil, err
	}
	argv, err := record.CanonicalJSON(prepared.Argv)
	if err != nil {
		return nil, fmt.Errorf("encode target argv identity: %w", err)
	}
	digests := []string{string(definition.inventorySHA256), string(definition.implementationSHA256), prepared.SHA256, runnerSHA256, string(record.HashBytes(argv))}
	wire := iowire.Bootstrap{Seed: seed}
	destinations := []*[sha256.Size]byte{&wire.InventoryHash, &wire.ImplementationHash, &wire.TargetHash, &wire.RunnerHash, &wire.ArgvHash}
	for index, value := range digests {
		decoded, decodeErr := parseSHA256(value)
		if decodeErr != nil {
			return nil, decodeErr
		}
		copy(destinations[index][:], decoded)
	}
	frame := iowire.EncodeBootstrap(wire)
	return frame[:], nil
}

func DecodeBootstrapFrame(frame []byte) (Bootstrap, error) {
	decoded, err := iowire.DecodeBootstrap(frame)
	if err != nil {
		return Bootstrap{}, fmt.Errorf("decode I/O profile bootstrap frame: %w", err)
	}
	digests := make([]string, 0, 5)
	for _, digest := range [][sha256.Size]byte{decoded.InventoryHash, decoded.ImplementationHash, decoded.TargetHash, decoded.RunnerHash, decoded.ArgvHash} {
		digests = append(digests, "sha256:"+hex.EncodeToString(digest[:]))
	}
	profile := Default()
	if profile.InventorySHA256() != record.SHA256(digests[0]) || profile.ImplementationSHA256() != record.SHA256(digests[1]) {
		return Bootstrap{}, errors.New("I/O profile bootstrap frame identity mismatch")
	}
	return Bootstrap{
		Profile: profile.Name(), InventorySHA256: record.SHA256(digests[0]), ImplementationSHA256: record.SHA256(digests[1]),
		TargetSHA256: digests[2], RunnerSHA256: digests[3], ArgvSHA256: record.SHA256(digests[4]), Seed: decoded.Seed,
	}, nil
}

func parseSHA256(value string) ([]byte, error) {
	decoded, err := record.SHA256(value).Bytes()
	if err != nil {
		return nil, fmt.Errorf("invalid SHA-256 identity %q: %w", value, err)
	}
	return decoded[:], nil
}
