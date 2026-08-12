package ioprofile

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

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

func (profile Profile) BootstrapFrame(prepared target.Prepared, runnerSHA256 string, seed uint64) ([]byte, error) {
	if profile.Name != Deterministic {
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
	profile, found := profileForIdentity(record.SHA256(digests[0]), record.SHA256(digests[1]))
	if !found {
		return Bootstrap{}, errors.New("I/O profile bootstrap frame identity mismatch")
	}
	return Bootstrap{
		Profile: profile.Name, InventorySHA256: record.SHA256(digests[0]), ImplementationSHA256: record.SHA256(digests[1]),
		TargetSHA256: digests[2], RunnerSHA256: digests[3], ArgvSHA256: record.SHA256(digests[4]), Seed: decoded.Seed,
	}, nil
}

func profileForIdentity(inventory, implementation record.SHA256) (Profile, bool) {
	for _, name := range []string{Deterministic} {
		profile, err := Resolve(name)
		if err == nil && profile.InventorySHA256 == inventory && profile.ImplementationSHA256 == implementation {
			return profile, true
		}
	}
	return Profile{}, false
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
