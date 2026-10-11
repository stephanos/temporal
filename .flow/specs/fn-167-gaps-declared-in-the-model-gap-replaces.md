# Gaps declared in the Model: `@gap` replaces the lint acceptance file

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
@gap(
  silent(activityWorker.respondCompleted, in = Phase.scheduled, Phase.paused),
  silent(activityWorker.respondFailed, in = Phase.scheduled, Phase.paused)
)(because = "a worker answers only an attempt it was given, and the server rejects a stale one",
  until = "fn-120 R5")
@gap(unconstrained(client.heartbeat, client.backoff))(
  because = "results are read through the protocol refinement; this machine pins phase transitions only"
)
object ActivityProduct extends Machine[State, Outcome, Fact], ...

@gap(unperformed(client.reset), uncovered(start.withScheduleToClose))(because = "...")
object ExhaustAfterHeartbeatRun extends Realization(...)

@gap(witnessOnly)(because = "a witness proves one path; free verification needs the Property to read repeated terminations")
val terminateSettles = find(...)
```

The annotation is repeatable: a declaration carries one `@gap` per reason, in source order. It is an annotation and not sugar because an annotation cannot change behavior, which is the property the modalities plan requires of acceptances (inert metadata, outside the fingerprint), and because it needs no core form, so the sugar rule stays untouched. Its arguments are trees in TASTy, so a subject is a typed reference the compiler checks and a rename moves, never a string [inferred].

**The subjects.** [inferred] One inert constructor per lint kind, named by the kind with its noun dropped, so the mapping from annotation to finding is mechanical: `silent(action, in = phases*)` (silent-rejection), `byDefault(action, in = phases*)` (disabled-by-default), `neverEnabled(action)`, `unconstrained(classes*)`, `unproduced(outcomesOrFacts*)`, `unperformed(actions*)`, `unevidenced(facts*)`, `uncovered(classes*)`, `unmapped(apiValues*)`, `unrealized`, `unread`, `witnessOnly`, `untaken(choices*)`, `unasked`, `unfired`, `unreachable(values*)`, `stuck(states*)`, `unreadObservation(observations*)`, `explicitWait(polls*)`. A subject's arguments are the Model's own references (an action, a class, a fact, a property, an API enum value), typed as the framework declares them, never strings. `witnessOnly`, `unrealized`, `unread`, `unasked` and `unfired` take no argument: their subject is the declaration they annotate.

**Owners and placement.** [inferred] A `@gap` sits on the declaration that owns the finding: a machine or composition object for the table kinds, a realization object for the coverage and evidence kinds, a query or property or refinement declaration for the query kinds. A gap on a facet's `rules` object is attributed to the machine that mixes the facet in. A gap on a derived machine may name only what that machine changed; a gap for an inherited row is declared on the base and inherited with it.

**Derivation in the IR.** [inferred] A lifted machine records the machine it derives from and what it changed: the actions it kept under `restrict`, the actions it rebound, the bindings it added, and whether it dropped its source's monitors or refinement. The lint reads this and reports a finding on a derived owner only when the finding is about something the owner changed; a finding on an inherited row is reported once, on the base. A gap declared on the base covers the inherited row on every derived owner. This is what removes the bulk, and it needs no author words.

**Gaps in the IR.** [inferred] Each `@gap` lowers to one entry per subject in the Model's `gaps`: kind, owner, the structural subject the lifter computes in the lint's own subject form, reason, expiry and source position. Gaps are inert: outside the fingerprint, outside equivalence identity, absent from the table pins, exactly as fn-120's choice names are. The capability waivers of `except` and `overriding` lower the same way, as gaps of kind `capability-waiver`, so the transient waivers file and the gate's copy-into-acceptances step go away.

**The lint.** [inferred] `umpire-lint` matches findings against the IR's gaps by kind, owner and subject, as it matches the acceptance file today. A finding no gap covers fails the run as now. A gap no finding matches fails the run naming the annotation's source position. A gap whose `until` names a Flow spec that is done fails the run naming the position and the spec. The acceptance file is not read beside a feature IR; one present is an error that says to remove it. Under `generated/legacy/`, the file form is still read, since those payloads have no source, and it dies with the payloads under fn-160.

**The gate.** [inferred] One new source rule beside the sugar rule: the only annotation a Model under `model/temporal` may carry is the framework's `gap`, and a subject inside it is one of the framework's constructors. The gate's `acceptWaivers` step and the `--update` path that maintained the acceptance file are removed.

**Migration.** [inferred] The present acceptance files are the input. A mechanical rule reads a feature's IR and acceptance file, drops every acceptance that derivation inheritance makes redundant, groups the rest by owner and reason, and writes the `@gap` stacks onto the owning declarations, keeping each reason's text. The rule is proven on the mini feature first and runs on the real features in the sweep, under fn-166's policy that a DSL change ships its migration as code.

## API Contracts

**Framework.** [inferred]

```scala
// framework/Gaps.scala
final class gap(subjects: Subject*)(because: String, until: String = "") extends StaticAnnotation
sealed trait Subject
def silent(action: Action[?], in: AnyRef*): Subject
def unconstrained(classes: ClassRef*): Subject
def unperformed(actions: Action[?]*): Subject
def unevidenced(facts: AnyRef*): Subject
def uncovered(classes: ClassRef*): Subject
def unmapped(values: (AnyRef, AnyRef)*): Subject
def witnessOnly: Subject
// ... one per lint kind, as listed above
```

**IR.** [inferred] `Model.gaps: repeated Gap`, `Gap { kind, owner, subject, because, until, position }`; `Machine.derives: Derivation { from, kept, rebound, added, dropped_monitors, dropped_refinement }`. Both are metadata: the IR fingerprint, equivalence comparison and table pins ignore them.

**Lint.** [inferred] `umpire-lint` keeps its command line. Its failure summary counts unaccepted findings, stale gaps and expired gaps. The `Accepted` reader survives only for `generated/legacy/`.

**Until.** [inferred] `until` names a Flow spec id, optionally with a requirement, `fn-120` or `fn-120 R5`. The lint reads the spec's status from Flow's record; a name that matches no spec is an error at lift time.

## Edge Cases & Constraints

- **A subject appears in at most one `@gap` of one owner.** [inferred] The second is a lift error at its position, as the acceptance reader refuses a finding accepted twice today.
- **Every `@gap` names at least one subject and a non-empty reason.** [inferred] Lift error otherwise.
- **A subject that resolves to nothing** on its declaration (an action the machine does not bind, a class no action of it has, a fact it does not declare) is a lift error at the annotation. [inferred]
- **A gap on a derived machine for an inherited row** is a lift error that names the base to declare it on. [inferred]
- **An annotation on a declaration the lifter does not read** (a helper def, a test) is a gate error from the source rule. [inferred]
- **Current mode.** [user, paraphrase] Only the Scala DSL is open; every Go, tooling and regeneration change is deferred to the owner's sweep. The framework, the lifter and the IR schema are Scala-side work and go first; the lint's reading of gaps and derivation, the retirement of the file and the real features' migration are the sweep's. Until the sweep, the real features are *behind* on this change in fn-166's sense, and the acceptance files stay as they are.
- **Legacy payloads** keep their acceptance files untouched until fn-160 retires them. [paraphrase]
- **No lint kind changes meaning** and no finding is removed from the lint by this spec except by derivation inheritance; the counts before and after are recorded in the migration's evidence. [inferred]

## Acceptance Criteria

- **R1:** The framework declares a repeatable `gap` annotation, `@gap(subjects*)(because, until = "")`, and one inert subject constructor per lint kind, each taking the Model's typed references and never a string. Errors: a `@gap` with no subject or a blank reason fails the lift at its position; so does a subject that resolves to nothing on the annotated declaration. [user, inferred]
- **R2:** The lifter lowers every `@gap` to IR gap entries (kind, owner, subject in the lint's subject form, reason, until, position) on the owning machine, composition, realization, query, property or refinement, attributing a gap on a facet to the machine that mixes it in. Errors: a subject named twice for one owner fails the lift naming both positions. [inferred]
- **R3:** Gaps and derivation are inert in the IR: the fingerprint, the equivalence comparison and the table pins are byte-identical with and without them. Errors: no error surface beyond the existing equivalence checks. [inferred]
- **R4:** `except` and `overriding` lower to gaps of kind `capability-waiver` with their reasons; the transient waivers file and the gate's step that copies waivers into the acceptance file are removed. Errors: none beyond R1's. [inferred]
- **R5:** A lifted machine records what it derives from and what it changed (kept, rebound, added, dropped monitors, dropped refinement). The lint reports a finding on a derived owner only for what the owner changed and once on the base otherwise, and a gap on the base covers the inherited row on every derived owner. Errors: a gap on a derived machine for an inherited row fails the lift naming the base. [inferred]
- **R6:** `umpire-lint` matches findings against the IR's gaps; a feature whose generated tree holds an acceptance file fails with a message to remove it; a gap no finding matches fails naming its source position; an unaccepted finding fails as today. Under `generated/legacy/` the file form is still read. Errors: as stated. [inferred]
- **R7:** A gap whose `until` names a Flow spec recorded as done fails the lint naming the position and the spec; an `until` that names no spec fails the lift. Errors: as stated. [inferred]
- **R8:** The gate's source rules admit no annotation in a Model but `gap`, and no subject but the framework's constructors, reported like the sugar rule. Errors: as stated. [inferred]
- **R9:** A mechanical migration reads a feature's IR and acceptance file and writes the `@gap` stacks, one per owner and reason, keeping each reason's text and dropping what R5 makes redundant; it is proven on the mini feature and leaves that feature's one hole and one waiver declared with `@gap`. Errors: an acceptance it cannot place is reported, not dropped. [inferred]
- **R10:** After the sweep, no feature with sources carries an acceptance file, and the migration's evidence records each feature's finding counts and gap counts before and after, with the standalone activity's 179 entries and 5,089 pairs as the before. Errors: no error surface beyond R6. [paraphrase]

## Boundaries

- No lint kind is added, removed or changed in what it judges. [inferred]
- No arm-level sugar: a silent rejection is declared from the machine with `silent(action, in = ...)`, or decided as a row with the existing `rejects(why).because(...)`. One mechanism. [inferred]
- The legacy payloads' acceptance files are not migrated; fn-160 retires them with the payloads. [paraphrase]
- The Go lint, the file's retirement and the real features' migration wait for the owner's sweep; this spec does not reopen the Go side. [user, paraphrase]
- Reasons are not rewritten for content during migration; stale wording is a later cleanup, not this spec's. [inferred]
- No `@gap` on a whole feature or package: a gap names a declaration. [inferred]

## Decision Context

**Why not the file.** It is the lint's word in the lint's place: a string identity that goes stale on a rename, extended by hand on every regeneration, and blind to derivation, so a six-thousand-line file holds about 120 decisions. A reason in the Model cannot outlive the thing it explains.

**Why an annotation and not sugar.** An annotation is inert by construction, which is what an acceptance must be; it needs no core form; and the lifter already reads trees off TASTy for `except`, so reading `@gap` is the same match. Its one limit, that it sits on declarations and not on arms, is met by naming the pair from the machine, which also keeps the surface to one mechanism.

**Why `gap`.** `accepted` is the lint's passive word and collides with `Outcome.accepted`. `ignored` says the finding was looked away from, means "skipped" in munit, and is the right word for an empty effect on an arm, where it should stay free. `hole` names only H1 to H5; the coverage kinds are gaps in evidence, not in specification. `waived` releases a requirement, and a silent rejection is a decision deferred, not released. `leaves` reads well and was passed over for a noun. `gap` is the word the existing reasons already use for themselves and fits every kind (owner, 2026-10-11).

**Why one annotation per reason.** It is the acceptance file's own unit, moved next to its subjects; a stack of short annotations keeps each reason beside what it explains.

**Why `until`.** The residue that would otherwise still need a file is the time-bound kind, "kept until fn-160 separates dispatch." Naming the spec turns a sentence nobody rereads into a check that fires when the spec closes, and removes the last reason to keep a file.

**Why derivation first.** Nine tenths of the file is inherited rows repeated per derived owner. Making the lint see derivation removes them without a single annotation and is the only change that needs no author words.

**The two linters.** The Scala side lints the source's form and the Go side lints the IR's semantics; the gate is the seam. Gaps straddled that line, authored in Scala terms, judged in Go, half-written by the Scala gate. This spec puts authoring wholly on the Scala side and judging wholly on the Go side.

## Requirement coverage

| R-ID | Covered by | Notes |
| --- | --- | --- |
| R1 | fn-167.1 (TBD - populate via /flow-next:plan) | framework |
| R2 | fn-167.2 (TBD - populate via /flow-next:plan) | lifter |
| R3 | fn-167.2 (TBD - populate via /flow-next:plan) | lifter, IR schema |
| R4 | fn-167.2 (TBD - populate via /flow-next:plan) | lifter, gate |
| R5 | fn-167.3 (TBD - populate via /flow-next:plan) | lifter (Scala), lint (sweep) |
| R6 | fn-167.4 (TBD - populate via /flow-next:plan) | lint, sweep |
| R7 | fn-167.4 (TBD - populate via /flow-next:plan) | lint, sweep |
| R8 | fn-167.1 (TBD - populate via /flow-next:plan) | gate source rule |
| R9 | fn-167.5 (TBD - populate via /flow-next:plan) | migration rule, mini feature |
| R10 | fn-167.5 (TBD - populate via /flow-next:plan) | sweep |
