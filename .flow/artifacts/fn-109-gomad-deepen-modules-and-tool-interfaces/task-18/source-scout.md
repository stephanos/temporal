# Filesystem handle source scout

Read-only investigation of the task-14 typed-command candidate at HEAD
`0dd05b313acd0986312da7fd3159520e6a21f1bf`. No implementation, tests,
generation, Flow mutation, bridge or native qualification was performed.
Dispatch: Codex thinking scout, requested `gpt-6.1-sol` at high effort.
Tier: session (jev-unavailable(no_key)); explicit routing remains authoritative.
Implement only after task 17 establishes its creation-time facade pattern.

## Representation and ownership

Use two filesystem handle representations with three backend test modes.
Standalone and in-process simulation already share one local FS model; do not
duplicate that model to manufacture a third handle implementation.

Exported Handle delegates through a private interface with Read, ReadAt, Write,
WriteAt, Truncate, Chmod, Chtimes, Chdir, Seek, Stat, Path, ReadDir, Close, Sync
and Map. Local implementation owns FS, node, name, offsets, access/append flags,
closed/revoked state and generation. Process implementation owns remote resource
ID, path and closed state, and uses task 14's typed commands/result validation.

Exported Mapping delegates Bytes and Close. Local mapping owns FS/node, region,
shared data, writable/accounting-owner flags, closed/revoked state and generation.
Process mapping owns remote resource ID, cached copied data and closed state.

Creation sites: `fs.go:385,455` FS.Open; `process_volume.go:41` processOpen;
`fs.go:1216,1237` local Map; `process_volume.go:123` processHandleMap.
New/NewSimulation/Current select FS context at `fs.go:155,173`, `runtime.go:35`.
Keep FS.process path-operation selection separately scoped. FS.handles/mappings
track local implementation pointers, so local lifecycle helpers never need
facade assertions. Host processVolumeResource can retain facade pointers.
Patched os and libc adapter interfaces stay intact. New overlay files require
descriptor entries and generation.

## Lifetime and locks

Local operations retain FS.mu and the error order: unavailable FS, revoked or
generation mismatch, then closed (`fs.go:1382`; mappings `:1318`). Close releases
handle counts; unlinked data remains until the last handle closes (`:1181,1395`).
Identical local mapping regions share a buffer and one mapped-byte charge;
closing the charged owner transfers ownership to a surviving alias (`:1277`).
Writes/truncation update mappings and reads/sync flush writable mappings
(`:1334,1349,1369`).

Host dispatch releases the process-resource registry mutex before invoking
local methods (`process_volume_host.go:198`). Resources stay domain-bound,
removed only on successful close, and revoked on lifecycle transitions.
Persistence, preflight, observer-before-journal mutation, sync, fork, revocation
and zeroing stay in the volume model (`volume.go:574,599,719,815,880,892,1014`).
Preserve FS-to-observer/run lock ordering.

## Explicit capabilities and differences

- Loaded mounts remain copied cached readonly nodes; mutation denial belongs to
  the local model. NewSimulation intentionally lacks the ambient loader.
- Local writable mappings require readable+writable access to a volatile file.
  Journaled and readonly files reject writable mapping. Process writable Map
  always returns ENOTSUP before checking closed/access/bounds.
- Process Mapping.Bytes copies and caches its first result; cached calls do not
  revalidate a revoked ID. Local aliases and write-through visibility must not
  be imposed on process mappings.
- Local limits: 100,000 handles/nodes/directory entries, 256 MiB per file,
  1 GiB total and 64 MiB unique mapped regions. Process resources add one global
  100,000-slot registry for handles and mappings together. Transfers retain
  64 MiB data/128 MiB frame bounds, read clamping and write codec rejection.
- Process ReadDir normalizes nil to an empty slice; local positive-count EOF
  returns nil. Shared tests need backend-specific expected values.

## Test map

Reuse task 17's fixture pattern for standalone, in-process and actual process
handle operations: partial ReadAt/EOF, offset preservation, append/WriteAt
rejection, truncate/metadata, sorted incremental directories, chdir, open-node
survival after unlink, close and ErrClosed identity.

Extend local mapping tests (`fs_test.go:400,428,459,517,564,736`) for alias charge
transfer, exact capacity failure, overlap precedence, survival after file close,
stale-generation rejection and zeroing. Existing mount tests are at
`:65,109,773` and Runner `io_filesystem_toolchain_test.go:84`.

Process proxy tests need copied/cached bytes, writable-map precedence,
close/revocation, malformed partial counts/data, registry rollback and domain
mismatch. Reuse task 14's host-dispatch partial/copy/stale vectors rather than
recapturing wire expectations.

Retain real transport volume restart (`volume_toolchain_test.go:146`) and hard
isolation (`cluster_toolchain_test.go:58`). Other volume Parity tests currently
use oneNodeVolumeSpec, which selects only in-process; they do not prove process
parity. New real-process cases must appear in the Runner root integration's
explicit selector. Stock-runtime loopback tests are developmental adapter
evidence, not supported-native process or isolation qualification.
