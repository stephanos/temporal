# Lazy read-only mounts for Gomad v3

## Goal

Let an unchanged target read explicitly approved host directories during a
recording run while preserving deterministic, host-independent replay. Gomad
captures only entries the target observes. Writes and undeclared host access
remain unavailable.

The first qualification target is the unchanged
`TestActivityAPIBatchSecurityTestSuite`. Its SQLite schema reads must work
without changing Temporal source or tests.

## Public interface

`gomad explore` accepts repeatable mappings:

```text
--io-ro-mount HOST_DIRECTORY=TARGET_DIRECTORY
```

For the first target:

```text
--io-ro-mount ./schema/sqlite/v3=go.temporal.io/server/schema/sqlite/v3
```

The host side is resolved relative to the Runner working directory during
preparation. The target side is normalized into Gomad's virtual absolute path
space, so the relative target path above is visible from the target's isolated
working directory. Mount destinations must not overlap. Mounts require an I/O
profile and become part of its recorded launch identity.

## Architecture

The target must not open mounted host paths directly. Its overlaid `os`
implementation sends typed lookup requests to a Runner-owned mount broker over
reserved descriptors. The broker alone owns pre-opened directory handles for
approved roots.

The broker resolves every path component relative to its root without following
symlinks. It supports only the operations needed for a general read-only Go
filesystem:

- open and sequential/random reads of regular files;
- `Stat` and file-handle `Stat`;
- `ReadDir` and file-handle directory reads;
- close and seek where required by standard `os.File` behavior.

The target receives file content and metadata, then serves subsequent operations
from its in-memory filesystem. Write-capable opens, creation, removal, rename,
chmod, and truncation fail with `EROFS`. Sockets, devices, FIFOs, hard-link
ambiguity, and other special entries fail closed as unsupported.

## Lazy capture and consistency

Record mode captures an entry on first observation:

1. The broker resolves the requested entry beneath a pre-opened mount root.
2. For a regular file it validates metadata, reads bounded contents, validates
   metadata again, and rejects the capture if identity, size, or modification
   state changed.
3. For a directory it captures one sorted, complete child listing with the type
   and stable metadata of every child. Later changes on the host do not alter the
   cached target view.
4. The broker returns the captured entry and appends its canonical representation
   to the I/O transcript/payload set before the target can observe it.

An entry already captured is never read from the host again during that run.
Directory listings and file contents therefore describe a coherent observed
view per entry, not an atomic snapshot of the entire mount. A rename or mutation
racing first capture produces a deterministic host-input failure rather than a
partially captured value.

## Artifacts and replay

Each retained artifact contains:

- canonical mount mappings with source identities stripped from semantic replay
  projections;
- every observed regular file's target path, mode, size, digest, and contents;
- every observed directory's target path, mode, and sorted complete listing;
- a digest over the canonical observed mount set;
- transcript records for lookups and reads.

Exact replay does not open the configured host roots. The broker is populated
only from the artifact payloads. A request for an entry or directory listing not
captured by the original run is a replay divergence. Verify-only replay validates
all payload digests and mappings without executing the target.

Successful runs do not need publication artifacts today, so qualification uses
same-seed transcript equality. A deterministic failure fixture exercises full
artifact publication and host-independent replay.

## Limits and errors

Runner configuration applies explicit per-run limits for mounted file count,
single-file bytes, total captured bytes, directory entries, request count, and
path length. Limit exhaustion is a structured host-input/capacity failure and
cannot degrade into target-visible partial data.

Malformed paths, traversal, overlapping destinations, duplicate mappings,
symlinks, special files, unstable capture, broker protocol errors, and unexpected
broker termination fail closed with stable classifications. Target attempts to
write return `EROFS`; they are target-visible errors and are recorded.

## Isolation and security

Host roots are opened and validated during preparation, then pinned by directory
handle for the run. Descendant lookup is relative to that handle and never uses
an attacker-controlled absolute path. The implementation does not invoke an OS
sandbox and does not claim to contain raw syscalls from trusted target code; it
extends the existing reviewed Gomad `os` boundary.

The broker protocol is bounded, framed, versioned, and has a single request per
sequence number. All bytes crossing it are validated before allocation. The
target cannot ask the broker to enumerate above a declared root.

## Testing

Implementation proceeds test-first in this order:

1. Configuration tests reject invalid, duplicate, overlapping, and non-directory
   mounts while binding normalized mappings into profile identity.
2. Broker unit tests cover regular files, sorted directories, empty files,
   traversal, symlinks, special files, mutations during capture, and every limit.
3. Toolchain tests prove `Open`, `ReadFile`, `Stat`, `ReadDir`, seek, and close use
   captured memory; writes fail with `EROFS`; and no undeclared host path is read.
4. Replay integration tests delete or change the host tree after recording and
   prove exact replay still succeeds, while an unrecorded lookup diverges.
5. The unchanged `TestActivityAPIBatchSecurityTestSuite` runs twice at seed 7 and
   must produce successful exits and byte-identical I/O transcripts.

After that suite passes, the
[functional-suite sweep](2026-08-11-functional-suite-sweep.md) is updated and
the remaining schema-blocked suites are rerun to reveal their next Gomad
boundary.

## Trade-offs and scaling

Lazy capture keeps artifacts proportional to observed input instead of mounted
tree size. It adds a broker round trip on first access; cached accesses are
in-process. Large or highly fragmented workloads remain bounded by configuration
and may fail capacity checks rather than consuming unbounded memory or disk.

A crash before artifact publication leaves the existing partial-run diagnostics.
An abrupt broker failure terminates the run as an infrastructure failure. At ten
times the current read load, memory and artifact size grow with unique observed
bytes, while repeated reads remain cheap; operators must raise reviewed limits
rather than silently accepting larger input.
