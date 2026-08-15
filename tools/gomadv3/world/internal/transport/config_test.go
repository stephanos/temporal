package transport

import (
	"bytes"
	"testing"
)

func TestConfigurationRoundTripsAndRejectsMalformedInput(t *testing.T) {
	expected := Config{TransitionLimit: 42, Seed: 7, ExpectedInitial: []byte("snapshot")}
	encoded, err := Encode(expected)
	if err != nil {
		t.Fatal(err)
	}
	config, err := Read(bytes.NewReader(encoded))
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
			if _, err := Read(bytes.NewReader(malformed)); err == nil {
				t.Fatal("Read() succeeded")
			}
		})
	}
	if _, err := Encode(Config{}); err == nil {
		t.Fatal("Encode() accepted zero")
	}
}
