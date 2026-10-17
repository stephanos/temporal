# Reachable exploration: Umpire models over plain Scala collections

Status: proposal, 2026-10-13, revised the same day on the owner's new basis. No source has been
changed. Paths are relative to the repository root (`/Users/stephan/Workspace/skunkworks/umpire/temporal`).
Line numbers come from the tree this was written against. Anything not confirmed by reading code or
by measurement is marked **(unverified)**.

## Summary

**The basis.** Umpire's semantics stop resting on the whole state catalog. There is one table, the
**reachable graph**:

- **Roots.** It is explored from the machine's starts, every Scenario start and every Exploration
  candidate.
- **On demand.** Rows are evaluated as exploration reaches them, within a reach bound declared per
  machine.
- **Finite inputs.** Inputs stay finite, through their type or through Quint-style `oneOf`.
- **Every check reads this graph.** That covers search, refinement, rule disjointness, lint, Case
  lowering, conformance and identity.
- **Eager full tables survive only in tests,** as a parity oracle during migration, and never as
  semantics.

State then holds plain `Map`, `Set`, `List` and `Int`, and exploration is done by Umpire's checker,
with TLC or Apalache through the export.

**Why this is mostly a change of plumbing.**

- *The search is already lazy over states.* It only calls `Table.RowsFrom(state)`
  (`tools/umpire/internal/engine/search.go:203-246`).
- *Compositions are already reachable-only tables,* built all or nothing under a ceiling
  (`engine/compose.go:308-345`).
- *The catalog enters elsewhere.* It is in what surrounds the search:
  - the eager build of every state × class (`tools/umpire/interp/machine.go:529-568`);
  - refinement over unreachable rows (`engine/refine.go:131-163`), which forces Voting's `inv`
    guard;
  - fingerprints over the full table (`engine/canonical.go:167-194`);
  - Scala's catalog walks (`model/framework/dsl/Machine.scala:332-347`;
    `model/framework/dsl/StepTables.scala:135-152`).

**Consequences of the basis.**

Every decision is **settled (owner, 2026-10-13)**; §10.1 has the table.

- **D1.** Open graphs are allowed per Query, with explicit acceptance:
  - an `open-graph` lint fails the gate until it is accepted with a reason;
  - a `verify` in a Temporal feature requires a closed graph unless it is marked
    `.open(because = ...)`.
- **D4.** Go only. No Scala catalog walk and no Scala explorer remain. The Scala overlap
  check, `Refinement.unclosed`, the machine law tests' catalog walks and the Scala step tables go.
- **D8.** Query totals retire everywhere. Explored counts replace them.
- **D2.** The Behavior Fingerprint is an **IR structural hash only**, over the machine's own
  declarations. It is independent of exploration, the reach bound and closure (§4.13, §5.5).
- **D3.** Backends may decide: any Query may be answered by Umpire, Apalache or TLC (§7.3).
- **D5–D7.** Native partial `Map` only; `Inbox` kept for now; structured key lookups; checked
  `int64`; a reach bound per machine with a feature default (§6, §3.3).

**What replaces the catalog-wide guarantees.**

- §1 weighs what full tables buy.
- §5 adds Go static checks on the IR that need no enumeration:
  - a structural rule-overlap check;
  - exhaustive-match and refinement-map coverage;
  - unbounded-growth warnings;
  - an optional refinement invariant;
  - a structural hash;
  - root reporting.
- What is genuinely lost: latent defects in unreachable territory, function-level refactor
  equivalence, sizing before anything runs, and a fingerprint that ignores meaning-preserving
  rewrites.

**What it costs and saves.**

- **One churn.** Every Behavior Fingerprint, and through it every checked-in Case, changes once.
  After that, meaning-preserving rewrites of behavior-affecting IR churn their machine's Cases.
- **Later sizing.** State explosion is found by exploration, not counted up front.
- **Governance (GOV-02).** Three texts need amendment:
  - the SPEC's "finite types" (`.plans/UMPIRE4_SPEC.md:42`);
  - the SPEC's Behavior Fingerprint definition (§4.13);
  - VISION's "never decide" (`.plans/UMPIRE4_VISION.md:101-102`; draft in §7.5).
- **Measured share of today's evaluations** (§12). The reachable graph is 0.24–6.3% of today's
  evaluations for the four large samples. The samples' Go check would fall from ≈117 s to ≈5–6 s
  (*estimate*).
- **The rest of the feedback loop.** Most of the 19-minute change-scoped check is Scala toolchain
  overhead. §12.6 lists fixes for it that do not depend on this design.

**Plan.** Six phases (§10). Phase 1 makes the reachable graph the only table. A parity test against
the old eager tables proves that answers are unchanged and that Case bytes are unchanged under the
fingerprint-stripping projection, on every current model. Then:

2. Scala retirement and Go static checks.
3. IR collections.
4. DSL and lifter.
5. Native samples.
6. Backends that decide.

**Reading note.** The task cites `samples-vs-quint.md` finding #6 for refinement over unreachable
rows. In the file it is finding **#4** (`model/design/samples-vs-quint.md:221-276`). Finding #6, "`UpTo`
starts at zero", is resolved by unbounded `Int` (§6.4).

---

## 1. What full tables buy, and why we drop them

Today's table is every catalog state × every class (`interp/machine.go:529-568`), and SEMANTICS says
"Reachability, stuck states, Definition IDs and the Behavior Fingerprint are computed from this
table" (`model/framework/SEMANTICS.md:443-444`). For each benefit: does it matter, and what replaces it.

### 1.1 Catalog-wide checks on unreachable states

**Rule overlap.**

- **Today.** The framework refuses two rules of one class that both hold in *any* catalog state
  (`Machine.scala:332-347`, run at `Syntax.scala:740-746, 993`). Rule order therefore changes no
  row anywhere (SEMANTICS §6, `model/framework/SEMANTICS.md:365-369`).
- **Does it matter?** Only on reachable rows, since a row no start reaches is no behavior. One
  benefit is real: a latent overlap is caught before a later change makes its state reachable.
- **Replaced by** §5.1's structural check, which covers phase-, role- and class-headed rules over
  *all* phases with no enumeration. Rules with a `where` guard fall back to a dynamic check at
  reached states, which reports a path (§4.3).

**Refinement.**

- **Today.** Every row of the table must have a carrier or be a stutter (`refine.go:131-163`). A
  rejected unreachable row has no witness (`witnessOf` returns nil, `refine.go:200-208`).
- **Does it matter?** No. It is the cause of Voting's 30-line invariant guard
  (`model/framework/samples/Voting.scala:125-129, 171, 182`; samples-vs-quint finding #4). TLA+'s
  refinement proof itself assumes the invariant: `Inv /\ [Next] => [C!Next]`
  (`Voting.qnt.test`, end).
- **Replaced by** refinement over reachable rows (§4.4), §5.2's map-coverage check, and §5.4's
  optional declared invariant.

**Role closedness.**

- **Today.** `Refinement.unclosed` maps every refining state and compares `Closed` roles
  (`model/framework/dsl/Refine.scala:21-29`). It runs from model tests only, because the IR carries
  no roles (SEMANTICS §13, line 743-744).
- **Does it matter?** Yes, as a property of the maps.
- **Replaced by** roles carried in the IR and §5.2's per-phase check, which is static for
  phase-to-phase maps, with a dynamic check at reached states otherwise.

### 1.2 Scenarios starting anywhere

- **Today.** A Scenario may start at any catalog state (`engine/search.go:56-60` only requires the
  start to be in `t.States`). Its region was pre-built as part of the full table.
- **Does it matter?** Yes. It is how a Model asks about a state its `init` does not lead to.
- **Replaced by** roots (§3.3, §5.6). Every Scenario and Exploration start is admitted and reported
  as a root, and exploration covers its region. Nothing is lost.

### 1.3 Whole-function refactor equivalence

- **Today.** Step tables run every effect on every state and class (`StepTables.scala:138-152`).
  Sealing compares them (`model/framework/check/equivalence/Equivalence.scala:165-355`; README
  "Sealing a batch", `model/README.md:331-363`), so a refactor is shown equal *as a function* over
  the catalog. It works while the IR is stale (`STEPS_ONLY`), because Scala runs the Models.
- **Does it matter?** Partly. Equivalence of *reachable behavior* is what a refactor must keep, and
  shared helpers are covered by each machine that reaches them. Two things are genuinely lost:
  - equivalence on unreachable inputs;
  - the ability to seal while the IR is stale.
- **Replaced by** Go-side reachable tables compared after a fresh lift (§4.11). The change-scoped
  check already compiles and lifts before its equivalence stage (`model/framework/check/ChangeCheck.scala:219-246`).

### 1.4 Sizing before anything runs

- **Today.** `Preflight` refuses a machine before any evaluation when states or states × classes
  pass a ceiling (`interp/machine.go:380-393`; `interp/limits.go:21`). Totals give a reviewer the
  size of a question (`tools/umpire/ir/totals.go:13-27`; README "Counting a Query's total",
  `model/README.md:1965-2007`). The README says of totals that "nothing else reads it".
- **Does it matter?** Early refusal matters. The total as a figure matters little.
- **Replaced by:**
  - a reach bound that counts states and evaluations before doing them (§3.3);
  - §5.3's static growth warnings;
  - the dynamic growth report on open results (§5.3);
  - explored counts in every receipt (§4.6).
- **Lost:** a size known before running.

### 1.5 The simple fingerprint

- **Today.** The Behavior Fingerprint hashes the full table: every catalog state, class, outcome
  and fact, and every result of every row (`canonical.go:167-194`). It depends on nothing but the
  machine's declaration.
- **Does it matter?** Yes. Cases embed it (`lower/internal/producer/producer.go:426-444`), and pinned
  Cases go stale by it (`lower/generated.go:718-740`).
- **Replaced by** an IR structural hash of the machine's own declarations (D2; §4.13, §5.5). It is
  as independent of exploration as today's.
- **Lost:** stability under meaning-preserving rewrites of behavior-affecting IR, which today churn
  nothing when every row stays equal.
- **Gained:** edits to rows no root reaches stop churning Cases.

### 1.6 The `unreachable-value` lint

- **Today.** `lint/kinds.go:408-500` reports leaf values (enum cases, Booleans, integer ranges) that
  no reachable state holds. It already reads `Table.Reachable`. Its leaf domains come from *field*
  types, not from the state catalog.
- **Does it matter?** Yes. It survives unchanged for finite leaves.
- **Lost:** nothing for finite leaves. `Int` and collection leaves are combinations and are not
  reported, as channels already are not (`kinds.go:443-446`).

### 1.7 Holes at unreachable states

- **Today:**
  - A hole row at an unreachable state is recorded (`interp/machine.go:536-540`), but checks read
    `ReachableHoles` (`machine.go:197-206`).
  - A hole *outside* a step at **any** state leaves the machine without a table (SEMANTICS §12,
    line 714-716). This covers `ends`, which is evaluated on every state (`machine.go:689-703`),
    and the refinement map, which `fieldValues` calls on every state (`check/claims.go:208-234`).
- **Does it matter?** The second is a defect of the full table. A partial `end` or `toProduct` in
  unreachable territory disables every check of the machine.
- **After the change:** holes are found where exploration reaches them, with a path. §5.2's
  exhaustiveness check reports missing enum cases statically, wherever they are.

### 1.8 What is genuinely lost, and what full tables cost today

**Lost:**

1. **Latent defects in unreachable territory:**
   - `where`-guard overlaps;
   - refinement mismatches whose map is not a pure function of the phase;
   - value-dependent holes.

   These surface only when a change makes their state reachable, and then with a path.
2. **Function-level refactor equivalence,** and sealing while the IR is stale.
3. **Sizing before running,** and totals as a review figure.
4. **A fingerprint that ignores meaning-preserving rewrites** (D2; §4.13).
5. **Author-time refusals in Scala.** The overlap refusal moves from rule construction to the Go
   gate, seconds later.

**What full tables cost today:**

- **Go check time** tracks catalog × classes, not reachability. The samples' Go check takes ≈117 s,
  and Bank and Voting alone take ≈100 s. Their reachable graphs are 6.3% and 4.1% of the
  evaluations (§12.1).
- **The interpretation runs twice.** `check.Check` interprets the full tables a second time to
  replay witnesses (`tools/umpire/check/checking.go:370-375`). GC used 58% of CPU in a profile of
  Bank's check (§12.2).
- **Modelling distortions:**
  - Voting's `inv` guard leaks into its negative control (finding #4).
  - Collections are hand-encoded as records and ladders (findings 1, 20).
  - `UpTo` offsets stand in for signed integers (finding 6).
  - A fourth TwoPhaseCommit resource manager is refused at 786,432 states
    (`model/design/bounded-collections.md:503-511`).
- **Ceilings that refuse whole models.** 65,536 states and 1,048,576 evaluations
  (`interp/limits.go:21`) rule out partial maps entirely: Bank would need 83,521 states and
  TwoPhaseCommit 96,000 (`bounded-collections.md:142-145`).

---

## 2. Inventory: what each full-table dependency becomes

Every entry is *replaced*. Nothing keeps a catalog path beside the reachable one. "Finite domain"
means inputs, outcomes, facts and enum cases, which stay finite.

### 2.1 Scala (`model/framework`)

| Where | Today | Becomes |
| --- | --- | --- |
| `dsl/Domain.scala:14-20, 53-66, 80-83` | `Finite[T].values`, a full product with no ceiling. Required of every state type. | `Finite` remains only for input, outcome, fact and phase types. State types need `Data[T]` (§6.5). |
| `dsl/Domain.scala:23` | `CanEqual` exists only where `Finite` does. | `CanEqual` comes from `Data`. |
| `dsl/Machine.scala:400-404, 545, 579` | `Machine` requires `Finite[S]`, `Finite[O]`, `Finite[F]`. | `Data[S]`, `Finite[O]`, `Finite[F]`. |
| `dsl/Machine.scala:332-347`; `Syntax.scala:740-746, 993` | The overlap check walks every catalog state. | **Removed.** Go's §5.1 and §4.3 replace it. |
| `dsl/Refine.scala:21-29` | `unclosed` walks every state. | **Removed.** Roles in the IR, and §5.2. |
| `dsl/MachineLaws.test.scala:46-47, 57-58, 70-79` | Law tests over every `fs` value: refinement totality, and steps that stay in catalogs. | **Removed.** Go checks the same at reached states. |
| `dsl/StepTables.scala:131-223, 288` | Full step tables, and composition `ends` over member products (`endLimit` 1,000,000 at line 33). | **Removed.** Go reachable tables replace them (§4.11). |
| `dsl/Channel.scala:33-50, 99-108` | A contents catalog. An unordered `send` inserts by catalog index. | Bound by `capacity`. Canonical order is type order (§4.9). |
| `dsl/Monitor.scala:19, 33-46`; `dsl/Gaps.scala:13-64` | `Finite` as a typing constraint only. | `Data` / none. |
| `dsl/Claims.scala:106-152` | `Query.total`. | **Removed** (D8). |
| `dsl/Action.scala:43, 79-97, 199-209`; `Machine.scala:317-320` | Inputs need `Finite`; classes are their product. | Unchanged, plus `oneOf` (§3.2). Classes stay finite. |
| `check/equivalence/{Equivalence,StepLedger,InterpreterTables}.scala` | Compare full step tables; pin catalog sizes and hashes. | Compare Go reachable dumps; pin reached counts (§4.11). |
| `lint/Conventions.scala:181, 318-329, 505-510` | `R4.enum-wildcard` covers enums with `Finite` givens. | Covers every Model enum. §5.2 builds on it. |

### 2.2 The lifter (`model/framework/export`)

| Where | Today | Becomes |
| --- | --- | --- |
| `Types.scala:40-84, 104-142, 144-169` | Refuses `Int` fields, list fields, and non-enum, non-record types ("no finite catalog"). | Admits `Int`, `Map`, `Set`, `List` (§6.5). |
| `Types.scala:187-202` | Reads `given Finite` blocks for channel wiring. | Channel recognized by type. |
| `Declarations.scala:1061-1078, 1191-1202` | `valuesOf` for input catalogs; per-class lowering. | Unchanged: inputs stay finite. |
| `Declarations.scala:1127-1187` | Rule headings lowered into one `if` chain. The rules themselves are lost to the IR. | Also emits `RuleGuard` per rule (§5.1). |
| `Declarations.scala:1754-1769, 1862-1863` | Refuses `Int`/`List` channel messages and monitor states. | Admits them. |
| `Capabilities.scala:1246-1308`; `Lift.scala:153-161` | Counts static totals. | **Removed** (D8). |
| `inspect/LiftableSubset.scala:6-8, 47-77` | No dynamic collection operations. | §6.5's rows. |

### 2.3 Go (`tools/umpire`)

| Where | Today | Becomes |
| --- | --- | --- |
| `ir/validate.go:433-441, 906-915` | Refuses `int`/`list` state fields, monitor states and messages. | Admits them (§6.2). |
| `ir/validate_keys.go:103-136` | `Preflight` plus a full state catalog listing. | Class keys only. |
| `ir/totals.go` | Static totals. | **Removed** (D8). |
| `interp/machine.go:208-252, 325-376, 529-568` | Eager table: every state × class. | On-demand rows + `Explore` (§4.1). The eager build moves to a test oracle. |
| `interp/machine.go:380-393`; `limits.go:98-188` | `Preflight` and closed-form `Size`. | **Removed.** The reach bound replaces them. |
| `interp/machine.go:572-591` | A result must lie in the catalog. | `Conforms`. |
| `interp/machine.go:664-705` | `ends` is evaluated on every state. | At reached states. |
| `interp/machine.go:61-72, 476-522` | Catalog-wide key distinctness. | At reached values (§4.2). |
| `engine/table.go:43-80, 249-291, 352-442` | Full `States`/`Rows`; reachability by sweeping all rows. | `States`/`Rows` are the explored graph. |
| `engine/search.go:56-71` | A Scenario start must be in `t.States`. | A root (§5.6). |
| `engine/refine.go:131-163, 212-269` | Every row. | Explored rows, with the product explored from mapped states (§4.4). |
| `engine/progress.go:97-131, 184-197` | Reads eager rows; limits from `Scope.Progress`. | The provider; limits from the reach bound (§4.8). |
| `engine/compose.go:272-291, 312-345` | Moves from eager member tables. | Member providers (§4.10). |
| `check/claims.go:208-234, 464-479, 554-608` | `fieldValues` maps every state; monitor states must be in their catalog. | Reached states; `Conforms`. |
| `check/checking.go:370-375` | A second full interpretation for replay. | Replay reads witness rows on demand (§12.2). |
| `lint/kinds.go:408-525`, `holes.go`, `bindings.go` | Mixed full and reachable. | Explored graph only (§4.5). |
| `engine/table.go:207-241`; `canonical.go:78-336`; `producer/correlated.go:118-195` | IDs and fingerprints over full catalogs. | Reached members for IDs; IR structural hash v2 for fingerprints (§4.13). |
| `engine/lower.go:43-86, 108-164` | `Lower` over all rows; "fixed" against every state. | Explored rows and reached states (§4.12). |
| `conformance/plan.go:116, 186-328` | Indexes every state and row. | On demand along the Run (§4.12). |
| `export/agreement.go:189-232`; `quint.go:827-835` | Already reachable. | Unchanged. |

---

## 3. Target semantics

### 3.1 What stays finite

| Stays finite | Declared by | Why |
| --- | --- | --- |
| **Action input domains** | An input's type when it has a catalog (enum, Boolean, `Option`, record of these, `UpTo[N]`), or `oneOf(...)`. | A state's successors must be finite to enumerate. This is Quint's `nondet x = oneOf(S)` (`model/framework/samples/Voting.qnt.test`, `Next`). |
| **Starts** | `init`, or a finite `starts(oneOf(...))` (samples-vs-quint finding 7, `samples-vs-quint.md:365-394`). | Roots. |
| **Scenario starts, pinned schedules, Exploration candidates** | As today. | Roots (§3.3). |
| **Outcomes, facts, phases and roles** | Enums. | Refinement compares outcome catalogs (`refine.go:212-218`), lowering alters outcomes (`lower.go:121-131`), evidence covers every fact (`machine.go:710-728`), and `when[Role]` reads phase cases (`Syntax.scala:612-627`). |
| **Limits and the reach bound** | `temporal/Bounds.scala` (SEMANTICS §9, line 599-602) and `Machine.reach` (§3.3). | Every result names its bounds. |

**May be unbounded:** state fields, monitor states and channel messages, holding `Map[K, V]`,
`Set[A]`, `List[A]` or `Int`, and records, enums and `Option`s of these.

**Quantifiers:** over a *type*, enums only (`K.values.forall`); over a *value*, its finite contents.

### 3.2 Input domains, Quint-style

```scala
val a = input[Acceptor]("a")                                       // a finite type: unchanged
val b = input[Int]("b", oneOf(0 to 1))                             // Quint's `nondet b = oneOf(0.to(1))`
val rms = input[Set[RM]]("rms", oneOf(Set(RM.rm1), Set(RM.rm1, RM.rm2)))
```

- **The domain is per input.** `oneOf(xs*)` builds a `Finite[A]` for that input only, so
  `classesOf` (`Machine.scala:317-320`) and the lifter's `valuesOf` (`Declarations.scala:1061-1078`)
  are unchanged.
- **IR.** An explicit domain is carried on the input `Field` (§6.1).
- **Refusal.** An `Int` input with no `oneOf` is refused at its declaration, as an `Int` field is
  today (`model/framework/export/Types.scala:61-66`).

### 3.3 The reachable graph

- **Roots:**
  - the machine's `starts` (`proto/internal/temporal/server/api/umpire/v1/machine.proto:61`);
  - every Scenario `start` (`claim.proto:52`);
  - every Exploration candidate start (`tools/umpire/ir/exploration.go`).

  Each root is admitted by `Conforms` (`interp/eval.go:425-461`) and reported (§5.6).
- **Exploration** is breadth-first from the roots, in class order then result order. That is the
  order the search already uses (`search.go:8-12, 203-246`), so witnesses do not change.
- **The reach bound** is a `Limits` value (steps and a state ceiling), declared once per machine
  (`val reach = Bounds.wide`), with a feature-wide default in `temporal/Bounds.scala`. It is carried
  as `Machine.reach` (§6.1). It replaces the Go-defaulted `interp.Ceilings` (`limits.go:21`),
  `ComposeCeiling` and `Scope.Progress` (`check/checking.go:27-37`), which PLN-01 wants declared
  (`.plans/UMPIRE4_SPEC.md`, "Search and Limit rules"). Per machine, with a feature-wide default in `temporal/Bounds.scala` (D7, settled).
- **Closed** means exploration reached its fixpoint within the bound. Every reachable state and every
  row from it is known. This is the exact graph today's `Table.Reachable` computes
  (`engine/table.go:266-291`), minus the unreachable rest.
- **Open** means the bound stopped exploration first. Every row of every state within depth *d* is
  known.
- **Regions, for closure only.** The machine's graph is the region of its own starts. Each Scenario
  and Exploration root adds a region of its own. Closure is reported per region, so a Query is
  closed exactly when its Scenario's region is. Regions play no part in identity (D2, §4.13).
- **Policy (D1, settled).** An open region is allowed per Query:
  - the `open-graph` lint fails the gate until it is accepted with a reason in
    `generated/ir.lint.json`;
  - a `verify` in a Temporal feature requires a closed region unless the Query is marked
    `.open(because = ...)`.

### 3.4 What a passing result means

| Checker | Scope | A pass means | A pass does not mean |
| --- | --- | --- | --- |
| Umpire, closed region | every state the roots reach | the claim holds on every path of the Scenario within the Query's Limits; refinement, disjointness and stuck states hold on **every reachable row** | anything about states no root reaches |
| Umpire, open region | states within the reach depth *d* | the same, within depth *d*; reported `verified-within-limits` with `open` | that the claim holds beyond *d* |
| Apalache (`quint verify`) | symbolic: every run of at most *k* steps over every value the transition relation admits (`tools/umpire/export/verify.go:132-179`) | no violation within *k* steps | an unbounded proof, unless an inductive invariant is checked **(unverified for Quint 0.33.0)** |
| TLC (no exporter today) | explicit-state: the whole reachable graph when it terminates | the claim holds on the reachable graph | termination on an infinite graph |

### 3.5 Wording in receipts and in the README

The receipt (`check.Receipt`, `tools/umpire/check/checking.go:142-192`) gains three fields:

- `Reach` (the bound's name);
- `Reached` (states and rows explored);
- `Closed`;
- `Checker` (which checker answered, with its version and scope; §7.3).

`TableRows` (line 176-177) becomes the explored row count.

```
verified-within-limits: consistencyHolds holds on every path of anyRun within `transaction`
  (10 steps); the reachable graph of TwoPhaseCommit.Model closed at 288 states under `wide`.  [288 measured, §12]
verified-within-limits (open, accepted): ... the region of anyRun is open at depth 12 under `wide`
  (65,536 states); paths longer than 12 steps were not explored.
refinement Voting.Model refines Consensus holds on every reachable row (closed, 599 states under `wide`).
                                                                    [599 measured on today's encoding, §12]
```

README text, replacing "Counting a Query's total" (`model/README.md:1965-2007`) and SEMANTICS §7's
line 443:

> A machine's table is the states its starts, its Scenarios' starts and its Explorations' candidates
> reach, explored breadth-first within its reach bound, and the rows from them. A table is *closed*
> when exploration reached every state it can; every check of a closed table is exhaustive over
> reachable behavior and says nothing of states no root reaches. A table that is not closed is
> *open*: every result read from it names the depth at which it stopped and proves nothing of longer
> paths, and the gate refuses it until it is accepted with a reason. A Query's size is what its
> answer reports, the states and rows it explored, never a count taken before anything runs.

PLN-03 ("Exhaustive means complete") holds: a closed graph is exhaustive, and an open one never
claims to be.

---

## 4. The reachable graph as the only table

### 4.1 On-demand rows and exploration

**Rows.**

- `interp/successors.go` (new) exposes `(*Machine).RowsFrom(state Value) ([]Row, []HoleRow, error)`.
  It evaluates every class at one state with the existing per-pair `in.steps`
  (`interp/machine.go:593-606`), memoized by `state.Key()`.
- Per-pair evaluation already exists: `Why` uses it (`interp/decisions.go:59-83`).
- `result`'s catalog test (`machine.go:586-588`) becomes `Conforms` plus the outcome and fact checks
  it already makes (`machine.go:574-583`).

**Exploration.**

- `Explore(roots, reach)` generalizes compose's `explore` (`engine/compose.go:312-345`). It counts
  states, evaluations and results before making them, and returns `Closed`, or the depth it stopped
  at.

**The table.**

- `engine.Table` is built from the explored graph. `States`, `Rows` and `Reachable` are the explored
  set, sorted by key, as compositions sort (`compose.go:349-360`).
- `RowsFrom`, `pair` and `steps` (`table.go:118-170`) read the memo.
- The `behaviors` cache keyed by declaration (`machine.go:325-334`) becomes a cache of row memos, so
  derived machines declared alike keep sharing evaluations.

**The eager build.**

- `rows` (`machine.go:529-568`) moves to a test helper, `interp/eager_test.go`, the parity oracle of
  Phase 1, and is deleted when migration ends.

### 4.2 Keys

- **Distinctness.** `distinctKeys`, `RowKeys` and `distinctRowKeys` (`machine.go:61-72, 476-522`)
  check reached values, with compose's collision check as the pattern (`compose.go:293-306`).
- **Spelling.** It is injective within a type by construction (§6.3).
- **`checkRows`.** A result outside `States` (`table.go:414-442`) is a newly reached state, not an
  error.

### 4.3 Rule disjointness, dynamic part

- **Static first.** §5.1 decides statically every pair of rules headed by phases, roles or classes.
- **Dynamic fallback.** For the remaining pairs (at least one `where` guard, with phase and class
  sets that intersect), `RowsFrom` evaluates both guards at each reached state and class in that
  intersection.
- **Refusal.** Two that hold give `declaration-error: overlapping-rules`, with a shortest path from
  a root (the `pathTo` that `witnessOf` uses, `refine.go:200-208`).

### 4.4 Refinement

- **Source rows.** `RefineTables` (`refine.go:131-163`) loops over explored source rows. A rejected
  row always has a witness.
- **Product rows.** `carrierOf` (`refine.go:248-269`) reads `dst.RowsFrom(mapped state)`, so the
  product is explored from mapped states, not from its catalog.
- **Field atoms.** `fieldValues` (`check/claims.go:208-234`) maps reached states only.
- **Closedness and coverage.** These are §5.2's.
- **Open regions.** On an open region, refinement holds "on rows within depth *d*", and a Query read
  `through` it (SEMANTICS §9, line 609-612) inherits `open`.

### 4.5 Lint

| Lint | Scope |
| --- | --- |
| `never-enabled`, `silent-rejection`, `stuck-state`, untaken choices, unreachable bindings | Explored rows. On an open region, they are reported "within depth *d*". `stuck-state` is not reported at the frontier. |
| `unreachable-value` (`kinds.go:408-500`) | Finite leaves only: enum, Boolean, `UpTo`. |
| Holes (`holes.go`) | Explored states only. `monitorReads` (`369-385`) iterates *reached* monitor states. |
| **new** `open-graph` | One per open region. It fails the gate until accepted (D1). |
| **new** `non-exhaustive-match`, `unmapped-phase`, `unbounded-growth`, `detached-root` | §5.2, §5.3, §5.6. |

### 4.6 Totals retire (D8)

- **Identity unaffected.** Totals are excluded from every identity (`ir/totals.go:83-91`;
  `claim.proto:86-92`), so retiring them changes no fingerprint, answer or Case.
- **Removed:**
  - the lifter's counter (`Capabilities.scala:1246-1308`, `Lift.scala:153-161`);
  - admission's check (`totals.go:95-155`);
  - `RequireTotals` (`totals.go:43-58`), and its caller in exploration (`explore/explore.go:117`);
  - `Query.total` (`claim.proto:86-92`), marked reserved.
- **A written `total` is refused** at its line, with a pointer to the receipt's counts.
  ReliableBroadcast's `total 4074840` (`model/framework/samples/ReliableBroadcast.scala:245-256`)
  goes.
- **The receipt's counts take over.** `Explored` and `Expanded` (already present) and the new
  `Reached` say how large a question was.

### 4.7 Monitors

- **Admission.** Monitor states may be unbounded: `catalog` admission (`validate.go:906-915`)
  admits them, and `watch` (`check/claims.go:554-608`) checks `Conforms`.
- **Product graph.** An unbounded monitor makes the *product* graph unbounded even when the machine
  graph is closed. `Limits.search` bounds it, as it bounds any search, and `limit-reached` already
  "proves nothing" (SEMANTICS §20, line 1248).
- **TwoLayeredCache** can then keep Quint's history list (samples-vs-quint finding 17).

### 4.8 Progress

- **Rows.** Progress already explores from the starts and reads `t.RowsFrom`
  (`engine/progress.go:97-131`).
- **Limits.** They come from the reach bound, not from `Scope.Progress`. Today the IR carries only
  `within` (`claim.proto:204-215`).
- **Fair classes** stay a finite class list (`progress.go:184-197`).
- **Fair cycles** need a closed region. On an open one, progress is `unresolved` unless a violation
  is found first (SEMANTICS §11, line 667-668).

### 4.9 Channels

- **Bound.** `Inbox` keeps `capacity` as its bound. The contents catalog (`Channel.scala:41-50`;
  `interp/channels.go:42-72`) is no longer listed.
- **Order.** The unordered canonical order becomes type order (§6.3). That is the same order for
  finite messages (`Channel.scala:99-108`; `channels.go:74-94`).
- **Wiring.** The state's hand-written `given Finite[S]` block (SEMANTICS §18, line 1014-1019;
  samples-vs-quint finding 14) goes, because the channel field is recognized by type.
- **Later (D5b, settled).** `Inbox` stays for now, and channels are re-expressed as a library over
  `List`/`Set` later. A backend may decide a channel model only once channels are exported or moved
  to that library (§7.3).

### 4.10 Compositions

- **Moves.** `movesFrom` (`compose.go:272-291`) calls member providers in place of eager `pair()`.
- **Unchanged.** Composed exploration is already reachable, and `checkStartProduct`
  (`composekeys.go:161-175`) stays.
- **Removed.** The Scala `StepTables.ends` product (`StepTables.scala:187-223`) goes with step tables.

### 4.11 Equivalence, sealing and table pins, in Go

- **Step tables.** The Scala step tables are removed (D4). Go dumps each explored graph (starts,
  reached states with end and refinement image, and the rows from them, with a `closed` / `open at
  d` header) from the same interpretation the check uses.
- **Comparison.** `Equivalence.compareTable` keeps comparing lines under a rename map
  (`Equivalence.scala:274-355`). `StepLedger.reorder` (`StepLedger.scala:370-410`) groups by action
  within a state block **(unverified that this is its only assumption)**.
- **`STEPS_ONLY` sealing** while the IR is stale retires.
  - `make umpire-check-equivalence` always lifts first.
  - The change-scoped check's equivalence stage (`ChangeCheck.scala:230-246`) runs after its own lift
    (`219-229`).
  - The `activity_mini` baseline (`model/temporal/check/Coverage.scala:19-22`) is re-captured once.
- **Table pins** (`InterpreterTables.scala:68-99`) pin reached counts and a hash of the sorted reached
  keys. The `missing-pair` mutant (`InterpreterMutants.scala:10-19`) stays meaningful at reached
  states.

### 4.12 Case lowering and conformance

**Lowering.**

- `Lower` (`engine/lower.go:43-86`) reads explored rows.
- `fixedRequirements` (`lower.go:108-164`) decides "fixed" against reached states of the Query's
  region and the finite outcome catalog.

This is a semantic change. A predicate that accepted some *unreachable* state is not "fixed" today,
and it may become fixed, which adds a state clause to a Contract. Phase 1's parity test shows
whether any checked-in Case changes (open question 5).

The rest of a Case already reads only reachable rows:

- evidence alternatives (`producer.go:658-731`);
- projections, which do a BFS from the witness (`correlated.go:197-236`);
- Contract states and local names.

**Conformance.**

- `conformance/plan.go:116, 186-328` explains a Run by calling `RowsFrom` at the states its
  explanation reaches, memoized and charged to `MaxReadings`
  (`tools/umpire/conformance/conformance.go:56-74`).

### 4.13 Identity (D2, settled: IR structural hash)

**Definition IDs** are names:

- Scala: `definitionId = qualifiedName` (`model/framework/export/Context.scala:350-352`).
- Go: `Family.ID(kind, owner, member)` (`engine/table.go:11-21`).

Only `IDs()` and `meanings()` (`table.go:207-241`; `canonical.go:78-97`) change: they list reached
members. No authored ID changes, and neither do Case IDs (`IdentityFor`, `lower/generated.go:116-122`)
or file names.

**The Behavior Fingerprint is an IR structural hash (settled, owner, 2026-10-13).** It hashes the
behavior-affecting parts of the lifted IR, in canonical form (§5.5). It reads no table. So it does
not depend on:

- exploration;
- the reach bound;
- whether a graph is closed or open;
- which Scenarios exist.

Concretely:

- **Machine fingerprint (the target).** The machine's own declarations: its types, bound functions
  and their callees, its own `starts`, `ends`, evidence, refinement and the monitors and assumptions
  it names. This is "own starts only".
- **Scenario and Query fingerprints.** A Scenario's declaration and start go into the Scenario
  fingerprint (`ScenarioSemantic`, `canonical.go:205-243`). The Query fingerprint embeds the target,
  Property and Scenario fingerprints with the Limits (`QueryCanonicalOf`, `canonical.go:307-336`).
  `finiteCompleteness` keeps the finite action domain and the start.
- **Property fingerprint.** The Property's own declaration (its selector and predicate functions in
  canonical form), with the target fingerprint in place of today's `meanings()` header
  (`canonical.go:298-303`).
- **Projection fingerprints** (`lower/internal/producer/correlated.go:166-195`) follow the target.
- **Tag.** The encoding changes, so the tag becomes `umpire.behavior-fingerprint/v2`
  (`canonical.go:17-21`). Stale readers then reject v1 data, as ART-04 requires.

**The trade-off.** A meaning-preserving refactor of behavior-affecting IR churns that machine's
fingerprint, and through it its Cases. Examples: rewriting a guard into an equivalent one, or
inlining a helper. Today the same refactor churns nothing as long as every row stays equal. To keep
this to genuine edits, canonicalization (§5.5) must:

- sort declarations, so that source order is not structure;
- erase helper-function and local names, so that renaming or moving a helper is not structure;
- strip positions, origins, choice names, totals and `because` text.

What remains is churn on edits a reviewer would call behavioral in form, if not always in meaning.

**What it gains.**

- Edits to rows no root reaches no longer churn anything, as they do today.
- Adding a Scenario churns only that Scenario's Queries and Cases.
- The fingerprint can be computed at admission, before any exploration.

**Where the churn lands.** Phase 1. The full table that v1 hashes no longer exists after Phase 1, so
the switch to v2 happens there, once. Later IR additions do not churn existing models as long as the
hash reads an explicit allowlist of behavior-affecting fields. These additions are `RuleGuard`,
`Match.scrutinee_type`, `Case.roles` and `Refinement.invariant` (§5). A lifter change that alters
the IR of an existing model, rather than adding to it, does churn. Any such change must be reviewed
with the regenerated Cases.

**Needs.**

| Consumer | Needs | Met by the structural hash |
| --- | --- | --- |
| **Live and pinned Cases** (`.live`; `tools/umpire/cmd/umpire-gen-cases/main.go:28-45`; stale by `lower/generated.go:718-740`) | staleness on every behavior change; stability otherwise | **Staleness:** yes, never missed. **Stability:** across moves, renames, reorders and doc edits, but not across equivalent rewrites. |
| **Change-scoped check** (`ChangeCheck.scala:219-246, 358-367`) | "did behavior change?" | Go reachable dumps (§4.11) for meaning. The hash is the cheap first test: an unchanged hash means unchanged behavior. |
| **Batch sealing** (`Equivalence.scala:374-388`) | none: it strips fingerprints | — |
| **Assessment identities** (`conformance.go:141-172`) | IR hashes | Unaffected. The structural hash is the same device, at machine scope. |

**The SPEC wording change (proposed, GOV-02).** Today the SPEC defines the term as: "Behavior
Fingerprint. A value computed from the behavior-affecting parts of a Model Definition. It changes
with behavior, but not with documentation, source location, or source order"
(`.plans/UMPIRE4_SPEC.md`, Core concepts). Proposed:

> **Behavior Fingerprint.** A value computed from the behavior-affecting declarations of a Model
> Definition in canonical form. It changes whenever those declarations change, and not with
> documentation, source location, source order, or the names of helper functions. A change that
> leaves behavior equal may still change it; a change of behavior always does.

The "Front end" concept (`.plans/UMPIRE4_SPEC.md:39-41`) already requires every front end to
reproduce the same fingerprints. With a structural hash, that means every front end must lift
identical IR, up to canonicalization. That is a stronger requirement than identical tables, and the
amendment should say so.

---

## 5. Static checks on the Umpire IR in Go

These checks close part of what §1.8 loses. Each reads the IR without enumerating a state type.
Their only finite domains are enum cases and classes. They live where their inputs live:

- IR-only checks in `tools/umpire/ir`, as admission;
- findings in `tools/umpire/lint`, which may import "ir, interp and check only"
  (`.plans/UMPIRE_MODULES.md`, Lint row).

### 5.1 Structural rule-overlap check

**IR today.** A `StepBinding` names one lowered function (`machine.proto:73-80`). The lifter has each
rule's heading:

- `Heading.In(projection, phases)`, `InSet`, `Role`, `Always`, `When(g)` and `And(inner, g)`
  (`model/framework/export/Declarations.scala:1127-1140`);
- the class each rule fires (`cls`, `Declarations.scala:1188-1198`).

It folds them into an `if` chain (`chain`, `1180-1187`). So the IR cannot tell two rules apart.

**Mechanism.**

- `StepBinding` gains `repeated RuleGuard rules`, where `RuleGuard` is
  `{index, heading, position, projection, phases, classes, where}`:
  - `projection`: the phase projection function;
  - `phases`: case names, empty meaning any;
  - `classes`: input values, empty meaning every class;
  - `where`: a guard function over the state and inputs, or empty.
- `InSet` (a def over the phase) is evaluated on each case of the phase enum. `Role` lowers to its
  case list (`model/framework/export/Roles.scala:159-170`).
- For each pair of rules of one action, the check decides, in order:
  1. **Disjoint by class** when their `classes` do not intersect.
  2. **Disjoint by phase** when their `phases` do not intersect.
  3. **Overlapping, statically**, when neither has `where`. The refusal names both rules and a
     shared phase and class. This holds whether or not that phase is reachable: it is an authoring
     error in itself.
  4. **Otherwise left to the dynamic check** at reached states, with a path (§4.3).

**Touch points:**

- `machine.proto:73-80`;
- `Declarations.scala:1127-1198` (emit);
- new `tools/umpire/ir/rules.go` (static, run at admission after `identities`,
  `ir/validate.go:79-86`);
- `interp/successors.go` (dynamic);
- removes `Machine.scala:332-347` and `Syntax.scala:740-746, 993`.

**Catches:** every overlap between rules headed by phases, roles or classes, over all phases, with no
state enumerated. That is the common shape in the Temporal Models (SEMANTICS §6's headings table,
`model/framework/SEMANTICS.md:387-396`).

**Cannot catch:** overlaps between `where` guards at states no root reaches.

### 5.2 Exhaustive-match and refinement-map coverage

**Today.**

- A value no case matches is an undeclared hole at evaluation (`interp/eval.go:657-683`).
- `R4.enum-wildcard` refuses whole-case wildcards over Model enums
  (`model/framework/lint/Conventions.scala:318-329, 505-512`), so matches name their cases. The Scala
  compiler's exhaustivity warning then covers them. Whether the build fails on that warning is
  **(unverified)**: no `-Werror` appears in `model/project.scala`.
- With full tables, a missing case on any catalog state showed up as a hole row. Under reachability,
  it surfaces only where a root leads.

**Exhaustiveness.**

- `Match` (`expression.proto:132-135`) gains `TypeRef scrutinee_type`, which the lifter knows. No
  typer exists in Go: the only `TypeOf` is for realization operands, `realization/operand.go:52`.
- For an enum-typed scrutinee, the uncovered cases are the enum's cases (`value.proto:27-34`) minus
  those matched by an unguarded case pattern (`MatchCase.guard`, `expression.proto:137-140`). A
  wildcard or bind covers all. An arm that reaches a declared hole counts as covered, since it is a
  deliberate unknown.
- A finding `non-exhaustive-match` names the function, the position and the missing cases.
- Nested field patterns are treated conservatively: a case pattern with a non-wildcard subpattern
  covers nothing. That can raise false findings, which the author resolves by restructuring or by an
  accepted reason.

**Refinement-map coverage.**

- The exhaustiveness check runs over the refinement map (`machine.proto:84-95`) and every function
  it calls.
- In addition, if the map's phase component is a `Match` on the source phase whose arms construct
  product phase literals, it is partially evaluated per source phase case. The finding
  `unmapped-phase` reports a case whose image is a hole or not a literal.

**Role closedness.**

- Enum `Case` (`value.proto:31-34`) gains `repeated string roles`, metadata the lifter has from
  `extends` clauses (SEMANTICS §2, line 105-112). Roles stay metadata and change no table.
- For a partially evaluable map, closedness is checked per case statically. Otherwise it is checked
  at reached states, with a path. This replaces `Refinement.unclosed` (`Refine.scala:21-29`) and its
  model tests.

**Catches:** missing enum cases anywhere in a function, reachable or not; refinement maps that leave
a phase unmapped; closedness for phase-to-phase maps.

**Cannot catch:** incompleteness over `Int` or collections (literal patterns); guard-dependent
incompleteness; value-dependent maps, which are checked only where reached.

### 5.3 Unbounded-growth warnings

**Static analysis.**

- For each step function, walk the state it constructs or copies (`Construct` and `Copy` in
  `expression.proto:33-54`).
- Classify each write of a field `f`:
  - **grows**: `s.f + e` with `e` a positive literal or an input whose domain is positive; `incl`,
    `updated` or `union` on a set or map; `:+` or `++` on a list;
  - **shrinks**: subtraction, `excl`, `removed`, `drop`;
  - **resets**: a literal or a start value.
- A growing write is **bounded** when a dominating condition compares `s.f` (or its size) with `<`
  or `<=` against a literal, or an expression of finite-typed fields. A dominating condition is an
  enclosing `If` or the rule's guard, now available through `RuleGuard` (§5.1).
- A field with growing writes, no shrinking or resetting write, and some unbounded growing write
  gives an `unbounded-growth` warning.

The analysis is a heuristic, and does not see through helpers beyond one call:

- **False positives.** Bank's balances are bounded by conservation (`toBalance + amount > supply`,
  `model/framework/samples/Bank.scala:100`), which a syntactic check cannot see.
- **False negatives.** Growth carried through another field.

So it warns, and never fails the gate by itself. It lives in a new `tools/umpire/lint/growth.go`
over the IR.

**Dynamic growth report.** On every open result, the report gives the distinct values per field at
each depth, and flags the fields still growing at the frontier
(an illustrative line: `Kettle.Snapshot.reading: 1, 2, 3, … still growing at depth 60`). It replaces the factorized catalog
message that `bounded-collections.md:496-511` proposed.

### 5.4 Optional refinement-declared invariant

```scala
object refinement extends Refinement(Consensus):
  def toProduct(s: Votes): Decision = Decision(chosen = chosen(s))
  // VotesSafe and OneValuePerBallot, which the proof of Spec => C!Spec rests on
  def invariant(s: Votes): Boolean = votesSafe(s) && oneValuePerBallot(s)
```

- **IR.** `Refinement` (`machine.proto:84-95`) gains `string invariant = 5`.
- **Go.** The invariant is checked on every reached state of the source graph. A violation is
  `refinement-rejected` with the rule `invariant` and a path. The refinement rows are checked as
  §4.4 says.
- **It is a claim, not a guard.** No row changes. This replaces Voting's invariant-as-guard
  (`Voting.scala:125-129, 171, 182`).
- **Relation to finding #4.** It is option (b) of that finding (`samples-vs-quint.md:265-270`), but
  checked rather than assumed.
- **Value:**
  - it states the obligation where TLA+ states it (`<1>2. Inv /\ [Next]_<<votes, maxBal>> =>
    [C!Next]_chosen`, `Voting.qnt.test`);
  - it ties the invariant to the refinement in receipts;
  - it is the candidate an Apalache inductive check would use **(unverified)**.
- **Not required** for soundness on a closed graph.

### 5.5 IR structural hash

- **Content: the machine's behavior closure.**
  - Its `Machine` declaration.
  - Its bindings' functions and their transitive callees (`Call` expressions).
  - The types reachable from its state, outcome, fact and input types.
  - The bound actions.
  - Its starts, `ends`, evidence and refinement functions.
  - The monitors and assumptions it names.
- **Stripped** of positions, origins, choice names, totals and `because` text. `because` is the
  fourth field of every framework `Step` construct (`model/framework/dsl/Machine.scala:10`), and
  that field's expression is erased.
- **Canonicalized.** Types are sorted by name; their names stay, because case names are spelled in
  keys and Definition IDs. Functions are hashed as a Merkle tree: each body with its callees replaced
  by their hashes, and with parameter, `let` and helper names erased. So moving, renaming or
  reordering a helper changes nothing, while inlining one does.
- **Allowlist.** Only the fields listed here are read. IR metadata added later (`RuleGuard`,
  `Match.scrutinee_type`, `Case.roles`) therefore changes no existing fingerprint.
- **Device.** This is the device of the assessment Model identity, which hashes the whole Model
  without positions, totals, origins and choice names (`tools/umpire/conformance/conformance.go:158-172`;
  helpers `ir/totals.go:83-91`, `ir/origins.go:9`). It is restricted to one machine and made
  order-insensitive.
- **Touch point:** new `tools/umpire/ir/structure.go`, `ir.StructureOf(model, machine)`.
- **Uses:** the Behavior Fingerprint itself (D2, §4.13), computed at admission before any
  exploration.

### 5.6 Root reporting

- **Admission.** Every Scenario start (`claim.proto:52`) and every Exploration candidate start
  (`ir/exploration.go`) is admitted as a root by `Conforms`. This replaces "start is in `t.States`"
  (`search.go:56-60`) and "start is in the domain" (`interp/machine.go:674-676`).
- **Reporting.** `umpire-lint --tables` (`lint/render.go`) lists each machine's roots with:
  - their source (init, Scenario *x*, Exploration *y*);
  - whether the machine region reaches them;
  - their region's closure and size.
- **`detached-root`.** A root the machine region does not reach gets an informational finding: the
  Scenario asks about states the machine never enters on its own. Whether Kettle's `coldest`
  Scenario is one is **(unverified)**.
- **Receipts** name the root of the Query's region (§3.5).

---

## 6. Collections in the IR and the interpreter

### 6.1 Schema (additive, `proto/internal/temporal/server/api/umpire/v1/`)

```proto
// value.proto
message TypeRef {
  oneof ref {
    // named = 1 … channel = 6 unchanged; `int` and `list` become admissible state fields.
    TypeRef set = 7;     // a finite set of values of any admissible type
    MapOf   map = 8;     // a partial map, Quint's `K -> V`
  }
}
message MapOf { TypeRef key = 1; TypeRef value = 2; }

message Value {
  oneof kind {
    // bool = 1 … text = 6 unchanged
    SetValue set = 7;    // canonical: items in type order, no repeats
    MapValue map = 8;    // canonical: entries in key type order, no repeated key
  }
}
message SetValue { TypeRef element = 1; repeated Value items = 2; }
message MapValue { TypeRef key = 1; TypeRef value = 2; repeated Entry entries = 3; }
message Entry    { Value key = 1; Value value = 2; }

// expression.proto: one Expr kind with an op enum, as bounded-collections §4.1 proposes.
message Collection {
  enum Op {
    OP_UNSPECIFIED = 0;
    OP_SET_OF = 1; OP_MAP_OF = 2;                        // literals; Go canonicalizes
    OP_INCL = 3; OP_EXCL = 4; OP_UNION = 5; OP_INTERSECT = 6; OP_DIFF = 7; OP_SUBSET = 8;
    OP_SIZE = 9; OP_GET = 10; OP_GET_OR = 11; OP_UPDATED = 12; OP_REMOVED = 13;
    OP_KEYS = 14; OP_HAS_KEY = 15;
    OP_FORALL = 16; OP_EXISTS = 17; OP_FILTER = 18; OP_MAP = 19; OP_FOLD = 20;
    OP_RANGE = 21;                                     // integer `lo until hi`, a list
    OP_HEAD_OPTION = 22; OP_DROP = 23; OP_APPEND = 24;
  }
  Op op = 1; TypeRef type = 2; Expr receiver = 3; repeated Expr args = 4; Lambda body = 5;
  TypeRef over = 6;                                    // an enum quantified over, when no receiver
}

// machine.proto
message Machine { /* … */ Limits reach = 16; }          // §3.3
message StepBinding { /* … */ repeated RuleGuard rules = 4; }   // §5.1
message RuleGuard { int32 index = 1; string heading = 2; string function = 3; Position position = 4; }
// action.proto: an input's explicit domain where it is not its type's catalog (§3.2)
message Field { /* … */ repeated Value domain = N; }
```

Binary arithmetic gains `OP_MUL`, `OP_DIV` and `OP_MOD`, which Quint has. Today only `+` and `-` lift
(`model/framework/SEMANTICS.md:1198-1199`).

### 6.2 Admission (`tools/umpire/ir/validate.go`)

- **`stateField`** (`433-441`) admits `int`, `list`, `set` and `map`, recursively. It still refuses
  `channel` inside a collection and a type that contains itself (`1081-1142`), since a recursive
  *type* is still meaningless.
- **`catalog`** (`906-915`) admits the same for monitor states and channel messages.
- **Action inputs** must have a finite domain: a type with a catalog, or a non-empty `Field.domain`.
  An `int` input without one is refused at admission, not later in `Size`, which is the gap noted at
  `validate.go:213-216`.
- **`value`** refuses non-canonical `SetValue`/`MapValue` literals.
- **`identities`** (`validate_keys.go:103-136`) stops calling `Preflight` and stops listing the
  state catalog (§4.2). Class keys are still checked in full,
  because classes stay finite. Outcome and fact catalogs are still listed: they are enums.
- **Totals** (`totals.go:117-155`) are refused: totals retire (D8, §4.6).

### 6.3 Values, spelling, order and equality

**Go values.**

- `interp.Value` (`interp/eval.go:31-46`) gains `SetValue` and `MapValue` kinds.
- A map's `Items` are `Entry` records, the device the channel already uses for `Delivery`
  (`interp/channels.go:22-24`).
- `Equal` (`eval.go:103-120`) is structural and needs no change, *given canonical form*.

**Type order** (new, catalog-free) is the one total order that decides canonical form and table
sort order:

| Kind | Order |
| --- | --- |
| Boolean | false < true |
| Int | numeric |
| Text | bytewise |
| Enum | case declaration index, then fields lexicographically |
| Record | fields lexicographically |
| List | lexicographic by items, a prefix first |
| Set, Map | lexicographic over their canonical item or entry sequences |

On every type that has a catalog today, type order *is* catalog order:

- An enum's catalog is its cases in order, each with its fields varying last-fastest. That is
  lexicographic.
- A record's catalog is the product with the last field fastest. That is lexicographic.
- `Option` lists `None` first, as the first case.
- An IntRange is ascending.
- A Boolean is false, then true.

So SEMANTICS §1's ordering rules become one rule. An unordered channel's canonicalization
(`channels.go:74-94`) can use type order without changing any existing key. This is
**(unverified by test)**, and the Phase 1 parity test covers it.

**Spelling** (`Value.Key`, `eval.go:48-79`):

| Value | Spelling | Example |
| --- | --- | --- |
| Int | decimal, as today (`strconv.FormatInt`) | `3`, `-1` |
| Set | items in type order, in braces | `{a1,a3}`, `{}` |
| Map | `key:value` entries in key type order, in braces; absent keys are not spelled | `{a1:{},a2:{0-v1}}` |
| List | as today | `[a,b]` |

**Negative integers collide with the key separator** (D5c, settled: structured lookups). Keys are joined with `-`
(`eval.go:58-70`) and split on `-` in four places:

- `machine.go:731-734` (`ActionOf`);
- `compose.go:520`;
- `refine.go:277` (`sameNamedKey`);
- `canonical.go:196-202` (`SetupKey`).

A record `Ballots(maxBal = -1)` spells `…--1`. Integer ranges never start below zero today:
`UpTo[N]` is `0..N` (`Domain.scala:88-96`). The four splits are replaced by structured lookups:

- the first field's value for `SetupKey`;
- the action ID for `ActionOf`;
- the fact value for `sameNamedKey`;
- the member state for the composition split.

Each site has the structured value at hand. Integers keep their decimal spelling, `-1`, which also
matches ITF and Quint. `SetupKey` is
already flagged as fragile (`model/design/bounded-collections.md:1290-1292`, open question 8).

**Scala equality.** Under `-language:strictEquality`, `CanEqual` today exists only where `Finite`
does (`Domain.scala:23`). A new typeclass `Data[T]` (§6.5) supplies `CanEqual[T, T]` for every
admissible type, including `Map`, `Set`, `List` and `Int`. Scala 3's standard `CanEqual` givens for
sequences and sets may make part of this redundant **(unverified)**.

### 6.4 Integers

- `OP_ADD` and `OP_SUB` are unchecked `int64` today (`eval.go:636-639`). Overflow was impossible
  because every result had to land in a catalog.
- **Settled (D6): checked `int64`.** An overflow is an undeclared hole at its row, reported as
  `incomplete` with the row (SEMANTICS §12). It is not an error of the Model, and it is not silent
  wrap-around.
- Quint's `int` and Apalache are unbounded (Bank's original works with `2^256`,
  `model/framework/samples/Bank.qnt.test:1-5`). The ITF reader already accepts `#bigint`
  (`export/itf.go:358-371`). So Go and the backends differ past 2^63. Go reports `incomplete` there,
  and §7.4 treats that as incomparable, not as a disagreement.

### 6.5 Scala surface and lifting

**`Data[T]`** (new, `framework.dsl`) is derived for enums, case classes, Boolean, `Int`, `String`
(not admitted in state; a fact or because text only), `Option`, `Map`, `Set` and `List`. It carries:

- `CanEqual`;
- `compare`, the type order.

`Finite[T] extends Data[T]`. `Machine[S, O, F]` takes `Data[S]`, `Finite[O]` and `Finite[F]`.
Inputs take `Finite[A]` or `oneOf`.

**Liftable subset additions** (SEMANTICS §20 requires a row and a fixture each, line 1166-1169):

| Authored | Receiver | IR |
| --- | --- | --- |
| `Set(a, b)`, `Set.empty`, `Map(k -> v)`, `Map.empty` | – | `OP_SET_OF`, `OP_MAP_OF` |
| `s.contains(x)`, `s.incl(x)`, `s.excl(x)`, `union`, `intersect`, `diff`, `subsetOf`, `size`, `isEmpty` | Set | as named |
| `m(k)` | Map | `OP_GET`. An absent key is an undeclared hole at the row, with a path (D5a), as a `match` with no case is (SEMANTICS §12, line 710-711). Quint's `get` on an absent key is likewise an error. |
| `m.getOrElse(k, d)`, `m.get(k)`, `m.contains(k)`, `m.updated(k, v)`, `m.removed(k)`, `m.keySet` | Map | `OP_GET_OR`, `OP_GET` to `Option`, `OP_HAS_KEY`, `OP_UPDATED`, `OP_REMOVED`, `OP_KEYS` |
| `xs.forall(p)`, `exists`, `filter`, `map` | Set, List, `m.keySet`, `m.values` | quantifier ops over canonical order |
| `xs.foldLeft(z)((acc, x) => …)` | Set, List | `OP_FOLD` in canonical order. Scala folds a `HashSet` in hash order, so a non-commutative body may disagree between the two. The lint refuses a fold whose body is not `+`, `&&`, `\|\|`, or `incl` of the accumulator **(the exact rule is open)**. |
| `(lo until hi)`, `(lo to hi)` with `forall` / `exists` | Int | `OP_RANGE` |
| `K.values.forall(p)` / `exists` | enum `K` | quantifier with `over = K` |
| `xs :+ x`, `xs.headOption`, `xs.drop(n)` | List | `OP_APPEND`, `OP_HEAD_OPTION`, `OP_DROP` |
| `*`, `/`, `%` | Int | new binary ops |

Refused, with the reason:

- `+` and `-` on collections. `.plans/DSL_OPERATORS.md` rule 2, "one symbol, one meaning", as in
  bounded-collections §3.1.
- `toList`, `head` and iteration-order reads of a `Set` or `Map`.
- `Map.forall((k, v) => …)`: tuples are no IR type, and the shape of Scala's untupled tree is open
  (bounded-collections §8.4.9). Write `m.keySet.forall(k => … m(k) …)`.

**Lifter types** (`model/framework/export/Types.scala`):

- `typeRef` maps `Map`, `Set`, `List` and `Int` fields. The refusals at `61-66` and `110-117` become
  refusals only for function-local uses that remain unsupported.
- `declareType`'s "no finite catalog" message (`137-141`) remains for types that are neither data
  nor enums (`String` in state, functions).
- `readFinites` (`187-202`) keeps reading opaque `Finite.upTo` ranges. Its channel wiring is replaced
  by type recognition (§4.9).

**Scala-side catalog walks** (bounded-collections §8.4.1, `bounded-collections.md:98-107`):

- The overlap check, `Refinement.unclosed`, the machine law tests and step tables are removed from
  Scala (D4; §4.4, §4.11, §5.1). No Scala code walks a catalog or explores.
- So a partial `m(k)` that throws on an *unreachable* state no longer fails the build. This is the
  main reason native `Map` was impractical under the catalog design (§8.4.6 there).

### 6.6 Overlap with `bounded-collections.md`

| That design | Here |
| --- | --- |
| §3.4 **input-reading guards**, `where((s, i) => …)` | **Survives unchanged, and is a prerequisite.** Every native sample in §8 uses it. |
| §3.1 **`Set` operations** and their liftable rows, and the refusals of `+`/`-` and order-dependent reads | Survive. |
| §3.2 **quantifier semantics by desugaring**, short-circuit and holes | Survives, over value contents in canonical order. |
| §4.1 **one `Collection` Expr kind with an op enum** | Survives in shape. `TotalOf`/`SeqOf{max}` become `MapOf` and unbounded `list`. |
| §4.3 **`interp/collections.go` as a deep module** | Survives. Its catalog index is replaced by type order: no catalog is listed. |
| §4.2 **admission** of collections, canonical literals, recursion | Survives, minus the size closed forms. |
| §2.3 **brace spellings**; canonical order by **K-catalog index** | Spelling survives. Order becomes type order, which equals catalog order for finite `K` but not the record-of-Booleans *set* order of §2.1. Sets are never enumerated, so that order no longer exists. |
| §2.1, §2.2 **`Total[K, V]`, `ListUpTo[A, N]`, `Finite` givens for collections, `catalog[K]`** | **Dropped** (D5a, settled: native partial `Map` only, no `Total`). |
| §5 **state-space counts, ceilings, factorized `LimitError`** | **Unnecessary** for unbounded types. The factorization idea survives as a *reached-values* report per field for open graphs (§7.2). |
| §4.4 **`unreachable-value` leaves per key** | Unnecessary. Collection leaves are combinations (§4.5). |
| §6 **Quint mapping** | Survives, with partial `Map` → `K -> V`. ITF re-canonicalization uses type order. |
| §6 **`Finite.spell`** for step tables | **Unnecessary.** Scala writes no step tables (D4). Spelling is Go's alone. |
| §9 Phase 4 **table-isomorphism test** (old encoding vs new) | Survives as a reachable-graph isomorphism test (§10, Phase 5). |
| §10 open question 11 (spelling `None` entries of `Total[K, Option[V]]`) | Resolved: native partial maps omit absent keys. |
| §8.4.9 conditions for option (a), native everywhere: "Or if the reader stops deriving tables over the whole catalog … and derives them over reachable states" | **This design is that condition.** |

---

## 7. Backends

### 7.1 Where they stand

The facts below come from `tools/umpire/export/README.md:22-36, 128-151`; `quint.go`;
`agreement.go`; `verify.go`; `tools_test.go:27-45`.

- **Quint is the only exporter.** Quint 0.33.0 via `quint.sh`, and Apalache 0.62.1 via
  `quint verify`. There is **no TLA+ export and no TLC runner**.
- **`make umpire-check-backends`** (`Makefile:674-675`) runs the opt-in Go tests over four Temporal
  slices (`quint_test.go:26`), not the samples.
- **The agreement compares Go's reachable relation** with Quint's evaluation of the exported step
  functions: starts, reachable states, ends, classes and every reachable state × class
  (`agreement.go:24-96, 189-252`).
- **Apalache checks monitor invariants** up to the product depth + 1 (`quint.go:1051-1069`;
  `verify.go:50-71`). Its counterexamples are replayed through Go (`slice.go:627`).
- **Not exported, verbatim reasons:**
  - refinement (`quint.go:117-119`);
  - progress (`130-131`);
  - Queries, Scenarios and Limits (`134-135`);
  - channels (`84`, `420`, `714`);
  - holes (`106`, `418`).

The agreement is already reachable-only. So is Quint's `reach` fixpoint, whose round count is the
only number Go passes in (`quint.go:827-835`). On-demand rows change nothing in it, except that "Go's
table depth" becomes the explored depth.

### 7.2 Collections in the export

| IR | Quint | Notes |
| --- | --- | --- |
| `int` (state) | `int` | Already the mapping for IntRange (`quint.go:217-218`). |
| `set` | `Set[T]` | `OP_SET_OF` → `Set(a, b)`; `OP_INCL` → `s.union(Set(x))`; `OP_EXCL` → `s.exclude(Set(x))`. |
| `map` | `K -> V` | `OP_MAP_OF` → `Map(k -> v)`; `OP_GET` → `m.get(k)`; `OP_UPDATED` → `m.set(k, v)` for an existing key or `m.put(k, v)` **(unverified name in 0.33.0)**; `OP_HAS_KEY` → `m.keys().contains(k)`; `OP_KEYS` → `m.keys()`. |
| `list` | `List[T]` | `OP_CONTAINS` on a list keeps `select(...).length() > 0` (`quint.go:480`). |
| quantifiers | `forall`, `exists`, `filter`, `map`, `fold` | `fold` over a Quint `Set` iterates in Quint's own order, so the same commutativity rule as §6.5 applies. |
| `OP_RANGE` | `lo.to(hi - 1)` | |
| input `domain` | `Set(v1, …)` in `membersOf` (`quint.go:227-276`) | |

**ITF read-back** (`export/itf.go:162-168, 340-414`) decodes `{"#set": […]}` and
`{"#map": [[k, v], …]}` and re-canonicalizes in type order before comparing `Key()`s. This is
bounded-collections §6 with type order in place of the catalog index.

### 7.3 Backends decide (D3, settled)

**Settled (owner, 2026-10-13):** any Query may be answered by Umpire, Apalache or TLC.

**How a Query selects its checker.**

- A Query declares it, with Umpire as the default:

  ```scala
  val consistencyHolds = query verify consistencyInv in anyRun limits transaction checkedBy apalache
  val everyoneCommits  = query find everyoneCommits in anyRun limits transaction            // Umpire
  val exhaustive       = query verify consistencyInv in anyRun limits transaction checkedBy tlc
  ```

- The IR's `Query` (`claim.proto:68-93`) gains `Checker checker = 11`, an enum of `UMPIRE`,
  `APALACHE` and `TLC` with Umpire as the zero value, plus `repeated Checker confirm = 12` for checkers
  that must agree.
- The IR's `Limits` (`claim.proto:192-200`) is the bound for every checker:
  - Apalache: `--max-steps = steps`;
  - TLC: depth `steps`, with no state cap unless `search` is set;
  - Umpire: as today.
- `.open(because = ...)` (D1) applies to every checker whose answer is bounded. That is always the
  case for Apalache without an inductive invariant, and for TLC only when it stops at a cap.

**What a backend must cover before it may decide a Query.** Admission refuses `checkedBy b` for a
Query any of whose ingredients `b`'s exporter does not cover, naming the ingredient. Today's
exporter lists these as not exported:

| Ingredient | Today (`tools/umpire/export`) | Needed before a backend decides |
| --- | --- | --- |
| Machine transitions | exported (`quint.go:781-858`) | — |
| Monitors | exported; Apalache checks their invariants (`quint.go:1051-1069`) | — |
| Properties (every step, `when`, `whenAction`) | exported as `claims` for agreement (`quint.go:953-1011`), not as Apalache invariants | invariants and `hist`-based transition properties |
| Scenarios and Limits | not exported: "no Scenario or Limits" (`quint.go:134-135`) | a start per Scenario; pinned schedules as a schedule variable; Limits as step bounds |
| `find` Queries | not exported | the negated Property as an invariant, whose counterexample is the witness |
| Refinement (`through`, and refinement checks) | not exported (`quint.go:117-119`) | the product step relation over `hist` (§7.6) |
| Progress | not exported (`quint.go:130-131`) | a deadline counter, and fairness where the backend has it (§7.6) |
| Channels | not exported (`quint.go:84, 420, 714`) | derived deliver, lose and redeliver rows as explicit actions, or channels re-expressed over `List`/`Set` (D5b) |
| Holes | not exportable (`quint.go:106, 418`) | never: a Query whose region reaches a hole is Umpire's alone, and admission refuses `checkedBy` once Go has found that hole |
| Compositions | exported only as Go's composed table (`composed.go:47-54`) | members and syncs exported natively |

**Witnesses, live Cases and receipts.**

- **Every trace a backend returns is replayed through Go** before it counts: a `find` witness, or a
  `verify` counterexample (`Slice.Replay`, `export/slice.go:627`). A trace that does not replay is
  `witness-rejected`.
- **Live Cases are lowered only from a Go-replayed witness.** `Lower` requires a `Found` receipt
  with a witness (`lower/lower.go:883-887`), so a backend's `find` lowers exactly when its trace
  replays.
- **Receipts name the checker that answered.** `check.Receipt` (`check/checking.go:142-192`) gains
  `Checker` (from Phase 1, always `umpire` until Phase 6), the checker's version, and its scope in
  its own terms (`"Apalache 0.62.1: every run of at most 10 steps"`; `verify.go:132-179`).
- **The VISION precedence rule still decides,** with "decide" now open to every checker: a violation
  established by any checker stands over a pass, and a hole Go found makes a pass `incomplete`.

### 7.4 How answers are compared

**When more than one checker answers a Query** — its `checker` plus its `confirm` list, or every
checker in the backend gate — their answers must agree. Disagreement is a `disagreed` receipt and
fails the gate.

| Comparison | Mechanism | Today |
| --- | --- | --- |
| Transition relation | Reachable states, ends, classes, rows per reachable state × class | Exists (`agreement.go:189-232`) |
| Graph size | Umpire's `Reached` against TLC's "distinct states found"; equal on a closed region | New |
| Verdicts | Per Query: kind against kind. Two bounded answers agree only within the smaller bound. | Exists for monitors (`verify.go:132-179`) |
| Counterexamples and witnesses | Replayed through Go; a non-replaying trace is `witness-rejected` | Exists |
| Normalization | ITF values decoded by IR types and compared by `Key()`, with sets and maps re-canonicalized in type order | Exists for enums and records; extend (§7.2) |
| Integers past 2^63 | Go reports `incomplete` (checked `int64`, D6; §6.4). Apalache and the Quint evaluator are unbounded. Go's `incomplete` there is reported as *incomparable*, not as a disagreement. TLC's integer range is narrower still **(unverified: 32-bit in TLC)**. | New |

### 7.5 The VISION amendment (draft for `.plans/UMPIRE4_VISION.md:101-102`)

Today:

> **One meaning, one source.** The IR has one interpretation that every Go component shares, and
> every fact has one declaration. Other checkers may give a second opinion but never decide.

Proposed:

> **One meaning, one source.** The IR has one interpretation, defined by `model/framework/SEMANTICS.md` and
> implemented by the Go reader, and every fact has one declaration. A Query may name the checker that
> decides it: the Go checker by default, or Apalache or TLC through the export of the same IR. A
> checker may decide only a Query whose every ingredient its export covers. Every trace it returns
> is replayed through the Go reader before it counts. Its receipt names it and the bounds it checked
> within. Where two checkers answer one Query, their answers must agree.

The Checker component's description (`.plans/UMPIRE4_VISION.md:62-67`, "Interprets the IR, answers
Queries by bounded exhaustive exploration") gains: "or delegates a Query to the backend it names, and
replays what that backend returns".

**Module impact.** `check` imports "Umpire IR, ir, interp and private engine", and `export` imports
`check` (`.plans/UMPIRE_MODULES.md`, Checking and Export rows). So deciding through a backend cannot
live in `check` without a cycle. It needs a thin orchestration module above both, for example
`tools/umpire/answer`. That module:

- runs `check.Check`;
- runs the named backends through `export`;
- replays their traces;
- emits one receipt per Query.

`lower`, `lint` and the commands then read receipts from it in place of calling `check.Check`
directly (`lower/lower.go:103-117`; `lint/lint.go:147-169`). The Export module's job changes from
"Compare Quint's reading with the reader" to "Export a Model to a backend, and run that backend to
decide or confirm a Query". `TestQuintOwnsBackendExport` (`.plans/UMPIRE_MODULES.md:364-368`)
changes with the TLA+ route (open question 13).

### 7.6 What each backend can and cannot answer

| Claim | Umpire | Apalache (via Quint) | TLC (needs a TLA+ path) |
| --- | --- | --- | --- |
| Safety Property / monitor invariant | yes | yes, bounded *k*; exported today for monitors | yes, exhaustive on a finite graph |
| Transition Property (`holdsAcross`) | yes | yes, over the existing `hist` variable (`quint.go:1017-1032`) **(Property export over `hist` unverified)** | yes, as an action property |
| Refinement | yes (§4.4) | as a step invariant over `hist`: `mapped(prev) == mapped(cur) or carried(prev, cur)`. Not exported today. Feasible. | yes, natively: `INSTANCE Consensus` and `Spec => C!Spec`, which is the theorem Voting cites |
| Progress (`leadsTo` within *n*, weak fairness) | yes, bounded; deadlock, fair cycle, missed deadline | the `within` bound encodes as an invariant over a step counter; fairness is not supported by Quint verify **(unverified)** | `<>`/`~>` with `WF_vars` is native; the `within` bound needs a counter |
| Channels | yes | not exported today (`quint.go:84`). Once a channel is a `List`/`Set` with explicit delivery actions, it exports like any field. | same |
| Holes | `incomplete` | not exportable (`quint.go:106, 418`) | not exportable |
| Pinned Scenarios and `find` | yes, with witnesses | a `run` test with `.then` chains, as `TwoPhaseCommit.qnt.test`'s `successTest` | `ASSUME` / trace exploration; impractical |

**A TLC path.** TLC may decide (D3), so a TLA+ route is required. There are two:

- Quint transpiles to TLA+ (`quint compile --target tlaplus`) **(unverified for 0.33.0)**;
- a direct TLA+ writer beside `quint.go`, which is a second exporter.

`TestQuintOwnsBackendExport` enforces a single export backend (`.plans/UMPIRE_MODULES.md:364-368`),
so either route changes that test's contract. Which route to take is open question 13.

---

## 8. Before and after

All after-code is **(proposed)**. It uses the input-reading guards of bounded-collections §3.4 and
the surface of §6.5. State counts marked *hand count* were computed by hand, not run; *measured* counts are §12's.

### 8.1 Bank

Before: `model/framework/samples/Bank.scala:23-57`. There are 35 lines of ladders for one map, a
catalog of 9⁴ = 6,561 states, and 4·4·2·4 = 128 classes, so 839,808 evaluations
(`bounded-collections.md:556-557`).

```scala
object Bank:
  enum Addr:
    case king, donkeykong, peaches, mario
  enum Denom:
    case banana, burger

  val supply = 2

  // Quint's `Addr -> Coins`: an account with no coins of a denomination has no entry for it, so
  // there is one spelling of "no balance" and `after.state == before` compares like for like.
  type Coins = Map[Denom, Int]
  final case class Balances(of: Map[Addr, Coins])

  def balanceOf(b: Balances, addr: Addr, denom: Denom): Int =
    b.of.getOrElse(addr, Map.empty).getOrElse(denom, 0)

  def withBalance(b: Balances, addr: Addr, denom: Denom, amount: Int): Balances =
    val coins = b.of.getOrElse(addr, Map.empty)
    val next = if amount == 0 then coins.removed(denom) else coins.updated(denom, amount)
    b.copy(of = if next.isEmpty then b.of.removed(addr) else b.of.updated(addr, next))

  def sumForDenom(b: Balances, denom: Denom): Int =
    b.of.keySet.foldLeft(0)((sum, addr) => sum + balanceOf(b, addr, denom))

  enum Outcome:
    case accepted, invalidCoinsOrInsufficientFunds, overflow
  given Ok[Outcome] = Ok(Outcome.accepted)

  object client extends Actor:
    // Quint draws amounts from the integers; the model draws them from -1..3, so a send may ask a
    // negative amount (finding 18) or too much.
    val send = action(this).input[Addr]("fromAddr").input[Addr]("toAddr")
      .input[Denom]("denom").input[Int]("amount", oneOf(-1 to 3))

  object Model extends Machine[Balances, Outcome, Nothing]:
    val init = Balances(of = Map(Addr.king -> Map(Denom.banana -> supply, Denom.burger -> supply)))
    def end(b: Balances) = true
    val reach = Bounds.wide

    object effects:
      def send(b: Balances, fromAddr: Addr, toAddr: Addr, denom: Denom, amount: Int) =
        val fromBalance = balanceOf(b, fromAddr, denom)
        val toBalance = balanceOf(b, toAddr, denom)
        if amount <= 0 || fromBalance < amount then
          reject(Outcome.invalidCoinsOrInsufficientFunds, b)
        else if fromAddr == toAddr then stay(b)
        else if toBalance + amount > supply then reject(Outcome.overflow, b)
        else enter(withBalance(withBalance(b, fromAddr, denom, fromBalance - amount),
          toAddr, denom, toBalance + amount))
      // sendFromNowhere as before, through balanceOf/withBalance

    object rules extends Rules:
      on(client.send) {
        always ~> effects.send
      }
  // Leaky, the Properties and the Queries as before (Bank.scala:128-158)
```

Every rejecting case of `send` is a real refusal with an outcome, so `always ~> effects.send` stays.
This is the "explicit about what it refuses" form (SEMANTICS §6, line 425-427).

**What the checker would report.**

- **Reachable states.** The reachable states are the distributions of two indistinguishable bananas
  and two burgers over four accounts. There are C(5,3) = 10 per denomination, so 100 states
  (*hand count*, and measured on today's encoding as 100, §12), against a catalog of 6,561.
- **Evaluations.** With 4·4·2·5 = 160 classes, that is 16,000 evaluations (*hand count*), against
  839,808.
- **The graph closes.** The sample's own comment says "every distribution of the supply is at most
  four sends from the start" (`Bank.scala:151`).
- **The four Queries keep their expected answers** (`model/framework/check/Samples.scala:37-64`):
  `Verified`, `Verified`, `Found`, `Counterexample`.
- **Leaky's coins come from nowhere,** so its reachable graph grows until `toBalance + amount >
  supply` stops it. It stays closed.

### 8.2 TwoPhaseCommit

Before: `model/framework/samples/TwoPhaseCommit.scala`, 294 lines:

- `States`, `Members`, `Messages` and seven ladder helpers (27-85);
- 12 per-RM effect wrappers (177-188);
- 12 per-class rules (209-244);
- a catalog of 64 × 3 × 8 × 32 = 49,152 states (`bounded-collections.md:559-560`).

```scala
object TwoPhaseCommit:
  enum ResourceManager:
    case rm1, rm2, rm3
  enum ResourceManagerState:
    case working, prepared, committed, aborted
  enum TransactionManagerState:
    case init, committed, aborted
  // Quint's message type: Abort | Commit | RMPrepared(rm)
  enum Message:
    case abort, commit
    case prepared(rm: ResourceManager)

  val resourceManagers: Set[ResourceManager] =
    Set(ResourceManager.rm1, ResourceManager.rm2, ResourceManager.rm3)

  final case class Transaction(
      rmStates: Map[ResourceManager, ResourceManagerState],
      tmState: TransactionManagerState,
      // the resource managers from which the transaction manager has received RMPrepared
      tmPrepared: Set[ResourceManager],
      messages: Set[Message]
  )

  def rmIs(s: Transaction, rm: ResourceManager, phase: ResourceManagerState): Boolean =
    s.rmStates(rm) == phase
  def done(phase: ResourceManagerState): Boolean =
    phase == ResourceManagerState.committed || phase == ResourceManagerState.aborted
  def allCommitted(s: Transaction): Boolean =
    resourceManagers.forall(rm => rmIs(s, rm, ResourceManagerState.committed))

  def mayPrepare(s: Transaction, rm: ResourceManager) = rmIs(s, rm, ResourceManagerState.working)
  def mayAbort(s: Transaction, rm: ResourceManager) =
    rmIs(s, rm, ResourceManagerState.working) || s.messages.contains(Message.abort)
  def mayCommit(s: Transaction, rm: ResourceManager) =
    s.messages.contains(Message.commit) && !rmIs(s, rm, ResourceManagerState.committed)
  def mayReceive(s: Transaction, rm: ResourceManager) =
    s.tmState == TransactionManagerState.init && s.messages.contains(Message.prepared(rm))

  enum Outcome:
    case accepted
  given Ok[Outcome] = Ok(Outcome.accepted)

  object rm:
    val prepare = internal.input[ResourceManager]("rm")
    val abort = internal.input[ResourceManager]("rm")
    val commit = internal.input[ResourceManager]("rm")
  object tm:
    val addPrepared = internal.input[ResourceManager]("rm")
    val decideOnCommit = internal
    val decideOnAbort = internal

  object Model extends Machine[Transaction, Outcome, Nothing]:
    val init = Transaction(
      rmStates = Map(ResourceManager.rm1 -> ResourceManagerState.working,
        ResourceManager.rm2 -> ResourceManagerState.working,
        ResourceManager.rm3 -> ResourceManagerState.working),
      tmState = TransactionManagerState.init, tmPrepared = Set.empty, messages = Set.empty)
    def end(s: Transaction) =
      s.tmState != TransactionManagerState.init && resourceManagers.forall(rm => done(s.rmStates(rm)))
    val reach = Bounds.wide

    object states:
      def eagerlyCommits(s: Transaction): Boolean =
        s.tmState == TransactionManagerState.init && s.tmPrepared.nonEmpty

    object effects:
      def prepare(s: Transaction, rm: ResourceManager) = enter(s.copy(
        rmStates = s.rmStates.updated(rm, ResourceManagerState.prepared),
        messages = s.messages.incl(Message.prepared(rm))))
      // there is no need to let others know that a resource manager aborted
      def abort(s: Transaction, rm: ResourceManager) =
        enter(s.copy(rmStates = s.rmStates.updated(rm, ResourceManagerState.aborted)))
      def commit(s: Transaction, rm: ResourceManager) =
        enter(s.copy(rmStates = s.rmStates.updated(rm, ResourceManagerState.committed)))
      def addPrepared(s: Transaction, rm: ResourceManager) =
        enter(s.copy(tmPrepared = s.tmPrepared.incl(rm)))
      def decideOnCommit(s: Transaction) = enter(s.copy(
        tmState = TransactionManagerState.committed, messages = s.messages.incl(Message.commit)))
      def decideOnAbort(s: Transaction) = enter(s.copy(
        tmState = TransactionManagerState.aborted, messages = s.messages.incl(Message.abort)))

    object rules extends Rules:
      on(rm.prepare) {
        where((s, r) => mayPrepare(s, r)) ~> effects.prepare
      }
      on(rm.abort) {
        where((s, r) => mayAbort(s, r)) ~> effects.abort
      }
      on(rm.commit) {
        where((s, r) => mayCommit(s, r)) ~> effects.commit
      }
      on(tm.addPrepared) {
        where((s, r) => mayReceive(s, r)) ~> effects.addPrepared
      }
      on(tm.decideOnCommit) {
        where(s => s.tmState == TransactionManagerState.init && s.tmPrepared == resourceManagers) ~>
          effects.decideOnCommit
      }
      on(tm.decideOnAbort) {
        where(_.tmState == TransactionManagerState.init) ~> effects.decideOnAbort
      }

  // EagerCommit, consistent, the claims, the progress claim and the Queries as before
  // (TwoPhaseCommit.scala:253-294), with `consistent` written as Quint writes it:
  def consistent(s: Transaction): Boolean =
    !(resourceManagers.exists(rm => rmIs(s, rm, ResourceManagerState.committed)) &&
      resourceManagers.exists(rm => rmIs(s, rm, ResourceManagerState.aborted)))
```

The file shrinks by roughly half (*estimate*). Adding a fourth resource manager is a one-line
change: an enum case and the `resourceManagers` and `init` literals. Today that same change is
refused at 786,432 states (`bounded-collections.md:503-511`).

**What the checker would report.**

- `consistencyHolds` `Verified`, `successTest` `Found`, `refutesEagerCommit` `Counterexample`.
- `commitReachesEveryone` `Verified`.
- The reachable count is reported by the checker. It is not computed here. It is far below the
  catalog of 49,152, because, for example, `tmPrepared` can only hold resource managers that have
  sent `prepared`.

The fixed `rmStates` keys keep `s.rmStates(rm)` total on every reachable state. A typo that removed a
key would surface as an `incomplete` row with its path (§6.5), not as a Scala exception in the
overlap check.

### 8.3 Voting, without `inv`

Before: `model/framework/samples/Voting.scala`.

- `Ballots` with `Option[UpTo[1]]` standing in for −1 (23-27).
- Quorums unrolled into three disjuncts (67-75, 92-95, 109-112).
- A 30-line invariant `inv` guarding every step (125-129, 171, 182), "because refinement is checked
  on unreachable rows".
- `refused` rows for disabled actions (29-30, 173, 185).
- A catalog of 27³ = 19,683 states × (6 + 12) classes = 354,294 evaluations.

```scala
object Voting:
  enum Acceptor:
    case a1, a2, a3
  enum Value:
    case v1, v2

  val acceptors: Set[Acceptor] = Set(Acceptor.a1, Acceptor.a2, Acceptor.a3)
  // any two of the three acceptors
  val quorums: Set[Set[Acceptor]] =
    Set(Set(Acceptor.a1, Acceptor.a2), Set(Acceptor.a1, Acceptor.a3), Set(Acceptor.a2, Acceptor.a3))
  val values: Set[Value] = Set(Value.v1, Value.v2)

  // Consensus: the set of values chosen, as Quint's `chosen`
  final case class Decision(chosen: Set[Value])

  final case class Vote(ballot: Int, value: Value)
  // Quint's `votes: str -> Set[(int, int)]` and `maxBal: str -> int`; -1 is "no ballot yet"
  final case class Votes(votes: Map[Acceptor, Set[Vote]], maxBal: Map[Acceptor, Int])

  enum Outcome:
    case accepted
  given Ok[Outcome] = Ok(Outcome.accepted)

  object voter extends Actor:
    // Quint's Ballot is Nat; the model draws it from 0..1, as Quint's `Next` draws `oneOf(Ballot)`
    val increaseMaxBal = action(this).input[Acceptor]("a").input[Int]("b", oneOf(0 to 1))
    val voteFor =
      action(this).input[Acceptor]("a").input[Int]("b", oneOf(0 to 1)).input[Value]("v")

  def votedFor(s: Votes, a: Acceptor, b: Int, v: Value): Boolean = s.votes(a).contains(Vote(b, v))
  def chosenAt(s: Votes, b: Int, v: Value): Boolean =
    quorums.exists(q => q.forall(a => votedFor(s, a, b, v)))
  // a value is chosen in some ballot in which some acceptor voted for it
  def isChosen(s: Votes, v: Value): Boolean =
    acceptors.exists(a => s.votes(a).exists(vt => vt.value == v && chosenAt(s, vt.ballot, v)))
  def chosen(s: Votes): Set[Value] = values.filter(v => isChosen(s, v))
  def didNotVoteAt(s: Votes, a: Acceptor, b: Int): Boolean = s.votes(a).forall(_.ballot != b)

  def showsSafeAt(s: Votes, q: Set[Acceptor], b: Int, v: Value): Boolean =
    q.forall(a => s.maxBal(a) >= b) &&
      (-1 until b).exists(c =>
        (c == -1 || q.exists(a => votedFor(s, a, c, v))) &&
          (c + 1 until b).forall(d => q.forall(a => didNotVoteAt(s, a, d))))

  def mayVote(s: Votes, a: Acceptor, b: Int, v: Value): Boolean =
    s.maxBal(a) <= b && didNotVoteAt(s, a, b) &&
      acceptors.excl(a).forall(c => s.votes(c).forall(vt => vt.ballot != b || vt.value == v)) &&
      quorums.exists(q => showsSafeAt(s, q, b, v))

  object Consensus extends Machine[Decision, Outcome, Nothing]:
    val init = Decision(chosen = Set.empty)
    def end(d: Decision) = true
    object effects:
      def choose(d: Decision, @unused a: Acceptor, @unused b: Int, v: Value) =
        enter(d.copy(chosen = Set(v)))
    object rules extends Rules:
      on(voter.voteFor) {
        where(_.chosen.isEmpty) ~> effects.choose
      }

  object Model extends Machine[Votes, Outcome, Nothing]:
    val init = Votes(
      votes = Map(Acceptor.a1 -> Set.empty, Acceptor.a2 -> Set.empty, Acceptor.a3 -> Set.empty),
      maxBal = Map(Acceptor.a1 -> -1, Acceptor.a2 -> -1, Acceptor.a3 -> -1))
    def end(s: Votes) = true
    val reach = Bounds.wide

    object refinement extends Refinement(Consensus):
      def toProduct(s: Votes): Decision = Decision(chosen = chosen(s))
      val unobservable = List(voter.increaseMaxBal)

    object effects:
      def increaseMaxBal(s: Votes, a: Acceptor, b: Int) =
        enter(s.copy(maxBal = s.maxBal.updated(a, b)))
      def voteFor(s: Votes, a: Acceptor, b: Int, v: Value) = enter(s.copy(
        votes = s.votes.updated(a, s.votes(a).incl(Vote(b, v))),
        maxBal = s.maxBal.updated(a, b)))
      // A vote that skips the quorum's showing that v is safe at b.
      def voteUnsafe(s: Votes, a: Acceptor, b: Int, v: Value) = voteFor(s, a, b, v)

    object rules extends Rules:
      on(voter.increaseMaxBal) {
        where((s, a, b) => b > s.maxBal(a)) ~> effects.increaseMaxBal
      }
      on(voter.voteFor) {
        where((s, a, b, v) => mayVote(s, a, b, v)) ~> effects.voteFor
      }

  object UnsafeVoting extends Derived(Model.unmonitored.rebind(on(voter.voteFor) {
        where((s, a, b, v) => s.maxBal(a) <= b && didNotVoteAt(s, a, b)) ~> Model.effects.voteUnsafe
      })), NegativeControl

  // VotesSafe and OneValuePerBallot, now a Property and no longer a guard
  def votesSafe(s: Votes): Boolean = ...      // as Quint's VotesSafe, over acceptors × s.votes(a)
  def oneValuePerBallot(s: Votes): Boolean =
    acceptors.forall(a1 => acceptors.forall(a2 => s.votes(a1).forall(x => s.votes(a2)
      .forall(y => x.ballot != y.ballot || x.value == y.value))))
  // chosenStays, consistency, invariant (over votesSafe && oneValuePerBallot), decides and the six
  // Queries as before (Voting.scala:205-234)
```

`Consensus.choose` keeps today's spelling. Whether `Set(v)` lifts there is part of §6.5's fixtures.
The `visibleOutcomes` line goes with `refused`, per samples-vs-quint finding 3
(`samples-vs-quint.md:209-210`). Whether an `accepted` vote that chooses nothing still reads as a
stutter without it depends on the refinement's default outcome visibility (`machine.proto:91-94`:
"Empty when the refinement names no outcome the refined machine sees"), so the line is not needed
**(confirm by the samples check)**.

**What the checker would report.**

- **The graph closes.** Votes ⊆ {0,1}×{v1,v2} per acceptor, maxBal ∈ {−1,0,1}, three acceptors. The
  reached count is the checker's to report.
- **Refinement:** `Voting.Model refines Consensus` holds on every reachable row (closed). No `inv`
  is needed: the unreachable half-chosen states that forced it (`samples-vs-quint.md:247-248`) are
  never explored.
- **`invariantHolds`** is still `Verified`. The author may instead declare the invariant on the
  refinement (§5.4), where Go checks it on every reached state. The invariant is now a claim the Model proves, not an
  assumption it enforces. This is TLA+'s `Invariance == Spec => []Inv` (`Voting.qnt.test`, near
  the end).
- **`refutesUnsafeVoting`** is `Counterexample`, and now on the intended machine. Today
  `UnsafeVoting`'s `increaseMaxBal` keeps the `inv` guard (`Voting.scala:171, 201-203`; finding #4's
  "the guard leaks into the negative control").
- **Changing `oneOf(0 to 1)` to `oneOf(0 to 2)`** is a one-token change. Today it needs a new `at2`
  field, `UpTo[2]`, and a re-derived `inv`.
- **If refinement failed on a reachable row,** the result would be `refinement-rejected` with a
  shortest path from `init`. That is the witness that is missing today for unreachable rows
  (`refine.go:200-208`).

---

## 9. Risks and mitigations

### 9.1 Silent weakening of guarantees

Three ways a result can say less than it seems to.

**1. Unreachable territory is no longer checked.** §1.8 lists what is lost.

The static checks of §5 still cover, over all states:

- every overlap between rules headed by phases, roles or classes (§5.1);
- missing enum cases in every function, reachable or not (§5.2);
- unmapped phases and role closedness for phase-to-phase refinement maps (§5.2).

They do **not** cover:

- `where`-guard overlaps;
- refinement mismatches that depend on values beyond the phase;
- value-dependent holes.

All three are found, with a path, as soon as a root reaches their state. They are never found
before.

**2. Open regions.** Every result over an open region is bounded. Today the only bounded result is
the Query search; every table-level check is exhaustive over the catalog. Mitigations:

- D1's policy: the `open-graph` lint, and `.open(because = ...)` on a Temporal `verify`;
- `Closed` in every receipt;
- the dynamic growth report (§5.3).

**3. A backend's bounds.** Under D3 a Query may be decided by Apalache, whose pass covers every run
of at most *k* steps (`verify.go:132-179`), or by TLC, which may stop at a cap. Such a pass is
bounded in the backend's own terms. Mitigations:

- the receipt's `Checker` and scope (§7.3);
- D1's `.open(because = ...)` required for a bounded answer to a Temporal `verify`;
- coverage admission: a backend never decides a Query it cannot express (§7.3);
- Go replay of every trace;
- agreement wherever a second checker runs (§7.4).

**4. Narrow inputs.** `oneOf(0 to 1)` silently excludes ballot 2. That is equally true of `UpTo[1]`
today. `oneOf` domains are listed in `umpire-lint --tables`.

**Roots.** A Query over a start that no root covers would be unchecked. §5.6 makes every Scenario
and Exploration start a root by construction, so no such Query can exist.

**Integers.** Past 2^63, Go reports `incomplete` while Apalache and the Quint evaluator carry on
(D6, §7.4). Answers there are incomparable, never silently equal.

**Parity.** Phase 1's test proves that, on every current model, every answer is unchanged and every
Case is unchanged except for fingerprints. So the weakening is confined to what §1.8 lists, and does
not show in today's results.

### 9.2 State explosion found late

`Preflight` refused oversized models statically (`interp/machine.go:380-393`), and it is gone.

**Mitigations:**

- The reach bound counts states and evaluations before making them (`compose.go:425-491` is the
  pattern).
- `unbounded-growth` warnings before running (§5.3), and the growth report after (§5.3).
- Explored counts recorded in every receipt from Phase 1, so that a regression of an order of
  magnitude is visible in review.

### 9.3 Fingerprint churn

**Once.** Every Behavior, Property, Query and projection fingerprint changes once, in Phase 1, when
the structural hash replaces the table hash (D2, §4.13). Through `CaseFingerprint`
(`common/testing/testpilot/assessment.go:192-202`), every checked-in Case changes, and so do both
pinned trees:

- the functional fixtures (`tests/testcore/testpilot/testdata/generated`);
- the canary (`tools/canary/casebinding/testdata`) (`model/README.md:2198-2203`).

**Afterwards.** A machine's Cases churn whenever its behavior-affecting IR changes, including
meaning-preserving rewrites such as an equivalent guard or an inlined helper. They do not churn on
moves, renames, reorders, documentation, `because` text, unreachable-only edits, or added Scenarios.
Compared with today, refactors churn more often and incidental edits churn less.

**Mitigations:**

- **One batch, sealed by the equivalence projection.** It already strips fingerprints
  (`Equivalence.scala:374-388`).
- **Goldens updated in the same change:**
  - `engine/fingerprint_test.go:13-24`;
  - `engine/composekeys_test.go:12-55`;
  - `check/realizer_test.go:96-102`;
  - `lower/internal/producer/fingerprint_test.go:46-53`.
- **A new tag,** `behavior-fingerprint/v2`, so stale readers reject v1 data (ART-04).
- **Canonicalization that erases helper names and sorts declarations** (§5.5).
- **An allowlist of hashed fields,** so later IR additions do not churn.
- **A reviewer's tool.** When a machine's hash changes, the change-scoped check also reports whether
  its Go reachable dump changed (§4.11). "Structure changed, behavior equal" is then visible, and the
  Case regeneration can be reviewed as a no-op.

### 9.4 Scala and Go disagreeing on order

Scala iterates hash order; Go canonicalizes in type order. Mitigations:

- Refuse order-dependent reads (§6.5).
- Restrict folds to commutative bodies (§6.5).
- Since Scala no longer spells keys for step tables (D4), the only remaining spelling contract is the
  lifter's literal keys for gap class keys (`model/framework/export/Gaps.scala:546-583`). It is
  pinned by a test against Go's `Key()`.

### 9.5 Performance

Memoized rows add map lookups. A graph whose reachable set is close to its catalog pays slightly
more than an eager build: TwoLayeredCache reaches 11% of evaluations and Kettle 29% (§12.1), so
neither is close.

**Mitigation:** Phase 1's parity test records timings, and the gate's wall time is the acceptance
bound.

### 9.6 Module boundaries (`.plans/UMPIRE_MODULES.md`)

| Module | Today's job / interface | Impact |
| --- | --- | --- |
| DSL (`model/framework/dsl`) | "Declare finite Models in Scala." "No native evaluator." | The job becomes "Declare Models in Scala". With D4 the DSL runs **no** Model code for checking. The overlap check, `unclosed` and step tables go, which is closer to "no native evaluator" than today. `Data` and `oneOf` join the interface. |
| IR (`api/umpire/v1`) | "Represent a lifted Model." | Additive: collections, `reach`, `RuleGuard`, `Match.scrutinee_type`, `Case.roles`, `Refinement.invariant`, input `domain`, `Query.checker`/`confirm`; `Query.total` reserved. The SPEC's "finite types" (`.plans/UMPIRE4_SPEC.md:42`) needs GOV-02. |
| IR generator (`model/framework/export`) | — | Emits the new fields; drops the totals counter. |
| Checks (`model/framework/check`) | `Equivalence`, `StepLedger` over Scala step tables | They compare Go dumps; `STEPS_ONLY` retires. |
| IR admission (`tools/umpire/ir`) | `WithTotals`, `RequireTotals` | Removed. Gains `rules.go` and `structure.go`. |
| Interpretation (`tools/umpire/interp`) | `Build`, `NewInterpreter`, "existing value, table, limits" | Gains the row provider and `Explore`. `Ceilings` gives way to the reach bound. Dependencies unchanged. |
| Engine (`internal/engine`) | "Evaluate finite table claims." | The job becomes "Evaluate claims over explored graphs". It stays private to interp and check. |
| Checking (`tools/umpire/check`) | receipt and scope data | Receipt fields (§3.5), including `Checker`. `Scope.Progress` and `Ceilings` retire. |
| **new** Answering (e.g. `tools/umpire/answer`) | — | Runs `check` and the named backends through `export`, replays traces, compares answers, and emits one receipt per Query (§7.5). `lower`, `lint` and the commands read receipts from it. |
| Lowering, Conformance | "never the engine directly"; "ir, interp, check, realization" | Unchanged dependencies. Conformance reads the provider through check. |
| Lint | "ir, interp and check only" | New findings (§4.5); dependencies unchanged. |
| Export (`tools/umpire/export`) | "Compare Quint's reading with the reader." | Its job becomes "Export a Model to a backend, and run that backend to decide or confirm a Query" (D3). The VISION amendment of §7.5 is required. |

---

## 10. Phased plan and decisions

### 10.1 Decisions

All **settled (owner, 2026-10-13)**.

| # | Decision | Where it lands |
| --- | --- | --- |
| D1 | Open graphs allowed per Query with explicit acceptance: the `open-graph` lint fails the gate until accepted with a reason; a Temporal `verify` needs a closed region unless `.open(because = ...)` | §3.3, §4.5; Phase 1 |
| D2 | The Behavior Fingerprint is an **IR structural hash only**: behavior-affecting IR without positions, origins, choice names, totals or `because`, canonicalized against source order, modelled on `conformance.go:158-172`. No reachable-graph hash. The machine covers its own declarations ("own starts only"); Scenario declarations and starts go into the Scenario and Query fingerprints. | §4.13, §5.5; Phase 1 |
| D3 | Backends may decide always: any Query may be answered by Umpire, Apalache or TLC | §7.3–§7.5; receipt field in Phase 1, the rest in Phase 6 |
| D4 | Go only: no Scala catalog walk and no Scala explorer | §2.1, §4.11; Phase 2 |
| D5a | Native partial `Map` only. `m(k)` on a missing key is a hole with a path; `getOrElse` and `get` are the safe reads. No `Total`. | §6.5; Phases 3–4 |
| D5b | Keep `Inbox` now; re-express channels as a library over `List`/`Set` later | §4.9, §7.3; before a backend decides a channel model |
| D5c | Replace the four split-on-`-` sites with structured lookups | §6.3; Phase 3 |
| D6 | Checked `int64`; overflow is a hole reported `incomplete` | §6.4, §7.4; Phase 3 |
| D7 | Reach bound per machine (`val reach = ...`), with a feature-wide default in `temporal/Bounds.scala` | §3.3; Phase 1 |
| D8 | Query totals retire everywhere; explored counts replace them | §4.6; Phase 2 |

The governance amendments under GOV-02 are:

- the SPEC's "finite types" (`.plans/UMPIRE4_SPEC.md:42`);
- the SPEC's Behavior Fingerprint definition (§4.13);
- the Front-end concept's reproduction requirement (§4.13);
- VISION's "One meaning, one source" (§7.5).

### 10.2 Phases

Every phase leaves `make umpire-check-model` green.

#### Phase 1: the reachable graph becomes the only table

**Go:**

- `tools/umpire/interp/machine.go`: `rows` moves to the test oracle, `Preflight` is removed,
  `startsAndEnds` reads reached states, and `result` uses `Conforms`.
- New `interp/successors.go` (`RowsFrom`, `Explore`).
- `interp/limits.go`: `Size` and `Ceilings` give way to the reach bound.
- `engine/table.go` (explored `States`/`Rows`); `engine/search.go`, `progress.go`, `compose.go`,
  `refine.go`, `lower.go`.
- `engine/canonical.go` and new `ir/structure.go`: the v2 fingerprint is the IR structural hash of
  the machine's own declarations; Scenario declarations and starts go in `ScenarioSemantic` (§4.13,
  §5.5).
- `check/claims.go`, `check/checking.go` (roots, replay on demand; the receipt fields `Reach`,
  `Reached`, `Closed` and `Checker`, the last always `umpire` until Phase 6).
- `conformance/plan.go`.
- `lint/kinds.go`, `holes.go`, `bindings.go`, `render.go` (`open-graph`, roots).

**IR:**

- `machine.proto` (`Machine.reach`, `StepBinding.rules`);
- the lifter's emission of both (`Declarations.scala`, `Machine.scala` `reach` default);
- the dynamic overlap check at reached states.

**Parity:**

- `rows` moves to `interp/eager_test.go`.
- New `interp/parity_test.go` cases. For every machine of every checked-in IR and every sample:
  - the explored rows equal the eager rows restricted to reachable states;
  - every receipt (kind, witness, `Explored`) is identical under both;
  - every generated Case is byte-identical under the equivalence projection that strips
    fingerprints (`Equivalence.scala:374-388`).

  A Case that differs beyond fingerprints, the `fixedRequirements` question (open question 5), stops
  the phase until it is understood.

**Then:**

- Regenerate every Case: `make umpire-gen-model`, `umpire-gen-fixtures`, `canary-gen-case`.
- Update the fingerprint goldens (§9.3).
- Drop Voting's `inv` from both effects and from `UnsafeVoting`. The samples check
  (`model/framework/check/Samples.scala:37-64`) must still expect the same kinds.

**Tests:**

- `engine/refinement_test.go`: a violating *unreachable* row passes; a reachable one is rejected
  with a path.
- `check/refinement_test.go`: a refinement-map hole at an unreachable state no longer fails.
- Overlap at an unreachable state is accepted; at a reachable state it gives `overlapping-rules`
  with a path.
- A Scenario start outside the machine region is explored and reported.

#### Phase 2: Scala retires; Go static checks

**Scala, removed:**

- `model/framework/dsl/{Machine,Syntax}.scala` (`overlap` and its callers);
- `Refine.scala` (`unclosed`);
- `StepTables.scala`;
- `MachineLaws.test.scala`'s catalog walks;
- the totals counter (`Capabilities.scala:1246-1308`, `Lift.scala:153-161`) and `Claims.scala`'s
  `total`.

**Checks:**

- `model/framework/check/equivalence/*` compares Go dumps.
- The table pins' dumper (`model/temporal/check/table-pins/README.md`) moves to reachable dumps.
- `model/temporal/check/Coverage.scala` re-captures its baseline.
- `ChangeCheck.scala:230-246` runs after its lift.

**Go:**

- `ir/rules.go` (§5.1);
- `ir/structure.go` (§5.5);
- the exhaustiveness and map-coverage checks (§5.2), with `Match.scrutinee_type`, `Case.roles` and
  `Refinement.invariant` in the proto;
- `lint/growth.go` (§5.3);
- `ir/totals.go` removed.

**Docs:** SEMANTICS §6, §7, §9, §13, §20; README "Sealing a batch" and "Counting a Query's total".

**Tests:**

- `ir/rules_test.go`: phase-disjoint, role-disjoint and class-disjoint pairs pass; a same-phase
  pair with no `where` is refused statically; a `where` pair defers.
- Exhaustiveness fixtures, including a declared-hole arm.
- An `unmapped-phase` fixture.
- Role closedness against `ActivityHeartbeatRegression.test.scala:149`'s current expectation.
- `Equivalence.test.scala`, `InterpreterTables.test.scala`.

#### Phase 3: IR and interpreter collections, unbounded `Int`

- **Files:**
  - `value.proto`, `expression.proto`, `action.proto` (input `domain`), then `make proto` and the
    ScalaPB jar;
  - `ir/validate.go`, `validate_keys.go`;
  - `interp/{eval.go,channels.go}` and new `interp/collections.go`;
  - the four key splits (§6.3);
  - `lint/kinds.go`.
- **Tests:** `interp/collections_test.go` (every op against its desugared chain; type order against
  catalog order on finite types); `ir/admission_test.go`; `interp/rowkeys_test.go` (negative
  integers); overflow as `incomplete`.

#### Phase 4: DSL and lifter

- **Files:**
  - `model/framework/dsl/{Domain,Machine,Action,Channel,Monitor}.scala` (`Data`, `oneOf`);
  - `model/framework/export/{Types,Expressions}.scala`;
  - `model/framework/inspect/LiftableSubset.scala`;
  - `model/framework/lint/Conventions.scala`;
  - SEMANTICS §1, §3, §5, §18, §20, §21.
- **Tests:** `model/framework/export/testdata/lifts/Collections.scala` with one `Fixtures.test.scala`
  case per liftable row; refusal controls in `model/framework/lint/LiftableSubset.test.scala`;
  `dsl/Equality.test.scala`; `dsl/Inputs.test.scala`.

#### Phase 5: native samples

- **Files:** the six samples (Bank, TwoPhaseCommit and Voting first; then Kettle with a signed
  temperature, TwoLayeredCache with Quint's history, ReliableBroadcast); `model/README.md:640-648`.
- **Tests:** `Samples.scala` expectations unchanged. A reachable-graph isomorphism test in
  `model/framework/check/Samples.test.scala`, with the old encodings kept as fixtures and an explicit
  state map.

#### Phase 6: backends that decide

**Already in place from earlier phases:** the receipt's `Checker` field (Phase 1), and the
structural hash that a backend's receipt also binds to (Phase 1).

**6a. Export coverage.** These are the ingredients §7.3's table lists as missing:

- Scenarios and Limits;
- Properties as Apalache invariants;
- `find` as a negated invariant;
- refinement over `hist`;
- progress deadlines;
- native compositions;
- channel export (or the D5b library, if it lands first).

Native collections in the Quint writer (§7.2) also land here. Files:
`tools/umpire/export/{quint.go,itf.go,composed.go,README.md}`.

**6b. Deciding.**

- **IR.** `Query.checker` and `confirm` in `claim.proto`.
- **DSL.** The `checkedBy` builder in `model/framework/dsl/Claims.scala`, lifted in
  `model/framework/export/Claims.scala`.
- **Admission.** Admission refuses a `checkedBy` whose ingredients are not covered.
- **Orchestration.** A new module (for example `tools/umpire/answer`) runs the named checker,
  replays traces through Go (`export/slice.go:627`), compares answers (§7.4) and emits receipts.
- **Consumers.** `lower` and `lint` switch to its receipts (`lower/lower.go:103-117`;
  `lint/lint.go:147-169`).
- **Docs.** `.plans/UMPIRE4_VISION.md` and `.plans/UMPIRE_MODULES.md`, after GOV-02.

**6c. The TLA+/TLC path** chosen under open question 13. `TestQuintOwnsBackendExport`'s contract is
updated with it.

**Tests:**

- `#set` and `#map` ITF cases.
- For a native sample, the same Query decided by each checker: the answers agree, and each witness
  replays.
- A backend trace that does not replay gives `witness-rejected`.
- `checkedBy apalache` on a channel model or a holed region is refused at admission.
- Graph-size agreement between Umpire and TLC on a closed region.
- An integer past 2^63 gives Go's `incomplete`, reported as incomparable.

---

## 11. Open questions

Every design decision is settled (§10.1). What remains is factual, or is a choice of route:

1. **Do all Temporal Models close** under a modest reach bound? They should, since their catalogs
   are finite today. Phase 1's receipts will show it. **(unverified)**
2. **Reachable counts for the native encodings.** §12 measures today's encodings: TwoPhaseCommit
   288, Voting 599, Bank 100. Native encodings with `oneOf` inputs should reach the same states.
   **(unverified)**
3. **Scala 3 `CanEqual`** for `Map`, `Set` and `List` under `-language:strictEquality`, alongside
   `Data`'s. **(unverified)**
4. **The Scala exhaustivity warning.** Is it fatal in the model build (§5.2)? **(unverified)**
5. **`fixedRequirements` over reached states** (`lower.go:108-164`): does any checked-in Case change
   beyond fingerprints? If so, define fixedness by the predicate's shape. **(unverified)**
6. **`StepLedger.reorder`'s assumptions** beyond per-action contiguity (`StepLedger.scala:370-410`).
   **(unverified)**
7. **Quint 0.33.0 built-ins:** map insert (`put`, or `set` on an absent key), `fold` over sets,
   inductive invariants and fairness in `quint verify`. **(unverified)**
8. **The fold commutativity rule** (§6.5): the exact admitted bodies.
9. **The hash's field allowlist** (§5.5): its exact contents, and a test that adding an IR metadata
   field leaves every existing fingerprint unchanged.
10. **IR version:** keep `version` 0 for additive fields (`validate.go:27-30`), or bump it.
11. **Kettle's `coldest` Scenario:** is it a `detached-root` (§5.6)? **(unverified)**
12. **Growth analysis precision:** how many false `unbounded-growth` warnings the Temporal Models
    give (§5.3). **(unverified)**
13. **The TLA+/TLC route** (D3 requires one). The options are `quint compile --target tlaplus`
    **(unverified for 0.33.0)**, or a direct TLA+ writer as a second exporter. The choice changes
    `TestQuintOwnsBackendExport` (`.plans/UMPIRE_MODULES.md:364-368`).
14. **TLC's integer range** relative to Go's checked `int64` (D6). It is believed narrower, which
    would make TLC-decided Queries over large integers incomparable as well. **(unverified)**
15. **Fairness in backends.** Apalache, via Quint, may lack weak fairness, which would keep fair-cycle
    progress claims Umpire's or TLC's alone until it is supported (§7.6). **(unverified)**

---

## 12. Performance and the feedback loop

The owner asks whether the full-catalog design is why Umpire's feedback cycle is slow. Partly. It
dominates the Go check of the samples. Most of the wall time, though, is Scala toolchain overhead
that does not depend on this design.

### 12.1 What was measured

All figures are from this machine on 2026-10-13, with warm caches.

The coordinator measured the Go check (`check.Check` on already-lifted IR) and the Scala steps:

| Sample | Catalog states | Go check |
| --- | --- | --- |
| TwoLayeredCache | 64 | 0.01 s |
| Kettle | 564 | 0.7 s |
| ReliableBroadcast | 30,870 | 5.5 s |
| TwoPhaseCommit | 49,152 | 10.6 s |
| Bank | 6,561 | 48.6 s |
| Voting | 19,683 | 52.0 s |
| **Total** | | **≈117 s** |

| Scala step | Time |
| --- | --- |
| `scala-cli package` of framework + samples | 32.6 s |
| `scala-cli compile --print-class-path` (same inputs) | 12.4 s |
| Lift Kettle (one JVM) | 40.7 s |
| Lift Voting (one JVM) | 30.5 s |

`make model-check-changed` took 19 minutes of wall time. Of that:

- lifter suites: 560 s;
- samples stage: 317 s;
- check suites: 67 s.

I added the reachable counts below. They were read from the same IR files
(`/tmp/kettle-check/out/*.check.json`) with `interp.NewInterpreter(model).Interpret(model)`, using
`Work` and `len(Table.Reachable)`, in a throwaway program outside the tree.

`Table.Reachable` is today's unbounded fixpoint from the starts (`engine/table.go:266-291`). It is
exactly the set a closed reachable exploration would materialize, minus any extra Scenario roots
(§3.3). These counts are for today's encodings, not for §8's native rewrites.

| Sample | Machine | Catalog states | Classes | Evaluations today | Reachable states | Reachable evaluations | Share |
| --- | --- | --- | --- | --- | --- | --- | --- |
| TwoLayeredCache | model / writeThroughL1 | 64 | 3 | 192 each | 7 each | 21 each | 11% |
| Kettle | model | 564 | 4 | 2,256 | 163 | 652 | 29% |
| ReliableBroadcast | model | 30,870 | 12 | 370,440 | 125 | 1,500 | 0.40% |
| | naive | 30,870 | 12 | 370,440 | 21 | 252 | 0.07% |
| TwoPhaseCommit | model | 49,152 | 14 | 688,128 | 288 | 4,032 | 0.59% |
| | eagerCommit | 49,152 | 14 | 688,128 | 498 | 6,972 | 1.0% |
| Bank | model | 6,561 | 128 | 839,808 | 100 | 12,800 | 1.5% |
| | leaky | 6,561 | 128 | 839,808 | 729 | 93,312 | 11.1% |
| Voting | model | 19,683 | 18 | 354,294 | 599 | 10,782 | 3.0% |
| | unsafeVoting | 19,683 | 18 | 354,294 | 1,021 | 18,378 | 5.2% |
| | consensus | 3 | 12 | 36 | 3 | 36 | 100% |

Bank's 100 confirms §8.1's hand count.

### 12.2 Where the Go check's time goes

**Table building.** The Go check's cost tracks evaluations, which are catalog × classes, not
reachability. Bank and Voting have the smallest catalogs of the four big samples, but the most
evaluations and the costliest guards. They take 86% of the 117 s.

**A second full interpretation.** `check.Check` interprets the Model twice:

- once to bind (`check/claims.go:44-54`);
- once more for a fresh binding through which witnesses are replayed (`checker.again`,
  `tools/umpire/check/checking.go:370-375`; `replayQuery`, `checking.go:678`).

**A CPU profile of the Bank check** confirms both points. It was taken under load, so its wall time
was 92 s, not 48.6 s.

- **Interpretation.** Of the main goroutine's 46.7 s, 35.6 s is `interp.(*Interpreter).Interpret`,
  and 22.2 s of that is the replay's second interpretation (`checker.again`).
- **Garbage collection.** The 840k-row tables keep `Value` trees live, and GC mark workers used
  98 s of CPU on other cores, 58% of all samples.
- **What remains.** The rest of the main goroutine (≈11 s) is search, fingerprinting and receipts.

### 12.3 What reachable exploration would save

**Estimate.** Scaling each sample's Go check by its reachable share of evaluations:

| Sample | Share | Today | After (estimate) |
| --- | --- | --- | --- |
| Bank | 6.3% | 48.6 s | ≈3 s |
| Voting | 4.1% | 52.0 s | ≈2 s |
| TwoPhaseCommit | 0.8% | 10.6 s | ≈0.1 s |
| ReliableBroadcast | 0.24% | 5.5 s | ≈0.01 s |
| Kettle | 29% | 0.7 s | ≈0.2 s |
| TwoLayeredCache | 11% | 0.01 s | ≈0 s |
| **Total** | | **≈117 s** | **≈5–6 s** |

The share for each sample sums its machines from the table in §12.1.

**Assumptions behind the estimate:**

- *Linear scaling.* GC cost grows with heap size, so the saving is likely at least linear.
- *Search is unchanged.* The Query search, bounded by Limits, keeps its cost.
- *Replay is cheaper.* The replay's second interpretation only needs the witness's rows on demand,
  so it should fall to near zero.

Phase 1 of §10 measures this directly. The figures above are **estimates**.

### 12.4 The Scala-side catalog walk

The rule-disjointness check runs when each rules object is constructed, inside the lift
(`IrFile.scala:86-92` forces `machine.bindings`). It walks the state catalog only for a pair of rules
of *one action* that both fire a common class (`Machine.scala:338-341`). Its body is
`earlier.decl == added.decl`, then `earlier.fires(inputs) && added.fires(inputs)`, and only then
`states.values`.

None of the six samples has such a pair:

- Bank, Voting, Kettle and TwoLayeredCache bind one rule per action.
- TwoPhaseCommit's and ReliableBroadcast's per-class rules fire disjoint classes
  (`TwoPhaseCommit.scala:209-244`; `ReliableBroadcast.scala:208-215`).

So the walk's share of the samples' lift time is effectively zero. The lift computes totals in
closed form (`Capabilities.scala:1294-1308`), and step tables are not run in the samples stage. The
measurements agree: Kettle (564 states) lifts in 40.7 s and Voting (19,683 states) in 30.5 s.

The lift is fixed JVM, scala-cli and TASTy-indexing overhead. The walk can matter for Temporal
features that bind several `when(Phase…)` rules to one action, at a cost of states × classes ×
earlier rules. That is **not measured**. Under this basis the walk is removed (D4); §5.1's
structural check and the dynamic check at reached states replace it.

### 12.5 The samples stage, split

| Part | Time | Share of 317 s |
| --- | --- | --- |
| Package + compile (twice over the same inputs) | 45 s | 14% |
| Go check | ≈117 s | 37% |
| Six lift JVMs (remainder) | ≈155 s | 49% |

Reachable exploration removes about a third of this stage. The other two thirds, and all of the
560 s of lifter suites and 67 s of check suites, are toolchain.

### 12.6 Toolchain fixes that do not depend on reachable exploration

1. **A warm compile server.** Every scala-cli call passes `--server=false`, so nothing compiles
   incrementally and every call starts cold:
   - `model/framework/check/Samples.scala:136, 143, 153`;
   - `ChangeCheck.scala:260` (every suite run, the 560 s and 67 s stages);
   - `Makefile:766-767, 779, 790, 794, 799, 811`;
   - `model/temporal/check/Gate.scala:510, 1079, 1088, 1096, 1110`.

   A Bloop server would make re-runs incremental. `Gate.test.scala:1228`'s fake scala-cli requires
   the flag, so that test changes too. Why the flag was chosen (determinism, sandboxing) is not
   documented in what I read **(unverified)**.
2. **Compile once, not twice.** `Samples.scala:134-145` runs `package` and then
   `compile --print-class-path` over the same inputs, and `Makefile:766-767` repeats the pattern for
   `$(MODEL_JAR)`. Without a server, the second call recompiles: that is 12.4 s.
3. **One lift run for all samples.** `Samples.scala:146-162` runs
   `scala-cli run model/framework/export … framework.export.lift` once per sample. Each run
   re-resolves the export project, starts a JVM and re-indexes the jar. The lifter already has a
   multi-output mode, `lift --ir <jar=prefix> <classpath> <directory> [name...]`, which "reads the
   compiled Models once and lifts every file apart" (`model/README.md:313-316`; used by
   `Makefile:779`). Either of two changes would let the samples use it:
   - each sample declares an IR file;
   - `lift` gains a multi-root, multi-output form.

   That saves five of six lift start-ups, ≈100–130 s (*estimate*).
4. **A packaged lifter.** Run the lifter from a packaged jar with `java -cp`, not
   `scala-cli run model/framework/export`, both here and at `Makefile:779`.
5. **Cache the samples jar** by source hash. `$(MODEL_JAR)` is already Make-cached on its sources
   (`Makefile:764`). The samples stage repackages the framework every time (`Samples.scala:134-138`).
6. **Build the Go checker once and check samples in parallel.** `Samples.scala:166-176` writes the
   runner and `go run`s it, compiling it every time. Its loop (`Samples.scala:28-80`) checks one
   sample after another.
7. **Keep fewer full tables in memory, until Phase 1 lands.** Independent of on-demand rows, the replay binding could
   share the first binding's tables where the replay only needs to re-evaluate witness rows. That
   halves today's interpretation (§12.2). This touches what "a second interpretation" is meant to
   independently confirm (SEMANTICS §20, `replay-failed`), so it is a semantic decision, not only a
   toolchain one.
