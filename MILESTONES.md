# Umpire4 milestones

The current state of Umpire work: what is being built, what is left, and what is not being done.
Flow (`.flow/`, `flowctl`) is the record for specs and tasks; this page is the overview across them.

As of 2026-10-12.

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

The DSL framework (`model/framework`) stays Temporal-agnostic as far as is realistic: Temporal's capability
vocabulary, properties, realization vocabulary and kit live under `model/temporal/`, and the lifter and Testpilot IR
are the parts that are Temporal's driver tooling by design (fn-114.12, fn-122.8).

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
- **Migration rules are written, not run.** A DSL change still ships its scalafix rule, proven on the
  mini feature (fn-166 R6). It is not applied to the real features; the owner's sweep runs it.
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
| 4 | fn-160 | layout and realization layer (.3/.4) |
| 5 | fn-166.11 | lifter suites scoped to what they lift; production fixtures retargeted to mini |
| 6 | fn-141 | lifter refactor (byte-identical lifts), pulled forward by the owner on 2026-10-12 |
| 7 | fn-160 | Protocol level, flat `features/`, per-feature collection (.6/.7/.2) |
| 8 | fn-163 | system/product capabilities, named system capabilities |
| 9 | fn-147 | Duration-typed elapsed-time authoring (source integrated; proof/closure open) |
| — | fn-140 | witness Queries and `.live` (source done; Go proofs held) |
| — | fn-149 | safety and liveness groups (source done; Go proofs held) |
| 10 | fn-164 | `expect:` blocks for Properties and consistent colon style |
| — | seal | the mini feature's step-table equivalence before the first meaning change |
| 11 | fn-123 | fault declarations, durability, `crashes(...)` |
| 12 | fn-153 | Property illustrations in the DSL |
| 13 | fn-165 | inputs as records: named classes, declared defaults, `:=` removed from classes (after fn-141) |

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
state; nothing is parked or deleted. A DSL change ships its migration as a scalafix rule proven on
the mini feature, so moving it into the real features is running the rule. The guide uses the mini
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
| fn-166.11 | ⬜ todo | Lifter suites scoped to what they lift: admission covers only sources the lifted roots reach; production-fixture tests retargeted to mini or held for the sweep; before/after suite timing (owner, 2026-10-12) |

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
level folders grouped by name prefix, and realizations get their own `realization/` layer. Since 2026-10-11 the tree is flat: a kind folder holds only shared parts and each form is a `<kind>_<form>` folder (fn-160.7). Layout, naming and the Protocol rename apply to the mini feature and the shared code; moving and renaming the real features' files waits for the sweep. Not marked ready.
[Spec](.flow/specs/fn-160-one-ir-file-per-feature-collected-from.md).

| Name | Status | What |
| --- | --- | --- |
| fn-160.2 | ⬜ todo | Collect one IR file per feature package, and per foundation/actor/kind package that declares Queries or realizations; remove `irFile`/`exports`; exploration checks only the asked Query; generic native realization discovery still needs source proof; regenerate and migrate consumers — Scala part open; Go part deferred |
| fn-160.3 | ⬜ todo | One machine per file, named after it; flat level folders grouped by name prefix (no nested packages); forms' files `<Form><Kind>.scala`; forms' shared parts under the kind's `shared/` (Nexus Product → `nexus/shared/product/`); the lifter enforces it |
| fn-160.4 | ⬜ todo | Realization layer: flat `realization/` folder, `<Subject>Run` naming, one-way layer dependencies, API only in realizations (after fn-160.3) |
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
| fn-123.3 | ⏸️ deferred (Go) | Go derives the crash row; `SEMANTICS.md` Faults section (proof) |
| fn-123.4 | ⬜ todo | Task-queue providers converted; `fault-overridden` lint; derived storage-loss assumption |
| fn-123.5 | ⬜ todo | Budgets: `budgetedBy`, four table rules in Go, `LostStartAnswer` bound to `lossAvailable` — Scala part open; Go part deferred |
| fn-123.6 | ⏸️ deferred (Go) | Choice-level fault performance; lowering refuses model-only faults and unperformed choices |
| fn-123.7 | ⏸️ deferred (Go) | Trace output and Quint export of derived crashes and budgets |
| fn-123.8 | ⏸️ deferred (Go) | `umpire-faults` report, docs; close |

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

### fn-141: Shrink the IR generator: one description of each DSL construct

Ready 2026-10-08. Pulled forward by the owner on 2026-10-12: it runs after fn-160.3/.4 and
fn-166.11, before the remaining DSL specs, which then land on the smaller lifter. Its
"spellings no Model uses" recount (fn-141.2) counts the current mini and shared code only, and must
keep any spelling an open spec in this batch is about to use.
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
| fn-141.14 | ⬜ todo | Dead lifter code, rules of record, docs, backends check, size report; close — backends check deferred |

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

### fn-128: Close the activity's precision gaps

Moved out of the Scala DSL work on 2026-10-11: it edits only the standalone activity, which that
work no longer touches. Resumes at the sweep, on the migrated activity. Tasks 1-5 are done. Tasks 7
and 8 are parallel candidates; task 6 closes the shared batch after both corrections and all activity
sources.

| Name | Status | What |
| --- | --- | --- |
| fn-128.7 | ⏸️ deferred (sweep) | Accepted fatal-settlement classification through typed public last-failure evidence; unchanged fatal Property satisfied in scratch recording/replay — live recording deferred |
| fn-128.8 | ⏸️ deferred (Go) | Separate caller-owned pauseResume and heartbeat-retry Profiles; immutable preflight/execution/replay binding, final-shape charge proof and admission negatives |
| fn-128.6 | ⏸️ deferred (Go) | Evidence map, live Cases run once; close |

### fn-129: Activity coverage

Moved out of the Scala DSL work on 2026-10-11, like fn-128: reset, `keepPaused` and the activity's
`find` Queries exist only in the standalone activity, not in the mini feature. Resumes at the sweep,
after fn-138. Tasks run in order.

| Name | Status | What |
| --- | --- | --- |
| fn-129.3 | ⏸️ deferred (sweep) | Reset with `keepPaused` and deferred settlement joined: 65 Queries, `deferredResetCompletes` lowers and prepares, reset-aware laws via `overriding(…, of = …)`; open: direct `keepPaused` live Case (lowering refuses a reset indistinguishable from the pause before it) — live Case deferred |
| fn-129.4 | ⏸️ deferred (Go) | Exploration on the activity's `find` Queries |
| fn-129.5 | ⏸️ deferred (Go) | New Cases listed, live run; close |

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
| fn-150.1 | ⏸️ deferred | Quick rerun no longer OOMs (23.8 s, 5.75 GB free); exits 1 only on stale `model/ir` loads; green after the deferred regeneration |
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
| fn-157.1 | ⏸️ deferred | Owner deferred after three immutable seals; complete primary oracle still missing; constructor experiment not started; no repair or gate-fit credit |
| fn-157.2 | ⬜ todo | Repair only the measured private native owner; ownership, mutation isolation and independent replay proof |
| fn-157.3 | ⬜ todo | Bound implicated consumer and complete seven-Model generator lifetimes; preserve full outputs and assertions |
| fn-157.4 | ⬜ todo | Repair owned gate/lift scratch lifetimes or establish provisioned capacity with ordinary overlap |
| fn-157.5 | ⬜ todo | Join full preservation, execute interrupted tests and restore exact native gates and measured capacity |

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
