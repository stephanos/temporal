package artifact

import "testing"

func TestR19ArtifactDataPayloadAliasesInput(t *testing.T) {
	for _, name := range []string{"stdout", "stderr", "io/transcript", "choice/trace", "world/snapshot", "world/transitions", "world/final", "io/mounts", "simulation/plan", "simulation/record"} {
		data := []byte(name)
		payload := artifactDataPayload(name, data, 0o600)
		if len(payload.Data) != len(data) || &payload.Data[0] != &data[0] {
			t.Fatalf("%s lost payload backing", name)
		}
	}
}
