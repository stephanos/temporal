# Umpire4 milestones

The current state of Umpire work: what is being built, what is left, and what is not being done.
Flow (`.flow/`, `flowctl`) is the record for specs and tasks; this page is the overview across them.

As of 2026-10-10.

Execution resumed on 2026-10-10 under the owner's instruction to complete all milestone work,
taking the conductor's recommendations without further questions. The compiler, format and
fairness checkpoints are merged into `umpire` (`ae3f36ee51`), as are fn-156's source close
(`39b7580104`) and fn-160.5's equivalence check (`ab8e0f25fa`), plus a model formatting commit
(`2ddc86bc32`). Parallel lanes: Batch 3's format close; Batch 1d core (framework/lifter), activity,
nexus and shared-model lanes, with the decisions recorded under Batches 1d and 5b; fn-150; fn-155.6's
canonical regeneration retry. JVM/Go work shares three slots (`.flow/tmp/bin/slot`); full gates are
exclusive.

## Keeping this page current

- This page describes the present. Rewrite a status in place; do not append dated entries.
- List each open spec's remaining tasks in a `Name | Status | What` table: the task ID, its status
  (🔄 in progress, ⬜ todo, ⏸️ deferred or paused) and a brief description. Set a status in place when
  it changes. Remove a task's row once it is done, and the spec once it is closed. Flow and git keep
  the history.
- Group scheduled specs under the batch that shares their regeneration (see Batches). Task rows
  record the task's own evidence; the batch-deferred checks are implied and need not be repeated.
- Only the root agent edits this page. Workers report status in their handoffs, so isolated
  worktrees never carry a conflicting copy of it into a join.
- Close a cancelled or abandoned spec in Flow (tasks blocked, a "Closed: won't do" note in the
  spec) and remove it from this page.
- Update the "As of" date with every edit.

## Execution strategy: batch first

Optimize for completed, integrated batches rather than individually green tasks. The owner
authorized aggressive batching, parallel implementation and temporary breaking changes on
2026-10-10. This strategy supersedes the start-order and per-task gate instructions in older
plans; it does not remove their functional requirements or final acceptance criteria.

- Assign workers substantial end-to-end batches, spanning related tasks or specs. Task IDs remain
  requirement and evidence anchors, not mandatory implementation, review or regeneration barriers.
  Do not stop after each small task to wait for review or a predecessor's formal closure.
- Run independent implementation lanes in parallel with sub-agents and isolated worktrees. Pin a
  minimal shared interface, assign each shared file one owner, then develop producers and consumers
  concurrently. Coordinate interface changes directly; do not turn every handoff into a new spec.
- Accept temporary compiler, test, schema and generated-artifact breakage inside an implementation
  batch. Migrate callers and remove obsolete machinery together; do not build compatibility layers
  merely to keep intermediate commits green. Record known breaks and who owns their repair.
- Dependencies constrain integration and closure, not all implementation starts. Work ahead of
  unfinished predecessors on a pinned baseline when the required interface is known; root records
  the exception in Flow and reconciles it at integration. An OOM or deferred full gate must not
  idle unrelated source work. Existing owner deferrals remain in force.
- Capture immutable pre-change inputs before broad edits. Keep separate equivalence comparisons
  for meaning-preserving migrations so later semantic changes cannot hide regressions. Those
  comparisons are integration obligations, not reasons to serialize all worker starts.
- Use focused checks during implementation. Run regeneration, full verification and independent
  review at coherent integration checkpoints, normally the batch boundary, rather than per task.
  Reviewers may examine different risks in parallel against the same integrated snapshot. Reuse
  valid evidence; after fixes, rerun only checks whose relevant inputs or results were invalidated.
- Prefer direct implementation of decided changes over additional prototypes, inventories or
  planning rounds. Add a proof or investigation only when required by acceptance or needed to
  resolve a concrete uncertainty. Keep handoffs short and move implementation forward.
- A batch is accepted only when its integrated result passes the required checks and review.
  Link shared evidence to each covered task before closing it in Flow. Temporary red results,
  partial proofs and deferred gates are never passing evidence or completed-spec credit.

The immediate implementation lanes are fn-156's compiler/lint rollout and the coordinated
fn-146/fn-147/fn-148 Testpilot format migration. Within the format lane, migrate Umpire runtime
operand lowering and Testpilot CEL admission/evaluation together, plus Duration and evidence
schemas, under one schema owner. Testpilot owns runtime CEL; the finite Scala Model evaluator
remains separate. Activate format 4.0 once at fn-148.6 and regenerate managed artifacts and Run
companions once at fn-148.7. These are scheduled batches, not claims that every task has started.
Fn-157 remains owner-deferred; its memory work is not resumed by this strategy.

## Verification instructions for agents

Apply these instructions when implementing the milestones:

- The root agent owns scheduling, dependency decisions, integration and Flow status. Sub-agents own
  task-local investigation, implementation, testing and fixes; independent reviewers assess the
  work. Workers return concise handoffs linking commits, verification evidence and remaining
  blockers. Root inspects the evidence needed for acceptance without importing entire task
  histories. Resume the same worker for fixes where possible.
- A blocked task holds dependent acceptance, not every dependent implementation start. Continue
  other lanes and documented work-ahead under the batch-first strategy, retaining ownership and
  heavy-command constraints.
- A completed predecessor satisfies its dependents once its Flow closure is committed in the
  current local branch's history. Publishing or landing on origin is not required. Uncommitted
  closures and work on unmerged branches do not count. Work-ahead may start from a pinned integrated
  baseline without claiming that closure, and must reconcile at integration. Publishing is a
  separate, explicitly authorized action.
- If a validation remains stuck for more than one hour, including repeated attempts, mark it
  deferred in Flow and this page and move to work that can be verified and delivered. Continue
  beyond that limit only when the validation blocks every other available work path. Record its
  command, elapsed time, failure evidence and condition for revisiting it. Preserve the failed
  result and follow-up obligation; deferral is not a passing gate.
- Reuse the previous task's passing baseline when its commands, source scope, fixtures and
  environment still apply. Inspect its recorded evidence; a new task or agent is not a reason to
  rerun it. Changes to relevant inputs invalidate the affected results.
- During implementation, run the smallest tests that exercise the changed behavior and its failure
  modes. At the batch integration checkpoint, run its required full tests, goldens, lint and
  dependency checks once. After a fix, repeat affected checks; repeat broader gates only when the
  change or failure invalidates their results. Preserve all required coverage and acceptance criteria.
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
  only for an explicit requirement or a concrete uncovered risk. Continue implementation within
  the batch without waiting for per-task full gates or independent review.

## Batches

A batch is a coordinated implementation of related specs that share one baseline:
one production regeneration, one full gate run
(Go suite, model gate, goldens, lint, dependency checks), one independent review and, when the batch
changes Cases or runtime behavior, one live run. Batching buys throughput with attribution, so a
batch boundary falls where a failure must stay attributable: after a meaning-preserving refactor's
equivalence proof that later work builds on, and at a format activation.

- Inside a batch, tasks run focused source, lowering and runtime checks and declare their IR change;
  production artifacts, full gates, review and live evidence are batch-deferred.
- A meaning-preserving spec seals its comparison (a scratch regeneration whose diff is exactly the
  stated mapping) before integrating a later semantic change onto that baseline. Workers may
  implement both concurrently against frozen inputs. The seal is focused evidence, not a full gate.
- Shared close: the batch's final close task runs the regeneration, gates, review and live run once;
  the other close tasks finish their own code and docs and link that evidence. No spec in a batch
  closes before the batch boundary.
- The conductor enforces source and acceptance dependencies at integration, documenting work-ahead
  exceptions in Flow. Existing dependency edges are not blanket prohibitions on parallel source work.
- Batch numbers describe integration order, not a global worker queue. Implement disjoint batches
  concurrently and integrate continuously where ownership and interfaces permit. Reconcile shared
  baselines before closure; serialize conflicting file edits and memory-heavy commands, not all
  implementation. Fn-155's missing baseline proof holds final acceptance, not fn-156 or format
  implementation. Batch 5 retains its final schema/authoring reconciliation obligations.

| # | Batch | Specs | Nature | Shared close | Live run |
| --- | --- | --- | --- | --- | --- |
| 1b | Activity abstractions | fn-155 (named repeated patterns) | meaning-preserving, mapped identities | fn-155.6 | none |
| 1c | Compiler and lint | fn-156 | byte-identical | fn-156.7 | none |
| 1d | Scala DSL (immediate) | fn-159, fn-160.3–.5, fn-161, fn-162.2–.3 (Scala parts); investigations fn-160.1, fn-162.1 | source only; IR left stale | fn-161.1 | none |
| 2 | Authoring | fn-140 → fn-149 → fn-123 | two equivalence seals, then meaning | fn-140.6, fn-149.5, fn-123.8 | once |
| 3 | Testpilot format | fn-146 → fn-147 → fn-148 | breaking format | fn-146.7, fn-147.4, fn-148.7 | once |
| 4 | Lifter | fn-141 | byte-identical | fn-141.14 | none |
| 5 | Activity model | fn-128 → fn-138 → fn-129 | meaning | fn-128.6, fn-129.5 | once |
| 5b | One regeneration | fn-162.2 (IR schema, lowering, Go assessment), fn-160.5 (IR projection), fn-160.2 close; Go and fixture consumers | meaning-preserving, mapped identities | fn-160.2 | once |

## Direction

The Scala model (`model/`) is the model. Lean is the past, and nothing has to look like it.
Scala declares, a lifter reads the declarations into the Umpire IR, and generic Go consumers check
that IR, lower it to Testpilot Cases, and run those Cases as functional tests and canary checks. The
Umpire IR and the Testpilot IR are what connect the parts. See [SCALA.md](.plans/SCALA.md) and
[UMPIRE4_SPEC.md](.plans/UMPIRE4_SPEC.md).

The DSL framework (`model/framework`) stays Temporal-agnostic as far as is realistic: Temporal's capability
vocabulary, properties, realization vocabulary and kit live under `model/temporal/`, and the lifter and Testpilot IR
are the parts that are Temporal's driver tooling by design (fn-114.12, fn-122.8).

Next, batch the compiler/lint rollout alongside the coordinated Testpilot format migration.
Overlap Scala authoring work where file ownership permits; complete its baseline reconciliation
before accepting the format batch, then finish lifter and Activity coverage integration.

## Specs

Listed in integration order, grouped by batch. Flow records spec and task dependencies; the conductor
also holds batch-close gates while scheduling parallel implementation under the batch-first strategy.


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

| Name | Status | What |
| --- | --- | --- |
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

[Spec](.flow/specs/fn-156-let-the-scala-compiler-and-lint-enforce.md). Tasks .1–.6 and .8 are done and
merged (`39b7580104`). A fresh lift of the final source equals a fresh lift of the pre-fn-156 source byte
for byte, positions included (11 files, 25,191 positions), and `make lint-model` passes. The reports are
joined and enforcement is documented (`model/README.md`, `.plans/SCALA_CAPTURE_CHECKING.md`). Explicit
nulls was trialled and dropped (`.plans/SCALA.md`). Task .7 closes once a full Go suite and model gate
run on a tree that includes it, which Batch 3's close supplies; fn-155.6's whole-regeneration proof stays
a separate hold. The compile-versus-lift items from the review remain notes on fn-141.

| Name | Status | What |
| --- | --- | --- |
| fn-156.7 | ⬜ todo | Join reports, document enforcement, exact artifact comparison, full gates and reviews; close |

### Batch 2, authoring: fn-140 → fn-149 → fn-123

Integrates onto fn-156's accepted baseline; implementation may work ahead under the batch-first
strategy, with task paths re-anchored to the completed moves,
Activity subject split, abstractions and compiler/lint enforcement. All three specs
rewrite Model declarations, so they share one production regeneration, one full gate, one review and
one live run at fn-123.8. fn-140 is meaning-preserving apart from its declared renames: fn-140.6
seals its assessment-equivalence comparison before fn-149 groups claims. Fn-149.5 seals the final
grouping, including documentation and executable-layout changes, before fn-123 changes meaning.
Each seal compares against its independently frozen predecessor; a fault change cannot hide inside
either migration.

Integration gates: fn-149 joins onto fn-140.6's committed witness seal; fn-123 joins onto fn-149.5's
committed final grouping seal. Implementation may overlap against pinned interfaces and frozen
comparison inputs. The source seals do not close their specs or discharge shared gates.
Fn-123 consumes the grouped baseline and composed identity/provenance mapping, using fn-140's
`when` blocks and `.live` together with the shared `Outcome`, fault instruction and typed `perform`.

Fn-149's insertion follows the conductor's recommendation under the owner's instruction to finish
all milestones and choose recommendations without further questions. The three plans and their
cross-spec handoffs passed independent review; fn-149 is ready. Sharing production regeneration
does not eliminate the separate source migrations or their equivalence proofs.

#### fn-140: One-sentence witness Queries with explicit live expectations

It rewrites the Models' `properties` and `queries` sections and its R5 renames Definition
IDs. Ready, with six M-sized requirement anchors. Batch the foundations and caller migration;
documentation and disjoint conversion work may run in parallel. Task 6 joins the equivalence seal.

`witness(<classes>).records(<fact>)` states a path-and-outcome claim in one declaration and lifts to the existing Scenario, Property and `find` Query, each named after it. `.live(<expectation>)` replaces `.expect` as the one word that generates a Case, and every non-satisfied expectation carries a reason. Pinned `find` Queries in every Model migrate, and the hand-written `terminate`, which repeats the capability-generated `terminateSettles`, is deleted. `query verify` and its triple are unchanged.

| Name | Status | What |
| --- | --- | --- |
| fn-140.1 | ⬜ todo | Typed witness builders and existing core Query values |
| fn-140.2 | ⬜ todo | Witness lifting, names, bounds, composed facts and capability duplicate refusals |
| fn-140.3 | ⬜ todo | Query `.live`; source-only non-satisfied reasons; all caller and fixture spellings |
| fn-140.4 | ⬜ todo | First Activity conversion and assessment equivalence proof; duplicate terminate removed |
| fn-140.5 | ⬜ todo | Author docs, core form and layout template witness |
| fn-140.6 | ⬜ todo | Other eligible Models, source-aware triple lint, sealed regeneration diff and equivalence proof |

#### fn-149: Safety and liveness groups for object properties

[Spec](.flow/specs/fn-149-safety-and-liveness-groups-for-object.md) has five M tasks covering all six
criteria. Its refreshed plan passed independent review on 2026-10-10. The independent Go report
lane (.3) is admitted as work-ahead on a pinned integrated baseline; it changes no Scala,
schema or managed artifact. Grouping integration follows fn-140's witness seal, before fault
work, and closure remains shared at fn-123.8.
The report checkpoint is integrated as `da8f78204a`: classification is derived, and progress
receipts expose their declared `within` separately from exploration limits. Focused integrated
checks preserve witness verdicts and independent safety failures; the full checker gate remains open.

Split authored claims into `properties.safety` and `properties.liveness`, enforce declaration kinds,
and carry the distinction into existing diagnostics and reports. Liveness retains explicit bounds
and assumptions. Runtime registration and lifted discovery have independent inventory proofs;
shared capability-law counts cannot use the candidate scans as their oracle. Complete grouping
comparisons preserve behavior, check results and Case/assessment meaning, with every identity and
provenance change explained. Fn-149.5 stays pending until shared gates, review and live evidence are
linked at fn-123.8, even after its committed source seal releases fault implementation.

| Name | Status | What |
| --- | --- | --- |
| fn-149.1 | ⬜ todo | Group discovery and independent runtime/lifter registration proof |
| fn-149.2 | ⬜ todo | Declaration-kind placement checks and source-attributed refusals |
| fn-149.3 | 🔄 in progress | Reporting checkpoint integrated; focused classification, distinct bounds and safety/progress coexistence checks pass; full gates/review held |
| fn-149.4 | ⬜ todo | Model/capability migration with complete independent behavior and identity seal |
| fn-149.5 | ⬜ todo | Author docs and final grouping seal before fn-123; shared gate/review/live close at fn-123.8 |

Tasks .1 and .3 are parallel candidates; .2 follows .1; .4 joins .1/.2/.3; .5 joins .3/.4.
Heavy commands remain serialized. Composition progress belongs to fn-150, not this grouping change.

#### fn-123: Declare faults as the environment's actions

Ready 2026-10-07. Batch implementation around shared fault/durability interfaces; disjoint consumers
and exports may run in parallel. Task 3
is the proof: Go derives crash rows equal to `crashDetail`'s from a durability classification, or the
work stops for the owner. Planning took defaults for five owner questions, listed in the spec's Open
Questions.

A fault is declared once with its kind, budget and whether it can be realized; a machine classifies
its fields as durable or in-memory, and Go derives the crash. The explorer half of R7 and the P clause
of R8 were dropped, since neither tool exists.

| Name | Status | What |
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
[Testpilot schema research](.plans/TESTPILOT_SCHEMA_RESEARCH.md). Implement fn-146, fn-147 and
fn-148 as one coordinated breaking migration, in parallel with fn-156. Reconcile authoring batch 2's
schema and vocabulary at final integration rather than waiting for its closure to start source work.
Assign shared schemas and generation to one owner; delegate runtime consumers and Umpire lowering
against that interface. Fn-141 integrates after this batch; other overlapping work uses explicit
file ownership. Deferred specs are not revived by this scheduling change.

Owner decision: breaking IR changes are allowed. Producers, consumers, generated artifacts and
recorded Case/Run companions migrate together. No compatibility decoders, legacy evaluators or
parallel old field spellings are required; retired formats reject explicitly. The declaration-only
split still proves semantic and artifact equivalence, not a historical compatibility promise.

That decision lets the three specs share one breaking format: fn-146.1's migration contract is the
contract for all three, and fn-148.6 activates format 4.0 once, carrying fn-146's and fn-147's
changes. fn-146.7 and fn-147.4 retire their machinery, categorize their identity changes and write
their docs; Run companions are regenerated once, at fn-148.7, with one full gate and review. Each
spec's proof task (fn-146.4, fn-147.1, fn-148.1) retains separately attributable evidence, but does
not require a separate implementation wave, full gate or review.

#### fn-146: Adopt CEL for runtime predicates and values

Testpilot owns the restricted CEL environment and descriptor-aware value adapter;
Umpire lowers symbolic realization operands into it. Finite Model expressions, `ModelValue` and
descriptor-exact `ValueType` remain separate. Formats and identities move together, without legacy
runtime paths. Batch admission, value adaptation, execution and Umpire lowering against one shared
interface; overlapping edit surfaces have one owner, not one global serial task queue.
On 2026-10-10 the owner authorized this Testpilot implementation lane in parallel with fn-156.
Task .1 can establish its format/identity boundary against the integrated local baseline.
Authoring predecessors remain final integration holds, especially for .6; format 4.0 activation
still belongs exclusively to fn-148.6. CEL adoption is decided; no additional prototype is planned.

The coordinated Go/schema/generated-API source checkpoint is integrated as `49f6809ae9`.
The corrective native-input work/limit, lint and precise module-import checkpoint is integrated
as `823db65754`; the worker's final changed-package lint and focused checks passed. The final
combined tree passed 52 focused top-level tests across six packages, including native captured
message offline admission, payload limits, schema/ownership, fairness and claim reporting.
Typed Run Event payload response-byte/fanout admission is an explicit format 4 source-policy
delta: only fixed descriptor-derived canonical TypeURL overhead is compensated, never above the
unchanged global hard cap; empty native payloads remain admitted at response cap 1. Final corpus
and default/ceiling reconciliation remains required. Extension-range protobuf resolver authority
is unproven and remains an acceptance blocker. Scala proposals are unapplied; managed artifacts,
full gates, live corpus, final cost measurements and shared review remain open.

| Name | Status | What |
| --- | --- | --- |
| fn-146.2 | 🔄 in progress | Canonical CEL AST, restricted admission, pinned engine bridge and budgets |
| fn-146.3 | 🔄 in progress | Standard CEL values with authoritative descriptors, exact numbers and opaque `Any` |
| fn-146.4 | 🔄 in progress | Native execution and Driver/worker values; descriptor/capture and online/offline proof |
| fn-146.5 | 🔄 in progress | Contract and correlated verification, rule expansion and reference walkers |
| fn-146.6 | 🔄 in progress | Umpire operand lowering, Run Event guards and conformance agreement |
| fn-146.7 | 🔄 in progress | Retire custom machinery, categorize identities, docs; close (companions and full gates at fn-148.7) |

#### fn-147: Migrate elapsed-time fields to protobuf Duration

Integration dependency: reconcile with fn-146.6; implement alongside CEL under the shared schema
owner. Seven Testpilot elapsed-time fields and corresponding Umpire hints/defaults
migrate through checked whole-millisecond conversions. Absent polling interval means one read;
present positive interval means polling. Counts, logical bounds, percentages and timestamps stay.
Scalar-only singleton-oneof presence cleanup belongs here. Batch schema, producer and consumer changes.

| Name | Status | What |
| --- | --- | --- |
| fn-147.1 | 🔄 in progress | Inventory, exact conversion bounds, defaults, presence and monotonicity; proof |
| fn-147.2 | 🔄 in progress | Replace schemas and migrate Scala authoring, lifting, realization admission and producers |
| fn-147.3 | 🔄 in progress | Runtime consumers, polling policy and scalar presence |
| fn-147.4 | 🔄 in progress | Categorize identities, docs; close (artifacts, Run companions and full gates at fn-148.7) |

#### fn-148: Consolidate Testpilot evidence and correlated state schemas

Integration dependency: reconcile with fn-147.3; implement alongside CEL and Duration rather than
waiting for separate spec completion. One generalized evidence declaration replaces inline extraction. Response lifts
keep ordered first-match behavior; Run Event overlaps still fail. Contract evidence policies remain
independent. Complete states include both atom and fields; results are separate from authorized
prior-state transitions, and projection result order stays explicit. Batch the migration around
shared schema/binder contracts, with one coordinated activation and final checkpoint.

| Name | Status | What |
| --- | --- | --- |
| fn-148.1 | 🔄 in progress | Generalized evidence declarations in a leaf schema; full lift-shape proof |
| fn-148.2 | 🔄 in progress | One binder and lowering path; source selection and independent Contract policies |
| fn-148.3 | 🔄 in progress | Complete-state and result tables in schema/lowering; compact-size measurement |
| fn-148.4 | 🔄 in progress | Normalized admission/verification, causal authorization and expanded-work ceilings |
| fn-148.5 | 🔄 in progress | Derived Contract kind, fixed correlated clock and explicit support Boolean |
| fn-148.6 | 🔄 in progress | Local references, cardinality, Empty markers and coordinated format 4.0 activation (the batch's single activation) |
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

| Name | Status | What |
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

**Paused (owner, 2026-10-10):** remaining activity edits wait for Batch 1d's activity lane, then
finish on the new DSL before Batch 5b (see Batches 1d and 5b below).

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

| Name | Status | What |
| --- | --- | --- |
| fn-128.7 | ⏸️ paused (1d) | Accepted fatal-settlement classification through typed public last-failure evidence; unchanged fatal Property satisfied in scratch recording/replay |
| fn-128.8 | ⏸️ paused (1d) | Separate caller-owned pauseResume and heartbeat-retry Profiles; immutable preflight/execution/replay binding, final-shape charge proof and admission negatives |
| fn-128.6 | ⏸️ paused (1d) | Evidence map, live Cases run once (the batch's live run); close |

#### fn-138: Retries and Deadline capabilities

All three tasks are done; whole-spec review and shared Batch 5 closure are pending, so no task
rows remain. The done source tasks do not replace the production equivalence, full gates or live-run evidence
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


#### fn-129: Activity coverage

Runs after fn-138. Tasks run in order.

| Name | Status | What |
| --- | --- | --- |
| fn-129.3 | ⏸️ paused (1d) | Reset with `keepPaused` and deferred settlement joined: 65 Queries, `deferredResetCompletes` lowers and prepares, reset-aware laws via `overriding(…, of = …)`; open: direct `keepPaused` live Case (lowering refuses a reset indistinguishable from the pause before it) |
| fn-129.4 | ⏸️ paused (1d) | Exploration on the activity's `find` Queries |
| fn-129.5 | ⏸️ paused (1d) | New Cases listed, live run (the batch's live run); close |

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

| Name | Status | What |
| --- | --- | --- |
| fn-150.1 | ⏸️ deferred | Scoped admission/checker checkpoint integrated; original Quick OOM retained, review and closure held |
| fn-150.2 | 🔄 in progress | Finite structured fairness checkpoint integrated; typed catalog, inheritance and exact member/sync pins pass; Scala/full acceptance held |
| fn-150.3 | ⬜ todo | Typed composition progress and fairness authoring/lifting |
| fn-150.4 | 🔄 in progress | Boundary, typed replay and structured fairness checkpoints integrated; focused proofs retained, full gates/review remain |
| fn-150.5 | ⬜ todo | Temporal positive/negative example, docs and integrated gates |

Integration order: `.1` then `.2`; `.3` and `.4` join before `.5`. Source work may overlap
under the batch-first strategy rather than waiting for per-task closure.
Fn-150 has no hard dependency on fn-149. Its hand-authored Go checker proof changes no Model source,
schema or managed artifacts and is disjoint from fn-155's remaining fixture/test work. Serialize
later overlapping edits and regeneration with the approved delivery chain and re-anchor their
paths before dispatch. This admission does not release fn-156's fn-155 closure dependency.

Task .1's original Quick remains RED. Its source checkpoint is integrated as `9c47f975a0`,
with task .4's boundary/replay checkpoint as `d01aefebf1`; neither integration grants review,
gate or completion credit. The Flow block reason retains the original verification obligation.
Task .2 is admitted on this pinned baseline with a recorded work-ahead exception. Its finite
assumption/progress schema leaves do not overlap the format lane's realization/script schemas;
compiler-owned Scala packaging and typed authoring wait for the compiler-only equivalence seal.
The conductor's integrated task .4 Quick is recorded under `.flow/tmp/fn150/integrated/`.
That run exited 1: the engine package passed, but the unchanged Activity property-row checker
hit the canonical 30-minute timeout during `startedByPollingWorker_is_about_the_backoff`.
The checker package elapsed 1,846.865 seconds. Its terminal log and result are retained;
this is not a restored full gate. Structured fairness source is integrated through `bd4511d8b1`,
preserving both CEL/Duration and finite fairness schema inventories after a serial join repair.
The final combined focused check passed; the join's inherited CEL lint failure and exact restoration
of its unowned autofixes remain recorded separately. Scala closure, managed source-position
reconciliation, canonical full checks and review remain open. Work is paused at the owner's request,
and fn-157's owner deferral remains in force.

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

| Name | Status | What |
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

| Name | Status | What |
| --- | --- | --- |
| fn-119.3 | ⏸️ deferred | Workflow-scheduled activities and awaited outcomes in the realization DSL, lifter and lowering |
| fn-119.4 | ⏸️ deferred | Activity workflow example modeled; its Queries run live from the gate |
| fn-119.5 | ⏸️ deferred | Faulty variant and the zero-Go check |
| fn-119.6 | ⏸️ deferred | Walkthrough, one-command entry point; close |

### fn-125: Represent dynamic configuration in the Models

Deferred 2026-10-05. Evidence: `.plans/DYNAMIC_CONFIG.md`.

| Name | Status | What |
| --- | --- | --- |
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

| Name | Status | What |
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

| Name | Status | What |
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
- Batch integration order retains Flow's predecessor dependencies; the batch-first strategy above
  permits documented parallel implementation ahead of those closures. Do not restore per-spec
  regenerations or task-by-task review barriers. The activity model batch integrates after the
  other feature batches at the owner's request, followed only by Batch 5b (fn-159–fn-162's single
  regeneration; owner, 2026-10-10; their Scala changes land earlier, in Batch 1d): until it closes, batches 2 and 3's live runs include activity Cases with
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

| Name | Status | What |
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

## Batches 1d and 5b: Scala DSL now, one regeneration last (fn-159–fn-162)

**Owner decision (2026-10-10):** the Scala DSL changes go in immediately. The IR, Cases, Go
consumers and other downstream artifacts may stay broken until one final regeneration. So the four
Model-readability specs split by language, not by spec:

- **Batch 1d, Scala DSL (immediate; shared close fn-161.1):** every Scala change of fn-159–fn-162
  (framework, lifter rules, model migrations, renames, file layout) lands now, in dependency order,
  ahead of the other batches. Checked-in IR and Cases go stale and the model gate fails. That is
  expected and is not a regression to fix inside 1d.
- **Batch 5b, one regeneration (last; shared close fn-160.2):** the IR schema change, the lowering
  and Go assessment of realization-to-Query bindings, the Go and fixture consumers of renamed ids and moved files,
  then **one** regeneration of IR, Cases, lint acceptances and pinned fixture trees, an IR-level
  equivalence check under the batch's rename map, full gate, review and one live run. It waits for
  Batch 3 (which owns the IR/Case schema) and integrates after Batch 5.

**How 1d is verified without IR:** before the first 1d edit, `make umpire-capture-baseline` captured
the baseline at `.flow/tmp/fn160/baseline-ae3f36ee51/` (59 step tables, IR, Cases, interpreter dumps).
Each 1d change seals with `make umpire-check-equivalence BASE=<baseline> MAP=<batch map> STEPS_ONLY=1`,
plus the Scala build, framework tests and model tests; the lifter builds again, so its fixtures run too.
Batch 5b's IR equivalence check is `make umpire-check-equivalence BASE=<baseline> MAP=<batch map>
DUMPER=<fn-155 dump>`, which adds the IR, lint, Case and manifest projection and the interpreter tables.
The tasks' `make umpire-check-model` / lint / fixture criteria are met by Batch 5b's shared close
evidence, as the batch rules allow.

**Interactions with other batches:**
- **Batch 4 (fn-141, not started)** now comes after 1d, so it describes only the final DSL (qualified
  forms, `enters`, facets, per-feature collection, realization-to-Query bindings).
- **Batch 2 (authoring):** fn-140 and fn-162.2 are co-designed by one owner (owner, 2026-10-10), since
  both define what a Query says. Batch 2's remaining Scala work lands on the 1d DSL.
- **Batch 5 (activity model) is paused (owner, 2026-10-10)** for its remaining activity edits (fn-129's
  in-progress task and its two todo tasks, fn-128's three todo tasks) until Batch 1d's activity lane
  lands. They then finish on the new DSL (qualified names, `enters`, facets, realization-to-Query bindings,
  flat layout) before Batch 5b. Work already done stays; Batch 1d's activity lane migrates it.

**Batch 1d order (Scala only):**

Rows run in order; fn-160.1, fn-162.1 and fn-160.5 (baseline and step-table comparison) are done.

| Name | Status | What |
| --- | --- | --- |
| fn-159.1, fn-159.3, fn-159.6 | ⬜ todo | Qualified `on` targets and phases (framework runtime check for `on`, lifter rules written), `activityWorker`; `Observation` drops `read` |
| fn-159.2, .4, .5, .8 | ⬜ todo | `completionClient`; drop the product's `disabled(process.stop)`; bounds into `Bounds`; `Retryable`/`Expirable` |
| fn-159.7 | ⬜ todo | `enters(Phase.X)` construct and its migration in every model |
| fn-161.1 | ⬜ todo | Facet construct, ownership and vocabulary checks |
| fn-161.2 + .3, fn-161.4 | ⬜ todo | Activity facets in one pass; every other machine wrapped |
| fn-162.2 (Scala part), fn-162.3 | ⬜ todo | Realizations name the Queries they run; claim-container subjects removed; Properties declared once |
| fn-160.3, fn-160.4, fn-160.6 | ⬜ todo | One object per file, flat prefix layout, `realization/` layer, `<Subject>Run`, one-way layer dependencies; System level renamed Protocol |
| fn-160.2 (Scala part) | ⬜ todo | Collection by feature package; `irFile`/`exports` removed |

**Batch 1d lanes** (disjoint files run in parallel; only the activity lane is a chain):

| Lane | Tasks | Files |
| --- | --- | --- |
| Framework and lifter | fn-159.1/.3/.6/.7 constructs, fn-161.1, fn-162.2 (Scala), fn-160.2–.4 rules | `model/framework`, `model/irgen`, `model/check` |
| Activity | fn-159 migrations → fn-161.2+.3 → fn-162.3 → fn-160.3/.4 moves | `features/activity/` |
| Nexus | fn-159 migrations → fn-161.4 (nexus) → moves | `features/nexus/` |
| Task queue and worker | fn-159 migrations → fn-161.4 parts → moves | `foundations/`, `actors/` |

**Batch 5b order:** fn-162.2's IR schema, lowering and Go assessment (with fn-140's owner); fn-160.5's
IR projection; Go tests, canary harness and fixture trees updated for renamed ids and moved files;
then fn-160.2 closes with the single regeneration, IR equivalence check, full gate, review and live run.

**Decisions (conductor, 2026-10-10, under the owner's recommendation mandate):** capability kinds
`Retries` → `Retryable` and `Deadline` → `Expirable`; every bound in `Bounds`, the close policy's wider
budgets as `fourWide`/`fiveWide`/`twelveWide` and nexus `control` merged into `eight`; `disabled(...)`
targets are qualified like `on`; each feature lane picks its facet vocabulary and `<Subject>Run` names
from the spec guidance and records them in fn-161/fn-160.

Flow refuses task dependencies across specs, so the cross-spec order above is held here and by
the conductor. Within each spec, Flow records it.

## fn-159: Qualified action names in rules

Readability and naming cleanups across the Models. Rule targets and phases are always qualified
(`on(activityWorker.poll)`, `when(Phase.scheduled)`), and `from(X)` only groups (fn-159.1, fn-159.3).
Actors get clearer names (`activityWorker`, `completionClient`), as do the capability kinds `Retryable`
and `Expirable` (fn-159.2, fn-159.8). Phase-only effects give way to `enters(Phase.X)` (fn-159.7). The
product's never-enabled `disabled(process.stop)` binding goes, unless the lift needs it (fn-159.4), every `Limits` bound moves into
`Bounds` (fn-159.5), and `Observation` drops its unchecked `read` text (fn-159.6). fn-159.1, fn-159.3
and fn-159.6 leave the IR unchanged; the renames, bound moves and `enters` change ids or function names
only. The IR-regenerating tasks wait on the tree's IR `Duration` migration. Not marked ready.
[Spec](.flow/specs/fn-159-qualified-action-names-in-rules.md).

| Name | Status | What |
| --- | --- | --- |
| fn-159.1 | ⬜ todo | Require qualified `on` targets in irgen and the framework; refuse `import X.*` in `from`; rename the bound `worker` to `activityWorker`; migrate models, tests and README |
| fn-159.2 | ⬜ todo | Rename the standalone activity's By-ID actor `service` to `completionClient`; regenerate the activity IR |
| fn-159.3 | ⬜ todo | Qualify every phase reference (`when(Phase.scheduled)`, `phase = Phase.started`); irgen refuses phase-member imports |
| fn-159.4 | ⬜ todo | Remove `disabled(process.stop)` from the activity product; needs the tree's IR `Duration` migration to build for the lift check |
| fn-159.5 | ⬜ todo | Move every `Limits` bound into `Bounds`; resolve the clashing names `four`/`five`/`twelve` and merge `eight`/`control` |
| fn-159.6 | ⬜ todo | Drop `Observation`'s free-text `read`; name the response field beside each realization's typed read |
| fn-159.7 | ⬜ todo | `enters(Phase.X)` framework step replaces the ~29 effects that only set a phase (after fn-159.1, fn-159.3) |
| fn-159.8 | ⬜ todo | Rename capability kinds `Retries` → `Retryable`, `Deadline` → `Expirable` (names to confirm); IR capability ids change |

Open: whether `disabled(...)` targets fall under the same rule.

## fn-160: One IR file per feature, collected from the feature's package

Replace the hand-written `irFile` root lists, three files each for the standalone activity and nexus
workflow, with one IR file per feature folder, collected from the feature's package. Nothing is listed,
so no section can be forgotten. The model content stays the same; file boundaries and derived names
(Cases, lint acceptances, export slices) change. Each Model file then holds at most one machine, in flat
level folders grouped by name prefix, and realizations get their own `realization/` layer. Not marked ready.
[Spec](.flow/specs/fn-160-one-ir-file-per-feature-collected-from.md).

| Name | Status | What |
| --- | --- | --- |
| fn-160.2 | ⬜ todo | Collect one IR file per feature package, and per foundation/actor/kind package that declares Queries or realizations; remove `irFile`/`exports`; exploration checks only the asked Query; regenerate and migrate consumers |
| fn-160.3 | ⬜ todo | One machine per file, named after it; flat level folders grouped by name prefix (no nested packages); the lifter enforces it (after fn-160.2) |
| fn-160.4 | ⬜ todo | Realization layer: flat `realization/` folder, `<Subject>Run` naming, one-way layer dependencies, API only in realizations (after fn-160.3) |
| fn-160.6 | ⬜ todo | Rename the System level to Protocol (folders, packages, machines, files, lint vocabulary, docs), so the levels read Product / Protocol / Realization |

## fn-161: Facets: action-owning behaviour areas of a feature

A feature's machine is assembled from **facets**. Each facet is a feature-local behaviour area (Pausing,
Cancellation, Reset, …) that owns a set of actions and holds their rules, effects, claims and the
capabilities it declares. Each action has exactly one owning facet; cross-facet step use is declared in
the facet's type. Only facets define rules, effects and capabilities in a declared machine, and each
feature's levels share one facet vocabulary. Facets are named as nouns, capabilities as adjectives. Starts with the standalone
activity, in two stages, after fn-159.7 (`enters`). Not marked ready.
[Spec](.flow/specs/fn-161-facets-action-owning-behaviour-areas-of.md).

| Name | Status | What |
| --- | --- | --- |
| fn-161.1 | ⬜ todo | Facet construct: framework, lifter, one-owner-per-action check, layout rules, docs |
| fn-161.2 | ⬜ todo | Stage 1: standalone activity effects and claims into facet files (rules stay by actor) |
| fn-161.3 | ⬜ todo | Stage 2: standalone activity rules and capabilities into their owning facets |
| fn-161.4 | ⬜ todo | Wrap the other 11 declared machines into facets; switch facets-only and facet-vocabulary rules on globally |

## fn-162: Queries name their realization; each Property declared once

The activity's 17 claim containers (copies of a machine that change nothing, e.g. the three in
`Heartbeat.scala`) exist because the lowering and the assessment need exactly one realization per
machine. A copy is how a Query picks its realization, or has none. fn-162.1 found a Case is
already assessed against only its Query's one Property (plus the machine's monitors). The fix: each
realization names the Queries it runs (unnamed find Queries stay model-only), and each Property is
declared once in its facet, under its existing name, so the copies go. Query answers and Case outcomes stay the same; machine-scoped ids move
(in the rename map). Ideally lands before fn-160.3 splits those subjects into files. Not marked ready.
[Spec](.flow/specs/fn-162-queries-name-their-realization-each.md).

| Name | Status | What |
| --- | --- | --- |
| fn-162.2 | ⬜ todo | Realizations name the Queries they run (unnamed find Queries are model-only): framework, lifter, IR, lowering, assessment; duplicate-Property detection |
| fn-162.3 | ⬜ todo | Migrate the standalone activity off claim-container subjects |

## fn-163: System capabilities: distributed-systems patterns as named capabilities

Scheduled after Batch 5b (owner, 2026-10-10), on the facet, realization-binding and lifter baseline.
Not planned yet. [Spec](.flow/specs/fn-163-system-capabilities-distributed-systems.md).

Name the distributed-systems promises the Models still write by hand (fencing of stale attempts, a
single active attempt, idempotent requests, a single outcome) as first-class capabilities, declared once
with laws that lower to the existing IR. Capabilities split into `capabilities/system/` (server-side
protocol promises) and `capabilities/product/` (what a user sees an entity can be). A pattern is named
even with a single use (owner): the name improves readability and is easy to revise when more uses
arrive. Idempotency targets actions rather than entities, and fencing/single outcome need history, so
some laws may need synthesized monitors; verdicts, answers and Case outcomes stay unchanged under a
checked rename map.
