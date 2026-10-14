package clock

import (
	"bytes"
	"testing"
	"testing/fstest"
)

func TestBrokerImplementationBindsEverySharedClockSource(t *testing.T) {
	sources := fstest.MapFS{}
	for _, name := range []string{"simulation_broker.go", "simulation_progress.go", "simulation_time.go"} {
		source, err := brokerImplementationSources.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		sources[name] = &fstest.MapFile{Data: source}
	}
	baseline := BrokerImplementationSHA256()
	if got := brokerImplementationSHA256(sources); got == "" || got != baseline {
		t.Fatalf("complete source identity = %q, want %q", got, baseline)
	}
	for _, name := range []string{"simulation_broker.go", "simulation_progress.go", "simulation_time.go"} {
		for _, mutation := range []string{"missing", "tampered"} {
			t.Run(name+"/"+mutation, func(t *testing.T) {
				changed := fstest.MapFS{}
				for path, file := range sources {
					changed[path] = &fstest.MapFile{Data: bytes.Clone(file.Data)}
				}
				if mutation == "missing" {
					delete(changed, name)
				} else {
					changed[name].Data[0] ^= 1
				}
				got := brokerImplementationSHA256(changed)
				if mutation == "missing" && got != "" || mutation == "tampered" && (got == "" || got == baseline) {
					t.Fatalf("%s source identity = %q; original = %q", mutation, got, baseline)
				}
			})
		}
	}
}
