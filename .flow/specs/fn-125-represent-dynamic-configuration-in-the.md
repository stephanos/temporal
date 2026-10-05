# Represent dynamic configuration in the Models

## Goal & Context
<!-- scope: business -->

Temporal's dynamic configuration decides what a feature does: whether a method exists (`Unimplemented` when a flag is off), which implementation runs (HSM or CHASM for workflow-scheduled Nexus operations, with different `attempt` counting), how long the server waits (timer shift, retry backoff, request timeouts). Today none of it is declared where the vision says knowledge lives. The evidence is in `.plans/DYNAMIC_CONFIG.md` (research of 2026-10-04); in short:

- **One declared setting.** `RequiredSetting("nexusoperation.enableStandalone", "true")` (fn-122.4) is the only one in any Model. The pause and unpause Cases rely on `history.enableStandaloneActivityOperatorCommands` (default `false`), which nothing declares. The harness sets it, with `activity.enableStandalone` and `history.enableChasm`, on every Case that binds no Nexus endpoint (`tests/testpilot_generated_test.go:168-172`).
- **The HSM/CHASM switch lives in Go and is wrong.** `tests/testcore/testpilot/switch.go` says the switch "is not a Model parameter: the Case bytes do not depend on it". Its `chasm` value never sets `nexusoperation.chasmWorkflowOperationsRolloutPercent` (default 0), so `UseChasmForWorkflow` (`chasm/lib/nexusoperation/config.go:69`) returns false and workflow operations run on HSM under both values. `testpilot_generated_test.go:182` appends `EnableChasm=true` after the `hsm` value's `false`, and the last value wins. Both values have therefore run nearly the same server, so no divergence could show.
- **Durations the Cases rest on are assumed, never checked.** The timer slack assumes `history.timerProcessorMaxTimeShift` is 1 s. The Nexus retry poll assumes a 1 s backoff. A Profile value changes a server timer: the Driver takes a Nexus operation's schedule-to-close timeout from the instruction timeout (`common/testing/testpilot/temporal/worker/typed.go:76`), and QLF-01 (`UMPIRE4_SPEC.md`) forbids that.
- **A hand-kept key table.** The harness turns a required key into a typed setting through a table that knows one key (`testpilot_generated_test.go:197-199`).

This spec makes dynamic configuration a declared input. A setting that changes what the Model says is a finite Model input, bound by each Query like a Limit and recorded in every Case. A setting that only enables an API is a precondition declared once, on the API. A duration a wait bound rests on is a declared inequality, and preparation checks it. Every other environment value stays in the Profile and must not change behavior. It serves the model author, who states a setting once instead of discovering it in a flaky run, and the reviewer, who can read from a Case which server it needs.

## Architecture & Data Models
<!-- scope: technical -->

**Four kinds, one rule each.** The research's three classes, with "realization precondition" split by where the fact belongs:

| Kind | Example | Declared | Reaches the Case as |
| --- | --- | --- | --- |
| Model setting | Nexus implementation (attempt counting); later a small capacity limit (`MODEL_ASSURANCE.md:307-308`) | framework `setting[T]` over a finite domain, read by step functions | a valuation per Case, encoded as required settings |
| API precondition | operator commands on Pause/Unpause, `activity.enableStandalone`, `nexusoperation.enableStandalone` + `history.enableChasm`, `enableCancelWorkerPollsOnShutdown` on the `workerStop` fault | kit, on the typed method, performed command kind or Testpilot fault kind (fn-118 behavior metadata) | required settings of each Case whose Program uses it |
| Bound assumption | `timerProcessorMaxTimeShift <= 1s` under the timer slack; Nexus `retryPolicy.initialInterval` under the attempt poll | kit, on the fn-118 wait bound or kit deadline that rests on it | required settings with a relation (`atMost`, `atLeast`) |
| Behavior-neutral value | RPS limits, scanners off, partitions, long-poll intervals not under a bound, endpoints, the bound scale factor | Profile only (QLF-01) | nothing |

A value a request field can state (retry policy, deadlines, schedule-to-close) is set in the request and declared as none of these. A request field is per Case and visible in the Case's bytes; a server default is neither.

**Part A. Settings in the framework (Temporal-agnostic).** `setting[T]` declares a named input over a `Finite` domain, exactly as `input[T]` declares an action input. A step function, guard, `ends` or start reads it through a given `Valuation`. The lifter records the declaration and a `setting` expression. Go derives which settings a machine reads, transitively through calls, members and refinements, and builds one table per valuation. A setting is constant for the whole run: a setting no step can change is a constant, as in TLA+ `CONSTANT`, Quint `const` and P `param`. A Query binds settings with `under`: `s := v` fixes one value and `s.each` ranges over the domain. A Query that binds several settings covers the product of their values. A ranged Query means one Query per valuation, keyed `<query>-<valuation key>` the way an action class is keyed by its inputs. Each of these Queries is checked, answered, counted and lowered on its own. A Query that binds no setting keeps its key. `total` multiplies the per-valuation count by the number of valuations. A Query must bind every setting its machine reads. It may also bind a setting the machine does not read: that states that the behavior holds under each value, which is what the switch claims today.

**Part B. The kit maps settings to dynamic configuration (Temporal).** `model/temporal/realize` declares each dynamic-config key once, typed (bool, int, duration, string) with its scope and a citation of its definition. A Temporal-side Go test pins these declarations to the server registry (`common/dynamicconfig/registry.go`, `RegisteredSettingMetadata` in `metadata.go`): the key exists and has the same codec and scope. A realization encodes each value of each Model setting it covers as a complete assignment of kit keys (`hsm` and `chasm` each name the six keys `tests/nexus_workflow_test.go:82-94` sets, rollout percent included). API preconditions and bound assumptions are kit declarations beside fn-118's hints, and they travel in the same IR structure. The realization-level `requiredSettings` (fn-122.4) is retired: Nexus operation's flag moves to its methods.

**Part C. Cases and Testpilot.** Lowering derives each Case's `Program.required_settings` as the union of: the encoding of its Query's valuation, the preconditions of every method, performed command and fault kind its Program uses, and the assumptions of every bound its waits use. Each entry records its relation and origin. When two origins require one key with different values, lowering refuses the Case and names both origins. Nothing is resolved by "last wins". There is one Case per valuation, and its name and manifest entry carry the valuation. Preparation keeps fn-122.4's check and adds the relations. The live harness builds each cluster from exactly the Case's required settings, using a key-to-setting lookup derived from the registry, and sets an inequality's key to its declared value. A remote Profile must state the server's value for every key a Case names, or the Case is `PreparationUnavailable`. `switch.go`, `CheckSwitchAgreement`, the per-value subtests, `requiredSettingKinds` and the blanket settings go. Agreement across implementations becomes each Case's own expectation under its own valuation.

**Part D. Durations.** The Model holds no duration: timers stay actions (`API_BEHAVIOR_HINTS.md:150-160`). One declared value configures the server locally and stands behind the bound: the kit's assumption `timerMaxTimeShift atMost 1.second` is the value the functional cluster runs with and the premise of the timer slack. A remote server is checked against it. The Profile's scale factor (fn-118 R5) multiplies wait bounds only. It never changes a required value, a request field or a server timer.

**Fix-now items and where they go.**

| Item | Where | Why |
| --- | --- | --- |
| Switch rollout percent; `EnableChasm` override order | task 1 (Go only) | Until both are fixed, no HSM/CHASM evidence exists, and R12's decision depends on that evidence. The fix changes no Case bytes and waits for no spec. |
| Profile-derived Nexus schedule-to-close (`typed.go:76`) | task 1 (Go only) | It is the one place a Profile value changes server behavior. The Driver passes only the timeouts the command carries. The caller's timeout paths set their deadline explicitly (`nexuscaller/Realization.scala:385-397`), and no Model path has a timeout the default would realize. This answers fn-118's unexplained item 2. |
| Undeclared operator-commands flag | R7's task, as a precondition on Pause/Unpause | A realization-level setting would land on every activity Case and move again to the methods, changing Case bytes twice. Until then the harness's blanket setting keeps local Runs correct. A Profile without the flag fails these Cases loudly with `Unimplemented`, never vacuously. |
| ShutdownWorker fan-out race (`MILESTONES.md`) | Q1 (decided 2026-10-04): R7's precondition `frontend.enableMatchingFanOutForPollCancellation=false` on the `workerStop` fault with a reason citing upstream #9424, removed when the server fix lands; the upstream report is drafted for the owner | Not a Profile choice: `enableMatchingFanOutForPollCancellation` changes who cancels polls, so QLF-01 forbids it. Not an fn-123 fault either: a fault is an environment action the Model admits, and this is a server defect the Model should not admit. |

## API Contracts
<!-- scope: technical -->

A sketch. The framework task settles spellings within `.plans/DSL_OPERATORS.md`, and the kit task follows fn-118.2's surface.

```scala
// model/temporal/features/nexuscaller/Model.scala: a Model setting (framework construct)
enum Implementation derives Finite:
  case hsm, chasm
val implementation = setting[Implementation]

private def failAttempt(s: State)(using Valuation): Vector[Step] =  // HSM raises on failure
  val raised = if implementation.value == Implementation.hsm then saturatingSucc(s.attempts) else s.attempts
  ...

// Queries.scala
val retries = find(backsOff) in retryScenario limits bounded under (implementation.each) total 96
val terminates = find(terminated) in terminateScenario limits bounded under (implementation := Implementation.chasm) total 40

// model/temporal/realize/Settings.scala: keys, once, each citing its definition
val enableChasm = DynamicConfig.bool("history.enableChasm", Scope.namespace)            // DC:3320
val chasmWorkflowOps = DynamicConfig.bool("nexusoperation.enableChasmWorkflowOperations", Scope.namespace)
val chasmWorkflowOpsRollout = DynamicConfig.int("nexusoperation.chasmWorkflowOperationsRolloutPercent", Scope.namespace)
val operatorCommands = DynamicConfig.bool("history.enableStandaloneActivityOperatorCommands", Scope.namespace)
val timerMaxTimeShift = DynamicConfig.duration("history.timerProcessorMaxTimeShift", Scope.global)

// the caller's realization: every value, every key that selects the path
encode(implementation)(
  Implementation.hsm -> Vector(enableChasm := false, chasmWorkflowOps := false, chasmWorkflowOpsRollout := 0, ...),
  Implementation.chasm -> Vector(enableChasm := true, chasmWorkflowOps := true, chasmWorkflowOpsRollout := 100, ...)
)

// Behavior.scala (fn-118): preconditions and bound assumptions
WorkflowServiceGrpc.METHOD_PAUSE_ACTIVITY_EXECUTION.requires(standaloneActivity := true, operatorCommands := true)
FaultKind.workerStop.requires(cancelPollsOnShutdown := true)
CauseKind.timer.boundedBy(WaitBound(250, 3000)).assuming(timerMaxTimeShift atMost 1.second)
```

**Umpire IR.** New: `Setting {id, name, position, domain}` (a catalog type) on the IR file; an expression `setting(id)`, whose value is the valuation's (SEMANTICS gains a Settings section, and Query totals gains the multiplier); `Query.under`, a repeated `SettingBinding {setting, values, position}`, where one value fixes the setting and all values range over it; `Realization.encodings`, a repeated `SettingEncoding {setting, repeated ValueEncoding {value, repeated RequiredSetting}}`; preconditions on fn-118's `ApiBehavior` (target: method, cause kind or fault kind, plus a repeated `RequiredSetting`); and `CauseBound.assumes`. `RequiredSetting` gains `Relation relation` (`EQUAL = 0`, `AT_MOST`, `AT_LEAST`), `position` and `because`. `Realization.required_settings = 15` is reserved once R7 lands. Results, witnesses and receipts carry the valuation key.

**Testpilot IR.** `RequiredSetting` gains `relation = 3`, `origin = 4` (`PauseActivityExecution`, `implementation=chasm`, `timer bound`) and `SourceLocation source = 5`, so a refusal names what needs the setting. `CaseProvenance` gains `valuation`. The lowered-Case manifest keys a Case by Query and valuation. Values use the registry codec's text form (`true`, `100`, `1s`). Default-empty fields leave existing IR and Case bytes unchanged. Field numbers are the next free ones when a task lands: fn-118's sketch names `Realization` field 15, which fn-122.4 has since taken.

## Edge Cases & Constraints
<!-- scope: technical -->

- **Declared, not defaulted.** No Go component and no Model assumes a server default. A key a Case needs is in its required settings, and a key no Case names is either behavior-neutral or a defect. A remote Profile that cannot state a value fails closed.
- **One valuation per run.** Composition members, a refining machine and its product all see the same valuation. A refinement is checked per valuation of the union of the settings both sides read.
- **Plain Scala still runs.** Step-function tests supply a `given Valuation`. Reading a setting with no valuation in scope does not compile.
- **Frozen where unbound.** Machines and Queries that read and bind no setting keep tables, keys, totals, answers, Definition IDs and Case bytes exactly, proven by the baseline goldens while they exist (fn-124 R7). Case-byte changes are the recorded deltas each task lists: required settings from R7, Nexus caller Cases per valuation from R8 and R12, and origins.
- **Encodings are complete and distinct.** Every value of a covered setting has an encoding. Two values with the same encoding would be one environment and are refused. The `hsm` encoding matches what the upstream HSM suites set, and `chasm` matches `tests/nexus_workflow_test.go:82-94`.
- **Exploration cost.** A ranged Query explores once per valuation. Today only the caller ranges, over two values.
- **Exports.** Quint writes a setting as a `const` bound per instance. P writes it as a `param`, or reports `UnsupportedError` with the Scala position.
- **Scope.** The functional harness sets keys globally. A namespace-scoped key on a remote server is checked for the Case's namespace binding.
- **Order.**

| Phase | Gate (conductor-held across specs) | Work |
| --- | --- | --- |
| task 1: R1, R2 | none | Go harness and Driver only, no Case bytes |
| framework: R3-R5 | fn-114 closed (paths after fn-114.9, Case freeze); not concurrent with fn-124.8 (package split) | `model/umpire`, irgen, checker, exports |
| kit and preconditions: R6, R7, R9 (harness half) | fn-118.2 done (`ApiBehavior` IR) | operator-commands and `workerStop` preconditions; harness table and blanket settings go |
| encodings and Nexus caller: R8, R9 (switch deletion), R12 | R1, R3-R7; fn-118.5 done (caller realization migrated); owner's Q2 for the Model half of R12 only | one Case per valuation; switch deleted; fn-121's Case-name golden regenerated |
| durations: R10, R11 | fn-118.4 done (derived bounds); coordinated with fn-124.3 | bound assumptions; implicit-assumption inventory |
| close: R13 | all above | docs, gates |

flowctl records no spec-level dependency, so task 1 can run now; the conductor holds the gates in this table, since flowctl stores task dependencies only within one spec. fn-122's open tasks (.5-.7) and fn-123 touch no setting surface. fn-123's environment actions are where a mid-run change of a setting would go (Boundaries).

## Acceptance Criteria
<!-- scope: both -->

- **R1:** The Nexus switch's `chasm` value sets every key that selects CHASM for workflow operations, including `nexusoperation.chasmWorkflowOperationsRolloutPercent=100`. The `hsm` value sets the HSM keys. A unit test shows that `UseChasmForWorkflow` is true under `chasm` and false under `hsm` for exactly the settings the harness passes. The harness refuses one key given two values, which ends the `EnableChasm` override at `testpilot_generated_test.go:182`. The switch applies only to Cases whose Program schedules a workflow Nexus operation. The Nexus Cases run live once under both values, and every divergence is listed for the owner. Errors: a divergence is not hidden by a Model, Case or expectation change in this task.
- **R2:** The Driver passes a schedule command's timeouts exactly as the command carries them and derives none from the instruction timeout (`worker/typed.go:76`). A test shows that changing the Profile's default instruction timeout or scale factor changes no request field the Driver sends. Case bytes are unchanged. Errors: a Case whose operation then never closes fails by its declared wait bound, naming it.
- **R3:** `setting[T]` over a `Finite` domain is declarable in `model/umpire` and readable in step functions, guards, `ends` and starts. The IR generator records the declaration and each read. Go derives the settings each machine reads and builds a table per valuation. Errors: a non-finite domain, a second declaration of one setting, and a read the IR generator cannot translate are refused at their line.
- **R4:** A Query binds settings with `under` (fix or `each`). A ranged Query is one Query per valuation, keyed with the valuation, answered and lowered separately, with the valuation in results, witnesses and receipts. `total` counts valuations. SEMANTICS states this. Errors: a Query that leaves a setting its machine reads unbound, a value outside the domain, and a setting bound twice are refused at the Query's line, naming the setting.
- **R5:** The Quint export carries settings as constants per valuation, and the agreement check passes on a fixture machine that reads one. The P export carries them or refuses. Errors: an unsupported construct is an `UnsupportedError` with its Scala position.
- **R6:** The kit declares each dynamic-config key a Model uses once, typed and scoped, citing its definition. A Temporal-side Go test checks every declared key, and every required value in every lowered Case, against the server registry (exists, codec, scope). Errors: an unknown key, a type mismatch or an unparsable value fails the test, naming the key, the value and the kit line.
- **R7:** A method, a performed command kind or a Testpilot fault kind declares preconditions in the kit. Lowering derives each Case's required settings from its Program's uses, with origin and source. Declared this way: `activity.enableStandalone` on standalone activity methods, the operator-commands flag on Pause and Unpause, `history.enableChasm` and `nexusoperation.enableStandalone` on standalone Nexus methods, and `frontend.enableCancelWorkerPollsOnShutdown` on `workerStop`. The realization-level `requiredSettings` is gone. Errors: two origins requiring one key differently are refused at lowering, naming both, and a key the kit does not declare is refused at its line.
- **R8:** A realization encodes every value of every setting its machine reads as a complete assignment of kit keys. Lowering emits one Case per valuation, with the valuation in its name, manifest entry and provenance, and the encoding in its required settings. Errors: a value with no encoding, an encoding naming an undeclared key, and two values with equal encodings are refused at their line.
- **R9:** The live harness builds each cluster from exactly its Case's required settings, through a registry-derived lookup, and records them in the Profile. `switch.go`'s switch and agreement check, the per-value subtests, `requiredSettingKinds` and the blanket activity and CHASM settings are deleted. Errors: a required key the registry lacks fails the Case, naming the key.
- **R10:** A wait bound or kit deadline that rests on a server duration declares it as an `atMost` or `atLeast` assumption. Preparation checks relations against the Profile, and the functional harness runs the key at its declared value. Each relation in `DYNAMIC_CONFIG.md` section 4 is either declared or listed with the Case that would need it. Errors: an unmet or unstated relation is `PreparationUnavailable`, naming the key, the relation, the value and the origin's position.
- **R11:** Each implicit assumption in `DYNAMIC_CONFIG.md` section 3 has a recorded disposition: request field, precondition, bound assumption, Model setting, behavior-neutral, or out of scope with a reason. Where a request field can state the value, the realization sets it. Errors: an assumption with no disposition is listed for the owner.
- **R12:** The Nexus caller's Queries bind `implementation`, so each caller Case exists under HSM and under CHASM. Where R1's runs and the owner's Q2 show the observable differs (attempt counting), the Model reads the setting. Errors: a Case that fails under one valuation is a failing Case naming its valuation, not a switch divergence.
- **R13:** `model/README.md` explains how to declare, bind and encode a setting, a precondition and a bound assumption. The Testpilot README states QLF-01 for Profiles. The model gate, `make lint-model`, the Go tooling and Testpilot suites, `make umpire-check-cases` with the recorded deltas, the live generated Cases and `make lint-code-fast` pass at the closing task.

## Boundaries
<!-- scope: business -->

- No mid-run change of a setting. A change during a run is state, not a setting, and would come as an fn-123-style environment action with a budget.
- No setting with an infinite or continuous domain, and no Model durations.
- No limit reached on purpose here: concurrency (30 HSM / 2000 CHASM), blob sizes and ID lengths stay unmodeled until a Query needs one. Part A is the way in when one does.
- No audit of the ~45 global functional-suite overrides (`tests/testcore/dynamic_config_overrides.go`) beyond R11. They stay behavior-neutral by assumption.
- Environment-specific values (callback URL template, allowed addresses) stay Profile endpoints (QLF-01).
- Realization guards do not read settings. Evidence that differs per implementation is a Model difference that the Model states.
- No server fix in this repository. Q1 (decided) requests one upstream; the `workerStop` precondition stands until it lands.

## Decision Context
<!-- scope: both -->

**A setting is a constant, as in other checkers.** TLA+ binds `CONSTANT`s per model in a `.cfg` file. Apalache ranges them with `--cinit`. Quint binds `const` per instance, and P enumerates `test param` combinations. In each, one spec is checked under several bindings, and each binding is a separate check. A state field that never changes would also work, but every member of a composition would hold its own copy, and every state key would change. A hidden counter-style context would escape plain-Scala tests.

**One Case per valuation, not one Case with a list.** The checker already treats each valuation separately, and so does the harness: one cluster each. A list would need a Testpilot field that Testpilot only passes through. Case files double for the caller only. fn-121 already shards per Case and per implementation.

**Preconditions on the API, not the realization.** The server gates the method (`chasm/lib/activity/frontend.go:460-555`). Declared on the method, the precondition holds for every Model that pauses and reaches only the Cases that pause. This is fn-118's argument for hints ("true for every Model that pauses and describes").

**Inequalities, not equalities, for durations.** A bound stays sound as long as the server's value is within the relation. Locally the harness runs the declared value, so "one value configures both" holds. A canary server keeps its own value and is checked against the relation.

**Deviations from the proposed direction (2026-10-04).** Item 3: required settings come from three sources, not only from the Query's valuation, because feature flags are API facts. Item 1: a Query may also bind a setting its machine does not read, which keeps the switch's claim that the behavior holds under each implementation. Item 4: inequalities are checked against what the Profile states, and the local harness sets the declared value. The race item: not a modeled fault (research option 2), for the reason in the table.

## Open questions for the owner

1. **ShutdownWorker race. Decided 2026-10-04: both.** Report upstream (#9424: the early return when the root partition is not loaded) and meanwhile declare `frontend.enableMatchingFanOutForPollCancellation=false` as a `workerStop` precondition with that reason (task 6). The precondition makes fn-121.3's CI green, is visible in every Case it affects, and is removed when the server fix lands. Never a Profile value.
2. **If R1 shows the caller's retry Case diverging** (CHASM counts `attempt` at schedule, HSM on failure): is the difference intended and to be modeled (R12, the Model reads `implementation`), or a CHASM defect to report, with the Model kept on HSM semantics and the CHASM retry Case expected to fail? **Open** until task 1's evidence; task 8 waits for this answer.
3. **Remote Profiles. Decided 2026-10-04: fail closed.** A remote or canary Profile must state every required key's value; otherwise the Case is `PreparationUnavailable`. No registered default is trusted (task 9).
4. **One Case per valuation. Decided 2026-10-04: yes.** The caller's Case files double and are renamed (task 7); no Testpilot field lists valuations.

## Parked unknowns

- Whether `recordCancelRequestCompletionEvents` (HSM-only events, set globally by the functional suites) becomes a Model setting once a close-policy Query reads those events.
- Whether `history.defaultActivityRetryPolicy` stops mattering once the activity realization states `retry_policy` in every start (R11 decides it per Case).
- Where the registry pin test lives: `tests/` or the Temporal Driver package.

## Quick commands

```bash
make umpire-check-model
go test -tags test_dep -p 2 -timeout 30m ./tools/umpire/... ./common/testing/testpilot/...
make umpire-check-cases
make umpire-check-live-tests
```

## Requirement coverage

| Task | Requirements | Gate |
| --- | --- | --- |
| .1 Nexus switch, its scope, and Driver schedule-to-close | R1, R2 | none |
| .2 settings in model/umpire, lifted, one table per valuation | R3 | fn-114 closed; not concurrent with fn-124.8 |
| .3 Query `under`, one Query per valuation | R4 | as .2; after .2 |
| .4 Quint and P exports | R5 | as .2; after .3 |
| .5 kit dynamic-config keys and registry pin | R6 | fn-118.2 done; not concurrent with fn-124.8 |
| .6 API preconditions, required settings with origin, harness from required settings | R7, R9 (harness half), Q1 | fn-118.2 done; after .5 |
| .7 Nexus implementation encodings, one Case per valuation, switch deleted | R8, R9 (switch), R12 (binding) | fn-118.5 done; after .1, .3, .6 |
| .8 caller Model reads `implementation` per the owner | R12 (Model) | owner's Q2 answer; after .7 |
| .9 bound assumptions checked at preparation | R10, Q3 | fn-118.4 done; coordinated with fn-124.3; after .6 |
| .10 dispositions of implicit assumptions | R11 | coordinated with fn-124.3; after .7, .9 |
| .11 docs and close | R13 | after all |

R9 is split because deleting the switch before the caller's encoding exists would run its Cases on server defaults.

## Status

Deferred by the owner on 2026-10-05. Task 1 (HSM/CHASM switch fixes, schedule-to-close no longer from the Profile) is done and merged; tasks 2-11 are blocked until the spec is revived. The evidence stays in `.plans/DYNAMIC_CONFIG.md`.
