package wasi

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVolumeFaultDecisionCorpus(t *testing.T) {
	type step struct {
		Resource    string            `json:"resource"`
		Operation   string            `json:"operation"`
		Count       uint64            `json:"count"`
		Now         uint64            `json:"now"`
		Written     uint64            `json:"written"`
		Errno       uint16            `json:"errno"`
		Reply       []byte            `json:"reply_base64"`
		Events      []byte            `json:"events_base64"`
		Occurrences map[string]uint64 `json:"occurrences"`
		Sequence    uint64            `json:"sequence"`
		Clock       int64             `json:"clock"`
	}
	type scenario struct {
		Name            string        `json:"name"`
		Storage         *StorageModel `json:"storage"`
		TranscriptLimit uint64        `json:"transcript_limit"`
		Steps           []step        `json:"steps"`
	}
	var scenarios []scenario
	for _, name := range []string{"actions-and-occurrences", "nonshortening-write", "transcript-capacity", "event-json-escaping"} {
		config := storageConfig()
		steps := []step{}
		switch name {
		case "actions-and-occurrences":
			config.Storage.Faults = []StorageFault{
				{Name: "a", Resource: "/workspace/db", Operation: "write", Occurrence: 2, Action: "torn-write", Bytes: 2},
				{Name: "b", Resource: "/workspace/db", Operation: "sync", Occurrence: 1, Action: "crash-before"},
				{Name: "c", Resource: "/workspace/db", Operation: "open", Occurrence: 1, Action: "no-space"},
			}
			steps = []step{{Resource: "/workspace/db", Operation: "write", Count: 5, Now: 7}, {Resource: "/workspace/db", Operation: "write", Count: 5, Now: 8}, {Resource: "/workspace/db", Operation: "open", Now: 9}, {Resource: "/workspace/db", Operation: "sync", Now: 9}}
		case "nonshortening-write":
			config.Storage.Faults = []StorageFault{{Name: "a", Resource: "/workspace/db", Operation: "write", Occurrence: 1, Action: "short-write", Bytes: 2}}
			steps = []step{{Resource: "/workspace/db", Operation: "write", Count: 2, Now: 1}}
		case "transcript-capacity":
			config.Limits.TranscriptBytes = 10
			steps = []step{{Resource: "/workspace/db", Operation: "open", Now: 1}}
		case "event-json-escaping":
			config.Storage.Faults = []StorageFault{{Name: "f&g", Resource: "/workspace/<x>", Operation: "write", Occurrence: 2, Action: "short-write", Bytes: 2}}
			steps = []step{{Resource: "/workspace/db", Operation: "open"}, {Resource: "/workspace/<x>", Operation: "write", Count: 4}, {Resource: "/workspace/db", Operation: "read"}, {Resource: "/workspace/<x>", Operation: "write", Count: 4}}
		}
		e := newTestEnvironment(t, config)
		for i := range steps {
			s := &steps[i]
			e.now = s.Now
			var err error
			s.Written, s.Errno, err = e.storageDecision(s.Resource, s.Operation, s.Count)
			s.Reply = volumeConformanceReply(t, struct{}{}, err)
			s.Events = append([]byte(nil), e.StorageTranscript()...)
			s.Occurrences = cloneStorageOccurrences(e.namespace.storage.occurrences)
			s.Sequence = e.namespace.storage.sequence
			s.Clock = e.namespace.storage.clock.Value
		}
		if name == "actions-and-occurrences" {
			require.Equal(t, uint64(5), steps[0].Written)
			require.Equal(t, uint64(2), steps[1].Written)
			require.Equal(t, uint16(29), steps[1].Errno)
			require.Equal(t, uint16(51), steps[2].Errno)
			require.True(t, bytes.Equal([]byte(`{"storage_crash":"b"}`), steps[3].Reply), "storage crash reply must match exact bytes")
			require.Equal(t, uint64(4), steps[3].Sequence)
			require.Equal(t, uint64(2), steps[3].Occurrences["/workspace/db\x00write"])
		}
		if name == "nonshortening-write" || name == "transcript-capacity" {
			require.Empty(t, steps[0].Events)
			require.NotEqual(t, "{}", string(steps[0].Reply))
		}
		scenarios = append(scenarios, scenario{name, config.Storage, config.Limits.TranscriptBytes, steps})
	}
	data, err := json.MarshalIndent(scenarios, "", "  ")
	require.NoError(t, err)
	data = append(data, '\n')
	name := filepath.Join("testdata", "volume-conformance", "vectors", "fault-decisions.json")
	if *volumeConformanceOutput != "" {
		name = filepath.Join(*volumeConformanceOutput, "vectors", "fault-decisions.json")
		require.NoError(t, os.MkdirAll(filepath.Dir(name), 0o755))
		require.NoError(t, os.WriteFile(name, data, 0o644))
		return
	}
	want, err := os.ReadFile(name)
	require.NoError(t, err)
	require.Equal(t, string(want), string(data))
}
