package network

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
)

const healedEpochName = "healed-stall-epoch-exact-100-101-201"

func TestNetworkHealedEpochLiteralGuards(t *testing.T) {
	valid := networkCorpusHealedEpoch(t)
	if err := checkNetworkHealedEpoch(valid); err != nil {
		t.Fatal(err)
	}
	for _, mutation := range []struct {
		name string
		edit func(*networkCorpus)
	}{
		{"scaled-timeout", func(c *networkCorpus) { c.Config.Limits.StallNanos = 10 }},
		{"changed-write", func(c *networkCorpus) { c.Steps[6].Input.Request.Bytes = []byte("other") }},
		{"changed-timeline", func(c *networkCorpus) { c.Steps[10].Input.Nanos = 11 }},
		{"early-reset", func(c *networkCorpus) { c.Steps[10].Expected.Snapshot.Connections[0].Reset = true }},
		{"wrong-replacement-deadline", func(c *networkCorpus) { c.Steps[10].Expected.Deadline = 16 }},
		{"missing-reset", func(c *networkCorpus) { c.Steps[11].Expected.Snapshot.Connections[0].Reset = false }},
		{"reservation-leaked", func(c *networkCorpus) { c.Steps[11].Expected.State.HistoryReserved = 1 }},
		{"short-write", func(c *networkCorpus) { c.Steps[6].Expected.Result.Count = 3 }},
	} {
		t.Run(mutation.name, func(t *testing.T) {
			data, err := json.Marshal(valid)
			if err != nil {
				t.Fatal(err)
			}
			var changed networkCorpus
			if err := json.Unmarshal(data, &changed); err != nil {
				t.Fatal(err)
			}
			mutation.edit(&changed)
			if err := checkNetworkHealedEpoch(changed); err == nil {
				t.Fatal("changed original epoch assertion accepted")
			}
		})
	}
}

func networkCorpusHealedEpoch(t *testing.T) networkCorpus {
	b := newNetworkCorpus(t, healedEpochName, networkCorpusConfig(Limits{Listeners: 4, Connections: 4, Bytes: 32, ConnectionBytes: 4, StallNanos: 100, Transitions: 32}))
	a, c := b.domains()
	_, x, _, _ := b.pair(a, c, 0)
	b.link("partition", a, c, 0, false)
	b.io(a, "write", x, "held", 0)
	b.advance(100)
	b.link("heal", a, c, 0, false)
	b.link("partition", a, c, 0, false)
	b.advance(101)
	b.advance(201)
	return b.finish()
}

func checkNetworkHealedEpoch(c networkCorpus) error {
	if c.Name != healedEpochName {
		return nil
	}
	if c.Schema != "gomad.network-operation-corpus/v1" || c.Run != 1 || c.Seed != 0 || c.CreationError != "" || c.FinalError != "" || c.RecordError != "" || c.Final == nil {
		return fmt.Errorf("epoch creation/final record differs")
	}
	if !reflect.DeepEqual(c.Config, networkCorpusConfig(Limits{Listeners: 4, Connections: 4, Bytes: 32, ConnectionBytes: 4, StallNanos: 100, Transitions: 32})) {
		return fmt.Errorf("epoch original configuration differs")
	}
	var inputs []networkCorpusInput
	if err := json.Unmarshal([]byte(`[
	 {"action":"register","domain":{"run":1,"node":"a","incarnation":1,"address":"a"}},
	 {"action":"register","domain":{"run":1,"node":"b","incarnation":1,"address":"b"}},
	 {"action":"dispatch","request":{"domain":{"run":1,"node":"b","incarnation":1,"address":"b"},"op":"listen","request_id":1}},
	 {"action":"dispatch","request":{"domain":{"run":1,"node":"a","incarnation":1,"address":"a"},"op":"dial","host":"b","port":20000,"request_id":2}},
	 {"action":"dispatch","request":{"domain":{"run":1,"node":"b","incarnation":1,"address":"b"},"op":"accept","handle":1,"request_id":3}},
	 {"action":"partition","left":"a","right":"b"},
	 {"action":"dispatch","request":{"domain":{"run":1,"node":"a","incarnation":1,"address":"a"},"op":"write","handle":2,"bytes":"aGVsZA==","request_id":4}},
	 {"action":"advance","nanos":100}, {"action":"heal","left":"a","right":"b"},
	 {"action":"partition","left":"a","right":"b"},
	 {"action":"advance","nanos":101}, {"action":"advance","nanos":201}
	]`), &inputs); err != nil {
		return err
	}
	if len(c.Steps) != len(inputs) {
		return fmt.Errorf("epoch step count differs")
	}
	for i, step := range c.Steps {
		if !reflect.DeepEqual(step.Input, inputs[i]) || step.Expected.Error != "" || step.Expected.SnapshotError != "" {
			return fmt.Errorf("epoch original input/success differs at step %d", i)
		}
	}
	written := c.Steps[6].Expected.Result
	before, after := c.Steps[10].Expected, c.Steps[11].Expected
	if written == nil || written.State != "completed" || written.Error != "" || written.Count != 4 {
		return fmt.Errorf("epoch held write did not complete four bytes")
	}
	if before.Snapshot == nil || len(before.Snapshot.Connections) != 1 || before.Snapshot.Connections[0].Identity != 1 || before.Snapshot.Connections[0].Reset || before.Deadline != 201 {
		return fmt.Errorf("epoch stale due reset connection or changed deadline201")
	}
	if after.Snapshot == nil || len(after.Snapshot.Connections) != 1 || after.Snapshot.Connections[0].Identity != 1 || !after.Snapshot.Connections[0].Reset || after.State.HistoryReserved != 0 {
		return fmt.Errorf("epoch replacement due did not reset/release reservation")
	}
	return nil
}
