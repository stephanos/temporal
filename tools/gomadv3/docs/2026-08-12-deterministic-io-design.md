# Deterministic I/O for Gomad v3

## Goal

Gomad-managed programs use deterministic I/O without application changes,
named profiles, suite selectors, or application-specific source overlays. A
target process starts with an empty writable in-memory filesystem. Explicit
read-only mounts are its only source of host filesystem input.

The design must support ordinary Go filesystem users and dependencies that
bypass Go's `os` package through a supported libc implementation. It must never
silently substitute ambient host I/O for a supported deterministic operation.

## Public interface

Deterministic I/O is part of every Runner-managed Gomad execution. There is no
enablement flag and no profile name.

The existing repeatable mount option remains the only filesystem input surface:

```text
--io-ro-mount HOST_DIRECTORY=TARGET_DIRECTORY
```

The Runner resolves the host source during preparation. The target sees only
the normalized target destination. Mount sources do not enter semantic replay
identity because replay never accesses them; mount targets, limits, and
captured contents do.

Direct `GOMADSEED` execution without Runner remains useful for runtime-only
testing. It has deterministic time and scheduling but no brokered host inputs.
Its filesystem still starts empty rather than falling through to the host.

## Filesystem engine

Add a standard-library-internal `internal/gomadfs` package. It owns all mutable
filesystem state and exposes a small operation-oriented interface suitable for
both the patched `os` package and supported dependency adapters. It must not
import `os`; public boundary code translates its internal metadata and errors
into the caller's native types.

Each target process gets one engine containing only the root directory at
startup. The engine owns:

- normalized absolute paths and a process-local current working directory;
- directories, regular files, modes, sizes, and Gomad-clock timestamps;
- open descriptions, per-handle offsets, and directory iteration state;
- create, open, read, write, seek, truncate, stat, rename, unlink, and directory
  operations needed by Go and supported libc consumers;
- mount points and lazily materialized immutable nodes; and
- explicit capacity accounting for nodes, handles, bytes, path length, and
  directory entries.

Nodes and names are distinct. An open handle retains its node after rename or
unlink until the last handle closes. Rename, exclusive creation, append,
truncate, and concurrent offset updates are atomic under engine locks. Directory
enumeration is stable and lexicographically ordered. Unsupported special files,
links, ownership changes, device nodes, and filesystem-specific controls return
stable errors until their semantics are explicitly implemented.

Virtual file timestamps come from Gomad's process clock. Timestamp observation
does not advance time. Operations that mutate metadata use the current logical
instant and deterministic tie behavior when several mutations occur at that
instant.

## Read-only mounts

The existing Runner-owned mount broker remains the only component that can read
approved host roots. `internal/gomadfs` asks the broker to materialize a path on
its first lookup beneath a mount point. The engine then installs the returned
file or directory as an immutable node and serves later operations from memory.

The broker continues to pin and validate source roots, reject traversal,
symlinks, hard links, special files, unstable captures, and capacity overflow,
and persist both positive and negative observations. The engine enforces mount
immutability uniformly across path and handle operations:

- reads and metadata queries use captured memory;
- missing captured paths return `ENOENT`;
- mutation beneath a mount returns `EROFS`; and
- cross-boundary rename returns `EXDEV`.

An undeclared path is resolved only against the writable in-memory namespace.
If it does not exist there, reads return `ENOENT`; no supported code path opens
the host filesystem.

## Go standard-library integration

When Gomad is active, patched `os` entry points delegate to `internal/gomadfs`.
Disabled execution retains the upstream Go implementation exactly. Integration
must cover path operations and `os.File` methods as one coherent boundary,
rather than special-casing mounted handles separately.

The initial supported surface includes:

- `Open`, `OpenFile`, `Create`, and temporary file or directory creation;
- `Read`, `ReadAt`, `Write`, `WriteAt`, `Seek`, `Sync`, and `Truncate`;
- `Stat`, `Lstat`, file-handle `Stat`, and directory enumeration;
- `Mkdir`, `MkdirAll`, `Remove`, `RemoveAll`, and `Rename`;
- `Chmod`, `Chtimes`, working-directory operations, and path existence checks;
  and
- deterministic error wrapping compatible with normal `os.PathError` and
  `os.LinkError` behavior.

Operations not yet supported fail closed with stable errors. They must not call
the upstream host implementation merely because the in-memory engine lacks an
operation.

## Generic dependency adapters

Some pure-Go dependencies implement foreign runtimes and invoke POSIX-like
operations without using Go's `os` package. Gomad handles these through
version-pinned adapters selected from target build metadata.

The first adapter targets the supported `modernc.org/libc` version, not SQLite.
It translates libc filesystem, entropy, clock, and related descriptor
operations into Gomad's existing generic boundaries. Consequently SQLite and
other modernc consumers use the same filesystem engine without Gomad knowing
their packages, databases, schemas, or tests.

Target preparation performs these steps:

1. Inspect the prepared target's module and build metadata.
2. Match detected foreign-runtime dependencies to reviewed adapter versions.
3. Apply only the corresponding pinned source transformation or build overlay.
4. Rebuild and verify the final target.
5. Bind adapter name, dependency version, source digest, and implementation
   digest into launch and replay identity.

An unsupported detected version fails preparation with a compatibility error.
The adapter registry is generic and closed: adding a dependency requires a
reviewed adapter and conformance tests, not a test-specific I/O profile.

Raw syscalls, cgo, plugins, external binaries, and unrecognized native I/O are
outside the trusted deterministic boundary. Target preparation rejects them
when build metadata exposes them. Gomad does not claim an operating-system
sandbox against a target deliberately hiding such behavior.

## Afero decision

Gomad does not use `github.com/spf13/afero` as its production filesystem.
Although Afero is already in the repository dependency graph, it is an
application-facing abstraction that imports `os`. The patched `os` package
cannot depend on it without an import cycle, and unchanged applications would
not use it automatically. Its memory implementation also does not cover libc
callers or Gomad's replay identity, capacity, timestamp, and fail-closed
requirements.

Afero behavior and tests may be used as reference cases when useful. Gomad's
engine remains a small purpose-built deep module with no target-module
dependency.

## Deterministic transcript

Supported external observations are recorded at the boundary where they become
visible to the target. Filesystem transcript records use normalized paths,
operation kinds, semantic flags, result classes, byte counts, content digests,
and handle-independent ordering data. They do not contain host paths, pointer
values, raw descriptors, or host timestamps.

Replay supplies the expected transcript before target initialization. The same
execution recreates writable state from operations. A mismatch in operation,
arguments, result, content, or order is an exact replay divergence. Read-only
mount artifacts independently provide the captured input bytes needed to
execute those operations without the source tree.

Writable filesystem snapshots are not stored. This keeps the artifact model
small and tests determinism rather than hiding it by restoring end state. A
crash artifact retains the transcript and observed mount inputs up to the
failure.

## Identity and configuration

Remove named I/O profiles and their selector validation. Runner launch identity
instead binds:

- Gomad Runner and patched toolchain builds;
- target bytes, build metadata, arguments, and environment;
- deterministic I/O protocol and filesystem implementation versions;
- detected adapter identities;
- normalized mount targets and all filesystem limits; and
- transcript and captured-input schemas.

The bootstrap configuration carries these identities and limits before package
initialization. Replay validates them before executing the stored target. A
different adapter, filesystem implementation, mount layout, or limit is not an
exact replay of the original run.

## Errors and failure classification

Application-visible filesystem conditions use stable POSIX-compatible errors.
Invalid Runner configuration, malformed broker frames, capacity exhaustion,
adapter mismatch, transcript overflow, and impossible engine invariants are
structured Gomad failures. Error paths must preserve the distinction between:

- a normal target-visible filesystem result;
- deterministic replay divergence;
- invalid or unsupported target preparation;
- bounded-capacity exhaustion; and
- Runner, broker, or toolchain failure.

No error path may retry through ambient host I/O.

## Testing strategy

Implementation proceeds test-first in layers:

1. `internal/gomadfs` unit tests cover path normalization, open flags, file and
   directory lifecycle, offsets, rename and unlink with open handles, timestamps,
   ordering, mount immutability, concurrency, every limit, and stable errors.
2. Standard-library overlay fixtures run unchanged Go `os` calls and prove the
   in-memory engine matches the supported contract while the disabled toolchain
   retains upstream behavior.
3. Mount integration tests prove lazy capture, undeclared-host invisibility,
   negative lookup persistence, and replay after deleting or changing sources.
4. Adapter tests exercise a small generic modernc/libc fixture. SQLite is one
   consumer-level conformance case, not an adapter identity or special path.
5. Preparation tests reject unsupported adapter versions and unsafe target
   features while binding supported adapter identities.
6. Record and replay tests prove writable state is reconstructed by execution
   and divergences are located precisely.
7. Unchanged Temporal functional suites validate the generic boundary. Their
   names and selectors appear only in external test commands and the suite
   results document, never in Gomad implementation or identity schemas.

Existing named-profile tests stay green until their replacement behavior has a
failing test and an implemented migration. The final change removes those
profiles and the Temporal-specific build overlay together, so Gomad never has a
mixed public model.

## Performance, scaling, and security trade-offs

The filesystem engine serializes namespace mutations and individual open
description offsets. This favors clear deterministic semantics over host
filesystem throughput. Reads of immutable captured data can share storage; file
writes use bounded in-memory buffers. At ten times the current workload,
resource use grows with created nodes and unique written or captured bytes,
while repeated reads remain in process. Limits fail explicitly rather than
allowing unbounded memory growth.

Lazy mounts keep artifacts proportional to observed host input. Re-executing
writes makes replay cost proportional to target work, which is intentional.

The process boundary, closed environment, in-memory filesystem, and brokered
mounts prevent accidental host dependence by supported trusted code. They do
not contain adversarial raw syscalls. Extending security claims would require a
separate operating-system sandbox design and is outside this work.

## Migration sequence

1. Introduce and qualify `internal/gomadfs` behind the existing activation path.
2. Move current virtual directories and mounted handles into the engine.
3. Complete the supported `os` surface and eliminate supported host fallback.
4. Add the generic adapter registry and modernc/libc adapter.
5. Make deterministic I/O bootstrap unconditional for Runner-managed targets.
6. Remove named profiles, selector validation, and the Temporal-specific SQLite
   overlay.
7. Rerun the functional-suite inventory and classify only genuinely unsupported
   generic boundaries.

Each migration step keeps replay schemas versioned and either backward
compatible or explicitly rejects older artifacts. The final public model has
one deterministic I/O contract rather than parallel legacy and generic modes.
