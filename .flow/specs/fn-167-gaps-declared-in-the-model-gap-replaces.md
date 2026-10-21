# Gaps declared in the Model: `@gap` replaces the lint acceptance file

## Execution status (2026-10-21 backlog audit)

Deferred by the owner. The Rust decision has been made and the active workspace is `model/`; the former Scala tree is frozen under `model/archived/` and is not expected to build (AGENTS.md and MILESTONES.md). The earlier “until the Rust verdict” note is historical, not an automatic release of this spec. Existing task states and old SHIP reviews describe historical checkpoints, not current implementation readiness.

Resume only after an explicit owner decision and a fresh scope/dependency review against the Rust architecture. Scala paths and commands below describe the archived layout; do not run them as current gates or mechanically port the plan. Preserve unfinished requirements and completion evidence; successor Rust work does not discharge this spec without an explicit requirement mapping. Taskless proposals remain deferred captures, not executable plans.


DSL changes ship a task-named Scalafix rule under `model/migrate/rules/`, prove it on copied `activity_mini` sources before the real-feature sweep, and follow `model/migrate/README.md`; production loading remains held in fn-166.10.

> HTML render lens: `.flow/artifacts/fn-167-gaps-declared-in-the-model-gap-replaces/spec.html` — local-only; regenerable, markdown is the record. <!-- flow-next:artifact-link -->

## Conversation Evidence

> user (2026-10-11): "we're managing temporal/model/temporal/features/activity/standalone/generated/ir.lint.json outselves by hand?? that seems crazy, there are so many lints!"
> user: "what's the suggested re-design"
> user: "instead of this file; should we have an annotation in the mdoel itself? to colocate them?"
> user: "also; where is the line between the model/ linter and the tools/umpire linter?"
> user: "what would annotations look like?"
> user: "could we have literal `@...` annotations for the parts that otehrwise need to be in the separate file?"
> user: "is @accepted the right word?"
> user: "can there be more than one annotaiton with a because each?"
> user: "what about @ignored or sth?"
> user: "use @gap - then write the flow spec"
> agent measurement (shown to the user): the standalone activity's acceptance file holds 179 entries and 5,089 owner-by-subject pairs, which collapse to about 530 distinct kind-by-subject pairs and about 120 distinct reasons across 35 owners; the two nexus files hold 69 and 61 pairs; the legacy files 997 more. `silent-rejection` (1,887) and `unconstrained-result` (1,804) are most of it, and derived owners such as `byIDCompletion` each repeat the same 118 classes of their base with a templated reason. Those derived owners are not findable by name in the current Scala sources.

## Goal & Context
<!-- scope: business -->
<!-- Goal & Context: 40% [user], 40% [paraphrase], 20% [inferred] -->

The lint's accepted findings live in a hand-written JSON file beside each feature's IR, and the owner found the standalone activity's at six thousand lines [user]. The file was designed under fn-120 R7 when findings were few, so that every accepted hole carries a human reason; it is read by `umpire-lint`, never written by it, and the only entries a tool maintains are the capability waivers the gate copies in from the Model's `except` and `overriding` declarations [paraphrase]. Two things broke it. The IR does not record derivation, so the lint reports every inherited row again on every derived owner, and each regeneration since fn-151 extended the subject lists by hand instead of redesigning them [inferred]. The reasons now name owners that no longer exist under those names, which a reason in a separate file cannot notice [paraphrase].

The owner's direction: the acceptance should be an annotation in the Model, colocated with what it explains, one annotation per reason, and the word is `gap` [user]. This spec makes `@gap` the one way a Model declares a known gap, makes the lifter carry gaps into the IR as inert metadata, makes derivation visible to the lint so inherited findings are reported once, and retires the acceptance file for every feature that has sources. It keeps the lint's kinds and judgments unchanged: what the lint finds is the same, only where the Model answers it moves [inferred].

The owner also asked where the line between the Scala lint and the Go lint runs. The answer this spec records: the Scala side holds the source to its form (formatting, scalafix, sugar with a core form, comments, vocabulary); the Go side holds the lifted IR to its semantics (what nothing reaches, takes, asks, evidences or realizes); the gate is the seam. Gaps are authored on the Scala side and judged on the Go side, which is where every other declaration already sits [paraphrase].

## Architecture & Data Models
<!-- scope: technical -->

**The annotation.** [user, inferred] One Scala annotation class in the framework, `gap`, carries a list of subjects, a reason and an optional expiry:

```scala
@gap(silent(activityWorker.respondCompleted, in = Phase.scheduled, Phase.paused))(
  because = "a worker answers only an attempt it was given", until = "fn-120 R5")
@gap(unconstrained(client.heartbeat, client.backoff))(because = "results are read through the protocol refinement")
object ActivityProduct extends Machine[State, Outcome, Fact], ...
```

```scala
@gap(unperformed(client.reset), uncovered(start.withScheduleToClose))(because = "...")
object ExhaustAfterHeartbeatRun extends Realization(...)

@gap(witnessOnly)(because = "a witness proves one path; free verification needs the Property to read repeated terminations")
val terminateSettles = property(...)
```

The annotation is repeatable: a declaration carries one `@gap` per reason, in source order. It declares metadata, not behavior; inertness is a tested contract, not a consequence of extending `StaticAnnotation`. Its arguments are typed trees, never string identities. A disposable Scala compile confirmed repeated curried annotations on objects and vals; compiler output reverses their order and mirrors object annotations onto the module val and class. The lifter orders by source position and deduplicates synthetic mirrors, while refusing genuine duplicate user declarations. Full TASTy extraction and rename behavior remain proof obligations of the early task.

**The subjects.** [inferred] One inert constructor per lint kind, named by the kind with its noun dropped, so the mapping from annotation to finding is mechanical: `silent(action, in = phases*)` (silent-rejection), `byDefault(action, in = phases*)` (disabled-by-default), `neverEnabled(action)`, `unconstrained(classes*)`, `unproduced(outcomesOrFacts*)`, `unperformed(actions*)`, `unevidenced(facts*)`, `uncovered(classes*)`, `unmapped(apiValues*)`, `unrealized`, `unread`, `witnessOnly`, `untaken(choices*)`, `unasked`, `unfired`, `unreachable(values*)`, `stuck(states*)`, `unreadObservation(observations*)`, `explicitWait(polls*)`. A subject's arguments are the Model's own references (an action, a class, a fact, a property, an API enum value), typed as the framework declares them, never strings. `witnessOnly`, `unrealized`, `unread`, `unasked` and `unfired` take no argument: their subject is the declaration they annotate.

**Owners and placement.** A declaration's source identity and a finding's owner are distinct. Machine/composition table gaps use the existing machine/composition owner; a facet is attributed to each concrete consuming machine. Realization `unreachable-binding` and `uncovered-class` findings use the realization name, while action/fact/API/observation/wait evidence findings aggregate under its machine. Property `witnessOnly` and `unasked` use the Property's machine and Property name; Query `unfired` and `unrealized` use the Property's or Scenario's machine and Query name; refinement `unread` keeps its existing machine and structural subject. No finding key is renamed merely to match annotation placement. Whole-action shorthand expands to the existing concrete-class subjects and canonical phase lists. Add `unreachableBinding` and optional `mustNotPinned` to the constructor inventory; optional-kind behavior stays unchanged.

**Derivation in the IR.** A lifted machine records its source and ordered derivation operations: kept, rebound and added bindings, monitor/refinement drops, refinement replacement and added assumptions. These describe existing behavior, never change it. An unchanged binding alone does not prove an unchanged finding: restrictions, assumptions and refinement can change judgments. The lint first computes existing findings unchanged, then suppresses only proven duplicate inherited findings and reports them on the base. Derived-only findings remain visible. A base gap covers only the corresponding inherited finding. The size of the reduction is measured, not assumed.

**Gaps in the IR.** [inferred] Each `@gap` lowers to one entry per subject in the Model's `gaps`: kind, owner, the structural subject the lifter computes in the lint's own subject form, reason, expiry and source position. Gaps are inert: outside the fingerprint, outside equivalence identity, absent from the table pins, exactly as fn-120's choice names are. The capability waivers of `except` and `overriding` lower the same way, as gaps of kind `capability-waiver`, so the transient waivers file and the gate's copy-into-acceptances step go away.

**The lint.** `umpire-lint` matches finding-backed gaps by kind, owner and subject, as it matches acceptance entries today. A finding no gap covers fails as now; an unmatched finding-backed gap fails at its source position. Synthesized `capability-waiver` metadata retains the existing stale-check exemption: it is validated through the capability declaration, not matched to a nonexistent lint finding. No new lint kind or waiver annotation is added. A gap whose `until` names a done Flow spec fails at its position. Acceptance files are rejected for source features and retained for legacy payloads until their retirement owner removes them.

**The gate.** One source rule beside the sugar rule: the only Model gap/acceptance annotation is the framework's `gap`, with framework-owned subject constructors. Existing compiler annotations remain allowed; this does not ban compiler directives such as `unused`, `targetName` or `unchecked`. Unsupported placements, lookalike gap annotations and foreign subject constructors fail at their source positions. The gate's waiver-copy and acceptance-update paths are removed only with the transport and consumer switch in the sweep.

**Migration.** The present acceptance files are the input. A mechanical rule drops proven redundant inherited acceptances and writes the remaining finding-backed `@gap` stacks by declaration and reason, preserving each reason's text. Capability-waiver entries are reconciled against the existing `except`/`overriding` reason and lower automatically; the rule preserves those behavior-bearing declarations rather than replacing them or inserting a second waiver annotation. A mismatch is reported, not silently rewritten. The mini proves one annotated hole and one automatically transported existing waiver before real-feature migration in the sweep.

## API Contracts

**Framework.** [inferred]

```scala
final class gap(subjects: Subject*)(because: String, until: String = "") extends StaticAnnotation
sealed trait Subject
```

**IR.** [inferred] `Model.gaps: repeated Gap`, `Gap { kind, owner, subject, because, until, position }`; `Machine.derives: Derivation { from, kept, rebound, added, dropped_monitors, dropped_refinement }`. Both are metadata: the IR fingerprint, equivalence comparison and table pins ignore them.

**Lint.** [inferred] `umpire-lint` keeps its command line. Its failure summary counts unaccepted findings, stale gaps and expired gaps. The `Accepted` reader survives only for `generated/legacy/`.

**Until.** `until` names a Flow spec id, optionally with a requirement, `fn-120` or `fn-120 R5`. Resolve an exact canonical id first, otherwise a unique short-id match; missing or ambiguous references fail at lift time. The optional requirement suffix is a reference, not a new requirement-level completion policy. The lint reads the spec's authoritative status from Flow's record. The status adapter and its shared runtime-state handling are not yet established and must be proven before the sweep consumer is admitted.

## Edge Cases & Constraints

- **A finding key `(kind, owner, subject)` appears in at most one user-authored gap.** The second is a lift error naming both source positions; synthetic compiler mirrors are not second declarations.
- **Every `@gap` names at least one subject and a non-empty reason.** [inferred] Lift error otherwise.
- **A subject that resolves to nothing** on its declaration (an action the machine does not bind, a class no action of it has, a fact it does not declare) is a lift error at the annotation. [inferred]
- **A gap on a derived machine for an inherited row** is a lift error that names the base to declare it on. [inferred]
- **An annotation on a declaration the lifter does not read** (a helper def, a test) is a gate error from the source rule. [inferred]
- **Current mode.** [user, paraphrase] Only Scala work in the Model is open. The lifter consumes generated ScalaPB classes from the shared protobuf schema; there is no independent Scala schema to edit. Current tasks implement typed authoring, source admission, internal resolved metadata, derivation inspection and a pure migration transformation. They retain existing ProtoJSON and waiver output. Schema changes, generated code, metadata serialization, Go consumers, transport retirement and full migration/regeneration belong to explicitly held sweep tasks. Source groundwork is not completion of the requirements that also need those consumers. Real-feature sources and artifacts stay untouched.
- **Legacy payloads** keep their acceptance files untouched until fn-160 retires them. [paraphrase]
- **No lint kind changes meaning** and no finding is removed from the lint by this spec except by derivation inheritance; the counts before and after are recorded in the migration's evidence. [inferred]

## Acceptance Criteria

- **R1:** The framework declares a repeatable `gap` annotation, `@gap(subjects*)(because, until = "")`, and one inert subject constructor per lint kind, each taking the Model's typed references and never a string. Errors: a `@gap` with no subject or a blank reason fails the lift at its position; so does a subject that resolves to nothing on the annotated declaration. [user]
- **R2:** *(2026-10-18, one-language series: moved to fn-200.3: gaps resolve natively, not through IR entries)* The lifter lowers every `@gap` to IR gap entries (kind, owner, subject in the lint's subject form, reason, until, position) from supported machine, composition, realization, query, property or refinement declarations, with the finding-owner mapping above and facet attribution to consuming machines. Source order is retained and synthetic mirrors emit once. Errors: a duplicate `(kind, owner, subject)` fails the lift naming both positions.
- **R3:** *(2026-10-18, one-language series: moved to fn-200: no IR, no fingerprint or equivalence interaction remains)* Gaps and derivation are inert in the IR: the fingerprint, the equivalence comparison and the table pins are byte-identical with and without them. Errors: no error surface beyond the existing equivalence checks.
- **R4:** *(2026-10-18, one-language series: moved to fn-200.3: waivers are a gap kind; no waivers file)* `except` and `overriding` lower to gaps of kind `capability-waiver` with their reasons; the transient waivers file and the gate's step that copies waivers into the acceptance file are removed. Errors: none beyond R1's.
- **R5:** A lifted machine records its source and derivation changes, including chains, kept/rebound/added bindings, dropped monitors/refinement, replaced refinement and added assumptions. The lint reports proven duplicate inherited findings once on the base, with base gaps covering those findings, without changing any kind's judgment or hiding derived-only findings. Errors: a derived gap for a proven inherited finding fails naming the base; binding equality alone is not evidence of finding equivalence. [inferred]
- **R6:** *(2026-10-18, one-language series: moved to fn-200.3: the native judge replaces `umpire-lint` matching)* `umpire-lint` matches findings against finding-backed IR gaps; an unmatched such gap fails at its source position and an unaccepted finding fails as today. Synthesized `capability-waiver` metadata retains its existing stale-check exemption and is validated through the capability declaration, without a new lint kind. A source feature with an acceptance file fails with a removal message; legacy payloads retain file compatibility. Errors: as stated.
- **R7:** *(2026-10-18, one-language series: moved to fn-200.3: spec-reference expiry is checked by the native judge)* A gap whose `until` names a Flow spec recorded as done fails the lint naming the position and the spec; missing or ambiguous spec references fail the lift. An optional requirement suffix does not change spec-level expiry. Errors: as stated. [inferred]
- **R8:** The gate admits only the framework's `gap` as a Model gap/acceptance annotation and only framework subject constructors; existing compiler annotations remain permitted. Errors: lookalike annotations, foreign constructors and unsupported helper/test placements fail at source positions, reported like the sugar rule.
- **R9:** *(2026-10-18, one-language series: stays with the real-feature sweep, using fn-200's lint rather than IR plus .lint.json)* A mechanical migration reads a feature's IR and acceptance file and writes finding-backed `@gap` stacks by owning declaration and exact reason, dropping only what R5 proves redundant. The mini proof leaves one hole annotated with `@gap` and its existing `except`/`overriding` waiver automatically lowered to gap metadata with the same reason; no second waiver annotation or behavior change is introduced. Errors: unplaceable acceptances, unsafe redundancy and mismatched waiver reasons are reported, not dropped or rewritten. [inferred]
- **R10:** *(2026-10-18, one-language series: stays with the real-feature sweep; the before counts remain the recorded evidence)* After the sweep, no feature with sources carries an acceptance file, and the migration's evidence records each feature's finding counts and gap counts before and after, with the standalone activity's 179 entries and 5,089 pairs as the before. Errors: no error surface beyond R6. [paraphrase]

## Boundaries

- No lint kind is added, removed or changed in what it judges. [inferred]
- No arm-level sugar: a silent rejection is declared from the machine with `silent(action, in = ...)`, or decided as a row with the existing `rejects(why).because(...)`. One mechanism. [inferred]
- The legacy payloads' acceptance files are not migrated; fn-160 retires them with the payloads. [paraphrase]
- The Go lint, the file's retirement and the real features' migration wait for the owner's sweep; this spec does not reopen the Go side. [user, paraphrase]
- Reasons are not rewritten for content during migration; stale wording is a later cleanup, not this spec's. [inferred]
- No `@gap` on a whole feature or package: a gap names a declaration. [inferred]

## Decision Context

Maintainability (plan review): duplication - R4/R9 repeated the waiver reason in an annotation and its existing declaration; resolved by automatic transport from `except`/`overriding` alone; structure - none identified.

**Planning boundary.** Source-only groundwork precedes a held transport/consumer/migration sweep. Internal collected metadata is a lifter implementation detail, not a second IR, sidecar format or public exporter. No whole-spec dependency is added on the mini or legacy-layout work. The source migration proof consumes the mini protocol's committed closure; the held migration adapter consumes the migration infrastructure's committed closure. This supersedes any earlier instruction to merge source-backed acceptance files during layout migration: source features migrate to gaps, while legacy-file retention and payload retirement remain the legacy-layout owner's work.

**Unconfirmed inferences.** Derivation-based suppression across every lint kind, expiry's authoritative Flow adapter and full migration placement/redundancy remain unconfirmed (R5, R7, R9). Their tasks must demonstrate the stated boundaries before treating them as supported. Typed references do not by themselves prove automatic rename migration, and no percentage reduction is promised. Broad generated API drift gates and new CI coverage remain excluded by the existing declined decision.

**Why not the file.** It is the lint's word in the lint's place: a string identity that goes stale on a rename, extended by hand on every regeneration, and blind to derivation, so a six-thousand-line file holds about 120 decisions. A reason in the Model cannot outlive the thing it explains.

**Why an annotation and not sugar.** An annotation is inert by construction, which is what an acceptance must be; it needs no core form; and the lifter already reads trees off TASTy for `except`, so reading `@gap` is the same match. Its one limit, that it sits on declarations and not on arms, is met by naming the pair from the machine, which also keeps the surface to one mechanism.

**Why `gap`.** `accepted` is the lint's passive word and collides with `Outcome.accepted`. `ignored` says the finding was looked away from, means "skipped" in munit, and is the right word for an empty effect on an arm, where it should stay free. `hole` names only H1 to H5; the coverage kinds are gaps in evidence, not in specification. `waived` releases a requirement, and a silent rejection is a decision deferred, not released. `leaves` reads well and was passed over for a noun. `gap` is the word the existing reasons already use for themselves and fits every kind (owner, 2026-10-11).

**Why one annotation per reason.** It is the acceptance file's own unit, moved next to its subjects; a stack of short annotations keeps each reason beside what it explains.

**Why `until`.** The residue that would otherwise still need a file is the time-bound kind, "kept until fn-160 separates dispatch." Naming the spec turns a sentence nobody rereads into a check that fires when the spec closes, and removes the last reason to keep a file.

**Why derivation first.** Nine tenths of the file is inherited rows repeated per derived owner. Making the lint see derivation removes them without a single annotation and is the only change that needs no author words.

**The two linters.** The Scala side lints the source's form and the Go side lints the IR's semantics; the gate is the seam. Gaps straddled that line, authored in Scala terms, judged in Go, half-written by the Scala gate. This spec puts authoring wholly on the Scala side and judging wholly on the Go side.

2026-10-18, one-language series: R2, R3, R4, R6 and R7 move to fn-200 (Model lint is native; `@gap` is the only acceptance). R9 and R10 stay with the real-feature sweep and run on fn-200's lint instead of the IR and `.lint.json`. Tasks .5 and .6 are closed as obsolete (lane A); .7 is recut by lane C.

2026-10-18, deferred (owner): all specs other than fn-210 (the Rust prototype of the framework in `temporal/model2`) are deferred until fn-210's write-up recommends whether to extend the Rust prototype or stay on Scala. Flow has no deferred status; this line is the record. MILESTONES.md was reset to the prototype alone and no longer lists this spec. Nothing in this spec changes; resume or close it after that decision.

## Quick commands

Run Scala commands against independently validated existing generated jars; do not regenerate those jars as part of source-only tasks. A held prerequisite is reported, never counted as a pass.

```bash
mise exec -- scala-cli compile model/project.scala model/framework --server=false
mise exec -- scala-cli test model/irgen --server=false --test-only 'umpire.irgen.ConventionsFixtures'
mise exec -- scala-cli test model/check --server=false --test-only 'umpire.check.EquivalenceSuite'
```

## Early proof point

The first task proves repeated typed annotations survive TASTy with correct declaration attribution, source order and synthetic-mirror handling. If that fails, reconsider the annotation extraction approach before admitting later tasks.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | The framework declares a repeatable `gap` annotation, `@gap(subjects*)(because, until = "")`, and one inert subject constructor per lint kind, each taking the Model's typed references and never a string. Errors: a `@gap` with no subject or a blank reason fails the lift at its position; so does a subject that resolves to nothing on the annotated declaration. | fn-167-gaps-declared-in-the-model-gap-replaces.1, fn-167-gaps-declared-in-the-model-gap-replaces.2 | — |
| R2 | The lifter lowers every `@gap` to IR gap entries (kind, owner, subject in the lint's subject form, reason, until, position) from supported machine, composition, realization, query, property or refinement declarations, with the finding-owner mapping above and facet attribution to consuming machines. Source order is retained and synthetic mirrors emit once. Errors: a duplicate `(kind, owner, subject)` fails the lift naming both positions. | fn-167-gaps-declared-in-the-model-gap-replaces.1, fn-167-gaps-declared-in-the-model-gap-replaces.2, fn-167-gaps-declared-in-the-model-gap-replaces.5 | — |
| R3 | Gaps and derivation are inert in the IR: the fingerprint, the equivalence comparison and the table pins are byte-identical with and without them. Errors: no error surface beyond the existing equivalence checks. | fn-167-gaps-declared-in-the-model-gap-replaces.2, fn-167-gaps-declared-in-the-model-gap-replaces.3, fn-167-gaps-declared-in-the-model-gap-replaces.5 | — |
| R4 | `except` and `overriding` lower to gaps of kind `capability-waiver` with their reasons; the transient waivers file and the gate's step that copies waivers into the acceptance file are removed. Errors: none beyond R1's. | fn-167-gaps-declared-in-the-model-gap-replaces.2, fn-167-gaps-declared-in-the-model-gap-replaces.5, fn-167-gaps-declared-in-the-model-gap-replaces.6 | — |
| R5 | A lifted machine records its source and derivation changes, including chains, kept/rebound/added bindings, dropped monitors/refinement, replaced refinement and added assumptions. The lint reports proven duplicate inherited findings once on the base, with base gaps covering those findings, without changing any kind's judgment or hiding derived-only findings. Errors: a derived gap for a proven inherited finding fails naming the base; binding equality alone is not evidence of finding equivalence. | fn-167-gaps-declared-in-the-model-gap-replaces.3, fn-167-gaps-declared-in-the-model-gap-replaces.5, fn-167-gaps-declared-in-the-model-gap-replaces.6 | ⚠️ unconfirmed suppression; per-kind sweep proof required |
| R6 | `umpire-lint` matches findings against finding-backed IR gaps; an unmatched such gap fails at its source position and an unaccepted finding fails as today. Synthesized `capability-waiver` metadata retains its existing stale-check exemption and is validated through the capability declaration, without a new lint kind. A source feature with an acceptance file fails with a removal message; legacy payloads retain file compatibility. Errors: as stated. | fn-167-gaps-declared-in-the-model-gap-replaces.6 | — |
| R7 | A gap whose `until` names a Flow spec recorded as done fails the lint naming the position and the spec; missing or ambiguous spec references fail the lift. An optional requirement suffix does not change spec-level expiry. Errors: as stated. | fn-167-gaps-declared-in-the-model-gap-replaces.2, fn-167-gaps-declared-in-the-model-gap-replaces.6 | ⚠️ unconfirmed authoritative status adapter; sweep proof required |
| R8 | The gate admits only the framework's `gap` as a Model gap/acceptance annotation and only framework subject constructors; existing compiler annotations remain permitted. Errors: lookalike annotations, foreign constructors and unsupported helper/test placements fail at source positions, reported like the sugar rule. | fn-167-gaps-declared-in-the-model-gap-replaces.1 | — |
| R9 | A mechanical migration reads a feature's IR and acceptance file and writes finding-backed `@gap` stacks by owning declaration and exact reason, dropping only what R5 proves redundant. The mini proof leaves one hole annotated with `@gap` and its existing `except`/`overriding` waiver automatically lowered to gap metadata with the same reason; no second waiver annotation or behavior change is introduced. Errors: unplaceable acceptances, unsafe redundancy and mismatched waiver reasons are reported, not dropped or rewritten. | fn-167-gaps-declared-in-the-model-gap-replaces.4, fn-167-gaps-declared-in-the-model-gap-replaces.7 | ⚠️ full placement/redundancy proof remains task 7 |
| R10 | After the sweep, no feature with sources carries an acceptance file, and the migration's evidence records each feature's finding counts and gap counts before and after, with the standalone activity's 179 entries and 5,089 pairs as the before. Errors: no error surface beyond R6. | fn-167-gaps-declared-in-the-model-gap-replaces.7 | — |

## Scope: the mini feature only (owner, 2026-10-11)

Within the Scala DSL batch (`MILESTONES.md`), this spec changes only the framework, the lifter, the shared code and `features/activity_mini`. The real features (standalone activity, Nexus workflow, Nexus standalone and their legacy payloads) are not edited, migrated, lifted or regenerated, and they stay behind until the owner's sweep, which runs this spec's scalafix rules on them. Equivalence compares `FEATURES=activity_mini` against the mini baseline. This overrides any statement in this spec that edits, migrates or proves on a real feature. See fn-166's Scope.

## Redesign impact (fn-173, 2026-10-13)

Owner-approved redesign `fn-173` (reachable exploration; `model/design/reachable-exploration.md`) drops full catalog tables, Scala step tables and equivalence sealing, and Query totals; the Behavior Fingerprint becomes an IR structural hash. For this spec:

- **.5/.6/.7:** fn-173's `open-graph` acceptance needs a gap subject, because this spec retires `ir.lint.json`. Lint matches the explored graph only.
- **Fingerprints:** fingerprint exclusion becomes automatic (the structural hash reads only allowlisted fields), and the Scala equivalence exclusion is moot.
