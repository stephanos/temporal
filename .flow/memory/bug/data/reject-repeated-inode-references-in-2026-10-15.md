---
title: Reject repeated inode references in recovery path maps
date: "2026-10-15"
track: bug
category: data
module: tools/gomad/internal/backends/wasm/wasmhost/src/volume/image.rs
tags: [recovery, volume, inode]
problem_type: data
symptoms: Recovered writes can update another path after an image aliases one inode
root_cause: Object inode uniqueness did not check path reference uniqueness
resolution_type: fix
---

## Problem
An authenticated recovery image could map two durable paths to one inode inside a volume. Recovery rebuilt separate nodes, while inode-based writes resolved the first node, so writing through one recovered path could change another path's contents and contradict storage evidence.

## What Didn't Work
The decoder already rejected duplicate object inode IDs across volumes and reserved nodes, but validating object uniqueness did not validate uniqueness of inode references in the path map. A valid image digest did not establish structural consistency.

## Solution
RecoveryImage::decode now tracks inode references while decoding each volume's persisted paths and refuses a repeated reference before rebuilding. The regression rehashes an authentic Go corpus image with file or directory aliases and asserts the structural refusal. Six Go/Rust conformance cases preserve cross-directory file and directory rename recovery with no parent sync or either parent synced.

## Prevention
Validate both object identity and path-to-object cardinality when reconstructing a namespace from untrusted images. Recompute digests in malformed-image tests so checksum refusal cannot mask the intended invariant. Audit the reference model's atomic rename persistence before rejecting aliases; this model has no hard-link operation and records both sides of a rename in one persistence effect.
