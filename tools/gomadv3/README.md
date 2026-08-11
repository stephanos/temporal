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

When `GOMADSEED` is absent, the toolchain follows the upstream runtime paths.
When it is present, the runtime parses it as a `uint64`, forces the initial
`GOMAXPROCS` to one, disables asynchronous preemption, and seeds existing
runtime choice paths. Seed `0` is valid; empty, malformed, and overflowing
values fail before user initialization.

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

The checked runner retains at most 1 MiB from each child output stream while
continuing to drain both streams. Every result directory records
`output-truncated` separately from `timed-out` and the child `status`.

The mode is intended only for trusted tests. Deterministic map seeds remove a
hash-randomization defense and must not be enabled in production. Each process
uses one P, so run different seeds in separate processes for parallelism. The
shared runtime random state also means program changes can change later choices.

## Development

Run source validation and the black-box suite with:

```sh
make -C tools/gomadv3 test
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
