# fn-227 audit: are we making a big mistake?

Read-only investigation, 2026-10-21, at `e68f3d4a20` (branch `umpire`). Base for "before fn-227"
is `a76c358162`, the commit just before the first fn-227 code commit (`c3d85d9cf9`). Every commit
touching `model/umpire/` in `a76c358162..HEAD` belongs to fn-227. I checked this with
`git log a76c358162..HEAD -- model/umpire`.

## Verdict

**The work already built is a moderate over-scope. The remaining plan, as written, is a big mistake.**

- The built native half (.1 World, .2 Finite and inputs) is sound. It is on fn-228's critical
  path, so keep it. The built source half (.7, .8) is paid for. It gives fn-211 a parameterized
  lifted tree, so keep it as well.
- The remaining eight tasks are a different matter. About 75% of their planned scope is on no
  consumer's critical path:
  - .11 Phased/reject source admission.
  - .9 deadline facts.
  - .10 nominal Option values.
  - Most of .4: the four timeouts, Tick and cooperative cancellation.
  - The refinement harness in .3 and .4.
  - The "derive the World from the observed request" state machine in .5.

  Most of this scope exists for two reasons. One is to make the activity Model liftable to TLC.
  The other is to model Temporal timeout semantics that no filed consumer asks for.
- One spec rule doubles the cost of every new DSL construct: "no implementation may ...
  substitute native exploration for independent source semantics" (spec:110). Under that rule,
  every construct is built twice, once natively and once in the lifter. The scaling argument
  that justified the second build still has no Model behind it, and fn-227 will not produce one
  either (§1.4).

## 1. The native/source split

### 1.1 Measured cost per task

Lines are additions (and deletions) from `git show --numstat`, grouped by path. "Native" means
`dsl/`, `dsl/macros/`, `checker/`, `suite/` and `wasm/`. "Source" means `lift/`, `backends/tlc/`
and `testing/`. Test lines include `tests/`, `expected/`, `ui/`, `pass/` and the shared
`umpire/tests/fixtures`.

| Task | Commits | Native src | Source src | Native tests | Source/shared tests |
| --- | --- | ---: | ---: | ---: | ---: |
| .1 native World | c3d85d9cf9, b5b91b4f82, 9147fbdf6b, f0adb5afdb | +1,539 / -384 | 0 | +1,646 | 0 |
| .7 source World | 53349d60ac, 380ed2a607, aa5bca1c23, d953705b13 | +16 (checker) | +1,309 / -165 | 0 | +912 |
| .2 native inputs | cabe589636 (+ guest cba0503137) | +1,257 / -180 | +2 | +1,248 | 0 |
| .8 source inputs | 3fd1d2f551, 0469334f91 | 0 | +1,624 / -131 | 0 | +939 |
| **Total** | | **2,812 gross / 2,248 net** | **2,935 gross / 2,638 net** | **2,894** | **1,851** |

Whole-crate `src/` totals confirm the net figures. I counted `.rs` lines under each crate's
`src/` at the two commits:

| Crate `src/` | Before (a76c358162) | Now (HEAD) | Growth |
| --- | ---: | ---: | ---: |
| `dsl/macros` | 2,276 | 3,891 | +1,615 (+71%) |
| `dsl` | 2,135 | 2,414 | +279 |
| `checker` | 1,422 | 1,520 | +98 |
| `suite` | 2,163 | 2,410 | +247 |
| `wasm` | 734 | 743 | +9 |
| **Native subtotal** | 8,730 | 10,978 | **+2,248** |
| `lift` | 3,076 | 5,334 | +2,258 (+73%) |
| `backends/tlc` | 1,300 | 1,355 | +55 |
| `testing` | 1,122 | 1,447 | +325 |
| **Source subtotal** | 5,498 | 8,136 | **+2,638** |
| All `model/umpire/**/src/*.rs` | 17,515 | 22,401 | **+4,886 (+28%)** |
| All other `model/umpire/**/*.rs` (tests, fixtures) | 14,592 | 18,707 | +4,115 |

**The source side accounts for 54% of the net new framework source.** Each feature costs about
the same on both sides:

- World: 1,155 lines native vs 1,144 lines source (net).
- Inputs: 1,077 lines native vs 1,493 lines source (net).

The lifter (5,334 lines) is now larger than it was before fn-223 deleted the manual-trait lowering.
DESIGN.md:12-14 records that fn-223 cut it from 5,048 to 3,022 lines to simplify it.

### 1.2 Of the source cost, roughly half is binding plumbing, not semantics

The lifter is independent of native types, but it still needs the World values, the starts and
the input domain values. fn-227 therefore built a pipeline that obtains those values from native
code through generated code:

- .7: `lift/src/build.rs` +394 lines. This generates an augmented, compile-only copy of
  `model.rs` in `OUT_DIR`, with a private-field projection inside the machine module.
- .7: `tree.rs` +160 lines (binding representation) and `testing/src/{backend,discovery}.rs` +110
  lines (replay and discovery overrides).
- .8: `tree.rs` +348 lines (catalog/schema metadata), `build.rs` +143 lines (domain binding) and
  `testing/src/backend.rs` +241 lines (typed payload replay).

The translation of semantics proper is smaller:

- .7: about 600 lines across `shape.rs`, `body/*` and `emit.rs`.
- .8: `body/inputs.rs` (439 new lines) and `emit.rs` (+199 lines).

So the "independent reading" is already only partial. The task contracts themselves say that the
World, the complete starts and the domain values come from native code:

> .7 Approach 4: "Complete starts may be enumerated natively as R1 requires"

> .8 Approach 2: "Domain values may be supplied by the typed finite iterator route"

Only the guards, effects and claims are read independently. fn-227 pays for a bridge back to the
native side and for an independent translator at the same time.

### 1.3 Inside the macro crate, the byte-identity rule produced a second expansion path

`dsl/macros/src/machine.rs:874-875` reads:

```rust
if declared::present(items)? { return declared::expand(input, rejecting, staying); }
```

`machine/declared.rs` is an 892-line parallel generator. It emits its own `starts`, `actions`,
`invariants`, `postconditions`, `progress`, `assumptions` and `claim_order`
(`declared.rs:566-575`, `:660`), beside the 969-line unit-World generator in `machine.rs`
(`:937-964`). The reason is .1's Approach 1: "Keep the existing absent-World expansion branch
literally unchanged". That instruction protects existing expansion snapshots byte for byte. A
unit World is a declared World with zero fields, so one path would do. The cost of the second
path is test churn (regenerated snapshots), not semantics.

### 1.4 Why the lifter is independent, and whether that still holds

These are the recorded rationales:

1. **Scaling (owner, 2026-10-19), quoted in `model/design/second-opinion.md`:**

   > we'll need TLC for scaling our models; our own model checker is a simple, non-scalable
   > version that won't hold

   The note adds that the current corpus "does not bound the Models the framework must
   eventually support".

2. **Independence of the second opinion, from `second-opinion.md` ("What TLC buys"):**

   > These benefits require an independent reading of source. Exporting the native checker's
   > already computed graph to TLC would share the very transition interpretation being checked.

3. **Not rustc IR; use a `syn` pass:** the fn-212 Decision Context of 2026-10-18 (stock toolchain,
   no nightly).

4. **Historical background, from fn-210:14:** Rust was expected to "make lifting a macro's
   by-product". The fn-212 note later chose a separate `syn` pass over capturing tokens in
   `#[machine]`, which made the lifter a second front end rather than a by-product.

5. **Archived Scala precedent:** the Scala framework had an IR, a Go reader and a Quint
   "second opinion" export. On 2026-10-17 the owner retired the IR/Go loop in favour of the
   native checker as oracle (fn-196). The Quint comparison was left "held; needs the tools its
   README names" (`model/archived/README.md:232`). The last multi-reader pipeline was retired
   for its cost, and the second opinion was never gated.

**Does the rationale still hold?**

- **Scaling holds as a direction, but not as a justification for fn-227's cost.**
  - fn-227's own Goal says TLC and Apalache "have no Model whose state space exceeds the native
    checker" (spec:20). fn-227 does not fix that.
  - **The projected activity World (my estimate):** `maximum_attempts` 1..=3, four
    `Option<Ticks 1..=3>` timeouts and one `bool`. That is 3 × 4⁴ × 2 = **1,536 Worlds**.
  - **The largest World:** at most a few thousand reachable states, estimated from phase ×
    attempt × four countdowns of 0..=3 or None × `cancel_requested`.
  - **Native speed:** about 283 states in 66-103 µs in release (DESIGN.md:766-772). That is
    about 3-4 million states per second, so native checks such a World in milliseconds.
  - **For a parameterized Model, the scaling axis is the number of Worlds,** and on that axis
    TLC is the worse engine. It needs one JVM run per World and per configuration family (main,
    invariant, postcondition, negative and progress; .7 Approach 6).
  - **The gate has grown:**
    - Required-backend gate: 182 s (.1), 557 s (.7, 9m17s), 416 s (.2).
    - DESIGN.md:113 records 144 s for the whole required gate after fn-212.
  - **No scaling measurement exists.** `second-opinion.md` says "The current corpus provides no
    TLC/native scaling measurement". That note also required "the first larger Model must record
    state/edge counts, memory, time ... for both engines". The activity after fn-227 would not be
    that Model.
- **Independence holds only in a narrow sense, and it is weaker than it sounds.**
  - It catches macro adapter mistakes and native explorer or progress defects.
  - The explorer already has a brute-force oracle over 256 generated Worlds
    (`second-opinion.md`, "Brute-force oracle"), and the macros have expansion goldens and
    trybuild tests.
  - I found no commit in `git log --all` whose subject records a native defect found by TLC
    disagreement. Every `fix(lift)`/`fix(tlc)` commit fixes the lifter or backend itself, for
    example `380ed2a607`, `aa5bca1c23`, `d953705b13`, `31ae8ec7f3` and `afecc53237`.
  - fn-227 has already weakened the independence (§1.2).
  - The independence also constrains authors. The lifter's subset determines what Rust a Model
    may contain (§3.3).
- **The rule that hurts is the policy that makes source admission a precondition, not the lifter
  itself.** That rule is spec:110 together with the task gates "no native-graph emission" and
  "must not hide an unsupported construct". Before fn-227 the framework already tolerated
  native-only Models: the activity has been a located lift refusal since fn-221. Today's
  `temporal/feature/activity/expected/lift.snap.txt` reads "trait or generic impl in machine
  module is unsupported; TLA+ emission refused". fn-227 turned "TLC where the subset admits it"
  into "every new construct must be lifted before the Model may use it".

## 2. Scope fit: what each consumer actually needs

| Consumer | What its spec consumes from fn-227 | Evidence |
| --- | --- | --- |
| **fn-217 fill/validate** | **Nothing.** fn-217 is `done`. It is a *prerequisite* of fn-227 (`fn-227.json depends_on_epics: [fn-218, fn-217]`), and all seven tasks are done with no fn-227 reference. The spec's claim (spec:20) that "fn-217's `fill`/`validate` need request fields with domains" is stale. | `.flow/specs/fn-217*.json` status `done`; `grep fn-227 .flow/tasks/fn-217*` finds nothing |
| **fn-233 Kettle service** | **Ordering only.** The Kettle Model stays unit-World and unchanged ("No change to the Kettle Model or its goldens"). `Runtime::with_initial` is fn-233's own work. Its only fn-227 coupling is "Preserve fn-227.12's optional Observation.request tag 4 ... serialize this migration after that task" and "Task .1 ... after fn-227.12". | fn-233 spec, "Envelope migration (R1)" and "Build and scheduling" |
| **fn-228 Runner** | A declared World and native action inputs, to fill Propose: "proposing a request needs a world and action inputs to fill it from" (fn-228:17). The activity must retry: R4 says "start, retry, completion, cancellation, host refusal". Same-invocation World admission at Start (fn-228:68, fn-228.4). It explicitly needs no TLC: "No DSL change beyond what fn-227 delivered" refers to the DSL, and nothing in fn-228 reads the lifter. | fn-228 R2/R4, "Bootstrap and pending state" |
| **fn-211 Apalache** | The lifted tree with selected World, complete starts and concrete inputs: "Preserve selected World, complete starts and concrete action inputs from fn-227" (fn-211:59). Its first acceptance is a Kettle check (fn-211:53). Nothing names timeouts, facts, Option or the activity. | fn-211 spec |

**What fn-227 builds, against those needs:**

| fn-227 piece | Task | Consumer on its critical path? |
| --- | --- | --- |
| Declared World, computed starts, World-typed callbacks, generics | .1 (done) | **Yes**: fn-228 |
| Source World binding | .7 (done) | **Yes**: fn-211 |
| Finite, `#[domain]`, checked catalog | .2 (done) | **Yes**: fn-228 Propose |
| Source inputs, exact-payload fairness | .8 (committed) | Partly: fn-211 needs source inputs. Exact-payload fairness is needed only once a Model has stale payload self-loops, which means the activity's `Poll{expected_attempt}`. |
| Phased/`reject!`/closure-law source admission | .11 | **No.** Its only purpose is to make the activity TLC-liftable. |
| Activity retries (World `maximum_attempts`, `RespondFailed{retryable}`) | .3 | **Yes**: fn-228 R4 |
| Retained-lifecycle refinement harness and immutable reference tree | .3 + .4 | **No.** This proof protects a 130-line, 6-state, 11-edge Model, and the harness will be larger than the Model. |
| `Option<F>` handler facts | .9 | **No.** It serves only the activity's expiry fact. |
| Nominal bounded Option values in source and decode | .10 | **No.** It is entirely source-side; native Rust `Option` already works. |
| Four optional timeouts, Tick, countdown priority | .4 | **No.** |
| Heartbeat, cooperative cancel, worker acknowledgement | .4 + .5 | **No.** fn-228 needs "cancellation", and the existing atomic Cancel already satisfies that. |
| Observation tag 4 (request transport) | .12 | **Yes, as an ordering point**: fn-233.1 and fn-228's admission |
| Transactional live World *derived* from the request (quantization table, admission freeze, cooperative-cancel protocol binding) | .5 | **Partly.** fn-228 fills Start *from a preselected World* ("filled from the selected run configuration", fn-228:68). Admission then only has to check that the actual request matches that World; it does not need to derive the World with a per-field quantization table. |
| DESIGN write-up | .6 | Documentation only |

## 3. Complexity trajectory

### 3.1 Projection for the remaining eight tasks

The basis is the four completed tasks: each "M" task added 1,078-1,493 net framework src lines
and 912-1,646 test lines. Each remaining task is scaled by its own **Files** list.

| Task | Main crates | Projected src | Projected tests |
| --- | --- | ---: | ---: |
| .11 | lift (+ testing replay) | 800-1,300 | 600-900 (new `phased_rejection` fixture) |
| .9 | macros + lift | 600-1,000 | 600-800 |
| .10 | lift + tlc decode ("roughly fifteen source consumers") | 1,000-1,500 | 700-1,000 |
| .3 | activity + new `compatibility/` harness | 200-400 | 700-1,200 (+ goldens) |
| .4 | activity | 200-400 | 800-1,200 |
| .12 | proto, realization, Go | 200-400 | 400-700 |
| .5 | realization facade, Go host | 600-1,200 | 1,000-1,500 |
| .6 | docs | 0 | 0 (150-300 doc lines) |
| **Total** | | **3,600-6,200** | **4,800-7,300** |

**End state if the plan runs as written (estimates):**

- `model/umpire` src grows to about 25-26.5k lines, against 17.5k before fn-227 (+43-51%).
- `lift/src` grows to about 7.5-8.6k lines. That is 2.5-2.8× its pre-fn-227 size and about 1.6×
  the 5,048 lines that fn-223 deleted down from.
- The whole of fn-227 reaches roughly 8.5-11k src lines and 9-11.5k test lines.

**Planning overhead is already large:**

- The fn-227 spec and task files hold 142 KB of prose.
- `.flow/artifacts/fn-227-*` holds 34 files (400 KB, 1,249 lines of `.md`).
- `.flow/tmp` holds 345 `fn227*` scratch files.

**.8 is not gate-clean.** Its only logged Full gate attempt (`.flow/tmp/fn2278-full-check.log`)
exits 2 with `stale guest ... existing=3332182 bytes, built=3332201 bytes`. The cause is .8's
`Cargo.lock`/`umpire/tests` dependency change (`umpire-tlc` added). The guest must be refreshed
before .8 can close.

### 3.2 R30: "proc-macro crates stay small"

fn-210 R30 (`fn-210 spec:71`) lists "proc-macro crates stay small" among the rules that keep the
loop snappy. Size and timing so far:

- **Macro crate size:**
  - 910 lines when R30 was recorded (DESIGN.md:430).
  - 2,276 lines before fn-227.
  - 3,891 lines now. That is 4.3× the R30-era size and +71% inside fn-227.
- **.9 adds more to the macro crate:** the `Option<F>` handler-return adaptation across "five
  existing production owners".
- **Inner-loop timing, one sample per task:**
  - 1.596 s before fn-227 (`fn2271-baseline-time.log`).
  - 5.035 s after .1, 7.362 s after .7 and 9.852 s after .2, against a 10 s target.
- **Default-gate timing:** noisy (the baseline was already 86 s), so I do not attribute it to
  fn-227. The owner raised the default-gate soft target from 30 s to 60 s on 2026-10-21
  (`9b22d28bbd`), in the middle of fn-227.

### 3.3 Author ergonomics: plain Rust vs the lifter's subset

These are the owner rules involved:

- "statements and ordinary impls over signatures and attributes"
  (memory: author-ergonomics, fn-221 entry).
- "Author ergonomics over plugin/DSL convenience: plain Rust" (conductor brief).

**`parameterized_world`** (`umpire/tests/fixtures/parameterized_world/model.rs`, 93 lines) reads
well:

- `#[derive(World)]`.
- A `#[starts] impl Model { fn starts(selected: &World) }` block.
- A trailing `selected: &World` on the handler, invariant, postcondition, scope, assumption and
  progress.

`tests.rs` (98 lines) is the weak spot. It needs a `thread_local!` `SELECTION` and a
`choose_world()` hook to drive two Worlds through `fixture!`, plus construction and enumeration
counters, to prove "one selection, one enumeration".

**`action_inputs`** (`model.rs`, 123 lines; `tests.rs`, 212 lines) shows the costs:

- **Fair lists must be literal arrays of concrete payloads,** e.g.
  `fair([Worker::Poll(1), Worker::Poll(2)])` (model.rs:81). The lifter refuses anything else:
  `lift/src/body/progress.rs:93` reads "assumption requires an explicit action array". For the
  activity, `Poll`'s domain is `1..=w.maximum_attempts`, a World parameter, but the author must
  hard-code `Poll{expected_attempt: 1}`, `{2}` and `{3}` in the fair list.
- **World-dependent actors lose wildcard and outcome-filter `path!` forms.** README:507-513 says:

  > a `User::*` or `User::Set { .. }` pattern without a World refuses at compile time, so
  > contextual queries pass the selected World explicitly:
  > `Pattern::action(User::actions_in(&world).map(Action::from), None, None)`

  The README also notes that "an outcome filter (`=>`) still requires a static `Actor`". This is
  a two-trait split (`Actor` / `ActorValue`), and the .2 review already lists an `.internal()`
  ambiguity (follow-up L5).
- **The TLC mutation test hard-codes emitted TLA+ text** (tests.rs:169-202). It is brittle
  framework scaffolding, but it is a fixture, not author code.

**The projected activity Model** (my sketch from spec:30 and the .4/.10 contracts) would carry:

- World: six fields.
- State: `phase`, `attempt`, four `Option<Remaining>` countdowns and `cancel_requested`.
- Two `derive(Finite)` newtypes with `#[inputs(1..=3)]` / `#[inputs(0..=3)]`.
- An `Expired` fact enum and `Deadline::Tick -> Option<Expired>`.
- `Worker::{Poll{#[domain] expected_attempt}, RespondCompleted, RespondFailed{retryable},
  Heartbeat, AcknowledgeCancel}`.
- A per-value fair list.
- Authored bounds invariants using generated `contains`.

**The decisive constraint is .10's admitted Option grammar:**

> keep ... match/if-let/combinators, unwrap/expect and mutable access as located refusals

Admitted are only presence tests, `==`, `Some`/`None` and eager `unwrap_or` with `.0`
projection. A plain-Rust author would write each of the four countdown decrements as
`if let Some(Remaining(n)) = &mut self.x { *n -= 1 }`. That form is forbidden. Each decrement
has to be spelled with `is_some()`, `unwrap_or(Remaining(0)).0` and reassignment, and the Tick
handler has an expiry priority over four deadlines.

My estimate is that the Model grows from 130 lines to about 300-380, plus a test-local checked
World factory and the `compatibility/` harness. **The lifter subset, not the domain, would decide
how the Model's Rust looks.** That runs against the owner's ergonomics rule.

## 4. Cheaper alternatives

Each estimate below is measured against the remaining plan in §3.1.

### A. Native-first parameterization; TLC only where the subset already admits the construct

- **What it is:** Freeze source admission at .8 (World and inputs). Any new construct a Model
  uses beyond that subset gets a located lift refusal, as the activity has today. Optionally add
  a small **native-graph TLA+ export** (states as records, `Next` as the explicit edge set, the
  authored WF obligations) so that TLC can independently check the native *progress/fairness
  verdicts* on every Model.
- **Savings:** all of .11 and .10; the source half of .9 (about 60%); and the TLC/larger-World
  parts of .3 and .4. That comes to about **2,800-4,300 src and 2,500-3,700 test lines**, or
  roughly 3 of the 8 tasks. The graph export costs once, about 300-500 lines, with no
  per-feature cost.
- **What is lost:**
  - An independent check of guard/effect semantics for new constructs (the macro expansion of
    Phased/reject, facts and Option).
  - TLC scaling for Models outside the subset. No such Model exists.
  - Apalache on the activity. fn-211 still has Kettle and the two parameterized fixtures.
- **Note:** a graph export does give a genuinely independent check of the most intricate native
  code, the fair-cycle/progress checker. That is where the observer and fairness encoding effort
  went (`second-opinion.md`, "Machinery and semantic work").

### B. Defer source admission of phases, rejection and optional values; keep .8's inputs

- **What it is:** The same as A without the graph export. Mark .11 and .10 deferred, cut .9 to
  its native half, and keep .8.
- **Savings:** about **2,400-3,800 src and 1,900-2,700 test lines**. It also removes the
  serial-ownership coupling between .9, .10 and .11 on `lift/`.
- **What is lost:** the activity stays TLC-refused, as it has been since fn-221. R3's "so the
  activity Model is no longer refused" moves to a future spec.

### C. Shrink the activity extension to retries, plus at most one timeout as it exists today

- **What it is:**
  - World: `maximum_attempts` only, plus `asynchronous_cancel: false` implied.
  - State: `attempt`.
  - `RespondFailed { retryable }`.
  - Keep the existing `Deadline::{ScheduleToStart, StartToClose}` internal actions unchanged:
    no Tick, no countdowns, no Option, no facts.
  - Optionally `Poll{expected_attempt}`, only if the realization needs stale-attempt rejection.
    Otherwise Poll stays a unit action and the observed attempt maps onto state.
- **Savings:** .9 and .10 entirely, nearly all of .4, the cooperative-cancel protocol binding
  in .5 and the timeout quantization table. That is about **2,200-3,600 src and 2,500-3,500 test
  lines**, and the activity Model grows by about 40 lines instead of 170-250.
- **What is lost:**
  - The model of Temporal timeout semantics: schedule-to-close spanning attempts, heartbeat
    resets, the expiry fact and the cooperative-cancel race.
  - R5 entirely.
  - These remain true facts about the server that the Model would not check. No consumer needs
    them yet.

### D. Drop lifecycle refinement

- **What it is:** Keep the old six-phase goldens as a commit-pinned artifact in git history.
  Rely on the new goldens, the terminal laws (already parametric) and the live SDK slice. Do not
  build the `compatibility/` tree, the projection or the mutation controls, and do not re-run
  them in .4.
- **Savings:** about **500-1,000 test/harness lines**, plus the hardest-to-review contract in
  the spec (R4 and the 40-line "Plan review follow-up" negotiation).
- **What is lost:** a mechanical proof that the extension preserves the old lifecycle. With C,
  the change is small enough (retry adds one edge family) to review directly.

### E. Fold .12 and .5 into fn-228 and make admission "match", not "derive"

- **What it is:** fn-228 already redefines bootstrap, Reset and pending admission (fn-228:68),
  and it fills Start from the selected World. Admission becomes: the actual filled Start request
  and result from the same invocation must match the selected World's `maximum_attempts` (and
  timeouts, if C keeps any). Tag 4 is added once, directly in the envelope package that fn-233.1
  moves to `umpire.realization.v1`.
- **Savings:**
  - About **400-900 src and 600-1,000 test lines**: no quantization table, no
    cooperative-cancel binding and no admission-freeze variant for out-of-domain derivations.
  - The double edit of the envelopes. Today .12 adds tag 4 in `temporal/proto`, and then fn-233.1
    moves the same envelopes.
- **What is lost:** live admission of a World that was never preselected (observing an arbitrary
  third-party Start). No consumer asks for that.

**Combined A/B + C + D + E:** almost all of the eight remaining tasks collapse into two pieces.

- **Close .8.** Finish its review and refresh the guest.
- **One native task, "activity retries".** About 40 Model lines, new goldens and an explicit
  fn-218 `Cases(n)`; the realization maps observed attempts.

Live admission moves to fn-228. Estimated avoided work: **about 3,000-5,500 src lines and
4,000-6,500 test lines**. `lift/` stays at about 5.3k lines instead of 7.5-8.6k.

## 5. Recommendation

**Keep:**

- **.1 native World** (c3d85d9cf9..f0adb5afdb), which fn-228 needs.
- **.2 Finite, `#[domain]` and checked catalog** (cabe589636), which fn-228 Propose needs.
- **.7 source World binding** (53349d60ac..d953705b13) and **.8 source inputs** (3fd1d2f551,
  0469334f91), which fn-211 needs. Finish .8's review and refresh the embedded guest (it is
  stale per `fn2278-full-check.log`) before closing it. Keep its exact-payload fairness: it is
  built and tested, and it is correct.

**Re-scope fn-227 to "Parameterized Models core": R1, R2 and the World/input part of R3.**

- Close the spec after .8 and a slimmed .6. In .6, update DESIGN's readiness table rows for
  World, computed starts, inputs and generics to "delivered", and record the measured counts.
- Move these out of fn-227 into a new deferred spec, "Activity timeouts and TLC admission of the
  activity":
  - .11 (Phased/reject source admission).
  - .9 (facts).
  - .10 (Option values).
  - .4 (timeouts, Tick, cooperative cancel).
  - The refinement part of .3 and .4.
  - R5.

  The trigger for reviving that spec should be a named consumer: a Runner question about
  timeouts, a Model that exceeds the native checker, or Apalache needing the activity.

**Replace .3 with a native-only "activity retries" task.**

- Add `maximum_attempts` to the World, `attempt` to the state and `RespondFailed { retryable }`.
- Leave the old deadline actions untouched.
- Make the activity lift golden stay a located refusal.
- Add new default goldens and fn-218 `Cases(n)`.
- In the realization, map Fail to `retryable=false` until fn-228 supplies actual retryability.

**Move .12 and .5 into fn-228** as "admit the actual Start against the selected World".

- Add tag 4 after fn-233.1's envelope move, so the envelopes are edited once.

**Fix the dependency edges:**

- Remove fn-233's spec-level dependency on fn-227. fn-233 can start now: its .1 envelope move
  goes first, and tag 4 is added afterwards in the moved package.
- fn-228 depends on fn-227 core plus the retries task.
- fn-211 depends on fn-227 core. It also inherits the scaling-justification gap (§1.4), and the
  owner should re-confirm that separately.

**Change the policy that causes the doubling.**

- Amend spec:110 and the R3 language so that new DSL constructs may ship native-first with a
  located lift refusal. Source admission is added when a TLC or Apalache consumer needs that
  construct.
- Record the first actual native-vs-TLC scaling measurement before lifter growth is justified by
  scaling.

**Housekeeping, low priority:**

- Fold `machine/declared.rs` and the unit expansion into one generator when expansion snapshots
  can be regenerated (§1.3).
- Give R30's "proc-macro crates stay small" a number, or retire the rule. At 3,891 lines it is
  no longer being applied.

## Method

The commands behind the numbers above:

- `git show --numstat <commit>` per task, grouped by path prefix and test/src heuristics. The
  script is in the session scratchpad.
- `git ls-tree` + `git show <commit>:<file> | wc -l` over `model/umpire/**/src/*.rs` at
  `a76c358162` and `HEAD`.
- `wc -l` on the fixtures and activity files.
- Timings come from task done summaries and `.flow/tmp/fn227{1,2,7,8}*-time.log` /
  `*-check.log`.
- Consumer needs come from `.flow/specs/fn-{217,228,233,211}*.md` and their `.json` status and
  dependency fields.
- Rationale comes from `model/design/second-opinion.md`, fn-210 and fn-212 specs, DESIGN.md
  §TLC/§Readiness/§Measurements and `model/archived/README.md`.
- Line projections for the remaining tasks are estimates scaled from completed tasks and each
  task's Files list. They are labelled as estimates throughout.
