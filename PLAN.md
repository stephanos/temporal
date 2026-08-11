---
status: implemented
scope: tools/gomadv3
target: tests/activity_api_batch_cancel_test.go
---

# Plan: Transparently run the unchanged activity batch-cancel test under Gomad

## Implementation Status

Implemented under `tools/gomadv3` without changing Temporal source or tests. The final `darwin/arm64` qualification passed 20 fresh seed-7 processes with one identical transcript digest, then passed the complete seed 0–31 matrix. The deterministic fail-closed fixture publishes a transcript-bearing artifact and replays it exactly, including operation-by-operation transcript validation.

## Objective

Run the existing `TestActivityAPIBatchCancelClientTestSuite` as a deterministic-system test without changing the test, its shared helpers, `tests/testcore`, or Temporal production code. For the pinned target closure, Gomad must inventory every semantic application-I/O entry point reached by the selected suite and replace it with an in-memory deterministic alternative before package initialization. Unsupported operations entering an inventoried standard-library or generated-overlay boundary fail closed before host dispatch. Bounded Runner-owned bootstrap, transcript, stdout, and stderr channels remain explicitly allowlisted non-semantic host I/O.

This is a target-specific pilot, not a general-purpose virtual operating system or a security boundary against hostile code. The initial profile supports `darwin/arm64` because its first generated `modernc.org/sqlite` overlay and I/O inventory are pinned to that platform's generated dependency sources. Other platforms require their own reviewed inventory and overlay hashes.

Definition of done:

- all repository changes, other than this plan, are under `tools/gomadv3`;
- `tests/activity_api_batch_cancel_test.go` and the rest of Temporal remain byte-for-byte unchanged;
- the Runner accepts only the exact package and exact full-suite selector for this profile;
- loopback TCP, required filesystem state, hostname, entropy, and SQLite VFS time/entropy are Gomad-owned;
- profile activation is nonambient and Runner-mediated, and every inventoried operation is redirected or rejected before host dispatch;
- the existing `require.Eventually` polling progresses through ordinary runnable goroutines and Gomad native timers;
- one schedule seed repeats in fresh processes with the same result and I/O transcript digest;
- changing the schedule seed does not change the profile's fixed entropy stream; and
- a deterministic Gomad fixture failure publishes and replays exactly.

## Architecture Decision

Use a versioned Runner profile, `temporal-activity-api-batch-cancel/v1`, that composes three Gomad-owned layers:

```text
gomad explore --io-profile temporal-activity-api-batch-cancel/v1
        |
        | exact target/argv validation
        | prepared-binary identity + trusted inherited descriptor
        v
patched target, configured before package init
        |
        +-- standard-library shims
        |     net: in-memory loopback TCP
        |     os: minimal in-memory filesystem + fixed hostname
        |     crypto/rand: fixed profile entropy stream
        |
        +-- target-build overlay
        |     exact modernc SQLite raw time/entropy sites -> Gomad hooks
        |
        +-- bounded I/O transcript
              ordered canonical records -> inherited shared memory
              terminal digest -> Runner record pipe
```

`GOMADSEED` remains solely the schedule-choice seed. The profile version fixes a separate entropy key and all other deterministic constants; there is no public I/O-seed flag. Reads from the entropy stream are serialized, while the schedule determines which concurrent consumer reads next.

Profile activation is not an environment-variable capability. The Runner writes a canonical, versioned configuration frame to a dedicated inherited descriptor during bootstrap. The frame binds the profile to the prepared target identity, canonical argv, Runner identity, architecture, profile implementation digest, and canonical I/O-inventory digest. Runtime initialization consumes and closes it before package initialization. Missing, malformed, repeated, stale, or mismatched frames fail closed. A package-private test seam activates a separate `gomadv3-io-fixture/v1` profile for Gomad fixtures; the public CLI cannot select it. The threat model is trusted test code and accidental activation, not a malicious target forging inherited descriptors: the capability proves Runner-mediated launch within that model, not cryptographic provenance against code executing as the same user.

The standard-library shims provide normal in-memory behavior. A narrowly generated Go build overlay redirects the exact `modernc.org/sqlite` VFS calls that otherwise obtain time or randomness through `modernc.org/libc` raw syscalls. The overlay generator validates the pinned dependency version, input hashes, and exact rewrite anchors, writes only temporary build inputs, and aborts on drift. Its inputs and output digest are part of the prepared-target identity. No module-cache, vendored, Temporal, or test source is edited.

The enforceable boundary is the set of entry points controlled by the patched toolchain and generated target overlay. Target preparation pins the dependency closure, verifies the profile's classified I/O-site inventory, and rejects source/hash drift or a known unclassified site. Qualification assumes trusted target code reaches no direct host syscall outside that reviewed inventory. Replay proves modeled I/O behavior; it does not claim to detect a newly introduced direct-syscall bypass until the inventory is updated.

## Supported Boundary

The public profile is valid only when all of these are true:

- target kind is `go-test`;
- built package is exactly `go.temporal.io/server/tests`;
- target argv is exactly the one-element vector `[-test.run=^TestActivityAPIBatchCancelClientTestSuite$]`, supplied canonically by the Runner rather than augmented by the target;
- user environment additions and package-defined configuration flags are rejected; the profile validator allowlists the complete Runner-owned environment needed by this target;
- the pinned Go toolchain and `darwin/arm64` overlay inventory match the profile; and
- cgo, plugins, external linking, and unsupported persistence configuration are absent.

The initial modeled operations are:

- `net.Listen`, `net.ListenTCP`, `ListenConfig.Listen`, `net.Dial`, `net.DialTCP`, and `Dialer.DialContext` for `tcp`/`tcp4` on literal loopback or `localhost`;
- deterministic sequential `:0` leases and the concrete `*net.TCPListener`, `*net.TCPConn`, and `*net.TCPAddr` behavior reached by Temporal, gRPC, pprof, and `common/testing/freeport`;
- reliable ordered full-duplex streams, accept/dial cancellation, deadlines, close, half-close, and address reporting;
- `crypto/rand.Reader` and `crypto/rand.Read` using a profile-fixed deterministic stream;
- a fixed hostname (`gomad-host`) and the exact directory creation/metadata behavior reached by `.testoutput` setup;
- the existing SQLite `mode=memory` state, with its VFS time and entropy inputs redirected into Gomad; and
- bounded stdout/stderr plus trusted bootstrap/record descriptors as non-semantic host channels.

DNS, non-loopback sockets, packet/Unix sockets, arbitrary files, external databases, subprocesses, plugins, cgo, external linking, and signal-dependent application behavior are unsupported. `SyscallConn` and any other escape hatch either receive explicitly modeled behavior proven necessary by the target or return a stable unsupported-boundary error; they never expose a host descriptor.

## Implementation Plan

### 1. Establish the selected suite's I/O inventory

- Add a minimized unchanged SQLite fixture under `tools/gomadv3/testdata` using the same `modernc` in-memory configuration as the target.
- Build a non-mutating source/call-site inventory rooted at the selected suite and package initializers across the pinned standard library and module dependency closure. Classify every reached `net`, `os`, entropy, process, signal, `syscall`, `x/sys`, and `modernc.org/libc` I/O site as standard-library shimmed, target-overlay redirected, explicitly allowlisted Runner-owned non-semantic channel, or unreachable under the exact target/profile configuration.
- Inspect the pinned `modernc.org/sqlite`/`modernc.org/libc` sources for open, schema setup, query, transaction, and close. Confirm the known `/dev/urandom` and wall-clock paths and identify every additional VFS file, descriptor, time, or entropy site requiring redirection.
- Persist exact module versions, source hashes, classified call sites, reserved Runner descriptor roles, and their disposition in a canonical versioned Gomad-owned profile inventory. Hash its exact canonical bytes into the prepared-target identity. Target preparation rejects a missing classification, changed hash, newly resolved direct-I/O symbol where the linker exposes one, or an unsupported platform before execution.
- Add positive fixtures only after their corresponding shim/overlay exists. Negative fixtures must fail target preparation or exercise an already-intercepted unsupported boundary; do not execute an uncontained raw host call as a canary.
- Do not expand the modeled boundary without a positive fixture, a safe fail-closed negative fixture, and inventory/replay coverage.
- Gate: the inventory, canonical encoding/digest, source evidence, and planned disposition for every reached site are reviewed before implementation; behavioral proof follows in the step that implements each disposition.

### 2. Add the target-scoped profile and trusted bootstrap frame

- Add `--io-profile` to the Gomad CLI and Runner configuration. Accept only the public profile above; do not add `--io-seed`.
- Implement a single validator for target kind, built package, exact one-element argv, exact Runner-owned environment, platform, toolchain, build mode, and persistence assumptions. Reject `--env` for this profile. Reuse the validator for explore and replay so policy cannot drift.
- Extend the existing bootstrap protocol with a bounded canonical I/O configuration frame. Bind it to the prepared-target identity, canonical argv, Runner identity, architecture, profile implementation digest, and canonical inventory digest; pass it on a reserved descriptor, consume it before package init, and reject direct environment activation.
- Store the selected profile, implementation digest, canonical inventory bytes, and inventory digest in the manifest. Restore the trusted frame from validated manifest data during replay rather than trusting recorded child input.
- Add the harness-only fixture profile through a package-private Runner/test seam.
- Tests: exact valid invocation; every additional test argument including `-test.count`, `-test.list`, `-test.short`, `-test.parallel`, benchmark/fuzz flags, and package flags; wrong package/kind/platform/architecture; unknown profile; user environment input; child environment spoofing; malformed/replayed descriptor; identity mismatch; direct execution without the descriptor; coordinator round trip; manifest mismatch.
- Gate: under the trusted-target threat model, only a process launched through the Runner bootstrap can activate the public profile, and only for the exact full suite and environment.

### 3. Admit the minimum toolchain and target-overlay surface

- Extend `tools/gomadv3/test.sh` allowlists only for exact portable standard-library files needed in `net`, `os`, `crypto/rand` (or its internal implementation), runtime, and a small standard-library-internal Gomad package.
- Preserve bans on unrelated, generated, assembly, cgo, platform-specific, test, and binary files. Add validator cases for every admitted path and adjacent/path-traversal rejections.
- Add the minimal standard-library-internal profile state and linkable time/entropy hook declarations required for overlay compilation; their full implementations and transcript state follow in Step 4.
- Add a deterministic target-overlay generator for the pinned `modernc.org/sqlite` source. It validates module version, file hashes, and exact syntax/anchors, redirects only reached VFS entropy/time sites to the declared Gomad hooks, and emits a temporary `go build -overlay` map.
- Include the generator, source expectations, canonical inventory digest, generated overlay digest, standard-library patch, and runtime overlay in immutable build/prepared-target identity. A cache entry created without the profile overlay or with different inventory bytes must not be reusable with it.
- Add a preparation-time closure verifier for the profile inventory. It verifies the pinned dependency/source hashes and every classified site disposition, then rejects drift or known unresolved host-I/O sites rather than launching the child.
- Tests: dependency version/hash drift, missing/duplicate anchors, unrelated source rejection, deterministic overlay output, build-cache separation, and unchanged module-cache hashes.
- Gate: all persistent implementation material remains under `tools/gomadv3`; after Step 4 supplies the hooks, the generated target compiles without editing dependency or Temporal sources. Final behavioral closure is checked after Steps 4–6.

### 4. Implement pre-init policy, fixed entropy, and transcript state

- Centralize read-only profile state in a small standard-library-internal Gomad module initialized from the trusted frame. Disabled and clock-only modes preserve upstream behavior exactly.
- Implement a counter-based SHA-256 stream keyed by a constant derived from the profile name/version and a versioned domain. Define serialization, zero-length reads, partial/large reads, and counter exhaustion. Patch retained `crypto/rand.Reader` and direct `crypto/rand.Read` paths.
- Provide fixed profile values needed by other shims, including hostname, port range, capacities, and path policy.
- Record a bounded ordered transcript in a 64 MiB Runner-created, immediately unlinked shared-memory backing object inherited only by descriptor. Assign ordinals at the modeled adapters' existing linearization points; each record contains its ordinal, operation phase/class, canonical arguments, byte count/content digest, result class, entropy range when applicable, and pre/post adapter-state digests. Store digests rather than payload bytes.
- The target maps the region before package initialization and appends without host readiness or target callbacks. A fixed capacity is part of the profile; overflow is a stable Gomad failure. During replay, a second read-only expected region lets the shim validate each operation immediately and report the first divergent ordinal.
- Patch `os.Exit` to freeze transcript admission and emit exactly one fixed-format terminal frame before its upstream exit path. The controlled Gomad deadlock/unsupported terminal paths invoke the same once-guarded finalizer; ordinary Go test failures are recovered by `testing` and reach `os.Exit`. Unrecovered runtime corruption, external kill, or wall-watchdog termination may omit the frame and is classified as incomplete and non-replayable.
- Keep the terminal frame at or below 512 bytes (`PIPE_BUF`'s portable minimum). Give it a dedicated pipe with no other writers, create it empty, and continuously drain its read end before releasing the child, so its one raw write does not depend on scheduler-visible host readiness. Missing, duplicate, oversized, short, or invalid frames make the run incomplete.
- Tests use custom-toolchain black-box fixtures for retained/direct/concurrent entropy reads, fixed bytes across schedule seeds, counter boundary where reachable, disabled mode, ordered transcript determinism, operation-by-operation replay divergence, capacity exhaustion, and malformed/missing terminal frames. Use an internal test hook only for an otherwise unreachable counter-exhaustion edge.
- Gate: the target cannot reach OS randomness, and every completed run has a validated I/O transcript digest.

### 5. Implement transparent in-memory loopback TCP

- Intercept all reached portable entry points before DNS, socket creation, or netpoll. Canonicalize literal IPv4 loopback and `localhost` without host resolution.
- Build a private listener registry keyed by canonical network/address/port. Allocate `:0` ports sequentially from a fixed range, support close and rebind, reject duplicate binds, and match concrete `TCPAddr` assertions.
- Provide fake-capable branches for `*net.TCPListener` and `*net.TCPConn` so `ListenTCP`, `DialTCP`, deadlines, close, half-close, address methods, and reached concrete methods remain transparent. `SyscallConn`, `File`, packet, multicast, Unix, DNS, and non-loopback paths fail closed unless the target audit proves a modeled behavior is required.
- Use ordinary Go synchronization and Gomad native timers. Never call consumer code while registry locks are held. Bound listeners, pending accepts/dials, connections, and queued bytes with deterministic errors.
- Tests: free-port probe/close/rebind; `localhost`; direct TCP APIs; gRPC-like concurrency; accept/dial cancellation; read/write deadlines; close races; half-close; address assertions; capacity edges; escape-hatch canaries; disabled-mode real loopback.
- Gate: the unchanged services bind and dial their existing endpoints without a socket or host netpoll readiness event.

### 6. Model the reached filesystem and hostname behavior

- Audit the target under fail-closed shims to enumerate `.testoutput`, config, certificate, and other reached `os` paths. Admit only operations required by the unchanged selected suite.
- Implement normalized in-memory directory state for the required create/stat/mkdir-all behavior, including directory/file distinction, permissions, idempotence, and stable `PathError` values. Return `gomad-host` through the reached hostname path.
- Keep stdout/stderr and reserved bootstrap/transcript descriptors explicit, bounded, and recorded as Runner-owned non-semantic inventory entries. Arbitrary open/read/write/remove/rename/temp operations and all other numeric descriptors fail before a host syscall.
- Verify that SQLite remains `mode=memory`, private-cache, and embedded-schema based. Its raw VFS time/entropy is handled by the generated overlay; the closure verifier rejects preparation if any inventoried VFS file/time/entropy site lacks an explicit disposition.
- Tests: one positive fixture per admitted operation; canonicalization/traversal; permissions/errors; hostname; arbitrary-file and descriptor canaries; disabled upstream behavior.
- Gate: every inventory disposition now has behavioral fixture proof, the closure verifier accepts the exact prepared target, and no semantic host filesystem or hostname observation is reachable in the selected suite.

### 7. Prove native polling progress

- Add a Gomad fixture with the same semantic shape as the target's existing `require.Eventually`: an asynchronous condition depends on in-memory connection work while ticker and timeout timers are pending.
- Verify runnable handlers prevent virtual-clock advancement, connection readiness wakes normal goroutines, and the existing timer/quiescence rules decide timeout races. Do not add an application `Await`, a second event loop, or test-specific scheduling hooks.
- Cover connection cancellation versus same-deadline timeout, close versus pending read, and quiescence with no producer. Preserve distinct outcomes for deadlock, unsupported I/O, assertion failure, crash/incomplete record, and wall watchdog.
- Gate: unchanged namespace setup, visibility polling, and batch cancellation polling progress using normal APIs.

### 8. Make replay evidence durable

- Extend the manifest/result schema with the profile implementation digest, canonical inventory bytes/digest, ordered canonical transcript, and its terminal digest/counters. Do not record payload bytes or noncanonical host paths.
- During replay, validate the target, argv, environment, toolchain, profile inventory, target overlay, dependency closure, platform, and profile before constructing the trusted frame. Load the expected transcript into the read-only replay region, fail at the first divergent modeled operation, and finally compare result, bounded streams, transcript length, and terminal digest.
- Use a deterministic fail-closed Gomad fixture for failure/replay qualification; do not add transition injection and do not intentionally fail the Temporal test. If exploration naturally finds a target failure, its normal artifact must also replay.
- Tests: successful transcript equality, deterministic unsupported-operation replay, changed inventory bytes with a stale digest, target/inventory/dependency/overlay/profile mismatch, corrupt/missing transcript, and artifact publication bounds.
- Gate: every target artifact restores the same profile, while the fixture proves exact failing replay.

### 9. Qualify the unchanged test

- Record a checksum for `tests/activity_api_batch_cancel_test.go` and a baseline of `git status --short --untracked-files=all` before implementation. After every generator or repository-level command, reject newly changed paths outside `PLAN.md` and `tools/gomadv3/**`; use tracked-path diffs for the protected application tree rather than hashing the whole repository.
- Build `./tests` once with the pinned prepared target and invoke exactly `-test.run=^TestActivityAPIBatchCancelClientTestSuite$`; do not stage qualification through source edits or subtest selectors.
- Repeat one schedule seed in at least 20 fresh processes, including under host CPU pressure, and compare result, bounded streams, profile/overlay identity, and terminal I/O transcript. Then run a reviewed bounded set of schedule seeds and require the unchanged assertions to pass.
- Document the exact explore/replay commands, `darwin/arm64` overlay status, admitted operations, fixed entropy contract, fail-closed boundary, and direct-syscall limitation in `tools/gomadv3/README.md`.
- Gate: the exact existing test suite passes the DST matrix through the complete pinned profile inventory with no application/test changes.

## Failure Modes and Trade-offs

- **Unknown I/O:** an unclassified site in the pinned inventory or changed dependency hash is a preparation error. An unsupported call entering a controlled shim/overlay boundary returns a stable error. A direct raw syscall outside the reviewed inventory is not contained by this profile and invalidates qualification.
- **Dependency drift:** the target-overlay generator rejects changed versions, hashes, or rewrite anchors. Supporting a new SQLite version requires an explicit reviewed update.
- **Connection races:** registry transitions are linearized under private locks; wakeups happen after unlock, leaving eligible-goroutine order to the schedule seed.
- **Deadlock:** an in-memory wait with no possible producer remains a Gomad deterministic deadlock, distinct from shim rejection and the wall watchdog.
- **Crash:** modeled state is process-local. The Runner publishes an artifact only when its existing atomic publication and bounded terminal-record contracts succeed.
- **Entropy:** bytes are deterministic and non-cryptographic. Concurrent assignment of bytes remains schedule-dependent and is captured by the transcript.
- **Fidelity:** loopback is a reliable byte stream; no packet loss, latency, reordering, partitions, filesystem durability, or production-database timing is modeled.
- **Performance:** real gRPC/SQLite/Temporal code still executes, while semantic application kernel I/O is replaced; bounded Runner control/record streams remain. Cost grows roughly linearly with fresh-process seeds and stays bounded by Runner concurrency and per-child capacities.
- **Ten-times load:** fixed listener, connection, queue, transcript, output, and wall limits produce deterministic capacity failures instead of unbounded memory or disk use.
- **Complexity:** the target overlay is less general than syscall emulation and more invasive than standard-library shims alone, but it closes the demonstrated raw-syscall bypass without modifying Temporal or dependency sources.
- **Security:** the nonambient bootstrap descriptor prevents accidental activation for trusted tests but is not an authentication boundary against hostile same-user code. There is no OS sandbox or general raw-syscall containment; this test-only deterministic entropy profile must never be usable as a production security mode.

## Verification

- `make -C tools/gomadv3 validate` — exact toolchain/target-overlay paths and neighboring rejection fixtures pass.
- `make -C tools/gomadv3 test-harness` — bootstrap, descriptor, bounds, and subprocess harness tests pass.
- `make -C tools/gomadv3 runner-test` — Runner, target validator, manifest, coordinator, transcript, and replay tests pass with the target's existing `test_dep` configuration.
- `make -C tools/gomadv3 world-test` — existing World race coverage remains green.
- `make -C tools/gomadv3 test` — custom-toolchain entropy, TCP, filesystem/hostname, SQLite VFS, preparation-rejection, polling, and disabled upstream `net`, `os`, `crypto/rand`, `runtime`, `time`, and `testing/synctest` fixtures pass. Extend `test.sh` so this single target owns those exact fixture names; never execute a negative fixture whose unclassified call could reach the host.
- `make gomadv3-runner` — the pinned toolchain and Runner build successfully; existing cross-build checks remain green and focused validation rejects platforms without a reviewed overlay inventory.
- Repeat seed 7 in 20 fresh processes with `--parallel 1` and a distinct artifact root per repetition; compare result, bounded streams, profile/overlay identity, ordered transcript, and terminal digest:

  ```sh
  for repetition in $(seq 1 20); do
    tools/gomadv3/.bin/gomad explore --io-profile temporal-activity-api-batch-cancel/v1 --seeds 7 --parallel 1 --run-timeout 2m --overall-timeout 5m --artifacts ".gomad/qualify/repeat-${repetition}" go-test ./tests -- '-test.run=^TestActivityAPIBatchCancelClientTestSuite$'
  done
  ```
- Explore the initial reviewed seed set with `tools/gomadv3/.bin/gomad explore --io-profile temporal-activity-api-batch-cancel/v1 --seeds 0-31 --parallel 4 --run-timeout 2m --overall-timeout 30m --artifacts .gomad/qualify/seeds-0-31 go-test ./tests -- '-test.run=^TestActivityAPIBatchCancelClientTestSuite$'`. A larger range requires a separately reviewed time budget.
- `tools/gomadv3/.bin/gomad replay ARTIFACT_DIR` — replay the deterministic failing Gomad fixture and any naturally produced target failure; `--verify-only` also validates their immutable inputs.
- Format only the explicit changed Go files under `tools/gomadv3` with `gofmt` and the repository's pinned import formatter. Run `make GOLANGCI_LINT_FIX=false lint-code`; do not invoke a repository-wide fixer.
- Final guards: compare current Git status with the captured baseline and reject new entries outside `PLAN.md` and `tools/gomadv3/**`; verify the target checksum; run `git diff --exit-code -- tests common service temporal` and inspect `git diff --name-only`.

## Context Files

- `GOMADv3_CLOCK.md` — native virtual time, quiescence, and host-I/O boundary.
- `GOMADv3_WORLD.md` — adapter ownership, fail-closed behavior, evidence, and replay.
- `GOMADv3_RUNNER.md` — target preparation, bootstrap, artifacts, and replay.
- `GOMADv3_NEXT.md` — transparent Temporal-pilot success criteria.
- `tools/gomadv3/README.md` — current activation and Runner contract.
- `tools/gomadv3/test.sh` — patch/overlay validation and custom-toolchain fixtures.
- `tools/gomadv3/build.sh` — immutable toolchain construction.
- `tools/gomadv3/overlay/src/runtime/gomad.go` — pre-package-init activation.
- `tools/gomadv3/internal/target/target.go` — prepared target identity and Go build.
- `tools/gomadv3/internal/process/bootstrap_unix.go` — inherited bootstrap descriptors.
- `tools/gomadv3/internal/runner/runner.go` — per-seed isolation and artifact construction.
- `tools/gomadv3/internal/replay/replay.go` — exact replay validation.
- `tools/gomadv3/internal/record/types.go` — manifest and persisted evidence.
- `tests/activity_api_batch_cancel_test.go` — unchanged qualification target.
- `tests/activity_api_batch_terminate_test.go` — unchanged shared batch helpers.
- `tests/testcore/test_cluster.go` and `tests/testcore/onebox.go` — unchanged cluster startup, SQLite, ports, pprof, and diagnostics.
- `common/testing/freeport/freeport.go` — direct TCP APIs and concrete address behavior.
- `common/resource/fx.go` — hostname and RPC consumers.
- `service/frontend/http_api_server.go` — direct `net.ListenTCP` usage.
- `common/persistence/sql/sqlplugin/sqlite/plugin.go` and `schema/sqlite/setup.go` — unchanged in-memory persistence path.
