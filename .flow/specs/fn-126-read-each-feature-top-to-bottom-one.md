# Read each feature top to bottom: one object per machine

## Goal

A feature developer who wants to know what a machine does, what it promises and what is asked of it should read one file from the top down, the way a Quint module reads. The file holds the types, then the actors and the actions they take, then one object per machine, then the IR files. Each machine object is the machine itself: its start and end, its effects, the rules that say when each action fires, its Properties, its laws and its Queries, each in a section of its own. `Realization.scala` stays beside the feature file.

**Guiding principle (owner, 2026-10-05): lighten the author's cognitive load.** An author writes what the Model means and nothing the tooling can infer: identities come from where code lives (decision 23), roles from the section a member sits in, types from `extends Machine[S, O, F]` (decision 18). Structure is enforced by the gate and shown by a template (R20), not remembered. When two designs are otherwise equal, prefer the one with fewer things an author must declare, name or keep in sync.

Owner decisions (2026-10-05):

- **Layout.** Adopt option (a) of the Quint-module study, `.plans/QUINT_MODULE_LAYOUT.md`: one feature file per folder, read in a fixed order, with a lint that keeps that order safe, because Scala initializes an object's vals in the order they are written.
- **Folded in from the DSL-simplification study** (`.plans/DSL_SIMPLIFICATION.md`), so files and names change once:
  - actions grouped by actor (rank 3);
  - single-use Scenarios inlined and one shared `Limits` source (rank 4);
  - the package renames `admission/` → `record/` and `compositions/` → `withTaskQueue/`;
  - one closing batch of renames (rank 5).
- **Later decisions, all approved.** They replace the study's "layout only" scope:
  - the machine object is the machine;
  - rules say when an action fires, and effects say what it does;
  - every kind of member sits in a section object;
  - the two levels of a feature are named Product and System.

What the Models mean does not change until the closing rename batch (R18). That batch changes names only, and accepts the new Definition IDs with one golden re-capture.

Paths are the ones fn-114.9 gives: `model/irgen`, `model/check`, `model/temporal/features/*`, `model/temporal/shared/*`.

## Why

The study measured the standalone activity after fn-112, fn-114 and fn-122:

- **Many files.** The feature has 15 files, 1,762 lines, 141 top-level declarations and 3 `DefinitionScope` pins. The Nexus caller repeats the pattern in 9 files and 2,450 lines.
- **Many hops.** Following `activityProduct` from its state type to a Query takes 4 files and about 675 lines of context. A typical Query is 5 files away from its state, and a Case 9 to 11.
- **References no import shows.** Files refer to each other through same-package top-level vals, so no `import` says which file a name comes from. `Properties.scala` uses about 42 symbols from `Model.scala`, `Queries.scala` about 67, and `Realization.scala` about 70.
- **A step far from its claim.** The step `attemptResult(completed)` is in `Model.scala:270-284` and bound at `:385`. Its Property is in `Properties.scala:12-14`, its Scenario in `Queries.scala:25-26` and its Query at `:101-102`.
- **When and what are mixed.** 45 of the Models' 74 `disabled` are inverted guards (`if !g then disabled else …`). Step functions mix when an action may fire with what it does, so to see which phases enable `control` a reader has to work through a 30-line match (`standaloneactivity/Model.scala:293-322`).
- **The actor is invisible where it matters.** A Scenario reads `actions(start(), attemptStart, attemptResult(completed))`. Which party takes each step is stated only in the action's declaration.

Some of the file splits were chosen on purpose and some only happened.

- **On purpose.** fn-112 split files by kind: monitors stay in `Model.scala`, because two files would initialize each other in a cycle, and Properties form "the specification a reviewer reads on its own". fn-112 also made the subject folders and the per-machine vocabulary objects.
- **Only happened.** `Capabilities.scala` (fn-122.8) and `IrFiles.scala` (fn-114.1) each have a declaration placed on purpose, but nobody argued for a separate file. The per-kind file names themselves were never argued either.

Inside one file, the init-cycle reason goes away, provided the declaration order is checked. "Read on its own" survives as the `properties` section. The lifter already lifts object members (`Control.forgedCompletion`, `Functional.completion`).

After the change, following the product machine from its state to its Query means reading one object of about 120 lines. A step's rule, its effect and its Property sit in the same object, and a Scenario reads `actions(caller.start(), worker.poll, worker.respond(completed))`.

The cost:

- two regenerations of the IR and Cases with only recorded deltas (the layout tasks and the machine-object tasks);
- one reshaping of the lifter's machine declaration;
- one golden re-capture for the rename batch.

## Requirements

### Layout (tasks 1 and 2)

- **R1 One feature file per folder.** Each Model folder holds one feature file named after the folder, its `Realization.scala` where it has one, and its tests. Altogether, 30 per-kind files become 8 feature files.

  | Folder | Feature file | Machine and composition objects (names before R18 → after) |
  | --- | --- | --- |
  | `features/standaloneactivity` | `StandaloneActivity.scala` | `ActivityProduct`; `ActivityProtocol` → `ActivitySystem`; `ActivityWorker`; `StandaloneActivity` |
  | `…/standaloneactivity/record` (was `admission/`) | `Record.scala` | `CurrentAdmission` → `RecheckingRecord`; `StaleAdmission` → `TrustingRecord`; `HeldAdmission` → `HeldDispatchRecord`; `AdmissionResponseLoss` → `LostStartAnswer` |
  | `…/standaloneactivity/withTaskQueue` (was `compositions/`) | `WithTaskQueue.scala` | `CurrentRecord` → `RecheckingMember`, `StaleRecord` → `TrustingMember`, and the compositions over the queue (`CurrentOverQueue` → `RecheckingOverQueue`, …) |
  | `features/nexuscaller` | `NexusCaller.scala` | `NexusProduct`; `NexusProtocol` → `NexusSystem`; the control Model's objects |
  | `…/nexuscaller/closepolicy` | `ClosePolicy.scala` | `RejectAfterClose` and its nine derived designs |
  | `features/nexusoperation` | `NexusOperation.scala` | `NexusOperation` |
  | `shared/taskqueue` | `TaskQueue.scala` | the opaque contract (`DispatchQueue` and its storage-loss variant); the matching provider (`MatchingQueue`, lossy, forgetful, volatile) |
  | `shared/worker` | `Worker.scala` | `Polling` |

  The layout tasks (1 and 2) group each machine's declarations into one module object, using today's declaration forms. Task 4 turns each module object into the machine object of R15.

- **R2 Reading order.** A feature file has the following sections, in this order:
  1. A header comment that names the objects in order, then the imports, the file-level pin and the family.
  2. **Types.** Every enum, state case class and type alias, at the top level.
  3. **Signature.** Entities and inputs, the actor objects with their actions (R14), observations, choices, and the top-level `given`s.
  4. **Machine and composition objects**, in dependency order: a refined machine before the machine that refines it, a base machine before the machines derived from it, and the members of a composition before the composition.
  5. **`object Files`.** The feature's `irFile` roots.

  Inside a machine object, the order is:
  1. the header members (`start`, `end`, and where declared `entity`, `refines`, `visible`, `unobservable`, `evidence`), then the vocabulary (status sets and constants);
  2. `effects`;
  3. `monitors` (monitors, assumptions, holes and channels);
  4. `rules`;
  5. `properties`;
  6. `laws`;
  7. `queries` (Scenarios, then Queries).

  Errors: R4 refuses a section out of order.

- **R3 Placement.** Each declaration has one place:
  - an effect is a member of its machine's `effects`;
  - a Property, a capabilities declaration or a Scenario is a member of the `properties`, `laws` or `queries` section of the machine or composition it is over;
  - a Query lives with its Scenario;
  - a declaration over two machines that is not itself a composition does not exist today; if one is written, it goes in the `queries` section of the composition that holds both;
  - `irFile` roots go in `object Files`.

  Capabilities that read the realization (today's `protocolCapabilities`, which reads `ActivityRealization.activityStatus`) go in their machine's `laws` section. A section object initializes lazily, on first use, so the realization may read the machine object, but it must not read a `laws` section; R4 (b) refuses that cycle. Errors: R4 refuses a misplaced declaration and names the section it belongs in.

- **R4 Declaration-order lint in the model gate.** `make umpire-check-model` (`model/check`) runs a lint over every Model. It refuses each of the following at `<file>:<line>`:
  - (a) a val that is read during initialization before it is declared in the same owner;
  - (b) an initialization cycle between the owners of one Model package tree: package objects, machine, composition and section objects, actor objects, `Files` and realization objects;
  - (c) a member out of R2's order;
  - (d) a declaration that breaks R3;
  - (e) from task 4 on, R17's section rules.

  Reads inside a `def`, a lambda, a by-name argument or a lazy val do not count as initialization reads. Task 1 decides whether (a) and (b) come from Scala's own checkers (`-Wsafe-init`, `-Ysafe-init-global` in `model/project.scala`, with `-Werror`) or from a pass in `model/irgen`, and records the decision. Errors: each kind has one refusal fixture. The lint passes on all Models and refuses nothing in the lifter's passing fixtures.

- **R5 Meaning and identity frozen until R18.** Until the rename batch, the following stay exactly as they are:
  - Definition IDs and IR type names;
  - the names of machines, compositions, Properties, Scenarios, Queries, Limits and law instances (`<machine>.<law>`);
  - transition tables, refinement rows and table fingerprints;
  - Query answers, lint findings and coverage;
  - Quint/P agreement;
  - every Contract and Case byte, apart from the deltas below.

  Only these deltas are allowed, and each is recorded:
  - source file paths and lines, in the IR and in Cases;
  - the IR `source` root strings;
  - the symbols of moved functions (`Model$package$`/`Product$` → `StandaloneActivity$package$`/`ActivityProduct$effects$`), including functions nested in section objects;
  - from task 4 on: the derived names of rule-lowered step functions (R16), and lifted function bodies where rules replace a step function, provided the transition table each one gives is unchanged;
  - the string-literal count rises only by the Scenario names that R13 keeps.

  They are recorded in `tools/umpire/internal/golden/config.json` (`function_name_substitutions`, `source_root_moves`, path entries) and, where the projection cannot express a restructured body, in a re-captured `original.json` that only the task introducing the difference extends. Either way, a before/after projection of the reader's tables shows them identical. Errors: any other difference stops the task.

- **R6 Pins.** Every machine, composition or realization object that holds an ID-bearing declaration pins its former owner once, through the existing `DefinitionScope`. The ID-bearing kinds are actions, monitors, assumptions, holes, channels and realizations (`irgen/Context.scala:239-245`). Section objects and actor objects pin nothing: they are transparent (R14). Types stay at the top level under the file-level pin, which keeps their `pkg.Type` IR names. No per-declaration ID string is added, and no pin string beyond today's former owners. Two owners may pin one former owner as long as their names stay distinct; `pinOf` and `idTakenBy` allow this today, and task 3 adds the fixture. Errors: the lifter refuses a duplicate ID, as it does today.

- **R7 Names without literals.** A machine, composition or actor object is named by its object's name with the first letter lowered, so `object ActivityProduct` is `activityProduct` and `object caller` is `caller`. This is the "names come from declarations" rule fn-114 settled, applied to objects. Until R18, every machine keeps today's name, so derived Query names (`<machine>.<scenario>.<property>`) and law names stay the same. No `machine("…")` string is added.

- **R8 What stays.** `Realization.scala` stays a sibling file. Its changes are imports, qualified references, and the new action and machine names. The subject folders stay subpackages under their new names (R12). `model/temporal/{capabilities,realize}` and the IR schema are untouched. `model/umpire` changes only as R14 to R16 state.

- **R9 Docs.** The following documents describe the final layout and declaration shape and name no removed file or form:
  - `model/README.md`: the layout section under "Writing a Model", with the activity as its example and R2's order; "Where things are"; the sentence on capabilities that says "its own `Capabilities.scala`"; and the line-referenced examples such as `nexuscaller/Properties.scala` line 22;
  - `model/SEMANTICS.md`;
  - `.plans/UMPIRE_MODULES.md`: the Models row, the Nexus operation row and line 321;
  - `.plans/UMPIRE4_VISION.md`;
  - `AGENTS.md`;
  - `.plans/DSL_OPERATORS.md`: its rejected guard helper (record the owner's reversal: `when` and `in` are rule headings, never a guard inside a step);
  - `.plans/DSL_SIMPLIFICATION.md` and `.plans/QUINT_MODULE_LAYOUT.md`: mark what landed.

  Errors: R10 finds a stale path.

- **R10 Old names stay retired.** fn-114.9's layout test (`tools/umpire/model/layout_test.go`, or wherever fn-124.8 moves it) fails in three cases:
  - a file named `Model.scala`, `Properties.scala`, `Queries.scala`, `Capabilities.scala` or `IrFiles.scala` exists under `model/temporal/features` or `model/temporal/shared`;
  - a live file names such a path. This includes Go tests that assert a Scala position, such as `lower/activity_test.go:742`;
  - from R18 on, a live file names a retired machine, action or folder name from R18's table. Archives and `.flow/` are exempt.

  The kit's `model/temporal/capabilities/Capabilities.scala` is not matched. Errors: a test case proves that each of the five file names is detected.

- **R11 Evidence.** The closing done summary states, for each folder:
  - files and lines before task 1 and after task 6 (`umpire.check.metrics`);
  - for `activityProduct` and `nexusProduct`, the files and lines read from the state type to a Query;
  - the count of cross-file same-package references;
  - the count of `disabled` and of inverted guards.

  It also lists every recorded delta from R5 and every rename from R18.

### Names and grouping (tasks 1 to 3)

- **R12 Package renames.** `standaloneactivity/admission` becomes `standaloneactivity/record`, and `standaloneactivity/compositions` becomes `standaloneactivity/withTaskQueue`, as folders and packages. IDs and type names stay, through the existing `…System$package$` pins. Function names and paths change and are recorded as R5 deltas. Errors: any ID or type-name change stops the task.

- **R13 Shared bounds and inlined Scenarios.**
  - **Bounds.** One `Limits` source, `model/temporal/shared/Bounds.scala`, declares every bound two folders share (`three` is declared identically three times today). Each Query's Limits keep their name, so the IR changes only in positions.
  - **Inlining.** A Scenario that exactly one Query uses is written inside that Query and keeps its name string (`scenario("completed").actions(…)`), so its IR is identical. This applies to 8 of the standalone activity's 11 paths, to `nexusoperation` and inside the close policy's `designQueries`.
  - A Scenario used by two or more Queries stays a named val in `queries`, before its first Query.

  Errors: a Query whose Scenario name or Limits name changed stops the task.

- **R14 Actions grouped by actor, IDs kept by transparent sections.**
  - **Actor objects.** The signature declares each feature's actions in objects named after who takes them, so every call site shows the actor. Parties become actor objects: `object caller extends Actor` is the party `caller`, and its members are the actions it takes. Groups that are not parties extend `Section`: `timers`, `deadline`, `history` (internal steps), `queue`, `faults`.
  - **Transparency.** `umpire.Section` and `umpire.Actor` (an `Actor` is a party and a `Section`) are framework markers. A section object is transparent to Definition IDs: each member takes the ID it would have as a direct member of the section's enclosing owner. At the top level of a file, that owner is the file's package object, so the file's pin applies. One lifter rule implements this, with a fixture.
  - **Where sections may sit.** A section may sit at the top level of a Model file or directly in a machine, composition or derived-machine object. Errors: a section inside a section, a section anywhere else, and two members of one pinned owner that would share an ID are refused at their line.
  - **Moves keep names.** Actions keep their val names here, so their IDs stay: `worker.attemptStart` until R18 renames it to `worker.poll`.
  - **Inputs.** The deadline inputs move out of `object Inputs` to the top level once the deadline timers live in `deadline`, because their names no longer collide. `Inputs` keeps only an input named like an action of the actor object that takes it (`Inputs.control`), since inside `object caller` the name `control` is the action.
  - **Two objects for one party.** The shared worker Model declares `object worker extends Actor` (party `worker`, today `Party("worker")`) with `workerStop`, `workerResume` and `serve`. The standalone activity's own worker actions (poll, respond) are performed by that party, so they live in a feature `object worker extends Section`. The feature imports the shared party under another name: `import shared.worker.{worker as process}`, then `process.stop`. This is the one place where two objects name one party.

### Machine objects, effects, rules and sections (tasks 4 and 5)

- **R15 The machine object is the machine.** `val m = machine[S, O, F] { forEntity; starts; ends; steps }` is replaced by `object M extends Machine[S, O, F]`, whose members are:
  - `val start` (the initial state, or states) and `def end(s)`;
  - where needed, `entity`, `refines`, `visible`, `unobservable` and `evidence`;
  - the vocabulary;
  - the sections `effects`, `monitors`, `rules`, `properties`, `laws` and `queries`.

  Each section is an object extending `Section`; `rules` extends `Rules`, which is one. Inside a machine object, Properties and Scenarios name their machine implicitly through inherited members: `property when c holds`, `property.once(…)`, `scenario.actions(…)`, and `capabilities(limits = …)(…)` in `laws`. Not adding:
  - bare `once`/`never` words, because `when` would then collide with the rule heading;
  - a `path` synonym for `scenario`.

  **Derived machines** are objects too: `object TrustingRecord extends Derived(RecheckingRecord.rebind(…))`.
  - The derivation stays an expression of today's core operations (`rebind`, `extend`, `restrict`, `refining`, `assuming`, `unmonitored`, `withMember`), passed to `Derived`.
  - The object adds only the sections that are its own (`properties`, `laws`, `queries`).
  - It reuses the base's effects as `RecheckingRecord.effects.x`.
  - Every derived machine is an object, including the one-line composition members (`object RecheckingMember extends Derived(RecheckingRecord.unmonitored)`), so there is one form.

  **Compositions** are `object RecheckingOverQueue extends Composition[OverQueue](_.activity -> RecheckingMember, _.queue -> DispatchQueue)`. Their members are:
  - `def end(s)`;
  - `object syncs extends Syncs`, whose statements are today's `sync(…)` calls, named syncs included;
  - their own `properties`, `laws` and `queries`.

  `withMember` derivations are composition objects, `object TrustingOverQueue extends Composition(RecheckingOverQueue.withMember(_.activity -> TrustingMember))` (amended in task 4: one class cannot extend both `Machine` and `Composition`, and helpers over compositions need `synced` and `own`).

  **The lifter is reshaped once.** It reads an object extending `Machine`, `Derived` or `Composition` together with its member sections. The `machine[S, O, F] { … }` builder, `steps(…)`, `starts`/`ends` and `compose(…)` value forms are retired from `model/umpire` and the lifter once every Model and lifter fixture uses the object forms, so the lifter supports one declaration shape. IR files keep explicit roots: a root names a machine, composition, Query, `laws` declaration or realization as today. A machine root does not pull in its sections, because that would change what each IR file holds; `competingTimers` sits in `activity-system.json`, not in `activity.json`.

  Errors, each with a refusal fixture: a machine object without `start` or `end`; a section outside a machine, composition or file top level; a nested section; an effect outside `effects`; a machine object without `rules`; two objects whose lowered names collide. The runtime wiring needs no reflection; task 4 picks it and proves it with munit.

- **R16 Rules say when, effects say what.**
  - **Effects** are plain defs in `effects` with no guard: `def startAttempt(s) = enter(s.copy(phase = started), statusStarted)`. An effect may branch on its input and on state to choose what happens. It never returns `disabled` or `Nil`. Where the branch is really "which phase are we in", the style is to write separate rules.
  - **Rules** list firing conditions in `object rules extends Rules`:
    - `when(g) { action ~> effects.x }` fires while `g` holds;
    - in a phase-driven machine, `object rules extends Rules(_.phase)` declares the phase projection, and `in(p1, p2) { … }` fires in those phases;
    - a rule may name a whole action (`caller.control ~> effects.notFound`) or one class of it (`worker.respond(AttemptResult.completed) ~> effects.complete`);
    - `disabled(action)` binds an action that no state enables. It replaces today's all-`disabled` step functions such as `Product.workerStop`.
  - **Disjointness.** Rules for one action class must have disjoint guards. Intended nondeterminism is an explicit `choose` with named alternatives, as today. The check runs when the machine is constructed. It evaluates every pair of rules of one action class over every state of the `Finite` state type (and every input value), and refuses an overlap naming the machine, the action class, both rules by heading and position in order, and a state where both hold. The model gate initializes every IR-file root, so an overlap fails the gate.
  - **Lowering.** In order, the rules of one action lower to one step function, `(s, i) => if g1 then e1 else if g2 then e2 else Nil`, bound with the existing `action ~> step` core. Its derived name is `<machine object>.rules.<action>`, recorded as a golden function-name substitution. The transition table is unchanged.
  - **Sugar.** Rules are sugar in `umpire/Syntax.scala`. Each heading has a `Core form:` doc naming that lowering, and paired fixtures (rules / hand-written core step) lift to identical tables. `SyntaxRule` learns the new names.
  - **Derived machines.** In `rebind`, `action ~> effects.y` keeps that action's rule guards and replaces the effect; it is refused when the action has more than one rule with different effects. A rule argument such as `rebind(when(g) { action ~> e })` replaces that action's rules. `extend` takes rules only.
  - **`when` has one design.** It is a rule heading and nothing else: effects have no inner `when`, and the study's standalone guard sugar (rank 2) is not added in fn-127.
  - **fn-112.6.** That decision (no wildcard arm; each disabled phase explicit with its reason) is honored as follows:
    - every rule states its phases or condition positively;
    - there is no catch-all or `otherwise` rule;
    - an action bound but never enabled is written `disabled(action)`;
    - the pairs the server rejects (the seven pause/unpause pairs) keep their reasons in the accepted `silent-rejection` findings of `model/ir/*.lint.json`, where fn-120.3 already records them; the lint reports `never-enabled` and `silent-rejection` pairs as today.
  - Errors: an overlap; `disabled`/`Nil` in an effect; `in` without a declared phase projection; a rule naming an action class the machine does not bind; a bare binding in `extend`.

- **R17 Section lint.** R4's lint also refuses:
  - a member of the wrong section;
  - sections out of R2's order;
  - an effect outside `effects`;
  - a step function written outside `rules`, that is, a hand-written `action ~> step` in a Model. Core spellings stay legal in the lifter's core fixtures.

  Errors: one refusal fixture per kind.

### Renames and terminology (task 6)

- **R18 One batch of renames.** One task renames the following, accepts the new Definition IDs, re-captures the golden baseline once and regenerates once. Moves done earlier keep their IDs; only these renames change them.

  | Kind | Before → after |
  | --- | --- |
  | Levels (machines) | `activityProtocol` → `activitySystem`, `nexusProtocol` → `nexusSystem` (objects `ActivityProtocol`/`NexusProtocol` → `ActivitySystem`/`NexusSystem`) |
  | Level types | `ProtocolState` → `SystemState`, `ProtocolFact` → `SystemFact`, in both features; `ProtocolStep` aliases go |
  | Record designs (decision 19) | `currentAdmission` → `activityRecord`, `staleAdmission` → `trustingActivityRecord`, `heldAdmission` → `heldDispatch`, `admissionResponseLoss` → `lostStartAnswer`; members `currentRecord` → `recordMember`, `staleRecord` → `trustingRecordMember`; objects `Admission` → `ActivityRecord`, `StaleAdmission` → `TrustingActivityRecord`, `HeldAdmission` → `HeldDispatch`, `ResponseLoss` → `LostStartAnswer` |
  | Compositions (decision 19) | `currentOverQueue` → `recordOverQueue`, `staleOverQueue` → `trustingRecordOverQueue`, `currentOverMatching` → `recordOverMatching`, `staleOverMatching` → `trustingRecordOverMatching`, `currentOverForgetful` → `recordOverForgetful`, `currentOverVolatile` → `recordOverVolatile`, `currentOverLossyMatching` → `recordOverLossyMatching`; their Query vectors accordingly |
  | Actions | `worker.attemptStart` → `worker.poll`, `worker.attemptResult` → `worker.respond`, `history.answerDelivery` → `history.answerMatching`, `handler.handlerReply` → `handler.reply` (Nexus caller and Nexus operation), `network.transportFault` → `network.fault`, `caller.callerClose` → `caller.close`, `handler.handlerFinish` → `handler.finish`, `worker.workerStop` → `worker.stop`, `worker.workerResume` → `worker.resume`; action classes follow (`attemptResult-completed` → `respond-completed`) |
  | Shared task queue (R20 level names) | `DispatchQueue` → `TaskQueueProduct` (`dispatchQueue` → `taskQueueProduct`, `dispatchQueueUnderStorageLoss` → `taskQueueProductUnderStorageLoss`), `MatchingQueue` → `TaskQueueSystem` (`matchingQueue` → `taskQueueSystem`); the variants (`forgetfulQueue`, `volatileQueue`, `lossyMatchingQueue`) keep their names; Query vectors accordingly |
| System-contract level | `SystemFamily` → `RecordFamily`, family `temporal.activity.standalone.system` → `temporal.activity.standalone.record` (the shared task queue keeps the record's family), IR file `activity-system` → `activity-record` (`.json`, `.laws.json`, `.lint.json`), `activitySystemFile` → `activityRecordFile`, prose "system contract" → "history record" |

  The batch updates everything that carries these names:
  - law claim names (`activityProtocol.terminateSettles` → `activitySystem.terminateSettles`);
  - default Query names `<machine>.<scenario>.<property>`;
  - Case file names: `activity-activityProtocol.cancelIsRequested-case.json` → `activity-activitySystem.cancelIsRequested-case.json`, the same for `terminateSettles`, `activity-race-heldAdmission.staleDelivery` → `activity-race-heldDispatch.staleDelivery`, and `activity-race-admissionResponseLoss.committed` → `activity-race-lostStartAnswer.committed`;
  - Case and Contract bytes of every Case that names a renamed action, machine or family;
  - `model/cases/manifest.json`;
  - the Case-name golden `tests/testcore/testpilot/testdata/generated-case-names.txt` and the pinned Cases under `tests/testcore/testpilot/testdata/generated`;
  - `model/ir/*.laws.json`;
  - the lint acceptance keys in `model/ir/*.lint.json` (`owner` and `subjects`, e.g. `activityProtocol`, `attemptStart in started, …`, `attemptResult-canceled`);
  - the Quint/P exports;
  - Go tests that name these values (`tools/umpire/{model,lower,lint,export,conformance,cmd,internal}`, `tests/testcore/testpilot`);
  - the canary bindings (`tools/canary/{casebinding,assessment,preflight}`, `canary-gen-case`);
  - `tools/umpire/internal/golden/{config.json,original.json}`.

  Types not listed keep their IR names (`Admission*`, `OverQueue`, `OverMatching`). The `DefinitionScope("…System$package$")` pin strings stay: they name the former file `System.scala`, not a level, and changing them would change every ID under them. A comment beside each says so. The done summary lists every Query, Case, law and lint-key name that changed. Errors: a difference beyond the renamed names (a table, answer, verdict or fingerprint) stops the task.

- **R19 Product and System.** The two levels of a feature are named the Product (what a caller reads) and the System (how the server gets there). The docs of R9 use these words. "Protocol machine" and "protocol" for this level are retired from live prose. "System contract" becomes "history record". The reserved party `system`, the server, is consistent with the System level and stays. Errors: R10's retired-name check covers the identifiers. A prose grep of the R9 files for "protocol machine" and "system contract" is empty.

- **R20 Structure lint (owner request 2026-10-05, task 6).** The layout and level names of decisions 11-17 are enforced for every feature, in the TASTy lint the model gate runs (`model/irgen/Order.scala` or a sibling pass), each refusal at `<file>:<line>`:
  - (a) a Model folder under `features/` or `shared/` whose Models include a refinement pair has `product/Product.scala` and `system/System.scala` (a single-level folder such as `shared/worker` or `nexusoperation` keeps one feature file); its root feature file holds types, signature and `object exports` and no machine; zoomed-in Models sit only in subfolders of `product/` or `system/`, each with its own feature file;
  - (b) the machine in `product/` is `<Prefix>Product` and refines nothing; the System machine in `system/` is `<Prefix>System`, with the same prefix, and its `object refinement` refines that Product;
  - (c) section names form a closed set (`states`, `effects`, `monitors`, `rules`, `properties`, `implements`, `queries`, `refinement`, `syncs`, the signature's actor and section objects); any other `Section` is refused; a feature has exactly one `object exports`, in its root feature file.
  - `shared/` follows the same rules (owner, 2026-10-05): a shared Model's contract is its Product and its provider its System. A positive fixture, `model/irgen/testdata/layout/` (a small two-level feature, the template for new features), passes; one refusal fixture per rule. `model/README.md` "Writing a Model" states the layout and points to the fixture. The Go layout test (R10) keeps the repo-wide prose and path checks.

- **R21 `stuck-state` lint (owner request 2026-10-05).** A new default lint kind in `tools/umpire/lint`: a reachable state of a machine (or composition) that is not an `end` state and in which no action class has an enabled row. It reports the machine, the state and a shortest path to it, at the machine's position. It catches a forgotten timer or internal-step case, which `silent-rejection` (actor actions only) and `never-enabled` (only an action enabled nowhere) miss. Deliberate stuck states are accepted in `model/ir/*.lint.json` with a reason, like every finding. One finding fixture and one passing fixture; the README's lint table lists it.

## API sketch

The final shape, after task 6. Before task 6, the same objects carry today's names (`ActivityProtocol`, `worker.attemptStart`).

```scala
/* The standalone activity: ActivityProduct, ActivitySystem (refines it), ActivityWorker and
 * StandaloneActivity (the system with the worker of its task queue), then the IR files.
 * Realization.scala realizes it; record/ and withTaskQueue/ hold the history record. */
package temporal
package features.standaloneactivity

import umpire.*
import shared.worker.{worker as process}

given DefinitionScope = DefinitionScope("temporal.standaloneactivity.Model$package$")
given Family = Family("temporal.activity.standalone")

// -- types (top level, so IR type names stay temporal.standaloneactivity.<Type>)
enum ProductPhase derives Finite: …
final case class ProductState(phase: ProductPhase) derives Finite
enum Phase derives Finite: …
final case class SystemState(phase: Phase, attempts: UpTo[2], scheduleToClose: Timeout, …) derives Finite
enum Outcome derives Finite:
  case accepted, notFound
given Ok[Outcome] = Ok(Outcome.accepted)

// -- signature: who acts, and on what. Actor and section objects are transparent to Definition
// IDs, so every action keeps the ID the file's pin gives it.
val activity = Entity(key = "activityId")
val scheduleToStart = input[Timeout]                     // no longer collides with the timer
…
object caller extends Actor:
  val start = action(this).input(scheduleToClose).input(scheduleToStart).input(startToClose).creates(activity)…
  val control = action(this).on(activity).input(Inputs.control)…
object worker extends Section:                           // the shared worker party, on this activity
  val poll = action(process).on(activity).schema[PollActivityTaskQueueResponse]
  val respond = action(process).on(activity).input(Inputs.result)…
object timers extends Section:
  val timeout = timer
  val backoff = timer
object deadline extends Section:
  val scheduleToClose = timer
  val scheduleToStart = timer
  val startToClose = timer

object ActivityProduct extends Machine[ProductState, Outcome, ProductFact]:
  import ProductPhase.*, ProductFact.*
  val entity = activity
  val start = ProductState(scheduled)
  def end(s: ProductState) = terminal(s.phase)
  def phase(s: ProductState) = s.phase
  def terminal(p: ProductPhase) = p.in(completed, failed, canceled, terminated, timedOut)
  def paused(s: ProductState) = s.phase == ProductPhase.paused
  def running(s: ProductState) = s.phase == started

  object effects extends Section:
    def startAttempt(s: ProductState) = enter(ProductState(started), statusStarted)
    def complete(s: ProductState) = enter(ProductState(completed), statusCompleted)
    def retry(s: ProductState) = enter(ProductState(scheduled), statusScheduled)
    def notFound(s: ProductState) = List(Step(Outcome.notFound, s))
    …

  object rules extends Rules(_.phase):
    in(scheduled) { worker.poll ~> effects.startAttempt }
    in(started) { worker.respond(AttemptResult.failed(true)) ~> effects.retry }
    in(scheduled, started) { caller.control(Control.pause) ~> effects.pause }
    in(paused) { caller.control(Control.unpause) ~> effects.resume }
    in(completed, failed, canceled, terminated, timedOut) { caller.control ~> effects.notFound }
    …
    disabled(process.stop)                               // the activity does not feel its worker stop

  object laws extends Section:
    val adopted = capabilities(limits = three)(
      Closable(status = phase, terminal = terminal, rejected = cited(Outcome.notFound, notFoundCode)),
      Pausable(pause = caller.control(Control.pause), unpause = caller.control(Control.unpause), paused = paused),
      Pollable(dispatch = worker.poll, running = running)
    )

object ActivitySystem extends Machine[SystemState, Outcome, SystemFact]:
  import Phase.*, SystemFact.*
  val entity = activity
  val start = SystemState(unstarted, UpTo(0), Timeout.unset, Timeout.unset, Timeout.unset)
  def end(s: SystemState) = terminal(s.phase)
  val refines = refinement(ActivityProduct)(productOf)
  val unobservable = List(timers.backoff)
  def terminal(p: Phase) = p.in(completed, failed, canceled, terminated, timedOut)
  def live(p: Phase) = p.in(scheduled, backingOff, started, paused, pauseRequested, cancelRequested)
  def productOf(s: SystemState): ProductState = …

  object effects extends Section:                        // what happens; never whether
    def schedule(s: SystemState, toClose: Timeout, toStart: Timeout, startToClose: Timeout) =
      enter(SystemState(scheduled, UpTo(0), toClose, toStart, startToClose), statusScheduled)
    def startAttempt(s: SystemState) =
      enter(s.copy(phase = started, attempts = saturatingSucc(s.attempts)), statusStarted, attemptCount)
    def complete(s: SystemState) = enter(s.copy(phase = completed), statusCompleted)
    def cancel(s: SystemState) = enter(s.copy(phase = canceled), statusCanceled)
    def backOff(s: SystemState) = enter(s.copy(phase = backingOff), statusScheduled, attemptCount)
      .because("a retryable failure backs off; the caller reads scheduled again")
    def requestPause(s: SystemState) = enter(s.copy(phase = pauseRequested), statusPaused)
      .because("the worker learns of the pause on its next heartbeat")
    def timedOut(s: SystemState, t: TimeoutType) = enter(s.copy(phase = timedOut), statusTimedOut(t))
    def keep(s: SystemState) = stay(s)
    …

  object rules extends Rules(_.phase):                   // when each action fires; disjoint per class
    in(unstarted) { caller.start ~> effects.schedule }
    in(scheduled) { worker.poll ~> effects.startAttempt }
    in(started, pauseRequested, cancelRequested) {
      worker.respond(AttemptResult.completed) ~> effects.complete
      worker.respond(AttemptResult.failed(false)) ~> effects.fail
    }
    in(started) {
      worker.respond(AttemptResult.failed(true)) ~> effects.backOff
      caller.control(Control.pause) ~> effects.requestPause
    }
    in(cancelRequested) {
      worker.respond(AttemptResult.failed(true)) ~> effects.cancel
      worker.respond(AttemptResult.canceled) ~> effects.cancel
    }
    in(scheduled, backingOff) { caller.control(Control.pause) ~> effects.pause }
    in(paused) { caller.control(Control.unpause) ~> effects.resume }
    in(backingOff) { timers.backoff ~> effects.retry }
    when(s => live(s.phase)) {
      caller.control(Control.requestCancel) ~> effects.requestCancel
      caller.control(Control.terminate) ~> effects.terminate
    }
    when(s => terminal(s.phase)) { caller.control ~> effects.notFound }
    when(s => live(s.phase) && s.scheduleToClose == Timeout.expires) {
      deadline.scheduleToClose ~> (s => effects.timedOut(s, TimeoutType.scheduleToClose))
    }
    …
    when(_ => true) { process.stop ~> effects.keep }     // keeps the state; a Known Gap

  object properties extends Section:                     // what it promises, read on its own
    val completes = property when worker.respond(AttemptResult.completed) holds { s =>
      s.state.phase == completed && s.records(statusCompleted)
    }
    …

  object laws extends Section:                           // may read the realization (R3)
    val adopted = capabilities(limits = three)(
      Terminable(terminate = caller.control(Control.terminate), settled = statusTerminated, …),
      Cancelable(requestCancel = caller.control(Control.requestCancel), …),
      Describable(status = ActivityRealization.activityStatus)
    )

  object queries extends Section:
    val cancelRequestedThenCanceled = scenario.actions(caller.start(), worker.poll, …)  // two Queries
    val completion = (query find properties.completes in scenario("completed")
      .actions(caller.start(), worker.poll, worker.respond(AttemptResult.completed))
      limits three total 864).expect(satisfied)
    val cancel = query find properties.canceledByWorker in cancelRequestedThenCanceled limits four total 1152
    …

object ActivityWorker extends Derived(shared.worker.Polling.restrict(process.stop, process.serve))

final case class StandaloneActivityState(activity: SystemState, worker: WorkerState)

object StandaloneActivity extends Composition[StandaloneActivityState](
      _.activity -> ActivitySystem, _.worker -> ActivityWorker):
  def end(s: StandaloneActivityState) = ActivitySystem.terminal(s.activity.phase)
  object syncs extends Syncs:
    sync(_.activity -> process.stop, _.worker -> process.stop)
    sync(_.activity -> worker.poll, _.worker -> process.serve)
  object properties extends Section:
    val startedByPollingWorker = …
  object queries extends Section:
    val stoppedWorkerStartsNothing = query verify properties.startedByPollingWorker in … limits six total 3456

object Files:
  val activityFile = irFile("activity")(StandaloneActivity, ActivityProduct, ActivitySystem.queries.all, …)
```

A derived machine and a composition, in `record/Record.scala` and `withTaskQueue/WithTaskQueue.scala`:

```scala
// The trusting design admits whatever arrives; the rechecking one rereads eligibility first.
object TrustingRecord extends Derived(
      RecheckingRecord.rebind(when(_ => true) { worker.poll ~> RecheckingRecord.effects.admit })):
  object queries extends Section:
    val staleDeliveryAdmitsTwo = query verify … limits five total …

object RecheckingMember extends Derived(RecheckingRecord.unmonitored)

final case class OverQueue(activity: AdmissionState, queue: QueueView)

object RecheckingOverQueue extends Composition[OverQueue](
      _.activity -> RecheckingMember, _.queue -> DispatchQueue):
  def end(s: OverQueue) = RecheckingRecord.end(s.activity)
  object syncs extends Syncs:
    sync(_.activity -> history.dispatch, _.queue -> queue.enqueue)
    sync("admit", _.activity -> worker.poll, _.queue -> queue.deliver)
    sync("settle", _.activity -> history.answerMatching, _.queue -> queue.acknowledge)
  object laws extends Section:
    val adopted = overQueueCapabilities(this)            // Pausable(paused = through(_.activity, RecheckingRecord.paused)), …
  object queries extends Section: …

object TrustingOverQueue extends Composition(RecheckingOverQueue.withMember(_.activity -> TrustingMember))
```

## Boundaries / Non-goals

- No Quint syntax and no keyword aliases (`run`, `temporal`, `const`). fn-120's boundary stands. The Quint export still emits one module per IR file (`tools/umpire/export/quint.go`) and does not depend on the Scala layout.
- No semantic change before R18 and none in it: no IR schema change, and no change to the reader, the lowering or the exports beyond renamed values. The new framework code is limited to:
  - the `Section`, `Actor`, `Rules`, `Syncs`, `Machine`, `Derived` and `Composition` object forms;
  - the rule sugar;
  - the lint.
- `Realization.scala` is not nested in a machine object. A nested pin is refused, and nesting it unpinned would change its Definition IDs.
- The subject folders stay, renamed (R12). Their reason, that otherwise each file holds "three unrelated subjects", is unchanged.
- No index or re-export module (option b of the study). Scala `export` creates forwarder vals that the lifter would see as a second declaration of the same name.
- A machine root does not imply its sections in an IR file (R15).
- No settings (fn-125), faults (fn-123) or rejecting rows here.
- Ranks 7 and 8 of the study (realization helpers, the deadline helper, `UpTo.succ`, enum status methods) are not here; see `.plans/DSL_SIMPLIFICATION.md`.

**fn-112 and fn-114 decisions this supersedes:**

- fn-112's "files by kind, folders by subject" becomes "files by subject, sections by kind". This replaces the layout table in fn-112's Architecture section and R11's per-subject Model/Properties/Queries files. Their reasons still hold or are no longer needed:
  - the init cycle that kept monitors in `Model.scala` is gone, because R4 checks order and cycles;
  - "Properties read on their own" survives as the `properties` section;
  - "three unrelated subjects per file" survives as the subject folders.
- fn-112.6's explicit disabled arms become positive rules plus `disabled(action)` plus the accepted lint findings (R16).
- fn-114 R10's four-file rule, and the `model/README.md` text that states it, are replaced by R1 and R2.
- fn-114 R7 placed `irFile` values in each folder's `IrFiles.scala`. Only that placement is superseded, by `object Files`; the `irFile` value form is unchanged.
- fn-114's captured val names extend to object names (R7).
- fn-122.8 gave each folder its own `Capabilities.scala`. That file becomes the `laws` section. fn-122.8's reason, that an entity's adoption stays with its Model, still holds.
- **Kept:** fn-112's objects per machine (they become the machine objects), the core/sugar file rule, the `DefinitionScope` rule and the semantic freeze; fn-120's ban on Quint syntax.

**Where this deviates from the study's sketch, and why:**

1. **Actions stay at the feature level, in actor objects (R14), not in the machine objects.** Product, System, the composition and the realization bind the same actions.
2. **The machine object is the machine.** The owner reversed the study's "defer `extends Machine`": names come from object names (R7), and the cost is the one lifter reshaping of R15.
3. **The feature section holds only the IR files (`object Files`).** Everything else it held now has a machine or composition object home. An object named after the feature would collide with the composition `StandaloneActivity`.
4. **The capabilities section is `laws`, not `capabilities`.** Inside `object capabilities`, the name `capabilities(…)` would resolve to the object itself and shadow the declaration form.
5. **The lint covers more than order inside an object:** it also checks cycles between owners, placement and sections.

## Owner decisions

Decided 2026-10-05.

1. **Properties form their own section after the machine.** Adopted, now as `object properties`. A Property reads its machine. A section object is lazy, so a Property is never `null` at initialization, and R4 still keeps the reading order.
2. **One machine object per machine everywhere,** including one-machine folders. One rule for the lint and the README.
3. **The `irFile` roots go in the feature file,** now `object Files`, not in `IrFiles.scala`.
4. **Step naming.** Superseded. Step functions became effects in `effects`, named for what they do (`startAttempt`, `settle`), so no step shadows its action and no `Steps` object or `…Step` suffix is needed.
5. **Land before fn-124.7 retires the golden harness.** *Revised 2026-10-05 (host, for the owner):* from task 4 on, R5 is proved by the reader projection (tables and every Check receipt) and a before/after IR projection, not by the harness; fn-124.7 retires the harness beside task 5, and task 6 proves R18 the same way, with no golden re-capture.
6. **Fold the approved study items in.** Ranks 3, 4 and 5 and the package renames, so files and names change once.
7. **Rules and effects.** One block that says only when things fire, referencing effects defined separately. `when` exists only as a rule heading (R16).
8. **Sections for every kind of member.** Section objects are transparent to Definition IDs. Only actions, monitors, assumptions, holes, channels and realizations bear IDs; Properties, Scenarios, Queries, capabilities and machines are named by their simple names, so nesting changes none of them, only function symbols. The marker still serves the lint and the ID-bearing `monitors` section and actor objects.
9. **Product and System; the system contract becomes the history record.** The System level does not absorb the record's machines. That would merge machine objects across the `record/` subject folder against R1's one object per machine, so the record level is renamed instead (R18, R19).
10. **Derived machines are `Derived` objects**, not vals in a `variants` section. One declaration form for every machine, and each design's Queries sit with it. The close policy's 116-line `designQueries` splits across its ten objects.

### Later owner decisions (2026-10-05, during task 4)

These amend R2, R15, R18 and the API sketch where they differ.

11. **`init`, not `start`.** A machine object's initial state is `val init`, as in Quint and TLA+; `start` is Temporal's own word (StartActivityExecution, `caller.start`). `def end(s)` stays.
12. **`object states`.** A machine's vocabulary (named state sets, projections and constants: `phase`, `terminal`, `paused`, `running`, …) is a section, `object states`, first after the header members.
13. **`object implements`, not `laws`.** The capabilities section is `object implements` ("Product implements Closable, Pausable"); `implements` is no Scala keyword. `Traits` was rejected because `trait` is one and the Models hold real traits. "Capabilities" stays the noun for what is adopted (`Closable`, `Pausable`, …) and "laws" for what they bring (`<machine>.<law>`, `*.laws.json`). R2's machine order becomes: header, `states`, `effects`, `monitors`, `rules`, `properties`, `implements`, `queries`.
14. **`object refinement`.** A machine that refines another groups its refinement in `object refinement`: the abstract machine, the mapping (`toProduct`, renamed from `productOf`), and `visible`/`unobservable` where declared. One refinement per machine, as the IR holds today. It replaces R15's loose `refines`, `visible` and `unobservable` members.
15. **`object exports`, not `Files`,** for a feature's `irFile` roots, each val named after its IR file (`exports.activity`). `export` itself is a Scala 3 keyword. An IR file spans several machines (and folders), so exports stay per feature, not per machine.
16. **A folder per level, by audience (amends R1 and R12; replaces the earlier flat Product/System file split).** A feature with two levels has `product/` and `system/` subfolders, each with one feature file named after it (`product/Product.scala`, `system/System.scala`), because different people read them. The feature file named after the feature folder keeps the types (under its pin), the signature and `object exports`. Folders follow audience, not refinement edges: a zoomed-in Model that explains how the server keeps a promise goes under `system/` whatever it refines, and its `object refinement` says what it refines. So `standaloneactivity/record/` and `withTaskQueue/` move into `system/` (flattened by decision 22); the composition with the worker sits in `system/`; in `nexuscaller/`, the close policy and the control go under `system/`; in `shared/taskqueue/`, the opaque contract goes under `product/` and the matching provider with its lossy, forgetful and volatile variants under `system/`. `product/` has room for Product-level elaborations. One feature file per folder still holds, so the lint needs no exception; the R10 layout test learns the new folders. Pins keep every Definition ID and type name; only paths, packages and function symbols move. Task 6 does it with the R18 renames.
17. **The Nexus caller's `object Control` is renamed `TrustingCaller`** (owner, 2026-10-05; it matches decision 19's `TrustingActivityRecord`: `Trusting<X>` is the design that skips a check the correct design makes; the `forgedCompletion` Query keeps its name) (it is a negative control, a deliberately wrong caller design, and it collides with the activity's `enum Control` and shadows the feature's `caller`). Task 8 renames it to `TrustingCaller` with the R18 batch (moved from task 5: renaming the object renames its machine, `forgedCompletion`, which R5 freezes until task 8).
18. **`type State`.** `Machine` exports its state type as `type State = S`, so a machine object's members write `s: State` and the state type is written once, in `extends Machine[S, O, F]`. No `type Outcome` alias (it would shadow each feature's `enum Outcome`). The lifter dealiases it; IR unchanged. Task 4 adds it and uses it in the activity; task 5 in the remaining Models.
19. **`ActivityRecord`.** History's record of the activity is named for what it is: the corrected design is `ActivityRecord`, the faulty one `TrustingActivityRecord`, the held race `HeldDispatch`, the lost response `LostStartAnswer`, and the compositions `RecordOver…`/`TrustingRecordOver…`. These replace R18's Record-design and Composition rows (`recheckingRecord`, `trustingRecord`, `heldDispatchRecord`, `recheckingOver…`); task 6 renames once.
20. **Failure models and negative controls are marked.** Two framework marker traits, mixed into any machine form (`Machine`, `Derived`, `Composition`) and transparent to Definition IDs (no IR change): `FailureModel`, the real server under a fault the environment can cause, whose promise must still hold; `NegativeControl`, a deliberately wrong design the checks must refuse. The lint enforces: a `NegativeControl` has at least one Query expecting a violation or counterexample, is no feature's Product or System, and nothing refines it; a `FailureModel` binds at least one fault action (the party `fault` or a `faults` section) and its Queries expect the promise to hold unless one declares otherwise; an unmarked machine that binds a fault action is refused. One refusal fixture per rule. Task 5 adds the traits and lint and marks every Model: failure models `ResponseLoss`, `HeldAdmission`, the task queue's storage-loss variant; negative controls `StaleAdmission` and its compositions, the forgetful, volatile and lossy queue providers, and the Nexus `TrustingCaller`. fn-123 (faults as environment actions, deferred) later replaces the hand-rolled fault budgets inside failure models.
21. *(Superseded by decision 23.)* Definition IDs as `<family>.<name>`, pins removed.
22. **Zoom-ins flatten into their level folder.** With IDs following packages, not files (decision 23), `system/record/` and `system/withTaskQueue/` become `system/Record.scala` and `system/WithTaskQueue.scala` beside `system/System.scala`. A level folder holds one file per subject, the level's own file named after the folder; R1's one-feature-file-per-folder rule and lint rule (d) change accordingly, and R20 checks it.
23. **A Definition ID is the declaration's fully qualified Scala name (replaces 21; reverses R6's pins and transparency and R18's "pin strings stay").** `temporal.features.standaloneactivity.caller.start`, `…standaloneactivity.system.ActivityRecord.atMostOneActiveAttempt`: package and enclosing objects included, section and actor objects too. Moving a declaration to another package or object, or renaming it, changes its ID, and that is accepted; moving it between files of one package does not. Removed: `DefinitionScope` and the lifter's pin lookup, the former-owner strings, the `Family` givens and `…Family` objects, and fn-126.3's section transparency rule (its fixtures invert). The IR's `family` field, and every ID derived from it (`<family>.query.<name>`, action-class claims, state-field atoms), takes the declaring Scala package; the lifter fills it, nobody writes it. Scala already makes fully qualified names unique; the whole-index check stays for derived IDs. Every ID changes once, in task 6's batch, with the R18 renames and one golden re-capture; tasks 4 and 5 keep today's mechanism.
24. **No hand-listed roots (`val all`).** The capabilities section is itself the declaration: `object implements extends Implements(limits = three)(Closable(…), …)`, and `exports` names `ActivityProduct.implements`. `exports` may name a `queries` section, which roots every Query in it; the lifter collects them from TASTy. A Query exported to another IR file moves to the section of the machine that file is about, rather than being skipped by a hidden rule. No IR content changes; root strings move (R5). Task 7.
25. **No `extends Section` marker on machine-level sections.** With IDs as fully qualified names (decision 23) the marker's only remaining job is telling the lint what a nested object is, and R20(c)'s closed set of names already says that: `object effects:`, `object states:` are written plainly. Base classes that carry behaviour stay (`Rules`, `Syncs`, `Refinement`, `Implements`, `Actor`); a top-level object that holds actions is a signature group without a marker. Task 7.
26. **One word: `Actor` (owner, 2026-10-05; `Party` is retired).** With decisions 23 and 25, `Actor`'s `Section` half has no job left, so one concept remains, named `Actor` everywhere: authors declare `object caller extends Actor` holding the actions it takes; `val fault = Party()` with `object faults` becomes `object fault extends Actor`; a group of actions another actor takes is a plain object (`object worker` with `action(process)`); `Party.system` becomes `Actor.system`. The IR field `Action.party = 4` is renamed `actor = 4` (same number; the Umpire IR is outside `buf breaking`), so the `model/ir` JSON key changes with task 7's ID rewrite; Go and docs follow ("the actor who takes an action"). No Case or Testpilot proto carries it. Actor names (`caller`, `worker`, `fault`, `system`) are unchanged. Task 7.
27. **Rules are grouped by action (owner, 2026-10-05).** `object rules extends Rules(_.phase)` holds one block per action or action class, `on(action) { in(phases) ~> effects.x }`; each line is a case: `in(…)` takes phases or named phase sets from `states`, `.where(…)` adds a condition beyond the phase, `always` holds in every state, and an effect taking arguments binds directly (`effects.timeOut(scheduleToClose)`). `when` is retired as a rule heading. Cases of one block must be disjoint (the existing check); unlisted means disabled (no catch-all). Lowering, tables and IR are unchanged; only function bodies may reorder (R5). Task 7 changes the sugar and converts every Model's rules mechanically.
28. **Less ceremony, no meaning change (from `.plans/ACTIVITY_MODEL_COMPARISON.md`).** Task 7: the gate computes each Query's search `total`, and an author's value becomes an optional check; a machine's `entity` is inferred from its actions' `.on(…)`; a `reject(outcome, s)` helper replaces the `…Step` aliases and explicit `List[…]` result types; state literals use named arguments. Task 8: Scenario names come from their `val` (the rename batch may change names), and each feature header states the Model's intent as upstream's Go model does: update the Model independently of the implementation, and when conformance fails, ask a human rather than fitting the Model to the code (also an `AGENTS.md` rule).
29. **A shared `Client`, the API client, in the Temporal kit (owner, 2026-10-06).** The kit (`model/temporal/…`) declares `trait Client extends Actor`: a caller of Temporal's public API through the frontend. A feature's API caller is `object client extends Client` holding its own actions (`client.start`, `client.control`): the standalone activity's and the standalone Nexus operation's `caller` become `client`. The Nexus caller feature keeps `caller`/`handler`, its Nexus roles. Client-wide facts (how a client issues requests) may later be declared once on the trait. Task 8 renames with the R18 batch (actor name, actions' IDs, Case and law names follow); the trait itself is plain framework-free kit code.

## Ordering

- **Entry:** fn-114, fn-118 and fn-122 are closed, and fn-127 (Simplify the DSL's words) is closed. That spec renames `accept` → `enter` across every Model and the realization words in every `Realization.scala`. Landing it first means the layout move carries renamed code and is never rebased onto a word rename.
- **Not concurrently with fn-124.8**, which moves `tools/umpire/model` (where R10's test lives) and forbids other edits to `model/` and `tools/umpire/model` while it runs.
- **Before fn-124.7**, which retires the golden harness that R5 and R18 record into.
- **Before fn-125 resumes** (tasks 2 to 11 are blocked). Its settings then land as members of the machine objects, like Quint `const`s: a setting that one machine reads goes in that machine's header members, and one read by several machines goes in the signature. fn-125's sketch, which today names `nexuscaller/Model.scala` and `.setting`, is amended when it resumes.
- **fn-123** (faults) has no tasks yet. If it is planned first, it must not convert the queue providers or the response-loss machine while this spec moves them.
- **Inside the spec:** task 1 → task 2 → task 3 → task 4 → task 5 → task 6. Each task regenerates IR and Cases serially; no two run at once.

## Early proof points

- **Task 1** converts `standaloneactivity` (all three subpackages, renamed) and lands R4. If R5 needs any delta beyond positions, root strings and function symbols, stop and re-evaluate before the Nexus folders and `shared/`.
- **Task 4** reshapes the lifter on the standalone activity alone. If the activity's tables, refinement rows or Query answers move, or if the disjointness check cannot be made exhaustive over the `Finite` domain in the gate, stop before task 5.

## Verification

```bash
make umpire-check-model && make lint-model && make lint-code-fast   # the gate also runs the Models' munit tests (StandaloneActivityPins, Catalog)
go test -count=1 -tags test_dep -p 2 ./tools/umpire/...      # golden harness, layout test, Go tests naming Scala positions
make umpire-check-cases && make umpire-check-fixtures && make canary-check-case
```

Regenerate with the gate's `--update`, then `make umpire-gen-cases umpire-gen-fixtures canary-gen-case`. Tasks 1 to 5: the reader's tables, Definition IDs, IR type names, Query answers, lint findings and Contracts are identical before and after. Diffs in `model/ir/**` and `model/cases/**` contain only R5's recorded deltas. Task 6: the diffs contain only R18's renames. Each R4, R15, R16 and R17 refusal fixture is refused at its line.

## Requirement coverage

| Task | Requirements | Gate |
| --- | --- | --- |
| .1 standalone activity in feature files, `record/`/`withTaskQueue/`, declaration-order lint | R1-R8 (activity), R4, R12 | fn-114, fn-118, fn-122 and fn-127 closed; not concurrent with fn-124.8 |
| .2 Nexus folders and `shared/` in feature files, shared bounds, layout test, layout docs | R1-R3, R5, R9 (layout), R10, R13 (bounds) | after .1 |
| .3 actor objects, transparent sections | R6, R14, R5 | after .2 |
| .4 machine objects, effects, rules and sections: framework, lifter, standalone activity | R15, R16, R17, R13 (inlining, activity), R4 (e), R5 | after .3; before fn-124.7 |
| .5 remaining Models as machine objects; builder forms retired; docs | R15-R17, R13, R9 | after .4 |
| .6 rename batch, Product and System, history record; close | R18, R19, R20, R10 (names), R11, R9 | after .5; before fn-124.7 |
