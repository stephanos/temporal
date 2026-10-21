I’ll read the epic and task specs, then trace the relevant code paths to check feasibility and consistency.
The World authoring and source-binding tasks specify useful controls. The later tasks leave two contracts open: preserving the old activity goldens after adding attempts and inputs, and sourcing asynchronous cancellation during live admission. I’ll check those against the model and protocol, along with retry, timeout, and fairness behavior.
The old activity snapshots contain deadline edges that cannot occur in the proposed no-timeout world, so the literal compatibility requirement is currently impossible. I also confirmed that live observations carry responses but no start request, while timeout modeling needs option-type lifting and fact emission that the current DSL lacks. Those gaps need explicit implementation contracts.
Severity: P0
Confidence: 100
Classification: introduced
File:Line: .flow/tasks/fn-227-parameterized-models-declared-worlds.3.md:11
R-IDs: [R4, R5]
Location: Legacy activity compatibility witness.
Problem: The specified witness cannot reproduce the existing goldens. Current `expected/rows.snap.txt` includes ScheduleToStart and StartToClose edges reaching TimedOut. R4 selects a world with no timeouts, and R5 disables Tick there. Adding `attempt` and action inputs also changes the state/action renderings pinned by those snapshots. The epic acknowledges this conflict but leaves the impossible acceptance criterion intact.
Suggestion: Resolve the compatibility witness before the activity rewrite. State an executable check that satisfies the recorded compatibility decision, including deadline edges and rendering identity, and reconcile it across R4, R5 and task .3.

Severity: P1
Confidence: 75
Classification: introduced
File:Line: .flow/tasks/fn-227-parameterized-models-declared-worlds.5.md:7
R-IDs: [R6]
Location: Start evidence transport and transactional World admission.
Problem: The task requires reading an observed Start request but defines no route for delivering it. `Observation` contains only method and response/error; `StartActivityExecutionResponse` supplies no activity options. Furthermore, `Realize::observe` cannot replace the runtime’s private machine, and `Runtime::prepare` discards transactions on refusals. Implementing the task as a realization mapper cannot install the selected World or return Unsupported while committing the required freeze. The epic’s transactional-admission requirements are absent from the task’s approach and acceptance controls.
Suggestion: Specify how the actual request and successful result reach admission within the envelope constraint. Carry World installation, exploration invalidation, admission-specific freezing and publication rollback into task .5, with controls for failed starts, failed publication and subsequent operations.

Severity: P1
Confidence: 100
Classification: introduced
File:Line: .flow/tasks/fn-227-parameterized-models-declared-worlds.5.md:7
R-IDs: [R6]
Location: Cancellation-type input to live World construction.
Problem: Task .5 still instructs the implementer to read cancellation type from Start options. The current Start request mirror has no such field, and the epic explicitly confirms that neither the request nor standalone SDK options supplies it. Consequently, the task cannot construct `asynchronous_cancel` from its stated evidence without inventing a value or source.
Suggestion: Identify the authorized evidence source and its transport before task .5, then update its construction contract and tests. Merely recording this as an open question does not make the current construction instruction executable.

Severity: P1
Confidence: 75
Classification: introduced
File:Line: .flow/tasks/fn-227-parameterized-models-declared-worlds.4.md:7
R-IDs: [R5]
Location: Expired-deadline fact emission.
Problem: The task requires an expired-deadline fact without defining or assigning the necessary authoring support. Generated machines currently hard-code `Fact = ()`, require handlers to return nothing, and emit empty fact vectors. The lifter likewise emits `__umpire_facts' = <<>>`. Changing only the activity state and handlers cannot meet R5; emitting facts only natively also breaks backend replay’s outcome/fact comparison.
Suggestion: Specify the narrow deadline-fact authoring contract and include its macro, source-lowering and replay work. Require matching native/TLC facts for each expiration and empty facts for ordinary transitions.

Severity: P1
Confidence: 75
Classification: introduced
File:Line: .flow/tasks/fn-227-parameterized-models-declared-worlds.4.md:7
R-IDs: [R3, R5]
Location: Source admission for optional timeout values.
Problem: Task .4 introduces `Option<Ticks>` and requires TLC verification, but no task specifies their source representation or expression lowering. The current lifter admits scalar integers, bools, fieldless enums and vectors; `Environment::ty` refuses `Option<Ticks>`. Task .7 binds values using that existing representation, while .2’s native range-newtype support does not establish optional-value lifting. Following the stated approaches leaves the timeout Model refused by TLC.
Suggestion: Assign the concrete `Option<Ticks>` admission, binding, expression and decoding behavior within the existing tasks, with None/Some backend controls. Define the bounded tick representation without introducing general-purpose algebraic-type support.

Severity: P2
Confidence: 100
Classification: introduced
File:Line: .flow/specs/fn-227-parameterized-models-declared-worlds.md:24
R-IDs: [R2]
Location: Placement of `#[domain]`.
Problem: Architecture places `#[domain]` on the variant, while API Contracts and task .2 place it on fields. These produce different authoring syntax and different meanings for multi-field variants.
Suggestion: Choose one placement, define the iterator item type, and align the epic, task and compile controls.

Plan checks: The test strategy is stated and includes native checks, trybuild, goldens, actual TLC and live SDK proof. Dependencies form the serial chain `.1 → .7 → .2 → .3 → .4 → .5 → .6`; no independent Touches overlap exists. Single-iteration readiness for .4/.5 depends on resolving the supporting contracts above. Existing live-slice latency logging provides observability. Performance budgets, dependency restrictions, protocol preservation and guest privacy checks are stated.

maintainability:
  duplication: none identified
  structure: none identified

```json
{"classification_counts":{"introduced":6,"pre_existing":0},"unaddressed":["R2","R3","R4","R5","R6"]}
```

<verdict>NEEDS_WORK</verdict> - Plan has gaps that need addressing
