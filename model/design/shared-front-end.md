# Shared front end

Design for fn-238.1, written 2026-10-21 at `ae4527965f`, after fn-227.13 (one `#[machine]` generator)
and fn-227.14 (derived value handoff). Paths are relative to `model/`. Line counts are physical
lines of the `src/` ranges cited, counted with `sed -n 'a,bp' | wc -l`.

**Status, 2026-10-22.** Option A landed in .2–.4, with .8's guest identity fix. The core crates
went from 8,190 to 8,297 lines (+107), missing the −850 to −1,600 estimate below. The owner
confirmed that .5 and .6 (Option B) stay deferred. The boundary as built, the measurements and
the reasons for the miss are in [DESIGN.md](../DESIGN.md#front-end-and-back-ends-fn-238).

## Decision

**Take a staged A-then-B path.**

- **Option A, a shared admission front end, comes first.** A new crate `umpire-front` reads each
  `#[machine]` module once, inside the proc macro. It produces a typed admitted structure and the
  lowered body IR, with located diagnostics. The macros generate the executable Model from that
  structure. The lifter stops reading source: it emits TLA+ from the IR, which the compiled Model
  carries. A delivers spec R2–R4 and most of the line reduction, at no cost to native speed,
  debugging or the wasm guest.
- **Option B, the native checker interpreting the shared IR, comes second.** It is the owner's
  "the native checker acts like an interpreter". It starts as a differential check in the default
  gate: every admitted Model is explored both interpreted and compiled, and the two results must be
  identical. The interpreter becomes the checker's execution path only if that check is green and
  a measured performance gate passes (task .6).
- **The guest and typed replay keep executing the compiled Rust bodies in both stages.** rustc's
  reading of a handler therefore stays a cross-check on the front end's reading. That cross-check
  runs in the default gate, with no JVM involved.

The order follows from one measurement. Today the duplication sits almost entirely in reading
declarations: attributes, registration, schema, domains, starts and claim signatures. Handler and
claim bodies are read only once, by the lifter, because rustc gives the native side their meaning
for free. A removes the declaration duplication. B adds a third reading of every body construct:
front end, interpreter and emitter. That is the archived Scala tree's "every operator written three
times" (fn-192), and it is why B on its own neither shrinks the code nor halves the per-construct
cost.

## Today's split

- **Macros: registration only.** `umpire-dsl-macros` (3,306 lines) reads attributes, registration,
  schema, domains and starts. It generates adapters that call the compiled method bodies
  (`dsl/macros/src/machine.rs:841-920`, `:969-998`). It never lowers a body. Its only body-level
  checks are token scans for `stay!`/`reject!` (`machine.rs:28-83`), validation of `path!` claims
  (`machine.rs:374-397`, `path.rs:727-759`) and `&mut self` guards (`machine.rs:148-323`).
- **Lifter: a second reading.** `umpire-tla` (4,884 lines) parses `model.rs` again with `syn`
  (`lift/src/lib.rs:233-267`). It re-admits the same declarations (`lift/src/shape.rs:14-505`), lowers
  the bodies into a typed guarded-command tree, and emits TLA+:
  - **The tree.** `lift/src/tree.rs:469-555` defines `Branch { guard, updates, checks, outcome }`,
    and `lift/src/expr.rs` holds the typed expressions.
  - **Emission** is in `lift/src/emit.rs`.
  - **Values come from native code.** World values, complete starts, `#[domain]` values and the
    concrete catalog cross from the compiled Model through fn-227.14's `Values`/`ActionValues`
    exports (`dsl/src/value.rs`). They are then validated again (`tree.rs:65-468`).
- **Execution.** The native checker (`checker/src/explore.rs:77`) and the wasm guest
  (`wasm/src/runtime.rs:67`, `:321-327`) both execute the compiled `Machine::checked_step`.
  Typed replay of TLC counterexamples also runs through the compiled `checked_step`
  (`testing/src/backend.rs:142`, `:306`).

The subset gap is real. Of 20 Model fixtures, 6 carry a located lift refusal in their `lift.snap.txt`,
including the only Temporal Model (`temporal/feature/activity/expected/lift.snap.txt`: "trait or
generic impl in machine module is unsupported"). The native checker checks all 20.

## Measured duplication

Each row is one concern that both crates implement independently. Every listed range was read.

| Concern | Macros (`dsl/macros/src/`) | Lines | Lifter (`lift/src/`) | Lines |
| --- | --- | ---: | --- | ---: |
| Attribute kinds, exclusivity, order, legacy forms, registration loop | `machine.rs:10-15`, `:103-121`, `:399-489`; `machine/registration.rs:86-174` | 205 | `shape.rs:227-241`, `:374-456`, `:598-612` | 113 |
| Module shape: one State, actors, generics refusal, declared World, impl owner | `machine.rs:85-101`, `:123-134`, `:549-636` | 117 | `shape.rs:14-40`, `:92-206`, `:214-226`, `:262-278`, `:563-570` | 180 |
| `&World` parameter classification (once, last, immutable) | `machine/registration.rs:57-84` | 28 | `body/mod.rs:94-178` | 85 |
| Starts: field ranges, computed `#[starts]`, "declare starts once" | `machine/registration.rs:30-55`; `machine.rs:638-657`; `state.rs:190-216` | 73 | `shape.rs:279-316`, `:522-552` | 69 |
| Handler signature (actor enum, `&mut self`, no return) | `machine.rs:325-359`; `machine/registration.rs:8-28` | 56 | `body/effects.rs:12-32` | 21 |
| Claim signatures (invariant, postcondition arity, `after =`) | `machine.rs:361-372`; `machine/registration.rs:135-150` | 28 | `body/claim.rs:10-59` | 50 |
| Progress, `path!` claim, assumptions | `machine.rs:374-397`, `:960-967`; `path.rs:727-759` | 65 | `body/progress.rs:12-225` | 214 |
| Mutable receivers outside `#[on]`/`#[effect]` | `machine.rs:148-323` | 176 | `shape.rs:431-440` | 10 |
| `stay!()` / `reject!` detection | `machine.rs:28-83` | 56 | `body/effects.rs:222-242` | 21 |
| Action input domains (`#[inputs]`, `#[domain]`, Finite, misplaced attributes, integer widths) | `action.rs:30-66`, `:165-178`, `:718-780`; `finite.rs:80-148` | 183 | `body/inputs.rs:18-240`; `body/typer.rs:54-68` | 238 |
| State/World field schema and admitted types | `state.rs:141-150` | 10 | `body/typer.rs:10-52` | 43 |
| Local fieldless enums and their values | `machine.rs:689-715` | 27 | `shape.rs:121-150` | 30 |
| **Duplicated reading** | | **1,024** | | **1,074** |
| Re-validation of natively supplied values (World, starts, domain product, catalog order) | — | — | `tree.rs:65-468` | 404 |
| File-level reading that rustc already does for the macro (canonical imports, aliases) | — | — | `shape.rs:41-76`, `:613-654` | 78 |

The table supports four findings.

- **About 2,100 lines read the same declarations twice.** Another 480 lines exist only because the
  second reading is separate: it must re-validate the values the native side computes, and it must
  re-resolve imports that rustc has already resolved for the macro. The fn-227 audit measured the
  same thing from the other side: World took 1,155 native lines against 1,144 source lines, and
  inputs took 1,077 against 1,493. About half of each source share was binding plumbing
  (`.flow/artifacts/fn-227-parameterized-models-declared-worlds/audit-report.md`, §1.1–1.2).
- **The copies have already drifted.** Two examples:
  - The macros accept any `#[derive(State)]` name (`machine.rs:558-563`). The lifter requires
    `Model` (`shape.rs:269-271`).
  - The macros admit action inputs without a declared World (`action.rs:94-97`). The lifter refuses
    them (`shape.rs:338-344`).

  The same refusal is also often worded differently: "declare World once as
  `#[derive(World)] struct World`" (`machine.rs:622`) against "declare one World" (`shape.rs:160`).
  Some messages were copied verbatim instead, for example "the World parameter must be an immutable
  &World" (`registration.rs:77`, `body/mod.rs:121`).
- **The lowered bodies are not duplicated.** About 1,380 lifter lines have no macro counterpart:
  - `body/expr.rs`
  - `body/effects.rs:33-221`
  - `body/vector.rs`
  - `body/claim.rs:60-154`
  - the pattern matcher in `body/inputs.rs:244-439`
  - `expr.rs`
  - the declarations in `tree.rs`

  This is the only part of the lifter that interprets meaning, and the only part Option B would
  have to implement a second time.
- **The value types are also duplicated, in small form.** `dsl/src/value.rs` (`Value`) mirrors
  `lift/src/lib.rs:103-113` (`Literal`), and `lib.rs:24-50` (`Type`) is the lifter's own schema.

The remaining macro code, about 2,280 lines, is generation and stays a back end in both options:
derives, catalogs, adapters, helpers and the `path!` query language. Emission is about 1,000
lines (`emit.rs` 746, `names.rs` 283) plus fixture glue (`build.rs` 241).

## Option A: shared admission front end

**Crate.** Add `umpire-front` at `umpire/front/`. It is an ordinary library on `syn`/`quote`, with
no Temporal and no runtime dependencies, so the boundary scanner applies unchanged. It owns:

- the attribute grammar, registration and claim order;
- the State/World/action schema with admitted types;
- input domains and starts;
- `&World` classification;
- the admitted expression and statement subset, lowered to today's tree (`Branch`, `Expr`,
  `ClaimDef`, `ProgressDef`, `AssumptionDef`);
- one `Diagnostic { line, column, message }`. The consumer supplies the file name, so today's
  `lift.snap.txt` text (`path:line:col: warning: …; TLA+ emission refused`) is reproduced exactly.

The IR data types (`Type`, `Literal`/`Value`, the tree) move to a small runtime module that the
compiled Model can name. Model crates already depend only on `umpire-dsl`, so `umpire-dsl` re-exports
it `#[doc(hidden)]`. The IR stays internal, with no stability promise and no checked-in IR file
(fn-201's lesson).

**Two kinds of refusal.**

- **Structural refusals are compile errors**, as today. The macro cannot generate a Model without
  that structure; one example is an actor without `#[on]`.
- **Body-subset refusals are soft.** The macro still generates the executable Model and embeds the
  refusal in place of the IR. The lifter reports the refusal exactly as it does now. The activity
  Model and the five refused fixtures therefore keep checking natively, and their lift goldens keep
  their bytes.

This keeps today's authoring contract: TLC wherever the subset admits the Model.

**Where it runs.** The front end runs inside `#[machine]`, which already receives the module's
tokens. The macro emits a hidden `fn __umpire_lifted() -> Result<Lifted, Diagnostic>` that builds
the IR as data. It also emits a set of the source identifiers, which `names.rs` uses for TLA+
collision checks. The function is gated off `wasm32`, so guest bytes need not change.

The lifter then becomes `Tla::emit(&M::lifted(), values)`, called at test time with the
same-process World, starts and catalog. This route is what bound fixtures already take today
(`lift/src/build.rs:25-34`). It removes the following:

- the second `syn` read of `model.rs`;
- the canonical-import and alias checks, because rustc has resolved the macro path;
- most of `tree.rs:65-468`. Schema and catalog order can no longer disagree, because they come from
  one reading. The TLC `.cfg` integer range check and the value-type checks stay.
- the build-script source path.

The test-only `fixture!` hook's supplemental negative claims (`body/claim.rs:60-101`) keep the
one small read of `tests.rs` they have today. Moving them into a proc macro is a later cleanup, not
part of this migration.

**This reverses the 2026-10-19 choice.** That decision chose a build-time `syn` pass over token
recording (`design/second-opinion.md:142-160`) for three reasons: one route for both attribute and
manual Kettles, retained file locations, and registration-only macros. None of the three holds any
longer:

- fn-223 deleted the manual source form.
- The owner (2026-10-21) traded independent admission for shared code.
- Proc macros on Rust 1.88+ can recover locations, either through stable `Span` line/column or
  through `quote_spanned!(span=> line!())`. Task .4 must confirm this on the pinned 1.99 toolchain
  before it deletes the source read.

**Native.** Unchanged. The macros generate the same adapters, calling the same compiled bodies.

## Option B: shared semantic IR, interpreted natively

**Interpreter.** The checker gains `Interpreted<M>`, a `Machine` whose `step` evaluates the
embedded IR:

1. Guards in written order, first match wins (`effects.rs:40-67` already produces the effective
   guards).
2. Updates.
3. Checked-arithmetic checks.
4. `stay!`/`reject!` outcomes.

Claims evaluate `ClaimDef`/`PostDef`. Progress and assumptions come from the `ProgressDef` and
`AssumptionDef` the front end already produces.

**State representation.** The choice decides most of B's cost.

- **Slot vector (`Vec<Value>`).** This is the simplest interpreter, but it changes `M::State`. Every
  report, rows, states and paths golden renders `Display` of the typed state, and Realization code
  reads typed fields (`Realize::observe(.., state: &M::State)`, `wasm/src/runtime.rs:67-78`). A slot
  vector would break both, or need a typed rendering bridge.
- **Typed state through derived slot accessors (preferred).** `derive(State/World/Action)` would
  generate `get(slot) -> Value` / `set(slot, Value)`. The checker keeps `M::State`, so the goldens and
  Realization are untouched, and only body evaluation is dynamic. The catch is composites: every
  composite field type also needs `FromValue`. fn-211 R6 needs that inverse anyway, to build a
  World from decoded constants.

**Guest.** The guest keeps executing the compiled bodies. Interpreting there would need the IR and
the interpreter in the guest bytes, and would gain nothing at a few steps per observation. The
cost is that the checker (interpreted) and the guest (compiled) would execute two readings of one
Model. Stage B therefore requires the differential check in every gate, on every admitted Model:
same states, edges, outcomes, claim verdicts and rows.

## Comparison

| Criterion | A: shared admission | B: interpreted shared IR (after A) |
| --- | --- | --- |
| **Code size** (today: macros 3,306 + lift 4,884 = 8,190) | `umpire-front` about 3,200–3,500 (lifter admission and lowering moved, merged with the macros' reading, plus an IR-to-tokens printer of about 250). Macros about 2,350–2,450 (−1,024 reading, +100 glue). Lift about 1,300–1,500 (emission, names, API). `testing/` glue −100 to −250. **Net about −850 to −1,600 lines (10–19%).** | Adds the interpreter (900–1,300: expression and effect evaluation, Seq/Int ops, checked arithmetic, outcomes, catalog from IR domains) and accessor generation (100–150). The compiled adapters stay for the guest, replay and the differential check. **Net about −550 to +600 against today**, which gives back most of A's reduction. |
| **Per-construct cost: declaration-level** (World, inputs, new attribute or claim kind) | Read once; no binding plumbing. fn-227 measured World 1,155 + 1,144 and inputs 1,077 + 1,493; under A such a construct costs **about 55–60% of today**. This is where the spec's "roughly half" holds. | As A, plus a small interpreter share (inputs: catalog product and payload patterns, about 150–250). |
| **Per-construct cost: body-level** (Quint values and operators) | Unchanged from today: one front-end lowering plus emission and decoding. The `imbl::Vector` precedent is `body/vector.rs` (261) plus its emit/decode share. Native support is free from rustc. | **+35–55%.** Each construct also needs an interpreter case and a typed `Value` bridge (a `FromValue` for composite types). This is the fn-192 "written three times" cost. |
| **Authoring limits** | Unchanged. Native accepts any Rust; TLC applies where the subset admits it, with one located refusal from the front end. Making refusal a hard error is a separate policy, not needed here. | The interpreter covers only admitted Models. The activity Model (`impl Phased`, generic `closable`) would stay compiled until the subset grows, so a checker that "acts like an interpreter" holds only for liftable Models. Plain Rust stays: no IR authoring, as in the Scala lesson. |
| **wasm guest and Realization** | Unchanged: compiled bodies, typed `M::State`. The `__umpire_lifted` fn is gated off `wasm32`; otherwise refresh the guest the standard way. | The guest stays compiled. The checker's reading would differ from the guest's unless the differential check runs. The slot-vector design would break `Realize<M>`; typed accessors avoid that. |
| **Native checker performance** | Unchanged: Kettle runs 283 states in 66–103 µs in release (`DESIGN.md:771-780`), 2.7–4.3 M states/s. | Measured core loop below: interpretation costs **1.3–3.2× with typed accessors** and **4–6.4× with slot vectors**. Projected whole-checker cost on Kettle is about 1.2–1.3× (typed) to about 2× (slots). Heavier bodies move toward the core ratio. Slot vectors also make each state about 15× larger (120 B against 8 B for Kettle). |
| **Debugging** | Unchanged. Breakpoints, `dbg!` and panics inside handlers behave the same in the checker, tests and guest. One front-end diagnostic replaces two drifting texts. | Worse in the checker: it no longer runs the author's Rust. `dbg!` and breakpoints in handlers do nothing during exploration, and overflow or panic messages must be rebuilt from IR spans. The gain is that a TLC disagreement can only be an emission bug, because the front end is already checked by the differential run. |
| **Five TLC configuration families** (main `emit.rs:224`, invariant `:234`, postcondition `:255`, negative `:268`, progress `:378`) | All five are emitted from the IR and are byte-identical while the IR is. The CONSTANT block comes from the same-process World instead of `Binding`, with the same text. | No further effect: emission is shared. |
| **Typed replay** | Unchanged: TLC counterexamples replay through the compiled `checked_step` (`testing/src/backend.rs:142-306`). | Deliberately unchanged: replay stays on the compiled bodies, so rustc's reading still cross-checks both the front end and emission. |
| **fn-211, including symbolic Worlds (R6)** | fn-211 consumes the IR. Snowcat annotations and the safety projection are emission-only. R6 needs two things from the front end: lowered `#[starts]` and `#[domain]` bodies, or a per-World start table that the embedded route builds cheaply in-process. It also needs a `FromValues` World for replay. | Same front-end work. The interpreter's pure-expression evaluator (B's first slice) gives a natively confirmed result for every World: lifted starts evaluated per World must equal the compiled starts. R6 needs that check anyway. |
| **Quint parity** | The value side is front-end lowering of Rust/`imbl` types plus emission; the native side is free. This covers sets, maps, records, tuples, `Option`, payload enums, `pure def` helpers and module composition. Nondeterministic choice goes through action inputs or ordered choices, in the DSL runtime. Temporal operators and fairness are checker algorithms plus emission. Symbolic constants are emission plus R6. | The same constructs, each also implemented in the interpreter and the composite `Value` bridge. Nothing in Quint's surface needs the native side to interpret rather than run compiled code, and native checking cannot be symbolic in either option. |

## Interpreter cost evidence

The probe is `.flow/tmp/fn2381-interpreter-probe.rs`, scratch only. It is built with `rustc -C
opt-level=3 -C codegen-units=1 -C overflow-checks=on` and run three times. It explores Kettle
breadth-first from its 81 starts and checks one invariant and one postcondition per edge. It runs
both compiled and as an interpreter of the same guarded-command branches the lifter builds. A wide
variant uses 2,001 starts: 4,903 states and 6,904 edges.

| Variant | Kettle (283 states) µs | Ratio | Wide (4,903 states) µs | Ratio |
| --- | ---: | ---: | ---: | ---: |
| Compiled, typed state | 15.2–23.9 | 1 | 212–431 | 1 |
| Tree-walk, typed state via slot accessors | 28.2–35.7 | 1.3–2.2× | 467–1,383 | 2.1–3.2× |
| Tree-walk, `Vec<Value>` slots | 83.7–96.3 | 4.0–6.4× | 1,233–2,265 | 5.3–6.4× |
| Closure-compiled IR, `Vec<Value>` slots | 86.7–97.9 | 4.1–6.0× | 1,061–1,673 | 3.9–5.4× |

How to read these numbers:

- **The probe measures only the core loop.** It includes `step`, the claims and hashed insertion,
  but none of the checker's other bookkeeping. The real checker spends about 45–50 µs per Kettle
  exploration outside that loop, which dilutes the interpreter's share.
- **Allocation and hashing of slot vectors dominate their cost.** Closure compilation does not
  help.
- **TLC is far slower per run.** The required fixture run executes about 44 TLC configurations per
  render (14 main plus 30 property configurations). The double render took 90.94 s
  (`.flow/tmp/fn22714-full2-model-check.log:708`), about 1 s per JVM run. The whole required gate
  took 460 s, against 50.5 s for the default gate.
- **So native speed is not what limits any gate today.** B's slowdown matters for the inner loop
  and for native explorations of larger Models, not for scaling, which the owner assigns to
  TLC/Apalache.

## What the archived Scala tree learned

The Scala tree went IR-first, then moved away from it.

- **Go interpreted the IR at first.** "Scala writes it …; Go reads it … Go never runs Scala code"
  (`archived/README.md:202-206`). fn-113 deleted the native evaluator as "the largest
  duplication" (`.flow/specs/fn-113-*.md:117`).
- **fn-174 then made the Model construct its own IR.** Every DSL construct "is described twice",
  and the lifter had reached 12,541 lines (`.flow/specs/fn-174-*.md:30-34`).
- **fn-196 reversed the IR-first choice on 2026-10-17.** It made the compiled checker the oracle,
  because "the step through the IR and a Go interpreter is slow and indirect … one oracle is
  enough" (`.flow/specs/fn-196-*.md:14-30`). It rejected both keeping two checkers and building the
  IR evaluator first (`:150`).
- **fn-201 kept the IR internal**, "a tree … that text emitters walk", because a public IR keeps
  "two sources of truth" (`.flow/specs/fn-201-*.md:17`, `:60`).

| Reason recorded then | Applies now? |
| --- | --- |
| Interpretation was slow. The Go check took 0.7 s on Kettle, 48.6 s on Bank and 117 s for all samples; it tracked "catalog × classes, not reachability", "the interpretation runs twice", and GC used 58% of CPU (`archived/framework/design/reachable-exploration.md:226-236`, `:1995-2004`). | **Partly.** A Rust interpreter would explore reachable states, not full tables. The probe measures 1.3–6.4× instead of orders of magnitude. The cost is real but not decisive. |
| "Every operator … written three times": lifter, Go interpreter and Quint (`.flow/specs/fn-192-*.md:18`). | **Yes, for B.** The front end, the interpreter and the TLA+ emitter would be three implementations, with Apalache annotations a fourth concern. A keeps two, because rustc supplies native meaning. This is the main reason B comes second and only behind a gate. |
| One oracle is enough; two checkers were rejected (fn-196). | **Changed by the owner.** Both checkers stay (fn-238 Decision Context). B's interpreter is introduced as a self-check against compiled bodies, not as a third oracle with its own verdicts. |
| A public IR means two sources of truth (fn-201). | **Yes.** The embedded IR stays `#[doc(hidden)]` and is never checked in. Goldens stay TLA+, cfg and reports. |
| Macro pickling and staleness, and lost caller names (fn-174:49-58). | **No.** Rust proc macro output is regenerated on every compile, and the front end sees the author's tokens and names. |
| Authors must write plain `match`/`if`/`copy`, not an IR DSL (`archived/framework/design/scala-macros-export.md:554-559`). | **Holds and is kept.** Both options lower plain Rust. B only narrows what the native checker interprets. |
| Portability: Scala Native and wasm were blocked (fn-209, fn-210). | **No.** Stock Rust covers both options. |

## Migration

All tasks follow one preservation plan.

**Baseline.** Before .2, write `.flow/tmp/fn238-baseline.sha256` over every golden:
- report, rows, states and paths;
- `@tla`/`@cfg`, `tlc.snap.txt` and `lift.snap.txt`;
- expansion snapshots;
- trybuild `.stderr`;
- `negative_report`, `coverage` and `realization`;
- the embedded guest bytes.

**Checks on every task:**
- The task's quick gates, default and `UMPIRE_BACKENDS=require`, compare against the baseline. Only
  the differences a task lists are allowed. There is never a global `model-update`.
- A scratch IR dump (`{:#?}` of every admitted fixture's tree, in `.flow/tmp`, not committed)
  compares the IR before and after .2–.4, so that drift shows up before it reaches emission.
- TLC agreement is part of each full receipt.
- Typed replay keeps calling the compiled `checked_step`.

| Task | Scope | Acceptance | Depends on |
| --- | --- | --- | --- |
| **.2 `umpire-front`: move the lifter's admission** | New crate `umpire/front/`. Move `shape.rs`, `body/*`, `expr.rs`, `syntax.rs` and the declarations in `tree.rs` from `umpire-tla` with no behaviour change. `Diagnostic` carries line, column and message; the consumer formats the file. The lifter calls `umpire_front::read`. | Every golden is byte-identical, including all `lift.snap.txt`. Lift tests pass unchanged or move with their code. Line counts are recorded. | fn-227.14 (done) |
| **.3 Macros read through the front end** | Replace the macro readers in the duplication table (1,024 lines) with `umpire-front`'s structure. The macros keep generation only. Settle the two drifts: the State name and inputs without a declared World. Keep the native-accepting form, and widen the front end and lifter where emission already supports it; otherwise use a located refusal. Unify each pair of differing messages into one front-end text. | One diagnostic per unsupported form, the same for both consumers. Allowed golden changes are only the listed trybuild `.stderr` / `lift.snap.txt` messages, each justified in the task. Expansion snapshots, checker, TLA+/cfg and TLC goldens are byte-identical. | .2 |
| **.4 Embed the admitted Model; lifter emits from it** | The macro emits `__umpire_lifted()` (off `wasm32`) and the source identifier set. The lifter emits from the compiled Model and same-process values. Delete the `model.rs` re-read, the canonical-import checks, the tautological parts of `tree.rs:65-468` and the build-script source path. Keep the `.cfg` range and value-type checks, and the `tests.rs` hook read. First a spike: located diagnostics inside the macro on 1.99. | TLA+/cfg, TLC agreement, replay, `lift.snap.txt` (file path from the runner) and guest bytes are all byte-identical. If guest bytes change, refresh them in a separate commit with the slice and gap gates. Measured deletion against the −850 to −1,600 estimate. `make model-time` recorded. | .3 |
| **.5 Interpreter as a differential check (B, step 1)** | `Interpreted<M>` in `umpire-checker`: typed state through accessors generated by `derive(State/World/Action)`, evaluation of the embedded IR, outcomes, checked arithmetic. The fixture runner explores every admitted Model both ways and requires identical states, edges, outcomes, claim verdicts and rows. It writes no new goldens; a difference fails. Include fn-211 R6's pure-expression evaluation of lifted starts, if fn-211 has landed the lowering. | Every admitted fixture agrees. No golden changes. Measured interpreted/compiled ratio, with the fn2107 driver and the probe shapes. The default gate stays within its 60 s and 10 s soft targets. | .4 |
| **.6 Interpreted checker by default (B, step 2, gated)** | Switch `Checker` to `Interpreted<M>` for admitted Models. Refused Models stay compiled. The guest and replay stay compiled. The differential run from .5 stays in the gate. **Gate:** .5 green, a whole-checker slowdown of at most 2× on Kettle and the wide shape, and a measured per-construct cost on the first Quint-parity construct (sets) no more than 25% above A. If the gate fails, stop at .5 and record why. | Every golden is byte-identical. Typed replay is unchanged. `make model-time` is recorded and within its soft targets. | .5 |
| **.7 Boundary write-up and measurements (spec R4)** | DESIGN.md, describing the front end / back end boundary and retiring "independent source admission" for new work. MILESTONES updated. Line counts per crate, the `make model-time` output, and the .5/.6 ratios. | The numbers are reproducible with the commands given. | .4; update after .6 if taken |

fn-211 consumes the front end from .4 onward. Its R6 start lowering can land as an fn-211 task
on top of .4, and .5's evaluator then confirms it natively.

## Decisions taken on the owner's behalf

- **Recommend staged A-then-B, not B alone.** B alone keeps every compiled adapter, because the guest
  and replay still need them. It adds an interpreter, and it costs more per body construct than A.
  The owner's "acts like an interpreter" is reached in .6, behind a measured gate. It is not
  assumed.
- **Subset refusals stay soft.** A `#[machine]` whose bodies the front end refuses still compiles
  and checks natively, so the activity Model keeps working. Making refusal a hard error would be an
  authoring-policy change for a later spec.
- **The guest and typed replay always run compiled bodies.** rustc's reading stays the cross-check
  on the shared front end, in place of the independence fn-238 gives up.
- **The IR is embedded from the macro, not re-read from the file.** This reverses
  `design/second-opinion.md`'s 2026-10-19 choice, for the reasons given under Option A.

Still open, for the owner:
- the .6 gate thresholds (2× and 25%);
- whether `fixture!` negative claims should later move into a proc macro, so that no source file
  is ever read.
