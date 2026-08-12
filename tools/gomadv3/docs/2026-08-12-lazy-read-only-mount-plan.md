# Plan: Lazy read-only mounts for Gomad v3

## Context

Eighty-one root Temporal functional suites currently stop because the Gomad
target runs from an isolated working directory and cannot read the SQLite schema
tree. Implement the approved lazy read-through mount design entirely in Gomad,
starting with the unchanged `TestActivityAPIBatchSecurityTestSuite`.

The target may observe only explicitly mounted paths. Record mode captures each
entry on first access; replay serves only captured entries without reopening the
host tree. The existing batch-cancel profile and artifacts remain compatible.

## Pattern survey

### Analogous features

- `internal/process` already transports bounded data through reserved descriptors,
  keeps descriptor ownership explicit across Runner, supervisor, bootstrap, and
  target, and enforces one absolute process deadline.
- `internal/ioprofile` already binds configuration into profile inventory,
  bootstrap identity, deterministic transcripts, and exact replay.
- `internal/artifact` already publishes immutable payloads through private staging,
  canonical manifests, content hashes, and atomic no-replace publication.
- `overlay/src/os/gomad.go` already owns the virtual filesystem namespace and
  fail-closed path normalization used by profile-enabled targets.

### Reusable utilities

- Use `os.OpenRoot` for host-root pinning and descendant lookup; it provides the
  standard library's symlink-safe root boundary on the supported platform.
- Use `record.CanonicalJSON`, `record.StrictDecode`, `record.HashBytes`, and
  `artifact.ReadPayload` for persisted mount metadata and replay validation.
- Extend `process.Request`/`process.Result` and the existing descriptor plumbing
  instead of creating a second process launcher.
- Extend `record.IOProfile`, `artifact.Input`, and the existing I/O artifact
  directory instead of adding an unrelated artifact subsystem.

### Convention anchors

- New public configuration is validated before target preparation and converted
  into canonical internal values once.
- Every allocation and stream is bounded before data is read.
- Host/infrastructure failures remain distinct from target-visible filesystem
  errors and exact-replay divergences.
- Tests use real processes and the patched toolchain for overlay behavior; unit
  tests isolate framing, mount lookup, limits, and artifact validation.

### Proposed alignment

Create `internal/romount` as the deep module owning mount parsing, secure host
capture, replay lookup, canonical captured state, limits, and broker service.
Keep the standard-library-side client in `internal/gomadio`/`os`, and keep process
descriptor lifecycle in `internal/process`.

## Implementation steps

1. **Pin the public mount contract with failing tests**
   - Add Runner/CLI tests for repeatable `--io-ro-mount HOST=TARGET` mappings in
     `cmd/gomad/main_test.go`, `internal/runner/runner_test.go`, and
     `internal/ioprofile/profile_test.go`.
   - Require an I/O profile; normalize the host source relative to the Runner
     working directory and the target destination into the virtual absolute
     namespace; reject empty, duplicate, nested, and overlapping destinations.
   - Add explicit mount limits to `runner.Config`, coordinator wire state, process
     requests, run records, and replay validation.

2. **Implement and test the isolated host capture module**
   - Add `internal/romount` with `Prepare`, `Serve`, `Captured`, and `Replay`
     interfaces so callers do not manage traversal or cache details.
   - Pin each host directory with `os.OpenRoot`; reject a non-directory root and
     resolve descendants only through the pinned root.
   - Capture regular files with metadata-before/read/metadata-after validation;
     capture sorted complete directory listings; reject symlinks, hard-linked
     regular files, and special entries.
   - Cache immutable entries by normalized target path and enforce path, request,
     file-count, directory-entry, single-file-byte, and aggregate-byte limits
     before allocation or target observation.
   - Unit-test regular/empty files, directories, concurrent duplicate requests,
     traversal, symlinks, hard links, special files, mutation, every capacity
     boundary, deterministic ordering, and clean shutdown.

3. **Add the bounded target/broker channel**
   - Extend `internal/process/process_unix.go` and `bootstrap_unix.go` with a
     full-duplex broker descriptor pair whose ownership and closure mirror the
     existing I/O transcript descriptors.
   - Add a small fixed-header, versioned, length-bounded protocol in
     `internal/romount/wire.go`; the overlaid standard library cannot depend on
     Temporal's protobuf runtime, so this channel contains only request ordinal,
     operation, normalized path, status, metadata, and bounded content bytes.
   - Run the broker inside the trusted Runner process, terminate it within the
     same absolute deadline, propagate protocol/capture failures as structured
     host failures, and return canonical captured state in `process.Result`.
   - Test malformed frames, oversized lengths, bad ordinals, premature EOF,
     broker failure, target exit with an open channel, cancellation, and descriptor
     absence when mounts are disabled.

4. **Serve mounted files through the overlaid standard library**
   - Extend `overlay/src/internal/gomadio` with the mount client, immutable entry
     cache, and per-open in-memory handle state.
   - Extend `overlay/src/os/gomad.go` plus the minimal `go1.26.4.patch` hooks so
     `OpenFile`, `Read`, `ReadAt`, `Seek`, `Close`, `Stat`, file `Stat`, `ReadDir`,
     `Readdir`, and `Readdirnames` preserve normal `os.File` semantics for mounted
     entries.
   - Keep the existing virtual writable-directory behavior outside mounts;
     reject every mutating path or handle operation within a mount with `EROFS`.
   - Record mount lookup/read/write-rejection operations through the existing
     bounded transcript before returning their results.
   - Add a focused toolchain fixture under `testdata` proving reads never create
     host state, cached reads survive host mutation, directory order is stable,
     undeclared paths fail closed, and writes return `EROFS`.

5. **Persist captured inputs and make replay host-independent**
   - Extend `record.IOProfile` with canonical target-side mount mappings, captured
     state identity, limits, and an optional `io/mounts.json` descriptor; never
     place host source paths in semantic replay/failure projections.
   - Extend `artifact.Store` to write canonical mount metadata and content-addressed
     observed file payloads under `io/mounts/`, validate every digest/size, and
     include them in `Manifest.Files`.
   - Extend `artifact.Open` and `replay` preflight to validate the observed set and
     construct `romount.Replay` without resolving or opening original host roots.
   - Make any replay lookup absent from the captured set a precise I/O replay
     divergence and retain the existing transcript ordinal.
   - Add an exact failure-artifact test that records, removes the host tree,
     replays successfully, verifies payload corruption is rejected, and verifies
     an unrecorded lookup diverges.

6. **Qualify the unchanged Temporal suite**
   - Add a new exact profile identity for
     `TestActivityAPIBatchSecurityTestSuite`, reusing the shared I/O implementation
     while preserving the batch-cancel profile identity.
   - Run the suite unchanged with
     `--io-ro-mount ./schema/sqlite/v3=go.temporal.io/server/schema/sqlite/v3` in
     two fresh seed-7 processes and require exit zero plus byte-identical
     transcripts.
   - Run exact replay for a deterministic failure fixture, update the CLI/README,
     and update the
     [functional-suite sweep](2026-08-11-functional-suite-sweep.md) only from
     observed results.

7. **Regression and standards verification**
   - Run focused `romount`, process, artifact, record, replay, runner, and I/O
     profile tests with `-tags test_dep`.
   - Run `make -C tools/gomadv3 runner-test`, `make -C tools/gomadv3 world-test`,
     the full `tools/gomadv3/test.sh`, formatting, and relevant lint checks.
   - Confirm `git diff -- tests` is empty and rerun representative profile-disabled
     standard-library tests to prove upstream behavior is unchanged.

## Error handling and failure modes

- Configuration and root-validation failures stop before target execution.
- `ENOENT`, `EROFS`, and unsupported-entry errors are deterministic target-visible
  filesystem results and are transcripted.
- Mutation during capture, capacity exhaustion, malformed protocol data, or
  broker termination are structured host failures and cannot publish an exact
  artifact with incomplete captured input.
- Target or Runner crashes retain existing partial diagnostics; publication stays
  atomic and occurs only after the broker returns a complete canonical state.
- Replay never falls back to a live host root.

## Trade-offs

- First access pays one broker round trip and copies the complete observed file;
  subsequent operations are in-memory. This favors deterministic replay over
  streaming very large files, which are rejected by reviewed limits.
- Memory and artifact size scale with unique observed bytes, not total mount size.
  A tenfold read load over the same files mostly increases transcript requests;
  a tenfold unique dataset reaches explicit capacity limits instead of exhausting
  the host.
- Keeping the broker interface narrow adds overlay hooks but prevents host paths,
  descriptors, and capture races from leaking into Runner and replay callers.

## Context files

- `tools/gomadv3/docs/2026-08-12-lazy-read-only-mount-design.md`
- `tools/gomadv3/internal/process/process.go`
- `tools/gomadv3/internal/process/process_unix.go`
- `tools/gomadv3/internal/process/bootstrap_unix.go`
- `tools/gomadv3/internal/ioprofile/profile.go`
- `tools/gomadv3/internal/ioprofile/bootstrap.go`
- `tools/gomadv3/internal/artifact/store.go`
- `tools/gomadv3/internal/record/types.go`
- `tools/gomadv3/internal/replay/replay.go`
- `tools/gomadv3/internal/runner/runner.go`
- `tools/gomadv3/overlay/src/os/gomad.go`
- `tools/gomadv3/overlay/src/internal/gomadio/gomadio.go`
- `tools/gomadv3/go1.26.4.patch`
