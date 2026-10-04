# fn-114-state-every-scala-model-declaration-once.12 Keep model/umpire Temporal-agnostic: move the Temporal realization vocabulary to temporal/realize and guard it

## Description
Owner direction (2026-10-04): `model/umpire` (the DSL framework) must not know Temporal-specific patterns; it stays Temporal-agnostic as far as realistic and reasonable. Temporal belongs in `model/temporal/**`.

Audit on 2026-10-04 (branch umpire-fn122): apart from the capability vocabulary and laws, which fn-122.8 moves, the Temporal knowledge in the framework is the realization layer `model/umpire/realize/**`. It has these pieces:
- Activation kinds `Workflow`, `NexusHandler` and `Activity`, with workflow and activity types, workers and task queues.
- `RoleKind { endpoint, worker, taskQueue, participant }` and a `Role` with `namespace`.
- `FaultKind { workerStop, workerResume, admissionResponseLoss }`.
- Replies `WorkflowCommand`, `NexusReply` and `NexusCompletion`.
- A failure documented as `temporal.api.failure.v1.Failure`.
- History reads (`HistoryRef`, `TypedHistory`, attempt-of-an-activity wording).
- `RequiredSetting`, phrased as server dynamic configuration.

The shared kit already lives in `model/temporal/realize/Kit.scala`.

**Entry gate:** after fn-114.7, before fn-114.9 (folder rename) and fn-114.8 (close). fn-118 and fn-119 extend the realization layer afterwards, so they build on the moved vocabulary.

**Size:** M
**Files:** `model/umpire/realize/**` → generic script machinery stays, Temporal vocabulary moves to `model/temporal/realize/**`; `model/lifter/Realizations.scala` (fully qualified names it matches); realization fixtures and refusals; Model `Realization.scala` imports; `tools/umpire/internal/golden/**`; `model/README.md`, `model/SEMANTICS.md`, `.plans/UMPIRE_MODULES.md`.
**Touches:** [model/umpire/**, model/temporal/**, model/lifter/**, tools/umpire/internal/golden/**, tools/umpire/model/**, model/README.md, model/SEMANTICS.md, .plans/UMPIRE_MODULES.md]

### Approach
- **What stays in `model/umpire/realize`:** what a realization of any system needs. That is commands, instructions, ordering (`after`, `perform`), evidence and observation plumbing, learned values, typed proto and gRPC request/response slots (gRPC and protobuf are transport, not Temporal), timeouts, and scripts.
- **What moves to `model/temporal/realize`:** the activation kinds, role kinds, fault kinds, reply kinds, history reads, `RequiredSetting` as dynamic configuration, and any Temporal wording in docs.
- **Mechanism for a closed set:** where the lifter needs a closed set (an enum it maps to Testpilot IR fields), keep the enum in the Temporal package and let the lifter match it by its new fully qualified name. The lifter and the Testpilot IR remain Temporal's driver tooling. Record that boundary in `.plans/UMPIRE_MODULES.md`, with which lifter files are Temporal-coupled.
- **Guard test:** add a test that fails when `model/umpire/**` names Temporal terms. Use a word list with whole-word, case-insensitive matching: temporal, workflow, activity, nexus, namespace, task queue/taskQueue, worker, history, chasm, matching, frontend, plus the six capability kinds. Allow only listed exceptions, each with a reason. If fn-122.8 hasn't landed yet, temporarily allowlist `model/umpire/Capabilities.scala` and `model/umpire/laws/**` with a pointer to fn-122.8.
- **Behavior frozen:** `model/ir/**`, `model/cases/**` and `lifts/expected/**` stay byte-identical, apart from positions and recorded symbol moves. Definition IDs stay unchanged.

## Acceptance
- [ ] `model/umpire/**` names no Temporal concept outside the guard test's reasoned allowlist; the guard fails on a planted Temporal term.
- [ ] Temporal realization vocabulary lives in `model/temporal/realize/**`; Models and fixtures import it from there; the lifter matches the new names and its realization refusals still fire at the author's line.
- [ ] IR, Cases, lifter expected output and Definition IDs are unchanged apart from positions and recorded symbol moves; original-baseline and migration goldens, model gate, `lint-model`, Go tooling suite and `lint-code-fast` pass.
- [ ] `.plans/UMPIRE_MODULES.md` and `model/README.md` state the agnosticism rule and which lifter/driver parts are Temporal-specific by design.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
