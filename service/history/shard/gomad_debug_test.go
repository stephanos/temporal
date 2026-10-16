//go:build gomad && wasip1 && wasm

package shard

import (
	"testing"

	persistencespb "go.temporal.io/server/api/persistence/v1"
	"go.temporal.io/server/common/gomaddebug"
)

func TestGomadShardDumpUsesSnapshotWithLockHeld(t *testing.T) {
	s := &ContextImpl{shardID: 7, shardInfo: &persistencespb.ShardInfo{RangeId: 19}, state: contextStateAcquired}
	s.rwLock.Lock()
	slot := gomadShardSequence % uint64(len(gomadShardViews))
	s.gomadDebugSnapshot()
	s.shardInfo.RangeId = 23
	var w gomaddebug.DumpWriter
	dumpGomadShards(&w)
	s.rwLock.Unlock()
	if gomadShardViews[slot].rangeID != 19 {
		t.Fatal("dump did not retain last completed snapshot")
	}
}
