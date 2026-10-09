# Umpire4 milestones

The current state of Umpire work: what is being built, what is left, and what is not being done.
Flow (`.flow/`, `flowctl`) is the record for specs and tasks; this page is the overview across them.

As of 2026-10-09.

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
  lower, model and export test binaries take up to about 6.5 GB each (export), so never more than
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
  batch. The activity model batch moved last (2026-10-08), so batch 1 has no batch predecessor and
  batch 5 waits on batch 4.
  The next batch's preparation may run in isolated worktrees alongside the
  current one, but joins serially onto the closed baseline.

| # | Batch | Specs | Nature | Shared close | Live run |
| --- | --- | --- | --- | --- | --- |
| 1 | Structural moves | fn-142 → fn-143 → fn-145 | equivalence | fn-142.1, fn-143.1, fn-145.4 | none |
| 2 | Authoring | fn-140 → fn-123 | equivalence sealed, then meaning | fn-140.6, fn-123.8 | once |
| 3 | Testpilot format | fn-146 → fn-147 → fn-148 | breaking format | fn-146.7, fn-147.4, fn-148.7 | once |
| 4 | Lifter | fn-141 | byte-identical | fn-141.14 | none |
| 5 | Activity model | fn-128 → fn-138 → fn-129 | meaning | fn-128.6, fn-129.5 | once |

Batches 1 to 4 replace eight per-spec regeneration and full-gate cycles (and three Run-companion
migrations) with four batch closes and one companion migration.

## Direction

The Scala model (`model/`) is the model. Lean is the past, and nothing has to look like it.
Scala declares, a lifter reads the declarations into the Umpire IR, and generic Go consumers check
that IR, lower it to Testpilot Cases, and run those Cases as functional tests and canary checks. The
Umpire IR and the Testpilot IR are what connect the parts. See [SCALA.md](.plans/SCALA.md) and
[UMPIRE4_SPEC.md](.plans/UMPIRE4_SPEC.md).

The DSL framework (`model/framework`) stays Temporal-agnostic as far as is realistic: Temporal's capability
vocabulary, properties, realization vocabulary and kit live under `model/temporal/`, and the lifter and Testpilot IR
are the parts that are Temporal's driver tooling by design (fn-114.12, fn-122.8).

The planned work below continues that direction: the Scala layer first, then the Models.

## Specs

Listed in delivery order, grouped by batch. Flow records spec and task dependencies; the conductor
also holds batch-close gates and serializes work that shares a regeneration baseline.

### Batch 1, structural moves: fn-142 → fn-143 → fn-145

Starts from the current `umpire` baseline, which includes NewProducer's shared-table speedup
(68.7 s → 9.6 s per call). The activity batch's joined tasks are in the tree but
production `model/ir` and `model/cases` were not regenerated for them, so the equivalence gate compares
scratch lifts before and after the moves, not the checked-in IR, and fn-145.4's regeneration also
carries the joined activity changes; attribute those to their activity tasks. All three change
layout, not meaning: the IR differs only in paths, positions, the `umpire.`→`framework.` package
mapping and the schema's file layout. One regeneration and one equivalence gate prove the combined
diff is exactly those mappings; one full Go suite and one review close the batch at fn-145.4. No
live run is needed. This batch contract supersedes fn-142's original production-regeneration and
full-gate acceptance: fn-142 closes on its scratch equivalence and focused layout verification.

fn-142's isolated move and fn-145.1 to .3 (protobuf files, descriptors and the Scala jar exclusions)
were prepared in isolation (fn-145.1–.3 on `agent/fn145-prep` in `../lane-batch2`); they join serially in the order
fn-142, fn-143, fn-145, and each re-anchors its paths to the moves before it. The fn-142 join dry-run
found a milestones-file conflict; preserve its isolated work and serially rerun the move onto the
current baseline.

Moving fn-145 ahead of fn-140 and fn-123 means both author against the modular schema, so fn-123.1's
IR `Fault` record lands in its responsibility file instead of being moved later.

#### fn-142: Split `model/temporal/shared` into `foundations` and `actors`

Mechanical move: the task queue goes to `foundations/taskqueue`, the worker to `actors/worker`,
`Client.scala` to `actors/client/`, and `Bounds.scala` to `model/temporal/`; the IR differs only in
paths and positions.

| Task | Status | What |
| --- | --- | --- |
| fn-142.1 | ✅ done | Layout-only move and scratch equivalence passed; production IR/Case regeneration, the full Go suite and integrated batch review remain at fn-145.4 |

#### fn-143: Rename `model/umpire` to `model/framework`

Source gate: after fn-142.1. Folder and package both become `framework`; product names (`tools/umpire`, `umpire-*` targets, `umpire.v1`) stay. The IR differs only in paths, positions, framework qualified names and implied fingerprints.

| Task | Status | What |
| --- | --- | --- |
| fn-143.1 | ✅ done | Moved the framework and Scala package; scratch equivalence and focused source/lifter/Go checks passed, with production regeneration and the integrated batch gate remaining at fn-145.4 |

#### fn-145: Modularize the Umpire IR schema

Source gate: after fn-143.1 joins. Nine files, one existing protobuf package and an acyclic import
graph. This owns fn-131's realization extraction slice; fn-131 retains metadata, canonicalization,
level checks and producer provenance. No declaration is removed merely because generated production
Models omit it. Shared settings and disposition/cleanup leaves receive a maintenance evaluation, not
an assumed shared schema. Planned 2026-10-06 from the schema research (see batch 3).

| Task | Status | What |
| --- | --- | --- |
| fn-145.1 | ✅ done | Full schema closure, linked descriptors and Scala jar exclusions passed their focused generation, compile and descriptor proofs |
| fn-145.2 | ✅ done | Extracted the nine responsibility files; generation, descriptor-union and closure checks pass without a semantic delta |
| fn-145.3 | ✅ done | Replaced the local marker with `google.protobuf.Empty`; wire, JSON, presence, identity and retired-name checks passed |
| fn-145.4 | 🔄 in progress | Production artifacts and exact scratch equivalence passed; exact lint refresh applied, fixture repairs and row-key allocation optimization under verification; no-update Model gate running, full Batch 1 gates and inherited activity assessment failures remain open |

### Batch 2, authoring: fn-140 → fn-123

Starts from batch 1's closed baseline, with task paths re-anchored to the completed moves. Both specs
rewrite Model declarations, so they share one production regeneration, one full gate, one review and
one live run at fn-123.8. fn-140 is meaning-preserving apart from its declared renames: fn-140.6
seals its assessment-equivalence comparison before fn-123 changes meaning, so a fault change can
never hide inside the witness migration.

Source gate: fn-123 implementation starts after fn-140.6's seal. fn-123 is written against the names
fn-140 leaves (`when` blocks, `.live`, the shared `Outcome`, the fault instruction and typed `perform`).

Batching candidate: fn-149 rewrites the same `properties` sections. If the owner schedules it, it
joins this batch between fn-140 and fn-123 (its fn-149.4 migration and fn-140.6 touch every Model
once instead of twice), with its own equivalence seal.

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

| Task | Status | What |
| --- | --- | --- |
| fn-146.1 | ⬜ todo | Breaking format, deterministic canonical identity and companion migration contract (the batch's contract) |
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
Tasks run in order; 1 and 2 depend on nothing, and 6 and 7 need only them. Task 5 is the proof for the
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
and `cancelIsRequested` is Nexus's own claim. Until the batch regeneration, the carriers and source-position
checks fail on stale production IR.
Current-source replay exposes three inherited assessment failures that must be resolved before
closure: ordinary worker completion is observationally ambiguous with by-ID completion
(`completes` is `inconclusive(never_evaluated)`), the non-retryable failure claim has a visibility
ambiguity, and pause/resume exceeds its correlated per-event work ceiling. Batch 1's full Go gate
reproduces all three; its structural equivalence proof does not discharge them. None is waived or
counted as passing evidence. Tasks fn-128.7 and .8 own the latter two corrections; the completion
ambiguity has no scheduled correction yet.
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

## Planned, not yet scheduled

### fn-149: Safety and liveness groups for object properties

[Spec](.flow/specs/fn-149-safety-and-liveness-groups-for-object.md) planned with five M-sized tasks
covering all six acceptance criteria; not marked ready. Split authored claims into `properties.safety` and
`properties.liveness`, enforce declaration kinds, and carry the distinction into existing diagnostics
and reports. Liveness retains explicit bounds and assumptions. Migrate Models, shared laws and docs
while preserving behavior and check results.

Coordinate with fn-140's property/Query authoring changes and fn-141's declaration lifting changes.
Scheduling remains open; batch 2 names it as a batching candidate.

Cross-machine safety uses compositions today. New composition progress support is tracked separately
in fn-150; it is not a prerequisite for fn-149's grouping.

| Task | Status | What |
| --- | --- | --- |
| fn-149.1 | ⬜ todo | Group discovery and registration equivalence proof |
| fn-149.2 | ⬜ todo | Declaration-kind placement checks and source-attributed refusals |
| fn-149.3 | ⬜ todo | Derived safety/liveness classification in existing reports |
| fn-149.4 | ⬜ todo | Model and capability migration with behavior/identity pins |
| fn-149.5 | ⬜ todo | Author docs, integrated artifact checks and final gates |

Waves: `.1` and `.3` are parallel candidates; `.2` follows `.1`; `.4` follows `.2` and `.3`;
`.5` joins `.3` and `.4`. Shared generation and test resources still require serialization.

### fn-150: Bounded liveness across composed machines

[Spec](.flow/specs/fn-150-bounded-liveness-across-composed.md) planned with five M-sized tasks covering
all five acceptance criteria; not marked ready. Let a composition own bounded progress claims over multiple
member states. Count composed steps, resolve fairness against synchronized and member-only actions,
and preserve deadlock, cycle, deadline and incomplete-check distinctions. Include one concrete
Temporal composition with passing and negative examples.

Related to fn-149's safety/liveness groups. Coordinate with fn-141 and the queued schema changes;
execution remains unscheduled. Live Case generation for compositions is outside this spec.

| Task | Status | What |
| --- | --- | --- |
| fn-150.1 | ⬜ todo | Composition progress admission and existing-checker proof |
| fn-150.2 | ⬜ todo | Structured fairness references, inherited/replacement mapping |
| fn-150.3 | ⬜ todo | Typed composition progress and fairness authoring/lifting |
| fn-150.4 | ⬜ todo | Bound, fairness, holes, starts and witness-replay regressions |
| fn-150.5 | ⬜ todo | Temporal positive/negative example, docs and integrated gates |

Waves: `.1` then `.2`; `.3` and `.4` are parallel candidates; `.5` joins them.
Neither spec has a hard dependency on the other. Serialize their overlapping edits and regeneration
with each other and the approved delivery chain, and re-anchor paths after completed migrations.
If both are scheduled together, they form one batch: one regeneration and gate for both.

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

Deferred 2026-10-05 before task planning; the spec has no tasks yet. When revived, starts after fn-126 closes. Evidence: `.plans/MODEL_VISUALIZATION.md`.

Rendered views per Model (signature, phase diagram, refinement, compositions, derived-design diff, witness paths), checked in as `.d2` plus `.svg` under `model/views/` and gated; D2 as a Go library with ELK; no DSL declaration.

### Other deferred items

- fn-122.7 (the Pausable capability Property on fn-119's example) waits for fn-119.
- `make umpire-check-backends` in CI; it runs locally after `make umpire-install-backends`.
- The IR explorer (fn-120.4) was removed. fn-112, fn-114, fn-118, fn-120, fn-121, fn-122 and fn-127 are closed.

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
  its two inherited failures (non-retryable failure visibility, pause/resume work ceiling), which those
  runs report but do not own.
