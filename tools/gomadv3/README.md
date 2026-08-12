# Gomad v3

Gomad v3 is an opt-in Go 1.26.4 toolchain with a small deterministic-runtime
patch and source overlay. It uses native Go goroutines, channels, `select`,
maps, synchronization, `go run`, and `go test`.

Build the cached toolchain from the repository root:

```sh
make gomadv3-go
```

Run a command or test package with a seed:

```sh
GOMADSEED=1 make gomadv3-run GOMADV3_RUN=./cmd/example
GOMADSEED=1 make gomadv3-test GOMADV3_PACKAGES=./path/to/package
```

The Make targets remove `GOMADSEED` from the custom `go` process and use Go's
`-exec` hook to enable Gomad only in the resulting binary or generated test
binary. Direct execution of a prebuilt binary remains
`GOMADSEED=<seed> ./binary`.

Build the bounded multi-seed Runner:

```sh
make gomadv3-runner
```

Explore `go run`, `go test`, or a prepared executable target, then replay a
retained failure exactly or verify its immutable inputs without executing it:

```sh
tools/gomadv3/.bin/gomad explore --seeds 0-999 go-run ./cmd/example -- arg
tools/gomadv3/.bin/gomad explore --seeds 0,7,42 go-test ./path/to/package -- -test.run=TestName
tools/gomadv3/.bin/gomad explore --seeds 0-99 exec --provenance ./example.provenance.json -- ./example arg
tools/gomadv3/.bin/gomad replay .gomad/artifacts/v1/run-*/failures/sha256-*
tools/gomadv3/.bin/gomad replay --verify-only .gomad/artifacts/v1/run-*/failures/sha256-*
```

The Runner prepares one immutable target, launches every seed in a fresh
contained process and work directory, enforces wall deadlines, computes full
stream hashes while retaining bounded output, and publishes canonical,
content-addressed artifacts. Arguments following `--` use an argv-safe
interface. Trusted tooling preparing an `exec` target must generate the typed
provenance consumed by the Runner; an arbitrary binary is rejected.

### Temporal activity batch-cancel profile

The `temporal-activity-api-batch-cancel/v1` profile runs the existing Temporal
suite without changing its test, shared test infrastructure, or production
code. It is intentionally restricted to Go 1.26.4 on `darwin/arm64`, the exact
`./tests` package, the `test_dep` build tag, and the complete suite selector:

```sh
tools/gomadv3/.bin/gomad explore \
  --io-profile temporal-activity-api-batch-cancel/v1 \
  --seeds 7 --parallel 1 --run-timeout 2m --overall-timeout 5m \
  --artifacts .gomad/qualify/seed-7 \
  go-test ./tests -- '-test.run=^TestActivityAPIBatchCancelClientTestSuite$'
```

For this profile, Gomad replaces the reached loopback TCP operations, minimal
directory metadata, hostname, and entropy stream with process-local in-memory
implementations. A generated build overlay redirects the pinned
`modernc.org/sqlite` VFS time and entropy sites without editing the module
cache. Entropy is fixed by the profile and independent of `GOMADSEED`; that
seed controls scheduling only.

Every modeled operation is appended to a bounded shared-memory transcript.
Failure artifacts retain the canonical transcript, and replay supplies it to
the target through a read-only shared-memory region so divergence stops at the
first mismatching ordinal:

```sh
tools/gomadv3/.bin/gomad replay ARTIFACT_DIR
tools/gomadv3/.bin/gomad replay --verify-only ARTIFACT_DIR
```

Unsupported calls entering an inventoried shim fail closed before host I/O.
This target-specific profile is not an OS sandbox: trusted target code must not
bypass the reviewed boundaries with a direct raw syscall. DNS, non-loopback
sockets, arbitrary files, subprocesses, cgo, plugins, and external linking are
outside its supported contract.

### Lazy read-only inputs

An I/O profile can expose an explicit host directory through a repeatable lazy
read-only mount. The Runner captures only entries first observed by the target,
serves subsequent reads from memory, and stores captured inputs in retained
failure artifacts so exact replay does not reopen the host directory:

```sh
tools/gomadv3/.bin/gomad explore \
  --io-profile temporal-activity-api-batch-security/v1 \
  --io-ro-mount ./schema/sqlite/v3=go.temporal.io/server/schema/sqlite/v3 \
  --seeds 7 --parallel 1 --run-timeout 4m --overall-timeout 8m \
  --artifacts .gomad/batch-security \
  go-test ./tests -- '-test.run=^TestActivityAPIBatchSecurityTestSuite$'
```

Mount sources are resolved relative to the Runner working directory; target
destinations are normalized into its virtual absolute namespace and may not
overlap. Symlinks, hard-linked files, special entries, unstable captures, and
capacity overflow fail closed. Write-capable opens within mounts return
`EROFS`, and reads outside declared mounts do not fall through to host files.

`GOMADV3_RUN`, `GOMADV3_PACKAGES`, and `GOMADV3_ARGS` are trusted Make recipe
shell fragments, not an argv-safe public interface. Shell metacharacters and
values that require quoting must be quoted for both Make and the recipe shell.

The stable Go command is `tools/gomadv3/.toolchain/bin/go`. The build verifies
the official Go source checksum, snapshots and validates `go1.26.4.patch` and
`overlay`, rejects upstream overlay collisions, copies the exact overlay
snapshot, applies the exact patch snapshot with zero fuzz, and caches immutable
builds by the Go version, source checksum, patch and overlay checksums, host OS
and architecture, bootstrap Go version, and canonical build environment.
Same-key builds use an atomic owner lock, and ambient Go experiment,
architecture, C/C++ tool, and compiler/linker tuning is cleared before
`make.bash`. Set `GOMADV3_BOOTSTRAP_GO` to choose a bootstrap `go` command.

## Contract

A directly launched target activates Gomad with `GOMADSEED`. A Runner-managed
I/O profile activates through `GOMADV3_IO_PROFILE` plus an identity-bound
inherited bootstrap frame that supplies the seed. When neither path is present,
the toolchain follows the upstream runtime paths. Activation forces the initial
`GOMAXPROCS` to one, disables asynchronous preemption, and seeds existing
runtime choice paths. Seed `0` is valid; empty, malformed, and overflowing
direct seed values fail before user initialization.

Enabled targets start at midnight UTC on 2000-01-01. Standard `time.Now`,
monotonic elapsed time, sleeps, timers, tickers, callbacks, and context
deadlines use the process virtual clock. When no goroutine is runnable, the
runtime advances directly to the earliest native timer deadline. Runnable
work is never skipped to deliver a future timer, and equal-deadline timers use
the seeded runtime choice stream. An explicit `testing/synctest` bubble keeps
its private clock and takes precedence over the process clock.

The standard `go test` harness also observes virtual time. In particular,
`-test.timeout` is a logical-time deadline and may fire immediately in wall
time when it is the next event. A separate wall-time process watchdog is still
required for CPU loops, unsupported host operations, and toolchain failures.

For a fixed toolchain, architecture, program, deterministic external inputs,
and seed, supported runtime-controlled choices repeat across fresh processes.
Different seeds explore different choices when alternatives exist. Runtime
choices must finish before output or other external I/O is performed.

Deterministic mode supports internally linked pure-Go targets on Unix-like
hosts. Enabled cgo or externally linked binaries fail before package
initialization. Windows, plugins, foreign threads, the race detector, signals,
finalizers, and host-dependent network, filesystem, process, and other I/O
readiness are outside the contract. The launch targets compile with
`CGO_ENABLED=0`, set `TZ=UTC`, and preserve `-tags test_dep` for tests.

The runtime system monitor is disabled with asynchronous preemption, so a
CPU-bound goroutine or `select` polling loop may run forever and prevent
virtual-time advancement. Unsupported blocking I/O is likewise bounded by the
external wall watchdog rather than treated as a clock event. Calling
`runtime.GOMAXPROCS` to raise the value after startup is unsupported.

The shell test harness retains at most 1 MiB from each child output stream while
continuing to drain both streams. Every harness result directory records
`output-truncated` separately from `timed-out` and the child `status`. The Gomad
Runner has a separate configurable per-stream limit that defaults to 8 MiB.

The mode is intended only for trusted tests. Deterministic map seeds remove a
hash-randomization defense and must not be enabled in production. Each process
uses one P, so run different seeds in separate processes for parallelism. The
shared runtime random state also means program changes can change later choices.

## World

`world` is a pure in-memory model for deterministic events outside the Go
runtime. It performs no host I/O, starts no goroutines, invokes no callbacks,
and requires no runtime hook. Callers register requests, mark them ready, and
explicitly quiesce to choose and deliver ready events.

`world/mailbox` is the initial explicit adapter. It demonstrates lifecycle,
snapshot/restore, and replay without giving World ownership of application
state. `internal/worldrecord` composes World semantic records with the Runner's
raw process record while keeping those identities separate. A target connects
its World with `world/child.Open`, takes the session-owned World returned by
`Session.World`, performs all modeled work, and calls
`Session.Finish` after that work has stopped, or `Session.FinishError` for a
typed World error. The trusted bootstrap validates replay input before target
activation; `Open` installs that recorded initial World rather than accepting a
target-created substitute and returns it through `Session.World` before modeled work.
The session writes one bounded record with a structured idle, deadlock,
capacity, invalid-input, or replay-divergence terminal result through inherited
descriptors only at the process boundary, so host pipe readiness cannot affect
event ordering. Connected replay requires the executing child to emit the same
semantic bundle and fails closed if it is missing or divergent.

## Design and roadmap

- [Architecture](ARCHITECTURE.md) records the durable runtime, Runner, World,
  artifact, replay, and I/O-profile decisions.
- [Roadmap](docs/roadmap.md) tracks remaining capability work.
- [Testing backlog](docs/testing-backlog.md) tracks runtime and toolchain
  coverage gaps.
- [Functional-suite sweep](docs/2026-08-11-functional-suite-sweep.md) records
  the current unchanged-Temporal integration evidence.

## Development

Run source validation and the black-box suite with:

```sh
make -C tools/gomadv3 test
make -C tools/gomadv3 runner-test
make -C tools/gomadv3 world-test
```

The suite compares disabled `go run` and `go test` behavior with a local stock
Go 1.26.4 toolchain; benchmarks disabled clock reads against that toolchain;
covers the fixed clock, native timer behavior, context deadlines, logical test
timeouts, nested synctest, cgo/link rejection, non-progress, bounded output,
and deadlock; runs focused upstream `runtime`, `time`, and `testing/synctest`
tests; audits map key families across seeds; and repeats prebuilt map and
scheduler fixtures under distinct allocation layouts and bounded unrelated CPU
load. Set `GOMADV3_STOCK_GO` when the stock Go executable cannot be resolved
from the module-selected toolchain in `PATH`; the test never downloads one.

The address-only check disables automatic GC so retained padding changes the
tested layout without adding GC activity that consumes the shared seeded
runtime stream. The ordinary repeatability and host-load checks retain the
default GC behavior.

Generated source, binaries, downloads, and toolchain builds remain under
`tools/gomadv3/.toolchain` and are not committed.
