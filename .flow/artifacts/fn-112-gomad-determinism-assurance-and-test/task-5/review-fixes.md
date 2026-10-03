# fn-112 task 5: review follow-up (NEEDS_WORK on `c2fd6d2f0`)

Host: **linux/arm64 with the development harness** (development evidence only, not darwin/arm64 or
linux/amd64 evidence). Base `c2fd6d2f0`; new toolchain build key (harness on)
`8af6857bfddfc22398fcca013dd1b945d1ef15a059ca6b1fd8f5ec85e5c4cd54`. The patch was regenerated with
`gomadtool patch-regenerate --candidate-root=<materialized tree>`; no hunk was hand-edited.

## Findings and changes

1. **Netpoll batches drew seeded at host-timed moments (Medium).** `injectglist` -> `runqputbatch`
   shuffles a batch of two or more goroutines through `gomadChoiceShuffleSeeded`, and the netpoll
   paths (`startTheWorldWithSema`, the two `findRunnable` polls, `pollWork`, `sysmon`) inject while
   the M can hold the P. Rerouted: those six call sites now call `gomadInjectHostList` (overlay),
   which brackets the injection with `gomadHostTimedEnter/Exit` and raises `m.gomadHostBatch`
   (new field beside `m.gomadHostTimed`, patch `runtime2.go`); `gomadChoiceShuffleSeeded` then
   draws `gomadHostCheapRandN`. Target-originated batches (GC, cleanup, sweep waiters, arrival and
   global-queue admission) stay on the seeded stream. Inventory: new entry
   `gomadChoiceShuffleSeeded -> gomadHostCheapRandN` (host-timed, with reason); the `runqputbatch`
   reason names the caller-side switch; the file comment states that the inventory classifies the
   containing function, not its callers. This is a reroute: a run that injects a netpoll batch of
   two or more while holding the P can schedule differently from `c2fd6d2f0`; the new toolchain is a
   new identity. Conformance fixtures do not use netpoll, and their outputs are unchanged (see the
   runtime tier below).
2. **Green Tea GC (Medium).** The seeded runtime now refuses a binary built with
   `goexperiment.greenteagc`: `gomadInit` prints `runtime: GOMADSEED requires
   GOEXPERIMENT=nogreenteagc` and exits 2, right after the cgo/external-link check and before any
   seeded state exists; with Gomad disabled nothing changes. Direct target builds now set
   `GOEXPERIMENT=nogreenteagc`: the runtime campaign (`conformance/runtime_campaign.go`, every
   request), `make test-simulation`'s seeded `go test -exec`, the process-simulation integration
   build (`simulation_root_integration_test.go`), the live-capability toolchain tests, and the
   overlay's `gomadchoicewire` diagnostic runtime test. Prepared targets already used it
   (`target/internal/build`). Check (same build, scheduler fixture): default build + `GOMADSEED=1`
   -> exit 2 with the message; same binary unseeded -> exit 0; `nogreenteagc` build seeded -> exit 0.
3. **The fault bracketed itself (Low).** `gomadHostCheapRand`'s host-timed fault now only routes the
   draw to `gomadRuntimeCheapRand`; it no longer raises `m.gomadHostTimed`. The fault stays armed,
   so the first later host-timed draw inside a real bracket stops the process. `draw_check` gained an
   idle window (`time.Sleep(time.Millisecond)` after its workers) so the steal pass runs before it
   prints. Proof that the bracket is now load-bearing (direct runs with choice and diagnostic traces,
   seed 7): normal build, `host-timed:5` -> exit 125 with the message, no stdout; the same fixture
   built with `-overlay` replacing `runtime/proc.go` by a copy without the steal-pass bracket ->
   exit 0, `120 148`. Brackets now: the idle steal pass, the six netpoll injections, and the
   CPU-profiler setup in `execute` (finding 6). The draft contract sentence in
   `evidence-summary.md` is narrowed to those brackets.
4. **"Pins GOMAXPROCS" (Low).** Reworded in the inventory (`enlistWorker` reason) and in
   `evidence-summary.md` Q3 and the blocked list: unreachable while `GOMAXPROCS` stays one (Gomad
   starts with one; raising it is unsupported, not prevented).
5. **Inventory enumeration (Low).** The scan also counts direct uses of `gomadRuntimeRandom`,
   `gomadRuntimeCheapRandom`, `gomadTimerRandom`, `gomadClockTickState`, `gomadChoiceSelectRandom`,
   `gomadChoiceRunqRandom`, `gomadChoiceSchedulerRandom` (declared names are not uses); every use
   must be an `implementation` entry, and today they are exactly the accessors plus
   `gomadChoiceSeedRandom` and `gomadClockTickInit` (15 new keys). A new negative case requires the
   "seeded state is used outside its accessor or seeding function" failure. The compiler scan now
   walks all of `cmd/compile` except `internal/typecheck/builtin.go` (and `_builtin`); it found no
   by-name call beyond `walk/builtin.go walkMakeMap rand`.
6. **os_linux.go wording (Nit).** The reason now says the `os_` prohibited runtime area, and
   `execute` brackets its `setThreadCPUProfiler` call, so a diagnostics run that turns on CPU
   profiling stops with exit 125.

## Gates

See `review-fixes-gates.md`.
