# Gomad v3

Gomad v3 is an opt-in Go 1.26.4 toolchain with a small deterministic-runtime
patch and source overlay. It uses native Go goroutines, channels, `select`,
maps, synchronization, `go run`, and `go test`.

Build the argv-safe CLI and cached toolchain from the repository root:

```sh
make gomadv3
```

The complete Gomad v3 Runner and deterministic-I/O contract is qualified only
on `darwin/arm64`. The builder rejects other hosts before starting a toolchain
build. Runtime source may remain portable to other Unix systems, but those
systems are not supported Runner platforms until they have their own adapters,
publication primitives, and complete test gate.

Use the CLI directly so every package and target argument crosses exactly one
argv boundary:

```sh
tools/gomadv3/.bin/gomad explore --seeds 1 go-run ./cmd/example -- arg
tools/gomadv3/.bin/gomad explore --seeds 1 go-test ./path/to/package -- -test.run=TestName
```

`make gomadv3-runner` is an alias for the same CLI build, while
`make gomadv3-go` builds only the toolchain. The older single-seed wrappers
remain for compatibility:

```sh
GOMADSEED=1 make gomadv3-run GOMADV3_RUN=./cmd/example
GOMADSEED=1 make gomadv3-test GOMADV3_PACKAGES=./path/to/package
```

Those wrappers remove `GOMADSEED` from the custom `go` process and use Go's
`-exec` hook to enable Gomad only in the resulting binary or generated test
binary. Direct execution of a prebuilt binary remains
`GOMADSEED=<seed> ./binary`.

Check whether the complete Runner and deterministic-I/O contract is available
before starting a campaign:

```sh
tools/gomadv3/.bin/gomad doctor
tools/gomadv3/.bin/gomad doctor --json --artifacts .gomad/artifacts
```

The report includes the host, toolchain and Runner build identities, boundary
manifest, I/O implementation, pinned adapter, artifact-directory access, and
the exact build command needed to repair missing local components.

Explore `go run`, `go test`, or a prepared executable target, then replay a
retained failure exactly or verify its immutable inputs without executing it:

```sh
tools/gomadv3/.bin/gomad explore --seeds 0-999 go-run ./cmd/example -- arg
tools/gomadv3/.bin/gomad explore --count 1000 go-run ./cmd/example -- arg
tools/gomadv3/.bin/gomad explore --coverage=semantic --keep-successes=novel --success-limit=32 --success-bytes=1GiB --count 1000 go-run ./cmd/example -- arg
tools/gomadv3/.bin/gomad explore --seeds 0,7,42 go-test ./path/to/package -- -test.run=TestName
tools/gomadv3/.bin/gomad explore --seeds 0-99 exec --provenance ./example.provenance.json -- ./example arg
tools/gomadv3/.bin/gomad qualify --seed 7 --repeat 2 go-test ./path/to/package -- -test.run=TestName
tools/gomadv3/.bin/gomad resume .gomad/artifacts/v1/run-INTERRUPTED
tools/gomadv3/.bin/gomad inspect .gomad/artifacts/v1/run-*
tools/gomadv3/.bin/gomad inspect .gomad/artifacts/v1/run-*/failures/sha256-*
tools/gomadv3/.bin/gomad inspect .gomad/artifacts/v1/run-*/successes/sha256-*
tools/gomadv3/.bin/gomad replay .gomad/artifacts/v1/run-*/failures/sha256-*
tools/gomadv3/.bin/gomad replay .gomad/artifacts/v1/run-*/successes/sha256-*
tools/gomadv3/.bin/gomad replay --verify-only .gomad/artifacts/v1/run-*/failures/sha256-*
```

Human-readable exploration writes preparation and running progress to stderr,
including attempted, active, successful, failed, watchdog, replay-divergence,
distinct-failure, retained-success, and retained-success byte counts. Its final
result and every retained artifact path with a copy-paste replay command are
written to stdout.

Add `--json` to emit newline-delimited `gomadv3.explore-event/v1` records on
stdout and no routine output on stderr. Event types are `progress`, `result`,
`artifact`, and `error`. Result classifications are `success`,
`target_failure`, `watchdog_observation`, `replay_divergence`, and
`mixed_failure`; error classifications are `invalid_input`,
`unsupported_target`, `semantic_coverage_failure`, and `runner_failure`.

Use `--coverage=semantic` to include the versioned union of observed reviewed
boundary probes and its stable digest in the result. Repeat `--require-probe`
to make an unobserved known probe fail the campaign with classification
`semantic_coverage_failure` and status 1:

```sh
tools/gomadv3/.bin/gomad explore --coverage=semantic \
  --require-probe=stdlib.os.openfile --count 100 go-test ./path/to/package
```

Successful runs are discarded by default. `--keep-successes=novel` retains the
first completed success that adds a new semantic probe and therefore requires
`--coverage=semantic`; `--keep-successes=all` retains every success. Both modes
require a positive `--success-limit` and `--success-bytes`. Crossing either
bound fails the campaign visibly instead of silently dropping replay evidence.
Each retained success is an immutable exact-replay artifact, and its stored byte
count and novelty reasons are recorded in the batch journal. Success replay
returns status 0 only when the recorded successful outcome matches.

`gomad qualify` prepares and executes the target independently two or more
times with one seed, compares bounded canonical evidence, and automatically
retains a private `gomadv3.qualification/v1` report below
`ARTIFACTS/qualifications/v1`. Evidence includes the exact target, argv,
toolchain and Runner identities, full output hashes, transcript, captured-mount
identity, World identity, outcome, and semantic probes. A retained target
failure is replayed once and its result is recorded. Repeat `--require-probe`
to enforce known conditional probes; `--repeat` is bounded to 2 through 32.
Add `--json` for newline-delimited `gomadv3.qualify-event/v1` progress, result,
and error records. Unsupported targets retain their first boundary and exact
command in the qualification report.

Run the checked representative Temporal set with:

```sh
make -C tools/gomadv3 qualification-set
```

`qualification/temporal.json` is a versioned, bounded inventory of unchanged
Temporal tests and exact expected dispositions. The Go orchestrator runs every
entry through `gomad qualify`, validates each private report and executed
command, and atomically publishes the self-contained canonical aggregate at
`.toolchain/qualification-set.json`. The current set covers clock/context time,
future synchronization, timer gates, the functional activity-batch dependency
closure, and repository-backed SQLite persistence preparation. An expected
unsupported boundary is recorded as such; it does not claim that workload is
supported. Any changed boundary, lost required probe, nondeterminism, replay
divergence, or unexpected outcome fails the set while retaining all evidence.
Scheduled and manually dispatched CI runs upload the aggregate and its
referenced artifacts for 90 days.

An interrupted campaign retains a canonical `gomadv3.batch-plan/v1` beside
its prepared target. `gomad resume BATCH` locks that batch, verifies the exact
Runner, toolchain, I/O profile, prepared binary, completed records, and every
referenced failure or successful-run artifact, archives incomplete per-seed state, and schedules
only unfinished selection ordinals. It appends to and eventually publishes the
original batch; repeated resumes are safe when the recorded aggregate deadline
is too short to finish all remaining seeds. Published batches, changed inputs,
concurrent resumes, and interrupted preparation fail closed. Add `--json` to
use the same stable campaign event stream as `explore`.

| Status | `explore` / `resume` | `qualify` | `replay` |
| --- | --- | --- | --- |
| 0 | All selected or remaining runs succeeded. | Every repetition succeeded with identical evidence. | Verification-only succeeded, or a retained success replayed exactly. |
| 1 | A target failure, watchdog observation, or replay divergence was retained. | Evidence diverged, a target failed, a required probe was absent, or replay diverged. | The stored observation reproduced exactly, or replay diverged; inspect `reproduced=true|false`. |
| 2 | Input is invalid, the target is unsupported, or the resume journal is incompatible. | Input was invalid or the unsupported boundary was retained. | Input or artifact compatibility validation failed. |
| 3 | Runner or host infrastructure failed. | Qualification or report infrastructure failed. | Replay infrastructure failed. |

The Runner prepares one immutable target, launches every seed in a fresh
contained process and work directory, enforces wall deadlines, computes full
stream hashes while retaining bounded output, and publishes canonical,
content-addressed artifacts. `--count N` selects seeds `0` through `N-1` and is
mutually exclusive with `--seeds`. Arguments following `--` use an argv-safe
interface. Trusted repository tooling preparing an `exec` target must use
`target.ReviewCapabilityClosure` and `target.WriteProvenance` to produce v2
provenance for the exact binary. Runner revalidates its package policy, pinned
standard-library membership, module closure, build information, and binary
identity; v1 or arbitrary binaries are rejected.

`gomad inspect` validates the batch journal or immutable failure/success artifact before
printing its identity, outcome, transcripts, captured mounts, truncation,
distinct failure paths, retained successes and byte totals, novelty reasons,
and copy-paste replay commands. Add `--json` for the
stable `gomadv3.inspect/v1` report.

### Deterministic I/O

Every Runner-managed target uses the versioned deterministic-I/O boundary by
default. It is independent of the target package, arguments, and application:

```sh
tools/gomadv3/.bin/gomad explore \
  --seeds 7 --parallel 1 --run-timeout 2m --overall-timeout 5m \
  --artifacts .gomad/qualify/seed-7 \
  go-test ./path/to/package -- '-test.run=^TestName$'
```

Gomad replaces supported loopback TCP operations, filesystem operations,
hostname, and entropy with process-local in-memory implementations. A reviewed,
version-pinned adapter redirects supported `modernc.org/libc` filesystem,
entropy, and time operations to those same generic boundaries. Entropy is
independent of `GOMADSEED`; that seed controls scheduling only.

The version-pinned compiler inserts typed entry prologues into the selected
`os` and `net` definitions before optimization. This keeps the standard names,
method sets, interfaces, and call sites intact while routing every invocation
form through additive same-package hooks. Before rewriting, the compiler
validates each definition's complete formatted declaration fingerprint as well
as its name and signature, so a signature-stable upstream body change fails the
build. It marks intercepted definitions non-inline so serialized pre-rewrite
bodies cannot bypass the hook.

Every modeled operation is appended to a bounded shared-memory transcript.
Retained artifacts keep the canonical transcript, and replay supplies it to
the target through a read-only shared-memory region so divergence stops at the
first mismatching ordinal:

```sh
tools/gomadv3/.bin/gomad replay ARTIFACT_DIR
tools/gomadv3/.bin/gomad replay --verify-only ARTIFACT_DIR
```

Unsupported calls entering an inventoried shim fail closed before host I/O.
This boundary is not an OS sandbox: trusted target code must not bypass the
reviewed boundaries with a direct raw syscall. DNS, non-loopback sockets,
subprocesses, cgo, plugins, external linking, and unrecognized native I/O are
outside its supported contract.

### Lazy read-only inputs

The Runner can expose an explicit host directory through a repeatable lazy
read-only mount. It captures only entries first observed by the target,
serves subsequent reads from memory, and stores captured inputs in retained
failure artifacts so exact replay does not reopen the host directory:

```sh
tools/gomadv3/.bin/gomad explore \
  --io-ro-mount ./fixtures=/fixtures \
  --seeds 7 --parallel 1 \
  go-test ./path/to/package -- '-test.run=^TestName$'
```

Mount sources are resolved relative to the Runner working directory; target
destinations are normalized into its virtual absolute namespace and may not
overlap. Symlinks, hard-linked files, special entries, unstable captures, and
capacity overflow fail closed. Write-capable opens within mounts return
`EROFS`, and reads outside declared mounts do not fall through to host files.

The compatibility-only `GOMADV3_RUN`, `GOMADV3_PACKAGES`, and `GOMADV3_ARGS`
variables are trusted Make recipe shell fragments, not an argv-safe public
interface. Shell metacharacters and values that require quoting must be quoted
for both Make and the recipe shell.

The stable Go command is `tools/gomadv3/.toolchain/bin/go`. The build verifies
the official Go source checksum, snapshots and validates `go1.26.4.patch` and
`overlay`, rejects upstream overlay collisions, copies the exact overlay
snapshot, applies the exact patch snapshot with zero fuzz, and caches immutable
builds by the Go version, source checksum, patch and overlay checksums, host OS
and architecture, bootstrap Go version, and canonical build environment.
Same-key builds use an atomic owner lock, and ambient Go experiment,
architecture, C/C++ tool, and compiler/linker tuning is cleared before
`make.bash`. Set `GOMADV3_BOOTSTRAP_GO` to choose a bootstrap `go` command.

To upgrade Go, update the canonical `version.json` descriptor and
`boundary/manifest.json`, materialize the old patch against the new pinned
source, and regenerate the patch with `regenerate-patch.sh`. `make -C
tools/gomadv3 generate` derives the shell, Make, Go, compiler-spec,
interception-report, public-inventory, and upgrade-guide consumers. The
descriptor's patch and overlay allowlists must exactly equal the checked trees.

Run the version-specific command from the generated upgrade guide, or directly:

```sh
make -C tools/gomadv3 upgrade-dossier GOMADV3_BASELINE_REF=<previous-commit>
```

The command publishes `.toolchain/upgrade-dossier.json` even when a behavioral
gate fails. It records the complete upstream patch, semantic boundary diff,
interception evidence, overlay collision audit, disabled upstream results,
mandatory probes, host-clock audit, optional retained-corpus evidence, and
platform qualification. Pass
`GOMADV3_CORPUS_REPORT=.toolchain/qualification-set.json` to require and embed a
fully validated representative qualification set. Supported-host CI uploads
the dossier on every run and includes that corpus evidence on scheduled and
manual qualification runs.

The standard-library boundary is declared in
`tools/gomadv3/boundary/manifest.json`, and the cross-process deterministic-I/O
layouts are declared in `tools/gomadv3/protocol/iowire.json`. After changing
either schema or its templates, regenerate and verify the derived artifacts
with:

```sh
make -C tools/gomadv3 generate
make -C tools/gomadv3 validate
```

Generated host and overlay tests consume the same golden vectors. The normal
Gomad v3 test target also tests the overlay codec and typed mount client inside
the patched toolchain.

## Contract

A directly launched target activates Gomad with `GOMADSEED`. A Runner-managed
target activates deterministic I/O through an identity-bound inherited
bootstrap frame and a private activation marker. When neither path is present,
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

Deterministic mode supports internally linked pure-Go targets on the qualified
`darwin/arm64` host. Enabled cgo or externally linked binaries fail before package
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
  artifact, replay, and deterministic-I/O decisions.
- [Active TODO](../../GOMADv3_TODO.md) tracks the prioritized implementation and
  verification backlog.
- [Functional-suite sweep](docs/2026-08-11-functional-suite-sweep.md) records
  the current unchanged-Temporal integration evidence.

## Development

Run source validation and the black-box suite with:

```sh
make -C tools/gomadv3 test
make -C tools/gomadv3 test-builder
make -C tools/gomadv3 test-runtime
make -C tools/gomadv3 test-upstream
make -C tools/gomadv3 runner-test
make -C tools/gomadv3 world-test
make -C tools/gomadv3 upgrade-dossier GOMADV3_BASELINE_REF=<previous-commit>
```

`test` retains the full gate and runs the builder, runtime, and upstream tiers
in that order. The focused targets reproduce the corresponding portion without
weakening the full gate.

The suite compares disabled `go run` and `go test` behavior with a local stock
Go 1.26.4 toolchain; benchmarks disabled clock reads against that toolchain;
covers the fixed clock, native timer behavior, context deadlines, logical test
timeouts, nested synctest, cgo/link rejection, non-progress, bounded output,
and deadlock; runs focused upstream `runtime`, `time`, and `testing/synctest`
tests; audits map key families across seeds; and repeats prebuilt map and
scheduler fixtures under distinct allocation layouts and bounded unrelated CPU
load. Supported-host CI additionally runs `make clock-audit`: a privileged,
positive-controlled DTrace gate that rejects seeded calls to `clock_gettime` or
`mach_absolute_time` after Gomad activation. Set `GOMADV3_STOCK_GO` when the
stock Go executable cannot be resolved
from the module-selected toolchain in `PATH`; the test never downloads one.

The address-only check disables automatic GC so retained padding changes the
tested layout without adding GC activity that consumes the shared seeded
runtime stream. The ordinary repeatability and host-load checks retain the
default GC behavior.

Generated source, binaries, downloads, and toolchain builds remain under
`tools/gomadv3/.toolchain` and are not committed.
