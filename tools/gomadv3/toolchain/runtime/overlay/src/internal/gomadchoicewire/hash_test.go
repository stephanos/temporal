package gomadchoicewire_test

import (
	"testing"

	"internal/gomadchoicewire"
)

func TestHasherMatchesOneShotHashAcrossChunks(t *testing.T) {
	input := []byte("streaming gomadv3 output hash across several blocks and partial blocks")
	hasher := gomadchoicewire.NewHasher()
	for len(input) != 0 {
		count := min(7, len(input))
		hasher.Write(input[:count])
		input = input[count:]
	}
	if got, want := hasher.Sum(), gomadchoicewire.Hash([]byte("streaming gomadv3 output hash across several blocks and partial blocks")); got != want {
		t.Fatalf("streaming hash = %x, want %x", got, want)
	}
	if got, want := gomadchoicewire.NewHasher().Sum(), gomadchoicewire.Hash(nil); got != want {
		t.Fatalf("empty streaming hash = %x, want %x", got, want)
	}
}
