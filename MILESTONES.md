# Umpire4 milestones

The current state of Umpire work: what is being built, what is left, and what is not being done.
Flow (`.flow/`, `flowctl`) is the record for specs and tasks; this page is the overview across them.

As of 2026-10-15.

**Current execution boundary (owner, 2026-10-14):** focus on framework code under `model/` and its six samples. `model/temporal/`, `tools/umpire/` and their integration/Go checks are held for now. Continue the fn-174 compiler-plugin foundation with framework-only Scala checks. Mini/Temporal equivalence, full scoped gates and downstream propagation remain open obligations; framework-only checkpoints do not establish their completion.

**Current goal (owner, 2026-10-12): complete the mini Scala DSL, then propagate its IR and Go changes end to end.**
The source phase covers all remaining non-held work under Open: Scala DSL work, including fn-167's
internal metadata, derivation and pure migration tasks. Source tasks use mini-focused checks;
broad Scala validation, independent review and a requirement-by-requirement audit run at source
batch end. Only after that boundary does the respective mini IR/schema/Go Umpire propagation,
regeneration and consumer validation begin. The goal remains active through both phases.

**Scope (owner, 2026-10-11): `activity_mini` is the only feature.** The Scala DSL work changes the
framework, the lifter, the shared code (foundations, actors, capabilities, `Bounds`) and
`features/activity_mini`, and nothing else. The real features (the standalone activity, the Nexus
workflow and the Nexus standalone operation, with their `generated/legacy/` payloads) are ignored:
when the DSL changes they are not edited, migrated, lifted or regenerated, and they stay behind until
the owner's sweep. The rules are under [Open: Scala DSL work](#open-scala-dsl-work).

The conductor follows its recommendations for routine decisions. Fn-166's source proving ground
is complete and the remaining Scala batch is active. Its approved Go exceptions remain bounded.
The canonical
Make gate currently fails on owner-held linked protobuf generation and stale real-feature artifacts;
compiled exports also differ from their historical IR inventories. The spec permits the direct
Scala route, whose failures are recorded rather than counted as passes.

**Fast iteration (owner, 2026-10-11):** steps run mini-focused checks and targeted tests for changed
framework/lifter files. Broad validation and independent review run once at the Scala batch end,
not at every task or spec. This supersedes the older per-task/per-spec cadence below; each task
records the deferred gates explicitly, without claiming unrun checks passed or review reached SHIP.
Existing findings and their fixes remain part of the batch review. Go, regeneration and consumer
work stays held until the complete source-phase boundary. Real-feature migration, lint/manual
acceptance, push/PR and production deployment remain outside this run's authority.

**Faster tasks (owner, 2026-10-13):** a task's evidence is its `make model-check-changed` record plus
the mini step-table equivalence; no bespoke proof scripts, hash manifests or immutable captures for
meaning-preserving changes. Workers iterate with only the suites their change touches and run the
full check once at the end, citing the previous task's record as the baseline. The conductor trims
and re-scopes each task to the mini feature and current paths before dispatch, and gives a small
spec's tasks to one worker.

## Keeping this page current

- This page describes the present. Rewrite a status in place; do not append dated entries.
- List each open spec's remaining tasks in a `Name | Status | What` table: the task ID, its status
  (🔄 in progress, ⬜ todo, ⏸️ deferred) and a brief description. Remove a task's row once it is done,
  and the spec once it is closed. Flow and git keep the history.
- Only the root agent edits this page. Workers report status in their handoffs.
- Update the "As of" date with every edit.
- Task/spec branch integration follows `AGENTS.md`'s Local milestone execution policy.

## Direction

The Scala model (`model/`) is the model. Lean is the past, and nothing has to look like it.
Scala declares, a lifter reads the declarations into the Umpire IR, and generic Go consumers check
that IR, lower it to Testpilot Cases, and run those Cases as functional tests and canary checks. The
Umpire IR and the Testpilot IR are what connect the parts. See [SCALA.md](.plans/SCALA.md) and
[UMPIRE4_SPEC.md](.plans/UMPIRE4_SPEC.md).

The framework (`model/framework`) is self-contained and Temporal-free, reusable for other systems
(owner, 2026-10-12, fn-169): it holds the DSL, the generic realization core, the lifter, the checks and the
migration harness, and compiles with no Temporal API or `temporal.*` source on its classpath. Temporal's
capability vocabulary, Properties, realization kit, lifter plug-in, gate and migration rules live under
`model/temporal/`. This supersedes the earlier position that the lifter is Temporal's driver tooling by
design (fn-114.12, fn-122.8); the Go consumers' Temporal adapter and the IR namespace follow at the sweep.

Now, complete the mini Scala DSL. Then propagate its respective IR and Go consumer changes.
Real-feature sources and historical payloads stay behind for the owner's separate sweep.

## Open: Scala DSL work

**The Scala batch is active at the owner's request; fn-166's source lane is complete.**
Completed mini and qualification source checkpoints are integrated into local `umpire`, including
fn-149's grouping source seal (.6/.7, merged from `w-1d`, with the mini claims grouped into `safety`
and `liveness`). The remaining work resumes in dependency order.

**Only the mini feature (owner, 2026-10-11).** Every spec below is scoped to `features/activity_mini`
plus the framework, the lifter and the shared code:

- **Real features are not touched.** A DSL change does not edit, migrate, lift or regenerate the
  standalone activity, the Nexus workflow or the Nexus standalone operation, nor their legacy
  payloads. A change that breaks them is expected and leaves them behind (fn-166). They do not
  constrain a DSL change, and no task proves anything on them.
- **Real-feature tasks are retargeted or deferred.** A task written against a real feature applies its
  change to the mini feature where the mini feature has the construct. What concerns only a real
  feature moves to the sweep; such rows are marked "⏸️ deferred (sweep)".
- **Migrations are recipes, not rules (owner, 2026-10-13).** A DSL change ships a short migration
  recipe (concrete steps for a real feature) in the migrate README instead of a proven Scalafix rule;
  this supersedes fn-166 R6's rule-per-change. The owner's sweep applies the recipes by hand. Rules
  already written stay until the sweep applies them; no new rule is written unless a change is too
  large to migrate by hand.
- **Equivalence uses a mini baseline.** A step-table baseline of the mini feature is captured once,
  before the next DSL change. Per-spec equivalence and the pre-fault seal compare
  `FEATURES=activity_mini` against it. The batch-1d baseline and map stay untouched as the real
  features' "before" for the sweep.
  The native baseline is `.flow/tmp/scala-dsl/mini-selected-baseline-00d22597d2/`: 16 tables,
  11,325 rows from the immutable realization-source jar, including the selected shared query roots.
  The earlier 10-table mini-reachable dump remains unchanged beside it; all ten tables are exact
  members of this complete set. Neither contains a generated Case or live-run baseline;
  `baseline.txt` records the pre-change source/jar provenance and the discovery-scope correction.
- **The batch-end gate is the mini feature's.** It runs with `FEATURES=activity_mini`. A default
  (every-feature) run stays red, naming the real features as behind, until the sweep.

One task owner at a time takes the overlapping model work below from the integrated local tip. Only `model/` is edited
(one exception, granted 2026-10-11: fn-166.1 moves each feature's IR and Cases into its own `generated/` tree and
re-points the Go readers, and safely scopes existing functional/canary pin publication, since a per-feature run would otherwise rewrite unselected Cases)
and validated: Scala compile per step; the framework, Model, lifter and gate Scala tests per spec; and the step-table
equivalence per spec, over `FEATURES=activity_mini` against the mini baseline. The real features' checked-in
IR and Cases stay stale. The equivalence also runs once before fn-123, so a fault or activity change cannot
hide inside the meaning-preserving migrations before it. Rows marked "Go part deferred" finish their Go half later.

**Exception for fn-173 Phases 1–2 (owner, 2026-10-13).** The reachable-exploration redesign's
first two phases are Go engine work, but they run within the Scala batch (order 9, after fn-141).
Every later spec would otherwise prove its change with the step-table equivalence this redesign
retires, and build catalog-based work (totals, budget rules over every catalog state) only to
redo it. They stay mini-scoped: the regeneration covers `activity_mini`, the framework samples and
the fixtures, and the real features remain held for the sweep. Specs whose Go work must follow
Phase 1 are blocked on it in Flow (fn-123.3/.5–.8, fn-150.1, fn-170.2/.3; fn-150.2/.4/.5 are in progress and left to their owner) or depend on fn-173
(fn-128, fn-129, fn-130).

If Make prerequisites require owner-held linked-API generation, record that failure and run the
same Scala `--check-equivalence ... --steps-only` gate directly using independently validated jars
and the immutable baseline's absolute main-checkout path. This does not earn canonical Make credit.
Fn-140.4's pre-edit historical comparison has 34 inherited differences; its required zero-difference
incremental pre/post proof cannot replace reconciliation at fn-140.10 and the pre-fault seal.
Strict native admission failures remain source debt, separate from deferred Go assessments.

**Decisions (owner and conductor, 2026-10-10/11):** capability kinds `Retries` → `Retryable`,
`Deadline` → `Expirable`; every bound in `Bounds`, the close policy's wider budgets as
`fourWide`/`fiveWide`/`twelveWide`, nexus `control` merged into `eight`; `disabled(...)` targets are
qualified like `on`; the System level is named Protocol; a form's file is `<Form><Kind>.scala`; parts
two forms share live under the kind's `shared/`; a form's main realization is `<Form><Kind>Run`;
facet vocabularies are recorded in fn-161.

| Order | Spec | Scala scope |
| --- | --- | --- |
| 0 | fn-166 | the feature as the unit (`FEATURES`, discovered by default), the mini feature `activity_mini`, `make model-mini`; the batch below changes only it and ships each change's migration as a rule the sweep runs on the real features |
| 1 | fn-159 | qualified names, renames, `enters` (source integrated; proof/closure open) |
| 2 | fn-161 | facets |
| 3 | fn-162 | realizations name their Queries; claim containers removed |
| 4 | fn-160 | layout and realization layer (.3/.4, done) |
| 5 | fn-166.11-.14 (done) | the change-scoped model loop (.13 done: `make model-check-changed` is the default per-change check, a mini change 463 s, a lifter change 798 s; .14 done: fixtures build in-process once per run, full irgen 672 s to 425 s, lifter-change check 798 s to 506 s, 5-minute target missed; .15 after fn-141 shares the TASTy read; .11 done: the full lifter suite is green on the mini scope, 274/0 with 6 sweep-held skips, 972 s to 755 s; .12 done: model Make targets run on cached API jars, opt-in `make gen-model-api`, mini loop about 52 s with held Go stages skipped by name): lifter suites scoped to what they lift, Make without held API generation, one diff-scoped check with an evidence record (owner, 2026-10-12) |
| 6 | fn-169.1/.5/.6 (done) | the Temporal-free framework: the DSL and generic export/check/migrate moved into `model/framework`, the one-file samples (Kettle, Bank, TwoPhaseCommit) are checked, and the interpreter-table comparator is Scala; Temporal parts are in `model/temporal`, build boundary enforced (owner, 2026-10-12) |
| 7 | fn-172 | faster model feedback loop: warm compile server, one compile and one lift run for the samples, a prebuilt parallel Go checker (.6 done); independent of the redesign, so it goes first (owner, 2026-10-13) |
| — | fn-173.1 (done) | governance for the redesign (GOV-02: finite inputs/native state, structural Behavior Fingerprint, selected backends and the planned answer module); docs closed locally, implementation remains open |
| 8 | fn-141.6/.7, then fn-174 | the Model constructs its own IR (owner, 2026-10-13): fn-141's Part B mechanism is replaced by conversions on the DSL types and compile-time capture in a compiler plugin (host amended 2026-10-14 after the prototype in `model/design/compiler-plugin-prototype.md`), byte identity is no longer required, and fn-141's .5 and .8–.14 are superseded; fn-169.2's plug-in seam follows fn-174.2 |
| 9 | fn-173.2/.3 | reachable exploration Phases 1–2: the reachable graph becomes the only table (Voting drops `inv`), Scala step tables and Query totals retire, Go static checks on the IR; a Go exception to the Scala-first rule, mini-scoped (owner, 2026-10-13) |
| 10 | fn-160 | Protocol level, flat `features/`, per-feature collection (.6/.7/.2) |
| 11 | fn-163 | system/product capabilities, named system capabilities |
| 12 | fn-147 | Duration-typed elapsed-time authoring (source integrated; proof/closure open) |
| — | fn-140 | witness Queries and `.live` (source done; Go proofs held) |
| — | fn-149 | safety and liveness groups (source done; Go proofs held) |
| 13 | fn-164 | `expect:` blocks for Properties and consistent colon style |
| — | seal | ⏸️ superseded by fn-173: the Scala step-table equivalence retires; evidence becomes fresh-lift IR identity (owner, 2026-10-13) |
| 14 | fn-123 | fault declarations, durability, `crashes(...)` |
| 15 | fn-153 | Property illustrations in the DSL |
| 16 | fn-165 | inputs as records: named classes, declared defaults, `:=` removed from classes (after fn-141) |
| 17 | fn-173.4–.6 | native collections: IR and interpreter `Set`/`Map`/`List`/checked `Int`, then DSL and lifter, then the six samples rewritten |
| 18 | fn-173.7–.9 | backends that decide: export coverage, `checkedBy`, the TLA+/TLC route |

fn-128 and fn-129, the remaining standalone activity edits, are about a real feature only and moved
to [Deferred for now](#deferred-for-now).

### fn-166: Mini activity: the smallest complete feature, the DSL's fast loop and its example

Captured 2026-10-11; reshaped the same day once the owner decided the mini Model stays forever and
asked for as little special casing as possible. The ordinary edit loop is about two minutes of tool
time; the complete source seal is 20 to 30 minutes and scales with Model size, and the time around
it is where fn-149.6 went. The spec makes the feature the unit of every model operation (`FEATURES`
on compile, jar, gate, lint, suites and equivalence, discovered from the tree by default, never
listed; `make model-mini` is the alias for the mini feature) and adds the smallest complete feature,
`features/activity_mini`: three real levels, about 800 lines against 4,600, holding one of each hard
instance the real features have (refinement with a verify through it, a shared input, a shared status
fact, a cross-facet guard, a waiver and an override, a composition with a `sync` and a channel, a
failure model with a negative control, fair progress, a monitor, a hole, a real realization with a
`.live` witness). A real feature not yet migrated to a DSL change is *behind*, a per-feature gate
state; nothing is parked or deleted. A DSL change ships its migration: since 2026-10-13 as a written recipe the sweep applies by hand,
earlier as a scalafix rule proven on the mini feature. The guide uses the mini
feature as its running example. Excluding the real features from a run is on demand only, `FEATURES=activity_mini`,
with no persistent state; the default build is red while they are behind, until the owner's sweep runs the rules on them
(owner, 2026-10-11). The batch changes only the mini feature (see the scope above). Each feature keeps its outputs in its own `generated/` tree (`ir.json`, `ir.lint.json`, `manifest.json`,
`<query>.case.json`), discovered by glob, so the scope of a run is the folder and CODEOWNERS covers a feature with
one line; the central `model/ir` and `model/cases` go away. Extra historical payloads stay losslessly under
their owning feature's `generated/legacy/` until fn-160 merges them; pins and equivalence baselines stay unchanged
(owner, 2026-10-11). The Go exception also covers the narrowly required subset-safe publication of existing functional/canary pins (fn-166.1), approved without changing pin identities or unselected artifacts. Restoring the canonical Make route (`make protoc`, owner-held) is
the first thing to do and is not a task here. First in the batch. Not marked ready.
[Spec](.flow/specs/fn-166-mini-activity-a-small-complete-model-to.md).

Lint checks and manual acceptance work are set aside for this run at the owner's request
(2026-10-11). Existing lint files are preserved; deferred checks are not reported as passing.

| Name | Status | What |
| --- | --- | --- |
| fn-166.6 | ⏸️ deferred | Path-only Case subtree move to `generated/cases/`, preserving historical Cases; owner-held Go/tooling sweep |
| fn-166.4 | ⏸️ deferred | Case lowering and one recorded live verdict after source task 8 and Case-layout task 6 |
| fn-166.10 | ⏸️ deferred | Runnable custom-rule loading/Make integration and mini worked application; separate infrastructure producer for fn-167.7 |
| fn-166.5 | ⏸️ deferred | Three canonical warm loops under 90 s and authorized consumer/full-set closure; source docs and rules are tasks 8/9 |
| fn-166.7 | ⏸️ deferred | Model-side Make target rename and live-reference sweep; runtime target names retained |
| fn-166.15 | ⬜ todo | After fn-141: share the TASTy read across lifts (about 403 lifts at 0.8 s each), in-process conventions lint, bounded parallel suites; target a lifter-change check under 5 minutes |

### fn-167: Gaps declared in the Model: `@gap` replaces the lint file

Captured 2026-10-11 after the owner found the standalone activity's hand-written lint acceptance
file at six thousand lines: 179 entries and 5,089 owner-by-subject pairs that collapse to about 120
reasons, most of them the same inherited rows repeated for every derived owner because the IR does
not record derivation. The spec makes `@gap(subjects*)(because, until)` the one way a Model declares
a known gap, repeatable, on the declaration that owns the finding, with one inert subject
constructor per lint kind taking typed references (`silent(action, in = ...)`, `unconstrained(classes*)`,
`unperformed(actions*)`, `witnessOnly`, ...). The lifter lowers gaps and derivation into the IR as
inert metadata, outside the fingerprint, equivalence and table pins; `except` and `overriding` lower
the same way. The Go lint reports an inherited finding once, on the base, matches the rest against
the IR's gaps, fails a stale gap at its source position and an `until` whose spec is done, and no
longer reads an acceptance file beside a feature with sources (legacy payloads keep theirs until
fn-160). Migration is a mechanical rule proven on the mini feature. The Scala side is done
(.2-.4): gaps resolve to lint finding keys during the lift, derivation provenance is recorded, and
the pure migration is proven on copied mini sources; nothing new is written to the IR yet. The IR uses the shared
protobuf schema, so schema transport/generation, lint consumers, file retirement and real-feature
migration wait for the owner's sweep. Seven tasks are planned; plan review is SHIP, not marked ready.
The mini source proof consumes fn-166.3's committed closure; source groundwork needs no whole-spec
dependency that would bind it to held live/lint work. Full transport-dependent requirements remain
open after the Scala groundwork.
[Spec](.flow/specs/fn-167-gaps-declared-in-the-model-gap-replaces.md).

| Name | Status | What |
| --- | --- | --- |
| fn-167.5 | ⏸️ deferred | Shared proto/ScalaPB/Go transport and independent metadata identity exclusions; owner sweep |
| fn-167.6 | ⏸️ deferred | Gap-aware lint, proven inherited duplicates, stale/expiry errors and legacy compatibility; owner sweep |
| fn-167.7 | ⏸️ deferred | Runnable migration adapter, actual mini proof and real-feature acceptance retirement/counts; owner sweep |

### fn-159: Qualified action names in rules

Readability and naming cleanups across the Models. Rule targets and phases are always qualified
(`on(activityWorker.poll)`, `when(Phase.scheduled)`), and `from(X)` only groups (fn-159.1, fn-159.3).
Actors get clearer names (`activityWorker`, `completionClient`), as do the capability kinds `Retryable`
and `Expirable` (fn-159.2, fn-159.8). Phase-only effects give way to `enters(Phase.X)` (fn-159.7). The
product's never-enabled `disabled(process.stop)` binding goes, unless the lift needs it (fn-159.4), every `Limits` bound moves into
`Bounds` (fn-159.5), `Observation` drops its unchecked `read` text (fn-159.6), and a bounded integer has
one spelling, `UpTo[N]` on the field, with the lifter reading ranges per field (fn-159.9), and an input a class may omit declares its
default on its token, so `start()` reads as every default and no form reads catalog order (fn-159.10).
fn-159.1, fn-159.3, fn-159.6 and fn-159.10 preserve the authored meaning; the renames, bound moves and `enters` change ids or function names
only. Fn-159.9 also fixes a proven bounded-Option identity collision: five type/reference leaves change,
while ranges, source positions, Query metadata and all sixteen mini/shared step tables stay unchanged.
The IR-regenerating tasks wait on the tree's IR `Duration` migration. Not marked ready.
The source changes already integrated into the real features stay; the open proofs and closures run
on the mini feature, and the real features' lift and equivalence proofs wait for the sweep.
[Spec](.flow/specs/fn-159-qualified-action-names-in-rules.md).

| Name | Status | What |
| --- | --- | --- |
| fn-159.2 | ⏸️ deferred (sweep) | By-ID actor renamed to `completionClient` in real-feature source; mini has no By-ID actor; real-feature proof/regeneration waits for the sweep |
| fn-159.4 | ⏸️ deferred (sweep) | Real Product's `disabled(process.stop)` removed in source; mini has no such binding; real-feature proof waits for the sweep |

`disabled(...)` targets are qualified like `on` (decided 2026-10-10).

### fn-161: Facets: action-owning behaviour areas of a feature

A feature's machine is assembled from **facets**. Each facet is a feature-local behaviour area (Pausing,
Cancellation, Reset, …) that owns a set of actions and holds their rules, effects, claims and the
capabilities it declares. Each action has exactly one owning facet; cross-facet step use is declared in
the facet's type. Only facets define rules, effects and capabilities in a declared machine, and each
feature's levels share one facet vocabulary. Facets are named as nouns, capabilities as adjectives. Applies to
the mini feature. The integrated contract, selected admission and compiled wrapping migration
checkpoints are committed in `umpire`. The standalone activity's two stages
wait for the sweep. Qualified effect paths define collisions; distinct namespaces may reuse a
local effect name. The parent spec records this interpretation explicitly. Not marked ready.
[Spec](.flow/specs/fn-161-facets-action-owning-behaviour-areas-of.md).

| Name | Status | What |
| --- | --- | --- |
| fn-161.2 | ⏸️ deferred (sweep) | Stage 1: standalone activity effects and claims into facet files (rules stay by actor) |
| fn-161.3 | ⏸️ deferred (sweep) | Stage 2: standalone activity rules and capabilities into their owning facets |

### fn-162: Queries name their realization; each Property declared once

The activity's 17 claim containers (copies of a machine that change nothing, e.g. the three in
`Heartbeat.scala`) exist because the lowering and the assessment need exactly one realization per
machine. A copy is how a Query picks its realization, or has none. fn-162.1 found a Case is
already assessed against only its Query's one Property (plus the machine's monitors). The fix: each
realization names the Queries it runs (unnamed find Queries stay model-only), and each Property is
declared once in its facet, under its existing name, so the copies go. Query answers and Case outcomes stay the same; machine-scoped ids move
(in the rename map). The Scala part is done: realizations name their Queries (`Realizes(machine, runs = ...)`), a Property declared twice is refused, and the mini feature and shared code hold no claim containers. The standalone activity's identity proof waits for the sweep.
[Spec](.flow/specs/fn-162-queries-name-their-realization-each.md).

| Name | Status | What |
| --- | --- | --- |
| fn-162.4 | ⏸️ deferred (Go) | Go consumers read the realization binding: regenerate the Go API, `lower.Realizable`, `conformance.realizationOf` and model lint; split from .2 |

### fn-160: One IR file per feature, collected from the feature's package

Replace the hand-written `irFile` root lists, three files each for the standalone activity and nexus
workflow, with one IR file per feature folder, collected from the feature's package. Nothing is listed,
so no section can be forgotten. The model content stays the same; file boundaries and derived names
(Cases, lint acceptances, export slices) change. Since 2026-10-11 the file lives in the feature's `generated/` tree (fn-166.1). Each Model file then holds at most one machine, in flat
level folders grouped by name prefix, and realizations get their own `realization/` layer. Since 2026-10-11 the tree is flat: a kind folder holds only shared parts and each form is a `<kind>_<form>` folder (fn-160.7). Layout, naming and the Protocol rename apply to the mini feature and the shared code; moving and renaming the real features' files waits for the sweep. One machine per file, family-prefixed level files (`enum FamilyName` for extra families) and `<Subject>Run` realizations are enforced for every lifted Model (.3/.4 done; six mini/task-queue machines renamed, with the `Fn160Task3FamilyNames` rule for the sweep). The real features are behind on these rules, and the task-queue renames stop the standalone activity compiling until the sweep runs the rule.
[Spec](.flow/specs/fn-160-one-ir-file-per-feature-collected-from.md).

| Name | Status | What |
| --- | --- | --- |
| fn-160.2 | ⬜ todo | Collect one IR file per feature package, and per foundation/actor/kind package that declares Queries or realizations; remove `irFile`/`exports`; exploration checks only the asked Query; generic native realization discovery still needs source proof; regenerate and migrate consumers — Scala part open; Go part deferred |
| fn-160.6 | ⬜ todo | Rename the System level to Protocol (folders, packages, machines, files, lint vocabulary, docs), so the levels read Product / Protocol / Realization |
| fn-160.7 | ⬜ todo | Flatten `features/`: `activity/` and `nexus/` hold only shared parts; forms are `activity_standalone/`, `activity_mini/`, `nexus_workflow/`, `nexus_standalone/`; packages follow; lint refuses nesting (added 2026-10-11, after fn-160.3); now only `activity_mini` and the shared parts it uses, the real forms' moves at the sweep |
| fn-160.8 | ⬜ todo | Drop the `scala.` vestige from Case ids, `explore.` prefixes, activity type names and provenance; the set derives from the feature package (`temporal.features.<feature>.case.<query>`); fixtures, canary binding and Driver Profile regenerated; baseline compared under a mapped identity (R15, owner 2026-10-11); after .2 and .7, in the sweep |

### fn-163: System capabilities: distributed-systems patterns as named capabilities

Open for its Scala part (owner, 2026-10-11); three tasks and the pattern inventory are recorded.
The package split and named law source are integrated, but complete naming, history-law,
equivalence and documentation proofs remain open; IR changes wait for deferred regeneration.
[Spec](.flow/specs/fn-163-system-capabilities-distributed-systems.md).

Name the distributed-systems promises the Models still write by hand (fencing of stale attempts, a
single active attempt, idempotent requests, a single outcome) as first-class capabilities, declared once
with laws that lower to the existing IR. Capabilities split into `capabilities/system/` (server-side
protocol promises) and `capabilities/product/` (what a user sees an entity can be). A pattern is named
even with a single use (owner): the name improves readability and is easy to revise when more uses
arrive. Idempotency targets actions rather than entities, and fencing/single outcome need history, so
some laws may need synthesized monitors; verdicts, answers and Case outcomes stay unchanged under a
checked rename map. The pattern inventory and naming proofs cover the mini feature and the shared code.

| Name | Status | What |
| --- | --- | --- |
| fn-163.1 | ⬜ todo | System/product package split integrated; complete placement/module/lint proof and closure open |
| fn-163.2 | ⬜ todo | Exclusive/Fenced/Idempotent laws integrated; complete inventory/naming/history-law, observable realization-evidence and unchanged-answer proof open; singleOutcome remains hand-written under the recorded history exception |
| fn-163.3 | ⬜ todo | Verify author docs and single-use naming rule against the final source, preserving the existing SEMANTICS draft |

### fn-147 (Scala part): Duration-typed elapsed-time authoring

From Batch 3's format migration: the IR and Testpilot already carry protobuf `Duration`. The
framework/realization/lifter source now accepts native Duration authoring, including optional timeout
and interval fields and typed timer/backoff values. Independent presence/range/refusal, fixture,
equivalence and documentation evidence still needs closure on the current DSL; the older
duration-authoring/runtime-semantics patches are provenance, not unapplied work to replay.
The rest of fn-146/147/148 is deferred below.

| Name | Status | What |
| --- | --- | --- |
| fn-147.2 (Scala) | ⬜ todo | Native Duration source integrated; close focused presence/range/refusal, fixture, equivalence and author-doc proof; shared Go/artifact closure deferred |

### fn-140: One-sentence witness Queries with explicit live expectations

It rewrites the Models' `properties` and `queries` sections and its R5 renames Definition
IDs. Scala tasks .1-.5 and .10 carry the authoring migration; .7/.8/.9 retain the owner-deferred
consumer proofs split from .2/.3/.4, and .6 retains shared closure. The current one-worker lane
is sequential; task .10's committed Scala seal releases later source work without waiting on Go.
Source/closure commits `b5374fbb0b`/`c59b5bdf21` are integrated. The Scala seal's
338-Query core/reader/span proof, complete 226-row refusal contract, 52,244 native rows and
historical equivalence were independently verified, including its immutable 5,272-file seal.
Ten lifter/two export managed comparisons remain held; this releases fn-149 source work only.

`witness(<classes>).records(<fact>)` states a path-and-outcome claim in one declaration and lifts to the existing Scenario, Property and `find` Query, each named after it. `.live(<expectation>)` replaces `.expect` as the one word that generates a Case, and every non-satisfied expectation carries a reason. Pinned `find` Queries in every Model migrate, and the hand-written `terminate`, which repeats the capability-generated `terminateSettles`, is deleted. `query verify` and its triple are unchanged.

| Name | Status | What |
| --- | --- | --- |
| fn-140.6 | ⏸️ deferred | Full consumer/regeneration/equivalence closure after .7/.8/.9/.10; links the shared fn-123.8 sweep |
| fn-140.7 | ⏸️ deferred (Go) | Exact-composed selector admission/check/lint/export proof, split from .2 without relaxing its obligations |
| fn-140.8 | ⏸️ deferred (Go) | Live/model-only Case generation switch and complete manifest/membership proof, split from .3 |
| fn-140.9 | ⏸️ deferred (Go) | Original first-Activity assessment/retry differential, Query receipts and Case/regeneration proof, split from .4 |

### fn-149: Safety and liveness groups for object properties

[Spec](.flow/specs/fn-149-safety-and-liveness-groups-for-object.md) has seven M tasks covering all six
criteria. Its original full-contract plan passed independent review on 2026-10-10; the current
source/consumer partition follows the owner's Scala-only mode and its review remains deferred.
The Go report lane (.3) is held at its existing checkpoint. Scala grouping follows fn-140.10's
committed witness source seal, then releases fn-164 before the pre-fault comparison and fn-123.
Consumer closure remains shared at fn-123.8.
Discovery source/closure commits `b751748cae`/`1c6ee11392` are integrated locally.
The independently authenticated 792-file source seal proves six Properties/four Progress claims,
native initialization and full six-file scratch-lift parity; native step tables remain unchanged.
Placement source/closure commits `77825c5bd2`/`25e12fa6a1` are integrated locally.
Its independently authenticated 3,654-file seal preserves the original six fixture IRs and proves
29 exact kind refusals, seven wrong-owner roots and valid omitted/empty groups; 60 focused tests pass.
Seven production scratch artifacts are byte-identical to the original-admission runner.
The historical discovery bridge is explicit: 19 section roots and the already-authored, formerly
unqueried `asyncStarts` Property/predicate, with every other IR field and waiver unchanged.
This releases migration task .6, not consumer closure or the final grouping seal.
The report checkpoint is integrated as `da8f78204a`: classification is derived, and progress
receipts expose their declared `within` separately from exploration limits. Focused integrated
checks preserve witness verdicts and independent safety failures; the full checker gate remains open.

Split authored claims into `properties.safety` and `properties.liveness`, enforce declaration kinds,
and carry the distinction into existing diagnostics and reports. Liveness retains explicit bounds
and assumptions. Runtime registration and lifted discovery have independent inventory proofs;
shared capability-law counts cannot use the candidate scans as their oracle. Complete grouping
comparisons preserve behavior, check results and Case/assessment meaning, with every identity and
provenance change explained. Fn-149.7 commits the final Scala docs/layout/source seal and releases
fn-164's expectation/colon migration. Fn-149.4/.5 retain complete Go receipt, Case, assessment and
shared gate/review/live proofs at fn-123.8; source results substitute for none of them.

| Name | Status | What |
| --- | --- | --- |
| fn-149.3 | ⏸️ deferred (Go) | Reporting checkpoint integrated; focused classification, distinct bounds and safety/progress coexistence checks pass; full gates/review held |
| fn-149.4 | ⏸️ deferred (Go) | Original complete consumer behavior, Query/progress receipt, Case and expected/live-replay assessment proof; consumes .6 |
| fn-149.5 | ⏸️ deferred | Consumer reporting docs and original integrated regeneration/gates/review/live closure; consumes .4/.7 at fn-123.8 |

The grouping source seal (.6/.7) is integrated into `umpire` with the mini claims grouped; the real
features' grouping, already done by hand on `w-1d`, came with the merge and is proved at the sweep.
Deferred .4 joins .1/.2/.3/.6; .5 joins .3/.4/.7.
Heavy commands remain serialized. Composition progress belongs to fn-150, not this grouping change.

### fn-164: Expect blocks for Properties

Added by the owner on 2026-10-11, after fn-149 and before the seal: a meaning-preserving
migration whose IR differs only in source positions, so the seal's step-table equivalence covers it.
Six sequential tasks now cover R1-R8, including short-circuit/hole behavior, typed contexts,
core-IR parity, complete caller migration and the final colon-style enforcement.
[Spec](.flow/specs/fn-164-expect-blocks-for-properties.md).

A Property's predicate becomes an `expect:` block that reads like the `effect:` it checks: one
requirement per line (`records(fact)`, `check(condition)`), `if` instead of `implies`, and a rare
`oneOf:` for the few genuine disjunctions. Temporal forms (Queries, claim patterns, `leadsTo`,
monitors) stay outer combinators that take these blocks. The lowering adds no IR construct. Every
`holds`/`holdsAcross` lambda Property migrates, and the lambda forms go with no compatibility layer.
fn-141, which runs later, must keep the block's lowering within its "a sugar's
definition is its only lowering" rule.

The owner also requested consistent colon-and-indentation DSL blocks (`rules`, `on`, `effect`,
`expect`, and the other optional blocks). This follows the expectation migration and precedes the
seal, using the pinned formatter's built-in rewrite where possible and focused lint for any gaps.

| Name | Status | What |
| --- | --- | --- |
| fn-164.2 | ⬜ todo | Typed expectation contexts and action heads; before/after reads; short-circuit alternatives; compiler scope refusals |
| fn-164.3 | ⬜ todo | Exact core-IR lifting, temporal predicate bridges, source-positioned refusals and early parity proof |
| fn-164.4 | ⬜ todo | Capability/foundation predicates migrated with full declaration/function parity |
| fn-164.5 | ⬜ todo | The mini feature's Properties migrated with exact preserved IR; the real features' Properties (heartbeat/reset alternatives) at the sweep |
| fn-164.6 | ⬜ todo | Remove lambda finishers together; migrate fixtures/core forms; author docs and final expectation source seal |
| fn-164.1 | ⬜ todo | Consistent colon blocks across the DSL, fixtures and docs; formatter/lint enforcement; source-only lift changes and unchanged step tables |

### fn-123: Declare faults as the environment's actions

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
| fn-123.3 | ⏸️ deferred (Go) | Go derives the crash row; `SEMANTICS.md` Faults section (proof); waits for fn-173.2 (owner, 2026-10-13) |
| fn-123.4 | ⬜ todo | Task-queue providers converted; `fault-overridden` lint; derived storage-loss assumption |
| fn-123.5 | ⬜ todo | Budgets: `budgetedBy`, four table rules in Go, `LostStartAnswer` bound to `lossAvailable` — Scala part open; Go part deferred; waits for fn-173.2 (owner, 2026-10-13) |
| fn-123.6 | ⏸️ deferred (Go) | Choice-level fault performance; lowering refuses model-only faults and unperformed choices; waits for fn-173.2 (owner, 2026-10-13) |
| fn-123.7 | ⏸️ deferred (Go) | Trace output and Quint export of derived crashes and budgets; waits for fn-173.2 (owner, 2026-10-13) |
| fn-123.8 | ⏸️ deferred (Go) | `umpire-faults` report, docs; close; waits for fn-173.2 (owner, 2026-10-13) |
### fn-153: Checked property examples pilot

Six M-sized tasks cover all seven requirements. The plan passed independent review on 2026-10-10.
Its authoring parts are open in the Scala DSL work, after fn-123. The pilot's three Properties are
the mini feature's, not Nexus's.
[Spec](.flow/specs/fn-153-checked-property-examples-pilot.md).

Pilot checked examples and counterexamples attached to three existing Properties: a same-step
promise, a before/after promise, and a promise with a subtle applicability boundary, all three from
the mini feature. The existing
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
| fn-153.3 | ⏸️ deferred (Go) | Existing Property selector/predicate classifier, errors and local mutation controls |
| fn-153.4 | ⬜ todo | Compact deterministic document and gate freshness; handwritten assessment outside its managed subtree — gate freshness deferred |
| fn-153.5 | ⬜ todo | Three existing mini-feature Properties' contrast pairs and classification drift evidence |
| fn-153.6 | ⬜ todo | Evidence-limited assessment, docs, complete compatibility comparison and full gates — full gates deferred |

Tasks run in order. Re-anchor against the actual baseline after the delivery chain; keep
hypothetical violations, actual model failures and live coverage separate. Mechanical checks
do not establish developer comprehension, and absent feedback must remain explicit.

### fn-169: A Temporal-free framework: the lifter and checks join `model/framework`

Captured and approved by the owner on 2026-10-12. `model/framework` becomes self-contained and
Temporal-free, so it could be reused for other systems. Most of `model/irgen`, `model/check` and the
`model/migrate` harness is generic and moves into `framework/{export,check,migrate}`; the Temporal kit
lifting, layout conventions, API descriptors, gate and task rules move into `temporal/{export,check,migrate}`.
A build boundary keeps the Temporal API and `temporal.*` sources off the framework's classpath. The
mechanical move runs before fn-141; the remaining layout/API plug-in seam follows fn-174.2,
which supersedes fn-141.8's realization producer. Kit types own their constructed conversion;
the seam does not restore tree-side realization lifting. The IR namespace and the Go
`tools/umpire` Temporal adapter are held for the sweep.
fn-169.1 is done (2026-10-13): `model/irgen`, `model/check` and `model/migrate` are gone. The DSL is
`framework.dsl`, the lifter `framework/export` (with a small plug-in hook implemented by
`temporal/export`), the generic checks `framework/check/{lint,equivalence}`, the harness
`framework/migrate/prove.sh` (`FEATURES`-parameterized), Temporal's halves under
`temporal/{export,check,migrate}`. `make check-model-boundary` enforces the Temporal-free framework and
runs inside `model-check-changed`. Residual Temporal names in the lifter and checks are listed in
`model/README.md` for fn-169.2. The IR now names four DSL types under `framework.dsl.`; Go follows in
fn-169.4.
Migration rules are temporary: `temporal/migrate` keeps only rules the sweep has not applied, and each
is deleted once applied; the harness in `framework/migrate` is permanent (owner, 2026-10-12).
[Spec](.flow/specs/fn-169-a-temporal-free-framework-the-lifter.md).

| Name | Status | What |
| --- | --- | --- |
| fn-169.2 | ⬜ todo | Plug-in seam: layout conventions and API descriptors supplied by `temporal/export`; generic proto support with a registered API (proto lint, gate proto stage); realization lifting leaves the seam, since under fn-174 the kit's types convert themselves; after fn-174.2 |
| fn-169.3 | ⏸️ deferred (sweep) | A Temporal-free Umpire IR with no extension schema: Umpire namespace, generic realization layer, Temporal vocabulary as data validated against descriptors and Driver capabilities (owner, 2026-10-13) |
| fn-169.4 | ⏸️ deferred (Go) | Go consumers follow the `framework.dsl` IR type names (Testpilot separation moved to fn-171, `tools/umpire` layout to fn-170.2) |

### fn-141: Shrink the IR generator: one description of each DSL construct

**Part B superseded by fn-174 (owner, 2026-10-13).** Parts C, D and A (.1–.4) are done and stand.
Part B's mechanism, an exporter per kind plus function lookup by recorded span, is replaced by
fn-174: each DSL type builds its own `ir` message and capture points lift function bodies at compile
time. Only .6 (refusal ledger) and .7 (complete realization values) remain here, as prerequisites of
fn-174.8 and .2. Byte identity per step (R11) is retired; two-run determinism (R10) carries over.
fn-141 closes when fn-174.8 retires the lifter.

The lint extraction (.1) is complete: `framework/inspect` owns the shared typed-tree index,
`framework/lint` owns source and marker checks, and the gate runs them around export in one
inspector invocation. After declaration expansion (.4), exporter production sources are 12,541 lines;
the census across export, lint and inspect is 15,874 lines (546 more than the .2 checkpoint,
including generic expansion and subset checking, not a net-shrink claim). The unused-spelling checkpoint (.2) retires nine direct typed
constructors while retaining canonical factories. `Bindings` is reserved for .3/.4's core proof
oracles; its final retirement remains open in .10/.14. Generic function-body sugar expansion (.3)
is complete: the seven named families use indexed definitions, with caller positions and strict
argument evaluation preserved. Its scoped gate passed 604 tests, three samples and mini step-table
equivalence. Declaration sugar (.4) is complete: claim patterns and sticky monitors follow their
definitions, author-time lint shares typed library admission with the lifter, and SEMANTICS documents
the subset with fixtures. The old pattern comparison has its own suite; frozen IR and refusal
contracts pass. Its final scoped gate passed 610 tests, three samples and mini equivalence.
Capture and function lookup by span (.5) are superseded; fn-174.1 owns the compile-time body proof.

Ready 2026-10-08. Pulled forward by the owner on 2026-10-12: it runs after fn-160.3/.4,
fn-166.11-.13 and fn-169.1 (so it works in the framework's final homes), before the remaining DSL specs, which then land on the smaller lifter. Its
"spellings no Model uses" recount (fn-141.2) counts the current mini and shared code only, and must
keep any spelling an open spec in this batch is about to use. As each task rewrites a construct, it splits that
construct's tests out of `Fixtures.test.scala` to sit beside the construct's code (owner, 2026-10-12).
To keep the lifter from being overfitted to Scala syntax (owner, 2026-10-13): fn-141.4 added an author-time
lint that refuses non-liftable Scala in step functions, and one documented liftable-subset table in
SEMANTICS (tree forms and allowed library operations, one fixture each, sharing one allowlist with the
lifter). fn-141.1 records, and fn-141.14 reports, the lifter's lines by category, so it shows whether the
re-parsing of declarations, sugar and kit syntax is gone.
Because its tasks were planned against the earlier tree, each re-reads its files and recounts first.
Tasks follow Flow's branching dependencies. Tasks 1 and 2 have overlapping declared files and run
serially; after both finish, 3, 6 and 7 are disjoint parallel candidates. Task 5 is the proof for the
export part and task 8 its size proof; either can stop tasks 9 to 13, and tasks 1 to 4 stand without
them. Source: a full read of `model/irgen` on 2026-10-06 (8,242 lines, of which about 10% lifts
function bodies and types and 1,479 are lints that emit no IR).

Every step leaves the mini feature's `generated/` tree byte-identical, so each task's gate is that
byte-identity plus the lifter's own tests; the Go suite's baseline from batch 3 stays valid while the
IR is unchanged, and runs once more at fn-141.14. Spellings only the real features use do not count
as used.

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
| fn-141.5, .8–.14 | ⏸️ superseded | by fn-174.1–.8 (owner, 2026-10-13); the `Bindings` retirement and the unused-spelling audit move to fn-174.4 and .8 |

The refusal ledger (.6) is committed and closed locally at `4363a3f509`.
[Ledger](.plans/REFUSAL_LEDGER.md) accounts for all 617 direct production sites in 61 kinds:
149 macro, 379 construction and 89 source-lint assignments, with no Go-only home or proven
deletion. Each kind has an fn-174 owner, source-position obligation and existing fixture or
explicit coverage gap. The census, unique-site reconciliation and cited paths passed verification.
These are intended migration homes, not implemented enforcement; fn-174.8 verifies them.
Unused effects, folded-away result paths and held real-feature refusals remain explicit obligations.

Realization values (.7) are committed and closed locally at `97f0ff053f`, source `8ad875931a`.
Factories retain ordered scopes, reads, nested literals, typed field paths/message identities,
operators, aliases, evidence keys and deadline bindings; 37 construction tests exercise readback
and evaluation. The memory-constrained scoped gate passed in 825.5 s with six samples, exact mini
step-table equivalence and 673 tests. Conversion and deferred declaration/family identities belong
to fn-174.2; opaque selectors must be explicitly refused there, not assigned guessed paths.
The current lifter, schema and generated payloads remain unchanged. Batch-end review remains open.

### fn-165: Inputs as records: named classes, declared defaults, no `:=`

Captured 2026-10-11 from `model/SEMANTICS_FEEDBACK.md` 3.3. An action's inputs become a case class
(`final case class Start(scheduleToStart: Timeout = Timeout.unset, ...)`, declared with
`action(this).inputs[Start]`), so a class is plain Scala: `start()` is every default,
`start(maxAttempts = MaxAttempts.two)` names what differs, and a step function reads `in.startToClose`
as it reads `s.phase`. Single-input actions keep `finish(Result.ok)`. A shared input across actions is
a field with the same name and type, and a consumer that needs it writes a structural bound the
compiler checks (option 1 of five weighed; see the spec's Decision Context). The IR, Cases and Go do
not change; the equivalence seal proves it. Placed after fn-141 so the class lowering is written once
against the final lifter; fn-159.10 (explicit defaults on tokens) is superseded by it or skipped
(R10). Not marked ready. [Spec](.flow/specs/fn-165-inputs-as-records-named-classes.md).

| Name | Status | What |
| --- | --- | --- |
| fn-165.1 | ⬜ todo | Inventory: actions, class calls, input-reading steps, token-identity consumers; freeze field order, defaults and shapes |
| fn-165.2 | ⬜ todo | Framework: `inputs[In]`, `(s, in)` step functions, structural shared inputs; proof that class order and `Param` names are identical |
| fn-165.3 | ⬜ todo | Lifter: classes from record construction, `in.field` reads, refusals (positional, computed default, `copy`), fixtures |
| fn-165.4 | ⬜ todo | Realization kit: `deadlines` and presets by field selector |
| fn-165.5 | ⬜ todo | Migrate the mini feature and the shared code, byte-identical IR, equivalence; the real Models by the rule at the sweep |
| fn-165.6 | ⬜ todo | Remove the token form and `:=` on classes; README and SEMANTICS |

### fn-172: Faster model feedback loop: toolchain fixes independent of reachable exploration

Captured on 2026-10-13 at the owner's request, from timings of the framework samples and from
`model/design/reachable-exploration.md` §10. A samples change took 19 minutes under
`make model-check-changed`. One lift costs 30–40 s whatever the model's size, packaging and
compiling the same inputs cost 45 s, and the Go check of Bank and Voting costs about 50 s each.
These fixes need no change of design and keep every receipt, IR byte and identity unchanged. Order
7: first, before fn-141. [Spec](.flow/specs/fn-172-faster-model-feedback-loop-toolchain.md).

All six tasks are committed and closed locally; independent review remains at the source-batch
boundary. Guarded warm compilation preserves normal
compiler protection for Temporal headers and direct `-Werror` calls; sample inputs compile once,
and six independent sample lifts share one JVM. The .3 lifter stage fell from 30.153 s to 16.738 s;
all six raw/adapted IRs and complete Go reports are unchanged. Its scoped check passed in 368.1 s,
with the final missing-output correction covered by focused verification (18 tests). Packaging and
safe cache reuse (.4) reduce a warm lift from 20.09 s to 7.343 s. All 18 raw/adapted IR and complete
Report pairs remain identical. Its scoped record retains a red verdict from a synthetic test
diagnostic despite passing computational stages, mini equivalence and 311 lifter/164 check tests;
the final fixture-only correction passed all 164 checks through the unchanged compiler-verdict
guard. The native Go checker (.5) builds once per run, uses two workers, joins every check and
publishes diagnostics in sample order. All 18 raw/adapted IR and complete parallel Report pairs
remain identical. The scoped gate passed in 988.6 s (311 lifter and 166 check tests, six samples,
mini equivalence); a subsequent test-only ordering correction passed focused tests and all 166
checks through the unchanged verdict guard. The quiet warm pipeline was 126.137 s versus
114.899 s serial (9.8% slower); observed memory high waters rose from about 8.31 to 11.25 GiB.
Bounded parallelism is implemented, but no pipeline speedup is claimed and higher concurrency is
not justified by these measurements. The subsequent fn-141.7 gate exposed a capacity limit:
the kernel OOM-killed the sample checker at about 9.7 GiB resident memory while the warm compiler
and unrelated workloads occupied the remaining host RAM. Both failed records remain evidence;
the unchanged two-worker retry with the explicit Go soft budget `GOMEMLIMIT=6GiB` passed in
825.5 s: six samples, mini equivalence and 673 tests. Observed checker high water was about
6.43 GiB; the soft budget is not a hard resident-memory limit.
No default-memory pass or production memory fix is claimed.

### fn-173: Reachable exploration: drop full tables, native collections, backends that decide

Captured on 2026-10-13 at the owner's request. The design and its settled decisions D1–D8 are in
`model/design/reachable-exploration.md`. Umpire stops building every machine's table over the whole
catalog: the only table is the graph reachable from the machine's starts, Scenario starts and
Exploration candidates, explored on demand within a per-machine reach bound. Inputs stay finite.
Open graphs need explicit acceptance; the Behavior Fingerprint becomes an IR structural hash; Scala
step tables, equivalence sealing and Query totals retire, and static Go checks on the IR cover the
gaps; state may hold native `Map`, `Set`, `List` and `Int`; Apalache and TLC may decide Queries. It
supersedes the seal row and fn-157.1/.2/.3/.5. Every affected open spec carries a "Redesign impact"
note: most move their evidence from step-table equivalence to fresh-lift IR identity, and fn-123's,
fn-150's, fn-128's, fn-129's, fn-130's and fn-170.2/.3's Go work should wait for Phase 1. It is
placed at order 9 (Phases 1–2, after fn-141), 17 (.4–.6) and 18 (.7–.9).
[Spec](.flow/specs/fn-173-reachable-exploration-drop-full-tables.md).

Governance (.1) is committed and closed locally at `662380279a`. SPEC, VISION and the module map
record the owner's D1–D8 approval under GOV-02, distinguishing finite inputs from native state,
reached-graph scope from independently inconclusive statuses, structural identity from equivalence,
and the planned answer module from current implementation. Documentation checks passed and all
existing rule ID/title headings are unchanged. No engine, backend or regeneration credit follows;
independent review remains at the source-batch boundary.

| Name | Status | What |
| --- | --- | --- |
| fn-173.2 | ⬜ todo | After .1: Phase 1, the reachable graph is the only table; reach bound, `open-graph` lint, receipt fields, structural fingerprint, parity test, one regeneration; Voting drops `inv` |
| fn-173.3 | ⬜ todo | After .2: Phase 2, Scala catalog machinery and Query totals retire; Go static checks on the IR |
| fn-173.4 | ⬜ todo | After .2: Phase 3, IR and interpreter collections, checked `int64`, structured key lookups |
| fn-173.5 | ⬜ todo | After .3/.4: Phase 4, DSL and reify plugin (fn-174): native collections and `Int`, `oneOf` inputs, input-reading guards |
| fn-173.6 | ⬜ todo | After .5: Phase 5, the six framework samples rewritten natively |
| fn-173.7 | ⬜ todo | After .5: Phase 6a, export coverage (refinement, progress, Queries, channels, collections) |
| fn-173.8 | ⬜ todo | After .1/.7: Phase 6b, backends decide (`checkedBy`, coverage admission, Go replay, agreement) |
| fn-173.9 | ⬜ todo | After .8: Phase 6c, the TLA+/TLC route |

### fn-174: The Model constructs its own IR: DSL types build `ir` messages, capture points lift function bodies, the lifter retires

Captured on 2026-10-13 at the owner's request, from the spike recorded in
`model/design/scala-macros-export.md` (§8). Every construct is described twice today: the running
Model builds it, then the lifter's JVM rebuilds it from TASTy. fn-141 Part B planned an exporter per
kind plus a lookup of each function by recorded span; this spec replaces that mechanism. Each DSL type
builds its own `ir` message at construction; every capture point (`on`, `effect`/`is`, `holds`,
monitor and refinement functions, capability fields) lifts the body it holds to an `ir.Function` at
compile time through one module, `model/framework/reify`, which is the lifter's body lifter moved,
not rewritten; lift is "construct, then print". Staleness under incremental compilation is removed
at the definition site: a helper reached across files carries `@lifted`, and a capture never reads
outside its own compilation unit otherwise. **Host amended 2026-10-14:** `reify` is a Scala compiler
plugin with phases before `pickler` (store a `@lifted` def's IR on its symbol, capture at call sites,
insert the caller's spelling on values named by reference), not capture macros plus a macro
annotation; no `-experimental`, no `-Yretain-trees`.
Owner decisions: byte-for-byte IR identity is not required during the migration (the Go checks'
receipts, Cases and backend answers are the oracle); declaration positions may be empty; names stay
(reflection already supplies them). Kinds move in fn-141's order, realizations first. After fn-172;
fn-141.6/.7 are prerequisites and stay in fn-141, whose .5 and .8–.14 are superseded. Order 8.
[Spec](.flow/specs/fn-174-the-model-constructs-its-own-ir-dsl.md).

The first attempts at .1 and .2 stopped on two design conflicts: a macro annotation stores a
helper's IR after `pickler`, where no caller macro can see it, so the caller-specific `choose`
refusal could not stay a compile error (`.flow/tmp/fn174-boundary-conflict.md`); and constructed
values lose which declaration a caller spelled, `scripted(original)` versus `scripted(alias)`
(`.flow/tmp/fn1742-name-conflict.md`). A compiler-plugin prototype (`.flow/tmp/fn174-plugin-prototype/`,
verdict in `model/design/compiler-plugin-prototype.md`) solved both on the pinned compiler: the stored
IR crosses the jar boundary and is visible to a later plugin phase and to a macro, the refusal lands
at the caller's alternative with today's text, a warm helper-only edit recompiles the caller through
zinc, and the two spellings give ids `original` and `alias` from constructed values. The spec was
amended on 2026-10-14 to the plugin host; .1's framework foundation is committed and .2 is now
in progress. Both tasks retain their open integration obligations.
The framework checkpoints (`ad563e6a43`, `f73a6139f2`, formatted at `333ba92f0b`) share the real
body engine with the old host,
store linked function/type IR at definition sites and capture typed bindings. Focused fixtures prove
jar capture, refusals,
chunked carriers and warm helper invalidation; the old-host oracle covers all 167 sample functions.
The current focused framework run passes 23 cases: ownership, 16 real plugin boundary cases and
six old-host sample comparisons. It covers all supported `~>` arities, stored choice copies,
nested `records`, selector refusals and compiler-run-local jar-root reuse. The full plugin sample
oracle and held mini/Temporal integration gates remain open; these checks do not complete .1.
All six samples compile against separately packaged framework and plugin jars with the native
strict flags; emitted bytecode confirms the four direct rebind sites use captured bindings.
Three green pure-conversion snippets from the first .2 attempt are recoverable under
`.flow/tmp/fn1742-pure-conversions/`.
The .2 framework checkpoint is committed at `df63813a49`. Frozen checks pass: 33 native/conversion
cases (18 existing construction and 15 conversion cases), 23 reify cases, seven compiled-client/export
cases and four native step-table cases. The conversion cases cover empty `perform`/`onPath` refusals,
foreign same-name fact rejection and refusal of dropped entity binding metadata. Compiled clients cover
caller spelling through helpers, defaults, constructors
and varargs; top-type identity preservation; RPC variants and single scope evaluation; located
nonreference refusals; one independent old-host realization comparison; actual action/channel
signature bridging; and private/nested roots with sorted one-time construction and failure atomicity.
Empty factories now refuse at native construction, an intentional behavior change matching the old
export refusal. Primitive timer/internal ownership and two separate-JVM exports pass; a seeded
mutation fails specifically at their equality assertion and is restored before final verification.
The df63813a49 checkpoint's production Scala/project-directive diff is +788/-46 lines (net +742),
excluding tests, fixtures and README. Both migration paths still coexist; this is growth, not
the eventual size-reduction proof.
Family provenance is committed at `7960ab6348`: ordinary authored helpers/vals retain their own
source package, while vocabulary helpers inherit caller context. The frozen framework checks pass
76 cases (35 native/conversion, 23 reify, 14 compiled-client/export and four step-table cases).
An independent old-exporter realization oracle, separate-JVM exports and warm helper invalidation
pass. Source contracts refuse inspection and opaque forwarding across jars; trusted dependencies
use binary artifact/runtime-image identity, with a spoofed-package regression. The driver validates
root contracts before initialization; previously compiled roots must be recompiled with the plugin.
This checkpoint adds net 453 production Scala lines (+461/-8), excluding tests, fixtures and README.
Instruction-copy reference retention is committed at `61dbc6bd2d`, including polymorphic RPC/poll
copies without changing native product shape. The affected checks pass 36 native/conversion and
14 compiled-client cases; deliberate RPC/poll metadata-loss mutations fail the new assertions.
Source ownership is committed at `c2207956bd`: machine-owned fact companions retain their identity
through helpers and derivations, derived action bindings retain entity names, and named arguments
retain family provenance. Native action inputs are checked against their actual domains before
encoding; shared rejection reasons convert without accepting foreign same-spelling reasons.
Frozen framework checks pass 84 cases (40 native/conversion, 23 reify, 17 compiled-client/export and
four step-table cases), with formatting checked on 12 paths. Independent old-reader comparisons
match binding actions/realizations and fact realizations after removing only positions; the binding
fixture includes structured defaults. A deliberate late-registration mutation fails construction
inside a machine and is restored before final verification. Across `7960ab6348..c2207956bd`, production
Scala grows by 175 lines (+210/-35), excluding tests, fixtures and README. The supported generic
framework conversion lane has no known remaining hole from this pass; kit conversions, mini
equivalence, complete reader retirement and full acceptance remain open.
The extracted framework source checkpoint .9 closed at `7754e2480b`. Native finite schemas are
committed at `183f41444a`. Witnesses retain nominal identity, ordered structure, bounds and typed
unresolved Inbox references independently of catalogs, including custom and empty witnesses and
shared opaque-domain witnesses. Helper jars retain schemas without foreign author-tree reads.
Frozen focused checks pass 213 cases (117 DSL, 40 realization, eight schema, 23 reify, 17 naming
and eight bounded-integer cases), plus configured formatting on ten paths. Nine independently
retained old-reader type messages match after removing only positions; all six unchanged samples
compile with the plugin. This does not establish all-function plugin parity. Production signature
emission keeps its existing producer. This checkpoint adds 324 production Scala lines (+350/-26),
excluding tests, fixtures and README. The .10 closure is committed at `de24ae7ff3`.
Native Action/Input/Channel contributions are committed at `26c7d076fa`, retaining typed schema
identity through helper jars, aliases, generic input prefixes, action copies and channel forwarding.
Frozen focused checks pass 227 cases (125 DSL, 40 realization, four compiled signatures, eight
schema, 23 reify, 17 naming, two defaults and eight bounded-integer cases), plus configured
formatting on ten paths. An independent compiled-signature rerun passes all four cases. Old
actions, channels and referenced type closure match after removing only positions. Nested
`Some(None)` defaults are explicitly qualified: native schema-directed encoding uses the declared
Option type, while the old reader emits `scala.None`; callable defaults remain unchanged. The
retained old raw output and fixture source are unchanged, but the original temporary framework
jar has been pruned by the test scratch-retention policy. This checkpoint adds 395 production
Scala lines (+434/-39), excluding tests, fixtures and README. An unexpected empty-tree commit
was repaired additively without changing working files or rewriting history; the repaired tree
differs from its pre-task base only in the 13 approved paths. Full .3 still depends on full .2,
.10, .11, .12, .13, .14 and .15; activation, remaining signature integration, reader retirement
and full acceptance remain there. Composed references are committed at `a4ceca13b9`: exact
member/sync coordinates, all four reference variants, helper forwarding and derived replacements,
plus lossless ordinary/composed fairness chains. Frozen checks pass 224 cases across eight suites,
with formatting checked on 13 Scala paths. An independent committed-source rerun passes all four
reference cases, including the five frozen old references in two JVMs and current-reader replay.
Capture validation/helper storage remains before pickling; runtime retargeting now follows pickling
so the existing reader retains original binding trees. Native/runtime bytecode controls pass.
This checkpoint adds 394 production Scala lines (+423/-29), excluding tests, fixtures and README.
It uses native bound action declarations without running transitions. Complete Composition messages
and full consumer ownership/refinement remain open; reference conversion does not depend on full .5.
Residual signatures are committed at `c7ba35f761`: complete dormant assumptions, holes and Limits,
with exact omitted/nonempty/explicit-empty name provenance for Action, Entity, Observation,
Assumption and Limits. Action conversion and implicit sync share one effective-name rule. Raw
values, evaluation and case-class product/copy/extractor shapes are retained through supported
aliases and helper jars. Canonical identity is separate from early Entity spelling; actual
anonymous Action/Hole and other declaration-kind controls refuse. The inherited Given-plus-`given_`
refusal also applies to authored named Givens; ordinary `given_` vals remain admitted.
Frozen checks pass 236 cases, with configured formatting on 14 Scala paths. An independent
committed-source rerun passes all three residual cases: complete old/native messages except
positions, exact empty names, implicit sync, two-JVM determinism, saved-tree replay and qualified
helper/copy controls. All original source/oracle and retained pre-change jar hashes remain
unchanged. This checkpoint adds 266 production Scala lines (+310/-44), excluding tests, fixtures
and README. Actor IR naming is committed at `aa081d1e00`. A private name method preserves exact
plain constructor strings, including empty, and uses post-pickler compiler spelling for Module
classes. Public names/getter overrides, toString, references and constructor/helper evaluation
remain unchanged. Missing/conflicting metadata identifies the Action; early superclass/trait
conversion refuses before registration and deferred conversion uses the final module name.
Frozen checks pass 243 cases, with formatting on 11 Scala paths. An independent committed-source
rerun passes all three Actor cases, including all 14 frozen old Actions except positions,
nested/inherited/backticked names, two-JVM determinism, current-reader replay and helper jars.
Old source/raw/jar hashes remain unchanged. New/default/named constructors, module aliases,
ordinary/jar helpers and inherited class-owned Actions remain qualified native controls; the
helper-subclass constructor's existing compiler refusal is preserved. This checkpoint adds 28
production Scala lines (+29/-1), excluding tests, fixtures and README. State-field Inbox ownership
is committed at `e4524c0b15` (.15). Exact witness/schema/declaring-type/case/field evidence retains
deferred lexical Channel receivers after pickling and resolves isolated native state types without
reading catalogs. Structural Finite.schema remains unchanged. Frozen checks pass 255 cases, with
configured formatting on 16 Scala paths. An independent committed-source rerun passes all five
Inbox cases: complete seven-Type/six-Channel old messages except positions, two-JVM determinism,
saved-tree replay, aliases/helper jars and lazy catalog/receiver timing. Retained source/raw/jar
hashes remain unchanged. Missing/conflicting ownership and duplicate record/enum storage refuse;
same-state distinct same-message channels have no new supported spelling. Nested closure needs
explicitly selected nested witnesses. This checkpoint adds 226 production Scala lines (+228/-2),
excluding tests, fixtures and README. Complete Machine delivery/loss validation remains .4 work.
Core binding source preparation is committed at `17706a4127` (.16). Captured function IR and exact
Action declaration references survive native tables and scoped derivations. Private conversion
preserves compiler-owned named/anonymous roots, callable identity and lazy table timing, constructing
ordered binding rows and reachable function/type closure without callbacks or catalog evaluation.
Initial duplicates retain historical anonymous last-definition behavior; pre-reduction evidence
refuses invalid derived operations even after restriction. Imported Type duplicates ignore only
top-level Position and retain the first complete message; real conflicts and stale declarations refuse.
Frozen checks pass 273 cases, with configured formatting on 14 Scala paths. Independent checks pass
13 focused cases and all five compiled cases from a checksum-identical committed snapshot, including
two-JVM old/native projection, saved-tree replay, helper jars and all arities. The first root compiled
attempt failed during setup on host disk exhaustion; no semantic result is inferred from it.
All retained strict old source/raw/jar hashes are unchanged. Copy/function-alias controls remain
qualified. This checkpoint adds 317 production Scala lines (+352/-35), excluding tests, fixtures
and README. It constructs no complete Machine and does not activate production emission.
Machine header/type source preparation is committed at `cb871e4406` (.17). Private five-field
contributions retain compiler-owned family/name and exact S/O/F selection on original witnesses.
Scalar blanks and Nothing omission preserve historical projection. Derived objects own names;
Built intermediates forward selections without declaration identity. Missing/premature/conflicting
metadata, wrong keys, empty roots/fields/shapes and incomplete closure refuse. Conversion reads no
init/end, rules/table, phase/refinement, guards/effects or catalogs; native construction timing stays
unchanged. Nested Inbox ownership still needs exact selected nested witnesses. Frozen checks pass
232 cases, with configured formatting on 12 Scala paths. Root independently passes nine runtime,
four compiled header and three Actor cases, including complete old/native projections, two-JVM
determinism, saved-tree replay, helper jars and actual admission qualifications. All 15 committed
blobs match the frozen manifest; original source/raw/jar hashes and owner design edits are unchanged.
UpTo/opaque states, shared opaque outcome and counter helper forms keep their actual old refusals.
This checkpoint adds 186 production Scala lines (+197/-11), excluding tests, fixtures and README.
Complete Machine assembly/evidence, ordinary rule lowering, monitors, refinement, automatic nested
ownership and activation remain open. Independent review and broad gates stay at source-batch end.
Consumer-specific fairness remains subsequent integration work. Production activation,
reader retirement, full .2/.3 and R5 remain open.
Role-bearing enum Finite attachment is repaired at `385ebc0ab8` (.18). Two production lines retain
the actual enum-case singleton symbol before widening loses its identity. Complete parent schemas,
roles and native catalogs match fixed pre-change old output. Worker checks pass 28 scoped cases,
qualifying four suites before the final standalone-negative test isolation. Root independently
passes all 12 final schema cases and seven-file formatting, verifies all ten committed source blobs
and unchanged old/owner artifacts. Missing Closed and field-bearing role sets keep their actual
consumer-stage qualifications. The .19 endpoint checkpoint uses this repair. Full .3/.4 retain .18 alongside
every original integration prerequisite; fn-169.2 still requires full .2.
Endpoint source checkpoint .19 is committed at `ff1ae5dfe6`. Shared Expr builders capture authored
starts and explicit/direct default Phased ends; the private contribution includes complete reachable
dependencies and forwards original source metadata through Derived/Built without native evaluation.
Worker checks pass 264 scoped cases with qualified DSL/endpoint reuse after only the prospective
header fixture/harness changed. Root independently passes 225 committed-source cases and 26-file
formatting, verifies all 29 source hashes and unchanged original fixtures, old outputs and samples.
Mixed role/header fixtures stay frozen; prospective native-only controls retain their actual old
refusals. Seven production Scala paths add 372 lines (+408/-36), excluding tests, fixtures and README.
Full Machine assembly, ordinary Rules, evidence, monitors/refinement, nested ownership, G01,
consumer integration, production activation/reader retirement, R5 and source-batch review remain open.

Ordinary Rules checkpoint .20 reaches source `a76b8e3674`. Typed guard/effect fragments and
definition-owned section IR/refusal carriers construct isolated ordered StepBindings, Functions,
Types and external references through shared IR-only lowering. Structural class domains, disabled
rows, partial effects through six inputs and exact typed clause groups preserve old output, source
positions, saved-tree replay and two-JVM determinism. First table initialization remains native;
later collection adds zero evaluations. Worker checks pass 293 cases; root independently passes
243 committed-source cases, checks all 35 Scala files with the pinned formatter and verifies all
38 source hashes. Original fixtures, sample inputs and protected designs remain unchanged.
Fourteen production Scala paths add 1963 and remove 467 lines. External mixed-scope checkpoint
`d46d4b8a79` and AGENTS-only `d8322e2687` remain qualified in the receipt. Full .1/.2/.3/.4,
Facets/refinement integration, complete derivation integration, Machine assembly, evidence/monitors/nested ownership/G01,
consumer integration, source-batch review, R5, sole production activation and reader retirement
remain open. Temporal, tools/Umpire, Go, generated outputs and broad gates stay held.

Facets checkpoint .21 reaches source `08a8506e7579`. Definition-owned typed rule templates and
concrete assembly-owned phase context construct complete isolated ordered bindings and helper/Type
closure through the existing lowering. Trait-before-client jars, per-assembly A/B isolation,
capture-avoiding substitution, exact live namespace/Class/callback provenance, native evaluation
timing and same/jar single/combined saved-tree replay are verified. Worker checks pass 310 cases;
root independently passes 245 committed-source cases and the pinned 56-file formatting check.
All 66 tested source blobs match the commit; original fixtures, six sample sources and protected
designs remain unchanged. Thirteen production Scala paths add 2149 and remove 210 lines.
Frozen JSON EOF whitespace remains a qualified advisory in the receipt, not a green full whitespace
check. The old combined contextual-helper behavior and exact SourceFile-collision position
qualification remain preserved; full .4 retains model-wide helper/source-index reconciliation.
Complete derivation/Machine/evidence/monitor/refinement integration, nested ownership,
consumers, R5, activation/reader retirement and source-batch review remain open.

Rule-aware derivation checkpoint .22 reaches source `51634ed090a5`. Ordered source, operand,
native-read and adapted-callback evidence constructs complete isolated ordinary Rules, Facets and
mixed core/rule contributions through shared lowering. Every intermediate is validated before later
restriction or overwrite; source rules mode, historical effect identity, original native table timing
and source/derived/A/B isolation are preserved. Reader bookkeeping and native callback behavior remain
separate authorities. Complete reached closure, positions, two-JVM determinism, same-unit/precompiled
facet/helper variants, raw saved-tree replay and warm edit/restoration pass. Worker checks pass all
319 cases across ten framework commands; root independently passes 247 committed-source cases,
14-file formatting and all 164 tested/committed blob checks. Original fixtures, six samples and
531 protected entries remain unchanged; all 9428 parent paths are retained. Ten production Scala
paths add 1274 and remove 97 lines. The raw 137 Models stay unchanged; an exact independent ledger
corrects only 79 malformed primitive-state parameter slots across 61 Models to Bool. Frozen JSON EOF
whitespace remains a qualified advisory. External Hole callbacks are not executed; their nonempty
reference closure is checked structurally. Full .1/.2/.3/.4, complete Machine/evidence/monitor/
refinement/assuming assembly, model-wide helper/source-index and legacy reconciliation, nested and
cross-level ownership, actual Inbox supplier/G01/name/consumer validation, R5, activation/reader
retirement and source-batch review remain open. Temporal/tools/Go and broad integration gates stay held.

| Name | Status | What |
| --- | --- | --- |
| fn-174.1 | 🔄 in progress | Framework foundation committed; 23 focused cases and six-sample plugin compile pass, with direct rebind captures verified in bytecode. Full plugin sample parity open; mini/Temporal integration and full scoped gate held |
| fn-174.2 | 🔄 in progress | Framework construction, provenance, reference copies, companion facts, derived bindings and domain/rejection conversions committed through c2207956bd; 84 frozen focused cases and formatting pass. Temporal kit, mini oracle, complete reader retirement and full scoped gate remain held |
| fn-174.3 | ⬜ todo | After full .2, .10-.15 and .18: integrate the complete native signature, state-field Inbox ownership and consumer fairness checks; production switch, reader retirement and full acceptance remain here |
| fn-174.4 | ⬜ todo | After full .1/.3, .16-.23: integrate complete Machines, direct Rules/Facets and rule-aware derivation contributions, model-wide contextual helpers/source-index compatibility, complete derivation/refinement ownership, evidence, monitors and consumer validation; sole production activation/reader retirement remain here |
| fn-174.23 | 🔄 in progress | Capture authored/default Machine evidence and construct complete isolated dependency contributions; preserve direct-Match completion, inherited roots/positions and zero extra native evaluation; complete Machine integration remains in .4 |
| fn-174.5 | ⬜ todo | After .4: compositions and syncs |
| fn-174.6 | ⬜ todo | After .4: Properties, Scenarios, Queries and progress; the lifter's fold goes; comprehension fixture |
| fn-174.7 | ⬜ todo | After .6: capability expansions and gaps read values, not trees |
| fn-174.8 | ⬜ todo | After .5/.7 and fn-141.6: retire the lifter for IR; inspector for lints only; fixtures in-process; refusal ledger applied; rules of record; line counts; close fn-141 |

## Deferred for now

Deferred by the owner on 2026-10-11 until the Scala DSL work is done. Then, in one sweep: the Go halves
of the open specs above, one regeneration of IR, Cases, lint acceptances and fixtures, the IR
equivalence under the batch map (`make umpire-check-equivalence` with `DUMPER=`), the full Go gate,
one review and one live run. That sweep also closes the source-complete specs below. It first brings
the real features forward: it runs each DSL change's scalafix rule on them, does the rows marked
"⏸️ deferred (sweep)" above, and proves their equivalence against the batch-1d baseline.

- **Test speed** (branch `w-perf`, 11 commits, parked): Producer and lint check through their
  Realizer's interpretation (two builds instead of three, replay stays independent), `Row.Results`
  aliasing fixed, per-Model sharing in check/lint/conformance tests, streaming fingerprints (one full
  Activity Check 245 s → 55 s), fail-fast IR loading, `export`'s whole-Model tests opt-in
  (`make umpire-check-export`), no `-count=1` in the gate. Left: the ~3.5 GB live heap, lower/conformance
  sharing, a clean per-package before/after table, and cheaper Testpilot work accounting (defaults
  raised to 32M/16.4G by fn-148, to be lowered after measuring).

### fn-170: Clean up tools/umpire

Captured on 2026-10-12 at the owner's request. It runs in the Go propagation phase after the Scala
source batch. Earlier Go cleanups (fn-124, fn-84, fn-61, fn-66) are done. A read-only research pass (2026-10-12)
found layering good and most packages deep, with Temporal code inside four generic packages, two
alias facades and some drift; its findings are tasks .2-.4. Once it lands, the `tools/umpire`
items of fn-169.4 have moved here (fn-170.2), and `tools/umpire` mirrors `model/`: a Temporal-free
`framework/` and a `temporal/` adapter (owner, 2026-10-12).
[Spec](.flow/specs/fn-170-clean-up-toolsumpire.md).

| Name | Status | What |
| --- | --- | --- |
| fn-170.3 | ⏸️ deferred (Go) | Fold `internal/engine` into `interp` and `check`; remove the 56 alias re-exports (research, 2026-10-12); waits for fn-173.2 (owner, 2026-10-13) |
| fn-170.4 | ⏸️ deferred (Go) | Tidy: fold `ir/paths`, one duration conversion, `export` package name, over-exports, README/module-map drift |
| fn-170.2 | ⏸️ deferred (Go) | After .3/.4: layout mirroring `model/`: Temporal-free `tools/umpire/framework/` (incl. `export/quint`) and the `tools/umpire/temporal/` adapter; boundary test; outputs byte-identical (owner, 2026-10-12); waits for fn-173.2 (owner, 2026-10-13) |
| fn-170.1 | ⏸️ deferred (Go) | After .2: one `umpire` CLI with subcommands replaces the eight `cmd/umpire-*` binaries; the generic CLI is Temporal-free (API descriptors as a descriptor-set file, a target interface), its `main` in the Temporal adapter wires the Driver, SDK and test cluster; urfave/cli v2 unless the owner picks cobra; Make target names kept |

### fn-171: Clean up Testpilot

Captured on 2026-10-12 from a research pass over `common/testing/testpilot` and `tests/testcore/testpilot`.
Package jobs are clear, internals well hidden and the middle deep; no relayout is warranted, and Testpilot
stays in place. Generic with no Temporal import: `internal/ir`, `internal/verification`, `cel`, `predicate`,
`duration`, `casefile`, `recordedrun`, `evaluation`, `publish`, `campaign`, `replay`. The Case instruction
set and about 1,500 lines of `internal/execution` are Temporal's own. Held for the Go phase.
[Spec](.flow/specs/fn-171-clean-up-testpilot.md).

| Name | Status | What |
| --- | --- | --- |
| fn-171.1 | ⏸️ deferred (Go) | Generic core self-contained: remove the two back-edges into `temporal/binding`, an absolute import rule, offline readers off the runtime, Temporal files named in `execution` |
| fn-171.2 | ⏸️ deferred (Go) | One spelling per Driver type: the adapter imports `contract`; shrink the root's 60 aliases |
| fn-171.3 | ⏸️ deferred (Go) | Tidy: stale Case-version docs, package docs, test-only exports to `_test.go`, generated-case loaders to `tools/umpire/lower`, canary preflight import |

### fn-128: Close the activity's precision gaps

Moved out of the Scala DSL work on 2026-10-11: it edits only the standalone activity, which that
work no longer touches. Resumes at the sweep, on the migrated activity. Tasks 1-5 are done. Tasks 7
and 8 are parallel candidates; task 6 closes the shared batch after both corrections and all activity
sources.

| Name | Status | What |
| --- | --- | --- |
| fn-128.7 | ⏸️ deferred (sweep) | Accepted fatal-settlement classification through typed public last-failure evidence; unchanged fatal Property satisfied in scratch recording/replay — live recording deferred; waits for fn-173.2 (owner, 2026-10-13) |
| fn-128.8 | ⏸️ deferred (Go) | Separate caller-owned pauseResume and heartbeat-retry Profiles; immutable preflight/execution/replay binding, final-shape charge proof and admission negatives; waits for fn-173.2 (owner, 2026-10-13) |
| fn-128.6 | ⏸️ deferred (Go) | Evidence map, live Cases run once; close; waits for fn-173.2 (owner, 2026-10-13) |
### fn-129: Activity coverage

Moved out of the Scala DSL work on 2026-10-11, like fn-128: reset, `keepPaused` and the activity's
`find` Queries exist only in the standalone activity, not in the mini feature. Resumes at the sweep,
after fn-138. Tasks run in order.

| Name | Status | What |
| --- | --- | --- |
| fn-129.3 | ⏸️ deferred (sweep) | Reset with `keepPaused` and deferred settlement joined: 65 Queries, `deferredResetCompletes` lowers and prepares, reset-aware laws via `overriding(…, of = …)`; open: direct `keepPaused` live Case (lowering refuses a reset indistinguishable from the pause before it) — live Case deferred; waits for fn-173.2 (owner, 2026-10-13) |
| fn-129.4 | ⏸️ deferred (Go) | Exploration on the activity's `find` Queries; waits for fn-173.2 (owner, 2026-10-13) |
| fn-129.5 | ⏸️ deferred (Go) | New Cases listed, live run; close; waits for fn-173.2 (owner, 2026-10-13) |
### fn-168: Workflow Update: a real feature at the mini size over a workflow task foundation

Captured and deferred by the owner on 2026-10-11. The owner asked whether a real Temporal feature exists
that is small enough to be completely defined at the mini activity's size while showcasing the DSL and
serving as a basis for its rapid iteration; Workflow Update is that feature. Its documented semantics
(`docs/architecture/workflow-update.md`) hold one natural instance of nearly every hard construct fn-166
lists, plus constructs the mini activity lacks: `Idempotent` through update ID deduplication, an outcome
the caller sees but history never records (rejection), and a composition with a foundation that itself has
attempts and a deadline. What keeps it small is what keeps the mini activity small: dispatch is a
foundation. The spec adds `foundations/workflowtask` (Product and Protocol, about 400 lines: speculative,
transient and normal tasks, bounded attempts, the start-to-close timer, stale completion rejected) and
`features/update` (three levels, about 800 lines) over it. Scala only; the realization's worker
instruction (`UpdateReply`) and every Go half are the sweep's. The mini activity stays the loop's feature
and the guide's example. When taken up, its place in the batch is the owner's call: before fn-159 it is a
second proving ground migrated by rule; after fn-165 it is written against the final DSL. Not planned,
not marked ready. [Spec](.flow/specs/fn-168-workflow-update-a-real-feature-at-the.md).

### fn-155: Name the standalone activity's repeated patterns

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
| fn-155.6 | 🔄 in progress | R7 proof complete on fn-155.5 base `551bf89c68`: Nexus IR byte-identical, projection differences only the 4 mapped effect merges (7 bodies proven equal phase by phase, 25 controls), 46 tables/13,891,948 records equal (nil/empty slice normalization accepted), 31 Case files reconciled (21 controls); full gates deferred to the sweep |

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

### fn-156: Let the Scala compiler and lint enforce Model correctness

[Spec](.flow/specs/fn-156-let-the-scala-compiler-and-lint-enforce.md). Tasks .1–.6 and .8 are done and
merged (`39b7580104`). A fresh lift of the final source equals a fresh lift of the pre-fn-156 source byte
for byte, positions included (11 files, 25,191 positions), and `make lint-model` passes. The reports are
joined and enforcement is documented (`model/README.md`, `.plans/SCALA_CAPTURE_CHECKING.md`). Explicit
nulls was trialled and dropped (`.plans/SCALA.md`). Task .7 closes once a full Go suite and model gate
run on a tree that includes it, which the deferred sweep supplies; fn-155.6's whole-regeneration proof stays
a separate hold. The compile-versus-lift items from the review remain notes on fn-141.

| Name | Status | What |
| --- | --- | --- |
| fn-156.7 | ⬜ todo | Join reports, document enforcement, exact artifact comparison, full gates and reviews; close |

### fn-146: Adopt CEL for runtime predicates and values

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

### fn-147: Migrate elapsed-time fields to protobuf Duration

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

### fn-148: Consolidate Testpilot evidence and correlated state schemas

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
| fn-148.6 | 🔄 in progress | Local references, cardinality, Empty markers and coordinated format 4.0 activation |
| fn-148.7 | ⬜ todo | Artifact/companion migration, measurements, full gates and ownership docs; close |

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
| fn-150.1 | ⏸️ deferred | Quick rerun no longer OOMs (23.8 s, 5.75 GB free); exits 1 only on stale `model/ir` loads; green after the deferred regeneration; waits for fn-173.2 (owner, 2026-10-13) |
| fn-150.2 | 🔄 in progress | Structured fairness references end to end (Scala schema, lifter, Go admission/binding; engine 101 tests); full gates at the deferred sweep |
| fn-150.3 | 🔄 in progress | Composition `leadsTo` and composed `fair` authoring/lifting with located refusals (lifter 111/0, framework 60/0); gates at the deferred sweep |
| fn-150.4 | 🔄 in progress | Boundary, replay and fairness checkpoints plus lifted-fixture admission test; full gates/review at the deferred sweep |
| fn-150.5 | 🔄 in progress | `ServedDispatch`/`UnservedDispatch` example (passes / deadlock witness) and docs merged (`d6428e905b`); gates at the deferred sweep |

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

### fn-138: Retries and Deadline capabilities

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
| fn-157.1 | ⏸️ superseded | By fn-173: the memory it bounds comes from eager full-table interpretations, which fn-173 Phase 1 removes (owner, 2026-10-13) |
| fn-157.2 | ⏸️ superseded | By fn-173 Phase 1 (eager full tables removed) |
| fn-157.3 | ⏸️ superseded | By fn-173 Phase 1 (eager full tables removed) |
| fn-157.4 | ⬜ todo | Repair owned gate/lift scratch lifetimes or establish provisioned capacity with ordinary overlap |
| fn-157.5 | ⏸️ superseded | By fn-173 Phase 1 (eager full tables removed) |

### Deferred by the owner earlier

These stay deferred for their own reasons. fn-119.3, fn-125.2/.3 and fn-152 contain DSL work; they
return only when the owner revives them.

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
