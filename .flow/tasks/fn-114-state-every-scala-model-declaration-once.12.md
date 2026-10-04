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
`model/umpire` no longer names Temporal concepts. The Temporal realization vocabulary now lives in `model/temporal/realize`, and a guard test keeps it out. Commits: 0b31bab957 (move, lifter, guard, docs) and 8294e77c9e (the guard uses `require`, as lint-code requires).

**What moved.** These now live in the new file `model/temporal/realize/Realize.scala`, beside `Kit.scala`:
- `Role` and `RoleKind`
- `RequiredSetting`, which is dynamic configuration
- `WorkerActivation.{Workflow, NexusHandler, Activity}`
- `WorkerInstruction.{AttemptFailure, AttemptCanceled, Fault, WorkflowCommand, NexusReply, NexusCompletion}`, and `FaultKind`
- `WorkflowHistory.event`, which replaces `Recorded.history`

**What stays generic.** `model/umpire/realize` keeps the generic machinery and five open traits that a kit extends:
- `Addressee`, for roles
- `Activation`, now holding only the case object `Controller`
- `Instruction`, with TypedRpc, TypedPoll, AwaitLearned, AwaitCommand, Finish, Hold and Release, now case classes
- `Recorded`, with TypedRead, TypedSingle and TypedRunEvent
- `Setting`

`Evidence.history` and `HistoryRef` became the generic `Evidence.keyed` and `KeyedRef`. The framework docs and examples were reworded with neutral (orders) examples.

**Lifter** (`model/lifter/Realizations.scala`):
- It accepts both vocabulary packages (`vocabularyPackages`).
- It writes a member of a vocabulary class or object by name and never follows its body. The kit is lifted source, so its synthetic `apply`, `$new` and enum-case vals would otherwise be inlined. The kit's top-level helpers are still followed.
- It writes vocabulary case objects as it writes enum cases.
- `identified` matches any `Addressee`.
- It matches `temporal.realize.WorkflowHistory.event` by its fully qualified name.
- A fact case excludes the vocabulary packages.

**Gate.** `model/gate/ProtoLiterals.scala` now recognizes `Evidence.keyed` and `WorkerActivation.NexusHandler`. This file is outside the Touches list, but the move needs it: without it the gate refused the `service` binding at nexuscaller `Realization.scala:314`.

**Guard.** `tools/umpire/model/framework_test.go` adds `TestFrameworkNamesNoTemporal` and `TestTemporalTermsAreFound`.
- It matches whole words, case-insensitively, after splitting identifiers at camelCase and underscore boundaries. The word list is the spec's, with plurals added.
- Its allowlist has reasons: `model/umpire/Capabilities.scala` and `model/umpire/laws/`, both pointing at fn-122.8.
- A stale entry fails the test, so the list only shrinks. Merging fn-122.8 therefore requires removing both entries.
- A planted term failed it, at `Domain.scala:97` (planted.log).

**Behavior.**
- `model/ir` and `model/cases/manifest.json` differ only in the line numbers in nexuscaller's and standaloneactivity's `Realization.scala`. The import edits caused this, and positions are projected by file.
- `lifts/expected` is byte-identical, and no refusal pin moved.
- Definition IDs are unchanged. `original.json` and the golden config are untouched.

**Decisions:**
- Each open trait keeps its name, and the Temporal enums take distinct names (`WorkerActivation`, `WorkerInstruction`, `WorkflowHistory`). This avoids ambiguity between the two wildcard imports that every Model has. `WorkflowHistory` also avoids `io.temporal.api.history.v1.History`.
- `Finish`, `AttemptOf`, `EventKind.diagnostic` and `Target.Lift` stay generic, with reworded docs: they are activation and attempt concepts any system has.
- To keep pinned lines in place, the lifter fixtures put their new imports on existing lines as comma-separated imports.
- Kit.scala is unchanged.

**Docs.** `model/README.md` and `.plans/UMPIRE_MODULES.md` state the rule and the guard. They record which parts are Temporal driver tooling by design: `Realizations.scala`'s vocabulary matching, `lifted()`, the `field :=` hook, `Capabilities.scala`, the IR's realization messages, `validate_realization.go`, `tools/umpire/lower` and Testpilot. `model/SEMANTICS.md` needed no change.

**Review.** claude-opus-5-5 at high via `--spec claude:claude-opus-5-5:high`; writer and reviewer are the same family (Opus). Round 1: SHIP. Deferred:
- **P3:** `KeyedRef`'s constructor is now public, so the compiler no longer ties `Root` to the record; the lifter still refuses a direct construction.
- **Pre-existing P3:** the duplicate `Evidence.read` and `Evidence.keyed` cases in the lifter.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 0b31bab957, 8294e77c9e
- Tests: make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks (exit 0), go test -tags test_dep -count=1 -p 2 -run 'OriginalBaseline|MigrationGoldens|MigrationProjection|IRInventory' ./tools/umpire/internal/golden ./tools/umpire/model ./tools/umpire/lower (exit 0), make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks (exit 0), make lint-model (exit 0), go test -tags test_dep -count=1 -p 2 -timeout 40m -json ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/... (exit 0, 332 s, 46 packages), GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main make lint-code-fast (exit 0), scala-cli test model/lifter (exit 0, 50/50), planted Temporal term in model/umpire fails TestFrameworkNamesNoTemporal (exit 1, expected)
- PRs: