package artifact

import (
	"errors"

	"go.temporal.io/server/tools/gomad3/record"
)

// SharedTarget names the target copy an artifact has in common with the other
// artifacts it is kept with. The zero value is an artifact that owns its copy.
type SharedTarget struct {
	SHA256 record.SHA256
	Bytes  uint64
}

// RetainedBytes applies the stored-bytes rule to artifacts that are kept
// together.
//
// An artifact's stored bytes (Artifact.StoredBytes) are its manifest and every
// file the manifest lists, the target included. They are a property of the
// artifact alone: an artifact whose target is a link to its store's pool has
// the same stored bytes as one that owns its copy, so manifests, identities,
// and every bound on one artifact are the same with and without sharing.
//
// A bound on several artifacts sums them here, and a target the artifacts share
// is counted once: the first artifact of a shared target adds all of its stored
// bytes, and each later one adds its stored bytes without the target.
//
// The caller says which artifacts share a target, and the callers differ:
//
//   - A corpus shares a target among the cases whose target file is its pool's
//     entry on disk (PoolTarget). A case with a private copy counts in full.
//   - A merged campaign shares a target among all evidence that records the same
//     target SHA-256, whether or not the shard stores share the file on disk.
//     It accounts for one store that holds all of the merged evidence. Merge
//     copies no evidence, so shards of two artifacts roots keep one copy per
//     root while the merged limits count one.
//   - The byte limits inside one campaign do not sum here: every retained
//     artifact counts its full stored bytes, linked to the pool or not. The
//     campaign record states retained_success_bytes as the sum of its
//     executions' success_artifact_bytes and holds that sum to the success-byte
//     limit beside it. Counting a shared target once would change that recorded
//     sum or let it pass its recorded limit, and neither the record nor its
//     journal says which artifacts share a target. A campaign's limits
//     therefore bound what its artifacts take as standalone copies, which is
//     more than they take in a store that shares their target.
type RetainedBytes struct {
	total   uint64
	targets map[record.SHA256]struct{}
}

// Total is the sum of the artifacts added so far.
func (retained *RetainedBytes) Total() uint64 {
	return retained.total
}

// Cost is what Add would add for this artifact.
func (retained *RetainedBytes) Cost(storedBytes uint64, shared SharedTarget) (uint64, error) {
	if shared == (SharedTarget{}) {
		return storedBytes, nil
	}
	if shared.SHA256 == "" || shared.Bytes == 0 || shared.Bytes >= storedBytes {
		return 0, errors.New("artifact shared target is not a part of its stored bytes")
	}
	if _, counted := retained.targets[shared.SHA256]; counted {
		return storedBytes - shared.Bytes, nil
	}
	return storedBytes, nil
}

// Add accounts for one artifact and returns the bytes it added.
func (retained *RetainedBytes) Add(storedBytes uint64, shared SharedTarget) (uint64, error) {
	cost, err := retained.Cost(storedBytes, shared)
	if err != nil {
		return 0, err
	}
	if cost > ^uint64(0)-retained.total {
		return 0, errors.New("artifact byte count overflows uint64")
	}
	retained.total += cost
	if shared != (SharedTarget{}) {
		if retained.targets == nil {
			retained.targets = make(map[record.SHA256]struct{})
		}
		retained.targets[shared.SHA256] = struct{}{}
	}
	return cost, nil
}

// TargetOf is the target of a manifest as the copy its artifact would share.
func TargetOf(manifest record.ExecutionRecord) SharedTarget {
	return SharedTarget{SHA256: manifest.Target.SHA256, Bytes: uint64(manifest.Target.Size)}
}

// PoolTarget is the target an artifact shares through the pool: its target
// when that file is the pool's entry, and the zero value when the artifact owns
// a private copy.
func PoolTarget(pool, artifactPath string, manifest record.ExecutionRecord) SharedTarget {
	if pool == "" || !sharesPoolEntry(pool, artifactPath, manifest) {
		return SharedTarget{}
	}
	return TargetOf(manifest)
}
