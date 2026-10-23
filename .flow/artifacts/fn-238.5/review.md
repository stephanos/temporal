# fn-238.5 review: interpreted checker as a differential check

Commit 4820633b9b (`work/fn2385-interpreted`), base 8df7e6f142. Reviewer had fresh context. No source was changed. Every probe was reverted, and the worktree is clean.

## Verdict: NEEDS_WORK

The core is sound. Step semantics, checked arithmetic, `check!` text, claims and progress all hold up. I checked this by reading the code and with a probe of 236 Worlds (below). Two things block SHIP:

- One confirmed semantic divergence on an admitted Model that is already in the repo: start-domain laziness.
- The acceptance claim "every admitted fixture agrees" is wider than what the gate runs. The Temporal activity and the backend fixtures are not in the differential.

Both fixes are small.

## What I verified

- **Tests.** `cargo test -p umpire-fixtures --test interpreted` passes, and so does `--test fixtures` (including `interpreted_agrees_with_compiled`). These also pass:
  - `umpire-lift`
  - `umpire-dsl-macros` (expansion snapshots and trybuild)
  - `umpire-front`
  - `umpire-checker`
  - `umpire-apalache` (emission without the tool)
  - `umpire-testing`
  - `umpire-ir`
- **Goldens.** The commit touches no golden, snapshot, `.stderr` or `lift.snap.txt` file. `ExprKind::Check` is read through correctly in `prime`/`render` (emit.rs:717,778), `rename` (names.rs:220) and the Apalache walker (safety/value.rs:58). The walker's memo is keyed on the inner Expr, so sharing is unchanged. `infallible()` delegates. `Lifts`/`Interprets` are `cfg(not(wasm32))`, so the guest is unaffected.
- **Expansion snapshots.** They stay byte-identical, but the reason differs from the handoff's. The snapshot harness (`dsl/macros/tests/fixtures.rs`) compiles only `src/machine.rs`, never `lifted::embed`, so neither `Lifts` nor the new `Interprets` impl is snapshotted. The author-visible surface is unchanged: the trait is `#[doc(hidden)]` under `__lifted`, the impl is `#[doc(hidden)]`, and there are no new attributes.
- **Step semantics.** They match `stepping()` (machine.rs:450-500):
  - Front `Branch.checks` holds every fallible value in source order: lets, assignments, if-conditions and pushes (effects.rs:134,165,186,194,203). Evaluating checks before outcome and updates therefore reproduces Rust's first panic, despite lazy locals and the field-order updates.
  - The guards are already the front's exclusive effective guards.
  - `RejectedAfterMutation`, `stay!` and fixed rejections behave as compiled. Facts are always empty, on both sides.
- **Arithmetic.** It is i128 with a range check against `expr.ty`. Admitted ints are at most 32-bit (typer.rs:19), so i128 never overflows natively.
  - All eight panic messages match Rust: div/rem by zero, `MIN/-1`, `MIN%-1` and negate.
  - Truncating division and the sign of the remainder match.
  - The workspace sets `overflow-checks = true` in dev and release, so the release timing is comparable.
- **Claims.**
  - `check!` keeps going after a failure, and `Checks` evaluates every statement, which matches.
  - Claim `if`/`else` lowers to `Or(Not c, …)` / `And`, and short-circuiting gives exactly the compiled branch.
  - The verdict comes only from `claim::fail`. That is correct, because the front admits only `check!`, `if` and laws in claim bodies.
  - Postconditions bind `self` to `Next` (after) and the parameter to `Field` (before), and `after = Action` uses exact action equality on both sides.
  - Progress and assumptions match too.
- **Check text.** `__umpire_text!` re-stringifies through an `$expression:expr` capture, the same as `check!`. Text from law bodies comes from library source and is only stringified.
- **R2 coverage of `compare`.** It covers:
  - `states.txt`;
  - `rows.txt` for every state (edges, choice, outcome, facts, target; built from the stored transitions);
  - `report.txt` (verdicts, counts, warnings);
  - `paths/*`;
  - `negative_report.txt`.

  That covers states, edges, outcomes, claim verdicts and rows. Interpreted counts are dropped (`Ok((actual, _))`), but they appear in `report.txt`.
- **`imbl`.** It is an existing workspace dependency, already used by `umpire-dsl`, `umpire-lift` and `umpire-apalache`. In `umpire-fixtures` it is a dev-dependency only, needed because `lift/tests/fixtures/vector*.rs` write `use imbl::Vector`. Cargo.lock adds an edge only, and there is no new third-party crate.
- **Nothing Temporal under `model/umpire`.** None of the added lines mention Temporal or the activity.
- **Timing reproduced** (release, host load about 9–13): Kettle 90.5 µs vs 233.9 µs (2.58×), wide 1.507 ms vs 3.974 ms (2.64×). This matches the handoff's 2.59× and 2.60×. The method is fair:
  - each repetition runs compiled then interpreted;
  - medians are taken;
  - only `explore_within` is timed, claims included;
  - both sides use the same starts.
- **Probe 1, Temporal activity.** I added `umpire_testing::interpreted(world, None, 64, None)?` to `temporal/feature/activity/tests.rs::run()` temporarily, then reverted it. Result: **agree**, 18 states, 48 edges, about 2.5×.
- **Probe 2, backend fixtures.** I ran all `umpire/backends/apalache/tests/fixtures/*.rs` and tlc `scaling_model.rs` through `interpreted()` in a temporary test, then reverted it. Arithmetic was swept over mode 0..=10 × start {-128,-1,0,1,127} × divisor {-1,0,1,2}. **233 of 236 agree.** This is strong evidence that division, remainder and overflow behave correctly. The three that do not agree are `start_*` (finding 1).

## Findings

### 1. major: start domains are evaluated eagerly; compiled evaluates them lazily, nested in field order
`model/umpire/checker/src/interpret.rs:294-313`

`starts()` evaluates every field's domain before `product`. The compiled `derive(State)` nests one `for` per field (`dsl/macros/src/state.rs:164-165`), so a later field's range expression is evaluated only inside each value of the earlier one. An empty earlier domain never evaluates the later one.

**Failure, confirmed by probe.** The repo's admitted Model `umpire/backends/apalache/tests/fixtures/start_empty_first.rs` has `spare: #[starts(1..0)]` before `value: #[starts(1..=(LIMIT+1))]`:
- compiled `starts()` is `[]` (and `controls.rs:325` asserts exactly that);
- `Interpreted::new(..).starts()` panics with "attempt to add with overflow".

The file's own doc comment states the rule: "An empty outer range prevents evaluation of the later overflowing endpoint". A legal Model gets different behaviour, and the differential never runs it.

**Fix:**
- Evaluate each domain lazily inside `product` when that depth is first reached. Memoising per depth is fine, because domains read no state and the first evaluation decides any panic.
- Add `start_empty_first`, `start_fault_first` and `start_later_fault` as controls to the `interpreted` target.

### 2. major: the differential does not cover every admitted Model, and the docs and handoff say it does
- `model/umpire/fixtures/fixtures.rs:68-98`
- `model/DESIGN.md:84`
- `model/README.md:792-793`
- handoff "Acceptance [x]"

The task and design ask that "the fixture runner explores every admitted Model both ways". The gate covers umpire-fixtures, two checker controls and the seven lifter Models. Not covered:
- **The Temporal activity**, whose `lift.snap.txt` says "admitted". It is the largest and most important admitted Model. Its hand-written `tests.rs` defines no `interpreted()`, and `umpire-temporal-tests` has no equivalent of `interpreted_agrees_with_compiled`.
- **The admitted backend Models**: Apalache `arithmetic`, `arithmetic_prefix`, `categories`, `empty_condition`, `replay`, `start_*`, and tlc `scaling_model`.
- **The checker fixtures** `progress_chain_path` and `rejecting_mutation`.

DESIGN.md says "Every fixture whose World embeds an admitted Model … explores identically", and README says the runner "runs it for every fixture". Finding 1 shows what the gap costs: the one in-repo divergence sits in an uncovered Model.

My probes show the activity and 233 backend Worlds agree today, so this is a coverage gap, not a hidden failure. Without it, though, .6's "differential stays in the gate" would not protect the activity.

**Fix:**
- Add a one-line `umpire_testing::interpreted(world, None, 64, None)?` to the activity's `run()`, or an `interpreted` test in `umpire-temporal-tests`. Coordinate with lane A.
- Add the backend Models to `umpire/fixtures/interpreted.rs`, with the arithmetic mode sweep.
- Narrow the DESIGN/README wording to what is actually covered, or extend the coverage until it matches.

### 3. minor: a start panic escapes the differential instead of being compared
`model/umpire/testing/src/interpreted.rs:98`

`explored()` calls `world.starts()` outside any `catch_unwind`. When the compiled starts panic (`start_fault_first`, `start_later_fault`), or only the interpreted ones do (finding 1), `interpreted()` panics. It never reports "compiled: X / interpreted: Y". Two equal start faults can never be confirmed as agreeing.

**Fix:** catch the panic around `starts()` on each side, render it as a refusal string carrying the panic message, and compare those strings as `compare` already does for refusals.

### 4. minor: length arithmetic is range-checked against a 32-bit index type, but Rust computes it in `usize`
- `model/umpire/checker/src/interpret.rs:762-764, 943-971`
- root cause `model/umpire/front/src/body/vector.rs:10-12`

The front types `len()` and its arithmetic as `Int{0, i32::MAX}`. The interpreter then panics on any intermediate value above `i32::MAX`, while the compiled `usize` arithmetic succeeds.

**Failure, confirmed by probe.** I used a `Vector<u8>` Model that grows to length 3, with `check!(self.values.len() * 1_000_000_000 != 7)`. The front admits it. Compiled gives "outcome: Verified"; interpreted gives "outcome: Refuted", with "attempt to multiply with overflow".

The values involved are contrived, so the severity is minor. Still, it is a wrong verdict on an admitted Model, and the front misreading reaches both back ends.

**Fix:** either give the front a `usize`-wide index type for Length-derived arithmetic (and have emission check it the same way), or refuse arithmetic on `len()` beyond comparison and indexing. Add the probe as a control.

### 5. minor: the 64-claim cap is honest but untested, and it costs code size
`model/umpire/checker/src/interpret.rs:24, 432-444, 601-609`

The refusal is honest. It happens at construction, the message is clear ("N invariants exceed the interpreter's 64"), and in the fixture differential it is a failure, not a silent skip.

But:
- **No test covers it.** No test builds a Model with 65 claims.
- **Every `M` instantiates 4 × 64 monomorphized callbacks**, all `match` arms, even for a Model with one claim.
- **.6 would turn the cap into a hard refusal** for large Models. The owner expects the Model to "expand dramatically".

**Fix:**
- Add a negative test now.
- For .6, plan either a compiled fallback when the cap is hit, or a `Claim` constructor that carries an index or context, so the cap goes away.

### 6. minor: the mutants and negative tests do not reach every R2 dimension
`model/umpire/fixtures/interpreted.rs:153-186`

The two mutants exercise:
- an update, which shows up in counts in the report;
- a postcondition literal, which shows up as a verdict.

Nothing shows that the comparison catches a changed outcome (Accepted to Rejected on the same target state, which only `rows.txt` shows), a changed start set or order, a progress trigger or target, or a swapped `check!` text index. With `rows.txt` only rendered by the harness, an outcome-only regression has no proof that it would be caught.

**Fix:** add mutants for those four:
- the outcome, changing `Branch.outcome`;
- a `Start::Value`;
- a `ProgressDef.target`;
- a `Check` index.

Assert the named file for each: `rows.txt`, `states.txt`, `report.txt`, and `paths/*` or `report.txt`.

### 7. minor: dead public API
`model/umpire/checker/src/interpret.rs:248-256`

`Interpreted::compiled()` and `Interpreted::check_texts()` have no callers in the workspace. `CLAIM_LIMIT` is re-exported (lib.rs:13) and also unused outside the module.

**Fix:** remove them, or make them private until .6 needs them.

### 8. nit: the handoff says `run()` is unchanged, but two checker controls' `run()` changed
- `model/umpire/checker/tests/fixtures/progress_bounds/model.rs:224`
- `.../progress_explicit_fairness/model.rs:64`

The outputs are unchanged, so this is harmless. In progress_explicit_fairness the inserted line also sits at a different indentation from the existing body. The fixture files are evidently not rustfmt'd.

**Fix:** correct handoff decision 5, and align the indentation.

### 9. nit: a `Check` index has no table outside the macro
`model/umpire/front/src/lib.rs:67-69, 129`; `body/mod.rs:144`

`admit()` discards the check table. Hook-claim environments (`Environment::of`) start a fresh table, so their `Check` indices restart at 0 and point into a table that is never emitted. Emitters ignore the indices, so nothing breaks today. But the IR comment ("by its position in the Model's check table") does not hold for every tree.

A related point: `Check` makes two textually identical `check!`s structurally unequal, so `Expr` equality and hashing on claims changed. No consumer depends on it today, because claim lowering uses `implies`/`and`, not `select`.

**Fix:** add one sentence to the `ExprKind::Check` comment: the index is meaningful only for the tree `#[machine]` embeds.

### 10. nit: acceptance item 2 is marked [x], but the fn2107 driver was not used, and item 3 is open
`.flow/tmp/fn-238.5/handoff.md`

The ratio comes from the new harness on Kettle and on the wide shape. Those are the shapes the .6 gate names, and I reproduced the figures. The task, however, asks for "fn2107 driver and probe shapes", and the default-gate 60 s / 10 s check was not measured (the handoff says so). The ratio is above the 2× gate in .6. It does not block .5, but it should go into the .6 decision as it stands.

**Fix:** mark item 2 [~] with the reason, or run the fn2107 driver. Leave item 3 open until `make model-time` is recorded.

## Not findings (checked and fine)
- Short-circuiting of `&&`/`||`, `If` and claim implications.
- `GetOr` evaluates its arguments in order, as `get().copied().unwrap_or()` does.
- Whole-state `==`.
- Enum, Outcome and Rejection encode/decode use absolute stock paths.
- Raw identifiers.
- Integer decode goes through the exact Rust type.
- `RejectedAfterMutation` parity: it is tested by linear_returns and matches stepping().
- The negative claim, judged by the compiled World on interpreted states, is acceptable as a test hook.
- No new author-visible vocabulary. Comments are mostly "why" comments.
