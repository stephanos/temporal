package deterministicio

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	iowire "go.temporal.io/server/tools/gomadv3/deterministicio/internal/wire"
	"go.temporal.io/server/tools/gomadv3/target"
)

const bootstrapFrameBytes = iowire.BootstrapFrameBytes

type Bootstrap struct {
	Profile              string
	InventorySHA256      Digest
	ImplementationSHA256 Digest
	TargetSHA256         string
	RunnerSHA256         string
	ArgvSHA256           Digest
	Seed                 uint64
}

func (profile Spec) BootstrapFrame(prepared target.Prepared, runnerSHA256 string, seed uint64) ([]byte, error) {
	_, err := profile.validated()
	if err != nil {
		return nil, err
	}
	argv, err := canonicalJSON(prepared.Argv)
	if err != nil {
		return nil, fmt.Errorf("encode target argv identity: %w", err)
	}
	identity := profile.Identity()
	digests := []string{string(identity.InventorySHA256), string(identity.ImplementationSHA256), prepared.SHA256, runnerSHA256, string(hashBytes(argv))}
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
	identity := Contract{Name: profile.Name(), InventorySHA256: Digest(digests[0]), ImplementationSHA256: Digest(digests[1])}
	if !profile.Matches(identity) {
		return Bootstrap{}, errors.New("I/O profile bootstrap frame identity mismatch")
	}
	return Bootstrap{
		Profile: identity.Name, InventorySHA256: identity.InventorySHA256, ImplementationSHA256: identity.ImplementationSHA256,
		TargetSHA256: digests[2], RunnerSHA256: digests[3], ArgvSHA256: Digest(digests[4]), Seed: decoded.Seed,
	}, nil
}

func parseSHA256(value string) ([]byte, error) {
	decoded, err := Digest(value).Bytes()
	if err != nil {
		return nil, fmt.Errorf("invalid SHA-256 identity %q: %w", value, err)
	}
	return decoded[:], nil
}
