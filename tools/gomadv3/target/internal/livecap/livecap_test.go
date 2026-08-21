package livecap

import (
	"bytes"
	"crypto/sha256"
	"debug/macho"
	"encoding/binary"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomadv3/evidence"
)

func TestLookupBoundarySymbolMatchesLinkerNames(t *testing.T) {
	for _, symbol := range []string{"os.(*File).Read", "os.(*File).Read.abi0", "os.(*File).Read-fm"} {
		boundary, ok := LookupBoundarySymbol("os", symbol)
		if !ok || boundary.Target != "(*File).Read" || boundary.Disposition != DispositionModeled {
			t.Fatalf("LookupBoundarySymbol(%q) = %#v, %t", symbol, boundary, ok)
		}
	}
	if _, ok := LookupBoundarySymbol("os", "os.(*File).ReadByte"); ok {
		t.Fatal("LookupBoundarySymbol() matched a different boundary")
	}
}

func TestIsGuardSymbolMatchesCompilerABINames(t *testing.T) {
	for _, symbol := range []string{GuardSymbol, GuardSymbol + ".abi0", GuardSymbol + ".abiinternal"} {
		if !IsGuardSymbol(symbol) {
			t.Fatalf("IsGuardSymbol(%q) = false", symbol)
		}
	}
	if IsGuardSymbol(GuardSymbol + "Changed") {
		t.Fatal("IsGuardSymbol() matched a different symbol")
	}
}

func TestExtractMachORecordRequiresOneReadOnlyInBoundsSymbol(t *testing.T) {
	payload := []byte("payload")
	record := liveCapabilityRecord(payload, 0)
	sectionBytes := append([]byte("prefix"), record...)
	section := &macho.Section{
		SectionHeader: macho.SectionHeader{Name: "__rodata", Seg: "__TEXT", Addr: 0x1000, Size: uint64(len(sectionBytes))},
		ReaderAt:      bytes.NewReader(sectionBytes),
	}
	symbols := []macho.Symbol{{Name: ReservedSymbol, Value: section.Addr + uint64(len("prefix"))}}
	got, err := extractMachORecord(symbols, []*macho.Section{section})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, record) {
		t.Fatalf("extractMachORecord() = %x, want %x", got, record)
	}

	duplicate := append(append([]macho.Symbol(nil), symbols...), symbols[0])
	if _, err := extractMachORecord(duplicate, []*macho.Section{section}); err == nil || !strings.Contains(err.Error(), "exactly one") {
		t.Fatalf("extractMachORecord(duplicate) error = %v", err)
	}
	writable := *section
	writable.Seg = "__DATA"
	if _, err := extractMachORecord(symbols, []*macho.Section{&writable}); err == nil || !strings.Contains(err.Error(), "read-only") {
		t.Fatalf("extractMachORecord(writable) error = %v", err)
	}
	truncated := *section
	truncated.Size--
	if _, err := extractMachORecord(symbols, []*macho.Section{&truncated}); err == nil || !strings.Contains(err.Error(), "bounds") {
		t.Fatalf("extractMachORecord(truncated) error = %v", err)
	}
	shortRead := *section
	shortRead.ReaderAt = bytes.NewReader(sectionBytes[:len(sectionBytes)-1])
	if _, err := extractMachORecord(symbols, []*macho.Section{&shortRead}); err == nil || !strings.Contains(err.Error(), "read live capability record") {
		t.Fatalf("extractMachORecord(short read) error = %v", err)
	}
}

func TestDecodeValidatesCanonicalManifestAndHeaderIdentity(t *testing.T) {
	expected := Expectation{
		GoVersion: "go1.26.4", ToolchainBuildKey: strings.Repeat("1", 64), GOOS: "darwin", GOARCH: "arm64",
	}
	manifest := Manifest{
		CapabilityUniverseSHA256: CapabilityUniverseSHA256,
		Facts: []Fact{{
			Capability: "import:syscall", Kind: FactKindCapability, OwnerPackage: "example.test/dependency",
			OwnerSource: "dependency.go", OwnerSymbol: "example.test/dependency.Live",
		}},
		GoVersion: expected.GoVersion, GOARCH: expected.GOARCH, GOOS: expected.GOOS,
		GuardImplementationSHA256:    GuardImplementationSHA256,
		Limits:                       Limits{Facts: MaximumFacts, OwnerFacts: MaximumOwnerFacts, PayloadBytes: MaximumPayloadBytes, StringBytes: MaximumStringBytes},
		ProducerImplementationSHA256: ProducerImplementationSHA256,
		Schema:                       ManifestSchema, ToolchainBuildKey: expected.ToolchainBuildKey,
	}
	payload, err := evidence.CanonicalJSON(manifest)
	if err != nil {
		t.Fatal(err)
	}
	record := liveCapabilityRecord(payload, uint64(len(manifest.Facts)))
	decoded, err := Decode(record, expected)
	if err != nil {
		t.Fatal(err)
	}
	if string(decoded.Payload) != string(payload) || decoded.Manifest.Facts[0] != manifest.Facts[0] || decoded.SHA256 != evidence.HashBytes(payload) {
		t.Fatalf("Decode() = %#v", decoded)
	}

	mutated := append([]byte(nil), record...)
	mutated[len(mutated)-1] ^= 1
	if _, err := Decode(mutated, expected); err == nil || !strings.Contains(err.Error(), "payload SHA-256") {
		t.Fatalf("Decode(mutated payload) error = %v", err)
	}
}

func TestDecodeRejectsUnsortedFactsAndHeaderCountMismatch(t *testing.T) {
	expected := Expectation{GoVersion: "go1.26.4", ToolchainBuildKey: strings.Repeat("1", 64), GOOS: "darwin", GOARCH: "arm64"}
	manifest := Manifest{
		CapabilityUniverseSHA256: CapabilityUniverseSHA256,
		Facts: []Fact{
			{Capability: "import:syscall", Kind: FactKindCapability, OwnerPackage: "z.test/package", OwnerSymbol: "z.test/package.Live"},
			{Capability: "import:os/exec", Kind: FactKindCapability, OwnerPackage: "a.test/package", OwnerSymbol: "a.test/package.Live"},
		},
		GoVersion: expected.GoVersion, GOARCH: expected.GOARCH, GOOS: expected.GOOS,
		GuardImplementationSHA256:    GuardImplementationSHA256,
		Limits:                       Limits{Facts: MaximumFacts, OwnerFacts: MaximumOwnerFacts, PayloadBytes: MaximumPayloadBytes, StringBytes: MaximumStringBytes},
		ProducerImplementationSHA256: ProducerImplementationSHA256,
		Schema:                       ManifestSchema, ToolchainBuildKey: expected.ToolchainBuildKey,
	}
	payload, err := evidence.CanonicalJSON(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Decode(liveCapabilityRecord(payload, uint64(len(manifest.Facts))), expected); err == nil || !strings.Contains(err.Error(), "sorted") {
		t.Fatalf("Decode(unsorted facts) error = %v", err)
	}
	manifest.Facts[0], manifest.Facts[1] = manifest.Facts[1], manifest.Facts[0]
	payload, err = evidence.CanonicalJSON(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Decode(liveCapabilityRecord(payload, 1), expected); err == nil || !strings.Contains(err.Error(), "fact count") {
		t.Fatalf("Decode(count mismatch) error = %v", err)
	}
}

func TestDecodeValidatesGuardFactsAndIdentity(t *testing.T) {
	expected := Expectation{GoVersion: "go1.26.4", ToolchainBuildKey: strings.Repeat("1", 64), GOOS: "darwin", GOARCH: "arm64"}
	manifest := Manifest{
		CapabilityUniverseSHA256: CapabilityUniverseSHA256,
		Facts: []Fact{{
			Capability: "import:os/exec", Disposition: DispositionGuarded, Kind: FactKindGuard,
			OwnerPackage: "example.test/target", OwnerSymbol: "example.test/target.main", ReferencedSymbol: "os/exec.Command",
		}},
		GoVersion: expected.GoVersion, GOARCH: expected.GOARCH, GOOS: expected.GOOS,
		GuardImplementationSHA256:    GuardImplementationSHA256,
		Limits:                       Limits{Facts: MaximumFacts, OwnerFacts: MaximumOwnerFacts, PayloadBytes: MaximumPayloadBytes, StringBytes: MaximumStringBytes},
		ProducerImplementationSHA256: ProducerImplementationSHA256,
		Schema:                       ManifestSchema, ToolchainBuildKey: expected.ToolchainBuildKey,
	}
	payload, err := evidence.CanonicalJSON(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Decode(liveCapabilityRecord(payload, 1), expected); err != nil {
		t.Fatalf("Decode(guarded manifest): %v", err)
	}

	manifest.Facts[0].Capability = "network.udp-listen"
	manifest.Facts[0].ReferencedSymbol = "net.ListenUDP"
	payload, err = evidence.CanonicalJSON(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Decode(liveCapabilityRecord(payload, 1), expected); err != nil {
		t.Fatalf("Decode(guarded denied boundary): %v", err)
	}
	manifest.Facts[0].Capability = "unknown.boundary"
	payload, err = evidence.CanonicalJSON(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Decode(liveCapabilityRecord(payload, 1), expected); err == nil || !strings.Contains(err.Error(), "guard fact") {
		t.Fatalf("Decode(unknown guarded boundary) error = %v", err)
	}
	manifest.Facts[0].Capability = "network.udp-listen"

	manifest.GuardImplementationSHA256 = "sha256:" + strings.Repeat("0", 64)
	payload, err = evidence.CanonicalJSON(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Decode(liveCapabilityRecord(payload, 1), expected); err == nil || !strings.Contains(err.Error(), "guard implementation identity") {
		t.Fatalf("Decode(guard identity drift) error = %v", err)
	}
}

func liveCapabilityRecord(payload []byte, facts uint64) []byte {
	record := make([]byte, HeaderBytes+len(payload))
	copy(record[:16], HeaderMagic[:])
	binary.LittleEndian.PutUint32(record[16:20], ProtocolVersion)
	binary.LittleEndian.PutUint32(record[20:24], HeaderBytes)
	binary.LittleEndian.PutUint64(record[24:32], uint64(len(payload)))
	binary.LittleEndian.PutUint64(record[32:40], facts)
	payloadDigest := sha256.Sum256(payload)
	copy(record[40:72], payloadDigest[:])
	producer, err := evidence.ParseSHA256(ProducerImplementationSHA256)
	if err != nil {
		panic(err)
	}
	producerDigest, err := producer.Bytes()
	if err != nil {
		panic(err)
	}
	copy(record[72:104], producerDigest[:])
	copy(record[HeaderBytes:], payload)
	return record
}
