package ioprofile

import (
	"encoding/binary"
	"slices"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomadv3/internal/iowire"
	"go.temporal.io/server/tools/gomadv3/internal/record"
)

func TestDecodeSemanticCoverageReturnsStableSortedBoundaryProbes(t *testing.T) {
	transcript := append(semanticProbeRecord(t, 0, generatedBoundaryProbes[1].ID), semanticProbeRecord(t, 1, generatedBoundaryProbes[0].ID)...)
	coverage, err := DecodeSemanticCoverage(transcript)
	if err != nil {
		t.Fatal(err)
	}
	wantProbes := []string{"stdlib.os.file.read", "stdlib.os.openfile"}
	if coverage.Schema != "gomadv3.semantic-coverage/v1" || !slices.Equal(coverage.Probes, wantProbes) || coverage.Digest != record.SHA256("sha256:877093d0a648ac4ad279f69ebcec942a4fa9d5ddc852d3e5ffab2fea79a3ca0e") {
		t.Fatalf("coverage = %#v", coverage)
	}
}

func TestDecodeSemanticCoverageRejectsUnknownAndDuplicateProbes(t *testing.T) {
	unknown := semanticProbeRecord(t, 0, ^uint64(0))
	if _, err := DecodeSemanticCoverage(unknown); err == nil || !strings.Contains(err.Error(), "unknown boundary probe") {
		t.Fatalf("unknown probe error = %v", err)
	}
	duplicate := append(semanticProbeRecord(t, 0, generatedBoundaryProbes[0].ID), semanticProbeRecord(t, 1, generatedBoundaryProbes[0].ID)...)
	if _, err := DecodeSemanticCoverage(duplicate); err == nil || !strings.Contains(err.Error(), "duplicate boundary probe") {
		t.Fatalf("duplicate probe error = %v", err)
	}
}

func TestMissingRequiredSemanticProbesValidatesNames(t *testing.T) {
	coverage := SemanticCoverage{Probes: []string{"stdlib.os.openfile"}}
	missing, err := MissingRequiredSemanticProbes(coverage, []string{"stdlib.os.file.read", "stdlib.os.openfile"})
	if err != nil || !slices.Equal(missing, []string{"stdlib.os.file.read"}) {
		t.Fatalf("MissingRequiredSemanticProbes() = %v, %v", missing, err)
	}
	if _, err := MissingRequiredSemanticProbes(coverage, []string{"unknown.probe"}); err == nil {
		t.Fatal("MissingRequiredSemanticProbes accepted an unknown probe")
	}
}

func TestSummarizeSemanticProbesCanonicalizesAUnion(t *testing.T) {
	coverage, err := SummarizeSemanticProbes([]string{"stdlib.os.openfile", "stdlib.os.file.read", "stdlib.os.openfile"})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(coverage.Probes, []string{"stdlib.os.file.read", "stdlib.os.openfile"}) || coverage.Digest != record.SHA256("sha256:877093d0a648ac4ad279f69ebcec942a4fa9d5ddc852d3e5ffab2fea79a3ca0e") {
		t.Fatalf("coverage = %#v", coverage)
	}
}

func TestSemanticInstrumentationIdentityBindsProbeNamesAndStableIDs(t *testing.T) {
	boundaryVersion, boundaryDigest := BoundaryManifestIdentity()
	instrumentation := SemanticInstrumentationIdentity()
	if boundaryVersion != "go1.26.4-darwin-arm64-v1" || boundaryDigest != record.SHA256("sha256:52d72583e74c8f355ac67b01bbab628f3fddd8c89a93cc82027beee7177948f0") {
		t.Fatalf("boundary identity = %q, %s", boundaryVersion, boundaryDigest)
	}
	if instrumentation == "" || instrumentation == boundaryDigest {
		t.Fatalf("instrumentation identity = %s", instrumentation)
	}
	if got := SemanticInstrumentationIdentity(); got != instrumentation {
		t.Fatalf("instrumentation identity changed between calls: %s, %s", instrumentation, got)
	}
}

func semanticProbeRecord(t *testing.T, ordinal, id uint64) []byte {
	t.Helper()
	var argument [8]byte
	binary.BigEndian.PutUint64(argument[:], id)
	encoded, err := iowire.EncodeTranscriptRecord(iowire.TranscriptRecord{
		Ordinal: ordinal, Operation: "boundary.probe", ArgumentHash: iowire.Hash(argument[:]),
	})
	if err != nil {
		t.Fatal(err)
	}
	return encoded[:]
}
