---
title: "Shard merge and prepared-target cache must bind source identity, not go.mod"
date: "2026-09-29"
track: bug
category: integration
module: tools/gomad3/qualification/set
tags: [gomad3, qualification, cache, sharding]
problem_type: integration
symptoms: mixed-revision shards merged green; stale prepared binaries after dependency go directive change
root_cause: identity checks compared configuration digests instead of the reviewed closure and all build inputs
resolution_type: fix
related_to: [bug/integration/go-mod-download-inside-a-target-module-2026-09-28]
---

## Problem
The qualify-set shard merge compared only the run identity (module go.mod digest, toolchain, profile) and dropped per-workload analyses, so shards run against different server source revisions merged into one green report. The prepared-target cache identity hashed only the root module files, so a dependency's `go` directive change (loop-variable semantics) restored a stale binary; default VCS stamping put unbound repository state into restored binaries; and concurrent shards trimming one shared go build cache could delete archives another build had already looked up.

## What Didn't Work
Treating the go.mod digest as the source identity, and assuming "the go command rebuilds a missing cache entry" makes trimming safe under concurrency (it only rebuilds on lookup, not after).

## Solution
- merge.go rejects shards whose analyses disagree on the closure SHA for one analysis identity (package, tags, capability mode).
- prepared_cache.go binds each dependency module's effective GoVersion and a local module's go.mod (schema v2); target.go passes -buildvcs=false.
- builds hold hostfs.Shared on `<cache>/gomad-cache.lock`; TrimCache takes hostfs.Try exclusively and skips when contended; ErrNotExist during the walk is tolerated.

## Prevention
Any cache keyed by "every build input" needs a test per input class that flips only that input and asserts different behavior (dependency language version, VCS state). Any aggregate of partial runs must compare the evidence that names sources, not just configuration.
