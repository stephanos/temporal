package runner

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"unsafe"
)

func TestR19LogicalPolicyBothSeedSources(t *testing.T) {
	rows := make(map[string]map[string]uint64)
	for _, jobs := range []uint64{10, 100} {
		selection, err := ParseSeeds(fmt.Sprintf("1-%d", jobs))
		if err != nil {
			t.Fatal(err)
		}
		controller, err := newShardedSeedController(selection, CampaignShard{}, map[uint64]struct{}{}, 2, PolicyAll, 1, CampaignResult{})
		if err != nil {
			t.Fatal(err)
		}
		controllerSource := pendingJobs{seeds: selection.Iterator(), completed: map[uint64]struct{}{}}
		orderingSource := pendingJobs{seeds: selection.Iterator(), completed: map[uint64]struct{}{}}
		if &controllerSource.seeds.ranges[0] != &orderingSource.seeds.ranges[0] {
			t.Fatal("seed range backing is duplicated")
		}
		components := map[string]uint64{
			"controller":                   uint64(unsafe.Sizeof(*controller)),
			"selection_header":             uint64(unsafe.Sizeof(selection)),
			"controller_job_source":        uint64(unsafe.Sizeof(controllerSource)),
			"ordering_job_source":          uint64(unsafe.Sizeof(orderingSource)),
			"controller_iterator":          uint64(unsafe.Sizeof(*controllerSource.seeds)),
			"ordering_iterator":            uint64(unsafe.Sizeof(*orderingSource.seeds)),
			"one_shared_range_backing":     uint64(cap(selection.ranges)) * uint64(unsafe.Sizeof(seedRange{})),
			"two_completion_channel_slots": 4 * uint64(unsafe.Sizeof(runCompletion{})),
			"two_channel_handles":          2 * uint64(unsafe.Sizeof(make(chan runCompletion, 2))),
			"completion_order_done_handle": uint64(unsafe.Sizeof(make(chan struct{}))),
		}
		var total uint64
		for _, count := range components {
			total += count
		}
		components["total_named_logical_bytes_both_roles"] = total
		rows[fmt.Sprint(jobs)] = components
	}
	encoded, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	dir := os.Getenv("R19_RESULT_DIR")
	if dir == "" {
		t.Fatal("R19_RESULT_DIR required")
	}
	if err := os.WriteFile(filepath.Join(dir, "logical-policy-both-roles.json"), append(encoded, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Log(string(encoded))
}
