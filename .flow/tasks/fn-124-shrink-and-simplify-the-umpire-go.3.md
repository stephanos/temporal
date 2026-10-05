---
satisfies: [R3]
---
# fn-124-shrink-and-simplify-the-umpire-go.3 Declare activity attempts, start carriers, lost admissions, causal parents, timeouts and history event names in the realization

## Description
Implements R3. Cross-spec entry gate: start after fn-118 lands (it edits waits, timeouts and realization surfaces). Each Temporal fact listed in R3 moves to one declaration in the realization (model/temporal/realize or the feature's Realization.scala), is lowered into the Case, and the Go runtime, lowering and conformance read the declared value instead of their own rule; remove the Go copies. Case bytes may change only by the new declared fields; record each change. Run the live generated Cases once.
## Acceptance
- [ ] History event names: the lowering and Testpilot take a history kind's recorded message from the realization's own history read (the read method's response type at the read path, which the Case already carries) and its oneof from the attributes member it names. `historyEventMessage` in `tools/umpire/lower/descriptor.go` and `common/testing/testpilot/internal/execution/evidence.go`, and the literal `attributes` oneof name next to them, are gone. No Case byte changes for this.
- [ ] Activity attempts: the Temporal kit declares once (`temporalBehavior`) how the server numbers an activity's attempts (from 1, every attempt of one activity in its one run). The lowering writes it into each Case activity entrypoint; Testpilot's scheduler classifies a reservation outcome by the declared numbering (no `ordinal+1`, no unconditional one-run rule) and refuses an activity entrypoint that declares none; the Temporal Driver refuses, at Profile derivation, a numbering other than the one it routes (from 1, one run); conformance's one-operation-one-run rule reads the realization's declaration. `guard.go` and the attempt role in `conformance/evidence.go` already read declared values (AttemptOf.number, the Run protocol's typed attempt); recorded, not changed.
- [ ] Default instruction timeouts: the kit declares the limits of an instruction that writes none, with its reason. The lowering writes them into each Case's Program; Testpilot resolves an instruction's limits as its own, else the Program's declared defaults, else the Profile's (a generic Profile option no Temporal Profile sets). `temporal.DefaultInstructionLimits` and the canary's literal copy are gone; a reader of the default (canary controller) reads the Case's.
- [ ] Causal parents: the kit declares that one operation's evidence is ordered by the Run's record across sources. The lowering writes it into the Case; Testpilot names the previous evidence of the operation as a causal parent only when the Case declares it.
- [ ] Lost admission: Testpilot's rule for a lost admission response is kept, because it checks the Driver's outcome against the Testpilot IR's own definition of `FAULT_KIND_ADMISSION_RESPONSE_LOSS` (it replaces one committed admission response), not a server fact: the server fact is the Model's (`committedThenLost`/`failedThenLost`, `committedDespiteLostResponse`), and removing the check would report a Driver bug as a product violation. Its attempt check is a presence check (no "from 1"), and `run.proto` no longer restates the numbering.
- [ ] Start carriers and "a failed start starts nothing" stay in the Temporal Driver (owner decision, recorded in R3): the Driver may know Temporal, carriers are its own interception mechanism, and a failed start releasing reservations can only make a Run incomplete. The carrier table is defined once (`delivery.Carried`), read by Profile derivation, the ledger and the worker Driver's validation.
- [ ] Case bytes change only by the new declared fields; each change is recorded with a before/after projection under `.flow/tmp/fn124-3/`. The golden harness and `original.json` are extended only for differences this task introduces. The Driver catalog identity rotates with the Testpilot proto; the pinned Runs and receipts follow as fn-118.3 did.
- [ ] Docs (model/README.md, model/SEMANTICS.md, Testpilot README where they state these rules) describe the declarations.
- [ ] Gates pass: model gate, original-baseline check, full Go tooling suite + Testpilot + canary tests (`-json`, timed), `make lint-code-fast`, `make umpire-check-cases`, `make umpire-check-fixtures`, `make canary-check-case`; the live generated Cases run once (known INCONCLUSIVEs and the stale `TestTestpilotAssessRecordedRuns` expectation excepted).
## Done summary
The Temporal kit now declares the facts the Go runtime, lowering and conformance used to hard-code. The lowering writes them into the Case, and Go reads the declared values; the Go copies are removed.

**Each fact, with its single source**
- **Activity attempts.** Covers `scheduler.go` (`ordinal+1` and one run) and `assessor.go`'s one-operation-one-run rule; the spec's `assessor.go:346-353` is that rule, now at line 164.
  - Source: `temporalBehavior.attemptNumbering = Some(AttemptNumbering(first = 1, oneRun = true))` in `model/temporal/realize/Behavior.scala`, cited to `chasm/lib/activity`.
  - Carried as IR `ApiBehavior.attempt_numbering` and Case `ActivityActivation.attempt_numbering`.
  - Testpilot judges each attempt as `first + ordinal`, checks one run only under `one_run`, and refuses an entrypoint that declares no numbering.
  - The Temporal Driver refuses, at `DeriveProfile`, a numbering other than the `{1, one run}` its ledger routes.
  - Conformance applies its one-run rule only where the realization declares it.
  - `guard.go` and `conformance/evidence.go` already read declared values, so they are unchanged.
  - The Testpilot schema docs and the lost-admission check no longer restate "from 1".
- **Default instruction limits** (`temporal/profile.go`).
  - Source: `instructionDefaults = Some(InstructionLimit(10000 ms, 1 attempt))`, with its reason. No API fact bounds Driver waits, controls, or calls whose response no read waits on.
  - Carried as Case `Program.instruction_defaults`.
  - Preparation resolves an instruction's limits in order: its own, the Program's, the Profile's. The Profile's is a generic Testpilot option that no Temporal Profile sets.
  - Removed: `DefaultInstructionLimits`, the testcore default and the canary's literal copy. The canary run and reconcile take their RPC timeout from the Case and error if it declares none.
- **Causal parents** (`execution/values.go`).
  - Source: `runOrderIsCausal = true`, with its reason. Carried as `Program.run_order_is_causal`.
  - The runtime names a cross-source causal parent only when the Case declares it.
- **History event names.**
  - These were already declared by the realization's own history read: the read method's response type at its path.
  - The lowering and Testpilot now take the event message from that read and the oneof from the member's containing oneof. The constants and the literal `"attributes"` are gone, with no Case byte change.
  - A history kind with no lifting read is refused. Lint's `Element` now takes the realization.
- **Start carriers and "a failed start starts nothing" stay in the Temporal Driver** (owner decision, now recorded in R3).
  - The Driver is allowed to know Temporal, carriers are its own interception mechanism, and a failed start releasing its reservations can only make a Run incomplete, never change a verdict.
  - A declared-carrier alternative was rejected because it would still leave two sources.
  - The table is defined once, in `delivery.Carried`. Profile derivation, the ledger and the worker Driver's validation all read it.
- **Lost-admission rule: kept.** It checks the Driver's report against the Testpilot IR's own definition of `FAULT_KIND_ADMISSION_RESPONSE_LOSS` ("replaces one committed admission response"), not a server fact. The server fact is the Model's (`committedThenLost`/`failedThenLost`, `committedDespiteLostResponse`). Removing the check would report a Driver bug as a product violation. Its attempt check is now a presence check.
- The reader refuses a first attempt number below 1 and non-positive default limits, at the declaration.

**Schema**
- IR: `ApiBehavior` fields 3–5, plus messages `AttemptNumbering` and `InstructionLimit`.
- Testpilot: `Program` fields 9–10, `ActivityActivation` field 4, and message `AttemptNumbering`.
- `buf lint` passes. `lint-api` reports two findings in `ir.proto` (`core::0203` at line 573, `core::0191` file layout) that are not from this change.

**Case bytes** (projections in `.flow/tmp/fn124-3/case-projection.txt`; all 34 Cases are identical after projection)
- All 20 `model/cases` gain `program.instructionDefaults` (10000/1) and `program.runOrderIsCausal`. The 8 Cases with an activity entrypoint also gain `activity.attemptNumbering` (1, one run).
- The 8 generated functional fixtures and the canary pin gain the same members.
- Hand-written fixtures declare the defaults their Profile used to supply: 10000/1 for most, plus `runOrderIsCausal` for `nexusPairTests`, and 1000/1 for `synthetic-case.json`. The runtime-conformance corpus keeps its bytes.

**Pinned identities**
- The canary Case identity moved, `32ec0ce1…` to `fbd796e5…`; the policy is re-pinned.
- The catalog identity rotated, `d1108efd…` to `3364057f…`.
- `make umpire-rerecord-pinned-runs` re-recorded the control and canary Runs live and re-rendered the receipts.
- The control companion is now the current generated control Case.

**Golden harness**
- `original.json` gains `declared_case_members`, an explicit allowlist of the three members.
- Current Cases are compared without them, identities are derived from the projected bytes, and a baseline Case carrying one is refused.
- Tests show each member is needed, and that a fourth member, an unlisted member or a changed byte still fails. `config.json` is unchanged.

**Merge**
- `umpire` at 358793b26f merged as `996a9240da` with no conflicts, and regeneration on the merged tree gave no diff.
- The merge commit has git's default message without the attribution trailer; it is not rewritten.

**Gates** (first round, on the merged tree)
- Model gate passed. The original-baseline and migration tests pass within the suite. `lint-model`, `lint-code-fast`, `umpire-check-cases`, `umpire-check-fixtures` and `canary-check-case` passed.
- Full Go suite (`-json`, `-p 2`): 288 s wall, 48 of 49 packages pass. `tools/umpire/model` was OOM-killed on the shared machine and passed when rerun alone at `-p 1`, 141.7 s (225 s wall).
- Slowest tests: `TestOriginalBaselineCases` 50.8 s, `TestOriginalBaselineModel` 48.0 s, `TestMigrationProjectionPreservesSemantics` 44.3 s.
- Live Cases: exit 2. 25 top-level tests pass, including `TestTestpilotAssessRecordedRuns`. Failing:
  - `nexus-caller-scheduleToStartTimeout/chasm`, `TestTestpilotWorkerOutageCase` and `TestTestpilotWorkerOutageCaseLeavesAnotherQueueAlone`: INCONCLUSIVE, the known ShutdownWorker race.
  - `activity-pauseResume`: fails the same assertion (expected 1, actual 3) in the same ShutdownWorker poll-rejection pattern on the unmodified `umpire` tree. The two runs differ in duration (30.1 s base vs 10.2 s in this run), and neither log names the unresolved rules. So it is consistent with a pre-existing failure, but not proven identical.
  - The race is not deterministic: `cancelIsRequested` and `scheduleToStartTimeout`, INCONCLUSIVE in fn-118.5's run, passed this time.

**Subagents:** one (same model), which built the golden-harness delta.
Review: claude-opus-5-5 at high, fresh context (host-dispatched subagent). Writer and reviewer are the same family (Opus).
- Round 1: NEEDS_WORK, no P1, four P2s.
  - The acceptance said the lost-admission rule was removed.
  - The Driver hard-coded attempt numbering with no check against the declaration.
  - The carrier table was restated in the worker Driver.
  - Start carriers left in the Driver needed a recorded decision.
  - All four, and the P3s, were fixed in 8c115253ab..bd884defc1. The host (acting for the owner) decided to keep start carriers in the Driver and recorded it in R3.
- Round 2: SHIP. One P3 was applied by the host in 5ae34249c2: the lost-admission attempt check is `<= 0`, so a negative attempt is still refused. The execution package tests pass.
- Left as follow-ups (P3, not blocking):
  - The numbering check runs only in `DeriveProfile`, so a hand-assembled Profile skips it. No such Profile runs an activity Case today. Repeating the check in the worker Driver's validation would close the gap.
  - The Testpilot-side refusal of history evidence with no lifting read is untested; the lowering side is tested.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: b7fc44bf25, 371359982f, bda99a1881, 0e1fd97fc5, 996a9240da, 498cd331a5, 8c115253ab, 0aa9acd927, 256b449ec7, 89df08a849, bd884defc1, 5ae34249c2
- Tests: make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks (exit 0), make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks (exit 0, no diff on merged tree), go test -count=1 -json -tags test_dep -p 2 -timeout 30m ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/... (48/49 pass; tools/umpire/model OOM-killed, rerun -p 1 pass; .flow/tmp/fn124-3/go-suite.json, go-suite-model.json), make lint-model (exit 0), make GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main lint-code-fast (exit 0), make umpire-check-cases && make umpire-check-fixtures && make canary-check-case (exit 0), make umpire-rerecord-pinned-runs (control and canary re-recorded), make umpire-check-live-tests (exit 2: ShutdownWorker-race family INCONCLUSIVE, activity-pauseResume also fails on base; .flow/tmp/fn124-3/live.log), review round: go test -tags test_dep -count=1 -p 2 ./common/testing/testpilot/... ./tools/canary/... ./tools/umpire/internal/golden/... ./tools/umpire/conformance/... ./tools/umpire/cmd/... ./tests/testcore/testpilot/... (pass; review-focused.log), review round: go test -run 'OriginalBaseline|Migration|TestALoweredCaseIsTheComparative' ./tools/umpire/lower (pass; review-lower.log), review round: make umpire-check-cases, canary-check-case, umpire-check-fixtures (pass); make lint-protos (pass); lint-code-fast origin/main no-fix (pass), go test -count=1 -tags test_dep ./common/testing/testpilot/internal/execution/ (after 5ae34249c2, ok)
- PRs: