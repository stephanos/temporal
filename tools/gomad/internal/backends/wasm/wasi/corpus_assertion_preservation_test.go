package wasi

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/model/world"
)

func TestBrokerLifecycleCorpusPreservesIdleReturn(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{"crash", "stop"} {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()
			b := newBrokerLifecycleCorpus(t, "survivor-progress-"+kind, brokerTestConfig())
			for _, input := range []brokerLifecycleInput{
				{Action: kind, Session: "a"},
				{Action: "finished", Session: "a"},
				{Action: "idle", Session: "b", Deadline: 1000},
			} {
				b.step(input)
			}
			fields := corpusAssertionFields(t, b.corpus.Steps[2].Expected)
			var now uint64
			if err := json.Unmarshal(fields["idle_now"], &now); err != nil {
				t.Fatalf("idle observation must preserve its returned time: %v", err)
			}
			if now != 1000 {
				t.Fatalf("surviving timer returned %d, want 1000", now)
			}
		})
	}
}

func TestBrokerLifecycleCorpusPreservesCreationCapacityIdentity(t *testing.T) {
	t.Parallel()
	var positive, negative, missing int
	corpora := brokerLifecycleCorpora(t)
	for _, corpus := range corpora {
		fields := corpusAssertionFields(t, corpus)
		var capacity bool
		if err := json.Unmarshal(fields["creation_world_capacity"], &capacity); err != nil {
			missing++
		} else if capacity {
			positive++
		} else {
			negative++
		}
		t.Run(corpus.Name, func(t *testing.T) {
			t.Parallel()
			if err := json.Unmarshal(fields["creation_world_capacity"], &capacity); err != nil {
				t.Fatalf("creation must preserve World capacity identity: %v", err)
			}
			want := corpus.Name == "compressed-profile-oversize-refused"
			if capacity != want {
				t.Fatalf("creation World capacity = %t, want %t", capacity, want)
			}
		})
	}
	if len(corpora) != 10 || positive != 1 || negative != 9 || missing != 0 {
		t.Fatalf("creation classifications: cases=%d capacity=%d other=%d missing=%d, want 10/1/9/0", len(corpora), positive, negative, missing)
	}
}

func TestBrokerLifecycleCorpusDistinguishesSameTextCapacityErrors(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name string
		err  error
		want bool
	}{
		{name: "sentinel", err: world.ErrCapacity, want: true},
		{name: "wrapped-sentinel", err: fmt.Errorf("admission: %w", world.ErrCapacity), want: true},
		{name: "same-text-wrong-error", err: errors.New(world.ErrCapacity.Error()), want: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			b := newBrokerLifecycleCorpus(t, test.name, brokerTestConfig())
			fields := corpusAssertionFields(t, b.observe(nil, test.err))
			var capacity bool
			if err := json.Unmarshal(fields["world_capacity"], &capacity); err != nil {
				t.Fatal(err)
			}
			if capacity != test.want {
				t.Fatalf("World capacity = %t, want %t", capacity, test.want)
			}
		})
	}
}

func TestNetworkClusterCorpusPreservesExactResourceExhaustion(t *testing.T) {
	t.Parallel()
	want := map[string][]bool{
		"group-rejection-atomic-partition-scheduled-false": {true},
		"group-rejection-atomic-heal-scheduled-false":      {true},
		"group-rejection-atomic-delay-scheduled-false":     {true},
		"group-rejection-atomic-partition-scheduled-true":  {true},
		"group-rejection-atomic-heal-scheduled-true":       {true},
		"group-rejection-atomic-delay-scheduled-true":      {true},
		"group-reservation-ceiling-12":                     {false, false, false, false, false, true, false},
		"group-reservation-ceiling-13":                     {false, false, false, false, false, false, true, false},
	}
	seen := 0
	networkClusterBrokerCorpora(func(name string, config BrokerConfig, run func(*networkClusterBuilder)) {
		classifications, ok := want[name]
		if !ok {
			return
		}
		seen++
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			b := newNetworkClusterCorpus(t, name, config)
			run(b)
			if len(b.corpus.Steps) != len(classifications) {
				t.Fatalf("steps=%d, want %d", len(b.corpus.Steps), len(classifications))
			}
			for index, step := range b.corpus.Steps {
				fields := corpusAssertionFields(t, step)
				var exhausted bool
				if err := json.Unmarshal(fields["resource_exhausted"], &exhausted); err != nil {
					t.Fatalf("step %d must preserve exact Go resource sentinel identity: %v", index, err)
				}
				if exhausted != classifications[index] {
					t.Fatalf("step %d resource exhausted = %t, want %t", index, exhausted, classifications[index])
				}
			}
		})
	}, func() BrokerConfig {
		config := brokerTestConfig()
		config.Network = streamConfig()
		config.Addresses = map[string]string{}
		return config
	})
	if seen != 8 {
		t.Fatalf("resource exhaustion scenarios=%d, want 8", seen)
	}
}

func corpusAssertionFields(t *testing.T, value any) map[string]json.RawMessage {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	return fields
}
