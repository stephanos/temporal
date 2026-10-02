# Feasibility: schedule-search and runtime-side ideas in GOMAD_CMP.md

Dated research snapshot (2026-10-01); read-only code study, not a statement of current support.

Date: 2026-10-01. Read-only study of `/Users/stephan/Workspace/temporal/gomad` (branch `stephanos/gomad`).
Nothing was built or executed against a target; timing numbers come from retained reports.

Conventions

- Paths are relative to the repo root. `patch:N` means line N of
  `tools/gomad3/toolchain/runtime/go1.27.1.patch`. `gomad.go:N` means
  `tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go`.
- "built source" means the already-materialised patched Go 1.27.1 tree at
  `tools/gomad3/.toolchain/builds/8d28bd44…/src` (key `8d28bd44` is the one the milestones cite).
  I only read it.
- **[V]** = verified by reading the cited code or retained report. **[I]** = inferred (reasoning
  from code I read plus upstream Go behaviour, not confirmed by an execution).
- Size: S = days, M = 1–2 weeks, L = a month or more.

---

## 0. Findings that change the framing

These six facts are what I would weigh before picking any item from the idea list.

1. **Under Gomad every goroutine switch is voluntary. There are no preemptions to bound.** [V]
   Activation sets `debug.asyncpreemptoff = 1`, `haveSysmon = false`, `randomizeScheduler = true`
   (`gomad.go:118-120`); `retake` returns 0 (`patch:533-538`). A recorded Runnable decision only
   happens inside `runqget` after the current goroutine has already left the P
   (`patch:578-594`). So the current goroutine is never an alternative at a decision. Preemption
   bound 0 is the *entire* space the runtime offers today. Preemption bounding and PCT's
   priority-change points only become meaningful after new yield points exist. The CMP ordering
   ("visibility before search sophistication") is right, and stronger than the doc states.

2. **`runnext` is dead under Gomad, so the run-queue choice really is the full local enabled set.** [V]
   Upstream `runqput` has `if !haveSysmon && next { next = false }` (built source
   `runtime/proc.go:7701`, visible as context in `patch:549-554`). Gomad turns `haveSysmon` into a
   variable and clears it (`patch:503-508`, `gomad.go:119`). Every readied or newly created
   goroutine goes to the tail of the local run queue and `runqget` picks uniformly over the whole
   queue. The patched `gomadChoiceRunnextSeeded` coin (`patch:551-558`, `gomad.go:453-458`) is
   unreachable in Gomad mode.

3. **A choice record does not contain the alternatives, the running goroutine, or why the previous
   goroutine stopped.** [V] A record is 96 bytes: ordinal, kind, flags, alternative *count*,
   selected *rank*, `data`, site, the selected identity, and a SHA-256 of the sorted alternative
   set (`gomad.go:313-323`, `595-608`). Runnable records always have `SiteMissing` and `data = 0`
   (`gomad.go:801`). This is why the explorer forces "rank N" rather than "goroutine X"
   (`choice/tape.go:207-234`). PCT, preemption accounting, DPOR and timelines all need data that
   the runtime has in hand at that point but does not write.

4. **Raw BFS only ever flips the first `--max-choice-depth` decisions of the run.** [V]
   `expandCandidate` skips a decision when `ordinal+1 > MaxChoiceDepth`
   (`runner/internal/exploration/choice/engine.go:357-361`). A functional suite has 8.6k–58k
   branching decisions per execution (section 3). With depth 32 the explorer permutes process
   start-up and never reaches test logic. The primitive underneath (`BuildRankPrefix` at any
   ordinal) does not have this limit; the BFS controller does.

5. **About half of the branching decisions are select poll-order shuffles, most of which cannot
   change behaviour.** [V for the mechanism and counts, I for "most"] The hook sits inside the
   Fisher-Yates loop that builds `pollorder` for *every* `selectgo` call (`patch:689-698`), before
   readiness is known. An n-case select emits n-1 decisions whether zero, one or all cases are
   ready. In `TestSignalWorkflowTestSuiteChasm` seed 11: 57,801 decisions = 30,936 runnable +
   26,865 select-poll
   (`.flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/fn105-d14-qualification-seed11.json`,
   `evidence.choices`). Poll order only matters in selectgo's pass 1 when at least two cases are
   ready.

6. **The "no advantage over seed sampling" finding is a property of the test's metric.** [V]
   See section 3. It asserts both strategies see 2 outcomes in 16 executions; it does not measure
   executions-to-discovery, and the fixture has one goroutine and one meaningful decision.

---

## 1. Scheduling visibility

### (a) What exists

**Recorded, tape-forcible decisions (two kinds).** [V]

| Kind | Where | Alternatives | Identity |
| --- | --- | --- | --- |
| `Runnable` (1) | `runqget`, when `gomaxprocs == 1` and the local queue holds more than one G (`patch:578-594`; `gomad.go:785-802`) | every G in the local run queue, max 256 | per-goroutine `gomadIdentity` |
| `SelectPoll` (2) | each Fisher-Yates step of `selectgo` poll order (`patch:689-698`; `gomad.go:804-819`) | insertion position among cases seen so far | hash(select call-site text offset, case ordinal, send/recv) (`gomad.go:821-834`); not goroutine-specific |

Kinds are enumerated in `overlay/src/runtime/gomad_choicewire_generated.go:14-16`. A third kind,
`SelectResult` (3), is an *observation* written at `retc` with the chosen case
(`patch:708-721`); replay compares it but cannot force it.

**Seeded but unrecorded choices (seed changes them, a tape cannot).** [V]

- Timer tie-break at equal deadlines: `gomadTimerRand`, its own stream
  (`gomad.go:363-373`; `patch:774-810`).
- `runqputslow` / `runqputbatch` shuffles (`patch:560-577`; `gomad.go:460-465`). Physical order
  only; the pick is by identity, so this does not change the alternative set.
- Map hash seeds, `rand`/`cheaprand` consumers on the P (`patch:614-647`; `gomad.go:375-404`).
- `time.Now` forward ticks (`gomad.go:123-170`).
- World equivalent-event order: stateless HMAC rank, never a runtime choice
  (`tools/gomad3/world/choice.go:9-20`).

**Not choices at all (deterministic policy).** [V]

- GC mark worker takes the slot whenever work is queued (`patch:206-212`; `gomad.go:890-897`);
  idle-time marking is off (`patch:236-241`).
- Global-queue goroutines and syscall arrivals are admitted only when the local queue is empty
  (`patch:216-232`, `245-266`; `gomad.go:935-953`). The `schedtick%61` fairness check is disabled.

**Scheduling opportunities under this profile** (points where the running goroutine can lose the P):

| Operation | Yields? | Evidence |
| --- | --- | --- |
| Blocking chan send/recv, blocking `select`, `sync.Mutex` slow path, `WaitGroup.Wait`, `Cond.Wait`, `time.Sleep`, modeled net/pipe waits | yes, `gopark` | [I] upstream; consistent with `sync`/`channels` fixtures |
| Goroutine exit | yes | [I] |
| `runtime.Gosched` | yes, but the yielder goes to the **global** queue (built source `proc.go`, `goschedImpl` → `globrunqput`) and re-enters only once the local queue is empty | [V] |
| `sync.Mutex.Unlock` in starvation hand-off | yes, `goyield` to local tail (built source `runtime/sema.go:286`) | [V] |
| Runner-answered blocking syscalls (`gomadBlockingRead/Write`, `gomadHostRead`) | yes, `entersyscallblock` hands off the P (`gomad.go:1191-1236`); return is queued as an arrival | [V] |
| Allocation | **sometimes**: GC start blocks on `<-ready` while starting workers (built source `runtime/mgc.go:1711-1721`), and an assist without credit parks (`runtime/mgcmark.go:822`) | [V] |
| Function-prologue preemption check | the check is compiled in, but nothing sets the flag: `retake` is off, `enlistWorker` returns for `gomaxprocs <= 1` (built source `runtime/mgcpacer.go:726-728`) | [V] |
| `sync/atomic` operations | never | [V] intrinsics, `cmd/compile/internal/ssagen/intrinsics.go:1296-1343` |
| Uncontended `Mutex.Lock`/`Unlock`, `RWMutex`, `Once`, `WaitGroup.Add/Done` | never (CAS fast path, built source `internal/sync/mutex.go:61-71`) | [V] |
| Non-blocking chan ops, `select` with a ready case or `default`, `close` | never; they may ready another G, the caller keeps running | [I] |
| CPU loop | never; documented hang (`tools/gomad3/README.md:762-766`) | [V] |

**Can a goroutine be descheduled between two atomic operations today?** Not by the scheduler.
Only if the code between them blocks, calls `Gosched`, does blocking modeled I/O, or allocates at a
moment when the collector parks it. The last case is real and heap-layout dependent, so it is
deterministic for a fixed binary and seed but is not something a tape can target. [V]

The repo already carries the witness for the missing boundary: the `preemption` fixture spins on
`atomic.Bool` and the conformance driver *requires* it to time out under a seed
(`tools/gomad3/internal/gomadtool/conformance/testdata/preemption/main.go`;
`conformance/runtime_repeatability.go:106-124`). The other scheduling fixtures (`sync`,
`scheduler`, `runqueue`, `channels`) all insert explicit `runtime.Gosched()` to obtain
interleavings. [V]

### (b) Gaps

1. No scheduling point at atomics or uncontended sync fast paths; lost-update and
   check-then-act witnesses are unreachable.
2. No scheduling point at wake-ups. After `Unlock`, `close`, `cancel()`, `wg.Done()`, a
   non-blocking send, the waker always runs on to its next blocking point. "Wakee runs before the
   waker's next statement" is a legal multi-P Go behaviour that Gomad cannot produce. This is the
   bigger hole for the CMP witness list (cancellation/completion, lock hand-off, close/send). [I]
3. Timer ties are not a recorded decision (they do end up as a Runnable choice once the woken
   goroutines are queued, but timer *callback* order is seed-only).
4. Goroutines parked in the global queue (after `Gosched`, or overflow beyond 256 runnable) are
   absent from the alternative set until the local queue drains. [V] mechanism; [I] that overflow
   occurs in suites with ~1,900 peak goroutines.

### How the existing compiler interception works, and whether it can host these hooks

- `gomadintercept.Apply` and `gomadguard.Apply` run once per compiled package right after the
  package is loaded and before inlining (`patch:14-21`; built source
  `cmd/compile/internal/gc/main.go:230-231`). [V]
- `gomadintercept` is **callee-side body rewriting**: for each spec whose `PackagePath` equals the
  package being compiled it finds the target function and a hook function *in the same package*,
  marks the target `Noinline`, and prepends `results..., handled := hook(args...); if handled {
  return results }` (`overlay/src/cmd/compile/internal/gomadintercept/intercept.go:53-134`,
  `325-361`). It aborts when the target has no Go body (`intercept.go:83-85`). Specs are generated
  from the boundary manifest (`gomadintercept/spec_go127.go`, source
  `tools/gomad3/deterministicio/boundary/manifest.json`).
- `gomadguard` shows the other half that would be needed: it prepends a call to a **runtime**
  symbol from any package (`overlay/src/cmd/compile/internal/gomadguard/guard.go:21-43`,
  `71-84`).

Consequences [V for the mechanism, I for the applicability judgement]:

| Target | Coverable by the existing callee-prologue mechanism? |
| --- | --- |
| `(*sync.Mutex).Lock/Unlock`, `RWMutex`, `WaitGroup`, `Once`, `Cond` | Yes. They have Go bodies (`sync/mutex.go:45-47` wraps `internal/sync`). Cost: the method becomes non-inlinable everywhere, including inside the standard library. |
| Typed atomics `(*atomic.Int64).Add`, `(*atomic.Bool).Load`, `atomic.Value` | Yes in principle: one-line Go bodies (`sync/atomic/type.go:19`, `128`). Marking them `Noinline` keeps the intrinsic from being reached via inlining at the caller. |
| `atomic.Pointer[T]` | Probably not: generic methods are instantiated in the caller's package, where the spec's package filter does not match. |
| Raw `atomic.AddInt64(&x, 1)` etc. | No. Body-less assembly stubs replaced by SSA intrinsics. Needs a **caller-side** pass. |

Temporal's non-test server code uses both styles in similar volume: 136 raw calls and 113 typed
declarations under `service common chasm components temporal` (grep count). A callee-only hook
would miss half. [V]

A caller-side pass does **not** need an SSA change. It can live in the same pre-inlining slot as a
new overlay package: walk each function body, find call expressions whose callee belongs to
`sync/atomic` (and optionally `sync`), and insert a call to a runtime hook before the enclosing
statement, exactly as `guardFunction` builds a runtime call. It also scopes naturally by *calling*
package, which is what keeps the standard library's own atomics out of the choice stream. The
alternative (disabling the intrinsics) leaves only assembly stubs to hook, and platform assembly is
a prohibited patch area.

Runtime side of such a hook: a new decision kind whose alternatives are "continue" plus the run
queue, followed by `goyield`-style requeue to the *local* queue. All of that is overlay code in
`gomad.go`; it needs no new patch hunk if it is reached through a linknamed entry point. It does
need a choice-wire version bump (`tools/gomad3/choice/schema/choicewire.json`), which changes the
choice implementation identity (`choice/trace.go:100-116`).

### Patch-policy constraints

- The patch may only touch files in `patch_allowlist`
  (`tools/gomad3/toolchain/version/version.json`; enforced in `toolchain/patch.go:106-127`).
  `proc.go`, `select.go`, `runtime2.go`, `time.go`, `gc/main.go` are already on it. `chan.go`,
  `sema.go`, `malloc*`, `mgc*`, and all platform files are classified prohibited
  (`toolchain/patch.go:262-285`). New overlay files must be added to `overlay_allowlist`
  (`patch.go:191-235`).
- "The collector patch prohibition remains in force" (`MILESTONES.md:124`).
- fn-110 wants the patch *smaller* and explicitly leaves "compiler/linker hook consolidation …
  and interception redesign" outside its scope (`MILESTONES.md:295-329`).
- Preemption bounding and DPOR are listed out of scope for the current milestones
  (`MILESTONES.md:369-370`); BUG-7 carries them as research
  (`.plans/GOMAD_NEXT.md:54-63`).

So: sync/atomic hooks are overlay-only on the compiler side and overlay-only on the runtime side.
Wake-up yield points need a few lines in `proc.go` (allowed, but adds to the patch fn-110 is
shrinking). Hooks inside channel or semaphore code are off the table under the current policy.

### (c) Smallest experiment

Three fixtures, each with a broken and a corrected variant, added to the conformance module:
(1) atomic load-then-store counter, (2) `mu.Unlock()` then a read of state the woken goroutine
mutates, (3) `cancel()` then completion bookkeeping. Run each for 1,000 seeds plus
`--strategy=choice-exploration`. Expected under today's runtime: zero failures for all three
([V] for the first by the `preemption` fixture's contract, [I] for the other two). That report is
the "visibility audit" stage of the CMP table and costs no toolchain work.

Then, in a throwaway toolchain build, prototype *one* hook: wake-up yield in `ready()` (section
8, N1). It is the cheapest and reaches witnesses (2) and (3).

### (d) Size and risk

- Audit fixtures and report: **S**. Risk: none.
- Wake-up yield point in `proc.go`: **S–M** prototype, **M** to qualify. Risk: every functional
  suite's schedule changes; choice counts rise; D12-style replay divergence gets more surface.
- Callee-side sync + typed-atomic hooks via the existing spec mechanism: **M**. Risk: making
  `Mutex.Lock` non-inlinable shifts timing and heap behaviour of every target; choice volume in
  the standard library.
- Caller-side atomic call-site pass: **M–L**. Risk: correctness of IR insertion across statement
  forms (atomics inside conditions, `defer`, closures), trace volume (64 MiB cap ≈ 699k records,
  `choice/session.go:11`; the Signal suite already needs 8.3 MiB), and a new execution profile
  that must be qualified separately.

### (e) Stale or contradictory relative to GOMAD_CMP.md

- CMP's claim that enumerating current choices misses the atomic witness is correct and already
  pinned by the `preemption` fixture.
- CMP does not mention wake-up boundaries; its witness list (cancellation/completion, lock
  hand-off, close/send) is mostly about them, not about atomics.
- CMP says hooks "must respect runtime critical sections, GC/write barriers". The existing
  cooperative-preemption path already enforces that (`canPreemptM` in `newstack`, built source
  `runtime/stack.go:1122-1174`), which is an argument for reusing it rather than writing a new
  yield path.
- README wording: "every non-selected runnable or ready-`select` rank"
  (`tools/gomad3/README.md:141-142`). The code enumerates poll-order positions for every
  multi-case select, ready or not. README is imprecise; CMP inherits it implicitly.
- README says equal-deadline timers "use the seeded runtime choice stream"
  (`README.md:700-701`). They use a separate, unrecorded stream (`gomad.go:363-373`). Relevant
  to CMP's "deadline/delivery ties" witness: a tape cannot flip a timer tie.

---

## 2. Preemption bounding, PCT, delay bounding

### (a) What the controller has at each decision

In-process, at the decision point (`gomadChoiceRunqIndex`, `gomad.go:785-802`) the runtime has
the P and the actual `*g` for every alternative, hence identity, `waitreason`, `gopc`, `startpc`,
parent, and anything else on the G. [V]

Offline, the Runner sees only the record described in 0.3.

- **Stable logical goroutine identities: yes, with one exception.** A child's identity is
  `H(parent identity, parent's child ordinal, go-statement text offset)`
  (`gomad.go:750-776`; hook at `patch:334-341`). That is independent of scheduling order as long
  as the parent's own spawn sequence is. Exception: goroutines created with no identified parent
  take a process-wide counter (`gomad.go:766-770`). `time.AfterFunc` callbacks are created from
  the scheduler's g0 (built source `time/sleep.go:181-183`), so every `context.WithTimeout`
  callback goroutine gets a creation-order identity that *does* depend on the schedule. [V] for
  the code path; [I] that g0 has a zero identity and therefore takes the counter branch.
- **Full enabled set: in-process yes, on the wire no.** Only the count and a digest of the sorted
  identities are written (`gomad.go:524`, `533-537`). No record is written when exactly one
  goroutine is runnable (`patch:582-585` requires `t-h > 1`), so the trace is a list of branch
  points, not a schedule.
- **Could the current goroutine have continued: never (see 0.1), and not recorded.** Runnable
  records carry no information about the goroutine that just left.
- Select records identify the site, not the goroutine executing the select.

### Is the policy pluggable?

There is a seam, and the decision does not have to come from a tape. [V]

- The patch computes `offset := gomadChoiceRunqSeeded(t-h)` then
  `offset = gomadChoiceRunqIndex(pp, h, t, offset)` (`patch:586-588`). Both functions are overlay
  code. `gomadChoiceDecision` treats the incoming index as "what the policy wants", records it by
  identity, and only overrides it while a tape has unconsumed records (`gomad.go:538-567`).
- A different in-process policy is therefore an overlay-only change inside
  `gomadChoiceRunqIndex`: compute the index from the `*g` array instead of using `seeded`.
  Recording and exact replay are unaffected because replay is by identity.
- Policy selection has a precedent: `GOMAD3_CLOCK_TICK` is read in `gomadClockTickInit`
  (`gomad.go:140-151`) and is bound into Campaign/Artifact identity through the recorded
  environment (`tools/gomad3/record/validation.go:514`; `runner/runner.go:1356`).
- Modes today: seed, record, replay, prefix (`gomad_choicewire_generated.go:24-27`). Prefix mode
  forces the tape and then falls back to the seeded stream (`gomad.go:539`).

### (b) What a PCT-like policy is missing

1. A priority per goroutine. Cheapest: derive it as `H(policy seed, gomadIdentity)`; no new `g`
   field, no patch. A mutable priority needs a field in `g` (`runtime2.go` is allowlisted; the
   existing identity fields cost a `sizeof_test.go` edit, `patch:648-676`, `724-735`).
2. Priority-change points. PCT lowers the running thread's priority at d-1 random *steps*. Steps
   here can only be scheduler entries, so a change point can reorder who runs after a block, but
   cannot interrupt a goroutine. Until yield points exist this is "random priorities with
   demotions", which is a different sampling distribution over the same schedules that uniform
   random already covers.
3. Wake-up semantics. PCT assumes a higher-priority thread runs as soon as it is enabled. Here a
   waker keeps the P (gap 2 in section 1).
4. Step counting for the depth/length parameters (`n`, `k`): the decision count is available
   (`gomadChoiceDecisionRecords`, `gomad.go:43`), total steps are not known in advance, so `k`
   must come from a prior recorded run of the same target (the Runner has it:
   `ChoiceTrace.Decisions`, `runner/choice_exploration_campaign.go:332`).
5. System goroutines are in the alternative set (section 8, N5); PCT needs a rule for them.
6. On the wire: nothing identifies the policy. A policy id and parameters must enter the
   execution identity (same route as the clock tick).

For preemption/delay bounding additionally: a record of "the running goroutine was offered a
yield and continued / yielded", which only exists once section 1's hooks do.

### (c) Smallest experiment

Overlay-only, no patch hunk: `GOMAD3_CHOICE_POLICY=priority` selects, inside
`gomadChoiceRunqIndex`, the alternative with the greatest `H(seed, identity)` with d-1 demotions
at seeded decision ordinals. Compare against uniform seeds at equal execution count on the
section-1 fixtures plus two that are reachable today (a blocking-order deadlock and an
ordering-dependent assertion across three goroutines). Report executions-to-first-failure
distributions. Expectation to test: without new yield points the gain is confined to bugs that need
one goroutine to be starved across many consecutive decisions, where uniform random is
exponentially unlikely.

### (d) Size and risk

- Priority policy in the overlay plus identity plumbing: **M** (toolchain rebuild, policy identity,
  qualification of one fixture set). Risk: neutral result because of 0.1; a strict priority
  scheduler can livelock targets that rely on fairness (a high-priority poller with backoff timers
  is fine, a high-priority `Gosched` loop is not, because the yielder leaves the local queue).
- Preemption or delay bounding proper: blocked on section 1; then **M** on top.

### (e) Stale or contradictory

- CMP: "Try preemption bounding after the controller distinguishes continuing, blocking, yielding,
  and exiting." The controller has nothing to distinguish yet. The dependency is on visibility,
  not on bookkeeping.
- CMP and BUG-7 ask for "stable actors, enabled sets". Stable identities exist; enabled sets exist
  only in-process. This is closer to done than the docs suggest for an *in-process* policy and
  further away for an *offline* controller.
- CMP: "An approximate priority scheduler does not inherit the paper's probabilistic guarantee."
  Agreed, and the reason is concrete: no immediate preemption on wake-up.

---

## 3. Existing exploration

### (a) What the BFS choice-prefix exploration does [V]

Controller: `tools/gomad3/runner/internal/exploration/choice/engine.go` (pure, 593 lines);
driver: `tools/gomad3/runner/choice_exploration_campaign.go`.

1. Root candidate = base seed, no prefix (`engine.go:157-168`). It runs in record mode
   (`choice_exploration_campaign.go:229-239`).
2. Each completed trace is projected to a Decision Tape: observations and single-alternative
   records are dropped and ordinals renumbered (`choice/tape.go:133-156`).
3. Expansion: for every decision at index ≥ the parent's forced depth, and every rank other than
   the selected one, build a child prefix = parent tape truncated after that decision, with that
   decision replaced by a rank override (`engine.go:349-384`; `tape.go:207-234`).
4. Children are deduplicated by candidate hash against `Seen`, then admitted until
   `MaxExecutions` (counted as seen prefixes, `engine.go:392-395`) or `MaxExplorationBytes` of
   queued candidates (`engine.go:403-409`) is hit. Decisions deeper than `MaxChoiceDepth` are
   counted as omitted (`engine.go:357-361`).
5. The queue is sorted by (forced depth, candidate hash) (`engine.go:550-560`); a round takes the
   first `Parallel` candidates (`runner/internal/exploration/engine.go:10-20`). Completions are
   committed in candidate order regardless of host finish order
   (`choice_exploration_campaign.go:248-272`), each round as a hash-linked segment
   (`engine.go:202-298`).
6. Outcome deduplication (stdout/stderr/transcript/World hashes,
   `choice_exploration_campaign.go:508-531`) only reduces retained evidence. There is no
   state-based or equivalence pruning.

**How a prefix is forced.** The child process gets `GOMAD3_CHOICE_MODE=3` and a read-only mapped
tape (`runner/internal/execution/process.go:22-28`; `gomad.go:251-297`). For each decision while
the cursor is inside the tape, the runtime compares kind, site, alternative count, and
alternative-set digest, then applies the recorded identity, or for the final rank-override record
the identity at that rank in sorted order (`gomad.go:539-567`, `573-587`). After the tape the
*same base seed's* streams continue; the seeded draw is consumed on every decision, forced or not
(`patch:586-588`), so the suffix is deterministic per (seed, prefix). A mismatch exits 125 with a
divergence terminal frame (`gomad.go:693-698`).

**Combined simulation exploration** (`runner/simulation_exploration_campaign.go`,
`runner/internal/exploration/simulation/frontier.go`, plan wire in `tools/gomad3sim/exploration.go`)
is the same breadth-first frontier over six dimensions (runtime, scenario, network, storage,
fault, crash) with per-dimension ordinal limits and `MaxForcedDecisions`
(`frontier.go:51-70`, `427-471`). Runtime overrides must compose into one rank prefix
(`simulationrecord/wire.go:212-256`). Guidance (`runner/guidance.go`) only re-selects whole
seeds; it "never forces runtime choices" (`README.md:224-226`). `runner/coverage.go` projects
choice features (site, branching site, first/last/interior rank class, adjacent pairs,
`choice/trace.go:64-98`, `268-312`) used for novelty retention, not for steering.

### The "two-outcome fixture" finding [V]

- Fixture: `tools/gomad3/internal/gomadtool/conformance/testdata/choice_exploration/main.go`.
  One goroutine, one `select` over two ready buffered channels, prints which case won.
- Benchmark: `TestRunChoiceExplorationPinnedOutcomeEfficiencyMatchesEqualBudgetSeedSampling`
  (`tools/gomad3/runner/runner_test.go:1155-1206`). Seeds 211–226 versus exploration from seed
  211 with `MaxExecutions=16`, depth 32. It asserts each side attempted 16 executions and saw 2
  distinct stdout hashes, and that outcomes-per-execution are equal.
- No report is retained; the evidence is the test. `docs/research/gomad/GOMAD_CMPv2.md:270-274`
  restates it as "both found the two declared outcomes … in sixteen executions".
- What it does not show: how many executions each strategy needed. The explorer needs at most the
  root plus one flip of the select decision; the other executions are spent on frontier entries
  that cannot change the output. The test also shows that a one-goroutine program yields at least
  16 candidates, i.e. the frontier is dominated by decisions unrelated to user logic (N5).

### Per-execution fixed cost (retained numbers, darwin/arm64) [V]

`wall_elapsed_nanos` is `finishedAt - startedAt` around one execution, from journal creation to
output collection (`runner/runner.go:1381`, `1462`, `879`).

| Target | Wall per execution | Decisions / records | Source |
| --- | --- | --- | --- |
| Trivial `go-run` probe (two forced GCs, four prints) | 0.24–0.31 s | 26 / 26, peak goroutines 2 | `.flow/artifacts/fn-105-…/fn105-d21-qualification-control*.json` |
| `TestUserTimersTestSuite` (untraced) | 1.5–1.6 s | n/a | `fn105-d13-untraced-qualification-seed11.json` |
| Six F6 functional suites, traced, 64 MiB cap | 1.5–1.8 s | 8.6k–25.9k decisions | table in `.flow/tasks/fn-100-gomad-f6-a-package-level-functional.2.md:12-31` |
| `TestSignalWorkflowTestSuiteChasm`, traced | 3.2–4.4 s | 57,801 / 86,243, peak 1,945 | `fn105-d14-qualification-seed11.json`, `-seed17.json` |

So the floor is roughly a quarter of a second per fresh process and a real suite is 1.5–4 s. No
retained breakdown separates supervisor launch, the darwin ASLR re-exec
(`overlay/src/runtime/gomad_aslr_darwin.go`), runtime start-up (eight reserved Ms,
`gomad.go:836-851`, `955-980`), package init, and evidence collection. linux/amd64 has no
re-exec and no retained per-execution numbers that I found.

### (b) Gaps

- Depth is an absolute ordinal, so BFS cannot reach mid-run decisions of any real target.
- No way to sample: the frontier is exhaustive-or-truncated in hash order within a depth.
- No pruning of provably equivalent flips (select with fewer than two ready cases, system
  goroutine ordering).
- A diverging prefix aborts the campaign as a host error
  (`choice_exploration_campaign.go:279-282`, `324-326`) rather than being recorded as an
  infeasible candidate. Correct for BFS from a deterministic parent; wrong for mutation across
  changed inputs. [I] that a divergence surfaces as `completion.err`.
- No discovery-cost metric anywhere in the summary (`engine.go:116-133`).

### (c) Smallest experiment

Controller-only, no toolchain change: add a second controller beside the BFS one that picks flip
ordinals by sampling (uniform over the parent's decisions, or weighted to rare sites) and reports
executions-to-first-new-outcome. Run BFS, sampler, and seeds at equal execution budgets on
(i) the two-outcome fixture, (ii) a three-goroutine ordering bug, (iii) one F6 suite with
`first_failure`. This turns "no advantage" into a measured statement and tells you whether prefix
replay is worth 1.5–4 s per candidate on real suites.

### (d) Size and risk

**S–M.** Risk: on real suites the answer may simply be that a single flip rarely changes the
outcome, which is itself the evidence GOMAD_NEXT asks for.

### (e) Stale or contradictory

- GOMAD_NEXT "Search evidence" and CMP both treat the fixture result as a baseline. It is a
  conformance pin, not a benchmark; it cannot show an advantage by construction.
- CMP's adoption table proposes comparing "preemption bounds, PCT against the raw frontier". The
  raw frontier is not a usable baseline on functional suites because of the depth semantics.

---

## 4. Schedule mutation (parent prefix + changed alternative + fresh suffix)

### (a) How close the primitive is [V]

It already is this operation, with three restrictions.

- `choice.BuildRankPrefix(source, decisionOrdinal, rank)` takes a complete validated parent tape,
  keeps decisions `[0..k]`, and replaces decision k with a rank override
  (`choice/tape.go:199-234`). `k` can be any ordinal in the parent; nothing in `choice/` limits it.
- The suffix is recorded fresh in the same execution (prefix mode keeps appending records,
  `gomad.go:568-569`), and the result is itself a complete trace that can be projected and
  mutated again. Multi-point mutations are built generation by generation.
- `Tape.Prefix(n)` gives a pure truncation without a flip (`tape.go:236-241`).

Restrictions:

1. Exactly one rank override, and it must be the last record
   (`tape.go:185-195`; runtime check `gomad.go:620-624`). Earlier decisions must carry exact
   identities, so a mutation always needs the parent's *observed* trace up to k.
2. The override is a rank in sorted-identity order because the tape does not know the other
   identities (0.3). You cannot ask for "run goroutine X here" unless X was the selected identity
   of some earlier record.
3. "Fresh suffix" means "the base seed's continuation". There is no way to draw several different
   suffixes for one prefix. Changing the seed changes map seeds, timer ties and runtime randomness
   from process start, which invalidates the prefix.

### Validation on a forced prefix [V]

- Host side before launch: digest, header (target hash, implementation hash, toolchain build key,
  platform hash), payload hash, ordinal continuity, every record a branching decision
  (`tape.go:158-197`, `305-349`).
- Runtime at start: magic, version, sizes, header checksum, payload hash, per-record shape
  (`gomad.go:251-297`, `620-624`).
- Runtime per decision: kind, site and flags, alternative count, alternative-set digest, selected
  identity present in the set (`gomad.go:543-553`, `573-587`).
- At exit: tape fully consumed, otherwise `TapeUnconsumed` (`gomad.go:681-691`).
- Host side after the run: the new trace must reproduce the prefix decision-for-decision and
  resolve the override to a real identity (`engine.go:482-506`).
- Divergence reasons are typed and the terminal frame carries expected and observed records
  (`gomad_choicewire_generated.go:28-38`; `gomad.go:708-743`; `tape.go:412-468`).

### (b) Gaps

- No public entry point: "Prefix replay is an internal bounded-exploration primitive"
  (`README.md:138-139`). Only the BFS and combined controllers call it.
- No suffix reseed.
- No "infeasible candidate" outcome class for the plain choice explorer (section 3b).
- Candidate identity includes the base seed and controller identity (`engine.go:420-429`), so a
  mutator is a new controller identity and a new journal schema.
- Cost: each mutant replays the whole prefix in a fresh process (1.5–4 s on suites).

### (c) Smallest experiment

Same sampler as 3(c); it *is* the single-point schedule mutator. Add a suffix-reseed variable
(overlay-only: when the tape cursor reaches the end, re-initialise `gomadChoiceRunqRandom` and
`gomadChoiceSelectRandom` from a second seed; `gomad.go:354-361`, `539`) only if the first
experiment shows that prefixes matter.

### (d) Size and risk

Sampler **S–M**; suffix reseed **S** in code plus a toolchain rebuild and identity plumbing.
Risk: exact replay of a reseeded artifact needs the suffix seed in the recorded environment, or
replay must use full-tape mode (which already works, since the complete trace is recorded).

### (e) Stale or contradictory

CMP describes this as something to build ("replay a validated parent prefix, change an offered
alternative, and record a fresh suffix"). The mechanism exists and is validated more strictly than
CMP assumes; what is missing is a controller that chooses mutation points other than
breadth-first, and non-fatal handling of infeasible prefixes. CMP's "Candidate infeasibility is
search feedback" is not what the choice explorer does today.

---

## 5. DPOR

### (a) What dependency information exists [V]

- **Runtime choice records: none.** No channel, mutex, semaphore, or timer identity; no "blocked
  on" or "woken by"; no goroutine identity on select records; no record at all for non-branching
  steps. The only causal fact derivable is goroutine parentage, and even that is hashed into the
  identity rather than recorded.
- **Available in-process but unexported:** `gp.waitreason`, the `sudog`/`hchan` a goroutine is
  parked on, and the waker at `ready()` time. `chan.go` and `sema.go` are in the prohibited patch
  class, so the only allowed observation points are `proc.go` (`ready`, `park_m`) and
  `select.go`.
- **World:** every request names `ResourceID{Adapter, Kind, Key}`
  (`tools/gomad3/world/types.go:38-49`); event order is keyed on it (ARCHITECTURE "Event
  ordering"); an adapter can declare an `EquivalenceClass` meaning "swapping these events cannot
  change semantics" (`world/types.go:51-57`; `world/choice.go:9-20`). That is a declared
  independence relation, already used to randomise order. Transitions are a hash-linked log
  (`world/replay.go:16-25`).
- **gomad3sim:** network and volume transitions are partitioned into causal lanes: per
  connection, per listener, one topology lane, per (node, volume)
  (`tools/gomad3sim/record.go:1109-1124`, `962-964`), and replay validates per lane.
- Explicit architectural limit: "An explicit-model digest cannot establish equivalence of
  arbitrary native Go execution state or justify pruning its schedule frontier"
  (`tools/gomad3/ARCHITECTURE.md:159-160`, `275-276`).

### (b) Gaps

No happens-before between goroutines, no per-step footprint, no vector clocks, and no complete
step sequence. Native DPOR is not approachable from the current trace. Model-level DPOR has its
independence relation half-defined (lanes), but the model decisions are a small share of the
frontier compared with tens of thousands of runtime decisions per execution.

### (c) Smallest experiment

Skip general DPOR. Two sound reductions are available from facts the runtime can observe cheaply:

1. Select-poll flips where fewer than two cases were ready are no-ops. Count ready cases at the
   `SelectResult` observation (all channels are locked there) and write it into a spare field;
   the controller then never expands those decisions. Validate by running reduced and unreduced
   BFS on a finite fixture and comparing outcome sets and deadlocks, which is exactly CMP's
   adoption gate.
2. In the combined frontier, treat two model decisions on different lanes as independent and
   compare reduced versus unreduced search on `tools/gomad3sim/testdata/simulation_exploration`.

### (d) Size and risk

(1) **M** (select.go hunk, wire bump, controller rule, agreement test). (2) **M**. General native
DPOR: **L+** and conflicts with the patch policy. Risk for (1): the patch grows; the argument that
poll order is irrelevant with fewer than two ready cases must hold for the blocking path too (the
enqueue order in pass 2 follows lock order, not poll order: [I] from upstream `selectgo`).

### (e) Stale or contradictory

CMP's DPOR paragraph matches the code. One addition: the World already has an explicit
independence declaration (`EquivalenceClass`) and lanes, so "start with an explicit mailbox/network
or storage model" has more to build on than the text implies.

---

## 6. Race detector and coverage instrumentation

### (a) What exists [V]

- Build arguments are fixed by target preparation: `go build` or `go test -c` with `-trimpath
  -buildvcs=false -o`, optional `-gcflags=all=-gomadcap [-gomadguard]`, `-ldflags=-linkmode=internal
  -gomadcap=…`, `-overlay`, `-modfile`, `-tags` (`tools/gomad3/target/target.go:702-726`). The
  only user-controllable build input is `--build-tag` (`tools/gomad3/CLI.md:58`). There is no path
  to pass `-race` or `-cover`.
- The `race` build tag is refused (`tools/gomad3/target/internal/build/context.go:65`).
- A prebuilt `exec` target whose build info says `-race=true` is refused, as are `CGO_ENABLED≠0`,
  non-exe build modes, and external linking (`target/target.go:827-850`).
- The runtime exits before user init if `iscgo || gomadExternal` (`gomad.go:107-110`), and the
  environment forces `CGO_ENABLED=0` (`README.md:70-71`).
- Every `runtime/race*` file is in the prohibited patch class (`toolchain/patch.go:277`).
- Docs give no rationale beyond listing it as outside the contract (`README.md:755`) and
  ".plans/GOMAD_CLOUD.md:75-76": "race detection is a separate unsupported execution profile".

**Why `-race` cannot simply be switched on** [I]: the race runtime is ThreadSanitizer, foreign
code with its own allocator and shadow memory; on linux it requires cgo, which Gomad rejects; and
a single-P, non-preemptive execution exposes few of the concurrent accesses TSan looks for,
although its happens-before analysis would still flag unsynchronised access pairs across
goroutine switches. One interaction worth knowing: Gomad sets `randomizeScheduler = true`, the
switch the race build normally turns on (`patch:542-547`), so that part of race-mode behaviour is
already present.

**`-cover`:** not rejected anywhere, and not reachable. `validateDeterministicBuildInfo` has no
`-cover` check (`target/target.go:827-850`), so a provenance-backed `exec` binary built with
coverage would pass that function. Whether it passes capability-closure review and whether the
counter dump at exit survives the deterministic I/O boundary (empty environment, modeled `os`
entry points) is untested. [I] README and ARCHITECTURE state code coverage "is not collected by
this mode" (`README.md:226-228`; `ARCHITECTURE.md:452-454`).

### (b) Gaps

No coverage profile, no identity field for an instrumentation profile, no fixture.

### (c) Smallest experiment

Build the `scheduler` conformance fixture with `-cover` through the pinned toolchain in a scratch
copy, run it with `GOMADSEED` directly (no Runner), and check same-seed stdout plus whether
`GOCOVERDIR` output appears. Then the same through `exec --provenance`. That answers whether
coverage is a build-flag plumbing task or a boundary-modeling task.

### (d) Size and risk

Probe **S**. Supported coverage profile **M** (flag plumbing, profile identity, counters through
the transcript or a dedicated descriptor, replay qualification). Race profile **L**, and probably
not worth it relative to keeping stock `-race` as a separate lane, which is what CMP recommends.

### (e) Stale or contradictory

None. CMP's "qualify coverage instrumentation as a separate execution profile" matches the code:
nothing exists yet.

---

## 7. Bug benchmark

### (a) What exists [V]

Nothing that is a "known bug + corrected counterpart" pair. Related material:

| Item | Location | What it is |
| --- | --- | --- |
| Update-admission ordering regression | Fix in `service/history/workflow/update/registry.go:349`, `512-520` and `update.go:56-100`; unit test `service/history/workflow/update/admission_order_test.go`; narrative in `.flow/tasks/fn-100-gomad-f6-a-package-level-functional.2.md:40`; commit `97d9925ac` | Only the corrected code is in the tree. The broken variant exists only as history. No fixture toggles it. |
| `preemption` | `conformance/testdata/preemption` | Atomic spin; pinned to *hang* under Gomad. A visibility witness, not a bug pair. |
| `choice_exploration` | `conformance/testdata/choice_exploration` | Two-outcome select. No failure. |
| `sync`, `scheduler`, `runqueue`, `channels`, `select`, `clock_race`, `io_net_races`, `io_handoff_contention`, `clock_deadlock` | `conformance/testdata/…` | Order-printing and model-contract fixtures. Correct programs. |
| Target-failure expansion test | `runner/runner_test.go:1135-1153` | Exploration over a target that fails on every path. |
| Duplicate-delivery scenario | `tools/gomad3sim/temporal_scenario_toolchain_test.go:17` | A scripted matching-style failure with exact replay. No corrected twin that I found. |
| `simulation_exploration` | `tools/gomad3sim/testdata/simulation_exploration/main.go` | Two-route scenario choice. No failure. |
| Qualification manifests | `tools/gomad3integration/qualification/{smoke,temporal,tests}.json`, `tools/gomad3/qualification/core.json` | Expectations are about Gomad repeatability (`qualified`, `unrepeatable`, `unsupported_target`), not target bugs. |

A caveat on the one real regression: per the commit message it was "two updates admitted within one
virtual clock tick reordered" through a sort over a map. It is triggered by strict virtual-clock
ties and map order, not by a rare interleaving, and it failed the suite outright. It is a good
*oracle/replay* benchmark and a poor *schedule-search* benchmark: any seed likely finds it. [I]

### (b) Gaps

No bug families, no held-out set, no discovery-cost harness, no place in the manifest schema for
"expected to fail with signature S within budget B".

### (c) Smallest experiment

A `bugs/` group in the conformance module: for each family in CMP's list one program with a
`-broken` flag (same binary, so target identity is shared between variants and only argv differs).
Six programs: atomic check-then-act, unlock/wake ordering, close-versus-send, cancel-versus-complete,
lock-order deadlock, equal-deadline timer tie. Plus the update-admission regression behind a
`gomad`-only build tag or test hook that removes the `admittedSeq` tie-break (allowed by the
"server source changes are allowed but bounded" constraint, `MILESTONES.md:149-153`).
Record for each: reachable today (yes/no), failures per 1,000 seeds.

### (d) Size and risk

**S** for the fixtures and a table; **M** with a harness that reports discovery-time
distributions. Risk: tuning policies on six toy programs; mitigate with the held-out rule CMP
already states.

### (e) Stale or contradictory

CMP says to "retain the real Temporal update-admission ordering regression … using its task
evidence". The task evidence is one sentence and a commit reference; there is nothing retained to
run. It has to be reconstructed.

---

## 8. Opportunities not in GOMAD_CMP.md

**N1. Wake-up yield points through the existing cooperative-preemption flag.** [I]
In `ready()` (proc.go, already patched), when Gomad is enabled and the caller is a user goroutine,
make a recorded two-way decision "waker continues / waker yields". For "yields", set
`gp.preempt = true; gp.stackguard0 = stackPreempt` on the current goroutine. The next function
prologue enters `newstack`, which already refuses to preempt while `m.locks != 0`, during
allocation, or on g0 (`runtime/stack.go:1122-1174`), and then calls `gopreempt_m`. No compiler
change, no new yield path, no prohibited file. This is the single change that makes preemption
bounding and PCT meaningful (bound = number of "yields" taken) and it reaches most of CMP's
witness list. It does not reach atomic check-then-act. Two details: `gopreempt_m` requeues to the
global queue, so the Gomad branch should requeue locally to keep the waker in the alternative set;
and the yield lands at the next prologue rather than at the wake-up itself, which is deterministic
for a fixed binary but coarser than a true boundary. Size **S–M** to prototype.

**N2. Put what the runtime already knows into the Runnable record.** [V that the fields are free]
`data` is 0 on Runnable records (`gomad.go:801`) and two header bytes are reserved-zero
(`gomad.go:623`, `record[10:12]`). Candidates: why the previous goroutine left
(blocked/yielded/exited/syscall plus `waitreason`), and a short tag of the previous goroutine's
identity. That gives the controller the continue/block/yield/exit distinction CMP asks for, and
gives `inspect` a real timeline. Needs a wire version bump. **S–M.**

**N3. Record the running goroutine on select records.** Select identities are site-only
(`gomad.go:821-834`). Adding the executing goroutine's identity (the selected-identity field is
unused for observations) makes per-actor timelines possible and lets coverage distinguish "same
site, different actor". **S.**

**N4. Ready-case count on `SelectResult`** enabling the sound select reduction in 5(c). Could
remove up to ~46% of branching decisions from the frontier of a functional suite. **M.**

**N5. System goroutines are explorable alternatives.** [V counts, I cause] The trivial probe has
peak user goroutines 2 and still records 26 branching Runnable decisions
(`fn105-d21-qualification-control-choices-11.json`). `gomadChoiceRunqIndex` does not distinguish
system goroutines (`gomad.go:794-800`). A fixed rule (system goroutines first, in identity order)
would shrink every frontier and make BFS depth mean something. It changes schedules, so it is a
new implementation identity. **S–M.**

**N6. A sampling mutator instead of BFS** (sections 3c, 4c). Controller-only. **S–M.**

**N7. Suffix reseed in prefix mode** (4c). Overlay-only. **S.**

**N8. `time.AfterFunc` goroutine identities are creation-order based** (section 2a). Deriving them
from the timer's creation site and creating goroutine instead of the global counter would make
context-deadline goroutines stable across mutated schedules. Otherwise mutated prefixes will tend
to diverge on alternative-set digests as soon as two deadline callbacks fire in a different order.
**M** (needs an identity stored on the timer; `time.go` is allowlisted). [I]

**N9. A discovery-cost metric.** Neither explorer summary reports the execution ordinal at which
each distinct outcome or failure first appeared, although `LogicalExecutions` and per-execution
`ElapsedNanos` are already journaled (`choice_exploration_campaign.go:344-350`). Without it every
comparison in CMP's adoption table has to be reconstructed by hand. **S.**

---

## 9. Suggested order, if the goal is evidence per unit of effort

1. Bug fixtures with broken/corrected flags plus the reachability table (7c, 1c). **S.**
2. Discovery-cost metric and sampling mutator; rerun the three-way comparison (N9, 3c). **S–M.**
3. Wake-up yield prototype in a scratch toolchain; rerun the table (N1). **S–M.**
4. Only then: richer records (N2–N4), in-process priority policy (2c), system-goroutine rule (N5).
5. Atomic call-site instrumentation as its own execution profile, last (1d).

Steps 1 and 2 need no toolchain change and no patch-policy decision. Step 3 is the first that
touches the patch, and it is the one that decides whether PCT and preemption bounding are worth
building.
