// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gomadcap

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestEncodeCanonicalRecord(t *testing.T) {
	buildKey := strings.Repeat("a", 64)
	record, err := Encode(Input{
		GOARCH:            "arm64",
		GOOS:              "darwin",
		GoVersion:         "go1.26.4",
		ToolchainBuildKey: buildKey,
		Facts: []Fact{
			{Kind: FactKindCapability, OwnerPackage: "example.com/p", OwnerSymbol: "second", Capability: "import:syscall"},
			{Kind: FactKindCapability, OwnerPackage: "example.com/p", OwnerSymbol: "first", Capability: "import:syscall"},
			{Kind: FactKindCapability, OwnerPackage: "example.com/p", OwnerSymbol: "first", Capability: "import:syscall"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(record) <= HeaderBytes {
		t.Fatalf("record length = %d, want more than %d", len(record), HeaderBytes)
	}
	if !bytes.Equal(record[:16], HeaderMagic[:]) {
		t.Fatalf("magic = %q, want %q", record[:16], HeaderMagic)
	}
	if got := binary.LittleEndian.Uint32(record[16:20]); got != ProtocolVersion {
		t.Fatalf("protocol version = %d, want %d", got, ProtocolVersion)
	}
	if got := binary.LittleEndian.Uint32(record[20:24]); got != HeaderBytes {
		t.Fatalf("header bytes = %d, want %d", got, HeaderBytes)
	}
	if got := binary.LittleEndian.Uint64(record[24:32]); got != uint64(len(record)-HeaderBytes) {
		t.Fatalf("payload bytes = %d, want %d", got, len(record)-HeaderBytes)
	}
	if got := binary.LittleEndian.Uint64(record[32:40]); got != 2 {
		t.Fatalf("fact count = %d, want 2", got)
	}
	payloadDigest := sha256.Sum256(record[HeaderBytes:])
	if !bytes.Equal(record[40:72], payloadDigest[:]) {
		t.Fatal("payload digest does not match")
	}
	producerDigest, err := parseSHA256(ProducerImplementationSHA256)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(record[72:104], producerDigest[:]) {
		t.Fatal("producer digest does not match")
	}
	if !bytes.Equal(record[104:HeaderBytes], make([]byte, HeaderBytes-104)) {
		t.Fatal("reserved header bytes are nonzero")
	}

	var manifest Manifest
	if err := json.Unmarshal(record[HeaderBytes:], &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.ToolchainBuildKey != buildKey || manifest.GOOS != "darwin" || manifest.GOARCH != "arm64" {
		t.Fatalf("manifest identity = %#v", manifest)
	}
	if len(manifest.Facts) != 2 || manifest.Facts[0].OwnerSymbol != "first" || manifest.Facts[1].OwnerSymbol != "second" {
		t.Fatalf("facts = %#v, want sorted unique facts", manifest.Facts)
	}
}

func TestEncodeRejectsConflictingFacts(t *testing.T) {
	fact := Fact{Kind: FactKindBoundary, OwnerPackage: "os", OwnerSymbol: "OpenFile", Capability: "filesystem.open", Disposition: DispositionModeled}
	conflict := fact
	conflict.Disposition = DispositionDenied
	_, err := Encode(Input{
		GOARCH:            "arm64",
		GOOS:              "darwin",
		GoVersion:         "go1.26.4",
		ToolchainBuildKey: strings.Repeat("b", 64),
		Facts:             []Fact{fact, conflict},
	})
	if err == nil || !strings.Contains(err.Error(), "conflicting") {
		t.Fatalf("error = %v, want conflicting facts", err)
	}
}

func TestEncodeRejectsOwnerFactOverflow(t *testing.T) {
	facts := make([]Fact, MaximumOwnerFacts+1)
	for index := range facts {
		facts[index] = Fact{
			Kind:         FactKindCapability,
			OwnerPackage: "example.com/p",
			OwnerSymbol:  "owner",
			Capability:   fmt.Sprintf("import:example.com/forbidden/%05d", index),
		}
	}
	_, err := Encode(Input{
		GOARCH:            "arm64",
		GOOS:              "darwin",
		GoVersion:         "go1.26.4",
		ToolchainBuildKey: strings.Repeat("c", 64),
		Facts:             facts,
	})
	if err == nil || !strings.Contains(err.Error(), "owner facts") {
		t.Fatalf("error = %v, want owner facts capacity error", err)
	}
}

func TestRelocationFactsProjectImportsAndBoundaries(t *testing.T) {
	facts := RelocationFacts("example.com/p", "example.com/p.Live", "syscall", "syscall.Syscall6", false)
	if len(facts) != 1 || facts[0].Kind != FactKindCapability || facts[0].Capability != "import:syscall" {
		t.Fatalf("syscall relocation facts = %#v", facts)
	}
	facts = RelocationFacts("example.com/p", "example.com/p.Live", "syscall", "syscall.Syscall6", true)
	if len(facts) != 1 || facts[0].Kind != FactKindGuard || facts[0].Capability != "import:syscall" || facts[0].Disposition != DispositionGuarded || facts[0].ReferencedSymbol != "syscall.Syscall6" {
		t.Fatalf("guarded syscall relocation facts = %#v", facts)
	}
	facts = RelocationFacts("example.com/p", "example.com/p.Read", "os", "os.(*File).Read", false)
	if len(facts) != 1 || facts[0].Kind != FactKindBoundary || facts[0].Capability != "filesystem.read" || facts[0].Disposition != DispositionModeled {
		t.Fatalf("boundary relocation facts = %#v", facts)
	}
	if facts := RelocationFacts("example.com/p", "example.com/p.Safe", "fmt", "fmt.Println", false); len(facts) != 0 {
		t.Fatalf("safe relocation facts = %#v", facts)
	}
	if facts := RelocationFacts("bytes", "bytes.Title.stkobj", "os/exec", "runtime.gcbits.0200000000000000", false); len(facts) != 0 {
		t.Fatalf("shared symbol relocation facts = %#v", facts)
	}
	if facts := RelocationFacts("bytes", "bytes.Title.stkobj", "os/exec", "os/exec..stmp_13", false); len(facts) != 0 {
		t.Fatalf("deduplicated static relocation facts = %#v", facts)
	}
	if facts := RelocationFacts("syscall", "syscall.Open", "syscall", "syscall.open", false); len(facts) != 0 {
		t.Fatalf("same-package relocation facts = %#v", facts)
	}
}
