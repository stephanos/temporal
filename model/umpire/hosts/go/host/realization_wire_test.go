package host_test

import (
	"bytes"
	"google.golang.org/protobuf/proto"
	"testing"
	realizationv1 "umpire.local/model/gen/umpire/realization/v1"
)

func TestSharedObservationRequestWire(t *testing.T) {
	observation := &realizationv1.Observation{Method: "x"}
	if name := string(observation.ProtoReflect().Descriptor().FullName()); name != "umpire.realization.v1.Observation" {
		t.Fatal(name)
	}
	field := observation.ProtoReflect().Descriptor().Fields().ByName("request")
	if field.Number() != 4 {
		t.Fatal(field.Number())
	}
	encoded, err := proto.Marshal(observation)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(encoded, []byte{10, 1, 'x'}) {
		t.Fatalf("absent request encoding: %x", encoded)
	}
}
