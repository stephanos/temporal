# Gomad runner, artifact, and seed replay plan

## Decision summary

Implement the post-v3 Runner as a Go command in `tools/gomadv3`. It prepares
one immutable target binary, then launches one fresh target process per seed
with bounded cross-process parallelism. Each process receives the same target,
arguments, allowlisted environment, World snapshot, toolchain, and
architecture; only `GOMADSEED` changes. The Runner enforces separate per-run
and overall host deadlines, drains bounded stdout and stderr, and terminates the
target's complete process group on cancellation or timeout.

Failures are published as versioned, atomic artifact directories. An artifact
contains a JSON manifest, the exact target binary, bounded stdout and stderr,
the initial World snapshot, World transition transcript, optional final World
snapshot, and SHA-256 identities for every payload. A domain-separated failure
signature groups byte-equivalent observations without treating the seed itself
as the failure identity. Seed replay validates the complete record identity,
then runs exactly the recorded seed; it never silently rebuilds or substitutes
an input.

The initial implementation supports seed exploration and seed replay. Forced
external-event replay and delta minimization are later phases behind the same
record contract. Runner, Record, and World remain separate deep modules. World
owns its snapshot, transition, and semantic-digest schemas. Record owns the
outer artifact envelope and raw payload hashes. Runner composes the two without
either package importing the other.

No new third-party dependency is required. Deterministic targets remain
unsupported on Windows and with cgo, external linking, plugins, foreign
threads, or the race detector.

## Goals

- Prepare or build a target once and run every selected seed against those
  immutable bytes.
- Run explicit seeds and inclusive seed ranges with bounded parallelism.
- Bound every target by a host-time deadline and the complete exploration by a
  separate overall deadline.
- Drain stdout and stderr concurrently without allowing output volume or pipe
  backpressure to hang a target or exhaust Runner memory.
- Distinguish target outcomes from failures in preparation, launch,
  containment, capture, publication, or the host watchdog.
- Publish one self-contained artifact per distinct failure signature, with
  content hashes and recoverable partial diagnostics.
- Replay one exact seed only after strict schema, payload, toolchain,
  architecture, target, argument, environment, and World identity validation.
- Keep per-running-seed memory bounded and make a 10x larger seed set cost
  approximately 10x more work rather than 10x more retained memory.

## Non-goals

- Exhaustive schedule exploration or a claim that a seed range covers the state
  space.
- Numerically minimizing a seed. A seed is an opaque schedule selector.
- External-event replay, runtime-choice replay, or delta minimization in the
  first implementation.
- Making host filesystem, socket, DNS, database, process, signal, or readiness
  behavior deterministic.
- Treating stdout, stderr, host completion order, or host duration as semantic
  scheduling inputs.
- Running more than one seed in a target process.
- Supporting multiple `go test` packages in one exploration. Each package has
  a different generated test binary and is a separate Runner invocation.
- Windows, cgo, external linking, plugins, foreign threads, the race detector,
  or hostile programs that deliberately escape their process group.

## Execution contract

### Target preparation

Runner supports three target forms:

1. `exec` copies an existing regular executable and its required Gomad
   provenance descriptor into a private preparation directory and validates
   the recorded Go version, toolchain build key, target platform, and binary
   hash before any seed starts. The descriptor is a trusted build attestation,
   not a signature; exact replay ultimately relies on the stored binary bytes.
2. `go-run` invokes the pinned Gomad Go command once with `go build`, then runs
   the resulting binary for every seed.
3. `go-test` invokes the pinned Gomad Go command once with `go test -c`, always
   includes `-tags test_dep`, then runs that generated test binary for every
   seed.

Preparation runs outside deterministic mode with `GOMADSEED` and
`GOMADV3_CHILD_SEED` removed, `CGO_ENABLED=0`, `GOTOOLCHAIN=local`, and
`TZ=UTC`. The configured Go command must resolve through
`tools/gomadv3/.toolchain/bin/go`, and its build key must match
`tools/gomadv3/.toolchain/build-key`. Runner does not download or rebuild the
custom toolchain; a missing or stale toolchain is a host preparation failure
with the existing `make -C tools/gomadv3 toolchain` remediation.

Build tags are a sorted, deduplicated list. `go-test` adds `test_dep` even when
the caller supplies no tags. Runner rejects `race`, empty tags, tags containing
commas or whitespace, cgo enablement, external-link flags, and plugin build
modes. It passes argument vectors directly and never evaluates shell text.

The prepared target is read-only for the remainder of the batch. Runner hashes
it again before artifact publication so mutation of its private preparation
directory becomes a host integrity failure rather than a misleading target
result.

### Per-seed isolation and containment

Runner uses a small hidden supervisor mode of its own executable on Unix. The
supervisor starts the target as leader of a new process group, reports the PID
and process-group ID to Runner, waits for the target, and reaps it. A liveness
pipe connects Runner to the supervisor. EOF, an explicit cancel message, or the
supervisor's copy of the host deadline starts termination even if Runner is
stalled or exits unexpectedly.

Termination is:

1. send `SIGTERM` to the negative process-group ID;
2. wait the configured termination grace period;
3. send `SIGKILL` to the process group if any member remains;
4. wait for the group leader and verify `kill(-pgid, 0)` returns `ESRCH`.

Failure to establish the group, signal it, reap the leader, or prove the group
gone is a host containment failure. Descendants that deliberately create a new
session violate the trusted-target contract. Arbitrary host subprocesses are
already outside deterministic mode; the group rule exists to contain bugs and
unsupported subprocess use, not adversarial evasion.

Every seed receives a new empty working directory and a stable logical
`argv[0]` of `gomadv3-target`. Reading the absolute working directory,
`os.Executable`, or other host path state is outside the deterministic contract.

### Deadlines

- `--run-timeout` is host time for one prepared target process. It defaults to
  30 seconds and must be positive.
- `--overall-timeout` starts before preparation and defaults to 10 minutes. It
  bounds preparation, all seed runs, publication, and shutdown.
- `--terminate-grace` defaults to 2 seconds and is included inside both
  deadlines rather than extending them.
- The effective deadline for a target is the earlier of its per-run deadline
  and the overall deadline.

The standard `go test` timeout remains virtual logical time, as specified by
`GOMADv3_CLOCK.md`. A logical test timeout is a target failure. A per-run host
watchdog timeout is a distinct `watchdog` run outcome, not a deterministic
target result and not a Runner infrastructure failure. Once the process group
is successfully terminated and reaped, it participates in stop policies and
exploration may continue. Failure to contain or reap the group is a Runner/host
failure. An overall timeout means the batch is incomplete and is classified
separately from completed target and watchdog outcomes.

### Environment allowlist

Target environments are constructed from an empty slice. Runner always sets
only:

- `GOMADSEED` to the selected unsigned decimal seed;
- `TZ=UTC`; and
- values supplied by repeated `--env NAME=VALUE` flags.

`GOMADSEED`, `GOMADV3_CHILD_SEED`, `TZ`, `CGO_ENABLED`, `GODEBUG`,
`GOMAXPROCS`, `GOEXPERIMENT`, and dynamic-loader variables are reserved and
cannot be supplied. Names must match `[A-Za-z_][A-Za-z0-9_]*`, must be unique,
and cannot contain NUL. Values cannot contain NUL. Entries are sorted by name
before launch and recording.

Runner has no ambient-environment inheritance flag. A target environment value
is deterministic input and is stored verbatim in the artifact; callers must not
put credentials in it. Host build credentials may be used by the outside
preparation command but are never copied into the target environment or record.

### Bounded streaming output

Runner drains stdout and stderr concurrently from process start until EOF. For
each stream it computes SHA-256 over every byte and counts the complete byte
length while retaining a bounded head and tail. The default per-stream limit is
8 MiB: the first 6 MiB and last 2 MiB. A smaller configured limit keeps 75% as
head and 25% as tail, rounding the tail down; zero is rejected.

When the complete stream exceeds the limit, the artifact file contains the
head, a fixed UTF-8 truncation marker containing the discarded byte count, and
the tail. The manifest records the full-stream hash, retained-file hash, total
bytes, retained bytes, discarded bytes, and truncation flag. Failure signatures
use the full-stream hash, so different discarded output cannot be deduplicated
merely because its retained head and tail match. Capture uses constant memory
per stream and continues draining after the retention limit.

## Pattern survey

- `Makefile:170` — `gomadv3-run` already keeps `cmd/go` outside Gomad and uses
  the child activation wrapper for one seed.
- `Makefile:177` — `gomadv3-test` already supplies `-tags test_dep`, disables
  cgo, and transfers one seed to a generated test binary.
- `tools/gomadv3/exec.sh:5` — the existing wrapper validates the child-only seed
  handoff, preserves argv with `exec "$@"`, and removes
  `GOMADV3_CHILD_SEED` before target start.
- `tools/gomadv3/testlib.sh:3` — `gomad_run_checked` captures outputs/status and
  kills a process group, but has no output bound, record, or replay contract.
- `tools/gomadv3/test.sh:687` — `require_clock_behavior` distinguishes
  repeatable target behavior, process-group watchdog timeouts, and deterministic
  deadlock.
- `tools/gomadv3/build.sh:118` — the existing build key already identifies the
  Go source, patch, overlay, platform, bootstrap, and build environment.
- `tools/gomadv3/build.sh:191` — `publish_toolchain` uses temporary files and
  rename to publish the stable launcher and build-key stamp.
- `tools/gomadv2/gomadruntime/testmain.go:77` — v2 `parseSeeds` supports sets and
  ranges but uses signed seeds in one process and under-validates ranges.
- `tools/gomadv2/metatesting/metatest.go:81` and `:158` — v2 reuses a protocol
  child without concurrent-call, deadline, restart, or cleanup policy; v3 uses
  a fresh supervised process per seed.
- `GOMADv3_NEXT.md:31` — Runner, World, Adapters, and Record are already the
  intended post-v3 deep modules.
- `GOMADv3_NEXT.md:202` — the roadmap requires isolated children, bounded
  parallelism, deadlines, containment, signatures, budgets, and atomic records.
- `GOMADv3_CLOCK.md:189` — only the target may receive `GOMADSEED`; the Go build
  driver performs host I/O and must remain outside deterministic mode.
- `GOMADv3_CLOCK.md:587` — target deadlock/logical timeout, host timeout,
  truncation, and toolchain failure are already distinct outcomes.

These patterns anchor the design: `GOMADSEED` remains the only runtime switch;
one P is per target; parallelism is across processes; host deadlines never
enter virtual time; World choices do not consume the runtime stream; every Go
test command includes `-tags test_dep`; and v3 retains its
standard-library-only module.

## Alternatives and recommendation

### A. Go Runner with one prepared binary and fresh supervised children — recommended

This matches the reproducibility unit, builds once, scales across cores, and
centralizes limits, classification, records, and replay. Its costs are process
startup per seed, completion-order-sensitive early stopping, target copies in
distinct artifacts, and process groups that contain trusted descendants but
are not a hostile-code sandbox.

### B. Reuse one long-lived target process for many seeds

This lowers startup cost and resembles v2 metatesting, but violates the v3
process reproducibility unit. Globals, goroutines, descriptors, runtime random
state, and allocator state can leak between seeds; protocol recovery,
per-request cleanup, and synchronization add more failure modes. A hung request
can poison the worker and complicate exact replay. Reject this design.

### C. Shell orchestration around Make targets

This reuses current one-seed entry points with little Go code, but portable
bounded streaming, atomic records, strict replay, structured wait status, and
crash-aware supervision become fragile. Shell quoting encourages command
strings and splits schema policy across tools. Retain the shell harness for
black-box tests, not as the product Runner.

## Command-line interface

The command is built as `tools/gomadv3/.bin/gomad`. Runner flags precede the
target kind, and `--` separates the target's arguments:

```text
gomad explore [runner flags] exec --provenance FILE -- BINARY [ARG ...]
gomad explore [runner flags] go-run PACKAGE -- [ARG ...]
gomad explore [runner flags] go-test PACKAGE -- [TEST_BINARY_ARG ...]
gomad replay [--verify-only] ARTIFACT_DIR
```

Concrete examples:

```sh
tools/gomadv3/.bin/gomad explore --seeds 0-999 --parallel 8 \
  --on-failure first go-run ./tools/gomadv3/testdata/clock -- timers

tools/gomadv3/.bin/gomad explore --seeds 1,7,11-20 --parallel 4 \
  --on-failure all go-test ./common/timer -- \
  -test.run=TestGate -test.count=1 -test.timeout=48h

tools/gomadv3/.bin/gomad replay \
  .gomad/artifacts/v1/run-20260810T120000Z-a1b2/failures/sha256-2f6d9a8c
```

Explore flags are:

| Flag | Default | Contract |
| --- | --- | --- |
| `--seeds` | `1` | Comma-separated unsigned decimal seeds and inclusive ranges |
| `--parallel` | `min(host CPUs, 8)` | Positive maximum active target processes |
| `--run-timeout` | `30s` | Positive host deadline per seed |
| `--overall-timeout` | `10m` | Positive host deadline including preparation/publication |
| `--terminate-grace` | `2s` | Nonnegative grace inside the active deadline |
| `--on-failure` | `first` | `first`, `budget`, or `all` |
| `--failure-budget` | `1` | Positive distinct-signature threshold in `budget` mode only |
| `--output-limit` | `8MiB` | Positive retained bytes per stdout/stderr stream |
| `--world-transition-limit` | `64MiB` | Positive complete World transcript capacity; never silent truncation |
| `--artifacts` | `.gomad/artifacts` | Artifact root |
| `--env` | none | Repeated target `NAME=VALUE` entry |
| `--build-tag` | none | Repeated validated Go build tag |

The World connection phase adds a typed initial-snapshot input only after an
adapter can consume it without reading host files inside the deterministic
region. The initial Runner must not expose a snapshot-path flag that merely
makes the target perform unsupported host I/O.

Seed values use the full `uint64` range. Empty terms, signs, whitespace,
reversed ranges, overflow, duplicates, and overlapping ranges are errors.
Ranges are iterated lazily and the selected count is checked for overflow; the
full selection is never expanded in memory.

`first` stops dispatch and cancels active targets after the first target or
watchdog failure. `budget` stops dispatch after the configured number of
distinct target or watchdog failure signatures and lets active targets finish.
Results already completed
when the coordinator observes either threshold are preserved, so a parallel
batch can report more failures than the stopping threshold. `all` attempts the
complete selected set unless an overall deadline or host failure makes the
batch untrustworthy.

CLI exit statuses are stable: 0 means all attempted targets succeeded, 1 means
one or more target/watchdog failures or a reproduced failure, 2 means invalid
CLI or incompatible replay input, and 3 means a Runner/host failure prevented a
trustworthy completed batch.

## Go package and file layout

| Path | Module and interface |
| --- | --- |
| `tools/gomadv3/cmd/gomad/main.go` | Parse subcommands, render summaries, map typed results to exit status |
| `tools/gomadv3/internal/runner/runner.go` | Deep orchestration module: `Run(context.Context, Config) (Summary, error)` |
| `tools/gomadv3/internal/runner/seeds.go` | Lazy validated seed/range iterator |
| `tools/gomadv3/internal/target/target.go` | `Prepare(context.Context, Spec) (Prepared, error)` over exec/go-run/go-test adapters |
| `tools/gomadv3/internal/process/process_unix.go` | `Run(context.Context, Request) Result`; supervisor protocol, process groups, wait status, deadlines |
| `tools/gomadv3/internal/process/output.go` | Concurrent full hashing plus bounded head/tail capture |
| `tools/gomadv3/internal/record/record.go` | Versioned manifest, batch, outcome, stream, file, and World reference types |
| `tools/gomadv3/internal/record/canonical.go` | Canonical JSON/JSONL validation and domain-separated identities |
| `tools/gomadv3/internal/artifact/store.go` | `Store.Publish(record.Input) (Artifact, error)` and partial-run lifecycle |
| `tools/gomadv3/internal/artifact/open.go` | Strict manifest/path/hash reader used by replay |
| `tools/gomadv3/internal/replay/replay.go` | `Replay(context.Context, Artifact) (Result, error)`; identity preflight and one-seed comparison |

Runner's interface hides preparation, worker scheduling, stop policy, and
publication. Process has one interface because native execution and a fake
adapter are both required by tests. Target has three real adapters behind one
preparation interface. Record contains no filesystem or World semantics.
Artifact contains no scheduling policy. Record and World do not import each
other. The phase-five composition layer in Runner imports both, stores World's
canonical data as opaque payloads, and copies World's semantic digests into the
manifest.

All implementation packages use only the Go standard library. Unix-specific
process code has Darwin and Linux files where `SysProcAttr` differs, plus a
compile-time unsupported-platform file that returns a stable configuration
error.

## State machines and data flow

### Batch state

```text
validate configuration
          |
          v
prepare and hash one target ----failure----> publish host batch diagnostic
          |
          v
dispatch seeds up to parallel limit
          |
          +<---------------+
          v                |
collect one run result     |
          |                |
          v                |
classify, sign, publish ---+
          |
 stop policy / seeds exhausted / overall deadline
          |
          v
cancel or drain workers -> publish batch summary -> return stable status
```

At most `--parallel` supervisors and target processes are active. Workers own
only their seed, process request, capture buffers, and partial directory. A
single coordinator owns the failure-signature map, stop-policy state, batch
counts, and `runs.jsonl` append, avoiding races in deduplication and budgets.

Each run advances through staging, starting, running, exited, captured, and
classified. Every transition atomically replaces `partial.json`. Any host
failure moves through terminate, reap, and preserve-partial. Deduplication waits
for output EOF, wait status, and successful group cleanup.

## Record and artifact contract

### Directory layout

```text
.gomad/artifacts/v1/RUN_ID/
  batch.json
  runs.jsonl
  failures/
    sha256-SIGNATURE_PREFIX/
      manifest.json
      target
      stdout
      stderr
      world/
        snapshot.json
        transitions.jsonl
        final-snapshot.json
  .partial/
    SEED-RUN_ID/
      partial.json
      stdout.head
      stderr.head
      work/
```

`RUN_ID` is a diagnostic UTC timestamp plus 128 bits from `crypto/rand`; it is
not a reproducibility identity. `SIGNATURE_PREFIX` is the first 16 bytes of the
failure signature in lowercase hexadecimal. The complete signature is in the
manifest, and a prefix collision uses the complete hex value as the directory
name.

`runs.jsonl` contains one bounded canonical result summary per completed seed
in host completion order. Every line carries the selection ordinal and seed, so
consumers sort explicitly when needed. Completion order and host durations are
diagnostics and never enter record or failure identities. `batch.json` is
published only after controlled shutdown and contains selection, counts,
stopping reason, hashes of `runs.jsonl`, and references to distinct failures.

Successes do not retain a target or output artifact. Each distinct target,
watchdog, or Runner/host failure retains the first complete representative
artifact; later matches add result lines referring to its complete failure
signature. This bounds retained failure data by distinct observations rather
than selected seeds.

### Manifest fields

`manifest.json` has `schema_version: 1` and these required fields:

| Object | Required fields |
| --- | --- |
| root | `schema_version`, `artifact_kind`, `record_hash`, `created_at`, `batch_id`, `selection_ordinal`, `seed`, `replay_mode` |
| runner | `record_contract`, `runner_build`, `host_os`, `host_arch` |
| toolchain | `go_version`, `build_key`, `target_goos`, `target_goarch` |
| target | `kind`, `source`, `file`, `sha256`, `size`, `argv`, `build_tags`, `build_info` |
| environment | sorted array of `{name,value}` |
| limits | decimal-string nanoseconds for run/overall/grace, output bytes, World event bytes |
| world | initial snapshot schema/file/raw hash/semantic digest, transition schema/file/raw hash/count/transcript digest, final snapshot schema/file/raw hash/semantic digest |
| outcome | `domain`, `reason`, `termination`, `exit_code`, `signal`, `deadline`, `failure_signature`, `replay_match` |
| streams | stdout and stderr file, retained hash, full hash, total/retained/discarded bytes, truncation |
| files | sorted array of relative path, mode, size, and raw SHA-256 for every payload except `manifest.json` |
| host | start/end RFC3339Nano times and decimal-string elapsed nanoseconds |

Seeds, sizes, counts, exit codes, and nanoseconds are decimal strings in the
wire schema. This avoids JSON number precision loss and keeps canonical parsing
independent of machine integer width. Fields that do not apply use JSON `null`;
they are not omitted. Paths are slash-separated relative paths with no empty,
`.` or `..` segment.

`artifact_kind` is `gomadv3.target-failure/v1`,
`gomadv3.watchdog-timeout/v1`, or `gomadv3.runner-failure/v1`.
`replay_mode` is `exact` for complete target failures, `diagnostic` for a
cleanly contained watchdog timeout, and `none` for Runner/host failures or
incomplete records. Diagnostic replay invokes the exact seed and timeout but
does not claim that host elapsed time is deterministic.

### Canonical JSON and hashing

Canonical JSON is defined locally rather than by a third-party library:

- UTF-8, no BOM, no insignificant whitespace, and no trailing newline;
- object keys sorted by raw UTF-8 byte order;
- arrays retain semantic order;
- strings use JSON escaping with HTML escaping disabled;
- version discriminators are bounded JSON integers; seeds, IDs, sizes, counts,
  exit codes, and nanoseconds are validated decimal strings; floating-point
  values are forbidden in record projections;
- duplicate object keys, invalid UTF-8, unknown manifest fields, and trailing
  JSON tokens are rejected.

Canonical JSONL is one canonical JSON object followed by `\n` per record. The
empty transition transcript is zero bytes. Readers validate duplicate keys
before decoding with `json.Decoder.DisallowUnknownFields`.

Raw payload hashes are `SHA-256(file bytes)`, rendered as
`sha256:` plus 64 lowercase hexadecimal digits. Record-owned structured
identities are:

```text
record hash       = SHA-256("gomadv3-run-record-v1\0" || canonical record projection)
failure signature = SHA-256("gomadv3-failure-signature-v1\0" || canonical failure projection)
```

World state and transcript digests are computed by World over its own
versioned canonical binary encoding. Record never recomputes those semantic
digests. It stores them alongside raw SHA-256 hashes of the serialized payload
files, so corruption of either the wire bytes or their claimed meaning is
detectable without giving Record knowledge of World state transitions.

The record projection contains schema, toolchain, target hash and argv,
environment, limits, seed, World schemas, semantic digests, raw payload hashes,
outcome termination, and full and retained stream hashes. It excludes
`record_hash`, `failure_signature`,
`created_at`, `batch_id`, host timestamps/durations, selection ordinal, and
artifact paths.

The failure projection contains schema, toolchain build key, target OS/arch and
target hash, argv, environment, initial and final World semantic digests,
transition transcript digest and raw hashes, outcome
domain/reason/termination, and full stdout/stderr hashes. It excludes seed,
selection ordinal, host time, directory names, and the record hash.
Deduplication is therefore conservative: two failures group only when their
complete observable deterministic records match. A watchdog signature is
diagnostic and does not become a deterministic target identity.

### World payloads

Before World is connected, Runner writes canonical JSON `null` as both initial
and final snapshots, labels them `gomadv3.world.snapshot/none`, and writes a
zero-byte transcript labeled `gomadv3.world.transitions/none`. These are real,
hashable payloads rather than absent fields.

The first connected World supplies `gomadv3.world.snapshot/v1` snapshots,
`gomadv3.world.transitions/v1` transition JSONL, state digests, a rolling
transcript digest, and transition count through its typed API. Record specifies
only the outer envelope and file hashing; it does not redefine request IDs,
resource IDs, payloads, cancellation, quiescence, sequence origin, or logical
time. The composition layer rejects a semantic digest or count that disagrees
with World's validator before publication or replay.

World transition recording is lossless or fails deterministically. Before a
transition that would exceed the configured World limit affects target-visible
state, World returns its stable capacity failure. Runner never truncates the
transcript and marks an artifact incomplete if the World producer closes
without a validated final snapshot and transcript digest.

## Atomic publication and partial diagnostics

All directories use mode `0700`; manifest and data files use `0600`; the stored
target uses `0700`. Runner rejects symlinks, devices, sockets, hard links with
unexpected link count, and paths escaping the artifact root.

Publishing a failure uses a staging directory in the destination parent:

1. create a cryptographically random `.publish-*` directory;
2. copy the already prepared target and finalized stream/World files while
   hashing them;
3. fsync every payload file;
4. construct, canonicalize, and write `manifest.json` last;
5. fsync the staging directory;
6. rename it to the final signature directory without overwriting an existing
   artifact;
7. fsync `failures/`.

An existing signature directory is reused only after its manifest and every
payload hash validate and its complete failure signature matches. A mismatch is
a host integrity failure; Runner never replaces it.

Per-run staging begins before process start and keeps bounded partial output
plus an atomically replaced `partial.json`. Controlled launch, deadline,
capture, containment, or publication failures are promoted to host-failure
artifacts when sufficient metadata exists. Abrupt Runner termination leaves the
`.partial` directory intact and no `manifest.json`, so readers cannot mistake it
for a complete record. Later invocations do not delete partial diagnostics.

The supervisor's liveness pipe and independent deadline contain active targets
when Runner exits normally, panics, or loses its control path. `SIGKILL` of both
Runner and supervisor can still leave OS-owned cleanup and a partial directory;
no filesystem protocol can publish a complete record for work whose final wait
status was never observed.

## Seed replay

`gomad replay` performs preflight before executing anything:

1. open the artifact directory without following symlinks;
2. parse one supported manifest with duplicate/unknown-field rejection;
3. require `replay_mode` to be `exact` or `diagnostic`;
4. verify every listed payload's type, mode, size, and raw hash and reject
   unlisted files other than documented host metadata;
5. validate World's semantic digests through World when connected, then
   recompute raw payload hashes, record hash, and failure signature;
6. require current host OS/architecture, Go version, and custom toolchain build
   key to match exactly;
7. validate the stored target with `debug/buildinfo` and its content hash;
8. reconstruct the recorded argv, allowlisted environment, limits, initial
   snapshot, and exactly one recorded seed.

Replay runs the stored target; it does not invoke `go build`, `go test`, or the
original source path. `--verify-only` stops after preflight. Normal exact replay
computes a new record projection and succeeds as reproduction only when the
failure signature, target termination, full stdout/stderr hashes, and World
state/transcript digests equal the artifact. Diagnostic watchdog replay uses
the recorded host timeout and reports whether the same timeout class and
bounded observations recur without promoting that result to a deterministic
guarantee. The command prints the first field-level mismatch on divergence and
returns status 1. It never substitutes the local target or attempts schema
migration.

External-event replay will later use the recorded World transition plan as an
input and report the first incompatible request or result. Seed replay records
and compares transitions but does not force them. Runtime-choice replay remains
outside this contract.

## Error handling and failure classification

### Target domain

A target outcome exists only after successful start, capture, wait, and process
group cleanup. Exit 0 is success. Otherwise the outcome domain is `target` and
termination is `exit` or `signal`. Stable informational reasons may refine it:

- `nonzero_exit` for an ordinary target or test failure;
- `panic_or_runtime_fatal` for exact Go panic/runtime fatal prefixes;
- `deterministic_deadlock` for the exact runtime deadlock diagnostic;
- `logical_test_timeout` for the standard generated-test timeout diagnostic;
- `unsupported_deterministic_mode` for the stable cgo/external-link rejection;
- `world_failure` for a structured World terminal result; and
- `external_signal` when Runner did not initiate the signal.

Reason refinement never changes domain, exit code, signal, or raw output hashes.
Unrecognized output remains `nonzero_exit`; Runner does not depend on fragile
diagnostic parsing for correctness.

### Watchdog domain

A per-run deadline that expires before a target outcome exists is
`watchdog_timeout`. If termination, pipe draining, wait, and process-group
cleanup all complete, the run is a bounded observation: it receives a
diagnostic failure signature, counts toward `first` or `budget`, and does not
poison other seeds. Its artifact records `replay_mode: diagnostic`. Any failure
in cleanup promotes the result to the Runner/host domain.

### Runner/host domain

These failures prevent a trustworthy completed-run result and return CLI
status 3:

- invalid or missing toolchain and target preparation/build failure;
- overall deadline before controlled batch completion;
- spawn, supervisor protocol, pipe, capture, signal, reap, or group-cleanup
  failure;
- prepared-target mutation or build-info mismatch;
- World record corruption, premature close, or capacity protocol violation;
- artifact hashing, fsync, rename, collision-validation, or disk-capacity
  failure; and
- Runner invariant violation.

Overall cancellation caused by `first` or `budget` is `runner_cancelled`, not
another target failure. It preserves partial diagnostics but does not consume
the failure budget or produce a complete run artifact. A Runner/host failure
stops new dispatch immediately and cancels all active targets because
subsequent results cannot be claimed trustworthy.

## Implementation phases

### Phase 1: Record schema and pure validation

- Add canonical JSON/JSONL, typed manifest fields, domain-separated hashing,
  seed/range parsing, failure projections, and strict readers.
- Define the `none` World payloads and the v1 World envelope identities.
- Add golden encoding, malformed-input, duplicate-key, path traversal, integer
  boundary, and hash-domain tests.

Exit when the same typed record produces identical bytes/hashes across fresh
processes and every malformed or unsupported form fails closed.

### Phase 2: Preparation, process supervision, and bounded capture

- Add exec/go-run/go-test preparation and immutable target hashing.
- Add the Unix supervisor, liveness/control protocol, process-group lifecycle,
  deadlines, wait-status decoding, and bounded full-hash output capture.
- Keep target activation as direct `GOMADSEED`; do not run `cmd/go` in Gomad.

Exit when one prepared target runs many isolated seeds, timeouts leave no group
members, Runner death closes the liveness path, and output floods remain bounded.

### Phase 3: Exploration, classification, and artifacts

- Add bounded workers, lazy seed dispatch, stop policies, failure signatures,
  coordinator-owned deduplication, partial records, atomic publication,
  `runs.jsonl`, and final batch summaries.
- Add root Make targets to build the Runner and exercise representative direct,
  go-run, and go-test paths.

Exit when a mixed success/failure range publishes exactly one complete artifact
per distinct signature and a simulated crash exposes only explicit partials.

### Phase 4: Strict seed replay

- Add artifact opening, complete identity preflight, exact one-seed execution,
  result comparison, `--verify-only`, and divergence diagnostics.
- Test artifacts moved between directories and reject any changed target,
  payload, environment, toolchain key, architecture, or schema.

Exit when known target failures reproduce from stored target bytes without
rebuilding, watchdog artifacts can be rerun without a deterministic claim, and
every identity mismatch prevents target start.

### Phase 5: World connection behind the stable contract

- Let World supply a typed initial snapshot, lossless transition transcript,
  final snapshot, and semantic digests through the composition layer.
- Preserve native timer ownership and add no runtime coordination hook until a
  concrete adapter demonstrates the need described in `GOMADv3_CLOCK.md`.
- Keep World semantic tests independent of Runner, then add cross-module record
  round trips.

External-event replay follows this phase as a separate implementation. Delta
minimization follows only after replay candidates can be proved reproducible;
it minimizes World snapshot/transition/fault data, test selection, and arguments,
never the numeric seed.

## Test matrix

| Area | Required cases | Required result |
| --- | --- | --- |
| Seed selection | zero, max uint64, sets, ranges, overlap, reverse, empty, signs, overflow | exact lazy sequence or stable configuration error |
| Preparation | exec snapshot, go-run, go-test, build failure, mutation | one immutable target; `test_dep`; host failure on drift |
| Activation | seed zero/max, parent seed present, reserved env | only target receives exact `GOMADSEED` |
| Isolation | globals, goroutines, FDs, cwd, two same-seed processes | fresh process state and repeatable supported output |
| Parallelism | 1, default, configured maximum, more seeds than workers | active targets never exceed limit |
| Per-run deadline | exit before deadline, busy loop, blocking I/O | result or complete group termination |
| Overall deadline | preparation, queued seeds, active seeds, publication | bounded cancellation and host classification |
| Process tree | child/grandchild, TERM ignore, signal exit, Runner control EOF | TERM/KILL escalation, reap, no remaining group |
| Output | empty, exact limit, over limit, binary bytes, no newline, flood | full hash, exact head/marker/tail, constant memory, no block |
| Environment | empty, sorted explicit entries, duplicate/reserved/invalid | exact recorded environment or early rejection |
| Classification | success, exit, panic, deadlock, logical timeout, watchdog timeout, Runner failure | correct domain, termination, and stable reason |
| Stop policy | first, budget, all, simultaneous results, duplicates | documented dispatch/cancel/drain behavior |
| Signature | same failure across seeds, changed stream/World/outcome | stable grouping only for complete matching projection |
| Artifact | success, collision, disk full, fsync/rename failure, crash points | atomic complete directory or explicit partial only |
| Canonical data | key order, escapes, unknown/duplicate keys, JSONL truncation | one encoding; malformed data rejected |
| Replay | moved artifact, exact match, changed file/toolchain/arch/env/argv | rebuild-free match or pre-execution rejection |
| World seam | none payload, snapshots, transitions, semantic digests, record limit | lossless data with one semantic owner |
| Scale | 10x seeds, failures, output floods | linear work, bounded per-run memory, bounded workers |
| Unsupported | Windows, race, cgo, external link, plugin | stable configuration or target rejection |

The dependency-free module uses `testing` fatal checks rather than adding an
assertion library. Timing tests synchronize through channels, contexts, and
process watchdogs rather than `time.Sleep`. Process tests use helper
subprocesses rather than making the test process a signal target.

## Performance, scalability, complexity, and security

Target preparation is paid once. Each seed adds one supervisor/target startup,
execution, streaming SHA-256 work, and a small canonical result line. CPU work
is approximately linear in seeds and output bytes. Retained memory is
`O(parallel * (output limit + protocol state))`; the seed iterator and batch
writer are constant-memory.

One target still uses one P. Increasing seed count by 10x keeps target
parallelism fixed and increases elapsed work roughly 10x divided by available
workers. Output is drained and summarized incrementally; successful runs do not
retain binaries or streams. Failure-heavy runs can grow with the number of
distinct failures, which is intentional evidence rather than a memory leak.

The deep Runner interface hides concurrency, budgets, deadlines, and
classification. Process concentrates Unix containment and output capture.
Record concentrates schema and identity. Artifact concentrates crash-safe
storage. World semantics do not leak into these implementations. Removing any
one module would spread its complexity across the others, so each seam earns
its interface.

The supervisor and exact schema are the main complexity costs; runtime-choice
tracing, content-addressed storage, and migration remain excluded. Security is
fail-closed but not sandboxing: argv is never shell-evaluated, target environment
starts empty, private artifacts may contain sensitive data, and replay rejects
symlinks, traversal, unlisted files, hash drift, and executable mutation.
World should store hashes/lengths instead of secrets unless its schema explicitly
requires raw payloads. Deterministic runtime randomness remains test-only.

## Crash recovery and failure modes

- Runner cancellation or panic closes supervisor liveness pipes; supervisors
  terminate and reap their target groups.
- An overall deadline is independently known to supervisors, so a stuck
  coordinator cannot extend target life.
- Abrupt death leaves bounded partial diagnostics and never a complete manifest.
- Disk-full during capture retains the already bounded in-memory head/tail,
  cancels the target, and reports a host failure to stderr even if artifact
  promotion also fails.
- Disk-full or rename failure during publication leaves `.publish-*` data that
  is not a complete artifact; it is not automatically deleted.
- A target that closes stdout/stderr early is still waited and classified by
  its process status.
- A target that keeps descendant pipe writers open is ended by the process-group
  deadline; capture EOF is required before a target result is complete.
- A supervisor protocol failure causes direct best-effort group kill using the
  last acknowledged PGID, then stops the batch as untrustworthy.
- An overall Runner/host failure cancels queued and active work. No later success may
  downgrade it.

## World and clock integration boundaries

Runner does not advance logical time, order World transitions, allocate
resource identities, or consume the World random stream. It supplies the seed,
records World's typed snapshots and ordered transitions, and enforces storage
limits. World remains the sole owner of external resource state and events.

The World design must honor these cross-document contracts:

- own the snapshot, transition, and semantic-digest schemas; Record owns only
  the outer envelope and raw file hashes;
- assign stable request, event, resource, and transition identities without
  pointers, goroutine IDs, host time, host readiness order, or map iteration;
- use absolute process virtual nanoseconds for logical time;
- keep World choice randomness domain-separated from the runtime stream;
- make transcript capacity failure occur before an unrecorded transition affects
  target-visible state;
- emit a final snapshot or a stable explicit no-final-state result;
- never add a second native Go timer queue; and
- when an adapter proves coordination necessary, compare the earliest native
  timer and World event at runtime-proven quiescence and make every event at the
  selected instant eligible before scheduling resumes.

Sharing an artifact envelope does not merge modules: Record knows bytes and
raw hashes, World knows meaning and semantic digests, Runner knows execution
and persistence, and the runtime clock knows native timer quiescence.

## Verification commands

Every Go test includes `test_dep`. Run focused checks first, then the existing
toolchain and repository standards:

```sh
tools/gomadv3/.toolchain/bin/go -C tools/gomadv3 test -tags test_dep ./internal/record ./internal/artifact
tools/gomadv3/.toolchain/bin/go -C tools/gomadv3 test -tags test_dep ./internal/process ./internal/target ./internal/runner ./internal/replay
tools/gomadv3/.toolchain/bin/go -C tools/gomadv3 test -tags test_dep ./...
make -C tools/gomadv3 test-harness
make -C tools/gomadv3 test
make fmt-imports
make lint-code
```

Black-box verification additionally explores a bounded clock fixture range,
replays a stored deterministic failure, floods both output streams, times out a
descendant tree, interrupts Runner during every publication stage, and confirms
that no target or complete artifact survives incorrectly.

## Completion criteria

- Exec, go-run, and one-package go-test targets are prepared once and each seed
  runs in a fresh isolated process.
- Per-run and overall host deadlines terminate and reap complete target process
  groups, including when Runner loses its control path.
- Stdout and stderr are always drained, fully hashed, and retained within the
  configured memory and artifact bounds.
- Target, watchdog, and Runner/host failures are never conflated, and CLI status reflects the
  strongest observed domain.
- Each distinct failure signature has one atomic, self-contained, versioned
  artifact; incomplete work is visibly partial and has `replay_mode: none`.
- Manifest, target, stream, and World hashes validate before replay starts.
- Seed replay uses the stored target and exactly one recorded seed and reports
  an exact match or first divergence without rebuilding or compatibility
  fallback.
- Ten times as many seeds preserves fixed worker/output memory and increases
  work approximately linearly without leaking processes.
- Unsupported Windows, cgo, external-link, plugin, race-detector, and foreign-
  thread configurations fail explicitly.
- Focused tests, the existing Gomad v3 harness, formatting, and `make lint-code`
  pass without adding a third-party dependency.
