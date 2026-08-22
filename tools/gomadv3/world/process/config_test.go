package process

import (
	"bytes"
	"testing"
)

func TestConfigurationRoundTripsAndRejectsMalformedInput(t *testing.T) {
	expected := SessionSpec{TransitionLimit: 42, Seed: 7, ExpectedInitial: []byte("snapshot")}
	encoded, err := EncodeSessionSpec(expected)
	if err != nil {
		t.Fatal(err)
	}
	config, err := readSessionSpec(bytes.NewReader(encoded))
	if err != nil {
		t.Fatal(err)
	}
	if config.TransitionLimit != expected.TransitionLimit || config.Seed != expected.Seed || !bytes.Equal(config.ExpectedInitial, expected.ExpectedInitial) {
		t.Fatalf("config = %#v, want %#v", config, expected)
	}
	for name, malformed := range map[string][]byte{
		"short":      encoded[:len(encoded)-1],
		"bad-magic":  append([]byte{0}, encoded[1:]...),
		"zero-limit": append(append([]byte(nil), encoded[:8]...), make([]byte, len(encoded)-8)...),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := readSessionSpec(bytes.NewReader(malformed)); err == nil {
				t.Fatal("readSessionSpec() succeeded")
			}
		})
	}
	if _, err := EncodeSessionSpec(SessionSpec{}); err == nil {
		t.Fatal("EncodeSessionSpec() accepted zero")
	}
}
