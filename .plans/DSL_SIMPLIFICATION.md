# Simplifying the Scala DSL

Research note, 2026-10-05. The question: which repetitions, collisions and missing abstractions make the
Scala Models hard to read, and what Scala 3.9 can do about them without weakening what the lifter
reads? The note is grounded in the 45 files under `model/temporal` after fn-114.9's folder grouping,
`model/umpire`, `model/irgen`, `.plans/QUINT_MODULE_LAYOUT.md`, `.plans/DSL_OPERATORS.md` and the
specs fn-112, fn-114, fn-118, fn-120, fn-122 and fn-125. Line numbers are from the day of the study
and drift with later edits.

## Owner decisions (2026-10-05)

All approved. Where each one went:

| Rank | Change | Decision | Spec |
| --- | --- | --- | --- |
| 1 | `accept` → `enter`, `Accepted` → `Ok`, realize `poll` → `readUntil`, `.setting` → `.withFields`, `always` → `everyCase`, realize `Outcome` → `PropertyOutcome`; Model `Outcome.accepted` stays | adopted, **done** (fn-127.1) | fn-127 (Simplify the DSL's words) |
| 2 | `when(g) { steps }` guard sugar | **superseded**: guards became fn-126's rules, a `rules` block that says only when an action fires (`when(g) { action ~> effects.x }`, `in(phases) { … }`), with unguarded effects. `when` is a rule heading only, never a guard inside a step | fn-126 R16 |
| 3 | actions in per-actor objects; `admission/` → `record/`, `compositions/` → `withTaskQueue/` | adopted, IDs kept by transparent section objects. The package renames landed in fn-126.1; the actor and section objects (`umpire.Actor`, `umpire.Section`) in fn-126.3, with the val names kept. The close policy's sections are `callerSide` and `handlerSide`, not `caller` and `handler`, which would compile to class files whose names differ from its types `Caller` and `Handler` only in case | fn-126 R12, R14 |
| 4 | inline single-use Scenarios (name string kept); one shared `Limits` source | adopted. The shared source landed in fn-126.2: `model/temporal/shared/Bounds.scala` declares `three` (three folders) and `four` (two). The study's `five` and `twelve` are not shared: the close policy's search further than the activity's and the task queue's of the same names, so each stays in its folder | fn-126 R13 |
| 5 | one batch of val renames and new Definition IDs | adopted and **done** in fn-126's last task, together with Product/System level names and the history-record terminology; projection replaced the retired golden harness | fn-126 R18, R19 |
| 6 | `through(selector, read)` for composition law parameters | adopted, **done** (fn-127.2): one parameter list, since the curried form infers no composed state; the forwarding objects are gone | fn-127 |
| 7 | request helper, `perform(… , then = await)` | later, not planned | — |
| 8 | deadline step helper, `UpTo.succ`, enum status methods | later, if still felt after fn-126 | — |

The owner went beyond the study on one point. The study's "defer: `object X extends Machine`" (section 2) was reversed, so the machine object is the machine. Its name comes from the object (`ActivitySystem` → `activitySystem`). Its members are `init`, `end`, and the section objects `states`, `refinement`, `effects`, `monitors`, `rules`, `properties`, `implements` and `queries` (decisions 11-14). Derived machines and compositions are objects too (fn-126 R15). This **landed**: the standalone activity in fn-126.4, every other Model in fn-126.5, which retired the `machine[S, O, F] { … }` builder; rank 2's guard sugar is the rules' `when` and `in` headings. That form removes the `worker.poll ~> poll` line the study kept (section 2, "Honest answer"): a rule names the action and the effect, and nothing else binds it.

## Findings

Paths: `M` = `/Users/stephan/Workspace/skunkworks/umpire/temporal/model`; `SA` = `M/temporal/features/standaloneactivity`, `NC` = `M/temporal/features/nexuscaller`, `TQ` = `M/temporal/shared/taskqueue`. Counts are static greps over the 45 files (5,950 lines, 3,774 code) under `M/temporal`. `.plans/QUINT_MODULE_LAYOUT.md` now exists (owner adopted option (a), module object per machine, layout only); everything below builds on it and does not restate it.

### 1. Inventory: what authors write, and where the noise is

| Idiom | Count | Hot spots | Verdict |
|---|---|---|---|
| `accept(` / `stay(` / `disabled` / raw `List(Step(` | 101 / 4 / 74 / 24 | `SA/Model.scala` 29/1/22; `TQ/Model.scala` 12 raw `List(Step(` because its outcomes are not the `given Accepted` | 45 of the 74 `disabled` are inverted guards `if !g then disabled else …` (`SA/Model.scala:142,271`, `NC/Model.scala:301`); 14 `else disabled`; only 12 are match arms |
| `s.copy(` | 77 | 17 in `SA/Model.scala`, 17 in `NC/closepolicy/Model.scala` | unavoidable without priming (`x' = e`), which TASTy lifting cannot read; keep |
| `given Accepted[Outcome]` | 6 | one per Model + `TQ/Model.scala:60` in a companion | noise is small; the *name* is the problem (section 6) |
| `type XStep = Step[…]` + `: List[XStep]` return types | 11 aliases, 22 of 76 step defs annotated | required only where the body is `disabled`/`stay` (README:381-384) | fine |
| machine block: `forEntity/starts/ends/steps` | 13 machines, 190 lines, 76 `~>` + 15 in `rebind/extend` | 30 bindings are `x ~> Obj.x` (same name), 32 are `x ~> xStep/xDetail/xView` | the `~>` list is the one place a binding is declared; the duplication is the *suffix* naming convention in NC/TQ, not the list |
| explicit `disabled` arms | 12 arms total | `SA/Model.scala:304-319` (6 arms cover 12 phases ×2) | fn-112.6's "168" counts table rows, not source arms; the source already is compact |
| `object Inputs` / `input[…]` | 3 objects, 14 tokens, 14 `.input(` | exists because timers own the same names (`SA/Model.scala:48-54`) | layout, not syntax |
| `choose`/`choice` | 8 / 12 | each alternative one step, as fn-120 requires | fine; `TQ/Model.scala:74-87` is 14 lines because its outcomes are written out |
| `scenario.actions(` / `query` / `limits` / `total` | 63 / 88 / 82 / 88 | every Query restates `limits X total N`; 11 of SA's Scenarios are used by exactly one Query (`SA/Queries.scala:101-167` → `:177-194`) | `total` is an owner decision (fn-112.11 R19); the Scenario/Query split is layout |
| `Limits(` | 15 declarations, 9 names; `three` declared identically 3× (`SA/Queries.scala:170`, `NC/Queries.scala:93`, `…/nexusoperation/Queries.scala:10`) | | one shared source is missing |
| bundles (case class of Properties) | 7 classes, 37 fields, each name written 3× (`NC/closepolicy/Properties.scala:61-80` has 18) | | price of the lambda-free fold |
| `given Family` / `DefinitionScope` / `import X.given` | 8 / 12 (5 pins) / 8 | | fixed by the module layout (one pin per module object) |
| realization: `field(_.x) := …` | 48 lines; `field(_.namespace) := workerNamespace` 15×, `field(_.<id>) := run` 11× | `SA/Realization.scala:81-108`, `…/nexusoperation/Realization.scala:51-69` | the biggest *mechanical* repetition in the tree |
| realization: `perform(x -> cmd)` + `onPath(x)(await)` pairs | 20 / 13 | `SA/Realization.scala:122-145` | pattern, see section 4 |
| Realization lines vs Model lines | 973 vs 1,062 | NC 536 vs 513 | realizations are as long as the Models they realize |

Longest declarations: `closepolicy/Queries.scala:18-133 designQueries` 116 lines, `closepolicy/Properties.scala:82-159 designClaims` 78, `SA/Model.scala:293-322 System.control` 30.

### 2. Scala 3.9 syntax: can we have Quint's `action`?

Quint: `action withdraw(account, amount) = all { balances.get(account) >= amount, balances' = … }` ([quint.sh/docs/lang](https://quint.sh/docs/lang)). The three ingredients are a named declaration whose body is the step, a guard that is a conjunct, and priming. Scala 3 gives the first two; priming is impossible without a macro or evaluator, and `derives Finite` state classes with `copy` are the honest substitute.

Status per feature (sources fetched 2026-10-05; project pins `//> using scala 3.9.0`, `M/project.scala:5`):

| Candidate | 3.9 status | Lifter can read it as today's IR? | Lifter cost | Sugar/core rule | Verdict |
|---|---|---|---|---|---|
| `def` step + guard sugar `when(g) { steps }` lowering to `if g then steps else Nil` | plain `def` (stable) | yes, same `if` IR | low: one case in `M/irgen/Syntax.scala` + 2 fixtures | sugar (`umpire/Syntax.scala`), `Core form:` doc | **adopt**; note `.plans/DSL_OPERATORS.md` rejected `guard(cond)` as "plain Scala"; the owner now asks for it, and 45 inverted guards are the evidence. `when` collides with `property when class` only by word; receivers differ, the precedent is `in` (DSL_OPERATORS, candidate 2) |
| `require(g)` as the guard | supported today (`M/irgen/Expressions.scala:237-244`) | **no**: it lifts as a *precondition*; a call outside it is an error, not a disabled step (`M/SEMANTICS.md:55`) | — | — | do not reuse for guards |
| `inline` / `transparent inline` helpers | stable; bodies kept, call sites become `INLINED(expansion, call)` ([inline](https://docs.scala-lang.org/scala3/reference/metaprogramming/inline.html), TastyFormat 28.9) | only `Inlined(_, Nil, e)` is unwrapped (`M/irgen/Expressions.scala:104,119,286`) | each inline helper's expansion leaks into every call site | DSL_OPERATORS rule 5 forbids | no (keep `compose` as the single exception, `M/umpire/Compose.scala:106`) |
| macro annotations `@action` (SIP-63) | **experimental** in 3.9 (`@experimental MacroAnnotation`, [PR #80](https://github.com/scala/improvement-proposals/pull/80) under review) | n/a | — | — | no |
| plain annotation `@binds(poll) def …` (ANNOTATION kept in TASTy) | stable | lifter could collect bindings; the *runtime* `Machine` cannot (no reflection) → lifter and Scala disagree; breaks munit pins | medium | violates "declared, not defaulted" | no |
| `object ActivityProduct extends Machine[S,O,F]` with `val poll = on(action) { s => … }` members | stable | new declaration shape: ClassDef template instead of the lambda `Block` at `M/irgen/Declarations.scala:149-152`; name from object symbol (changes `activityProduct` → `ActivityProduct` in IR, Query names, Case IDs unless a name literal is kept) | **high** | core | defer; the module-layout spec gives the cohesion with the existing `machine {}` block |
| bindings derived by name (`steps` finds `def poll` for `val poll`) | — | lifter could; runtime could not | — | "declared, not defaulted" | no |
| named tuples for bundles (SIP-58, stable 3.7) | stable | names erase; selection is `NamedTuple.apply` by index → lifter must map name→index from the type | medium | — | no; case-class bundle stays |
| enum methods `s.phase.terminal` | stable (enums desugar to vals/classes in TASTy) | lifter lifts top-level/object defs with explicit params; a member def on the enum class is new | medium | core | nice-to-have, after layout |
| named-field patterns `case failed(retryable = r)` (folded into SIP-58) | stable | positional UNAPPLY in TASTy, free | none | — | free readability, optional |
| `into` (SIP-71, stable 3.9), `tracked` (experimental), relaxed lambdas (experimental), `export` | — | `export` makes forwarder vals the lifter refuses as duplicates (QUINT doc §4b) | — | — | no author-facing use |
| indentation syntax `machine[…]:` instead of `{ }` | stable | identical trees | none | style | optional |

**Honest answer:** Scala cannot give a keyword, but `object Module: … def poll(s) = when(s.phase == scheduled)(enter(ProductState(started), statusStarted))` plus the machine's `steps(poll ~> poll)` line is the Quint shape minus priming. The `~>` line is the one duplication Quint avoids; removing it costs either the high-cost `extends Machine` form or an undeclared by-name binding. Recommend keeping `~>` for now.

#### Before/after (real features)

(a) Guarded step with positive guards and the renamed verb (`SA/Model.scala:270-284`, 15 → 13 lines, reads top-down):
```scala
def respond(s: SystemState, result: AttemptResult) = when(held(s.phase)):
  result match
    case AttemptResult.completed         => enter(s.copy(phase = completed), statusCompleted)
    case AttemptResult.failed(retryable) =>
      if !retryable then enter(s.copy(phase = failed), statusFailed)
      else if s.phase == cancelRequested then enter(s.copy(phase = canceled), statusCanceled)
      else if s.phase == pauseRequested then enter(s.copy(phase = paused), statusPaused)
      else enter(s.copy(phase = backingOff), statusScheduled, attemptCount)
        .because("a retryable failure backs off; the caller reads scheduled again")
    case AttemptResult.canceled =>
      when(s.phase == cancelRequested)(enter(s.copy(phase = canceled), statusCanceled))
```
Same IR (`if g then … else Nil`), zero Case change.

(b) Scenario inlined into its one Query (`SA/Queries.scala:102-103` + `:178-179`, 6 lines across two objects → 2):
```scala
val completion = (query find completes in activitySystem.scenario("completed")
  .actions(start(), poll, respond(AttemptResult.completed)) limits three total 864).expect(satisfied)
```
Already legal today; IR identical if the Scenario keeps its name string. Applies to 8 of SA's 11 paths (~25 lines), to `nexusoperation`, and inside `designQueries`.

(c) Request-field repetition (`SA/Realization.scala:92-108`, 17 → 5 lines) with a feature-local kit helper:
```scala
private def activityCall(method: …) = rpc(workflowService, method) { field(_.namespace) := workerNamespace; field(_.activityId) := run }
private val pauseActivity = activityCall(METHOD_PAUSE_ACTIVITY_EXECUTION)
```
Blocked today: each `rpc` fixes `Req` from the method constant, and `field(_.activityId)` needs one `Req`; a helper over several request types needs the lifter to fold a def with a method-constant parameter through `M/irgen/Realizations.scala` (verify; medium). Saves ~40 lines across the three realizations.

(d) Shared Limits: delete two of the three identical `three` (`NC/Queries.scala:93`, `nexusoperation/Queries.scala:10`) for one `M/temporal/shared/Bounds.scala`; Limits are inlined per Query in IR, name preserved → IR unchanged except positions.

### 3. Structural simplifications and missing abstractions

| # | Repetition | Where (count) | Sketch | Lines | Layer | IR impact | Over-abstraction risk |
|---|---|---|---|---|---|---|---|
| S1 | positive guards (`when`) | 45 inverted guards | section 2 | ~0-1/function, readability | framework sugar | none | low |
| S2 | Scenario used once, declared elsewhere | SA 8/11, nexusoperation, closepolicy | inline `scenario("name").actions(…)` into the Query | ~25 (SA) | convention | none (keep name) | none; README:518 rationale ("Queries.scala holds Scenarios") survives inside a module section |
| S3 | identical `Limits` | 3× `three`, 2× `five`/`twelve` | one shared object | ~8 | kit (`temporal/shared`) | positions only | none |
| S4 | composition forwarding objects | `SA/compositions/Model.scala:31-35, 53-57` (8 defs); recurs per composition | core `through(_.activity)(Admission.paused)` folding selector∘predicate for law parameters | ~16 now, more per future composition | framework core | lifter medium (fold composition where a lambda is refused today, README:618-620) | low |
| S5 | deadline timer steps | `SA/Model.scala:331-344`, `NC/Model.scala:364-385`, `…/admission/Model.scala:129-138` (9 defs, 5 lines each) | `deadline(covers = waiting, armed = _.scheduleToStart)(fires = …)` in `temporal/shared` | ~25 | kit | same IR if lifted as a call of a step-giving function with function-valued params (verify; the fold binds defs for `Declares[S]` functions, README:402-407, not yet documented for step functions) | medium: the three guards differ per feature (`live`/`waiting`/`held`); keep explicit if the helper needs more than 2 parameters |
| S6 | `saturatingSucc` | `SA/Model.scala:240`, `NC/Model.scala:267` | `UpTo.succ` in framework `Domain.scala` | 2 | framework | function name moves (`function_name_substitutions`) | none |
| S7 | status-set defs per machine (`phase/terminal/ends/paused/running/held`) | 26 defs, ~40 lines | enum methods (`s.phase.terminal`) | ~10 | framework | lifter medium | low; module layout already co-locates them — do after |
| S8 | explicit disabled arms | `SA/Model.scala:304-319` | none; owner decision (fn-112.6: no wildcard in a feature step) and fn-120's lint reports never-enabled pairs | 0 | — | — | flag only |
| S9 | control-as-one-action vs four actions | SA folds pause/unpause/requestCancel/terminate into `control(Control.x)` (`SA/Model.scala:75-84`); `nexusoperation` declares `requestCancel`, `terminate` separately | pick one convention per kit | 0 | convention | SA change = new actions, IDs, Cases | flag; defer |
| S10 | bundle field names ×3 | 7 bundles | none viable (named tuples erase) | 0 | — | — | — |
| S11 | `perform(x -> cmd)` + `onPath(x)(await)` pairs | 13 pairs | `perform(control(terminate) -> terminateActivity, then = awaitTerminated)` → two Items | ~10/realization | kit core (`umpire/realize/Scripts.scala`) | lifter: realizations lift "as written" (`M/irgen/Lift.scala:16-18`); a helper returning two Items needs a fold → medium | medium |
| S12 | parallel System machines (SA vs NC: start/attempt/backoff/3 timers/`productOf`, ~140 lines each) | 2 | a shared "attempted operation" template | large on paper | — | — | **high**: the phases differ (paused/pauseRequested vs unscheduled); fn-122 deliberately shares *laws*, not machines (vision #PROTOCOLS). Do not |
| S13 | `total n` author-computed | 88 | owner decision fn-112.11 R19 ("no automatic helper"); reader already prints both numbers and factors | 0 | — | — | respect; mention only that the module layout halves the distance between a Scenario and its total |

What steps already imply (facts/outcomes lists, evidence defaults, start defaults) was settled by fn-112/fn-114 and is not restated twice today; evidence lists only exceptions (`SA/Model.scala:378-381`).

### 4. Naming (owner feedback)

#### 4a. DSL keywords vs Temporal vocabulary

| DSL word | Where | Temporal meaning it collides with | Proposal | Cost |
|---|---|---|---|---|
| `accept(...)`, `given Accepted[O]` | `M/umpire/Syntax.scala:14-27`, 101 uses | Update *accepted* (`WorkflowExecutionUpdateAccepted`), Nexus accepted | `enter(state, facts*)` / `stay(s)` / `disabled` / `choose`; `given Ok[Outcome] = Ok(Outcome.accepted)` ("ok" is the gRPC word every reader owns) | sugar rename: `Syntax.scala` ×2 + irgen case names + fixtures + scalafix rewrite; **zero IR** |
| `Outcome.accepted` enum case | 4 Models + `closepolicy Answer.accepted` | same | leave; Model vocabulary in IR type catalogs, fingerprints and Case bytes (`"definitionId":"accepted"` in `M/cases/activity-completion-case.json`) | flag only |
| `poll(...) { }` | `M/umpire/realize/Scripts.scala:89` | worker long-poll (`PollActivityTaskQueue`) | `readUntil` (kit already says `await`) | core realize rename, lifter name match; no IR |
| `.setting { field := }` | `Scripts.scala:104` | fn-125 `setting[T]` (dynamic configuration) — certain collision when fn-125 lands | `.withFields { }` | low; do before fn-125 |
| `always(command)` | `Scripts.scala:48` | the temporal operator DSL_OPERATORS reserves (`always`/`eventually`, Do-not-do 7) | `everyCase(command)` | low |
| `umpire.realize.Outcome` (satisfied/violated) vs Model `enum Outcome` | `…/admission/Queries.scala:6` already aliases `Outcome as RunOutcome` | two meanings in one file | rename realize's to `PropertyOutcome` or `Assessment` | low; IR enum names unchanged |
| `results("Delivery")` | 2 uses | activity *result* | `resultDomain("Delivery")` | low |
| `starts(...)`/`ends(...)` | 13 machines | *Started* events | mild; leave, or `initial`/`final` when the module layout rewrites every block anyway | lifter name match |
| `admission`, `admitted` | folder + 6 facts | server jargon (RecordActivityTaskStarted) | section 4c | — |

#### 4b. Action naming: actor and direction

Survey (43 actions: 24 party, 10 timers, 9 internal; all `val x = action(party)`): the actor is in the declaration (`action(shared.worker.party)`, `SA/Model.scala:64`) and invisible at every call site (`Paths.completed = …actions(start(), poll, respond(completed))`). Encoding it in names (`workerTakesAttempt`) makes Scenarios long. Proposal: **show the party by structure, at the call site** — declare actions in per-actor objects, so Scenarios read as scripts: `actions(caller.start(), worker.poll, worker.respond(completed))`. This fits the module layout (actions are shared by Product/System/record machines, so they live at feature level, and feature-level objects are where the layout spec puts the pins).

| Feature | today | proposed (object.val) | note |
|---|---|---|---|
| SA | `start`, `control(c)` | `caller.start`, `caller.control(c)` | keep names |
| SA | `poll`, `respond(r)` | `worker.poll`, `worker.respond(r)` | `poll` = PollActivityTaskQueue, `respond` = RespondActivityTask* |
| SA timers | `backoff`, `scheduleToStart`… | `expires.scheduleToStart`, `timers.backoff` | frees the `Inputs.` prefix: `start(scheduleToStart := expires)` |
| admission | `dispatch`, `answerMatching` | `history.dispatch`, `history.answerMatching` | internal steps named by the server component |
| NC | `schedule`, `reply(r)`, `complete(r)`, `fault`, `Control.inspect` | `callerWorkflow.schedule`, `handler.reply(r)`, `handler.complete(r)`, `network.fault`, `caller.inspect` | |
| closepolicy | `close`, `reset`, `requestCancel(p)`, `finish(r)`, `deliverCancel` | `caller.close`, `caller.reset`, `caller.requestCancel(p)`, `handler.finish(r)`, `server.deliverCancel` | |
| nexusoperation | `start/requestCancel/terminate`, `reply/complete` | `caller.x`, `handler.x` | |
| taskqueue | `enqueue/deliver/acknowledge`, `crash/ackLoss/storageLoss` | `queue.x`, `faults.x` | |
| worker | `stop/resume/serve` | `worker.stop/resume/serve` | the file keeps the prefix *because* of IDs (`M/temporal/shared/worker/Model.scala:56-59`) |

**Cost of a rename, precisely.** A Definition ID is owner + val name (`M/irgen/Context.scala:241-245`). *Moving* a val into `object worker` with `given DefinitionScope = DefinitionScope("temporal.worker.Worker$package$")` keeps the ID only if the val name stays; the file-level pin must then go (a pin inside a pinning owner is refused, `Context.scala:297-330`; whether two objects may pin the same former owner needs a fixture — I believe yes, since the rule is per-owner and per-ID). *Renaming* the val changes the action ID, class IDs (`respond-completed`), IR step bindings, Case Program/Contract bytes (`M/cases/activity-completion-case.json` embeds `temporal.activity.standalone.action.activitySystem.poll`), `Taking(poll, 1)` in realizations, manifest, and the Quint/P exports. The fn-115 golden (`tools/umpire/internal/golden/config.json`) knows `source_path_renames`, `function_name_substitutions`, `type_name_substitutions`, `source_label_substitutions`, `positions_by_file`, `functions_by_reference`, `inert_fields` — **no action/ID substitution**, so a rename means a re-captured `original.json`, then `umpire-gen-model`, `umpire-gen-fixtures` (pinned Cases under `tests/testcore/testpilot/testdata/generated`), `canary-gen-case`, and `.lint.json` acceptances keyed by name. Machine renames are the most expensive: they ripple into law claims `<machine>.<law>`, default Query names `<m>.<scenario>.<property>` and Case file names (`activity-activitySystem.cancelIsRequested-case.json`).

#### 4c. `admission/` and `compositions/`

| today | what it models | proposed | fits the module layout as |
|---|---|---|---|
| `admission/` | history's record of one activity and whether a delivery becomes a started attempt | `record/` (package), machine `historyRecord` | `object HistoryRecord` section after the System |
| `activityRecord` / `trustingActivityRecord` | design that re-reads eligibility / design that trusts the message | `recheckingRecord` / `trustingRecord` | derived machines inside the object |
| `heldDispatch` | the race run against a server with the dispatch held | `heldDispatchRecord` | same |
| `admissionResponseLoss` | one lost start answer | `lostStartAnswer` | same |
| `compositions/` | the record composed with the task queue | `withTaskQueue/`; `recordOverQueue` → `recheckingOverQueue`, `OverQueue`/`OverMatching` state types unchanged | `object OverTaskQueue` |

Package renames alone are cheap (types keep `temporal.standaloneactivity.<Type>` through the `System$package$` pin, README:543-547; function names and positions change → `function_name_substitutions`, `source_path_renames`, one regeneration). Machine val renames carry the full cost above (`activity-race-heldDispatch.staleDelivery-case.json`, lint keys).

### 5. Constraints respected

Vision rules (`.plans/UMPIRE4_VISION.md:86-101`): every proposal keeps knowledge in Models; S3/S5/S11 go to the kit, S1/S4/S6/S7 to the framework (no Temporal word). DSL_OPERATORS: no new symbol; `when`, `enter`, `through`, `readUntil` are words; the sugar-vs-core rule and `make lint-model`'s syntax check (`M/check/SyntaxRule.scala`) apply to S1 and the verb rename (both sugar, with `Core form:` docs and paired fixtures). fn-120: `choose` untouched, no Quint syntax (`fn-120` Boundaries). fn-122: capabilities untouched; S4 serves law parameters. fn-118: hints untouched. fn-125: rename `.setting` first. Lifter: everything is a `val`/`def` with TASTy positions; nothing inline or annotated.

### 6. Ranked recommendation

| Rank | Change | Saves / effect | Lifter | IR / Case | Risk | Spec |
|---|---|---|---|---|---|---|
| 1 | verb rename `accept`→`enter`, `Accepted`→`Ok`; `poll`→`readUntil`, `.setting`→`.withFields`, `always`→`everyCase`, realize `Outcome`→`PropertyOutcome` | readability; removes 4 collisions | name matches + fixtures | none | low | **done**: fn-127.1 |
| 2 | `when(g) { steps }` guard sugar | 45 inverted guards read forward | low | none | low | same spec |
| 3 | per-actor action objects + package renames (`record/`, `withTaskQueue/`), names kept | actor visible at every call site; `Inputs.` prefix gone | none (pins) | positions, function names | low-medium (pin fixture) | module-per-machine spec |
| 4 | inline single-use Scenarios; shared `Limits` | ~35 lines SA, cohesion | none | positions | none | module-per-machine spec |
| 5 | val renames (actions, machines, designs) in one batch | names say actor and subject | none | IDs, Cases, golden re-capture | medium | its own task at the end of the layout spec, one regeneration |
| 6 | `through(selector)(predicate)` for composition law parameters | 16 lines now, scales | medium | none | low | **done**: fn-127.2, as `through(selector, read)` |
| 7 | `activityCall(method)` request helper; `perform(… , then = await)` | ~50 lines in realizations | medium (realization fold) | none | medium | after fn-118.5 |
| 8 | deadline step helper, `UpTo.succ`, enum status methods | ~35 lines | medium | function names | medium | later, if still felt after layout |
| — | `object X extends Machine`, by-name bindings, annotations, inline, named tuples, shared protocol template | — | high / unsound | — | high | not recommended |

**Can we have Quint-like `action` syntax?** Not as a keyword and not with priming. With ranks 1-4 a step reads `def poll(s) = when(s.phase == scheduled)(enter(s.copy(phase = started), statusStarted))` beside its machine, its claims and its Queries in one module object, and a Scenario reads `actions(caller.start(), worker.poll, worker.respond(completed))`. The one line Quint saves and Scala keeps is `worker.poll ~> poll`; dropping it costs a new declaration shape in the lifter and a machine rename, which is not worth it today.
