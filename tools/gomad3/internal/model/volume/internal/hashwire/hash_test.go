package hashwire

import (
	"crypto/sha256"
	"testing"
)

func TestProjectedHasherMatchesSHA256AcrossBlockBoundaries(t *testing.T) {
	for _, length := range []int{0, 1, 55, 56, 63, 64, 65, 127, 128, 257} {
		data := make([]byte, length)
		for index := range data {
			data[index] = byte(index)
		}
		hasher := NewHasher()
		for start := 0; start < len(data); start += 13 {
			hasher.Write(data[start:min(start+13, len(data))])
		}
		if got, want := hasher.Sum(), sha256.Sum256(data); got != want {
			t.Fatalf("length %d: %x, want %x", length, got, want)
		}
		if got, want := Hash(data), sha256.Sum256(data); got != want {
			t.Fatalf("codec hash length %d: %x, want %x", length, got, want)
		}
	}
}
