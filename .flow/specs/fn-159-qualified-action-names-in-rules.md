# Qualified action names in rules

## Execution status (2026-10-21 backlog audit)

Deferred by the owner. The Rust decision has been made and the active workspace is `model/`; the former Scala tree is frozen under `model/archived/` and is not expected to build (AGENTS.md and MILESTONES.md). The earlier “until the Rust verdict” note is historical, not an automatic release of this spec. Existing task states and old SHIP reviews describe historical checkpoints, not current implementation readiness.

Resume only after an explicit owner decision and a fresh scope/dependency review against the Rust architecture. Scala paths and commands below describe the archived layout; do not run them as current gates or mechanically port the plan. Preserve unfinished requirements and completion evidence; successor Rust work does not discharge this spec without an explicit requirement mapping. Taskless proposals remain deferred captures, not executable plans.


DSL changes ship a task-named Scalafix rule under `model/migrate/rules/`, prove it on copied `activity_mini` sources before the real-feature sweep, and follow `model/migrate/README.md`; production loading remains held in fn-166.10.

## Conversation Evidence

> user (turn 3): "on(poll) {"
> user (turn 3): "would it be better if we enforce qualified names? ie worker.poll? it's wrapped in from(worker) {} but it could be easy to miss"
> user (turn 4, selected): "yes option 1"
> user (turn 4): "will it be enforced that eveery `on` is using a qualified form?"
> user (turn 4): "let's open a new flow next spec; and this will be its first task. we'll ad more as we go."
> user (turn 7): "yes, add task to rename to completionClient"
> user (turn 9): "same as with the `on(<actor>.<action>` rule; can we change and enforce `when(scheduled)` to be `when(Phase.scheduled)` ?"
> user (turn 9, selected): "Every phase reference"
> user (turn 12): "add Removing disabled(process.stop) as a task to the spec"
> user (turn 13): "yes, add task to move all bounds to Bounds package"
> user (turn 14): "C. Rename the standalone object. yes combine."
> user (turn 14): "Drop read."
> user (turn 22): "I like only  1. Remove the trivial effects"
> user (turn 27): "yes make that a task"

## Goal & Context
<!-- scope: business -->
<!-- Goal & Context: 60% [user], 40% [paraphrase] -->

A machine's `rules` section names the action each `on` block fires. Inside `from(worker) { import worker.*; ... }` that name is bare (`on(poll)`), and which object declares it depends on a `from` header and a wildcard import that may sit dozens of lines above. Reading the block, a reviewer can miss which actor's action it is. A diff hunk usually doesn't show the header. A grep for `worker.poll` misses the rule. And one bare name can mean different things in the same file: in the standalone activity's product machine, `heartbeat` is a start input, a worker action and a deadline.

The framework and the IR generator already refuse a `from(X)` block whose `on` names an action `X` does not declare, so correctness is not the problem. Readability is. The goal is that every `on` names its action through its declarer, so each rule line can be read on its own.

More tasks will be added to this spec as related work turns up.

## Architecture & Data Models
<!-- scope: technical -->

A Model's `rules` section is written as `on(<action or class>) { cases }` blocks, which can be grouped as `from(<declarer>) { ... }`. Two layers read the rules, and both have to enforce the same contract:

- **The framework's rule builder**, which runs when a machine object initializes. It gets the source text of each `on` target and already checks that a target inside a `from` belongs to that declarer.
- **The IR generator's lifter**, which reads the typed tree of the rules section. It already requires `import <declarer>.*` as the first statement of a `from` and checks the same ownership.

After the change, `from(X)` is only a grouping header with an ownership check. It no longer brings `X`'s members into scope.

## API Contracts
<!-- scope: technical -->

The rules surface after the change:

- Accepted: `from(worker) { on(worker.poll) { ... } }`, `on(worker.respondFailed(Failure.fatal)) { ... }`, `on(client.pause, client.unpause) { ... }` and `on(deadline.heartbeat) { ... }` outside any `from`.
- Refused: `from(worker) { import worker.*; on(poll) { ... } }`, and a bare `on(poll)` anywhere.

## Edge Cases & Constraints
<!-- scope: technical -->

- An `on` target can be a whole action (`on(worker.poll)`), a class of an action (`on(worker.respondFailed(Failure.fatal))`), or two to four of either (`on(client.pause, client.unpause)`). The qualification rule applies to the action reference in each of them. The arguments of a class form are not action references.
- A bare name can reach an `on` without any `from`, through an ordinary import. The rule applies there too.
- The change is syntactic. The IR the generator emits for every migrated Model must stay identical.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** Every `on` target names its action through the object that declares it, whether or not it sits in a `from` block, and in the whole-action, class and multi-target forms alike. [paraphrase] Errors: a bare action name makes the IR generator fail, naming the action and its position and spelling out the qualified form to write; a class form's arguments (`Failure.fatal`) are not checked as action references.
- **R2:** The framework's rule builder refuses an unqualified `on` target when the machine initializes, so a Model fails before the IR generator runs. [inferred] Errors: the same refusal and message shape as R1.
- **R3:** `from(X)` stays a grouping header. It still refuses an `on` whose action `X` does not declare, in both the framework and the IR generator. [paraphrase] Errors: no error surface beyond the existing ownership refusal.
- **R4:** A `from(X)` block that imports `X`'s members (`import X.*`) is refused, in both the framework where it can detect it and the IR generator. [paraphrase] Errors: the refusal names the `from` and says to qualify its actions instead.
- **R5:** Every `from` block and `on` target in the temporal Models is migrated to the qualified form, and the checked-in IR files regenerate identically. [inferred] Errors: no error surface beyond R1–R4.
- **R6:** The README's rules spec and the framework's doc comments describe only the qualified form. [inferred] Errors: none (documentation).
- **R7:** IR generator tests cover a bare name inside a `from`, a bare name outside one (through an import), an import inside a `from`, and the accepted qualified forms (whole action, class, multi-target). [inferred] Errors: none (tests).
- **R8:** The standalone activity's By-ID actor is named `completionClient`, not `service`, and is documented as a client that settles an activity asynchronously by activity id, without a task token and independent of the worker process. [user] ("yes, add task to rename to completionClient") Errors: no error surface; unlike R5, this changes the actor name and qualified action ids in the activity-standalone IR files, and nothing else.
- **R9:** Every reference to a machine's phase is qualified by its phase type (`when(Phase.scheduled)`, `phase = Phase.started`), in rule cases, effects, init, states and properties. The IR generator refuses any import of a phase type's members, and a bare phase argument to the phase form of `when(...)`. [user] ("can we change and enforce `when(scheduled)` to be `when(Phase.scheduled)` ?"; selected: "Every phase reference") Errors: an import of phase members, wildcard or selective, at any level of a lifted source, fails the lift, naming the import and its position; a bare `when` argument fails likewise; `when(states.set)` and `when[Role]` are unaffected; there is no runtime check, because the phase form of `when` takes varargs that are not inline.
- **R10:** The standalone activity's product machine does not bind the worker process's stop, which it never enabled; the System keeps its stutter row for it. [user] ("add Removing disabled(process.stop) as a task to the spec") Errors: if the IR generator, lint or refinement requires the binding, it stays, with a comment giving the reason, and this criterion is met by recording that finding.
- **R11:** Every query bound (`Limits`) is declared once, in the shared `Bounds` object, and no two bounds share a name with different budgets. [user] ("yes, add task to move all bounds to Bounds package") Errors: no error surface; renamed bounds change the bound names recorded in Query receipts, and nothing else.
- **R12:** The standalone activity's bound worker object is named `activityWorker`, so its rules read `on(activityWorker.poll)` and its signature reaches the activity kind's worker actions without the fully qualified path. [user] (selected: "C. Rename the standalone object. yes combine.") Errors: no error surface; the IR does not change, because action ids come from the kind's declaration.
- **R13:** `Observation` carries no free-text `read` field; the response field an observation stands for is named in a comment beside the realization's typed read. [user] (selected: "Drop read.") Errors: if anything downstream turns out to read the text, the field stays, and the finding is reported instead.
- **R14:** Rules and `choose` alternatives enter a phase through a framework step, `enters(Phase.X)`, and no model declares an effect whose only work is setting the phase, so `effects` holds only steps with logic. [user] ("I like only  1. Remove the trivial effects") Errors: an effect that records another fact, rejects or chooses stays; the IR keeps every row, outcome, fact and Case, and only function names and hashes may change.
- **R15:** Every capability kind is named as an adjective for what an entity can be: `Retries` becomes `Retryable` and `Deadline` becomes `Expirable` (names to be confirmed), keeping capabilities distinct from feature-local facets, which are named as nouns. [user] ("yes make that a task", on the Retries/Deadline renames) Errors: no error surface; only the capability Property qualified names in the IR change, and other "deadline" names stay.
- **R16:** A bounded integer has one spelling: `UpTo[N]` on the field, or as a channel's message type. `Finite.upTo` ranges only a named opaque type, and the IR generator reads every Int range from the field's or message's own type, never from a per-state `given Finite[Int]`. [inferred] (from `model/framework/SEMANTICS_FEEDBACK.md` 1.3, 2026-10-11) Errors: a plain `Int` state field, and an `Int` channel message, are refused at their declaration with a message that names `UpTo[N]`.
- **R17:** An input that a class may omit declares its default on its token (`val startToClose = input[Timeout](default = Timeout.unset)`); `start()` is every input at its declared default, and a named call (`start(maxAttempts := MaxAttempts.two)`) leaves only inputs with a declared default unsupplied. No form reads a domain's first value. [user] ("Make the default explicit ... I like that, make that a flow task"; "`start()` means all defaults", 2026-10-11) Errors: an empty or named call that omits an input with no default is refused by the framework when the machine, Scenario or realization that holds it initializes, and by the IR generator at the call, naming the input.

## Boundaries
<!-- scope: business -->

- Removing `from` blocks entirely is not part of this spec. [paraphrase]
- Accessor generation for `effect { }` blocks (`phase` / `phase_=`) is not part of this spec; it was discussed and deferred until more levels use the sugar. [inferred]

## Decision Context
<!-- scope: both — conditionally substructured -->

Three options were weighed, and the user chose option 1 ("yes option 1"). [user]

- **Option 1 (chosen):** keep `from(X)` as a grouping header, require qualified names inside it, and drop the `import X.*`. This keeps reading a machine's rules by actor and keeps the ownership check, while every rule line says whose action it fires. The cost is a repeated prefix. [paraphrase]
- **Option 2 (rejected):** drop `from` and always qualify. One form and less framework and generator code, but you lose the grouping by actor and its ownership check. [paraphrase]
- **Option 3 (rejected):** leave it as is. Defensible, since ownership is already enforced, but it leaves the readability problem the user raised. [paraphrase]

2026-10-18, deferred (owner): all specs other than fn-210 (the Rust prototype of the framework in `temporal/model2`) are deferred until fn-210's write-up recommends whether to extend the Rust prototype or stay on Scala. Flow has no deferred status; this line is the record. MILESTONES.md was reset to the prototype alone and no longer lists this spec. Nothing in this spec changes; resume or close it after that decision.

## Parked unknowns

- Whether `disabled(...)` targets (such as `disabled(process.stop)`) fall under the same qualification rule. Today they are written qualified, but nothing enforces it. A decision from the user resolves this. [inferred]

## Scope: the mini feature only (owner, 2026-10-11)

Within the Scala DSL batch (`MILESTONES.md`), this spec changes only the framework, the lifter, the shared code and `features/activity_mini`. The real features (standalone activity, Nexus workflow, Nexus standalone and their legacy payloads) are not edited, migrated, lifted or regenerated, and they stay behind until the owner's sweep, which runs this spec's scalafix rules on them. Equivalence compares `FEATURES=activity_mini` against the mini baseline. This overrides any statement in this spec that edits, migrates or proves on a real feature. See fn-166's Scope. The source changes already integrated into the real features stay. Open proofs and closures run on the mini feature; the real features' lift and equivalence proofs wait for the sweep.
