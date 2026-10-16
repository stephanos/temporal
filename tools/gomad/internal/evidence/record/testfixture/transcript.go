//go:build !gomad_native

// Package testfixture supplies backend-neutral evidence for host tests.
package testfixture

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"runtime"
	"slices"
	"sort"

	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	iowire "go.temporal.io/server/tools/gomad/internal/model/iowire"
)

const Deterministic = "gomad-deterministic/v1"

type Spec struct{}
type Digest = record.SHA256
type Contract struct{ GoVersion, GOOS, GOARCH string }

func Default() Spec                          { return Spec{} }
func (Spec) Name() string                    { return Deterministic }
func (Spec) Inventory() []byte               { return []byte(`{"schema":"fixture.inventory/v1"}`) }
func (profile Spec) InventorySHA256() Digest { return record.HashBytes(profile.Inventory()) }
func (Spec) ImplementationSHA256() Digest    { return record.HashBytes([]byte("fixture implementation")) }
func (Spec) TargetContract() Contract {
	return Contract{runtime.Version(), runtime.GOOS, runtime.GOARCH}
}
func BoundaryManifestIdentity() (string, Digest) {
	return "fixture/v1", record.HashBytes([]byte("fixture boundary"))
}
func SemanticInstrumentationIdentity() Digest { return record.HashBytes([]byte("fixture probes")) }
func RequiredSemanticProbes() []string {
	return []string{"stdlib.net.dialtcp", "stdlib.os.file.close", "stdlib.os.file.read", "stdlib.os.openfile"}
}

type Operation struct {
	Ordinal                   uint64
	Name                      string
	ArgumentHash, ContentHash [sha256.Size]byte
	Count                     uint64
	Result                    uint32
	EntropyStart, EntropyEnd  uint64
}

func HashArgument(encoded []byte) [sha256.Size]byte { return iowire.Hash(encoded) }

func EncodeTranscript(operations []Operation) ([]byte, error) {
	var output []byte
	for index, operation := range operations {
		if operation.Ordinal != uint64(index) {
			return nil, fmt.Errorf("I/O transcript ordinal %d at record %d", operation.Ordinal, index)
		}
		encoded, err := iowire.EncodeTranscriptRecord(iowire.TranscriptRecord{Ordinal: operation.Ordinal, Operation: operation.Name, ArgumentHash: operation.ArgumentHash, ContentHash: operation.ContentHash, Count: operation.Count, Result: operation.Result, EntropyStart: operation.EntropyStart, EntropyEnd: operation.EntropyEnd})
		if err != nil {
			return nil, err
		}
		output = append(output, encoded[:]...)
	}
	return output, nil
}

func TranscriptRecordCount(encoded []byte) (uint64, error) {
	if len(encoded)%iowire.TranscriptRecordBytes != 0 {
		return 0, fmt.Errorf("I/O transcript has invalid length %d", len(encoded))
	}
	return uint64(len(encoded) / iowire.TranscriptRecordBytes), nil
}

func DecodeTranscript(encoded []byte) ([]Operation, error) {
	count, err := TranscriptRecordCount(encoded)
	if err != nil {
		return nil, err
	}
	operations := make([]Operation, 0, count)
	for offset := 0; offset < len(encoded); offset += iowire.TranscriptRecordBytes {
		decoded, err := iowire.DecodeTranscriptRecord(encoded[offset : offset+iowire.TranscriptRecordBytes])
		if err != nil {
			return nil, fmt.Errorf("decode I/O transcript record %d: %w", offset/iowire.TranscriptRecordBytes, err)
		}
		if decoded.Ordinal != uint64(len(operations)) {
			return nil, fmt.Errorf("I/O transcript ordinal %d at record %d", decoded.Ordinal, len(operations))
		}
		operations = append(operations, Operation{decoded.Ordinal, decoded.Operation, decoded.ArgumentHash, decoded.ContentHash, decoded.Count, decoded.Result, decoded.EntropyStart, decoded.EntropyEnd})
	}
	return operations, nil
}

func SummarizeSemanticProbes(probes []string) (record.SemanticCoverage, error) {
	return record.NewSemanticCoverage(probes), nil
}

func DecodeSemanticCoverage(encoded []byte) (record.SemanticCoverage, error) {
	operations, err := DecodeTranscript(encoded)
	if err != nil {
		return record.SemanticCoverage{}, err
	}
	names := map[[sha256.Size]byte]string{}
	for _, name := range RequiredSemanticProbes() {
		digest := sha256.Sum256([]byte("gomad-boundary-probe/v1\x00" + name))
		var argument [8]byte
		binary.BigEndian.PutUint64(argument[:], binary.BigEndian.Uint64(digest[:8])&((1<<63)-1))
		names[HashArgument(argument[:])] = name
	}
	var probes []string
	for _, operation := range operations {
		if operation.Name != "boundary.probe" {
			continue
		}
		name, found := names[operation.ArgumentHash]
		if !found {
			return record.SemanticCoverage{}, fmt.Errorf("I/O transcript contains unknown boundary probe %x", operation.ArgumentHash)
		}
		if slices.Contains(probes, name) {
			return record.SemanticCoverage{}, fmt.Errorf("I/O transcript contains duplicate boundary probe %s", name)
		}
		probes = append(probes, name)
	}
	return record.NewSemanticCoverage(probes), nil
}

func MissingRequiredSemanticProbes(coverage record.SemanticCoverage, required []string) ([]string, error) {
	seen := map[string]bool{}
	missing := []string{}
	for _, probe := range required {
		if !slices.Contains(RequiredSemanticProbes(), probe) {
			return nil, fmt.Errorf("unknown required semantic probe %q", probe)
		}
		if seen[probe] {
			return nil, fmt.Errorf("required semantic probe is duplicated: %s", probe)
		}
		seen[probe] = true
		if !slices.Contains(coverage.Probes, probe) {
			missing = append(missing, probe)
		}
	}
	sort.Strings(missing)
	return missing, nil
}
