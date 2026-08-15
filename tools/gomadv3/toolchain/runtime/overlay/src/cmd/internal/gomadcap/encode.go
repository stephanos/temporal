// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gomadcap

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
)

type Input struct {
	Facts             []Fact
	GoVersion         string
	GOARCH            string
	GOOS              string
	ToolchainBuildKey string
}

func RelocationFacts(ownerPackage, ownerSymbol, targetPackage, targetSymbol string) []Fact {
	facts := []Fact{}
	if ownerPackage != targetPackage && IsForbiddenImport(targetPackage) && symbolBelongsToPackage(targetPackage, targetSymbol) {
		facts = append(facts, Fact{
			Capability:       "import:" + targetPackage,
			Kind:             FactKindCapability,
			OwnerPackage:     ownerPackage,
			OwnerSymbol:      ownerSymbol,
			ReferencedSymbol: targetSymbol,
		})
	}
	if boundary, ok := LookupBoundarySymbol(targetPackage, targetSymbol); ok {
		facts = append(facts, Fact{
			Capability:       boundary.Operation,
			Disposition:      boundary.Disposition,
			Kind:             FactKindBoundary,
			OwnerPackage:     ownerPackage,
			OwnerSymbol:      ownerSymbol,
			ReferencedSymbol: targetSymbol,
		})
	}
	return facts
}

func symbolBelongsToPackage(packagePath, symbol string) bool {
	prefix := packagePath + "."
	return packagePath != "" && len(symbol) > len(prefix) && symbol[:len(prefix)] == prefix && !strings.HasPrefix(symbol, packagePath+"..stmp_")
}

func Encode(input Input) ([]byte, error) {
	if len(input.ToolchainBuildKey) != 64 || !lowerHex(input.ToolchainBuildKey) {
		return nil, errors.New("live capability toolchain build key must be 64 lowercase hexadecimal characters")
	}
	if input.GoVersion == "" || input.GOOS == "" || input.GOARCH == "" {
		return nil, errors.New("live capability target identity is incomplete")
	}
	facts, err := normalizeFacts(input.Facts)
	if err != nil {
		return nil, err
	}
	manifest := Manifest{
		CapabilityUniverseSHA256:     CapabilityUniverseSHA256,
		Facts:                        facts,
		GoVersion:                    input.GoVersion,
		GOARCH:                       input.GOARCH,
		GOOS:                         input.GOOS,
		Limits:                       Limits{Facts: MaximumFacts, OwnerFacts: MaximumOwnerFacts, PayloadBytes: MaximumPayloadBytes, StringBytes: MaximumStringBytes},
		ProducerImplementationSHA256: ProducerImplementationSHA256,
		Schema:                       ManifestSchema,
		ToolchainBuildKey:            input.ToolchainBuildKey,
	}
	var payload bytes.Buffer
	encoder := json.NewEncoder(&payload)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(manifest); err != nil {
		return nil, fmt.Errorf("encode live capability manifest: %w", err)
	}
	payloadBytes := bytes.TrimSuffix(payload.Bytes(), []byte{'\n'})
	if len(payloadBytes) > MaximumPayloadBytes {
		return nil, fmt.Errorf("live capability payload bytes requires %d, maximum is %d", len(payloadBytes), MaximumPayloadBytes)
	}
	producerDigest, err := parseSHA256(ProducerImplementationSHA256)
	if err != nil {
		return nil, fmt.Errorf("decode live capability producer identity: %w", err)
	}
	record := make([]byte, HeaderBytes+len(payloadBytes))
	copy(record[:16], HeaderMagic[:])
	binary.LittleEndian.PutUint32(record[16:20], ProtocolVersion)
	binary.LittleEndian.PutUint32(record[20:24], HeaderBytes)
	binary.LittleEndian.PutUint64(record[24:32], uint64(len(payloadBytes)))
	binary.LittleEndian.PutUint64(record[32:40], uint64(len(facts)))
	payloadDigest := sha256.Sum256(payloadBytes)
	copy(record[40:72], payloadDigest[:])
	copy(record[72:104], producerDigest[:])
	copy(record[HeaderBytes:], payloadBytes)
	return record, nil
}

func normalizeFacts(input []Fact) ([]Fact, error) {
	if len(input) > MaximumFacts {
		return nil, fmt.Errorf("live capability facts requires %d, maximum is %d", len(input), MaximumFacts)
	}
	facts := slices.Clone(input)
	slices.SortFunc(facts, compareFact)
	result := make([]Fact, 0, len(facts))
	ownerCounts := make(map[string]uint64)
	for _, fact := range facts {
		if err := validateFact(fact); err != nil {
			return nil, err
		}
		if len(result) != 0 && compareFact(result[len(result)-1], fact) == 0 {
			if result[len(result)-1] != fact {
				return nil, fmt.Errorf("conflicting live capability facts for %s.%s", fact.OwnerPackage, fact.OwnerSymbol)
			}
			continue
		}
		owner := fact.OwnerPackage + "\x00" + fact.ForTest + "\x00" + fact.OwnerSymbol
		ownerCounts[owner]++
		if ownerCounts[owner] > MaximumOwnerFacts {
			return nil, fmt.Errorf("live capability owner facts for %s requires %d, maximum is %d", fact.OwnerSymbol, ownerCounts[owner], MaximumOwnerFacts)
		}
		result = append(result, fact)
	}
	return result, nil
}

func validateFact(fact Fact) error {
	if fact.OwnerPackage == "" || fact.OwnerSymbol == "" || fact.Capability == "" {
		return errors.New("live capability owner package, owner symbol, and capability are required")
	}
	for _, value := range []string{
		fact.Capability, string(fact.Disposition), fact.ForTest, string(fact.Kind), fact.OwnerPackage,
		fact.OwnerSource, fact.OwnerSymbol, fact.ReferencedSymbol,
	} {
		if !utf8.ValidString(value) {
			return errors.New("live capability strings must be valid UTF-8")
		}
		if len(value) > MaximumStringBytes {
			return fmt.Errorf("live capability string bytes requires %d, maximum is %d", len(value), MaximumStringBytes)
		}
	}
	switch fact.Kind {
	case FactKindBoundary:
		if fact.Disposition != DispositionDenied && fact.Disposition != DispositionModeled {
			return errors.New("live capability boundary disposition is invalid")
		}
	case FactKindCapability:
		if fact.Disposition != "" || !strings.HasPrefix(fact.Capability, "import:") {
			return errors.New("live capability import fact is invalid")
		}
	case FactKindForeign:
		if fact.Disposition != "" || fact.OwnerSource == "" || !strings.HasPrefix(fact.Capability, "foreign:") {
			return errors.New("live capability foreign fact is invalid")
		}
	case FactKindLinkname:
		if fact.Disposition != "" || fact.OwnerSource == "" || fact.ReferencedSymbol == "" {
			return errors.New("live capability linkname fact is invalid")
		}
	default:
		return fmt.Errorf("unknown live capability fact kind %q", fact.Kind)
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

func parseSHA256(value string) ([sha256.Size]byte, error) {
	var result [sha256.Size]byte
	const prefix = "sha256:"
	if !strings.HasPrefix(value, prefix) {
		return result, errors.New("SHA-256 identity prefix is invalid")
	}
	decoded, err := hex.DecodeString(value[len(prefix):])
	if err != nil || len(decoded) != sha256.Size {
		return result, errors.New("SHA-256 identity is invalid")
	}
	copy(result[:], decoded)
	return result, nil
}

func lowerHex(value string) bool {
	for _, character := range value {
		if character < '0' || character > '9' && character < 'a' || character > 'f' {
			return false
		}
	}
	return true
}
