# Gomad virtual time plan

## Decision summary

Implement transparent virtual time by activating the dormant process-global
`faketime` path already present in Go 1.26.4. In Gomad mode, standard
`time.Now`, `time.Sleep`, timers, tickers, and the timers behind context
deadlines use one virtual clock. The existing runtime timer heaps remain the
only clock event queue, and the existing `checkdead` path advances the clock to
the next deadline only after the runtime proves that no goroutine is runnable.

This is lighter than putting the whole program in a `testing/synctest` bubble:
it preserves normal `runtime.main`, package initialization, binaries, and the
`testing` harness; it also permits existing Temporal tests to create their own
nested `synctest` bubbles. The implementation should reuse synctest's semantics
and equal-deadline timer ordering, but not make a process-wide bubble the
primary mechanism.

The clock is enabled by the existing `GOMADSEED` gate. No application changes,
clock injection, source rewriting, or new third-party dependency are required.
The deterministic region must not consume host I/O or host readiness. Virtual
time does not attempt to make filesystem, network, DNS, process, signal, cgo,
or other external operations deterministic.

## Implementation status

Implemented on 2026-08-10. The runtime overlay activates the fixed process
clock and rejects cgo/external linking, while the Go patch routes wall and
monotonic reads through it, marks external links reliably, handles saturated
deadlines, and seeds equal-deadline timer insertion and rearm ordering. The root
run/test targets now keep `cmd/go` outside Gomad and activate only target
binaries through `tools/gomadv3/exec.sh`.

The checked-in suite covers direct binaries, `go run`, `go test`, logical test
timeouts, native timer and context behavior, ticker coalescing, simultaneous
deadlines, nested `testing/synctest`, runnable non-progress, unsupported
blocking netpoll I/O, deterministic deadlock, bounded output, and
cgo/external-link rejection. Focused upstream `runtime`, `time`, and
`testing/synctest` suites run with Gomad disabled. The
unchanged `./common/timer` and `./common/testing/testcontext` Temporal packages
also pass through `gomadv3-test`.

## Goals

- Run unmodified Go binaries and unmodified `go test` packages under virtual
  time.
- Make all standard Go time behavior inside the Gomad process deterministic:
  wall-clock reads, monotonic elapsed time, sleeps, one-shot timers, callbacks,
  tickers, stop/reset, and context deadlines/cancellation.
- Advance time only when no user goroutine is runnable; runnable work must never
  be skipped to deliver a future timer.
- Reproduce the ordering of simultaneous deadlines for the same seed, while
  allowing different seeds to explore legitimate timer-order choices.
- Preserve ordinary upstream behavior when `GOMADSEED` is absent.
- Preserve explicit `testing/synctest` behavior in tests that already use it.
- Keep the Go patch and public surface as small as possible.

## Non-goals

- Deterministic host I/O, including files, sockets, DNS, databases, subprocesses,
  signals, or readiness notification.
- Making blocking syscalls or polling loops advance virtual time.
- Supporting cgo, plugins, foreign threads, the race detector, or externally
  linked binaries in deterministic mode.
- Multiple virtual clocks or time domains within one Gomad process.
- A public Gomad clock API or changes to Temporal's `common/clock` interfaces.
- Cross-version replay. Repeatability still requires the same Gomad toolchain
  build key, program, architecture, deterministic inputs, and seed.
- Replacing the future World event queue needed for deterministic external
  adapters. This plan solves native Go time only.
- Detecting or sandboxing every prohibited I/O operation. The clock milestone
  relies on selecting code whose deterministic region obeys the no-I/O
  contract; enforcing that contract requires the later Runner sandbox or World
  adapters.

## Deterministic clock contract

### Activation and initial value

- `GOMADSEED` remains the sole target-runtime activation switch.
- A successfully parsed seed initializes process `faketime` to
  `946684800000000000`, midnight UTC on 2000-01-01. This matches synctest and is
  nonzero, which is required by the existing runtime faketime checks.
- Initialization happens in `gomadInit`, before package initialization and user
  code. Invalid, empty, or overflowing seeds continue to fail before user code.
- The initial instant is fixed and does not vary by seed. The seed controls
  choices, not the meaning of time.
- `GOMADSEED` absent leaves `faketime` zero and all upstream clock paths intact.

### Clock reads

- Outside a synctest bubble, `time.Now` returns the process virtual wall time
  and a virtual monotonic component derived from the same value.
- `runtime.nanotime`, and therefore standard monotonic duration measurement,
  returns process virtual time while Gomad is enabled.
- Inside an explicit synctest bubble, that bubble's clock takes precedence,
  exactly as it does upstream. The process clock remains frozen while only
  bubbled work is advancing its private clock.
- Time never follows host wall-clock adjustments after Gomad activation.
- `time.Local` must not trigger a host timezone lookup in a supported run.
  Gomad launchers set `TZ=UTC`; explicit `time.LoadLocation` remains unsupported
  host filesystem I/O unless data is supplied by a future deterministic adapter.

### Advancement and quiescence

The runtime follows this loop:

```text
run every currently runnable goroutine
              |
              v
runtime proves no goroutine is runnable
              |
       +------+------+
       |             |
next timer exists   no timer exists
       |             |
       v             v
jump to earliest    ordinary stable
deadline            deadlock outcome
       |
       v
make all timers at that instant eligible
       |
       +------------> schedule runnable work
```

- Existing `checkdead` accounting is the quiescence proof. No wall-clock polling,
  sleeps, controller goroutine, or application handshake is added.
- Existing `timeSleepUntil` scans the native per-P timer heaps and supplies the
  earliest future deadline.
- Gomad distinguishes an empty timer set from a saturated `maxWhen` deadline,
  so the latter can advance without changing the upstream disabled path.
- Existing faketime scheduler branches avoid waiting for host netpoll before
  the quiescence check.
- The clock jumps directly to the next deadline; it does not tick through
  intermediate instants.
- A goroutine that is runnable, including a busy loop or `select` with a
  `default` polling branch, prevents time from advancing. The external Runner's
  real-time watchdog terminates such a process.
- An unsupported host read that happens to complete is not made deterministic.
  While Go netpoll has waiters, Gomad does not treat the process as timer-only
  quiescence; a blocked read is terminated by the wall watchdog.
- If the program is blocked with no future timer, the normal deterministic
  deadlock failure remains distinguishable from a wall-watchdog timeout.

### Timers and simultaneous deadlines

- Continue to use Go's native runtime timer implementation for `Sleep`,
  `NewTimer`, `After`, `AfterFunc`, `NewTicker`, `Tick`, `Stop`, and `Reset`.
  Context timeouts and deadlines inherit these semantics through package
  `time`.
- Preserve all upstream zero, negative, stop/reset, stale-send, ticker coalescing,
  cancellation, and duration-overflow behavior. Gomad changes when timers are
  delivered, not their public API contract.
- On every insertion or reinsertion of a process-global Gomad timer, assign its
  existing `timer.rand` tie-break from the Gomad-seeded runtime PRNG. The
  existing `timerWhen.less` comparison then orders timers with identical
  deadlines.
- Apply this randomization only when `gomadEnabled` is true or the timer is an
  upstream synctest fake timer. Do not key it merely on nonzero `faketime`, which
  would accidentally change the upstream playground build-tag behavior.
- All timers due at the advanced instant remain eligible. Seeded ordering may
  choose which callback or goroutine becomes runnable first, but it may not
  drop, postpone, duplicate, or fabricate a timer event.
- Timer tie-breaks consume the existing private runtime choice stream. As with
  other v3 choices, adding unrelated program activity may change later choices;
  stability is promised for an unchanged program, not across source changes.

### `go test` behavior

- The complete generated test binary, including package initialization and the
  standard `testing` harness, runs on the process virtual clock.
- `-test.timeout` is therefore a logical-time timeout. A test suite that becomes
  quiescent with only its timeout timer pending advances immediately to that
  timer and fails deterministically.
- A test that intentionally sleeps past `-test.timeout` must raise or disable
  the logical timeout. A separate host-time Runner watchdog is always retained
  for CPU loops, unsupported syscalls, toolchain crashes, and other cases where
  virtual time cannot progress.
- `testing.T.Deadline` and test duration reporting become repeatable virtual-time
  observations.
- Existing tests using `synctest.Test` continue to use their isolated bubble
  clock. This compatibility is a required black-box test, not an assumption.
- Multiple test packages are still separate processes. Deterministic aggregation
  of their output belongs to the Runner; use serial package execution initially
  when combined output ordering matters.

## Process boundary for binaries and `go test`

The custom `go` command itself performs host I/O and must not inherit
`GOMADSEED`. Only the binary it produces may enter the deterministic region.
Use Go's existing `-exec` support rather than teaching the runtime to distinguish
`cmd/go` from a target:

```text
host make / Runner
  |
  | GOMADSEED unset
  v
patched cmd/go run|test -exec tools/gomadv3/exec.sh
  |
  | GOMADV3_CHILD_SEED=<seed>, CGO_ENABLED=0, TZ=UTC
  v
exec.sh validates its child-only input and execs target with GOMADSEED=<seed>
  |
  v
target binary or generated test binary: deterministic region
```

The wrapper is deliberately small: accept no shell command string, preserve the
argument vector exactly, require the child seed variable to be present, remove
the child-only variable, set `GOMADSEED`, and `exec` the supplied target. The
runtime remains the single canonical parser for empty, malformed, and
overflowing values. The wrapper must not use `eval`. Direct execution of a
prebuilt binary remains
`GOMADSEED=<seed> ./binary ...`.

Set `CGO_ENABLED=0` for target compilation and reject enabled-mode cgo at runtime
as a defense in depth. The custom linker marks external links for the runtime,
including Darwin/amd64 links that do not load `runtime/cgo`. Upstream's faketime
test documents that external linking and cgo bypass the `checkdead` advancement
assumption. A stable early error is better than a hang or partially
deterministic execution.

The supported deterministic program consumes no host input or readiness.
Stdout/stderr may be captured only as non-semantic diagnostics: program choices
must not depend on their timing or consumer behavior, and the Runner must drain
and bound them. A stricter zero-host-syscall mode belongs to the later adapter
work and is not claimed by this clock feature.

## Pattern Survey

### Analogous Features
- `tools/gomadv3/README.md:3` — v3 is already an opt-in patched Go 1.26.4 toolchain that preserves native `go run`, `go test`, goroutines, channels, `select`, maps, and sync.
- `tools/gomadv3/README.md:32` — `GOMADSEED` is the single activation gate; enabled mode forces initial `GOMAXPROCS=1`, disables async preemption, and seeds runtime choice paths before user initialization.
- `tools/gomadv3/README.md:38` — the current repeatability contract is fixed toolchain/arch/program/seed plus deterministic external inputs, with host timers and host I/O readiness outside the contract.
- `go1.26.4/src/runtime/time_nofake.go:11` — the normal runtime build already has a process-global `faketime` variable where zero means “off,” making the dormant path compatible with opt-in activation.
- `go1.26.4/src/runtime/time_fake.go:16` — the `faketime` build-tag variant demonstrates process-global simulated time, `nanotime`, `time.now`, and stdout/stderr playback framing.
- `go1.26.4/src/runtime/proc.go:3739` — `findRunnable` already treats faketime specially by polling netpoll without wall delay and stopping the M so `checkdead` can advance time.
- `go1.26.4/src/runtime/proc.go:6437` — `checkdead` already jumps `faketime` to the next timer wake, obtains an idle P/M, and wakes timer work instead of declaring deadlock.
- `go1.26.4/src/runtime/time.go:1319` — `timeSleepUntil` already scans all P timer heaps for the next timer deadline used by sysmon and `checkdead`.
- `go1.26.4/src/runtime/time_test.go:19` — upstream's faketime test builds a dedicated test program with `-tags=faketime` and notes that advancement depends on `checkdead` and internal linking.
- `go1.26.4/src/runtime/testdata/testfaketime/faketime.go:16` — the test fixture shows process-global faketime ordering stdout/stderr frames and advancing through `time.Sleep`.
- `go1.26.4/src/runtime/proc.go:179` — `runtime.main` locks the main goroutine to the main OS thread through initialization and later calls `main_main` directly.
- `go1.26.4/src/runtime/synctest.go:170` — synctest creates a bubble by spawning the function as a new goroutine, which is heavier than activating process-global faketime for the already-running program.
- `go1.26.4/src/testing/synctest/synctest.go:274` — public `synctest.Test` executes a function in a new bubble and must not itself be called from within a bubble.
- `service/worker/pernamespaceworker_test.go:52` — Temporal tests already use `testing/synctest`, so a process-wide Gomad bubble would conflict with existing nested synctest tests.
- `service/matching/workers/registry_impl_test.go:427` — existing tests rely on `synctest.Test` around real `time.NewTicker`/`time.Sleep` behavior.
- `common/testing/testcontext/context_test.go:21` — existing test helpers already use `synctest.Test` for virtual-time context deadlines.
- `go1.26.4/src/runtime/time.go:167` — fake timers with identical deadlines are ordered by a per-timer random value, which aligns with Gomad's seeded tie-breaking needs.
- `go1.26.4/src/runtime/time.go:16` — synctest bubble fake clocks still take precedence for bubbled goroutines, making synctest compatible nested prior art.
- `GOMADv3_NEXT.md:31` — the post-v3 design is already organized around Runner, World, Adapters, and Record as deep modules with small interfaces.
- `GOMADv3_NEXT.md:140` — deterministic adapters already define the no-host-I/O boundary for filesystem, persistence, network, processes, environment, and entropy.
- `tools/gomadv2/doc.go:50` — gomadv2 achieved transparent standard-library behavior through runtime/syscall-level Linux emulation, which is broader than the lighter v3 faketime substrate.
- `common/clock/event_time_source.go:11` — Temporal already has a synchronous fake `TimeSource` for DI-backed tests, but it is not transparent for unmodified binaries.
- `service/history/workflow/timeskipping.go:211` — workflow time skipping already wraps mutable state's `TimeSource` with a live virtual-time offset.

### Reusable Utilities
- `tools/gomadv3/overlay/src/runtime/gomad.go:10` — `gomadInit` — owns seed parsing, feature activation, async preemption disabling, sysmon disabling, and scheduler randomization.
- `tools/gomadv3/overlay/src/runtime/gomad.go:29` — `gomadStartUserCode` — resets per-M seeded randomness and scheduler tick state immediately before user code.
- `go1.26.4/src/runtime/time_nofake.go:15` — `faketime` — dormant process-global nanosecond clock whose zero value keeps normal runtime behavior.
- `go1.26.4/src/runtime/time_nofake.go:32` — `nanotime` — normal wrapper around `nanotime1`; the smallest activation surface for process-global faketime is here and `time_runtimeNow`.
- `go1.26.4/src/runtime/time_fake.go:40` — `time_now` — build-tag prior art for returning process-global faketime through `time.Now`.
- `go1.26.4/src/runtime/proc.go:6375` — `checkdead` — existing quiescence point that advances process faketime to the next timer rather than sleeping on host time.
- `go1.26.4/src/runtime/proc.go:3397` — `findRunnable` — existing scheduler path whose faketime branch prevents host netpoll delay from deciding timer progress.
- `go1.26.4/src/runtime/time.go:1322` — `timeSleepUntil` — shared next-deadline scanner for process faketime advancement.
- `go1.26.4/src/runtime/time.go:167` — `timerWhen.less` — existing equal-deadline fake-timer randomization hook.
- `go1.26.4/src/runtime/time.go:389` — `newTimer` — runtime allocation path for `time.Timer`/`time.Ticker`; already marks synctest timers fake when created inside a bubble.
- `go1.26.4/src/runtime/time.go:1009` — `(*timers).check` — shared timer runner that can run ready timers from either normal P timers or synctest bubble timers.
- `go1.26.4/src/runtime/synctest.go:170` — `synctestRun` — compatible nested prior art for isolated bubble fake time, not the preferred whole-process substrate.
- `go1.26.4/src/runtime/synctest.go:282` — `synctestWait` — runtime implementation of durable-blocking wait for existing tests that explicitly enter synctest.
- `tools/gomadv3/testlib.sh:3` — `gomad_run_checked` — runs child processes with stdout/stderr/status capture, process-group timeout, and diagnostic reporting.
- `tools/gomadv3/build.sh:191` — `publish_toolchain` — publishes a stable `.toolchain/bin/go` launcher keyed to the immutable patched build.
- `tools/gomadv3/test.sh:34` — `validate_runtime_path` — enforces the small runtime-patch/overlay surface that can still include top-level `runtime/time*.go` files.
- `common/clock/time_source.go:9` — `TimeSource` — existing injectable clock abstraction for Temporal code that already has seams.
- `common/clock/event_time_source.go:126` — `AdvanceNext` — existing earliest-timer advancement primitive for non-transparent fake-time tests.
- `service/history/interfaces/mutable_state.go:420` — `Now` / `ToRealTime` — existing virtual-time versus wall-time conversion contract in mutable state.
- `service/history/workflow/timeskipping.go:432` — `findNextSkipTarget` — existing Temporal logic for selecting the earliest future virtual-time target.

### Convention Anchors
- At design time, the inputs included the then-uncommitted v3 runtime and test harness changes now recorded in this branch.
- Process-global faketime over process-wide bubble: `runtime/time_nofake.go:15`, `runtime/proc.go:6437`, and `runtime/time.go:1319` provide a lighter virtual-time substrate that preserves normal process/main structure.
- Dormant means not yet active: `runtime/time_nofake.go:32` still returns host `nanotime1` in the normal build, while `runtime/time_fake.go:40` shows the build-tag version's `time.now` behavior.
- Main-thread/init preservation: `runtime/proc.go:179` locks `runtime.main` to the main OS thread through init, and `runtime/proc.go:293` calls `main_main` directly; a whole-process synctest wrapper would not match this shape.
- Synctest compatibility: existing Temporal tests call `testing/synctest`, and `testing/synctest/synctest.go:279` forbids calling `synctest.Test` from inside another bubble.
- Small patch discipline: `tools/gomadv3/test.sh:34` limits patch/overlay files to top-level `src/runtime/*.go` and excludes channel, select, sema, GC, netpoll, OS, signal, platform, generated, and binary areas.
- Activation compatibility: `tools/gomadv3/README.md:32`, `tools/gomadv3/test.sh:473`, and `tools/gomadv3/test.sh:660` keep disabled mode stock-compatible and enabled mode solely `GOMADSEED`-gated.
- Separate-process exploration: `tools/gomadv3/README.md`, `GOMADv3_NEXT.md`, and `tools/gomadv3/test.sh` all assume one P per process and parallelism by running multiple child processes.
- No host I/O in deterministic region: `tools/gomadv3/README.md`, `GOMADv3_NEXT.md`, and `testing/synctest/synctest.go` all draw the same boundary: host file/socket/DNS/process readiness is not deterministic input.
- Faketime output framing is optional prior art: `runtime/time_fake.go:45` frames stdout/stderr for playground playback, but v3 already captures child stdout/stderr in the Runner harness.
- Internal-linking/cgo constraint: `runtime/time_test.go:24` documents that faketime advancement depends on `checkdead`, and v3 already keeps cgo outside the deterministic contract.
- External-choice stream separation: `GOMADv3_NEXT.md` keeps World tie-break randomness separate from the runtime's private stream, while `runtime/time.go` shows synctest fake timer ties consume `cheaprand`.
- Temporal seam convention: `common/clock/time_source.go:9` and `service/history/workflow/timeskipping.go:211` support DI-backed pilots, but transparent unmodified binaries require runtime/stdlib time interception.
- Prior-art boundary: `tools/gomadv2/doc.go:50` achieved transparent standard-library behavior through Linux syscall emulation, while `GOMADv3_NEXT.md` keeps post-v3 external behavior outside the Go runtime where possible.
- Test harness convention: `GOMADv3_TESTS.md` and `tools/gomadv3/testlib.sh` keep behavior fixtures outside the patch/overlay and require bounded, status-checked child execution.

### Proposed Alignment
Use the dormant process-global faketime path as the primary lightweight virtual-time substrate: Gomad mode can activate `faketime` and route runtime/time reads through it, letting existing `checkdead` plus `timeSleepUntil` advance standard `time` timers when the process is quiescent without running `main` inside a synctest bubble. Keep `testing/synctest` as compatible nested prior art for tests that explicitly opt into bubbles, and reuse its seeded equal-deadline fake-timer randomization where simultaneous Gomad deadlines need a runtime tie-break. Continue to put filesystem, network, process, environment, entropy, and persistence readiness behind World/adapters/records so process faketime solves transparent time only, not host I/O.

## Alternatives

### A. Activate dormant process-global faketime — recommended

Use `faketime`, `checkdead`, `timeSleepUntil`, and native timer heaps already
compiled into the ordinary runtime, adding Gomad-gated clock reads and seeded
same-deadline ordering.

Advantages:

- Smallest transparent implementation surface.
- Covers package init, normal binaries, and the standard test harness.
- Reuses mature timer stop/reset/ticker/context behavior.
- Preserves normal main-goroutine and main-thread initialization semantics.
- Coexists with tests that explicitly use synctest.
- No new public API, controller goroutine, queue, source rewrite, or dependency.

Costs and risks:

- Depends on internal Go runtime timer and deadlock machinery and must be
  revalidated for every pinned Go upgrade.
- Requires internal linking and no cgo.
- Quiescence is process-wide; one runnable polling goroutine intentionally
  prevents clock advancement.
- Does not virtualize any I/O completion source.

### B. Run the process in a synctest-derived bubble

Create a private whole-process bubble using the runtime synctest machinery, and
run user initialization/main or each test within it.

Advantages:

- Reuses Go's explicit durable-blocking model, private timer heap, fake clock,
  and established synctest semantics.
- Makes the virtual-time mechanism conceptually isolated from global faketime.
- Provides an explicit `Wait`-style quiescence primitive if future control APIs
  need one.

Costs and risks:

- `synctestRun` starts the target function in a new goroutine; wrapping
  `runtime.main` risks changing its locked-main-thread initialization contract.
- Public `synctest.Test` changes test semantics and cannot be nested. Temporal
  already contains tests that call it, so a process-wide bubble would require
  nested bubble support or special escape behavior.
- Covering package init and the `testing` harness would require patches outside
  the narrow runtime hook or a more invasive runtime main refactor.
- Adds bubble membership and durable-block accounting where the global
  `checkdead` path already supplies the needed process-level proof.

Choose this only if a minimized test demonstrates that process faketime cannot
express a required, supported blocking state and the additional main/nesting
complexity has a tested solution.

### C. World-backed injected clock

Use Temporal's existing `common/clock.TimeSource`, or a new small World clock
interface, and inject it into application components and tests.

Advantages:

- No Go runtime timer patch.
- Clock state and timer queue can be tested in isolation.
- Naturally composes with future filesystem/network/process World events and a
  separate external-choice random stream.
- Best choice for code that already has a clock seam.

Costs and risks:

- Does not meet the requirement for arbitrary unmodified Go code.
- Misses package init, third-party uses of package `time`, standard-library
  context deadlines, and the testing harness unless every call path is adapted.
- A repository-wide conversion would be much larger than the runtime change
  and could leave mixed real/virtual clock bugs.

Retain this for unit tests and future external adapters, but not as Gomad's
transparent process clock.

### D. Rewrite or intercept package `time`

Transform source, use a compiler hook, or overlay package `time` so its public
functions call a Gomad controller.

Advantages:

- Could keep a userland event queue and policy.
- May avoid direct changes to some scheduler paths.

Costs and risks:

- Source rewriting cannot reliably cover compiled dependencies, package init,
  linknamed monotonic-time users, or arbitrary binaries.
- An overlay of package `time` still depends on runtime timer hooks and is
  broader than changing the existing runtime wrappers directly.
- A controller introduces new scheduling, wake-up, reentrancy, and deadlock
  problems.
- Greater maintenance surface with no benefit for the no-I/O clock-only goal.

Reject unless runtime faketime proves unusable and a specific interception seam
can be shown to cover all standard time behavior.

### Comparison

| Option | Unmodified binaries | Unmodified `go test` | Existing synctest tests | Patch/adapter size | Future World composition |
| --- | --- | --- | --- | --- | --- |
| A. Process faketime | Yes | Yes | Compatible by design | Small | Clock must later coordinate |
| B. Whole-process bubble | Risk around init/main | Risk around harness | Conflicts without nesting | Medium/high | Private bubble queue |
| C. Injected World clock | No | Only adapted tests | Compatible | Small core, large adoption | Best |
| D. Rewrite/intercept | Incomplete or tool-specific | Incomplete | Uncertain | High | Controller-specific |

## Planned file impact

| File | Purpose |
| --- | --- |
| `tools/gomadv3/overlay/src/runtime/gomad.go` | Activate the fixed process clock under the existing seed gate and reject cgo |
| `tools/gomadv3/go1.26.4.patch` | Patch runtime clock reads, quiescence, timer tie ordering, and the linker external-mode marker |
| `tools/gomadv3/exec.sh` | Transfer the child-only seed to a binary or generated test binary without activating `cmd/go` |
| `Makefile` | Use `-exec`, unset parent `GOMADSEED`, disable cgo, set UTC, and preserve `test_dep` |
| `tools/gomadv3/test.sh` | Add bounded enabled/disabled black-box clock coverage |
| `tools/gomadv3/testlib.sh` and `testlib_test.sh` | Reuse and test process-group wall-time containment and wrapper behavior |
| `tools/gomadv3/testdata/clock*` | Keep behavioral fixtures outside the runtime patch/overlay |
| `tools/gomadv3/README.md` | Publish the supported clock contract and failure boundary |
| `GOMADv3_NEXT.md` and `GOMADv3_TESTS.md` | Record the evidence-backed runtime exception and updated test boundary |

No production Temporal package or `common/clock` file is expected to change.

## Detailed implementation plan

### Phase 0: prove the runtime premise with a disposable spike

Before changing the checked-in patch, make a local toolchain experiment against
the pinned Go 1.26.4 source:

1. Set `faketime` to the fixed base in `gomadInit` after seed validation.
2. Route Gomad-mode `nanotime` and `time_runtimeNow` through it.
3. Run a minimal binary that records `time.Now`, sleeps 24 logical hours, and
   records `time.Now` again under the existing real-time watchdog.
4. Run a minimal `go test` case with native timers and a separate case that
   calls `synctest.Test`.
5. Confirm the target is internally linked with cgo disabled.

The premise is accepted only if the binary completes quickly, observes exactly
24 hours of logical elapsed time, disabled mode still sleeps on host time, and
the nested synctest case retains upstream behavior. If it fails, minimize the
runtime state that prevents `checkdead` from advancing before choosing option B.
Do not commit the disposable spike or broaden the patch during diagnosis.

### Phase 1: add test fixtures before production changes

Add black-box fixtures under `tools/gomadv3/testdata`, keeping behavior tests
outside the runtime patch and overlay:

- `clock/main.go`: fixed initial time; frozen time during runnable work; long
  sleep; timer, callback, ticker, stop/reset, context deadline/cancel, and edge
  cases; emits a compact deterministic result only after assertions pass.
- `clock_gotest/clock_test.go`: the same public behavior through the standard
  test binary, including `T.Deadline` and logical `-test.timeout` behavior.
- `clock_synctest/clock_test.go`: explicit `synctest.Test` under global Gomad
  mode, proving bubble time takes precedence and no nesting panic is introduced.
- `clock_race/main.go`: many simultaneous deadlines whose complete order is
  compared across fresh same-seed processes and across a bounded seed set.
- `clock_spin/main.go`: a runnable busy loop that must end only through the
  Runner's wall timeout, proving virtual time does not skip runnable work.
- `clock_deadlock/main.go`: blocked goroutines with no future timer, proving the
  outcome differs from the spin timeout.

Use `require` in Go tests. Shell assertions must use `gomad_run_checked` for
bounded process-group execution, exact status checking, and stdout/stderr
diagnostics. Avoid `time.Sleep` in the host test harness; use its wall-time
watchdog facility.

### Phase 2: activate process faketime

Modify `tools/gomadv3/overlay/src/runtime/gomad.go`:

- define the fixed virtual clock base near the existing Gomad activation state;
- after successful `GOMADSEED` parsing, set `faketime` before any user package
  initialization;
- preserve every existing comment and the existing disabled path;
- reject enabled-mode cgo/external-link configurations at the earliest runtime
  point where the condition is reliable, with a stable diagnostic.

Modify the Go 1.26.4 patch for `src/runtime/time_nofake.go`:

- make `nanotime` return `faketime` only when Gomad is enabled;
- otherwise call `nanotime1` exactly as upstream does;
- retain the signature, linkname, and nosplit constraints.

Modify the Go 1.26.4 patch for `src/runtime/time.go`:

- preserve the existing synctest bubble check first in `time_runtimeNow` and
  `time_runtimeNano`;
- when unbubbled Gomad mode is active, return wall seconds/nanoseconds and a
  monotonic value derived from process `faketime`;
- preserve the upstream host-clock path when disabled;
- assign or reassign `timer.rand` for process-global Gomad timers when they are
  inserted into a heap, mirroring the existing synctest fake-timer behavior;
- leave timer public semantics, heap algorithms, callback execution, and
  `checkdead` unchanged unless the Phase 0 spike produces a minimized proof
  that one additional hook is necessary.

Do not copy `time_fake.go`, its stdout/stderr playback framing, or synctest's
bubble controller. Reusing the dormant state and existing scheduler branches is
the source of the design's small size.

### Phase 3: isolate `cmd/go` from the deterministic child

Add `tools/gomadv3/exec.sh` as the single child activation boundary:

- require `GOMADV3_CHILD_SEED` to be present and leave value validation to the
  runtime's existing unsigned 64-bit parser;
- require at least one target argument;
- unset `GOMADV3_CHILD_SEED`, set `GOMADSEED`, and `exec "$@"`;
- never invoke a shell command string or `eval`;
- return stable usage/validation diagnostics before starting a target.

Update root `gomadv3-run` and `gomadv3-test` targets:

- explicitly use `env -u GOMADSEED` so the invocation variable cannot reach the
  custom `go` process;
- pass the requested seed as `GOMADV3_CHILD_SEED`;
- pass `-exec $(ROOT)/tools/gomadv3/exec.sh` to both `go run` and `go test`;
- force `CGO_ENABLED=0` and `TZ=UTC` for the child contract;
- retain `-tags test_dep` for every test invocation;
- preserve arguments as argument vectors and document any current Make quoting
  limitations rather than adding `eval`.

Add wrapper unit tests to `tools/gomadv3/testlib_test.sh` or a focused shell
test: missing seed, malformed seed, seed zero, maximum uint64, missing command,
argument preservation, inherited `GOMADSEED` removal, target exit status, and
signal behavior.

### Phase 4: extend validation and documentation

Update `tools/gomadv3/test.sh` to:

- validate that the expanded patch touches only the allowed top-level runtime
  files and linker marker site and that disabled source remains upstream-compatible;
- run the clock fixtures in enabled and disabled modes;
- compare many fresh processes for same-seed equality;
- require at least one legitimate simultaneous-timer ordering difference across
  a bounded seed set without requiring every seed to differ;
- distinguish expected deadlock, logical test-timeout, and host-watchdog status;
- verify `go run`, `go test`, and direct prebuilt binary entry paths;
- verify the `go` parent is not in Gomad mode;
- verify cgo/external-link attempts fail with the documented stable result;
- keep output and process lifetime bounded.

Update `tools/gomadv3/README.md`:

- move timers into the supported deterministic contract;
- document the fixed initial instant, quiescence advancement, `go test` logical
  timeout, explicit synctest compatibility, cgo/internal-link requirement,
  `TZ=UTC`, and the external wall watchdog;
- state that polling loops and unsupported I/O prevent progress rather than
  causing virtual time to advance;
- retain the fixed-toolchain/program/architecture/seed reproducibility boundary.

Update `GOMADv3_NEXT.md` and `GOMADv3_TESTS.md` only where their architectural status
would otherwise become false: transparent time is an evidence-backed exception
to the earlier dependency-injection preference, while World remains the future
owner of external events. Avoid duplicating this complete clock contract there;
link to this document instead.

### Phase 5: pilot unchanged Temporal code

Run an unchanged timer-heavy package through `gomadv3-test`. Start with
`./common/timer`, whose local gate tests combine a real `TimeSource`, native
timers, selects, and deadlines. The pilot must not modify production or test
code to inject a Gomad clock.

Then select one unchanged package already using `synctest.Test` to prove that
global virtual time and local bubbles coexist. Expand the pilot only after both
small cases pass; broad repository runs would mix in unsupported I/O and obscure
the clock result.

## Test matrix

| Area | Required cases | Required result |
| --- | --- | --- |
| Activation | seed absent, zero, max uint64, malformed, empty, overflow | stock path when absent; deterministic early validation otherwise |
| Initial clock | init-time and main/test-time `time.Now` | midnight UTC 2000-01-01 until runnable work quiesces |
| Clock reads | `Now`, `Since`, `Until`, monotonic subtraction | exact repeatable logical durations |
| Advancement | long sleep, chained deadlines, runnable work before timer | quick host completion; no skipped runnable work |
| One-shot timers | `Sleep`, `NewTimer`, `After`, `AfterFunc` | upstream results at exact logical deadlines |
| Mutable timers | Stop before/after fire, Reset active/stopped/expired | upstream return values and no stale/duplicate delivery |
| Tickers | multiple ticks, slow receiver, Stop, reset | upstream coalescing and deterministic logical timestamps |
| Context | timeout, deadline, parent cancel, child cancel | exact cause and no timer leak/late delivery |
| Same deadline | channels, callbacks, contexts, timer reset | same seed repeats; seed set explores; every event completes once |
| Edges | zero, negative, near-overflow, overflow saturation, no future timer | upstream semantics; no wrap or host sleep |
| `go test` | package init, `T.Deadline`, logical timeout, multiple tests | deterministic harness behavior and bounded host execution |
| Nested synctest | explicit `synctest.Test`, `Wait`, timer operations | upstream bubble behavior; no nested-bubble panic from Gomad |
| Non-progress | busy loop, `select default`, unsupported blocking I/O fixture | host watchdog, never virtual timer delivery |
| Deadlock | all goroutines blocked, no timer | stable runtime deadlock distinct from watchdog timeout |
| Entry points | `go run`, `go test`, prebuilt executable | target activated; build driver not activated |
| Linking | cgo/external link attempt | rejected rather than partially deterministic |
| Disabled mode | representative runtime/time/synctest upstream tests | stock host-clock behavior |
| Real pilot | unchanged `./common/timer` and one synctest-using package | pass with `-tags test_dep` and no clock injection |

For ordering tests, compare semantic event identities rather than pointer values,
goroutine IDs, raw map order, or host timestamps. Run repetitions in fresh
processes because the reproducibility unit is a process.

## Error handling and failure classification

- Configuration errors: invalid child seed, unsupported cgo/linking, or malformed
  launcher use fail before user initialization with stable diagnostics and a
  nonzero status.
- Logical timeout: the virtual `testing` timeout fires at its deterministic
  deadline and retains the normal test failure form.
- Deterministic deadlock: runtime finds no runnable goroutine and no future
  timer; retain the normal fatal deadlock result.
- Host timeout: Runner kills the complete process group and records a distinct
  timeout status for busy loops, unsupported I/O, or runtime regressions.
- Runtime invariant violation: keep upstream `throw` behavior for impossible
  timer/P/M states; capture the full stderr in the reproduction artifact.
- Output limit: Runner drains both streams but applies a fixed bound and records
  truncation deterministically in metadata; child output never blocks progress.
- Toolchain failure: report separately from the target's result and include the
  immutable build key.

No error path may silently fall back to host time after Gomad activation.

## Performance, scale, complexity, and security

### Performance

- Disabled mode adds at most one predictable branch in runtime clock wrappers;
  benchmark it against the stock-pathed toolchain to catch a surprising hot-path
  regression.
- Enabled mode keeps native timer heaps: insertion/reset remains `O(log n)`,
  next-deadline lookup uses existing heap summaries, and storage remains `O(n)`.
- Time jumps remove host waiting but do not remove CPU work. Ten times as many
  timers costs approximately `O(10n log(10n))` timer work and `O(10n)` memory.

### Scalability

- One Gomad process still uses one P. Scale seed exploration across isolated
  processes with bounded Runner concurrency.
- A large seed set grows work approximately linearly and must stream bounded
  results rather than retain every process output.
- One runnable goroutine can starve all virtual timers by design; the wall
  watchdog is the resource bound, not a scheduler escape hatch.

### Complexity

- The deep clock module is the existing runtime timer system. Gomad contributes
  only activation, clock-read selection, and seeded tie ordering.
- There is no second timer registry to reconcile and no public interface to
  version.
- The principal maintenance cost is auditing the pinned Go runtime at every
  version upgrade. Build validation and black-box semantics make drift explicit.

### Security

- Gomad remains for trusted tests only. Deterministic runtime randomness removes
  security hardening and must never be a production default.
- The exec wrapper treats arguments as an argv array and never evaluates text as
  shell code.
- No deterministic run receives host credentials or ambient external
  capabilities. The Runner should use an allowlist for preserved environment
  variables as that module matures.
- Rejecting cgo and external linking prevents unsupported foreign code from
  observing host time or holding runtime liveness outside Go's accounting.

## Relationship to World and records

For the no-I/O milestone, the native runtime timer heap is deliberately the
single ordered clock queue. Adding a second World timer queue now would duplicate
state and make quiescence coordination harder without serving the unmodified-code
requirement.

When deterministic external adapters are added, do not let their events advance
independently. Introduce a minimized coordination hook only after a concrete
adapter pilot establishes it. At each proven quiescence point, it must compare
the earliest native timer with the earliest World event, advance one shared
logical instant, and make every event at that instant eligible before scheduling
continues. World choice randomness remains domain-separated from the runtime's
private timer/scheduler stream.

Reproduction records should add clock policy and initial instant implicitly
through the immutable Gomad toolchain build key at first. Add explicit schema
fields only when more than one clock policy is supported. Records continue to
store seed, target identity, architecture, deterministic inputs, logical result,
wall watchdog result, and bounded diagnostics.

## Verification commands

Use the project's checked-in commands and always include `test_dep` for Go
tests. The implementation phase should finish with, in increasing scope:

```sh
make -C tools/gomadv3 test-harness
make -C tools/gomadv3 test
GOMADSEED=1 make gomadv3-run GOMADV3_RUN=./tools/gomadv3/testdata/clock/main.go GOMADV3_ARGS=initial
GOMADSEED=1 make gomadv3-test GOMADV3_PACKAGES=./common/timer
GOMADSEED=1 make gomadv3-test GOMADV3_PACKAGES=./common/testing/testcontext
make fmt-imports
make lint-code
```

If a chosen synctest pilot is an integration test, include both `test_dep` and
`integration`; otherwise do not add the integration tag. Run focused upstream
runtime/time and testing/synctest tests through the rebuilt custom toolchain as
part of `tools/gomadv3/test`.

## Completion criteria

- Unmodified binaries and unmodified test packages observe the fixed virtual
  clock and complete timer-heavy behavior without host waiting.
- Virtual time advances only after runtime-proven process quiescence.
- Every standard timer/context behavior in the matrix matches upstream public
  semantics.
- Same-seed timer races repeat across fresh processes, and a bounded seed set
  demonstrates permitted diversity.
- Explicit nested synctest tests continue to work.
- Busy runnable code cannot cause a clock jump and is bounded by the external
  wall watchdog.
- `cmd/go` never enters deterministic mode; only the target does.
- Unsupported cgo/linking and invalid configuration fail early and stably.
- Disabled mode matches the stock Go 1.26.4 clock behavior.
- The patch remains within the validated top-level runtime surface plus the
  single linker marker site, and no third-party library or application clock
  rewrite is added.
- The toolchain suite, unchanged Temporal pilots, formatting, and linting pass.
