package hostexec

import (
	"bytes"
	"crypto/sha256"
	"testing"
)

func TestCaptureRetainsExactStreamWithinLimit(t *testing.T) {
	capture, err := New(8)
	if err != nil {
		t.Fatal(err)
	}
	input := []byte("abcdefgh")
	if written, err := capture.Write(input); err != nil || written != len(input) {
		t.Fatalf("Write() = %d, %v", written, err)
	}
	result := capture.Result()
	if !bytes.Equal(result.Bytes, input) {
		t.Fatalf("retained bytes = %q, want %q", result.Bytes, input)
	}
	if result.TotalBytes != 8 || result.RetainedBytes != 8 || result.DiscardedBytes != 0 || result.Truncated {
		t.Fatalf("stream accounting = %#v", result)
	}
	if want := sha256.Sum256(input); result.FullSHA256 != want || result.RetainedSHA256 != want {
		t.Fatal("stream hashes do not match complete input")
	}
}

func TestCaptureKeepsHeadMarkerAndTail(t *testing.T) {
	capture, err := New(8)
	if err != nil {
		t.Fatal(err)
	}
	input := []byte("abcdefghijk")
	if _, err := capture.Write(input); err != nil {
		t.Fatal(err)
	}
	result := capture.Result()
	wantBytes := []byte("abcdef\n--- gomadv3 output truncated: 3 bytes discarded ---\njk")
	if !bytes.Equal(result.Bytes, wantBytes) {
		t.Fatalf("retained bytes = %q, want %q", result.Bytes, wantBytes)
	}
	if result.TotalBytes != 11 || result.RetainedBytes != 8 || result.DiscardedBytes != 3 || !result.Truncated {
		t.Fatalf("stream accounting = %#v", result)
	}
	if want := sha256.Sum256(input); result.FullSHA256 != want {
		t.Fatal("full hash omitted discarded bytes")
	}
	if want := sha256.Sum256(wantBytes); result.RetainedSHA256 != want {
		t.Fatal("retained hash does not match artifact bytes")
	}
}

func TestCaptureIsIndependentOfWriteChunking(t *testing.T) {
	input := []byte("0123456789abcdefghijklmnopqrstuvwxyz")
	oneWrite, err := New(12)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := oneWrite.Write(input); err != nil {
		t.Fatal(err)
	}
	byteWrites, err := New(12)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range input {
		if _, err := byteWrites.Write([]byte{value}); err != nil {
			t.Fatal(err)
		}
	}
	if got, want := byteWrites.Result(), oneWrite.Result(); !bytes.Equal(got.Bytes, want.Bytes) || got.FullSHA256 != want.FullSHA256 || got.RetainedSHA256 != want.RetainedSHA256 || got.TotalBytes != want.TotalBytes || got.RetainedBytes != want.RetainedBytes || got.DiscardedBytes != want.DiscardedBytes || got.Truncated != want.Truncated {
		t.Fatalf("chunked result = %#v, want %#v", got, want)
	}
}

func TestNewRejectsZeroLimit(t *testing.T) {
	if _, err := New(0); err == nil {
		t.Fatal("New(0) succeeded")
	}
}
