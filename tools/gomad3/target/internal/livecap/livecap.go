package livecap

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomad3/record"
)

type Expectation struct {
	GoVersion         string
	ToolchainBuildKey string
	GOOS              string
	GOARCH            string
}

type Record struct {
	Manifest Manifest
	Payload  []byte
	SHA256   record.SHA256
}

type CapacityError struct {
	Resource string
	Required uint64
	Maximum  uint64
}

func (err *CapacityError) Error() string {
	return fmt.Sprintf("live capability %s requires %d, maximum is %d", err.Resource, err.Required, err.Maximum)
}

func Decode(data []byte, expected Expectation) (Record, error) {
	if len(data) < HeaderBytes {
		return Record{}, errors.New("live capability record is truncated")
	}
	if !bytes.Equal(data[:16], HeaderMagic[:]) {
		return Record{}, errors.New("live capability record magic is invalid")
	}
	if binary.LittleEndian.Uint32(data[16:20]) != ProtocolVersion || binary.LittleEndian.Uint32(data[20:24]) != HeaderBytes {
		return Record{}, errors.New("live capability record version or header length is invalid")
	}
	payloadBytes := binary.LittleEndian.Uint64(data[24:32])
	factCount := binary.LittleEndian.Uint64(data[32:40])
	if payloadBytes > MaximumPayloadBytes {
		return Record{}, &CapacityError{Resource: "payload bytes", Required: payloadBytes, Maximum: MaximumPayloadBytes}
	}
	if factCount > MaximumFacts {
		return Record{}, &CapacityError{Resource: "fact count", Required: factCount, Maximum: MaximumFacts}
	}
	if payloadBytes != uint64(len(data)-HeaderBytes) {
		return Record{}, errors.New("live capability payload length does not match its record")
	}
	if !allZero(data[104:HeaderBytes]) {
		return Record{}, errors.New("live capability reserved header bytes are nonzero")
	}
	producer, err := record.ParseSHA256(ProducerImplementationSHA256)
	if err != nil {
		return Record{}, fmt.Errorf("decode live capability producer identity: %w", err)
	}
	producerDigest, err := producer.Bytes()
	if err != nil {
		return Record{}, fmt.Errorf("decode live capability producer identity: %w", err)
	}
	if !bytes.Equal(data[72:104], producerDigest[:]) {
		return Record{}, errors.New("live capability header producer identity does not match this Gomad build")
	}
	payload := data[HeaderBytes:]
	digest := sha256.Sum256(payload)
	if !bytes.Equal(data[40:72], digest[:]) {
		return Record{}, errors.New("live capability payload SHA-256 does not match its header")
	}
	var manifest Manifest
	if err := canonicaljson.DecodeCanonicalJSON(payload, &manifest); err != nil {
		return Record{}, fmt.Errorf("decode live capability manifest: %w", err)
	}
	if err := validateManifest(manifest, expected, factCount, payloadBytes); err != nil {
		return Record{}, err
	}
	return Record{Manifest: manifest, Payload: bytes.Clone(payload), SHA256: record.SHA256FromSum(digest)}, nil
}

func validateManifest(manifest Manifest, expected Expectation, factCount, payloadBytes uint64) error {
	if manifest.Schema != ManifestSchema || manifest.ProducerImplementationSHA256 != ProducerImplementationSHA256 || manifest.CapabilityUniverseSHA256 != CapabilityUniverseSHA256 {
		return errors.New("live capability manifest implementation identity is invalid")
	}
	if manifest.GuardImplementationSHA256 != GuardImplementationSHA256 {
		return errors.New("live capability guard implementation identity is invalid")
	}
	if manifest.GoVersion != expected.GoVersion || manifest.ToolchainBuildKey != expected.ToolchainBuildKey || manifest.GOOS != expected.GOOS || manifest.GOARCH != expected.GOARCH {
		return errors.New("live capability manifest target identity does not match the prepared target")
	}
	if len(expected.ToolchainBuildKey) != 64 || !lowerHex(expected.ToolchainBuildKey) {
		return errors.New("live capability expected toolchain build key is invalid")
	}
	wantLimits := Limits{Facts: MaximumFacts, OwnerFacts: MaximumOwnerFacts, PayloadBytes: MaximumPayloadBytes, StringBytes: MaximumStringBytes}
	if manifest.Limits != wantLimits {
		return errors.New("live capability manifest limits do not match the protocol")
	}
	if manifest.Facts == nil || uint64(len(manifest.Facts)) != factCount {
		return errors.New("live capability manifest fact count does not match its header")
	}
	if payloadBytes > manifest.Limits.PayloadBytes {
		return &CapacityError{Resource: "payload bytes", Required: payloadBytes, Maximum: manifest.Limits.PayloadBytes}
	}
	ownerCounts := make(map[string]uint64)
	for index, fact := range manifest.Facts {
		if index > 0 && compareFact(manifest.Facts[index-1], fact) >= 0 {
			return errors.New("live capability facts must be sorted, unique, and conflict-free")
		}
		if err := validateFact(fact); err != nil {
			return fmt.Errorf("live capability fact %d: %w", index, err)
		}
		owner := fact.OwnerPackage + "\x00" + fact.ForTest + "\x00" + fact.OwnerSymbol
		ownerCounts[owner]++
		if ownerCounts[owner] > manifest.Limits.OwnerFacts {
			return &CapacityError{Resource: "facts for owner " + fact.OwnerSymbol, Required: ownerCounts[owner], Maximum: manifest.Limits.OwnerFacts}
		}
	}
	return nil
}

func validateFact(fact Fact) error {
	if fact.OwnerPackage == "" || fact.OwnerSymbol == "" || fact.Capability == "" {
		return errors.New("owner package, owner symbol, and capability are required")
	}
	for _, value := range []string{
		fact.Capability, string(fact.Disposition), fact.ForTest, string(fact.Kind), fact.OwnerPackage,
		fact.OwnerSource, fact.OwnerSymbol, fact.ReferencedSymbol,
	} {
		if len(value) > MaximumStringBytes {
			return &CapacityError{Resource: "string bytes", Required: uint64(len(value)), Maximum: MaximumStringBytes}
		}
	}
	switch fact.Kind {
	case FactKindBoundary:
		if fact.Disposition != DispositionDenied && fact.Disposition != DispositionModeled {
			return errors.New("boundary fact disposition is invalid")
		}
	case FactKindCapability:
		if fact.Disposition != "" || !strings.HasPrefix(fact.Capability, "import:") {
			return errors.New("capability fact is invalid")
		}
	case FactKindForeign:
		if fact.Disposition != "" || fact.OwnerSource == "" || !strings.HasPrefix(fact.Capability, "foreign:") {
			return errors.New("foreign fact is invalid")
		}
	case FactKindGuard:
		if fact.Disposition != DispositionGuarded || !IsValidGuardFact(fact.Capability, fact.ReferencedSymbol) {
			return errors.New("guard fact is invalid")
		}
	case FactKindLinkname:
		if fact.Disposition != "" || fact.OwnerSource == "" || fact.ReferencedSymbol == "" {
			return errors.New("linkname fact is invalid")
		}
	default:
		return fmt.Errorf("unknown fact kind %q", fact.Kind)
	}
	return nil
}

func compareFact(left, right Fact) int {
	for _, values := range [][2]string{
		{string(left.Kind), string(right.Kind)},
		{left.OwnerPackage, right.OwnerPackage},
		{left.ForTest, right.ForTest},
		{left.OwnerSource, right.OwnerSource},
		{left.OwnerSymbol, right.OwnerSymbol},
		{left.Capability, right.Capability},
		{left.ReferencedSymbol, right.ReferencedSymbol},
	} {
		if comparison := strings.Compare(values[0], values[1]); comparison != 0 {
			return comparison
		}
	}
	return 0
}

func allZero(value []byte) bool {
	for _, item := range value {
		if item != 0 {
			return false
		}
	}
	return true
}

func lowerHex(value string) bool {
	for _, character := range value {
		if character < '0' || character > '9' && character < 'a' || character > 'f' {
			return false
		}
	}
	return true
}
