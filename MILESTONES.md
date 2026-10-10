# Umpire4 milestones

The current state of Umpire work: what is being built, what is left, and what is not being done.
Flow (`.flow/`, `flowctl`) is the record for specs and tasks; this page is the overview across them.

As of 2026-10-10.

## Keeping this page current

- This page describes the present. Rewrite a status in place; do not append dated entries.
- List each open spec's tasks with ID, status (✅ done, 🔄 in progress, ⬜ todo, ⏸️ deferred) and a brief
  description; set a task's status in place
  when it changes. Keep completed tasks listed until the whole spec is complete, then remove the spec.
  Flow and git keep the history.
- Group scheduled specs under the batch that shares their regeneration (see Batches). Task rows
  record the task's own evidence; the batch-deferred checks are implied and need not be repeated.
- Only the root agent edits this page. Workers report status in their handoffs, so isolated
  worktrees never carry a conflicting copy of it into a join.
- Close a cancelled or abandoned spec in Flow (tasks blocked, a "Closed: won't do" note in the
  spec) and remove it from this page.
- Update the "As of" date with every edit.

## Verification instructions for agents

Apply these instructions when implementing the milestones:

- The root agent owns scheduling, dependency decisions, integration and Flow status. Sub-agents own
  task-local investigation, implementation, testing and fixes; independent reviewers assess the
  work. Workers return concise handoffs linking commits, verification evidence and remaining
  blockers. Root inspects the evidence needed for acceptance without importing entire task
  histories. Resume the same worker for fixes where possible.
- A blocked task holds its dependents, not unrelated work. Continue at the safe ready frontier,
  using isolated worktrees and the workflow's ownership, dependency and heavy-command constraints.
- A completed predecessor satisfies its dependents once its Flow closure is committed in the
  current local branch's history. Publishing or landing on origin is not required. Uncommitted
  closures and work on unmerged branches do not count; dependent work starts from the integrated
  local baseline. Publishing is a separate, explicitly authorized action.
- If a validation remains stuck for more than one hour, including repeated attempts, mark it
  deferred in Flow and this page and move to work that can be verified and delivered. Continue
  beyond that limit only when the validation blocks every other available work path. Record its
  command, elapsed time, failure evidence and condition for revisiting it. Preserve the failed
  result and follow-up obligation; deferral is not a passing gate.
- Reuse the previous task's passing baseline when its commands, source scope, fixtures and
  environment still apply. Inspect its recorded evidence; a new task or agent is not a reason to
  rerun it. Changes to relevant inputs invalidate the affected results.
- During implementation, run the smallest tests that exercise the changed behavior and its failure
  modes. Once ready for review, run the task's required full tests, goldens, lint and dependency
  checks once. After a fix, repeat affected checks; repeat broader gates only when the change or
  failure invalidates their results. Preserve all required coverage and acceptance criteria.
- Give reviewers the source scope, commands, results and log paths from that run. Reviewers inspect
  this evidence and request an additional check only for a concrete unresolved concern. Resume the
  same review after fixes, with the changed code and relevant new results.
- Measure the next already-required full Go test run with `-json`, retaining its exit status and
  output in the task's `.flow/tmp/` directory. Use test completion events to identify slow tests;
  package times overlap and must not be summed as wall time. Record elapsed wall time separately.
  Keep a running process running; add instrumentation to the next run instead of restarting it.
- Optimize measured bottlenecks, starting with duplicated model construction if the timings
  implicate it. Preserve independent assertions and immutable fixtures; shared
  test setup must not leak mutable state. Compare timings on equivalent inputs before claiming a
  speedup. Do not add a profiling or caching framework without evidence it is needed.
- Commands that hold: run the full Go tooling suite with `-tags test_dep -p 2 -timeout 30m` (the
  lower, model and export test binaries are memory-heavy, so never more than
  two at once, and `-p 1` doubles wall time); run the model gate with `MODEL_GATE_ARGS=--skip-go-checks` when the Go
  suite runs separately, since its Go phase repeats it. Agents sharing one machine serialize heavy
  suites with one `flock` lock file. After any Model or lifter change, run `make umpire-gen-model` (inside a batch, only at the batch's regeneration; see Batches),
  review the diff of `model/ir` and `model/cases`, and run `make umpire-check-cases`: the reader's
  tests over `model/ir` pin what a Model means, and the managed Case trees what its Cases contain.
- Keep handovers concise and link existing evidence. Add audits, inventories or verification gates
  only for an explicit requirement or a concrete uncovered risk. Move to the next implementation
  task once the required checks and review pass.

## Batches

A batch is a run of specs that share one baseline: one production regeneration, one full gate run
(Go suite, model gate, goldens, lint, dependency checks), one independent review and, when the batch
changes Cases or runtime behavior, one live run. Batching buys throughput with attribution, so a
batch boundary falls where a failure must stay attributable: after a meaning-preserving refactor's
equivalence proof that later work builds on, and at a format activation.

- Inside a batch, tasks run focused source, lowering and runtime checks and declare their IR change;
  production artifacts, full gates, review and live evidence are batch-deferred.
- A meaning-preserving spec seals its comparison (a scratch regeneration whose diff is exactly the
  stated mapping) before a later spec in the same batch changes meaning. The seal is focused evidence,
  not a full gate.
- Shared close: the batch's final close task runs the regeneration, gates, review and live run once;
  the other close tasks finish their own code and docs and link that evidence. No spec in a batch
  closes before the batch boundary.
- Flow cannot express cross-spec task edges, so the conductor enforces the listed source gates
  without spec-close dependencies inside a batch.
- Batches run serially; Flow records that as each spec depending on every spec of the previous
  batch. The owner prioritized the Activity subject split (fn-151) immediately after batch 1,
  followed by naming the Activity's repeated patterns (fn-155), then compiler and lint enforcement
  (fn-156); authoring batch 2 waits on all three.
  The structural baseline is closed; batch 5 waits on batch 4.
  The next batch's preparation may run in isolated worktrees alongside the
  current one, but joins serially onto the closed baseline.

| # | Batch | Specs | Nature | Shared close | Live run |
| --- | --- | --- | --- | --- | --- |
| 1b | Activity abstractions | fn-155 (named repeated patterns) | meaning-preserving, mapped identities | fn-155.6 | none |
| 1c | Compiler and lint | fn-156 | byte-identical | fn-156.7 | none |
| 2 | Authoring | fn-140 → fn-149 → fn-123 | two equivalence seals, then meaning | fn-140.6, fn-149.5, fn-123.8 | once |
| 3 | Testpilot format | fn-146 → fn-147 → fn-148 | breaking format | fn-146.7, fn-147.4, fn-148.7 | once |
| 4 | Lifter | fn-141 | byte-identical | fn-141.14 | none |
| 5 | Activity model | fn-128 → fn-138 → fn-129 | meaning | fn-128.6, fn-129.5 | once |

## Direction

The Scala model (`model/`) is the model. Lean is the past, and nothing has to look like it.
Scala declares, a lifter reads the declarations into the Umpire IR, and generic Go consumers check
that IR, lower it to Testpilot Cases, and run those Cases as functional tests and canary checks. The
Umpire IR and the Testpilot IR are what connect the parts. See [SCALA.md](.plans/SCALA.md) and
[UMPIRE4_SPEC.md](.plans/UMPIRE4_SPEC.md).

The DSL framework (`model/framework`) stays Temporal-agnostic as far as is realistic: Temporal's capability
vocabulary, properties, realization vocabulary and kit live under `model/temporal/`, and the lifter and Testpilot IR
are the parts that are Temporal's driver tooling by design (fn-114.12, fn-122.8).

Next, name the split standalone Activity's repeated patterns, then complete compiler and lint
enforcement; continue the Scala authoring and format work before completing Activity coverage.

## Specs

Listed in delivery order, grouped by batch. Flow records spec and task dependencies; the conductor
also holds batch-close gates and serializes work that shares a regeneration baseline.


### Batch 1b, Activity abstractions: fn-155

#### fn-155: Name the standalone activity's repeated patterns

[Spec](.flow/specs/fn-155-name-the-standalone-activitys-repeated.md) runs on fn-151's closed
baseline, before fn-140/fn-123, so the subject-split files are refactored once. Fn-151 is closed and
committed on the integrated local baseline with whole-spec review SHIP; that satisfies its
dependency without publication. It names the
held attempt's ending (effects that differ only in landing phase collapse through a landing function;
named guards; every deadline rule reads the `armed` predicate its capability declares), moves Product
single-fact steps onto its `Recorded` effects, gives the six reset-aware overrides one shared reason,
and removes smaller duplication in the machines, Dispatch models and Realization.scala. The baseline
step table and an effect-name-erasing IR projection are the equivalence proof. Items needing a framework or lifter change
(capability preemption, typed `overriding`, enum-case product readings, generated per-policy starts,
splatted scenario prefixes) are deferred to fn-138/fn-141 or later. It moves no models between
files and changes no behavior; renamed effect and Property identities are mapped and checked at
one regeneration. Task 1 captures the baseline and builds the equivalence projection; tasks 2, 4 and 5
are parallel candidates after it; task 3 follows task 2; task 6 regenerates and closes. Open for the
owner: whether fn-129.3's open remainder lands before or re-anchors after this spec's reset changes,
and the pending-control remodel.

| Task | Status | What |
| --- | --- | --- |
| fn-155.1 | ✅ done | Sealed complete baseline and projection; 46 native/Pins tables, semantic mutation controls, lifter fallbacks; three review lanes SHIP |
| fn-155.2 | ✅ done | System landing/guards/reset refactor; all 46 tables/13,891,948 records equal, explicit predicate proof, three-axis SHIP and post-review 65 Scala tests/lint pass |
| fn-155.3 | ✅ done | Product Recorded effects/shared reasons/examples; 20 fresh Cases byte-exact, complete guarded structural proof, three-axis SHIP and integrated Activity tests/lint pass |
| fn-155.4 | ✅ done | Proven R5 fallbacks retained; shared waiver reasons; 23 complete tables/1,361,467 records equal, three-axis SHIP and integrated lint/Scala checks pass |
| fn-155.5 | ✅ done | Realization evidence builders; reviewed twelve-leaf provenance proof, three-axis SHIP, integrated 24 realization/lowering tests and lint pass |
| fn-155.6 | ⏸️ deferred | Canonical regeneration/validation exceeds the one-hour rule; scoped fixtures, tests and mapping preserved; complete R7/all31/whole-manifest proof and gates remain required |

Final validation remains deferred, not passed. Canonical generation attempts retain at least
2,380 executed seconds and a 3h46m unresolved wall span; the fourth wrapper receipt is missing.
The focused four-test QualifiedNames retry passed unchanged, while complete scratch lowering was
killed and produced no Cases. Evidence and revisit conditions are recorded in fn-155.6's Flow
block reason and its preserved task6 worktree. The seventeen passing scoped partitions grant no
whole-artifact or closure credit. Fn-155 stays open and continues to hold fn-156's final baseline acceptance; independently
verifiable work may proceed without relaxing its complete provenance contract.
Read-only kernel attribution confirms global OOM for the scratch generator and candidate
lowering test. The host has about 16 GB RAM and no swap; victim RSS does not establish required
successful capacity. Partial checkpoint `2250d1f9ae` and its handover remain on the task6 branch,
unintegrated and unreviewed. The original failures and the later kernel excerpts are preserved.

### Batch 1c, compiler and lint: fn-156

#### fn-156: Let the Scala compiler and lint enforce Model correctness

[Spec](.flow/specs/fn-156-let-the-scala-compiler-and-lint-enforce.md) is scheduled immediately
after fn-155, before batch 2, by the owner's instruction on 2026-10-09. Its eight-task plan passed
independent review and is ready. On 2026-10-10 the owner authorized isolated implementation
ahead of fn-155's closure, in parallel with fn-146, and deferred fn-157. Task .1 pins the current
integrated fn-155.1–.5 source and complete managed raw-byte manifests before edits; fn-155.6's
missing fresh-generation proof remains a reconciliation hold, not inherited passing evidence.
The source lane runs .1 through .6 in order. Report-only .8 runs independently in an isolated
checkout of the same baseline; .7 joins both lanes for documentation and the full close.

It changes no IR or Case byte, including positions, and serializes with other Model source work.
Task .1 proves sound equality evidence and byte-preserving exhaustive-match lowering before
rollout; an infeasible proof stops that source lane without relaxing acceptance. The
compile-versus-lift items from the same review remain notes on fn-141.

| Task | Status | What |
| --- | --- | --- |
| fn-156.1 | 🔄 in progress | Isolated compiler lane: warning checks, finite equality evidence, and exhaustive-match/raw-byte compatibility proof; final fn-155 baseline reconciliation held |
| fn-156.2 | ⬜ todo | Strict equality and precise equality-lint policy across authoring, lifter, fixtures and gate tooling |
| fn-156.3 | ⬜ todo | Section order, dotted membership, permitted imports and exhaustive enum/state matches |
| fn-156.4 | ⬜ todo | Discover every exported machine for totality, closedness, relation and binding-order laws |
| fn-156.5 | ⬜ todo | Complete finite-domain tests and independent interpreter table pins for every Activity/Nexus machine |
| fn-156.6 | ⬜ todo | Explicit-nulls trial across compiler roots; adopt with boundary fixes or report findings and drop |
| fn-156.8 | ⬜ todo | Two-day isolated Draft capture-confinement spike; diagnostics, limitations and adoption-cost report |
| fn-156.7 | ⬜ todo | Join reports, document enforcement, exact artifact comparison, full gates and reviews; close |

### Batch 2, authoring: fn-140 → fn-149 → fn-123

Starts from fn-156's closed baseline, with task paths re-anchored to the completed moves,
Activity subject split, abstractions and compiler/lint enforcement. All three specs
rewrite Model declarations, so they share one production regeneration, one full gate, one review and
one live run at fn-123.8. fn-140 is meaning-preserving apart from its declared renames: fn-140.6
seals its assessment-equivalence comparison before fn-149 groups claims. Fn-149.5 seals the final
grouping, including documentation and executable-layout changes, before fn-123 changes meaning.
Each seal compares against its independently frozen predecessor; a fault change cannot hide inside
either migration.

Source gates: fn-149 starts after fn-140.6's committed witness seal; fn-123 starts after fn-149.5's
committed final grouping seal. The source seals do not close their specs or discharge shared gates.
Fn-123 consumes the grouped baseline and composed identity/provenance mapping, using fn-140's
`when` blocks and `.live` together with the shared `Outcome`, fault instruction and typed `perform`.

Fn-149's insertion follows the conductor's recommendation under the owner's instruction to finish
all milestones and choose recommendations without further questions. The three plans and their
cross-spec handoffs passed independent review; fn-149 is ready. Sharing production regeneration
does not eliminate the separate source migrations or their equivalence proofs.

#### fn-140: One-sentence witness Queries with explicit live expectations

It rewrites the Models' `properties` and `queries` sections and its R5 renames Definition
IDs. Ready, with six M-sized tasks. The three foundations run in order; task 4 and the documentation
task 5 are disjoint parallel candidates, and task 6 joins them for the equivalence seal.

`witness(<classes>).records(<fact>)` states a path-and-outcome claim in one declaration and lifts to the existing Scenario, Property and `find` Query, each named after it. `.live(<expectation>)` replaces `.expect` as the one word that generates a Case, and every non-satisfied expectation carries a reason. Pinned `find` Queries in every Model migrate, and the hand-written `terminate`, which repeats the capability-generated `terminateSettles`, is deleted. `query verify` and its triple are unchanged.

| Task | Status | What |
| --- | --- | --- |
| fn-140.1 | ⬜ todo | Typed witness builders and existing core Query values |
| fn-140.2 | ⬜ todo | Witness lifting, names, bounds, composed facts and capability duplicate refusals |
| fn-140.3 | ⬜ todo | Query `.live`; source-only non-satisfied reasons; all caller and fixture spellings |
| fn-140.4 | ⬜ todo | First Activity conversion and assessment equivalence proof; duplicate terminate removed |
| fn-140.5 | ⬜ todo | Author docs, core form and layout template witness |
| fn-140.6 | ⬜ todo | Other eligible Models, source-aware triple lint, sealed regeneration diff and equivalence proof |

#### fn-149: Safety and liveness groups for object properties

[Spec](.flow/specs/fn-149-safety-and-liveness-groups-for-object.md) has five M tasks covering all six
criteria. Its refreshed plan passed independent review on 2026-10-10; implementation is not started.
It follows fn-140's witness seal, before fault work, and shares closure at fn-123.8.

Split authored claims into `properties.safety` and `properties.liveness`, enforce declaration kinds,
and carry the distinction into existing diagnostics and reports. Liveness retains explicit bounds
and assumptions. Runtime registration and lifted discovery have independent inventory proofs;
shared capability-law counts cannot use the candidate scans as their oracle. Complete grouping
comparisons preserve behavior, check results and Case/assessment meaning, with every identity and
provenance change explained. Fn-149.5 stays pending until shared gates, review and live evidence are
linked at fn-123.8, even after its committed source seal releases fault implementation.

| Task | Status | What |
| --- | --- | --- |
| fn-149.1 | ⬜ todo | Group discovery and independent runtime/lifter registration proof |
| fn-149.2 | ⬜ todo | Declaration-kind placement checks and source-attributed refusals |
| fn-149.3 | ⬜ todo | Derived safety/liveness classification in existing reports |
| fn-149.4 | ⬜ todo | Model/capability migration with complete independent behavior and identity seal |
| fn-149.5 | ⬜ todo | Author docs and final grouping seal before fn-123; shared gate/review/live close at fn-123.8 |

Tasks .1 and .3 are parallel candidates; .2 follows .1; .4 joins .1/.2/.3; .5 joins .3/.4.
Heavy commands remain serialized. Composition progress belongs to fn-150, not this grouping change.

#### fn-123: Declare faults as the environment's actions

Ready 2026-10-07. Tasks run in order; 6 and 7 need only 5, so they are parallel candidates. Task 3
is the proof: Go derives crash rows equal to `crashDetail`'s from a durability classification, or the
work stops for the owner. Planning took defaults for five owner questions, listed in the spec's Open
Questions.

A fault is declared once with its kind, budget and whether it can be realized; a machine classifies
its fields as durable or in-memory, and Go derives the crash. The explorer half of R7 and the P clause
of R8 were dropped, since neither tool exists.

| Task | Status | What |
| --- | --- | --- |
| fn-123.1 | ⬜ todo | Fault kinds and `modelOnly` on the `fault` actor's actions; IR `Fault` record; spellings settled |
| fn-123.2 | ⬜ todo | Durability classification and `crashes(…)` in the DSL and lifter, on fixtures |
| fn-123.3 | ⬜ todo | Go derives the crash row; `SEMANTICS.md` Faults section (proof) |
| fn-123.4 | ⬜ todo | Task-queue providers converted; `fault-overridden` lint; derived storage-loss assumption |
| fn-123.5 | ⬜ todo | Budgets: `budgetedBy`, four table rules in Go, `LostStartAnswer` bound to `lossAvailable` |
| fn-123.6 | ⬜ todo | Choice-level fault performance; lowering refuses model-only faults and unperformed choices |
| fn-123.7 | ⬜ todo | Trace output and Quint export of derived crashes and budgets |
| fn-123.8 | ⬜ todo | `umpire-faults` report, docs; close (the batch's regeneration, gates and live run) |

### Batch 3, Testpilot format: fn-146 → fn-147 → fn-148

Planned 2026-10-06 from [Umpire IR schema research](.plans/UMPIRE_IR_SCHEMA_RESEARCH.md) and
[Testpilot schema research](.plans/TESTPILOT_SCHEMA_RESEARCH.md). Starts from batch 2's closed
baseline; each spec's implementation starts after its predecessor's last implementation task, and
fn-141 executes after this batch. Serialize fn-131 and any revived fn-144 or Model batch against it;
re-anchor their tasks to the schema and vocabulary left by completed work. Deferred specs do not block
this chain.

Owner decision: breaking IR changes are allowed. Producers, consumers, generated artifacts and
recorded Case/Run companions migrate together. No compatibility decoders, legacy evaluators or
parallel old field spellings are required; retired formats reject explicitly. The declaration-only
split still proves semantic and artifact equivalence, not a historical compatibility promise.

That decision lets the three specs share one breaking format: fn-146.1's migration contract is the
contract for all three, and fn-148.6 activates format 4.0 once, carrying fn-146's and fn-147's
changes. fn-146.7 and fn-147.4 retire their machinery, categorize their identity changes and write
their docs; Run companions are regenerated once, at fn-148.7, with one full gate and review. Each
spec's proof task (fn-146.4, fn-147.1, fn-148.1) still runs on its own, so a failure stays
attributable to one spec.

#### fn-146: Adopt CEL for runtime predicates and values

Testpilot owns the restricted CEL environment and descriptor-aware value adapter;
Umpire lowers symbolic realization operands into it. Finite Model expressions, `ModelValue` and
descriptor-exact `ValueType` remain separate. Formats and identities move together, without legacy
runtime paths. Tasks run in order; admission and value adaptation share edit surfaces.
On 2026-10-10 the owner authorized this Testpilot implementation lane in parallel with fn-156.
Task .1 can establish its format/identity boundary against the integrated local baseline.
Authoring predecessors remain final integration holds, especially for .6; format 4.0 activation
still belongs exclusively to fn-148.6. CEL adoption is decided; no additional prototype is planned.

| Task | Status | What |
| --- | --- | --- |
| fn-146.1 | 🔄 in progress | Isolated Testpilot lane: breaking format, deterministic canonical identity and companion migration contract; no early 4.0 activation |
| fn-146.2 | ⬜ todo | Canonical CEL AST, restricted admission, pinned engine bridge and budgets |
| fn-146.3 | ⬜ todo | Standard CEL values with authoritative descriptors, exact numbers and opaque `Any` |
| fn-146.4 | ⬜ todo | Native execution and Driver/worker values; descriptor/capture and online/offline proof |
| fn-146.5 | ⬜ todo | Contract and correlated verification, rule expansion and reference walkers |
| fn-146.6 | ⬜ todo | Umpire operand lowering, Run Event guards and conformance agreement |
| fn-146.7 | ⬜ todo | Retire custom machinery, categorize identities, docs; close (companions and full gates at fn-148.7) |

#### fn-147: Migrate elapsed-time fields to protobuf Duration

Source gate: after fn-146.6. Seven Testpilot elapsed-time fields and corresponding Umpire hints/defaults
migrate through checked whole-millisecond conversions. Absent polling interval means one read;
present positive interval means polling. Counts, logical bounds, percentages and timestamps stay.
Scalar-only singleton-oneof presence cleanup belongs here. Tasks run in order.

| Task | Status | What |
| --- | --- | --- |
| fn-147.1 | ⬜ todo | Inventory, exact conversion bounds, defaults, presence and monotonicity; proof |
| fn-147.2 | ⬜ todo | Replace schemas and migrate Scala authoring, lifting, realization admission and producers |
| fn-147.3 | ⬜ todo | Runtime consumers, polling policy and scalar presence |
| fn-147.4 | ⬜ todo | Categorize identities, docs; close (artifacts, Run companions and full gates at fn-148.7) |

#### fn-148: Consolidate Testpilot evidence and correlated state schemas

Source gate: after fn-147.3. One generalized evidence declaration replaces inline extraction. Response lifts
keep ordered first-match behavior; Run Event overlaps still fail. Contract evidence policies remain
independent. Complete states include both atom and fields; results are separate from authorized
prior-state transitions, and projection result order stays explicit. Tasks run in order.

| Task | Status | What |
| --- | --- | --- |
| fn-148.1 | ⬜ todo | Generalized evidence declarations in a leaf schema; full lift-shape proof |
| fn-148.2 | ⬜ todo | One binder and lowering path; source selection and independent Contract policies |
| fn-148.3 | ⬜ todo | Complete-state and result tables in schema/lowering; compact-size measurement |
| fn-148.4 | ⬜ todo | Normalized admission/verification, causal authorization and expanded-work ceilings |
| fn-148.5 | ⬜ todo | Derived Contract kind, fixed correlated clock and explicit support Boolean |
| fn-148.6 | ⬜ todo | Local references, cardinality, Empty markers and coordinated format 4.0 activation (the batch's single activation) |
| fn-148.7 | ⬜ todo | Artifact/companion migration, measurements, full gates and ownership docs; close (the batch's regeneration, gates and live run) |

### Batch 4, lifter: fn-141

#### fn-141: Shrink the IR generator: one description of each DSL construct

Ready 2026-10-08. The delivery order puts this spec after batch 3 closes and before the activity
batch, against the settled schema and the vocabulary left by fn-140, fn-123 and fn-145 through fn-148.
Because its tasks were planned against the earlier tree, each re-reads its files and recounts first.
Tasks follow Flow's branching dependencies. Tasks 1 and 2 have overlapping declared files and run
serially; after both finish, 3, 6 and 7 are disjoint parallel candidates. Task 5 is the proof for the
export part and task 8 its size proof; either can stop tasks 9 to 13, and tasks 1 to 4 stand without
them. Source: a full read of `model/irgen` on 2026-10-06 (8,242 lines, of which about 10% lifts
function bodies and types and 1,479 are lints that emit no IR).

Every step leaves `model/ir` and `model/cases` byte-identical, so each task's gate is that
byte-identity plus the lifter's own tests; the Go suite's baseline from batch 3 stays valid while the
IR is unchanged, and runs once more at fn-141.14.

Each DSL construct is described once. Four parts, in this order: the declaration-order lint, the
structure lint and the marker checks leave `model/irgen` for their own gate step; spellings no Model
or kit uses are retired; a sugar's definition becomes its only lowering, so the lifter knows core
constructs and types alone; and declarations are exported from the constructed Models, one kind at
a time with realizations first, so the lifter lifts function bodies and types only. Declaration-level
Scala becomes free and function bodies stay in the liftable subset, so the IR schema, the Go consumers
and the Quint export do not change, and every step leaves `model/ir` and `model/cases` byte-identical.
It revises `.plans/DSL_OPERATORS.md` rule 5 and fn-113 R15: the framework uses `inline` and macros at
its capture points only. Proof point: the realization step stops the work unless it removes at least
half of the realization lifter's lines net of what it adds.

| Task | Status | What |
| --- | --- | --- |
| fn-141.1 | ⬜ todo | Order lint, structure lint and marker checks out of `model/irgen` into their own gate step |
| fn-141.2 | ⬜ todo | Spellings no Model or kit uses retired, after a recount the owner confirms |
| fn-141.3 | ⬜ todo | Generic sugar expansion; `enter`, `stay`, `reject`, `disabled`, `in`, `implies`, `records` lifted from their definitions (proof for the sugar part) |
| fn-141.4 | ⬜ todo | Claim patterns and sticky monitors by definition; lifter's sugar file gone; syntax lint holds the lifter to core |
| fn-141.5 | ⬜ todo | Capture points (name, ID, position; function span and captured values) and lifting a function by span (proof for the export part) |
| fn-141.6 | ⬜ todo | Refusal ledger: one outcome per refusal kind (deleted, kept and where, left to Go) |
| fn-141.7 | ⬜ todo | Realization factories keep everything they are given |
| fn-141.8 | ⬜ todo | Exporter; realizations exported; two-run determinism; init-order refusal; size proof with stop |
| fn-141.9 | ⬜ todo | Signature exported: actions, inputs, channels, assumptions, holes, Limits; `codeOf` gone |
| fn-141.10 | ⬜ todo | Machines exported: header, rules, monitors, refinement, derivations |
| fn-141.11 | ⬜ todo | Compositions and syncs exported; `sync` and `replaces` record what they pair |
| fn-141.12 | ⬜ todo | Properties, Scenarios, Queries and progress exported; the lifter's fold deleted; comprehension fixture |
| fn-141.13 | ⬜ todo | Capability expansions exported, in the shape the current framework leaves |
| fn-141.14 | ⬜ todo | Dead lifter code, rules of record, docs, backends check, size report; close |

### Batch 5, activity model: fn-128 → fn-138 → fn-129

Moved last on 2026-10-08 at the owner's request: starts after batch 4 closes. Its done and joined
tasks (fn-128.1–.5, fn-138, fn-129.1–.2 and fn-129.3's joined source) stay in the tree; the remaining tasks re-read their files
and re-anchor to the moves, witness syntax, fault declarations, schema and Testpilot format left by
batches 1 to 4. One regeneration, one gate run, and one live run serve fn-128.6 and fn-129.5.
Source: `.plans/ACTIVITY_MODEL_COMPARISON.md`. Each task declares its IR change, which is checked at
the batch regeneration.

Approved source gates: fn-138 implementation starts after fn-128.5 is done; fn-129 implementation
starts after fn-138.3 is done. fn-128.6 and fn-129.5 share the regeneration, review and live-run
evidence; no activity spec closes before that boundary.
Activity-batch regressions found while preparing the structural moves (schema ledger, framework names, source positions,
stale lifter fixtures, the one-bringer Cancelable capability) are fixed on `umpire`; Cancelable is retired
and `cancelIsRequested` is Nexus's own claim. The structural baseline now publishes the joined Activity
declarations; remaining tasks re-anchor again after the authoring and format batches.
Current-source replay exposes three inherited assessment failures that must be resolved before
closure: ordinary worker completion is observationally ambiguous with by-ID completion
(`completes` is `inconclusive(never_evaluated)`), the non-retryable failure claim has a visibility
ambiguity, and pause/resume exceeds its correlated per-event work ceiling. Batch 1's full Go gate
retains failure evidence for all three; its structural equivalence proof does not discharge them. None is waived or
counted as passing evidence. Tasks fn-128.7 and .8 own the latter two corrections; the completion
ambiguity has no scheduled correction yet.
The [completion diagnosis](.flow/tmp/completion-trace/report.md) traces the worker and by-ID
responses to identical recorded facts. The worker's offered response is not server acceptance;
the correction needs truthful response-source evidence, with terminal status retained. The same
source distinction must be resolved before fn-128.7's fatal classification can establish its
worker-only Property. Join those source changes before fn-128.8 seals final Case/Profile bounds.
The [SDK response observation notes](.flow/tmp/completion-trace/sdk-response-observation.md)
identify an existing token-response observation seam, not an accepted correction. RPC success alone
does not prove completion: an oversized completed result can become a failure while returning
success. Any correction must prove truthful source attribution together with the terminal outcome,
including retries and concurrent responses, without changing the strict Property to fit the server.
The independently reviewed correction plan seals fn-138's original/adopted R3 comparison before
fn-129 changes Source, then runs fn-128.7 and .8 as disjoint parallel candidates after fn-129.4.
The close checks every authored expected status/reason exactly, including the retained retry
Property-only `explanationsDisagree` expectations. Fatal satisfaction, satisfied Contracts,
conformance and bounded pause/resume execution/replay remain required; only the named
ShutdownWorker race permits an additional inconclusive.

#### fn-128: Close the activity's precision gaps

Tasks 1-5 are done. Tasks 7 and 8 are parallel candidates after the source gates above; task 6
closes the shared batch after both corrections and all activity sources.

| Task | Status | What |
| --- | --- | --- |
| fn-128.1 | ✅ done | Dispatch as a field replacing the `backingOff` phase; start delay; unpause-after-backoff and schedule-to-start-in-backoff fixed |
| fn-128.2 | ✅ done | Explicit rejection rows and repeated RequestCancel refusal; owned lint subjects removed; focused tests pass |
| fn-128.3 | ✅ done | 2026-10-08: finite retry policy and retryable start-to-close timeout; exhaustion/timeout-retry Queries and bounded withholding bridge; focused source/lowering/runtime checks pass |
| fn-128.4 | ✅ done | All public status facts visible; Product carries creation/held pause/withdrawal; focused checks and current-source refinement/mutant replay pass |
| fn-128.5 | ✅ done | Checked `cancelIsNotUndone`; final typed raw attempt-count read in all 13 supported Cases; seven timer explanations; integrated source proofs pass |
| fn-128.7 | ⬜ todo | Accepted fatal-settlement classification through typed public last-failure evidence; unchanged fatal Property satisfied in scratch recording/replay |
| fn-128.8 | ⬜ todo | Separate caller-owned pauseResume and heartbeat-retry Profiles; immutable preflight/execution/replay binding, final-shape charge proof and admission negatives |
| fn-128.6 | ⬜ todo | Evidence map, live Cases run once (the batch's live run); close |

#### fn-138: Retries and Deadline capabilities

Implementation complete (3/3 tasks); whole-spec review and shared Batch 5 closure pending.
The done source tasks do not replace the production equivalence, full gates or live-run evidence
reserved for that boundary, so this spec remains listed until closure.

Runs after fn-128.5: fn-128.1 replaces the `backingOff` phase and fn-128.3 adds the retry policy,
both of which Retries reads. The approved recommendations resolve the earlier owner questions;
the refreshed plan passed independent review on 2026-10-08.

Retries checks before-state policy eligibility and control-aware failure settlement, including
Nexus failures from Waiting. Deadline checks armed timer windows and retry/terminal settlement.
Activity and Nexus declare both; Nexus capabilities are explicit IR roots. Existing Query and Case
semantics remain protected by the reviewed comparison contract.
Retries' finite policy projection uses `Option[UpTo[N]]`, with explicit unlimited `None` and a
policy-value catalog independent of the attempt-count domain. This type correction passed the
same independent plan review; the task updates both existing Scala and Go vocabulary gates.

| Task | Status | What |
| --- | --- | --- |
| fn-138.1 | ✅ done | Retries and selected before-state transition verification integrated; 130 fixture checks and 98 integrated focused checks pass; full generated-IR parity pending at the batch close |
| fn-138.2 | ✅ done | Typed Deadline capability integrated; three exact goldens, 30 located refusals and 123 fresh integrated focused checks pass; production generation pending at the batch close |
| fn-138.3 | ✅ done | Activity/Nexus adoption integrated; strict R3 proof preserves 121 Queries and 21 exact Cases; 24 native and 42 integrated reader checks pass |

#### fn-129: Activity coverage

Runs after fn-138. Tasks run in order.

| Task | Status | What |
| --- | --- | --- |
| fn-129.1 | ✅ done | Typed heartbeat protocol; canonical lint, native/six-root checks, three fresh scratch Case recordings/replays and Profile negatives pass |
| fn-129.2 | ✅ done | Independent by-ID service settlement; four fresh Case recordings replay exactly, three candidate native Quick commands pass; by-ID witnesses expect `explanationsDisagree` |
| fn-129.3 | 🔄 in progress | Reset with `keepPaused` and deferred settlement joined: 65 Queries, `deferredResetCompletes` lowers and prepares, reset-aware laws via `overriding(…, of = …)`; open: direct `keepPaused` live Case (lowering refuses a reset indistinguishable from the pause before it) |
| fn-129.4 | ⬜ todo | Exploration on the activity's `find` Queries |
| fn-129.5 | ⬜ todo | New Cases listed, live run (the batch's live run); close |

## Independent work while fn-155 validation is deferred

### fn-150: Bounded liveness across composed machines

[Spec](.flow/specs/fn-150-bounded-liveness-across-composed.md) planned with five M-sized tasks covering
all five acceptance criteria. Task .1 is admitted as an independent checker proof while fn-155's
canonical validation is deferred. Let a composition own bounded progress claims over multiple
member states. Count composed steps, resolve fairness against synchronized and member-only actions,
and preserve deadlock, cycle, deadline and incomplete-check distinctions. Include one concrete
Temporal composition with passing and negative examples.

Related to fn-149's safety/liveness groups. Coordinate with fn-141 and the queued schema changes;
the later schema/authoring/example tasks re-anchor before their own admission. Live Case
generation for compositions is outside this spec.

| Task | Status | What |
| --- | --- | --- |
| fn-150.1 | ⏸️ deferred | Complete scoped implementation checkpoint; 29 focused tests and lint pass, original Quick OOM before/after and after private daemon exit; review and closure held |
| fn-150.2 | ⬜ todo | Structured fairness references, inherited/replacement mapping |
| fn-150.3 | ⬜ todo | Typed composition progress and fairness authoring/lifting |
| fn-150.4 | ⬜ todo | Bound, fairness, holes, starts and witness-replay regressions |
| fn-150.5 | ⬜ todo | Temporal positive/negative example, docs and integrated gates |

Waves: `.1` then `.2`; `.3` and `.4` are parallel candidates; `.5` joins them.
Fn-150 has no hard dependency on fn-149. Its hand-authored Go checker proof changes no Model source,
schema or managed artifacts and is disjoint from fn-155's remaining fixture/test work. Serialize
later overlapping edits and regeneration with the approved delivery chain and re-anchor their
paths before dispatch. This admission does not release fn-156's fn-155 closure dependency.

Task .1's original Quick remains RED. Checkpoint `2d64387595` stays unintegrated and unreviewed
on its task branch; its Flow block reason links the complete scoped proof and unchanged full
verification obligation. Task .2 cannot start until .1 is verified and closed.

The one unchanged Quick retry after the private fn-155 compiler daemon exited also failed
naturally: exit 1 after 39.102 seconds, checker killed after 36.698 seconds and reader cached pass.
Kernel evidence records global OOM; minimum sampled available host RAM was 319,632 KiB.
The default runtime, original selector and shared heavy-command lock were preserved. The terminal
receipt and kernel logs remain in the task worktree's `.flow/tmp/fn150/task1/revisit-private-exit/`.
Revisit only after a material memory-capacity change or a preservation-safe repair, not for a
disk-only improvement. Passing focused checks do not replace the original Quick obligation.

### fn-157: Bound native verification memory and scratch storage

[Spec](.flow/specs/fn-157-bound-native-verification-memory-and.md) has five reviewed tasks.
The conductor activates measurement .1 on committed integrated baseline `551bf89c68`, with a
documented initial runner budget, because global OOM now holds fn-155 and fn-150 verification.
This follows the owner's recommendation mandate; it does not grant a passing gate or authorize
production repair before the complete independent oracle and causal measurements.

The host reports 16,593,792 KiB RAM, no swap and no cgroup memory maximum. Initial snapshots
separate overlay inode shortage from available host scratch; .1 must measure actual demand
and allocation ownership. Victim RSS and later free-space snapshots are not those measurements.
Preserve every table, Query, receipt, replay identity, strict assertion and canonical `-p 2`
concurrency. Full restored gates remain required; Quint JSON work stays with fn-154, strict
Activity semantic debt with Batch 5, and Canary failures retain their separate attribution.

Retained fn-151 RED commands and input pins remain in its task3 proof archive. Fn-155's later
global-OOM excerpts and full failure/partial handover also remain recoverable. No shared daemon,
broad cache or unowned scratch cleanup is authorized. Measurement .1 precedes native .2/.3 and
the separate scratch .4 lane; .5 joins complete preservation and exact no-update gate receipts.

The historical proof archive is now copied and content-validated, and the activation inventory
pins seven Models and all 340 Queries, including the original 154 Activity occurrences. Fresh
primary Activity Check completed in 20.510 seconds; producer construction was killed after
17.725 seconds. Complete primary tables and another 34 reader-surface processes across all
seven Models finished naturally; the latter batch took 219.265 seconds. The six non-primary
Models also completed Producer/Lower capture for all 275 Queries; the primary Model's 65 Lower
results remain missing. These are preservation captures, not restored gate-fit evidence.
After the exclusively owned historical fn-155 private
Bloop daemon exited normally, the full fresh primary Producer was still OOM-killed after
19.258 seconds, before Cases or lowering inventories were captured. The shared compiler remains
untouched. Phase logs, heap profiles and both failed attempts remain preserved.

The ordinary no-update model gate then failed after 9.955 seconds: Java fixture directory
creation still targeted `/tmp` despite host `TMPDIR`/`GOTMPDIR`. During-run samples show overlay
free inodes falling from 2,001 to one with about 29.9 GB free bytes; the failures report ENOSPC.
This establishes inode exhaustion at fixture setup, not successful full gate scratch demand or
Lift/Case overlap. Storage capacity subsequently increased externally: `/tmp` had over 6.8 million
free inodes. The unchanged no-update gate retry passed scratch tests but its Case generator was
globally OOM-killed; the gate exited 2 after 37.691 seconds. Its generated-proto timestamp warning
remains a separate finding. RAM availability was not materially changed, so the primary Producer
was not retried for a disk-only improvement. No successful full-gate scratch peak is established.

Measurement .1 remains held on the complete independent oracle; root has resumed only the
remaining independent error/replay/isolation controls and safe whole-Model captures. The primary
Producer and unchanged ordinary gate are not retried without material memory improvement.
Receipt precedence and replay-error controls passed. A pinned `activityProduct` caller-mutation
control failed naturally: repeated interpreter builds retain shared nested `Row.Results`, despite
distinct outer rows; a fresh interpreter remains independent in that control. The RED assertion
is retained under the task worktree's `controls-resume/` evidence. This is an additional ownership
obligation for the held repair, not a complete oracle, accepted strategy or passing full gate.
The resumed captures now include all 61 claimed-table owners across seven Models (59 full streams
and two typed refinement refusals), plus native-supplied dump serialization/admission controls
for the six non-primary Models. These dump
controls did not run an external Quint evaluator, regardless of stock receipt wording. Separate
generated-manifest joins cover all 275 non-primary Queries and their twelve executable Cases.
Conformance captures retain twelve full private plans and fourteen nonempty Nexus streams across
seven workflow Cases after caller Model/Case mutation. Runtime evaluation and matching replay now
cover those seven Cases in both history modes, with foreign-assessment and causal-reversal controls.
These remain partial preservation evidence:
the primary Producer's 65 Lower results, primary dump and full all-owner conformance/replay oracle
are still missing. Root verified the new 525-file seal and the unchanged original 574-file seal.
Parallel read-only audits confirmed the checked integrity and identified five remaining nonempty
Case/replay/isolation controls on the other three small Models. Those controls are resumed;
task-local parallel lanes must use disjoint evidence directories and serialize heavy commands.
The repeated-caller RED is retained baseline evidence, not a demand for repair before measurement;
external Quint execution remains fn-154 work and is not an added fn-157 measurement gate.
No repair, review, gate-fit or closure credit follows from these successful capture commands.
The first continuation's handover accounts for the original 34 surfaces as sixteen captured,
twelve partial, five missing and one RED. Its 47 probes used 150.591 measured wall seconds;
with the earlier 52 probes/412.923 seconds, the retained total is 99 probes/563.514 seconds.
The one-hour limit was not reached. Revisit the primary captures on capacity sufficient for the
unchanged whole primary Producer and ordinary compiler overlap, retaining effective JVM scratch
and all original inputs/assertions. All production repairs remain held. Managed artifacts and
fresh lifted candidates have separate pins; historical source freshness is not assumed.

The dedicated read-only memory investigation found three complete interpretations overlapping
inside `NewProducer`: its Realizer, Check's first binding and independent replay binding. The
retained Producer heap profile attributes 1,037.53 MiB of 2,348.56 MiB sampled retained allocations
to transition-slice clones. This is a snapshot allocation-site account, not the OOM peak or a
measured saving. Check also builds complete canonical fingerprint strings before Lower, making
incremental exact-byte hashing another candidate; streaming final Case output alone misses this
failure phase. No production correction or successful memory budget follows from these findings.

A possible future disposable .1 diagnostic separates the complete frozen Check/replay process from a
whole-Model lowering Realizer, supplying sealed receipts to unchanged Lower and private producer
code through a pinned test-only constructor seam. It must first match all 275 ordinary outputs
on the six complete Models, including standings, errors, artifacts, identities and inventories.
Only that comparison can admit the primary Model's full 65-Query capture. A mismatch rejects
the seam; successful capture supplies a functional oracle, never unchanged constructor or
canonical gate-fit credit. All prior evidence and the caller-mutation RED remain preserved.

The owner deferred fn-157 on 2026-10-10. All workers and owned commands are terminal; the
constructor experiment never started. The third immutable evidence generation seals 151 files;
all three seals were independently verified. Remaining controls bring the native diagnostic total
to 114 probes / 596.297 executed seconds, with 28 captured surfaces, one captured RED and five
still missing. These are partial functional receipts, not complete-oracle, repair or gate-fit credit.
Resume from the preserved deferral handover and seals, without rerunning or changing their inputs.

| Task | Status | What |
| --- | --- | --- |
| fn-157.1 | ⏸️ deferred | Owner deferred after three immutable seals; complete primary oracle still missing; constructor experiment not started; no repair or gate-fit credit |
| fn-157.2 | ⬜ todo | Repair only the measured private native owner; ownership, mutation isolation and independent replay proof |
| fn-157.3 | ⬜ todo | Bound implicated consumer and complete seven-Model generator lifetimes; preserve full outputs and assertions |
| fn-157.4 | ⬜ todo | Repair owned gate/lift scratch lifetimes or establish provisioned capacity with ordinary overlap |
| fn-157.5 | ⬜ todo | Join full preservation, execute interrupted tests and restore exact native gates and measured capacity |

## Deferred

Specs the owner deferred keep their tasks here so they can be revived as planned.

### fn-119: Show one Go SDK workflow driven end to end from the IRs

Deferred 2026-10-04.

| Task | Status | What |
| --- | --- | --- |
| fn-119.1 | ✅ done | Driver's workflow schedules an activity, awaits it, completes with its result |
| fn-119.2 | ✅ done | Workflow-scheduled activity attempts routed to the Driver's interpreter |
| fn-119.3 | ⏸️ deferred | Workflow-scheduled activities and awaited outcomes in the realization DSL, lifter and lowering |
| fn-119.4 | ⏸️ deferred | Activity workflow example modeled; its Queries run live from the gate |
| fn-119.5 | ⏸️ deferred | Faulty variant and the zero-Go check |
| fn-119.6 | ⏸️ deferred | Walkthrough, one-command entry point; close |

### fn-125: Represent dynamic configuration in the Models

Deferred 2026-10-05. Evidence: `.plans/DYNAMIC_CONFIG.md`.

| Task | Status | What |
| --- | --- | --- |
| fn-125.1 | ✅ done | HSM/CHASM switch fixed; schedule-to-close no longer from the Profile |
| fn-125.2 | ⏸️ deferred | `setting[T]` over finite domains in the framework, lifted |
| fn-125.3 | ⏸️ deferred | Query `under`: one Query per valuation |
| fn-125.4 | ⏸️ deferred | Settings in the Quint/P exports |
| fn-125.5 | ⏸️ deferred | Dynamic-config keys declared once in the kit, pinned to the server registry |
| fn-125.6 | ⏸️ deferred | API preconditions; derived required settings; ShutdownWorker precondition |
| fn-125.7 | ⏸️ deferred | Nexus implementation encoded; one Case per valuation; switch retired |
| fn-125.8 | ⏸️ deferred | Caller attempt semantics as the owner chooses |
| fn-125.9 | ⏸️ deferred | Bound assumptions on server durations checked at preparation |
| fn-125.10 | ⏸️ deferred | Disposition for every implicit assumption |
| fn-125.11 | ⏸️ deferred | Docs; close |

### fn-130: Model views

Execution remains owner-deferred from 2026-10-05. Preparation now has seven tasks covering the
original R1-R5; independent plan review returned SHIP after fixing nested phase/queue projection
coverage and enabled-guard retention from lint's existing evaluation. Fn-126's committed closure satisfies the
sole feature dependency. The proposed delivery placement is after Batch 5, with actual admission
and baseline re-anchor still conductor-owned. Preparation grants no production gate credit.
Evidence: `.plans/MODEL_VISUALIZATION.md`.

Rendered views per Model (signature, phase diagram, refinement, compositions, derived-design diff, witness paths), checked in as `.d2` plus `.svg` under `model/views/` and gated; D2 as a Go library with ELK; no DSL declaration.

| Task | Status | What |
| --- | --- | --- |
| fn-130.1 | ⬜ todo | Disposable renderer/projection proof and full-workload added-cost feasibility; no production rollout on a failed proof |
| fn-130.2 | ⬜ todo | Pinned phase renderer, nested/queue projections, retained enabled guards and live command/ownership foundation |
| fn-130.3 | ⬜ todo | Signature and composition/sync views with actual actor/substitution/source metadata |
| fn-130.4 | ⬜ todo | Existing checked state-map public seam and refinement/carrier/stutter views |
| fn-130.5 | ⬜ todo | Inferred comparison bases, truthful ties/identical siblings and full concrete-result diffs |
| fn-130.6 | ⬜ todo | Actual replay-validated expecting-find witnesses and complete per-IR overviews |
| fn-130.7 | ⬜ todo | Safe complete-tree freshness/publication, foreground reuse, docs and original full gates/cost/preservation proof |

The historical research names a retired reader package. Tasks use the current public reader and
lint owners; they add no alternate evaluator or IR view declarations. The original few-seconds
added gate target, all seven view families, complete inventories and independent replay remain
required. Native repair/oracle holds, owner deferrals and the delivery chain are unchanged.

### fn-151: Nexus matching model and bug-finding evidence

⏸️ Deferred 2026-10-08 before task planning; the spec has no tasks and is not ready for execution. [Spec](.flow/specs/fn-151-nexus-matching-model-and-bug-finding.md).

Bounded Nexus dispatch model covering request/response correlation, cancellation and deadline races, one forwarding hop, routing changes, and shutdown. Validate against real Go matching and independently selected historical or seeded defects; compare with existing tests and report misses, limitations, and cost. Excludes endpoint management, persistent backlogs, and the full Nexus operation lifecycle. Revisit the proposed scope and available hooks before activation.

### fn-152: Named edge-case situations

⏸️ Deferred 2026-10-08 before task planning; the spec has no tasks and is not ready for execution.
[Spec](.flow/specs/fn-152-named-edge-case-situations.md).

Define reusable, named Situations for interesting states and bounded event orderings, separately
from the Properties that judge their correctness. Targeted regressions must prove the Situation
occurred; documentation and reports reuse its identity and witness evidence. The motivating example
is an original Nexus reply arriving after retry dispatch, with distinct attempt identities and
reply-order variants. Resolve the receiving boundary and allowed Nexus outcomes when revived.
Coordinate with fn-140's witness authoring and the deferred Nexus matching model; automatic discovery
and campaign-wide coverage are outside the proposed initial scope.

### fn-154: Bound memory for exhaustive Quint JSON agreement

⏸️ Deferred by the owner on 2026-10-09; not ready for execution.
[Spec](.flow/specs/fn-154-bound-memory-for-exhaustive-quint-json.md).

Generate and consume Quint ITF JSON incrementally instead of retaining a complete multi-gigabyte
document. Keep JSON at the external protocol boundary and native Go tables or existing protobuf
data internally. Preserve exhaustive states/actions, claims, products, ordered receipts, replay
witnesses and JSON refusal behavior. Assess generation and consumption together, including the
real Quint file path and API compatibility. Large generated dumps remain temporary and out of git.
This records the deferred check and resource work, not a passing result or an Activity failure waiver.

The last canonical command was `mise exec -- go test -tags test_dep -p 2 -timeout 30m -json ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/...`: exit 1, 49 packages passed, three had no tests and conformance was OOM-killed after completion failed. Export passed all four faithful fixtures. Evidence remains under `/tmp/umpire-fn1454.aTUadX/.flow/tmp/fn1454/go-full-raw-reader-canonical.*` and in Flow's structural completion receipts. Revisit concurrent memory fit after incremental generation and consumption preserve the complete JSON/native/receipt/replay contracts; the inherited Activity assertions remain separate Batch 5 obligations.

| Task | Status | What |
| --- | --- | --- |
| fn-154.1 | ⏸️ deferred | Incremental JSON generation and consumption, preserved full coverage and measured concurrent memory fit before restoring the deferred gate |

### Other deferred items

- fn-122.7 (the Pausable capability Property on fn-119's example) waits for fn-119.
- `make umpire-check-backends` in CI; it runs locally after `make umpire-install-backends`.

## Open for the owner

- 691 lint findings accepted with reasons in `model/ir/*.lint.json` (fn-120.3); review the H2 reasons first.
- Behavior-freeze follow-ups from fn-112: the witness-only Properties `terminated` and
  `cancelRequestedWhileStarted`, and seven pause/unpause rows the server rejects. fn-140 deletes
  `terminated` with its Query (R6) and turns `cancelRequestedWhileStarted` into a witness (R5).
- Whether HSM and CHASM may count Nexus `attempt` differently; no current Query shows a difference (fn-125, deferred).
- Whether upstream's Go conformance harness (`tests/activity_driver.go`) should run our IR through the Go interpreter instead of its hand-written model, making one Model drive both (`.plans/ACTIVITY_MODEL_COMPARISON.md` P3-12); needs the owning team. A workflow-scheduled activity realization (P3-11) overlaps the deferred fn-119.
- The canary policy's `workflowPath` names the deleted production-canary workflow, so production dispatch
  fails closed.
- The three by-ID witnesses (`ByIDCompletion`, `ByIDFailure`, `ByIDCancellation`) expect
  `inconclusive(explanationsDisagree)`: the Model's NotFound rows let a silent rejected repeat explain the
  same evidence, as for `terminate` and `cancelIsRequested` (fn-129.2). Sharper witnesses would need
  repeated-call rejection to be observable.
- Batch reorganization (2026-10-08): batches 1 to 3 replace the earlier per-spec migrations, and
  Flow's spec dependencies match: each spec depends on every spec of the previous batch, with no
  spec dependencies inside a batch. Reverting to the earlier order means restoring per-spec
  regenerations and those dependencies; the tasks are unchanged. The activity model batch then moved
  last at the owner's request: until it closes, batches 2 and 3's live runs include activity Cases with
  its three inherited failures (ordinary completion ambiguity, non-retryable failure visibility,
  pause/resume work ceiling), which those runs report but do not own.

## fn-153: Checked property examples pilot

Six M-sized tasks cover all seven requirements. The plan passed independent review on 2026-10-10;
execution is scheduled after Batch 5 and is not marked ready. This placement avoids shared
compiler/lifter/schema baselines without inventing a feature dependency.
[Spec](.flow/specs/fn-153-checked-property-examples-pilot.md).

Pilot checked examples and counterexamples attached to three existing Properties: a same-step
promise, a before/after promise, and a promise with a subtle applicability boundary. Start with
Nexus synchronous success, contrasting its required state and completion fact. The existing
evaluator checks each illustration; unrelated actions are not applicable, and hypothetical
violations remain distinct from reachable model counterexamples.

Generate a compact explanation beside each Property, verify that stale classifications and
deliberately weakened predicates are detected, and assess authoring burden and explanatory value
before expanding. Preserve model behavior and behavioral identities. Coordinate with fn-140,
fn-149 and fn-130; fn-152's named Situations are not a prerequisite. Live test generation,
temporal-sequence examples and a repository-wide rollout are outside the pilot.

| Task | Status | What |
| --- | --- | --- |
| fn-153.1 | ⬜ todo | Optional illustration carriers, exact domain admission and frozen behavior/identity pins |
| fn-153.2 | ⬜ todo | Typed authoring attachment, lifting and nested source locations |
| fn-153.3 | ⬜ todo | Existing Property selector/predicate classifier, errors and local mutation controls |
| fn-153.4 | ⬜ todo | Compact deterministic document and gate freshness; handwritten assessment outside its managed subtree |
| fn-153.5 | ⬜ todo | Three existing Nexus Properties' contrast pairs and classification drift evidence |
| fn-153.6 | ⬜ todo | Evidence-limited assessment, docs, complete compatibility comparison and full gates |

Tasks run in order. Re-anchor against the actual baseline after the delivery chain; keep
hypothetical violations, actual model failures and live coverage separate. Mechanical checks
do not establish developer comprehension, and absent feedback must remain explicit.
